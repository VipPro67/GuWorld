"use client";

import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { Player, CombatState, Combatant, GuSpec, GuInstance, useGameStore, guSpecFor } from "@/store/useGameStore";
import { Sparkles, Play, Swords, Eye, Shield, Zap, ChevronDown, ChevronUp, Wand2, AlertTriangle, PawPrint, Users } from "lucide-react";
import { CombatReplayView } from "./CombatReplayView";
import { CombatReportView } from "./CombatReportView";
import { Num } from "./Num";
import { getPathColor } from "@/lib/utils";
import { masteryKeyForPoints, masteryLevelForPoints, pathLabels } from "@/lib/translate";
import { GamePanel, GameButton, GameCard, GameProgressBar, GameSection } from "@/components/ui";
import { rankLabel, subRankLabel, typeColor, gridXLabel, getCombatEffects } from "@/lib/combat";
import { captureBonus, investigation, isCombatEligible, operationsBy } from "@/lib/effect-blocks";
// UI migrated to shared components

interface CombatScreenProps {
  player: Player;
  enemy: CombatState;
  guDictionary: Record<string, GuSpec>;
  actionLoading: boolean;
  isPracticeMode?: boolean;
  practiceDummyConfig?: ApiResponseData;
  onPracticeDummyConfigChange?: (config: ApiResponseData) => void;
  onAttack?: () => Promise<unknown>;
  onRetreat?: () => Promise<ApiResponseData>;
}

export const CombatScreen: React.FC<CombatScreenProps> = ({
  player,
  enemy,
  guDictionary,
  actionLoading,
  isPracticeMode,
  practiceDummyConfig,
  onPracticeDummyConfigChange,
  onRetreat,
}) => {
  const [selectedGuIds, setSelectedGuIds] = useState<string[]>([]);

  const [simulationLogs, setSimulationLogs] = useState<Record<string, unknown>[]>([]);
  const [simulationWinner, setSimulationWinner] = useState<string | null>(null);
  const [simulationCombatants, setSimulationCombatants] = useState<Record<string, unknown>[]>([]);

  const [autoHealHP, setAutoHealHP] = useState(false);
  const [healThreshold, setHealThreshold] = useState(30);
  const [autoRestoreMana, setAutoRestoreMana] = useState(false);
  const [manaThreshold, setManaThreshold] = useState(30);
  const [guSortMode, setGuSortMode] = useState<"rank" | "path">("rank");
  const [showRecommendation, setShowRecommendation] = useState(true);
  const [settingsLoaded, setSettingsLoaded] = useState(false);
  const autoStartEncounterRef = useRef<string | null>(null);
  const autoStartTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startSimulationRef = useRef<() => Promise<boolean>>(async () => false);
  const actionLoadingRef = useRef(actionLoading);
  actionLoadingRef.current = actionLoading;

  const executeAutoCombat = useGameStore((state) => state.executeAutoCombat);
  const executePracticeCombat = useGameStore((state) => state.executePracticeCombat);
  const resolveTribulation = useGameStore((state) => state.resolveTribulation);
  const fetchPlayerStatus = useGameStore((state) => state.fetchPlayerStatus);
  const fetchMineSlots = useGameStore((state) => state.fetchMineSlots);
  const combatSettings = useGameStore((state) => state.combatSettings);
  const fetchCombatSettings = useGameStore((state) => state.fetchCombatSettings);
  const addToast = useGameStore((state) => state.addToast);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);
  const party = useGameStore((state) => state.party);
  const fetchPartyStatus = useGameStore((state) => state.fetchPartyStatus);
  const activeCombatReplay = useGameStore((state) => state.activeCombatReplay);

  const isPartyLeader = party ? party.leader_id === player.id : true;
  const isPartyMemberNotLeader = Boolean(player.party_id) && (party === null || !isPartyLeader);

  useEffect(() => {
    if (activeCombatReplay && activeCombatReplay.logs && activeCombatReplay.logs.length > 0) {
      setSimulationLogs(activeCombatReplay.logs);
      setSimulationWinner(activeCombatReplay.winner || null);
      setSimulationCombatants(activeCombatReplay.combatants || []);
    }
  }, [activeCombatReplay]);

  useEffect(() => {
    if (!isPracticeMode && !player?.active_combat && (!activeCombatReplay || !activeCombatReplay.logs || activeCombatReplay.logs.length === 0) && simulationLogs.length === 0) {
      void handleExitSimulation();
    }
  }, [player?.active_combat, activeCombatReplay, simulationLogs.length, isPracticeMode]);

  useEffect(() => {
    if (player.party_id && !party) {
      fetchPartyStatus();
    }
  }, [player.party_id, party, fetchPartyStatus]);

  const notifiedPartyMemberRef = useRef(false);
  useEffect(() => {
    if (isPartyMemberNotLeader && !notifiedPartyMemberRef.current) {
      notifiedPartyMemberRef.current = true;
      addToast("Bạn đang trong tổ đội. Trưởng nhóm sẽ điều khiển trận chiến.", "info");
    }
  }, [isPartyMemberNotLeader, addToast]);

  // Pre-populate auto settings from persistent combat config
  const combatSettingsFetched = useRef(false);
  useEffect(() => {
    if (!combatSettings && !combatSettingsFetched.current) {
      combatSettingsFetched.current = true;
      fetchCombatSettings().finally(() => setSettingsLoaded(true));
    } else if (combatSettings) {
      setSettingsLoaded(true);
      setAutoHealHP(combatSettings.auto_heal_hp);
      setHealThreshold(combatSettings.heal_threshold);
      setAutoRestoreMana(combatSettings.auto_restore_mana);
      setManaThreshold(combatSettings.mana_threshold);
    }
  }, [combatSettings]);

  const isCombatGu = useCallback((gu: GuInstance): boolean => {
    const spec = guSpecFor(gu.gu_id, gu.rank);
    if (!spec) return false;
    return isCombatEligible(spec.effects);
  }, [guDictionary]);

  const eligibleGus = useMemo(
    () => (player.gu_inventory || []).filter((gu) => gu.status === "ACTIVE" && gu.equipped && isCombatGu(gu)),
    [player.gu_inventory, isCombatGu]
  );

  const investigationLevel = useMemo(() => {
    let inv = 1.0;
    (player.gu_inventory || []).forEach((gu) => {
      if (gu.status !== "ACTIVE" || !gu.equipped) return;
      const spec = guSpecFor(gu.gu_id, gu.rank);
      if (!spec) return;
      const ops = operationsBy(spec.effects, { trigger: "ALWAYS_ON", op: "STAT_MOD", stat: "INVESTIGATION" });
      for (const op of ops) inv += op.value ?? 0;
    });
    return inv;
  }, [player.gu_inventory, guDictionary]);

  const enemyInfoRevealed = useMemo(() => {
    if (!enemy) return { showHP: false, showStats: false, showLootHints: false };
    const inv = investigationLevel;
    return {
      showHP: inv >= 2,
      showStats: inv >= 4,
      showLootHints: inv >= 8,
    };
  }, [investigationLevel, enemy]);

  const handleStartSimulation = async (): Promise<boolean> => {
    const tribulationEventId = isPracticeMode ? undefined : player.active_combat?.tribulation_event_id;
    const loadout = {
      GuIds: selectedGuIds,
      ImmortalGuId: null,
      AutoHealHP: autoHealHP,
      HealThreshold: healThreshold,
      AutoRestoreMana: autoRestoreMana,
      ManaThreshold: manaThreshold,
    };

    let result;
    if (isPracticeMode) {
      result = await executePracticeCombat(loadout, practiceDummyConfig);
    } else {
      result = await executeAutoCombat(loadout);
    }

    if (result && result.logs) {
      setSimulationLogs(result.logs);
      setSimulationWinner(result.winner);
      setSimulationCombatants(result.combatants || []);

      if (result.winner === "Phe Ta" && tribulationEventId) {
        await resolveTribulation(tribulationEventId);
      }

      // Show mine rewards as toast if present
      if (Array.isArray(result.logs)) {
        for (const log of result.logs) {
          if (log.action === "MINE_LOOT" || log.action === "MINE_OCCUPY") {
            addToast(log.detail, log.action === "MINE_LOOT" ? "success" : "info");
          }
        }
      }
      return true;
    }
    return false;
  };

  startSimulationRef.current = handleStartSimulation;

  useEffect(() => {
    const encounterId = player.active_combat?.world_monster_encounter_id;
    if (!encounterId || isPracticeMode || isPartyMemberNotLeader) return;

    let cancelled = false;
    const startWhenReady = async () => {
      if (cancelled || autoStartEncounterRef.current === encounterId) return;
      if (actionLoadingRef.current) {
        autoStartTimerRef.current = setTimeout(startWhenReady, 5000);
        return;
      }
      autoStartEncounterRef.current = encounterId;
      addToast("Đạo hữu chuẩn bị quá lâu, trận chiến tự động bắt đầu theo đội hình hiện tại.", "info");
      const started = await startSimulationRef.current();
      if (!started && !cancelled) {
        autoStartEncounterRef.current = null;
        autoStartTimerRef.current = setTimeout(startWhenReady, 10000);
      }
    };

    autoStartTimerRef.current = setTimeout(startWhenReady, 90_000);
    return () => {
      cancelled = true;
      if (autoStartTimerRef.current) clearTimeout(autoStartTimerRef.current);
      autoStartTimerRef.current = null;
    };
  }, [player.active_combat?.world_monster_encounter_id, isPracticeMode, isPartyMemberNotLeader, addToast]);

  const toggleGu = (id: string) => {
    if (selectedGuIds.includes(id)) {
      setSelectedGuIds(selectedGuIds.filter(i => i !== id));
    } else {
      if (selectedGuIds.length >= 15) {
        addToast("Chỉ được mang tối đa 15 Cổ!", "error");
        return;
      }
      setSelectedGuIds([...selectedGuIds, id]);
    }
  };

  // ── Tank/Shield detection ──
  const hasTankOrShield = useMemo(() => {
    const selectedIds = selectedGuIds;
    if (selectedIds.length === 0) return true;

    const tankRoles = new Set(["Defense", "Tank", "Guardian"]);

    for (const id of selectedIds) {
      const gu = eligibleGus.find(g => g.instance_id === id);
      if (!gu) continue;
      const spec = guSpecFor(gu.gu_id, gu.rank);
      if (!spec) continue;

      // Check primary role / combat roles
      if (tankRoles.has(spec.primary_role)) return true;
      if (spec.combat_roles?.some(r => tankRoles.has(r))) return true;

      // Check effects for shield/defense keys
      if (spec.effects?.blocks?.some(b => b.operations?.some(op => op.op === "SHIELD" || (op.op === "STAT_MOD" && op.stat === "DEFENSE")))) {
        return true;
      }
    }
    return false;
  }, [selectedGuIds, eligibleGus, guDictionary]);

  const handleExitSimulation = async () => {
    useGameStore.getState().setActiveCombatReplay(null);
    setSimulationLogs([]);
    setSimulationCombatants([]);
    setSimulationWinner(null);

    if (isPracticeMode) {
      return;
    }
    await fetchPlayerStatus(player.id, true);
    fetchMineSlots();
  };

  // ── Recommendation helpers ──

  const playerBestPaths = useMemo(() => {
    const dm = player.dao_marks || {};
    const pm = player.path_masteries || {};
    const paths = Object.keys({ ...dm, ...pm });
    return paths
      .map(p => ({
        path: p,
        daoMarks: dm[p] || 0,
        mastery: masteryKeyForPoints(pm[p] || 0),
        masteryLevel: masteryLevelForPoints(pm[p] || 0),
      }))
      .sort((a, b) => b.daoMarks - a.daoMarks || b.masteryLevel - a.masteryLevel);
  }, [player.dao_marks, player.path_masteries]);

  const bestPath = useMemo(() => playerBestPaths[0]?.path || "", [playerBestPaths]);

  const pathLabel = (path: string) => {
    if (!path) return "";
    let key = path.endsWith(" Path") ? path : path.charAt(0).toUpperCase() + path.slice(1).toLowerCase() + " Path";
    if (pathLabels[key]) return pathLabels[key];

    const lowerKey = key.toLowerCase();
    for (const [k, v] of Object.entries(pathLabels)) {
      if (k.toLowerCase() === lowerKey) return v;
    }

    const base = path.endsWith(" Path") ? path.slice(0, -5) : path;
    return base.charAt(0).toUpperCase() + base.slice(1).toLowerCase() + " Đạo";
  };

  const multiPathPenalty = useMemo(() => {
    const selectedIds = selectedGuIds;
    if (selectedIds.length <= 6) return 0;
    const paths = new Set(selectedIds.flatMap(id => {
      const gu = eligibleGus.find(g => g.instance_id === id);
      return gu ? guSpecFor(gu.gu_id, gu.rank)?.path || [] : [];
    }).filter(Boolean));
    const uniquePaths = paths.size;
    return Math.min(0.75, (selectedIds.length - 6) * 0.12 + (uniquePaths - 1) * 0.08);
  }, [selectedGuIds, eligibleGus, guDictionary]);

  const recommendLoadout = useCallback(() => {
    const playerRank = player.rank || 1;
    const effectivePath = bestPath || "strength";

    const combatRoles = new Set([
      "ATTACK_CORE",
      "FIGHTER",
      "DEFENSE_CORE",
      "HEAL_CORE",
      "CONTROL_CORE",
      "BUFF_CORE",
      "SUMMON_CORE",
      "MOVEMENT_CORE",
      "TRANSFORMATION_CORE",
      "PASSIVE_BODY",
      "SOUL_HEAL",
      "CLEANSE_WILL",
    ]);

    const nonCombatRoles = new Set([
      "UTILITY_SUPPORT",
      "STORAGE",
      "REFINEMENT_CORE",
      "INVESTIGATION_CORE",
      "SUPPORT_DEDUCTION",
      "MANA_REFINE",
      "ECONOMY_BOOST",
      "APTITUDE_UPGRADE",
      "CONSUMABLE",
    ]);

    const combatTags = new Set([
      "damage",
      "defense",
      "heal",
      "control",
      "stun",
      "freeze",
      "slow",
      "paralyze",
      "charm",
      "bind",
      "burn",
      "poison",
      "bleed",
      "life-steal",
      "lifesteal",
      "summon",
      "movement",
      "flight",
      "attack-speed",
      "add-base-attack",
      "add-base-defense",
      "virtual-shield",
      "orbit-shield",
      "team-shield",
      "shield-flat",
      "transformation",
      "soul-refinement",
    ]);

    const utilityOnlyTags = new Set([
      "utility_support",
      "generate-primeval-stones",
      "refine-mana-tier",
      "fire-refinement",
      "formation-core",
      "storage",
      "investigation",
      "rank-progress",
      "refinement",
      "economy",
      "crafting",
      "utility_boost",
    ]);

    const parseArray = (value: ApiResponseData): string[] => {
      if (!value) return [];

      if (Array.isArray(value)) {
        return value.map(String);
      }

      if (typeof value !== "string") return [];

      try {
        const parsed = JSON.parse(value);
        return Array.isArray(parsed) ? parsed.map(String) : [];
      } catch {
        return value
          .split(",")
          .map((v: string) => v.trim())
          .filter(Boolean);
      }
    };

    const normalize = (value: ApiResponseData) =>
      String(value || "")
        .trim()
        .toLowerCase()
        .replace(/\s+/g, "_")
        .replace(/-/g, "_");

    const isCombatGu = (gu: GuInstance): boolean => {
      const spec = guSpecFor(gu.gu_id, gu.rank);
      if (!spec) return false;

      const primaryRole = normalize(spec.primary_role);
      const roles = parseArray(spec.combat_roles).map(normalize);
      const tags = [
        ...parseArray(spec.combat_tags),
        ...parseArray(spec.support_tags),
      ].map(normalize);

      if (nonCombatRoles.has(primaryRole.toUpperCase())) {
        return false;
      }

      if (
        roles.some(role =>
          [...nonCombatRoles].some(
            nonCombatRole => normalize(nonCombatRole) === role
          )
        )
      ) {
        const hasCombatRole = roles.some(role =>
          [...combatRoles].some(combatRole => normalize(combatRole) === role)
        );

        if (!hasCombatRole) {
          return false;
        }
      }

      const hasCombatRole =
        combatRoles.has(primaryRole.toUpperCase()) ||
        roles.some(role =>
          [...combatRoles].some(
            combatRole => normalize(combatRole) === role
          )
        );

      const hasCombatTag = tags.some(tag =>
        [...combatTags].some(combatTag => normalize(combatTag) === tag)
      );

      const hasOnlyUtilityTags =
        tags.length > 0 &&
        tags.every(tag =>
          [...utilityOnlyTags].some(
            utilityTag => normalize(utilityTag) === tag
          )
        );

      if (hasOnlyUtilityTags && !hasCombatRole) {
        return false;
      }

      return hasCombatRole || hasCombatTag;
    };

    const gus = eligibleGus.filter(isCombatGu);

    if (gus.length === 0) {
      setSelectedGuIds([]);
      return;
    }

    const attackRoles = new Set([
      "ATTACK_CORE",
      "FIGHTER",
    ]);

    const shieldKeys = new Set([
      "defense",
      "add_base_defense",
      "virtual_shield",
      "orbit_shield",
      "team_shield",
      "shield_flat",
    ]);

    const defenseRoles = new Set([
      "DEFENSE_CORE",
      "HEAL_CORE",
      "CONTROL_CORE",
    ]);

    const scored = gus.map(gu => {
      const spec = guSpecFor(gu.gu_id, gu.rank);

      if (!spec) {
        return {
          gu,
          score: 0,
          pathMatch: false,
          isDefense: false,
          isAttack: false,
        };
      }

      const paths = parseArray(spec.path);

      const pathMatch = paths.some(path =>
        normalize(path).includes(normalize(effectivePath))
      );

      let maxDaoMark = 0;
      let maxMastery = 0;

      for (const path of paths) {
        const cleanPath = normalize(path).replace("_path", "");

        const daoValue =
          player.dao_marks?.[cleanPath] ||
          player.dao_marks?.[path] ||
          0;

        if (daoValue > maxDaoMark) {
          maxDaoMark = daoValue;
        }

        const masteryPoints =
          player.path_masteries?.[cleanPath] ||
          player.path_masteries?.[path] ||
          0;

        const masteryLevel = masteryLevelForPoints(masteryPoints);

        if (masteryLevel > maxMastery) {
          maxMastery = masteryLevel;
        }
      }

      const primaryRole = String(spec.primary_role || "").toUpperCase();

      const roles = parseArray(spec.combat_roles).map(role =>
        String(role).toUpperCase()
      );

      const isAttack =
        attackRoles.has(primaryRole) ||
        roles.some(role => attackRoles.has(role));

      const isDefense =
        defenseRoles.has(primaryRole) ||
        roles.some(role => defenseRoles.has(role)) ||
        Boolean(spec.effects?.blocks?.some(b => b.operations?.some(op => op.op === "SHIELD" || (op.op === "STAT_MOD" && op.stat === "DEFENSE")))) ||
        (() => {
          for (const section of [
            "active",
            "passive",
            "support",
          ] as const) {
            const effects = spec.effects?.[section as keyof typeof spec.effects];

            if (!effects) continue;

            for (const key of Object.keys(effects)) {
              if (
                shieldKeys.has(key) &&
                Number(effects[key as keyof typeof effects]) > 0
              ) {
                return true;
              }
            }
          }

          return false;
        })();

      let score = 0;

      if (spec.rank <= playerRank) {
        score += 1000;
      } else {
        score += Math.max(
          0,
          300 - (spec.rank - playerRank) * 100
        );
      }

      score += spec.rank * 50;

      if (pathMatch) {
        score += 800;
      }

      if (maxDaoMark > 0) {
        score += Math.min(500, maxDaoMark / 100);
      }

      score += maxMastery * 150;

      if (isDefense) {
        score += 200;
      }

      if (isAttack) {
        score += 100;
      }

      return {
        gu,
        score,
        pathMatch,
        isDefense,
        isAttack,
      };
    });

    scored.sort((a, b) => b.score - a.score);

    const best = scored[0];

    if (!best) {
      setSelectedGuIds([]);
      return;
    }

    const picked = [best.gu.instance_id];

    if (!best.isDefense) {
      const bestDefense = scored.find(
        item =>
          item.isDefense &&
          item.gu.instance_id !== best.gu.instance_id
      );

      if (bestDefense) {
        picked.push(bestDefense.gu.instance_id);
      }
    }

    const pathPriority =
      parseArray(
        guSpecFor(best.gu.gu_id, best.gu.rank)?.path
      );

    const remaining = scored
      .filter(item => !picked.includes(item.gu.instance_id))
      .sort((a, b) => {
        const aSpec = guSpecFor(
          a.gu.gu_id,
          a.gu.rank
        );

        const bSpec = guSpecFor(
          b.gu.gu_id,
          b.gu.rank
        );

        const aPaths = parseArray(aSpec?.path);
        const bPaths = parseArray(bSpec?.path);

        const aPathMatch = aPaths.some(path =>
          pathPriority.some(
            priority =>
              normalize(priority) === normalize(path)
          )
        )
          ? 1
          : 0;

        const bPathMatch = bPaths.some(path =>
          pathPriority.some(
            priority =>
              normalize(priority) === normalize(path)
          )
        )
          ? 1
          : 0;

        return (
          bPathMatch * 1000 +
          b.score -
          (aPathMatch * 1000 + a.score)
        );
      });

    for (const item of remaining) {
      if (picked.length >= 15) break;

      picked.push(item.gu.instance_id);
    }

    setSelectedGuIds(picked);

  }, [
    eligibleGus,
    guDictionary,
    bestPath,
    player.rank,
    player.dao_marks,
    player.path_masteries,
    player.gu_inventory,
  ]);

  const hasAutoRecommended = useRef(false);
  useEffect(() => {
    if (!settingsLoaded || eligibleGus.length === 0 || hasAutoRecommended.current) {
      return;
    }

    const loadout = combatSettings?.gu_priorities;
    if (loadout && loadout.length > 0) {
      const instanceIds: string[] = [];
      for (const t of loadout) {
        if (!t.gu_id) continue;
        const best = eligibleGus
          .filter(g => g.gu_id === t.gu_id)
          .sort((a, b) => b.rank - a.rank)[0];
        if (best && !instanceIds.includes(best.instance_id)) {
          instanceIds.push(best.instance_id);
        }
      }

      if (instanceIds.length > 0) {
        recommendLoadout();
        setSelectedGuIds(instanceIds.slice(0, 15));
        hasAutoRecommended.current = true;
        return;
      }
    }

    setTimeout(() => {
      recommendLoadout();
    }, 0);
    hasAutoRecommended.current = true;
  }, [settingsLoaded, eligibleGus, combatSettings, recommendLoadout]);

  // Sorting for Gu list
  const sortedEligibleGus = useMemo(() => {
    const gus = [...eligibleGus];
    if (guSortMode === "rank") {
      gus.sort((a, b) => (b.rank || guSpecFor(b.gu_id, b.rank)?.rank || 0) - (a.rank || guSpecFor(a.gu_id, a.rank)?.rank || 0));
    } else {
      const firstSelected = selectedGuIds.length > 0 ? selectedGuIds[0] : null;
      const firstSelectedGu = eligibleGus.find(g => g.instance_id === firstSelected);
      const firstSpec = firstSelectedGu ? guSpecFor(firstSelectedGu.gu_id, firstSelectedGu.rank) : null;
      const mainPath = firstSpec?.path || bestPath;
      gus.sort((a, b) => {
        const aSpec = guSpecFor(a.gu_id, a.rank);
        const bSpec = guSpecFor(b.gu_id, b.rank);
        const aMatch = aSpec?.path === mainPath ? 1 : 0;
        const bMatch = bSpec?.path === mainPath ? 1 : 0;
        return (bMatch - aMatch) || (b.rank || bSpec?.rank || 0) - (a.rank || aSpec?.rank || 0);
      });
    }
    return gus;
  }, [eligibleGus, guSortMode, guDictionary, selectedGuIds, bestPath]);

  const renderCombatantList = (title: string, combatants: Combatant[], icon: React.ReactNode, isEnemy: boolean) => {
    if (!combatants || combatants.length === 0) return null;
    const frontRow = combatants.filter(c => c.grid_x === 1);
    const midRow = combatants.filter(c => c.grid_x === 2);
    const backRow = combatants.filter(c => c.grid_x === 3);
    const borderColor = isEnemy ? "border-rose-800" : "border-emerald-800";
    const headerBg = isEnemy ? "bg-rose-950/20" : "bg-emerald-950/20";

    return (
      <div className={`bg-zinc-900/60 border ${borderColor} rounded-xl p-4 mb-4`}>
        <div className="flex items-center gap-2 mb-3">
          {icon}
          <h3 className="text-sm font-bold text-zinc-300">{title}</h3>
          <span className="text-[10px] text-zinc-600 ml-auto">
            {combatants.length} mục tiêu
          </span>
        </div>

        <div className="grid grid-cols-3 gap-2 mb-3 text-center text-[10px]">
          <div className={`rounded-lg py-1.5 ${headerBg} text-zinc-400 border ${borderColor}/30`}>
            Hàng Tiền: {frontRow.length}
          </div>
          <div className={`rounded-lg py-1.5 ${headerBg} text-zinc-400 border ${borderColor}/30`}>
            Hàng Trung: {midRow.length}
          </div>
          <div className={`rounded-lg py-1.5 ${headerBg} text-zinc-400 border ${borderColor}/30`}>
            Hàng Hậu: {backRow.length}
          </div>
        </div>

        <div className="space-y-1.5">
          {combatants.map(c => (
            <div key={c.id} className="flex items-center gap-2 text-xs bg-zinc-950/40 border border-zinc-800/50 rounded-lg px-3 py-2">
              <div className="w-5 h-5 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[8px] font-bold text-zinc-400">
                {c.avatar_url ? (
                  <img src={c.avatar_url} alt="" className="w-full h-full object-cover" />
                ) : (
                  c.name.slice(0, 2).toUpperCase()
                )}
              </div>
              <span className="text-zinc-200 font-medium truncate flex-1">{c.name}</span>
              <span className="text-zinc-400 shrink-0">{rankLabel(c.rank)}</span>
              <span className="text-zinc-500 shrink-0">{subRankLabel(c.sub_rank, c.rank)}</span>
              <span className={`shrink-0 px-1.5 py-0.5 rounded text-[10px] font-bold ${c.grid_x === 1 ? 'text-rose-400 bg-rose-950/40' : c.grid_x === 2 ? 'text-amber-400 bg-amber-950/40' : 'text-cyan-400 bg-cyan-950/40'}`}>
                {gridXLabel(c.grid_x)}
              </span>
              {(!isEnemy || enemyInfoRevealed.showHP) && (
                <span className="text-zinc-500 shrink-0">
                  <Num value={c.max_hp || 0} /> HP
                </span>
              )}
              {enemyInfoRevealed.showStats && (
                <>
                  {(c.base_attack ?? 0) > 0 && (
                    <span className="text-rose-500 shrink-0 text-[10px] font-mono" title="Tấn công">
                      ATK <Num value={c.base_attack || 0} />
                    </span>
                  )}
                  {(c.base_defense ?? 0) > 0 && (
                    <span className="text-blue-500 shrink-0 text-[10px] font-mono" title="Phòng thủ">
                      DEF <Num value={c.base_defense || 0} />
                    </span>
                  )}
                  {(c.speed ?? 0) > 0 && (
                    <span className="text-cyan-500 shrink-0 text-[10px] font-mono" title="Tốc độ">
                      SPD <Num value={c.speed || 0} />
                    </span>
                  )}
                </>
              )}
              {isEnemy && c.kind && (
                <span className={`px-1.5 py-0.5 rounded text-[10px] border ${typeColor(c.kind)} shrink-0`}>
                  {c.kind}
                </span>
              )}
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderEnemyIntel = () => {
    if (!enemy) return null;
    if (isPracticeMode && practiceDummyConfig && onPracticeDummyConfigChange) {
      const { dummyHP, dummyShield, dummyAtk, dummyDef, dummySpeed } = practiceDummyConfig;
      const updateConfig = (key: string, val: number) => {
        onPracticeDummyConfigChange({ ...practiceDummyConfig, [key]: val });
      };

      const applyPreset = (hp: number, shield: number, atk: number, def: number, speed: number) => {
        onPracticeDummyConfigChange({ dummyHP: hp, dummyShield: shield, dummyAtk: atk, dummyDef: def, dummySpeed: speed });
      };

      return (
        <div className="flex flex-col gap-4">
          <div className="bg-gradient-to-b from-zinc-900/80 to-zinc-950/80 border border-rose-800/50 rounded-2xl overflow-hidden shadow-[0_0_20px_rgba(225,29,72,0.1)]">
            <div className="bg-rose-950/30 border-b border-rose-900/30 px-5 py-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Swords className="w-5 h-5 text-rose-400" />
                <h3 className="text-base font-bold text-zinc-200">Cấu Hình Hình Nhân Gỗ</h3>
              </div>
              <div className="flex gap-2">
                <button onClick={() => applyPreset(100000, 0, 1000, 500, 1000)} className="px-2 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded text-[10px] transition-colors border border-zinc-700">Cơ Bản</button>
                <button onClick={() => applyPreset(10000000, 1000000, 50000, 20000, 5000)} className="px-2 py-1 bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 rounded text-[10px] transition-colors border border-rose-800/50">Cao Thủ</button>
                <button onClick={() => applyPreset(1000000000, 50000000, 2000000, 1000000, 20000)} className="px-2 py-1 bg-purple-950/40 hover:bg-purple-900/60 text-purple-300 rounded text-[10px] transition-colors border border-purple-800/50">Đại Năng</button>
              </div>
            </div>

            <div className="p-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-4">
                <div className="space-y-3">
                  <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/80 rounded-lg p-2.5 transition-all focus-within:border-rose-500/50 focus-within:bg-zinc-900/80 group">
                    <label className="text-xs font-bold text-rose-400/80 group-focus-within:text-rose-400 w-24">Máu (HP)</label>
                    <input
                      type="number"
                      value={dummyHP}
                      onChange={e => updateConfig('dummyHP', parseInt(e.target.value) || 0)}
                      className="w-full bg-transparent text-right text-zinc-200 text-sm font-mono focus:outline-none"
                    />
                  </div>
                  <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/80 rounded-lg p-2.5 transition-all focus-within:border-blue-500/50 focus-within:bg-zinc-900/80 group">
                    <label className="text-xs font-bold text-blue-400/80 group-focus-within:text-blue-400 w-24">Khiên Ảo</label>
                    <input
                      type="number"
                      value={dummyShield}
                      onChange={e => updateConfig('dummyShield', parseInt(e.target.value) || 0)}
                      className="w-full bg-transparent text-right text-zinc-200 text-sm font-mono focus:outline-none"
                    />
                  </div>
                </div>

                <div className="space-y-3">
                  <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/80 rounded-lg p-2.5 transition-all focus-within:border-amber-500/50 focus-within:bg-zinc-900/80 group">
                    <label className="text-xs font-bold text-amber-400/80 group-focus-within:text-amber-400 w-24">Công Kích</label>
                    <input
                      type="number"
                      value={dummyAtk}
                      onChange={e => updateConfig('dummyAtk', parseInt(e.target.value) || 0)}
                      className="w-full bg-transparent text-right text-zinc-200 text-sm font-mono focus:outline-none"
                    />
                  </div>
                  <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/80 rounded-lg p-2.5 transition-all focus-within:border-emerald-500/50 focus-within:bg-zinc-900/80 group">
                    <label className="text-xs font-bold text-emerald-400/80 group-focus-within:text-emerald-400 w-24">Phòng Thủ</label>
                    <input
                      type="number"
                      value={dummyDef}
                      onChange={e => updateConfig('dummyDef', parseInt(e.target.value) || 0)}
                      className="w-full bg-transparent text-right text-zinc-200 text-sm font-mono focus:outline-none"
                    />
                  </div>
                  <div className="flex items-center justify-between bg-zinc-950/50 border border-zinc-800/80 rounded-lg p-2.5 transition-all focus-within:border-cyan-500/50 focus-within:bg-zinc-900/80 group">
                    <label className="text-xs font-bold text-cyan-400/80 group-focus-within:text-cyan-400 w-24">Tốc Độ</label>
                    <input
                      type="number"
                      value={dummySpeed}
                      onChange={e => updateConfig('dummySpeed', parseInt(e.target.value) || 1)}
                      className="w-full bg-transparent text-right text-zinc-200 text-sm font-mono focus:outline-none"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
          {renderCombatantList(
            "Mục Tiêu",
            enemy.enemies || [],
            <Eye className="w-4 h-4 text-rose-400" />,
            true
          )}
        </div>
      );
    }

    const allEnemies = enemy.enemies && enemy.enemies.length > 0 ? enemy.enemies : [];
    return renderCombatantList(
      "Thông Tin Địch",
      allEnemies,
      <Eye className={`w-4 h-4 ${investigationLevel >= 5 ? 'text-emerald-400' : 'text-rose-400'}`} />,
      true
    );
  };

  const renderAllyIntel = () => {
    if (!enemy) return null;
    const allAllies = enemy.allies && enemy.allies.length > 0 ? enemy.allies : [];
    if (allAllies.length === 0) return null;
    return renderCombatantList(
      "Đồng Đội",
      allAllies,
      <Shield className="w-4 h-4 text-emerald-400" />,
      false
    );
  };

  const renderApertureBar = () => {
    if (!enemy || !enemy.aperture_max_hp) return null;
    const hp = enemy.aperture_hp ?? 0;
    const max = enemy.aperture_max_hp ?? 1;
    const pct = Math.max(0, Math.min(100, (hp / max) * 100));
    return (
      <div className="bg-gradient-to-b from-amber-950/40 to-amber-900/20 border border-amber-700/40 rounded-2xl p-4">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-amber-400" />
            <span className="text-sm font-bold text-amber-300">Máu Tiên Khiếu</span>
          </div>
          <span className="text-xs font-mono text-amber-200"><Num value={Math.round(hp)} /> / <Num value={Math.round(max)} /></span>
        </div>
        <div className="h-3 w-full bg-zinc-900 rounded-full overflow-hidden">
          <div className="h-full bg-gradient-to-r from-amber-500 to-orange-500 rounded-full transition-all" style={{ width: `${pct}%` }} />
        </div>
      </div>
    );
  };

  const renderLoadoutPhase = () => (
    <div className="flex flex-col gap-6">
      {/* Aperture (Tiên Khiếu) HP — tribulation combat only */}
      {renderApertureBar()}

      {/* Enemy Intel Panel */}
      {renderEnemyIntel()}

      {/* Ally Intel Panel */}
      {renderAllyIntel()}

      {isPartyMemberNotLeader ? (
        <div className="bg-gradient-to-r from-amber-950/40 via-zinc-900/80 to-amber-950/40 border border-amber-500/40 rounded-2xl p-6 text-center shadow-lg my-2">
          <div className="flex items-center justify-center gap-2 text-amber-400 font-bold text-base mb-2">
            <Users className="w-5 h-5" />
            <span>Trạng Thái Tổ Đội (Thành Viên)</span>
          </div>
          <p className="text-sm text-zinc-200">
            Bạn đang tham gia trận chiến với vai trò thành viên tổ đội. Trưởng nhóm{" "}
            <span className="text-amber-300 font-bold font-serif">
              {party?.members?.find((m) => m.id === party.leader_id)?.username || "Trưởng nhóm"}
            </span>{" "}
            sẽ trực tiếp điều khiển và tiến hành giao chiến.
          </p>
          <p className="text-xs text-zinc-500 mt-2">
            * Giao diện chuẩn bị đội hình và các thao tác giao chiến đã được ẩn đối với thành viên tổ đội.
          </p>
        </div>
      ) : (
        <>
          {/* Action Buttons (Tấn công, Bắt quái, Rút lui) */}
      <div className="flex gap-4 flex-wrap">
        <button
          onClick={handleStartSimulation}
          disabled={actionLoading || selectedGuIds.length === 0}
          className="flex-1 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold py-4 rounded-xl transition-all shadow-[0_0_15px_rgba(16,185,129,0.3)] disabled:opacity-50 flex items-center justify-center gap-2"
        >
          <Play className="w-6 h-6" />
          <span>Tấn Công / Chiến Đấu</span>
        </button>
        {(() => {
          if (isPracticeMode) return null;
          const monsterSpec = monsterDictionary[enemy.enemy_template_id || ""];
          const captureCandidates = selectedGuIds
            .map(id => {
              const gu = eligibleGus.find(g => g.instance_id === id);
              if (!gu) return null;
              const spec = guSpecFor(gu.gu_id, gu.rank);
              return spec && captureBonus(spec) > 0 ? { gu, spec } : null;
            })
            .filter((x): x is { gu: GuInstance; spec: GuSpec } => x !== null);
          const hasCaptureBonus = captureCandidates.length > 0;
          const bestCaptureRank = captureCandidates.reduce((max, c) => Math.max(max, c.spec.rank || 0), 0);
          const monsterRank = enemy.enemy_rank ?? 0;
          const isCapturable = monsterSpec?.is_capturable && hasCaptureBonus;
          if (!isCapturable) return null;
          const rankInsufficient = bestCaptureRank < monsterRank;
          if (rankInsufficient) {
            return (
              <div className="flex flex-col gap-1 w-full sm:w-auto">
                <button
                  disabled
                  className="w-full sm:px-6 bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white font-bold py-4 rounded-xl opacity-40 flex items-center justify-center gap-2 cursor-not-allowed"
                >
                  <PawPrint className="w-6 h-6" />
                  <span>Thu Phục</span>
                </button>
                <div className="flex items-center gap-1.5 text-[11px] text-amber-400">
                  <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                  <span>Cổ bắt rank {bestCaptureRank} không đủ bắt quái rank {monsterRank}. Cần rank {monsterRank} trở lên.</span>
                </div>
              </div>
            );
          }
          return (
            <button
              onClick={async () => {
                const result = await useGameStore.getState().captureMonster();
                if (result?.captured) {
                  handleExitSimulation();
                } else if (result) {
                  handleStartSimulation();
                }
              }}
              disabled={actionLoading}
              className="px-6 bg-gradient-to-r from-purple-600 to-fuchsia-600 hover:from-purple-500 hover:to-fuchsia-500 text-white font-bold py-4 rounded-xl transition-all shadow-[0_0_15px_rgba(168,85,247,0.3)] disabled:opacity-50 flex items-center justify-center gap-2"
            >
              <PawPrint className="w-6 h-6" />
              <span>Thu Phục</span>
            </button>
          );
        })()}
        {onRetreat && (
          <button
            onClick={async () => {
              const res = await onRetreat();
              if (res && res.flee_success === false) {
                const simulatedLogs = (res.logs || []).map((line: string, idx: number) => ({
                  turn: 1,
                  actor_id: "enemy",
                  actor_name: "Địch",
                  action: "BASIC_ATTACK",
                  detail: line
                }));
                const combatants = [
                  ...(res.enemy?.allies || []),
                  ...(res.enemy?.enemies || [])
                ];
                if (res.defeat || res.victory) {
                  // The counterattack settled the fight: show an end replay
                  // (defeat => Phe Địch, victory => Phe Ta).
                  const winner = res.defeat ? "Phe Địch" : "Phe Ta";
                  useGameStore.getState().setActiveCombatReplay({
                    logs: simulatedLogs,
                    winner,
                    combatants,
                    enemy: res.enemy
                  });
                  setSimulationLogs(simulatedLogs);
                  setSimulationWinner(winner);
                  setSimulationCombatants(combatants);
                } else {
                  // Survived the counterattack — continue the battle
                  // immediately using the already-loaded (settings or
                  // recommended) loadout.
                  setSimulationLogs([]);
                  setSimulationWinner(null);
                  setSimulationCombatants([]);
                  useGameStore.getState().setActiveCombatReplay(null);
                  await handleStartSimulation();
                }
              }
            }}
            disabled={actionLoading}
            className="px-6 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 font-bold py-4 rounded-xl border border-zinc-800 transition-all flex items-center gap-2 justify-center"
          >
            <span>Rút Lui</span>
          </button>
        )}
      </div>

      {/* Recommendation Panel */}
      {playerBestPaths.length > 0 && showRecommendation && (
        <div className="bg-gradient-to-r from-emerald-950/30 to-cyan-950/30 border border-emerald-800/30 rounded-xl p-4 mb-2">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-bold text-emerald-300 flex items-center gap-2">
              <Wand2 className="w-4 h-4 text-emerald-400" />
              Gợi ý đội hình
            </h3>
            <button onClick={() => setShowRecommendation(false)} className="text-[10px] text-zinc-600 hover:text-zinc-400">Ẩn</button>
          </div>
          <div className="flex flex-wrap gap-4 text-xs">
            <div className="flex-1 min-w-[200px]">
              <span className="text-zinc-500">Đường lĩnh ngộ mạnh nhất:</span>
              <div className="flex flex-wrap gap-1.5 mt-1.5">
                {playerBestPaths.slice(0, 5).map(p => (
                  <span key={p.path} className={`px-2 py-0.5 rounded-full border text-[10px] font-bold ${p.path === bestPath ? 'bg-emerald-900/40 text-emerald-300 border-emerald-700' : 'bg-zinc-800 text-zinc-400 border-zinc-700'
                    }`}>
                    {pathLabel(p.path)} · <Num value={p.daoMarks} /> Đạo
                  </span>
                ))}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={recommendLoadout}
                className="px-3 py-1.5 bg-emerald-900/50 hover:bg-emerald-800/50 text-emerald-300 rounded-lg text-xs font-bold border border-emerald-700 transition-all flex items-center gap-1.5"
              >
                <Zap className="w-3.5 h-3.5" />
                Tự động chọn
              </button>
            </div>
          </div>
          {playerBestPaths[0] && (
            <div className="mt-2 text-[10px] text-zinc-500">
              Gợi ý: ưu tiên Cổ cùng hệ {pathLabel(bestPath)} để tối ưu Đạo Ấn và Đường Lĩnh Ngộ.
            </div>
          )}
        </div>
      )}

      <div className="bg-black/40 border border-zinc-800 p-6 rounded-2xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-cyan-400 flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-emerald-400" />
            Chuẩn bị Đội Hình
          </h2>
          <div className="flex items-center gap-2">
            <div className="flex bg-zinc-900 rounded-lg border border-zinc-800 p-0.5">
              <button
                onClick={() => setGuSortMode("rank")}
                className={`px-2 py-1 rounded text-[10px] font-bold transition-all ${guSortMode === "rank" ? 'bg-zinc-700 text-zinc-200' : 'text-zinc-500 hover:text-zinc-300'}`}
              >
                <ChevronDown className="w-3 h-3 inline" /> Bậc
              </button>
              <button
                onClick={() => setGuSortMode("path")}
                className={`px-2 py-1 rounded text-[10px] font-bold transition-all ${guSortMode === "path" ? 'bg-zinc-700 text-zinc-200' : 'text-zinc-500 hover:text-zinc-300'}`}
              >
                <ChevronUp className="w-3 h-3 inline" /> Hệ
              </button>
            </div>
            <button
              onClick={recommendLoadout}
              className="px-2 py-1 bg-emerald-900/50 hover:bg-emerald-800/50 text-emerald-300 rounded text-[10px] font-bold border border-emerald-800 transition-all flex items-center gap-1"
            >
              <Wand2 className="w-3 h-3" />
              Gợi ý
            </button>
          </div>
        </div>

        {/* Warnings */}
        <div className="space-y-2 mb-4">
          {multiPathPenalty > 0 && (
            <div className="flex items-center gap-2 px-3 py-2 bg-amber-950/30 border border-amber-800/30 rounded-lg text-[11px] text-amber-300">
              <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
              Cảnh báo: {multiPathPenalty >= 0.5 ? 'Quá nhiều' : 'Nhiều'} hệ Cổ làm giảm tỉ lệ thành công ({Math.round(multiPathPenalty * 100)}%).
            </div>
          )}
          {!hasTankOrShield && selectedGuIds.length > 0 && (
            <div className="flex items-center gap-2 px-3 py-2 bg-rose-950/30 border border-rose-800/30 rounded-lg text-[11px] text-rose-300">
              <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
              Không có Cổ chống chịu / khiên ảo! Nên chọn ít nhất 1 Cổ có kỹ năng tạo Khiên Ảo hoặc phòng thủ.
            </div>
          )}
        </div>

        {/* Backlash Summary */}
        {(() => {
          const backlashCount = selectedGuIds.filter(id => {
            const gu = eligibleGus.find(g => g.instance_id === id);
            const spec = gu ? guSpecFor(gu.gu_id, gu.rank) : null;
            return !!(spec && gu && (gu.rank || spec.rank || 0) > (player.rank || 1));
          }).length;
          return (
            <div className="flex flex-wrap items-center gap-3 mb-4 px-3 py-2 bg-zinc-900/60 border border-zinc-800 rounded-lg text-xs">
              {backlashCount > 0 && (
                <span className="text-rose-400 flex items-center gap-1">
                  <AlertTriangle className="w-3 h-3" />
                  {backlashCount} Cổ bậc cao → có thể mất {backlashCount * 5} Sinh lực/lần sài
                </span>
              )}
              <span className="text-zinc-600 ml-auto">{selectedGuIds.length}/7 đã chọn</span>
            </div>
          );
        })()}



        {/* Single Gu Selection Grid */}
        <div className="mb-6">
          {eligibleGus.length === 0 ? (
            <p className="text-sm text-zinc-650 italic py-4 text-center">Không có Cổ nào đủ điều kiện (cần ACTIVE + equipped)</p>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
              {sortedEligibleGus.map(gu => {
                const spec = guSpecFor(gu.gu_id, gu.rank);
                const isSelected = selectedGuIds.includes(gu.instance_id);
                const playerRank = player.rank || 1;
                const guRank = gu.rank || spec?.rank || 0;
                const isHigherRank = guRank > playerRank;
                const pathMatch = spec?.path && spec.path.some(p => p.toLowerCase().includes(bestPath.toLowerCase()));
                return (
                  <div
                    key={gu.instance_id}
                    onClick={() => toggleGu(gu.instance_id)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all relative ${isSelected ? 'border-emerald-500 bg-emerald-950/30 ring-1 ring-emerald-500/30' :
                        isHigherRank ? 'border-rose-800/40 bg-rose-950/10' :
                          pathMatch ? 'border-emerald-800/40 bg-emerald-950/10' :
                            'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'
                      }`}
                  >
                    <div className="font-bold text-sm text-zinc-200 truncate">{spec?.name || gu.gu_id}</div>
                    <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                      <span className={`text-[10px] ${isHigherRank ? 'text-rose-400 font-bold' : 'text-zinc-500'}`}>
                        Bậc {guRank}
                      </span>
                      {spec?.path && spec.path.map(p => (
                        <span key={p} className={`text-[9px] px-1.5 py-0.5 rounded-full ${p.toLowerCase().includes(bestPath.toLowerCase()) ? 'bg-emerald-900/40 text-emerald-300' : 'bg-zinc-800 text-zinc-500'
                          }`}>
                          {pathLabel(p)}
                        </span>
                      ))}
                    </div>
                    {spec?.primary_role && (
                      <div className="mt-1.5 text-[9px] text-zinc-500">
                        <span className="text-zinc-600">{spec.primary_role}</span>
                      </div>
                    )}
                    {(() => {
                      const effects = getCombatEffects(spec);
                      if (effects.length === 0) return null;
                      return (
                        <div className="flex flex-wrap gap-1 mt-1.5">
                          {effects.map(e => (
                            <span key={e.key} className={`text-[8px] px-1.5 py-0.5 rounded-full border ${e.className}`}>
                              {e.label}
                            </span>
                          ))}
                        </div>
                      );
                    })()}
                    {isHigherRank && (
                      <div className="mt-1 text-[8px] text-rose-500/80 flex items-center gap-0.5">
                        <AlertTriangle className="w-2.5 h-2.5" /> Phản Phệ -5 HP
                      </div>
                    )}
                    {isSelected && (
                      <div className="absolute top-1 right-1 w-4 h-4 bg-emerald-500 rounded-full flex items-center justify-center">
                        <span className="text-[8px] text-white font-bold">✓</span>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Auto Combat Settings */}
        <div className="mb-6 bg-zinc-900/40 border border-zinc-800 rounded-xl p-4">
          <h3 className="text-sm font-bold text-zinc-300 mb-3 flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-amber-400" />
            Tự Động Chiến Đấu
          </h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" className="sr-only peer" checked={autoHealHP} onChange={e => setAutoHealHP(e.target.checked)} />
                  <div className="w-9 h-5 bg-zinc-700 rounded-full peer peer-checked:bg-emerald-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
                </label>
                <span className="text-xs text-zinc-300">Tự động dùng dược liệu hồi HP</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-[10px] text-zinc-500">HP &le;</span>
                <input type="range" min="10" max="80" value={healThreshold} onChange={e => setHealThreshold(Number(e.target.value))} disabled={!autoHealHP} className="w-20 h-1 accent-emerald-500" />
                <span className="text-xs text-zinc-300 w-8">{healThreshold}%</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" className="sr-only peer" checked={autoRestoreMana} onChange={e => setAutoRestoreMana(e.target.checked)} />
                  <div className="w-9 h-5 bg-zinc-700 rounded-full peer peer-checked:bg-cyan-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
                </label>
                <span className="text-xs text-zinc-300">Tự động dùng Nguyên Thạch hồi Mana</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-[10px] text-zinc-500">Mana &le;</span>
                <input type="range" min="10" max="80" value={manaThreshold} onChange={e => setManaThreshold(Number(e.target.value))} disabled={!autoRestoreMana} className="w-20 h-1 accent-cyan-500" />
                <span className="text-xs text-zinc-300 w-8">{manaThreshold}%</span>
              </div>
            </div>
          </div>
          <div className="mt-2 text-[10px] text-zinc-600">
            {player.materials_inventory && Object.keys(player.materials_inventory).length > 0
              ? `Dược liệu trong túi: ${Object.entries(player.materials_inventory).filter(([k]) => k !== "primeval_stone").length} loại`
              : "Không có dược liệu trong túi"}
            · Nguyên Thạch: <Num value={player.primeval_stones || 0} />
            · Tiên Nguyên Thạch: <Num value={player.immortal_stones || 0} />
          </div>
        </div>
      </div>
    </>
  )}
</div>
  );

  const renderReplayPhase = () => {
    const displayCombatants = (() => {
      const list: Record<string, unknown>[] = [];
      const pObj = player as unknown as Record<string, unknown>;

      // 1. Ally (Player)
      list.push({
        id: player.id,
        name: player.username,
        kind: "ally",
        max_hp: player.base_hp || player.hp || 1000,
        hp: player.hp ?? 1000,
        max_mana: player.max_mana || 100,
        mana: player.current_mana ?? 100,
        rank: player.rank || 1,
        grid_x: 0,
        grid_y: 0,
        avatar_url: pObj.avatar_url,
      });

      // 2. If simulationCombatants provided
      if (simulationCombatants && simulationCombatants.length > 0) {
        simulationCombatants.forEach((c) => {
          const isPlayer = c.name === player.username || c.id === player.id;
          if (isPlayer) {
            list[0] = { ...list[0], ...c, kind: "ally", grid_x: 0, grid_y: 0 };
          } else {
            list.push({
              ...c,
              kind: "enemy",
              grid_x: (c.grid_x as number | undefined) ?? 4,
              grid_y: (c.grid_y as number | undefined) ?? (list.length - 1),
            });
          }
        });
        return list;
      }

      // 3. Fallback from enemy active combat state if simulationCombatants empty
      if (enemy) {
        const enemyList = (enemy.enemies && enemy.enemies.length > 0)
          ? enemy.enemies
          : [{
              id: "enemy_1",
              name: enemy.enemy_name || "Kẻ Địch",
              max_hp: enemy.enemy_max_hp || 1000,
              hp: enemy.enemy_hp || 1000,
              rank: 1,
            }];

        enemyList.forEach((e: Record<string, unknown>, idx: number) => {
          list.push({
            id: String(e.id || `enemy_${idx}`),
            name: String(e.name || enemy.enemy_name || "Kẻ Địch"),
            kind: "enemy",
            max_hp: e.max_hp || 1000,
            hp: e.hp ?? e.max_hp ?? 1000,
            max_mana: e.max_mana || 100,
            mana: e.current_mana ?? 100,
            rank: e.rank || 1,
            grid_x: 4,
            grid_y: idx,
            kind_detail: e.kind || enemy.enemy_type || "",
          });
        });
      }

      return list;
    })();

    return (
      <CombatReportView
        logs={simulationLogs as unknown as Parameters<typeof CombatReportView>[0]["logs"]}
        combatants={displayCombatants as unknown as Parameters<typeof CombatReportView>[0]["combatants"]}
        winner={simulationWinner || "Phe Địch"}
        onBack={handleExitSimulation}
        playerName={player.username}
        teamAName={player.username || "Phe Ta"}
        teamBName={enemy?.enemy_name || "Kẻ Địch"}
        title="Chiến Báo Trận Đấu Mô Phỏng"
        apertureHp={enemy?.aperture_hp ?? 0}
        apertureMaxHp={enemy?.aperture_max_hp ?? 0}
      />
    );
  };

  return (
    <GamePanel variant="default" className="w-full mx-auto">
      {simulationLogs.length === 0
        ? (player?.active_combat || isPracticeMode ? renderLoadoutPhase() : null)
        : renderReplayPhase()}
    </GamePanel>
  );
};
