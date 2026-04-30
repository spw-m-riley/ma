# Add a stats view for trends, top commands, and outcomes

## Type

AFK

## User stories covered

5, 6, 7, 9, 18

## What to build

Expand the dashboard’s stats surface so users can understand long-term value, not just raw totals. Build a dedicated stats view that reports bytes, words, and token savings over time, highlights the commands used most often, and distinguishes successful runs from failed ones using the same persisted run-history model introduced for dashboard observability.

## Acceptance criteria

- [ ] The dashboard exposes a dedicated stats view for historical usage and savings.
- [ ] The stats view shows bytes, words, and approximate tokens saved over time plus top-command usage counts.
- [ ] The stats view distinguishes successful runs from failed runs so failures are not inferred only from missing stats.
- [ ] Historical aggregates remain available after process restarts because they are derived from durable run history.
- [ ] The stats view reuses the existing `ma` command result and metrics vocabulary rather than inventing a parallel reporting model.

## Blocked by

- `dashboard-1` - Launch `ma dashboard` and show durable stats
