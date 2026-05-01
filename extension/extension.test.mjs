import test from "node:test";
import assert from "node:assert/strict";

import {
    createSessionConfig,
    MA_READ_INTENT_CONTEXT,
    MA_SESSION_CONTEXT,
    READ_INTENT_PATTERNS,
} from "./session-config.mjs";

function getTools(config) {
    return Object.fromEntries(config.tools.map((tool) => [tool.name, tool]));
}

function makeConfig(overrides = {}) {
    return createSessionConfig({
        log: async () => {},
        isLargeFile: () => false,
        isTargetedRead: () => false,
        ...overrides,
    });
}

test("tool descriptions lead with MA-vs-view decision boundaries", () => {
    const tools = getTools(makeConfig());

    assert.match(tools.ma_smart_read.description, /^Use instead of `view`/);
    assert.match(tools.ma_smart_read.description, /≥\s*200 lines/);

    assert.match(tools.ma_skeleton.description, /^Use instead of `view` or `ma_smart_read`/);
    assert.match(tools.ma_skeleton.description, /API surface/);

    assert.match(tools.ma_minify_schema.description, /^Use instead of `view`/);
    assert.match(tools.ma_minify_schema.description, /JSON\/YAML schema/);

    assert.equal(
        tools.ma_compress.description,
        "Apply deterministic prose compression to a natural language file. " +
            "Returns compressed text with stats. Does not modify the file.",
    );
    assert.equal(
        tools.ma_dedup.description,
        "Detect exact and near-duplicate sentences across one or more instruction files. " +
            "Returns a report of duplicates found. Does not modify files.",
    );
});

test("onSessionStart injects the MA session preference rule", async () => {
    const { hooks } = makeConfig();
    const result = await hooks.onSessionStart();

    assert.equal(result.additionalContext, MA_SESSION_CONTEXT);
    assert.match(result.additionalContext, /ma_smart_read/);
    assert.match(result.additionalContext, /ma_skeleton/);
    assert.match(result.additionalContext, /ma_minify_schema/);
    assert.match(result.additionalContext, /view` only/i);
});

test("onPreToolUse denies full-file view reads on large files", async () => {
    const logs = [];
    const { hooks } = makeConfig({
        log: async (message) => logs.push(message),
        isLargeFile: () => true,
    });

    const result = await hooks.onPreToolUse({
        toolName: "view",
        toolArgs: { path: "/tmp/large.md" },
        cwd: "/tmp",
    });

    assert.equal(result.permissionDecision, "deny");
    assert.match(result.permissionDecisionReason, /ma_smart_read/);
    assert.match(result.permissionDecisionReason, /view_range/);
    assert.equal(logs.length, 1);
});

test("onPreToolUse denies full-file view reads when the runtime sends stringified toolArgs", async () => {
    const { hooks } = makeConfig({
        isLargeFile: () => true,
    });

    const result = await hooks.onPreToolUse({
        toolName: "view",
        toolArgs: JSON.stringify({ path: "/tmp/large.md" }),
        cwd: "/tmp",
    });

    assert.equal(result.permissionDecision, "deny");
});

test("onPreToolUse passes through bounded targeted reads", async () => {
    const { hooks } = makeConfig({
        isLargeFile: () => true,
        isTargetedRead: () => true,
    });

    const result = await hooks.onPreToolUse({
        toolName: "view",
        toolArgs: { path: "/tmp/large.md", view_range: [1, 20] },
        cwd: "/tmp",
    });

    assert.equal(result, undefined);
});

test("onPreToolUse treats [n, -1] reads as full-file reads", async () => {
    const { hooks } = makeConfig({
        isLargeFile: () => true,
        isTargetedRead: () => false,
    });

    const result = await hooks.onPreToolUse({
        toolName: "view",
        toolArgs: { path: "/tmp/large.md", view_range: [20, -1] },
        cwd: "/tmp",
    });

    assert.equal(result.permissionDecision, "deny");
});

test("onPreToolUse passes through forceReadLargeFiles overrides", async () => {
    const { hooks } = makeConfig({
        isLargeFile: () => true,
    });

    const result = await hooks.onPreToolUse({
        toolName: "view",
        toolArgs: { path: "/tmp/large.md", forceReadLargeFiles: true },
        cwd: "/tmp",
    });

    assert.equal(result, undefined);
});

test("onPreToolUse fails open for non-view tools and detector errors", async () => {
    const passThrough = makeConfig({ isLargeFile: () => true }).hooks;
    assert.equal(
        await passThrough.onPreToolUse({
            toolName: "ma_smart_read",
            toolArgs: { path: "/tmp/large.md" },
            cwd: "/tmp",
        }),
        undefined,
    );

    const failOpen = makeConfig({
        isLargeFile: () => {
            throw new Error("boom");
        },
    }).hooks;
    assert.equal(
        await failOpen.onPreToolUse({
            toolName: "view",
            toolArgs: { path: "/tmp/large.md" },
            cwd: "/tmp",
        }),
        undefined,
    );
});

test("onPostToolUse nudges after large full-file view reads", async () => {
    const { hooks } = makeConfig({
        isLargeFile: () => true,
    });

    const result = await hooks.onPostToolUse({
        toolName: "view",
        toolArgs: { path: "/tmp/large.md" },
        cwd: "/tmp",
    });

    assert.match(result.additionalContext, /ma_smart_read/);
    assert.match(result.additionalContext, /large file/i);
});

test("onPostToolUse stays silent for targeted reads, small files, and detector errors", async () => {
    const targeted = makeConfig({
        isLargeFile: () => true,
        isTargetedRead: () => true,
    }).hooks;
    assert.equal(
        await targeted.onPostToolUse({
            toolName: "view",
            toolArgs: { path: "/tmp/large.md", view_range: [1, 20] },
            cwd: "/tmp",
        }),
        undefined,
    );

    const small = makeConfig({ isLargeFile: () => false }).hooks;
    assert.equal(
        await small.onPostToolUse({
            toolName: "view",
            toolArgs: { path: "/tmp/small.md" },
            cwd: "/tmp",
        }),
        undefined,
    );

    const failOpen = makeConfig({
        isLargeFile: () => {
            throw new Error("boom");
        },
    }).hooks;
    assert.equal(
        await failOpen.onPostToolUse({
            toolName: "view",
            toolArgs: { path: "/tmp/large.md" },
            cwd: "/tmp",
        }),
        undefined,
    );
});

test("onUserPromptSubmitted injects a brief MA reminder for read-intent prompts", async () => {
    const { hooks } = makeConfig();
    const result = await hooks.onUserPromptSubmitted("show me how the view intercept works");

    assert.equal(result.additionalContext, MA_READ_INTENT_CONTEXT);
    assert.match(result.additionalContext, /ma_smart_read/);
});

test("onUserPromptSubmitted ignores edit and execution prompts", async () => {
    const { hooks } = makeConfig();

    assert.equal(await hooks.onUserPromptSubmitted("fix the runtime helper bug"), undefined);
    assert.equal(await hooks.onUserPromptSubmitted("run the extension tests"), undefined);
});

test("onUserPromptSubmitted deduplicates consecutive read-intent nudges and resets after context shifts", async () => {
    const { hooks } = makeConfig();

    assert.equal(
        (await hooks.onUserPromptSubmitted("read extension/runtime.mjs")).additionalContext,
        MA_READ_INTENT_CONTEXT,
    );
    assert.equal(await hooks.onUserPromptSubmitted("look at extension/extension.mjs"), undefined);
    assert.equal(await hooks.onUserPromptSubmitted("run the runtime tests"), undefined);
    assert.equal(
        (await hooks.onUserPromptSubmitted("understand the session-start hook")).additionalContext,
        MA_READ_INTENT_CONTEXT,
    );
});

test("onSubagentStart reuses the MA session preference context", async () => {
    const { hooks } = makeConfig();
    await hooks.onUserPromptSubmitted("show me the extension config");

    const result = await hooks.onSubagentStart({ agentType: "explore" });

    assert.equal(result.additionalContext, MA_SESSION_CONTEXT);
});

test("read intent keywords stay documented in a shared constant", () => {
    assert.ok(Array.isArray(READ_INTENT_PATTERNS));
    assert.ok(READ_INTENT_PATTERNS.length > 0);
    for (const pattern of READ_INTENT_PATTERNS) {
        assert.ok(pattern instanceof RegExp);
    }
});
