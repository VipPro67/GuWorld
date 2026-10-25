import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface ReferralsSlice {
  referralsData: {
    ref_code: string;
    referees: ApiResponseData[];
    total_ref: number;
    has_referrer?: boolean;
    can_bind_ref?: boolean;
  } | null;
  passiveStats: { stats: Record<string, number>; details: { key: string; total: number; sources: { source: string; name: string; value: number; final: number }[] }[] } | null;
  fetchReferrals: () => Promise<void>;
  claimReferralReward: (refereeId: string, milestone: number) => Promise<{success: boolean; message?: string}>;
  bindReferralCode: (refCode: string) => Promise<{success: boolean; message?: string}>;
  fetchPassiveStats: () => Promise<void>;
}

export const createReferralsSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ReferralsSlice> = (set, get) => ({
  referralsData: null,
  passiveStats: null,

  fetchReferrals: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals`);
      if (response.ok) {
        const data = await response.json();
        set({ referralsData: data });
      }
    } catch (err) {
      console.error("Failed to fetch referrals", err);
    }
  },

  claimReferralReward: async (refereeId: string, milestone: number) => {
    if (!get().beginAction("referrals")) return { success: false };
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals/claim`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ referee_id: refereeId, milestone }),
      });
      const data = await response.json();
      if (response.ok) {
        get().addLog(`[Hệ thống] ${data.message}`);
        // Reload referrals to update status
        await get().fetchReferrals();
        // Also refresh player stats (like primeval stones)
        await get().fetchPlayerProfile(get().player?.id || "");
        return { success: true, message: data.message };
      }
      return { success: false, message: data.error };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Failed to claim referral reward", err);
      return { success: false, message: "Lỗi kết nối máy chủ." };
    } finally {
      get().endAction("referrals");
    }
  },

  bindReferralCode: async (refCode: string) => {
    if (!get().beginAction("referrals")) return { success: false };
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals/bind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ref_code: refCode }),
      });
      const data = await response.json();
      if (response.ok) {
        get().addLog(`[Hệ thống] ${data.message}`);
        await get().fetchReferrals();
        return { success: true, message: data.message };
      }
      return { success: false, message: data.error };
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Failed to bind referral code", err);
      return { success: false, message: "Lỗi kết nối máy chủ." };
    } finally {
      get().endAction("referrals");
    }
  },

  fetchPassiveStats: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/passive-stats`);
      if (response.ok) {
        const data = await response.json();
        set({ passiveStats: data });
      }
    } catch (err) {
      console.error("Failed to fetch passive stats", err);
    }
  },

});
