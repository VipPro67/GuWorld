import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { GroundDrop, Player } from "@/store/useGameStore";
import { executeGameAction } from "@/store/actionGateway";

let groundDropRequestRevision = 0;
let groundDropRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let groundDropExpiryTimer: ReturnType<typeof setTimeout> | null = null;

export function resetWorldSliceModuleState() {
  groundDropRequestRevision = 0;
  if (groundDropRefreshTimer !== null) {
    clearTimeout(groundDropRefreshTimer);
    groundDropRefreshTimer = null;
  }
  if (groundDropExpiryTimer !== null) {
    clearTimeout(groundDropExpiryTimer);
    groundDropExpiryTimer = null;
  }
}

export interface GroundDropViewport {
  x: number;
  y: number;
  radius: number;
}

export interface WorldSlice {
  groundDrops: GroundDrop[];
  lastDropViewport: GroundDropViewport | null;
  fetchGroundDrops: (viewport?: GroundDropViewport) => Promise<void>;
  applyWorldDropsEvent: (event: { region_id?: string }) => void;
  lootGroundDrop: (dropId: string) => Promise<{ success: boolean; message: string; looted?: string[] } | null>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: ApiResponseData } | null>;
  fetchGrid: (radius?: number) => Promise<{ x: number; y: number; vision_radius: number; max_vision_radius?: number; cells: ApiResponseData[]; stats: ApiResponseData; players_on_map?: ApiResponseData[]; ground_drops?: ApiResponseData[]; world_width?: number; world_height?: number } | null>;
}

export const createWorldSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, WorldSlice> = (set, get) => ({
  groundDrops: [],
  lastDropViewport: null,

  fetchGroundDrops: async (viewport) => {
    if (viewport) set({ lastDropViewport: viewport });
    const requestRevision = ++groundDropRequestRevision;
    try {
      const params = new URLSearchParams();
      if (viewport) {
        params.set("x", String(Math.floor(viewport.x)));
        params.set("y", String(Math.floor(viewport.y)));
        params.set("radius", String(Math.max(0, Math.floor(viewport.radius))));
      }
      const res = await authFetch(`${BACKEND_URL}/api/world/drops?${params.toString()}`);
      if (!res.ok) return;
      const data = await res.json();
      if (requestRevision === groundDropRequestRevision) {
        const drops = data.drops || [];
        set({ groundDrops: drops });
        if (groundDropExpiryTimer) {
          clearTimeout(groundDropExpiryTimer);
          groundDropExpiryTimer = null;
        }
        const center = viewport ?? get().lastDropViewport;
        let nextChange = Number.POSITIVE_INFINITY;
        for (const drop of drops) {
          for (const iso of [drop.expires_at, drop.protected_until]) {
            if (!iso) continue;
            const at = new Date(iso).getTime();
            if (Number.isFinite(at) && at > Date.now() && at < nextChange) nextChange = at;
          }
        }
        if (center && Number.isFinite(nextChange)) {
          groundDropExpiryTimer = setTimeout(() => {
            groundDropExpiryTimer = null;
            void get().fetchGroundDrops(center);
          }, Math.max(1000, nextChange - Date.now() + 1000));
        }
      }
    } catch (e) {
      console.error("fetchGroundDrops error:", e);
    }
  },

  applyWorldDropsEvent: (event) => {
    const currentRegion = get().player?.current_region;
    if (!event?.region_id || (currentRegion && event.region_id !== currentRegion)) return;
    const viewport = get().lastDropViewport;
    if (!viewport) return;
    if (groundDropRefreshTimer) clearTimeout(groundDropRefreshTimer);
    groundDropRefreshTimer = setTimeout(() => {
      groundDropRefreshTimer = null;
      void get().fetchGroundDrops(viewport);
    }, 250);
  },

  lootGroundDrop: async (dropId) => {
    const res = await executeGameAction<{ message?: string; looted?: string[] }>(get, set, {
      endpoint: `/api/world/drops/${dropId}/loot`,
      method: "POST",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        set({ groundDrops: get().groundDrops.filter((d: ApiResponseData) => d.id !== dropId) });
      },
      successToast: (data) => data.message || "Thu thập chiến lợi phẩm thành công!",
      errorMessage: "Nhặt chiến lợi phẩm thất bại",
    });
    if (!res.ok || !res.data) {
      return res.error ? { success: false, message: res.error } : null;
    }
    return { success: true, message: res.data.message || "", looted: res.data.looted };
  },

  movePlayer: async (x, y, attackTargetId) => {
    const { player, token, actionLoading } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          x,
          y,
          attack_target_id: attackTargetId,
        }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      get().applyInventoryFromResponse(data);
      get().addLog(data.log || "Di chuyển thành công.");

      if (data.skip_result) {
        const skipResult = data.skip_result;
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!`, "success");
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }

      return data;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Thao tác thất bại.";
      set({ error: message });
      get().addToast(message, "error");
      get().addLog(`Di chuyển thất bại: ${message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchGrid: async (radius?: number) => {
    const { token } = get();
    if (!token) return null;

    try {
      const url = radius ? `${BACKEND_URL}/api/explore/grid?radius=${radius}` : `${BACKEND_URL}/api/explore/grid`;
      const response = await authFetch(url);
      if (!response.ok) {
        throw new Error("Failed to load map grid viewport");
      }
      return await response.json();
    } catch (e) {
      console.error("fetchGrid error:", e);
      return null;
    }
  },
});
