## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Sync the completed extension from `extension/` to `~/.copilot/extensions/ma/` and verify all hooks fire correctly in a live Copilot CLI session.

Steps:

1. Copy `extension/extension.mjs`, `extension/runtime.mjs`, and `extension/runtime.test.mjs` to `~/.copilot/extensions/ma/`.
2. Run `node --test ~/.copilot/extensions/ma/runtime.test.mjs` to verify the deployed tests pass.
3. Reload extensions in Copilot CLI via `extensions_reload`.
4. Verify all hooks are registered by inspecting the loaded extension metadata.
5. Smoke-test each shipped hook in a live session:
   - `onSessionStart`: Start a new session and confirm the MA preference context appears in the session's initial context.
   - `onPreToolUse`: Attempt a full-file `view` on a large file (> 200 lines) and confirm it is denied with a redirect suggestion.
   - `onPostToolUse`: Use `forceReadLargeFiles` to bypass the pre-hook on a large file and confirm the post-read nudge appears.
   - `onUserPromptSubmitted`: Deferred follow-on work in issue #5. Do not block this deploy/verify slice on it.
   - `onSubagentStart`: Deferred follow-on work in issue #6. Do not block this deploy/verify slice on it.

## Acceptance criteria

- [ ] All three extension files synced to `~/.copilot/extensions/ma/`
- [ ] `node --test ~/.copilot/extensions/ma/runtime.test.mjs` passes all tests
- [ ] `extensions_reload` completes without errors
- [ ] Extension metadata shows the 3 shipped hooks registered (onSessionStart, onPreToolUse, onPostToolUse)
- [ ] Smoke-test confirms each shipped hook fires in the expected scenario
- [ ] Deferred follow-ons #5 and #6 remain out of scope for this deploy/verify slice
- [ ] No regression in existing tool handlers (`ma_smart_read`, `ma_skeleton`, `ma_compress`, `ma_dedup`, `ma_minify_schema`)

## Blocked by

- #1 — Runtime helpers
- #2 — Session-start preference and tool descriptions
- #3 — View intercept
- #4 — Post-view nudge
