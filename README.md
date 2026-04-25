# Trailhead

Self-hosted MCP server so AI agents can query production logs with **structured, citeable** results (line IDs), not raw dumps. See [PRD.md](PRD.md) for product scope.

> Open source. Self-hosted. Logs never leave your infra.

![CI](https://github.com/stefiix92/Trailhead/actions/workflows/ci.yml/badge.svg)
![Release](https://img.shields.io/github/v/release/stefiix92/Trailhead)
![License](https://img.shields.io/github/license/stefiix92/Trailhead)

## What this looks like in practice (30s)

- **Watch**: asciinema recording at [`docs/demo/upload-500.cast`](docs/demo/upload-500.cast) (play locally with `asciinema play docs/demo/upload-500.cast`).
- **Flow**: `summarize_errors` → pick a cluster → cite `line_id`s → `get_lines_by_id` to show the exact lines behind the claim.

## Install

### Prebuilt binaries (recommended)

Download from the GitHub Releases page and verify checksums:

```bash
# Example for macOS arm64; see Releases for all assets.
curl -L -o trailhead.tar.gz \
  https://github.com/stefiix92/Trailhead/releases/latest/download/trailhead_v0.1.0_darwin_arm64.tar.gz

curl -L -o checksums.txt \
  https://github.com/stefiix92/Trailhead/releases/latest/download/checksums.txt

shasum -a 256 -c checksums.txt | grep trailhead_v0.1.0_darwin_arm64.tar.gz

tar -xzf trailhead.tar.gz
sudo mv trailhead_v0.1.0_darwin_arm64/trailhead-mcp /usr/local/bin/trailhead-mcp
sudo mv trailhead_v0.1.0_darwin_arm64/trailhead /usr/local/bin/trailhead
```

### Docker (GHCR)

```bash
docker pull ghcr.io/stefiix92/trailhead:v0.1.0
```

The image includes `trailhead-mcp` and `trailhead` under `/trailhead-mcp` and `/trailhead`. The default entrypoint is stdio MCP (`/trailhead-mcp`).

## Configure in Cursor (MCP)

Add a stdio server (paths absolute on your machine):

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

## Try in about 60 seconds (from source)

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

3. **Smoke test** (stdio JSON-RPC):

   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./trailhead-mcp
   ```

## Tools (v0.1)

| Tool | Role |
|------|------|
| `search` | Bounded search (`file`, `loki`, `docker`, `journald`) |
| `summarize_errors` | TF–IDF clustering + coverage + representative line_ids |
| `sample_cluster` | More samples from a cluster after summarize |
| `diff_error_rate` | Compare error counts in two windows (**Loki only**) |
| `correlated_events` | Events around a timestamp + optional markers file |
| `get_lines_by_id` | Resolve cited line_ids in-session |

Line IDs are **session-scoped** and **source-prefixed** (`file:…`, `loki:…`, `docker:…`, `journald:…`).

## Verify a cited line_id (CLI)

In terminal/CI workflows you can resolve a previously cited `line_id` into full text:

```bash
./trailhead show loki:42 file:12
```

By default the CLI runs `./trailhead-mcp` as a subprocess and calls `get_lines_by_id` over stdio. To point it at another server binary/path:

```bash
TRAILHEAD_MCP_CMD=/absolute/path/to/trailhead-mcp ./trailhead show loki:42
```

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

## Contributing

Issues and PRs welcome — especially new backends and sharper “incident triage” primitives.

- See [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Security issues: see [`SECURITY.md`](SECURITY.md)

## License

MIT. See [`LICENSE`](LICENSE).

## Dogfooding status (be honest before promoting)

- This is intentionally a checklist, not marketing copy. If you haven’t done these yet, it’s better to say so than to imply production readiness.
- [ ] Run against real Loki or `file` logs for at least two weeks of local use.
- [ ] Confirm the “upload-500” flow end-to-end with cited `line_id`s (and keep the demo recording up to date).
- [ ] Publish a short write-up (blog or README story) using the scenario in [PRD.md](PRD.md).
