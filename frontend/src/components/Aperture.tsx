"use client";
import React, { useEffect, useState } from "react";
import { estimateManaFromSnapshot, estimateHPFromSnapshot, GuInstance, GuSpec, Player, ImmortalAperture, useGameStore } from "@/store/useGameStore";
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
  Gift,
  Ticket,
  Check,
  TrendingUp,
  Leaf,
  RefreshCw,
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

const TRIB_TYPE_LABELS: Record<string, string> = {
  earth_calamity_count: "Địa Tai",
  heavenly_tribulation_count: "Thiên Kiếp",
  grand_tribulation_count: "Hạo Kiếp",
  myriad_tribulation_count: "Vạn Kiếp",
  chaos_calamity_count: "Hỗn Độn",
};

function getTribulationTypesForRank(rank: number): string[] {
  switch (rank) {
    case 6: return ["earth_calamity_count", "heavenly_tribulation_count"];
    case 7: return ["earth_calamity_count", "heavenly_tribulation_count", "grand_tribulation_count"];
    case 8: return ["heavenly_tribulation_count", "grand_tribulation_count", "myriad_tribulation_count"];
    case 9: return ["grand_tribulation_count", "myriad_tribulation_count", "chaos_calamity_count"];
    default: return [];
  }
}

function getTribCount(a: ImmortalAperture, key: string): number {
  switch (key) {
    case "earth_calamity_count": return a.earth_calamity_count;
    case "heavenly_tribulation_count": return a.heavenly_tribulation_count;
    case "grand_tribulation_count": return a.grand_tribulation_count;
    case "myriad_tribulation_count": return a.myriad_tribulation_count;
    case "chaos_calamity_count": return a.chaos_calamity_count;
    default: return 0;
  }
}

function buildTribulationDisplay(rank: number, aperture: ImmortalAperture | null): string {
  if (!aperture) return "Cổ Tiên";
  const types = getTribulationTypesForRank(rank);
  const parts: string[] = [];
  for (const key of types) {
    const val = getTribCount(aperture, key);
    const label = TRIB_TYPE_LABELS[key] || key;
    if (val > 0) parts.push(`${label}: ${val}`);
  }
  if (parts.length === 0) return "Cổ Tiên";
  return "[" + parts.join(", ") + "]";
}

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
  text: string; textLight: string; waveSolid: string; waveAlpha: string;
}> = {
  1: { from: "#1b4332", to: "#081c15", glow: "rgba(52,211,153,0.35)", ring: "#2d6a4f", text: "var(--rank-1-text)", textLight: "#6ee7b7", waveSolid: "#10b981", waveAlpha: "rgba(16,185,129,0.4)" },
  2: { from: "#450a0a", to: "#1a0505", glow: "rgba(239,68,68,0.35)", ring: "#b91c1c", text: "var(--rank-2-text)", textLight: "#fca5a5", waveSolid: "#ef4444", waveAlpha: "rgba(239,68,68,0.4)" },
  3: { from: "#0f172a", to: "#020617", glow: "rgba(59,130,246,0.35)", ring: "#2563eb", text: "var(--rank-3-text)", textLight: "#93c5fd", waveSolid: "#3b82f6", waveAlpha: "rgba(59,130,246,0.4)" },
  4: { from: "#3b1f00", to: "#1c0f00", glow: "rgba(245,158,11,0.35)", ring: "#d97706", text: "var(--rank-4-text)", textLight: "#fde68a", waveSolid: "#f59e0b", waveAlpha: "rgba(245,158,11,0.4)" },
  5: { from: "#2e1065", to: "#12002f", glow: "rgba(139,92,246,0.4)", ring: "#7c3aed", text: "var(--rank-5-text)", textLight: "#c4b5fd", waveSolid: "#8b5cf6", waveAlpha: "rgba(139,92,246,0.4)" },
  6: { from: "#083344", to: "#02151e", glow: "rgba(6,182,212,0.4)", ring: "#0891b2", text: "var(--rank-6-text)", textLight: "#67e8f9", waveSolid: "#06b6d4", waveAlpha: "rgba(6,182,212,0.4)" },
  7: { from: "#500724", to: "#25000f", glow: "rgba(236,72,153,0.4)", ring: "#db2777", text: "var(--rank-7-text)", textLight: "#f9a8d4", waveSolid: "#ec4899", waveAlpha: "rgba(236,72,153,0.4)" },
  8: { from: "#1c1917", to: "#0c0a09", glow: "rgba(168,162,158,0.4)", ring: "#a8a29e", text: "var(--rank-8-text)", textLight: "#e7e5e4", waveSolid: "#a8a29e", waveAlpha: "rgba(168,162,158,0.4)" },
  9: { from: "#78350f", to: "#2d0f00", glow: "rgba(251,191,36,0.55)", ring: "#fbbf24", text: "var(--rank-9-text)", textLight: "#fef08a", waveSolid: "#fbbf24", waveAlpha: "rgba(251,191,36,0.5)" },
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

interface TribulationRealTimeInfoProps {
  aperture: ImmortalAperture;
  isCompact?: boolean;
}

const TribulationRealTimeInfo: React.FC<TribulationRealTimeInfoProps> = ({ aperture, isCompact }) => {
  const [timeLeftText, setTimeLeftText] = useState<string>("");
  const [arrivalTimeText, setArrivalTimeText] = useState<string>("");
  const [isPassed, setIsPassed] = useState<boolean>(false);

  useEffect(() => {
    const updateTime = () => {
      if (aperture.state === "TRIBULATION_ACTIVE") {
        setTimeLeftText("Đang diễn ra");
        setArrivalTimeText("Đang chiến đấu");
        setIsPassed(true);
        return;
      }

      const rate = aperture.time_rate || 1.0;
      const internalYearsLeft = aperture.next_tribulation_year - aperture.internal_years_passed;
      if (internalYearsLeft <= 0) {
        setTimeLeftText("Đang ập đến");
        setArrivalTimeText("Bất cứ lúc nào");
        setIsPassed(true);
        return;
      }

      const remainingRealSeconds = (internalYearsLeft * 43200.0) / rate;
      const updatedAtMs = new Date(aperture.updated_at).getTime();
      const arrivalTimeMs = updatedAtMs + remainingRealSeconds * 1000;
      const now = Date.now();
      const diff = arrivalTimeMs - now;

      if (diff <= 0) {
        setTimeLeftText("Đang ập đến");
        setArrivalTimeText("Bất cứ lúc nào");
        setIsPassed(true);
        return;
      }

      setIsPassed(false);

      const arrivalDate = new Date(arrivalTimeMs);
      const hours = String(arrivalDate.getHours()).padStart(2, '0');
      const minutes = String(arrivalDate.getMinutes()).padStart(2, '0');
      const seconds = String(arrivalDate.getSeconds()).padStart(2, '0');
      const day = String(arrivalDate.getDate()).padStart(2, '0');
      const month = String(arrivalDate.getMonth() + 1).padStart(2, '0');
      const year = arrivalDate.getFullYear();
      
      setArrivalTimeText(`${hours}:${minutes}:${seconds} ${day}/${month}/${year}`);

      const diffHours = Math.floor(diff / (60 * 60 * 1000));
      const diffMinutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
      const diffSeconds = Math.floor((diff % (60 * 1000)) / 1000);

      const parts = [];
      if (diffHours > 0) parts.push(`${diffHours}g`);
      if (diffMinutes > 0 || diffHours > 0) parts.push(`${diffMinutes}p`);
      parts.push(`${diffSeconds}s`);
      setTimeLeftText(parts.join(" "));
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, [aperture.next_tribulation_year, aperture.internal_years_passed, aperture.time_rate, aperture.state, aperture.updated_at]);

  if (isCompact) {
    return (
      <div className="flex justify-between items-center text-[10px] text-zinc-400 font-mono">
        <span>Thời gian thực:</span>
        <span className={isPassed ? "text-rose-400 font-bold animate-pulse" : "text-amber-400 font-bold"}>
          {arrivalTimeText} ({timeLeftText})
        </span>
      </div>
    );
  }

  return (
    <div className="space-y-1 bg-slate-950/40 p-3 rounded-lg border border-slate-800/50 mt-2">
      <div className="flex justify-between items-center text-xs">
        <span className="text-slate-400">Thời gian thực tế đến:</span>
        <span className={`font-mono font-semibold ${isPassed ? "text-rose-400 font-bold animate-pulse" : "text-amber-400 font-bold"}`}>
          {arrivalTimeText}
        </span>
      </div>
      <div className="flex justify-between items-center text-xs">
        <span className="text-slate-400">Còn lại :</span>
        <span className={`font-mono font-semibold ${isPassed ? "text-rose-400 font-bold animate-pulse" : "text-cyan-400 font-bold"}`}>
          {timeLeftText}
        </span>
      </div>
    </div>
  );
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

  // Build tribulation info string for immortals (rank 6+)
  const tribulationInfo = isImmortal ? buildTribulationDisplay(rank, store.aperture) : "";
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

  const [activeTab, setActiveTab] = useState<"aperture" | "growth" | "referrals" | "settings">("aperture");
  const [giftCodeInput, setGiftCodeInput] = useState("");
  const [giftCodeMsg, setGiftCodeMsg] = useState<string | null>(null);
  const [giftCodeLoading, setGiftCodeLoading] = useState(false);

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
    if (activeTab === "settings" && !combatSettingsLoaded && !store.combatSettingsLoading) {
      store.fetchCombatSettings().then(() => {
        setCombatSettingsLoaded(true);
      });
    }
  }, [activeTab, combatSettingsLoaded]);

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
    if (activeTab === "referrals") {
      void store.fetchReferrals();
    } else if (activeTab === "aperture" && isImmortal) {
      store.fetchAperture();
      store.fetchDaoMarks();
      store.fetchActiveTribulation();
    }
  }, [activeTab, isImmortal]);

  const [isReloading, setIsReloading] = useState(false);
  const handleReloadAll = async () => {
    setIsReloading(true);
    const promises: Promise<any>[] = [store.fetchReferrals()];
    if (isImmortal) {
      promises.push(store.fetchAperture(), store.fetchDaoMarks(), store.fetchActiveTribulation());
    }
    if (activeTab === "settings") {
      promises.push(store.fetchCombatSettings());
    }
    await Promise.all(promises);
    setIsReloading(false);
  };

  const aperture = store.aperture;
  const essenceProd = aperture?.essence_production_per_year ?? 10;
  const immortalRegen = (essenceProd / 120.0) * (1 + (Math.min(Math.max(subRank, 1), 4) - 1) * 0.15);
  const mortalRegen = aptitudeType === "Giáp" ? 8 : aptitudeType === "Ất" ? 6 : aptitudeType === "Bính" ? 4 : 2;
  const regenRate = (isImmortal ? immortalRegen : mortalRegen).toFixed(2);

  // Calculate real-world hourly regen rates using TimeSpeedMultiplier
  const timeSpeedMultiplier = player.time_speed_multiplier || 1;
  const hpRegenRealHourly = (player.hp_regen_rate || 0) * timeSpeedMultiplier;
  const manaRegenRealHourly = (player.mana_regen_rate || Number(regenRate)) * timeSpeedMultiplier;

  // Helper to format time remaining
  const formatTimeRemaining = (current: number, max: number, realRegenPerHour: number) => {
    if (current >= max || realRegenPerHour <= 0) return "Đã đầy";
    const hoursRemaining = (max - current) / realRegenPerHour;
    const totalMinutes = Math.floor(hoursRemaining * 60);
    const h = Math.floor(totalMinutes / 60);
    const m = totalMinutes % 60;
    if (h > 0) return `Đầy sau ${h}h ${m}m`;
    return `Đầy sau ${m}m`;
  };

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
          Tu Vi: <span className="font-bold" style={{ color: c.text }}>{rankName}</span>
          {isImmortal ? (
            <span className="font-bold text-[9px]" style={{ color: c.textLight }}>
              &nbsp;&nbsp;❖&nbsp;&nbsp;{tribulationInfo}
            </span>
          ) : (
            <span className="font-bold" style={{ color: c.text }}>
              &nbsp;&nbsp;❖&nbsp;&nbsp;Cảnh Giới: {subRankName}
            </span>
          )}
        </p>


        {/* Reload + Tab Selection */}
        <div className="flex gap-2 w-full bg-zinc-950/60 p-1.5 rounded-2xl border border-zinc-900 mb-5 z-10 items-center">
          <button
            onClick={handleReloadAll}
            disabled={isReloading}
            className="px-2.5 py-1.5 rounded-xl text-zinc-400 hover:text-amber-400 border border-transparent hover:border-amber-500/20 transition-all cursor-pointer disabled:opacity-50 shrink-0"
            title="Tải lại dữ liệu"
          >
            <RefreshCw className={`w-4 h-4 ${isReloading ? "animate-spin" : ""}`} />
          </button>
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
            onClick={() => setActiveTab("growth")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "growth"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Trưởng Thành
          </button>
          <button
            onClick={() => setActiveTab("referrals")}
            className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${activeTab === "referrals"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Phúc Lợi
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
                    <div className="text-sm font-bold text-zinc-100 font-mono tracking-wide" style={{ color: c.textLight }}>
                      {manaQuality}:
                    </div>
                    <div className="text-xl font-bold font-mono text-cyan-400">
                      {player ? Math.floor(currentMana).toLocaleString() : "--"} viên
                    </div>
                    <div className="text-[10px] text-zinc-400 mt-0.5 tracking-wider uppercase font-semibold">{energyPoolName}</div>
                    <div className="text-[10px] text-zinc-300 mt-2 font-mono" style={{ color: c.textLight }}>
                      Tốc lưu tiên khiếu: +{manaRegenRealHourly.toFixed(1)} viên/giờ
                    </div>
                    {!isImmortal && currentMana < maxMana && manaRegenRealHourly > 0 && (
                      <div className="text-[9px] text-zinc-400 mt-0.5 font-mono">
                        {formatTimeRemaining(currentMana, maxMana, manaRegenRealHourly)}
                      </div>
                    )}
                  </div>
                </div>
              )}
              {/* Center text for mortals */}
              {!isImmortal && (
                <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none z-10 select-none">
                  <span className="text-3xl font-extrabold tracking-wide drop-shadow-[0_2px_8px_rgba(0,0,0,0.98)] font-mono" style={{ color: c.textLight }}>
                    {percent.toFixed(1)}%
                  </span>
                  <span className="text-xs font-bold mt-1 drop-shadow-[0_1.5px_4px_rgba(0,0,0,0.95)] font-mono" style={{ color: c.textLight }}>
                    {currentMana.toFixed(1)} / {maxMana.toFixed(0)}
                  </span>
                  <span className="text-[9px] mt-1 text-zinc-400 drop-shadow-[0_1px_3px_rgba(0,0,0,0.9)] uppercase font-semibold tracking-wider font-mono">
                    {energyPoolName}
                  </span>
                  {currentMana < maxMana && manaRegenRealHourly > 0 && (
                    <span className="text-[9px] mt-1 text-zinc-300 drop-shadow-[0_1px_3px_rgba(0,0,0,0.9)] font-mono">
                      +{manaRegenRealHourly.toFixed(1)}/giờ • {formatTimeRemaining(currentMana, maxMana, manaRegenRealHourly)}
                    </span>
                  )}
                </div>
              )}
            </div>

            {/* Aperture Stress Bar */}
            {!isImmortal && (
              <div className="relative h-6 w-full max-w-[280px] mx-auto overflow-hidden rounded-xl border border-amber-950/60 bg-zinc-950/60 shadow-[inset_0_2px_4px_rgba(0,0,0,0.6)] mb-4 flex items-center px-2 z-10">
                <ShieldAlert className="w-3.5 h-3.5 text-amber-500 mr-2 shrink-0 animate-pulse" />
                <div className="relative flex-grow h-3 rounded-md overflow-hidden bg-zinc-900/60">
                  <div
                    className="h-full bg-gradient-to-r from-amber-800 via-amber-600 to-yellow-500 rounded-md transition-all duration-300 shadow-[0_0_8px_rgba(245,158,11,0.5)]"
                    style={{ width: `${Math.min(100, player.aperture_stress ?? 0)}%` }}
                  />
                </div>
                <span className="text-[10px] font-mono font-bold text-zinc-300 ml-2.5 shrink-0">
                  Áp Lực: {(player.aperture_stress ?? 0).toFixed(0)}%
                </span>
              </div>
            )}

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
                    ? "bg-emerald-950/20 text-emerald-400 border-emerald-500/20"
                    : vitalGu.status === "SLEEPING"
                      ? "bg-amber-950/20 text-amber-400 border-amber-500/20"
                      : "bg-rose-950/20 text-rose-400 border-rose-500/20"
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


        {activeTab === "growth" && (
          <div className="w-full z-10 space-y-5">
            {/* Growth Boosts */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <TrendingUp className="w-5 h-5 text-emerald-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cốt Tủy Không Khiếu</span>
              </div>
              {(() => {
                const gd = player.growth_data;
                const b = gd?.boosts;
                const hasBoosts = b && (b.attack > 0 || b.defense > 0 || b.hp > 0 || b.speed > 0);
                if (!hasBoosts) return <div className="text-xs text-zinc-500 italic py-2">Chưa có tăng trưởng nào. Hãy trang bị Cổ trùng có hiệu ứng Trưởng Thành.</div>;
                return (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {b.attack > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Công Kích</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{b.attack}</div>
                      </div>
                    )}
                    {b.defense > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Phòng Ngự</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{b.defense}</div>
                      </div>
                    )}
                    {b.hp > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Máu Tối Đa</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{b.hp}</div>
                      </div>
                    )}
                    {b.speed > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Tốc Độ</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{b.speed}</div>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>

            {/* Passive Effects from Equipped Gu */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Zap className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Bị Động Cổ Trùng Đang Trang Bị</span>
              </div>
              {(() => {
                const passiveTotals: Record<string, number> = {};
                for (const gu of player.gu_inventory) {
                  if (!gu.equipped || gu.status !== "ACTIVE") continue;
                  const spec = guDictionary[gu.gu_id];
                  if (!spec?.effects?.passive) continue;
                  const p = spec.effects.passive;
                  for (const [k, v] of Object.entries(p)) {
                    if (typeof v === "number" && v !== 0) {
                      passiveTotals[k] = (passiveTotals[k] || 0) + v;
                    }
                  }
                }
                const labels: Record<string, string> = {
                  stat_base_attack: "Công Kích", stat_base_defense: "Phòng Ngự", stat_base_hp: "Máu",
                  stat_base_speed: "Tốc Độ", soul_refinement: "Hồn Lực", storage_capacity: "Vật Chứa",
                  movement: "Di Chuyển", evasion: "Né Tránh", investigation: "Trinh Sát",
                  hearing: "Thính Giác", mana_regen_mult: "Hồi Pháp",
                  damage_multiplier_on_hit: "Sát Thương Khi Mất Máu",
                  refine_mana_tier: "Tinh Luyện", cleansing_power: "Tẩy Trừ",
                  path_conflict_reduction: "Giảm Xung Đột",
                };
                const entries = Object.entries(passiveTotals).filter(([k]) => labels[k]);
                if (entries.length === 0) return <div className="text-xs text-zinc-500 italic py-2">Không có Cổ trùng bị động nào đang trang bị.</div>;
                return (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {entries.map(([k, v]) => (
                      <div key={k} className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">{labels[k] || k}</div>
                        <div className="text-base font-bold text-zinc-100 font-mono">+{v.toFixed(1)}</div>
                      </div>
                    ))}
                  </div>
                );
              })()}
            </div>

            {/* In-progress + Fully Grown Gu List */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner space-y-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Leaf className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cổ Trùng Đang Có Hiệu Ứng</span>
                <span className="ml-auto text-[10px] text-zinc-500 font-mono">{(player.growth_data?.fully_grown || []).length} loại</span>
              </div>
              {(() => {
                const growing = (player.gu_inventory || []).filter(gu => {
                  if (!gu.equipped || gu.status !== "ACTIVE") return false;
                  const spec = guDictionary[gu.gu_id];
                  if (!spec?.effects?.growth?.hours_to_max) return false;
                  const isDone = (player.growth_data?.fully_grown || []).some(fg => fg.gu_id === gu.gu_id);
                  return !isDone;
                });
                const fullyGrown = player.growth_data?.fully_grown || [];
                const hasAny = growing.length > 0 || fullyGrown.length > 0;
                if (!hasAny) return <div className="text-xs text-zinc-500 italic py-2">Chưa có Cổ trùng nào đang có hiệu ứng.</div>;
                return (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {growing.map((gu, i) => {
                      const spec = guDictionary[gu.gu_id];
                      const g = spec?.effects?.growth;
                      const maxH = g?.hours_to_max || 168;
                      const curH = gu.growth_hours || 0;
                      const pct = Math.min(100, Math.round((curH / maxH) * 100));
                      const parts: string[] = [];
                      if (g?.max_attack) parts.push(`Công +${g.max_attack}`);
                      if (g?.max_defense) parts.push(`Thủ +${g.max_defense}`);
                      if (g?.max_hp) parts.push(`Máu +${g.max_hp}`);
                      if (g?.max_speed) parts.push(`Tốc +${g.max_speed}`);
                      return (
                        <div key={gu.instance_id} className="flex items-center gap-3 bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                          <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center text-xs font-bold text-amber-500">
                            {(spec?.name || gu.gu_id).slice(0, 2)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="text-xs font-bold text-zinc-200 font-mono truncate">{spec?.name || gu.gu_id}</div>
                            <div className="text-[10px] text-zinc-400">{parts.join(" | ")}</div>
                            <div className="mt-1.5 flex items-center gap-2">
                              <div className="flex-1 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
                                <div className="h-full bg-emerald-500 rounded-full transition-all" style={{width: `${pct}%`}} />
                              </div>
                              <span className="text-[10px] text-zinc-500 font-mono shrink-0">{pct}%</span>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                    {fullyGrown.map((fg, i) => {
                      const parts: string[] = [];
                      if (fg.attack > 0) parts.push(`Công +${fg.attack}`);
                      if (fg.defense > 0) parts.push(`Thủ +${fg.defense}`);
                      if (fg.hp > 0) parts.push(`Máu +${fg.hp}`);
                      if (fg.speed > 0) parts.push(`Tốc +${fg.speed}`);
                      return (
                        <div key={`fg-${i}`} className="flex items-center gap-3 bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                          <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center text-xs font-bold text-zinc-400">
                            {(guDictionary[fg.gu_id]?.name || fg.gu_id).slice(0, 2)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="text-xs font-bold text-zinc-200 font-mono truncate">{guDictionary[fg.gu_id]?.name || fg.gu_id}</div>
                            <div className="text-[10px] text-emerald-400">{parts.join(" | ")}</div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                );
              })()}
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
                          store.addToast("Đã sao chép Liên Kết Sư Đồ!", "success");
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

            {/* Gift Code Redeem Card */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Gift className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Đổi Mã Quà</span>
              </div>
              <div className="flex flex-col md:flex-row items-stretch md:items-center gap-3">
                <input
                  value={giftCodeInput}
                  onChange={e => setGiftCodeInput(e.target.value.toUpperCase())}
                  placeholder="NHẬP MÃ QUÀ..."
                  className="flex-1 px-3 py-2.5 bg-zinc-950 border border-zinc-800 rounded-xl text-xs text-zinc-200 font-mono uppercase tracking-wider focus:outline-none focus:border-amber-500/50"
                />
                <button
                  disabled={giftCodeLoading || !giftCodeInput.trim()}
                  onClick={async () => {
                    setGiftCodeMsg(null);
                    setGiftCodeLoading(true);
                    const result = await store.redeemGiftCode(giftCodeInput.trim());
                    setGiftCodeLoading(false);
                    if (result?.success) {
                      setGiftCodeMsg(result.message || "Đổi mã quà thành công!");
                      setGiftCodeInput("");
                    } else {
                      setGiftCodeMsg(result?.message || "Mã quà không hợp lệ.");
                    }
                  }}
                  className="px-4 py-2.5 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-xl text-xs font-bold font-mono cursor-pointer transition-all whitespace-nowrap"
                >
                  {giftCodeLoading ? "Đang xử lý..." : "Nhận quà"}
                </button>
              </div>
              {giftCodeMsg && (
                <div className={`text-[10px] font-mono ${giftCodeMsg.includes("Nhận được") || giftCodeMsg.includes("thành công") ? "text-emerald-400" : "text-red-400"}`}>
                  {giftCodeMsg}
                </div>
              )}
            </div>

            {/* Simulated Purchase (Simulator Widget) */}
            {/* <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-3">
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
            </div> */}

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
                        <td colSpan={4} className="py-8 text-center text-zinc-500 italic">
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
                  <label htmlFor="settings-username" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Đạo Hiệu</label>
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
                  <label htmlFor="settings-bio" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Lời giới thiệu</label>
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
                    <div className="w-16 h-16 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center text-white font-bold border-2 border-zinc-900 shrink-0 overflow-hidden">
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

                {/* Ngụy Trang Khí Tức */}
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
                      <ShieldAlert className="w-3.5 h-3.5 inline mr-1.5 shrink-0 align-text-top text-rose-400" />
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
                            <span className="text-[8px] bg-rose-950/40 text-rose-400 border border-rose-900/30 px-1.5 py-0.5 rounded font-mono font-semibold uppercase">Yêu cầu Biến Hóa/Thái Độ Cổ</span>
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
                      <span className="text-xs text-zinc-300">Tự dùng Nguyên Thạch hồi Chân nguyên</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] text-zinc-500">Chân nguyên ≤</span>
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
                                <option value="mana_above">Chân nguyên &gt; %</option>
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
                                <option value="mana_above">Chân nguyên &gt; %</option>
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
                        store.addToast("Đã lưu cài đặt chiến thuật tự động thành công!", "success");
                      } else {
                        store.addToast("Lưu cài đặt chiến thuật tự động thất bại!", "error");
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
                  className="w-full mt-2 bg-gradient-to-r from-amber-600 to-orange-500 hover:from-amber-500 hover:to-orange-400 disabled:from-zinc-900 disabled:to-zinc-900 disabled:opacity-40 border border-amber-500/20 text-white rounded-xl py-2.5 text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 active:scale-98 shadow-md hover:shadow-orange-500/10"
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

  if (player.rank === 5 && player.sub_rank === 4 && (player.wall_progress ?? 0) >= 1) {
    return (
      <div className="w-full space-y-3.5">
        <div className="w-full flex flex-col items-center justify-center p-4 border border-rose-950 bg-rose-950/20 rounded-2xl shadow-[inset_0_0_12px_rgba(244,63,94,0.1)]">
          <div className="text-rose-400 font-bold mb-1 text-center text-sm tracking-wide uppercase flex items-center gap-1.5">
            <ShieldAlert className="w-4 h-4 text-rose-500 animate-bounce" />
            Cơ Hội Thăng Tiên
          </div>
          <div className="text-[10.5px] text-zinc-400 mb-3 text-center leading-relaxed">
            Ngươi đã đạt tới Ngũ Chuyển Đỉnh Phong. Nhân khí tích lũy: <span className="font-bold text-amber-400 font-mono">{player.human_qi?.toFixed(1) || 0}</span> <br />
            Thăng Tiên là quá trình độ kiếp cực kỳ nguy hiểm, nếu thất bại tu vi sẽ vỡ nát!
          </div>
          <button
            className="px-5 py-2.5 bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 text-white font-bold text-xs rounded-xl shadow-[0_0_15px_rgba(244,3,94,0.45)] cursor-pointer transition-all active:scale-95 uppercase tracking-wider"
            onClick={async () => {
              if (await store.showConfirm("Thăng tiên là cửu tử nhất sinh! Ngươi đã chuẩn bị kỹ?")) {
                const res = await store.ascension();
                if (res) store.addToast(res.message, "success");
              }
            }}
          >
            Liều Mạng Thăng Tiên
          </button>
        </div>

        {/* Stones Recharge Section */}
        <div className="bg-zinc-950/50 border border-zinc-900 rounded-xl p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5 font-mono">
              <Coins className="w-3.5 h-3.5 text-cyan-400" />
              Nạp Nguyên Thạch
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
                  if (rechargeStones <= 0) return store.addToast('Nhập số nguyên thạch để nạp', "error");
                  const res = await store.rechargeWithStones(rechargeStones);
                  if (res) store.addToast(res.message, "success");
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
          <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5 font-mono">
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
            <span className="text-[10px] text-zinc-500 font-mono font-semibold ml-2">Chân nguyên</span>
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
                if (manaSpend <= 0) return store.addToast('Nhập lượng mana cần dùng', "error");
                const res = await store.breakWall(manaSpend);
                if (res) store.addToast(res.message, "success");
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
          <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5 font-mono">
            <Coins className="w-3.5 h-3.5 text-cyan-400" />
            Nạp Nguyên Thạch
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
                if (rechargeStones <= 0) return store.addToast('Nhập số nguyên thạch để nạp', "error");
                const res = await store.rechargeWithStones(rechargeStones);
                if (res) store.addToast(res.message, "success");
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
                  {(player.lifespan ?? 0).toFixed(1)} năm
        </span>
      </div>
    </div>
  );
}

function BlessedLandControls() {
  const store = useGameStore();
  const player = store.player;
  const aperture = store.aperture;
  const [rechargeStones, setRechargeStones] = React.useState<number>(10);

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
          {(player.lifespan ?? 0).toFixed(1)} năm
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
                <TribulationRealTimeInfo aperture={aperture} isCompact={true} />
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
            <span className="text-[10px] text-zinc-400 font-bold uppercase tracking-wider flex items-center gap-1.5 font-mono">
              <Sparkles className="w-4 h-4 text-violet-400" />
              Nạp Tiên Nguyên Thạch
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
                  if (rechargeStones <= 0) return store.addToast('Nhập số tiên nguyên thạch để nạp', "error");
                  const res = await store.rechargeWithStones(rechargeStones);
                  if (res) store.addToast(res.message, "success");
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
          <span className="text-[9px] text-zinc-550 uppercase tracking-wider block mb-2 font-mono">
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
