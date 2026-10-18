"use client";

import React from "react";
import { SkillArtBackground } from "./SkillArtBackground";
import { SkillIcon } from "./SkillIcon";
import type { PvPSkill } from "@/store/slices/pvpSlice";

interface MobaSkillSlotProps {
  skill: PvPSkill;
  index: number;
  name: string;
  imageUrl?: string;
  hotkey: string;
  cooldownRemainingMs: number;
  isSelected?: boolean;
  isAiming?: boolean;
  shape?: "circle" | "rounded";
  size?: number; // size in px, e.g. 56, 64, 72
  showDetails?: boolean;
  onPointerDown?: (event: React.PointerEvent) => void;
  onPointerMove?: (event: React.PointerEvent) => void;
  onPointerUp?: (event: React.PointerEvent) => void;
  onPointerCancel?: (event: React.PointerEvent) => void;
  onClick?: (event: React.MouseEvent) => void;
  className?: string;
}

const MODE_LABELS: Record<string, string> = {
  TARGET: "Mục Tiêu",
  DIRECTION: "Định Hướng",
  GROUND: "Vùng Đất",
  SELF: "Tự Thân",
};

export const MobaSkillSlot: React.FC<MobaSkillSlotProps> = ({
  skill,
  index: _index,
  name,
  imageUrl,
  hotkey,
  cooldownRemainingMs,
  isSelected = false,
  isAiming = false,
  shape = "circle",
  size = 64,
  showDetails = false,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onPointerCancel,
  onClick,
  className = "",
}) => {
  const ready = cooldownRemainingMs <= 0;
  const cooldownFraction = skill.cooldown_ms > 0 ? Math.min(1, Math.max(0, cooldownRemainingMs / skill.cooldown_ms)) : 0;
  const cooldownSec = (cooldownRemainingMs / 1000).toFixed(1);

  const isCircle = shape === "circle";
  const radiusClass = isCircle ? "rounded-full" : "rounded-xl";

  const modeBadge = skill.move_distance > 0 ? "LƯỚT" : (MODE_LABELS[skill.cast_mode] ?? skill.cast_mode);

  return (
    <div
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerCancel}
      onClick={onClick}
      onContextMenu={(e) => e.preventDefault()}
      style={{ width: `${size}px`, height: showDetails ? undefined : `${size}px` }}
      className={`relative flex select-none touch-none flex-col items-center cursor-pointer transition-transform active:scale-95 ${className}`}
    >
      {/* Skill Button Core Orb/Box */}
      <div
        style={{ width: `${size}px`, height: `${size}px` }}
        className={`relative overflow-hidden ${radiusClass} border-2 transition-all ${
          isAiming || isSelected
            ? "border-amber-300 ring-2 ring-amber-400/50 shadow-[0_0_15px_rgba(251,191,36,0.6)]"
            : ready
              ? "border-emerald-500/60 hover:border-emerald-400 shadow-[0_0_10px_rgba(16,185,129,0.3)]"
              : "border-zinc-700/80 opacity-60"
        } bg-[#0c1210]`}
      >
        {/* Background Artwork */}
        {imageUrl ? (
          <img src={imageUrl} alt={name} className="h-full w-full object-cover" />
        ) : (
          <SkillArtBackground skill={skill} name={name} />
        )}

        {/* Central Glyph Overlay */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none drop-shadow-[0_2px_4px_rgba(0,0,0,0.9)]">
          <SkillIcon skill={skill} size={Math.round(size * 0.46)} className="transition-transform group-hover:scale-105" />
        </div>

        {/* Runic Trim / Vignette */}
        <div className={`absolute inset-0 ${radiusClass} pointer-events-none border border-white/20 shadow-[inset_0_0_8px_rgba(0,0,0,0.8)]`} />

        {/* Cooldown Sweep Overlay */}
        {!ready && (
          <div
            className={`absolute inset-0 ${radiusClass} flex items-center justify-center bg-black/75 backdrop-blur-[1px]`}
            style={{
              background: `conic-gradient(rgba(0,0,0,0.85) ${cooldownFraction * 360}deg, transparent 0deg)`,
            }}
          >
            <span className="font-mono text-xs font-black tracking-tighter text-amber-300 drop-shadow-[0_2px_4px_rgba(0,0,0,1)]">
              {cooldownSec}s
            </span>
          </div>
        )}

        {/* Hotkey Tag (Top Left for PC/Display) */}
        {hotkey && (
          <div className="absolute left-1 top-1 flex h-4 min-w-4 items-center justify-center rounded bg-black/80 px-1 font-mono text-[9px] font-bold text-amber-300 shadow border border-amber-400/30">
            {hotkey}
          </div>
        )}

        {/* Mana Cost Tag (Bottom Right) */}
        {skill.mana_cost > 0 && (
          <div className="absolute right-1 bottom-1 flex items-center gap-0.5 rounded bg-sky-950/80 px-1 py-0.2 font-mono text-[8px] font-bold text-sky-300 border border-sky-400/30">
            <span className="text-[7px]">💧</span>
            <span>{skill.mana_cost}</span>
          </div>
        )}

        {/* Aiming Pulse Glow */}
        {isAiming && (
          <div className={`absolute inset-0 ${radiusClass} animate-pulse border-2 border-amber-300 pointer-events-none`} />
        )}
      </div>

      {/* Optional Under-text Details (For PC Bar) */}
      {showDetails && (
        <div className="mt-1 flex w-full flex-col items-center text-center">
          <span className="w-full truncate text-[11px] font-medium text-zinc-100">{name}</span>
          <span className="rounded bg-white/5 px-1 py-0.5 text-[8px] font-semibold uppercase tracking-wider text-emerald-400/90">
            {modeBadge}
          </span>
        </div>
      )}
    </div>
  );
};
