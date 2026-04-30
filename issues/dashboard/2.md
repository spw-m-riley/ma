# Show live started, finished, and failed runs in the dashboard

## Type

AFK

## User stories covered

2, 3, 7, 8, 15, 16, 18

## What to build

Add a local event-publishing path between normal CLI invocations and an active dashboard session so the live screen updates as runs begin and complete. This slice should make `ma` visibly observable with the current one-shot command model: a user can keep invoking commands normally and watch a live dashboard list update with started, finished, and failed lifecycle events plus the final structured result for completed runs.

## Acceptance criteria

- [ ] With `ma dashboard` running, a command invocation appears on the live screen as started and then transitions to finished or failed in order.
- [ ] Finished live entries include the final structured result summary derived from the existing command result contract.
- [ ] Failed runs are visually distinct from successful runs.
- [ ] Normal CLI commands still complete successfully when no dashboard session is active or event delivery is unavailable.
- [ ] The live screen remains useful without adding step-by-step progress instrumentation to every command.

## Blocked by

- `dashboard-1` - Launch `ma dashboard` and show durable stats
