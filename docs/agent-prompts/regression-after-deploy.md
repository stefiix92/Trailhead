## Prompt: Regression after deploy (Trailhead)

Goal: Determine whether an error spike correlates with a deploy/config change and produce **cited evidence**.

### Constraints

- Do not tail raw logs unless summaries/clusters fail.
- Every claim must cite LineIDs (and/or tool fields that directly support the claim).

### Workflow

1. Summarize errors in the suspected regression window:
   - `summarize_errors(service=..., since=...)`
2. Identify the first spike timestamp and top cluster:
   - cite the tool’s counts/buckets (if provided)
3. Sample the top cluster:
   - `sample_cluster(cluster_id=..., n=5)`
4. Correlate with deploy markers/restarts around the spike:
   - `correlated_events(around=..., window="5m")`
5. Compare vs a baseline window:
   - `diff_error_rate(service=..., window_a=..., window_b=...)`
6. Write the conclusion with explicit evidence:
   - “Regression starts at …” + cite LineIDs that show first occurrence(s)
   - “Deploy marker at …” + cite LineIDs from correlated events
   - “Top cluster is …” + cite representative lines

