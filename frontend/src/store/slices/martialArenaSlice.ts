import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface MartialTournament {
  id: string;
  status: string;
  rank: number;
  scheduled_at: string;
  started_at?: string;
  finished_at?: string;
  total_players: number;
  current_round: number;
  total_rounds: number;
  champion_player_id?: string;
  created_at: string;
}

export interface MartialCombatant {
  id: string;
  tournament_id: string;
  player_id: string;
  username: string;
  avatar_url: string;
  seed_number: number;
  rank: number;
  sub_rank: number;
  eliminated: boolean;
  eliminated_at_round?: number;
  final_placement?: number;
  created_at: string;
}

export interface MartialMatch {
  id: string;
  tournament_id: string;
  round_number: number;
  bracket_position: number;
  combatant_a_id: string;
  combatant_a_name: string;
  combatant_b_id?: string;
  combatant_b_name?: string;
  winner_combatant_id?: string;
  status: string;
  reward_stones: number;
  created_at: string;
  finished_at?: string;
}

export interface MartialReward {
  id: string;
  tournament_id: string;
  player_id: string;
  username: string;
  placement: number;
  combatant_id: string;
  title: string;
  reward_stones: number;
  reward_immortal_stones: number;
  reward_gu_id?: string;
  reward_gu_rank?: number;
  reward_items: ApiResponseData[];
  claimed: boolean;
  created_at: string;
}

export interface MartialCurrentEntry {
  rank: number;
  tournament: MartialTournament;
  my_combatant?: MartialCombatant;
}

export interface MartialBracketEntry {
  rank: number;
  tournament: MartialTournament;
  matches: MartialMatch[];
}

export interface MartialArenaSlice {
  martialCurrent: MartialCurrentEntry[];
  martialBracket: MartialBracketEntry[];
  martialMyMatches: MartialMatch[];
  martialRewards: MartialReward[];
  martialHistory: MartialTournament[];
  martialLoading: boolean;
  fetchMartialCurrent: () => Promise<void>;
  fetchMartialBracket: () => Promise<void>;
  fetchMartialMyMatches: () => Promise<void>;
  fetchMartialRewards: () => Promise<void>;
  fetchMartialHistory: () => Promise<void>;
  claimMartialReward: (rewardId: string) => Promise<boolean>;
  fetchMartialMatchLog: (matchId: string) => Promise<ApiResponseData>;
}

export const createMartialArenaSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, MartialArenaSlice> = (set, get) => ({
  martialCurrent: [],
  martialBracket: [],
  martialMyMatches: [],
  martialRewards: [],
  martialHistory: [],
  martialLoading: false,

  fetchMartialCurrent: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ martialLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/current`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ martialCurrent: Array.isArray(data.tournaments) ? data.tournaments : [] });
    } catch (err) {
      console.error("fetch martial arena current failed", err);
      get().addToast("Lỗi tải thông tin Đại Hội Võ Thuật.", "error");
    } finally {
      set({ martialLoading: false });
    }
  },

  fetchMartialBracket: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/bracket`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ martialBracket: Array.isArray(data.tournaments) ? data.tournaments : [] });
    } catch (err) {
      console.error("fetch martial arena bracket failed", err);
      get().addToast("Lỗi tải bảng đấu.", "error");
    }
  },

  fetchMartialMyMatches: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/matches`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ martialMyMatches: Array.isArray(data.matches) ? data.matches : [] });
    } catch (err) {
      console.error("fetch martial arena matches failed", err);
      get().addToast("Lỗi tải trận đấu.", "error");
    }
  },

  fetchMartialRewards: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/rewards`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ martialRewards: Array.isArray(data.rewards) ? data.rewards : [] });
    } catch (err) {
      console.error("fetch martial arena rewards failed", err);
      get().addToast("Lỗi tải phần thưởng.", "error");
    }
  },

  fetchMartialHistory: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/history`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ martialHistory: Array.isArray(data.tournaments) ? data.tournaments : [] });
    } catch (err) {
      console.error("fetch martial arena history failed", err);
      get().addToast("Lỗi tải lịch sử.", "error");
    }
  },

  claimMartialReward: async (rewardId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/claim-reward`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reward_id: rewardId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      get().addToast(data.message || "Nhận thưởng thành công!", "success");
      await get().fetchMartialRewards();
      await get().fetchPlayerStatus(get().player!.id);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Nhận thưởng thất bại.", "error");
      return false;
    } finally {
      get().endAction("arena");
    }
  },

  fetchMartialMatchLog: async (matchId: string) => {
    const { token, logout } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/martial-arena/matches/${matchId}/log`);
      if (response.status === 401) { logout(); return null; }
      if (!response.ok) return null;
      return await response.json();
    } catch (err) {
      console.error("fetch martial match log failed", err);
      return null;
    }
  },
});
