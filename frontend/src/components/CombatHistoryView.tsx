"use client";

import React, { useEffect, useState } from "react";
import { useGameStore, CombatHistoryEntry } from "@/store/useGameStore";
import {
  Swords,
  Flame,
  ArrowLeft,
  Clock,
  MapPin,
  Heart,
  Eye,
} from "lucide-react";
import { CombatReportView } from "./CombatReportView";
import type { CombatLogEntry } from "./CombatReplayView";
import { Num } from "./Num";

export const CombatHistoryView: React.FC = () => {
  const { combatHistory, combatHistoryLoading, fetchCombatHistory, setView, fetchCombatHistoryEntry, player } = useGameStore();
  const [selectedEntry, setSelectedEntry] = useState<CombatHistoryEntry | null>(null);
  const [replayLogs, setReplayLogs] = useState<CombatLogEntry[]>([]);

  useEffect(() => {
    fetchCombatHistory();
  }, [fetchCombatHistory]);

  const handleSelectEntry = async (entry: CombatHistoryEntry) => {
    const full = await fetchCombatHistoryEntry(entry.id);
    if (full) {
      setSelectedEntry(full);
      setReplayLogs(full.logs || []);
    }
  };

  const formatDate = (d: string) => {
    const date = new Date(d);
    return date.toLocaleString("vi-VN", {
      day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit",
    });
  };

  if (selectedEntry) {
    const combatants = selectedEntry.combatants || [];

    return (
      <div className="w-full mx-auto px-4 py-4">
        <CombatReportView
          logs={replayLogs}
          combatants={combatants as unknown as Parameters<typeof CombatReportView>[0]["combatants"]}
          winner={selectedEntry.winner}
          onBack={() => { setSelectedEntry(null); setReplayLogs([]); }}
          playerName={player?.username}
          teamAName={player?.username || "Phe Ta"}
          teamBName={selectedEntry.enemy_name || selectedEntry.enemy_type || "Kẻ Địch"}
          title={`Chiến Báo - vs ${selectedEntry.enemy_name || selectedEntry.enemy_type || "Kẻ Địch"}`}
        />
      </div>
    );
  }

  return (
    <div className="w-full mx-auto px-4 py-4">
      <button
        onClick={() => setView("dashboard")}
        className="flex items-center gap-2 text-zinc-400 hover:text-zinc-200 mb-4 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        <span className="text-sm">Quay lại</span>
      </button>

      <h2 className="text-xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-amber-400 to-rose-400 mb-6 flex items-center gap-2">
        <Flame className="w-5 h-5 text-amber-400" />
        Lịch sử chiến đấu
      </h2>

      {combatHistoryLoading ? (
        <div className="text-zinc-500 text-center py-12">Đang tải...</div>
      ) : combatHistory.length === 0 ? (
        <div className="bg-zinc-900/30 border border-zinc-800 rounded-xl p-8 text-center">
          <Swords className="w-12 h-12 text-zinc-700 mx-auto mb-3" />
          <p className="text-zinc-500">Chưa có trận chiến nào được ghi lại.</p>
          <p className="text-zinc-600 text-sm mt-1">Hãy tham gia chiến đấu để có dữ liệu xem lại.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {combatHistory.map((entry) => {
            const isVictory = entry.winner === "Phe Ta";
            const summary = entry.summary || {};
            const opponent = entry.enemy_name || entry.enemy_type || "Kẻ địch";
            return (
              <button
                key={entry.id}
                onClick={() => handleSelectEntry(entry)}
                className="group w-full overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/50 text-left transition-all hover:border-amber-700/60 hover:bg-zinc-900/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500"
              >
                <div className="relative flex h-32 items-center justify-around overflow-hidden bg-gradient-to-br from-zinc-800/80 via-zinc-900 to-rose-950/30 px-4">
                  <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(245,158,11,0.08),transparent_65%)]" />
                  <div className="relative flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border border-cyan-500/30 bg-zinc-950/70 shadow-lg shadow-cyan-950/30">
                    <img src={player?.avatar_url || "/assets/world/cultivator.webp"} alt={player?.username || "Nhân vật"} className="h-full w-full object-cover" />
                    <span className="absolute bottom-0 inset-x-0 bg-zinc-950/75 py-0.5 text-center text-[9px] font-semibold text-zinc-200">{player?.username || "Bạn"}</span>
                  </div>
                  <span className="relative rounded-full border border-zinc-600 bg-zinc-950/80 px-2 py-1 text-[10px] font-black text-amber-300">VS</span>
                  <div className="relative flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border border-rose-500/30 bg-zinc-950/70 shadow-lg shadow-rose-950/30">
                    <img src="/assets/world/spirit-beast.webp" alt={opponent} className="h-full w-full object-cover" />
                    <span className="absolute bottom-0 inset-x-0 truncate bg-zinc-950/75 px-1 py-0.5 text-center text-[9px] font-semibold text-zinc-200">{opponent}</span>
                  </div>
                </div>
                <div className="p-3.5">
                  <div className="mb-3 flex items-center justify-between gap-2">
                    <span className={`rounded border px-2 py-0.5 text-[10px] font-bold ${isVictory ? 'bg-emerald-900/50 text-emerald-400 border-emerald-800' : 'bg-rose-900/50 text-rose-400 border-rose-800'}`}>
                      {isVictory ? 'THẮNG' : 'BẠI'}
                    </span>
                    <span className="flex items-center gap-1 text-[10px] text-zinc-500"><Clock className="h-3 w-3" />{formatDate(entry.created_at)}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-x-3 gap-y-2 text-[11px] text-zinc-400">
                    <span className="flex min-w-0 items-center gap-1"><MapPin className="h-3 w-3 shrink-0 text-zinc-500" /><span className="truncate">{entry.region || "Không rõ khu vực"}</span></span>
                    <span className="flex items-center gap-1"><Heart className="h-3 w-3 shrink-0 text-rose-400" />HP còn: {summary.final_player_hp ?? "?"}</span>
                    <span className="flex items-center gap-1"><Swords className="h-3 w-3 shrink-0 text-amber-400" />DMG: <Num value={summary.total_damage_dealt || 0} /></span>
                    <span className="flex items-center gap-1"><Eye className="h-3 w-3 shrink-0 text-zinc-500" />{entry.logs_count ?? entry.logs?.length ?? 0} bước</span>
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};
