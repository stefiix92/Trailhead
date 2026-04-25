# Trailhead

Self-hosted MCP server so AI agents can query production logs with **structured, citeable** results (line IDs), not raw dumps. See [PRD.md](PRD.md) for product scope.

## Try in about 60 seconds

1. **Build** (static binary, no cgo):

   ```bash
   CGO_ENABLED=0 go build -o trailhead-mcp ./cmd/trailhead-mcp
   CGO_ENABLED=0 go build -o trailhead ./cmd/trailhead
   ```

2. **Optional: Loki** — Set before starting the server:

   | Variable | Meaning |
   |----------|---------|
   | `TRAILHEAD_LOKI_URL` | Base URL, e.g. `https://loki.example.com` |
   | `TRAILHEAD_LOKI_TENANT` | `X-Scope-OrgID` when using multi-tenant Loki |
   | `TRAILHEAD_LOKI_BEARER_TOKEN` | Bearer token for Loki if required |
   | `TRAILHEAD_MARKERS_FILE` | JSON file of deploy markers (see below) |
   | `TRAILHEAD_MAX_SESSION_LINES` | Cap stored line records (default `100000`) |
   | `TRAILHEAD_DEV_TOOLS=1` | Expose non-log `test_coverage` (Go projects only) |

3. **Cursor MCP** — Add a stdio server (paths absolute on your machine):

   ```json
   {
     "mcpServers": {
       "trailhead": {
         "command": "/absolute/path/to/trailhead-mcp",
         "env": {
           "TRAILHEAD_LOKI_URL": "https://loki.example.com"
         }
       }
     }
   }
   ```

4. **Smoke test** (stdio JSON-RPC):

   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./trailhead-mcp
   ```

## Docker

```bash
docker build -t trailhead-mcp:0.1.0 --build-arg VERSION=0.1.0 .
# Mount config / pass env at run time; stdio is the default entrypoint.
```

Image includes `trailhead-mcp` and `trailhead` CLI under `/trailhead-mcp` and `/trailhead`.

## Tools (v0.1)

| Tool | Role |
|------|------|
| `search` | Bounded search (`file` or `loki`) |
| `summarize_errors` | TF–IDF clustering + coverage + representative line_ids |
| `sample_cluster` | More samples from a cluster after summarize |
| `diff_error_rate` | Compare error counts in two windows (**Loki only**) |
| `correlated_events` | Events around a timestamp + optional markers file |
| `get_lines_by_id` | Resolve cited line_ids in-session |

Line IDs are **session-scoped** and **source-prefixed** (`file:…`, `loki:…`). The `trailhead show` CLI explains that resolution is via `get_lines_by_id` on the live server.

## Canonical demo (product script)

1. `summarize_errors` with `source=loki`, `lokiService` (or `lokiStreamSelector`), `since: 30m`.
2. `sample_cluster` on the top `cluster_id`, `n: 5`.
3. `correlated_events` with `around` at the first spike (RFC3339), `window: 2m`.
4. Report with **cited** `line_id` values from tool output.

## Deploy markers (`TRAILHEAD_MARKERS_FILE`)

JSON array of objects with `time` (RFC3339) and `label`:

```json
[
  {"time": "2026-04-24T14:01:47Z", "label": "deploy abc123"}
]
```

Included in `correlated_events` when the time falls inside the query window.

## Development

```bash
CGO_ENABLED=0 go test ./...
```

On some macOS setups, tests crash unless `CGO_ENABLED=0` (static test binary).

Internal `test_coverage` is **off** unless `TRAILHEAD_DEV_TOOLS=1`.

## Adoption checklist (dogfood before a public push)

- [ ] Run against real Loki or `file` logs for at least two weeks of local use.
- [ ] Confirm the upload-500 style flow end-to-end with cited line IDs.
- [ ] Publish a short write-up (blog or README story) using the scenario in [PRD.md](PRD.md).
