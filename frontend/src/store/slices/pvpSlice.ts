import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { wsManager } from "@/lib/websocket";

export type PvPCastMode = "TARGET" | "DIRECTION" | "GROUND" | "SELF";
export type PvPShape = "SINGLE" | "LINE" | "CIRCLE" | "CONE";

export interface PvPObstacle {
  kind: "circle" | "rect";
  x: number;
  y: number;
  r: number;
  w: number;
  h: number;
}

export interface PvPSkill {
  gu_id: string;
  rank: number;
  cast_mode: PvPCastMode;
  shape: PvPShape;
  radius: number;
  length: number;
  width: number;
  angle_deg: number;
  cast_range: number;
  cast_time_ms: number;
  cooldown_ms: number;
  move_distance: number;
  mana_cost: number;
}

export interface PvPSetupPlayer {
  id: string;
  name: string;
  is_dummy: boolean;
  x: number;
  y: number;
  hp: number;
  max_hp: number;
  mana: number;
  kit: PvPSkill[];
}

export interface PvPSetup {
  match_id: string;
  width: number;
  height: number;
  actor_radius: number;
  speed_to_tiles_per_tick: number;
  tick_ms: number;
  obstacles: PvPObstacle[];
  players: PvPSetupPlayer[];
}

export interface PvPCastState {
  gu_id: string;
  start_tick: number;
  ends_tick: number;
  cast_mode: PvPCastMode;
  shape: PvPShape;
  radius?: number;
  length?: number;
  width?: number;
  angle_deg?: number;
  move_distance?: number;
  aim_x: number;
  aim_y: number;
  target_id?: string;
}

export interface PvPParticipantState {
  id: string;
  name: string;
  is_dummy?: boolean;
  x: number;
  y: number;
  facing: number;
  hp: number;
  max_hp: number;
  mana: number;
  shield: number;
  speed: number;
  ack_seq: number;
  cast?: PvPCastState;
  cd_ms?: Record<string, number>;
  statuses?: string[];
}

export interface PvPProjectileState {
  id: number;
  gu_id: string;
  owner: string;
  x: number;
  y: number;
  dx: number;
  dy: number;
}

export interface PvPEvent {
  tick: number;
  type: string;
  source?: string;
  target?: string;
  gu_id?: string;
  x: number;
  y: number;
  damage?: number;
  amount?: number;
}

export interface PvPFrame {
  tick: number;
  ended: boolean;
  winner?: string;
  reason?: string;
  participants: PvPParticipantState[];
  projectiles: PvPProjectileState[];
  events: PvPEvent[];
}

export interface PvPCastIntent {
  gu_id: string;
  rank: number;
  mode: PvPCastMode;
  ax?: number;
  ay?: number;
  target_id?: string;
}

export interface PvPInput {
  seq: number;
  move?: { dx: number; dy: number };
  casts?: PvPCastIntent[];
}

/** Live control state the arena view owns and the canvas reads each frame. */
export interface PvPControlState {
  move: { dx: number; dy: number };
  /** Aim position in arena tiles. */
  aim: { x: number; y: number };
  skill: PvPSkill | null;
  /** True while the player is holding the cast button and the indicator shows. */
  aiming: boolean;
  /** True when the player is dragging into the cancel zone so the indicator turns red. */
  cancelCast?: boolean;
  /**
   * Written by the canvas every frame so the view can map a pointer position to
   * arena tiles without duplicating the fit-to-canvas math.
   */
  view: { scale: number; offsetX: number; offsetY: number };
}

export type PvPStatus = "idle" | "loading" | "live" | "ended";

export interface PvPSlice {
  pvpSetup: PvPSetup | null;
  pvpFrame: PvPFrame | null;
  pvpStatus: PvPStatus;
  pvpError: string | null;
  startPracticePvP: () => Promise<boolean>;
  stopPvP: () => void;
  sendPvPInput: (input: PvPInput) => void;
  clearPvPError: () => void;
}

// Module-level: a subscription handle is not render state.
let unsubscribeMatch: (() => void) | null = null;

export const createPvPSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, PvPSlice> = (set, get) => ({
  pvpSetup: null,
  pvpFrame: null,
  pvpStatus: "idle",
  pvpError: null,

  clearPvPError: () => set({ pvpError: null }),

  startPracticePvP: async () => {
    const { token, pvpStatus } = get();
    if (!token || pvpStatus === "loading") return false;

    set({ pvpStatus: "loading", pvpError: null, pvpFrame: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pvp/match/practice`, { method: "POST" });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}) as { error?: string });
        throw new Error(body.error || "Không thể mở trận đấu.");
      }
      const setup = (await response.json()) as PvPSetup;

      // The match channel only exists once the server has created the match.
      get().subscribeEvents();
      unsubscribeMatch?.();
      unsubscribeMatch = wsManager.subscribe<PvPFrame>(`match:${setup.match_id}`, (raw) => {
        // Go marshals a nil slice as null. Normalize once, here, so every
        // consumer can iterate the lists without guarding.
        const frame: PvPFrame = {
          ...raw,
          participants: raw.participants ?? [],
          projectiles: raw.projectiles ?? [],
          events: raw.events ?? [],
        };
        set({ pvpFrame: frame, pvpStatus: frame.ended ? "ended" : "live" });
      });

      set({ pvpSetup: setup, pvpStatus: "live" });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ pvpStatus: "idle", pvpError: err.message });
      get().addToast(err.message, "error");
      return false;
    }
  },

  stopPvP: () => {
    unsubscribeMatch?.();
    unsubscribeMatch = null;
    set({ pvpSetup: null, pvpFrame: null, pvpStatus: "idle", pvpError: null });
  },

  sendPvPInput: (input) => {
    if (!get().pvpSetup) return;
    wsManager.command("match.input", input);
  },
});
