"use client";

import React, { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Trophy, Eye, EyeOff, Users, RefreshCw, Coins, Sparkles, Bug, Star } from "lucide-react";

const CATEGORIES = [
  { id: "rank", label: "Tu Luyện", icon: Trophy, color: "emerald" },
  { id: "primeval_stones", label: "Nguyên Thạch", icon: Coins, color: "yellow" },
  { id: "gu_strongest", label: "Cổ Trùng", icon: Bug, color: "violet" },
  { id: "immortal_stones", label: "Tiên Nguyên", icon: Sparkles, color: "purple" },
];

const getRankName = (rank: number, sub: number): string => {
  const ranks = ["Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"];
  const rankNames = [
    "Nhất Chuyển", "Nhị Chuyển", "Tam Chuyển", "Tứ Chuyển",
    "Ngũ Chuyển", "Lục Tiên", "Thất Tiên", "Bát Tiên", "Cửu Tôn",
  ];
  const rName = rankNames[rank - 1] || `${rank} Chuyển`;
  const sName = ranks[sub - 1] || `${sub} Giai`;
  return rank >= 6 ? rName : `${rName} • ${sName}`;
};

const getInitials = (username: string): string => {
  if (!username) return "?";
  return username.slice(0, 2).toUpperCase();
};

export const LeaderboardView: React.FC = () => {
  const {
    player,
    leaderboardEntries,
    leaderboardCategory,
    myLeaderboardRank,
    leaderboardTotalPlayers,
    leaderboardShowOnLeaderboard,
    leaderboardLoading,
    fetchLeaderboard,
    toggleLeaderboardVisibility,
    fetchLeaderboardSettings,
    fetchPlayerProfile,
  } = useGameStore();

  useEffect(() => {
    fetchLeaderboard(leaderboardCategory);
    fetchLeaderboardSettings();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleCategoryChange = (catId: string) => {
    if (catId === leaderboardCategory) return;
    fetchLeaderboard(catId);
  };

  const handleRefresh = () => {
    fetchLeaderboard(leaderboardCategory);
  };

  const handleToggleVisibility = () => {
    toggleLeaderboardVisibility();
  };

  if (!player) return null;

  const currentCat = CATEGORIES.find(c => c.id === leaderboardCategory) || CATEGORIES[0];

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 text-left relative overflow-hidden group">
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(217,119,6,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(217,119,6,0.01)_1px,transparent_1px)] bg-[size:16px_28px] pointer-events-none" />

      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-4 relative z-10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-amber-950/40 border border-amber-500/20 flex items-center justify-center">
            <Trophy className="w-5 h-5 text-amber-400" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-zinc-100 font-serif">Bảng Xếp Hạng</h3>
            <p className="text-xs text-zinc-500">Thiên hạ anh tài, tranh phong đấu thế.</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleToggleVisibility}
            className={`p-2 border rounded-xl transition-all cursor-pointer font-mono shrink-0 flex items-center gap-1.5 text-xs font-semibold ${
              leaderboardShowOnLeaderboard
                ? "bg-emerald-950/40 border-emerald-500/50 text-emerald-500 hover:bg-emerald-950/60"
                : "bg-rose-950/40 border-rose-500/50 text-rose-500 hover:bg-rose-950/60"
            }`}
            title={leaderboardShowOnLeaderboard ? "Ẩn khỏi bảng xếp hạng" : "Hiển thị trên bảng xếp hạng"}
          >
            {leaderboardShowOnLeaderboard ? <Eye className="w-3.5 h-3.5" /> : <EyeOff className="w-3.5 h-3.5" />}
            {leaderboardShowOnLeaderboard ? "Đang tham gia" : "Đã ẩn"}
          </button>

          <button
            onClick={handleRefresh}
            disabled={leaderboardLoading}
            className="p-2 border border-zinc-800 hover:border-zinc-700 bg-zinc-950 hover:bg-zinc-900 rounded-xl text-zinc-455 hover:text-zinc-200 transition-all cursor-pointer font-mono shrink-0 flex items-center gap-1.5 text-xs font-semibold"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${leaderboardLoading ? "animate-spin text-amber-400" : ""}`} />
            Tải lại
          </button>
        </div>
      </div>

      {/* Player rank summary banner */}
      <div className="bg-gradient-to-r from-amber-950/20 via-zinc-950/60 to-zinc-950/60 border border-amber-900/20 rounded-2xl p-5 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 relative z-10 shadow-lg">
        <div className="space-y-1">
          <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono">
            Xếp hạng của bản tôn — {currentCat.label}
          </span>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-black text-amber-400 font-serif">
              {myLeaderboardRank ? `#${myLeaderboardRank.rank}` : "Chưa xếp hạng"}
            </span>
            <span className="text-xs text-zinc-400 font-mono">
              / {leaderboardTotalPlayers} người chơi
            </span>
          </div>
          {myLeaderboardRank && (
            <div className="text-xs text-zinc-500 font-mono mt-1">
              Giá trị: <span className="text-amber-400 font-bold">{myLeaderboardRank.value}</span>
            </div>
          )}
        </div>

        <div className="flex gap-4">
          <div className="text-left bg-zinc-950/80 px-4 py-2 border border-zinc-900 rounded-xl">
            <div className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">Tu vi</div>
            <span className="text-sm font-bold text-emerald-400 font-mono">
              {getRankName(player.rank, player.sub_rank)}
            </span>
          </div>
          <div className="text-left bg-zinc-950/80 px-4 py-2 border border-zinc-900 rounded-xl">
            <div className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">Trạng thái BXH</div>
            <span className={`text-sm font-bold font-mono ${leaderboardShowOnLeaderboard ? "text-emerald-400" : "text-rose-400"}`}>
              {leaderboardShowOnLeaderboard ? "Hiển thị" : "Ẩn"}
            </span>
          </div>
        </div>
      </div>

      {/* Category Tabs */}
      <div className="flex gap-2 relative z-10 overflow-x-auto pb-1">
        {CATEGORIES.map((cat) => {
          const isActive = cat.id === leaderboardCategory;
          const Icon = cat.icon;
          return (
            <button
              key={cat.id}
              onClick={() => handleCategoryChange(cat.id)}
              className={`flex items-center gap-1.5 px-3 py-2 rounded-xl text-xs font-semibold font-mono border transition-all cursor-pointer shrink-0 ${
                isActive
                  ? `bg-${cat.color}-950/40 border-${cat.color}-500/50 text-${cat.color}-400`
                  : "bg-zinc-950/40 border-zinc-800 text-zinc-500 hover:border-zinc-700 hover:text-zinc-300"
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              {cat.label}
            </button>
          );
        })}
      </div>

      {/* Leaderboard list */}
      <div className="space-y-2 relative z-10 max-h-[480px] overflow-y-auto pr-1">
        {leaderboardLoading ? (
          <div className="flex items-center justify-center py-16">
            <RefreshCw className="w-6 h-6 text-amber-400 animate-spin" />
          </div>
        ) : leaderboardEntries.length === 0 ? (
          <div className="text-center py-16 text-zinc-500 italic text-xs font-mono">
            Chưa có dữ liệu bảng xếp hạng cho danh mục này.
          </div>
        ) : (
          leaderboardEntries.map((entry) => {
            const isMe = entry.player_id === player.id;
            return (
              <div
                key={entry.player_id}
                className={`flex items-center justify-between p-4 border rounded-2xl transition-all duration-300 ${
                  isMe
                    ? "bg-amber-950/10 border-amber-500/30 shadow-[0_0_12px_rgba(245,158,11,0.05)]"
                    : "bg-zinc-950/40 border-zinc-900 hover:border-zinc-800/80 hover:bg-zinc-950/60"
                }`}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div className={`w-8 h-8 rounded-lg flex items-center justify-center font-serif font-bold text-sm shrink-0 ${
                    entry.position === 1 ? "bg-amber-500/10 border border-amber-500 text-amber-400" :
                    entry.position === 2 ? "bg-zinc-300/10 border border-zinc-300 text-zinc-300" :
                    entry.position === 3 ? "bg-amber-800/10 border border-amber-800 text-amber-700" :
                    "bg-zinc-900 border border-zinc-800 text-zinc-500"
                  }`}>
                    {entry.position}
                  </div>

                  <div className="w-8 h-8 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[9px] font-bold text-zinc-500">
                    {entry.avatar_url ? (
                      <img src={entry.avatar_url} alt="" className="w-full h-full object-cover" />
                    ) : (
                      getInitials(entry.username)
                    )}
                  </div>

                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span
                        onClick={() => !isMe && fetchPlayerProfile(entry.username)}
                        className={`font-serif text-sm font-bold truncate ${isMe ? "text-amber-400" : "text-zinc-200 cursor-pointer hover:text-amber-400 hover:underline decoration-dotted transition-colors"}`}
                        title={!isMe ? "Bấm để trinh thám cổ sư" : undefined}
                      >
                        {entry.username}
                      </span>
                      {isMe && (
                        <span className="text-[8px] bg-amber-950/40 text-amber-500 border border-amber-900/30 px-1.5 py-0.2 rounded font-mono uppercase font-bold tracking-wider shrink-0">
                          Đại Diện
                        </span>
                      )}
                      {entry.position <= 3 && (
                        <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500/20 shrink-0" />
                      )}
                    </div>
                    <span className="text-[10px] text-zinc-500 font-mono">
                      {getRankName(entry.cultivation_rank, entry.sub_rank)}
                    </span>
                  </div>
                </div>

                <div className="shrink-0 ml-3 text-right">
                  <div className="text-sm font-bold text-amber-400 font-mono">
                    {leaderboardCategory === "primeval_stones"
                      ? Number(entry.value).toLocaleString()
                      : leaderboardCategory === "immortal_stones"
                      ? Number(entry.value).toLocaleString()
                      : leaderboardCategory === "gu_strongest"
                      ? `Cấp ${entry.value}`
                      : entry.value_label}
                  </div>
                  <div className="text-[9px] text-zinc-600 font-mono uppercase tracking-wider">
                    {currentCat.label}
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {leaderboardEntries.length > 0 && (
        <div className="text-[10px] text-zinc-600 font-mono text-center relative z-10 pt-2 border-t border-zinc-900/50">
          <Users className="w-3 h-3 inline mr-1" />
          Tổng số {leaderboardTotalPlayers} người chơi đang tham gia bảng xếp hạng
          {!leaderboardShowOnLeaderboard && (
            <span className="text-rose-500"> • Bạn đang ẩn khỏi bảng xếp hạng</span>
          )}
        </div>
      )}
    </div>
  );
};
