"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Edit3, Plus, RefreshCw, Save, Trash2, X } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type SettlementRow = Record<string, unknown>;

export const ApertureSettlementsAdmin: React.FC<{
  apertureId: string;
  canEdit: boolean;
  canDelete: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ apertureId, canEdit, canDelete, addToast, exec, prepareDelete }) => {
  const [rows, setRows] = useState<SettlementRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<{ row: SettlementRow | null; json: string } | null>(null);
  const base = `${BACKEND_URL}/api/admin/apertures/${apertureId}/settlements`;
  const load = useCallback(async () => {
    setBusy(true);
    try { const res = await authFetch(base); if (!res.ok) throw new Error("Không thể tải khu định cư"); setRows(await res.json()); }
    catch (error) { addToast(error instanceof Error ? error.message : "Lỗi tải khu định cư", "error"); }
    finally { setBusy(false); }
  }, [base, addToast]);
  useEffect(() => { void load(); }, [load]);

  const edit = (row: SettlementRow | null) => setEditing({ row, json: JSON.stringify(row || {
    race: "", name: "", settlement_type: "VILLAGE", population_capacity: 10000, housing: 0, food_storage: 5000, security: 50,
  }, null, 2) });
  const save = async () => {
    if (!editing) return;
    let payload: SettlementRow;
    try { payload = JSON.parse(editing.json) as SettlementRow; }
    catch { addToast("JSON không hợp lệ", "error"); return; }
    const id = String(editing.row?.id || "");
    if (await exec(`${base}${id ? `/${id}` : ""}`, id ? "PUT" : "POST", payload as ApiResponseData)) { setEditing(null); void load(); }
  };
  const remove = async (row: SettlementRow) => {
    const id = String(row.id);
    const prepared = prepareDelete("xóa khu định cư", `${row.race} · ${row.name} · ${id}`);
    if (prepared && await exec(`${base}/${id}`, "DELETE", prepared)) void load();
  };

  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between"><h4 className="text-xs font-bold text-zinc-200">Khu định cư dị nhân ({rows.length})</h4><div className="flex gap-2"><button onClick={() => void load()} disabled={busy} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${busy ? "animate-spin" : ""}`}/>Làm mới</button>{canEdit && <button onClick={() => edit(null)} className="px-2 py-1 rounded bg-amber-600/20 text-amber-400 text-[10px]"><Plus className="inline w-3 h-3 mr-1"/>Thêm</button>}</div></div>
    {rows.length===0 ? <p className="text-[10px] text-zinc-500">Chưa có khu định cư.</p> : rows.map(row => <div key={String(row.id)} className="flex items-center justify-between gap-2 border-t border-zinc-800/60 pt-2"><div className="min-w-0"><div className="text-xs text-zinc-300 truncate">{String(row.name || row.race)} · {String(row.settlement_type)}</div><div className="text-[10px] text-zinc-500">Sức chứa {String(row.population_capacity)} · nhà {String(row.housing)} · lương thực {String(row.food_storage)} · an ninh {String(row.security)}%</div></div><div className="flex gap-1">{canEdit&&<button onClick={()=>edit(row)} className="p-1 text-zinc-500 hover:text-amber-400"><Edit3 className="w-3.5 h-3.5"/></button>}{canDelete&&<button onClick={()=>void remove(row)} className="p-1 text-zinc-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5"/></button>}</div></div>)}
    {editing&&<div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4" onClick={()=>setEditing(null)}><div onClick={e=>e.stopPropagation()} className="w-full max-w-2xl bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"><div className="flex justify-between items-center"><h3 className="text-sm font-bold text-zinc-200">{editing.row?"Sửa khu định cư":"Thêm khu định cư"}</h3><button onClick={()=>setEditing(null)}><X className="w-4 h-4 text-zinc-500"/></button></div><p className="text-[10px] text-zinc-500">Khu định cư cần gắn với chủng tộc đã có dân cư trong tiên khiếu.</p><textarea value={editing.json} onChange={e=>setEditing({...editing,json:e.target.value})} rows={12} spellCheck={false} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-[11px] font-mono text-zinc-200"/><div className="flex justify-end"><button onClick={()=>void save()} className="px-3 py-2 rounded bg-amber-600 text-xs text-white"><Save className="inline w-3.5 h-3.5 mr-1"/>Lưu</button></div></div></div>}
  </section>;
};
