# Contributing to Trailhead

Thanks for contributing — Trailhead is intentionally small and scope-disciplined.

## What we’re building (and what we’re not)

Please skim [`PRD.md`](PRD.md) before starting work.

- **In scope (v0.1 mindset)**: logs only; backends (Loki/Docker/journald/files); query primitives (`search`, `summarize_errors`, `sample_cluster`, `correlated_events`, `diff_error_rate`, `get_lines_by_id`).
- **Out of scope**: metrics/traces, UI, write operations, hosted/SaaS modes.

## Quickstart (dev)

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go build ./...
```

## How to contribute

### Reporting bugs

Open a bug report in Issues. Include:

- What backend you used (`loki`, `docker`, `journald`, `file`)
- Exact tool call payload and the tool output (redact secrets)
- Expected vs actual behavior
- If relevant: a small sample of log lines (redacted) that reproduces the issue

### Suggesting features

Feature requests are welcome — but **Trailhead stays query-oriented**.

Great requests include:

- A concrete incident-triage workflow to support
- The exact tool signature (inputs/outputs) you think is needed
- Why existing primitives can’t express the query

### Adding a new backend

This is a high-impact contribution.

Please aim for:

- **Thin adapter**: backend code translates backend APIs into the internal log-event model.
- **No “dump everything” tool**: keep results bounded; prefer aggregates + sampling.
- **Line IDs everywhere**: any returned log content must have a session-scoped `line_id`.

### Improving tools / schemas

Trailhead is opinionated about anti-hallucination behavior:

- Return **structured JSON**, not prose.
- Include **counts + coverage** when summarizing clusters.
- Make evidence easy to cite (representative `line_ids`).

## Code style & quality

- Keep diffs small and focused.
- Prefer adding tests for bug fixes.
- Avoid introducing dependencies unless the benefit is clear.

## Submitting a PR

- Keep the PR description crisp: **what changed, why, and how to test**.
- If you touched tool outputs, include a before/after example in the PR body.
- If you add a new tool, document it in `README.md` and ensure it follows the repo’s evidence rules in `AGENTS.md`.

## License

By contributing, you agree that your contributions will be licensed under the MIT License (see [`LICENSE`](LICENSE)).
