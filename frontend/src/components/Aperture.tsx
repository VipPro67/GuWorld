"use client";
import React, { useEffect, useState } from "react";
import { defaultQuests, formatRewards } from "@/data/quests";
import { estimateManaFromSnapshot, estimateHPFromSnapshot, GuInstance, GuSpec, Player, useGameStore } from "@/store/useGameStore";
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
  Clock,
  Eye,
  EyeOff,
  User,
  Camera,
  Swords,
} from "lucide-react";

interface ApertureProps {
  player: Player;
  guDictionary: Record<string, GuSpec>;
}

const RANK_NAMES: Record<number, string> = {
  0: "Phàm Nhân",
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
  1: { from: "#1b4332", to: "#081c15", glow: "rgba(52,211,153,0.35)", ring: "#2d6a4f", text: "#6ee7b7", waveSolid: "#10b981", waveAlpha: "rgba(16,185,129,0.4)" },
  2: { from: "#450a0a", to: "#1a0505", glow: "rgba(239,68,68,0.35)", ring: "#b91c1c", text: "#fca5a5", waveSolid: "#ef4444", waveAlpha: "rgba(239,68,68,0.4)" },
  3: { from: "#0f172a", to: "#020617", glow: "rgba(59,130,246,0.35)", ring: "#2563eb", text: "#93c5fd", waveSolid: "#3b82f6", waveAlpha: "rgba(59,130,246,0.4)" },
  4: { from: "#3b1f00", to: "#1c0f00", glow: "rgba(245,158,11,0.35)", ring: "#d97706", text: "#fde68a", waveSolid: "#f59e0b", waveAlpha: "rgba(245,158,11,0.4)" },
  5: { from: "#2e1065", to: "#12002f", glow: "rgba(139,92,246,0.4)", ring: "#7c3aed", text: "#c4b5fd", waveSolid: "#8b5cf6", waveAlpha: "rgba(139,92,246,0.4)" },
  6: { from: "#083344", to: "#02151e", glow: "rgba(6,182,212,0.4)", ring: "#0891b2", text: "#67e8f9", waveSolid: "#06b6d4", waveAlpha: "rgba(6,182,212,0.4)" },
  7: { from: "#500724", to: "#25000f", glow: "rgba(236,72,153,0.4)", ring: "#db2777", text: "#f9a8d4", waveSolid: "#ec4899", waveAlpha: "rgba(236,72,153,0.4)" },
  8: { from: "#1c1917", to: "#0c0a09", glow: "rgba(168,162,158,0.4)", ring: "#a8a29e", text: "#e7e5e4", waveSolid: "#a8a29e", waveAlpha: "rgba(168,162,158,0.4)" },
  9: { from: "#78350f", to: "#2d0f00", glow: "rgba(251,191,36,0.55)", ring: "#fbbf24", text: "#fef08a", waveSolid: "#fbbf24", waveAlpha: "rgba(251,191,36,0.5)" },
};

// SVG layout constants — expanded to 300 viewBox to prevent text clipping
const VB = 300;
const CX = 150;
const CY = 150;
const R = 80;
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
  player,
  guDictionary,
}) => {
  const store = useGameStore();
  const manaSnapshot = store.manaSnapshot;
  const [manaNow, setManaNow] = useState(() => Date.now());

  useEffect(() => {
    if (!player || !manaSnapshot) return;

    const isManaFull = player.rank >= 6 ? false : (manaSnapshot.current_mana >= manaSnapshot.max_mana);
    const isHPFull = manaSnapshot.hp >= manaSnapshot.base_hp;

    if (isManaFull && isHPFull) return;

    const interval = setInterval(() => {
      setManaNow(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [manaSnapshot, player.id]);

  const currentMana = estimateManaFromSnapshot(
    manaSnapshot?.player_id === player.id ? manaSnapshot : null,
    manaNow,
  ) ?? player.current_mana;
  const maxMana = player.max_mana;
  const aptitudeType = player.aptitude_type;
  const aptitudePercent = player.aptitude_percent;
  const rank = player.rank;
  const subRank = player.sub_rank;
  const guInventory = player.gu_inventory || [];
  const eligibleGusForCombat = guInventory.filter(g => g.status === "ACTIVE" && g.equipped);
  const killerMoves = store.killerMoves || [];
  const percent = Math.min(100, Math.max(0, (currentMana / maxMana) * 100));
  const rankName = RANK_NAMES[rank] ?? `Rank ${rank}`;
  const subRankName = SUB_RANK_NAMES[subRank] ?? `Sub ${subRank}`;
  const isImmortal = rank >= 6;
  const energyType = isImmortal ? "Tiên Nguyên" : "Chân Nguyên";
  const apertureName = isImmortal ? "Tiên Khiếu" : "Không Khiếu";
  const energyPoolName = isImmortal ? "Tiên Nguyên Hải" : "Chân Nguyên Hải";
  const regenLabel = isImmortal ? "Tiên Khiếu ngưng tụ" : "Tốc độ hồi phục";
  const manaQuality = MANA_QUALITY[rank] ?? energyType;
  const c = RANK_COLORS[rank] ?? RANK_COLORS[1];

  const currentHP = player
    ? (estimateHPFromSnapshot(manaSnapshot?.player_id === player.id ? manaSnapshot : null, manaNow) ?? player.hp)
    : 0;
  const maxHP = player?.base_hp ?? 0;
  const hpPercent = maxHP > 0 ? Math.min(100, Math.max(0, (currentHP / maxHP) * 100)) : 0;

  const dashOffset = CIRC * (1 - percent / 100);

  const [activeTab, setActiveTab] = useState<"aperture" | "stats" | "quests" | "soul" | "referrals" | "settings">("aperture");

  // Disguise/stealth settings state
  const [isDisguised, setIsDisguised] = useState(player.is_disguised || false);
  const [fakeRank, setFakeRank] = useState(player.fake_rank || player.rank);
  const [fakeSubRank, setFakeSubRank] = useState(player.fake_sub_rank || player.sub_rank);
  const [fakeVitalGu, setFakeVitalGu] = useState(player.fake_vital_gu || "GU_SHADOW");

  // Auto combat settings state
  const defaults = { auto_heal_hp: false, heal_threshold: 30, auto_restore_mana: false, mana_threshold: 30, gu_priorities: [], km_priorities: [] };
  const saved = player.combat_settings || store.combatSettings || defaults;
  const [csAutoHealHP, setCsAutoHealHP] = useState(saved.auto_heal_hp);
  const [csHealThreshold, setCsHealThreshold] = useState(saved.heal_threshold);
  const [csAutoRestoreMana, setCsAutoRestoreMana] = useState(saved.auto_restore_mana);
  const [csManaThreshold, setCsManaThreshold] = useState(saved.mana_threshold);
  const [csGuPriorities, setCsGuPriorities] = useState(saved.gu_priorities || []);
  const [csKMPriorities, setCsKMPriorities] = useState(saved.km_priorities || []);
  const [combatSettingsLoaded, setCombatSettingsLoaded] = useState(false);

  React.useEffect(() => {
    if (!combatSettingsLoaded && !store.combatSettingsLoading) {
      store.fetchCombatSettings().then(() => {
        setCombatSettingsLoaded(true);
      });
    }
  }, [combatSettingsLoaded]);

  React.useEffect(() => {
    const cs = store.combatSettings || saved;
    if (combatSettingsLoaded && cs) {
      setCsAutoHealHP(cs.auto_heal_hp);
      setCsHealThreshold(cs.heal_threshold);
      setCsAutoRestoreMana(cs.auto_restore_mana);
      setCsManaThreshold(cs.mana_threshold);
      setCsGuPriorities(cs.gu_priorities || []);
      setCsKMPriorities(cs.km_priorities || []);
    }
  }, [store.combatSettings, combatSettingsLoaded]);

  React.useEffect(() => {
    setIsDisguised(player.is_disguised || false);
    setFakeRank(player.fake_rank || player.rank);
    setFakeSubRank(player.fake_sub_rank || player.sub_rank);
    setFakeVitalGu(player.fake_vital_gu || "GU_SHADOW");
  }, [player.is_disguised, player.fake_rank, player.fake_sub_rank, player.fake_vital_gu, player.rank, player.sub_rank]);

  React.useEffect(() => {
    void store.fetchReferrals();
  }, []);

  const aperture = store.aperture;
  const essenceProd = aperture?.essence_production_per_year ?? 10;
  const immortalRegen = (essenceProd / 20.0) * (1 + (Math.min(Math.max(subRank, 1), 4) - 1) * 0.15);
  const mortalRegen = aptitudeType === "Giáp" ? 8 : aptitudeType === "Ất" ? 6 : aptitudeType === "Bính" ? 4 : 2;
  const regenRate = (isImmortal ? immortalRegen : mortalRegen).toFixed(2);

  // Vital Gu
  const vitalGu = guInventory.find((g) => g.is_vital);
  const vitalSpec = vitalGu ? guDictionary[vitalGu.gu_id] : null;

  // Equipped Gu (max 6)
  let equippedGus = guInventory.filter((g) => g.equipped && g.status !== "DEAD" && g.is_vital !== true);
  equippedGus = equippedGus.slice(0, 6);

  // Compute disguise permissions based on active equipped Gu bugs
  const equippedActiveGus = guInventory.filter((g) => g.equipped && g.status === "ACTIVE");
  const hasShadow = equippedActiveGus.some((g) => g.gu_id === "GU_SHADOW");
  const hasShadowSteps = equippedActiveGus.some((g) => g.gu_id === "GU_SHADOW_STEPS");
  const hasStealthScales = equippedActiveGus.some((g) => g.gu_id === "GU_STEALTH_SCALES");
  const hasChangeForm = equippedActiveGus.some((g) => g.gu_id === "GU_CHANGE_FORM");
  const hasAttitude = equippedActiveGus.some((g) => g.gu_id === "GU_ATTITUDE");

  const hasConcealment = hasShadow || hasShadowSteps || hasStealthScales || hasChangeForm || hasAttitude;
  const hasFaceChange = hasChangeForm || hasAttitude;

  // Owned Gu IDs for fake Vital Gu select dropdown
  const ownedGuIds = Array.from(new Set([
    "GU_SHADOW",
    ...guInventory.map((g) => g.gu_id)
  ]));

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

  const gradId = `orbGrad-${rank}`;
  const clipId = `orbClip-${rank}`;
  const animId1 = `aperture-wave1-${rank}`;
  const animId2 = `aperture-wave2-${rank}`;

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
        className="flex flex-col items-center justify-center p-5 glass-panel-gold rounded-3xl w-full border border-amber-500/20 transition-all duration-300 relative overflow-hidden group"
        style={{
          boxShadow: `0 8px 32px 0 rgba(0, 0, 0, 0.45), 0 0 25px ${c.glow}20`,
          borderColor: c.ring + "2a"
        }}
      >
        {/* Decorative background grid pattern */}
        <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(16,185,129,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(16,185,129,0.01)_1px,transparent_1px)] bg-[size:14px_24px] pointer-events-none" />

        {/* Character Title */}
        <h3
          className="text-base sm:text-lg font-bold tracking-wider uppercase font-serif mb-1 flex items-center gap-1.5 z-10"
          style={{ color: c.text, textShadow: `0 0 10px ${c.glow}` }}
        >
          {player.username}
          <Sparkles className="w-3.5 h-3.5 text-amber-500 animate-pulse" />
        </h3>
        <p className="text-[10px] text-zinc-400 text-center mb-1 font-mono uppercase tracking-wide z-10">
          {isImmortal ? "Tiên Phàm Cách Biệt" : `Tư Chất: ${aptitudeType} (${aptitudePercent}%)`} • {manaQuality}
        </p>
        <p className="text-xs font-semibold tracking-widest text-center mb-4 text-zinc-150 uppercase z-10">
          Tu Vi: <span className="font-bold" style={{ color: c.text }}>{rankName}</span>&nbsp;&nbsp;❖&nbsp;&nbsp;Cảnh Giới: <span className="font-bold" style={{ color: c.text }}>{subRankName}</span>
        </p>

        {/* Health Bar (Sleek Glossy Bar) */}
        <div className="relative h-6 w-full overflow-hidden rounded-xl border border-red-950/60 bg-zinc-950/60 shadow-[inset_0_2px_4px_rgba(0,0,0,0.6)] mb-5 flex items-center px-2 z-10">
          <Heart className="w-3.5 h-3.5 text-rose-500 mr-2 shrink-0 animate-pulse" />
          <div className="relative flex-grow h-3 rounded-md overflow-hidden bg-zinc-900/60">
            <div
              className="h-full bg-gradient-to-r from-red-800 via-red-600 to-rose-500 rounded-md transition-all duration-300 shadow-[0_0_8px_rgba(239,68,68,0.5)]"
              style={{ width: `${hpPercent}%` }}
            />
          </div>
          <span className="text-[10px] font-mono font-bold text-zinc-300 ml-2.5 shrink-0">
            {player
              ? `${Math.round(currentHP)}/${Math.round(maxHP)} (${hpPercent.toFixed(0)}%)`
              : "--"}
          </span>
        </div>

        {/* Tab Selection */}
        <div className="flex gap-2 w-full bg-zinc-950/60 p-1.5 rounded-2xl border border-zinc-900 mb-5 z-10">
          <button
            onClick={() => setActiveTab("aperture")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "aperture"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Không Khiếu
          </button>
          <button
            onClick={() => setActiveTab("quests")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "quests"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Nhiệm Vụ
          </button>
          <button
            onClick={() => setActiveTab("soul")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "soul"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Hồn Phách
          </button>
          <button
            onClick={() => {
              setActiveTab("referrals");
              void store.fetchReferrals();
            }}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "referrals"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Sư Đồ
          </button>
          <button
            onClick={() => setActiveTab("settings")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "settings"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Cài Đặt
          </button>
        </div>

        {/* Tab Content */}
        {activeTab === "aperture" && (
          <div className="w-full space-y-4 z-10">
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
                      {player ? Math.floor(currentMana).toLocaleString() : "--"} viên
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
                Bản Mệnh Cổ
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
                  <span className={`text-[9px] font-mono px-2 py-0.5 rounded-lg font-bold border ${vitalGu.status === "ACTIVE"
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
                <div className="text-[10px] text-zinc-550 text-center italic py-1">Không Có Bản Mệnh Cổ</div>
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
          </div>
        )}
        {activeTab === "quests" && (
          <div className="w-full z-10">
            <MainQuestsPanel />
          </div>
        )}

        {activeTab === "soul" && (
          <div className="w-full z-10 space-y-5">
            {/* Soul Foundation info */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex justify-between items-center border-b border-zinc-900 pb-2.5">
                <div className="flex items-center gap-2">
                  <Activity className="w-5 h-5 text-indigo-400" />
                  <span className="font-serif font-bold text-zinc-100 text-sm">Nội Tình Hồn Phách</span>
                </div>
                <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-indigo-950/50 text-indigo-300 border border-indigo-800/40">
                  {player.soul_tier === "NHAN_HON" ? "Nhân Hồn" :
                    player.soul_tier === "BACH_NHAN" ? "Bách Nhân Hồn" :
                      player.soul_tier === "THIEN_NHAN" ? "Thiên Nhân Hồn" :
                        player.soul_tier === "VAN_NHAN" ? "Vạn Nhân Hồn" :
                          player.soul_tier === "BACH_VAN" ? "Bách Vạn Hồn" :
                            player.soul_tier === "THIEN_VAN" ? "Thiên Vạn Hồn" : "Ức Nhân Hồn"}
                </span>
              </div>

              {/* Progress HP bar */}
              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-xs font-mono text-zinc-400">
                  <span>Sức mạnh Linh hồn</span>
                  <span>{player.soul_hp} / {player.max_soul}</span>
                </div>
                <div className="w-full bg-zinc-950 rounded-full h-3 border border-zinc-900 overflow-hidden relative shadow-inner">
                  <div
                    className="bg-gradient-to-r from-indigo-600 to-purple-500 h-full rounded-full transition-all duration-300"
                    style={{ width: `${Math.min(100, (player.soul_hp / player.max_soul) * 100)}%` }}
                  />
                </div>
              </div>

              {/* Warning if needs refinement */}
              {player.needs_soul_refine ? (
                <div className="bg-red-950/20 border border-red-800/30 rounded-xl p-3 flex flex-col gap-2">
                  <div className="flex items-start gap-2.5">
                    <ShieldAlert className="w-4 h-4 text-red-400 mt-0.5 shrink-0" />
                    <div className="text-xs text-red-200">
                      <strong className="font-semibold">Cảnh báo:</strong> Hồn phách đang bị phù nề! Đạo hữu không thể dùng Can Đảm Cổ để tráng hồn. Hãy tiến vào Cốc Lạc Phách để rèn luyện nén chặt hồn phách.
                    </div>
                  </div>
                  <button
                    onClick={() => store.refineSoul()}
                    className="w-full bg-red-900/30 hover:bg-red-900/50 border border-red-800/40 text-red-200 rounded-xl py-2 text-xs font-mono font-bold transition-all cursor-pointer"
                  >
                    Luyện Hồn (Cốc Lạc Phách) - Giảm 80% Sức Mạnh Linh Hồn
                  </button>
                </div>
              ) : (
                <div className="flex gap-3">
                  <button
                    onClick={() => store.useGutsGu()}
                    disabled={player.needs_soul_refine || (player.materials_inventory?.["guts_gu"] || 0) <= 0}
                    className="flex-1 bg-indigo-600 hover:bg-indigo-500 disabled:bg-zinc-800 disabled:opacity-40 disabled:border-zinc-800/40 border border-indigo-500/20 text-white rounded-xl py-2 text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5"
                  >
                    <Sparkles className="w-3.5 h-3.5" />
                    Tráng Hồn ({player.materials_inventory?.["guts_gu"] || 0} Cổ Can Đảm)
                  </button>
                  <button
                    onClick={() => store.calmSoul()}
                    disabled={player.soul_hp >= player.max_soul || player.primeval_stones < 50}
                    className="flex-1 bg-purple-900/30 hover:bg-purple-900/50 disabled:bg-zinc-800 disabled:opacity-40 disabled:border-zinc-800/40 border border-purple-800/40 text-purple-200 rounded-xl py-2 text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5"
                  >
                    An Hồn (50 Nguyên Thạch)
                  </button>
                </div>
              )}
            </div>

            {/* Path Attainments */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner space-y-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Crown className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cảnh Giới Lưu Phái</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                {player.path_masteries && Object.keys(player.path_masteries).length > 0 ? (
                  Object.entries(player.path_masteries).map(([path, level]) => (
                    <div key={path} className="flex flex-col gap-1.5 bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                      <div className="flex justify-between items-center">
                        <span className="text-zinc-300 font-mono text-xs font-bold capitalize">{path} Path</span>
                        <span className={`px-2 py-0.5 rounded text-[9px] font-bold font-mono ${level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                          level === "Great Grandmaster" ? "bg-red-950/50 text-red-300 border border-red-800/40" :
                            level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                              level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                                "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                          }`}>
                          {level}
                        </span>
                      </div>
                      <div className="text-[10px] text-zinc-500 leading-normal">
                        {level === "Master" && "Trực giác chiến đấu hệ, nhận +10% hiệu quả kỹ năng."}
                        {level === "Grandmaster" && "Bỏ qua hình phạt giảm 30% sức mạnh khi sử dụng Cổ trùng hệ này."}
                        {level === "Great Grandmaster" && "Cộng hưởng Thiên Địa Đạo Ngân, +50% hiệu ứng tại map cùng hệ."}
                        {level === "Supreme Grandmaster" && "Cảnh giới Vô Thượng tối cao, kiểm soát tuyệt đối đại đạo."}
                        {level === "Ordinary" && "Đạt hiểu biết phổ thông cơ bản về đạo pháp."}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="col-span-2 text-center text-xs text-zinc-650 italic py-4">Chưa ngộ chân ý bất kỳ lưu phái nào. Hãy thám hiểm Mộng Cảnh!</div>
                )}
              </div>
            </div>
          </div>
        )}

        {activeTab === "referrals" && (
          <div className="w-full z-10 space-y-5 text-left">
            {/* Master Referral Card */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Crown className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Liên Kết Sư Đồ</span>
              </div>
              <p className="text-xs text-zinc-400 leading-relaxed font-serif">
                Mời bằng hữu cùng tham gia thế giới Cổ Giới nghịch thiên cải mệnh. Khi đệ tử đạt tu vi cao hoặc nạp tiền, bạn sẽ nhận được phần thưởng Nguyên Thạch quý hiếm cùng Cổ trùng bản mệnh ngẫu nhiên!
              </p>

              {/* Referral Code & Link */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-2">
                <div className="bg-zinc-950/80 p-3.5 rounded-xl border border-zinc-800 flex flex-col gap-1">
                  <span className="text-[10px] uppercase font-mono tracking-wider text-zinc-500">Mã Sư Phụ</span>
                  <span className="text-lg font-mono font-bold text-amber-400 tracking-wider">
                    {store.referralsData?.ref_code || "CHƯA_CÓ"}
                  </span>
                </div>
                <div className="bg-zinc-950/80 p-3.5 rounded-xl border border-zinc-800 flex flex-col gap-1 relative group">
                  <span className="text-[10px] uppercase font-mono tracking-wider text-zinc-500">Liên Kết Sư Đồ</span>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-mono text-zinc-300 truncate max-w-[200px]">
                      {typeof window !== "undefined"
                        ? `${window.location.origin}/?ref=${store.referralsData?.ref_code || ""}`
                        : "---"}
                    </span>
                    <button
                      onClick={() => {
                        const link = typeof window !== "undefined"
                          ? `${window.location.origin}/?ref=${store.referralsData?.ref_code || ""}`
                          : "";
                        if (link) {
                          navigator.clipboard.writeText(link);
                          alert("Đã sao chép Liên Kết Sư Đồ!");
                        }
                      }}
                      className="text-[10px] font-mono font-bold text-zinc-950 bg-amber-400 hover:bg-amber-300 px-2.5 py-1 rounded transition-all cursor-pointer shadow-md"
                    >
                      Copy Link
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Simulated Purchase (Simulator Widget) */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-3">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2">
                <Coins className="w-5 h-5 text-emerald-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Giả Lập Nạp Tiền</span>
              </div>
              <p className="text-xs text-zinc-400 leading-relaxed font-serif">
                Giả lập nạp tiền bằng Thẻ Nguyên Thạch để kiểm nghiệm tính năng nhận hoa hồng sư đồ (10% giá trị nạp tiền sẽ được gửi trực tiếp cho sư phụ của bạn nếu đệ tử đã verified).
              </p>
              <div className="flex flex-wrap gap-2.5 mt-1">
                <button
                  onClick={() => store.simulateDeposit(100)}
                  disabled={store.actionLoading}
                  className="flex-1 min-w-[120px] py-2 bg-emerald-650 hover:bg-emerald-600 disabled:bg-zinc-800 text-zinc-100 font-mono text-xs rounded-xl transition-all cursor-pointer font-bold shadow-md border border-emerald-500/25"
                >
                  Nạp $1 (100 NT)
                </button>
                <button
                  onClick={() => store.simulateDeposit(1000)}
                  disabled={store.actionLoading}
                  className="flex-1 min-w-[120px] py-2 bg-emerald-650 hover:bg-emerald-600 disabled:bg-zinc-800 text-zinc-100 font-mono text-xs rounded-xl transition-all cursor-pointer font-bold shadow-md border border-emerald-500/25"
                >
                  Nạp $10 (1000 NT)
                </button>
                <button
                  onClick={() => store.simulateDeposit(5000)}
                  disabled={store.actionLoading}
                  className="flex-1 min-w-[120px] py-2 bg-emerald-650 hover:bg-emerald-600 disabled:bg-zinc-800 text-zinc-100 font-mono text-xs rounded-xl transition-all cursor-pointer font-bold shadow-md border border-emerald-500/25"
                >
                  Nạp $50 (5000 NT)
                </button>
              </div>
            </div>

            {/* Referees list */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center justify-between border-b border-zinc-900 pb-2.5">
                <div className="flex items-center gap-2">
                  <User className="w-5 h-5 text-amber-400" />
                  <span className="font-serif font-bold text-zinc-100 text-sm">Danh Sách Đệ Tử ({store.referralsData?.total_ref || 0})</span>
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs font-mono">
                  <thead>
                    <tr className="border-b border-zinc-800 text-zinc-500 uppercase tracking-wider text-[10px]">
                      <th className="py-2.5">Đạo hiệu</th>
                      <th className="py-2.5">Tu vi</th>
                      <th className="py-2.5">IP/Thiết bị</th>
                      <th className="py-2.5 text-center">Trạng Thái</th>
                      <th className="py-2.5 text-right">Mốc Quà</th>
                    </tr>
                  </thead>
                  <tbody>
                    {store.referralsData?.referees && store.referralsData.referees.length > 0 ? (
                      store.referralsData.referees.map((referee: any) => (
                        <tr key={referee.id} className="border-b border-zinc-900 hover:bg-zinc-950/20">
                          <td className="py-3 font-semibold text-zinc-200">{referee.username}</td>
                          <td className="py-3 text-amber-500">{RANK_NAMES[referee.rank] || `Chuyển ${referee.rank}`}</td>
                          <td className="py-3 text-[10px] text-zinc-500">
                            {referee.ip || "Không rõ"}
                          </td>
                          <td className="py-3 text-center">
                            {referee.is_suspicious ? (
                              <span className="bg-red-500/10 text-red-400 border border-red-500/20 px-2 py-0.5 rounded text-[10px]">
                                ⚠️ Trùng IP
                              </span>
                            ) : (
                              <span className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-2 py-0.5 rounded text-[10px]">
                                Hợp lệ
                              </span>
                            )}
                          </td>
                          <td className="py-3 text-right space-y-1">
                            <div className="flex flex-col gap-1 items-end">
                              <span className={`text-[10px] px-1.5 py-0.5 rounded ${referee.milestone1_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                Mốc 1 (Cấp 2): {referee.milestone1_rewarded ? "Đã nhận 500 NT" : "Chưa đạt"}
                              </span>
                              <span className={`text-[10px] px-1.5 py-0.5 rounded ${referee.milestone2_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                Mốc 2 (Cấp 3): {referee.milestone2_rewarded ? "Đã nhận Cổ trùng" : "Chưa đạt"}
                              </span>
                              <span className={`text-[10px] px-1.5 py-0.5 rounded ${referee.milestone3_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : referee.has_deposited ? 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                Mốc 3 (Nạp thẻ): {referee.milestone3_rewarded ? "Đã nhận (10%)" : referee.has_deposited ? "Chờ verified..." : "Chưa nạp"}
                              </span>
                            </div>
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td colSpan={5} className="py-8 text-center text-zinc-500 italic">
                          Chưa chiêu mộ được đệ tử nào. Hãy gửi liên kết sư đồ cho bằng hữu!
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {activeTab === "settings" && (
          <div className="w-full z-10 space-y-5">
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4 text-left">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Crown className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Thiết Lập Hồ Sơ</span>
              </div>

              <form onSubmit={async (e) => {
                e.preventDefault();
                const usernameInput = (e.currentTarget.elements.namedItem("username") as HTMLInputElement).value;
                const bioInput = (e.currentTarget.elements.namedItem("bio") as HTMLTextAreaElement).value;
                const newPasswordInput = (e.currentTarget.elements.namedItem("newPassword") as HTMLInputElement).value;

                const success = await store.updateProfileSettings({
                  username: usernameInput,
                  bio: bioInput,
                  new_password: newPasswordInput || undefined,
                  is_disguised: isDisguised,
                  fake_rank: Number(fakeRank),
                  fake_sub_rank: Number(fakeSubRank),
                  fake_vital_gu: fakeVitalGu,
                });
                if (success) {
                  // Clear password field
                  (e.currentTarget?.elements.namedItem("newPassword") as HTMLInputElement).value = "";
                }
              }} className="space-y-4">
                {/* Username */}
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="settings-username" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Đạo Hiệu (Username)</label>
                  <input
                    id="settings-username"
                    name="username"
                    type="text"
                    defaultValue={player.username || ""}
                    required
                    className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-3.5 py-2 text-xs text-zinc-200 placeholder-zinc-700 focus:outline-none transition-all font-mono"
                  />
                </div>

                {/* Lời giới thiệu / Bio */}
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="settings-bio" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Lời giới thiệu (Bio)</label>
                  <textarea
                    id="settings-bio"
                    name="bio"
                    defaultValue={player.bio || ""}
                    maxLength={150}
                    placeholder="Viết lời giới thiệu về nhân vật của đạo hữu..."
                    rows={3}
                    className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-3.5 py-2 text-xs text-zinc-200 placeholder-zinc-750 focus:outline-none transition-all font-mono resize-none leading-relaxed"
                  />
                </div>

                {/* Ảnh đại diện / Avatar Upload */}
                <div className="flex flex-col gap-2 border-t border-zinc-900/60 pt-3.5">
                  <label className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                    <Camera className="w-3 h-3" /> Ảnh Đại Diện
                  </label>
                  <div className="flex items-center gap-4">
                    <div className="w-16 h-16 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center text-zinc-100 font-bold border-2 border-zinc-900 shrink-0 overflow-hidden">
                      {player.avatar_url ? (
                        <img src={player.avatar_url} alt={player.username} className="w-full h-full object-cover" />
                      ) : (
                        <User className="w-7 h-7" />
                      )}
                    </div>
                    <div className="flex flex-col gap-2 flex-1">
                      <label
                        htmlFor="avatar-upload"
                        className="cursor-pointer inline-flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-mono bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-zinc-100 border border-zinc-800 rounded-lg transition-all"
                      >
                        <Camera className="w-3.5 h-3.5" /> Chọn ảnh
                      </label>
                      <input
                        id="avatar-upload"
                        type="file"
                        accept="image/jpeg,image/png,image/webp"
                        className="hidden"
                        onChange={async (e) => {
                          const file = e.target.files?.[0];
                          if (!file) return;
                          if (file.size > 2 * 1024 * 1024) {
                            store.addToast("Ảnh quá lớn! Tối đa 2MB.", "error");
                            return;
                          }
                          await store.uploadAvatar(file);
                          e.target.value = "";
                        }}
                      />
                      <span className="text-[9px] text-zinc-600 font-mono">JPEG, PNG, WebP • Tối đa 2MB</span>
                    </div>
                  </div>
                </div>

                {/* Đổi mật khẩu mới */}
                <div className="flex flex-col gap-1.5 border-t border-zinc-900/60 pt-3.5">
                  <label htmlFor="settings-newPassword" className="text-[10px] font-mono text-rose-400 uppercase tracking-wider">Mật khẩu mới (Bỏ trống nếu giữ nguyên)</label>
                  <input
                    id="settings-newPassword"
                    name="newPassword"
                    type="password"
                    placeholder="Mật khẩu mới (tối thiểu 6 ký tự)"
                    className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-3.5 py-2 text-xs text-zinc-200 placeholder-zinc-750 focus:outline-none transition-all font-mono"
                  />
                </div>

                {/* Ngụy Trang Khí Tức (Disguise System) */}
                <div className="flex flex-col gap-4 border-t border-zinc-900/60 pt-4 mt-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {isDisguised ? (
                        <EyeOff className="w-4 h-4 text-amber-500" />
                      ) : (
                        <Eye className="w-4 h-4 text-zinc-500" />
                      )}
                      <span className="text-xs font-mono font-bold text-zinc-200 uppercase tracking-wider">Ngụy Trang Hồ Sơ</span>
                    </div>

                    <label className="relative inline-flex items-center cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={isDisguised}
                        disabled={!hasConcealment}
                        onChange={(e) => setIsDisguised(e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-9 h-5 bg-zinc-950 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-zinc-700 after:border-zinc-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-amber-600 peer-checked:after:bg-amber-100 peer-disabled:opacity-20 peer-disabled:cursor-not-allowed"></div>
                    </label>
                  </div>

                  {!hasConcealment && (
                    <div className="p-3 bg-rose-950/20 border border-rose-500/20 rounded-xl text-[10px] text-rose-300 leading-relaxed font-sans">
                      <ShieldAlert className="w-3.5 h-3.5 inline mr-1.5 shrink-0 align-text-top text-rose-450" />
                      Đạo hữu không trang bị cổ trùng ngụy trang nào hoạt động! Hãy trang bị các cổ trùng như Ẩn Lân Cổ, Biến Hóa Cổ, Thái Độ Cổ... để kích hoạt tính năng Ngụy Trang.
                    </div>
                  )}

                  {hasConcealment && isDisguised && (
                    <div className="space-y-3 p-3 bg-zinc-950/40 border border-zinc-900 rounded-xl">
                      {/* Fake Rank & Sub Rank */}
                      <div className="grid grid-cols-2 gap-3">
                        <div className="flex flex-col gap-1.5">
                          <label htmlFor="settings-fakeRank" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Tu Vi Giả</label>
                          <select
                            id="settings-fakeRank"
                            value={fakeRank}
                            onChange={(e) => setFakeRank(Number(e.target.value))}
                            className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-2 py-2 text-xs text-zinc-200 focus:outline-none transition-all font-mono"
                          >
                            {Array.from({ length: hasAttitude ? 9 : player.rank }, (_, i) => i + 1).map((r) => (
                              <option key={r} value={r}>
                                {RANK_NAMES[r] || `${r} Chuyển`}
                              </option>
                            ))}
                          </select>
                          {!hasAttitude && (
                            <span className="text-[8px] text-amber-500 font-mono leading-tight">
                              * Chỉ giả yếu (&le; {RANK_NAMES[player.rank]}). Cần Thái Độ Cổ để giả mạnh!
                            </span>
                          )}
                        </div>

                        {/* Fake Sub Rank */}
                        <div className="flex flex-col gap-1.5">
                          <label htmlFor="settings-fakeSubRank" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Cảnh Giới Giả</label>
                          <select
                            id="settings-fakeSubRank"
                            value={fakeSubRank}
                            onChange={(e) => setFakeSubRank(Number(e.target.value))}
                            className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-2 py-2 text-xs text-zinc-200 focus:outline-none transition-all font-mono"
                          >
                            {Object.entries(SUB_RANK_NAMES).map(([key, val]) => (
                              <option key={key} value={key}>
                                {val}
                              </option>
                            ))}
                          </select>
                        </div>
                      </div>

                      {/* Fake Vital Gu */}
                      <div className="flex flex-col gap-1.5">
                        <div className="flex justify-between items-center">
                          <label htmlFor="settings-fakeVitalGu" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Bản Mệnh Cổ Giả</label>
                          {!hasFaceChange && (
                            <span className="text-[8px] bg-rose-950/40 text-rose-450 border border-rose-900/30 px-1.5 py-0.5 rounded font-mono font-semibold uppercase">Yêu cầu Biến Hóa/Thái Độ Cổ</span>
                          )}
                        </div>
                        <select
                          id="settings-fakeVitalGu"
                          disabled={!hasFaceChange}
                          value={fakeVitalGu}
                          onChange={(e) => setFakeVitalGu(e.target.value)}
                          className="w-full bg-zinc-950 disabled:bg-zinc-950/30 disabled:opacity-40 border border-zinc-900 focus:border-zinc-850 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none transition-all font-mono"
                        >
                          {ownedGuIds.map((guId) => {
                            const name = guDictionary[guId]?.name || guId;
                            return (
                              <option key={guId} value={guId}>
                                {name} ({guId})
                              </option>
                            );
                          })}
                        </select>
                      </div>
                    </div>
                  )}
                </div>

                {/* Cài Đặt Chiến Đấu Tự Động */}
                <div className="flex flex-col gap-4 border-t border-zinc-900/60 pt-4 mt-2">
                  <div className="flex items-center gap-2">
                    <Swords className="w-4 h-4 text-amber-400" />
                    <span className="text-xs font-mono font-bold text-zinc-200 uppercase tracking-wider">Cài Đặt Chiến Thuật Tự Động</span>
                  </div>

                  {/* Auto Heal HP */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input type="checkbox" className="sr-only peer" checked={csAutoHealHP}
                          onChange={e => setCsAutoHealHP(e.target.checked)} />
                        <div className="w-9 h-5 bg-zinc-700 rounded-full peer peer-checked:bg-emerald-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
                      </label>
                      <span className="text-xs text-zinc-300">Tự dùng dược liệu hồi HP</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] text-zinc-500">HP ≤</span>
                      <input type="range" min="10" max="80" value={csHealThreshold}
                        onChange={e => setCsHealThreshold(Number(e.target.value))}
                        disabled={!csAutoHealHP} className="w-20 h-1 accent-emerald-500" />
                      <span className="text-xs text-zinc-300 w-8">{csHealThreshold}%</span>
                    </div>
                  </div>

                  {/* Auto Restore Mana */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input type="checkbox" className="sr-only peer" checked={csAutoRestoreMana}
                          onChange={e => setCsAutoRestoreMana(e.target.checked)} />
                        <div className="w-9 h-5 bg-zinc-700 rounded-full peer peer-checked:bg-cyan-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
                      </label>
                      <span className="text-xs text-zinc-300">Tự dùng Nguyên Thạch hồi Mana</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] text-zinc-500">Mana ≤</span>
                      <input type="range" min="10" max="80" value={csManaThreshold}
                        onChange={e => setCsManaThreshold(Number(e.target.value))}
                        disabled={!csAutoRestoreMana} className="w-20 h-1 accent-cyan-500" />
                      <span className="text-xs text-zinc-300 w-8">{csManaThreshold}%</span>
                    </div>
                  </div>

                  {/* Gu Priorities */}
                  <div className="border-t border-zinc-900/30 pt-3">
                    <div className="text-[10px] text-zinc-400 font-bold mb-2">Cổ Trùng (sắp xếp theo độ ưu tiên)</div>
                    {eligibleGusForCombat.length === 0 ? (
                      <p className="text-[10px] text-zinc-600 italic">Chưa có cổ trùng nào được trang bị.</p>
                    ) : (
                      <div className="space-y-1.5 max-h-48 overflow-y-auto pr-1">
                        {eligibleGusForCombat.map((gu, idx) => {
                          const spec = guDictionary[gu.gu_id];
                          const tactic = csGuPriorities.find(t => t.gu_id === gu.gu_id);
                          const priority = tactic?.priority ?? idx + 1;
                          const condition = tactic?.use_condition || "always";
                          const condVal = tactic?.condition_value || 30;
                          return (
                            <div key={gu.instance_id} className="flex items-center gap-2 bg-zinc-950/50 rounded-lg px-2.5 py-1.5 border border-zinc-900">
                              <span className="text-[10px] text-zinc-500 font-mono w-4">{priority}</span>
                              <span className="text-[11px] text-zinc-200 flex-1 truncate">{spec?.name || gu.gu_id}</span>
                              <select value={condition}
                                onChange={e => {
                                  const newList = csGuPriorities.filter(t => t.gu_id !== gu.gu_id);
                                  newList.push({ gu_id: gu.gu_id, priority, use_condition: e.target.value, condition_value: condVal });
                                  setCsGuPriorities(newList);
                                }}
                                className="bg-zinc-950 border border-zinc-800 rounded text-[9px] text-zinc-300 px-1.5 py-0.5 font-mono"
                              >
                                <option value="always">Luôn</option>
                                <option value="hp_below">HP &lt; %</option>
                                <option value="mana_above">Mana &gt; %</option>
                                <option value="enemy_hp_below">Máu địch &lt; %</option>
                              </select>
                              {condition !== "always" && (
                                <input type="number" min="1" max="99" value={condVal}
                                  onChange={e => {
                                    const newList = csGuPriorities.filter(t => t.gu_id !== gu.gu_id);
                                    newList.push({ gu_id: gu.gu_id, priority, use_condition: condition, condition_value: Number(e.target.value) });
                                    setCsGuPriorities(newList);
                                  }}
                                  className="w-12 bg-zinc-950 border border-zinc-800 rounded text-[9px] text-zinc-300 px-1.5 py-0.5 font-mono text-center"
                                />
                              )}
                              <div className="flex gap-0.5">
                                <button type="button"
                                  onClick={() => {
                                    const newList = csGuPriorities.map(t => t.gu_id === gu.gu_id ? { ...t, priority: Math.max(1, priority - 1) } : t);
                                    setCsGuPriorities(newList);
                                  }}
                                  className="text-[10px] px-1.5 py-0.5 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded">▲</button>
                                <button type="button"
                                  onClick={() => {
                                    const newList = csGuPriorities.map(t => t.gu_id === gu.gu_id ? { ...t, priority: priority + 1 } : t);
                                    setCsGuPriorities(newList);
                                  }}
                                  className="text-[10px] px-1.5 py-0.5 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded">▼</button>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>

                  {/* Killer Move Priorities */}
                  <div className="border-t border-zinc-900/30 pt-3">
                    <div className="text-[10px] text-zinc-400 font-bold mb-2">Sát Chiêu (ưu tiên)</div>
                    {killerMoves.length === 0 ? (
                      <p className="text-[10px] text-zinc-600 italic">Chưa có sát chiêu nào. Vào Sát Chiêu Đường để tạo.</p>
                    ) : (
                      <div className="space-y-1.5 max-h-48 overflow-y-auto pr-1">
                        {killerMoves.map((km, idx) => {
                          const tactic = csKMPriorities.find(t => t.killer_move_id === km.id);
                          const priority = tactic?.priority ?? idx + 1;
                          const condition = tactic?.use_condition || "always";
                          const condVal = tactic?.condition_value || 30;
                          return (
                            <div key={km.id} className="flex items-center gap-2 bg-zinc-950/50 rounded-lg px-2.5 py-1.5 border border-zinc-900">
                              <span className="text-[10px] text-zinc-500 font-mono w-4">{priority}</span>
                              <span className="text-[11px] text-zinc-200 flex-1 truncate">{km.name}</span>
                              <select value={condition}
                                onChange={e => {
                                  const newList = csKMPriorities.filter(t => t.killer_move_id !== km.id);
                                  newList.push({ killer_move_id: km.id, priority, use_condition: e.target.value, condition_value: condVal });
                                  setCsKMPriorities(newList);
                                }}
                                className="bg-zinc-950 border border-zinc-800 rounded text-[9px] text-zinc-300 px-1.5 py-0.5 font-mono"
                              >
                                <option value="always">Luôn</option>
                                <option value="hp_below">HP &lt; %</option>
                                <option value="mana_above">Mana &gt; %</option>
                                <option value="enemy_hp_below">Máu địch &lt; %</option>
                              </select>
                              {condition !== "always" && (
                                <input type="number" min="1" max="99" value={condVal}
                                  onChange={e => {
                                    const newList = csKMPriorities.filter(t => t.killer_move_id !== km.id);
                                    newList.push({ killer_move_id: km.id, priority, use_condition: condition, condition_value: Number(e.target.value) });
                                    setCsKMPriorities(newList);
                                  }}
                                  className="w-12 bg-zinc-950 border border-zinc-800 rounded text-[9px] text-zinc-300 px-1.5 py-0.5 font-mono text-center"
                                />
                              )}
                              <div className="flex gap-0.5">
                                <button type="button"
                                  onClick={() => {
                                    const newList = csKMPriorities.map(t => t.killer_move_id === km.id ? { ...t, priority: Math.max(1, priority - 1) } : t);
                                    setCsKMPriorities(newList);
                                  }}
                                  className="text-[10px] px-1.5 py-0.5 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded">▲</button>
                                <button type="button"
                                  onClick={() => {
                                    const newList = csKMPriorities.map(t => t.killer_move_id === km.id ? { ...t, priority: priority + 1 } : t);
                                    setCsKMPriorities(newList);
                                  }}
                                  className="text-[10px] px-1.5 py-0.5 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded">▼</button>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>

                  {/* Save Combat Settings */}
                  <button
                    type="button"
                    disabled={store.actionLoading}
                    onClick={async () => {
                      const success = await store.saveCombatSettings({
                        auto_heal_hp: csAutoHealHP,
                        heal_threshold: csHealThreshold,
                        auto_restore_mana: csAutoRestoreMana,
                        mana_threshold: csManaThreshold,
                        gu_priorities: csGuPriorities,
                        km_priorities: csKMPriorities,
                      });
                      if (success) {
                        store.addLog("Đã lưu cài đặt chiến thuật tự động.");
                      }
                    }}
                    className="w-full bg-gradient-to-r from-emerald-700 to-teal-600 hover:from-emerald-600 hover:to-teal-500 disabled:from-zinc-900 disabled:to-zinc-900 disabled:opacity-40 border border-emerald-500/20 text-white rounded-xl py-2 text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 active:scale-98 shadow-md"
                  >
                    {store.actionLoading ? (
                      <>
                        <div className="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                        <span>Đang lưu...</span>
                      </>
                    ) : (
                      <>
                        <Swords className="w-3.5 h-3.5" />
                        <span>Lưu Chiến Thuật</span>
                      </>
                    )}
                  </button>
                </div>

                {/* Submit button */}
                <button
                  type="submit"
                  disabled={store.actionLoading}
                  className="w-full mt-2 bg-gradient-to-r from-amber-600 to-orange-500 hover:from-amber-500 hover:to-orange-450 disabled:from-zinc-900 disabled:to-zinc-900 disabled:opacity-40 border border-amber-500/20 text-white rounded-xl py-2.5 text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 active:scale-98 shadow-md hover:shadow-orange-500/10"
                >
                  {store.actionLoading ? (
                    <>
                      <div className="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                      <span>Đang lưu...</span>
                    </>
                  ) : (
                    <span>Lưu Cài Đặt Hồ Sơ</span>
                  )}
                </button>
              </form>
            </div>
          </div>
        )}
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
            Ngươi đã đạt tới Ngũ Chuyển Đỉnh Phong. Nhân khí tích lũy: <span className="font-bold text-amber-400 font-mono">{player.human_qi?.toFixed(1) || 0}/1000</span> <br />
            Thăng Tiên là quá trình độ kiếp cực kỳ nguy hiểm, nếu thất bại tu vi sẽ vỡ nát!
          </div>
          <button
            className="px-5 py-2.5 bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 text-white font-bold text-xs rounded-xl shadow-[0_0_15px_rgba(244,3,94,0.45)] cursor-pointer transition-all active:scale-95 uppercase tracking-wider"
            onClick={async () => {
              if (confirm("Thăng tiên là cửu tử nhất sinh! Ngươi đã chuẩn bị kỹ?")) {
                const res = await store.ascension();
                if (res) alert(res.message);
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
            Trùng Kích Vách Khiếu
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
      <div className="flex justify-between items-center text-xs">
        <span className="text-zinc-400 font-medium">Thọ Nguyên Còn Lại:</span>
        <span className="text-xs font-semibold text-zinc-300 block">
          {(player.max_lifespan - player.lifespan).toFixed(1)} năm
        </span>
      </div>
      {/* Aperture Stress Level */}
      <div className="mt-3.5 p-3 bg-zinc-950/70 border border-zinc-900 rounded-xl">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5">
            <Activity className="w-3.5 h-3.5 text-rose-450 animate-pulse" />
            Áp Lực Không Khiếu
          </span>
          <span className={`text-xs font-mono font-bold ${(player.aperture_stress ?? 0) >= 80 ? "text-rose-500 animate-pulse font-extrabold" :
            (player.aperture_stress ?? 0) >= 50 ? "text-amber-500" :
              "text-emerald-400"
            }`}>
            {(player.aperture_stress ?? 0).toFixed(1)}%
          </span>
        </div>

        {/* Dial / Progress Bar */}
        <div className="relative w-full h-3.5 bg-zinc-900/80 rounded-full overflow-hidden border border-zinc-850 p-[2px]">
          <div
            className={`h-full rounded-full transition-all duration-300 ${(player.aperture_stress ?? 0) >= 80 ? "bg-gradient-to-r from-red-600 via-rose-500 to-red-500 animate-pulse shadow-[0_0_8px_rgba(239,68,68,0.6)]" :
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
  const aperture = store.aperture;
  const [rechargeStones, setRechargeStones] = React.useState<number>(10);

  React.useEffect(() => {
    store.fetchAperture();
    store.fetchDaoMarks();
    store.fetchActiveTribulation();
  }, []);

  if (!player) return null;

  // Render helper for grade
  const getGradeText = (g?: string) => {
    if (g === "LOW") return "Hạ Đẳng";
    if (g === "MID") return "Trung Đẳng";
    if (g === "HIGH") return "Thượng Đẳng";
    if (g === "EXTREME") return "Chí Cao";
    return "Không Xác Định";
  };

  // Render helper for state
  const getStateText = (s?: string) => {
    if (s === "STABLE") return { text: "Ổn Định", color: "text-emerald-400" };
    if (s === "TRIBULATION_ACTIVE") return { text: "Có Tai Kiếp Đang Tấn Công", color: "text-rose-400 animate-pulse font-bold" };
    if (s === "TRIBULATION_PENDING") return { text: "Kiếp Lực Tích Lũy", color: "text-yellow-400" };
    if (s === "DAMAGED") return { text: "Hư Tổn", color: "text-amber-500" };
    return { text: "Ổn Định", color: "text-emerald-400" };
  };

  const stateInfo = getStateText(aperture?.state);

  // Fallback map if database daoMarks isn't synced yet
  const displayMarks = store.daoMarks.length > 0
    ? store.daoMarks
    : Object.entries(player.dao_marks || {}).map(([path, amount]) => ({ path, amount }));

  return (
    <div className="w-full">
      <div className="p-4 border border-zinc-900 bg-zinc-950/30 rounded-2xl space-y-4">
        {/* Tiên Khiếu Stats Summary */}
        <div className="space-y-2.5">
          <div className="flex justify-between items-center text-xs">
            <span className="text-zinc-400 font-medium">Quy mô Tiên Khiếu:</span>
            <span className="text-zinc-200 font-bold">
              {aperture ? getGradeText(aperture.grade) : "Đang tải..."}
            </span>
          </div>

          <div className="flex justify-between items-center text-xs">
            <span className="text-zinc-400 font-medium">Trạng thái:</span>
            <span className={stateInfo.color}>{stateInfo.text}</span>
          </div>

          {aperture && (
            <>
              {/* Stats Grid */}
              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-400 font-medium">Thọ Nguyên Còn Lại:</span>
                <span className="text-xs font-semibold text-zinc-300 block">
                  {(player.max_lifespan - player.lifespan).toFixed(1)} năm
                </span>
              </div>
              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-400 font-medium">Lưu tốc thời gian:</span>
                <span className="text-cyan-400 font-semibold">x{aperture.time_rate} lần</span>
              </div>

              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-400 font-medium">Tiên Nguyên trong khiếu:</span>
                <span className="text-yellow-400 font-semibold">{Math.floor(player.current_mana)} hạt</span>
              </div>

              <div className="space-y-1">
                <div className="flex justify-between text-[10px] text-zinc-500 font-mono">
                  <span>Chu kỳ Kiếp nạn ({aperture.internal_years_passed.toFixed(2)} / {aperture.next_tribulation_year.toFixed(1)} năm)</span>
                  <span>{((aperture.internal_years_passed / aperture.next_tribulation_year) * 100).toFixed(1)}%</span>
                </div>
                <div className="w-full bg-zinc-950 h-1.5 rounded-full overflow-hidden border border-zinc-900">
                  <div
                    className="h-full bg-gradient-to-r from-cyan-600 to-amber-500"
                    style={{ width: `${Math.min(100, (aperture.internal_years_passed / aperture.next_tribulation_year) * 100)}%` }}
                  />
                </div>
              </div>
            </>
          )}
        </div>

        {/* Enter Tiên Khiếu Button */}
        <div className="flex justify-center pt-2">
          <button
            onClick={() => store.setView("aperture")}
            className="w-full py-2 bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white text-xs font-bold rounded-xl shadow-lg active:scale-95 transition-all cursor-pointer uppercase tracking-wider flex items-center justify-center gap-1"
          >
            <Sparkles className="w-4 h-4 animate-pulse" />
            Vào Tiên Khiếu Phúc Địa
          </button>
        </div>

        {/* Stones Recharge Section for Immortals */}
        <div className="bg-zinc-950/50 border border-zinc-900 rounded-xl p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[10px] text-zinc-450 font-bold uppercase tracking-wider flex items-center gap-1.5">
              <Sparkles className="w-4 h-4 text-violet-400" />
              Nạp Tiên Nguyên Thạch (Recharge)
            </span>
            <span className="text-[10px] text-cyan-400 font-mono font-bold">
              Khả dụng: {player.immortal_stones.toLocaleString()} Tiên Nguyên Thạch
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
                max={player.immortal_stones}
              />
              <span className="text-[10px] text-zinc-500 font-mono font-semibold ml-2">Tiên Nguyên Thạch</span>
            </div>
            <div className="flex gap-1.5">
              <button
                type="button"
                onClick={() => setRechargeStones(player.immortal_stones)}
                className="px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-zinc-850 rounded-xl text-[10px] text-zinc-300 font-bold transition-all active:scale-95 cursor-pointer font-mono"
              >
                MAX
              </button>
              <button
                type="button"
                className="px-4 py-2 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 text-white font-bold rounded-xl text-xs shadow-lg shadow-cyan-950/20 active:scale-95 transition-all cursor-pointer shrink-0 uppercase tracking-wide"
                onClick={async () => {
                  if (rechargeStones <= 0) return alert('Nhập số tiên nguyên thạch để nạp');
                  const res = await store.rechargeWithStones(rechargeStones);
                  if (res) alert(res.message);
                }}
              >
                Luyện hóa Tiên Nguyên
              </button>
            </div>
          </div>
        </div>

        {/* Dao Marks summary */}
        <div className="border-t border-zinc-900 pt-3">
          <div className="text-[9px] text-zinc-550 text-center mb-2 uppercase tracking-widest font-bold font-mono">Đạo Ngân Tích Lũy</div>
          <div className="flex flex-wrap gap-2 justify-center">
            {displayMarks.map((mark: any, idx: number) => (
              <div key={idx} className="px-2.5 py-1 bg-zinc-950/80 border border-zinc-900 rounded-xl text-[10px] font-mono">
                <span className="text-amber-500 font-bold uppercase">{mark.path}</span>: <span className="text-zinc-200 font-bold">{mark.amount}</span>
              </div>
            ))}
            {displayMarks.length === 0 && (
              <div className="text-[10px] text-zinc-650 italic">Tiên Khiếu chưa ngưng tụ Đạo Ngân</div>
            )}
          </div>
        </div>
        {/* Path Masteries Section */}
        <div className="w-full text-left pt-2">
          <span className="text-[9px] text-zinc-500 uppercase tracking-wider block mb-2 font-mono">
            Lĩnh Ngộ Lưu Phái
          </span>
          <div className="space-y-1.5 max-h-36 overflow-y-auto pr-1">
            {player.path_masteries && Object.keys(player.path_masteries).length > 0 ? (
              Object.entries(player.path_masteries).map(([path, level]) => (
                <div key={path} className="flex justify-between items-center bg-zinc-950/30 px-2.5 py-1.5 rounded-lg border border-zinc-900/50 text-xs">
                  <span className="text-zinc-400 font-medium">{path}</span>
                  <span className={`px-2 py-0.5 rounded text-[10px] font-semibold font-mono ${level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                    level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                      level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                        "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                    }`}>
                    {level}
                  </span>
                </div>
              ))
            ) : (
              <div className="text-[10px] text-zinc-655 italic">Chưa có lĩnh ngộ lưu phái đặc thù.</div>
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

  const checkCtx = {
    rank: player.rank,
    guInventory: player.gu_inventory || [],
    clearedCells: player.cleared_cells || {},
    guDictionary: store.guDictionary || {},
  };
  const quests = defaultQuests(checkCtx);

  const getQuestIcon = (icon: string) => {
    switch (icon) {
      case "sparkles": return <Sparkles className="w-3.5 h-3.5 text-emerald-400" />;
      case "flame": return <Flame className="w-3.5 h-3.5 text-amber-500 animate-pulse" />;
      case "zap": return <Zap className="w-3.5 h-3.5 text-cyan-400" />;
      case "atom": return <Atom className="w-3.5 h-3.5 text-purple-450" />;
      case "crown": return <Crown className="w-3.5 h-3.5 text-yellow-500 animate-bounce" />;
      case "shield_alert": return <ShieldAlert className="w-3.5 h-3.5 text-rose-500" />;
      default: return <Award className="w-3.5 h-3.5 text-zinc-400" />;
    }
  };

  const buttonLabel = (q: typeof quests[0], isDone: boolean, canClaim: boolean) => {
    if (isDone) return "ĐÃ NHẬN";
    if (!canClaim) return "CHƯA ĐẠT";
    const rewardStr = formatRewards(q.rewards);
    return `NHẬN${rewardStr ? " " + rewardStr : ""}`;
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
              className={`p-2.5 rounded-xl border text-xs flex justify-between items-center transition-all duration-300 hover:scale-[1.01] ${isDone ? "bg-emerald-950/10 border-emerald-950/20 opacity-70" :
                canClaim ? "bg-amber-950/15 border-amber-500/40 shadow-[0_0_10px_rgba(245,158,11,0.08)] animate-pulse" :
                  "bg-zinc-950/40 border-zinc-900"
                }`}
            >
              <div className="text-left flex-1 min-w-0 mr-3">
                <div className={`font-semibold flex items-center gap-2 ${isDone ? "text-emerald-450 line-through" : "text-zinc-200"}`}>
                  {getQuestIcon(q.icon)}
                  <span className="truncate">{q.title}</span>
                </div>
                <div className="text-[10px] text-zinc-500 mt-0.5 truncate">{q.desc}</div>
                <div className="text-[9px] text-zinc-400 mt-0.5 italic font-mono">{q.req_desc}</div>
              </div>

              <button
                disabled={isDone || !canClaim || store.actionLoading}
                onClick={async () => {
                  const res = await store.completeQuest(q.id);
                  if (res && res.success) {
                    alert(res.message);
                  }
                }}
                className={`px-3 py-1.5 rounded-xl text-[10px] font-bold font-mono transition-all shrink-0 cursor-pointer ${isDone
                  ? "bg-zinc-900/60 text-zinc-650 border border-zinc-950 cursor-not-allowed"
                  : canClaim
                    ? "bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 shadow-[0_0_10px_rgba(245,158,11,0.3)] active:scale-95"
                    : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                  }`}
              >
                {buttonLabel(q, isDone, canClaim)}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
