"use client";

import React, { useRef, useState, useCallback, useEffect } from "react";
import { MobaSkillSlot } from "./MobaSkillSlot";
import type { PvPSkill } from "@/store/slices/pvpSlice";

interface MobaControlsProps {
  skills: PvPSkill[];
  basicAttackSkill?: PvPSkill;
  guDictionary: Record<string, { name?: string; image_url?: string }>;
  cooldowns: Record<string, number>;
  selectedIndex: number;
  aimingSkillIndex: number | null;
  isCancelHovered: boolean;
  onMove: (dx: number, dy: number) => void;
  onBasicAttack?: () => void;
  onSkillTap: (index: number, skill: PvPSkill) => void;
  onAimStart: (index: number, skill: PvPSkill, clientX: number, clientY: number) => void;
  onAimMove: (clientX: number, clientY: number) => void;
  onAimEnd: (clientX: number, clientY: number) => void;
  cancelZoneRef: React.RefObject<HTMLDivElement | null>;
}

interface SkillSlotPosition {
  bottom: number;
  right: number;
  size: number;
}

/**
 * Portrait configuration:
 * Optimally compact 2-column arc cluster positioned strictly on the right side
 * (right: 18px to 148px, bottom: 18px to 230px).
 * Leaves the entire left half completely unobstructed for the virtual joystick,
 * and does not reach upward into the arena playing area.
 */
const PORTRAIT_CONFIGS: Record<number, SkillSlotPosition[]> = {
  1: [{ bottom: 20, right: 104, size: 50 }],
  2: [
    { bottom: 20, right: 104, size: 48 },
    { bottom: 82, right: 86, size: 50 },
  ],
  3: [
    { bottom: 20, right: 104, size: 48 },
    { bottom: 78, right: 94, size: 48 },
    { bottom: 100, right: 22, size: 54 },
  ],
  4: [
    { bottom: 20, right: 104, size: 48 },
    { bottom: 76, right: 94, size: 48 },
    { bottom: 98, right: 22, size: 48 },
    { bottom: 148, right: 44, size: 54 }, // Ultimate
  ],
  5: [
    { bottom: 18, right: 104, size: 46 },
    { bottom: 72, right: 96, size: 46 },
    { bottom: 96, right: 22, size: 46 },
    { bottom: 124, right: 86, size: 46 },
    { bottom: 154, right: 30, size: 52 }, // Ultimate
  ],
  6: [
    // Skill 1: Bottom left of Attack
    { bottom: 18, right: 104, size: 46 },
    // Skill 2: Above Skill 1
    { bottom: 72, right: 98, size: 46 },
    // Skill 3: Above Attack
    { bottom: 96, right: 22, size: 46 },
    // Skill 4: Above Skill 2
    { bottom: 124, right: 98, size: 46 },
    // Skill 5: Above Skill 3
    { bottom: 148, right: 22, size: 46 },
    // Skill 6 (Ultimate): Apex at the top, crowned between columns
    { bottom: 178, right: 60, size: 52 },
  ],
};

/**
 * Landscape configuration:
 * Wide sweeping MOBA fan arc curving around the main attack button.
 */
const LANDSCAPE_CONFIGS: Record<number, SkillSlotPosition[]> = {
  1: [{ bottom: 20, right: 110, size: 56 }],
  2: [
    { bottom: 20, right: 140, size: 54 },
    { bottom: 88, right: 88, size: 54 },
  ],
  3: [
    { bottom: 20, right: 165, size: 54 },
    { bottom: 85, right: 115, size: 54 },
    { bottom: 145, right: 20, size: 60 },
  ],
  4: [
    { bottom: 18, right: 180, size: 52 },
    { bottom: 72, right: 135, size: 52 },
    { bottom: 125, right: 82, size: 52 },
    { bottom: 172, right: 18, size: 58 },
  ],
  5: [
    { bottom: 16, right: 195, size: 50 },
    { bottom: 62, right: 150, size: 50 },
    { bottom: 108, right: 106, size: 50 },
    { bottom: 152, right: 62, size: 50 },
    { bottom: 190, right: 16, size: 56 },
  ],
  6: [
    { bottom: 16, right: 210, size: 48 },
    { bottom: 58, right: 168, size: 48 },
    { bottom: 100, right: 126, size: 48 },
    { bottom: 140, right: 86, size: 48 },
    { bottom: 178, right: 48, size: 48 },
    { bottom: 216, right: 14, size: 54 },
  ],
};

const getArcPosition = (i: number, total: number, isPortrait: boolean): SkillSlotPosition => {
  const configs = isPortrait ? PORTRAIT_CONFIGS : LANDSCAPE_CONFIGS;
  const list = configs[total] ?? configs[6] ?? configs[4];
  return list[i] ?? { bottom: 20 + i * 40, right: 20 + i * 30, size: 48 };
};

export const MobaControls: React.FC<MobaControlsProps> = ({
  skills,
  basicAttackSkill,
  guDictionary,
  cooldowns,
  selectedIndex,
  aimingSkillIndex,
  isCancelHovered,
  onMove,
  onBasicAttack,
  onSkillTap,
  onAimStart,
  onAimMove,
  onAimEnd,
  cancelZoneRef,
}) => {
  // Screen orientation detection for dynamic layout adaptation
  const [isPortrait, setIsPortrait] = useState(() => {
    if (typeof window !== "undefined") {
      return window.innerHeight > window.innerWidth;
    }
    return true;
  });

  useEffect(() => {
    const handleResize = () => {
      setIsPortrait(window.innerHeight > window.innerWidth);
    };
    window.addEventListener("resize", handleResize);
    window.addEventListener("orientationchange", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("orientationchange", handleResize);
    };
  }, []);

  const stickRadius = isPortrait ? 42 : 50;

  // ── Joystick state ────────────────────────────────────────────────────────
  const stickBaseRef = useRef<HTMLDivElement>(null);
  const stickKnobRef = useRef<HTMLDivElement>(null);
  const [stickActive, setStickActive] = useState(false);
  const [knobOffset, setKnobOffset] = useState({ x: 0, y: 0 });
  const stickPointerId = useRef<number | null>(null);
  const stickCenter = useRef({ x: 0, y: 0 });

  const handleStickDown = useCallback(
    (e: React.PointerEvent) => {
      if (stickPointerId.current !== null) return;
      stickPointerId.current = e.pointerId;
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);

      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      stickCenter.current = {
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2,
      };

      setStickActive(true);
      const dx = e.clientX - stickCenter.current.x;
      const dy = e.clientY - stickCenter.current.y;
      const len = Math.hypot(dx, dy);
      const clampedLen = Math.min(len, stickRadius);
      const angle = Math.atan2(dy, dx);
      const nx = Math.cos(angle) * clampedLen;
      const ny = Math.sin(angle) * clampedLen;

      setKnobOffset({ x: nx, y: ny });
      if (len > 5) {
        onMove(nx / stickRadius, ny / stickRadius);
      } else {
        onMove(0, 0);
      }
    },
    [onMove, stickRadius],
  );

  const handleStickMove = useCallback(
    (e: React.PointerEvent) => {
      if (stickPointerId.current !== e.pointerId) return;
      const dx = e.clientX - stickCenter.current.x;
      const dy = e.clientY - stickCenter.current.y;
      const len = Math.hypot(dx, dy);
      const clampedLen = Math.min(len, stickRadius);
      const angle = Math.atan2(dy, dx);
      const nx = Math.cos(angle) * clampedLen;
      const ny = Math.sin(angle) * clampedLen;

      setKnobOffset({ x: nx, y: ny });
      if (len > 5) {
        onMove(nx / stickRadius, ny / stickRadius);
      } else {
        onMove(0, 0);
      }
    },
    [onMove, stickRadius],
  );

  const handleStickUp = useCallback(
    (e: React.PointerEvent) => {
      if (stickPointerId.current !== e.pointerId) return;
      stickPointerId.current = null;
      setStickActive(false);
      setKnobOffset({ x: 0, y: 0 });
      onMove(0, 0);
    },
    [onMove],
  );

  // ── Skill Aiming Touch Tracker ─────────────────────────────────────────────
  const skillTouchRef = useRef<{
    pointerId: number;
    skillIndex: number;
    startX: number;
    startY: number;
    startTime: number;
    isAiming: boolean;
  } | null>(null);

  const handleSkillPointerDown = useCallback(
    (e: React.PointerEvent, index: number, skill: PvPSkill) => {
      e.stopPropagation();
      const remaining = cooldowns[skill.gu_id] ?? 0;
      if (remaining > 0) return;

      skillTouchRef.current = {
        pointerId: e.pointerId,
        skillIndex: index,
        startX: e.clientX,
        startY: e.clientY,
        startTime: performance.now(),
        isAiming: false,
      };

      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      onAimStart(index, skill, e.clientX, e.clientY);
    },
    [cooldowns, onAimStart],
  );

  const handleSkillPointerMove = useCallback(
    (e: React.PointerEvent) => {
      const touch = skillTouchRef.current;
      if (!touch || touch.pointerId !== e.pointerId) return;
      onAimMove(e.clientX, e.clientY);
    },
    [onAimMove],
  );

  const handleSkillPointerUp = useCallback(
    (e: React.PointerEvent, index: number, skill: PvPSkill) => {
      const touch = skillTouchRef.current;
      if (!touch || touch.pointerId !== e.pointerId) return;
      skillTouchRef.current = null;

      const elapsed = performance.now() - touch.startTime;
      const dist = Math.hypot(e.clientX - touch.startX, e.clientY - touch.startY);

      if (elapsed < 220 && dist < 14) {
        onSkillTap(index, skill);
      } else {
        onAimEnd(e.clientX, e.clientY);
      }
    },
    [onSkillTap, onAimEnd],
  );

  const attackSize = isPortrait ? 68 : 74;

  return (
    <div className="pointer-events-none absolute inset-0 z-30 select-none overflow-hidden touch-none">
      {/* ── Cancel Zone ────────────────────────────────────────────────────── */}
      {aimingSkillIndex !== null && (
        <div
          ref={cancelZoneRef}
          style={
            isPortrait
              ? { bottom: "242px", right: "20px" }
              : { top: "3.5rem", right: "1.5rem" }
          }
          className={`pointer-events-auto absolute flex flex-col items-center justify-center transition-all ${
            isCancelHovered
              ? "scale-105 shadow-[0_0_25px_rgba(239,68,68,0.9)] bg-rose-600/90 border-rose-300"
              : "shadow-[0_0_12px_rgba(239,68,68,0.4)] bg-rose-950/75 border-rose-500/60"
          } rounded-2xl border-2 px-4 py-2 backdrop-blur-md`}
        >
          <div className="flex items-center gap-1.5 text-xs font-bold text-rose-100">
            <span className="text-sm leading-none">✕</span>
            <span>{isCancelHovered ? "THẢ ĐỂ HỦY CHIÊU" : "KÉO LÊN ĐÂY ĐỂ HỦY"}</span>
          </div>
          <span className="text-[9px] text-rose-300/80">Thả ra ngoài để tiếp tục thi triển</span>
        </div>
      )}

      {/* ── Virtual Joystick (Bottom Left) ─────────────────────────────────── */}
      <div
        ref={stickBaseRef}
        onPointerDown={handleStickDown}
        onPointerMove={handleStickMove}
        onPointerUp={handleStickUp}
        onPointerCancel={handleStickUp}
        style={{
          bottom: isPortrait ? "20px" : "24px",
          left: isPortrait ? "18px" : "24px",
          width: isPortrait ? "124px" : "140px",
          height: isPortrait ? "124px" : "140px",
          opacity: stickActive ? 0.95 : 0.65,
        }}
        className="pointer-events-auto absolute flex items-center justify-center rounded-full border-2 border-emerald-500/35 bg-emerald-950/25 backdrop-blur-sm touch-none transition-opacity"
      >
        {/* Cardinal guidelines */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="h-full w-0.5 bg-emerald-500/15" />
          <div className="h-0.5 w-full bg-emerald-500/15" />
        </div>
        <div
          style={{ width: isPortrait ? "68px" : "78px", height: isPortrait ? "68px" : "78px" }}
          className="rounded-full border border-emerald-400/25 pointer-events-none"
        />

        {/* Joystick Knob */}
        <div
          ref={stickKnobRef}
          style={{
            transform: `translate(${knobOffset.x}px, ${knobOffset.y}px)`,
            width: isPortrait ? "46px" : "54px",
            height: isPortrait ? "46px" : "54px",
          }}
          className="relative flex items-center justify-center rounded-full border-2 border-emerald-300 bg-gradient-to-b from-emerald-400/80 to-emerald-700/90 shadow-[0_0_15px_rgba(52,211,153,0.5)] transition-transform duration-75 pointer-events-none"
        >
          <div className="h-3.5 w-3.5 rounded-full bg-white/70 shadow" />
        </div>
      </div>

      {/* ── Main Basic Attack Button (Bottom Right) ────────────────────────── */}
      {basicAttackSkill && (
        <div
          style={{
            position: "absolute",
            bottom: isPortrait ? "20px" : "22px",
            right: isPortrait ? "20px" : "22px",
          }}
          className="pointer-events-auto"
        >
          <MobaSkillSlot
            skill={basicAttackSkill}
            index={99}
            name="Đánh Thường"
            hotkey=""
            cooldownRemainingMs={cooldowns[basicAttackSkill.gu_id] ?? 0}
            isSelected={false}
            isAiming={false}
            shape="circle"
            size={attackSize}
            showDetails={false}
            onPointerDown={(e) => {
              e.stopPropagation();
              onBasicAttack?.();
            }}
          />
        </div>
      )}

      {/* ── Skill Cluster (Curving Around Main Attack) ─────────────────────── */}
      <div className="pointer-events-auto absolute inset-0 touch-none">
        {skills.map((skill, index) => {
          const remaining = cooldowns[skill.gu_id] ?? 0;
          const pos = getArcPosition(index, skills.length, isPortrait);
          const isAiming = aimingSkillIndex === index;
          const name = guDictionary[skill.gu_id]?.name || skill.gu_id.replace(/^GU_/, "").replace(/_/g, " ");
          const imageUrl = guDictionary[skill.gu_id]?.image_url;

          return (
            <div
              key={`${skill.gu_id}-${skill.rank}`}
              style={{
                position: "absolute",
                bottom: `${pos.bottom}px`,
                right: `${pos.right}px`,
              }}
            >
              <MobaSkillSlot
                skill={skill}
                index={index}
                name={name}
                imageUrl={imageUrl}
                hotkey=""
                cooldownRemainingMs={remaining}
                isSelected={selectedIndex === index}
                isAiming={isAiming}
                shape="circle"
                size={pos.size}
                showDetails={false}
                onPointerDown={(e) => handleSkillPointerDown(e, index, skill)}
                onPointerMove={handleSkillPointerMove}
                onPointerUp={(e) => handleSkillPointerUp(e, index, skill)}
                onPointerCancel={(e) => handleSkillPointerUp(e, index, skill)}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
};
