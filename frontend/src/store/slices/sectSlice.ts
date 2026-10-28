import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { Sect, MySectStatus, SectMission, SectMemberInfo, SectJoinRequest, SectPromotionRequest, SectRelocationEstimate } from "@/store/useGameStore";

let fetchMySectStatusPromise: Promise<void> | null = null;
let lastMySectFetchPlayerId: string | null = null;
let lastMySectFetchTime = 0;
const MY_SECT_TTL_MS = 20000;

export interface SectSlice {
  sects: Sect[];
  mySectStatus: MySectStatus | null;
  sectMissions: SectMission[];
  sectMembers: SectMemberInfo[];
  sectJoinRequests: SectJoinRequest[];
  sectPromotionRequests: SectPromotionRequest[];
  fetchSects: () => Promise<void>;
  joinSect: (sectId: string) => Promise<boolean>;
  cancelSectJoinRequest: () => Promise<boolean>;
  leaveSect: () => Promise<boolean>;
  fetchMySectStatus: (forceRefresh?: boolean) => Promise<void>;
  promoteSectRank: () => Promise<boolean>;
  donateSectStones: (amount: number) => Promise<boolean>;
  donateSectMaterials: (materialId: string, amount: number) => Promise<boolean>;
  donateSectGu: (guInstanceId: string) => Promise<boolean>;
  donateSectRecipe: (recipeId: string) => Promise<boolean>;
  donateSectBulk: (payload: {
    materials: { material_id: string; amount: number }[];
    gus: { gu_instance_id: string }[];
    recipes: { recipe_id: string }[];
  }) => Promise<boolean>;
  buyFromSectVault: (itemType: string, itemId: string, quantity?: number, rank?: number) => Promise<boolean>;
  fetchSectMissions: () => Promise<void>;
  acceptSectMission: (missionId: string) => Promise<boolean>;
  completeSectMission: (missionId: string) => Promise<boolean>;
  deleteSectMission: (missionId: string) => Promise<boolean>;
  abandonSectMission: (missionId: string) => Promise<boolean>;
  createSect: (name: string, description: string, region: string) => Promise<boolean>;
  fetchSectMembers: () => Promise<void>;
  setMemberRank: (memberId: string, rank: string) => Promise<boolean>;
  kickMember: (memberId: string) => Promise<boolean>;
  createCustomMission: (payload: {
    title: string;
    description: string;
    mission_type: string;
    target_id: string;
    target_amount: number;
    reward_contribution: number;
    required_member_rank: string;
    duration_hours?: number;
  }) => Promise<boolean>;
  stockSectVault: (itemType: 'gu' | 'material' | 'recipe', itemId: string, quantity: number) => Promise<boolean>;
  stockSectVaultBulk: (items: { item_type: 'gu' | 'material' | 'recipe'; item_id: string; quantity: number }[]) => Promise<boolean>;
  destroySectVaultItem: (itemType: 'gu' | 'material' | 'recipe', itemId: string, rank?: number) => Promise<boolean>;
  
  toggleSectApproval: (requireApproval: boolean) => Promise<boolean>;
  toggleSectPromotionApproval: (requirePromotionApproval: boolean) => Promise<boolean>;
  paySectSalary: () => Promise<boolean>;
  fetchJoinRequests: () => Promise<void>;
  actionJoinRequest: (requestId: string, action: 'approve' | 'reject') => Promise<boolean>;
  fetchPromotionRequests: () => Promise<void>;
  actionPromotionRequest: (requestId: string, action: "approve" | "reject") => Promise<boolean>;
  sectMap: ApiResponseData[];
  fetchSectsMap: () => Promise<void>;
  attackSect: (targetSectId: string, attackerId?: string) => Promise<boolean>;
  defendSect: (defenderType: 'member' | 'beast', playerId?: string, monsterId?: string, monsterRank?: number) => Promise<boolean>;
  undefendSect: (defenderId: string) => Promise<boolean>;
  toggleDefendSect: () => Promise<boolean>;
  upgradeSectBuilding: (buildingName: string) => Promise<boolean>;
  getRelocationEstimate: () => Promise<SectRelocationEstimate | null>;
  relocateSect: () => Promise<boolean>;
  claimSectLeadership: () => Promise<boolean>;
}

export const createSectSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, SectSlice> = (set, get) => ({
  sects: [],
  mySectStatus: null,
  sectMissions: [],
  sectMembers: [],
  sectJoinRequests: [],
  sectPromotionRequests: [],
  sectMap: [],

  fetchSects: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sects: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sects:", err);
      get().addToast("Lỗi tải danh sách tông môn.", "error");
    }
  },

  joinSect: async (sectId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/${sectId}/join`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gia nhập tông môn thất bại.");
      get().addToast(data.message || "Gia nhập tông môn thành công!", "success");
      get().addLog(data.message || `Gia nhập tông môn: ${sectId}`);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  leaveSect: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/leave`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tông môn thất bại.");
      get().addToast(data.message || "Đã rời tông môn!", "info");
      get().addLog(data.message || "Rời tông môn.");
      set({ mySectStatus: null });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  cancelSectJoinRequest: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-request/cancel`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đơn xin gia nhập thất bại.");
      get().addToast(data.message || "Đã hủy đơn xin gia nhập tông môn!", "info");
      get().addLog(data.message || "Hủy đơn xin gia nhập tông môn.");
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchMySectStatus: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    if (
      !forceRefresh &&
      lastMySectFetchPlayerId === currentPlayerId &&
      now - lastMySectFetchTime < MY_SECT_TTL_MS &&
      get().mySectStatus !== null
    ) {
      return;
    }
    if (fetchMySectStatusPromise) return fetchMySectStatusPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/sects/me`);
        if (response.status === 401) { logout(); return; }
        if (response.ok) {
          const data = await response.json();
          lastMySectFetchPlayerId = currentPlayerId;
          lastMySectFetchTime = Date.now();
          set({ mySectStatus: data });
        }
      } catch (err) {
        console.error("Failed to fetch my sect status:", err);
        get().addToast("Lỗi tải thông tin tông môn.", "error");
      } finally {
        fetchMySectStatusPromise = null;
      }
    };
    fetchMySectStatusPromise = executeFetch();
    return fetchMySectStatusPromise;
  },

  promoteSectRank: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promote`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khảo hạch thăng cấp thất bại.");
      get().addToast(data.message || "Thăng cấp thành công!", "success");
      get().addLog(data.message || "Thăng cấp tông môn.");
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  donateSectStones: async (amount: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/stones`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp Nguyên Thạch thành công!", "success");
      get().addLog(data.message || `Quyên góp ${amount} Nguyên Thạch.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  donateSectMaterials: async (materialId: string, amount: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/materials`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ material_id: materialId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp nguyên liệu thành công!", "success");
      get().addLog(data.message || `Quyên góp ${amount}x ${materialId}.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  donateSectGu: async (guInstanceId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: guInstanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp Cổ trùng thành công!", "success");
      get().addLog(data.message || `Quyên góp Cổ trùng.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  donateSectRecipe: async (recipeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/recipe`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ recipe_id: recipeId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) {
        // A recipe selection can be stale (donated twice, lost to a death
        // penalty, or removed in another tab). Refresh the server list so the
        // next attempt uses real inventory instead of repeating this error.
        void get().refreshRecipeProgress();
        throw new Error(data.error || "Quyên góp Bí phương thất bại.");
      }
      get().addToast(data.message || "Quyên góp Bí phương thành công!", "success");
      get().addLog(data.message || `Quyên góp Bí phương.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      void get().refreshRecipeProgress();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  donateSectBulk: async (payload: {
    materials: { material_id: string; amount: number }[];
    gus: { gu_instance_id: string }[];
    recipes: { recipe_id: string }[];
  }) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) {
        // Stale recipe rows are reported per-item by the backend; refresh the
        // authoritative list before surfacing the error so a retry succeeds.
        void get().refreshRecipeProgress();
        throw new Error(data.error || "Quyên góp hàng loạt thất bại.");
      }
      get().addToast(data.message || "Quyên góp hàng loạt thành công!", "success");
      if (Array.isArray(data.skipped_recipes) && data.skipped_recipes.length > 0) {
        get().addToast(`${data.skipped_recipes.length} bí phương đã chọn không còn trong túi đồ nên được bỏ qua.`, "info");
      }
      get().addLog(data.message || `Quyên góp hàng loạt vật phẩm.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      void get().refreshRecipeProgress();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  buyFromSectVault: async (itemType: string, itemId: string, quantity = 1, rank?: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_type: itemType, item_id: itemId, quantity, rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) {
        if (itemType === "recipe") void get().refreshRecipeProgress();
        throw new Error(data.error || "Trao đổi vật phẩm thất bại.");
      }
      get().addToast(data.message || "Trao đổi vật phẩm thành công!", "success");
      get().addLog(data.message || `Đổi ${itemType}: ${itemId} x${quantity} từ bảo khố.`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      if (itemType === "recipe") void get().refreshRecipeProgress();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchSectMissions: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMissions: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect missions:", err);
      get().addToast("Lỗi tải nhiệm vụ tông môn.", "error");
    }
  },

  acceptSectMission: async (missionId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/accept`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận nhiệm vụ thất bại.");
      get().addToast(data.message || "Nhận nhiệm vụ thành công!", "success");
      get().addLog(data.message || `Nhận nhiệm vụ tông môn: ${missionId}`);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  completeSectMission: async (missionId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/complete`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hoàn thành nhiệm vụ thất bại.");
      get().addToast(data.message || "Hoàn thành nhiệm vụ thành công!", "success");
      get().addLog(data.message || `Hoàn thành nhiệm vụ tông môn: ${missionId}`);
      get().applyInventoryFromResponse(data);
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  deleteSectMission: async (missionId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xóa nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã xóa nhiệm vụ tông môn!", "success");
      get().addLog(data.message || `Xóa nhiệm vụ tông môn: ${missionId}`);
      await get().fetchSectMissions();
      await get().fetchMySectStatus(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  abandonSectMission: async (missionId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/abandon`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã hủy nhiệm vụ tông môn!", "success");
      get().addLog(data.message || `Hủy nhiệm vụ tông môn: ${missionId}`);
      await get().fetchMySectStatus(true);
      await get().fetchSectMissions();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  createSect: async (name: string, description: string, region: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, description, region }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lập phái thất bại.");
      get().addToast(data.message || "Khai tông lập phái thành công!", "success");
      get().addLog(data.message || `Lập phái: ${name}`);
      await get().fetchMySectStatus(true);
      await get().fetchSects();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchSectMembers: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/me/members`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMembers: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect members:", err);
      get().addToast("Lỗi tải thành viên tông môn.", "error");
    }
  },

  setMemberRank: async (memberId: string, rank: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/members/${memberId}/rank`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Điều chỉnh chức vụ thất bại.");
      get().addToast(data.message || "Đã điều chỉnh chức vụ!", "success");
      get().addLog(data.message || `Điều chỉnh chức vụ thành viên: ${memberId} sang ${rank}`);
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  kickMember: async (memberId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/members/${memberId}/kick`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Trục xuất đệ tử thất bại.");
      get().addToast(data.message || "Đã trục xuất đệ tử!", "info");
      get().addLog(data.message || `Trục xuất đệ tử: ${memberId}`);
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  createCustomMission: async (payload) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: payload.title,
          description: payload.description,
          mission_type: payload.mission_type,
          target_id: payload.target_id,
          target_amount: payload.target_amount,
          reward_contribution: payload.reward_contribution,
          required_member_rank: payload.required_member_rank,
          duration_hours: payload.duration_hours,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Phát hành nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã phát hành nhiệm vụ môn phái!", "success");
      get().addLog(data.message || `Phát hành nhiệm vụ: ${payload.title}`);
      await get().fetchSectMissions();
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  stockSectVault: async (itemType, itemId, quantity) => {
    const { token, logout, itemDictionary } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    const item = itemDictionary?.[itemId];
    const itType = (item?.item_type || item?.type || "").toLowerCase();
    const argType = (itemType || "").toLowerCase();
    if (itType === "consumable" || itType === "currency" || argType === "consumable" || argType === "currency") {
      get().addToast("Không thể nhập vật phẩm loại Tiêu hao hoặc Tiền tệ vào bảo khố!", "error");
      get().endAction("sect");
      return false;
    }
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/stock`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          item_type: itemType,
          item_id: itemId,
          quantity: quantity,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhập kho bảo khố thất bại.");
      get().addToast(data.message || "Đã nhập kho bảo khố thành công!", "success");
      get().addLog(`Nhập bảo khố: ${quantity}x ${itemId}`);
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  stockSectVaultBulk: async (items) => {
    const { token, logout, itemDictionary } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    const invalidItem = items.find(it => {
      const item = itemDictionary?.[it.item_id];
      const itType = (item?.item_type || item?.type || "").toLowerCase();
      const argType = (it.item_type || "").toLowerCase();
      return itType === "consumable" || itType === "currency" || argType === "consumable" || argType === "currency";
    });
    if (invalidItem) {
      get().addToast("Không thể nhập vật phẩm loại Tiêu hao hoặc Tiền tệ vào bảo khố!", "error");
      get().endAction("sect");
      return false;
    }
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/stock/bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhập kho bảo khố thất bại.");
      get().addToast(data.message || "Đã nhập kho bảo khố thành công!", "success");
      get().addLog(`Nhập bảo khố hàng loạt: ${items.length} mặt hàng`);
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  destroySectVaultItem: async (itemType, itemId, rank) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/destroy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_type: itemType, item_id: itemId, rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tiêu huỷ vật phẩm thất bại.");
      get().addToast(data.message || "Đã tiêu huỷ vật phẩm!", "success");
      get().addLog(`Tiêu huỷ khỏi bảo khố: ${itemId}${rank ? ` (rank ${rank})` : ""}`);
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  

  toggleSectApproval: async (requireApproval) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/toggle-approval`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ require_approval: requireApproval }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cài đặt thất bại.");
      get().addToast(data.message || "Đã cập nhật cài đặt duyệt thành viên!", "success");
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  toggleSectPromotionApproval: async (requirePromotionApproval) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/toggle-promotion-approval`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ require_promotion_approval: requirePromotionApproval }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cài đặt thất bại.");
      get().addToast(data.message || "Đã cập nhật cài đặt duyệt lên cấp!", "success");
      await get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  paySectSalary: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/pay-salary`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ confirm: true }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Phát lương thất bại.");
      get().addToast(data.message || "Đã phát lương thành công!", "success");
      await get().fetchMySectStatus();
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchJoinRequests: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-requests`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectJoinRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch join requests:", err);
      get().addToast("Lỗi tải yêu cầu gia nhập.", "error");
    }
  },

  actionJoinRequest: async (requestId, action) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-requests/${requestId}/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      get().addToast(data.message || "Thao tác thành công!", "success");
      await get().fetchJoinRequests();
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchPromotionRequests: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promotion-requests`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectPromotionRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch promotion requests:", err);
      get().addToast("Lỗi tải yêu cầu thăng chức.", "error");
    }
  },

  actionPromotionRequest: async (requestId, action) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("sect")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promotion-requests/${requestId}/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      get().addToast(data.message || "Thao tác thành công!", "success");
      await get().fetchPromotionRequests();
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("sect"); }
  },

  fetchSectsMap: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/map`, { method: "GET" });
      if (!response.ok) throw new Error("Failed to fetch sects map");
      const data = await response.json();
      set({ sectMap: data || [] });
    } catch (err) {
      console.error("Failed to fetch sect map", err);
    }
  },

  attackSect: async (targetSectId: string, attackerId?: string) => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_sect_id: targetSectId, attacker_id: attackerId || "" }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (response.status === 409 && data.formation_cancelled) {
        addToast(data.error || "Hộ Sơn Đại Trận đã chặn cuộc tấn công.", "info");
        addLog(data.error || "Hộ Sơn Đại Trận đã chặn cuộc tấn công.");
        get().fetchSectsMap();
        return false;
      }
      if (!response.ok) throw new Error(data.error || "Lỗi khi tấn công Tông môn");
      addToast(data.message || "Đã tấn công Tông môn!", data.won ? "success" : "info");
      addLog(data.message || "Tấn công Tông môn.");
      get().fetchPlayerProfile(get().player?.username || "");
      get().fetchSectsMap();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi khi tấn công", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  toggleDefendSect: async () => {
    const { token, logout, addToast, addLog, mySectStatus } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/toggle-defend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật trạng thái thất bại");
      addToast(data.message || "Đã cập nhật trạng thái Trấn Thủ!", "success");
      addLog(data.message || "Cập nhật Trấn Thủ Tông Môn.");
      get().fetchMySectStatus();
      get().fetchSectsMap();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi cập nhật Trấn Thủ", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  defendSect: async (defenderType: 'member' | 'beast', playerId?: string, monsterId?: string, monsterRank?: number) => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/defend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          defender_type: defenderType,
          player_id: playerId || "",
          monster_id: monsterId || "",
          monster_rank: monsterRank || 1,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bố trí phòng thủ thất bại.");
      addToast(data.message || "Đã bố trí phòng thủ!", "success");
      addLog(data.message || "Bố trí phòng thủ tông môn.");
      get().fetchSectsMap();
      get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi bố trí phòng thủ", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  undefendSect: async (defenderId: string) => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/undefend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ defender_id: defenderId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút phòng thủ thất bại.");
      addToast(data.message || "Đã rút phòng thủ!", "success");
      addLog(data.message || "Rút phòng thủ tông môn.");
      get().fetchSectsMap();
      get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi rút phòng thủ", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  upgradeSectBuilding: async (buildingName: string) => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/building/upgrade`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ building_name: buildingName }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nâng cấp kiến trúc thất bại.");
      addToast(data.message || "Đã nâng cấp kiến trúc!", "success");
      addLog(data.message || "Nâng cấp kiến trúc tông môn.");
      get().fetchMySectStatus();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi nâng cấp", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  getRelocationEstimate: async () => {
    const { token, logout, addToast } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/relocate/estimate`);
      if (response.status === 401) { logout(); return null; }
      const data = await response.json();
      if (!response.ok) {
        addToast(data.error || "Không thể tải dự toán di dời", "error");
        return null;
      }
      return data as SectRelocationEstimate;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi tải thông tin di dời", "error");
      return null;
    }
  },

  relocateSect: async () => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/relocate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Di dời sơn môn thất bại.");
      addToast(data.message || "Di dời sơn môn thành công!", "success");
      addLog(data.message || "Di dời sơn môn thành công!");
      await get().fetchMySectStatus(true);
      await get().fetchSectsMap();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi di dời sơn môn", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },

  claimSectLeadership: async () => {
    const { token, logout, addToast, addLog } = get();
    if (!token || !get().beginAction("sect")) return false;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/claim-leadership`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tiếp quản Chưởng Môn thất bại.");
      addToast(data.message || "Tiếp quản chức vụ Chưởng Môn thành công!", "success");
      addLog(data.message || "Tiếp quản chức vụ Chưởng Môn.");
      await get().fetchMySectStatus(true);
      await get().fetchSectMembers();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      addToast(err.message || "Lỗi tiếp quản Chưởng Môn", "error");
      return false;
    } finally {
      get().endAction("sect");
    }
  },
});
