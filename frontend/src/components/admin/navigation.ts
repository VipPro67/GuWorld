import type { LucideIcon } from "lucide-react";
import {
  Activity,
  BookOpen,
  Building2,
  Coins,
  Database,
  Gift,
  Globe,
  History,
  Layers,
  Package,
  Users,
} from "lucide-react";

export const ADMIN_MODULE_KEYS = [
  "players", "sects", "data", "logs", "monitoring", "shop", "mines",
  "gift_codes", "apertures", "topup_wallet", "gu_history",
] as const;

export type AdminModuleKey = (typeof ADMIN_MODULE_KEYS)[number];

export type AdminNavigationItem = {
  key: AdminModuleKey;
  label: string;
  icon: LucideIcon;
  route: string;
  permission: string;
  group: "game" | "content" | "economy" | "system";
  order: number;
};

export const ADMIN_NAVIGATION: readonly AdminNavigationItem[] = [
  { key: "players", label: "Người chơi", icon: Users, route: "/admin/players", permission: "players.read", group: "game", order: 10 },
  { key: "sects", label: "Tông môn", icon: Building2, route: "/admin/sects", permission: "sects.read", group: "game", order: 20 },
  { key: "apertures", label: "Tiên Khiếu", icon: Globe, route: "/admin/apertures", permission: "apertures.read", group: "game", order: 30 },
  { key: "gu_history", label: "Lịch Sử Tiên Cổ", icon: History, route: "/admin/gu_history", permission: "gus.read", group: "game", order: 40 },
  { key: "data", label: "Dữ liệu", icon: Database, route: "/admin/data", permission: "game_data.read", group: "content", order: 50 },
  { key: "shop", label: "Cửa hàng", icon: Package, route: "/admin/shop", permission: "shop.read", group: "economy", order: 60 },
  { key: "mines", label: "Mỏ", icon: Layers, route: "/admin/mines", permission: "mine_slots.read", group: "economy", order: 70 },
  { key: "gift_codes", label: "Mã Quà", icon: Gift, route: "/admin/gift_codes", permission: "gift_codes.read", group: "economy", order: 80 },
  { key: "topup_wallet", label: "Nạp Tiền & Ví", icon: Coins, route: "/admin/topup_wallet", permission: "billing.read", group: "economy", order: 90 },
  { key: "logs", label: "Nhật ký", icon: BookOpen, route: "/admin/logs", permission: "audit.read", group: "system", order: 100 },
  { key: "monitoring", label: "Giám sát", icon: Activity, route: "/admin/monitoring", permission: "monitoring.read", group: "system", order: 110 },
];

export const ADMIN_GROUP_LABELS: Record<AdminNavigationItem["group"], string> = {
  game: "Trò chơi",
  content: "Nội dung",
  economy: "Kinh tế",
  system: "Hệ thống",
};

export function findAdminModule(key: string): AdminNavigationItem | undefined {
  return ADMIN_NAVIGATION.find((item) => item.key === key);
}
