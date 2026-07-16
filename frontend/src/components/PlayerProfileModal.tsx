"use client";

import React from "react";
import { useGameStore } from "@/store/useGameStore";
import { X, Eye, EyeOff, Flame, Coins, Sparkles, Crown, Award, Activity, Heart } from "lucide-react";

const pathNameMap: Record<string, string> = {
  wisdom: "Trí Đạo",
  strength: "Lực Đạo",
  sword: "Kiếm Đạo",
  dark: "Ám Đạo",
  light: "Quang Đạo",
  fire: "Hỏa Đạo",
  water: "Thủy Đạo",
  wind: "Phong Đạo",
  lightning: "Lôi Đạo",
  earth: "Thổ Đạo",
  time: "Thời Đạo",
  space: "Không Đạo",
  information: "Tín Đạo",
  refinement: "Luyện Đạo",
  luck: "Vận Đạo",
  soul: "Hồn Đạo",
  blood: "Huyết Đạo",
  dream: "Mộng Đạo",
  phantom: "Huyễn Đạo",
  transformation: "Biến Hóa Đạo",
  rule: "Luật Đạo",
  food: "Thực Đạo",
  qi: "Khí Đạo",
  painting: "Họa Đạo",
  heaven: "Thiên Đạo",
  human: "Nhân Đạo",
};

const translatePath = (path: string): string => {
  const normalized = path.toLowerCase().trim();
  if (pathNameMap[normalized]) return pathNameMap[normalized];
  return path.charAt(0).toUpperCase() + path.slice(1) + " Đạo";
};

export const PlayerProfileModal: React.FC = () => {
  const { inspectedProfile, inspectedUsernameQuery, clearInspectedProfile, guDictionary, player, initiateTrade } = useGameStore();

  if (!inspectedProfile) return null;

  const {
    username,
    rank,
    sub_rank,
    aptitude_type,
    aptitude_percent,
    current_mana,
    max_mana,
    primeval_stones,
    immortal_stones,
    hp,
    base_hp,
    sect_name,
    equipped_gus,
    scouting_status,
    status_type,
    dominant_path,
    dao_marks,
    bio,
    avatar_url,
  } = inspectedProfile;

  const isImmortal = rank >= 6;

  const getRankName = (rankNum: number, subNum: number): string => {
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
    ];
    const rName = rankNames[rankNum - 1] || `${rankNum} Chuyển`;
    const sName = ranks[subNum - 1] || `${subNum} Giai`;
    return `${rName} • ${sName}`;
  };

  // Styling configurations based on rank (Cổ Sư / Cổ Tiên) and scouting status
  let themeClass = "";
  let glowClass = "";
  let avatarBg = "";
  let ringColor = "";

  if (isImmortal) {
    if (status_type === "success") {
      themeClass = "from-violet-950/40 via-purple-950/20 to-zinc-950 border-purple-500/30";
      glowClass = "shadow-[0_0_35px_rgba(168,85,247,0.25)]";
      avatarBg = "from-violet-500 via-fuchsia-500 to-purple-600";
      ringColor = "border-purple-500/40";
    } else if (status_type === "partial") {
      themeClass = "from-indigo-950/40 via-cyan-950/15 to-zinc-950 border-indigo-500/30";
      glowClass = "shadow-[0_0_30px_rgba(99,102,241,0.2)]";
      avatarBg = "from-indigo-500 to-cyan-600";
      ringColor = "border-indigo-500/40";
    } else {
      themeClass = "from-rose-950/40 via-zinc-950 to-zinc-950 border-rose-500/30";
      glowClass = "shadow-[0_0_30px_rgba(239,68,68,0.15)]";
      avatarBg = "from-rose-500 to-amber-600";
      ringColor = "border-rose-500/40";
    }
  } else {
    if (status_type === "success") {
      themeClass = "from-emerald-950/45 via-teal-950/15 to-zinc-950 border-emerald-500/30";
      glowClass = "shadow-[0_0_30px_rgba(16,185,129,0.15)]";
      avatarBg = "from-emerald-500 to-teal-600";
      ringColor = "border-emerald-500/40";
    } else if (status_type === "partial") {
      themeClass = "from-cyan-950/40 via-blue-950/15 to-zinc-950 border-cyan-500/30";
      glowClass = "shadow-[0_0_30px_rgba(6,182,212,0.15)]";
      avatarBg = "from-cyan-500 to-blue-600";
      ringColor = "border-cyan-500/40";
    } else {
      themeClass = "from-rose-950/40 via-zinc-950 to-zinc-950 border-rose-500/30";
      glowClass = "shadow-[0_0_30px_rgba(239,68,68,0.15)]";
      avatarBg = "from-rose-500 to-amber-600";
      ringColor = "border-rose-500/40";
    }
  }

  let statusBadge = "bg-emerald-950/60 text-emerald-400 border-emerald-800/50";
  let statusIcon = <Eye className="w-4 h-4 text-emerald-400" />;

  if (status_type === "partial") {
    statusBadge = "bg-cyan-950/60 text-cyan-400 border-cyan-800/50";
    statusIcon = <Eye className="w-4 h-4 text-cyan-400" />;
  } else if (status_type === "failure") {
    statusBadge = "bg-rose-950/60 text-rose-400 border-rose-800/50";
    statusIcon = <EyeOff className="w-4 h-4 text-rose-400" />;
  }

  const getInitials = (name: string): string => {
    if (!name) return "?";
    return name.slice(0, 2).toUpperCase();
  };

  return (
    <div
      onClick={clearInspectedProfile}
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm transition-opacity animate-fade-in cursor-pointer"
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className={`w-full max-w-lg bg-zinc-950 border rounded-3xl p-6 relative overflow-hidden bg-gradient-to-br ${themeClass} ${glowClass} border shadow-2xl flex flex-col gap-6 animate-scale-up cursor-default`}
      >
        {/* Status Indicator Banner with Close Button */}
        <div className={`flex items-center justify-between gap-2.5 px-4 py-3 rounded-2xl border ${statusBadge} backdrop-blur-md relative`}>
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="shrink-0 animate-pulse">{statusIcon}</div>
            <p className="text-xs font-mono font-medium leading-relaxed">{scouting_status}</p>
          </div>
          <button
            onClick={clearInspectedProfile}
            className="p-1.5 rounded-lg bg-zinc-900/60 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 transition-colors border border-zinc-900/60 cursor-pointer shrink-0 z-10"
            title="Đóng"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>

        {/* Profile Header Details */}
        <div className="flex items-center gap-4 border-b border-zinc-900/60 pb-5">
          <div className="relative shrink-0">
            <div className={`w-16 h-16 rounded-full flex items-center justify-center font-bold text-lg border-2 border-zinc-900 bg-gradient-to-br ${avatarBg} text-white overflow-hidden`}>
              {avatar_url ? (
                <img src={avatar_url} alt={username} className="w-full h-full object-cover" />
              ) : (
                getInitials(username)
              )}
            </div>
            {/* Pulsing ring indicator */}
            <div
              className={`absolute -inset-1 rounded-full border border-dashed animate-spin ${ringColor}`}
              style={{ animationDuration: "12s" }}
            />
          </div>

          <div className="flex flex-col min-w-0 text-left">
            <h3 className="text-lg font-serif font-bold text-zinc-100 truncate flex items-center gap-1.5">
              {username}
              {status_type === "failure" && username === "Ẩn Danh Cổ Sư" && (
                <Crown className="w-4 h-4 text-amber-500 animate-bounce" />
              )}
            </h3>
            <div className="flex flex-wrap items-center gap-2 mt-0.5">
              <span className="text-xs text-zinc-500 font-mono font-semibold">
                {getRankName(rank, sub_rank)}
              </span>
              {/* For Cổ Sư, show dominant path in header if it is known */}
              {!isImmortal && dominant_path && dominant_path !== "Chưa rõ" && (
                <span className="text-[10px] px-2 py-0.5 rounded-md bg-emerald-950/40 text-emerald-400 border border-emerald-900/40 font-mono font-semibold">
                  {translatePath(dominant_path)}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Bio / Quote Section */}
        {bio && (
          <div className={`px-4 py-3 bg-zinc-900/10 border-l-2 ${
            isImmortal ? "border-purple-500/40" : "border-emerald-500/40"
          } text-zinc-400 italic text-xs rounded-r-xl font-mono leading-relaxed bg-zinc-950/20 text-left`}>
            "{bio}"
          </div>
        )}

        {/* Main Profile Info Section */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-left">
          {/* Sect Card */}
          <div className="flex flex-col gap-3.5 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4">
            <div className="flex items-center gap-2 text-zinc-500">
              <Crown className="w-4 h-4" />
              <span className="text-[10px] uppercase tracking-widest font-mono">Tông Môn</span>
            </div>
            <span className="text-sm font-semibold text-zinc-200 truncate">{sect_name}</span>
          </div>

          {/* Aptitude Card (For Cổ Sư) OR Dominant Path Card (For Cổ Tiên) */}
          {isImmortal ? (
            <div className="flex flex-col gap-3.5 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4">
              <div className="flex items-center gap-2 text-zinc-500">
                <Sparkles className="w-4 h-4 text-purple-400" />
                <span className="text-[10px] uppercase tracking-widest font-mono">Lưu Phái Chủ Đạo</span>
              </div>
              <span className="text-sm font-bold text-zinc-200">
                {dominant_path ? translatePath(dominant_path) : "Chưa rõ"}
              </span>
            </div>
          ) : (
            <div className="flex flex-col gap-3.5 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4">
              <div className="flex items-center gap-2 text-zinc-500">
                <Award className="w-4 h-4" />
                <span className="text-[10px] uppercase tracking-widest font-mono">Tư chất</span>
              </div>
              <span className="text-sm font-bold text-zinc-200">
                {aptitude_type === "???" ? "???" : `Tư chất ${aptitude_type} (${aptitude_percent}%)`}
              </span>
            </div>
          )}

          {/* Cultivator Pools (HP & Mana/Essence) */}
          <div className="sm:col-span-2 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4 flex flex-col gap-4">
            {/* HP */}
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between text-[10px] font-mono text-zinc-400">
                <span className="flex items-center gap-1">
                  <Heart className="w-3.5 h-3.5 text-rose-500" /> Sinh Mệnh
                </span>
                <span>
                  {base_hp === 0 || base_hp === -1 ? "???" : `${Math.round(hp)} / ${Math.round(base_hp)}`}
                </span>
              </div>
              <div className="w-full bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden">
                <div
                  className="bg-gradient-to-r from-red-600 to-rose-500 h-full rounded-full transition-all duration-300"
                  style={{ width: base_hp <= 0 ? "0%" : `${Math.min(100, (hp / base_hp) * 100)}%` }}
                />
              </div>
            </div>

            {/* mana (Chân Nguyên for Cổ Sư, Tiên Nguyên for Cổ Tiên) */}
            {isImmortal ? (
              <div className="flex items-center justify-between border-t border-zinc-900/60 pt-3">
                <div className="flex items-center gap-2 text-zinc-400 text-[10px] font-mono uppercase tracking-wider">
                  <Flame className="w-3.5 h-3.5 text-purple-400 animate-pulse" />
                  <span>Tiên Nguyên</span>
                </div>
                <span className="font-mono font-bold text-purple-400 text-sm">
                  {status_type === "failure" || status_type === "partial" ? "???" : `${Math.round(current_mana)} Hạt`}
                </span>
              </div>
            ) : (
              <div className="flex flex-col gap-1.5 border-t border-zinc-900/60 pt-3">
                <div className="flex items-center justify-between text-[10px] font-mono text-zinc-400">
                  <span className="flex items-center gap-1">
                    <Activity className="w-3.5 h-3.5 text-cyan-500" /> Chân Nguyên
                  </span>
                  <span>
                    {max_mana === 0 || max_mana === -1 ? "???" : `${Math.round(current_mana)} / ${Math.round(max_mana)}`}
                  </span>
                </div>
                <div className="w-full bg-zinc-950 rounded-full h-2 border border-zinc-900 overflow-hidden">
                  <div
                    className="bg-gradient-to-r from-cyan-600 to-blue-500 h-full rounded-full transition-all duration-300"
                    style={{ width: max_mana <= 0 ? "0%" : `${Math.min(100, (current_mana / max_mana) * 100)}%` }}
                  />
                </div>
              </div>
            )}
          </div>

          {/* Main Currency (Nguyên Thạch / Tiên Nguyên Thạch) */}
          {isImmortal ? (
            <div className="sm:col-span-2 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4 flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-lg bg-purple-550/10 flex items-center justify-center shrink-0">
                  <Coins className="w-4 h-4 text-purple-400" />
                </div>
                <span className="text-xs text-zinc-400 font-mono uppercase tracking-wider">Tiên Nguyên Thạch sở hữu</span>
              </div>
              <span className="font-mono font-bold text-purple-400 text-sm">
                {status_type === "failure" || status_type === "partial" ? (
                  "???"
                ) : (
                  `${Number(immortal_stones).toLocaleString(undefined, { maximumFractionDigits: 1 })} Viên`
                )}
              </span>
            </div>
          ) : (
            <div className="sm:col-span-2 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4 flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-lg bg-yellow-500/10 flex items-center justify-center shrink-0">
                  <Coins className="w-4 h-4 text-yellow-500" />
                </div>
                <span className="text-xs text-zinc-400 font-mono uppercase tracking-wider">Nguyên Thạch sở hữu</span>
              </div>
              <span className="font-mono font-bold text-yellow-500 text-sm">
                {status_type === "failure" || status_type === "partial" ? "???" : `${primeval_stones.toLocaleString()} Đá`}
              </span>
            </div>
          )}

          {/* Dao Marks (For Cổ Tiên) */}
          {isImmortal && (
            <div className="sm:col-span-2 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4 space-y-3">
              <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">Đạo Ngân tích lũy</span>
              <div className="flex flex-wrap gap-2 animate-fade-in">
                {status_type === "failure" || status_type === "partial" ? (
                  <span className="text-xs px-3 py-1.5 rounded-xl border border-dashed border-zinc-800 text-zinc-650 bg-zinc-950/20 font-mono">
                    Bị che giấu
                  </span>
                ) : Object.keys(dao_marks || {}).length > 0 ? (
                  Object.entries(dao_marks as Record<string, number>).map(([path, amount]) => {
                    if (amount <= 0) return null;
                    return (
                      <span
                        key={path}
                        className="text-xs px-3 py-1.5 rounded-xl border font-mono font-semibold flex items-center gap-1.5 bg-purple-950/20 text-purple-400 border-purple-900/40 hover:border-purple-700/60 transition-colors"
                      >
                        <Sparkles className="w-3 h-3 text-purple-400 shrink-0" />
                        {translatePath(path)}: +{amount.toLocaleString()}
                      </span>
                    );
                  })
                ) : (
                  <p className="text-xs text-zinc-600 italic">Chưa tích lũy Đạo Ngân nào</p>
                )}
              </div>
            </div>
          )}

          {/* Equipped Gu Bugs */}
          <div className="sm:col-span-2 bg-zinc-900/20 border border-zinc-900/60 rounded-2xl p-4 space-y-3">
            <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">Cổ trùng trang bị</span>
            <div className="flex flex-wrap gap-2">
              {equipped_gus && equipped_gus.length > 0 ? (
                equipped_gus.map((guId: string, index: number) => {
                  const spec = guDictionary[guId];
                  const guName = spec?.name || guId;
                  const isObfuscated = guId === "???";

                  return (
                    <span
                      key={index}
                      className={`text-xs px-3 py-1.5 rounded-xl border font-serif font-semibold flex items-center gap-1.5 transition-all hover:scale-102 ${
                        isObfuscated
                          ? "bg-zinc-900/60 text-zinc-600 border-zinc-900/40 select-none cursor-not-allowed"
                          : isImmortal
                          ? "bg-purple-950/40 hover:bg-purple-900/40 text-purple-300 border-purple-900/40 hover:border-purple-700/60 cursor-default"
                          : "bg-emerald-950/40 hover:bg-emerald-900/40 text-emerald-400 border-emerald-900/40 hover:border-emerald-700/60 cursor-default"
                      }`}
                    >
                      {!isObfuscated && (
                        <Sparkles className={`w-3 h-3 shrink-0 ${isImmortal ? "text-purple-400" : "text-emerald-500"}`} />
                      )}
                      {guName}
                    </span>
                  );
                })
              ) : (
                <p className="text-xs text-zinc-600 italic">Không trang bị cổ trùng nào</p>
              )}
            </div>
          </div>

          {/* Direct P2P Trade Action */}
          {player && inspectedProfile && player.username !== username && (
            <div className="sm:col-span-2 border-t border-zinc-900/40 pt-4 flex justify-end">
              <button
                onClick={() => {
                  if (inspectedUsernameQuery) {
                    initiateTrade(inspectedUsernameQuery);
                  }
                  clearInspectedProfile();
                }}
                className="w-full py-3 rounded-2xl font-serif font-bold text-sm bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 text-zinc-100 shadow-lg shadow-emerald-950/20 active:scale-[0.98] transition-all flex items-center justify-center gap-2 cursor-pointer border border-emerald-500/20"
              >
                <Sparkles className="w-4 h-4" /> Bắt Đầu Giao Dịch
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
