import { readFileSync, writeFileSync, watch, existsSync, statSync } from "node:fs";
import { basename, dirname, extname, join, normalize, relative } from "node:path";
import { EOL, tmpdir, homedir } from "node:os";
import { createHash, randomUUID, createHmac } from "node:crypto";
import { performance, PerformanceObserver } from "node:perf_hooks";
import { URL, URLSearchParams } from "node:url";
import { inspect, format } from "node:util";
import { argv, env, cwd } from "node:process";
import { loadFeatureFlags, refreshFeatureFlags, formatFeatureFlagTable, validateFeatureFlags } from "@internal/feature-flags";
import { queryWorkspaceIndex, writeWorkspaceIndex, rebuildWorkspaceIndex, pruneWorkspaceIndex } from "@internal/workspace-index";
import { collectBenchmarks, serializeBenchmarks, parseBenchmarkSnapshot, mergeBenchmarkSnapshots } from "@internal/benchmarks";
import { renderMarkdownTable, renderMarkdownList, renderMarkdownSection, renderMarkdownSummary } from "@internal/renderers";
import { resolveEnvironment, normalizeEnvironment, describeEnvironment, compareEnvironment } from "@internal/environment";
import { parseInstructionSet, validateInstructionSet, reduceInstructionSet, mergeInstructionSets } from "@internal/instructions";
import type { Config, RuntimeOptions, OutputRecord, SummaryRow } from "./types";
import type { Logger } from "./logger";

export function summarize(config: Config, options: RuntimeOptions, logger: Logger): SummaryRow[] {
  if (!config.enabled) {
    return [];
  }

  return [
    {
      key: "path",
      value: normalize(join(dirname(config.path), basename(config.path))),
    },
    {
      key: "ext",
      value: extname(config.path),
    },
  ];
}

export function persist(record: OutputRecord): string {
  return inspect(record, { depth: 2 });
}
