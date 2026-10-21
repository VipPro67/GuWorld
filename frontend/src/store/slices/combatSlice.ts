import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useUIStore } from "@/store/useUIStore";
import type {
  Player,
  CombatState,
  CombatHistoryEntry,
  AutoCombatConfig,
  ExploreResponse,
} from "@/store/useGameStore";

export interface CombatSlice {
  explore: () => Promise<void>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: ApiResponseData } | null>;
  executeAutoCombat: (loadout: ApiResponseData) => Promise<{ logs: ApiResponseData[]; winner: string; combatants?: ApiResponseData[] } | null>;
  executePracticeCombat: (loadout: ApiResponseData, dummyConfig: ApiResponseData) => Promise<{ logs: ApiResponseData[]; winner: string; combatants?: ApiResponseData[], player_won?: boolean } | null>;
  combatHistory: CombatHistoryEntry[];
  combatHistoryLoading: boolean;
  fetchCombatHistory: () => Promise<void>;
  fetchCombatHistoryEntry: (id: string) => Promise<CombatHistoryEntry | null>;
  activeCombatReplay: { logs: ApiResponseData[]; winner: string; combatants: ApiResponseData[]; enemy: CombatState } | null;
  setActiveCombatReplay: (replay: { logs: ApiResponseData[]; winner: string; combatants: ApiResponseData[]; enemy: CombatState } | null) => void;
  combatSettings: AutoCombatConfig | null;
  combatSettingsLoading: boolean;
  fetchCombatSettings: () => Promise<void>;
  saveCombatSettings: (settings: AutoCombatConfig) => Promise<boolean>;
  retreat: () => Promise<void>;
  fleeCombat: () => Promise<ApiResponseData>;
}

export const createCombatSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, CombatSlice> = (set, get) => ({
  combatSettings: null,
  combatSettingsLoading: false,
  combatHistory: [],
  combatHistoryLoading: false,

  explore: async () => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:combat")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/explore/${player.id}`, {
        method: "POST",
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Khám phá thất bại.");
      }

      const data: ExploreResponse = await response.json();
      get().applyInventoryFromResponse(data);
      get().addLog(data.log);
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:combat");
    }
  },

  exploreNode: async (nodeId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          node_id: nodeId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Khám phá thất bại.");
      }

      get().applyInventoryFromResponse(data);
      get().addLog(data.log);

      // Handle skip battle result from explore
      if (data.skip_result) {
        const skipResult = data.skip_result;
        const allies = Array.isArray(skipResult.allies) ? skipResult.allies.slice(1) : [];
        const allyText = allies.length > 0 ? ` Cùng ${allies.map((a: { name: string }) => a.name).join(", ")} xuất trận.` : " Một mình xuất trận.";
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!${allyText}`, "success");
          if (skipResult.mine_occupied) {
            get().addToast(`Đã chiếm mỏ ${skipResult.mine_occupied}!`, "info");
          }
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }

      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return null;
    } finally {
      get().endAction("player:combat");
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

  executeAutoCombat: async (loadout: ApiResponseData) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:combat")) return null;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/start`, {
        method: "POST",
        // Full auto-combat simulations legitimately take longer than ordinary
        // CRUD requests, especially for parties and multi-enemy encounters.
        timeoutMs: 120000,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ loadout }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        if (response.status === 400 && (data.error?.includes("Không trong trạng thái chiến đấu") || data.error?.includes("Không có trận chiến nào"))) {
          const currentPlayer = get().player;
          if (currentPlayer && currentPlayer.active_combat) {
            set((state: ApiResponseData) => ({
              player: { ...currentPlayer, active_combat: null },
              playerRevision: state.playerRevision + 1,
              activeCombatReplay: null,
            }));
          }
        }
        throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      }

      // We just return the logs and winner, we don't need to update player immediately
      // The frontend handles the replay animation
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      const message = err?.name === "AbortError"
        ? "Mô phỏng chiến đấu quá thời gian. Trạng thái nhân vật sẽ được đồng bộ lại."
        : (err?.message || "Thao tác thất bại.");
      set({ error: message });
      get().addToast(message, "error");
      get().addLog(`Mô phỏng thất bại: ${message}`);
      if (err?.name === "AbortError") {
        setTimeout(() => void get().fetchPlayerStatus(player.id), 0);
      }
      return null;
    } finally {
      get().endAction("player:combat");
    }
  },

  executePracticeCombat: async (loadout: ApiResponseData, dummyConfig: ApiResponseData) => {
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
          loadout,
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
        throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      }

      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mô phỏng thất bại: ${err.message}`);
      return null;
    } finally {
      get().endAction("player:combat");
    }
  },

  activeCombatReplay: null,
  setActiveCombatReplay: (replay) => set({ activeCombatReplay: replay }),

  retreat: async () => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:combat")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/retreat/${player.id}`, {
        method: "POST",
        timeoutMs: 30000,
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Rút lui thất bại.");
      }

      set({ activeCombatReplay: null });
      useUIStore.getState().setView("dashboard");
      get().addLog(data.log || "Đạo hữu đã rút lui an toàn.");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Rút lui thất bại: ${err.message}`);
    } finally {
      get().endAction("player:combat");
    }
  },

  fleeCombat: async () => {
    const { player, token, logout, showConfirm } = get();
    if (!player || !token || !get().beginAction("player:combat")) return;

    // HP quá thấp (< 20% max): bỏ chạy trừ 20% Max HP — gần như chắc chắn ngất
    // xỉu, mất toàn bộ tài nguyên thu thập. Cảnh báo và chặn lại cho đạo hữu cân nhắc.
    // PvP được bỏ chạy tự do nên bỏ qua cảnh báo này.
    const maxHP = player.base_hp || player.hp || 1;
    const isPVP = player.active_combat?.enemy_type === "PVP";
    if (!isPVP && player.hp <= 0.2 * maxHP) {

      const proceed = await showConfirm(
        "HP của đạo hữu đang rất thấp. Bỏ chạy khỏi chiến đấu sẽ khiến đạo hữu chịu thêm 20% sát thương theo HP tối đa — gần như chắc chắn sẽ ngất xỉu, làm rơi mất toàn bộ Nguyên Thạch, Cổ trùng và tài nguyên thu thập. Vẫn muốn bỏ chạy?",
        { confirmText: "Vẫn bỏ chạy", cancelText: "Hủy" },
      );
      if (!proceed) {
        get().endAction("player:combat");
        return;
      }
    }

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/flee/${player.id}`, {
        method: "POST",
        timeoutMs: 30000,
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        if (response.status === 400 && data.error?.includes("Không có trận chiến nào")) {
          const currentPlayer = get().player;
          if (currentPlayer && currentPlayer.active_combat) {
            set((state: ApiResponseData) => ({
              player: { ...currentPlayer, active_combat: null },
              playerRevision: state.playerRevision + 1,
              activeCombatReplay: null,
            }));
          } else {
            set({ activeCombatReplay: null });
          }
        }
        throw new Error(data.error || "Bỏ chạy thất bại.");
      }

      if (data.flee_success === false) {
        get().applyInventoryFromResponse(data);
        if (data.logs && Array.isArray(data.logs)) {
          data.logs.forEach((logLine: string) => {
            get().addLog(logLine);
          });
        } else {
          get().addLog("Rút lui thất bại!");
        }
        if (data.victory) {
          get().addToast("Địch nhân tự chuốc lấy diệt vong! Chiến thắng bất ngờ!", "success");
        } else if (data.defeat) {
          get().addToast("Rút lui thất bại! Đạo hữu bị đánh bại!", "error");
        } else {
          get().addToast("Cố gắng rút lui thất bại! Địch nhân thừa cơ tấn công!", "info");
        }
        return data;
      }

      const combat = player.active_combat;
      let returnView = useUIStore.getState().currentView;
      if (combat) {
        if (combat.combat_mode === "virtual") returnView = "arena";
        else if (combat.tribulation_event_id) returnView = "aperture";
      }
      set({ activeCombatReplay: null });
      get().applyInventoryFromResponse(data);
      useUIStore.getState().setView(returnView);
      get().addLog(data.log || "Đạo hữu đã bỏ chạy an toàn.");
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bỏ chạy thất bại: ${err.message}`);
    } finally {
      get().endAction("player:combat");
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
