"use client";

import React, { useReducer, useMemo, useRef, useEffect, useCallback } from "react";
import {
  Swords, SkipForward, Heart, Shield, Skull,
  Zap, FlaskConical, Sparkles, Flame, Play, Pause,
  ArrowLeft, ChevronRight, Droplets, Coins, Gem,
  ChevronDown,
} from "lucide-react";
import { formatNumber } from "@/lib/utils";
import { Num } from "./Num";
import { FormattedText } from "./FormattedText";
import { translateMastery, masteryKeyForPoints } from "@/lib/translate";
import { guSpecFor, useGameStore } from "@/store/useGameStore";

export interface StatModSource {
  key: string;
  name: string;
  pct: number;
}

export interface StatProvenance {
  stat: string;
  base_value?: number;
  final_value?: number;
  mods?: StatModSource[];
}

export interface DamageBreakdown {
  base_damage?: number;
  skill_mult?: number;
  dao_mult?: number;
  buff_mult?: number;
  crit_mult?: number;
  path_bonus?: number;
  target_def?: number;
  natural_resist?: number;
  path_resist?: number;
  damage_reduction?: number;
  shield_absorbed?: number;
  hp_damage?: number;
  final_damage?: number;
  raw_damage?: number;
  hit_count?: number;
  reason?: string;
  attacker_attack?: StatProvenance;
  defender_defense?: StatProvenance;
  healer_healing?: StatProvenance;
  shielder_defense?: StatProvenance;
  healing_base?: number;
  healing_dao_bonus?: number;
  healing_final?: number;
  shield_base?: number;
  shield_skill_mult?: number;
  shield_buff_mult?: number;
  shield_dao_mult?: number;
  shield_dao_bonus?: number;
  shield_final?: number;
}

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
  breakdown?: DamageBreakdown;
  snapshot?: {
    turn: number;
    allies?: CombatantSnapshotEntry[];
    enemies?: CombatantSnapshotEntry[];
    aperture_hp?: number;
    aperture_max_hp?: number;
  };
}

interface CombatantSnapshotEntry {
  id: string;
  name: string;
  hp: number;
  max_hp: number;
  mana: number;
  max_mana: number;
  shield: number;
  damage: number;
  defense: number;
  speed: number;
  statuses?: CombatStatusEntry[];
}

interface CombatStatusEntry {
  id: string;
  type: string;
  tier?: number;
  value?: number;
  remaining_turns: number;
  direct_hp?: boolean;
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
  gu_inventory?: Array<Record<string, unknown>>;
  guInventory?: Array<Record<string, unknown>>;
  avatar_url?: string;
}

interface CombatReplayViewProps {
  logs: CombatLogEntry[];
  combatants?: CombatantReplayInfo[];
  winner: string;
  onBack: () => void;
  playerName?: string;
  apertureHp?: number;
  apertureMaxHp?: number;
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
  guInventory?: Array<{ gu_id?: string; guId?: string; rank: number; status?: string }>;
  avatarUrl?: string;
  statuses: CombatStatusEntry[];
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
  apertureHp: number;
  apertureMaxHp: number;
}

type ReplayAction =
  | { type: "INIT"; combatants: Record<string, CombatantState>; apertureHp?: number; apertureMaxHp?: number }
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
        apertureHp: action.apertureHp ?? state.apertureHp,
        apertureMaxHp: action.apertureMaxHp ?? state.apertureMaxHp,
      };

    case "ADVANCE": {
      const log = action.log;
      const next = { ...state.combatantStates };

      if (log.snapshot) {
        for (const [kind, snapshotCombatants] of [["ally", log.snapshot.allies], ["enemy", log.snapshot.enemies]] as const) {
          for (const snapshot of snapshotCombatants || []) {
            const current = next[snapshot.id];
            if (!current) continue;
            next[snapshot.id] = {
              ...current,
              kind,
              hp: snapshot.hp,
              maxHp: snapshot.max_hp,
              mp: snapshot.mana,
              maxMp: snapshot.max_mana,
              shield: snapshot.shield,
              damage: snapshot.damage,
              defense: snapshot.defense,
              speed: snapshot.speed,
              statuses: snapshot.statuses || [],
            };
          }
        }
      }

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

      let apertureHp = state.apertureHp;
      if (log.action === "APERTURE_DAMAGE" && log.damage) {
        apertureHp = Math.max(0, apertureHp - log.damage);
      }

      return {
        ...state,
        combatantStates: next,
        currentIndex: state.currentIndex + 1,
        highlightedActor: log.actor_id || null,
        highlightedTarget: log.target_id || null,
        events,
        apertureHp,
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
    case "GU_BACKLASH": return <Zap className="w-3 h-3 text-red-400" />;
    case "DIED": return <Skull className="w-3 h-3" />;
    case "LIFESTEAL": return <FlaskConical className="w-3 h-3" />;
    case "STUNNED": return <Zap className="w-3 h-3" />;
    case "MINE_LOOT": return <Coins className="w-3 h-3" />;
    case "MINE_OCCUPY": return <Gem className="w-3 h-3" />;
    case "APERTURE_DAMAGE": return <Sparkles className="w-3 h-3" />;
    case "BUFF": return <Sparkles className="w-3 h-3" />;
    case "DEFEND": return <Shield className="w-3 h-3" />;
    default: return <ChevronRight className="w-3 h-3" />;
  }
};

const actionColor = (action: string) => {
  switch (action) {
    case "DAMAGE": case "BASIC_ATTACK": case "SHIELD_BREAK": case "SHIELD_PIERCE": case "SHIELD_BYPASS_ATTACK": return "text-rose-400";
    case "GU_SKILL": return "text-zinc-200";
    case "KILLER_MOVE": return "text-purple-400";
    case "HEAL": case "ITEM_HEAL": return "text-emerald-400";
    case "SHIELD_GAIN": case "SHIELD_ABSORB": case "DEFEND": return "text-cyan-400";
    case "DOT_TICK": return "text-yellow-400";
    case "GU_BACKLASH": return "text-red-400 font-semibold";
    case "DIED": return "text-rose-500";
    case "LIFESTEAL": return "text-pink-400";
    case "BATTLE_START": case "BATTLE_END": return "text-amber-400";
    case "MINE_LOOT": return "text-yellow-400";
    case "MINE_OCCUPY": return "text-emerald-400";
    case "APERTURE_DAMAGE": return "text-amber-400";
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
  apertureHp: initialApertureHp = 0,
  apertureMaxHp: initialApertureMaxHp = 0,
}) => {
  const guDictionary = useGameStore((state) => state.guDictionary);
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
          guInventory: (c.gu_inventory || c.guInventory) as CombatantState["guInventory"],
          avatarUrl: c.avatar_url,
          statuses: [],
        };
      }
      return map;
    }

    // Check log snapshot if initialCombatants is empty
    for (const log of logs) {
      const snap = (log as unknown as Record<string, unknown>).snapshot as
        | { allies?: Record<string, unknown>[]; enemies?: Record<string, unknown>[] }
        | undefined;
      if (snap && (snap.allies || snap.enemies)) {
        const map: Record<string, CombatantState> = {};
        if (Array.isArray(snap.allies)) {
          snap.allies.forEach((a, idx) => {
            const id = String(a.id || `ally_${idx}`);
            map[id] = {
              id,
              name: String(a.name || "Phe Ta"),
              kind: "ally",
              maxHp: Number(a.max_hp || a.maxHp || 1000),
              hp: Number(a.hp ?? a.max_hp ?? 1000),
              maxMp: Number(a.max_mana || a.maxMana || 0),
              mp: Number(a.mana ?? a.current_mana ?? 0),
              shield: Number(a.shield || 0),
              gridX: Number(a.grid_x && Number(a.grid_x) >= 1 && Number(a.grid_x) <= 3 ? a.grid_x : (idx % 3) + 1),
              gridY: Number(a.grid_y && Number(a.grid_y) >= 1 && Number(a.grid_y) <= 3 ? a.grid_y : Math.floor(idx / 3) + 1),
              damage: Number(a.damage || 0),
              defense: Number(a.defense || 0),
              speed: Number(a.speed || 0),
              kindDetail: String(a.kind || ""),
              rank: Number(a.rank || 1),
              subRank: Number(a.sub_rank || 0),
              avatarUrl: (a.avatar_url as string) || undefined,
              statuses: (Array.isArray(a.statuses) ? a.statuses : []) as CombatStatusEntry[],
            };
          });
        }
        if (Array.isArray(snap.enemies)) {
          snap.enemies.forEach((e, idx) => {
            const id = String(e.id || `enemy_${idx}`);
            map[id] = {
              id,
              name: String(e.name || "Kẻ Địch"),
              kind: "enemy",
              maxHp: Number(e.max_hp || e.maxHp || 1000),
              hp: Number(e.hp ?? e.max_hp ?? 1000),
              maxMp: Number(e.max_mana || e.maxMana || 0),
              mp: Number(e.mana ?? e.current_mana ?? 0),
              shield: Number(e.shield || 0),
              gridX: Number(e.grid_x && Number(e.grid_x) >= 1 && Number(e.grid_x) <= 3 ? e.grid_x : (idx % 3) + 1),
              gridY: Number(e.grid_y && Number(e.grid_y) >= 1 && Number(e.grid_y) <= 3 ? e.grid_y : Math.floor(idx / 3) + 1),
              damage: Number(e.damage || 0),
              defense: Number(e.defense || 0),
              speed: Number(e.speed || 0),
              kindDetail: String(e.kind || ""),
              rank: Number(e.rank || 1),
              subRank: Number(e.sub_rank || 0),
              avatarUrl: (e.avatar_url as string) || undefined,
              statuses: (Array.isArray(e.statuses) ? e.statuses : []) as CombatStatusEntry[],
            };
          });
        }
        if (Object.keys(map).length > 0) return map;
      }
    }

    const map: Record<string, CombatantState> = {};
    let allyCount = 0;
    let enemyCount = 0;

    for (const log of logs) {
      const isActorAlly = playerName && (log.actor_name === playerName || log.actor_id === playerName);
      const isTargetAlly = playerName && (log.target_name === playerName || log.target_id === playerName);

      if (log.actor_id && !map[log.actor_id]) {
        const isAlly = Boolean(isActorAlly);
        const idx = isAlly ? allyCount++ : enemyCount++;
        map[log.actor_id] = {
          id: log.actor_id,
          name: log.actor_name || "???",
          kind: isAlly ? "ally" : "enemy",
          maxHp: 1000,
          hp: 1000,
          maxMp: 0,
          mp: 0,
          shield: 0,
          gridX: (idx % 3) + 1,
          gridY: Math.floor(idx / 3) + 1,
          damage: 0,
          defense: 0,
          speed: 0,
          kindDetail: "",
          statuses: [],
          rank: 0,
          subRank: 0,
        };
      }
      if (log.target_id && !map[log.target_id]) {
        const isAlly = Boolean(isTargetAlly);
        const idx = isAlly ? allyCount++ : enemyCount++;
        map[log.target_id] = {
          id: log.target_id,
          name: log.target_name || log.target_id,
          kind: isAlly ? "ally" : "enemy",
          maxHp: 1000,
          hp: 1000,
          maxMp: 0,
          mp: 0,
          shield: 0,
          gridX: (idx % 3) + 1,
          gridY: Math.floor(idx / 3) + 1,
          damage: 0,
          defense: 0,
          speed: 0,
          kindDetail: "",
          statuses: [],
          rank: 0,
          subRank: 0,
        };
      }
    }
    return map;
  }, [initialCombatants, logs, playerName]);

  const [state, dispatch] = useReducer(replayReducer, {
    combatantStates: parsedCombatants,
    currentIndex: 0,
    isPlaying: false,
    currentTurn: 0,
    highlightedActor: null,
    highlightedTarget: null,
    events: [],
    apertureHp: initialApertureHp,
    apertureMaxHp: initialApertureMaxHp,
  });

  // Re-initialize when combatants change
  useEffect(() => {
    dispatch({ type: "INIT", combatants: parsedCombatants, apertureHp: initialApertureHp, apertureMaxHp: initialApertureMaxHp });

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
      dispatch({ type: "INIT", combatants: parsedCombatants, apertureHp: initialApertureHp, apertureMaxHp: initialApertureMaxHp });
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
  const [isVitalStatsExpanded, setIsVitalStatsExpanded] = React.useState(false);
  const [isDaoMarksExpanded, setIsDaoMarksExpanded] = React.useState(false);
  const [isPathMasteriesExpanded, setIsPathMasteriesExpanded] = React.useState(false);
  const [isGuInventoryExpanded, setIsGuInventoryExpanded] = React.useState(false);

  const renderGridCell = (c: CombatantState | undefined, gridX: number, gridY: number) => {
    const isAlly = c?.kind === "ally";
    const isEmpty = !c;
    const isHighlighted = state.highlightedActor === c?.id;
    const isTargeted = state.highlightedTarget === c?.id;
    const pct = c ? hpPct(c) : 0;

    const borderGlow = isHighlighted
      ? "border-emerald-400 shadow-[0_0_16px_rgba(52,211,153,0.45)] ring-1 ring-emerald-400/50"
      : isTargeted
      ? "border-rose-400 shadow-[0_0_16px_rgba(244,63,94,0.45)] ring-1 ring-rose-400/50"
      : "";

    return (
      <div
        key={`${gridX}-${gridY}`}
        onClick={() => c && setDetailCombatant(detailCombatant?.id === c.id ? null : c)}
        className={`rounded-xl border ${
          isEmpty
            ? 'border-zinc-800/40 bg-zinc-950/30'
            : c.hp <= 0
              ? 'border-zinc-800/50 bg-zinc-950/70 opacity-40'
              : isAlly
                ? `border-emerald-800/50 bg-gradient-to-b from-emerald-950/30 to-zinc-950/90 shadow-sm ${borderGlow}`
                : `border-rose-800/50 bg-gradient-to-b from-rose-950/30 to-zinc-950/90 shadow-sm ${borderGlow}`
        } flex flex-col justify-between p-2 sm:p-2.5 relative cursor-pointer transition-all hover:brightness-125 hover:border-zinc-500/50 min-h-[145px] sm:min-h-[160px]`}
      >
        {c ? (
          <>
            {/* Header: Avatar + Name + Badges */}
            <div className="flex items-center gap-2 w-full min-w-0">
              {/* Avatar */}
              <div className={`w-9 h-9 sm:w-10 sm:h-10 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-xs font-black text-zinc-300 border-2 ${
                isAlly ? 'border-emerald-500/60 shadow-emerald-950/50' : 'border-rose-500/60 shadow-rose-950/50'
              } shadow-md`}>
                {c.avatarUrl ? (
                  <img src={c.avatarUrl} alt="" className="w-full h-full object-cover" />
                ) : (
                  c.name.slice(0, 2).toUpperCase()
                )}
              </div>

              {/* Name & Badges */}
              <div className="min-w-0 flex-1 flex flex-col justify-center">
                <div className="flex items-center gap-1 min-w-0">
                  <span className={`text-[11px] sm:text-xs font-bold truncate ${
                    c.hp <= 0 ? 'text-zinc-600 line-through' : isAlly ? 'text-emerald-300' : 'text-rose-300'
                  }`}>
                    {c.name}
                  </span>
                  {c.hp <= 0 && <span className="text-xs text-rose-500 font-bold shrink-0">†</span>}
                </div>

                <div className="flex items-center gap-1 mt-0.5 flex-wrap">
                  {c.rank ? (
                    <span className="text-[8px] sm:text-[9px] px-1 py-0.2 rounded bg-zinc-800/90 text-amber-300 font-mono font-bold border border-amber-500/30">
                      {c.rank >= 6 ? `Chuyển ${c.rank}` : `H${c.rank}`}
                    </span>
                  ) : null}
                  {!isAlly && c.kindDetail && c.kindDetail !== "" && (
                    <span className={`text-[8px] sm:text-[9px] px-1 py-0.2 rounded border font-bold ${
                      c.kindDetail === "Boss" ? 'bg-red-950/80 text-red-300 border-red-700' :
                      c.kindDetail === "Elite" ? 'bg-purple-950/80 text-purple-300 border-purple-700' :
                      'bg-zinc-800/80 text-zinc-400 border-zinc-700'
                    }`}>
                      {c.kindDetail.toUpperCase()}
                    </span>
                  )}
                </div>
              </div>
            </div>

            {/* Vital Bars Section (HP, Shield, Mana) */}
            <div className="w-full space-y-1.5 my-1.5">
              {/* HP Bar & Value */}
              <div>
                <div className="flex justify-between items-center text-[10px] font-mono leading-tight mb-0.5">
                  <span className="text-zinc-400 font-semibold flex items-center gap-0.5">
                    <Heart className="w-2.5 h-2.5 text-rose-400 shrink-0" />
                    HP
                  </span>
                  <span className={`font-bold ${c.hp <= 0 ? 'text-zinc-600' : 'text-zinc-200'}`}>
                    <Num value={Math.round(c.hp)} /> / <Num value={Math.round(c.maxHp)} />
                  </span>
                </div>
                <div className="w-full h-1.5 bg-zinc-900 rounded-full overflow-hidden border border-zinc-800">
                  <div
                    className={`h-full rounded-full transition-all duration-300 ${
                      c.hp <= 0
                        ? 'bg-zinc-700'
                        : isAlly
                        ? 'bg-gradient-to-r from-emerald-600 to-emerald-400'
                        : 'bg-gradient-to-r from-rose-600 to-rose-400'
                    }`}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>

              {/* Shield Bar (if active) */}
              {c.shield > 0 && (
                <div className="flex items-center justify-between text-[9px] font-mono font-bold text-cyan-300 bg-cyan-950/50 border border-cyan-800/60 px-1.5 py-0.5 rounded">
                  <span className="flex items-center gap-1">
                    <Shield className="w-2.5 h-2.5 text-cyan-400" /> Khiên
                  </span>
                  <span>+<Num value={Math.round(c.shield)} /></span>
                </div>
              )}

              {c.statuses.length > 0 && (
                <div className="flex flex-wrap gap-1 pt-1" aria-label="Trạng thái hiện tại">
                  {c.statuses.slice(0, 3).map((status, statusIndex) => (
                    <span key={`${status.id}-${statusIndex}`} className={`max-w-full truncate rounded border px-1 py-0.5 text-[8px] ${status.type === "buff" ? "border-emerald-800/60 bg-emerald-950/40 text-emerald-300" : "border-rose-800/60 bg-rose-950/40 text-rose-300"}`} title={`${status.id} · còn ${status.remaining_turns} lượt`}>
                      {status.id.replaceAll("_", " ")} · {status.remaining_turns}L
                    </span>
                  ))}
                  {c.statuses.length > 3 && <span className="text-[8px] text-zinc-500">+{c.statuses.length - 3}</span>}
                </div>
              )}

              {/* Mana Bar */}
              {c.maxMp > 0 && (isAlly || (c.kindDetail && c.kindDetail.toLowerCase() === "player")) && (
                <div>
                  <div className="flex justify-between items-center text-[9px] font-mono leading-tight mb-0.5">
                    <span className="text-purple-400/90 font-medium flex items-center gap-0.5">
                      <Zap className="w-2.5 h-2.5 text-purple-400 shrink-0" />
                      {c.rank >= 6 ? "Tiên Nguyên" : "MP"}
                    </span>
                    <span className="text-purple-300 font-bold">
                      {c.rank >= 6 ? (
                        <Num value={Math.round(c.mp)} />
                      ) : (
                        <><Num value={Math.round(c.mp)} /> / <Num value={Math.round(c.maxMp)} /></>
                      )}
                    </span>
                  </div>
                  <div className="w-full h-1 bg-zinc-900 rounded-full overflow-hidden border border-zinc-800">
                    <div
                      className="h-full rounded-full bg-gradient-to-r from-purple-600 to-indigo-400 transition-all duration-300"
                      style={{ width: `${c.rank >= 6 ? 100 : c.maxMp > 0 ? Math.min(100, (c.mp / c.maxMp) * 100) : 0}%` }}
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Combat Stats Footer (Công ⚔, Thủ 🛡, Tốc ⚡) */}
            <div className="grid grid-cols-3 gap-1 pt-1.5 border-t border-zinc-800/60 w-full mt-auto text-center text-[9px] font-mono font-bold">
              <div
                className="bg-rose-950/40 border border-rose-900/40 text-rose-300 rounded py-0.5 px-0.5 truncate"
                title={`Công kích: ${c.damage.toLocaleString()}`}
              >
                ⚔ <Num value={Math.round(c.damage)} />
              </div>
              <div
                className="bg-blue-950/40 border border-blue-900/40 text-blue-300 rounded py-0.5 px-0.5 truncate"
                title={`Phòng thủ: ${c.defense.toLocaleString()}`}
              >
                🛡 <Num value={Math.round(c.defense)} />
              </div>
              <div
                className="bg-amber-950/40 border border-amber-900/40 text-amber-300 rounded py-0.5 px-0.5 truncate"
                title={`Tốc độ: ${c.speed.toLocaleString()}`}
              >
                ⚡ <Num value={Math.round(c.speed)} />
              </div>
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center h-full my-auto text-zinc-700/60 font-mono text-[10px]">
            <span className="opacity-40">{gridX},{gridY}</span>
          </div>
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
        className="fixed inset-0 z-50 h-[100dvh] flex items-center justify-center bg-black/60 px-4 pt-[calc(env(safe-area-inset-top)+1rem)] pb-[calc(env(safe-area-inset-bottom)+1rem)] backdrop-blur-sm"
        onClick={() => setDetailCombatant(null)}
      >
        <div
          className={`bg-zinc-900 border ${isAlly ? 'border-emerald-800/50' : 'border-rose-800/50'} rounded-2xl p-5 shadow-2xl max-h-full max-w-sm w-full mx-4 overflow-y-auto`}
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
            <div className="mt-2 bg-zinc-950/60 rounded-xl p-3">
              <button
                type="button"
                aria-expanded={isVitalStatsExpanded}
                onClick={() => setIsVitalStatsExpanded((expanded) => !expanded)}
                className="flex w-full items-center gap-1.5 text-left text-zinc-400 text-[10px] font-semibold"
              >
                {isVitalStatsExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                Chỉ Số Sinh Tồn
              </button>
              {isVitalStatsExpanded && <div className="grid grid-cols-2 gap-2 mt-2">
              <div><span className="text-zinc-500">Sinh lực</span><br /><span className="text-emerald-400 font-bold text-sm"><Num value={Math.round(c.hp)} /> / <Num value={Math.round(c.maxHp)} /></span></div>
              <div>
                <span className="text-zinc-500">{c.rank >= 6 ? "Tiên nguyên" : "Chân nguyên"}</span><br />
                <span className="text-purple-400 font-bold text-sm">
                  {c.rank >= 6 ? (
                    <Num value={Math.round(c.mp)} />
                  ) : (
                    <><Num value={Math.round(c.mp)} /> / <Num value={Math.round(c.maxMp)} /></>
                  )}
                </span>
              </div>
              <div><span className="text-zinc-500">Tấn công</span><br /><span className="text-rose-400 font-bold"><Num value={c.damage} /></span></div>
              <div><span className="text-zinc-500">Phòng thủ</span><br /><span className="text-blue-400 font-bold"><Num value={c.defense} /></span></div>
              <div><span className="text-zinc-500">Tốc độ</span><br /><span className="text-cyan-400 font-bold"><Num value={c.speed} /></span></div>
              <div><span className="text-zinc-500">Khiên</span><br /><span className="text-indigo-400 font-bold"><Num value={c.shield} /></span></div>
              </div>}
            </div>

            {/* Position */}
            <div className="text-zinc-500">
              Vị trí: <span className="text-zinc-300 font-mono">({c.gridX},{c.gridY})</span>
            </div>

            {c.statuses.length > 0 && <div className="border-t border-zinc-800 pt-2 mt-2">
              <div className="mb-1 text-[10px] text-zinc-500">Trạng thái ở lượt {state.currentTurn}</div>
              <div className="space-y-1">
                {c.statuses.map((status, index) => <div key={`${status.id}-${index}`} className="flex items-center justify-between gap-2 text-[10px]">
                  <span className={status.type === "buff" ? "text-emerald-300" : status.type === "dot" ? "text-amber-300" : "text-rose-300"}>
                    {status.id.replaceAll("_", " ").toUpperCase()}{status.tier ? ` · T${status.tier}` : ""}
                    {status.value ? ` · ${status.type === "dot" ? `${status.value} / lượt` : `GT ${status.value}`}` : ""}
                    {status.direct_hp ? " · xuyên khiên" : ""}
                  </span>
                  <span className="shrink-0 text-zinc-500">còn {status.remaining_turns} lượt</span>
                </div>)}
              </div>
            </div>}

            {/* Dao Marks */}
            {c.daoMarks && Object.keys(c.daoMarks).length > 0 && (
              <div className="border-t border-zinc-800 pt-2 mt-2">
                <button
                  type="button"
                  aria-expanded={isDaoMarksExpanded}
                  onClick={() => setIsDaoMarksExpanded((expanded) => !expanded)}
                  className="flex items-center gap-1.5 text-left text-zinc-500 text-[10px]"
                >
                  {isDaoMarksExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                  Đạo Ấn:
                </button>
                {isDaoMarksExpanded && <div className="flex flex-wrap gap-1 mt-1">
                  {Object.entries(c.daoMarks).map(([k, v]) => (
                    <span key={k} className="text-[9px] px-1.5 py-0.5 rounded bg-amber-950/30 text-amber-400 border border-amber-800/40 font-mono">
                      {k}: {v}
                    </span>
                  ))}
                </div>}
              </div>
            )}

            {/* Path Masteries */}
            {c.pathMasteries && Object.keys(c.pathMasteries).length > 0 && (
              <div className="border-t border-zinc-800 pt-2 mt-2">
                <button
                  type="button"
                  aria-expanded={isPathMasteriesExpanded}
                  onClick={() => setIsPathMasteriesExpanded((expanded) => !expanded)}
                  className="flex items-center gap-1.5 text-left text-zinc-500 text-[10px]"
                >
                  {isPathMasteriesExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                  Đường Tu:
                </button>
                {isPathMasteriesExpanded && <div className="flex flex-wrap gap-1 mt-1">
                  {Object.entries(c.pathMasteries).map(([k, points]) => (
                    <span key={k} className="text-[9px] px-1.5 py-0.5 rounded bg-cyan-950/30 text-cyan-400 border border-cyan-800/40 font-mono">
                      {k}: {translateMastery(masteryKeyForPoints(points as number))}
                    </span>
                  ))}
                </div>}
              </div>
            )}

            {/* Equipped Gu List */}
            {c.guInventory && c.guInventory.length > 0 && (
              <div className="border-t border-zinc-800 pt-2 mt-2">
                <button
                  type="button"
                  aria-expanded={isGuInventoryExpanded}
                  onClick={() => setIsGuInventoryExpanded((expanded) => !expanded)}
                  className="flex items-center gap-1.5 text-left text-zinc-500 text-[10px]"
                >
                  {isGuInventoryExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                  Cổ Trùng Trang Bị ({c.guInventory.length}):
                </button>
                {isGuInventoryExpanded && <div className="flex flex-wrap gap-1.5 mt-1">
                  {c.guInventory.map((g, idx: number) => {
                    const item = g as Record<string, unknown>;
                    const rawId = String(item.gu_id || item.guId || item.GuID || item.id || "");
                    const rankVal = Number(item.rank || g.rank || 1);
                    const specName = guSpecFor(rawId, rankVal)?.name || String(item.gu_name || item.name || item.GuName || rawId || "Cổ trùng");
                    const isImmortal = rankVal >= 6;
                    return (
                      <span key={idx} className={`text-[10px] px-2 py-0.5 rounded-md font-medium border ${
                        isImmortal
                          ? 'bg-amber-950/40 text-amber-300 border-amber-700/60 shadow-[0_0_8px_rgba(245,158,11,0.2)]'
                          : 'bg-purple-950/40 text-purple-300 border-purple-800/60'
                      }`}>
                        {isImmortal ? '✦ ' : ''}{specName} <span className="text-[9px] opacity-75 font-mono">(R{rankVal})</span>
                      </span>
                    );
                  })}
                </div>}
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

  const rewards = useMemo(() => {
    return logs.filter(log => ["LOOT", "ARENA_REWARD", "ARENA_RANK_UP", "MINE_LOOT", "MINE_OCCUPY", "PATH_MASTERY_GAIN"].includes(log.action));
  }, [logs]);

  const losses = useMemo(() => {
    return logs.filter(log => {
      if (log.action === "PET_DEATH") return true;
      if (log.action === "BATTLE_END" && log.detail?.includes("rơi rớt mất")) return true;
      return false;
    });
  }, [logs]);

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
      ["BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "DAMAGE", "HEAL", "ITEM_HEAL", "SHIELD_GAIN", "DOT_TICK", "DIED", "LIFESTEAL", "STUNNED", "PHANTOM_ATTACK", "THORNS", "SHIELD_ABSORB", "SHIELD_BREAK", "BLEED_APPLIED", "POISON_APPLIED", "STUN_APPLIED", "MANA_RESTORE", "MANA_COST", "PRE_COMBAT_BARRIER", "MULTI_HIT", "LOOT", "AUTO_ITEM_USAGE", "AUTO_STONE_USAGE", "MINE_LOOT", "MINE_OCCUPY", "APERTURE_DAMAGE"].includes(log.action)
    ), [visibleLogs]);

  return (
    <div className="w-full mx-auto flex flex-col gap-4">
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

      {/* Aperture (Tiên Khiếu) HP — tribulation combat only */}
      {initialApertureMaxHp > 0 && (
        <div className="bg-black/50 border border-amber-800/50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-amber-400" />
              <span className="text-sm font-bold text-amber-300">Máu Tiên Khiếu</span>
            </div>
            <span className="text-xs font-mono text-amber-200">
              {state.apertureHp.toLocaleString()} / {state.apertureMaxHp.toLocaleString()}
            </span>
          </div>
          <div className="h-3 w-full bg-zinc-900 rounded-full overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-amber-500 to-orange-500 rounded-full transition-all"
              style={{ width: `${state.apertureMaxHp > 0 ? Math.max(0, Math.min(100, (state.apertureHp / state.apertureMaxHp) * 100)) : 0}%` }}
            />
          </div>
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

          {/* Rewards & Gained */}
          {rewards.length > 0 && (
            <div className="mt-4 border-t border-zinc-800 pt-3">
              <div className="text-xs font-semibold text-emerald-400 mb-1.5 flex items-center gap-1">
                <Sparkles className="w-3.5 h-3.5" />
                Nhận được:
              </div>
              <div className="space-y-1 bg-emerald-950/20 border border-emerald-900/30 rounded-lg p-2.5">
                {rewards.map((r, idx) => (
                  <div key={idx} className="text-xs text-emerald-300 flex items-start gap-1.5">
                    <span className="text-emerald-500">•</span>
                    <span>{r.detail}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Losses */}
          {winner !== 'Phe Ta' && losses.length > 0 && (
            <div className="mt-4 border-t border-zinc-800 pt-3">
              <div className="text-xs font-semibold text-rose-400 mb-1.5 flex items-center gap-1">
                <Skull className="w-3.5 h-3.5" />
                Tổn thất:
              </div>
              <div className="space-y-1 bg-rose-950/20 border border-rose-900/30 rounded-lg p-2.5">
                {losses.map((l, idx) => (
                  <div key={idx} className="text-xs text-rose-300 flex items-start gap-1.5">
                    <span className="text-rose-500">•</span>
                    <span>{l.detail}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {isCompleted && (
        <button
          onClick={onBack}
          className="w-full bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-bold py-3 rounded-xl transition-all"
        >
          Trở Về
        </button>
      )}

      {/* Grid Battlefield: 2 grids side by side */}
      <div className="flex flex-col md:flex-row gap-4">
        {/* Allies Grid */}
        <div className="flex-1 bg-zinc-950/60 border border-emerald-800/40 rounded-xl p-3">
          <h3 className="text-xs font-bold text-emerald-300 mb-2 flex items-center gap-2">
            <Heart className="w-3.5 h-3.5 text-emerald-400" />
            Phe Ta
            <span className="text-[10px] text-zinc-600 ml-auto">{allies.length} chiến binh</span>
          </h3>
          <div className="grid grid-cols-3 gap-2 sm:gap-2.5">
            {[1, 2, 3].map((gridX) =>
              [3, 2, 1].map((gridY) => {
                const list = Object.values(state.combatantStates).filter((c) => c.kind === "ally");
                const cellCombatant =
                  list.find((c) => c.gridX === gridX && c.gridY === gridY) ||
                  list.find((c, idx) => {
                    const gx = c.gridX >= 1 && c.gridX <= 3 ? c.gridX : (idx % 3) + 1;
                    const gy = c.gridY >= 1 && c.gridY <= 3 ? c.gridY : Math.floor(idx / 3) + 1;
                    return gx === gridX && gy === gridY;
                  });
                return renderGridCell(cellCombatant, gridX, gridY);
              })
            )}
          </div>
        </div>

        {/* Enemies Grid */}
        <div className="flex-1 bg-zinc-950/60 border border-rose-800/40 rounded-xl p-3">
          <h3 className="text-xs font-bold text-rose-300 mb-2 flex items-center gap-2">
            <Swords className="w-3.5 h-3.5 text-rose-400" />
            Kẻ Địch
            <span className="text-[10px] text-zinc-600 ml-auto">{enemies.length} mục tiêu</span>
          </h3>
          <div className="grid grid-cols-3 gap-2 sm:gap-2.5">
            {[1, 2, 3].map((gridX) =>
              [1, 2, 3].map((gridY) => {
                const list = Object.values(state.combatantStates).filter((c) => c.kind === "enemy");
                const cellCombatant =
                  list.find((c) => c.gridX === gridX && c.gridY === gridY) ||
                  list.find((c, idx) => {
                    const gx = c.gridX >= 1 && c.gridX <= 3 ? c.gridX : (idx % 3) + 1;
                    const gy = c.gridY >= 1 && c.gridY <= 3 ? c.gridY : Math.floor(idx / 3) + 1;
                    return gx === gridX && gy === gridY;
                  });
                return renderGridCell(cellCombatant, gridX, gridY);
              })
            )}
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
            const isAction = ["BASIC_ATTACK", "GU_SKILL", "KILLER_MOVE", "DAMAGE", "HEAL", "ITEM_HEAL", "SHIELD_GAIN", "DOT_TICK", "DIED", "LIFESTEAL", "STUNNED", "PHANTOM_ATTACK", "THORNS", "SHIELD_ABSORB", "SHIELD_BREAK", "BLEED_APPLIED", "POISON_APPLIED", "STUN_APPLIED", "MANA_RESTORE", "MANA_COST", "PRE_COMBAT_BARRIER", "MULTI_HIT", "LOOT", "AUTO_ITEM_USAGE", "AUTO_STONE_USAGE", "MINE_LOOT", "MINE_OCCUPY", "SHIELD_PIERCE", "BUFF", "DEFEND", "SHIELD_BYPASS_ATTACK", "EFFECT_EXPIRED", "SLOW_APPLIED", "SLOW1_APPLIED", "SLOW_DAMAGE_APPLIED", "WEAK_APPLIED", "FREEZE_APPLIED", "SILENCE_APPLIED", "THORNS_BUFF", "REGEN_BUFF", "ATK_UP_BUFF", "DEF_UP_BUFF", "TAUNT_BUFF", "TRIGGERED", "INSUFFICIENT_MANA", "PATH_MASTERY_GAIN", "SUMMONED", "TRANSFORM", "DISPEL", "INTERRUPT", "CHAOS_GAMBLE", "LIFESTEAL_SHIELD", "EVADE", "GU_BACKLASH"].includes(log.action);

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

            if (log.action === "TURN_END") {
              return <div key={idx} className={`py-1 text-center text-[9px] ${idx === state.currentIndex ? "text-cyan-300" : "text-zinc-600"}`}>
                Đã ghi nhận chỉ số và trạng thái cuối lượt {log.turn}
              </div>;
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
                  <FormattedText text={log.detail ?? ""} />
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



      {renderDetailPopup()}
    </div>
  );
};
