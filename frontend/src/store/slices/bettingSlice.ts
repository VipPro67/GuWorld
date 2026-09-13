import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface BettingSlice {
  currentBettingRound: any | null;
  myBetTeam: string | null;
  myBetAmount: number | null;
  bettingHistory: any[];
  fetchCurrentBettingRound: () => Promise<void>;
  fetchBettingHistory: () => Promise<void>;
  placeBeastBet: (team: string, amount: number) => Promise<{ success: boolean; message: string } | null>;
}

export const createBettingSlice: StateCreator<any, any, any, BettingSlice> = (set, get) => ({
  currentBettingRound: null,
  myBetTeam: null,
  myBetAmount: null,
  bettingHistory: [],

  fetchCurrentBettingRound: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/current`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({
          currentBettingRound: data.round,
          myBetTeam: data.my_bet_team || null,
          myBetAmount: data.my_bet_amount || null,
        });
      }
    } catch (err: any) {
      console.error("Fetch betting round error:", err.message);
    }
  },

  fetchBettingHistory: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/history`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ bettingHistory: data.history || [] });
      }
    } catch (err: any) {
      console.error("Fetch betting history error:", err.message);
    }
  },

  placeBeastBet: async (team, amount) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/bet`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ team, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Đặt cược thất bại."); }

      // Update player stones locally
      if (player.primeval_stones !== undefined) {
        set({
          player: {
            ...player,
            primeval_stones: player.primeval_stones - amount
          }
        });
      }

      get().addLog(data.message || "Đặt cược thành công!");
      get().addToast(data.message || "Đặt cược thành công!", "success");

      // Refresh current betting round
      await get().fetchCurrentBettingRound();

      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đặt cược lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },
});
