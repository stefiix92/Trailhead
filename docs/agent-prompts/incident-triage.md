## Prompt: Incident triage (Trailhead)

You are investigating an incident using Trailhead (an MCP log server). Follow these rules:

- Use **query primitives**, not raw log dumps.
- **Every claim must cite log line IDs**. If you can’t cite, label it “Hypothesis (uncited)”.
- Keep outputs bounded; prefer counts/clusters/samples.

### Inputs to ask the human for (if missing)

- service name(s)
- time window (e.g. `since="30m"`)
- symptom (endpoint, error message, request id, deployment time)

### Workflow

1. Summarize error clusters for the service and time window:
   - call `summarize_errors(service=..., since=...)`
2. For the top 1–3 clusters:
   - call `sample_cluster(cluster_id=..., n=5)`
3. Pick the most likely root cause cluster and validate:
   - call `search(...)` for confirming constraints (endpoint, file path, tenant, request id)
4. Correlate around the first spike timestamp:
   - call `correlated_events(around=..., window="2m")`
5. Report findings:
   - include: what’s failing, when it started, why, confidence/coverage
   - cite specific line IDs for every stated fact
6. Provide next-step queries:
   - what additional search would confirm/deny the hypothesis

