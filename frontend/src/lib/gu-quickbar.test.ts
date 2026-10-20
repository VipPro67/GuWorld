import test from "node:test";
import assert from "node:assert/strict";
import type { EffectBlock } from "./effect-blocks.ts";
import {
  COMBAT_QUICKBAR_EFFECT_TYPES,
  classifyGuQuickbarEffect,
  isFlyingStatus,
} from "./gu-quickbar.ts";

const block = (trigger: string, operations: EffectBlock["operations"]): EffectBlock => ({
  id: `${trigger}_BLOCK`,
  trigger,
  target: { selector: "SELF" },
  operations,
});

test("pure flying Gu is exposed as a fly quickbar slot", () => {
  const effect = classifyGuQuickbarEffect([
    block("ALWAYS_ON", [{ op: "STAT_MOD", stat: "SPEED", sub_type: "ADD_FLAT", value: 2 }]),
    block("ON_ACTION", [{ op: "STATUS", status_id: "FLYING", turns: 1, value: 0, chance: 1 }]),
  ]);
  assert.deepEqual(effect, { type: "fly", label: "Phi Hành", isTargeted: false });
  assert.equal(COMBAT_QUICKBAR_EFFECT_TYPES.has(effect!.type), false);
});

test("flying wins over a haste combat status on an existing flying Gu", () => {
  const effect = classifyGuQuickbarEffect([
    block("ON_ACTION", [{ op: "STATUS", status_id: "HASTE", turns: 2, value: 35, chance: 1 }]),
    block("ON_ACTION", [{ op: "STATUS", status_id: "FLYING", turns: 5, value: 0, chance: 1 }]),
  ]);
  assert.equal(effect?.type, "fly");
});

test("flying wins over damage on a hybrid sword-wing Gu", () => {
  const effect = classifyGuQuickbarEffect([
    block("ON_ACTION", [
      { op: "DAMAGE", value: 3000, sub_type: "FLAT" },
      { op: "STATUS", status_id: "BLEED", turns: 3, value: 300, chance: 1 },
    ]),
    block("ON_ACTION", [{ op: "STATUS", status_id: "FLYING", turns: 5, value: 0, chance: 1 }]),
  ]);
  assert.equal(effect?.type, "fly");
  assert.equal(COMBAT_QUICKBAR_EFFECT_TYPES.has(effect!.type), false);
});

test("legacy PHI_HANH status is treated as flying", () => {
  assert.equal(isFlyingStatus("phi_hanh"), true);
  assert.equal(isFlyingStatus("PHI_HANH"), true);
  assert.equal(isFlyingStatus("FLYING"), true);
  assert.equal(isFlyingStatus("HASTE"), false);

  const effect = classifyGuQuickbarEffect([
    block("ON_ACTION", [{ op: "STATUS", status_id: "PHI_HANH", turns: 3, value: 0, chance: 1 }]),
  ]);
  assert.equal(effect?.type, "fly");
});

test("ordinary combat effects stay off the world-map quickbar", () => {
  const haste = classifyGuQuickbarEffect([
    block("ON_ACTION", [{ op: "STATUS", status_id: "HASTE", turns: 2, value: 35, chance: 1 }]),
  ]);
  assert.equal(haste?.type, "buff_speed");
  assert.equal(COMBAT_QUICKBAR_EFFECT_TYPES.has(haste!.type), true);

  const damage = classifyGuQuickbarEffect([
    block("ON_ACTION", [{ op: "DAMAGE", value: 30, sub_type: "FLAT" }]),
  ]);
  assert.equal(damage?.type, "damage");
  assert.equal(COMBAT_QUICKBAR_EFFECT_TYPES.has(damage!.type), true);
});

test("non-castable Gu has no quickbar slot", () => {
  assert.equal(classifyGuQuickbarEffect([
    block("ALWAYS_ON", [{ op: "STAT_MOD", stat: "SPEED", sub_type: "ADD_FLAT", value: 2 }]),
  ]), null);
});
