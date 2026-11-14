import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { GuInstance, FeedResponse } from "@/store/useGameStore";
import { LatestRequestGate } from "@/lib/player-reconciliation";
import { executeGameAction } from "@/store/actionGateway";

let fetchGuInventoryPromise: Promise<void> | null = null;
const guInventoryRequests = new LatestRequestGate();

/**
 * Resets module-level state that must not survive a logout/login cycle.
 * Called from the logout action in authSlice.
 */
export function resetInventorySliceModuleState() {
  fetchGuInventoryPromise = null;
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
  subjugateGu: (instanceId: string) => Promise<boolean>;
  nourishGu: (instanceId: string) => Promise<boolean>;
  repairGu: (instanceId: string) => Promise<boolean>;
  unequipAllGu: () => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string, teleport?: { mode: "random" | "coordinates"; x?: number; y?: number }) => Promise<void>;
  useSupportGu: (instanceId: string) => Promise<void>;
  useGus: (instanceIds: string[]) => Promise<boolean>;
  useItem: (itemId: string) => Promise<void>;
  useItems: (itemIds: string[]) => Promise<boolean>;
  merchants: MerchantView[];
  merchantAtVillage: boolean;
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
  merchants: [],
  merchantAtVillage: false,

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
    await executeGameAction<FeedResponse>(get, set, {
      endpoint: "/api/inventory/feed",
      body: { gu_instance_id: guInstanceId },
      actionLock: "player:inventory",
      errorMessage: "Không thể cho ăn.",
    });
  },

  feedGuGroup: async (guId, rank) => {
    await executeGameAction<FeedResponse>(get, set, {
      endpoint: "/api/inventory/feed-group",
      body: { gu_id: guId, ...(rank ? { rank } : {}) },
      actionLock: "player:inventory",
      errorMessage: "Không thể cho ăn hàng loạt.",
    });
  },

  feedAllGu: async (scope?: "all" | "equipped" | "rest" | "unequipped") => {
    await executeGameAction<FeedResponse>(get, set, {
      endpoint: "/api/inventory/feed-all",
      body: scope ? { scope } : {},
      actionLock: "player:inventory",
      errorMessage: "Không thể cho tất cả Cổ trùng ăn.",
    });
  },

  buyShopItem: async (itemId, quantity, rank = 0) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/buy",
      actionLock: "player:inventory",
      body: { player_id: player.id, item_id: itemId, quantity, rank },
      errorMessage: "Giao dịch thất bại.",
    });
    return res.ok;
  },

  sellShopItem: async (itemId, quantity, itemType, rank = 0) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/sell",
      actionLock: "player:inventory",
      body: { player_id: player.id, item_id: itemId, quantity, item_type: itemType, rank },
      errorMessage: "Bán hàng thất bại.",
    });
    return res.ok;
  },

  sellShopItemsBulk: async (items) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/sell-bulk",
      actionLock: "player:inventory",
      body: { player_id: player.id, items },
      errorMessage: "Bán hàng thất bại.",
    });
    return res.ok;
  },

  fetchMerchants: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/merchant`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data = await response.json();
      set({
        merchants: (data.merchants || []) as MerchantView[],
        merchantAtVillage: !!data.at_village,
      });
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
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/merchant/sell",
      actionLock: "player:inventory",
      body: { merchant_id: merchantId, items },
      errorMessage: "Bán hàng thất bại.",
    });
    return res.ok;
  },

  buyFromMerchant: async (merchantId, items) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/merchant/buy",
      actionLock: "player:inventory",
      body: { merchant_id: merchantId, items },
      errorMessage: "Mua hàng thất bại.",
    });
    return res.ok;
  },

  dropGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/drop",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      errorMessage: "Vứt Cổ thất bại.",
    });
    return res.ok;
  },
  dropGus: async (instanceIds) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/drop-batch",
      actionLock: "player:inventory",
      body: { gu_instance_ids: instanceIds },
      errorMessage: "Vứt Cổ thất bại.",
    });
    return res.ok;
  },
  equipGu: async (instanceId, equip) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/equip",
      actionLock: "player:inventory",
      body: { instance_id: instanceId, equip },
      errorMessage: "Thao tác trang bị thất bại.",
    });
  },

  subjugateGu: async (instanceId) => {
    const res = await executeGameAction<{ remaining_mana?: number; message?: string }>(get, set, {
      endpoint: "/api/inventory/subjugate-gu",
      actionLock: "player:inventory",
      body: { instance_id: instanceId },
      onSuccess: async (data) => {
        if (typeof data.remaining_mana === "number") {
          const p = get().player;
          if (p) {
            set({ player: { ...p, current_mana: data.remaining_mana } });
          }
        }
        await get().fetchGuInventory(true);
        const msg = data.message || "Luyện hóa Cổ trùng thành công!";
        get().addLog(msg);
      },
      successToast: (data) => data.message || "Luyện hóa Cổ trùng thành công!",
      errorMessage: "Luyện hóa Cổ trùng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Luyện hóa thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },

  nourishGu: async (instanceId) => {
    const res = await executeGameAction<{ remaining_mana?: number; message?: string }>(get, set, {
      endpoint: "/api/inventory/nourish-gu",
      method: "POST",
      body: { instance_id: instanceId },
      actionLock: "player:inventory",
      errorMessage: "Ôn dưỡng Cổ trùng thất bại.",
      onSuccess: async (data) => {
        if (typeof data.remaining_mana === "number") {
          const p = get().player;
          if (p) {
            set({ player: { ...p, current_mana: data.remaining_mana } });
          }
        }
        await get().fetchGuInventory(true);
        const msg = data.message || "Ôn dưỡng Cổ trùng thành công!";
        get().addLog(msg);
        get().addToast(msg, "success");
      },
    });
    return res.ok;
  },

  repairGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/repair-gu",
      method: "POST",
      body: { instance_id: instanceId },
      actionLock: "player:inventory",
      errorMessage: "Tu bổ Cổ trùng thất bại.",
      onSuccess: async (data) => {
        await get().fetchGuInventory(true);
        await get().refreshPlayerSummary();
        const msg = data.message || "Tu bổ Cổ trùng thành công!";
        get().addLog(msg);
        get().addToast(msg, "success");
      },
    });
    return res.ok;
  },

  unequipAllGu: async () => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/unequip-all",
      actionLock: "player:inventory",
      errorMessage: "Tháo cổ trùng thất bại.",
    });
  },

  updateGuStatus: async (instanceId, status) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/status",
      actionLock: "player:inventory",
      body: { instance_id: instanceId, status },
      errorMessage: "Cập nhật trạng thái thất bại.",
    });
  },

  setVitalGu: async (instanceId) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/set-vital",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      errorMessage: "Thiết lập Bản Mệnh Cổ thất bại.",
    });
  },

  useGu: async (instanceId, teleport) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/use-gu",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId, ...(teleport ? { teleport_mode: teleport.mode, ...(teleport.mode === "coordinates" ? { x: teleport.x, y: teleport.y } : {}) } : {}) },
      errorMessage: "Sử dụng Cổ trùng thất bại.",
    });
  },

  useSupportGu: async (instanceId) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/use-support",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      errorMessage: "Sử dụng Cổ Hỗ Trợ thất bại.",
    });
  },

  useItem: async (itemId) => {
    await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/player/use-item",
      actionLock: "player:inventory",
      body: { item_id: itemId },
      errorMessage: "Sử dụng vật phẩm thất bại.",
    });
  },

  useItems: async (itemIds) => {
    if (!itemIds?.length) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/player/use-items",
      actionLock: "player:inventory",
      body: { item_ids: itemIds },
      errorMessage: "Không thể sử dụng vật phẩm.",
    });
    return res.ok;
  },

  useGus: async (instanceIds) => {
    if (!instanceIds?.length) return false;
    let allOk = true;
    for (const guInstanceId of instanceIds) {
      const res = await executeGameAction<{ message?: string }>(get, set, {
        endpoint: "/api/inventory/use-gu",
        actionLock: "player:inventory",
        body: { gu_instance_id: guInstanceId },
        errorMessage: `Không thể sử dụng Cổ trùng ${guInstanceId}.`,
      });
      if (!res.ok) allOk = false;
    }
    return allOk;
  },
});
