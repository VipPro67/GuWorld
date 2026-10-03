"use client";

import { create } from "zustand";
import { useGameStore, type Player, type GroundDrop } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";
import { wsManager } from "@/lib/websocket";

export interface WorldObject {
  id?: string;
  region_id: string;
  x: number;
  y: number;
  cell_type: string;
  content_id: string;
  rank?: number;
  respawn_seconds?: number;
  cleared?: boolean;
  dropData?: GroundDrop;
}

export interface NearbyCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
  rank?: number;
  sub_rank?: number;
  region_id?: string;
}

type CellCacheState = {
  cellChunks: Record<string, CellChunk>;
  nearbyCells: NearbyCell[];
  nearbyWindow: { x: number; y: number; r: number } | null;
  cellRegion: string;
};

export type CellChunk = {
  cells: Record<string, NearbyCell>;
  touchedAt: number;
  version?: number;
};

export const CELL_CHUNK_SIZE = 64;
const MAX_CELL_CHUNKS = 512;

const currentCellRegion = () => useGameStore.getState().player?.current_region ?? "";
const chunkKey = (region: string, x: number, y: number) =>
  `${region}:${Math.floor(x / CELL_CHUNK_SIZE)},${Math.floor(y / CELL_CHUNK_SIZE)}`;
const cellKey = (x: number, y: number) => `${x},${y}`;

function visibleCells(chunks: Record<string, CellChunk>, region: string, w: CellCacheState["nearbyWindow"]): NearbyCell[] {
  if (!w) return [];
  const visible: NearbyCell[] = [];
  const minChunkX = Math.floor((w.x - w.r) / CELL_CHUNK_SIZE);
  const maxChunkX = Math.floor((w.x + w.r) / CELL_CHUNK_SIZE);
  const minChunkY = Math.floor((w.y - w.r) / CELL_CHUNK_SIZE);
  const maxChunkY = Math.floor((w.y + w.r) / CELL_CHUNK_SIZE);
  for (let cx = minChunkX; cx <= maxChunkX; cx++) {
    for (let cy = minChunkY; cy <= maxChunkY; cy++) {
      const chunk = chunks[`${region}:${cx},${cy}`];
      if (!chunk) continue;
      for (const cell of Object.values(chunk.cells)) {
        if (cell.x >= w.x - w.r && cell.x <= w.x + w.r && cell.y >= w.y - w.r && cell.y <= w.y + w.r) visible.push(cell);
      }
    }
  }
  return visible;
}

// Copy only chunks touched by this response. A flat cache used to clone and scan
// up to 150k entries for every websocket delta.
function mergeCellCache(
  s: CellCacheState,
  cells: NearbyCell[],
  version?: number,
  coverage?: { minX: number; minY: number; maxX: number; maxY: number; region: string },
): Partial<CellCacheState> {
  const region = coverage?.region ?? currentCellRegion();
  const chunks = s.cellRegion === region ? { ...s.cellChunks } : {};
  const changed = new Map<string, CellChunk>();
  const now = Date.now();
  // A rectangle response is an authoritative snapshot of its covered area.
  // Remove cached cells omitted by the server so missed realtime events heal on
  // the next reconciliation poll.
  if (coverage) {
    const minChunkX = Math.floor(coverage.minX / CELL_CHUNK_SIZE);
    const maxChunkX = Math.floor(coverage.maxX / CELL_CHUNK_SIZE);
    const minChunkY = Math.floor(coverage.minY / CELL_CHUNK_SIZE);
    const maxChunkY = Math.floor(coverage.maxY / CELL_CHUNK_SIZE);
    for (let cx = minChunkX; cx <= maxChunkX; cx++) {
      for (let cy = minChunkY; cy <= maxChunkY; cy++) {
        const key = `${region}:${cx},${cy}`;
        const oldChunk = chunks[key];
        if (!oldChunk) continue;
        const nextCells = { ...oldChunk.cells };
        for (const [key, cell] of Object.entries(nextCells)) {
          if (cell.x >= coverage.minX && cell.x <= coverage.maxX && cell.y >= coverage.minY && cell.y <= coverage.maxY) delete nextCells[key];
        }
        chunks[key] = { ...oldChunk, cells: nextCells, touchedAt: now, version };
        changed.set(key, chunks[key]);
      }
    }
  }
  for (const c of cells) {
    const key = chunkKey(region, c.x, c.y);
    let chunk = changed.get(key);
    if (!chunk) {
      chunk = { cells: { ...(chunks[key]?.cells ?? {}) }, touchedAt: now, version };
      changed.set(key, chunk);
      chunks[key] = chunk;
    }
    chunk.cells[cellKey(c.x, c.y)] = c;
  }
  const keys = Object.keys(chunks);
  if (keys.length > MAX_CELL_CHUNKS) {
    keys.sort((a, b) => chunks[a].touchedAt - chunks[b].touchedAt);
    for (let i = 0; i < keys.length - MAX_CELL_CHUNKS; i++) delete chunks[keys[i]];
  }
  return { cellChunks: chunks, cellRegion: region, nearbyCells: visibleCells(chunks, region, s.nearbyWindow) };
}

export interface NearbyPlayer {
  id: string;
  username: string;
  avatar_url: string;
  x: number;
  y: number;
  region_id?: string;
}

export interface WorldPlayer {
  x: number;
  y: number;
  current_region: string;
}

export interface TeleportResponse {
  player: Player;
  x: number;
  y: number;
  region: string;
  cost: number;
  currency: string;
  message: string;
}

export interface SkillFxEvent {
  id: string;
  type: string;
  value?: number;
  text: string;
  x: number;
  y: number;
  timestamp: number;
}

export interface SelectedTarget {
  x: number;
  y: number;
  cellType: string;
  contentId: string;
  name: string;
  rank?: number;
  hp: number;
  maxHp: number;
}

import { BACKEND_URL } from "@/lib/config";
import { authFetch } from "@/lib/auth";

interface WorldStoreState {
  objects: WorldObject[];
  nearbyCells: NearbyCell[];
  cellChunks: Record<string, CellChunk>;
  cellRegion: string;
  nearbyWindow: { x: number; y: number; r: number } | null;
  playerPositions: Record<string, NearbyPlayer & { ts: number }>;
  nearbyPlayers: NearbyPlayer[];
  exploredSubZones: Set<string>;
  movementRange: number;
  investigationRange: number;
  selectedObject: WorldObject | null;
  selectedTarget: SelectedTarget | null;
  targetingGu: { guInstanceId: string; guId: string; name: string } | null;
  skillFxEvents: SkillFxEvent[];
  isMoving: boolean;
  loading: boolean;
  error: string | null;
  worldLogs: { id: string; text: string; timestamp: number }[];
  cameraCenterRequest: number;
  liveX: number;
  liveY: number;
  cameraScale: number;
  cameraX: number;
  cameraY: number;
  fetchObjects: () => Promise<void>;
  fetchNearbyCells: (x: number, y: number, r: number) => Promise<boolean>;
  fetchCellsRect: (minX: number, minY: number, maxX: number, maxY: number) => Promise<boolean>;
  getCachedCell: (x: number, y: number) => NearbyCell | undefined;
  invalidateCell: (x: number, y: number) => void;
  mergeWSCells: (cells: NearbyCell[]) => void;
  fetchNearbyPlayers: (x: number, y: number, r: number) => Promise<void>;
  upsertPlayerPosition: (p: NearbyPlayer) => void;
  pruneStalePlayers: () => void;
  movePath: (x: number, y: number) => Promise<{
    player: Partial<Player> & Pick<Player, "x" | "y">;
    synced_party_members?: Array<{ id: string; username: string; avatar_url: string; region: string; x: number; y: number }>;
    log: string;
    combat_triggered: boolean;
    reached: boolean;
    skip_result?: unknown;
  } | null>;
  useGuOnMap: (guInstanceId: string, targetX: number, targetY: number) => Promise<{ message: string; logs: string[]; effect_type?: string; effect_value?: number; damage_dealt?: number; target_defeated?: boolean; target_hp?: number; target_max_hp?: number } | null>;
  setTargetingGu: (gu: { guInstanceId: string; guId: string; name: string } | null) => void;
  selectTarget: (target: SelectedTarget | null) => void;
  triggerSkillFx: (type: string, value: number, text: string, x: number, y: number) => void;
  clearSkillFx: (id: string) => void;
  setExploredSubZones: (keys: string[]) => void;
  setMovementRange: (r: number) => void;
  setInvestigationRange: (r: number) => void;
  setMoving: (v: boolean) => void;
  selectObject: (obj: WorldObject | null) => void;
  requestCenterCamera: () => void;
  setLivePosition: (x: number, y: number) => void;
  setCamera: (scale: number, x: number, y: number) => void;
  addWorldLog: (text: string) => void;
}

export const useWorldStore = create<WorldStoreState>((set, get) => ({
  objects: [],
  nearbyCells: [],
  cellChunks: {},
  cellRegion: "",
  nearbyWindow: null,
  playerPositions: {},
  nearbyPlayers: [],
  exploredSubZones: new Set<string>(),
  movementRange: 1,
  investigationRange: 10,
  selectedObject: null,
  selectedTarget: null,
  targetingGu: null,
  skillFxEvents: [],
  isMoving: false,
  loading: false,
  error: null,
  worldLogs: [],
  cameraCenterRequest: 0,
  liveX: 0,
  liveY: 0,
  cameraScale: 0,
  cameraX: 0,
  cameraY: 0,

  setTargetingGu: (gu) => set({ targetingGu: gu }),
  selectTarget: (target) => set({ selectedTarget: target }),
  triggerSkillFx: (type, value, text, x, y) =>
    set((s) => ({
      skillFxEvents: [
        ...s.skillFxEvents,
        { id: Math.random().toString(36).substring(2, 9), type, value, text, x, y, timestamp: Date.now() },
      ].slice(-20),
    })),
  clearSkillFx: (id) => set((s) => ({ skillFxEvents: s.skillFxEvents.filter((e) => e.id !== id) })),

  requestCenterCamera: () => set((s) => ({ cameraCenterRequest: s.cameraCenterRequest + 1 })),
  addWorldLog: (text: string) =>
    set((s) => ({
      worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text, timestamp: Date.now() }].slice(-50),
    })),

  fetchObjects: async () => {
    set({ loading: true, error: null });
    try {
      if (wsManager.isConnected()) {
        const data = await wsManager.request<WorldObject[]>("world.objects", {}, 15000);
        set({ objects: Array.isArray(data) ? data : [], loading: false });
        return;
      }
      const res = await authFetch(`${BACKEND_URL}/api/explore/objects`);
      if (!res.ok) throw new Error("Không thể tải các điểm trên bản đồ");
      const data = await res.json();
      set({ objects: Array.isArray(data) ? data : [], loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  movePath: async (x: number, y: number) => {
    // WorldCanvas owns the complete visual walk lifecycle. A transport request
    // finishing is not the same as the chained movement finishing.
    set({ error: null });
    try {
      if (wsManager.isConnected()) {
        return await wsManager.request<{
          player: Partial<Player> & Pick<Player, "x" | "y">;
          synced_party_members?: Array<{ id: string; username: string; avatar_url: string; region: string; x: number; y: number }>;
          log: string;
          combat_triggered: boolean;
          reached: boolean;
          skip_result?: unknown;
        }>("world.move_path", { x: Math.round(x), y: Math.round(y) }, 60000);
      }
      const res = await authFetch(`${BACKEND_URL}/api/explore/move-path`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ x: Math.round(x), y: Math.round(y) }),
      });
      const body = await res.text();
      if (!res.ok) {
        let msg = `Di chuyển thất bại: ${res.status}`;
        try {
          const errData = JSON.parse(body);
          if (errData.error) msg = errData.error;
        } catch {
          /* ignore */
        }
        useUIStore.getState().addToast(msg, "error");
        set((s) => ({
          error: null,
          worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
        }));
        return null;
      }
      const data = JSON.parse(body);
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      useGameStore.getState().addToast(msg, "error");
      set((s) => ({
        error: null,
        worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
      }));
      return null;
    }
  },

  useGuOnMap: async (guInstanceId: string, targetX: number, targetY: number) => {
    set({ loading: true, error: null });
    try {
      const curTarget = get().selectedTarget;
      const tx = targetX || (curTarget ? curTarget.x : 0);
      const ty = targetY || (curTarget ? curTarget.y : 0);

      const res = await authFetch(`${BACKEND_URL}/api/explore/use_gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: guInstanceId, target_x: tx, target_y: ty }),
      });
      const body = await res.text();
      let data;
      try {
        data = JSON.parse(body);
      } catch {
        throw new Error(body || "Invalid response");
      }
      if (!res.ok) {
        const msg = data?.error || data?.message || "Failed to use Gu on map";
        useUIStore.getState().addToast(msg, "error");
        set((s) => ({
          error: null,
          loading: false,
          worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
        }));
        return null;
      }
      useGameStore.getState().setPlayerFromServer(data.player);
      if (data.player) {
        get().fetchNearbyCells(data.player.x, data.player.y, 300);
        get().fetchNearbyPlayers(data.player.x, data.player.y, 300);

        if (data.damage_dealt && data.damage_dealt > 0) {
          const fxX = data.target_x || tx || data.player.x;
          const fxY = data.target_y || ty || data.player.y;
          get().triggerSkillFx(
            "damage",
            data.damage_dealt,
            `-${Math.round(data.damage_dealt)}`,
            fxX,
            fxY
          );

          if (data.target_defeated) {
            useUIStore.getState().addToast(`Tiêu diệt thành công ${data.target_name || "quái vật"}!`, "success");
            set({ selectedTarget: null });
          } else if (data.target_hp !== undefined) {
            set((s) => ({
              selectedTarget: s.selectedTarget
                ? { ...s.selectedTarget, hp: data.target_hp, maxHp: data.target_max_hp || s.selectedTarget.maxHp }
                : null,
            }));
          }
        } else if (data.effect_type) {
          get().triggerSkillFx(
            data.effect_type,
            data.effect_value || 0,
            data.message || "Kích hoạt Cổ trùng!",
            data.player.x,
            data.player.y
          );
        }

        if (data.logs && Array.isArray(data.logs) && data.logs.length > 0) {
          for (const logText of data.logs) {
            get().addWorldLog(logText);
            useGameStore.getState().addLog(logText);
          }
        } else if (data.message) {
          get().addWorldLog(data.message);
          useGameStore.getState().addLog(data.message);
        }
      }
      set({ loading: false, targetingGu: null });
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Error";
      useGameStore.getState().addToast(msg, "error");
      set((s) => ({
        error: null,
        loading: false,
        worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
      }));
      return null;
    }
  },

  teleportToRegion: async (regionId: string) => {
    set({ loading: true, error: null });
    try {
      const res = await authFetch(`${BACKEND_URL}/api/explore/teleport`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ region_id: regionId }),
      });
      const body = await res.text();
      if (!res.ok) {
        let msg = `Di Chuyển thất bại: ${res.status}`;
        try {
          const errData = JSON.parse(body);
          if (errData.error) msg = errData.error;
        } catch {
          /* ignore */
        }
        useUIStore.getState().addToast(msg, "error");
        set((s) => ({
          loading: false,
          error: null,
          worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
        }));
        return null;
      }
      const data = JSON.parse(body);
      // Update local position immediately for smooth UX
      if (data.x !== undefined && data.y !== undefined) {
        set({ liveX: data.x, liveY: data.y });
        // Also update game store player position
        if (typeof window !== "undefined") {
          const { useGameStore } = await import("@/store/useGameStore");
          const currentPlayer = useGameStore.getState().player;
          if (currentPlayer) {
            useGameStore.getState().setPlayerFromServer({
              ...currentPlayer,
              x: data.x,
              y: data.y,
              current_region: data.region,
              primeval_stones: data.player?.primeval_stones ?? currentPlayer.primeval_stones,
              immortal_stones: data.player?.immortal_stones ?? currentPlayer.immortal_stones,
            });
          }
        }
      }
      // Refetch nearby cells for new position
      setTimeout(() => {
        useWorldStore.getState().fetchNearbyCells(data.x, data.y, 300);
        useWorldStore.getState().fetchNearbyPlayers(data.x, data.y, 300);
      }, 100);
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      useGameStore.getState().addToast(msg, "error");
      set((s) => ({
        error: null,
        loading: false,
        worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
      }));
      return null;
    } finally {
      set({ loading: false });
    }
  },

  setExploredSubZones: (keys) => set({ exploredSubZones: new Set<string>(keys) }),
  setMovementRange: (r) => set({ movementRange: r > 0 ? r : 1 }),
  setInvestigationRange: (r) => set({ investigationRange: Math.min(Math.max(r, 10), 25) }),
  setMoving: (v) => set({ isMoving: v }),
  selectObject: (obj) => set({ selectedObject: obj }),
  setLivePosition: (x, y) => set({ liveX: x, liveY: y }),
  setCamera: (scale, x, y) => set({ cameraScale: scale, cameraX: x, cameraY: y }),

  fetchCellsRect: async (minX, minY, maxX, maxY) => {
    const requestRegion = currentCellRegion();
    const rounded = {
      minX: Math.round(minX), minY: Math.round(minY), maxX: Math.round(maxX), maxY: Math.round(maxY),
    };
    try {
      let data: unknown;
      if (wsManager.isConnected()) {
        data = await wsManager.request<NearbyCell[]>("world.cells_rect", {
          min_x: rounded.minX, min_y: rounded.minY, max_x: rounded.maxX, max_y: rounded.maxY,
        }, 30000);
      } else {
        const url = `${BACKEND_URL}/api/explore/cells/rect?min_x=${rounded.minX}&min_y=${rounded.minY}&max_x=${rounded.maxX}&max_y=${rounded.maxY}`;
        const res = await authFetch(url);
        if (!res.ok) throw new Error("Không thể tải dữ liệu bản đồ");
        data = await res.json();
      }
      if (currentCellRegion() !== requestRegion) return false;
      const cells: NearbyCell[] = Array.isArray(data) ? data : [];
      set((s) => mergeCellCache(s, cells, undefined, { ...rounded, region: requestRegion }));
      return true;
    } catch (err) {
      console.error("fetchCellsRect error:", err instanceof Error ? err.message : err);
      return false;
    }
  },

  fetchNearbyCells: async (x, y, r) => {
    // Keep the request within the backend's 256x256 work bound while retaining
    // the player at the center. Larger areas are progressively loaded by the
    // existing 150-cell viewport prefetch tiles.
    const boundedRadius = Math.min(Math.max(Math.round(r), 1), 127);
    set({ nearbyWindow: { x, y, r: boundedRadius } });
    return get().fetchCellsRect(x - boundedRadius, y - boundedRadius, x + boundedRadius, y + boundedRadius);
  },

  getCachedCell: (x, y) => {
    const region = currentCellRegion();
    return get().cellChunks[chunkKey(region, x, y)]?.cells[cellKey(x, y)];
  },

  invalidateCell: (x, y) => {
    set((s) => {
      const region = currentCellRegion();
      const key = chunkKey(region, x, y);
      const oldChunk = s.cellChunks[key];
      if (!oldChunk?.cells[cellKey(x, y)]) return {};
      const cells = { ...oldChunk.cells };
      delete cells[cellKey(x, y)];
      const cellChunks = { ...s.cellChunks, [key]: { ...oldChunk, cells, touchedAt: Date.now() } };
      return { cellChunks, nearbyCells: s.nearbyCells.filter((c) => c.x !== x || c.y !== y) };
    });
  },

  mergeWSCells: (cells: NearbyCell[]) => {
    set((s) => mergeCellCache(s, cells));
  },

  fetchNearbyPlayers: async (x, y, r) => {
    try {
      const url = `${BACKEND_URL}/api/explore/players?x=${Math.round(x)}&y=${Math.round(y)}&r=${Math.round(r)}`;
      const res = await authFetch(url);
      if (!res.ok) throw new Error("Không thể tải người chơi trên bản đồ");
      const data = await res.json();
      const players: NearbyPlayer[] = Array.isArray(data) ? data : [];
      set((s) => {
        const positions = { ...s.playerPositions };
        const now = Date.now();
        for (const p of players) {
          const prev = positions[p.id];
          if (!prev || prev.ts <= now) {
            positions[p.id] = { ...p, ts: now };
          } else {
            // Keep the freshest data but refresh presence so stale pruning keeps them.
            positions[p.id] = { ...p, ts: prev.ts };
          }
        }
        // Player presence has its own AOI. Do not overwrite nearbyWindow: that
        // window belongs to the bounded cell cache and may use a smaller radius.
        const w = { x, y, r };
        let nearbyPlayers = s.nearbyPlayers;
        nearbyPlayers = Object.keys(positions)
          .map((id) => positions[id])
          .filter((p2) => p2.x >= w.x - w.r && p2.x <= w.x + w.r && p2.y >= w.y - w.r && p2.y <= w.y + w.r);
        return { playerPositions: positions, nearbyPlayers };
      });
    } catch (err) {
      console.error("fetchNearbyPlayers error:", err instanceof Error ? err.message : err);
    }
  },

  upsertPlayerPosition: (p) => {
    set((s) => {
      const now = Date.now();
      const updated = { ...s.playerPositions, [p.id]: { ...p, ts: now } };
      const w = s.nearbyWindow;
      let nearby: NearbyPlayer[];
      if (s.nearbyPlayers.some((p2) => p2.id === p.id)) {
        nearby = s.nearbyPlayers.map((p2) => (p2.id === p.id ? { ...p2, ...p } : p2));
      } else if (!w || (p.x >= w.x - w.r && p.x <= w.x + w.r && p.y >= w.y - w.r && p.y <= w.y + w.r)) {
        nearby = [...s.nearbyPlayers, p];
      } else {
        nearby = s.nearbyPlayers;
      }
      return { playerPositions: updated, nearbyPlayers: nearby };
    });
  },

  pruneStalePlayers: () => {
    set((s) => {
      const cutoff = Date.now() - 60_000;
      let changed = false;
      const updated: Record<string, NearbyPlayer & { ts: number }> = {};
      for (const id of Object.keys(s.playerPositions)) {
        if (s.playerPositions[id].ts >= cutoff) {
          updated[id] = s.playerPositions[id];
        } else {
          changed = true;
        }
      }
      if (!changed) return {};
      return { playerPositions: updated, nearbyPlayers: s.nearbyPlayers.filter((p) => !!updated[p.id]) };
    });
  },
}));
