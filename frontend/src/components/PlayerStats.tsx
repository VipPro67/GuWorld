"use client";

import React from "react";
import { Player } from "@/store/useGameStore";
import { Award, Shield, Compass, Sparkles } from "lucide-react";

interface PlayerStatsProps {
  player: Player;
  onLogout: () => void;
}

const getRankName = (rank: number): string => {
  const ranks = [
    "Không",
    "Nhất Chuyển (Rank 1)",
    "Nhị Chuyển (Rank 2)",
    "Tam Chuyển (Rank 3)",
    "Tứ Chuyển (Rank 4)",
    "Ngũ Chuyển (Rank 5)",
    "Lục Chuyển (Rank 6)",
    "Thất Chuyển (Rank 7)",
    "Bát Chuyển (Rank 8)",
    "Cửu Chuyển (Rank 9)",
  ];
  return ranks[rank] || `Rank ${rank}`;
};

const getSubRankName = (subRank: number): string => {
  const subRanks = ["Sơ Giai", "Trung Giai", "Cao Giai", "Đỉnh Phong"];
  return subRanks[subRank - 1] || "Khởi Đầu";
};

export const PlayerStats: React.FC<PlayerStatsProps> = ({ player, onLogout }) => {
  return (
    <div className="glass-panel rounded-3xl p-6 w-full max-w-sm flex flex-col justify-between h-full relative overflow-hidden group">
      {/* Decorative background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(16,185,129,0.02)_1px,transparent_1px),linear-gradient(to_bottom,rgba(16,185,129,0.02)_1px,transparent_1px)] bg-[size:14px_24px] pointer-events-none" />

      <div>
        <div className="flex justify-between items-center mb-6">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-ping" />
            <span className="text-[10px] text-emerald-400 font-mono uppercase tracking-widest">
              Đang hoạt động
            </span>
          </div>
          <button
            onClick={onLogout}
            className="text-xs text-zinc-500 hover:text-rose-400 transition-colors cursor-pointer border border-zinc-800 hover:border-rose-900/50 px-2.5 py-1 rounded-lg bg-zinc-950/40"
          >
            Đổi Nhân Vật
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <span className="text-[10px] text-zinc-500 uppercase tracking-wider block mb-0.5">
              Cổ Sư (Cultivator)
            </span>
            <h2 className="text-2xl font-bold text-zinc-100 flex items-center gap-2 font-serif">
              {player.username}
              <Sparkles className="w-4 h-4 text-amber-500 animate-pulse" />
            </h2>
          </div>

          <div className="border-t border-zinc-800/80 my-3" />

          {/* Stats Grid */}
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900/60">
              <div className="flex items-center gap-1.5 text-zinc-500 mb-1">
                <Award className="w-3.5 h-3.5 text-amber-500" />
                <span className="text-[10px] uppercase tracking-wider">Tu Vi</span>
              </div>
              <span className="text-sm font-semibold text-zinc-200 block truncate">
                {getRankName(player.rank)}
              </span>
            </div>

            <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900/60">
              <div className="flex items-center gap-1.5 text-zinc-500 mb-1">
                <Shield className="w-3.5 h-3.5 text-emerald-500" />
                <span className="text-[10px] uppercase tracking-wider">Cảnh Giới</span>
              </div>
              <span className="text-sm font-semibold text-zinc-200 block">
                {getSubRankName(player.sub_rank)}
              </span>
            </div>

            <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900/60">
              <div className="flex items-center gap-1.5 text-zinc-500 mb-1">
                <Compass className="w-3.5 h-3.5 text-blue-400" />
                <span className="text-[10px] uppercase tracking-wider">Tư Chất</span>
              </div>
              <span className="text-sm font-semibold text-zinc-200 block">
                Cấp {player.aptitude_type} ({player.aptitude_percent}%)
              </span>
            </div>

            <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900/60">
              <div className="flex items-center gap-1.5 text-zinc-500 mb-1">
                <div className="w-3 h-3 rounded-full bg-amber-500 flex items-center justify-center text-[8px] text-zinc-950 font-bold">
                  $
                </div>
                <span className="text-[10px] uppercase tracking-wider">Nguyên Thạch</span>
              </div>
              <span className="text-sm font-bold text-amber-400 block">
                {player.primeval_stones} viên
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="mt-8 pt-4 border-t border-zinc-900 text-[10px] text-zinc-600 flex justify-between">
        <span>Khai thiên lập địa:</span>
        <span>{new Date(player.created_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
};
