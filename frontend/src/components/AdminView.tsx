import React, { useEffect, useState, useCallback } from "react";
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
} from "lucide-react";
import { translateGuType, translateItemType } from "@/lib/translate";
import { TopupWalletAdminTab } from "./admin/TopupWalletAdminTab";
import { DictionariesAdminTab } from "./admin/DictionariesAdminTab";
import { WorldExtendedAdminTab } from "./admin/WorldExtendedAdminTab";
import { SectMinesAdminTab } from "./admin/SectMinesAdminTab";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

const EFFECT_SUGGESTIONS: Record<string, { key: string; label: string; type: "string" | "number" | "boolean" }[]> = {
  active: [
    { key: "damage_flat", label: "Damage Flat (Sát thương cố định)", type: "number" },
    { key: "damage_mult", label: "Damage Mult (Hệ số sát thương)", type: "number" },
    { key: "damage_path", label: "Damage Path (Hệ sát thương)", type: "string" },
    { key: "heal_flat", label: "Heal Flat (Hồi máu)", type: "number" },
    { key: "shield_flat", label: "Shield Flat (Hộ thuẫn)", type: "number" },
    { key: "mana_regen", label: "Mana Regen (Hồi mana)", type: "number" },
    { key: "lifesteal_rate", label: "Life Steal Rate (Tỉ lệ hút máu)", type: "number" },
    { key: "apply_status", label: "Apply Status (Áp dụng trạng thái)", type: "string" },
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
    { key: "summon_growth_phantoms", label: "Summon Growth Phantoms", type: "boolean" }
  ],
  passive: [
    { key: "stat_base_attack", label: "Base Attack (Tấn công cơ bản)", type: "number" },
    { key: "stat_base_defense", label: "Base Defense (Phòng thủ cơ bản)", type: "number" },
    { key: "stat_base_hp", label: "Base HP (HP cơ bản)", type: "number" },
    { key: "stat_base_speed", label: "Base Speed (Tốc độ cơ bản)", type: "number" },
    { key: "mana_regen_mult", label: "Mana Regen Mult (Hệ số hồi mana)", type: "number" },
    { key: "soul_refinement", label: "Soul Refinement", type: "number" },
    { key: "movement", label: "Movement (Tốc độ di chuyển)", type: "number" },
    { key: "evasion", label: "Evasion (Né tránh)", type: "number" },
    { key: "storage_capacity", label: "Storage (Dung lượng chứa)", type: "number" },
    { key: "investigation", label: "Investigation (Khảo sát/Điều tra)", type: "number" },
    { key: "hearing", label: "Hearing (Thính giác)", type: "number" },
    { key: "path_conflict_reduction", label: "Path Conflict Reduct (Giảm xung đột hệ)", type: "number" },
    { key: "damage_multiplier_on_hit", label: "Damage Mult On Hit (Hệ số ST khi đánh)", type: "number" },
    { key: "refine_mana_tier", label: "Refine Mana Tier", type: "number" },
    { key: "cleansing_power", label: "Cleansing Power", type: "number" },
    { key: "luck", label: "Luck (May mắn)", type: "number" }
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
  primeval_stones: number; immortal_stones: number;
  is_admin: boolean; created_at: string; current_region: string;
  aptitude_type: string; wall_progress: number; human_qi: number;
  lifespan: number;
}

interface PlayerDetail {
  id: string; username: string; email: string; rank: number; sub_rank: number;
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
  gu_count: number;
  item_count: number;
}

interface PlayerItemRow {
  item_id: string;
  name: string;
  item_type: string;
  rank: number;
  category: string;
  quantity: number;
}

interface PlayerGuRow {
  instance_id: string;
  gu_id: string;
  name: string;
  rank: number;
  hunger: number;
  status: string;
  equipped: boolean;
  is_vital: boolean;
}

const rankNames = [
  "Nhất Chuyển", "Nhị Chuyển", "Tam Chuyển", "Tứ Chuyển",
  "Ngũ Chuyển", "Lục Chuyển Cổ Tiên", "Thất Chuyển Cổ Tiên", "Bát Chuyển Cổ Tiên", "Cửu Chuyển Cổ Tôn",
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

const AUDIT_ACTIONS = [
  { action: "gu.feed", label: "Cho ăn Cổ trùng", icon: "🌱" },
  { action: "gu.feed_group", label: "Cho ăn hàng loạt", icon: "🌱" },
  { action: "gu.equip", label: "Trang bị Cổ trùng", icon: "⚔️" },
  { action: "gu.unequip", label: "Tháo Cổ trùng", icon: "🛡️" },
  { action: "gu.set_vital", label: "Khóa Bản Mệnh Cổ", icon: "✨" },
  { action: "gu.drop", label: "Vứt Cổ trùng", icon: "🗑️" },
  { action: "gu.drop_batch", label: "Vứt Cổ trùng hàng loạt", icon: "🗑️" },
  { action: "gu.use_consumable", label: "Sử dụng Cổ trùng", icon: "🧪" },
  { action: "gu.starve_death", label: "Cổ trùng chết đói", icon: "💀" },
  { action: "gu.starve_vital_death", label: "Bản mệnh Cổ chết đói", icon: "💀" },
  { action: "refine.gu", label: "Hợp luyện Cổ", icon: "🔥" },
  { action: "refine.deduce_recipe", label: "Suy luận công thức", icon: "📜" },
  { action: "combat.started", label: "Bắt đầu chiến đấu", icon: "⚔️" },
  { action: "combat.ended", label: "Kết thúc chiến đấu", icon: "⚔️" },
  { action: "combat.finish", label: "Hoàn thành chiến đấu", icon: "🏆" },
  { action: "combat.flee", label: "Bỏ chạy chiến đấu", icon: "🏃" },
  { action: "combat.retreat", label: "Rút lui chiến đấu", icon: "🏃" },
  { action: "cultivate.breakthrough", label: "Đột phá tu vi", icon: "🌟" },
  { action: "cultivate.break_wall", label: "Phá vách", icon: "🧱" },
  { action: "cultivate.wall_cracked", label: "Vách nứt", icon: "🧱" },
  { action: "soul.fortify", label: "Luyện hồn", icon: "👻" },
  { action: "soul.refine", label: "Tinh luyện hồn", icon: "👻" },
  { action: "gacha.open", label: "Mở đá", icon: "💎" },
  { action: "gacha.open_bulk", label: "Mở đá hàng loạt", icon: "💎" },
  { action: "shop.buy", label: "Mua vật phẩm", icon: "🛒" },
  { action: "shop.buy_package", label: "Mua gói vật phẩm", icon: "🛒" },
  { action: "shop.sell", label: "Bán vật phẩm", icon: "💰" },
  { action: "marketplace.create_listing", label: "Đăng bán chợ", icon: "🏪" },
  { action: "marketplace.buy", label: "Mua từ chợ", icon: "🏪" },
  { action: "marketplace.sold", label: "Chợ bán thành công", icon: "🏪" },
  { action: "marketplace.cancel_listing", label: "Hủy đăng bán", icon: "🏪" },
  { action: "marketplace.bid", label: "Đặt giá đấu giá", icon: "🏷️" },
  { action: "marketplace.settle_auction_win", label: "Kết toán đấu giá thắng", icon: "🏷️" },
  { action: "marketplace.settle_auction_expired", label: "Kết toán đấu giá hết hạn", icon: "🏷️" },
  { action: "trade.initiate", label: "Mở giao dịch", icon: "🤝" },
  { action: "trade.confirm", label: "Xác nhận giao dịch", icon: "🤝" },
  { action: "trade.accept", label: "Chấp nhận giao dịch", icon: "🤝" },
  { action: "trade.complete", label: "Hoàn thành giao dịch", icon: "🤝" },
  { action: "trade.cancel", label: "Hủy giao dịch", icon: "🤝" },
  { action: "trade.decline", label: "Từ chối giao dịch", icon: "🤝" },
  { action: "trade.lock", label: "Khóa giao dịch", icon: "🔒" },
  { action: "trade.updated", label: "Cập nhật giao dịch", icon: "🤝" },
  { action: "mine.occupy", label: "Chiếm mỏ", icon: "⛏️" },
  { action: "mine.claim", label: "Thu hoạch mỏ", icon: "⛏️" },
  { action: "mine.abandon", label: "Bỏ mỏ", icon: "⛏️" },
  { action: "mine.attack", label: "Tấn công mỏ", icon: "⚔️" },
  { action: "quest.complete", label: "Hoàn thành nhiệm vụ", icon: "📋" },
  { action: "achievement.claim", label: "Nhận thành tựu", icon: "🏅" },
  { action: "sect.join", label: "Vào tông môn", icon: "🏯" },
  { action: "sect.leave", label: "Rời tông môn", icon: "🏯" },
  { action: "sect.donate_stones", label: "Quyên góp đá tông môn", icon: "🏯" },
  { action: "sect.donate_materials", label: "Quyên góp nguyên liệu tông môn", icon: "🏯" },
  { action: "sect.donate_gu", label: "Quyên góp Cổ trùng tông môn", icon: "🏯" },
  { action: "sect.buy_vault", label: "Mua kho tông môn", icon: "🏯" },
  { action: "player.register", label: "Đăng ký tài khoản", icon: "👤" },
  { action: "player.login", label: "Đăng nhập", icon: "🔑" },
  { action: "player.logout", label: "Đăng xuất", icon: "🔑" },
  { action: "player.updated", label: "Cập nhật nhân vật", icon: "👤" },
  { action: "player.heartbeat", label: "Heartbeat", icon: "💓" },
  { action: "gift_code.redeem", label: "Đổi mã quà", icon: "🎁" },
  { action: "topup.create_order", label: "Tạo đơn nạp tiền", icon: "💳" },
  { action: "topup.webhook_applied", label: "Nạp tiền thành công", icon: "💳" },
  { action: "topup.webhook_failed", label: "Nạp tiền thất bại", icon: "💳" },
  { action: "deposit.simulate", label: "Mô phỏng nạp tiền", icon: "💳" },
  { action: "admin.create_gu", label: "Admin tạo Cổ trùng", icon: "🔧" },
  { action: "admin.update_gu", label: "Admin sửa Cổ trùng", icon: "🔧" },
  { action: "admin.delete_gu", label: "Admin xóa Cổ trùng", icon: "🔧" },
  { action: "admin.add_gus", label: "Admin thêm Cổ trùng", icon: "🔧" },
  { action: "admin.remove_gu", label: "Admin xóa Cổ trùng NL", icon: "🔧" },
  { action: "admin.create_item", label: "Admin tạo vật phẩm", icon: "🔧" },
  { action: "admin.update_item", label: "Admin sửa vật phẩm", icon: "🔧" },
  { action: "admin.delete_item", label: "Admin xóa vật phẩm", icon: "🔧" },
  { action: "admin.add_items", label: "Admin thêm vật phẩm", icon: "🔧" },
  { action: "admin.remove_items", label: "Admin xóa vật phẩm NL", icon: "🔧" },
  { action: "admin.create_recipe", label: "Admin tạo công thức", icon: "🔧" },
  { action: "admin.update_recipe", label: "Admin sửa công thức", icon: "🔧" },
  { action: "admin.delete_recipe", label: "Admin xóa công thức", icon: "🔧" },
  { action: "admin.change_stones", label: "Admin đổi đá", icon: "🔧" },
  { action: "admin.update_player", label: "Admin sửa nhân vật", icon: "🔧" },
  { action: "admin.create_shop_item", label: "Admin tạo cửa hàng", icon: "🔧" },
  { action: "admin.update_shop_item", label: "Admin sửa cửa hàng", icon: "🔧" },
  { action: "admin.delete_shop_item", label: "Admin xóa cửa hàng", icon: "🔧" },
  { action: "admin.create_arena_shop_item", label: "Admin tạo cửa hàng ĐT", icon: "🔧" },
  { action: "admin.update_arena_shop_item", label: "Admin sửa cửa hàng ĐT", icon: "🔧" },
  { action: "admin.delete_arena_shop_item", label: "Admin xóa cửa hàng ĐT", icon: "🔧" },
  { action: "admin.create_gift_code", label: "Admin tạo mã quà", icon: "🔧" },
  { action: "admin.create_sect_mission", label: "Admin tạo nhiệm vụ TM", icon: "🔧" },
  { action: "admin.delete_sect_mission", label: "Admin xóa nhiệm vụ TM", icon: "🔧" },
  { action: "admin.update_sect", label: "Admin sửa tông môn", icon: "🔧" },
  { action: "admin.delete_sect", label: "Admin xóa tông môn", icon: "🔧" },
  { action: "admin.set_sect_leader", label: "Admin đặt trưởng lão TM", icon: "🔧" },
  { action: "admin.kick_sect_member", label: "Admin kick thành viên TM", icon: "🔧" },
  { action: "admin.stock_sect_vault", label: "Admin nạp kho tông môn", icon: "🔧" },
  { action: "admin.remove_sect_vault", label: "Admin rút kho tông môn", icon: "🔧" },
  { action: "admin.add_to_sect", label: "Admin thêm vào tông môn", icon: "🔧" },
  { action: "admin.update_mine_slot", label: "Admin sửa ô mỏ", icon: "🔧" },
  { action: "admin.reset_mine_slot", label: "Admin reset ô mỏ", icon: "🔧" },
  { action: "admin.import_data", label: "Admin nhập dữ liệu", icon: "📥" },
  { action: "admin.export_data", label: "Admin xuất dữ liệu", icon: "📤" },
  { action: "mana.snapshot", label: "Chụp snapshot mana", icon: "📸" },
];

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

export const AdminView: React.FC = () => {
  const { player, setView, addToast, guDictionary, itemDictionary, constitutionDictionary, raceDictionary } = useGameStore();
  const [activeTab, setActiveTab] = useState<"items" | "gus" | "pets">("items");
  const [players, setPlayers] = useState<PlayerRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [search, setSearch] = useState("");
  const [searchText, setSearchText] = useState("");
  const [searchBy, setSearchBy] = useState<"all" | "id" | "username" | "email" | "item" | "gu">("all");
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

  const [playerPets, setPlayerPets] = useState<any[]>([]);
  const [playerPetsTotal, setPlayerPetsTotal] = useState(0);
  const [playerPetsPage, setPlayerPetsPage] = useState(1);
  const [playerPetsLimit] = useState(20);
  const [playerPetsLoading, setPlayerPetsLoading] = useState(false);
  const [playerPetsSearch, setPlayerPetsSearch] = useState("");

  // Edit player modal
  const [showEditPlayer, setShowEditPlayer] = useState(false);
  const [editPlayerForm, setEditPlayerForm] = useState<Record<string, string>>({});

  const [showAddItem, setShowAddItem] = useState(false);
  const [addItemId, setAddItemId] = useState("");
  const [addItemQty, setAddItemQty] = useState(1);

  const [showRemoveItem, setShowRemoveItem] = useState<string | null>(null);
  const [removeItemQty, setRemoveItemQty] = useState(1);

  const [showAddGu, setShowAddGu] = useState(false);
  const [addGuId, setAddGuId] = useState("");
  const [addGuRank, setAddGuRank] = useState("");
  const [addGuQty, setAddGuQty] = useState(1);

  const [showStones, setShowStones] = useState(false);
  const [stonesPrimeval, setStonesPrimeval] = useState("");
  const [stonesImmortal, setStonesImmortal] = useState("");

  const [actionLoading, setActionLoading] = useState(false);
  const [tableLoading, setTableLoading] = useState<string | null>(null);
  const [section, setSection] = useState<"players" | "sects" | "data" | "logs" | "shop" | "mines" | "gift_codes" | "apertures" | "topup_wallet">("players");
  const [playersTab, setPlayersTab] = useState<"list" | "online">("list");
  const [minesTab, setMinesTab] = useState<"slots" | "sect_mines">("slots");

  // ----- Aperture state -----
  const [apertures, setApertures] = useState<any[]>([]);
  const [aperturesTotal, setAperturesTotal] = useState(0);
  const [aperturesLoading, setAperturesLoading] = useState(false);
  const [aperturesPage, setAperturesPage] = useState(1);
  const [aperturesLimit] = useState(50);
  const [aperturesSearch, setAperturesSearch] = useState("");
  const [selectedApertureId, setSelectedApertureId] = useState<string | null>(null);
  const [apertureDetail, setApertureDetail] = useState<any | null>(null);
  const [apertureDetailLoading, setApertureDetailLoading] = useState(false);
  const [apertureForm, setApertureForm] = useState<any>({});
  const [apertureSaving, setApertureSaving] = useState(false);

  // ----- Audit log state -----
  const [auditLogs, setAuditLogs] = useState<any[]>([]);
  const [auditLogTotal, setAuditLogTotal] = useState(0);
  const [auditLogLoading, setAuditLogLoading] = useState(false);
  const [auditLogPage, setAuditLogPage] = useState(0);
  const [auditLogLimit] = useState(50);
  const [auditLogFilterAction, setAuditLogFilterAction] = useState("");
  const [auditLogFilterPlayer, setAuditLogFilterPlayer] = useState("");
  const [auditLogFilterSuccess, setAuditLogFilterSuccess] = useState("");
  const [auditLogFilterSearch, setAuditLogFilterSearch] = useState("");
  const [auditLogExpandId, setAuditLogExpandId] = useState<string | null>(null);
  const [auditLogActionOpen, setAuditLogActionOpen] = useState(false);
  const [auditLogActionSearch, setAuditLogActionSearch] = useState("");

  // ----- Shop state -----
  const [shopItems, setShopItems] = useState<any[]>([]);
  const [shopLoading, setShopLoading] = useState(false);
  const [showShopAdd, setShowShopAdd] = useState(false);
  const [showShopEdit, setShowShopEdit] = useState<string | null>(null);
  const [shopForm, setShopForm] = useState({ id: "", rank: 0, price: 100, stock: -1, is_default: true, sort_order: 0 });

  // ----- Arena Shop state -----
  const [arenaShopItems, setArenaShopItems] = useState<any[]>([]);
  const [arenaShopLoading, setArenaShopLoading] = useState(false);
  const [showArenaShopAdd, setShowArenaShopAdd] = useState(false);
  const [showArenaShopEdit, setShowArenaShopEdit] = useState<string | null>(null);
  const [arenaShopForm, setArenaShopForm] = useState({ id: "", price: 50, qty: 1, stock: -1, is_default: true, sort_order: 0 });
  const [shopTab, setShopTab] = useState<"caravan" | "arena">("caravan");
  const [shopSearchGus, setShopSearchGus] = useState<any[]>([]);
  const [shopSearchItems, setShopSearchItems] = useState<any[]>([]);
  const [shopSearchRecipes, setShopSearchRecipes] = useState<any[]>([]);
  const [shopSearchLoaded, setShopSearchLoaded] = useState(false);
  const [itemPicker, setItemPicker] = useState<{source: string; tab: "gu" | "item" | "recipe"; query: string} | null>(null);
  const [giftPickerIdx, setGiftPickerIdx] = useState<number | null>(null);

  // ----- Mine state -----
  const [mineSlots, setMineSlots] = useState<any[]>([]);
  const [mineLoading, setMineLoading] = useState(false);
  const [showMineEdit, setShowMineEdit] = useState<number | null>(null);
  const [mineEditRarity, setMineEditRarity] = useState("");

  // ----- Gift codes state -----
  const [giftCodes, setGiftCodes] = useState<any[]>([]);
  const [giftLoading, setGiftLoading] = useState(false);
  const [newGiftCode, setNewGiftCode] = useState("");
  const [newGiftMaxUses, setNewGiftMaxUses] = useState(100);
  const [newGiftPerPlayer, setNewGiftPerPlayer] = useState(1);
  const [newGiftExpiry, setNewGiftExpiry] = useState(30);
  const [newGiftRewards, setNewGiftRewards] = useState<{type: string; id: string; amount: number; label: string; rank?: number}[]>([
    { type: "primeval_stones", id: "", amount: 1000, label: "1.000 Nguyên Thạch" },
  ]);

  // ----- Online players state -----
  const [onlinePlayers, setOnlinePlayers] = useState<any[]>([]);
  const [onlineLoading, setOnlineLoading] = useState(false);

  // ----- Dictionary state -----
  const [dataTab, setDataTab] = useState<"import" | "tables" | "gus" | "items" | "recipes" | "monsters" | "loot_tables" | "regions" | "map_cells" | "constitution" | "race" | "status" | "spawns" | "quests" | "dungeons">("import");
  const [regions, setRegions] = useState<any[]>([]);
  const [regionsLoading, setRegionsLoading] = useState(false);
  const [regionEdit, setRegionEdit] = useState<any | null>(null);
  const [regionSearch, setRegionSearch] = useState("");
  const [regionsPage, setRegionsPage] = useState(1);
  const [regionsTotal, setRegionsTotal] = useState(0);
  const [regionsTotalPages, setRegionsTotalPages] = useState(1);

  const [mapCells, setMapCells] = useState<any[]>([]);
  const [mapCellsLoading, setMapCellsLoading] = useState(false);
  const [mapCellEdit, setMapCellEdit] = useState<any | null>(null);
  const [mapCellSearch, setMapCellSearch] = useState("");
  const [mapCellRegionFilter, setMapCellRegionFilter] = useState("");
  const [mapCellTypeFilter, setMapCellTypeFilter] = useState("");
  const [mapCellsPage, setMapCellsPage] = useState(1);
  const [mapCellsTotal, setMapCellsTotal] = useState(0);
  const [mapCellsTotalPages, setMapCellsTotalPages] = useState(1);
  const [gus, setGus] = useState<any[]>([]);
  const [gusLoading, setGusLoading] = useState(false);
  const [guEdit, setGuEdit] = useState<any | null>(null);
  const [items, setItems] = useState<any[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  const [itemEdit, setItemEdit] = useState<any | null>(null);
  const [uploading, setUploading] = useState(false);
  const [recipes, setRecipes] = useState<any[]>([]);
  const [recipesLoading, setRecipesLoading] = useState(false);
  const [recipeEdit, setRecipeEdit] = useState<any | null>(null);
  const [recipeSearch, setRecipeSearch] = useState("");
  const [monsters, setMonsters] = useState<any[]>([]);
  const [monstersLoading, setMonstersLoading] = useState(false);
  const [monsterEdit, setMonsterEdit] = useState<any | null>(null);
  const [monsterSearch, setMonsterSearch] = useState("");
  const [monsterSortField, setMonsterSortField] = useState<string>("id");
  const [monsterSortDir, setMonsterSortDir] = useState<"asc" | "desc">("asc");
  const [lootTables, setLootTables] = useState<any[]>([]);
  const [lootTablesLoading, setLootTablesLoading] = useState(false);
  const [lootTableEdit, setLootTableEdit] = useState<any | null>(null);
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

      const payload: any = {};
      const addIfChanged = (key: string, originalVal: any, newVal: any) => {
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
        addIfChanged("tribulation_count", Number(apertureDetail.tribulation_count), Number(apertureForm.tribulation_count));
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
          tribulation_count: Number(apertureForm.tribulation_count),
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
    } catch (err: any) {
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
    } catch (err: any) {
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
    } catch (err: any) {
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
      if (auditLogFilterSearch) params.set("search", auditLogFilterSearch);
      const res = await authFetch(`${BACKEND_URL}/api/admin/audit-logs?${params}`);
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data.data || []);
        setAuditLogTotal(data.total || 0);
      }
    } catch {} finally { setAuditLogLoading(false); }
  }, [auditLogPage, auditLogLimit, auditLogFilterAction, auditLogFilterPlayer, auditLogFilterSuccess, auditLogFilterSearch]);

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
    if (section === "logs") fetchAuditLogs();
  }, [section, fetchAuditLogs]);

  const [auditLogSearchDebounce, setAuditLogSearchDebounce] = useState("");
  useEffect(() => {
    const t = setTimeout(() => setAuditLogFilterSearch(auditLogSearchDebounce), 400);
    return () => clearTimeout(t);
  }, [auditLogSearchDebounce]);
  useEffect(() => {
    if (section === "logs") { setAuditLogPage(0); }
  }, [auditLogFilterSearch]);

  useEffect(() => {
    if (!auditLogActionOpen) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest("[data-audit-action-combobox]")) {
        setAuditLogActionOpen(false);
        setAuditLogActionSearch("");
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [auditLogActionOpen]);

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
      if (dataTab === "gus") fetchGus();
      else if (dataTab === "items") fetchItems();
      else if (dataTab === "recipes") fetchRecipes();
      else if (dataTab === "monsters") fetchMonsters();
      else if (dataTab === "loot_tables") fetchLootTables();
      else if (dataTab === "regions") fetchRegions(1, regionSearch);
      else if (dataTab === "map_cells") fetchMapCells(1, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter);
    }
  }, [section, dataTab, fetchGus, fetchItems, fetchRecipes, fetchMonsters, fetchLootTables, fetchRegions, fetchMapCells, regionSearch, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter]);

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
      const params = new URLSearchParams({ page: String(page), limit: String(limit) });
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
    } catch (err: any) {
      setError(err.message);
    } finally { setLoading(false); }
  }, [page, limit, search, searchBy]);

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

  useEffect(() => {
    if (selectedId && activeTab === "items") fetchPlayerItems(selectedId);
  }, [selectedId, activeTab, playerItemsPage, playerItemsSearch, fetchPlayerItems]);

  useEffect(() => {
    if (selectedId && activeTab === "gus") fetchPlayerGus(selectedId);
  }, [selectedId, activeTab, playerGusPage, playerGusSearch, fetchPlayerGus]);

  useEffect(() => {
    if (selectedId && activeTab === "pets") fetchPlayerPets(selectedId);
  }, [selectedId, activeTab, playerPetsPage, playerPetsSearch, fetchPlayerPets]);

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
      if (selectedId) {
        fetchDetail(selectedId);
        if (activeTab === "items") fetchPlayerItems(selectedId);
        else if (activeTab === "gus") fetchPlayerGus(selectedId);
      }
      if (addToast) addToast(data.message || "Thành công", "success");
      return true;
    } catch (err: any) {
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
            let list: any[] = [];
            if (itemPicker.tab === "gu") list = shopSearchGus;
            else if (itemPicker.tab === "item") list = shopSearchItems;
            else list = shopSearchRecipes;
            const q = itemPicker.query.toLowerCase();
            if (q) list = list.filter((x: any) => (x.id || "").toLowerCase().includes(q) || (x.name || "").toLowerCase().includes(q));
            if (itemPicker.tab === "recipe") {
              list = list.filter((x: any) => x.target_gu_id);
            }
            return list.length === 0 ? (
              <div className="text-center py-8 text-zinc-500 text-xs italic">Không tìm thấy</div>
            ) : (
              list.slice(0, 100).map((x: any) => (
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
          <div className="flex gap-2 flex-wrap">
            <button onClick={() => setShowStones(true)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer">
              <Coins className="w-3.5 h-3.5" />
              Điều chỉnh tài nguyên
            </button>
            <button onClick={() => {
              if (detail) {
                setEditPlayerForm({
                  rank: String(detail.rank), sub_rank: String(detail.sub_rank),
                  aptitude_type: detail.aptitude_type || "", aptitude_percent: String(detail.aptitude_percent ?? ""),
                  aptitude_grade: detail.aptitude_grade || "", current_mana: String(detail.current_mana ?? ""),
                  max_mana: String(detail.max_mana ?? ""), hp: String(detail.hp ?? ""), base_hp: String(detail.base_hp ?? ""),
                  wall_progress: String(detail.wall_progress ?? ""), human_qi: String(detail.human_qi ?? ""),
                  lifespan: String(detail.lifespan ?? ""), current_region: detail.current_region || "",
                  x: String(detail.x ?? 0), y: String(detail.y ?? 0),
                  is_admin: String(detail.is_admin), soul_hp: String(detail.soul_hp ?? ""),
                  max_soul: String(detail.max_soul ?? ""), soul_tier: detail.soul_tier || "",
                  explore_count: String(detail.explore_count ?? ""), aperture_stress: String(detail.aperture_stress ?? ""),
                  bio: detail.bio || "", avatar_url: detail.avatar_url || "",
                  race: detail.race || "", gender: detail.gender || "", constitution: detail.constitution || "",
                  statuses: (detail.statuses || []).map((s: string | { id: string }) => typeof s === "string" ? s : s.id).join(", "),
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
            </button>
            <button onClick={() => {
              if (detail) {
                setSection("apertures");
                setApertureSearchDebounce(detail.username);
                setSelectedId(null);
                setDetail(null);
              }
            }} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer disabled:opacity-50">
              <Globe className="w-3.5 h-3.5 text-amber-500" />
              Xem Tiên Khiếu
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
                  <button onClick={() => { setAddItemId(""); setAddItemQty(1); setShowAddItem(true); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>
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
                          <th className="px-3 py-2.5 font-semibold">Bậc</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Số lượng</th>
                          <th className="px-3 py-2.5 font-semibold text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {playerItems.map((it) => {
                          const spec = itemDictionary[it.item_id];
                          return (
                            <tr key={it.item_id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                              <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[100px] truncate" title={it.item_id}>{it.item_id}</td>
                              <td className="px-3 py-2.5 font-semibold text-zinc-200">{it.name || spec?.name || it.item_id}</td>
                              <td className="px-3 py-2.5">
                                <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-zinc-900 border border-zinc-800 text-zinc-400">{translateItemType(it.item_type || spec?.item_type) || "?"}</span>
                              </td>
                              <td className="px-3 py-2.5 text-zinc-400">{it.rank || spec?.rank || "?"}</td>
                              <td className="px-3 py-2.5 text-right font-mono font-bold text-zinc-200">{it.quantity.toLocaleString()}</td>
                              <td className="px-3 py-2.5 text-right">
                                <button onClick={() => { setShowRemoveItem(it.item_id); setRemoveItemQty(1); }} className="p-1 rounded-lg hover:bg-red-950/40 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer" title="Xóa">
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
                  <button onClick={() => { setAddGuId(""); setAddGuRank(""); setAddGuQty(1); setShowAddGu(true); ensureShopSearchLoaded(); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
                    <Plus className="w-3.5 h-3.5" /> Thêm
                  </button>
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
                          <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                          <th className="px-3 py-2.5 font-semibold">Đói</th>
                          <th className="px-3 py-2.5 font-semibold">Trang bị</th>
                          <th className="px-3 py-2.5 font-semibold">Bản Mệnh</th>
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
                        {playerPets.map((pet: any) => (
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
                                  if (!confirm(`Xóa vĩnh viễn sủng vật ${pet.monster_name || pet.monster_template_id}?`)) return;
                                  const res = await authFetch(`${BACKEND_URL}/api/admin/players/${selectedId}/pets/${pet.id}`, { method: "DELETE" });
                                  if (res.ok) fetchPlayerPets(selectedId!);
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
                <div className="flex gap-2 mt-1">
                  <input value={addItemId} onChange={e => setAddItemId(e.target.value)} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "add-item", tab: "item", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                </div>
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
              <p className="text-xs text-zinc-400">Vật phẩm: <span className="text-zinc-200 font-bold">{itemDictionary[showRemoveItem]?.name || playerItems.find(i => i.item_id === showRemoveItem)?.name || showRemoveItem}</span></p>
              <p className="text-xs text-zinc-500">Hiện có: <span className="text-zinc-300 font-bold">{(() => { const row = playerItems.find(i => i.item_id === showRemoveItem); return (row?.quantity || 0).toLocaleString(); })()}</span></p>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lượng xóa</label>
                <input type="number" min={1} max={playerItems.find(i => i.item_id === showRemoveItem)?.quantity || 1} value={removeItemQty} onChange={e => setRemoveItemQty(Math.max(1, Math.min(parseInt(e.target.value) || 1, playerItems.find(i => i.item_id === showRemoveItem)?.quantity || 1)))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
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
        {showAddGu && (() => {
          const guRanks = addGuId ? shopSearchGus
            .filter((x: any) => x.id === addGuId)
            .map((x: any) => x.rank)
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
                  const ok = await exec(`${BACKEND_URL}/api/admin/players/${selectedId}/gus`, "POST", { gu_id: addGuId, quantity: addGuQty, ...(submitRank ? { rank: submitRank } : {}) });
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
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tu vi (1-9)</label>
                  <input type="number" min={1} max={9} value={editPlayerForm.rank || ""} onChange={e => setEditPlayerForm(f => ({ ...f, rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
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
                    {Object.values(constitutionDictionary || {}).map((c: any) => (
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
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.is_admin === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, is_admin: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Quyền Admin
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.is_disguised === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, is_disguised: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Đang Cải Trang
                  </label>
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.needs_onboarding === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, needs_onboarding: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Cần Onboarding
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
                  const body: any = {};
                  const num = (v: string) => (v === undefined || v === null || v.trim() === "" ? undefined : Number(v));
                  const set = (field: string, cur: any, next: any) => {
                    if (next === undefined || next === "") return;
                    if (String(cur) !== String(next)) body[field] = next;
                  };
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
                  set("is_admin", detail.is_admin, editPlayerForm.is_admin === "true");
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
                      {selectedSect.vault_recipes.map((vr: any, i: number) => (
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
                <select value={stockVaultType} onChange={e => setStockVaultType(e.target.value as any)} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50">
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
      <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
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
        <button onClick={() => setSection("gift_codes")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "gift_codes" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Gift className="w-3.5 h-3.5 inline mr-1.5" />Mã Quà
        </button>
        <button onClick={() => setSection("apertures")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "apertures" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Globe className="w-3.5 h-3.5 inline mr-1.5" />Tiên Khiếu
        </button>
        <button onClick={() => setSection("topup_wallet")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "topup_wallet" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Coins className="w-3.5 h-3.5 inline mr-1.5" />Nạp Tiền & Ví
        </button>
      </div>

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
                <select value={searchBy} onChange={e => { setSearchBy(e.target.value as any); setPage(1); }} className="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors cursor-pointer">
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
                <input type="text" value={searchText} onChange={e => setSearchText(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { setPage(1); setSearch(searchText); } }} placeholder={searchBy === "item" ? "ID hoặc tên vật phẩm..." : searchBy === "gu" ? "ID hoặc tên cổ trùng..." : "ID, tên đăng nhập hoặc email..."} className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors placeholder:text-zinc-600" />
              </div>
              <button onClick={() => { setPage(1); setSearch(searchText); }} className="px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">Tìm</button>
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
                      <th className="px-3 py-2.5 font-semibold">Sinh lực</th>
                      <th className="px-3 py-2.5 font-semibold">Chân nguyên</th>
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
                        {onlinePlayers.map((p: any) => (
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
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-6">
              <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-2">
                <Database className="w-4 h-4 text-emerald-400" />
                Import / Export CSV
              </h3>
              <p className="text-xs text-zinc-500 mb-6">
                Import dữ liệu từ file CSV vào database, hoặc export dữ liệu từ database ra file CSV.
              </p>
              <div className="flex flex-wrap gap-4">
                <button disabled={actionLoading} onClick={async () => { if (!confirm("Import sẽ ghi đè dữ liệu hiện tại theo cơ chế UPSERT. Tiếp tục?")) return; const ok = await exec(`${BACKEND_URL}/api/admin/data/import`, "POST", {}); if (ok) addToast("Import thành công!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <Database className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Import CSV → Database"}
                </button>
                <button disabled={actionLoading} onClick={async () => { const ok = await exec(`${BACKEND_URL}/api/admin/data/export`, "POST", {}); if (ok) addToast("Export thành công!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <RefreshCw className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Export Database → CSV"}
                </button>
                <button disabled={actionLoading} onClick={async () => { const ok = await exec(`${BACKEND_URL}/api/admin/clear-cache`, "POST", {}); if (ok) addToast("Cache đã được xóa và tải lại!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <Trash2 className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Xóa Cache Dictionary"}
                </button>
                <button disabled={actionLoading} onClick={async () => { if (!confirm("Thao tác này sẽ xóa toàn bộ cache Redis (bao gồm session, cooldown...). Tiếp tục?")) return; const ok = await exec(`${BACKEND_URL}/api/admin/clear-redis-cache`, "POST", {}); if (ok) addToast("Toàn bộ cache Redis đã được xóa!", "success"); }} className="flex items-center gap-2 px-5 py-3 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer">
                  <Trash2 className="w-4 h-4" />
                  {actionLoading ? "Đang xử lý..." : "Xóa Cache Redis"}
                </button>
              </div>
            </div>
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
                {[
                  { key: "item", label: "Vật phẩm", icon: Package, csv: "Item.csv" },
                  { key: "gu", label: "Cổ trùng", icon: Bug, csv: "Gu.csv" },
                  { key: "constitution", label: "Thể chất", icon: Activity, csv: "Constitution.csv" },
                  { key: "race", label: "Chủng tộc", icon: Users, csv: "Race.csv" },
                  { key: "recipe", label: "Cổ phương", icon: FlaskConical, csv: "Recipe.csv" },
                  { key: "shop_item", label: "Cửa hàng", icon: ShoppingBag, csv: "ShopItem.csv" },
                  { key: "arena_shop_item", label: "Cửa hàng Đấu trường", icon: Swords, csv: "ArenaShopItem.csv" },
                  { key: "region", label: "Khu vực", icon: Globe, csv: "Region.csv" },
                  { key: "loot_table", label: "Bảng loot", icon: Gift, csv: "LootTable.csv" },
                  { key: "monster", label: "Quái vật", icon: PawPrint, csv: "Monster.csv" },
                  { key: "monster_spawn", label: "Spawn quái", icon: MapPin, csv: "MonsterSpawn.csv" },
                  { key: "map_cell", label: "Bản đồ (boss/tài nguyên cao)", icon: Globe, csv: "MapCell.csv" },
                  { key: "sect", label: "Tông môn", icon: Building2, csv: "Sect.csv" },
                  { key: "sect_mission", label: "Nhiệm vụ tông môn", icon: ListTodo, csv: "SectMission.csv" },
                  { key: "quest", label: "Nhiệm vụ", icon: BookOpen, csv: "Quest.csv" },
                  { key: "dream_realm", label: "Mộng cảnh", icon: Star, csv: "DreamRealm.csv" },
                ].map(t => (
                  <div key={t.key} className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4 flex flex-col gap-3">
                    <div className="flex items-center gap-2">
                      <t.icon className="w-4 h-4 text-amber-400" />
                      <span className="text-xs font-semibold text-zinc-200">{t.label}</span>
                      <span className="text-[10px] text-zinc-500 font-mono ml-auto">{t.csv}</span>
                    </div>
                    <div className="flex gap-2">
                      <button
                        disabled={tableLoading === t.key}
                        onClick={async () => {
                          if (!confirm(`Import "${t.label}" từ CSV? Dữ liệu sẽ được UPSERT.`)) return;
                          setTableLoading(t.key);
                          const ok = await exec(`${BACKEND_URL}/api/admin/data/import/${t.key}`, "POST", {});
                          setTableLoading(null);
                          if (ok) addToast(`Import ${t.label} thành công!`, "success");
                        }}
                        className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-lg text-[11px] font-semibold transition-all cursor-pointer"
                      >
                        <Upload className="w-3 h-3" />
                        {tableLoading === t.key ? "..." : "Import"}
                      </button>
                      <button
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
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {dataTab === "gus" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><Bug className="w-4 h-4 text-amber-400" />Danh Sách Cổ Trùng</h3>
                <button onClick={() => setGuEdit({ id: "", name: "", rank: 1, path: [], gu_type: "Attack", base_mana_cost: 0, effects: { active: {}, passive: {}, house: {}, other: [] }, food_types: [], hunger_rate: 1, max_hunger: 100, description: "", target_type: "SINGLE", rarity: 0, is_immortal: false, primary_role: "", combat_roles: [], combat_tags: [], support_tags: [], base_price: 0, image_url: "", _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
              </div>
              <div className="relative mb-3">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                <input value={dictGuSearch} onChange={e => setDictGuSearch(e.target.value)} placeholder="Tìm kiếm theo ID hoặc tên..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500" />
              </div>
              {gusLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : gus.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có cổ trùng nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0">
                      {["ID", "Tên", "Rank", "Loại", "Giá"].map(col => {
                        const fieldMap: Record<string, string> = { "ID": "id", "Tên": "name", "Rank": "rank", "Loại": "gu_type", "Giá": "base_price" };
                        const f = fieldMap[col];
                        return <th key={col} onClick={() => { if (f) { if (guSortField === f) setGuSortDir(d => d === "asc" ? "desc" : "asc"); else { setGuSortField(f); setGuSortDir("asc"); } } }} className={`px-3 py-2.5 font-semibold ${f ? "cursor-pointer hover:text-zinc-200 select-none" : "cursor-default"} ${guSortField === f ? "text-amber-400" : ""} ${col === "Giá" ? "text-right" : ""}`}>{col}{guSortField === f ? (guSortDir === "asc" ? " ▲" : " ▼") : ""}</th>
                      })}
                      <th className="px-3 py-2.5"></th>
                    </tr></thead>
                    <tbody>{(() => {
                      const q = dictGuSearch.toLowerCase();
                      const filtered = q ? gus.filter((g: any) => (g.id?.toLowerCase() || "").includes(q) || (g.name?.toLowerCase() || "").includes(q)) : [...gus];
                      filtered.sort((a: any, b: any) => {
                        let va = a[guSortField], vb = b[guSortField];
                        if (typeof va === "string") va = va.toLowerCase();
                        if (typeof vb === "string") vb = vb.toLowerCase();
                        if (va < vb) return guSortDir === "asc" ? -1 : 1;
                        if (va > vb) return guSortDir === "asc" ? 1 : -1;
                        return 0;
                      });
                      return filtered.map((g: any) => (
                        <tr key={g.id + g.rank} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono">{g.id}</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200 flex items-center gap-2">
                            {g.image_url ? (
                              <img src={g.image_url} alt="" className="w-7 h-7 rounded object-cover border border-zinc-700 flex-shrink-0" />
                            ) : (
                              <span className="text-base flex-shrink-0">{({Active: "⚡", Buff: "✨", Consumable: "💊", Control: "🌀", Defense: "🛡️", Healing: "💚", Investigation: "🔍", Movement: "💨", Passive: "🧘", Refinement: "🔧", Special: "⭐", Storage: "📦", Summon: "👻", Support: "🎯", Transformation: "🦋", Vital: "💎"} as Record<string, string>)[g.gu_type] || "🏠"}</span>
                            )}
                            {g.name}
                          </td>
                          <td className="px-3 py-2.5 text-zinc-400">{g.rank}</td>
                          <td className="px-3 py-2.5 text-zinc-400">{translateGuType(g.gu_type)}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{g.base_price?.toLocaleString() || 0}</td>
                          <td className="px-3 py-2.5 text-right">
                            <button onClick={() => setGuEdit({ ...g, _old_rank: g.rank, effects: typeof g.effects === "string" ? JSON.parse(g.effects) : g.effects })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                            <button onClick={async () => { if (!confirm(`Xóa ${g.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/gu-dictionary/${g.id}`, "DELETE"); if (ok) fetchGus(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                          </td>
                        </tr>
                      ));
                    })()}</tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {dataTab === "items" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><Package className="w-4 h-4 text-emerald-400" />Danh Sách Vật Phẩm</h3>
                <button onClick={() => setItemEdit({ id: "", name: "", item_type: "Material", rank: 1, effects: {}, description: "", base_price: 0, image_url: "", _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
              </div>
              <div className="relative mb-3">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                <input value={dictItemSearch} onChange={e => setDictItemSearch(e.target.value)} placeholder="Tìm kiếm theo ID hoặc tên..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500" />
              </div>
              {itemsLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : items.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có vật phẩm nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0">
                      {["ID", "Tên", "Loại", "Rank", "Giá"].map(col => {
                        const fieldMap: Record<string, string> = { "ID": "id", "Tên": "name", "Loại": "item_type", "Rank": "rank", "Giá": "base_price" };
                        const f = fieldMap[col];
                        return <th key={col} onClick={() => { if (f) { if (itemSortField === f) setItemSortDir(d => d === "asc" ? "desc" : "asc"); else { setItemSortField(f); setItemSortDir("asc"); } } }} className={`px-3 py-2.5 font-semibold ${f ? "cursor-pointer hover:text-zinc-200 select-none" : "cursor-default"} ${itemSortField === f ? "text-amber-400" : ""} ${col === "Giá" ? "text-right" : ""}`}>{col}{itemSortField === f ? (itemSortDir === "asc" ? " ▲" : " ▼") : ""}</th>
                      })}
                      <th className="px-3 py-2.5"></th>
                    </tr></thead>
                    <tbody>{(() => {
                      const q = dictItemSearch.toLowerCase();
                      const filtered = q ? items.filter((it: any) => (it.id?.toLowerCase() || "").includes(q) || (it.name?.toLowerCase() || "").includes(q)) : [...items];
                      filtered.sort((a: any, b: any) => {
                        let va = a[itemSortField], vb = b[itemSortField];
                        if (typeof va === "string") va = va.toLowerCase();
                        if (typeof vb === "string") vb = vb.toLowerCase();
                        if (va < vb) return itemSortDir === "asc" ? -1 : 1;
                        if (va > vb) return itemSortDir === "asc" ? 1 : -1;
                        return 0;
                      });
                      return filtered.map((it: any) => (
                        <tr key={it.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono">{it.id}</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200 flex items-center gap-2">
                            {it.image_url ? (
                              <img src={it.image_url} alt="" className="w-7 h-7 rounded object-cover border border-zinc-700 flex-shrink-0" />
                            ) : (
                              <Package className="w-4 h-4 text-zinc-500 flex-shrink-0" />
                            )}
                            {it.name}
                          </td>
                          <td className="px-3 py-2.5 text-zinc-400">{translateItemType(it.item_type)}</td>
                          <td className="px-3 py-2.5 text-zinc-400">{it.rank}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{it.base_price?.toLocaleString() || 0}</td>
                          <td className="px-3 py-2.5 text-right">
                            <button onClick={() => setItemEdit({ ...it, effects: typeof it.effects === "string" ? JSON.parse(it.effects) : it.effects })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                            <button onClick={async () => { if (!confirm(`Xóa ${it.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/item-dictionary/${it.id}`, "DELETE"); if (ok) fetchItems(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                          </td>
                        </tr>
                      ));
                    })()}</tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {dataTab === "recipes" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><FlaskConical className="w-4 h-4 text-emerald-400" />Danh Sách Cổ Phương</h3>
                <button onClick={() => setRecipeEdit({ id: "", target_gu_id: "", target_gu_name: "", required_gus: {}, required_materials: {}, success_rate: 0.5, primeval_stones_cost: 0, immortal_essence_stones_cost: 0, _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
              </div>
              <div className="relative mb-3">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                <input value={recipeSearch} onChange={e => setRecipeSearch(e.target.value)} placeholder="Tìm kiếm theo ID hoặc tên..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500" />
              </div>
              {recipesLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : recipes.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có cổ phương nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0"><th className="px-3 py-2.5 font-semibold">ID</th><th className="px-3 py-2.5 font-semibold">Cổ trùng đích</th><th className="px-3 py-2.5 font-semibold text-right">Tỉ lệ</th><th className="px-3 py-2.5 font-semibold text-right">Nguyên Thạch</th><th className="px-3 py-2.5"></th></tr></thead>
                    <tbody>{recipes.filter(r => r.id.toLowerCase().includes(recipeSearch.toLowerCase()) || (r.target_gu_name || r.target_gu_id || "").toLowerCase().includes(recipeSearch.toLowerCase())).map((r: any) => (
                      <tr key={r.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 text-zinc-500 font-mono">{r.id}</td>
                        <td className="px-3 py-2.5 font-semibold text-zinc-200">{r.target_gu_name || r.target_gu_id}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{(r.success_rate * 100).toFixed(0)}%</td>
                        <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{r.primeval_stones_cost?.toLocaleString() || 0}</td>
                        <td className="px-3 py-2.5 text-right">
                          <button onClick={() => setRecipeEdit({ ...r, required_gus: typeof r.required_gus === "string" ? JSON.parse(r.required_gus) : r.required_gus, required_materials: typeof r.required_materials === "string" ? JSON.parse(r.required_materials) : r.required_materials })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                          <button onClick={async () => { if (!confirm(`Xóa ${r.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/recipes/${r.id}`, "DELETE"); if (ok) fetchRecipes(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                        </td>
                      </tr>
                    ))}</tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {dataTab === "monsters" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><PawPrint className="w-4 h-4 text-amber-400" />Danh Sách Quái Vật</h3>
                <button onClick={() => setMonsterEdit({ id: "", name: "", type: "beast", max_hp: 100, base_attack: 10, base_defense: 5, gu_inventory: [], item_inventory: {}, dao_marks: {}, path_masteries: {}, ai_profile: "beast_basic", loot_table_id: "", description: "", is_active: true, is_capturable: false, capture_chance: 0.20, pet_hunger_per_hour: 2.0, _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
              </div>
              <div className="relative mb-3">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                <input value={monsterSearch} onChange={e => setMonsterSearch(e.target.value)} placeholder="Tìm kiếm theo ID hoặc tên..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500" />
              </div>
              {monstersLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : monsters.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có quái vật nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0">
                      {["ID", "Tên", "Loại", "HP", "Tấn công", "Phòng thủ", "Loot"].map(col => {
                        const fieldMap: Record<string, string> = { "ID": "id", "Tên": "name", "Loại": "type", "HP": "max_hp", "Tấn công": "base_attack", "Phòng thủ": "base_defense", "Loot": "loot_table_id" };
                        const f = fieldMap[col];
                        return <th key={col} onClick={() => { if (f) { if (monsterSortField === f) setMonsterSortDir(d => d === "asc" ? "desc" : "asc"); else { setMonsterSortField(f); setMonsterSortDir("asc"); } } }} className={`px-3 py-2.5 font-semibold ${f ? "cursor-pointer hover:text-zinc-200 select-none" : "cursor-default"} ${monsterSortField === f ? "text-amber-400" : ""}`}>{col}{monsterSortField === f ? (monsterSortDir === "asc" ? " ▲" : " ▼") : ""}</th>
                      })}
                      <th className="px-3 py-2.5"></th>
                    </tr></thead>
                    <tbody>{(() => {
                      const q = monsterSearch.toLowerCase();
                      const filtered = q ? monsters.filter((m: any) => (m.id?.toLowerCase() || "").includes(q) || (m.name?.toLowerCase() || "").includes(q)) : [...monsters];
                      filtered.sort((a: any, b: any) => {
                        let va = a[monsterSortField], vb = b[monsterSortField];
                        if (typeof va === "string") va = va.toLowerCase();
                        if (typeof vb === "string") vb = vb.toLowerCase();
                        if (va < vb) return monsterSortDir === "asc" ? -1 : 1;
                        if (va > vb) return monsterSortDir === "asc" ? 1 : -1;
                        return 0;
                      });
                      return filtered.map((m: any) => (
                        <tr key={m.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono">{m.id}</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200">{m.name}</td>
                          <td className="px-3 py-2.5 text-zinc-300">{m.type}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-red-400">{m.max_hp}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-orange-400">{m.base_attack}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-blue-400">{m.base_defense}</td>
                          <td className="px-3 py-2.5 text-zinc-400 font-mono text-[10px]">{m.loot_table_id || "-"}</td>
                          <td className="px-3 py-2.5 text-right">
                            <button onClick={() => setMonsterEdit({ ...m, gu_inventory: typeof m.gu_inventory === "string" ? JSON.parse(m.gu_inventory) : m.gu_inventory || [], item_inventory: typeof m.item_inventory === "string" ? JSON.parse(m.item_inventory) : m.item_inventory || {}, dao_marks: typeof m.dao_marks === "string" ? JSON.parse(m.dao_marks) : m.dao_marks || {}, path_masteries: typeof m.path_masteries === "string" ? JSON.parse(m.path_masteries) : m.path_masteries || {} })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                            <button onClick={async () => { if (!confirm(`Xóa quái vật ${m.name}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/monsters/${m.id}`, "DELETE"); if (ok) fetchMonsters(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                          </td>
                        </tr>
                      ));
                    })()}</tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {dataTab === "loot_tables" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><Gift className="w-4 h-4 text-amber-400" />Danh Sách Bảng Loot</h3>
                <button onClick={() => setLootTableEdit({ id: "", name: "", entries: [], _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
              </div>
              <div className="relative mb-3">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
                <input value={lootTableSearch} onChange={e => setLootTableSearch(e.target.value)} placeholder="Tìm kiếm theo ID hoặc tên..." className="w-full pl-9 pr-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500" />
              </div>
              {lootTablesLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : lootTables.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có bảng loot nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0">
                      {["ID", "Tên", "Số loot"].map(col => {
                        const fieldMap: Record<string, string> = { "ID": "id", "Tên": "name" };
                        const f = fieldMap[col];
                        return <th key={col} onClick={() => { if (f) { if (lootTableSortField === f) setLootTableSortDir(d => d === "asc" ? "desc" : "asc"); else { setLootTableSortField(f); setLootTableSortDir("asc"); } } }} className={`px-3 py-2.5 font-semibold ${f ? "cursor-pointer hover:text-zinc-200 select-none" : "cursor-default"} ${lootTableSortField === f ? "text-amber-400" : ""}`}>{col}{lootTableSortField === f ? (lootTableSortDir === "asc" ? " ▲" : " ▼") : ""}</th>
                      })}
                      <th className="px-3 py-2.5"></th>
                    </tr></thead>
                    <tbody>{(() => {
                      const q = lootTableSearch.toLowerCase();
                      const filtered = q ? lootTables.filter((t: any) => (t.id?.toLowerCase() || "").includes(q) || (t.name?.toLowerCase() || "").includes(q)) : [...lootTables];
                      filtered.sort((a: any, b: any) => {
                        let va = a[lootTableSortField], vb = b[lootTableSortField];
                        if (typeof va === "string") va = va.toLowerCase();
                        if (typeof vb === "string") vb = vb.toLowerCase();
                        if (va < vb) return lootTableSortDir === "asc" ? -1 : 1;
                        if (va > vb) return lootTableSortDir === "asc" ? 1 : -1;
                        return 0;
                      });
                      return filtered.map((t: any) => {
                        const entries = typeof t.entries === "string" ? JSON.parse(t.entries) : (t.entries || []);
                        return (
                          <tr key={t.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                            <td className="px-3 py-2.5 text-zinc-500 font-mono">{t.id}</td>
                            <td className="px-3 py-2.5 font-semibold text-zinc-200">{t.name}</td>
                            <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{entries.length}</td>
                            <td className="px-3 py-2.5 text-right">
                              <button onClick={() => setLootTableEdit({ ...t, entries })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                              <button onClick={async () => { if (!confirm(`Xóa bảng loot ${t.name}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/loot-tables/${t.id}`, "DELETE"); if (ok) fetchLootTables(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                            </td>
                          </tr>
                        );
                      });
                    })()}</tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {dataTab === "regions" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <Globe className="w-4 h-4 text-amber-400" />Quản Lý Khu Vực
                </h3>
                <button 
                  onClick={() => setRegionEdit({ id: "", name: "", description: "", width: 100, height: 100, min_rank: 1, max_rank: 5, is_active: true, zone_type: "KHU_VUC", min_x: 0, min_y: 0, max_x: 0, max_y: 0, entry_x: 0, entry_y: 0, pool_weights: { cell_type_weights: {}, resources: [], wild_gus: [] }, _isNew: true })} 
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer w-full sm:w-auto"
                >
                  <Plus className="w-3.5 h-3.5" />Thêm Khu Vực
                </button>
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
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2">ID</th>
                          <th className="px-3 py-2">Tên</th>
                          <th className="px-3 py-2">Mô tả</th>
                          <th className="px-3 py-2">Loại</th>
                          <th className="px-3 py-2">Kích thước</th>
                          <th className="px-3 py-2">Tọa độ</th>
                          <th className="px-3 py-2">Cảnh giới</th>
                          <th className="px-3 py-2 text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {regions.map((r: any) => (
                          <tr key={r.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                            <td className="px-3 py-2 font-mono text-zinc-400">{r.id}</td>
                            <td className="px-3 py-2 font-semibold text-zinc-200">{r.name}</td>
                            <td className="px-3 py-2 text-zinc-400 max-w-[150px] truncate" title={r.description}>{r.description || "–"}</td>
                            <td className="px-3 py-2 text-amber-400">{r.zone_type}</td>
                            <td className="px-3 py-2 text-zinc-400">{r.width}x{r.height}</td>
                            <td className="px-3 py-2 text-zinc-500 font-mono text-[10px]">({r.min_x},{r.min_y}) to ({r.max_x},{r.max_y})</td>
                            <td className="px-3 py-2 text-zinc-300">C{r.min_rank} - C{r.max_rank}</td>
                            <td className="px-3 py-2 text-right flex justify-end gap-2">
                              <button onClick={() => setRegionEdit(r)} className="text-amber-500 hover:text-amber-400 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5" /></button>
                              <button onClick={async () => { if (!confirm(`Xóa khu vực "${r.name}" (${r.id})? Hành động này có thể ảnh hưởng đến dữ liệu liên kết.`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/regions/${r.id}`, "DELETE"); if (ok) fetchRegions(regionsPage, regionSearch); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5" /></button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
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

          {dataTab === "map_cells" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <MapPin className="w-4 h-4 text-amber-400" />Quản Lý Ô Bản Đồ
                </h3>
                <button 
                  onClick={() => setMapCellEdit({ id: "", region_id: "", x: 0, y: 0, cell_type: "Resource", content_id: "", metadata: {}, respawn_seconds: 300, is_active: true, _isNew: true })} 
                  className="flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer w-full sm:w-auto"
                >
                  <Plus className="w-3.5 h-3.5" />Thêm Ô Bản Đồ
                </button>
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
                  <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                    <table className="w-full text-left text-xs">
                      <thead>
                        <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                          <th className="px-3 py-2">Khu vực</th>
                          <th className="px-3 py-2">Tọa độ (X, Y)</th>
                          <th className="px-3 py-2">Loại ô</th>
                          <th className="px-3 py-2">Content ID</th>
                          <th className="px-3 py-2">Respawn</th>
                          <th className="px-3 py-2">Metadata</th>
                          <th className="px-3 py-2">Active</th>
                          <th className="px-3 py-2 text-right">Thao tác</th>
                        </tr>
                      </thead>
                      <tbody>
                        {mapCells.map((c: any) => (
                          <tr key={c.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                            <td className="px-3 py-2 text-zinc-300 font-mono text-[10px]">{c.region_id}</td>
                            <td className="px-3 py-2 font-mono text-amber-400">({c.x}, {c.y})</td>
                            <td className="px-3 py-2">
                              <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                c.cell_type === "Safe" ? "bg-emerald-950/80 text-emerald-400 border border-emerald-500/30" :
                                c.cell_type === "Monster" ? "bg-red-950/80 text-red-400 border border-red-500/30" :
                                c.cell_type === "Portal" ? "bg-purple-950/80 text-purple-400 border border-purple-500/30" :
                                "bg-amber-950/80 text-amber-400 border border-amber-500/30"
                              }`}>{c.cell_type}</span>
                            </td>
                            <td className="px-3 py-2 font-mono text-zinc-400">{c.content_id || "–"}</td>
                            <td className="px-3 py-2 text-zinc-400">{c.respawn_seconds}s</td>
                            <td className="px-3 py-2 text-zinc-500 font-mono text-[10px] max-w-[120px] truncate" title={JSON.stringify(c.metadata)}>{JSON.stringify(c.metadata)}</td>
                            <td className="px-3 py-2">{c.is_active ? <span className="text-emerald-400 font-bold">ON</span> : <span className="text-zinc-600">OFF</span>}</td>
                            <td className="px-3 py-2 text-right flex justify-end gap-2">
                              <button onClick={() => setMapCellEdit(c)} className="text-amber-500 hover:text-amber-400 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5" /></button>
                              <button onClick={async () => { if (!confirm(`Xóa ô bản đồ tại (${c.x}, ${c.y})?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/map-cells/${c.id}`, "DELETE"); if (ok) fetchMapCells(mapCellsPage, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5" /></button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
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

          {dataTab === "constitution" && <DictionariesAdminTab type="constitution" addToast={addToast} />}
          {dataTab === "race" && <DictionariesAdminTab type="race" addToast={addToast} />}
          {dataTab === "status" && <DictionariesAdminTab type="status" addToast={addToast} />}
          {dataTab === "spawns" && <WorldExtendedAdminTab type="spawns" addToast={addToast} />}
          {dataTab === "quests" && <WorldExtendedAdminTab type="quests" addToast={addToast} />}
          {dataTab === "dungeons" && <WorldExtendedAdminTab type="dungeons" addToast={addToast} />}
        </div>
      )}

      {/* Topup & Wallet Section */}
      {section === "topup_wallet" && <TopupWalletAdminTab addToast={addToast} />}

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
              <div className="flex-1 min-w-[200px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tìm kiếm</label>
                <input value={auditLogSearchDebounce} onChange={e => setAuditLogSearchDebounce(e.target.value)} placeholder="Tìm trong action, mô tả, entity..." className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex-1 min-w-[180px] relative" data-audit-action-combobox>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Hành động</label>
                <div className="relative">
                  <button onClick={() => setAuditLogActionOpen(!auditLogActionOpen)} className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 text-left flex items-center justify-between focus:outline-none focus:border-amber-500/50 cursor-pointer">
                    <span className={auditLogFilterAction ? "" : "text-zinc-600"}>{auditLogFilterAction ? AUDIT_ACTIONS.find(a => a.action === auditLogFilterAction)?.label || auditLogFilterAction : "-- Tất cả --"}</span>
                    <svg className={`w-3 h-3 text-zinc-500 transition-transform ${auditLogActionOpen ? "rotate-180" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" /></svg>
                  </button>
                  {auditLogActionOpen && (
                    <div data-audit-action-combobox className="absolute z-50 mt-1 w-full bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl max-h-[280px] overflow-hidden flex flex-col">
                      <div className="p-1.5 border-b border-zinc-800">
                        <input autoFocus value={auditLogActionSearch} onChange={e => setAuditLogActionSearch(e.target.value)} placeholder="Tìm action..." className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none" />
                      </div>
                      <div className="overflow-y-auto max-h-[230px]">
                        <button onClick={() => { setAuditLogFilterAction(""); setAuditLogActionOpen(false); setAuditLogActionSearch(""); setAuditLogPage(0); }} className="w-full text-left px-3 py-1.5 text-xs hover:bg-amber-600/20 text-zinc-400 hover:text-amber-400 cursor-pointer">-- Tất cả --</button>
                        {AUDIT_ACTIONS.filter(a => !auditLogActionSearch || a.action.includes(auditLogActionSearch.toLowerCase()) || a.label.toLowerCase().includes(auditLogActionSearch.toLowerCase())).map(a => (
                          <button key={a.action} onClick={() => { setAuditLogFilterAction(a.action); setAuditLogActionOpen(false); setAuditLogActionSearch(""); setAuditLogPage(0); }} className={`w-full text-left px-3 py-1.5 text-xs hover:bg-amber-600/20 cursor-pointer flex items-center gap-2 ${auditLogFilterAction === a.action ? "text-amber-400 bg-amber-600/10" : "text-zinc-300"}`}>
                            <span>{a.icon}</span>
                            <span className="truncate">{a.label}</span>
                            <span className="ml-auto text-[10px] text-zinc-600 font-mono shrink-0">{a.action}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <div className="flex-1 min-w-[120px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Player ID</label>
                <input value={auditLogFilterPlayer} onChange={e => setAuditLogFilterPlayer(e.target.value)} placeholder="UUID..." className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="w-[100px]">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Kết quả</label>
                <select value={auditLogFilterSuccess} onChange={e => { setAuditLogFilterSuccess(e.target.value); setAuditLogPage(0); }} className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50">
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
                <button onClick={() => { setShopForm({ id: "", rank: 0, price: 100, stock: -1, is_default: true, sort_order: 0 }); setShowShopAdd(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer">
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
                        <th className="px-3 py-2.5 font-semibold">Rank</th>
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
                        <tr key={`${item.id}-${item.rank ?? 0}`} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono">{item.id}</td>
                          <td className="px-3 py-2.5 text-violet-400 font-mono">{item.rank ?? 0}</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200">{item.name || item.id}</td>
                          <td className="px-3 py-2.5 text-zinc-400">{translateItemType(item.item_type) || "—"}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-yellow-500">{item.price?.toLocaleString()}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{item.stock === -1 ? "∞" : item.stock}</td>
                          <td className="px-3 py-2.5 text-center">{item.is_default ? <span className="text-emerald-400">✓</span> : <span className="text-zinc-600">–</span>}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-400">{item.sort_order}</td>
                          <td className="px-3 py-2.5 text-right">
                            <button onClick={() => { setShopForm({ id: item.id, rank: item.rank ?? 0, price: item.price, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowShopEdit(`${item.id}/${item.rank ?? 0}`); }} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                            <button onClick={async () => { if (!confirm(`Xóa ${item.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/shop/${item.id}/${item.rank ?? 0}`, "DELETE"); if (ok) fetchShopItems(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
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
                <button onClick={() => { setArenaShopForm({ id: "", price: 50, qty: 1, stock: -1, is_default: true, sort_order: 0 }); setShowArenaShopAdd(true); }} className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">
                  <Plus className="w-3.5 h-3.5" />Thêm
                </button>
              </div>
              {arenaShopLoading ? (
                <div className="flex justify-center py-10"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div>
              ) : arenaShopItems.length === 0 ? (
                <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                  <Sword className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
                  <p className="text-xs text-zinc-500">Chưa có mặt hàng nào.</p>
                </div>
              ) : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                  <table className="w-full text-left text-xs">
                    <thead>
                      <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                        <th className="px-3 py-2.5 font-semibold">ID</th>
                        <th className="px-3 py-2.5 font-semibold">Tên</th>
                        <th className="px-3 py-2.5 font-semibold text-right">Giá (Xu ĐT)</th>
                        <th className="px-3 py-2.5 font-semibold text-right">SL nhận</th>
                        <th className="px-3 py-2.5 font-semibold text-right">Tồn kho</th>
                        <th className="px-3 py-2.5 font-semibold text-center">Mặc định</th>
                        <th className="px-3 py-2.5 font-semibold text-right">Thứ tự</th>
                        <th className="px-3 py-2.5"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {arenaShopItems.map((item: any) => (
                        <tr key={item.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                          <td className="px-3 py-2.5 text-zinc-500 font-mono">{item.id}</td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200">{item.name || item.id}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-amber-500">{item.price?.toLocaleString()}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{item.qty}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{item.stock === -1 ? "∞" : item.stock}</td>
                          <td className="px-3 py-2.5 text-center">{item.is_default ? <span className="text-emerald-400">✓</span> : <span className="text-zinc-600">–</span>}</td>
                          <td className="px-3 py-2.5 text-right font-mono text-zinc-400">{item.sort_order}</td>
                          <td className="px-3 py-2.5 text-right">
                            <button onClick={() => { setArenaShopForm({ id: item.id, price: item.price, qty: item.qty, stock: item.stock, is_default: item.is_default, sort_order: item.sort_order }); setShowArenaShopEdit(item.id); }} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
                            <button onClick={async () => { if (!confirm(`Xóa ${item.id}?`)) return; const ok = await exec(`${BACKEND_URL}/api/admin/arena-shop/${item.id}`, "DELETE"); if (ok) fetchArenaShopItems(); }} className="text-red-400 hover:text-red-300 cursor-pointer" title="Xóa"><Trash2 className="w-3.5 h-3.5 inline" /></button>
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
                    giftCodes.map((gc: any) => (
                      <tr key={gc.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5 font-mono font-bold text-amber-400">{gc.code}</td>
                        <td className="px-3 py-2.5 text-zinc-300 max-w-[200px] truncate">{gc.rewards?.map((r: any) => r.label).join(", ")}</td>
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
                    <th className="px-3 py-2.5 font-semibold">Duyệt lên cấp</th>
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
                      <td className="px-3 py-2.5">{s.require_promotion_approval ? <span className="text-amber-400 text-[10px]">✓</span> : <span className="text-zinc-600">–</span>}</td>
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
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                      <th className="px-3 py-2.5 font-semibold">Người chơi</th>
                      <th className="px-3 py-2.5 font-semibold">Phẩm cấp</th>
                      <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Tốc độ TG</th>
                      <th className="px-3 py-2.5 font-semibold text-right">HP Tiên Khiếu</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Diện tích</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Tiên Nguyên</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Thiên Kiếp Kế</th>
                      <th className="px-3 py-2.5 font-semibold text-right">Số Kiếp Nạn</th>
                      <th className="px-3 py-2.5 font-semibold text-center">Hành động</th>
                    </tr>
                  </thead>
                  <tbody>
                    {apertures.map((ap: any) => (
                      <tr key={ap.id} className="border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors">
                        <td className="px-3 py-2.5">
                          <span className="font-semibold text-zinc-200 block">{ap.username}</span>
                          <span className="text-[10px] text-zinc-500 font-mono">{ap.id}</span>
                        </td>
                        <td className="px-3 py-2.5 font-bold text-zinc-300">
                          {ap.grade === "LOW" ? "Hạ Phẩm" : ap.grade === "MID" ? "Trung Phẩm" : ap.grade === "HIGH" ? "Thượng Phẩm" : ap.grade === "EXTREME" ? "Chí Tôn" : ap.grade}
                        </td>
                        <td className="px-3 py-2.5 font-semibold">
                          <span className={`px-2 py-0.5 rounded text-[10px] uppercase font-mono ${
                            ap.state === "STABLE" ? "bg-emerald-950/55 text-emerald-400 border border-emerald-800/30" : 
                            ap.state === "TRIBULATION_PENDING" ? "bg-amber-950/55 text-amber-400 border border-amber-800/30" : 
                            ap.state === "TRIBULATION_ACTIVE" ? "bg-red-950/55 text-red-400 border border-red-800/30 animate-pulse" : 
                            "bg-zinc-800 text-zinc-400"
                          }`}>
                            {ap.state === "STABLE" ? "Ổn định" : ap.state === "TRIBULATION_PENDING" ? "Đợi Kiếp" : ap.state === "TRIBULATION_ACTIVE" ? "Đang Kiếp" : ap.state}
                          </span>
                        </td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{ap.time_rate}x</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{ap.aperture_hp?.toLocaleString()}/{ap.max_aperture_hp?.toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{ap.area_size?.toLocaleString()} mẫu</td>
                        <td className="px-3 py-2.5 text-right font-mono text-yellow-500 font-bold">{ap.immortal_essence?.toLocaleString()}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">Năm {Math.round(ap.next_tribulation_year || 0)}</td>
                        <td className="px-3 py-2.5 text-right font-mono text-zinc-300">{ap.tribulation_count}</td>
                        <td className="px-3 py-2.5 text-center">
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
                      <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tổng số lần độ kiếp</label>
                      <input type="number" value={apertureForm.tribulation_count || 0} onChange={e => setApertureForm({ ...apertureForm, tribulation_count: parseInt(e.target.value) || 0 })} className="w-full px-3 py-2 bg-zinc-850 border border-zinc-700 rounded-lg text-xs text-zinc-200" />
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
                      apertureForm.dao_marks_array.map((entry: any, index: number) => (
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
                              const arr = apertureForm.dao_marks_array.filter((_: any, idx: number) => idx !== index);
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
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-2xl mx-4 shadow-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
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
              <div><label className="flex items-center gap-2 pt-5"><input type="checkbox" checked={guEdit.is_immortal} onChange={e => setGuEdit({...guEdit, is_immortal: e.target.checked})} className="rounded bg-zinc-800 border-zinc-700" /><span className="text-[11px] text-zinc-400">Bất Tử</span></label></div>
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
                      } catch (err: any) {
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
            {/* Gu Effects - structured */}
            <div className="mt-4 border-t border-zinc-800 pt-4">
              <h4 className="text-xs font-bold text-zinc-400 mb-3 uppercase tracking-wider">Hiệu Ứng</h4>
              {["active", "passive", "house", "growth"].map(cat => {
                const eff = guEdit.effects?.[cat] || {};
                return (
                  <details key={cat} className="mb-2 bg-zinc-800/30 rounded-lg">
                    <summary className="px-3 py-2 text-xs font-bold text-zinc-400 uppercase cursor-pointer">{cat} ({Object.keys(eff).length} fields)</summary>
                    <div className="px-3 pb-3">
                      {Object.entries(eff).map(([k, val], idx) => {
                        const v = val as any;
                        const fieldType = typeof v === "boolean" ? "boolean" : typeof v === "number" ? "number" : "string";
                        return (
                          <div key={idx} className="flex gap-2 items-center mb-1.5">
                            <div className="flex-1">
                              <input
                                list={`suggestions-${cat}`}
                                value={k}
                                onChange={e => {
                                  const newKey = e.target.value;
                                  const newEff = { ...guEdit.effects };
                                  const catEff = { ...newEff[cat] };
                                  let currentVal = catEff[k];
                                  
                                  const suggestedType = getEffectFieldType(cat, newKey);
                                  if (suggestedType === "boolean" && typeof currentVal !== "boolean") {
                                    currentVal = false;
                                  } else if (suggestedType === "number" && typeof currentVal !== "number") {
                                    currentVal = 0;
                                  } else if (suggestedType === "string" && typeof currentVal !== "string") {
                                    currentVal = "";
                                  }

                                  delete catEff[k];
                                  catEff[newKey] = currentVal;
                                  newEff[cat] = catEff;
                                  setGuEdit({ ...guEdit, effects: newEff });
                                }}
                                className="w-full px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 font-mono"
                                placeholder="Key"
                              />
                            </div>
                            <div className="w-16">
                              <select
                                value={fieldType}
                                onChange={e => {
                                  const newType = e.target.value;
                                  const newEff = { ...guEdit.effects };
                                  const catEff = { ...newEff[cat] };
                                  if (newType === "boolean") {
                                    catEff[k] = false;
                                  } else if (newType === "number") {
                                    catEff[k] = 0;
                                  } else {
                                    catEff[k] = "";
                                  }
                                  newEff[cat] = catEff;
                                  setGuEdit({ ...guEdit, effects: newEff });
                                }}
                                className="w-full px-1 py-1 bg-zinc-800 border border-zinc-700 rounded text-[10px] text-zinc-300 font-medium focus:ring-0 focus:outline-none cursor-pointer"
                              >
                                <option value="number">Số</option>
                                <option value="string">Chữ</option>
                                <option value="boolean">Bool</option>
                              </select>
                            </div>
                            <div className="w-36">
                              {fieldType === "boolean" ? (
                                <label className="flex items-center gap-1.5 pt-1.5 cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={!!v}
                                    onChange={e => {
                                      const newEff = { ...guEdit.effects };
                                      const catEff = { ...newEff[cat] };
                                      catEff[k] = e.target.checked;
                                      newEff[cat] = catEff;
                                      setGuEdit({ ...guEdit, effects: newEff });
                                    }}
                                    className="rounded bg-zinc-800 border-zinc-700 w-3.5 h-3.5 text-amber-600"
                                  />
                                  <span className="text-[10px] text-zinc-400">Kích hoạt</span>
                                </label>
                              ) : fieldType === "number" ? (
                                <input
                                  type="number"
                                  step="any"
                                  value={v === undefined || v === null ? "" : v}
                                  onChange={e => {
                                    const val = parseFloat(e.target.value);
                                    const newEff = { ...guEdit.effects };
                                    const catEff = { ...newEff[cat] };
                                    catEff[k] = isNaN(val) ? 0 : val;
                                    newEff[cat] = catEff;
                                    setGuEdit({ ...guEdit, effects: newEff });
                                  }}
                                  className="w-full px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                  placeholder="Giá trị"
                                />
                              ) : (
                                <input
                                  type="text"
                                  value={v || ""}
                                  onChange={e => {
                                    const newEff = { ...guEdit.effects };
                                    const catEff = { ...newEff[cat] };
                                    catEff[k] = e.target.value;
                                    newEff[cat] = catEff;
                                    setGuEdit({ ...guEdit, effects: newEff });
                                  }}
                                  className="w-full px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                  placeholder="Giá trị"
                                />
                              )}
                            </div>
                            <button
                              onClick={() => {
                                const newEff = { ...guEdit.effects };
                                const catEff = { ...newEff[cat] };
                                delete catEff[k];
                                newEff[cat] = catEff;
                                setGuEdit({ ...guEdit, effects: newEff });
                              }}
                              className="text-red-400 hover:text-red-300 p-1 cursor-pointer"
                            >
                              <X className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        );
                      })}
                      <button
                        onClick={() => {
                          const newEff = { ...guEdit.effects || {} };
                          if (!newEff[cat]) newEff[cat] = {};
                          let keyName = "";
                          if (newEff[cat].hasOwnProperty("")) {
                            let counter = 1;
                            while (newEff[cat].hasOwnProperty(`new_effect_${counter}`)) {
                              counter++;
                            }
                            keyName = `new_effect_${counter}`;
                          }
                          newEff[cat] = { ...newEff[cat], [keyName]: "" };
                          setGuEdit({ ...guEdit, effects: newEff });
                        }}
                        className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1"
                      >
                        + Thêm hiệu ứng
                      </button>
                      <datalist id={`suggestions-${cat}`}>
                        {(EFFECT_SUGGESTIONS[cat] || []).map(item => (
                          <option key={item.key} value={item.key}>
                            {item.label}
                          </option>
                        ))}
                      </datalist>
                    </div>
                  </details>
                );
              })}
              {/* Other section - Special Mechanics */}
              <details className="mb-2 bg-zinc-800/30 rounded-lg">
                <summary className="px-3 py-2 text-xs font-bold text-zinc-400 uppercase cursor-pointer">other ({(guEdit.effects?.other || []).length} mechanics)</summary>
                <div className="px-3 pb-3">
                  {(guEdit.effects?.other || []).map((m: any, i: number) => (
                    <div key={i} className="grid grid-cols-3 gap-2 mb-2 p-2 bg-zinc-800/50 rounded">
                      <div>
                        <label className="text-[10px] text-zinc-500 block">Sự kiện kích hoạt</label>
                        <input value={m.event_trigger || ""} onChange={e => { const newEff = {...guEdit.effects}; const arr = [...(newEff.other || [])]; arr[i] = {...arr[i], event_trigger: e.target.value}; newEff.other = arr; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200" />
                      </div>
                      <div>
                        <label className="text-[10px] text-zinc-500 block">Mã hành động</label>
                        <input value={m.action_code || ""} onChange={e => { const newEff = {...guEdit.effects}; const arr = [...(newEff.other || [])]; arr[i] = {...arr[i], action_code: e.target.value}; newEff.other = arr; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200" />
                      </div>
                      <div>
                        <label className="text-[10px] text-zinc-500 block">Params (JSON)</label>
                        <textarea value={JSON.stringify(m.params || {})} onChange={e => { let p; try { p = JSON.parse(e.target.value) } catch { p = m.params } const newEff = {...guEdit.effects}; const arr = [...(newEff.other || [])]; arr[i] = {...arr[i], params: p}; newEff.other = arr; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200 font-mono" rows={2} />
                      </div>
                      <div className="col-span-3 flex justify-end">
                        <button onClick={() => { const newEff = {...guEdit.effects}; newEff.other = (newEff.other || []).filter((_: any, j: number) => j !== i); setGuEdit({...guEdit, effects: newEff}) }} className="text-[10px] text-red-400 hover:text-red-300">Xóa</button>
                      </div>
                    </div>
                  ))}
                  <button onClick={() => { const newEff = {...guEdit.effects}; newEff.other = [...(newEff.other || []), { event_trigger: "", action_code: "", params: {} }]; setGuEdit({...guEdit, effects: newEff}) }} className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 text-xs text-zinc-300 rounded">+ Thêm Mechanic</button>
                </div>
              </details>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setGuEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!guEdit._isNew;
                const { _isNew, ...body } = guEdit;
                if (body.effects) {
                  const cleaned = { ...body.effects };
                  ["active", "passive", "house", "growth"].forEach(cat => {
                    if (cleaned[cat]) {
                      cleaned[cat] = Object.fromEntries(
                        Object.entries(cleaned[cat]).filter(([k]) => k.trim() !== "")
                      );
                    }
                  });
                  body.effects = cleaned;
                }
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
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={itemEdit.id || ""} onChange={e => setItemEdit({...itemEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tên</label><input value={itemEdit.name || ""} onChange={e => setItemEdit({...itemEdit, name: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Loại</label><select value={itemEdit.item_type} onChange={e => setItemEdit({...itemEdit, item_type: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200"><option value="Material">Vật liệu</option><option value="Consumable">Tiêu hao</option><option value="Food">Thức ăn</option><option value="Currency">Tiền tệ</option></select></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Bậc</label><input type="number" min={1} value={itemEdit.rank} onChange={e => setItemEdit({...itemEdit, rank: parseInt(e.target.value) || 1})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Giá cơ bản</label><input type="number" value={itemEdit.base_price} onChange={e => setItemEdit({...itemEdit, base_price: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
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
                        } catch (err: any) {
                          if (addToast) addToast(err.message, "error");
                        } finally { setUploading(false); e.target.value = ""; }
                      }} />
                    </label>
                  </div>
                </div>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Mô tả</label><textarea value={itemEdit.description || ""} onChange={e => setItemEdit({...itemEdit, description: e.target.value})} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              {/* Effects key-value pairs */}
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Hiệu ứng — key: value</label>
                {Object.entries(itemEdit.effects || {}).map(([k, v], idx) => (
                  <div key={idx} className="flex gap-2 mb-1.5">
                    <input value={k} onChange={e => { const newEff = {...itemEdit.effects}; const val = newEff[k]; delete newEff[k]; newEff[e.target.value] = val; setItemEdit({...itemEdit, effects: newEff}) }} className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" placeholder="key" />
                    <input type="number" step="any" value={v as number} onChange={e => setItemEdit({...itemEdit, effects: {...itemEdit.effects, [k]: parseFloat(e.target.value) || 0}})} className="w-24 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" placeholder="value" />
                    <button onClick={() => { const newEff = {...itemEdit.effects}; delete newEff[k]; setItemEdit({...itemEdit, effects: newEff}) }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setItemEdit({...itemEdit, effects: {...itemEdit.effects, [""]: 0}})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm hiệu ứng</button>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setItemEdit(null)} className="px-4 py-2 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer">Hủy</button>
              <button onClick={async () => {
                const isNew = !!itemEdit._isNew;
                const itemUrl = isNew ? `${BACKEND_URL}/api/admin/item-dictionary` : `${BACKEND_URL}/api/admin/item-dictionary/${itemEdit.id}`;
                const clean = {...itemEdit, effects: Object.fromEntries(Object.entries(itemEdit.effects || {}).filter(([k]) => k.trim() !== ""))}; delete clean._isNew;
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
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">ID</label><input value={recipeEdit.id || ""} onChange={e => setRecipeEdit({...recipeEdit, id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Cổ trùng đích</label>
                  <select value={recipeEdit.target_gu_id} onChange={e => setRecipeEdit({...recipeEdit, target_gu_id: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200">
                    <option value="">Chọn...</option>
                    {gus.map((g: any) => <option key={g.id} value={g.id}>{g.name} ({g.id})</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tỉ lệ thành công</label><input type="number" step="0.01" min={0} max={1} value={recipeEdit.success_rate} onChange={e => setRecipeEdit({...recipeEdit, success_rate: parseFloat(e.target.value) || 0.5})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Nguyên Thạch</label><input type="number" value={recipeEdit.primeval_stones_cost} onChange={e => setRecipeEdit({...recipeEdit, primeval_stones_cost: parseInt(e.target.value) || 0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
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
              <datalist id="gu-list">{gus.map((g: any) => <option key={g.id} value={g.id} />)}</datalist>
              <datalist id="item-list">{items.map((it: any) => <option key={it.id} value={it.id} />)}</datalist>
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
                <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Tỉ lệ bắt</label><input type="number" step="0.01" min={0} max={1} value={monsterEdit.capture_chance || 0.20} onChange={e => setMonsterEdit({...monsterEdit, capture_chance: parseFloat(e.target.value) || 0.20})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              </div>
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Đói/giờ (pet)</label><input type="number" step="0.1" min={0} value={monsterEdit.pet_hunger_per_hour || 2.0} onChange={e => setMonsterEdit({...monsterEdit, pet_hunger_per_hour: parseFloat(e.target.value) || 2.0})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
              {/* JSONB fields as raw JSON */}
              <div className="border-t border-zinc-800 pt-3">
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">gu_inventory (JSON)</label>
                <textarea value={typeof monsterEdit.gu_inventory === "string" ? monsterEdit.gu_inventory : JSON.stringify(monsterEdit.gu_inventory || [], null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, gu_inventory: JSON.parse(e.target.value)}); } catch {} }} rows={3} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">item_inventory (JSON)</label>
                <textarea value={typeof monsterEdit.item_inventory === "string" ? monsterEdit.item_inventory : JSON.stringify(monsterEdit.item_inventory || {}, null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, item_inventory: JSON.parse(e.target.value)}); } catch {} }} rows={3} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
              </div>
              <div>
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">dao_marks (JSON)</label>
                <textarea value={typeof monsterEdit.dao_marks === "string" ? monsterEdit.dao_marks : JSON.stringify(monsterEdit.dao_marks || {}, null, 2)} onChange={e => { try { setMonsterEdit({...monsterEdit, dao_marks: JSON.parse(e.target.value)}); } catch {} }} rows={2} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono" />
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
              <datalist id="loot-table-list">{lootTables.map((lt: any) => <option key={lt.id} value={lt.id} />)}</datalist>
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
                {(Array.isArray(lootTableEdit.entries) ? lootTableEdit.entries : []).map((entry: any, idx: number) => (
                  <div key={idx} className="flex gap-2 mb-2 items-center">
                    <input value={entry.item_id || ""} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], item_id: e.target.value}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="item_id" className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="item-list-lt" />
                    <input value={entry.gu_id || ""} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], gu_id: e.target.value}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="gu_id" className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="gu-list-lt" />
                    <input type="number" min={1} value={entry.min_qty || 1} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], min_qty: parseInt(e.target.value) || 1}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="min" className="w-14 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <input type="number" min={1} value={entry.max_qty || 1} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], max_qty: parseInt(e.target.value) || 1}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="max" className="w-14 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <input type="number" step="0.01" min={0} max={1} value={entry.chance || 0.5} onChange={e => { const newEntries = [...lootTableEdit.entries]; newEntries[idx] = {...newEntries[idx], chance: parseFloat(e.target.value) || 0.5}; setLootTableEdit({...lootTableEdit, entries: newEntries}); }} placeholder="%" className="w-16 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <button onClick={() => { const newEntries = lootTableEdit.entries.filter((_: any, i: number) => i !== idx); setLootTableEdit({...lootTableEdit, entries: newEntries}); }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setLootTableEdit({...lootTableEdit, entries: [...(lootTableEdit.entries || []), { item_id: "", gu_id: "", min_qty: 1, max_qty: 1, chance: 0.5 }]})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm loot entry</button>
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
              <datalist id="item-list-lt">{items.map((it: any) => <option key={it.id} value={it.id} />)}</datalist>
              <datalist id="gu-list-lt">{gus.map((g: any) => <option key={g.id} value={g.id} />)}</datalist>
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
                if (!body.pool_weights) {
                  body.pool_weights = { cell_type_weights: {}, resources: [], wild_gus: [] };
                }
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
                    body.metadata = {};
                  }
                }
                const url = isNew ? `${BACKEND_URL}/api/admin/map-cells` : `${BACKEND_URL}/api/admin/map-cells/${mapCellEdit.id}`;
                const method = isNew ? "POST" : "PUT";
                const ok = await exec(url, method, body);
                if (ok) { setMapCellEdit(null); fetchMapCells(mapCellsPage, mapCellSearch, mapCellRegionFilter, mapCellTypeFilter); }
              }} className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold cursor-pointer">Lưu</button>
            </div>
          </div>
        </div>
      )}
      {renderItemPicker()}
    </div>
  );
};
