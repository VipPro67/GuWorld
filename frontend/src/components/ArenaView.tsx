"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Swords, Trophy, ShieldAlert, Award, RefreshCw, Star, Store } from "lucide-react";
import { ArenaShopView } from "./ArenaShopView";

export const ArenaView: React.FC = () => {
  const { player, arenaLeaderboard, arenaDailyChallenges, fetchArenaLeaderboard, challengeArena, actionLoading, fetchPlayerProfile } = useGameStore();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'ranking' | 'shop'>('ranking');

  const handleRefresh = async () => {
    setLoading(true);
    try {
      await fetchArenaLeaderboard();
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    handleRefresh();
  }, []);

  const handleChallenge = async (targetId: string, opponentUsername: string) => {
    if (actionLoading) return;
    const ok = await challengeArena(targetId);
    if (ok) {
      // Game store challengeArena automatically forces view/state redirect.
    }
  };

  if (!player) return null;

  const playerRank = player.arena_rank || 999;
  
  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 text-left relative overflow-hidden group">
      {/* Decorative background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(217,119,6,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(217,119,6,0.01)_1px,transparent_1px)] bg-[size:16px_28px] pointer-events-none" />

      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-4 relative z-10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-amber-950/40 border border-amber-500/20 flex items-center justify-center">
            <Trophy className="w-5 h-5 text-amber-400 animate-pulse" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-zinc-100 font-serif">Đấu Kỹ Võ Đài</h3>
            <p className="text-xs text-zinc-500">Nơi tranh đoạt hư danh, so tài cao thấp, hoán đổi thứ tự Thiên địa bảng.</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab(activeTab === 'ranking' ? 'shop' : 'ranking')}
            className={`p-2 border rounded-xl transition-all cursor-pointer font-mono shrink-0 flex items-center gap-1.5 text-xs font-semibold ${
              activeTab === 'shop' 
                ? "bg-amber-950/40 border-amber-500/50 text-amber-500" 
                : "border-zinc-800 hover:border-zinc-700 bg-zinc-950 hover:bg-zinc-900 text-zinc-455 hover:text-zinc-200"
            }`}
          >
            <Store className="w-3.5 h-3.5" />
            {activeTab === 'ranking' ? 'Bảo Các' : 'Xếp Hạng'}
          </button>

          <button
            onClick={handleRefresh}
            disabled={loading || actionLoading}
            className="p-2 border border-zinc-800 hover:border-zinc-700 bg-zinc-950 hover:bg-zinc-900 rounded-xl text-zinc-455 hover:text-zinc-200 transition-all cursor-pointer font-mono shrink-0 flex items-center gap-1.5 text-xs font-semibold"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-amber-400" : ""}`} />
            Tải lại
          </button>
        </div>
      </div>

      {/* Player ranking summary banner */}
      <div className="bg-gradient-to-r from-amber-950/20 via-zinc-950/60 to-zinc-950/60 border border-amber-900/20 rounded-2xl p-5 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 relative z-10 shadow-lg">
        <div className="space-y-1">
          <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono">Xếp hạng của bản tôn</span>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-black text-amber-400 font-serif">
              {player.arena_rank ? `#${player.arena_rank}` : "Vô Danh"}
            </span>
            <span className="text-xs text-zinc-400 font-mono">
              (Lượt khiêu chiến hôm nay: {Math.max(0, 3 - (arenaDailyChallenges || 0))}/3)
            </span>
          </div>
        </div>

        <div className="flex gap-4">
          <div className="text-left bg-zinc-950/80 px-4 py-2 border border-zinc-900 rounded-xl">
            <div className="text-[9px] text-amber-500/80 uppercase tracking-widest font-mono">Xu Đấu Trường</div>
            <span className="text-sm font-bold text-amber-400 font-mono">{player.materials_inventory?.arena_coin || 0} xu</span>
          </div>
          <div className="text-left bg-zinc-950/80 px-4 py-2 border border-zinc-900 rounded-xl">
            <div className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">Tu vi đạo hạnh</div>
            <span className="text-sm font-bold text-emerald-600 dark:text-emerald-400">Bậc {player.rank} - {player.sub_rank}</span>
          </div>
        </div>
      </div>

      <div className="text-[10px] text-zinc-550 font-mono max-w-xl">
        * Quy tắc Võ Đài: Cổ sư có thể khiêu chiến <span className="text-amber-500 font-bold">tự do</span>, tối đa 3 lần/ngày.
        Nếu chiến thắng kẻ có hạng cao hơn, hai bên sẽ <span className="text-emerald-600 dark:text-emerald-400 font-bold">hoán đổi vị trí xếp hạng</span> cho nhau.
        Kết toán phát Xu Đấu Trường mỗi 22:10 hằng ngày.
      </div>

      {activeTab === 'shop' ? (
        <ArenaShopView />
      ) : (
        <>
          {/* Leaderboard list */}
          <div className="space-y-2 relative z-10 max-h-[420px] overflow-y-auto pr-1">
        {arenaLeaderboard.length === 0 ? (
          <div className="text-center py-12 text-zinc-650 italic text-xs">
            Chưa có cổ sư nào ghi tên trên bảng võ đài. Hãy là người đầu tiên!
          </div>
        ) : (
          arenaLeaderboard.map((row) => {
            const isMe = row.player_id === player.id;
            const canChallenge = !isMe && arenaDailyChallenges < 99;
            
            return (
              <div
                key={row.player_id}
                className={`flex flex-col sm:flex-row justify-between items-start sm:items-center p-4 border rounded-2xl transition-all duration-300 ${
                  isMe
                    ? "bg-amber-955/10 border-amber-500/30 shadow-[0_0_12px_rgba(245,158,11,0.05)]"
                    : "bg-zinc-950/40 border-zinc-900 hover:border-zinc-800/80 hover:bg-zinc-950/60"
                }`}
              >
                <div className="flex items-center gap-3">
                  {/* Rank Badge */}
                  <div className={`w-8 h-8 rounded-lg flex items-center justify-center font-serif font-bold text-sm ${
                    row.rank === 1 ? "bg-amber-500/10 border border-amber-500 text-amber-400" :
                    row.rank === 2 ? "bg-zinc-350/10 border border-zinc-300 text-zinc-300" :
                    row.rank === 3 ? "bg-amber-800/10 border border-amber-800 text-amber-700" :
                    "bg-zinc-900 border border-zinc-850 text-zinc-500"
                  }`}>
                    {row.rank}
                  </div>
                  
                  <div>
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[9px] font-bold text-zinc-500">
                        {row.avatar_url ? (
                          <img src={row.avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          row.username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                      <span
                        onClick={() => !isMe && fetchPlayerProfile(row.username)}
                        className={`font-serif text-sm font-bold ${isMe ? "text-amber-400" : "text-zinc-200 cursor-pointer hover:text-amber-400 hover:underline decoration-dotted transition-colors"}`}
                        title={!isMe ? "Bấm để trinh thám cổ sư" : undefined}
                      >
                        {row.username}
                      </span>
                      {isMe && (
                        <span className="text-[8px] bg-amber-950/40 text-amber-500 border border-amber-900/30 px-1.5 py-0.2 rounded font-mono uppercase font-bold tracking-wider">
                          Đại Diện
                        </span>
                      )}
                      {row.rank <= 3 && (
                        <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500/20" />
                      )}
                    </div>
                    <span className="text-[10px] text-zinc-500 font-mono mt-0.5 block">
                      Tích điểm: {row.points} | Cập nhật: {new Date(row.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>

                <div className="mt-3 sm:mt-0 w-full sm:w-auto shrink-0 flex items-center gap-3">
                  {canChallenge ? (
                    <button
                      onClick={() => handleChallenge(row.player_id, row.username)}
                      disabled={actionLoading}
                      className="w-full sm:w-auto py-1.5 px-4 bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-600 hover:to-amber-700 text-zinc-950 font-bold text-xs rounded-xl transition-all active:scale-[0.98] shadow-md hover:shadow-amber-500/10 cursor-pointer flex items-center justify-center gap-1.5 font-mono"
                    >
                      <Swords className="w-3.5 h-3.5" />
                      Khiêu Chiến
                    </button>
                  ) : !isMe ? (
                    <span className="text-[10px] text-zinc-600 font-mono italic select-none">
                      (Đã hết lượt)
                    </span>
                  ) : (
                    <span className="text-[10px] text-amber-500 font-serif font-bold flex items-center gap-1">
                      👑 Bản Tôn Vị Trí
                    </span>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
      </>
      )}

      {player.active_combat && (
        <div className="bg-rose-950/20 border border-rose-900/40 rounded-2xl p-4 flex items-center gap-3 relative z-10 animate-pulse">
          <ShieldAlert className="w-5 h-5 text-rose-500 shrink-0" />
          <div className="text-left text-xs">
            <span className="font-bold text-rose-600 dark:text-rose-400 font-serif block">Đang ở trong trạng thái chiến đấu!</span>
            <span className="text-zinc-550 block">Hãy hoàn thành trận chiến dã thú hiện tại trước khi tỷ đấu võ đài.</span>
          </div>
        </div>
      )}
    </div>
  );
};
