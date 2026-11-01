import type { ApertureResourcePoint } from "@/store/useGameStore";
import type { WorldMarkerResource } from "./types.ts";

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
  }));
}
