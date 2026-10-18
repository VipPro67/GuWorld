import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { PlayerInstanceSession, InstanceTemplate, InstanceListItem } from "@/store/useGameStore";

export interface DreamRealmSlice {
  activeDreamSession: PlayerInstanceSession | null;
  activeDreamTemplate: InstanceTemplate | null;
  availableDreamRealms: InstanceListItem[];
  dreamDailyEntriesUsed: number;
  dreamDailyEntriesMax: number;
  dreamActionLoading: boolean;
  fetchDreamRealms: () => Promise<void>;
  fetchActiveDreamSession: () => Promise<void>;
  enterDreamRealm: (instanceId: string) => Promise<boolean>;
  actionDreamRealm: (optionIndex: number) => Promise<boolean>;
  exitDreamRealm: () => Promise<boolean>;
}

let fetchDreamRealmsPromise: Promise<void> | null = null;

export const createDreamRealmSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, DreamRealmSlice> = (set, get) => ({
  activeDreamSession: null,
  activeDreamTemplate: null,
  availableDreamRealms: [],
  dreamDailyEntriesUsed: 0,
  dreamDailyEntriesMax: 5,
  dreamActionLoading: false,

  fetchDreamRealms: async () => {
    if (fetchDreamRealmsPromise) return fetchDreamRealmsPromise;
    const executeFetch = async () => {
      const { token } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/dream-realms`);
        if (response.ok) {
          const data = await response.json();
          set({
            availableDreamRealms: data.instances || [],
            dreamDailyEntriesUsed: data.daily_entries_used ?? 0,
            dreamDailyEntriesMax: data.daily_entries_max ?? 5,
          });
        }
      } catch (err) {
        console.error("Failed to fetch dream realms:", err);
      } finally {
        fetchDreamRealmsPromise = null;
      }
    };
    fetchDreamRealmsPromise = executeFetch();
    return fetchDreamRealmsPromise;
  },

  fetchActiveDreamSession: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dream-realms/active`);
      if (response.ok) {
        const data = await response.json();
        if (data.active) {
          set({
            activeDreamSession: data.session,
            activeDreamTemplate: data.template,
          });
        } else {
          set({
            activeDreamSession: null,
            activeDreamTemplate: null,
          });
        }
      }
    } catch (err) {
      console.error("Failed to fetch active dream session:", err);
      get().addToast("Lỗi tải thông tin mộng cảnh.", "error");
    }
  },

  enterDreamRealm: async (instanceId: string) => {
    const { token } = get();
    if (!token || !get().beginAction("dream_realm")) return false;
    set({ error: null, dreamActionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dream-realms/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_id: instanceId }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tiến vào mộng cảnh.");
      set({
        activeDreamSession: data.session,
        activeDreamTemplate: data.template,
      });
      get().addToast(data.message || "Tiến vào mộng cảnh thành công!", "success");
      get().addLog(data.message || `Tiến vào mộng cảnh: ${instanceId}`);
      await Promise.all([get().refreshPlayerSummary(), get().fetchDreamRealms()]);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ dreamActionLoading: false });
      get().endAction("dream_realm");
    }
  },

  actionDreamRealm: async (optionIndex: number) => {
    const { token } = get();
    if (!token || !get().beginAction("dream_realm")) return false;
    set({ error: null, dreamActionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dream-realms/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ option_index: optionIndex }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hành động trong mộng cảnh thất bại.");

      if (data.session) {
        set({ activeDreamSession: data.session });
        if (data.session.status !== "ACTIVE") {
          set({
            activeDreamSession: null,
            activeDreamTemplate: null,
          });
        }
      }
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Lựa chọn quyết định thành công!", "success");
      get().addLog(data.message || "Đã hành động trong mộng cảnh.");
      await get().refreshPlayerSummary();
      if (data.session?.status !== "ACTIVE") {
        await get().fetchDreamRealms();
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ dreamActionLoading: false });
      get().endAction("dream_realm");
    }
  },

  exitDreamRealm: async () => {
    const { token } = get();
    if (!token || !get().beginAction("dream_realm")) return false;
    set({ error: null, dreamActionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dream-realms/exit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể rời khỏi mộng cảnh.");

      set({
        activeDreamSession: null,
        activeDreamTemplate: null,
      });
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Đã rời khỏi mộng cảnh.", "info");
      get().addLog(data.message || "Đã rời khỏi mộng cảnh.");
      await Promise.all([get().refreshPlayerSummary(), get().fetchDreamRealms()]);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ dreamActionLoading: false });
      get().endAction("dream_realm");
    }
  },
});
