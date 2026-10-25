import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface TeleportArray {
  id: string;
  name: string;
  owner_id: string;
  owner_name?: string;
  sect_id?: string;
  sect_name?: string;
  region_id: string;
  region_name?: string;
  x: number;
  y: number;
  tier: string;
  max_range: number;
  is_cross_region: boolean;
  direction_type: "TWO_WAY" | "ONE_WAY_SEND" | "ONE_WAY_RECEIVE";
  access_policy: "PERSONAL" | "SECT" | "PUBLIC";
  fee_enabled: boolean;
  fee_currency: "primeval_stones" | "immortal_stones";
  fee_amount: number;
  vault_balance: number;
  current_durability: number;
  max_durability: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TeleportTemplate {
  tier: string;
  name: string;
  description: string;
  min_rank: number;
  min_formation_realm?: number;
  base_success_rate?: number;
  max_range: number;
  is_cross_region: boolean;
  max_durability: number;
  cost_primeval_stones: number;
  cost_immortal_stones: number;
  required_gu: string[];
  required_items: Record<string, number>;
}

export interface TeleportDirectoryItem {
  array: TeleportArray;
  distance: number;
  is_cross_region_jump: boolean;
  can_use: boolean;
  reason?: string;
  fee_to_pay: number;
  fee_currency: string;
}

export interface TeleportDirectoryData {
  is_standing_on_array: boolean;
  source_array: TeleportArray | null;
  destinations: TeleportDirectoryItem[];
}

export interface TeleportBuildResult {
  success: boolean;
  message: string;
  array?: TeleportArray;
  success_rate?: number;
  formation_realm?: string;
  formation_level?: number;
  discount_percent?: number;
  mastery_gained?: number;
  paid_primeval_stones?: number;
  paid_immortal_stones?: number;
}

export interface TeleportSlice {
  teleportTemplates: TeleportTemplate[];
  myTeleportArrays: TeleportArray[];
  teleportDirectory: TeleportDirectoryData | null;
  teleportLoading: boolean;

  fetchTeleportTemplates: () => Promise<void>;
  fetchMyTeleportArrays: () => Promise<void>;
  fetchTeleportDirectory: () => Promise<void>;
  buildTeleportArray: (req: {
    name: string;
    tier: string;
    direction_type?: string;
    access_policy?: string;
    fee_enabled?: boolean;
    fee_currency?: string;
    fee_amount?: number;
  }) => Promise<TeleportBuildResult>;
  travelTeleport: (targetArrayId: string) => Promise<{ success: boolean; message: string }>;
  manageTeleportArray: (req: {
    array_id: string;
    name?: string;
    direction_type?: string;
    access_policy?: string;
    fee_enabled?: boolean;
    fee_currency?: string;
    fee_amount?: number;
  }) => Promise<boolean>;
  repairTeleportArray: (arrayId: string, amount?: number) => Promise<boolean>;
  withdrawTeleportVault: (arrayId: string, amount?: number) => Promise<boolean>;
  dismantleTeleportArray: (arrayId: string) => Promise<boolean>;
}

export const createTeleportSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, TeleportSlice> = (set, get) => ({
  teleportTemplates: [],
  myTeleportArrays: [],
  teleportDirectory: null,
  teleportLoading: false,

  fetchTeleportTemplates: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/templates`);
      if (!response.ok) return;
      const data: TeleportTemplate[] = await response.json();
      set({ teleportTemplates: data });
    } catch (err) {
      console.error("Failed to fetch teleport templates", err);
    }
  },

  fetchMyTeleportArrays: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/mine`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data: TeleportArray[] = await response.json();
      set({ myTeleportArrays: data });
    } catch (err) {
      console.error("Failed to fetch player teleport arrays", err);
    }
  },

  fetchTeleportDirectory: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      set({ teleportLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/teleport/directory`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data: TeleportDirectoryData = await response.json();
      set({ teleportDirectory: data });
    } catch (err) {
      console.error("Failed to fetch teleport directory", err);
    } finally {
      set({ teleportLoading: false });
    }
  },

  buildTeleportArray: async (req) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return { success: false, message: "Đang bận" };
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/build`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bố trí trận pháp thất bại");

      if (data.success) {
        get().addToast(data.message || "Bố trí trận pháp thành công!", "success");
        get().addLog(data.message || "Bố trí trận pháp thành công!");
      } else {
        get().addToast(data.message || "Bố trận thất bại!", "warning");
        get().addLog(data.message || "Bố trận thất bại!");
      }

      await get().fetchPlayerProfile();
      await get().fetchMyTeleportArrays();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Thất bại: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      get().endAction("player:position");
    }
  },

  travelTeleport: async (targetArrayId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return { success: false, message: "Đang bận" };
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/travel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_array_id: targetArrayId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Truyền tống thất bại");

      get().addToast(data.message || "Truyền tống thành công!", "success");
      get().addLog(data.message || "Truyền tống thành công!");
      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      await get().fetchTeleportDirectory();
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Truyền tống thất bại: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      get().endAction("player:position");
    }
  },

  manageTeleportArray: async (req) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/manage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật cấu hình thất bại");

      get().addToast(data.message || "Cập nhật thành công!", "success");
      get().addLog(data.message || "Cập nhật thành công!");
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  repairTeleportArray: async (arrayId, amount = 0) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/repair`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sửa chữa thất bại");

      get().addToast(data.message || "Sửa chữa thành công!", "success");
      get().addLog(data.message || "Sửa chữa thành công!");
      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  withdrawTeleportVault: async (arrayId, amount = 0) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/withdraw`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút tiền thất bại");

      get().addToast(data.message || "Rút tiền thành công!", "success");
      get().addLog(data.message || "Rút tiền thành công!");
      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      await get().fetchMyTeleportArrays();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  dismantleTeleportArray: async (arrayId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/dismantle`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thu hồi thất bại");

      get().addToast(data.message || "Thu hồi trận pháp thành công!", "success");
      get().addLog(data.message || "Thu hồi trận pháp thành công!");
      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },
});
