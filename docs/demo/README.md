## Trailhead demo (upload-500)

This folder contains the canonical 30-second recording referenced from the main README.

### Prereqs

- Install `asciinema` (`brew install asciinema`, or see your distro package manager)
- A terminal width of ~100 columns works well

### Recording script (suggested)

1. Start Cursor with Trailhead configured as an MCP server.
2. Trigger the canonical flow in the Agent chat:
   - `summarize_errors` (narrowed to the demo service / selector)
   - pick the top `cluster_id`
   - cite a couple of `line_id`s in the explanation
   - `get_lines_by_id` to show the cited evidence
3. Keep it short; the goal is to show “query → cite → verify”.

### Record

From repo root:

```bash
asciinema rec -c "bash -lc 'cat docs/demo/script.txt'" -t \"Trailhead: summarize_errors → cite line_ids\" docs/demo/upload-500.cast
```

Then review:

```bash
asciinema play docs/demo/upload-500.cast
```

