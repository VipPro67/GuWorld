import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface MineSlot {
  id: number;
  rarity: string;
  mine_type?: string;
  occupier_id?: string;
  occupier_name?: string;
  monster_id?: string;
  occupied_at?: string;
  last_claimed_at?: string;
  accumulated_stones: number;
  accumulated_qi: number;
  accumulated_chests: number;
}

export interface MineSlice {
  mineSlots: MineSlot[];
  fetchMineSlots: () => Promise<void>;
  occupyMineSlot: (slotId: number) => Promise<boolean>;
  claimMineRewards: (slotId: number) => Promise<boolean>;
  abandonMineSlot: (slotId: number) => Promise<boolean>;
  attackMineSlot: (slotId: number) => Promise<boolean>;
}

export const createMineSlice: StateCreator<any, any, any, MineSlice> = (set, get) => ({
  mineSlots: [],

  fetchMineSlots: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/slots`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ mineSlots: Array.isArray(data) ? data : [] });
      }
    } catch (err) {
      console.error("fetch mine slots failed", err);
      get().addToast("Lỗi tải mỏ.", "error");
    }
  },

  occupyMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/occupy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chiếm mỏ thất bại");
      get().addLog(data.message || "Chiếm mỏ thành công!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Chiếm mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  claimMineRewards: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      get().addLog(data.message || "Đã nhận thưởng tích lũy!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Nhận thưởng mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  abandonMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/abandon`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút lui thất bại");
      get().addLog(data.message || "Đã rời khỏi mỏ và nhận hết thưởng!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rời mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  attackMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tấn công mỏ thất bại");
      get().addLog(data.message || "Bắt đầu thách đấu chiếm mỏ!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tấn công mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },
});
