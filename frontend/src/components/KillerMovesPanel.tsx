"use client";

import { useCallback, useEffect, useMemo, useState, type DragEvent } from "react";
import { BookOpenCheck, Check, GripVertical, Plus, Save, Search, ShieldAlert, Sparkles, Trash2, WandSparkles, Castle } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import type { KillerMoveDraft, KillerMoveEffectOption, KillerMovePreview } from "@/store/slices/killerMovesSlice";
import { GuHousePanel } from "@/components/GuHousePanel";

const formatPercent = (value?: number) => `${Math.round((value || 0) * 100)}%`;

export function KillerMovesPanel() {
  const guDictionary = useGameStore((s) => s.guDictionary);
  const gus = useGameStore((s) => s.killerMoveGus);
  const effects = useGameStore((s) => s.killerMoveEffects);
  const moves = useGameStore((s) => s.killerMoves);
  const loading = useGameStore((s) => s.killerMovesLoading);
  const maxSaved = useGameStore((s) => s.killerMovesMaxSaved);
  const maxEquipped = useGameStore((s) => s.killerMovesMaxEquipped);
  const fetch = useGameStore((s) => s.fetchKillerMoves);
  const previewDraft = useGameStore((s) => s.previewKillerMove);
  const create = useGameStore((s) => s.createKillerMove);
  const toggleEquip = useGameStore((s) => s.toggleKillerMoveEquip);
  const remove = useGameStore((s) => s.deleteKillerMove);
  const useMove = useGameStore((s) => s.useKillerMove);

  const [mainId, setMainId] = useState("");
  const [mainSearch, setMainSearch] = useState("");
  const [components, setComponents] = useState<string[]>([]);
  const [componentSearch, setComponentSearch] = useState("");
  const [selectedEffects, setSelectedEffects] = useState<string[]>([]);
  const [name, setName] = useState("");
  const [preview, setPreview] = useState<KillerMovePreview | null>(null);
  const [previewError, setPreviewError] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<"spells" | "houses">("spells");

  useEffect(() => { void fetch(); }, [fetch]);

  useEffect(() => {
    if (!mainId && gus.length > 0) setMainId(gus[0].instance_id);
  }, [gus, mainId]);

  const selectedGuIDs = useMemo(() => new Set([mainId, ...components]), [mainId, components]);
  const availableEffects = useMemo(
    () => effects.filter((effect) => selectedGuIDs.has(effect.source_instance_id)),
    [effects, selectedGuIDs],
  );
  const selectedEffectOptions = useMemo(
    () => selectedEffects.map((key) => effects.find((effect) => effect.key === key)).filter((effect): effect is KillerMoveEffectOption => !!effect),
    [effects, selectedEffects],
  );
  const guName = useCallback((id: string, rank: number) => {
    const gu = gus.find((entry) => entry.instance_id === id);
    const spec = gu ? guDictionary[gu.gu_id] : undefined;
    return `${spec?.name || gu?.gu_id || "Cổ trùng"} · Rank ${rank}`;
  }, [guDictionary, gus]);
  const filteredMainGus = useMemo(() => {
    const query = mainSearch.trim().toLowerCase();
    return gus.filter((gu) => {
      const spec = guDictionary[gu.gu_id];
      return !query || `${spec?.name || ""} ${gu.gu_id}`.toLowerCase().includes(query);
    });
  }, [gus, guDictionary, mainSearch]);
  const filteredComponentGus = useMemo(() => {
    const query = componentSearch.trim().toLowerCase();
    return gus.filter((gu) => {
      if (gu.instance_id === mainId || components.includes(gu.instance_id)) return false;
      const spec = guDictionary[gu.gu_id];
      return !query || `${spec?.name || ""} ${gu.gu_id}`.toLowerCase().includes(query);
    });
  }, [gus, guDictionary, mainId, components, componentSearch]);

  const draft = useMemo<KillerMoveDraft | null>(() => {
    if (!mainId || selectedEffects.length === 0) return null;
    return { main_gu_instance_id: mainId, component_instance_ids: components, selected_effect_keys: selectedEffects };
  }, [mainId, components, selectedEffects]);

  useEffect(() => {
    if (!draft) { setPreview(null); setPreviewError(""); return; }
    let live = true;
    const timer = window.setTimeout(() => {
      void previewDraft(draft).then((result) => {
        if (!live) return;
        setPreview(result);
        setPreviewError(result ? "" : "Công thức chưa hợp lệ hoặc vượt giới hạn.");
      });
    }, 250);
    return () => { live = false; window.clearTimeout(timer); };
  }, [draft, previewDraft]);

  const addEffect = (key: string) => {
    setSelectedEffects((current) => current.includes(key) || current.length >= 8 ? current : [...current, key]);
  };
  const removeEffect = (key: string) => setSelectedEffects((current) => current.filter((entry) => entry !== key));
  const onDrop = (event: DragEvent) => {
    event.preventDefault(); setDragOver(false);
    const key = event.dataTransfer.getData("text/killer-move-effect");
    if (key) addEffect(key);
  };

  const saveMove = async () => {
    if (!draft || !preview || !name.trim()) return;
    setSaving(true);
    try {
      if (await create({ ...draft, name: name.trim() })) {
        setName(""); setComponents([]); setSelectedEffects([]); setPreview(null);
      }
    } finally { setSaving(false); }
  };

  const equippedCount = moves.filter((move) => move.is_equipped).length;
  const selectedMain = gus.find((gu) => gu.instance_id === mainId);

  return (
    <div className="space-y-5 text-left">
      <div className="flex border-b border-zinc-800">
        <button
          type="button"
          onClick={() => setViewMode("spells")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-colors ${
            viewMode === "spells"
              ? "border-amber-500 text-amber-200"
              : "border-transparent text-zinc-500 hover:text-zinc-300"
          }`}
        >
          <WandSparkles className="h-4 w-4" />
          Sát Chiêu Cá Nhân
        </button>
        <button
          type="button"
          onClick={() => setViewMode("houses")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-colors ${
            viewMode === "houses"
              ? "border-amber-500 text-amber-200"
              : "border-transparent text-zinc-500 hover:text-zinc-300"
          }`}
        >
          <Castle className="h-4 w-4" />
          Tiên Cổ Ốc (Pháo Đài)
        </button>
      </div>

      {viewMode === "houses" ? (
        <GuHousePanel />
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 className="flex items-center gap-2 text-base font-bold text-amber-200"><WandSparkles className="h-5 w-5 text-amber-400" />Sáng Tạo Sát Chiêu</h3>
              <p className="mt-1 text-xs text-zinc-400">Chọn Cổ, kéo hiệu ứng vào công thức. Hệ thống tự cân bằng theo rank và độ hòa hợp.</p>
            </div>
            <div className="text-right text-[11px] font-mono text-zinc-400">Đã lưu {moves.length}/{maxSaved} · Đang trang bị {equippedCount}/{maxEquipped}</div>
          </div>

          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,.8fr)]">
        <section className="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-950/70 p-4">
          <div>
            <label className="mb-1.5 block text-[11px] font-mono uppercase tracking-wider text-zinc-400">Cổ chính</label>
            <div className="mb-2 rounded-xl border border-amber-900/40 bg-amber-950/15 px-3 py-2 text-sm text-amber-100">
              {mainId ? `${guName(mainId, selectedMain?.rank || 1)} · No ${Math.floor(selectedMain?.hunger || 0)}` : "Chưa chọn Cổ chính"}
            </div>
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
              <input
                type="search"
                value={mainSearch}
                onChange={(event) => setMainSearch(event.target.value)}
                placeholder="Tìm tên hoặc mã Cổ..."
                aria-label="Tìm Cổ chính"
                className="w-full rounded-xl border border-zinc-700 bg-zinc-900 py-2.5 pl-9 pr-3 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-amber-500"
              />
            </div>
            <div className="mt-2 max-h-40 space-y-1 overflow-y-auto rounded-xl border border-zinc-800 bg-zinc-900/70 p-1">
              {filteredMainGus.length === 0 ? <p className="px-3 py-2 text-xs text-zinc-500">Không tìm thấy Cổ phù hợp.</p> : filteredMainGus.map((gu) => (
                <button
                  key={gu.instance_id}
                  type="button"
                  aria-pressed={mainId === gu.instance_id}
                  onClick={() => { setMainId(gu.instance_id); setComponents([]); setSelectedEffects([]); setComponentSearch(""); }}
                  className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-xs transition-colors ${mainId === gu.instance_id ? "bg-amber-950/50 text-amber-200" : "text-zinc-300 hover:bg-zinc-800"}`}
                >
                  <span className="truncate">{guName(gu.instance_id, gu.rank)}</span>
                  <span className="ml-2 shrink-0 text-[10px] text-zinc-500">No {Math.floor(gu.hunger)}</span>
                </button>
              ))}
            </div>
          </div>

          <div>
            <div className="mb-2 flex items-center justify-between"><span className="text-[11px] font-mono uppercase tracking-wider text-zinc-400">Cổ phụ tùy chọn</span><span className="text-[10px] text-zinc-600">{components.length + (mainId ? 1 : 0)}/6</span></div>
            <div className="flex flex-wrap gap-2">
              {components.map((id) => {
                const gu = gus.find((entry) => entry.instance_id === id);
                return <button key={id} onClick={() => { setComponents((current) => current.filter((value) => value !== id)); setSelectedEffects((current) => current.filter((key) => effects.find((effect) => effect.key === key)?.source_instance_id !== id)); }} className="rounded-lg border border-cyan-900/60 bg-cyan-950/30 px-2.5 py-1.5 text-[11px] text-cyan-200 hover:border-red-800">{guName(id, gu?.rank || 1)} ×</button>;
              })}
            </div>
            {components.length < 5 && <div className="mt-2">
              <div className="relative">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-500" />
                <input
                  type="search"
                  value={componentSearch}
                  onChange={(event) => setComponentSearch(event.target.value)}
                  placeholder="+ Tìm Cổ phụ theo tên hoặc mã..."
                  aria-label="Tìm Cổ phụ"
                  className="w-full rounded-lg border border-dashed border-zinc-700 bg-zinc-900 py-2 pl-8 pr-3 text-xs text-zinc-200 outline-none placeholder:text-zinc-600 focus:border-amber-500"
                />
              </div>
              <div className="mt-1 max-h-36 space-y-1 overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-900/70 p-1">
                {filteredComponentGus.length === 0 ? <p className="px-2.5 py-2 text-[11px] text-zinc-500">Không tìm thấy Cổ phụ phù hợp.</p> : filteredComponentGus.map((gu) => (
                  <button
                    key={gu.instance_id}
                    type="button"
                    onClick={() => { setComponents((current) => [...current, gu.instance_id]); setComponentSearch(""); }}
                    className="w-full rounded-md px-2.5 py-1.5 text-left text-[11px] text-zinc-300 hover:bg-zinc-800"
                  >
                    {guName(gu.instance_id, gu.rank)}
                  </button>
                ))}
              </div>
            </div>}
            {selectedMain && <p className="mt-2 text-[10px] text-zinc-600">Cổ thành phần vẫn được dùng trong các sát chiêu khác; nếu bị mất hoặc chết, chiêu này sẽ tạm mất hiệu lực.</p>}
          </div>

          <div>
            <div className="mb-2 flex items-center justify-between"><span className="text-[11px] font-mono uppercase tracking-wider text-zinc-400">Hiệu ứng có thể chọn</span><span className="text-[10px] text-zinc-600">Kéo hoặc bấm thêm</span></div>
            {availableEffects.length === 0 ? <div className="rounded-xl border border-dashed border-zinc-800 p-4 text-center text-xs text-zinc-600">Cổ đã chọn chưa có hiệu ứng được hỗ trợ.</div> : <div className="grid max-h-72 grid-cols-1 gap-2 overflow-y-auto sm:grid-cols-2">
              {availableEffects.map((effect) => <div key={effect.key} draggable onDragStart={(event) => event.dataTransfer.setData("text/killer-move-effect", effect.key)} className="group flex items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900/80 p-2.5 cursor-grab active:cursor-grabbing">
                <GripVertical className="h-4 w-4 shrink-0 text-zinc-600" />
                <div className="min-w-0 flex-1"><div className="truncate text-xs font-semibold text-zinc-200">{effect.label}</div><div className="truncate text-[10px] text-zinc-500">{effect.source_name} · {effect.operation.value ? `Giá trị ${effect.operation.value}` : effect.operation.status_id || effect.operation.sub_type || effect.operation.op}</div></div>
                <button aria-label="Thêm hiệu ứng" onClick={() => addEffect(effect.key)} disabled={selectedEffects.includes(effect.key) || selectedEffects.length>=8} className="rounded-md border border-zinc-700 p-1 text-zinc-400 hover:border-amber-600 hover:text-amber-300 disabled:opacity-30"><Plus className="h-3.5 w-3.5" /></button>
              </div>)}
            </div>}
          </div>
        </section>

        <section className={`flex min-h-64 flex-col rounded-2xl border p-4 transition-colors ${dragOver ? "border-amber-400 bg-amber-950/20" : "border-amber-900/50 bg-zinc-950/80"}`} onDragOver={(event) => { event.preventDefault(); setDragOver(true); }} onDragLeave={() => setDragOver(false)} onDrop={onDrop}>
          <div className="mb-3 flex items-center justify-between"><h4 className="flex items-center gap-2 text-sm font-bold text-amber-200"><Sparkles className="h-4 w-4" />Công Thức</h4><span className="text-[10px] text-zinc-500">{selectedEffects.length}/8 hiệu ứng</span></div>
          {selectedEffectOptions.length === 0 ? <div className="flex flex-1 items-center justify-center rounded-xl border border-dashed border-zinc-800 p-5 text-center text-xs text-zinc-600">Kéo hiệu ứng thả vào đây<br />hoặc bấm dấu + cạnh hiệu ứng.</div> : <div className="mb-4 space-y-2">
            {selectedEffectOptions.map((effect, index) => <div key={effect.key} className="flex items-center gap-2 rounded-xl border border-amber-900/40 bg-amber-950/15 px-3 py-2"><span className="font-mono text-[10px] text-amber-500">{index+1}</span><div className="min-w-0 flex-1"><div className="text-xs font-semibold text-zinc-200">{effect.label}</div><div className="truncate text-[10px] text-zinc-500">Nguồn: {effect.source_name} · {effect.target.selector}</div></div><button onClick={() => removeEffect(effect.key)} className="text-zinc-600 hover:text-red-300" aria-label="Bỏ hiệu ứng"><Trash2 className="h-3.5 w-3.5" /></button></div>)}
          </div>}

          {preview && <div className="mb-4 grid grid-cols-2 gap-2 text-[11px]">
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Tỉ lệ thành công</div><div className="mt-0.5 font-mono text-emerald-300">{formatPercent(preview.success_chance)}</div></div>
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Hòa hợp</div><div className="mt-0.5 font-mono text-cyan-300">{formatPercent(preview.harmony)}</div></div>
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Phạm vi mục tiêu</div><div className="mt-0.5 font-mono text-amber-300">{preview.combat_profile?.target_type === "ALL_ENEMIES" ? "Toàn bộ địch (AoE)" : preview.combat_profile?.target_type === "ROW" ? "Hàng ngang (ROW)" : "Đơn thể"}</div></div>
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Hồi chiêu</div><div className="mt-0.5 font-mono text-sky-300">{preview.combat_profile?.cooldown_turns || 2} lượt</div></div>
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Chi phí chân nguyên</div><div className="mt-0.5 font-mono text-violet-300">{preview.mana_cost.toFixed(1)}</div></div>
            <div className="rounded-lg bg-zinc-900 p-2"><div className="text-zinc-500">Rủi ro phản phệ</div><div className="mt-0.5 font-mono text-rose-400">Mất {formatPercent(preview.combat_profile?.backfire_hp_pct || 0.12)} HP</div></div>
            {preview.combat_profile?.warmup_turns ? <div className="col-span-2 rounded-lg border border-purple-900/50 bg-purple-950/20 p-2 text-purple-200">⚡ Sát chiêu Tiên đạo: Cần {preview.combat_profile.warmup_turns} lượt ngưng tụ chân nguyên trước khi bộc phát.</div> : null}
            {preview.effect_scale < 0.999 && <div className="col-span-2 rounded-lg border border-amber-900/50 bg-amber-950/20 p-2 text-amber-200">Uy lực effect sau cân bằng: {formatPercent(preview.effect_scale)}</div>}
            {preview.warnings?.map((warning) => <div key={warning} className="col-span-2 flex gap-1.5 text-amber-300"><ShieldAlert className="h-3.5 w-3.5 shrink-0" />{warning}</div>)}
          </div>}
          {previewError && <p className="mb-3 text-[11px] text-red-300">{previewError}</p>}
          <div className="mt-auto space-y-2">
            <input value={name} onChange={(event) => setName(event.target.value)} maxLength={80} placeholder="Đặt tên sát chiêu" className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-xs text-zinc-100 outline-none focus:border-amber-500" />
            <button onClick={() => void saveMove()} disabled={saving || !preview || !name.trim() || moves.length>=maxSaved} className="flex w-full items-center justify-center gap-2 rounded-lg bg-amber-600 px-3 py-2.5 text-xs font-bold text-white hover:bg-amber-500 disabled:cursor-not-allowed disabled:opacity-40"><Save className="h-4 w-4" />{saving ? "Đang lưu…" : "Lưu Sát Chiêu"}</button>
          </div>
        </section>
      </div>

      <section className="space-y-3">
        <div className="flex items-center gap-2 text-sm font-bold text-zinc-200"><BookOpenCheck className="h-4 w-4 text-cyan-300" />Sát Chiêu Đã Sáng Tạo</div>
        {loading ? <div className="py-6 text-center text-xs text-zinc-500">Đang tải danh sách…</div> : moves.length === 0 ? <div className="rounded-xl border border-dashed border-zinc-800 p-5 text-center text-xs text-zinc-600">Chưa có sát chiêu nào.</div> : <div className="grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {moves.map((move) => <article key={move.id} className="rounded-xl border border-zinc-800 bg-zinc-950/70 p-3.5">
            <div className="flex items-start justify-between gap-3"><div><h4 className="text-sm font-bold text-zinc-100">{move.name}</h4><p className="mt-1 text-[10px] text-zinc-500">Rank {move.main_gu_rank} · {move.components.length} Cổ · {move.combat_profile?.target_type === "ALL_ENEMIES" ? "AoE Toàn thể" : move.combat_profile?.target_type === "ROW" ? "Hàng ngang" : "Đơn thể"} · Hồi chiêu {move.combat_profile?.cooldown_turns || 2} lượt</p></div>{move.is_equipped && <span className="rounded-full border border-emerald-900 bg-emerald-950/40 px-2 py-0.5 text-[9px] text-emerald-300">Đang trang bị</span>}</div>
            <div className="mt-3 flex flex-wrap gap-1.5">{move.components.map((component) => <span key={component.instance_id} className="rounded-md bg-zinc-900 px-2 py-1 text-[9px] text-zinc-400">{guName(component.instance_id, component.rank)}</span>)}</div>
            <div className="mt-3 flex items-center justify-between text-[10px] font-mono"><span className={move.available ? "text-emerald-400" : "text-red-300"}>{move.available ? <><Check className="mr-1 inline h-3 w-3" />Sẵn sàng</> : "Thiếu Cổ thành phần"}</span><span className="text-cyan-300">Thành thạo {move.mastery_level} · {move.mastery_xp} XP</span></div>
            <div className="mt-2 text-[10px] text-zinc-500">Thành công hiện tại: <span className="text-amber-200">{formatPercent(move.success_chance)}</span> · Hòa hợp {formatPercent(move.harmony)}</div>
            <div className="mt-3 flex gap-2">
              <button onClick={() => void toggleEquip(move.id)} disabled={!move.available || (!move.is_equipped && equippedCount>=maxEquipped)} className={`flex-1 rounded-lg border px-2 py-2 text-[10px] font-bold disabled:opacity-40 ${move.is_equipped ? "border-zinc-700 text-zinc-300 hover:bg-zinc-900" : "border-emerald-900 bg-emerald-950/30 text-emerald-300 hover:bg-emerald-950/50"}`}>{move.is_equipped ? "Tháo trang bị" : "Trang bị"}</button>
              {move.selected_effects.some((effect) => effect.trigger === "ON_USE") && <button onClick={() => void useMove(move.id)} disabled={!move.is_equipped || !move.available} className="rounded-lg border border-violet-900 bg-violet-950/30 px-3 text-[10px] text-violet-200 hover:bg-violet-950/50 disabled:opacity-40">Dùng</button>}
              <button onClick={() => void remove(move.id)} className="rounded-lg border border-red-950 px-3 text-red-400 hover:bg-red-950/30" aria-label="Xóa sát chiêu"><Trash2 className="h-3.5 w-3.5" /></button>
            </div>
          </article>)}
        </div>}
      </section>
        </>
      )}
    </div>
  );
}
