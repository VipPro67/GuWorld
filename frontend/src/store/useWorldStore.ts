"use client";

import { create } from "zustand";
import { BACKEND_URL } from "@/lib/config";
import { authFetch } from "@/lib/auth";
import { useGameStore, type ApertureResourcePoint, type Region } from "@/store/useGameStore";

export interface NearbyVeinInfo {
  id: string;
  region_id: string;
  x: number;
  y: number;
  vein_type: string;
  rarity: string;
  accumulated_stones: number;
  capacity_stones: number;
  accumulated_qi: number;
  accumulated_chests: number;
  expires_at: string;
}

export interface NearbyCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
  rank?: number;
  sub_rank?: number;
  region_id?: string;
  vein?: NearbyVeinInfo;
}

/**
 * Static structure marker from GET /api/world/map-objects. The backend folds
 * sects and teleport arrays into one contract so the map layer, radar and
 * proximity panel share a single source of truth.
 */
export interface MapMarkerView {
  id: string;
  kind: string;
  region_id: string;
  x: number;
  y: number;
  name: string;
  detail?: Record<string, unknown>;
}

export interface WorldChangedEvent {
	event_id?: string;
	entity_type?: string;
	action?: string;
	scope?: { region_id?: string; world_map_id?: string; map_instance_id?: string };
  epoch?: string;
  revision?: number;
  data?: { actor?: WorldMonsterActor; [key: string]: unknown };
  type?: string;
  region_id?: string;
  x?: number;
  y?: number;
}

export interface WorldMonsterActor {
  instance_id: string;
  owner_player_id?: string;
  monster_template_id: string;
  region_id: string;
  x: number;
  y: number;
  hp: number;
  max_hp: number;
  is_boss?: boolean;
  revision?: number;
}

interface WorldStoreState {
  nearbyCells: NearbyCell[];
  zoneMarkers: MapMarkerView[];
  zoneMarkersRegion: string | null;
  /** Public mines of the current region (WORLD-scope resource points). */
  resourcePoints: ApertureResourcePoint[];
  resourcePointsRegion: string | null;
  worldMonsters: WorldMonsterActor[];
  worldRevisions: Record<string, { epoch: string; revision: number }>;
  mapRevision: number;
  /** Last rectangle queried for cells; world events reuse it to refetch. */
  lastCellsViewport: { x: number; y: number; radius: number } | null;
  loading: boolean;
  error: string | null;

  fetchNearbyCells: (x: number, y: number, radius: number) => Promise<boolean>;
  fetchZoneMarkers: (regionId: string) => Promise<boolean>;
  fetchWorldResourcePoints: (regionId: string) => Promise<boolean>;
  fetchWorldMonsters: () => Promise<boolean>;
  engageAscensionWorldMonster: (actor: WorldMonsterActor) => Promise<boolean>;
  collectWorldResourcePoint: (pointId: string) => Promise<boolean>;
  transplantWorldResourcePoint: (pointId: string) => Promise<boolean>;
  harvestVein: (veinId: string) => Promise<boolean>;
  /** Event-driven replacement for the old 30s cell/marker polling. */
  applyWorldCellEvent: (event: WorldChangedEvent) => void;
  /** Event-driven replacement for the old 10s/30s resource-point polling. */
  applyWorldPointsEvent: (event: WorldChangedEvent) => void;
  applyWorldChangeEvent: (event: WorldChangedEvent) => void;
  resyncWorldScope: (scope?: WorldChangedEvent["scope"]) => Promise<void>;
  checkWorldRevisions: () => Promise<boolean>;
  recordWorldSnapshot: (response: Response, revisionsAtStart: Record<string, { epoch: string; revision: number }>) => boolean;
}

let cellsRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let pointsRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let cellsExpiryTimer: ReturnType<typeof setTimeout> | null = null;

// Veins disappear when expires_at passes. The old 30s cell poll used to drop
// them; a one-shot timer keeps the map correct without polling.
function scheduleCellsExpiryRefresh(
  get: () => WorldStoreState,
  cells: NearbyCell[],
  viewport: { x: number; y: number; radius: number },
) {
  if (cellsExpiryTimer) {
    clearTimeout(cellsExpiryTimer);
    cellsExpiryTimer = null;
  }
  let nextExpiry = Number.POSITIVE_INFINITY;
  for (const cell of cells) {
    if (!cell.vein?.expires_at) continue;
    const at = new Date(cell.vein.expires_at).getTime();
    if (Number.isFinite(at) && at > Date.now() && at < nextExpiry) nextExpiry = at;
  }
  if (!Number.isFinite(nextExpiry)) return;
  cellsExpiryTimer = setTimeout(() => {
    cellsExpiryTimer = null;
    void get().fetchNearbyCells(viewport.x, viewport.y, viewport.radius);
  }, Math.max(1000, nextExpiry - Date.now() + 1000));
}

function worldEventMatchesRegion(regionID: string | undefined): boolean {
  if (!regionID) return false;
  const state = useGameStore.getState();
  const currentRegion = state.player?.current_region;
  if (!currentRegion) return false;
  if (currentRegion === regionID) return true;
  for (let region: Region | undefined = state.regions.find((entry) => entry.id === currentRegion); region; ) {
    if (region.id === regionID) return true;
    region = region.parent_id ? state.regions.find((entry) => entry.id === region?.parent_id) : undefined;
  }
  return false;
}

function worldEventRegion(event: WorldChangedEvent): string | undefined {
  return event.scope?.region_id || event.region_id;
}

function worldScopeKey(scope: { region_id?: string; world_map_id?: string; map_instance_id?: string }): string {
  return JSON.stringify([scope.region_id || "", scope.world_map_id || "world_map", scope.map_instance_id || ""]);
}

function snapshotRevision(response: Response): { key: string; epoch: string; revision: number; consistent: boolean } | null {
  const revision = Number(response.headers.get("X-World-Revision"));
  const region = response.headers.get("X-World-Revision-Region");
  if (!region || !Number.isSafeInteger(revision) || revision < 0) return null;
  const epoch = response.headers.get("X-World-Epoch") || "";
  const startRevisionHeader = response.headers.get("X-World-Start-Revision");
  const startRevision = startRevisionHeader === null ? revision : Number(startRevisionHeader);
  const startEpoch = response.headers.get("X-World-Start-Epoch") || epoch;
  return {
    key: worldScopeKey({
      region_id: region,
      world_map_id: response.headers.get("X-World-Revision-Map") || "world_map",
      map_instance_id: response.headers.get("X-World-Revision-Instance") || "",
    }),
    epoch,
    revision,
    consistent: startEpoch === epoch && startRevision === revision,
  };
}

interface WorldSequenceSnapshot {
  scope: { region_id?: string; world_map_id?: string; map_instance_id?: string };
  epoch: string;
  revision: number;
}

function worldSequenceSnapshot(value: unknown): WorldSequenceSnapshot[] | null {
  if (!Array.isArray(value)) return null;
  const sequences: WorldSequenceSnapshot[] = [];
  for (const item of value) {
    if (!item || typeof item !== "object") return null;
    const candidate = item as Record<string, unknown>;
    const scope = candidate.scope as WorldSequenceSnapshot["scope"] | undefined;
    if (!scope?.region_id || typeof candidate.epoch !== "string" || typeof candidate.revision !== "number" || !Number.isSafeInteger(candidate.revision)) return null;
    sequences.push({ scope, epoch: candidate.epoch, revision: candidate.revision });
  }
  return sequences;
}

const snapshotRetryTimers = new Set<ReturnType<typeof setTimeout>>();
let worldRevisionCheckInFlight = false;

function retryWorldSnapshot(request: () => void) {
  const timer = setTimeout(() => {
    snapshotRetryTimers.delete(timer);
    request();
  }, 50);
  snapshotRetryTimers.add(timer);
}

export const useWorldStore = create<WorldStoreState>((set, get) => ({
  nearbyCells: [],
  zoneMarkers: [],
  zoneMarkersRegion: null,
  resourcePoints: [],
  resourcePointsRegion: null,
  worldMonsters: [],
  worldRevisions: {},
  mapRevision: 0,
  lastCellsViewport: null,
  loading: false,
  error: null,

  fetchNearbyCells: async (x: number, y: number, radius: number) => {
    const revisionsAtStart = { ...get().worldRevisions };
    const boundedRadius = Math.min(Math.max(Math.round(radius), 1), 127);
    const minX = Math.round(x - boundedRadius);
    const minY = Math.round(y - boundedRadius);
    const maxX = Math.round(x + boundedRadius);
    const maxY = Math.round(y + boundedRadius);

    try {
      const res = await authFetch(`${BACKEND_URL}/api/explore/cells/rect?min_x=${minX}&min_y=${minY}&max_x=${maxX}&max_y=${maxY}`);
      if (!res.ok) return false;
      const data: unknown = await res.json();
      if (!Array.isArray(data)) return false;
      const marker = snapshotRevision(res);
      if (marker && !marker.consistent) {
        retryWorldSnapshot(() => { void get().fetchNearbyCells(x, y, radius); });
        return true;
      }
      const currentMarker = marker ? get().worldRevisions[marker.key] : undefined;
      const startedMarker = marker ? revisionsAtStart[marker.key] : undefined;
      const stale = marker && currentMarker && (
        (currentMarker.epoch === marker.epoch && currentMarker.revision > marker.revision) ||
        ((!startedMarker || currentMarker.epoch !== startedMarker.epoch) && currentMarker.epoch !== marker.epoch)
      );
      if (stale) {
        retryWorldSnapshot(() => { void get().fetchNearbyCells(x, y, radius); });
        return true;
      }

      const cells: NearbyCell[] = data.map((item: Record<string, unknown>) => ({
        x: typeof item.x === "number" ? item.x : 0,
        y: typeof item.y === "number" ? item.y : 0,
        cell_type: typeof item.cell_type === "string" ? item.cell_type : (item.type as string) || "Empty",
        content: typeof item.content === "string" ? item.content : "",
        rank: typeof item.rank === "number" ? item.rank : undefined,
        sub_rank: typeof item.sub_rank === "number" ? item.sub_rank : undefined,
        region_id: typeof item.region === "string" ? item.region : (item.region_id as string) || undefined,
        vein: item.vein as NearbyVeinInfo | undefined,
      }));

      set((state) => ({
        nearbyCells: cells,
        lastCellsViewport: { x: Math.round(x), y: Math.round(y), radius: boundedRadius },
        ...(marker ? { worldRevisions: { ...state.worldRevisions, [marker.key]: { epoch: marker.epoch, revision: marker.revision } } } : {}),
      }));
      scheduleCellsExpiryRefresh(get, cells, { x: Math.round(x), y: Math.round(y), radius: boundedRadius });
      return true;
    } catch (err) {
      console.warn("fetchNearbyCells failed:", err);
      return false;
    }
  },

  fetchZoneMarkers: async (regionId: string) => {
    if (!regionId) return false;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/world/map-objects?region=${encodeURIComponent(regionId)}`);
      if (!res.ok) throw new Error("Could not fetch zone markers");
      const data = await res.json();
      if (!Array.isArray(data)) return false;
      set({ zoneMarkers: data as MapMarkerView[], zoneMarkersRegion: regionId });
      return true;
    } catch (err) {
      console.warn("fetchZoneMarkers failed:", err);
      return false;
    }
  },

  fetchWorldResourcePoints: async (regionId: string) => {
    if (!regionId) return false;
    const revisionsAtStart = { ...get().worldRevisions };
    try {
      const res = await authFetch(`${BACKEND_URL}/api/immortal-aperture/world-points`);
      if (!res.ok) throw new Error("Could not fetch world resource points");
      const data = await res.json();
      if (!Array.isArray(data)) return false;
      const marker = snapshotRevision(res);
      if (marker && !marker.consistent) {
        retryWorldSnapshot(() => { void get().fetchWorldResourcePoints(regionId); });
        return true;
      }
      const currentMarker = marker ? get().worldRevisions[marker.key] : undefined;
      const startedMarker = marker ? revisionsAtStart[marker.key] : undefined;
      const stale = marker && currentMarker && (
        (currentMarker.epoch === marker.epoch && currentMarker.revision > marker.revision) ||
        ((!startedMarker || currentMarker.epoch !== startedMarker.epoch) && currentMarker.epoch !== marker.epoch)
      );
      if (stale) {
        retryWorldSnapshot(() => { void get().fetchWorldResourcePoints(regionId); });
        return true;
      }
      set((state) => ({
        resourcePoints: data as ApertureResourcePoint[], resourcePointsRegion: regionId,
        ...(marker ? { worldRevisions: { ...state.worldRevisions, [marker.key]: { epoch: marker.epoch, revision: marker.revision } } } : {}),
      }));
      return true;
    } catch (err) {
      console.warn("fetchWorldResourcePoints failed:", err);
      return false;
    }
  },

  fetchWorldMonsters: async () => {
    const revisionsAtStart = { ...get().worldRevisions };
    try {
      const res = await authFetch(`${BACKEND_URL}/api/world/monsters`);
      if (!res.ok) return false;
      const data = await res.json();
      const marker = snapshotRevision(res);
      const startSequences = worldSequenceSnapshot(data.world_revisions_start);
      const endSequences = worldSequenceSnapshot(data.world_revisions);
      const hasVector = startSequences !== null && endSequences !== null;
      if (!hasVector) {
        retryWorldSnapshot(() => { void get().fetchWorldMonsters(); });
        return true;
      }
      const startByScope = new Map<string, WorldSequenceSnapshot>((startSequences || []).map((sequence) => [worldScopeKey(sequence.scope), sequence] as const));
      const vectorConsistent = endSequences!.length === startSequences!.length && endSequences!.every((sequence) => {
        const start = startByScope.get(worldScopeKey(sequence.scope));
        return Boolean(start && start.epoch === sequence.epoch && start.revision === sequence.revision);
      });
      if ((marker && !marker.consistent) || !vectorConsistent) {
        retryWorldSnapshot(() => { void get().fetchWorldMonsters(); });
        return true;
      }
      const sequenceMarkers = endSequences!.map((sequence) => ({ key: worldScopeKey(sequence.scope), epoch: sequence.epoch, revision: sequence.revision }));
      const currentRevisions = get().worldRevisions;
      const stale = sequenceMarkers.some((sequence) => {
        const current = currentRevisions[sequence.key];
        const started = revisionsAtStart[sequence.key];
        if (!current) return false;
        if (current.epoch === sequence.epoch) return current.revision > sequence.revision;
        return (!started || current.epoch !== started.epoch) && current.epoch !== sequence.epoch;
      });
      if (stale) {
        retryWorldSnapshot(() => { void get().fetchWorldMonsters(); });
        return true;
      }
      set((state) => ({
        worldMonsters: Array.isArray(data.monsters) ? data.monsters as WorldMonsterActor[] : [],
        ...(sequenceMarkers.length > 0 ? {
          worldRevisions: {
            ...state.worldRevisions,
            ...Object.fromEntries(sequenceMarkers.map((sequence) => [sequence.key, { epoch: sequence.epoch, revision: sequence.revision }])),
          },
        } : {}),
      }));
      return true;
    } catch {
      return false;
    }
  },

  engageAscensionWorldMonster: async (actor) => {
    const game = useGameStore.getState();
    try {
      const res = await authFetch(`${BACKEND_URL}/api/cultivate/ascension/calamity/engage/${encodeURIComponent(actor.instance_id)}?region=${encodeURIComponent(actor.region_id)}`, { method: "POST" });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        game.addToast(data.error || "Không thể nghênh chiến Địa Tai.", "error");
        return false;
      }
      if (data.active_combat) game.setPlayerFromServer({ active_combat: data.active_combat });
      game.addToast(data.message || "Địa Tai đã nhận lời nghênh chiến.", "info");
      return true;
    } catch (err) {
      game.addToast(err instanceof Error ? err.message : "Không thể nghênh chiến Địa Tai.", "error");
      return false;
    }
  },

  collectWorldResourcePoint: async (pointId: string) => {
    const game = useGameStore.getState();
    try {
      const res = await authFetch(`${BACKEND_URL}/api/immortal-aperture/world-points/${pointId}/collect`, { method: "POST" });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        game.addToast(data.error || "Khai thác mỏ thất bại.", "error");
        return false;
      }
      game.applyInventoryFromResponse(data);
      game.addToast(data.message || "Khai thác mỏ thành công!", "success");
      const region = game.player?.current_region || get().resourcePointsRegion || "";
      if (region) void get().fetchWorldResourcePoints(region);
      return true;
    } catch (err) {
      game.addToast(err instanceof Error ? err.message : "Khai thác mỏ thất bại.", "error");
      return false;
    }
  },

  transplantWorldResourcePoint: async (pointId: string) => {
    const game = useGameStore.getState();
    try {
      const res = await authFetch(`${BACKEND_URL}/api/immortal-aperture/world-points/${pointId}/transplant`, { method: "POST" });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        game.addToast(data.error || "Di dời mỏ thất bại.", "error");
        return false;
      }
      // 202 = ritual started (Tiên Nguyên already burnt); 200 = mine moved in.
      game.addToast(data.message || (data.completed ? "Di dời mỏ vào Tiên Khiếu thành công!" : "Bắt đầu nghi thức di dời."), "success");
      // The ritual spends Tiên Nguyên immediately, so refresh the pool.
      void game.refreshPlayerSummary();
      const region = game.player?.current_region || get().resourcePointsRegion || "";
      if (region) void get().fetchWorldResourcePoints(region);
      return true;
    } catch (err) {
      game.addToast(err instanceof Error ? err.message : "Di dời mỏ thất bại.", "error");
      return false;
    }
  },

  harvestVein: async (veinId: string) => {
    try {
      const res = await authFetch(`${BACKEND_URL}/api/veins/${veinId}/harvest`, { method: "POST" });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        useGameStore.getState().addToast(data.error || "Thu hoạch Nguyên Tuyền thất bại.", "error");
        return false;
      }
      useGameStore.getState().applyInventoryFromResponse(data);
      useGameStore.getState().addToast(data.message || "Thu hoạch Nguyên Tuyền thành công!", "success");
      const p = useGameStore.getState().player;
      if (p) void get().fetchNearbyCells(p.x, p.y, 28);
      return true;
    } catch (err) {
      useGameStore.getState().addToast(err instanceof Error ? err.message : "Thu hoạch Nguyên Tuyền thất bại.", "error");
      return false;
    }
  },

  applyWorldCellEvent: (event: WorldChangedEvent) => {
    const regionID = worldEventRegion(event);
    if (!worldEventMatchesRegion(regionID)) return;
    const x = event.x;
    const y = event.y;
    if (event.action === "cleared" && typeof x === "number" && typeof y === "number") {
      set((state) => ({ nearbyCells: state.nearbyCells.filter((cell) => cell.x !== x || cell.y !== y) }));
    }
    const viewport = get().lastCellsViewport;
    if (cellsRefreshTimer) clearTimeout(cellsRefreshTimer);
    cellsRefreshTimer = setTimeout(() => {
      cellsRefreshTimer = null;
      if (viewport && event.action !== "cleared") {
        void get().fetchNearbyCells(viewport.x, viewport.y, viewport.radius);
      }
      if (regionID && ["portal_spawn", "portal_despawn", "portal_update"].includes(event.action || "")) {
        void get().fetchZoneMarkers(regionID);
      }
    }, 50);
  },

  applyWorldPointsEvent: (event: WorldChangedEvent) => {
    const eventRegion = worldEventRegion(event);
    if (!worldEventMatchesRegion(eventRegion)) return;
    const region = get().resourcePointsRegion || useGameStore.getState().player?.current_region || eventRegion || "";
    if (!region) return;
    if (pointsRefreshTimer) clearTimeout(pointsRefreshTimer);
    pointsRefreshTimer = setTimeout(() => {
      pointsRefreshTimer = null;
      void get().fetchWorldResourcePoints(region);
    }, 50);
  },

  applyWorldChangeEvent: (event: WorldChangedEvent) => {
    if (!event || !worldEventMatchesRegion(worldEventRegion(event))) return;
    const player = useGameStore.getState().player;
    const eventInstance = event.scope?.map_instance_id || "";
    if (player?.map_instance_id && player.map_instance_id !== eventInstance) return;
    if (!player?.map_instance_id && eventInstance) return;

    const scope = event.scope || { region_id: worldEventRegion(event) };
    const key = worldScopeKey(scope);
    if (typeof event.revision === "number" && Number.isSafeInteger(event.revision) && event.revision > 0) {
      const currentRevision = get().worldRevisions[key];
      if (currentRevision && currentRevision.epoch === (event.epoch || "") && event.revision <= currentRevision.revision) return;
      const hasGap = !currentRevision || currentRevision.epoch !== (event.epoch || "")
        ? event.revision > 1 || Boolean(currentRevision)
        : event.revision !== currentRevision.revision + 1;
      set((state) => ({ worldRevisions: { ...state.worldRevisions, [key]: { epoch: event.epoch || "", revision: event.revision! } } }));
      if (hasGap) {
        void get().resyncWorldScope(event.scope);
        return;
      }
    }

    switch (event.entity_type) {
      case "map_cell":
        set((state) => ({ mapRevision: state.mapRevision + 1 }));
        get().applyWorldCellEvent(event);
        break;
      case "ground_drop":
        set((state) => ({ mapRevision: state.mapRevision + 1 }));
        break;
      case "resource_point":
        set((state) => ({ mapRevision: state.mapRevision + 1 }));
        get().applyWorldPointsEvent(event);
        break;
      case "world_monster":
        {
          const actor = event.data?.actor;
          if (!actor || typeof actor.instance_id !== "string") {
            if (event.action === "killed" && typeof event.x === "number" && typeof event.y === "number") {
              const region = worldEventRegion(event);
              const matchingActorExists = get().worldMonsters.some((monster) =>
                monster.region_id === region && Math.round(monster.x) === event.x && Math.round(monster.y) === event.y,
              );
              set((state) => ({
                worldMonsters: state.worldMonsters.filter((monster) =>
                  monster.region_id !== region || Math.round(monster.x) !== event.x || Math.round(monster.y) !== event.y,
                ),
                mapRevision: state.mapRevision + 1,
              }));
              if (!matchingActorExists) void get().resyncWorldScope(event.scope);
              break;
            }
            void get().resyncWorldScope(event.scope);
            break;
          }
          const isGone = ["defeated", "killed", "despawned"].includes(event.action || "") || actor.hp <= 0;
          const player = useGameStore.getState().player;
          const inView = Boolean(player && worldEventMatchesRegion(actor.region_id) && Math.hypot(actor.x - player.x, actor.y - player.y) <= 25);
          set((state) => {
            const remaining = state.worldMonsters.filter((monster) => monster.instance_id !== actor.instance_id);
            return {
              worldMonsters: !isGone && inView ? [...remaining, actor] : remaining,
              ...(actor.is_boss ? { mapRevision: state.mapRevision + 1 } : {}),
            };
          });
        break;
        }
    }
  },

  resyncWorldScope: async (scope) => {
    const player = useGameStore.getState().player;
    if (!player || (scope && (scope.map_instance_id || "") !== (player.map_instance_id || ""))) return;
    const region = player.current_region;
    const viewport = get().lastCellsViewport;
    const requests: Promise<boolean>[] = [
      get().fetchWorldMonsters(),
      get().fetchWorldResourcePoints(region),
      get().fetchZoneMarkers(region),
    ];
    if (viewport) requests.push(get().fetchNearbyCells(viewport.x, viewport.y, viewport.radius));
    await Promise.all(requests);
    set((state) => ({ mapRevision: state.mapRevision + 1 }));
  },

  checkWorldRevisions: async () => {
    if (worldRevisionCheckInFlight) return false;
    const player = useGameStore.getState().player;
    if (!player) return false;
    worldRevisionCheckInFlight = true;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/world/revisions`);
      if (!response.ok) return false;
      const data = await response.json();
      const sequences = worldSequenceSnapshot(data.world_revisions);
      if (!sequences) return false;
      const currentRevisions = get().worldRevisions;
      const needsResync = sequences.some((sequence) => {
        const current = currentRevisions[worldScopeKey(sequence.scope)];
        if (!current) return sequence.revision > 0;
        return current.epoch !== sequence.epoch || current.revision !== sequence.revision;
      });
      if (needsResync) await get().resyncWorldScope();
      return true;
    } catch {
      return false;
    } finally {
      worldRevisionCheckInFlight = false;
    }
  },

  recordWorldSnapshot: (response, revisionsAtStart) => {
    const marker = snapshotRevision(response);
    if (!marker) return true;
    const current = get().worldRevisions[marker.key];
    const started = revisionsAtStart[marker.key];
    const scope = {
      region_id: response.headers.get("X-World-Revision-Region") || undefined,
      world_map_id: response.headers.get("X-World-Revision-Map") || undefined,
      map_instance_id: response.headers.get("X-World-Revision-Instance") || undefined,
    };
    const sequenceChangedDuringRequest = current && (!started || current.epoch !== started.epoch) && marker.epoch !== current.epoch;
    if (!marker.consistent || sequenceChangedDuringRequest || (current?.epoch === marker.epoch && current.revision > marker.revision)) {
      void get().resyncWorldScope(scope);
      return false;
    }
    set((state) => ({
      worldRevisions: { ...state.worldRevisions, [marker.key]: { epoch: marker.epoch, revision: marker.revision } },
    }));
    return true;
  },
}));
