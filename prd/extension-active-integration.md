# Active extension integration PRD

## Problem Statement

The MA extension and skill already exist, but the integration is entirely passive. The agent only uses MA tools when it independently decides to match a tool description or when the skill is explicitly activated. Nothing actively steers the agent toward MA tools for context reduction — `view` wins by default because the agent has no signal to prefer `ma_smart_read` over the built-in file reader.

The current extension registers no hooks for context injection, the tool descriptions describe their mechanisms rather than their selection criteria, and the global instructions never mention MA. A user who installs the extension and skill gets reduced context only when the agent happens to pick the right tool, which means the integration's full value is unrealized in most sessions.

## Solution

Make the MA extension actively steer the Copilot CLI agent toward context-reduced file reads at every opportunity by using the extension hook system and rewriting tool descriptions to compete explicitly with built-in tools. The extension should inject preference context at session start, intercept full-file `view` calls on large files and redirect them to `ma_smart_read`, and nudge the agent after any large-file read that bypassed the intercept. The tool descriptions should lead with decision boundaries rather than mechanism summaries, naming `view` as the tool they replace and the conditions under which they should be preferred.

The integration should remain fail-open and non-breaking. If the `ma` binary is unavailable, if a file is too small to benefit from reduction, or if the agent genuinely needs exact content, the system should silently degrade to standard behavior. The goal is to make MA the default for understanding reads without making it impossible to do exact reads when needed.

## User Stories

1. As a Copilot CLI user, I want the agent to automatically prefer `ma_smart_read` over `view` for files over 200 lines when reading to understand, so that my context window is not consumed by unreduced file bodies.
2. As a Copilot CLI user, I want the agent to receive an explicit preference rule at session start, so that the tool-selection bias toward MA is durable across the entire session.
3. As a Copilot CLI user, I want full-file `view` calls on large files to be intercepted and redirected to `ma_smart_read`, so that the default read path goes through context reduction.
4. As a Copilot CLI user, I want targeted `view` calls with a scoped `view_range` to pass through unintercepted, so that edit preparation and line-precise inspection still work.
5. As a Copilot CLI user, I want the `forceReadLargeFiles` flag on `view` to bypass the intercept, so that I have an explicit override when I genuinely need raw content.
6. As a Copilot CLI user, I want the tool descriptions to clearly state when each MA tool should be used instead of `view`, so that the agent's tool selection is informed by decision criteria rather than mechanism summaries.
7. As a Copilot CLI user, I want a post-read nudge when a large file was read via `view` without reduction, so that the agent learns to prefer MA tools for subsequent reads in the same session.
8. As a Copilot CLI user, I want the intercept and nudge to fail open on any error, so that a broken hook never prevents file reads.

## Implementation Decisions

### Hooks

Introduce three extension hooks in the existing `extension.mjs`:

- **`onSessionStart`** — Return `{ additionalContext }` with a compact MA preference rule (~130 tokens). This fires once per session and injects the preference before the first model turn, making it the most reliable steering signal. The content should name `ma_smart_read` as the preferred tool for understanding reads, `ma_skeleton` for code API shape, and `ma_minify_schema` for schemas, and should explicitly say to use `view` only for edit-preparation and line-precise inspection.

- **`onPreToolUse`** — Intercept `view` calls on large files and deny them with a suggestion to use `ma_smart_read`. The intercept should:
  - Only fire for `toolName === "view"`.
  - Pass through calls with a `view_range` that covers fewer than 200 lines (bounded partial reads).
  - Pass through calls with `forceReadLargeFiles: true`.
  - Check actual file size via `statSync`. If the file is under approximately 10 KB, pass through.
  - Return `{ permissionDecision: "deny", permissionDecisionReason: "..." }` with a clear suggestion to use `ma_smart_read` and a note that `view` with a `view_range` is available for exact content.
  - Fail open (return `undefined`) on any error during size checking.
  - Treat `view_range` with end value `-1` (meaning "to end of file") as a full-file read — do not pass through.

- **`onPostToolUse`** — After a `view` call completes on a large file without a scoped `view_range`, inject a brief `{ additionalContext }` nudge suggesting `ma_smart_read` for future reads. This fires only when the pre-hook did not deny the call (for example, when the file was borderline or the agent used `forceReadLargeFiles`). The nudge is non-blocking — it appends context after the tool result rather than preventing the read.

### Deferred Hooks

Two additional hook surfaces are documented but deferred from the initial implementation:

- **`onUserPromptSubmitted`** — Detect read-intent prompts via keyword matching and inject MA guidance per-turn. Deferred because `onSessionStart` already covers session-level preference, and per-turn detection adds complexity (regex tuning, session state caching) with incremental value.

- **`onSubagentStart`** — Propagate MA preference to fleet/multi-agent child agents. Deferred because it depends on the `onUserPromptSubmitted` cache and fleet mode is not the primary use case.

### Tool Description Rewrites

Rewrite the three tool descriptions that compete with built-in tools to lead with decision boundaries:

- **`ma_smart_read`** — Lead with "Use instead of `view` when reading a file (≥200 lines) to understand, explore, or research — not when editing or patching."
- **`ma_skeleton`** — Lead with "Use instead of `view` or `ma_smart_read` when you need a source file's API surface without implementation bodies."
- **`ma_minify_schema`** — Lead with "Use instead of `view` for JSON/YAML schema files you are reading to understand structure."

`ma_compress` and `ma_dedup` have no built-in competitor and retain their current descriptions.

### Repository Layout

The canonical extension source lives at `extension/` in the repository root, alongside `cmd/`, `internal/`, and `prd/`. The deployed version lives at `~/.copilot/extensions/ma/`, which is the location the Copilot CLI discovers and loads user-level extensions from.

### File Structure

- `extension/extension.mjs` — Main entry point. Tool definitions, hook registrations, binary runner.
- `extension/runtime.mjs` — Shared helpers: sensitive-path detection, binary discovery, fallback read, file-size estimation.
- `extension/runtime.test.mjs` — Tests covering runtime helpers and hook logic.

## Testing Decisions

- Validate that the `onSessionStart` hook returns `{ additionalContext }` with the expected preference text.
- Validate that `onPreToolUse` denies full-file `view` calls on files exceeding the size threshold.
- Validate that `onPreToolUse` passes through `view` calls with bounded `view_range` under 200 lines.
- Validate that `onPreToolUse` passes through `view` calls with `forceReadLargeFiles: true`.
- Validate that `onPreToolUse` intercepts `view_range` with end value `-1` as a full-file read.
- Validate that `onPreToolUse` fails open when `statSync` throws (file not found, permission error).
- Validate that `onPostToolUse` injects a nudge after large-file `view` reads and stays silent for small files.
- Validate that all existing tool handler tests continue to pass.
- Validate that the runtime test suite runs cleanly with `node --test extension/runtime.test.mjs`.

## Out of Scope

- Intercepting or reducing shell command output. That remains RTK's responsibility.
- Replacing the `view` tool globally. The intercept is conditional — small files, targeted reads, and explicit overrides all pass through.
- `onUserPromptSubmitted` intent detection. Deferred to a future iteration.
- `onSubagentStart` fleet propagation. Deferred to a future iteration.
- Context-budget-aware dynamic threshold adjustment. The 200-line threshold is static.
- Global `copilot-instructions.md` changes. Those are a separate maintenance action in the user's dotfiles.

## Further Notes

- The size threshold for the view intercept uses approximately 10 KB as a fast `statSync` gate, corresponding to roughly 200 lines at 50 bytes per line. This is a conservative proxy — false positives (large minified files) result in a redirect to `ma_smart_read` which will classify them as unsupported and pass through, so the cost is one extra tool call, not a broken read.
- The `view_range` escape hatch is intentionally conservative: only bounded partial reads under 200 lines pass through. An unbounded range (`[1, -1]`) or a range exceeding 200 lines is treated as a full-file read and intercepted. This prevents the agent from learning to use `view_range: [1, -1]` to bypass the policy.
- The `onPostToolUse` nudge is deliberately low-priority. In most sessions, the `onSessionStart` preference and `onPreToolUse` intercept will handle redirection. The nudge exists for edge cases where a file slips through (borderline size, `forceReadLargeFiles` override, or a new tool name).
- The existing passive skill at `~/.copilot/skills/ma/` does not need changes. It provides deep routing guidance and guardrails when explicitly activated, complementing the always-on hooks documented here.
