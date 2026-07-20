"use client";

import React, { useEffect, useState } from "react";
import { Gift, Sparkles, Coins, Heart, Zap, Gem, Crown, X, ChevronRight, Star, Repeat, Clock } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";

const POOL_ICONS: Record<string, React.ReactNode> = {
  da_thuong: <Coins className="w-5 h-5 text-yellow-400" />,
  tho_nguyen_co_tu: <Heart className="w-5 h-5 text-emerald-400" />,
  van_menh_co_tu: <Crown className="w-5 h-5 text-purple-400" />,
};

const POOL_COLORS: Record<string, string> = {
  da_thuong: "from-yellow-600/30 to-amber-900/30 border-yellow-700/40",
  tho_nguyen_co_tu: "from-emerald-600/30 to-green-900/30 border-emerald-700/40",
  van_menh_co_tu: "from-purple-600/30 to-violet-900/30 border-purple-700/40",
};

const POOL_BTN: Record<string, string> = {
  da_thuong: "bg-yellow-600 hover:bg-yellow-500",
  tho_nguyen_co_tu: "bg-emerald-600 hover:bg-emerald-500",
  van_menh_co_tu: "bg-purple-600 hover:bg-purple-500",
};

export const GachaPanel: React.FC = () => {
  const { gachaPools, fetchGachaPools, openGacha, openGacha10, actionLoading, addToast, player } = useGameStore();
  const [results, setResults] = useState<{ poolId: string; rewards: any[] } | null>(null);
  const [showAnim, setShowAnim] = useState(false);
  const [animIndex, setAnimIndex] = useState(0);
  const [animRewards, setAnimRewards] = useState<any[]>([]);

  useEffect(() => {
    fetchGachaPools();
  }, [fetchGachaPools]);

  useEffect(() => {
    if (showAnim && animIndex < animRewards.length) {
      const timer = setTimeout(() => setAnimIndex(animIndex + 1), 600);
      return () => clearTimeout(timer);
    }
    if (showAnim && animIndex >= animRewards.length) {
      const timer = setTimeout(() => { setShowAnim(false); setResults(null); }, 2000);
      return () => clearTimeout(timer);
    }
  }, [showAnim, animIndex, animRewards]);

  const handleOpen = async (poolId: string, count: 1 | 10) => {
    if (actionLoading) return;
    const result = count === 1 ? await openGacha(poolId) : await openGacha10(poolId);
    if (result?.success) {
      const rewards = "reward" in result && result.reward ? [result.reward] : ("rewards" in result ? (result.rewards || []) : []);
      setAnimRewards(rewards);
      setAnimIndex(0);
      setShowAnim(true);
    }
  };

  const renderRewardIcon = (r: any) => {
    switch (r?.type) {
      case "primeval_stones": return <Coins className="w-8 h-8 text-yellow-400" />;
      case "lifespan": return <Heart className="w-8 h-8 text-emerald-400" />;
      case "human_qi": return <Zap className="w-8 h-8 text-cyan-400" />;
      case "wall_progress": return <Sparkles className="w-8 h-8 text-amber-400" />;
      case "item": return <Gift className="w-8 h-8 text-blue-400" />;
      case "gu": return <Gem className={`w-8 h-8 ${r.is_rare ? "text-purple-400 animate-pulse" : "text-amber-400"}`} />;
      default: return <Star className="w-8 h-8 text-zinc-400" />;
    }
  };

  const rarityClass = (r: any) => {
    if (r?.is_rare) return "bg-gradient-to-br from-purple-900/80 to-rose-900/80 border-purple-500/60 shadow-[0_0_20px_rgba(168,85,247,0.3)]";
    return "bg-zinc-900/80 border-zinc-700/60";
  };

  if (showAnim) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80" onClick={() => { setShowAnim(false); setResults(null); }}>
        <div className="max-w-lg w-full mx-4" onClick={e => e.stopPropagation()}>
          <div className="text-center mb-6">
            <h3 className="text-lg font-bold font-serif text-amber-400 animate-pulse">MỞ CỔ TỪ</h3>
            <div className="text-[10px] text-zinc-500 font-mono mt-1">{animIndex}/{animRewards.length}</div>
          </div>
          <div className="grid grid-cols-5 gap-3 justify-items-center">
            {animRewards.map((r, i) => (
              <div
                key={i}
                className={`w-full aspect-square rounded-2xl border-2 flex flex-col items-center justify-center gap-1 p-1 transition-all duration-500 ${i < animIndex ? rarityClass(r) : "bg-zinc-950/80 border-zinc-800/40 opacity-30"}`}
              >
                {i < animIndex ? (
                  <>
                    {renderRewardIcon(r)}
                    <span className="text-[7px] font-mono text-zinc-300 text-center leading-tight truncate w-full px-0.5">{r.label}</span>
                    {r.is_rare && <Star className="w-3 h-3 text-yellow-400 absolute top-1 right-1" />}
                  </>
                ) : (
                  <div className="text-2xl text-zinc-700">?</div>
                )}
              </div>
            ))}
          </div>
          {animIndex >= animRewards.length && (
            <div className="text-center mt-4 text-emerald-400 text-xs font-mono font-bold animate-fade-in">
              ✦ Nhận thưởng thành công! ✦
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-1">
        <Gift className="w-5 h-5 text-amber-400" />
        <span className="font-serif font-bold text-zinc-100 text-sm">Cổ Từ Bảo Khố</span>
        <span className="text-[10px] text-zinc-500 font-mono ml-auto">Thử vận may — Cơ duyên Tiên Đạo!</span>
      </div>

      {gachaPools.length === 0 ? (
        <div className="text-center py-10 text-zinc-600 italic text-xs">Đang tải...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {gachaPools.map((pool: any) => (
            <div key={pool.id} className={`bg-gradient-to-br ${POOL_COLORS[pool.id] || "from-zinc-800/50 to-zinc-900/50"} rounded-2xl border p-4 flex flex-col gap-3`}>
              <div className="flex items-center gap-2">
                {POOL_ICONS[pool.id] || <Gift className="w-5 h-5 text-zinc-400" />}
                <div>
                  <div className="text-xs font-bold text-zinc-100">{pool.name}</div>
                  <div className="text-[9px] text-zinc-500 font-mono">
                    {pool.cost_type === "primeval_stones" ? `${pool.cost_amount.toLocaleString()} Nguyên Thạch` : `${pool.cost_amount} năm Thọ Nguyên`}
                  </div>
                </div>
              </div>
              <div className="text-[10px] text-zinc-400 leading-relaxed min-h-[40px]">{pool.description}</div>
              <div className="flex gap-2 mt-auto">
                <button
                  disabled={actionLoading}
                  onClick={() => handleOpen(pool.id, 1)}
                  className={`flex-1 py-2 ${POOL_BTN[pool.id] || "bg-amber-600 hover:bg-amber-500"} disabled:opacity-50 text-white rounded-xl text-[10px] font-bold font-mono cursor-pointer transition-all`}
                >
                  Mở 1
                </button>
                <button
                  disabled={actionLoading}
                  onClick={() => handleOpen(pool.id, 10)}
                  className={`flex-1 py-2 ${POOL_BTN[pool.id] || "bg-amber-600 hover:bg-amber-500"} disabled:opacity-50 text-white rounded-xl text-[10px] font-bold font-mono cursor-pointer transition-all flex items-center justify-center gap-1`}
                >
                  <Repeat className="w-3 h-3" />Mở 10
                  <span className="text-[8px] opacity-70">-10%</span>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
