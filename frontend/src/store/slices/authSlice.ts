import type { StateCreator } from "zustand";
import FingerprintJS from "@fingerprintjs/fingerprintjs";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useUIStore } from "@/store/useUIStore";
import { AUTHENTICATED_SESSION } from "@/store/useGameStore";
import type { AuthResponse, Player } from "@/store/useGameStore";

export interface AuthSlice {
  login: (email: string, password: string) => Promise<boolean>;
  sendOTP: (email: string) => Promise<boolean>;
  getBrowserFingerprint: () => Promise<string>;
  register: (
    username: string,
    password: string,
    email: string,
    startingRegion: string,
    setVitalNow: boolean,
    referralCode?: string,
    attackGuId?: string,
    defenseGuId?: string,
    vitalGuChoice?: string,
    otp?: string,
    race?: string,
    gender?: string
  ) => Promise<boolean>;
  logout: () => void;
  clearError: () => void;
  completeOnboarding: (data: {
    starting_region: string;
    attack_gu_id: string;
    defense_gu_id: string;
    vital_gu_choice: string;
  }) => Promise<boolean>;
}

export const createAuthSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, AuthSlice> = (set, get) => ({
  login: async (email, password) => {
    set({ isLoading: true, error: null });
    try {
      const fingerprint = await get().getBrowserFingerprint();
      const response = await authFetch(`${BACKEND_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, fingerprint }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Mật khẩu hoặc Email không chính xác.");
      }

      const data: AuthResponse = await response.json();
      get().setPlayerFromServer(data.player, { token: data.token || AUTHENTICATED_SESSION });
      if (data.token) {
        localStorage.setItem("gu_auth_token", data.token);
      }
      localStorage.setItem("gu_player_id", data.player.id);

      void get().fetchRecipes();
      void get().fetchGuInventory();
      await get().fetchUnreadNotificationsCount();
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      get().addLog(`Đồng chí ${data.player.username} đã quy vị!`);
      const today = new Date().toISOString().split('T')[0];
      const hiddenUntil = typeof window !== "undefined" ? localStorage.getItem("gu_changelog_hide_until") : null;
      if (hiddenUntil !== today) {
        useUIStore.getState().setShowChangelog(true);
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  sendOTP: async (email) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/send-otp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Gửi OTP thất bại.");
      }

      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  getBrowserFingerprint: async (): Promise<string> => {
    if (typeof window === "undefined") {
      return "server-ssr-fallback";
    }
    try {
      const fp = await FingerprintJS.load();
      const result = await fp.get();
      return result.visitorId;
    } catch (e) {
      return "fingerprint-error";
    }
  },

  register: async (
    username,
    password,
    email,
    startingRegion,
    setVitalNow,
    referralCode,
    attackGuId,
    defenseGuId,
    vitalGuChoice,
    otp,
    race?,
    gender?
  ) => {
    set({ isLoading: true, error: null });
    try {
      const fingerprint = await get().getBrowserFingerprint();
      const response = await authFetch(`${BACKEND_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username,
          password,
          email,
          starting_region: startingRegion,
          set_vital_now: setVitalNow,
          referral_code: referralCode || "",
          attack_gu_id: attackGuId,
          defense_gu_id: defenseGuId,
          vital_gu_choice: vitalGuChoice,
          otp: otp || "",
          race: race || "human",
          gender: gender || "male",
          fingerprint,
        }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể khai mở Không Khiếu.");
      }

      const data: AuthResponse = await response.json();
      get().setPlayerFromServer(data.player, { token: data.token || AUTHENTICATED_SESSION });
      if (data.token) {
        localStorage.setItem("gu_auth_token", data.token);
      }
      localStorage.setItem("gu_player_id", data.player.id);

      void get().fetchRecipes();
      void get().fetchGuInventory();
      await get().fetchUnreadNotificationsCount();
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      get().addLog(`Cổ sư ${data.player.username} khai mở Không Khiếu! Tư chất: ${data.player.aptitude_type} (${data.player.aptitude_percent}%)`);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  completeOnboarding: async (data) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/complete-onboarding`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể hoàn tất thiết lập.");
      }

      const responseData = await response.json();
      const player: Player = responseData.player;
      get().setPlayerFromServer(player);
      get().addLog(`Thiết lập ban đầu hoàn tất! Khu vực: ${player.current_region}`);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  logout: () => {
    get().unsubscribePlayerEvents();
    get().unsubscribeTradeEvents();
    get().unsubscribeBaoHoangMessages();
    void authFetch(`${BACKEND_URL}/api/auth/logout`, { method: "POST" });
    set({ player: null, manaSnapshot: null, token: null, recipes: [], recipesLoaded: true, quests: [], guInventoryLoaded: false, notifications: [], notificationsLoaded: false, unreadNotifications: 0 });
    localStorage.removeItem("gu_auth_token");
    localStorage.removeItem("gu_player_id");
    get().addLog("Đã rời khỏi thế giới Cổ sư.");
  },

  clearError: () => set({ error: null }),
});
