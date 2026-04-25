# Trailhead

> A self-hosted MCP server that lets AI coding agents investigate production logs without hallucinating.

Trailhead connects Claude Code, Cursor, and other MCP-compatible agents to the logs you already have — Loki, Docker, journald, plain files — and gives them a **query interface** instead of a raw log dump, with **mandatory line-ID citations** on every finding.

Open source. Self-hostable. EU-friendly. No logs leave your infra.

---

## Why this exists

The canonical AI-debugging workflow — *"hey, there's a bug on prod, check logs and tell me what's going on"* — already works well if you pay Datadog. Their MCP server is mature, their team learned the right lessons (query interface > raw dump, actionable errors, 40% token reduction), and GA is March 2026.

Everyone else is stuck. Teams running Loki, Graylog, Docker Compose, or plain journald on their own infra have three options today:

1. **Copy-paste from a web UI into the chat.** Breaks flow, drops context, doesn't scale beyond a few hundred lines.
2. **Naive MCP servers that expose `get_logs()`.** Agent pulls 5,000 lines, context window fills up, model hallucinates a plausible root cause from whatever survived truncation.
3. **Nothing.** Debug by hand, like it's 2022.

None of these are good. The gap isn't "does an AI-debugging tool exist" — it's "does one exist that works on self-hosted infrastructure, doesn't lie, and doesn't send your customer logs to a US SaaS."

Trailhead fills that gap.

---

## Principles

Three hard rules shape every design decision.

**1. Query, don't dump.** Agents get primitives for *asking questions about* logs, not for reading them line by line. "Top error types in the last hour." "Error rate diff vs. yesterday at this time." "20 representative samples of error cluster X." Raw tail is a last resort, not a default.

**2. Every claim cites a line.** If the agent reports "the DB pool saturates around 14:02," the response includes the log line IDs that support it. No citation = the agent is guessing, and Trailhead's prompts and tool schemas are designed to make that visible to the human.

**3. Never leaves your infra.** No cloud, no telemetry, no "helpful" SaaS mode in v1. Runs as a single binary or Docker container. Credentials stay local. This is the entire reason someone would pick Trailhead over Datadog's MCP.

---

## Scenario — the canonical demo

**Developer:** "There's a 500 spike on the upload endpoint. Check app-backend logs for the last 30 minutes."

**Claude Code (via Trailhead):**

1. `trailhead.summarize_errors(service="app-backend", since="30m")` → 47 errors in 3 clusters, top cluster is `FileNotFoundError` (38 occurrences).
2. `trailhead.sample_cluster(cluster_id="c1", n=5)` → 5 representative lines with IDs `log:4021, log:4044, log:4098, log:4122, log:4156`.
3. `trailhead.correlated_events(around="2026-04-24T14:02:00Z", window="2m")` → deploy marker at 14:01:47, commit `abc123`.

**Claude responds:** *"The upload endpoint is failing because `/tmp/uploads` doesn't exist on the container filesystem. This started at 14:02, right after deploy abc123, which moved upload handling to a new worker image. Evidence: lines `log:4021, log:4044, log:4098` — all `FileNotFoundError: /tmp/uploads/<uuid>`. Likely fix: add the directory to the Dockerfile or mount a volume."*

Every claim has a log line ID. The human can click any of them to verify in 2 seconds.

---

## Scope — v0.1

**In:**

- Loki backend (most common self-hosted log stack in 2026).
- Docker logs backend (reads `docker logs` directly, for teams without a log aggregator).
- Local file / journald backend (for single-server setups).
- Core MCP tools: `search`, `summarize_errors`, `sample_cluster`, `correlated_events`, `diff_error_rate`, `get_lines_by_id`.
- Line-ID citation format: every tool result includes stable IDs the agent can quote.
- Single binary or Docker image. `docker run` gets you running in under a minute.
- MIT license.

**Out (for v0.1):**

- Metrics and traces. Logs only. Scope discipline matters.
- Hosted / SaaS version. Self-hosted first; if there's demand, a hosted tier can come later.
- Write operations. Trailhead is read-only to logs. No "delete these logs" or "acknowledge alert."
- Paid features. Everything in the repo is free.
- UI. It's an MCP server; the agent is the UI.

**Maybe later:**

- Graylog, OpenSearch, ELK backends (community contributions welcome).
- Deploy-history correlation (GitHub Actions, GitLab CI, raw git log).
- Sentry / error-tracker correlation.
- Solana validator log support — my own itch, plugin-style.

---

## Architecture

```
┌──────────────┐    MCP      ┌──────────────┐    backend    ┌──────────┐
│ Claude Code  │ ──────────▶ │  Trailhead   │ ────────────▶ │  Loki    │
│ Cursor       │  (stdio or  │  server      │               │  Docker  │
│ other agent  │   HTTP)     │              │               │  files   │
└──────────────┘             └──────────────┘               └──────────┘
                                    │
                                    ▼
                             ┌──────────────┐
                             │  Query layer │  ← the actual product
                             │  clustering, │
                             │  sampling,   │
                             │  citations   │
                             └──────────────┘
```

The *backend adapter* is a thin translation layer. The *query layer* is where the value lives — that's what turns "dump 5000 lines" into "top 3 error clusters with representative samples." Anyone can write a backend adapter; the query layer is the moat.

**Language:** Go. Single binary, cross-compile cleanly, fast enough to do clustering on the fly, low memory footprint for a long-running sidecar. (Rust was tempting but Go wins on contributor onboarding.)

**Transport:** stdio for local agent integrations (Claude Desktop, Cursor), HTTP for remote deployments.

---

## Anti-hallucination design

This is what makes Trailhead different from the 20 other "log MCP" repos on GitHub.

**Tool outputs are structured, not prose.** Every tool returns JSON with explicit fields: `line_ids`, `timestamps`, `counts`, `sample_messages`. The agent can't mistake aggregate counts for specific evidence.

**Line IDs are stable within a session.** When Trailhead returns a log line, it assigns a short ID (`log:4021`). The agent can reference that ID later to pull the full line for a human. IDs are ephemeral (session-scoped) but predictable within a conversation.

**Summaries include confidence hints.** "47 errors across 3 clusters, covering 94% of error volume" tells the agent how much it can trust the summary. "2 errors total, too few for clustering" tells it to fall back to raw lines.

**Prompts in the MCP tool descriptions instruct the agent to cite.** The tool description for `summarize_errors` ends with: *"When reporting findings to the user, cite specific line IDs from the `representative_lines` field. If you make a claim you cannot cite, say so explicitly."* This doesn't force citation — nothing can — but it makes the right behavior the easy behavior.

---

## Non-goals — explicitly

- **Not replacing your log UI.** Grafana, Kibana, Dozzle are good. Trailhead is for when a human wants an agent to investigate *for* them, not *instead of* them.
- **Not a full observability platform.** If you want unified logs + metrics + traces, use SigNoz or pay Datadog. Trailhead does logs well and that's it.
- **Not for production autonomy.** An agent reading logs and reporting findings to a human is fine. An agent reading logs and taking automated action on prod is out of scope — Trailhead will never ship a write tool.
- **Not trying to be fast-growing.** This is a side project. It succeeds if a few hundred people use it and a handful contribute backends.

---

## Success criteria

**6 months:**

- Works cleanly with Claude Code and Cursor on Loki and Docker backends.
- 5+ organic users who aren't me.
- Clear "try it in 60 seconds" path in the README.

**12 months:**

- At least one community-contributed backend (likely Graylog or OpenSearch).
- A blog post or two from someone who used it to debug a real incident.
- If usage is real and consistent, consider a hosted tier. If not, keep it as a focused OSS tool and that's fine.

**Explicit failure mode I'm avoiding:** turning this into a feature-bloated framework. Scope creep is how OSS log tools die.

---

## Open questions

- **Clustering algorithm.** TF-IDF + cheap similarity works for 90% of cases; BERT-mini is overkill for v0.1 but worth prototyping in v0.3.
- **Line ID format.** `log:4021` is ergonomic but collides across sources. Maybe `loki:4021` / `docker:4021`? Decide before v0.1 ships — renaming later is a migration.
- **Backend auth.** Loki supports multi-tenant via `X-Scope-OrgID`. Need a clean config for teams that have per-environment tenants.
- **How does a human verify a cited line?** In Claude Desktop / Cursor the answer is "click through." For CLI agents it's murkier. A `trailhead show log:4021` sidecar command might be the answer.

---

## What's next

1. Register the name (`trailhead-mcp` on npm/crates/Docker Hub, `trailhead` on GitHub if available).
2. Decide: Go vs. TypeScript. Go is the right answer for the reasons above, but if the MCP Go SDK is immature, TypeScript + ncc-bundled binary is a pragmatic fallback. Verify before committing.
3. Skeleton: one backend (Loki), one tool (`search`), end-to-end with Claude Code. Get the citation format right before adding anything else.
4. Add `summarize_errors` with TF-IDF clustering. This is the tool that justifies the project's existence.
5. Dogfood on your own infra for 2 weeks before telling anyone.
6. Write a blog post with the "500 on upload" scenario as the demo. Post it where self-hosted devs actually read — r/selfhosted, Hacker News, Lobsters. Not LinkedIn.
