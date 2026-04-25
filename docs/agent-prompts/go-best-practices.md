# Prompt: Go best practices (Trailhead)

You are implementing or reviewing Go code in the Trailhead repo. Optimize for correctness, maintainability, and “single-binary” operational simplicity.

## Non-negotiables

- Prefer small, focused packages with clear responsibilities (`internal/<area>`).
- Keep public surface area minimal; default to `internal/` packages.
- Return **structured data** from query-like functions; avoid “stringly typed” APIs.
- Make failure modes explicit and testable: wrap errors with context; avoid silent fallbacks.
- Keep outputs bounded by default (limits, time windows, max samples).
- Assume log content may contain secrets/PII: avoid dumping raw logs; prefer IDs + samples.

## Code structure & APIs

- Define domain types for core concepts (LineID, ClusterID, TimeRange, Source).
- Prefer constructor functions returning interfaces only where multiple implementations are expected.
- Make input validation a first-class concern at boundaries (tool args, config parsing).
- Keep package names short and specific (e.g. `lineid`, `backends`, `query`).
- Avoid cyclical dependencies; if you feel pressure, introduce a small “types” package.

## Concurrency & cancellation

- Thread `context.Context` through all backend/query operations.
- Do not ignore cancellation; check `ctx.Done()` in loops and long operations.
- Avoid goroutine leaks; if you spawn goroutines, ensure bounded lifetimes and joins.

## Error handling & logging

- Wrap errors with `fmt.Errorf("...: %w", err)` to preserve cause.
- Prefer sentinel errors for expected conditions (unsupported source, invalid args).
- Avoid global logging; if needed, use `log/slog` and keep it injectable/configurable.

## Testing

- Table-driven unit tests for pure helpers.
- For backends, use small fixture files and deterministic queries.
- When a bug is found, add a regression test first (TDD).

## Review checklist (quick)

- Is the package boundary clean and minimal?
- Are defaults safe and bounded?
- Are contexts honored everywhere?
- Are error messages actionable and wrapped?
- Are tool outputs structured (JSON fields) rather than prose?
