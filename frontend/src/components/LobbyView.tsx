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
} from "lucide-react";
import { LogConsole } from "./LogConsole";
import { Num } from "./Num";
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
  } = useGameStore();
  const [manaNow, setManaNow] = useState(() => Date.now());

  useEffect(() => {
    if (player && player.rank >= 6 && !aperture) {
      void fetchAperture();
    }
  }, [player, aperture, fetchAperture]);

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
      id: "mine",
      label: "Mỏ Linh Thạch",
      icon: Coins,
      gradient: "from-yellow-500 to-amber-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(234,179,8,0.45)]",
      borderColor: "border-yellow-400/30",
      iconColor: "text-yellow-500",
    },
    {
      id: "dungeon",
      label: "Mộng Cảnh & Bí Cảnh",
      icon: Gem,
      gradient: "from-purple-500 to-fuchsia-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(168,85,247,0.45)]",
      borderColor: "border-purple-400/30",
      iconColor: "text-purple-500",
      //locked: true, // Locked for dev
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
      id: "world-map",
      label: "Ngũ Vực",
      icon: Compass,
      gradient: "from-teal-500 to-emerald-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(20,184,166,0.45)]",
      borderColor: "border-teal-400/30",
      iconColor: "text-teal-500",
      //locked: true, // Locked for dev
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
    setView(item.id);
  };

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
      "Thập ???",
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
    <div className="w-full mx-auto px-2 sm:px-4 py-2 sm:py-4 flex flex-col gap-4 sm:gap-8">
      {/* 1. CHARACTER & RESOURCE HUD */}
      {player && (
        <div className="bg-[var(--bg-surface)] border border-[var(--border-default)] rounded-2xl sm:rounded-3xl p-3 sm:p-5 flex flex-col md:flex-row items-center justify-between gap-3 sm:gap-6 shadow-xl">
          {/* Left Block: Avatar & Name & Tu vi (Reduced width & wrapped text) */}
          <div className="flex items-center gap-3 sm:gap-4 w-full md:w-48 shrink-0">
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
            </div>
          </div>

          {/* Center Block: HP, Mana, Thought & Soul Bars (Optimized to 1 line each) */}
          <div className="flex flex-col gap-2 w-full md:w-[380px] grow max-w-md">
            {/* HP Bar */}
            <div
              className="flex items-center justify-between text-[11px] font-mono text-zinc-400 gap-2 hover:bg-zinc-900/10 p-0.5 rounded cursor-help transition-all"
              title={player && hpRegenRealHourly > 0 && displayedHP < displayedMaxHP ? `Hồi máu: +${hpRegenRealHourly.toFixed(1)}/giờ • ${formatTimeRemaining(displayedHP, displayedMaxHP, hpRegenRealHourly)}` : "Máu đầy"}
            >
              <div className="w-24 sm:w-28 flex items-center justify-between shrink-0 min-w-0">
                <span className="font-semibold text-zinc-300">Máu</span>
                {player && hpRegenRealHourly > 0 && displayedHP < displayedMaxHP && (
                  <span className="text-[9px] text-emerald-500/80 font-semibold">
                    +{hpRegenRealHourly.toFixed(1)}/g
                  </span>
                )}
              </div>
              <div className="flex-1 bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden relative">
                <div
                  className="bg-gradient-to-r from-red-600 to-rose-500 h-full rounded-full transition-all duration-300"
                  style={{ width: `${displayedHPPercent}%` }}
                />
              </div>
              <span className="w-20 text-right text-[10.5px] text-zinc-350 shrink-0 font-medium font-mono">
                {Math.round(displayedHP)}/{Math.round(displayedMaxHP)}
              </span>
            </div>

            {/* Mana Bar */}
            <div
              className="flex items-center justify-between text-[11px] font-mono text-zinc-400 gap-2 hover:bg-zinc-900/10 p-0.5 rounded cursor-help transition-all"
              title={player && manaRegenRealHourly > 0 && (player.rank >= 6 || displayedMana < displayedMaxMana) ? `${player.rank >= 6 ? 'Hồi Tiên Nguyên' : 'Hồi Chân Nguyên'}: +${manaRegenRealHourly.toFixed(1)}/giờ${player.rank < 6 ? ` • ${formatTimeRemaining(displayedMana, displayedMaxMana, manaRegenRealHourly)}` : ""}` : "Năng lượng đầy"}
            >
              <div className="w-24 sm:w-28 flex items-center justify-between shrink-0 min-w-0">
                <span className="font-semibold text-zinc-300 truncate">
                  {player.rank >= 6 ? "Tiên Nguyên" : "Chân Nguyên"}
                </span>
                {player && manaRegenRealHourly > 0 && (player.rank >= 6 || displayedMana < displayedMaxMana) && (
                  <span className="text-[9px] text-emerald-500/80 font-semibold">
                    +{manaRegenRealHourly.toFixed(1)}/g
                  </span>
                )}
              </div>
              <div className="flex-1 bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden relative">
                <div
                  className="bg-gradient-to-r from-cyan-600 to-blue-500 h-full rounded-full transition-all duration-300"
                  style={{ width: `${player.rank >= 6 ? 100 : displayedManaPercent}%` }}
                />
              </div>
              <span className="w-20 text-right text-[10.5px] text-zinc-350 shrink-0 font-medium font-mono">
                {player.rank >= 6
                  ? `${Math.round(displayedMana)}`
                  : `${Math.round(displayedMana)}/${Math.round(displayedMaxMana)}`}
              </span>
            </div>

            {/* Thought Bar */}
            <div
              className="flex items-center justify-between text-[11px] font-mono text-zinc-400 gap-2 hover:bg-zinc-900/10 p-0.5 rounded cursor-help transition-all"
              title={player && thoughtRegenRealHourly > 0 && (player.current_thought < player.max_thought) ? `Hồi Niệm Lực: +${thoughtRegenRealHourly.toFixed(1)}/giờ • ${formatTimeRemaining(player.current_thought, player.max_thought, thoughtRegenRealHourly)}` : "Niệm Lực đầy"}
            >
              <div className="w-24 sm:w-28 flex items-center justify-between shrink-0 min-w-0">
                <span className="font-semibold text-zinc-300">Niệm Lực</span>
                {player && thoughtRegenRealHourly > 0 && player.current_thought < player.max_thought && (
                  <span className="text-[9px] text-emerald-500/80 font-semibold">
                    +{thoughtRegenRealHourly.toFixed(1)}/g
                  </span>
                )}
              </div>
              <div className="flex-1 bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden relative">
                <div
                  className="bg-gradient-to-r from-violet-600 to-purple-500 h-full rounded-full transition-all duration-300"
                  style={{ width: `${displayedThoughtPercent}%` }}
                />
              </div>
              <span className="w-20 text-right text-[10.5px] text-zinc-350 shrink-0 font-medium font-mono">
                {(player.current_thought ?? 0).toFixed(1)}/{(player.max_thought ?? 100).toFixed(0)}
              </span>
            </div>

            {/* Soul Bar */}
            <div
              className="flex items-center justify-between text-[11px] font-mono text-zinc-400 gap-2 hover:bg-zinc-900/10 p-0.5 rounded cursor-help transition-all"
              title={player && soulRegenRealHourly > 0 && player.soul_hp < player.max_soul ? `Hồi Linh Hồn: +${soulRegenRealHourly.toFixed(1)}/giờ • ${formatTimeRemaining(player.soul_hp, player.max_soul, soulRegenRealHourly)}` : "Linh Hồn đầy"}
            >
              <div className="w-24 sm:w-28 flex items-center justify-between shrink-0 min-w-0">
                <span className="font-semibold text-zinc-300">Linh Hồn</span>
                {player && soulRegenRealHourly > 0 && player.soul_hp < player.max_soul && (
                  <span className="text-[9px] text-emerald-500/80 font-semibold">
                    +{soulRegenRealHourly.toFixed(1)}/g
                  </span>
                )}
              </div>
              <div className="flex-1 bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden relative">
                <div
                  className="bg-gradient-to-r from-indigo-600 to-purple-500 h-full rounded-full transition-all duration-300"
                  style={{ width: `${displayedSoulPercent}%` }}
                />
              </div>
              <span className="w-20 text-right text-[10.5px] text-zinc-350 shrink-0 font-medium font-mono">
                {player.soul_hp}/{player.max_soul}
              </span>
            </div>
          </div>
          {/* Right Block: Resources (Stacked vertically) */}
          <div className="flex flex-col justify-center items-center md:items-end gap-3 w-full md:w-auto border-t md:border-t-0 border-zinc-900/60 pt-3 md:pt-0 shrink-0">
            {/* Primeval Stones */}
            <div className="flex flex-col items-center md:items-end">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Nguyên Thạch</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Coins className="w-4 h-4 text-yellow-500" />
                <span className="text-sm font-bold text-yellow-500/90 font-mono">
                  <Num value={player.primeval_stones || 0} />
                </span>
              </div>
            </div>

            {/* Immortal Stones */}
            <div className="flex flex-col items-center md:items-end">
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

      {/* 2. Category: CHỨC NĂNG */}
      <div className="flex flex-col gap-3 sm:gap-5">
        <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
          <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-mono font-semibold">
            Chức năng chính
          </h2>
          <span className="text-[10px] text-zinc-600 font-mono">
            {managementItems.length} chức năng
          </span>
        </div>

        <div className="grid grid-cols-3 sm:grid-cols-5 gap-1.5 sm:gap-4 justify-center">
          {managementItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.id}
                onClick={() => handleItemClick(item)}
                className="flex flex-col items-center p-2 rounded-2xl transition-all duration-300 hover:bg-zinc-900/20 cursor-pointer group select-none"
              >
                <div
                  className={`w-14 h-14 sm:w-16 sm:h-16 md:w-20 md:h-20 rounded-full flex items-center justify-center bg-gradient-to-br ${item.gradient} border ${item.borderColor} transition-all duration-300 group-hover:scale-110 shadow-lg active:scale-95 relative ${item.glow}`}
                >
                  <Icon className={`w-6 h-6 sm:w-7 sm:h-7 md:w-8 md:h-8 ${item.iconColor} drop-shadow-[0_2px_4px_rgba(0,0,0,0.4)]`} />
                </div>
                <span className="font-serif text-[10px] sm:text-[11px] font-semibold text-zinc-400 mt-2 tracking-wide text-center group-hover:text-zinc-100 transition-colors">
                  {item.label}
                </span>
              </div>
            );
          })}
        </div>
      </div>
      {/* 3. Category: HOẠT ĐỘNG */}
      <div className="flex flex-col gap-3 sm:gap-5">
        <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
          <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-mono font-semibold">
            Hoạt động khác
          </h2>
          <span className="text-[10px] text-zinc-600 font-mono">
            {activityItems.length} hoạt động
          </span>
        </div>

        <div className="grid grid-cols-3 sm:grid-cols-6 gap-1.5 sm:gap-4 justify-center">
          {activityItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.id}
                onClick={() => handleItemClick(item)}
                className={`flex flex-col items-center p-2 rounded-2xl transition-all duration-300 select-none ${item.locked
                    ? "opacity-50 cursor-not-allowed"
                    : "hover:bg-zinc-900/20 cursor-pointer group"
                  }`}
              >
                <div
                  className={`w-14 h-14 sm:w-16 sm:h-16 md:w-20 md:h-20 rounded-full flex items-center justify-center ${item.locked
                      ? "bg-zinc-950 border border-zinc-900"
                      : `bg-gradient-to-br ${item.gradient} border ${item.borderColor} hover:border-zinc-400 dark:hover:border-zinc-800 transition-all duration-300 group-hover:scale-110 shadow-md active:scale-95`
                    } relative ${item.glow}`}
                >
                  {!item.locked && (
                    <div className="absolute inset-1 rounded-full border border-dashed border-white/40 dark:border-zinc-800/80" />
                  )}
                  <Icon className={`w-6 h-6 sm:w-7 sm:h-7 md:w-8 md:h-8 ${item.iconColor}`} />
                  {item.locked && (
                    <div className="absolute -bottom-1 -right-1 bg-zinc-900 border border-zinc-800 p-1 rounded-full">
                      <Lock className="w-2.5 h-2.5 text-zinc-500" />
                    </div>
                  )}
                </div>
                <span className={`font-serif text-[10px] sm:text-[11px] font-semibold mt-2 tracking-wide text-center transition-colors ${item.locked
                    ? "text-zinc-600"
                    : "text-[var(--text-secondary)] group-hover:text-[var(--text-primary)]"
                  }`}>
                  {item.label}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};
