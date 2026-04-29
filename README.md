# ma

`ma` is a deterministic, offline Go CLI for reducing LLM context payload size without calling external APIs.

This repository is under active implementation. The final command set will cover:

- prose compression
- markdown optimization
- schema minification
- code skeleton extraction
- import trimming
- instruction-file deduplication
- transcript compaction

RTK remains the separate tool-output filtering layer; `ma` does not proxy shell commands.
