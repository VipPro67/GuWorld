"use client";

import React from "react";
import type { PvPSkill } from "@/store/slices/pvpSlice";

interface SkillArtBackgroundProps {
  skill: PvPSkill;
  name?: string;
  className?: string;
}

/**
 * Procedural MOBA skill artwork & element backgrounds.
 * Renders rich stylized backgrounds for Gu skills with atmospheric gradients,
 * elemental particles, runic glyphs, and high-impact VFX flair.
 */
export const SkillArtBackground: React.FC<SkillArtBackgroundProps> = ({ skill, name = "", className = "" }) => {
  const id = skill.gu_id.toUpperCase();

  // 0. Basic Attack (Đánh thường / Đao Trảm / Quyền)
  if (id.includes("BASIC_ATTACK") || name.includes("Đánh Thường") || name.includes("Công Kích")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="atk-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#7c2d12" />
            <stop offset="55%" stopColor="#361006" />
            <stop offset="100%" stopColor="#0a0301" />
          </radialGradient>
          <linearGradient id="atk-slash" x1="0%" y1="100%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#ea580c" stopOpacity="0.2" />
            <stop offset="40%" stopColor="#f59e0b" stopOpacity="0.9" />
            <stop offset="60%" stopColor="#fef08a" stopOpacity="1" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="1" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#atk-bg)" />
        {/* Dynamic martial strike slashes */}
        <path d="M15 85 L85 15" stroke="url(#atk-slash)" strokeWidth="4.5" strokeLinecap="round" />
        <path d="M85 85 L15 15" stroke="url(#atk-slash)" strokeWidth="2.5" strokeLinecap="round" opacity="0.75" />
        {/* Core strike burst */}
        <circle cx="50" cy="50" r="16" stroke="#fbbf24" strokeWidth="1.8" fill="#ea580c" fillOpacity="0.35" />
        <circle cx="50" cy="50" r="5" fill="#ffffff" />
        <circle cx="28" cy="40" r="1.5" fill="#fef08a" />
        <circle cx="72" cy="60" r="1.5" fill="#fef08a" />
      </svg>
    );
  }

  // 1. Blade Aura (Đao Khí / Kim)
  if (id.includes("BLADE") || id.includes("DAO") || id.includes("KIEM") || name.includes("Đao") || name.includes("Kiếm")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="blade-bg" cx="50%" cy="50%" r="65%">
            <stop offset="0%" stopColor="#451a03" />
            <stop offset="60%" stopColor="#1a0b02" />
            <stop offset="100%" stopColor="#080301" />
          </radialGradient>
          <linearGradient id="blade-glow" x1="0%" y1="100%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#d97706" stopOpacity="0.2" />
            <stop offset="50%" stopColor="#fbbf24" stopOpacity="0.9" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="1" />
          </linearGradient>
          <linearGradient id="blade-slash" x1="10%" y1="90%" x2="90%" y2="10%">
            <stop offset="0%" stopColor="#f59e0b" stopOpacity="0" />
            <stop offset="45%" stopColor="#fde68a" stopOpacity="0.95" />
            <stop offset="55%" stopColor="#ffffff" stopOpacity="1" />
            <stop offset="100%" stopColor="#f59e0b" stopOpacity="0" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#blade-bg)" />
        {/* Radiating slash sparks */}
        <path d="M15 85 L85 15" stroke="url(#blade-glow)" strokeWidth="6" strokeLinecap="round" filter="blur(1px)" />
        <path d="M12 88 L88 12" stroke="url(#blade-slash)" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M28 82 L72 38" stroke="#fef08a" strokeWidth="1.2" strokeOpacity="0.75" />
        <path d="M38 62 L82 18" stroke="#fef08a" strokeWidth="1.2" strokeOpacity="0.75" />
        {/* Energy burst crescent */}
        <path d="M30 40 Q50 48 70 30 Q54 62 30 40 Z" fill="#f59e0b" fillOpacity="0.3" />
        {/* Flying golden embers */}
        <circle cx="28" cy="30" r="1.5" fill="#fde68a" />
        <circle cx="72" cy="65" r="1.2" fill="#fde68a" />
        <circle cx="45" cy="22" r="1.8" fill="#ffffff" />
        <circle cx="60" cy="78" r="1" fill="#fde68a" />
      </svg>
    );
  }

  // 2. Water Arrow (Thủy Tiễn / Thủy)
  if (id.includes("WATER") || id.includes("THUY") || id.includes("TIEN") || name.includes("Thủy")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="water-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#083344" />
            <stop offset="65%" stopColor="#021a24" />
            <stop offset="100%" stopColor="#010c12" />
          </radialGradient>
          <linearGradient id="water-arrow-grad" x1="0%" y1="50%" x2="100%" y2="50%">
            <stop offset="0%" stopColor="#0284c7" stopOpacity="0.2" />
            <stop offset="70%" stopColor="#38bdf8" stopOpacity="0.9" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="1" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#water-bg)" />
        {/* Hydro swirling vortex rings */}
        <circle cx="50" cy="50" r="38" stroke="#0ea5e9" strokeWidth="1" strokeDasharray="6 8" strokeOpacity="0.35" />
        <circle cx="50" cy="50" r="26" stroke="#38bdf8" strokeWidth="1.2" strokeDasharray="4 6" strokeOpacity="0.45" />
        {/* Arrow projectile rushing right */}
        <path d="M16 50 Q36 46 64 50" stroke="#0284c7" strokeWidth="4" strokeOpacity="0.5" filter="blur(1px)" />
        <path d="M20 50 L78 50" stroke="url(#water-arrow-grad)" strokeWidth="2.5" strokeLinecap="round" />
        <path d="M62 36 L86 50 L62 64 L68 50 Z" fill="url(#water-arrow-grad)" />
        {/* Water wave splashes */}
        <path d="M30 42 Q40 32 54 44" stroke="#7dd3fc" strokeWidth="1.2" fill="none" strokeOpacity="0.6" />
        <path d="M34 58 Q46 68 58 56" stroke="#7dd3fc" strokeWidth="1.2" fill="none" strokeOpacity="0.6" />
        <circle cx="78" cy="38" r="1.5" fill="#bae6fd" />
        <circle cx="82" cy="62" r="1.2" fill="#bae6fd" />
      </svg>
    );
  }

  // 3. Star Fall (Tinh Lạc / Tinh Tú / Thiên Thạch)
  if (id.includes("STAR") || id.includes("TINH") || name.includes("Tinh") || name.includes("Sao")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="star-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#2e1065" />
            <stop offset="60%" stopColor="#14062e" />
            <stop offset="100%" stopColor="#060210" />
          </radialGradient>
          <linearGradient id="meteor-tail" x1="0%" y1="0%" x2="70%" y2="70%">
            <stop offset="0%" stopColor="#c084fc" stopOpacity="0" />
            <stop offset="60%" stopColor="#a855f7" stopOpacity="0.6" />
            <stop offset="90%" stopColor="#f3e8ff" stopOpacity="1" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="1" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#star-bg)" />
        {/* Constellation background */}
        <circle cx="20" cy="24" r="1.2" fill="#e9d5ff" opacity="0.8" />
        <circle cx="36" cy="18" r="1" fill="#e9d5ff" opacity="0.6" />
        <circle cx="78" cy="22" r="1.5" fill="#f5d0fe" opacity="0.9" />
        <circle cx="84" cy="42" r="1.2" fill="#e9d5ff" opacity="0.7" />
        <circle cx="22" cy="74" r="1" fill="#e9d5ff" opacity="0.5" />
        <line x1="20" y1="24" x2="36" y2="18" stroke="#a855f7" strokeWidth="0.7" strokeOpacity="0.4" />
        <line x1="78" y1="22" x2="84" y2="42" stroke="#a855f7" strokeWidth="0.7" strokeOpacity="0.4" />
        {/* Falling meteor */}
        <path d="M12 14 L68 70" stroke="url(#meteor-tail)" strokeWidth="4" strokeLinecap="round" filter="blur(0.8px)" />
        <path d="M16 18 L68 70" stroke="#ffffff" strokeWidth="1.6" strokeLinecap="round" />
        {/* Impact crater rings */}
        <ellipse cx="68" cy="70" rx="18" ry="10" stroke="#c084fc" strokeWidth="1.2" fill="none" strokeOpacity="0.7" />
        <ellipse cx="68" cy="70" rx="10" ry="5.5" stroke="#f3e8ff" strokeWidth="1.5" fill="#9333ea" fillOpacity="0.4" />
        <circle cx="68" cy="70" r="3" fill="#ffffff" />
      </svg>
    );
  }

  // 4. Soundwave (Ám Lăng / Âm Ba / Sóng Âm)
  if (id.includes("SOUND") || id.includes("AM_LANG") || id.includes("SONG") || name.includes("Ám Lăng") || name.includes("Âm")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="sound-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#3b0764" />
            <stop offset="60%" stopColor="#1e0338" />
            <stop offset="100%" stopColor="#080110" />
          </radialGradient>
        </defs>
        <rect width="100" height="100" fill="url(#sound-bg)" />
        {/* Radiating sonic wave arcs */}
        <path d="M22 66 A15 15 0 0 1 22 34" stroke="#d8b4fe" strokeWidth="1.5" fill="none" strokeLinecap="round" opacity="0.9" />
        <path d="M34 76 A30 30 0 0 1 34 24" stroke="#c084fc" strokeWidth="2" fill="none" strokeLinecap="round" opacity="0.8" />
        <path d="M48 86 A45 45 0 0 1 48 14" stroke="#a855f7" strokeWidth="2.5" fill="none" strokeLinecap="round" opacity="0.65" />
        <path d="M64 94 A62 62 0 0 1 64 6" stroke="#9333ea" strokeWidth="2" fill="none" strokeLinecap="round" strokeDasharray="3 3" opacity="0.5" />
        {/* Core sound source */}
        <circle cx="15" cy="50" r="6" fill="#f3e8ff" />
        <circle cx="15" cy="50" r="10" stroke="#e9d5ff" strokeWidth="1" fill="none" opacity="0.75" />
        {/* Resonance lines */}
        <line x1="15" y1="50" x2="85" y2="20" stroke="#c084fc" strokeWidth="0.8" strokeDasharray="2 3" opacity="0.4" />
        <line x1="15" y1="50" x2="85" y2="80" stroke="#c084fc" strokeWidth="0.8" strokeDasharray="2 3" opacity="0.4" />
      </svg>
    );
  }

  // 5. Space Door (Không Gian Môn / Hư Không / Dịch Chuyển)
  if (id.includes("SPACE") || id.includes("KHONG_GIAN") || skill.move_distance > 0 || name.includes("Không Gian") || name.includes("Dịch Chuyển")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="space-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#164e63" />
            <stop offset="50%" stopColor="#1e1b4b" />
            <stop offset="100%" stopColor="#040212" />
          </radialGradient>
          <linearGradient id="space-glow" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#22d3ee" />
            <stop offset="50%" stopColor="#818cf8" />
            <stop offset="100%" stopColor="#e879f9" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#space-bg)" />
        {/* Dimensional rift diamond & portal rings */}
        <polygon points="50,12 88,50 50,88 12,50" stroke="url(#space-glow)" strokeWidth="1.8" fill="#082f49" fillOpacity="0.4" />
        <polygon points="50,24 76,50 50,76 24,50" stroke="#67e8f9" strokeWidth="1.4" fill="#312e81" fillOpacity="0.5" />
        <circle cx="50" cy="50" r="14" stroke="#e879f9" strokeWidth="1.5" fill="#030712" fillOpacity="0.85" />
        {/* Singularity core */}
        <circle cx="50" cy="50" r="5" fill="#ffffff" filter="blur(0.5px)" />
        {/* Warp rays */}
        <line x1="50" y1="6" x2="50" y2="24" stroke="#a5f3fc" strokeWidth="1.2" />
        <line x1="50" y1="76" x2="50" y2="94" stroke="#a5f3fc" strokeWidth="1.2" />
        <line x1="6" y1="50" x2="24" y2="50" stroke="#a5f3fc" strokeWidth="1.2" />
        <line x1="76" y1="50" x2="94" y2="50" stroke="#a5f3fc" strokeWidth="1.2" />
      </svg>
    );
  }

  // 6. Law Shield (Pháp Thuẫn / Phòng Ngự / Khiên)
  if (id.includes("SHIELD") || id.includes("THUAN") || id.includes("LAW") || skill.cast_mode === "SELF" || name.includes("Thuẫn") || name.includes("Hộ")) {
    return (
      <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
        <defs>
          <radialGradient id="shield-bg" cx="50%" cy="50%" r="70%">
            <stop offset="0%" stopColor="#064e3b" />
            <stop offset="60%" stopColor="#022c22" />
            <stop offset="100%" stopColor="#01140e" />
          </radialGradient>
          <linearGradient id="shield-rim" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#34d399" />
            <stop offset="50%" stopColor="#10b981" />
            <stop offset="100%" stopColor="#047857" />
          </linearGradient>
        </defs>
        <rect width="100" height="100" fill="url(#shield-bg)" />
        {/* Hexagonal barrier matrix */}
        <polygon points="50,14 82,32 82,68 50,86 18,68 18,32" stroke="url(#shield-rim)" strokeWidth="2.2" fill="#047857" fillOpacity="0.25" />
        <polygon points="50,24 74,38 74,62 50,76 26,62 26,38" stroke="#6ee7b7" strokeWidth="1.2" fill="#065f46" fillOpacity="0.35" strokeDasharray="3 2" />
        {/* Core Bagua sacred node */}
        <circle cx="50" cy="50" r="11" stroke="#a7f3d0" strokeWidth="1.5" fill="#022c22" />
        <circle cx="50" cy="50" r="4" fill="#fef08a" />
        {/* Corner protective runes */}
        <circle cx="50" cy="14" r="2.5" fill="#6ee7b7" />
        <circle cx="82" cy="32" r="2.5" fill="#6ee7b7" />
        <circle cx="82" cy="68" r="2.5" fill="#6ee7b7" />
        <circle cx="50" cy="86" r="2.5" fill="#6ee7b7" />
        <circle cx="18" cy="68" r="2.5" fill="#6ee7b7" />
        <circle cx="18" cy="32" r="2.5" fill="#6ee7b7" />
      </svg>
    );
  }

  // 7. Generic Fallback by Shape / Cast Mode
  return (
    <svg viewBox="0 0 100 100" className={`absolute inset-0 h-full w-full ${className}`} aria-hidden>
      <defs>
        <radialGradient id="generic-bg" cx="50%" cy="50%" r="70%">
          <stop offset="0%" stopColor="#1e293b" />
          <stop offset="70%" stopColor="#0f172a" />
          <stop offset="100%" stopColor="#020617" />
        </radialGradient>
      </defs>
      <rect width="100" height="100" fill="url(#generic-bg)" />
      {/* Mystic runic circle */}
      <circle cx="50" cy="50" r="38" stroke="#38bdf8" strokeWidth="1" strokeDasharray="4 4" strokeOpacity="0.3" />
      <circle cx="50" cy="50" r="24" stroke="#818cf8" strokeWidth="1.2" strokeOpacity="0.4" />
      <circle cx="50" cy="50" r="6" fill="#38bdf8" fillOpacity="0.6" />
    </svg>
  );
};
