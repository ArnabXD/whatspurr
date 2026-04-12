#!/usr/bin/env bun
/**
 * Cross-compile Go bridge for all supported platforms.
 * Usage: bun run scripts/release.ts
 *
 * Outputs binaries to dist/ directory.
 */

import { $ } from "bun";
import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";

const ROOT = join(import.meta.dirname, "..");
const GO_DIR = join(ROOT, "go");
const DIST_DIR = join(ROOT, "dist");

const targets = [
  { goos: "darwin", goarch: "arm64" },
  { goos: "darwin", goarch: "amd64" },
  { goos: "linux", goarch: "amd64" },
  { goos: "linux", goarch: "arm64" },
  { goos: "windows", goarch: "amd64" },
];

if (!existsSync(DIST_DIR)) {
  mkdirSync(DIST_DIR);
}

console.log("Resolving Go dependencies...");
await $`cd ${GO_DIR} && go mod tidy`;

for (const { goos, goarch } of targets) {
  const ext = goos === "windows" ? ".exe" : "";
  const output = join(DIST_DIR, `bridge-${goos}-${goarch}${ext}`);
  console.log(`Building ${goos}/${goarch}...`);
  await $`cd ${GO_DIR} && CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} go build -trimpath -ldflags="-s -w" -o ${output} .`;
}

console.log(`\n✓ All binaries built in ${DIST_DIR}/`);