# ma

> Ma (間, lit. 'gap, space, pause') is a Japanese concept of negative space
[(Source)](https://en.wikipedia.org/wiki/Ma_(negative_space))

`ma` is a deterministic, offline Go CLI for reducing LLM context payload size without making external API calls.

## Install

### Homebrew

The packaged install path is currently **macOS-only**. Release builds are
published from the CGO-enabled macOS build used for the tree-sitter-backed
TS/JS reducers.

```bash
brew tap spw-m-riley/tools
brew install --cask spw-m-riley/tools/ma
```

### Source build

If you are outside the Homebrew/macOS path, build from source:

```bash
go build ./cmd/ma
```

## What it covers

`ma` currently provides:

| Command | Purpose |
| --- | --- |
| `ma compress <file>` | Deterministic prose compression for markdown/text-style guidance files |
| `ma validate <original> <candidate>` | Structural preservation checks for headings, code fences, URLs, paths, and bullet drift |
| `ma optimize-md <file>` | Markdown structure cleanup for blank lines, list markers, and tables |
| `ma minify-schema <file>` | JSON/YAML schema minification by removing verbose metadata |
| `ma skeleton <file>` | Code skeleton extraction for Go and parser-backed TS/JS reduction, with heuristic fallback when built without CGo |
| `ma trim-imports <file>` | Import-block summarization for Go and TS/JS-style files, with heuristic fallback for TS/JS when built without CGo |
| `ma dedup <path...>` | Exact and near-duplicate reporting across instruction-style documents |
| `ma compact-history <transcript>` | Transcript compaction for an explicit JSON message contract |
| `ma smart-read <file>` | Automatic file reduction that classifies large files and applies the matching reducer; code files use import summarization plus skeletonization when supported |
| `ma maintain <directory>` | Batch compress and deduplicate instruction-style prose trees with validation and optional writes |
| `ma dashboard` | Local-only observability companion with durable stats and recent run details |

All commands are deterministic and offline.

JSON stats keep the existing `inputApproxTokens` and `outputApproxTokens` field names, but the values are now `cl100k_base` token counts — exact for that encoding and still approximate relative to any specific model tokenizer.

## Product status

| Surface | Status | Notes |
| --- | --- | --- |
| Core reducers (`compress`, `optimize-md`, `minify-schema`, `skeleton`, `trim-imports`, `compact-history`) | Stable | Primary shipped CLI surface |
| Validation and analysis (`validate`, `dedup`, `maintain`) | Stable | Local analysis and hygiene tooling |
| `smart-read` | Stable | Reduces supported file types and explicitly passes through unsupported ones |
| `dashboard` | Experimental | Local-only companion UI for live runs, durable stats, and recent detail pages |
| Copilot extension (`.github/extensions/ma`) | Experimental | Reuses the same CLI reducers for context-aware file reads |
| Packaged distribution | macOS-only today | Homebrew cask is the supported prebuilt path; source builds remain available elsewhere |

## Smart-read support matrix

`ma smart-read` only applies reducers when the file type is one the reducer stack actually supports:

| File type | Reducer path |
| --- | --- |
| `.md`, `.txt` | Prose compression |
| `.go` | Import summarization + Go skeletonization |
| `.ts`, `.tsx`, `.js`, `.jsx` | Import summarization + TS/JS skeletonization (tree-sitter with heuristic fallback when built without CGO) |
| `.json`, `.yaml`, `.yml` | Schema minification |
| Other large files | Passed through unchanged with an explicit passthrough finding |

## Boundaries

- `ma` does **not** proxy shell commands or reduce tool output streams. Use **RTK** for that layer.
- `ma` does **not** embed an LLM or call any remote API.
- `ma` keeps the dashboard and Copilot extension local-only; neither surface is intended as a remote or multi-user service.
- For prose files, `ma compress` handles the deterministic pass. A Copilot agent may optionally do a second semantic-polish pass afterward, then re-run `ma validate`.

## Shared write contract

Commands are read-only by default.

Mutating commands (`compress`, `optimize-md`, `minify-schema`, `compact-history`) only write when `--write` is passed. On write:

1. the original file is backed up as `<path>.ma.bak`
2. transformed output is written through a temp file in the same directory
3. the final file is swapped into place

## Build

```bash
go build ./cmd/ma
go test ./...
node --test .github/extensions/ma/runtime.test.mjs
```

Go code reduction stays on the standard-library parser and formatter path. When built with `CGO_ENABLED=1`, `ma skeleton` and `ma trim-imports` use tree-sitter for `.ts`, `.tsx`, `.js`, and `.jsx`. That path requires a working C toolchain. Builds with `CGO_ENABLED=0` still work, but TS/JS processing falls back to the existing heuristic reducers.

## Examples

```bash
ma compress CLAUDE.md --json
ma compress --write CLAUDE.md && ma validate CLAUDE.md.ma.bak CLAUDE.md

ma optimize-md README.md --json
ma minify-schema schema.json --write

ma skeleton internal/service.go
ma trim-imports internal/service.go
ma trim-imports src/file.ts --json
ma smart-read internal/service.go --json
ma maintain instructions --json
ma dashboard

ma dedup .github/copilot-instructions.md instructions/*.instructions.md
ma compact-history transcript.json --json
```

## Relationship to RTK

`ma` and RTK are complementary:

- use **RTK** to reduce noisy command output before it enters the context window
- use **ma** to reduce static artifacts and transcript payloads already on disk
