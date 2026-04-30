# Phase 3: TypeScript/JavaScript Parsing with Tree-sitter

## Overview

Phase 3 is a **conditional** recommendation. It replaces the regex-based TS/JS heuristic in `skeleton` and `trim-imports` with a proper parser, but the dependency cost is significantly higher than Phases 1 and 2. This phase should only proceed if the team decides the accuracy improvement justifies the added complexity.

| Package Option | Approach | Tradeoff |
|---------------|----------|----------|
| `smacker/go-tree-sitter` | CGo bindings to tree-sitter C library | Fast, battle-tested, requires CGo toolchain |
| `nicktomlin/go-tree-sitter` (v2) | Pure Go via WASM/wazero runtime | No CGo, but slower and adds wazero runtime (~20MB) |

**Files changed**: `internal/codectx/skeleton.go`, `internal/codectx/imports.go`, `internal/codectx/treesitter.go` (new), `internal/codectx/skeleton_ts_cgo.go` (new), `internal/codectx/skeleton_ts_nocgo.go` (new), `internal/codectx/imports_ts_cgo.go` (new), `internal/codectx/imports_ts_nocgo.go` (new), `internal/codectx/skeleton_test.go`, `internal/codectx/imports_test.go`, `go.mod`, `go.sum`, `README.md`

---

## Current State

### Skeleton: `internal/codectx/skeleton.go`

The `SkeletonHeuristic` function (lines 51–83) uses a single regex:

```go
var functionSignaturePattern = regexp.MustCompile(
    `^\s*(export\s+)?(async\s+)?function\s+\w+.*\{\s*$`,
)
```

This matches `function` keyword declarations only. It **misses**:
- Arrow functions: `const handler = async (req, res) => {`
- Class methods: `class Foo { bar() { ... } }`
- Interfaces/type declarations (which should be preserved, not skeletonized)
- Default exports: `export default function() {`
- Namespace/module patterns: `namespace Foo { function bar() { ... } }`

The heuristic is honestly labeled — `SkeletonFile` returns a warning `"heuristic skeleton used for non-Go source"` for all TS/JS files.

### Imports: `internal/codectx/imports.go`

Two regex patterns handle imports (lines 11–12):

```go
var namedImportPattern = regexp.MustCompile(
    `^import\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`,
)
var typeImportPattern = regexp.MustCompile(
    `^import\s+type\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`,
)
```

These match only single-line named imports. They **miss**:
- Multi-line imports: `import {\n  foo,\n  bar\n} from 'mod'`
- Default imports: `import React from 'react'`
- Namespace imports: `import * as path from 'path'`
- Side-effect imports: `import './polyfills'`
- Re-exports: `export { foo } from 'bar'`

---

## Decision Gate

Before implementing Phase 3, answer these questions:

1. **Is CGo acceptable in the build pipeline?**
   - If yes → `smacker/go-tree-sitter` (faster, smaller binary impact)
   - If no → pure Go WASM option or defer entirely

2. **What percentage of `ma` usage involves TS/JS files?**
   - If most usage is markdown/YAML/JSON → Phase 3 ROI is low
   - If significant TS/JS processing → Phase 3 ROI is high

3. **Is the binary size increase acceptable?**
   - CGo: +5–10 MB (tree-sitter C library + TypeScript grammar)
   - WASM: +15–25 MB (wazero runtime + WASM grammar modules)

4. **Is the heuristic warning sufficient for users?**
   - The existing tool already warns about heuristic results
   - Users know to take TS/JS skeletons with a grain of salt

**If all answers favor proceeding**, implement the plan below. Otherwise, defer Phase 3 and document the decision.

**Evidence-based precondition**: Before starting implementation, run the existing heuristic against a corpus of 10+ real-world TS/JS files from projects that use `ma`. Document which files produce incorrect or incomplete skeletons/imports. If the heuristic handles ≥90% of cases correctly, defer Phase 3 — the complexity cost outweighs the marginal improvement.

---

## Implementation Plan (CGo path: `smacker/go-tree-sitter`)

### Step 1: Add dependencies

```bash
go get github.com/smacker/go-tree-sitter@v0.0.0-20240827094217-dd6d95a4ba63
go get github.com/smacker/go-tree-sitter/typescript@v0.0.0-20240827094217-dd6d95a4ba63
go get github.com/smacker/go-tree-sitter/javascript@v0.0.0-20240827094217-dd6d95a4ba63
```

Pin to a specific commit hash — do not use `@latest`. Update only when explicitly needed.

This brings in CGo bindings. The build requires a C compiler (gcc/clang).

**Extension → Grammar mapping:**

| Extension | Grammar Package | Why |
|-----------|----------------|-----|
| `.ts` | `typescript/typescript` | Standard TypeScript syntax |
| `.tsx` | `typescript/tsx` | TypeScript + JSX elements |
| `.js` | `javascript` | Standard JavaScript syntax |
| `.jsx` | `javascript` | The tree-sitter JavaScript grammar supports JSX natively. `.jsx` is **in scope** for Phase 3 — no separate grammar needed. |

Each extension must have at least one skeleton and one imports test case. Required test functions:
- `TestSkeletonTS` (`.ts`), `TestSkeletonTSX` (`.tsx`), `TestSkeletonJS` (`.js`), `TestSkeletonJSX` (`.jsx`)
- `TestTrimImportsTS` (`.ts`), `TestTrimImportsTSX` (`.tsx`), `TestTrimImportsJS` (`.js`), `TestTrimImportsJSX` (`.jsx`)

### Step 2: Create `internal/codectx/treesitter.go`

Isolate tree-sitter code behind a build tag. Create parsers per-call to avoid global mutable state and concurrency hazards:

```go
//go:build cgo

package codectx

import (
    "fmt"

    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/javascript"
    ts "github.com/smacker/go-tree-sitter/typescript/typescript"
    tsx "github.com/smacker/go-tree-sitter/typescript/tsx"
)

// newParserForExt returns a configured tree-sitter parser for the
// given file extension. Returns an error for unsupported extensions.
func newParserForExt(ext string) (*sitter.Parser, error) {
    p := sitter.NewParser()
    switch ext {
    case ".ts":
        p.SetLanguage(ts.GetLanguage())
    case ".tsx":
        p.SetLanguage(tsx.GetLanguage())
    case ".js", ".jsx":
        p.SetLanguage(javascript.GetLanguage())
    default:
        return nil, fmt.Errorf("unsupported tree-sitter extension %q", ext)
    }
    return p, nil
}
```

No `init()` side effects. No global parser state.

### Step 3: Implement `skeletonTreeSitter`

Walk the tree-sitter CST to extract function signatures, class declarations, interfaces, and type aliases — stripping function/method bodies:

```go
//go:build cgo

func skeletonTreeSitter(ext string, src []byte) (string, []string, error) {
    parser, err := newParserForExt(ext)
    if err != nil {
        return "", nil, err
    }
    tree, err := parser.ParseCtx(context.Background(), nil, src)
    if err != nil {
        // Parse failure → fall back to heuristic with warning
        return "", []string{"tree-sitter parse failed, using heuristic fallback"}, err
    }
    root := tree.RootNode()

    // Check for ERROR nodes — partial parse
    if hasErrorNodes(root) {
        // Partial parse → fall back to heuristic with warning
        return "", []string{"tree-sitter partial parse (ERROR nodes), using heuristic fallback"}, fmt.Errorf("partial parse")
    }

    // Walk top-level nodes:
    // function_declaration → keep signature, replace body with ";"
    // class_declaration → keep class + method signatures, strip method bodies
    // interface_declaration → keep entirely (no body to strip)
    // type_alias_declaration → keep entirely
    // enum_declaration → keep entirely
    // For everything else → keep as-is
    //
    // Return: skeleton string, nil warnings (clean parse), nil error
}
```

Specific node types to handle:
- `function_declaration` → extract params + return type, drop body
- `arrow_function` (in variable_declarator) → extract params + return type, drop body
- `method_definition` (in class body) → extract params + return type, drop body
- `interface_declaration` → keep entirely
- `type_alias_declaration` → keep entirely
- `enum_declaration` → keep entirely
- `import_statement` → handled separately by imports code

### Step 4: Implement `trimImportsTreeSitter`

Walk the tree-sitter CST to extract all import forms:

```go
//go:build cgo

func trimImportsTreeSitter(ext string, src []byte) (string, []string, error) {
    parser, err := newParserForExt(ext)
    if err != nil {
        return "", nil, err
    }
    tree, err := parser.ParseCtx(context.Background(), nil, src)
    if err != nil {
        return "", []string{"tree-sitter parse failed, using heuristic fallback"}, err
    }
    root := tree.RootNode()

    if hasErrorNodes(root) {
        return "", []string{"tree-sitter partial parse, using heuristic fallback"}, fmt.Errorf("partial parse")
    }

    // Walk top-level nodes:
    // Collect all import_statement nodes
    // For each: extract module path, import names, and type
    // Produce summarized comment lines (same format as current output)
    // Return non-import source with summary header
}
```

Import node subtypes to handle:
- `import_clause` with `named_imports` → `{ foo, bar }`
- `import_clause` with `identifier` → default import
- `import_clause` with `namespace_import` → `* as name`
- `import_statement` without clause → side-effect import
- `type_import` modifier → type-only import

**Explicitly out of scope for Phase 3**: re-exports (`export { foo } from 'bar'`). These are listed in the "Current State" gaps but are a separate feature addition, not a parity replacement. Document this as a known limitation for a follow-up.

### Step 5: Create dispatcher files with build tags

**`skeleton.go` and `imports.go` stay untagged** — they remain stable dispatchers that call the appropriate implementation. The TS/JS-specific tree-sitter code lives in paired build-tagged files.

**`internal/codectx/skeleton_ts_cgo.go`:**
```go
//go:build cgo

package codectx

// tsjsSkeleton dispatches to tree-sitter for TS/JS skeleton extraction.
// On parse failure or ERROR nodes, returns a non-nil error to signal
// the caller to fall back to the heuristic.
func tsjsSkeleton(ext string, src []byte) (string, []string, error) {
    return skeletonTreeSitter(ext, src)
}
```

**`internal/codectx/skeleton_ts_nocgo.go`:**
```go
//go:build !cgo

package codectx

import "fmt"

// tsjsSkeleton is the no-CGo stub. Always returns an error to trigger
// heuristic fallback. This preserves the existing behavior for pure-Go builds.
func tsjsSkeleton(ext string, src []byte) (string, []string, error) {
    return "", nil, fmt.Errorf("tree-sitter unavailable (CGo disabled)")
}
```

**`internal/codectx/imports_ts_cgo.go`:**
```go
//go:build cgo

package codectx

func tsjsTrimImports(ext string, src []byte) (string, []string, error) {
    return trimImportsTreeSitter(ext, src)
}
```

**`internal/codectx/imports_ts_nocgo.go`:**
```go
//go:build !cgo

package codectx

import "fmt"

func tsjsTrimImports(ext string, src []byte) (string, []string, error) {
    return "", nil, fmt.Errorf("tree-sitter unavailable (CGo disabled)")
}
```

**Updated `SkeletonFile` in `skeleton.go` (no build tag):**
```go
func SkeletonFile(path string, src []byte) (string, []string, error) {
    switch filepath.Ext(path) {
    case ".go":
        output, err := SkeletonGo(src)
        return output, nil, err
    case ".ts", ".tsx", ".js", ".jsx":
        ext := filepath.Ext(path)
        output, warnings, err := tsjsSkeleton(ext, src)
        if err != nil {
            // Tree-sitter unavailable or failed → heuristic fallback.
            // SkeletonHeuristic takes string, returns string (no warnings).
            // Caller appends the fixed heuristic warning.
            heuristicOutput := SkeletonHeuristic(string(src))
            warnings = append(warnings, "heuristic skeleton used for non-Go source")
            return heuristicOutput, warnings, nil
        }
        return output, warnings, nil
    default:
        return "", nil, fmt.Errorf("unsupported skeleton extension %q", filepath.Ext(path))
    }
}
```

**Updated `TrimImportsFile` in `imports.go` (no build tag):**
```go
func TrimImportsFile(path string, src []byte) (string, []string, error) {
    switch filepath.Ext(path) {
    case ".ts", ".tsx", ".js", ".jsx":
        ext := filepath.Ext(path)
        output, warnings, err := tsjsTrimImports(ext, src)
        if err != nil {
            // Tree-sitter unavailable or failed → regex heuristic fallback.
            // trimJSImportBlock takes string, returns string (no warnings).
            heuristicOutput := trimJSImportBlock(string(src))
            return heuristicOutput, warnings, nil
        }
        return output, warnings, nil
    default:
        return "", nil, fmt.Errorf("unsupported import trimming extension %q", filepath.Ext(path))
    }
}
```

**Heuristic helper signatures (unchanged):**
- `SkeletonHeuristic(input string) string` — stays as-is, no refactor needed
- `trimJSImportBlock(input string) string` — stays as-is, no refactor needed
- Callers append the fixed warning string themselves when falling back

**Parse failure contract:**
- `tsjsSkeleton`/`tsjsTrimImports` return a non-nil error → caller falls back to heuristic
- Warnings from tree-sitter (partial parse, fallback) are appended to the warnings slice and propagated to the user
- When tree-sitter succeeds cleanly, the heuristic warning (`"heuristic skeleton used for non-Go source"`) is **not** emitted
- When tree-sitter fails and heuristic is used, the original heuristic warning **is** emitted plus any tree-sitter diagnostic warnings
- Grammar init failure (unsupported extension) → same fallback path

Note: The heuristic warning is removed for clean tree-sitter output because it's no longer a heuristic. The warning is preserved for the fallback path.

### Step 6: Verify heuristic fallback

The existing `SkeletonHeuristic` and regex imports functions remain in their current files, unchanged. They are called as the fallback path when:
- CGo is disabled (`//go:build !cgo` stubs return errors)
- Tree-sitter parse fails (ERROR nodes, invalid input)
- Grammar init fails (unsupported extension reaching tree-sitter)

**Verification commands:**
```bash
# Build and test with CGo (tree-sitter active)
CGO_ENABLED=1 go build ./cmd/ma
CGO_ENABLED=1 go test -v ./internal/codectx/...

# Build and test without CGo (heuristic fallback)
CGO_ENABLED=0 go build ./cmd/ma
CGO_ENABLED=0 go test -v ./internal/codectx/...

# Cross-compile check (pure-Go, no C compiler)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/ma
```

---

## Alternative: Pure Go WASM Path (Deferred)

If CGo is rejected at the decision gate, **defer Phase 3 entirely** rather than implementing the WASM path. The WASM alternative (`nicktomlin/go-tree-sitter` v2) exists but adds significant binary size (~20MB wazero runtime) and is not worth the tradeoff for a lightweight CLI tool. Document the deferral decision and revisit if CGo becomes acceptable or a better pure-Go parser emerges.

The WASM path details are preserved below for reference only — they are **not implementable** without full expansion to CGo-path detail.

### Differences from CGo path

> **Note**: The WASM path is documented here for completeness. If CGo is rejected and the WASM path is chosen, expand this section to the same level of detail as the CGo path before implementation. As written, this section is **not implementable** — it is a decision reference only.

1. **No build tags needed** — pure Go, works everywhere
2. **Grammar loaded at runtime** from embedded WASM modules
3. **Significantly slower** — WASM interpretation overhead
4. **Larger binary** — wazero runtime adds ~15–20 MB
5. **API differences** — node types and query syntax differ slightly

### Step 1: Add dependency

```bash
go get github.com/nicktomlin/go-tree-sitter@v2  # Pin to specific release when chosen
```

### Steps 2–6: Same structure as CGo path

The tree-walking logic is the same; only the parser initialization and node access API differ. **Expand this section to full detail before implementation.**

---

## Acceptance Criteria

All criteria must be met before Phase 3 is considered complete.

### Decision Gate

- [ ] CGo vs pure Go decision documented with rationale
- [ ] Binary size impact measured and accepted: `ls -la $(CGO_ENABLED=1 go build -o /tmp/ma-cgo ./cmd/ma && echo /tmp/ma-cgo)` vs `ls -la $(CGO_ENABLED=0 go build -o /tmp/ma-nocgo ./cmd/ma && echo /tmp/ma-nocgo)`
- [ ] TS/JS usage frequency assessed (is this worth the complexity?)
- [ ] Evidence-based corpus test: run existing heuristic against 10+ real TS/JS files and document accuracy

### Functional

- [ ] `CGO_ENABLED=1 go test ./...` passes with zero failures
- [ ] `CGO_ENABLED=0 go test ./...` passes with zero failures (heuristic fallback)
- [ ] `CGO_ENABLED=1 go build ./cmd/ma` succeeds
- [ ] `CGO_ENABLED=0 go build ./cmd/ma` succeeds
- [ ] `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/ma` cross-compiles cleanly
- [ ] Go skeleton output is **identical** to current output (Go path unchanged)
- [ ] TS/JS skeleton extracts (with tree-sitter, tested per extension: `.ts`, `.tsx`, `.js`, `.jsx`):
  - [ ] `function` declarations (existing behavior preserved)
  - [ ] Arrow functions assigned to `const`/`let`/`var`
  - [ ] Class declarations with method signatures
  - [ ] Interface declarations (preserved entirely)
  - [ ] Type aliases (preserved entirely)
  - [ ] Enum declarations (preserved entirely)
- [ ] TS/JS import trimming handles (with tree-sitter):
  - [ ] Named imports `{ foo, bar }` (existing behavior preserved)
  - [ ] Type imports `import type { ... }` (existing behavior preserved)
  - [ ] Default imports `import React from 'react'`
  - [ ] Namespace imports `import * as path from 'path'`
  - [ ] Side-effect imports `import './polyfills'`
  - [ ] Multi-line imports
- [ ] Re-exports (`export { foo } from 'bar'`) are explicitly **out of scope** — documented as known limitation
- [ ] Heuristic warning removed when tree-sitter succeeds cleanly
- [ ] Heuristic warning preserved when tree-sitter fails and heuristic fallback is used
- [ ] Tree-sitter diagnostic warnings propagated to user on partial parse / fallback
- [ ] Heuristic fallback works when CGo is disabled (nocgo stubs return error → heuristic path)

### Non-Functional

- [ ] Binary size increase documented (CGo vs nocgo comparison)
- [ ] Benchmark results compared before/after:
  ```bash
  # Before Phase 3 (baseline):
  go test ./internal/codectx -run '^$' -bench 'Benchmark(Skeleton|TrimImports)' -benchmem > bench-before.txt

  # After Phase 3:
  CGO_ENABLED=1 go test ./internal/codectx -run '^$' -bench 'Benchmark(Skeleton|TrimImports)' -benchmem > bench-after-cgo.txt
  CGO_ENABLED=0 go test ./internal/codectx -run '^$' -bench 'Benchmark(Skeleton|TrimImports)' -benchmem > bench-after-nocgo.txt
  ```
- [ ] Parse time for a 1000-line TS file < 100ms
- [ ] No new network calls introduced
- [ ] All commands remain deterministic

### Testing

- [ ] Existing Go skeleton test passes unchanged: `go test -v -run 'TestSkeletonGo' ./internal/codectx/...`
- [ ] Existing TS heuristic skeleton test: **rewrite expected output** to match tree-sitter's improved output when CGo is enabled. The current test expects a `"heuristic skeleton used for non-Go source"` warning — under CGo builds, this warning should NOT appear. Under nocgo builds, this warning must still appear. Create two test variants if needed:
  ```bash
  CGO_ENABLED=1 go test -v -run 'TestSkeletonTS' ./internal/codectx/...  # no heuristic warning
  CGO_ENABLED=0 go test -v -run 'TestSkeletonTS' ./internal/codectx/...  # heuristic warning present
  ```
- [ ] New test: arrow function skeleton extraction (`go test -v -run 'TestSkeletonArrow' ./internal/codectx/...`)
- [ ] New test: class method skeleton extraction (`go test -v -run 'TestSkeletonClass' ./internal/codectx/...`)
- [ ] New test: interface and type alias preservation (`go test -v -run 'TestSkeletonInterfaceTypeAlias' ./internal/codectx/...`)
- [ ] New test: default import trimming (`go test -v -run 'TestTrimImportsDefault' ./internal/codectx/...`)
- [ ] New test: namespace import trimming (`go test -v -run 'TestTrimImportsNamespace' ./internal/codectx/...`)
- [ ] New test: side-effect import handling (`go test -v -run 'TestTrimImportsSideEffect' ./internal/codectx/...`)
- [ ] New test: multi-line import trimming (`go test -v -run 'TestTrimImportsMultiline' ./internal/codectx/...`)
- [ ] New test: mixed import styles in single file
- [ ] New test: parse failure fallback — invalid TS input triggers heuristic with warning (`go test -v -run 'TestSkeletonParseFallback' ./internal/codectx/...`)
- [ ] New test: per-extension coverage with named test functions:
  - `TestSkeletonTS`, `TestSkeletonTSX`, `TestSkeletonJS`, `TestSkeletonJSX`
  - `TestTrimImportsTS`, `TestTrimImportsTSX`, `TestTrimImportsJS`, `TestTrimImportsJSX`
  - Run: `CGO_ENABLED=1 go test -v -run 'TestSkeleton(TS|TSX|JS|JSX)$|TestTrimImports(TS|TSX|JS|JSX)$' ./internal/codectx/...`
- [ ] Existing reduction threshold tests pass (thresholds may improve — document)
- [ ] All benchmarks run and results documented

### Documentation

- [ ] `go.mod` shows new direct dependency
- [ ] `README.md` updated: remove "heuristic" language for TS/JS if tree-sitter is used
- [ ] Build requirements documented (CGo + C compiler if CGo path)
- [ ] Fallback behavior documented if CGo path with `!cgo` build tag

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| CGo breaks cross-compilation | High | High | Use build tags with paired cgo/nocgo files. Test `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build`. Dispatchers stay untagged. |
| Tree-sitter grammar version drift | Medium | Medium | Pin grammar version in go.mod (explicit commit hash, not @latest). Add golden test that verifies parse output for known input. |
| WASM path is too slow for large files | Medium | Medium | Benchmark with realistic file sizes (100–5000 lines). Set timeout on parse. |
| Binary size increase is unacceptable | Medium | High | Measure before committing. If too large, defer Phase 3 entirely. |
| Parser fails on non-standard TS syntax (decorators, JSX) | Medium | Medium | Test with real-world TS/TSX files across all 4 extensions. Fall back to heuristic on parse error with diagnostic warning. |
| Node type mapping is incomplete | Medium | Medium | Start with the 6 core node types. Add more as edge cases are found in testing. |
| Parse failure produces silent regression | Medium | High | Parse failure contract: tree-sitter errors → return error → caller falls back to heuristic with warnings. Tests cover invalid input, ERROR nodes, and grammar init failure. |

---

## Constraints

- **Phase 1 and Phase 2 must be complete before Phase 3** — earlier phases establish the dependency management pattern and testing approach.
- **This phase is conditional** — it only proceeds if the decision gate criteria are met.
- **Go skeleton path is never modified** — `SkeletonGo` uses the Go AST and is already production-quality.
- **Heuristic fallback must always be available** — even if tree-sitter is added, the tool must build and run without CGo (if CGo path chosen).
- **Import output format is preserved** — `// imports:` and `// types:` comment syntax stays the same, with expanded coverage for additional import forms.

---

## Rollback Plan

If Phase 3 causes issues after merge:

1. **Quick rollback**: Revert the go.mod/go.sum changes and the new files. The heuristic fallback (`skeleton_nocgo.go` or original `skeleton.go`) becomes the only path again.
2. **Partial rollback**: Keep tree-sitter for skeleton but revert import changes (or vice versa).
3. **Build tag isolation**: If CGo path, users can build with `CGO_ENABLED=0` to get the heuristic fallback without removing any code.
