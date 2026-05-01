import { lstatSync, readFileSync, realpathSync, statSync } from "node:fs";

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
