import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useUIStore } from "@/store/useUIStore";
import type { DeathArenaTournament, DeathArenaMatch, DeathArenaTeam, DeathArenaTeamMember, DeathArenaReward } from "@/store/useGameStore";

export interface ArenaSlice {
  arenaLeaderboard: ApiResponseData[];
  arenaDailyChallenges: number;
  arenaShopItems: ApiResponseData[];
  fetchArenaLeaderboard: () => Promise<void>;
  fetchArenaShop: () => Promise<void>;
  challengeArena: (targetPlayerId: string) => Promise<boolean>;
  buyArenaItem: (itemId: string, quantity?: number) => Promise<boolean>;
  deathArenaTournament: DeathArenaTournament | null;
  deathArenaBracket: DeathArenaMatch[];
  deathArenaMyTeam: DeathArenaTeam | null;
  deathArenaMyTeamMembers: DeathArenaTeamMember[];
  deathArenaMyMatches: DeathArenaMatch[];
  deathArenaRewards: DeathArenaReward[];
  deathArenaHistory: DeathArenaTournament[];
  deathArenaLoading: boolean;
  fetchDeathArenaCurrent: () => Promise<void>;
  registerDeathArena: () => Promise<boolean>;
  unregisterDeathArena: () => Promise<boolean>;
  fetchDeathArenaBracket: () => Promise<void>;
  fetchDeathArenaMyTeam: () => Promise<void>;
  fetchDeathArenaMyMatches: () => Promise<void>;
  fetchDeathArenaRewards: () => Promise<void>;
  fetchDeathArenaHistory: () => Promise<void>;
  claimDeathArenaReward: (rewardId: string) => Promise<boolean>;
  fetchDeathArenaMatchLog: (matchId: string) => Promise<ApiResponseData>;
  leaderboardEntries: ApiResponseData[];
  leaderboardCategory: string;
  myLeaderboardRank: { rank: number; value: number } | null;
  leaderboardTotalPlayers: number;
  leaderboardShowOnLeaderboard: boolean;
  leaderboardLoading: boolean;
  leaderboardLastUpdated: string;
  fetchLeaderboard: (category: string) => Promise<void>;
  toggleLeaderboardVisibility: () => Promise<void>;
  fetchLeaderboardSettings: () => Promise<void>;
  _onDeathArenaEvent: (payload: ApiResponseData) => void;
}

export const createArenaSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ArenaSlice> = (set, get) => ({
  arenaLeaderboard: [],
  arenaDailyChallenges: 0,
  arenaShopItems: [],
  deathArenaTournament: null,
  deathArenaBracket: [],
  deathArenaMyTeam: null,
  deathArenaMyTeamMembers: [],
  deathArenaMyMatches: [],
  deathArenaRewards: [],
  deathArenaHistory: [],
  deathArenaLoading: false,
  leaderboardEntries: [],
  leaderboardCategory: "rank",
  myLeaderboardRank: null,
  leaderboardTotalPlayers: 0,
  leaderboardShowOnLeaderboard: true,
  leaderboardLoading: false,
  leaderboardLastUpdated: "",

  fetchArenaLeaderboard: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/leaderboard`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        arenaLeaderboard: Array.isArray(data.leaderboard) ? data.leaderboard : [],
        arenaDailyChallenges: data.daily_challenges || 0,
      });
    } catch (err) {
      console.error("fetch arena leaderboard failed", err);
      get().addToast("Lỗi tải bảng xếp hạng.", "error");
    }
  },

  fetchArenaShop: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/shop`);
      if (response.ok) {
        const data = await response.json();
        set({ arenaShopItems: data.items || [] });
      }
    } catch (err) {
      console.error("fetch arena shop failed", err);
      get().addToast("Lỗi tải cửa hàng đấu trường.", "error");
    }
  },

  buyArenaItem: async (itemId, quantity) => {
    const { token } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/shop/buy`, {
        method: "POST",
        body: JSON.stringify({ item_id: itemId, quantity: quantity || 1 }),
      });
      const data = await response.json();
      if (response.ok) {
        get().addToast(data.message, "success");
        get().applyInventoryFromResponse(data);
        return true;
      } else {
        set({ error: data.error });
        get().addToast(data.error || "Lỗi giao dịch", "error");
        return false;
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast("Lỗi kết nối", "error");
      return false;
    } finally { get().endAction("arena"); }
  },

  challengeArena: async (targetPlayerId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/challenge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_player_id: targetPlayerId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khiêu chiến thất bại");
      useUIStore.getState().setView("arena");
      get().addLog(data.message || "Đã thách đấu đối thủ!");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { get().endAction("arena"); }
  },

  fetchDeathArenaCurrent: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ deathArenaLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/current`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set((state: ApiResponseData) => {
        const nextT = data.tournament || null;
        const curT = state.deathArenaTournament;
        const sameT = curT && nextT && curT.id === nextT.id && curT.status === nextT.status && curT.current_round === nextT.current_round;
        const nextTeam = data.team || null;
        const curTeam = state.deathArenaMyTeam;
        const sameTeam = curTeam && nextTeam && curTeam.id === nextTeam.id && curTeam.eliminated === nextTeam.eliminated;
        return {
          deathArenaTournament: sameT ? curT : nextT,
          deathArenaMyTeam: sameTeam ? curTeam : nextTeam,
          deathArenaMyTeamMembers: data.members || [],
        };
      });
    } catch (err) {
      console.error("fetch death arena current failed", err);
      get().addToast("Lỗi tải thông tin Đấu Trường Sinh Tử.", "error");
    } finally { set({ deathArenaLoading: false }); }
  },

  registerDeathArena: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/register`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng ký thất bại");
      get().addToast(data.message || "Đăng ký thành công!", "success");
      get().addLog(data.message || "Đã đăng ký Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaCurrent();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Đăng ký thất bại.", "error");
      return false;
    } finally { get().endAction("arena"); }
  },

  unregisterDeathArena: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/register`, { method: "DELETE" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đăng ký thất bại");
      get().addToast(data.message || "Hủy đăng ký thành công!", "success");
      get().addLog(data.message || "Đã hủy đăng ký Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaCurrent();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Hủy đăng ký thất bại.", "error");
      return false;
    } finally { get().endAction("arena"); }
  },

  fetchDeathArenaBracket: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/bracket`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set((state: ApiResponseData) => {
        const nextT = data.tournament || null;
        const curT = state.deathArenaTournament;
        const sameT = curT && nextT && curT.id === nextT.id && curT.status === nextT.status && curT.current_round === nextT.current_round;
        return { deathArenaTournament: sameT ? curT : (nextT ?? curT), deathArenaBracket: data.matches || [] };
      });
    } catch (err) {
      console.error("fetch death arena bracket failed", err);
      get().addToast("Lỗi tải bảng đấu.", "error");
    }
  },

  fetchDeathArenaMyTeam: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/my-team`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set((state: ApiResponseData) => {
        const nextTeam = data.team || null;
        const curTeam = state.deathArenaMyTeam;
        const sameTeam = curTeam && nextTeam && curTeam.id === nextTeam.id && curTeam.eliminated === nextTeam.eliminated;
        return { deathArenaMyTeam: sameTeam ? curTeam : nextTeam, deathArenaMyTeamMembers: data.members || [] };
      });
    } catch (err) {
      console.error("fetch death arena my team failed", err);
      get().addToast("Lỗi tải thông tin đội.", "error");
    }
  },

  fetchDeathArenaMyMatches: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/matches`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaMyMatches: data.matches || [] });
    } catch (err) {
      console.error("fetch death arena matches failed", err);
      get().addToast("Lỗi tải trận đấu.", "error");
    }
  },

  fetchDeathArenaRewards: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/rewards`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaRewards: data.rewards || [] });
    } catch (err) {
      console.error("fetch death arena rewards failed", err);
      get().addToast("Lỗi tải phần thưởng.", "error");
    }
  },

  fetchDeathArenaHistory: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/history`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaHistory: data.tournaments || [] });
    } catch (err) {
      console.error("fetch death arena history failed", err);
      get().addToast("Lỗi tải lịch sử.", "error");
    }
  },

  claimDeathArenaReward: async (rewardId) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("arena")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/claim-reward`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reward_id: rewardId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      get().addToast(data.message || "Nhận thưởng thành công!", "success");
      get().addLog(data.message || "Đã nhận phần thưởng Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaRewards();
      await get().fetchPlayerStatus(get().player!.id);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Nhận thưởng thất bại.", "error");
      return false;
    } finally { get().endAction("arena"); }
  },

  fetchDeathArenaMatchLog: async (matchId: string) => {
    const { token, logout } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/matches/${matchId}/log`);
      if (response.status === 401) { logout(); return null; }
      if (!response.ok) return null;
      return await response.json();
    } catch (err) {
      console.error("fetch match log failed", err);
      return null;
    }
  },

  fetchLeaderboard: async (category) => {
    const { token, logout } = get();
    if (!token) return;
    set({ leaderboardLoading: true, leaderboardCategory: category });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/${category}`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        leaderboardEntries: Array.isArray(data.entries) ? data.entries : [],
        myLeaderboardRank: data.my_rank ? { rank: data.my_rank, value: data.my_value } : null,
        leaderboardTotalPlayers: data.total_players || 0,
        leaderboardLastUpdated: data.last_updated || "",
      });
    } catch (err) {
      console.error("fetch leaderboard failed", err);
      get().addToast("Lỗi tải bảng xếp hạng.", "error");
    } finally { set({ leaderboardLoading: false }); }
  },

  toggleLeaderboardVisibility: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/toggle`, { method: "POST" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ leaderboardShowOnLeaderboard: data.show_on_leaderboard });
      get().addLog(data.show_on_leaderboard ? "Đã tham gia bảng xếp hạng" : "Đã rút khỏi bảng xếp hạng");
    } catch (err) {
      console.error("toggle leaderboard visibility failed", err);
      get().addToast("Lỗi thay đổi hiển thị.", "error");
    }
  },

  fetchLeaderboardSettings: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/settings`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ leaderboardShowOnLeaderboard: data.show_on_leaderboard });
    } catch (err) {
      console.error("fetch leaderboard settings failed", err);
      get().addToast("Lỗi tải cài đặt bảng xếp hạng.", "error");
    }
  },

  _onDeathArenaEvent: (payload: ApiResponseData) => {
    const eventType = payload?.type;
    const eventData = payload?.data;
    if (eventType === "match_updated" && eventData?.match_id) {
      set((state: ApiResponseData) => {
        const updated = (state.deathArenaBracket || []).map((m: ApiResponseData) =>
          m.id === eventData.match_id ? { ...m, winner_team_id: eventData.winner_team_id, status: eventData.status } : m
        );
        return { deathArenaBracket: updated };
      });
    } else if (eventType === "round_advanced" || eventType === "tournament_started") {
      get().fetchDeathArenaBracket();
      get().fetchDeathArenaCurrent();
    } else if (eventType === "tournament_finished") {
      set((state: ApiResponseData) => {
        if (!state.deathArenaTournament) return {};
        return { deathArenaTournament: { ...state.deathArenaTournament, status: "FINISHED", champion_team_id: eventData?.champion_team_id || state.deathArenaTournament.champion_team_id } };
      });
      get().fetchDeathArenaBracket();
    }
  },
});
