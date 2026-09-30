/** Canonical frontend contract for the backend Effect Block v2 envelope. */
export type EffectScalar = string | number | boolean | null;

export interface EffectTarget {
  selector: string;
  bypass_frontline?: boolean;
}

export interface EffectCondition {
  field: string;
  operator: string;
  value: number;
}

export interface EffectOperation {
  op: string;
  sub_type?: string;
  target?: EffectTarget;
  value?: number;
  unit?: string;
  turns?: number;
  chance?: number;
  path?: string;
  status_id?: string;
  stat?: string;
  resource?: string;
  stack_policy?: string;
  code?: string;
  params?: Record<string, unknown>;
}

export interface EffectBlock {
  id: string;
  trigger: string;
  target: EffectTarget;
  conditions?: EffectCondition[];
  operations: EffectOperation[];
}

export interface EffectEnvelope {
  schema_version?: number;
  blocks: EffectBlock[];
}

export interface EffectDictionaryEntry {
  tier?: number;
  key: string;
  name: string;
  effects?: EffectEnvelope;
}

/** Metadata is supplied by the backend catalog; the frontend owns no enum copy. */
export interface EffectCatalogOperation {
  name: string; sub_types: string[]; triggers: string[]; targets: string[];
  required_fields: string[]; allowed_fields: string[];
}
export interface EffectParamSpec {
  type: "bool" | "string" | "integer" | "number"; required: boolean;
  min?: number; max?: number; enum?: Record<string, boolean>;
}
export interface EffectNativeHook {
  code: string; domain: string; triggers: string[]; params: Record<string, EffectParamSpec>;
}
export interface EffectCatalogMetadata {
  schema_version?: number; triggers: string[]; targets: string[]; operations: EffectCatalogOperation[];
  stats: string[]; statuses: string[]; resources: string[]; units: string[];
  stack_policies: string[]; condition_fields: string[]; condition_operators: string[];
  native_codes: string[]; native_hooks: EffectNativeHook[];
}

export const EMPTY_EFFECT_ENVELOPE: EffectEnvelope = { blocks: [] };

/**
 * Accepts both the database API envelope and the bare block array used by
 * Gu.csv. Keeping this conversion in one place prevents the admin editor from
 * silently showing an empty effect when it opens CSV-shaped data.
 */
export function normalizeEffectEnvelope(value: unknown): EffectEnvelope {
  let candidate = value;
  if (typeof candidate === "string") {
    try {
      candidate = JSON.parse(candidate);
    } catch {
      return { schema_version: 2, blocks: [] };
    }
  }
  if (Array.isArray(candidate)) {
    return { schema_version: 2, blocks: candidate as EffectBlock[] };
  }
  if (candidate && typeof candidate === "object" && Array.isArray((candidate as Partial<EffectEnvelope>).blocks)) {
    return { schema_version: 2, blocks: (candidate as Partial<EffectEnvelope>).blocks as EffectBlock[] };
  }
  return { schema_version: 2, blocks: [] };
}

export type EffectLike = EffectEnvelope | EffectBlock[] | { effects?: EffectEnvelope | EffectBlock[] | null } | null | undefined;

export function effectEnvelope(value: EffectLike): EffectEnvelope {
  const candidate = value && typeof value === "object" && "effects" in value
    ? (value as { effects?: EffectEnvelope | EffectBlock[] | null }).effects
    : value;
  if (Array.isArray(candidate)) {
    return { blocks: candidate };
  }
  if (candidate && typeof candidate === "object") {
    if ("blocks" in candidate && Array.isArray((candidate as EffectEnvelope).blocks)) {
      return candidate as EffectEnvelope;
    }
    // Fallback: parse legacy/struct GameFX object if present
    const obj = candidate as Record<string, unknown>;
    const blocks: EffectBlock[] = [];
    const addOp = (id: string, code: string, val: unknown) => {
      blocks.push({
        id,
        trigger: "ALWAYS_ON",
        target: { selector: "SELF" },
        operations: [{ op: "UNIQUE", code, params: { value: val } }],
      });
    };
    if (obj.stat_mods && typeof obj.stat_mods === "object" && Object.keys(obj.stat_mods).length > 0) {
      addOp("STAT_MODS", "STAT_MODS", obj.stat_mods);
    }
    if (obj.dao_mark && typeof obj.dao_mark === "object" && Object.keys(obj.dao_mark).length > 0) {
      addOp("DAO_MARK_MODIFIER", "DAO_MARK_MODIFIER", obj.dao_mark);
    }
    if (obj.path_damage && typeof obj.path_damage === "object" && Object.keys(obj.path_damage).length > 0) {
      addOp("PATH_DAMAGE_MODIFIER", "PATH_DAMAGE_MODIFIER", obj.path_damage);
    }
    if (obj.path_resistance && typeof obj.path_resistance === "object" && Object.keys(obj.path_resistance).length > 0) {
      addOp("PATH_RESISTANCE_MODIFIER", "PATH_RESISTANCE_MODIFIER", obj.path_resistance);
    }
    if (typeof obj.refine === "number" && obj.refine !== 0) addOp("REFINE_SUCCESS", "REFINE_SUCCESS", obj.refine);
    if (typeof obj.luck === "number" && obj.luck !== 0) addOp("LUCK", "LUCK", obj.luck);
    if (typeof obj.max_mana === "number" && obj.max_mana !== 0) addOp("MAX_MANA", "MAX_MANA", obj.max_mana);
    if (typeof obj.mana_regen === "number" && obj.mana_regen !== 0) addOp("MANA_REGEN", "MANA_REGEN", obj.mana_regen);
    if (typeof obj.cultivation_efficiency === "number" && obj.cultivation_efficiency !== 0) addOp("CULTIVATION_EFFICIENCY", "CULTIVATION_EFFICIENCY", obj.cultivation_efficiency);
    if (typeof obj.dao_mark_gain === "number" && obj.dao_mark_gain !== 0) addOp("DAO_MARK_GAIN", "DAO_MARK_GAIN", obj.dao_mark_gain);
    if (typeof obj.healing_received === "number" && obj.healing_received !== 0) addOp("HEALING_RECEIVED", "HEALING_RECEIVED", obj.healing_received);
    if (typeof obj.hp_regen === "number" && obj.hp_regen !== 0) addOp("HP_REGEN", "HP_REGEN", obj.hp_regen);
    if (typeof obj.mental_resistance === "number" && obj.mental_resistance !== 0) addOp("MENTAL_RESISTANCE", "MENTAL_RESISTANCE", obj.mental_resistance);
    if (typeof obj.soul_defense === "number" && obj.soul_defense !== 0) addOp("SOUL_DEFENSE", "SOUL_DEFENSE", obj.soul_defense);
    if (typeof obj.soul_regen === "number" && obj.soul_regen !== 0) addOp("SOUL_REGEN", "SOUL_REGEN", obj.soul_regen);
    if (typeof obj.tenacity === "number" && obj.tenacity !== 0) addOp("TENACITY", "TENACITY", obj.tenacity);
    if (typeof obj.cc_resist === "number" && obj.cc_resist !== 0) addOp("CC_RESIST", "CC_RESIST", obj.cc_resist);
    if (typeof obj.aperture_stress_bonus === "number" && obj.aperture_stress_bonus !== 0) addOp("APERTURE_STRESS", "APERTURE_STRESS", obj.aperture_stress_bonus);
    if (obj.non_conflicting_dao_marks) addOp("NON_CONFLICTING_DAO_MARKS", "NON_CONFLICTING_DAO_MARKS", true);
    if (obj.immortal) addOp("IMMORTAL", "IMMORTAL", true);
    if (obj.disable_hp_regen) addOp("DISABLE_HP_REGEN", "DISABLE_HP_REGEN", true);
    if (obj.disable_mana_regen) addOp("DISABLE_MANA_REGEN", "DISABLE_MANA_REGEN", true);
    if (obj.disable_resource_regen) addOp("DISABLE_RESOURCE_REGEN", "DISABLE_RESOURCE_REGEN", true);
    if (obj.disable_soul_regen) addOp("DISABLE_SOUL_REGEN", "DISABLE_SOUL_REGEN", true);

    if (blocks.length > 0) {
      return { blocks };
    }
  }
  return EMPTY_EFFECT_ENVELOPE;
}

export function blocksByTrigger(value: EffectLike, ...triggers: string[]): EffectBlock[] {
  const wanted = new Set(triggers.map((v) => v.toUpperCase()));
  return effectEnvelope(value).blocks.filter((block) => wanted.has(block.trigger.toUpperCase()));
}

export interface OperationQuery {
  trigger?: string | string[];
  op?: string;
  code?: string;
  stat?: string;
  status?: string;
  subType?: string;
}

export function operationsBy(value: EffectLike, query: OperationQuery = {}): EffectOperation[] {
  const triggers = query.trigger ? (Array.isArray(query.trigger) ? query.trigger : [query.trigger]) : null;
  const blocks = triggers ? blocksByTrigger(value, ...triggers) : effectEnvelope(value).blocks;
  const eq = (actual: string | undefined, expected: string | undefined) =>
    !expected || actual?.toUpperCase() === expected.toUpperCase();
  return blocks.flatMap((block) => block.operations || []).filter((operation) =>
    eq(operation.op, query.op) && eq(operation.code, query.code) && eq(operation.stat, query.stat) &&
    eq(operation.status_id, query.status) && eq(operation.sub_type, query.subType));
}

export const hasOperation = (value: EffectLike, query: OperationQuery): boolean => operationsBy(value, query).length > 0;

export interface NumericEffectValue { value: number; unit: string }
export function numericValue(operation: EffectOperation | undefined, fallback = 0): NumericEffectValue {
  return { value: typeof operation?.value === "number" ? operation.value : fallback, unit: operation?.unit || "FLAT" };
}
export function sumNumericValues(value: EffectLike, query: OperationQuery): NumericEffectValue[] {
  return operationsBy(value, query).map((operation) => numericValue(operation));
}

export function hasCapability(value: EffectLike, code: string): boolean {
  return hasOperation(value, { trigger: "CAPABILITY", op: "UNIQUE", code });
}

export function statModifier(value: EffectLike, stat: string, trigger: string | string[] = "ALWAYS_ON"): number {
  return operationsBy(value, { trigger, op: "STAT_MOD", stat })
    .reduce((sum, operation) => sum + numericValue(operation).value, 0);
}

export const investigation = (value: EffectLike) => statModifier(value, "INVESTIGATION");
export const disguiseLevel = (value: EffectLike) => statModifier(value, "DISGUISE");

export type CombatCategory = "attack" | "defense" | "support" | "other";
export function combatCategory(value: EffectLike): CombatCategory {
  if (hasOperation(value, { op: "DAMAGE" }) || hasOperation(value, { op: "DIRECT_HP" })) return "attack";
  if (hasOperation(value, { op: "SHIELD" }) || hasOperation(value, { op: "BARRIER" }) || hasOperation(value, { op: "STAT_MOD", stat: "DEFENSE" })) return "defense";
  if (hasOperation(value, { op: "HEAL" }) || hasOperation(value, { op: "CLEANSE" }) || hasOperation(value, { op: "STATUS" })) return "support";
  return "other";
}
export function isCombatEligible(value: EffectLike): boolean {
  return blocksByTrigger(value, "ON_ACTION", "ON_HIT", "BEFORE_HIT", "ON_TURN_START", "ON_DEATH", "COMBAT_PASSIVE", "ON_BATTLE_START", "BEFORE_ENEMY_ATTACK", "ON_CAST").length > 0;
}

export interface GrowthConfig { max_attack: number; max_defense: number; max_hp: number; max_speed: number; hours_to_max: number }
export function growthConfig(value: EffectLike): GrowthConfig | null {
  const cap = (stat: string) => operationsBy(value, { op: "STAT_MOD", stat, subType: "GROWTH_CAP" })[0]?.value || 0;
  const duration = operationsBy(value, { op: "UNIQUE", code: "GROWTH_DURATION_HOURS" })[0];
  const hours = typeof duration?.params?.value === "number" ? duration.params.value : numericValue(duration).value;
  if (!hours && !cap("ATTACK") && !cap("DEFENSE") && !cap("HP") && !cap("SPEED")) return null;
  return { max_attack: cap("ATTACK"), max_defense: cap("DEFENSE"), max_hp: cap("HP"), max_speed: cap("SPEED"), hours_to_max: hours };
}

export const canCreatePortal = (value: EffectLike) =>
  hasCapability(value, "PORTAL_CREATE") || hasCapability(value, "IGNORE_MOVEMENT_RANGE");
export const usesRandomPortal = (value: EffectLike) => hasCapability(value, "RANDOM_PORTAL");
export const captureBonus = (value: EffectLike) => operationsBy(value, { code: "CAPTURE_BONUS" })
  .reduce((sum, op) => sum + (typeof op.params?.value === "number" ? op.params.value : numericValue(op).value), 0);
export const refinementTier = (value: EffectLike) => operationsBy(value, { op: "STAT_MOD", stat: "REFINE_MANA_TIER" })[0]?.value ?? 0;
export const hasRefinementMechanic = (value: EffectLike, code: string) => hasOperation(value, { op: "UNIQUE", code });
export const nativeNumericValue = (value: EffectLike, code: string) => operationsBy(value, { op: "UNIQUE", code })
  .reduce((sum, op) => sum + (typeof op.params?.value === "number" ? op.params.value : numericValue(op).value), 0);
export const manaRegenPerTick = (value: EffectLike) => operationsBy(value, { trigger: "ON_TICK", op: "UNIQUE", code: "MANA_REGEN_TICK" })
  .reduce((sum, op) => sum + (typeof op.params?.value === "number" ? op.params.value : numericValue(op, 0.5).value), 0);
export const restoreHunger = (value: EffectLike) => nativeNumericValue(value, "RESTORE_HUNGER");
