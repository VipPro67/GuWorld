"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { RefreshCw, UserRoundPlus, UserRoundCog, XCircle } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type SectRequest = Record<string, unknown>;

export const SectRequestsAdmin: React.FC<{
  sectId: string;
  canCancel: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ sectId, canCancel, addToast, exec, prepareDelete }) => {
  const [rows, setRows] = useState<SectRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const base = `${BACKEND_URL}/api/admin/sects/${sectId}/requests`;
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await authFetch(base);
      if (!response.ok) throw new Error("Không thể tải yêu cầu tông môn");
      setRows(await response.json() as SectRequest[]);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải yêu cầu tông môn", "error");
    } finally { setLoading(false); }
  }, [base, addToast]);
  useEffect(() => { void load(); }, [load]);
  const cancel = async (row: SectRequest) => {
    const id = String(row.id ?? "");
    const type = String(row.request_type ?? "");
    const label = `${type === "join" ? "yêu cầu gia nhập" : "yêu cầu thăng chức"} của ${String(row.username ?? row.player_id)} · ${id}`;
    const prepared = prepareDelete("hủy yêu cầu đang chờ", label);
    if (prepared && await exec(`${base}/${type}/${id}`, "DELETE", prepared)) void load();
  };

  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between">
      <h4 className="text-xs font-bold text-zinc-200">Yêu cầu tông môn ({rows.length})</h4>
      <button onClick={() => void load()} disabled={loading} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${loading ? "animate-spin" : ""}`} />Làm mới</button>
    </div>
    {rows.length === 0 ? <p className="text-[10px] text-zinc-500">Không có yêu cầu tông môn.</p> : rows.map(row => {
      const type = String(row.request_type);
      const pending = String(row.status).toUpperCase() === "PENDING";
      return <div key={`${type}:${String(row.id)}`} className="flex items-center justify-between gap-2 border-t border-zinc-800/60 pt-2">
        <div className="min-w-0">
          <div className="text-xs text-zinc-200 truncate">{type === "join" ? <UserRoundPlus className="inline w-3.5 h-3.5 mr-1 text-emerald-400" /> : <UserRoundCog className="inline w-3.5 h-3.5 mr-1 text-amber-400" />}{String(row.username)} · {type === "join" ? "Xin gia nhập" : `Thăng ${String(row.current_rank)} → ${String(row.target_rank)}`}</div>
          <div className="text-[10px] text-zinc-500 truncate">{String(row.status)} · tu vi rank {String(row.player_rank)} · {String(row.created_at)} · ID {String(row.player_id)}</div>
        </div>
        {canCancel && pending && <button title="Hủy yêu cầu đang chờ" onClick={() => void cancel(row)} className="p-1 text-zinc-500 hover:text-red-400"><XCircle className="w-3.5 h-3.5" /></button>}
      </div>;
    })}
  </section>;
};
