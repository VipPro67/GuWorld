"use client";

import React, { useRef, useState, useCallback, useEffect } from "react";

interface VirtualJoystickProps {
  onMove: (vector: { x: number; y: number }) => void;
  className?: string;
}

const MAX_RADIUS = 42; // Max pixel offset from center
const DEAD_ZONE = 4;   // Min pixel offset to register movement

export const VirtualJoystick: React.FC<VirtualJoystickProps> = ({ onMove, className = "" }) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const touchIdRef = useRef<number | null>(null);
  const [knobPos, setKnobPos] = useState<{ x: number; y: number }>({ x: 0, y: 0 });
  const [isActive, setIsActive] = useState(false);

  const updatePosition = useCallback((clientX: number, clientY: number) => {
    if (!containerRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;

    const dx = clientX - centerX;
    const dy = clientY - centerY;
    const distance = Math.hypot(dx, dy);

    if (distance < DEAD_ZONE) {
      setKnobPos({ x: 0, y: 0 });
      onMove({ x: 0, y: 0 });
      return;
    }

    const clampedDist = Math.min(distance, MAX_RADIUS);
    const angle = Math.atan2(dy, dx);
    const knobX = Math.cos(angle) * clampedDist;
    const knobY = Math.sin(angle) * clampedDist;

    setKnobPos({ x: knobX, y: knobY });

    // Output normalized vector (-1 to 1)
    const normX = dx / Math.max(distance, MAX_RADIUS);
    const normY = dy / Math.max(distance, MAX_RADIUS);
    onMove({ x: normX, y: normY });
  }, [onMove]);

  const handleTouchStart = useCallback((e: React.TouchEvent<HTMLDivElement>) => {
    e.preventDefault();
    if (touchIdRef.current !== null) return;
    const touch = e.changedTouches[0];
    touchIdRef.current = touch.identifier;
    setIsActive(true);
    updatePosition(touch.clientX, touch.clientY);
  }, [updatePosition]);

  const handleTouchMove = useCallback((e: React.TouchEvent<HTMLDivElement>) => {
    e.preventDefault();
    if (touchIdRef.current === null) return;
    for (let i = 0; i < e.changedTouches.length; i++) {
      const touch = e.changedTouches[i];
      if (touch.identifier === touchIdRef.current) {
        updatePosition(touch.clientX, touch.clientY);
        break;
      }
    }
  }, [updatePosition]);

  const handleTouchEnd = useCallback((e: React.TouchEvent<HTMLDivElement>) => {
    e.preventDefault();
    if (touchIdRef.current === null) return;
    for (let i = 0; i < e.changedTouches.length; i++) {
      const touch = e.changedTouches[i];
      if (touch.identifier === touchIdRef.current) {
        touchIdRef.current = null;
        setIsActive(false);
        setKnobPos({ x: 0, y: 0 });
        onMove({ x: 0, y: 0 });
        break;
      }
    }
  }, [onMove]);

  // Pointer fallback for testing or mouse drag
  const handlePointerDown = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    if (e.pointerType === "touch") return; // Touch is handled by touch events
    (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
    setIsActive(true);
    updatePosition(e.clientX, e.clientY);
  }, [updatePosition]);

  const handlePointerMove = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    if (e.pointerType === "touch" || !isActive) return;
    updatePosition(e.clientX, e.clientY);
  }, [isActive, updatePosition]);

  const handlePointerUp = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    if (e.pointerType === "touch") return;
    setIsActive(false);
    setKnobPos({ x: 0, y: 0 });
    onMove({ x: 0, y: 0 });
  }, [onMove]);

  useEffect(() => {
    return () => {
      onMove({ x: 0, y: 0 });
    };
  }, [onMove]);

  return (
    <div
      ref={containerRef}
      role="application"
      aria-label="Cần điều khiển di chuyển"
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onTouchCancel={handleTouchEnd}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      className={`relative w-28 h-28 select-none touch-none rounded-full flex items-center justify-center border-2 transition-colors duration-150 ${
        isActive
          ? "border-[#f4d894]/80 bg-[#16100b]/80 shadow-[0_0_20px_rgba(244,216,148,0.25)]"
          : "border-[#a88952]/50 bg-[#16100b]/55 shadow-[0_4px_16px_rgba(0,0,0,0.6)]"
      } backdrop-blur-sm ${className}`}
    >
      {/* 4 Cardinal notches (Bắc, Nam, Tây, Đông) */}
      <div className="absolute top-1 left-1/2 -translate-x-1/2 text-[8px] font-sans font-bold text-[#c9a35b]/70 select-none pointer-events-none">▲</div>
      <div className="absolute bottom-1 left-1/2 -translate-x-1/2 text-[8px] font-sans font-bold text-[#c9a35b]/70 select-none pointer-events-none">▼</div>
      <div className="absolute left-1 top-1/2 -translate-y-1/2 text-[8px] font-sans font-bold text-[#c9a35b]/70 select-none pointer-events-none">◄</div>
      <div className="absolute right-1 top-1/2 -translate-y-1/2 text-[8px] font-sans font-bold text-[#c9a35b]/70 select-none pointer-events-none">►</div>

      {/* Outer crosshairs */}
      <div className="absolute inset-0 flex items-center justify-center pointer-events-none opacity-25">
        <div className="w-full h-px bg-[#e7d3a2]" />
        <div className="h-full w-px bg-[#e7d3a2] absolute" />
      </div>

      {/* Inner stick / knob */}
      <div
        className="w-12 h-12 rounded-full border-2 border-[#f5dfab] bg-gradient-to-br from-[#5a3e26] via-[#382312] to-[#1a1007] shadow-[0_2px_8px_rgba(0,0,0,0.8),inset_0_1px_2px_rgba(255,255,255,0.4)] flex items-center justify-center transition-transform duration-75 will-change-transform pointer-events-none"
        style={{
          transform: `translate3d(${knobPos.x}px, ${knobPos.y}px, 0)`,
        }}
      >
        <div className="w-4 h-4 rounded-full border border-[#f5dfab]/60 bg-gradient-to-tr from-[#a88241] to-[#f4d894] shadow-[0_0_6px_rgba(244,216,148,0.5)]" />
      </div>
    </div>
  );
};
