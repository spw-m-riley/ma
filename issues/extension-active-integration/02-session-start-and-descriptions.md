## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Register an `onSessionStart` hook and rewrite three tool descriptions so the agent is steered toward MA tools from the first turn of every session.

**Session-start hook:** Return `{ additionalContext }` with a compact (~130 token) MA preference rule that:

- Names `ma_smart_read` as the preferred tool for understanding reads on files ≥ 200 lines.
- Names `ma_skeleton` for code API shape and `ma_minify_schema` for schemas.
- Explicitly says to use `view` only for edit preparation and line-precise inspection.
- Fires once per session before the first model turn.

**Tool description rewrites:** Rewrite three tool descriptions to lead with decision boundaries instead of mechanism summaries:

- **`ma_smart_read`** — "Use instead of `view` when reading a file (≥ 200 lines) to understand, explore, or research — not when editing or patching."
- **`ma_skeleton`** — "Use instead of `view` or `ma_smart_read` when you need a source file's API surface without implementation bodies."
- **`ma_minify_schema`** — "Use instead of `view` for JSON/YAML schema files you are reading to understand structure."

`ma_compress` and `ma_dedup` retain their current descriptions (no built-in competitor).

## Acceptance criteria

- [ ] `onSessionStart` hook registered in `extension/extension.mjs`
- [ ] Hook returns `{ additionalContext }` containing the MA preference rule text
- [ ] Preference text names `ma_smart_read`, `ma_skeleton`, `ma_minify_schema` with clear selection criteria
- [ ] Preference text names `view` as the tool to use only for edit preparation and line-precise inspection
- [ ] `ma_smart_read` description leads with "Use instead of `view`" and states the ≥ 200 line threshold
- [ ] `ma_skeleton` description leads with "Use instead of `view` or `ma_smart_read`" and references API surface
- [ ] `ma_minify_schema` description leads with "Use instead of `view`" and references JSON/YAML schemas
- [ ] `ma_compress` and `ma_dedup` descriptions unchanged

## Blocked by

None — can start immediately. Independent of the runtime helpers in #1.
