import { lstatSync, readFileSync, realpathSync, statSync, openSync, readSync, closeSync } from "node:fs";
import path from "node:path";

const SENSITIVE_BASENAMES = new Set([
    ".env", ".env.local", "id_rsa", "id_ed25519",
    "credentials", "known_hosts", "authorized_keys",
]);

const SENSITIVE_COMPONENTS = new Set([".ssh", ".aws", ".gnupg", ".kube"]);

export function isSensitivePath(filePath) {
    const parts = filePath.split(/[\\/]/);
    const base = parts[parts.length - 1];
    if (SENSITIVE_BASENAMES.has(base)) return true;
    return parts.some((part) => SENSITIVE_COMPONENTS.has(part));
}

export function isSensitivePathResolved(filePath) {
    if (isSensitivePath(filePath)) {
        return true;
    }

    try {
        return isSensitivePath(realpathSync(filePath));
    } catch {
        try {
            return lstatSync(filePath).isSymbolicLink();
        } catch {
            return false;
        }
    }
}

export function sensitivePathResponse(filePath) {
    return {
        textResultForLlm: `Refused: sensitive path ${filePath}`,
        resultType: "denied",
    };
}

export function fallbackForRead(filePath, error) {
    if (looksLikeSensitivePathError(error) || isSensitivePathResolved(filePath)) {
        return sensitivePathResponse(filePath);
    }

    try {
        return readFileSync(filePath, "utf-8");
    } catch (readError) {
        return {
            textResultForLlm: `Error reading file: ${readError.message}`,
            resultType: "failure",
        };
    }
}

export function findMaBinary() {
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

function looksLikeSensitivePathError(error) {
    const message = typeof error?.message === "string" ? error.message : String(error ?? "");
    return message.includes("refusing sensitive path");
}

// --- View intercept helpers ---

export const VIEW_LINE_THRESHOLD = 200;
export const VIEW_SIZE_GATE_BYTES = 10 * 1024; // 10KB fast gate before line counting

export function isLargeFile(filePath, cwd) {
    try {
        const resolved = path.isAbsolute(filePath)
            ? filePath
            : path.join(cwd || process.cwd(), filePath);
        const { size } = statSync(resolved);
        if (size < VIEW_SIZE_GATE_BYTES) return false;

        // Count newlines up to threshold without reading the whole file
        return countLinesExceeds(resolved, VIEW_LINE_THRESHOLD);
    } catch {
        return false; // fail open
    }
}

function countLinesExceeds(filePath, threshold) {
    let fd;
    try {
        fd = openSync(filePath, "r");
        const buf = Buffer.alloc(32768);
        let lines = 0;
        let bytesRead;
        while ((bytesRead = readSync(fd, buf, 0, buf.length)) > 0) {
            for (let i = 0; i < bytesRead; i++) {
                if (buf[i] === 0x0a) lines++;
            }
            if (lines >= threshold) return true;
        }
        return false;
    } catch {
        return false; // fail open
    } finally {
        if (fd !== undefined) {
            try { closeSync(fd); } catch { /* ignore */ }
        }
    }
}

export function isTargetedRead(toolArgs) {
    const range = toolArgs?.view_range;
    if (!range || !Array.isArray(range) || range.length < 2) return false;
    const [start, end] = range;
    // end === -1 means "to EOF" — treat as full read
    if (end === -1) return false;
    const span = end - start + 1;
    return span > 0 && span < VIEW_LINE_THRESHOLD;
}
