import React, { useState, useMemo } from "react";
import { Swords, Trophy, Shield, Heart, ScrollText, Users, Play, ArrowLeft, ChevronDown, ChevronUp, Sparkles, Calculator, ArrowRight, Zap, TrendingUp, ShieldAlert } from "lucide-react";
import { CombatReplayView, type CombatLogEntry, type StatProvenance } from "./CombatReplayView";
import { FormattedText } from "./FormattedText";
import { Num } from "./Num";
import { formatNumber } from "@/lib/utils";
import { guSpecFor, useGameStore } from "@/store/useGameStore";

// StatProvenanceRows renders one "base → final" stat row plus the itemized
// status modifiers (e.g. Trọng Thương −50%) for attacker/defender provenance.
function StatProvenanceRows({ prov, label }: { prov: StatProvenance; label: string }) {
  const mods = prov.mods ?? [];
  if (mods.length === 0) return null;
  return (
    <div className="pt-1.5 border-t border-zinc-800/50 space-y-1">
      <div className="flex justify-between items-center">
        <span>{label}:</span>
        <span className="text-zinc-100 font-bold">
          <Num value={prov.base_value ?? 0} />
          <span className="text-zinc-500 mx-1">→</span>
          <Num value={prov.final_value ?? 0} />
        </span>
      </div>
      {mods.map((m) => (
        <div key={m.key} className="flex justify-between items-center text-[10px]">
          <span className="text-rose-300/90">{m.name}:</span>
          <span className={`font-bold ${m.pct < 0 ? "text-rose-400" : "text-emerald-400"}`}>
            {m.pct > 0 ? "+" : ""}{(m.pct * 100).toFixed(0)}%
          </span>
        </div>
      ))}
    </div>
  );
}

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
  const [expandedLogIdx, setExpandedLogIdx] = useState<number | null>(null);

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

              const hasBreakdown = !!entry.breakdown;
              const isExpanded = expandedLogIdx === idx;
              const bLog = entry.breakdown;
              const isShieldLog = (bLog?.shield_final ?? 0) > 0 || (bLog?.shield_base ?? 0) > 0;
              const isHealLog = (bLog?.healing_final ?? 0) > 0 || (bLog?.healing_base ?? 0) > 0;
              const borderExpandedColor = isShieldLog
                ? "border-cyan-500/40"
                : isHealLog
                ? "border-emerald-500/40"
                : "border-amber-500/40";
              const breakdownTitle = isShieldLog
                ? "Xem chi tiết tính toán khiên ảo"
                : isHealLog
                ? "Xem chi tiết tính toán hồi phục"
                : "Xem chi tiết tính toán sát thương";
              const chevronColor = isShieldLog
                ? "text-cyan-400"
                : isHealLog
                ? "text-emerald-400"
                : "text-amber-400";

              return (
                <div
                  key={idx}
                  className={`rounded-lg transition-all border ${
                    isExpanded
                      ? `bg-zinc-900/80 ${borderExpandedColor} p-2 my-1`
                      : "hover:bg-zinc-900/40 border-transparent border-b-zinc-900/60 last:border-b-0 py-1.5 px-2"
                  }`}
                >
                  <div
                    className={`flex gap-2 items-start ${hasBreakdown ? "cursor-pointer select-none" : ""}`}
                    onClick={() => {
                      if (hasBreakdown) {
                        setExpandedLogIdx(isExpanded ? null : idx);
                      }
                    }}
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
                      {entry.damage && entry.damage > 0 && !entry.detail?.includes("gây ") && !entry.breakdown?.healing_base && !entry.breakdown?.shield_base && entry.action !== "SHIELD_GAIN" ? (
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
                    {hasBreakdown && (
                      <button
                        type="button"
                        className="text-zinc-500 hover:text-amber-400 shrink-0 p-0.5 rounded transition mt-0.5"
                        title={breakdownTitle}
                      >
                        {isExpanded ? (
                          <ChevronUp className={`w-3.5 h-3.5 ${chevronColor}`} />
                        ) : (
                          <ChevronDown className="w-3.5 h-3.5 text-zinc-500 hover:text-zinc-300" />
                        )}
                      </button>
                    )}
                  </div>

                  {/* Expandable Breakdown Card */}
                  {isExpanded && entry.breakdown && (() => {
                    const b = entry.breakdown;
                    const baseDmg = b.base_damage || 0;
                    const skillMult = b.skill_mult || 1;
                    const daoMult = b.dao_mult || 1;
                    const buffMult = b.buff_mult || 1;
                    const critMult = b.crit_mult || 1;
                    const pathBonus = b.path_bonus || 0;

                    // 1. Raw Output Damage
                    const rawOutputDmg = baseDmg * skillMult * daoMult * buffMult * critMult * (1 + pathBonus);

                    // 2. Defense Reduction (Trừ giáp)
                    const targetDef = b.target_def || 0;
                    const afterDefDmg = Math.max(1, rawOutputDmg - targetDef);

                    // 3. Path Resistance & Damage Reduction (Kháng hệ & Giảm ST)
                    const pathResist = b.path_resist || 0;
                    const dmgReduction = b.damage_reduction || 0;
                    const finalDmg = b.final_damage || Math.max(1, afterDefDmg * (1 - pathResist) * (1 - dmgReduction));

                    // 4. Shield Absorbed & HP Lost
                    const shieldAbsorbed = b.shield_absorbed || 0;
                    const hpLost = b.hp_damage || Math.max(0, finalDmg - shieldAbsorbed);

                    // Heal breakdown variant (skill heals / item heals)
                    const healBase = b.healing_base ?? 0;
                    const healDao = b.healing_dao_bonus ?? 0;
                    const healFinal = b.healing_final ?? 0;
                    const healPct = b.healer_healing?.final_value ?? 0;
                    const isHeal = healFinal > 0 || healBase > 0;

                    // Shield breakdown variant (skill shields / shield buffs)
                    const shieldBase = b.shield_base ?? 0;
                    const shieldSkillMult = b.shield_skill_mult ?? 1;
                    const shieldBuffMult = b.shield_buff_mult ?? 1;
                    const shieldDaoMult = b.shield_dao_mult ?? 1;
                    const shieldDaoBonus = b.shield_dao_bonus ?? 0;
                    const shieldFinal = b.shield_final ?? (shieldBase * (shieldSkillMult + shieldBuffMult - 1) + shieldDaoBonus);
                    const isShield = shieldFinal > 0 || shieldBase > 0;
                    const baseWithMult = shieldBase * (shieldSkillMult + shieldBuffMult - 1);

                    const hasDamage = baseDmg > 0 || (b.final_damage !== undefined && b.final_damage > 0) || (b.hp_damage !== undefined && b.hp_damage > 0);

                    const renderHealCard = () => (
                      <div className="p-3.5 rounded-xl bg-black/95 border border-zinc-800 text-xs font-mono space-y-3 shadow-2xl animate-in fade-in duration-200">
                        {/* Header */}
                        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-2">
                          <span className="font-bold text-emerald-400 flex items-center gap-1.5 text-xs">
                            <Heart className="w-4 h-4 text-emerald-400" />
                            Chi Tiết Công Thức Hồi Phục
                          </span>
                          {b.reason && (
                            <span className="text-[10px] text-rose-400 bg-rose-950/50 border border-rose-800/50 px-2 py-0.5 rounded-full font-sans">
                              {b.reason}
                            </span>
                          )}
                        </div>

                        {/* Pipeline Step Summary Bar */}
                        <div className="p-2 rounded-lg bg-zinc-900/90 border border-zinc-800 flex flex-wrap items-center justify-between gap-2 text-[11px]">
                          <div className="flex items-center gap-1">
                            <span className="text-zinc-500">Hồi gốc:</span>
                            <span className="text-emerald-300 font-bold"><Num value={healBase} /></span>
                          </div>
                          {healDao > 0 && (
                            <>
                              <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />
                              <div className="flex items-center gap-1">
                                <span className="text-zinc-500">Đạo Ngân:</span>
                                <span className="text-indigo-300 font-bold">+<Num value={healDao} /></span>
                              </div>
                            </>
                          )}
                          <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />
                          <div className="flex items-center gap-1">
                            <span className="text-zinc-500">Hưởng:</span>
                            <span className={`font-bold ${healPct < 0 ? "text-rose-400" : "text-emerald-400"}`}>
                              {healPct > 0 ? "+" : ""}{(healPct * 100).toFixed(0)}%
                            </span>
                          </div>
                          <div className="flex items-center gap-1 bg-emerald-950/40 border border-emerald-800/40 px-2 py-0.5 rounded">
                            <span className="text-emerald-300 font-semibold">Hồi Thực:</span>
                            <span className="text-emerald-400 font-extrabold text-xs" title={healFinal.toLocaleString()}>
                              +<Num value={healFinal} /> HP
                            </span>
                          </div>
                        </div>

                        {/* Detailed 2-Stage Cards Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-[11px]">
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-emerald-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Sparkles className="w-3 h-3 text-emerald-400" />
                                1. Hồi Phục Đầu Ra
                              </span>
                              <span className="text-emerald-300 font-extrabold text-xs">
                                +<Num value={healBase + healDao} />
                              </span>
                            </div>
                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Hồi Phục Gốc:</span>
                                <span className="text-emerald-300 font-bold"><Num value={healBase} /></span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Đạo Ngân Cộng Thêm:</span>
                                <span className="text-indigo-300 font-bold">+<Num value={healDao} /></span>
                              </div>
                            </div>
                          </div>

                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-emerald-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Heart className="w-3 h-3 text-emerald-400" />
                                2. Hệ Số Hưởng & Kết Quả
                              </span>
                              <span className="text-emerald-400 font-extrabold text-xs">
                                +<Num value={healFinal} /> HP
                              </span>
                            </div>
                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Hưởng Hồi Phục:</span>
                                <span className={`font-bold ${healPct < 0 ? "text-rose-400" : "text-emerald-300"}`}>
                                  {healPct > 0 ? "+" : ""}{(healPct * 100).toFixed(0)}%
                                </span>
                              </div>
                              {(b.healer_healing?.mods ?? []).map((m) => (
                                <div key={m.key} className="flex justify-between items-center text-[10px]">
                                  <span className="text-rose-300/90">{m.name}:</span>
                                  <span className={`font-bold ${m.pct < 0 ? "text-rose-400" : "text-emerald-400"}`}>
                                    {m.pct > 0 ? "+" : ""}{(m.pct * 100).toFixed(0)}%
                                  </span>
                                </div>
                              ))}
                              <div className="flex justify-between items-center pt-1 border-t border-zinc-800/50 text-emerald-300">
                                <span className="font-semibold">Tổng Hồi Phục:</span>
                                <span className="text-emerald-400 font-black text-sm">
                                  +<Num value={healFinal} /> HP
                                </span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    );

                    const renderShieldCard = () => (
                      <div className="p-3.5 rounded-xl bg-black/95 border border-zinc-800 text-xs font-mono space-y-3 shadow-2xl animate-in fade-in duration-200">
                        {/* Header */}
                        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-2">
                          <span className="font-bold text-cyan-400 flex items-center gap-1.5 text-xs">
                            <Shield className="w-4 h-4 text-cyan-400" />
                            Chi Tiết Công Thức Tạo Khiên Ảo
                          </span>
                          {b.reason && (
                            <span className="text-[10px] text-cyan-400 bg-cyan-950/50 border border-cyan-800/50 px-2 py-0.5 rounded-full font-sans">
                              {b.reason}
                            </span>
                          )}
                        </div>

                        {/* Pipeline Step Summary Bar */}
                        <div className="p-2 rounded-lg bg-zinc-900/90 border border-zinc-800 flex flex-wrap items-center justify-between gap-2 text-[11px]">
                          <div className="flex items-center gap-1">
                            <span className="text-zinc-500">Khiên gốc:</span>
                            <span className="text-cyan-300 font-bold" title={shieldBase.toLocaleString()}>
                              <Num value={shieldBase} />
                            </span>
                          </div>

                          {(shieldSkillMult !== 1 || shieldBuffMult !== 1) && (
                            <>
                              <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />
                              <div className="flex items-center gap-1">
                                <span className="text-zinc-500">Hệ số:</span>
                                <span className="text-purple-300 font-bold">
                                  x{(shieldSkillMult + shieldBuffMult - 1).toFixed(2)}
                                </span>
                              </div>
                            </>
                          )}

                          {shieldDaoBonus > 0 && (
                            <>
                              <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />
                              <div className="flex items-center gap-1">
                                <span className="text-zinc-500">Đạo Ngân:</span>
                                <span className="text-indigo-300 font-bold" title={shieldDaoBonus.toLocaleString()}>
                                  +<Num value={shieldDaoBonus} />
                                </span>
                              </div>
                            </>
                          )}

                          <div className="flex items-center gap-1 bg-cyan-950/40 border border-cyan-800/40 px-2 py-0.5 rounded">
                            <span className="text-cyan-300 font-semibold">Khiên Thực:</span>
                            <span className="text-cyan-400 font-extrabold text-xs" title={shieldFinal.toLocaleString()}>
                              +<Num value={shieldFinal} /> Khiên
                            </span>
                          </div>
                        </div>

                        {/* Detailed 3-Stage Cards Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-[11px]">
                          {/* Stage 1: Khiên Đầu Ra */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-cyan-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Zap className="w-3 h-3 text-cyan-400" />
                                1. Khiên Đầu Ra
                              </span>
                              <span className="text-cyan-300 font-extrabold text-xs">
                                <Num value={baseWithMult} />
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Khiên Gốc:</span>
                                <span className="text-cyan-300 font-bold"><Num value={shieldBase} /></span>
                              </div>
                              <div className="flex justify-between items-start">
                                <span>Hệ Số Chiêu:</span>
                                <div className="text-right">
                                  <span className="text-purple-300 font-bold">
                                    x{(shieldSkillMult + shieldBuffMult - 1).toFixed(2)}
                                  </span>
                                  {b.reason && (
                                    <div className="text-[10px] text-amber-400/90 font-sans">
                                      ({b.reason})
                                    </div>
                                  )}
                                  {shieldBuffMult !== 1 && (
                                    <div className="text-[10px] text-purple-400/80">
                                      ({shieldSkillMult.toFixed(2)} chiêu + {shieldBuffMult.toFixed(2)} buff − 1)
                                    </div>
                                  )}
                                </div>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Sau Hệ Số:</span>
                                <span className="text-cyan-200 font-bold"><Num value={baseWithMult} /></span>
                              </div>
                              {b.shielder_defense && (b.shielder_defense.mods?.length ?? 0) > 0 && (
                                <StatProvenanceRows prov={b.shielder_defense} label="Phòng Ngự" />
                              )}
                            </div>
                          </div>

                          {/* Stage 2: Đạo Ngân Bổ Trợ */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-indigo-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Sparkles className="w-3 h-3 text-indigo-400" />
                                2. Đạo Ngân Bổ Trợ
                              </span>
                              <span className="text-indigo-300 font-extrabold text-xs">
                                +<Num value={shieldDaoBonus} />
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Hệ Số Đạo Ngân:</span>
                                <span className="text-indigo-300 font-bold">x<Num value={shieldDaoMult} /></span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Khiên Đạo Ngân:</span>
                                <span className="text-indigo-300 font-bold">+<Num value={shieldDaoBonus} /></span>
                              </div>
                              <div className="flex justify-between items-center text-[10px] text-zinc-500 pt-1 border-t border-zinc-800/50">
                                <span>Khuếch đại lưu phái</span>
                                <span className="text-indigo-400 font-semibold">
                                  {shieldDaoMult > 1 ? `+${((shieldDaoMult - 1) * 100).toFixed(0)}%` : "0%"}
                                </span>
                              </div>
                            </div>
                          </div>

                          {/* Stage 3: Kết Quả Khiên Ảo */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-cyan-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Shield className="w-3 h-3 text-cyan-400" />
                                3. Kết Quả Khiên Ảo
                              </span>
                              <span className="text-cyan-400 font-extrabold text-xs">
                                +<Num value={shieldFinal} /> Khiên
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Khiên Đầu Ra:</span>
                                <span className="text-cyan-300 font-bold"><Num value={baseWithMult} /></span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Đạo Ngân Bổ Sung:</span>
                                <span className="text-indigo-300 font-bold">+<Num value={shieldDaoBonus} /></span>
                              </div>
                              <div className="flex justify-between items-center pt-1 border-t border-zinc-800/50 text-cyan-300">
                                <span className="font-semibold">Tổng Khiên Nhận:</span>
                                <span className="text-cyan-400 font-black text-sm">
                                  +<Num value={shieldFinal} /> Khiên Ảo
                                </span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    );

                    const renderDamageCard = () => (
                      <div className="p-3.5 rounded-xl bg-black/95 border border-zinc-800 text-xs font-mono space-y-3 shadow-2xl animate-in fade-in duration-200">
                        {/* Header */}
                        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-2">
                          <span className="font-bold text-amber-400 flex items-center gap-1.5 text-xs">
                            <Calculator className="w-4 h-4 text-amber-400" />
                            Chi Tiết Công Thức Tính Sát Thương
                          </span>
                          {b.reason && (
                            <span className="text-[10px] text-rose-400 bg-rose-950/50 border border-rose-800/50 px-2 py-0.5 rounded-full font-sans">
                              {b.reason}
                            </span>
                          )}
                        </div>

                        {/* Pipeline Step Summary Bar */}
                        <div className="p-2 rounded-lg bg-zinc-900/90 border border-zinc-800 flex flex-wrap items-center justify-between gap-2 text-[11px]">
                          {/* 1. Raw Output */}
                          <div className="flex items-center gap-1">
                            <span className="text-zinc-500">Đầu ra:</span>
                            <span className="text-amber-300 font-bold" title={rawOutputDmg.toLocaleString()}>
                              <Num value={rawOutputDmg} />
                            </span>
                          </div>

                          <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />

                          {/* 2. Minus Armor */}
                          <div className="flex items-center gap-1">
                            <span className="text-zinc-500">Trừ giáp:</span>
                            <span className="text-cyan-400 font-bold">
                              -<Num value={targetDef} />
                            </span>
                            <span className="text-zinc-500">➔ Còn:</span>
                            <span className="text-cyan-300 font-bold" title={afterDefDmg.toLocaleString()}>
                              <Num value={afterDefDmg} />
                            </span>
                          </div>

                          <ArrowRight className="w-3.5 h-3.5 text-zinc-600 shrink-0 hidden sm:inline" />

                          {/* 3. Shield or Final */}
                          <div className="flex items-center gap-1">
                            {pathResist > 0 || dmgReduction > 0 ? (
                              <>
                                <span className="text-zinc-500">Kháng:</span>
                                <span className="text-emerald-400 font-bold">
                                  -{((pathResist + dmgReduction) * 100).toFixed(0)}%
                                </span>
                                <span className="text-zinc-500">➔</span>
                              </>
                            ) : null}
                            {shieldAbsorbed > 0 && (
                              <>
                                <span className="text-zinc-500">Khiên cản:</span>
                                <span className="text-cyan-400 font-bold">
                                  <Num value={shieldAbsorbed} />
                                </span>
                                <span className="text-zinc-500">➔</span>
                              </>
                            )}
                          </div>

                          {/* 4. HP Lost */}
                          <div className="flex items-center gap-1 bg-rose-950/40 border border-rose-800/40 px-2 py-0.5 rounded">
                            <span className="text-rose-300 font-semibold">Trừ Máu:</span>
                            <span className="text-rose-400 font-extrabold text-xs" title={hpLost.toLocaleString()}>
                              -<Num value={hpLost} />
                            </span>
                          </div>
                        </div>

                        {/* Detailed 3-Stage Cards Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-[11px]">
                          {/* Stage 1: Xuất Chiêu & Khuếch Đại */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-amber-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Zap className="w-3 h-3 text-amber-400" />
                                1. Sát Thương Đầu Ra
                              </span>
                              <span className="text-amber-300 font-extrabold text-xs">
                                <Num value={rawOutputDmg} />
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Dame Gốc:</span>
                                <span className="text-amber-300 font-bold"><Num value={baseDmg} /></span>
                              </div>
                              <div className="flex justify-between items-start">
                                <span>Hệ Số Chiêu:</span>
                                <div className="text-right">
                                  <span className="text-purple-300 font-bold">
                                    x{(skillMult * buffMult).toFixed(2)}
                                  </span>
                                  {b.reason && (
                                    <div className="text-[10px] text-amber-400/90 font-sans">
                                      ({b.reason})
                                    </div>
                                  )}
                                  {buffMult !== 1 && (
                                    <div className="text-[10px] text-purple-400/80">
                                      ({skillMult.toFixed(2)} chiêu × {buffMult.toFixed(2)} buff)
                                    </div>
                                  )}
                                </div>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Đạo Ngân:</span>
                                <span className="text-indigo-300 font-bold">x<Num value={daoMult} /></span>
                              </div>
                              {critMult > 1 && (
                                <div className="flex justify-between items-center">
                                  <span>Bạo Kích (Crit):</span>
                                  <span className="text-rose-300 font-bold">x{critMult.toFixed(2)}</span>
                                </div>
                              )}
                              {pathBonus > 0 && (
                                <div className="flex justify-between items-center">
                                  <span>Tăng ST Hệ:</span>
                                  <span className="text-emerald-300 font-bold">+{(pathBonus * 100).toFixed(0)}%</span>
                                </div>
                              )}
                              {b.attacker_attack && (b.attacker_attack.mods?.length ?? 0) > 0 && (
                                <StatProvenanceRows prov={b.attacker_attack} label="Tấn Công" />
                              )}
                            </div>
                          </div>

                          {/* Stage 2: Phòng Thủ & Giảm Trừ */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-cyan-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Shield className="w-3 h-3 text-cyan-400" />
                                2. Giảm Trừ Phòng Ngự
                              </span>
                              <span className="text-cyan-300 font-extrabold text-xs">
                                <Num value={finalDmg} />
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>Thủ Mục Tiêu:</span>
                                <span className="text-cyan-400 font-bold">-<Num value={targetDef} /> Giáp</span>
                              </div>
                              {b.defender_defense && (b.defender_defense.mods?.length ?? 0) > 0 && (
                                <StatProvenanceRows prov={b.defender_defense} label="Phòng Ngự" />
                              )}
                              <div className="flex justify-between items-center">
                                <span>ST Sau Trừ Giáp:</span>
                                <span className="text-cyan-200 font-bold"><Num value={afterDefDmg} /></span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Kháng Thuộc Tính:</span>
                                <span className="text-emerald-300 font-bold">{(pathResist * 100).toFixed(0)}% Kháng</span>
                              </div>
                              {dmgReduction > 0 && (
                                <div className="flex justify-between items-center">
                                  <span>Giảm Sát Thương:</span>
                                  <span className="text-emerald-300 font-bold">-{(dmgReduction * 100).toFixed(0)}% ST</span>
                                </div>
                              )}
                              <div className="flex justify-between items-center pt-1 border-t border-zinc-800/50">
                                <span>ST Thực Gây Ra:</span>
                                <span className="text-emerald-400 font-extrabold"><Num value={finalDmg} /></span>
                              </div>
                            </div>
                          </div>

                          {/* Stage 3: Phân Bổ Vào Mục Tiêu */}
                          <div className="bg-zinc-900/60 p-2.5 rounded-xl border border-zinc-800/80 space-y-2">
                            <div className="text-[10px] text-rose-400 font-bold uppercase flex items-center justify-between border-b border-zinc-800 pb-1.5">
                              <span className="flex items-center gap-1">
                                <Heart className="w-3 h-3 text-rose-400" />
                                3. Tác Động Lên Mục Tiêu
                              </span>
                              <span className="text-rose-400 font-extrabold text-xs">
                                -<Num value={hpLost} /> HP
                              </span>
                            </div>

                            <div className="space-y-1.5 text-zinc-400 text-[11px]">
                              <div className="flex justify-between items-center">
                                <span>ST Thực Nhận:</span>
                                <span className="text-zinc-200 font-bold"><Num value={finalDmg} /></span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span>Khiên Hấp Thụ:</span>
                                <span className={`font-bold ${shieldAbsorbed > 0 ? "text-cyan-400" : "text-zinc-500"}`}>
                                  {shieldAbsorbed > 0 ? <Num value={shieldAbsorbed} /> : "0"}
                                </span>
                              </div>
                              <div className="flex justify-between items-center pt-1 border-t border-zinc-800/50 text-rose-300">
                                <span className="font-semibold">Trừ Thẳng Vào Máu:</span>
                                <span className="text-rose-400 font-black text-sm">
                                  <Num value={hpLost} />
                                </span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    );

                    return (
                      <div className="mt-2.5 space-y-2.5">
                        {isHeal && renderHealCard()}
                        {hasDamage && renderDamageCard()}
                        {isShield && renderShieldCard()}
                      </div>
                    );
                  })()}
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
