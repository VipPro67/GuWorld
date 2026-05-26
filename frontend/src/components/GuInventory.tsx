"use client";

import React, { useState } from "react";
import { Player, GuSpec, ItemSpec } from "@/store/useGameStore";
import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle } from "lucide-react";

interface GuInventoryProps {
  player: Player;
  onFeed: (guInstanceId: string) => Promise<void>;
  actionLoading: boolean;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
}

export const GuInventory: React.FC<GuInventoryProps> = ({
  player,
  onFeed,
  actionLoading,
  guDictionary,
  itemDictionary,
}) => {
  const [feedingId, setFeedingId] = useState<string | null>(null);

  const materials = player.materials_inventory || {};
  const guList = player.gu_inventory || [];

  const handleFeedClick = async (instanceId: string) => {
    setFeedingId(instanceId);
    try {
      await onFeed(instanceId);
    } finally {
      setFeedingId(null);
    }
  };

  // Helper to render material icon
  const renderMaterialIcon = (key: string) => {
    switch (key) {
      case "moon_orchid_petal":
        return <Flower2 className="w-4 h-4 text-emerald-400" />;
      case "green_bamboo_wine":
        return <Wine className="w-4 h-4 text-amber-500" />;
      case "jade_stone":
        return <Gem className="w-4 h-4 text-cyan-400" />;
      default:
        return <HelpCircle className="w-4 h-4 text-zinc-500" />;
    }
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-lg space-y-6">
      {/* Header */}
      <div className="flex items-center gap-2.5 border-b border-zinc-800/80 pb-3">
        <Sparkles className="w-5 h-5 text-amber-400 animate-pulse" />
        <h3 className="text-lg font-bold text-zinc-100 font-serif tracking-wide">
          Bách Bảo Nang (Inventory & Gu Bags)
        </h3>
      </div>

      {/* Materials Inventory section */}
      <div className="space-y-2">
        <span className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest block">
          Nguyên Liệu Nuôi Cổ (Materials)
        </span>
        <div className="flex flex-wrap gap-3">
          {Object.keys(itemDictionary).length === 0 ? (
            <div className="text-xs text-zinc-600 italic">Đang tải dược phẩm...</div>
          ) : (
            Object.keys(itemDictionary).map((key) => {
              const spec = itemDictionary[key];
              const qty = materials[key] || 0;
              return (
                <div
                  key={key}
                  className="flex items-center gap-2 bg-zinc-950/70 border border-zinc-900 px-3.5 py-2 rounded-xl"
                  title={spec.description}
                >
                  {renderMaterialIcon(key)}
                  <div className="text-left">
                    <div className="text-[10px] text-zinc-400 font-medium leading-none">
                      {spec.name}
                    </div>
                    <div className="text-xs font-bold text-zinc-200 mt-1 font-mono">
                      x{qty}
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Gu Bug list section */}
      <div className="space-y-3">
        <span className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest block">
          Cổ Trùng Bản Mệnh (My Gu Bugs)
        </span>

        {guList.length === 0 ? (
          <div className="text-center py-8 border border-dashed border-zinc-800 rounded-2xl text-xs text-zinc-600 italic">
            Chưa sở hữu bất kỳ Cổ trùng nào.
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {guList.map((gu) => {
              const spec = guDictionary[gu.gu_id] || {
                id: gu.gu_id,
                name: gu.gu_id,
                rank: 1,
                type: "Phàm Cổ",
                base_power: 0,
                hunger_rate: 0,
                food_type: "unknown",
                max_hunger: 100,
                description: "Cổ trùng chưa được giải nghĩa.",
              };

              const hunger = gu.hunger;
              const status = gu.status;
              const isDead = status === "DEAD";
              const isSleeping = status === "SLEEPING";

              // Find food specifications dynamically
              const foodSpec = itemDictionary[spec.food_type];
              const foodName = foodSpec ? foodSpec.name : spec.food_type;
              const availableFood = materials[spec.food_type] || 0;
              const canFeed = availableFood > 0 && !isDead;
              const isCurrentlyFeeding = feedingId === gu.instance_id;

              // Choose hunger bar colors
              let barColor = "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]";
              if (hunger <= 20) {
                barColor = "bg-rose-500 animate-pulse shadow-[0_0_10px_rgba(239,68,68,0.5)]";
              } else if (hunger <= 50) {
                barColor = "bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.3)]";
              }

              return (
                <div
                  key={gu.instance_id}
                  className={`relative p-4 rounded-2xl border transition-all flex flex-col justify-between h-44 ${
                    isDead
                      ? "bg-zinc-950/30 border-zinc-900 opacity-40 select-none"
                      : isSleeping
                      ? "bg-zinc-950/70 border-amber-600/30 shadow-[inset_0_0_15px_rgba(245,158,11,0.05)]"
                      : "bg-zinc-950/60 border-zinc-850 hover:border-emerald-500/20"
                  }`}
                >
                  {/* Top: Name and Type */}
                  <div>
                    <div className="flex justify-between items-start gap-1">
                      <h4 className="text-sm font-bold text-zinc-100 font-serif">
                        {spec.name}
                      </h4>
                      <span className="text-[8px] font-mono font-bold text-amber-400 border border-amber-500/20 px-1 py-0.5 rounded uppercase shrink-0">
                        Rank {spec.rank}
                      </span>
                    </div>
                    <span className="text-[9px] text-zinc-500 font-mono block mt-0.5">
                      {spec.type}
                    </span>
                    <p className="text-[10px] text-zinc-400 mt-2 line-clamp-2 leading-relaxed">
                      {spec.description}
                    </p>
                  </div>

                  {/* Bottom: Hunger bar and Feeding */}
                  <div className="mt-4 space-y-2">
                    {/* Hunger bar */}
                    {!isDead && (
                      <div className="space-y-1">
                        <div className="flex justify-between text-[9px] font-mono">
                          <span className="text-zinc-500">Độ no (Hunger):</span>
                          <span
                            className={
                              hunger <= 20
                                ? "text-rose-400 font-bold"
                                : hunger <= 50
                                ? "text-yellow-400"
                                : "text-emerald-400"
                            }
                          >
                            {hunger}/{spec.max_hunger}
                          </span>
                        </div>
                        <div className="w-full bg-zinc-900 rounded-full h-1.5 overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all duration-500 ${barColor}`}
                            style={{ width: `${(hunger / spec.max_hunger) * 100}%` }}
                          />
                        </div>
                      </div>
                    )}

                    {/* Status Badge & Button */}
                    <div className="flex items-center justify-between gap-2 pt-1 border-t border-zinc-900/60">
                      {/* Status display */}
                      {isDead ? (
                        <div className="flex items-center gap-1 text-[10px] text-rose-500 font-bold uppercase tracking-wider">
                          <ShieldAlert className="w-3 h-3" />
                          Đã Chết
                        </div>
                      ) : isSleeping ? (
                        <span className="inline-block text-[9px] font-bold text-amber-500 uppercase bg-amber-950/20 border border-amber-500/20 px-1.5 py-0.5 rounded animate-pulse">
                          Ngủ Đông
                        </span>
                      ) : (
                        <span className="inline-block text-[9px] font-bold text-emerald-400 uppercase bg-emerald-950/20 border border-emerald-500/20 px-1.5 py-0.5 rounded">
                          Hoạt Động
                        </span>
                      )}

                      {/* Feed button */}
                      {!isDead && (
                        <button
                          type="button"
                          disabled={actionLoading || !canFeed}
                          onClick={() => handleFeedClick(gu.instance_id)}
                          title={`Ăn: 1x ${foodName}`}
                          className={`text-[10px] font-semibold py-1.5 px-3 rounded-lg flex items-center justify-center gap-1 transition-all cursor-pointer ${
                            canFeed
                              ? "bg-emerald-600 text-white hover:bg-emerald-500 shadow-md shadow-emerald-900/10 active:scale-[0.96]"
                              : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                          }`}
                        >
                          {isCurrentlyFeeding ? (
                            <div className="w-2.5 h-2.5 border border-zinc-300 border-t-transparent rounded-full animate-spin" />
                          ) : (
                            "Cho Ăn"
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
