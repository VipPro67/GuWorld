import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
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
  useGu: (instanceId: string, teleport?: { mode: "random" | "coordinates"; x?: number; y?: number }) => Promise<void>;
  useSupportGu: (instanceId: string) => Promise<void>;
  useGus: (instanceIds: string[]) => Promise<boolean>;
  useItem: (itemId: string) => Promise<void>;
  useItems: (itemIds: string[]) => Promise<boolean>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: ApiResponseData } | null>;
  fetchGrid: (radius?: number) => Promise<{ x: number; y: number; vision_radius: number; max_vision_radius?: number; cells: ApiResponseData[]; stats: ApiResponseData; players_on_map?: ApiResponseData[]; ground_drops?: ApiResponseData[]; world_width?: number; world_height?: number } | null>;
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

  fetchGroundDrops: async (viewport) => {
    if (viewport) set({ lastDropViewport: viewport });
    const requestRevision = ++groundDropRequestRevision;
    try {
      const params = new URLSearchParams();
      if (viewport) {
        params.set("x", String(Math.floor(viewport.x)));
        params.set("y", String(Math.floor(viewport.y)));
        params.set("radius", String(Math.max(0, Math.floor(viewport.radius))));
      }
      const res = await authFetch(`${BACKEND_URL}/api/world/drops?${params.toString()}`);
      if (!res.ok) return;
      const data = await res.json();
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

  movePlayer: async (x, y, attackTargetId) => {
    const { player, token, actionLoading } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          x,
          y,
          attack_target_id: attackTargetId,
        }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      if (data.player) {
        get().setPlayerFromServer(data.player);
      }
      get().applyInventoryFromResponse(data);
      get().addLog(data.log || "Di chuyển thành công.");

      if (data.skip_result) {
        const skipResult = data.skip_result;
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!`, "success");
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }

      return data;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Thao tác thất bại.";
      set({ error: message });
      get().addToast(message, "error");
      get().addLog(`Di chuyển thất bại: ${message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchGrid: async (radius?: number) => {
    const { token } = get();
    if (!token) return null;

    try {
      const url = radius ? `${BACKEND_URL}/api/explore/grid?radius=${radius}` : `${BACKEND_URL}/api/explore/grid`;
      const response = await authFetch(url);
      if (!response.ok) {
        throw new Error("Failed to load map grid viewport");
      }
      return await response.json();
    } catch (e) {
      console.error("fetchGrid error:", e);
      return null;
    }
  },
});
