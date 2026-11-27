import React, { useEffect, useState, useRef } from "react";
import { estimateManaFromSnapshot, estimateHPFromSnapshot, ImmortalAperture, useGameStore } from "@/store/useGameStore";
import {
  User,
  Briefcase,
  Flame,
  Store,
  Gem,
  Compass,
  Coins,
  Swords,
  Lock,
  Send,
  Loader2,
  Sparkles,
  Crown,
  Sword,
  History,
  MessageSquare,
  X,
  Shield,
  Gift,
  Award,
  Trophy,
  PawPrint,
  Radar,
  Layers,
  Eye,
  Settings2,
  ChevronRight,
  MapPin,
  LayoutGrid,
  Pin,
  Search,
  SlidersHorizontal,
} from "lucide-react";
import { LogConsole } from "./LogConsole";
import { Num } from "./Num";
import { SpatialRealmsModal } from "./SpatialRealmsModal";
import { LobbySettingsPanel } from "./LobbySettingsPanel";
import { LobbyGridSettingsEditor } from "./LobbyGridSettingsEditor";
import { CORE_LOBBY_FEATURE_IDS, getFeatureLockInfo, LOBBY_ENTRY_LABELS, LOBBY_FEATURE_IDS, LOBBY_MENU_GROUPS } from "@/lib/lobby-menu";
import { QuestDef, STORY_CHAPTER_NAMES, formatRewards } from "@/data/quests";
import { updateUiPreferences, useUiPreferences } from "@/lib/ui-preferences";
import { useAccessibleDialog } from "@/hooks/useAccessibleDialog";
import Link from "next/link";

interface HubItem {
  id: string;
  label: string;
  icon: React.ComponentType<ApiResponseData>;
  gradient: string;
  glow: string;
  borderColor: string;
  iconColor: string;
  artwork?: string;
  locked?: boolean;
  lockLabel?: string;
  lockDesc?: string;
  desc?: string;
  isGroup?: boolean;
  childCount?: number;
}

const formatRelativeTime = (createdAtStr: string): string => {
  if (!createdAtStr) return "";
  const date = new Date(createdAtStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.max(0, Math.floor(diffMs / 1000));

  if (diffSec < 60) return "vừa xong";

  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}p trước`;

  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}g trước`;

  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} ngày trước`;
};

export const LobbyView: React.FC = () => {
  const {
    player,
    manaSnapshot,
    logs,
    setView,
    fetchPlayerProfile,
    openDivinationModal,
    toggleRadarDrawer,
    mySectStatus,
    addToast,
    aperture,
    fetchAperture,
    setSpatialDirectoryOpen,
    regions,
    regionsLoaded,
    fetchRegions,
    quests,
    unlockedFeatures,
    fetchQuests,
    completeQuest,
    actionLoading,
  } = useGameStore();
  const [manaNow, setManaNow] = useState(() => Date.now());
  const [combatSettingsOpen, setCombatSettingsOpen] = useState(false);
  const [activeGroupId, setActiveGroupId] = useState<string | null>(null);
  const [allFeaturesOpen, setAllFeaturesOpen] = useState(false);
  const [allFeaturesQuery, setAllFeaturesQuery] = useState("");
  const [gridSettingsOpen, setGridSettingsOpen] = useState(false);
  const [roadmapOpen, setRoadmapOpen] = useState(false);
  const [selectedChapter, setSelectedChapter] = useState(1);
  const [lockedModalItem, setLockedModalItem] = useState<HubItem | null>(null);
  const combatSettingsDialogRef = useRef<HTMLElement>(null);
  const { lobbyGridItems, lobbyGridLimit } = useUiPreferences();
  const { dialogRef: groupDialogRef, onKeyDown: groupDialogKeyDown } = useAccessibleDialog<HTMLElement>(
    () => setActiveGroupId(null),
    Boolean(activeGroupId),
    activeGroupId ?? undefined
  );
  const { dialogRef: allFeaturesDialogRef, onKeyDown: allFeaturesDialogKeyDown } = useAccessibleDialog<HTMLElement>(
    () => setAllFeaturesOpen(false),
    allFeaturesOpen
  );
  const { dialogRef: gridSettingsDialogRef, onKeyDown: gridSettingsDialogKeyDown } = useAccessibleDialog<HTMLElement>(
    () => setGridSettingsOpen(false),
    gridSettingsOpen
  );

  useEffect(() => {
    if (!combatSettingsOpen) return;
    const dialog = combatSettingsDialogRef.current;
    if (!dialog) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const getControls = () => dialog.querySelectorAll<HTMLElement>(
      'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    );
    (getControls()[0] || dialog).focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setCombatSettingsOpen(false);
        return;
      }
      if (event.key !== "Tab") return;
      const controls = getControls();
      if (!controls.length) {
        event.preventDefault();
        dialog.focus();
      } else if (event.shiftKey && document.activeElement === controls[0]) {
        event.preventDefault();
        controls[controls.length - 1].focus();
      } else if (!event.shiftKey && document.activeElement === controls[controls.length - 1]) {
        event.preventDefault();
        controls[0].focus();
      }
    };
    dialog.addEventListener("keydown", onKeyDown);
    return () => {
      dialog.removeEventListener("keydown", onKeyDown);
      if (previousFocus?.isConnected) previousFocus.focus();
    };
  }, [combatSettingsOpen]);

  useEffect(() => {
    if (player && player.rank >= 6 && !aperture) {
      void fetchAperture();
    }
  }, [player, aperture, fetchAperture]);

  useEffect(() => {
    if (player && quests.length === 0) {
      void fetchQuests();
    }
  }, [player, quests.length, fetchQuests]);

  const typedQuests = (quests as QuestDef[]) || [];
  const currentQuest = typedQuests.find((q) => q.is_current) || typedQuests.find((q) => q.status !== "COMPLETED") || null;
  const completedQuestCount = typedQuests.filter((q) => q.status === "COMPLETED").length;

  useEffect(() => {
    if (!regionsLoaded) void fetchRegions();
  }, [regionsLoaded, fetchRegions]);

  useEffect(() => {
    if (!player || !manaSnapshot) return;

    const isManaFull = player.rank >= 6 ? false : (manaSnapshot.current_mana >= manaSnapshot.max_mana);
    const isHPFull = manaSnapshot.hp >= manaSnapshot.base_hp;

    if (isManaFull && isHPFull) return;

    const interval = setInterval(() => {
      setManaNow(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [player, manaSnapshot]);

  const displayedMana = player
    ? estimateManaFromSnapshot(manaSnapshot?.player_id === player.id ? manaSnapshot : null, manaNow) ?? player.current_mana
    : 0;
  const displayedMaxMana = player?.max_mana || 0;
  const displayedManaPercent = displayedMaxMana > 0 ? Math.min(100, (displayedMana / displayedMaxMana) * 100) : 0;

  const displayedHP = player
    ? estimateHPFromSnapshot(manaSnapshot?.player_id === player.id ? manaSnapshot : null, manaNow) ?? player.hp
    : 0;
  const displayedMaxHP = player?.base_hp || 0;
  const displayedHPPercent = displayedMaxHP > 0 ? Math.min(100, (displayedHP / displayedMaxHP) * 100) : 0;

  const timeSpeedMultiplier = player?.time_speed_multiplier || 1;
  const isImmortal = player ? player.rank >= 6 : false;
  const effectiveManaMultiplier = isImmortal ? timeSpeedMultiplier * (aperture?.time_rate ?? 10) : timeSpeedMultiplier;
  const hpRegenRealHourly = (player?.hp_regen_rate || 0) * timeSpeedMultiplier;
  const manaRegenRealHourly = (player?.mana_regen_rate || 0) * effectiveManaMultiplier;
  const thoughtRegenRealHourly = (player?.thought_regen_rate || 0) * timeSpeedMultiplier;
  const soulRegenRealHourly = (player?.soul_regen_rate ?? (player?.max_soul || 0) * 0.10) * timeSpeedMultiplier;

  const displayedThoughtPercent = player
    ? (player.max_thought > 0 ? Math.min(100, ((player.current_thought || 0) / player.max_thought) * 100) : 0)
    : 0;

  const displayedSoulPercent = player
    ? (player.max_soul > 0 ? Math.min(100, ((player.soul_hp || 0) / player.max_soul) * 100) : 0)
    : 0;
  const currentRegion = regions.find((region) => region.id === player?.current_region);
  const currentLocation = currentRegion?.name || player?.current_region || "Chưa xác định";

  const formatTimeRemaining = (current: number, max: number, realRegenPerHour: number) => {
    if (current >= max || realRegenPerHour <= 0) return "Đã đầy";
    const hoursRemaining = (max - current) / realRegenPerHour;
    const totalMinutes = Math.floor(hoursRemaining * 60);
    const h = Math.floor(totalMinutes / 60);
    const m = totalMinutes % 60;
    if (h > 0) return `Đầy sau ${h}h ${m}m`;
    return `Đầy sau ${m}m`;
  };

  // Management items (Removed "Truyền Tin" as requested)
  const managementItems: HubItem[] = [
    {
      id: "character",
      label: "Nhân Vật",
      icon: User,
      gradient: "from-emerald-500 to-teal-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(16,185,129,0.4)]",
      borderColor: "border-emerald-400/20",
      iconColor: "text-emerald-100",
    },
    {
      id: "inventory",
      label: "Túi Đồ",
      icon: Briefcase,
      gradient: "from-cyan-500 to-blue-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(6,182,212,0.4)]",
      borderColor: "border-cyan-400/20",
      iconColor: "text-cyan-100",
    },
    {
      id: "refine",
      label: "Luyện Cổ",
      icon: Flame,
      gradient: "from-amber-500 to-orange-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(245,158,11,0.4)]",
      borderColor: "border-amber-400/20",
      iconColor: "text-amber-100",
    },
    {
      id: "aperture",
      label: "Tiên Khiếu",
      icon: Sparkles,
      gradient: player && player.rank >= 6 ? "from-indigo-500 to-purple-600" : "from-zinc-800 to-zinc-950",
      glow: player && player.rank >= 6 ? "group-hover:shadow-[0_0_25px_rgba(99,102,241,0.45)]" : "",
      borderColor: player && player.rank >= 6 ? "border-indigo-400/30" : "border-zinc-800",
      iconColor: player && player.rank >= 6 ? "text-indigo-400" : "text-zinc-600",
      locked: !player || player.rank < 6,
    },
    {
      id: "premium-store",
      label: "Cát Bảo Các",
      icon: Sparkles,
      gradient: "from-violet-500 to-emerald-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(139,92,246,0.4)]",
      borderColor: "border-violet-400/20",
      iconColor: "text-violet-100",
    },
    {
      id: "shop",
      label: "Cửa Hàng",
      icon: Store,
      gradient: "from-yellow-400 to-amber-500",
      glow: "group-hover:shadow-[0_0_20px_rgba(234,179,8,0.4)]",
      borderColor: "border-yellow-400/20",
      iconColor: "text-yellow-100",
    },
    {
      id: "auction",
      label: "Chợ, Đấu Giá",
      icon: Gem,
      gradient: "from-violet-500 to-purple-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(139,92,246,0.4)]",
      borderColor: "border-violet-400/20",
      iconColor: "text-violet-100",
    },
    {
      id: "pets",
      label: "Nô Thú",
      icon: PawPrint,
      gradient: "from-purple-500 to-fuchsia-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(168,85,247,0.45)]",
      borderColor: "border-purple-400/30",
      iconColor: "text-purple-500",
    },
    {
      id: "sect",
      label: "Tông Môn",
      icon: Crown,
      gradient: "from-blue-600 to-indigo-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(37,99,235,0.45)]",
      borderColor: "border-blue-400/30",
      iconColor: "text-blue-400",
      locked: false,
    },
    {
      id: "practice-room",
      label: "Phòng Tập",
      icon: Swords,
      gradient: "from-blue-500 to-indigo-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(99,102,241,0.45)]",
      borderColor: "border-blue-400/30",
      iconColor: "text-blue-500",
    },{
      id: "arena",
      label: "Võ Đài",
      icon: Swords,
      gradient: "from-rose-500 to-red-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(239,68,68,0.45)]",
      borderColor: "border-rose-400/30",
      iconColor: "text-rose-500",
    },
    {
      id: "infinite-tower",
      label: "Tháp Chiến Thuật",
      icon: Trophy,
      gradient: "from-amber-500 to-orange-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
      borderColor: "border-amber-400/30",
      iconColor: "text-amber-400",
    },
    {
      id: "death-arena",
      label: "Đấu Trường Sinh Tử",
      icon: Swords,
      gradient: "from-red-600 to-orange-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(234,88,12,0.45)]",
      borderColor: "border-red-500/30",
      iconColor: "text-red-600",
    },
    {
      id: "martial-arena",
      label: "Đại Hội Võ Thuật",
      icon: Trophy,
      gradient: "from-orange-500 to-amber-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
      borderColor: "border-orange-400/30",
      iconColor: "text-orange-500",
    },
    {
      id: "achievements",
      label: "Nhiệm Vụ & Thành Tựu",
      icon: Award,
      gradient: "from-amber-500 to-yellow-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
      borderColor: "border-amber-400/30",
      iconColor: "text-amber-500",
    },
    {
      id: "utilities",
      label: "Phong Ấn Cổ Trùng",
      icon: Layers,
      gradient: "from-amber-600 to-orange-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
      borderColor: "border-amber-500/30",
      iconColor: "text-amber-500",
    },
  ];

  const activityItems: HubItem[] = [
    {
      id: "explore",
      label: player?.current_region === "aperture_interior" && aperture?.id === player.map_instance_id
        ? "Lưới Địa Hình Tiên Khiếu"
        : "Khám Phá Dã Ngoại",
      icon: Compass,
      gradient: "from-emerald-600 to-teal-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(16,185,129,0.45)]",
      borderColor: "border-emerald-400/30",
      iconColor: "text-emerald-400",
    },
    {
      id: "dream-realm",
      label: "Mộng Cảnh",
      icon: Gem,
      gradient: "from-purple-500 to-indigo-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(168,85,247,0.45)]",
      borderColor: "border-purple-400/30",
      iconColor: "text-purple-400",
    },
    {
      id: "spatial-realms",
      label: "Bí Cảnh & Truyền Thừa",
      icon: Compass,
      gradient: "from-amber-500 to-orange-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
      borderColor: "border-amber-400/30",
      iconColor: "text-amber-400",
    },
    {
      id: "gacha",
      label: "Giải Trí",
      icon: Sparkles,
      gradient: "from-pink-500 to-rose-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(236,72,153,0.45)]",
      borderColor: "border-pink-400/30",
      iconColor: "text-pink-500",
    },
    {
      id: "leaderboard",
      label: "Bảng Xếp Hạng",
      icon: Trophy,
      gradient: "from-amber-500 to-yellow-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(217,119,6,0.45)]",
      borderColor: "border-amber-400/30",
      iconColor: "text-amber-500",
    },
    {
      id: "divination",
      label: "Thiên Cơ Các",
      icon: Eye,
      gradient: "from-amber-600 via-purple-600 to-cyan-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(168,85,247,0.45)]",
      borderColor: "border-purple-400/40",
      iconColor: "text-purple-300",
    },
    {
      id: "feedback",
      label: "Thỉnh Thiên Đạo",
      icon: Sparkles,
      gradient: "from-fuchsia-500 to-pink-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(217,70,239,0.4)]",
      borderColor: "border-fuchsia-400/20",
      iconColor: "text-fuchsia-100",
    },
    {
      id: "combat-history",
      label: "Lịch Sử Chiến Đấu",
      icon: History,
      gradient: "from-rose-500 to-red-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(239,68,68,0.4)]",
      borderColor: "border-rose-400/20",
      iconColor: "text-rose-100",
    },
    {
      id: "audit-logs",
      label: "Ký Sự Tu Hành",
      icon: History,
      gradient: "from-blue-500 to-indigo-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(59,130,246,0.4)]",
      borderColor: "border-blue-400/20",
      iconColor: "text-blue-100",
    },
  ];

  const groupIconMap: Record<string, React.ComponentType<ApiResponseData>> = {
    "combat-hub": Swords,
    "commerce-hub": Store,
    "adventure-hub": Compass,
    "system-hub": Layers,
  };

  const groupItems: HubItem[] = LOBBY_MENU_GROUPS.map((group) => ({
    id: group.id,
    label: group.label,
    desc: group.desc,
    icon: groupIconMap[group.id] || LayoutGrid,
    gradient: "from-amber-500 to-orange-600",
    glow: "group-hover:shadow-[0_0_25px_rgba(245,158,11,0.45)]",
    borderColor: "border-amber-400/30",
    iconColor: "text-amber-200",
    isGroup: true,
    childCount: group.children.length,
  }));

  const handleItemClick = (item: HubItem) => {
    if (item.locked) {
      setLockedModalItem(item);
      return;
    }
    if (item.isGroup) {
      setActiveGroupId(item.id);
      return;
    }
    setActiveGroupId(null);
    setAllFeaturesOpen(false);
    if (item.id === "divination") {
      openDivinationModal();
      return;
    }
    if (item.id === "spatial-realms") {
      setSpatialDirectoryOpen(true);
      return;
    }
    if (item.id === "explore" && player?.current_region === "aperture_interior" && aperture?.id === player.map_instance_id) {
      setView("aperture");
      return;
    }
    setView(item.id);
  };

  const togglePinFeature = (featureId: string) => {
    const exists = lobbyGridItems.includes(featureId);
    const next = exists
      ? lobbyGridItems.filter((id) => id !== featureId)
      : [...lobbyGridItems, featureId];
    updateUiPreferences({ lobbyGridItems: next });
  };

  const lobbyArtwork: Record<string, string> = {
    "combat-hub": "/assets/world/form-six-arm-ice-demon.webp",
    "commerce-hub": "/assets/items/icons/chest.webp",
    "adventure-hub": "/assets/world/spirit-beast.webp",
    "system-hub": "/assets/items/icons/scroll.webp",
    character: "/assets/world/form-reckless-savage.webp",
    inventory: "/assets/items/preview.webp",
    refine: "/assets/world/spirit-herb.webp",
    aperture: "/assets/world/ground-moss.webp",
    "caravan-shop": "/assets/world/spirit-herb.webp",
    "premium-store": "/assets/items/icons/chest.webp",
    shop: "/assets/items/icons/gourd.webp",
    auction: "/assets/items/icons/crystal.webp",
    explore: "/assets/world/great-regions-atlas.png",
    "dream-realm": "/assets/world/spirit-beast.webp",
    "spatial-realms": "/assets/world/ground-moss.webp",
    sect: "/assets/world/cultivator.webp",
    "practice-room": "/assets/world/form-reckless-savage.webp",
    arena: "/assets/world/form-six-arm-ice-demon.webp",
    "infinite-tower": "/assets/world/form-reckless-savage.webp",
    "death-arena": "/assets/world/form-six-arm-ice-demon.webp",
    "martial-arena": "/assets/world/form-reckless-savage.webp",
    achievements: "/assets/items/icons/scroll.webp",
    utilities: "/assets/items/icons/formation.webp",
    pets: "/assets/world/spirit-beast.webp",
    gacha: "/assets/items/icons/star.webp",
    leaderboard: "/assets/items/icons/star.webp",
    divination: "/assets/items/icons/eye.webp",
    feedback: "/assets/items/icons/scroll.webp",
    "combat-history": "/assets/items/icons/blade.webp",
    "audit-logs": "/assets/items/icons/map.webp",
  };

  const entryById = new Map<string, HubItem>(
    [...managementItems, ...activityItems, ...groupItems].map((item) => {
      const lockInfo = item.isGroup
        ? { locked: false, rule: undefined }
        : getFeatureLockInfo(item.id, player?.rank || 1, unlockedFeatures || []);
      const isLocked = Boolean(item.locked || lockInfo.locked);
      return [
        item.id,
        {
          ...item,
          locked: isLocked,
          lockLabel: lockInfo.rule?.lockLabel || (item.locked ? "Đạt Lục Chuyển để mở" : undefined),
          lockDesc: lockInfo.rule?.desc || item.desc,
          artwork: lobbyArtwork[item.id],
        },
      ];
    })
  );
  const effectiveGridLimit = lobbyGridLimit === 0 ? lobbyGridItems.length : lobbyGridLimit;
  const visibleLobbyItems = lobbyGridItems
    .slice(0, effectiveGridLimit)
    .map((id) => entryById.get(id))
    .filter((item): item is HubItem => Boolean(item));
  const activeGroup = activeGroupId
    ? LOBBY_MENU_GROUPS.find((group) => group.id === activeGroupId) ?? null
    : null;

  const getRankName = (rank: number, sub: number, aperture?: ImmortalAperture | null): string => {
    const ranks = ["Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"];
    const rankNames = [
      "Nhất Chuyển",
      "Nhị Chuyển",
      "Tam Chuyển",
      "Tứ Chuyển",
      "Ngũ Chuyển",
      "Lục Tiên",
      "Thất Tiên",
      "Bát Tiên",
      "Cửu Tôn",
      "Vĩnh Sinh",
    ];
    const rName = rankNames[rank - 1] || `${rank} Chuyển`;
    if (rank >= 6) {
      if (!aperture) return rName;
      const tribMap: Record<number, Array<{ key: string; label: string }>> = {
        6: [{ key: "earth_calamity_count", label: "Địa Tai" }, { key: "heavenly_tribulation_count", label: "Thiên Kiếp" }],
        7: [{ key: "earth_calamity_count", label: "Địa Tai" }, { key: "heavenly_tribulation_count", label: "Thiên Kiếp" }, { key: "grand_tribulation_count", label: "Hạo Kiếp" }],
        8: [{ key: "heavenly_tribulation_count", label: "Thiên Kiếp" }, { key: "grand_tribulation_count", label: "Hạo Kiếp" }, { key: "myriad_tribulation_count", label: "Vạn Kiếp" }],
        9: [{ key: "grand_tribulation_count", label: "Hạo Kiếp" }, { key: "myriad_tribulation_count", label: "Vạn Kiếp" }, { key: "chaos_calamity_count", label: "Hỗn Độn" }],
      };
      const types = tribMap[rank] || [];
      const getCnt = (k: string): number => { switch (k) { case "earth_calamity_count": return aperture.earth_calamity_count; case "heavenly_tribulation_count": return aperture.heavenly_tribulation_count; case "grand_tribulation_count": return aperture.grand_tribulation_count; case "myriad_tribulation_count": return aperture.myriad_tribulation_count; case "chaos_calamity_count": return aperture.chaos_calamity_count; default: return 0; } };
      const parts = types.filter(t => getCnt(t.key) > 0).map(t => `${t.label}: ${getCnt(t.key)}`);
      const tribStr = parts.length > 0 ? ` [${parts.join(", ")}]` : "";
      return `${rName}${tribStr}`;
    }
    const sName = ranks[sub - 1] || `${sub} Giai`;
    return `${rName} • ${sName}`;
  };

  const getInitials = (username: string): string => {
    if (!username) return "?";
    return username.slice(0, 2).toUpperCase();
  };

  return (
    <div
      className="mmo-paper-theme mmo-paper-surface relative isolate w-full mx-auto px-2 sm:px-4 py-2 sm:py-4 flex flex-col gap-4 sm:gap-6 text-[var(--text-primary,#f4e8cb)]"
      style={{
        backgroundImage: "linear-gradient(rgba(18, 13, 9, 0.88), rgba(18, 13, 9, 0.92)), url('/assets/world/great-regions-atlas.png')",
        backgroundSize: "cover, 700% 100%",
        backgroundPosition: "center, left center",
      }}
    >
      {/* 1. CHARACTER & RESOURCE HUD */}
      {player && (
        <div className="relative overflow-hidden bg-[var(--bg-surface)]/95 border border-[var(--border-default)] rounded-2xl sm:rounded-3xl p-3 sm:p-5 flex flex-col md:flex-row items-center justify-between gap-3 sm:gap-6 shadow-xl">
          <div aria-hidden="true" className="absolute inset-0 pointer-events-none bg-[radial-gradient(ellipse_at_0%_0%,rgba(180,132,55,0.12),transparent_42%),linear-gradient(110deg,transparent_45%,rgba(14,32,24,0.28))]" />
          {/* Left Block: Avatar & Name & Tu vi (Reduced width & wrapped text) */}
          <div className="relative z-10 flex items-center gap-3 sm:gap-4 w-full md:w-48 lg:w-80 shrink-0">
            <div className="w-12 h-12 sm:w-14 sm:h-14 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center text-white font-bold border border-[var(--border-default)] shrink-0 overflow-hidden">
              {player.avatar_url ? (
                <img src={player.avatar_url} alt={player.username} className="w-full h-full object-cover" />
              ) : (
                getInitials(player.username)
              )}
            </div>
            <div className="flex flex-col min-w-0">
              <h2 className="text-base sm:text-lg font-serif font-bold text-zinc-100 break-words whitespace-normal leading-tight">
                {player.username}
              </h2>
              <span className="text-xs text-zinc-500 font-mono font-semibold break-words whitespace-normal leading-tight mt-1">
                {getRankName(player.rank, player.sub_rank, aperture)}
              </span>
              <span className="mt-1.5 inline-flex min-w-0 items-center gap-1 text-[10px] leading-tight text-amber-200/80" title={`${currentLocation} (${player.x}, ${player.y})`}>
                <MapPin className="h-3 w-3 shrink-0 text-amber-400" />
                <span className="truncate">{currentLocation}</span>
                <span className="shrink-0 font-mono text-amber-100/55">({player.x}, {player.y})</span>
              </span>
            </div>
          </div>

          {/* Resource bars */}
          <div className="relative z-10 grid w-full grow grid-cols-2 gap-x-4 gap-y-3 md:max-w-[620px] md:gap-x-6">
            {[
              {
                label: "Sinh Lực",
                value: `${Math.round(displayedHP)}/${Math.round(displayedMaxHP)}`,
                percent: displayedHPPercent,
                regen: hpRegenRealHourly,
                current: displayedHP,
                max: displayedMaxHP,
                uncapped: false,
                color: "from-rose-500 to-red-400",
                textColor: "text-rose-200",
              },
              {
                label: player.rank >= 6 ? "Tiên Nguyên" : "Chân Nguyên",
                value: player.rank >= 6 ? Math.round(displayedMana).toLocaleString("vi-VN") : `${Math.round(displayedMana)}/${Math.round(displayedMaxMana)}`,
                percent: player.rank >= 6 ? 100 : displayedManaPercent,
                regen: manaRegenRealHourly,
                current: displayedMana,
                max: displayedMaxMana,
                uncapped: player.rank >= 6,
                color: "from-sky-500 to-cyan-300",
                textColor: "text-cyan-200",
              },
              {
                label: "Niệm Lực",
                value: `${(player.current_thought ?? 0).toFixed(1)}/${(player.max_thought ?? 100).toFixed(0)}`,
                percent: displayedThoughtPercent,
                regen: thoughtRegenRealHourly,
                current: player.current_thought ?? 0,
                max: player.max_thought ?? 100,
                uncapped: false,
                color: "from-violet-500 to-fuchsia-300",
                textColor: "text-violet-200",
              },
              {
                label: "Linh Hồn",
                value: `${player.soul_hp}/${player.max_soul}`,
                percent: displayedSoulPercent,
                regen: soulRegenRealHourly,
                current: player.soul_hp,
                max: player.max_soul,
                uncapped: false,
                color: "from-indigo-500 to-purple-300",
                textColor: "text-indigo-200",
              },
            ].map((resource) => {
              const isFull = resource.current >= resource.max;
              const regenText = resource.regen > 0 && (!isFull || resource.label === "Tiên Nguyên")
                ? `+${resource.regen.toFixed(1)}/giờ`
                : "";
              const remainingText = !resource.uncapped && !isFull && resource.regen > 0
                ? ` • ${formatTimeRemaining(resource.current, resource.max, resource.regen)}`
                : "";

              return (
                <div
                  key={resource.label}
                  className="min-w-0 rounded-lg border border-white/5 bg-black/15 px-2.5 py-2"
                  title={`${resource.label}: ${resource.value}${regenText ? ` • Hồi ${regenText}${remainingText}` : ""}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-[10px] font-semibold text-zinc-300 sm:text-[11px]">{resource.label}</span>
                    <span className={`shrink-0 text-[10px] font-mono font-semibold sm:text-[11px] ${resource.textColor}`}>{resource.value}</span>
                  </div>
                  <div className="mt-1.5 h-2 overflow-hidden rounded-full border border-black/40 bg-zinc-950">
                    <div
                      className={`h-full rounded-full bg-gradient-to-r ${resource.color} transition-[width] duration-300`}
                      style={{ width: `${Math.max(0, Math.min(100, resource.percent))}%` }}
                    />
                  </div>
                  <div className="mt-1 h-3 text-right text-[9px] font-mono text-emerald-400/75">
                    {regenText}
                  </div>
                </div>
              );
            })}
          </div>
          {/* Right Block: Resources (Stacked vertically) */}
          <div className="relative z-10 grid grid-cols-2 md:flex md:flex-col justify-center items-center md:items-end gap-2 md:gap-3 w-full md:w-auto border-t md:border-t-0 border-zinc-900/60 pt-3 md:pt-0 shrink-0">
            {/* Primeval Stones */}
            <div className="flex flex-col items-center md:items-end rounded-xl border border-amber-900/25 bg-black/15 px-3 py-2">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Nguyên Thạch</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Coins className="w-4 h-4 text-yellow-500" />
                <span className="text-sm font-bold text-yellow-500/90 font-mono">
                  <Num value={player.primeval_stones || 0} />
                </span>
              </div>
            </div>

            {/* Immortal Stones */}
            <div className="flex flex-col items-center md:items-end rounded-xl border border-violet-900/25 bg-black/15 px-3 py-2">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Tiên Nguyên Thạch</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Sparkles className="w-4 h-4 text-violet-400" />
                <span className="text-sm font-bold text-purple-400 font-mono">
                  <Num value={player.immortal_stones || 0} />
                </span>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="flex sm:gap-6 w-full">
        <LogConsole logs={logs} />
      </div>

      {currentQuest && (
        <section
          aria-label="Lộ trình tu hành hiện tại"
          className="rounded-2xl border border-amber-500/40 bg-gradient-to-r from-amber-950/35 via-zinc-950/80 to-emerald-950/20 p-3.5 sm:p-4 shadow-[0_0_24px_rgba(234,179,8,0.08)]"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2 text-[10px] font-mono uppercase tracking-wider text-amber-400">
                <span>{currentQuest.chapter ? (STORY_CHAPTER_NAMES[currentQuest.chapter] || `Chặng ${currentQuest.chapter}`) : "Lộ Trình Tu Hành"}</span>
                <span className="text-zinc-500">•</span>
                <span className="text-emerald-400">{completedQuestCount}/{typedQuests.length} bước</span>
              </div>
              <h3 className="mt-1 font-serif text-sm sm:text-base font-bold text-amber-100">
                Mục tiêu: {currentQuest.title}
              </h3>
              {(currentQuest.guide_tip || currentQuest.desc) && (
                <p className="mt-1 text-xs leading-relaxed text-zinc-300">
                  {currentQuest.guide_tip || currentQuest.desc}
                </p>
              )}
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <span className="text-[11px] font-mono text-amber-300/90">
                  {currentQuest.req_desc}
                  {typeof currentQuest.target_count === "number" && currentQuest.target_count > 1
                    ? ` (${currentQuest.current_count ?? 0}/${currentQuest.target_count})`
                    : ""}
                </span>
                {currentQuest.rewards && currentQuest.rewards.length > 0 && (
                  <span className="text-[11px] font-mono text-emerald-400">
                    Thưởng: {formatRewards(currentQuest.rewards)}
                  </span>
                )}
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={() => {
                  setSelectedChapter(currentQuest.chapter || 1);
                  setRoadmapOpen(true);
                }}
                className="inline-flex min-h-9 items-center gap-1.5 rounded-xl border border-zinc-700 bg-zinc-900/90 px-3 py-1.5 text-xs font-semibold text-zinc-200 hover:border-amber-400/60 cursor-pointer"
              >
                <Layers className="h-3.5 w-3.5 text-amber-400" />
                <span>Xem 7 Chặng</span>
              </button>
              {currentQuest.can_claim ? (
                <button
                  type="button"
                  disabled={actionLoading}
                  onClick={() => {
                    void completeQuest(currentQuest.id);
                  }}
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-xl bg-gradient-to-r from-amber-500 to-yellow-500 px-3.5 py-1.5 text-xs font-bold text-zinc-950 shadow-[0_0_12px_rgba(234,179,8,0.3)] hover:from-amber-400 hover:to-yellow-400 cursor-pointer disabled:opacity-50"
                >
                  <Gift className="h-3.5 w-3.5" />
                  <span>Nhận thưởng</span>
                </button>
              ) : (
                currentQuest.target_route && (
                  <button
                    type="button"
                    onClick={() => {
                      const targetItem = entryById.get(currentQuest.target_route!);
                      if (targetItem) {
                        handleItemClick(targetItem);
                      } else {
                        setView(currentQuest.target_route!);
                      }
                    }}
                    className="inline-flex min-h-9 items-center gap-1.5 rounded-xl bg-gradient-to-r from-amber-500 to-yellow-500 px-3.5 py-1.5 text-xs font-bold text-zinc-950 shadow-[0_0_12px_rgba(234,179,8,0.25)] hover:from-amber-400 hover:to-yellow-400 cursor-pointer"
                  >
                    <span>Thực hiện ({LOBBY_ENTRY_LABELS[currentQuest.target_route] || currentQuest.target_route})</span>
                    <ChevronRight className="h-3.5 w-3.5" />
                  </button>
                )
              )}
            </div>
          </div>
        </section>
      )}

      <div className="flex flex-col gap-3 sm:gap-4">
        <div className="flex flex-wrap items-end justify-between gap-2 border-b border-[var(--border-default)]/60 pb-2">
          <div>
            <h2 className="font-serif text-base sm:text-lg font-bold tracking-wide text-[var(--text-primary,#f4e8cb)]">
              Điểm Đến Cổ Giới
            </h2>
            <p className="mt-0.5 text-[10px] sm:text-xs text-[var(--text-muted,#ad9878)]">
              Chọn một nơi để tiếp tục hành trình
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => {
                setAllFeaturesQuery("");
                setAllFeaturesOpen(true);
              }}
              className="inline-flex min-h-9 items-center gap-1.5 rounded-full border border-[var(--border-default)]/80 bg-black/25 px-3 py-1 text-[11px] font-semibold text-[var(--text-primary,#f4e8cb)] transition hover:border-[var(--mmo-menu-accent,#e4b96c)] hover:bg-black/40 cursor-pointer"
            >
              <LayoutGrid className="h-3.5 w-3.5 text-[var(--mmo-menu-accent,#e4b96c)]" />
              <span>Xem tất cả ({LOBBY_FEATURE_IDS.length})</span>
            </button>
            <button
              type="button"
              onClick={() => setGridSettingsOpen(true)}
              className="inline-flex min-h-9 items-center gap-1.5 rounded-full border border-[var(--mmo-menu-border-soft,rgba(168,137,82,0.45))] bg-[var(--mmo-theme-elevated,#352618)] px-3 py-1 text-[11px] font-semibold text-[var(--text-primary,#f4e8cb)] transition hover:border-[var(--mmo-menu-accent,#e4b96c)] hover:brightness-110 cursor-pointer"
            >
              <SlidersHorizontal className="h-3.5 w-3.5 text-[var(--mmo-menu-accent,#e4b96c)]" />
              <span>Cài đặt lưới</span>
            </button>
          </div>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6 gap-2 sm:gap-3">
          {visibleLobbyItems.map((item) => {
            const Icon = item.icon;
            const isTarget = !item.locked && currentQuest?.target_route === item.id;
            return (
              <button
                type="button"
                key={item.id}
                onClick={() => handleItemClick(item)}
                aria-label={item.locked ? `${item.label} (chưa mở khóa)` : item.label}
                title={item.locked ? `${item.label} — ${item.lockLabel || "chưa mở khóa"}` : item.label}
                className={`mmo-game-tile group relative flex min-h-[92px] items-center gap-3 overflow-hidden rounded-xl border px-3 py-2.5 text-left transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] duration-200 sm:min-h-[108px] sm:px-3.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400/70 ${
                  item.locked
                    ? "opacity-60 border-dashed"
                    : isTarget
                    ? "border-amber-400/80 ring-1 ring-amber-400/40 shadow-[0_0_16px_rgba(234,179,8,0.18)]"
                    : ""
                }`}
                data-locked={item.locked || undefined}
              >
                {item.artwork && (
                  <>
                    <img src={item.artwork} alt="" aria-hidden="true" loading="lazy" className="pointer-events-none absolute inset-y-0 right-0 h-full w-[58%] object-cover object-center opacity-35 transition duration-300 group-hover:scale-105 group-hover:opacity-50" />
                    <span aria-hidden="true" className="pointer-events-none absolute inset-0 bg-gradient-to-r from-[var(--mmo-menu-bg,#211810)]/58 via-[var(--mmo-menu-bg,#211810)]/30 to-transparent" />
                    <span aria-hidden="true" className="pointer-events-none absolute inset-x-0 bottom-0 h-px bg-gradient-to-r from-amber-300/45 via-amber-200/10 to-transparent" />
                  </>
                )}
                <div
                  className={`mmo-game-tile-icon relative z-[1] grid h-10 w-10 shrink-0 place-items-center rounded-xl border shadow-md transition-transform duration-200 sm:h-12 sm:w-12 sm:rounded-2xl ${
                    item.locked
                      ? "border-zinc-800 bg-zinc-950"
                      : "group-hover:scale-105"
                  }`}
                >
                  <Icon className={`h-5 w-5 sm:h-6 sm:w-6 ${item.locked ? "text-zinc-600" : "text-[var(--mmo-menu-accent,#e4b96c)]"}`} />
                  {item.locked && (
                    <div className="absolute -bottom-1 -right-1 rounded-full border border-zinc-800 bg-zinc-950 p-1">
                      <Lock className="h-2.5 w-2.5 text-zinc-500" />
                    </div>
                  )}
                </div>
                <div className="relative z-[1] min-w-0 flex-1">
                  <span className={`block font-serif text-[11px] font-semibold leading-snug tracking-wide transition-colors sm:text-xs ${
                    item.locked
                      ? "text-zinc-500"
                      : "text-[var(--text-secondary)] group-hover:text-[var(--text-primary)]"
                  }`}>
                    {item.label}
                  </span>
                  {item.isGroup && (
                    <span className="mt-0.5 block text-[10px] text-[var(--text-muted,#ad9878)]">
                      Nhóm menu • {item.childCount} mục
                    </span>
                  )}
                </div>
                {!item.locked && <ChevronRight className="relative z-[1] h-4 w-4 shrink-0 text-amber-500/65 transition-transform group-hover:translate-x-0.5 group-hover:text-amber-300" />}
              </button>
            );
          })}
        </div>
      </div>
      <div className="flex justify-end border-t border-zinc-900/60 pt-3">
        <button
          type="button"
          onClick={() => setCombatSettingsOpen(true)}
          className="flex min-h-11 items-center gap-2 rounded-xl border border-[var(--mmo-menu-border-soft,rgba(168,137,82,0.4))] bg-[var(--mmo-theme-elevated,#352618)] px-4 py-2 text-xs font-semibold text-[var(--text-primary,#f4e8cb)] transition hover:border-[var(--mmo-menu-accent,#e4b96c)] hover:brightness-110"
        >
          <Settings2 className="h-4 w-4 text-[var(--mmo-menu-accent,#e4b96c)]" />
          Cài đặt
        </button>
      </div>
      {combatSettingsOpen && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/75 p-3 backdrop-blur-sm sm:p-6">
          <section
            ref={combatSettingsDialogRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="lobby-settings-title"
            tabIndex={-1}
            className="flex max-h-[90dvh] w-full max-w-2xl flex-col overflow-hidden overscroll-contain rounded-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 id="lobby-settings-title" className="font-serif text-base font-bold text-amber-100">Cài Đặt</h2>
                <p className="text-[10px] text-zinc-500">Menu lobby, giao diện và chiến thuật turn-based</p>
              </div>
              <button
                type="button"
                aria-label="Đóng cài đặt Lobby"
                onClick={() => setCombatSettingsOpen(false)}
                className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-amber-300"
              >
                <X className="h-4 w-4" />
              </button>
            </header>
            <div className="overflow-y-auto p-3 sm:p-5">
                <LobbySettingsPanel />
            </div>
          </section>
        </div>
      )}
      {activeGroup && (
        <div
          className="fixed inset-0 z-[100] flex items-end justify-center bg-black/75 p-0 backdrop-blur-sm sm:items-center sm:p-6"
          onClick={(event) => {
            if (event.target === event.currentTarget) setActiveGroupId(null);
          }}
        >
          <section
            ref={groupDialogRef}
            onKeyDown={groupDialogKeyDown}
            role="dialog"
            aria-modal="true"
            aria-labelledby="lobby-group-modal-title"
            tabIndex={-1}
            className="flex max-h-[86dvh] w-full max-w-xl flex-col overflow-hidden overscroll-contain rounded-t-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl sm:rounded-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 id="lobby-group-modal-title" className="font-serif text-base font-bold text-amber-100">
                  {activeGroup.label}
                </h2>
                <p className="text-[11px] text-zinc-500">{activeGroup.desc}</p>
              </div>
              <button
                type="button"
                aria-label="Đóng nhóm menu"
                onClick={() => setActiveGroupId(null)}
                className="inline-flex min-h-10 min-w-10 items-center justify-center rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </header>
            <div className="grid grid-cols-1 gap-2.5 overflow-y-auto p-4 sm:grid-cols-2 sm:p-5">
              {activeGroup.children.map((childId) => {
                const child = entryById.get(childId);
                if (!child) return null;
                const ChildIcon = child.icon;
                const isPinned = lobbyGridItems.includes(child.id);
                return (
                  <div
                    key={child.id}
                    className="flex items-center justify-between gap-2 rounded-xl border border-zinc-800 bg-zinc-900/70 p-2.5"
                  >
                    <button
                      type="button"
                      onClick={() => handleItemClick(child)}
                      className={`flex min-w-0 flex-1 items-center gap-2.5 text-left cursor-pointer ${
                        child.locked ? "opacity-55 hover:opacity-80" : "hover:text-amber-200"
                      }`}
                    >
                      <div className="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-zinc-700/70 bg-zinc-950">
                        {child.locked ? <Lock className="h-4 w-4 text-zinc-500" /> : <ChildIcon className="h-4 w-4 text-amber-300" />}
                      </div>
                      <div className="min-w-0">
                        <span className="block truncate font-serif text-xs font-semibold text-zinc-100">
                          {child.label}
                        </span>
                        {child.locked && child.lockLabel && (
                          <span className="block truncate text-[10px] text-zinc-500">{child.lockLabel}</span>
                        )}
                      </div>
                    </button>
                    <button
                      type="button"
                      onClick={() => togglePinFeature(child.id)}
                      className={`inline-flex min-h-8 shrink-0 items-center gap-1 rounded-lg border px-2 py-1 text-[11px] font-semibold transition cursor-pointer ${
                        isPinned
                          ? "border-amber-400/60 bg-amber-950/50 text-amber-200"
                          : "border-zinc-700 bg-zinc-950 text-zinc-400 hover:border-zinc-500 hover:text-zinc-200"
                      }`}
                    >
                      <Pin className="h-3 w-3" />
                      <span>{isPinned ? "Đã ghim" : "Ghim"}</span>
                    </button>
                  </div>
                );
              })}
            </div>
          </section>
        </div>
      )}

      {allFeaturesOpen && (
        <div
          className="fixed inset-0 z-[100] flex items-end justify-center bg-black/75 p-0 backdrop-blur-sm sm:items-center sm:p-6"
          onClick={(event) => {
            if (event.target === event.currentTarget) setAllFeaturesOpen(false);
          }}
        >
          <section
            ref={allFeaturesDialogRef}
            onKeyDown={allFeaturesDialogKeyDown}
            role="dialog"
            aria-modal="true"
            aria-labelledby="lobby-all-features-title"
            tabIndex={-1}
            className="flex max-h-[88dvh] w-full max-w-2xl flex-col overflow-hidden overscroll-contain rounded-t-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl sm:rounded-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 id="lobby-all-features-title" className="font-serif text-base font-bold text-amber-100">
                  Tất Cả Chức Năng ({LOBBY_FEATURE_IDS.length})
                </h2>
                <p className="text-[11px] text-zinc-500">Truy cập nhanh mọi tính năng dù đã ẩn khỏi Lưới Lobby</p>
              </div>
              <button
                type="button"
                aria-label="Đóng danh sách chức năng"
                onClick={() => setAllFeaturesOpen(false)}
                className="inline-flex min-h-10 min-w-10 items-center justify-center rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </header>
            <div className="border-b border-zinc-800 px-4 py-2.5 sm:px-5">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
                <input
                  type="text"
                  value={allFeaturesQuery}
                  onChange={(event) => setAllFeaturesQuery(event.target.value)}
                  placeholder="Tìm nhanh tính năng..."
                  className="w-full rounded-xl border border-zinc-800 bg-zinc-900 py-2 pl-9 pr-3 text-xs text-zinc-100 placeholder:text-zinc-500 focus:border-amber-500/60 focus:outline-none"
                />
              </div>
            </div>
            <div className="space-y-4 overflow-y-auto p-4 sm:p-5">
              {[
                { title: "Cốt Lõi", ids: CORE_LOBBY_FEATURE_IDS as readonly string[] },
                ...LOBBY_MENU_GROUPS.map((group) => ({ title: group.label, ids: group.children })),
              ].map((section) => {
                const q = allFeaturesQuery.trim().toLowerCase();
                const matched = section.ids
                  .map((id) => entryById.get(id))
                  .filter((item): item is HubItem => Boolean(item && (!q || item.label.toLowerCase().includes(q))));
                if (matched.length === 0) return null;
                return (
                  <div key={section.title}>
                    <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-amber-300/80">
                      {section.title}
                    </h3>
                    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                      {matched.map((item) => {
                        const ItemIcon = item.icon;
                        return (
                          <button
                            key={item.id}
                            type="button"
                            disabled={item.locked}
                            onClick={() => handleItemClick(item)}
                            className={`flex items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900/70 px-2.5 py-2 text-left text-xs font-semibold transition ${
                              item.locked
                                ? "cursor-not-allowed opacity-50 text-zinc-500"
                                : "cursor-pointer text-zinc-200 hover:border-amber-500/50 hover:bg-zinc-900"
                            }`}
                          >
                            <ItemIcon className="h-4 w-4 shrink-0 text-amber-300" />
                            <span className="truncate">{item.label}</span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          </section>
        </div>
      )}

      {gridSettingsOpen && (
        <div
          className="fixed inset-0 z-[100] flex items-end justify-center bg-black/75 p-0 backdrop-blur-sm sm:items-center sm:p-6"
          onClick={(event) => {
            if (event.target === event.currentTarget) setGridSettingsOpen(false);
          }}
        >
          <section
            ref={gridSettingsDialogRef}
            onKeyDown={gridSettingsDialogKeyDown}
            role="dialog"
            aria-modal="true"
            aria-labelledby="lobby-grid-settings-title"
            tabIndex={-1}
            className="flex max-h-[90dvh] w-full max-w-2xl flex-col overflow-hidden overscroll-contain rounded-t-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl sm:rounded-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 id="lobby-grid-settings-title" className="font-serif text-base font-bold text-amber-100">
                  Cài Đặt Lưới Lobby
                </h2>
                <p className="text-[11px] text-zinc-500">Chỉnh số lượng, thứ tự và ghim tính năng ra Lobby</p>
              </div>
              <button
                type="button"
                aria-label="Đóng cài đặt lưới Lobby"
                onClick={() => setGridSettingsOpen(false)}
                className="inline-flex min-h-10 min-w-10 items-center justify-center rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </header>
            <div className="overflow-y-auto p-4 sm:p-5">
              <LobbyGridSettingsEditor />
            </div>
          </section>
        </div>
      )}

      {roadmapOpen && (
        <div
          className="fixed inset-0 z-[105] flex items-end justify-center bg-black/75 p-0 backdrop-blur-sm sm:items-center sm:p-6"
          onClick={(event) => {
            if (event.target === event.currentTarget) setRoadmapOpen(false);
          }}
        >
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="roadmap-modal-title"
            className="flex max-h-[88dvh] w-full max-w-4xl flex-col overflow-hidden rounded-t-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl sm:rounded-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 id="roadmap-modal-title" className="font-serif text-base font-bold text-amber-100">
                  Lộ Trình Tu Hành (Nhất Chuyển → Cửu Chuyển Cổ Tôn)
                </h2>
                <p className="text-[11px] text-zinc-400">
                  Tiến độ: {completedQuestCount}/{typedQuests.length} bước
                </p>
              </div>
              <button
                type="button"
                aria-label="Đóng lộ trình"
                onClick={() => setRoadmapOpen(false)}
                className="inline-flex min-h-10 min-w-10 items-center justify-center rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </header>
            <div className="grid grid-cols-1 gap-4 overflow-y-auto p-4 sm:grid-cols-[240px_1fr] sm:p-5">
              <div className="space-y-2">
                {[1, 2, 3, 4, 5, 6, 7].map((ch) => {
                  const chQuests = typedQuests.filter((q) => (q.chapter || 1) === ch);
                  const chDone = chQuests.length > 0 && chQuests.every((q) => q.status === "COMPLETED");
                  const isCurrentCh = currentQuest?.chapter === ch;
                  return (
                    <button
                      key={ch}
                      type="button"
                      onClick={() => setSelectedChapter(ch)}
                      className={`w-full rounded-xl border px-3 py-2.5 text-left transition cursor-pointer ${
                        selectedChapter === ch
                          ? "border-amber-500/70 bg-amber-950/35 text-amber-100"
                          : "border-zinc-800 bg-zinc-900/60 text-zinc-300 hover:border-zinc-700"
                      }`}
                    >
                      <div className="text-xs font-serif font-bold">{STORY_CHAPTER_NAMES[ch]}</div>
                      <div className="mt-0.5 text-[10px] font-mono text-zinc-400">
                        {chDone ? "✓ Đã hoàn thành" : isCurrentCh ? "★ Đang thực hiện" : `${chQuests.filter((q) => q.status === "COMPLETED").length}/${chQuests.length} bước`}
                      </div>
                    </button>
                  );
                })}
              </div>
              <div className="space-y-2.5">
                {typedQuests
                  .filter((q) => (q.chapter || 1) === selectedChapter)
                  .map((q) => {
                    const isDone = q.status === "COMPLETED";
                    const isCurrent = currentQuest?.id === q.id;
                    return (
                      <div
                        key={q.id}
                        className={`rounded-xl border p-3 ${
                          isDone
                            ? "border-emerald-900/50 bg-emerald-950/15 opacity-75"
                            : isCurrent
                            ? "border-amber-500/60 bg-amber-950/20"
                            : "border-zinc-800 bg-zinc-900/50"
                        }`}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className={`font-serif text-sm font-bold ${isDone ? "text-emerald-400 line-through" : "text-zinc-100"}`}>
                            {isDone ? "✓ " : isCurrent ? "★ " : ""}{q.title}
                          </span>
                          <span className="text-[10px] font-mono text-zinc-400">{q.req_desc}</span>
                        </div>
                        <p className="mt-1 text-xs leading-relaxed text-zinc-300">{q.guide_tip || q.desc}</p>
                        <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
                          <span className="text-[10px] font-mono text-emerald-400">
                            {q.rewards?.length ? `Thưởng: ${formatRewards(q.rewards)}` : ""}
                          </span>
                          {!isDone && q.can_claim && (
                            <button
                              type="button"
                              disabled={actionLoading}
                              onClick={() => void completeQuest(q.id)}
                              className="rounded-lg bg-amber-500 px-3 py-1 text-[11px] font-bold text-zinc-950 hover:bg-amber-400 cursor-pointer"
                            >
                              Nhận thưởng
                            </button>
                          )}
                          {!isDone && !q.can_claim && isCurrent && q.target_route && (
                            <button
                              type="button"
                              onClick={() => {
                                setRoadmapOpen(false);
                                const targetItem = entryById.get(q.target_route!);
                                if (targetItem) handleItemClick(targetItem);
                                else setView(q.target_route!);
                              }}
                              className="rounded-lg border border-amber-500/50 bg-amber-950/40 px-3 py-1 text-[11px] font-semibold text-amber-200 hover:bg-amber-900/40 cursor-pointer"
                            >
                              Đi tới →
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
              </div>
            </div>
          </section>
        </div>
      )}

      {lockedModalItem && (
        <div
          className="fixed inset-0 z-[110] flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm"
          onClick={(event) => {
            if (event.target === event.currentTarget) setLockedModalItem(null);
          }}
        >
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="locked-feature-title"
            className="w-full max-w-md rounded-2xl border border-amber-900/60 bg-zinc-950 p-5 shadow-2xl"
          >
            <div className="flex items-center justify-between gap-2">
              <h2 id="locked-feature-title" className="flex items-center gap-2 font-serif text-base font-bold text-amber-200">
                <Lock className="h-4 w-4 text-amber-400" />
                <span>{lockedModalItem.label} (Chưa mở khóa)</span>
              </h2>
              <button
                type="button"
                aria-label="Đóng"
                onClick={() => setLockedModalItem(null)}
                className="rounded-lg border border-zinc-800 p-1.5 text-zinc-400 hover:text-white cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-3 text-xs leading-relaxed text-zinc-300">
              {lockedModalItem.lockDesc || lockedModalItem.desc || "Tính năng này sẽ mở khóa khi bạn tiến xa hơn trên Lộ Trình Tu Hành."}
            </p>
            <div className="mt-3 rounded-xl border border-zinc-800 bg-zinc-900/70 px-3 py-2 text-xs text-amber-300 font-mono">
              Điều kiện mở: {lockedModalItem.lockLabel || "Hoàn thành Lộ Trình Tu Hành tương ứng"}
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setLockedModalItem(null)}
                className="rounded-xl border border-zinc-700 px-3.5 py-1.5 text-xs text-zinc-300 hover:bg-zinc-900 cursor-pointer"
              >
                Đóng
              </button>
              {currentQuest?.target_route && (
                <button
                  type="button"
                  onClick={() => {
                    const route = currentQuest.target_route!;
                    setLockedModalItem(null);
                    setActiveGroupId(null);
                    const targetItem = entryById.get(route);
                    if (targetItem && !targetItem.locked) handleItemClick(targetItem);
                    else setView(route);
                  }}
                  className="rounded-xl bg-gradient-to-r from-amber-500 to-yellow-500 px-3.5 py-1.5 text-xs font-bold text-zinc-950 hover:from-amber-400 hover:to-yellow-400 cursor-pointer"
                >
                  Đi tới bước hiện tại →
                </button>
              )}
            </div>
          </section>
        </div>
      )}

      <SpatialRealmsModal />
    </div>
  );
};
