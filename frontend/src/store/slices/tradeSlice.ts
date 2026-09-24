import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface TradeSlice {
  activeTrade: ApiResponseData | null;
  fetchActiveTrade: () => Promise<void>;
  initiateTrade: (receiverUsername: string) => Promise<boolean>;
  acceptTrade: (tradeId: string) => Promise<boolean>;
  declineTrade: (tradeId: string) => Promise<boolean>;
  cancelTrade: (tradeId: string) => Promise<boolean>;
  updateTradeOffer: (tradeId: string, offer: { primeval_stones: number; immortal_stones: number; gus: string[]; materials: Record<string, number>; recipes: Record<string, number> }) => Promise<boolean>;
  lockTrade: (tradeId: string) => Promise<boolean>;
  unlockTrade: (tradeId: string) => Promise<boolean>;
  confirmTrade: (tradeId: string) => Promise<boolean>;
}

export const createTradeSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, TradeSlice> = (set, get) => ({
  activeTrade: null,

  fetchActiveTrade: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/active`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data = await response.json();
      set({ activeTrade: data.trade });
    } catch (err) {
      console.error("Failed to fetch active trade", err);
      get().addToast("Lỗi tải giao dịch.", "error");
    }
  },

  initiateTrade: async (receiverUsername) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/initiate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: receiverUsername }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi yêu cầu giao dịch thất bại");
      get().addLog(data.message || "Yêu cầu giao dịch đã được gửi!");
      get().addToast(data.message || "Yêu cầu giao dịch đã được gửi!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Giao dịch thất bại: ${err.message}`);
      return false;
    } finally {
      get().endAction("player:inventory");
    }
  },

  acceptTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/accept/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chấp nhận giao dịch thất bại");
      get().addLog(data.message || "Đã chấp nhận giao dịch!");
      get().addToast("Đã thiết lập kết nối Thông Thiên Cổ!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return false;
    }
  },

  declineTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/decline/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Từ chối giao dịch thất bại");
      get().addLog(data.message || "Đã từ chối giao dịch!");
      set({ activeTrade: null });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return false;
    }
  },

  cancelTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/cancel/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy giao dịch thất bại");
      get().addLog(data.message || "Đã hủy giao dịch!");
      get().addToast("Đã hủy kết nối giao dịch.", "info");
      set({ activeTrade: null });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return false;
    }
  },

  updateTradeOffer: async (tradeId, offer) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/offer/${tradeId}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(offer),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật ưu đãi thất bại");
      await get().fetchActiveTrade();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return false;
    }
  },

  lockTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/lock/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khóa giao dịch thất bại");
      get().addLog("Đã khóa ưu đãi giao dịch!");
      get().addToast("Đã khóa ưu đãi giao dịch!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      await get().fetchActiveTrade();
      get().addToast(err.message, "error");
      return false;
    }
  },

  unlockTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/unlock/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mở khóa giao dịch thất bại");
      get().addLog("Đã mở khóa giao dịch để chỉnh sửa!");
      get().addToast("Đã mở khóa giao dịch!", "info");
      await get().fetchActiveTrade();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      await get().fetchActiveTrade();
      get().addToast(err.message, "error");
      return false;
    }
  },

  confirmTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/confirm/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xác nhận giao dịch thất bại");

      const bothConfirmed = data.sender_confirmed && data.receiver_confirmed;
      if (bothConfirmed) {
        get().addLog("Giao dịch thành công!");
        get().addToast("Giao dịch thành công!", "success");
        set({ activeTrade: null });
        await Promise.all([
          get().refreshPlayerSummary(),
          get().fetchGuInventory(),
          get().refreshRecipeProgress(),
        ]);
      } else {
        get().addLog("Đã xác nhận giao dịch. Chờ đối phương xác nhận...");
        get().addToast("Đã xác nhận giao dịch. Chờ đối phương xác nhận...", "info");
        await get().fetchActiveTrade();
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(err.message, "error");
      return false;
    }
  },
});
