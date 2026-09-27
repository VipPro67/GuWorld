"use client";

import React, { useEffect, useMemo, useState } from "react";
import { AlertCircle, FileCode, Layers, Plus, Trash2 } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { EMPTY_EFFECT_ENVELOPE, type EffectBlock, type EffectCatalogMetadata, type EffectEnvelope, type EffectOperation } from "@/lib/effect-blocks";
import { EffectEnvelopeView } from "@/components/EffectBlockView";

interface ValidationIssue { path: string; error_code: string; message: string; block_id?: string; operation_index?: number }
interface Props {
  effects: ApiResponseData;
  onChange: (effects: EffectEnvelope) => void;
  source?: "gu" | "item" | "race" | "constitution" | "status";
  effectKey?: string;
}

const fallbackMetadata: EffectCatalogMetadata = {
  schema_version: 2, triggers: [], targets: [], operations: [], stats: [], statuses: [], resources: [], units: [],
  stack_policies: [], condition_fields: [], condition_operators: [], native_codes: [],
	  native_hooks: [],
};
const asEnvelope = (effects: ApiResponseData): EffectEnvelope => {
  const value = effects as Partial<EffectEnvelope>;
  return value.schema_version === 2 && Array.isArray(value.blocks) ? value as EffectEnvelope : EMPTY_EFFECT_ENVELOPE;
};
const canonicalId = (index: number) => `BLOCK_${index + 1}_${Date.now().toString(36).toUpperCase()}`;

export function GuEffectsEditor({ effects, onChange, source = "gu", effectKey = "preview" }: Props) {
  const envelope = asEnvelope(effects);
  const [metadata, setMetadata] = useState(fallbackMetadata);
  const [mode, setMode] = useState<"visual" | "json">("visual");
  const [jsonText, setJsonText] = useState(() => JSON.stringify(envelope, null, 2));
  const [issues, setIssues] = useState<ValidationIssue[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    authFetch(`${BACKEND_URL}/api/admin/effect-catalog/metadata`).then(async (response) => {
      if (response.ok) setMetadata(await response.json());
    }).catch(() => setMetadata(fallbackMetadata));
  }, []);

  const operationMap = useMemo(() => new Map(metadata.operations.map((op) => [op.name, op])), [metadata]);
  const replaceBlock = (index: number, block: EffectBlock) => onChange({ schema_version: 2, blocks: envelope.blocks.map((old, i) => i === index ? block : old) });
  const addBlock = () => onChange({ schema_version: 2, blocks: [...envelope.blocks, {
    id: canonicalId(envelope.blocks.length), trigger: metadata.triggers[0] || "ON_ACTION",
    target: { selector: metadata.targets[0] || "TARGET" }, operations: [],
  }] });
  const addOperation = (blockIndex: number) => {
    const block = envelope.blocks[blockIndex];
    const name = metadata.operations[0]?.name || "DAMAGE";
    replaceBlock(blockIndex, { ...block, operations: [...block.operations, { op: name }] });
  };
  const patchOperation = (blockIndex: number, operationIndex: number, patch: Partial<EffectOperation>) => {
    const block = envelope.blocks[blockIndex];
    replaceBlock(blockIndex, { ...block, operations: block.operations.map((op, i) => i === operationIndex ? { ...op, ...patch } : op) });
  };
  const validateJson = async () => {
    setLoading(true);
    try {
      const parsed = JSON.parse(jsonText) as EffectEnvelope;
      const response = await authFetch(`${BACKEND_URL}/api/admin/effect-catalog/validate`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ source, key: effectKey, effects: parsed }),
      });
      const result = await response.json();
      const nextIssues: ValidationIssue[] = result.issues || [{ path: "$", error_code: "request_failed", message: result.error || "Validation thất bại" }];
      setIssues(nextIssues);
      if (response.ok && result.valid && parsed.schema_version === 2 && Object.keys(parsed).every((key) => key === "schema_version" || key === "blocks")) {
        onChange(parsed); setMode("visual");
      }
    } catch (error) {
      setIssues([{ path: "$", error_code: "invalid_json", message: error instanceof Error ? error.message : "JSON không hợp lệ" }]);
    } finally { setLoading(false); }
  };

  return <div className="mt-4 border-t border-zinc-800 pt-4 space-y-3">
    <div className="flex items-center justify-between">
      <h4 className="flex items-center gap-2 text-xs font-bold text-zinc-300 uppercase"><Layers className="w-4 h-4 text-amber-500" />Effect Block v2 · {envelope.blocks.length} blocks</h4>
      <button type="button" onClick={() => { setJsonText(JSON.stringify(envelope, null, 2)); setIssues([]); setMode(mode === "json" ? "visual" : "json"); }} className="text-xs px-2 py-1 rounded bg-zinc-800 text-amber-300"><FileCode className="inline w-3 h-3 mr-1" />{mode === "json" ? "Trực quan" : "JSON"}</button>
    </div>
    {mode === "json" ? <>
      <textarea className="w-full min-h-72 rounded bg-zinc-950 border border-zinc-700 p-3 font-mono text-xs" value={jsonText} onChange={(event) => setJsonText(event.target.value)} />
      {issues.map((issue, index) => <div key={`${issue.path}-${index}`} className="text-xs text-red-400 flex gap-2"><AlertCircle className="w-3 h-3" /><code>{issue.path}</code> {issue.message}</div>)}
      <button type="button" disabled={loading} onClick={validateJson} className="px-3 py-1.5 rounded bg-amber-700 disabled:opacity-50 text-xs">{loading ? "Đang kiểm tra…" : "Validate & áp dụng"}</button>
    </> : <>
      {metadata.triggers.length === 0 && <p className="text-[11px] text-amber-400">Không tải được catalog; chỉ có thể xem hoặc dùng JSON mode.</p>}
      {envelope.blocks.map((block, blockIndex) => <div key={block.id} className="rounded border border-zinc-800 p-3 space-y-2">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
          <input value={block.id} onChange={(e) => replaceBlock(blockIndex, { ...block, id: e.target.value.toUpperCase() })} className="bg-zinc-950 border border-zinc-700 rounded p-1.5 text-xs font-mono" />
          <select value={block.trigger} onChange={(e) => replaceBlock(blockIndex, { ...block, trigger: e.target.value })} className="bg-zinc-950 border border-zinc-700 rounded p-1.5 text-xs">{metadata.triggers.map((v) => <option key={v}>{v}</option>)}</select>
          <select value={block.target.selector} onChange={(e) => replaceBlock(blockIndex, { ...block, target: { ...block.target, selector: e.target.value } })} className="bg-zinc-950 border border-zinc-700 rounded p-1.5 text-xs">{metadata.targets.map((v) => <option key={v}>{v}</option>)}</select>
        </div>
        {block.operations.map((operation, operationIndex) => { const definition = operationMap.get(operation.op); return <div key={operationIndex} className="flex flex-wrap gap-2 items-center bg-zinc-950/60 p-2 rounded">
          <select value={operation.op} onChange={(e) => patchOperation(blockIndex, operationIndex, { op: e.target.value })} className="bg-zinc-900 border border-zinc-700 rounded p-1 text-xs">{metadata.operations.map((v) => <option key={v.name}>{v.name}</option>)}</select>
          {definition?.sub_types?.length ? <select value={operation.sub_type || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { sub_type: e.target.value })} className="bg-zinc-900 border border-zinc-700 rounded p-1 text-xs"><option value="">sub_type</option>{definition.sub_types.map((v) => <option key={v}>{v}</option>)}</select> : null}
          {definition?.allowed_fields.includes("stat") && <select value={operation.stat || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { stat: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">stat</option>{metadata.stats.map((v) => <option key={v}>{v}</option>)}</select>}
          {definition?.allowed_fields.includes("status_id") && <select value={operation.status_id || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { status_id: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">status</option>{metadata.statuses.map((v) => <option key={v}>{v}</option>)}</select>}
          {definition?.allowed_fields.includes("code") && <select value={operation.code || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { code: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">native code</option>{metadata.native_codes.map((v) => <option key={v}>{v}</option>)}</select>}
          {definition?.allowed_fields.includes("value") && <input type="number" value={operation.value ?? ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { value: Number(e.target.value) })} placeholder="value" className="w-24 bg-zinc-900 rounded p-1 text-xs" />}
          {definition?.allowed_fields.includes("unit") && <select value={operation.unit || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { unit: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">unit</option>{metadata.units.map((v) => <option key={v}>{v}</option>)}</select>}
          {definition?.allowed_fields.includes("resource") && <select value={operation.resource || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { resource: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">resource</option>{metadata.resources.map((v) => <option key={v}>{v}</option>)}</select>}
          {definition?.allowed_fields.includes("chance") && <input type="number" min="0" max="1" step="0.01" value={operation.chance ?? ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { chance: Number(e.target.value) })} placeholder="chance" className="w-20 bg-zinc-900 rounded p-1 text-xs" />}
          {definition?.allowed_fields.includes("turns") && <input type="number" min="1" value={operation.turns ?? ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { turns: Number(e.target.value) })} placeholder="turns" className="w-20 bg-zinc-900 rounded p-1 text-xs" />}
          {definition?.allowed_fields.includes("stack_policy") && <select value={operation.stack_policy || ""} onChange={(e) => patchOperation(blockIndex, operationIndex, { stack_policy: e.target.value })} className="bg-zinc-900 rounded p-1 text-xs"><option value="">stack</option>{metadata.stack_policies.map((v) => <option key={v}>{v}</option>)}</select>}
          <button type="button" onClick={() => replaceBlock(blockIndex, { ...block, operations: block.operations.filter((_, i) => i !== operationIndex) })}><Trash2 className="w-3 h-3 text-red-400" /></button>
        </div>; })}
        <button type="button" onClick={() => addOperation(blockIndex)} className="text-xs text-amber-400"><Plus className="inline w-3 h-3" /> operation</button>
        <button type="button" onClick={() => onChange({ schema_version: 2, blocks: envelope.blocks.filter((_, i) => i !== blockIndex) })} className="ml-3 text-xs text-red-400"><Trash2 className="inline w-3 h-3" /> block</button>
      </div>)}
      <button type="button" onClick={addBlock} className="text-xs px-3 py-1.5 rounded bg-amber-800"><Plus className="inline w-3 h-3" /> block</button>
      <EffectEnvelopeView effects={envelope} />
    </>}
  </div>;
}
