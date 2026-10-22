import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { PlayerInstanceSession, InstanceTemplate, InstanceListItem } from "@/store/useGameStore";

export interface InstanceSlice {
  activeInstanceSession: PlayerInstanceSession | null;
  activeInstanceTemplate: InstanceTemplate | null;
  availableInstances: InstanceListItem[];
  instanceDailyEntriesUsed: number;
  instanceDailyEntriesMax: number;
  fetchInstances: () => Promise<void>;
  fetchActiveInstance: () => Promise<void>;
  enterInstance: (instanceId: string) => Promise<boolean>;
  actionInstance: (optionIndex: number) => Promise<boolean>;
}

let fetchInstancesPromise: Promise<void> | null = null;

export const createInstanceSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, InstanceSlice> = (set, get) => ({
  activeInstanceSession: null,
  activeInstanceTemplate: null,
  availableInstances: [],
  instanceDailyEntriesUsed: 0,
  instanceDailyEntriesMax: 2,

  fetchInstances: async () => {
    if (fetchInstancesPromise) return fetchInstancesPromise;
    const executeFetch = async () => {
      const { token } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/instance/list`);
        if (response.ok) {
          const data = await response.json();
          set({
            availableInstances: data.instances || [],
            instanceDailyEntriesUsed: data.daily_entries_used ?? 0,
            instanceDailyEntriesMax: data.daily_entries_max ?? 2,
          });
        }
      } catch (err) {
        console.error("Failed to fetch instances:", err);
      } finally {
        fetchInstancesPromise = null;
      }
    };
    fetchInstancesPromise = executeFetch();
    return fetchInstancesPromise;
  },

  fetchActiveInstance: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/active`);
      if (response.ok) {
        const data = await response.json();
        if (data.active) {
          set({
            activeInstanceSession: data.session,
            activeInstanceTemplate: data.template,
          });
        } else {
          set({
            activeInstanceSession: null,
            activeInstanceTemplate: null,
          });
        }
      }
    } catch (err) {
      console.error("Failed to fetch active instance:", err);
      get().addToast("Lỗi tải thông tin thám hiểm.", "error");
    }
  },

  enterInstance: async (instanceId: string) => {
    const { token } = get();
    if (!token || !get().beginAction("instance")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_id: instanceId }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tiến vào thám hiểm.");
      set({
        activeInstanceSession: data.session,
        activeInstanceTemplate: data.template,
      });
      get().addToast(data.message || "Tiến vào thành công!", "success");
      get().addLog(data.message || `Tiến vào: ${instanceId}`);
      await Promise.all([get().refreshPlayerSummary(), get().fetchInstances()]);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("instance"); }
  },

  actionInstance: async (optionIndex: number) => {
    const { token } = get();
    if (!token || !get().beginAction("instance")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ option_index: optionIndex }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hành động thám hiểm thất bại.");

      if (data.session) {
        set({ activeInstanceSession: data.session });
        if (data.session.status !== "ACTIVE") {
          set({
            activeInstanceSession: null,
            activeInstanceTemplate: null,
          });
        }
      }
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Thực hiện hành động thành công!", "success");
      get().addLog(data.message || "Đã hành động trong thám hiểm.");
      await get().refreshPlayerSummary();
      if (data.session?.status !== "ACTIVE") {
        await get().fetchInstances();
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("instance"); }
  },
});
