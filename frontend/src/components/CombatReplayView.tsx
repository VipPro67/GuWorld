"use client";

import React, { useReducer, useMemo, useRef, useEffect, useCallback } from "react";
import {
  Swords, SkipForward, Heart, Shield, Skull,
  Zap, FlaskConical, Sparkles, Flame, Play, Pause,
  ArrowLeft, ChevronRight, Droplets, Coins, Gem,
} from "lucide-react";
import { formatNumber } from "@/lib/utils";
import { Num } from "./Num";
import { translateMastery, masteryKeyForPoints } from "@/lib/translate";

export interface CombatLogEntry {
  turn: number;
  actor_id: string;
  actor_name: string;
  action: string;
  target_id?: string;
  target_name?: string;
  damage?: number;
  is_crit?: boolean;
  trigger?: string;
  detail?: string;
}

interface CombatantReplayInfo {
  id: string;
  name: string;
  kind: "ally" | "enemy";
  max_hp: number;
  hp: number;
  max_mana: number;
  mana: number;
  grid_x: number;
  grid_y: number;
  shield?: number;
  damage?: number;
  defense?: number;
  speed?: number;
  kind_detail?: string;
  rank?: number;
  sub_rank?: number;
  dao_marks?: Record<string, number>;
  path_masteries?: Record<string, number>;
  avatar_url?: string;
}

interface CombatReplayViewProps {
  logs: CombatLogEntry[];
  combatants?: CombatantReplayInfo[];
  winner: string;
  onBack: () => void;
  playerName?: string;
}

interface CombatantState {
  id: string;
  name: string;
  kind: "ally" | "enemy";
  maxHp: number;
  hp: number;
  maxMp: number;
  mp: number;
  shield: number;
  gridX: number;
  gridY: number;
  damage: number;
  defense: number;
  speed: number;
  kindDetail: string;
  rank: number;
  subRank: number;
  daoMarks?: Record<string, number>;
  pathMasteries?: Record<string, number>;
  avatarUrl?: string;
}

interface FloatingEvent {
  id: number;
  actorId: string;
  targetId?: string;
  text: string;
  type: "damage" | "heal" | "shield" | "crit" | "info" | "skill" | "death";
}

interface ReplayState {
  combatantStates: Record<string, CombatantState>;
  currentIndex: number;
  isPlaying: boolean;
  currentTurn: number;
  highlightedActor: string | null;
  highlightedTarget: string | null;
  events: FloatingEvent[];
}

type ReplayAction =
  | { type: "INIT"; combatants: Record<string, CombatantState> }
  | { type: "ADVANCE"; log: CombatLogEntry }
  | { type: "SET_INDEX"; index: number }
  | { type: "TOGGLE_PLAY" }
  | { type: "STOP" }
  | { type: "CLEAR_HIGHLIGHTS" }
  | { type: "SET_EVENTS"; events: FloatingEvent[] }
  | { type: "SET_TURN"; turn: number }
  | { type: "UPDATE_STATES"; combatants: Record<string, CombatantState> };

const TURN_REGEX = /=== LƯỢT (\d+) ===/;

let nextEventId = 0;

function replayReducer(state: ReplayState, action: ReplayAction): ReplayState {
  switch (action.type) {
    case "INIT":
      return {
        combatantStates: action.combatants,
        currentIndex: 0,
        isPlaying: true,
        currentTurn: 0,
        highlightedActor: null,
        highlightedTarget: null,
        events: [],
      };

    case "ADVANCE": {
      const log = action.log;
      const next = { ...state.combatantStates };

      // Update combatant states
      if (log.actor_id && next[log.actor_id]) {
        const actor = { ...next[log.actor_id] };
        if (["HEAL", "ITEM_HEAL", "LIFESTEAL"].includes(log.action)) {
          actor.hp = Math.min(actor.maxHp, actor.hp + (log.damage || 0));
        }
        if (log.action === "SHIELD_GAIN" || log.action === "PRE_COMBAT_BARRIER") {
          actor.shield += log.damage || 0;
        }
        if (log.action === "SHIELD_BREAK") {
          actor.shield = 0;
        }
        if (log.action === "SHIELD_ABSORB") {
          actor.shield = Math.max(0, actor.shield - (log.damage || 0));
        }
        if (log.action === "DIED") {
          actor.hp = 0;
        }
        if (log.action === "MANA_COST") {
          actor.mp = Math.max(0, actor.mp - (log.damage || 0));
        }
        if (log.action === "MANA_RESTORE") {
          actor.mp = Math.min(actor.maxMp, actor.mp + (log.damage || 0));
        }
        next[log.actor_id] = actor;
      }

      if (log.target_id && next[log.target_id]) {
        const target = { ...next[log.target_id] };
        if (["DAMAGE", "BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "PHANTOM_ATTACK", "THORNS", "DOT_TICK"].includes(log.action)) {
          target.hp = Math.max(0, target.hp - (log.damage || 0));
        }
        if (log.action === "SHIELD_ABSORB") {
          target.shield = Math.max(0, target.shield - (log.damage || 0));
        }
        if (log.action === "SHIELD_BREAK") {
          target.shield = 0;
        }
        if (log.action === "DIED") {
          target.hp = 0;
        }
        next[log.target_id] = target;
      }

      // Build events
      const events: FloatingEvent[] = [];
      if (log.damage && ["DAMAGE", "DOT_TICK", "THORNS", "PHANTOM_ATTACK", "SHIELD_PIERCE", "SHIELD_BYPASS_ATTACK"].includes(log.action)) {
        events.push({
          id: ++nextEventId, actorId: log.actor_id, targetId: log.target_id,
          text: `-${formatNumber(log.damage)}`,
          type: log.is_crit ? "crit" : "damage",
        });
      }
      if (["HEAL", "ITEM_HEAL", "LIFESTEAL"].includes(log.action)) {
        events.push({
          id: ++nextEventId, actorId: log.actor_id,
          text: `+${formatNumber(log.damage || 0)}`,
          type: "heal",
        });
      }
      if (log.action === "SHIELD_GAIN") {
        events.push({
          id: ++nextEventId, actorId: log.actor_id,
          text: `🛡 +${formatNumber(log.damage || 0)}`,
          type: "shield",
        });
      }
      if (log.action === "DIED") {
        events.push({
          id: ++nextEventId, actorId: log.target_id || log.actor_id,
          text: "†",
          type: "death",
        });
      }
      if (["GU_SKILL", "KILLER_MOVE", "BASIC_ATTACK"].includes(log.action)) {
        const detail = log.detail || "";
        const idxDung = detail.indexOf("dùng [");
        const idxSuDung = detail.indexOf("sử dụng");
        const idx = idxDung >= 0 ? idxDung : idxSuDung;
        if (idx >= 0) {
          const afterKeyword = detail.substring(idx).replace(/^.*?\[([^\]]+)\].*$/, "[$1]");
          const bracketMatch = detail.substring(idx).match(/\[([^\]]+)\]/);
          if (bracketMatch) {
            events.push({
              id: ++nextEventId, actorId: log.actor_id,
              text: bracketMatch[1],
              type: "skill",
            });
          }
        }
      }

      return {
        ...state,
        combatantStates: next,
        currentIndex: state.currentIndex + 1,
        highlightedActor: log.actor_id || null,
        highlightedTarget: log.target_id || null,
        events,
      };
    }

    case "SET_INDEX":
      return { ...state, currentIndex: action.index };

    case "TOGGLE_PLAY":
      return { ...state, isPlaying: !state.isPlaying };

    case "STOP":
      return { ...state, isPlaying: false };

    case "CLEAR_HIGHLIGHTS":
      return { ...state, highlightedActor: null, highlightedTarget: null };

    case "SET_EVENTS":
      return { ...state, events: action.events };

    case "SET_TURN":
      return { ...state, currentTurn: action.turn };

    case "UPDATE_STATES":
      return { ...state, combatantStates: action.combatants };

    default:
      return state;
  }
}

const actionIcon = (action: string) => {
  switch (action) {
    case "BASIC_ATTACK": return <Swords className="w-3 h-3" />;
    case "GU_SKILL": return <Sparkles className="w-3 h-3" />;
    case "KILLER_MOVE": return <Zap className="w-3 h-3" />;
    case "HEAL": case "ITEM_HEAL": return <Heart className="w-3 h-3" />;
    case "SHIELD_GAIN": case "SHIELD_ABSORB": return <Shield className="w-3 h-3" />;
    case "SHIELD_BREAK": case "SHIELD_PIERCE": case "SHIELD_BYPASS_ATTACK": return <Swords className="w-3 h-3" />;
    case "DAMAGE": return <Swords className="w-3 h-3" />;
    case "DOT_TICK": return <Droplets className="w-3 h-3" />;
    case "DIED": return <Skull className="w-3 h-3" />;
    case "LIFESTEAL": return <FlaskConical className="w-3 h-3" />;
    case "STUNNED": return <Zap className="w-3 h-3" />;
    case "MINE_LOOT": return <Coins className="w-3 h-3" />;
    case "MINE_OCCUPY": return <Gem className="w-3 h-3" />;
    case "BUFF": return <Sparkles className="w-3 h-3" />;
    case "DEFEND": return <Shield className="w-3 h-3" />;
    default: return <ChevronRight className="w-3 h-3" />;
  }
};

const actionColor = (action: string) => {
  switch (action) {
    case "DAMAGE": case "BASIC_ATTACK": case "SHIELD_BREAK": case "SHIELD_PIERCE": case "SHIELD_BYPASS_ATTACK": return "text-rose-400";
    case "GU_SKILL": return "text-emerald-400";
    case "KILLER_MOVE": return "text-purple-400";
    case "HEAL": case "ITEM_HEAL": return "text-emerald-400";
    case "SHIELD_GAIN": case "SHIELD_ABSORB": case "DEFEND": return "text-blue-400";
    case "DOT_TICK": return "text-yellow-400";
    case "DIED": return "text-rose-500";
    case "LIFESTEAL": return "text-pink-400";
    case "BATTLE_START": case "BATTLE_END": return "text-amber-400";
    case "MINE_LOOT": return "text-yellow-400";
    case "MINE_OCCUPY": return "text-emerald-400";
    case "BUFF": return "text-emerald-300";
    default: return "text-zinc-400";
  }
};

export const CombatReplayView: React.FC<CombatReplayViewProps> = ({
  logs,
  combatants: initialCombatants,
  winner,
  onBack,
  playerName,
}) => {
  const [speed, setSpeed] = React.useState(1);
  const logContainerRef = useRef<HTMLDivElement>(null);
  const logEndRef = useRef<HTMLDivElement>(null);

  // Parse initial combatant states
  const parsedCombatants = useMemo(() => {
    if (initialCombatants && initialCombatants.length > 0) {
      const map: Record<string, CombatantState> = {};
      for (const c of initialCombatants) {
        map[c.id] = {
          id: c.id,
          name: c.name,
          kind: c.kind,
          maxHp: c.max_hp || 100,
          hp: c.hp || (c.max_hp || 100),
          maxMp: c.max_mana || 0,
          mp: c.mana || 0,
          shield: c.shield || 0,
          gridX: c.grid_x || 2,
          gridY: c.grid_y || 2,
          damage: c.damage || 0,
          defense: c.defense || 0,
          speed: c.speed || 0,
          kindDetail: c.kind_detail || c.kind || "",
          rank: c.rank || 0,
          subRank: c.sub_rank || 0,
          daoMarks: c.dao_marks,
          pathMasteries: c.path_masteries,
          avatarUrl: c.avatar_url,
        };
      }
      return map;
    }

    const map: Record<string, CombatantState> = {};
    for (const log of logs) {
      if (log.actor_id && !map[log.actor_id]) {
        map[log.actor_id] = {
          id: log.actor_id,
          name: log.actor_name || "???",
          kind: "enemy",
          maxHp: 1000,
          hp: 1000,
          maxMp: 0,
          mp: 0,
          shield: 0,
          gridX: 1,
          gridY: 2,
          damage: 0,
          defense: 0,
          speed: 0,
          kindDetail: "",
          rank: 0,
          subRank: 0,
        };
      }
      if (log.target_id && !map[log.target_id]) {
        map[log.target_id] = {
          id: log.target_id,
          name: log.target_name || log.target_id,
          kind: "enemy",
          maxHp: 1000,
          hp: 1000,
          maxMp: 0,
          mp: 0,
          shield: 0,
          gridX: 1,
          gridY: 2,
          damage: 0,
          defense: 0,
          speed: 0,
          kindDetail: "",
          rank: 0,
          subRank: 0,
        };
      }
    }
    return map;
  }, [initialCombatants, logs]);

  const [state, dispatch] = useReducer(replayReducer, {
    combatantStates: parsedCombatants,
    currentIndex: 0,
    isPlaying: false,
    currentTurn: 0,
    highlightedActor: null,
    highlightedTarget: null,
    events: [],
  });

  // Re-initialize when combatants change
  useEffect(() => {
    dispatch({ type: "INIT", combatants: parsedCombatants });
  }, [parsedCombatants]);

  const intervalMs = useMemo(() => {
    switch (speed) {
      case 1: return 500;
      case 2: return 200;
      case 3: return 80;
      default: return 500;
    }
  }, [speed]);

  const processNextLog = useCallback(() => {
    const idx = state.currentIndex;
    if (idx < 0 || idx >= logs.length) return;

    const log = logs[idx];

    if (log.action === "TURN_START") {
      const m = TURN_REGEX.exec(log.detail || "");
      if (m) dispatch({ type: "SET_TURN", turn: parseInt(m[1]) });
      dispatch({ type: "SET_INDEX", index: idx + 1 });
      return;
    }

    if (log.action === "BATTLE_START") {
      dispatch({ type: "SET_TURN", turn: 0 });
      dispatch({ type: "SET_INDEX", index: idx + 1 });
      return;
    }

    dispatch({ type: "ADVANCE", log });
  }, [state.currentIndex, logs]);

  // Auto-play timer
  useEffect(() => {
    if (!state.isPlaying || state.currentIndex >= logs.length) return;

    if (state.currentIndex >= logs.length - 1) {
      dispatch({ type: "STOP" });
      return;
    }

    const timer = setTimeout(() => {
      processNextLog();
    }, intervalMs);
    return () => clearTimeout(timer);
  }, [state.isPlaying, state.currentIndex, logs.length, intervalMs, processNextLog]);

  // Clear highlights after a delay
  useEffect(() => {
    if (!state.highlightedActor && !state.highlightedTarget) return;
    const timer = setTimeout(() => {
      dispatch({ type: "CLEAR_HIGHLIGHTS" });
    }, intervalMs * 0.8);
    return () => clearTimeout(timer);
  }, [state.highlightedActor, state.highlightedTarget, intervalMs]);

  // Clear events after a delay
  useEffect(() => {
    if (state.events.length === 0) return;
    const timer = setTimeout(() => {
      dispatch({ type: "SET_EVENTS", events: [] });
    }, 1200);
    return () => clearTimeout(timer);
  }, [state.events]);

  // Scroll log without jumping the whole page
  useEffect(() => {
    if (logContainerRef.current) {
      const container = logContainerRef.current;
      // Only auto-scroll if we are relatively close to the bottom to allow manual scrolling
      const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
      if (isNearBottom || state.currentIndex <= 2) {
        container.scrollTop = container.scrollHeight;
      }
    }
  }, [state.currentIndex]);

  const handleTogglePlay = useCallback(() => {
    if (state.currentIndex >= logs.length - 1) {
      // Reset and play
      dispatch({ type: "INIT", combatants: parsedCombatants });
      setTimeout(() => dispatch({ type: "TOGGLE_PLAY" }), 0);
    } else {
      dispatch({ type: "TOGGLE_PLAY" });
    }
  }, [state.currentIndex, logs.length, parsedCombatants]);

  const handleSkip = useCallback(() => {
    // Process all remaining logs
    let current = { ...state, currentIndex: state.currentIndex };
    for (let i = state.currentIndex; i < logs.length; i++) {
      const log = logs[i];
      if (log.action === "TURN_START" || log.action === "BATTLE_START") {
        if (log.action === "TURN_START") {
          const m = TURN_REGEX.exec(log.detail || "");
          if (m) current.currentTurn = parseInt(m[1]);
        }
        current.currentIndex = i + 1;
        continue;
      }
      // Apply log to state
      const action: ReplayAction = { type: "ADVANCE", log };
      current = replayReducer(current, action);
    }
    // Set final state without resetting currentIndex
    dispatch({ type: "UPDATE_STATES", combatants: current.combatantStates });
    dispatch({ type: "SET_INDEX", index: logs.length - 1 });
    dispatch({ type: "STOP" });
  }, [state, logs]);

  const isCompleted = state.currentIndex >= logs.length - 1;

  // Group combatants by side
  const allies = useMemo(() =>
    Object.values(state.combatantStates).filter(c => c.kind === "ally"),
    [state.combatantStates]
  );
  const enemies = useMemo(() =>
    Object.values(state.combatantStates).filter(c => c.kind === "enemy"),
    [state.combatantStates]
  );

  const hpPct = (c: CombatantState) =>
    c.maxHp > 0 ? Math.max(0, Math.min(100, (c.hp / c.maxHp) * 100)) : 0;

  const [detailCombatant, setDetailCombatant] = React.useState<CombatantState | null>(null);

  const renderGridCell = (c: CombatantState | undefined, gridX: number, gridY: number) => {
    const isAlly = c?.kind === "ally";
    const isEmpty = !c;
    const isHighlighted = state.highlightedActor === c?.id;
    const isTargeted = state.highlightedTarget === c?.id;
    const pct = c ? hpPct(c) : 0;

    const borderGlow = isHighlighted ? "border-emerald-400 shadow-[0_0_12px_rgba(52,211,153,0.35)]" :
      isTargeted ? "border-rose-400 shadow-[0_0_12px_rgba(244,63,94,0.35)]" : "";

    return (
      <div
        key={`${gridX}-${gridY}`}
        onClick={() => c && setDetailCombatant(detailCombatant?.id === c.id ? null : c)}
        className={`rounded-lg border ${
          isEmpty
            ? 'border-zinc-800/30 bg-zinc-900/20'
            : c.hp <= 0
              ? 'border-zinc-800/40 bg-zinc-950/40 opacity-50'
              : isAlly
                ? `border-emerald-800/40 bg-emerald-950/20 ${borderGlow}`
                : `border-rose-800/40 bg-rose-950/20 ${borderGlow}`
        } flex flex-col items-center p-1.5 relative cursor-pointer transition-all hover:brightness-125 aspect-square`}
      >
        {c ? (
          <>
            {/* Avatar */}
            <div className="w-8 h-8 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[11px] font-bold text-zinc-500 border-2 border-white/10 mb-0.5">
              {c.avatarUrl ? (
                <img src={c.avatarUrl} alt="" className="w-full h-full object-cover" />
              ) : (
                c.name.slice(0, 2).toUpperCase()
              )}
            </div>

            {/* Name + death marker */}
            <div className="text-center min-w-0 w-full">
              <span className={`text-[9px] font-bold truncate block leading-tight ${c.hp <= 0 ? 'text-zinc-700' : isAlly ? 'text-emerald-300' : 'text-rose-300'}`}>
                {c.name}
                {c.hp <= 0 && '†'}
              </span>
            </div>

            {/* Type badge */}
            {!isAlly && c.kindDetail && c.kindDetail !== "" && (
              <span className={`text-[6px] px-1 rounded border font-bold mt-0.5 ${
                c.kindDetail === "Boss" ? 'bg-red-900/60 text-red-300 border-red-800' :
                c.kindDetail === "Elite" ? 'bg-purple-900/60 text-purple-300 border-purple-800' :
                'bg-zinc-800/60 text-zinc-400 border-zinc-700'
              }`}>
                {c.kindDetail.toUpperCase()}
              </span>
            )}

            {/* HP Bar */}
            <div className="w-full h-1 bg-zinc-800 rounded-full overflow-hidden mt-0.5">
              <div
                className={`h-full rounded-full transition-all ${c.hp <= 0 ? 'bg-zinc-700' : isAlly ? 'bg-emerald-500' : 'bg-rose-500'}`}
                style={{ width: `${pct}%` }}
              />
            </div>

            {/* HP number */}
            <div className="text-[7px] font-mono mt-0.5">
              <span className={c.hp <= 0 ? 'text-zinc-700' : 'text-zinc-400'}>
                {Math.round(c.hp)}/{Math.round(c.maxHp)}
              </span>
            </div>

            {/* Mana Bar */}
            {c.maxMp > 0 && (
              <>
                <div className="w-full h-1 bg-zinc-800 rounded-full overflow-hidden mt-0.5">
                  <div
                    className="h-full rounded-full bg-purple-500/70 transition-all"
                    style={{ width: `${c.maxMp > 0 ? Math.min(100, (c.mp / c.maxMp) * 100) : 0}%` }}
                  />
                </div>
                <div className="text-[6px] font-mono text-purple-400/70 mt-0.5">
                  MP {Math.round(c.mp)}/{Math.round(c.maxMp)}
                </div>
              </>
            )}

            {/* Shield */}
            {c.shield > 0 && (
              <div className="text-[6px] font-mono text-cyan-400/80 mt-0.5 flex items-center gap-0.5">
                🛡 <Num value={Math.round(c.shield)} />
              </div>
            )}
          </>
        ) : (
          <span className="text-[8px] text-zinc-800 mx-auto mt-auto mb-auto">
            {gridX},{gridY}
          </span>
        )}
      </div>
    );
  };

  // Detailed info popup
  const renderDetailPopup = () => {
    if (!detailCombatant) return null;
    const c = detailCombatant;
    const isAlly = c.kind === "ally";
    const kindLabel = c.kindDetail || c.kind;

    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
        onClick={() => setDetailCombatant(null)}
      >
        <div
          className={`bg-zinc-900 border ${isAlly ? 'border-emerald-800/50' : 'border-rose-800/50'} rounded-2xl p-5 shadow-2xl max-w-sm w-full mx-4`}
          onClick={e => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[11px] font-bold text-zinc-500">
                {c.avatarUrl ? (
                  <img src={c.avatarUrl} alt="" className="w-full h-full object-cover" />
                ) : (
                  c.name.slice(0, 2).toUpperCase()
                )}
              </div>
              <h3 className={`text-base font-bold ${isAlly ? 'text-emerald-300' : 'text-rose-300'}`}>{c.name}</h3>
            </div>
            <button
              onClick={() => setDetailCombatant(null)}
              className="text-zinc-600 hover:text-zinc-300 text-lg leading-none"
            >
              ✕
            </button>
          </div>

          <div className="space-y-2 text-xs">
            {/* Kind + Rank */}
            <div className="flex items-center gap-2">
              {kindLabel && kindLabel !== "Player" && (
                <span className={`px-2 py-0.5 rounded text-[10px] font-bold border ${
                  kindLabel === "Boss" ? 'bg-red-900/60 text-red-300 border-red-800' :
                  kindLabel === "Elite" ? 'bg-purple-900/60 text-purple-300 border-purple-800' :
                  'bg-zinc-800/60 text-zinc-400 border-zinc-700'
                }`}>{kindLabel.toUpperCase()}</span>
              )}
              {c.rank > 0 && <span className="text-amber-400 font-bold">R{c.rank}.{c.subRank}</span>}
            </div>

            {/* Vital stats */}
            <div className="grid grid-cols-2 gap-2 bg-zinc-950/60 rounded-xl p-3 mt-2">
              <div><span className="text-zinc-500">Sinh lực</span><br /><span className="text-emerald-400 font-bold text-sm"><Num value={c.hp} />/<Num value={c.maxHp} /></span></div>
              <div><span className="text-zinc-500">Chân nguyên</span><br /><span className="text-purple-400 font-bold text-sm"><Num value={c.mp} />/<Num value={c.maxMp} /></span></div>
              <div><span className="text-zinc-500">Tấn công</span><br /><span className="text-rose-400 font-bold"><Num value={c.damage} /></span></div>
              <div><span className="text-zinc-500">Phòng thủ</span><br /><span className="text-blue-400 font-bold"><Num value={c.defense} /></span></div>
              <div><span className="text-zinc-500">Tốc độ</span><br /><span className="text-cyan-400 font-bold"><Num value={c.speed} /></span></div>
              <div><span className="text-zinc-500">Khiên</span><br /><span className="text-indigo-400 font-bold"><Num value={c.shield} /></span></div>
            </div>

            {/* Position */}
            <div className="text-zinc-500">
              Vị trí: <span className="text-zinc-300 font-mono">({c.gridX},{c.gridY})</span>
            </div>

            {/* Dao Marks */}
            {c.daoMarks && Object.keys(c.daoMarks).length > 0 && (
              <div className="border-t border-zinc-800 pt-2 mt-2">
                <span className="text-zinc-500 text-[10px]">Đạo Ấn:</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {Object.entries(c.daoMarks).map(([k, v]) => (
                    <span key={k} className="text-[9px] px-1.5 py-0.5 rounded bg-amber-950/30 text-amber-400 border border-amber-800/40 font-mono">
                      {k}: {v}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Path Masteries */}
            {c.pathMasteries && Object.keys(c.pathMasteries).length > 0 && (
              <div className="border-t border-zinc-800 pt-2 mt-2">
                <span className="text-zinc-500 text-[10px]">Đường Tu:</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {Object.entries(c.pathMasteries).map(([k, points]) => (
                    <span key={k} className="text-[9px] px-1.5 py-0.5 rounded bg-cyan-950/30 text-cyan-400 border border-cyan-800/40 font-mono">
                      {k}: {translateMastery(masteryKeyForPoints(points as number))}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Status indicator */}
            <div className="border-t border-zinc-800 pt-2 mt-2 text-center">
              {c.hp <= 0 ? (
                <span className="text-rose-500 font-bold">† Đã gục ngã</span>
              ) : (
                <span className="text-emerald-500">● Đang chiến đấu</span>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  };

  const parseSummary = useCallback(() => {
    let totalDmgDealt = 0;
    let totalDmgTaken = 0;
    let shieldsUsed = 0;
    let manaCost = 0;
    let itemsUsed = 0;
    let stonesUsed = 0;

    for (const log of logs) {
      if (log.damage && log.damage > 0) {
        const isFromPlayer = log.actor_name === playerName || log.actor_id === playerName;
        if (isFromPlayer) totalDmgDealt += log.damage;
        else totalDmgTaken += log.damage;
      }
      if (log.action === "SHIELD_GAIN" || log.action === "SHIELD_ABSORB") {
        shieldsUsed += log.damage || 0;
      }
      if (log.action === "MANA_COST") {
        const isFromPlayer = log.actor_name === playerName || log.actor_id === playerName;
        if (isFromPlayer) manaCost += log.damage || 0;
      }
      if (log.action === "AUTO_ITEM_USAGE") {
        const m = log.detail?.match(/(\d+)/);
        if (m) itemsUsed += parseInt(m[1]);
      }
      if (log.action === "AUTO_STONE_USAGE") {
        const m = log.detail?.match(/(\d+)/);
        if (m) stonesUsed += parseInt(m[1]);
      }
    }
    return { totalDmgDealt, totalDmgTaken, shieldsUsed, manaCost, itemsUsed, stonesUsed };
  }, [logs, playerName]);

  const summary = useMemo(() => {
    if (isCompleted) return parseSummary();
    return null;
  }, [isCompleted, parseSummary]);

  const totalTurns = useMemo(() => {
    let maxTurn = 0;
    for (const log of logs) {
      if (log.turn > maxTurn) maxTurn = log.turn;
    }
    return maxTurn;
  }, [logs]);

  const visibleLogs = useMemo(() => logs.slice(0, state.currentIndex + 1), [logs, state.currentIndex]);

  const actionLogs = useMemo(() =>
    visibleLogs.filter(log =>
      ["BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "DAMAGE", "HEAL", "ITEM_HEAL", "SHIELD_GAIN", "DOT_TICK", "DIED", "LIFESTEAL", "STUNNED", "PHANTOM_ATTACK", "THORNS", "SHIELD_ABSORB", "SHIELD_BREAK", "BLEED_APPLIED", "POISON_APPLIED", "STUN_APPLIED", "MANA_RESTORE", "MANA_COST", "PRE_COMBAT_BARRIER", "MULTI_HIT", "LOOT", "AUTO_ITEM_USAGE", "AUTO_STONE_USAGE", "MINE_LOOT", "MINE_OCCUPY"].includes(log.action)
    ), [visibleLogs]);

  return (
    <div className="w-full max-w-5xl mx-auto flex flex-col gap-4">
      {/* Header */}
      <div className="bg-black/60 border border-zinc-800 rounded-xl p-4 flex items-center justify-between flex-wrap gap-3">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="p-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-400 rounded-lg transition-colors"
            title="Quay lại"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <div>
            <h2 className="text-lg font-bold text-zinc-100 flex items-center gap-2">
              <Flame className="w-5 h-5 text-rose-500" />
              Chiến Đấu Mô Phỏng
              {isCompleted && (
                <span className={`text-xs px-2 py-0.5 rounded-full font-bold ${winner === 'Phe Ta' ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-rose-900/50 text-rose-400 border border-rose-800'}`}>
                  {winner === 'Phe Ta' ? 'THẮNG' : 'BẠI'}
                </span>
              )}
            </h2>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-zinc-500 mr-1">Lượt {state.currentTurn}/{totalTurns}</span>
          {[1, 2, 3].map(s => (
            <button
              key={s}
              onClick={() => setSpeed(s)}
              className={`px-2 py-1 rounded text-xs font-bold transition-all ${speed === s ? 'bg-cyan-900/50 text-cyan-300 border border-cyan-800' : 'bg-zinc-800 text-zinc-400 border border-zinc-700 hover:bg-zinc-700'}`}
            >
              {s}x
            </button>
          ))}
          <button
            onClick={handleSkip}
            className="px-2 py-1 bg-zinc-800 text-zinc-400 rounded hover:bg-zinc-700 transition-colors"
            title="Skip"
          >
            <SkipForward className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={handleTogglePlay}
            className={`px-3 py-1 rounded text-xs font-bold transition-all ${state.isPlaying ? 'bg-amber-900/50 text-amber-300 border border-amber-800' : 'bg-purple-900/50 text-purple-300 border border-purple-800 hover:bg-purple-800/50'}`}
          >
            {state.isPlaying ? <Pause className="w-3.5 h-3.5 inline" /> : <Play className="w-3.5 h-3.5 inline" />}
            {state.isPlaying ? ' Tạm dừng' : isCompleted ? ' Phát lại' : ' Phát'}
          </button>
        </div>
      </div>

      {/* Grid Battlefield: 2 grids side by side */}
      <div className="flex flex-col md:flex-row gap-4">
        {/* Allies Grid */}
        <div className="flex-1 bg-zinc-950/60 border border-emerald-800/40 rounded-xl p-3">
          <h3 className="text-xs font-bold text-emerald-300 mb-2 flex items-center gap-2">
            <Heart className="w-3.5 h-3.5 text-emerald-400" />
            Phe Ta
            <span className="text-[10px] text-zinc-600 ml-auto">{allies.length} chiến binh</span>
          </h3>
          <div className="grid grid-cols-3 gap-1.5">
            {[1, 2, 3].map(gridX => (
              [3, 2, 1].map(gridY => {
                const cellCombatant = Object.values(state.combatantStates).find(
                  c => c.kind === "ally" && c.gridX === gridX && c.gridY === gridY
                );
                return renderGridCell(cellCombatant, gridX, gridY);
              })
            ))}
          </div>
        </div>

        {/* Enemies Grid */}
        <div className="flex-1 bg-zinc-950/60 border border-rose-800/40 rounded-xl p-3">
          <h3 className="text-xs font-bold text-rose-300 mb-2 flex items-center gap-2">
            <Swords className="w-3.5 h-3.5 text-rose-400" />
            Kẻ Địch
            <span className="text-[10px] text-zinc-600 ml-auto">{enemies.length} mục tiêu</span>
          </h3>
          <div className="grid grid-cols-3 gap-1.5">
            {[1, 2, 3].map(gridX => (
              [1, 2, 3].map(gridY => {
                const cellCombatant = Object.values(state.combatantStates).find(
                  c => c.kind === "enemy" && c.gridX === gridX && c.gridY === gridY
                );
                return renderGridCell(cellCombatant, gridX, gridY);
              })
            ))}
          </div>
        </div>
      </div>

      {/* Floating event overlay */}
      {state.events.length > 0 && (
        <div className="fixed inset-0 pointer-events-none z-50">
          {state.events.map(evt => {
            const targetState = evt.targetId ? state.combatantStates[evt.targetId] : null;
            const actorState = evt.actorId ? state.combatantStates[evt.actorId] : null;
            const side = targetState?.kind === "ally" ? "left" :
              targetState?.kind === "enemy" ? "right" :
              actorState?.kind === "ally" ? "left" :
              actorState?.kind === "enemy" ? "right" : "center";
            const align = side === "left" ? "justify-start ml-[25%]" : side === "right" ? "justify-end mr-[25%]" : "justify-center";
            const color = evt.type === "damage" ? "text-red-400" :
              evt.type === "crit" ? "text-yellow-300" :
              evt.type === "heal" ? "text-emerald-400" :
              evt.type === "shield" ? "text-blue-400" :
              evt.type === "death" ? "text-red-500" :
              "text-zinc-300";
            const size = evt.type === "death" ? "text-4xl" :
              evt.type === "crit" ? "text-2xl" :
              "text-lg";
            return (
              <div key={evt.id} className={`absolute top-1/3 left-0 right-0 flex ${align} ${evt.type === "death" ? "animate-ping" : "animate-bounce"}`}>
                <span className={`font-bold ${size} ${color} drop-shadow-[0_0_10px_rgba(0,0,0,0.8)] px-3 py-1 rounded-lg bg-black/40 backdrop-blur-sm`}>
                  {evt.text}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Result Summary */}
      {isCompleted && summary && (
        <div className="bg-black/50 border border-zinc-800 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-3">
            <Swords className="w-4 h-4 text-amber-400" />
            <h3 className="text-sm font-bold text-zinc-200">Kết quả</h3>
            <span className={`ml-auto text-xs px-2 py-0.5 rounded font-bold ${winner === 'Phe Ta' ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-rose-900/50 text-rose-400 border border-rose-800'}`}>
              {winner === 'Phe Ta' ? 'THẮNG LỢI' : 'THẤT BẠI'}
            </span>
          </div>
          <div className="grid grid-cols-3 gap-3 text-xs">
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Sát thương gây</div>
              <div className="font-bold text-lg text-amber-400"><Num value={summary.totalDmgDealt} /></div>
            </div>
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Sát thương nhận</div>
              <div className="font-bold text-lg text-rose-400"><Num value={summary.totalDmgTaken} /></div>
            </div>
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Khiên dùng</div>
              <div className="font-bold text-lg text-blue-400"><Num value={summary.shieldsUsed} /></div>
            </div>
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Mana tiêu hao</div>
              <div className="font-bold text-lg text-purple-400"><Num value={summary.manaCost} /></div>
            </div>
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Số item dùng</div>
              <div className="font-bold text-lg text-emerald-400"><Num value={summary.itemsUsed} /></div>
            </div>
            <div className="bg-zinc-900/60 border border-zinc-800 rounded-lg p-3 text-center">
              <div className="text-zinc-500">Nguyên Thạch dùng</div>
              <div className="font-bold text-lg text-orange-400"><Num value={summary.stonesUsed} /></div>
            </div>
          </div>
        </div>
      )}

      {/* Log Viewer */}
      <div className="bg-zinc-950/80 border border-zinc-800 rounded-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800">
          <h3 className="text-xs font-bold text-zinc-400 flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-zinc-600 animate-pulse" />
            Nhật ký chiến đấu
          </h3>
          <span className="text-[10px] text-zinc-600">
            {actionLogs.length} sự kiện
          </span>
        </div>
        <div ref={logContainerRef} className="overflow-y-auto max-h-[350px] p-3 space-y-1 font-mono text-xs">
          {visibleLogs.length === 0 && (
            <div className="text-zinc-700 text-center py-8 italic">Chưa có dữ liệu</div>
          )}
          {visibleLogs.map((log, idx) => {
            const isTurnStart = log.action === "TURN_START";
            const isBattleBoundary = log.action === "BATTLE_START" || log.action === "BATTLE_END";
            const isAction = ["BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "DAMAGE", "HEAL", "ITEM_HEAL", "SHIELD_GAIN", "DOT_TICK", "DIED", "LIFESTEAL", "STUNNED", "PHANTOM_ATTACK", "THORNS", "SHIELD_ABSORB", "SHIELD_BREAK", "BLEED_APPLIED", "POISON_APPLIED", "STUN_APPLIED", "MANA_RESTORE", "MANA_COST", "PRE_COMBAT_BARRIER", "MULTI_HIT", "LOOT", "AUTO_ITEM_USAGE", "AUTO_STONE_USAGE", "MINE_LOOT", "MINE_OCCUPY", "SHIELD_PIERCE", "BUFF", "DEFEND", "SHIELD_BYPASS_ATTACK", "EFFECT_EXPIRED", "SLOW_APPLIED", "SLOW1_APPLIED", "SLOW_DAMAGE_APPLIED", "WEAK_APPLIED", "FREEZE_APPLIED", "SILENCE_APPLIED", "THORNS_BUFF", "REGEN_BUFF", "ATK_UP_BUFF", "DEF_UP_BUFF", "TAUNT_BUFF", "TRIGGERED", "INSUFFICIENT_MANA", "PATH_MASTERY_GAIN", "SUMMONED", "TRANSFORM", "DISPEL", "INTERRUPT", "CHAOS_GAMBLE", "LIFESTEAL_SHIELD", "EVADE"].includes(log.action);

            if (isTurnStart) {
              return (
                <div key={idx} className="flex items-center gap-2 py-2">
                  <div className="flex-1 h-px bg-zinc-800" />
                  <span className="text-[10px] text-amber-500 font-bold bg-amber-950/20 px-3 py-0.5 rounded-full border border-amber-800/30">
                    LƯỢT {log.turn}
                  </span>
                  <div className="flex-1 h-px bg-zinc-800" />
                </div>
              );
            }

            if (isBattleBoundary) {
              return (
                <div key={idx} className="text-center py-2">
                  <span className={`text-[10px] font-bold ${log.action === "BATTLE_START" ? 'text-emerald-500' : 'text-amber-500'} bg-zinc-900/60 px-4 py-1 rounded-full border border-zinc-800`}>
                    {log.detail}
                  </span>
                </div>
              );
            }

            if (!isAction) return null;

            const baseColor = actionColor(log.action);
            const isCurrent = idx === state.currentIndex;

            return (
              <div
                key={idx}
                className={`flex items-start gap-2 px-2 py-1.5 rounded-lg transition-all ${isCurrent ? 'bg-zinc-800/40 border border-zinc-700/50' : 'hover:bg-zinc-900/30'}`}
              >
                <span className={`mt-0.5 shrink-0 ${baseColor}`}>
                  {actionIcon(log.action)}
                </span>
                
                {log.trigger && (
                  <span className="text-[8px] text-cyan-500 bg-cyan-950/40 px-1 rounded shrink-0">Kích hoạt</span>
                )}
                <span className={`${baseColor} leading-tight ${isCurrent ? 'font-medium' : ''}`}>
                  {log.detail}
                </span>
                {log.damage && log.damage > 0 && !["DOT_TICK", "SHIELD_GAIN", "SHIELD_ABSORB", "HEAL", "ITEM_HEAL", "LIFESTEAL"].includes(log.action) && (
                  <span className="text-red-400 font-bold shrink-0 ml-auto">
                    -<Num value={log.damage} />
                  </span>
                )}
                {log.is_crit && (
                  <span className="text-yellow-400 text-[8px] font-bold bg-yellow-950/40 px-1 rounded shrink-0">CHÍ MẠNG</span>
                )}
              </div>
            );
          })}
          <div ref={logEndRef} />
        </div>
      </div>

      {isCompleted && (
        <button
          onClick={onBack}
          className="w-full bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-bold py-3 rounded-xl transition-all"
        >
          Trở Về
        </button>
      )}

      {renderDetailPopup()}
    </div>
  );
};
