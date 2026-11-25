import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface MiscSlice {
  auditLogs: ApiResponseData[];
  auditLogsLoading: boolean;
  fetchAuditLogs: (filters?: { action?: string; entity_type?: string; success?: boolean; search?: string; limit?: number; offset?: number }) => Promise<void>;
  clearStress: () => Promise<{ success: boolean; message: string } | null>;
}

export const createMiscSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, MiscSlice> = (set, get) => ({
  auditLogs: [],
  auditLogsLoading: false,

  clearStress: async () => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:misc")) return null;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/clear-stress`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });

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
      get().endAction("player:misc");
    }
  },

  fetchAuditLogs: async (filters) => {
    const { token } = get();
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
      if (!response.ok) return;
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
