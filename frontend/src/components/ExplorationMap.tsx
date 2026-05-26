"use client";

import React, { useState } from "react";
import { Player, useGameStore } from "@/store/useGameStore";
import { Compass, Sparkles, MapPin, ShieldAlert, Award, Footprints } from "lucide-react";

interface ExplorationMapProps {
  player: Player;
  actionLoading: boolean;
  onExplore: (nodeId: string) => Promise<{ player: any; log: string; combat_triggered: boolean } | null>;
  onBack?: () => void;
}

interface MapNode {
  id: string;
  name: string;
  region: string;
  recommendedRank: string;
  description: string;
  colorClass: string;
  bgGlow: string;
}

const MAP_NODES: MapNode[] = [
  {
    id: "qing_mao_mountain",
    name: "Thanh Mao Sơn",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhất Chuyển (Rank 1)",
    description: "Ngọn núi phủ đầy trúc xanh, nơi có ba gia tộc lớn sinh sống. Có dã thú cấp thấp rình rập trong rừng sâu.",
    colorClass: "text-emerald-400 border-emerald-500/20 hover:border-emerald-500/50 bg-emerald-950/10",
    bgGlow: "from-emerald-500/5 to-transparent",
  },
  {
    id: "gu_luan_slope",
    name: "Cổ Loạn Sườn",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhất - Nhị Chuyển (Rank 1-2)",
    description: "Khu vực núi đá hiểm trở hoang sơ, chứa nhiều bộ xương khô dã thú cổ đại và các bầy thú dữ lang thang.",
    colorClass: "text-blue-400 border-blue-500/20 hover:border-blue-500/50 bg-blue-950/10",
    bgGlow: "from-blue-500/5 to-transparent",
  },
  {
    id: "three_kings_inheritance",
    name: "Tam Vương Truyền Thừa",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhị - Tam Chuyển (Rank 2-3)",
    description: "Bảo cảnh kế thừa của ba vị ma đạo cổ sư: Thiết Vương, Khương Vương, và Tín Vương. Vô cùng hung hiểm.",
    colorClass: "text-purple-400 border-purple-500/20 hover:border-purple-500/50 bg-purple-950/10",
    bgGlow: "from-purple-500/5 to-transparent",
  },
];

export const ExplorationMap: React.FC<ExplorationMapProps> = ({
  player,
  actionLoading,
  onExplore,
  onBack,
}) => {
  const [lastLog, setLastLog] = useState<string | null>(null);
  const [activeNode, setActiveNode] = useState<string | null>(null);

  const handleNodeClick = async (node: MapNode) => {
    if (actionLoading || player.active_combat) return;

    setActiveNode(node.id);
    setLastLog(null);

    try {
      const res = await onExplore(node.id);
      if (res) {
        setLastLog(res.log);
      }
    } finally {
      setActiveNode(null);
    }
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2">
          <Compass className="w-5 h-5 text-amber-500 animate-spin" />
          <h3 className="text-lg font-bold text-zinc-100 font-serif">
            Cổ Giới Khám Phá (World Map & Inheritances)
          </h3>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-[10px] text-zinc-500 font-mono">
            Vị trí hiện tại: {player.current_region}
          </span>
          {onBack && (
            <button
              type="button"
              onClick={onBack}
              disabled={actionLoading}
              className="py-1 px-3 border border-zinc-800 rounded-xl text-[10px] text-zinc-400 hover:bg-zinc-900 transition-all cursor-pointer font-mono font-bold"
            >
              Trở Về
            </button>
          )}
        </div>
      </div>

      <div className="text-xs text-zinc-400 max-w-xl">
        * Mỗi lần cất bước khám phá sẽ tiêu hao <span className="text-emerald-400 font-bold">5 Chân Nguyên</span>. Có xác suất chạm trán dã thú dữ dội, lượm lặt Nguyên Thạch hoặc dược liệu bản địa.
      </div>

      {/* Grid Nodes */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {MAP_NODES.map((node) => {
          const isCurrentExploring = activeNode === node.id;
          const hasMana = player.current_mana >= 5;

          return (
            <div
              key={node.id}
              className={`border rounded-2xl p-4 flex flex-col justify-between h-56 transition-all bg-gradient-to-b ${node.bgGlow} to-zinc-950 relative overflow-hidden group ${
                node.colorClass
              } ${!!player.active_combat ? "opacity-40 cursor-not-allowed" : ""}`}
            >
              <div className="space-y-2">
                <div className="flex items-start justify-between">
                  <span className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">
                    {node.region}
                  </span>
                  <MapPin className="w-3.5 h-3.5 opacity-60 group-hover:scale-110 transition-transform" />
                </div>

                <h4 className="text-base font-bold text-zinc-100 font-serif group-hover:text-amber-400 transition-colors">
                  {node.name}
                </h4>

                <div className="flex items-center gap-1 text-[10px] text-zinc-500 font-mono">
                  <Award className="w-3 h-3 text-amber-500" />
                  Khuyên dùng: {node.recommendedRank}
                </div>

                <p className="text-[11px] text-zinc-400 leading-normal line-clamp-3">
                  {node.description}
                </p>
              </div>

              <div className="mt-4 pt-3 border-t border-zinc-900/60">
                <button
                  type="button"
                  disabled={actionLoading || !!player.active_combat || !hasMana}
                  onClick={() => handleNodeClick(node)}
                  className={`w-full py-2 px-3 text-xs font-bold rounded-xl flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                    hasMana && !player.active_combat
                      ? "bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-zinc-200 active:scale-[0.98]"
                      : "bg-zinc-950 text-zinc-600 border border-zinc-900/80 cursor-not-allowed"
                  }`}
                >
                  {isCurrentExploring ? (
                    <>
                      <Compass className="w-3.5 h-3.5 animate-spin text-amber-400" />
                      <span>Đang hành trình...</span>
                    </>
                  ) : (
                    <>
                      <Footprints className="w-3.5 h-3.5 text-emerald-400" />
                      <span>Cất Bước Khám Phá</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Exploration Log Overlay / Notification */}
      {lastLog && (
        <div className="bg-zinc-950/70 border border-zinc-900 rounded-2xl p-4 flex items-start gap-3 animate-fade-in">
          <div className="w-7 h-7 rounded-lg bg-amber-500/10 flex items-center justify-center shrink-0 mt-0.5">
            <Sparkles className="w-4 h-4 text-amber-400 animate-pulse" />
          </div>
          <div className="space-y-1">
            <span className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono block">
              Ghi chép du hành (Explorer Log)
            </span>
            <p className="text-xs text-zinc-200 leading-relaxed font-mono">
              {lastLog}
            </p>
          </div>
        </div>
      )}

      {player.active_combat && (
        <div className="bg-rose-950/20 border border-rose-900/40 rounded-2xl p-4 flex items-center justify-between animate-pulse">
          <div className="flex items-center gap-2.5">
            <ShieldAlert className="w-5 h-5 text-rose-500" />
            <div className="text-left">
              <div className="text-xs font-bold text-rose-400 font-serif">Đang chạm trán dã thú!</div>
              <div className="text-[10px] text-zinc-500">
                Đạo hữu phải giải quyết cuộc chiến trước khi tiếp tục hành trình.
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
