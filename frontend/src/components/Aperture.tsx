"use client";

import React from "react";
import { GuInstance, GuSpec } from "@/store/useGameStore";

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
  1: { from: "#2d6a4f", to: "#1b4332", glow: "rgba(45,106,79,0.6)",   ring: "#2d6a4f", text: "#6ee7b7", waveSolid: "#34d399", waveAlpha: "rgba(52,211,153,0.5)"  },
  2: { from: "#7f1d1d", to: "#450a0a", glow: "rgba(185,28,28,0.6)",   ring: "#b91c1c", text: "#fca5a5", waveSolid: "#ef4444", waveAlpha: "rgba(239,68,68,0.5)"   },
  3: { from: "#1e3a5f", to: "#0f172a", glow: "rgba(37,99,235,0.6)",   ring: "#2563eb", text: "#93c5fd", waveSolid: "#60a5fa", waveAlpha: "rgba(96,165,250,0.5)"  },
  4: { from: "#713f12", to: "#3b1f00", glow: "rgba(202,138,4,0.6)",   ring: "#ca8a04", text: "#fde68a", waveSolid: "#facc15", waveAlpha: "rgba(250,204,21,0.5)"  },
  5: { from: "#4c1d95", to: "#2e1065", glow: "rgba(124,58,237,0.65)", ring: "#7c3aed", text: "#c4b5fd", waveSolid: "#a78bfa", waveAlpha: "rgba(167,139,250,0.5)" },
  6: { from: "#164e63", to: "#083344", glow: "rgba(8,145,178,0.65)",  ring: "#0891b2", text: "#67e8f9", waveSolid: "#22d3ee", waveAlpha: "rgba(34,211,238,0.5)"  },
  7: { from: "#831843", to: "#500724", glow: "rgba(219,39,119,0.65)", ring: "#db2777", text: "#f9a8d4", waveSolid: "#f472b6", waveAlpha: "rgba(244,114,182,0.5)" },
  8: { from: "#1c1917", to: "#0c0a09", glow: "rgba(168,162,158,0.65)",ring: "#a8a29e", text: "#e7e5e4", waveSolid: "#d6d3d1", waveAlpha: "rgba(214,211,209,0.5)"},
  9: { from: "#78350f", to: "#1c0a00", glow: "rgba(251,191,36,0.8)",  ring: "#fbbf24", text: "#fef08a", waveSolid: "#fde047", waveAlpha: "rgba(253,224,71,0.6)"  },
};

// SVG layout constants — larger viewBox so orbit dots aren't clipped
const VB   = 260;          // viewBox size
const CX   = 130;          // center x
const CY   = 130;          // center y
const R    = 88;           // orb radius
const CIRC = 2 * Math.PI * R;
const ORBIT_R = 118;       // orbit radius for Gu dots (fits inside 260px viewBox with room)

// Build a sinusoidal wave path that spans 3× the orb width so it can scroll
// The wave sits at y=0 relative to the fill level; we translate it up/down via the group.
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
  // Close downward to fill
  d += ` L ${startX + totalWidth} ${VB} L ${startX} ${VB} Z`;
  return d;
}

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
  const percent   = Math.min(100, Math.max(0, (currentMana / maxMana) * 100));
  const rankName  = RANK_NAMES[rank]    ?? `Rank ${rank}`;
  const subRankName = SUB_RANK_NAMES[subRank] ?? `Sub ${subRank}`;
  const manaQuality = MANA_QUALITY[rank] ?? "Chân Nguyên";
  const c = RANK_COLORS[rank] ?? RANK_COLORS[1];

  const dashOffset = CIRC * (1 - percent / 100);

  const regenRate =
    aptitudeType === "Giáp"  ? (2   * rank).toFixed(1)
    : aptitudeType === "Ất"  ? (1.5 * rank).toFixed(1)
    : aptitudeType === "Bính"? (1   * rank).toFixed(1)
    : (0.5 * rank).toFixed(1);

  // Vital Gu
  const vitalGu   = guInventory.find((g) => g.is_vital);
  const vitalSpec = vitalGu ? guDictionary[vitalGu.gu_id] : null;

  // Equipped Gu (max 6)
  const equippedGus = guInventory.filter((g) => g.equipped && g.status !== "DEAD").slice(0, 6);

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

  // Fill level: the top of the liquid inside the orb
  // CY + R is the bottom of the orb; we fill upward by percent
  const fillTop = CY + R - (percent / 100) * (R * 2);

  // Wave paths (static shapes; CSS animation scrolls them horizontally)
  const wave1 = buildWavePath(5, 60, 0);
  const wave2 = buildWavePath(4, 45, 20);

  const gradId   = `orbGrad-${rank}`;
  const clipId   = `orbClip-${rank}`;
  const animId1  = `aperture-wave1-${rank}`;
  const animId2  = `aperture-wave2-${rank}`;

  return (
    <>
      {/* Inline keyframes — scoped by animId so multiple ranks don't clash */}
      <style>{`
        @keyframes ${animId1} {
          from { transform: translateX(0); }
          to   { transform: translateX(${VB}px); }
        }
        @keyframes ${animId2} {
          from { transform: translateX(0); }
          to   { transform: translateX(-${VB}px); }
        }
        .wave-group-1-${rank} { animation: ${animId1} 4s linear infinite; }
        .wave-group-2-${rank} { animation: ${animId2} 6s linear infinite; }
      `}</style>

      <div className="flex flex-col items-center justify-center p-4 sm:p-5 glass-panel-gold rounded-3xl w-full">
        {/* Header */}
        <h3
          className="text-sm sm:text-base font-semibold tracking-wide uppercase font-serif mb-0.5"
          style={{ color: c.text }}
        >
          Không Khiếu (Aperture)
        </h3>
        <p className="text-[9px] sm:text-[10px] text-zinc-500 italic text-center mb-1">
          Tư Chất: {aptitudeType} ({aptitudePercent}%) • {manaQuality}
        </p>
        <p className="text-[10px] sm:text-xs font-semibold tracking-wide text-center mb-3" style={{ color: c.text }}>
          Tu Vi: {rankName}&nbsp;&nbsp;|&nbsp;&nbsp;Cảnh Giới: {subRankName}
        </p>

        {/* Orb SVG — fluid width, square aspect ratio */}
        <div className="relative w-full max-w-[260px] mx-auto mb-3" style={{ aspectRatio: "1 / 1" }}>
          <svg
            viewBox={`0 0 ${VB} ${VB}`}
            width="100%"
            height="100%"
            className="absolute inset-0 overflow-visible"
            style={{ filter: `drop-shadow(0 0 20px ${c.glow})` }}
          >
            <defs>
              <radialGradient id={gradId} cx="50%" cy="35%" r="65%">
                <stop offset="0%" stopColor={c.from} />
                <stop offset="100%" stopColor={c.to} />
              </radialGradient>
              <clipPath id={clipId}>
                <circle cx={CX} cy={CY} r={R} />
              </clipPath>
            </defs>

            {/* Dark orb background */}
            <circle cx={CX} cy={CY} r={R} fill={c.to} opacity="0.95" />

            {/* Animated liquid fill — clipped to orb */}
            <g clipPath={`url(#${clipId})`}>
              {/* Solid fill below the wave */}
              <rect
                x={CX - R} y={fillTop}
                width={R * 2} height={R * 2 + 10}
                fill={c.waveSolid}
                opacity="0.35"
              />

              {/* Wave 1 — scrolls right */}
              <g
                className={`wave-group-1-${rank}`}
                style={{ transformOrigin: `${CX}px ${fillTop}px` }}
              >
                <path
                  d={wave1}
                  fill={c.waveSolid}
                  opacity="0.55"
                  transform={`translate(0, ${fillTop})`}
                />
              </g>

              {/* Wave 2 — scrolls left, slightly transparent */}
              <g
                className={`wave-group-2-${rank}`}
                style={{ transformOrigin: `${CX}px ${fillTop}px` }}
              >
                <path
                  d={wave2}
                  fill={c.waveAlpha}
                  opacity="0.45"
                  transform={`translate(0, ${fillTop})`}
                />
              </g>

              {/* Subtle top shimmer line */}
              <rect
                x={CX - R} y={fillTop - 1}
                width={R * 2} height={3}
                fill="rgba(255,255,255,0.18)"
              />
            </g>

            {/* Track ring (dim) */}
            <circle
              cx={CX} cy={CY} r={R}
              fill="none"
              stroke={c.ring}
              strokeWidth="4"
              opacity="0.18"
            />
            {/* Progress arc */}
            <circle
              cx={CX} cy={CY} r={R}
              fill="none"
              stroke={c.ring}
              strokeWidth="4"
              strokeLinecap="round"
              strokeDasharray={CIRC}
              strokeDashoffset={dashOffset}
              transform={`rotate(-90 ${CX} ${CY})`}
              opacity="0.9"
            />

            {/* Orbit dots for equipped Gu */}
            {orbitDots.map(({ g, spec, x, y }, i) => (
              <g key={g.instance_id}>
                {/* Connector line from orb edge to dot */}
                <line
                  x1={CX + (R + 2) * Math.cos((i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2)}
                  y1={CY + (R + 2) * Math.sin((i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2)}
                  x2={x} y2={y}
                  stroke={c.ring}
                  strokeWidth="0.8"
                  opacity="0.3"
                />
                {/* Outer glow ring */}
                <circle cx={x} cy={y} r={13} fill={c.glow} opacity="0.25" />
                {/* Dot background */}
                <circle cx={x} cy={y} r={10} fill={c.to} stroke={c.ring} strokeWidth="1.5" opacity="0.95" />
                {/* Inner fill dot */}
                <circle cx={x} cy={y} r={4.5} fill={c.waveSolid} opacity="0.9" />
                {/* Label */}
                <text
                  x={x} y={y + 22}
                  textAnchor="middle"
                  fontSize="7.5"
                  fill={c.text}
                  opacity="0.85"
                  fontFamily="monospace"
                >
                  {spec ? spec.name.slice(0, 4) : `G${i + 1}`}
                </text>
              </g>
            ))}
          </svg>

          {/* Center text — absolutely positioned over SVG */}
          <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none z-10">
            <span
              className="text-3xl font-bold drop-shadow-[0_2px_8px_rgba(0,0,0,0.95)]"
              style={{ color: c.text }}
            >
              {percent.toFixed(1)}%
            </span>
            <span
              className="text-xs font-semibold mt-0.5 drop-shadow-[0_1px_4px_rgba(0,0,0,0.9)]"
              style={{ color: c.text }}
            >
              {currentMana.toFixed(1)} / {maxMana.toFixed(0)}
            </span>
            <span className="text-[9px] mt-1 text-zinc-400 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">
              Chân Nguyên Hải
            </span>
          </div>
        </div>

        {/* Vital Gu badge */}
        <div className="w-full mb-3">
          <div className="text-[9px] text-zinc-500 uppercase tracking-widest text-center mb-1">
            Bản Mệnh Cổ
          </div>
          {vitalGu ? (
            <div
              className="flex items-center gap-2 px-3 py-1.5 rounded-xl border mx-auto w-fit"
              style={{ borderColor: c.ring + "55", background: c.to + "88" }}
            >
              <span
                className="w-2.5 h-2.5 rounded-full flex-shrink-0 animate-pulse"
                style={{ background: c.waveSolid, boxShadow: `0 0 6px ${c.glow}` }}
              />
              <span className="text-xs font-semibold" style={{ color: c.text }}>
                {vitalSpec ? vitalSpec.name : vitalGu.gu_id}
              </span>
              <span className="text-[9px] text-zinc-500 ml-1">
                {vitalGu.status === "ACTIVE" ? "Hoạt Động"
                  : vitalGu.status === "SLEEPING" ? "Ngủ"
                  : "Tử Vong"}
              </span>
            </div>
          ) : (
            <div className="text-[10px] text-zinc-600 text-center italic">Chưa thiết lập</div>
          )}
        </div>

        {/* Equipped Gu list */}
        <div className="w-full mb-3">
          <div className="text-[9px] text-zinc-500 uppercase tracking-widest text-center mb-1.5">
            Cổ Đang Trang Bị ({equippedGus.length})
          </div>
          {equippedGus.length > 0 ? (
            <div className="flex flex-wrap gap-1.5 justify-center">
              {equippedGus.map((g) => {
                const spec = guDictionary[g.gu_id];
                return (
                  <div
                    key={g.instance_id}
                    className="flex items-center gap-1 px-2 py-0.5 rounded-lg border text-[10px]"
                    style={{ borderColor: c.ring + "44", background: c.to + "66" }}
                    title={spec?.description ?? g.gu_id}
                  >
                    <span
                      className="w-2 h-2 rounded-full flex-shrink-0"
                      style={{ background: c.waveSolid }}
                    />
                    <span style={{ color: c.text }}>
                      {spec ? spec.name : g.gu_id.slice(0, 8)}
                    </span>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-[10px] text-zinc-600 text-center italic">Chưa trang bị Cổ nào</div>
          )}
        </div>

        {/* Regen */}
        <div className="text-[10px] text-zinc-500 text-center">
          Tái sinh: {regenRate} chân nguyên/giờ
        </div>
      </div>
    </>
  );
};
