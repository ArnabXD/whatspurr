#!/usr/bin/env bun

/**
 * Build the Go bridge binary for local development.
 * Usage: bun run scripts/build-go.ts
 */

import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { $ } from "bun";

const ROOT = join(import.meta.dirname, "..");
const GO_DIR = join(ROOT, "go");
const BIN_DIR = join(ROOT, "bin");
const OUTPUT = join(BIN_DIR, process.platform === "win32" ? "bridge.exe" : "bridge");

if (!existsSync(BIN_DIR)) {
  mkdirSync(BIN_DIR);
}

console.log("Resolving Go dependencies...");
await $`cd ${GO_DIR} && go mod tidy`;

console.log("Building Go bridge binary...");
await $`cd ${GO_DIR} && go build -o ${OUTPUT} .`;

console.log(`✓ Built: ${OUTPUT}`);
