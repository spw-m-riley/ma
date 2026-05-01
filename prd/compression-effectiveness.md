# Compression effectiveness PRD

## Problem Statement

`ma compress` is the tool's core command, but it does not meaningfully reduce real-world input. On the instruction files, READMEs, skill files, and documentation that users actually pass through it, the command achieves 0.0–0.1% token reduction — effectively zero. A user who runs `ma compress` on their Copilot instruction files, project docs, or LLM system prompts gets back nearly identical output and has no reason to trust that the tool is doing useful work.

The underlying cause is invisible to the user but stark in measurement: the compression engine has 27 rules, and those rules match virtually nothing in real English prose. They were tuned to synthetic test fixtures containing exact phrases like `"The goal of this tool is to"` and `"while still keeping the result"`, which never appear in production text. Meanwhile, real instruction files contain hundreds of compressible patterns — contractions, wordy phrases, filler adverbs, hedging language, verbose technical terms — that the engine ignores entirely.

This makes the compress command unreliable as the tool's flagship feature. A user who integrates `ma` into their workflow expecting meaningful context savings gets negligible results, and the dashboard shows runs with near-zero reduction, undermining trust in the entire product.

## Solution

Make `ma compress` reliably reduce real-world prose by 15–30% on verbose content and 3–8% on already-terse text, without changing the tool's core guarantees: deterministic, offline, no external dependencies, structurally safe.

The compression engine should target the linguistic categories that actually appear in instruction files, documentation, and LLM-generated text — contractions, subject elision, hedging removal, wordy phrase compression, filler adverb removal, transition simplification, demonstrative pruning, and technical abbreviations. Rules should be organized by category and applied in a strict phase order that prevents semantic errors, particularly negation inversion.

A user who runs `ma compress` on a verbose system prompt should see a 20–30% token reduction. A user who runs it on a terse instruction file should see a 3–8% reduction. The output should remain structurally identical: same headings, same code blocks, same URLs, same paths, same bullet count. The only changes should be shorter, more direct prose that an LLM can reconstruct without loss of meaning.

## User Stories

1. As a user compressing instruction files, I want the tool to reduce verbose English patterns like `"in order to"`, `"it is important to note that"`, and `"do not"`, so that real prose gets meaningfully shorter instead of passing through unchanged.
2. As a user compressing technical documentation, I want common long-form terms like `configuration`, `repository`, and `authentication` shortened to their standard abbreviations, so that technical prose gets tighter without losing clarity.
3. As a user compressing LLM-generated text, I want hedging phrases like `"please note that"`, `"it is recommended to"`, and `"be sure to"` removed, so that verbose AI-written content becomes direct and token-efficient.
4. As a user compressing instruction files that contain negation, I want `"You should not delete files"` to become `"You shouldn't delete files"` rather than `"delete files"`, so that meaning is preserved and never inverted.
5. As a user who has already written terse prose, I want the tool to still find 3–8% savings from contractions and abbreviations, so that the command remains useful even on carefully edited content.
6. As a user, I want compression to leave headings, code fences, inline code, URLs, and file paths untouched, so that structural elements remain intact after compression.
7. As a user, I want the bullet count to remain stable after compression, so that list structure is preserved even when individual bullet text gets shorter.
8. As a user running the tool in a pipeline, I want the same input to always produce the same output, so that I can trust compression in automated workflows without worrying about nondeterminism.
9. As a user, I want to see token reduction stats in the output and on the dashboard, so that I can judge whether compression is providing real value on my specific content.
10. As a user compressing files with mixed content, I want prose inside bullet points and paragraphs to be compressed while code blocks and headings are left alone, so that only natural language is transformed.
11. As a user who cares about readability, I want compressed output to remain grammatical and understandable by humans, so that the output is useful for both LLMs and people who read it.
12. As a user, I want the validate check to continue catching structural drift after compression, so that I have a safety net against rules that accidentally damage document structure.
13. As a user compressing files that contain transition words like `"However,"` and `"Furthermore,"`, I want them simplified or removed, so that connective filler does not consume tokens when the meaning is clear from context.
14. As a user compressing files with subject-heavy phrasing like `"You should always"` and `"You will need to"`, I want the redundant subject removed, so that imperative-style instructions become more direct.
15. As a maintainer adding new rules, I want rules organized by linguistic category so that I can add or test rules in one category without affecting others.
16. As a maintainer, I want each rule category applied in a defined phase order, so that safety-critical ordering (contractions before subject elision) is enforced by the engine rather than by convention.
17. As a maintainer writing tests, I want test fixtures that represent real-world workloads — system prompts, API docs, project READMEs — so that reduction assertions reflect actual performance rather than synthetic best-case scenarios.
18. As a maintainer, I want reduction threshold tests that assert meaningful minimums (≥15% on verbose content, ≥3% on terse content), so that regressions in real-world effectiveness are caught automatically.

## Implementation Decisions

- **Phased rule application.** The compression engine should apply rules in strict category phases rather than a single flat pass. Phase order matters for correctness: contraction rules must run before subject-elision rules to prevent negation inversion (`"should not"` → `"shouldn't"` must happen before `"You should"` → `""` is attempted). The recommended phase order is: contractions, subject elision, hedging removal, wordy phrases, transitions, filler adverbs, demonstrative pruning, technical abbreviations, then whitespace normalization.
- **Rule categories as separate compilation units.** Each rule category (contractions, abbreviations, wordy phrases, etc.) should be defined in its own unit and export a function that returns its rule set. This keeps each category testable in isolation and lets maintainers add rules to one category without touching others. The registry should compose all categories into an ordered pipeline.
- **~150–200 rules across 8 categories.** The current 27 rules should be replaced with a comprehensive set covering: contractions (~15 rules), subject elision (~6), hedging/softener removal (~8), wordy phrase compression (~22), transition simplification (~8), filler adverb removal (~6), demonstrative/determiner pruning (~11), and technical abbreviations (~16). The existing 27 rules should be reviewed for overlap and migrated into the appropriate categories or retired.
- **Zone system unchanged.** The existing zone-splitting architecture (prose, code fences, headings, inline code, URLs, paths) is correct and complete. Rules continue to apply only to prose zones. No changes needed to the zone parser.
- **Validate constraints unchanged.** The existing structural validator (heading preservation, code block preservation, URL preservation, path preservation, bullet count stability) is compatible with all proposed rule categories. No changes needed, but it should be exercised against the new rule set to confirm.
- **Write contract unchanged.** The `--write` flag, backup-swap mechanism, and read-only default behavior remain as-is. The only change is that the engine produces more reduction, which flows through the existing result and stats model.
- **No new external dependencies.** All rules are regex-based, implemented in Go standard library. No NLP libraries, no models, no network calls. This maintains the tool's offline and deterministic guarantees.
- **Sensitive-path checks unchanged.** The existing `detect.IsSensitivePathResolved()` gate continues to run before any file read. New rules do not affect the file-reading path.

## Testing Decisions

- **Realistic golden fixtures.** Add test fixtures that represent real workloads: a verbose system prompt, a technical API doc section, a project README, and an LLM-generated instructional guide. Each fixture should have a corresponding expected output and a minimum reduction threshold assertion. Verbose fixtures should assert ≥20% token reduction; terse fixtures should assert ≥3%.
- **Negation preservation tests.** Dedicated tests for the negation-inversion bug: inputs containing `"should not"`, `"do not"`, `"cannot"`, and `"must not"` followed by verbs should produce contracted forms, never bare verbs. This is the highest-priority safety test.
- **Per-category unit tests.** Each rule category should have its own test exercising representative patterns: contractions produce correct apostrophe forms, abbreviations shorten only word-interior matches, subject elision does not match after contractions, filler removal handles comma-separated adverbs.
- **Phase ordering tests.** A test that verifies the engine applies phases in the correct order by providing input where wrong ordering would produce incorrect output (the negation-inversion case is the canonical example).
- **Structural preservation tests.** End-to-end tests that run compression on mixed-content documents and verify via the existing validator that headings, code blocks, URLs, paths, and bullet counts are preserved.
- **Existing fixture compatibility.** The three existing prose fixtures should be updated with new expected outputs reflecting the expanded rules. If any existing expected output becomes incorrect under the new rules, the fixture should be re-baselined rather than the rules constrained to match legacy expectations.
- **Prior art in the codebase.** The closest existing tests are the golden-fixture reduction threshold test and the structural preservation tests via `validate.Compare()`. The new tests should follow the same patterns: golden fixture pairs loaded via the test utility helper, threshold assertions via the approximate token reduction helper, and structural checks via the validator.

## Out of Scope

- **POS-aware compression.** Using a part-of-speech tagger (spaCy-style) for smarter token removal. This would deliver higher reduction but requires either a Go POS library or an external dependency, violating the tool's offline/no-dependency constraint. Deferred as a potential future `--aggressive` mode.
- **Article removal.** Stripping `a`, `an`, `the` globally. This matches what Caveman Compression recommends for maximum reduction, but makes output telegraphic and less human-readable. Deferred as a potential opt-in aggressive mode.
- **Compression levels flag.** A `--level` flag (conservative/standard/aggressive) that enables different rule subsets. Useful but adds surface area; the initial implementation should ship the full safe rule set as the single default. Levels can be layered on once the base performance is proven.
- **LLM-based compression.** Approaches like LLMLingua that use a small language model to score token predictability. These achieve 50–80% reduction but require Python, PyTorch, and a model file — incompatible with `ma`'s design constraints.
- **Markdown structural optimization.** The `optimize-md` command handles markdown-specific cleanup (blank lines, list markers, tables). This PRD covers only prose compression, not markdown structure.
- **New CLI flags or subcommands.** No new commands or flags are introduced. The existing `ma compress` interface and `--write`/`--json` flags remain unchanged.

## Further Notes

- **Existing rule migration.** The current 27 rules should be audited against the new categories. Some may map cleanly into the new taxonomy; others may be too specific to justify keeping. The migration should be explicit: each legacy rule is either placed in a category or retired with a note.
- **Abbreviation boundary safety.** Technical abbreviations like `configuration` → `config` are safe in prose zones because the zone system protects headings, code blocks, and inline code. However, edge cases may exist where a technical term appears in a heading that is styled as prose (e.g., a bullet that mentions "Configuration"). The abbreviation rules should use word-boundary anchors and be tested against such edge cases.
- **Dashboard impact.** Once compression effectiveness improves, the dashboard will show meaningfully higher reduction stats for compress runs. No dashboard changes are needed — the existing stats model already tracks token, word, and byte deltas.
- **Benchmark comparison.** The research compared the proposed approach with Caveman Compression NLP (spaCy, 15–30% reduction) and found that ~150 regex rules achieve the same range while staying pure Go. This validates the regex-only approach for the tool's design constraints.
- **Contraction readability.** Contracting `"do not"` → `"don't"` improves token efficiency but changes the register of the text. For LLM context reduction this is fine (LLMs handle contractions natively), but users who inspect compressed output may notice the tonal shift. This is an acceptable tradeoff for the token savings.
