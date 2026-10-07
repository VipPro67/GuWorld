import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { wsManager } from "@/lib/websocket";

export interface ChatSlice {
  inbox: ApiResponseData[];
  baoHoangMessages: ApiResponseData[];
  activeChannel: "world" | "general" | "sect" | "region" | "private";
  chatOpen: boolean;
  chatMessages: { world: ApiResponseData[]; general: ApiResponseData[]; sect: ApiResponseData[]; region: ApiResponseData[] };
  unreadChannels: { world: boolean; general: boolean; sect: boolean; region: boolean; private: boolean };
  setActiveChannel: (channel: "world" | "general" | "sect" | "region" | "private") => void;
  setChatOpen: (open: boolean) => void;
  subscribeBaoHoangMessages: (channel: string) => void;
  unsubscribeBaoHoangMessages: () => void;
  subscribeAllBaoHoangMessages: () => void;
  unsubscribeAllBaoHoangMessages: () => void;
  sendPrivateMail: (receiverUsername: string, content: string) => Promise<boolean>;
  fetchInbox: () => Promise<void>;
  sendBaoHoangChat: (content: string, channel: string) => Promise<boolean>;
  fetchBaoHoangMessages: (channel: string) => Promise<string | null>;
  subscribeRegionChat: (channelKey: string) => () => void;
  getCurrentRegionChatChannel: () => Promise<string | null>;
  privateConversations: ApiResponseData[];
  privateMessages: ApiResponseData[];
  activePrivateUsername: string;
  fetchPrivateConversations: () => Promise<void>;
  fetchPrivateThread: (username: string) => Promise<void>;
  sendPrivateChat: (username: string, content: string) => Promise<boolean>;
  onPrivateChatEvent: (payload: ApiResponseData) => void;
}

export const createChatSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ChatSlice> = (set, get) => ({
  inbox: [],
  baoHoangMessages: [],
  activeChannel: "world",
  chatOpen: false,
  chatMessages: { world: [], general: [], sect: [], region: [], private: [] },
  unreadChannels: { world: false, general: false, sect: false, region: false, private: false },
  privateConversations: [],
  privateMessages: [],
  activePrivateUsername: "",

  setActiveChannel: (channel) => {
    set((state: ApiResponseData) => {
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
    set((state: ApiResponseData) => {
      const active = state.activeChannel || "world";
      const newUnread = { ...state.unreadChannels };
      if (open) newUnread[active] = false;
      return { chatOpen: open, unreadChannels: newUnread };
    });
  },

  subscribeBaoHoangMessages: (channel) => {
    get().subscribeEvents();
    get().setActiveChannel(channel as ApiResponseData);
  },

  unsubscribeBaoHoangMessages: () => {},

  subscribeAllBaoHoangMessages: () => {
    get().subscribeEvents();
  },

  subscribeRegionChat: (channelKey) => wsManager.subscribe(channelKey, (payload: ApiResponseData) => {
    set((state: ApiResponseData) => {
      const existing = state.chatMessages.region || [];
      const incoming = payload?.message ? [...existing, payload.message] : payload?.messages;
      if (!Array.isArray(incoming)) return {};
      const deduped = incoming.filter((message: ApiResponseData, index: number, all: ApiResponseData[]) => all.findIndex((candidate) => candidate.id === message.id) === index);
      const unreadChannels = { ...state.unreadChannels };
      if (state.activeChannel !== "region" || !state.chatOpen) unreadChannels.region = true;
      return {
        chatMessages: { ...state.chatMessages, region: deduped },
        unreadChannels,
        baoHoangMessages: state.activeChannel === "region" ? deduped : state.baoHoangMessages,
      };
    });
  }),

  getCurrentRegionChatChannel: async () => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=region&scope_only=1`);
      if (!response.ok) return null;
      const data = await response.json();
      return typeof data.channel_key === "string" ? data.channel_key : null;
    } catch (err) {
      console.error("resolve current region chat failed", err);
      return null;
    }
  },

  fetchPrivateConversations: async () => {
    if (!get().token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/conversations`);
      if (!response.ok) throw new Error("Không thể tải danh sách hội thoại riêng");
      const data = await response.json();
      set({ privateConversations: Array.isArray(data.conversations) ? data.conversations : [] });
    } catch (err) {
      console.error("fetch private conversations failed", err);
      get().addToast("Lỗi tải danh sách tin nhắn riêng.", "error");
    }
  },

  fetchPrivateThread: async (username) => {
    const cleanUsername = username.trim();
    if (!get().token || !cleanUsername) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/thread?username=${encodeURIComponent(cleanUsername)}`);
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tải hội thoại");
      const messages = Array.isArray(data.messages) ? data.messages : [];
      set((state: ApiResponseData) => ({
        activePrivateUsername: data.username || cleanUsername,
        privateMessages: messages,
        baoHoangMessages: state.activeChannel === "private" ? messages : state.baoHoangMessages,
        unreadChannels: { ...state.unreadChannels, private: false },
      }));
      void get().fetchPrivateConversations();
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message || "Lỗi tải hội thoại riêng.", "error");
    }
  },

  sendPrivateChat: async (username, content) => {
    const { token } = get();
    const cleanUsername = username.trim();
    if (!token || !cleanUsername || !get().beginAction("chat")) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: cleanUsername, content }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi tin nhắn riêng thất bại");
      get().addLog(data.message || "Đã gửi tin nhắn riêng.");
      await get().fetchPrivateThread(cleanUsername);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Gửi tin nhắn riêng thất bại.", "error");
      return false;
    } finally { get().endAction("chat"); }
  },

  onPrivateChatEvent: (payload) => {
    const message = payload?.message;
    const username = payload?.conversation_username;
    if (!message || typeof username !== "string") return;
    set((state: ApiResponseData) => {
      const mine = message.sender_id === state.player?.id;
      const unreadChannels = { ...state.unreadChannels };
      const isOpenThread = state.activeChannel === "private" && state.chatOpen && state.activePrivateUsername === username;
      if (!mine && !isOpenThread) unreadChannels.private = true;
      const conversations = [...(state.privateConversations || [])];
      const index = conversations.findIndex((entry: ApiResponseData) => entry.username === username);
      const previous = index >= 0 ? conversations[index] : { username, unread_count: 0 };
      const updated = {
        ...previous,
        last_message: message.content,
        last_message_at: message.created_at,
        last_message_mine: mine,
        unread_count: mine || isOpenThread ? 0 : (Number(previous.unread_count) || 0) + 1,
      };
      if (index >= 0) conversations.splice(index, 1);
      conversations.unshift(updated);
      const privateMessages = isOpenThread && !state.privateMessages.some((entry: ApiResponseData) => entry.id === message.id)
        ? [...state.privateMessages, message]
        : state.privateMessages;
      return {
        privateConversations: conversations,
        privateMessages,
        baoHoangMessages: isOpenThread ? privateMessages : state.baoHoangMessages,
        unreadChannels,
      };
    });
    const state = get();
    if (message.sender_id !== state.player?.id && state.activeChannel === "private" && state.chatOpen && state.activePrivateUsername === username) {
      void get().fetchPrivateThread(username);
    }
  },

  unsubscribeAllBaoHoangMessages: () => {},

  sendPrivateMail: async (receiverUsername, content) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("chat")) return false;
    set({ error: null });
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
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { get().endAction("chat"); }
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
    const { token } = get();
    if (!token || !get().beginAction("chat")) return false;
    set({ error: null });
    try {
      const data = await wsManager.request<{ message?: string }>("chat.send", { content, channel });
      get().addLog(data.message || "Đã phát truyền tin nhắn!");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { get().endAction("chat"); }
  },

  fetchBaoHoangMessages: async (channel) => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=${encodeURIComponent(channel)}`);
      if (response.status === 401) return null;
      const data = await response.json();
      const messages = Array.isArray(data.messages) ? data.messages : [];
      set((state: ApiResponseData) => ({
        chatMessages: { ...state.chatMessages, [channel]: messages },
        baoHoangMessages: state.activeChannel === channel ? messages : state.baoHoangMessages,
      }));
      return typeof data.channel_key === "string" ? data.channel_key : null;
    } catch (err) {
      console.error("fetch chat messages failed", err);
      get().addToast("Lỗi tải tin nhắn.", "error");
      return null;
    }
  },
});
