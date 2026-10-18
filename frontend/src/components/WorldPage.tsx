"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useWorldStore } from "@/store/useWorldStore";
import { guSpecFor, useGameStore } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";
import TeleportModal from "./TeleportModal";
import type { TeleportArray } from "@/store/slices/teleportSlice";
import { X, User, Briefcase, Flower2, Flame, Sparkles, Shield, Store, Trophy, Gift, Swords, Gem, Compass, Zap, Users, HelpCircle, Coins, ScrollText, MessageSquare, Settings, Map, Landmark, Eye, Terminal } from "lucide-react";
import { ClearCacheButton } from "./ClearCacheModal";
import { ThemeToggle } from "./ThemeToggle";
import { WorldAtlasModal } from "./WorldAtlasModal";
import { MovementJoystick } from "./MovementJoystick";
import { GuQuickBar } from "./GuQuickBar";
import { LogConsole } from "./LogConsole";
import { DirectActionHotbar, type HotbarAction } from "./DirectActionHotbar";
import { NotificationBell } from "./NotificationBell";
import { dispatchHotbarAction } from "@/lib/action-hotbar";
import { SpatialRealmHUD } from "./SpatialRealmHUD";
import { InheritanceAltarModal } from "./InheritanceAltarModal";
import { SpatialRealmsModal } from "./SpatialRealmsModal";

const WorldCanvas = dynamic(() => import("@/components/WorldCanvas"), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center text-zinc-500 font-serif">
      Đang tải thế giới...
    </div>
  ),
});

const GuInventory = dynamic(() => import("./GuInventory").then(m => m.GuInventory), { ssr: false });
const Aperture = dynamic(() => import("./Aperture").then(m => m.Aperture), { ssr: false });
const RefiningFurnace = dynamic(() => import("./RefiningFurnace").then(m => m.RefiningFurnace), { ssr: false });
const SectView = dynamic(() => import("./SectView").then(m => m.SectView), { ssr: false });
const BaoHoangMarket = dynamic(() => import("./BaoHoangMarket").then(m => m.BaoHoangMarket), { ssr: false });
const LeaderboardView = dynamic(() => import("./LeaderboardView").then(m => m.LeaderboardView), { ssr: false });
const EntertainmentPanel = dynamic(() => import("./EntertainmentPanel").then(m => m.EntertainmentPanel), { ssr: false });
const ArenaView = dynamic(() => import("./ArenaView").then(m => m.ArenaView), { ssr: false });
const PremiumStoreView = dynamic(() => import("./PremiumStoreView").then(m => m.PremiumStoreView), { ssr: false });
const DungeonView = dynamic(() => import("./DungeonView").then(m => m.DungeonView), { ssr: false });
const DreamRealmView = dynamic(() => import("./DreamRealmView").then(m => m.DreamRealmView), { ssr: false });
const ImmortalApertureView = dynamic(() => import("./ImmortalApertureView").then(m => m.ImmortalApertureView), { ssr: false });
const PartyView = dynamic(() => import("./PartyView").then(m => m.PartyView), { ssr: false });
const TargetHud = dynamic(() => import("./TargetHud").then(m => m.TargetHud), { ssr: false });
const TeleportBuildModal = dynamic(() => import("./Teleport/TeleportBuildModal").then(m => m.TeleportBuildModal), { ssr: false });
const TeleportDirectoryModal = dynamic(() => import("./Teleport/TeleportDirectoryModal").then(m => m.TeleportDirectoryModal), { ssr: false });
const TeleportManagementModal = dynamic(() => import("./Teleport/TeleportManagementModal").then(m => m.TeleportManagementModal), { ssr: false });

const TYPE_CONFIG: Record<string, { icon: string; label: string; color: string; borderColor: string; bgColor: string }> = {
  Boss:          { icon: "⚔️",  label: "Boss",             color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Monster:       { icon: "🐺",  label: "Yêu Thú",          color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Resource:      { icon: "💎",  label: "Linh Tài",         color: "text-emerald-300", borderColor: "border-emerald-500/30", bgColor: "bg-emerald-950/60" },
  WildGu:        { icon: "🐛",  label: "Cổ Trùng",         color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
  Empty:         { icon: "🏯",  label: "Địa Điểm",         color: "text-amber-300",   borderColor: "border-amber-500/30",   bgColor: "bg-amber-950/60" },
  Player:        { icon: "🥷",  label: "Cổ Sư",            color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
  GroundDrop:    { icon: "🎁",  label: "Chiến Lợi Phẩm",  color: "text-amber-300",   borderColor: "border-amber-500/40",   bgColor: "bg-amber-950/70" },
  Sect:          { icon: "🏯",  label: "Tông Môn",         color: "text-yellow-300",  borderColor: "border-yellow-500/50",  bgColor: "bg-yellow-950/70" },
  TeleportArray: { icon: "🌀",  label: "Truyền Tống Trận", color: "text-cyan-300",    borderColor: "border-cyan-500/50",    bgColor: "bg-cyan-950/70" },
  ENTRY_PORTAL:      { icon: "🚪",  label: "Cổng Bí Cảnh",      color: "text-amber-300",   borderColor: "border-amber-500/50",   bgColor: "bg-amber-950/70" },
  INHERITANCE_ALTAR: { icon: "🏛️",  label: "Đài Truyền Thừa",   color: "text-amber-200",   borderColor: "border-amber-400/50",   bgColor: "bg-amber-950/70" },
};
const DEFAULT_CONFIG = { icon: "📌", label: "Địa Điểm", color: "text-zinc-300", borderColor: "border-zinc-500/30", bgColor: "bg-zinc-800/60" };

type MapPerformanceMode = "full" | "light" | "minimal" | "off";
const MAP_PERFORMANCE_STORAGE_KEY = "gu_world_map_performance_mode";
const MAP_PERFORMANCE_OPTIONS: { id: MapPerformanceMode; label: string; description: string }[] = [
  { id: "full", label: "Đầy đủ", description: "Bản đồ, quái và người chơi cập nhật như hiện tại." },
  { id: "light", label: "Giảm nhẹ", description: "Giảm cập nhật người chơi và tải trước khu vực." },
  { id: "minimal", label: "Giảm nhiều", description: "Giữ bản đồ để di chuyển, giảm mạnh cập nhật dữ liệu nền." },
  { id: "off", label: "Tắt bản đồ", description: "Không dựng bản đồ và không gọi API nền của bản đồ." },
];

export function WorldPage() {
  const router = useRouter();
  const currentView = useUIStore((s) => s.currentView);
  const regions = useGameStore((s) => s.regions);
  const regionsLoaded = useGameStore((s) => s.regionsLoaded);
  const fetchRegions = useGameStore((s) => s.fetchRegions);
  const gamePlayer = useGameStore((s) => s.player);
  const logs = useGameStore((s) => s.logs);
  const openDivinationModal = useGameStore((s) => s.openDivinationModal);
  const error = useWorldStore((s) => s.error);
  const loading = useWorldStore((s) => s.loading);
  const selectedObject = useWorldStore((s) => s.selectedObject);
  const selectObject = useWorldStore((s) => s.selectObject);
  const isMoving = useWorldStore((s) => s.isMoving);
  const setMovementRange = useWorldStore((s) => s.setMovementRange);
  const liveX = useWorldStore((s) => s.liveX);
  const liveY = useWorldStore((s) => s.liveY);
  const fetchGrid = useGameStore((s) => s.fetchGrid);

  const monsterDictionary = useGameStore((s) => s.monsterDictionary);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const fetchPlayerProfile = useGameStore((s) => s.fetchPlayerProfile);
  const movePlayer = useGameStore((s) => s.movePlayer);

  const feedGu = useGameStore((s) => s.feedGu);
  const feedGuGroup = useGameStore((s) => s.feedGuGroup);
  const feedAllGu = useGameStore((s) => s.feedAllGu);
  const unequipAllGu = useGameStore((s) => s.unequipAllGu);
  const actionLoading = useGameStore((s) => s.actionLoading);
  const equipGu = useGameStore((s) => s.equipGu);
  const useItem = useGameStore((s) => s.useItem);
  const useItems = useGameStore((s) => s.useItems);
  const updateGuStatus = useGameStore((s) => s.updateGuStatus);
  const setVitalGu = useGameStore((s) => s.setVitalGu);
  const useGu = useGameStore((s) => s.useGu);
  const useSupportGu = useGameStore((s) => s.useSupportGu);
  const useGus = useGameStore((s) => s.useGus);
  const dropGu = useGameStore((s) => s.dropGu);
  const dropGus = useGameStore((s) => s.dropGus);
  const sectMap = useGameStore((s) => s.sectMap);
  const fetchSectsMap = useGameStore((s) => s.fetchSectsMap);
  const attackSect = useGameStore((s) => s.attackSect);
  const sectMembers = useGameStore((s) => s.sectMembers);
  const fetchSectMembers = useGameStore((s) => s.fetchSectMembers);
  const mySectStatus = useGameStore((s) => s.mySectStatus);
  const fetchMySectStatus = useGameStore((s) => s.fetchMySectStatus);
  const [attackTargetSect, setAttackTargetSect] = useState<ApiResponseData | null>(null);

  // RefiningFurnace
  const refineGu = useGameStore((s) => s.refineGu);

  // BaoHoangMarket
  const marketplaceListings = useGameStore((s) => s.marketplaceListings);
  const fetchMarketplaceListings = useGameStore((s) => s.fetchMarketplaceListings);
  const createMarketplaceListing = useGameStore((s) => s.createMarketplaceListing);
  const placeMarketplaceBid = useGameStore((s) => s.placeMarketplaceBid);
  const cancelMarketplaceListing = useGameStore((s) => s.cancelMarketplaceListing);
  const buyMarketplaceListing = useGameStore((s) => s.buyMarketplaceListing);

  // Party
  const fetchPartyStatus = useGameStore((s) => s.fetchPartyStatus);
  const fetchPartyInvitations = useGameStore((s) => s.fetchPartyInvitations);

  // Ground Drops
  const groundDrops = useGameStore((s) => s.groundDrops);
  const lootGroundDrop = useGameStore((s) => s.lootGroundDrop);

  // Teleportation Arrays
  const myTeleportArrays = useGameStore((s) => s.myTeleportArrays);
  const fetchMyTeleportArrays = useGameStore((s) => s.fetchMyTeleportArrays);
  const [managingTeleportArray, setManagingTeleportArray] = useState<TeleportArray | null>(null);

  const [showActivityLog, setShowActivityLog] = useState(false);
  const [activeModal, setActiveModal] = useState<string | null>(null);
  const [showGeneralSettings, setShowGeneralSettings] = useState(false);
  const [showWorldAtlas, setShowWorldAtlas] = useState(false);
  const [showSpatialRealms, setShowSpatialRealms] = useState(false);
  const [mapPerformanceMode, setMapPerformanceMode] = useState<MapPerformanceMode>(() => {
    if (typeof window === "undefined") return "full";
    const saved = localStorage.getItem(MAP_PERFORMANCE_STORAGE_KEY);
    return saved === "light" || saved === "minimal" || saved === "off" ? saved : "full";
  });
  interface MapLabelSettings {
    monster: boolean;
    bossNearby: boolean;
    hub: boolean;
    khu_vuc: boolean;
    ngu_vuc: boolean;
  }

  const [labelSettings, setLabelSettings] = useState<MapLabelSettings>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("gu_world_map_label_settings");
      if (saved) {
        try {
          const parsed = JSON.parse(saved);
          return {
            monster: !!parsed.monster,
            bossNearby: !!parsed.bossNearby,
            hub: parsed.hub !== false,
            khu_vuc: parsed.khu_vuc !== false,
            ngu_vuc: parsed.ngu_vuc !== false,
          };
        } catch (e) {}
      }
    }
    return { monster: false, bossNearby: false, hub: true, khu_vuc: true, ngu_vuc: true };
  });

  const updateLabelSetting = (key: keyof MapLabelSettings, val: boolean) => {
    setLabelSettings((prev) => {
      const next = { ...prev, [key]: val };
      localStorage.setItem("gu_world_map_label_settings", JSON.stringify(next));
      return next;
    });
  };
  const updateMapPerformanceMode = (mode: MapPerformanceMode) => {
    setMapPerformanceMode(mode);
    localStorage.setItem(MAP_PERFORMANCE_STORAGE_KEY, mode);
  };
  const setView = useGameStore((s) => s.setView);
  const logout = useGameStore((s) => s.logout);
  const showWorldActions = currentView === "dashboard" || currentView === "world-map";

  useEffect(() => {
    if (!showWorldActions) {
      setShowActivityLog(false);
      setShowGeneralSettings(false);
      setShowWorldAtlas(false);
    }
  }, [showWorldActions]);
  const isImmortal = gamePlayer ? gamePlayer.rank >= 6 : false;

  const menuItems = [
    { id: "character", label: "Nhân Vật", icon: User, color: "text-emerald-400" },
    { id: "inventory", label: "Túi Đồ", icon: Briefcase, color: "text-cyan-400" },
    { id: "inventory-gu", label: "Cổ Trùng", icon: Flower2, color: "text-purple-400" },
    { id: "aperture", label: "Không Khiếu", icon: Flame, color: "text-rose-400" },
    { id: "refine", label: "Luyện Cổ", icon: Sparkles, color: "text-purple-400" },
    { id: "cultivation", label: "Tu Luyện", icon: Zap, color: "text-violet-400" },
    { id: "practice-room", label: "Phòng Tập", icon: Swords, color: "text-indigo-400" },
    { id: "pvp-arena", label: "Đấu Trường Realtime", icon: Swords, color: "text-emerald-300" },
    { id: "spatial-realms", label: "Bí Cảnh", icon: Landmark, color: "text-amber-400" },
    { id: "dream-realm", label: "Mộng Cảnh", icon: Compass, color: "text-purple-400" },
    { id: "party", label: "Tổ Đội", icon: Users, color: "text-amber-400" },
    { id: "sect", label: "Tông Môn", icon: Shield, color: "text-blue-400" },
    { id: "auction", label: "Chợ Tiên", icon: Store, color: "text-amber-400", locked: !isImmortal },
    { id: "shop", label: "Thương Hội", icon: Gem, color: "text-sky-400" },
    { id: "pets", label: "Dị Nhân", icon: Gift, color: "text-pink-400" },
    { id: "arena", label: "Đấu Trường", icon: Swords, color: "text-red-400" },
    { id: "death-arena", label: "Sinh Tử Đài", icon: Swords, color: "text-rose-400" },
    { id: "martial-arena", label: "Đại Hội Võ Thuật", icon: Trophy, color: "text-orange-400" },
    { id: "leaderboard", label: "Xếp Hạng", icon: Trophy, color: "text-yellow-400" },
    { id: "gacha", label: "Tiên Duyên", icon: Gift, color: "text-fuchsia-400" },
    { id: "achievements", label: "Thành Tựu", icon: Trophy, color: "text-lime-400" },
    { id: "mine", label: "Khai Khoáng", icon: Gem, color: "text-orange-400" },
    { id: "premium-store", label: "Cửa Hàng", icon: Store, color: "text-sky-400" },
    { id: "topup", label: "Nạp Tiên Ngọc", icon: Coins, color: "text-amber-300" },
    { id: "feedback", label: "Góp Ý", icon: MessageSquare, color: "text-cyan-300" },
    { id: "combat-history", label: "Chiến Sử", icon: Swords, color: "text-red-300" },
    { id: "audit-logs", label: "Ký Sự Tu Hành", icon: ScrollText, color: "text-blue-300" },
    { id: "general-settings", label: "Cài Đặt Chung", icon: Settings, color: "text-zinc-300" },
    { id: "utilities", label: "Tiện Ích", icon: Settings, color: "text-zinc-300" },
    { id: "wiki", label: "Wiki", icon: HelpCircle, color: "text-teal-400" },
    { id: "teleport", label: "Dịch Chuyển", icon: Compass, color: "text-cyan-400" },
    { id: "world-atlas", label: "Bản Đồ Thế Giới", icon: Map, color: "text-emerald-400" },
    { id: "teleport-directory", label: "Danh Bạ Truyền Tống Trận", icon: Compass, color: "text-cyan-300" },
    { id: "teleport-build", label: "Bố Trí Truyền Tống Trận", icon: Landmark, color: "text-blue-300" },
    { id: "divination", label: "Thiên Cơ Các", icon: Eye, color: "text-purple-300" },
    { id: "activity-log", label: "Nhật Ký Hành Động", icon: Terminal, color: "text-emerald-300" },
  ];

  const menuGroups = [
    { id: "character", label: "Nhân vật", icon: User, itemIds: ["character", "inventory", "inventory-gu", "aperture", "refine", "cultivation", "practice-room"] },
    { id: "world", label: "Khám phá", icon: Compass, itemIds: ["spatial-realms", "dream-realm", "dungeon", "mine", "teleport", "party", "sect", "world-atlas", "teleport-directory", "teleport-build", "divination"] },
    { id: "trade", label: "Giao dịch", icon: Store, itemIds: ["auction", "shop", "premium-store", "topup"] },
    { id: "activities", label: "Hoạt động", icon: Swords, itemIds: ["arena", "death-arena", "martial-arena", "pvp-arena", "leaderboard", "gacha", "achievements", "pets"] },
    { id: "other", label: "Khác", icon: Settings, itemIds: ["feedback", "combat-history", "audit-logs", "activity-log", "general-settings", "utilities", "wiki"] },
  ].map((group) => ({ ...group, items: menuItems.filter((item) => group.itemIds.includes(item.id)) }));

  const hotbarActions: HotbarAction[] = menuItems.map((item) => ({
    ...item,
    group: menuGroups.find((group) => group.itemIds.includes(item.id))?.label ?? "Khác",
  }));

  const openHotbarAction = (actionId: string) => {
    if (actionId === "wiki") {
      router.push("/wiki");
      return;
    }
    if (actionId === "spatial-realms") {
      setShowSpatialRealms(true);
      return;
    }
    dispatchHotbarAction(actionId, {
      setView,
      openGeneralSettings: () => setShowGeneralSettings(true),
      openTeleport: () => setActiveModal("teleport"),
      openWorldAtlas: () => setShowWorldAtlas(true),
      openTeleportDirectory: () => setActiveModal("teleport_directory"),
      openTeleportBuild: () => setActiveModal("teleport_build"),
      openDivination: openDivinationModal,
      openActivityLog: () => setShowActivityLog(true),
    });
  };

  useEffect(() => {
    if (mapPerformanceMode !== "off" && !regionsLoaded) fetchRegions();
  }, [mapPerformanceMode, regionsLoaded, fetchRegions]);

  useEffect(() => {
    fetchPartyStatus();
    fetchPartyInvitations();
    fetchMySectStatus();
    if (mapPerformanceMode !== "off") {
      fetchSectsMap();
      fetchMyTeleportArrays();
    }
  }, [mapPerformanceMode, fetchPartyStatus, fetchPartyInvitations, fetchSectsMap, fetchMyTeleportArrays, fetchMySectStatus]);

  const setInvestigationRange = useWorldStore((s) => s.setInvestigationRange);

  // Movement and Investigation ranges come from the same stats as the grid viewport.
  useEffect(() => {
    if (mapPerformanceMode === "off") return;
    fetchGrid(true).then((data) => {
      if (data?.stats) {
        if (data.stats.movement) setMovementRange(data.stats.movement);
        if (data.stats.investigation) setInvestigationRange(data.stats.investigation);
      }
    });
  }, [mapPerformanceMode, fetchGrid, setMovementRange, setInvestigationRange]);

  const currentZone = gamePlayer?.current_region
    ? regions.find((r) => r.id === gamePlayer.current_region)
    : undefined;

  const cfg = selectedObject ? TYPE_CONFIG[selectedObject.cell_type] || DEFAULT_CONFIG : DEFAULT_CONFIG;

  return (
    <div className="w-full h-full relative flex flex-col min-h-0 overflow-hidden">
      {loading && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-zinc-900/80 rounded-xl border border-zinc-800 text-xs text-zinc-400 font-serif z-20">
          Đang khám phá thế giới...
        </div>
      )}
      {error && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-red-950/80 rounded-xl border border-red-800/40 text-xs text-red-400 font-serif z-20">
          {error}
        </div>
      )}

      <div className="w-full h-full flex-1 min-h-0 relative">
        {mapPerformanceMode === "off" ? (
          <div className="flex h-full items-center justify-center bg-zinc-950/80 px-6 text-center">
            <div className="max-w-sm rounded-2xl border border-zinc-800 bg-zinc-900/80 p-5 shadow-xl">
              <Map className="mx-auto mb-3 h-7 w-7 text-amber-300" />
              <p className="font-serif text-sm text-zinc-200">Bản đồ đang tắt</p>
              <p className="mt-1 text-xs text-zinc-500">Bật lại trong Cài Đặt Chung. Các chức năng khác vẫn dùng được qua thanh thao tác.</p>
            </div>
          </div>
        ) : (
          <WorldCanvas mode="select" labelSettings={labelSettings} performanceMode={mapPerformanceMode} />
        )}
      </div>

      {mapPerformanceMode !== "off" && <TargetHud />}

      {showWorldActions && (
        <div className="absolute right-2 top-1/2 z-[60] -translate-y-1/2 pointer-events-auto sm:bottom-[calc(env(safe-area-inset-bottom)+6rem)] sm:left-1/2 sm:right-auto sm:top-auto sm:-translate-x-1/2 sm:translate-y-0">
          <DirectActionHotbar actions={hotbarActions} onAction={openHotbarAction} />
        </div>
      )}

      {showActivityLog && (
        <div className="fixed inset-0 z-[80] h-[100dvh] flex items-center justify-center bg-black/70 p-3 pt-[calc(env(safe-area-inset-top)+0.75rem)] pb-[calc(env(safe-area-inset-bottom)+0.75rem)] backdrop-blur-sm" onClick={() => setShowActivityLog(false)}>
          <div className="relative w-full max-w-xl" onClick={(event) => event.stopPropagation()}>
            <button onClick={() => setShowActivityLog(false)} aria-label="Đóng nhật ký" className="absolute right-2 top-2 z-10 grid h-8 w-8 place-items-center rounded-lg border border-white/10 bg-zinc-950 text-zinc-400 hover:text-white">
              <X className="h-4 w-4" />
            </button>
            <LogConsole logs={logs} />
          </div>
        </div>
      )}

      {activeModal && (
        <div className="fixed inset-0 z-40 h-[100dvh] bg-black/70 flex items-center justify-center px-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] sm:px-6 sm:py-6 backdrop-blur-sm overflow-hidden pointer-events-auto">
          <div className="relative w-full h-full max-w-7xl max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] bg-zinc-950/95 rounded-2xl border border-zinc-800 shadow-2xl flex flex-col overflow-hidden">
            {activeModal === "inventory" && (
              <div className="flex min-h-14 shrink-0 items-center justify-between gap-3 border-b border-white/10 bg-black/20 px-3 sm:px-5">
                <div className="truncate font-serif text-base font-semibold text-emerald-50">Túi đồ</div>
                <div className="flex shrink-0 items-center gap-2">
                  <NotificationBell />
                  <button
                    type="button"
                    onClick={() => setActiveModal(null)}
                    aria-label="Đóng túi đồ"
                    className="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-white/10 bg-white/5 text-zinc-300 transition hover:bg-white/10 hover:text-white"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )}
            {activeModal !== "inventory" && (
              <button
                onClick={() => setActiveModal(null)}
                className="absolute top-4 right-6 z-50 p-2 rounded-xl bg-zinc-800/80 hover:bg-zinc-700 text-zinc-300 hover:text-white transition-colors border border-zinc-600 cursor-pointer shadow-lg"
                title="Đóng"
              >
                <X className="w-6 h-6" />
              </button>
            )}
            <div className="flex-1 min-h-0 w-full p-2 sm:p-4 pb-28 sm:pb-8 overflow-y-auto relative custom-scrollbar">
              {activeModal === "character" && <Aperture player={gamePlayer!} guDictionary={guDictionary} />}
              {activeModal === "inventory" && (
                <GuInventory
                  player={gamePlayer!}
                  onFeed={feedGu}
                  onFeedGroup={feedGuGroup}
                  onFeedAll={feedAllGu}
                  onUnequipAll={unequipAllGu}
                  actionLoading={actionLoading}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                  onEquipToggle={equipGu}
                  onUseItem={useItem}
                  onUseItems={useItems}
                  onUpdateStatus={updateGuStatus}
                  onSetVital={setVitalGu}
                  onUseGu={useGu}
                  onUseSupportGu={useSupportGu}
                  onUseGus={useGus}
                  onDropGu={dropGu}
                  onDropGus={dropGus}
                />
              )}
              {activeModal === "aperture" && <ImmortalApertureView />}
              {activeModal === "refine" && (
                <RefiningFurnace
                  player={gamePlayer!}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                  actionLoading={actionLoading}
                  onRefine={refineGu}
                />
              )}
              {activeModal === "party" && <PartyView />}
              {activeModal === "sect" && <SectView />}
              {activeModal === "market" && (
                <BaoHoangMarket
                  player={gamePlayer!}
                  guInventory={gamePlayer!.gu_inventory || []}
                  materials={gamePlayer!.materials_inventory || {}}
                  actionLoading={actionLoading}
                  listings={marketplaceListings}
                  onRefresh={fetchMarketplaceListings}
                  onCreate={createMarketplaceListing}
                  onBid={placeMarketplaceBid}
                  onCancel={cancelMarketplaceListing}
                  onBuy={buyMarketplaceListing}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                />
              )}
              {activeModal === "leaderboard" && <LeaderboardView />}
              {activeModal === "gacha" && <EntertainmentPanel />}
              {activeModal === "arena" && <ArenaView />}
              {activeModal === "premium_store" && <PremiumStoreView />}
              {(activeModal === "dungeon" || activeModal === "dream-realm") && <DreamRealmView />}
              {activeModal === "teleport_directory" && gamePlayer && (
                <TeleportDirectoryModal
                  player={gamePlayer}
                  onClose={() => setActiveModal(null)}
                  onOpenManage={(arrayId) => {
                    const found = myTeleportArrays.find((a: TeleportArray) => a.id === arrayId);
                    if (found) {
                      setActiveModal(null);
                      setManagingTeleportArray(found);
                    }
                  }}
                />
              )}
              {activeModal === "teleport_build" && gamePlayer && (
                <TeleportBuildModal
                  player={gamePlayer}
                  onClose={() => setActiveModal(null)}
                  onSuccess={() => {
                    setActiveModal(null);
                    fetchMyTeleportArrays();
                    if (mapPerformanceMode !== "off") fetchGrid(true);
                  }}
                />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Gu Teleport Modal */}
      {activeModal === "teleport" && (
        <TeleportModal isOpen={true} onClose={() => setActiveModal(null)} />
      )}

      {/* Standalone Teleport Management Modal */}
      {managingTeleportArray && gamePlayer && (
        <TeleportManagementModal
          player={gamePlayer}
          array={managingTeleportArray}
          onClose={() => setManagingTeleportArray(null)}
          onUpdated={() => {
            fetchMyTeleportArrays();
            if (mapPerformanceMode !== "off") fetchGrid(true);
          }}
        />
      )}

      <div className="absolute top-2 left-2 flex flex-col gap-1.5 pointer-events-none z-10">
        {gamePlayer && (
          <div className="px-0.5 py-0.5 flex flex-col items-start text-xs text-zinc-200 font-serif pointer-events-auto">
            <div className="w-24 sm:w-28 flex flex-col gap-0.5">
              <div className="relative w-full h-2 sm:h-2.5 bg-zinc-950/70 rounded-sm overflow-hidden shadow-sm" title={`Máu: ${Math.round(gamePlayer.hp)}`}>
                <div className="h-full bg-red-500 transition-all duration-300" style={{ width: `${Math.min(100, (gamePlayer.hp / (gamePlayer.base_hp || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[8px] sm:text-[9px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.hp).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-full h-2 sm:h-2.5 bg-zinc-950/70 rounded-sm overflow-hidden shadow-sm" title={`Chân Nguyên: ${Math.floor(gamePlayer.current_mana)}`}>
                <div className="h-full bg-sky-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.current_mana / (gamePlayer.max_mana || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[8px] sm:text-[9px] font-bold text-white drop-shadow-md">
                  {Math.floor(gamePlayer.current_mana).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-full h-2 sm:h-2.5 bg-zinc-950/70 rounded-sm overflow-hidden shadow-sm" title={`Linh Hồn: ${Math.round(gamePlayer.soul_hp)}`}>
                <div className="h-full bg-purple-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.soul_hp / (gamePlayer.max_soul || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[8px] sm:text-[9px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.soul_hp).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-full h-2 sm:h-2.5 bg-zinc-950/70 rounded-sm overflow-hidden shadow-sm" title={`Tinh Thần: ${Math.round(gamePlayer.current_thought)}`}>
                <div className="h-full bg-amber-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.current_thought / (gamePlayer.max_thought || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[8px] sm:text-[9px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.current_thought).toLocaleString('vi-VN')}
                </span>
              </div>
            </div>
            <div className="flex flex-col items-start justify-center mt-0.5 sm:mt-1">
              <div className="max-w-32 sm:max-w-40 truncate text-emerald-300 font-bold text-[11px] sm:text-xs drop-shadow-md" title={currentZone?.name || gamePlayer.current_region || "Ngoài bản đồ"}>
                <span>{currentZone?.name || gamePlayer.current_region || "Ngoài bản đồ"}</span>
              </div>
              <div className="mt-0.5">
                {currentZone?.zone_type === "HUB" || currentZone?.zone_type === "SAFE_ZONE" ? (
                  <span className="px-1 sm:px-1.5 py-0.5 text-[8.5px] sm:text-[9px] font-mono font-semibold rounded bg-emerald-950/85 text-emerald-300 shadow-sm" title="Vùng An Toàn: Nghiêm cấm đồ sát / Không thể PvP">
                    🛡️ An toàn
                  </span>
                ) : currentZone?.zone_type === "PVP_ZONE" ? (
                  <span className="px-1 sm:px-1.5 py-0.5 text-[8.5px] sm:text-[9px] font-mono font-semibold rounded bg-red-950/85 text-red-300 shadow-sm" title="Vùng PvP Nguy Hiểm: Tự do đồ sát, nguy cơ rơi đồ cao!">
                    ⚔️ PvP nguy hiểm
                  </span>
                ) : (
                  <span className="px-1 sm:px-1.5 py-0.5 text-[8.5px] sm:text-[9px] font-mono font-semibold rounded bg-amber-950/85 text-amber-200 shadow-sm" title="Vùng Dã Ngoại: Cho phép giao tranh PvP">
                    ⚠️ PvP mở
                  </span>
                )}
              </div>
            </div>
            <div className="text-[8.5px] sm:text-[9px] text-zinc-200 font-mono mt-0.5 drop-shadow-md">
              X:{Math.round(liveX || gamePlayer.x)} Y:{Math.round(liveY || gamePlayer.y)}
            </div>
          </div>
        )}

      </div>

      {showWorldAtlas && gamePlayer && (
        <WorldAtlasModal
          regions={regions}
          currentRegionId={gamePlayer.current_region}
          x={liveX || gamePlayer.x}
          y={liveY || gamePlayer.y}
          onClose={() => setShowWorldAtlas(false)}
        />
      )}

      {showGeneralSettings && (
        <div
          className="fixed inset-0 z-[90] h-[100dvh] flex items-center justify-center bg-black/70 p-4 pt-[calc(env(safe-area-inset-top)+1rem)] pb-[calc(env(safe-area-inset-bottom)+1rem)] backdrop-blur-sm"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setShowGeneralSettings(false);
          }}
        >
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="general-settings-title"
            className="w-full max-w-md max-h-[90dvh] overflow-y-auto rounded-2xl border border-zinc-700 bg-zinc-950 p-4 text-zinc-100 shadow-2xl"
          >
            <div className="mb-4 flex items-center justify-between border-b border-zinc-800 pb-3">
              <div>
                <h2 id="general-settings-title" className="font-serif text-base font-semibold">Cài Đặt Chung</h2>
                <p className="mt-0.5 text-[11px] text-zinc-500">Giao diện, bản đồ và tài khoản</p>
              </div>
              <button
                onClick={() => setShowGeneralSettings(false)}
                aria-label="Đóng cài đặt"
                className="grid h-8 w-8 place-items-center rounded-lg text-zinc-400 hover:bg-white/10 hover:text-white"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-4">
              <section className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
                <h3 className="mb-2 text-xs font-semibold text-zinc-200">Giao diện</h3>
                <div className="flex items-center justify-between gap-3 text-xs text-zinc-400">
                  <span>Chế độ sáng / tối</span>
                  <ThemeToggle />
                </div>
              </section>

              <section className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
                <h3 className="mb-2 text-xs font-semibold text-zinc-200">Hiển thị trên bản đồ</h3>
                <div className="mb-3 space-y-1.5">
                  <label htmlFor="map-performance-mode" className="text-xs text-zinc-300">Mức hiển thị bản đồ</label>
                  <select
                    id="map-performance-mode"
                    value={mapPerformanceMode}
                    onChange={(event) => updateMapPerformanceMode(event.target.value as MapPerformanceMode)}
                    className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 outline-none focus:border-purple-500"
                  >
                    {MAP_PERFORMANCE_OPTIONS.map((option) => <option key={option.id} value={option.id}>{option.label}</option>)}
                  </select>
                  <p className="text-[11px] leading-relaxed text-zinc-500">
                    {MAP_PERFORMANCE_OPTIONS.find((option) => option.id === mapPerformanceMode)?.description}
                  </p>
                </div>
                <div className="flex flex-col gap-2">
                  {([
                    ["monster", "Quái & Cổ trùng"],
                    ["bossNearby", "Boss lân cận"],
                    ["hub", "Điểm tập trung (Hub)"],
                    ["khu_vuc", "Khu vực"],
                    ["ngu_vuc", "Ngự vực"],
                  ] as const).map(([key, label]) => (
                    <label key={key} className="flex cursor-pointer items-center gap-2 text-xs text-zinc-300 hover:text-white">
                      <input
                        type="checkbox"
                        checked={labelSettings[key]}
                        onChange={(event) => updateLabelSetting(key, event.target.checked)}
                        className="accent-purple-500"
                      />
                      <span>{label}</span>
                    </label>
                  ))}
                </div>
              </section>

              <section className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
                <h3 className="mb-2 text-xs font-semibold text-zinc-200">Tài khoản & hỗ trợ</h3>
                <div className="flex flex-wrap gap-2">
                  <Link href="/wiki" className="rounded-lg border border-white/10 px-3 py-2 text-xs text-emerald-300 hover:bg-white/[0.06]">
                    Wiki Game
                  </Link>
                  <ClearCacheButton showLabel />
                  <button
                    onClick={() => logout()}
                    className="rounded-lg border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-300 hover:bg-rose-500/20"
                  >
                    Đăng xuất
                  </button>
                </div>
              </section>
            </div>
          </section>
        </div>
      )}

      {mapPerformanceMode !== "off" && selectedObject && (
        <div className="absolute bottom-28 left-1/2 -translate-x-1/2 pointer-events-auto animate-fade-in z-20 max-w-[calc(100vw-2rem)]">
          <div className={`relative w-72 max-w-full px-4 py-4 rounded-2xl border ${cfg.borderColor} ${cfg.bgColor} backdrop-blur-md shadow-2xl max-h-[calc(100dvh-10rem)] overflow-y-auto custom-scrollbar`}>
            <button
              onClick={() => selectObject(null)}
              className="absolute top-2 right-2 w-6 h-6 flex items-center justify-center rounded-full bg-zinc-800/60 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs transition-colors cursor-pointer"
            >
              ✕
            </button>

            <div className="flex items-center gap-2.5 mb-3">
              <div className={`w-9 h-9 rounded-xl ${cfg.bgColor} border ${cfg.borderColor} flex items-center justify-center text-lg shrink-0`}>
                {cfg.icon}
              </div>
              <div>
                <div className={`font-bold font-serif text-sm ${cfg.color}`}>
                  {selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                    ? monsterDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "Resource"
                    ? itemDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "WildGu"
                    ? guSpecFor(selectedObject.content_id, selectedObject.rank)?.name || selectedObject.content_id
                    : selectedObject.cell_type === "GroundDrop"
                    ? `Hài cốt: ${(selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id))?.loser_name || "Vô danh"}`
                    : selectedObject.cell_type === "Sect"
                    ? `Tông Môn: ${selectedObject.content_id}`
                    : selectedObject.cell_type === "ENTRY_PORTAL"
                    ? `Cổng Vào: ${selectedObject.content_id.replace(/^spatial_realm_/, "").replace(/_/g, " ")}`
                    : selectedObject.cell_type === "INHERITANCE_ALTAR"
                    ? `Đài Truyền Thừa: ${selectedObject.content_id.replace(/^altar_/, "").replace(/_/g, " ")}`
                    : selectedObject.content_id}
                </div>
                <div className="text-[10px] text-zinc-500 font-serif">
                  {cfg.label} {selectedObject.rank ? `• Rank ${selectedObject.rank}` : ""}
                </div>
              </div>
            </div>

            <div className="flex gap-3 text-[10px] text-zinc-500 font-mono mb-3">
              <span>X: {selectedObject.x}</span>
              <span>Y: {selectedObject.y}</span>
            </div>

            {selectedObject.cell_type === "Sect" && (() => {
              const sect = sectMap.find((s: ApiResponseData) => s.id === selectedObject.id || (s.map_x === selectedObject.x && s.map_y === selectedObject.y));
              if (!sect) return null;
              const isMySect = sect.id === mySectStatus?.sect?.id;
              const isShielded = !!sect.shield_until && new Date(sect.shield_until).getTime() > Date.now();

              return (
                <div className="mb-3 space-y-1.5 text-xs font-serif bg-amber-950/20 border border-amber-500/30 rounded-xl p-2.5">
                  <div className="flex justify-between items-center text-[11px] font-mono text-zinc-300">
                    <span>🛡️ Phòng Thủ: <strong className="text-rose-400">{Math.round(sect.total_defense_power || 0).toLocaleString()}</strong></span>
                    {isMySect ? (
                      <span className="text-[10px] text-blue-400 font-bold bg-blue-950/40 border border-blue-500/30 px-1.5 py-0.5 rounded">Tông Môn Ta</span>
                    ) : isShielded ? (
                      <span className="text-[10px] text-rose-400 font-bold bg-rose-950/40 border border-rose-500/30 px-1.5 py-0.5 rounded">Đại Trận Bảo Hộ</span>
                    ) : (
                      <span className="text-[10px] text-emerald-400 font-bold bg-emerald-950/40 border border-emerald-500/30 px-1.5 py-0.5 rounded">Có Thể Khiêu Chiến</span>
                    )}
                  </div>
                  {isShielded && (
                    <div className="text-[10px] text-rose-400 font-mono">
                      Bảo hộ đến: {new Date(sect.shield_until).toLocaleTimeString()}
                    </div>
                  )}
                </div>
              );
            })()}

            {selectedObject.cell_type === "Boss" && selectedObject.cleared && (
              <div className="px-2 py-1.5 rounded-lg bg-zinc-800/60 text-[10px] text-zinc-400 font-serif mb-3">
                💀 Quái vật đang hồi sinh — hãy quay lại sau.
              </div>
            )}

            {selectedObject.cell_type === "GroundDrop" && (() => {
              const drop = selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id);
              if (!drop) return null;
              const protExpiry = drop.protected_until ? new Date(drop.protected_until) : null;
              const isProtected = drop.is_protected || (protExpiry ? protExpiry.getTime() > Date.now() : false);
              const isKiller = gamePlayer?.id && drop.killer_id === gamePlayer.id;

              return (
                <div className="mb-3 space-y-1.5 text-xs font-serif">
                  <div className="flex items-center justify-between text-[11px] text-zinc-400 bg-zinc-900/60 p-2 rounded-lg border border-zinc-800">
                    <span>⚔️ Kẻ diệt: <strong className="text-amber-400">{drop.killer_name || "Yêu thú/Hệ thống"}</strong></span>
                    {isProtected ? (
                      isKiller ? (
                        <span className="text-emerald-400 text-[10px] font-bold">✨ Quyền riêng</span>
                      ) : (
                        <span className="text-rose-400 text-[10px] font-bold">🔒 Khóa Killer</span>
                      )
                    ) : (
                      <span className="text-amber-400 text-[10px] font-bold">🔓 Tự do nhặt</span>
                    )}
                  </div>
                  <div className="grid grid-cols-2 gap-1 text-[11px] text-amber-200/90 bg-amber-950/30 p-2 rounded-lg border border-amber-900/40 font-mono">
                    <div>💎 {drop.loot_data?.primeval_stones || 0} Đá NT</div>
                    <div>✨ {drop.loot_data?.immortal_stones || 0} Tiên Thạch</div>
                    {drop.loot_data?.items && drop.loot_data.items.length > 0 && (
                      <div>📦 {drop.loot_data.items.length} Vật phẩm</div>
                    )}
                    {drop.loot_data?.gus && drop.loot_data.gus.length > 0 && (
                      <div>🐛 {drop.loot_data.gus.length} Cổ trùng</div>
                    )}
                    {drop.loot_data?.recipes && drop.loot_data.recipes.length > 0 && (
                      <div>📜 {drop.loot_data.recipes.length} Công thức</div>
                    )}
                  </div>
                </div>
              );
            })()}

            {selectedObject.cell_type === "TeleportArray" && (() => {
              const tpArray = myTeleportArrays.find((a: TeleportArray) => a.id === selectedObject.id || (a.x === selectedObject.x && a.y === selectedObject.y));

              return (
                <div className="mb-3 space-y-1.5 text-xs font-serif bg-cyan-950/30 border border-cyan-500/30 rounded-xl p-2.5">
                  <div className="flex justify-between items-center text-[11px] font-mono text-zinc-300">
                    <span>🌀 Cấp: <strong className="text-cyan-300">{tpArray?.tier || "TIER_1"}</strong></span>
                    <span className="text-[10px] text-cyan-400 font-bold bg-cyan-950/40 border border-cyan-500/30 px-1.5 py-0.5 rounded">
                      {tpArray?.direction_type === "ONE_WAY_SEND" ? "Gửi 1 Chiều" : tpArray?.direction_type === "ONE_WAY_RECEIVE" ? "Nhận 1 Chiều" : "2 Chiều"}
                    </span>
                  </div>
                  {tpArray ? (
                    <div className="space-y-1 text-[11px] font-mono text-zinc-400 pt-1 border-t border-cyan-900/30">
                      <div className="flex justify-between">
                        <span>Độ bền:</span>
                        <span className="text-emerald-400 font-bold">{tpArray.current_durability} / {tpArray.max_durability}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Quyền hạn:</span>
                        <span className="text-amber-400 font-bold">{tpArray.access_policy === "PUBLIC" ? "Công Khai" : tpArray.access_policy === "SECT" ? "Tông Môn" : "Cá Nhân"}</span>
                      </div>
                      {tpArray.fee_enabled && (
                        <div className="flex justify-between">
                          <span>Phí dùng:</span>
                          <span className="text-yellow-400 font-bold">{tpArray.fee_amount.toLocaleString()} {tpArray.fee_currency === "immortal_stones" ? "Tiên Thạch" : "Nguyên Thạch"}</span>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="text-[10px] text-zinc-400 font-mono">
                      Truyền Tống Trận đồ cổ xưa trên Ngũ Vực.
                    </div>
                  )}
                </div>
              );
            })()}

            <div className="flex gap-2.5">
              {selectedObject.cell_type === "Player" ? (
                (() => {
                  const dx = Math.abs((gamePlayer?.x ?? 0) - selectedObject.x);
                  const dy = Math.abs((gamePlayer?.y ?? 0) - selectedObject.y);
                  const dist = Math.max(dx, dy);
                  const inPvPRange = dist <= 2;

                  return (
                    <>
                      <button
                        disabled={isMoving}
                        title={isMoving ? "Đang di chuyển..." : inPvPRange ? "Tấn Công (PvP)" : "Tiếp Cận Đối Phương"}
                        onClick={async () => {
                          if (inPvPRange) {
                            movePlayer(gamePlayer!.x, gamePlayer!.y, selectedObject.id);
                          } else {
                            window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                          }
                        }}
                        className={`flex-1 flex items-center justify-center py-2.5 rounded-xl ${
                          inPvPRange ? "bg-red-600 hover:bg-red-500 shadow-red-950/30" : "bg-amber-600 hover:bg-amber-500 shadow-amber-950/30"
                        } disabled:opacity-60 text-white text-base transition-all cursor-pointer shadow-lg`}
                      >
                        <span>{inPvPRange ? "⚔️" : "🏃"}</span>
                      </button>
                  <button
                    title="Trinh Thám"
                    onClick={() => fetchPlayerProfile(selectedObject.content_id)}
                    className="flex-1 flex items-center justify-center py-2.5 rounded-xl bg-purple-600 hover:bg-purple-500 shadow-purple-950/30 text-white text-base transition-all cursor-pointer shadow-lg"
                  >
                    <span>👁️</span>
                  </button>
                  {gamePlayer && gamePlayer.username !== selectedObject.content_id && (
                    <button
                      title="Mời Tổ Đội"
                      onClick={async () => {
                        let currentParty = useGameStore.getState().party;
                        if (!currentParty) {
                          const created = await useGameStore.getState().createParty();
                          if (!created) return;
                          currentParty = useGameStore.getState().party;
                        }
                        if (currentParty && currentParty.leader_id !== gamePlayer.id) {
                          useGameStore.getState().addToast("Chỉ có Trưởng nhóm mới có quyền mời!", "error");
                          return;
                        }
                        await useGameStore.getState().invitePlayer(selectedObject.content_id);
                      }}
                      className="flex-1 flex items-center justify-center py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-base transition-all cursor-pointer shadow-lg"
                    >
                      <span>🤝</span>
                    </button>
                  )}
                </>
              );
                })()
              ) : selectedObject.cell_type === "GroundDrop" ? (
                gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y ? (
                  <button
                    onClick={async () => {
                      const drop = selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id);
                      const res = await lootGroundDrop(drop?.id || selectedObject.id!);
                      if (res?.success) {
                        selectObject(null);
                      }
                    }}
                    className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-sm font-bold font-serif transition-all cursor-pointer shadow-lg"
                  >
                    <span>🎁</span>
                    <span>Lụm</span>
                  </button>
                ) : (
                  <button
                    disabled={isMoving}
                    title="Tiếp Cận Chiến Lợi Phẩm"
                    onClick={() => {
                      window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                    }}
                    className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-sm font-bold font-serif transition-all cursor-pointer shadow-lg"
                  >
                    <span>🏃</span>
                    <span>Tiếp Cận</span>
                  </button>
                )
              ) : selectedObject.cell_type === "Sect" ? (
                (() => {
                  const sect = sectMap.find((s: ApiResponseData) => s.id === selectedObject.id || (s.map_x === selectedObject.x && s.map_y === selectedObject.y));
                  const isMySect = sect?.id === mySectStatus?.sect?.id;
                  const isShielded = !!sect?.shield_until && new Date(sect.shield_until).getTime() > Date.now();

                  return (
                    <>
                      {isMySect ? (
                        <button
                          onClick={() => setActiveModal("sect")}
                          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg"
                        >
                          <span>🏛️</span>
                          <span>Trụ Sở Tông Môn</span>
                        </button>
                      ) : (
                        <button
                          disabled={actionLoading || isShielded}
                          title={isShielded ? "Tông môn đang trong thời gian bảo hộ" : "Phát động công kích Tông môn"}
onClick={() => {
                              setAttackTargetSect(sect || selectedObject);
                              fetchSectMembers();
                            }}
                          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-gradient-to-r from-rose-600 to-red-600 hover:from-rose-500 hover:to-red-500 disabled:opacity-50 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg"
                        >
                          <span>⚔️</span>
                        </button>
                      )}

                      <button
                        disabled={isMoving}
                        title="Tiếp Cận Tông Môn"
                        onClick={() => {
                          window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                        }}
                        className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg"
                      >
                        <span>🏃</span>
                        <span>Tiếp Cận</span>
                      </button>
                    </>
                  );
                })()
              ) : selectedObject.cell_type === "TeleportArray" ? (
                (() => {
                  const tpArray = myTeleportArrays.find((a: TeleportArray) => a.id === selectedObject.id || (a.x === selectedObject.x && a.y === selectedObject.y));
                  const isOwner = gamePlayer?.id && tpArray?.owner_id === gamePlayer.id;
                  const isMySect = tpArray?.sect_id && tpArray.sect_id === mySectStatus?.sect?.id;
                  const canManage = isOwner || (isMySect && (mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO"));
                  const isStanding = gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y;

                  return (
                    <>
                      <button
                        onClick={() => setActiveModal("teleport_directory")}
                        className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg"
                      >
                        <span>🌀</span>
                        <span>{isStanding ? "Truyền Tống" : "Xem Danh Bạ"}</span>
                      </button>
                      {canManage && tpArray && (
                        <button
                          onClick={() => setManagingTeleportArray(tpArray)}
                          title="Quản Lý Trận Pháp"
                          className="px-3 flex items-center justify-center rounded-xl bg-zinc-800 hover:bg-zinc-700 text-cyan-300 border border-cyan-700/40 text-xs font-serif transition-all cursor-pointer shadow-lg"
                        >
                          <span>⚙️</span>
                        </button>
                      )}
                      {!isStanding && (
                        <button
                          disabled={isMoving}
                          title="Tiếp Cận Truyền Tống Trận"
                          onClick={() => {
                            window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                          }}
                          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg"
                        >
                          <span>🏃</span>
                          <span>Tiếp Cận</span>
                        </button>
                      )}
                    </>
                  );
                })()
              ) : selectedObject.cell_type === "INHERITANCE_ALTAR" ? (
                (() => {
                  const isStanding = gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y;
                  return (
                    <>
                      <button
                        onClick={() => useGameStore.getState().fetchAltarInfo(selectedObject.content_id)}
                        className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg shadow-amber-950/40"
                      >
                        <span>🏛️</span>
                        <span>Khảo Nghiệm Đài</span>
                      </button>
                      {!isStanding && (
                        <button
                          disabled={isMoving}
                          title="Tiếp Cận Đài Truyền Thừa"
                          onClick={() => {
                            window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                          }}
                          className="px-3 flex items-center justify-center gap-1 py-2.5 rounded-xl bg-zinc-700 hover:bg-zinc-600 text-white text-xs font-bold font-serif transition-all cursor-pointer"
                        >
                          <span>🏃</span>
                          <span>Đến Gần</span>
                        </button>
                      )}
                    </>
                  );
                })()
              ) : selectedObject.cell_type === "ENTRY_PORTAL" ? (
                (() => {
                  const isStanding = gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y;
                  const realmId = selectedObject.content_id.replace(/^spatial_realm_/, "");
                  return (
                    <>
                      <button
                        onClick={() => useGameStore.getState().enterSpatialRealm(realmId)}
                        className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-bold font-serif transition-all cursor-pointer shadow-lg shadow-amber-950/40"
                      >
                        <span>🚪</span>
                        <span>Tiến Vào Bí Cảnh</span>
                      </button>
                      {!isStanding && (
                        <button
                          disabled={isMoving}
                          title="Tiếp Cận Cổng Bí Cảnh"
                          onClick={() => {
                            window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                          }}
                          className="px-3 flex items-center justify-center gap-1 py-2.5 rounded-xl bg-zinc-700 hover:bg-zinc-600 text-white text-xs font-bold font-serif transition-all cursor-pointer"
                        >
                          <span>🏃</span>
                          <span>Đến Gần</span>
                        </button>
                      )}
                    </>
                  );
                })()
              ) : (
                <button
                  disabled={isMoving || !!selectedObject.cleared}
                  title={isMoving ? "Đang di chuyển..." : selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "Chiến Đấu" : "Tiếp Cận"}
                  onClick={() => {
                    window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                  }}
                  className={`flex-1 flex items-center justify-center py-2.5 rounded-xl ${
                    selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                      ? "bg-red-600 hover:bg-red-500 shadow-red-950/30"
                      : "bg-emerald-600 hover:bg-emerald-500 shadow-emerald-950/30"
                  } disabled:opacity-60 text-white text-base transition-all cursor-pointer shadow-lg`}
                >
                  <span>{selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "⚔️" : "🏃"}</span>
                </button>
              )}
              <button
                title="Đóng"
                onClick={() => selectObject(null)}
                className="px-4 py-2.5 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-sm font-bold transition-all cursor-pointer border border-zinc-700/40"
              >
                ✕
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Challenge Sect Modal */}
      {attackTargetSect && (
        <div className="fixed inset-0 z-50 h-[100dvh] bg-black/80 flex items-center justify-center p-4 pt-[calc(env(safe-area-inset-top)+1rem)] pb-[calc(env(safe-area-inset-bottom)+1rem)] backdrop-blur-sm pointer-events-auto">
          <div className="w-full max-w-md max-h-[calc(100dvh-2rem)] flex flex-col overflow-hidden bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 text-left shadow-2xl">
            <div className="shrink-0 flex items-center justify-between border-b border-zinc-900 pb-3">
              <div>
                <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
                  <span>🏯</span> Khiêu Chiến Tông Môn — {attackTargetSect.name || attackTargetSect.content_id}
                </h3>
                <span className="text-[10px] font-mono text-zinc-500 block mt-0.5">
                  Tọa độ: ({attackTargetSect.map_x ?? attackTargetSect.x}, {attackTargetSect.map_y ?? attackTargetSect.y}) · Cấp {attackTargetSect.level || attackTargetSect.rank || 1}
                </span>
              </div>
              <button onClick={() => setAttackTargetSect(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto space-y-3 custom-scrollbar pr-1">
              <div className="flex flex-col gap-1 text-xs font-mono bg-zinc-900/40 p-3 rounded-xl border border-zinc-900">
                <div className="flex justify-between">
                  <span className="text-zinc-500">Sức Phòng Thủ Hộ Sơn:</span>
                  <span className="text-rose-400 font-bold">🛡 {Math.round(attackTargetSect.total_defense_power || 0).toLocaleString()}</span>
                </div>
                {attackTargetSect.shield_until && new Date(attackTargetSect.shield_until).getTime() > Date.now() && (
                  <div className="flex justify-between text-rose-400 font-bold mt-1">
                    <span>Đại Trận Bảo Hộ:</span>
                    <span>{new Date(attackTargetSect.shield_until).toLocaleTimeString()}</span>
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-1.5">
                <p className="text-xs text-zinc-500 font-mono">
                  Đấu sẽ sử dụng Hộ Sơn Thần Thú + Đọc allied: Độc Lí + Minh Quân hội üy hội + Thú Nuôi
                </p>
              </div>
            </div>

            <div className="shrink-0 pt-2 border-t border-zinc-900">
              <button
                disabled={actionLoading || (attackTargetSect.shield_until ? new Date(attackTargetSect.shield_until).getTime() > Date.now() : false)}
                onClick={async () => {
                  const ok = await attackSect(attackTargetSect.id);
                  if (ok) setAttackTargetSect(null);
                }}
                className="w-full py-2.5 bg-gradient-to-r from-rose-600 to-red-600 hover:from-rose-500 hover:to-red-600 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer"
              >
                {attackTargetSect.shield_until && new Date(attackTargetSect.shield_until).getTime() > Date.now() ? "Đang Bảo Hộ" : "Phát Động Công Kích Tông Môn"}
              </button>
            </div>
          </div>
        </div>
      )}

      {mapPerformanceMode !== "off" && <GuQuickBar />}
      {mapPerformanceMode !== "off" && <MovementJoystick />}

      <SpatialRealmHUD />
      <InheritanceAltarModal />
      <SpatialRealmsModal isOpen={showSpatialRealms} onClose={() => setShowSpatialRealms(false)} />
    </div>
  );
}
