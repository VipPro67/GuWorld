import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

// Refine timed session slice (phiên luyện cổ) — mirrors backend
// /api/refine/sessions contract (WF-REF-05 §9.1) and the "refine" WS channel.

export interface RefineStepEntry {
  index: number;
  type: "TAP" | "SEQUENCE" | "CHOICE" | "ORDER";
  state: "DONE" | "OPEN" | "LOCKED" | "MISSED";
  score?: number | null;
}

export interface RefineActiveStep {
  index: number;
  type: "TAP" | "SEQUENCE" | "CHOICE" | "ORDER";
  nonce: string;
  window_start: string;
  window_end: string;
  params: {
    marker_period_ms?: number;
    center?: number;
    half_width?: number;
    symbols?: number[];
    display_ms?: number;
    target_input_ms?: number;
    situation?: string;
    options?: string[];
    items?: { id: string; name: string; icon: string; rank: number }[];
  };
}

export interface RefineRateBreakdownExtra {
  accuracy?: number;
  accuracy_bonus?: number;
  accuracy_max_bonus?: number;
  death_penalty?: number;
  step_scores?: number[];
}

export interface RefineSessionResult {
  reason: "completed" | "expired" | "aborted";
  outcome?: string;
  message?: string;
  new_gu?: ApiResponseData | null;
  rate_breakdown?: (Record<string, ApiResponseData> & RefineRateBreakdownExtra) | null;
}

export interface RefineSession {
  id: string;
  status: "ACTIVE" | "RESOLVED" | "ABORTED";
  recipe_id: string;
  target_gu_id: string;
  target_rank: number;
  variant: number;
  started_at: string;
  ends_at: string;
  accuracy: number;
  server_now: string;
  steps: RefineStepEntry[];
  step: RefineActiveStep | null;
  next_step_at: string | null;
  result?: RefineSessionResult | null;
  acknowledged_at?: string | null;
}

export interface RefineSessionSlice {
  refineSession: RefineSession | null;
  refinePanelOpen: boolean;
  applyRefineSessionEvent: (payload: ApiResponseData) => void;
  openRefinePanel: () => void;
  closeRefinePanel: () => void;
  fetchActiveRefineSession: () => Promise<void>;
  createRefineSession: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ success: boolean; message?: string; code?: string } | null>;
  submitRefineStep: (stepIndex: number, nonce: string, action: string, payload: unknown) => Promise<ApiResponseData | null>;
  abortRefineSession: () => Promise<void>;
  ackRefineSession: () => Promise<void>;
}

export const createRefineSessionSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, RefineSessionSlice> = (set, get) => ({
  refineSession: null,
  refinePanelOpen: false,

  applyRefineSessionEvent: (payload) => {
    try {
      const data = (payload?.data ?? payload) as ApiResponseData;
      if (!data || data.type !== "refine.session") return;
      const session = data.session as RefineSession | null | undefined;
      set({ refineSession: session ?? null });
      if (session && session.status !== "ACTIVE" && session.result?.message) {
        const ok = session.result.outcome === "SUCCESS";
        get().addLog(session.result.message);
        get().addToast(session.result.message, ok ? "success" : "error");
      }
      if (session && session.status !== "ACTIVE") {
        set({ refinePanelOpen: false });
      }
    } catch (err: unknown) {
      console.error("refine.session event error", err);
    }
  },

  openRefinePanel: () => {
    set({ refinePanelOpen: true });
  },

  closeRefinePanel: () => {
    set({ refinePanelOpen: false });
  },

  fetchActiveRefineSession: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/sessions/active`);
      if (!response.ok) return;
      const data = await response.json();
      set({ refineSession: data.session ?? null });
    } catch (err: unknown) {
      console.error("fetch refine session error", err);
    }
  },

  createRefineSession: async (recipeId, selectedInstanceIds) => {
    const { player, token } = get();
    if (!player || !token || !get().beginAction("player:cultivation")) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/sessions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ recipe_id: recipeId, selected_gu_instance_ids: selectedInstanceIds }),
      });
      const data = await response.json().catch(() => ({}));
      if (response.status === 409 && data.code === "REFINE_SESSION_ACTIVE") {
        get().addToast(data.error || "Đang có phiên luyện đang chạy!", "error");
        return { success: false, message: data.error, code: data.code };
      }
      if (!response.ok) {
        get().addToast(data.error || "Không thể mở phiên luyện Cổ.", "error");
        return { success: false, message: data.error, code: data.code };
      }
      get().applyInventoryFromResponse(data);
      const session = data.session as RefineSession | undefined;
      set({ refineSession: session ?? null, refinePanelOpen: true });
      get().addLog("Phiên luyện Cổ đã được khởi động. Hãy giữ nhịp các bước trong lò!");
      return { success: true, message: data.message };
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      get().addToast(message || "Không thể mở phiên luyện Cổ.", "error");
      return { success: false, message };
    } finally {
      get().endAction("player:cultivation");
    }
  },

  submitRefineStep: async (stepIndex, nonce, action, payload) => {
    const { refineSession, token } = get();
    if (!refineSession || !token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/sessions/${refineSession.id}/steps`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ step_index: stepIndex, nonce, action, payload, client_ts: Date.now() }),
      });
      const data = await response.json().catch(() => ({}));
      if (response.status === 401) return null;
      if (!response.ok) {
        get().addToast(data.error || "Bước luyện thất bại.", "error");
        if (data.session) set({ refineSession: data.session });
        return null;
      }
      if (data.session) {
        set({ refineSession: data.session });
      }
      if (data.player_ok === undefined && data.player) {
        // resolve path returns the full action payload with inventory
      }
      get().applyInventoryFromResponse(data);
      const result = data.result as RefineSessionResult | undefined;
      if (result && result.message) {
        const ok = result.outcome === "SUCCESS";
        get().addLog(result.message);
        get().addToast(result.message, ok ? "success" : "error");
        if (ok) get().openRefinePanel();
      }
      return data;
    } catch (err: unknown) {
      console.error("submit refine step error", err);
      return null;
    }
  },

  abortRefineSession: async () => {
    const { refineSession, token, player } = get();
    if (!refineSession || !player || !token || !get().beginAction("player:cultivation")) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/sessions/${refineSession.id}/abort`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        get().addToast(data.error || "Không thể hủy phiên luyện.", "error");
        return;
      }
      get().applyInventoryFromResponse(data);
      set({ refineSession: data.session ?? null, refinePanelOpen: false });
      if (data.session?.result?.message) get().addLog(data.session.result.message);
      get().addToast("Đã hủy phiên luyện. Nguyên liệu hoàn lại 50%, Cổ nguyên liệu rơi vào Hấp Hối.", "info");
    } catch (err: unknown) {
      console.error("abort refine session error", err);
    } finally {
      get().endAction("player:cultivation");
    }
  },

  ackRefineSession: async () => {
    const { refineSession, token } = get();
    if (!refineSession || !token) return;
    try {
      await authFetch(`${BACKEND_URL}/api/refine/sessions/${refineSession.id}/ack`, { method: "POST" });
      const session = get().refineSession;
      if (session) {
        set({ refineSession: { ...session, acknowledged_at: new Date().toISOString() } });
      }
    } catch (err: unknown) {
      console.error("ack refine session error", err);
    }
  },
});
