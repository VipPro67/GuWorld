import { type EffectBlock, type EffectLike, type EffectTarget, effectEnvelope } from "./effect-blocks.ts";

export function formatEffectTarget(target?: EffectTarget): string | null {
  if (!target?.selector) return null;
  const labels: Record<string, string> = { TARGET: "Mục tiêu", SINGLE: "Mục tiêu", ROW: "Hàng ngang", COL: "Hàng dọc", COLUMN: "Hàng dọc", ALL_ENEMIES: "Toàn bộ địch", ALL_ALLIES: "Toàn bộ đồng đội", ALL: "Toàn trận", SELF: target.bypass_frontline ? "Bản thân" : "" };
  const label = labels[target.selector.toUpperCase()] ?? target.selector.toUpperCase();
  if (!label && !target.bypass_frontline) return null;
  return target.bypass_frontline ? (label ? `${label} (Xuyên hàng)` : "Xuyên hàng") : label;
}

export function renderableEffectBlocks(effects?: EffectLike | null): { blocks: EffectBlock[]; unsupportedSchema?: number } {
  if (!effects) return { blocks: [] };
  if (typeof effects === "object" && "schema_version" in effects && typeof effects.schema_version === "number" && effects.schema_version > 2) {
    return { blocks: [], unsupportedSchema: effects.schema_version };
  }
  return { blocks: effectEnvelope(effects).blocks };
}
