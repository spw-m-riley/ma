import { readFileSync, writeFileSync } from "node:fs";
import type { Config } from "./types";

export function render(config: Config): string {
  if (!config.enabled) {
    return "";
  }

  return readFileSync(config.path, "utf8");
}

export function save(config: Config, value: string): void {
  writeFileSync(config.path, value);
}
