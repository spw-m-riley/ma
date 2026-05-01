import test from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { fallbackForRead, isSensitivePathResolved } from "./runtime.mjs";

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
    const path = join(root, "notes.md");
    writeFileSync(path, "safe fallback\n");

    const response = fallbackForRead(path, new Error("ma unavailable"));

    assert.equal(response, "safe fallback\n");
});
