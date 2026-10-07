import { operationsBy, numericValue, type EffectEnvelope } from "@/lib/effect-blocks";

export type EffectBadge = { key: string; label: string; className: string };

export const rankLabel = (r: number) => {
  const names = ["Phàm Nhân", "Nhất Chuyển", "Nhị Chuyển", "Tam Chuyển", "Tứ Chuyển", "Ngũ Chuyển", "Lục Tiên", "Thất Tiên", "Bát Tiên", "Cửu Tôn"];
  return names[r] || `Cửu Tôn`;
};

export const subRankLabel = (s: number, r?: number) => {
  if (r && r >= 6) return "";
  return ["", "Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"][s] || "";
};

export const typeColor = (t?: string) => {
  switch (t) {
    case "Boss": return "text-rose-400 border-rose-800 bg-rose-950/30";
    case "Elite": return "text-purple-400 border-purple-800 bg-purple-950/30";
    case "Normal":
    case "MONSTER": return "text-zinc-400 border-zinc-700 bg-zinc-900/50";
    default: return "text-zinc-400 border-zinc-700 bg-zinc-900/50";
  }
};

export const gridXLabel = (x?: number) => {
  switch (x) {
    case 1: return "Tiền";
    case 2: return "Trung";
    case 3: return "Hậu";
    default: return "?";
  }
};

export const EFFECT_STYLE: Record<string, string> = {
  damage: "bg-red-900/50 text-red-300 border-red-500/30",
  shield: "bg-blue-900/50 text-blue-300 border-blue-500/30",
  STUN: "bg-yellow-900/50 text-yellow-300 border-yellow-500/30",
  BLEED: "bg-rose-900/50 text-rose-300 border-rose-500/30",
  POISON: "bg-lime-900/50 text-lime-300 border-lime-500/30",
  SLOW: "bg-sky-900/50 text-sky-300 border-sky-500/30",
  SLOW1: "bg-sky-900/50 text-sky-200 border-sky-400/30",
  WEAK: "bg-orange-900/50 text-orange-300 border-orange-500/30",
  FREEZE: "bg-cyan-900/50 text-cyan-300 border-cyan-500/30",
  SILENCE: "bg-zinc-700/50 text-zinc-300 border-zinc-500/30",
  THORNS: "bg-emerald-900/50 text-emerald-300 border-emerald-500/30",
  REGEN: "bg-green-900/50 text-green-300 border-green-500/30",
  ATK_UP: "bg-amber-900/50 text-amber-300 border-amber-500/30",
  DEF_UP: "bg-teal-900/50 text-teal-300 border-teal-500/30",
  TAUNT: "bg-indigo-900/50 text-indigo-300 border-indigo-500/30",
  FORGET_EGO: "bg-violet-900/50 text-violet-300 border-violet-500/30",
  FORGET_REALITY: "bg-fuchsia-900/50 text-fuchsia-300 border-fuchsia-500/30",
  SLEEP: "bg-indigo-900/50 text-indigo-300 border-indigo-500/30",
  lifesteal: "bg-rose-900/50 text-rose-300 border-rose-500/30",
  heal: "bg-green-900/50 text-green-300 border-green-500/30",
  mana: "bg-purple-900/50 text-purple-300 border-purple-500/30",
  capture: "bg-pink-900/50 text-pink-300 border-pink-500/30",
};

export const EFFECT_LABEL: Record<string, string> = {
  damage: "Sát thương",
  shield: "Khiên",
  STUN: "Choáng",
  BLEED: "Chảy máu",
  POISON: "Độc",
  SLOW: "Chậm",
  SLOW1: "Chậm 100%",
  WEAK: "Yếu",
  FREEZE: "Đóng băng",
  SILENCE: "Im lặng",
  THORNS: "Gai",
  REGEN: "Hồi máu",
  ATK_UP: "Tăng Công",
  DEF_UP: "Tăng Thủ",
  TAUNT: "Khiêu khích",
  FORGET_EGO: "Quên Bản Ngã",
  FORGET_REALITY: "Quên Thực Tại",
  SLEEP: "Ngủ",
  lifesteal: "Hút HP",
  heal: "Chữa lành",
  mana: "Hồi Nguyên",
  capture: "Bắt",
};

export const getCombatEffects = (spec: unknown): EffectBadge[] => {
  if (!spec || typeof spec !== "object") return [];
  const effects = (spec as { effects?: EffectEnvelope }).effects;
  if (!effects) return [];
  const out: EffectBadge[] = [];
  for (const op of operationsBy(effects)) {
    const { value, unit } = numericValue(op);
    const shown = unit === "PERCENT" || unit === "RATIO" ? `${Math.round(value * 100)}%` : `${value}`;
    if (op.op === "DAMAGE" || op.op === "DIRECT_HP") out.push({ key: "damage", label: `DMG ${shown}`, className: EFFECT_STYLE.damage });
    if (op.op === "SHIELD" || op.op === "BARRIER") out.push({ key: "shield", label: `Khiên ${shown}`, className: EFFECT_STYLE.shield });
    if (op.op === "HEAL") out.push({ key: op.resource === "MANA" ? "mana" : "heal", label: `${op.resource === "MANA" ? "+" : "Chữa "}${shown}`, className: EFFECT_STYLE[op.resource === "MANA" ? "mana" : "heal"] });
    if (op.op === "STATUS" && op.status_id) {
      const upper = op.status_id.toUpperCase();
      out.push({ key: upper, label: EFFECT_LABEL[upper] || upper, className: EFFECT_STYLE[upper] || EFFECT_STYLE.mana });
    }
  }
  return out.slice(0, 4);
};
