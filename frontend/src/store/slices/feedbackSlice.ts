import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { Feedback, FeedbackComment } from "@/store/useGameStore";

export interface FeedbackSlice {
  feedbacks: Feedback[];
  feedbackComments: Record<string, FeedbackComment[]>;
  fetchFeedbacks: (category?: string, status?: string) => Promise<void>;
  submitFeedback: (title: string, description: string, category: string) => Promise<boolean>;
  voteFeedback: (feedbackId: string, voteType: number) => Promise<boolean>;
  updateFeedbackStatus: (feedbackId: string, status: string, developerResponse: string) => Promise<boolean>;
  fetchFeedbackComments: (feedbackId: string) => Promise<boolean>;
  addFeedbackComment: (feedbackId: string, content: string) => Promise<boolean>;
  deleteFeedbackComment: (feedbackId: string, commentId: string) => Promise<boolean>;
}

export const createFeedbackSlice: StateCreator<any, any, any, FeedbackSlice> = (set, get) => ({
  feedbacks: [],
  feedbackComments: {},

  fetchFeedbacks: async (category, status) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      let url = `${BACKEND_URL}/api/feedback`;
      const params = [];
      if (category) params.push(`category=${encodeURIComponent(category)}`);
      if (status) params.push(`status=${encodeURIComponent(status)}`);
      if (params.length > 0) {
        url += `?${params.join("&")}`;
      }
      const response = await authFetch(url);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ feedbacks: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch feedbacks:", err);
      get().addToast("Lỗi tải phản hồi.", "error");
    }
  },

  submitFeedback: async (title, description, category) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title, description, category }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi góp ý thất bại.");
      get().addToast(data.message || "Gửi góp ý thành công!", "success");
      await get().fetchFeedbacks();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  voteFeedback: async (feedbackId, voteType) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });

    // Optimistic UI update
    const prevFeedbacks = [...get().feedbacks];
    const updatedFeedbacks = prevFeedbacks.map((f) => {
      if (f.id === feedbackId) {
        const prevVote = f.user_vote;
        let newTotal = f.total_votes;
        if (prevVote === undefined || prevVote === null) {
          newTotal += voteType;
        } else if (prevVote === voteType) {
          newTotal -= voteType; // Toggle off
        } else {
          newTotal += 2 * voteType; // Changed vote type
        }
        return {
          ...f,
          total_votes: newTotal,
          user_vote: prevVote === voteType ? null : voteType,
        };
      }
      return f;
    });
    set({ feedbacks: updatedFeedbacks });

    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/vote`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ vote_type: voteType }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bình chọn thất bại.");
      get().addToast(data.message || "Bình chọn thành công!", "success");
      // Update with exact values from server
      set((state: any) => ({
        feedbacks: state.feedbacks.map((f: any) =>
          f.id === feedbackId ? { ...f, total_votes: data.total_votes, user_vote: data.user_vote } : f
        ),
      }));
      return true;
    } catch (err: any) {
      // Revert on error
      set({ feedbacks: prevFeedbacks, error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  updateFeedbackStatus: async (feedbackId, status, developerResponse) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status, developer_response: developerResponse }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật trạng thái thất bại.");
      get().addToast(data.message || "Cập nhật trạng thái thành công!", "success");
      await get().fetchFeedbacks();
      // Also refresh player stats in case they received a reward
      if (get().player) {
        await get().fetchPlayerStatus(get().player!.id);
      }
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchFeedbackComments: async (feedbackId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments`);
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tải góp ý thất bại.");
      set((state: any) => ({ feedbackComments: { ...state.feedbackComments, [feedbackId]: data || [] } }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    }
  },

  addFeedbackComment: async (feedbackId, content) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Góp ý thất bại.");
      get().addToast(data.message || "Đã góp ý!", "success");
      await get().fetchFeedbackComments(feedbackId);
      set((state: any) => ({
        feedbacks: state.feedbacks.map((f: any) =>
          f.id === feedbackId ? { ...f, comment_count: f.comment_count + 1 } : f
        ),
      }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  deleteFeedbackComment: async (feedbackId, commentId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments/${commentId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xóa góp ý thất bại.");
      get().addToast(data.message || "Đã xóa góp ý!", "success");
      await get().fetchFeedbackComments(feedbackId);
      set((state: any) => ({
        feedbacks: state.feedbacks.map((f: any) =>
          f.id === feedbackId ? { ...f, comment_count: Math.max(0, f.comment_count - 1) } : f
        ),
      }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },
});
