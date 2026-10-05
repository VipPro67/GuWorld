"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { RefreshCw, Shield, UserMinus } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type Defender = Record<string, unknown>;

export const SectDefendersAdmin: React.FC<{
  sectId: string;
  canRemove: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ sectId, canRemove, addToast, exec, prepareDelete }) => {
  const [rows, setRows] = useState<Defender[]>([]);
  const [loading, setLoading] = useState(false);
  const base = `${BACKEND_URL}/api/admin/sects/${sectId}/defenders`;
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await authFetch(base);
      if (!response.ok) throw new Error("Không thể tải lực lượng phòng thủ");
      setRows(await response.json() as Defender[]);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi tải lực lượng phòng thủ", "error");
    } finally { setLoading(false); }
  }, [base, addToast]);
  useEffect(() => { void load(); }, [load]);
  const remove = async (row: Defender) => {
    const id = String(row.id ?? "");
    const label = `${String(row.name ?? "")}${row.player_username ? ` · ${String(row.player_username)}` : ""} · ${id}`;
    const prepared = prepareDelete("rút lực lượng phòng thủ", label);
    if (prepared && await exec(`${base}/${id}`, "DELETE", prepared)) void load();
  };

  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between">
      <h4 className="text-xs font-bold text-zinc-200"><Shield className="inline w-3.5 h-3.5 mr-1.5 text-amber-400" />Lực lượng phòng thủ ({rows.length})</h4>
      <button onClick={() => void load()} disabled={loading} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${loading ? "animate-spin" : ""}`} />Làm mới</button>
    </div>
    {rows.length === 0 ? <p className="text-[10px] text-zinc-500">Tông môn chưa bố trí lực lượng phòng thủ.</p> : rows.map(row => <div key={String(row.id)} className="flex items-center justify-between gap-2 border-t border-zinc-800/60 pt-2">
      <div className="min-w-0">
        <div className="text-xs text-zinc-200 truncate">{String(row.name)} · {row.defender_type === "beast" ? String(row.monster_name || row.monster_id) : String(row.player_username || row.player_id)}</div>
        <div className="text-[10px] text-zinc-500 truncate">{String(row.defender_type)} · Rank {String(row.monster_rank)}.{String(row.monster_sub_rank)} · sức mạnh {String(row.power)} · giao nhiệm vụ {String(row.assigned_at)}</div>
      </div>
      {canRemove && <button title="Rút lực lượng phòng thủ" onClick={() => void remove(row)} className="p-1 text-zinc-500 hover:text-red-400"><UserMinus className="w-3.5 h-3.5" /></button>}
    </div>)}
  </section>;
};
