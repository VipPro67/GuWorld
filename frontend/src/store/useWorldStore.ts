"use client";

import { create } from "zustand";
import { BACKEND_URL } from "@/lib/config";
import { authFetch } from "@/lib/auth";
import { useGameStore } from "@/store/useGameStore";
import { wsManager } from "@/lib/websocket";

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

interface WorldStoreState {
  nearbyCells: NearbyCell[];
  loading: boolean;
  error: string | null;

  fetchNearbyCells: (x: number, y: number, radius: number) => Promise<boolean>;
  harvestVein: (veinId: string) => Promise<boolean>;
}

export const useWorldStore = create<WorldStoreState>((set, get) => ({
  nearbyCells: [],
  loading: false,
  error: null,

  fetchNearbyCells: async (x: number, y: number, radius: number) => {
    const boundedRadius = Math.min(Math.max(Math.round(radius), 1), 127);
    const minX = Math.round(x - boundedRadius);
    const minY = Math.round(y - boundedRadius);
    const maxX = Math.round(x + boundedRadius);
    const maxY = Math.round(y + boundedRadius);

    try {
      const fetchHttp = async () => {
        const url = `${BACKEND_URL}/api/explore/cells/rect?min_x=${minX}&min_y=${minY}&max_x=${maxX}&max_y=${maxY}`;
        const res = await authFetch(url);
        if (!res.ok) throw new Error("Could not fetch nearby cells");
        return res.json();
      };
      let data: unknown;
      if (wsManager.isConnected()) {
        try {
          data = await wsManager.request<unknown>("world.cells_rect", { min_x: minX, min_y: minY, max_x: maxX, max_y: maxY });
        } catch {
          data = await fetchHttp();
        }
      } else {
        data = await fetchHttp();
      }
      if (!Array.isArray(data)) return false;

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

      set({ nearbyCells: cells });
      return true;
    } catch (err) {
      console.warn("fetchNearbyCells failed:", err);
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
}));
