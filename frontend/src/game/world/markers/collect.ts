import type { MapMarkerView } from "@/store/useWorldStore";
import {
  MARKER_INTERACTION_RADIUS,
  type WorldMarker,
  type WorldMarkerCell,
  type WorldMarkerDrop,
  type WorldMarkerKind,
  type WorldMarkerPortal,
  type WorldMarkerResource,
} from "./types.ts";

export interface MarkerSources {
  regionId: string;
  /** Nearby resolved cells (veins, authored teleport cells) from the rect feed. */
  cells?: WorldMarkerCell[];
  /** Static structures from GET /api/world/map-objects. */
  zoneMarkers?: MapMarkerView[];
  groundDrops?: WorldMarkerDrop[];
  /** Authored edge portals from the Tiled map. */
  mapPortals?: WorldMarkerPortal[];
  /** Public mines of the region (WORLD-scope resource points). */
  resourcePoints?: WorldMarkerResource[];
  mySectId?: string;
}

function sectMarker(marker: MapMarkerView, mySectId: string | undefined): WorldMarker {
  const detail = (marker.detail ?? {}) as {
    level?: number;
    buildings?: Record<string, number>;
    shield_until?: string | null;
    is_mine?: boolean;
  };
  const isMine = detail.is_mine ?? (Boolean(mySectId) && mySectId === marker.id);
  const level = detail.level ?? 1;
  const mainHallLevel = detail.buildings?.main_hall ?? level;
  const shieldUntil = detail.shield_until ? new Date(detail.shield_until).getTime() : null;
  return {
    id: marker.id,
    kind: "sect",
    region: marker.region_id,
    x: marker.x,
    y: marker.y,
    name: marker.name,
    tone: isMine ? "self" : "enemy",
    radiusCells: 5 + mainHallLevel,
    interaction: {
      radiusCells: MARKER_INTERACTION_RADIUS.sect,
      label: isMine ? "Mở tông môn" : "Tấn công",
      action: { type: isMine ? "open-sect" : "attack-sect", targetId: marker.id },
    },
    sect: { level, mainHallLevel, shieldUntil, isMine },
  };
}

function teleportMarker(marker: MapMarkerView): WorldMarker {
  const detail = (marker.detail ?? {}) as { tier?: string; access_policy?: string };
  return {
    id: marker.id,
    kind: "teleport",
    region: marker.region_id,
    x: marker.x,
    y: marker.y,
    name: marker.name || "Truyền Tống Trận",
    tone: "neutral",
    interaction: {
      radiusCells: MARKER_INTERACTION_RADIUS.teleport,
      label: "Danh bạ",
      action: { type: "open-teleport", targetId: marker.id },
    },
    teleport: { tier: detail.tier ?? "", accessPolicy: detail.access_policy ?? "" },
  };
}

/**
 * Normalises every map source into WorldMarker[]. Order matters: structured
 * markers are collected before loose cells so a dedupe by (kind, x, y) keeps the
 * richer payload.
 */
export function collectMarkers(sources: MarkerSources): WorldMarker[] {
  const region = sources.regionId;
  const out: WorldMarker[] = [];
  const seen = new Set<string>();
  const push = (marker: WorldMarker) => {
    const key = `${marker.kind}:${marker.x}:${marker.y}`;
    if (seen.has(key)) return;
    seen.add(key);
    out.push(marker);
  };

  for (const marker of sources.zoneMarkers ?? []) {
    if (marker.region_id && marker.region_id !== region) continue;
    if (marker.kind === "sect") push(sectMarker(marker, sources.mySectId));
    else if (marker.kind === "teleport") push(teleportMarker(marker));
  }

  for (const cell of sources.cells ?? []) {
    if (cell.region_id && cell.region_id !== region) continue;
    if (cell.cell_type === "PrimevalVein" || cell.vein) {
      const immortal = cell.vein?.vein_type === "immortal";
      push({
        id: cell.vein?.id || `vein:${cell.x}:${cell.y}`,
        kind: "vein",
        region: cell.region_id || region,
        x: cell.x,
        y: cell.y,
        name: immortal ? "Tiên Tuyền" : "Nguyên Tuyền",
        tone: immortal ? "rare" : "neutral",
        vein: cell.vein,
        interaction: {
          radiusCells: MARKER_INTERACTION_RADIUS.vein,
          label: "Thu hoạch",
          action: { type: "harvest-vein", targetId: cell.vein?.id || "" },
        },
      });
      continue;
    }
    if (cell.cell_type === "TeleportArray" || cell.cell_type === "ENTRY_PORTAL") {
      push({
        id: `cell:teleport:${cell.x}:${cell.y}`,
        kind: "teleport",
        region: cell.region_id || region,
        x: cell.x,
        y: cell.y,
        name: cell.content || "Truyền Tống Trận",
        tone: "neutral",
        interaction: {
          radiusCells: MARKER_INTERACTION_RADIUS.teleport,
          label: "Danh bạ",
          action: { type: "open-teleport", targetId: cell.content },
        },
      });
    }
  }

  for (const point of sources.resourcePoints ?? []) {
    if (point.region_id && point.region_id !== region) continue;
    const ready = point.state === "ACTIVE" && point.stock_total > 0;
    const famous = point.point_class === "FAMOUS";
    push({
      id: point.id,
      kind: "resource",
      region: point.region_id || region,
      x: point.x,
      y: point.y,
      name: point.name,
      tone: famous ? "rare" : "neutral",
      resource: point,
      interaction: {
        radiusCells: MARKER_INTERACTION_RADIUS.resource,
        label: ready ? "Khai thác" : "Chưa có tài nguyên",
        action: { type: "harvest-resource", targetId: point.id },
      },
    });
  }

  for (const drop of sources.groundDrops ?? []) {
    if (drop.region_id && drop.region_id !== region) continue;
    push({
      id: drop.id,
      kind: "drop",
      region: drop.region_id || region,
      x: drop.x,
      y: drop.y,
      name: drop.killer_name || "Rơi đồ",
      tone: "rare",
      drop,
      interaction: {
        radiusCells: MARKER_INTERACTION_RADIUS.drop,
        label: "Nhặt",
        action: { type: "loot-drop", targetId: drop.id },
      },
    });
  }

  for (const portal of sources.mapPortals ?? []) {
    push({
      id: `portal:${portal.x}:${portal.y}:${portal.targetZoneId}`,
      kind: "portal",
      region,
      x: portal.x,
      y: portal.y,
      name: portal.name?.trim() || portal.targetZoneId.replaceAll("_", " "),
      tone: "neutral",
      portal,
    });
  }

  return out;
}

/** Nearest marker of a kind whose interaction range covers the player. */
export function findNearestMarker(
  markers: WorldMarker[],
  position: { x: number; y: number },
  kind: WorldMarkerKind,
): WorldMarker | null {
  let best: WorldMarker | null = null;
  let bestDistance = Infinity;
  for (const marker of markers) {
    if (marker.kind !== kind || !marker.interaction) continue;
    const distance = Math.hypot(marker.x - position.x, marker.y - position.y);
    if (distance <= marker.interaction.radiusCells && distance < bestDistance) {
      best = marker;
      bestDistance = distance;
    }
  }
  return best;
}
