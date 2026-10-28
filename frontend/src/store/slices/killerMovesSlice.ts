import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface KillerMoveGuOption {
  instance_id: string;
  gu_id: string;
  rank: number;
  hunger: number;
  state: string;
  location: string;
}

export interface KillerMoveEffectOption {
  key: string;
  source_instance_id: string;
  source_name: string;
  gu_id: string;
  rank: number;
  label: string;
  trigger: string;
  target: { selector: string; bypass_frontline?: boolean };
  operation: {
    op: string;
    sub_type?: string;
    value?: number;
    turns?: number;
    chance?: number;
    status_id?: string;
    stat?: string;
  };
}

export interface KillerMoveCombatProfile {
  id?: string;
  name?: string;
  rank: number;
  main_path: string;
  all_paths: string[];
  target_type: string;
  bypass: boolean;
  realtime_windup_ms?: number;
  realtime_cooldown_ms?: number;
  priority: number;
  mana_cost: number;
  hunger_cost: number;
  base_damage?: number;
  damage_mult?: number;
  base_shield?: number;
  base_healing?: number;
  lifesteal_pct?: number;
  crit_bonus?: number;
  success_chance: number;
  backfire_hp_pct: number;
  backfire_status?: string;
}

export interface GuHouseCombatProfile {
  id?: string;
  name?: string;
  rank: number;
  core_gu_ids: string[];
  aux_gu_count: number;
  max_house_hp: number;
  current_house_hp: number;
  house_defense: number;
  damage_reduction: number;
  immune_debuffs: boolean;
  zero_backfire: boolean;
  actions: KillerMoveCombatProfile[];
}

export interface GuHouseRecord {
  id: string;
  player_id: string;
  name: string;
  rank: number;
  core_gu_instances: string[];
  aux_gu_count: number;
  house_profile: GuHouseCombatProfile;
  current_durability: number;
  is_equipped: boolean;
  created_at: string;
}

export interface GuHouseDraft {
  name: string;
  core_gu_instance_ids: string[];
  aux_gu_count: number;
}

export interface KillerMoveRecord {
  id: string;
  name: string;
  main_gu_instance_id: string;
  main_gu_id: string;
  main_gu_rank: number;
  components: Array<{ instance_id: string; gu_id: string; rank: number }>;
  selected_effects: Array<{ key: string; trigger: string; instance_id: string }>;
  path: string;
  power_budget: number;
  power_load: number;
  harmony: number;
  complexity: number;
  success_chance: number;
  combat_profile?: KillerMoveCombatProfile;
  mastery_xp: number;
  mastery_level: number;
  is_equipped: boolean;
  available: boolean;
  realtime_supported?: boolean;
  realtime_unavailable_reason?: string;
  missing_gu?: string[];
  balance_version: number;
}

export interface KillerMoveDraft {
  name?: string;
  main_gu_instance_id: string;
  component_instance_ids: string[];
  selected_effect_keys: string[];
}

export interface KillerMovePreview {
  components: Array<{ instance_id: string; gu_id: string; rank: number }>;
  selected_effects: Array<{ key: string; instance_id: string }>;
  rank: number;
  path: string;
  power_budget: number;
  power_load: number;
  effect_scale: number;
  harmony: number;
  complexity: number;
  success_chance: number;
  mana_cost: number;
  combat_profile?: KillerMoveCombatProfile;
  warnings: string[];
}

export interface KillerMovesSlice {
  killerMoveGus: KillerMoveGuOption[];
  killerMoveEffects: KillerMoveEffectOption[];
  killerMoves: KillerMoveRecord[];
  killerMovesMaxSaved: number;
  killerMovesMaxEquipped: number;
  killerMovesLoading: boolean;
  guHouses: GuHouseRecord[];
  guHousesLoading: boolean;
  fetchKillerMoves: () => Promise<void>;
  previewKillerMove: (draft: KillerMoveDraft) => Promise<KillerMovePreview | null>;
  createKillerMove: (draft: KillerMoveDraft) => Promise<boolean>;
  toggleKillerMoveEquip: (id: string) => Promise<boolean>;
  deleteKillerMove: (id: string) => Promise<boolean>;
  useKillerMove: (id: string) => Promise<boolean>;
  fetchGuHouses: () => Promise<void>;
  previewGuHouse: (draft: GuHouseDraft) => Promise<GuHouseCombatProfile | null>;
  createGuHouse: (draft: GuHouseDraft) => Promise<boolean>;
  toggleEquipGuHouse: (id: string) => Promise<boolean>;
  repairGuHouse: (id: string) => Promise<boolean>;
}

export const createKillerMovesSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, KillerMovesSlice> = (set, get) => ({
  killerMoveGus: [],
  killerMoveEffects: [],
  killerMoves: [],
  killerMovesMaxSaved: 12,
  killerMovesMaxEquipped: 4,
  killerMovesLoading: false,
  guHouses: [],
  guHousesLoading: false,

  fetchKillerMoves: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ killerMovesLoading: true });
    try {
      const [optionsRes, movesRes] = await Promise.all([
        authFetch(`${BACKEND_URL}/api/killer-moves/options`),
        authFetch(`${BACKEND_URL}/api/killer-moves`),
      ]);
      if (optionsRes.status === 401 || movesRes.status === 401) { logout(); return; }
      const options = await optionsRes.json();
      const moves = await movesRes.json();
      if (!optionsRes.ok) throw new Error(options.error || "Không thể tải Cổ và hiệu ứng");
      if (!movesRes.ok) throw new Error(moves.error || "Không thể tải sát chiêu");
      set({
        killerMoveGus: options.gus || [],
        killerMoveEffects: options.effect_options || [],
        killerMoves: moves.killer_moves || [],
        killerMovesMaxSaved: moves.max_saved || options.max_saved || 12,
        killerMovesMaxEquipped: moves.max_equipped || options.max_equipped || 4,
        killerMovesLoading: false,
      });
    } catch (caught) {
      set({ killerMovesLoading: false });
      get().addToast(caught instanceof Error ? caught.message : "Không thể tải sát chiêu", "error");
    }
  },

  previewKillerMove: async (draft) => {
    const { token, logout } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves/preview`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(draft),
      });
      if (response.status === 401) { logout(); return null; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Công thức sát chiêu chưa hợp lệ");
      return data as KillerMovePreview;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể tính công thức", "error");
      return null;
    }
  },

  createKillerMove: async (draft) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(draft),
      });
      if (response.status === 401) { logout(); return false; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể lưu sát chiêu");
      get().addToast(data.message || "Đã sáng tạo sát chiêu", "success");
      await get().fetchKillerMoves();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể lưu sát chiêu", "error");
      return false;
    }
  },

  toggleKillerMoveEquip: async (id) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves/${encodeURIComponent(id)}/equip`, { method: "POST" });
      if (response.status === 401) { logout(); return false; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể trang bị sát chiêu");
      await get().fetchKillerMoves();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể trang bị sát chiêu", "error");
      return false;
    }
  },

  deleteKillerMove: async (id) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves/${encodeURIComponent(id)}`, { method: "DELETE" });
      if (response.status === 401) { logout(); return false; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể xóa sát chiêu");
      await get().fetchKillerMoves();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể xóa sát chiêu", "error");
      return false;
    }
  },

  useKillerMove: async (id) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves/${encodeURIComponent(id)}/use`, { method: "POST" });
      if (response.status === 401) { logout(); return false; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể dùng sát chiêu");
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Đã dùng sát chiêu", "success");
      await get().fetchKillerMoves();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể dùng sát chiêu", "error");
      return false;
    } finally {
      get().endAction("player:inventory");
    }
  },

  fetchGuHouses: async () => {
    const { token } = get();
    if (!token) return;
    set({ guHousesLoading: true });
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses`);
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể tải danh sách Tiên Cổ Ốc");
      set({ guHouses: data.gu_houses || [], guHousesLoading: false });
    } catch (caught) {
      set({ guHousesLoading: false });
      get().addToast(caught instanceof Error ? caught.message : "Lỗi tải Tiên Cổ Ốc", "error");
    }
  },

  previewGuHouse: async (draft) => {
    const { token } = get();
    if (!token) return null;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(draft),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Công thức Cổ Ốc chưa hợp lệ");
      return data as GuHouseCombatProfile;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Lỗi tính toán Tiên Cổ Ốc", "error");
      return null;
    }
  },

  createGuHouse: async (draft) => {
    const { token } = get();
    if (!token) return false;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(draft),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể chế tạo Tiên Cổ Ốc");
      get().addToast(data.message || "Đã chế tạo thành công Tiên Cổ Ốc", "success");
      await get().fetchGuHouses();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Lỗi chế tạo", "error");
      return false;
    }
  },

  toggleEquipGuHouse: async (id) => {
    const { token } = get();
    if (!token) return false;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses/${encodeURIComponent(id)}/equip`, { method: "POST" });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể cập nhật trang bị");
      await get().fetchGuHouses();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Lỗi trang bị Cổ Ốc", "error");
      return false;
    }
  },

  repairGuHouse: async (id) => {
    const { token } = get();
    if (!token) return false;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses/${encodeURIComponent(id)}/repair`, { method: "POST" });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể tu sửa Tiên Cổ Ốc");
      get().addToast(data.message || "Đã tu sửa Tiên Cổ Ốc hoàn hảo", "success");
      await get().fetchGuHouses();
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Lỗi tu sửa", "error");
      return false;
    }
  },
});
