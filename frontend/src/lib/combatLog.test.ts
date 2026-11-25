import test from "node:test";
import assert from "node:assert/strict";
import {
  combatLogKind,
  isActionLog,
  isHealKind,
  addsShieldKind,
  reducesTargetHp,
  showsFloatingDamage,
  showsInlineDamage,
  isRewardKind,
  isSkillCastKind,
  turnNumberFromLog,
  skillNameFromDetail,
  autoUsageCount,
  isLossLog,
  fallbackActionLabel,
} from "./combatLog.ts";

test("combatLogKind: every known action maps to a distinct kind", () => {
  const actions = [
    "BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "DAMAGE", "DAMAGE_APPLIED",
    "DOT_TICK", "THORNS", "PHANTOM_ATTACK", "SHIELD_PIERCE", "SHIELD_BYPASS_ATTACK",
    "HEAL", "ITEM_HEAL", "LIFESTEAL", "SHIELD_GAIN", "PRE_COMBAT_BARRIER",
    "SHIELD_ABSORB", "SHIELD_BREAK", "DIED", "MANA_COST", "MANA_RESTORE",
    "APERTURE_DAMAGE", "AUTO_ITEM_USAGE", "AUTO_STONE_USAGE", "GU_BACKLASH",
    "STUNNED", "BUFF", "DEFEND", "MINE_LOOT", "MINE_OCCUPY", "LOOT",
    "ARENA_REWARD", "ARENA_RANK_UP", "PATH_MASTERY_GAIN", "PET_DEATH",
    "TURN_START", "TURN_END", "BATTLE_START", "BATTLE_END", "AWAIT_COMMAND",
  ];
  const kinds = new Set(actions.map((action) => combatLogKind({ action })));
  assert.equal(kinds.size, actions.length);
  assert.equal(kinds.has("other"), false);
});

test("combatLogKind: unknown and missing actions are 'other'", () => {
  assert.equal(combatLogKind({ action: "SOMETHING_NEW" }), "other");
  assert.equal(combatLogKind({}), "other");
});

test("isActionLog: turn and battle framing is not an action", () => {
  for (const action of ["TURN_START", "TURN_END", "BATTLE_START", "BATTLE_END", "AWAIT_COMMAND"]) {
    assert.equal(isActionLog({ action }), false, action);
  }
  for (const action of ["DIED", "LOOT", "PET_DEATH", "MANA_COST"]) {
    assert.equal(isActionLog({ action }), true, action);
  }
});

test("kind groups: heal, shield gain, target HP damage, floating damage", () => {
  for (const action of ["HEAL", "ITEM_HEAL", "LIFESTEAL"]) {
    assert.equal(isHealKind(combatLogKind({ action })), true, action);
  }
  for (const action of ["SHIELD_GAIN", "PRE_COMBAT_BARRIER"]) {
    assert.equal(addsShieldKind(combatLogKind({ action })), true, action);
  }
  for (const action of ["DAMAGE", "BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "PHANTOM_ATTACK", "THORNS", "DOT_TICK"]) {
    assert.equal(reducesTargetHp(combatLogKind({ action })), true, action);
  }
  assert.equal(reducesTargetHp(combatLogKind({ action: "SHIELD_PIERCE" })), false);
  for (const action of ["DAMAGE", "DOT_TICK", "THORNS", "PHANTOM_ATTACK", "SHIELD_PIERCE", "SHIELD_BYPASS_ATTACK"]) {
    assert.equal(showsFloatingDamage(combatLogKind({ action })), true, action);
  }
  assert.equal(showsFloatingDamage(combatLogKind({ action: "HEAL" })), false);
});

test("kind groups: rewards and skill casts", () => {
  for (const action of ["LOOT", "ARENA_REWARD", "ARENA_RANK_UP", "MINE_LOOT", "MINE_OCCUPY", "PATH_MASTERY_GAIN"]) {
    assert.equal(isRewardKind(combatLogKind({ action })), true, action);
  }
  assert.equal(isRewardKind(combatLogKind({ action: "DAMAGE" })), false);
  for (const action of ["GU_SKILL", "KILLER_MOVE", "BASIC_ATTACK"]) {
    assert.equal(isSkillCastKind(combatLogKind({ action })), true, action);
  }
  assert.equal(isSkillCastKind(combatLogKind({ action: "HEAL" })), false);
});

test("turnNumberFromLog: parses the turn header only", () => {
  assert.equal(turnNumberFromLog({ detail: "=== LƯỢT 12 ===" }), 12);
  assert.equal(turnNumberFromLog({ detail: "LƯỢT 12" }), null);
  assert.equal(turnNumberFromLog({}), null);
});

test("skillNameFromDetail: handles both cast keywords and misses", () => {
  assert.equal(skillNameFromDetail("A dùng [Hỏa Cầu] lên B"), "Hỏa Cầu");
  assert.equal(skillNameFromDetail("A sử dụng [Băng Kiếm]"), "Băng Kiếm");
  assert.equal(skillNameFromDetail("A sử dụng chiêu thức"), null);
  assert.equal(skillNameFromDetail("A dùng [Không Đóng] lên B"), "Không Đóng");
  assert.equal(skillNameFromDetail(undefined), null);
  assert.equal(skillNameFromDetail(""), null);
});

test("autoUsageCount: reads the first integer from the detail", () => {
  assert.equal(autoUsageCount("Tự động dùng 3 viên đan"), 3);
  assert.equal(autoUsageCount("Tự động dùng 12 Nguyên Thạch"), 12);
  assert.equal(autoUsageCount("không có số"), 0);
  assert.equal(autoUsageCount(undefined), 0);
});

test("isLossLog: pet death and loot-drop battle end are losses", () => {
  assert.equal(isLossLog({ action: "PET_DEATH" }), true);
  assert.equal(isLossLog({ action: "BATTLE_END", detail: "Bạn đã rơi rớt mất 3 vật phẩm" }), true);
  assert.equal(isLossLog({ action: "BATTLE_END", detail: "Chiến thắng" }), false);
  assert.equal(isLossLog({ action: "DIED" }), false);
});

test("showsInlineDamage: DoT, shields and heals suppress the inline chip", () => {
  for (const action of ["DOT_TICK", "SHIELD_GAIN", "SHIELD_ABSORB", "HEAL", "ITEM_HEAL", "LIFESTEAL"]) {
    assert.equal(showsInlineDamage(combatLogKind({ action })), false, action);
  }
  for (const action of ["DAMAGE", "BASIC_ATTACK", "GU_SKILL", "MANA_COST"]) {
    assert.equal(showsInlineDamage(combatLogKind({ action })), true, action);
  }
});

test("fallbackActionLabel: humanizes the wire action", () => {
  assert.equal(fallbackActionLabel({ action: "GU_SKILL" }), "gu skill");
  assert.equal(fallbackActionLabel({}), "");
});
