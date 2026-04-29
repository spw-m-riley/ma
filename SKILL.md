# ma skill

Use `ma` when you want a deterministic, offline reduction pass over files that would otherwise consume Copilot context.

## Rules

- `ma` is offline-only. Do not describe it as using any remote API.
- Use RTK for command-output filtering. Do not route shell output through `ma`.
- Prefer `--json` when another tool or agent needs structured stats.
- Mutating commands are read-only unless `--write` is explicitly present.

## Recommended workflows

### Prose compression

1. Run `ma compress PATH --json` to inspect deterministic savings.
2. If you want the file updated, run `ma compress --write PATH`.
3. Validate the write with `ma validate PATH.ma.bak PATH`.
4. If the prose is still too wordy, the agent may apply a **manual semantic polish** after the deterministic pass.
5. Re-run `ma validate PATH.ma.bak PATH` after any manual polish.

### Markdown cleanup

1. Run `ma optimize-md PATH --json`.
2. If the structural cleanup looks right, run `ma optimize-md --write PATH`.

### Schema reduction

1. Run `ma minify-schema PATH --json`.
2. For JSON, expect removal of `description`, `examples`, and `default`.
3. For YAML, expect the supported subset only; unsupported features should fail explicitly.

### Code-context reduction

1. Use `ma skeleton PATH --json` when the goal is API shape rather than implementation detail.
2. Use `ma trim-imports PATH --json` when imports dominate a TS/JS file.

### Instruction cleanup

1. Run `ma dedup PATH... --json` to find exact and near-duplicate guidance.
2. Treat the output as a report; the command does not rewrite files.

### Transcript compaction

1. Run `ma compact-history transcript.json --json` on transcripts that already follow the explicit JSON message contract.
2. Use `ma compact-history --write transcript.json` only when you want the compacted transcript written back with a `.ma.bak` backup.

## Transcript contract

`ma compact-history` expects a JSON array of message objects like:

```json
[
  {
    "role": "assistant",
    "content": "tool output or prose",
    "toolName": "bash",
    "filePath": "optional/path",
    "timestamp": "2026-04-29T10:00:10Z"
  }
]
```

## When not to use ma

- Do not use it to proxy shell commands; use RTK.
- Do not use it when you need semantic rewriting only an LLM can perform.
- Do not use `--write` on sensitive files or credentials.
