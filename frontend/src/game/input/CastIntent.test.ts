import assert from "node:assert/strict";
import test from "node:test";
import { buildCastIntent } from "./CastIntent.ts";

const caster = { x: 2, y: 3 };

test("buildCastIntent creates mode-specific payloads without caller policy", () => {
  assert.deepEqual(buildCastIntent({ slot: 1, mode: "SELF" }, caster), {
    slot: 1,
    mode: "SELF",
  });
  assert.deepEqual(buildCastIntent({ slot: 2, mode: "TARGET" }, caster, { x: 4, y: 5 }, "mob-1"), {
    slot: 2,
    mode: "TARGET",
    target_id: "mob-1",
  });
  assert.deepEqual(buildCastIntent({ slot: 3, mode: "TARGET" }, caster, { x: 5, y: 7 }), {
    slot: 3,
    mode: "TARGET",
    aim_x: 3,
    aim_y: 4,
  });
  assert.deepEqual(buildCastIntent({ slot: 4, mode: "DIRECTION" }, caster, { x: 5, y: 7 }), {
    slot: 4,
    mode: "DIRECTION",
    aim_x: 0.6,
    aim_y: 0.8,
  });
  assert.deepEqual(buildCastIntent({ slot: 5, mode: "GROUND" }, caster, { x: 5, y: 7 }), {
    slot: 5,
    mode: "GROUND",
    x: 5,
    y: 7,
  });
});

test("buildCastIntent rejects missing or degenerate aim points", () => {
  assert.equal(buildCastIntent({ slot: 1, mode: "TARGET" }, caster), null);
  assert.equal(buildCastIntent({ slot: 1, mode: "DIRECTION" }, caster, { x: 2.05, y: 3 }), null);
  assert.equal(buildCastIntent({ slot: 1, mode: "GROUND" }, caster), null);
});