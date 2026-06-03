"use client";

import React, { useState } from "react";
import { Player, GuSpec, useGameStore } from "@/store/useGameStore";
import { Award, Shield, Compass, Sparkles, PackageOpen, ShieldAlert, Zap, Compass as RadarIcon, Heart, Clock } from "lucide-react";

interface PlayerStatsProps {
  player: Player;
  onLogout: () => void;
  guDictionary: Record<string, GuSpec>;
  onEquipToggle: (instanceId: string, equip: boolean) => Promise<void>;
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

export const PlayerStats: React.FC<PlayerStatsProps> = ({
  player,
  onLogout,
  guDictionary,
  onEquipToggle,
}) => {
  const [actionLoading, setActionLoading] = useState(false);

  // Compute stats
  const stats = {
    attack: 10 + (player.rank - 1) * 10 + (player.sub_rank - 1) * 2 + (player.dao_marks?.["strength"] || 0) / 10,
    defense: 5 + (player.rank - 1) * 5 + (player.sub_rank - 1) * 1,
    healing: 0,
    movement: 1,
    investigation: 1,
    storage: 20,
  };

  // Add passive permanent body modifications buffs
  if (player.body_mods) {
    player.body_mods.forEach((mod) => {
      switch (mod) {
        case "Iron Bone":
          stats.defense += 15;
          stats.attack += 5;
          break;
        case "Jade Skin Mod":
          stats.defense += 25;
          break;
        case "Wind Walk":
          stats.movement += 1;
          break;
      }
    });
  }

  player.gu_inventory.forEach((gu) => {
    if (gu.equipped && gu.status === "ACTIVE") {
      const spec = guDictionary[gu.gu_id];
      if (spec) {
        let masteryMultiplier = 1.0;
        if (player.path_masteries && player.path_masteries[spec.path]) {
          const mLevel = player.path_masteries[spec.path];
          if (mLevel === "Master") masteryMultiplier = 1.1;
          else if (mLevel === "Grandmaster") masteryMultiplier = 1.25;
          else if (mLevel === "Supreme Grandmaster") masteryMultiplier = 1.5;
        }

        if (spec.effects) {
          if (spec.effects.damage) stats.attack += spec.effects.damage * masteryMultiplier;
          if (spec.effects.add_base_attack) stats.attack += spec.effects.add_base_attack * masteryMultiplier;
          if (spec.effects.defense) stats.defense += spec.effects.defense * masteryMultiplier;
          if (spec.effects.add_base_defense) stats.defense += spec.effects.add_base_defense * masteryMultiplier;
          if (spec.effects.refine_mana_tier) stats.healing += spec.effects.refine_mana_tier * 10;
          if (spec.effects.cleansing_power) stats.healing += spec.effects.cleansing_power * 15;
          if (spec.effects.movement) stats.movement += spec.effects.movement;
          if (spec.effects.investigation) stats.investigation += spec.effects.investigation;
          if (spec.effects.storage) stats.storage += spec.effects.storage;
        }
      }
    }
  });

  if (player.rank >= 6) {
    stats.storage = 999;
  }

  // Generate SVG Radar Chart points
  const center = 100;
  const axes = [
    { key: "attack", angle: -Math.PI / 2, max: 100, label: "Tấn Công" },
    { key: "defense", angle: -Math.PI / 6, max: 100, label: "Phòng Thủ" },
    { key: "healing", angle: Math.PI / 6, max: 100, label: "Trị Liệu" },
    { key: "movement", angle: Math.PI / 2, max: 5, label: "Tốc Độ" },
    { key: "investigation", angle: (5 * Math.PI) / 6, max: 5, label: "Tầm Nhìn" },
    { key: "storage", angle: (7 * Math.PI) / 6, max: 100, label: "Hộp Trữ" },
  ];

  const radarPoints = axes.map((axis) => {
    const rawVal = stats[axis.key as keyof typeof stats] || 0;
    const val = Math.min(axis.max, rawVal);
    const ratio = val / axis.max;
    const radius = 15 + ratio * 65; // radius from 15 to 80
    const x = center + radius * Math.cos(axis.angle);
    const y = center + radius * Math.sin(axis.angle);
    return { x, y, label: axis.label, val: rawVal, angle: axis.angle };
  });

  const radarPath = radarPoints.map((p) => `${p.x},${p.y}`).join(" ");

  // Grid steps (20%, 40%, 60%, 80%, 100%)
  const gridSteps = [0.2, 0.4, 0.6, 0.8, 1.0];

  const equippedGus = player.gu_inventory.filter((gu) => gu.equipped && gu.status === "ACTIVE");

  const handleUnequip = async (instanceId: string) => {
    setActionLoading(true);
    try {
      await onEquipToggle(instanceId, false);
    } finally {
      setActionLoading(false);
    }
  };

  const getPathColor = (path: string): string => {
    switch (path) {
      case "Moon Path":
        return "text-indigo-400 border-indigo-900/40 bg-indigo-950/20";
      case "Strength Path":
        return "text-orange-400 border-orange-900/40 bg-orange-950/20";
      case "Earth Path":
        return "text-amber-400 border-amber-900/40 bg-amber-950/20";
      case "Refinement Path":
        return "text-emerald-400 border-emerald-900/40 bg-emerald-950/20";
      case "Rule Path":
        return "text-sky-400 border-sky-900/40 bg-sky-950/20";
      default:
        return "text-zinc-400 border-zinc-900/40 bg-zinc-950/20";
    }
  };

  return (
    <div className="glass-panel rounded-3xl p-4 sm:p-6 w-full flex flex-col justify-between h-full relative overflow-hidden group">
      {/* Decorative background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(16,185,129,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(16,185,129,0.01)_1px,transparent_1px)] bg-[size:14px_24px] pointer-events-none" />

      <div>
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping" />
            <span className="text-[9px] text-emerald-400 font-mono uppercase tracking-widest">
              X: {player.x} | Y: {player.y}
            </span>
          </div>
          <button
            onClick={onLogout}
            className="text-[10px] text-zinc-500 hover:text-rose-450 transition-colors cursor-pointer border border-zinc-800 hover:border-rose-900/30 px-2 py-0.5 rounded bg-zinc-950/40 font-mono"
          >
            Đăng Xuất
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <span className="text-[9px] text-zinc-500 uppercase tracking-wider block mb-0.5">
              {player.rank >= 6 ? "Cổ Tiên Tiên Khiếu" : "Cổ Sư Không Khiếu"}
            </span>
            <h2 className="text-xl font-bold text-zinc-100 flex items-center gap-1.5 font-serif">
              {player.username}
              <Sparkles className="w-3.5 h-3.5 text-amber-500 animate-pulse" />
            </h2>
            {player.lifespan <= 0 && (
              <div className="bg-rose-950/40 border border-rose-500/20 text-rose-450 p-2 rounded-xl text-[10px] font-mono animate-pulse mt-2">
                ⚠️ LÃO HÓA: Thọ nguyên đã tận! Khí hải khô kiệt, hồi phục giảm 50%. Hãy tìm Thọ Cổ!
              </div>
            )}
          </div>

          <div className="border-t border-zinc-900 my-1" />

          {/* Stats Grid */}
          <div className="grid grid-cols-2 gap-2.5 text-xs">
            <div className="bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-900">
              <div className="flex items-center gap-1 text-zinc-500 mb-0.5">
                <Award className="w-3 h-3 text-amber-500" />
                <span className="text-[8px] uppercase tracking-wider">Tu Vi</span>
              </div>
              <span className="text-xs font-semibold text-zinc-300 block truncate">
                {getRankName(player.rank)}
              </span>
            </div>

            <div className="bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-900">
              <div className="flex items-center gap-1 text-zinc-500 mb-0.5">
                <Shield className="w-3 h-3 text-emerald-500" />
                <span className="text-[8px] uppercase tracking-wider">Cảnh Giới</span>
              </div>
              <span className="text-xs font-semibold text-zinc-300 block">
                {getSubRankName(player.sub_rank)}
              </span>
            </div>

            <div className="bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-900">
              <div className="flex items-center gap-1 text-zinc-500 mb-0.5">
                <ShieldAlert className="w-3 h-3 text-rose-400" />
                <span className="text-[8px] uppercase tracking-wider">Thể Trạng</span>
              </div>
              <span className="text-xs font-semibold text-zinc-300 block">
                {Math.round(player.hp)} / {Math.round(player.base_hp)}
              </span>
            </div>

            <div className="bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-900">
              <div className="flex items-center gap-1 text-zinc-500 mb-0.5">
                <Heart className="w-3 h-3 text-rose-500" />
                <span className="text-[8px] uppercase tracking-wider">Thọ Nguyên</span>
              </div>
              <span className="text-xs font-semibold text-zinc-300 block">
                {player.lifespan !== undefined ? player.lifespan.toFixed(1) : "0"}/{player.max_lifespan !== undefined ? player.max_lifespan.toFixed(0) : "100"} năm
              </span>
            </div>

            <div className="bg-zinc-950/40 p-2.5 rounded-xl border border-zinc-900 col-span-2">
              <div className="flex items-center gap-1 text-zinc-500 mb-0.5">
                <Clock className="w-3 h-3 text-cyan-400" />
                <span className="text-[8px] uppercase tracking-wider">Hệ Số Lưu Tốc Tiên Khiếu (Aperture Flow)</span>
              </div>
              <span className="text-xs font-semibold text-zinc-300 block">
                Nhanh gấp x{player.time_speed_multiplier !== undefined ? player.time_speed_multiplier.toFixed(0) : "1"} lần
              </span>
            </div>
          </div>

          {/* SVG Radar Chart Display */}
          <div className="flex justify-center items-center py-2 relative">
            <svg viewBox="0 0 200 200" className="w-full max-w-[200px] overflow-visible">
              {/* Concentric grid lines */}
              {gridSteps.map((step, idx) => {
                const r = 15 + step * 65;
                const gridPoints = axes.map((axis) => {
                  const gx = center + r * Math.cos(axis.angle);
                  const gy = center + r * Math.sin(axis.angle);
                  return `${gx},${gy}`;
                }).join(" ");
                return (
                  <polygon
                    key={idx}
                    points={gridPoints}
                    className="stroke-zinc-800/60 fill-none"
                    strokeWidth="0.5"
                  />
                );
              })}

              {/* Radial lines */}
              {axes.map((axis, idx) => {
                const lx = center + 80 * Math.cos(axis.angle);
                const ly = center + 80 * Math.sin(axis.angle);
                return (
                  <line
                    key={idx}
                    x1={center}
                    y1={center}
                    x2={lx}
                    y2={ly}
                    className="stroke-zinc-900"
                    strokeWidth="0.5"
                  />
                );
              })}

              {/* Glowing outer boundary */}
              <polygon
                points={axes.map((axis) => `${center + 80 * Math.cos(axis.angle)},${center + 80 * Math.sin(axis.angle)}`).join(" ")}
                className="stroke-zinc-800/80 fill-none"
                strokeWidth="1"
              />

              {/* Actual stats polygon */}
              <polygon
                points={radarPath}
                className="fill-emerald-500/10 stroke-emerald-500 shadow-lg"
                strokeWidth="1.5"
              />

              {/* Vertices circles */}
              {radarPoints.map((p, idx) => (
                <circle
                  key={idx}
                  cx={p.x}
                  cy={p.y}
                  r="3.5"
                  className="fill-emerald-450 stroke-zinc-950"
                  strokeWidth="1"
                >
                  <title>{`${p.label}: ${p.val.toFixed(0)}`}</title>
                </circle>
              ))}

              {/* Labels */}
              {axes.map((axis, idx) => {
                const lx = center + 95 * Math.cos(axis.angle);
                const ly = center + 95 * Math.sin(axis.angle);
                const cos = Math.cos(axis.angle);
                const anchor = Math.abs(cos) < 0.1 ? "middle" : cos > 0 ? "start" : "end";
                return (
                  <text
                    key={idx}
                    x={lx}
                    y={ly + 3}
                    textAnchor={anchor}
                    className="fill-zinc-400 font-mono font-bold text-[8px]"
                  >
                    {axis.label} ({stats[axis.key as keyof typeof stats].toFixed(0)})
                  </text>
                );
              })}
            </svg>
          </div>

          {/* Path Masteries Section */}
          <div className="mt-4 pt-3 border-t border-zinc-900/60">
            <span className="text-[9px] text-zinc-500 uppercase tracking-wider block mb-2 font-mono">
              Lĩnh Ngộ Lưu Phái (Path Masteries)
            </span>
            <div className="space-y-1.5 max-h-36 overflow-y-auto pr-1">
              {player.path_masteries && Object.keys(player.path_masteries).length > 0 ? (
                Object.entries(player.path_masteries).map(([path, level]) => (
                  <div key={path} className="flex justify-between items-center bg-zinc-950/30 px-2.5 py-1.5 rounded-lg border border-zinc-900/50 text-xs">
                    <span className="text-zinc-400 font-medium">{path}</span>
                    <span className={`px-2 py-0.5 rounded text-[10px] font-semibold font-mono ${
                      level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                      level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                      level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                      "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                    }`}>
                      {level}
                    </span>
                  </div>
                ))
              ) : (
                <div className="text-[10px] text-zinc-600 italic">Chưa có lĩnh ngộ lưu phái đặc thù.</div>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="mt-4 pt-3 border-t border-zinc-900 text-[9px] text-zinc-600 flex justify-between">
        <span>Khai thiên lập địa:</span>
        <span>{new Date(player.created_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
};
