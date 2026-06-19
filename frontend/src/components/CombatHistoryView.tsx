"use client";

import React, { useEffect, useState } from "react";
import { useGameStore, CombatHistoryEntry } from "@/store/useGameStore";
import {
  Swords,
  Flame,
  Eye,
  Skull,
  Heart,
  Shield,
  ArrowLeft,
  Clock,
  MapPin,
  Play,
  FastForward,
  ShieldAlert,
} from "lucide-react";

export const CombatHistoryView: React.FC = () => {
  const { combatHistory, combatHistoryLoading, fetchCombatHistory, setView, fetchCombatHistoryEntry, player } = useGameStore();
  const [selectedEntry, setSelectedEntry] = useState<CombatHistoryEntry | null>(null);
  const [replayLogs, setReplayLogs] = useState<any[]>([]);
  const [currentLogIndex, setCurrentLogIndex] = useState(-1);
  const [isReplaying, setIsReplaying] = useState(false);

  useEffect(() => {
    fetchCombatHistory();
  }, [fetchCombatHistory]);

  useEffect(() => {
    if (isReplaying && currentLogIndex < replayLogs.length - 1) {
      const timer = setTimeout(() => setCurrentLogIndex((prev) => prev + 1), 400);
      return () => clearTimeout(timer);
    } else if (isReplaying && currentLogIndex >= replayLogs.length - 1) {
      setIsReplaying(false);
    }
  }, [isReplaying, currentLogIndex, replayLogs]);

  const handleSelectEntry = async (entry: CombatHistoryEntry) => {
    const full = await fetchCombatHistoryEntry(entry.id);
    if (full) {
      setSelectedEntry(full);
      setReplayLogs(full.logs || []);
      setCurrentLogIndex(full.logs ? full.logs.length - 1 : -1);
      setIsReplaying(false);
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
    const summary = selectedEntry.summary || {};
    const isVictory = selectedEntry.winner === "Phe Ta";

    return (
      <div className="w-full max-w-4xl mx-auto px-4 py-4">
        <button
          onClick={() => { setSelectedEntry(null); setReplayLogs([]); setCurrentLogIndex(-1); }}
          className="flex items-center gap-2 text-zinc-400 hover:text-zinc-200 mb-4 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span className="text-sm">Quay lại danh sách</span>
        </button>

        <div className="bg-zinc-900/60 border border-zinc-800 rounded-xl p-4 mb-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
              <Swords className="w-4 h-4 text-amber-400" />
              Kết quả chiến đấu
            </h3>
            <span className={`px-3 py-1 rounded-lg text-xs font-bold ${isVictory ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-rose-900/50 text-rose-400 border border-rose-800'}`}>
              {isVictory ? 'THẮNG LỢI' : 'THẤT BẠI'}
            </span>
          </div>
          <div className="flex flex-wrap gap-2 text-xs text-zinc-500 mb-3">
            <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> {formatDate(selectedEntry.created_at)}</span>
            <span className="flex items-center gap-1"><MapPin className="w-3 h-3" /> {selectedEntry.region}</span>
            <span className="flex items-center gap-1"><Skull className="w-3 h-3" /> {selectedEntry.enemy_name || selectedEntry.enemy_type}</span>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
            <div className="bg-zinc-950/50 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500 mb-1">HP còn lại</div>
              <div className={`font-bold text-lg ${summary.final_player_hp > 50 ? 'text-emerald-400' : summary.final_player_hp > 0 ? 'text-yellow-400' : 'text-rose-400'}`}>
                {summary.final_player_hp ?? "?"}
              </div>
            </div>
            <div className="bg-zinc-950/50 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500 mb-1">Sát thương gây</div>
              <div className="font-bold text-lg text-amber-400">{summary.total_damage_dealt?.toLocaleString() || 0}</div>
            </div>
            <div className="bg-zinc-950/50 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500 mb-1">Sát thương nhận</div>
              <div className="font-bold text-lg text-rose-400">{summary.total_damage_taken?.toLocaleString() || 0}</div>
            </div>
            <div className="bg-zinc-950/50 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500 mb-1">Khiên dùng</div>
              <div className="font-bold text-lg text-blue-400">{summary.shields_used || 0}</div>
            </div>
          </div>
        </div>

        <div className="bg-zinc-900/60 border border-zinc-800 rounded-xl p-4 mb-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-bold text-zinc-200">Nhật ký chiến đấu</h3>
            <div className="flex gap-2">
              {!isReplaying ? (
                <button
                  onClick={() => { setCurrentLogIndex(0); setIsReplaying(true); }}
                  className="px-3 py-1.5 bg-purple-900/50 text-purple-300 rounded text-xs font-bold border border-purple-800 hover:bg-purple-800/50 transition-all flex items-center gap-1"
                  disabled={replayLogs.length === 0}
                >
                  <Play className="w-3 h-3" /> Xem lại
                </button>
              ) : (
                <button
                  onClick={() => { setCurrentLogIndex(replayLogs.length - 1); setIsReplaying(false); }}
                  className="px-3 py-1.5 bg-zinc-800 text-zinc-300 rounded text-xs hover:bg-zinc-700 transition-all flex items-center gap-1"
                >
                  <FastForward className="w-3 h-3" /> Bỏ qua
                </button>
              )}
            </div>
          </div>

          <div className="bg-zinc-950 border border-zinc-800 rounded-xl p-4 overflow-y-auto font-mono text-sm space-y-3 max-h-[500px] shadow-inner">
            {replayLogs.slice(0, currentLogIndex + 1).map((log, idx) => (
              <div key={idx} className={`p-3 rounded-lg border ${log.action === 'BATTLE_END' ? 'border-amber-500/50 bg-amber-950/20' : log.trigger ? 'border-cyan-500/30 bg-cyan-950/10' : log.action === 'DIED' ? 'border-rose-900 bg-rose-950/30 text-rose-400' : 'border-zinc-800/50 bg-zinc-900/30'}`}>
                <div className="flex items-center gap-2 mb-1">
                  {log.turn > 0 && <span className="text-[10px] px-1.5 py-0.5 bg-zinc-800 text-zinc-400 rounded">Turn {log.turn}</span>}
                  {log.trigger && <span className="text-[10px] px-1.5 py-0.5 bg-cyan-900 text-cyan-200 rounded flex items-center gap-1"><ShieldAlert className="w-3 h-3" /> Trigger</span>}
                  {log.action === 'DIED' && <span className="text-[10px] px-1.5 py-0.5 bg-rose-900 text-rose-200 rounded flex items-center gap-1"><Skull className="w-3 h-3" /> Tử</span>}
                  {log.action === 'DOT_TICK' && <span className="text-[10px] px-1.5 py-0.5 bg-green-900 text-green-200 rounded">DOT</span>}
                </div>
                <div className={`${log.action === 'BATTLE_END' ? 'text-amber-400 font-bold' : 'text-zinc-300'}`}>
                  {log.detail}
                </div>
                {log.damage > 0 && log.action !== 'DOT_TICK' && log.action !== 'SHIELD_GAIN' && (
                  <div className="text-rose-400 font-bold text-xs mt-1">-{log.damage} HP</div>
                )}
              </div>
            ))}
            {replayLogs.length === 0 && (
              <div className="text-zinc-600 text-center py-8">Không có dữ liệu nhật ký</div>
            )}
          </div>
        </div>
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
                  <span className="flex items-center gap-1"><Swords className="w-3 h-3" /> DMG: {summary.total_damage_dealt?.toLocaleString() || 0}</span>
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
