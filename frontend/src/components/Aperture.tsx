"use client";
import React, { useEffect, useState, useMemo, useRef } from "react";
import { estimateManaFromSnapshot, estimateHPFromSnapshot, guSpecFor, GuInstance, GuSpec, Player, ImmortalAperture, PlayerStatusInstance, useGameStore } from "@/store/useGameStore";
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
  ChevronRight,
  ChevronDown,
  User,
  Camera,
  Gift,
  Ticket,
  Check,
  TrendingUp,
  Leaf,
  RefreshCw,
  Settings2,
  UserPlus,
  AlertTriangle,
  Trash2,
} from "lucide-react";
import { ClearCacheButton } from "@/components/ClearCacheModal";
import { formatNumber, statusRemainLabel, calculateTribulationProgress } from "@/lib/utils";
import { disguiseLevel, growthConfig, hasCapability } from "@/lib/effect-blocks";
import { EffectEnvelopeView } from "@/components/EffectBlockView";
import { translatePath, translateMastery, masteryKeyForPoints, masteryLevelForPoints, PATH_MASTERY_MILESTONES } from "@/lib/translate";
import { getRenameCost, getImmortalPenaltyRates } from "@/lib/balance";
import { GamePanel, GameTabs, GameButton, GameCard, GameProgressBar, GameBadge, GameSection, SearchableSelect } from "@/components/ui";
import { TribulationRealTimeInfo } from "./TribulationRealTimeInfo";
import { Num } from "./Num";
// UI migrated to shared components

function formatStat(val: number): string {
  return Math.abs(val) >= 10000
    ? formatNumber(val)
    : (val % 1 === 0 ? val.toString() : val.toFixed(2));
}

export type ApertureTab = "aperture" | "growth" | "referrals" | "settings";

interface ApertureProps {
  player: Player;
  guDictionary: Record<string, GuSpec>;
  initialTab?: ApertureTab;
  expandMmoSettings?: boolean;
  onRealtimeHotbarSaved?: () => void;
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
  10: "Vĩnh Sinh",
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
  10: "Hỗn Độn Tiên Nguyên",
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

// Owned growth type keyed by (gu_id, rank). Growth runs once per type, so we
// aggregate progress by type and keep it visible even when unequipped.
function collectOwnedGrowthTypes(player: Player, guDictionary: Record<string, GuSpec>): { gu_id: string; rank: number; growth_hours: number }[] {
  const map = new Map<string, { gu_id: string; rank: number; growth_hours: number }>();
  (player.gu_inventory || []).forEach((gu) => {
    const spec = guSpecFor(gu.gu_id, gu.rank);
    if (!spec || !growthConfig(spec)?.hours_to_max) return;
    const key = `${gu.gu_id}@${gu.rank}`;
    const gh = gu.growth_hours || 0;
    const existing = map.get(key);
    if (!existing || gh > existing.growth_hours) {
      map.set(key, { gu_id: gu.gu_id, rank: gu.rank, growth_hours: gh });
    }
  });
  return Array.from(map.values());
}

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
  initialTab = "aperture",
  expandMmoSettings = false,
  onRealtimeHotbarSaved,
}) => {
  const store = useGameStore();
  const constitutionDictionary = store.constitutionDictionary;
  const raceDictionary = store.raceDictionary;
  const statusDictionary = store.statusDictionary;
  const selectedTitleAchievement = store.achievements?.find((achievement: ApiResponseData) => achievement.id === store.selectedTitle);
  const selectedTitleEffect = selectedTitleAchievement?.title_effect as ApiResponseData | undefined;
  const manaSnapshot = store.manaSnapshot;
  const passiveStats = store.passiveStats;
  const fetchPassiveStats = store.fetchPassiveStats;
  const [manaNow, setManaNow] = useState(() => Date.now());

  useEffect(() => {
    void store.fetchAchievements();
  }, [store.fetchAchievements]);

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

  const [activeTab, setActiveTab] = useState<ApertureTab>(initialTab);
  const [isGrowthCoreExpanded, setIsGrowthCoreExpanded] = useState(true);
  const [isPassiveStatsExpanded, setIsPassiveStatsExpanded] = useState(true);
  const [isActiveGrowthExpanded, setIsActiveGrowthExpanded] = useState(true);
  const [isPathAttainmentsExpanded, setIsPathAttainmentsExpanded] = useState(true);
  const [giftCodeInput, setGiftCodeInput] = useState("");
  const [giftCodeMsg, setGiftCodeMsg] = useState<string | null>(null);
  const [giftCodeLoading, setGiftCodeLoading] = useState(false);
  const [bindRefInput, setBindRefInput] = useState("");

  useEffect(() => {
    if (activeTab !== "growth") return;
    fetchPassiveStats();
    // Growth accumulates on the server every 60s; poll so the progress bars
    // visibly advance instead of looking frozen.
    const interval = setInterval(() => {
      if (document.visibilityState === "visible") {
        fetchPassiveStats();
      }
    }, 30000);
    return () => clearInterval(interval);
  }, [activeTab, fetchPassiveStats]);

  // Disguise/stealth settings state
  const [isDisguised, setIsDisguised] = useState(player.is_disguised || false);
  const [fakeRank, setFakeRank] = useState(player.fake_rank || player.rank);
  const [fakeSubRank, setFakeSubRank] = useState(player.fake_sub_rank || player.sub_rank);
  const [fakeVitalGu, setFakeVitalGu] = useState(player.fake_vital_gu || "");
  const [settingsUsername, setSettingsUsername] = useState(player.username || "");

  useEffect(() => {
    setSettingsUsername(player.username || "");
  }, [player.username]);

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
    const promises: Promise<ApiResponseData>[] = [store.fetchReferrals()];
    if (isImmortal) {
      promises.push(store.fetchAperture(), store.fetchDaoMarks(), store.fetchActiveTribulation());
    }
    await Promise.all(promises);
    setIsReloading(false);
  };

  const aperture = store.aperture;
  const essenceProd = aperture?.essence_production_per_year ?? 10;
  const immortalRegen = essenceProd / 10.0;
  const mortalRegen = aptitudeType === "Giáp" ? 8 : aptitudeType === "Ất" ? 6 : aptitudeType === "Bính" ? 4 : 2;
  const regenRate = (isImmortal ? immortalRegen : mortalRegen).toFixed(2);

  // Calculate real-world hourly regen rates using TimeSpeedMultiplier
  const timeSpeedMultiplier = player.time_speed_multiplier || 1;
  const effectiveMultiplier = isImmortal ? timeSpeedMultiplier * (aperture?.time_rate ?? 10) : timeSpeedMultiplier;
  const hpRegenRealHourly = (player.hp_regen_rate || 0) * timeSpeedMultiplier;
  const manaRegenRealHourly = (player.mana_regen_rate || Number(regenRate)) * effectiveMultiplier;

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
  const vitalSpec = vitalGu ? guSpecFor(vitalGu.gu_id, vitalGu.rank) : null;

  // Equipped Gu (max 6)
  let equippedGus = guInventory.filter((g) => g.equipped && g.status !== "DEAD" && g.is_vital !== true);
  equippedGus = equippedGus.slice(0, 6);

  // Compute disguise permissions dynamically based on active equipped Gu effect mechanics
  const equippedActiveGus = guInventory.filter((g) => g.equipped && g.status === "ACTIVE");

  const checkMechanic = (actionCode: string) => {
    return equippedActiveGus.some((g) => {
      const spec = guSpecFor(g.gu_id, g.rank);
      return Boolean(spec && hasCapability(spec, actionCode));
    });
  };

  const hasConcealment = checkMechanic("DISGUISE") || equippedActiveGus.some((g) => {
    const spec = guSpecFor(g.gu_id, g.rank);
    return Boolean(spec && disguiseLevel(spec) > 0);
  });
  const hasAttitude = checkMechanic("PERFECT_DISGUISE");
  const hasFormChange = checkMechanic("CHANGE_FORM_DISGUISE");

  const hasFaceChange = hasFormChange || hasAttitude;

  React.useEffect(() => {
    setIsDisguised(Boolean(hasConcealment && player.is_disguised));
    setFakeRank(player.fake_rank || player.rank);
    setFakeSubRank(player.fake_sub_rank || player.sub_rank);
    setFakeVitalGu(player.fake_vital_gu || "");
  }, [hasConcealment, player.is_disguised, player.fake_rank, player.fake_sub_rank, player.fake_vital_gu, player.rank, player.sub_rank]);

  // Owned Gu IDs for fake Vital Gu select dropdown
  const ownedGuIds = Array.from(new Set(guInventory.map((g) => g.gu_id)));
  const fakeVitalGuOptions = useMemo(() => {
    return ownedGuIds.map((guId) => {
      const spec = guDictionary[guId];
      return {
        value: guId,
        label: spec?.name || guId,
        subLabel: guId,
        badge: spec?.rank ? `Rank ${spec.rank}` : undefined,
      };
    });
  }, [ownedGuIds, guDictionary]);

  // Orbit dot positions — evenly spaced, starting from top
  const orbitDots = equippedGus.map((g, i) => {
    const angle = (i / Math.max(equippedGus.length, 1)) * 2 * Math.PI - Math.PI / 2;
    return {
      g,
      spec: guSpecFor(g.gu_id, g.rank),
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
    <GamePanel variant="gold" className="space-y-4">
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
        className="flex flex-col items-center justify-center p-3 sm:p-5 glass-panel-gold rounded-3xl w-full border border-amber-500/20 transition-all duration-300 relative overflow-hidden group"
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
        <div className="flex gap-2 w-full mb-5 z-10 items-center">
          <GameButton
            onClick={handleReloadAll}
            disabled={isReloading}
            variant="ghost"
            size="sm"
            title="Tải lại dữ liệu"
          >
            <RefreshCw className={`w-4 h-4 ${isReloading ? "animate-spin" : ""}`} />
          </GameButton>
          <GameTabs
            items={[
              { id: "aperture", label: "Không Khiếu", disabled: false },
              { id: "growth", label: "Bản Thân", disabled: false },
              { id: "referrals", label: "Phúc Lợi", disabled: false },
              { id: "settings", label: "Hồ Sơ", disabled: false },
            ]}
            activeId={activeTab}
            onChange={(id) => setActiveTab(id as ApertureTab)}
            variant="pills"
            size="sm"
            scrollable={true}
            fullWidth={true}
          >
          </GameTabs>
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
              <div className="flex justify-center mt-2">
                <button
                  onClick={() => useGameStore.getState().unequipAllGu()}
                  disabled={equippedGus.filter(g => !g.is_vital).length === 0}
                  className="px-4 py-1.5 rounded-xl border text-[11px] font-bold transition-all disabled:opacity-50 disabled:cursor-not-allowed bg-red-950/30 text-red-400 border-red-500/30 hover:bg-red-900/40"
                >
                  Tháo Tất Cả Cổ Trùng
                </button>
              </div>
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
            <div className="bg-amber-950/15 border border-amber-500/20 rounded-2xl p-3 sm:p-4">
              <div className="flex items-center gap-2 text-amber-300 text-sm font-bold">
                <Award className="w-4 h-4" />
                <span>Hiệu Ứng Danh Hiệu Đang Trang Bị</span>
              </div>
              {selectedTitleAchievement && selectedTitleEffect ? (
                <div className="mt-2 text-xs text-zinc-300 space-y-1">
                  <div className="font-semibold text-zinc-100">{selectedTitleAchievement.name}</div>
                  <div>Tăng {selectedTitleEffect.pve_damage_pct}% sát thương lên quái loại {selectedTitleEffect.pve_monster_type}.</div>
                  {selectedTitleEffect.utility_pct > 0 && <div>Tăng {selectedTitleEffect.utility_pct}% sản lượng khi thu hoạch Tiên Khiếu.</div>}
                  <div className="text-[10px] text-zinc-500">Chỉ áp dụng PvE với đúng loại quái; trận đang chạy giữ snapshot khi bắt đầu.</div>
                </div>
              ) : (
                <div className="mt-2 text-xs text-zinc-500">Chưa trang bị danh hiệu có hiệu ứng.</div>
              )}
            </div>
            {/* Character Identity */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <User className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Thân Phận</span>
              </div>
              {(() => {
                const genderLabels: Record<string, string> = { male: "Nam", female: "Nữ" };
                const raceLabel = (key: string): string =>
                  raceDictionary?.[key]?.name || (key === "human" ? "Nhân Tộc" : key);
                const raceEffects = player.race ? raceDictionary?.[player.race]?.effects : undefined;
                const constitutionLabel = (key: string): string =>
                  constitutionDictionary?.[key]?.name || key;
                const constitutionEffects = player.constitution ? constitutionDictionary?.[player.constitution]?.effects : undefined;
                const bodyModLabels: Record<string, { name: string; desc: string }> = {
                  sovereign_immortal_body: { name: "Tiên Thể Chí Tôn", desc: "Miễn nhiễm xung đột Đạo Ngân." },
                };
                const cell = (label: string, value: string, accent = false) => (
                  <div className={`p-3 rounded-xl border ${accent ? "bg-zinc-950/40 border-zinc-800" : "bg-zinc-950/40 border-zinc-900"}`}>
                    <div className="text-[10px] text-zinc-500 font-mono">{label}</div>
                    <div className="text-base font-bold text-zinc-100 font-mono mt-0.5">{value}</div>
                  </div>
                );
                return (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    <div className="p-3 rounded-xl border bg-zinc-950/40 border-zinc-900">
                      <div className="text-[10px] text-zinc-500 font-mono">Chủng Tộc</div>
                      <div className="text-base font-bold text-zinc-100 font-mono mt-0.5">{raceLabel(player.race) || "Nhân Tộc"}</div>
                      {raceEffects && <EffectEnvelopeView effects={raceEffects} />}
                    </div>
                    {cell("Giới Tính", genderLabels[player.gender] || player.gender || "Không rõ")}
                    <div className="p-3 rounded-xl border bg-zinc-950/40 border-zinc-800">
                      <div className="text-[10px] text-zinc-500 font-mono">Thể Chất</div>
                      <div className="text-base font-bold text-zinc-100 font-mono mt-0.5">{constitutionLabel(player.constitution) || "Không có"}</div>
                      {constitutionEffects && <EffectEnvelopeView effects={constitutionEffects} />}
                    </div>
                    <div className="p-3 rounded-xl border bg-zinc-950/40 border-zinc-900">
                      <div className="text-[10px] text-zinc-500 font-mono">Trạng Thái</div>
                      {player.statuses && player.statuses.length > 0 ? (
                        <div className="flex flex-col gap-1.5 mt-1">
                          {player.statuses.map((s: PlayerStatusInstance) => {
                            let displayName = statusDictionary?.[s.id]?.name || s.id;
                            if (s.id.startsWith("cd_gu_")) {
                              const instanceId = s.id.substring(6);
                              const gu = player.gu_inventory?.find((g: ApiResponseData) => g.instance_id === instanceId);
                              const specName = gu ? guSpecFor(gu.gu_id, gu.rank)?.name || "Cổ Trùng" : "Cổ Trùng";
                              displayName = `Hồi chiêu: ${specName}`;
                            } else if (s.id.startsWith("cd_km_")) {
                              displayName = "Hồi chiêu Sát Chiêu";
                            } else if (s.id.startsWith("refine_insight_")) {
                              const r = s.id.substring("refine_insight_".length) || "?";
                              const stacks = Math.floor(s.value ?? 0);
                              displayName = `Tâm Đắc R${r} · ${stacks} tầng (+${stacks}% luyện R${r})`;
                            }
                            const statusEffects = statusDictionary?.[s.id]?.effects;
                            const regionalWallPct = s.id === "regional_wall_suppression" ? Math.round(Math.abs(s.value ?? 0) * 100) : null;
                            return (
                              <div key={s.id} className="flex flex-col gap-1">
                                <span className="text-[10px] px-2 py-0.5 rounded-lg bg-rose-950/40 text-rose-400 border border-rose-900/40 font-mono font-semibold w-fit">
                                  {displayName}{statusRemainLabel(s.expires_at)}
                                </span>
                                {regionalWallPct !== null ? (
                                  <div className="flex flex-col gap-0.5">
                                    <span className="text-[10px] text-rose-300/80">Công / Thủ / Tốc giảm {regionalWallPct}%</span>
                                    {s.source ? (
                                      <span className="text-[9px] text-zinc-400 font-mono">Lý do: {s.source}</span>
                                    ) : null}
                                  </div>
                                ) : statusEffects ? <EffectEnvelopeView effects={statusEffects} /> : null}
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="text-base font-bold text-zinc-100 font-mono mt-0.5">Bình thường</div>
                      )}
                    </div>
                    {player.body_mods && player.body_mods.length > 0 && (
                      <div className="p-3 rounded-xl border bg-amber-950/20 border-amber-500/30 col-span-2 sm:col-span-4">
                        <div className="text-[10px] text-amber-500/80 font-mono">Thân Thể</div>
                        <div className="flex flex-wrap gap-1.5 mt-1">
                          {player.body_mods.map((mod: string) => (
                            <span key={mod} title={bodyModLabels[mod]?.desc} className="text-xs px-2 py-0.5 rounded-lg bg-amber-900/40 text-amber-200 border border-amber-700/40 font-mono font-semibold">
                              {bodyModLabels[mod]?.name || mod}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>

            {/* Growth Boosts */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
              <button
                type="button"
                aria-expanded={isGrowthCoreExpanded}
                onClick={() => setIsGrowthCoreExpanded((expanded) => !expanded)}
                className="flex items-center gap-2 border-b border-zinc-900 pb-2.5 text-left"
              >
                {isGrowthCoreExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                <TrendingUp className="w-5 h-5 text-emerald-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cốt Tủy Không Khiếu</span>
              </button>
              {isGrowthCoreExpanded && (() => {
                const gd = player.growth_data;
                const b = gd?.boosts;
                const hasBoosts = b && (b.attack > 0 || b.defense > 0 || b.hp > 0 || b.speed > 0);
                if (!hasBoosts) return <div className="text-xs text-zinc-500 italic py-2">Chưa có tăng trưởng nào. Hãy trang bị Cổ trùng có hiệu ứng Trưởng Thành.</div>;
                return (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {b.attack > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Công Kích</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{(b.attack || 0).toFixed(2)}</div>
                      </div>
                    )}
                    {b.defense > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Phòng Ngự</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{(b.defense || 0).toFixed(2)}</div>
                      </div>
                    )}
                    {b.hp > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Máu Tối Đa</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{(b.hp || 0).toFixed(2)}</div>
                      </div>
                    )}
                    {b.speed > 0 && (
                      <div className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">Tốc Độ</div>
                        <div className="text-lg font-bold text-zinc-100 font-mono">+{(b.speed || 0).toFixed(2)}</div>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>

            {/* Passive Effects from Equipped Gu */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
              <button
                type="button"
                aria-expanded={isPassiveStatsExpanded}
                onClick={() => setIsPassiveStatsExpanded((expanded) => !expanded)}
                className="flex items-center gap-2 border-b border-zinc-900 pb-2.5 text-left"
              >
                {isPassiveStatsExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                <Zap className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Thống Kê Bị Động</span>
              </button>
              {isPassiveStatsExpanded && (() => {
                if (!passiveStats) return <div className="text-xs text-zinc-500 italic py-2">Đang tải...</div>;
                const labels: Record<string, string> = {
                  attack: "Công Kích", defense: "Phòng Ngự", hp: "Máu",
                  speed: "Tốc Độ", healing: "Hồi Phục", soul_refinement: "Hồn Lực",
                  storage: "Vật Chứa", movement: "Di Chuyển (ô)",
                  investigation: "Trinh Sát", hearing: "Thính Giác",
                  mana_regen_mult: "Hồi Pháp", damage_multiplier_on_hit: "Sát Thương Khi Mất Máu",
                  refine_mana_tier: "Tinh Luyện", cleansing_power: "Tẩy Trừ",
                  path_conflict_reduction: "Giảm Xung Đột (%)",
                };
                const details = passiveStats.details.filter(d => d.total !== 0);
                if (details.length === 0) return <div className="text-xs text-zinc-500 italic py-2">Không có chỉ số bị động nào.</div>;

                // Deduplicate by key to guarantee exactly 1 tile per stat
                const dedupedMap = new Map<string, (typeof details)[0]>();
                for (const d of details) {
                  const existing = dedupedMap.get(d.key);
                  if (existing) {
                    existing.total += d.total;
                    existing.sources = [...existing.sources, ...d.sources];
                  } else {
                    dedupedMap.set(d.key, { ...d, sources: [...d.sources] });
                  }
                }
                const dedupedDetails = Array.from(dedupedMap.values());

                return (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    {dedupedDetails.map(d => (
                      <div key={d.key} className="bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="text-[10px] text-zinc-500 font-mono">{labels[d.key] || d.key}</div>
                        <div className="text-base font-bold text-zinc-100 font-mono">+{formatStat(d.total)}</div>
                        {d.sources.length > 1 && (
                          <div className="mt-1 text-[9px] text-zinc-600 leading-tight">
                            {d.sources.map((s, i) => (
                              <div key={i}>{s.name}: +{formatStat(s.final)}</div>
                            ))}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                );
              })()}
            </div>

            {/* In-progress + Fully Grown Gu List */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner space-y-4">
              <button
                type="button"
                aria-expanded={isActiveGrowthExpanded}
                onClick={() => setIsActiveGrowthExpanded((expanded) => !expanded)}
                className="flex w-full items-center gap-2 border-b border-zinc-900 pb-2.5 text-left"
              >
                {isActiveGrowthExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                <Leaf className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cổ Trùng Đang Có Hiệu Ứng</span>
                <span className="ml-auto text-[10px] text-zinc-500 font-mono">{(() => {
                  const types = collectOwnedGrowthTypes(player, guDictionary);
                  const doneSet = new Set((player.growth_data?.fully_grown || []).map(fg => `${fg.gu_id}@${fg.rank}`));
                  return `${types.filter(t => !doneSet.has(`${t.gu_id}@${t.rank}`)).length} đang / ${types.length} loại`;
                })()}</span>
              </button>
              {isActiveGrowthExpanded && (() => {
                // 1 GU ID + rank chỉ tăng 1 lần: tiến độ gom theo loại (gu_id+rank)
                // và giữ vĩnh viễn, kể cả khi Cổ trùng đã được gỡ ra khỏi Không Khiếu.
                const fullyGrown = player.growth_data?.fully_grown || [];
                const doneSet = new Set(fullyGrown.map(fg => `${fg.gu_id}@${fg.rank}`));
                const ownedTypes = collectOwnedGrowthTypes(player, guDictionary);
                const growing = ownedTypes.filter(t => !doneSet.has(`${t.gu_id}@${t.rank}`));
                const hasAny = growing.length > 0 || fullyGrown.length > 0;
                if (!hasAny) return <div className="text-xs text-zinc-500 italic py-2">Chưa có Cổ trùng nào có hiệu ứng.</div>;
                return (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {growing.map(t => {
                      const spec = guSpecFor(t.gu_id, t.rank);
                      const g = spec ? growthConfig(spec) : null;
                      const maxH = g?.hours_to_max || 168;
                      const curH = t.growth_hours || 0;
                      const pct = Math.min(100, Math.round((curH / maxH) * 1000) / 10);
                      const parts: string[] = [];
                      if (g?.max_attack) parts.push(`Công +${g.max_attack}`);
                      if (g?.max_defense) parts.push(`Thủ +${g.max_defense}`);
                      if (g?.max_hp) parts.push(`Máu +${g.max_hp}`);
                      if (g?.max_speed) parts.push(`Tốc +${g.max_speed}`);
                      const isEquipped = (player.gu_inventory || []).some(x => x.gu_id === t.gu_id && x.rank === t.rank && x.equipped && x.status === "ACTIVE");
                      return (
                        <div key={`grow-${t.gu_id}-${t.rank}`} className="flex items-center gap-3 bg-zinc-950/40 p-3 rounded-xl border border-zinc-800">
                          <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center text-xs font-bold text-amber-500">
                            {(spec?.name || t.gu_id).slice(0, 2)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-xs font-bold text-zinc-200 font-mono truncate">{spec?.name || t.gu_id}</span>
                              {isEquipped && (
                                <span className="text-[9px] px-1.5 py-0.5 rounded-md bg-emerald-950/50 text-emerald-400 border border-emerald-900/50 font-mono shrink-0">Đang trang bị</span>
                              )}
                            </div>
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
                        <div key={`fg-${fg.gu_id}-${fg.rank}-${i}`} className="flex items-center gap-3 bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                          <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center text-xs font-bold text-zinc-400">
                            {(guSpecFor(fg.gu_id, fg.rank)?.name || fg.gu_id).slice(0, 2)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="text-xs font-bold text-zinc-200 font-mono truncate">{guSpecFor(fg.gu_id, fg.rank)?.name || fg.gu_id}</div>
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
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner space-y-4">
              <button
                type="button"
                aria-expanded={isPathAttainmentsExpanded}
                onClick={() => setIsPathAttainmentsExpanded((expanded) => !expanded)}
                className="flex w-full items-center gap-2 border-b border-zinc-900 pb-2.5 text-left"
              >
                {isPathAttainmentsExpanded ? <ChevronDown className="w-4 h-4 text-zinc-400" /> : <ChevronRight className="w-4 h-4 text-zinc-400" />}
                <Crown className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Cảnh Giới Lưu Phái</span>
              </button>
              {isPathAttainmentsExpanded && <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                {player.path_masteries && Object.keys(player.path_masteries).length > 0 ? (
                  Object.entries(player.path_masteries).map(([path, points]) => {
                    const level = masteryKeyForPoints(points as number);
                    return (
                      <div key={path} className="flex flex-col gap-1.5 bg-zinc-950/40 p-3 rounded-xl border border-zinc-900">
                        <div className="flex justify-between items-center">
                          <span className="text-zinc-300 font-mono text-xs font-bold capitalize">{translatePath(path)}</span>
                          <span className={`px-2 py-0.5 rounded text-[9px] font-bold font-mono ${level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                            level === "Great Grandmaster" ? "bg-red-950/50 text-red-300 border border-red-800/40" :
                              level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                                level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                                  "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                            }`}>
                            {translateMastery(level)}
                          </span>
                        </div>
                        <div className="text-[10px] text-zinc-500 leading-normal">
                          <span className="text-zinc-400 font-mono">{points} điểm</span>
                          {level === "Master" && " — Trực giác chiến đấu hệ, nhận +10% hiệu quả kỹ năng."}
                          {level === "Grandmaster" && " — Bỏ qua hình phạt giảm 30% sức mạnh khi sử dụng Cổ trùng hệ này."}
                          {level === "Great Grandmaster" && " — Cộng hưởng Thiên Địa Đạo Ngân, +50% hiệu ứng tại map cùng hệ."}
                          {level === "Supreme Grandmaster" && " — Cảnh giới Vô Thượng tối cao, kiểm soát tuyệt đối đại đạo."}
                          {level === "Ordinary" && " — Đạt hiểu biết phổ thông cơ bản về đạo pháp."}
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="col-span-2 text-center text-xs text-zinc-650 italic py-4">Chưa ngộ chân ý bất kỳ lưu phái nào. Hãy thám hiểm Mộng Cảnh!</div>
                )}
              </div>}
            </div>
          </div>
        )}

        {activeTab === "referrals" && (
          <div className="w-full z-10 space-y-5 text-left">
            {/* Master Referral Card */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
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

            {/* Bind Referral Code for Rank 1 players who haven't bound a referrer */}
            {player?.rank === 1 && store.referralsData?.has_referrer === false && (
              <div className="bg-amber-950/20 border border-amber-500/30 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex flex-col gap-0.5">
                  <span className="text-xs font-mono font-bold text-amber-400 flex items-center gap-1.5">
                    <UserPlus className="w-4 h-4" /> Bái Sư (Nhập Mã Giới Thiệu)
                  </span>
                  <span className="text-[10px] text-zinc-400 font-serif">
                    Dành riêng cho Cổ Sư Nhất Chuyển chưa có sư phụ hướng dẫn.
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder="Mã sư phụ..."
                    value={bindRefInput}
                    onChange={(e) => setBindRefInput(e.target.value)}
                    className="bg-zinc-950 border border-zinc-800 rounded-xl px-3 py-1.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-amber-500 w-36 uppercase"
                  />
                  <button
                    disabled={store.actionLoading || !bindRefInput.trim()}
                    onClick={async () => {
                      const res = await store.bindReferralCode(bindRefInput.trim());
                      if (res?.success) {
                        store.addToast(res.message || "Bái sư thành công!", "success");
                        setBindRefInput("");
                      } else {
                        store.addToast(res?.message || "Lỗi", "error");
                      }
                    }}
                    className="bg-amber-500 hover:bg-amber-400 text-zinc-950 text-xs font-mono font-bold px-3.5 py-1.5 rounded-xl transition-all cursor-pointer disabled:opacity-50 shrink-0"
                  >
                    Bái Sư
                  </button>
                </div>
              </div>
            )}

            {/* Gift Code Redeem Card */}
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
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
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4">
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
                      store.referralsData.referees.map((referee: {
                        id: string;
                        username: string;
                        rank: number;
                        is_suspicious: boolean;
                        has_deposited: boolean;
                        milestone1_rewarded: boolean;
                        milestone2_rewarded: boolean;
                        milestone3_rewarded: boolean;
                        milestone4_rewarded: boolean;
                        milestone5_rewarded: boolean;
                        is_verified: boolean;
                      }) => (
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
                              {/* Mốc 1: Tân Thủ */}
                              <div className={`flex items-center justify-between w-52 text-[10px] px-2 py-1 rounded ${referee.milestone1_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                <span>Mốc 1 (Tân Thủ):</span>
                                {referee.milestone1_rewarded ? (
                                  <span>Đã nhận 100 NT</span>
                                ) : referee.is_verified ? (
                                  <button onClick={() => store.claimReferralReward(referee.id, 1)} disabled={store.actionLoading} className="bg-emerald-600/80 hover:bg-emerald-500 text-white px-2 py-0.5 rounded cursor-pointer disabled:opacity-50 transition-colors">Nhận 100 NT</button>
                                ) : (
                                  <span>Chưa đạt</span>
                                )}
                              </div>
                              {/* Mốc 2: Nhị Chuyển */}
                              <div className={`flex items-center justify-between w-52 text-[10px] px-2 py-1 rounded ${referee.milestone2_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                <span>Mốc 2 (Nhị Chuyển):</span>
                                {referee.milestone2_rewarded ? (
                                  <span>Đã nhận 200 NT</span>
                                ) : (referee.is_verified && referee.rank >= 2) ? (
                                  <button onClick={() => store.claimReferralReward(referee.id, 2)} disabled={store.actionLoading} className="bg-emerald-600/80 hover:bg-emerald-500 text-white px-2 py-0.5 rounded cursor-pointer disabled:opacity-50 transition-colors">Nhận 200 NT</button>
                                ) : (
                                  <span>Chưa đạt</span>
                                )}
                              </div>
                              {/* Mốc 3: Tam Chuyển */}
                              <div className={`flex items-center justify-between w-52 text-[10px] px-2 py-1 rounded ${referee.milestone3_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                <span>Mốc 3 (Tam Chuyển):</span>
                                {referee.milestone3_rewarded ? (
                                  <span>Đã nhận Cổ trùng</span>
                                ) : (referee.is_verified && referee.rank >= 3) ? (
                                  <button onClick={() => store.claimReferralReward(referee.id, 3)} disabled={store.actionLoading} className="bg-emerald-600/80 hover:bg-emerald-500 text-white px-2 py-0.5 rounded cursor-pointer disabled:opacity-50 transition-colors">Nhận Cổ trùng</button>
                                ) : (
                                  <span>Chưa đạt</span>
                                )}
                              </div>
                              {/* Mốc 4: Tứ Chuyển */}
                              <div className={`flex items-center justify-between w-52 text-[10px] px-2 py-1 rounded ${referee.milestone4_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                <span>Mốc 4 (Tứ Chuyển):</span>
                                {referee.milestone4_rewarded ? (
                                  <span>Đã nhận 500 NT</span>
                                ) : (referee.is_verified && referee.rank >= 4) ? (
                                  <button onClick={() => store.claimReferralReward(referee.id, 4)} disabled={store.actionLoading} className="bg-emerald-600/80 hover:bg-emerald-500 text-white px-2 py-0.5 rounded cursor-pointer disabled:opacity-50 transition-colors">Nhận 500 NT</button>
                                ) : (
                                  <span>Chưa đạt</span>
                                )}
                              </div>
                              {/* Mốc 5: Nạp Thẻ */}
                              <div className={`flex items-center justify-between w-52 text-[10px] px-2 py-1 rounded ${referee.milestone5_rewarded ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : referee.has_deposited ? 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20' : 'bg-zinc-900 text-zinc-500'}`}>
                                <span>Mốc 5 (Nạp thẻ):</span>
                                {referee.milestone5_rewarded ? (
                                  <span>Đã nhận (5%)</span>
                                ) : (referee.is_verified && referee.has_deposited) ? (
                                  <button onClick={() => store.claimReferralReward(referee.id, 5)} disabled={store.actionLoading} className="bg-yellow-600/80 hover:bg-yellow-500 text-white px-2 py-0.5 rounded cursor-pointer disabled:opacity-50 transition-colors">Nhận (5%)</button>
                                ) : referee.has_deposited ? (
                                  <span>Chờ verified...</span>
                                ) : (
                                  <span>Chưa nạp</span>
                                )}
                              </div>
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
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-3 sm:p-5 shadow-inner flex flex-col gap-4 text-left">
              <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
                <Crown className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Thiết Lập Hồ Sơ</span>
              </div>

              <form onSubmit={async (e) => {
                e.preventDefault();
                const usernameInput = (e.currentTarget.elements.namedItem("username") as HTMLInputElement).value;
                const bioInput = (e.currentTarget.elements.namedItem("bio") as HTMLTextAreaElement).value;
                const newPasswordInput = (e.currentTarget.elements.namedItem("newPassword") as HTMLInputElement).value;

                const isChangingName = usernameInput.trim() !== "" && usernameInput.trim() !== player.username;
                if (isChangingName && player.rank >= 6) {
                  const penalty = getImmortalPenaltyRates(player.rank);
                  const confirmMsg = `⚠️ CẢNH BÁO NGHỊCH THIÊN:\n\nĐạo hữu đang đổi đạo hiệu từ "${player.username}" thành "${usernameInput.trim()}".\n\nỞ cảnh giới Cổ Tiên ${player.rank} Chuyển, hành động này sẽ chịu đại đạo phản phệ cực lớn:\n- Khấu trừ Tiên Nguyên Thạch\n- Tụt ${penalty.lossPercent}% toàn bộ Đạo Ngân\n- Tụt ${penalty.lossPercent}% Điểm Chân Ý Lưu Phái\n- Tổn hại Tiên Khiếu\n- Phát thông cáo toàn Server\n\nĐạo hữu có tuyệt đối chắc chắn muốn tiến hành?`;
                  if (!window.confirm(confirmMsg)) {
                    return;
                  }
                }

                const success = await store.updateProfileSettings({
                  username: usernameInput,
                  bio: bioInput,
                  new_password: newPasswordInput || undefined,
                  is_disguised: Boolean(hasConcealment && isDisguised),
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
                  <div className="flex items-center justify-between">
                    <label htmlFor="settings-username" className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">Đạo Hiệu</label>
                    <span className="text-[10px] font-mono text-zinc-500">
                      {player.rank <= 2 ? "Miễn phí đổi tên" : player.rank <= 5 ? "Tốn Nguyên Thạch" : "Tốn Tiên Thạch & Đại Đạo Phản Phệ"}
                    </span>
                  </div>
                  <input
                    id="settings-username"
                    name="username"
                    type="text"
                    value={settingsUsername}
                    onChange={(e) => setSettingsUsername(e.target.value)}
                    required
                    className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-850 rounded-xl px-3.5 py-2 text-xs text-zinc-200 placeholder-zinc-700 focus:outline-none transition-all font-mono"
                  />
                  <p className="text-[9px] text-zinc-600">Đạo hiệu được so sánh không phân biệt chữ hoa/thường.</p>
                  {/* Notice / Warning box if name is being changed */}
                  {settingsUsername.trim() !== "" && settingsUsername.trim().toLowerCase() !== player.username.toLowerCase() && (() => {
                    const renameCost = getRenameCost(player.rank);
                    const penalty = getImmortalPenaltyRates(player.rank);
                    return (
                      <div className={`p-2.5 rounded-xl border text-[11px] leading-relaxed flex flex-col gap-1 ${
                        player.rank <= 2
                          ? "bg-emerald-950/30 border-emerald-900/50 text-emerald-300"
                          : player.rank <= 5
                          ? "bg-amber-950/30 border-amber-900/50 text-amber-300"
                          : "bg-red-950/40 border-red-800/60 text-red-200"
                      }`}>
                        <div className="flex items-center gap-1.5 font-bold">
                          {player.rank >= 6 ? (
                            <AlertTriangle className="w-3.5 h-3.5 text-red-400 shrink-0" />
                          ) : (
                            <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" />
                          )}
                          <span>
                            {player.rank <= 5
                              ? renameCost.label
                              : `CẢNH BÁO NGHỊCH THIÊN (Cổ Tiên ${player.rank} Chuyển)`}
                          </span>
                        </div>
                        <div className="text-[10px] text-zinc-300">
                          {player.rank <= 2 && "Đạo hữu ở cảnh giới Nhất / Nhị Chuyển có thể tự do đổi tên mà không chịu chi phí hay phản phệ."}
                          {player.rank >= 3 && player.rank <= 5 && `Đổi tên sẽ khấu trừ ${renameCost.amount.toLocaleString("vi-VN")} Nguyên Thạch (Hiện có: ${Math.floor(player.primeval_stones).toLocaleString()} NT).`}
                          {player.rank >= 6 && (
                            <div className="space-y-0.5 text-red-300 font-mono">
                              <p>• Tiêu hao: {renameCost.amount.toLocaleString("vi-VN")} Tiên Nguyên Thạch (Hiện có: {Math.floor(player.immortal_stones).toLocaleString()} TNT).</p>
                              <p>• Phản phệ: Tụt <span className="font-bold text-red-400">{penalty.lossPercent}% toàn bộ Đạo Ngân</span> trong Tiên Khiếu.</p>
                              <p>• Phản phệ: Tụt <span className="font-bold text-red-400">{penalty.lossPercent}% Điểm Chân Ý</span> trên tất cả Cảnh Giới Lưu Phái.</p>
                              <p>• Chấn động: Tổn hại {penalty.hpLossPercent}% HP Tiên Khiếu và tăng +{penalty.stressGain} Áp Lực Tiên Khiếu.</p>
                              <p>• Thiên Đạo Biến Động: Phát thông cáo chấn động toàn Server trên kênh Thế Giới.</p>
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })()}
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
                              * Chỉ giả yếu (&le; {RANK_NAMES[player.rank]}). Cần hiệu ứng Ngụy Trang Hoàn Mỹ để giả mạnh!
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
                            <span className="text-[8px] bg-rose-950/40 text-rose-400 border border-rose-900/30 px-1.5 py-0.5 rounded font-mono font-semibold uppercase">Yêu cầu hiệu ứng Biến Hóa / Ngụy Trang Hoàn Mỹ</span>
                          )}
                        </div>
                        <SearchableSelect
                          id="settings-fakeVitalGu"
                          disabled={!hasFaceChange}
                          placeholder="-- Mặc định (Không ngụy tạo) --"
                          searchPlaceholder="Tìm kiếm tên Cổ hoặc mã..."
                          options={fakeVitalGuOptions}
                          value={fakeVitalGu}
                          onChange={setFakeVitalGu}
                          clearable={true}
                        />
                      </div>
                    </div>
                  )}
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
                    <span>Lưu Hồ Sơ</span>
                  )}
                </button>
              </form>

              {/* Dọn dẹp bộ nhớ đệm cục bộ */}
              <div className="bg-zinc-950/40 border border-zinc-800/80 rounded-2xl p-4 sm:p-5 flex flex-col gap-3">
                <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-2.5">
                  <Trash2 className="w-4 h-4 text-amber-400" />
                  <span className="text-xs font-mono font-bold text-zinc-200 uppercase tracking-wider">
                    Bộ Nhớ Đệm & Dữ Liệu Cục Bộ
                  </span>
                </div>
                <p className="text-xs text-zinc-400 leading-relaxed">
                  Nếu bạn gặp tình trạng kẹt tải dữ liệu, hiển thị sai thông số, hoặc không nhìn thấy cập nhật mới, hãy dọn dẹp bộ nhớ đệm cục bộ (IndexedDB, Cache Storage, LocalStorage).
                </p>
                <div className="flex items-center gap-2 pt-1">
                  <ClearCacheButton showLabel={true} className="!w-full !py-2.5 !justify-center !text-amber-400 !bg-amber-500/10 hover:!bg-amber-500/20 !border-amber-500/30 font-bold" />
                </div>
              </div>
            </div>
          </div>
        )}
</div>
    </GamePanel>
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
        <AscensionRitual />

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
            Luyện Hóa Nguyên Thạch
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
              Luyện Hóa
            </button>
          </div>
        </div>
      </div>

      <div className="text-[10px] text-zinc-500 text-center font-mono font-semibold">
        Tiến độ phá vách khiếu: <span className="text-emerald-400 font-bold">{(Math.min(1.0, Math.max(0, player.wall_progress ?? 0)) * 100).toFixed(1)}%</span>
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

// ── Nghi thức Thăng Tiên: Nát Khiếu → Dẫn Khí → Thả Cổ ──

const ASCENSION_GRADE_TIERS = [
  { grade: "Sơ", qi: 100000, color: "text-slate-300" },
  { grade: "Trung", qi: 230000, color: "text-emerald-400" },
  { grade: "Thượng", qi: 370000, color: "text-cyan-400" },
  { grade: "Chí Cao", qi: 480000, color: "text-amber-400" },
];

const ASCENSION_SHATTER_BASE = 15; // % vỡ khiếu lần đầu
const ASCENSION_SHATTER_STEP = 15; // % cộng dồn mỗi lần hụt
const ASCENSION_SHATTER_MAX = 90;
const ASCENSION_SHATTER_COST_RATIO = 0.30; // 30% chân nguyên tối đa mỗi lần

const ASCENSION_TEMPER_STAGES: { title: string; options: { text: string; soul?: number; manaPct?: number; stones?: number }[] }[] = [
  {
    title: "Định Tâm",
    options: [
      { text: "Tĩnh tọa điều tức, lấy thần thức áp chế tâm ma (−15 Soul HP)", soul: 15 },
      { text: "Vận chân nguyên hòa giải 2 khí thiên địa (−25% Chân Nguyên tối đa)", manaPct: 0.25 },
    ],
  },
  {
    title: "Ký Địa",
    options: [
      { text: "Mở huyệt Long Không, dẫn địa mạch nhập khiếu (−25 Soul HP)", soul: 25 },
      { text: "Đốt nguyên thạch trấn yểm địa mạch (−2.000 Nguyên Thạch)", stones: 2000 },
    ],
  },
  {
    title: "Thông Thiên",
    options: [
      { text: "Ngửa mặt đón thiên khí, lấy Nhân Khí làm mồi (−35 Soul HP)", soul: 35 },
      { text: "Dốc toàn chân nguyên đón tiếp 2 khí thiên địa (−40% Chân Nguyên tối đa)", manaPct: 0.40 },
    ],
  },
];

function AscensionRitual() {
  const store = useGameStore();
  const player = store.player;
  const session = store.ascensionSession;
  const [selectedGu, setSelectedGu] = useState<string[]>([]);
  const [, forceTick] = useState(0);

  useEffect(() => {
    store.fetchAscensionSession();
    const t = setInterval(() => forceTick((n) => n + 1), 1000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (session?.step !== "CALAMITY") return;
    const t = setInterval(() => store.fetchAscensionSession(), 3000);
    return () => clearInterval(t);
  }, [session?.step]);

  if (!player) return null;

  const humanQi = player.human_qi || 0;
  const hasVital = !!(player.gu_inventory ?? []).some((g) => g.is_vital && g.status !== "DEAD");
  const maxMana = player.max_mana ?? 0;
  const shatterCost = Math.max(500, maxMana * ASCENSION_SHATTER_COST_RATIO);
  const sessionStep = session?.step;

  // Hiệu lực phiên còn lại (đếm ngược).
  let deadlineLeft: string | null = null;
  if (session?.expires_at && ["GUID_QI", "CALAMITY", "RELEASE"].includes(sessionStep ?? "")) {
    const ms = new Date(session.expires_at).getTime() - Date.now();
    if (ms <= 0) deadlineLeft = "Hết hạn!";
    else {
      const h = Math.floor(ms / 3600000);
      const m = Math.floor((ms % 3600000) / 60000);
      const s = Math.floor((ms % 60000) / 1000);
      deadlineLeft = `${h}h ${m}p ${s}s`;
    }
  }

  const stepIndex = sessionStep === "GUID_QI" ? 1 : sessionStep === "CALAMITY" ? 2 : sessionStep === "RELEASE" ? 3 : 0;

  // Pool thả cổ: cổ 5C (không phải bản mệnh) của lưu phái đạt Đại Sư.
  const releasePool = (player.gu_inventory ?? []).filter((g) => {
    if (g.rank !== 5 || g.is_vital || g.status === "DEAD") return false;
    const spec = guSpecFor(g.gu_id, g.rank);
    if (!spec) return false;
    const paths: string[] = Array.isArray(spec.path) ? spec.path : [spec.path];
    return paths.some((p) => masteryLevelForPoints(player.path_masteries?.[translateMasteryPathKey(p)] ?? 0) >= 1);
  });

  const temperStage = session?.temper_stage ?? 0;
  const inTemper = sessionStep === "GUID_QI" && temperStage < ASCENSION_TEMPER_STAGES.length;

  return (
    <div className="w-full flex flex-col items-center justify-center p-4 border border-rose-950 bg-rose-950/20 rounded-2xl shadow-[inset_0_0_12px_rgba(244,63,94,0.1)]">
      <div className="text-rose-400 font-bold mb-1 text-center text-sm tracking-wide uppercase flex items-center gap-1.5">
        <ShieldAlert className="w-4 h-4 text-rose-500 animate-bounce" />
        Nghi Thức Thăng Tiên
      </div>
      <div className="text-[10.5px] text-zinc-400 mb-3 text-center leading-relaxed max-w-md">
        Nát Khiếu → Dẫn Khí → Thả Cổ. Nhân Khí hiện tại: <span className="font-bold text-amber-400 font-mono">{humanQi.toLocaleString("vi-VN", { maximumFractionDigits: 0 })}</span>
        {" "}— dưới <span className="font-mono text-rose-400">100.000</span> thì Dẫn Khí tất thất bại.
        {deadlineLeft && <> Thời hạn phiên: <span className="font-mono text-rose-300 font-bold">{deadlineLeft}</span></>}
      </div>

      {/* Chuỗi khâu */}
      <div className="w-full max-w-md grid grid-cols-4 gap-1 mb-3 text-center">
        {["Nát Khiếu", "Dẫn Khí", "Địa Tai", "Thả Cổ"].map((label, i) => {
          const active = sessionStep
            ? (sessionStep === "SHATTERING" && i === 0) ||
              (sessionStep === "GUID_QI" && i === 1) ||
              (sessionStep === "CALAMITY" && i === 2) ||
              (sessionStep === "RELEASE" && i === 3)
            : false;
          const done = i < stepIndex;
          return (
            <div key={label} className={`rounded-lg py-1.5 text-[10px] font-bold border ${done ? "border-emerald-600/50 bg-emerald-950/30 text-emerald-400" : active ? "border-rose-600/60 bg-rose-950/40 text-rose-300 animate-pulse" : "border-zinc-800 bg-zinc-950/50 text-zinc-500"}`}>
              {done ? "✓ " : ""}{label}
            </div>
          );
        })}
      </div>

      {/* ── Khâu 0/1: chuẩn bị + nát khiếu ── */}
      {(!session || sessionStep === "SHATTERING" || sessionStep === "FAILED" || sessionStep === "COMPLETED") && (
        <div className="w-full max-w-md space-y-2">
          <div className="w-full bg-zinc-950/70 border border-rose-900/40 rounded-xl p-2.5 text-xs space-y-1 font-mono">
            <div className="flex justify-between text-zinc-400">
              <span>Cổ Bản Mệnh (bắt buộc):</span>
              {hasVital ? <span className="text-emerald-400 font-bold">✓ Có</span> : <span className="text-rose-400 font-bold">✗ Thiếu</span>}
            </div>
            {ASCENSION_GRADE_TIERS.map((t) => (
              <div key={t.grade} className="flex justify-between text-zinc-400">
                <span>Phúc Địa {t.grade}:</span>
                <span className={`${humanQi >= t.qi ? t.color : "text-zinc-500"} font-bold`}>{t.qi.toLocaleString("vi-VN")} Nhân Khí{humanQi >= t.qi ? " ✓" : ""}</span>
              </div>
            ))}
            {session?.step === "SHATTERING" && (
              <div className="flex justify-between text-zinc-400">
                <span>Số lần đã trùng kích:</span>
                <span className="text-rose-300 font-bold">{session.attempts} (tỉ lệ vỡ kế tiếp {Math.min(ASCENSION_SHATTER_MAX, ASCENSION_SHATTER_BASE + ASCENSION_SHATTER_STEP * session.attempts)}%)</span>
              </div>
            )}
          </div>
          {!hasVital && (
            <div className="text-[10px] text-rose-400 text-center">
              Phải trang bị Cổ Bản Mệnh để trấn giữ không khiếu trước khi liều mạng nát khiếu!
            </div>
          )}
          <button
            disabled={!hasVital || player.current_mana < shatterCost || store.actionLoading}
            className="w-full px-5 py-2.5 bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 disabled:from-zinc-800 disabled:to-zinc-800 disabled:text-zinc-500 text-white font-bold text-xs rounded-xl shadow-[0_0_15px_rgba(244,3,94,0.45)] cursor-pointer disabled:cursor-not-allowed transition-all active:scale-95 uppercase tracking-wider"
            onClick={async () => {
              const attempts = session?.step === "SHATTERING" ? session.attempts : 0;
              const chance = Math.min(ASCENSION_SHATTER_MAX, ASCENSION_SHATTER_BASE + ASCENSION_SHATTER_STEP * attempts);
              if (!(await store.showConfirm(`Trùng kích nát khiếu tốn ${Math.round(shatterCost).toLocaleString("vi-VN")} Chân Nguyên, tỉ lệ vỡ hiện tại ${chance}%. Nát khiếu là mở đầu nghi thức cửu tử nhất sinh — ngươi chắc chứ?`))) return;
              const res = await store.shatterAperture();
              if (res) store.addToast(res.message, res.shattered ? "success" : "error");
            }}
          >
            Trùng Kích Nát Khiếu (−{Math.round(shatterCost).toLocaleString("vi-VN")} Chân Nguyên)
          </button>
        </div>
      )}

      {/* ── Khâu 2a: Dẫn Khí — khảo nghiệm tâm tính ── */}
      {inTemper && (
        <div className="w-full max-w-md space-y-2">
          <div className="text-[11px] text-zinc-300 font-bold text-center">
            Khảo nghiệm tâm tính {temperStage + 1}/{ASCENSION_TEMPER_STAGES.length}: {ASCENSION_TEMPER_STAGES[temperStage].title}
          </div>
          {ASCENSION_TEMPER_STAGES[temperStage].options.map((opt, i) => {
            const affordable =
              (opt.soul ?? 0) <= (player.soul_hp ?? 0) &&
              (opt.manaPct ?? 0) * maxMana <= player.current_mana &&
              (opt.stones ?? 0) <= (player.primeval_stones ?? 0);
            return (
              <button
                key={i}
                disabled={!affordable || store.actionLoading}
                className="w-full text-left px-3 py-2 bg-zinc-950/70 hover:bg-zinc-900 border border-zinc-800 hover:border-rose-700/60 disabled:opacity-40 disabled:cursor-not-allowed rounded-xl text-[11px] text-zinc-300 transition-all cursor-pointer disabled:cursor-not-allowed"
                onClick={async () => {
                  const res = await store.temperAscension(i);
                  if (res) store.addToast(res.message, res.success ? "success" : "error");
                }}
              >
                {opt.text}
                {!affordable && <span className="text-rose-500"> (không đủ tài nguyên)</span>}
              </button>
            );
          })}
        </div>
      )}

      {/* ── Khâu 2b: Địa Tai bản yếu ── */}
      {sessionStep === "CALAMITY" && (
        <div className="w-full max-w-md space-y-2 text-center">
          <div className="text-[11px] text-rose-300 font-bold animate-pulse">
            ĐỊA TAI BẢN YẾU giáng lâm! 2 khí thiên địa đang cuồn cuộn chờ thân chủ điều hòa.
          </div>
          <button
            disabled={store.actionLoading}
            className="w-full px-5 py-2.5 bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 disabled:from-zinc-800 disabled:to-zinc-800 disabled:text-zinc-500 text-white font-bold text-xs rounded-xl transition-all active:scale-95 uppercase tracking-wider cursor-pointer"
            onClick={async () => {
              const res = await store.engageAscensionCalamity();
              if (res) store.addToast(res.message, "info");
            }}
          >
            Nghênh Chiến Địa Tai Bản Yếu
          </button>
          <div className="text-[10px] text-zinc-500">Thua trận = nghi thức thất bại, mất một phần Nhân Khí. Rút lui an toàn được phép, có thể nghênh chiến lại trong thời hạn.</div>
        </div>
      )}

      {/* ── Khâu 3: Thả Cổ ── */}
      {sessionStep === "RELEASE" && (
        <div className="w-full max-w-md space-y-2">
          <div className="text-[11px] text-zinc-300 text-center">
            Chọn cổ <span className="font-bold text-amber-400">Ngũ Chuyển</span> của lưu phái đạt <span className="font-bold text-emerald-400">Đại Sư</span> để thả vào khiếu — mỗi cổ tiêu hủy nhận <span className="font-mono text-amber-400">300 Đạo Ngân</span> theo lưu phái. Cổ bị hủy vĩnh viễn!
          </div>
          {releasePool.length === 0 && (
            <div className="text-[10px] text-rose-400 text-center">Không có cổ 5C nào thuộc lưu phái đạt Đại Sư — chuẩn bị thiếu trầm trọng!</div>
          )}
          <div className="max-h-40 overflow-y-auto space-y-1">
            {releasePool.map((g) => {
              const spec = guSpecFor(g.gu_id, g.rank);
              const paths: string[] = spec ? (Array.isArray(spec.path) ? spec.path : [spec.path]) : [];
              const chosen = selectedGu.includes(g.instance_id);
              return (
                <button
                  key={g.instance_id}
                  className={`w-full text-left px-3 py-1.5 rounded-lg border text-[11px] transition-all cursor-pointer ${chosen ? "border-amber-500/60 bg-amber-950/30 text-amber-300" : "border-zinc-800 bg-zinc-950/60 text-zinc-300 hover:border-zinc-700"}`}
                  onClick={() => setSelectedGu((cur) => chosen ? cur.filter((id) => id !== g.instance_id) : [...cur, g.instance_id])}
                >
                  {chosen ? "☑" : "☐"} {spec?.name || g.gu_id} — {paths.map((p) => translatePath(p)).join(", ")} (+300 Đạo Ngân)
                </button>
              );
            })}
          </div>
          <button
            disabled={selectedGu.length === 0 || store.actionLoading}
            className="w-full px-5 py-2.5 bg-gradient-to-r from-amber-600 to-yellow-500 hover:from-amber-500 hover:to-yellow-400 disabled:from-zinc-800 disabled:to-zinc-800 disabled:text-zinc-500 text-zinc-950 font-black text-xs rounded-xl transition-all active:scale-95 uppercase tracking-wider cursor-pointer disabled:cursor-not-allowed"
            onClick={async () => {
              const totalMarks = selectedGu.length * 300;
              if (!(await store.showConfirm(`Thả ${selectedGu.length} cổ 5C để tiêu hủy, nhận ${totalMarks} Đạo Ngân và THĂNG TIÊN? Hành động không thể đảo ngược!`))) return;
              const res = await store.releaseAscensionGu(selectedGu);
              if (res) {
                store.addToast(res.message, res.success ? "success" : "error");
                if (res.success) setSelectedGu([]);
              }
            }}
          >
            Thả Cổ — Hoàn Thành Thăng Tiên ({selectedGu.length} cổ · {selectedGu.length * 300} Đạo Ngân)
          </button>
        </div>
      )}

      {sessionStep === "FAILED" && (
        <div className="text-[11px] text-rose-400 text-center">
          Nghi thức thất bại! Đã mất một phần Nhân Khí và toàn bộ chuẩn bị. Có thể liều mạng nát khiếu lại bất cứ lúc nào.
        </div>
      )}
    </div>
  );
}

// Chuyển "Fire Path" → key "fire" để tra path_masteries.
function translateMasteryPathKey(path: string): string {
  return path.toLowerCase().replace(/\s+path$/i, "").trim();
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
                <span className="text-zinc-400 font-medium">Lưu tốc tiên khiếu:</span>
                <span className="text-cyan-400 font-semibold">x{aperture.time_rate} lần</span>
              </div>
              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-400 font-medium">Lưu tốc bản thân:</span>
                <span className="text-violet-400 font-semibold">x{(player.time_speed_multiplier || 1).toFixed(2)} lần</span>
              </div>

              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-400 font-medium">Tiên Nguyên trong khiếu:</span>
                <span className="text-yellow-400 font-semibold">{Math.floor(player.current_mana)} hạt</span>
              </div>

              <div className="space-y-1">
                <div className="flex justify-between text-[10px] text-zinc-500 font-mono">
                  <span>
                    {player.race === "cuong_thi"
                      ? "Chu kỳ Kiếp nạn (Tử Khiếu - Ngưng trệ)"
                      : player.rank >= 10
                        ? "Vĩnh Sinh (Bất diệt - Không còn tai kiếp)"
                        : `Chu kỳ Kiếp nạn (${Math.floor(aperture.internal_years_passed)} / ${Math.round(aperture.next_tribulation_year)} năm)`}
                  </span>
                  <span>{player.race === "cuong_thi" || player.rank >= 10 ? "0%" : `${calculateTribulationProgress(aperture, player.race).toFixed(1)}%`}</span>
                </div>
                <div className="w-full bg-zinc-950 h-1.5 rounded-full overflow-hidden border border-zinc-900">
                  <div
                    className="h-full bg-gradient-to-r from-cyan-600 to-amber-500"
                    style={{ width: `${calculateTribulationProgress(aperture, player.race)}%` }}
                  />
                </div>
                <TribulationRealTimeInfo aperture={aperture} timeSpeedMultiplier={player.time_speed_multiplier || 1} isCompact={true} />
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
              Luyện Hóa Tiên Nguyên Thạch
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
                Luyện hóa
              </button>
            </div>
          </div>
        </div>

        {/* Dao Marks summary */}
        <div className="border-t border-zinc-900 pt-3">
          <div className="text-[9px] text-zinc-550 text-center mb-2 uppercase tracking-widest font-bold font-mono">Đạo Ngân Tích Lũy</div>
          <div className="flex flex-wrap gap-2 justify-center">
            {displayMarks.map((mark: ApiResponseData, idx: number) => (
              <div key={idx} className="px-2.5 py-1 bg-zinc-950/80 border border-zinc-900 rounded-xl text-[10px] font-mono">
                <span className="text-amber-500 font-bold uppercase">{translatePath(mark.path)}</span>: <span className="text-zinc-200 font-bold"><Num value={Number(mark.amount) || 0} /></span>
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
              Object.entries(player.path_masteries).map(([path, points]) => {
                const pts = points as number;
                const level = masteryKeyForPoints(pts);
                const levelIdx = masteryLevelForPoints(pts);
                let currentMin = 0;
                let nextMax = PATH_MASTERY_MILESTONES[0];
                if (levelIdx < PATH_MASTERY_MILESTONES.length) {
                  nextMax = PATH_MASTERY_MILESTONES[levelIdx];
                  if (levelIdx > 0) currentMin = PATH_MASTERY_MILESTONES[levelIdx - 1];
                } else {
                  currentMin = PATH_MASTERY_MILESTONES[PATH_MASTERY_MILESTONES.length - 1];
                  nextMax = currentMin;
                }
                const progress = nextMax > currentMin ? Math.min(100, Math.max(0, ((pts - currentMin) / (nextMax - currentMin)) * 100)) : 100;

                return (
                  <div key={path} className="flex flex-col gap-1.5 bg-zinc-950/30 px-2.5 py-1.5 rounded-lg border border-zinc-900/50 text-xs">
                    <div className="flex justify-between items-center">
                      <div className="flex items-center gap-1.5">
                        <span className="text-zinc-400 font-medium">{translatePath(path)}</span>
                        <span className="text-[10px] font-mono text-zinc-500">({pts})</span>
                      </div>
                      <span className={`px-2 py-0.5 rounded text-[10px] font-semibold font-mono ${level === "Supreme Grandmaster" ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" :
                        level === "Great Grandmaster" ? "bg-red-950/50 text-red-300 border border-red-800/40" :
                          level === "Grandmaster" ? "bg-amber-950/50 text-amber-300 border border-amber-800/40" :
                            level === "Master" ? "bg-blue-950/50 text-blue-300 border border-blue-800/40" :
                              "bg-zinc-800/50 text-zinc-400 border border-zinc-700/40"
                        }`}>
                        {translateMastery(level)}
                      </span>
                    </div>
                    {nextMax > currentMin && (
                      <div className="w-full h-1 bg-zinc-900 rounded-full overflow-hidden">
                        <div className="h-full bg-indigo-500 transition-all duration-300" style={{ width: `${progress}%` }} />
                      </div>
                    )}
                  </div>
                );
              })
            ) : (
              <div className="text-[10px] text-zinc-655 italic">Chưa có lĩnh ngộ lưu phái đặc thù.</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
