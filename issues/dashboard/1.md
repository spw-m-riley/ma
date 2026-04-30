# Launch `ma dashboard` and show durable stats

## Type

AFK

## User stories covered

1, 2, 5, 6, 9, 12, 13, 14, 17, 18

## What to build

Add an explicit `ma dashboard` command that starts a loopback-only local web server and renders the first server-side dashboard view. Normal `ma` invocations should keep working through the existing CLI entrypoints while also recording lightweight run-history records that reuse the current `app.Result` and `app.Stats` vocabulary. The first dashboard slice should prove durable value on its own by showing persisted aggregate stats such as total savings and command usage counts without storing full input or output bodies.

## Acceptance criteria

- [ ] `ma dashboard` starts a long-running local server bound only to `127.0.0.1`.
- [ ] Existing CLI commands still run normally without a wrapper or alternate invocation mode.
- [ ] Every `ma` invocation records durable lightweight history including command identity, timestamps, success or failure, changed state, and existing bytes/words/token stats even when the dashboard is not running.
- [ ] The dashboard renders a server-side page that shows persisted aggregate stats, including total savings and per-command usage counts, after restart.
- [ ] Durable history does not persist full input or output bodies and does not weaken current sensitive-path protections.

## Blocked by

None - can start immediately.
