import React, { useEffect, useState, useCallback } from "react";
import { useGameStore } from "@/store/useGameStore";
import type { GuSpec, ItemSpec } from "@/store/useGameStore";
import {
  Shield,
  Search,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Users,
  ArrowLeft,
  Plus,
  Trash2,
  Package,
  Bug,
  Info,
  X,
  Coins,
  Sparkles,
  Heart,
  Zap,
  Globe,
  Calendar,
  Sword,
  Shield as ShieldIcon,
  Layers,
  Star,
  Filter,
  Building2,
  Crown,
  UserMinus,
  UserPlus,
  Edit3,
  ListTodo,
  BookOpen,
  Database,
} from "lucide-react";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(^| )${name}=([^;]+)`));
  return match ? decodeURIComponent(match[2]) : null;
}

function authFetch(input: RequestInfo | URL, init: RequestInit = {}) {
  const csrfToken = getCookie("gu_csrf_token");
  const headers: Record<string, string> = {};
  if (init.headers) {
    if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) headers[k] = v;
    } else if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => { headers[k] = v; });
    } else {
      Object.assign(headers, init.headers);
    }
  }
  if (csrfToken && init.method && init.method !== "GET" && init.method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }
  return fetch(input, { ...init, headers, credentials: "include" });
}

interface PlayerRow {
  id: string; username: string; email: string; rank: number; sub_rank: number;
  current_mana: number; max_mana: number; hp: number; base_hp: number;
  primeval_stones: number; immortal_stones: number; wisdom: number;
  is_admin: boolean; created_at: string; current_region: string;
  aptitude_type: string; wall_progress: number; human_qi: number;
  lifespan: number; max_lifespan: number;
}

interface PlayerDetail {
  id: string; username: string; email: string; rank: number; sub_rank: number;
  aptitude_type: string; aptitude_percent: number; aptitude_grade: number;
  current_mana: number; max_mana: number; primeval_stones: number;
  immortal_stones: number; wisdom: number; hp: number; base_hp: number;
  wall_progress: number; human_qi: number; current_region: string;
  is_admin: boolean; created_at: string; lifespan: number; max_lifespan: number;
  soul_hp: number; max_soul: number; soul_tier: string;
  explore_count: number; aperture_stress: number;
  avatar_url: string; bio: string;
  gu_inventory: GuInstanceDetail[];
  materials_inventory: Record<string, number>;
}

interface GuInstanceDetail {
  instance_id: string;
  gu_id: string;
  hunger: number;
  status: string;
  equipped: boolean;
  is_vital: boolean;
  integration_time?: number;
}

const rankNames = [
  "Nhất Chuyển", "Nhị Chuyển", "Tam Chuyển", "Tứ Chuyển",
  "Ngũ Chuyển", "Lục Tiên", "Thất Tiên", "Bát Tiên", "Cửu Tôn",
];
const rankSub = ["Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"];

function getRankName(rank: number, sub: number): string {
  const r = rankNames[rank - 1] || `${rank} Chuyển`;
  const s = rankSub[sub - 1] || `${sub} Giai`;
  return `${r} • ${s}`;
}

function formatDate(dateStr: string) {
  if (!dateStr) return "";
  const d = new Date(dateStr);
  return d.toLocaleString("vi-VN", {
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit",
  });
}

function statusColor(status: string): string {
  switch (status) {
    case "ACTIVE": return "text-emerald-400 border-emerald-500/30 bg-emerald-950/60";
    case "SLEEPING": return "text-blue-400 border-blue-500/30 bg-blue-950/60";
    case "DEAD": return "text-red-400 border-red-500/30 bg-red-950/60";
    default: return "text-zinc-400 border-zinc-700 bg-zinc-900";
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "ACTIVE": return "Hoạt động";
    case "SLEEPING": return "Ngủ";
    case "DEAD": return "Chết";
    default: return status;
  }
}

function rankColor(rank: number): string {
  if (rank >= 7) return "text-purple-400";
  if (rank >= 6) return "text-indigo-400";
  if (rank >= 4) return "text-emerald-400";
  if (rank >= 2) return "text-blue-400";
  return "text-zinc-400";
}

function memberRankStyle(rank: string): string {
  switch (rank) {
    case "CHUONG_MON": return "text-amber-400 border-amber-500/30 bg-amber-950/60";
    case "TRUONG_LAO": return "text-purple-400 border-purple-500/30 bg-purple-950/60";
    case "CHAN_TRUYEN": return "text-blue-400 border-blue-500/30 bg-blue-950/60";
    case "TINH_ANH": return "text-emerald-400 border-emerald-500/30 bg-emerald-950/60";
    case "NOI_MON": return "text-cyan-400 border-cyan-500/30 bg-cyan-950/60";
    default: return "text-zinc-400 border-zinc-700 bg-zinc-900";
  }
}

function memberRankLabel(rank: string): string {
  switch (rank) {
    case "CHUONG_MON": return "CHƯỞNG MÔN";
    case "TRUONG_LAO": return "TRƯỞNG LÃO";
    case "CHAN_TRUYEN": return "CHÂN TRUYỀN";
    case "TINH_ANH": return "TINH ANH";
    case "NOI_MON": return "NỘI MÔN";
    default: return "NGOẠI MÔN";
  }
}

export const AdminView: React.FC = () => {
  const { player, setView, addToast, guDictionary, itemDictionary } = useGameStore();
  const [activeTab, setActiveTab] = useState<"items" | "gus">("items");
  const [players, setPlayers] = useState<PlayerRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<PlayerDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const [itemSearch, setItemSearch] = useState("");
  const [guSearch, setGuSearch] = useState("");

  const [showAddItem, setShowAddItem] = useState(false);
  const [addItemId, setAddItemId] = useState("");
  const [addItemQty, setAddItemQty] = useState(1);

  const [showRemoveItem, setShowRemoveItem] = useState<string | null>(null);
  const [removeItemQty, setRemoveItemQty] = useState(1);

  const [showAddGu, setShowAddGu] = useState(false);
  const [addGuId, setAddGuId] = useState("");
  const [addGuQty, setAddGuQty] = useState(1);

  const [showStones, setShowStones] = useState(false);
  const [stonesPrimeval, setStonesPrimeval] = useState("");
  const [stonesImmortal, setStonesImmortal] = useState("");

  const [actionLoading, setActionLoading] = useState(false);
  const [section, setSection] = useState<"players" | "sects" | "data" | "logs" | "shop" | "mines">("players");

  // ----- Audit log state -----
  const [auditLogs, setAuditLogs] = useState<any[]>([]);
  const [auditLogTotal, setAuditLogTotal] = useState(0);
  const [auditLogLoading, setAuditLogLoading] = useState(false);
  const [auditLogPage, setAuditLogPage] = useState(0);
  const [auditLogLimit] = useState(50);
  const [auditLogFilterAction, setAuditLogFilterAction] = useState("");
  const [auditLogFilterPlayer, setAuditLogFilterPlayer] = useState("");
  const [auditLogFilterSuccess, setAuditLogFilterSuccess] = useState("");
  const [auditLogExpandId, setAuditLogExpandId] = useState<string | null>(null);

  // ----- Shop state -----
  const [shopItems, setShopItems] = useState<any[]>([]);
  const [shopLoading, setShopLoading] = useState(false);
  const [showShopAdd, setShowShopAdd] = useState(false);
  const [showShopEdit, setShowShopEdit] = useState<string | null>(null);
  const [shopForm, setShopForm] = useState({ id: "", price: 100, stock: -1, is_default: true, sort_order: 0 });

  // ----- Mine state -----
  const [mineSlots, setMineSlots] = useState<any[]>([]);
  const [mineLoading, setMineLoading] = useState(false);
  const [showMineEdit, setShowMineEdit] = useState<number | null>(null);
  const [mineEditRarity, setMineEditRarity] = useState("");

  // ----- Sect state -----
  const [sects, setSects] = useState<any[]>([]);
  const [sectsLoading, setSectsLoading] = useState(false);
  const [selectedSect, setSelectedSect] = useState<any | null>(null);
  const [selectedSectMembers, setSelectedSectMembers] = useState<any[]>([]);
  const [selectedSectId, setSelectedSectId] = useState<string | null>(null);
  const [showSectEdit, setShowSectEdit] = useState(false);
  const [showSectAddMember, setShowSectAddMember] = useState(false);
  const [sectAddPlayerId, setSectAddPlayerId] = useState("");
  const [sectAddPlayerRank, setSectAddPlayerRank] = useState("NGOAI_MON");
  const [sectEditName, setSectEditName] = useState("");
  const [sectEditDesc, setSectEditDesc] = useState("");
  const [sectEditLevel, setSectEditLevel] = useState("");
  const [sectEditFunds, setSectEditFunds] = useState("");
  const [sectDetailTab, setSectDetailTab] = useState<"members" | "vault" | "missions">("members");

  const [sectMissions, setSectMissions] = useState<any[]>([]);
  const [sectMissionsLoading, setSectMissionsLoading] = useState(false);
  const [showCreateMission, setShowCreateMission] = useState(false);
  const [newMission, setNewMission] = useState({ title: "", description: "", mission_type: "GATHER", target_id: "", target_amount: 1, reward_contribution: 10, required_member_rank: "NGOAI_MON" });
  const [missionTargetSearch, setMissionTargetSearch] = useState("");

  const [showStockVault, setShowStockVault] = useState(false);
  const [stockVaultType, setStockVaultType] = useState<"gu" | "material" | "recipe">("material");
  const [stockVaultId, setStockVaultId] = useState("");
  const [stockVaultSearch, setStockVaultSearch] = useState("");
  const [stockVaultQty, setStockVaultQty] = useState(1);

  const [showRemoveVault, setShowRemoveVault] = useState<{ type: "gu" | "material"; id: string; maxQty: number } | null>(null);
  const [removeVaultQty, setRemoveVaultQty] = useState(1);

  const fetchSectMissions = useCallback(async (sectId: string) => {
    setSectMissionsLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/sects/${sectId}/missions`);
      if (res.ok) setSectMissions(await res.json());
    } catch {} finally { setSectMissionsLoading(false); }
  }, []);

  const fetchSects = useCallback(async () => {
    setSectsLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/sects`);
      if (!res.ok) throw new Error("Lỗi tải tông môn");
      setSects(await res.json());
    } catch {} finally { setSectsLoading(false); }
  }, []);

  const fetchSectDetail = useCallback(async (id: string) => {
    setSectsLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/sects/${id}`);
      if (!res.ok) throw new Error("Lỗi tải chi tiết");
      const data = await res.json();
      setSelectedSect(data.sect);
      setSelectedSectMembers(data.members);
    } catch {} finally { setSectsLoading(false); }
  }, []);

  const fetchShopItems = useCallback(async () => {
    setShopLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/shop`);
      if (res.ok) setShopItems(await res.json());
    } catch {} finally { setShopLoading(false); }
  }, []);

  const fetchMineSlots = useCallback(async () => {
    setMineLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/mine-slots`);
      if (res.ok) setMineSlots(await res.json());
    } catch {} finally { setMineLoading(false); }
  }, []);

  const fetchAuditLogs = useCallback(async () => {
    setAuditLogLoading(true);
    try {
      const params = new URLSearchParams({
        limit: String(auditLogLimit),
        offset: String(auditLogPage * auditLogLimit),
      });
      if (auditLogFilterAction) params.set("action", auditLogFilterAction);
      if (auditLogFilterPlayer) params.set("player_id", auditLogFilterPlayer);
      if (auditLogFilterSuccess) params.set("success", auditLogFilterSuccess);
      const res = await authFetch(`${BACKEND_URL}/api/admin/audit-logs?${params}`);
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data.data || []);
        setAuditLogTotal(data.total || 0);
      }
    } catch {} finally { setAuditLogLoading(false); }
  }, [auditLogPage, auditLogLimit, auditLogFilterAction, auditLogFilterPlayer, auditLogFilterSuccess]);

  useEffect(() => {
    if (section === "sects") fetchSects();
  }, [section, fetchSects]);

  useEffect(() => {
    if (section === "logs") fetchAuditLogs();
  }, [section, fetchAuditLogs]);

  useEffect(() => {
    if (section === "shop") fetchShopItems();
  }, [section, fetchShopItems]);

  useEffect(() => {
    if (section === "mines") fetchMineSlots();
  }, [section, fetchMineSlots]);

  useEffect(() => {
    if (selectedSectId) {
      fetchSectDetail(selectedSectId);
      fetchSectMissions(selectedSectId);
    }
  }, [selectedSectId, fetchSectDetail, fetchSectMissions]);

  const fetchPlayers = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(limit) });
      if (search.trim()) params.set("search", search.trim());
      const res = await authFetch(`${BACKEND_URL}/api/admin/players?${params}`);
      if (!res.ok) throw new Error((await res.json().catch(() => ({ error: "Lỗi" }))).error || "Lỗi tải");
      const data = await res.json();
      setPlayers(data.data);
      setTotal(data.total);
    } catch (err: any) {
      setError(err.message);
    } finally { setLoading(false); }
  }, [page, limit, search]);

  useEffect(() => { fetchPlayers(); }, [fetchPlayers]);

  const fetchDetail = useCallback(async (id: string) => {
    setDetailLoading(true);
    setDetail(null);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}`);
      if (!res.ok) throw new Error("Không thể tải thông tin");
      setDetail(await res.json());
    } catch { setDetail(null); }
    finally { setDetailLoading(false); }
  }, []);

  useEffect(() => {
    if (selectedId) fetchDetail(selectedId);
  }, [selectedId, fetchDetail]);

  const exec = async (url: string, method: string, body?: any) => {
    setActionLoading(true);
    try {
      const res = await authFetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: body ? JSON.stringify(body) : undefined,
      });
      const data = await res.json().catch(() => ({ error: "Lỗi không xác định" }));
      if (!res.ok) throw new Error(data.error || "Thất bại");
      if (selectedId) fetchDetail(selectedId);
      if (addToast) addToast(data.message || "Thành công", "success");
      return true;
    } catch (err: any) {
      if (addToast) addToast(err.message, "error");
      return false;
    } finally { setActionLoading(false); }
  };

  const totalPages = Math.ceil(total / limit);

  if (!player?.is_admin) {
    return (
      <div className="max-w-4xl mx-auto text-center py-20 text-zinc-500 font-mono">
        <Shield className="w-12 h-12 mx-auto mb-4 text-zinc-700" />
        <p>Bạn không có quyền truy cập khu vực này.</p>
      </div>
    );
  }

  if (selectedId) {
    return (
      <div className="w-full max-w-6xl mx-auto px-4 py-6 space-y-4">
        {/* Detail Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button onClick={() => { setSelectedId(null); setDetail(null); }} className="p-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 transition-colors cursor-pointer">
              <ArrowLeft className="w-4 h-4 text-zinc-400" />
            </button>
            <div>
              <h1 className="text-lg font-serif font-bold text-zinc-100 flex items-center gap-2">
                <Shield className="w-5 h-5 text-amber-400" />
                {detail ? detail.username : "Đang tải..."}
              </h1>
              {detail && (
                <p className="text-xs text-zinc-500 font-mono">{detail.id} • {detail.email}</p>
              )}
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={() => setShowStones(true)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
              <Coins className="w-3.5 h-3.5" />
              Điều chỉnh tài nguyên
            </button>
            <button onClick={() => { setSelectedId(null); setDetail(null); }} disabled={actionLoading} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
              <X className="w-3.5 h-3.5" />
              Đóng
            </button>
          </div>
        </div>

        {detailLoading ? (
          <div className="flex flex-col items-center py-20"><RefreshCw className="w-8 h-8 text-amber-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-3">Đang tải thông tin...</span></div>
        ) : detail ? (
          <>
            {/* Quick Stats */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Sword className="w-3.5 h-3.5" />Tu vi</div>
                <span className={`text-sm font-bold ${rankColor(detail.rank)}`}>{getRankName(detail.rank, detail.sub_rank)}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Heart className="w-3.5 h-3.5 text-red-400" />Máu</div>
                <span className="text-sm font-bold text-zinc-200">{Math.round(detail.hp)}/{Math.round(detail.base_hp)}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Zap className="w-3.5 h-3.5 text-blue-400" />Mana</div>
                <span className="text-sm font-bold text-zinc-200">{Math.round(detail.current_mana)}/{Math.round(detail.max_mana)}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Coins className="w-3.5 h-3.5 text-yellow-500" />Nguyên Thạch</div>
                <span className="text-sm font-bold text-yellow-500">{detail.primeval_stones.toLocaleString()}</span>
              </div>
            </div>

            {/* Tabs */}
            <div className="border-b border-zinc-800/80 flex gap-0">
              <button onClick={() => setActiveTab("items")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "items" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Package className="w-3.5 h-3.5 inline mr-1.5" />Vật phẩm ({Object.keys(detail.materials_inventory || {}).length})
              </button>
              <button onClick={() => setActiveTab("gus")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "gus" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Bug className="w-3.5 h-3.5 inline mr-1.5" />Cổ trùng ({detail.gu_inventory?.length || 0})
              </button>
            </div>

            {/* Items Tab */}
            {activeTab === "items" && (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                    <input type="text" value={itemSearch} onChange={e => setItemSearch(e.target.value)} placeholder="Tìm vật phẩm trong túi đồ..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setAddItemId(""); setAddItemQty(1); setShowAddItem(true); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>
                </div>

                {Object.keys(detail.materials_inventory || {}).length === 0 ? (
                  <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                    <Package className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                    <p className="text-xs text-zinc-500">Túi đồ trống</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2.5 font-semibold">ID</th>
                          <th className="px-3 py-2.5 font-semibold">Tên</th>
                          <th className="px-3 py-2.5 font-semibold">Loại</th>
                          <th className="px-3 py-2.5 font-semibold">Rank</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Số lượng</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(detail.materials_inventory)
                          .filter(([id]) => {
                            if (!itemSearch.trim()) return true;
                            const spec = itemDictionary[id];
                            const name = spec?.name || "";
                            return id.toLowerCase().includes(itemSearch.toLowerCase()) || name.toLowerCase().includes(itemSearch.toLowerCase());
                          })
                          .sort((a, b) => b[1] - a[1])
                          .map(([itemId, qty]) => {
                            const spec = itemDictionary[itemId];
                            return (
                              <tr key={itemId} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                                <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[100px] truncate" title={itemId}>{itemId}</td>
                                <td className="px-3 py-2.5 font-semibold text-zinc-200">{spec?.name || itemId}</td>
                                <td className="px-3 py-2.5">
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-zinc-900 border border-zinc-800 text-zinc-400">{spec?.item_type || "?"}</span>
                                </td>
                                <td className="px-3 py-2.5 text-zinc-400">{spec?.rank || "?"}</td>
                                <td className="px-3 py-2.5 text-right font-mono font-bold text-zinc-200">{qty.toLocaleString()}</td>
                                <td className="px-3 py-2.5 text-right">
                                  <button onClick={() => { setShowRemoveItem(itemId); setRemoveItemQty(1); }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer" title="Xóa">
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </td>
                              </tr>
                            );
                          })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {/* Gus Tab */}
            {activeTab === "gus" && (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                    <input type="text" value={guSearch} onChange={e => setGuSearch(e.target.value)} placeholder="Tìm cổ trùng..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setAddGuId(""); setAddGuQty(1); setShowAddGu(true); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>
                </div>

                {(!detail.gu_inventory || detail.gu_inventory.length === 0) ? (
                  <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                    <Bug className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                    <p className="text-xs text-zinc-500">Không có cổ trùng nào</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2.5 font-semibold">Instance ID</th>
                          <th className="px-3 py-2.5 font-semibold">Tên</th>
                          <th className="px-3 py-2.5 font-semibold">Rank</th>
                          <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                          <th className="px-3 py-2.5 font-semibold">Đói</th>
                          <th className="px-3 py-2.5 font-semibold">Trang bị</th>
                          <th className="px-3 py-2.5 font-semibold">Bản Mệnh</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detail.gu_inventory
                          .filter((gu) => {
                            if (!guSearch.trim()) return true;
                            const spec = guDictionary[gu.gu_id];
                            const name = spec?.name || "";
                            return gu.gu_id.toLowerCase().includes(guSearch.toLowerCase()) || name.toLowerCase().includes(guSearch.toLowerCase());
                          })
                          .map((gu) => {
                            const spec = guDictionary[gu.gu_id];
                            return (
                              <tr key={gu.instance_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                                <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate" title={gu.instance_id}>{gu.instance_id.slice(0, 8)}...</td>
                                <td className="px-3 py-2.5 font-semibold text-zinc-200">{spec?.name || gu.gu_id}</td>
                                <td className={`px-3 py-2.5 font-bold ${rankColor(spec?.rank || 1)}`}>{spec?.rank || "?"}</td>
                                <td className="px-3 py-2.5">
                                  <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold border ${statusColor(gu.status)}`}>{statusLabel(gu.status)}</span>
                                </td>
                                <td className="px-3 py-2.5 font-mono text-zinc-300">{gu.hunger}</td>
                                <td className="px-3 py-2.5">{gu.equipped ? <span className="text-emerald-400 text-[11px] font-bold">✓</span> : <span className="text-zinc-600">–</span>}</td>
                                <td className="px-3 py-2.5">{gu.is_vital ? <Star className="w-3.5 h-3.5 text-amber-400" /> : <span className="text-zinc-600">–</span>}</td>
                                <td className="px-3 py-2.5 text-right">
                                  <button
                                    onClick={async () => {
                                      if (!confirm("Xóa cổ trùng này?")) return;
                                      await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/gus/${gu.instance_id}`, "DELETE");
                                    }}
                                    className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer"
                                    title="Xóa"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </td>
                              </tr>
                            );
                          })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-16">
            <p className="text-sm text-red-400">Không thể tải thông tin người chơi</p>
            <button onClick={() => selectedId && fetchDetail(selectedId)} className="mt-2 text-xs text-amber-500 underline cursor-pointer">Thử lại</button>
          </div>
        )}

        {/* Stones Modal */}
        {showStones && detail && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Điều chỉnh tài nguyên</h3>
                <button onClick={() => setShowStones(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div className="space-y-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Nguyên Thạch (hiện tại: {detail.primeval_stones.toLocaleString()})</label>
                  <input type="number" value={stonesPrimeval} onChange={e => setStonesPrimeval(e.target.value)} placeholder="Để trống nếu không đổi" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiên Nguyên Thạch (hiện tại: {detail.immortal_stones.toLocaleString()})</label>
                  <input type="number" step="any" value={stonesImmortal} onChange={e => setStonesImmortal(e.target.value)} placeholder="Để trống nếu không đổi" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowStones(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const body: any = {};
                  if (stonesPrimeval) body.primeval_stones = parseInt(stonesPrimeval);
                  if (stonesImmortal) body.immortal_stones = parseFloat(stonesImmortal);
                  if (Object.keys(body).length === 0) return;
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/stones`, "PUT", body);
                  if (ok) setShowStones(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Lưu"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Add Item Modal */}
        {showAddItem && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm vật phẩm</h3>
                <button onClick={() => setShowAddItem(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn vật phẩm</label>
                <select value={addItemId} onChange={e => setAddItemId(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                  <option value="">-- Chọn --</option>
                  {Object.values(itemDictionary)
                    .sort((a, b) => a.name.localeCompare(b.name))
                    .map(spec => (
                      <option key={spec.id} value={spec.id}>{spec.name} ({spec.id}) [{spec.item_type}]</option>
                    ))}
                </select>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng</label>
                <input type="number" min={1} value={addItemQty} onChange={e => setAddItemQty(Math.max(1, parseInt(e.target.value) || 1))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowAddItem(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !addItemId} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/items`, "POST", { item_id: addItemId, quantity: addItemQty });
                  if (ok) setShowAddItem(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Remove Item Modal */}
        {showRemoveItem && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Xóa vật phẩm</h3>
                <button onClick={() => setShowRemoveItem(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <p className="text-xs text-zinc-400">Vật phẩm: <span className="text-zinc-200 font-bold">{itemDictionary[showRemoveItem]?.name || showRemoveItem}</span></p>
              <p className="text-xs text-zinc-500">Hiện có: <span className="text-zinc-300 font-bold">{detail?.materials_inventory[showRemoveItem]?.toLocaleString() || 0}</span></p>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng xóa</label>
                <input type="number" min={1} max={detail?.materials_inventory[showRemoveItem] || 1} value={removeItemQty} onChange={e => setRemoveItemQty(Math.max(1, Math.min(parseInt(e.target.value) || 1, detail?.materials_inventory[showRemoveItem] || 1)))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowRemoveItem(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/items`, "DELETE", { item_id: showRemoveItem, quantity: removeItemQty });
                  if (ok) setShowRemoveItem(null);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-red-600 hover:bg-red-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Xóa"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Add Gu Modal */}
        {showAddGu && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm cổ trùng</h3>
                <button onClick={() => setShowAddGu(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn cổ trùng</label>
                <select value={addGuId} onChange={e => setAddGuId(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                  <option value="">-- Chọn --</option>
                  {Object.values(guDictionary)
                    .sort((a, b) => (b.rank || 0) - (a.rank || 0))
                    .map(spec => (
                      <option key={spec.id} value={spec.id}>{spec.name} (Rank {spec.rank}) [{spec.gu_type}]</option>
                    ))}
                </select>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng</label>
                <input type="number" min={1} value={addGuQty} onChange={e => setAddGuQty(Math.max(1, parseInt(e.target.value) || 1))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowAddGu(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !addGuId} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/gus`, "POST", { gu_id: addGuId, quantity: addGuQty });
                  if (ok) setShowAddGu(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  }

  // ===== Main render (player detail or sect detail take priority) =====
  if (selectedSectId) {
    return (
      <div className="w-full max-w-6xl mx-auto px-4 py-6 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button onClick={() => { setSelectedSectId(null); setSelectedSect(null); setSelectedSectMembers([]); }} className="p-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 transition-colors cursor-pointer">
              <ArrowLeft className="w-4 h-4 text-zinc-400" />
            </button>
            <div>
              <h1 className="text-lg font-serif font-bold text-zinc-100 flex items-center gap-2">
                <Building2 className="w-5 h-5 text-amber-400" />
                {selectedSect?.name || "Đang tải..."}
              </h1>
              {selectedSect && (
                <p className="text-xs text-zinc-500 font-mono">{selectedSect.id} • Cấp {selectedSect.level} • {selectedSect.region}</p>
              )}
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={() => { if (selectedSect) { setSectEditName(selectedSect.name); setSectEditDesc(selectedSect.description || ""); setSectEditLevel(String(selectedSect.level)); setSectEditFunds(String(selectedSect.funds)); setShowSectEdit(true); } }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
              <Edit3 className="w-3.5 h-3.5" /> Sửa
            </button>
            <button onClick={() => { setSectAddPlayerId(""); setSectAddPlayerRank("NGOAI_MON"); setShowSectAddMember(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs text-zinc-100 transition-colors cursor-pointer">
              <UserPlus className="w-3.5 h-3.5" /> Thêm thành viên
            </button>
            <button onClick={async () => {
              if (!confirm(`Giải tán "${selectedSect?.name}"? Hành động này không thể hoàn tác!`)) return;
              const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}`, "DELETE");
              if (ok) { setSelectedSectId(null); setSelectedSect(null); fetchSects(); }
            }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-900/40 hover:bg-red-900/60 border border-red-800/40 text-xs text-red-400 transition-colors cursor-pointer">
              <Trash2 className="w-3.5 h-3.5" /> Giải tán
            </button>
          </div>
        </div>

        {sectsLoading ? (
          <div className="flex flex-col items-center py-20"><RefreshCw className="w-8 h-8 text-amber-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-3">Đang tải...</span></div>
        ) : selectedSect ? (
          <>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="text-xs text-zinc-500 font-mono mb-1"><Layers className="w-3.5 h-3.5 inline mr-1" />Cấp độ</div>
                <span className="text-sm font-bold text-zinc-200">{selectedSect.level}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="text-xs text-zinc-500 font-mono mb-1"><Coins className="w-3.5 h-3.5 inline mr-1 text-yellow-500" />Quỹ</div>
                <span className="text-sm font-bold text-yellow-500">{selectedSect.funds?.toLocaleString() || 0}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="text-xs text-zinc-500 font-mono mb-1"><Globe className="w-3.5 h-3.5 inline mr-1" />Khu vực</div>
                <span className="text-sm font-bold text-zinc-200">{selectedSect.region}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="text-xs text-zinc-500 font-mono mb-1"><Users className="w-3.5 h-3.5 inline mr-1" />Thành viên</div>
                <span className="text-sm font-bold text-zinc-200">{selectedSectMembers?.length}</span>
              </div>
            </div>

            {/* Detail tabs */}
            <div className="border-b border-zinc-800/80 flex gap-0">
              <button onClick={() => setSectDetailTab("members")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${sectDetailTab === "members" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Users className="w-3.5 h-3.5 inline mr-1.5" />Thành viên ({selectedSectMembers?.length})
              </button>
              <button onClick={() => setSectDetailTab("vault")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${sectDetailTab === "vault" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Package className="w-3.5 h-3.5 inline mr-1.5" />Bảo khố
              </button>
              <button onClick={() => { setSectDetailTab("missions"); if (selectedSectId) fetchSectMissions(selectedSectId); }} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${sectDetailTab === "missions" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <ListTodo className="w-3.5 h-3.5 inline mr-1.5" />Nhiệm vụ ({sectMissions?.length})
              </button>
            </div>

            {/* Members tab */}
            {sectDetailTab === "members" && (
              <>
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                  <table className="w-full text-left text-xs">
                    <thead>
                      <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                        <th className="px-3 py-2.5 font-semibold">Player ID</th>
                        <th className="px-3 py-2.5 font-semibold">Tên</th>
                        <th className="px-3 py-2.5 font-semibold">Cấp</th>
                        <th className="px-3 py-2.5 font-semibold">Chức vụ</th>
                        <th className="px-3 py-2.5 font-semibold text-right">Cống hiến</th>
                        <th className="px-3 py-2.5 font-semibold">Ngày gia nhập</th>
                        <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedSectMembers?.map((m: any) => (
                        <tr key={m.player_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate" title={m.player_id}>{m.player_id.slice(0, 8)}...</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200 flex items-center gap-1.5">
                            {m.member_rank === "CHUONG_MON" && <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" />}
                            {m.username}
                          </td>
                          <td className={`px-3 py-2.5 font-bold ${rankColor(m.player_rank)}`}>{getRankName(m.player_rank, 1)}</td>
                          <td className="px-3 py-2.5">
                            <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold border ${memberRankStyle(m.member_rank)}`}>{memberRankLabel(m.member_rank)}</span>
                          </td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{m.contribution_points?.toLocaleString() || 0}</td>
                          <td className="px-3 py-2.5 text-zinc-500 font-mono text-[10px]">{formatDate(m.joined_at)}</td>
                          <td className="px-3 py-2.5 text-right">
                            <div className="flex items-center justify-end gap-1">
                              {m.member_rank !== "CHUONG_MON" && (
                                <button onClick={async () => {
                                  if (!confirm(`Bổ nhiệm ${m.username} làm Chưởng môn?`)) return;
                                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/members/${m.player_id}/leader`, "POST");
                                  if (ok) fetchSectDetail(selectedSectId);
                                }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 transition-colors cursor-pointer" title="Bổ nhiệm Chưởng môn">
                                  <Crown className="w-3.5 h-3.5" />
                                </button>
                              )}
                              <button onClick={async () => {
                                if (!confirm(`Đuổi ${m.username} khỏi tông môn?`)) return;
                                const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/members/${m.player_id}`, "DELETE");
                                if (ok) fetchSectDetail(selectedSectId);
                              }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer" title="Đuổi">
                                <UserMinus className="w-3.5 h-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}

            {/* Vault tab */}
            {sectDetailTab === "vault" && (
              <div className="space-y-4">
                <div className="flex justify-end">
                  <button onClick={() => { setStockVaultType("material"); setStockVaultId(""); setStockVaultQty(1); setShowStockVault(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 rounded-lg transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm vật phẩm
                  </button>
                </div>

                {/* vault_gu */}
                <div>
                  <h4 className="text-xs font-bold text-zinc-400 font-mono uppercase tracking-wider mb-2">Cổ trùng trong bảo khố</h4>
                  {selectedSect.vault_gu && Array.isArray(selectedSect.vault_gu) && selectedSect.vault_gu.length > 0 ? (
                    <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                      <table className="w-full text-left text-xs">
                        <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono text-[10px]"><th className="px-3 py-2 font-semibold">ID</th><th className="px-3 py-2 font-semibold">Tên</th><th className="px-3 py-2 font-semibold text-right">Giá</th><th className="px-3 py-2 font-semibold text-right">SL</th><th className="px-3 py-2 font-semibold text-right"></th></tr></thead>
                        <tbody>
                          {selectedSect.vault_gu.map((item: any, i: number) => {
                            const spec = guDictionary[item.id];
                            return (
                              <tr key={i} className="border-t border-zinc-800/40">
                                <td className="px-3 py-2 text-zinc-500 font-mono">{item.id}</td>
                                <td className="px-3 py-2 font-semibold text-zinc-200">{spec?.name || item.id}</td>
                                <td className="px-3 py-2 text-right font-mono text-zinc-400">{item.cost?.toLocaleString() || 0}</td>
                                <td className="px-3 py-2 text-right font-mono text-zinc-300">{item.quantity}</td>
                                <td className="px-3 py-2 text-right">
                                  <button onClick={() => { setShowRemoveVault({ type: "gu", id: item.id, maxQty: item.quantity }); setRemoveVaultQty(1); }} className="p-1 hover:bg-red-950/40 rounded text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3 h-3" /></button>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  ) : <p className="text-xs text-zinc-600 italic py-4 text-center">Trống</p>}
                </div>

                {/* vault_materials */}
                <div>
                  <h4 className="text-xs font-bold text-zinc-400 font-mono uppercase tracking-wider mb-2">Vật liệu trong bảo khố</h4>
                  {selectedSect.vault_materials && typeof selectedSect.vault_materials === "object" && Object.keys(selectedSect.vault_materials).length > 0 ? (
                    <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                      <table className="w-full text-left text-xs">
                        <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono text-[10px]"><th className="px-3 py-2 font-semibold">ID</th><th className="px-3 py-2 font-semibold">Tên</th><th className="px-3 py-2 font-semibold text-right">SL</th><th className="px-3 py-2 font-semibold text-right"></th></tr></thead>
                        <tbody>
                          {Object.entries(selectedSect.vault_materials).map(([id, qty]: [string, any]) => {
                            const spec = itemDictionary[id];
                            return (
                              <tr key={id} className="border-t border-zinc-800/40">
                                <td className="px-3 py-2 text-zinc-500 font-mono">{id}</td>
                                <td className="px-3 py-2 font-semibold text-zinc-200">{spec?.name || id}</td>
                                <td className="px-3 py-2 text-right font-mono text-zinc-300">{qty}</td>
                                <td className="px-3 py-2 text-right">
                                  <button onClick={() => { setShowRemoveVault({ type: "material", id, maxQty: qty }); setRemoveVaultQty(1); }} className="p-1 hover:bg-red-950/40 rounded text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3 h-3" /></button>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  ) : <p className="text-xs text-zinc-600 italic py-4 text-center">Trống</p>}
                </div>

                {/* vault_recipes */}
                <div>
                  <h4 className="text-xs font-bold text-zinc-400 font-mono uppercase tracking-wider mb-2">Công thức trong bảo khố</h4>
                  {selectedSect.vault_recipes && Array.isArray(selectedSect.vault_recipes) && selectedSect.vault_recipes.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {selectedSect.vault_recipes.map((recipeId: string, i: number) => (
                        <span key={i} className="px-2 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 font-mono">{recipeId}</span>
                      ))}
                    </div>
                  ) : <p className="text-xs text-zinc-600 italic py-4 text-center">Trống</p>}
                </div>
              </div>
            )}

            {/* Missions tab */}
            {sectDetailTab === "missions" && (
              <div className="space-y-3">
                <div className="flex justify-end">
                  <button onClick={() => { setNewMission({ title: "", description: "", mission_type: "GATHER", target_id: "", target_amount: 1, reward_contribution: 10, required_member_rank: "NGOAI_MON" }); setShowCreateMission(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 rounded-lg transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Tạo nhiệm vụ
                  </button>
                </div>
                {sectMissionsLoading ? (
                  <div className="flex justify-center py-8"><RefreshCw className="w-5 h-5 text-amber-400 animate-spin" /></div>
                ) : sectMissions?.length === 0 ? (
                  <p className="text-xs text-zinc-600 italic py-8 text-center">Chưa có nhiệm vụ nào</p>
                ) : (
                  <div className="space-y-2">
                    {sectMissions?.map((m: any) => (
                      <div key={m.id} className="bg-zinc-900/30 border border-zinc-800/60 rounded-xl p-3 flex items-start justify-between gap-3">
                        <div className="space-y-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-bold text-zinc-200">{m.title}</span>
                            <span className={`px-1.5 py-0.5 rounded text-[9px] font-bold border ${memberRankStyle(m.required_member_rank)}`}>{memberRankLabel(m.required_member_rank)}</span>
                          </div>
                          <p className="text-[11px] text-zinc-500">{m.description}</p>
                          <div className="flex flex-wrap gap-2 text-[10px] text-zinc-400 font-mono">
                            <span>Loại: {m.mission_type}</span>
                            <span>Mục tiêu: {m.target_id} x{m.target_amount}</span>
                            <span>Thưởng: {m.reward_contribution} cống hiến</span>
                          </div>
                        </div>
                        <button onClick={async () => {
                          if (!confirm(`Xóa nhiệm vụ "${m.title}"?`)) return;
                          const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/missions/${m.id}`, "DELETE");
                          if (ok && selectedSectId) fetchSectMissions(selectedSectId);
                        }} className="p-1.5 hover:bg-red-950/40 rounded-lg text-zinc-500 hover:text-red-400 transition-colors shrink-0 cursor-pointer">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-16"><p className="text-sm text-red-400">Không thể tải thông tin</p></div>
        )}

        {/* Edit Sect Modal */}
        {showSectEdit && selectedSect && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa tông môn</h3>
                <button onClick={() => setShowSectEdit(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div className="space-y-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tên</label>
                  <input type="text" value={sectEditName} onChange={e => setSectEditName(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Mô tả</label>
                  <textarea value={sectEditDesc} onChange={e => setSectEditDesc(e.target.value)} rows={3} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Cấp</label>
                    <input type="number" min={1} value={sectEditLevel} onChange={e => setSectEditLevel(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  </div>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Quỹ</label>
                    <input type="number" min={0} value={sectEditFunds} onChange={e => setSectEditFunds(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  </div>
                </div>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowSectEdit(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const body: any = {};
                  if (sectEditName !== selectedSect.name) body.name = sectEditName;
                  if (sectEditDesc !== (selectedSect.description || "")) body.description = sectEditDesc;
                  if (Number(sectEditLevel) !== selectedSect.level) body.level = Number(sectEditLevel);
                  if (Number(sectEditFunds) !== selectedSect.funds) body.funds = Number(sectEditFunds);
                  if (Object.keys(body).length === 0) { setShowSectEdit(false); return; }
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}`, "PUT", body);
                  if (ok) { fetchSectDetail(selectedSectId); setShowSectEdit(false); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Lưu"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Add Member Modal */}
        {showSectAddMember && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm thành viên</h3>
                <button onClick={() => setShowSectAddMember(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Player ID (UUID)</label>
                <input type="text" value={sectAddPlayerId} onChange={e => setSectAddPlayerId(e.target.value)} placeholder="player-uuid" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Chức vụ</label>
                <select value={sectAddPlayerRank} onChange={e => setSectAddPlayerRank(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                  <option value="NGOAI_MON">Ngoại Môn</option>
                  <option value="NOI_MON">Nội Môn</option>
                  <option value="TINH_ANH">Tinh Anh</option>
                  <option value="CHAN_TRUYEN">Chân Truyền</option>
                  <option value="TRUONG_LAO">Trưởng Lão</option>
                  <option value="CHUONG_MON">Chưởng Môn</option>
                </select>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowSectAddMember(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !sectAddPlayerId} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/members`, "POST", { player_id: sectAddPlayerId, member_rank: sectAddPlayerRank });
                  if (ok) { fetchSectDetail(selectedSectId); setShowSectAddMember(false); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Stock Vault Modal */}
        {showStockVault && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm vào bảo khố</h3>
                <button onClick={() => setShowStockVault(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Loại</label>
                <select value={stockVaultType} onChange={e => setStockVaultType(e.target.value as any)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                  <option value="material">Vật liệu</option>
                  <option value="gu">Cổ trùng</option>
                  <option value="recipe">Công thức</option>
                </select>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Vật phẩm</label>
                {stockVaultType === "recipe" ? (
                  <input type="text" value={stockVaultId} onChange={e => setStockVaultId(e.target.value)} placeholder="recipe_id" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                ) : (
                  <div className="mt-1 flex flex-col gap-1">
                    <input type="text" value={stockVaultSearch} onChange={e => { setStockVaultSearch(e.target.value); setStockVaultId(""); }} placeholder="Lọc nhanh..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50" />
                    <select value={stockVaultId} onChange={e => setStockVaultId(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50">
                      <option value="">-- Chọn --</option>
                      {(stockVaultType === "gu" ? Object.entries(guDictionary) : Object.entries(itemDictionary))
                        .filter(([id, spec]) => !stockVaultSearch || id.toLowerCase().includes(stockVaultSearch.toLowerCase()) || (spec as any)?.name?.toLowerCase().includes(stockVaultSearch.toLowerCase()))
                        .sort(([, a], [, b]) => ((a as any)?.rank || 99) - ((b as any)?.rank || 99))
                        .map(([id, spec]) => (
                          <option key={id} value={id}>{(spec as any)?.name || id}</option>
                        ))}
                    </select>
                  </div>
                )}
              </div>
              {stockVaultType !== "recipe" && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng</label>
                  <input type="number" min={1} value={stockVaultQty} onChange={e => setStockVaultQty(Math.max(1, parseInt(e.target.value) || 1))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
              )}
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowStockVault(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !stockVaultId} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/vault/stock`, "POST", { item_type: stockVaultType, item_id: stockVaultId, quantity: stockVaultType === "recipe" ? 1 : stockVaultQty });
                  if (ok) { fetchSectDetail(selectedSectId!); setShowStockVault(false); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Remove Vault Item Modal */}
        {showRemoveVault && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Xóa khỏi bảo khố</h3>
                <button onClick={() => setShowRemoveVault(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <p className="text-xs text-zinc-400">ID: <span className="text-zinc-200 font-bold">{showRemoveVault.id}</span></p>
              <p className="text-xs text-zinc-500">Hiện có: <span className="text-zinc-300 font-bold">{showRemoveVault.maxQty}</span></p>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng xóa</label>
                <input type="number" min={1} max={showRemoveVault.maxQty} value={removeVaultQty} onChange={e => setRemoveVaultQty(Math.max(1, Math.min(parseInt(e.target.value) || 1, showRemoveVault.maxQty)))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowRemoveVault(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/vault/stock`, "DELETE", { item_type: showRemoveVault.type, item_id: showRemoveVault.id, quantity: removeVaultQty });
                  if (ok) { fetchSectDetail(selectedSectId!); setShowRemoveVault(null); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-red-600 hover:bg-red-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Xóa"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Create Mission Modal */}
        {showCreateMission && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Tạo nhiệm vụ</h3>
                <button onClick={() => setShowCreateMission(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div className="space-y-3 max-h-[60vh] overflow-y-auto">
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiêu đề</label>
                  <input type="text" value={newMission.title} onChange={e => setNewMission(p => ({ ...p, title: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Mô tả</label>
                  <textarea value={newMission.description} onChange={e => setNewMission(p => ({ ...p, description: e.target.value }))} rows={2} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Loại</label>
                    <select value={newMission.mission_type} onChange={e => setNewMission(p => ({ ...p, mission_type: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                      <option value="GATHER">Thu thập linh tài</option>
                      <option value="DONATE">Cống hiến cổ trùng</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Chức vụ yêu cầu</label>
                    <select value={newMission.required_member_rank} onChange={e => setNewMission(p => ({ ...p, required_member_rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                      <option value="NGOAI_MON">Ngoại Môn</option>
                      <option value="NOI_MON">Nội Môn</option>
                      <option value="TINH_ANH">Tinh Anh</option>
                      <option value="CHAN_TRUYEN">Chân Truyền</option>
                      <option value="TRUONG_LAO">Trưởng Lão</option>
                    </select>
                  </div>
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Mục tiêu</label>
                  <div className="mt-1 flex flex-col gap-1">
                    <input type="text" value={missionTargetSearch} onChange={e => { setMissionTargetSearch(e.target.value); setNewMission(p => ({ ...p, target_id: "" })); }} placeholder="Lọc nhanh..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50" />
                    <select value={newMission.target_id} onChange={e => setNewMission(p => ({ ...p, target_id: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50">
                      <option value="">-- Chọn --</option>
                      {(newMission.mission_type === "DONATE" ? Object.entries(guDictionary) : Object.entries(itemDictionary))
                        //Search both by id and name (if available)
                        .filter(([id, spec]) => !missionTargetSearch || id.toLowerCase().includes(missionTargetSearch.toLowerCase()) || (spec as any)?.name?.toLowerCase().includes(missionTargetSearch.toLowerCase()))
                        .sort(([, a], [, b]) => ((a as any)?.rank || 99) - ((b as any)?.rank || 99))
                        .map(([id, spec]) => (
                          <option key={id} value={id}>{(spec as any)?.name || id}</option>
                        ))}
                    </select>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng yêu cầu</label>
                    <input type="number" min={1} value={newMission.target_amount} onChange={e => setNewMission(p => ({ ...p, target_amount: Math.max(1, parseInt(e.target.value) || 1) }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  </div>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Thưởng cống hiến</label>
                    <input type="number" min={1} value={newMission.reward_contribution} onChange={e => setNewMission(p => ({ ...p, reward_contribution: Math.max(1, parseInt(e.target.value) || 1) }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  </div>
                </div>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowCreateMission(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !newMission.title || !newMission.target_id} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}/missions`, "POST", newMission);
                  if (ok && selectedSectId) { fetchSectMissions(selectedSectId); setShowCreateMission(false); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Tạo"}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  }

  // ===== Main Admin Page =====
  return (
    <div className="w-full max-w-6xl mx-auto px-4 py-6">
      {/* Header with section tabs */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between mb-6 gap-3">
        <div className="flex items-center gap-3">
          <button onClick={() => setView("dashboard")} className="p-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 transition-colors cursor-pointer">
            <ArrowLeft className="w-4 h-4 text-zinc-400" />
          </button>
          <div>
            <h1 className="text-xl font-serif font-bold text-zinc-100 flex items-center gap-2">
              <Shield className="w-5 h-5 text-amber-400" />
              Bảng Điều Khiển Admin
            </h1>
            <p className="text-xs text-zinc-500 font-mono mt-0.5">Quản lý tài nguyên và người chơi</p>
          </div>
        </div>
      </div>

      {/* Section tabs */}
      <div className="border-b border-zinc-800/80 flex gap-0 mb-4">
        <button onClick={() => setSection("players")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "players" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Users className="w-3.5 h-3.5 inline mr-1.5" />Người chơi
        </button>
        <button onClick={() => setSection("sects")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "sects" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Building2 className="w-3.5 h-3.5 inline mr-1.5" />Tông môn
        </button>
        <button onClick={() => setSection("data")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "data" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Database className="w-3.5 h-3.5 inline mr-1.5" />Dữ liệu
        </button>
        <button onClick={() => setSection("logs")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "logs" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <BookOpen className="w-3.5 h-3.5 inline mr-1.5" />Nhật ký
        </button>
        <button onClick={() => setSection("shop")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "shop" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Package className="w-3.5 h-3.5 inline mr-1.5" />Cửa hàng
        </button>
        <button onClick={() => setSection("mines")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "mines" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Layers className="w-3.5 h-3.5 inline mr-1.5" />Mỏ
        </button>
      </div>

      {/* Players section */}
      {section === "players" && (
        <>
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4 mb-4">
            <div className="flex gap-3 items-end">
              <div className="flex-1 space-y-1">
                <label className="text-[11px] font-bold text-zinc-500 uppercase tracking-wider font-mono flex items-center gap-1">
                  <Search className="w-3 h-3" /> Tìm kiếm
                </label>
                <input type="text" value={search} onChange={e => setSearch(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { setPage(1); fetchPlayers(); } }} placeholder="Tên đăng nhập hoặc email..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors placeholder:text-zinc-600" />
              </div>
              <button onClick={() => { setPage(1); fetchPlayers(); }} className="px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">Tìm</button>
              <button onClick={fetchPlayers} disabled={loading} className="flex items-center gap-1.5 px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50 cursor-pointer">
                <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>

          {loading ? (
            <div className="flex flex-col items-center justify-center py-20 space-y-3">
              <RefreshCw className="w-8 h-8 text-amber-400 animate-spin" />
              <span className="text-xs text-zinc-500 font-mono">Đang tải danh sách người chơi...</span>
            </div>
          ) : error ? (
            <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-red-900/50 rounded-xl space-y-2">
              <p className="text-sm text-red-400 font-medium">{error}</p>
              <button onClick={fetchPlayers} className="text-xs text-amber-500 hover:text-amber-400 underline cursor-pointer">Thử lại</button>
            </div>
          ) : players.length === 0 ? (
            <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
              <Users className="w-10 h-10 text-zinc-700 mx-auto" />
              <p className="text-sm text-zinc-400 font-medium">Không tìm thấy người chơi nào.</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold">ID</th>
                      <th className="px-3 py-2.5 font-semibold">Tên</th>
                      <th className="px-3 py-2.5 font-semibold">Email</th>
                      <th className="px-3 py-2.5 font-semibold">Tu vi</th>
                      <th className="px-3 py-2.5 font-semibold">HP</th>
                      <th className="px-3 py-2.5 font-semibold">Mana</th>
                      <th className="px-3 py-2.5 font-semibold">Nguyên Thạch</th>
                      <th className="px-3 py-2.5 font-semibold">Tiên Nguyên</th>
                      <th className="px-3 py-2.5 font-semibold">Admin</th>
                      <th className="px-3 py-2.5 font-semibold">Khu vực</th>
                      <th className="px-3 py-2.5 font-semibold">Ngày tạo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {players.map((p) => (
                      <tr key={p.id} onClick={() => setSelectedId(p.id)} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors cursor-pointer">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate" title={p.id}>{p.id.slice(0, 8)}...</td>
                        <td className="px-3 py-2.5 font-semibold text-zinc-200">{p.username}</td>
                        <td className="px-3 py-2.5 text-zinc-400">{p.email}</td>
                        <td className={`px-3 py-2.5 font-bold ${rankColor(p.rank)}`}>{getRankName(p.rank, p.sub_rank)}</td>
                        <td className="px-3 py-2.5 font-mono text-zinc-300">{Math.round(p.hp)}/{Math.round(p.base_hp)}</td>
                        <td className="px-3 py-2.5 font-mono text-zinc-300">{Math.round(p.current_mana)}/{Math.round(p.max_mana)}</td>
                        <td className="px-3 py-2.5 font-mono text-yellow-500">{p.primeval_stones.toLocaleString()}</td>
                        <td className="px-3 py-2.5 font-mono text-purple-400">{p.immortal_stones.toLocaleString()}</td>
                        <td className="px-3 py-2.5">{p.is_admin ? <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-950/80 border border-amber-500/40 text-amber-400">ADMIN</span> : <span className="text-zinc-600">–</span>}</td>
                        <td className="px-3 py-2.5 text-zinc-400 max-w-[100px] truncate">{p.current_region}</td>
                        <td className="px-3 py-2.5 text-zinc-500 font-mono text-[10px]">{formatDate(p.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="flex items-center justify-between mt-4 text-xs font-semibold text-zinc-500">
                <div>Tổng số: <span className="text-zinc-300">{total.toLocaleString()}</span> người chơi</div>
                <div className="flex items-center gap-3">
                  <span>Trang {page}/{totalPages || 1}</span>
                  <div className="flex gap-2">
                    <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page <= 1 || loading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                      <ChevronLeft className="w-4 h-4" /> Trước
                    </button>
                    <button onClick={() => setPage(page + 1)} disabled={page >= totalPages || loading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                      Sau <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            </>
          )}
        </>
      )}

      {/* Data section: import/export CSV */}
      {section === "data" && (
        <div className="space-y-6">
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-6">
            <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-2">
              <Database className="w-4 h-4 text-emerald-400" />
              Quản Lý Dữ Liệu
            </h3>
            <p className="text-xs text-zinc-500 mb-6">
              Import dữ liệu từ file CSV vào database, hoặc export dữ liệu từ database ra file CSV.
              Dùng khi cần đồng bộ dữ liệu game sau khi chỉnh sửa file CSV.
            </p>
            <div className="flex flex-wrap gap-4">
              <button
                disabled={actionLoading}
                onClick={async () => {
                  if (!confirm("Import sẽ ghi đè dữ liệu hiện tại theo cơ chế UPSERT. Tiếp tục?")) return;
                  const ok = await exec(`${BACKEND_URL}/api/admin/data/import`, "POST", {});
                  if (ok) addToast("Import dữ liệu từ CSV thành công!", "success");
                }}
                className="flex items-center gap-2 px-5 py-3 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer"
              >
                <Database className="w-4 h-4" />
                {actionLoading ? "Đang xử lý..." : "Import CSV → Database"}
              </button>
              <button
                disabled={actionLoading}
                onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/data/export`, "POST", {});
                  if (ok) addToast("Export dữ liệu ra CSV thành công!", "success");
                }}
                className="flex items-center gap-2 px-5 py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer"
              >
                <RefreshCw className="w-4 h-4" />
                {actionLoading ? "Đang xử lý..." : "Export Database → CSV"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Audit Logs section */}
      {section === "logs" && (
        <div className="space-y-4">
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
            <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-3">
              <BookOpen className="w-4 h-4 text-amber-400" />
              Nhật Ký Hệ Thống
            </h3>
            {/* Filters */}
            <div className="flex flex-wrap gap-3 mb-4">
              <div className="flex-1 min-w-[120px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Hành động</label>
                <input value={auditLogFilterAction} onChange={e => setAuditLogFilterAction(e.target.value)} placeholder="Lọc action..." className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex-1 min-w-[120px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Player ID</label>
                <input value={auditLogFilterPlayer} onChange={e => setAuditLogFilterPlayer(e.target.value)} placeholder="UUID..." className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="w-[100px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Kết quả</label>
                <select value={auditLogFilterSuccess} onChange={e => setAuditLogFilterSuccess(e.target.value)} className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50">
                  <option value="">Tất cả</option>
                  <option value="true">Thành công</option>
                  <option value="false">Thất bại</option>
                </select>
              </div>
              <div className="flex items-end">
                <button onClick={fetchAuditLogs} disabled={auditLogLoading} className="px-3 py-1.5 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold cursor-pointer">
                  {auditLogLoading ? "..." : "Tìm"}
                </button>
              </div>
            </div>
            {/* Log list */}
            {auditLogLoading ? (
              <div className="flex justify-center py-10">
                <RefreshCw className="w-6 h-6 text-amber-400 animate-spin" />
              </div>
            ) : auditLogs.length === 0 ? (
              <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                <BookOpen className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                <p className="text-xs text-zinc-500">Không có nhật ký nào.</p>
              </div>
            ) : (
              <>
                <div className="space-y-1 max-h-[500px] overflow-y-auto">
                  {auditLogs.map((log: any) => (
                    <div key={log.id}>
                      <div onClick={() => setAuditLogExpandId(auditLogExpandId === log.id ? null : log.id)} className="flex items-center gap-3 px-3 py-2 bg-zinc-800/40 hover:bg-zinc-800/60 rounded-lg cursor-pointer transition-colors">
                        <div className={`w-2 h-2 rounded-full shrink-0 ${log.success ? "bg-emerald-500" : "bg-red-500"}`} />
                        <span className="text-xs font-mono text-zinc-500 w-[160px] shrink-0">{log.created_at}</span>
                        <span className="text-xs font-semibold text-amber-400 w-[160px] shrink-0">{log.action}</span>
                        <span className="text-xs text-zinc-400 truncate flex-1">{log.request_summary}</span>
                        <span className="text-[10px] text-zinc-600 w-[36px] text-right">{log.player_id ? log.player_id.substring(0, 8) : "—"}</span>
                      </div>
                      {auditLogExpandId === log.id && (
                        <div className="mx-3 mb-1 px-4 py-2 bg-zinc-900/60 border border-zinc-800/60 rounded-lg text-[11px] text-zinc-400 font-mono space-y-1">
                          {log.player_id && <div>player_id: {log.player_id}</div>}
                          {log.entity_type && <div>entity_type: {log.entity_type}</div>}
                          {log.entity_id && <div>entity_id: {log.entity_id}</div>}
                          {log.error_message && <div className="text-red-400">error: {log.error_message}</div>}
                          {log.ip_address && <div>ip: {log.ip_address}</div>}
                          {log.metadata && Object.keys(log.metadata).length > 0 && (
                            <div>metadata: {JSON.stringify(log.metadata, null, 1)}</div>
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
                {/* Pagination */}
                <div className="flex items-center justify-between mt-4 pt-3 border-t border-zinc-800/60">
                  <span className="text-[11px] text-zinc-500 font-mono">{auditLogTotal} bản ghi</span>
                  <div className="flex gap-2">
                    <button disabled={auditLogPage <= 0} onClick={() => setAuditLogPage(p => Math.max(0, p - 1))} className="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 text-zinc-300 rounded-lg text-[11px] font-semibold cursor-pointer">Trước</button>
                    <button disabled={(auditLogPage + 1) * auditLogLimit >= auditLogTotal} onClick={() => setAuditLogPage(p => p + 1)} className="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 text-zinc-300 rounded-lg text-[11px] font-semibold cursor-pointer">Sau</button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}
      {/* Shop section */}
      {section === "shop" && (
        <div className="space-y-4">
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                <Package className="w-4 h-4 text-emerald-400" />
                Quản Lý Cửa Hàng
              </h3>
              <button onClick={() => { setShopForm({ id: "", price: 100, stock: -1, is_default: true, sort_order: 0 }); setShowShopAdd(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer">
                <Plus className="w-3.5 h-3.5" />Thêm
              </button>
            </div>
            {shopLoading ? (
              <div className="flex justify-center py-10"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
            ) : shopItems.length === 0 ? (
              <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                <Package className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                <p className="text-xs text-zinc-500">Chưa có mặt hàng nào.</p>
              </div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold">ID</th>
                      <th className="px-3 py-2.5 font-semibold">Tên</th>
                      <th className="px-3 py-2.5 font-semibold">Loại</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Giá</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Tồn kho</th>
                      <th className="px-3 py-2.5 font-semibold text-center">Mặc định</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Thứ tự</th>
                      <th className="px-3 py-2.5"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {shopItems.map((item: any) => (
                      <tr key={item.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono">{item.id}</td>
                        <td className="px-3 py-2.5 font-semibold text-zinc-200">{item.name || item.id}</td>
                        <td className="px-3 py-2.5 text-zinc-400">{item.item_type || "—"}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{item.price?.toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{item.stock === -1 ? "∞" : item.stock}</td>
                        <td className="px-3 py-2.5 text-center">{item.is_default ? <span className="text-emerald-400">✓</span> : <span className="text-zinc-600">–</span>}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-400">{item.sort_order}</td>
                        <td className="px-3 py-2.5 text-right">
                          <button onClick={() => { setShopForm({ id: item.id, price: item.price, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowShopEdit(item.id); }} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                          <button onClick={async () => { if (!confirm(`Xóa ${item.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/shop/${item.id}`, "DELETE"); if (ok) fetchShopItems(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
      {/* Add Shop Item Modal */}
      {showShopAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowShopAdd(false)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Thêm mặt hàng</h3>
            <div className="space-y-3">
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID (item_dictionary / gu_dictionary)</label>
                <input value={shopForm.id} onChange={e => setShopForm(f => ({ ...f, id: e.target.value }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá</label>
                  <input type="number" value={shopForm.price} onChange={e => setShopForm(f => ({ ...f, price: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tồn kho (-1: vô hạn)</label>
                  <input type="number" value={shopForm.stock} onChange={e => setShopForm(f => ({ ...f, stock: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <div className="flex gap-3 items-end">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Thứ tự</label>
                  <input type="number" value={shopForm.sort_order} onChange={e => setShopForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <label className="flex items-center gap-2 pb-2">
                  <input type="checkbox" checked={shopForm.is_default} onChange={e => setShopForm(f => ({ ...f, is_default: e.target.checked }))} className="rounded bg-zinc-800 border-zinc-700" />
                  <span className="text-[11px] text-zinc-400">Mặc định</span>
                </label>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowShopAdd(false)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const ok = await exec(`${BACKEND_URL}/api/admin/shop`, "POST", shopForm);
                if (ok) { setShowShopAdd(false); fetchShopItems(); }
              }} className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Thêm</button>
            </div>
          </div>
        </div>
      )}
      {/* Edit Shop Item Modal */}
      {showShopEdit && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowShopEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Sửa mặt hàng: {shopForm.id}</h3>
            <div className="space-y-3">
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá</label>
                  <input type="number" value={shopForm.price} onChange={e => setShopForm(f => ({ ...f, price: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tồn kho</label>
                  <input type="number" value={shopForm.stock} onChange={e => setShopForm(f => ({ ...f, stock: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <div className="flex gap-3 items-end">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Thứ tự</label>
                  <input type="number" value={shopForm.sort_order} onChange={e => setShopForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <label className="flex items-center gap-2 pb-2">
                  <input type="checkbox" checked={shopForm.is_default} onChange={e => setShopForm(f => ({ ...f, is_default: e.target.checked }))} className="rounded bg-zinc-800 border-zinc-700" />
                  <span className="text-[11px] text-zinc-400">Mặc định</span>
                </label>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowShopEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const ok = await exec(`${BACKEND_URL}/api/admin/shop/${showShopEdit}`, "PUT", shopForm);
                if (ok) { setShowShopEdit(null); fetchShopItems(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}
      {/* Mines section */}
      {section === "mines" && (
        <div className="space-y-4">
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
            <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-4">
              <Layers className="w-4 h-4 text-amber-400" />
              Quản Lý Mỏ Linh Thạch
            </h3>
            {mineLoading ? (
              <div className="flex justify-center py-10"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
            ) : mineSlots.length === 0 ? (
              <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                <Layers className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                <p className="text-xs text-zinc-500">Không có mỏ nào.</p>
              </div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold">#</th>
                      <th className="px-3 py-2.5 font-semibold">Phẩm chất</th>
                      <th className="px-3 py-2.5 font-semibold">Chủ sở hữu</th>
                      <th className="px-3 py-2.5 font-semibold">Quái canh</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Cấp Q</th>
                      <th className="px-3 py-2.5 font-semibold">Chiếm giữ</th>
                      <th className="px-3 py-2.5 font-semibold"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {mineSlots.map((slot: any) => (
                      <tr key={slot.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono">{slot.id}</td>
                        <td className="px-3 py-2.5 font-semibold">{slot.rarity}</td>
                        <td className="px-3 py-2.5 text-zinc-300">{slot.occupier_name || <span className="text-zinc-600 italic">Trống</span>}</td>
                        <td className="px-3 py-2.5 text-zinc-400">{slot.monster_id || <span className="text-zinc-600">—</span>}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{slot.monster_rank > 0 ? `${slot.monster_rank}.${slot.monster_sub_rank}` : "—"}</td>
                        <td className="px-3 py-2.5 text-[10px] text-zinc-500">{slot.occupied_at ? new Date(slot.occupied_at).toLocaleDateString("vi-VN") : "—"}</td>
                        <td className="px-3 py-2.5 text-right whitespace-nowrap">
                          <button onClick={() => { setMineEditRarity(slot.rarity); setShowMineEdit(slot.id); }} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                          <button onClick={async () => {
                            if (!confirm(`Reset mỏ #${slot.id}?`)) return;
                            const ok = await exec(`${BACKEND_URL}/api/admin/mine-slots/${slot.id}/reset`, "POST");
                            if (ok) fetchMineSlots();
                          }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Reset"><RefreshCw className="w-3.5 h-3.5 inline" /></button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
      {/* Edit Mine Slot Modal */}
      {showMineEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowMineEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-sm mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Sửa mỏ #{showMineEdit}</h3>
            <div className="space-y-3">
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Phẩm chất</label>
                <select value={mineEditRarity} onChange={e => setMineEditRarity(e.target.value)} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50">
                  <option value="Thường">Thường</option>
                  <option value="Hiếm">Hiếm</option>
                  <option value="Sử Thi">Sử Thi</option>
                  <option value="Truyền Thuyết">Truyền Thuyết</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowMineEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const ok = await exec(`${BACKEND_URL}/api/admin/mine-slots/${showMineEdit}`, "PUT", { rarity: mineEditRarity });
                if (ok) { setShowMineEdit(null); fetchMineSlots(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}
      {/* Sects section */}
      {section === "sects" && (
        <>
          <div className="flex items-center justify-end mb-4">
            <button onClick={fetchSects} disabled={sectsLoading} className="flex items-center gap-1.5 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50 cursor-pointer">
              <RefreshCw className={`w-3.5 h-3.5 ${sectsLoading ? "animate-spin" : ""}`} /> Làm mới
            </button>
          </div>

          {sectsLoading && sects.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 space-y-3">
              <RefreshCw className="w-8 h-8 text-amber-400 animate-spin" />
              <span className="text-xs text-zinc-500 font-mono">Đang tải danh sách tông môn...</span>
            </div>
          ) : sects.length === 0 ? (
            <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
              <Building2 className="w-10 h-10 text-zinc-700 mx-auto" />
              <p className="text-sm text-zinc-400 font-medium">Không có tông môn nào.</p>
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                    <th className="px-3 py-2.5 font-semibold">ID</th>
                    <th className="px-3 py-2.5 font-semibold">Tên</th>
                    <th className="px-3 py-2.5 font-semibold">Khu vực</th>
                    <th className="px-3 py-2.5 font-semibold">Cấp</th>
                    <th className="px-3 py-2.5 font-semibold text-right">Quỹ</th>
                    <th className="px-3 py-2.5 font-semibold text-right">Thành viên</th>
                    <th className="px-3 py-2.5 font-semibold">Chưởng môn</th>
                    <th className="px-3 py-2.5 font-semibold">Phê duyệt</th>
                  </tr>
                </thead>
                <tbody>
                  {sects.map((s: any) => (
                    <tr key={s.id} onClick={() => { setSelectedSectId(s.id); }} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors cursor-pointer">
                      <td className="px-3 py-2.5 text-zinc-500 font-mono">{s.id}</td>
                      <td className="px-3 py-2.5 font-semibold text-zinc-200">{s.name}</td>
                      <td className="px-3 py-2.5 text-zinc-400">{s.region}</td>
                      <td className="px-3 py-2.5 font-bold text-zinc-300">{s.level}</td>
                      <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{s.funds?.toLocaleString() || 0}</td>
                      <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{s.member_count}</td>
                      <td className="px-3 py-2.5 text-zinc-300">{s.leader_name || <span className="text-zinc-600 italic">Trống</span>}</td>
                      <td className="px-3 py-2.5">{s.require_approval ? <span className="text-amber-400 text-[10px]">✓</span> : <span className="text-zinc-600">–</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
};
