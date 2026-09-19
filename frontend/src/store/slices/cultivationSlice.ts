import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { getDictionaryCache, setDictionaryCache } from "@/lib/db-cache";
import type {
  GuInstance,
  RefineRateBreakdown,
  RefiningRecipe,
  Region,
  WorldConfig,
} from "@/store/useGameStore";

export interface CultivationSlice {
  recipes: RefiningRecipe[];
  recipesLoaded: boolean;
  regions: Region[];
  regionsLoaded: boolean;
  world: WorldConfig | null;
  fetchRecipes: () => Promise<void>;
  fetchRegions: () => Promise<void>;
  fetchWorld: () => Promise<WorldConfig | null>;
  quests: ApiResponseData[];
  fetchQuests: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance; rate_breakdown?: RefineRateBreakdown } | null>;
  deduceRecipe: (recipeId: string) => Promise<{ success: boolean; message: string; completion_rate: number } | null>;
  breakWall: (manaSpend: number) => Promise<{ success: boolean; message: string } | null>;
  rechargeWithStones: (stones: number) => Promise<{ success: boolean; message: string } | null>;
  useRelicGu: (relicKey: string) => Promise<{ success: boolean; message: string } | null>;
  ascension: () => Promise<{ success: boolean; message: string } | null>;
  triggerTribulation: () => Promise<{ success: boolean; message: string } | null>;
  completeQuest: (questId: string) => Promise<{ success: boolean; message: string } | null>;
  redeemGiftCode: (code: string) => Promise<{ success: boolean; message: string } | null>;
}

export const createCultivationSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, CultivationSlice> = (set, get) => ({
  recipes: [],
  recipesLoaded: false,
  quests: [],
  regions: [],
  regionsLoaded: false,
  world: null,

  fetchRecipes: async () => {
    if (get().recipesLoaded) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/recipes`);
      if (response.status === 401) {
        set({ recipes: [], recipesLoaded: true });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load recipes");
      }
      const data = await response.json();
      set({ recipes: data || [], recipesLoaded: true });
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Recipes sync error:", err.message);
      set({ recipesLoaded: true });
    }
  },

  fetchQuests: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/quest`);
      if (response.status === 401) {
        set({ quests: [] });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load quests");
      }
      const data = await response.json();
      set({ quests: data.quests || [] });
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Quests sync error:", err.message);
    }
  },

  fetchRegions: async () => {
    let cachedEtag: string | undefined;

    // 1. Read from IndexedDB for 0ms instant display
    try {
      const cached = await getDictionaryCache<Region[]>("game_explore_regions");
      if (cached && cached.data && Array.isArray(cached.data) && cached.data.length > 0) {
        cachedEtag = cached.etag;
        set({ regions: cached.data, regionsLoaded: true });
      }
    } catch (err) {
      console.warn("[Regions] Failed to read from IndexedDB:", err);
    }

    // 2. Fetch from backend with ETag revalidation
    try {
      const headers: Record<string, string> = {};
      if (cachedEtag) {
        headers["If-None-Match"] = cachedEtag;
      }

      const response = await authFetch(`${BACKEND_URL}/api/explore/regions`, { headers });
      if (response.status === 304) {
        // Cached data is freshest
        set({ regionsLoaded: true });
        return;
      }
      if (response.status === 401) {
        if (!get().regionsLoaded) {
          set({ regions: [], regionsLoaded: true });
        }
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load regions");
      }
      const data = await response.json();
      const etag = response.headers.get("ETag") || undefined;
      set({ regions: data || [], regionsLoaded: true });
      void setDictionaryCache("game_explore_regions", data || [], etag);
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Regions sync error:", err.message);
      set({ regionsLoaded: true });
    }
  },

  fetchWorld: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/world`);
      if (!response.ok) {
        throw new Error("Failed to load world config");
      }
      const data = await response.json();
      set({ world: data || null });
      return data || null;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("World config sync error:", err.message);
      return null;
    }
  },

  refineGu: async (recipeId, selectedInstanceIds) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          recipe_id: recipeId,
          selected_gu_instance_ids: selectedInstanceIds,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Hợp luyện thất bại.");
      }

      get().applyInventoryFromResponse(data);
      get().addLog(data.message);
      get().addToast(data.message, data.status === "success" ? "success" : "error");
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Hợp luyện thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return { status: "failed", message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  deduceRecipe: async (recipeId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/deduce`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          recipe_id: recipeId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thôi diễn thất bại.");
      }

      get().applyInventoryFromResponse(data);
      // update completion rate in local recipes state
      set((state: ApiResponseData) => ({
        recipes: state.recipes.map((r: ApiResponseData) =>
          r.id === recipeId ? { ...r, completion_rate: data.completion_rate } : r
        ),
      }));

      get().addLog(data.message);
      get().addToast(data.message, "success");
      return { success: true, message: data.message, completion_rate: data.completion_rate };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addLog(`Thôi diễn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return { success: false, message: err.message, completion_rate: 0 };
    } finally {
      set({ actionLoading: false });
    }
  },

  breakWall: async (manaSpend) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    // Client-side validation to prevent unnecessary requests
    if (manaSpend < 0 || manaSpend > Math.floor(player.current_mana)) {
      set({ error: "Số lượng Chân Nguyên không hợp lệ." });
      return null;
    }

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/break-wall`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mana_spend: manaSpend}),
      });

      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Trùng kích thất bại."); }

      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Trùng kích hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Trùng kích lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  rechargeWithStones: async (stones) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/recharge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, stones: stones }),
      });

      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nạp nguyên thạch thất bại.");

      get().addLog(data.message || "Nạp nguyên thạch thành công.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Nạp nguyên thạch lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  useRelicGu: async (relicKey) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/use-relic-gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ relic_key: relicKey }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể dùng Cổ Xá Lợi."); }
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Đã sử dụng Cổ Xá Lợi.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Dùng Cổ Xá Lợi lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  ascension: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/ascension`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Thăng tiên thất bại."); }
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Thăng tiên hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Thăng tiên lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  completeQuest: async (questId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/quest/complete`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ quest_id: questId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể nhận thưởng nhiệm vụ."); }
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Hoàn thành nhiệm vụ.", "success");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(`Nhận thưởng nhiệm vụ lỗi: ${err.message}`, "error");
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  redeemGiftCode: async (code) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/redeem-gift-code`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mã quà không hợp lệ."); }
      get().applyInventoryFromResponse(data);
      get().addLog(data.message || "Đổi mã quà thành công!");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đổi mã quà lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },


  triggerTribulation: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/tribulation`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể kích hoạt thiên kiếp."); }
      get().addLog(data.message || "Thiên kiếp giáng lâm.");
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi độ kiếp: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },
});
