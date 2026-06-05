import { create } from "zustand";

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
  equipped: boolean;
  is_vital: boolean;
  integration_time?: number;
}

export interface Combatant {
  id: string;
  name: string;
  kind: string; // "Player", "Normal", "Boss", etc.
  rank: number;
  sub_rank: number;
  hp: number;
  max_hp: number;
  current_mana: number;
  max_mana: number;
  base_attack: number;
  base_defense: number;
  gu_inventory?: GuInstance[];
  dao_marks?: Record<string, number>;
  path_masteries?: Record<string, string>;
  ai_profile?: string;
  loot_table_id?: string;
  grid_x?: number;
  grid_y?: number;
  speed?: number;
  virtual_shield?: number;
}

export interface CombatState {
  enemy_name: string;
  enemy_hp: number;
  enemy_max_hp: number;
  enemy_damage: number;
  enemy_defense: number;
  charge_points: number;
  target_player_id?: string;
  enemy_template_id?: string;
  allies?: Combatant[];
  enemies?: Combatant[];
  enemy_type?: string;
  enemy_rank?: number;
  enemy_sub_rank?: number;
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
  wall_progress?: number;
  human_qi?: number;
  next_tribulation_time?: string | null;
  tribulation_count?: number;
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
  lifespan: number;
  max_lifespan: number;
  time_speed_multiplier: number;
  party_id?: string | null;
  arena_rank?: number | null;
  completed_quests: string[];
  aperture_stress?: number;
  last_cultivate_time?: string;
  temp_loot?: {
    gu_inventory: GuInstance[];
    materials_inventory: Record<string, number>;
    primeval_stones: number;
  };
}

export interface ImmortalAperture {
  id: string;
  player_id: string;
  grade: 'LOW' | 'MID' | 'HIGH' | 'EXTREME';
  time_rate: number;
  aperture_hp: number;
  max_aperture_hp: number;
  area_size: number;
  immortal_essence: number;
  essence_production_per_year: number;
  internal_years_passed: number;
  next_tribulation_year: number;
  tribulation_count: number;
  state: 'STABLE' | 'TRIBULATION_PENDING' | 'TRIBULATION_ACTIVE' | 'DAMAGED' | 'DESTROYED';
  created_at: string;
  updated_at: string;
}

export interface DaoMark {
  id: string;
  player_id: string;
  path: string;
  amount: number;
  created_at: string;
  updated_at: string;
}

export interface ApertureResourceNode {
  id: string;
  aperture_id: string;
  node_type: string;
  name: string;
  path: string;
  level: number;
  hp: number;
  max_hp: number;
  x: number;
  y: number;
  production_type: string;
  production_amount: number;
  production_interval_hours: number;
  last_collected_at: string | null;
  is_destroyed: boolean;
  created_at: string;
  updated_at: string;
}

export interface TribulationEvent {
  id: string;
  aperture_id: string;
  player_id: string;
  tribulation_type: 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION';
  path: string;
  difficulty: number;
  status: 'ACTIVE' | 'SUCCESS' | 'FAILED';
  aperture_hp_before: number;
  aperture_hp_after: number | null;
  reward_dao_marks: number;
  started_at: string;
  ended_at: string | null;
}

export interface TribulationEnemy {
  id: string;
  event_id: string;
  name: string;
  enemy_type: string;
  path: string;
  hp: number;
  max_hp: number;
  attack: number;
  defense: number;
  target_node_id: string | null;
  x: number;
  y: number;
  is_boss: boolean;
  is_dead: boolean;
  created_at: string;
  updated_at: string;
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
  food_types: string[];
  max_hunger: number;
  description: string;
}

export interface ItemSpec {
  id: string;
  name: string;
  item_type: string;
  type?: string;
  rarity?: number;
  rank?: number;
  path?: string;
  effects?: Record<string, number>;
  tags?: string[];
  description: string;
}

interface AuthResponse {
  player: Player;
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
  immortal_essence_stones_cost: number;
}

export interface ToastMessage {
  id: string;
  message: string;
  type: "success" | "error" | "info";
  duration?: number;
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
  register: (username: string, password: string, email: string, phoneNumber: string, startingRegion: string, setVitalNow: boolean) => Promise<boolean>;
  logout: () => void;
  clearError: () => void;
  fetchPlayerStatus: (id: string) => Promise<void>;
  initializeSession: () => Promise<void>;
  explore: () => Promise<void>;
  feedGu: (guInstanceId: string) => Promise<void>;
  fetchDictionary: () => Promise<void>;
  fetchRecipes: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance } | null>;
  breakWall: (manaSpend: number) => Promise<{ success: boolean; message: string } | null>;
  useRelicGu: (relicKey: string) => Promise<{ success: boolean; message: string } | null>;
  ascension: () => Promise<{ success: boolean; message: string } | null>;
  triggerTribulation: () => Promise<{ success: boolean; message: string } | null>;
  completeQuest: (questId: string) => Promise<{ success: boolean; message: string } | null>;
  rechargeWithStones: (stones: number) => Promise<{ success: boolean; message: string } | null>;
  clearStress: () => Promise<{ success: boolean; message: string } | null>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  executeCombatTurn: (equippedGuIds: string[], isCharge?: boolean, targetId?: string, targetX?: number, targetY?: number) => Promise<{ player: Player; logs: string[]; enemy: CombatState | null; victory: boolean; defeat: boolean } | null>;
  addLog: (message: string) => void;
  currentView: string;
  setView: (view: string) => void;
  retreat: () => Promise<void>;
  equipGu: (instanceId: string, equip: boolean) => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useItem: (itemId: string) => Promise<void>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  fetchGrid: () => Promise<{ x: number; y: number; vision_radius: number; cells: any[]; explored_cells: string[]; stats: any; players_on_map?: any[] } | null>;
  dropGu: (instanceId: string) => Promise<boolean>;
  buyShopItem: (itemId: string, quantity: number) => Promise<boolean>;
  sellShopItem: (itemId: string, quantity: number, itemType: "Gu" | "Material") => Promise<boolean>;
  marketplaceListings: any[];
  fetchMarketplaceListings: () => Promise<void>;
  createMarketplaceListing: (payload: { itemType: "Gu" | "Material"; guInstanceId?: string; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal" }) => Promise<boolean>;
  placeMarketplaceBid: (listingId: string, amount: number) => Promise<boolean>;
  cancelMarketplaceListing: (listingId: string) => Promise<boolean>;

  // Party State & Actions
  party: { id: string; leader_id: string; members: { id: string; username: string }[] } | null;
  createParty: () => Promise<boolean>;
  fetchPartyStatus: () => Promise<void>;
  invitePlayer: (username: string) => Promise<boolean>;
  leaveParty: () => Promise<boolean>;

  // Arena State & Actions
  arenaLeaderboard: any[];
  fetchArenaLeaderboard: () => Promise<void>;
  challengeArena: (targetPlayerId: string) => Promise<boolean>;

  // Chat/Mail State & Actions
  inbox: any[];
  baoHoangMessages: any[];
  sendPrivateMail: (receiverUsername: string, content: string) => Promise<boolean>;
  fetchInbox: () => Promise<void>;
  sendBaoHoangChat: (content: string, channel: string) => Promise<boolean>;
  fetchBaoHoangMessages: (channel: string) => Promise<void>;

  // Mine State & Actions
  mineSlots: MineSlot[];
  fetchMineSlots: () => Promise<void>;
  occupyMineSlot: (slotId: number) => Promise<boolean>;
  claimMineRewards: (slotId: number) => Promise<boolean>;
  abandonMineSlot: (slotId: number) => Promise<boolean>;
  attackMineSlot: (slotId: number) => Promise<boolean>;

  // Toast System State & Actions
  toasts: ToastMessage[];
  addToast: (message: string, type?: "success" | "error" | "info", duration?: number) => void;
  removeToast: (id: string) => void;

  // Immortal Aperture State & Actions
  aperture: ImmortalAperture | null;
  daoMarks: DaoMark[];
  apertureNodes: ApertureResourceNode[];
  activeTribulation: { event: TribulationEvent; enemies: TribulationEnemy[] } | null;
  fetchAperture: () => Promise<void>;
  fetchDaoMarks: () => Promise<void>;
  fetchApertureNodes: () => Promise<void>;
  placeApertureNode: (nodeType: string, x: number, y: number) => Promise<boolean>;
  collectResourceNode: (nodeId: string) => Promise<boolean>;
  uninstallApertureNode: (nodeId: string) => Promise<boolean>;
  deleteApertureNode: (nodeId: string) => Promise<boolean>;
  fetchActiveTribulation: () => Promise<void>;
  attackTribulationEnemy: (eventId: string, enemyId: string, guId: string) => Promise<boolean>;
  resolveTribulation: (eventId: string) => Promise<boolean>;
}

export interface MineSlot {
  id: number;
  rarity: string;
  occupier_id?: string;
  occupier_name?: string;
  occupied_at?: string;
  last_claimed_at?: string;
  accumulated_stones: number;
  accumulated_qi: number;
  accumulated_chests: number;
}


const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const AUTHENTICATED_SESSION = "cookie-session";

const authFetch = (input: RequestInfo | URL, init: RequestInit = {}) =>
  fetch(input, {
    ...init,
    credentials: "include",
  });

export const useGameStore = create<GameState>((set, get) => {
  const normalizePlayer = (player: Player | null): Player | null => {
    if (!player) return null;
    const guInventory = (player.gu_inventory || []).map((g) => ({ ...g, equipped: !!g.equipped || !!g.is_vital }));
    const equippedSet = new Set<string>(player.equipped_gus || []);
    const vital = guInventory.find((g) => g.is_vital);
    if (vital && vital.status !== "DEAD") equippedSet.add(vital.instance_id);
    return { ...player, gu_inventory: guInventory, equipped_gus: Array.from(equippedSet) };
  };

  const translateIDs = (msg: string): string => {
    if (!msg) return "";
    let formatted = msg;
    const { guDictionary, itemDictionary } = get();

    // Replace Gu IDs
    Object.keys(guDictionary || {}).forEach((id) => {
      const name = guDictionary[id]?.name;
      if (name) {
        formatted = formatted.split(id).join(name);
      }
    });

    // Replace Item IDs
    Object.keys(itemDictionary || {}).forEach((id) => {
      const name = itemDictionary[id]?.name;
      if (name) {
        formatted = formatted.split(id).join(name);
      }
    });

    return formatted;
  };

  return ({
  player: null,
  token: null,
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
  party: null,
  arenaLeaderboard: [],
  inbox: [],
  baoHoangMessages: [],
  mineSlots: [],
  aperture: null,
  daoMarks: [],
  apertureNodes: [],
  activeTribulation: null,
  setView: (view) => set({ currentView: view }),

  toasts: [],
  addToast: (message, type = "info", duration = 4000) => {
    const formatted = translateIDs(message);
    const id = Math.random().toString(36).substring(2, 9);
    set((state) => ({
      toasts: [...state.toasts, { id, message: formatted, type, duration }],
    }));
    setTimeout(() => {
      get().removeToast(id);
    }, duration);
  },
  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  login: async (username, password) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Mật khẩu hoặc Đạo Hiệu không chính xác.");
      }

      const data: AuthResponse = await response.json();
      set({ player: normalizePlayer(data.player), token: AUTHENTICATED_SESSION });
      localStorage.removeItem("gu_auth_token");
      localStorage.setItem("gu_player_id", data.player.id);
      
      get().addLog(`Đồng chí ${data.player.username} đã quy vị!`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  register: async (username, password, email, phoneNumber, startingRegion, setVitalNow) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username,
          password,
          email,
          phone_number: phoneNumber,
          starting_region: startingRegion,
          set_vital_now: setVitalNow,
        }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể khai mở Không Khiếu.");
      }

      const data: AuthResponse = await response.json();
      set({ player: normalizePlayer(data.player), token: AUTHENTICATED_SESSION });
      localStorage.removeItem("gu_auth_token");
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
    void authFetch(`${BACKEND_URL}/api/auth/logout`, { method: "POST" });
    set({ player: null, token: null });
    localStorage.removeItem("gu_auth_token");
    localStorage.removeItem("gu_player_id");
    get().addLog("Đã rời khỏi thế giới Cổ sư.");
  },

  clearError: () => set({ error: null }),

  addLog: (message) => {
    const formatted = translateIDs(message);
    set((state) => ({
      logs: [formatted, ...state.logs].slice(0, 50),
    }));
  },

  initializeSession: async () => {
    if (typeof window !== "undefined") {
      localStorage.removeItem("gu_auth_token");
    }

    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/me`);
      if (response.status === 401 || response.status === 403) {
        set({ player: null, token: null });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to restore session");
      }

      const player: Player = await response.json();
      set({ player: normalizePlayer(player), token: AUTHENTICATED_SESSION });
      if (typeof window !== "undefined") {
        localStorage.setItem("gu_player_id", player.id);
      }
    } catch (err: any) {
      console.error("Session restore failed:", err.message);
      set({ player: null, token: null });
    }
  },

  fetchPlayerStatus: async (id) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/status/${id}`);

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
        set({ player: normalizePlayer(player) });
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
      const response = await authFetch(`${BACKEND_URL}/api/player/explore/${player.id}`, {
        method: "POST",
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
      set({ player: normalizePlayer(data.player) });
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
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
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
      set({ player: normalizePlayer(data.player) });
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Cho Cổ ăn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchDictionary: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dictionary`);
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
      const response = await authFetch(`${BACKEND_URL}/api/recipes`);
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
      get().addToast(data.message, data.status === "success" ? "success" : "error");
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Hợp luyện thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return { status: "failed", message: err.message };
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

      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Trùng kích thất bại."); }

      set({ player: data.player });
      get().addLog(data.message || "Trùng kích hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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

      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nạp nguyên thạch thất bại.");

      set({ player: normalizePlayer(data.player) });
      get().addLog(data.message || "Nạp nguyên thạch thành công.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể dùng Cổ Xá Lợi."); }
      set({ player: data.player });
      get().addLog(data.message || "Đã sử dụng Cổ Xá Lợi.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Thăng tiên thất bại."); }
      set({ player: data.player });
      get().addLog(data.message || "Thăng tiên hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể nhận thưởng nhiệm vụ."); }
      set({ player: normalizePlayer(data.player) });
      get().addLog(data.message || "Hoàn thành nhiệm vụ.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Nhận thưởng nhiệm vụ lỗi: ${err.message}`);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể kích hoạt thiên kiếp."); }
      set({ player: data.player });
      get().addLog(data.message || "Thiên kiếp giáng lâm.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi độ kiếp: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  exploreNode: async (nodeId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
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

  clearStress: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/clear-stress`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tịnh hóa Không khiếu thất bại.");

      set({ player: data.player });
      get().addLog(data.message || "Tịnh hóa Không khiếu hoàn tất.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tịnh hóa lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  executeCombatTurn: async (equippedGuIds, isCharge = false, targetId, targetX, targetY) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/attack`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          equipped_gu_ids: equippedGuIds,
          is_charge: isCharge,
          target_id: targetId || "",
          target_x: targetX || 0,
          target_y: targetY || 0,
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
      const response = await authFetch(`${BACKEND_URL}/api/explore/retreat/${player.id}`, {
        method: "POST",
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
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, {});
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
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ item_type: payload.itemType, gu_instance_id: payload.guInstanceId || "", item_id: payload.itemId || "", quantity: payload.quantity, price: payload.price, duration_hours: payload.durationHours, currency: payload.currency }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng bán thất bại");
      get().addLog(data.message || "Đăng bán thành công");
      await get().initializeSession();
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
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/bid`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId, amount }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đặt giá thất bại");
      get().addLog(data.message || "Đặt giá thành công");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/cancel`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy listing thất bại");
      get().addLog(data.message || "Đã hủy");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
  },
  buyShopItem: async (itemId, quantity) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;

    set({ actionLoading: true, error: null });
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
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
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
      const response = await authFetch(`${BACKEND_URL}/api/inventory/drop`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
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

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thao tác trang bị thất bại.");
      }

      set({ player: data.player });
      const successMsg = data.message || (equip ? "Trang bị Cổ trùng thành công!" : "Tháo Cổ trùng thành công!");
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Thao tác trang bị thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  updateGuStatus: async (instanceId, status) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
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
      const response = await authFetch(`${BACKEND_URL}/api/inventory/set-vital`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
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
      const successMsg = data.message || "Thiết lập Bản Mệnh Cổ thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi thiết lập Bản Mệnh Cổ: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  useGu: async (instanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
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

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng Cổ trùng thất bại.");
      }

      set({ player: data.player });
      const successMsg = data.message || "Sử dụng Cổ trùng thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi sử dụng Cổ trùng: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  useItem: async (itemId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
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

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng vật phẩm thất bại.");
      }

      set({ player: data.player });
      const successMsg = data.message || "Sử dụng vật phẩm thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  movePlayer: async (x, y, attackTargetId?) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
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
      const response = await authFetch(`${BACKEND_URL}/api/explore/grid`, {
        headers: {
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

  createParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/create`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo tổ đội thất bại");
      set({ party: data.party });
      get().addLog(data.message || "Đã lập tổ đội thành công!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi tạo party: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyStatus: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/status`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ party: data.party });
    } catch (err) {
      console.error("fetch party status failed", err);
    }
  },

  invitePlayer: async (username) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invite`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể mời");
      set({ party: data.party });
      get().addLog(data.message || `Đã mời người chơi ${username} vào tổ đội.`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mời tổ đội thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  leaveParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/leave`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tổ đội thất bại");
      set({ party: null });
      get().addLog(data.message || "Bạn đã rời khỏi tổ đội.");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rời tổ đội thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchArenaLeaderboard: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/leaderboard`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ arenaLeaderboard: Array.isArray(data.leaderboard) ? data.leaderboard : [] });
    } catch (err) {
      console.error("fetch arena leaderboard failed", err);
    }
  },

  challengeArena: async (targetPlayerId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/challenge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_player_id: targetPlayerId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khiêu chiến thất bại");
      set({ player: data.player, currentView: "dashboard" }); // Force update player state which triggers combat screen
      get().addLog(data.message || "Đã thách đấu đối thủ!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khiêu chiến võ đài lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  sendPrivateMail: async (receiverUsername, content) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: receiverUsername, content }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi thư thất bại");
      get().addLog(data.message || "Đã gửi thư truyền tin!");
      await get().initializeSession(); // update stone count
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Truyền tin lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchInbox: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/inbox`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ inbox: Array.isArray(data.inbox) ? data.inbox : [] });
    } catch (err) {
      console.error("fetch inbox failed", err);
    }
  },

  sendBaoHoangChat: async (content, channel) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content, channel }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi tin nhắn thất bại");
      get().addLog(data.message || "Đã phát truyền tin nhắn!");
      await get().initializeSession(); // refresh player state (mana, inventory, etc)
      await get().fetchBaoHoangMessages(channel);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi gửi tin nhắn: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchBaoHoangMessages: async (channel) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=${channel}`);
      if (response.status === 401 || response.status === 403) { return; }
      const data = await response.json();
      set({ baoHoangMessages: Array.isArray(data.messages) ? data.messages : [] });
    } catch (err) {
      console.error("fetch chat messages failed", err);
    }
  },

  fetchMineSlots: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/slots`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ mineSlots: Array.isArray(data) ? data : [] });
      }
    } catch (err) {
      console.error("fetch mine slots failed", err);
    }
  },

  occupyMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/occupy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chiếm mỏ thất bại");
      get().addLog(data.message || "Chiếm mỏ thành công!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Chiếm mỏ lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  claimMineRewards: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      get().addLog(data.message || "Đã nhận thưởng tích lũy!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Nhận thưởng mỏ lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  abandonMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/abandon`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút lui thất bại");
      get().addLog(data.message || "Đã rời khỏi mỏ và nhận hết thưởng!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rời mỏ lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  attackMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tấn công mỏ thất bại");
      get().addLog(data.message || "Bắt đầu thách đấu chiếm mỏ!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tấn công mỏ lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchAperture: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (!response.ok) {
        set({ aperture: null });
        return;
      }
      const data = await response.json();
      set({ aperture: data });
    } catch (err) {
      console.error("Failed to fetch aperture:", err);
      set({ aperture: null });
    }
  },

  fetchDaoMarks: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dao-marks`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ daoMarks: data });
      }
    } catch (err) {
      console.error("Failed to fetch Dao Marks:", err);
    }
  },

  fetchApertureNodes: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ apertureNodes: data });
      }
    } catch (err) {
      console.error("Failed to fetch aperture nodes:", err);
    }
  },

  placeApertureNode: async (nodeType: string, x: number, y: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_type: nodeType, x, y }),
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể xây dựng điểm tài nguyên");
      }
      get().addToast(data.message || "Xây dựng mỏ tài nguyên thành công!", "success");
      get().addLog(data.message || "Đã đặt điểm tài nguyên mới.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  collectResourceNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/collect`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể thu hoạch tài nguyên");
      }
      get().addToast(data.message || "Thu hoạch thành công!", "success");
      get().addLog(data.message || "Thu hoạch tài nguyên thành công.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  uninstallApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/uninstall`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể tháo dỡ mỏ tài nguyên");
      }
      get().addToast(data.message || "Tháo dỡ thành công!", "success");
      get().addLog(data.message || "Đã tháo dỡ mỏ tài nguyên.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  deleteApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}`, {
        method: "DELETE",
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể san phẳng mỏ tài nguyên");
      }
      get().addToast(data.message || "San phẳng thành công!", "success");
      get().addLog(data.message || "Đã san phẳng mỏ tài nguyên.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchActiveTribulation: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/active`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ activeTribulation: data });
      }
    } catch (err) {
      console.error("Failed to fetch active tribulation:", err);
    }
  },

  attackTribulationEnemy: async (eventId: string, enemyId: string, guId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enemy_id: enemyId, gu_id: guId }),
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tấn công thất bại");
      }
      get().addToast(data.message || "Tấn công kẻ địch tai kiếp!", "success");
      get().addLog(data.message || "Gây sát thương lên kẻ địch tai kiếp.");
      await get().initializeSession();
      await get().fetchActiveTribulation();
      await get().fetchAperture();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  resolveTribulation: async (eventId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/resolve`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Kết toán tai kiếp thất bại");
      }
      get().addToast(data.message || "Kết toán tai kiếp hoàn thành!", "success");
      get().addLog(data.message || "Kết toán tai kiếp hoàn thành.");
      await get().initializeSession();
      await get().fetchAperture();
      await get().fetchDaoMarks();
      set({ activeTribulation: null });
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },
})});
