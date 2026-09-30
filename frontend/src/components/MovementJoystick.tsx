"use client";

import { useCallback, useEffect, useRef, useState } from "react";

const RADIUS = 40;
const DEAD_ZONE = 0.22;

function emitDirection(x: number, y: number, fine: boolean = false) {
  window.dispatchEvent(new CustomEvent("gu_world_direction", { detail: { x, y, fine } }));
}

function quantizeDirection(x: number, y: number) {
  const sector = Math.round(Math.atan2(y, x) / (Math.PI / 4));
  const angle = sector * (Math.PI / 4);
  return {
    x: Math.abs(Math.cos(angle)) < 0.5 ? 0 : Math.sign(Math.cos(angle)),
    y: Math.abs(Math.sin(angle)) < 0.5 ? 0 : Math.sign(Math.sin(angle)),
  };
}

export function MovementJoystick() {
  const baseRef = useRef<HTMLDivElement>(null);
  const activePointerRef = useRef<number | null>(null);
  const emittedDirectionRef = useRef({ x: 0, y: 0 });
  const lastFineRef = useRef(false);
  const [knob, setKnob] = useState({ x: 0, y: 0 });

  const emitChangedDirection = useCallback((x: number, y: number, fine: boolean = false) => {
    const next = x === 0 && y === 0 ? { x: 0, y: 0 } : quantizeDirection(x, y);
    if (next.x === emittedDirectionRef.current.x && next.y === emittedDirectionRef.current.y && lastFineRef.current === fine) return;
    emittedDirectionRef.current = next;
    lastFineRef.current = fine;
    emitDirection(next.x, next.y, fine);
  }, []);

  const update = useCallback((clientX: number, clientY: number) => {
    const rect = baseRef.current?.getBoundingClientRect();
    if (!rect) return;
    const rawX = clientX - (rect.left + rect.width / 2);
    const rawY = clientY - (rect.top + rect.height / 2);
    const distance = Math.hypot(rawX, rawY);
    const scale = distance > RADIUS ? RADIUS / distance : 1;
    const x = rawX * scale;
    const y = rawY * scale;
    setKnob({ x, y });

    if (distance < RADIUS * DEAD_ZONE) {
      emitChangedDirection(0, 0, false);
    } else {
      const fine = distance < RADIUS * 0.65;
      emitChangedDirection(x, y, fine);
    }
  }, [emitChangedDirection]);

  const release = useCallback((pointerId?: number) => {
    if (pointerId !== undefined && activePointerRef.current !== null && activePointerRef.current !== pointerId) return;
    activePointerRef.current = null;
    setKnob({ x: 0, y: 0 });
    emitChangedDirection(0, 0, false);
  }, [emitChangedDirection]);

  useEffect(() => {
    const handleGlobalPointerUp = (event: PointerEvent) => {
      if (activePointerRef.current !== null && (event.pointerId === activePointerRef.current || event.pointerType === "touch")) {
        release();
      }
    };
    window.addEventListener("pointerup", handleGlobalPointerUp);
    window.addEventListener("pointercancel", handleGlobalPointerUp);
    return () => {
      window.removeEventListener("pointerup", handleGlobalPointerUp);
      window.removeEventListener("pointercancel", handleGlobalPointerUp);
    };
  }, [release]);

  return (
    <div className="absolute bottom-[calc(env(safe-area-inset-bottom)+5.5rem)] left-[calc(env(safe-area-inset-left)+1rem)] z-30 pointer-events-auto md:hidden">
      <div
        ref={baseRef}
        role="application"
        aria-label="Cần điều khiển di chuyển"
        className="relative h-24 w-24 touch-none select-none rounded-full border border-white/20 bg-zinc-950/55 shadow-2xl backdrop-blur-sm"
        onPointerDown={(event) => {
          activePointerRef.current = event.pointerId;
          event.currentTarget.setPointerCapture(event.pointerId);
          update(event.clientX, event.clientY);
        }}
        onPointerMove={(event) => {
          if (activePointerRef.current === event.pointerId) update(event.clientX, event.clientY);
        }}
        onPointerUp={(event) => release(event.pointerId)}
        onPointerCancel={(event) => release(event.pointerId)}
        onLostPointerCapture={() => release()}
      >
        <span className="absolute left-1/2 top-1 -translate-x-1/2 text-xs text-white/35">▲</span>
        <span className="absolute bottom-1 left-1/2 -translate-x-1/2 text-xs text-white/35">▼</span>
        <span className="absolute left-1 top-1/2 -translate-y-1/2 text-xs text-white/35">◀</span>
        <span className="absolute right-1 top-1/2 -translate-y-1/2 text-xs text-white/35">▶</span>
        <div
          className="absolute left-1/2 top-1/2 h-11 w-11 -ml-[22px] -mt-[22px] rounded-full border border-emerald-300/60 bg-emerald-500/35 shadow-lg shadow-emerald-950/60"
          style={{ transform: `translate3d(${knob.x}px, ${knob.y}px, 0)` }}
        />
      </div>
    </div>
  );
}
