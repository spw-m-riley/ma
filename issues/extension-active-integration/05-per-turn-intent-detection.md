## Parent

PRD: `prd/extension-active-integration.md`

## What to build

Implement the `onUserPromptSubmitted` hook in `extension/extension.mjs` that detects read-intent prompts and injects per-turn MA guidance.

When the user's prompt suggests the agent is about to read or explore files (e.g., keywords like "read", "look at", "show me", "explore", "understand", "what does X do"), inject `{ additionalContext }` with a brief reminder to prefer `ma_smart_read` over `view` for understanding reads.

Design considerations:

- **Keyword matching** — Use a focused set of read-intent keywords/phrases. Avoid over-matching on prompts that are clearly about editing, debugging, or running commands.
- **Deduplication** — Track whether the nudge has already fired this session (via a module-level flag or lightweight cache) so the same reminder is not injected every turn. Consider resetting the flag after N turns or when the prompt context shifts.
- **Complement, don't duplicate** — The `onSessionStart` hook already sets the session-level preference. This per-turn hook adds a timely reminder when a read is imminent, not a repetition of the full preference rule.

## Acceptance criteria

- [ ] `onUserPromptSubmitted` hook registered in `extension/extension.mjs`
- [ ] Hook detects read-intent prompts via keyword matching and injects `{ additionalContext }`
- [ ] Injected context is a brief MA preference reminder, not a repeat of the full session-start rule
- [ ] Hook does not fire on prompts with no read-intent signals (editing, running tests, etc.)
- [ ] Hook deduplicates — does not inject the same reminder on consecutive read-intent prompts
- [ ] Hook returns `undefined` (no injection) when dedup suppresses or no read-intent is detected
- [ ] Keyword set is documented in a comment or constant for future tuning

## Blocked by

- #2 — Session-start preference injection (so the per-turn nudge complements rather than conflicts with the session-level rule)
