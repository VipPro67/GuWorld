import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface AchievementsSlice {
  achievements: ApiResponseData[];
  earnedAchievements: ApiResponseData[];
  selectedTitle: string;
  selectedFrame: string;
  dailyMissions: ApiResponseData[];
  dailyMissionsResetAt: string;
  fetchAchievements: () => Promise<void>;
  fetchDailyMissions: () => Promise<void>;
  claimDailyMission: (missionId: string) => Promise<{ success: boolean; message: string } | null>;
  claimAchievement: (achievementId: string) => Promise<{ success: boolean; message: string } | null>;
  selectTitleFrame: (title: string, frame: string) => Promise<{ success: boolean; message: string } | null>;
}

export const createAchievementsSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, AchievementsSlice> = (set, get) => ({
  achievements: [],
  earnedAchievements: [],
  selectedTitle: '',
  selectedFrame: '',
  dailyMissions: [],
  dailyMissionsResetAt: '',

  fetchDailyMissions: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/daily-missions`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ dailyMissions: data.missions || [], dailyMissionsResetAt: data.reset_at || '' });
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Daily mission fetch error:", err.message);
      get().addToast("Lỗi tải nhiệm vụ hằng ngày.", "error");
    }
  },

  claimDailyMission: async (missionId) => {
    const { token, logout } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/daily-missions/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mission_id: missionId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng nhiệm vụ thất bại.");
      await Promise.all([get().fetchDailyMissions(), get().refreshPlayerSummary()]);
      return { success: true, message: data.message || "Đã nhận thưởng nhiệm vụ." };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return { success: false, message: err.message };
    }
  },

  fetchAchievements: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ achievements: data.achievements || [], earnedAchievements: data.earned || [], selectedTitle: data.selected_title || '', selectedFrame: data.selected_frame || '' });
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Achievements fetch error:", err.message);
      get().addToast("Lỗi tải thành tích.", "error");
    }
  },

  claimAchievement: async (achievementId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("achievements")) return null;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ achievement_id: achievementId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Nhận thưởng thất bại."); }
      get().applyInventoryFromResponse(data);
      get().fetchAchievements();
      get().addLog(data.message || "Nhận thưởng thành tựu thành công!");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Nhận thưởng thành tựu lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { get().endAction("achievements"); }
  },

  selectTitleFrame: async (title, frame) => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements/select`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title, frame }),
      });
      if (response.status === 401) return null;
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật thất bại.");
      set({ selectedTitle: title, selectedFrame: frame });
      get().addLog(data.message || "Đã cập nhật danh hiệu và frame.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Select title/frame error:", err.message);
      get().addToast("Lỗi chọn danh hiệu.", "error");
      return { success: false, message: err.message };
    }
  },
});
