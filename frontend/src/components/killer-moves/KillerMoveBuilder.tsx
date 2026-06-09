"use client";

import { useEffect, useMemo, useState } from "react";
import { GuInstance, GuSpec, Player, useGameStore } from "@/store/useGameStore";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type KillerMoveSlotPreview = {
  player_gu_id: string;
  gu_id: string;
  gu_name: string;
  slot_type: "CORE" | "SUPPORT";
  slot_index: number;
  compatibility_score: number;
  applied: boolean;
  warnings: string[];
};

type KillerMovePreview = {
  name?: string;
  path: string;
  move_type: string;
  rank_required: number;
  total_mana_cost: number;
  stability: number;
  damage_value: number;
  defense_value: number;
  cooldown: number;
  cast_time: number;
  effects: Record<string, number>;
  warnings: string[];
  slot_results: KillerMoveSlotPreview[];
};

type BuilderSlot = {
  type: "CORE" | "SUPPORT";
  index: number;
  gu: GuInstance | null;
};

const apiFetch = (input: RequestInfo | URL, init: RequestInit = {}) =>
  fetch(input, {
    ...init,
    credentials: "include",
  });

export function KillerMoveBuilder() {
  const { player, guDictionary, initializeSession, fetchDictionary } = useGameStore();
  const [coreGu, setCoreGu] = useState<GuInstance | null>(null);
  const [supportGus, setSupportGus] = useState<(GuInstance | null)[]>(Array(8).fill(null));
  const [preview, setPreview] = useState<KillerMovePreview | null>(null);
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isCalculating, setIsCalculating] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [savedMessage, setSavedMessage] = useState<string | null>(null);

  useEffect(() => {
    void fetchDictionary();
    void initializeSession();
  }, [fetchDictionary, initializeSession]);

  const usableGus = useMemo(() => {
    const selectedIds = new Set([coreGu?.instance_id, ...supportGus.map((gu) => gu?.instance_id)].filter(Boolean));
    return (player?.gu_inventory || []).filter((gu) => gu.status === "ACTIVE" && !selectedIds.has(gu.instance_id));
  }, [coreGu, player?.gu_inventory, supportGus]);

  useEffect(() => {
    if (!coreGu) {
      const resetTimer = window.setTimeout(() => setPreview(null), 0);
      return () => window.clearTimeout(resetTimer);
    }

    const timer = window.setTimeout(async () => {
      setIsCalculating(true);
      setError(null);
      try {
        const response = await apiFetch(`${BACKEND_URL}/api/killer-moves/calculate`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            core_player_gu_id: coreGu.instance_id,
            support_player_gu_ids: supportGus.filter(Boolean).map((gu) => gu!.instance_id),
          }),
        });
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || "Không thể tính sát chiêu");
        setPreview(data);
      } catch (err) {
        setPreview(null);
        setError(err instanceof Error ? err.message : "Không thể tính sát chiêu");
      } finally {
        setIsCalculating(false);
      }
    }, 300);

    return () => window.clearTimeout(timer);
  }, [coreGu, supportGus]);

  const saveKillerMove = async () => {
    if (!coreGu) return;
    setIsSaving(true);
    setError(null);
    setSavedMessage(null);
    try {
      const response = await apiFetch(`${BACKEND_URL}/api/killer-moves`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim() || `Sát chiêu ${guName(coreGu, guDictionary)}`,
          move_type: preview?.move_type || "UTILITY",
          core_player_gu_id: coreGu.instance_id,
          support_player_gu_ids: supportGus.filter(Boolean).map((gu) => gu!.instance_id),
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể lưu sát chiêu");
      setSavedMessage("Đã lưu sát chiêu vào đạo kho.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không thể lưu sát chiêu");
    } finally {
      setIsSaving(false);
    }
  };

  if (!player) {
    return <div className="rounded-2xl border border-amber-500/30 bg-slate-950 p-6 text-amber-100">Đăng nhập để xây dựng sát chiêu.</div>;
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[1.3fr_0.7fr]">
      <section className="rounded-3xl border border-purple-500/30 bg-slate-950/90 p-6 shadow-2xl shadow-purple-950/30">
        <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-sm uppercase tracking-[0.3em] text-purple-300">Killer Move System</p>
            <h1 className="text-3xl font-bold text-amber-100">Sát Chiêu Đường</h1>
            <p className="mt-2 text-sm text-slate-300">Đặt Cổ hạch tâm ở giữa, tối đa 8 Cổ phụ trợ xung quanh. Preview tự tính sau 300ms.</p>
          </div>
          <input
            className="rounded-xl border border-purple-400/40 bg-slate-900 px-4 py-3 text-amber-50 outline-none focus:border-amber-300"
            placeholder="Tên sát chiêu"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </div>

        <div className="grid gap-6 lg:grid-cols-[1fr_280px]">
          <div className="relative mx-auto grid h-[420px] w-full max-w-[520px] place-items-center rounded-full border border-purple-400/20 bg-[radial-gradient(circle,rgba(168,85,247,0.20),rgba(15,23,42,0.20)_45%,rgba(2,6,23,0.80))]">
            {supportGus.map((gu, index) => (
              <GuSlot
                key={index}
                slot={{ type: "SUPPORT", index, gu }}
                guDictionary={guDictionary}
                onDropGu={(dropped) => setSupportGus((current) => current.map((item, idx) => (idx === index ? dropped : item)))}
                onClear={() => setSupportGus((current) => current.map((item, idx) => (idx === index ? null : item)))}
              />
            ))}
            <GuSlot
              slot={{ type: "CORE", index: 0, gu: coreGu }}
              guDictionary={guDictionary}
              onDropGu={setCoreGu}
              onClear={() => setCoreGu(null)}
            />
          </div>

          <GuInventoryPanel gus={usableGus} guDictionary={guDictionary} />
        </div>
      </section>

      <KillerMovePreviewPanel
        preview={preview}
        error={error}
        isCalculating={isCalculating}
        isSaving={isSaving}
        savedMessage={savedMessage}
        canSave={Boolean(coreGu && preview)}
        onSave={saveKillerMove}
        player={player}
      />
    </div>
  );
}

function GuSlot({ slot, guDictionary, onDropGu, onClear }: { slot: BuilderSlot; guDictionary: Record<string, GuSpec>; onDropGu: (gu: GuInstance) => void; onClear: () => void }) {
  const angle = (slot.index / 8) * Math.PI * 2 - Math.PI / 2;
  const radius = 160;
  const style = slot.type === "CORE" ? undefined : { transform: `translate(${Math.cos(angle) * radius}px, ${Math.sin(angle) * radius}px)` };

  return (
    <button
      className={`absolute grid min-h-24 w-32 place-items-center rounded-2xl border p-3 text-center transition hover:scale-105 ${slot.type === "CORE" ? "border-amber-300 bg-amber-500/15 text-amber-100" : "border-purple-300/50 bg-purple-950/70 text-purple-100"}`}
      style={style}
      onDragOver={(event) => event.preventDefault()}
      onDrop={(event) => {
        event.preventDefault();
        const raw = event.dataTransfer.getData("application/json");
        if (raw) onDropGu(JSON.parse(raw));
      }}
      onClick={() => slot.gu && onClear()}
      title={slot.gu ? "Click để gỡ Cổ" : "Kéo Cổ vào đây"}
    >
      <span className="text-xs uppercase tracking-widest opacity-70">{slot.type === "CORE" ? "Core Gu" : `Support ${slot.index + 1}`}</span>
      <strong className="text-sm">{slot.gu ? guName(slot.gu, guDictionary) : "Trống"}</strong>
      {slot.gu && <EffectChips effects={(slot.type === "CORE" ? guDictionary[slot.gu.gu_id]?.effects?.active : guDictionary[slot.gu.gu_id]?.effects?.support) || {}} />}
    </button>
  );
}

function GuInventoryPanel({ gus, guDictionary }: { gus: GuInstance[]; guDictionary: Record<string, GuSpec> }) {
  return (
    <aside className="rounded-2xl border border-slate-700 bg-slate-900/80 p-4">
      <h2 className="mb-3 font-semibold text-amber-100">Túi Cổ khả dụng</h2>
      <div className="max-h-[360px] space-y-2 overflow-y-auto pr-1">
        {gus.length === 0 && <p className="text-sm text-slate-400">Không có Cổ ACTIVE khả dụng.</p>}
        {gus.map((gu) => (
          <div
            key={gu.instance_id}
            draggable
            onDragStart={(event) => event.dataTransfer.setData("application/json", JSON.stringify(gu))}
            className="cursor-grab rounded-xl border border-slate-700 bg-slate-950 p-3 active:cursor-grabbing"
          >
            <div className="font-medium text-amber-100">{guName(gu, guDictionary)}</div>
            <div className="text-xs text-slate-400">{guDictionary[gu.gu_id]?.path || "Unknown Path"} · Hunger {gu.hunger}</div>
          </div>
        ))}
      </div>
    </aside>
  );
}

function KillerMovePreviewPanel({ preview, error, isCalculating, isSaving, savedMessage, canSave, onSave, player }: { preview: KillerMovePreview | null; error: string | null; isCalculating: boolean; isSaving: boolean; savedMessage: string | null; canSave: boolean; onSave: () => void; player: Player }) {
  return (
    <aside className="rounded-3xl border border-amber-500/30 bg-slate-950/90 p-6 shadow-2xl shadow-amber-950/20">
      <h2 className="text-2xl font-bold text-amber-100">Preview</h2>
      {isCalculating && <p className="mt-3 text-sm text-purple-200">Đang diễn toán cấu trúc...</p>}
      {error && <p className="mt-3 rounded-xl border border-red-500/40 bg-red-950/40 p-3 text-sm text-red-100">{error}</p>}
      {savedMessage && <p className="mt-3 rounded-xl border border-emerald-500/40 bg-emerald-950/40 p-3 text-sm text-emerald-100">{savedMessage}</p>}
      {preview ? (
        <div className="mt-5 space-y-4 text-sm text-slate-200">
          <StabilityWarning stability={preview.stability} />
          <Stat label="Chân/Tiên nguyên" value={`${preview.total_mana_cost.toFixed(1)} / ${player.current_mana.toFixed(1)}`} />
          <Stat label="Sát thương" value={preview.damage_value.toFixed(1)} />
          <Stat label="Phòng ngự" value={preview.defense_value.toFixed(1)} />
          <Stat label="Lưu phái" value={preview.path} />
          <Stat label="Loại" value={preview.move_type} />
          <Stat label="Cooldown" value={`${preview.cooldown} lượt`} />
          <EffectSection title="Applied Killer Move Effects" effects={preview.effects || {}} emptyText="Không có hiệu ứng được cộng." />
          {preview.slot_results?.length > 0 && (
            <div>
              <h3 className="mb-2 font-semibold text-amber-100">Slot Compatibility</h3>
              <div className="space-y-2">
                {preview.slot_results.map((slot) => (
                  <div key={`${slot.slot_type}-${slot.slot_index}-${slot.player_gu_id}`} className={`rounded-xl border p-3 text-xs ${slot.applied ? "border-emerald-500/30 bg-emerald-950/20" : "border-red-500/40 bg-red-950/30"}`}>
                    <div className="flex justify-between gap-3 text-slate-100">
                      <span>{slot.slot_type} · {slot.gu_name}</span>
                      <strong>{slot.compatibility_score}</strong>
                    </div>
                    {!slot.applied && <p className="mt-1 text-red-200">Không cộng support effect.</p>}
                    {slot.warnings.map((warning) => <p key={warning} className="mt-1 text-amber-200">{warning}</p>)}
                  </div>
                ))}
              </div>
            </div>
          )}
          <button
            disabled={!canSave || isSaving}
            onClick={onSave}
            className="w-full rounded-xl bg-amber-400 px-4 py-3 font-bold text-slate-950 transition hover:bg-amber-300 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSaving ? "Đang lưu..." : "Lưu sát chiêu"}
          </button>
        </div>
      ) : (
        <p className="mt-5 text-sm text-slate-400">Kéo một Cổ vào slot hạch tâm để bắt đầu.</p>
      )}
    </aside>
  );
}

function StabilityWarning({ stability }: { stability: number }) {
  const tone = stability >= 80 ? "border-emerald-400/50 bg-emerald-950/40 text-emerald-100" : stability >= 50 ? "border-yellow-400/50 bg-yellow-950/40 text-yellow-100" : "border-red-400/50 bg-red-950/40 text-red-100";
  const label = stability >= 80 ? "Ổn định" : stability >= 50 ? "Có rủi ro" : "Nguy hiểm";
  return <div className={`rounded-xl border p-4 ${tone}`}><strong>{label}</strong><span className="float-right">{stability.toFixed(1)}%</span></div>;
}

function EffectSection({ title, effects, emptyText }: { title: string; effects: Record<string, number>; emptyText: string }) {
  const entries = Object.entries(effects);
  return (
    <div>
      <h3 className="mb-2 font-semibold text-amber-100">{title}</h3>
      <div className="flex flex-wrap gap-2 rounded-xl bg-slate-900 p-3 text-xs text-slate-300">
        {entries.length === 0 ? emptyText : entries.map(([key, value]) => <span key={key} className="rounded-full border border-purple-400/30 bg-purple-950/30 px-2 py-1">{key}: {value}</span>)}
      </div>
    </div>
  );
}

function EffectChips({ effects }: { effects: Record<string, number> }) {
  const entries = Object.entries(effects).slice(0, 2);
  if (entries.length === 0) return null;
  return <span className="mt-1 text-[10px] text-slate-300">{entries.map(([key, value]) => `${key} ${value}`).join(" · ")}</span>;
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div className="flex justify-between rounded-xl bg-slate-900 px-4 py-3"><span className="text-slate-400">{label}</span><strong className="text-amber-100">{value}</strong></div>;
}

function guName(gu: GuInstance, guDictionary: Record<string, GuSpec>) {
  return guDictionary[gu.gu_id]?.name || gu.gu_id;
}
