"use client";

import React, { useEffect, useState } from "react";
import { Gift, Sparkles, Coins, Heart, Zap, Gem, Crown, X, ChevronRight, Star, Repeat, Clock, Minus, Plus, Layers } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";

const POOL_ICONS: Record<string, React.ReactNode> = {
  da_thuong: <Coins className="w-5 h-5 text-yellow-400" />,
  da_vip: <Crown className="w-5 h-5 text-purple-400" />,
  da_thuong_pham: <Sparkles className="w-5 h-5 text-emerald-400" />,
  da_cuc_pham: <Gem className="w-5 h-5 text-rose-400" />,
};

const POOL_COLORS: Record<string, string> = {
  da_thuong: "from-yellow-600/30 to-amber-900/30 border-yellow-700/40",
  da_vip: "from-purple-600/30 to-violet-900/30 border-purple-700/40",
  da_thuong_pham: "from-emerald-600/30 to-teal-900/30 border-emerald-700/40",
  da_cuc_pham: "from-rose-600/30 to-red-900/30 border-rose-700/40",
};

const POOL_BTN: Record<string, string> = {
  da_thuong: "bg-yellow-600 hover:bg-yellow-500",
  da_vip: "bg-purple-600 hover:bg-purple-500",
  da_thuong_pham: "bg-emerald-600 hover:bg-emerald-500",
  da_cuc_pham: "bg-rose-600 hover:bg-rose-500",
};

const QUICK_QTYS = [1, 10, 50, 100];
const COOLDOWN_MS = 12000;

interface GachaPoolShape {
  id: string;
  name: string;
  description: string;
  cost_type: string;
  cost_amount: number;
}

export const GachaPanel: React.FC = () => {
  const { gachaPools, fetchGachaPools, openGacha, actionLoading, player } = useGameStore();
  const [results, setResults] = useState<{ poolId: string; rewards: ApiResponseData[] } | null>(null);
  const [showAnim, setShowAnim] = useState(false);
  const [animIndex, setAnimIndex] = useState(0);
  const [animRewards, setAnimRewards] = useState<ApiResponseData[]>([]);
  const [quantity, setQuantity] = useState<number>(1);
  const [cooldownUntil, setCooldownUntil] = useState(0);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    fetchGachaPools();
  }, [fetchGachaPools]);

  // Tick the cooldown countdown while it is active.
  useEffect(() => {
    if (cooldownUntil <= Date.now()) return;
    setNow(Date.now());
    const timer = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(timer);
  }, [cooldownUntil]);

  const cooldownLeft = Math.max(0, Math.ceil((cooldownUntil - now) / 1000));

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

  const clampQty = (n: number): number => Math.max(1, Math.min(100, Math.floor(n) || 1));

  const handleOpen = async (poolId: string, count: number) => {
    if (actionLoading || Date.now() < cooldownUntil) return;
    const result = await openGacha(poolId, count);
    // Start the cooldown on any attempt (success or server 429) so the client
    // stops firing opens while the server rate-limit window is still hot.
    setCooldownUntil(Date.now() + COOLDOWN_MS);
    if (result?.success) {
      const rewards = "reward" in result && result.reward ? [result.reward] : ("rewards" in result ? (result.rewards || []) : []);
      setAnimRewards(rewards);
      setAnimIndex(0);
      setShowAnim(true);
    }
  };

  const totalCost = (pool: GachaPoolShape): number => {
    const base = pool.cost_amount * quantity;
    return quantity >= 10 ? Math.floor(base * 0.9) : Math.floor(base);
  };

  const costLabel = (pool: GachaPoolShape): string => {
    const v = totalCost(pool);
    return pool.cost_type === "primeval_stones" ? `${v.toLocaleString()} Nguyên Thạch` : `${v} Tiên Nguyên Thạch`;
  };

  const canAfford = (pool: GachaPoolShape): boolean => {
    const v = totalCost(pool);
    return pool.cost_type === "primeval_stones"
      ? (player?.primeval_stones || 0) >= v
      : (player?.immortal_stones || 0) >= v;
  };

  const renderRewardIcon = (r: ApiResponseData) => {
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

  const rarityClass = (r: ApiResponseData) => {
    if (r?.is_rare) return "bg-gradient-to-br from-purple-900/80 to-rose-900/80 border-purple-500/60 shadow-[0_0_20px_rgba(168,85,247,0.3)]";
    return "bg-zinc-900/80 border-zinc-700/60";
  };

  if (showAnim) {
    const shown = animRewards.slice(0, 20);
    const hiddenCount = animRewards.length - shown.length;
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80" onClick={() => { setShowAnim(false); setResults(null); }}>
        <div className="max-w-lg w-full mx-4" onClick={e => e.stopPropagation()}>
          <div className="text-center mb-6">
            <h3 className="text-lg font-bold font-serif text-amber-400 animate-pulse">Mở Đá</h3>
            <div className="text-[10px] text-zinc-500 font-mono mt-1">{Math.min(animIndex, shown.length)}/{animRewards.length} lần</div>
          </div>
          <div className="grid grid-cols-5 gap-3 justify-items-center">
            {shown.map((r, i) => (
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
          {hiddenCount > 0 && (
            <div className="text-center mt-3 text-[10px] text-zinc-500 font-mono">
              ... và {hiddenCount} lần mở khác
            </div>
          )}
          {animIndex >= shown.length && (
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
        <>
          {/* Quantity selector */}
          <div className="bg-zinc-900/30 border border-zinc-800/60 rounded-2xl p-4 flex flex-col sm:flex-row items-center gap-4">
            <div className="flex items-center gap-2">
              <Layers className="w-4 h-4 text-amber-400" />
              <span className="text-xs font-serif font-bold text-zinc-300">Số lần mở</span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setQuantity((q) => clampQty(q - 1))}
                className="w-8 h-8 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 flex items-center justify-center cursor-pointer transition-colors"
              >
                <Minus className="w-4 h-4" />
              </button>
              <input
                type="number"
                min={1}
                max={100}
                value={quantity}
                onChange={(e) => setQuantity(clampQty(parseInt(e.target.value) || 1))}
                className="w-20 h-8 bg-zinc-950 border border-zinc-800 rounded-lg text-center text-sm font-mono font-bold text-amber-400 outline-none focus:border-amber-600"
              />
              <button
                onClick={() => setQuantity((q) => clampQty(q + 1))}
                className="w-8 h-8 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 flex items-center justify-center cursor-pointer transition-colors"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="flex items-center gap-1.5">
              {QUICK_QTYS.map((q) => (
                <button
                  key={q}
                  onClick={() => setQuantity(q)}
                  className={`px-2.5 py-1 rounded-lg text-[10px] font-mono font-bold transition-all cursor-pointer ${quantity === q ? "bg-amber-600 text-white" : "bg-zinc-800 text-zinc-400 hover:text-zinc-200"}`}
                >
                  {q}
                </button>
              ))}
            </div>
            {quantity >= 10 && (
              <span className="text-[10px] font-mono font-bold text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 px-2 py-1 rounded-lg">
                -10% chi phí
              </span>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {gachaPools.map((pool: GachaPoolShape) => (
              <div key={pool.id} className={`bg-gradient-to-br ${POOL_COLORS[pool.id] || "from-zinc-800/50 to-zinc-900/50"} rounded-2xl border p-4 flex flex-col gap-3`}>
                <div className="flex items-center gap-2">
                  {POOL_ICONS[pool.id] || <Gift className="w-5 h-5 text-zinc-400" />}
                  <div>
                    <div className="text-xs font-bold text-zinc-100">{pool.name}</div>
                    <div className="text-[9px] text-zinc-500 font-mono">
                      {pool.cost_type === "primeval_stones" ? `${pool.cost_amount.toLocaleString()} Nguyên Thạch / lần` : `${pool.cost_amount} Tiên Nguyên Thạch / lần`}
                    </div>
                  </div>
                </div>
                <div className="text-[10px] text-zinc-400 leading-relaxed min-h-[40px]">{pool.description}</div>

                <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/60 rounded-xl px-3 py-2">
                  <span className="text-[9px] text-zinc-500 font-mono">Tổng chi phí</span>
                  <span className="text-[10px] font-mono font-bold text-yellow-500">{costLabel(pool)}</span>
                </div>

                <button
                  disabled={actionLoading || cooldownLeft > 0 || !canAfford(pool)}
                  onClick={() => handleOpen(pool.id, quantity)}
                  className={`w-full py-2.5 ${POOL_BTN[pool.id] || "bg-amber-600 hover:bg-amber-500"} disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-xl text-[11px] font-bold font-mono cursor-pointer transition-all flex items-center justify-center gap-1.5`}
                >
                  {cooldownLeft > 0 ? (
                    <>
                      <Clock className="w-3 h-3" />
                      Chờ {cooldownLeft}s
                    </>
                  ) : (
                    <>
                      <Repeat className="w-3 h-3" />
                      {quantity > 1 ? `Mở ${quantity} lần` : "Mở 1"}
                    </>
                  )}
                </button>
                {!canAfford(pool) && (
                  <span className="text-[9px] text-red-400/80 font-mono text-center">Không đủ tài nguyên</span>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};
