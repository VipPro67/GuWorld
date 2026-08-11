export function formatNumber(n: number): string {
    const units = [
        { value: 1e33, suffix: "Dc" },
        { value: 1e30, suffix: "No" },
        { value: 1e27, suffix: "Oc" },
        { value: 1e24, suffix: "Sp" },
        { value: 1e21, suffix: "Sx" },
        { value: 1e18, suffix: "Qi" },
        { value: 1e15, suffix: "Qa" },
        { value: 1e12, suffix: "T" },
        { value: 1e9, suffix: "B" },
        { value: 1e6, suffix: "M" },
        { value: 1e3, suffix: "K" }
    ];

    const abs = Math.abs(n);

    for (const unit of units) {
        if (abs >= unit.value) {
            const v = n / unit.value;
            return `${Number(v.toFixed(1))}${unit.suffix}`;
        }
    }

    return n.toLocaleString();
}
const PATH_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  Fire:       { bg: "bg-red-900/40",    text: "text-red-300",     border: "border-red-500/40" },
  Sword:      { bg: "bg-slate-700/40",  text: "text-slate-200",   border: "border-slate-400/40" },
  Strength:   { bg: "bg-orange-900/40", text: "text-orange-300",  border: "border-orange-500/40" },
  Lightning:  { bg: "bg-yellow-900/40", text: "text-yellow-300",  border: "border-yellow-500/40" },
  Ice:        { bg: "bg-cyan-900/40",   text: "text-cyan-300",    border: "border-cyan-500/40" },
  Water:      { bg: "bg-blue-900/40",   text: "text-blue-300",    border: "border-blue-500/40" },
  Wind:       { bg: "bg-teal-900/40",   text: "text-teal-300",    border: "border-teal-500/40" },
  Wood:       { bg: "bg-green-900/40",  text: "text-green-300",   border: "border-green-500/40" },
  Earth:      { bg: "bg-amber-900/40",  text: "text-amber-300",   border: "border-amber-500/40" },
  Metal:      { bg: "bg-zinc-700/40",   text: "text-zinc-200",    border: "border-zinc-400/40" },
  Blood:      { bg: "bg-rose-900/40",   text: "text-rose-300",    border: "border-rose-500/40" },
  Poison:     { bg: "bg-lime-900/40",   text: "text-lime-300",    border: "border-lime-500/40" },
  Soul:       { bg: "bg-purple-900/40", text: "text-purple-300",  border: "border-purple-500/40" },
  Illusion:   { bg: "bg-fuchsia-900/40",text: "text-fuchsia-300", border: "border-fuchsia-500/40" },
  Dream:      { bg: "bg-indigo-900/40", text: "text-indigo-300",  border: "border-indigo-500/40" },
  Qi:         { bg: "bg-violet-900/40", text: "text-violet-300",  border: "border-violet-500/40" },
  Time:       { bg: "bg-sky-900/40",    text: "text-sky-300",     border: "border-sky-500/40" },
  Space:      { bg: "bg-violet-900/40", text: "text-violet-200",  border: "border-violet-400/40" },
  Light:      { bg: "bg-amber-900/40",  text: "text-amber-200",   border: "border-amber-400/40" },
  Dark:       { bg: "bg-zinc-900/60",   text: "text-zinc-300",    border: "border-zinc-500/40" },
  Sound:      { bg: "bg-pink-900/40",   text: "text-pink-300",    border: "border-pink-500/40" },
  Star:       { bg: "bg-indigo-900/40", text: "text-indigo-200",  border: "border-indigo-400/40" },
  Cloud:      { bg: "bg-slate-700/40",  text: "text-slate-300",   border: "border-slate-500/40" },
  Moon:       { bg: "bg-sky-900/40",    text: "text-sky-200",     border: "border-sky-400/40" },
  Sun:        { bg: "bg-yellow-900/40", text: "text-yellow-200",  border: "border-yellow-400/40" },
  Theft:      { bg: "bg-emerald-900/40",text: "text-emerald-300", border: "border-emerald-500/40" },
  Information:{ bg: "bg-cyan-900/40",   text: "text-cyan-200",    border: "border-cyan-400/40" },
  Bone:       { bg: "bg-neutral-700/40",text: "text-neutral-200", border: "border-neutral-400/40" },
  Phantom:    { bg: "bg-purple-900/40", text: "text-purple-200",  border: "border-purple-400/40" },
  Luck:       { bg: "bg-lime-900/40",   text: "text-lime-200",    border: "border-lime-400/40" },
  Food:       { bg: "bg-orange-900/40", text: "text-orange-200",  border: "border-orange-400/40" },
  Painting:   { bg: "bg-rose-900/40",   text: "text-rose-200",    border: "border-rose-400/40" },
  Heaven:     { bg: "bg-amber-900/40",  text: "text-amber-100",   border: "border-amber-300/40" },
  Human:      { bg: "bg-orange-900/40", text: "text-orange-200",  border: "border-orange-400/40" },
  Weapon:     { bg: "bg-slate-700/40",  text: "text-slate-100",   border: "border-slate-300/40" },
  Formation:  { bg: "bg-teal-900/40",   text: "text-teal-200",    border: "border-teal-400/40" },
  Yin:        { bg: "bg-indigo-900/40", text: "text-indigo-300",  border: "border-indigo-500/40" },
  Yang:       { bg: "bg-yellow-900/40", text: "text-yellow-300",  border: "border-yellow-500/40" },
  Enslavement:{ bg: "bg-red-900/40",    text: "text-red-200",     border: "border-red-400/40" },
  Refinement: { bg: "bg-orange-900/40", text: "text-orange-300",  border: "border-orange-500/40" },
  Rule:       { bg: "bg-zinc-700/40",   text: "text-zinc-100",    border: "border-zinc-300/40" },
  Transformation: { bg: "bg-fuchsia-900/40", text: "text-fuchsia-300", border: "border-fuchsia-500/40" },
  Wisdom:     { bg: "bg-sky-900/40",    text: "text-sky-300",     border: "border-sky-500/40" },
};

const DEFAULT_PATH_COLOR = { bg: "bg-zinc-800/60", text: "text-zinc-400", border: "border-zinc-600/40" };

export function getPathColor(path: string): { bg: string; text: string; border: string } {
  const key = path.replace(/\s*Đạo$/, "").trim();
  return PATH_COLORS[key] || DEFAULT_PATH_COLOR;
}
