import type { ApertureResourcePoint } from "@/store/useGameStore";
import type { WorldMarkerResource } from "./types.ts";

export const WORLD_HARVEST_RANGE = 6;
export const TRANSPLANT_MANA: Record<string, number> = { LOW: 20, MID: 30, TOP: 50 };
export const TRANSPLANT_RITUAL_SECS: Record<string, number> = { LOW: 180, MID: 300, TOP: 480 };

export function transplantManaCost(grade: string, rank: number): number {
  const base = TRANSPLANT_MANA[grade] ?? TRANSPLANT_MANA.LOW;
  return Math.round(base * (rank > 6 ? 1 + 0.5 * (rank - 6) : 1));
}

export function transplantRitualSeconds(grade: string, rank: number): number {
  const base = TRANSPLANT_RITUAL_SECS[grade] ?? TRANSPLANT_RITUAL_SECS.LOW;
  if (rank <= 6) return base;
  return Math.max(60, Math.round(base * Math.max(0.35, 1 - 0.12 * (rank - 6))));
}

export function formatDuration(totalSeconds: number): string {
  const secs = Math.max(0, Math.ceil(totalSeconds));
  if (secs < 60) return `${secs} giây`;
  const mins = Math.floor(secs / 60);
  const rest = secs % 60;
  return rest === 0 ? `${mins} phút` : `${mins} phút ${rest} giây`;
}

export function formatClock(totalSeconds: number): string {
  const secs = Math.max(0, Math.ceil(totalSeconds));
  return `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, "0")}`;
}

/**
 * Narrows the API resource-point payload to the fields the map layer, radar and
 * proximity panel read. Shared so the Phaser scene and the HUD cannot disagree.
 */
export function toWorldMarkerResources(points: ApertureResourcePoint[]): WorldMarkerResource[] {
  return points.map((point) => ({
    id: point.id,
    region_id: point.region_id ?? null,
    x: point.x,
    y: point.y,
    name: point.name,
    grade: point.grade,
    point_class: point.point_class,
    state: point.state,
    vitality: point.vitality,
    stock_total: Object.values(point.stock || {}).reduce((sum, amount) => sum + amount, 0),
    stock_cap: point.stock_cap,
    next_item_id: point.next_item_id,
    transplant: point.transplant,
  }));
}
