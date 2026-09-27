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
  params?: Record<string, EffectScalar>;
}

export interface EffectBlock {
  id: string;
  trigger: string;
  target: EffectTarget;
  conditions?: EffectCondition[];
  operations: EffectOperation[];
}

export interface EffectEnvelope {
  schema_version: 2;
  blocks: EffectBlock[];
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
  schema_version: 2; triggers: string[]; targets: string[]; operations: EffectCatalogOperation[];
  stats: string[]; statuses: string[]; resources: string[]; units: string[];
  stack_policies: string[]; condition_fields: string[]; condition_operators: string[];
  native_codes: string[]; native_hooks: EffectNativeHook[];
}

export const EMPTY_EFFECT_ENVELOPE: EffectEnvelope = { schema_version: 2, blocks: [] };

type TransitionalEffects = Partial<EffectEnvelope> & { blocks?: EffectBlock[] | null };
export type EffectLike = EffectEnvelope | TransitionalEffects | { effects?: TransitionalEffects | null } | null | undefined;

export function effectEnvelope(value: EffectLike): EffectEnvelope {
  const candidate = value && typeof value === "object" && "effects" in value ? value.effects : value;
  // Transitional reads accept block-bearing rows created before schema_version
  // was added. Authoring and validation remain strict Block v2.
  return candidate && "blocks" in candidate && Array.isArray(candidate.blocks)
    ? { schema_version: 2, blocks: candidate.blocks }
    : EMPTY_EFFECT_ENVELOPE;
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
  const hours = operationsBy(value, { op: "UNIQUE", code: "GROWTH_DURATION_HOURS" })[0]?.value || 0;
  if (!hours && !cap("ATTACK") && !cap("DEFENSE") && !cap("HP") && !cap("SPEED")) return null;
  return { max_attack: cap("ATTACK"), max_defense: cap("DEFENSE"), max_hp: cap("HP"), max_speed: cap("SPEED"), hours_to_max: hours };
}

export const canCreatePortal = (value: EffectLike) => hasCapability(value, "PORTAL_CREATE");
export const usesRandomPortal = (value: EffectLike) => hasCapability(value, "RANDOM_PORTAL");
export const captureBonus = (value: EffectLike) => operationsBy(value, { code: "CAPTURE_BONUS" }).reduce((sum, op) => sum + numericValue(op).value, 0);
export const refinementTier = (value: EffectLike) => operationsBy(value, { op: "STAT_MOD", stat: "REFINE_MANA_TIER" })[0]?.value ?? 0;
export const hasRefinementMechanic = (value: EffectLike, code: string) => hasOperation(value, { op: "UNIQUE", code });
export const manaRegenPerTick = (value: EffectLike) => operationsBy(value, { trigger: "ON_TICK", op: "UNIQUE", code: "MANA_REGEN_TICK" })
  .reduce((sum, op) => sum + (typeof op.params?.value === "number" ? op.params.value : numericValue(op, 0.5).value), 0);
