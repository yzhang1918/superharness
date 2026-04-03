#!/usr/bin/env node

import fs from "node:fs";
import { createRequire } from "node:module";

const require = createRequire(new URL("../web/package.json", import.meta.url));
const { PNG } = require("pngjs");
const pixelmatchModule = require("pixelmatch");
const pixelmatch = pixelmatchModule.default ?? pixelmatchModule;

if (process.argv.length < 5) {
  console.error("usage: compare-review-visual.mjs <baseline> <candidate> <label>");
  process.exit(1);
}

const [, , baselinePath, candidatePath, label] = process.argv;
const baseline = PNG.sync.read(fs.readFileSync(baselinePath));
const candidate = PNG.sync.read(fs.readFileSync(candidatePath));

if (baseline.width !== candidate.width || baseline.height !== candidate.height) {
  console.error(
    `${label} dimensions changed: baseline ${baseline.width}x${baseline.height}, candidate ${candidate.width}x${candidate.height}`,
  );
  process.exit(1);
}

const diffPixels = pixelmatch(
  baseline.data,
  candidate.data,
  null,
  baseline.width,
  baseline.height,
  {
    threshold: 0.16,
    includeAA: false,
  },
);

const ratio = diffPixels / (baseline.width * baseline.height);
if (ratio > 0.01) {
  console.error(`${label} visual regression exceeded tolerance: ${diffPixels} pixels (${(ratio * 100).toFixed(2)}%)`);
  process.exit(1);
}
