import React, { useEffect, useState, useCallback } from "react";
import { authFetch } from "@/lib/auth";
import { useGameStore, guSpecFor } from "@/store/useGameStore";
import type { GuSpec, ItemSpec } from "@/store/useGameStore";
import { reqGuQty, reqGuRank } from "@/lib/gu-dict";
import type { RequiredGuItem } from "@/lib/gu-dict";
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
  Eye,
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
  List,
  BookOpen,
  Database,
  FlaskConical,
  Activity,
  Gift,
  Ticket,
  Check,
  Clock,
  Copy,
  Upload,
  PawPrint,
  ShoppingBag,
  Swords,
  MapPin,
  History as HistoryIcon,
  ChevronDown,
  User,
  TrendingUp,
  Leaf,
} from "lucide-react";
import { translateGuType, translateItemType, translatePath, translateMastery, masteryKeyForPoints } from "@/lib/translate";
import { EffectEnvelopeView } from "@/components/EffectBlockView";
import { statusRemainLabel, formatNumber } from "@/lib/utils";
import { Num } from "./Num";
import { TopupWalletAdminTab } from "./admin/TopupWalletAdminTab";
import { DictionariesAdminTab } from "./admin/DictionariesAdminTab";
import { WorldExtendedAdminTab } from "./admin/WorldExtendedAdminTab";
import { SectMinesAdminTab } from "./admin/SectMinesAdminTab";
import { ApertureChildrenAdmin } from "./admin/ApertureChildrenAdmin";
import { ApertureRefinementQueueAdmin } from "./admin/ApertureRefinementQueueAdmin";
import { ApertureSettlementsAdmin } from "./admin/ApertureSettlementsAdmin";
import { ApertureEventsAdmin } from "./admin/ApertureEventsAdmin";
import { ApertureNPCAdmin } from "./admin/ApertureNPCAdmin";
import { ApertureTribulationsAdmin } from "./admin/ApertureTribulationsAdmin";
import { SectDefendersAdmin } from "./admin/SectDefendersAdmin";
import { SectRequestsAdmin } from "./admin/SectRequestsAdmin";
import { GuHistoryAdminTab } from "./admin/GuHistoryAdminTab";
import { AuditLogsAdminTab } from "./admin/modules/AuditLogsAdminTab";
import { MonitoringAdminTab } from "./admin/modules/MonitoringAdminTab";
import { CoreDataAdminTab } from "./admin/CoreDataAdminTab";
import { AdminDataGrid } from "./admin/AdminDataGrid";
import { useAdminUrlState } from "./admin/useAdminUrlState";
import { GuEffectsEditor, validateAllEffectEditors } from "./admin/GuEffectsEditor";
import type { AdminModuleKey } from "./admin/navigation";
import { useAdminAccess } from "./admin/AdminAccessContext";
import { canPerformAdminAction, prepareAdminAction } from "./admin/actions";
import { DataImportWorkflow } from "./admin/DataImportWorkflow";
import { PlayerRolesEditor } from "./admin/PlayerRolesEditor";
import { CombatHistoryAdminTab } from "./admin/CombatHistoryAdminTab";
import { MapDropsAdminTab } from "./admin/MapDropsAdminTab";
import { ArenaRankingsAdminTab } from "./admin/ArenaRankingsAdminTab";
import { WorldBossesAdminTab } from "./admin/WorldBossesAdminTab";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const PLAYER_QUERY_DEFAULTS = { page: "1", page_size: "50", search: "", search_by: "all", sort: "created_at", direction: "desc" };
const PLAYER_QUERY_KEYS = Object.keys(PLAYER_QUERY_DEFAULTS);

type DatasetDescriptor = {
  key: string;
  label: string;
  csv_file: string;
  capabilities: { import: boolean; export: boolean; preflight: boolean };
  cache_invalidation: string[];
  registry_reload: string[];
  permissions: { read: string; import?: string; export?: string };
  ui_supported: boolean;
};

const EFFECT_SUGGESTIONS: Record<string, { key: string; label: string; type: "string" | "number" | "boolean" }[]> = {
  active: [
    { key: "damage_flat", label: "Damage Flat (Sát thương cố định)", type: "number" },
    { key: "damage_mult", label: "Damage Mult (Hệ số sát thương)", type: "number" },
    { key: "damage_path", label: "Damage Path (Hệ sát thương)", type: "string" },
    { key: "heal_flat", label: "Heal Flat (Hồi máu)", type: "number" },
    { key: "shield_flat", label: "Shield Flat (Hộ thuẫn)", type: "number" },
    { key: "mana_regen", label: "Mana Regen (Hồi mana)", type: "number" },
    { key: "lifesteal_rate", label: "Life Steal Rate (Tỉ lệ hút máu)", type: "number" },
    { key: "apply_status", label: "Apply Status — mảng [{\"id\":\"BLEED\",\"value\":10,\"turns\":3}] hoặc chuỗi (Áp dụng trạng thái)", type: "string" },
    { key: "status_tick_damage", label: "Status Tick Dmg (ST trạng thái/tick)", type: "number" },
    { key: "status_duration_ticks", label: "Status Duration Ticks (Thời gian trạng thái)", type: "number" },
    { key: "refine_mana_tier", label: "Refine Mana Tier", type: "number" },
    { key: "cleansing_power", label: "Cleansing Power", type: "number" },
    { key: "range", label: "Range (Tầm đánh)", type: "number" },
    { key: "target_selector", label: "Target Selector (Chọn mục tiêu)", type: "string" },
    { key: "target_count", label: "Target Count (Số lượng mục tiêu)", type: "number" },
    { key: "summon_monster_id", label: "Summon Monster ID (ID Quái triệu hồi)", type: "string" },
    { key: "summon_position", label: "Summon Position (Vị trí triệu hồi)", type: "string" },
    { key: "inherit_stats_percent", label: "Inherit Stats % (Thừa kế thuộc tính %)", type: "number" },
    { key: "phantom_damage_scale", label: "Phantom Damage Scale", type: "number" },
    { key: "summon_count_per_tick", label: "Summon Count/Tick", type: "number" },
    { key: "summon_phantom", label: "Summon Phantom (Triệu hồi ảo ảnh)", type: "boolean" },
    { key: "generate_gu", label: "Generate Gu (Tạo cổ trùng)", type: "boolean" },
    { key: "blood_drain", label: "Blood Drain (Hút máu)", type: "boolean" },
    { key: "wood_transformation", label: "Wood Transformation (Biến đổi gỗ)", type: "boolean" },
    { key: "curve_attack", label: "Curve Attack (Tấn công cong)", type: "boolean" },
    { key: "replicate", label: "Replicate (Nhân bản)", type: "boolean" },
    { key: "summon_growth_phantoms", label: "Summon Growth Phantoms", type: "boolean" },
    { key: "damage_hp_percent", label: "% MaxHP trực tiếp (0..1)", type: "number" },
    { key: "shield_flat_reduce_percent", label: "Tước % khiên (0..1)", type: "number" },
    { key: "disable_gu_next_turn", label: "Phong ấn Cổ lượt sau", type: "boolean" },
    { key: "forget_reality_chance", label: "Tỉ lệ proc bonus (0..1)", type: "number" },
    { key: "target_damage_reduce_percent", label: "Giảm ST địch (0..1)", type: "number" }
  ],
  passive: [
    { key: "stat_base_attack", label: "Base Attack (Tấn công cơ bản)", type: "number" },
    { key: "stat_base_defense", label: "Base Defense (Phòng thủ cơ bản)", type: "number" },
    { key: "stat_base_hp", label: "Base HP (HP cơ bản)", type: "number" },
    { key: "stat_base_speed", label: "Base Speed (Tốc độ cơ bản)", type: "number" },
    { key: "mana_regen_mult", label: "Mana Regen Mult (Hệ số hồi mana)", type: "number" },
    { key: "soul_refinement", label: "Soul Refinement", type: "number" },
    { key: "movement", label: "Movement (Phạm vi di chuyển, ô)", type: "number" },
    { key: "storage_capacity", label: "Storage (Dung lượng chứa)", type: "number" },
    { key: "investigation", label: "Investigation (Khảo sát/Điều tra)", type: "number" },
    { key: "hearing", label: "Hearing (Thính giác)", type: "number" },
    { key: "path_conflict_reduction", label: "Path Conflict Reduction (%)", type: "number" },
    { key: "damage_multiplier_on_hit", label: "Damage Mult On Hit (Hệ số ST khi đánh)", type: "number" },
    { key: "refine_mana_tier", label: "Refine Mana Tier", type: "number" },
    { key: "cleansing_power", label: "Cleansing Power", type: "number" },
    { key: "luck", label: "Luck (May mắn)", type: "number" },
    { key: "anti_heaven_damage_reduction_percent", label: "Kháng Thiên đạo (0..1)", type: "number" },
    { key: "anti_earth_damage_reduction_percent", label: "Kháng Thổ đạo (0..1)", type: "number" },
    { key: "anti_dream_damage_reduction_percent", label: "Kháng Mộng đạo (0..1)", type: "number" },
    { key: "immune_dream_status", label: "Miễn nhiễm Mộng đạo", type: "boolean" },
    { key: "forget_heaven_protection", label: "Lãng Quên Thiên (cờ)", type: "boolean" },
    { key: "forget_earth_protection", label: "Lãng Quên Địa (cờ)", type: "boolean" }
  ],
  growth: [
    { key: "max_attack", label: "Max Attack (Tấn công tối đa)", type: "number" },
    { key: "max_defense", label: "Max Defense (Phòng thủ tối đa)", type: "number" },
    { key: "max_hp", label: "Max HP (HP tối đa)", type: "number" },
    { key: "max_speed", label: "Max Speed (Tốc độ tối đa)", type: "number" },
    { key: "hours_to_max", label: "Hours To Max (Số giờ đạt tối đa)", type: "number" }
  ]
};

const getEffectFieldType = (cat: string, key: string): "string" | "number" | "boolean" => {
  const suggestions = EFFECT_SUGGESTIONS[cat] || [];
  const found = suggestions.find(s => s.key === key);
  if (found) return found.type;
  for (const c of Object.keys(EFFECT_SUGGESTIONS)) {
    const f = EFFECT_SUGGESTIONS[c].find(s => s.key === key);
    if (f) return f.type;
  }
  return "number";
};

interface PlayerRow {
  id: string; username: string; rank: number; sub_rank: number;
  current_mana: number; max_mana: number; hp: number; base_hp: number;
  primeval_stones: number; immortal_stones: number;
  is_admin: boolean; created_at: string; current_region: string;
  is_banned?: boolean;
  aptitude_type: string; wall_progress: number; human_qi: number;
  lifespan: number;
}

interface EquippedGuDetail {
  instance_id: string;
  gu_id: string;
  name: string;
  rank: number;
  hunger: number;
  status: string;
  is_vital: boolean;
  growth_hours?: number;
  path?: string[];
  description?: string;
}

interface PassiveStatSource {
  source: string;
  name: string;
  value: number;
  mult: number;
  dao: number;
  final: number;
}

interface PassiveStatDetail {
  key: string;
  total: number;
  sources: PassiveStatSource[];
}

interface PassiveStatsResult {
  stats: Record<string, number>;
  details: PassiveStatDetail[];
}

interface GrowthBoosts {
  attack: number;
  defense: number;
  hp: number;
  speed: number;
}

interface FullyGrownInfo {
  gu_id: string;
  rank: number;
  attack: number;
  defense: number;
  hp: number;
  speed: number;
}

interface GrowthData {
  boosts?: GrowthBoosts;
  fully_grown?: FullyGrownInfo[];
}

interface PlayerDetail {
  id: string; username: string; rank: number; sub_rank: number;
  aptitude_type: string; aptitude_percent: number; aptitude_grade: string;
  current_mana: number; max_mana: number; primeval_stones: number;
  immortal_stones: number; hp: number; base_hp: number;
  wall_progress: number; human_qi: number; current_region: string;
  x?: number; y?: number;
  is_admin: boolean; created_at: string; lifespan: number;
  soul_hp: number; max_soul: number; soul_tier: string;
  explore_count: number; aperture_stress: number;
  avatar_url: string; bio: string;
  race: string; gender: string; constitution: string; statuses?: (string | { id: string; expires_at?: string | null })[];
  current_thought?: number; max_thought?: number; thought_regen_rate?: number;
  time_speed_multiplier?: number; selected_title?: string; selected_frame?: string;
  fake_rank?: number; fake_sub_rank?: number; fake_vital_gu?: string;
  is_disguised?: boolean; needs_onboarding?: boolean; needs_soul_refine?: boolean;
  immortal_jade?: number;
  is_banned?: boolean;
  ban_reason?: string;
  banned_until?: string | null;
  is_chat_muted?: boolean;
  chat_muted_until?: string | null;
  gu_count: number;
  item_count: number;
  growth_data?: GrowthData;
  path_masteries?: Record<string, number>;
  passive_stats?: PassiveStatsResult;
  equipped_gus?: EquippedGuDetail[];
}

interface PlayerItemRow {
  item_id: string;
  name: string;
  item_type: string;
  rank: number;
  category: string;
  quantity: number;
  completion_rate?: number;
  is_unlocked?: boolean;
  deduce_count?: number;
  last_deduce_at?: string | null;
  favorite?: boolean;
  created_at?: string;
  updated_at?: string;
}

interface PlayerQuestRow {
  quest_id: string;
  status: string;
  progress_data: unknown;
  completed_at?: string | null;
  created_at?: string;
  updated_at?: string;
}

interface PlayerInstanceSessionRow {
  id: string;
  instance_id: string;
  current_stage: number;
  status: "ACTIVE" | "COMPLETED" | "FAILED";
  temp_inventory: unknown;
  created_at: string;
}

interface PlayerSectMissionRow {
  id: string; mission_id: string; title: string; sect_id: string; sect_name: string;
  mission_type: string; target_id: string; target_amount: number; reward_contribution: number;
  current_progress: number; status: string; assigned_at: string; expires_at?: string | null;
}

interface PlayerTrackingMarkRow {
  id: string; tracker_id: string; tracker_username: string; target_id: string; target_username: string;
  mark_type: string; power: number; expires_at: string; created_at: string; is_tracker: boolean;
}

interface PlayerAchievementRow { player_id:string; achievement_id:string; selected_title:string; selected_frame:string; unlocked_at:string; }



interface PlayerGuRow {
  instance_id: string;
  gu_id: string;
  name: string;
  rank: number;
  hunger: number;
  status: string;
  equipped: boolean;
  is_vital: boolean;
  gu_rank: number;
  level: number;
  location: string;
  is_locked: boolean;
  integration_time: number;
  equipped_at?: string | null;
  growth_hours: number;
  created_at?: string;
  updated_at?: string;
  last_fed_at?: string | null;
}

const raceLabels: Record<string, string> = {
  human: "Nhân Tộc", cuong_thi: "Cương Thi", fur_people: "Người Lông", merman: "Nhân Ngư",
  feathered: "Vũ Dân", inkman: "Mặc Nhân", rockman: "Thạch Nhân",
  snowman: "Tuyết Dân", variant_beastman: "Thú Nhân", dragonman: "Long Nhân",
  giant_sun_descendant: "Cự Dương Huyết Mạch",
};
const genderLabels: Record<string, string> = { male: "Nam", female: "Nữ" };
const constitutionLabels: Record<string, string> = {
  solar_sun_moon_body: "Thái Nhật Dương Mãng Thể",
  ancient_moon_yin_desolate_body: "Cổ Nguyệt Âm Hoang Thể",
  northern_dark_ice_soul_body: "Bắc Minh Băng Phách Thể",
  forest_sea_reincarnation_body: "Sâm Hải Luân Hồi Thể",
  myriad_gold_brilliant_flower_body: "Vạn Kim Diệu Hoa Thể",
  blazing_glory_lightning_brilliance_body: "Viêm Hoàng Lôi Trạch Thể",
  great_strength_true_martial_body: "Đại Lực Chân Võ Thể",
  carefree_wisdom_heart_body: "Tiêu Dao Trí Tâm Thể",
  great_earth_origin_body: "Hậu Thổ Nguyên Ương Thể",
  universe_great_evolution_body: "Vũ Trụ Đại Diễn Thể",
  pure_dream_reality_seeker_body: "Thuần Mộng Cầu Chân Thể",
};
const statusLabels: Record<string, string> = {
  severely_wounded: "Trọng Thương",
  poisoned: "Trúng Độc",
  exhausted: "Kiệt Sức",
  blessed: "Phúc Trạch",
  death_penalty: "Hình Phạt Tử Vong",
  regional_wall_suppression: "Áp Chế Giới Vực",
};

function formatStat(val: number): string {
  if (Math.abs(val) >= 10000) return val.toLocaleString();
  return val % 1 === 0 ? val.toString() : val.toFixed(2);
}

const rankNames = [
  "Nhất Chuyển", "Nhị Chuyển", "Tam Chuyển", "Tứ Chuyển",
  "Ngũ Chuyển", "Lục Chuyển Cổ Tiên", "Thất Chuyển Cổ Tiên", "Bát Chuyển Cổ Tiên", "Cửu Chuyển Cổ Tôn", "Vĩnh Sinh",
];
const rankSub = ["Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"];

function getRankName(rank: number, sub: number): string {
  const r = rankNames[rank - 1] || `${rank} Chuyển`;
  if (rank >= 6) return r;
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
    case "HO_PHAP": return "text-orange-400 border-orange-500/30 bg-orange-950/60";
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
    case "HO_PHAP": return "HỘ PHÁP";
    case "TRUONG_LAO": return "TRƯỞNG LÃO";
    case "CHAN_TRUYEN": return "CHÂN TRUYỀN";
    case "TINH_ANH": return "TINH ANH";
    case "NOI_MON": return "NỘI MÔN";
    default: return "NGOẠI MÔN";
  }
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function guLocationStyle(loc: string): string {
  switch (loc) {
    case "inventory": return "text-emerald-400 border-emerald-500/30 bg-emerald-950/60";
    case "auction": return "text-amber-400 border-amber-500/30 bg-amber-950/60";
    case "shop": return "text-blue-400 border-blue-500/30 bg-blue-950/60";
    default: return "text-zinc-400 border-zinc-700 bg-zinc-900";
  }
}

function guLocationLabel(loc: string): string {
  switch (loc) {
    case "inventory": return "Túi đồ";
    case "auction": return "Đấu giá";
    case "shop": return "Cửa hàng";
    default: return loc;
  }
}

export const AdminView: React.FC<{ authorized?: boolean; embedded?: boolean; initialSection?: AdminModuleKey }> = ({ authorized = false, embedded = false, initialSection = "players" }) => {
  const adminAccess = useAdminAccess();
  const canAdminAction = (action: Parameters<typeof canPerformAdminAction>[1]) => canPerformAdminAction(adminAccess.effective_permissions, action);
  const { player, setView, addToast, guDictionary, itemDictionary, constitutionDictionary, raceDictionary, statusDictionary } = useGameStore();
  const [activeTab, setActiveTab] = useState<"items" | "gus" | "pets" | "quests" | "instances" | "sect_missions" | "tracking_marks" | "achievements" | "combat">("items");
  const [isCombatEquippedExpanded, setIsCombatEquippedExpanded] = useState(true);
  const [isCombatStatsExpanded, setIsCombatStatsExpanded] = useState(true);
  const [isCombatGrowthExpanded, setIsCombatGrowthExpanded] = useState(true);
  const [isCombatIdentityExpanded, setIsCombatIdentityExpanded] = useState(true);
  const [isCombatPathExpanded, setIsCombatPathExpanded] = useState(true);
  const [players, setPlayers] = useState<PlayerRow[]>([]);
  const [total, setTotal] = useState(0);
  const [playerQuery, updatePlayerQuery] = useAdminUrlState(PLAYER_QUERY_DEFAULTS, PLAYER_QUERY_KEYS);
  const page = Math.max(1, Number(playerQuery.page) || 1);
  const limit = Math.max(1, Number(playerQuery.page_size) || 50);
  const search = playerQuery.search;
  const [searchText, setSearchText] = useState(playerQuery.search);
  const searchBy = playerQuery.search_by as "all" | "id" | "username" | "email" | "item" | "gu";
  const setPage = (next: number) => updatePlayerQuery({ page: String(next) });
  useEffect(() => setSearchText(playerQuery.search), [playerQuery.search]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<PlayerDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // Lazy-loaded player detail tabs
  const [playerItems, setPlayerItems] = useState<PlayerItemRow[]>([]);
  const [playerItemsTotal, setPlayerItemsTotal] = useState(0);
  const [playerItemsPage, setPlayerItemsPage] = useState(1);
  const [playerItemsLimit] = useState(20);
  const [playerItemsLoading, setPlayerItemsLoading] = useState(false);
  const [playerItemsSearch, setPlayerItemsSearch] = useState("");

  const [playerGus, setPlayerGus] = useState<PlayerGuRow[]>([]);
  const [playerGusTotal, setPlayerGusTotal] = useState(0);
  const [playerGusPage, setPlayerGusPage] = useState(1);
  const [playerGusLimit] = useState(20);
  const [playerGusLoading, setPlayerGusLoading] = useState(false);
  const [playerGusSearch, setPlayerGusSearch] = useState("");

  const [playerPets, setPlayerPets] = useState<ApiResponseData[]>([]);
  const [playerPetsTotal, setPlayerPetsTotal] = useState(0);
  const [playerPetsPage, setPlayerPetsPage] = useState(1);
  const [playerPetsLimit] = useState(20);
  const [playerPetsLoading, setPlayerPetsLoading] = useState(false);
  const [playerPetsSearch, setPlayerPetsSearch] = useState("");
  const [playerQuests, setPlayerQuests] = useState<PlayerQuestRow[]>([]);
  const [playerQuestsLoading, setPlayerQuestsLoading] = useState(false);
  const [showQuestEditor, setShowQuestEditor] = useState<PlayerQuestRow | null>(null);
  const [questStatus, setQuestStatus] = useState("ACTIVE");
  const [questProgressData, setQuestProgressData] = useState("{}");
  const [questCompletedAt, setQuestCompletedAt] = useState("");
  const [instanceSessions, setInstanceSessions] = useState<PlayerInstanceSessionRow[]>([]);
  const [instanceSessionsLoading, setInstanceSessionsLoading] = useState(false);
  const [showInstanceSessionEditor, setShowInstanceSessionEditor] = useState<PlayerInstanceSessionRow | null>(null);
  const [instanceSessionStage, setInstanceSessionStage] = useState("0");
  const [instanceSessionStatus, setInstanceSessionStatus] = useState("ACTIVE");
  const [instanceSessionInventory, setInstanceSessionInventory] = useState("[]");
  const [playerSectMissions, setPlayerSectMissions] = useState<PlayerSectMissionRow[]>([]);
  const [playerSectMissionsLoading, setPlayerSectMissionsLoading] = useState(false);
  const [showPlayerSectMissionEditor, setShowPlayerSectMissionEditor] = useState<PlayerSectMissionRow | null>(null);
  const [playerSectMissionProgress, setPlayerSectMissionProgress] = useState("0");
  const [playerSectMissionStatus, setPlayerSectMissionStatus] = useState("IN_PROGRESS");
  const [playerSectMissionExpiry, setPlayerSectMissionExpiry] = useState("");
  const [playerTrackingMarks, setPlayerTrackingMarks] = useState<PlayerTrackingMarkRow[]>([]);
  const [playerTrackingMarksLoading, setPlayerTrackingMarksLoading] = useState(false);
  const [showTrackingMarkEditor, setShowTrackingMarkEditor] = useState<PlayerTrackingMarkRow | null>(null);
  const [trackingMarkType, setTrackingMarkType] = useState("wisdom");
  const [trackingMarkPower, setTrackingMarkPower] = useState("0");
  const [trackingMarkExpiry, setTrackingMarkExpiry] = useState("");
  const [playerAchievements, setPlayerAchievements] = useState<PlayerAchievementRow[]>([]);
  const [availableAchievements, setAvailableAchievements] = useState<ApiResponseData[]>([]);
  const [playerAchievementsLoading, setPlayerAchievementsLoading] = useState(false);
  const [showAchievementEditor, setShowAchievementEditor] = useState<PlayerAchievementRow | null>(null);
  const [achievementTitle, setAchievementTitle] = useState("");
  const [achievementFrame, setAchievementFrame] = useState("");
  const [achievementUnlockedAt, setAchievementUnlockedAt] = useState("");
  const [achievementToGrant, setAchievementToGrant] = useState("");

  // Edit player modal
  const [showEditPlayer, setShowEditPlayer] = useState(false);
  const [showRolesEditor, setShowRolesEditor] = useState(false);
  const [editPlayerForm, setEditPlayerForm] = useState<Record<string, string>>({});

  const [showAddItem, setShowAddItem] = useState(false);
  const [addItemId, setAddItemId] = useState("");
  const [addItemQty, setAddItemQty] = useState(1);
  const [addItemCategory, setAddItemCategory] = useState("materials");
  const [showRecipeEditor, setShowRecipeEditor] = useState<PlayerItemRow | null>(null);
  const [recipeCompletion, setRecipeCompletion] = useState("0");
  const [recipeUnlocked, setRecipeUnlocked] = useState(false);
  const [recipeDeduceCount, setRecipeDeduceCount] = useState("0");
  const [recipeFavorite, setRecipeFavorite] = useState(false);
  const [showInventoryItemEditor, setShowInventoryItemEditor] = useState<PlayerItemRow | null>(null);
  const [inventoryEditQuantity, setInventoryEditQuantity] = useState("0");
  const [inventoryEditCategory, setInventoryEditCategory] = useState("materials");

  const [showRemoveItem, setShowRemoveItem] = useState<string | null>(null);
  const [removeItemQty, setRemoveItemQty] = useState(1);

  const [showAddGu, setShowAddGu] = useState(false);
  const [addGuId, setAddGuId] = useState("");
  const [addGuRank, setAddGuRank] = useState("");
  const [addGuQty, setAddGuQty] = useState(1);
  const [showGuEditor, setShowGuEditor] = useState<PlayerGuRow | null>(null);
  const [guEditForm, setGuEditForm] = useState({ rank: "1", level: "1", hunger: "100", status: "ACTIVE", equipped: false, is_vital: false, is_locked: false, integration_time: "0" });

  const [showStones, setShowStones] = useState(false);
  const [stonesPrimeval, setStonesPrimeval] = useState("");
  const [stonesImmortal, setStonesImmortal] = useState("");

  const [showModeration, setShowModeration] = useState(false);
  const [modReason, setModReason] = useState("");
  const [modHours, setModHours] = useState("72");
  const [modPermanent, setModPermanent] = useState(false);
  const [modChatMute, setModChatMute] = useState(false);
  const [modBlockDevice, setModBlockDevice] = useState(true);

  const [actionLoading, setActionLoading] = useState(false);
  const [tableLoading, setTableLoading] = useState<string | null>(null);
  const [datasets, setDatasets] = useState<DatasetDescriptor[]>([]);
  const section: AdminModuleKey = initialSection;
  const [playersTab, setPlayersTab] = useState<"list" | "online">("list");
  const [minesTab, setMinesTab] = useState<"slots" | "sect_mines">("slots");

  // ----- Aperture state -----
  const [apertures, setApertures] = useState<ApiResponseData[]>([]);
  const [aperturesTotal, setAperturesTotal] = useState(0);
  const [aperturesLoading, setAperturesLoading] = useState(false);
  const [aperturesPage, setAperturesPage] = useState(1);
  const [aperturesLimit] = useState(50);
  const [aperturesSearch, setAperturesSearch] = useState("");
  const [selectedApertureId, setSelectedApertureId] = useState<string | null>(null);
  const [apertureDetail, setApertureDetail] = useState<ApiResponseData | null>(null);
  const [apertureDetailLoading, setApertureDetailLoading] = useState(false);
  const [apertureForm, setApertureForm] = useState<ApiResponseData>({});
  const [apertureSaving, setApertureSaving] = useState(false);

  // ----- Shop state -----
  const [shopItems, setShopItems] = useState<ApiResponseData[]>([]);
  const [shopLoading, setShopLoading] = useState(false);
  const [showShopAdd, setShowShopAdd] = useState(false);
  const [showShopEdit, setShowShopEdit] = useState<string | null>(null);
  const [shopForm, setShopForm] = useState({ id: "", rank: 0, price: 100, stock: -1, is_default: true, sort_order: 0 });

  // ----- Arena Shop state -----
  const [arenaShopItems, setArenaShopItems] = useState<ApiResponseData[]>([]);
  const [arenaShopLoading, setArenaShopLoading] = useState(false);
  const [showArenaShopAdd, setShowArenaShopAdd] = useState(false);
  const [showArenaShopEdit, setShowArenaShopEdit] = useState<string | null>(null);
  const [arenaShopForm, setArenaShopForm] = useState({ id: "", price: 50, qty: 1, stock: -1, is_default: true, sort_order: 0 });
  const [shopTab, setShopTab] = useState<"caravan" | "arena">("caravan");
  const [shopSearchGus, setShopSearchGus] = useState<ApiResponseData[]>([]);
  const [shopSearchItems, setShopSearchItems] = useState<ApiResponseData[]>([]);
  const [shopSearchRecipes, setShopSearchRecipes] = useState<ApiResponseData[]>([]);
  const [shopSearchLoaded, setShopSearchLoaded] = useState(false);
  const [itemPicker, setItemPicker] = useState<{source: string; tab: "gu" | "item" | "recipe"; query: string} | null>(null);
  const [giftPickerIdx, setGiftPickerIdx] = useState<number | null>(null);

  // ----- Mine state -----
  const [mineSlots, setMineSlots] = useState<ApiResponseData[]>([]);
  const [mineLoading, setMineLoading] = useState(false);
  const [showMineEdit, setShowMineEdit] = useState<number | null>(null);
  const [mineEditRarity, setMineEditRarity] = useState("");

  // ----- Gift codes state -----
  const [giftCodes, setGiftCodes] = useState<ApiResponseData[]>([]);
  const [giftLoading, setGiftLoading] = useState(false);
  const [newGiftCode, setNewGiftCode] = useState("");
  const [newGiftMaxUses, setNewGiftMaxUses] = useState(100);
  const [newGiftPerPlayer, setNewGiftPerPlayer] = useState(1);
  const [newGiftExpiry, setNewGiftExpiry] = useState(30);
  const [newGiftRewards, setNewGiftRewards] = useState<{type: string; id: string; amount: number; label: string; rank?: number}[]>([
    { type: "primeval_stones", id: "", amount: 1000, label: "1.000 Nguyên Thạch" },
  ]);

  // ----- Online players state -----
  const [onlinePlayers, setOnlinePlayers] = useState<ApiResponseData[]>([]);
  const [onlineLoading, setOnlineLoading] = useState(false);

  // ----- Dictionary state -----
  const [dataTab, setDataTab] = useState<"import" | "tables" | "gus" | "items" | "recipes" | "monsters" | "loot_tables" | "regions" | "map_cells" | "scouting" | "constitution" | "race" | "status" | "spawns" | "quests" | "dungeons">("import");
  const [regions, setRegions] = useState<ApiResponseData[]>([]);
  const [regionsLoading, setRegionsLoading] = useState(false);
  const [regionEdit, setRegionEdit] = useState<ApiResponseData | null>(null);
  const [regionRulesDraft, setRegionRulesDraft] = useState("{}");
  const [regionSearch, setRegionSearch] = useState("");
  const [regionsPage, setRegionsPage] = useState(1);
  const [regionsTotal, setRegionsTotal] = useState(0);
  const [regionsTotalPages, setRegionsTotalPages] = useState(1);

  const openRegionEditor = (region: ApiResponseData) => {
    setRegionRulesDraft(JSON.stringify(region.rules || {}, null, 2));
    setRegionEdit(region);
  };

  const [mapCells, setMapCells] = useState<ApiResponseData[]>([]);
  const [mapCellsLoading, setMapCellsLoading] = useState(false);
  const [mapCellEdit, setMapCellEdit] = useState<ApiResponseData | null>(null);
  const [mapCellSearch, setMapCellSearch] = useState("");
  const [mapCellRegionFilter, setMapCellRegionFilter] = useState("");
  const [mapCellTypeFilter, setMapCellTypeFilter] = useState("");
  const [mapCellsPage, setMapCellsPage] = useState(1);
  const [mapCellsTotal, setMapCellsTotal] = useState(0);
  const [mapCellsTotalPages, setMapCellsTotalPages] = useState(1);
  const [residualTraces, setResidualTraces] = useState<ApiResponseData[]>([]);
  const [residualTracesLoading, setResidualTracesLoading] = useState(false);
  const [residualTraceSearch, setResidualTraceSearch] = useState("");
  const [residualTracePage, setResidualTracePage] = useState(1);
  const [residualTraceTotal, setResidualTraceTotal] = useState(0);
  const [residualTraceEdit, setResidualTraceEdit] = useState<ApiResponseData | null>(null);
  const [residualTraceForm, setResidualTraceForm] = useState({region_id:"",x:"0",y:"0",player_id:"",action_type:"move",dominant_path:"",dao_pulse_strength:"0",expires_at:""});
  const [gus, setGus] = useState<ApiResponseData[]>([]);
  const [gusLoading, setGusLoading] = useState(false);
  const [guEdit, setGuEdit] = useState<ApiResponseData | null>(null);
  const [items, setItems] = useState<ApiResponseData[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  const [itemEdit, setItemEdit] = useState<ApiResponseData | null>(null);
  const [uploading, setUploading] = useState(false);
  const [recipes, setRecipes] = useState<ApiResponseData[]>([]);
  const [recipesLoading, setRecipesLoading] = useState(false);
  const [recipeEdit, setRecipeEdit] = useState<ApiResponseData | null>(null);
  const [recipeSearch, setRecipeSearch] = useState("");
  const [monsters, setMonsters] = useState<ApiResponseData[]>([]);
  const [monstersLoading, setMonstersLoading] = useState(false);
  const [monsterEdit, setMonsterEdit] = useState<ApiResponseData | null>(null);
  const [monsterSearch, setMonsterSearch] = useState("");
  const [monsterSortField, setMonsterSortField] = useState<string>("id");
  const [monsterSortDir, setMonsterSortDir] = useState<"asc" | "desc">("asc");
  const [lootTables, setLootTables] = useState<ApiResponseData[]>([]);
  const [lootTablesLoading, setLootTablesLoading] = useState(false);
  const [lootTableEdit, setLootTableEdit] = useState<ApiResponseData | null>(null);
  const [lootTableSearch, setLootTableSearch] = useState("");
  const [lootTableSortField, setLootTableSortField] = useState<string>("id");
  const [lootTableSortDir, setLootTableSortDir] = useState<"asc" | "desc">("asc");
  const [dictGuSearch, setDictGuSearch] = useState("");
  const [guSortField, setGuSortField] = useState<string>("id");
  const [guSortDir, setGuSortDir] = useState<"asc" | "desc">("asc");
  const [dictItemSearch, setDictItemSearch] = useState("");
  const [itemSortField, setItemSortField] = useState<string>("id");
  const [itemSortDir, setItemSortDir] = useState<"asc" | "desc">("asc");

  // ----- Sect state -----
  const [sects, setSects] = useState<ApiResponseData[]>([]);
  const [sectsLoading, setSectsLoading] = useState(false);
  const [selectedSect, setSelectedSect] = useState<ApiResponseData | null>(null);
  const [selectedSectMembers, setSelectedSectMembers] = useState<ApiResponseData[]>([]);
  const [selectedSectId, setSelectedSectId] = useState<string | null>(null);
  const [showSectEdit, setShowSectEdit] = useState(false);
  const [showSectAddMember, setShowSectAddMember] = useState(false);
  const [sectAddPlayerId, setSectAddPlayerId] = useState("");
  const [sectAddPlayerRank, setSectAddPlayerRank] = useState("NGOAI_MON");
  const [sectEditName, setSectEditName] = useState("");
  const [sectEditDesc, setSectEditDesc] = useState("");
  const [sectEditRegion, setSectEditRegion] = useState("");
  const [sectEditLevel, setSectEditLevel] = useState("");
  const [sectEditFunds, setSectEditFunds] = useState("");
  const [sectEditRequireApproval, setSectEditRequireApproval] = useState(false);
  const [sectEditRequirePromotionApproval, setSectEditRequirePromotionApproval] = useState(false);
  const [sectDetailTab, setSectDetailTab] = useState<"members" | "vault" | "missions">("members");

  const [sectMissions, setSectMissions] = useState<ApiResponseData[]>([]);
  const [sectMissionsLoading, setSectMissionsLoading] = useState(false);
  const [showCreateMission, setShowCreateMission] = useState(false);
  const [newMission, setNewMission] = useState({ title: "", description: "", mission_type: "GATHER", target_id: "", target_amount: 1, reward_contribution: 10, required_member_rank: "NGOAI_MON", duration_hours: 24 });
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

  const fetchApertures = useCallback(async () => {
    setAperturesLoading(true);
    try {
      const params = new URLSearchParams();
      params.set("page", aperturesPage.toString());
      params.set("limit", aperturesLimit.toString());
      if (aperturesSearch) params.set("search", aperturesSearch);
      const res = await authFetch(`${BACKEND_URL}/api/admin/apertures?${params}`);
      if (res.ok) {
        const data = await res.json();
        setApertures(data.data || []);
        setAperturesTotal(data.total || 0);
      }
    } catch {} finally {
      setAperturesLoading(false);
    }
  }, [aperturesPage, aperturesLimit, aperturesSearch]);

  const fetchApertureDetail = useCallback(async (id: string) => {
    setApertureDetailLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/apertures/${id}`);
      if (res.ok) {
        const data = await res.json();
        setApertureDetail(data.aperture);
        setApertureForm({
          ...data.aperture,
          dao_marks_array: Object.entries(data.aperture.dao_marks || {}).map(([path, amount]) => ({ path, amount })),
        });
      }
    } catch {} finally {
      setApertureDetailLoading(false);
    }
  }, []);

  const saveApertureDetail = async () => {
    if (!selectedApertureId) return;
    setApertureSaving(true);
    try {
      const dao_marks: Record<string, number> = {};
      if (apertureForm.dao_marks_array) {
        for (const entry of apertureForm.dao_marks_array) {
          if (entry.path.trim()) {
            dao_marks[entry.path.trim()] = Number(entry.amount) || 0;
          }
        }
      }

      const payload: ApiResponseData = {};
      const addIfChanged = (key: string, originalVal: ApiResponseData, newVal: ApiResponseData) => {
        if (originalVal !== newVal) {
          payload[key] = newVal;
        }
      };

      if (apertureDetail) {
        addIfChanged("grade", apertureDetail.grade, apertureForm.grade);
        addIfChanged("time_rate", Number(apertureDetail.time_rate), Number(apertureForm.time_rate));
        addIfChanged("aperture_hp", Number(apertureDetail.aperture_hp), Number(apertureForm.aperture_hp));
        addIfChanged("max_aperture_hp", Number(apertureDetail.max_aperture_hp), Number(apertureForm.max_aperture_hp));
        addIfChanged("area_size", Number(apertureDetail.area_size), Number(apertureForm.area_size));
        addIfChanged("immortal_essence", Number(apertureDetail.immortal_essence), Number(apertureForm.immortal_essence));
        addIfChanged("essence_production_per_year", Number(apertureDetail.essence_production_per_year), Number(apertureForm.essence_production_per_year));
        addIfChanged("internal_years_passed", Number(apertureDetail.internal_years_passed), Number(apertureForm.internal_years_passed));
        addIfChanged("next_tribulation_year", Number(apertureDetail.next_tribulation_year), Number(apertureForm.next_tribulation_year));
        addIfChanged("rank_start_year", Number(apertureDetail.rank_start_year), Number(apertureForm.rank_start_year));
        addIfChanged("earth_calamity_count", Number(apertureDetail.earth_calamity_count), Number(apertureForm.earth_calamity_count));
        addIfChanged("heavenly_tribulation_count", Number(apertureDetail.heavenly_tribulation_count), Number(apertureForm.heavenly_tribulation_count));
        addIfChanged("grand_tribulation_count", Number(apertureDetail.grand_tribulation_count), Number(apertureForm.grand_tribulation_count));
        addIfChanged("myriad_tribulation_count", Number(apertureDetail.myriad_tribulation_count), Number(apertureForm.myriad_tribulation_count));
        addIfChanged("chaos_calamity_count", Number(apertureDetail.chaos_calamity_count), Number(apertureForm.chaos_calamity_count));
        addIfChanged("state", apertureDetail.state, apertureForm.state);

        const originalDao = JSON.stringify(apertureDetail.dao_marks || {});
        const newDao = JSON.stringify(dao_marks);
        if (originalDao !== newDao) {
          payload["dao_marks"] = dao_marks;
        }
      } else {
        // Fallback fallback if detail is not loaded
        Object.assign(payload, {
          grade: apertureForm.grade,
          time_rate: Number(apertureForm.time_rate),
          aperture_hp: Number(apertureForm.aperture_hp),
          max_aperture_hp: Number(apertureForm.max_aperture_hp),
          area_size: Number(apertureForm.area_size),
          immortal_essence: Number(apertureForm.immortal_essence),
          essence_production_per_year: Number(apertureForm.essence_production_per_year),
          internal_years_passed: Number(apertureForm.internal_years_passed),
          next_tribulation_year: Number(apertureForm.next_tribulation_year),
          rank_start_year: Number(apertureForm.rank_start_year),
          earth_calamity_count: Number(apertureForm.earth_calamity_count),
          heavenly_tribulation_count: Number(apertureForm.heavenly_tribulation_count),
          grand_tribulation_count: Number(apertureForm.grand_tribulation_count),
          myriad_tribulation_count: Number(apertureForm.myriad_tribulation_count),
          chaos_calamity_count: Number(apertureForm.chaos_calamity_count),
          state: apertureForm.state,
          dao_marks,
        });
      }

      if (Object.keys(payload).length === 0) {
        setSelectedApertureId(null);
        setApertureDetail(null);
        return;
      }

      const res = await authFetch(`${BACKEND_URL}/api/admin/apertures/${selectedApertureId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setSelectedApertureId(null);
        setApertureDetail(null);
        fetchApertures();
      } else {
        const err = await res.json();
        alert("Lỗi: " + (err.error || "Không thể lưu tiên khiếu"));
      }
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      alert("Lỗi kết nối: " + err.message);
    } finally {
      setApertureSaving(false);
    }
  };

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

  const fetchArenaShopItems = useCallback(async () => {
    setArenaShopLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/arena-shop`);
      if (res.ok) setArenaShopItems(await res.json());
    } catch {} finally { setArenaShopLoading(false); }
  }, []);

  const fetchMineSlots = useCallback(async () => {
    setMineLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/mine-slots`);
      if (res.ok) setMineSlots(await res.json());
    } catch {} finally { setMineLoading(false); }
  }, []);

  const fetchOnlinePlayers = useCallback(async () => {
    setOnlineLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/online?minutes=5`);
      if (res.ok) setOnlinePlayers(await res.json());
    } catch {} finally { setOnlineLoading(false); }
  }, []);

  const fetchGiftCodes = useCallback(async () => {
    setGiftLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/gift-codes`);
      if (!res.ok) throw new Error("Không thể tải mã quà");
      const data = await res.json();
      setGiftCodes(data.codes || []);
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error(err);
    } finally { setGiftLoading(false); }
  }, []);

  const handleCreateGiftCode = async () => {
    if (!newGiftCode.trim()) { if (addToast) addToast("Vui lòng nhập mã code", "error"); return; }
    if (newGiftRewards.length === 0) { if (addToast) addToast("Vui lòng thêm ít nhất 1 phần thưởng", "error"); return; }
    setGiftLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/gift-codes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          code: newGiftCode.trim(),
          max_uses: newGiftMaxUses,
          max_per_player: newGiftPerPlayer,
          expires_in_days: newGiftExpiry,
          rewards: newGiftRewards,
        }),
      });
      const data = await res.json().catch(() => ({ error: "Lỗi không xác định" }));
      if (!res.ok) throw new Error(data.error || "Thất bại");
      if (addToast) addToast(data.message || "Tạo mã quà thành công", "success");
      setNewGiftCode("");
      setNewGiftRewards([{ type: "primeval_stones", id: "", amount: 1000, label: "1.000 Nguyên Thạch" }]);
      fetchGiftCodes();
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      if (addToast) addToast(err.message, "error");
    } finally { setGiftLoading(false); }
  };

  const fetchGus = useCallback(async () => {
    setGusLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/gu-dictionary`); if (res.ok) { const data = await res.json(); setGus(Array.isArray(data) ? data : []); } }
    catch {} finally { setGusLoading(false); }
  }, []);
  const fetchItems = useCallback(async () => {
    setItemsLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/item-dictionary`); if (res.ok) { const data = await res.json(); setItems(Array.isArray(data) ? data : []); } }
    catch {} finally { setItemsLoading(false); }
  }, []);
  const fetchRecipes = useCallback(async () => {
    setRecipesLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/recipes`); if (res.ok) { const data = await res.json(); if (Array.isArray(data)) setRecipes(data); else setRecipes([]); } }
    catch {} finally { setRecipesLoading(false); }
  }, []);
  const fetchMonsters = useCallback(async () => {
    setMonstersLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/monsters`); if (res.ok) { const data = await res.json(); setMonsters(Array.isArray(data) ? data : []); } }
    catch {} finally { setMonstersLoading(false); }
  }, []);
  const fetchLootTables = useCallback(async () => {
    setLootTablesLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/loot-tables`); if (res.ok) { const data = await res.json(); setLootTables(Array.isArray(data) ? data : []); } }
    catch {} finally { setLootTablesLoading(false); }
  }, []);

  const fetchRegions = useCallback(async (pageVal = regionsPage, searchVal = regionSearch) => {
    setRegionsLoading(true);
    try {
      const params = new URLSearchParams({
        page: pageVal.toString(),
        page_size: "20",
        search: searchVal,
      });
      const res = await authFetch(`${BACKEND_URL}/api/admin/regions?${params.toString()}`);
      if (res.ok) {
        const json = await res.json();
        setRegions(json.data || []);
        setRegionsTotal(json.total || 0);
        setRegionsTotalPages(json.last_page || 1);
        setRegionsPage(json.page || 1);
      }
    } catch {} finally { setRegionsLoading(false); }
  }, [regionsPage, regionSearch]);

  const fetchMapCells = useCallback(async (pageVal = mapCellsPage, searchVal = mapCellSearch, rFilter = mapCellRegionFilter, tFilter = mapCellTypeFilter) => {
    setMapCellsLoading(true);
    try {
      const params = new URLSearchParams({
        page: pageVal.toString(),
        page_size: "20",
        search: searchVal,
        region_id: rFilter,
        cell_type: tFilter,
      });
      const res = await authFetch(`${BACKEND_URL}/api/admin/map-cells?${params.toString()}`);
      if (res.ok) {
        const json = await res.json();
        setMapCells(json.data || []);
        setMapCellsTotal(json.total || 0);
        setMapCellsTotalPages(json.last_page || 1);
        setMapCellsPage(json.page || 1);
      }
    } catch {} finally { setMapCellsLoading(false); }
  }, [mapCellsPage, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter]);

  const fetchResidualTraces = useCallback(async (pageVal = residualTracePage, searchVal = residualTraceSearch) => {
    setResidualTracesLoading(true);
    try { const params=new URLSearchParams({page:String(pageVal),limit:"30",search:searchVal});const res=await authFetch(`${BACKEND_URL}/api/admin/residual-traces?${params}`);if(res.ok){const data=await res.json();setResidualTraces(data.data||[]);setResidualTraceTotal(data.total||0);} }
    catch {} finally { setResidualTracesLoading(false); }
  }, [residualTracePage,residualTraceSearch]);

  const ensureShopSearchLoaded = useCallback(async () => {
    if (shopSearchLoaded) return;
    const [g, i, r] = await Promise.all([
      authFetch(`${BACKEND_URL}/api/admin/gu-dictionary`),
      authFetch(`${BACKEND_URL}/api/admin/item-dictionary`),
      authFetch(`${BACKEND_URL}/api/admin/recipes`),
    ]);
    if (g.ok) setShopSearchGus(await g.json());
    if (i.ok) setShopSearchItems(await i.json());
    if (r.ok) setShopSearchRecipes(await r.json());
    setShopSearchLoaded(true);
  }, [shopSearchLoaded]);

  useEffect(() => {
    if (section === "sects") fetchSects();
  }, [section, fetchSects]);

  useEffect(() => {
    if (section === "apertures") fetchApertures();
  }, [section, fetchApertures]);

  const [apertureSearchDebounce, setApertureSearchDebounce] = useState("");
  useEffect(() => {
    const t = setTimeout(() => {
      setAperturesSearch(apertureSearchDebounce);
      setAperturesPage(1);
    }, 400);
    return () => clearTimeout(t);
  }, [apertureSearchDebounce]);

  useEffect(() => {
    if (selectedApertureId) {
      fetchApertureDetail(selectedApertureId);
    }
  }, [selectedApertureId, fetchApertureDetail]);

  useEffect(() => {
    if (section === "shop") {
      fetchShopItems();
      fetchArenaShopItems();
    }
  }, [section, fetchShopItems, fetchArenaShopItems]);

  useEffect(() => {
    if (section === "mines") fetchMineSlots();
  }, [section, fetchMineSlots]);

  useEffect(() => {
    if (section === "gift_codes") fetchGiftCodes();
  }, [section, fetchGiftCodes]);

  useEffect(() => {
    if (section === "data") {
      if (dataTab === "tables") {
        void authFetch(`${BACKEND_URL}/api/admin/data/tables`)
          .then(async response => {
            if (!response.ok) throw new Error("Không thể tải danh sách dataset");
            const body = await response.json() as { datasets?: DatasetDescriptor[] };
            setDatasets((body.datasets ?? []).filter(dataset => dataset.ui_supported));
          })
          .catch(error => addToast(error instanceof Error ? error.message : "Không thể tải danh sách dataset", "error"));
      }
      if (dataTab === "gus") fetchGus();
      else if (dataTab === "items") fetchItems();
      else if (dataTab === "recipes") fetchRecipes();
      else if (dataTab === "monsters") fetchMonsters();
      else if (dataTab === "loot_tables") fetchLootTables();
      else if (dataTab === "regions") fetchRegions(1, regionSearch);
      else if (dataTab === "map_cells") fetchMapCells(1, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter);
      else if (dataTab === "scouting") fetchResidualTraces(1, residualTraceSearch);
    }
  }, [section, dataTab, fetchGus, fetchItems, fetchRecipes, fetchMonsters, fetchLootTables, fetchRegions, fetchMapCells, fetchResidualTraces, regionSearch, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter, residualTraceSearch, addToast]);

  useEffect(() => {
    if (section === "players" && playersTab === "online") fetchOnlinePlayers();
  }, [section, playersTab, fetchOnlinePlayers]);


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
      const params = new URLSearchParams({ page: String(page), page_size: String(limit), sort: playerQuery.sort, direction: playerQuery.direction });
      const q = search.trim();
      if (q) {
        params.set("search", q);
        let by = searchBy;
        if (by === "all" && UUID_RE.test(q)) by = "id";
        if (by !== "all") params.set("search_by", by);
      }
      const res = await authFetch(`${BACKEND_URL}/api/admin/players?${params}`);
      if (!res.ok) throw new Error((await res.json().catch(() => ({ error: "Lỗi" }))).error || "Lỗi tải");
      const data = await res.json();
      setPlayers(data.data);
      setTotal(data.total);
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      setError(err.message);
    } finally { setLoading(false); }
  }, [page, limit, search, searchBy, playerQuery.direction, playerQuery.sort]);

  useEffect(() => {
    if (section === "players") fetchPlayers();
  }, [section, fetchPlayers]);

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

  const fetchPlayerItems = useCallback(async (id: string) => {
    setPlayerItemsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(playerItemsPage), limit: String(playerItemsLimit) });
      if (playerItemsSearch.trim()) params.set("search", playerItemsSearch.trim());
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/items?${params}`);
      if (res.ok) {
        const data = await res.json();
        setPlayerItems(data.data || []);
        setPlayerItemsTotal(data.total || 0);
      }
    } catch {} finally { setPlayerItemsLoading(false); }
  }, [playerItemsPage, playerItemsLimit, playerItemsSearch]);

  const fetchPlayerGus = useCallback(async (id: string) => {
    setPlayerGusLoading(true);
    try {
      const params = new URLSearchParams({ page: String(playerGusPage), limit: String(playerGusLimit) });
      if (playerGusSearch.trim()) params.set("search", playerGusSearch.trim());
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/gus?${params}`);
      if (res.ok) {
        const data = await res.json();
        setPlayerGus(data.data || []);
        setPlayerGusTotal(data.total || 0);
      }
    } catch {} finally { setPlayerGusLoading(false); }
  }, [playerGusPage, playerGusLimit, playerGusSearch]);

  const fetchPlayerPets = useCallback(async (id: string) => {
    setPlayerPetsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(playerPetsPage), limit: String(playerPetsLimit) });
      if (playerPetsSearch.trim()) params.set("search", playerPetsSearch.trim());
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/pets?${params}`);
      if (res.ok) {
        const data = await res.json();
        setPlayerPets(data.data || []);
        setPlayerPetsTotal(data.total || 0);
      }
    } catch {} finally { setPlayerPetsLoading(false); }
  }, [playerPetsPage, playerPetsLimit, playerPetsSearch]);

  const fetchPlayerQuests = useCallback(async (id: string) => {
    setPlayerQuestsLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/quests`);
      if (res.ok) setPlayerQuests((await res.json()).data || []);
    } catch {} finally { setPlayerQuestsLoading(false); }
  }, []);

  const fetchInstanceSessions = useCallback(async (id: string) => {
    setInstanceSessionsLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/instance-sessions`);
      if (res.ok) setInstanceSessions((await res.json()).data || []);
    } catch {} finally { setInstanceSessionsLoading(false); }
  }, []);

  const fetchPlayerSectMissions = useCallback(async (id: string) => {
    setPlayerSectMissionsLoading(true);
    try { const res = await authFetch(`${BACKEND_URL}/api/admin/players/${id}/sect-missions`); if(res.ok) setPlayerSectMissions((await res.json()).data || []); }
    catch {} finally { setPlayerSectMissionsLoading(false); }
  }, []);

  const fetchPlayerTrackingMarks = useCallback(async (id: string) => {
    setPlayerTrackingMarksLoading(true);
    try { const res=await authFetch(`${BACKEND_URL}/api/admin/players/${id}/tracking-marks`);if(res.ok)setPlayerTrackingMarks((await res.json()).data||[]); }
    catch {} finally { setPlayerTrackingMarksLoading(false); }
  }, []);

  const fetchPlayerAchievements = useCallback(async (id: string) => {
    setPlayerAchievementsLoading(true);
    try { const res=await authFetch(`${BACKEND_URL}/api/admin/players/${id}/achievements`);if(res.ok){const data=await res.json();setPlayerAchievements(data.data||[]);setAvailableAchievements(data.available||[]);if(data.available?.length)setAchievementToGrant(current=>current||data.available[0].id);} }
    catch {} finally { setPlayerAchievementsLoading(false); }
  }, []);



  useEffect(() => {
    if (selectedId && activeTab === "items") fetchPlayerItems(selectedId);
  }, [selectedId, activeTab, playerItemsPage, playerItemsSearch, fetchPlayerItems]);

  useEffect(() => {
    if (selectedId && activeTab === "gus") fetchPlayerGus(selectedId);
  }, [selectedId, activeTab, playerGusPage, playerGusSearch, fetchPlayerGus]);

  useEffect(() => {
    if (selectedId && activeTab === "pets") fetchPlayerPets(selectedId);
  }, [selectedId, activeTab, playerPetsPage, playerPetsSearch, fetchPlayerPets]);

  useEffect(() => {
    if (selectedId && activeTab === "quests") fetchPlayerQuests(selectedId);
  }, [selectedId, activeTab, fetchPlayerQuests]);

  useEffect(() => {
    if (selectedId && activeTab === "instances") fetchInstanceSessions(selectedId);
  }, [selectedId, activeTab, fetchInstanceSessions]);

  useEffect(() => {
    if (selectedId && activeTab === "sect_missions") fetchPlayerSectMissions(selectedId);
  }, [selectedId, activeTab, fetchPlayerSectMissions]);

  useEffect(() => {
    if (selectedId && activeTab === "tracking_marks") fetchPlayerTrackingMarks(selectedId);
  }, [selectedId, activeTab, fetchPlayerTrackingMarks]);

  useEffect(() => {
    if (selectedId && activeTab === "achievements") fetchPlayerAchievements(selectedId);
  }, [selectedId, activeTab, fetchPlayerAchievements]);



  const exec = async (url: string, method: string, body?: ApiResponseData) => {
    setActionLoading(true);
    try {
	  const idempotencyKey = typeof body?.idempotencyKey === "string" ? body.idempotencyKey : undefined;
      const requestBody = body ? { ...body } : undefined;
      if (requestBody && typeof requestBody.confirmation_target === "string") {
        const pathTargetRoute = method === "DELETE" || /\/api\/admin\/(?:players\/[^/]+\/(items(?:\/|$)|recipes\/|quests\/|instance-sessions\/|sect-missions\/|tracking-marks\/|achievements(?:\/|$))|residual-traces\/)/.test(new URL(url, window.location.origin).pathname);
        if (pathTargetRoute) requestBody.confirmation_target = new URL(url, window.location.origin).pathname;
      }
      const res = await authFetch(url, {
        method,
		headers: { "Content-Type": "application/json", ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}) },
        body: requestBody ? JSON.stringify(requestBody) : undefined,
      });
      const data = await res.json().catch(() => ({ error: "Lỗi không xác định" }));
      if (!res.ok) throw new Error(data.error || "Thất bại");
      if (selectedId) {
        fetchDetail(selectedId);
        if (activeTab === "items") fetchPlayerItems(selectedId);
        else if (activeTab === "gus") fetchPlayerGus(selectedId);
        else if (activeTab === "pets") fetchPlayerPets(selectedId);
        else if (activeTab === "quests") fetchPlayerQuests(selectedId);
        else if (activeTab === "instances") fetchInstanceSessions(selectedId);
        else if (activeTab === "sect_missions") fetchPlayerSectMissions(selectedId);
        else if (activeTab === "tracking_marks") fetchPlayerTrackingMarks(selectedId);
        else if (activeTab === "achievements") fetchPlayerAchievements(selectedId);
      }
      if (addToast) addToast(data.message || "Thành công", "success");
      return true;
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      if (addToast) addToast(err.message, "error");
      return false;
    } finally { setActionLoading(false); }
  };

  const totalPages = Math.ceil(total / limit);

  const renderItemPicker = () => itemPicker && (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/70" onClick={() => setItemPicker(null)}>
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 w-full max-w-2xl mx-4 shadow-2xl max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-serif font-bold text-zinc-200">Chọn vật phẩm</h3>
          <button onClick={() => setItemPicker(null)} className="text-zinc-500 hover:text-zinc-300 cursor-pointer"><X className="w-4 h-4" /></button>
        </div>
        <div className="flex gap-1 bg-zinc-800/60 rounded-lg p-1 mb-3">
          {(["gu", "item", "recipe"] as const).filter(t => {
            const s = itemPicker.source;
            if (s === "add-gu" || s === "mission-donate") return t === "gu";
            if (s === "add-item") return t === "item" || t === "recipe";
            if (s === "vault") return true;
            return true;
          }).map(t => (
            <button key={t} onClick={() => setItemPicker(p => p ? {...p, tab: t} : p)} className={`flex-1 px-3 py-1.5 text-xs font-semibold font-mono rounded-md transition-colors cursor-pointer ${itemPicker.tab === t ? "bg-zinc-700 text-zinc-200" : "text-zinc-500 hover:text-zinc-300"}`}>
              {t === "gu" ? "Cổ Trùng" : t === "item" ? "Item" : "Công thức"}
            </button>
          ))}
        </div>
        <div className="relative mb-3">
          <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input value={itemPicker.query} onChange={e => setItemPicker(p => p ? {...p, query: e.target.value} : p)} placeholder="Tìm kiếm..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" autoFocus />
        </div>
        <div className="overflow-y-auto flex-1 space-y-0.5">
          {(() => {
            let list: ApiResponseData[] = [];
            if (itemPicker.tab === "gu") list = shopSearchGus;
            else if (itemPicker.tab === "item") list = shopSearchItems;
            else list = shopSearchRecipes;
            const q = itemPicker.query.toLowerCase();
            if (q) list = list.filter((x: ApiResponseData) => (x.id || "").toLowerCase().includes(q) || (x.name || "").toLowerCase().includes(q));
            if (itemPicker.tab === "recipe") {
              list = list.filter((x: ApiResponseData) => x.target_gu_id);
            }
            if (itemPicker.source === "vault" && itemPicker.tab === "item") {
              list = list.filter((x: ApiResponseData) => {
                const itType = (x.item_type || x.type || "").toLowerCase();
                return itType !== "consumable" && itType !== "currency";
              });
            }
            return list.length === 0 ? (
              <div className="text-center py-8 text-zinc-500 text-xs italic">Không tìm thấy</div>
            ) : (
              list.slice(0, 100).map((x: ApiResponseData) => (
                <button key={itemPicker.tab === "gu" ? `${x.id}:${x.rank ?? 0}` : x.id} onClick={() => {
                  const s = itemPicker.source;
                  if (s === "shop") setShopForm(f => ({ ...f, id: x.id }));
                  else if (s === "arena") setArenaShopForm(f => ({ ...f, id: x.id }));
                  else if (s === "add-item") setAddItemId(x.id);
                  else if (s === "add-gu") { setAddGuId(x.id); setAddGuRank(x.rank ? String(x.rank) : ""); }
                  else if (s === "vault") {
                    const t = itemPicker.tab === "gu" ? "gu" : itemPicker.tab === "item" ? "material" : "recipe";
                    setStockVaultType(t);
                    setStockVaultId(x.id);
                  } else if (s === "mission" || s === "mission-donate") {
                    setNewMission(p => ({ ...p, target_id: x.id }));
                  } else if (s === "gift" && giftPickerIdx !== null) {
                    const upd = [...newGiftRewards];
                    upd[giftPickerIdx] = { ...upd[giftPickerIdx], id: x.id, amount: 1, label: x.name || x.id };
                    setNewGiftRewards(upd);
                    setGiftPickerIdx(null);
                  }
                  setItemPicker(null);
                }} className="w-full text-left px-3 py-2 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer flex items-center gap-3">
                  <span className="text-xs font-mono text-zinc-400 min-w-[140px]">{x.id}</span>
                  <span className="text-xs text-zinc-200 truncate">{x.name || x.target_gu_id || "—"}</span>
                  {itemPicker.tab === "gu" && x.rank && <span className="text-[10px] text-zinc-500 ml-auto">Bậc {x.rank}</span>}
                  {itemPicker.tab === "item" && x.item_type && <span className="text-[10px] text-zinc-500 ml-auto">{translateItemType(x.item_type)}</span>}
                  {itemPicker.tab === "recipe" && x.target_gu_id && <span className="text-[10px] text-zinc-500 ml-auto">→ {x.target_gu_id}</span>}
                </button>
              ))
            );
          })()}
        </div>
      </div>
    </div>
  );

  if (!authorized && !player?.is_admin) {
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
                <p className="text-xs text-zinc-500 font-mono">{detail.id}</p>
              )}
            </div>
          </div>
          <div className="flex gap-2 flex-wrap">
            {canAdminAction("updatePlayer") && <button onClick={() => setShowStones(true)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
              <Coins className="w-3.5 h-3.5" />
              Điều chỉnh tài nguyên
            </button>}
              {canAdminAction("updatePlayer") && <button onClick={() => {
              if (detail) {
                setEditPlayerForm({
                  username: detail.username || "",
                  rank: String(detail.rank), sub_rank: String(detail.sub_rank),
                  aptitude_type: detail.aptitude_type || "", aptitude_percent: String(detail.aptitude_percent ?? ""),
                  aptitude_grade: detail.aptitude_grade || "", current_mana: String(detail.current_mana ?? ""),
                  max_mana: String(detail.max_mana ?? ""), hp: String(detail.hp ?? ""), base_hp: String(detail.base_hp ?? ""),
                  wall_progress: String(detail.wall_progress ?? ""), human_qi: String(detail.human_qi ?? ""),
                  lifespan: String(detail.lifespan ?? ""), current_region: detail.current_region || "",
                  x: String(detail.x ?? 0), y: String(detail.y ?? 0),
                  soul_hp: String(detail.soul_hp ?? ""),
                  max_soul: String(detail.max_soul ?? ""), soul_tier: detail.soul_tier || "",
                  explore_count: String(detail.explore_count ?? ""), aperture_stress: String(detail.aperture_stress ?? ""),
                  bio: detail.bio || "", avatar_url: detail.avatar_url || "",
                  race: detail.race || "", gender: detail.gender || "", constitution: detail.constitution || "",
                  statuses: (detail.statuses || []).map((s: string | { id: string }) => typeof s === "string" ? s : s.id).join(", "),
                  path_masteries: JSON.stringify(detail.path_masteries || {}, null, 2),
                  current_thought: String(detail.current_thought ?? 0), max_thought: String(detail.max_thought ?? 0),
                  thought_regen_rate: String(detail.thought_regen_rate ?? 0), time_speed_multiplier: String(detail.time_speed_multiplier ?? 1),
                  selected_title: detail.selected_title || "", selected_frame: detail.selected_frame || "",
                  fake_rank: String(detail.fake_rank ?? 0), fake_sub_rank: String(detail.fake_sub_rank ?? 0),
                  fake_vital_gu: detail.fake_vital_gu || "", is_disguised: String(detail.is_disguised ?? false),
                  needs_onboarding: String(detail.needs_onboarding ?? false), needs_soul_refine: String(detail.needs_soul_refine ?? false),
                  immortal_jade: String(detail.immortal_jade ?? 0),
                });
                setShowEditPlayer(true);
              }
            }} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer disabled:opacity-50">
              <Edit3 className="w-3.5 h-3.5" />
              Sửa thông tin
            </button>}
            <button onClick={() => {
              if (detail) {
                window.location.assign("/admin/apertures");
                setApertureSearchDebounce(detail.username);
                setSelectedId(null);
                setDetail(null);
              }
            }} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer disabled:opacity-50">
              <Globe className="w-3.5 h-3.5 text-amber-500" />
              Xem Tiên Khiếu
            </button>
            {(adminAccess.effective_permissions.includes("admin.*") || adminAccess.effective_permissions.includes("admin_roles.read") || canAdminAction("assignPlayerRoles")) && <button onClick={() => setShowRolesEditor(true)} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer disabled:opacity-50">
              <Crown className="w-3.5 h-3.5 text-amber-400" />
              Vai trò
            </button>}
            {canAdminAction("banPlayer") && <button onClick={() => setShowModeration(true)} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-red-900/60 text-xs text-red-300 hover:text-white hover:bg-red-950/40 transition-colors cursor-pointer disabled:opacity-50">
              <Shield className="w-3.5 h-3.5 text-red-400" />
              {detail?.is_banned ? "Đã khóa" : "Khóa / Hạn chế"}
            </button>}
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
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Zap className="w-3.5 h-3.5 text-blue-400" />{detail.rank < 6 ? "Chân nguyên" : "Tiên nguyên"}</div>
                <span className="text-sm font-bold text-zinc-200">{Math.round(detail.current_mana)}/{Math.round(detail.max_mana)}</span>
              </div>
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Coins className="w-3.5 h-3.5 text-yellow-500" />Nguyên Thạch</div>
                <span className="text-sm font-bold text-yellow-500">{detail.primeval_stones.toLocaleString()}</span>
              </div>
              {/*Tiên Nguyên Thạch */}
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-3">
                <div className="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-1"><Star className="w-3.5 h-3.5 text-purple-500" />Tiên Nguyên Thạch</div>
                <span className="text-sm font-bold text-purple-500">{detail.immortal_stones.toLocaleString()}</span>
              </div>
            </div>

            {/* Tabs */}
            <div className="border-b border-zinc-800/80 flex gap-0 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
              <button onClick={() => setActiveTab("items")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "items" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Package className="w-3.5 h-3.5 inline mr-1.5" />Vật phẩm ({detail.item_count || 0})
              </button>
              <button onClick={() => setActiveTab("gus")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "gus" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Bug className="w-3.5 h-3.5 inline mr-1.5" />Cổ trùng ({detail.gu_count || 0})
              </button>
              <button onClick={() => setActiveTab("pets")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "pets" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <PawPrint className="w-3.5 h-3.5 inline mr-1.5" />Sủng vật ({playerPetsTotal})
              </button>
              <button onClick={() => setActiveTab("quests")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "quests" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <ListTodo className="w-3.5 h-3.5 inline mr-1.5" />Nhiệm vụ ({playerQuests.length})
              </button>
              <button onClick={() => setActiveTab("instances")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "instances" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Globe className="w-3.5 h-3.5 inline mr-1.5" />Phiên bí cảnh ({instanceSessions.length})
              </button>
              <button onClick={() => setActiveTab("sect_missions")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "sect_missions" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <ListTodo className="w-3.5 h-3.5 inline mr-1.5" />Nhiệm vụ tông môn ({playerSectMissions.length})
              </button>
              <button onClick={() => setActiveTab("tracking_marks")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "tracking_marks" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Eye className="w-3.5 h-3.5 inline mr-1.5" />Ấn truy tung ({playerTrackingMarks.length})
              </button>
              <button onClick={() => setActiveTab("achievements")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "achievements" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Star className="w-3.5 h-3.5 inline mr-1.5" />Thành tựu ({playerAchievements.length})
              </button>
              <button onClick={() => setActiveTab("combat")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "combat" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Swords className="w-3.5 h-3.5 inline mr-1.5" />Chỉ số & Chiến đấu ({detail.equipped_gus?.length || 0})
              </button>
            </div>

            {/* Items Tab */}
            {activeTab === "items" && (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                    <input type="text" value={playerItemsSearch} onChange={e => setPlayerItemsSearch(e.target.value)} onKeyDown={e => { if (e.key === "Enter") setPlayerItemsPage(1); }} placeholder="Tìm vật phẩm trong túi đồ..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setPlayerItemsPage(1); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-bold text-zinc-300 transition-colors cursor-pointer">
                    <Search className="w-3.5 h-3.5" /> Tìm
                  </button>
                  {canAdminAction("grantPlayerInventory") && <button onClick={() => { setAddItemId(""); setAddItemQty(1); setAddItemCategory("materials"); setShowAddItem(true); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>}
                </div>

                {playerItemsLoading ? (
                  <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-2">Đang tải vật phẩm...</span></div>
                ) : playerItems.length === 0 ? (
                  <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                    <Package className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                    <p className="text-xs text-zinc-500">Túi đồ trống</p>
                  </div>
                ) : (
                  <>
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2.5 font-semibold">ID</th>
                          <th className="px-3 py-2.5 font-semibold">Tên</th>
                          <th className="px-3 py-2.5 font-semibold">Loại</th>
                          <th className="px-3 py-2.5 font-semibold">Nhóm</th>
                          <th className="px-3 py-2.5 font-semibold">Tiến độ</th>
                          <th className="px-3 py-2.5 font-semibold">Bậc</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Số lượng</th>
                          <th className="px-3 py-2.5 font-semibold">Tạo / cập nhật</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {playerItems.map((it) => {
                          const spec = itemDictionary[it.item_id];
                          return (
                            <tr key={`${it.item_id}::${it.category}`} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                              <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[100px] truncate" title={it.item_id}>{it.item_id}</td>
                              <td className="px-3 py-2.5 font-semibold text-zinc-200">{it.name || spec?.name || it.item_id}</td>
                              <td className="px-3 py-2.5">
                                <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-zinc-900 border border-zinc-800 text-zinc-400">{translateItemType(it.item_type || spec?.item_type) || "?"}</span>
                              </td>
                              <td className="px-3 py-2.5 text-zinc-400">{it.category || "materials"}</td>
                              <td className="px-3 py-2.5 text-zinc-400">{it.category === "recipe" ? `${it.completion_rate ?? 0}%${it.is_unlocked ? " · mở" : " · khóa"}` : "—"}</td>
                              <td className="px-3 py-2.5 text-zinc-400">{it.rank || spec?.rank || "?"}</td>
                              <td className="px-3 py-2.5 text-right font-mono font-bold text-zinc-200">{it.quantity.toLocaleString()}</td>
                              <td className="px-3 py-2.5 text-zinc-500">{it.created_at ? new Date(it.created_at).toLocaleDateString("vi-VN") : "—"}<span className="block text-[10px]">{it.updated_at ? new Date(it.updated_at).toLocaleDateString("vi-VN") : "—"}</span></td>
                              <td className="px-3 py-2.5 text-right">
                                {it.category !== "recipe" && canAdminAction("updatePlayerInventoryItem") && <button onClick={() => { setInventoryEditQuantity(String(it.quantity)); setInventoryEditCategory(it.category || "materials"); setShowInventoryItemEditor(it); }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 transition-colors cursor-pointer mr-1" title="Sửa số lượng/nhóm"><Edit3 className="w-3.5 h-3.5" /></button>}
                                {it.category === "recipe" && canAdminAction("updatePlayerRecipe") && <button onClick={() => { setRecipeCompletion(String(it.completion_rate ?? 0)); setRecipeUnlocked(Boolean(it.is_unlocked)); setRecipeDeduceCount(String(it.deduce_count ?? 0)); setRecipeFavorite(Boolean(it.favorite)); setShowRecipeEditor(it); }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 transition-colors cursor-pointer mr-1" title="Sửa tiến độ"><Edit3 className="w-3.5 h-3.5" /></button>}
                                {canAdminAction("deletePlayerInventoryItem") && <button onClick={() => { setShowRemoveItem(`${it.item_id}::${it.category}`); setRemoveItemQty(1); }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer" title="Xóa">
                                  <Trash2 className="w-3.5 h-3.5" />
                                </button>}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                  <div className="flex items-center justify-between mt-2 text-xs font-semibold text-zinc-500">
                    <div>Tổng: <span className="text-zinc-300">{playerItemsTotal.toLocaleString()}</span> vật phẩm</div>
                    <div className="flex items-center gap-3">
                      <span>Trang {playerItemsPage}/{Math.max(1, Math.ceil(playerItemsTotal / playerItemsLimit))}</span>
                      <div className="flex gap-2">
                        <button onClick={() => setPlayerItemsPage(Math.max(1, playerItemsPage - 1))} disabled={playerItemsPage <= 1 || playerItemsLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                          <ChevronLeft className="w-4 h-4" /> Trước
                        </button>
                        <button onClick={() => setPlayerItemsPage(playerItemsPage + 1)} disabled={playerItemsPage >= Math.ceil(playerItemsTotal / playerItemsLimit) || playerItemsLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                          Sau <ChevronRight className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  </>
                )}
              </div>
            )}

            {/* Gus Tab */}
            {activeTab === "gus" && (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                    <input type="text" value={playerGusSearch} onChange={e => setPlayerGusSearch(e.target.value)} onKeyDown={e => { if (e.key === "Enter") setPlayerGusPage(1); }} placeholder="Tìm cổ trùng (ID, tên, instance ID)..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setPlayerGusPage(1); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-bold text-zinc-300 transition-colors cursor-pointer">
                    <Search className="w-3.5 h-3.5" /> Tìm
                  </button>
                  {canAdminAction("grantPlayerInventory") && <button onClick={() => { setAddGuId(""); setAddGuRank(""); setAddGuQty(1); setShowAddGu(true); ensureShopSearchLoaded(); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>}
                </div>

                {playerGusLoading ? (
                  <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-2">Đang tải cổ trùng...</span></div>
                ) : playerGus.length === 0 ? (
                  <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                    <Bug className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                    <p className="text-xs text-zinc-500">Không có cổ trùng nào</p>
                  </div>
                ) : (
                  <>
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2.5 font-semibold">Instance ID</th>
                          <th className="px-3 py-2.5 font-semibold">Tên</th>
                          <th className="px-3 py-2.5 font-semibold">Bậc</th>
                          <th className="px-3 py-2.5 font-semibold">Cấp</th>
                          <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                          <th className="px-3 py-2.5 font-semibold">Đói</th>
                          <th className="px-3 py-2.5 font-semibold">Vị trí</th>
                          <th className="px-3 py-2.5 font-semibold">Trang bị</th>
                          <th className="px-3 py-2.5 font-semibold">Bản Mệnh</th>
                          <th className="px-3 py-2.5 font-semibold">Khóa</th>
                          <th className="px-3 py-2.5 font-semibold">Dung hợp / Tăng trưởng</th>
                          <th className="px-3 py-2.5 font-semibold">Cập nhật / Cho ăn</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {playerGus.map((gu) => {
                          const spec = guSpecFor(gu.gu_id, gu.rank);
                          return (
                            <tr key={gu.instance_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                              <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate" title={gu.instance_id}>{gu.instance_id.slice(0, 8)}...</td>
                              <td className="px-3 py-2.5 font-semibold text-zinc-200">{gu.name || spec?.name || gu.gu_id}</td>
                              <td className={`px-3 py-2.5 font-bold ${rankColor(gu.rank || spec?.rank || 1)}`}>{gu.rank || spec?.rank || "?"}</td>
                              <td className="px-3 py-2.5 text-zinc-300">{gu.level}</td>
                              <td className="px-3 py-2.5">
                                <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold border ${statusColor(gu.status)}`}>{statusLabel(gu.status)}</span>
                              </td>
                              <td className="px-3 py-2.5 font-mono text-zinc-300">{gu.hunger}</td>
                              <td className="px-3 py-2.5 text-zinc-400">{gu.location}</td>
                              <td className="px-3 py-2.5">{gu.equipped ? <span className="text-emerald-400 text-[11px] font-bold">✓</span> : <span className="text-zinc-600">–</span>}</td>
                              <td className="px-3 py-2.5">{gu.is_vital ? <Star className="w-3.5 h-3.5 text-amber-400" /> : <span className="text-zinc-600">–</span>}</td>
                              <td className="px-3 py-2.5">{gu.is_locked ? "🔒" : "–"}</td>
                              <td className="px-3 py-2.5 text-zinc-400">{gu.integration_time} / {gu.growth_hours}h</td>
                              <td className="px-3 py-2.5 text-zinc-500">{gu.updated_at ? new Date(gu.updated_at).toLocaleDateString("vi-VN") : "—"}<span className="block text-[10px]">Ăn: {gu.last_fed_at ? new Date(gu.last_fed_at).toLocaleDateString("vi-VN") : "—"}</span></td>
                              <td className="px-3 py-2.5 text-right">
                                {canAdminAction("updatePlayerGu") && <button onClick={() => { setGuEditForm({ rank: String(gu.rank), level: String(gu.level), hunger: String(gu.hunger), status: gu.status, equipped: gu.equipped, is_vital: gu.is_vital, is_locked: gu.is_locked, integration_time: String(gu.integration_time) }); setShowGuEditor(gu); }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 transition-colors cursor-pointer mr-1" title="Sửa cổ trùng"><Edit3 className="w-3.5 h-3.5" /></button>}
                                <button
                                  onClick={async () => {
                                    if (!canAdminAction("deletePlayerGu")) return;
                                    const endpoint = `${BACKEND_URL}/api/admin/players/${selectedId}/gus/${gu.instance_id}`;
                                    const prepared = prepareAdminAction("deletePlayerGu", "xóa cổ trùng", endpoint);
                                    if (!prepared) return;
                                    await exec(endpoint, "DELETE", prepared);
                                  }}
                                  className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer"
                                  title="Xóa"
                                  disabled={!canAdminAction("deletePlayerGu")}
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
                  <div className="flex items-center justify-between mt-2 text-xs font-semibold text-zinc-500">
                    <div>Tổng: <span className="text-zinc-300">{playerGusTotal.toLocaleString()}</span> cổ trùng</div>
                    <div className="flex items-center gap-3">
                      <span>Trang {playerGusPage}/{Math.max(1, Math.ceil(playerGusTotal / playerGusLimit))}</span>
                      <div className="flex gap-2">
                        <button onClick={() => setPlayerGusPage(Math.max(1, playerGusPage - 1))} disabled={playerGusPage <= 1 || playerGusLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                          <ChevronLeft className="w-4 h-4" /> Trước
                        </button>
                        <button onClick={() => setPlayerGusPage(playerGusPage + 1)} disabled={playerGusPage >= Math.ceil(playerGusTotal / playerGusLimit) || playerGusLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                          Sau <ChevronRight className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  </>
                )}
              </div>
            )}

            {/* Pets Tab */}
            {activeTab === "pets" && (
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                    <input type="text" value={playerPetsSearch} onChange={e => setPlayerPetsSearch(e.target.value)} onKeyDown={e => { if (e.key === "Enter") setPlayerPetsPage(1); }} placeholder="Tìm sủng vật..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setPlayerPetsPage(1); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-bold text-zinc-300 transition-colors cursor-pointer">
                    <Search className="w-3.5 h-3.5" /> Tìm
                  </button>
                </div>

                {playerPetsLoading ? (
                  <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-purple-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-2">Đang tải sủng vật...</span></div>
                ) : playerPets.length === 0 ? (
                  <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                    <PawPrint className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                    <p className="text-xs text-zinc-500">Không có sủng vật nào</p>
                  </div>
                ) : (
                  <>
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="border-b border-zinc-800/60 text-zinc-500">
                          <th className="px-3 py-2 font-mono font-semibold">ID</th>
                          <th className="px-3 py-2 font-mono font-semibold">Monster</th>
                          <th className="px-3 py-2 font-mono font-semibold">Rank</th>
                          <th className="px-3 py-2 font-mono font-semibold">HP</th>
                          <th className="px-3 py-2 font-mono font-semibold">ATK</th>
                          <th className="px-3 py-2 font-mono font-semibold">DEF</th>
                          <th className="px-3 py-2 font-mono font-semibold">No</th>
                          <th className="px-3 py-2 font-mono font-semibold">Trạng thái</th>
                          <th className="px-3 py-2 font-mono font-semibold">Formation</th>
                          <th className="px-3 py-2 font-mono font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-800/40">
                        {playerPets.map((pet: ApiResponseData) => (
                          <tr key={pet.id} className="hover:bg-zinc-900/30 transition-colors">
                            <td className="px-3 py-2 font-mono text-zinc-400">{pet.id.substring(0, 8)}...</td>
                            <td className="px-3 py-2 font-bold text-zinc-200">{pet.monster_name || pet.monster_template_id}</td>
                            <td className="px-3 py-2 text-zinc-300">Rank {pet.rank}</td>
                            <td className="px-3 py-2 text-red-400">{Math.round(pet.current_hp)}/{Math.round(pet.max_hp)}</td>
                            <td className="px-3 py-2 text-amber-400">{Math.round(pet.attack)}</td>
                            <td className="px-3 py-2 text-blue-400">{Math.round(pet.defense)}</td>
                            <td className="px-3 py-2 text-purple-400">{Math.round(pet.hunger)}%</td>
                            <td className="px-3 py-2">
                              <span className={`px-2 py-0.5 rounded text-[10px] font-bold font-mono ${pet.status === "active" ? "bg-emerald-900/40 text-emerald-400" : "bg-zinc-800 text-zinc-500"}`}>
                                {pet.status === "active" ? "Hoạt động" : "Ngủ"}
                              </span>
                            </td>
                            <td className="px-3 py-2 text-zinc-300">{pet.formation_slot === 0 ? "Storage" : `Slot ${pet.formation_slot}`}</td>
                            <td className="px-3 py-2 text-right">
                              <button
                                onClick={async () => {
                                  if (!canAdminAction("deletePlayerPet")) return;
                                  const endpoint = `${BACKEND_URL}/api/admin/players/${selectedId}/pets/${pet.id}`;
                                  const prepared = prepareAdminAction("deletePlayerPet", "xóa sủng vật", `${pet.monster_name || pet.monster_template_id} · ${pet.id}`);
                                  if (!prepared) return;
                                  await exec(endpoint, "DELETE", prepared);
                                }}
                                className="px-2 py-1 rounded bg-red-900/40 hover:bg-red-800 text-red-400 text-[10px] font-bold transition-colors cursor-pointer"
                              >
                                <Trash2 className="w-3 h-3 inline" />
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  <div className="flex items-center justify-between text-[11px] text-zinc-500 font-mono">
                    <span>Trang {playerPetsPage} / {Math.max(1, Math.ceil(playerPetsTotal / playerPetsLimit))} • Tổng: {playerPetsTotal}</span>
                    <div className="flex gap-2">
                      <button onClick={() => setPlayerPetsPage(Math.max(1, playerPetsPage - 1))} disabled={playerPetsPage <= 1 || playerPetsLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        <ChevronLeft className="w-4 h-4" /> Trước
                      </button>
                      <button onClick={() => setPlayerPetsPage(playerPetsPage + 1)} disabled={playerPetsPage >= Math.ceil(playerPetsTotal / playerPetsLimit) || playerPetsLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        Sau <ChevronRight className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                  </>
                )}
              </div>
            )}

            {/* Quests Tab */}
            {activeTab === "quests" && (
              <div className="space-y-3">
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]"><th className="px-3 py-2.5">Quest ID</th><th className="px-3 py-2.5">Trạng thái</th><th className="px-3 py-2.5">Progress data</th><th className="px-3 py-2.5">Hoàn thành</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{playerQuests.map(q => <tr key={q.quest_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30"><td className="px-3 py-2.5 text-zinc-300 font-mono">{q.quest_id}</td><td className="px-3 py-2.5"><span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-zinc-900 border border-zinc-800 text-zinc-400">{q.status}</span></td><td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[360px] truncate" title={JSON.stringify(q.progress_data)}>{JSON.stringify(q.progress_data)}</td><td className="px-3 py-2.5 text-zinc-500">{q.completed_at ? new Date(q.completed_at).toLocaleString("vi-VN") : "—"}</td><td className="px-3 py-2.5 text-right">{canAdminAction("updatePlayerQuest") && <button onClick={() => { setQuestStatus(q.status); setQuestProgressData(JSON.stringify(q.progress_data || {}, null, 2)); setQuestCompletedAt(q.completed_at ? new Date(q.completed_at).toISOString().slice(0, 16) : ""); setShowQuestEditor(q); }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 mr-1 cursor-pointer"><Edit3 className="w-3.5 h-3.5" /></button>}{canAdminAction("deletePlayerQuest") && <button onClick={async () => { const endpoint=`${BACKEND_URL}/api/admin/players/${selectedId}/quests/${encodeURIComponent(q.quest_id)}`; const prepared=prepareAdminAction("deletePlayerQuest","xóa tiến độ nhiệm vụ",endpoint); if(!prepared)return; await exec(endpoint,"DELETE",prepared); }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3.5 h-3.5" /></button>}</td></tr>)}</tbody></table></div>
              </div>
            )}

            {/* Instance Sessions Tab */}
            {activeTab === "instances" && (
              <div className="space-y-3">
                {instanceSessionsLoading ? <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /><span className="text-xs text-zinc-500 font-mono mt-2">Đang tải phiên bí cảnh...</span></div> : instanceSessions.length === 0 ? <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl"><Globe className="w-8 h-8 text-zinc-700 mx-auto mb-2" /><p className="text-xs text-zinc-500">Người chơi chưa có phiên bí cảnh</p></div> : <div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]"><th className="px-3 py-2.5">Session ID</th><th className="px-3 py-2.5">Instance</th><th className="px-3 py-2.5">Stage</th><th className="px-3 py-2.5">Status</th><th className="px-3 py-2.5">Inventory</th><th className="px-3 py-2.5">Tạo lúc</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{instanceSessions.map(session => <tr key={session.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30"><td className="px-3 py-2.5 font-mono text-zinc-500" title={session.id}>{session.id.slice(0, 8)}…</td><td className="px-3 py-2.5 text-zinc-200">{session.instance_id}</td><td className="px-3 py-2.5 font-mono text-zinc-300">{session.current_stage}</td><td className="px-3 py-2.5 text-zinc-300">{session.status}</td><td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[240px] truncate" title={JSON.stringify(session.temp_inventory)}>{JSON.stringify(session.temp_inventory)}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(session.created_at).toLocaleString("vi-VN")}</td><td className="px-3 py-2.5 text-right">{canAdminAction("updatePlayerInstanceSession") && <button onClick={() => { setInstanceSessionStage(String(session.current_stage)); setInstanceSessionStatus(session.status); setInstanceSessionInventory(JSON.stringify(session.temp_inventory || [], null, 2)); setShowInstanceSessionEditor(session); }} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 mr-1 cursor-pointer"><Edit3 className="w-3.5 h-3.5" /></button>}{canAdminAction("deletePlayerInstanceSession") && <button onClick={async () => { const endpoint=`${BACKEND_URL}/api/admin/players/${selectedId}/instance-sessions/${session.id}`;const prepared=prepareAdminAction("deletePlayerInstanceSession","xóa phiên bí cảnh",endpoint);if(!prepared)return;await exec(endpoint,"DELETE",prepared); }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3.5 h-3.5" /></button>}</td></tr>)}</tbody></table></div>}
              </div>
            )}

            {activeTab === "sect_missions" && (
              <div className="space-y-3">
                {playerSectMissionsLoading ? <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin"/><span className="text-xs text-zinc-500 mt-2">Đang tải nhiệm vụ tông môn...</span></div> : playerSectMissions.length === 0 ? <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl"><ListTodo className="w-8 h-8 text-zinc-700 mx-auto mb-2"/><p className="text-xs text-zinc-500">Người chơi chưa có tiến độ nhiệm vụ tông môn</p></div> : <div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]"><th className="px-3 py-2.5">Nhiệm vụ / Tông môn</th><th className="px-3 py-2.5">Mục tiêu</th><th className="px-3 py-2.5">Tiến độ</th><th className="px-3 py-2.5">Trạng thái</th><th className="px-3 py-2.5">Nhận lúc</th><th className="px-3 py-2.5">Hạn</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{playerSectMissions.map(m=><tr key={m.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30"><td className="px-3 py-2.5 text-zinc-200">{m.title}<span className="block text-[10px] text-zinc-500">{m.sect_name} · {m.mission_id}</span></td><td className="px-3 py-2.5 text-zinc-400">{m.mission_type}: {m.target_id} ({m.target_amount}) · thưởng {m.reward_contribution}</td><td className="px-3 py-2.5 font-mono text-zinc-300">{m.current_progress}/{m.target_amount}</td><td className="px-3 py-2.5 text-zinc-300">{m.status}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(m.assigned_at).toLocaleString("vi-VN")}</td><td className="px-3 py-2.5 text-zinc-500">{m.expires_at ? new Date(m.expires_at).toLocaleString("vi-VN") : "Không hạn"}</td><td className="px-3 py-2.5 text-right">{canAdminAction("updatePlayerSectMission") && <button onClick={()=>{setPlayerSectMissionProgress(String(m.current_progress));setPlayerSectMissionStatus(m.status);setPlayerSectMissionExpiry(m.expires_at ? new Date(m.expires_at).toISOString().slice(0,16) : "");setShowPlayerSectMissionEditor(m);}} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 cursor-pointer" title="Sửa tiến độ"><Edit3 className="w-3.5 h-3.5"/></button>}</td></tr>)}</tbody></table></div>}
              </div>
            )}

            {activeTab === "tracking_marks" && (
              <div className="space-y-3">
                {playerTrackingMarksLoading ? <div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin"/><span className="text-xs text-zinc-500 mt-2">Đang tải ấn truy tung...</span></div> : playerTrackingMarks.length===0 ? <div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl"><Eye className="w-8 h-8 text-zinc-700 mx-auto mb-2"/><p className="text-xs text-zinc-500">Không có ấn truy tung liên quan</p></div> : <div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]"><th className="px-3 py-2.5">Tracker</th><th className="px-3 py-2.5">Mục tiêu</th><th className="px-3 py-2.5">Loại ấn</th><th className="px-3 py-2.5">Uy lực</th><th className="px-3 py-2.5">Tạo lúc</th><th className="px-3 py-2.5">Hết hạn</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{playerTrackingMarks.map(mark=><tr key={mark.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30"><td className="px-3 py-2.5 text-zinc-300">{mark.tracker_username}<span className="block text-[10px] text-zinc-600 font-mono">{mark.tracker_id}</span></td><td className="px-3 py-2.5 text-zinc-300">{mark.target_username}<span className="block text-[10px] text-zinc-600 font-mono">{mark.target_id}</span></td><td className="px-3 py-2.5 text-zinc-300">{mark.mark_type}</td><td className="px-3 py-2.5 text-zinc-300">{mark.power}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(mark.created_at).toLocaleString("vi-VN")}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(mark.expires_at).toLocaleString("vi-VN")}</td><td className="px-3 py-2.5 text-right">{canAdminAction("updatePlayerTrackingMark")&&<button onClick={()=>{setTrackingMarkType(mark.mark_type);setTrackingMarkPower(String(mark.power));setTrackingMarkExpiry(new Date(mark.expires_at).toISOString().slice(0,16));setShowTrackingMarkEditor(mark);}} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 mr-1 cursor-pointer"><Edit3 className="w-3.5 h-3.5"/></button>}{canAdminAction("deletePlayerTrackingMark")&&<button onClick={async()=>{const endpoint=`${BACKEND_URL}/api/admin/players/${selectedId}/tracking-marks/${mark.id}`;const prepared=prepareAdminAction("deletePlayerTrackingMark","thu hồi ấn truy tung",endpoint);if(!prepared)return;await exec(endpoint,"DELETE",prepared);}} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3.5 h-3.5"/></button>}</td></tr>)}</tbody></table></div>}
              </div>
            )}

            {activeTab === "achievements" && (
              <div className="space-y-3">
                {canAdminAction("grantPlayerAchievement")&&availableAchievements.length>0&&<div className="flex gap-2"><select value={achievementToGrant} onChange={e=>setAchievementToGrant(e.target.value)} className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200">{availableAchievements.map(a=><option key={String(a.id)} value={String(a.id)}>{String(a.name)} · {String(a.id)}</option>)}</select><button onClick={async()=>{const endpoint=`${BACKEND_URL}/api/admin/players/${selectedId}/achievements`;const prepared=prepareAdminAction("grantPlayerAchievement","cấp thành tựu",endpoint);if(!prepared)return;await exec(endpoint,"POST",{achievement_id:achievementToGrant,...prepared});}} className="px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 cursor-pointer"><Plus className="w-3.5 h-3.5 inline mr-1"/>Cấp</button></div>}
                {playerAchievementsLoading?<div className="flex flex-col items-center py-12"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin"/><span className="text-xs text-zinc-500 mt-2">Đang tải thành tựu...</span></div>:playerAchievements.length===0?<div className="text-center py-12 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl"><Star className="w-8 h-8 text-zinc-700 mx-auto mb-2"/><p className="text-xs text-zinc-500">Người chơi chưa mở khóa thành tựu</p></div>:<div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]"><th className="px-3 py-2.5">Mã thành tựu</th><th className="px-3 py-2.5">Danh hiệu</th><th className="px-3 py-2.5">Khung</th><th className="px-3 py-2.5">Mở khóa lúc</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{playerAchievements.map(a=><tr key={a.achievement_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30"><td className="px-3 py-2.5 font-mono text-zinc-300">{a.achievement_id}</td><td className="px-3 py-2.5 text-zinc-300">{a.selected_title||"—"}</td><td className="px-3 py-2.5 text-zinc-400">{a.selected_frame||"—"}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(a.unlocked_at).toLocaleString("vi-VN")}</td><td className="px-3 py-2.5 text-right">{canAdminAction("updatePlayerAchievement")&&<button onClick={()=>{setAchievementTitle(a.selected_title);setAchievementFrame(a.selected_frame);setAchievementUnlockedAt(new Date(a.unlocked_at).toISOString().slice(0,16));setShowAchievementEditor(a);}} className="p-1 rounded-lg hover:bg-amber-950/40 text-zinc-500 hover:text-amber-400 mr-1 cursor-pointer"><Edit3 className="w-3.5 h-3.5"/></button>}{canAdminAction("deletePlayerAchievement")&&<button onClick={async()=>{const endpoint=`${BACKEND_URL}/api/admin/players/${selectedId}/achievements/${encodeURIComponent(a.achievement_id)}`;const prepared=prepareAdminAction("deletePlayerAchievement","thu hồi thành tựu",endpoint);if(!prepared)return;await exec(endpoint,"DELETE",prepared);}} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3.5 h-3.5"/></button>}</td></tr>)}</tbody></table></div>}
              </div>
            )}

            {/* Combat & Player Stats Tab */}
            {activeTab === "combat" && (
              <div className="space-y-5">
                {/* 1. Equipped Gu List */}
                <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
                  <button
                    type="button"
                    onClick={() => setIsCombatEquippedExpanded((prev) => !prev)}
                    className="flex w-full items-center gap-2 border-b border-zinc-800/80 pb-2.5 text-left cursor-pointer"
                  >
                    {isCombatEquippedExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                    <Bug className="w-5 h-5 text-amber-400" />
                    <span className="font-serif font-bold text-zinc-100 text-sm">Cổ Trùng Đang Trang Bị</span>
                    <span className="ml-auto text-xs px-2.5 py-0.5 rounded-full bg-amber-950/40 text-amber-400 border border-amber-800/40 font-mono font-semibold">
                      {detail.equipped_gus?.length || 0} Cổ
                    </span>
                  </button>

                  {isCombatEquippedExpanded && (
                    <>
                      {!detail.equipped_gus || detail.equipped_gus.length === 0 ? (
                        <div className="text-center py-8 bg-zinc-950/30 border border-dashed border-zinc-800/80 rounded-xl">
                          <Bug className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                          <p className="text-xs text-zinc-500 italic">Người chơi hiện chưa trang bị Cổ trùng nào trong Không Khiếu.</p>
                        </div>
                      ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                          {detail.equipped_gus.map((gu) => {
                            const spec = guSpecFor(gu.gu_id, gu.rank);
                            const displayName = gu.name || spec?.name || gu.gu_id;

                            return (
                              <div
                                key={gu.instance_id}
                                className={`bg-zinc-950/50 border rounded-xl p-3.5 flex flex-col gap-2.5 transition-all ${
                                  gu.is_vital ? "border-amber-500/40 shadow-[0_0_15px_rgba(245,158,11,0.1)]" : "border-zinc-800 hover:border-zinc-700"
                                }`}
                              >
                                <div className="flex items-start gap-3">
                                  <div className={`w-11 h-11 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center font-bold text-base shrink-0 ${rankColor(gu.rank)}`}>
                                    {displayName.slice(0, 2)}
                                  </div>
                                  <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-1.5 flex-wrap">
                                      <span className={`text-sm font-bold font-serif truncate ${rankColor(gu.rank)}`}>
                                        {displayName}
                                      </span>
                                      {gu.is_vital && (
                                        <span className="px-1.5 py-0.5 rounded text-[9px] font-bold bg-amber-950/60 text-amber-300 border border-amber-800/60 flex items-center gap-1 shrink-0">
                                          <Star className="w-2.5 h-2.5 text-amber-400 fill-amber-400" />
                                          Bản Mệnh
                                        </span>
                                      )}
                                    </div>
                                    <div className="text-[11px] text-zinc-500 font-mono mt-0.5 flex items-center gap-2">
                                      <span>Rank {gu.rank}</span>
                                      <span>•</span>
                                      <span className={`px-1.5 py-0.2 rounded text-[9px] font-bold border ${statusColor(gu.status)}`}>
                                        {statusLabel(gu.status)}
                                      </span>
                                    </div>
                                  </div>
                                </div>

                                <div className="grid grid-cols-2 gap-2 text-[10px] font-mono border-t border-zinc-900 pt-2 text-zinc-400">
                                  <div>Độ no: <span className="text-zinc-200 font-bold">{gu.hunger}/100</span></div>
                                  <div>
                                    Lưu phái:{" "}
                                    <span className="text-amber-400 font-semibold">
                                      {gu.path && gu.path.length > 0 ? gu.path.map((p) => translatePath(p)).join(", ") : "Chưa rõ"}
                                    </span>
                                  </div>
                                  {gu.growth_hours && gu.growth_hours > 0 ? (
                                    <div className="col-span-2 text-emerald-400">
                                      Đã nuôi dưỡng: <strong>{gu.growth_hours.toFixed(1)} giờ</strong>
                                    </div>
                                  ) : null}
                                </div>

                                {/* Canonical Effect Block v2 summary */}
                                {spec?.effects && (
                                  <div className="bg-zinc-900/60 rounded-lg p-2">
                                    <EffectEnvelopeView effects={spec.effects} />
                                  </div>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </>
                  )}
                </div>

                {/* 2. Combat & Passive Stats Breakdown */}
                <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
                  <button
                    type="button"
                    onClick={() => setIsCombatStatsExpanded((prev) => !prev)}
                    className="flex w-full items-center gap-2 border-b border-zinc-800/80 pb-2.5 text-left cursor-pointer"
                  >
                    {isCombatStatsExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                    <Swords className="w-5 h-5 text-emerald-400" />
                    <span className="font-serif font-bold text-zinc-100 text-sm">Tổng Hợp Chỉ Số Chiến Đấu & Bị Động</span>
                  </button>

                  {isCombatStatsExpanded && (() => {
                    const ps = detail.passive_stats;
                    const statLabelsMap: Record<string, string> = {
                      attack: "Công Kích",
                      defense: "Phòng Ngự",
                      hp: "Máu Tối Đa",
                      speed: "Tốc Độ",
                      healing: "Hồi Phục",
                      soul: "Hồn Lực",
                      storage: "Vật Chứa",
                      movement: "Di Chuyển (ô)",
                      investigation: "Trinh Sát",
                      hearing: "Thính Giác",
                      mana_regen_mult: "Hồi Pháp",
                      damage_multiplier: "Sát Thương Khi Mất Máu",
                      refine_mana_tier: "Tinh Luyện",
                      cleansing_power: "Tẩy Trừ",
                      path_conflict_reduction: "Giảm Xung Đột (%)",
                    };

                    const details = (ps?.details || []).filter((d) => d.total !== 0);

                    // Deduplicate details by key
                    const dedupedMap = new Map<string, PassiveStatDetail>();
                    for (const d of details) {
                      const existing = dedupedMap.get(d.key);
                      if (existing) {
                        existing.total += d.total;
                        existing.sources = [...existing.sources, ...d.sources];
                      } else {
                        dedupedMap.set(d.key, { ...d, sources: [...d.sources] });
                      }
                    }
                    const dedupedDetails = Array.from(dedupedMap.values());

                    // Quick Combat Cards values
                    const totalAtk = ps?.stats?.attack ?? dedupedMap.get("attack")?.total ?? (10 + (detail.rank - 1) * 10 + (detail.sub_rank - 1) * 2);
                    const totalDef = ps?.stats?.defense ?? dedupedMap.get("defense")?.total ?? (5 + (detail.rank - 1) * 5 + (detail.sub_rank - 1) * 1);
                    const totalSpd = ps?.stats?.speed ?? dedupedMap.get("speed")?.total ?? (10 + (detail.rank - 1) * 2 + (detail.sub_rank - 1) * 0.5);
                    const totalHP = detail.base_hp || 100;

                    return (
                      <div className="space-y-4">
                        {/* 4 Core Combat Cards */}
                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                          <div className="bg-zinc-950/60 p-3.5 rounded-xl border border-red-900/30">
                            <div className="flex items-center gap-1.5 text-[10px] text-red-400 font-mono font-bold uppercase tracking-wider mb-1">
                              <Sword className="w-3.5 h-3.5" /> Tổng Công Kích
                            </div>
                            <div className="text-xl font-bold text-red-300 font-mono">
                              {formatStat(totalAtk)}
                            </div>
                          </div>
                          <div className="bg-zinc-950/60 p-3.5 rounded-xl border border-blue-900/30">
                            <div className="flex items-center gap-1.5 text-[10px] text-blue-400 font-mono font-bold uppercase tracking-wider mb-1">
                              <Shield className="w-3.5 h-3.5" /> Tổng Phòng Ngự
                            </div>
                            <div className="text-xl font-bold text-blue-300 font-mono">
                              {formatStat(totalDef)}
                            </div>
                          </div>
                          <div className="bg-zinc-950/60 p-3.5 rounded-xl border border-amber-900/30">
                            <div className="flex items-center gap-1.5 text-[10px] text-amber-400 font-mono font-bold uppercase tracking-wider mb-1">
                              <Zap className="w-3.5 h-3.5" /> Tổng Tốc Độ
                            </div>
                            <div className="text-xl font-bold text-amber-300 font-mono">
                              {formatStat(totalSpd)}
                            </div>
                          </div>
                          <div className="bg-zinc-950/60 p-3.5 rounded-xl border border-emerald-900/30">
                            <div className="flex items-center gap-1.5 text-[10px] text-emerald-400 font-mono font-bold uppercase tracking-wider mb-1">
                              <Heart className="w-3.5 h-3.5" /> Sinh Mệnh (HP)
                            </div>
                            <div className="text-xl font-bold text-emerald-300 font-mono">
                              {Math.round(detail.hp)} / {Math.round(totalHP)}
                            </div>
                          </div>
                        </div>

                        {/* Detailed Passive Stats Grid with Source Attribution */}
                        <div>
                          <h4 className="text-xs font-mono font-semibold text-zinc-400 mb-2.5 uppercase tracking-wider">
                            Chi Tiết Đóng Góp Từng Nguồn
                          </h4>
                          {dedupedDetails.length === 0 ? (
                            <div className="text-xs text-zinc-500 italic py-3">Không có chỉ số bị động bổ sung nào.</div>
                          ) : (
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                              {dedupedDetails.map((d) => (
                                <div key={d.key} className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-800/80 flex flex-col justify-between">
                                  <div className="flex items-center justify-between">
                                    <span className="text-[11px] text-zinc-400 font-mono">{statLabelsMap[d.key] || d.key}</span>
                                    <span className="text-base font-bold text-zinc-100 font-mono">+{formatStat(d.total)}</span>
                                  </div>
                                  {d.sources && d.sources.length > 0 && (
                                    <div className="mt-2 pt-2 border-t border-zinc-900 text-[10px] text-zinc-500 space-y-1">
                                      {d.sources.map((s, idx) => (
                                        <div key={idx} className="flex items-center justify-between text-zinc-400">
                                          <span className="truncate max-w-[150px]">{s.name}</span>
                                          <span className="font-mono text-zinc-300 font-semibold">+{formatStat(s.final)}</span>
                                        </div>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })()}
                </div>

                {/* 3. Growth Boosts & Fully Grown Gus */}
                <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
                  <button
                    type="button"
                    onClick={() => setIsCombatGrowthExpanded((prev) => !prev)}
                    className="flex w-full items-center gap-2 border-b border-zinc-800/80 pb-2.5 text-left cursor-pointer"
                  >
                    {isCombatGrowthExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                    <TrendingUp className="w-5 h-5 text-emerald-400" />
                    <span className="font-serif font-bold text-zinc-100 text-sm">Cốt Tủy Không Khiếu (Tăng Trưởng Vĩnh Viễn)</span>
                  </button>

                  {isCombatGrowthExpanded && (() => {
                    const gd = detail.growth_data;
                    const b = gd?.boosts;
                    const fullyGrown = gd?.fully_grown || [];
                    const hasBoosts = b && (b.attack > 0 || b.defense > 0 || b.hp > 0 || b.speed > 0);

                    return (
                      <div className="space-y-4">
                        {!hasBoosts ? (
                          <div className="text-xs text-zinc-500 italic py-2">Chưa có tăng trưởng cốt tủy nào.</div>
                        ) : (
                          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                            {b.attack > 0 && (
                              <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                                <div className="text-[10px] text-zinc-500 font-mono">Công Kích</div>
                                <div className="text-lg font-bold text-zinc-100 font-mono">+{b.attack.toFixed(2)}</div>
                              </div>
                            )}
                            {b.defense > 0 && (
                              <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                                <div className="text-[10px] text-zinc-500 font-mono">Phòng Ngự</div>
                                <div className="text-lg font-bold text-zinc-100 font-mono">+{b.defense.toFixed(2)}</div>
                              </div>
                            )}
                            {b.hp > 0 && (
                              <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                                <div className="text-[10px] text-zinc-500 font-mono">Máu Tối Đa</div>
                                <div className="text-lg font-bold text-zinc-100 font-mono">+{b.hp.toFixed(2)}</div>
                              </div>
                            )}
                            {b.speed > 0 && (
                              <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                                <div className="text-[10px] text-zinc-500 font-mono">Tốc Độ</div>
                                <div className="text-lg font-bold text-zinc-100 font-mono">+{b.speed.toFixed(2)}</div>
                              </div>
                            )}
                          </div>
                        )}

                        {/* Fully Grown Gus */}
                        {fullyGrown.length > 0 && (
                          <div className="space-y-2 pt-2 border-t border-zinc-800/60">
                            <h4 className="text-xs font-mono font-semibold text-zinc-400 uppercase tracking-wider">
                              Cổ Trùng Đã Nuôi Dưỡng Hoàn Tất ({fullyGrown.length})
                            </h4>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                              {fullyGrown.map((fg, i) => {
                                const spec = guSpecFor(fg.gu_id, fg.rank);
                                const parts: string[] = [];
                                if (fg.attack > 0) parts.push(`Công +${fg.attack}`);
                                if (fg.defense > 0) parts.push(`Thủ +${fg.defense}`);
                                if (fg.hp > 0) parts.push(`Máu +${fg.hp}`);
                                if (fg.speed > 0) parts.push(`Tốc +${fg.speed}`);

                                return (
                                  <div key={i} className="flex items-center gap-3 bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-800/60">
                                    <div className="w-8 h-8 rounded-lg bg-zinc-900 flex items-center justify-center text-xs font-bold text-emerald-400">
                                      {(spec?.name || fg.gu_id).slice(0, 2)}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                      <div className="text-xs font-bold text-zinc-200 font-mono truncate">
                                        {spec?.name || fg.gu_id} <span className="text-zinc-500 text-[10px]">(Rank {fg.rank})</span>
                                      </div>
                                      <div className="text-[10px] text-emerald-400 font-mono">{parts.join(" | ")}</div>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })()}
                </div>

                {/* 4. Identity, Constitution & Statuses */}
                <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
                  <button
                    type="button"
                    onClick={() => setIsCombatIdentityExpanded((prev) => !prev)}
                    className="flex w-full items-center gap-2 border-b border-zinc-800/80 pb-2.5 text-left cursor-pointer"
                  >
                    {isCombatIdentityExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                    <User className="w-5 h-5 text-amber-400" />
                    <span className="font-serif font-bold text-zinc-100 text-sm">Thân Phận, Thể Chất & Trạng Thái</span>
                  </button>

                  {isCombatIdentityExpanded && (() => {
                    const raceLabel = (key: string): string => raceDictionary?.[key]?.name || raceLabels[key] || key;
                    const raceEffects = detail.race ? raceDictionary?.[detail.race]?.effects : undefined;
                    const constitutionLabel = (key: string): string => constitutionDictionary?.[key]?.name || constitutionLabels[key] || key;
                    const constitutionEffects = detail.constitution ? constitutionDictionary?.[detail.constitution]?.effects : undefined;

                    return (
                      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                        {/* Race */}
                        <div className="p-3.5 rounded-xl border bg-zinc-950/40 border-zinc-800/80 flex flex-col gap-1">
                          <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Chủng Tộc</div>
                          <div className="text-base font-bold text-zinc-100 font-mono">{raceLabel(detail.race) || "Nhân Tộc"}</div>
                          {raceEffects && <EffectEnvelopeView effects={raceEffects} />}
                        </div>

                        {/* Gender */}
                        <div className="p-3.5 rounded-xl border bg-zinc-950/40 border-zinc-800/80 flex flex-col gap-1">
                          <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Giới Tính</div>
                          <div className="text-base font-bold text-zinc-100 font-mono">{genderLabels[detail.gender] || detail.gender || "Không rõ"}</div>
                        </div>

                        {/* Constitution */}
                        <div className="p-3.5 rounded-xl border bg-zinc-950/40 border-zinc-800/80 flex flex-col gap-1">
                          <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Thể Chất</div>
                          <div className="text-base font-bold text-zinc-100 font-mono">{constitutionLabel(detail.constitution) || "Không có"}</div>
                          {constitutionEffects && <EffectEnvelopeView effects={constitutionEffects} />}
                        </div>

                        {/* Statuses */}
                        <div className="p-3.5 rounded-xl border bg-zinc-950/40 border-zinc-800/80 flex flex-col gap-1">
                          <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Trạng Thái</div>
                          {detail.statuses && detail.statuses.length > 0 ? (
                            <div className="flex flex-col gap-1.5 mt-1">
                              {detail.statuses.map((s: string | { id: string; expires_at?: string | null }) => {
                                const sId = typeof s === "string" ? s : s.id;
                                const sExp = typeof s === "string" ? undefined : s.expires_at;
                                let displayName = statusDictionary?.[sId]?.name || statusLabels[sId] || sId;
                                if (sId.startsWith("cd_gu_")) displayName = "Hồi chiêu Cổ Trùng";
                                const statusEffects = statusDictionary?.[sId]?.effects;

                                return (
                                  <div key={sId} className="flex flex-col gap-1">
                                    <span className="text-[10px] px-2 py-0.5 rounded-lg bg-rose-950/40 text-rose-400 border border-rose-900/40 font-mono font-semibold w-fit">
                                      {displayName}{statusRemainLabel(sExp)}
                                    </span>
                                    {statusEffects && <EffectEnvelopeView effects={statusEffects} />}
                                  </div>
                                );
                              })}
                            </div>
                          ) : (
                            <div className="text-base font-bold text-zinc-100 font-mono">Bình thường</div>
                          )}
                        </div>
                      </div>
                    );
                  })()}
                </div>

                {/* 5. Path Attainments & Dao Marks */}
                <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
                  <button
                    type="button"
                    onClick={() => setIsCombatPathExpanded((prev) => !prev)}
                    className="flex w-full items-center gap-2 border-b border-zinc-800/80 pb-2.5 text-left cursor-pointer"
                  >
                    {isCombatPathExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                    <Crown className="w-5 h-5 text-amber-400" />
                    <span className="font-serif font-bold text-zinc-100 text-sm">Cảnh Giới Lưu Phái & Đạo Ngân</span>
                  </button>

                  {isCombatPathExpanded && (
                    <div className="space-y-4">
                      {/* Path Masteries */}
                      <div>
                        <h4 className="text-xs font-mono font-semibold text-zinc-400 mb-2.5 uppercase tracking-wider">
                          Cảnh Giới Lưu Phái
                        </h4>
                        {detail.path_masteries && Object.keys(detail.path_masteries).length > 0 ? (
                          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                            {Object.entries(detail.path_masteries).map(([path, points]) => {
                              const level = masteryKeyForPoints(points as number);
                              return (
                                <div key={path} className="flex flex-col gap-1.5 bg-zinc-950/40 p-3 rounded-xl border border-zinc-800/80">
                                  <div className="flex justify-between items-center">
                                    <span className="text-zinc-200 font-mono text-xs font-bold">{translatePath(path)}</span>
                                    <span className={`px-2 py-0.5 rounded text-[9px] font-bold font-mono ${
                                      level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                                      level === "Great Grandmaster" ? "bg-red-950/50 text-red-300 border border-red-800/40" :
                                      level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                                      level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                                      "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                                    }`}>
                                      {translateMastery(level)}
                                    </span>
                                  </div>
                                  <div className="text-[10px] text-zinc-500 font-mono">
                                    {points} điểm Chân Ý
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        ) : (
                          <div className="text-xs text-zinc-500 italic py-2">Chưa ngộ chân ý bất kỳ lưu phái nào.</div>
                        )}
                      </div>

                    </div>
                  )}
                </div>
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
                  const body: ApiResponseData = {};
                  if (stonesPrimeval) body.primeval_stones = parseFloat(stonesPrimeval);
                  if (stonesImmortal) body.immortal_stones = parseFloat(stonesImmortal);
                  if (Object.keys(body).length === 0) return;
				  const prepared = prepareAdminAction("adjustPlayerResources", `điều chỉnh tài nguyên của ${detail.username}`, selectedId || "player");
				  if (!prepared) return;
				  Object.assign(body, prepared);
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/stones`, "PUT", body);
                  if (ok) setShowStones(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Lưu"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Moderation Modal */}
        {showModeration && detail && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Khóa / Hạn chế người chơi</h3>
                <button onClick={() => setShowModeration(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <p className="text-xs text-zinc-500 font-mono">{detail.username} · {detail.id}</p>
              <div className="space-y-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Lý do (bắt buộc)</label>
                  <textarea value={modReason} onChange={e => setModReason(e.target.value)} rows={2} placeholder="Vi phạm gì..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex items-end gap-3">
                  <div className="flex-1">
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Thời hạn (giờ)</label>
                    <input type="number" min={1} value={modHours} disabled={modPermanent} onChange={e => setModHours(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50 disabled:opacity-50" />
                  </div>
                  <label className="flex items-center gap-1.5 text-xs text-zinc-400 pb-2">
                    <input type="checkbox" checked={modPermanent} onChange={e => setModPermanent(e.target.checked)} /> Vĩnh viễn
                  </label>
                </div>
                <label className="flex items-center gap-1.5 text-xs text-zinc-400">
                  <input type="checkbox" checked={modChatMute} onChange={e => setModChatMute(e.target.checked)} /> Đồng thời hạn chế chat
                </label>
                <label className="flex items-center gap-1.5 text-xs text-zinc-400">
                  <input type="checkbox" checked={modBlockDevice} onChange={e => setModBlockDevice(e.target.checked)} /> Chặn thiết bị (IP / fingerprint)
                </label>
              </div>
              <div className="flex gap-2 flex-wrap justify-end">
                {detail.is_banned && <button disabled={actionLoading || !modReason.trim()} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/unban`, "PUT", { reason: modReason.trim(), confirmation_target: detail.id });
                  if (ok) setShowModeration(false);
                }} className="px-3 py-2 rounded-lg text-xs font-bold bg-emerald-700 hover:bg-emerald-600 text-zinc-100 cursor-pointer disabled:opacity-50">Mở khóa</button>}
                {detail.is_chat_muted && <button disabled={actionLoading || !modReason.trim()} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/chat-unmute`, "PUT", { reason: modReason.trim(), confirmation_target: detail.id });
                  if (ok) setShowModeration(false);
                }} className="px-3 py-2 rounded-lg text-xs font-bold bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer disabled:opacity-50">Bỏ hạn chế chat</button>}
                <button disabled={actionLoading || !modReason.trim()} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/chat-mute`, "PUT", { reason: modReason.trim(), confirmation_target: detail.id, duration_hours: modPermanent ? 0 : Number(modHours) || 0, permanent: modPermanent });
                  if (ok) setShowModeration(false);
                }} className="px-3 py-2 rounded-lg text-xs font-bold bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer disabled:opacity-50">Hạn chế chat</button>
                <button disabled={actionLoading || !modReason.trim()} onClick={async () => {
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/ban`, "PUT", { reason: modReason.trim(), confirmation_target: detail.id, duration_hours: modPermanent ? 0 : Number(modHours) || 0, permanent: modPermanent, chat_mute: modChatMute, chat_mute_hours: modPermanent ? 0 : Number(modHours) || 0, block_device: modBlockDevice });
                  if (ok) setShowModeration(false);
                }} className="px-3 py-2 rounded-lg text-xs font-bold bg-red-600 hover:bg-red-500 text-white cursor-pointer disabled:opacity-50">Khóa tài khoản</button>
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
                <div className="flex gap-2 mt-1">
                  <input value={addItemId} onChange={e => setAddItemId(e.target.value)} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "add-item", tab: "item", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                </div>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng</label>
                <input type="number" min={1} value={addItemQty} onChange={e => setAddItemQty(Math.max(1, parseInt(e.target.value) || 1))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Category</label>
                <select value={addItemCategory} onChange={e => setAddItemCategory(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"><option value="materials">Nguyên liệu</option><option value="consumables">Tiêu hao</option><option value="equipment">Trang bị</option><option value="quest">Nhiệm vụ</option><option value="misc">Khác</option></select>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowAddItem(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !addItemId} onClick={async () => {
				  const prepared = prepareAdminAction("grantPlayerInventory", "cấp vật phẩm cho người chơi", detail?.username || selectedId || "player");
                  if (!prepared) return;
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/items`, "POST", { item_id: addItemId, quantity: addItemQty, category: addItemCategory, ...prepared });
                  if (ok) setShowAddItem(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Instance Session Modal */}
        {showGuEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm"><div className="w-full max-w-2xl bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa Cổ trùng</h3><button onClick={() => setShowGuEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button></div>
            <div className="text-xs text-zinc-400">{showGuEditor.name} · <span className="font-mono">{showGuEditor.instance_id}</span><div className="mt-1">Gu ID: {showGuEditor.gu_id} · vị trí: {showGuEditor.location} · tạo: {showGuEditor.created_at ? new Date(showGuEditor.created_at).toLocaleString("vi-VN") : "—"}</div></div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              <label className="text-[11px] font-bold text-zinc-500">Bậc<input type="number" min={1} max={9} value={guEditForm.rank} onChange={e => setGuEditForm({...guEditForm, rank:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              <label className="text-[11px] font-bold text-zinc-500">Cấp<input type="number" min={1} value={guEditForm.level} onChange={e => setGuEditForm({...guEditForm, level:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              <label className="text-[11px] font-bold text-zinc-500">Đói (0–100)<input type="number" min={0} max={100} step={0.1} value={guEditForm.hunger} onChange={e => setGuEditForm({...guEditForm, hunger:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              <label className="text-[11px] font-bold text-zinc-500">Trạng thái<select value={guEditForm.status} onChange={e => setGuEditForm({...guEditForm, status:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1">{["ACTIVE","SLEEPING","SEALED","DEAD","EQUIPPING"].map(x=><option key={x}>{x}</option>)}</select></label>
              <label className="text-[11px] font-bold text-zinc-500">Thời gian dung hợp<input type="number" min={0} step={0.1} value={guEditForm.integration_time} onChange={e => setGuEditForm({...guEditForm, integration_time:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              <div className="space-y-2 text-xs text-zinc-300 pt-4"><label className="flex gap-2"><input type="checkbox" checked={guEditForm.equipped} onChange={e => setGuEditForm({...guEditForm, equipped:e.target.checked})} /> Trang bị</label><label className="flex gap-2"><input type="checkbox" checked={guEditForm.is_vital} onChange={e => setGuEditForm({...guEditForm, is_vital:e.target.checked, ...(e.target.checked ? {equipped:true,status:"ACTIVE"} : {})})} /> Bản Mệnh</label><label className="flex gap-2"><input type="checkbox" checked={guEditForm.is_locked} onChange={e => setGuEditForm({...guEditForm, is_locked:e.target.checked})} /> Khóa</label></div>
            </div>
            <div className="text-[10px] text-zinc-600">Gu rank: {showGuEditor.gu_rank} · tăng trưởng: {showGuEditor.growth_hours}h · cập nhật: {showGuEditor.updated_at ? new Date(showGuEditor.updated_at).toLocaleString("vi-VN") : "—"} · cho ăn: {showGuEditor.last_fed_at ? new Date(showGuEditor.last_fed_at).toLocaleString("vi-VN") : "—"}</div>
            <div className="flex gap-3 justify-end"><button onClick={() => setShowGuEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async () => { const rank=Number(guEditForm.rank), level=Number(guEditForm.level), hunger=Number(guEditForm.hunger), integration=Number(guEditForm.integration_time); if (![rank,level,hunger,integration].every(Number.isFinite) || rank<1 || rank>9 || !Number.isInteger(rank) || level<1 || !Number.isInteger(level) || hunger<0 || hunger>100 || integration<0) { addToast("Giá trị bậc, cấp, đói hoặc dung hợp không hợp lệ", "error"); return; } const prepared=prepareAdminAction("updatePlayerGu","cập nhật Cổ trùng",`${showGuEditor.name} · ${showGuEditor.instance_id}`); if(!prepared)return; const ok=await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/gus/${showGuEditor.instance_id}`,"PUT",{rank,level,hunger,state:guEditForm.status,equipped:guEditForm.equipped,is_vital:guEditForm.is_vital,is_locked:guEditForm.is_locked,integration_time:integration,...prepared}); if(ok){setShowGuEditor(null); if(selectedId)fetchPlayerGus(selectedId);} }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading ? "Đang xử lý..." : "Lưu"}</button></div>
          </div></div>
        )}

        {/* Instance Session Modal */}
        {showInstanceSessionEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm"><div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa phiên bí cảnh</h3><button onClick={() => setShowInstanceSessionEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button></div>
            <p className="text-xs text-zinc-400">Instance: <span className="font-bold text-zinc-200">{showInstanceSessionEditor.instance_id}</span> · ID <span className="font-mono">{showInstanceSessionEditor.id}</span></p>
            <div className="grid grid-cols-2 gap-3"><label className="text-[11px] font-bold text-zinc-500 font-mono">Current stage<input type="number" min={0} step={1} value={instanceSessionStage} onChange={e => setInstanceSessionStage(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label><label className="text-[11px] font-bold text-zinc-500 font-mono">Status<select value={instanceSessionStatus} onChange={e => setInstanceSessionStatus(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"><option>ACTIVE</option><option>COMPLETED</option><option>FAILED</option></select></label></div>
            <label className="text-[11px] font-bold text-zinc-500 font-mono">Temp inventory (JSON array)<textarea rows={6} value={instanceSessionInventory} onChange={e => setInstanceSessionInventory(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 font-mono mt-1" /></label>
            <div className="flex gap-3 justify-end"><button onClick={() => setShowInstanceSessionEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async () => { const stage = Number(instanceSessionStage); let inventory: unknown; try { inventory = JSON.parse(instanceSessionInventory); } catch { addToast("Temp inventory không phải JSON hợp lệ", "error"); return; } if (!Number.isSafeInteger(stage) || stage < 0 || !Array.isArray(inventory)) { addToast("Stage phải là số nguyên không âm và inventory phải là mảng", "error"); return; } const prepared = prepareAdminAction("updatePlayerInstanceSession", "cập nhật phiên bí cảnh", `${showInstanceSessionEditor.instance_id} · ${showInstanceSessionEditor.id}`); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/instance-sessions/${showInstanceSessionEditor.id}`, "PUT", { current_stage: stage, status: instanceSessionStatus, temp_inventory: inventory, ...prepared }); if (ok) { setShowInstanceSessionEditor(null); if (selectedId) fetchInstanceSessions(selectedId); } }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading ? "Đang xử lý..." : "Lưu"}</button></div>
          </div></div>
        )}

        {/* Quest Progress Modal */}
        {showQuestEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa tiến độ nhiệm vụ</h3><button onClick={() => setShowQuestEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button></div>
              <p className="text-xs text-zinc-400">Quest: <span className="text-zinc-200 font-bold">{showQuestEditor.quest_id}</span></p>
              <label className="text-[11px] font-bold text-zinc-500 font-mono">Trạng thái<select value={questStatus} onChange={e => setQuestStatus(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"><option value="ACTIVE">ACTIVE</option><option value="COMPLETED">COMPLETED</option><option value="FAILED">FAILED</option></select></label>
              <label className="text-[11px] font-bold text-zinc-500 font-mono">Progress data (JSON)<textarea value={questProgressData} onChange={e => setQuestProgressData(e.target.value)} rows={6} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 font-mono mt-1" /></label>
              <label className="text-[11px] font-bold text-zinc-500 font-mono">Completed at<input type="datetime-local" value={questCompletedAt} onChange={e => setQuestCompletedAt(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              <div className="flex gap-3 justify-end"><button onClick={() => setShowQuestEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async () => { let parsed: unknown; try { parsed = JSON.parse(questProgressData || "{}"); } catch { addToast("progress_data không phải JSON hợp lệ", "error"); return; } const prepared = prepareAdminAction("updatePlayerQuest", "cập nhật tiến độ nhiệm vụ", detail?.username || selectedId || "player"); if (!prepared) return; const completed = questCompletedAt ? new Date(questCompletedAt).toISOString() : null; const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/quests/${encodeURIComponent(showQuestEditor.quest_id)}`, "PUT", { status: questStatus, progress_data: parsed, completed_at: completed, ...prepared }); if (ok) { setShowQuestEditor(null); if (selectedId) fetchPlayerQuests(selectedId); } }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading ? "Đang xử lý..." : "Lưu"}</button></div>
            </div>
          </div>
        )}

        {showPlayerSectMissionEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm"><div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa nhiệm vụ tông môn</h3><button onClick={()=>setShowPlayerSectMissionEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500"/></button></div>
            <div className="text-xs text-zinc-400">{showPlayerSectMissionEditor.title} · {showPlayerSectMissionEditor.sect_name}<span className="block font-mono text-[10px] mt-1">Row {showPlayerSectMissionEditor.id} · Mission {showPlayerSectMissionEditor.mission_id}</span></div>
            <div className="grid grid-cols-2 gap-3"><label className="text-[11px] font-bold text-zinc-500">Tiến độ<input type="number" min={0} step={1} value={playerSectMissionProgress} onChange={e=>setPlayerSectMissionProgress(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label><label className="text-[11px] font-bold text-zinc-500">Trạng thái<select value={playerSectMissionStatus} onChange={e=>setPlayerSectMissionStatus(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"><option>IN_PROGRESS</option><option>EXPIRED</option></select></label></div>
            <label className="text-[11px] font-bold text-zinc-500">Hạn nhiệm vụ (để trống nếu không hạn)<input type="datetime-local" value={playerSectMissionExpiry} onChange={e=>setPlayerSectMissionExpiry(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
            <div className="text-[10px] text-zinc-600">Nhận lúc: {new Date(showPlayerSectMissionEditor.assigned_at).toLocaleString("vi-VN")} · Mục tiêu: {showPlayerSectMissionEditor.target_amount}</div>
            <div className="flex gap-3 justify-end"><button onClick={()=>setShowPlayerSectMissionEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async()=>{const progress=Number(playerSectMissionProgress);if(!Number.isSafeInteger(progress)||progress<0){addToast("Tiến độ phải là số nguyên không âm","error");return;}const prepared=prepareAdminAction("updatePlayerSectMission","cập nhật nhiệm vụ tông môn",`${showPlayerSectMissionEditor.title} · ${showPlayerSectMissionEditor.id}`);if(!prepared)return;const expiry=playerSectMissionExpiry?new Date(playerSectMissionExpiry).toISOString():"";const ok=await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/sect-missions/${showPlayerSectMissionEditor.id}`,"PUT",{current_progress:progress,status:playerSectMissionStatus,expires_at:expiry,...prepared});if(ok){setShowPlayerSectMissionEditor(null);if(selectedId)fetchPlayerSectMissions(selectedId);}}} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading?"Đang xử lý...":"Lưu"}</button></div>
          </div></div>
        )}

        {showTrackingMarkEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm"><div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa ấn truy tung</h3><button onClick={()=>setShowTrackingMarkEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500"/></button></div>
            <div className="text-xs text-zinc-400">{showTrackingMarkEditor.tracker_username} → {showTrackingMarkEditor.target_username}<span className="block text-[10px] font-mono text-zinc-600">{showTrackingMarkEditor.id}</span></div>
            <div className="grid grid-cols-2 gap-3"><label className="text-[11px] font-bold text-zinc-500">Loại ấn<select value={trackingMarkType} onChange={e=>setTrackingMarkType(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"><option value="wisdom">wisdom</option><option value="blood">blood</option></select></label><label className="text-[11px] font-bold text-zinc-500">Uy lực<input type="number" min={0} step={0.1} value={trackingMarkPower} onChange={e=>setTrackingMarkPower(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label></div>
            <label className="text-[11px] font-bold text-zinc-500">Hết hạn<input type="datetime-local" value={trackingMarkExpiry} onChange={e=>setTrackingMarkExpiry(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
            <div className="text-[10px] text-zinc-600">Tạo lúc: {new Date(showTrackingMarkEditor.created_at).toLocaleString("vi-VN")}</div>
            <div className="flex gap-3 justify-end"><button onClick={()=>setShowTrackingMarkEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async()=>{const power=Number(trackingMarkPower);if(!Number.isFinite(power)||power<0||power>1e9||!trackingMarkExpiry){addToast("Uy lực hoặc hạn dùng không hợp lệ","error");return;}const prepared=prepareAdminAction("updatePlayerTrackingMark","cập nhật ấn truy tung",showTrackingMarkEditor.id);if(!prepared)return;const ok=await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/tracking-marks/${showTrackingMarkEditor.id}`,"PUT",{mark_type:trackingMarkType,power,expires_at:new Date(trackingMarkExpiry).toISOString(),...prepared});if(ok){setShowTrackingMarkEditor(null);if(selectedId)fetchPlayerTrackingMarks(selectedId);}}} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading?"Đang xử lý...":"Lưu"}</button></div>
          </div></div>
        )}

        {showAchievementEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80"><div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <h3 className="text-sm font-bold text-zinc-100">Sửa thành tựu {showAchievementEditor.achievement_id}</h3>
            <label className="text-[11px] font-bold text-zinc-500">Danh hiệu<input maxLength={100} value={achievementTitle} onChange={e=>setAchievementTitle(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
            <label className="text-[11px] font-bold text-zinc-500">Khung<input maxLength={50} value={achievementFrame} onChange={e=>setAchievementFrame(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
            <label className="text-[11px] font-bold text-zinc-500">Mở khóa lúc<input type="datetime-local" value={achievementUnlockedAt} onChange={e=>setAchievementUnlockedAt(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
            <div className="flex gap-3 justify-end"><button onClick={()=>setShowAchievementEditor(null)} className="px-4 py-2 rounded-lg text-xs bg-zinc-900 text-zinc-400 border border-zinc-800">Hủy</button><button disabled={actionLoading} onClick={async()=>{if(!achievementUnlockedAt)return;const prepared=prepareAdminAction("updatePlayerAchievement","cập nhật thành tựu",showAchievementEditor.achievement_id);if(!prepared)return;const ok=await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/achievements/${encodeURIComponent(showAchievementEditor.achievement_id)}`,"PUT",{selected_title:achievementTitle,selected_frame:achievementFrame,unlocked_at:new Date(achievementUnlockedAt).toISOString(),...prepared});if(ok){setShowAchievementEditor(null);if(selectedId)fetchPlayerAchievements(selectedId);}}} className="px-4 py-2 rounded-lg text-xs bg-amber-600 text-zinc-100">Lưu</button></div>
          </div></div>
        )}
        {/* Recipe Progress Modal */}
        {showRecipeEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa tiến độ công thức</h3>
                <button onClick={() => setShowRecipeEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <p className="text-xs text-zinc-400">Công thức: <span className="text-zinc-200 font-bold">{showRecipeEditor.name || showRecipeEditor.item_id}</span></p>
              <div className="grid grid-cols-2 gap-3">
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Hoàn thành (%)<input type="number" min={0} max={100} step="0.01" value={recipeCompletion} onChange={e => setRecipeCompletion(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lần suy diễn<input type="number" min={0} value={recipeDeduceCount} onChange={e => setRecipeDeduceCount(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label>
              </div>
              <div className="flex gap-5 text-xs text-zinc-300">
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={recipeUnlocked} onChange={e => setRecipeUnlocked(e.target.checked)} /> Đã mở khóa</label>
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={recipeFavorite} onChange={e => setRecipeFavorite(e.target.checked)} /> Yêu thích</label>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowRecipeEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const prepared = prepareAdminAction("updatePlayerRecipe", "cập nhật tiến độ công thức", detail?.username || selectedId || "player");
                  if (!prepared) return;
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/recipes/${encodeURIComponent(showRecipeEditor.item_id)}`, "PUT", { completion_rate: Math.max(0, Math.min(100, Number(recipeCompletion) || 0)), is_unlocked: recipeUnlocked, deduce_count: Math.max(0, parseInt(recipeDeduceCount, 10) || 0), favorite: recipeFavorite, ...prepared });
                  if (ok) { setShowRecipeEditor(null); if (selectedId) fetchPlayerItems(selectedId); }
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">{actionLoading ? "Đang xử lý..." : "Lưu"}</button>
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
              <p className="text-xs text-zinc-400">Vật phẩm: <span className="text-zinc-200 font-bold">{itemDictionary[showRemoveItem.split("::")[0]]?.name || playerItems.find(i => `${i.item_id}::${i.category}` === showRemoveItem)?.name || showRemoveItem.split("::")[0]}</span></p>
              <p className="text-xs text-zinc-500">Hiện có: <span className="text-zinc-300 font-bold">{(() => { const row = playerItems.find(i => `${i.item_id}::${i.category}` === showRemoveItem); return (row?.quantity || 0).toLocaleString(); })()}</span></p>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng xóa</label>
                <input type="number" min={1} max={playerItems.find(i => `${i.item_id}::${i.category}` === showRemoveItem)?.quantity || 1} value={removeItemQty} onChange={e => setRemoveItemQty(Math.max(1, Math.min(parseInt(e.target.value) || 1, playerItems.find(i => `${i.item_id}::${i.category}` === showRemoveItem)?.quantity || 1)))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowRemoveItem(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const [itemId, category] = showRemoveItem.split("::");
                  const endpoint = `${BACKEND_URL}/api/admin/players/${selectedId}/items`;
                  const prepared = prepareAdminAction("deletePlayerInventoryItem", "thu hồi vật phẩm", endpoint);
                  if (!prepared) return;
                  const ok = await exec(endpoint, "DELETE", { item_id: itemId, category, quantity: removeItemQty, ...prepared });
                  if (ok) setShowRemoveItem(null);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-red-600 hover:bg-red-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Xóa"}
                </button>
              </div>
            </div>
          </div>
        )}

        {showInventoryItemEditor && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm"><div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa vật phẩm túi đồ</h3><button onClick={() => setShowInventoryItemEditor(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button></div>
            <div className="text-xs text-zinc-400">{showInventoryItemEditor.name || showInventoryItemEditor.item_id} · <span className="font-mono">{showInventoryItemEditor.item_id}</span></div>
            <div className="grid grid-cols-2 gap-3"><label className="text-[11px] font-bold text-zinc-500">Số lượng<input type="number" min={0} step={1} value={inventoryEditQuantity} onChange={e => setInventoryEditQuantity(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1" /></label><label className="text-[11px] font-bold text-zinc-500">Nhóm<select value={inventoryEditCategory} onChange={e => setInventoryEditCategory(e.target.value)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1">{["materials","consumables","equipment","quest","misc"].map(x=><option key={x}>{x}</option>)}</select></label></div>
            <div className="text-[10px] text-zinc-600">Tạo: {showInventoryItemEditor.created_at ? new Date(showInventoryItemEditor.created_at).toLocaleString("vi-VN") : "—"} · Cập nhật: {showInventoryItemEditor.updated_at ? new Date(showInventoryItemEditor.updated_at).toLocaleString("vi-VN") : "—"}</div>
            <div className="flex gap-3 justify-end"><button onClick={() => setShowInventoryItemEditor(null)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 cursor-pointer">Hủy</button><button disabled={actionLoading} onClick={async () => { const quantity=Number(inventoryEditQuantity); if(!Number.isSafeInteger(quantity)||quantity<0){addToast("Số lượng phải là số nguyên không âm", "error");return;} const prepared=prepareAdminAction("updatePlayerInventoryItem","cập nhật vật phẩm",`${showInventoryItemEditor.item_id} · ${showInventoryItemEditor.category}`);if(!prepared)return;const ok=await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/items/${encodeURIComponent(showInventoryItemEditor.item_id)}/${encodeURIComponent(showInventoryItemEditor.category)}`,"PUT",{quantity,category:inventoryEditCategory,...prepared});if(ok){setShowInventoryItemEditor(null);if(selectedId)fetchPlayerItems(selectedId);}}} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 cursor-pointer disabled:opacity-50">{actionLoading ? "Đang xử lý..." : "Lưu"}</button></div>
          </div></div>
        )}

        {/* Add Gu Modal */}
        {showAddGu && (() => {
          const guRanks = addGuId ? shopSearchGus
            .filter((x: ApiResponseData) => x.id === addGuId)
            .map((x: ApiResponseData) => x.rank)
            .filter((r: number) => r > 0)
            .sort((a: number, b: number) => a - b) : [];
          const selectedRank = addGuRank ? Number(addGuRank) : 0;
          const rankRequired = guRanks.length > 1 && !selectedRank;
          const submitRank = guRanks.length === 1 ? guRanks[0] : selectedRank;
          return (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm cổ trùng</h3>
                <button onClick={() => setShowAddGu(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn cổ trùng</label>
                <div className="flex gap-2 mt-1">
                  <input value={addGuId} onChange={e => { setAddGuId(e.target.value); setAddGuRank(""); }} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "add-gu", tab: "gu", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                </div>
              </div>
              {guRanks.length > 0 && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Bậc {guRanks.length > 1 && <span className="text-amber-400">(bắt buộc chọn)</span>}</label>
                  {guRanks.length === 1 ? (
                    <div className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-400 mt-1">Bậc {guRanks[0]}</div>
                  ) : (
                    <select value={addGuRank} onChange={e => setAddGuRank(e.target.value)} className={`w-full bg-zinc-900 border rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50 ${rankRequired ? "border-red-600/60" : "border-zinc-800"}`}>
                      <option value="">Chọn bậc...</option>
                      {guRanks.map(r => <option key={r} value={r}>Bậc {r}</option>)}
                    </select>
                  )}
                </div>
              )}
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng</label>
                <input type="number" min={1} value={addGuQty} onChange={e => setAddGuQty(Math.max(1, parseInt(e.target.value) || 1))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowAddGu(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading || !addGuId || rankRequired} onClick={async () => {
				  const prepared = prepareAdminAction("grantPlayerInventory", "cấp Cổ trùng cho người chơi", detail?.username || selectedId || "player");
                  if (!prepared) return;
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/gus`, "POST", { gu_id: addGuId, quantity: addGuQty, ...(submitRank ? { rank: submitRank } : {}), ...prepared });
                  if (ok) setShowAddGu(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Thêm"}
                </button>
              </div>
            </div>
          </div>
          );
        })()}
        {/* Edit Player Modal */}
        {showEditPlayer && detail && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-2xl bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa thông tin người chơi</h3>
                <button onClick={() => setShowEditPlayer(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tên đăng nhập</label>
                  <input type="text" value={editPlayerForm.username || ""} onChange={e => setEditPlayerForm(f => ({ ...f, username: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Path mastery (JSON)</label>
                  <textarea value={editPlayerForm.path_masteries || "{}"} onChange={e => setEditPlayerForm(f => ({ ...f, path_masteries: e.target.value }))} rows={3} placeholder={'{"fire": 100, "sword": 50}'} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 font-mono mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tu vi (1-9)</label>
                  <input type="number" min={1} max={10} value={editPlayerForm.rank || ""} onChange={e => setEditPlayerForm(f => ({ ...f, rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiểu cảnh giới (1-4)</label>
                  <input type="number" min={1} max={4} value={editPlayerForm.sub_rank || ""} onChange={e => setEditPlayerForm(f => ({ ...f, sub_rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Thiên phú</label>
                  <input type="text" value={editPlayerForm.aptitude_type || ""} onChange={e => setEditPlayerForm(f => ({ ...f, aptitude_type: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Phần trăm thiên phú (0-100)</label>
                  <input type="number" min={0} max={100} value={editPlayerForm.aptitude_percent || ""} onChange={e => setEditPlayerForm(f => ({ ...f, aptitude_percent: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hạng thiên phú</label>
                  <input type="text" value={editPlayerForm.aptitude_grade || ""} onChange={e => setEditPlayerForm(f => ({ ...f, aptitude_grade: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Chân nguyên hiện tại</label>
                  <input type="number" value={editPlayerForm.current_mana || ""} onChange={e => setEditPlayerForm(f => ({ ...f, current_mana: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Chân nguyên tối đa</label>
                  <input type="number" value={editPlayerForm.max_mana || ""} onChange={e => setEditPlayerForm(f => ({ ...f, max_mana: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Máu hiện tại</label>
                  <input type="number" value={editPlayerForm.hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Máu tối đa</label>
                  <input type="number" value={editPlayerForm.base_hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, base_hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tuổi thọ</label>
                  <input type="number" value={editPlayerForm.lifespan || ""} onChange={e => setEditPlayerForm(f => ({ ...f, lifespan: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Khu vực hiện tại</label>
                  <input type="text" value={editPlayerForm.current_region || ""} onChange={e => setEditPlayerForm(f => ({ ...f, current_region: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiến trình phá bức tường</label>
                  <input type="number" value={editPlayerForm.wall_progress || ""} onChange={e => setEditPlayerForm(f => ({ ...f, wall_progress: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Nhân khí</label>
                  <input type="number" value={editPlayerForm.human_qi || ""} onChange={e => setEditPlayerForm(f => ({ ...f, human_qi: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hồn lực</label>
                  <input type="number" value={editPlayerForm.soul_hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, soul_hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hồn lực tối đa</label>
                  <input type="number" value={editPlayerForm.max_soul || ""} onChange={e => setEditPlayerForm(f => ({ ...f, max_soul: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tầng hồn</label>
                  <input type="text" value={editPlayerForm.soul_tier || ""} onChange={e => setEditPlayerForm(f => ({ ...f, soul_tier: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lần khai phá</label>
                  <input type="number" min={0} value={editPlayerForm.explore_count || ""} onChange={e => setEditPlayerForm(f => ({ ...f, explore_count: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Áp lực đan điền</label>
                  <input type="number" value={editPlayerForm.aperture_stress || ""} onChange={e => setEditPlayerForm(f => ({ ...f, aperture_stress: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Avatar URL</label>
                  <input type="text" value={editPlayerForm.avatar_url || ""} onChange={e => setEditPlayerForm(f => ({ ...f, avatar_url: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Chủng tộc</label>
                  <input type="text" value={editPlayerForm.race || ""} onChange={e => setEditPlayerForm(f => ({ ...f, race: e.target.value }))} list="race-list" placeholder="human, fur_people, merman, feathered..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  <datalist id="race-list">
                    {Object.values(raceDictionary || {}).map((r) => (
                      <option key={r.key} value={r.key}>{r.name}</option>
                    ))}
                  </datalist>
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Giới tính</label>
                  <input type="text" value={editPlayerForm.gender || ""} onChange={e => setEditPlayerForm(f => ({ ...f, gender: e.target.value }))} placeholder="male" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Thể chất</label>
                  <input type="text" value={editPlayerForm.constitution || ""} onChange={e => setEditPlayerForm(f => ({ ...f, constitution: e.target.value }))} list="constitution-list" placeholder="solar_sun_moon_body, great_strength_true_martial_body..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  <datalist id="constitution-list">
                    {Object.values(constitutionDictionary || {}).map((c: ApiResponseData) => (
                      <option key={c.key} value={c.key}>{c.name}</option>
                    ))}
                  </datalist>
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Trạng thái (cách nhau bằng dấu phẩy)</label>
                  <input type="text" value={editPlayerForm.statuses || ""} onChange={e => setEditPlayerForm(f => ({ ...f, statuses: e.target.value }))} placeholder="trống" className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tọa độ X</label>
                  <input type="number" value={editPlayerForm.x || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, x: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tọa độ Y</label>
                  <input type="number" value={editPlayerForm.y || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, y: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Niệm Lực hiện tại</label>
                  <input type="number" value={editPlayerForm.current_thought || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, current_thought: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Niệm Lực tối đa</label>
                  <input type="number" value={editPlayerForm.max_thought || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, max_thought: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tỉ lệ hồi Niệm Lực</label>
                  <input type="number" value={editPlayerForm.thought_regen_rate || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, thought_regen_rate: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tốc độ thời gian (Multiplier)</label>
                  <input type="number" step="0.1" value={editPlayerForm.time_speed_multiplier || "1"} onChange={e => setEditPlayerForm(f => ({ ...f, time_speed_multiplier: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiên Ngọc (Immortal Jade)</label>
                  <input type="number" value={editPlayerForm.immortal_jade || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, immortal_jade: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Danh hiệu đã chọn</label>
                  <input type="text" value={editPlayerForm.selected_title || ""} onChange={e => setEditPlayerForm(f => ({ ...f, selected_title: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Khung avatar đã chọn</label>
                  <input type="text" value={editPlayerForm.selected_frame || ""} onChange={e => setEditPlayerForm(f => ({ ...f, selected_frame: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Cải trang: Tu vi giả</label>
                  <input type="number" value={editPlayerForm.fake_rank || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, fake_rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Cải trang: Cảnh giới giả</label>
                  <input type="number" value={editPlayerForm.fake_sub_rank || "0"} onChange={e => setEditPlayerForm(f => ({ ...f, fake_sub_rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Cải trang: Bản mệnh Gu giả</label>
                  <input type="text" value={editPlayerForm.fake_vital_gu || ""} onChange={e => setEditPlayerForm(f => ({ ...f, fake_vital_gu: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex items-center gap-4 pt-2">
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono">
                    <input type="checkbox" checked={detail.is_admin} disabled className="accent-amber-500 w-4 h-4 disabled:opacity-60" />
                    Quyền Admin (chỉ đọc)
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.is_disguised === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, is_disguised: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Đang Cải Trang
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.needs_onboarding === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, needs_onboarding: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Cần Onboarding
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.needs_soul_refine === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, needs_soul_refine: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Tôi Luyện Linh Hồn
                  </label>
                </div>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiểu sử</label>
                <textarea value={editPlayerForm.bio || ""} onChange={e => setEditPlayerForm(f => ({ ...f, bio: e.target.value }))} rows={2} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowEditPlayer(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const body: ApiResponseData = {};
                  const num = (v: string) => (v === undefined || v === null || v.trim() === "" ? undefined : Number(v));
                  const set = (field: string, cur: ApiResponseData, next: ApiResponseData) => {
                    if (next === undefined || next === "") return;
                    if (String(cur) !== String(next)) body[field] = next;
                  };
                  set("username", detail.username, editPlayerForm.username);
                  set("rank", detail.rank, num(editPlayerForm.rank));
                  set("sub_rank", detail.sub_rank, num(editPlayerForm.sub_rank));
                  set("aptitude_type", detail.aptitude_type || "", editPlayerForm.aptitude_type);
                  set("aptitude_percent", detail.aptitude_percent, num(editPlayerForm.aptitude_percent));
                  set("aptitude_grade", detail.aptitude_grade || "", editPlayerForm.aptitude_grade);
                  set("current_mana", detail.current_mana, num(editPlayerForm.current_mana));
                  set("max_mana", detail.max_mana, num(editPlayerForm.max_mana));
                  set("hp", detail.hp, num(editPlayerForm.hp));
                  set("base_hp", detail.base_hp, num(editPlayerForm.base_hp));
                  set("wall_progress", detail.wall_progress, num(editPlayerForm.wall_progress));
                  set("human_qi", detail.human_qi, num(editPlayerForm.human_qi));
                  set("lifespan", detail.lifespan, num(editPlayerForm.lifespan));
                  set("current_region", detail.current_region || "", editPlayerForm.current_region);
                  set("x", detail.x, num(editPlayerForm.x));
                  set("y", detail.y, num(editPlayerForm.y));
                  set("soul_hp", detail.soul_hp, num(editPlayerForm.soul_hp));
                  set("max_soul", detail.max_soul, num(editPlayerForm.max_soul));
                  set("soul_tier", detail.soul_tier || "", editPlayerForm.soul_tier);
                  set("explore_count", detail.explore_count, num(editPlayerForm.explore_count));
                  set("aperture_stress", detail.aperture_stress, num(editPlayerForm.aperture_stress));
                  set("bio", detail.bio || "", editPlayerForm.bio);
                  set("avatar_url", detail.avatar_url || "", editPlayerForm.avatar_url);
                  set("race", detail.race || "", editPlayerForm.race);
                  set("gender", detail.gender || "", editPlayerForm.gender);
                  set("constitution", detail.constitution || "", editPlayerForm.constitution);
                  set("current_thought", detail.current_thought, num(editPlayerForm.current_thought));
                  set("max_thought", detail.max_thought, num(editPlayerForm.max_thought));
                  set("thought_regen_rate", detail.thought_regen_rate, num(editPlayerForm.thought_regen_rate));
                  set("time_speed_multiplier", detail.time_speed_multiplier, num(editPlayerForm.time_speed_multiplier));
                  set("selected_title", detail.selected_title || "", editPlayerForm.selected_title);
                  set("selected_frame", detail.selected_frame || "", editPlayerForm.selected_frame);
                  set("fake_rank", detail.fake_rank, num(editPlayerForm.fake_rank));
                  set("fake_sub_rank", detail.fake_sub_rank, num(editPlayerForm.fake_sub_rank));
                  set("fake_vital_gu", detail.fake_vital_gu || "", editPlayerForm.fake_vital_gu);
                  set("is_disguised", detail.is_disguised, editPlayerForm.is_disguised === "true");
                  set("needs_onboarding", detail.needs_onboarding, editPlayerForm.needs_onboarding === "true");
                  set("needs_soul_refine", detail.needs_soul_refine, editPlayerForm.needs_soul_refine === "true");
                  set("immortal_jade", detail.immortal_jade, num(editPlayerForm.immortal_jade));

                  let pathMasteries: Record<string, number> | undefined;
                  try {
                    const parsed = JSON.parse(editPlayerForm.path_masteries || "{}");
                    if (!parsed || Array.isArray(parsed) || typeof parsed !== "object" || Object.entries(parsed).some(([key, value]) => !key.trim() || typeof value !== "number" || !Number.isInteger(value) || value < 0)) throw new Error("invalid");
                    pathMasteries = parsed as Record<string, number>;
                  } catch {
                    if (addToast) addToast("Path mastery phải là JSON dạng { path: số nguyên không âm }", "error");
                    return;
                  }
                  if (JSON.stringify(pathMasteries) !== JSON.stringify(detail.path_masteries || {})) body["path_masteries"] = pathMasteries;

                  const statusesRaw = (editPlayerForm.statuses || "").split(",").map(s => s.trim()).filter(Boolean);
                  const detailStatusIds = (detail.statuses || []).map((s: string | { id: string }) => typeof s === "string" ? s : s.id);
                  const statusesChanged = JSON.stringify(statusesRaw) !== JSON.stringify(detailStatusIds);
                  if (statusesChanged) body["statuses"] = statusesRaw;
                  if (Object.keys(body).length === 0) {
                    if (addToast) addToast("Không có thay đổi nào", "info");
                    return;
                  }
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}`, "PUT", body);
                  if (ok) setShowEditPlayer(false);
                }} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {actionLoading ? "Đang xử lý..." : "Lưu"}
                </button>
              </div>
            </div>
          </div>
        )}
        {showRolesEditor && detail && selectedId && (
          <PlayerRolesEditor playerId={selectedId} username={detail.username} canAssign={canAdminAction("assignPlayerRoles")} addToast={addToast} onClose={() => setShowRolesEditor(false)} onSaved={() => { fetchDetail(selectedId); fetchPlayers(); }} />
        )}
        {renderItemPicker()}
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
            <button onClick={() => { if (selectedSect) { setSectEditName(selectedSect.name); setSectEditDesc(selectedSect.description || ""); setSectEditRegion(selectedSect.region || ""); setSectEditLevel(String(selectedSect.level)); setSectEditFunds(String(selectedSect.funds)); setSectEditRequireApproval(Boolean(selectedSect.require_approval)); setSectEditRequirePromotionApproval(Boolean(selectedSect.require_promotion_approval)); setShowSectEdit(true); } }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
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
            <div className="border-b border-zinc-800/80 flex gap-0 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
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
                      {selectedSectMembers?.map((m: ApiResponseData) => (
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
                          {selectedSect.vault_gu.map((item: ApiResponseData, i: number) => {
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
                          {Object.entries(selectedSect.vault_materials).map(([id, qty]: [string, ApiResponseData]) => {
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
                      {selectedSect.vault_recipes.map((vr: ApiResponseData, i: number) => (
                        <span key={i} className="px-2 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 font-mono">
                          {vr.recipe_id} ({Number(vr.completion_rate || 0).toFixed(0)}%)
                        </span>
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
                  <button onClick={() => { setNewMission({ title: "", description: "", mission_type: "GATHER", target_id: "", target_amount: 1, reward_contribution: 10, required_member_rank: "NGOAI_MON", duration_hours: 24 }); setShowCreateMission(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 rounded-lg transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Tạo nhiệm vụ
                  </button>
                </div>
                {sectMissionsLoading ? (
                  <div className="flex justify-center py-8"><RefreshCw className="w-5 h-5 text-amber-400 animate-spin" /></div>
                ) : sectMissions?.length === 0 ? (
                  <p className="text-xs text-zinc-600 italic py-8 text-center">Chưa có nhiệm vụ nào</p>
                ) : (
                  <div className="space-y-2">
                    {sectMissions?.map((m: ApiResponseData) => (
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
                            <span className={m.expires_at && new Date(m.expires_at).getTime() < Date.now() ? "text-red-400" : "text-amber-400/80"}>
                              {m.expires_at ? `Hết hạn: ${new Date(m.expires_at).toLocaleString("vi-VN")}` : "Không hết hạn"}
                            </span>
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
            <SectDefendersAdmin sectId={selectedSectId} canRemove={canAdminAction("removeSectDefender")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("removeSectDefender", label, target)} />
            <SectRequestsAdmin sectId={selectedSectId} canCancel={canAdminAction("cancelSectRequest")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("cancelSectRequest", label, target)} />
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
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Khu vực</label>
                  <input type="text" value={sectEditRegion} onChange={e => setSectEditRegion(e.target.value)} placeholder="Southern_Border..." className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
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
                <div className="flex flex-col gap-2 pt-1">
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-400 font-mono cursor-pointer">
                    <input type="checkbox" checked={sectEditRequireApproval} onChange={e => setSectEditRequireApproval(e.target.checked)} className="accent-amber-500 w-4 h-4" />
                    Cần phê duyệt khi gia nhập Tông môn
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-400 font-mono cursor-pointer">
                    <input type="checkbox" checked={sectEditRequirePromotionApproval} onChange={e => setSectEditRequirePromotionApproval(e.target.checked)} className="accent-amber-500 w-4 h-4" />
                    Cần phê duyệt khi thăng cấp chức vụ
                  </label>
                </div>
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowSectEdit(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const body: ApiResponseData = {};
                  if (sectEditName !== selectedSect.name) body.name = sectEditName;
                  if (sectEditDesc !== (selectedSect.description || "")) body.description = sectEditDesc;
                  if (sectEditRegion !== (selectedSect.region || "")) body.region = sectEditRegion;
                  if (Number(sectEditLevel) !== selectedSect.level) body.level = Number(sectEditLevel);
                  if (Number(sectEditFunds) !== selectedSect.funds) body.funds = Number(sectEditFunds);
                  if (sectEditRequireApproval !== Boolean(selectedSect.require_approval)) body.require_approval = sectEditRequireApproval;
                  if (sectEditRequirePromotionApproval !== Boolean(selectedSect.require_promotion_approval)) body.require_promotion_approval = sectEditRequirePromotionApproval;
                  if (Object.keys(body).length === 0) { setShowSectEdit(false); return; }
                  const ok = await exec(`${BACKEND_URL}/api/admin/sects/${selectedSectId}`, "PUT", body);
                  if (ok) { fetchSectDetail(selectedSectId); fetchSects(); setShowSectEdit(false); }
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
                  <option value="HO_PHAP">Hộ Pháp</option>
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
                <select value={stockVaultType} onChange={e => setStockVaultType(e.target.value as ApiResponseData)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
                  <option value="material">Vật liệu</option>
                  <option value="gu">Cổ trùng</option>
                  <option value="recipe">Công thức</option>
                </select>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Vật phẩm</label>
                <div className="flex gap-2 mt-1">
                  <input value={stockVaultId} onChange={e => setStockVaultId(e.target.value)} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "vault", tab: stockVaultType === "recipe" ? "recipe" : stockVaultType === "gu" ? "gu" : "item", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                </div>
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
                  const targetSpec = itemDictionary[stockVaultId] || shopSearchItems.find(it => it.id === stockVaultId);
                  const itType = (targetSpec?.item_type || targetSpec?.type || "").toLowerCase();
                  if (stockVaultType === "material" && (itType === "consumable" || itType === "currency")) {
                    if (addToast) addToast("Không thể nhập vật phẩm loại Tiêu hao hoặc Tiền tệ vào bảo khố", "error");
                    return;
                  }
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
                      <option value="HO_PHAP">Hộ Pháp</option>
                    </select>
                  </div>
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Mục tiêu</label>
                  <div className="flex gap-2 mt-1">
                    <input value={newMission.target_id} onChange={e => setNewMission(p => ({ ...p, target_id: e.target.value }))} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                    <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: newMission.mission_type === "DONATE" ? "mission-donate" : "mission", tab: newMission.mission_type === "DONATE" ? "gu" : "item", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
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
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Thời hạn (giờ)</label>
                  <input type="number" min={-1} value={newMission.duration_hours} onChange={e => setNewMission(p => ({ ...p, duration_hours: parseInt(e.target.value) || 0 }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                  <span className="text-[10px] text-zinc-600 font-mono">0 = mặc định 24h · -1 = không hết hạn</span>
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
        {renderItemPicker()}
      </div>
    );
  }

  // ===== Main Admin Page =====
  return (
    <div className="w-full max-w-6xl mx-auto px-4 py-6">
      {/* Header with section tabs */}
      {!embedded && <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between mb-6 gap-3">
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
      </div>}

      {/* Players section */}
      {section === "players" && (
        <>
          <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
            <button onClick={() => setPlayersTab("list")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${playersTab === "list" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Users className="w-3.5 h-3.5 inline mr-1.5" />Danh sách
            </button>
            <button onClick={() => setPlayersTab("online")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${playersTab === "online" ? "text-emerald-400 border-emerald-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Activity className="w-3.5 h-3.5 inline mr-1.5" />Trực tuyến
            </button>
          </div>

          {playersTab === "list" && (
          <>
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4 mb-4">
            <div className="flex gap-3 items-end">
              <div className="space-y-1">
                <label className="text-[11px] font-bold text-zinc-500 uppercase tracking-wider font-mono flex items-center gap-1">
                  <Filter className="w-3 h-3" /> Loại tìm kiếm
                </label>
                <select value={searchBy} onChange={e => { updatePlayerQuery({ search_by: e.target.value, page: "1" }); }} className="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors cursor-pointer">
                  <option value="all">Tất cả (tự nhận diện ID)</option>
                  <option value="id">ID</option>
                  <option value="username">Tên đăng nhập</option>
                  <option value="email">Email</option>
                  <option value="item">Vật phẩm sở hữu</option>
                  <option value="gu">Cổ trùng sở hữu</option>
                </select>
              </div>
              <div className="flex-1 space-y-1">
                <label className="text-[11px] font-bold text-zinc-500 uppercase tracking-wider font-mono flex items-center gap-1">
                  <Search className="w-3 h-3" /> Tìm kiếm
                </label>
                <input type="text" value={searchText} onChange={e => setSearchText(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { updatePlayerQuery({ page: "1", search: searchText }); } }} placeholder={searchBy === "item" ? "ID hoặc tên vật phẩm..." : searchBy === "gu" ? "ID hoặc tên cổ trùng..." : "ID, tên đăng nhập hoặc email..."} className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors placeholder:text-zinc-600" />
              </div>
              <button onClick={() => { updatePlayerQuery({ page: "1", search: searchText }); }} className="px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">Tìm</button>
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
                <table className="w-full text-left text-xs min-w-max">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">ID</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Tên</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Tu vi</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Sinh lực</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Chân nguyên</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Nguyên Thạch</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Tiên Nguyên</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Admin</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Khu vực</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Ngày tạo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {players.map((p) => (
                      <tr key={p.id} onClick={() => setSelectedId(p.id)} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors cursor-pointer">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate whitespace-nowrap" title={p.id}>{p.id.slice(0, 8)}...</td>
                        <td className="px-3 py-2.5 font-semibold text-zinc-200 whitespace-nowrap">{p.username}{p.is_banned && <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] font-bold bg-red-950/80 border border-red-500/40 text-red-400">ĐÃ KHÓA</span>}</td>
                        <td className={`px-3 py-2.5 font-bold whitespace-nowrap ${rankColor(p.rank)}`}>{getRankName(p.rank, p.sub_rank)}</td>
                        <td className="px-3 py-2.5 font-mono text-zinc-300 whitespace-nowrap">{Math.round(p.hp)}/{Math.round(p.base_hp)}</td>
                        <td className="px-3 py-2.5 font-mono text-zinc-300 whitespace-nowrap">{Math.round(p.current_mana)}/{Math.round(p.max_mana)}</td>
                        <td className="px-3 py-2.5 font-mono text-yellow-500 whitespace-nowrap">{p.primeval_stones.toLocaleString()}</td>
                        <td className="px-3 py-2.5 font-mono text-purple-400 whitespace-nowrap">{p.immortal_stones.toLocaleString()}</td>
                        <td className="px-3 py-2.5 whitespace-nowrap">{p.is_admin ? <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-950/80 border border-amber-500/40 text-amber-400">ADMIN</span> : <span className="text-zinc-600">–</span>}</td>
                        <td className="px-3 py-2.5 text-zinc-400 max-w-[120px] truncate whitespace-nowrap" title={p.current_region}>{p.current_region}</td>
                        <td className="px-3 py-2.5 text-zinc-500 font-mono text-[10px] whitespace-nowrap">{formatDate(p.created_at)}</td>
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

          {playersTab === "online" && (
            <div className="space-y-4">
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                    <Activity className="w-4 h-4 text-emerald-400" />
                    Người chơi Trực tuyến (5 phút qua)
                  </h3>
                  <button onClick={fetchOnlinePlayers} disabled={onlineLoading} className="flex items-center gap-1.5 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50 cursor-pointer">
                    <RefreshCw className={`w-3.5 h-3.5 ${onlineLoading ? "animate-spin" : ""}`} /> Làm mới
                  </button>
                </div>
                {onlineLoading ? (
                  <div className="flex flex-col items-center justify-center py-16 space-y-3">
                    <RefreshCw className="w-8 h-8 text-emerald-400 animate-spin" />
                    <span className="text-xs text-zinc-500 font-mono">Đang tải...</span>
                  </div>
                ) : onlinePlayers.length === 0 ? (
                  <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
                    <Users className="w-10 h-10 text-zinc-700 mx-auto" />
                    <p className="text-sm text-zinc-400 font-medium">Không có người chơi nào hoạt động gần đây.</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="border-b border-zinc-800">
                          <th className="text-left py-2 px-3 text-zinc-500 font-mono font-semibold uppercase tracking-wider">Username</th>
                          <th className="text-left py-2 px-3 text-zinc-500 font-mono font-semibold uppercase tracking-wider">Cảnh giới</th>
                          <th className="text-left py-2 px-3 text-zinc-500 font-mono font-semibold uppercase tracking-wider">Trạng thái</th>
                          <th className="text-left py-2 px-3 text-zinc-500 font-mono font-semibold uppercase tracking-wider">Hoạt động lần cuối</th>
                        </tr>
                      </thead>
                      <tbody>
                        {onlinePlayers.map((p: ApiResponseData) => (
                          <tr key={p.id} className="border-b border-zinc-800/50 hover:bg-zinc-900/40 transition-colors">
                            <td className="py-2.5 px-3 text-zinc-200 font-medium">{p.username}</td>
                            <td className="py-2.5 px-3">
                              <span className={`${rankColor(p.rank)}`}>{getRankName(p.rank, p.sub_rank)}</span>
                            </td>
                            <td className="py-2.5 px-3">
                              {p.is_online ? (
                                <span className="inline-flex items-center gap-1 text-emerald-400">
                                  <span className="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-pulse" />
                                  Trực tuyến
                                </span>
                              ) : (
                                <span className="text-zinc-500">Ngoại tuyến</span>
                              )}
                            </td>
                            <td className="py-2.5 px-3 text-zinc-400">
                              {p.last_activity ? formatDate(p.last_activity) : "Không có dữ liệu"}
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


        </>
      )}

      {/* Data section: import/export + dictionary management */}
      {section === "data" && (
        <div className="space-y-4">
          <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
            <button onClick={() => setDataTab("import")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "import" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Database className="w-3.5 h-3.5 inline mr-1.5" />Import / Export
            </button>
            <button onClick={() => setDataTab("tables")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "tables" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <List className="w-3.5 h-3.5 inline mr-1.5" />Bảng dữ liệu
            </button>
            <button onClick={() => setDataTab("gus")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "gus" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Bug className="w-3.5 h-3.5 inline mr-1.5" />Cổ trùng
            </button>
            <button onClick={() => setDataTab("items")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "items" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Package className="w-3.5 h-3.5 inline mr-1.5" />Vật phẩm
            </button>
            <button onClick={() => setDataTab("recipes")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "recipes" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <FlaskConical className="w-3.5 h-3.5 inline mr-1.5" />Cổ phương
            </button>
            <button onClick={() => setDataTab("monsters")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "monsters" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <PawPrint className="w-3.5 h-3.5 inline mr-1.5" />Quái vật
            </button>
            <button onClick={() => setDataTab("loot_tables")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "loot_tables" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Gift className="w-3.5 h-3.5 inline mr-1.5" />Bảng Loot
            </button>
            <button onClick={() => setDataTab("regions")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "regions" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Globe className="w-3.5 h-3.5 inline mr-1.5" />Khu vực
            </button>
            <button onClick={() => setDataTab("map_cells")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "map_cells" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <MapPin className="w-3.5 h-3.5 inline mr-1.5" />Ô bản đồ
            </button>
            <button onClick={() => setDataTab("scouting")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "scouting" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Eye className="w-3.5 h-3.5 inline mr-1.5" />Dấu vết truy tung
            </button>
            <button onClick={() => setDataTab("constitution")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "constitution" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Shield className="w-3.5 h-3.5 inline mr-1.5" />Thể chất
            </button>
            <button onClick={() => setDataTab("race")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "race" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Users className="w-3.5 h-3.5 inline mr-1.5" />Chủng tộc
            </button>
            <button onClick={() => setDataTab("status")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "status" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Activity className="w-3.5 h-3.5 inline mr-1.5" />Trạng thái
            </button>
            <button onClick={() => setDataTab("spawns")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "spawns" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <MapPin className="w-3.5 h-3.5 inline mr-1.5" />Điểm Spawn
            </button>
            <button onClick={() => setDataTab("quests")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "quests" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <ListTodo className="w-3.5 h-3.5 inline mr-1.5" />Nhiệm vụ SG
            </button>
            <button onClick={() => setDataTab("dungeons")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex-shrink-0 ${dataTab === "dungeons" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Shield className="w-3.5 h-3.5 inline mr-1.5" />Phụ bản
            </button>
          </div>

          {dataTab === "import" && (
            <>
            <DataImportWorkflow canImportDataset={canAdminAction("importGameDataset")} canImportAll={canAdminAction("importAllGameData")} addToast={addToast} />
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-6">
              <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-2">
                <Database className="w-4 h-4 text-emerald-400" />
                Import / Export CSV
              </h3>
              <p className="text-xs text-zinc-500 mb-6">
                Import dữ liệu từ file CSV vào database, hoặc export dữ liệu từ database ra file CSV.
              </p>
              <div className="flex flex-wrap gap-4">
                <button disabled={actionLoading} onClick={async () => { const ok = await exec(`${BACKEND_URL}/api/admin/data/export`, "POST", {}); if (ok) addToast("Export thành công!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <RefreshCw className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Export Database → CSV"}
                </button>
                <button disabled={actionLoading} onClick={async () => { const ok = await exec(`${BACKEND_URL}/api/admin/clear-cache`, "POST", {}); if (ok) addToast("Cache đã được xóa và tải lại!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <Trash2 className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Xóa Cache Dictionary"}
                </button>
                {canAdminAction("flushRedis") && <button disabled={actionLoading} onClick={async () => { const prepared = prepareAdminAction("flushRedis", "xóa toàn bộ Redis cache", "Redis"); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/clear-redis-cache`, "POST", prepared); if (ok) addToast("Toàn bộ cache Redis đã được xóa!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <Trash2 className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Xóa Cache Redis"}
                </button>}
              </div>
            </div>
            </>
          )}

          {dataTab === "tables" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-6">
              <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-2">
                <List className="w-4 h-4 text-emerald-400" />
                Import / Export từng bảng
              </h3>
              <p className="text-xs text-zinc-500 mb-6">
                Import hoặc export dữ liệu cho từng bảng riêng lẻ. Mỗi thao tác chỉ xử lý một bảng.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {datasets.map(t => (
                  <div key={t.key} className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4 flex flex-col gap-3">
                    <div className="flex items-center gap-2">
                      <Database className="w-4 h-4 text-amber-400" />
                      <span className="text-xs font-semibold text-zinc-200">{t.label}</span>
                      <span className="text-[10px] text-zinc-500 font-mono ml-auto">{t.csv_file}</span>
                    </div>
                    <div className="flex gap-2">
                      {t.capabilities.import && (adminAccess.effective_permissions.includes("admin.*") || (!!t.permissions.import && adminAccess.effective_permissions.includes(t.permissions.import))) && <button
                        disabled={tableLoading === t.key}
                        onClick={() => setDataTab("import")}
                        className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-lg text-[11px] font-semibold transition-all cursor-pointer"
                      >
                        <Upload className="w-3 h-3" />
                        {tableLoading === t.key ? "..." : "Import"}
                      </button>}
                      {t.capabilities.export && (adminAccess.effective_permissions.includes("admin.*") || (!!t.permissions.export && adminAccess.effective_permissions.includes(t.permissions.export))) && <button
                        disabled={tableLoading === t.key}
                        onClick={async () => {
                          setTableLoading(t.key);
                          const ok = await exec(`${BACKEND_URL}/api/admin/data/export/${t.key}`, "POST", {});
                          setTableLoading(null);
                          if (ok) addToast(`Export ${t.label} thành công!`, "success");
                        }}
                        className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-lg text-[11px] font-semibold transition-all cursor-pointer"
                      >
                        <RefreshCw className="w-3 h-3" />
                        {tableLoading === t.key ? "..." : "Export"}
                      </button>}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {dataTab === "gus" && (
            <CoreDataAdminTab kind="gus" rows={gus} loading={gusLoading}
              onCreate={() => setGuEdit({ id: "", name: "", rank: 1, path: [], gu_type: "Active", base_mana_cost: 0, effects: [], food_types: [], hunger_rate: 1, max_hunger: 100, description: "", target_type: "SINGLE", rarity: 0, primary_role: "", combat_roles: [], combat_tags: [], support_tags: [], base_price: 0, image_url: "", _isNew: true })}
              onEdit={(g) => setGuEdit({ ...structuredClone(g), _old_rank: g.rank, effects: typeof g.effects === "string" ? JSON.parse(g.effects) : g.effects })}
              onCopy={(g) => setGuEdit({ ...structuredClone(g), id: "", _isNew: true, effects: typeof g.effects === "string" ? JSON.parse(g.effects) : g.effects })}
              onDelete={async (g) => { if (!confirm(`Xóa toàn bộ các rank của ${g.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/gu-dictionary/${g.id}`, "DELETE"); if (ok) fetchGus(); }} />
          )}

          {dataTab === "items" && (
            <CoreDataAdminTab kind="items" rows={items} loading={itemsLoading}
              canCreate={canAdminAction("updateItemDictionary")} canEdit={canAdminAction("updateItemDictionary")} canCopy={canAdminAction("updateItemDictionary")} canDelete={canAdminAction("deleteItemDictionary")}
              onCreate={() => setItemEdit({ id: "", name: "", item_type: "Material", rank: 1, effects: { schema_version: 2, blocks: [] }, description: "", base_price: 0, image_url: "", rarity: 0, path: "", _isNew: true })}
              onEdit={(item) => setItemEdit({ ...structuredClone(item), effects: typeof item.effects === "string" ? JSON.parse(item.effects) : item.effects })}
              onCopy={(item) => setItemEdit({ ...structuredClone(item), id: "", _isNew: true, effects: typeof item.effects === "string" ? JSON.parse(item.effects) : item.effects })}
              onDelete={async (item) => { const prepared = prepareAdminAction("deleteItemDictionary", "xóa vật phẩm", String(item.id)); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/item-dictionary/${item.id}`, "DELETE", prepared); if (ok) fetchItems(); }} />
          )}

          {dataTab === "recipes" && (
            <CoreDataAdminTab kind="recipes" rows={recipes} loading={recipesLoading}
              canCreate={canAdminAction("updateRecipe")} canEdit={canAdminAction("updateRecipe")} canCopy={canAdminAction("updateRecipe")} canDelete={canAdminAction("deleteRecipe")}
              onCreate={() => setRecipeEdit({ id: "", target_gu_id: "", target_rank: 1, target_gu_name: "", required_gus: {}, required_materials: {}, success_rate: 0.5, primeval_stones_cost: 0, immortal_essence_stones_cost: 0, _isNew: true })}
              onEdit={(recipe) => setRecipeEdit({ ...structuredClone(recipe), required_gus: typeof recipe.required_gus === "string" ? JSON.parse(recipe.required_gus) : recipe.required_gus, required_materials: typeof recipe.required_materials === "string" ? JSON.parse(recipe.required_materials) : recipe.required_materials })}
              onCopy={(recipe) => setRecipeEdit({ ...structuredClone(recipe), id: "", _isNew: true, required_gus: typeof recipe.required_gus === "string" ? JSON.parse(recipe.required_gus) : recipe.required_gus, required_materials: typeof recipe.required_materials === "string" ? JSON.parse(recipe.required_materials) : recipe.required_materials })}
              onDelete={async (recipe) => { const prepared = prepareAdminAction("deleteRecipe", "xóa cổ phương", String(recipe.id)); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/recipes/${recipe.id}`, "DELETE", prepared); if (ok) fetchRecipes(); }} />
          )}

          {dataTab === "monsters" && (
            <CoreDataAdminTab kind="monsters" rows={monsters} loading={monstersLoading}
              canCreate={canAdminAction("updateMonster")} canEdit={canAdminAction("updateMonster")} canCopy={canAdminAction("updateMonster")} canDelete={canAdminAction("deleteMonster")}
              onCreate={() => setMonsterEdit({ id: "", name: "", type: "beast", max_hp: 100, base_attack: 10, base_defense: 5, gu_inventory: [], gu_pool: [], item_inventory: {}, dao_marks: {}, path_masteries: {}, ai_profile: "beast_basic", loot_table_id: "", description: "", is_active: true, is_capturable: false, capture_chance: 0.20, pet_hunger_per_hour: 2.0, _isNew: true })}
              onEdit={(monster) => setMonsterEdit({ ...structuredClone(monster), gu_inventory: typeof monster.gu_inventory === "string" ? JSON.parse(monster.gu_inventory) : monster.gu_inventory || [], gu_pool: typeof monster.gu_pool === "string" ? JSON.parse(monster.gu_pool) : monster.gu_pool || [], item_inventory: typeof monster.item_inventory === "string" ? JSON.parse(monster.item_inventory) : monster.item_inventory || {}, dao_marks: typeof monster.dao_marks === "string" ? JSON.parse(monster.dao_marks) : monster.dao_marks || {}, path_masteries: typeof monster.path_masteries === "string" ? JSON.parse(monster.path_masteries) : monster.path_masteries || {} })}
              onCopy={(monster) => setMonsterEdit({ ...structuredClone(monster), id: "", _isNew: true, gu_inventory: typeof monster.gu_inventory === "string" ? JSON.parse(monster.gu_inventory) : monster.gu_inventory || [], gu_pool: typeof monster.gu_pool === "string" ? JSON.parse(monster.gu_pool) : monster.gu_pool || [], item_inventory: typeof monster.item_inventory === "string" ? JSON.parse(monster.item_inventory) : monster.item_inventory || {}, dao_marks: typeof monster.dao_marks === "string" ? JSON.parse(monster.dao_marks) : monster.dao_marks || {}, path_masteries: typeof monster.path_masteries === "string" ? JSON.parse(monster.path_masteries) : monster.path_masteries || {} })}
              onDelete={async (monster) => { const prepared = prepareAdminAction("deleteMonster", "xóa quái vật", String(monster.name)); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/monsters/${monster.id}`, "DELETE", prepared); if (ok) fetchMonsters(); }} />
          )}

          {dataTab === "loot_tables" && (
            <CoreDataAdminTab kind="loot_tables" rows={lootTables} loading={lootTablesLoading}
              canCreate={canAdminAction("updateLootTable")} canEdit={canAdminAction("updateLootTable")} canCopy={canAdminAction("updateLootTable")} canDelete={canAdminAction("deleteLootTable")}
              onCreate={() => setLootTableEdit({ id: "", name: "", entries: [], _isNew: true })}
              onEdit={(table) => setLootTableEdit({ ...structuredClone(table), entries: typeof table.entries === "string" ? JSON.parse(table.entries) : table.entries || [] })}
              onCopy={(table) => setLootTableEdit({ ...structuredClone(table), id: "", _isNew: true, entries: typeof table.entries === "string" ? JSON.parse(table.entries) : table.entries || [] })}
              onDelete={async (table) => { const prepared = prepareAdminAction("deleteLootTable", "xóa bảng loot", String(table.name)); if (!prepared) return; const ok = await exec(`${BACKEND_URL}/api/admin/loot-tables/${table.id}`, "DELETE", prepared); if (ok) fetchLootTables(); }} />
          )}

          {dataTab === "regions" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <Globe className="w-4 h-4 text-amber-400" />Quản Lý Khu Vực
                </h3>
                {canAdminAction("updateRegion") && <button
                  onClick={() => openRegionEditor({ id: "", name: "", description: "", world_map_id: "world_map", width: 100, height: 100, min_rank: 1, max_rank: 5, is_active: true, zone_type: "KHU_VUC", min_x: 0, min_y: 0, max_x: 0, max_y: 0, entry_x: 0, entry_y: 0, safe_zone_x: null, safe_zone_y: null, parent_id: "", paths: [], rules: {}, pool_weights: { cell_type_weights: {}, resources: [], wild_gus: [], monster_gus: [] }, _isNew: true })}
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer w-full sm:w-auto"
                >
                  <Plus className="w-3.5 h-3.5" />Thêm Khu Vực
                </button>}
              </div>

              <div className="flex gap-2 mb-3">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                  <input
                    value={regionSearch}
                    onChange={e => setRegionSearch(e.target.value)}
                    onKeyDown={e => { if (e.key === "Enter") fetchRegions(1, regionSearch); }}
                    placeholder="Tìm kiếm khu vực theo ID hoặc Tên..."
                    className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500"
                  />
                </div>
                <button onClick={() => fetchRegions(1, regionSearch)} className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-xs font-bold text-zinc-200 cursor-pointer">Tìm</button>
              </div>

              {regionsLoading ? (
                <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
              ) : regions.length === 0 ? (
                <div className="text-center py-8 text-zinc-500 text-xs">Không tìm thấy khu vực nào.</div>
              ) : (
                <>
                  <AdminDataGrid rows={regions} loading={regionsLoading} showSearch={false} showPagination={false} sortable={false} rowKey={(region) => region.id} columns={[
                      { key: "id", label: "ID", value: (region) => region.id, className: "font-mono text-zinc-400" }, { key: "name", label: "Tên", value: (region) => region.name, className: "font-semibold text-zinc-200" }, { key: "world_map_id", label: "Tầng", value: (region) => region.world_map_id || "world_map", className: "text-cyan-300" }, { key: "description", label: "Mô tả", value: (region) => region.description, className: "max-w-[150px] truncate text-zinc-400" }, { key: "zone_type", label: "Loại", value: (region) => region.zone_type, className: "text-amber-400" }, { key: "size", label: "Kích thước", value: (region) => `${region.width}x${region.height}` }, { key: "coordinates", label: "Tọa độ", value: (region) => `${region.min_x},${region.min_y},${region.max_x},${region.max_y}`, render: (region) => `(${region.min_x},${region.min_y}) → (${region.max_x},${region.max_y})` }, { key: "rank", label: "Cảnh giới", value: (region) => `${region.min_rank}-${region.max_rank}`, render: (region) => `C${region.min_rank} - C${region.max_rank}` },
                    ]} onEdit={canAdminAction("updateRegion") ? (region) => openRegionEditor(structuredClone(region)) : undefined} onCopy={canAdminAction("updateRegion") ? (region) => openRegionEditor({ ...structuredClone(region), id: "", _isNew: true }) : undefined} onDelete={canAdminAction("deleteRegion") ? async (region) => { const endpoint = `${BACKEND_URL}/api/admin/regions/${region.id}`; const prepared = prepareAdminAction("deleteRegion", "xóa khu vực", new URL(endpoint).pathname); if (!prepared) return; const ok = await exec(endpoint, "DELETE", prepared); if (ok) fetchRegions(regionsPage, regionSearch); } : undefined} />
                  <div className="flex items-center justify-between mt-4 text-xs font-semibold text-zinc-500">
                    <div>Tổng số: <span className="text-zinc-300">{regionsTotal}</span> khu vực</div>
                    <div className="flex items-center gap-2">
                      <span>Trang {regionsPage}/{regionsTotalPages}</span>
                      <button onClick={() => fetchRegions(regionsPage - 1)} disabled={regionsPage <= 1} className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded disabled:opacity-50 text-zinc-300 hover:text-white cursor-pointer">Trước</button>
                      <button onClick={() => fetchRegions(regionsPage + 1)} disabled={regionsPage >= regionsTotalPages} className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded disabled:opacity-50 text-zinc-300 hover:text-white cursor-pointer">Sau</button>
                    </div>
                  </div>
                </>
              )}
            </div>
          )}

          {dataTab === "scouting" && (
            <div className="space-y-3">
              <div className="flex gap-2"><input value={residualTraceSearch} onChange={e=>setResidualTraceSearch(e.target.value)} onKeyDown={e=>{if(e.key==="Enter"){setResidualTracePage(1);fetchResidualTraces(1,residualTraceSearch);}}} placeholder="Tìm khu vực, người chơi, loại hành động hoặc đạo mạch..." className="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200"/><button onClick={()=>{setResidualTracePage(1);fetchResidualTraces(1,residualTraceSearch);}} className="px-3 py-2 rounded-lg bg-zinc-800 text-xs text-zinc-200"><Search className="w-3.5 h-3.5 inline mr-1"/>Tìm</button></div>
              {residualTracesLoading?<div className="py-12 text-center text-xs text-zinc-500">Đang tải dấu vết thế giới...</div>:residualTraces.length===0?<div className="py-12 text-center text-xs text-zinc-500 border border-dashed border-zinc-800 rounded-xl">Không có dấu vết truy tung</div>:<div className="overflow-x-auto rounded-xl border border-zinc-800/80"><table className="w-full text-left text-xs"><thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase text-[10px]"><th className="px-3 py-2.5">Khu vực / tọa độ</th><th className="px-3 py-2.5">Người chơi</th><th className="px-3 py-2.5">Sự kiện</th><th className="px-3 py-2.5">Đạo mạch / xung lực</th><th className="px-3 py-2.5">Tạo / hết hạn</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody>{residualTraces.map(trace=><tr key={String(trace.id)} className="border-t border-zinc-800/40"><td className="px-3 py-2.5 text-zinc-300">{String(trace.region_id)} ({String(trace.x)}, {String(trace.y)})</td><td className="px-3 py-2.5 text-zinc-300">{String(trace.player_username)}<span className="block text-[10px] text-zinc-600 font-mono">{String(trace.player_id)}</span></td><td className="px-3 py-2.5 text-zinc-400">{String(trace.action_type)}</td><td className="px-3 py-2.5 text-zinc-400">{String(trace.dominant_path)} · {String(trace.dao_pulse_strength)}</td><td className="px-3 py-2.5 text-zinc-500">{new Date(String(trace.created_at)).toLocaleString("vi-VN")}<span className="block text-[10px]">Hết: {new Date(String(trace.expires_at)).toLocaleString("vi-VN")}</span></td><td className="px-3 py-2.5 text-right">{canAdminAction("updateResidualTrace")&&<button onClick={()=>{setResidualTraceForm({region_id:String(trace.region_id),x:String(trace.x),y:String(trace.y),player_id:String(trace.player_id),action_type:String(trace.action_type),dominant_path:String(trace.dominant_path),dao_pulse_strength:String(trace.dao_pulse_strength),expires_at:new Date(String(trace.expires_at)).toISOString().slice(0,16)});setResidualTraceEdit(trace);}} className="p-1 text-zinc-500 hover:text-amber-400 cursor-pointer mr-1"><Edit3 className="w-3.5 h-3.5"/></button>}{canAdminAction("deleteResidualTrace")&&<button onClick={async()=>{const endpoint=`${BACKEND_URL}/api/admin/residual-traces/${trace.id}`;const prepared=prepareAdminAction("deleteResidualTrace","xóa dấu vết truy tung",`${trace.action_type} · ${trace.id}`);if(!prepared)return;await exec(endpoint,"DELETE",prepared);}} className="p-1 text-zinc-500 hover:text-red-400 cursor-pointer"><Trash2 className="w-3.5 h-3.5"/></button>}</td></tr>)}</tbody></table></div>}
              <div className="flex justify-between text-xs text-zinc-500"><span>Tổng {residualTraceTotal} dấu vết</span><div className="flex gap-2"><button onClick={()=>{const p=Math.max(1,residualTracePage-1);setResidualTracePage(p);fetchResidualTraces(p,residualTraceSearch);}} disabled={residualTracePage<=1} className="px-3 py-1 bg-zinc-900 rounded disabled:opacity-40">Trước</button><span className="py-1">Trang {residualTracePage}/{Math.max(1,Math.ceil(residualTraceTotal/30))}</span><button onClick={()=>{const p=residualTracePage+1;setResidualTracePage(p);fetchResidualTraces(p,residualTraceSearch);}} disabled={residualTracePage>=Math.ceil(residualTraceTotal/30)} className="px-3 py-1 bg-zinc-900 rounded disabled:opacity-40">Sau</button></div></div>
            </div>
          )}

          {dataTab === "map_cells" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <MapPin className="w-4 h-4 text-amber-400" />Quản Lý Ô Bản Đồ
                </h3>
                {canAdminAction("updateMapCell") && <button
                  onClick={() => setMapCellEdit({ id: "", region_id: "", x: 0, y: 0, cell_type: "Resource", content_id: "", metadata: {}, respawn_seconds: 300, is_active: true, last_cleared_at: null, cleared_by: null, _isNew: true })}
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer w-full sm:w-auto"
                >
                  <Plus className="w-3.5 h-3.5" />Thêm Ô Bản Đồ
                </button>}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-4 gap-2 mb-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tìm kiếm</label>
                  <input
                    value={mapCellSearch}
                    onChange={e => setMapCellSearch(e.target.value)}
                    onKeyDown={e => { if (e.key === "Enter") fetchMapCells(1, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter); }}
                    placeholder="Tọa độ x, y hoặc Content ID..."
                    className="w-full px-3 py-1.5 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"
                  />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Khu vực</label>
                  <input
                    value={mapCellRegionFilter}
                    onChange={e => setMapCellRegionFilter(e.target.value)}
                    placeholder="ID khu vực..."
                    className="w-full px-3 py-1.5 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"
                  />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại ô</label>
                  <select
                    value={mapCellTypeFilter}
                    onChange={e => setMapCellTypeFilter(e.target.value)}
                    className="w-full px-3 py-1.5 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"
                  >
                    <option value="">Tất cả</option>
                    <option value="Resource">Resource (Tài nguyên)</option>
                    <option value="Monster">Monster (Quái/Boss)</option>
                    <option value="Safe">Safe (Làng/Trạm nghỉ)</option>
                    <option value="Portal">Portal (Cổng chuyển dịch)</option>
                  </select>
                </div>
                <div className="flex items-end">
                  <button
                    onClick={() => fetchMapCells(1, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter)}
                    className="w-full py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-xs font-bold text-zinc-200 cursor-pointer"
                  >
                    Tìm Kiếm
                  </button>
                </div>
              </div>

              {mapCellsLoading ? (
                <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
              ) : mapCells.length === 0 ? (
                <div className="text-center py-8 text-zinc-500 text-xs">Không tìm thấy ô bản đồ nào.</div>
              ) : (
                <>
                  <AdminDataGrid rows={mapCells} loading={mapCellsLoading} showSearch={false} showPagination={false} sortable={false} rowKey={(cell) => String(cell.id)} columns={[
                      { key: "region_id", label: "Khu vực", value: (cell) => cell.region_id, className: "font-mono text-[10px]" }, { key: "coordinates", label: "Tọa độ", value: (cell) => `${cell.x},${cell.y}`, render: (cell) => `(${cell.x}, ${cell.y})`, className: "font-mono text-amber-400" }, { key: "cell_type", label: "Loại ô", value: (cell) => cell.cell_type }, { key: "content_id", label: "Content ID", value: (cell) => cell.content_id, className: "font-mono text-zinc-400" }, { key: "respawn_seconds", label: "Respawn", value: (cell) => cell.respawn_seconds, render: (cell) => `${cell.respawn_seconds}s` }, { key: "metadata", label: "Metadata", value: (cell) => cell.metadata, render: (cell) => <span className="block max-w-[120px] truncate font-mono text-[10px] text-zinc-500">{JSON.stringify(cell.metadata)}</span> }, { key: "is_active", label: "Active", value: (cell) => cell.is_active, render: (cell) => cell.is_active ? "ON" : "OFF" },
                    ]} onEdit={canAdminAction("updateMapCell") ? (cell) => setMapCellEdit(structuredClone(cell)) : undefined} onCopy={canAdminAction("updateMapCell") ? (cell) => setMapCellEdit({ ...structuredClone(cell), id: "", last_cleared_at: null, cleared_by: null, _isNew: true }) : undefined} onDelete={canAdminAction("deleteMapCell") ? async (cell) => { const endpoint = `${BACKEND_URL}/api/admin/map-cells/${cell.id}`; const prepared = prepareAdminAction("deleteMapCell", "xóa ô bản đồ", new URL(endpoint).pathname); if (!prepared) return; const ok = await exec(endpoint, "DELETE", prepared); if (ok) fetchMapCells(mapCellsPage, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter); } : undefined} />
                  <div className="flex items-center justify-between mt-4 text-xs font-semibold text-zinc-500">
                    <div>Tổng số: <span className="text-zinc-300">{mapCellsTotal}</span> ô</div>
                    <div className="flex items-center gap-2">
                      <span>Trang {mapCellsPage}/{mapCellsTotalPages}</span>
                      <button onClick={() => fetchMapCells(mapCellsPage - 1)} disabled={mapCellsPage <= 1} className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded disabled:opacity-50 text-zinc-300 hover:text-white cursor-pointer">Trước</button>
                      <button onClick={() => fetchMapCells(mapCellsPage + 1)} disabled={mapCellsPage >= mapCellsTotalPages} className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded disabled:opacity-50 text-zinc-300 hover:text-white cursor-pointer">Sau</button>
                    </div>
                  </div>
                </>
              )}
            </div>
          )}

          {dataTab === "constitution" && <DictionariesAdminTab type="constitution" addToast={addToast} canWrite={canAdminAction("updateConstitution")} canDelete={canAdminAction("deleteConstitution")} />}
          {dataTab === "race" && <DictionariesAdminTab type="race" addToast={addToast} canWrite={canAdminAction("updateRaceDictionary")} canDelete={canAdminAction("deleteRaceDictionary")} />}
          {dataTab === "status" && <DictionariesAdminTab type="status" addToast={addToast} canWrite={canAdminAction("updateStatusDictionary")} canDelete={canAdminAction("deleteStatusDictionary")} />}
          {dataTab === "spawns" && <WorldExtendedAdminTab type="spawns" addToast={addToast} canWrite={canAdminAction("updateMonsterSpawn")} canDelete={canAdminAction("deleteMonsterSpawn")} />}
          {dataTab === "quests" && <WorldExtendedAdminTab type="quests" addToast={addToast} canWrite={canAdminAction("updateWorldQuest")} canDelete={canAdminAction("deleteWorldQuest")} />}
          {dataTab === "dungeons" && <WorldExtendedAdminTab type="dungeons" addToast={addToast} canWrite={canAdminAction("updateDungeon")} canDelete={canAdminAction("deleteDungeon")} />}
        </div>
      )}

      {/* Topup & Wallet Section */}
      {section === "topup_wallet" && <TopupWalletAdminTab addToast={addToast} />}

      {/* Gu History Section */}
      {section === "gu_history" && <GuHistoryAdminTab addToast={addToast} />}

      {/* Runtime monitoring Section */}
      {section === "monitoring" && <MonitoringAdminTab addToast={addToast} />}

      {/* Audit Logs section */}
      {section === "logs" && <AuditLogsAdminTab />}
      {section === "combat_history" && <CombatHistoryAdminTab addToast={addToast} />}
      {section === "map_drops" && <MapDropsAdminTab addToast={addToast} canEdit={canAdminAction("updateMapDrop")} canDelete={canAdminAction("deleteMapDrop")} />}
      {section === "arena_rankings" && <ArenaRankingsAdminTab addToast={addToast} canEdit={canAdminAction("updateArenaRanking")} />}
      {section === "world_bosses" && <WorldBossesAdminTab addToast={addToast} canEdit={canAdminAction("updateWorldBoss")} canDelete={canAdminAction("purgeWorldBosses")} />}

      {/* Shop section */}
      {section === "shop" && (
        <div className="space-y-4">
          {/* Sub-toggle */}
          <div className="flex gap-1 bg-zinc-900/60 border border-zinc-800/60 rounded-xl p-1 w-fit">
            <button onClick={() => setShopTab("caravan")} className={`px-4 py-2 text-xs font-semibold font-mono rounded-lg transition-colors cursor-pointer ${shopTab === "caravan" ? "bg-emerald-600/30 text-emerald-400 border border-emerald-700/50" : "text-zinc-500 hover:text-zinc-300 border border-transparent"}`}>
              <Package className="w-3.5 h-3.5 inline mr-1.5" />Cửa hàng thường
            </button>
            <button onClick={() => setShopTab("arena")} className={`px-4 py-2 text-xs font-semibold font-mono rounded-lg transition-colors cursor-pointer ${shopTab === "arena" ? "bg-amber-600/30 text-amber-400 border border-amber-700/50" : "text-zinc-500 hover:text-zinc-300 border border-transparent"}`}>
              <Sword className="w-3.5 h-3.5 inline mr-1.5" />Đấu Trường
            </button>
          </div>

          {shopTab === "caravan" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <Package className="w-4 h-4 text-emerald-400" />
                  Quản Lý Cửa Hàng
                </h3>
                {canAdminAction("updateShop") && <button onClick={() => { setShopForm({ id: "", rank: 0, price: 100, stock: -1, is_default: true, sort_order: 0 }); setShowShopAdd(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer">
                  <Plus className="w-3.5 h-3.5" />Thêm
                </button>}
              </div>
              {shopLoading ? (
                <div className="flex justify-center py-10"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
              ) : shopItems.length === 0 ? (
                <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                  <Package className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                  <p className="text-xs text-zinc-500">Chưa có mặt hàng nào.</p>
                </div>
              ) : (
                <AdminDataGrid rows={shopItems} loading={shopLoading} rowKey={(item) => `${item.id}:${item.rank ?? 0}`} columns={[{ key: "id", label: "ID", value: (item) => item.id }, { key: "rank", label: "Rank", value: (item) => item.rank ?? 0 }, { key: "name", label: "Tên", value: (item) => item.name || item.id }, { key: "item_type", label: "Loại", value: (item) => item.item_type }, { key: "price", label: "Giá", value: (item) => item.price }, { key: "stock", label: "Tồn kho", value: (item) => item.stock, render: (item) => item.stock === -1 ? "∞" : item.stock }, { key: "is_default", label: "Mặc định", value: (item) => item.is_default, render: (item) => item.is_default ? "✓" : "–" }, { key: "sort_order", label: "Thứ tự", value: (item) => item.sort_order }]} onEdit={canAdminAction("updateShop") ? (item) => { setShopForm({ id: item.id, rank: item.rank ?? 0, price: item.price, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowShopEdit(`${item.id}/${item.rank ?? 0}`); } : undefined} onCopy={canAdminAction("updateShop") ? (item) => { setShopForm({ id: "", rank: item.rank ?? 0, price: item.price, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowShopAdd(true); } : undefined} onDelete={canAdminAction("deleteShop") ? async (item) => { const endpoint = `${BACKEND_URL}/api/admin/shop/${item.id}/${item.rank ?? 0}`; const prepared = prepareAdminAction("deleteShop", "xóa mặt hàng khỏi cửa hàng", new URL(endpoint).pathname); if (!prepared) return; const ok = await exec(endpoint, "DELETE", prepared); if (ok) fetchShopItems(); } : undefined} />
              )}
            </div>
          )}

          {shopTab === "arena" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <Sword className="w-4 h-4 text-amber-400" />
                  Quản Lý Shop Đấu Trường
                </h3>
                {canAdminAction("updateArenaShop") && <button onClick={() => { setArenaShopForm({ id: "", price: 50, qty: 1, stock: -1, is_default: true, sort_order: 0 }); setShowArenaShopAdd(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">
                  <Plus className="w-3.5 h-3.5" />Thêm
                </button>}
              </div>
              {arenaShopLoading ? (
                <div className="flex justify-center py-10"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
              ) : arenaShopItems.length === 0 ? (
                <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                  <Sword className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                  <p className="text-xs text-zinc-500">Chưa có mặt hàng nào.</p>
                </div>
              ) : (
                <AdminDataGrid rows={arenaShopItems} loading={arenaShopLoading} rowKey={(item) => item.id} columns={[{ key: "id", label: "ID", value: (item) => item.id }, { key: "name", label: "Tên", value: (item) => item.name || item.id }, { key: "price", label: "Giá (Xu ĐT)", value: (item) => item.price }, { key: "qty", label: "SL nhận", value: (item) => item.qty }, { key: "stock", label: "Tồn kho", value: (item) => item.stock, render: (item) => item.stock === -1 ? "∞" : item.stock }, { key: "is_default", label: "Mặc định", value: (item) => item.is_default, render: (item) => item.is_default ? "✓" : "–" }, { key: "sort_order", label: "Thứ tự", value: (item) => item.sort_order }]} onEdit={canAdminAction("updateArenaShop") ? (item) => { setArenaShopForm({ id: item.id, price: item.price, qty: item.qty, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowArenaShopEdit(item.id); } : undefined} onCopy={canAdminAction("updateArenaShop") ? (item) => { setArenaShopForm({ id: "", price: item.price, qty: item.qty, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowArenaShopAdd(true); } : undefined} onDelete={canAdminAction("deleteArenaShop") ? async (item) => { const endpoint = `${BACKEND_URL}/api/admin/arena-shop/${item.id}`; const prepared = prepareAdminAction("deleteArenaShop", "xóa mặt hàng Arena Shop", new URL(endpoint).pathname); if (!prepared) return; const ok = await exec(endpoint, "DELETE", prepared); if (ok) fetchArenaShopItems(); } : undefined} />
              )}
            </div>
          )}
        </div>
      )}
      {/* Add Shop Item Modal */}
      {showShopAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowShopAdd(false)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Thêm mặt hàng</h3>
            <div className="space-y-3">
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label>
                <div className="flex gap-2">
                  <input value={shopForm.id} onChange={e => setShopForm(f => ({ ...f, id: e.target.value }))} placeholder="Nhập ID hoặc chọn..." className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "shop", tab: "gu", query: ""}); }} className="px-3 py-2 bg-zinc-700 hover:bg-zinc-600 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1">
                    <Search className="w-3 h-3" />Chọn
                  </button>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Rank (0 = tự động)</label>
                  <input type="number" value={shopForm.rank} onChange={e => setShopForm(f => ({ ...f, rank: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
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
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Sửa mặt hàng: {shopForm.id} (rank {shopForm.rank ?? 0})</h3>
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
      {/* Add Arena Shop Item Modal */}
      {showArenaShopAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowArenaShopAdd(false)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Thêm mặt hàng Arena Shop</h3>
            <div className="space-y-3">
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label>
                <div className="flex gap-2">
                  <input value={arenaShopForm.id} onChange={e => setArenaShopForm(f => ({ ...f, id: e.target.value }))} placeholder="Nhập ID hoặc chọn..." className="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "arena", tab: "item", query: ""}); }} className="px-3 py-2 bg-zinc-700 hover:bg-zinc-600 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1">
                    <Search className="w-3 h-3" />Chọn
                  </button>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá (Xu ĐT)</label>
                  <input type="number" value={arenaShopForm.price} onChange={e => setArenaShopForm(f => ({ ...f, price: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">SL nhận</label>
                  <input type="number" value={arenaShopForm.qty} onChange={e => setArenaShopForm(f => ({ ...f, qty: parseInt(e.target.value) || 1 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tồn kho (-1: vô hạn)</label>
                  <input type="number" value={arenaShopForm.stock} onChange={e => setArenaShopForm(f => ({ ...f, stock: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Thứ tự</label>
                  <input type="number" value={arenaShopForm.sort_order} onChange={e => setArenaShopForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={arenaShopForm.is_default} onChange={e => setArenaShopForm(f => ({ ...f, is_default: e.target.checked }))} className="rounded bg-zinc-800 border-zinc-700" />
                <span className="text-[11px] text-zinc-400">Mặc định</span>
              </label>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowArenaShopAdd(false)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const ok = await exec(`${BACKEND_URL}/api/admin/arena-shop`, "POST", arenaShopForm);
                if (ok) { setShowArenaShopAdd(false); fetchArenaShopItems(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Thêm</button>
            </div>
          </div>
        </div>
      )}
      {/* Edit Arena Shop Item Modal */}
      {showArenaShopEdit && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowArenaShopEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">Sửa mặt hàng Arena Shop: {arenaShopForm.id}</h3>
            <div className="space-y-3">
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá (Xu ĐT)</label>
                  <input type="number" value={arenaShopForm.price} onChange={e => setArenaShopForm(f => ({ ...f, price: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">SL nhận</label>
                  <input type="number" value={arenaShopForm.qty} onChange={e => setArenaShopForm(f => ({ ...f, qty: parseInt(e.target.value) || 1 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tồn kho</label>
                  <input type="number" value={arenaShopForm.stock} onChange={e => setArenaShopForm(f => ({ ...f, stock: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div className="flex-1">
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Thứ tự</label>
                  <input type="number" value={arenaShopForm.sort_order} onChange={e => setArenaShopForm(f => ({ ...f, sort_order: parseInt(e.target.value) || 0 }))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={arenaShopForm.is_default} onChange={e => setArenaShopForm(f => ({ ...f, is_default: e.target.checked }))} className="rounded bg-zinc-800 border-zinc-700" />
                <span className="text-[11px] text-zinc-400">Mặc định</span>
              </label>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowArenaShopEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const ok = await exec(`${BACKEND_URL}/api/admin/arena-shop/${showArenaShopEdit}`, "PUT", arenaShopForm);
                if (ok) { setShowArenaShopEdit(null); fetchArenaShopItems(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}
      {/* Mines section */}
      {section === "mines" && (
        <div className="space-y-4">
          <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap">
            <button
              onClick={() => setMinesTab("slots")}
              className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${
                minesTab === "slots" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
              }`}
            >
              <Layers className="w-3.5 h-3.5 inline mr-1.5" />Mỏ Cá Nhân (31 Ô)
            </button>
            <button
              onClick={() => setMinesTab("sect_mines")}
              className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${
                minesTab === "sect_mines" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
              }`}
            >
              <Shield className="w-3.5 h-3.5 inline mr-1.5" />Mỏ Tông Môn (5 Mỏ Lớn)
            </button>
          </div>

          {minesTab === "sect_mines" ? (
            <SectMinesAdminTab addToast={addToast} />
          ) : (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-4">
                <Layers className="w-4 h-4 text-amber-400" />
                Quản Lý Mỏ Linh Thạch Cá Nhân
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
                      <th className="px-3 py-2.5 font-semibold">Loại mỏ</th>
                      <th className="px-3 py-2.5 font-semibold">Phẩm chất</th>
                      <th className="px-3 py-2.5 font-semibold">Chủ sở hữu</th>
                      <th className="px-3 py-2.5 font-semibold">Quái canh</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Cấp Q</th>
                      <th className="px-3 py-2.5 font-semibold">Chiếm giữ</th>
                      <th className="px-3 py-2.5 font-semibold"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {mineSlots.map((slot: ApiResponseData) => (
                      <tr key={slot.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono">{slot.id}</td>
                        <td className="px-3 py-2.5 text-zinc-400">{slot.mine_type || "mortal"}</td>
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
          )}
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
                  <option value="Siêu Cấp">Siêu Cấp</option>
                  <option value="Truyền Thuyết">Truyện Thuyết</option>
                  <option value="Sử Thi">Sử Thi</option>
                  <option value="Hiếm">Hiếm</option>
                  <option value="Thường">Thường</option>
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
      {/* Gift Codes section */}
      {section === "gift_codes" && (
        <div className="space-y-4">
          <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
            <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-4">
              <Gift className="w-4 h-4 text-amber-400" />
              Mã Quà Tặng
            </h3>

            {/* Create new gift code */}
            <div className="bg-zinc-950/50 border border-zinc-800/40 rounded-lg p-4 mb-4">
              <h4 className="text-xs font-bold text-zinc-300 mb-3">Tạo mã quà mới</h4>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3 mb-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mã code</label>
                  <input value={newGiftCode} onChange={e => setNewGiftCode(e.target.value.toUpperCase())} placeholder="VIP2024" className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 font-mono uppercase" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lượt tối đa</label>
                  <input type="number" min={1} value={newGiftMaxUses} onChange={e => setNewGiftMaxUses(Number(e.target.value))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tối đa / người</label>
                  <input type="number" min={1} value={newGiftPerPlayer} onChange={e => setNewGiftPerPlayer(Number(e.target.value))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Hạn (ngày)</label>
                  <input type="number" min={1} value={newGiftExpiry} onChange={e => setNewGiftExpiry(Number(e.target.value))} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                </div>
              </div>

              {/* Rewards builder */}
              <div className="bg-zinc-950/30 border border-zinc-800/30 rounded-lg p-3 mb-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Phần thưởng</label>
                {newGiftRewards.map((r, i) => (
                  <div key={i} className="flex items-center gap-2 mb-2">
                    <select value={r.type} onChange={e => { const upd = [...newGiftRewards]; upd[i].type = e.target.value; setNewGiftRewards(upd); }} className="px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-[10px] text-zinc-200 focus:outline-none">
                      <option value="primeval_stones">Nguyên Thạch</option>
                      <option value="immortal_stones">Tiên Nguyên Thạch</option>
                      <option value="human_qi">Nhân Khí</option>
                      <option value="item">Vật phẩm</option>
                      <option value="gu">Cổ Trùng</option>
                      <option value="recipe">Bí Phương</option>
                      <option value="lifespan">Thọ Nguyên</option>
                      <option value="wall_progress">Phá Vách</option>
                      <option value="dao_mark">Đạo Ấn</option>
                    </select>
                    {(r.type === "item" || r.type === "gu" || r.type === "dao_mark" || r.type === "recipe") && (
                      <div className="flex items-center gap-1">
                        <span className="text-[10px] font-mono text-zinc-400 min-w-[80px] truncate" title={r.id || "Chưa chọn"}>{r.id || "—"}</span>
                        <button onClick={() => { setGiftPickerIdx(i); ensureShopSearchLoaded(); setItemPicker({source: "gift", tab: r.type === "gu" ? "gu" : r.type === "recipe" ? "recipe" : "item", query: ""}); }} className="px-2 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded text-[10px] font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                      </div>
                    )}
                    {r.type === "gu" && (
                      <input type="number" min={0} max={9} placeholder="Rank (mặc định=0)" value={r.rank || ""} onChange={e => { const upd = [...newGiftRewards]; upd[i].rank = Number(e.target.value) || 0; setNewGiftRewards(upd); }} className="w-32 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-[10px] text-zinc-200 focus:outline-none" />
                    )}
                    <input type="number" min={1} value={r.amount} onChange={e => { const upd = [...newGiftRewards]; upd[i].amount = Number(e.target.value); setNewGiftRewards(upd); }} className="w-16 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-[10px] text-zinc-200 focus:outline-none" />
                    <input value={r.label} onChange={e => { const upd = [...newGiftRewards]; upd[i].label = e.target.value; setNewGiftRewards(upd); }} placeholder="Nhãn" className="w-24 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-[10px] text-zinc-200 focus:outline-none" />
                    <button onClick={() => setNewGiftRewards(newGiftRewards.filter((_, j) => j !== i))} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setNewGiftRewards([...newGiftRewards, { type: "primeval_stones", id: "", amount: 100, label: "Đá" }])} className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-dashed border-zinc-600 rounded-lg text-[10px] text-zinc-400 hover:text-zinc-200 transition-all cursor-pointer">
                  + Thêm phần thưởng
                </button>
              </div>

              <button onClick={handleCreateGiftCode} disabled={giftLoading} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold disabled:opacity-50 cursor-pointer">
                {giftLoading ? "Đang tạo..." : "Tạo mã quà"}
              </button>
            </div>

            {/* Gift codes list */}
            <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                    <th className="px-3 py-2.5 font-semibold">Mã</th>
                    <th className="px-3 py-2.5 font-semibold">Phần thưởng</th>
                    <th className="px-3 py-2.5 font-semibold">Lượt</th>
                    <th className="px-3 py-2.5 font-semibold">Hạn</th>
                    <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                  </tr>
                </thead>
                <tbody>
                  {giftCodes.length === 0 ? (
                    <tr><td colSpan={5} className="text-center py-10 text-zinc-600 italic">Chưa có mã quà nào.</td></tr>
                  ) : (
                    giftCodes.map((gc: ApiResponseData) => (
                      <tr key={gc.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 font-mono font-bold text-amber-400">{gc.code}</td>
                        <td className="px-3 py-2.5 text-zinc-300 max-w-[200px] truncate">{gc.rewards?.map((r: ApiResponseData) => r.label).join(", ")}</td>
                        <td className="px-3 py-2.5 text-zinc-400">{gc.used_count}/{gc.max_uses}</td>
                        <td className="px-3 py-2.5 text-[10px] text-zinc-500">{gc.expires_at ? new Date(gc.expires_at).toLocaleDateString("vi-VN") : "—"}</td>
                        <td className="px-3 py-2.5">
                          <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold ${gc.is_active ? "bg-emerald-900/40 text-emerald-400 border border-emerald-700/50" : "bg-red-900/40 text-red-400 border border-red-700/50"}`}>
                            {gc.is_active ? <Check className="w-3 h-3" /> : <X className="w-3 h-3" />}
                            {gc.is_active ? "Hoạt động" : "Vô hiệu"}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
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

          {sectsLoading && sects?.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 space-y-3">
              <RefreshCw className="w-8 h-8 text-amber-400 animate-spin" />
              <span className="text-xs text-zinc-500 font-mono">Đang tải danh sách tông môn...</span>
            </div>
          ) : sects?.length === 0 ? (
            <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
              <Building2 className="w-10 h-10 text-zinc-700 mx-auto" />
              <p className="text-sm text-zinc-400 font-medium">Không có tông môn nào.</p>
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
              <table className="w-full text-left text-xs min-w-max">
                <thead>
                  <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">ID</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Tên</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Khu vực</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Cấp</th>
                    <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Quỹ</th>
                    <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Thành viên</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Chưởng môn</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Phê duyệt</th>
                    <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Duyệt lên cấp</th>
                  </tr>
                </thead>
                <tbody>
                  {sects?.map((s: ApiResponseData) => (
                    <tr key={s.id} onClick={() => { setSelectedSectId(s.id); }} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors cursor-pointer">
                      <td className="px-3 py-2.5 text-zinc-500 font-mono whitespace-nowrap">{s.id}</td>
                      <td className="px-3 py-2.5 font-semibold text-zinc-200 whitespace-nowrap">{s.name}</td>
                      <td className="px-3 py-2.5 text-zinc-400 whitespace-nowrap">{s.region}</td>
                      <td className="px-3 py-2.5 font-bold text-zinc-300 whitespace-nowrap">{s.level}</td>
                      <td className="px-3 py-2.5 text-right font-mono text-yellow-500 whitespace-nowrap">{s.funds?.toLocaleString() || 0}</td>
                      <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">{s.member_count}</td>
                      <td className="px-3 py-2.5 text-zinc-300 whitespace-nowrap">{s.leader_name || <span className="text-zinc-600 italic">Trống</span>}</td>
                      <td className="px-3 py-2.5 whitespace-nowrap">{s.require_approval ? <span className="text-amber-400 text-[10px]">✓</span> : <span className="text-zinc-600">–</span>}</td>
                      <td className="px-3 py-2.5 whitespace-nowrap">{s.require_promotion_approval ? <span className="text-amber-400 text-[10px]">✓</span> : <span className="text-zinc-600">–</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Apertures section */}
      {section === "apertures" && (
        <>
          <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 mb-4">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-2.5" />
              <input
                value={apertureSearchDebounce}
                onChange={e => setApertureSearchDebounce(e.target.value)}
                placeholder="Tìm tiên khiếu theo tên người chơi, email, ID tiên khiếu..."
                className="w-full pl-9 pr-4 py-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500/50"
              />
            </div>
            <button onClick={fetchApertures} disabled={aperturesLoading} className="flex items-center justify-center gap-1.5 px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50 cursor-pointer">
              <RefreshCw className={`w-3.5 h-3.5 ${aperturesLoading ? "animate-spin" : ""}`} /> Làm mới
            </button>
          </div>

          {aperturesLoading && apertures.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 space-y-3">
              <RefreshCw className="w-8 h-8 text-amber-400 animate-spin" />
              <span className="text-xs text-zinc-500 font-mono">Đang tải danh sách tiên khiếu...</span>
            </div>
          ) : apertures.length === 0 ? (
            <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
              <Globe className="w-10 h-10 text-zinc-700 mx-auto" />
              <p className="text-sm text-zinc-400 font-medium">Không tìm thấy tiên khiếu nào.</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto rounded-xl border border-zinc-800/80 bg-zinc-950/20">
                <table className="w-full text-left text-xs min-w-max">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Người chơi</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Phẩm cấp</th>
                      <th className="px-3 py-2.5 font-semibold whitespace-nowrap">Trạng thái</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Tốc độ TG</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">HP Tiên Khiếu</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Diện tích</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Tiên Nguyên</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Thiên Kiếp Kế</th>
                      <th className="px-3 py-2.5 font-semibold text-right whitespace-nowrap">Số Kiếp Nạn</th>
                      <th className="px-3 py-2.5 font-semibold text-center whitespace-nowrap">Hành động</th>
                    </tr>
                  </thead>
                  <tbody>
                    {apertures.map((ap: ApiResponseData) => (
                      <tr key={ap.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 whitespace-nowrap">
                          <span className="font-semibold text-zinc-200 block">{ap.username}</span>
                          <span className="text-[10px] text-zinc-500 font-mono block max-w-[140px] truncate" title={ap.id}>{ap.id}</span>
                        </td>
                        <td className="px-3 py-2.5 font-bold text-zinc-300 whitespace-nowrap">
                          {ap.grade === "LOW" ? "Hạ Phẩm" : ap.grade === "MID" ? "Trung Phẩm" : ap.grade === "HIGH" ? "Thượng Phẩm" : ap.grade === "EXTREME" ? "Chí Tôn" : ap.grade}
                        </td>
                        <td className="px-3 py-2.5 font-semibold whitespace-nowrap">
                          <span className={`px-2 py-0.5 rounded text-[10px] uppercase font-mono whitespace-nowrap inline-block ${
                            ap.state === "STABLE" ? "bg-emerald-950/55 text-emerald-400 border border-emerald-800/30" :
                            ap.state === "TRIBULATION_PENDING" ? "bg-amber-950/55 text-amber-400 border border-amber-800/30" :
                            ap.state === "TRIBULATION_ACTIVE" ? "bg-red-950/55 text-red-400 border border-red-800/30 animate-pulse" :
                            "bg-zinc-800 text-zinc-400"
                          }`}>
                            {ap.state === "STABLE" ? "Ổn định" : ap.state === "TRIBULATION_PENDING" ? "Đợi Kiếp" : ap.state === "TRIBULATION_ACTIVE" ? "Đang Kiếp" : ap.state}
                          </span>
                        </td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">{ap.time_rate}x</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">{ap.aperture_hp?.toLocaleString()}/{ap.max_aperture_hp?.toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">{ap.area_size?.toLocaleString()} mẫu</td>
                        <td className="px-3 py-2.5 text-right font-mono text-yellow-500 font-bold whitespace-nowrap">{ap.immortal_essence?.toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">Năm {Math.round(ap.next_tribulation_year || 0)}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300 whitespace-nowrap">{ap.tribulation_count}</td>
                        <td className="px-3 py-2.5 text-center whitespace-nowrap">
                          <button onClick={() => setSelectedApertureId(ap.id)} className="px-2 py-1 bg-amber-600/20 hover:bg-amber-600/30 border border-amber-500/20 hover:border-amber-500/40 text-amber-400 rounded text-[10px] font-bold cursor-pointer transition-colors">
                            Sửa
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {aperturesTotal > aperturesLimit && (
                <div className="flex items-center justify-between mt-4 text-xs font-semibold text-zinc-500">
                  <div>Tổng số: <span className="text-zinc-300">{aperturesTotal.toLocaleString()}</span> tiên khiếu</div>
                  <div className="flex items-center gap-3">
                    <span>Trang {aperturesPage}/{Math.ceil(aperturesTotal / aperturesLimit) || 1}</span>
                    <div className="flex gap-2">
                      <button onClick={() => setAperturesPage(Math.max(1, aperturesPage - 1))} disabled={aperturesPage <= 1 || aperturesLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        <ChevronLeft className="w-4 h-4" /> Trước
                      </button>
                      <button onClick={() => setAperturesPage(aperturesPage + 1)} disabled={aperturesPage >= Math.ceil(aperturesTotal / aperturesLimit) || aperturesLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        Sau <ChevronRight className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </>
      )}

      {/* Aperture Edit Modal */}
      {selectedApertureId !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => { setSelectedApertureId(null); setApertureDetail(null); }}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-3xl mx-4 shadow-2xl max-h-[90vh] overflow-y-auto space-y-4" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
              <div>
                <h3 className="text-sm font-serif font-bold text-zinc-200">Chỉnh sửa Tiên Khiếu</h3>
                <p className="text-xs text-zinc-500 font-mono mt-0.5">Người chơi: <span className="text-amber-400 font-semibold">{apertureDetailLoading ? "Đang tải..." : apertureDetail?.player_id ? apertureDetail.player_id : ""} ({apertureDetailLoading ? "..." : (apertureForm.username || "")})</span></p>
              </div>
              <button onClick={() => { setSelectedApertureId(null); setApertureDetail(null); }} className="p-1 rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 transition-colors cursor-pointer">
                <X className="w-4 h-4 text-zinc-400" />
              </button>
            </div>

            {apertureDetailLoading ? (
              <div className="flex flex-col items-center justify-center py-20 space-y-3">
                <RefreshCw className="w-8 h-8 text-amber-400 animate-spin" />
                <span className="text-xs text-zinc-500 font-mono">Đang tải chi tiết tiên khiếu...</span>
              </div>
            ) : (
              <div className="space-y-4">
                {/* Basic Properties */}
                <div className="bg-zinc-950/40 border border-zinc-800/60 rounded-xl p-4 space-y-3">
                  <h4 className="text-xs font-bold text-zinc-300 uppercase tracking-wider">Thuộc tính cơ bản</h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Phẩm cấp</label>
                      <select value={apertureForm.grade || "LOW"} onChange={e => setApertureForm({ ...apertureForm, grade: e.target.value })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                        <option value="LOW">Hạ Phẩm Tiên Khiếu (Lục chuyển)</option>
                        <option value="MID">Trung Phẩm Tiên Khiếu (Lục chuyển)</option>
                        <option value="HIGH">Thượng Phẩm Tiên Khiếu (Lục chuyển)</option>
                        <option value="EXTREME">Chí Tôn Tiên Khiếu (Chí Tôn)</option>
                      </select>
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Trạng thái</label>
                      <select value={apertureForm.state || "STABLE"} onChange={e => setApertureForm({ ...apertureForm, state: e.target.value })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                        <option value="STABLE">Ổn định (Stable)</option>
                        <option value="TRIBULATION_PENDING">Chờ thiên kiếp (Pending)</option>
                        <option value="TRIBULATION_ACTIVE">Đang chịu kiếp (Active)</option>
                        <option value="DAMAGED">Bị tổn hại (Damaged)</option>
                        <option value="DESTROYED">Bị phá huỷ (Destroyed)</option>
                      </select>
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tốc độ thời gian (x)</label>
                      <input type="number" step="0.1" value={apertureForm.time_rate || 0} onChange={e => setApertureForm({ ...apertureForm, time_rate: parseFloat(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">HP Hiện tại</label>
                      <input type="number" value={apertureForm.aperture_hp || 0} onChange={e => setApertureForm({ ...apertureForm, aperture_hp: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">HP Tối đa</label>
                      <input type="number" value={apertureForm.max_aperture_hp || 0} onChange={e => setApertureForm({ ...apertureForm, max_aperture_hp: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Diện tích (mẫu)</label>
                      <input type="number" value={apertureForm.area_size || 0} onChange={e => setApertureForm({ ...apertureForm, area_size: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tiên Nguyên hiện tại</label>
                      <input type="number" value={apertureForm.immortal_essence || 0} onChange={e => setApertureForm({ ...apertureForm, immortal_essence: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-yellow-500 font-bold" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Sản lượng TN / năm</label>
                      <input type="number" value={apertureForm.essence_production_per_year || 0} onChange={e => setApertureForm({ ...apertureForm, essence_production_per_year: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Thời gian trôi qua (năm)</label>
                      <input type="number" step="0.001" value={apertureForm.internal_years_passed || 0} onChange={e => setApertureForm({ ...apertureForm, internal_years_passed: parseFloat(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                  </div>
                </div>

                {/* Tribulation Properties */}
                <div className="bg-zinc-950/40 border border-zinc-800/60 rounded-xl p-4 space-y-3">
                  <h4 className="text-xs font-bold text-zinc-300 uppercase tracking-wider">Thông tin Thiên Kiếp / Địa Tai</h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Năm chịu kiếp tiếp theo</label>
                      <input type="number" step="0.1" value={apertureForm.next_tribulation_year || 0} onChange={e => setApertureForm({ ...apertureForm, next_tribulation_year: parseFloat(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Năm bắt đầu cảnh giới hiện tại</label>
                      <input type="number" step="0.1" value={apertureForm.rank_start_year || 0} onChange={e => setApertureForm({ ...apertureForm, rank_start_year: parseFloat(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lần Địa Tai</label>
                      <input type="number" value={apertureForm.earth_calamity_count || 0} onChange={e => setApertureForm({ ...apertureForm, earth_calamity_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lần Thiên Kiếp</label>
                      <input type="number" value={apertureForm.heavenly_tribulation_count || 0} onChange={e => setApertureForm({ ...apertureForm, heavenly_tribulation_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lần Đại Kiếp</label>
                      <input type="number" value={apertureForm.grand_tribulation_count || 0} onChange={e => setApertureForm({ ...apertureForm, grand_tribulation_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lần Vạn Kiếp</label>
                      <input type="number" value={apertureForm.myriad_tribulation_count || 0} onChange={e => setApertureForm({ ...apertureForm, myriad_tribulation_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                    <div>
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Số lần Hỗn Độn Kiếp</label>
                      <input type="number" value={apertureForm.chaos_calamity_count || 0} onChange={e => setApertureForm({ ...apertureForm, chaos_calamity_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                    </div>
                  </div>
                </div>

                {/* Dao Marks Editor */}
                <div className="bg-zinc-950/40 border border-zinc-800/60 rounded-xl p-4 space-y-3">
                  <h4 className="text-xs font-bold text-zinc-300 uppercase tracking-wider flex items-center justify-between">
                    <span>Đạo Ngân của Tiên Khiếu</span>
                    <button type="button" onClick={() => {
                      const arr = [...(apertureForm.dao_marks_array || []), { path: "", amount: 0 }];
                      setApertureForm({ ...apertureForm, dao_marks_array: arr });
                    }} className="flex items-center gap-1 text-[10px] font-bold text-amber-500 hover:text-amber-400 bg-amber-950/40 hover:bg-amber-950/70 border border-amber-900/50 px-2 py-1 rounded cursor-pointer transition-all">
                      <Plus className="w-3 h-3" /> Thêm Đạo Ngân
                    </button>
                  </h4>
                  <div className="space-y-2 max-h-[200px] overflow-y-auto pr-1">
                    {!apertureForm.dao_marks_array || apertureForm.dao_marks_array.length === 0 ? (
                      <p className="text-xs text-zinc-500 italic">Chưa có đạo ngân nào trong tiên khiếu.</p>
                    ) : (
                      apertureForm.dao_marks_array.map((entry: ApiResponseData, index: number) => (
                        <div key={index} className="flex items-center gap-2">
                          <input
                            value={entry.path}
                            onChange={e => {
                              const arr = [...apertureForm.dao_marks_array];
                              arr[index].path = e.target.value;
                              setApertureForm({ ...apertureForm, dao_marks_array: arr });
                            }}
                            placeholder="Hệ Đạo ngân (ví dụ: strength, fire, refinement...)"
                            className="flex-1 px-3 py-1.5 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none"
                          />
                          <input
                            type="number"
                            value={entry.amount}
                            onChange={e => {
                              const arr = [...apertureForm.dao_marks_array];
                              arr[index].amount = parseInt(e.target.value) || 0;
                              setApertureForm({ ...apertureForm, dao_marks_array: arr });
                            }}
                            className="w-32 px-3 py-1.5 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200 text-right focus:outline-none font-mono"
                          />
                          <button
                            type="button"
                            onClick={() => {
                              const arr = apertureForm.dao_marks_array.filter((_: ApiResponseData, idx: number) => idx !== index);
                              setApertureForm({ ...apertureForm, dao_marks_array: arr });
                            }}
                            className="p-1.5 text-red-500 hover:text-red-400 hover:bg-red-950/20 border border-transparent hover:border-red-900/30 rounded-lg transition-colors cursor-pointer"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {selectedApertureId && <div className="border-t border-zinc-800/80 pt-4 space-y-4"><ApertureTribulationsAdmin apertureId={selectedApertureId} addToast={addToast} /><ApertureChildrenAdmin apertureId={selectedApertureId} ownerUsername={String(apertures.find(ap => ap.id === selectedApertureId)?.username || "")} canEdit={canAdminAction("updateApertureChild")} canDelete={canAdminAction("deleteApertureChild")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("deleteApertureChild", label, target)} /><ApertureRefinementQueueAdmin apertureId={selectedApertureId} canEdit={canAdminAction("updateApertureChild")} canDelete={canAdminAction("deleteApertureChild")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("deleteApertureChild", label, target)} /><ApertureSettlementsAdmin apertureId={selectedApertureId} canEdit={canAdminAction("updateApertureChild")} canDelete={canAdminAction("deleteApertureChild")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("deleteApertureChild", label, target)} /><ApertureEventsAdmin apertureId={selectedApertureId} canEdit={canAdminAction("updateApertureChild")} canDelete={canAdminAction("deleteApertureChild")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("deleteApertureChild", label, target)} /><ApertureNPCAdmin apertureId={selectedApertureId} canEdit={canAdminAction("updateVariantNPC")} canDelete={canAdminAction("dismissVariantNPC")} addToast={addToast} exec={exec} prepareDelete={(label, target) => prepareAdminAction("dismissVariantNPC", label, target)} /></div>}

                {/* Actions */}
                <div className="flex items-center justify-end gap-2 border-t border-zinc-800/80 pt-4">
                  <button type="button" onClick={() => { setSelectedApertureId(null); setApertureDetail(null); }} className="px-4 py-2 bg-zinc-850 border border-zinc-700 hover:bg-zinc-800 text-zinc-300 rounded-lg text-xs font-semibold cursor-pointer transition-colors">
                    Hủy
                  </button>
                  <button type="button" onClick={saveApertureDetail} disabled={apertureSaving} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold cursor-pointer transition-colors flex items-center gap-1.5">
                    {apertureSaving ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Lưu thay đổi
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Gu Edit Modal */}
      {guEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setGuEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-6xl mx-4 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{guEdit.id ? "Sửa" : "Thêm"} Cổ Trùng</h3>
            <div className="grid grid-cols-2 gap-3 mb-4">
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={guEdit.id || ""} onChange={e => setGuEdit({...guEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên</label><input value={guEdit.name || ""} onChange={e => setGuEdit({...guEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Bậc</label><input type="number" min={1} max={10} value={guEdit.rank} onChange={e => setGuEdit({...guEdit, rank: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại</label><select value={guEdit.gu_type} onChange={e => setGuEdit({...guEdit, gu_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"><option value="Active">Chủ động</option><option value="Buff">Cường hoá</option><option value="Consumable">Tiêu hao</option><option value="Control">Khống chế</option><option value="Defense">Phòng thủ</option><option value="Healing">Hồi phục</option><option value="Investigation">Trinh sát</option><option value="Movement">Di chuyển</option><option value="Passive">Bị động</option><option value="Refinement">Luyện cổ</option><option value="Special">Đặc thù</option><option value="Storage">Lưu trữ</option><option value="Summon">Triệu hồi</option><option value="Support">Hỗ trợ</option><option value="Transformation">Biến hoá</option><option value="Vital">Bản mệnh</option></select></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tốc độ đói</label><input type="number" min={1} value={guEdit.hunger_rate} onChange={e => setGuEdit({...guEdit, hunger_rate: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Độ no tối đa</label><input type="number" value={guEdit.max_hunger} onChange={e => setGuEdit({...guEdit, max_hunger: parseInt(e.target.value) || 100})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tiêu hao CN cơ bản</label><input type="number" step="0.1" value={guEdit.base_mana_cost} onChange={e => setGuEdit({...guEdit, base_mana_cost: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá cơ bản</label><input type="number" value={guEdit.base_price} onChange={e => setGuEdit({...guEdit, base_price: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mục tiêu</label><select value={guEdit.target_type || "SINGLE"} onChange={e => setGuEdit({...guEdit, target_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"><option value="SINGLE">Đơn mục tiêu</option><option value="ALL">Toàn thể</option></select></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Độ hiếm (1-9)</label><input type="number" min={0} max={9} value={guEdit.rarity || 0} onChange={e => setGuEdit({...guEdit, rarity: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Vai trò chính</label><input value={guEdit.primary_role || ""} onChange={e => setGuEdit({...guEdit, primary_role: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
            </div>
            <div className="mb-4"><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mô tả</label><textarea value={guEdit.description || ""} onChange={e => setGuEdit({...guEdit, description: e.target.value})} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
            <div className="mb-4">
              <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Ảnh</label>
              <div className="flex items-center gap-3">
                {guEdit.image_url ? (
                  <img src={guEdit.image_url} alt="" className="w-14 h-14 rounded-lg object-cover border border-zinc-700" />
                ) : (
                  <div className="w-14 h-14 rounded-lg bg-zinc-800 border border-zinc-700 flex items-center justify-center text-zinc-600 text-lg">{({Active: "⚡", Buff: "✨", Consumable: "💊", Control: "🌀", Defense: "🛡️", Healing: "💚", Investigation: "🔍", Movement: "💨", Passive: "🧘", Refinement: "🔧", Special: "⭐", Storage: "📦", Summon: "👻", Support: "🎯", Transformation: "🦋", Vital: "💎"} as Record<string, string>)[guEdit.gu_type] || "🏠"}</div>
                )}
                <div className="flex-1">
                  <input value={guEdit.image_url || ""} onChange={e => setGuEdit({...guEdit, image_url: e.target.value})} placeholder="URL ảnh hoặc để trống và upload..." className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 mb-1.5" />
                  <label className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-[10px] font-semibold cursor-pointer">
                    <Upload className="w-3 h-3" />
                    {uploading ? "Đang tải..." : "Upload ảnh"}
                    <input type="file" accept="image/jpeg,image/png,image/webp" className="hidden" disabled={uploading} onChange={async (e) => {
                      const file = e.target.files?.[0];
                      if (!file || !guEdit.id) return;
                      setUploading(true);
                      try {
                        const fd = new FormData();
                        fd.append("file", file);
                        fd.append("entity_type", "gu");
                        fd.append("entity_id", guEdit.id);
                        const res = await authFetch(`${BACKEND_URL}/api/admin/upload-image`, { method: "POST", body: fd });
                        const data = await res.json();
                        if (!res.ok) throw new Error(data.error || "Upload thất bại");
                        setGuEdit({...guEdit, image_url: data.image_url});
                        if (addToast) addToast(data.message || "Tải ảnh lên thành công!", "success");
                      } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
                        if (addToast) addToast(err.message, "error");
                      } finally { setUploading(false); e.target.value = ""; }
                    }} />
                  </label>
                </div>
              </div>
            </div>
            {/* JSON arrays as comma input */}
            {[
              { key: "path", label: "Path (cách nhau bằng dấu phẩy)" },
              { key: "food_types", label: "Food Types (cách nhau bằng dấu phẩy)" },
              { key: "combat_roles", label: "Combat Roles" },
              { key: "combat_tags", label: "Combat Tags" },
              { key: "support_tags", label: "Support Tags" },
            ].map(f => (
              <div key={f.key} className="mb-2">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">{f.label}</label>
                <input value={(Array.isArray(guEdit[f.key]) ? guEdit[f.key] : []).join(", ")} onChange={e => setGuEdit({...guEdit, [f.key]: e.target.value.split(",").map((s: string) => s.trim()).filter(Boolean)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
              </div>
            ))}
            {/* Gu Effects - Composable Blocks */}
            <GuEffectsEditor
              effects={guEdit.effects || {}}
              source="gu"
              effectKey={`${guEdit.id || "new"}:${guEdit.rank || 1}`}
              onChange={(newEffects) => setGuEdit({ ...guEdit, effects: newEffects })}
            />
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setGuEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                if (!await validateAllEffectEditors()) { addToast?.("Effect Block chưa hợp lệ", "error"); return; }
                const isNew = !!guEdit._isNew;
                const { _isNew, ...body } = guEdit;
                const url = isNew ? `${BACKEND_URL}/api/admin/gu-dictionary` : `${BACKEND_URL}/api/admin/gu-dictionary/${guEdit.id}?old_rank=${guEdit._old_rank || body.rank}`;
                const method = isNew ? "POST" : "PUT";
                const ok = await exec(url, method, body);
                if (ok) { setGuEdit(null); fetchGus(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}

      {/* Item Edit Modal */}
      {itemEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setItemEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{itemEdit.id ? "Sửa" : "Thêm"} Vật Phẩm</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={itemEdit.id || ""} disabled={!itemEdit._isNew} onChange={e => setItemEdit({...itemEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-60" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên</label><input value={itemEdit.name || ""} onChange={e => setItemEdit({...itemEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại</label><select value={itemEdit.item_type} onChange={e => setItemEdit({...itemEdit, item_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"><option value="Material">Vật liệu</option><option value="Consumable">Tiêu hao</option><option value="Food">Thức ăn</option><option value="Currency">Tiền tệ</option></select></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Bậc</label><input type="number" min={1} value={itemEdit.rank} onChange={e => setItemEdit({...itemEdit, rank: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá cơ bản</label><input type="number" min={0} step="0.001" value={itemEdit.base_price} onChange={e => setItemEdit({...itemEdit, base_price: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Độ hiếm</label><input type="number" min={0} max={9} value={itemEdit.rarity ?? 0} onChange={e => setItemEdit({...itemEdit, rarity: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Lưu Phái</label><input maxLength={50} value={itemEdit.path || ""} onChange={e => setItemEdit({...itemEdit, path: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Ảnh</label>
                <div className="flex items-center gap-3">
                  {itemEdit.image_url ? (
                    <img src={itemEdit.image_url} alt="" className="w-14 h-14 rounded-lg object-cover border border-zinc-700" />
                  ) : (
                    <div className="w-14 h-14 rounded-lg bg-zinc-800 border border-zinc-700 flex items-center justify-center text-zinc-600 text-lg"><Package className="w-6 h-6" /></div>
                  )}
                  <div className="flex-1">
                    <input value={itemEdit.image_url || ""} onChange={e => setItemEdit({...itemEdit, image_url: e.target.value})} placeholder="URL ảnh hoặc upload..." className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 mb-1.5" />
                    <label className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-[10px] font-semibold cursor-pointer">
                      <Upload className="w-3 h-3" />
                      {uploading ? "Đang tải..." : "Upload ảnh"}
                      <input type="file" accept="image/jpeg,image/png,image/webp" className="hidden" disabled={uploading} onChange={async (e) => {
                        const file = e.target.files?.[0];
                        if (!file || !itemEdit.id) return;
                        setUploading(true);
                        try {
                          const fd = new FormData();
                          fd.append("file", file);
                          fd.append("entity_type", "item");
                          fd.append("entity_id", itemEdit.id);
                          const res = await authFetch(`${BACKEND_URL}/api/admin/upload-image`, { method: "POST", body: fd });
                          const data = await res.json();
                          if (!res.ok) throw new Error(data.error || "Upload thất bại");
                          setItemEdit({...itemEdit, image_url: data.image_url});
                          if (addToast) addToast(data.message || "Tải ảnh lên thành công!", "success");
                        } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
                          if (addToast) addToast(err.message, "error");
                        } finally { setUploading(false); e.target.value = ""; }
                      }} />
                    </label>
                  </div>
                </div>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mô tả</label><textarea value={itemEdit.description || ""} onChange={e => setItemEdit({...itemEdit, description: e.target.value})} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <GuEffectsEditor effects={itemEdit.effects || []} source="item" effectKey={itemEdit.id || "new"} onChange={(effects) => setItemEdit({ ...itemEdit, effects })} />
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setItemEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                if (!await validateAllEffectEditors()) { addToast?.("Effect Block chưa hợp lệ", "error"); return; }
                const isNew = !!itemEdit._isNew;
                const itemUrl = isNew ? `${BACKEND_URL}/api/admin/item-dictionary` : `${BACKEND_URL}/api/admin/item-dictionary/${itemEdit.id}`;
                const clean = {...itemEdit}; delete clean._isNew;
                const ok = await exec(itemUrl, isNew ? "POST" : "PUT", clean);
                if (ok) { setItemEdit(null); fetchItems(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}

      {/* Recipe Edit Modal */}
      {recipeEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setRecipeEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{recipeEdit.id ? "Sửa" : "Thêm"} Cổ Phương</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={recipeEdit.id || ""} disabled={!recipeEdit._isNew} onChange={e => setRecipeEdit({...recipeEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-60" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Cổ trùng đích</label>
                  <select value={recipeEdit.target_gu_id} disabled={!recipeEdit._isNew} onChange={e => { const target = gus.find((g: ApiResponseData) => g.id === e.target.value); setRecipeEdit({...recipeEdit, target_gu_id: e.target.value, target_rank: target?.rank || 1}); }} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-60">
                    <option value="">Chọn...</option>
                    {gus.map((g: ApiResponseData) => <option key={g.id} value={g.id}>{g.name} ({g.id})</option>)}
                  </select>
                </div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Bậc cổ trùng đích</label><input type="number" min={1} disabled={!recipeEdit._isNew} value={recipeEdit.target_rank || 1} onChange={e => setRecipeEdit({...recipeEdit, target_rank: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-60" /></div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tỉ lệ thành công</label><input type="number" step="0.01" min={0} max={1} value={recipeEdit.success_rate} onChange={e => setRecipeEdit({...recipeEdit, success_rate: Number(e.target.value)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Nguyên Thạch</label><input type="number" min={0} step={0.001} value={recipeEdit.primeval_stones_cost} onChange={e => setRecipeEdit({...recipeEdit, primeval_stones_cost: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tiên nguyên thạch</label><input type="number" step="any" value={recipeEdit.immortal_essence_stones_cost} onChange={e => setRecipeEdit({...recipeEdit, immortal_essence_stones_cost: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              {/* Required Gus as key-value */}
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Cổ trùng yêu cầu (required_gus): ID → {`{rank, qty}`} (rank 0 = bất kỳ bậc)</label>
                {Object.entries(recipeEdit.required_gus || {}).map(([k, v]) => (
                  <div key={k} className="flex gap-2 mb-1.5">
                    <input value={k} onChange={e => { const newG = {...recipeEdit.required_gus}; const val = newG[k]; delete newG[k]; newG[e.target.value] = val; setRecipeEdit({...recipeEdit, required_gus: newG}) }} className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="gu-list" />
                    <input type="number" min={0} value={reqGuRank(v as number | RequiredGuItem)} onChange={e => setRecipeEdit({...recipeEdit, required_gus: {...recipeEdit.required_gus, [k]: { rank: parseInt(e.target.value) || 0, qty: reqGuQty(recipeEdit.required_gus[k] as number | RequiredGuItem) }}})} className="w-20 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" title="Bậc yêu cầu (0 = bất kỳ)" />
                    <input type="number" min={1} value={reqGuQty(v as number | RequiredGuItem)} onChange={e => setRecipeEdit({...recipeEdit, required_gus: {...recipeEdit.required_gus, [k]: { rank: reqGuRank(recipeEdit.required_gus[k] as number | RequiredGuItem), qty: parseInt(e.target.value) || 1 }}})} className="w-20 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" title="Số lượng" />
                    <button onClick={() => { const newG = {...recipeEdit.required_gus}; delete newG[k]; setRecipeEdit({...recipeEdit, required_gus: newG}) }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setRecipeEdit({...recipeEdit, required_gus: {...recipeEdit.required_gus, "": { rank: 0, qty: 1 }}})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm Gu</button>
              </div>
              {/* Required Materials as key-value */}
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Nguyên liệu yêu cầu (required_materials): ID → số lượng</label>
                {Object.entries(recipeEdit.required_materials || {}).map(([k, v]) => (
                  <div key={k} className="flex gap-2 mb-1.5">
                    <input value={k} onChange={e => { const newM = {...recipeEdit.required_materials}; const val = newM[k]; delete newM[k]; newM[e.target.value] = val; setRecipeEdit({...recipeEdit, required_materials: newM}) }} className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="item-list" />
                    <input type="number" min={1} value={v as number} onChange={e => setRecipeEdit({...recipeEdit, required_materials: {...recipeEdit.required_materials, [k]: parseInt(e.target.value) || 1}})} className="w-20 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <button onClick={() => { const newM = {...recipeEdit.required_materials}; delete newM[k]; setRecipeEdit({...recipeEdit, required_materials: newM}) }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setRecipeEdit({...recipeEdit, required_materials: {...recipeEdit.required_materials, "": 1}})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm nguyên liệu</button>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setRecipeEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!recipeEdit._isNew;
                const recipeUrl = isNew ? `${BACKEND_URL}/api/admin/recipes` : `${BACKEND_URL}/api/admin/recipes/${recipeEdit.id}`;
                const { _isNew, ...cleanRecipe } = recipeEdit;
                const clean = {
                  ...cleanRecipe,
                  required_gus: Object.fromEntries(Object.entries(recipeEdit.required_gus || {}).filter(([k]) => k.trim() !== "")),
                  required_materials: Object.fromEntries(Object.entries(recipeEdit.required_materials || {}).filter(([k]) => k.trim() !== "")),
                };
                const ok = await exec(recipeUrl, isNew ? "POST" : "PUT", clean);
                if (ok) { setRecipeEdit(null); fetchRecipes(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
              <datalist id="gu-list">{gus.map((g: ApiResponseData) => <option key={g.id} value={g.id} />)}</datalist>
              <datalist id="item-list">{items.map((it: ApiResponseData) => <option key={it.id} value={it.id} />)}</datalist>
            </div>
          </div>
        </div>
      )}

      {/* Monster Edit Modal */}
      {monsterEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setMonsterEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{monsterEdit._isNew ? "Thêm" : "Sửa"} Quái Vật</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={monsterEdit.id || ""} onChange={e => setMonsterEdit({...monsterEdit, id: e.target.value})} disabled={!monsterEdit._isNew} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-50" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên</label><input value={monsterEdit.name || ""} onChange={e => setMonsterEdit({...monsterEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại</label>
                  <select value={monsterEdit.type || "beast"} onChange={e => setMonsterEdit({...monsterEdit, type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                    {["beast", "humanoid", "spirit", "undead", "elemental", "demon", "mechanical", "plant"].map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">AI Profile</label>
                  <select value={monsterEdit.ai_profile || "beast_basic"} onChange={e => setMonsterEdit({...monsterEdit, ai_profile: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                    {["beast_basic", "humanoid_basic", "boss_basic", "passive"].map(p => <option key={p} value={p}>{p}</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">HP</label><input type="number" min={1} value={monsterEdit.max_hp || 100} onChange={e => setMonsterEdit({...monsterEdit, max_hp: parseFloat(e.target.value) || 100})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tấn công</label><input type="number" min={0} value={monsterEdit.base_attack || 0} onChange={e => setMonsterEdit({...monsterEdit, base_attack: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Phòng thủ</label><input type="number" min={0} value={monsterEdit.base_defense || 0} onChange={e => setMonsterEdit({...monsterEdit, base_defense: parseFloat(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loot Table ID</label><input value={monsterEdit.loot_table_id || ""} onChange={e => setMonsterEdit({...monsterEdit, loot_table_id: e.target.value})} placeholder="VD: loot_beast_small" className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" list="loot-table-list" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mô tả</label><input value={monsterEdit.description || ""} onChange={e => setMonsterEdit({...monsterEdit, description: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div className="flex items-center gap-2 pt-5"><input type="checkbox" checked={monsterEdit.is_active !== false} onChange={e => setMonsterEdit({...monsterEdit, is_active: e.target.checked})} className="rounded" /><label className="text-xs text-zinc-300">Hoạt động</label></div>
                <div className="flex items-center gap-2 pt-5"><input type="checkbox" checked={monsterEdit.is_capturable === true} onChange={e => setMonsterEdit({...monsterEdit, is_capturable: e.target.checked})} className="rounded" /><label className="text-xs text-zinc-300">Bắt được</label></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tỉ lệ bắt</label><input type="number" step="0.01" min={0} max={1} value={monsterEdit.capture_chance ?? 0.20} onChange={e => setMonsterEdit({...monsterEdit, capture_chance: Number(e.target.value)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Đói/giờ (pet)</label><input type="number" step="0.1" min={0} value={monsterEdit.pet_hunger_per_hour ?? 2.0} onChange={e => setMonsterEdit({...monsterEdit, pet_hunger_per_hour: Number(e.target.value)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              {/* JSONB fields as raw JSON */}
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">gu_inventory (JSON)</label>
                <textarea value={typeof monsterEdit.gu_inventory === "string" ? monsterEdit.gu_inventory : JSON.stringify(monsterEdit.gu_inventory || [], null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, gu_inventory: JSON.parse(e.target.value)}); } catch {} }} rows={3} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">gu_pool (JSON)</label><textarea value={typeof monsterEdit.gu_pool === "string" ? monsterEdit.gu_pool : JSON.stringify(monsterEdit.gu_pool || [], null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, gu_pool: JSON.parse(e.target.value)}); } catch {} }} rows={3} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" /></div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">item_inventory (JSON)</label>
                <textarea value={typeof monsterEdit.item_inventory === "string" ? monsterEdit.item_inventory : JSON.stringify(monsterEdit.item_inventory || {}, null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, item_inventory: JSON.parse(e.target.value)}); } catch {} }} rows={3} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">dao_marks (JSON)</label>
                <textarea value={typeof monsterEdit.dao_marks === "string" ? monsterEdit.dao_marks : JSON.stringify(monsterEdit.dao_marks || {}, null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, dao_marks: JSON.parse(e.target.value)}); } catch {} }} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">path_masteries (JSON)</label>
                <textarea value={typeof monsterEdit.path_masteries === "string" ? monsterEdit.path_masteries : JSON.stringify(monsterEdit.path_masteries || {}, null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, path_masteries: JSON.parse(e.target.value)}); } catch {} }} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setMonsterEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!monsterEdit._isNew;
                const url = isNew ? `${BACKEND_URL}/api/admin/monsters` : `${BACKEND_URL}/api/admin/monsters/${monsterEdit.id}`;
                const { _isNew, ...clean } = monsterEdit;
                const ok = await exec(url, isNew ? "POST" : "PUT", clean);
                if (ok) { setMonsterEdit(null); fetchMonsters(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
              <datalist id="loot-table-list">{lootTables.map((lt: ApiResponseData) => <option key={lt.id} value={lt.id} />)}</datalist>
            </div>
          </div>
        </div>
      )}

      {/* Loot Table Edit Modal */}
      {lootTableEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setLootTableEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{lootTableEdit._isNew ? "Thêm" : "Sửa"} Bảng Loot</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={lootTableEdit.id || ""} onChange={e => setLootTableEdit({...lootTableEdit, id: e.target.value})} disabled={!lootTableEdit._isNew} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-50" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên</label><input value={lootTableEdit.name || ""} onChange={e => setLootTableEdit({...lootTableEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Danh sách loot</label>
                {(Array.isArray(lootTableEdit.entries) ? lootTableEdit.entries : []).map((entry: ApiResponseData, idx: number) => (
                  <div key={idx} className="flex gap-2 mb-2 items-center">
                    <input value={entry.item_id || ""} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], item_id: e.target.value}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="item_id" className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="item-list-lt" />
                    <input value={entry.gu_id || ""} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], gu_id: e.target.value}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="gu_id" className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="gu-list-lt" />
                    <input type="number" min={1} value={entry.min_qty || 1} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], min_qty: parseInt(e.target.value) || 1}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="min" className="w-14 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <input type="number" min={1} value={entry.max_qty || 1} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], max_qty: parseInt(e.target.value) || 1}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="max" className="w-14 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <input type="number" step="0.01" min={0} max={1} value={entry.chance ?? 0.5} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], chance: Number(e.target.value)}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="%" className="w-16 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <input type="number" min={0} max={9} value={entry.rank ?? 0} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], rank: parseInt(e.target.value) || 0}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="rank" title="Rank (0 = phổ quát/đặc thù)" className="w-12 px-1.5 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <label title="Jackpot: luôn rơi dù lệch rank" className="flex items-center gap-1 text-[10px] text-zinc-500 cursor-pointer"><input type="checkbox" checked={!!entry.jackpot} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], jackpot: e.target.checked}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} className="accent-amber-500" />Jackpot</label>
                    <button onClick={() => { const newEntries = lootTableEdit.entries.filter((_: ApiResponseData, i: number) => i !== idx); setLootTableEdit({...lootTableEdit, entries: newEntries}); }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setLootTableEdit({...lootTableEdit, entries: [...(lootTableEdit.entries || []), { item_id: "", gu_id: "", min_qty: 1, max_qty: 1, chance: 0.5, rank: 0, jackpot: false }]})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm loot entry</button>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setLootTableEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!lootTableEdit._isNew;
                const url = isNew ? `${BACKEND_URL}/api/admin/loot-tables` : `${BACKEND_URL}/api/admin/loot-tables/${lootTableEdit.id}`;
                const { _isNew, ...clean } = lootTableEdit;
                const ok = await exec(url, isNew ? "POST" : "PUT", clean);
                if (ok) { setLootTableEdit(null); fetchLootTables(); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
              <datalist id="item-list-lt">{items.map((it: ApiResponseData) => <option key={it.id} value={it.id} />)}</datalist>
              <datalist id="gu-list-lt">{gus.map((g: ApiResponseData) => <option key={g.id} value={g.id} />)}</datalist>
            </div>
          </div>
        </div>
      )}
      {/* Region Edit Modal */}
      {regionEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 overflow-y-auto" onClick={() => setRegionEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-2xl mx-4 my-8 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{regionEdit._isNew ? "Thêm" : "Sửa"} Khu Vực</h3>
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID khu vực</label>
                  <input disabled={!regionEdit._isNew} value={regionEdit.id || ""} onChange={e => setRegionEdit({...regionEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 disabled:opacity-50" placeholder="e.g. hub_nam_cuong" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên khu vực</label>
                  <input value={regionEdit.name || ""} onChange={e => setRegionEdit({...regionEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" placeholder="e.g. Nam Cương" />
                </div>
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mô tả</label>
                <textarea value={regionEdit.description || ""} onChange={e => setRegionEdit({...regionEdit, description: e.target.value})} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tầng bản đồ</label>
                <select value={regionEdit.world_map_id || "world_map"} onChange={e => setRegionEdit({...regionEdit, world_map_id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                  <option value="world_map">WorldMap (5 Đại Vực hiện tại)</option>
                  <option value="bai_thien" disabled>Bạch Thiên (đang khóa)</option>
                  <option value="hac_thien" disabled>Hắc Thiên (đang khóa)</option>
                </select>
                <p className="mt-1 text-[10px] text-amber-300/80">Gán tầng chỉ là cấu hình dữ liệu. Gameplay tầng 2/3 chưa mở cho đến khi vị trí, hiện diện và combat được phân tầng.</p>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Width</label>
                  <input type="number" value={regionEdit.width} onChange={e => setRegionEdit({...regionEdit, width: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Height</label>
                  <input type="number" value={regionEdit.height} onChange={e => setRegionEdit({...regionEdit, height: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Min Rank</label>
                  <input type="number" value={regionEdit.min_rank} onChange={e => setRegionEdit({...regionEdit, min_rank: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Max Rank</label>
                  <input type="number" value={regionEdit.max_rank} onChange={e => setRegionEdit({...regionEdit, max_rank: parseInt(e.target.value) || 5})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Safe Zone X (để trống nếu không có)</label><input type="number" value={regionEdit.safe_zone_x ?? ""} onChange={e => setRegionEdit({...regionEdit, safe_zone_x: e.target.value === "" ? null : parseInt(e.target.value)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Safe Zone Y (để trống nếu không có)</label><input type="number" value={regionEdit.safe_zone_y ?? ""} onChange={e => setRegionEdit({...regionEdit, safe_zone_y: e.target.value === "" ? null : parseInt(e.target.value)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Min X</label>
                  <input type="number" value={regionEdit.min_x} onChange={e => setRegionEdit({...regionEdit, min_x: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Min Y</label>
                  <input type="number" value={regionEdit.min_y} onChange={e => setRegionEdit({...regionEdit, min_y: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Max X</label>
                  <input type="number" value={regionEdit.max_x} onChange={e => setRegionEdit({...regionEdit, max_x: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Max Y</label>
                  <input type="number" value={regionEdit.max_y} onChange={e => setRegionEdit({...regionEdit, max_y: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Entry X</label>
                  <input type="number" value={regionEdit.entry_x} onChange={e => setRegionEdit({...regionEdit, entry_x: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Entry Y</label>
                  <input type="number" value={regionEdit.entry_y} onChange={e => setRegionEdit({...regionEdit, entry_y: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại Zone</label>
                  <select value={regionEdit.zone_type} onChange={e => setRegionEdit({...regionEdit, zone_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                    <option value="HUB">HUB</option>
                    <option value="SAFE_ZONE">SAFE_ZONE</option>
                    <option value="BOSS_ZONE">BOSS_ZONE</option>
                    <option value="SECRET_REALM">SECRET_REALM</option>
                    <option value="DUNGEON">DUNGEON</option>
                    <option value="PVP_ZONE">PVP_ZONE</option>
                    <option value="NGU_VUC">NGU_VUC</option>
                    <option value="KHU_VUC">KHU_VUC</option>
                  </select>
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Parent ID</label>
                  <input value={regionEdit.parent_id || ""} onChange={e => setRegionEdit({...regionEdit, parent_id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" placeholder="e.g. central_continent" />
                </div>
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Pool weights (JSON)</label>
                <textarea value={typeof regionEdit.pool_weights === "string" ? regionEdit.pool_weights : JSON.stringify(regionEdit.pool_weights || {}, null, 2)} onChange={e => { try { setRegionEdit({...regionEdit, pool_weights: JSON.parse(e.target.value)}); } catch {} }} rows={5} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Cấm chế khu vực (JSON)</label>
                <textarea value={regionRulesDraft} onChange={e => setRegionRulesDraft(e.target.value)} rows={9} spellCheck={false} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
                <p className="mt-1 text-[10px] text-zinc-500">Nhóm không khai báo kế thừa từ khu cha; khai báo nhóm ở đây sẽ thay cả nhóm cha. Dùng nhóm rỗng để xóa luật kế thừa.</p>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Đạo mạch khu vực (paths, cách nhau bằng dấu phẩy)</label><input value={(Array.isArray(regionEdit.paths) ? regionEdit.paths : []).join(", ")} onChange={e => setRegionEdit({...regionEdit, paths: e.target.value.split(",").map((path: string) => path.trim()).filter(Boolean)})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div className="flex items-center gap-2 mt-4">
                  <input type="checkbox" checked={!!regionEdit.is_active} onChange={e => setRegionEdit({...regionEdit, is_active: e.target.checked})} className="w-4 h-4 bg-zinc-850 border-zinc-700 rounded" />
                  <label className="text-xs font-semibold text-zinc-300">Hoạt động (is_active)</label>
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setRegionEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!regionEdit._isNew;
                const { _isNew, ...body } = regionEdit;
                try {
                  body.rules = JSON.parse(regionRulesDraft);
                  if (!body.rules || Array.isArray(body.rules) || typeof body.rules !== "object") throw new Error("Luật phải là JSON object.");
                } catch {
                  addToast("Cấm chế phải là một JSON object hợp lệ", "error");
                  return;
                }
                if (!body.pool_weights) {
                  body.pool_weights = { cell_type_weights: {}, resources: [], wild_gus: [], monster_gus: [] };
                }
                if (!Array.isArray(body.paths)) body.paths = [];
                const url = isNew ? `${BACKEND_URL}/api/admin/regions` : `${BACKEND_URL}/api/admin/regions/${regionEdit.id}`;
                const method = isNew ? "POST" : "PUT";
                const ok = await exec(url, method, body);
                if (ok) { setRegionEdit(null); fetchRegions(regionsPage, regionSearch); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}

      {/* MapCell Edit Modal */}
      {mapCellEdit !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 overflow-y-auto" onClick={() => setMapCellEdit(null)}>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-serif font-bold text-zinc-200 mb-4">{mapCellEdit._isNew ? "Thêm" : "Sửa"} Ô Bản Đồ</h3>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Region ID</label>
                  <input value={mapCellEdit.region_id || ""} onChange={e => setMapCellEdit({...mapCellEdit, region_id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" placeholder="e.g. hub_nam_cuong" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại Ô</label>
                  <select value={mapCellEdit.cell_type} onChange={e => setMapCellEdit({...mapCellEdit, cell_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                    <option value="Resource">Resource</option>
                    <option value="Monster">Monster</option>
                    <option value="Safe">Safe</option>
                    <option value="Portal">Portal</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tọa độ X</label>
                  <input type="number" value={mapCellEdit.x} onChange={e => setMapCellEdit({...mapCellEdit, x: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tọa độ Y</label>
                  <input type="number" value={mapCellEdit.y} onChange={e => setMapCellEdit({...mapCellEdit, y: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Content ID</label>
                  <input value={mapCellEdit.content_id || ""} onChange={e => setMapCellEdit({...mapCellEdit, content_id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" placeholder="e.g. boss_beast" />
                </div>
                <div>
                  <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Hồi chiêu (seconds)</label>
                  <input type="number" value={mapCellEdit.respawn_seconds} onChange={e => setMapCellEdit({...mapCellEdit, respawn_seconds: parseInt(e.target.value) || 300})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
                </div>
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Metadata (JSON string)</label>
                <textarea
                  value={typeof mapCellEdit.metadata === 'object' ? JSON.stringify(mapCellEdit.metadata) : mapCellEdit.metadata || ""}
                  onChange={e => {
                    try {
                      const parsed = JSON.parse(e.target.value);
                      setMapCellEdit({...mapCellEdit, metadata: parsed});
                    } catch {
                      setMapCellEdit({...mapCellEdit, metadata: e.target.value});
                    }
                  }}
                  rows={3}
                  placeholder='{"rank": 5}'
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs font-mono text-zinc-200"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Lần xóa gần nhất</label><input type="datetime-local" value={mapCellEdit.last_cleared_at ? new Date(String(mapCellEdit.last_cleared_at)).toISOString().slice(0, 16) : ""} onChange={e => setMapCellEdit({...mapCellEdit, last_cleared_at: e.target.value || null})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Cleared by (Player UUID)</label><input value={mapCellEdit.cleared_by || ""} onChange={e => setMapCellEdit({...mapCellEdit, cleared_by: e.target.value || null})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div className="flex items-center gap-2 mt-3">
                <input type="checkbox" checked={!!mapCellEdit.is_active} onChange={e => setMapCellEdit({...mapCellEdit, is_active: e.target.checked})} className="w-4 h-4 bg-zinc-850 border-zinc-700 rounded" />
                <label className="text-xs font-semibold text-zinc-300">Hoạt động (is_active)</label>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setMapCellEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!mapCellEdit._isNew;
                const { _isNew, ...body } = mapCellEdit;
                if (typeof body.metadata === 'string') {
                  try {
                    body.metadata = JSON.parse(body.metadata);
                  } catch {
                    addToast("Metadata không phải JSON hợp lệ", "error");
                    return;
                  }
                }
                if (typeof body.last_cleared_at === "string" && body.last_cleared_at) body.last_cleared_at = new Date(body.last_cleared_at).toISOString();
                else body.last_cleared_at = null;
                if (typeof body.cleared_by === "string" && !body.cleared_by.trim()) body.cleared_by = null;
                const url = isNew ? `${BACKEND_URL}/api/admin/map-cells` : `${BACKEND_URL}/api/admin/map-cells/${mapCellEdit.id}`;
                const method = isNew ? "POST" : "PUT";
                const ok = await exec(url, method, body);
                if (ok) { setMapCellEdit(null); fetchMapCells(mapCellsPage, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}
      {residualTraceEdit && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80"><div className="w-full max-w-2xl bg-zinc-950 border border-zinc-800 rounded-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
          <div className="flex items-center justify-between"><h3 className="text-sm font-bold text-zinc-100">Sửa dấu vết thế giới</h3><button onClick={()=>setResidualTraceEdit(null)} className="p-1 text-zinc-500 hover:text-white"><X className="w-4 h-4"/></button></div>
          <div className="text-[10px] text-zinc-500 font-mono">ID {String(residualTraceEdit.id)} · tạo {new Date(String(residualTraceEdit.created_at)).toLocaleString("vi-VN")}</div>
          <div className="grid grid-cols-2 gap-3">{([ ["region_id","Khu vực"],["x","X"],["y","Y"],["player_id","ID người chơi"],["action_type","Loại hành động"],["dominant_path","Đạo mạch"],["dao_pulse_strength","Cường độ xung lực"] ] as const).map(([key,label])=><label key={key} className="text-[11px] font-bold text-zinc-500">{label}<input value={residualTraceForm[key]} onChange={e=>setResidualTraceForm({...residualTraceForm,[key]:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>)}
            <label className="text-[11px] font-bold text-zinc-500">Hết hạn<input type="datetime-local" value={residualTraceForm.expires_at} onChange={e=>setResidualTraceForm({...residualTraceForm,expires_at:e.target.value})} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1"/></label>
          </div>
          <div className="flex justify-end gap-3"><button onClick={()=>setResidualTraceEdit(null)} className="px-4 py-2 rounded-lg text-xs text-zinc-400 border border-zinc-800">Hủy</button><button disabled={actionLoading} onClick={async()=>{const x=Number(residualTraceForm.x),y=Number(residualTraceForm.y),power=Number(residualTraceForm.dao_pulse_strength);if(!Number.isInteger(x)||!Number.isInteger(y)||!Number.isFinite(power)||power<0||!residualTraceForm.expires_at){addToast("Tọa độ, cường độ hoặc hạn dùng không hợp lệ","error");return;}const endpoint=`${BACKEND_URL}/api/admin/residual-traces/${residualTraceEdit.id}`;const prepared=prepareAdminAction("updateResidualTrace","cập nhật dấu vết thế giới",`${residualTraceEdit.action_type} · ${residualTraceEdit.id}`);if(!prepared)return;const ok=await exec(endpoint,"PUT",{...residualTraceForm,x,y,dao_pulse_strength:power,expires_at:new Date(residualTraceForm.expires_at).toISOString(),...prepared});if(ok){setResidualTraceEdit(null);fetchResidualTraces(residualTracePage,residualTraceSearch);}}} className="px-4 py-2 rounded-lg text-xs text-white bg-amber-600 hover:bg-amber-500 disabled:opacity-50">{actionLoading?"Đang lưu...":"Lưu"}</button></div>
        </div></div>
      )}
      {renderItemPicker()}
    </div>
  );
};
