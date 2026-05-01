## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Implement the `onPreToolUse` hook in `extension/extension.mjs` that intercepts full-file `view` calls on large files and redirects the agent to `ma_smart_read`.

The hook should:

1. Only fire for `toolName === "view"`.
2. Pass through calls with a bounded `view_range` under 200 lines (uses `isTargetedRead` from #1).
3. Pass through calls with `forceReadLargeFiles: true`.
4. Check file size via `isLargeFile` from #1. If the file is small (under ~10 KB / 200 lines), pass through.
5. Return `{ permissionDecision: "deny", permissionDecisionReason: "..." }` with a clear suggestion to use `ma_smart_read` and a note that `view` with a scoped `view_range` is available for exact content.
6. Treat `view_range` with end value `-1` as a full-file read — do not pass through.
7. Return `undefined` (fail open) on any error during the check.

## Acceptance criteria

- [ ] `onPreToolUse` hook registered in `extension/extension.mjs`
- [ ] `view` calls on files exceeding 200 lines / 10 KB are denied with `permissionDecision: "deny"`
- [ ] Denial reason names `ma_smart_read` as the alternative and mentions `view` with `view_range` for exact content
- [ ] `view` calls with bounded `view_range` < 200 lines pass through unintercepted
- [ ] `view` calls with `view_range: [n, -1]` are intercepted (treated as full-file read)
- [ ] `view` calls with `forceReadLargeFiles: true` pass through unintercepted
- [ ] `view` calls on small files pass through unintercepted
- [ ] Non-`view` tool calls are never affected
- [ ] Any error during file-size checking results in pass-through (fail open)

## Blocked by

- #1 — Runtime helpers (`isLargeFile`, `isTargetedRead`)
