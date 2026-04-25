# Trailhead
A self-hosted MCP server that lets AI coding agents investigate production logs without hallucinating.

## Dev quickstart

Build:

```bash
go build ./...
```

Run (stdio JSON):

```bash
go run ./cmd/trailhead-mcp
```

Try a tool call (example request; paste into stdin):

```json
{"jsonrpc":"2.0","id":1,"method":"tools/list"}
```

