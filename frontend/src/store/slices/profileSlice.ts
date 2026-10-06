import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useUIStore } from "@/store/useUIStore";
import { AUTHENTICATED_SESSION } from "@/store/useGameStore";
import type { Player } from "@/store/useGameStore";
import { shouldApplyReconciliation, shouldRefreshCollections } from "@/lib/player-reconciliation";

const lastStatusFetchTimes = new Map<string, number>();
const pendingStatusFetches = new Map<string, Promise<void>>();
const STATUS_FETCH_TTL_MS = 3000;
let refreshPlayerSummaryPromise: Promise<void> | null = null;

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
  reconcilePlayer: (forceCollections?: boolean) => Promise<void>;
  initializeSession: () => Promise<void>;
  refreshPlayerSummary: () => Promise<void>;
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
		    get().setPlayerFromServer(null, { token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 });
        set((state: ApiResponseData) => ({ player: null, playerRevision: state.playerRevision + 1, manaSnapshot: null, token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 }));
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
  	  get().setPlayerFromServer(null, { token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 });
      set((state: ApiResponseData) => ({ player: null, playerRevision: state.playerRevision + 1, manaSnapshot: null, token: null, notifications: [], notificationsLoaded: false, unreadNotifications: 0 }));
    }
  },

  refreshPlayerSummary: async () => {
    if (refreshPlayerSummaryPromise) return refreshPlayerSummaryPromise;

    const executeRefresh = async () => {
      const { token, logout } = get();
      if (!token) return;

      try {
        const response = await authFetch(`${BACKEND_URL}/api/auth/me`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (!response.ok) throw new Error("Failed to refresh player summary");

        const player: Player = await response.json();
        if (get().token !== token) return;
        const currentPlayer = get().player;
        get().setPlayerFromServer({
          ...player,
          // This endpoint carries a sparse Gu projection and also happens to
          // include recipe progress. Those resources have dedicated refreshes.
          gu_inventory: currentPlayer?.gu_inventory ?? player.gu_inventory,
          equipped_gus: currentPlayer?.equipped_gus ?? player.equipped_gus,
          unlocked_recipes: currentPlayer?.unlocked_recipes ?? player.unlocked_recipes,
        });
      } catch (caught: unknown) {
        const err = caught instanceof Error ? caught : new Error(String(caught));
        console.error("Player summary refresh failed:", err.message);
      } finally {
        refreshPlayerSummaryPromise = null;
      }
    };

    refreshPlayerSummaryPromise = executeRefresh();
    return refreshPlayerSummaryPromise;
  },

  fetchPlayerStatus: async (id, forceRefresh = false) => {
    const { token, actionLoading, logout } = get();
    if (!token || (actionLoading && !forceRefresh)) return;

    const now = Date.now();
    const lastFetch = lastStatusFetchTimes.get(id) || 0;
    if (!forceRefresh && now - lastFetch < STATUS_FETCH_TTL_MS) {
      return;
    }

    const inFlight = pendingStatusFetches.get(id);
    if (inFlight) return inFlight;
    const revision = get().playerRevision;

    const executeFetch = async () => {
      const revisionAtStart = get().playerStateRevision as number | null;
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
          const appliedRevision = get().playerStateRevision as number | null;
          // A status request may have started before a mutation. Never let that
          // older projection overwrite the mutation response or realtime event.
          if (player.state_revision < (appliedRevision ?? revisionAtStart ?? -1)) return;
          // The status endpoint only projects equipped/vital Gus. Preserve the
          // full inventory already in state so it is not clobbered by the sparse
          // snapshot.
          get().setPlayerFromServer({
            ...player,
            gu_inventory: currentPlayer.gu_inventory?.length ? currentPlayer.gu_inventory : player.gu_inventory,
            equipped_gus: currentPlayer.gu_inventory?.length ? currentPlayer.equipped_gus : player.equipped_gus,
          }, undefined, revision);
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

  reconcilePlayer: async (forceCollections = false) => {
    const { player, token } = get();
    if (!player || !token) return;

    const requestRevision = (get().reconciliationRevision ?? 0) + 1;
    set({ reconciliationRevision: requestRevision, reconciliationInFlight: true });
    const clientVersion = player.version ?? 0;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/status/${player.id}`);
      if (response.status === 401) {
        if (requestRevision === get().reconciliationRevision) get().logout();
        return;
      }
      if (!response.ok) throw new Error("Failed to reconcile player");

      const serverPlayer: Player = await response.json();
      const currentPlayer = get().player;
      if (!currentPlayer || currentPlayer.id !== player.id) return;
      const serverVersion = serverPlayer.version ?? 0;
      if (!shouldApplyReconciliation(requestRevision, get().reconciliationRevision, serverVersion, currentPlayer.version ?? 0)) return;

      get().setPlayerFromServer({
        ...serverPlayer,
        gu_inventory: currentPlayer.gu_inventory,
        equipped_gus: currentPlayer.equipped_gus,
      });

      // Lifecycle refreshes are deliberately authoritative because several
      // collection tables are updated independently from the player row. A
      // version advance is also sufficient to identify a missed event burst.
      if (shouldRefreshCollections(forceCollections, clientVersion, serverVersion)) {
        await Promise.all([
          get().fetchGuInventory(true, requestRevision),
          get().fetchRecipes(true),
        ]);
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Player reconciliation failed:", err.message);
    } finally {
      if (requestRevision === get().reconciliationRevision) {
        set({ reconciliationInFlight: false });
      }
    }
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
    const { token, addToast } = get();
    if (!token || !get().beginAction("player:profile")) return false;

    set({ error: null });
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
      get().endAction("player:profile");
    }
  },

  uploadAvatar: async (file: File) => {
    const { token, addToast } = get();
    if (!token || !get().beginAction("player:profile")) return null;

    set({ error: null });
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
        get().setPlayerFromServer({ avatar_url: data.avatar_url });
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
      get().endAction("player:profile");
    }
  },
});
