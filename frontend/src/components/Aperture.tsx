"use client";
import React, { useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GuInstance, GuSpec } from "@/store/useGameStore";
import {
  Sparkles,
  ShieldAlert,
  Award,
  Zap,
  Flame,
  Coins,
  Atom,
  Crown,
  Heart,
  Activity,
  Compass,
} from "lucide-react";

interface ApertureProps {
  currentMana: number;
  maxMana: number;
  aptitudeType: string;
  aptitudePercent: number;
  rank: number;
  subRank: number;
  guInventory?: GuInstance[];
  guDictionary?: Record<string, GuSpec>;
}

const RANK_NAMES: Record<number, string> = {
  1: "Nhất Chuyển",
  2: "Nhị Chuyển",
  3: "Tam Chuyển",
  4: "Tứ Chuyển",
  5: "Ngũ Chuyển",
  6: "Lục Chuyển",
  7: "Thất Chuyển",
  8: "Bát Chuyển",
  9: "Cửu Chuyển",
};

const SUB_RANK_NAMES: Record<number, string> = {
  1: "Sơ Giai",
  2: "Trung Giai",
  3: "Cao Giai",
  4: "Đỉnh Phong",
};

const MANA_QUALITY: Record<number, string> = {
  1: "Thanh Đồng Chân Nguyên",
  2: "Xích Thiết Chân Nguyên",
  3: "Bạch Ngân Chân Nguyên",
  4: "Hoàng Kim Chân Nguyên",
  5: "Tử Tinh Chân Nguyên",
  6: "Thanh Đề Tiên Nguyên",
  7: "Hồng Tảo Tiên Nguyên",
  8: "Bạch Lệ Tiên Nguyên",
  9: "Hoàng Hạnh Tiên Nguyên",
};

const RANK_COLORS: Record<number, {
  from: string; to: string; glow: string; ring: string;
  text: string; waveSolid: string; waveAlpha: string;
}> = {
  1: { from: "#1b4332", to: "#081c15", glow: "rgba(52,211,153,0.35)",  ring: "#2d6a4f", text: "#6ee7b7", waveSolid: "#10b981", waveAlpha: "rgba(16,185,129,0.4)" },
  2: { from: "#450a0a", to: "#1a0505", glow: "rgba(239,68,68,0.35)",   ring: "#b91c1c", text: "#fca5a5", waveSolid: "#ef4444", waveAlpha: "rgba(239,68,68,0.4)"  },
  3: { from: "#0f172a", to: "#020617", glow: "rgba(59,130,246,0.35)",  ring: "#2563eb", text: "#93c5fd", waveSolid: "#3b82f6", waveAlpha: "rgba(59,130,246,0.4)" },
  4: { from: "#3b1f00", to: "#1c0f00", glow: "rgba(245,158,11,0.35)",  ring: "#d97706", text: "#fde68a", waveSolid: "#f59e0b", waveAlpha: "rgba(245,158,11,0.4)" },
  5: { from: "#2e1065", to: "#12002f", glow: "rgba(139,92,246,0.4)",   ring: "#7c3aed", text: "#c4b5fd", waveSolid: "#8b5cf6", waveAlpha: "rgba(139,92,246,0.4)" },
  6: { from: "#083344", to: "#02151e", glow: "rgba(6,182,212,0.4)",    ring: "#0891b2", text: "#67e8f9", waveSolid: "#06b6d4", waveAlpha: "rgba(6,182,212,0.4)"  },
  7: { from: "#500724", to: "#25000f", glow: "rgba(236,72,153,0.4)",   ring: "#db2777", text: "#f9a8d4", waveSolid: "#ec4899", waveAlpha: "rgba(236,72,153,0.4)" },
  8: { from: "#1c1917", to: "#0c0a09", glow: "rgba(168,162,158,0.4)",  ring: "#a8a29e", text: "#e7e5e4", waveSolid: "#a8a29e", waveAlpha: "rgba(168,162,158,0.4)" },
  9: { from: "#78350f", to: "#2d0f00", glow: "rgba(251,191,36,0.55)",  ring: "#fbbf24", text: "#fef08a", waveSolid: "#fbbf24", waveAlpha: "rgba(251,191,36,0.5)"  },
};

// SVG layout constants — expanded to 300 viewBox to prevent text clipping
const VB   = 300;          
const CX   = 150;          
const CY   = 150;          
const R    = 80;           
const CIRC = 2 * Math.PI * R;
const ORBIT_R = 114;       

// Build a sinusoidal wave path that spans 3× the orb width
function buildWavePath(amplitude: number, wavelength: number, phase: number): string {
  const steps = 60;
  const totalWidth = VB * 3;
  const startX = -VB;
  let d = `M ${startX} 0`;
  for (let i = 0; i <= steps; i++) {
    const x = startX + (i / steps) * totalWidth;
    const y = amplitude * Math.sin((2 * Math.PI * (x - phase)) / wavelength);
    d += ` L ${x.toFixed(2)} ${y.toFixed(2)}`;
  }
  d += ` L ${startX + totalWidth} ${VB} L ${startX} ${VB} Z`;
  return d;
}

// Cleanly abbreviate Vietnamese Gu names (e.g. "Nguyệt Quang Cổ" -> "Nguyệt Q.")
const formatGuName = (name: string) => {
  const clean = name.replace(/\s*Cổ\s*$/, "");
  if (clean.length <= 7) return clean;
  const parts = clean.split(" ");
  if (parts.length >= 2) {
    return parts.slice(0, -1).join(" ") + " " + parts[parts.length - 1][0] + ".";
  }
  return clean.slice(0, 6) + ".";
};

export const Aperture: React.FC<ApertureProps> = ({
  currentMana,
  maxMana,
  aptitudeType,
  aptitudePercent,
  rank,
  subRank,
  guInventory = [],
  guDictionary = {},
}) => {
  const store = useGameStore();
  const playerFromStore = store.player;
  const percent   = Math.min(100, Math.max(0, (currentMana / maxMana) * 100));
  const rankName  = RANK_NAMES[rank]    ?? `Rank ${rank}`;
  const subRankName = SUB_RANK_NAMES[subRank] ?? `Sub ${subRank}`;
  const isImmortal = rank >= 6;
  const energyType = isImmortal ? "Tiên Nguyên" : "Chân Nguyên";
  const apertureName = isImmortal ? "Tiên Khiếu (Immortal Aperture)" : "Không Khiếu (Aperture)";
  const energyPoolName = isImmortal ? "Tiên Nguyên Hải" : "Chân Nguyên Hải";
  const regenLabel = isImmortal ? "Tiên Khiếu ngưng tụ" : "Tốc độ hồi phục";
  const manaQuality = MANA_QUALITY[rank] ?? energyType;
  const c = RANK_COLORS[rank] ?? RANK_COLORS[1];

  const currentHP = playerFromStore?.hp ?? 0;
  const maxHP = playerFromStore?.base_hp ?? 0;
  const hpPercent = maxHP > 0 ? Math.min(100, Math.max(0, (currentHP / maxHP) * 100)) : 0;

  const dashOffset = CIRC * (1 - percent / 100);

  const immortalBaseRegen: Record<number, number> = { 6: 0.5, 7: 0.8, 8: 1.2, 9: 2.0 };
  const mortalRegen = aptitudeType === "Giáp" ? 8 : aptitudeType === "Ất" ? 6 : aptitudeType === "Bính" ? 4 : 2;
  const immortalRegen = (immortalBaseRegen[rank] ?? 0.5 + (rank - 6) * 0.35) * (1 + (Math.min(Math.max(subRank, 1), 4) - 1) * 0.15);
  const regenRate = (isImmortal ? immortalRegen : mortalRegen).toFixed(1);

  // Vital Gu
  const vitalGu   = guInventory.find((g) => g.is_vital);
  const vitalSpec = vitalGu ? guDictionary[vitalGu.gu_id] : null;

  // Equipped Gu (max 6)
  let equippedGus = guInventory.filter((g) => g.equipped && g.status !== "DEAD" && g.is_vital !== true);
  equippedGus = equippedGus.slice(0, 6);

  // Orbit dot positions — evenly spaced, starting from top
  const orbitDots = equippedGus.map((g, i) => {
    const angle = (i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2;
    return {
      g,
      spec: guDictionary[g.gu_id],
      x: CX + ORBIT_R * Math.cos(angle),
      y: CY + ORBIT_R * Math.sin(angle),
    };
  });

  // Fill level
  const fillTop = CY + R - (percent / 100) * (R * 2);

  // Wave paths
  const wave1 = buildWavePath(4, 55, 0);
  const wave2 = buildWavePath(3, 40, 15);

  const gradId   = `orbGrad-${rank}`;
  const clipId   = `orbClip-${rank}`;
  const animId1  = `aperture-wave1-${rank}`;
  const animId2  = `aperture-wave2-${rank}`;

  return (
    <>
      <style>{`
        @keyframes ${animId1} {
          from { transform: translateX(0); }
          to   { transform: translateX(${VB}px); }
        }
        @keyframes ${animId2} {
          from { transform: translateX(0); }
          to   { transform: translateX(-${VB}px); }
        }
        .wave-group-1-${rank} { animation: ${animId1} 4.5s linear infinite; }
        .wave-group-2-${rank} { animation: ${animId2} 6.5s linear infinite; }
      `}</style>

      <div 
        className="flex flex-col items-center justify-center p-5 glass-panel-gold rounded-3xl w-full border border-amber-500/20 transition-all duration-300"
        style={{ 
          boxShadow: `0 8px 32px 0 rgba(0, 0, 0, 0.45), 0 0 25px ${c.glow}20`,
          borderColor: c.ring + "2a"
        }}
      >
        {/* Header */}
        <h3
          className="text-base sm:text-lg font-bold tracking-wider uppercase font-serif mb-1 flex items-center gap-1.5"
          style={{ color: c.text, textShadow: `0 0 10px ${c.glow}` }}
        >
          {apertureName}
        </h3>
        <p className="text-[10px] text-zinc-400 text-center mb-1 font-mono uppercase tracking-wide">
          {isImmortal ? "Tiên Phàm Cách Biệt" : `Tư Chất: ${aptitudeType} (${aptitudePercent}%)`} • {manaQuality}
        </p>
        <p className="text-xs font-semibold tracking-widest text-center mb-4 text-zinc-150 uppercase">
          Tu Vi: <span className="font-bold" style={{ color: c.text }}>{rankName}</span>&nbsp;&nbsp;❖&nbsp;&nbsp;Cảnh Giới: <span className="font-bold" style={{ color: c.text }}>{subRankName}</span>
        </p>

        {/* Health Bar (Sleek Glossy Bar) */}
        <div className="relative h-6 w-full overflow-hidden rounded-xl border border-red-950/60 bg-zinc-950/60 shadow-[inset_0_2px_4px_rgba(0,0,0,0.6)] mb-5 flex items-center px-2">
          <Heart className="w-3.5 h-3.5 text-rose-500 mr-2 shrink-0 animate-pulse" />
          <div className="relative flex-grow h-3 rounded-md overflow-hidden bg-zinc-900/60">
            <div
              className="h-full bg-gradient-to-r from-red-800 via-red-600 to-rose-500 rounded-md transition-all duration-300 shadow-[0_0_8px_rgba(239,68,68,0.5)]"
              style={{ width: `${hpPercent}%` }}
            />
          </div>
          <span className="text-[10px] font-mono font-bold text-zinc-300 ml-2.5 shrink-0">
            {playerFromStore
              ? `${Math.round(currentHP)}/${Math.round(maxHP)} (${hpPercent.toFixed(0)}%)`
              : "--"}
          </span>
        </div>

        {/* Orb Area */}
        <div className="relative w-full max-w-[280px] mx-auto mb-4" style={{ aspectRatio: "1 / 1" }}>
          {!isImmortal ? (
            <svg
              viewBox={`0 0 ${VB} ${VB}`}
              width="100%"
              height="100%"
              className="absolute inset-0 overflow-visible"
              style={{ filter: `drop-shadow(0 0 25px ${c.glow})` }}
            >
              <defs>
                <radialGradient id={gradId} cx="50%" cy="30%" r="70%">
                  <stop offset="0%" stopColor={c.from} />
                  <stop offset="100%" stopColor={c.to} />
                </radialGradient>
                <clipPath id={clipId}>
                  <circle cx={CX} cy={CY} r={R} />
                </clipPath>
              </defs>

              {/* Decorative spinning orbit ring */}
              <circle
                cx={CX}
                cy={CY}
                r={R + 12}
                fill="none"
                stroke={c.ring}
                strokeWidth="1"
                strokeDasharray="4, 7"
                opacity="0.25"
                className="animate-spin"
                style={{ animationDuration: "16s", transformOrigin: `${CX}px ${CY}px` }}
              />

              {/* Sphere Background */}
              <circle cx={CX} cy={CY} r={R} fill={`url(#${gradId})`} />

              {/* Liquid fill */}
              <g clipPath={`url(#${clipId})`}>
                <rect
                  x={CX - R} y={fillTop}
                  width={R * 2} height={R * 2 + 10}
                  fill={c.waveSolid}
                  opacity="0.25"
                />
                <g className={`wave-group-1-${rank}`} style={{ transformOrigin: `${CX}px ${fillTop}px` }}>
                  <path d={wave1} fill={c.waveSolid} opacity="0.65" transform={`translate(0, ${fillTop})`} />
                </g>
                <g className={`wave-group-2-${rank}`} style={{ transformOrigin: `${CX}px ${fillTop}px` }}>
                  <path d={wave2} fill={c.waveAlpha} opacity="0.5" transform={`translate(0, ${fillTop})`} />
                </g>
                {/* Surface line */}
                <rect x={CX - R} y={fillTop - 1} width={R * 2} height={2} fill="rgba(255,255,255,0.25)" />
              </g>

              {/* Glass shine crescent overlay */}
              <path 
                d={`M ${CX - R * 0.7} ${CY - R * 0.7} A ${R} ${R} 0 0 1 ${CX + R * 0.7} ${CY - R * 0.7} A ${R * 0.95} ${R * 0.9} 0 0 0 ${CX - R * 0.7} ${CY - R * 0.7} Z`} 
                fill="rgba(255,255,255,0.12)" 
              />

              {/* Orbit dots track (faint) */}
              <circle cx={CX} cy={CY} r={ORBIT_R} fill="none" stroke={c.ring} strokeWidth="1" strokeDasharray="3, 5" opacity="0.12" />

              {/* Progress arc border */}
              <circle cx={CX} cy={CY} r={R} fill="none" stroke={c.ring} strokeWidth="3" opacity="0.2" />
              <circle cx={CX} cy={CY} r={R} fill="none" stroke={c.ring} strokeWidth="3.5" strokeLinecap="round" strokeDasharray={CIRC} strokeDashoffset={dashOffset} transform={`rotate(-90 ${CX} ${CY})`} opacity="0.9" />

              {/* Orbit dots for equipped Gu */}
              {orbitDots.map(({ g, spec, x, y }, i) => (
                <g key={g.instance_id}>
                  {/* Connecting thread to center */}
                  <line
                    x1={CX + (R + 1) * Math.cos((i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2)}
                    y1={CY + (R + 1) * Math.sin((i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2)}
                    x2={x} y2={y}
                    stroke={c.ring}
                    strokeWidth="0.8"
                    opacity="0.35"
                  />
                  {/* Glowing background halo */}
                  <circle cx={x} cy={y} r={14} fill={c.glow} opacity="0.25" />
                  <circle cx={x} cy={y} r={11} fill="#0d0d12" stroke={c.ring} strokeWidth="1.5" opacity="0.95" />
                  <circle cx={x} cy={y} r={5} fill={c.waveSolid} opacity="0.9" className="animate-pulse" />
                  {/* Unclipped name tag */}
                  <text 
                    x={x} 
                    y={y + 19} 
                    textAnchor="middle" 
                    fontSize="8.5" 
                    fill="#e4e4e7" 
                    fontWeight="bold"
                    className="font-sans drop-shadow-[0_1px_3px_rgba(0,0,0,0.95)]"
                  >
                    {spec ? formatGuName(spec.name) : `G${i + 1}`}
                  </text>
                </g>
              ))}
            </svg>
          ) : (
            <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none z-10">
              <div className="flex flex-col items-center gap-1.5 p-4 bg-zinc-950/40 border border-cyan-500/10 rounded-3xl backdrop-blur-sm">
                <span className="text-3xl animate-bounce">💎</span>
                <div className="text-sm font-bold text-zinc-100 font-mono tracking-wide" style={{ color: c.text }}>
                  {manaQuality}:
                </div>
                <div className="text-xl font-bold font-mono text-cyan-400">
                  {playerFromStore ? playerFromStore.immortal_stones.toLocaleString() : "--"} viên
                </div>
                <div className="text-[10px] text-zinc-400 mt-0.5 tracking-wider uppercase font-semibold">{energyPoolName}</div>
                <div className="text-[10px] text-zinc-300 mt-2 font-mono" style={{ color: c.text }}>
                  Tốc lưu tiên khiếu: +{Math.round(Number(regenRate) * 24)} viên/ngày
                </div>
              </div>
            </div>
          )}

          {/* Center text for mortals */}
          {!isImmortal && (
            <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none z-10 select-none">
              <span className="text-3xl font-extrabold tracking-wide drop-shadow-[0_2px_8px_rgba(0,0,0,0.98)] font-mono" style={{ color: c.text }}>
                {percent.toFixed(1)}%
              </span>
              <span className="text-xs font-bold mt-1 drop-shadow-[0_1.5px_4px_rgba(0,0,0,0.95)] font-mono" style={{ color: c.text }}>
                {currentMana.toFixed(1)} / {maxMana.toFixed(0)}
              </span>
              <span className="text-[9px] mt-1 text-zinc-400 drop-shadow-[0_1px_3px_rgba(0,0,0,0.9)] uppercase font-semibold tracking-wider font-mono">
                {energyPoolName}
              </span>
            </div>
          )}
        </div>

        {/* Vital Gu Section */}
        <div className="w-full mb-4 bg-zinc-950/40 border border-zinc-900 rounded-2xl p-3">
          <div className="text-[9px] text-zinc-550 uppercase tracking-widest text-center mb-2 font-bold font-mono flex items-center justify-center gap-1.5">
            <Flame className="w-3.5 h-3.5 text-amber-500 animate-pulse" />
            Bản Mệnh Cổ (Vital Gu)
          </div>
          {vitalGu ? (
            <div
              className="flex items-center justify-between px-3 py-2 rounded-xl border max-w-sm mx-auto"
              style={{ borderColor: c.ring + "33", background: c.to + "44" }}
            >
              <div className="flex items-center gap-2">
                <span
                  className="w-2 h-2 rounded-full flex-shrink-0 animate-pulse"
                  style={{ background: c.waveSolid, boxShadow: `0 0 6px ${c.glow}` }}
                />
                <span className="text-xs font-bold text-zinc-100">
                  {vitalSpec ? vitalSpec.name : vitalGu.gu_id}
                </span>
              </div>
              <span className={`text-[9px] font-mono px-2 py-0.5 rounded-lg font-bold border ${
                vitalGu.status === "ACTIVE" 
                  ? "bg-emerald-950/20 text-emerald-450 border-emerald-500/20"
                  : vitalGu.status === "SLEEPING" 
                  ? "bg-amber-950/20 text-amber-400 border-amber-500/20"
                  : "bg-rose-950/20 text-rose-450 border-rose-500/20"
              }`}>
                {vitalGu.status === "ACTIVE" ? "ACTIVE"
                  : vitalGu.status === "SLEEPING" ? "SLEEPING"
                  : "DEAD"}
              </span>
            </div>
          ) : (
            <div className="text-[10px] text-zinc-500 text-center italic py-1">Không Có Bản Mệnh Cổ</div>
          )}
        </div>

        {/* Equipped Gu List */}
        <div className="w-full mb-4 bg-zinc-950/40 border border-zinc-900 rounded-2xl p-3">
          <div className="text-[9px] text-zinc-550 uppercase tracking-widest text-center mb-2.5 font-bold font-mono flex items-center justify-center gap-1.5">
            <Activity className="w-3.5 h-3.5 text-cyan-400" />
            Cổ Trùng Đang Trang Bị ({equippedGus.length})
          </div>
          {equippedGus.length > 0 ? (
            <div className="flex flex-wrap gap-2 justify-center">
              {equippedGus.map((g) => {
                const spec = guDictionary[g.gu_id];
                return (
                  <div
                    key={g.instance_id}
                    className="flex items-center gap-2 px-2.5 py-1 rounded-xl border text-xs font-semibold backdrop-blur-sm transition-all hover:scale-105"
                    style={{ borderColor: c.ring + "2d", background: c.to + "33" }}
                    title={spec?.description ?? g.gu_id}
                  >
                    <span
                      className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                      style={{ background: c.waveSolid, boxShadow: `0 0 4px ${c.glow}` }}
                    />
                    <span className="text-zinc-250">
                      {spec ? spec.name : g.gu_id.slice(0, 8)}
                    </span>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-[10px] text-zinc-500 text-center italic py-1">Chưa trang bị Cổ trùng</div>
          )}
        </div>

        {/* Cultivation / Blessed Land Controls */}
        {!isImmortal ? (
          <div className="w-full mb-4">
            <WallControls />
          </div>
        ) : (
          <div className="w-full mb-4">
            <BlessedLandControls />
          </div>
        )}

        {/* Main Quests Panel */}
        <div className="w-full mb-4 border-t border-zinc-900 pt-4">
          <MainQuestsPanel />
        </div>

        {/* Regen details */}
        <div className="text-[10px] text-zinc-500 text-center space-y-1.5 w-full bg-black/25 border border-zinc-900 p-2.5 rounded-xl">
          <div className="font-semibold flex items-center justify-center gap-1">
            <Compass className="w-3 h-3 text-zinc-400 animate-spin" style={{ animationDuration: "12s" }} />
            <span>{regenLabel}: <span className="font-bold text-zinc-300">{regenRate}</span> {energyType.toLowerCase()}/giờ</span>
          </div>
          <div className="text-[9px] text-zinc-650 leading-relaxed max-w-sm mx-auto">
            {isImmortal
              ? "Cổ Tiên hấp thụ Tiên Nguyên Thạch, dung tích tùy thuộc vào quy mô và đẳng cấp của Tiên Khiếu."
              : "Cổ sư hấp thụ Chân Nguyên trong không khiếu, tự động tích lũy và phục hồi theo hạn mức tư chất."}
          </div>
        </div>
      </div>
    </>
  );
};

function WallControls() {
  const store = useGameStore();
  const player = store.player;
  const [manaSpend, setManaSpend] = useState<number>(10);
  const [rechargeStones, setRechargeStones] = useState<number>(10);

  if (!player) return <div className="text-xs text-zinc-500 text-center py-2">Không có thông tin người chơi</div>;

  if (player.rank === 5 && player.sub_rank === 4) {
    return (
      <div className="w-full space-y-3.5">
        <div className="w-full flex flex-col items-center justify-center p-4 border border-rose-950 bg-rose-950/20 rounded-2xl shadow-[inset_0_0_12px_rgba(244,63,94,0.1)]">
          <div className="text-rose-450 font-bold mb-1 text-center text-sm tracking-wide uppercase flex items-center gap-1.5">
            <ShieldAlert className="w-4 h-4 text-rose-500 animate-bounce" />
            Cơ Hội Thăng Tiên
          </div>
          <div className="text-[10.5px] text-zinc-400 mb-3 text-center leading-relaxed">
            Ngươi đã đạt tới Ngũ Chuyển Đỉnh Phong. Nhân khí tích lũy: <span className="font-bold text-amber-400 font-mono">{player.human_qi?.toFixed(1) || 0}/1000</span> <br/>
            Thăng Tiên là quá trình độ kiếp cực kỳ nguy hiểm, nếu thất bại tu vi sẽ vỡ nát!
          </div>
          <button 
            className="px-5 py-2.5 bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 text-white font-bold text-xs rounded-xl shadow-[0_0_15px_rgba(244,3,94,0.45)] cursor-pointer transition-all active:scale-95 uppercase tracking-wider"
            onClick={async () => {
               if(confirm("Thăng tiên là cửu tử nhất sinh! Ngươi đã chuẩn bị kỹ?")) {
                 const res = await store.ascension();
                 if(res) alert(res.message);
               }
            }}
          >
            Liều Mạng Thăng Tiên
          </button>
        </div>

        {/* Stones Recharge Section */}
        <div className="bg-zinc-950/50 border border-zinc-900 rounded-xl p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[10px] text-zinc-450 font-bold uppercase tracking-wider flex items-center gap-1.5">
              <Coins className="w-3.5 h-3.5 text-cyan-400" />
              Nạp Nguyên Thạch (Recharge)
            </span>
            <span className="text-[10px] text-cyan-400 font-mono font-bold">
              Khả dụng: {player.primeval_stones.toLocaleString()}
            </span>
          </div>
          <div className="flex gap-2">
            <div className="relative flex-grow flex items-center bg-zinc-950 border border-zinc-900 hover:border-zinc-800 focus-within:border-cyan-500/50 rounded-xl px-3 transition-colors">
              <input 
                type="number" 
                className="w-full py-2 bg-transparent text-sm text-zinc-100 font-semibold focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" 
                value={rechargeStones} 
                onChange={(e) => setRechargeStones(Math.max(0, Number(e.target.value)))} 
                min={0} 
                max={player.primeval_stones} 
              />
              <span className="text-[10px] text-zinc-500 font-mono font-semibold ml-2">Nguyên Thạch</span>
            </div>
            <div className="flex gap-1.5">
              <button 
                type="button"
                onClick={() => setRechargeStones(player.primeval_stones)}
                className="px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-zinc-850 rounded-xl text-[10px] text-zinc-300 font-bold transition-all active:scale-95 cursor-pointer font-mono"
              >
                MAX
              </button>
              <button 
                type="button"
                className="px-4 py-2 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 text-white font-bold rounded-xl text-xs shadow-lg shadow-cyan-950/20 active:scale-95 transition-all cursor-pointer shrink-0 uppercase tracking-wide" 
                onClick={async () => {
                  if (rechargeStones <= 0) return alert('Nhập số nguyên thạch để nạp');
                  const res = await store.rechargeWithStones(rechargeStones);
                  if (res) alert(res.message);
                }}
              >
                Nạp Chân Nguyên
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full space-y-3.5">
      {/* Mana Shock Section */}
      <div className="bg-zinc-950/50 border border-zinc-900 rounded-xl p-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-zinc-450 font-bold uppercase tracking-wider flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5 text-emerald-400" />
            Trùng Kích Vách Khiếu (Mana Shock)
          </span>
          <span className="text-[10px] text-emerald-400 font-mono font-bold">
            Khả dụng: {Math.floor(player.current_mana)}
          </span>
        </div>
        <div className="flex gap-2">
          <div className="relative flex-grow flex items-center bg-zinc-950 border border-zinc-900 hover:border-zinc-800 focus-within:border-emerald-500/50 rounded-xl px-3 transition-colors">
            <input 
              type="number" 
              className="w-full py-2 bg-transparent text-sm text-zinc-100 font-semibold focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" 
              value={manaSpend} 
              onChange={(e) => setManaSpend(Math.max(0, Number(e.target.value)))} 
              min={0} 
              max={Math.floor(player.current_mana)} 
            />
            <span className="text-[10px] text-zinc-500 font-mono font-semibold ml-2">Mana</span>
          </div>
          <div className="flex gap-1.5">
            <button 
              type="button"
              onClick={() => setManaSpend(Math.floor(player.current_mana))}
              className="px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-zinc-850 rounded-xl text-[10px] text-zinc-300 font-bold transition-all active:scale-95 cursor-pointer font-mono"
            >
              MAX
            </button>
            <button 
              type="button"
              className="px-4 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold rounded-xl text-xs shadow-lg shadow-emerald-950/20 active:scale-95 transition-all cursor-pointer shrink-0 uppercase tracking-wide" 
              onClick={async () => {
                if (manaSpend <= 0) return alert('Nhập lượng mana cần dùng');
                const res = await store.breakWall(manaSpend);
                if (res) alert(res.message);
              }}
            >
              Trùng Kích
            </button>
          </div>
        </div>
      </div>

      {/* Stones Recharge Section */}
      <div className="bg-zinc-950/50 border border-zinc-900 rounded-xl p-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-zinc-450 font-bold uppercase tracking-wider flex items-center gap-1.5">
            <Coins className="w-3.5 h-3.5 text-cyan-400" />
            Nạp Nguyên Thạch (Recharge)
          </span>
          <span className="text-[10px] text-cyan-400 font-mono font-bold">
            Khả dụng: {player.primeval_stones.toLocaleString()}
          </span>
        </div>
        <div className="flex gap-2">
          <div className="relative flex-grow flex items-center bg-zinc-950 border border-zinc-900 hover:border-zinc-800 focus-within:border-cyan-500/50 rounded-xl px-3 transition-colors">
            <input 
              type="number" 
              className="w-full py-2 bg-transparent text-sm text-zinc-100 font-semibold focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none" 
              value={rechargeStones} 
              onChange={(e) => setRechargeStones(Math.max(0, Number(e.target.value)))} 
              min={0} 
              max={player.primeval_stones} 
            />
            <span className="text-[10px] text-zinc-500 font-mono font-semibold ml-2">Nguyên Thạch</span>
          </div>
          <div className="flex gap-1.5">
            <button 
              type="button"
              onClick={() => setRechargeStones(player.primeval_stones)}
              className="px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-zinc-850 rounded-xl text-[10px] text-zinc-300 font-bold transition-all active:scale-95 cursor-pointer font-mono"
            >
              MAX
            </button>
            <button 
              type="button"
              className="px-4 py-2 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 text-white font-bold rounded-xl text-xs shadow-lg shadow-cyan-950/20 active:scale-95 transition-all cursor-pointer shrink-0 uppercase tracking-wide" 
              onClick={async () => {
                if (rechargeStones <= 0) return alert('Nhập số nguyên thạch để nạp');
                const res = await store.rechargeWithStones(rechargeStones);
                if (res) alert(res.message);
              }}
            >
              Nạp Chân Nguyên
            </button>
          </div>
        </div>
      </div>

      <div className="text-[10px] text-zinc-500 text-center font-mono font-semibold">
        Tiến độ phá vách khiếu: <span className="text-emerald-400 font-bold">{((player.wall_progress ?? 0) * 100).toFixed(1)}%</span>
      </div>

      {/* Aperture Stress Level */}
      <div className="mt-3.5 p-3 bg-zinc-950/70 border border-zinc-900 rounded-xl">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5">
            <Activity className="w-3.5 h-3.5 text-rose-450 animate-pulse" />
            Áp Lực Không Khiếu (Aperture Stress)
          </span>
          <span className={`text-xs font-mono font-bold ${
            (player.aperture_stress ?? 0) >= 80 ? "text-rose-500 animate-pulse font-extrabold" :
            (player.aperture_stress ?? 0) >= 50 ? "text-amber-500" :
            "text-emerald-400"
          }`}>
            {(player.aperture_stress ?? 0).toFixed(1)}%
          </span>
        </div>

        {/* Dial / Progress Bar */}
        <div className="relative w-full h-3.5 bg-zinc-900/80 rounded-full overflow-hidden border border-zinc-850 p-[2px]">
          <div
            className={`h-full rounded-full transition-all duration-300 ${
              (player.aperture_stress ?? 0) >= 80 ? "bg-gradient-to-r from-red-600 via-rose-500 to-red-500 animate-pulse shadow-[0_0_8px_rgba(239,68,68,0.6)]" :
              (player.aperture_stress ?? 0) >= 50 ? "bg-gradient-to-r from-amber-600 to-amber-400" :
              "bg-gradient-to-r from-emerald-600 to-emerald-400"
            }`}
            style={{ width: `${Math.min(100, player.aperture_stress ?? 0)}%` }}
          />
        </div>

        {(player.aperture_stress ?? 0) >= 80 && (
          <div className="text-[9.5px] text-rose-400 mt-1.5 leading-tight font-semibold flex items-center gap-1">
            <ShieldAlert className="w-3 h-3 text-rose-500 shrink-0" />
            <span>Không Khiếu quá tải! Tỷ lệ trùng kích giảm cực mạnh. Quá 100% sẽ bị nứt vỡ rớt tu vi!</span>
          </div>
        )}

        <div className="flex items-center justify-between mt-3 gap-2">
          <span className="text-[9px] text-zinc-500 font-mono italic">
            Tự giảm: -10%/giờ thực tế
          </span>
          <button
            type="button"
            className="px-3 py-1.5 bg-gradient-to-r from-purple-700 to-violet-600 hover:from-purple-600 hover:to-violet-500 text-white font-bold rounded-lg text-[10px] shadow-md shadow-purple-950/20 active:scale-95 transition-all cursor-pointer uppercase tracking-wider flex items-center gap-1"
            onClick={async () => {
              if (player.primeval_stones < 100) {
                return alert("Không đủ Nguyên Thạch (Cần 100 Nguyên Thạch)");
              }
              if (confirm("Dùng 100 Nguyên Thạch để tịnh hóa Không khiếu, giảm 50% áp lực?")) {
                const res = await store.clearStress();
                if (res) alert(res.message);
              }
            }}
          >
            <Sparkles className="w-3 h-3" />
            Tịnh Hóa (-50% Stress) [100 💸]
          </button>
        </div>
      </div>
    </div>
  );
}

function BlessedLandControls() {
  const store = useGameStore();
  const player = store.player;

  if (!player) return null;

  const nextTrib = player.next_tribulation_time ? new Date(player.next_tribulation_time) : null;
  const now = new Date();
  const isTimeForTribulation = nextTrib && now >= nextTrib;

  return (
    <div className="w-full">
      <div className="p-4 border border-zinc-900 bg-zinc-950/30 rounded-2xl">
        <div className="text-center mb-3">
          <div className="text-[10px] font-bold text-zinc-450 uppercase tracking-wider mb-1">Thời Gian Tới Thiên Kiếp</div>
          {nextTrib ? (
             <div className={`text-base font-mono font-bold ${isTimeForTribulation ? "text-rose-500 animate-pulse font-extrabold" : "text-cyan-400"}`}>
               {isTimeForTribulation ? "★ THIÊN KIẾP GIÁNG LÂM! ★" : nextTrib.toLocaleString()}
             </div>
          ) : (
             <div className="text-xs text-zinc-650 italic">Chưa xác định chu kỳ thiên kiếp</div>
          )}
        </div>
        
        {isTimeForTribulation && (
          <div className="flex justify-center mt-3">
             <button 
               className="px-5 py-2 bg-gradient-to-r from-rose-750 to-red-650 hover:from-rose-650 hover:to-red-550 text-white text-xs font-bold rounded-xl shadow-lg shadow-rose-950/30 active:scale-95 transition-all cursor-pointer uppercase tracking-wider animate-bounce"
               onClick={async () => {
                 const res = await store.triggerTribulation();
                 if(res) alert(res.message);
               }}
             >
               Nghênh Đón Thiên Kiếp
             </button>
          </div>
        )}

        <div className="mt-4 border-t border-zinc-900 pt-3">
          <div className="text-[9px] text-zinc-550 text-center mb-2 uppercase tracking-widest font-bold font-mono">Đạo Ngân Tích Lũy</div>
          <div className="flex flex-wrap gap-2 justify-center">
            {Object.entries(player.dao_marks || {}).map(([type, count]) => (
               <div key={type} className="px-2.5 py-1 bg-zinc-950/80 border border-zinc-900 rounded-xl text-[10px] font-mono">
                 <span className="text-amber-500 font-bold uppercase">{type}</span>: <span className="text-zinc-200 font-bold">{count}</span>
               </div>
            ))}
            {(!player.dao_marks || Object.keys(player.dao_marks).length === 0) && (
              <div className="text-[10px] text-zinc-600 italic">Tiên Khiếu chưa ngưng tụ Đạo Ngân</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function MainQuestsPanel() {
  const store = useGameStore();
  const player = store.player;
  if (!player) return null;

  const completed = player.completed_quests || [];

  const quests = [
    {
      id: "quest_1_awaken",
      title: "Giác Tỉnh Không Khiếu",
      desc: "Khai mở không khiếu thành công",
      req: "Cảnh giới Nhất Chuyển Sơ Giai trở lên",
      check: () => player.rank >= 1,
    },
    {
      id: "quest_2_vital_gu",
      title: "Cổ Trùng Bản Mệnh",
      desc: "Luyện hóa hoặc trang bị Cổ trùng",
      req: "Đang trang bị hoặc có Cổ trùng bản mệnh",
      check: () => player.gu_inventory?.some(g => g.is_vital || g.equipped),
    },
    {
      id: "quest_3_rank_2",
      title: "Trùng Kích Nhị Chuyển",
      desc: "Đột phá vách khiếu thăng Nhị Chuyển",
      req: "Tu vi đạt Nhị Chuyển",
      check: () => player.rank >= 2,
    },
    {
      id: "quest_4_refine_gu",
      title: "Luyện Chế Cổ Trùng Nhị Chuyển",
      desc: "Luyện thành công Cổ trùng bậc 2 trở lên",
      req: "Có Cổ trùng bậc 2 trở lên",
      check: () => {
        return player.gu_inventory?.some(g => {
          const spec = store.guDictionary?.[g.gu_id];
          return spec && spec.rank >= 2;
        });
      },
    },
    {
      id: "quest_5_fame",
      title: "Danh Chấn Thanh Bản Thôn",
      desc: "Chiến thắng quái thú dọn sạch 5 địa khu",
      req: `Đã quét dọn ${Object.keys(player.cleared_cells || {}).length}/5 ô`,
      check: () => Object.keys(player.cleared_cells || {}).length >= 5,
    },
    {
      id: "quest_6_rank_5",
      title: "Đỉnh Phong Chuẩn Bị",
      desc: "Tu vi đỉnh phong, sẵn sàng độ kiếp thăng tiên",
      req: "Tu vi đạt Ngũ Chuyển",
      check: () => player.rank >= 5,
    },
  ];

  const getQuestIcon = (id: string) => {
    switch (id) {
      case "quest_1_awaken": return <Sparkles className="w-3.5 h-3.5 text-emerald-400" />;
      case "quest_2_vital_gu": return <Flame className="w-3.5 h-3.5 text-amber-500 animate-pulse" />;
      case "quest_3_rank_2": return <Zap className="w-3.5 h-3.5 text-cyan-400" />;
      case "quest_4_refine_gu": return <Atom className="w-3.5 h-3.5 text-purple-450" />;
      case "quest_5_fame": return <Crown className="w-3.5 h-3.5 text-yellow-500 animate-bounce" />;
      case "quest_6_rank_5": return <ShieldAlert className="w-3.5 h-3.5 text-rose-500" />;
      default: return <Award className="w-3.5 h-3.5 text-zinc-400" />;
    }
  };

  return (
    <div className="w-full text-center">
      <div className="text-[10px] text-zinc-500 uppercase tracking-widest text-center mb-3 font-mono font-bold flex items-center justify-center gap-1.5">
        <Award className="w-4 h-4 text-amber-500 animate-pulse" />
        Nhiệm Vụ Chính Tuyến
      </div>
      <div className="space-y-2 max-h-60 overflow-y-auto pr-1">
        {quests.map((q) => {
          const isDone = completed.includes(q.id);
          const canClaim = !isDone && q.check();
          
          return (
            <div 
              key={q.id} 
              className={`p-2.5 rounded-xl border text-xs flex justify-between items-center transition-all duration-300 hover:scale-[1.01] ${
                isDone ? "bg-emerald-950/10 border-emerald-950/20 opacity-70" :
                canClaim ? "bg-amber-950/15 border-amber-500/40 shadow-[0_0_10px_rgba(245,158,11,0.08)] animate-pulse" :
                "bg-zinc-950/40 border-zinc-900"
              }`}
            >
              <div className="text-left flex-1 min-w-0 mr-3">
                <div className={`font-semibold flex items-center gap-2 ${isDone ? "text-emerald-450 line-through" : "text-zinc-200"}`}>
                  {getQuestIcon(q.id)}
                  <span className="truncate">{q.title}</span>
                </div>
                <div className="text-[10px] text-zinc-500 mt-0.5 truncate">{q.desc}</div>
                <div className="text-[9px] text-zinc-400 mt-0.5 italic font-mono">{q.req}</div>
              </div>
              
              <button
                disabled={isDone || !canClaim || store.actionLoading}
                onClick={async () => {
                  const res = await store.completeQuest(q.id);
                  if (res && res.success) {
                    alert(res.message);
                  }
                }}
                className={`px-3 py-1.5 rounded-xl text-[10px] font-bold font-mono transition-all shrink-0 cursor-pointer ${
                  isDone 
                    ? "bg-zinc-900/60 text-zinc-650 border border-zinc-950 cursor-not-allowed" 
                    : canClaim 
                    ? "bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 shadow-[0_0_10px_rgba(245,158,11,0.3)] active:scale-95" 
                    : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                }`}
              >
                {isDone ? "ĐÃ NHẬN" : canClaim ? "NHẬN +50" : "CHƯA ĐẠT"}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
