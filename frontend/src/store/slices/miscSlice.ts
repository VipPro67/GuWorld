import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface MiscSlice {
  auditLogs: ApiResponseData[];
  auditLogsLoading: boolean;
  fetchAuditLogs: (filters?: { action?: string; entity_type?: string; success?: boolean; search?: string; limit?: number; offset?: number }) => Promise<void>;
  clearStress: () => Promise<{ success: boolean; message: string } | null>;
  sacrificeNPC: (npcId: string) => Promise<{ success: boolean; message: string } | null>;
}

export const createMiscSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, MiscSlice> = (set, get) => ({
  auditLogs: [],
  auditLogsLoading: false,

  clearStress: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/clear-stress`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tịnh hóa Không khiếu thất bại.");

      get().addLog(data.message || "Tịnh hóa Không khiếu hoàn tất.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Tịnh hóa lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  sacrificeNPC: async (npcId: string) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/sacrifice`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ npc_id: npcId }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hiến tế thất bại.");

      const updatedPlayer = { ...get().player };
      if (updatedPlayer) {
        updatedPlayer.aptitude_percent = data.new_aptitude;
        updatedPlayer.aptitude_type = data.aptitude_type;
        updatedPlayer.aperture_stress = data.aperture_stress;
        set({ player: updatedPlayer as ApiResponseData });
      }

      get().addLog(data.message || "Hiến tế hoàn tất.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Hiến tế lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchAuditLogs: async (filters) => {
    const { token, logout } = get();
    if (!token) return;

    set({ auditLogsLoading: true });
    try {
      let url = `${BACKEND_URL}/api/player/audit-logs`;
      const queryParams: string[] = [];
      if (filters) {
        if (filters.action) queryParams.push(`action=${encodeURIComponent(filters.action)}`);
        if (filters.entity_type) queryParams.push(`entity_type=${encodeURIComponent(filters.entity_type)}`);
        if (filters.success !== undefined) queryParams.push(`success=${filters.success}`);
        if (filters.search) queryParams.push(`search=${encodeURIComponent(filters.search)}`);
        if (filters.limit !== undefined) queryParams.push(`limit=${filters.limit}`);
        if (filters.offset !== undefined) queryParams.push(`offset=${filters.offset}`);
      }
      if (queryParams.length > 0) {
        url += `?${queryParams.join("&")}`;
      }

      const response = await authFetch(url, { method: "GET" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ auditLogs: data || [] });
      }
    } catch {
      // Non-critical
    } finally {
      set({ auditLogsLoading: false });
    }
  },
});
