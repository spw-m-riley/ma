## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Implement the `onPostToolUse` hook in `extension/extension.mjs` that nudges the agent after a large-file `view` read that was not intercepted by the `onPreToolUse` hook.

This fires when:

- The completed tool is `view`.
- The file is large (checked via `isLargeFile` from #1).
- The call was not a targeted read (no bounded `view_range` under 200 lines).

In that case, return `{ additionalContext }` with a brief message suggesting `ma_smart_read` for future reads of similar files. The nudge is non-blocking — it appends context after the tool result rather than preventing the read.

This covers edge cases where a large file slips past the pre-hook: borderline file sizes, `forceReadLargeFiles` overrides, or any future tool name variants.

## Acceptance criteria

- [ ] `onPostToolUse` hook registered in `extension/extension.mjs`
- [ ] Hook injects `{ additionalContext }` after large-file `view` reads without bounded `view_range`
- [ ] Nudge text names `ma_smart_read` as the preferred alternative
- [ ] Hook stays silent (returns `undefined`) for small files
- [ ] Hook stays silent for targeted reads with bounded `view_range`
- [ ] Hook stays silent for non-`view` tools
- [ ] Any error during file-size checking results in silence (fail open)

## Blocked by

- #1 — Runtime helpers (`isLargeFile`, `isTargetedRead`)
