import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { uiRealtimeManager } from "@/lib/realtime/UIRealtimeManager";
import type { GuInstance, GroundDrop, Player, FeedResponse } from "@/store/useGameStore";
import { LatestRequestGate } from "@/lib/player-reconciliation";
import { executeGameAction } from "@/store/actionGateway";

let fetchGuInventoryPromise: Promise<void> | null = null;
const guInventoryRequests = new LatestRequestGate();
let groundDropRequestRevision = 0;
let groundDropRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let groundDropExpiryTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Resets module-level state that must not survive a logout/login cycle.
 * Called from the logout action in authSlice.
 */
export function resetInventorySliceModuleState() {
  fetchGuInventoryPromise = null;
  groundDropRequestRevision = 0;
  if (groundDropRefreshTimer !== null) {
    clearTimeout(groundDropRefreshTimer);
    groundDropRefreshTimer = null;
  }
  if (groundDropExpiryTimer !== null) {
    clearTimeout(groundDropExpiryTimer);
    groundDropExpiryTimer = null;
  }
}

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
  subjugateGu: (instanceId: string) => Promise<boolean>;
  nourishGu: (instanceId: string) => Promise<boolean>;
  repairGu: (instanceId: string) => Promise<boolean>;
  unequipAllGu: () => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useSupportGu: (instanceId: string) => Promise<void>;
  useGus: (instanceIds: string[]) => Promise<boolean>;
  useItem: (itemId: string) => Promise<void>;
  useItems: (itemIds: string[]) => Promise<boolean>;
  groundDrops: GroundDrop[];
  fetchGroundDrops: (viewport?: GroundDropViewport) => Promise<void>;
  /** Last viewport fetched; world:drops events refetch this projection. */
  lastDropViewport: GroundDropViewport | null;
  applyWorldDropsEvent: (event: { region_id?: string }) => void;
  lootGroundDrop: (dropId: string) => Promise<{ success: boolean; message: string; looted?: string[] } | null>;
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
  groundDrops: [],
  lastDropViewport: null,
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
      method: "POST",
      body: { gu_instance_id: guInstanceId },
      actionLock: "player:inventory",
      errorMessage: "Không thể cho ăn.",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message);
        get().addToast(data.message, "success");
      },
    });
  },

  feedGuGroup: async (guId, rank) => {
    await executeGameAction<FeedResponse>(get, set, {
      endpoint: "/api/inventory/feed-group",
      method: "POST",
      body: { gu_id: guId, ...(rank ? { rank } : {}) },
      actionLock: "player:inventory",
      errorMessage: "Không thể cho ăn hàng loạt.",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message);
        get().addToast(data.message, "success");
      },
    });
  },

  feedAllGu: async (scope?: "all" | "equipped" | "rest" | "unequipped") => {
    await executeGameAction<FeedResponse>(get, set, {
      endpoint: "/api/inventory/feed-all",
      method: "POST",
      body: scope ? { scope } : {},
      actionLock: "player:inventory",
      errorMessage: "Không thể cho tất cả Cổ trùng ăn.",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message);
        get().addToast(data.message, "success");
      },
    });
  },

  buyShopItem: async (itemId, quantity, rank = 0) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/buy",
      actionLock: "player:inventory",
      body: { player_id: player.id, item_id: itemId, quantity, rank },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Mua hàng thành công!");
      },
      errorMessage: "Giao dịch thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Mua hàng thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },

  sellShopItem: async (itemId, quantity, itemType, rank = 0) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/sell",
      actionLock: "player:inventory",
      body: { player_id: player.id, item_id: itemId, quantity, item_type: itemType, rank },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Bán hàng thành công!");
      },
      errorMessage: "Bán hàng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Bán hàng thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },

  sellShopItemsBulk: async (items) => {
    const { player } = get();
    if (!player) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/shop/sell-bulk",
      actionLock: "player:inventory",
      body: { player_id: player.id, items },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Bán hàng thành công!");
      },
      errorMessage: "Bán hàng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Bán hàng thất bại: ${res.error}`);
      return false;
    }
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
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Bán hàng thành công!");
      },
      successToast: (data) => data.message || "Bán hàng thành công!",
      errorMessage: "Bán hàng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Bán cho thương nhân thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },

  buyFromMerchant: async (merchantId, items) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/merchant/buy",
      actionLock: "player:inventory",
      body: { merchant_id: merchantId, items },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Mua hàng thành công!");
      },
      successToast: (data) => data.message || "Mua hàng thành công!",
      errorMessage: "Mua hàng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Mua từ thương nhân thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },

  dropGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/drop",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Đã vứt Cổ trùng.");
      },
      errorMessage: "Vứt Cổ thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Vứt Cổ thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },
  dropGus: async (instanceIds) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/drop-batch",
      actionLock: "player:inventory",
      body: { gu_instance_ids: instanceIds },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Đã vứt Cổ trùng hàng loạt.");
      },
      errorMessage: "Vứt Cổ thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Vứt Cổ hàng loạt thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
  },
  equipGu: async (instanceId, equip) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/equip",
      actionLock: "player:inventory",
      body: { instance_id: instanceId, equip },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || (equip ? "Trang bị Cổ trùng thành công!" : "Tháo Cổ trùng thành công!");
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || (equip ? "Trang bị Cổ trùng thành công!" : "Tháo Cổ trùng thành công!"),
      errorMessage: "Thao tác trang bị thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Thao tác trang bị thất bại: ${res.error}`);
    }
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
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/unequip-all",
      actionLock: "player:inventory",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        if (data.message) {
          get().addLog(data.message);
        }
      },
      successToast: (data) => data.message || "Tháo toàn bộ Cổ trùng thành công!",
      errorMessage: "Tháo cổ trùng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Tháo cổ trùng thất bại: ${res.error}`);
    }
  },

  updateGuStatus: async (instanceId, status) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/status",
      actionLock: "player:inventory",
      body: { instance_id: instanceId, status },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        get().addLog(data.message || "Cập nhật trạng thái Cổ trùng thành công!");
      },
      errorMessage: "Cập nhật trạng thái thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Lỗi cập nhật trạng thái: ${res.error}`);
    }
  },

  setVitalGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/set-vital",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || "Thiết lập Bản Mệnh Cổ thành công!";
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || "Thiết lập Bản Mệnh Cổ thành công!",
      errorMessage: "Thiết lập Bản Mệnh Cổ thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Lỗi thiết lập Bản Mệnh Cổ: ${res.error}`);
    }
  },

  useGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/use-gu",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || "Sử dụng Cổ trùng thành công!";
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || "Sử dụng Cổ trùng thành công!",
      errorMessage: "Sử dụng Cổ trùng thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Lỗi sử dụng Cổ trùng: ${res.error}`);
    }
  },

  useSupportGu: async (instanceId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/inventory/use-support",
      actionLock: "player:inventory",
      body: { gu_instance_id: instanceId },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || "Sử dụng Cổ Hỗ Trợ thành công!";
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || "Sử dụng Cổ Hỗ Trợ thành công!",
      errorMessage: "Sử dụng Cổ Hỗ Trợ thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Sử dụng Cổ Hỗ Trợ thất bại: ${res.error}`);
    }
  },

  useItem: async (itemId) => {
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/player/use-item",
      actionLock: "player:inventory",
      body: { item_id: itemId },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || "Sử dụng vật phẩm thành công!";
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || "Sử dụng vật phẩm thành công!",
      errorMessage: "Sử dụng vật phẩm thất bại.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Sử dụng vật phẩm thất bại: ${res.error}`);
    }
  },

  useItems: async (itemIds) => {
    if (!itemIds?.length) return false;
    const res = await executeGameAction<{ message?: string }>(get, set, {
      endpoint: "/api/player/use-items",
      actionLock: "player:inventory",
      body: { item_ids: itemIds },
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        const successMsg = data.message || "Đã sử dụng vật phẩm.";
        get().addLog(successMsg);
      },
      successToast: (data) => data.message || "Đã sử dụng vật phẩm.",
      errorMessage: "Không thể sử dụng vật phẩm.",
    });
    if (!res.ok && res.error) {
      get().addLog(`Sử dụng vật phẩm thất bại: ${res.error}`);
      return false;
    }
    return res.ok;
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

  fetchGroundDrops: async (viewport) => {
    if (viewport) set({ lastDropViewport: viewport });
    const requestRevision = ++groundDropRequestRevision;
    try {
      const data = await uiRealtimeManager.request<{ drops?: GroundDrop[] }>("world.drops", viewport ? {
        x: Math.floor(viewport.x),
        y: Math.floor(viewport.y),
        radius: Math.max(0, Math.floor(viewport.radius)),
      } : {});
      if (requestRevision === groundDropRequestRevision) {
        const drops = data.drops || [];
        set({ groundDrops: drops });
        // Drops expire and lose protection on their own. The old 8s poll used
        // to clear them; reschedule a one-shot refresh at the next transition.
        if (groundDropExpiryTimer) {
          clearTimeout(groundDropExpiryTimer);
          groundDropExpiryTimer = null;
        }
        const center = viewport ?? get().lastDropViewport;
        let nextChange = Number.POSITIVE_INFINITY;
        for (const drop of drops) {
          for (const iso of [drop.expires_at, drop.protected_until]) {
            if (!iso) continue;
            const at = new Date(iso).getTime();
            if (Number.isFinite(at) && at > Date.now() && at < nextChange) nextChange = at;
          }
        }
        if (center && Number.isFinite(nextChange)) {
          groundDropExpiryTimer = setTimeout(() => {
            groundDropExpiryTimer = null;
            void get().fetchGroundDrops(center);
          }, Math.max(1000, nextChange - Date.now() + 1000));
        }
      }
    } catch (e) {
      console.error("fetchGroundDrops error:", e);
    }
  },

  applyWorldDropsEvent: (event) => {
    const currentRegion = get().player?.current_region;
    if (!event?.region_id || (currentRegion && event.region_id !== currentRegion)) return;
    const viewport = get().lastDropViewport;
    if (!viewport) return;
    if (groundDropRefreshTimer) clearTimeout(groundDropRefreshTimer);
    groundDropRefreshTimer = setTimeout(() => {
      groundDropRefreshTimer = null;
      void get().fetchGroundDrops(viewport);
    }, 250);
  },

  lootGroundDrop: async (dropId) => {
    const res = await executeGameAction<{ message?: string; looted?: string[] }>(get, set, {
      endpoint: `/api/world/drops/${dropId}/loot`,
      method: "POST",
      onSuccess: (data) => {
        get().applyInventoryFromResponse(data);
        // Immediately remove the looted drop from state so map UI updates instantly
        set({ groundDrops: get().groundDrops.filter((d: ApiResponseData) => d.id !== dropId) });
      },
      successToast: (data) => data.message || "Thu thập chiến lợi phẩm thành công!",
      errorMessage: "Nhặt chiến lợi phẩm thất bại",
    });
    if (!res.ok || !res.data) {
      return res.error ? { success: false, message: res.error } : null;
    }
    return { success: true, message: res.data.message || "", looted: res.data.looted };
  },
});
