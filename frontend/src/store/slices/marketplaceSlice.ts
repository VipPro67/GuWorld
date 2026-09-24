import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface BuybackOrder {
  id: string;
  buyer_id: string;
  buyer_username: string;
  order_type?: "BUY" | "SELL";
  item_type: "Gu" | "Material" | "Recipe";
  item_target: string | Record<string, unknown>;
  item_data?: string | Record<string, unknown>;
  unit_price: number;
  quantity_requested: number;
  quantity_filled: number;
  currency: "primeval" | "immortal";
  upfront_fee: number;
  total_locked: number;
  status: string;
  duration_hours: number;
  end_time: string;
  created_at: string;
}

export interface OfferedGuInfo {
  instance_id: string;
  gu_id: string;
  gu_name: string;
  rank: number;
}

export interface TradeOfferData {
  primeval_stones: number;
  immortal_stones: number;
  gus: string[];
  materials: Record<string, number>;
  recipes: Record<string, number>;
}

export interface AsyncTradeOffer {
  id: string;
  creator_id: string;
  creator_username: string;
  target_id?: string;
  target_username?: string;
  offered: TradeOfferData;
  offered_gu_details?: OfferedGuInfo[];
  requested: TradeOfferData;
  status: string;
  duration_hours: number;
  end_time: string;
  created_at: string;
}

export interface MarketplaceSlice {
  marketplaceListings: ApiResponseData[];
  buybackOrders: BuybackOrder[];
  tradeOffers: AsyncTradeOffer[];
  fetchMarketplaceListings: () => Promise<void>;
  createMarketplaceListing: (payload: { itemType: "Gu" | "Material" | "Recipe"; guInstanceId?: string; guInstanceIds?: string[]; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal"; listingType: "MARKET" | "AUCTION" | "TRADE"; targetBuyerUsername?: string }) => Promise<boolean>;
  placeMarketplaceBid: (listingId: string, amount: number) => Promise<boolean>;
  cancelMarketplaceListing: (listingId: string) => Promise<boolean>;
  buyMarketplaceListing: (listingId: string, quantity?: number) => Promise<boolean>;

  // Buyback Order Book
  fetchBuybackOrders: () => Promise<void>;
  createBuybackOrder: (payload: { orderType?: "BUY" | "SELL"; itemType: "Gu" | "Material" | "Recipe"; itemTarget: unknown; guInstanceId?: string; guInstanceIds?: string[]; unitPrice: number; quantityRequested: number; currency: "primeval" | "immortal"; durationHours: number }) => Promise<boolean>;
  sellToBuybackOrder: (payload: { orderId: string; quantity: number; guInstanceId?: string; guInstanceIds?: string[] }) => Promise<boolean>;
  cancelBuybackOrder: (orderId: string) => Promise<boolean>;

  // Async Trade Offers
  fetchTradeOffers: () => Promise<void>;
  createTradeOffer: (payload: { offered: TradeOfferData; requested: TradeOfferData; durationHours: number; targetUsername?: string }) => Promise<boolean>;
  acceptTradeOffer: (offerId: string) => Promise<boolean>;
  cancelTradeOffer: (offerId: string) => Promise<boolean>;
}

export const createMarketplaceSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, MarketplaceSlice> = (set, get) => ({
  marketplaceListings: [],
  buybackOrders: [],
  tradeOffers: [],

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
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
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
    } finally { get().endAction("player:inventory"); }
  },

  placeMarketplaceBid: async (listingId, amount) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
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
      const err = caught instanceof Error ? caught : new Error(String(caught)); set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); get().addToast(err.message || "Đặt giá thất bại.", "error"); return false; } finally { get().endAction("player:inventory"); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
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
      const err = caught instanceof Error ? caught : new Error(String(caught)); set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); get().addToast(err.message || "Hủy listing thất bại.", "error"); return false; } finally { get().endAction("player:inventory"); }
  },

  buyMarketplaceListing: async (listingId, quantity) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
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
    } finally { get().endAction("player:inventory"); }
  },

  // Buyback Orders
  fetchBuybackOrders: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buyback`, {});
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ buybackOrders: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("fetch buyback orders failed", err);
    }
  },

  createBuybackOrder: async (payload) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buyback`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          order_type: payload.orderType || "BUY",
          item_type: payload.itemType,
          item_target: payload.itemTarget,
          gu_instance_id: payload.guInstanceId || "",
          gu_instance_ids: payload.guInstanceIds || [],
          unit_price: payload.unitPrice,
          quantity_requested: payload.quantityRequested,
          currency: payload.currency,
          duration_hours: payload.durationHours,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo đơn thu mua thất bại");
      get().addLog(data.message || "Tạo đơn thu mua thành công");
      get().addToast(data.message || "Tạo đơn thu mua thành công", "success");
      get().applyInventoryFromResponse(data);
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchBuybackOrders();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  sellToBuybackOrder: async (payload) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buyback/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          order_id: payload.orderId,
          quantity: payload.quantity,
          gu_instance_id: payload.guInstanceId || "",
          gu_instance_ids: payload.guInstanceIds || [],
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán vào đơn thu mua thất bại");
      get().addLog(data.message || "Bán vật phẩm thành công");
      get().addToast(data.message || "Bán vật phẩm thành công", "success");
      get().applyInventoryFromResponse(data);
      await get().fetchGuInventory();
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchBuybackOrders();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  cancelBuybackOrder: async (orderId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buyback/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ order_id: orderId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đơn thu mua thất bại");
      get().addLog(data.message || "Hủy đơn thu mua thành công");
      get().addToast(data.message || "Hủy đơn thu mua thành công", "success");
      get().applyInventoryFromResponse(data);
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchBuybackOrders();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  // Async Trade Offers
  fetchTradeOffers: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/trade-offers`, {});
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ tradeOffers: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("fetch trade offers failed", err);
    }
  },

  createTradeOffer: async (payload) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/trade-offers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          offered: payload.offered,
          requested: payload.requested,
          duration_hours: payload.durationHours,
          target_username: payload.targetUsername || "",
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng đơn trao đổi thất bại");
      get().addLog(data.message || "Đăng đơn trao đổi thành công");
      get().addToast(data.message || "Đăng đơn trao đổi thành công", "success");
      get().applyInventoryFromResponse(data);
      await get().fetchGuInventory();
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchTradeOffers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  acceptTradeOffer: async (offerId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/trade-offers/accept`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ offer_id: offerId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chấp nhận trao đổi thất bại");
      get().addLog(data.message || "Trao đổi tự động thành công");
      get().addToast(data.message || "Trao đổi tự động thành công", "success");
      get().applyInventoryFromResponse(data);
      await get().fetchGuInventory();
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchTradeOffers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  cancelTradeOffer: async (offerId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/trade-offers/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ offer_id: offerId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đơn trao đổi thất bại");
      get().addLog(data.message || "Hủy đơn trao đổi thành công");
      get().addToast(data.message || "Hủy đơn trao đổi thành công", "success");
      get().applyInventoryFromResponse(data);
      await get().fetchGuInventory();
      const pid = get().player?.id;
      if (pid) await get().fetchPlayerStatus(pid, true);
      await get().fetchTradeOffers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("player:inventory"); }
  },
});
