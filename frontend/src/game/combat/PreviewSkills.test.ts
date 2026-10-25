import assert from "node:assert/strict";
import test from "node:test";
import { PREVIEW_SKILLS, previewSkill } from "./PreviewSkills.ts";

test("preview loadout exposes six slots with four target modes", () => {
  assert.equal(PREVIEW_SKILLS.length, 6);
  assert.deepEqual(
    PREVIEW_SKILLS.map((skill) => skill.slot),
    [1, 2, 3, 4, 5, 6],
  );
  assert.equal(previewSkill(2)?.mode, "DIRECTION");
  assert.equal(previewSkill(3)?.mode, "GROUND");
  assert.equal(previewSkill(4)?.mode, "SELF");
  assert.equal(previewSkill(1)?.mode, "TARGET");
  assert.equal(previewSkill(7), null);
});

test("preview skills carry no client damage or cooldown authority", () => {
  for (const skill of PREVIEW_SKILLS) {
    assert.equal("damage" in skill, false);
    assert.ok(skill.cooldownMs > 0);
    assert.ok(skill.manaCost >= 0);
  }
});
