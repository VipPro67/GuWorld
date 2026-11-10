import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type {
  CombatState,
  CombatHistoryEntry,
  AutoCombatConfig,
} from "@/store/useGameStore";

export interface CombatSlice {
  executeAutoCombat: (loadout: ApiResponseData) => Promise<ApiResponseData | null>;
  fleeCombat: () => Promise<ApiResponseData | undefined>;
  captureMonster: () => Promise<ApiResponseData | null>;
  resolveTribulation: (eventId: string) => Promise<boolean>;
  activeCombatReplay: { logs: ApiResponseData[]; winner: string; combatants: ApiResponseData[]; enemy: CombatState } | null;
  setActiveCombatReplay: (replay: CombatSlice["activeCombatReplay"]) => void;
  executePracticeCombat: (loadout: ApiResponseData, dummyConfig?: ApiResponseData) => Promise<ApiResponseData | null>;
  startPracticeCombat: (dummyConfig: ApiResponseData) => Promise<boolean>;
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
  activeCombatReplay: null,
  setActiveCombatReplay: (replay) => set({ activeCombatReplay: replay }),
  combatSettingsLoading: false,
  combatHistory: [],
  combatHistoryLoading: false,

  executeAutoCombat: async (loadout) => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/start`, {
        method: "POST", timeoutMs: 120000,
        headers: { "Content-Type": "application/json" }, body: JSON.stringify({ loadout }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      if (Array.isArray(data.logs)) {
        get().setActiveCombatReplay({
          logs: data.logs,
          winner: data.winner || "Phe Địch",
          combatants: Array.isArray(data.combatants) ? data.combatants : [],
          enemy: player.active_combat ?? {
            enemy_name: "Kẻ Địch",
            enemy_hp: 0,
            enemy_max_hp: 0,
            enemy_damage: 0,
            enemy_defense: 0,
            charge_points: 0,
          },
        });

        const rewardLines = data.logs
          .filter((entry: ApiResponseData) => entry?.action === "LOOT" || entry?.action === "ARENA_REWARD")
          .map((entry: ApiResponseData) => entry?.detail)
          .filter((detail: unknown): detail is string => typeof detail === "string" && detail.trim().length > 0);
        if (rewardLines.length > 0) {
          get().addToast(rewardLines.join(" "), "success");
        }

        if (data.winner === "Phe Ta" && get().player) {
          const playerID = get().player!.id;
          await Promise.all([
            get().fetchPlayerStatus(playerID, true),
            get().fetchGuInventory(true),
          ]);
        }
      }
      return data;
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "Mô phỏng thất bại.";
      set({ error: message }); get().addToast(message, "error"); return null;
    } finally { get().endAction("player:combat"); }
  },

  fleeCombat: async () => {
    const { player, token } = get();
    if (!player || !token) return;
    const response = await authFetch(`${BACKEND_URL}/api/combat/flee/${player.id}`, { method: "POST", timeoutMs: 30000 });
    const data = await response.json();
    if (!response.ok) { get().addToast(data.error || "Bỏ chạy thất bại.", "error"); return; }
    if (data.flee_success !== false) {
      set({ activeCombatReplay: null });
      get().setPlayerFromServer({ active_combat: null, hp: data.hp ?? player.hp });
    }
    return data;
  },

  captureMonster: async () => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/capture`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thu phục thất bại.");
      get().addToast(data.message || (data.captured ? "Thu phục thành công." : "Thu phục thất bại."), data.captured ? "success" : "warning");
      return data;
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "Thu phục thất bại.";
      set({ error: message });
      get().addToast(message, "error");
      return null;
    } finally {
      get().endAction("player:combat");
    }
  },

  resolveTribulation: async (eventId) => {
    const { token } = get();
    if (!token || !get().beginAction("aperture")) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/resolve`, {
        method: "POST",
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Kết toán tai kiếp thất bại.");
      get().addToast(data.message || "Kết toán tai kiếp hoàn thành!", "success");
      get().addLog(data.message || "Kết toán tai kiếp hoàn thành.");
      if (get().player) await get().fetchPlayerStatus(get().player!.id, true);
      await get().fetchAperture(true);
      await get().fetchDaoMarks(true);
      set({ activeTribulation: null });
      return true;
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "Kết toán tai kiếp thất bại.";
      set({ error: message });
      get().addToast(message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

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

  executePracticeCombat: async (_loadout: ApiResponseData, _dummyConfig?: ApiResponseData) => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/start`, {
        method: "POST",
        timeoutMs: 120000,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ loadout: _loadout }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mô phỏng tập luyện thất bại.");
      if (Array.isArray(data.logs)) get().setActiveCombatReplay({
        logs: data.logs, winner: data.winner || "Phe Địch",
        combatants: Array.isArray(data.combatants) ? data.combatants : [],
        enemy: player.active_combat!,
      });
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mô phỏng tập luyện thất bại: ${err.message}`);
      return null;
    } finally {
      get().endAction("player:combat");
    }
  },

  startPracticeCombat: async (dummyConfig) => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:combat")) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/practice`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          dummy_hp: Number(dummyConfig.dummyHP) || 100000,
          dummy_shield: Number(dummyConfig.dummyShield) || 0,
          dummy_atk: Number(dummyConfig.dummyAtk) || 10,
          dummy_def: Number(dummyConfig.dummyDef) || 5,
          dummy_speed: Number(dummyConfig.dummySpeed) || 1,
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bắt đầu tập luyện thất bại.");
      get().setPlayerFromServer({ active_combat: data.active_combat });
      return true;
    } catch (caught) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message }); get().addToast(err.message, "error"); return false;
    } finally { get().endAction("player:combat"); }
  },

  exitPracticeCombat: async () => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/practice/exit`, { method: "POST" });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Kết thúc tập luyện thất bại.");
      get().setPlayerFromServer({ active_combat: null });
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
