import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface RealmRules {
  duration_minutes: number;
  pvp_mode: "SAFE" | "FREE_PK" | "FACTION_PK" | string;
  max_allowed_rank: number;
  apply_rank_suppression: boolean;
  death_penalty: "EVICT" | "SAFE_RESPAWN" | string;
  allow_teleport: boolean;
  entry_item: string;
  entry_item_count: number;
}

export interface SpatialRealmItem {
  template_id: string;
  name: string;
  type: string;
  scope: "SHARED" | "PARTY" | "SOLO" | string;
  description: string;
  required_rank: number;
  duration_minutes: number;
  entry_item: string;
  entry_item_count: number;
  rules: RealmRules;
  spatial_region_id: string;
  entry_region_id: string;
  entry_x: number;
  entry_y: number;
  active_instance_id?: string;
  player_count: number;
  expires_at?: string;
}

export interface ActiveSpatialRealm {
  template_id: string;
  name: string;
  type: string;
  scope: string;
  description: string;
  rules: RealmRules;
  instance_id: string;
  region_id: string;
  expires_at: string;
  player_count: number;
}

export interface AltarInfo {
  altar_id: string;
  altar_name: string;
  path: string;
  tier: "MORTAL" | "IMMORTAL" | string;
  x: number;
  y: number;
  in_range: boolean;
  req_rank: number;
  req_dao_marks: Record<string, number>;
  required_item: string;
  required_item_count: number;
  guardian_defeated: boolean;
  already_claimed: boolean;
  legacy_text: string;
  rewards: {
    gus: string[];
    recipes: string[];
    primeval_stones: number;
    materials: Record<string, number>;
  };
}

export interface SpatialRealmSlice {
  spatialRealms: SpatialRealmItem[];
  activeSpatialRealm: ActiveSpatialRealm | null;
  spatialLoading: boolean;
  selectedAltarInfo: AltarInfo | null;
  altarModalOpen: boolean;
  altarLoading: boolean;
  spatialDirectoryOpen: boolean;

  fetchSpatialRealms: () => Promise<void>;
  fetchActiveSpatialRealm: () => Promise<void>;
  enterSpatialRealm: (templateId: string) => Promise<boolean>;
  exitSpatialRealm: () => Promise<boolean>;
  fetchAltarInfo: (altarId: string) => Promise<AltarInfo | null>;
  claimAltar: (altarId: string) => Promise<boolean>;
  setAltarModalOpen: (open: boolean) => void;
  setSelectedAltarInfo: (info: AltarInfo | null) => void;
  setSpatialDirectoryOpen: (open: boolean) => void;
}

let fetchSpatialRealmsPromise: Promise<void> | null = null;

export const createSpatialRealmSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, SpatialRealmSlice> = (set, get) => ({
  spatialRealms: [],
  activeSpatialRealm: null,
  spatialLoading: false,
  selectedAltarInfo: null,
  altarModalOpen: false,
  altarLoading: false,
  spatialDirectoryOpen: false,

  fetchSpatialRealms: async () => {
    if (fetchSpatialRealmsPromise) return fetchSpatialRealmsPromise;
    const executeFetch = async () => {
      const { token } = get();
      if (!token) return;
      try {
        set({ spatialLoading: true });
        const response = await authFetch(`${BACKEND_URL}/api/spatial-realms`);
        if (response.ok) {
          const data = await response.json();
          set({
            spatialRealms: data.realms || [],
          });
        }
      } catch (err) {
        console.error("Failed to fetch spatial realms:", err);
      } finally {
        set({ spatialLoading: false });
        fetchSpatialRealmsPromise = null;
      }
    };
    fetchSpatialRealmsPromise = executeFetch();
    return fetchSpatialRealmsPromise;
  },

  fetchActiveSpatialRealm: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/spatial-realms/active`);
      if (response.ok) {
        const data = await response.json();
        set({
          activeSpatialRealm: data.active ? data.realm : null,
        });
      } else {
        set({ activeSpatialRealm: null });
      }
    } catch (err) {
      console.error("Failed to fetch active spatial realm:", err);
      set({ activeSpatialRealm: null });
    }
  },

  enterSpatialRealm: async (templateId: string) => {
    const { token, addToast } = get();
    if (!token) return false;
    try {
      set({ spatialLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/spatial-realms/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ template_id: templateId }),
      });
      const data = await response.json();
      if (!response.ok) {
        addToast(data.error || "Không thể tiến vào bí cảnh", "error");
        return false;
      }
      if (data.zone_id && data.world_instance_id) {
        get().setPlayerFromServer({
          current_region: data.region_id,
          map_instance_id: data.instance_id,
          world_zone_id: data.zone_id,
          world_instance_id: data.world_instance_id,
          x: data.x,
          y: data.y,
        });
        get().requestWorldReconnect();
        get().setView("mmo-world");
      }
      addToast("Đã bước qua không gian môn hộ, tiến vào bí cảnh!", "success");
      await get().fetchActiveSpatialRealm();
      return true;
    } catch (err) {
      console.error("Failed to enter spatial realm:", err);
      addToast("Lỗi kết nối khi tiến vào bí cảnh", "error");
      return false;
    } finally {
      set({ spatialLoading: false });
    }
  },

  exitSpatialRealm: async () => {
    const { token, addToast } = get();
    if (!token) return false;
    try {
      set({ spatialLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/spatial-realms/exit`, {
        method: "POST",
      });
      const data = await response.json();
      if (!response.ok) {
        addToast(data.error || "Không thể rời khỏi bí cảnh", "error");
        return false;
      }
      if (data.zone_id) {
        get().setPlayerFromServer({
          current_region: data.region_id,
          map_instance_id: null,
          world_zone_id: data.zone_id,
          world_instance_id: data.world_instance_id || "",
          x: data.x,
          y: data.y,
        });
        get().requestWorldReconnect();
      }
      addToast("Đã an toàn quay trở về thế giới bên ngoài!", "success");
      set({ activeSpatialRealm: null });
      return true;
    } catch (err) {
      console.error("Failed to exit spatial realm:", err);
      addToast("Lỗi kết nối khi rời bí cảnh", "error");
      return false;
    } finally {
      set({ spatialLoading: false });
    }
  },

  fetchAltarInfo: async (altarId: string) => {
    const { token } = get();
    if (!token || !altarId) return null;
    try {
      set({ altarLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/spatial-realms/altar-info?altar_id=${encodeURIComponent(altarId)}`);
      if (response.ok) {
        const data = await response.json();
        set({ selectedAltarInfo: data, altarModalOpen: true });
        return data;
      }
      return null;
    } catch (err) {
      console.error("Failed to fetch altar info:", err);
      return null;
    } finally {
      set({ altarLoading: false });
    }
  },

  claimAltar: async (altarId: string) => {
    const { token, addToast, refreshPlayerSummary } = get();
    if (!token || !altarId) return false;
    try {
      set({ altarLoading: true });
      const response = await authFetch(`${BACKEND_URL}/api/spatial-realms/claim-altar`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ altar_id: altarId }),
      });
      const data = await response.json();
      if (!response.ok) {
        addToast(data.error || "Không thể tiếp nhận truyền thừa", "error");
        return false;
      }
      addToast(`Đắc nhận truyền thừa: [${data.altar_name}]!`, "success");
      await refreshPlayerSummary();
      await get().fetchAltarInfo(altarId);
      return true;
    } catch (err) {
      console.error("Failed to claim altar:", err);
      addToast("Lỗi kết nối khi tiếp nhận truyền thừa", "error");
      return false;
    } finally {
      set({ altarLoading: false });
    }
  },

  setAltarModalOpen: (open: boolean) => set({ altarModalOpen: open }),
  setSelectedAltarInfo: (info: AltarInfo | null) => set({ selectedAltarInfo: info }),
  setSpatialDirectoryOpen: (open: boolean) => set({ spatialDirectoryOpen: open }),
});
