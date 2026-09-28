import type { EffectBlock, EffectEnvelope, EffectTarget } from "./effect-blocks";

export function formatEffectTarget(target?: EffectTarget): string | null {
  if (!target?.selector) return null;
  const labels: Record<string, string> = { TARGET: "Mục tiêu", SINGLE: "Mục tiêu", ROW: "Hàng ngang", COL: "Hàng dọc", COLUMN: "Hàng dọc", ALL_ENEMIES: "Toàn bộ địch", ALL_ALLIES: "Toàn bộ đồng đội", ALL: "Toàn trận", SELF: target.bypass_frontline ? "Bản thân" : "" };
  const label = labels[target.selector.toUpperCase()] ?? target.selector.toUpperCase();
  if (!label && !target.bypass_frontline) return null;
  return target.bypass_frontline ? (label ? `${label} (Xuyên hàng)` : "Xuyên hàng") : label;
}

export function renderableEffectBlocks(effects?: EffectEnvelope | null): { blocks: EffectBlock[]; unsupportedSchema?: number } {
  if (!effects) return { blocks: [] };
  if (effects.schema_version !== 2) return { blocks: [], unsupportedSchema: effects.schema_version as number };
  return { blocks: effects.blocks };
}
