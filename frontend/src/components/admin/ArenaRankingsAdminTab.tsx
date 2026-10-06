"use client";

import React, { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, Edit3, RefreshCw, Search, Swords, X } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const PAGE_SIZE = 25;
type ArenaRanking = Record<string, unknown>;

function toDateTimeInput(value: unknown): string {
  if (!value) return "";
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return "";
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export const ArenaRankingsAdminTab: React.FC<{
  addToast: (message: string, type: "success" | "error" | "info") => void;
  canEdit: boolean;
}> = ({ addToast, canEdit }) => {
  const [rows, setRows] = useState<ArenaRanking[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [searchText, setSearchText] = useState("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [edit, setEdit] = useState<ArenaRanking | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => { setPage(1); setSearch(searchText.trim()); }, 350);
    return () => clearTimeout(timer);
  }, [searchText]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search });
      const response = await authFetch(`${BACKEND_URL}/api/admin/arena-rankings?${params}`);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể tải xếp hạng đấu trường");
      setRows(Array.isArray(result.data) ? result.data : []);
      setTotal(Number(result.total) || 0);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải xếp hạng đấu trường", "error");
    } finally { setLoading(false); }
  }, [page, search, addToast]);

  useEffect(() => { void load(); }, [load]);

  const save = async () => {
    if (!edit) return;
    const endpoint = `${BACKEND_URL}/api/admin/arena-rankings/${edit.player_id}`;
    const action = prepareAdminAction("updateArenaRanking", "điều chỉnh dữ liệu đấu trường", `${edit.username} · hạng ${edit.rank}`);
    if (!action) return;
    const lastChallenge = new Date(String(edit.last_challenge_at_input || ""));
    if (Number.isNaN(lastChallenge.getTime())) {
      addToast("Thời điểm thử gần nhất không hợp lệ", "error");
      return;
    }
    setSaving(true);
    try {
      const response = await authFetch(endpoint, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({
          points: Number(edit.points),
          daily_challenges: Number(edit.daily_challenges),
          last_challenge_at: lastChallenge.toISOString(),
          reason: action.reason,
        }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể cập nhật dữ liệu đấu trường");
      addToast(result.message || "Đã cập nhật dữ liệu đấu trường", "success");
      setEdit(null);
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi cập nhật đấu trường", "error");
    } finally { setSaving(false); }
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const date = (value: unknown) => value ? new Date(String(value)).toLocaleString("vi-VN") : "—";
  return <section className="space-y-4 rounded-xl border border-zinc-800 bg-zinc-900/30 p-4">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="flex items-center gap-2 text-sm font-bold text-zinc-100"><Swords className="h-4 w-4 text-amber-400" />Xếp hạng đấu trường</h2><p className="mt-1 text-xs text-zinc-500">Tra cứu theo hạng và người chơi. Hạng được hệ thống vận hành; các chỉ số có thể điều chỉnh đều được audit và yêu cầu lý do.</p></div><button type="button" onClick={() => void load()} disabled={loading} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"><RefreshCw className={`mr-1.5 inline h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />Làm mới</button></div>
    <label className="flex max-w-xl items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2"><Search className="h-3.5 w-3.5 text-zinc-500" /><input value={searchText} onChange={event => setSearchText(event.target.value)} placeholder="Tìm tên hoặc ID người chơi" className="w-full bg-transparent text-xs text-zinc-200 outline-none" /></label>
    <div className="overflow-x-auto rounded-lg border border-zinc-800"><table className="w-full min-w-[900px] text-left text-xs"><thead className="bg-zinc-950 text-[10px] uppercase text-zinc-500"><tr><th className="px-3 py-2.5">Hạng</th><th className="px-3 py-2.5">Người chơi</th><th className="px-3 py-2.5">Điểm</th><th className="px-3 py-2.5">Lượt thử hôm nay</th><th className="px-3 py-2.5">Thử gần nhất</th><th className="px-3 py-2.5">Cập nhật</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr></thead><tbody className="divide-y divide-zinc-800/70">
      {rows.length === 0 ? <tr><td colSpan={7} className="py-10 text-center text-zinc-500">{loading ? "Đang tải…" : "Không có dữ liệu xếp hạng phù hợp"}</td></tr> : rows.map(row => <tr key={String(row.player_id)} className="hover:bg-zinc-900/70"><td className="px-3 py-3 font-bold text-amber-400">#{String(row.rank)}</td><td className="px-3 py-3"><span className="text-zinc-200">{String(row.username)}</span><span className="mt-0.5 block font-mono text-[9px] text-zinc-600">{String(row.player_id)}</span></td><td className="px-3 py-3 font-semibold text-zinc-200">{String(row.points)}</td><td className="px-3 py-3 text-zinc-300">{String(row.daily_challenges)}</td><td className="px-3 py-3 text-zinc-400">{date(row.last_challenge_at)}</td><td className="px-3 py-3 text-zinc-500">{date(row.updated_at)}</td><td className="px-3 py-3 text-right">{canEdit && <button type="button" onClick={() => setEdit({ ...row, last_challenge_at_input: toDateTimeInput(row.last_challenge_at) })} title="Sửa chỉ số" className="rounded p-1.5 text-zinc-400 hover:bg-zinc-800 hover:text-amber-400"><Edit3 className="h-3.5 w-3.5" /></button>}</td></tr>)}
    </tbody></table></div>
    <div className="flex items-center justify-between text-[10px] text-zinc-500"><span>{total ? `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, total)} / ${total}` : "0 người chơi"}</span><div className="flex items-center gap-2"><button type="button" disabled={page <= 1 || loading} onClick={() => setPage(value => Math.max(1, value - 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronLeft className="h-4 w-4" /></button><span>Trang {page}/{totalPages}</span><button type="button" disabled={page >= totalPages || loading} onClick={() => setPage(value => Math.min(totalPages, value + 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronRight className="h-4 w-4" /></button></div></div>
    {edit && <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/80 p-4 backdrop-blur-sm"><div className="w-full max-w-lg space-y-4 rounded-2xl border border-zinc-800 bg-zinc-950 p-5"><div className="flex items-start justify-between"><div><h3 className="text-sm font-bold text-zinc-100">Điều chỉnh chỉ số đấu trường</h3><p className="mt-1 text-xs text-zinc-500">{String(edit.username)} · hạng #{String(edit.rank)} (không đổi)</p></div><button type="button" onClick={() => setEdit(null)} className="rounded p-1.5 text-zinc-400 hover:bg-zinc-800"><X className="h-4 w-4" /></button></div><div className="grid gap-3 sm:grid-cols-2"><Field label="Điểm" value={edit.points} onChange={value => setEdit({ ...edit, points: Number(value) })} /><Field label="Lượt thử hôm nay" value={edit.daily_challenges} onChange={value => setEdit({ ...edit, daily_challenges: Number(value) })} /><div className="sm:col-span-2"><Field label="Thử gần nhất" type="datetime-local" value={edit.last_challenge_at_input} onChange={value => setEdit({ ...edit, last_challenge_at_input: value })} /></div></div><div className="flex justify-end gap-2"><button type="button" onClick={() => setEdit(null)} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300">Hủy</button><button type="button" onClick={() => void save()} disabled={saving} className="rounded-lg bg-amber-500 px-4 py-2 text-xs font-bold text-zinc-950 disabled:opacity-50">{saving ? "Đang lưu…" : "Lưu thay đổi"}</button></div></div></div>}
  </section>;
};

const Field: React.FC<{ label: string; value: unknown; type?: string; onChange: (value: string) => void }> = ({ label, value, type = "number", onChange }) => <label className="block space-y-1"><span className="text-[10px] font-semibold uppercase text-zinc-500">{label}</span><input type={type} min={type === "number" ? 0 : undefined} value={String(value ?? "")} onChange={event => onChange(event.target.value)} className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-200 outline-none focus:border-amber-500/40" /></label>;
