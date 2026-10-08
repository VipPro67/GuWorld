"use client";

import { useWorldStore } from "@/store/useWorldStore";
import { X, ShieldAlert, Skull, Bug, User } from "lucide-react";

export function TargetHud() {
  const selectedTarget = useWorldStore((s) => s.selectedTarget);
  const selectTarget = useWorldStore((s) => s.selectTarget);

  if (!selectedTarget) {
    return null;
  }

  const getTargetIcon = (cellType: string) => {
    switch (cellType) {
      case "Boss":
        return <Skull className="w-5 h-5 text-red-400" />;
      case "Monster":
        return <ShieldAlert className="w-5 h-5 text-amber-400" />;
      case "WildGu":
        return <Bug className="w-5 h-5 text-purple-400" />;
      default:
        return <User className="w-5 h-5 text-sky-400" />;
    }
  };

  return (
    <div className="absolute top-3 left-1/2 -translate-x-1/2 z-30 pointer-events-auto select-none animate-fade-in-down">
      <div className="flex items-center gap-3 px-4 py-2 bg-zinc-950/95 backdrop-blur-md rounded-2xl border border-red-500/40 shadow-[0_0_15px_rgba(239,68,68,0.3)] min-w-[280px] max-w-sm">
        {/* Icon */}
        <div className="w-9 h-9 rounded-xl bg-red-950/60 border border-red-500/30 flex items-center justify-center shrink-0">
          {getTargetIcon(selectedTarget.cellType)}
        </div>

        {/* Target info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2 mb-1">
            <span className="font-bold font-serif text-xs text-red-300 truncate">
              {selectedTarget.name}
            </span>
            {selectedTarget.rank && (
              <span className="text-[9px] font-mono px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-300 border border-zinc-700 shrink-0">
                Rank {selectedTarget.rank}
              </span>
            )}
          </div>

          {selectedTarget.instanceId && (
            <div className="mt-1 text-[9px] text-amber-300/80 font-serif">
              Bấm kỹ năng sát thương hoặc phím 1–5 để tung chiêu
            </div>
          )}
        </div>

        {/* Clear target button */}
        <button
          onClick={() => selectTarget(null)}
          className="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors border border-zinc-800 cursor-pointer"
          title="Hủy chọn mục tiêu"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
