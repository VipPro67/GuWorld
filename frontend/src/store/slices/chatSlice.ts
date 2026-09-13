import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { wsManager } from "@/lib/websocket";

export interface ChatSlice {
  inbox: any[];
  baoHoangMessages: any[];
  activeChannel: "world" | "general" | "sect";
  chatOpen: boolean;
  chatMessages: { world: any[]; general: any[]; sect: any[] };
  unreadChannels: { world: boolean; general: boolean; sect: boolean };
  setActiveChannel: (channel: "world" | "general" | "sect") => void;
  setChatOpen: (open: boolean) => void;
  subscribeBaoHoangMessages: (channel: string) => void;
  unsubscribeBaoHoangMessages: () => void;
  subscribeAllBaoHoangMessages: () => void;
  unsubscribeAllBaoHoangMessages: () => void;
  sendPrivateMail: (receiverUsername: string, content: string) => Promise<boolean>;
  fetchInbox: () => Promise<void>;
  sendBaoHoangChat: (content: string, channel: string) => Promise<boolean>;
  fetchBaoHoangMessages: (channel: string) => Promise<void>;
}

export const createChatSlice: StateCreator<any, any, any, ChatSlice> = (set, get) => ({
  inbox: [],
  baoHoangMessages: [],
  activeChannel: "world",
  chatOpen: false,
  chatMessages: { world: [], general: [], sect: [] },
  unreadChannels: { world: false, general: false, sect: false },

  setActiveChannel: (channel) => {
    set((state: any) => {
      const newUnread = { ...state.unreadChannels };
      newUnread[channel] = false;
      return {
        activeChannel: channel,
        unreadChannels: newUnread,
        baoHoangMessages: state.chatMessages[channel] || [],
      };
    });
  },

  setChatOpen: (open) => {
    set((state: any) => {
      const active = state.activeChannel || "world";
      const newUnread = { ...state.unreadChannels };
      if (open) newUnread[active] = false;
      return { chatOpen: open, unreadChannels: newUnread };
    });
  },

  subscribeBaoHoangMessages: (channel) => {
    get().subscribeEvents();
    get().setActiveChannel(channel as any);
  },

  unsubscribeBaoHoangMessages: () => {},

  subscribeAllBaoHoangMessages: () => {
    get().subscribeEvents();
  },

  unsubscribeAllBaoHoangMessages: () => {},

  sendPrivateMail: async (receiverUsername, content) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: receiverUsername, content }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi thư thất bại");
      get().addLog(data.message || "Đã gửi thư truyền tin!");
      if (data.player) get().setPlayerFromServer(data.player);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchInbox: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/inbox`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ inbox: Array.isArray(data.inbox) ? data.inbox : [] });
    } catch (err) {
      console.error("fetch inbox failed", err);
      get().addToast("Lỗi tải hộp thư.", "error");
    }
  },

  sendBaoHoangChat: async (content, channel) => {
    const { token, actionLoading } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const data = await wsManager.request<{ message?: string }>("chat.send", { content, channel });
      get().addLog(data.message || "Đã phát truyền tin nhắn!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchBaoHoangMessages: async (channel) => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=${channel}`);
      if (response.status === 401) return;
      const data = await response.json();
      set({ baoHoangMessages: Array.isArray(data.messages) ? data.messages : [] });
    } catch (err) {
      console.error("fetch chat messages failed", err);
      get().addToast("Lỗi tải tin nhắn.", "error");
    }
  },
});
