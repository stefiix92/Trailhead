## Prompt: “Show me the evidence” follow-up (Trailhead)

The human asked to verify your claims. Respond by retrieving and presenting the cited lines.

### Rules

- Do not restate uncited conclusions as facts.
- Pull the exact lines by ID and keep the list minimal.

### Workflow

1. Collect the line IDs you cited (e.g. `log:4021`, `log:4044`, `log:4098`).
2. Fetch the full lines:
   - `get_lines_by_id(line_ids=[...])`
3. Present:
   - a short mapping from claim → LineIDs
   - then the fetched lines (or minimal excerpts) verbatim

