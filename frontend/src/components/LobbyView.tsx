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
} from "lucide-react";
import { LogConsole } from "./LogConsole";
import { Num } from "./Num";
import { SpatialRealmsModal } from "./SpatialRealmsModal";
import { LobbySettingsPanel } from "./LobbySettingsPanel";
import { useUiPreferences } from "@/lib/ui-preferences";
import Link from "next/link";

interface HubItem {
  id: string;
  label: string;
  icon: React.ComponentType<ApiResponseData>;
  gradient: string;
  glow: string;
  borderColor: string;
  iconColor: string;
  locked?: boolean;
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
  } = useGameStore();
  const [manaNow, setManaNow] = useState(() => Date.now());
  const [combatSettingsOpen, setCombatSettingsOpen] = useState(false);
  const { lobbyItems, hiddenLobbyItems } = useUiPreferences();

  useEffect(() => {
    if (player && player.rank >= 6 && !aperture) {
      void fetchAperture();
    }
  }, [player, aperture, fetchAperture]);

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
      label: "Tiện Ích",
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
      label: "Khám Phá Dã Ngoại",
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

  const handleItemClick = (item: HubItem) => {
    if (item.locked) return;
    if (item.id === "divination") {
      openDivinationModal();
      return;
    }
    if (item.id === "spatial-realms") {
      setSpatialDirectoryOpen(true);
      return;
    }
    setView(item.id);
  };

  const orderLobbyItems = (items: HubItem[]) => items
    .filter((item) => !hiddenLobbyItems.includes(item.id))
    .sort((a, b) => lobbyItems.indexOf(a.id) - lobbyItems.indexOf(b.id));
  const visibleLobbyItems = orderLobbyItems([...managementItems, ...activityItems]);

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
        backgroundImage: "linear-gradient(rgba(18, 13, 9, 0.94), rgba(18, 13, 9, 0.97)), url('/assets/world/great-regions-atlas.png')",
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

      <div className="flex flex-col gap-3 sm:gap-4">
        <div className="flex items-end justify-between border-b border-[var(--border-default)]/60 pb-2">
          <div>
            <h2 className="font-serif text-base sm:text-lg font-bold tracking-wide text-[var(--text-primary,#f4e8cb)]">
              Điểm Đến Cổ Giới
            </h2>
            <p className="mt-0.5 text-[10px] sm:text-xs text-[var(--text-muted,#ad9878)]">
              Chọn một nơi để tiếp tục hành trình
            </p>
          </div>
          <span className="rounded-full border border-[var(--border-default)]/70 bg-black/20 px-2.5 py-1 text-[10px] text-[var(--text-muted,#ad9878)] font-mono">
            {visibleLobbyItems.length} chức năng
          </span>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6 gap-2 sm:gap-3">
          {visibleLobbyItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                type="button"
                key={item.id}
                disabled={item.locked}
                onClick={() => handleItemClick(item)}
                aria-label={item.locked ? `${item.label} (đã khóa)` : item.label}
                title={item.locked ? `${item.label} — chưa mở khóa` : item.label}
                className={`group relative flex min-h-[72px] items-center gap-3 overflow-hidden rounded-xl border px-3 py-2.5 text-left transition-all duration-200 sm:min-h-[82px] sm:px-3.5 ${
                  item.locked
                    ? "cursor-not-allowed border-[var(--border-default)]/40 bg-black/20 opacity-55"
                    : `cursor-pointer border-[var(--border-default)]/70 bg-[linear-gradient(145deg,rgba(60,43,27,0.72),rgba(30,23,17,0.86))] shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] hover:-translate-y-0.5 hover:border-amber-500/45 hover:bg-[linear-gradient(145deg,rgba(76,54,31,0.85),rgba(37,27,19,0.94))] hover:shadow-[0_8px_24px_rgba(0,0,0,0.22)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400/70 active:translate-y-0`
                }`}
              >
                <div
                  className={`relative grid h-10 w-10 shrink-0 place-items-center rounded-xl border shadow-md transition-transform duration-200 sm:h-12 sm:w-12 sm:rounded-2xl ${
                    item.locked
                      ? "border-zinc-800 bg-zinc-950"
                      : `bg-gradient-to-br ${item.gradient} ${item.borderColor} group-hover:scale-105 ${item.glow}`
                  }`}
                >
                  <Icon className={`h-5 w-5 sm:h-6 sm:w-6 ${item.iconColor}`} />
                  {item.locked && (
                    <div className="absolute -bottom-1 -right-1 rounded-full border border-zinc-800 bg-zinc-950 p-1">
                      <Lock className="h-2.5 w-2.5 text-zinc-500" />
                    </div>
                  )}
                </div>
                <span className={`min-w-0 flex-1 font-serif text-[11px] font-semibold leading-snug tracking-wide transition-colors sm:text-xs ${
                  item.locked
                    ? "text-zinc-500"
                    : "text-[var(--text-secondary)] group-hover:text-[var(--text-primary)]"
                }`}>
                  {item.label}
                </span>
                {!item.locked && <ChevronRight className="h-4 w-4 shrink-0 text-amber-500/45 transition-transform group-hover:translate-x-0.5 group-hover:text-amber-300" />}
              </button>
            );
          })}
        </div>
      </div>
      <div className="flex justify-end border-t border-zinc-900/60 pt-3">
        <button
          type="button"
          onClick={() => setCombatSettingsOpen(true)}
          className="flex items-center gap-2 rounded-xl border border-[var(--mmo-menu-border-soft,rgba(168,137,82,0.4))] bg-[var(--mmo-theme-elevated,#352618)] px-4 py-2 text-xs font-semibold text-[var(--text-primary,#f4e8cb)] transition hover:border-[var(--mmo-menu-accent,#e4b96c)] hover:brightness-110"
        >
          <Settings2 className="h-4 w-4 text-[var(--mmo-menu-accent,#e4b96c)]" />
          Cài đặt
        </button>
      </div>
      {combatSettingsOpen && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/75 p-3 backdrop-blur-sm sm:p-6">
          <section
            role="dialog"
            aria-modal="true"
            aria-label="Cài đặt Lobby"
            className="flex max-h-[90dvh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-amber-900/60 bg-zinc-950 shadow-2xl"
          >
            <header className="flex items-center justify-between border-b border-zinc-800 px-4 py-3 sm:px-5">
              <div>
                <h2 className="font-serif text-base font-bold text-amber-100">Cài Đặt</h2>
                <p className="text-[10px] text-zinc-500">Menu lobby, giao diện và chiến thuật turn-based</p>
              </div>
              <button
                type="button"
                aria-label="Đóng cài đặt Lobby"
                onClick={() => setCombatSettingsOpen(false)}
                className="rounded-lg border border-zinc-800 p-2 text-zinc-400 hover:bg-zinc-900 hover:text-white"
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
      <SpatialRealmsModal />
    </div>
  );
};
