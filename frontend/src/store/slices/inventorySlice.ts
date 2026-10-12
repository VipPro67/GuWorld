import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { GuInstance, GroundDrop, Player, FeedResponse } from "@/store/useGameStore";
import { LatestRequestGate } from "@/lib/player-reconciliation";

let fetchGuInventoryPromise: Promise<void> | null = null;
const guInventoryRequests = new LatestRequestGate();
let groundDropRequestRevision = 0;

export interface GroundDropViewport {
  x: number;
  y: number;
  radius: number;
}

export interface InventorySlice {
  guInventoryLoaded: boolean;
  fetchGuInventory: (forceRefresh?: boolean, reconciliationRevision?: number) => Promise<void>;
  feedGu: (guInstanceId: string) => Promise<void>;
  feedGuGroup: (guId: string, rank?: number) => Promise<void>;
  feedAllGu: (scope?: "all" | "equipped" | "rest" | "unequipped") => Promise<void>;
  buyShopItem: (itemId: string, quantity: number, rank?: number) => Promise<boolean>;
  sellShopItem: (itemId: string, quantity: number, itemType: "Gu" | "Material" | "Recipe", rank?: number) => Promise<boolean>;
  sellShopItemsBulk: (items: { item_id: string; quantity: number; item_type: "Gu" | "Material" | "Recipe"; rank?: number }[]) => Promise<boolean>;
  dropGu: (instanceId: string) => Promise<boolean>;
  dropGus: (instanceIds: string[]) => Promise<boolean>;
  equipGu: (instanceId: string, equip: boolean) => Promise<void>;
  unequipAllGu: () => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useSupportGu: (instanceId: string) => Promise<void>;
  useGus: (instanceIds: string[]) => Promise<boolean>;
  useItem: (itemId: string) => Promise<void>;
  useItems: (itemIds: string[]) => Promise<boolean>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: ApiResponseData } | null>;
  fetchGrid: (statsOnly?: boolean) => Promise<{ x: number; y: number; vision_radius: number; cells?: ApiResponseData[]; stats: ApiResponseData; players_on_map?: ApiResponseData[]; ground_drops?: ApiResponseData[]; world_width?: number; world_height?: number } | null>;
  groundDrops: GroundDrop[];
  fetchGroundDrops: (viewport?: GroundDropViewport) => Promise<void>;
  lootGroundDrop: (dropId: string) => Promise<{ success: boolean; message: string; looted?: string[] } | null>;
  merchants: MerchantView[];
  fetchMerchants: () => Promise<void>;
  quoteMerchant: (merchantId: string, items: MerchantSellLine[]) => Promise<MerchantQuoteItem[]>;
  sellToMerchant: (merchantId: string, items: MerchantSellLine[]) => Promise<boolean>;
  buyFromMerchant: (merchantId: string, items: { item_id: string; rank?: number; quantity: number }[]) => Promise<boolean>;
}

export interface MerchantBuyTags {
  types?: string[];
  min_rank?: number;
  max_rank?: number;
  item_ids?: string[];
}

export interface MerchantStockItem {
  item_id: string;
  name: string;
  type: string;
  rank: number;
  price: number;
  stock: number;
}

export interface MerchantView {
  id: string;
  region_id: string;
  x: number;
  y: number;
  distance: number;
  profile_id: string;
  name: string;
  description: string;
  kind: string;
  buy_ratio: number;
  buy_floor_ratio: number;
  buy_tags: MerchantBuyTags;
  quota_per_item: number;
  quota_period: string;
  stock: MerchantStockItem[];
}

export interface MerchantSellLine {
  item_id: string;
  item_type: string;
  quantity: number;
  rank?: number;
}

export interface MerchantQuoteItem {
  item_id: string;
  rank: number;
  accepted: boolean;
  unit_price: number;
  quota_left: number;
}

export const createInventorySlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, InventorySlice> = (set, get) => ({
  guInventoryLoaded: false,
  groundDrops: [],
  merchants: [],

  fetchGuInventory: async (forceRefresh = false, requestRevision) => {
    if (fetchGuInventoryPromise && !forceRefresh) return fetchGuInventoryPromise;
    const inventoryRequestRevision = guInventoryRequests.next();
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;

      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/inventory/gus`);

        if (response.status === 401) {
          logout();
          return;
        }

        if (!response.ok) {
          throw new Error("Failed to fetch inventory");
        }

        const data: { gu_inventory: GuInstance[]; equipped_gus: string[] } = await response.json();
        const currentPlayer = get().player;
        if (!currentPlayer) return;
        if (!guInventoryRequests.isCurrent(inventoryRequestRevision)) return;
        if (requestRevision !== undefined && requestRevision !== get().reconciliationRevision) return;

        get().applyInventoryFromResponse(data);
      } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
        console.error("Gu inventory fetch failed:", err.message);
      } finally {
        if (fetchGuInventoryPromise === promise) fetchGuInventoryPromise = null;
      }
    };
    const promise = executeFetch();
    fetchGuInventoryPromise = promise;
    return promise;
  },

  feedGu: async (guInstanceId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ gu_instance_id: guInstanceId }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho ăn.");
      }

      const data: FeedResponse = await response.json();
      get().applyInventoryFromResponse(data);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Cho Cổ ăn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  feedGuGroup: async (guId, rank) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed-group`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ gu_id: guId, ...(rank ? { rank } : {}) }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho ăn hàng loạt.");
      }

      const data: FeedResponse = await response.json();
      get().applyInventoryFromResponse(data);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Cho ăn hàng loạt thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  feedAllGu: async (scope?: "all" | "equipped" | "rest" | "unequipped") => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed-all`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(scope ? { scope } : {}),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho tất cả Cổ trùng ăn.");
      }

      const data: FeedResponse = await response.json();
      get().applyInventoryFromResponse(data);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Cho tất cả Cổ trùng ăn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  buyShopItem: async (itemId, quantity, rank = 0) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return false;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/buy`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          item_id: itemId,
          quantity: quantity,
          rank: rank,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Giao dịch thất bại.");
      }

      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Mua hàng thành công!");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally {
      get().endAction("player:inventory");
    }
  },


  sellShopItem: async (itemId, quantity, itemType, rank = 0) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, item_id: itemId, quantity, item_type: itemType, rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bán hàng thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  sellShopItemsBulk: async (items) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell-bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bán hàng thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  fetchMerchants: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/merchant`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data = await response.json();
      set({ merchants: (data.merchants || []) as MerchantView[] });
    } catch (caught: unknown) {
      console.error("fetchMerchants failed", caught);
    }
  },

  quoteMerchant: async (merchantId, items) => {
    const { token, logout } = get();
    if (!token) return [];
    try {
      const response = await authFetch(`${BACKEND_URL}/api/merchant/quote`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ merchant_id: merchantId, items }),
      });
      if (response.status === 401) { logout(); return []; }
      if (!response.ok) return [];
      const data = await response.json();
      return (data.items || []) as MerchantQuoteItem[];
    } catch {
      return [];
    }
  },

  sellToMerchant: async (merchantId, items) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/merchant/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ merchant_id: merchantId, items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Bán hàng thành công!");
      get().addToast(data.message || "Bán hàng thành công!", "success");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bán cho thương nhân thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  buyFromMerchant: async (merchantId, items) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/merchant/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ merchant_id: merchantId, items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mua hàng thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Mua hàng thành công!");
      get().addToast(data.message || "Mua hàng thành công!", "success");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mua từ thương nhân thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },

  dropGu: async (instanceId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/drop`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: instanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Đã vứt Cổ trùng.");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Vứt Cổ thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },
  dropGus: async (instanceIds) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/drop-batch`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_ids: instanceIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Đã vứt Cổ trùng hàng loạt.");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Vứt Cổ hàng loạt thất bại: ${err.message}`);
      return false;
    } finally { get().endAction("player:inventory"); }
  },
  equipGu: async (instanceId, equip) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/equip`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          instance_id: instanceId,
          equip: equip,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thao tác trang bị thất bại.");
      }

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || (equip ? "Trang bị Cổ trùng thành công!" : "Tháo Cổ trùng thành công!");
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Thao tác trang bị thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  unequipAllGu: async () => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/unequip-all`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tháo cổ trùng thất bại.");
      }

      get().applyInventoryFromResponse(data);
      if (data.message) {
        get().addLog(data.message);
        get().addToast(data.message, "success");
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Tháo cổ trùng thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  updateGuStatus: async (instanceId, status) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          instance_id: instanceId,
          status: status,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Cập nhật trạng thái thất bại.");
      }

      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Cập nhật trạng thái Cổ trùng thành công!");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi cập nhật trạng thái: ${err.message}`);
    } finally {
      get().endAction("player:inventory");
    }
  },

  setVitalGu: async (instanceId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/set-vital`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thiết lập Bản Mệnh Cổ thất bại.");
      }

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || "Thiết lập Bản Mệnh Cổ thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Lỗi thiết lập Bản Mệnh Cổ: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  useGu: async (instanceId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/use-gu`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng Cổ trùng thất bại.");
      }

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || "Sử dụng Cổ trùng thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Lỗi sử dụng Cổ trùng: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  useSupportGu: async (instanceId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/use-support`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: instanceId }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sử dụng Cổ Hỗ Trợ thất bại.");

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || "Sử dụng Cổ Hỗ Trợ thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Sử dụng Cổ Hỗ Trợ thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  useItem: async (itemId) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/use-item`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          item_id: itemId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng vật phẩm thất bại.");
      }

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || "Sử dụng vật phẩm thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      get().endAction("player:inventory");
    }
  },

  useItems: async (itemIds) => {
    const { player, token, logout } = get();
    if (!player || !token || !itemIds?.length || !get().beginAction("player:inventory")) return false;

    set({ error: null });

    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/use-items`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ item_ids: itemIds }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        set({ error: data.error });
        get().addLog(`Sử dụng vật phẩm thất bại: ${data.error}`);
        get().addToast(data.error || "Không thể sử dụng vật phẩm.", "error");
        return false;
      }

      get().applyInventoryFromResponse(data);
      const successMsg = data.message || "Đã sử dụng vật phẩm.";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:inventory");
    }
  },

  useGus: async (instanceIds) => {
    const { player, token, logout } = get();
    if (!player || !token || !instanceIds?.length || !get().beginAction("player:inventory")) return false;

    set({ error: null });
    let successCount = 0;
    let failedCount = 0;

    for (const guInstanceId of instanceIds) {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/inventory/use-gu`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ gu_instance_id: guInstanceId }),
        });

        if (response.status === 401) {
          logout();
          throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
        }

        const data = await response.json();
        if (!response.ok) {
          failedCount += 1;
          get().addToast(data.error || `Không thể sử dụng Cổ trùng ${guInstanceId}.`, "error");
          continue;
        }

        get().applyInventoryFromResponse(data);
        const successMsg = data.message || `Đã sử dụng Cổ trùng ${guInstanceId}.`;
        get().addLog(successMsg);
        get().addToast(successMsg, "success");
        successCount += 1;
      } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
        failedCount += 1;
        set({ error: err.message });
        get().addLog(`Sử dụng Cổ trùng thất bại: ${err.message}`);
        get().addToast(err.message, "error");
      }
    }

    get().endAction("player:inventory");
    return successCount > 0 && failedCount === 0;
  },

  movePlayer: async (x, y, attackTargetId?) => {
    const { player, token, logout } = get();
    if (!player || !token || !get().beginAction("player:inventory")) return null;

    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/move`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          x: x,
          y: y,
          attack_target_id: attackTargetId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      get().applyInventoryFromResponse(data.player || data);
      get().addLog(data.log || "Di chuyển thành công.");

      // Handle skip battle result
      if (data.skip_result) {
        const skipResult = data.skip_result;
        const allies = Array.isArray(skipResult.allies) ? skipResult.allies.slice(1) : [];
        const allyText = allies.length > 0 ? ` Cùng ${allies.map((a: { name: string }) => a.name).join(", ")} xuất trận.` : " Một mình xuất trận.";
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!${allyText}`, "success");
          if (skipResult.mine_occupied) {
            get().addToast(`Đã chiếm mỏ ${skipResult.mine_occupied}!`, "info");
          }
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }

      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Di chuyển thất bại: ${err.message}`);
      return null;
    } finally {
      get().endAction("player:inventory");
    }
  },

  fetchGrid: async (statsOnly = false) => {
    const { token, logout } = get();
    if (!token) return null;

    try {
      const query = statsOnly ? "?stats_only=true" : "";
      const response = await authFetch(`${BACKEND_URL}/api/explore/grid${query}`, {
        headers: {
        },
      });

      if (response.status === 401) {
        logout();
        return null;
      }

      if (!response.ok) {
        throw new Error("Failed to load map grid viewport");
      }

      const data = await response.json();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Grid fetch failed:", err.message);
      get().addToast("Lỗi tải bản đồ.", "error");
      return null;
    }
  },

  fetchGroundDrops: async (viewport) => {
    const requestRevision = ++groundDropRequestRevision;
    try {
      const query = viewport
        ? `?x=${Math.floor(viewport.x)}&y=${Math.floor(viewport.y)}&radius=${Math.max(0, Math.floor(viewport.radius))}`
        : "";
      const response = await authFetch(`${BACKEND_URL}/api/world/drops${query}`);
      if (response.ok) {
        const data = await response.json();
        if (requestRevision === groundDropRequestRevision) {
          set({ groundDrops: data.drops || [] });
        }
      }
    } catch (e) {
      console.error("fetchGroundDrops error:", e);
    }
  },

  lootGroundDrop: async (dropId) => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/world/drops/${dropId}/loot`, {
        method: "POST",
      });
      const data = await response.json();
      if (!response.ok) {
        get().addToast(data.error || "Nhặt chiến lợi phẩm thất bại", "error");
        return { success: false, message: data.error };
      }

      const msg = data.message || "Thu thập chiến lợi phẩm thành công!";
      get().applyInventoryFromResponse(data);
      get().addToast(msg, "success");

      // Immediately remove the looted drop from state so map UI updates instantly
      set({ groundDrops: get().groundDrops.filter((d: ApiResponseData) => d.id !== dropId) });

      try {
        const { useWorldStore } = await import("@/store/useWorldStore");
        const ws = useWorldStore.getState();
        if (ws.selectedObject?.id === dropId) {
          ws.selectObject(null);
        }
        if (ws.addWorldLog) {
          ws.addWorldLog(msg);
        }
      } catch (e) {}

      if (get().player?.id) {
        await get().fetchPlayerStatus(get().player!.id);
      }
      return { success: true, message: data.message, looted: data.looted };
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      get().addToast(e.message || "Lỗi kết nối", "error");
      return null;
    }
  },
});
