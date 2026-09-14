import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { fetchWithDedupe } from "@/lib/fetch";
import { BACKEND_URL } from "@/lib/config";
import type { PlayerNotification } from "@/store/useGameStore";

export interface NotificationsSlice {
  notifications: PlayerNotification[];
  notificationsLoaded: boolean;
  unreadNotifications: number;
  fetchNotifications: () => Promise<void>;
  fetchUnreadNotificationsCount: () => Promise<void>;
  markNotificationRead: (id: string) => Promise<void>;
  markAllNotificationsRead: () => Promise<void>;
  _onNotificationEvent: (payload: PlayerNotification) => void;
}

export const createNotificationsSlice: StateCreator<any, any, any, NotificationsSlice> = (set, get) => ({
  notifications: [],
  notificationsLoaded: false,
  unreadNotifications: 0,

  fetchNotifications: async () => {
    const { token, logout } = get();
    if (!token) return;
    await fetchWithDedupe("fetchNotifications", async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/notifications?page=1&pageSize=50`);
        if (response.status === 401) { logout(); return; }
        const data = await response.json();
        const list = Array.isArray(data.notifications) ? data.notifications : [];
        set({
          notifications: list,
          notificationsLoaded: true,
        });
      } catch (err) {
        console.error("fetch notifications failed", err);
      }
    });
  },

  fetchUnreadNotificationsCount: async () => {
    const { token } = get();
    if (!token) return;
    await fetchWithDedupe("fetchUnreadNotificationsCount", async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/notifications/unread-count`);
        if (!response.ok) return;
        const data = await response.json();
        set({ unreadNotifications: typeof data.unread === "number" ? data.unread : 0 });
      } catch (err) {
        console.error("fetch unread count failed", err);
      }
    });
  },

  markNotificationRead: async (id) => {
    const { token } = get();
    if (!token) return;
    set((state: any) => ({
      notifications: state.notifications.map((n: PlayerNotification) => (n.id === id ? { ...n, is_read: true } : n)),
      unreadNotifications: Math.max(0, state.unreadNotifications - 1),
    }));
    try {
      await authFetch(`${BACKEND_URL}/api/player/notifications/read`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids: [id] }),
      });
    } catch (err) {
      console.error("mark notification read failed", err);
    }
  },

  markAllNotificationsRead: async () => {
    const { token } = get();
    if (!token) return;
    set((state: any) => ({
      notifications: state.notifications.map((n: PlayerNotification) => ({ ...n, is_read: true })),
      unreadNotifications: 0,
    }));
    try {
      await authFetch(`${BACKEND_URL}/api/player/notifications/read-all`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
    } catch (err) {
      console.error("mark all notifications read failed", err);
    }
  },

  _onNotificationEvent: (payload: PlayerNotification) => {
    if (!payload?.id) return;
    set((state: any) => {
      const exists = state.notifications.some((n: PlayerNotification) => n.id === payload.id);
      if (exists) return {};
      return {
        notifications: [payload, ...state.notifications].slice(0, 50),
        unreadNotifications: state.unreadNotifications + 1,
      };
    });
  },
});
