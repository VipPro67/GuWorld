"use client";

import type { PvPSkill } from "@/store/slices/pvpSlice";

const ACCENT: Record<string, string> = {
  TARGET: "#fbbf24",
  DIRECTION: "#38bdf8",
  GROUND: "#a78bfa",
  SELF: "#34d399",
};

/**
 * Skill icon.
 *
 * The repository ships no Gu artwork (`Gu.csv.image_url` is empty), so the
 * default icon is a procedural glyph built from the same aim/shape data the
 * server resolves: crosshair for point-and-click, a lane for a skillshot, a
 * sector for a cone, rings for a ground AoE, a dashed arrow for a blink and a
 * shield for a self-cast. An administrator-set `image_url` always wins.
 */
export const SkillIcon: React.FC<{ skill: PvPSkill; imageUrl?: string; size?: number; className?: string }> = ({
  skill,
  imageUrl,
  size = 30,
  className,
}) => {
  if (imageUrl) {
    return (
      <img src={imageUrl} alt="" width={size} height={size} className={`rounded-md object-cover ${className ?? ""}`} />
    );
  }
  const accent = ACCENT[skill.cast_mode] ?? "#94a3b8";
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
      {glyph(skill, accent)}
    </svg>
  );
};

function glyph(skill: PvPSkill, accent: string) {
  if (skill.gu_id === "BASIC_ATTACK") {
    // Crossed swords / strike blade glyph
    return (
      <>
        <path d="M14.5 17.5 L3 6 V3 H6 L17.5 14.5" stroke="#fef08a" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M13 19 L19 13" stroke="#f59e0b" strokeWidth="1.8" strokeLinecap="round" />
        <path d="M9.5 17.5 L21 6 V3 H18 L6.5 14.5" stroke="#ffffff" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M11 19 L5 13" stroke="#f59e0b" strokeWidth="1.8" strokeLinecap="round" />
      </>
    );
  }
  if (skill.cast_mode === "SELF") {
    // Shield: self-buff, cast instantly.
    return (
      <>
        <path d="M12 2.5 L19.5 5.5 V12 C19.5 16.6 16.2 19.9 12 21.5 C7.8 19.9 4.5 16.6 4.5 12 V5.5 Z" fill={accent} fillOpacity="0.25" stroke={accent} strokeWidth="1.6" />
        <path d="M8.6 12.2 L11 14.6 L15.6 9.6" stroke={accent} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
      </>
    );
  }
  if (skill.move_distance > 0) {
    // Blink: dashed path to a destination ring.
    return (
      <>
        <path d="M3 20 L14.5 8.5" stroke={accent} strokeWidth="2" strokeLinecap="round" strokeDasharray="3 2.5" />
        <path d="M14 8 L21 4.5 L18.5 11.5 Z" fill={accent} fillOpacity="0.3" stroke={accent} strokeWidth="1.5" strokeLinejoin="round" />
        <circle cx="5" cy="5" r="2.2" stroke={accent} strokeWidth="1.6" />
      </>
    );
  }
  if (skill.shape === "CONE") {
    // Sector: wave/fan from the caster.
    return (
      <>
        <path d="M12 21 L3.6 7.4 A16 16 0 0 1 20.4 7.4 Z" fill={accent} fillOpacity="0.22" stroke={accent} strokeWidth="1.6" strokeLinejoin="round" />
        <path d="M8.4 13.6 A8 8 0 0 1 15.6 13.6" stroke={accent} strokeWidth="1.4" strokeLinecap="round" opacity="0.75" />
      </>
    );
  }
  if (skill.shape === "LINE") {
    // Lane: skillshot travelling outward.
    return (
      <>
        <rect x="2.5" y="10.4" width="12.5" height="3.2" rx="1.6" fill={accent} fillOpacity="0.28" stroke={accent} strokeWidth="1.5" />
        <path d="M16 7.2 L22 12 L16 16.8 Z" fill={accent} fillOpacity="0.45" stroke={accent} strokeWidth="1.5" strokeLinejoin="round" />
      </>
    );
  }
  if (skill.shape === "CIRCLE") {
    // Concentric rings: area cast with a wind-up.
    return (
      <>
        <circle cx="12" cy="12" r="9" fill={accent} fillOpacity="0.15" stroke={accent} strokeWidth="1.5" />
        <circle cx="12" cy="12" r="5.4" stroke={accent} strokeWidth="1.6" />
        <circle cx="12" cy="12" r="1.9" fill={accent} />
      </>
    );
  }
  // Single target: crosshair.
  return (
    <>
      <circle cx="12" cy="12" r="5.2" stroke={accent} strokeWidth="1.7" />
      <path d="M12 2.2 V6 M12 18 V21.8 M2.2 12 H6 M18 12 H21.8" stroke={accent} strokeWidth="1.7" strokeLinecap="round" />
      <circle cx="12" cy="12" r="1.5" fill={accent} />
    </>
  );
}
