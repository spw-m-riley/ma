import { readFileSync, writeFileSync } from "node:fs";
import { basename, dirname, extname, join } from "node:path";
import { EOL } from "node:os";
import { createHash } from "node:crypto";
import { performance } from "node:perf_hooks";
import type { Config } from "./types";

export function render(config: Config): string {
  if (!config.enabled) {
    return "";
  }

  const startedAt = performance.now();
  const source = readFileSync(config.path, "utf8");
  const lines = source.split(/\r?\n/);
  const summary: string[] = [];

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    summary.push(trimmed.toUpperCase());
    if (summary.length >= 5) {
      break;
    }
  }

  const digest = createHash("sha1").update(source).digest("hex").slice(0, 8);
  const duration = Math.round(performance.now() - startedAt);
  const filename = basename(config.path);
  const directory = dirname(config.path);
  const extension = extname(config.path);

  return [
    `file=${filename}`,
    `dir=${directory}`,
    `ext=${extension}`,
    `digest=${digest}`,
    `duration=${duration}`,
    summary.join(" | "),
  ].join(EOL);
}

export function save(config: Config, value: string): void {
  const destination = join(dirname(config.path), basename(config.path));
  const normalized = value
    .split(/\r?\n/)
    .map((line) => line.trimEnd())
    .join(EOL);

  if (!config.enabled) {
    return;
  }

  writeFileSync(destination, normalized);
}
