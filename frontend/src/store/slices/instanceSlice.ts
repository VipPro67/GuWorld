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

export const createInstanceSlice: StateCreator<any, any, any, InstanceSlice> = (set, get) => ({
  activeInstanceSession: null,
  activeInstanceTemplate: null,
  availableInstances: [],
  instanceDailyEntriesUsed: 0,
  instanceDailyEntriesMax: 2,

  fetchInstances: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/list`);
      if (response.status === 401) { logout(); return; }
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
    }
  },

  fetchActiveInstance: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/active`);
      if (response.status === 401) { logout(); return; }
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
      get().addToast("Lỗi tải phó bản.", "error");
    }
  },

  enterInstance: async (instanceId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_id: instanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tiến vào phó bản.");
      set({
        activeInstanceSession: data.session,
        activeInstanceTemplate: data.template,
      });
      get().addToast(data.message || "Tiến vào phó bản thành công!", "success");
      get().addLog(data.message || `Tiến vào phó bản: ${instanceId}`);
      await get().initializeSession();
      await get().fetchInstances();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  actionInstance: async (optionIndex: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ option_index: optionIndex }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hành động phó bản thất bại.");

      if (data.session) {
        set({ activeInstanceSession: data.session });
        if (data.session.status !== "ACTIVE") {
          set({
            activeInstanceSession: null,
            activeInstanceTemplate: null,
          });
        }
      }
      get().addToast(data.message || "Thực hiện hành động thành công!", "success");
      get().addLog(data.message || "Đã hành động trong phó bản.");
      await get().initializeSession();
      if (data.session?.status !== "ACTIVE") {
        await get().fetchInstances();
      }
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },
});
