"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Clock, Shield, Swords, DoorOpen, Users, Compass, AlertCircle } from "lucide-react";

export const SpatialRealmHUD: React.FC = () => {
  const activeSpatialRealm = useGameStore((state) => state.activeSpatialRealm);
  const fetchActiveSpatialRealm = useGameStore((state) => state.fetchActiveSpatialRealm);
  const exitSpatialRealm = useGameStore((state) => state.exitSpatialRealm);
  const spatialLoading = useGameStore((state) => state.spatialLoading);
  const showConfirm = useGameStore((state) => state.showConfirm);
  const player = useGameStore((state) => state.player);

  const [timeLeft, setTimeLeft] = useState<string>("");
  const [isExpiringSoon, setIsExpiringSoon] = useState<boolean>(false);

  useEffect(() => {
    // Poll active spatial realm every 20s
    void fetchActiveSpatialRealm();
    const interval = setInterval(() => {
      void fetchActiveSpatialRealm();
    }, 20000);
    return () => clearInterval(interval);
  }, [fetchActiveSpatialRealm]);

  useEffect(() => {
    if (!activeSpatialRealm?.expires_at) {
      setTimeLeft("");
      return;
    }

    const updateTimer = () => {
      const expiresAt = new Date(activeSpatialRealm.expires_at).getTime();
      const now = Date.now();
      const diff = expiresAt - now;

      if (diff <= 0) {
        setTimeLeft("00:00 (Hết giờ)");
        setIsExpiringSoon(true);
        return;
      }

      const totalSeconds = Math.floor(diff / 1000);
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = totalSeconds % 60;

      setIsExpiringSoon(minutes < 5);
      setTimeLeft(
        `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`
      );
    };

    updateTimer();
    const timer = setInterval(updateTimer, 1000);
    return () => clearInterval(timer);
  }, [activeSpatialRealm?.expires_at]);

  if (!activeSpatialRealm) {
    return null;
  }

  const handleExit = async () => {
    const ok = await showConfirm(
      "Đạo hữu có chắc chắn muốn rời khỏi bí cảnh?\n\nNếu đang ở xa Cổng Trở Về, đạo hữu sẽ cần bước tới ô Cổng Trở Về để tiến hành rời đi an toàn."
    );
    if (ok) {
      await exitSpatialRealm();
    }
  };

  const scopeLabel =
    activeSpatialRealm.scope === "SHARED"
      ? "Bí Cảnh Chung"
      : activeSpatialRealm.scope === "PARTY"
      ? "Truyền Thừa Tổ Đội"
      : "Truyền Thừa Cá Nhân";

  const isSafePvP = (activeSpatialRealm.rules?.pvp_mode || "SAFE").toUpperCase() === "SAFE";

  return (
    <aside aria-label="Bảng trạng thái bí cảnh" className="w-full max-w-xl mx-auto mb-3 bg-gradient-to-r from-zinc-950/90 via-amber-950/30 to-zinc-950/90 border border-amber-500/40 rounded-2xl p-3 shadow-[0_0_20px_rgba(245,158,11,0.2)] backdrop-blur-md transition-all">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-amber-500/20 pb-2 mb-2">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-lg bg-amber-500/20 border border-amber-400/40 flex items-center justify-center text-amber-300">
            <Compass className="w-4 h-4 animate-spin-slow" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-amber-200 text-sm tracking-wide">
                {activeSpatialRealm.name}
              </span>
              <span className="px-2 py-0.5 text-[10px] font-semibold bg-amber-500/10 text-amber-300 border border-amber-500/30 rounded-full">
                {scopeLabel}
              </span>
            </div>
            {player && (
              <span className="text-[10px] text-zinc-400">
                Tọa độ hiện tại: ({player.x}, {player.y})
              </span>
            )}
          </div>
        </div>

        <button
          onClick={handleExit}
          disabled={spatialLoading}
          className="flex items-center gap-1.5 px-3 py-1 text-xs font-semibold bg-rose-950/60 hover:bg-rose-900/80 text-rose-300 border border-rose-500/40 rounded-xl transition-all hover:scale-105 active:scale-95 disabled:opacity-50"
        >
          <DoorOpen className="w-3.5 h-3.5" />
          <span>Rời Bí Cảnh</span>
        </button>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        {/* Countdown */}
        <div
          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg border font-mono font-medium ${
            isExpiringSoon
              ? "bg-rose-950/40 border-rose-500/50 text-rose-300 animate-pulse"
              : "bg-zinc-900/60 border-zinc-700/50 text-amber-300"
          }`}
        >
          <Clock className="w-3.5 h-3.5" />
          <span>Thời gian: {timeLeft || "--:--"}</span>
        </div>

        {/* Rules Badges */}
        <div className="flex flex-wrap items-center gap-1.5">
          {/* PvP Mode */}
          <span
            className={`flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium border ${
              isSafePvP
                ? "bg-emerald-950/40 border-emerald-500/40 text-emerald-300"
                : "bg-rose-950/40 border-rose-500/40 text-rose-300"
            }`}
          >
            {isSafePvP ? <Shield className="w-3 h-3" /> : <Swords className="w-3 h-3" />}
            <span>{isSafePvP ? "An Toàn (Cấm PK)" : "Đồ Sát Tự Do"}</span>
          </span>

          {/* Teleport restriction */}
          {!activeSpatialRealm.rules?.allow_teleport && (
            <span className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium bg-amber-950/40 border border-amber-500/30 text-amber-300">
              <AlertCircle className="w-3 h-3" />
              <span>Cấm Truyền Tống</span>
            </span>
          )}

          {/* Members count */}
          {activeSpatialRealm.player_count > 0 && (
            <span className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium bg-zinc-900/60 border border-zinc-700 text-zinc-300">
              <Users className="w-3 h-3" />
              <span>{activeSpatialRealm.player_count} Cổ sư</span>
            </span>
          )}
        </div>
      </div>
    </aside>
  );
};
