"use client";

import { useCallback, useEffect, useRef, useState } from "react";

const RADIUS = 40;
const DEAD_ZONE = 0.22;
const ACTIVATION_DISTANCE = 8;

type Point = { x: number; y: number };

function emitDirection(x: number, y: number, fine = false) {
  window.dispatchEvent(new CustomEvent("gu_world_direction", { detail: { x, y, fine } }));
}

/** Snap to 8 cardinal/diagonal directions (used only for very gentle nudges). */
function quantizeDirection(x: number, y: number) {
  const angle = Math.round(Math.atan2(y, x) / (Math.PI / 4)) * (Math.PI / 4);
  return {
    x: Math.abs(Math.cos(angle)) < 0.5 ? 0 : Math.sign(Math.cos(angle)),
    y: Math.abs(Math.sin(angle)) < 0.5 ? 0 : Math.sign(Math.sin(angle)),
  };
}

export function MovementJoystick() {
  const activePointerRef = useRef<number | null>(null);
  const originRef = useRef<Point | null>(null);
  const movedRef = useRef(false);
  const emittedDirectionRef = useRef({ x: 0, y: 0 });
  const lastFineRef = useRef(false);
  const [origin, setOrigin] = useState<Point | null>(null);
  const [knob, setKnob] = useState<Point>({ x: 0, y: 0 });

  const emitChangedDirection = useCallback((x: number, y: number, fine = false) => {
    // Fine (gentle nudge): snap to 8 directions to prevent micro-drift.
    // Full push: send the raw analog vector so diagonal movement is smooth
    // instead of being locked to 45° increments.
    let next: { x: number; y: number };
    if (x === 0 && y === 0) {
      next = { x: 0, y: 0 };
    } else if (fine) {
      next = quantizeDirection(x, y);
    } else {
      const distance = Math.hypot(x, y);
      next = { x: x / distance, y: y / distance };
    }
    if (next.x === emittedDirectionRef.current.x && next.y === emittedDirectionRef.current.y && lastFineRef.current === fine) return;
    emittedDirectionRef.current = next;
    lastFineRef.current = fine;
    emitDirection(next.x, next.y, fine);
  }, []);

  const update = useCallback((clientX: number, clientY: number) => {
    const start = originRef.current;
    if (!start) return;
    const rawX = clientX - start.x;
    const rawY = clientY - start.y;
    const distance = Math.hypot(rawX, rawY);
    if (distance >= ACTIVATION_DISTANCE) movedRef.current = true;
    const scale = distance > RADIUS ? RADIUS / distance : 1;
    const x = rawX * scale;
    const y = rawY * scale;
    setKnob({ x, y });

    if (distance < RADIUS * DEAD_ZONE) emitChangedDirection(0, 0);
    else emitChangedDirection(rawX, rawY, distance < RADIUS * 0.65);
  }, [emitChangedDirection]);

  const release = useCallback(() => {
    activePointerRef.current = null;
    originRef.current = null;
    setOrigin(null);
    setKnob({ x: 0, y: 0 });
    emitChangedDirection(0, 0);
  }, [emitChangedDirection]);

  useEffect(() => {
    const onPointerDown = (event: PointerEvent) => {
      if (event.pointerType !== "touch" || !(event.target instanceof HTMLCanvasElement)) return;
      if (activePointerRef.current !== null) return;

      const point = { x: event.clientX, y: event.clientY };
      activePointerRef.current = event.pointerId;
      originRef.current = point;
      movedRef.current = false;
      setOrigin(point);
      setKnob({ x: 0, y: 0 });
    };

    const onPointerMove = (event: PointerEvent) => {
      if (activePointerRef.current !== event.pointerId) return;
      update(event.clientX, event.clientY);
    };

    const onPointerEnd = (event: PointerEvent) => {
      if (activePointerRef.current === event.pointerId) release();
    };

    window.addEventListener("pointerdown", onPointerDown, true);
    window.addEventListener("pointermove", onPointerMove, true);
    window.addEventListener("pointerup", onPointerEnd, true);
    window.addEventListener("pointercancel", onPointerEnd, true);
    return () => {
      window.removeEventListener("pointerdown", onPointerDown, true);
      window.removeEventListener("pointermove", onPointerMove, true);
      window.removeEventListener("pointerup", onPointerEnd, true);
      window.removeEventListener("pointercancel", onPointerEnd, true);
      emitDirection(0, 0);
    };
  }, [release, update]);

  if (!origin) return null;

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed z-30 h-24 w-24 -translate-x-1/2 -translate-y-1/2 md:hidden"
      style={{ left: origin.x, top: origin.y }}
    >
      <div className={`relative h-full w-full rounded-full border border-white/30 bg-zinc-950/55 shadow-2xl backdrop-blur-sm transition-opacity ${knob.x !== 0 || knob.y !== 0 ? "opacity-100" : "opacity-60"}`}>
        <span className="absolute left-1/2 top-1 -translate-x-1/2 text-xs text-white/45">▲</span>
        <span className="absolute bottom-1 left-1/2 -translate-x-1/2 text-xs text-white/45">▼</span>
        <span className="absolute left-1 top-1/2 -translate-y-1/2 text-xs text-white/45">◀</span>
        <span className="absolute right-1 top-1/2 -translate-y-1/2 text-xs text-white/45">▶</span>
        <div
          className="absolute left-1/2 top-1/2 -ml-[22px] -mt-[22px] h-11 w-11 rounded-full border border-emerald-300/70 bg-emerald-500/45 shadow-lg shadow-emerald-950/60"
          style={{ transform: `translate3d(${knob.x}px, ${knob.y}px, 0)` }}
        />
      </div>
    </div>
  );
}
