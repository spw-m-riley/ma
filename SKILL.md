# ma — context reduction skill

## Proactive rules

When this skill is active, follow these rules automatically without user prompting:

### File reading strategy

- **Understanding reads**: Use `ma_smart_read` for files you are reading to understand context, explore a codebase, or gather information. The tool automatically classifies and reduces the file.
- **Edit reads**: Use standard file-reading tools (`view`, `cat`) for files you intend to edit. Accurate edit operations require the original uncompressed content.
- **Size threshold**: Skip `ma_smart_read` for files shorter than approximately 200 lines — the reduction overhead exceeds the savings at that size.

### Response style

- Produce terse, direct responses during sessions to reduce output token consumption.
- Lead with the answer or action, then provide supporting detail only if needed.
- Avoid restating the question, repeating context the user already provided, or adding boilerplate acknowledgements.
- Use bullet points and short sentences over flowing prose.
- When summarizing tool output, extract only the relevant finding rather than echoing the full result.

## Available tools

The `ma` extension provides these tools:

| Tool | Purpose | When to use |
|------|---------|-------------|
| `ma_smart_read` | Auto-classify and reduce a file | Reading files for understanding |
| `ma_compress` | Deterministic prose compression | When you need prose-specific reduction with stats |
| `ma_skeleton` | Extract declarations and signatures | When you need API shape without implementation |
| `ma_minify_schema` | Minify JSON/YAML schemas | When reading schema files for structure |
| `ma_dedup` | Detect duplicate instruction text | When auditing instruction files for redundancy |

## CLI reference

For explicit CLI control, the `ma` binary supports these workflows:

### Prose compression

1. `ma compress PATH --json` — inspect deterministic savings.
2. `ma compress --write PATH` — update the file with backup.
3. `ma validate PATH.ma.bak PATH` — verify structural preservation.

### Markdown cleanup

1. `ma optimize-md PATH --json` — inspect structural cleanup.
2. `ma optimize-md --write PATH` — apply with backup.

### Schema reduction

1. `ma minify-schema PATH --json` — minify JSON or YAML schema.

### Code-context reduction

1. `ma skeleton PATH --json` — extract API shape.
2. `ma trim-imports PATH --json` — summarize import blocks.

### Instruction cleanup

1. `ma dedup PATH... --json` — find duplicate guidance.
2. `ma maintain DIRECTORY --json` — batch compress and deduplicate instruction files.
3. `ma maintain --write DIRECTORY` — apply changes with `.ma.bak` backups.

### Transcript compaction

1. `ma compact-history transcript.json --json` — compact message history.

## Rules

- `ma` is offline-only. It never calls external APIs.
- Mutating commands are read-only unless `--write` is explicitly present.
- Use `--json` when structured stats are needed.
- Do not use `ma` to proxy shell commands or perform semantic rewriting.
