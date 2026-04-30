# Inspect recent run input and output without long-term retention

## Type

AFK

## User stories covered

4, 8, 10, 11

## What to build

Add a short-lived recent-session buffer and dashboard detail view that let a user inspect the final input, output, and result for recent runs without turning observability into a durable content archive. This slice should keep full bodies out of long-term persistence, make recent before-and-after inspection available from the live dashboard, and respect the same protected-path posture the CLI already enforces.

## Acceptance criteria

- [ ] Recent live entries can be opened to inspect final input and output for recent runs.
- [ ] Full document bodies are retained only in memory or another explicitly bounded short-lived recent-session buffer, not in durable history.
- [ ] Runs involving sensitive or protected paths are redacted or withheld consistently with existing sensitive-path protections.
- [ ] Restart behavior matches the retention boundary: aggregate stats survive restart, while recent payload inspection does not persist beyond the intended short-lived scope.

## Blocked by

- `dashboard-2` - Show live started, finished, and failed runs in the dashboard
