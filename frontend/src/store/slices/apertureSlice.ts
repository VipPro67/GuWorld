import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { createManaSnapshotFromPlayer } from "@/store/useGameStore";
import type {
  ImmortalAperture,
  DaoMark,
  DaoNganResult,
  ApertureResourceNode,
  ApertureVariantHuman,
  ApertureNodeWorker,
  ApertureRefinementJob,
  ApertureSettlement,
  ApertureVariantNPC,
  TribulationEvent,
  TribulationEnemy,
} from "@/store/useGameStore";

let fetchAperturePromise: Promise<void> | null = null;
let lastAperturePlayerId: string | null = null;
let lastApertureFetchTime = 0;
const APERTURE_TTL_MS = 20000;

let fetchApertureNodesPromise: Promise<void> | null = null;
let lastApertureNodesPlayerId: string | null = null;
let lastApertureNodesFetchTime = 0;
const APERTURE_NODES_TTL_MS = 10000;

let fetchVariantHumansPromise: Promise<void> | null = null;

let fetchDaoMarksPromise: Promise<void> | null = null;
let lastDaoMarksPlayerId: string | null = null;
let lastDaoMarksFetchTime = 0;
const DAO_MARKS_TTL_MS = 30000;

let fetchDaoNganPromise: Promise<void> | null = null;
let lastDaoNganPlayerId: string | null = null;
let lastDaoNganFetchTime = 0;

let fetchActiveTribulationPromise: Promise<void> | null = null;
let lastActiveTribulationPlayerId: string | null = null;
let lastActiveTribulationFetchTime = 0;
const ACTIVE_TRIBULATION_TTL_MS = 15000;

export interface ApertureSlice {
  aperture: ImmortalAperture | null;
  daoMarks: DaoMark[];
  daoNgan: DaoNganResult | null;
  apertureNodes: ApertureResourceNode[];
  variantHumans: ApertureVariantHuman[];
  activeTribulation: { event: TribulationEvent; enemies: TribulationEnemy[] } | null;
  fetchAperture: (forceRefresh?: boolean) => Promise<void>;
  fetchDaoMarks: (forceRefresh?: boolean) => Promise<void>;
  fetchDaoNgan: (forceRefresh?: boolean) => Promise<void>;
  cleanseDaoMarks: (path: string, marks: number) => Promise<boolean>;
  fetchApertureNodes: (forceRefresh?: boolean) => Promise<void>;
  fetchVariantHumans: () => Promise<void>;
  placeApertureNode: (nodeType: string, x: number, y: number, path?: string, prodKey?: string) => Promise<boolean>;
  collectResourceNode: (nodeId: string) => Promise<boolean>;
  collectAllResourceNodes: () => Promise<boolean>;
  uninstallApertureNode: (nodeId: string) => Promise<boolean>;
  deleteApertureNode: (nodeId: string) => Promise<boolean>;
  setApertureTimeRate: (rate: number) => Promise<boolean>;
  setApertureLabor: (race: string, intensity: string) => Promise<boolean>;
  repairApertureNode: (nodeId: string) => Promise<boolean>;
  repairAllApertureNodes: () => Promise<boolean>;
  upgradeApertureNode: (nodeId: string) => Promise<boolean>;
  fetchActiveTribulation: (forceRefresh?: boolean) => Promise<void>;
  engageTribulationCombat: (eventId: string) => Promise<boolean>;
  resolveTribulation: (eventId: string) => Promise<boolean>;
  nodeWorkers: ApertureNodeWorker[];
  refinementQueue: ApertureRefinementJob[];
  settlements: ApertureSettlement[];
  variantNPCs: ApertureVariantNPC[];
  fetchNodeWorkers: (nodeId: string) => Promise<void>;
  assignWorkerToNode: (nodeId: string, variantHumanId: string, population: number) => Promise<boolean>;
  removeWorkerFromNode: (nodeId: string, workerId: string) => Promise<boolean>;
  fetchRefinementQueue: () => Promise<void>;
  createRefinementJob: (recipeId: string, mode: string, targetQuantity: number) => Promise<boolean>;
  deleteRefinementJob: (jobId: string) => Promise<boolean>;
  fetchSettlements: () => Promise<void>;
  buildSettlement: (race: string, name?: string, settlementType?: string) => Promise<boolean>;
  upgradeSettlement: (settlementId: string, data: { name?: string; settlement_type?: string; population_capacity?: number; housing?: number; food_storage?: number }) => Promise<boolean>;
  fetchNPCs: () => Promise<void>;
  assignNPCRole: (npcId: string, role: string) => Promise<boolean>;
  setNPCFormation: (npcId: string, slot: number) => Promise<boolean>;
  cultivateNPC: (npcId: string) => Promise<boolean>;
  equipNPCGu: (npcId: string, guInstanceId: string, slot: number) => Promise<boolean>;
  unequipNPCGu: (npcId: string, guInstanceId: string) => Promise<boolean>;
  dismissNPC: (npcId: string) => Promise<boolean>;
  useGutsGu: () => Promise<boolean>;
  refineSoul: () => Promise<boolean>;
}

export const createApertureSlice: StateCreator<ApiResponseData, ApiResponseData, ApiResponseData, ApertureSlice> = (set, get) => ({
  aperture: null,
  daoMarks: [],
  daoNgan: null,
  apertureNodes: [],
  variantHumans: [],
  nodeWorkers: [],
  refinementQueue: [],
  settlements: [],
  variantNPCs: [],
  activeTribulation: null,

  fetchAperture: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    if (
      !forceRefresh &&
      lastAperturePlayerId === currentPlayerId &&
      now - lastApertureFetchTime < APERTURE_TTL_MS &&
      get().aperture !== null
    ) {
      return;
    }
    if (fetchAperturePromise) return fetchAperturePromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (!response.ok) {
          set({ aperture: null });
          return;
        }
        const data = await response.json();
        lastAperturePlayerId = currentPlayerId;
        lastApertureFetchTime = Date.now();
        set({ aperture: data });
        const player = get().player;
        if (player) {
          set({ manaSnapshot: createManaSnapshotFromPlayer(player, get().guDictionary, get().constitutionDictionary, data.time_rate) });
        }
      } catch (err) {
        console.error("Failed to fetch aperture:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
        set({ aperture: null });
      } finally {
        fetchAperturePromise = null;
      }
    };
    fetchAperturePromise = executeFetch();
    return fetchAperturePromise;
  },

  fetchDaoMarks: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    // Fire fetchDaoNgan in parallel
    get().fetchDaoNgan(forceRefresh).catch(() => {});
    if (
      !forceRefresh &&
      lastDaoMarksPlayerId === currentPlayerId &&
      now - lastDaoMarksFetchTime < DAO_MARKS_TTL_MS &&
      get().daoMarks.length > 0
    ) {
      return;
    }
    if (fetchDaoMarksPromise) return fetchDaoMarksPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/dao-marks`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          lastDaoMarksPlayerId = currentPlayerId;
          lastDaoMarksFetchTime = Date.now();
          set({ daoMarks: Array.isArray(data) ? data : [] });
        }
      } catch (err) {
        console.error("Failed to fetch Dao Marks:", err);
        get().addToast("Lỗi tải dấu đạo.", "error");
      } finally {
        fetchDaoMarksPromise = null;
      }
    };
    fetchDaoMarksPromise = executeFetch();
    return fetchDaoMarksPromise;
  },

  fetchDaoNgan: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    if (
      !forceRefresh &&
      lastDaoNganPlayerId === currentPlayerId &&
      now - lastDaoNganFetchTime < DAO_MARKS_TTL_MS &&
      get().daoNgan !== null
    ) {
      return;
    }
    if (fetchDaoNganPromise) return fetchDaoNganPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/dao-marks/dao-ngan`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          lastDaoNganPlayerId = currentPlayerId;
          lastDaoNganFetchTime = Date.now();
          set({ daoNgan: data });
        }
      } catch (err) {
        console.error("Failed to fetch Dao Ngan:", err);
      } finally {
        fetchDaoNganPromise = null;
      }
    };
    fetchDaoNganPromise = executeFetch();
    return fetchDaoNganPromise;
  },

  cleanseDaoMarks: async (path: string, marks: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/cleanse-dao-marks`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path, marks }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thanh Đạo Tẩy Thể thất bại");
      get().addToast(data.message || "Thanh Đạo Tẩy Thể thành công!", "success");
      get().addLog(data.message || "Thanh Đạo Tẩy Thể thành công.");
      get().applyInventoryFromResponse(data);
      await get().fetchDaoMarks(true);
      await get().fetchDaoNgan(true);
      const pid = get().player?.id;
      if (pid) {
        await get().fetchPlayerStatus(pid, true);
      }
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  fetchApertureNodes: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    if (
      !forceRefresh &&
      lastApertureNodesPlayerId === currentPlayerId &&
      now - lastApertureNodesFetchTime < APERTURE_NODES_TTL_MS &&
      get().apertureNodes.length > 0
    ) {
      return;
    }
    if (fetchApertureNodesPromise) return fetchApertureNodesPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          lastApertureNodesPlayerId = currentPlayerId;
          lastApertureNodesFetchTime = Date.now();
          set({ apertureNodes: Array.isArray(data) ? data : [] });
        }
      } catch (err) {
        console.error("Failed to fetch aperture nodes:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
      } finally {
        fetchApertureNodesPromise = null;
      }
    };
    fetchApertureNodesPromise = executeFetch();
    return fetchApertureNodesPromise;
  },

  placeApertureNode: async (nodeType: string, x: number, y: number, path?: string, prodKey?: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_type: nodeType, x, y, path, prod_key: prodKey }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể xây dựng điểm tài nguyên");
      }
      get().addToast(data.message || "Xây dựng mỏ tài nguyên thành công!", "success");
      get().addLog(data.message || "Đã đặt điểm tài nguyên mới.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      await get().fetchGuInventory();
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  collectResourceNode: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/collect`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể thu hoạch tài nguyên");
      }
      get().addToast(data.message || "Thu hoạch thành công!", "success");
      get().addLog(data.message || "Thu hoạch tài nguyên thành công.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  collectAllResourceNodes: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/collect-all`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể thu hoạch tài nguyên");
      }
      get().addToast(data.message || "Thu hoạch hàng loạt thành công!", "success");
      get().addLog(data.message || "Thu hoạch hàng loạt tài nguyên thành công.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  uninstallApertureNode: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/uninstall`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể tháo dỡ mỏ tài nguyên");
      }
      get().addToast(data.message || "Tháo dỡ thành công!", "success");
      get().addLog(data.message || "Đã tháo dỡ mỏ tài nguyên.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  deleteApertureNode: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}`, {
        method: "DELETE",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể san phẳng mỏ tài nguyên");
      }
      get().addToast(data.message || "San phẳng thành công!", "success");
      get().addLog(data.message || "Đã san phẳng mỏ tài nguyên.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  fetchVariantHumans: async () => {
    if (fetchVariantHumansPromise) return fetchVariantHumansPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/variant-humans`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          set({ variantHumans: data });
        }
      } catch (err) {
        console.error("Failed to fetch variant humans:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
      } finally {
        fetchVariantHumansPromise = null;
      }
    };
    fetchVariantHumansPromise = executeFetch();
    return fetchVariantHumansPromise;
  },

  // --- Node Workers ---
  fetchNodeWorkers: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/workers`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ nodeWorkers: data.workers || [] });
      }
    } catch (err) {
      console.error("Failed to fetch node workers:", err);
    }
  },

  assignWorkerToNode: async (nodeId: string, variantHumanId: string, population: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/workers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ variant_human_id: variantHumanId, assigned_population: population }),
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Phân công lao động thành công", "success");
        get().fetchNodeWorkers(nodeId);
        return true;
      }
      const data = await response.json();
      get().addToast(data.error || "Lỗi phân công", "error");
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  removeWorkerFromNode: async (nodeId: string, workerId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/workers/${workerId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Đã xóa phân công", "success");
        get().fetchNodeWorkers(nodeId);
        return true;
      }
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  // --- Refinement Queue ---
  fetchRefinementQueue: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/refinement-queue`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ refinementQueue: data.queue || [] });
      }
    } catch (err) {
      console.error("Failed to fetch refinement queue:", err);
    }
  },

  createRefinementJob: async (recipeId: string, mode: string, targetQuantity: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/refinement-queue`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ recipe_id: recipeId, mode, target_quantity: targetQuantity }),
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Tạo lệnh luyện cổ thành công", "success");
        get().fetchRefinementQueue();
        return true;
      }
      const data = await response.json();
      get().addToast(data.error || "Lỗi tạo lệnh", "error");
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  deleteRefinementJob: async (jobId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/refinement-queue/${jobId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Đã xóa lệnh luyện cổ", "success");
        get().fetchRefinementQueue();
        return true;
      }
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  // --- Settlements ---
  fetchSettlements: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/settlements`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ settlements: data.settlements || [] });
      }
    } catch (err) {
      console.error("Failed to fetch settlements:", err);
    }
  },

  buildSettlement: async (race: string, name?: string, settlementType?: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/settlements`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ race, name, settlement_type: settlementType }),
      });
      if (response.status === 401) { logout(); return false; }
      const res = await response.json();
      if (response.ok) {
        get().addToast(res.message || "Xây dựng cơ sở định cư thành công!", "success");
        get().fetchSettlements();
        get().fetchVariantHumans();
        return true;
      }
      get().addToast(res.error || "Lỗi tạo thành trì", "error");
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  upgradeSettlement: async (settlementId: string, data: { name?: string; population_capacity?: number; housing?: number; food_storage?: number }) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/settlements/${settlementId}/upgrade`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Cập nhật settlement thành công", "success");
        get().fetchSettlements();
        return true;
      }
      const res = await response.json();
      get().addToast(res.error || "Lỗi cập nhật", "error");
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  // --- Variant NPCs ---
  fetchNPCs: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ variantNPCs: data.npcs || [] });
      }
    } catch (err) {
      console.error("Failed to fetch NPCs:", err);
    }
  },

  assignNPCRole: async (npcId: string, role: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}/role`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role }),
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Phân công vai trò thành công", "success");
        get().fetchNPCs();
        return true;
      }
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  setNPCFormation: async (npcId, slot) => {
    const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}/formation`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ slot }) });
    if (response.ok) { get().addToast("Đã cập nhật đội hình NPC", "success"); await get().fetchNPCs(); return true; }
    get().addToast((await response.json().catch(() => ({}))).error || "Không thể cập nhật đội hình", "error"); return false;
  },

  cultivateNPC: async (npcId) => {
    const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}/cultivate`, { method: "POST" });
    const data = await response.json().catch(() => ({}));
    if (response.ok) { get().addToast(`Tu luyện +${data.gained} tiến độ`, "success"); await Promise.all([get().fetchNPCs(), get().initializeSession()]); return true; }
    get().addToast(data.error || "Không thể tu luyện", "error"); return false;
  },

  equipNPCGu: async (npcId, guInstanceId, slot) => {
    const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}/gu/equip`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ gu_instance_id: guInstanceId, slot }) });
    const data = await response.json().catch(() => ({}));
    if (response.ok) { get().addToast("Đã giao Cổ cho NPC", "success"); await Promise.all([get().fetchNPCs(), get().initializeSession()]); return true; }
    get().addToast(data.error || "Không thể trang bị Cổ", "error"); return false;
  },

  unequipNPCGu: async (npcId, guInstanceId) => {
    const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}/gu/${guInstanceId}`, { method: "DELETE" });
    if (response.ok) { get().addToast("Đã thu hồi Cổ", "success"); await Promise.all([get().fetchNPCs(), get().initializeSession()]); return true; }
    return false;
  },

  dismissNPC: async (npcId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/npcs/${npcId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); return false; }
      if (response.ok) {
        get().addToast("Đã sa thải NPC", "success");
        get().fetchNPCs();
        return true;
      }
      return false;
    } catch {
      get().addToast("Lỗi hệ thống", "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  setApertureTimeRate: async (rate: number) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/time-rate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ time_rate: rate }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể điều chỉnh dòng thời gian");
      }
      get().addToast(data.message || "Điều chỉnh dòng thời gian thành công!", "success");
      get().addLog(data.message || "Đã điều chỉnh dòng thời gian Tiên Khiếu.");
      await get().fetchAperture();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  setApertureLabor: async (race: string, intensity: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/labor`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ race, labor_intensity: intensity }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể điều chỉnh cường độ lao động");
      }
      get().addToast(data.message || "Điều chỉnh cường độ lao động thành công!", "success");
      get().addLog(data.message || "Đã điều chỉnh cường độ lao động của dị nhân.");
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  repairApertureNode: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/repair`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể sửa chữa điểm tài nguyên");
      }
      get().addToast(data.message || "Sửa chữa thành công!", "success");
      get().addLog(data.message || "Đã sửa chữa điểm tài nguyên.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  repairAllApertureNodes: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/repair-all`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể sửa chữa công trình");
      }
      get().addToast(data.message || "Sửa chữa hàng loạt thành công!", "success");
      get().addLog(data.message || "Đã sửa chữa hàng loạt công trình.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      await get().fetchVariantHumans();
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  upgradeApertureNode: async (nodeId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/upgrade`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể nâng cấp công trình");
      }
      get().addToast(data.message || "Nâng cấp thành công!", "success");
      get().addLog(data.message || "Đã nâng cấp công trình.");
      get().applyInventoryFromResponse(data);
      await get().fetchApertureNodes(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  fetchActiveTribulation: async (forceRefresh = false) => {
    const currentPlayerId = get().player?.id || "";
    const now = Date.now();
    if (
      !forceRefresh &&
      lastActiveTribulationPlayerId === currentPlayerId &&
      now - lastActiveTribulationFetchTime < ACTIVE_TRIBULATION_TTL_MS
    ) {
      return;
    }
    if (fetchActiveTribulationPromise) return fetchActiveTribulationPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/tribulations/active`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          lastActiveTribulationPlayerId = currentPlayerId;
          lastActiveTribulationFetchTime = Date.now();
          set({ activeTribulation: data });
        }
      } catch (err) {
        console.error("Failed to fetch active tribulation:", err);
        get().addToast("Lỗi tải thiên kiếp.", "error");
      } finally {
        fetchActiveTribulationPromise = null;
      }
    };
    fetchActiveTribulationPromise = executeFetch();
    return fetchActiveTribulationPromise;
  },

  engageTribulationCombat: async (eventId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/engage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Xuất trận thất bại");
      }
      get().addLog(data.message || "Xuất trận nghênh chiến quái tai kiếp.");
      get().applyInventoryFromResponse(data);
      if (data.active_combat) {
        const currentPlayer = get().player;
        if (currentPlayer) {
          set((state: ApiResponseData) => ({
            player: { ...currentPlayer, active_combat: data.active_combat },
            playerRevision: state.playerRevision + 1,
          }));
        }
      }
      await get().fetchActiveTribulation(true);
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  resolveTribulation: async (eventId: string) => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/resolve`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Kết toán tai kiếp thất bại");
      }
      get().addToast(data.message || "Kết toán tai kiếp hoàn thành!", "success");
      get().addLog(data.message || "Kết toán tai kiếp hoàn thành.");
      get().applyInventoryFromResponse(data);
      await get().fetchAperture(true);
      await get().fetchDaoMarks(true);
      const pid = get().player?.id;
      if (pid) {
        await get().fetchPlayerStatus(pid, true);
      }
      set({ activeTribulation: null });
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      get().endAction("aperture");
    }
  },

  useGutsGu: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/soul/use-guts`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sử dụng Cổ trùng Can Đảm thất bại.");
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Tráng hồn thành công!", "success");
      get().addLog(data.message || "Đã sử dụng 1x Can Đảm Cổ để tráng đại linh hồn.");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("aperture"); }
  },

  refineSoul: async () => {
    const { token, logout } = get();
    if (!token || !get().beginAction("aperture")) return false;
    set({ error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/soul/refine`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Luyện hồn thất bại.");
      get().applyInventoryFromResponse(data);
      get().addToast(data.message || "Luyện hồn thành công!", "success");
      get().addLog(data.message || "Đã luyện hồn tại Lạc Phách Cốc, nén ép linh hồn ngưng thực.");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { get().endAction("aperture"); }
  },
});
