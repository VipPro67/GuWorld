import assert from "node:assert/strict";
import test from "node:test";
import { EMPTY_EFFECT_ENVELOPE, captureBonus, combatCategory, disguiseLevel, effectEnvelope, growthConfig, hasCapability, investigation, isCombatEligible, normalizeEffectEnvelope, operationsBy, portalRange, statModifier, type EffectEnvelope } from "./effect-blocks.ts";
const effects: EffectEnvelope = { schema_version: 2, blocks: [
  { id: "passive", trigger: "ALWAYS_ON", target: { selector: "SELF" }, operations: [
    { op: "STAT_MOD", stat: "INVESTIGATION", value: 2 }, { op: "STAT_MOD", stat: "INVESTIGATION", value: 3 },
    { op: "STAT_MOD", stat: "DISGUISE", value: 4 }, { op: "STAT_MOD", stat: "SPEED", value: 0.2 },
  ] },
  { id: "attack", trigger: "ON_ACTION", target: { selector: "TARGET" }, operations: [{ op: "DAMAGE", value: 10 }] },
  { id: "capture", trigger: "CAPABILITY", target: { selector: "SELF" }, operations: [
    { op: "UNIQUE", code: "CAPTURE_BONUS", params: { value: 0.25 } }, { op: "UNIQUE", code: "DISGUISE" },
  ] },
  { id: "growth", trigger: "ALWAYS_ON", target: { selector: "SELF" }, operations: [
    { op: "STAT_MOD", sub_type: "GROWTH_CAP", stat: "ATTACK", value: 12 }, { op: "UNIQUE", code: "GROWTH_DURATION_HOURS", params: { value: 48 } },
  ] },
] };
test("effectEnvelope accepts block envelopes with or without an explicit schema version", () => { assert.deepEqual(effectEnvelope({ blocks: effects.blocks }), { blocks: effects.blocks }); assert.equal(effectEnvelope(effects), effects); assert.equal(effectEnvelope({ effects }), effects); assert.equal(effectEnvelope(null), EMPTY_EFFECT_ENVELOPE); });
test("normalizeEffectEnvelope loads Gu.csv arrays, API envelopes, and JSON strings", () => {
  assert.deepEqual(normalizeEffectEnvelope(effects.blocks), effects);
  assert.deepEqual(normalizeEffectEnvelope(effects), effects);
  assert.deepEqual(normalizeEffectEnvelope(JSON.stringify(effects.blocks)), effects);
  assert.deepEqual(normalizeEffectEnvelope("not json"), { schema_version: 2, blocks: [] });
});
test("modifier selectors sum canonical operations", () => { assert.equal(investigation(effects), 5); assert.equal(disguiseLevel(effects), 4); assert.equal(statModifier(effects, "speed"), 0.2); assert.equal(operationsBy(effects, { op: "damage" }).length, 1); });
test("domain selectors read block operations", () => { assert.equal(hasCapability(effects, "disguise"), true); assert.equal(isCombatEligible(effects), true); assert.equal(captureBonus(effects), 0.25); assert.equal(combatCategory(effects), "attack"); assert.deepEqual(growthConfig(effects), { max_attack: 12, max_defense: 0, max_hp: 0, max_speed: 0, hours_to_max: 48 }); });
test("capture-only Gu stays battle-eligible", () => {
  const captureOnly: EffectEnvelope = { schema_version: 2, blocks: [
    { id: "CAPTURE_BONUS", trigger: "CAPABILITY", target: { selector: "SELF" }, operations: [{ op: "UNIQUE", code: "CAPTURE_BONUS", params: { value: 0.1 } }] },
  ] };
  assert.equal(isCombatEligible(captureOnly), true);
  assert.equal(captureBonus(captureOnly), 0.1);
});

test("portalRange follows explicit params, ignore capability, and rank fallback", () => {
  const portal = (code: string, params?: Record<string, unknown>): EffectEnvelope => ({ schema_version: 2, blocks: [
    { id: "portal", trigger: "CAPABILITY", target: { selector: "SELF" }, operations: [{ op: "UNIQUE", code, params }] },
  ] });
  assert.equal(portalRange(portal("PORTAL_CREATE", { range: 1234 }), 5), 1234);
  assert.equal(portalRange(portal("RANDOM_PORTAL", { range: 777 }), 5), 777);
  assert.equal(portalRange(portal("PORTAL_CREATE", {}), 5), 1000);
  assert.equal(portalRange(portal("PORTAL_CREATE", {}), 9), 30000);
  assert.equal(portalRange(portal("IGNORE_MOVEMENT_RANGE"), 5), 0);
  const both: EffectEnvelope = { schema_version: 2, blocks: [
    { id: "portal", trigger: "CAPABILITY", target: { selector: "SELF" }, operations: [
      { op: "UNIQUE", code: "PORTAL_CREATE", params: { range: 1500 } }, { op: "UNIQUE", code: "IGNORE_MOVEMENT_RANGE" },
    ] },
  ] };
  assert.equal(portalRange(both, 5), 1500);
});
