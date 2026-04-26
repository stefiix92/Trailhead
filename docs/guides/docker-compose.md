## Docker Compose guide (Trailhead in a stack)

Trailhead can run as **just another service** in your `docker-compose.yml`. For Compose stacks, the easiest way to get logs is to use the **Docker backend** (Trailhead reads container logs via the local Docker Engine API).

This guide covers:

- Running Trailhead alongside your services (as a sidecar)
- How to let Trailhead read logs from other containers
- Common security + portability pitfalls

---

## Option A (recommended for Compose): read Docker logs via the Docker Engine socket

### How it works

- Docker stores logs for each container (typically via the `json-file` logging driver).
- Trailhead talks to the Docker Engine API and queries those logs.
- In Compose, the standard way to provide that access is mounting the Docker socket:
  - `/var/run/docker.sock:/var/run/docker.sock:ro`

### Example `docker-compose.yml`

This is a **starting point**. You may need to adjust the command/flags based on how you run Trailhead (stdio MCP vs HTTP server mode).

```yaml
services:
  app:
    image: your-app:latest
    # Optional but recommended: label which containers Trailhead is allowed to read.
    labels:
      - "trailhead.logs=true"

  trailhead:
    image: ghcr.io/stefiix92/trailhead:v0.1.0
    # The image contains /trailhead-mcp (stdio MCP) and /trailhead (CLI).
    # For Compose usage you typically want an HTTP server mode so other clients can connect.
    # If you don’t have HTTP mode yet, you can still run Trailhead outside Compose via Cursor/stdio.
    #
    # command: ["/trailhead-mcp", "... http mode flags ..."]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      # If your Docker backend supports selecting by label, use labels to avoid “read everything”.
      # TRAILHEAD_DOCKER_LABEL_SELECTOR: "trailhead.logs=true"
      #
      # Keep sessions bounded (see README for default).
      TRAILHEAD_MAX_SESSION_LINES: "100000"
    restart: unless-stopped
```

### Important notes (security)

- **Mounting the Docker socket is powerful**. Even read-only mounts can still allow substantial introspection depending on the Docker API endpoints exposed and client behavior.
- Prefer **scoping** what Trailhead can read:
  - Use container **labels** and a Trailhead-side selector (if available).
  - Run Trailhead on a dedicated Docker context/daemon for production when possible.

### Important notes (portability)

- The socket path is Linux-default. On Docker Desktop (macOS/Windows), Compose still exposes `/var/run/docker.sock` inside Linux containers, but host-level paths and permissions can differ.

---

## Option B (production-grade): send logs to Loki, point Trailhead at Loki

If you want better retention, faster querying, and cross-host aggregation, add a logs backend:

- Run **Loki** in (or alongside) the stack
- Run a shipper (commonly **Promtail** or **Grafana Alloy**) to push container logs to Loki
- Configure Trailhead with:
  - `TRAILHEAD_LOKI_URL`
  - `TRAILHEAD_LOKI_TENANT` (optional, for multi-tenant Loki)
  - `TRAILHEAD_LOKI_BEARER_TOKEN` (optional)

This is usually the right choice when:

- You need to investigate incidents across multiple services/hosts
- You need retention longer than what Docker keeps locally
- You want `diff_error_rate` and richer label-based querying

---

## Troubleshooting

### “Trailhead can’t see my service logs”

Check:

- The service is a container managed by the same Docker Engine Trailhead is connected to
- The container logging driver isn’t `none`
- Trailhead is actually using the Docker backend (vs Loki/file/journald)

### “Trailhead returns too many lines / hits session caps”

Prefer the Trailhead workflow:

- `summarize_errors` (cluster + representative line IDs)
- then `sample_cluster`
- then `get_lines_by_id` for the exact evidence

Avoid raw tails unless you truly need them.

