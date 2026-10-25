import type { Vector2 } from "./Prediction.ts";

/** Hard snap above this distance (world units); smooth otherwise. */
export const CORRECTION_SNAP_UNITS = 2;
/** Fraction of the residual applied per rendered frame for soft corrections. */
export const CORRECTION_SMOOTHING = 0.25;

export interface Correction {
  position: Vector2;
  snapped: boolean;
  distance: number;
}

/**
 * Chooses between smoothing a small authoritative correction and hard snapping
 * after a large error (teleport, respawn, long packet loss). Values are world
 * units from the shared map scale, never legacy tile constants.
 */
export function resolveCorrection(
  rendered: Vector2,
  authoritative: Vector2,
  smoothing = CORRECTION_SMOOTHING,
  snapThreshold = CORRECTION_SNAP_UNITS,
): Correction {
  const distance = Math.hypot(authoritative.x - rendered.x, authoritative.y - rendered.y);
  if (distance > snapThreshold) {
    return { position: { ...authoritative }, snapped: true, distance };
  }
  const factor = Math.min(1, Math.max(0, smoothing));
  return {
    position: {
      x: rendered.x + (authoritative.x - rendered.x) * factor,
      y: rendered.y + (authoritative.y - rendered.y) * factor,
    },
    snapped: false,
    distance,
  };
}
