"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import type { RefineActiveStep, RefineSession } from "@/store/slices/refineSessionSlice";
import { GameStepProgress } from "@/components/ui/GameProgressBar";

// Phiên Luyện Cổ chạy nền (WF-REF-07): timeline + 4 minigame + HUD nổi.
// Server chấm điểm; client chỉ gửi input rời rạc + nonce (idempotent).

const STEP_TYPE_LABEL: Record<string, string> = {
  CHOICE: "Dẫn Hỏa",
  TAP: "Canh Nhịp",
  SEQUENCE: "Dãy Ấn",
  ORDER: "Ghép Liệu",
};

// Estimate the local↔server clock offset from the last server_now snapshot.
function useServerOffset(session: RefineSession | null): number {
  const [offsetMs, setOffsetMs] = useState(0);
  const serverNow = session?.server_now;
  useEffect(() => {
    if (!serverNow) return;
    const parsed = Date.parse(serverNow);
    if (!Number.isNaN(parsed)) {
      setOffsetMs(parsed - Date.now());
    }
  }, [serverNow]);
  return offsetMs;
}

// Đồng hồ đếm giây dùng chung (tick 1s/tick) — dùng cho countdown tổng & bước sau.
function useNowTick(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [active]);
  return now;
}

function formatDuration(msTotal: number): string {
  const s = Math.max(0, Math.ceil(msTotal / 1000));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}:${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`;
  return `${m}:${String(sec).padStart(2, "0")}`;
}

function formatServerClock(serverMs: number, offsetMs: number): string {
  const d = new Date(serverMs + offsetMs); // hiển thị theo giờ local
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function CountdownBar({ windowEnd, offsetMs, durationMs = 20000 }: { windowEnd: string; offsetMs: number; durationMs?: number }) {
  const [left, setLeft] = useState(0);
  useEffect(() => {
    const end = Date.parse(windowEnd) - offsetMs;
    if (Number.isNaN(end)) return;
    const tick = () => setLeft(Math.max(0, end - (Date.now() + offsetMs)));
    tick();
    const id = setInterval(tick, 200);
    return () => clearInterval(id);
  }, [windowEnd, offsetMs]);

  const total = Math.max(1, durationMs);
  const pct = Math.min(100, Math.max(0, (left / total) * 100));
  return (
    <div className="h-1.5 w-full rounded bg-zinc-800 overflow-hidden">
      <div className="h-full bg-gradient-to-r from-rose-500 to-pink-500 transition-all" style={{ width: `${pct}%` }} />
    </div>
  );
}

/** Canh Nhịp — kim chạy trên thanh; tap một lần. Server chấm bằng NOW(). */
function TapGame({ step, offsetMs, onSubmit, submitting }: { step: RefineActiveStep; offsetMs: number; onSubmit: () => void; submitting: boolean }) {
  const period = step.params.marker_period_ms || 2000;
  const center = step.params.center ?? 0.5;
  const halfWidth = step.params.half_width ?? 0.12;
  const startMs = Date.parse(step.window_start) - offsetMs;
  const [pos, setPos] = useState(0);
  useEffect(() => {
    let raf = 0;
    const loop = () => {
      const elapsed = Math.max(0, Date.now() - startMs);
      const u = (elapsed % period) / period;
      setPos(1 - Math.abs(2 * u - 1));
      raf = requestAnimationFrame(loop);
    };
    raf = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(raf);
  }, [startMs, period]);

  return (
    <div className="w-full space-y-3">
      <p className="text-xs text-zinc-400">Click đúng khi kim đi vào vùng sáng của chu kỳ hỏa hậu.</p>
      <div className="relative h-10 rounded-lg bg-zinc-900 border border-zinc-700 overflow-hidden">
        {/* Vùng ngọt */}
        <div
          className="absolute top-0 bottom-0 bg-emerald-500/25 border-x border-emerald-400/60"
          style={{ left: `${Math.max(0, (center - halfWidth) * 100)}%`, width: `${halfWidth * 2 * 100}%` }}
        />
        {/* Kim */}
        <div className="absolute top-0 bottom-0 w-1 bg-rose-400 shadow-[0_0_8px_rgba(251,113,133,0.9)]" style={{ left: `calc(${pos * 100}% - 2px)` }} />
      </div>
      <button
        onClick={onSubmit}
        disabled={submitting}
        className="w-full py-2.5 rounded-lg bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white font-semibold transition-colors"
      >
        Khai Lò! (Space)
      </button>
      <SpaceKeyHandler onFire={onSubmit} disabled={submitting} />
    </div>
  );
}

function SpaceKeyHandler({ onFire, disabled }: { onFire: () => void; disabled: boolean }) {
  useEffect(() => {
    if (disabled) return;
    const handler = (e: KeyboardEvent) => {
      if (e.code === "Space" && !e.repeat) {
        e.preventDefault();
        onFire();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onFire, disabled]);
  return null;
}

/** Dãy Ấn — hiện 4 ký tự lần lượt rồi nhập lại bằng bàn ấm. */
function SequenceGame({ step, onSubmit, submitting }: { step: RefineActiveStep; onSubmit: (payload: unknown) => void; submitting: boolean }) {
  const symbols = step.params.symbols || [];
  const displayMs = step.params.display_ms || 500;
  const [shown, setShown] = useState(-1); // đếm hiển thị
  const [entered, setEntered] = useState<number[]>([]);

  useEffect(() => {
    const symbols = step?.params?.symbols;
    if (!symbols) return;
    setShown(0);
    setEntered([]);
    let i = 0;
    const timers: ReturnType<typeof setTimeout>[] = [];
    while (i < symbols.length) {
      const idx = i;
      timers.push(setTimeout(() => setShown(idx), idx * displayMs));
      timers.push(setTimeout(() => setShown(idx + 1), (idx + 1) * displayMs));
      i++;
    }
    return () => timers.forEach(clearTimeout);
  }, [symbols, displayMs]);

  const press = useCallback((num: number) => {
    setEntered((prev) => {
      if (prev.length >= symbols.length) return prev;
      const next = [...prev, num];
      if (next.length === symbols.length) {
        onSubmit({ sequence: next });
      }
      return next;
    });
  }, [symbols.length, onSubmit]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const n = parseInt(e.key, 10);
      if (!Number.isNaN(n) && n >= 1 && n <= 4 && !submitting) press(n - 1);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [press, submitting]);

  return (
    <div className="w-full space-y-3">
      <p className="text-xs text-zinc-400">Ghi nhớ dãy ấn rồi nhập lại theo đúng thứ tự (phím 1–4).</p>
      <div className="flex gap-2 justify-center">
        {symbols.map((_, i) => (
          <div key={i} className="w-9 h-9 rounded border border-zinc-700 font-mono flex items-center justify-center bg-zinc-900">
            <span className={`text-lg font-bold ${shown > i ? "text-emerald-400" : "text-zinc-600"}`}>
              {shown > i ? SYMBOL_GLYPHS[symbols[i]] : "·"}
            </span>
          </div>
        ))}
      </div>
      <div className="grid grid-cols-4 gap-2">
        {SYMBOL_GLYPHS.map((glyph, n) => (
          <button
            key={glyph}
            onClick={() => press(n)}
            disabled={submitting || entered.length >= symbols.length}
            className="py-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 border border-zinc-700 font-mono text-lg text-zinc-200"
          >
            {glyph}
          </button>
        ))}
      </div>
      <div className="text-center text-xs text-zinc-500 font-mono">
        {entered.map((n) => SYMBOL_GLYPHS[n]).join(" ") || "—"}
      </div>
    </div>
  );
}

const SYMBOL_GLYPHS = ["玄", "清", "炎", "幽"];

/** Chọn Hành Động — chọn 1 trong các phương án rồi chấm ngay. */
function ChoiceGame({ step, onSubmit, submitting }: { step: RefineActiveStep; onSubmit: (payload: unknown) => void; submitting: boolean }) {
  const options = step.params.options || [];
  return (
    <div className="w-full space-y-3">
      <div className="rounded-lg bg-zinc-900 border border-amber-700/40 p-3 text-sm text-amber-200/90">
        {step.params.situation}
      </div>
      <div className="space-y-2">
        {options.map((opt, idx) => (
          <button
            key={idx}
            onClick={() => onSubmit({ choice_index: idx })}
            disabled={submitting}
            className="w-full text-left px-3 py-2.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 border border-zinc-700 text-sm text-zinc-200"
          >
            <span className="font-mono text-amber-400 mr-2">{idx + 1}.</span>
            {opt}
          </button>
        ))}
      </div>
      <NumberKeyHandler count={options.length} onPick={(n) => onSubmit({ choice_index: n })} disabled={submitting} />
    </div>
  );
}

function NumberKeyHandler({ count, onPick, disabled }: { count: number; onPick: (n: number) => void; disabled: boolean }) {
  useEffect(() => {
    if (disabled) return;
    const handler = (e: KeyboardEvent) => {
      const n = parseInt(e.key, 10);
      if (!Number.isNaN(n) && n >= 1 && n <= count && !e.repeat) {
        e.preventDefault();
        onPick(n - 1);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [count, onPick, disabled]);
  return null;
}

/** Ghép Liệu — click các nguyên liệu theo thứ tự rank tăng dần. */
function OrderGame({ step, onSubmit, submitting }: { step: RefineActiveStep; onSubmit: (payload: unknown) => void; submitting: boolean }) {
  const items = step.params.items || [];
  const [order, setOrder] = useState<string[]>([]);
  const remaining = items.filter((it) => !order.includes(it.id));

  const pick = (id: string) => {
    if (submitting) return;
    const next = [...order, id];
    setOrder(next);
    if (next.length === items.length) {
      onSubmit({ order: next });
    }
  };
  const undo = (id: string) => {
    if (submitting) return;
    setOrder((prev) => prev.filter((x) => x !== id));
  };

  return (
    <div className="w-full space-y-3">
      <p className="text-xs text-zinc-400">Click nguyên liệu theo thứ tự bậc tăng dần (cùng bậc thì theo tên cổ). Hoàn thành tự chấm.</p>
      <div className="grid grid-cols-2 gap-2">
        {remaining.map((it) => (
          <button
            key={it.id}
            onClick={() => pick(it.id)}
            disabled={submitting}
            className="flex items-center gap-2 px-2.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 border border-zinc-700 text-left"
          >
            <span className="w-6 h-6 rounded bg-amber-900/60 border border-amber-700/50 flex items-center justify-center text-[10px] text-amber-300 font-mono">
              R{it.rank}
            </span>
            <span className="text-xs text-zinc-200 truncate">{it.name}</span>
          </button>
        ))}
      </div>
      <div className="min-h-[2rem] flex flex-wrap gap-1.5">
        {order.map((id, idx) => (
          <button
            key={`${id}-${idx}`}
            onClick={() => undo(id)}
            className="px-2 py-1 rounded bg-emerald-900/40 border border-emerald-700/50 text-xs text-emerald-200"
          >
            {idx + 1}. {items.find((it) => it.id === id)?.name ?? id}
          </button>
        ))}
      </div>
    </div>
  );
}

export function RefineSessionPanel() {
  const session = useGameStore((s) => s.refineSession);
  const panelOpen = useGameStore((s) => s.refinePanelOpen);
  const submitRefineStep = useGameStore((s) => s.submitRefineStep);
  const abortRefineSession = useGameStore((s) => s.abortRefineSession);
  const ackRefineSession = useGameStore((s) => s.ackRefineSession);
  const openRefinePanel = useGameStore((s) => s.openRefinePanel);
  const closeRefinePanel = useGameStore((s) => s.closeRefinePanel);

  const offsetMs = useServerOffset(session);
  const [submitting, setSubmitting] = useState(false);

  // Bước kế tiếp mở cửa sổ → tự đồng bộ session ngay khi tới giờ (không phụ
  // thuộc WS vì server chỉ publish khi có thao tác).
  const fetchActiveRefineSession = useGameStore((s) => s.fetchActiveRefineSession);
  const nextStepAt = session?.status === "ACTIVE" ? session.next_step_at : null;
  const nextStepKey = nextStepAt ? String(session?.id) : "";
  useEffect(() => {
    if (!nextStepAt) return;
    const openAt = Date.parse(nextStepAt) - offsetMs;
    if (Number.isNaN(openAt)) return;
    const delay = openAt - Date.now() + 600;
    if (delay <= 0) {
      void fetchActiveRefineSession();
      return;
    }
    const id = setTimeout(() => {
      void fetchActiveRefineSession();
    }, Math.min(delay, 10 * 60_000));
    return () => clearTimeout(id);
  }, [nextStepAt, nextStepKey, offsetMs, fetchActiveRefineSession]);

  const step = session?.step ?? null;
  const cleanupRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Phiên chốt nhưng client chưa thấy result (rare): tự dọn.
  useEffect(() => {
    if (session && session.status !== "ACTIVE" && !session.result) {
      if (cleanupRef.current == null) {
        cleanupRef.current = setTimeout(() => useGameStore.setState({ refineSession: null }), 15000);
      }
      return () => {
        if (cleanupRef.current) {
          clearTimeout(cleanupRef.current);
          cleanupRef.current = null;
        }
      };
    }
  }, [session]);
  const submit = useCallback(async (payload: unknown) => {
    if (!session?.step || submitting) return;
    setSubmitting(true);
    try {
      await submitRefineStep(session.step.index, session.step.nonce, session.step.type, payload);
    } finally {
      setSubmitting(false);
    }
  }, [session?.step, submitting, submitRefineStep]);

  const stepWindowMs = useMemo(() => {
    if (!step || !session) return 20000;
    const a = Date.parse(step.window_start), b = Date.parse(step.window_end);
    return Number.isNaN(a) || Number.isNaN(b) ? 20000 : Math.max(1000, b - a);
  }, [step, session]);

  const stepsForTimeline = useMemo(() => {
    return (session?.steps || []).map((s) => {
      let timeInfo = "";
      if (s.state === "LOCKED" && s.window_start) {
        const openMs = Date.parse(s.window_start);
        if (!Number.isNaN(openMs)) timeInfo = ` · mở ${formatServerClock(openMs, offsetMs)}`;
      } else if (s.state === "MISSED") {
        timeInfo = " · lỡ";
      } else if (s.state === "OPEN" && s.window_end) {
        const endMs = Date.parse(s.window_end);
        if (!Number.isNaN(endMs)) timeInfo = ` · đóng ${formatServerClock(endMs, offsetMs)}`;
      }
      return {
        label: `B${s.index} · ${STEP_TYPE_LABEL[s.type] ?? s.type}${timeInfo}`,
        completed: s.state === "DONE",
      };
    });
  }, [session?.steps, offsetMs]);
  const currentStepIndex = session?.step?.index ?? (session?.status !== "ACTIVE" ? (session?.steps?.length ?? 0) : 0);

  if (!session) return null;
  const finished = session.status !== "ACTIVE";
  // Đồng hồ hiển thị thời gian — gọi hook TRƯỚC mọi return sớm (rules of hooks).
  const nowTick = useNowTick(!finished && panelOpen);


  // ── Result overlay ──
  if (finished && session.result && !session.acknowledged_at) {
    const ok = session.result.outcome === "SUCCESS";
    const breakdown = (session.result.rate_breakdown || {}) as Record<string, unknown>;
    return (
      <div className="fixed inset-0 z-[95] bg-black/80 flex items-center justify-center p-4">
        <div className="max-w-md w-full rounded-2xl border border-zinc-700 bg-zinc-950 p-5 space-y-4">
          <div className={`text-center text-xl font-bold ${ok ? "text-amber-300" : "text-rose-300"}`}>
            {ok ? "🔥 Luyện Cổ Thành Công" : "Luyện Cổ Thất Bại"}
          </div>
          {ok && (
            <div className="text-center text-sm text-zinc-200">
              {typeof session.result.new_gu === "object" && session.result.new_gu
                ? `Cổ ${String((session.result.new_gu as Record<string, unknown>).gu_id ?? "")} R${String((session.result.new_gu as Record<string, unknown>).rank ?? "")} đã chào đời!`
                : ""}
            </div>
          )}
          <p className="text-sm text-zinc-300 text-center">{session.result.message}</p>
          <div className="text-xs text-zinc-500 space-y-1 font-mono">
            <div>Điểm chính xác: {session.accuracy}%</div>
            {"accuracy_bonus" in breakdown && <div>Bonus chính xác: +{Math.round(Number(breakdown.accuracy_bonus || 0) * 1000) / 10}%</div>}
            {"death_penalty" in breakdown && Number(breakdown.death_penalty || 0) > 0 && (
              <div className="text-rose-400">Phạt tâm thần: −{Math.round(Number(breakdown.death_penalty) * 1000) / 10}%</div>
            )}
            {Array.isArray(breakdown.step_scores) && (
              <div>Bước: {(breakdown.step_scores as number[]).map((s) => `${s}`).join(" · ")}</div>
            )}
          </div>
          <button
            onClick={async () => {
              await ackRefineSession();
              useGameStore.setState({ refineSession: null });
            }}
            className="w-full py-2.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-zinc-950 font-semibold"
          >
            Xác nhận
          </button>
        </div>
      </div>
    );
  }

  if (finished && session.acknowledged_at) return null;

  if (!panelOpen) {
    return (
      <button
        onClick={openRefinePanel}
        className="fixed bottom-20 right-4 z-40 px-3 py-2 rounded-full bg-rose-900/80 border border-rose-600/60 text-rose-200 text-xs shadow-lg animate-pulse"
      >
        🔥 Phiên luyện đang chạy
      </button>
    );
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 z-[90] md:bottom-4 md:right-4 md:left-auto md:w-[380px]">
      <div className="rounded-t-2xl md:rounded-2xl border border-zinc-700 bg-zinc-950/95 backdrop-blur p-4 space-y-3 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="text-sm font-semibold text-rose-300 tracking-wide">
            🔥 Phiên Luyện Cổ — R{session.target_rank}
          </div>
          <div className="flex items-center gap-2">
            {!finished && (() => {
              const endsMs = Date.parse(session.ends_at);
              if (Number.isNaN(endsMs)) return null;
              const remain = endsMs - (nowTick + offsetMs);
              return (
                <span className="font-mono text-[11px] text-amber-300/90" title="Thời gian còn lại của cả phiên">
                  ⏳ {formatDuration(remain)}
                </span>
              );
            })()}
            <button onClick={closeRefinePanel} className="text-zinc-400 hover:text-zinc-200 text-lg leading-none" aria-label="Thu nhỏ">
              −
            </button>
            <button
              onClick={() => { if (!submitting) void abortRefineSession(); }}
              className="text-[11px] px-2 py-1 rounded bg-zinc-800 hover:bg-rose-900/60 text-zinc-400 hover:text-rose-300 border border-zinc-700"
            >
              Hủy
            </button>
          </div>
        </div>

        {/* Timeline */}
        <GameStepProgress steps={stepsForTimeline} currentStep={currentStepIndex} variant="refinement" />

        {/* Active minigame */}
        {step ? (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="font-mono text-amber-400">
                Bước {step.index} · {STEP_TYPE_LABEL[step.type] ?? step.type}
              </span>
              <span className="text-zinc-500">Điểm hiện tại: {session.accuracy}%</span>
            </div>
            <CountdownBar windowEnd={step.window_end} offsetMs={offsetMs} durationMs={stepWindowMs} />
            {submitting && <div className="text-center text-xs text-amber-400">Đang nạp tại lò…</div>}
            {step.type === "TAP" && <TapGame step={step} offsetMs={offsetMs} onSubmit={() => void submit({})} submitting={submitting} />}
            {step.type === "SEQUENCE" && <SequenceGame step={step} onSubmit={(p) => void submit(p)} submitting={submitting} />}
            {step.type === "CHOICE" && <ChoiceGame step={step} onSubmit={(p) => void submit(p)} submitting={submitting} />}
            {step.type === "ORDER" && <OrderGame step={step} onSubmit={(p) => void submit(p)} submitting={submitting} />}
          </div>
        ) : session.next_step_at ? (
          (() => {
            const openMs = Date.parse(session.next_step_at);
            const openShown = Number.isNaN(openMs) ? null : formatServerClock(openMs, offsetMs);
            const left = Number.isNaN(openMs) ? 0 : openMs - (nowTick + offsetMs);
            return (
              <div className="text-center text-xs text-zinc-400 py-3 space-y-1">
                <div className="font-mono text-amber-300/90 text-sm">
                  Bước kế tiếp mở sau <span className="font-bold">{openMs && left > 0 ? formatDuration(left) : "0:00"}</span>
                  {openShown && <span className="text-zinc-500"> ({openShown})</span>}
                </div>
                <div className="text-zinc-500">
                  Phiên đang duy trì hỏa hậu trong lúc đạo hữu làm việc khác — lỡ cửa sổ bước nào bước đó tính 0 điểm.
                </div>
              </div>
            );
          })()
        ) : (
          <div className="text-center text-xs text-zinc-500 py-4">Đang chờ server chốt kết quả…</div>
        )}
      </div>
    </div>
  );
}


