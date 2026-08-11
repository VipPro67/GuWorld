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
import { CombatReplayView, type CombatLogEntry } from "./CombatReplayView";
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
      <div className="w-full max-w-5xl mx-auto px-4 py-4">
        <CombatReplayView
          logs={replayLogs}
          combatants={combatants}
          winner={selectedEntry.winner}
          onBack={() => { setSelectedEntry(null); setReplayLogs([]); }}
          playerName={player?.username}
        />
      </div>
    );
  }

  return (
    <div className="w-full max-w-4xl mx-auto px-4 py-4">
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
        <div className="space-y-3">
          {combatHistory.map((entry) => {
            const isVictory = entry.winner === "Phe Ta";
            const summary = entry.summary || {};
            return (
              <div
                key={entry.id}
                onClick={() => handleSelectEntry(entry)}
                className="bg-zinc-900/40 border border-zinc-800 hover:border-zinc-700 rounded-xl p-4 cursor-pointer transition-all hover:bg-zinc-900/60"
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${isVictory ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-rose-900/50 text-rose-400 border border-rose-800'}`}>
                      {isVictory ? 'THẮNG' : 'BẠI'}
                    </span>
                    <span className="text-sm text-zinc-200 font-medium">
                      {entry.enemy_name || entry.enemy_type || "Kẻ địch"}
                    </span>
                    <span className="text-[10px] text-zinc-600">{entry.enemy_type}</span>
                  </div>
                  <span className="text-[10px] text-zinc-600 flex items-center gap-1">
                    <Clock className="w-3 h-3" /> {formatDate(entry.created_at)}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-[11px] text-zinc-500">
                  <span className="flex items-center gap-1"><MapPin className="w-3 h-3" /> {entry.region}</span>
                  <span className="flex items-center gap-1"><Heart className="w-3 h-3" /> HP còn: {summary.final_player_hp ?? "?"}</span>
                  <span className="flex items-center gap-1"><Swords className="w-3 h-3" /> DMG: <Num value={summary.total_damage_dealt || 0} /></span>
                  <span><Eye className="w-3 h-3 inline" /> {entry.logs?.length || 0} bước</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
