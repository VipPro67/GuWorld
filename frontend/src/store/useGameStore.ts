import { create } from "zustand";

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
  equipped: boolean;
  is_vital: boolean;
}

export interface CombatState {
  enemy_name: string;
  enemy_hp: number;
  enemy_max_hp: number;
  enemy_damage: number;
  enemy_defense: number;
  charge_points: number;
  target_player_id?: string;
}

export interface Player {
  id: string;
  username: string;
  rank: number;
  sub_rank: number;
  aptitude_type: string;
  aptitude_percent: number;
  current_mana: number;
  max_mana: number;
  primeval_stones: number;
  immortal_stones: number;
  gu_inventory: GuInstance[];
  materials_inventory: Record<string, number>;
  hp: number;
  base_hp: number;
  body_mods: string[];
  current_region: string;
  dao_marks: Record<string, number>;
  path_masteries: Record<string, string>;
  x: number;
  y: number;
  explored_cells: string[];
  equipped_gus: string[];
  cleared_cells: Record<string, string>;
  active_combat: CombatState | null;
  last_mana_update: string;
  created_at: string;
}

export interface GuSpec {
  id: string;
  name: string;
  rank: number;
  path: string;
  gu_type: string;
  origin_region: string;
  base_mana_cost: number;
  effects: Record<string, number>;
  hunger_rate: number;
  food_type: string;
  max_hunger: number;
  description: string;
}

export interface ItemSpec {
  id: string;
  name: string;
  item_type: string;
  description: string;
}

interface AuthResponse {
  player: Player;
  token: string;
}

interface ExploreResponse {
  player: Player;
  reward_stones: number;
  log: string;
}

interface FeedResponse {
  player: Player;
  message: string;
}

export interface RefiningRecipe {
  id: string;
  target_gu_id: string;
  required_gus: Record<string, number>;
  required_materials: Record<string, number>;
  success_rate: number;
  primeval_stones_cost: number;
}

interface GameState {
  player: Player | null;
  token: string | null;
  isLoading: boolean;
  actionLoading: boolean;
  error: string | null;
  logs: string[];
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  dictionaryLoaded: boolean;
  recipes: RefiningRecipe[];
  recipesLoaded: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  register: (username: string, password: string, startingRegion: string, setVitalNow: boolean) => Promise<boolean>;
  logout: () => void;
  clearError: () => void;
  fetchPlayerStatus: (id: string) => Promise<void>;
  initializeSession: () => Promise<void>;
  explore: () => Promise<void>;
  feedGu: (guInstanceId: string) => Promise<void>;
  fetchDictionary: () => Promise<void>;
  fetchRecipes: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance } | null>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  executeCombatTurn: (equippedGuIds: string[], isCharge?: boolean) => Promise<{ player: Player; logs: string[]; enemy: CombatState | null; victory: boolean; defeat: boolean } | null>;
  addLog: (message: string) => void;
  currentView: string;
  setView: (view: string) => void;
  retreat: () => Promise<void>;
  equipGu: (instanceId: string, equip: boolean) => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useItem: (itemId: string) => Promise<void>;
  movePlayer: (x: number, y: number) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  fetchGrid: () => Promise<{ x: number; y: number; vision_radius: number; cells: any[]; explored_cells: string[]; stats: any } | null>;
  dropGu: (instanceId: string) => Promise<boolean>;
  sellShopItem: (itemId: string, quantity: number, itemType: "Gu" | "Material") => Promise<boolean>;
  marketplaceListings: any[];
  fetchMarketplaceListings: () => Promise<void>;
  createMarketplaceListing: (payload: { itemType: "Gu" | "Material"; guInstanceId?: string; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal" }) => Promise<boolean>;
  placeMarketplaceBid: (listingId: string, amount: number) => Promise<boolean>;
  cancelMarketplaceListing: (listingId: string) => Promise<boolean>;
}

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export const useGameStore = create<GameState>((set, get) => ({
  player: null,
  token: typeof window !== "undefined" ? localStorage.getItem("gu_auth_token") : null,
  isLoading: false,
  actionLoading: false,
  error: null,
  logs: ["Chào mừng đạo hữu đến với Cổ Giới. Hãy tu luyện và sinh tồn."],
  guDictionary: {},
  itemDictionary: {},
  dictionaryLoaded: false,
  recipes: [],
  recipesLoaded: false,
  currentView: "dashboard",
  marketplaceListings: [],
  setView: (view) => set({ currentView: view }),

  login: async (username, password) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Mật khẩu hoặc Đạo Hiệu không chính xác.");
      }

      const data: AuthResponse = await response.json();
      set({ player: data.player, token: data.token });
      localStorage.setItem("gu_auth_token", data.token);
      localStorage.setItem("gu_player_id", data.player.id);
      
      get().addLog(`Đồng chí ${data.player.username} đã quy vị! Tu vi: Nhất Chuyển Sơ Giai.`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  register: async (username, password, startingRegion, setVitalNow) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password, starting_region: startingRegion, set_vital_now: setVitalNow }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể khai mở Không Khiếu.");
      }

      const data: AuthResponse = await response.json();
      set({ player: data.player, token: data.token });
      localStorage.setItem("gu_auth_token", data.token);
      localStorage.setItem("gu_player_id", data.player.id);

      get().addLog(`Cổ sư ${data.player.username} khai mở Không Khiếu! Tư chất: ${data.player.aptitude_type} (${data.player.aptitude_percent}%)`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  logout: () => {
    set({ player: null, token: null });
    localStorage.removeItem("gu_auth_token");
    localStorage.removeItem("gu_player_id");
    get().addLog("Đã rời khỏi thế giới Cổ sư.");
  },

  clearError: () => set({ error: null }),

  addLog: (message) => {
    set((state) => ({
      logs: [message, ...state.logs].slice(0, 50),
    }));
  },

  initializeSession: async () => {
    const savedId = typeof window !== "undefined" ? localStorage.getItem("gu_player_id") : null;
    const savedToken = typeof window !== "undefined" ? localStorage.getItem("gu_auth_token") : null;
    if (!savedId || !savedToken) return;
    set({ token: savedToken });
    await get().fetchPlayerStatus(savedId);
  },

  fetchPlayerStatus: async (id) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return;

    try {
      const response = await fetch(`${BACKEND_URL}/api/player/status/${id}`, {
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }

      if (!response.ok) {
        throw new Error("Failed to fetch status");
      }

      const player: Player = await response.json();
      const currentPlayer = get().player;
      if (currentPlayer && currentPlayer.id === id) {
        set({ player });
      }
    } catch (err: any) {
      console.error("Status fetch failed:", err.message);
    }
  },

  explore: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/player/explore/${player.id}`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Khám phá thất bại.");
      }

      const data: ExploreResponse = await response.json();
      set({ player: data.player });
      get().addLog(data.log);
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  feedGu: async (guInstanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/feed`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({ gu_instance_id: guInstanceId }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho ăn.");
      }

      const data: FeedResponse = await response.json();
      set({ player: data.player });
      get().addLog(data.message);
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Cho Cổ ăn thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchDictionary: async () => {
    try {
      const response = await fetch(`${BACKEND_URL}/api/dictionary`);
      if (!response.ok) {
        throw new Error("Failed to load global metadata");
      }
      const data = await response.json();
      set({
        guDictionary: data.gu || {},
        itemDictionary: data.items || {},
        dictionaryLoaded: true,
      });
    } catch (err: any) {
      console.error("Dictionary sync error:", err.message);
    }
  },

  fetchRecipes: async () => {
    try {
      const response = await fetch(`${BACKEND_URL}/api/recipes`);
      if (!response.ok) {
        throw new Error("Failed to load recipes");
      }
      const data = await response.json();
      set({ recipes: data || [], recipesLoaded: true });
    } catch (err: any) {
      console.error("Recipes sync error:", err.message);
    }
  },

  refineGu: async (recipeId, selectedInstanceIds) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/refine`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          recipe_id: recipeId,
          selected_gu_instance_ids: selectedInstanceIds,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Hợp luyện thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message);
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Hợp luyện thất bại: ${err.message}`);
      return { status: "failed", message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  exploreNode: async (nodeId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/explore`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          player_id: player.id,
          node_id: nodeId,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Khám phá thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.log);
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  executeCombatTurn: async (equippedGuIds, isCharge = false) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/combat/attack`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          player_id: player.id,
          equipped_gu_ids: equippedGuIds,
          is_charge: isCharge,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tấn công thất bại.");
      }

      set({ player: data.player });
      if (data.logs && Array.isArray(data.logs)) {
        data.logs.forEach((logLine: string) => {
          get().addLog(logLine);
        });
      }
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tấn công thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  retreat: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/explore/retreat/${player.id}`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Rút lui thất bại.");
      }

      set({ player: data.player, currentView: "dashboard" });
      get().addLog(data.log || "Đạo hữu đã rút lui an toàn.");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rút lui thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },



  fetchMarketplaceListings: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, { headers: { "Authorization": `Bearer ${token}` } });
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ marketplaceListings: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("market listings failed", err);
    }
  },

  createMarketplaceListing: async (payload) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, { method: "POST", headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` }, body: JSON.stringify({ item_type: payload.itemType, gu_instance_id: payload.guInstanceId || "", item_id: payload.itemId || "", quantity: payload.quantity, price: payload.price, duration_hours: payload.durationHours, currency: payload.currency }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng bán thất bại");
      get().addLog(data.message || "Đăng bán thành công");
      await get().fetchPlayerStatus(localStorage.getItem("gu_player_id") || "");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Đăng bán thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  placeMarketplaceBid: async (listingId, amount) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/bao-hoang/bid`, { method: "POST", headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` }, body: JSON.stringify({ listing_id: listingId, amount }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đặt giá thất bại");
      get().addLog(data.message || "Đặt giá thành công");
      await get().fetchPlayerStatus(localStorage.getItem("gu_player_id") || "");
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/bao-hoang/cancel`, { method: "POST", headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` }, body: JSON.stringify({ listing_id: listingId }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy listing thất bại");
      get().addLog(data.message || "Đã hủy");
      await get().fetchPlayerStatus(localStorage.getItem("gu_player_id") || "");
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
  },
  buyShopItem: async (itemId, quantity) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/buy`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          player_id: player.id,
          item_id: itemId,
          quantity: quantity,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Giao dịch thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Mua hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },


  sellShopItem: async (itemId, quantity, itemType) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` },
        body: JSON.stringify({ player_id: player.id, item_id: itemId, quantity, item_type: itemType }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      set({ player: data.player });
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Bán hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  dropGu: async (instanceId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/drop`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` },
        body: JSON.stringify({ gu_instance_id: instanceId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      set({ player: data.player });
      get().addLog(data.message || "Đã vứt Cổ trùng.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Vứt Cổ thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },
  equipGu: async (instanceId, equip) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/equip`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          instance_id: instanceId,
          equip: equip,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thao tác trang bị thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Thao tác trang bị thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Trang bị Cổ trùng thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  updateGuStatus: async (instanceId, status) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          instance_id: instanceId,
          status: status,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Cập nhật trạng thái thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Cập nhật trạng thái Cổ trùng thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi cập nhật trạng thái: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  setVitalGu: async (instanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/set-vital`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thiết lập Bản Mệnh Cổ thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Thiết lập Bản Mệnh Cổ thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi thiết lập Bản Mệnh Cổ: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  useGu: async (instanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/inventory/use-gu`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng Cổ trùng thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Sử dụng Cổ trùng thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi sử dụng Cổ trùng: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  useItem: async (itemId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/player/use-item`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          item_id: itemId,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng vật phẩm thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.message || "Sử dụng vật phẩm thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  movePlayer: async (x, y) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/explore/move`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({
          x: x,
          y: y,
        }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      set({ player: data.player });
      get().addLog(data.log || "Di chuyển thành công.");
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Di chuyển thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchGrid: async () => {
    const { token, logout } = get();
    if (!token) return null;

    try {
      const response = await fetch(`${BACKEND_URL}/api/explore/grid`, {
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        return null;
      }

      if (!response.ok) {
        throw new Error("Failed to load map grid viewport");
      }

      const data = await response.json();
      return data;
    } catch (err: any) {
      console.error("Grid fetch failed:", err.message);
      return null;
    }
  },
}));
