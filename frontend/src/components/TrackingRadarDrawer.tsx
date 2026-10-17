"use client";

import React, { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { usePolling } from "@/lib/usePolling";
import { Crosshair, MapPin, Radar, RefreshCw, Scissors, ShieldAlert, X } from "lucide-react";

export const TrackingRadarDrawer: React.FC = () => {
  const {
    isRadarDrawerOpen,
    toggleRadarDrawer,
    trackedTargets,
    fetchTrackedTargets,
    severClues,
  } = useGameStore();

  useEffect(() => {
    if (isRadarDrawerOpen) fetchTrackedTargets();
  }, [isRadarDrawerOpen, fetchTrackedTargets]);
  usePolling(fetchTrackedTargets, 10000, isRadarDrawerOpen);

  if (!isRadarDrawerOpen) return null;

  const handleSeverClues = async () => {
    await severClues();
    fetchTrackedTargets();
  };

  return (
    <div className="fixed bottom-[calc(env(safe-area-inset-bottom)+1rem)] right-[calc(env(safe-area-inset-right)+1rem)] z-40 flex max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-2rem)] w-[min(24rem,calc(100vw-2rem))] flex-col overflow-hidden rounded-xl border border-cyan-500/30 bg-neutral-950/95 shadow-2xl backdrop-blur-md">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-cyan-500/20 bg-neutral-900 px-4 py-3">
        <div className="flex items-center gap-2">
          <Radar className="h-5 w-5 animate-pulse text-cyan-400" />
          <h3 className="font-bold text-cyan-300">Radar Truy Tung & Đoạn Tuyệt</h3>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => fetchTrackedTargets()}
            title="Làm mới"
            className="rounded p-1 text-neutral-400 hover:bg-neutral-800 hover:text-white"
          >
            <RefreshCw className="h-4 w-4" />
          </button>
          <button
            onClick={() => toggleRadarDrawer(false)}
            className="rounded p-1 text-neutral-400 hover:bg-neutral-800 hover:text-white"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Target List */}
      <div className="flex-1 min-h-0 space-y-2 overflow-y-auto p-4 text-xs">
        <div className="flex items-center justify-between text-neutral-400">
          <span>Mục tiêu đang bị gắn ấn ký ({trackedTargets.length})</span>
          <button
            onClick={handleSeverClues}
            className="flex items-center gap-1 rounded bg-red-950/60 px-2 py-1 font-semibold text-red-300 border border-red-800/50 hover:bg-red-900/60"
          >
            <Scissors className="h-3.5 w-3.5" />
            Chém Đứt Manh Mối
          </button>
        </div>

        {trackedTargets.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-6 text-neutral-500">
            <Crosshair className="mb-2 h-8 w-8 opacity-40" />
            <p>Chưa có mục tiêu nào bị gắn ấn ký.</p>
            <p className="text-[10px] text-neutral-600">Hãy dùng Suy Diễn Thiên Cơ để theo dấu con mồi.</p>
          </div>
        ) : (
          trackedTargets.map((target) => (
            <div
              key={target.id}
              className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/90 p-2.5"
            >
              <div>
                <div className="flex items-center gap-1.5 font-bold text-white">
                  <span>{target.target_username || "Cổ Sư Vô Danh"}</span>
                  <span className="rounded bg-cyan-950 px-1.5 py-0.2 font-mono text-[9px] text-cyan-300">
                    {target.mark_type === "blood" ? "Huyết Ấn" : "Trí Niệm"}
                  </span>
                </div>
                <div className="mt-1 flex items-center gap-2 text-[11px] text-neutral-400">
                  <MapPin className="h-3 w-3 text-amber-400" />
                  <span>
                    {target.current_region || "Vùng chưa rõ"} ({target.x ?? "?"}, {target.y ?? "?"})
                  </span>
                </div>
              </div>

              <div className="text-right">
                <span className="block font-mono text-[10px] text-neutral-400">
                  Hết hạn: {new Date(target.expires_at).toLocaleTimeString()}
                </span>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Footer warning */}
      <div className="border-t border-neutral-800/80 bg-neutral-900/50 px-4 py-2 text-[10px] text-neutral-400 flex items-center gap-1.5">
        <ShieldAlert className="h-3.5 w-3.5 text-amber-400 shrink-0" />
        <span>Nếu bạn nghi ngờ bị theo dõi, hãy nhấn &quot;Chém Đứt Manh Mối&quot; để thanh tẩy toàn thân.</span>
      </div>
    </div>
  );
};
