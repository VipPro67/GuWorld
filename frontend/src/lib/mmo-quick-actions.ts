import {
  Compass,
  Sparkles,
  Gem,
  Crown,
  PawPrint,
  Package,
  Flame,
  ScrollText,
  Users,
  Store,
  ShoppingBag,
  Swords,
  Trophy,
  History,
  BookOpen,
  Settings2,
  type LucideIcon,
} from "lucide-react";
import { useState, useEffect } from "react";
import type { MMOPanel } from "@/components/game/MMOBusinessPanels";

export interface QuickActionDef {
  id: string;
  label: string;
  icon: LucideIcon;
  category: "Thế giới" | "Nhân vật" | "Giao thương" | "Hoạt động" | "Khác";
  panel?: Exclude<MMOPanel, null>;
  view?: string;
  special?: "divination";
  minRank?: number;
}

export const ALL_QUICK_ACTIONS: QuickActionDef[] = [
  // Thế giới
  { id: "maps", label: "Bản đồ", icon: Compass, category: "Thế giới", panel: "maps" },
  { id: "aperture", label: "Tiên khiếu", icon: Sparkles, category: "Thế giới", view: "aperture", minRank: 6 },
  { id: "dungeon", label: "Bí cảnh", icon: Gem, category: "Thế giới", view: "dungeon" },
  { id: "dream-realm", label: "Mộng cảnh", icon: Compass, category: "Thế giới", view: "dream-realm" },
  { id: "sect", label: "Tông môn", icon: Crown, category: "Thế giới", view: "sect" },
  { id: "pets", label: "Nô thú", icon: PawPrint, category: "Thế giới", view: "pets" },

  // Nhân vật
  { id: "character", label: "Nhân vật", icon: Crown, category: "Nhân vật", view: "character" },
  { id: "inventory", label: "Túi đồ", icon: Package, category: "Nhân vật", panel: "inventory" },
  { id: "cultivation", label: "Tu luyện", icon: Sparkles, category: "Nhân vật", panel: "cultivation" },
  { id: "refine", label: "Luyện Cổ", icon: Flame, category: "Nhân vật", view: "refine" },
  { id: "achievements", label: "Nhiệm vụ", icon: ScrollText, category: "Nhân vật", view: "achievements" },
  { id: "party", label: "Tổ đội", icon: Users, category: "Nhân vật", panel: "party" },

  // Giao thương
  { id: "shop", label: "Cửa hàng", icon: Store, category: "Giao thương", view: "shop" },
  { id: "auction", label: "Chợ Tiên", icon: ShoppingBag, category: "Giao thương", view: "auction" },
  { id: "premium-store", label: "Cát Bảo Các", icon: Gem, category: "Giao thương", view: "premium-store" },

  // Hoạt động
  { id: "arena", label: "Võ đài", icon: Swords, category: "Hoạt động", view: "arena" },
  { id: "death-arena", label: "Sinh tử đài", icon: Swords, category: "Hoạt động", view: "death-arena" },
  { id: "martial-arena", label: "Đại hội võ", icon: Trophy, category: "Hoạt động", view: "martial-arena" },
  { id: "practice-room", label: "Phòng tập", icon: Swords, category: "Hoạt động", view: "practice-room" },
  { id: "leaderboard", label: "Xếp hạng", icon: Trophy, category: "Hoạt động", view: "leaderboard" },
  { id: "gacha", label: "Giải trí", icon: Sparkles, category: "Hoạt động", view: "gacha" },

  // Khác
  { id: "divination", label: "Thiên Cơ Các", icon: Compass, category: "Khác", special: "divination" },
  { id: "combat-history", label: "LS Chiến đấu", icon: History, category: "Khác", view: "combat-history" },
  { id: "audit-logs", label: "Ký sự tu hành", icon: BookOpen, category: "Khác", view: "audit-logs" },
  { id: "feedback", label: "Góp ý", icon: ScrollText, category: "Khác", view: "feedback" },
  { id: "settings", label: "Cài đặt", icon: Settings2, category: "Khác", panel: "settings" },
];

export const QUICK_ACTIONS_MAP: Record<string, QuickActionDef> = Object.fromEntries(
  ALL_QUICK_ACTIONS.map((item) => [item.id, item]),
);

export const DEFAULT_QUICK_ACTION_IDS = ["maps", "inventory", "cultivation", "shop"];
export const MAX_QUICK_ACTIONS = 5;
export const MIN_QUICK_ACTIONS = 1;

const STORAGE_KEY = "gu_mmo_quick_actions";
const EVENT_NAME = "gu_mmo_quick_actions_updated";

export function getQuickActionIds(): string[] {
  if (typeof window === "undefined") return DEFAULT_QUICK_ACTION_IDS;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_QUICK_ACTION_IDS;
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed) && parsed.length > 0) {
      const valid = parsed.filter((id) => typeof id === "string" && QUICK_ACTIONS_MAP[id]);
      if (valid.length > 0) return valid.slice(0, MAX_QUICK_ACTIONS);
    }
  } catch {}
  return DEFAULT_QUICK_ACTION_IDS;
}

export function saveQuickActionIds(ids: string[]): void {
  if (typeof window === "undefined") return;
  try {
    const valid = ids.filter((id) => QUICK_ACTIONS_MAP[id]).slice(0, MAX_QUICK_ACTIONS);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(valid.length > 0 ? valid : DEFAULT_QUICK_ACTION_IDS));
    window.dispatchEvent(new CustomEvent(EVENT_NAME, { detail: valid }));
  } catch {}
}

export function resetQuickActionIds(): string[] {
  saveQuickActionIds(DEFAULT_QUICK_ACTION_IDS);
  return DEFAULT_QUICK_ACTION_IDS;
}

export function useQuickActionIds(): {
  ids: string[];
  setIds: (ids: string[]) => void;
  reset: () => void;
} {
  const [ids, setIdsState] = useState<string[]>(DEFAULT_QUICK_ACTION_IDS);

  useEffect(() => {
    setIdsState(getQuickActionIds());
    const handler = () => {
      setIdsState(getQuickActionIds());
    };
    window.addEventListener(EVENT_NAME, handler);
    window.addEventListener("storage", handler);
    return () => {
      window.removeEventListener(EVENT_NAME, handler);
      window.removeEventListener("storage", handler);
    };
  }, []);

  const setIds = (next: string[]) => {
    saveQuickActionIds(next);
    setIdsState(next);
  };

  const reset = () => {
    const def = resetQuickActionIds();
    setIdsState(def);
  };

  return { ids, setIds, reset };
}
