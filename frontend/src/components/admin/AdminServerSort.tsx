"use client";

export function AdminServerSort({ sort, direction, options, onChange }: {
  sort: string;
  direction: string;
  options: Array<{ value: string; label: string }>;
  onChange: (sort: string, direction: "asc" | "desc") => void;
}) {
  return <div className="flex items-center gap-1.5 text-xs">
    <span className="text-zinc-500">Sắp xếp</span>
    <select value={sort} onChange={(event) => onChange(event.target.value, direction === "asc" ? "asc" : "desc")} className="rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-zinc-300">
      {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
    <button type="button" onClick={() => onChange(sort, direction === "asc" ? "desc" : "asc")} className="rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-zinc-300 hover:text-amber-400">
      {direction === "asc" ? "Tăng dần" : "Giảm dần"}
    </button>
  </div>;
}
