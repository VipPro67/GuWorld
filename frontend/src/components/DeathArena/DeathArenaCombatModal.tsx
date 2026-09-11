"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameModal } from "@/components/ui";
import { Swords, Trophy, Shield, Heart, ScrollText, Users, Play } from "lucide-react";
import { CombatReplayView, type CombatLogEntry } from "../CombatReplayView";

interface DeathArenaCombatModalProps {
  matchId: string | null;
  onClose: () => void;
}

export default function DeathArenaCombatModal({ matchId, onClose }: DeathArenaCombatModalProps) {
  const { fetchDeathArenaMatchLog } = useGameStore();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any>(null);
  const [viewMode, setViewMode] = useState<"summary" | "replay" | "logs">("summary");
  const [selectedTurn, setSelectedTurn] = useState<number | "all">("all");

  useEffect(() => {
    if (!matchId) {
      setData(null);
      setViewMode("summary");
      return;
    }

    let isMounted = true;
    setLoading(true);
    fetchDeathArenaMatchLog(matchId)
      .then((res) => {
        if (isMounted) {
          setData(res);
          if (res?.combat_log && res.combat_log.length > 0) {
            setViewMode("replay");
          } else {
            setViewMode("summary");
          }
        }
      })
      .catch((err) => {
        console.error("Error fetching match log:", err);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [matchId, fetchDeathArenaMatchLog]);

  if (!matchId) return null;

  const match = data?.match;
  const teamAMembers = data?.team_a_members || [];
  const teamBMembers = data?.team_b_members || [];
  const logs: CombatLogEntry[] = data?.combat_log || [];

  const isTeamAWinner = match?.winner_team_id && match?.winner_team_id === match?.team_a_id;
  const isTeamBWinner = match?.winner_team_id && match?.winner_team_id === match?.team_b_id;

  // Build combatants list for CombatReplayView
  const replayCombatants = [
    ...teamAMembers.map((m: any, idx: number) => ({
      id: m.player_id,
      name: m.username,
      kind: "ally" as const,
      max_hp: m.max_hp || 1000,
      hp: m.current_hp ?? m.max_hp ?? 1000,
      max_mana: m.max_mana || 100,
      mana: m.current_mana ?? m.max_mana ?? 100,
      grid_x: 0,
      grid_y: idx,
      rank: m.rank,
      avatar_url: m.avatar_url,
    })),
    ...teamBMembers.map((m: any, idx: number) => ({
      id: m.player_id,
      name: m.username,
      kind: "enemy" as const,
      max_hp: m.max_hp || 1000,
      hp: m.current_hp ?? m.max_hp ?? 1000,
      max_mana: m.max_mana || 100,
      mana: m.current_mana ?? m.max_mana ?? 100,
      grid_x: 4,
      grid_y: idx,
      rank: m.rank,
      avatar_url: m.avatar_url,
    })),
  ];

  const winnerKind = isTeamAWinner ? "ally" : isTeamBWinner ? "enemy" : "draw";

  // Group logs by turn for text log view
  const turns = Array.from(new Set(logs.map((l) => l.turn))).sort((a, b) => a - b);
  const filteredLogs = selectedTurn === "all" ? logs : logs.filter((l) => l.turn === selectedTurn);

  const getRoundTitle = (round: number) => {
    return `Chiến Báo Trận Đấu - Vòng ${round}`;
  };

  return (
    <GameModal
      open={matchId !== null}
      onClose={onClose}
      size="5xl"
      title={match ? getRoundTitle(match.round_number) : "Chiến Báo Trận Đấu"}
    >
      {loading ? (
        <div className="py-12 text-center text-gray-400 animate-pulse">
          <Swords className="w-8 h-8 mx-auto mb-2 animate-spin text-amber-500" />
          Đang tải dữ liệu chiến báo...
        </div>
      ) : !match ? (
        <div className="py-8 text-center text-red-400">
          Không tìm thấy thông tin trận đấu này.
        </div>
      ) : (
        <div className="space-y-4">
          {/* Match Teams Header */}
          <div className="flex items-center justify-between bg-black/40 p-4 rounded-xl border border-zinc-800">
            {/* Team A */}
            <div className={`flex-1 p-3 rounded-lg ${isTeamAWinner ? "bg-emerald-950/40 border border-emerald-500/50" : "bg-zinc-900/40"}`}>
              <div className="flex items-center gap-2 mb-1">
                {isTeamAWinner && <Trophy className="w-5 h-5 text-amber-400 shrink-0" />}
                <span className="font-bold text-base text-zinc-100 truncate">{match.team_a_name}</span>
                {isTeamAWinner && <span className="text-xs px-2 py-0.5 rounded bg-emerald-600/60 text-emerald-200">Thắng</span>}
              </div>
              <div className="text-xs text-zinc-400">
                {teamAMembers.length} thành viên • {teamAMembers.filter((m: any) => !m.eliminated).length} còn sống
              </div>
            </div>

            {/* VS divider */}
            <div className="px-4 text-center">
              <div className="text-xs font-bold text-amber-500/80 uppercase tracking-wider">VS</div>
              <div className="text-[10px] text-zinc-500 mt-0.5">Vị trí #{match.bracket_position}</div>
            </div>

            {/* Team B */}
            <div className={`flex-1 p-3 rounded-lg text-right ${isTeamBWinner ? "bg-emerald-950/40 border border-emerald-500/50" : "bg-zinc-900/40"}`}>
              <div className="flex items-center justify-end gap-2 mb-1">
                {isTeamBWinner && <span className="text-xs px-2 py-0.5 rounded bg-emerald-600/60 text-emerald-200">Thắng</span>}
                <span className="font-bold text-base text-zinc-100 truncate">{match.team_b_name || "BYE"}</span>
                {isTeamBWinner && <Trophy className="w-5 h-5 text-amber-400 shrink-0" />}
              </div>
              <div className="text-xs text-zinc-400">
                {teamBMembers.length > 0 ? `${teamBMembers.length} thành viên • ${teamBMembers.filter((m: any) => !m.eliminated).length} còn sống` : "Đối thủ trống (BYE)"}
              </div>
            </div>
          </div>

          {/* View mode toggle tabs */}
          {logs.length > 0 && (
            <div className="flex gap-2 border-b border-zinc-800 pb-2">
              <button
                onClick={() => setViewMode("replay")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  viewMode === "replay"
                    ? "bg-amber-500 text-black font-semibold"
                    : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
                }`}
              >
                <Play className="w-3.5 h-3.5" /> Mô Phỏng Trận Đấu
              </button>
              <button
                onClick={() => setViewMode("logs")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  viewMode === "logs"
                    ? "bg-amber-500 text-black font-semibold"
                    : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
                }`}
              >
                <ScrollText className="w-3.5 h-3.5" /> Nhật Ký Diễn Biến ({logs.length})
              </button>
              <button
                onClick={() => setViewMode("summary")}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  viewMode === "summary"
                    ? "bg-amber-500 text-black font-semibold"
                    : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
                }`}
              >
                <Users className="w-3.5 h-3.5" /> Đội Hình Ra Trận
              </button>
            </div>
          )}

          {/* Mode 1: Interactive Replay */}
          {viewMode === "replay" && logs.length > 0 && (
            <div className="max-h-[68vh] overflow-y-auto rounded-lg border border-zinc-800 p-2 bg-black/60">
              <CombatReplayView
                logs={logs}
                combatants={replayCombatants}
                winner={winnerKind}
                onBack={() => setViewMode("summary")}
              />
            </div>
          )}

          {/* Mode 2: Detailed Text Log */}
          {viewMode === "logs" && logs.length > 0 && (
            <div className="space-y-3">
              {/* Turn filter */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-zinc-400">Lọc theo hiệp:</span>
                <button
                  onClick={() => setSelectedTurn("all")}
                  className={`text-xs px-2.5 py-1 rounded ${
                    selectedTurn === "all" ? "bg-amber-500 text-black font-bold" : "bg-zinc-800 text-zinc-300"
                  }`}
                >
                  Tất cả ({turns.length} hiệp)
                </button>
                {turns.map((turn) => (
                  <button
                    key={turn}
                    onClick={() => setSelectedTurn(turn)}
                    className={`text-xs px-2 py-1 rounded ${
                      selectedTurn === turn ? "bg-amber-500 text-black font-bold" : "bg-zinc-800 text-zinc-300 hover:bg-zinc-700"
                    }`}
                  >
                    Hiệp {turn}
                  </button>
                ))}
              </div>

              {/* Logs container */}
              <div className="max-h-[55vh] overflow-y-auto space-y-1.5 p-3 rounded-lg bg-black/50 border border-zinc-800 font-mono text-xs">
                {filteredLogs.map((entry, idx) => (
                  <div key={idx} className="flex gap-2 py-1 border-b border-zinc-900/60 last:border-b-0 items-start">
                    <span className="text-zinc-500 font-bold shrink-0 w-16">[H{entry.turn}]</span>
                    <span className="font-semibold text-amber-400 shrink-0">{entry.actor_name}:</span>
                    <span className="text-zinc-200 flex-1 leading-relaxed">
                      {entry.detail || `${entry.action} lên mục tiêu`}
                      {entry.damage ? (
                        <span className={`ml-1 font-bold ${entry.is_crit ? "text-amber-400" : "text-rose-400"}`}>
                          (-{Math.round(entry.damage).toLocaleString("vi-VN")}{entry.is_crit ? " BẠO KÍCH!" : ""})
                        </span>
                      ) : null}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Mode 3: Team Roster Summary (also fallback when no logs) */}
          {(viewMode === "summary" || logs.length === 0) && (
            <div className="space-y-4">
              {logs.length === 0 && (
                <div className="p-3 bg-amber-950/20 border border-amber-800/40 rounded-lg text-xs text-amber-300 text-center">
                  Trận đấu này không có dữ liệu chi tiết từng lượt đánh (trận đấu BYE hoặc kết thúc trước phiên bản lưu trữ chiến báo).
                </div>
              )}

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Team A Roster */}
                <div className="space-y-2">
                  <h4 className="font-bold text-sm text-zinc-300 flex items-center gap-1.5">
                    <Shield className="w-4 h-4 text-emerald-400" />
                    Thành viên {match.team_a_name} ({teamAMembers.length})
                  </h4>
                  <div className="space-y-1.5">
                    {teamAMembers.map((m: any) => (
                      <div
                        key={m.id}
                        className={`p-2.5 rounded-lg flex justify-between items-center text-xs border ${
                          m.eliminated
                            ? "bg-zinc-950/50 border-zinc-900 text-zinc-500"
                            : "bg-zinc-900/70 border-zinc-800 text-zinc-200"
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-zinc-100">{m.username}</span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400">
                            Rank {m.rank}
                          </span>
                        </div>
                        <div className="flex items-center gap-3">
                          <span className="text-[11px] text-zinc-400 flex items-center gap-1">
                            <Heart className="w-3 h-3 text-rose-400" />
                            {Math.round(m.current_hp || 0)} / {Math.round(m.max_hp || 0)}
                          </span>
                          {m.eliminated ? (
                            <span className="text-[10px] text-rose-400 font-bold">Hạ gục</span>
                          ) : (
                            <span className="text-[10px] text-emerald-400 font-bold">Sống sót</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Team B Roster */}
                <div className="space-y-2">
                  <h4 className="font-bold text-sm text-zinc-300 flex items-center gap-1.5">
                    <Shield className="w-4 h-4 text-rose-400" />
                    Thành viên {match.team_b_name || "BYE"} ({teamBMembers.length})
                  </h4>
                  {teamBMembers.length === 0 ? (
                    <div className="p-4 text-center text-xs text-zinc-500 bg-zinc-950/30 rounded-lg border border-zinc-900">
                      Không có đối thủ (Nhận vé BYE vào thẳng vòng trong).
                    </div>
                  ) : (
                    <div className="space-y-1.5">
                      {teamBMembers.map((m: any) => (
                        <div
                          key={m.id}
                          className={`p-2.5 rounded-lg flex justify-between items-center text-xs border ${
                            m.eliminated
                              ? "bg-zinc-950/50 border-zinc-900 text-zinc-500"
                              : "bg-zinc-900/70 border-zinc-800 text-zinc-200"
                          }`}
                        >
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-zinc-100">{m.username}</span>
                            <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400">
                              Rank {m.rank}
                            </span>
                          </div>
                          <div className="flex items-center gap-3">
                            <span className="text-[11px] text-zinc-400 flex items-center gap-1">
                              <Heart className="w-3 h-3 text-rose-400" />
                              {Math.round(m.current_hp || 0)} / {Math.round(m.max_hp || 0)}
                            </span>
                            {m.eliminated ? (
                              <span className="text-[10px] text-rose-400 font-bold">Hạ gục</span>
                            ) : (
                              <span className="text-[10px] text-emerald-400 font-bold">Sống sót</span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </GameModal>
  );
}
