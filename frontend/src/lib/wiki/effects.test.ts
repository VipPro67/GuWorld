import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { EFFECT_OPERATIONS, EFFECT_TRIGGERS, NATIVE_HOOKS } from "./effect-glossary.ts";

const here = path.dirname(fileURLToPath(import.meta.url));
const guCsvPath = path.resolve(here, "../../../../backend/data/Gu.csv");

function collect(pattern: RegExp): Set<string> {
  const csv = fs.readFileSync(guCsvPath, "utf8");
  const found = new Set<string>();
  for (const match of csv.matchAll(pattern)) found.add(match[1]);
  return found;
}

// CSV quotes are doubled inside the effects field: ""trigger"": ""ON_ACTION"".
const csvTriggers = collect(/""trigger"":\s*""([A-Z_]+)""/g);
const csvOps = collect(/""op"":\s*""([A-Z_]+)""/g);
const csvCodes = collect(/""code"":\s*""([A-Z_]+)""/g);

test("glossary covers every trigger used by Gu.csv", () => {
  const known = new Set(EFFECT_TRIGGERS.map((t) => t.code));
  const missing = [...csvTriggers].filter((code) => !known.has(code)).sort();
  assert.deepEqual(missing, [], `Thêm trigger vào effect-glossary.ts: ${missing.join(", ")}`);
});

test("glossary covers every operation used by Gu.csv", () => {
  const known = new Set(EFFECT_OPERATIONS.map((op) => op.code));
  const missing = [...csvOps].filter((code) => !known.has(code)).sort();
  assert.deepEqual(missing, [], `Thêm operation vào effect-glossary.ts: ${missing.join(", ")}`);
});

test("glossary covers every native hook used by Gu.csv", () => {
  const known = new Set(NATIVE_HOOKS.map((hook) => hook.code));
  const missing = [...csvCodes].filter((code) => !known.has(code)).sort();
  assert.deepEqual(missing, [], `Thêm native hook vào effect-glossary.ts: ${missing.join(", ")}`);
});

test("native hook entries are unique and non-empty", () => {
  assert.equal(new Set(NATIVE_HOOKS.map((hook) => hook.code)).size, NATIVE_HOOKS.length);
  for (const hook of NATIVE_HOOKS) {
    assert.ok(hook.label && hook.trigger && hook.behavior, `Thiếu nội dung cho ${hook.code}`);
  }
});
