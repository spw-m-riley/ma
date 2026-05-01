import test from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
    fallbackForRead,
    isLargeFile,
    isSensitivePathResolved,
    isTargetedRead,
    VIEW_LINE_THRESHOLD,
    VIEW_SIZE_GATE_BYTES,
} from "./runtime.mjs";

// --- Sensitive path tests ---

test("isSensitivePathResolved blocks symlinks into protected locations", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const sshDir = join(root, ".ssh");
    const secretPath = join(sshDir, "fake_secret");
    const visiblePath = join(root, "visible.txt");

    mkdirSync(sshDir);
    writeFileSync(secretPath, "synthetic secret\n");
    symlinkSync(secretPath, visiblePath);

    assert.equal(isSensitivePathResolved(visiblePath), true);
});

test("fallbackForRead denies when the ma command refused a sensitive path", () => {
    const response = fallbackForRead(
        "/tmp/visible.txt",
        new Error('refusing sensitive path "/tmp/visible.txt"'),
    );

    assert.deepEqual(response, {
        textResultForLlm: "Refused: sensitive path /tmp/visible.txt",
        resultType: "denied",
    });
});

test("fallbackForRead only raw-falls back for safe paths", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const filePath = join(root, "notes.md");
    writeFileSync(filePath, "safe fallback\n");

    const response = fallbackForRead(filePath, new Error("ma unavailable"));

    assert.equal(response, "safe fallback\n");
});

// --- isTargetedRead tests ---

test("runtime exports the shared view thresholds", () => {
    assert.equal(VIEW_LINE_THRESHOLD, 200);
    assert.equal(VIEW_SIZE_GATE_BYTES, 10 * 1024);
});

test("isTargetedRead returns false when no view_range", () => {
    assert.equal(isTargetedRead({}), false);
    assert.equal(isTargetedRead({ path: "foo.ts" }), false);
    assert.equal(isTargetedRead(undefined), false);
});

test("isTargetedRead returns true for bounded partial reads under 200 lines", () => {
    assert.equal(isTargetedRead({ view_range: [1, 30] }), true);
    assert.equal(isTargetedRead({ view_range: [50, 100] }), true);
    assert.equal(isTargetedRead({ view_range: [100, 199] }), true);
});

test("isTargetedRead returns false for ranges >= 200 lines", () => {
    assert.equal(isTargetedRead({ view_range: [1, 200] }), false);
    assert.equal(isTargetedRead({ view_range: [1, 500] }), false);
});

test("isTargetedRead returns false for EOF marker (-1)", () => {
    assert.equal(isTargetedRead({ view_range: [1, -1] }), false);
    assert.equal(isTargetedRead({ view_range: [50, -1] }), false);
});

// --- isLargeFile tests ---

test("isLargeFile returns false for small files", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const filePath = join(root, "small.md");
    writeFileSync(filePath, "line 1\nline 2\nline 3\n");

    assert.equal(isLargeFile(filePath), false);
});

test("isLargeFile returns true for files exceeding 200 lines", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const filePath = join(root, "large.md");
    const lines = Array.from({ length: 250 }, (_, i) => `line ${i + 1}: ${"x".repeat(60)}`);
    writeFileSync(filePath, lines.join("\n") + "\n");

    assert.equal(isLargeFile(filePath), true);
});

test("isLargeFile returns false for nonexistent files (fail open)", () => {
    assert.equal(isLargeFile("/tmp/nonexistent-ma-test-file.txt"), false);
});

test("isLargeFile returns false for files under size gate even with many short lines", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const filePath = join(root, "short-lines.txt");
    // 300 lines but each is just 2 bytes — total ~600 bytes, under the 10KB gate
    const lines = Array.from({ length: 300 }, () => "x");
    writeFileSync(filePath, lines.join("\n") + "\n");

    assert.equal(isLargeFile(filePath), false);
});

test("isLargeFile resolves relative paths against cwd", () => {
    const root = mkdtempSync(join(tmpdir(), "ma-extension-"));
    const filePath = join(root, "relative.md");
    const lines = Array.from({ length: 250 }, (_, i) => `line ${i + 1}: ${"x".repeat(60)}`);
    writeFileSync(filePath, lines.join("\n") + "\n");

    assert.equal(isLargeFile("relative.md", root), true);
});
