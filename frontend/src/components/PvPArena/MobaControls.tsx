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

const STICK_RADIUS = 50;

/**
 * Mobile MOBA HUD overlay:
 * - Left: Virtual analog movement stick with visual runic boundary
 * - Right: Main Attack button + Ergonomic radial skill arc cluster
 * - Top-Right: Cancel Zone for aborting skill cast
 */
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
      const clampedLen = Math.min(len, STICK_RADIUS);
      const angle = Math.atan2(dy, dx);
      const nx = Math.cos(angle) * clampedLen;
      const ny = Math.sin(angle) * clampedLen;

      setKnobOffset({ x: nx, y: ny });
      if (len > 5) {
        onMove(nx / STICK_RADIUS, ny / STICK_RADIUS);
      } else {
        onMove(0, 0);
      }
    },
    [onMove],
  );

  const handleStickMove = useCallback(
    (e: React.PointerEvent) => {
      if (stickPointerId.current !== e.pointerId) return;
      const dx = e.clientX - stickCenter.current.x;
      const dy = e.clientY - stickCenter.current.y;
      const len = Math.hypot(dx, dy);
      const clampedLen = Math.min(len, STICK_RADIUS);
      const angle = Math.atan2(dy, dx);
      const nx = Math.cos(angle) * clampedLen;
      const ny = Math.sin(angle) * clampedLen;

      setKnobOffset({ x: nx, y: ny });
      if (len > 5) {
        onMove(nx / STICK_RADIUS, ny / STICK_RADIUS);
      } else {
        onMove(0, 0);
      }
    },
    [onMove],
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

      // Start aiming immediately so reticle shows up
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
        // Quick tap: smart auto cast
        onSkillTap(index, skill);
      } else {
        // Drag release
        onAimEnd(e.clientX, e.clientY);
      }
    },
    [onSkillTap, onAimEnd],
  );

  // Radial fan positions for skills curving around the main attack button at (bottom: 22, right: 22)
  const getArcPosition = (i: number, total: number) => {
    const configs: Record<number, Array<{ bottom: number; right: number; size: number }>> = {
      4: [
        { bottom: 20, right: 195, size: 58 },
        { bottom: 85, right: 145, size: 58 },
        { bottom: 145, right: 85, size: 58 },
        { bottom: 195, right: 20, size: 66 }, // Ultimate
      ],
      5: [
        { bottom: 18, right: 215, size: 56 },
        { bottom: 74, right: 165, size: 56 },
        { bottom: 128, right: 118, size: 56 },
        { bottom: 180, right: 68, size: 56 },
        { bottom: 232, right: 18, size: 64 }, // Ultimate
      ],
      6: [
        { bottom: 16, right: 225, size: 54 },
        { bottom: 68, right: 175, size: 54 },
        { bottom: 122, right: 128, size: 54 },
        { bottom: 174, right: 84, size: 54 },
        { bottom: 224, right: 44, size: 54 },
        { bottom: 275, right: 14, size: 62 }, // Ultimate
      ],
    };

    const list = configs[total] ?? configs[6] ?? configs[4];
    return list[i] ?? { bottom: 20 + i * 45, right: 20 + i * 35, size: 56 };
  };

  const HOTKEYS = ["1", "2", "3", "4", "5", "6"];

  return (
    <div className="pointer-events-none absolute inset-0 z-30 select-none overflow-hidden touch-none">
      {/* ── Cancel Zone (Top Right) ────────────────────────────────────────── */}
      {aimingSkillIndex !== null && (
        <div
          ref={cancelZoneRef}
          className={`pointer-events-auto absolute top-14 right-6 flex flex-col items-center justify-center transition-all ${
            isCancelHovered
              ? "scale-110 shadow-[0_0_25px_rgba(239,68,68,0.9)] bg-rose-600/90 border-rose-300"
              : "shadow-[0_0_12px_rgba(239,68,68,0.4)] bg-rose-950/70 border-rose-500/60"
          } rounded-2xl border-2 px-5 py-3 backdrop-blur-md`}
        >
          <div className="flex items-center gap-1.5 text-xs font-bold text-rose-100">
            <span className="text-base leading-none">✕</span>
            <span>{isCancelHovered ? "THẢ ĐỂ HỦY CHIÊU" : "KÉO VÀO ĐÂY ĐỂ HỦY"}</span>
          </div>
          <span className="text-[10px] text-rose-300/80">Kéo ra ngoài để tiếp tục ngắm</span>
        </div>
      )}

      {/* ── Virtual Joystick (Bottom Left) ─────────────────────────────────── */}
      <div
        ref={stickBaseRef}
        onPointerDown={handleStickDown}
        onPointerMove={handleStickMove}
        onPointerUp={handleStickUp}
        onPointerCancel={handleStickUp}
        className="pointer-events-auto absolute bottom-6 left-6 flex h-36 w-36 items-center justify-center rounded-full border-2 border-emerald-500/30 bg-emerald-950/20 backdrop-blur-sm touch-none transition-opacity"
        style={{ opacity: stickActive ? 0.95 : 0.65 }}
      >
        {/* Cardinal guidelines */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="h-full w-0.5 bg-emerald-500/15" />
          <div className="h-0.5 w-full bg-emerald-500/15" />
        </div>
        <div className="h-20 w-20 rounded-full border border-emerald-400/25 pointer-events-none" />

        {/* Joystick Knob */}
        <div
          ref={stickKnobRef}
          style={{
            transform: `translate(${knobOffset.x}px, ${knobOffset.y}px)`,
          }}
          className="relative flex h-14 w-14 items-center justify-center rounded-full border-2 border-emerald-300 bg-gradient-to-b from-emerald-400/80 to-emerald-700/90 shadow-[0_0_15px_rgba(52,211,153,0.5)] transition-transform duration-75 pointer-events-none"
        >
          <div className="h-4 w-4 rounded-full bg-white/70 shadow" />
        </div>
      </div>

      {/* ── Main Basic Attack Button (Bottom Right) ────────────────────────── */}
      {basicAttackSkill && (
        <div
          style={{
            position: "absolute",
            bottom: "22px",
            right: "22px",
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
            size={76}
            showDetails={false}
            onPointerDown={(e) => {
              e.stopPropagation();
              onBasicAttack?.();
            }}
          />
        </div>
      )}

      {/* ── Radial Skill Arc (Curving Around Main Attack) ──────────────────── */}
      <div className="pointer-events-auto absolute inset-0 touch-none">
        {skills.map((skill, index) => {
          const remaining = cooldowns[skill.gu_id] ?? 0;
          const pos = getArcPosition(index, skills.length);
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
                hotkey={HOTKEYS[index] ?? ""}
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
