import { execFile } from "node:child_process";
import path from "node:path";

import {
    fallbackForRead,
    findMaBinary,
    isLargeFile as defaultIsLargeFile,
    isSensitivePathResolved,
    isTargetedRead as defaultIsTargetedRead,
    sensitivePathResponse,
} from "./runtime.mjs";

export const MA_SESSION_CONTEXT = [
    "## File Reading Preference",
    "MA context-reduction tools are loaded. Follow these rules:",
    "- Use `ma_smart_read` instead of `view` for files ≥ 200 lines when reading to understand, explore, or research",
    "- Use `view` only when exact line-for-line fidelity is needed (editing, patching, line-precise inspection)",
    "- Use `view` with a `view_range` for targeted partial reads — these are not intercepted",
    "- Use `ma_skeleton` when you need API shape (declarations/signatures) before diving into code",
    "- Use `ma_minify_schema` for JSON/YAML schemas you are reading for structure, not editing",
].join("\n");

export const MA_READ_INTENT_CONTEXT =
    "Read-intent detected: prefer `ma_smart_read` over full-file `view` for understanding reads on large files. " +
    "Keep `view` for edit prep or exact `view_range` inspection.";

// Keep this keyword list tight so the reminder complements session-start guidance
// without firing on prompts that are clearly about editing or command execution.
export const READ_INTENT_PATTERNS = Object.freeze([
    /\bread\b/i,
    /\blook at\b/i,
    /\bshow me\b/i,
    /\bexplore\b/i,
    /\bunderstand\b/i,
    /\bwhat does\b/i,
]);

const NON_READ_INTENT_PATTERNS = Object.freeze([
    /\b(edit|modify|change|update|implement|fix|patch|refactor|rename)\b/i,
    /\b(run|test|build|lint|debug|profile|benchmark)\b/i,
]);

function runMaCommand(args, env) {
    return new Promise((resolve, reject) => {
        const binary = findMaBinary();
        const childEnv = { ...process.env, ...env };
        execFile(binary, args, { env: childEnv, timeout: 30000 }, (err, stdout, stderr) => {
            if (err) {
                reject(new Error(stderr || err.message));
            } else {
                resolve(stdout);
            }
        });
    });
}

async function safeLog(log, message) {
    try {
        await log(message, { ephemeral: true });
    } catch {
        // Logging should never block file reads or hook context injection.
    }
}

function normalizePromptInput(input) {
    if (typeof input === "string") {
        return input.trim();
    }

    if (typeof input?.prompt === "string") {
        return input.prompt.trim();
    }

    if (typeof input?.message === "string") {
        return input.message.trim();
    }

    if (typeof input?.input === "string") {
        return input.input.trim();
    }

    return "";
}

function normalizeToolArgs(toolArgs) {
    if (typeof toolArgs === "string") {
        try {
            return JSON.parse(toolArgs);
        } catch {
            return undefined;
        }
    }

    if (toolArgs && typeof toolArgs === "object") {
        return toolArgs;
    }

    return undefined;
}

export function hasReadIntent(input) {
    const prompt = normalizePromptInput(input).replace(/\s+/g, " ").trim();
    if (!prompt) {
        return false;
    }

    if (NON_READ_INTENT_PATTERNS.some((pattern) => pattern.test(prompt))) {
        return false;
    }

    return READ_INTENT_PATTERNS.some((pattern) => pattern.test(prompt));
}

function createTools() {
    return [
        {
            name: "ma_smart_read",
            description:
                "Use instead of `view` when reading a file (≥ 200 lines) to understand, explore, or " +
                "research — not when editing or patching. Auto-classifies (prose/code/config) and " +
                "applies the best reducer (compress/skeleton/minify-schema) to save context tokens. " +
                "Files under ~200 lines pass through unchanged.",
            parameters: {
                type: "object",
                properties: {
                    path: {
                        type: "string",
                        description: "Absolute or relative path to the file to read",
                    },
                },
                required: ["path"],
            },
            handler: async (args) => {
                if (!args?.path) {
                    return { textResultForLlm: "Error: path argument is required", resultType: "failure" };
                }
                if (isSensitivePathResolved(args.path)) {
                    return sensitivePathResponse(args.path);
                }
                try {
                    const output = await runMaCommand(
                        ["smart-read", args.path, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || fallbackForRead(args.path, new Error("empty smart-read output"));
                } catch (err) {
                    return fallbackForRead(args.path, err);
                }
            },
        },
        {
            name: "ma_compress",
            description:
                "Apply deterministic prose compression to a natural language file. " +
                "Returns compressed text with stats. Does not modify the file.",
            parameters: {
                type: "object",
                properties: {
                    path: {
                        type: "string",
                        description: "Path to the file to compress",
                    },
                },
                required: ["path"],
            },
            handler: async (args) => {
                if (!args?.path) {
                    return { textResultForLlm: "Error: path argument is required", resultType: "failure" };
                }
                if (isSensitivePathResolved(args.path)) {
                    return sensitivePathResponse(args.path);
                }
                try {
                    const output = await runMaCommand(
                        ["compress", args.path, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || output;
                } catch (err) {
                    return { textResultForLlm: err.message, resultType: "failure" };
                }
            },
        },
        {
            name: "ma_skeleton",
            description:
                "Use instead of `view` or `ma_smart_read` when you need a source file's API " +
                "surface (declarations, signatures) without implementation bodies. Fastest way " +
                "to understand what a Go/TS/JS module exports before deciding where to look deeper.",
            parameters: {
                type: "object",
                properties: {
                    path: {
                        type: "string",
                        description: "Path to the source file",
                    },
                },
                required: ["path"],
            },
            handler: async (args) => {
                if (!args?.path) {
                    return { textResultForLlm: "Error: path argument is required", resultType: "failure" };
                }
                if (isSensitivePathResolved(args.path)) {
                    return sensitivePathResponse(args.path);
                }
                try {
                    const output = await runMaCommand(
                        ["skeleton", args.path, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || output;
                } catch (err) {
                    return { textResultForLlm: err.message, resultType: "failure" };
                }
            },
        },
        {
            name: "ma_minify_schema",
            description:
                "Use instead of `view` for JSON/YAML schema files you are reading to " +
                "understand structure — removes verbose metadata (descriptions, defaults, " +
                "examples) to show the type skeleton only.",
            parameters: {
                type: "object",
                properties: {
                    path: {
                        type: "string",
                        description: "Path to the JSON or YAML schema file",
                    },
                },
                required: ["path"],
            },
            handler: async (args) => {
                if (!args?.path) {
                    return { textResultForLlm: "Error: path argument is required", resultType: "failure" };
                }
                if (isSensitivePathResolved(args.path)) {
                    return sensitivePathResponse(args.path);
                }
                try {
                    const output = await runMaCommand(
                        ["minify-schema", args.path, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || output;
                } catch (err) {
                    return { textResultForLlm: err.message, resultType: "failure" };
                }
            },
        },
        {
            name: "ma_dedup",
            description:
                "Detect exact and near-duplicate sentences across one or more instruction files. " +
                "Returns a report of duplicates found. Does not modify files.",
            parameters: {
                type: "object",
                properties: {
                    paths: {
                        type: "array",
                        items: { type: "string" },
                        description: "Paths to instruction files to analyze for duplicates",
                    },
                },
                required: ["paths"],
            },
            handler: async (args) => {
                const paths = args.paths || [];
                for (const currentPath of paths) {
                    if (isSensitivePathResolved(currentPath)) {
                        return sensitivePathResponse(currentPath);
                    }
                }
                try {
                    const output = await runMaCommand(
                        ["dedup", ...paths, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || output;
                } catch (err) {
                    return { textResultForLlm: err.message, resultType: "failure" };
                }
            },
        },
    ];
}

export function createSessionConfig({
    log = async () => {},
    isLargeFile = defaultIsLargeFile,
    isTargetedRead = defaultIsTargetedRead,
    sessionContext = MA_SESSION_CONTEXT,
    readIntentContext = MA_READ_INTENT_CONTEXT,
} = {}) {
    const state = {
        lastPromptWasReadIntent: false,
    };

    return {
        tools: createTools(),
        hooks: {
            onSessionStart: async () => {
                await safeLog(log, "ma: tools loaded, preference context injected");
                return { additionalContext: sessionContext };
            },

            onPreToolUse: async (input) => {
                try {
                    if (input.toolName !== "view") return;

                    const toolArgs = normalizeToolArgs(input.toolArgs);
                    const filePath = toolArgs?.path;
                    if (!filePath) return;

                    if (isTargetedRead(toolArgs)) return;
                    if (toolArgs?.forceReadLargeFiles) return;
                    if (!isLargeFile(filePath, input.cwd)) return;

                    await safeLog(
                        log,
                        `ma: intercepted view(${path.basename(filePath)}) — redirecting to ma_smart_read`,
                    );

                    return {
                        permissionDecision: "deny",
                        permissionDecisionReason:
                            `This file exceeds ~200 lines. Use \`ma_smart_read\` with path="${filePath}" ` +
                            `instead — it auto-classifies and reduces the file to save context tokens. ` +
                            `If you need exact content for editing, use \`view\` with a specific \`view_range\`.`,
                    };
                } catch {
                    return undefined;
                }
            },

            onPostToolUse: async (input) => {
                try {
                    if (input.toolName !== "view") return;

                    const filePath = input.toolArgs?.path;
                    if (!filePath) return;

                    if (isTargetedRead(input.toolArgs)) return;
                    if (!isLargeFile(filePath, input.cwd)) return;

                    return {
                        additionalContext:
                            `Note: ${path.basename(filePath)} is a large file. ` +
                            `For future reads of large files, prefer \`ma_smart_read\` — ` +
                            `it auto-reduces context while preserving structure.`,
                    };
                } catch {
                    return undefined;
                }
            },

            onUserPromptSubmitted: async (input) => {
                try {
                    if (!hasReadIntent(input)) {
                        state.lastPromptWasReadIntent = false;
                        return undefined;
                    }

                    if (state.lastPromptWasReadIntent) {
                        return undefined;
                    }

                    state.lastPromptWasReadIntent = true;
                    return { additionalContext: readIntentContext };
                } catch {
                    return undefined;
                }
            },

            // Forward-compatible handler: the shipped Copilot CLI 1.0.40 SDK does
            // not yet type or invoke `onSubagentStart`, but keeping the handler
            // here lets newer runtimes reuse the same session preference context.
            onSubagentStart: async () => {
                try {
                    return { additionalContext: sessionContext };
                } catch {
                    return undefined;
                }
            },
        },
    };
}
