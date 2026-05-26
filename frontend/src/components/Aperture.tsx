"use client";

import React from "react";

interface ApertureProps {
  currentMana: number;
  maxMana: number;
  aptitudeType: string;
  aptitudePercent: number;
}

export const Aperture: React.FC<ApertureProps> = ({
  currentMana,
  maxMana,
  aptitudeType,
  aptitudePercent,
}) => {
  const percent = Math.min(100, Math.max(0, (currentMana / maxMana) * 100));

  // Determine mana quality text based on Rank (which is 1 in Phase 1)
  const manaQuality = "Rank 1 - Copper Green Chân Nguyên";

  return (
    <div className="flex flex-col items-center justify-center p-6 glass-panel-gold rounded-3xl w-full max-w-sm">
      <h3 className="text-lg font-semibold text-amber-400 mb-2 tracking-wide uppercase font-serif">
        Không Khiếu (Aperture)
      </h3>
      <p className="text-xs text-zinc-400 mb-6 italic text-center">
        Tư Chất: {aptitudeType} ({aptitudePercent}%) • {manaQuality}
      </p>

      {/* Flask Container representing the Aperture Sea */}
      <div className="relative w-40 h-64 border-4 border-amber-500/20 rounded-t-full rounded-b-[45%] p-1.5 bg-zinc-950/90 shadow-[0_0_40px_rgba(16,185,129,0.1)] overflow-hidden mb-6">
        <div className="wave-container rounded-t-full rounded-b-[42%]">
          {/* Wave 1 */}
          <div
            className="wave-liquid"
            style={{
              top: `${100 - percent}%`,
            }}
          />
          {/* Wave 2 */}
          <div
            className="wave-liquid-secondary"
            style={{
              top: `${100 - percent}%`,
            }}
          />

          {/* Depth/Reflections */}
          <div className="absolute inset-0 bg-gradient-to-t from-transparent via-transparent to-zinc-950/20 pointer-events-none" />

          {/* Centered Percentage Display */}
          <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none z-10">
            <span className="text-2xl font-bold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
              {percent.toFixed(1)}%
            </span>
            <span className="text-xs text-zinc-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)] font-semibold mt-0.5">
              {currentMana.toFixed(1)} / {maxMana.toFixed(0)}
            </span>
          </div>

          {/* Level indicators */}
          <div className="absolute right-2 top-0 bottom-0 flex flex-col justify-between text-[9px] text-zinc-500 font-mono py-6 pointer-events-none">
            <span>80%</span>
            <span>60%</span>
            <span>40%</span>
            <span>20%</span>
          </div>
        </div>
      </div>

      <div className="w-full text-center">
        <div className="text-sm font-medium text-emerald-400">
          Chân Nguyên Hải (Primeval Sea)
        </div>
        <div className="text-[10px] text-zinc-500 mt-1">
          Regenerating: {aptitudeType === "Giáp" ? "8.0" : aptitudeType === "Ất" ? "6.0" : aptitudeType === "Bính" ? "4.0" : "2.0"} mana/hour
        </div>
      </div>
    </div>
  );
};
