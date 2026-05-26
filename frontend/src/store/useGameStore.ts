import { create } from "zustand";

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
}

export interface CombatState {
  enemy_name: string;
  enemy_hp: number;
  enemy_max_hp: number;
  enemy_damage: number;
  enemy_defense: number;
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
  gu_inventory: GuInstance[];
  materials_inventory: Record<string, number>;
  hp: number;
  current_region: string;
  dao_marks: Record<string, number>;
  active_combat: CombatState | null;
  last_mana_update: string;
  created_at: string;
}

export interface GuSpec {
  id: string;
  name: string;
  rank: number;
  type: string; // "Active", "Passive", "Support"
  base_power: number;
  hunger_rate: number;
  food_type: string;
  max_hunger: number;
  description: string;
  origin_region: string;
  base_damage: number;
  base_defense: number;
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
  register: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  clearError: () => void;
  fetchPlayerStatus: (id: string) => Promise<void>;
  explore: () => Promise<void>;
  feedGu: (guInstanceId: string) => Promise<void>;
  fetchDictionary: () => Promise<void>;
  fetchRecipes: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance } | null>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  executeCombatTurn: (equippedGuIds: string[]) => Promise<{ player: Player; logs: string[]; enemy: CombatState | null; victory: boolean; defeat: boolean } | null>;
  addLog: (message: string) => void;
  currentView: string;
  setView: (view: string) => void;
  retreat: () => Promise<void>;
  buyShopItem: (itemId: string, quantity: number) => Promise<boolean>;
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

  register: async (username, password) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${BACKEND_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
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

  executeCombatTurn: async (equippedGuIds) => {
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
}));
