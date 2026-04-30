# Phase 1: Token Estimation & YAML Parsing

## Overview

Phase 1 introduces two external packages that improve the correctness of existing `ma` features without adding new commands or changing the CLI interface. **No JSON output fields are renamed** — the `--json` contract is preserved; only the accuracy of values improves.

| Package | Replaces | Files Changed |
|---------|----------|---------------|
| `tiktoken-go/tokenizer` | Heuristic token estimation | `internal/app/metrics.go` |
| `gopkg.in/yaml.v3` | Line-by-line YAML minification | `internal/schema/yaml.go` |

Both packages address correctness problems in the existing implementation — `ma`'s core metric (token counts) has ~15–30% error, and the YAML minifier silently mishandles edge cases like colons in string values.

---

## 1. Token Estimation: `tiktoken-go/tokenizer`

### Current State

`internal/app/metrics.go` uses a heuristic that takes `max(bytes/4, words×4/3)`:

```go
// internal/app/metrics.go:25-32
func approxTokens(input string) int {
    chars := len(input) / 4
    words := (len(strings.Fields(input)) * 4) / 3
    if chars > words {
        return chars
    }
    return words
}
```

This function is called by `Measure()` (line 14) which populates `Stats.InputApproxTokens` and `Stats.OutputApproxTokens`. Every command uses `Measure()` to report token savings.

### Target State

Replace the heuristic with `tiktoken-go/tokenizer` using the `cl100k_base` encoding (GPT-4 / Claude-approximate). The encoder is initialized once at package level and reused. **JSON field names remain unchanged** (`inputApproxTokens`, `outputApproxTokens`) — the values become cl100k_base BPE counts, which are still approximate relative to any specific model's tokenizer.

### Assumptions

- `cl100k_base` is an appropriate baseline encoding. It matches GPT-4 and approximates other model tokenizers. The counts are **exact for cl100k_base**, but still approximate relative to any specific model's tokenizer — the field name `ApproxTokens` remains accurate.
- `tiktoken-go/tokenizer` embeds vocabularies at compile time and makes no network calls. This is verified by inspecting the package's `go.mod`.
- The binary size increase (~4–5 MB) is acceptable for the accuracy gain.

### Implementation Steps

#### Step 1: Capture baseline binary size and benchmarks

Before any code changes:

```bash
go build -o ./ma ./cmd/ma && ls -lh ./ma
go test -bench=. -benchmem -count=3 ./... | tee benchmark-before.txt
```

Save the binary size and benchmark output for before/after comparison.

#### Step 2: Add dependency

```bash
go get github.com/tiktoken-go/tokenizer@latest
```

Verify transitive dependencies:

```bash
go mod graph | grep tokenizer
```

This pulls in one transitive dependency: `github.com/dlclark/regexp2`.

#### Step 3: Modify `internal/app/metrics.go`

1. Add a package-level encoder instance:

```go
import "github.com/tiktoken-go/tokenizer"

var defaultCodec tokenizer.Codec

func init() {
    codec, err := tokenizer.Get(tokenizer.Cl100kBase)
    if err != nil {
        panic("failed to initialize tokenizer: " + err.Error())
    }
    defaultCodec = codec
}
```

2. Replace `approxTokens` with `countTokens`:

```go
func countTokens(input string) int {
    ids, _, _ := defaultCodec.Encode(input)
    return len(ids)
}
```

3. **Keep `Stats` field names unchanged.** The struct fields `InputApproxTokens` and `OutputApproxTokens` (and their JSON tags `inputApproxTokens`, `outputApproxTokens`) are preserved. Only the computation changes — values are now cl100k_base BPE counts instead of the heuristic.

4. Update `Measure()` to call `countTokens` instead of `approxTokens`.

#### Step 4: Verify no compile errors

```bash
go build ./...
```

Since no field names changed, **no other files need updating** — all references to `InputApproxTokens`, `OutputApproxTokens`, and `AssertApproxTokenReductionAtLeast` remain valid. Run a full search to confirm:

```bash
rg 'ApproxToken' --type go
```

All hits should be in `internal/app/metrics.go` (the struct/function), `internal/app/metrics_test.go`, `internal/testutil/fixtures.go`, and call sites. No changes needed since field names are preserved.

#### Step 5: Update golden test expectations

The token count values in tests will change because cl100k_base counts differ from heuristic estimates. For each test that asserts specific token counts or reduction percentages:

- **Do not change the reduction thresholds** — if real token counts show a lower reduction percentage than the heuristic predicted, that's a genuine finding, not a test bug.
- If a reduction assertion fails with exact tokens, investigate whether the compression/minification is genuinely less effective than the heuristic suggested, and adjust the threshold only if the reduction is still meaningful.
- Run the specific tests to identify failures:

```bash
go test -v -run 'TestMeasure|TestTokenReduction|TestCompressReduction|TestOptimize|TestMinifySchema|TestTrimImportsReduction|TestDedupReduction|TestCompactHistoryReduction' ./...
```

#### Step 6: Add tokenizer-specific tests

Add a test in `internal/app/metrics_test.go` that verifies:

- Known inputs produce expected exact cl100k_base token counts (regression anchor)
- The encoder handles empty strings, Unicode, code, and markdown
- `countTokens` is deterministic across runs

```go
func TestCountTokensDeterministic(t *testing.T) {
    input := "Hello, world!"
    expected := countTokens(input)
    for i := 0; i < 100; i++ {
        if got := countTokens(input); got != expected {
            t.Fatalf("non-deterministic: run %d got %d, expected %d", i, got, expected)
        }
    }
}
```

#### Step 7: Capture after benchmarks and binary size

```bash
go build -o ./ma ./cmd/ma && ls -lh ./ma
go test -bench=. -benchmem -count=3 ./... | tee benchmark-after.txt
diff benchmark-before.txt benchmark-after.txt
```

Document the delta in the PR description.

#### Step 8: Update README.md

Change any "approximate token" language to "cl100k_base token counts". Do **not** claim "exact" without qualification — clarify that counts are exact for the cl100k_base encoding but approximate relative to any specific model's tokenizer.

### Guardrails

- **No JSON output contract change**: Field names `inputApproxTokens` and `outputApproxTokens` are preserved. No downstream consumers break.
- **Offline guarantee preserved**: `tiktoken-go/tokenizer` embeds vocabularies at compile time. No network calls. Verify by checking the package's `go.mod` has no HTTP client dependencies.
- **Determinism preserved**: BPE tokenization is deterministic for the same encoding. Step 6 adds a regression test.
- **Binary size impact**: Expected +4–5 MB. Measured in Steps 1 and 7 with `go build -o ./ma ./cmd/ma && ls -lh ./ma`.
- **No new CLI flags in this step**: The encoding is hardcoded to `cl100k_base`. A `--encoding` flag is a future feature, not part of this phase.

---

## 2. YAML Parsing: `gopkg.in/yaml.v3`

### Current State

`internal/schema/yaml.go` processes YAML by splitting on newlines and using string matching:

```go
// internal/schema/yaml.go:14-50
func MinifyYAML(input string) (string, error) {
    lines := strings.Split(input, "\n")
    var out []string
    skipIndent := -1
    for _, line := range lines {
        if err := validateSupportedYAMLLine(line); err != nil {
            return "", err
        }
        // ...indentation-based key skipping...
    }
}
```

Known correctness issues:
- Colons in string values (`title: "key: value"`) are misparsed
- Block scalar indicators (`|`, `>`) are not handled
- Multi-line string values can confuse the indentation tracker
- Comments attached to removed keys are silently lost

### Target State

Replace line-by-line processing with `gopkg.in/yaml.v3`'s `yaml.Node` API, which provides a typed AST while preserving document structure.

### Assumptions

- `gopkg.in/yaml.v3` is already an indirect dependency in `go.sum`. Promoting it to direct adds zero new transitive deps.
- Comment fidelity is **out of scope** — `yaml.v3`'s `Marshal` may reflow or drop comments. This is acceptable because the output is consumed by LLMs, not humans, and the current line-based approach also loses comments attached to removed keys. If comment preservation becomes a requirement, it can be addressed in a follow-up.
- The removable key set (`description`, `default`, `examples`) is extracted from the current implementation and is correct.

### Implementation Steps

#### Step 1: Add dependency

```bash
go get gopkg.in/yaml.v3@latest
```

This package is already an indirect dependency (`go.yaml.in/yaml/v3 v3.0.4` in `go.sum`). Adding it as a direct dependency adds zero new transitive dependencies.

#### Step 2: Rewrite `internal/schema/yaml.go`

Replace `MinifyYAML` with an AST-based implementation. **Critical**: the pre-parse tab check must run before `yaml.Unmarshal` to preserve the existing error message contract. `yaml.Unmarshal` may accept or reject tabs differently than the current validator, so we enforce the rejection ourselves first.

```go
import "gopkg.in/yaml.v3"

var removableYAMLKeys = map[string]struct{}{
    "description": {},
    "default":     {},
    "examples":    {},
}

func MinifyYAML(input string) (string, error) {
    // Pre-parse validation: reject tabs before yaml.Unmarshal
    // to preserve the exact "unsupported yaml feature: tabs" error contract.
    if err := rejectTabs(input); err != nil {
        return "", err
    }

    var doc yaml.Node
    if err := yaml.Unmarshal([]byte(input), &doc); err != nil {
        return "", err
    }

    if err := validateNode(&doc); err != nil {
        return "", err
    }

    pruneKeys(&doc, removableYAMLKeys)

    out, err := yaml.Marshal(&doc)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(out)), nil
}

// rejectTabs preserves the existing tab-rejection behavior.
// yaml.Unmarshal may or may not reject tabs depending on context,
// so we enforce rejection ourselves to maintain a stable error contract.
func rejectTabs(input string) error {
    for _, line := range strings.Split(input, "\n") {
        if strings.ContainsRune(line, '\t') {
            return fmt.Errorf("unsupported yaml feature: tabs")
        }
    }
    return nil
}
```

#### Step 3: Implement `pruneKeys`

Walk the node tree and remove mapping entries whose key matches a removable name:

```go
func pruneKeys(node *yaml.Node, keys map[string]struct{}) {
    if node.Kind == yaml.DocumentNode {
        for _, child := range node.Content {
            pruneKeys(child, keys)
        }
        return
    }
    if node.Kind != yaml.MappingNode {
        for _, child := range node.Content {
            pruneKeys(child, keys)
        }
        return
    }
    // MappingNode.Content is [key₁, value₁, key₂, value₂, ...]
    filtered := make([]*yaml.Node, 0, len(node.Content))
    for i := 0; i < len(node.Content); i += 2 {
        key := node.Content[i]
        val := node.Content[i+1]
        if _, remove := keys[key.Value]; remove {
            continue
        }
        pruneKeys(val, keys)
        filtered = append(filtered, key, val)
    }
    node.Content = filtered
}
```

#### Step 4: Implement `validateNode`

Preserve the existing rejection of anchors, aliases, and merge keys, but do it via the AST instead of string matching. **Note**: `yaml.Unmarshal` into `yaml.Node` parses anchors, aliases, and merge keys as valid YAML syntax (they create `AliasNode` types and set `node.Anchor` fields), so `validateNode` will always run — these features don't cause parse errors.

```go
func validateNode(node *yaml.Node) error {
    if node.Kind == yaml.AliasNode {
        return fmt.Errorf("unsupported yaml feature: aliases")
    }
    if node.Anchor != "" {
        return fmt.Errorf("unsupported yaml feature: anchors")
    }
    // Check for merge keys (<<)
    if node.Kind == yaml.MappingNode {
        for i := 0; i < len(node.Content); i += 2 {
            if node.Content[i].Value == "<<" {
                return fmt.Errorf("unsupported yaml feature: merge keys")
            }
        }
    }
    for _, child := range node.Content {
        if err := validateNode(child); err != nil {
            return err
        }
    }
    return nil
}
```

**Error contract guarantee**: The validation order is:
1. `rejectTabs(input)` — pre-parse, returns `"unsupported yaml feature: tabs"`
2. `yaml.Unmarshal` — parse errors are surfaced as-is (these are malformed YAML, not unsupported features)
3. `validateNode(&doc)` — post-parse, returns exact `"unsupported yaml feature: anchors|aliases|merge keys"`

Since anchors, aliases, and merge keys are valid YAML syntax, `yaml.Unmarshal` will never reject them — `validateNode` is guaranteed to run and produce the correct error message for these cases.

#### Step 5: Remove dead code

After rewriting, the following functions are no longer needed:
- `validateSupportedYAMLLine` (replaced by `validateNode`)
- `yamlKey` (replaced by AST key access)
- `leadingSpaces` (replaced by AST scope)

#### Step 6: Update YAML tests

**Preserve all existing test cases** — they define the expected behavior:

- `TestMinifyYAMLSchema` — golden file test, may need updated expected output if `yaml.Marshal` formats differently than the original line-based output
- `TestMinifyYAMLRejectsUnsupportedFeatures` — must still pass with same error messages
- `TestMinifyYAMLAcceptsLiteralAmpersand` — must still accept `AT&T`
- `TestMinifyYAMLAcceptsLiteralAsterisk` — must still accept wildcard `*`

**Add new edge-case tests:**

- Colon in string value: `title: "key: value"` → key `title` preserved, no misparsing
- Block scalar: `description: |\n  multi\n  line` → entire block removed
- Nested mapping under removable key → entire subtree removed
- Empty document after pruning → valid empty YAML output
- Tab rejection: input with tab indentation → error `"unsupported yaml feature: tabs"` (exact message match)
- Tab rejection ordering: tabs are rejected _before_ any parse errors, preserving the existing contract
- Anchor rejection: input with `&anchor` → error `"unsupported yaml feature: anchors"` (exact message match)
- Alias rejection: input with `*alias` → error `"unsupported yaml feature: aliases"` (exact message match)
- Merge key rejection: input with `<<:` → error `"unsupported yaml feature: merge keys"` (exact message match)

**Comment fidelity is explicitly out of scope**: `yaml.v3` may drop or reflow comments during marshal. This is acceptable — the output targets LLMs, not human readers. Do not add tests that assert comment preservation.

#### Step 7: Update golden test fixtures

The expected output in `testdata/schema/tool.schema.expected.yaml` may need updating if `yaml.Marshal` produces different whitespace than the line-based approach. This is acceptable as long as the YAML is semantically equivalent.

### Guardrails

- **All existing YAML tests must pass** (modulo expected output formatting). If a test fails, determine whether the old or new behavior is correct before updating the expectation.
- **Rejection behavior preserved**: Anchors, aliases, merge keys, and tabs must still be rejected with the **exact same error messages**. The tab check runs _before_ `yaml.Unmarshal` to guarantee the error contract. Tests already cover this — verify they pass with exact message matching.
- **No new YAML features supported**: The minifier still rejects anchors/aliases/merge keys/tabs. The scope is correctness, not feature expansion.
- **Output must be valid YAML**: Marshal the output and unmarshal it again to verify round-trip validity. Add a test for this.
- **Comment fidelity out of scope**: `yaml.v3` may drop or reflow comments. This is acceptable for LLM-consumed output. Do not add comment-preservation assertions.
- **Formatting deltas documented**: If `yaml.Marshal` changes indentation style (e.g., 2-space vs 4-space), document the change. The output is consumed by LLMs, not humans, so formatting differences are acceptable as long as the YAML is valid.

---

## Acceptance Criteria

All criteria must be met before Phase 1 is considered complete.

### Functional

- [ ] `go test ./...` passes with zero failures
- [ ] `go build -o ./ma ./cmd/ma` succeeds
- [ ] All 8 commands produce correct output. Smoke test with these exact commands:
  ```bash
  ./ma compress testdata/prose/project-notes.input.md
  ./ma validate testdata/prose/project-notes.input.md testdata/prose/project-notes.expected.md
  ./ma optimize-md testdata/markdown/guide.input.md
  ./ma minify-schema testdata/schema/tool.schema.yaml
  ./ma minify-schema testdata/schema/tool.schema.json
  ./ma skeleton testdata/code/sample.go
  ./ma skeleton testdata/code/sample.ts
  ./ma trim-imports testdata/code/import-heavy.ts
  ./ma dedup testdata/dedup/rules-a.md testdata/dedup/rules-b.md
  ./ma compact-history testdata/history/transcript.json
  ```
- [ ] Token counts in `--json` output are cl100k_base BPE counts, not heuristic estimates
- [ ] JSON field names unchanged (`inputApproxTokens`, `outputApproxTokens`) — no `--json` contract break
- [ ] YAML minification correctly handles colons in string values, block scalars, and nested removable keys
- [ ] YAML minification still rejects anchors, aliases, merge keys, and tabs with exact same error messages

### Non-Functional

- [ ] Binary size increase documented (expected: +4–5 MB from tiktoken vocabularies). Measured with `go build -o ./ma ./cmd/ma && ls -lh ./ma` before and after.
- [ ] Benchmark results compared before/after: `go test -bench=. -benchmem -count=3 ./...`
- [ ] No new network calls introduced (offline guarantee preserved)
- [ ] All commands remain deterministic (same input → same output)

### Documentation

- [ ] `README.md` updated: change "approximate token" language to "cl100k_base token counts"
- [ ] `go.mod` shows new direct dependencies: `tiktoken-go/tokenizer`, `gopkg.in/yaml.v3`

### Testing

- [ ] New test: exact cl100k_base token count for known input (regression anchor)
- [ ] New test: tokenizer handles empty string, Unicode, code blocks, markdown
- [ ] New test: tokenizer is deterministic (100 runs, same result)
- [ ] New test: YAML colon-in-string-value edge case
- [ ] New test: YAML block scalar removal
- [ ] New test: YAML round-trip validity (marshal → unmarshal → compare)
- [ ] New test: YAML tab rejection with exact error message match
- [ ] New test: YAML tab rejection fires before parse validation
- [ ] Existing reduction threshold tests reviewed and adjusted only if justified
- [ ] All benchmarks run and results documented (no significant regression)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `yaml.Marshal` changes output formatting | High | Low | Formatting differences are acceptable for LLM-consumed output. Update golden test fixtures. |
| Reduction threshold tests fail with cl100k_base counts | Medium | Low | Investigate each failure. Adjust threshold only if the reduction is still meaningful — the old threshold was based on inaccurate counts. |
| `tiktoken-go/tokenizer` panics on edge-case input | Low | High | Add defensive tests for empty strings, binary content, very long inputs. |
| `gopkg.in/yaml.v3` interprets YAML differently than expected | Low | Medium | Run all existing YAML test fixtures through the new implementation before committing. |
| `yaml.v3` drops or reflows comments | High | Low | Explicitly out of scope. Output targets LLMs, not humans. Document in PR description. |
| `yaml.Unmarshal` accepts tabs that the current code rejects | Medium | Medium | Pre-parse `rejectTabs()` runs before `Unmarshal` to preserve the exact error contract. |

---

## Rollback Plan

If Phase 1 causes issues after merge:

1. **Quick rollback**: `git revert` the Phase 1 commit. The old heuristic and line-by-line YAML code are restored.
2. **Partial rollback (tokenizer only)**: Revert `internal/app/metrics.go` changes and remove `tiktoken-go/tokenizer` from `go.mod`. YAML changes can remain independently.
3. **Partial rollback (YAML only)**: Revert `internal/schema/yaml.go` changes and remove `gopkg.in/yaml.v3` from `go.mod`. Tokenizer changes can remain independently.

No downstream consumers break on rollback because **no JSON field names were changed**.
