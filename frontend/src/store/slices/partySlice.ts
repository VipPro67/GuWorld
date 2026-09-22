import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface PartySlice {
  party: { id: string; leader_id: string; members: { id: string; username: string; avatar_url?: string }[] } | null;
  partyInvitations: { id: string; party_id: string; inviter_name: string; inviter_avatar?: string }[];
  createParty: () => Promise<boolean>;
  fetchPartyStatus: () => Promise<void>;
  fetchPartyInvitations: () => Promise<void>;
  invitePlayer: (username: string) => Promise<boolean>;
  respondPartyInvitation: (id: string, action: "accept" | "reject") => Promise<boolean>;
  leaveParty: () => Promise<boolean>;
}

export const createPartySlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, PartySlice> = (set, get) => ({
  party: null,
  partyInvitations: [],

  createParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/create`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo tổ đội thất bại");
      set({ party: data.party });
      get().addLog(data.message || "Đã lập tổ đội thành công!");
      await get().refreshPlayerSummary();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyStatus: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/status`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ party: data.party });
    } catch (err) {
      console.error("fetch party status failed", err);
      get().addToast("Lỗi tải trạng thái đội.", "error");
    }
  },

  invitePlayer: async (username) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invite`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể mời");
      set({ party: data.party });
      get().addLog(data.message || `Đã mời người chơi ${username} vào tổ đội.`);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  leaveParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/leave`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tổ đội thất bại");
      set({ party: null });
      get().addLog(data.message || "Bạn đã rời khỏi tổ đội.");
      await get().refreshPlayerSummary();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyInvitations: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invitations`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ partyInvitations: data.invitations || [] });
    } catch (err) {
      console.error("fetch party invitations failed", err);
    }
  },

  respondPartyInvitation: async (id, action) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invitations/${id}/respond`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      if (action === "accept") {
        set({ party: data.party });
        await get().refreshPlayerSummary();
      }
      set((state: ApiResponseData) => ({
        partyInvitations: state.partyInvitations.filter((inv: ApiResponseData) => inv.id !== id),
      }));
      get().addLog(data.message || `Đã ${action === "accept" ? "chấp nhận" : "từ chối"} lời mời.`);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },
});
