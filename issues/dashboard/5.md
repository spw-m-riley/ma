# Degrade safely when dashboard delivery or persistence fails

## Type

AFK

## User stories covered

2, 7, 10, 11, 15

## What to build

Harden the dashboard observability path so it never becomes a dependency for normal `ma` command execution. This slice should cover the failure and privacy edges around the new feature: missing dashboard sessions, event-delivery outages, persistence write failures, and redacted runs should all leave the CLI functional while making the dashboard’s degraded or withheld state understandable to the user and maintainer.

## Acceptance criteria

- [ ] Normal CLI invocations keep their existing output and exit-code behavior when the dashboard is absent, event delivery fails, or history persistence cannot be updated.
- [ ] Failed, redacted, and successfully observed runs are distinguishable in dashboard-visible state so users do not have to guess why a run is missing details.
- [ ] Persistence or delivery failures are surfaced in a diagnosable way without turning observability into a required resident dependency.
- [ ] Tests cover dashboard-absent, event-delivery-unavailable, persistence-failure, and protected-content cases.

## Blocked by

- `dashboard-1` - Launch `ma dashboard` and show durable stats
- `dashboard-2` - Show live started, finished, and failed runs in the dashboard
- `dashboard-4` - Add a stats view for trends, top commands, and outcomes
