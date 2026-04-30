# Phase 2: Markdown Parsing with Goldmark

## Overview

Phase 2 introduces `yuin/goldmark` to replace hand-rolled regex and state-machine markdown parsing in two packages with a CommonMark-compliant AST. No new features are added — the same headings, code blocks, URLs, paths, and zone types are extracted, but using a proper parser instead of fragile string matching.

| Package | Replaces | Files Changed |
|---------|----------|---------------|
| `yuin/goldmark` | Regex heading/code-block extraction in validate | `internal/validate/validate.go` |
| `yuin/goldmark` | Manual fence-state machine in zones | `internal/zones/zones.go` |

**Explicitly out of scope**: `internal/markdown/optimize.go` remains line-by-line. Goldmark has no markdown-to-markdown renderer, so the optimizer stays unchanged.

---

## Why Goldmark

- **Zero external dependencies** — `go.mod` contains only `go 1.22`
- **4,735 stars**, actively maintained, used by Hugo and Gitea
- **CommonMark-compliant AST** with typed node types and a walker
- Provides `ast.KindHeading`, `ast.KindFencedCodeBlock`, `ast.KindCodeSpan`, `ast.KindAutoLink`, `ast.KindLink`
- Visitor pattern via `ast.Walk(node, func)` replaces manual state machines

---

## 1. Validate Package: `internal/validate/validate.go`

### Current State

The `Compare` function uses four regex patterns and a hand-rolled `extractCodeBlocks` state machine:

```go
var headingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)      // line 15
var urlPattern = regexp.MustCompile(`https?://[^\s)]+`)              // line 16
var pathPattern = regexp.MustCompile(...)                            // line 17
var bulletPattern = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)           // line 18
```

`extractCodeBlocks` (lines 80–108) is a manual fence tracker with `detectFence`, `closesFence`, and `countFenceChars`.

**Known limitations:**
- The heading regex matches `#` inside code blocks (e.g., `# comment` in a bash block is falsely extracted as a heading)
- The URL regex matches URLs inside code blocks, which may give false positives
- The fence state machine duplicates logic already in `zones.go`

### Target State

Use goldmark's AST to extract headings and code blocks. This eliminates false positives from code blocks because the AST distinguishes between structural headings and `#` characters inside fenced code.

### Implementation Steps

#### Step 1: Add dependency

```bash
go get github.com/yuin/goldmark@latest
```

Zero transitive dependencies added.

#### Step 2: Create `internal/validate/ast.go`

Isolate the goldmark-based extraction into a separate file to keep `validate.go` focused on comparison logic.

**Critical design decision**: Goldmark is used **only for structural classification** — identifying which byte ranges in the source are headings, code blocks, etc. Text is then extracted by **slicing the original source bytes** at the identified positions. This approach never interprets inline AST nodes (CodeSpan, Emphasis, Link, etc.), so no inline content can be dropped regardless of node type.

```go
package validate

import (
    "strings"

    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/text"
)

func extractHeadingsAST(source []byte) []string {
    reader := text.NewReader(source)
    parser := goldmark.DefaultParser()
    doc := parser.Parse(reader)

    var headings []string
    ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }
        if node.Kind() == ast.KindHeading {
            // Get the raw source line for this heading.
            // nodeSourceLine returns the full line including # prefix.
            // Strip #{1,6}\s+ to match current regex group(1) behavior.
            line := nodeSourceLine(node, source)
            line = strings.TrimRight(line, "\n\r")
            idx := strings.IndexByte(line, ' ')
            if idx >= 0 {
                line = line[idx+1:]
            }
            headings = append(headings, line)
        }
        return ast.WalkContinue, nil
    })
    return headings
}

func extractCodeBlocksAST(source []byte) []string {
    reader := text.NewReader(source)
    parser := goldmark.DefaultParser()
    doc := parser.Parse(reader)

    var blocks []string
    ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }
        if node.Kind() == ast.KindFencedCodeBlock {
            // Extract the FULL fenced block from source bytes,
            // including fence open/close lines and info string.
            blocks = append(blocks, fencedBlockSourceText(node, source))
        }
        return ast.WalkContinue, nil
    })
    return blocks
}

// nodeSourceLine extracts the full original source line(s) for a block node.
// For ATX headings this includes the # prefix and all inline markup as-is.
// Works by finding anchor byte positions from descendants, then expanding
// to full line boundaries — the line expansion is what guarantees all
// inline syntax (backticks, **, [](), etc.) is captured.
//
// FALLBACK: If nodeByteRange returns empty (start >= end), this is
// unexpected for headings/paragraphs. Rather than silently returning "",
// a test assertion must catch this. During implementation, if this case
// is ever hit in practice, investigate the goldmark node structure and
// add type-specific segment extraction for the missing node type.
func nodeSourceLine(node ast.Node, source []byte) string {
    start, end := nodeByteRange(node, source)
    if start >= end {
        return ""
    }
    // Expand to full line boundaries
    for start > 0 && source[start-1] != '\n' {
        start--
    }
    for end < len(source) && source[end-1] != '\n' {
        end++
    }
    return string(source[start:end])
}

// nodeByteRange finds anchor byte positions within a node's source range
// by walking ALL descendants and collecting Lines() and ast.Text segment
// positions. The returned (start, end) may not cover every syntax byte
// (e.g. backtick delimiters around CodeSpan are between Text segments),
// but that is fine: callers expand to full line boundaries via
// nodeSourceLine, which captures the complete original source line
// including all syntax characters.
//
// If no segments are found (start >= end), the node has no recoverable
// source content. nodeSourceLine returns "" in that case, which is safe:
// headings always have at least one Text child, and empty paragraphs
// are not emitted by goldmark. Add a regression test for a heading
// with only inline markup (e.g. "## `code`") to verify coverage.
func nodeByteRange(node ast.Node, source []byte) (int, int) {
    start := len(source)
    end := 0

    ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }
        // Check Lines() (block content lines)
        childLines := child.Lines()
        for i := 0; i < childLines.Len(); i++ {
            seg := childLines.At(i)
            if seg.Start < start { start = seg.Start }
            if seg.Stop > end { end = seg.Stop }
        }
        // Check Text segments (inline text nodes)
        if t, ok := child.(*ast.Text); ok {
            if t.Segment.Start < start { start = t.Segment.Start }
            if t.Segment.Stop > end { end = t.Segment.Stop }
        }
        return ast.WalkContinue, nil
    })

    return start, end
}

// fencedBlockSourceText reconstructs the full fenced code block
// from the source, including the opening/closing fence lines.
func fencedBlockSourceText(node ast.Node, source []byte) string {
    lines := node.Lines()
    if lines.Len() == 0 {
        return ""
    }
    contentStart := lines.At(0).Start
    contentEnd := lines.At(lines.Len() - 1).Stop

    // Walk backward from content to find fence opener start-of-line
    fenceStart := contentStart
    for fenceStart > 0 && source[fenceStart-1] != '\n' {
        fenceStart--
    }
    if fenceStart > 0 {
        fenceStart--
        for fenceStart > 0 && source[fenceStart-1] != '\n' {
            fenceStart--
        }
    }

    // Walk forward from content to find fence closer end-of-line
    fenceEnd := contentEnd
    for fenceEnd < len(source) && source[fenceEnd] != '\n' {
        fenceEnd++
    }
    if fenceEnd < len(source) {
        fenceEnd++
    }

    return string(source[fenceStart:fenceEnd])
}
```

**Why `nodeByteRange` + line-boundary expansion?** Both reviewers identified that per-node-type inline traversal fails for leaf nodes like `CodeSpan` and `AutoLink` that don't use `ast.Text` children. The two-step approach avoids this: `nodeByteRange` finds anchor positions from any descendant segments (it doesn't need to cover every syntax byte), then `nodeSourceLine` expands to full line boundaries, which captures the complete original source line including all inline markup syntax (backticks, `**`, `[](url)`, etc.). This design means no inline node type needs special handling — the line-boundary expansion is the guarantee, not the byte range alone.

#### Step 3: Update `validate.go`

Replace the regex-based heading extraction and manual code block extraction:

```go
func Compare(original string, candidate string) Report {
    report := Report{Valid: true}

    // AST-based extraction replaces regex
    if !equalStrings(
        extractHeadingsAST([]byte(original)),
        extractHeadingsAST([]byte(candidate)),
    ) {
        report.Valid = false
        report.Errors = append(report.Errors, "heading mismatch")
    }

    if !equalStrings(
        extractCodeBlocksAST([]byte(original)),
        extractCodeBlocksAST([]byte(candidate)),
    ) {
        report.Valid = false
        report.Errors = append(report.Errors, "code block mismatch")
    }

    // URL and path extraction remain regex-based — goldmark's link nodes
    // only capture markdown links, not bare URLs in prose
    // (same for paths, which are ma-specific)
    // ...rest unchanged...
}
```

#### Step 4: Remove dead code from `validate.go`

After switching to AST extraction:
- Remove `headingPattern` regex (line 15) — replaced by `extractHeadingsAST`
- Remove `extractCodeBlocks` function (lines 80–108) — replaced by `extractCodeBlocksAST`
- Remove `detectFence`, `closesFence`, `countFenceChars` helper functions (lines 110–144)
- Keep `urlPattern`, `pathPattern`, `bulletPattern` — these are not replaced by goldmark

#### Step 5: Verify behavior parity

Run existing tests to confirm parity:

```bash
go test -v -run 'TestCompare|TestExtractCodeBlocks|TestReport' ./internal/validate/...
```

Key behavioral differences to check:

- **Heading inside code block**: The old regex would extract `# comment` from a bash code block. The new AST-based extraction correctly skips it. This is a **correctness improvement**, not a behavior change — but verify tests don't depend on the old (incorrect) behavior.
- **Code block content**: The revised `extractCodeBlocksAST` preserves the **full fenced block** including fence markers and info string, matching current `extractCodeBlocks` behavior. Both old and new implementations include fence lines in the output.

---

## 2. Zones Package: `internal/zones/zones.go`

### Current State

The `Split` function (lines 28–87) uses a manual fence-state machine to split markdown into typed zones:

```go
for _, line := range lines {
    trimmed := strings.TrimSpace(line)
    if inFence {
        fence.WriteString(line)
        if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
            inFence = false
            flushFence()
        }
        continue
    }
    if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
        flushProse()
        inFence = true
        fence.WriteString(line)
        continue
    }
    // ...
}
```

**Known limitations:**
- The fence closer doesn't verify it matches the opener's fence character or length (a `~~~` could close a ```` ``` ```` fence)
- Headings inside code blocks would be mis-classified as `Heading` zones
- No support for indented code blocks (though this may be intentional)

### Target State

Use goldmark's AST to produce the same zone types. The `Zone` struct and `Kind` constants remain unchanged — only the splitting logic is replaced.

### Implementation Steps

#### Step 1: Rewrite `Split` function

```go
import (
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/text"
)

func Split(input string) []Zone {
    if input == "" {
        return nil
    }

    source := []byte(input)
    reader := text.NewReader(source)
    parser := goldmark.DefaultParser()
    doc := parser.Parse(reader)

    var zones []Zone
    for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
        zones = append(zones, nodeToZones(child, source)...)
    }
    return zones
}
```

#### Step 2: Implement `nodeToZones`

Map AST node types to zone types. **All text is extracted by slicing original source bytes** at positions identified by `nodeByteRange` / `nodeSourceLine`. No inline AST nodes are interpreted.

```go
func nodeToZones(node ast.Node, source []byte) []Zone {
    switch node.Kind() {
    case ast.KindHeading:
        // Preserve the full original heading line including # prefix
        return []Zone{{Kind: Heading, Text: nodeSourceLine(node, source)}}
    case ast.KindFencedCodeBlock:
        // Preserve the full fenced block including fence markers, info string, and content
        return []Zone{{Kind: CodeFence, Text: fencedBlockSourceText(node, source)}}
    case ast.KindParagraph:
        return splitProtectedText(nodeSourceLine(node, source))
    default:
        // Lists, blockquotes, thematic breaks → treat as prose
        t := nodeSourceLine(node, source)
        if t != "" {
            return splitProtectedText(t)
        }
        return nil
    }
}
```

#### Step 3: Implement source-range text extraction

The zones package reuses the same `nodeByteRange` + source-slicing pattern from `validate/ast.go`. These helpers are duplicated (not shared) to keep the packages independent.

```go
// nodeSourceLine, nodeByteRange, fencedBlockSourceText
// — same implementation as validate/ast.go (see above).
// Duplicated here to avoid coupling validate and zones packages.
```

#### Step 4: Preserve `splitProtectedText`

The `splitProtectedText` function (lines 89–109) and its helpers (`findNextProtectedToken`, `appendTextZone`) handle inline-level splitting for URLs, paths, and inline code. **These remain unchanged** — goldmark's AST handles block-level structure, but the inline token extraction (which uses regex for URLs and paths) stays.

#### Step 5: Handle edge cases

- **Empty input**: Return `nil` (same as current behavior)
- **Trailing newlines**: Source byte ranges preserve original whitespace exactly. Verify zone texts match the original line content including newlines.
- **Heading text**: Source byte ranges preserve the full original line including `#` prefix and trailing newline — matching current behavior exactly. No decision needed.
- **Code fence text**: Source byte ranges preserve the full block including fence markers, info string, content, and closing fence — matching current behavior exactly. This is critical because `compress` writes `zone.Text` back unchanged.

#### Step 6: Remove dead code

After rewriting:
- Remove the old `Split` function body (manual fence state machine)
- Remove `flushProse` and `flushFence` closures (replaced by AST walker)
- Keep `splitProtectedText`, `findNextProtectedToken`, `appendTextZone` (inline splitting stays)
- Keep `urlPattern` and `pathPattern` regexes (used by inline splitting)

---

## Acceptance Criteria

All criteria must be met before Phase 2 is considered complete.

### Functional

- [ ] `go test ./...` passes with zero failures
- [ ] `go build -o ./ma ./cmd/ma` succeeds
- [ ] `ma validate` produces identical reports. Capture baseline before Phase 2, then compare after:
  ```bash
  # BEFORE Phase 2 implementation — capture baselines:
  ./ma validate testdata/prose/project-notes.input.md testdata/prose/project-notes.input.md > validate-self-before.txt 2>&1
  ./ma validate testdata/prose/project-notes.input.md testdata/prose/project-notes.expected.md > validate-cross-before.txt 2>&1

  # AFTER Phase 2 implementation — verify parity:
  ./ma validate testdata/prose/project-notes.input.md testdata/prose/project-notes.input.md > validate-self-after.txt 2>&1
  ./ma validate testdata/prose/project-notes.input.md testdata/prose/project-notes.expected.md > validate-cross-after.txt 2>&1
  diff validate-self-before.txt validate-self-after.txt
  diff validate-cross-before.txt validate-cross-after.txt
  ```
- [ ] `ma compress` produces identical output (uses zones internally). Capture baseline before Phase 2, then compare after:
  ```bash
  # BEFORE:
  ./ma compress testdata/prose/project-notes.input.md > compress-notes-before.txt
  ./ma compress testdata/prose/mixed-with-code.input.md > compress-mixed-before.txt

  # AFTER:
  ./ma compress testdata/prose/project-notes.input.md > compress-notes-after.txt
  ./ma compress testdata/prose/mixed-with-code.input.md > compress-mixed-after.txt
  diff compress-notes-before.txt compress-notes-after.txt
  diff compress-mixed-before.txt compress-mixed-after.txt
  ```
- [ ] `ma optimize-md` produces identical output. Capture baseline before Phase 2, then compare after:
  ```bash
  # BEFORE:
  ./ma optimize-md testdata/markdown/guide.input.md > optimize-before.txt

  # AFTER:
  ./ma optimize-md testdata/markdown/guide.input.md > optimize-after.txt
  diff optimize-before.txt optimize-after.txt
  ```
  Note: `optimize-md` does NOT use zones internally — it uses `validate.Compare` for post-optimization validation only. The zones changes affect `compress`, not `optimize-md`. Capture "before" baselines **before starting Phase 2 implementation**.
- [ ] Heading extraction no longer matches `#` inside code blocks (correctness improvement)
- [ ] Code block extraction preserves full fenced block including fence markers and info strings
- [ ] Zone splitting produces the same zone types, boundaries, and exact text content as the current implementation (modulo correctness improvements)

### Non-Functional

- [ ] Binary size increase documented (expected: minimal, goldmark is ~few hundred KB). Measured with `go build -o ./ma ./cmd/ma && ls -lh ./ma`
- [ ] Benchmark results compared before/after: `go test -bench=. -benchmem -count=3 ./internal/validate/... ./internal/zones/... ./internal/prose/...`
- [ ] No new network calls introduced
- [ ] All commands remain deterministic

### Testing

- [ ] All existing validate tests pass: `go test -v -run 'TestCompare|TestExtractCodeBlocks|TestReport' ./internal/validate/...`
- [ ] All existing zones tests pass: `go test -v -run 'TestSplit' ./internal/zones/...`
- [ ] New test: heading inside code block is NOT extracted as a heading
- [ ] New test: heading with inline code (`` # Hello `world` ``) preserves inline markup in extracted text
- [ ] New test: heading with emphasis (`# Hello **world**`) preserves markup in extracted text
- [ ] New test: heading with link (`# See [docs](url)`) preserves markup in extracted text
- [ ] New test: mismatched fence characters (e.g., `~~~` closing a ```` ``` ```` block) are handled correctly
- [ ] New test: nested code fences (4-backtick wrapping 3-backtick)
- [ ] New test: code fence extraction includes fence markers and info string (exact text regression)
- [ ] New test: code-span-only heading (`## \`code\``) — verifies `nodeByteRange` finds anchor positions through CodeSpan's child Text nodes and `nodeSourceLine` recovers the full heading line. Run: `go test -v -run 'TestCodeSpanOnlyHeading' ./internal/validate/...`
- [ ] New test: `nodeByteRange` empty-range detection — construct a synthetic case where `nodeByteRange` returns `start >= end` and verify `nodeSourceLine` returns `""` without panic. This is a defensive test; if it triggers on real markdown, investigate and add type-specific segment extraction for the missing node type.
- [ ] New test: zone `CodeFence` text includes full fenced block (fence + content + close)
- [ ] Existing reduction threshold tests still pass with AST-based extraction
- [ ] All benchmarks run and results documented

### Documentation

- [ ] `go.mod` shows new direct dependency: `github.com/yuin/goldmark`
- [ ] Code comments explain which parts use goldmark AST vs. regex (URL/path/bullet remain regex)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| CommonMark strictness rejects non-standard markdown in instruction files | Medium | Medium | Test with real-world instruction files (Copilot instructions, AGENTS.md) to verify goldmark parses them correctly. |
| Zone text changes break downstream consumers | Low | High | Source byte range extraction guarantees exact original content. Exact-text regression tests verify. |
| Goldmark parser is slower than regex for simple documents | Low | Low | Benchmark before/after. For instruction-file sized documents (<100KB), any difference is negligible. |
| Inline markup in headings dropped during extraction | Low | High | `nodeByteRange` finds anchor positions from descendant segments, then `nodeSourceLine` expands to full line boundaries — capturing all original source bytes including inline syntax. Tests cover headings with inline code, emphasis, links, and code-span-only headings (e.g. `## \`code\``). |
| Source byte range calculation incorrect for edge cases | Medium | Medium | Add exact-text regression tests for headings and code fences. Verify `fencedBlockSourceText` against the original `extractCodeBlocks` output for the same input. Add a regression test for `nodeByteRange` returning empty range (start >= end) to verify `nodeSourceLine` returns "" safely. |

---

## Constraints

- **`internal/markdown/optimize.go` is NOT modified** — goldmark has no markdown-to-markdown renderer, so the line-by-line optimizer stays as-is. `optimize-md` interacts with validate only through `validate.Compare` for post-optimization checks.
- **URL, path, and bullet patterns stay regex-based** — goldmark's link AST node only captures `[text](url)` syntax, not bare URLs in prose. The regex patterns for these are accurate and performant.
- **The `Zone` struct and `Kind` constants are unchanged** — downstream consumers see the same types.
- **Zone text preserves exact original bytes** — source byte range extraction guarantees `zone.Text` matches what the current code produces. This is critical because `compress` writes zone text back unchanged.
- **Phase 1 must be complete before starting Phase 2** — Phase 2 depends on the `countTokens` function from Phase 1 for reduction threshold tests.

---

## Rollback Plan

If Phase 2 causes issues after merge:

1. **Quick rollback**: `git revert` the Phase 2 commit. The old regex/state-machine code is restored.
2. **Partial rollback (validate only)**: Revert `internal/validate/validate.go` and `internal/validate/ast.go`. Zones changes can remain independently.
3. **Partial rollback (zones only)**: Revert `internal/zones/zones.go`. Validate changes can remain independently.

Goldmark remains in `go.mod` only if at least one package still uses it; otherwise `go mod tidy` removes it.
