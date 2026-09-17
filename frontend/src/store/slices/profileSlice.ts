import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useUIStore } from "@/store/useUIStore";
import { AUTHENTICATED_SESSION } from "@/store/useGameStore";
import type { Player } from "@/store/useGameStore";

const lastStatusFetchTimes = new Map<string, number>();
const pendingStatusFetches = new Map<string, Promise<void>>();
const STATUS_FETCH_TTL_MS = 3000;

export interface ProfileSlice {
  inspectedProfile: ApiResponseData | null;
  inspectedUsernameQuery: string | null;
  profileLoading: boolean;
  fetchPlayerProfile: (username: string) => Promise<void>;
  clearInspectedProfile: () => void;
  updateProfileSettings: (settings: {
    username: string;
    bio: string;
    new_password?: string;
    fake_name?: string;
    fake_rank?: number;
    fake_sub_rank?: number;
    fake_vital_gu?: string;
    is_disguised?: boolean;
  }) => Promise<boolean>;
  uploadAvatar: (file: File) => Promise<string | null>;
  fetchPlayerStatus: (id: string, forceRefresh?: boolean) => Promise<void>;
  initializeSession: () => Promise<void>;
}

export const createProfileSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ProfileSlice> = (set, get) => ({
  inspectedProfile: null,
  inspectedUsernameQuery: null,
  profileLoading: false,

  initializeSession: async () => {
    if (typeof window !== "undefined") {
      const urlParams = new URLSearchParams(window.location.search);
      const tokenFromUrl = urlParams.get("token");
      if (tokenFromUrl) {
        localStorage.setItem("gu_auth_token", tokenFromUrl);
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
      }
      const errorFromUrl = urlParams.get("error");
      if (errorFromUrl) {
        set({ error: errorFromUrl });
        get().addToast(errorFromUrl, "error");
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
      }
    }

    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/me`);
      if (response.status === 401) {
        set({ player: null, manaSnapshot: null, token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to restore session");
      }

      const player: Player = await response.json();
      const localToken = (typeof window !== "undefined" ? localStorage.getItem("gu_auth_token") : null) || AUTHENTICATED_SESSION;
      set({ activeCombatReplay: null });
      get().setPlayerFromServer(player, { token: localToken });
      if (typeof window !== "undefined") {
        localStorage.setItem("gu_player_id", player.id);
      }
      void get().fetchMySectStatus();
      void get().fetchGuInventory();
      await get().fetchUnreadNotificationsCount();
      try {
        get().subscribePlayerEvents();
        get().subscribeTradeEvents();
      } catch (wsErr) {
        console.warn("[WebSocket] Failed to subscribe events during session restore:", wsErr);
      }
      const today = new Date().toISOString().split('T')[0];
      const hiddenUntil = typeof window !== "undefined" ? localStorage.getItem("gu_changelog_hide_until") : null;
      if (hiddenUntil !== today) {
        useUIStore.getState().setShowChangelog(true);
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Session restore failed:", err.message);
      set({ player: null, manaSnapshot: null, token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 });
    }
  },

  fetchPlayerStatus: async (id, forceRefresh = false) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return;

    const now = Date.now();
    const lastFetch = lastStatusFetchTimes.get(id) || 0;
    if (!forceRefresh && now - lastFetch < STATUS_FETCH_TTL_MS) {
      return;
    }

    const inFlight = pendingStatusFetches.get(id);
    if (inFlight) return inFlight;

    const executeFetch = async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/status/${id}`);

        if (response.status === 401) {
          logout();
          return;
        }

        if (!response.ok) {
          throw new Error("Failed to fetch status");
        }

        lastStatusFetchTimes.set(id, Date.now());
        const player: Player = await response.json();
        const currentPlayer = get().player;
        if (currentPlayer && currentPlayer.id === id) {
          // The status endpoint only projects equipped/vital Gus. Preserve the
          // full inventory already in state so it is not clobbered by the sparse
          // snapshot.
          get().setPlayerFromServer({
            ...player,
            gu_inventory: currentPlayer.gu_inventory?.length ? currentPlayer.gu_inventory : player.gu_inventory,
            equipped_gus: currentPlayer.gu_inventory?.length ? currentPlayer.equipped_gus : player.equipped_gus,
          });
        }
      } catch (caught: unknown) {
        const err = caught instanceof Error ? caught : new Error(String(caught));
        console.error("Status fetch failed:", err.message);
      } finally {
        pendingStatusFetches.delete(id);
      }
    };

    const promise = executeFetch();
    pendingStatusFetches.set(id, promise);
    return promise;
  },


  fetchPlayerProfile: async (username) => {
    const { token, logout } = get();
    if (!token) return;

    set({ profileLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/profile/${encodeURIComponent(username)}`);

      if (response.status === 401) {
        logout();
        return;
      }

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to load player profile");
      }

      const data = await response.json();
      set({ inspectedProfile: data, inspectedUsernameQuery: username });
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Profile fetch failed:", err.message);
      get().addToast("Lỗi tải hồ sơ.", "error");
      get().addToast(`Không thể trinh thám hồ sơ: ${err.message}`, "error");
    } finally {
      set({ profileLoading: false });
    }
  },

  clearInspectedProfile: () => set({ inspectedProfile: null, inspectedUsernameQuery: null }),

  updateProfileSettings: async (settings) => {
    const { token, actionLoading, addToast } = get();
    if (!token || actionLoading) return false;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/profile/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(settings),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Failed to update profile settings");
      }

      get().setPlayerFromServer(data.player);
      addToast(data.message || "Cập nhật hồ sơ thành công!", "success");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Profile settings update failed:", err.message);
      get().addToast("Lỗi cập nhật hồ sơ.", "error");
      set({ error: err.message });
      addToast(`Cập nhật thất bại: ${err.message}`, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  uploadAvatar: async (file: File) => {
    const { token, actionLoading, addToast } = get();
    if (!token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const formData = new FormData();
      formData.append("avatar", file);

      const response = await authFetch(`${BACKEND_URL}/api/player/avatar`, {
        method: "POST",
        body: formData,
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Upload thất bại");
      }

      // Update player avatar_url in store
      const { player } = get();
      if (player) {
        set({ player: { ...player, avatar_url: data.avatar_url } });
      }
      addToast(data.message || "Cập nhật ảnh đại diện thành công!", "success");
      return data.avatar_url as string;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Avatar upload failed:", err.message);
      get().addToast("Lỗi tải ảnh đại diện.", "error");
      set({ error: err.message });
      addToast(`Upload thất bại: ${err.message}`, "error");
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },
});
