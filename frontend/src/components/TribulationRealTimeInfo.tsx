"use client";

import React, { useState, useEffect } from "react";
import { ImmortalAperture, useGameStore } from "@/store/useGameStore";

interface TribulationRealTimeInfoProps {
  aperture: ImmortalAperture;
  timeSpeedMultiplier?: number;
  isCompact?: boolean;
}

export const TribulationRealTimeInfo: React.FC<TribulationRealTimeInfoProps> = ({ aperture, timeSpeedMultiplier = 1, isCompact }) => {
  const playerRace = useGameStore((state) => state.player?.race);
  const [timeLeftText, setTimeLeftText] = useState<string>("");
  const [arrivalTimeText, setArrivalTimeText] = useState<string>("");
  const [isPassed, setIsPassed] = useState<boolean>(false);

  useEffect(() => {
    const updateTime = () => {
      if (playerRace === "cuong_thi") {
        setTimeLeftText("Ngưng trệ (Tử Khiếu)");
        setArrivalTimeText("Không giáng kiếp");
        setIsPassed(false);
        return;
      }

      if (aperture.state === "TRIBULATION_ACTIVE") {
        setTimeLeftText("Đang diễn ra");
        setArrivalTimeText("Đang chiến đấu");
        setIsPassed(true);
        return;
      }

      const internalYearsLeft = aperture.next_tribulation_year - aperture.internal_years_passed;
      if (internalYearsLeft <= 0) {
        setTimeLeftText("Đang ập đến");
        setArrivalTimeText("Bất cứ lúc nào");
        setIsPassed(true);
        return;
      }

      // Prefer the backend timestamp, which includes all effective time-speed bonuses.
      const fallbackRate = (aperture.time_rate || 1.0) * (timeSpeedMultiplier > 0 ? timeSpeedMultiplier : 1.0);
      const fallbackSeconds = (internalYearsLeft * 36000.0) / fallbackRate;
      const fallbackArrivalMs = new Date(aperture.updated_at).getTime() + fallbackSeconds * 1000;
      const backendArrivalMs = aperture.next_tribulation_at ? new Date(aperture.next_tribulation_at).getTime() : NaN;
      const arrivalTimeMs = Number.isFinite(backendArrivalMs) ? backendArrivalMs : fallbackArrivalMs;
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
  }, [aperture.next_tribulation_year, aperture.internal_years_passed, aperture.time_rate, aperture.state, aperture.updated_at, timeSpeedMultiplier, playerRace]);

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