"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PvPArenaCanvas } from "./PvPArenaCanvas";
import { MobaSkillSlot } from "./MobaSkillSlot";
import { MobaControls } from "./MobaControls";
import { useGameStore } from "@/store/useGameStore";
import type { PvPControlState, PvPCastIntent, PvPParticipantState, PvPSkill } from "@/store/slices/pvpSlice";

interface PvPArenaViewProps {
  onClose: () => void;
}

/** Input frames per second. Matches the server's 20Hz tick with headroom. */
const INPUT_HZ = 30;
/** Radius in CSS pixels where a stick drag reaches full range. */
const STICK_RADIUS = 56;
/** Drag below this is treated as a tap rather than a direction. */
const STICK_DEADZONE = 10;

const KEY_VECTORS: Record<string, { dx: number; dy: number }> = {
  KeyW: { dx: 0, dy: -1 },
  ArrowUp: { dx: 0, dy: -1 },
  KeyS: { dx: 0, dy: 1 },
  ArrowDown: { dx: 0, dy: 1 },
  KeyA: { dx: -1, dy: 0 },
  ArrowLeft: { dx: -1, dy: 0 },
  KeyD: { dx: 1, dy: 0 },
  ArrowRight: { dx: 1, dy: 0 },
};

const PC_HOTKEY_SLOTS: Record<string, number> = {
  KeyQ: 0,
  Digit1: 0,
  KeyW: 1,
  Digit2: 1,
  KeyE: 2,
  Digit3: 2,
  KeyR: 3,
  Digit4: 3,
  KeyD: 4,
  Digit5: 4,
  KeyF: 5,
  Digit6: 5,
};

const HOTKEY_LABELS = ["Q", "W", "E", "R", "D", "F"];

const DEFAULT_BASIC_ATTACK: PvPSkill = {
  gu_id: "BASIC_ATTACK",
  rank: 1,
  cast_mode: "TARGET",
  shape: "SINGLE",
  radius: 0,
  length: 0,
  width: 0,
  angle_deg: 0,
  cast_range: 6.0,
  cast_time_ms: 80,
  cooldown_ms: 600,
  move_distance: 0,
  mana_cost: 0,
};

export const PvPArenaView: React.FC<PvPArenaViewProps> = ({ onClose }) => {
  const setup = useGameStore((state) => state.pvpSetup);
  const frame = useGameStore((state) => state.pvpFrame);
  const status = useGameStore((state) => state.pvpStatus);
  const error = useGameStore((state) => state.pvpError);
  const player = useGameStore((state) => state.player);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const startPracticePvP = useGameStore((state) => state.startPracticePvP);
  const stopPvP = useGameStore((state) => state.stopPvP);
  const sendPvPInput = useGameStore((state) => state.sendPvPInput);

  const [uiMode, setUiMode] = useState<"mobile" | "pc">(() => {
    if (typeof window !== "undefined") {
      return window.matchMedia("(pointer: coarse)").matches || window.innerWidth < 800 ? "mobile" : "pc";
    }
    return "pc";
  });

  const controlRef = useRef<PvPControlState>({
    move: { dx: 0, dy: 0 },
    aim: { x: 0, y: 0 },
    skill: null,
    aiming: false,
    cancelCast: false,
    view: { scale: 1, offsetX: 0, offsetY: 0 },
  });

  const pressedRef = useRef<Set<string>>(new Set());
  const activeKeyRef = useRef<string | null>(null);
  const seqRef = useRef(0);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [aimingSkillIndex, setAimingSkillIndex] = useState<number | null>(null);
  const [isCancelHovered, setIsCancelHovered] = useState(false);

  const stageRef = useRef<HTMLDivElement>(null);
  const cancelZoneRef = useRef<HTMLDivElement>(null);
  const aimDragRef = useRef<{ pointerId?: number; anchorX: number; anchorY: number; skill: PvPSkill } | null>(null);

  const [hintMessage, setHintMessage] = useState<string | null>(null);
  const hintTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const showHint = useCallback((msg: string) => {
    setHintMessage(msg);
    if (hintTimeoutRef.current) clearTimeout(hintTimeoutRef.current);
    hintTimeoutRef.current = setTimeout(() => {
      setHintMessage(null);
    }, 1800);
  }, []);

  const localPlayerId = player?.id ?? "";
  const rawKit = useMemo<PvPSkill[]>(() => {
    const entry = setup?.players.find((candidate) => candidate.id === localPlayerId);
    return entry?.kit ?? [];
  }, [setup, localPlayerId]);

  const basicAttackSkill = useMemo<PvPSkill>(() => {
    return rawKit.find((s) => s.gu_id === "BASIC_ATTACK") ?? DEFAULT_BASIC_ATTACK;
  }, [rawKit]);

  const activeSkills = useMemo<PvPSkill[]>(() => {
    const list = rawKit.filter((s) => s.gu_id !== "BASIC_ATTACK");
    return list.length > 0 ? list : rawKit;
  }, [rawKit]);

  const skillName = useCallback(
    (skill: PvPSkill) => {
      if (skill.gu_id === "BASIC_ATTACK") return "Đánh Thường";
      return guDictionary[skill.gu_id]?.name || skill.gu_id.replace(/^GU_/, "").replace(/_/g, " ");
    },
    [guDictionary],
  );

  useEffect(() => {
    controlRef.current.skill = activeSkills[selectedIndex] ?? null;
  }, [activeSkills, selectedIndex]);

  const nextSeq = useCallback(() => {
    seqRef.current += 1;
    return seqRef.current;
  }, []);

  const currentFrame = useCallback((): ReturnType<typeof useGameStore.getState>["pvpFrame"] => useGameStore.getState().pvpFrame, []);

  const localPosition = useCallback((): { x: number; y: number } => {
    const self = currentFrame()?.participants.find((candidate) => candidate.id === localPlayerId);
    return { x: self?.x ?? 0, y: self?.y ?? 0 };
  }, [currentFrame, localPlayerId]);

  const sendCast = useCallback(
    (skill: PvPSkill, aim: { x: number; y: number }, targetOverride?: string) => {
      const self = currentFrame()?.participants.find((p) => p.id === localPlayerId);
      const remaining = self?.cd_ms?.[skill.gu_id] ?? 0;
      if (remaining > 0) return;

      const origin = localPosition();
      const intent: PvPCastIntent = { gu_id: skill.gu_id, rank: skill.rank, mode: skill.cast_mode };

      if (skill.cast_mode === "DIRECTION") {
        const dx = aim.x - origin.x;
        const dy = aim.y - origin.y;
        const length = Math.hypot(dx, dy);
        if (length > 0.05) {
          intent.ax = dx / length;
          intent.ay = dy / length;
        } else {
          const facing = self?.facing ?? 0;
          intent.ax = Math.cos(facing);
          intent.ay = Math.sin(facing);
        }
      } else if (skill.cast_mode === "GROUND") {
        const dx = aim.x - origin.x;
        const dy = aim.y - origin.y;
        const dist = Math.hypot(dx, dy);
        if (dist > skill.cast_range) {
          const ratio = skill.cast_range / (dist || 1);
          intent.ax = origin.x + dx * ratio;
          intent.ay = origin.y + dy * ratio;
        } else {
          intent.ax = aim.x;
          intent.ay = aim.y;
        }
      } else if (skill.cast_mode === "TARGET") {
        const target = targetOverride
          ? currentFrame()?.participants.find((p) => p.id === targetOverride)
          : nearestOpponent(currentFrame()?.participants ?? [], localPlayerId, aim);
        if (!target) return;

        const distToTarget = Math.hypot(target.x - origin.x, target.y - origin.y);
        if (distToTarget > skill.cast_range) {
          showHint(`Mục tiêu ngoài tầm xuất chiêu (${Math.round(distToTarget)} / ${skill.cast_range} ô)`);
          return;
        }
        intent.target_id = target.id;
      }

      // Send intent immediately along with current movement intent
      const control = controlRef.current;
      sendPvPInput({
        seq: nextSeq(),
        move: { dx: control.move.dx, dy: control.move.dy },
        casts: [intent],
      });
    },
    [currentFrame, localPlayerId, localPosition, nextSeq, sendPvPInput, showHint],
  );

  /**
   * Smart auto-cast helper:
   * Instantly targets nearest opponent in range or casts in facing direction.
   */
  const smartAutoCast = useCallback(
    (skill: PvPSkill) => {
      const origin = localPosition();
      const self = currentFrame()?.participants.find((p) => p.id === localPlayerId);
      const facing = self?.facing ?? 0;
      const opponent = nearestOpponent(currentFrame()?.participants ?? [], localPlayerId, origin);

      if (skill.cast_mode === "SELF") {
        sendCast(skill, origin);
        return;
      }

      if (skill.cast_mode === "TARGET") {
        if (opponent) {
          sendCast(skill, { x: opponent.x, y: opponent.y }, opponent.id);
        }
        return;
      }

      if (skill.cast_mode === "DIRECTION") {
        if (opponent) {
          sendCast(skill, { x: opponent.x, y: opponent.y });
        } else {
          const reach = skill.length || skill.cast_range || 10;
          sendCast(skill, {
            x: origin.x + Math.cos(facing) * reach,
            y: origin.y + Math.sin(facing) * reach,
          });
        }
        return;
      }

      if (skill.cast_mode === "GROUND") {
        if (opponent) {
          const dx = opponent.x - origin.x;
          const dy = opponent.y - origin.y;
          const dist = Math.hypot(dx, dy);
          if (dist <= skill.cast_range) {
            sendCast(skill, { x: opponent.x, y: opponent.y });
          } else {
            const ratio = skill.cast_range / (dist || 1);
            sendCast(skill, { x: origin.x + dx * ratio, y: origin.y + dy * ratio });
          }
        } else {
          const reach = Math.min(skill.cast_range, 8);
          sendCast(skill, {
            x: origin.x + Math.cos(facing) * reach,
            y: origin.y + Math.sin(facing) * reach,
          });
        }
      }
    },
    [currentFrame, localPlayerId, localPosition, sendCast],
  );

  const performBasicAttack = useCallback(() => {
    const self = currentFrame()?.participants.find((p) => p.id === localPlayerId);
    const remaining = self?.cd_ms?.[basicAttackSkill.gu_id] ?? 0;
    if (remaining > 0) return;
    smartAutoCast(basicAttackSkill);
  }, [basicAttackSkill, currentFrame, localPlayerId, smartAutoCast]);

  // Movement input loop.
  useEffect(() => {
    const timer = setInterval(() => {
      const control = controlRef.current;
      sendPvPInput({ seq: nextSeq(), move: { dx: control.move.dx, dy: control.move.dy } });
    }, 1000 / INPUT_HZ);
    return () => clearInterval(timer);
  }, [nextSeq, sendPvPInput]);

  // Keyboard movement and MOBA skill hotkeys (desktop QWERDF / 1-6).
  useEffect(() => {
    const recomputeMove = () => {
      let dx = 0;
      let dy = 0;
      for (const code of pressedRef.current) {
        const vector = KEY_VECTORS[code];
        if (vector) {
          dx += vector.dx;
          dy += vector.dy;
        }
      }
      const length = Math.hypot(dx, dy);
      controlRef.current.move = length > 0 ? { dx: dx / length, dy: dy / length } : { dx: 0, dy: 0 };
    };

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.repeat) return;

      // WASD / Arrows
      if (KEY_VECTORS[event.code]) {
        pressedRef.current.add(event.code);
        recomputeMove();
        event.preventDefault();
        return;
      }

      // Space: Basic attack
      if (event.code === "Space") {
        performBasicAttack();
        event.preventDefault();
        return;
      }

      // ESC: cancel aiming
      if (event.code === "Escape") {
        controlRef.current.aiming = false;
        controlRef.current.cancelCast = false;
        setAimingSkillIndex(null);
        activeKeyRef.current = null;
        return;
      }

      // Skill hotkeys QWERDF or 123456
      const slotIndex = PC_HOTKEY_SLOTS[event.code];
      if (slotIndex !== undefined && slotIndex < activeSkills.length) {
        const skill = activeSkills[slotIndex];
        if (!skill) return;

        const self = currentFrame()?.participants.find((p) => p.id === localPlayerId);
        const remaining = self?.cd_ms?.[skill.gu_id] ?? 0;
        if (remaining > 0) return;

        setSelectedIndex(slotIndex);
        controlRef.current.skill = skill;

        if (skill.cast_mode === "SELF") {
          sendCast(skill, controlRef.current.aim);
          return;
        }

        // Start aiming towards mouse cursor
        controlRef.current.aiming = true;
        controlRef.current.cancelCast = false;
        setAimingSkillIndex(slotIndex);
        activeKeyRef.current = event.code;
        event.preventDefault();
      }
    };

    const onKeyUp = (event: KeyboardEvent) => {
      if (KEY_VECTORS[event.code]) {
        pressedRef.current.delete(event.code);
        recomputeMove();
        return;
      }

      // Skill key release: fire towards mouse cursor!
      if (activeKeyRef.current === event.code) {
        activeKeyRef.current = null;
        if (controlRef.current.skill && !controlRef.current.cancelCast) {
          sendCast(controlRef.current.skill, controlRef.current.aim);
        }
        controlRef.current.aiming = false;
        controlRef.current.cancelCast = false;
        setAimingSkillIndex(null);
      }
    };

    const onBlur = () => {
      pressedRef.current.clear();
      activeKeyRef.current = null;
      controlRef.current.aiming = false;
      controlRef.current.cancelCast = false;
      setAimingSkillIndex(null);
      recomputeMove();
    };

    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    window.addEventListener("blur", onBlur);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
      window.removeEventListener("blur", onBlur);
    };
  }, [activeSkills, currentFrame, localPlayerId, performBasicAttack, sendCast]);

  const pointerToArena = useCallback((clientX: number, clientY: number) => {
    const stage = stageRef.current;
    if (!stage) return { x: 0, y: 0 };
    const rect = stage.getBoundingClientRect();
    const { scale, offsetX, offsetY } = controlRef.current.view;
    if (scale <= 0) return { x: 0, y: 0 };
    return {
      x: (clientX - rect.left - offsetX) / scale,
      y: (clientY - rect.top - offsetY) / scale,
    };
  }, []);

  // ── Mouse / Desktop Arena Pointer Handlers ────────────────────────────────
  const handleStagePointerMove = useCallback(
    (event: React.PointerEvent) => {
      if (event.pointerType !== "mouse") return;
      // In PC mode, mouse coordinates continuously update the aim target in the arena
      controlRef.current.aim = pointerToArena(event.clientX, event.clientY);
    },
    [pointerToArena],
  );

  const handleStagePointerDown = useCallback(
    (event: React.PointerEvent) => {
      if (event.pointerType !== "mouse") return;

      // Right Click: cancel any active aim, or perform basic attack if not aiming
      if (event.button === 2) {
        event.preventDefault();
        if (controlRef.current.aiming || controlRef.current.skill) {
          controlRef.current.aiming = false;
          controlRef.current.cancelCast = false;
          setAimingSkillIndex(null);
          activeKeyRef.current = null;
        } else {
          performBasicAttack();
        }
        return;
      }

      // Left Click: if a skill is selected or armed, execute cast!
      if (event.button === 0 && controlRef.current.skill) {
        controlRef.current.aim = pointerToArena(event.clientX, event.clientY);
        sendCast(controlRef.current.skill, controlRef.current.aim);
        controlRef.current.aiming = false;
        controlRef.current.cancelCast = false;
        setAimingSkillIndex(null);
        activeKeyRef.current = null;
      }
    },
    [performBasicAttack, pointerToArena, sendCast],
  );

  // ── Mobile Drag-to-Aim Handlers ───────────────────────────────────────────
  const applyMobileAimDrag = useCallback(
    (clientX: number, clientY: number) => {
      const drag = aimDragRef.current;
      if (!drag) return;

      // Check Cancel Zone hover
      if (cancelZoneRef.current) {
        const cz = cancelZoneRef.current.getBoundingClientRect();
        if (clientX >= cz.left && clientX <= cz.right && clientY >= cz.top && clientY <= cz.bottom) {
          controlRef.current.cancelCast = true;
          setIsCancelHovered(true);
          return;
        }
      }

      controlRef.current.cancelCast = false;
      setIsCancelHovered(false);

      const dx = clientX - drag.anchorX;
      const dy = clientY - drag.anchorY;
      const length = Math.hypot(dx, dy);
      const origin = localPosition();
      const skill = drag.skill;
      const reach = skill.cast_mode === "DIRECTION" ? skill.length || skill.cast_range : skill.cast_range;

      if (length < STICK_DEADZONE) {
        const self = currentFrame()?.participants.find((candidate) => candidate.id === localPlayerId);
        const facing = self?.facing ?? 0;
        controlRef.current.aim = {
          x: origin.x + Math.cos(facing) * reach,
          y: origin.y + Math.sin(facing) * reach,
        };
        return;
      }

      const dirX = dx / length;
      const dirY = dy / length;
      const fraction = Math.min(1, Math.max(0, (length - STICK_DEADZONE) / STICK_RADIUS));
      const distance = skill.cast_mode === "DIRECTION" ? reach : Math.max(1, fraction * reach);
      controlRef.current.aim = { x: origin.x + dirX * distance, y: origin.y + dirY * distance };
    },
    [currentFrame, localPlayerId, localPosition],
  );

  const handleMobileAimStart = useCallback(
    (index: number, skill: PvPSkill, clientX: number, clientY: number) => {
      setSelectedIndex(index);
      setAimingSkillIndex(index);
      controlRef.current.skill = skill;
      controlRef.current.aiming = true;
      controlRef.current.cancelCast = false;
      setIsCancelHovered(false);

      aimDragRef.current = {
        anchorX: clientX,
        anchorY: clientY,
        skill,
      };

      applyMobileAimDrag(clientX, clientY);
    },
    [applyMobileAimDrag],
  );

  const handleMobileAimMove = useCallback(
    (clientX: number, clientY: number) => {
      applyMobileAimDrag(clientX, clientY);
    },
    [applyMobileAimDrag],
  );

  const handleMobileAimEnd = useCallback(
    (clientX: number, clientY: number) => {
      const drag = aimDragRef.current;
      if (!drag) return;
      applyMobileAimDrag(clientX, clientY);
      aimDragRef.current = null;

      const wasCancel = controlRef.current.cancelCast;
      controlRef.current.aiming = false;
      controlRef.current.cancelCast = false;
      setAimingSkillIndex(null);
      setIsCancelHovered(false);

      if (!wasCancel) {
        sendCast(drag.skill, controlRef.current.aim);
      }
    },
    [applyMobileAimDrag, sendCast],
  );

  const handleMobileSkillTap = useCallback(
    (index: number, skill: PvPSkill) => {
      setSelectedIndex(index);
      controlRef.current.skill = skill;
      controlRef.current.aiming = false;
      controlRef.current.cancelCast = false;
      setAimingSkillIndex(null);
      setIsCancelHovered(false);
      aimDragRef.current = null;
      smartAutoCast(skill);
    },
    [smartAutoCast],
  );

  // Leaving the mode must tear the match subscription down.
  useEffect(() => () => stopPvP(), [stopPvP]);

  const local: PvPParticipantState | undefined = frame?.participants.find((candidate) => candidate.id === localPlayerId);
  const opponent = frame?.participants.find((candidate) => candidate.id !== localPlayerId);
  const ended = frame?.ended ?? false;
  const won = ended && frame?.winner === localPlayerId;

  if (!setup) {
    return (
      <div className="flex h-full min-h-0 w-full flex-col items-center justify-center gap-4 bg-[#070d0b] p-6 text-center">
        <div className="font-serif text-2xl font-bold text-emerald-100">Đấu Trường Thời Gian Thực</div>
        <p className="max-w-xl text-sm text-zinc-400">
          Điều khiển nhân vật tự do trong đấu trường, ngắm chiêu theo hướng hoặc theo vùng và né đòn của đối thủ.
          {uiMode === "mobile"
            ? " Cần ảo bên trái để di chuyển. Cụm chiêu MOBA bên phải: Chạm để tự ngắm, Kéo để định hướng, Kéo lên trên để hủy."
            : " Dùng WASD di chuyển. Phím Q, W, E, R, D, F để dùng chiêu (Giữ phím để ngắm, thả để xuất chiêu, Chuột phải để hủy)."}
        </p>
        {error && <div className="text-sm text-rose-400">{error}</div>}
        <button
          onClick={() => void startPracticePvP()}
          disabled={status === "loading"}
          className="rounded-xl border border-emerald-400/40 bg-emerald-500/15 px-6 py-3 font-semibold text-emerald-100 transition hover:bg-emerald-500/25 disabled:opacity-50"
        >
          {status === "loading" ? "Đang mở trận..." : "Bắt đầu luyện tập"}
        </button>
        <button onClick={onClose} className="text-xs text-zinc-500 underline hover:text-zinc-300">
          Quay lại
        </button>
      </div>
    );
  }

  return (
    <div className="relative flex h-full min-h-0 w-full flex-col overflow-hidden bg-[#070d0b] select-none">
      {/* ── Top Header Bar ────────────────────────────────────────────────── */}
      <div className="z-20 flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1 border-b border-white/10 bg-black/60 backdrop-blur-md px-3 py-2">
        <div className="flex items-baseline gap-2.5">
          <span className="font-serif text-sm font-bold text-emerald-100 sm:text-base tracking-wide">
            Đấu Trường Thời Gian Thực
          </span>
          <span className="font-mono text-[11px] text-zinc-500">tick {frame?.tick ?? 0}</span>
        </div>

        <div className="flex items-center gap-2.5 text-xs">
          {/* Layout Mode Toggle */}
          <button
            onClick={() => setUiMode((m) => (m === "mobile" ? "pc" : "mobile"))}
            title="Đổi chế độ giao diện điều khiển MOBA"
            className="flex items-center gap-1 rounded-lg border border-emerald-500/40 bg-emerald-500/15 px-2.5 py-1 text-[11px] font-semibold text-emerald-200 transition hover:bg-emerald-500/25 active:scale-95"
          >
            {uiMode === "mobile" ? "📱 MOBA Mobile" : "💻 MOBA PC"}
          </button>

          <VitalBar label="HP" value={local?.hp ?? 0} max={local?.max_hp ?? 1} color="bg-emerald-500" />
          <VitalBar label="Mana" value={local?.mana ?? 0} max={100} color="bg-sky-500" />
          {opponent && (
            <VitalBar label={opponent.name} value={opponent.hp} max={opponent.max_hp} color="bg-rose-500" />
          )}

          <button
            onClick={() => {
              stopPvP();
              onClose();
            }}
            className="rounded-lg border border-white/10 bg-white/5 px-3 py-1 font-medium text-zinc-300 transition hover:bg-white/15 hover:text-white"
          >
            Thoát
          </button>
        </div>
      </div>

      {/* ── Arena Canvas Stage ────────────────────────────────────────────── */}
      <div
        ref={stageRef}
        onContextMenu={(e) => e.preventDefault()}
        onPointerDown={handleStagePointerDown}
        onPointerMove={handleStagePointerMove}
        className="relative min-h-0 flex-1 touch-none select-none overscroll-none"
      >
        <PvPArenaCanvas
          setup={setup}
          localPlayerId={localPlayerId}
          controlRef={controlRef}
          playerAvatarUrl={player?.avatar_url}
        />

        {/* Casting Indicator Banner */}
        {local?.cast && (
          <div className="pointer-events-none absolute left-1/2 top-4 -translate-x-1/2 rounded-full border border-amber-400/40 bg-black/80 px-4 py-1 text-xs font-semibold text-amber-300 shadow-[0_0_15px_rgba(251,191,36,0.3)]">
            Đang niệm {guDictionary[local.cast.gu_id]?.name || local.cast.gu_id}...
          </div>
        )}

        {/* Range / Feedback Warning Hint */}
        {hintMessage && (
          <div className="pointer-events-none absolute left-1/2 top-14 -translate-x-1/2 z-30 flex items-center gap-1.5 rounded-full border border-amber-400/60 bg-amber-950/90 px-4 py-1.5 text-xs font-bold text-amber-200 shadow-[0_0_20px_rgba(251,191,36,0.5)]">
            <span>⚠️</span>
            <span>{hintMessage}</span>
          </div>
        )}

        {/* Mobile MOBA Controls (Virtual Joystick + Radial Fan Arc + Cancel Zone) */}
        {uiMode === "mobile" && (
          <MobaControls
            skills={activeSkills}
            basicAttackSkill={basicAttackSkill}
            guDictionary={guDictionary}
            cooldowns={local?.cd_ms ?? {}}
            selectedIndex={selectedIndex}
            aimingSkillIndex={aimingSkillIndex}
            isCancelHovered={isCancelHovered}
            onMove={(dx, dy) => {
              controlRef.current.move = { dx, dy };
            }}
            onBasicAttack={performBasicAttack}
            onSkillTap={handleMobileSkillTap}
            onAimStart={handleMobileAimStart}
            onAimMove={handleMobileAimMove}
            onAimEnd={handleMobileAimEnd}
            cancelZoneRef={cancelZoneRef}
          />
        )}

        {/* End Game Modal */}
        {ended && (
          <div className="absolute inset-0 z-40 flex flex-col items-center justify-center gap-4 bg-black/80 backdrop-blur-md">
            <div className={`font-serif text-3xl font-extrabold tracking-wide ${won ? "text-emerald-300 drop-shadow-[0_0_15px_rgba(52,211,153,0.8)]" : "text-rose-400 drop-shadow-[0_0_15px_rgba(248,113,113,0.8)]"}`}>
              {won ? "CHIẾN THẮNG" : frame?.reason === "draw" ? "BẤT PHÂN THẮNG BẠI" : "BẠI TRẬN"}
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => void startPracticePvP()}
                className="rounded-xl border border-emerald-400/50 bg-emerald-500/25 px-6 py-2.5 font-bold text-emerald-100 transition hover:bg-emerald-500/40 shadow-lg"
              >
                Đánh lại
              </button>
              <button
                onClick={() => {
                  stopPvP();
                  onClose();
                }}
                className="rounded-xl border border-white/10 bg-white/10 px-6 py-2.5 font-medium text-zinc-300 transition hover:bg-white/20"
              >
                Quay lại
              </button>
            </div>
          </div>
        )}
      </div>

      {/* ── PC MOBA HUD Dock (Bottom Center) ──────────────────────────────── */}
      {uiMode === "pc" && (
        <div className="z-20 flex shrink-0 flex-col items-center border-t border-white/10 bg-gradient-to-t from-black via-black/80 to-transparent px-4 py-2.5">
          <div className="flex items-center gap-3">
            {/* Dedicated Basic Attack Slot */}
            <MobaSkillSlot
              key="basic-attack"
              skill={basicAttackSkill}
              index={-1}
              name="Đánh Thường"
              hotkey="Space"
              cooldownRemainingMs={local?.cd_ms?.[basicAttackSkill.gu_id] ?? 0}
              isSelected={false}
              isAiming={false}
              shape="circle"
              size={60}
              showDetails={true}
              onClick={performBasicAttack}
            />

            <div className="h-10 w-px bg-white/15 mx-0.5" />

            {activeSkills.map((skill, index) => {
              const remaining = local?.cd_ms?.[skill.gu_id] ?? 0;
              const hotkey = HOTKEY_LABELS[index] ?? String(index + 1);
              const isAiming = aimingSkillIndex === index;
              const isSelected = selectedIndex === index;
              const name = skillName(skill);
              const imageUrl = guDictionary[skill.gu_id]?.image_url;

              return (
                <MobaSkillSlot
                  key={`${skill.gu_id}-${skill.rank}`}
                  skill={skill}
                  index={index}
                  name={name}
                  imageUrl={imageUrl}
                  hotkey={hotkey}
                  cooldownRemainingMs={remaining}
                  isSelected={isSelected}
                  isAiming={isAiming}
                  shape="rounded"
                  size={58}
                  showDetails={true}
                  onClick={() => {
                    if (remaining > 0) return;
                    setSelectedIndex(index);
                    controlRef.current.skill = skill;
                    if (skill.cast_mode === "SELF") {
                      sendCast(skill, controlRef.current.aim);
                      return;
                    }
                    // Arm skill for left-click cast
                    controlRef.current.aiming = true;
                    controlRef.current.cancelCast = false;
                    setAimingSkillIndex(index);
                  }}
                />
              );
            })}
          </div>

          <div className="mt-2 text-[11px] font-mono text-zinc-400">
            <span className="text-emerald-400">WASD</span>: Di chuyển •{" "}
            <span className="text-amber-300">Space / Chuột Phải</span>: Đánh thường •{" "}
            <span className="text-sky-300">Q / W / E / R / D / F</span>: Xuất chiêu (Giữ ngắm - Thả ra chiêu) •{" "}
            <span className="text-teal-300">Chuột Trái</span>: Bắn chiêu •{" "}
            <span className="text-rose-400">ESC</span>: Hủy ngắm
          </div>
        </div>
      )}
    </div>
  );
};

const VitalBar: React.FC<{ label: string; value: number; max: number; color: string }> = ({
  label,
  value,
  max,
  color,
}) => (
  <div className="flex items-center gap-1.5">
    <span className="text-[10px] font-bold uppercase tracking-wider text-zinc-400">{label}</span>
    <div className="h-2 w-16 overflow-hidden rounded-full bg-zinc-800 border border-white/5 sm:w-20">
      <div
        className={`h-full ${color} transition-all duration-150`}
        style={{ width: `${max > 0 ? Math.max(0, Math.min(100, (value / max) * 100)) : 0}%` }}
      />
    </div>
    <span className="w-8 font-mono text-[10px] text-zinc-300">{Math.round(value)}</span>
  </div>
);

/** Nearest non-local participant to the aim point; used to resolve TARGET casts. */
function nearestOpponent(
  participants: PvPParticipantState[],
  localPlayerId: string,
  aim: { x: number; y: number },
): PvPParticipantState | null {
  let best: PvPParticipantState | null = null;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const candidate of participants) {
    if (candidate.id === localPlayerId || candidate.hp <= 0) continue;
    const distance = Math.hypot(candidate.x - aim.x, candidate.y - aim.y);
    if (distance < bestDistance) {
      best = candidate;
      bestDistance = distance;
    }
  }
  return best;
}
