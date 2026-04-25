# Trailhead — agent entrypoint

This repo builds **Trailhead**, a self-hosted MCP server that lets coding agents investigate production logs **without hallucinating**.

## Product intent (read this first)

- **Query, don’t dump**: agents should use query primitives (summaries, clustering, sampling, correlation) rather than pulling thousands of raw lines.
- **Every claim cites a line**: any externally stated finding about logs must include supporting **log line IDs**.
- **Never leaves infra**: Trailhead is self-hosted and read-only. No SaaS/telemetry mode. Avoid workflows that would exfiltrate logs.

Source of truth: `PRD.md`.

## Scope and non-goals (v0.1 mindset)

- **In scope**: logs only; backends like Loki / Docker logs / journald/files; tools like `search`, `summarize_errors`, `sample_cluster`, `correlated_events`, `diff_error_rate`, `get_lines_by_id`.
- **Out of scope**: metrics/traces, “write” operations, UI, hosted tier.

## Definitions (use consistently)

- **LineID**: a short stable identifier for a log line within a session (e.g. `log:4021`). Any log content returned by tools must have an ID.
- **Session-scoped**: LineIDs are only guaranteed stable during the current conversation/session.
- **Cluster**: a group of similar errors/events; must have an id and a size/count.
- **Representative lines**: a small sample of line IDs/messages that exemplify a cluster; used as evidence when reporting findings.

## Default investigation playbook (incident triage)

Use this ladder; stop once you have enough evidence.

1. **Summarize**: what changed, what’s most common, how big is it?
2. **Cluster**: group errors by message shape / signature.
3. **Sample**: fetch a few representative lines from top clusters (IDs included).
4. **Correlate**: look around key timestamps (deploy markers, config reloads, restarts).
5. **Diff**: compare to baseline windows (yesterday same time, last hour).
6. **Raw tail**: last resort, minimal and filtered.

## Evidence rules (hard)

- Any finding you present as fact must cite **one or more LineIDs**.
- If you can’t cite it, label it clearly as an **uncited hypothesis** and explain what query would confirm it.

## Safety & privacy

- Don’t paste large raw logs into chat. Keep outputs bounded and purpose-driven.
- Assume logs can contain secrets/PII. Prefer summaries + LineIDs; redact if needed.
- Trailhead should never encourage copying logs into third-party services.

## Prompt snippets (copy/paste)

Reusable prompt templates live in `docs/agent-prompts/`.

- `docs/agent-prompts/go-best-practices.md`
- `docs/agent-prompts/incident-triage.md`
- `docs/agent-prompts/regression-after-deploy.md`
- `docs/agent-prompts/error-rate-diff.md`
- `docs/agent-prompts/show-me-the-evidence.md`
- `docs/agent-prompts/senior-architect.md`
