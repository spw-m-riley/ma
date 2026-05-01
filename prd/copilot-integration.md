# Copilot CLI integration PRD

## Problem Statement

`ma` is a capable deterministic context-reduction tool, but today it requires a user to invoke it explicitly from the command line. A user who wants their Copilot CLI agent to benefit from reduced context payloads has to remember to run `ma compress` or `ma skeleton` by hand, inspect the output, and paste or pipe it onward. There is no way for the agent to automatically reduce the context it reads during a session, which means the tool's full value is only realized by users who remember to invoke it and know which command to use for which file type.

This makes `ma` harder to adopt, harder to benefit from consistently, and invisible during the sessions where it would help the most. A user who works with large instruction files, verbose schemas, or sprawling codebases cannot rely on `ma` to reduce their context budget without interrupting their workflow to invoke the right command manually.

## Solution

Make `ma` function as invisible context-reduction middleware inside the Copilot CLI. The tool should ship with an extension that exposes context-aware file-reading tools and a skill that tells the agent when and how to use them automatically. A user who installs `ma` should get reduced context payloads during Copilot CLI sessions without needing to ask for them, with the same deterministic, offline, and safe behavior the CLI already provides.

The integration should cover the surfaces where context accumulates during a session: files the agent reads for understanding, instruction files that load into the system prompt, and the agent's own response verbosity. File-reading reduction should be handled by the extension; instruction-file reduction should be handled by both a batch maintenance command and ad-hoc agent action; response terseness should be handled by skill instructions rather than the transformation engine.

## User Stories

1. As a Copilot CLI user, I want the agent to automatically reduce files it reads for understanding, so that my context window is not consumed by full file bodies when summaries would suffice.
2. As a Copilot CLI user, I want the agent to choose the right reduction strategy for each file type without me telling it, so that prose gets compressed, code gets skeletonized, and schemas get minified automatically.
3. As a Copilot CLI user, I want the agent to still read exact file content when it needs to make edits, so that edit operations remain accurate and reliable.
4. As a Copilot CLI user, I want small files to pass through without reduction, so that the tool does not add overhead where it is not needed.
5. As a Copilot CLI user, I want reduction to fail silently to raw content if something goes wrong, so that a classification mistake or missing binary never breaks my session.
6. As a Copilot CLI user, I want the extension and skill to ship with the `ma` binary, so that installing `ma` is enough to get the full integration without additional configuration.
7. As a Copilot CLI user, I want `ma` available as a pre-built binary, so that I can install it without needing a Go toolchain.
8. As a Copilot CLI user, I want to be able to call specific reduction tools directly when I need precise control, so that the smart default does not prevent me from choosing a particular strategy.
9. As a Copilot CLI user, I want the agent to produce terser responses during sessions, so that output tokens do not consume the context window unnecessarily.
10. As a Copilot CLI user, I want to run a batch maintenance command against my instruction files, so that the system prompt payload is reduced before sessions start.
11. As a Copilot CLI user, I want the maintenance command to both compress prose and detect duplicates across instruction files, so that I can reduce the instruction payload in one pass.
12. As a Copilot CLI user, I want reduction stats from the extension to appear in the dashboard, so that I can see how much context the invisible integration is saving over time.
13. As a Copilot CLI user, I want the extension to respect sensitive-path protections, so that the invisible integration does not create a new route for exposing protected content.
14. As a Copilot CLI user, I want the dashboard stats to include extension-mediated runs alongside normal CLI runs, so that the full picture of savings is visible in one place.
15. As a privacy-conscious user, I want the same sensitive-path rejection that protects CLI reads to apply to extension reads, so that invisible operation does not weaken existing protections.
16. As a maintainer, I want the extension to reuse the existing classification, transformation, and result contracts rather than introducing parallel pipelines, so that the integration stays consistent with the CLI.
17. As a maintainer, I want the smart-read tool to delegate to the same command implementations the CLI uses, so that reduction behavior is identical regardless of invocation path.
18. As a maintainer, I want the extension's fallback behavior to be testable, so that silent degradation to raw content can be validated rather than assumed.

## Implementation Decisions

- Introduce a Copilot CLI extension that ships in the `ma` repository and provides tools the agent can call during sessions. The extension should expose a smart-read tool that accepts a file path, classifies the file using the existing classification system, applies the appropriate reduction command, and returns reduced content with stats. If classification or reduction fails for any reason, the tool should fall back to returning raw file content without surfacing an error to the agent.
- Expose additional specific-purpose tools alongside the smart-read default, so that the agent or user can select a particular reduction strategy when the automatic classification is insufficient. The specific tools should cover prose compression, code skeletonization, schema minification, and duplicate detection.
- Rewrite the existing skill to shift from manual-invocation guidance to proactive agent rules. The skill should instruct the agent to prefer the smart-read tool for files it reads for understanding, use the standard file-reading tools for files it intends to edit, and skip reduction for files shorter than approximately 200 lines. The skill should also include terse-response instructions that reduce agent output verbosity, following the same general approach as output-reduction skills in the ecosystem.
- Introduce a batch maintenance command that walks an instruction-file tree, applies prose compression and duplicate detection, and reports aggregate savings. The maintenance command should support the same write contract as existing mutating commands, remaining read-only by default with an explicit write flag for applying changes.
- Integrate extension-mediated runs into the existing dashboard observation system so that stats from invisible reduction appear in the same history and summary views as normal CLI runs. The extension tools should produce the same structured result contract the CLI uses, making dashboard integration a matter of routing rather than translation.
- Distribute `ma` as pre-built binaries through GitHub Releases so that the extension can rely on the binary being available on the user's path without requiring a Go toolchain.
- Keep the existing boundary between `ma` and RTK: file content reduction is handled by `ma`, command output reduction is handled by RTK. The extension should not attempt to intercept or reduce shell command output.

## Testing Decisions

- Validate that the smart-read tool classifies representative file types correctly and applies the expected reduction strategy for each classification.
- Validate that the smart-read tool falls back to raw content when classification fails, when reduction produces an error, and when the binary is unavailable, without surfacing errors to the caller.
- Validate that the specific-purpose tools produce the same results as invoking the corresponding CLI commands directly.
- Validate that the extension respects sensitive-path protections and rejects the same paths the CLI rejects.
- Validate that extension-mediated runs record stats to the dashboard history store using the same contract as normal CLI runs.
- Validate that the maintenance command walks instruction trees, applies compression and duplicate detection, and reports aggregate savings accurately.
- Validate that the skill's threshold model produces the expected tool-choice behavior: smart-read for large files read for understanding, standard read for files being edited, and no reduction for files below the size threshold.
- Closest prior art in the repository comes from the existing command contract and result rendering, the file classification system, the sensitive-path rejection, the dashboard observation wrapper, and the golden-fixture test pattern used across command packages.

## Out of Scope

- Intercepting or reducing shell command output. That remains RTK's responsibility.
- Replacing the standard file-reading tools in the Copilot CLI. The extension provides additional tools; it does not override built-in behavior.
- Modifying the Copilot CLI runtime's system-prompt assembly or context-window management. The extension operates within the agent's tool-calling model.
- Real-time context-budget awareness that dynamically adjusts reduction aggressiveness based on remaining window capacity. The threshold model uses static rules.
- Multi-user, remote, or shared extension hosting. The integration is local and single-user, consistent with the rest of the product.
- LLM-powered semantic compression. All reduction remains deterministic and offline.

## Further Notes

- The repository currently has no CI pipeline or release automation, so the GitHub Releases distribution channel will need a release workflow introduced alongside or shortly after the extension and skill work.
- The extension format and packaging conventions for the Copilot CLI should be verified against current documentation before implementation begins, since the extension system may have constraints on tool naming, discovery, or binary invocation that affect the design.
- The maintenance command's tree-walking behavior should respect the same sensitive-path protections as individual file reads, and should handle mixed file types within an instruction directory gracefully.
- The terse-response instructions in the skill are prompt-engineering rather than deterministic transformation. Their effectiveness will vary by model and should be treated as best-effort rather than guaranteed.
- The existing SKILL.md will be replaced by the rewritten proactive skill. The current manual-invocation guidance should be preserved as a secondary reference or folded into the extension's tool descriptions.
