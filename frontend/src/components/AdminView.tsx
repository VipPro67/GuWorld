import React, { useEffect, useState, useCallback } from "react";
import { useGameStore, guSpecFor } from "@/store/useGameStore";
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
  FlaskConical,
  Activity,
  Gift,
  Ticket,
  Check,
  Clock,
  Copy,
  Upload,
} from "lucide-react";
import { translateGuType, translateItemType } from "@/lib/translate";

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
  is_admin: boolean; created_at: string; lifespan: number;
  soul_hp: number; max_soul: number; soul_tier: string;
  explore_count: number; aperture_stress: number;
  avatar_url: string; bio: string;
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
  const { player, setView, addToast, guDictionary, itemDictionary } = useGameStore();
  const [activeTab, setActiveTab] = useState<"items" | "gus">("items");
  const [players, setPlayers] = useState<PlayerRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [search, setSearch] = useState("");
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

  // Edit player modal
  const [showEditPlayer, setShowEditPlayer] = useState(false);
  const [editPlayerForm, setEditPlayerForm] = useState<Record<string, string>>({});

  // Global gu search (túi đồ / đấu giá / cửa hàng)
  const [guSearchId, setGuSearchId] = useState("");
  const [guSearchInstance, setGuSearchInstance] = useState("");
  const [guSearchResults, setGuSearchResults] = useState<any[]>([]);
  const [guSearchTotal, setGuSearchTotal] = useState(0);
  const [guSearchPage, setGuSearchPage] = useState(1);
  const [guSearchLimit] = useState(50);
  const [guSearchLoading, setGuSearchLoading] = useState(false);

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
  const [section, setSection] = useState<"players" | "sects" | "data" | "logs" | "shop" | "mines" | "gift_codes">("players");
  const [playersTab, setPlayersTab] = useState<"list" | "online" | "gusearch">("list");

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
  const [shopForm, setShopForm] = useState({ id: "", price: 100, stock: -1, is_default: true, sort_order: 0 });

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
  const [newGiftRewards, setNewGiftRewards] = useState<{type: string; id: string; amount: number; label: string}[]>([
    { type: "primeval_stones", id: "", amount: 1000, label: "1.000 Nguyên Thạch" },
  ]);

  // ----- Online players state -----
  const [onlinePlayers, setOnlinePlayers] = useState<any[]>([]);
  const [onlineLoading, setOnlineLoading] = useState(false);

  // ----- Dictionary state -----
  const [dataTab, setDataTab] = useState<"import" | "gus" | "items" | "recipes">("import");
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

  useEffect(() => { if (section === "data") { if (dataTab === "gus") fetchGus(); else if (dataTab === "items") fetchItems(); else if (dataTab === "recipes") fetchRecipes(); } }, [section, dataTab, fetchGus, fetchItems, fetchRecipes]);

  useEffect(() => {
    if (section === "players" && playersTab === "online") fetchOnlinePlayers();
  }, [section, playersTab, fetchOnlinePlayers]);

  const fetchGuSearch = useCallback(async () => {
    if (!guSearchId.trim() && !guSearchInstance.trim()) {
      setGuSearchResults([]);
      setGuSearchTotal(0);
      return;
    }
    setGuSearchLoading(true);
    try {
      const params = new URLSearchParams({ page: String(guSearchPage), limit: String(guSearchLimit) });
      if (guSearchId.trim()) params.set("gu_id", guSearchId.trim());
      if (guSearchInstance.trim()) params.set("instance_id", guSearchInstance.trim());
      const res = await authFetch(`${BACKEND_URL}/api/admin/gus/search?${params}`);
      if (res.ok) {
        const data = await res.json();
        setGuSearchResults(data.data || []);
        setGuSearchTotal(data.total || 0);
      }
    } catch {} finally { setGuSearchLoading(false); }
  }, [guSearchPage, guSearchLimit, guSearchId, guSearchInstance]);

  useEffect(() => {
    if (section === "players" && playersTab === "gusearch") fetchGuSearch();
  }, [section, playersTab, fetchGuSearch]);

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

  useEffect(() => {
    if (selectedId && activeTab === "items") fetchPlayerItems(selectedId);
  }, [selectedId, activeTab, playerItemsPage, playerItemsSearch, fetchPlayerItems]);

  useEffect(() => {
    if (selectedId && activeTab === "gus") fetchPlayerGus(selectedId);
  }, [selectedId, activeTab, playerGusPage, playerGusSearch, fetchPlayerGus]);

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
                <button key={x.id} onClick={() => {
                  const s = itemPicker.source;
                  if (s === "shop") setShopForm(f => ({ ...f, id: x.id }));
                  else if (s === "arena") setArenaShopForm(f => ({ ...f, id: x.id }));
                  else if (s === "add-item") setAddItemId(x.id);
                  else if (s === "add-gu") setAddGuId(x.id);
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
                  is_admin: String(detail.is_admin), soul_hp: String(detail.soul_hp ?? ""),
                  max_soul: String(detail.max_soul ?? ""), soul_tier: detail.soul_tier || "",
                  explore_count: String(detail.explore_count ?? ""), aperture_stress: String(detail.aperture_stress ?? ""),
                  bio: detail.bio || "", avatar_url: detail.avatar_url || "",
                });
                setShowEditPlayer(true);
              }
            }} disabled={!detail} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer disabled:opacity-50">
              <Edit3 className="w-3.5 h-3.5" />
              Sửa thông tin
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
            <div className="border-b border-zinc-800/80 flex gap-0">
              <button onClick={() => setActiveTab("items")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "items" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Package className="w-3.5 h-3.5 inline mr-1.5" />Vật phẩm ({detail.item_count || 0})
              </button>
              <button onClick={() => setActiveTab("gus")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${activeTab === "gus" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
                <Bug className="w-3.5 h-3.5 inline mr-1.5" />Cổ trùng ({detail.gu_count || 0})
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
                  <button onClick={() => { setAddGuId(""); setAddGuQty(1); setShowAddGu(true); }} className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">
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
        {showAddGu && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-zinc-100 font-serif">Thêm cổ trùng</h3>
                <button onClick={() => setShowAddGu(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn cổ trùng</label>
                <div className="flex gap-2 mt-1">
                  <input value={addGuId} onChange={e => setAddGuId(e.target.value)} placeholder="Nhập ID hoặc chọn..." className="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50" />
                  <button onClick={() => { ensureShopSearchLoaded(); setItemPicker({source: "add-gu", tab: "gu", query: ""}); }} className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-mono cursor-pointer flex items-center gap-1"><Search className="w-3 h-3" />Chọn</button>
                </div>
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
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tu vi (rank 1-9)</label>
                  <input type="number" min={1} max={9} value={editPlayerForm.rank || ""} onChange={e => setEditPlayerForm(f => ({ ...f, rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiểu cảnh giới (1-4)</label>
                  <input type="number" min={1} max={4} value={editPlayerForm.sub_rank || ""} onChange={e => setEditPlayerForm(f => ({ ...f, sub_rank: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Thiên phú (aptitude_type)</label>
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
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Máu hiện tại (hp)</label>
                  <input type="number" value={editPlayerForm.hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Máu tối đa (base_hp)</label>
                  <input type="number" value={editPlayerForm.base_hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, base_hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tuổi thọ (lifespan)</label>
                  <input type="number" value={editPlayerForm.lifespan || ""} onChange={e => setEditPlayerForm(f => ({ ...f, lifespan: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Khu vực hiện tại</label>
                  <input type="text" value={editPlayerForm.current_region || ""} onChange={e => setEditPlayerForm(f => ({ ...f, current_region: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tường đạo (wall_progress)</label>
                  <input type="number" value={editPlayerForm.wall_progress || ""} onChange={e => setEditPlayerForm(f => ({ ...f, wall_progress: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Nhân khí (human_qi)</label>
                  <input type="number" value={editPlayerForm.human_qi || ""} onChange={e => setEditPlayerForm(f => ({ ...f, human_qi: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hồn lực (soul_hp)</label>
                  <input type="number" value={editPlayerForm.soul_hp || ""} onChange={e => setEditPlayerForm(f => ({ ...f, soul_hp: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hồn lực tối đa (max_soul)</label>
                  <input type="number" value={editPlayerForm.max_soul || ""} onChange={e => setEditPlayerForm(f => ({ ...f, max_soul: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tầng hồn (soul_tier)</label>
                  <input type="text" value={editPlayerForm.soul_tier || ""} onChange={e => setEditPlayerForm(f => ({ ...f, soul_tier: e.target.value }))} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Số lần khai phá (explore_count)</label>
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
                <div className="flex items-end pb-1">
                  <label className="flex items-center gap-2 text-[11px] font-bold text-zinc-500 font-mono cursor-pointer">
                    <input type="checkbox" checked={editPlayerForm.is_admin === "true"} onChange={e => setEditPlayerForm(f => ({ ...f, is_admin: String(e.target.checked) }))} className="accent-amber-500 w-4 h-4" />
                    Quyền Admin
                  </label>
                </div>
              </div>
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Tiểu sử (bio)</label>
                <textarea value={editPlayerForm.bio || ""} onChange={e => setEditPlayerForm(f => ({ ...f, bio: e.target.value }))} rows={2} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50" />
              </div>
              <div className="flex gap-3 justify-end">
                <button onClick={() => setShowEditPlayer(false)} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer">Hủy</button>
                <button disabled={actionLoading} onClick={async () => {
                  const body: any = {};
                  const num = (v: string) => (v.trim() === "" ? undefined : Number(v));
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
                  set("is_admin", detail.is_admin, editPlayerForm.is_admin === "true");
                  set("soul_hp", detail.soul_hp, num(editPlayerForm.soul_hp));
                  set("max_soul", detail.max_soul, num(editPlayerForm.max_soul));
                  set("soul_tier", detail.soul_tier || "", editPlayerForm.soul_tier);
                  set("explore_count", detail.explore_count, num(editPlayerForm.explore_count));
                  set("aperture_stress", detail.aperture_stress, num(editPlayerForm.aperture_stress));
                  set("bio", detail.bio || "", editPlayerForm.bio);
                  set("avatar_url", detail.avatar_url || "", editPlayerForm.avatar_url);
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
        <button onClick={() => setSection("gift_codes")} className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${section === "gift_codes" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
          <Gift className="w-3.5 h-3.5 inline mr-1.5" />Mã Quà
        </button>
      </div>

      {/* Players section */}
      {section === "players" && (
        <>
          <div className="border-b border-zinc-800/80 flex gap-0 mb-4">
            <button onClick={() => setPlayersTab("list")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${playersTab === "list" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Users className="w-3.5 h-3.5 inline mr-1.5" />Danh sách
            </button>
            <button onClick={() => setPlayersTab("online")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${playersTab === "online" ? "text-emerald-400 border-emerald-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Activity className="w-3.5 h-3.5 inline mr-1.5" />Trực tuyến
            </button>
            <button onClick={() => setPlayersTab("gusearch")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${playersTab === "gusearch" ? "text-blue-400 border-blue-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Bug className="w-3.5 h-3.5 inline mr-1.5" />Tìm cổ trùng
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
                <input type="text" value={search} onChange={e => setSearch(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { setPage(1); fetchPlayers(); } }} placeholder={searchBy === "item" ? "ID hoặc tên vật phẩm..." : searchBy === "gu" ? "ID hoặc tên cổ trùng..." : "ID, tên đăng nhập hoặc email..."} className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 transition-colors placeholder:text-zinc-600" />
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

          {playersTab === "gusearch" && (
            <div className="space-y-4">
              <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
                <div className="flex flex-wrap gap-3 items-end">
                  <div className="flex-1 min-w-[200px] space-y-1">
                    <label className="text-[11px] font-bold text-zinc-500 uppercase tracking-wider font-mono flex items-center gap-1">
                      <Bug className="w-3 h-3" /> ID Cổ Trùng (gu_id)
                    </label>
                    <input type="text" value={guSearchId} onChange={e => setGuSearchId(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { setGuSearchPage(1); fetchGuSearch(); } }} placeholder="Ví dụ: GU_HUYEN_LONG..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-blue-500/50 transition-colors placeholder:text-zinc-600" />
                  </div>
                  <div className="flex-1 min-w-[200px] space-y-1">
                    <label className="text-[11px] font-bold text-zinc-500 uppercase tracking-wider font-mono flex items-center gap-1">
                      <Info className="w-3 h-3" /> Instance ID
                    </label>
                    <input type="text" value={guSearchInstance} onChange={e => setGuSearchInstance(e.target.value)} onKeyDown={e => { if (e.key === "Enter") { setGuSearchPage(1); fetchGuSearch(); } }} placeholder="UUID của cổ trùng..." className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-blue-500/50 transition-colors placeholder:text-zinc-600" />
                  </div>
                  <button onClick={() => { setGuSearchPage(1); fetchGuSearch(); }} className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-xs font-bold text-zinc-100 transition-colors cursor-pointer">Tìm</button>
                  <button onClick={fetchGuSearch} disabled={guSearchLoading} className="flex items-center gap-1.5 px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50 cursor-pointer">
                    <RefreshCw className={`w-3.5 h-3.5 ${guSearchLoading ? "animate-spin" : ""}`} />
                  </button>
                </div>
                <p className="text-[11px] text-zinc-500 mt-3 font-mono">Tìm kiếm trên 3 nguồn: Túi đồ người chơi, Phiên đấu giá, Cửa hàng (lữ hành & đấu trường).</p>
              </div>

              {!guSearchId.trim() && !guSearchInstance.trim() ? (
                <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
                  <Bug className="w-10 h-10 text-zinc-700 mx-auto" />
                  <p className="text-sm text-zinc-400 font-medium mt-2">Nhập ID hoặc Instance ID để tìm cổ trùng.</p>
                </div>
              ) : guSearchLoading ? (
                <div className="flex flex-col items-center justify-center py-20 space-y-3">
                  <RefreshCw className="w-8 h-8 text-blue-400 animate-spin" />
                  <span className="text-xs text-zinc-500 font-mono">Đang tìm cổ trùng...</span>
                </div>
              ) : guSearchResults.length === 0 ? (
                <div className="text-center py-16 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl space-y-2">
                  <Bug className="w-10 h-10 text-zinc-700 mx-auto" />
                  <p className="text-sm text-zinc-400 font-medium">Không tìm thấy cổ trùng nào.</p>
                </div>
              ) : (
                <>
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80">
                  <table className="w-full text-left text-xs">
                    <thead>
                      <tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px]">
                        <th className="px-3 py-2.5 font-semibold">Vị trí</th>
                        <th className="px-3 py-2.5 font-semibold">Tên</th>
                        <th className="px-3 py-2.5 font-semibold">gu_id</th>
                        <th className="px-3 py-2.5 font-semibold">Instance ID</th>
                        <th className="px-3 py-2.5 font-semibold">Bậc</th>
                        <th className="px-3 py-2.5 font-semibold">Trạng thái</th>
                        <th className="px-3 py-2.5 font-semibold">Đói</th>
                        <th className="px-3 py-2.5 font-semibold">Trang bị</th>
                        <th className="px-3 py-2.5 font-semibold">Bản Mệnh</th>
                        <th className="px-3 py-2.5 font-semibold">Chủ sở hữu</th>
                        <th className="px-3 py-2.5 font-semibold">Trạng thái phiên</th>
                      </tr>
                    </thead>
                    <tbody>
                      {guSearchResults.map((r: any, idx: number) => (
                        <tr key={idx} className={`border-t border-zinc-800/40 hover:bg-zinc-900/30 transition-colors ${r.location !== "shop" ? "cursor-pointer" : ""}`} onClick={() => { if (r.location !== "shop" && r.owner_id) setSelectedId(r.owner_id); }}>
                          <td className="px-3 py-2.5">
                            <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold border ${guLocationStyle(r.location)}`}>{guLocationLabel(r.location)}</span>
                          </td>
                          <td className="px-3 py-2.5 font-semibold text-zinc-200">{r.name || r.gu_id}</td>
                          <td className="px-3 py-2.5 text-zinc-400 font-mono">{r.gu_id}</td>
                          <td className="px-3 py-2.5 text-zinc-500 font-mono max-w-[80px] truncate" title={r.instance_id || ""}>{r.instance_id ? r.instance_id.slice(0, 8) + "..." : "—"}</td>
                          <td className={`px-3 py-2.5 font-bold ${rankColor(r.rank || 1)}`}>{r.rank || "?"}</td>
                          <td className="px-3 py-2.5">
                            <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold border ${statusColor(r.status)}`}>{r.status === "SHOP" ? "Cửa hàng" : statusLabel(r.status)}</span>
                          </td>
                          <td className="px-3 py-2.5 font-mono text-zinc-300">{r.hunger ?? "—"}</td>
                          <td className="px-3 py-2.5">{r.equipped ? <span className="text-emerald-400 text-[11px] font-bold">✓</span> : <span className="text-zinc-600">–</span>}</td>
                          <td className="px-3 py-2.5">{r.is_vital ? <Star className="w-3.5 h-3.5 text-amber-400" /> : <span className="text-zinc-600">–</span>}</td>
                          <td className="px-3 py-2.5">
                            {r.location === "shop" ? (
                              <span className="text-zinc-400">{r.shop_name || "Cửa hàng"}</span>
                            ) : r.owner_username ? (
                              <button onClick={() => setSelectedId(r.owner_id)} className="text-blue-400 hover:text-blue-300 underline cursor-pointer font-medium">{r.owner_username}</button>
                            ) : <span className="text-zinc-600">—</span>}
                          </td>
                          <td className="px-3 py-2.5 text-zinc-400">{r.listing_status || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <div className="flex items-center justify-between mt-4 text-xs font-semibold text-zinc-500">
                  <div>Tổng số: <span className="text-zinc-300">{guSearchTotal.toLocaleString()}</span> kết quả</div>
                  <div className="flex items-center gap-3">
                    <span>Trang {guSearchPage}/{Math.max(1, Math.ceil(guSearchTotal / guSearchLimit))}</span>
                    <div className="flex gap-2">
                      <button onClick={() => setGuSearchPage(Math.max(1, guSearchPage - 1))} disabled={guSearchPage <= 1 || guSearchLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        <ChevronLeft className="w-4 h-4" /> Trước
                      </button>
                      <button onClick={() => setGuSearchPage(guSearchPage + 1)} disabled={guSearchPage >= Math.ceil(guSearchTotal / guSearchLimit) || guSearchLoading} className="flex items-center gap-1 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-40 transition-colors cursor-pointer">
                        Sau <ChevronRight className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
                </>
              )}
            </div>
          )}
        </>
      )}

      {/* Data section: import/export + dictionary management */}
      {section === "data" && (
        <div className="space-y-4">
          <div className="border-b border-zinc-800/80 flex gap-0 mb-4">
            <button onClick={() => setDataTab("import")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${dataTab === "import" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Database className="w-3.5 h-3.5 inline mr-1.5" />Import / Export
            </button>
            <button onClick={() => setDataTab("gus")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${dataTab === "gus" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Bug className="w-3.5 h-3.5 inline mr-1.5" />Cổ trùng
            </button>
            <button onClick={() => setDataTab("items")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${dataTab === "items" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <Package className="w-3.5 h-3.5 inline mr-1.5" />Vật phẩm
            </button>
            <button onClick={() => setDataTab("recipes")} className={`px-4 py-2 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${dataTab === "recipes" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"}`}>
              <FlaskConical className="w-3.5 h-3.5 inline mr-1.5" />Cổ phương
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
              </div>
            </div>
          )}

          {dataTab === "gus" && (
            <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2"><Bug className="w-4 h-4 text-amber-400" />Danh Sách Cổ Trùng</h3>
                <button onClick={() => setGuEdit({ id: "", name: "", rank: 1, path: [], gu_type: "Attack", base_mana_cost: 0, effects: { active: {}, passive: {}, house: {}, other: [] }, food_types: [], hunger_rate: 1, max_hunger: 100, description: "", target_type: "SINGLE", rarity: "", is_immortal: false, primary_role: "", combat_roles: [], combat_tags: [], support_tags: [], base_price: 0, image_url: "", _isNew: true })} className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"><Plus className="w-3.5 h-3.5" />Thêm</button>
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
                            <button onClick={() => setGuEdit({ ...g, effects: typeof g.effects === "string" ? JSON.parse(g.effects) : g.effects })} className="text-amber-400 hover:text-amber-300 mr-2 cursor-pointer" title="Sửa"><Edit3 className="w-3.5 h-3.5 inline" /></button>
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
              {recipesLoading ? <div className="flex justify-center py-8"><RefreshCw className="w-6 h-6 text-amber-400 animate-spin" /></div> : recipes.length === 0 ? <div className="text-center py-8 text-zinc-500 text-xs">Không có cổ phương nào.</div> : (
                <div className="overflow-x-auto rounded-xl border border-zinc-800/80 max-h-[500px] overflow-y-auto">
                  <table className="w-full text-left text-xs">
                    <thead><tr className="bg-zinc-900/80 text-zinc-400 font-mono uppercase tracking-wider text-[10px] sticky top-0"><th className="px-3 py-2.5 font-semibold">ID</th><th className="px-3 py-2.5 font-semibold">Cổ trùng đích</th><th className="px-3 py-2.5 font-semibold text-right">Tỉ lệ</th><th className="px-3 py-2.5 font-semibold text-right">Nguyên Thạch</th><th className="px-3 py-2.5"></th></tr></thead>
                    <tbody>{recipes.map((r: any) => (
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
                          <td className="px-3 py-2.5 text-zinc-400">{translateItemType(item.item_type) || "—"}</td>
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
              <div><label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">Độ hiếm</label><input value={guEdit.rarity || ""} onChange={e => setGuEdit({...guEdit, rarity: e.target.value})} className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200" /></div>
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
              {["active", "passive", "house"].map(cat => {
                const eff = guEdit.effects?.[cat] || {};
                const activeEff = [
                  { key: "damage_flat", label: "Damage Flat", type: "float" },
                  { key: "damage_mult", label: "Damage Mult", type: "float" },
                  { key: "damage_path", label: "Damage Path", type: "string" },
                  { key: "heal_flat", label: "Heal Flat", type: "float" },
                  { key: "shield_flat", label: "Shield Flat", type: "float" },
                  { key: "mana_regen", label: "Mana Regen", type: "float" },
                  { key: "lifesteal_rate", label: "Life Steal", type: "float" },
                  { key: "apply_status", label: "Apply Status", type: "string" },
                  { key: "status_tick_damage", label: "Status Tick Dmg", type: "float" },
                  { key: "status_duration_ticks", label: "Status Duration Ticks", type: "float" },
                  { key: "refine_mana_tier", label: "Refine Mana Tier", type: "float" },
                  { key: "cleansing_power", label: "Cleansing Power", type: "float" },
                  { key: "range", label: "Range", type: "float" },
                  { key: "target_selector", label: "Target Selector", type: "string" },
                  { key: "target_count", label: "Target Count", type: "float" },
                  { key: "summon_monster_id", label: "Summon Monster ID", type: "string" },
                  { key: "summon_position", label: "Summon Position", type: "string" },
                  { key: "inherit_stats_percent", label: "Inherit Stats %", type: "float" },
                  { key: "phantom_damage_scale", label: "Phantom Damage Scale", type: "float" },
                  { key: "summon_count_per_tick", label: "Summon Count/Tick", type: "float" },
                ];
                const passiveEff = [
                  { key: "stat_base_attack", label: "Base Attack", type: "float" },
                  { key: "stat_base_defense", label: "Base Defense", type: "float" },
                  { key: "stat_base_hp", label: "Base HP", type: "float" },
                  { key: "stat_base_speed", label: "Base Speed", type: "float" },
                  { key: "mana_regen_mult", label: "Mana Regen Mult", type: "float" },
                  { key: "soul_refinement", label: "Soul Refinement", type: "float" },
                  { key: "movement", label: "Movement", type: "float" },
                  { key: "evasion", label: "Evasion", type: "float" },
                  { key: "storage_capacity", label: "Storage", type: "float" },
                  { key: "investigation", label: "Investigation", type: "float" },
                  { key: "hearing", label: "Hearing", type: "float" },
                  { key: "path_conflict_reduction", label: "Path Conflict Reduct", type: "float" },
                  { key: "damage_multiplier_on_hit", label: "Damage Mult On Hit", type: "float" },
                  { key: "refine_mana_tier", label: "Refine Mana Tier", type: "float" },
                  { key: "cleansing_power", label: "Cleansing Power", type: "float" },
                ];

                const boolKeys = cat === "active" ? ["summon_phantom", "generate_gu", "blood_drain", "wood_transformation", "curve_attack", "replicate", "summon_growth_phantoms"] : [];
                const fields = cat === "active" ? activeEff : cat === "passive" ? passiveEff : [];
                return (
                  <details key={cat} className="mb-2 bg-zinc-800/30 rounded-lg">
                    <summary className="px-3 py-2 text-xs font-bold text-zinc-400 uppercase cursor-pointer">{cat} ({Object.keys(eff).length} fields)</summary>
                    <div className="px-3 pb-3 grid grid-cols-3 gap-2">
                      {fields.map(f => (
                        <div key={f.key}>
                          <label className="text-[10px] text-zinc-500 block">{f.label}</label>
                          {f.type === "string" ? (
                            <input value={eff[f.key] || ""} onChange={e => { const newEff = {...guEdit.effects}; newEff[cat] = {...newEff[cat], [f.key]: e.target.value}; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200" />
                          ) : (
                            <input type="number" step="any" value={eff[f.key] || 0} onChange={e => { const v = parseFloat(e.target.value) || 0; const newEff = {...guEdit.effects}; newEff[cat] = {...newEff[cat], [f.key]: v}; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200" />
                          )}
                        </div>
                      ))}
                      {boolKeys.map(k => (
                        <label key={k} className="flex items-center gap-1.5 pt-4">
                          <input type="checkbox" checked={!!eff[k]} onChange={e => { const newEff = {...guEdit.effects}; newEff[cat] = {...newEff[cat], [k]: e.target.checked}; setGuEdit({...guEdit, effects: newEff}) }} className="rounded bg-zinc-800 border-zinc-700 w-3 h-3" />
                          <span className="text-[10px] text-zinc-400">{k}</span>
                        </label>
                      ))}
                    </div>
                  </details>
                );
              })}
              {/* Growth section - Permanent stat growth */}
              <details className="mb-2 bg-zinc-800/30 rounded-lg">
                <summary className="px-3 py-2 text-xs font-bold text-zinc-400 uppercase cursor-pointer">growth ({Object.keys(guEdit.effects?.growth || {}).length} fields)</summary>
                <div className="px-3 pb-3 grid grid-cols-3 gap-2">
                  {[
                    { key: "max_attack", label: "Max Attack", type: "float" },
                    { key: "max_defense", label: "Max Defense", type: "float" },
                    { key: "max_hp", label: "Max HP", type: "float" },
                    { key: "max_speed", label: "Max Speed", type: "float" },
                    { key: "hours_to_max", label: "Hours To Max", type: "float" },
                  ].map(f => {
                    const eff = guEdit.effects?.growth || {};
                    return (
                      <div key={f.key}>
                        <label className="text-[10px] text-zinc-500 block">{f.label}</label>
                        <input type="number" step="any" value={eff[f.key] || 0} onChange={e => { const v = parseFloat(e.target.value) || 0; const newEff = {...guEdit.effects}; newEff.growth = {...newEff.growth, [f.key]: v}; setGuEdit({...guEdit, effects: newEff}) }} className="w-full px-2 py-1 bg-zinc-700 border border-zinc-600 rounded text-xs text-zinc-200" />
                      </div>
                    );
                  })}
                </div>
              </details>
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
                const url = isNew ? `${BACKEND_URL}/api/admin/gu-dictionary` : `${BACKEND_URL}/api/admin/gu-dictionary/${guEdit.id}`;
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
                {Object.entries(itemEdit.effects || {}).map(([k, v]) => (
                  <div key={k} className="flex gap-2 mb-1.5">
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
                <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-2">Cổ trùng yêu cầu (required_gus): ID → số lượng</label>
                {Object.entries(recipeEdit.required_gus || {}).map(([k, v]) => (
                  <div key={k} className="flex gap-2 mb-1.5">
                    <input value={k} onChange={e => { const newG = {...recipeEdit.required_gus}; const val = newG[k]; delete newG[k]; newG[e.target.value] = val; setRecipeEdit({...recipeEdit, required_gus: newG}) }} className="flex-1 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" list="gu-list" />
                    <input type="number" min={1} value={v as number} onChange={e => setRecipeEdit({...recipeEdit, required_gus: {...recipeEdit.required_gus, [k]: parseInt(e.target.value) || 1}})} className="w-20 px-2 py-1.5 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200" />
                    <button onClick={() => { const newG = {...recipeEdit.required_gus}; delete newG[k]; setRecipeEdit({...recipeEdit, required_gus: newG}) }} className="text-red-400 hover:text-red-300 cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                  </div>
                ))}
                <button onClick={() => setRecipeEdit({...recipeEdit, required_gus: {...recipeEdit.required_gus, "": 1}})} className="text-xs text-amber-400 hover:text-amber-300 cursor-pointer mt-1">+ Thêm Gu</button>
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
      {renderItemPicker()}
    </div>
  );
};
