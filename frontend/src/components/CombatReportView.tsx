"use client";

import React, { useState, useMemo } from "react";
import { Swords, Trophy, Shield, Heart, ScrollText, Users, Play, ArrowLeft } from "lucide-react";
import { CombatReplayView, type CombatLogEntry } from "./CombatReplayView";
import { FormattedText } from "./FormattedText";
import { Num } from "./Num";
import { guSpecFor, useGameStore } from "@/store/useGameStore";

export interface CombatantInfo {
  id: string;
  name: string;
  kind: "ally" | "enemy";
  max_hp?: number;
  hp?: number;
  maxHp?: number;
  max_mana?: number;
  mana?: number;
  rank?: number;
  sub_rank?: number;
  avatar_url?: string;
  eliminated?: boolean;
  grid_x?: number;
  grid_y?: number;
  gu_inventory?: Array<Record<string, unknown>>;
  guInventory?: Array<Record<string, unknown>>;
}

export function extractCombatantsFromLogs(
  logs: CombatLogEntry[],
  playerName?: string
): CombatantInfo[] {
  // 1. Check snapshot in BATTLE_START or any log
  for (const log of logs) {
    const snap = (log as unknown as Record<string, unknown>).snapshot as
      | { allies?: Record<string, unknown>[]; enemies?: Record<string, unknown>[] }
      | undefined;
    if (snap && (snap.allies || snap.enemies)) {
      const result: CombatantInfo[] = [];
      if (Array.isArray(snap.allies)) {
        snap.allies.forEach((a, idx) => {
          result.push({
            id: String(a.id || `ally_${idx}`),
            name: String(a.name || "Phe Ta"),
            kind: "ally",
            max_hp: Number(a.max_hp || a.maxHp || 1000),
            hp: Number(a.hp ?? a.max_hp ?? 1000),
            max_mana: Number(a.max_mana || a.maxMana || 100),
            mana: Number(a.mana ?? a.current_mana ?? 100),
            rank: Number(a.rank || 1),
            grid_x: Number(a.grid_x && Number(a.grid_x) >= 1 && Number(a.grid_x) <= 3 ? a.grid_x : (idx % 3) + 1),
            grid_y: Number(a.grid_y && Number(a.grid_y) >= 1 && Number(a.grid_y) <= 3 ? a.grid_y : Math.floor(idx / 3) + 1),
            avatar_url: (a.avatar_url as string) || undefined,
          });
        });
      }
      if (Array.isArray(snap.enemies)) {
        snap.enemies.forEach((e, idx) => {
          result.push({
            id: String(e.id || `enemy_${idx}`),
            name: String(e.name || "Kẻ Địch"),
            kind: "enemy",
            max_hp: Number(e.max_hp || e.maxHp || 1000),
            hp: Number(e.hp ?? e.max_hp ?? 1000),
            max_mana: Number(e.max_mana || e.maxMana || 100),
            mana: Number(e.mana ?? e.current_mana ?? 100),
            rank: Number(e.rank || 1),
            grid_x: Number(e.grid_x && Number(e.grid_x) >= 1 && Number(e.grid_x) <= 3 ? e.grid_x : (idx % 3) + 1),
            grid_y: Number(e.grid_y && Number(e.grid_y) >= 1 && Number(e.grid_y) <= 3 ? e.grid_y : Math.floor(idx / 3) + 1),
            avatar_url: (e.avatar_url as string) || undefined,
          });
        });
      }
      if (result.length > 0) return result;
    }
  }

  // 2. Fallback: Parse actor/target IDs from logs
  const map: Record<string, CombatantInfo> = {};
  let allyIdx = 0;
  let enemyIdx = 0;

  logs.forEach((log) => {
    if (log.actor_id && !map[log.actor_id]) {
      const isAlly = Boolean(playerName && (log.actor_name === playerName || log.actor_id === playerName));
      map[log.actor_id] = {
        id: log.actor_id,
        name: log.actor_name || log.actor_id,
        kind: isAlly ? "ally" : "enemy",
        max_hp: 1000,
        hp: 1000,
        max_mana: 100,
        mana: 100,
        rank: 1,
        grid_x: isAlly ? ((allyIdx % 3) + 1) : ((enemyIdx % 3) + 1),
        grid_y: isAlly ? (Math.floor(allyIdx++ / 3) + 1) : (Math.floor(enemyIdx++ / 3) + 1),
      };
    }
    if (log.target_id && !map[log.target_id]) {
      const isAlly = Boolean(playerName && (log.target_name === playerName || log.target_id === playerName));
      map[log.target_id] = {
        id: log.target_id,
        name: log.target_name || log.target_id,
        kind: isAlly ? "ally" : "enemy",
        max_hp: 1000,
        hp: 1000,
        max_mana: 100,
        mana: 100,
        rank: 1,
        grid_x: isAlly ? ((allyIdx % 3) + 1) : ((enemyIdx % 3) + 1),
        grid_y: isAlly ? (Math.floor(allyIdx++ / 3) + 1) : (Math.floor(enemyIdx++ / 3) + 1),
      };
    }
  });

  return Object.values(map);
}

export interface CombatReportViewProps {
  logs: CombatLogEntry[];
  combatants?: CombatantInfo[];
  winner: string; // "Phe Ta" | "Phe Địch" | "ally" | "enemy" | "draw" | team_id
  onBack: () => void;
  title?: string;
  teamAName?: string;
  teamBName?: string;
  playerName?: string;
  apertureHp?: number;
  apertureMaxHp?: number;
  extraHeaderInfo?: React.ReactNode;
}

export const CombatReportView: React.FC<CombatReportViewProps> = ({
  logs = [],
  combatants = [],
  winner,
  onBack,
  title = "Chiến Báo Trận Đấu",
  teamAName,
  teamBName,
  playerName,
  apertureHp,
  apertureMaxHp,
  extraHeaderInfo,
}) => {
  const [viewMode, setViewMode] = useState<"replay" | "logs" | "summary">(
    logs.length > 0 ? "replay" : "summary"
  );
  const [selectedTurn, setSelectedTurn] = useState<number | "all">("all");

  const effectiveCombatants = useMemo(() => {
    if (combatants && combatants.length > 0) {
      return combatants;
    }
    return extractCombatantsFromLogs(logs, playerName);
  }, [combatants, logs, playerName]);

  const guDictionary = useGameStore((state) => state.guDictionary || {});

  // Separate allies & enemies cleanly
  const allies = useMemo(() => {
    const list = effectiveCombatants.filter((c) => c.kind === "ally");
    if (list.length > 0) return list;
    if (playerName) {
      return effectiveCombatants.filter((c) => c.name === playerName || c.id === playerName);
    }
    return [];
  }, [effectiveCombatants, playerName]);

  const enemies = useMemo(() => {
    const allySet = new Set(allies.map((a) => a.id));
    return effectiveCombatants.filter((c) => !allySet.has(c.id) && (c.kind === "enemy" || allies.length === 0));
  }, [effectiveCombatants, allies]);

  // Derive names if not passed
  const displayTeamA =
    teamAName ||
    (allies.length > 0
      ? allies.length === 1
        ? allies[0].name
        : playerName || allies[0].name
      : playerName || "Phe Ta");

  const displayTeamB =
    teamBName ||
    (enemies.length > 0
      ? enemies.length === 1
        ? enemies[0].name
        : "Phe Địch"
      : "Kẻ Địch");

  // Winner logic
  const isAllyWinner =
    winner === "Phe Ta" ||
    winner === "ally" ||
    (teamAName && winner === teamAName);

  const isEnemyWinner =
    winner === "Phe Địch" ||
    winner === "enemy" ||
    (teamBName && winner === teamBName);

  // Turns logic
  const turns = useMemo(() => {
    const set = new Set<number>();
    logs.forEach((l) => {
      if (l.turn && l.turn > 0) set.add(l.turn);
    });
    return Array.from(set).sort((a, b) => a - b);
  }, [logs]);

  const filteredLogs = useMemo(() => {
    if (selectedTurn === "all") return logs;
    return logs.filter((l) => l.turn === selectedTurn);
  }, [logs, selectedTurn]);

  // Count alive
  const aliveAllies = allies.filter((a) => {
    if (typeof a.eliminated === "boolean") return !a.eliminated;
    const hp = a.hp ?? 1;
    return hp > 0;
  }).length;

  const aliveEnemies = enemies.filter((e) => {
    if (typeof e.eliminated === "boolean") return !e.eliminated;
    const hp = e.hp ?? 1;
    return hp > 0;
  }).length;

  return (
    <div className="w-full mx-auto space-y-4">
      {/* Top Bar with Back Button & Title */}
      <div className="flex items-center justify-between bg-black/60 p-3 rounded-xl border border-zinc-800 flex-wrap gap-2">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="p-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg transition-colors flex items-center gap-1 text-xs font-semibold"
          >
            <ArrowLeft className="w-4 h-4" /> Quay lại
          </button>
          <h2 className="text-base font-bold text-zinc-100 flex items-center gap-2">
            <Swords className="w-5 h-5 text-amber-500" />
            {title}
          </h2>
        </div>
        {extraHeaderInfo}
      </div>

      {/* Matchup Header Card */}
      <div className="flex items-center justify-between bg-black/40 p-4 rounded-xl border border-zinc-800 gap-4 flex-col sm:flex-row">
        {/* Team A / Ally */}
        <div
          className={`w-full sm:flex-1 p-3 rounded-lg transition-all ${
            isAllyWinner
              ? "bg-emerald-950/40 border border-emerald-500/50"
              : "bg-zinc-900/40 border border-zinc-800/60"
          }`}
        >
          <div className="flex items-center gap-2 mb-1">
            {isAllyWinner && <Trophy className="w-5 h-5 text-amber-400 shrink-0" />}
            <span className="font-bold text-base text-zinc-100 truncate">
              {displayTeamA}
            </span>
            {isAllyWinner && (
              <span className="text-xs px-2 py-0.5 rounded bg-emerald-600/60 text-emerald-200 font-bold">
                Thắng
              </span>
            )}
          </div>
          <div className="text-xs text-zinc-400">
            {allies.length > 0
              ? `${allies.length} thành viên • ${aliveAllies} còn sống`
              : "Phe Ta"}
          </div>
        </div>

        {/* VS Divider */}
        <div className="px-2 text-center shrink-0">
          <div className="text-xs font-bold text-amber-500/80 uppercase tracking-widest">
            VS
          </div>
        </div>

        {/* Team B / Enemy */}
        <div
          className={`w-full sm:flex-1 p-3 rounded-lg text-right transition-all ${
            isEnemyWinner
              ? "bg-emerald-950/40 border border-emerald-500/50"
              : "bg-zinc-900/40 border border-zinc-800/60"
          }`}
        >
          <div className="flex items-center justify-end gap-2 mb-1">
            {isEnemyWinner && (
              <span className="text-xs px-2 py-0.5 rounded bg-emerald-600/60 text-emerald-200 font-bold">
                Thắng
              </span>
            )}
            <span className="font-bold text-base text-zinc-100 truncate">
              {displayTeamB}
            </span>
            {isEnemyWinner && <Trophy className="w-5 h-5 text-amber-400 shrink-0" />}
          </div>
          <div className="text-xs text-zinc-400">
            {enemies.length > 0
              ? `${enemies.length} mục tiêu • ${aliveEnemies} còn sống`
              : "Kẻ Địch"}
          </div>
        </div>
      </div>

      {/* View Mode Toggle Tabs */}
      <div className="flex gap-2 border-b border-zinc-800 pb-2 overflow-x-auto">
        {logs.length > 0 && (
          <button
            onClick={() => setViewMode("replay")}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium transition ${
              viewMode === "replay"
                ? "bg-amber-500 text-black font-semibold shadow-md"
                : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
            }`}
          >
            <Play className="w-3.5 h-3.5" /> Mô Phỏng Trận Đấu
          </button>
        )}
        {logs.length > 0 && (
          <button
            onClick={() => setViewMode("logs")}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium transition ${
              viewMode === "logs"
                ? "bg-amber-500 text-black font-semibold shadow-md"
                : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
            }`}
          >
            <ScrollText className="w-3.5 h-3.5" /> Nhật Ký Diễn Biến ({logs.length})
          </button>
        )}
        <button
          onClick={() => setViewMode("summary")}
          className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium transition ${
            viewMode === "summary"
              ? "bg-amber-500 text-black font-semibold shadow-md"
              : "bg-zinc-800/80 text-zinc-300 hover:bg-zinc-700"
          }`}
        >
          <Users className="w-3.5 h-3.5" /> Đội Hình Ra Trận ({effectiveCombatants.length})
        </button>
      </div>

      {/* Tab Mode 1: Interactive Grid Replay */}
      {viewMode === "replay" && logs.length > 0 && (
        <div className="rounded-xl border border-zinc-800 p-2 bg-black/60">
          <CombatReplayView
            logs={logs}
            combatants={effectiveCombatants as unknown as Parameters<typeof CombatReplayView>[0]["combatants"]}
            winner={winner}
            onBack={onBack}
            playerName={playerName}
            apertureHp={apertureHp}
            apertureMaxHp={apertureMaxHp}
          />
        </div>
      )}

      {/* Tab Mode 2: Detailed Turn-by-Turn Text Log */}
      {viewMode === "logs" && logs.length > 0 && (
        <div className="space-y-3">
          {/* Turn Filter Buttons */}
          <div className="flex items-center gap-1.5 flex-wrap bg-black/40 p-3 rounded-xl border border-zinc-800">
            <span className="text-xs text-zinc-400 font-medium mr-1">
              Lọc theo hiệp:
            </span>
            <button
              onClick={() => setSelectedTurn("all")}
              className={`text-xs px-2.5 py-1 rounded-md transition ${
                selectedTurn === "all"
                  ? "bg-amber-500 text-black font-bold"
                  : "bg-zinc-800 text-zinc-300 hover:bg-zinc-700"
              }`}
            >
              Tất cả ({turns.length} hiệp)
            </button>
            {turns.map((turn) => (
              <button
                key={turn}
                onClick={() => setSelectedTurn(turn)}
                className={`text-xs px-2.5 py-1 rounded-md transition ${
                  selectedTurn === turn
                    ? "bg-amber-500 text-black font-bold"
                    : "bg-zinc-800 text-zinc-300 hover:bg-zinc-700"
                }`}
              >
                Hiệp {turn}
              </button>
            ))}
          </div>

          {/* Log Entry List */}
          <div className="max-h-[60vh] overflow-y-auto space-y-1.5 p-3 rounded-xl bg-black/60 border border-zinc-800 font-mono text-xs">
            {filteredLogs.map((entry, idx) => {
              const isTurnStart = entry.action === "TURN_START";
              if (isTurnStart) {
                return (
                  <div key={idx} className="flex items-center gap-2 py-2">
                    <div className="flex-1 h-px bg-zinc-800" />
                    <span className="text-[10px] text-amber-500 font-bold bg-amber-950/40 px-3 py-0.5 rounded-full border border-amber-800/40">
                      LƯỢT {entry.turn}
                    </span>
                    <div className="flex-1 h-px bg-zinc-800" />
                  </div>
                );
              }

              return (
                <div
                  key={idx}
                  className="flex gap-2 py-1.5 px-2 rounded-lg hover:bg-zinc-900/40 border-b border-zinc-900/60 last:border-b-0 items-start transition-colors"
                >
                  <span className="text-zinc-500 font-bold shrink-0 w-14">
                    [H{entry.turn}]
                  </span>
                  {entry.actor_name && (
                    <span className="font-semibold text-amber-400 shrink-0">
                      {entry.actor_name}:
                    </span>
                  )}
                  <span className="text-zinc-200 flex-1 leading-relaxed">
                    <FormattedText
                      text={entry.detail || `${entry.action} lên mục tiêu`}
                    />
                    {entry.damage && entry.damage > 0 ? (
                      <span
                        className={`ml-1.5 font-bold ${
                          entry.is_crit ? "text-amber-400" : "text-rose-400"
                        }`}
                      >
                        (-<Num value={Math.round(entry.damage)} />
                        {entry.is_crit ? " BẠO KÍCH!" : ""})
                      </span>
                    ) : null}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Tab Mode 3: Roster Summary */}
      {(viewMode === "summary" || logs.length === 0) && (
        <div className="space-y-4">
          {logs.length === 0 && (
            <div className="p-3 bg-amber-950/20 border border-amber-800/40 rounded-lg text-xs text-amber-300 text-center">
              Trận đấu này không có dữ liệu chi tiết từng lượt đánh.
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Ally Roster */}
            <div className="space-y-2">
              <h4 className="font-bold text-sm text-zinc-300 flex items-center gap-1.5">
                <Shield className="w-4 h-4 text-emerald-400" />
                {displayTeamA} ({allies.length})
              </h4>
              <div className="space-y-1.5">
                {allies.length === 0 ? (
                  <div className="p-3 text-center text-xs text-zinc-500 bg-zinc-950/40 rounded-lg border border-zinc-900">
                    Không có thông tin thành viên.
                  </div>
                ) : (
                  allies.map((m, idx) => {
                    const hp = m.hp ?? m.max_hp ?? m.maxHp ?? 100;
                    const maxHp = m.max_hp ?? m.maxHp ?? 100;
                    const isDead =
                      typeof m.eliminated === "boolean"
                        ? m.eliminated
                        : hp <= 0;
                    const rawGus = m.gu_inventory || m.guInventory;
                    const mGus = Array.isArray(rawGus)
                      ? rawGus.filter((g: Record<string, unknown>) => g && g.equipped !== false && g.status !== "DORMANT" && g.status !== "DEAD")
                      : [];

                    return (
                      <div
                        key={m.id || idx}
                        className={`p-3 rounded-xl flex flex-col gap-1.5 text-xs border ${
                          isDead
                            ? "bg-zinc-950/50 border-zinc-900 text-zinc-500"
                            : "bg-zinc-900/60 border-zinc-800 text-zinc-200"
                        }`}
                      >
                        <div className="flex justify-between items-center w-full">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-zinc-100">
                              {m.name}
                            </span>
                            {m.rank && (
                              <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400 font-mono">
                                Cảnh giới {m.rank}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3">
                            <span className="text-[11px] text-zinc-400 flex items-center gap-1 font-mono">
                              <Heart className="w-3 h-3 text-rose-400" />
                              <Num value={Math.round(hp)} /> / <Num value={Math.round(maxHp)} />
                            </span>
                            {isDead ? (
                              <span className="text-[10px] text-rose-400 font-bold bg-rose-950/40 px-2 py-0.5 rounded border border-rose-900/50">
                                Hạ gục
                              </span>
                            ) : (
                              <span className="text-[10px] text-emerald-400 font-bold bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-900/50">
                                Sống sót
                              </span>
                            )}
                          </div>
                        </div>
                        {mGus.length > 0 && (
                          <div className="pt-1.5 border-t border-zinc-800/60 flex flex-wrap gap-1">
                            {mGus.map((gu: Record<string, unknown>, gIdx: number) => {
                              const guId = String(gu.gu_id || gu.guId || gu.id || "");
                              const rank = Number(gu.rank || gu.gu_rank || 1);
                              const spec = guSpecFor(guId, rank);
                              const name = spec?.name || guId || "Cổ trùng";
                              const isVital = Boolean(gu.is_vital || gu.isVital);
                              return (
                                <span
                                  key={gIdx}
                                  className={`text-[10px] px-1.5 py-0.5 rounded border flex items-center gap-1 ${
                                    isVital
                                      ? "bg-amber-950/40 border-amber-600/50 text-amber-300 font-semibold"
                                      : "bg-zinc-800/80 border-zinc-700/60 text-zinc-300"
                                  }`}
                                  title={`${name} (Hạng ${rank})`}
                                >
                                  {name}
                                  <span className="text-[9px] opacity-75 font-mono">H{rank}</span>
                                </span>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            {/* Enemy Roster */}
            <div className="space-y-2">
              <h4 className="font-bold text-sm text-zinc-300 flex items-center gap-1.5">
                <Shield className="w-4 h-4 text-rose-400" />
                {displayTeamB} ({enemies.length})
              </h4>
              <div className="space-y-1.5">
                {enemies.length === 0 ? (
                  <div className="p-3 text-center text-xs text-zinc-500 bg-zinc-950/40 rounded-lg border border-zinc-900">
                    Không có thông tin kẻ địch.
                  </div>
                ) : (
                  enemies.map((m, idx) => {
                    const hp = m.hp ?? m.max_hp ?? m.maxHp ?? 100;
                    const maxHp = m.max_hp ?? m.maxHp ?? 100;
                    const isDead =
                      typeof m.eliminated === "boolean"
                        ? m.eliminated
                        : hp <= 0;
                    const rawGus = m.gu_inventory || m.guInventory;
                    const mGus = Array.isArray(rawGus)
                      ? rawGus.filter((g: Record<string, unknown>) => g && g.equipped !== false && g.status !== "DORMANT" && g.status !== "DEAD")
                      : [];

                    return (
                      <div
                        key={m.id || idx}
                        className={`p-3 rounded-xl flex flex-col gap-1.5 text-xs border ${
                          isDead
                            ? "bg-zinc-950/50 border-zinc-900 text-zinc-500"
                            : "bg-zinc-900/60 border-zinc-800 text-zinc-200"
                        }`}
                      >
                        <div className="flex justify-between items-center w-full">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-zinc-100">
                              {m.name}
                            </span>
                            {m.rank && (
                              <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400 font-mono">
                                Cảnh giới {m.rank}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3">
                            <span className="text-[11px] text-zinc-400 flex items-center gap-1 font-mono">
                              <Heart className="w-3 h-3 text-rose-400" />
                              <Num value={Math.round(hp)} /> / <Num value={Math.round(maxHp)} />
                            </span>
                            {isDead ? (
                              <span className="text-[10px] text-rose-400 font-bold bg-rose-950/40 px-2 py-0.5 rounded border border-rose-900/50">
                                Hạ gục
                              </span>
                            ) : (
                              <span className="text-[10px] text-emerald-400 font-bold bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-900/50">
                                Sống sót
                              </span>
                            )}
                          </div>
                        </div>
                        {mGus.length > 0 && (
                          <div className="pt-1.5 border-t border-zinc-800/60 flex flex-wrap gap-1">
                            {mGus.map((gu: Record<string, unknown>, gIdx: number) => {
                              const guId = String(gu.gu_id || gu.guId || gu.id || "");
                              const rank = Number(gu.rank || gu.gu_rank || 1);
                              const spec = guSpecFor(guId, rank);
                              const name = spec?.name || guId || "Cổ trùng";
                              const isVital = Boolean(gu.is_vital || gu.isVital);
                              return (
                                <span
                                  key={gIdx}
                                  className={`text-[10px] px-1.5 py-0.5 rounded border flex items-center gap-1 ${
                                    isVital
                                      ? "bg-amber-950/40 border-amber-600/50 text-amber-300 font-semibold"
                                      : "bg-zinc-800/80 border-zinc-700/60 text-zinc-300"
                                  }`}
                                  title={`${name} (Hạng ${rank})`}
                                >
                                  {name}
                                  <span className="text-[9px] opacity-75 font-mono">H{rank}</span>
                                </span>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
