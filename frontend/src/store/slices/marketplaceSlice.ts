import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface MarketplaceSlice {
  marketplaceListings: ApiResponseData[];
  fetchMarketplaceListings: () => Promise<void>;
  createMarketplaceListing: (payload: { itemType: "Gu" | "Material" | "Recipe"; guInstanceId?: string; guInstanceIds?: string[]; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal"; listingType: "MARKET" | "AUCTION" | "TRADE"; targetBuyerUsername?: string }) => Promise<boolean>;
  placeMarketplaceBid: (listingId: string, amount: number) => Promise<boolean>;
  cancelMarketplaceListing: (listingId: string) => Promise<boolean>;
  buyMarketplaceListing: (listingId: string, quantity?: number) => Promise<boolean>;
}

export const createMarketplaceSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, MarketplaceSlice> = (set, get) => ({
  marketplaceListings: [],

  fetchMarketplaceListings: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, {});
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ marketplaceListings: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("market listings failed", err);
      get().addToast("Lỗi tải thị trường.", "error");
    }
  },

  createMarketplaceListing: async (payload) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          item_type: payload.itemType,
          gu_instance_id: payload.guInstanceId || "",
          gu_instance_ids: payload.guInstanceIds || [],
          item_id: payload.itemId || "",
          quantity: payload.quantity,
          price: payload.price,
          duration_hours: payload.durationHours,
          currency: payload.currency,
          listing_type: payload.listingType,
          target_buyer_username: payload.targetBuyerUsername || "",
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng bán thất bại");
      get().addLog(data.message || "Đăng bán thành công");
      get().applyInventoryFromResponse(data);
      if (payload.itemType === "Gu") {
        await get().fetchGuInventory();
      }
      const pid = get().player?.id;
      if (pid) {
        await get().fetchPlayerStatus(pid, true);
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đăng bán thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  placeMarketplaceBid: async (listingId, amount) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/bid`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId, amount }) });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đặt giá thất bại");
      get().addLog(data.message || "Đặt giá thành công");
      get().applyInventoryFromResponse(data);
      const pid = get().player?.id;
      if (pid) {
        await get().fetchPlayerStatus(pid, true);
      }
      await get().fetchMarketplaceListings();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught)); set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); get().addToast(err.message || "Đặt giá thất bại.", "error"); return false; } finally { set({ actionLoading: false }); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/cancel`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId }) });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy listing thất bại");
      get().addLog(data.message || "Đã hủy");
      get().applyInventoryFromResponse(data);
      await get().fetchGuInventory();
      const pid = get().player?.id;
      if (pid) {
        await get().fetchPlayerStatus(pid, true);
      }
      await get().fetchMarketplaceListings();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught)); set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); get().addToast(err.message || "Hủy listing thất bại.", "error"); return false; } finally { set({ actionLoading: false }); }
  },

  buyMarketplaceListing: async (listingId, quantity) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ listing_id: listingId, quantity }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mua hàng thất bại");
      get().addLog(data.message || "Mua hàng thành công");
      get().applyInventoryFromResponse(data);
      await get().fetchMarketplaceListings();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },
});
