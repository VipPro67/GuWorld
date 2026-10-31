import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type {
  CombatHistoryEntry,
  AutoCombatConfig,
} from "@/store/useGameStore";

export interface CombatSlice {
  executePracticeCombat: (loadout: ApiResponseData, dummyConfig: ApiResponseData) => Promise<{ instance_id: string; zone_id: string; origin_zone_id: string; is_practice: true } | null>;
  exitPracticeCombat: () => Promise<{ zone_id: string } | null>;
  combatHistory: CombatHistoryEntry[];
  combatHistoryLoading: boolean;
  fetchCombatHistory: () => Promise<void>;
  fetchCombatHistoryEntry: (id: string) => Promise<CombatHistoryEntry | null>;
  combatSettings: AutoCombatConfig | null;
  combatSettingsLoading: boolean;
  fetchCombatSettings: () => Promise<void>;
  saveCombatSettings: (settings: AutoCombatConfig) => Promise<boolean>;
}

export const createCombatSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, CombatSlice> = (set, get) => ({
  combatSettings: null,
  combatSettingsLoading: false,
  combatHistory: [],
  combatHistoryLoading: false,

  fetchCombatSettings: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ combatSettingsLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/combat-settings`);
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tải cài đặt chiến đấu.");
      set({ combatSettings: data.combat_settings || null, combatSettingsLoading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Không thể tải cài đặt chiến đấu.";
      set({ error: message, combatSettingsLoading: false });
    }
  },

  saveCombatSettings: async (settings) => {
    const { token, logout } = get();
    const previousPvPEnabled = get().combatSettings?.pvp_enabled ?? get().player?.combat_settings?.pvp_enabled ?? false;
    if (!token || !get().beginAction("player:combat")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/combat-settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ combat_settings: settings }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lưu cài đặt thất bại.");
      set({ combatSettings: data.combat_settings || settings });
      if (previousPvPEnabled !== (settings.pvp_enabled ?? false)) get().requestWorldReconnect();
      get().addLog(data.message || "Đã lưu cài đặt chiến đấu.");
      return true;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Lưu cài đặt thất bại.";
      set({ error: message });
      get().addLog(`Lưu cài đặt thất bại: ${message}`);
      return false;
    } finally {
      get().endAction("player:combat");
    }
  },

  executePracticeCombat: async (_loadout: ApiResponseData, dummyConfig: ApiResponseData) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/practice`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          dummy_hp: parseInt(dummyConfig.dummyHP as ApiResponseData) || 100,
          dummy_shield: parseInt(dummyConfig.dummyShield as ApiResponseData) || 0,
          dummy_atk: parseInt(dummyConfig.dummyAtk as ApiResponseData) || 10,
          dummy_def: parseInt(dummyConfig.dummyDef as ApiResponseData) || 5,
          dummy_speed: parseFloat(dummyConfig.dummySpeed as ApiResponseData) || 1.0
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Bắt đầu phiên tập luyện thất bại.");
      }
      get().setPlayerFromServer({ world_zone_id: data.zone_id });
      get().requestWorldReconnect();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Không thể vào phòng tập: ${err.message}`);
      return null;
    } finally {
      get().endAction("player:combat");
    }
  },

  exitPracticeCombat: async () => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/practice/exit`, { method: "POST" });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Kết thúc tập luyện thất bại.");
      get().setPlayerFromServer({ world_zone_id: data.zone_id });
      get().requestWorldReconnect();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return null;
    }
  },

  fetchCombatHistory: async () => {
    const { token, logout } = get();
    if (!token) return;

    set({ combatHistoryLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/history`, { method: "GET" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ combatHistory: data.entries || [] });
      }
    } catch {
      // Non-critical
    } finally {
      set({ combatHistoryLoading: false });
    }
  },

  fetchCombatHistoryEntry: async (id: string) => {
    const { token, logout } = get();
    if (!token) return null;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/history/${id}`, { method: "GET" });
      if (response.status === 401) { logout(); return null; }
      const data = await response.json();
      if (response.ok) return data;
    } catch {
      return null;
    }
  },
});
