"use client";

import { create } from "zustand";
import { useGameStore, type Player } from "@/store/useGameStore";

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
}

export interface NearbyCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
  rank?: number;
  sub_rank?: number;
}

export interface NearbyPlayer {
  id: string;
  username: string;
  avatar_url: string;
  x: number;
  y: number;
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

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

const getCookie = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
};

const authHeaders = (method?: string): Record<string, string> => {
  const headers: Record<string, string> = {};
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }
  const csrfToken = getCookie("gu_csrf_token");
  if (csrfToken && method && method !== "GET" && method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  return headers;
};

interface WorldStoreState {
  objects: WorldObject[];
  nearbyCells: NearbyCell[];
  cellCache: Record<string, NearbyCell>;
  nearbyWindow: { x: number; y: number; r: number } | null;
  playerPositions: Record<string, NearbyPlayer & { ts: number }>;
  nearbyPlayers: NearbyPlayer[];
  exploredSubZones: Set<string>;
  movementRange: number;
  investigationRange: number;
  selectedObject: WorldObject | null;
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
  fetchNearbyCells: (x: number, y: number, r: number) => Promise<void>;
  fetchCellsRect: (minX: number, minY: number, maxX: number, maxY: number) => Promise<void>;
  fetchNearbyPlayers: (x: number, y: number, r: number) => Promise<void>;
  upsertPlayerPosition: (p: NearbyPlayer) => void;
  pruneStalePlayers: () => void;
  movePath: (x: number, y: number) => Promise<{ player: Player; log: string; combat_triggered: boolean; reached: boolean; skip_result?: any } | null>;
  teleportToRegion: (regionId: string) => Promise<TeleportResponse | null>;
  useGuOnMap: (guInstanceId: string, targetX: number, targetY: number) => Promise<{ message: string; logs: string[] } | null>;
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
  cellCache: {},
  nearbyWindow: null,
  playerPositions: {},
  nearbyPlayers: [],
  exploredSubZones: new Set<string>(),
  movementRange: 1,
  investigationRange: 10,
  selectedObject: null,
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

  requestCenterCamera: () => set((s) => ({ cameraCenterRequest: s.cameraCenterRequest + 1 })),
  addWorldLog: (text: string) =>
    set((s) => ({
      worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text, timestamp: Date.now() }].slice(-50),
    })),

  fetchObjects: async () => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${BACKEND_URL}/api/explore/objects`, { headers: authHeaders("GET") });
      if (!res.ok) throw new Error("Không thể tải các điểm trên bản đồ");
      const data = await res.json();
      set({ objects: Array.isArray(data) ? data : [], loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  movePath: async (x: number, y: number) => {
    set({ isMoving: true, error: null });
    try {
      const res = await fetch(`${BACKEND_URL}/api/explore/move-path`, {
        method: "POST",
        headers: { ...authHeaders("POST"), "Content-Type": "application/json" },
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
        set((s) => ({
          isMoving: false,
          error: msg,
          worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
        }));
        return null;
      }
      const data = JSON.parse(body);
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      set((s) => ({
        error: msg,
        isMoving: false,
        worldLogs: [...s.worldLogs, { id: Math.random().toString(36).substr(2, 9), text: `❌ ${msg}`, timestamp: Date.now() }].slice(-50),
      }));
      return null;
    } finally {
      set({ isMoving: false });
    }
  },

  useGuOnMap: async (guInstanceId: string, targetX: number, targetY: number) => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${BACKEND_URL}/api/explore/use_gu`, {
        method: "POST",
        headers: { ...authHeaders("POST"), "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: guInstanceId, target_x: targetX, target_y: targetY }),
      });
      const body = await res.text();
      let data;
      try {
        data = JSON.parse(body);
      } catch (e) {
        throw new Error(body || "Invalid response");
      }
      if (!res.ok) {
        set({ error: data?.error || data?.message || "Failed to use Gu on map", loading: false });
        return null;
      }
      useGameStore.getState().setPlayerFromServer(data.player);
      if (data.player) {
        get().fetchNearbyCells(data.player.x, data.player.y, 300);
        get().fetchNearbyPlayers(data.player.x, data.player.y, 300);
      }
      set({ loading: false });
      return data;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : "Error", loading: false });
      return null;
    }
  },

  teleportToRegion: async (regionId: string) => {
    set({ loading: true, error: null });
    try {
      const res = await fetch(`${BACKEND_URL}/api/explore/teleport`, {
        method: "POST",
        headers: { ...authHeaders("POST"), "Content-Type": "application/json" },
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
        set((s) => ({
          loading: false,
          error: msg,
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
      set((s) => ({
        error: msg,
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
  setInvestigationRange: (r) => set({ investigationRange: r > 10 ? r : 10 }),
  setMoving: (v) => set({ isMoving: v }),
  selectObject: (obj) => set({ selectedObject: obj }),
  setLivePosition: (x, y) => set({ liveX: x, liveY: y }),
  setCamera: (scale, x, y) => set({ cameraScale: scale, cameraX: x, cameraY: y }),

  fetchCellsRect: async (minX, minY, maxX, maxY) => {
    try {
      const url = `${BACKEND_URL}/api/explore/cells/rect?min_x=${Math.round(minX)}&min_y=${Math.round(minY)}&max_x=${Math.round(maxX)}&max_y=${Math.round(maxY)}`;
      const res = await fetch(url, { headers: authHeaders("GET") });
      if (!res.ok) throw new Error("Không thể tải dữ liệu bản đồ");
      const data = await res.json();
      const cells: NearbyCell[] = Array.isArray(data) ? data : [];
      set((s) => {
        const cache = { ...s.cellCache };
        for (const c of cells) {
          cache[`${c.x},${c.y}`] = c;
        }
        // Bound memory: keep cells near the player window, drop the far ones.
        const keys = Object.keys(cache);
        if (keys.length > 150000) {
          const w = s.nearbyWindow;
          if (w) {
            const keep = Math.max(w.r, 300) * 3;
            for (const key of keys) {
              const [cx, cy] = key.split(",").map(Number);
              if (Math.abs(cx - w.x) > keep || Math.abs(cy - w.y) > keep) {
                delete cache[key];
              }
            }
          }
        }
        // Derive the visible window from whatever chunks have been loaded so
        // far; markers appear progressively as prefetch tiles resolve.
        let nearbyCells = s.nearbyCells;
        const w = s.nearbyWindow;
        if (w) {
          nearbyCells = [];
          for (const key of Object.keys(cache)) {
            const [cx, cy] = key.split(",").map(Number);
            if (cx >= w.x - w.r && cx <= w.x + w.r && cy >= w.y - w.r && cy <= w.y + w.r) {
              nearbyCells.push(cache[key]);
            }
          }
        }
        return { cellCache: cache, nearbyCells };
      });
    } catch (err) {
      console.error("fetchCellsRect error:", err instanceof Error ? err.message : err);
    }
  },

  fetchNearbyCells: async (x, y, r) => {
    set({ nearbyWindow: { x, y, r } });
    await useWorldStore.getState().fetchCellsRect(x - r, y - r, x + r, y + r);
  },

  fetchNearbyPlayers: async (x, y, r) => {
    try {
      const url = `${BACKEND_URL}/api/explore/players?x=${Math.round(x)}&y=${Math.round(y)}&r=${Math.round(r)}`;
      const res = await fetch(url, { headers: authHeaders("GET") });
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
        // Keep the nearby window in sync with the fast player poll so the
        // slower cell poll can back off without freezing the viewport filter.
        const w = { x, y, r };
        let nearbyPlayers = s.nearbyPlayers;
        nearbyPlayers = Object.keys(positions)
          .map((id) => positions[id])
          .filter((p2) => p2.x >= w.x - w.r && p2.x <= w.x + w.r && p2.y >= w.y - w.r && p2.y <= w.y + w.r);
        return { playerPositions: positions, nearbyPlayers, nearbyWindow: w };
      });
    } catch (err) {
      console.error("fetchNearbyPlayers error:", err instanceof Error ? err.message : err);
    }
  },

  upsertPlayerPosition: (p) => {
    set((s) => {
      const now = Date.now();
      const prev = s.playerPositions[p.id];
      if (prev && prev.ts > now) return {};
      const updated = { ...s.playerPositions, [p.id]: { ...p, ts: now } };
      const w = s.nearbyWindow;
      if (w) {
        const inWindow = (pp: NearbyPlayer) => pp.x >= w.x - w.r && pp.x <= w.x + w.r && pp.y >= w.y - w.r && pp.y <= w.y + w.r;
        let nearby: NearbyPlayer[];
        if (s.nearbyPlayers.some((p2) => p2.id === p.id)) {
          nearby = s.nearbyPlayers.map((p2) => (p2.id === p.id ? { ...p2, ...p } : p2));
        } else if (inWindow(p)) {
          nearby = [...s.nearbyPlayers, p];
        } else {
          nearby = s.nearbyPlayers;
        }
        return { playerPositions: updated, nearbyPlayers: nearby };
      }
      return { playerPositions: updated };
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
