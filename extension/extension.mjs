import { execFile } from "node:child_process";
import path from "node:path";
import { joinSession } from "@github/copilot-sdk/extension";
import {
    fallbackForRead,
    findMaBinary,
    isLargeFile,
    isSensitivePathResolved,
    isTargetedRead,
    sensitivePathResponse,
} from "./runtime.mjs";

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

const MA_SESSION_CONTEXT = [
    "## File Reading Preference",
    "MA context-reduction tools are loaded. Follow these rules:",
    "- Use `ma_smart_read` instead of `view` for files ≥200 lines when reading to understand, explore, or research",
    "- Use `view` only when exact line-for-line fidelity is needed (editing, patching, line-precise inspection)",
    "- Use `view` with a `view_range` for targeted partial reads — these are not intercepted",
    "- Use `ma_skeleton` when you need API shape (declarations/signatures) before diving into code",
    "- Use `ma_minify_schema` for JSON/YAML schemas you are reading for structure, not editing",
].join("\n");

const session = await joinSession({
    tools: [
        {
            name: "ma_smart_read",
            description:
                "Use instead of `view` when reading a file (≥200 lines) to understand, explore, or " +
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
                for (const p of paths) {
                    if (isSensitivePathResolved(p)) {
                        return sensitivePathResponse(p);
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
    ],
    hooks: {
        onSessionStart: async () => {
            await session.log("ma: tools loaded, preference context injected", { ephemeral: true });
            return { additionalContext: MA_SESSION_CONTEXT };
        },

        onPreToolUse: async (input) => {
            if (input.toolName !== "view") return;

            const filePath = input.toolArgs?.path;
            if (!filePath) return;

            // Bounded partial reads pass through
            if (isTargetedRead(input.toolArgs)) return;

            // Explicit override passes through
            if (input.toolArgs?.forceReadLargeFiles) return;

            // Check file size with line counting
            if (!isLargeFile(filePath, input.cwd)) return;

            await session.log(
                `ma: intercepted view(${path.basename(filePath)}) — redirecting to ma_smart_read`,
                { ephemeral: true },
            );

            return {
                permissionDecision: "deny",
                permissionDecisionReason:
                    `This file exceeds ~200 lines. Use \`ma_smart_read\` with path="${filePath}" ` +
                    `instead — it auto-classifies and reduces the file to save context tokens. ` +
                    `If you need exact content for editing, use \`view\` with a specific \`view_range\`.`,
            };
        },

        onPostToolUse: async (input) => {
            if (input.toolName !== "view") return;

            const filePath = input.toolArgs?.path;
            if (!filePath) return;

            // Don't nudge on targeted reads
            if (isTargetedRead(input.toolArgs)) return;

            // Only nudge on large files
            if (!isLargeFile(filePath, input.cwd)) return;

            return {
                additionalContext:
                    `Note: ${path.basename(filePath)} is a large file. ` +
                    `For future reads of large files, prefer \`ma_smart_read\` — ` +
                    `it auto-reduces context while preserving structure.`,
            };
        },
    },
});
