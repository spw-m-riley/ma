## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Add the view-intercept runtime helpers to `extension/runtime.mjs` that the hook slices depend on. These are the building blocks that decide whether a `view` call should be intercepted:

- **`isLargeFile(filePath, cwd)`** — Two-stage check: fast `statSync` gate at 10 KB, then `openSync`+`readSync` counting newlines up to a 200-line threshold. Returns `false` (fail open) on any I/O error.
- **`countLinesExceeds(filePath, threshold)`** — Reads 32 KB chunks counting `0x0a` bytes, short-circuits at threshold. Internal helper used by `isLargeFile`.
- **`isTargetedRead(toolArgs)`** — Returns `true` only when `view_range` is present, bounded (end ≠ −1), and spans fewer than 200 lines. Rejects unbounded `[start, -1]` ranges as full-file reads.
- Export `VIEW_LINE_THRESHOLD` and `VIEW_SIZE_GATE_BYTES` constants.

Cover all helpers with `node --test` tests in `extension/runtime.test.mjs`:

- `isLargeFile` returns `false` for small files, `true` for 250+ line files over 10 KB, `false` for nonexistent files (fail open), `false` for many short lines under 10 KB, correct resolution of relative paths against `cwd`.
- `isTargetedRead` returns `false` with no `view_range`, `true` for bounded partial reads under 200 lines, `false` for ranges ≥ 200 lines, `false` for EOF marker `[n, -1]`.

## Acceptance criteria

- [ ] `isLargeFile`, `isTargetedRead` exported from `extension/runtime.mjs`
- [ ] `countLinesExceeds` is internal (not exported) but covered indirectly through `isLargeFile` tests
- [ ] All new helpers fail open — any I/O error returns `false`/safe default
- [ ] `node --test extension/runtime.test.mjs` passes with ≥ 9 new test cases covering both helpers
- [ ] Existing sensitive-path and fallback-read tests continue to pass
- [ ] `path` import added to `runtime.mjs` for relative-path resolution

## Blocked by

None — can start immediately.
