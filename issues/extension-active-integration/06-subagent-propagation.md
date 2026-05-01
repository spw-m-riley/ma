## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Implement the `onSubagentStart` hook in `extension/extension.mjs` that propagates the MA preference context to fleet/multi-agent child agents.

When a subagent is spawned, inject `{ additionalContext }` with the same MA preference rule used by `onSessionStart` (or a compact variant) so that child agents also prefer MA tools over raw `view` for understanding reads.

Design considerations:

- **Reuse the session-start context** — The preference text from `onSessionStart` is already a compact ~130-token rule. Reusing it directly (or a slightly shortened version) avoids drift between parent and child preferences.
- **No per-turn state sharing** — The `onUserPromptSubmitted` dedup cache is parent-session-scoped and should not be forwarded. Each child agent starts fresh with the base preference.
- **Fleet mode awareness** — This hook is most valuable in fleet/explore mode where many subagents read files in parallel. Verify the hook fires for both `explore` and `general-purpose` agent types if the runtime distinguishes them.

## Acceptance criteria

- [ ] `onSubagentStart` hook registered in `extension/extension.mjs`
- [ ] Hook returns `{ additionalContext }` with the MA preference rule for every subagent start
- [ ] Injected context matches or is a compact variant of the `onSessionStart` preference rule
- [ ] Hook does not share per-turn dedup state from the parent session
- [ ] Hook fails open — returns `undefined` on any error

## Blocked by

- #5 — Per-turn intent detection (so the preference text and dedup strategy are settled before propagating to subagents)
