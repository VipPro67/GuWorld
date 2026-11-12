import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface TeleportArray {
  id: string;
  name: string;
  owner_id: string;
  owner_name?: string;
  sect_id?: string;
  sect_name?: string;
  region_id: string;
  region_name?: string;
  x: number;
  y: number;
  tier: string;
  max_range: number;
  is_cross_region: boolean;
  direction_type: "TWO_WAY" | "ONE_WAY_SEND" | "ONE_WAY_RECEIVE";
  access_policy: "PERSONAL" | "SECT" | "PUBLIC";
  fee_enabled: boolean;
  fee_currency: "primeval_stones" | "immortal_stones";
  fee_amount: number;
  vault_balance: number;
  current_durability: number;
  max_durability: number;
  is_active: boolean;
  effect_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface TeleportFormationGu {
  instance_id: string;
  gu_id: string;
  rank: number;
  hunger: number;
  state: string;
  location: string;
  is_locked: boolean;
}

export interface TeleportFormationEffectOption {
  key: string;
  source_block_id: string;
  operation_index: number;
  source_instance_id: string;
  source_name: string;
  gu_id: string;
  rank: number;
  label: string;
  trigger: string;
  operation: { op: string; sub_type?: string; value?: number; status_id?: string; resource?: string; turns?: number; chance?: number };
  target: { selector: string; bypass_frontline?: boolean };
  conditions?: Array<{ field: string; operator: string; value: number }>;
  travel_supported?: boolean;
  composition_capabilities?: string[];
}

export interface TeleportFormationRuleSet {
  schema_version: number;
  revision: string;
  rules: Array<{ id: string; product_profile: string; context: string; executor: string; kind: "COMBINE"; core_capability: string; support_capability: string }>;
}

export interface TeleportFormationCompositionGraph {
  schema_version: number;
  catalog_draft_id?: string;
  product: string;
  product_profile: string;
  context: string;
  executor: string;
  core_gu_instance_id: string;
  output_effect_key: string;
  rule_set_revision: string;
  nodes: Array<{ effect_key: string; source_block_id: string; source_revision?: string; operation_index: number; role: "OUTPUT" | "SUPPORT"; gu_instance_id: string; gu_id: string; rank: number; trigger: string; context: string; executor: string }>;
  edges: Array<{ from: string; to: string; kind: "COMBINE"; rule_id: string }>;
}

export interface TeleportCompositionTemplate {
  id: string;
  name: string;
  definition: { output_effect_key: string; sources: Array<{ key: string; gu_id: string; rank: number; block_id: string; operation_index: number; role: "OUTPUT" | "SUPPORT" }> };
}

export interface TeleportFormationEffectPreview {
  effect_enabled: boolean;
  product_profile: string;
  context: string;
  executor: string;
  gu_rank: number;
  composition_graph: TeleportFormationCompositionGraph;
  component_gu_instances: string[];
  effects: { schema_version: number; blocks: Array<{ id: string; trigger: string; target: { selector: string }; operations: TeleportFormationEffectOption["operation"][] }> };
  preview_fingerprint: string;
}

export interface TeleportTemplate {
  tier: string;
  name: string;
  description: string;
  min_rank: number;
  min_formation_realm?: number;
  base_success_rate?: number;
  max_range: number;
  is_cross_region: boolean;
  max_durability: number;
  cost_primeval_stones: number;
  cost_immortal_stones: number;
  required_gu: string[];
  required_items: Record<string, number>;
}

export interface TeleportDirectoryItem {
  array: TeleportArray;
  distance: number;
  is_cross_region_jump: boolean;
  can_use: boolean;
  reason?: string;
  fee_to_pay: number;
  fee_currency: string;
}

export interface TeleportDirectoryData {
  is_standing_on_array: boolean;
  source_array: TeleportArray | null;
  destinations: TeleportDirectoryItem[];
}

export interface TeleportBuildResult {
  success: boolean;
  message: string;
  array?: TeleportArray;
  success_rate?: number;
  formation_realm?: string;
  formation_level?: number;
  discount_percent?: number;
  mastery_gained?: number;
  paid_primeval_stones?: number;
  paid_immortal_stones?: number;
}

export interface TeleportSlice {
  teleportTemplates: TeleportTemplate[];
  myTeleportArrays: TeleportArray[];
  teleportDirectory: TeleportDirectoryData | null;
  teleportLoading: boolean;
  teleportFormationGus: TeleportFormationGu[];
  teleportFormationEffects: TeleportFormationEffectOption[];
  teleportFormationRuleSet: TeleportFormationRuleSet | null;
  teleportCompositionTemplates: TeleportCompositionTemplate[];

  fetchTeleportTemplates: () => Promise<void>;
  fetchMyTeleportArrays: () => Promise<void>;
  fetchTeleportDirectory: () => Promise<void>;
  fetchTeleportFormationOptions: () => Promise<void>;
  previewTeleportArray: (req: TeleportBuildRequest) => Promise<{ success: boolean; preview?: TeleportFormationEffectPreview; message: string }>;
  buildTeleportArray: (req: TeleportBuildRequest) => Promise<TeleportBuildResult>;
  travelTeleport: (targetArrayId: string) => Promise<{ success: boolean; message: string }>;
  manageTeleportArray: (req: {
    array_id: string;
    name?: string;
    direction_type?: string;
    access_policy?: string;
    fee_enabled?: boolean;
    fee_currency?: string;
    fee_amount?: number;
  }) => Promise<boolean>;
  repairTeleportArray: (arrayId: string, amount?: number) => Promise<boolean>;
  withdrawTeleportVault: (arrayId: string, amount?: number) => Promise<boolean>;
  dismantleTeleportArray: (arrayId: string) => Promise<boolean>;
}

export interface TeleportBuildRequest {
    name: string;
    tier: string;
    direction_type?: string;
    access_policy?: string;
    fee_enabled?: boolean;
    fee_currency?: string;
    fee_amount?: number;
    composition_draft_id?: string;
    composition_core_gu_instance_id?: string;
    composition_graph?: TeleportFormationCompositionGraph;
    preview_fingerprint?: string;
}

export const createTeleportSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, TeleportSlice> = (set, get) => ({
  teleportTemplates: [],
  myTeleportArrays: [],
  teleportDirectory: null,
  teleportLoading: false,
  teleportFormationGus: [],
  teleportFormationEffects: [],
  teleportFormationRuleSet: null,
  teleportCompositionTemplates: [],

  fetchTeleportTemplates: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/templates`);
      if (!response.ok) return;
      const data: TeleportTemplate[] = await response.json();
      set({ teleportTemplates: data });
    } catch (err) {
      console.error("Failed to fetch teleport templates", err);
    }
  },

  fetchTeleportFormationOptions: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/effect-options`);
      if (!response.ok) return;
      const data: { gus: TeleportFormationGu[]; effect_options: TeleportFormationEffectOption[]; composition_rule_set?: TeleportFormationRuleSet; composition_templates?: TeleportCompositionTemplate[] } = await response.json();
      set({ teleportFormationGus: data.gus || [], teleportFormationEffects: data.effect_options || [], teleportFormationRuleSet: data.composition_rule_set || null, teleportCompositionTemplates: data.composition_templates || [] });
    } catch (err) {
      console.error("Failed to fetch teleport formation effect options", err);
    }
  },

  previewTeleportArray: async (req) => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Preview effect Trận Pháp thất bại");
      return { success: true, preview: data as TeleportFormationEffectPreview, message: "Preview hợp lệ" };
    } catch (caught: unknown) {
      const error = caught instanceof Error ? caught : new Error(String(caught));
      return { success: false, message: error.message };
    }
  },

  fetchMyTeleportArrays: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/mine`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data: TeleportArray[] = await response.json();
      set({ myTeleportArrays: data });
    } catch (err) {
      console.error("Failed to fetch player teleport arrays", err);
    }
  },

  fetchTeleportDirectory: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      set({ teleportLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/teleport/directory`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data: TeleportDirectoryData = await response.json();
      set({ teleportDirectory: data });
    } catch (err) {
      console.error("Failed to fetch teleport directory", err);
    } finally {
      set({ teleportLoading: false });
    }
  },

  buildTeleportArray: async (req) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return { success: false, message: "Đang bận" };
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/build`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bố trí trận pháp thất bại");

      if (data.success) {
        get().addToast(data.message || "Bố trí trận pháp thành công!", "success");
        get().addLog(data.message || "Bố trí trận pháp thành công!");
      } else {
        get().addToast(data.message || "Bố trận thất bại!", "warning");
        get().addLog(data.message || "Bố trận thất bại!");
      }

      await get().fetchPlayerProfile();
      await get().fetchMyTeleportArrays();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Thất bại: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      get().endAction("player:position");
    }
  },

  travelTeleport: async (targetArrayId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return { success: false, message: "Đang bận" };
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/travel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_array_id: targetArrayId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Truyền tống thất bại");

      get().addToast(data.message || "Truyền tống thành công!", "success");
      get().addLog(data.message || "Truyền tống thành công!");
      if (data.player) {
        get().setPlayerFromServer({
          ...data.player,
          world_zone_id: data.player.current_region,
          world_instance_id: "",
        });
      }
      if (data.realtime_transfer) {
        // The backend released the realtime lease after persisting the
        // teleport; rejoin the destination zone with a fresh session.
        get().requestWorldReconnect();
      }
      await get().fetchTeleportDirectory();
      return { success: true, message: data.message };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Truyền tống thất bại: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      get().endAction("player:position");
    }
  },

  manageTeleportArray: async (req) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/manage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật cấu hình thất bại");

      get().addToast(data.message || "Cập nhật thành công!", "success");
      get().addLog(data.message || "Cập nhật thành công!");
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  repairTeleportArray: async (arrayId, amount = 0) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/repair`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sửa chữa thất bại");

      get().addToast(data.message || "Sửa chữa thành công!", "success");
      get().addLog(data.message || "Sửa chữa thành công!");
      if (data.player) {
        get().setPlayerFromServer({
          ...data.player,
          world_zone_id: data.player.current_region,
          world_instance_id: "",
        });
      }
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  withdrawTeleportVault: async (arrayId, amount = 0) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/withdraw`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút tiền thất bại");

      get().addToast(data.message || "Rút tiền thành công!", "success");
      get().addLog(data.message || "Rút tiền thành công!");
      if (data.player) {
        get().setPlayerFromServer({
          ...data.player,
          world_zone_id: data.player.current_region,
          world_instance_id: "",
        });
      }
      await get().fetchMyTeleportArrays();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },

  dismantleTeleportArray: async (arrayId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("player:position")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/teleport/dismantle`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ array_id: arrayId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thu hồi thất bại");

      get().addToast(data.message || "Thu hồi trận pháp thành công!", "success");
      get().addLog(data.message || "Thu hồi trận pháp thành công!");
      if (data.player) {
        get().setPlayerFromServer({
          ...data.player,
          world_zone_id: data.player.current_region,
          world_instance_id: "",
        });
      }
      await get().fetchMyTeleportArrays();
      await get().fetchTeleportDirectory();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("player:position");
    }
  },
});
