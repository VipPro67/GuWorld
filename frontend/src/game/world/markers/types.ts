import type { MapPortal } from "../MapLoader";
import type { GroundDrop } from "@/store/useGameStore";
import type { NearbyVeinInfo } from "@/store/useWorldStore";

/**
 * The single marker view-model shared by the Phaser world layer, the HUD radar
 * and the proximity panel. Every map structure is normalised into this shape so
 * the three renderers cannot disagree about what exists on the map.
 *
 * Payloads are deliberately narrowed to the fields the markers read. Full DTOs
 * from the store remain assignable, while callers (and tests) only need the
 * subset that actually reaches the screen.
 */
export type WorldMarkerKind = "sect" | "vein" | "teleport" | "portal" | "drop" | "resource";

export type WorldMarkerTone = "self" | "enemy" | "rare" | "neutral";

export interface WorldMarkerAction {
  type: "open-sect" | "attack-sect" | "harvest-vein" | "loot-drop" | "open-teleport" | "harvest-resource";
  targetId: string;
}

export interface WorldMarkerInteraction {
  radiusCells: number;
  label: string;
  action: WorldMarkerAction;
}

export interface WorldMarkerSect {
  level: number;
  mainHallLevel: number;
  shieldUntil: number | null;
  isMine: boolean;
}

export type WorldMarkerVein = Pick<NearbyVeinInfo, "id" | "vein_type" | "rarity" | "accumulated_stones">;

export type WorldMarkerDrop = Pick<GroundDrop, "id" | "x" | "y" | "killer_name"> & { region_id?: string };

export type WorldMarkerPortal = Pick<MapPortal, "x" | "y" | "name" | "targetZoneId">;

/** Narrowed public-mine payload: only what the map layer and radar render. */
export interface WorldMarkerResource {
  id: string;
  region_id?: string | null;
  x: number;
  y: number;
  name: string;
  grade: string;
  point_class: string;
  state: string;
  vitality: number;
  stock_total: number;
  stock_cap: number;
  next_item_id?: string;
}

export interface WorldMarkerCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
  region_id?: string;
  vein?: WorldMarkerVein;
}

export interface WorldMarker {
  /** Stable per-kind identity; coordinates are local cells of `region`. */
  id: string;
  kind: WorldMarkerKind;
  region: string;
  x: number;
  y: number;
  name: string;
  tone: WorldMarkerTone;
  /** Territory footprint in cells (sects only). */
  radiusCells?: number;
  interaction?: WorldMarkerInteraction;
  vein?: WorldMarkerVein;
  sect?: WorldMarkerSect;
  teleport?: { tier: string; accessPolicy: string };
  drop?: WorldMarkerDrop;
  portal?: WorldMarkerPortal;
  resource?: WorldMarkerResource;
}

/** Interaction ranges live here so the map and HUD cannot drift apart. */
export const MARKER_INTERACTION_RADIUS: Record<WorldMarkerKind, number> = {
  sect: 4.0,
  vein: 3.5,
  teleport: 3.0,
  drop: 3.2,
  portal: 0,
  // Backend worldHarvestRange (Chebyshev ≤ 6 cells).
  resource: 6,
};
