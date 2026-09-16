import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface GachaSlice {
  gachaPools: ApiResponseData[];
  fetchGachaPools: () => Promise<void>;
  openGacha: (poolId: string, quantity?: number) => Promise<{ success: boolean; message: string; reward?: ApiResponseData; rewards?: ApiResponseData[] } | null>;
}

export const createGachaSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, GachaSlice> = (set, get) => ({
  gachaPools: [],

  fetchGachaPools: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/pools`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ gachaPools: data.pools || [] });
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Gacha pools error:", err.message);
      get().addToast("Lỗi tải bách khoa.", "error");
    }
  },

  openGacha: async (poolId, quantity = 1) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/open`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pool_id: poolId, quantity }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mở Đá  thất bại."); }
      // Re-sync full inventory from server
      await get().initializeSession();
      get().addLog(data.message || "Mở Đá  thành công!");
      return { success: true, message: data.message, reward: data.reward, rewards: data.rewards };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mở Đá  lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },
});
