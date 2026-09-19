import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface TrackingMark {
  id: string;
  tracker_id: string;
  target_id: string;
  mark_type: "wisdom" | "blood" | "dao";
  power: number;
  expires_at: string;
  created_at: string;
  target_username?: string;
  target_avatar?: string;
  current_region?: string;
  x?: number;
  y?: number;
}

export interface DivinationReport {
  target_id: string;
  username: string;
  region: string;
  coordinates?: { x: number; y: number };
  approx_rank: number;
  sub_rank?: number;
  dominant_path: string;
  detected_gus: string[];
  status_summary: string;
  is_disguised: boolean;
}

export interface DivinationResponse {
  status_type: "full" | "partial" | "fogged" | "backfire";
  delta_score: number;
  divination_report?: DivinationReport;
  backfire_damage?: number;
  backfire_soul_damage?: number;
  can_apply_tracking_mark: boolean;
  message: string;
}

export interface ResonancePing {
  x: number;
  y: number;
  intensity: "high" | "medium" | "low";
  gu_id: string;
  entity_type: "wild_gu" | "player" | "mine";
}

export interface DaoPulsePing {
  region: string;
  x: number;
  y: number;
  path: string;
  event: string;
  expires_in_seconds: number;
}

export interface DivinationRequest {
  target_username: string;
  suspected_region?: string;
  suspected_sect?: string;
  suspected_gu?: string;
  item_clue_id?: string;
  resource_type: "mana" | "immortal_stone";
  resource_amount: number;
}

export interface DivineGuRequest {
  gu_query: string;
  target_rank?: number;
  resource_type: "mana" | "immortal_stone";
  resource_amount: number;
}

export interface DivineGuHolderInfo {
  holder_username?: string;
  region?: string;
  coordinates?: { x: number; y: number };
  is_vital?: boolean;
  equipped?: boolean;
  location_state?: string;
  concealed: boolean;
}

export interface DivineGuReport {
  gu_id: string;
  gu_name: string;
  rank: number;
  path: string;
  is_immortal: boolean;
  is_refined: boolean;
  world_count: number;
  holders?: DivineGuHolderInfo[];
  food_hint?: string;
  recipe_hint?: string;
  natural_habitat?: string;
  lore_status: string;
}

export interface DivineGuResponse {
  status_type: "full" | "partial" | "fogged" | "backfire";
  delta_score: number;
  report?: DivineGuReport;
  backfire_damage?: number;
  backfire_soul_damage?: number;
  message: string;
}

export type ScoutingTab = "player" | "gu" | "radar" | "sever";

export interface ScoutingSlice {
  trackedTargets: TrackingMark[];
  activeDivinationReport: DivinationResponse | null;
  activeGuDivinationReport: DivineGuResponse | null;
  divinationLoading: boolean;
  guDivinationLoading: boolean;
  resonancePings: ResonancePing[];
  daoPulsePings: DaoPulsePing[];
  isDivinationModalOpen: boolean;
  isRadarDrawerOpen: boolean;
  divinationTargetPreset: string;
  activeScoutingTab: ScoutingTab;

  openDivinationModal: (presetUsername?: string, initialTab?: ScoutingTab) => void;
  closeDivinationModal: () => void;
  setActiveScoutingTab: (tab: ScoutingTab) => void;
  toggleRadarDrawer: (open?: boolean) => void;

  executeDivination: (req: DivinationRequest) => Promise<DivinationResponse | null>;
  executeDivineGu: (req: DivineGuRequest) => Promise<DivineGuResponse | null>;
  applyTrackingMark: (targetId: string, markType?: "wisdom" | "blood") => Promise<boolean>;
  fetchTrackedTargets: () => Promise<void>;
  severClues: () => Promise<boolean>;
  executeResonanceScan: (targetPath?: string, minRank?: number, scanRegion?: string) => Promise<boolean>;
  inspectLocalPlayer: (targetId?: string, targetUsername?: string) => Promise<boolean>;
}

export const createScoutingSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ScoutingSlice> = (set, get) => ({
  trackedTargets: [],
  activeDivinationReport: null,
  activeGuDivinationReport: null,
  divinationLoading: false,
  guDivinationLoading: false,
  resonancePings: [],
  daoPulsePings: [],
  isDivinationModalOpen: false,
  isRadarDrawerOpen: false,
  divinationTargetPreset: "",
  activeScoutingTab: "player",

  openDivinationModal: (presetUsername = "", initialTab = "player") => {
    set({
      isDivinationModalOpen: true,
      divinationTargetPreset: presetUsername,
      activeScoutingTab: initialTab,
      activeDivinationReport: null,
      activeGuDivinationReport: null,
    });
    if (initialTab === "radar") {
      get().fetchTrackedTargets();
    }
  },

  closeDivinationModal: () => {
    set({ isDivinationModalOpen: false, divinationTargetPreset: "" });
  },

  setActiveScoutingTab: (tab: ScoutingTab) => {
    set({ activeScoutingTab: tab });
    if (tab === "radar") {
      get().fetchTrackedTargets();
    }
  },

  toggleRadarDrawer: (open) => {
    const current = get().isRadarDrawerOpen;
    const nextState = typeof open === "boolean" ? open : !current;
    set({ isRadarDrawerOpen: nextState });
    if (nextState) {
      get().fetchTrackedTargets();
    }
  },

  executeDivination: async (req: DivinationRequest) => {
    const { token, logout, addToast, addLog } = get();
    if (!token) return null;
    set({ divinationLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/divine`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Suy diễn thất bại.");
      }

      set({ activeDivinationReport: data });

      if (data.status_type === "backfire") {
        addToast(data.message, "error");
        addLog(`💥 [Thiên Cơ Phản Phệ] ${data.message}`);
      } else if (data.status_type === "full") {
        addToast(data.message, "success");
        addLog(`🔮 [Suy Diễn Đại Thành Công] ${data.message}`);
      } else if (data.status_type === "partial") {
        addToast(data.message, "info");
        addLog(`🔮 [Suy Diễn Tiểu Thành Công] ${data.message}`);
      } else {
        addToast(data.message, "warning");
      }

      await get().initializeSession();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi suy diễn.", "error");
      return null;
    } finally {
      set({ divinationLoading: false });
    }
  },

  executeDivineGu: async (req: DivineGuRequest) => {
    const { token, logout, addToast, addLog } = get();
    if (!token) return null;
    set({ guDivinationLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/divine-gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Suy tính cổ trùng thất bại.");
      }

      set({ activeGuDivinationReport: data });

      if (data.status_type === "backfire") {
        addToast(data.message, "error");
        addLog(`💥 [Thiên Cơ Phản Phệ] ${data.message}`);
      } else if (data.status_type === "full") {
        addToast(data.message, "success");
        addLog(`🐛 [Suy Tính Cổ Trùng - Đại Thành Công] ${data.message}`);
      } else if (data.status_type === "partial") {
        addToast(data.message, "info");
        addLog(`🐛 [Suy Tính Cổ Trùng - Tiểu Thành Công] ${data.message}`);
      } else {
        addToast(data.message, "warning");
      }

      await get().initializeSession();
      return data;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi suy tính cổ trùng.", "error");
      return null;
    } finally {
      set({ guDivinationLoading: false });
    }
  },

  applyTrackingMark: async (targetId: string, markType = "wisdom") => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !targetId) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/apply-mark`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_id: targetId, mark_type: markType }),
      });

      if (response.status === 401) {
        logout();
        return false;
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Gieo ấn ký thất bại.");
      }

      addToast(data.message || "Đã gieo ấn ký thành công!", "success");
      addLog(`🎯 [Ấn Ký Truy Tung] ${data.message}`);
      get().fetchTrackedTargets();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi gieo ấn ký.", "error");
      return false;
    }
  },

  fetchTrackedTargets: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/tracked-targets`);
      if (response.status === 401) {
        logout();
        return;
      }
      const data = await response.json();
      if (response.ok && data.tracked_targets) {
        set({ trackedTargets: data.tracked_targets });
      }
    } catch (err) {
      console.error("fetch tracked targets failed", err);
    }
  },

  severClues: async () => {
    const { token, logout, addToast, addLog } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/sever-clues`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ use_killer_move: false }),
      });

      if (response.status === 401) {
        logout();
        return false;
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thanh tẩy thất bại.");
      }

      addToast(data.message || "Đã chém đứt manh mối!", "success");
      addLog(`🗡️ [Đoạn Tuyệt Thiên Cơ] ${data.message}`);
      await get().initializeSession();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi chém đứt manh mối.", "error");
      return false;
    }
  },

  executeResonanceScan: async (targetPath = "", minRank = 0, scanRegion = "") => {
    const { token, logout, addToast, addLog } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/resonance-scan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          target_path: targetPath,
          min_rank: minRank,
          scan_region: scanRegion,
        }),
      });

      if (response.status === 401) {
        logout();
        return false;
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Cảm ứng thất bại.");
      }

      set({
        resonancePings: data.pings || [],
        daoPulsePings: data.dao_pulses || [],
      });

      const pingsCount = (data.pings || []).length;
      const pulseCount = (data.dao_pulses || []).length;
      addToast(`Cảm ứng phát hiện ${pingsCount} dao động Cổ trùng và ${pulseCount} xung kích Đạo ngân!`, "info");
      addLog(`📡 [Cảm Ứng Đạo Ngân] Đã quét thấy ${pingsCount} điểm dao động.`);
      await get().initializeSession();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi cảm ứng Cổ trùng.", "error");
      return false;
    }
  },

  inspectLocalPlayer: async (targetId = "", targetUsername = "") => {
    const { token, logout, addToast } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/scouting/inspect`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_id: targetId, target_username: targetUsername }),
      });

      if (response.status === 401) {
        logout();
        return false;
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể trinh sát mục tiêu.");
      }

      set({
        inspectedProfile: data,
        inspectedUsernameQuery: data.username || targetUsername,
      });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi trinh sát trực tiếp.", "error");
      return false;
    }
  },
});
