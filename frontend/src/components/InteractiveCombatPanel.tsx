"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { LogOut, Shield, Sparkles, Swords, Zap } from "lucide-react";
import { Num } from "@/components/Num";
import { MonsterSprite } from "@/components/MonsterSprite";

type GuHouseView = {
  id?: string;
  name?: string;
  rank?: number;
  hp?: number;
  max_hp?: number;
  active?: boolean;
  is_destroyed?: boolean;
  damage_reduction?: number;
};

type CombatActionView = Record<string, unknown> & {
  action_type?: string;
  name?: string;
  instance_id?: string;
  source_id?: string;
  target_type?: string;
  rank?: number;
  mana_cost?: number;
  hunger_cost?: number;
  damage?: number;
  defense?: number;
  healing?: number;
  reason?: string;
  target_ids?: string[];
  path?: string[];
  component_gu_ids?: string[];
  target_id?: string;
  plan?: CombatActionView[];
  is_from_gu_house?: boolean;
};

type CombatantView = Record<string, unknown> & {
  id?: string;
  name?: string;
  kind?: string;
  hp?: number;
  max_hp?: number;
  mana?: number;
  max_mana?: number;
  grid_x?: number;
  grid_y?: number;
  rank?: number;
  shield?: number;
  avatar_url?: string;
  dead?: boolean;
  gu_house?: GuHouseView;
  actions?: CombatActionView[];
};

type CombatLogView = Record<string, unknown> & {
  turn?: number;
  actor_name?: string;
  action?: string;
  target_name?: string;
  detail?: string;
  damage?: number;
  hp_damage?: number;
  shield_absorbed?: number;
};

interface InteractiveCombatPanelProps {
  session: Record<string, unknown>;
  playerId: string;
  targetId: string;
  actionLoading: boolean;
  onTarget: (targetId: string) => void;
  onAction: (action: CombatActionView) => Promise<unknown>;
}

const asCombatants = (value: unknown): CombatantView[] =>
  Array.isArray(value) ? value as CombatantView[] : [];

const asLogs = (value: unknown): CombatLogView[] =>
  Array.isArray(value) ? value as CombatLogView[] : [];

const value = (input: unknown): number => {
  const parsed = Number(input);
  return Number.isFinite(parsed) ? parsed : 0;
};

function GridCombatant({
  combatant,
  active,
  target,
  selectable,
  side,
  gridX,
  gridY,
  onClick,
}: {
  combatant: CombatantView;
  active: boolean;
  target: boolean;
  selectable: boolean;
  side: "ally" | "enemy";
  gridX: number;
  gridY: number;
  onClick: () => void;
}) {
  const hp = Math.max(0, value(combatant.hp));
  const maxHP = Math.max(1, value(combatant.max_hp));
  const hpPercent = Math.max(0, Math.min(100, (hp / maxHP) * 100));
  const shield = Math.max(0, value(combatant.shield));
  const dead = Boolean(combatant.dead) || hp <= 0;
  const guHouse = combatant.gu_house;
  const inGuHouse = Boolean(guHouse && guHouse.active && !guHouse.is_destroyed);
  const kind = String(combatant.kind || "");
  const icon = inGuHouse ? "🏛" : /pet|beast/i.test(kind) ? "🐾" : /boss|enemy|monster/i.test(kind) ? "👹" : String(combatant.name || "?").slice(0, 1).toUpperCase();
  const borderColor = target ? "border-amber-300 bg-amber-950/50 ring-2 ring-amber-300/60" :
    active ? side === "ally" ? "border-emerald-400 bg-emerald-950/40 ring-1 ring-emerald-400/50" : "border-rose-400 bg-rose-950/40 ring-1 ring-rose-400/50" :
    inGuHouse ? "border-amber-500/70 bg-amber-950/40" : side === "ally" ? "border-emerald-900/60 bg-zinc-900/80" : "border-rose-900/60 bg-zinc-900/80";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!selectable || dead}
      aria-pressed={target}
      aria-label={`${combatant.name || "Chiến binh"}, ô ${gridX}, ${gridY}, HP ${Math.round(hp)} trên ${Math.round(maxHP)}${dead ? ", đã gục" : ""}`}
      title={`${combatant.name || "Chiến binh"} · ô ${gridX},${gridY} · HP ${Math.round(hp)}/${Math.round(maxHP)}${inGuHouse ? ` · ${guHouse?.name || "Tiên Cổ Phòng"}` : ""}`}
      className={`relative flex h-full w-full max-w-full min-w-0 flex-col items-center gap-1 overflow-hidden rounded-xl border p-1.5 text-left transition sm:p-1.5 ${borderColor} ${dead ? "opacity-45" : selectable ? "cursor-pointer hover:border-amber-300/80" : ""} disabled:cursor-default`}
    >
      <span className="flex w-full min-w-0 items-center justify-between gap-1">
        <span className={`block min-w-0 flex-1 truncate text-[8px] font-bold sm:text-[10px] ${dead ? "text-zinc-500 line-through" : side === "ally" ? "text-emerald-300" : "text-rose-300"}`}>{String(combatant.name || "?")}</span>
        {dead ? <span aria-hidden="true" className="shrink-0 text-[9px] text-rose-500">†</span> : combatant.rank ? <span className="shrink-0 rounded bg-zinc-800/90 px-1 text-[7px] font-mono text-amber-300 sm:text-[8px]">{combatant.rank >= 6 ? `C${combatant.rank}` : `H${combatant.rank}`}</span> : active ? <span aria-hidden="true" className="shrink-0 text-[8px] text-amber-300">◆</span> : null}
      </span>
      <span className="flex min-h-0 w-full flex-1 items-center justify-center overflow-hidden">
        {side === "enemy" ? (
          <MonsterSprite templateId={String(combatant.id || "")} alt={String(combatant.name || "Kẻ địch")} className="h-[90%] w-[90%] object-contain drop-shadow-[0_0_10px_rgba(251,113,133,0.25)]" />
        ) : combatant.avatar_url ? (
          <img src={combatant.avatar_url} alt="" className="h-[62%] aspect-square rounded-full border-2 border-emerald-500/60 object-cover" />
        ) : (
          <span aria-hidden="true" className="flex h-[62%] aspect-square items-center justify-center overflow-hidden rounded-full border-2 border-emerald-500/60 bg-emerald-950 text-[10px] font-black text-emerald-200 sm:text-sm">{icon}</span>
        )}
      </span>
      <span className="w-full min-w-0">
        <span className="mb-0.5 flex items-center justify-between gap-1 text-[7px] font-mono leading-tight sm:text-[8px]">
          <span className="shrink-0 text-zinc-400">HP</span>
          <span className="min-w-0 truncate text-right font-bold text-zinc-200"><Num value={Math.round(hp)} /> / <Num value={Math.round(maxHP)} /></span>
        </span>
        <span className="block h-1.5 w-full overflow-hidden rounded-full border border-zinc-800 bg-zinc-900">
          <span className={`block h-full rounded-full transition-[width] ${dead ? "bg-zinc-700" : side === "ally" ? "bg-gradient-to-r from-emerald-600 to-emerald-400" : "bg-gradient-to-r from-rose-600 to-rose-400"}`} style={{ width: `${hpPercent}%` }} />
        </span>
        {shield > 0 && <span className="mt-0.5 block truncate text-center text-[7px] font-mono text-cyan-300">🛡 {Math.round(shield)}</span>}
      </span>
    </button>
  );
}

export function InteractiveCombatPanel({
  session,
  playerId,
  targetId,
  actionLoading,
  onTarget,
  onAction,
}: InteractiveCombatPanelProps) {
  const [submitting, setSubmitting] = useState(false);
  const [queuedPlan, setQueuedPlan] = useState<CombatActionView[]>([]);
  const [currentTime, setCurrentTime] = useState(() => Date.now());
  const logContainerRef = useRef<HTMLDivElement>(null);
  const allies = asCombatants(session.allies);
  const enemies = asCombatants(session.enemies);
  const logs = asLogs(session.logs);
  const planning = Boolean(session.planning);
  const planLocked = Boolean(session.my_plan_locked);
  const pendingActorId = planning ? playerId : String(session.pending_actor || "");
  const actor = useMemo(
    () => [...allies, ...enemies].find((combatant) => combatant.id === pendingActorId),
    [allies, enemies, pendingActorId],
  );
  const actorIsAlly = allies.some((combatant) => combatant.id === pendingActorId);
  const targets = actorIsAlly ? enemies : allies;
  const livingTargets = targets.filter((combatant) => !combatant.dead && value(combatant.hp) > 0);
  const actions = Array.from(new Map((actor?.actions || []).map((action, index) => {
    const actionType = String(action.action_type || "");
    const identity = String(action.instance_id || action.source_id || (actionType === "BASIC_ATTACK" ? "" : index));
    return [`${actionType}|${identity}`, action] as const;
  })).values());
  const ownsTurn = actor?.id === playerId;
  const canChooseTargets = ownsTurn && !planLocked && !actionLoading && !submitting;
  const maxGuActions = Math.max(1, Number(session.max_gu_actions || 3));
  const queuedGuActions = queuedPlan.filter((action) => action.action_type !== "BASIC_ATTACK" && action.action_type !== "EXIT_GU_HOUSE" && action.action_type !== "ENTER_GU_HOUSE").length;
  const lastTurn = logs.length > 0 ? value(logs[logs.length - 1].turn) : 0;
  const deadlineAt = typeof session.deadline_at === "string" ? Date.parse(session.deadline_at) : NaN;
  const remainingSeconds = Number.isFinite(deadlineAt) ? Math.max(0, Math.ceil((deadlineAt - currentTime) / 1000)) : null;

  useEffect(() => {
    if (!Number.isFinite(deadlineAt)) return;
    setCurrentTime(Date.now());
    const timer = window.setInterval(() => setCurrentTime(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [deadlineAt]);

  const renderTeamGrid = (members: CombatantView[], side: "ally" | "enemy") => (
    <div className={`grid grid-cols-5 gap-1 rounded-xl border p-1.5 sm:gap-1.5 sm:p-2 ${side === "ally" ? "border-emerald-900/40 bg-emerald-950/10" : "border-rose-900/40 bg-rose-950/10"}`}>
      {Array.from({ length: 25 }, (_, index) => {
        const gridX = Math.floor(index / 5) + 1;
        const gridY = (index % 5) + 1;
        const occupants = members.filter((combatant, memberIndex) => {
          const x = Number(combatant.grid_x) || (memberIndex % 5) + 1;
          const y = Number(combatant.grid_y) || Math.floor(memberIndex / 5) + 1;
          return x === gridX && y === gridY;
        });
        return (
          <div key={`${gridX}-${gridY}`} className={`aspect-square min-w-0 overflow-hidden rounded-lg ${occupants.length ? "" : "border border-dashed border-zinc-800/70 bg-black/15"}`}>
            {occupants.length ? occupants.map((combatant) => (
              <GridCombatant
                key={combatant.id}
                combatant={combatant}
                active={combatant.id === pendingActorId}
                target={combatant.id === targetId}
                selectable={canChooseTargets && (side === "ally" ? !actorIsAlly : actorIsAlly) && !Boolean(combatant.dead)}
                side={side}
                gridX={gridX}
                gridY={gridY}
                onClick={() => onTarget(String(combatant.id || ""))}
              />
            )) : <div className="flex h-full min-h-0 items-center justify-center text-[9px] font-mono text-zinc-800">{gridX},{gridY}</div>}
          </div>
        );
      })}
    </div>
  );

  const inGuHouse = Boolean(actor?.gu_house && actor.gu_house.active && !actor.gu_house.is_destroyed);
  const strikeActions = actions.filter((action) => action.is_from_gu_house || action.action_type === "EXIT_GU_HOUSE");
  const normalActions = actions.filter((action) => !action.is_from_gu_house && action.action_type !== "EXIT_GU_HOUSE");
  const displayActions = inGuHouse ? strikeActions : normalActions;

  const activeActions = displayActions.filter((action) => ["GU_SKILL", "KILLER_MOVE"].includes(String(action.action_type || "")));
  const targetActions = displayActions.filter((action) =>
    String(action.action_type || "") !== "FLEE" &&
    String(action.action_type || "") !== "EXIT_GU_HOUSE" &&
    String(action.action_type || "") !== "ENTER_GU_HOUSE" &&
    String(action.target_type || "").toUpperCase() !== "SELF",
  );
  const targetIsBlocked = targetActions.some((action) => Array.isArray(action.target_ids)) &&
    targetActions.every((action) => Array.isArray(action.target_ids) && !action.target_ids.includes(targetId));

  useEffect(() => {
    const container = logContainerRef.current;
    if (container) container.scrollTop = container.scrollHeight;
  }, [logs.length]);

  useEffect(() => { setQueuedPlan([]); }, [session.revision, pendingActorId]);

  const queueAction = (action: CombatActionView) => {
    if (planLocked) return;
    const actionType = String(action.action_type || "");
    if (actionType === "FLEE" || actionType === "EXIT_GU_HOUSE" || actionType === "ENTER_GU_HOUSE") {
      setQueuedPlan([{ ...action, target_id: "" }]);
      return;
    }
    if (queuedPlan.some((item) => ["FLEE", "EXIT_GU_HOUSE", "ENTER_GU_HOUSE"].includes(String(item.action_type || "")))) return;
    if (actionType === "BASIC_ATTACK") {
      setQueuedPlan((current) => [...current.filter((item) => item.action_type !== "BASIC_ATTACK"), { ...action, target_id: targetId }]);
      return;
    }
    if (queuedGuActions >= maxGuActions) return;
    const guIDs = [
      ...(Array.isArray(action.component_gu_ids) ? action.component_gu_ids : []),
      ...(actionType === "GU_SKILL" ? [String(action.instance_id || action.source_id || "")] : []),
    ].filter(Boolean);
    const usedGuIDs = new Set(queuedPlan.flatMap((item) => [
      ...(Array.isArray(item.component_gu_ids) ? item.component_gu_ids : []),
      ...(item.action_type === "GU_SKILL" ? [String(item.instance_id || item.source_id || "")] : []),
    ].filter(Boolean)));
    if (guIDs.some((id) => usedGuIDs.has(id))) return;
    setQueuedPlan((current) => [...current.filter((item) => item.action_type !== "BASIC_ATTACK"), { ...action, target_id: targetId }]);
  };

  const submitAction = async () => {
    if (actionLoading || submitting) return;
    if (!queuedPlan.length) return;
    setSubmitting(true);
    try {
      const submitted = await onAction({ action_type: "PLAN", plan: queuedPlan });
      if (submitted) setQueuedPlan([]);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="relative overflow-hidden rounded-3xl border border-amber-900/40 bg-zinc-950 shadow-2xl shadow-black/40">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_18%_0%,rgba(16,185,129,0.12),transparent_38%),radial-gradient(ellipse_at_82%_0%,rgba(244,63,94,0.11),transparent_38%)]" />
      <header className="relative flex flex-wrap items-center justify-between gap-3 border-b border-zinc-800/80 bg-gradient-to-r from-emerald-950/70 via-zinc-950 to-rose-950/60 px-5 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-amber-400/20 bg-amber-400/10">
            <Swords className="h-5 w-5 text-amber-300" />
          </div>
          <div>
            <h2 className="font-bold tracking-wide text-zinc-100">Chiến Đấu Mô Phỏng</h2>
            <p className="text-xs text-zinc-400">Chế độ điều khiển · Lượt {lastTurn || 1} · Chọn mục tiêu rồi ra chiêu</p>
          </div>
        </div>
        <div className={`flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-semibold ${
          ownsTurn ? "border-emerald-600/50 bg-emerald-950/60 text-emerald-200" : "border-amber-700/50 bg-amber-950/40 text-amber-200"
        }`}>
          <span className={`h-1.5 w-1.5 rounded-full ${ownsTurn ? "bg-emerald-400" : "animate-pulse bg-amber-400"}`} />
          {actor ? ownsTurn ? "Lượt của bạn" : `Đang hành động: ${actor.name || "đối thủ"}` : "Đang xử lý lượt"}
        </div>
      </header>

      <div className="relative grid grid-cols-1 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
        <div className="space-y-4 p-4 sm:p-5">
          <div className="rounded-2xl border border-zinc-800/80 bg-gradient-to-br from-zinc-900/80 via-zinc-950 to-zinc-900/60 p-3 sm:p-4">
            <div className="mb-3 flex items-center justify-center gap-3">
              <span className="h-px flex-1 bg-gradient-to-r from-transparent to-emerald-700/60" />
              <span className="rounded-full border border-zinc-700 bg-black/50 px-3 py-1 text-[10px] font-black tracking-[0.2em] text-amber-300">HIỆN TRƯỜNG</span>
              <span className="h-px flex-1 bg-gradient-to-l from-transparent to-rose-700/60" />
            </div>
            <div className="grid grid-cols-1 gap-3 xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] xl:items-center">
              <div className="min-w-0 space-y-2">
                <h3 className="flex items-center gap-2 px-1 text-[10px] font-bold uppercase tracking-[0.18em] text-emerald-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /> Phe ta
                  <span className="ml-auto text-zinc-600">5×5</span>
                </h3>
                {renderTeamGrid(allies, "ally")}
              </div>
              <div className="hidden items-center justify-center xl:flex">
                <span className="flex h-10 w-10 items-center justify-center rounded-full border border-amber-400/20 bg-zinc-950 text-[10px] font-black tracking-widest text-amber-300 shadow-lg shadow-amber-950/30">VS</span>
              </div>
              <div className="min-w-0 space-y-2">
                <h3 className="flex items-center gap-2 px-1 text-[10px] font-bold uppercase tracking-[0.18em] text-rose-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-rose-400" /> Phe địch
                  <span className="ml-auto text-zinc-600">5×5</span>
                </h3>
                {renderTeamGrid(enemies, "enemy")}
              </div>
            </div>
          </div>

          <div className="rounded-2xl border border-amber-900/40 bg-gradient-to-br from-amber-950/25 via-zinc-950/90 to-emerald-950/20 p-3 sm:p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="flex items-center gap-2 font-semibold text-amber-100">
                  <Sparkles className="h-4 w-4 text-amber-300" />
                  {ownsTurn
                    ? inGuHouse
                      ? `🏛️ Bảng Chiêu Thức Trận Kích: ${actor?.gu_house?.name || "Tiên Cổ Ốc"}`
                      : "Chọn hành động"
                    : `Đang chờ ${actor?.name || "đối thủ"}`}
                </h3>
                {remainingSeconds !== null && (
                  <div className="mt-2 max-w-xs rounded-lg border border-zinc-800 bg-black/30 px-2.5 py-2">
                    <div className="mb-1 flex items-center justify-between gap-3 text-xs">
                      <span className="font-semibold text-zinc-300">{ownsTurn ? "Còn thời gian chọn chiêu" : "Thời gian chờ lượt"}</span>
                      <span className={`font-mono font-bold tabular-nums ${remainingSeconds <= 8 ? "text-rose-300" : remainingSeconds <= 18 ? "text-amber-300" : "text-emerald-300"}`}>
                        {remainingSeconds === 0 ? "Hết giờ" : `0:${String(remainingSeconds).padStart(2, "0")}`}
                      </span>
                    </div>
                    <div
                      role="progressbar"
                      aria-label="Thời gian còn lại của lượt"
                      aria-valuemin={0}
                      aria-valuemax={45}
                      aria-valuenow={remainingSeconds}
                      className="h-1.5 overflow-hidden rounded-full bg-zinc-800"
                    >
                      <div
                        className={`h-full rounded-full transition-[width] duration-1000 ${remainingSeconds <= 8 ? "bg-rose-500" : remainingSeconds <= 18 ? "bg-amber-400" : "bg-emerald-500"}`}
                        style={{ width: `${Math.min(100, (remainingSeconds / 45) * 100)}%` }}
                      />
                    </div>
                    <p className="mt-1 text-[10px] text-zinc-600">Hết giờ, server sẽ tự xử lý lượt.</p>
                  </div>
                )}
                <p className="mt-1 text-xs text-zinc-400">
                  {planLocked
                    ? "Đã khóa kế hoạch; đang chờ đối thủ."
                    : ownsTurn
                      ? inGuHouse
                        ? `Đang Dung Nhập Cổ Phòng (Bậc ${actor?.gu_house?.rank || 6}) · Cổ kỹ cá nhân tạm khóa · Đã chọn ${queuedGuActions}/${maxGuActions} Trận Kích`
                        : `Đã chọn ${queuedGuActions}/${maxGuActions} chiêu Cổ · Mục tiêu mới: ${livingTargets.find((target) => target.id === targetId)?.name || "chọn mục tiêu trên đội hình"}`
                      : "Các lượt còn lại sẽ tự xử lý."}
                </p>
                {ownsTurn && targetId && targetIsBlocked && (
                  <p className="mt-1 text-xs text-amber-300">
                    Mục tiêu đang được hàng trước hoặc ẩn thân bảo vệ. Hãy chọn mục tiêu khác hoặc dùng chiêu xuyên hàng.
                  </p>
                )}
              </div>
              {ownsTurn && <span className="rounded-full border border-zinc-700 bg-zinc-950/80 px-2.5 py-1 text-[10px] text-zinc-400">Tự ra chiêu khi hết giờ</span>}
            </div>

            {inGuHouse && ownsTurn && (
              <div className="mb-3 rounded-xl border border-amber-500/40 bg-gradient-to-r from-amber-950/60 via-zinc-950 to-amber-950/40 p-3">
                <div className="flex items-center justify-between text-xs font-bold text-amber-300">
                  <span className="flex items-center gap-1.5">
                    <Shield className="h-4 w-4 text-amber-400" /> DUNG NHẬP TIÊN CỔ PHÒNG HOẠT HÓA
                  </span>
                  <span className="rounded bg-amber-500/20 px-2 py-0.5 text-[10px] text-amber-200 border border-amber-400/30">
                    Pháo Đài Phòng Ngự
                  </span>
                </div>
                <p className="mt-1 text-[11px] text-amber-200/80">
                  Nhục thân đã ẩn vào trận nhãn — Cổ kỹ cá nhân tạm thời vô hiệu hóa; chỉ có thể thôi động Trận Kích của Tiên Cổ Phòng.
                </p>
              </div>
            )}

            {ownsTurn && displayActions.length > 0 ? (
              <>
                {activeActions.length > 0 && (
                  <p className="mb-2 text-[10px] text-emerald-300/80">
                    {inGuHouse
                      ? `${activeActions.length} Trận Kích từ Tiên Cổ Phòng khả dụng lượt này.`
                      : `${activeActions.length} chiêu từ Cổ / Sát Chiêu đã chọn được nạp vào trận.`}
                  </p>
                )}
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  {displayActions.map((action, index) => {
                    const actionType = String(action.action_type || "");
                    const selfTarget = String(action.target_type || "").toUpperCase() === "SELF";
                    const isFlee = actionType === "FLEE";
                    const isExit = actionType === "EXIT_GU_HOUSE";
                    const isEnter = actionType === "ENTER_GU_HOUSE";
                    const isHouseStrike = Boolean(action.is_from_gu_house) && !isExit;
                    const targetIDs = Array.isArray(action.target_ids)
                      ? action.target_ids.filter((id): id is string => typeof id === "string")
                      : null;
                    const targetBlocked = targetIDs !== null && !selfTarget && !isFlee && !isExit && !isEnter && !targetIDs.includes(targetId);
                    const noTarget = livingTargets.length === 0 && !selfTarget && !isFlee && !isExit && !isEnter;
                    const isSkill = ["GU_SKILL", "KILLER_MOVE", "KILLER_MOVE_EFFECT"].includes(actionType);
                    const manaCost = value(action.mana_cost);
                    const hungerCost = value(action.hunger_cost);
                    const title = String(action.name || (actionType === "BASIC_ATTACK" ? "Đánh thường" : actionType));
                    const paths = Array.isArray(action.path) ? action.path.filter((path): path is string => typeof path === "string") : [];
                    const stats = [
                      value(action.damage) > 0 ? `Uy lực ${Math.round(value(action.damage))}` : "",
                      value(action.defense) > 0 ? `Khiên ${Math.round(value(action.defense))}` : "",
                      value(action.healing) > 0 ? `Hồi ${Math.round(value(action.healing))}` : "",
                    ].filter(Boolean);

                    const buttonBorderClass = isExit
                      ? "border-rose-900/60 bg-gradient-to-br from-rose-950/30 to-zinc-900 hover:border-rose-400 hover:bg-rose-950/50"
                      : isEnter
                        ? "border-amber-500/60 bg-gradient-to-br from-amber-950/30 to-zinc-900 hover:border-amber-400 hover:bg-amber-950/50"
                        : isHouseStrike
                          ? "border-amber-500/50 bg-gradient-to-br from-amber-950/30 via-zinc-900 to-zinc-950 hover:border-amber-300 hover:shadow-lg hover:shadow-amber-500/10"
                          : isFlee
                            ? "border-rose-900/50 bg-rose-950/20 hover:border-rose-500/60 hover:bg-rose-950/40"
                            : "border-zinc-700/80 bg-zinc-900/80 hover:border-amber-400/50 hover:bg-amber-950/30";

                    return (
                      <button
                        key={`${actionType}-${action.instance_id || action.source_id || index}`}
                        type="button"
                        disabled={actionLoading || submitting || planLocked || noTarget || targetBlocked || (!isFlee && !isExit && !isEnter && actionType !== "BASIC_ATTACK" && queuedGuActions >= maxGuActions)}
                        onClick={() => queueAction(action)}
                        className={`group flex min-h-[76px] items-center justify-between gap-3 rounded-xl border px-3 py-2.5 text-left transition ${buttonBorderClass} disabled:cursor-not-allowed disabled:opacity-45`}
                      >
                        <span className="flex min-w-0 items-center gap-2.5">
                          <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border ${
                            isExit
                              ? "border-rose-500/40 bg-rose-500/20 text-rose-300"
                              : isEnter || isHouseStrike
                                ? "border-amber-400/40 bg-amber-400/20 text-amber-300"
                                : isSkill
                                  ? "border-amber-400/20 bg-amber-400/10 text-amber-300"
                                  : isFlee
                                    ? "border-rose-400/20 bg-rose-400/10 text-rose-300"
                                    : "border-zinc-700 bg-zinc-800/70 text-zinc-300"
                          }`}>
                            {isExit ? <LogOut className="h-4 w-4" /> : isEnter ? <Shield className="h-4 w-4" /> : isHouseStrike ? <Sparkles className="h-4 w-4" /> : isSkill ? <Sparkles className="h-4 w-4" /> : isFlee ? <Shield className="h-4 w-4" /> : <Swords className="h-4 w-4" />}
                          </span>
                          <span className="min-w-0">
                            <span className="flex flex-wrap items-center gap-1.5">
                              <span className="block max-w-full truncate text-sm font-semibold text-zinc-100">{title}</span>
                              {isHouseStrike && <span className="rounded bg-amber-500/20 px-1.5 py-0.5 text-[9px] font-bold text-amber-300 border border-amber-500/30">Trận Kích</span>}
                              {!isHouseStrike && isSkill && Number(action.rank) > 0 && <span className="rounded bg-amber-950/70 px-1.5 py-0.5 text-[9px] font-bold text-amber-200">Bậc {Number(action.rank)}</span>}
                            </span>
                            <span className="mt-0.5 block truncate text-[10px] uppercase tracking-wider text-zinc-500">
                              {isExit ? "Rút về nhục thân (khẩn cấp)" : isEnter ? "Tiến vào Tiên Cổ Phòng" : isHouseStrike ? (paths.length > 0 ? paths.join(", ") : "Uy lực Cổ Phòng") : isSkill ? "Thêm vào kế hoạch" : isFlee ? "Chỉ hành động này" : "Đòn thường cuối lượt"}
                              {!isHouseStrike && paths.length > 0 && ` · ${paths.join(", ")}`}
                            </span>
                            {stats.length > 0 && <span className="mt-1 block text-[10px] text-zinc-400">{stats.join(" · ")}</span>}
                            {hungerCost > 0 && <span className="mt-0.5 block text-[10px] text-amber-400/80 font-mono">Độ no Cổ hạch tâm: -{hungerCost}</span>}
                            {targetBlocked && <span className="mt-1 block text-[10px] text-amber-300/80">Không thể nhắm mục tiêu này</span>}
                          </span>
                        </span>
                        {!isFlee && !isExit && !isEnter && (
                          <span className={`flex shrink-0 items-center gap-1 rounded-lg px-2 py-1 text-[10px] font-semibold ${
                            manaCost > 0 ? "bg-sky-950/60 text-sky-200" : "bg-zinc-800/70 text-zinc-400"
                          }`}>
                            <Zap className="h-3 w-3" />{manaCost > 0 ? <Num value={manaCost} /> : "Miễn phí"}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
                {queuedPlan.length > 0 && !planLocked && (
                  <div className="mt-3 rounded-xl border border-emerald-800/50 bg-emerald-950/20 p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-semibold text-emerald-200">Kế hoạch lượt này</span>
                      <button type="button" onClick={() => setQueuedPlan([])} className="text-[10px] text-zinc-400 hover:text-white">Xóa hết</button>
                    </div>
                    <ol className="space-y-1 text-xs text-zinc-300">
                      {queuedPlan.map((item, index) => <li key={`${item.instance_id || item.source_id || item.action_type}-${index}`} className="flex justify-between gap-3">
                        <span>{index + 1}. {String(item.name || item.action_type)} → {String(item.target_type || "").toUpperCase() === "SELF" ? "Bản thân" : livingTargets.find((target) => target.id === item.target_id)?.name || "Mục tiêu đã khóa"}</span>
                        <button type="button" onClick={() => setQueuedPlan((current) => current.filter((_, itemIndex) => itemIndex !== index))} className="text-zinc-500 hover:text-rose-300">Bỏ</button>
                      </li>)}
                    </ol>
                    <button type="button" disabled={actionLoading || submitting} onClick={() => void submitAction()} className="mt-3 w-full rounded-lg bg-emerald-700 px-3 py-2 text-sm font-bold text-white hover:bg-emerald-600 disabled:opacity-50">Khóa kế hoạch ({queuedPlan.length} hành động)</button>
                  </div>
                )}
                {actions.some((action) => Boolean(action.reason)) && (
                  <p className="mt-2 text-[10px] text-zinc-500">{String(actions.find((action) => action.reason)?.reason)}</p>
                )}
              </>
            ) : ownsTurn ? (
              <p className="rounded-xl border border-amber-800/40 bg-amber-950/20 p-3 text-sm text-amber-200">Không có chiêu khả dụng; hệ thống sẽ tự đánh thường.</p>
            ) : (
              <div className="flex items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900/70 p-3 text-sm text-zinc-300">
                <Shield className="h-4 w-4 text-amber-300" /> Đang chờ đối phương hành động hoặc hết thời gian.
              </div>
            )}
          </div>
        </div>

        <aside className="border-t border-zinc-800/80 bg-black/25 xl:border-l xl:border-t-0">
          <div className="flex items-center justify-between border-b border-zinc-800/80 px-4 py-3.5">
            <div>
              <h3 className="font-bold text-zinc-100">Diễn biến trận đấu</h3>
              <p className="text-xs text-zinc-500">Nhật ký chiến trường theo lượt</p>
            </div>
            <span className="rounded-full border border-zinc-800 bg-zinc-900 px-2.5 py-1 text-[10px] text-zinc-400">{logs.length} sự kiện</span>
          </div>
          <div ref={logContainerRef} className="h-[360px] space-y-2 overflow-y-auto p-3 xl:h-[660px]">
            {logs.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-sm text-zinc-500">
                <Swords className="h-6 w-6 text-zinc-700" />
                Trận đấu bắt đầu; diễn biến sẽ hiện tại đây.
              </div>
            ) : logs.map((log, index) => {
              const action = String(log.action || "");
              const isSystem = ["BATTLE_START", "TURN_START", "AWAIT_COMMAND"].includes(action);
              const isSkill = action === "GU_SKILL" || action === "KILLER_MOVE";
              const damage = value(log.hp_damage ?? log.damage);
              const shield = value(log.shield_absorbed);
              const actionLabel: Record<string, string> = {
                BASIC_ATTACK: "Đánh thường",
                GU_SKILL: "Cổ kỹ",
                KILLER_MOVE: "Sát chiêu",
                AWAIT_COMMAND: "Chờ ra lệnh",
                TURN_START: "Mở lượt",
                BATTLE_START: "Khai chiến",
                GU_HOUSE_EXIT: "Rời Cổ Phòng",
                GU_HOUSE_ENTER: "Dung Nhập",
                GU_HOUSE_RESTRICTED: "Hạn chế Cổ Phòng",
              };
              return (
                <article key={`${index}-${action}`} className={`rounded-xl border px-3 py-2.5 ${
                  isSystem ? "border-zinc-800/80 bg-zinc-900/40" : isSkill ? "border-amber-900/40 bg-amber-950/10" : "border-zinc-800 bg-zinc-900/70"
                }`}>
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
                    <span className="rounded-md border border-zinc-800 bg-black/40 px-1.5 py-0.5 font-mono text-[10px] text-zinc-500">L{value(log.turn) || 1}</span>
                    <strong className={isSystem ? "text-zinc-300" : isSkill ? "text-amber-200" : "text-emerald-200"}>{String(log.actor_name || (isSystem ? "Chiến trường" : "Chiến binh"))}</strong>
                    {!isSystem && <span className="rounded bg-black/30 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-zinc-400">{actionLabel[action] || action.replaceAll("_", " ")}</span>}
                    {log.target_name && <span className="text-zinc-500">→ {String(log.target_name)}</span>}
                  </div>
                  {damage > 0 && <p className="mt-1.5 text-xs font-semibold text-rose-300">Gây <Num value={Math.round(damage)} /> sát thương{shield > 0 && <span className="font-normal text-sky-300"> · khiên đỡ <Num value={Math.round(shield)} /></span>}</p>}
                  {log.detail && <p className="mt-1.5 whitespace-pre-wrap text-xs leading-relaxed text-zinc-400">{String(log.detail)}</p>}
                </article>
              );
            })}
          </div>
        </aside>
      </div>
    </section>
  );
}
