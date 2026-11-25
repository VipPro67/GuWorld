// Combat log semantics live here. Backend combat logs carry an `action` string
// plus optional Vietnamese `detail`; views must not compare those protocol
// strings or parse the detail text themselves. Everything that knows the wire
// format is in this module so a new action is added in one place.

export interface CombatLogLike {
  action?: string;
  detail?: string;
  damage?: number;
}

export type CombatLogKind =
  | "basicAttack"
  | "guSkill"
  | "killerMove"
  | "damage"
  | "damageApplied"
  | "dotTick"
  | "thorns"
  | "phantomAttack"
  | "shieldPierce"
  | "shieldBypassAttack"
  | "heal"
  | "itemHeal"
  | "lifesteal"
  | "shieldGain"
  | "preCombatBarrier"
  | "shieldAbsorb"
  | "shieldBreak"
  | "died"
  | "manaCost"
  | "manaRestore"
  | "apertureDamage"
  | "autoItemUsage"
  | "autoStoneUsage"
  | "guBacklash"
  | "stunned"
  | "buff"
  | "defend"
  | "mineLoot"
  | "mineOccupy"
  | "loot"
  | "arenaReward"
  | "arenaRankUp"
  | "pathMasteryGain"
  | "petDeath"
  | "turnStart"
  | "turnEnd"
  | "battleStart"
  | "battleEnd"
  | "awaitCommand"
  | "other";

const KIND_BY_ACTION: Record<string, CombatLogKind> = {
  BASIC_ATTACK: "basicAttack",
  GU_SKILL: "guSkill",
  KILLER_MOVE: "killerMove",
  DAMAGE: "damage",
  DAMAGE_APPLIED: "damageApplied",
  DOT_TICK: "dotTick",
  THORNS: "thorns",
  PHANTOM_ATTACK: "phantomAttack",
  SHIELD_PIERCE: "shieldPierce",
  SHIELD_BYPASS_ATTACK: "shieldBypassAttack",
  HEAL: "heal",
  ITEM_HEAL: "itemHeal",
  LIFESTEAL: "lifesteal",
  SHIELD_GAIN: "shieldGain",
  PRE_COMBAT_BARRIER: "preCombatBarrier",
  SHIELD_ABSORB: "shieldAbsorb",
  SHIELD_BREAK: "shieldBreak",
  DIED: "died",
  MANA_COST: "manaCost",
  MANA_RESTORE: "manaRestore",
  APERTURE_DAMAGE: "apertureDamage",
  AUTO_ITEM_USAGE: "autoItemUsage",
  AUTO_STONE_USAGE: "autoStoneUsage",
  GU_BACKLASH: "guBacklash",
  STUNNED: "stunned",
  BUFF: "buff",
  DEFEND: "defend",
  MINE_LOOT: "mineLoot",
  MINE_OCCUPY: "mineOccupy",
  LOOT: "loot",
  ARENA_REWARD: "arenaReward",
  ARENA_RANK_UP: "arenaRankUp",
  PATH_MASTERY_GAIN: "pathMasteryGain",
  PET_DEATH: "petDeath",
  TURN_START: "turnStart",
  TURN_END: "turnEnd",
  BATTLE_START: "battleStart",
  BATTLE_END: "battleEnd",
  AWAIT_COMMAND: "awaitCommand",
};

export function combatLogKind(log: CombatLogLike): CombatLogKind {
  return KIND_BY_ACTION[log.action || ""] ?? "other";
}

const BOUNDARY_KINDS = new Set<CombatLogKind>([
  "awaitCommand",
  "battleEnd",
  "battleStart",
  "turnEnd",
  "turnStart",
]);

/** isActionLog is false for log entries that only frame a turn or the battle. */
export function isActionLog(log: CombatLogLike): boolean {
  return !BOUNDARY_KINDS.has(combatLogKind(log));
}

const HEAL_KINDS = new Set<CombatLogKind>(["heal", "itemHeal", "lifesteal"]);
export function isHealKind(kind: CombatLogKind): boolean {
  return HEAL_KINDS.has(kind);
}

const SHIELD_ADD_KINDS = new Set<CombatLogKind>(["shieldGain", "preCombatBarrier"]);
export function addsShieldKind(kind: CombatLogKind): boolean {
  return SHIELD_ADD_KINDS.has(kind);
}

const TARGET_HP_DAMAGE_KINDS = new Set<CombatLogKind>([
  "damage",
  "basicAttack",
  "guSkill",
  "killerMove",
  "phantomAttack",
  "thorns",
  "dotTick",
]);
/** reducesTargetHp reports whether the log subtracts `damage` from target HP. */
export function reducesTargetHp(kind: CombatLogKind): boolean {
  return TARGET_HP_DAMAGE_KINDS.has(kind);
}

const FLOATING_DAMAGE_KINDS = new Set<CombatLogKind>([
  "damage",
  "dotTick",
  "thorns",
  "phantomAttack",
  "shieldPierce",
  "shieldBypassAttack",
]);
/** showsFloatingDamage reports whether the log shows a floating damage number. */
export function showsFloatingDamage(kind: CombatLogKind): boolean {
  return FLOATING_DAMAGE_KINDS.has(kind);
}

const INLINE_DAMAGE_HIDDEN_KINDS = new Set<CombatLogKind>([
  "dotTick",
  "shieldGain",
  "shieldAbsorb",
  "heal",
  "itemHeal",
  "lifesteal",
]);
/**
 * showsInlineDamage reports whether the log's damage should render as an inline
 * "-N" chip. DoT, shields and heals show their amount through the detail text.
 */
export function showsInlineDamage(kind: CombatLogKind): boolean {
  return !INLINE_DAMAGE_HIDDEN_KINDS.has(kind);
}

/** fallbackActionLabel is the dev-readable label when a log has no detail. */
export function fallbackActionLabel(log: CombatLogLike): string {
  return (log.action || "").replaceAll("_", " ").toLowerCase();
}

const REWARD_KINDS = new Set<CombatLogKind>([
  "loot",
  "arenaReward",
  "arenaRankUp",
  "mineLoot",
  "mineOccupy",
  "pathMasteryGain",
]);
export function isRewardKind(kind: CombatLogKind): boolean {
  return REWARD_KINDS.has(kind);
}

const SKILL_KINDS = new Set<CombatLogKind>(["guSkill", "killerMove", "basicAttack"]);
/** isSkillCastKind reports whether the log announces the skill that was used. */
export function isSkillCastKind(kind: CombatLogKind): boolean {
  return SKILL_KINDS.has(kind);
}

const TURN_LABEL = /=== LƯỢT (\d+) ===/;

/** turnNumberFromLog extracts the turn number from a TURN_START log. */
export function turnNumberFromLog(log: CombatLogLike): number | null {
  const match = TURN_LABEL.exec(log.detail || "");
  return match ? parseInt(match[1], 10) : null;
}

/**
 * skillNameFromDetail extracts the bracketed skill name from an action log
 * ("... dùng [Tên] ..." or "... sử dụng [Tên] ..."). Returns null when the
 * detail has no skill announcement; callers render no floating skill label.
 */
export function skillNameFromDetail(detail?: string): string | null {
  const text = detail || "";
  const idxDung = text.indexOf("dùng [");
  const idxSuDung = text.indexOf("sử dụng");
  const idx = idxDung >= 0 ? idxDung : idxSuDung;
  if (idx < 0) return null;
  const match = text.substring(idx).match(/\[([^\]]+)\]/);
  return match ? match[1] : null;
}

/** autoUsageCount extracts the item/stone count from an auto-usage detail. */
export function autoUsageCount(detail?: string): number {
  const match = (detail || "").match(/(\d+)/);
  return match ? parseInt(match[1], 10) : 0;
}

/** isLossLog reports whether a log entry is a loss line (pet death or loot drop). */
export function isLossLog(log: CombatLogLike): boolean {
  const kind = combatLogKind(log);
  if (kind === "petDeath") return true;
  return kind === "battleEnd" && (log.detail || "").includes("rơi rớt mất");
}
