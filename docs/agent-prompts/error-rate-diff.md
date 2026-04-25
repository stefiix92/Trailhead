## Prompt: Error-rate diff (Trailhead)

Goal: Compare error rates between two windows and explain what changed, with cited evidence.

### Workflow

1. Compute diff:
   - `diff_error_rate(service=..., window_a=..., window_b=...)`
2. Summarize errors for the worse window:
   - `summarize_errors(service=..., since=...)` (or an explicit window if supported)
3. For the top changing cluster(s):
   - `sample_cluster(cluster_id=..., n=5)`
4. Explain:
   - how much error volume changed (cite diff output fields)
   - which error types/clusters drove it (cite counts + representative line IDs)
   - what likely changed operationally (only if cited; otherwise label hypothesis)

