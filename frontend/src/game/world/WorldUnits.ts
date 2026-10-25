/** Shared world-unit conventions for the Phaser preview slice. */

/** Logical tile size in world units; one tile equals one unit while the MVP is grid based. */
export const WORLD_UNITS_PER_TILE = 1;

/** Render scale. Client-only: the server never receives pixels. */
export const PIXELS_PER_UNIT = 24;

/** Maximum distance the server accepts for a single click-to-move request. */
export const MAX_MOVE_TARGET_DISTANCE = 64;

export interface WorldPoint {
  x: number;
  y: number;
}

export function worldToPixels(point: WorldPoint): WorldPoint {
  return { x: point.x * PIXELS_PER_UNIT, y: point.y * PIXELS_PER_UNIT };
}

export function pixelsToWorld(point: WorldPoint): WorldPoint {
  return { x: point.x / PIXELS_PER_UNIT, y: point.y / PIXELS_PER_UNIT };
}

/** Mirrors the server's bounded click destination. */
export function clampMoveTarget(from: WorldPoint, target: WorldPoint): WorldPoint {
  const dx = target.x - from.x;
  const dy = target.y - from.y;
  const distance = Math.hypot(dx, dy);
  if (distance <= MAX_MOVE_TARGET_DISTANCE || distance === 0) return { ...target };
  const scale = MAX_MOVE_TARGET_DISTANCE / distance;
  return { x: from.x + dx * scale, y: from.y + dy * scale };
}

export function normalizeWorldVector(vector: WorldPoint): WorldPoint {
  const magnitude = Math.hypot(vector.x, vector.y);
  if (magnitude === 0) return { x: 0, y: 0 };
  const scale = magnitude > 1 ? 1 / magnitude : 1;
  return { x: vector.x * scale, y: vector.y * scale };
}
