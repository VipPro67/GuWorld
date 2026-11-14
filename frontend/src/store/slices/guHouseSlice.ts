import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type {
  KillerMoveCombatProfile,
  KillerMoveCompositionRuleSet,
  KillerMoveCompositionTemplate,
  KillerMoveCompositionGraph,
} from "@/store/slices/killerMovesSlice";

export interface GuHouseCombatProfile {
  profile_version?: number;
  product_profile?: string;
  context?: string;
  executor?: string;
  preview_fingerprint?: string;
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
  action_effect_keys?: Record<string, string>;
  action_composition_draft_ids?: Record<string, string>;
  action_composition_graphs?: Record<string, KillerMoveCompositionGraph>;
  preview_fingerprint?: string;
}

export interface GuHouseSlice {
  guHouses: GuHouseRecord[];
  guHouseCompositionRuleSet: KillerMoveCompositionRuleSet | null;
  guHouseCompositionTemplates: KillerMoveCompositionTemplate[];
  guHousesMinCoreGus: number;
  guHousesMaxCoreGus: number;
  guHousesLoading: boolean;
  fetchGuHouses: () => Promise<void>;
  previewGuHouse: (draft: GuHouseDraft) => Promise<GuHouseCombatProfile | null>;
  createGuHouse: (draft: GuHouseDraft) => Promise<boolean>;
  toggleEquipGuHouse: (id: string) => Promise<boolean>;
  repairGuHouse: (id: string) => Promise<boolean>;
  useGuHouse: (id: string, coreGuInstanceID: string, effectKey: string) => Promise<boolean>;
}

export const createGuHouseSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, GuHouseSlice> = (set, get) => ({
  guHouses: [],
  guHouseCompositionRuleSet: null,
  guHouseCompositionTemplates: [],
  guHousesMinCoreGus: 2,
  guHousesMaxCoreGus: 5,
  guHousesLoading: false,

  fetchGuHouses: async () => {
    const { token } = get();
    if (!token) return;
    set({ guHousesLoading: true });
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses`);
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể tải danh sách Tiên Cổ Ốc");
      set({
        guHouses: data.gu_houses || [],
        guHouseCompositionRuleSet: data.composition_rule_set || null,
        guHouseCompositionTemplates: data.composition_templates || [],
        guHousesMinCoreGus: data.min_core_gus || 2,
        guHousesMaxCoreGus: data.max_core_gus || 5,
        guHousesLoading: false,
      });
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

  useGuHouse: async (id, coreGuInstanceID, effectKey) => {
    const { token } = get();
    if (!token || !get().beginAction("player:inventory")) return false;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/gu-houses/${encodeURIComponent(id)}/use`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ core_gu_instance_id: coreGuInstanceID, effect_key: effectKey }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể dùng Tiên Cổ Ốc");
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Tiên Cổ Ốc đã thi triển", "success");
      await Promise.all([get().fetchGuHouses(), get().fetchKillerMoves()]);
      return true;
    } catch (caught) {
      get().addToast(caught instanceof Error ? caught.message : "Không thể dùng Tiên Cổ Ốc", "error");
      return false;
    } finally {
      get().endAction("player:inventory");
    }
  },
});
