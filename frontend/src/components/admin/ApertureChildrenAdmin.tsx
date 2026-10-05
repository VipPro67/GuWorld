"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Edit3, Plus, RefreshCw, Save, Trash2, X } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type RecordRow = Record<string, unknown>;

export const ApertureChildrenAdmin: React.FC<{
  apertureId: string;
  ownerUsername?: string;
  canEdit: boolean;
  canDelete: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ apertureId, ownerUsername, canEdit, canDelete, addToast, exec, prepareDelete }) => {
  const [nodes, setNodes] = useState<RecordRow[]>([]);
  const [people, setPeople] = useState<RecordRow[]>([]);
  const [editing, setEditing] = useState<{ kind: "node" | "person"; row: RecordRow | null; json: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [workers, setWorkers] = useState<Record<string, RecordRow[]>>({});
  const [workerChoice, setWorkerChoice] = useState("");
  const [workerAmount, setWorkerAmount] = useState("0");

  const load = useCallback(async () => {
    setBusy(true);
    try {
      const [nodesRes, peopleRes] = await Promise.all([
        authFetch(`${BACKEND_URL}/api/admin/apertures/${apertureId}/nodes`),
        authFetch(`${BACKEND_URL}/api/admin/apertures/${apertureId}/variant-humans`),
      ]);
      if (!nodesRes.ok || !peopleRes.ok) throw new Error("Không thể tải dữ liệu phụ của tiên khiếu");
      setNodes(await nodesRes.json());
      setPeople(await peopleRes.json());
    } catch (error) { addToast(error instanceof Error ? error.message : "Lỗi tải dữ liệu", "error"); }
    finally { setBusy(false); }
  }, [apertureId, addToast]);

  useEffect(() => { void load(); }, [load]);

  const save = async () => {
    if (!editing) return;
    let payload: RecordRow;
    try { payload = JSON.parse(editing.json) as RecordRow; }
    catch { addToast("JSON không hợp lệ", "error"); return; }
    const kindPath = editing.kind === "node" ? "nodes" : "variant-humans";
    const id = String(editing.row?.id || "");
    const url = `${BACKEND_URL}/api/admin/apertures/${apertureId}/${kindPath}${id ? `/${id}` : ""}`;
    const ok = await exec(url, id ? "PUT" : "POST", payload as ApiResponseData);
    if (ok) { setEditing(null); void load(); }
  };

  const loadWorkers = async (nodeId: string) => {
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/apertures/${apertureId}/nodes/${nodeId}/workers`);
      if (!res.ok) throw new Error("Không thể tải danh sách lao động");
      const data = await res.json();
      setWorkers(current => ({ ...current, [nodeId]: data }));
    } catch (error) { addToast(error instanceof Error ? error.message : "Lỗi tải lao động", "error"); }
  };

  const saveWorker = async (nodeId: string, variantId: string, assigned: number) => {
    if (!variantId || !Number.isInteger(assigned) || assigned < 0) { addToast("Nhóm dân cư hoặc số lao động không hợp lệ", "error"); return; }
    const ok = await exec(`${BACKEND_URL}/api/admin/apertures/${apertureId}/nodes/${nodeId}/workers`, "POST", { variant_human_id: variantId, assigned_population: assigned });
    if (ok) await loadWorkers(nodeId);
  };

  const remove = async (kind: "node" | "person", row: RecordRow) => {
    const collection = kind === "node" ? "nodes" : "variant-humans";
    const label = String(row.name || row.race || row.id);
    const url = `${BACKEND_URL}/api/admin/apertures/${apertureId}/${collection}/${row.id}`;
    const prepared = prepareDelete(`xóa ${kind === "node" ? "điểm tài nguyên" : "nhóm dân cư"}`, `${label} · ${row.id}`);
    if (!prepared) return;
    if (await exec(url, "DELETE", prepared)) void load();
  };

  const editor = (kind: "node" | "person", row: RecordRow | null) => setEditing({
    kind,
    row,
    json: JSON.stringify(row || (kind === "node" ? {
      node_type: "RESOURCE", name: "", path: "", level: 1, hp: 1000, max_hp: 1000, x: 0, y: 0,
      production_type: "", production_amount: 0, production_interval_hours: 24, is_destroyed: false, node_category: "RESOURCE",
      defense_power: 0, attack_range: 0, hp_bonus: 0, damage_reduction: 0, blocks_movement: false, required_gu_id: "",
      required_gu_rank: 0, zone_type: "", food_path: "", item_tier: "", production_key: "",
    } : {
      race: "", population: 0, happiness: 70, oppression: 0, rebellion_rate: 0, labor_intensity: "NORMAL", prosperity: 50,
      gu_master_count: 0, gu_immortal_count: 0, food_stored: 0, food_per_capita: 1, population_growth_rate: 0.001, mortality_rate: 0.0005,
    }), null, 2),
  });

  const list = (kind: "node" | "person", rows: RecordRow[]) => (
    <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
      <div className="flex items-center justify-between"><h4 className="text-xs font-bold text-zinc-200">{kind === "node" ? `Node tài nguyên · ${ownerUsername || "tiên khiếu đang chọn"} (${rows.length})` : `Dân cư dị nhân (${rows.length})`}</h4>{canEdit && <button onClick={() => editor(kind, null)} className="px-2 py-1 rounded bg-amber-600/20 text-amber-400 text-[10px]"><Plus className="inline w-3 h-3 mr-1"/>Thêm</button>}</div>
      {rows.length === 0 ? <p className="text-[10px] text-zinc-500">Chưa có dữ liệu.</p> : rows.map(row => <div key={String(row.id)} className="border-t border-zinc-800/60 pt-2 space-y-2"><div className="flex items-center justify-between gap-2"><div className="min-w-0"><div className="text-xs text-zinc-300 truncate">{String(row.name || row.race || row.id)}</div><div className="text-[10px] text-zinc-600 font-mono truncate">{kind === "node" ? `${row.node_type} · ${row.node_category} · (${row.x}, ${row.y})` : `${row.population} dân · ${row.labor_intensity} · ${row.gu_master_count} đại sư / ${row.gu_immortal_count} tiên`}</div></div><div className="flex gap-1">{kind === "node" && <button onClick={() => { const nodeId=String(row.id);const next=expandedNode===nodeId?null:nodeId;setExpandedNode(next);if(next)void loadWorkers(nodeId); }} className="px-2 py-1 text-[10px] text-sky-400">Lao động</button>}{canEdit && <button title="Sửa toàn bộ field" onClick={() => editor(kind, row)} className="p-1 text-zinc-500 hover:text-amber-400"><Edit3 className="w-3.5 h-3.5"/></button>}{canDelete && <button title="Xóa" onClick={() => void remove(kind, row)} className="p-1 text-zinc-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5"/></button>}</div></div>{kind==="node"&&expandedNode===String(row.id)&&<div className="rounded-lg bg-zinc-950 p-2 space-y-2">{(workers[String(row.id)]||[]).map(worker=><div key={String(worker.id)} className="flex items-center gap-2 text-[10px]"><span className="min-w-0 flex-1 truncate text-zinc-400">{String(worker.race)} · dân số {String(worker.population)}</span><input type="number" min="0" max={Number(worker.population)} defaultValue={Number(worker.assigned_population)} className="w-20 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-zinc-200" onChange={e=>{worker.assigned_population=Number(e.target.value)}}/><button onClick={()=>void saveWorker(String(row.id),String(worker.variant_human_id),Number(worker.assigned_population))} className="text-amber-400">Lưu</button>{canDelete&&<button onClick={async()=>{const endpoint=`${BACKEND_URL}/api/admin/apertures/${apertureId}/nodes/${row.id}/workers/${worker.id}`;const prepared=prepareDelete("xóa phân công lao động",`${worker.race} · ${worker.id}`);if(prepared&&await exec(endpoint,"DELETE",prepared))void loadWorkers(String(row.id));}} className="text-red-400">Xóa</button>}</div>)}{canEdit&&people.length>0&&<div className="flex gap-2"><select value={workerChoice} onChange={e=>setWorkerChoice(e.target.value)} className="min-w-0 flex-1 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-[10px] text-zinc-300"><option value="">Chọn nhóm dân cư</option>{people.map(p=><option key={String(p.id)} value={String(p.id)}>{String(p.race)} · dân số {String(p.population)}</option>)}</select><input value={workerAmount} onChange={e=>setWorkerAmount(e.target.value)} type="number" min="0" className="w-20 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-[10px] text-zinc-200"/><button onClick={()=>void saveWorker(String(row.id),workerChoice,Number(workerAmount))} className="text-amber-400 text-[10px]">Phân công</button></div>}</div>}</div>)}
    </section>
  );

  return <div className="space-y-3"><div className="flex justify-end"><button onClick={() => void load()} disabled={busy} className="text-[10px] text-zinc-500 hover:text-zinc-200"><RefreshCw className={`inline w-3 h-3 mr-1 ${busy ? "animate-spin" : ""}`}/>Làm mới</button></div><div className="grid lg:grid-cols-2 gap-3">{list("node", nodes)}{list("person", people)}</div>{editing && <div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4" onClick={() => setEditing(null)}><div onClick={e => e.stopPropagation()} className="w-full max-w-3xl bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"><div className="flex justify-between items-center"><h3 className="text-sm font-bold text-zinc-200">{editing.row ? "Sửa toàn bộ dữ liệu" : "Thêm bản ghi"} · {editing.kind === "node" ? "điểm tài nguyên" : "dân cư dị nhân"}</h3><button onClick={() => setEditing(null)}><X className="w-4 h-4 text-zinc-500"/></button></div><p className="text-[10px] text-zinc-500">JSON chứa toàn bộ field của bảng. Không sửa id/aperture_id khi cập nhật.</p><textarea value={editing.json} onChange={e => setEditing({ ...editing, json: e.target.value })} rows={20} spellCheck={false} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-[11px] font-mono text-zinc-200"/><div className="flex justify-end"><button onClick={() => void save()} className="px-3 py-2 rounded bg-amber-600 text-xs text-white"><Save className="inline w-3.5 h-3.5 mr-1"/>Lưu</button></div></div></div>}</div>;
};
