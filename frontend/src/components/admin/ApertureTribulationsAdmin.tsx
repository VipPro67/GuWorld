"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { ChevronLeft, ChevronRight, RefreshCw } from "lucide-react";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type EnemyRow = Record<string, unknown>;
type TribulationRow = Record<string, unknown> & { enemies?: EnemyRow[] };
const PAGE_SIZE = 25;

export const ApertureTribulationsAdmin: React.FC<{
  apertureId: string;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ apertureId, addToast }) => {
  const [rows, setRows] = useState<TribulationRow[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/apertures/${apertureId}/tribulations?limit=${PAGE_SIZE}&offset=${offset}`);
      if (!response.ok) throw new Error("Không thể tải lịch sử tai kiếp");
      const result = await response.json();
      setRows(result.data as TribulationRow[]);
      setTotal(Number(result.total) || 0);
    } catch (error) { addToast(error instanceof Error ? error.message : "Lỗi tải lịch sử tai kiếp", "error"); }
    finally { setLoading(false); }
  }, [apertureId, offset, addToast]);
  useEffect(() => { void load(); }, [load]);
  const fmt = (value: unknown) => value ? new Date(String(value)).toLocaleString("vi-VN") : "—";

  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between gap-2"><div><h4 className="text-xs font-bold text-zinc-200">Lịch sử tai kiếp ({total})</h4><p className="text-[10px] text-zinc-500">Đối chiếu sự kiện đã ghi nhận và các địch nhân được sinh ra.</p></div><button onClick={() => void load()} disabled={loading} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${loading ? "animate-spin" : ""}`} />Làm mới</button></div>
    {rows.length === 0 ? <p className="text-[10px] text-zinc-500">Tiên khiếu này chưa có sự kiện tai kiếp nào được lưu.</p> : rows.map(row => {
      const id = String(row.id);
      const enemies = Array.isArray(row.enemies) ? row.enemies : [];
      return <div key={id} className="border-t border-zinc-800/60 pt-2 space-y-1.5">
        <button onClick={() => setExpanded(expanded === id ? null : id)} className="w-full text-left">
          <div className="flex flex-wrap justify-between gap-x-3 gap-y-1 text-xs"><span className="font-semibold text-zinc-200">{String(row.tribulation_type)} · {String(row.path)}</span><span className={`${row.status === "ACTIVE" ? "text-amber-400" : row.status === "SUCCESS" ? "text-emerald-400" : "text-zinc-400"}`}>{String(row.status)}</span></div>
          <div className="text-[10px] text-zinc-500">Chủ nhân: {String(row.player_username)} · bắt đầu {fmt(row.started_at)} · kết thúc {fmt(row.ended_at)} · HP tiên khiếu {String(row.aperture_hp_before)} → {String(row.aperture_hp_after ?? "—")} · thưởng Đạo Ngân {String(row.reward_dao_marks)}</div>
        </button>
        {expanded === id && <div className="rounded bg-zinc-950 p-2 space-y-1.5"><div className="text-[10px] text-zinc-500 font-mono">Event ID: {id} · aperture: {String(row.aperture_id)} · player: {String(row.player_id)} · kẻ địch: {enemies.length}</div>{enemies.length === 0 ? <p className="text-[10px] text-zinc-600">Không có bản ghi địch nhân.</p> : enemies.map(enemy => <div key={String(enemy.id)} className="border-t border-zinc-800/60 pt-1 text-[10px] text-zinc-400"><div className="text-zinc-300">{String(enemy.name)} · {String(enemy.enemy_type)} · {String(enemy.path)} · rank {String(enemy.rank)}.{String(enemy.sub_rank)} {enemy.is_boss ? "· Boss" : ""} {enemy.is_dead ? "· Đã chết" : "· Còn sống"}</div><div>HP {String(enemy.hp)}/{String(enemy.max_hp)} · công {String(enemy.attack)} · thủ {String(enemy.defense)} · vị trí ({String(enemy.x)}, {String(enemy.y)}) · node mục tiêu {String(enemy.target_node_id || "—")} · tạo {fmt(enemy.created_at)} · cập nhật {fmt(enemy.updated_at)}</div></div>)}</div>}
      </div>;
    })}
    <div className="flex items-center justify-between border-t border-zinc-800/60 pt-2"><span className="text-[10px] text-zinc-600">{total ? `${offset + 1}–${Math.min(offset + PAGE_SIZE, total)} / ${total}` : "0"}</span><div className="flex gap-1"><button disabled={offset === 0 || loading} onClick={() => setOffset(value => Math.max(0, value - PAGE_SIZE))} className="p-1 text-zinc-400 disabled:opacity-30"><ChevronLeft className="w-3.5 h-3.5" /></button><button disabled={offset + PAGE_SIZE >= total || loading} onClick={() => setOffset(value => value + PAGE_SIZE)} className="p-1 text-zinc-400 disabled:opacity-30"><ChevronRight className="w-3.5 h-3.5" /></button></div></div>
  </section>;
};
