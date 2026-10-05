"use client";

import React, { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, Clock3, Edit3, MapPin, RefreshCw, Search, Trash2, X } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const PAGE_SIZE = 25;
type MapDrop = Record<string, unknown>;

function toDateTimeInput(value: unknown): string {
  if (!value) return "";
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return "";
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export const MapDropsAdminTab: React.FC<{
  addToast: (message: string, type: "success" | "error" | "info") => void;
  canEdit: boolean;
  canDelete: boolean;
}> = ({ addToast, canEdit, canDelete }) => {
  const [rows, setRows] = useState<MapDrop[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [searchText, setSearchText] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("active");
  const [loading, setLoading] = useState(false);
  const [edit, setEdit] = useState<MapDrop | null>(null);
  const [lootJSON, setLootJSON] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => { setPage(1); setSearch(searchText.trim()); }, 350);
    return () => clearTimeout(timer);
  }, [searchText]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search, status });
      const response = await authFetch(`${BACKEND_URL}/api/admin/map-drops?${params}`);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể tải map drops");
      setRows(Array.isArray(result.data) ? result.data : []);
      setTotal(Number(result.total) || 0);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải map drops", "error");
    } finally { setLoading(false); }
  }, [page, search, status, addToast]);

  useEffect(() => { void load(); }, [load]);

  const save = async () => {
    if (!edit) return;
    let lootData: unknown;
    try {
      lootData = JSON.parse(lootJSON);
      if (!lootData || Array.isArray(lootData) || typeof lootData !== "object") throw new Error("object required");
    } catch {
      addToast("Loot data phải là JSON object hợp lệ", "error");
      return;
    }
    const action = prepareAdminAction("updateMapDrop", "cập nhật map drop", String(edit.id));
    if (!action) return;
    setSaving(true);
    try {
      const payload = {
        region_id: String(edit.region_id ?? ""), x: Number(edit.x), y: Number(edit.y),
        loser_id: String(edit.loser_id ?? ""), loser_name: String(edit.loser_name ?? ""),
        killer_id: String(edit.killer_id ?? ""), killer_name: String(edit.killer_name ?? ""),
        loot_data: lootData,
        protected_until: edit.protected_until ? new Date(String(edit.protected_until)).toISOString() : "",
        expires_at: new Date(String(edit.expires_at)).toISOString(),
      };
      const response = await authFetch(`${BACKEND_URL}/api/admin/map-drops/${edit.id}`, {
        method: "PUT", headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey }, body: JSON.stringify(payload),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể cập nhật map drop");
      addToast(result.message || "Đã cập nhật map drop", "success");
      setEdit(null);
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi cập nhật map drop", "error");
    } finally { setSaving(false); }
  };

  const remove = async (drop: MapDrop) => {
    const endpoint = `${BACKEND_URL}/api/admin/map-drops/${drop.id}`;
    const action = prepareAdminAction("deleteMapDrop", "xóa map drop", `${drop.region_id} (${drop.x}, ${drop.y}) · ${drop.id}`);
    if (!action) return;
    try {
      const response = await authFetch(endpoint, {
        method: "DELETE", headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ ...action, confirmation_target: new URL(endpoint).pathname }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể xóa map drop");
      addToast(result.message || "Đã xóa map drop", "success");
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi xóa map drop", "error");
    }
  };

  const beginEdit = (drop: MapDrop) => {
    setEdit({ ...drop, protected_until: toDateTimeInput(drop.protected_until), expires_at: toDateTimeInput(drop.expires_at) });
    setLootJSON(JSON.stringify(drop.loot_data ?? {}, null, 2));
  };
  const date = (value: unknown) => value ? new Date(String(value)).toLocaleString("vi-VN") : "—";
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return <section className="rounded-xl border border-zinc-800 bg-zinc-900/30 p-4 space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="flex items-center gap-2 text-sm font-bold text-zinc-100"><MapPin className="h-4 w-4 text-sky-400" />Đồ rơi trên bản đồ</h2><p className="mt-1 text-xs text-zinc-500">Theo dõi vị trí, người rơi đồ, thời hạn và loot; sửa hoặc xóa có audit.</p></div><button type="button" onClick={() => void load()} disabled={loading} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"><RefreshCw className={`mr-1.5 inline h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />Làm mới</button></div>
    <div className="flex flex-wrap gap-2"><label className="flex min-w-[250px] flex-1 items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2"><Search className="h-3.5 w-3.5 text-zinc-500" /><input value={searchText} onChange={event => setSearchText(event.target.value)} placeholder="Tìm ID, khu vực, người chơi hoặc loot" className="w-full bg-transparent text-xs text-zinc-200 outline-none" /></label><select value={status} onChange={event => { setPage(1); setStatus(event.target.value); }} className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-300"><option value="active">Còn hạn</option><option value="expired">Đã hết hạn</option><option value="all">Tất cả</option></select></div>
    <div className="overflow-x-auto rounded-lg border border-zinc-800"><table className="w-full min-w-[1050px] text-left text-xs"><thead className="bg-zinc-950 text-[10px] uppercase text-zinc-500"><tr><th className="px-3 py-2.5">Vị trí</th><th className="px-3 py-2.5">Người rơi đồ</th><th className="px-3 py-2.5">Người kết liễu</th><th className="px-3 py-2.5">Loot</th><th className="px-3 py-2.5">Bảo vệ / hết hạn</th><th className="px-3 py-2.5">Tạo lúc</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody className="divide-y divide-zinc-800/70">
      {rows.length === 0 ? <tr><td colSpan={7} className="py-10 text-center text-zinc-500">{loading ? "Đang tải…" : "Không có map drop phù hợp"}</td></tr> : rows.map(drop => {
        const loot = drop.loot_data && typeof drop.loot_data === "object" ? drop.loot_data as Record<string, unknown> : {};
        const expired = Boolean(drop.is_expired);
        return <tr key={String(drop.id)} className="hover:bg-zinc-900/70"><td className="px-3 py-3"><span className="font-semibold text-zinc-200">{String(drop.region_id)} ({String(drop.x)}, {String(drop.y)})</span><span className="mt-0.5 block font-mono text-[9px] text-zinc-600">{String(drop.id)}</span></td><td className="px-3 py-3 text-zinc-300">{String(drop.loser_name || "—")}<span className="mt-0.5 block font-mono text-[9px] text-zinc-600">{String(drop.loser_id)}</span></td><td className="px-3 py-3 text-zinc-400">{String(drop.killer_name || "—")}</td><td className="max-w-64 truncate px-3 py-3 text-zinc-400" title={JSON.stringify(loot)}>{Object.keys(loot).map(key => `${key}: ${Array.isArray(loot[key]) ? loot[key].length : String(loot[key])}`).join(" · ") || "{}"}</td><td className="px-3 py-3"><span className={expired ? "text-zinc-500" : "text-emerald-400"}><Clock3 className="mr-1 inline h-3 w-3" />{expired ? "Hết hạn" : "Còn hạn"}</span><span className="mt-1 block text-[10px] text-zinc-500">Bảo vệ: {date(drop.protected_until)}</span><span className="block text-[10px] text-zinc-500">Hết: {date(drop.expires_at)}</span></td><td className="px-3 py-3 text-zinc-500">{date(drop.created_at)}</td><td className="px-3 py-3 text-right whitespace-nowrap">{canEdit && <button type="button" onClick={() => beginEdit(drop)} title="Sửa toàn bộ field" className="mr-1 rounded p-1.5 text-zinc-400 hover:bg-zinc-800 hover:text-amber-400"><Edit3 className="h-3.5 w-3.5" /></button>}{canDelete && <button type="button" onClick={() => void remove(drop)} title="Xóa có lý do và audit" className="rounded p-1.5 text-zinc-400 hover:bg-red-500/10 hover:text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>}</td></tr>;
      })}
    </tbody></table></div>
    <div className="flex items-center justify-between text-[10px] text-zinc-500"><span>{total ? `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, total)} / ${total}` : "0 drop"}</span><div className="flex items-center gap-2"><button type="button" disabled={page <= 1 || loading} onClick={() => setPage(value => Math.max(1, value - 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronLeft className="h-4 w-4" /></button><span>Trang {page}/{totalPages}</span><button type="button" disabled={page >= totalPages || loading} onClick={() => setPage(value => Math.min(totalPages, value + 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronRight className="h-4 w-4" /></button></div></div>

    {edit && <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/80 p-4 backdrop-blur-sm"><div className="max-h-[92vh] w-full max-w-3xl overflow-y-auto rounded-2xl border border-zinc-800 bg-zinc-950 p-5 space-y-4"><div className="flex items-center justify-between"><div><h3 className="text-sm font-bold text-zinc-100">Sửa map drop</h3><p className="mt-1 font-mono text-[10px] text-zinc-600">{String(edit.id)}</p></div><button type="button" onClick={() => setEdit(null)} className="rounded p-1.5 text-zinc-400 hover:bg-zinc-800"><X className="h-4 w-4" /></button></div><div className="grid gap-3 sm:grid-cols-2"><Field label="Region ID" value={edit.region_id} onChange={value => setEdit({ ...edit, region_id: value })} /><Field label="X" value={edit.x} type="number" onChange={value => setEdit({ ...edit, x: Number(value) })} /><Field label="Y" value={edit.y} type="number" onChange={value => setEdit({ ...edit, y: Number(value) })} /><Field label="Loser ID" value={edit.loser_id} onChange={value => setEdit({ ...edit, loser_id: value })} /><Field label="Tên người rơi đồ" value={edit.loser_name} onChange={value => setEdit({ ...edit, loser_name: value })} /><Field label="Killer ID (để trống nếu không có)" value={edit.killer_id} onChange={value => setEdit({ ...edit, killer_id: value })} /><Field label="Tên killer" value={edit.killer_name} onChange={value => setEdit({ ...edit, killer_name: value })} /><Field label="Bảo vệ đến" type="datetime-local" value={edit.protected_until} onChange={value => setEdit({ ...edit, protected_until: value })} /><Field label="Hết hạn" type="datetime-local" value={edit.expires_at} onChange={value => setEdit({ ...edit, expires_at: value })} /></div><label className="block space-y-1"><span className="text-[10px] font-semibold uppercase text-zinc-500">Loot data JSON</span><textarea value={lootJSON} onChange={event => setLootJSON(event.target.value)} rows={12} spellCheck={false} className="w-full rounded-lg border border-zinc-800 bg-black/40 p-3 font-mono text-[11px] text-zinc-300 outline-none focus:border-amber-500/40" /></label><div className="flex justify-end gap-2"><button type="button" onClick={() => setEdit(null)} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300">Hủy</button><button type="button" onClick={() => void save()} disabled={saving} className="rounded-lg bg-amber-500 px-4 py-2 text-xs font-bold text-zinc-950 disabled:opacity-50">{saving ? "Đang lưu…" : "Lưu thay đổi"}</button></div></div></div>}
  </section>;
};

const Field: React.FC<{ label: string; value: unknown; type?: string; onChange: (value: string) => void }> = ({ label, value, type = "text", onChange }) => <label className="block space-y-1"><span className="text-[10px] font-semibold uppercase text-zinc-500">{label}</span><input type={type} value={String(value ?? "")} onChange={event => onChange(event.target.value)} className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-200 outline-none focus:border-amber-500/40" /></label>;
