"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { Heart, Shield, Sparkles, Swords, Zap } from "lucide-react";
import { Num } from "@/components/Num";

type CombatActionView = Record<string, unknown> & {
  action_type?: string;
  name?: string;
  instance_id?: string;
  source_id?: string;
  target_type?: string;
  rank?: number;
  mana_cost?: number;
  damage?: number;
  defense?: number;
  healing?: number;
  reason?: string;
  target_ids?: string[];
  path?: string[];
  component_gu_ids?: string[];
  target_id?: string;
  plan?: CombatActionView[];
};

type CombatantView = Record<string, unknown> & {
  id?: string;
  name?: string;
  kind?: string;
  hp?: number;
  max_hp?: number;
  mana?: number;
  max_mana?: number;
  dead?: boolean;
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

function TeamCombatant({
  combatant,
  active,
  target,
  selectable,
  side,
  onClick,
}: {
  combatant: CombatantView;
  active: boolean;
  target: boolean;
  selectable: boolean;
  side: "ally" | "enemy";
  onClick: () => void;
}) {
  const hp = Math.max(0, value(combatant.hp));
  const maxHP = Math.max(1, value(combatant.max_hp));
  const hpPercent = Math.max(0, Math.min(100, (hp / maxHP) * 100));
  const mana = Math.max(0, value(combatant.mana));
  const maxMana = Math.max(1, value(combatant.max_mana));
  const manaPercent = Math.max(0, Math.min(100, (mana / maxMana) * 100));
  const dead = Boolean(combatant.dead) || hp <= 0;
  const borderColor = target
    ? "border-amber-300 bg-amber-950/30 ring-2 ring-amber-300/30"
    : active
      ? side === "ally"
        ? "border-emerald-400 bg-emerald-950/40 ring-1 ring-emerald-400/40"
        : "border-rose-400 bg-rose-950/40 ring-1 ring-rose-400/40"
      : "border-zinc-800 bg-zinc-900/70";

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!selectable || dead}
      aria-pressed={target}
      className={`w-full rounded-2xl border p-3 text-left shadow-lg shadow-black/20 transition ${borderColor} ${
        dead ? "opacity-45" : selectable ? "hover:-translate-y-0.5 hover:border-amber-300/70" : ""
      } disabled:cursor-default`}
    >
      <div className="mb-3 flex items-center gap-3">
        <div className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-full border bg-gradient-to-br ${
          side === "ally" ? "border-emerald-400/40 from-emerald-900 to-zinc-950 text-emerald-200" : "border-rose-400/40 from-rose-900 to-zinc-950 text-rose-200"
        }`}>
          <span className="text-lg font-black">{String(combatant.name || "?").slice(0, 1).toUpperCase()}</span>
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate font-bold text-zinc-100">{String(combatant.name || "Chiến binh")}</div>
          <div className="mt-0.5 text-[10px] uppercase tracking-[0.16em] text-zinc-500">
            {dead ? "Đã gục" : active ? "Đang hành động" : String(combatant.kind || "Chiến binh")}
          </div>
        </div>
        {target && <span className="rounded-full border border-amber-300/40 bg-amber-300/10 px-2 py-1 text-[9px] font-bold uppercase tracking-wider text-amber-200">Mục tiêu</span>}
      </div>
      <div className="mb-1.5 flex items-center justify-between text-[11px] text-zinc-400">
        <span className="flex items-center gap-1.5"><Heart className="h-3.5 w-3.5 text-rose-400" /> Sinh lực</span>
        <span className="font-mono text-zinc-200"><Num value={Math.round(hp)} /><span className="text-zinc-600"> / </span><Num value={Math.round(maxHP)} /></span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-black/70">
        <div className="h-full rounded-full bg-gradient-to-r from-rose-700 via-rose-500 to-rose-300 transition-all duration-500" style={{ width: `${hpPercent}%` }} />
      </div>
      <div className="mb-1.5 mt-3 flex items-center justify-between text-[11px] text-zinc-500">
        <span className="flex items-center gap-1.5"><Zap className="h-3.5 w-3.5 text-sky-400" /> Chân nguyên</span>
        <span className="font-mono text-zinc-300"><Num value={Math.round(mana)} /><span className="text-zinc-700"> / </span><Num value={Math.round(maxMana)} /></span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-black/70">
        <div className="h-full rounded-full bg-gradient-to-r from-sky-700 to-cyan-300 transition-all duration-500" style={{ width: `${manaPercent}%` }} />
      </div>
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
  const queuedGuActions = queuedPlan.filter((action) => action.action_type !== "BASIC_ATTACK").length;
  const lastTurn = logs.length > 0 ? value(logs[logs.length - 1].turn) : 0;
  const activeActions = actions.filter((action) => ["GU_SKILL", "KILLER_MOVE"].includes(String(action.action_type || "")));
  const targetActions = actions.filter((action) =>
    String(action.action_type || "") !== "FLEE" && String(action.target_type || "").toUpperCase() !== "SELF",
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
    if (actionType === "FLEE") {
      setQueuedPlan([{ ...action, target_id: "" }]);
      return;
    }
    if (queuedPlan.some((item) => item.action_type === "FLEE")) return;
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
            <div className="grid grid-cols-1 gap-3 md:grid-cols-[1fr_auto_1fr] md:items-center">
              <div className="space-y-2">
                <h3 className="flex items-center gap-2 px-1 text-[10px] font-bold uppercase tracking-[0.18em] text-emerald-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /> Phe ta
                </h3>
                {allies.map((combatant) => (
                  <TeamCombatant
                    key={combatant.id}
                    combatant={combatant}
                    active={combatant.id === pendingActorId}
                    target={combatant.id === targetId}
                    selectable={canChooseTargets && !actorIsAlly && !Boolean(combatant.dead)}
                    side="ally"
                    onClick={() => onTarget(String(combatant.id || ""))}
                  />
                ))}
              </div>
              <div className="hidden items-center justify-center md:flex">
                <span className="flex h-10 w-10 items-center justify-center rounded-full border border-amber-400/20 bg-zinc-950 text-[10px] font-black tracking-widest text-amber-300 shadow-lg shadow-amber-950/30">VS</span>
              </div>
              <div className="space-y-2">
                <h3 className="flex items-center gap-2 px-1 text-[10px] font-bold uppercase tracking-[0.18em] text-rose-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-rose-400" /> Phe địch
                </h3>
                {enemies.map((combatant) => (
                  <TeamCombatant
                    key={combatant.id}
                    combatant={combatant}
                    active={combatant.id === pendingActorId}
                    target={combatant.id === targetId}
                    selectable={canChooseTargets && actorIsAlly && !Boolean(combatant.dead)}
                    side="enemy"
                    onClick={() => onTarget(String(combatant.id || ""))}
                  />
                ))}
              </div>
            </div>
          </div>

          <div className="rounded-2xl border border-amber-900/40 bg-gradient-to-br from-amber-950/25 via-zinc-950/90 to-emerald-950/20 p-3 sm:p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="flex items-center gap-2 font-semibold text-amber-100">
                  <Sparkles className="h-4 w-4 text-amber-300" />
                  {ownsTurn ? "Chọn hành động" : `Đang chờ ${actor?.name || "đối thủ"}`}
                </h3>
                <p className="mt-1 text-xs text-zinc-400">
                {planLocked ? "Đã khóa kế hoạch; đang chờ đối thủ." : ownsTurn ? `Đã chọn ${queuedGuActions}/${maxGuActions} chiêu Cổ · Mục tiêu mới: ${livingTargets.find((target) => target.id === targetId)?.name || "chọn mục tiêu trên đội hình"}` : "Các lượt còn lại sẽ tự xử lý."}
                </p>
                {ownsTurn && targetId && targetIsBlocked && (
                  <p className="mt-1 text-xs text-amber-300">
                    Mục tiêu đang được hàng trước hoặc ẩn thân bảo vệ. Hãy chọn mục tiêu khác hoặc dùng chiêu xuyên hàng.
                  </p>
                )}
              </div>
              {ownsTurn && <span className="rounded-full border border-zinc-700 bg-zinc-950/80 px-2.5 py-1 text-[10px] text-zinc-400">Tự ra chiêu khi hết giờ</span>}
            </div>
            {ownsTurn && actions.length > 0 ? (
              <>
                {activeActions.length > 0 && (
                  <p className="mb-2 text-[10px] text-emerald-300/80">
                    {activeActions.length} chiêu từ Cổ / Sát Chiêu đã chọn được nạp vào trận.
                  </p>
                )}
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  {actions.map((action, index) => {
                    const actionType = String(action.action_type || "");
                    const selfTarget = String(action.target_type || "").toUpperCase() === "SELF";
                    const targetIDs = Array.isArray(action.target_ids)
                      ? action.target_ids.filter((id): id is string => typeof id === "string")
                      : null;
                    const targetBlocked = targetIDs !== null && !selfTarget && actionType !== "FLEE" && !targetIDs.includes(targetId);
                    const noTarget = livingTargets.length === 0 && !selfTarget && actionType !== "FLEE";
                    const isSkill = ["GU_SKILL", "KILLER_MOVE", "KILLER_MOVE_EFFECT"].includes(actionType);
                    const manaCost = value(action.mana_cost);
                    const title = String(action.name || (actionType === "BASIC_ATTACK" ? "Đánh thường" : actionType));
                    const paths = Array.isArray(action.path) ? action.path.filter((path): path is string => typeof path === "string") : [];
                    const stats = [
                      value(action.damage) > 0 ? `Uy lực ${Math.round(value(action.damage))}` : "",
                      value(action.defense) > 0 ? `Khiên ${Math.round(value(action.defense))}` : "",
                      value(action.healing) > 0 ? `Hồi ${Math.round(value(action.healing))}` : "",
                    ].filter(Boolean);
                    return (
                      <button
                        key={`${actionType}-${action.instance_id || action.source_id || index}`}
                        type="button"
                        disabled={actionLoading || submitting || planLocked || noTarget || targetBlocked || (actionType !== "BASIC_ATTACK" && actionType !== "FLEE" && queuedGuActions >= maxGuActions)}
                        onClick={() => queueAction(action)}
                        className={`group flex min-h-[76px] items-center justify-between gap-3 rounded-xl border px-3 py-2.5 text-left transition ${
                          actionType === "FLEE"
                            ? "border-rose-900/50 bg-rose-950/20 hover:border-rose-500/60 hover:bg-rose-950/40"
                            : "border-zinc-700/80 bg-zinc-900/80 hover:border-amber-400/50 hover:bg-amber-950/30"
                        } disabled:cursor-not-allowed disabled:opacity-45`}
                      >
                        <span className="flex min-w-0 items-center gap-2.5">
                          <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border ${
                            isSkill ? "border-amber-400/20 bg-amber-400/10 text-amber-300" : actionType === "FLEE" ? "border-rose-400/20 bg-rose-400/10 text-rose-300" : "border-zinc-700 bg-zinc-800/70 text-zinc-300"
                          }`}>
                            {isSkill ? <Sparkles className="h-4 w-4" /> : actionType === "FLEE" ? <Shield className="h-4 w-4" /> : <Swords className="h-4 w-4" />}
                          </span>
                          <span className="min-w-0">
                            <span className="flex flex-wrap items-center gap-1.5">
                              <span className="block max-w-full truncate text-sm font-semibold text-zinc-100">{title}</span>
                              {isSkill && Number(action.rank) > 0 && <span className="rounded bg-amber-950/70 px-1.5 py-0.5 text-[9px] font-bold text-amber-200">Bậc {Number(action.rank)}</span>}
                            </span>
                            <span className="mt-0.5 block truncate text-[10px] uppercase tracking-wider text-zinc-500">
                            {isSkill ? "Thêm vào kế hoạch" : actionType === "FLEE" ? "Chỉ hành động này" : "Đòn thường cuối lượt"}
                              {paths.length > 0 && ` · ${paths.join(", ")}`}
                            </span>
                            {stats.length > 0 && <span className="mt-1 block text-[10px] text-zinc-400">{stats.join(" · ")}</span>}
                            {targetBlocked && <span className="mt-1 block text-[10px] text-amber-300/80">Không thể nhắm mục tiêu này</span>}
                          </span>
                        </span>
                        {actionType !== "FLEE" && (
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
