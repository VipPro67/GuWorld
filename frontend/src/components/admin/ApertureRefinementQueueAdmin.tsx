"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Edit3, Plus, RefreshCw, Save, Trash2, X } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type QueueRow = Record<string, unknown>;

export const ApertureRefinementQueueAdmin: React.FC<{
  apertureId: string;
  canEdit: boolean;
  canDelete: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ apertureId, canEdit, canDelete, addToast, exec, prepareDelete }) => {
  const [jobs, setJobs] = useState<QueueRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<{ row: QueueRow | null; json: string } | null>(null);
  const base = `${BACKEND_URL}/api/admin/apertures/${apertureId}/refinement-queue`;
  const load = useCallback(async () => {
    setBusy(true);
    try { const res = await authFetch(base); if (!res.ok) throw new Error("Không thể tải hàng đợi luyện cổ"); setJobs(await res.json()); }
    catch (error) { addToast(error instanceof Error ? error.message : "Lỗi tải hàng đợi", "error"); }
    finally { setBusy(false); }
  }, [base, addToast]);
  useEffect(() => { void load(); }, [load]);

  const edit = (row: QueueRow | null) => setEditing({
    row,
    json: JSON.stringify(row || { recipe_id: "", mode: "FIXED_COUNT", target_quantity: 1, current_quantity: 0, priority: 0, enabled: true, last_error: "", last_processed_at: null }, null, 2),
  });
  const save = async () => {
    if (!editing) return;
    let payload: QueueRow;
    try { payload = JSON.parse(editing.json) as QueueRow; }
    catch { addToast("JSON không hợp lệ", "error"); return; }
    const id = String(editing.row?.id || "");
    if (await exec(`${base}${id ? `/${id}` : ""}`, id ? "PUT" : "POST", payload as ApiResponseData)) { setEditing(null); void load(); }
  };
  const remove = async (row: QueueRow) => {
    const id = String(row.id);
    const prepared = prepareDelete("xóa công việc luyện cổ", `${row.recipe_id} · ${id}`);
    if (prepared && await exec(`${base}/${id}`, "DELETE", prepared)) void load();
  };

  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between"><h4 className="text-xs font-bold text-zinc-200">Hàng đợi luyện cổ tự động ({jobs.length})</h4><div className="flex gap-2"><button onClick={() => void load()} disabled={busy} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${busy ? "animate-spin" : ""}`}/>Làm mới</button>{canEdit && <button onClick={() => edit(null)} className="px-2 py-1 rounded bg-amber-600/20 text-amber-400 text-[10px]"><Plus className="inline w-3 h-3 mr-1"/>Thêm</button>}</div></div>
    {jobs.length === 0 ? <p className="text-[10px] text-zinc-500">Chưa có công việc luyện cổ.</p> : jobs.map(row => <div key={String(row.id)} className="flex items-center justify-between gap-2 border-t border-zinc-800/60 pt-2"><div className="min-w-0"><div className="text-xs text-zinc-300 truncate">{String(row.recipe_id)} · {String(row.mode)}</div><div className="text-[10px] text-zinc-500">{String(row.current_quantity)} / {String(row.target_quantity)} · ưu tiên {String(row.priority)} · {row.enabled ? "đang bật" : "đã tắt"}{row.last_error ? ` · ${String(row.last_error)}` : ""}</div></div><div className="flex gap-1">{canEdit && <button onClick={() => edit(row)} className="p-1 text-zinc-500 hover:text-amber-400"><Edit3 className="w-3.5 h-3.5"/></button>}{canDelete && <button onClick={() => void remove(row)} className="p-1 text-zinc-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5"/></button>}</div></div>)}
    {editing && <div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4" onClick={() => setEditing(null)}><div onClick={e => e.stopPropagation()} className="w-full max-w-2xl bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"><div className="flex justify-between items-center"><h3 className="text-sm font-bold text-zinc-200">{editing.row ? "Sửa công việc luyện cổ" : "Thêm công việc luyện cổ"}</h3><button onClick={() => setEditing(null)}><X className="w-4 h-4 text-zinc-500"/></button></div><p className="text-[10px] text-zinc-500">Hỗ trợ FIXED_COUNT, UNTIL_SUCCESS, UNTIL_MATERIAL_EMPTY, MAINTAIN_STOCK. Có thể chỉnh cả tiến độ và thông báo lỗi.</p><textarea value={editing.json} onChange={e => setEditing({ ...editing, json: e.target.value })} rows={16} spellCheck={false} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-[11px] font-mono text-zinc-200"/><div className="flex justify-end"><button onClick={() => void save()} className="px-3 py-2 rounded bg-amber-600 text-xs text-white"><Save className="inline w-3.5 h-3.5 mr-1"/>Lưu</button></div></div></div>}
  </section>;
};
