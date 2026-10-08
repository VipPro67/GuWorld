"use client";

import React, { useCallback, useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, RefreshCw, Search, Swords } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { CombatReportView } from "@/components/CombatReportView";
import type { CombatLogEntry } from "@/components/CombatReplayView";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const PAGE_SIZE = 25;
type CombatRow = Record<string, unknown>;

export const CombatHistoryAdminTab: React.FC<{
  addToast: (message: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const [rows, setRows] = useState<CombatRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [searchText, setSearchText] = useState("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, CombatRow>>({});
  const [detailLoading, setDetailLoading] = useState<string | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => { setPage(1); setSearch(searchText.trim()); }, 350);
    return () => clearTimeout(timer);
  }, [searchText]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search });
      const response = await authFetch(`${BACKEND_URL}/api/admin/combat-history?${params}`);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể tải lịch sử chiến đấu");
      setRows(Array.isArray(result.data) ? result.data : []);
      setTotal(Number(result.total) || 0);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải lịch sử chiến đấu", "error");
    } finally { setLoading(false); }
  }, [page, search, addToast]);

  useEffect(() => { void load(); }, [load]);

  const toggleDetails = async (id: string) => {
    if (expanded === id) { setExpanded(null); return; }
    setExpanded(id);
    if (details[id]) return;
    setDetailLoading(id);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/combat-history/${id}`);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể tải replay");
      setDetails(current => ({ ...current, [id]: result as CombatRow }));
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải replay", "error");
      setExpanded(null);
    } finally { setDetailLoading(null); }
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const date = (value: unknown) => value ? new Date(String(value)).toLocaleString("vi-VN") : "—";

  return <section className="rounded-xl border border-zinc-800 bg-zinc-900/30 p-4 space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div><h2 className="flex items-center gap-2 text-sm font-bold text-zinc-100"><Swords className="h-4 w-4 text-amber-400" />Lịch sử chiến đấu</h2><p className="mt-1 text-xs text-zinc-500">Chỉ đọc. Gồm PVP và các trận PVE thua; mỗi người chơi tham chiến có bản ghi riêng; mở trận để xem logs và đội hình.</p></div>
      <button type="button" onClick={() => void load()} disabled={loading} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"><RefreshCw className={`mr-1.5 inline h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />Làm mới</button>
    </div>
    <label className="flex max-w-xl items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2"><Search className="h-3.5 w-3.5 text-zinc-500" /><input value={searchText} onChange={event => setSearchText(event.target.value)} placeholder="Tìm người chơi, đối thủ, khu vực hoặc ID trận" className="w-full bg-transparent text-xs text-zinc-200 outline-none" /></label>
    <div className="overflow-x-auto rounded-lg border border-zinc-800"><table className="w-full min-w-[850px] text-left text-xs">
      <thead className="bg-zinc-950 text-[10px] uppercase text-zinc-500"><tr><th className="px-3 py-2.5">Người chơi</th><th className="px-3 py-2.5">Đối thủ</th><th className="px-3 py-2.5">Kết quả</th><th className="px-3 py-2.5">Khu vực</th><th className="px-3 py-2.5">Logs</th><th className="px-3 py-2.5">Thời điểm</th><th className="px-3 py-2.5 text-right">Chi tiết</th></tr></thead>
      <tbody className="divide-y divide-zinc-800/70">
        {rows.length === 0 ? <tr><td colSpan={7} className="py-10 text-center text-zinc-500">{loading ? "Đang tải…" : "Không tìm thấy lịch sử chiến đấu"}</td></tr> : rows.map(row => {
          const id = String(row.id);
          const detail = details[id];
          return <React.Fragment key={id}><tr className="hover:bg-zinc-900/70"><td className="px-3 py-3"><span className="text-zinc-200">{String(row.player_username || "(đã xóa người chơi)")}</span><span className="mt-0.5 block font-mono text-[9px] text-zinc-600">{String(row.player_id)}</span></td><td className="px-3 py-3 text-zinc-300">{String(row.enemy_name || "—")}<span className="mt-0.5 block text-[9px] uppercase text-zinc-600">{String(row.enemy_type || "")}</span></td><td className={`px-3 py-3 font-semibold ${row.winner === "Phe Ta" ? "text-emerald-400" : "text-red-400"}`}>{String(row.winner)}</td><td className="px-3 py-3 text-zinc-400">{String(row.region || "—")}</td><td className="px-3 py-3 text-zinc-400">{String(row.logs_count ?? 0)}</td><td className="px-3 py-3 text-zinc-500">{date(row.created_at)}</td><td className="px-3 py-3 text-right"><button type="button" onClick={() => void toggleDetails(id)} className="rounded px-2 py-1 text-amber-400 hover:bg-amber-500/10">{expanded === id ? "Đóng" : "Xem replay"}</button></td></tr>
            {expanded === id && <tr><td colSpan={7} className="bg-zinc-950/70 p-3">{detailLoading === id ? <p className="text-xs text-zinc-500">Đang tải replay…</p> : detail ? <div className="space-y-2"><div className="flex items-center justify-between"><h3 className="text-xs font-bold text-zinc-300">Replay · {String(detail.player_username || row.player_username || "Người chơi")} vs {String(detail.enemy_name || row.enemy_name || "Kẻ địch")}</h3><span className="text-[10px] text-zinc-500">{date(detail.created_at || row.created_at)}</span></div><CombatReportView
              logs={(Array.isArray(detail.logs) ? detail.logs : []) as CombatLogEntry[]}
              combatants={(Array.isArray(detail.combatants) ? detail.combatants : []) as unknown as Parameters<typeof CombatReportView>[0]["combatants"]}
              winner={String(detail.winner || row.winner || "Phe Địch")}
              playerName={String(detail.player_username || row.player_username || "")}
              teamAName={String(detail.player_username || row.player_username || "Phe Ta")}
              teamBName={String(detail.enemy_name || row.enemy_name || detail.enemy_type || "Kẻ Địch")}
              title={`Chiến Báo - vs ${String(detail.enemy_name || row.enemy_name || "Kẻ Địch")}`}
              onBack={() => setExpanded(null)}
            /></div> : <p className="text-xs text-zinc-500">Không có dữ liệu replay.</p>}</td></tr>}
          </React.Fragment>;
        })}
      </tbody>
    </table></div>
    <div className="flex items-center justify-between text-[10px] text-zinc-500"><span>{total ? `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, total)} / ${total}` : "0 trận"}</span><div className="flex items-center gap-2"><button type="button" disabled={page <= 1 || loading} onClick={() => setPage(value => Math.max(1, value - 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronLeft className="h-4 w-4" /></button><span>Trang {page}/{totalPages}</span><button type="button" disabled={page >= totalPages || loading} onClick={() => setPage(value => Math.min(totalPages, value + 1))} className="rounded p-1.5 text-zinc-400 disabled:opacity-30"><ChevronRight className="h-4 w-4" /></button></div></div>
  </section>;
};
