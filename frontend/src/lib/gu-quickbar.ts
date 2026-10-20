import { canCreatePortal, operationsBy, usesRandomPortal, type EffectLike } from "./effect-blocks.ts";

/** Effect kinds the world-map quickbar can render. */
export type GuQuickbarEffectType =
  | "damage"
  | "heal"
  | "mana_regen"
  | "buff_atk"
  | "buff_def"
  | "buff_speed"
  | "shield"
  | "teleport"
  | "utility"
  | "fly";

export interface GuQuickbarEffect {
  type: GuQuickbarEffectType;
  label: string;
  isTargeted: boolean;
}

// Combat-only effects are cast from the combat screen, not the world-map bar.
export const COMBAT_QUICKBAR_EFFECT_TYPES = new Set<GuQuickbarEffectType>([
  "damage",
  "shield",
  "buff_atk",
  "buff_def",
  "buff_speed",
]);

export function isFlyingStatus(statusId?: string): boolean {
  const id = statusId?.toUpperCase();
  return id === "FLYING" || id === "PHI_HANH";
}

/**
 * Classify one Gu for the world-map quickbar.
 *
 * Flying is checked before damage/status so a Gu that also has combat effects
 * (Cốt Dực Cổ, Phi Yên Cổ, Phi Kiếm Dực Cổ…) still exposes its map-only
 * flight toggle instead of being hidden as a combat effect.
 */
export function classifyGuQuickbarEffect(effects: EffectLike): GuQuickbarEffect | null {
  const operations = operationsBy(effects, { trigger: ["ON_ACTION", "ON_USE", "CAPABILITY"] });
  if (operations.length === 0) {
    return null;
  }

  const first = operations[0];
  const damage = operations.find((operation) => operation.op === "DAMAGE");
  const flying = operations.find((operation) => operation.op === "STATUS" && isFlyingStatus(operation.status_id));
  const status = operations.find((operation) => operation.op === "STATUS");

  if (canCreatePortal(effects) || usesRandomPortal(effects)) {
    const isTargeted = canCreatePortal(effects);
    return {
      type: "teleport",
      isTargeted,
      label: isTargeted ? "Dịch Chuyển" : "Dịch Chuyển Ngẫu Nhiên",
    };
  }
  if (flying) {
    return { type: "fly", label: "Phi Hành", isTargeted: false };
  }
  if (damage) {
    return { type: "damage", label: `${damage.value || ""} Dmg`.trim(), isTargeted: true };
  }
  if (first.op === "HEAL") {
    return {
      type: first.resource === "MANA" ? "mana_regen" : "heal",
      label: `${first.resource === "MANA" ? "+" : "Hồi "}${first.value || 0} ${first.resource || "HP"}`,
      isTargeted: false,
    };
  }
  if (status?.status_id) {
    const id = status.status_id.toUpperCase();
    const type = id === "ATK_UP"
      ? "buff_atk"
      : id === "DEF_UP"
        ? "buff_def"
        : ["HASTE", "SPEED_UP"].includes(id)
          ? "buff_speed"
          : id === "REGEN"
            ? "heal"
            : "utility";
    return { type, label: `${id} ${status.value || ""}`.trim(), isTargeted: false };
  }
  if (first.op === "SHIELD" || first.op === "BARRIER") {
    return { type: "shield", label: `+${first.value || 0} Khiên`, isTargeted: false };
  }
  return {
    type: "utility",
    label: first.op === "DAMAGE" ? `${first.value || 0} Dmg` : "Kích hoạt",
    isTargeted: false,
  };
}
