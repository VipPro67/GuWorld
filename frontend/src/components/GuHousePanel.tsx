"use client";

import { useEffect, useState, useMemo } from "react";
import { Castle, Shield, Wrench, Sparkles, Plus, Check } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import type { GuHouseCombatProfile, KillerMoveCompositionGraph, KillerMoveEffectOption } from "@/store/slices/killerMovesSlice";


function isGuHouseActionSupported(effect: KillerMoveEffectOption) {
  const subtype = (effect.operation.sub_type || "FLAT").toUpperCase();
  if (subtype !== "FLAT" || (effect.conditions?.length || 0) > 0) return false;
  if (effect.operation.op === "DAMAGE") return effect.target.selector === "TARGET";
  if (["HEAL", "SHIELD", "BARRIER"].includes(effect.operation.op)) return effect.target.selector === "SELF";
  return false;
}

export function GuHousePanel() {
  const guHouses = useGameStore((s) => s.guHouses);
  const guHousesLoading = useGameStore((s) => s.guHousesLoading);
  const fetchGuHouses = useGameStore((s) => s.fetchGuHouses);
  const previewGuHouse = useGameStore((s) => s.previewGuHouse);
  const createGuHouse = useGameStore((s) => s.createGuHouse);
  const toggleEquip = useGameStore((s) => s.toggleEquipGuHouse);
  const repairHouse = useGameStore((s) => s.repairGuHouse);
  const useGuHouse = useGameStore((s) => s.useGuHouse);
  const fetchKillerMoves = useGameStore((s) => s.fetchKillerMoves);
  const gus = useGameStore((s) => s.killerMoveGus);
  const minCoreGus = useGameStore((s) => s.guHousesMinCoreGus);
  const maxCoreGus = useGameStore((s) => s.guHousesMaxCoreGus);
  const ruleSet = useGameStore((s) => s.guHouseCompositionRuleSet);
  const damageRule = ruleSet?.rules.find((rule) => rule.id === "combine-flat-single-target-damage");
  const effects = useGameStore((s) => s.killerMoveEffects);
  const guDictionary = useGameStore((s) => s.guDictionary);

  const [name, setName] = useState("");
  const [selectedCoreIDs, setSelectedCoreIDs] = useState<string[]>([]);
  const [auxGuCount, setAuxGuCount] = useState<number>(5000);
  const [actionEffectKeys, setActionEffectKeys] = useState<Record<string, string>>({});
  const [actionSupportKeys, setActionSupportKeys] = useState<Record<string, Record<string, string>>>({});
  const [preview, setPreview] = useState<GuHouseCombatProfile | null>(null);
  const [creating, setCreating] = useState(false);
  const [worldEffectKeys, setWorldEffectKeys] = useState<Record<string, string>>({});

  useEffect(() => {
    void fetchGuHouses();
    void fetchKillerMoves();
  }, [fetchGuHouses, fetchKillerMoves]);

  // Only Tiên Cổ (Rank >= 6) can be core hạch tâm
  const immortalGus = useMemo(() => {
    return gus.filter((g) => g.rank >= 6 && g.state === "ACTIVE" && g.location === "INVENTORY");
  }, [gus]);

  // Update preview
  useEffect(() => {
    if (selectedCoreIDs.length < minCoreGus) {
      setPreview(null);
      return;
    }
    let active = true;
    const timer = setTimeout(() => {
      void previewGuHouse({
        name: name || "Tiên Cổ Ốc",
        core_gu_instance_ids: selectedCoreIDs,
        aux_gu_count: auxGuCount,
        action_effect_keys: actionEffectKeys,
        action_composition_graphs: actionCompositionGraphs(),
      }).then((res) => {
        if (active) setPreview(res);
      });
    }, 200);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [selectedCoreIDs, auxGuCount, actionEffectKeys, actionSupportKeys, name, minCoreGus, ruleSet, previewGuHouse]);

  const effectsForCore = (id: string) => effects.filter((effect) => effect.source_instance_id === id && effect.combat_supported && isGuHouseActionSupported(effect));
  const worldEffectsForCore = (id: string) => effects.filter((effect) => effect.source_instance_id === id && effect.travel_supported);

  const actionCompositionGraphs = () => {
    const graphs: Record<string, KillerMoveCompositionGraph> = {};
    for (const coreID of selectedCoreIDs) {
      const output = effectsForCore(coreID).find((effect) => effect.key === actionEffectKeys[coreID]);
      const core = gus.find((gu) => gu.instance_id === coreID);
      if (!output || !core) continue;
      const nodes: KillerMoveCompositionGraph["nodes"] = [{ effect_key: output.key, source_block_id: output.source_block_id, operation_index: output.operation_index, role: "OUTPUT", gu_instance_id: coreID, gu_id: core.gu_id, rank: core.rank, trigger: output.trigger, context: "TURN_BASED_COMBAT", executor: "GU_HOUSE_ACTION_ADAPTER" }];
      const edges: KillerMoveCompositionGraph["edges"] = [];
      for (const [supportID, effectKey] of Object.entries(actionSupportKeys[coreID] || {})) {
        const support = effectsForCore(supportID).find((effect) => effect.key === effectKey && effect.composition_capabilities?.includes(damageRule?.support_capability || ""));
        const supportGu = gus.find((gu) => gu.instance_id === supportID);
        if (!support || !supportGu || !damageRule || !output.composition_capabilities?.includes(damageRule.core_capability)) continue;
        nodes.push({ effect_key: support.key, source_block_id: support.source_block_id, operation_index: support.operation_index, role: "SUPPORT", gu_instance_id: supportID, gu_id: supportGu.gu_id, rank: supportGu.rank, trigger: support.trigger, context: "TURN_BASED_COMBAT", executor: "GU_HOUSE_ACTION_ADAPTER" });
        edges.push({ from: support.key, to: output.key, kind: damageRule.kind, rule_id: damageRule.id });
      }
      graphs[coreID] = { schema_version: 1, product: "GU_HOUSE_ACTION", product_profile: "gu_house_action.v1", context: "TURN_BASED_COMBAT", executor: "GU_HOUSE_ACTION_ADAPTER", rule_set_revision: ruleSet?.revision, core_gu_instance_id: coreID, output_effect_key: output.key, nodes, edges };
    }
    return graphs;
  };

  const toggleCoreGu = (id: string) => {
    if (selectedCoreIDs.includes(id)) {
      setSelectedCoreIDs((prev) => prev.filter((item) => item !== id));
      setActionEffectKeys((prev) => { const next = { ...prev }; delete next[id]; return next; });
      setActionSupportKeys((prev) => { const next = { ...prev }; delete next[id]; for (const key of Object.keys(next)) { next[key] = { ...next[key] }; delete next[key][id]; } return next; });
    } else {
      if (selectedCoreIDs.length >= maxCoreGus) return;
      setSelectedCoreIDs((prev) => [...prev, id]);
      const options = effectsForCore(id);
      const chosen = options.find((effect) => effect.operation.op === "DAMAGE") || options[0];
      if (chosen) setActionEffectKeys((prev) => ({ ...prev, [id]: chosen.key }));
    }
  };

  const handleCreate = async () => {
    if (!name.trim() || selectedCoreIDs.length < minCoreGus) return;
    setCreating(true);
    const success = await createGuHouse({
      name: name.trim(),
      core_gu_instance_ids: selectedCoreIDs,
      aux_gu_count: auxGuCount,
      action_effect_keys: actionEffectKeys,
      action_composition_graphs: actionCompositionGraphs(),
      preview_fingerprint: preview?.preview_fingerprint,
    });
    setCreating(false);
    if (success) {
      setName("");
      setSelectedCoreIDs([]);
      setActionEffectKeys({});
      setActionSupportKeys({});
      setAuxGuCount(5000);
    }
  };

  return (
    <div className="space-y-6">
      {/* Intro Header */}
      <div className="rounded-2xl border border-amber-900/40 bg-zinc-950/80 p-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-amber-500/30 bg-amber-950/40 text-amber-400">
            <Castle className="h-5 w-5" />
          </div>

          <div>
            <h3 className="text-base font-bold text-amber-200">Tiên Cổ Ốc (Immortal Gu House)</h3>
            <p className="text-xs text-zinc-400">
              Sát chiêu Tiên đạo thực thể hóa thành chiến hạm / pháo đài. Hấp thụ toàn bộ sát thương và miễn nhiễm phản phệ cho Cổ Tiên.
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Crafting Section */}
        <section className="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-950/60 p-5">
          <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
            <h4 className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Sparkles className="h-4 w-4 text-amber-400" />
              Đúc Kết Tiên Cổ Ốc
            </h4>
            <span className="font-mono text-xs text-zinc-500">
              {selectedCoreIDs.length}/{maxCoreGus} Tiên Cổ
            </span>
          </div>

          <div>
            <label className="text-xs text-zinc-400">Tên Cổ Ốc</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="VD: Vấn Đạo Lâu, Vạn Niên Đấu Phi Xa..."
              className="mt-1 w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3.5 py-2 text-xs text-zinc-100 outline-none focus:border-amber-500"
            />
          </div>

          <div>
            <label className="text-xs text-zinc-400">
              Chọn Tiên Cổ Hạch Tâm (Cần {minCoreGus} đến {maxCoreGus} Tiên Cổ Rank 6+)
            </label>
            {immortalGus.length === 0 ? (
              <div className="mt-2 rounded-xl border border-dashed border-zinc-800 p-4 text-center text-xs text-zinc-600">
                Bạn chưa sở hữu Tiên Cổ (Rank 6 trở lên) để làm trụ cột Cổ Ốc.
              </div>
            ) : (
              <div className="mt-2 max-h-44 space-y-1.5 overflow-y-auto">
                {immortalGus.map((gu) => {
                  const spec = guDictionary[gu.gu_id];
                  const selected = selectedCoreIDs.includes(gu.instance_id);
                  return (
                    <button
                      key={gu.instance_id}
                      type="button"
                      onClick={() => toggleCoreGu(gu.instance_id)}
                      className={`flex w-full items-center justify-between rounded-xl border px-3 py-2 text-xs transition-colors ${
                        selected
                          ? "border-amber-600 bg-amber-950/40 text-amber-200"
                          : "border-zinc-800 bg-zinc-900/60 text-zinc-300 hover:border-zinc-700"
                      }`}
                    >
                      <span className="font-semibold">{spec?.name || gu.gu_id} · Rank {gu.rank}</span>
                      {selected ? <Check className="h-4 w-4 text-amber-400" /> : <Plus className="h-4 w-4 text-zinc-600" />}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          {selectedCoreIDs.length > 0 && <div>
            <label className="text-xs text-zinc-400">Chọn Trận Kích cho từng hạch tâm</label>
            <div className="mt-2 space-y-2">
              {selectedCoreIDs.map((id) => {
                const gu = gus.find((entry) => entry.instance_id === id);
                const choices = effectsForCore(id);
                return <div key={id} className="flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/60 p-2">
                  <span className="min-w-0 flex-1 truncate text-[11px] text-zinc-300">{gu ? (guDictionary[gu.gu_id]?.name || gu.gu_id) : id}</span>
                  {choices.length === 0 ? <span className="text-[10px] text-amber-300">Chưa có effect combat theo lượt được hỗ trợ</span> : <select value={actionEffectKeys[id] || ""} onChange={(event) => setActionEffectKeys((prev) => ({ ...prev, [id]: event.target.value }))} className="max-w-[60%] rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1.5 text-[10px] text-zinc-200">
                    <option value="">Không gán Trận Kích</option>
                    {choices.map((effect) => <option key={effect.key} value={effect.key}>{effect.label} · {effect.operation.value ? effect.operation.value : effect.operation.op}</option>)}
                  </select>}
                  {actionEffectKeys[id] && selectedCoreIDs.filter((supportID) => supportID !== id && effectsForCore(supportID).length > 0).map((supportID) => {
                    const supportGu = gus.find((entry) => entry.instance_id === supportID);
                    const supportChoices = effectsForCore(supportID);
                    return <div key={supportID} className="flex items-center gap-1">
                      <select aria-label="Cổ hỗ trợ Trận Kích" value={actionSupportKeys[id]?.[supportID] || ""} onChange={(event) => setActionSupportKeys((prev) => ({ ...prev, [id]: { ...prev[id], [supportID]: event.target.value } }))} className="max-w-[45%] rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1.5 text-[10px] text-zinc-200">
                        <option value="">Không ghép · {supportGu ? (guDictionary[supportGu.gu_id]?.name || supportGu.gu_id) : "Cổ hỗ trợ"}</option>
                        {supportChoices.filter((effect) => damageRule && effect.composition_capabilities?.includes(damageRule.support_capability) && choices.find((candidate) => candidate.key === actionEffectKeys[id])?.composition_capabilities?.includes(damageRule.core_capability)).map((effect) => <option key={effect.key} value={effect.key}>{effect.label} · {damageRule?.kind}</option>)}
                      </select>
                    </div>;
                  })}
                </div>;
              })}
            </div>
          </div>}

          <div>
            <div className="flex items-center justify-between">
              <label className="text-xs text-zinc-400">Số lượng Phàm Cổ làm vỏ ốc</label>
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  min={500}
                  max={100000}
                  step={500}
                  value={auxGuCount}
                  onChange={(e) => {
                    const val = Math.max(500, Math.min(100000, Number(e.target.value) || 500));
                    setAuxGuCount(val);
                  }}
                  className="w-24 rounded border border-zinc-700 bg-zinc-900 px-2 py-0.5 text-right font-mono text-xs text-amber-300 outline-none focus:border-amber-500"
                />
                <span className="text-xs text-zinc-500">Cổ</span>
              </div>
            </div>
            <input
              type="range"
              min={500}
              max={100000}
              step={500}
              value={auxGuCount}
              onChange={(e) => setAuxGuCount(Number(e.target.value))}
              className="mt-2 w-full accent-amber-500"
            />
            <p className="mt-1 text-[10px] text-zinc-500">
              Phàm Cổ cấu thành lớp giáp ngoài (từ 500 đến 100.000 con), trực tiếp gia tăng Máu (HP) và Phòng Thủ cho Cổ Ốc.
            </p>
          </div>

          {preview && (
            <div className="rounded-xl border border-amber-900/40 bg-amber-950/20 p-3.5 space-y-2 text-xs">
              <div className="font-bold text-amber-300">Dự Tính Chỉ Số Cổ Ốc:</div>
              <div className="grid grid-cols-3 gap-2 text-[11px] font-mono">
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Độ Bền / Máu</div>
                  <div className="text-emerald-400 font-bold mt-0.5">{preview.max_house_hp} HP</div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Giáp Phòng Ngự</div>
                  <div className="text-cyan-400 font-bold mt-0.5">{preview.house_defense} Giáp</div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Kháng Sát Thương</div>
                  <div className="text-amber-400 font-bold mt-0.5">{Math.round(preview.damage_reduction * 100)}%</div>
                </div>
              </div>
              <div className="text-[11px] text-zinc-400">
                Tích hợp {preview.actions?.length || 0} Trận Kích (Sát chiêu đặc hữu của Tiên Cổ Ốc).
              </div>
              {preview.actions?.map((action) => <div key={action.id} className="flex justify-between rounded-lg bg-zinc-900/70 px-2 py-1.5 text-[10px]">
                <span className="text-zinc-300">{action.name}</span>
                <span className="text-amber-200">{action.effect_label || (action.target_type === "SINGLE" ? `Đơn mục tiêu · ${action.base_damage || 0}` : `Hỗ trợ · khiên ${action.base_shield || 0} · hồi ${action.base_healing || 0}`)}</span>
              </div>)}
            </div>
          )}

          <button
            onClick={() => void handleCreate()}
            disabled={creating || selectedCoreIDs.length < minCoreGus || !name.trim() || !preview?.preview_fingerprint || !preview.actions?.length}
            className="w-full rounded-xl bg-amber-600 py-2.5 text-xs font-bold text-white shadow-lg shadow-amber-950/40 hover:bg-amber-500 disabled:opacity-40"
          >
            {creating ? "Đang đúc kết…" : "Đúc Kết Tiên Cổ Ốc"}
          </button>
        </section>

        {/* Owned List */}
        <section className="space-y-4">
          <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-3 text-sm font-bold text-zinc-100">
            <Castle className="h-4 w-4 text-cyan-400" />
            Tiên Cổ Ốc Sở Hữu
          </div>

          {guHousesLoading ? (
            <div className="py-8 text-center text-xs text-zinc-500">Đang tải Tiên Cổ Ốc…</div>
          ) : guHouses.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-zinc-800 p-8 text-center text-xs text-zinc-500">
              Chưa có Tiên Cổ Ốc nào. Hãy đúc kết một tòa pháo đài để mang vào chiến trận!
            </div>
          ) : (
            <div className="space-y-3">
              {guHouses.map((house) => (
                <article
                  key={house.id}
                  className="rounded-2xl border border-zinc-800 bg-zinc-950/70 p-4 space-y-3"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h5 className="text-sm font-bold text-amber-200">{house.name}</h5>
                      <p className="text-[11px] text-zinc-400">
                        Rank {house.rank} · {house.core_gu_instances.length} Tiên Cổ · {house.aux_gu_count} Phàm Cổ
                      </p>
                    </div>
                    {house.is_equipped && (
                      <span className="rounded-full border border-emerald-900 bg-emerald-950/50 px-2.5 py-0.5 text-[10px] font-semibold text-emerald-300">
                        Đang xuất chiến
                      </span>
                    )}
                  </div>

                  {/* Durability bar */}
                  <div>
                    <div className="flex justify-between text-[10px] font-mono text-zinc-400">
                      <span>Độ Bền Pháo Đài</span>
                      <span className={house.current_durability < 40 ? "text-red-400" : "text-emerald-400"}>
                        {Math.round(house.current_durability)}%
                      </span>
                    </div>
                    <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-zinc-800">
                      <div
                        className={`h-full transition-all ${
                          house.current_durability < 40 ? "bg-red-500" : "bg-emerald-500"
                        }`}
                        style={{ width: `${house.current_durability}%` }}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 text-[10px] font-mono text-zinc-400">
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Máu: <span className="text-emerald-300">{house.house_profile.max_house_hp}</span>
                    </div>
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Giáp: <span className="text-cyan-300">{house.house_profile.house_defense}</span>
                    </div>
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Giảm ST: <span className="text-amber-300">{Math.round(house.house_profile.damage_reduction * 100)}%</span>
                    </div>
                  </div>

                  {house.is_equipped && house.core_gu_instances.map((coreID) => {
                    const options = worldEffectsForCore(coreID);
                    if (options.length === 0) return null;
                    return <div key={coreID} className="flex gap-2">
                      <select aria-label={`Effect ngoài combat ${coreID}`} value={worldEffectKeys[coreID] || ""} onChange={(event) => setWorldEffectKeys((prev) => ({ ...prev, [coreID]: event.target.value }))} className="min-w-0 flex-1 rounded-lg border border-zinc-800 bg-zinc-900 px-2 py-2 text-[10px] text-zinc-200">
                        <option value="">Chọn effect ON_USE của Cổ hạch tâm</option>
                        {options.map((effect) => <option key={effect.key} value={effect.key}>{effect.label}</option>)}
                      </select>
                      <button disabled={!worldEffectKeys[coreID]} onClick={() => void useGuHouse(house.id, coreID, worldEffectKeys[coreID])} className="rounded-lg border border-amber-800/60 bg-amber-950/30 px-3 py-2 text-[10px] font-semibold text-amber-200 disabled:opacity-40">Thi triển</button>
                    </div>;
                  })}

                  <div className="flex gap-2 pt-1">
                    <button
                      onClick={() => void toggleEquip(house.id)}
                      className={`flex-1 rounded-xl border py-2 text-xs font-bold transition-colors ${
                        house.is_equipped
                          ? "border-zinc-700 bg-zinc-900 text-zinc-300"
                          : "border-emerald-900 bg-emerald-950/40 text-emerald-300 hover:bg-emerald-950/70"
                      }`}
                    >
                      {house.is_equipped ? "Thu Hồi" : "Xuất Chiến"}
                    </button>

                    <button
                      onClick={() => void repairHouse(house.id)}
                      className="flex items-center gap-1.5 rounded-xl border border-sky-900/60 bg-sky-950/30 px-3 py-2 text-xs font-semibold text-sky-200 hover:bg-sky-950/60"
                      title="Tu sửa độ bền và tự động đồng bộ Tiên Cổ hạch tâm đã nâng cấp / thay thế"
                    >
                      <Wrench className="h-3.5 w-3.5" />
                      {house.current_durability < 100 ? "Tu Sửa" : "Đồng Bộ Cổ"}
                    </button>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
