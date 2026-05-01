import { execFile } from "node:child_process";
import { readFileSync, statSync } from "node:fs";
import { joinSession } from "@github/copilot-sdk/extension";

// Sensitive path components and basenames mirroring internal/detect
const SENSITIVE_BASENAMES = new Set([
    ".env", ".env.local", "id_rsa", "id_ed25519",
    "credentials", "known_hosts", "authorized_keys",
]);
const SENSITIVE_COMPONENTS = new Set([".ssh", ".aws", ".gnupg", ".kube"]);

function isSensitivePath(filePath) {
    const parts = filePath.split(/[\\/]/);
    const base = parts[parts.length - 1];
    if (SENSITIVE_BASENAMES.has(base)) return true;
    return parts.some((p) => SENSITIVE_COMPONENTS.has(p));
}

function findMaBinary() {
    const repoRoot = process.cwd();
    const candidates = [`${repoRoot}/ma`, `${repoRoot}/cmd/ma/ma`];
    for (const candidate of candidates) {
        try {
            statSync(candidate);
            return candidate;
        } catch {
            // continue
        }
    }
    return "ma";
}

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

function rawFallback(filePath) {
    try {
        return readFileSync(filePath, "utf-8");
    } catch (err) {
        return `Error reading file: ${err.message}`;
    }
}

const session = await joinSession({
    tools: [
        {
            name: "ma_smart_read",
            description:
                "Read a file with automatic classification and context reduction. " +
                "Classifies the file (prose/code/config) and applies the matching reduction " +
                "(compress/skeleton/minify-schema). Files under ~200 lines pass through unchanged. " +
                "Use this for reading files for understanding; use standard file tools for files you intend to edit.",
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
                if (isSensitivePath(args.path)) {
                    return {
                        textResultForLlm: `Refused: sensitive path ${args.path}`,
                        resultType: "denied",
                    };
                }
                try {
                    const output = await runMaCommand(
                        ["smart-read", args.path, "--json"],
                        { MA_SOURCE: "extension" },
                    );
                    const result = JSON.parse(output);
                    return result.output || rawFallback(args.path);
                } catch {
                    return rawFallback(args.path);
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
                if (isSensitivePath(args.path)) {
                    return {
                        textResultForLlm: `Refused: sensitive path ${args.path}`,
                        resultType: "denied",
                    };
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
                "Extract declarations and signatures from a source code file, " +
                "stripping implementation details. Returns the structural skeleton.",
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
                if (isSensitivePath(args.path)) {
                    return {
                        textResultForLlm: `Refused: sensitive path ${args.path}`,
                        resultType: "denied",
                    };
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
                "Minify a JSON or YAML schema file by removing descriptions, defaults, " +
                "and examples. Returns the minified schema.",
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
                if (isSensitivePath(args.path)) {
                    return {
                        textResultForLlm: `Refused: sensitive path ${args.path}`,
                        resultType: "denied",
                    };
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
                    if (isSensitivePath(p)) {
                        return {
                            textResultForLlm: `Refused: sensitive path ${p}`,
                            resultType: "denied",
                        };
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
            await session.log("ma extension loaded — smart-read and reduction tools available");
        },
    },
});
