# Dashboard PRD

## Problem Statement

`ma` is a deterministic, offline CLI, but today it is mostly invisible while it works. A user who wants to understand what the tool is doing has to run commands one at a time and inspect terminal output after the fact. There is no simple way to watch runs happen live, compare input and output in one place, or understand whether `ma` is saving meaningful bytes, words, and tokens over time.

This makes the tool harder to trust, harder to demo, and harder to improve. A user who is curious about how `ma` behaves cannot easily answer basic questions such as which commands they use most, which commands save the most context, or whether a failed run behaved differently from a successful one. The current experience also makes it awkward to show someone else how the tool works without narrating a terminal session by hand.

## Solution

Add a local dashboard that lets a user watch `ma` working and review its historical impact without changing the existing CLI workflow. The dashboard should make `ma` feel observable: users can open a local page, watch command runs appear in real time, inspect the before-and-after output for recent activity, and review persistent aggregate stats about usage and savings.

The dashboard should stay true to `ma`'s current product boundaries. It should be local-only, offline, and additive to the CLI rather than a replacement for it. Users should still be able to run `ma` normally, while the dashboard quietly becomes a companion view that makes the tool easier to trust, measure, and demonstrate.

## User Stories

1. As a local `ma` user, I want to start a dashboard explicitly, so that the web UI is available only when I choose to observe the tool.
2. As a local `ma` user, I want existing CLI commands to keep working normally, so that I do not need a special wrapper or alternate invocation mode to benefit from the dashboard.
3. As a local `ma` user, I want to see each command run appear live as it starts, finishes, or fails, so that I can tell what the tool is doing right now.
4. As a local `ma` user, I want to inspect the final input and output for recent runs, so that I can understand what changed without reconstructing it from terminal history.
5. As a local `ma` user, I want to see byte, word, and token savings over time, so that I can judge whether the tool is providing real value.
6. As a local `ma` user, I want to see which commands I use most often, so that I can understand my own workflow and where the tool is most helpful.
7. As a local `ma` user, I want failed runs to appear distinctly from successful runs, so that I can spot problems without guessing from missing stats.
8. As a local `ma` user, I want the live screen to continue being useful even when a command does not stream step-by-step progress, so that the dashboard still adds value with the current one-shot command model.
9. As a local `ma` user, I want historical stats to remain available across restarts, so that the stats screen becomes more useful the longer I use the tool.
10. As a local `ma` user, I want the dashboard to avoid long-term storage of full document contents, so that observability does not quietly become a content archive.
11. As a privacy-conscious user, I want sensitive-path protections to remain respected, so that dashboard features do not create a new route for exposing protected content.
12. As a local `ma` user, I want the dashboard to bind only to my machine, so that I can use it without treating it like a remotely exposed service.
13. As a maintainer, I want the dashboard to reuse the tool's existing command result model and metrics vocabulary, so that the UI reflects real `ma` behavior instead of a parallel reporting system.
14. As a maintainer, I want lightweight stats to be recorded for every `ma` invocation, so that the stats screen is representative even when the dashboard is not open.
15. As a maintainer, I want live updates to activate only when the dashboard is running, so that the CLI does not take on unnecessary long-running behavior by default.
16. As a maintainer, I want the first version to stop at command lifecycle events plus final results, so that the feature ships without requiring every command to expose internal progress hooks.
17. As a maintainer, I want the UI layer to use server-rendered components with lightweight client-side updates, so that the dashboard fits the repo's deterministic, offline posture without introducing a heavy front-end app.
18. As someone demoing `ma`, I want a clear live view and a clear stats view, so that I can explain both immediate behavior and long-term value from the same local interface.

## Implementation Decisions

- Introduce an explicit long-running dashboard command that starts a local web server and owns the UI lifecycle. The existing command set remains one-shot and continues to be the primary way users invoke `ma`.
- Preserve the current command result contract as the backbone for dashboard reporting. Live updates should publish command lifecycle events such as started, finished, and failed, and should attach the final structured result when a run completes.
- Reuse the existing metrics vocabulary already exposed by `ma`, including bytes, words, and approximate tokens, so that the dashboard reports the same savings the CLI already knows how to measure.
- Add a lightweight persistent run-history store for aggregate stats. The stored record should include command identity, timestamps, success or failure, whether output changed, summary metrics, and enough source metadata to support useful reporting without becoming a full content archive.
- Keep full input and output bodies out of long-term persistence. Recent payloads may be retained only in memory or in a short-lived recent-session buffer to support the live screen's before-and-after inspection.
- Treat dashboard observability as additive to the CLI rather than a new execution mode. Standard invocations should still record lightweight historical stats even when the dashboard is not running, while live push behavior should activate only when a local dashboard session is present.
- Keep the dashboard local-only by binding to `127.0.0.1` and treating the product as single-user. Remote access, authentication, and shared multi-user semantics are intentionally excluded from the first version.
- Use a server-rendered UI approach with templ-based rendering, HTMX-driven updates, and existing templui components where possible. This keeps the dashboard aligned with the desired stack while avoiding a separate front-end build-heavy product surface.
- Introduce a dedicated local event-publishing mechanism between normal CLI invocations and an active dashboard session. That mechanism must tolerate the dashboard being absent and must not make command execution depend on a resident process.
- Respect the existing sensitive-path posture when deciding what content can be shown or retained. The dashboard should not weaken current protections by persisting full bodies from protected inputs.
- Keep the product offline and deterministic in the same sense as the rest of `ma`: local server only, no remote API dependency, and no change to the core transformation behavior of existing commands.

## Testing Decisions

- Validate that the dashboard command starts a local server successfully and binds only to the loopback interface.
- Validate that existing CLI commands still produce the same observable command results and terminal behavior when the dashboard feature exists but is not in use.
- Validate that every `ma` invocation records lightweight historical stats consistently, including success and failure cases, command identity, and savings metrics.
- Validate that the live updates screen receives started, finished, and failed events in order and shows the final structured result for completed runs.
- Validate that the live view can present recent input and output payloads for active or recent sessions without retaining those full bodies in long-term storage.
- Validate that the stats view reports durable aggregates correctly, including total savings, command frequency, and trend-style summaries built from the persistent run history.
- Validate that dashboard behavior respects the existing sensitive-path protections and does not persist protected content as long-term run artifacts.
- Validate restart behavior: historical stats survive process restarts, while short-lived recent-session content behaves according to the chosen retention boundary.
- Validate failure handling when the dashboard is not running, when event delivery is unavailable, and when persistence cannot be updated, ensuring normal command execution remains functional.
- Closest prior art in the repository comes from the existing command contract, structured result rendering, metrics measurement, command-focused tests, and sensitive-path protection behavior. New dashboard tests should follow that same behavior-first style rather than asserting internal implementation details.

## Out of Scope

- Replacing the normal CLI with a dashboard-first workflow.
- Remote dashboard access, shared viewing, authentication, or multi-user operation.
- Step-by-step in-command progress instrumentation for every existing command.
- Long-term storage of full input or output bodies.
- Reducing shell command streams or expanding `ma` into RTK-style command-output observability.
- New transformation logic for the existing `ma` commands beyond the reporting needed to support the dashboard.
- A rich front-end application with a separate client-side product architecture.

## Further Notes

- The repository currently has no verified issue-template or triage conventions, so this PRD is prepared as a markdown artifact rather than a published tracker item.
- The repo is currently a Cobra-based CLI with no web server, UI runtime, or SQLite dependency, so the dashboard introduces a meaningful new local application surface even though it remains consistent with `ma`'s offline scope.
- The persistence layer should remain durable across local restarts, but the exact SQLite packaging choice can be finalized during implementation as long as it preserves the offline, local-only requirement.
- The first version should prioritize trust and observability over internal command instrumentation depth. Lifecycle events plus final structured results are enough to make the product visibly useful without forcing a broad rewrite of command internals.
