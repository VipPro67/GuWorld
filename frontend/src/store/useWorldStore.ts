"use client";

import { create } from "zustand";
import type { Player } from "@/store/useGameStore";

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
  fetchNearbyPlayers: (x: number, y: number, r: number) => Promise<void>;
  movePath: (x: number, y: number) => Promise<{ player: Player; log: string; combat_triggered: boolean; reached: boolean } | null>;
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

export const useWorldStore = create<WorldStoreState>((set) => ({
  objects: [],
  nearbyCells: [],
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

  setExploredSubZones: (keys) => set({ exploredSubZones: new Set<string>(keys) }),
  setMovementRange: (r) => set({ movementRange: r > 0 ? r : 1 }),
  setInvestigationRange: (r) => set({ investigationRange: r > 10 ? r : 10 }),
  setMoving: (v) => set({ isMoving: v }),
  selectObject: (obj) => set({ selectedObject: obj }),
  setLivePosition: (x, y) => set({ liveX: x, liveY: y }),
  setCamera: (scale, x, y) => set({ cameraScale: scale, cameraX: x, cameraY: y }),

  fetchNearbyCells: async (x, y, r) => {
    try {
      const url = `${BACKEND_URL}/api/explore/cells?x=${Math.round(x)}&y=${Math.round(y)}&r=${Math.round(r)}`;
      const res = await fetch(url, { headers: authHeaders("GET") });
      if (!res.ok) throw new Error("Không thể tải quái vật trên bản đồ");
      const data = await res.json();
      set({ nearbyCells: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("fetchNearbyCells error:", err instanceof Error ? err.message : err);
    }
  },

  fetchNearbyPlayers: async (x, y, r) => {
    try {
      const url = `${BACKEND_URL}/api/explore/players?x=${Math.round(x)}&y=${Math.round(y)}&r=${Math.round(r)}`;
      const res = await fetch(url, { headers: authHeaders("GET") });
      if (!res.ok) throw new Error("Không thể tải người chơi trên bản đồ");
      const data = await res.json();
      set({ nearbyPlayers: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("fetchNearbyPlayers error:", err instanceof Error ? err.message : err);
    }
  },
}));
