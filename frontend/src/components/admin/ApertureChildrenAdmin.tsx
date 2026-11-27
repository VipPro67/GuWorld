"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Edit3, Plus, RefreshCw, Save, Sparkles, Trash2, X } from "lucide-react";
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

  const seedTestData = async () => {
    setBusy(true);
    try {
      const ok = await exec(`${BACKEND_URL}/api/admin/apertures/${apertureId}/seed-labor-test`, "POST");
      if (ok) void load();
    } finally {
      setBusy(false);
    }
  };

  const remove = async (kind: "node" | "person", row: RecordRow) => {
    const collection = kind === "node" ? "nodes" : "variant-humans";
    const label = String(row.name || row.race || row.id);
    const url = `${BACKEND_URL}/api/admin/apertures/${apertureId}/${collection}/${row.id}`;
    const prepared = prepareDelete(`xóa ${kind === "node" ? "công trình" : "nhóm dân cư"}`, `${label} · ${row.id}`);
    if (!prepared) return;
    if (await exec(url, "DELETE", prepared)) void load();
  };

  const editor = (kind: "node" | "person", row: RecordRow | null) => setEditing({
    kind,
    row,
    json: JSON.stringify(row || (kind === "node" ? {
      node_type: "DEFENSE", name: "", path: "", level: 1, hp: 1000, max_hp: 1000, x: 0, y: 0,
      is_destroyed: false, node_category: "DEFENSE", defense_power: 100, attack_range: 2, hp_bonus: 0, damage_reduction: 0, blocks_movement: false,
    } : {
      race: "HAIRY_MAN", population: 1000, happiness: 70, oppression: 0, rebellion_rate: 0, labor_intensity: "NORMAL", prosperity: 50,
      gu_master_count: 0, gu_immortal_count: 0, food_stored: 2000, food_per_capita: 1, population_growth_rate: 0.001, mortality_rate: 0.0005,
    }), null, 2),
  });

  const list = (kind: "node" | "person", rows: RecordRow[]) => (
    <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
      <div className="flex items-center justify-between"><h4 className="text-xs font-bold text-zinc-200">{kind === "node" ? `Công trình phòng thủ · ${ownerUsername || "tiên khiếu đang chọn"} (${rows.length})` : `Dân cư dị nhân (${rows.length})`}</h4>{canEdit && <button onClick={() => editor(kind, null)} className="px-2 py-1 rounded bg-amber-600/20 text-amber-400 text-[10px]"><Plus className="inline w-3 h-3 mr-1"/>Thêm</button>}</div>
      {rows.length === 0 ? <p className="text-[10px] text-zinc-500">Chưa có dữ liệu.</p> : rows.map(row => <div key={String(row.id)} className="border-t border-zinc-800/60 pt-2 flex items-center justify-between gap-2"><div className="min-w-0"><div className="text-xs text-zinc-300 truncate">{String(row.name || row.race || row.id)}</div><div className="text-[10px] text-zinc-600 font-mono truncate">{kind === "node" ? `${row.node_type} · ${row.node_category} · (${row.x}, ${row.y})` : `${row.population} dân · ${row.labor_intensity} · ${row.gu_master_count} sư / ${row.gu_immortal_count} tiên · lương ${row.food_stored}`}</div></div><div className="flex gap-1">{canEdit && <button title="Sửa toàn bộ field" onClick={() => editor(kind, row)} className="p-1 text-zinc-500 hover:text-amber-400"><Edit3 className="w-3.5 h-3.5"/></button>}{canDelete && <button title="Xóa" onClick={() => void remove(kind, row)} className="p-1 text-zinc-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5"/></button>}</div></div>)}
    </section>
  );

  return <div className="space-y-3"><div className="flex flex-wrap items-center justify-end gap-2">{canEdit && <button onClick={() => void seedTestData()} disabled={busy} className="px-2.5 py-1 rounded bg-purple-600/25 border border-purple-500/40 text-[10px] font-semibold text-purple-200 hover:bg-purple-600/40"><Sparkles className="inline w-3 h-3 mr-1"/>Nạp Data Test Lao Động (9 Tộc + Cổ Tiên)</button>}<button onClick={() => void load()} disabled={busy} className="text-[10px] text-zinc-500 hover:text-zinc-200"><RefreshCw className={`inline w-3 h-3 mr-1 ${busy ? "animate-spin" : ""}`}/>Làm mới</button></div><div className="grid lg:grid-cols-2 gap-3">{list("node", nodes)}{list("person", people)}</div>{editing && <div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4" onClick={() => setEditing(null)}><div onClick={e => e.stopPropagation()} className="w-full max-w-3xl bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"><div className="flex justify-between items-center"><h3 className="text-sm font-bold text-zinc-200">{editing.row ? "Sửa toàn bộ dữ liệu" : "Thêm bản ghi"} · {editing.kind === "node" ? "công trình" : "dân cư dị nhân"}</h3><button onClick={() => setEditing(null)}><X className="w-4 h-4 text-zinc-500"/></button></div><p className="text-[10px] text-zinc-500">JSON chứa toàn bộ field của bảng. Không sửa id/aperture_id khi cập nhật.</p><textarea value={editing.json} onChange={e => setEditing({ ...editing, json: e.target.value })} rows={20} spellCheck={false} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-[11px] font-mono text-zinc-200"/><div className="flex justify-end"><button onClick={() => void save()} className="px-3 py-2 rounded bg-amber-600 text-xs text-white"><Save className="inline w-3.5 h-3.5 mr-1"/>Lưu</button></div></div></div>}</div>;
};
