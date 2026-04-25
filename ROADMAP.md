# Roadmap

Trailhead is intentionally scope-disciplined: **logs-only**, self-hosted, evidence-first.

This roadmap is a lightweight guide to likely next work. It is not a promise.

## v0.1 (stabilize the core)

- Harden Loki/Docker/journald/file backends (correctness + performance + better errors)
- Improve “show me the evidence” workflows in agents and CLI
- Tighten tool schemas so “structured, citeable outputs” stay non-negotiable

## v0.2 (community backends + correlation)

- Encourage/accept a first community backend (e.g. Graylog, OpenSearch/ELK)
- Better deploy/history correlation primitives (still read-only)
- More incident-triage prompt snippets and examples in `docs/agent-prompts/`

## Later (only if demand is real)

- Additional backends (Sentry correlation, more aggregators)
- Optional smarter clustering experiments (only if they keep outputs citeable and bounded)

