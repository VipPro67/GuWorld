"use client";

import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { Player, CombatState, Combatant, GuSpec, GuInstance, useGameStore } from "@/store/useGameStore";
import { Sparkles, Play, Swords, Eye, Shield, Zap, ChevronDown, ChevronUp, Wand2, AlertTriangle } from "lucide-react";
import { CombatReplayView } from "./CombatReplayView";

interface CombatScreenProps {
  player: Player;
  enemy: CombatState;
  guDictionary: Record<string, GuSpec>;
  actionLoading: boolean;
  onAttack?: () => Promise<unknown>;
  onRetreat?: () => Promise<void>;
}

const rankLabel = (r: number) => {
  const names = ["", "Nhất chuyển", "Nhị chuyển", "Tam chuyển", "Tứ chuyển", "Ngũ chuyển", "Lục chuyển", "Thất chuyển", "Bát chuyển", "Cửu chuyển"];
  return names[r] || `Cửu chuyển+`;
};

const subRankLabel = (s: number) => ["", "Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"][s] || "";

const typeColor = (t?: string) => {
  switch (t) {
    case "Boss": return "text-rose-400 border-rose-800 bg-rose-950/30";
    case "Elite": return "text-purple-400 border-purple-800 bg-purple-950/30";
    case "Normal":
    case "MONSTER": return "text-zinc-400 border-zinc-700 bg-zinc-900/50";
    default: return "text-zinc-400 border-zinc-700 bg-zinc-900/50";
  }
};

const gridXLabel = (x?: number) => {
  switch (x) {
    case 1: return "Tiền";
    case 2: return "Trung";
    case 3: return "Hậu";
    default: return "?";
  }
};

export const CombatScreen: React.FC<CombatScreenProps> = ({
  player,
  enemy,
  guDictionary,
  actionLoading,
  onRetreat,
}) => {
  const [coreGuId, setCoreGuId] = useState<string>("");
  const [supportGuIds, setSupportGuIds] = useState<string[]>([]);
  const [selectedKillerMoveIds, setSelectedKillerMoveIds] = useState<string[]>([]);

  const [simulationLogs, setSimulationLogs] = useState<Record<string, unknown>[]>([]);
  const [simulationWinner, setSimulationWinner] = useState<string | null>(null);
  const [simulationCombatants, setSimulationCombatants] = useState<Record<string, unknown>[]>([]);

  const [autoHealHP, setAutoHealHP] = useState(false);
  const [healThreshold, setHealThreshold] = useState(30);
  const [autoRestoreMana, setAutoRestoreMana] = useState(false);
  const [manaThreshold, setManaThreshold] = useState(30);
  const [guSortMode, setGuSortMode] = useState<"rank" | "path">("rank");
  const [showRecommendation, setShowRecommendation] = useState(true);

  const executeAutoCombat = useGameStore((state) => state.executeAutoCombat);
  const fetchPlayerStatus = useGameStore((state) => state.fetchPlayerStatus);
  const fetchMineSlots = useGameStore((state) => state.fetchMineSlots);
  const killerMoves = useGameStore((state) => state.killerMoves);
  const fetchKillerMoves = useGameStore((state) => state.fetchKillerMoves);
  const combatSettings = useGameStore((state) => state.combatSettings);
  const fetchCombatSettings = useGameStore((state) => state.fetchCombatSettings);
  const addToast = useGameStore((state) => state.addToast);

  useEffect(() => {
    void fetchKillerMoves();
  }, [fetchKillerMoves]);

  // Pre-populate auto settings from persistent combat config
  const combatSettingsFetched = useRef(false);
  useEffect(() => {
    if (!combatSettings && !combatSettingsFetched.current) {
      combatSettingsFetched.current = true;
      fetchCombatSettings();
    } else if (combatSettings) {
      setAutoHealHP(combatSettings.auto_heal_hp);
      setHealThreshold(combatSettings.heal_threshold);
      setAutoRestoreMana(combatSettings.auto_restore_mana);
      setManaThreshold(combatSettings.mana_threshold);
    }
  }, [combatSettings]);

  // Filter: only show Gu with combat-relevant effects
  const isCombatGu = (gu: GuInstance): boolean => {
    const spec = guDictionary[gu.gu_id];
    if (!spec) return false;
    const combatRoles = new Set([
      "ATTACK_CORE", "DEFENSE_CORE", "HEAL_CORE", "CONTROL_CORE",
      "BUFF_CORE", "SUMMON_CORE", "MOVEMENT_CORE", "TRANSFORMATION_CORE",
      "PASSIVE_BODY", "FIGHTER",
    ]);
    if (combatRoles.has(spec.primary_role)) return true;
    const combatEffectKeys = new Set([
      "damage", "add_base_attack", "add_base_defense", "defense",
      "damage_multiplier", "damage_multiplier_on_hit",
      "blood_drain", "bleed", "life_steal", "lifesteal",
      "shield", "virtual_shield", "damage_reduction_pct",
      "heal", "regen_hp",
      "stun", "freeze", "slow", "paralyze", "charm", "bind",
      "burn", "poison",
      "summon_phantom", "summon_flying_bear_phantom", "summon_clones",
      "movement", "flight", "attack_speed",
      "mana_regen", "refine_mana_tier",
      "wood_transformation", "curve_attack", "range", "aoe_col", "aoe_all", "aoe_row",
      "damage_reflect", "thorns_damage", "phantom_damage",
    ]);
    for (const section of ["active", "passive", "support"] as const) {
      const effects = spec.effects?.[section];
      if (!effects) continue;
      for (const key of Object.keys(effects)) {
        if (combatEffectKeys.has(key)) return true;
      }
    }
    return false;
  };

  const eligibleGus = (player.gu_inventory || []).filter((gu) => gu.status === "ACTIVE" && gu.equipped && isCombatGu(gu));

  const computeInvestigation = (): number => {
    let inv = 1.0;
    (player.gu_inventory || []).forEach((gu) => {
      if (gu.status !== "ACTIVE" || !gu.equipped) return;
      const spec = guDictionary[gu.gu_id];
      if (spec?.effects?.passive?.investigation) {
        inv += spec.effects.passive.investigation;
      }
    });
    return inv;
  };

  const investigationLevel = useMemo(() => computeInvestigation(), [player.gu_inventory, guDictionary]);

  const enemyInfoRevealed = useMemo(() => {
    const inv = investigationLevel;
    const hpRange = enemy.enemy_max_hp > 10000 ? "Cao" : enemy.enemy_max_hp > 1000 ? "Trung bình" : "Thấp";
    return {
      showHP: inv >= 2,
      showStats: inv >= 4,
      showLootHints: inv >= 8,
    };
  }, [investigationLevel, enemy]);

  const handleStartSimulation = async () => {
    if (!coreGuId) {
      addToast("Vui lòng chọn ít nhất 1 Cổ Hạch Tâm!", "error");
      return;
    }

    const loadout = {
      CoreGuId: eligibleGus.find(g => g.instance_id === coreGuId)?.gu_id || coreGuId,
      SupportGuIds: supportGuIds.map(id => eligibleGus.find(g => g.instance_id === id)?.gu_id || id),
      KillerMoveIds: selectedKillerMoveIds,
      ImmortalGuId: null,
      AutoHealHP: autoHealHP,
      HealThreshold: healThreshold,
      AutoRestoreMana: autoRestoreMana,
      ManaThreshold: manaThreshold,
    };

    const result = await executeAutoCombat(loadout);
    if (result && result.logs) {
      setSimulationLogs(result.logs);
      setSimulationWinner(result.winner);
      setSimulationCombatants(result.combatants || []);
    }
  };

  const toggleSupportGu = (id: string) => {
    if (supportGuIds.includes(id)) {
      setSupportGuIds(supportGuIds.filter(i => i !== id));
    } else {
      if (supportGuIds.length >= 6) {
        addToast("Chỉ được mang tối đa 6 Cổ Phụ Trợ!", "error");
        return;
      }
      setSupportGuIds([...supportGuIds, id]);
    }
  };

  const toggleKillerMove = (id: string) => {
    if (selectedKillerMoveIds.includes(id)) {
      setSelectedKillerMoveIds(selectedKillerMoveIds.filter(i => i !== id));
    } else {
      setSelectedKillerMoveIds([...selectedKillerMoveIds, id]);
    }
  };

  // ── Tank/Shield detection ──
  const hasTankOrShield = useMemo(() => {
    const selectedIds = [coreGuId, ...supportGuIds].filter(Boolean);
    if (selectedIds.length === 0) return true;

    const shieldKeys = new Set(["defense", "add_base_defense", "virtual_shield", "orbit_shield", "team_shield"]);
    const tankRoles = new Set(["Defense", "Tank", "Guardian"]);

    for (const id of selectedIds) {
      const gu = eligibleGus.find(g => g.instance_id === id);
      if (!gu) continue;
      const spec = guDictionary[gu.gu_id];
      if (!spec) continue;

      // Check primary role / combat roles
      if (tankRoles.has(spec.primary_role)) return true;
      if (spec.combat_roles?.some(r => tankRoles.has(r))) return true;

      // Check effects for shield/defense keys
      for (const section of ["active", "passive", "support"] as const) {
        const effects = spec.effects?.[section];
        if (!effects) continue;
        for (const key of Object.keys(effects)) {
          if (shieldKeys.has(key) && (effects[key] || 0) > 0) return true;
        }
      }
    }
    return false;
  }, [coreGuId, supportGuIds, eligibleGus, guDictionary]);

  const handleExitSimulation = async () => {
    await fetchPlayerStatus(player.id);
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
        mastery: pm[p] || "Ordinary",
        masteryLevel: ["Ordinary", "Master", "Grandmaster", "Great Grandmaster", "Supreme Grandmaster"].indexOf(pm[p] || "Ordinary"),
      }))
      .sort((a, b) => b.daoMarks - a.daoMarks || b.masteryLevel - a.masteryLevel);
  }, [player.dao_marks, player.path_masteries]);

  const bestPath = useMemo(() => playerBestPaths[0]?.path || "", [playerBestPaths]);

  const pathLabel = (path: string) => {
    const labels: Record<string, string> = {
      water: "Thủy", fire: "Hỏa", earth: "Thổ", wind: "Phong",
      qi: "Khí", moon: "Nguyệt", strength: "Lực", poison: "Độc",
      blood: "Huyết", shadow: "Ảnh", wood: "Mộc", metal: "Kim",
      lightning: "Lôi", time: "Thời Gian", space: "Không Gian",
    };
    return labels[path] || path;
  };

  const multiPathPenalty = useMemo(() => {
    const selectedIds = [coreGuId, ...supportGuIds].filter(Boolean);
    if (selectedIds.length <= 1) return 0;
    const paths = new Set(selectedIds.flatMap(id => {
      const gu = eligibleGus.find(g => g.instance_id === id);
      return gu ? guDictionary[gu.gu_id]?.path || [] : [];
    }).filter(Boolean));
    const uniquePaths = paths.size;
    return Math.min(0.75, (selectedIds.length - 1) * 0.12 + (uniquePaths - 1) * 0.08);
  }, [coreGuId, supportGuIds, eligibleGus, guDictionary]);

  const recommendLoadout = useCallback(() => {
    const gus = eligibleGus;
    if (gus.length === 0) return;

    const effectivePath = bestPath || "strength";

    // Score each Gu for recommendation
    const scored = gus.map(gu => {
      const spec = guDictionary[gu.gu_id];
      if (!spec) return { gu, score: 0, pathMatch: false };
      const pathMatch = spec.path && spec.path.some(p => p.toLowerCase().includes(effectivePath.toLowerCase()));
      let maxDaoMark = 0;
      let maxMastery = 0;
      if (spec.path) {
        spec.path.forEach(p => {
          const cleanP = p.trim().toLowerCase().replace(" path", "");
          const daoVal = player.dao_marks?.[cleanP] || player.dao_marks?.[p] || 0;
          if (daoVal > maxDaoMark) maxDaoMark = daoVal;

          const levelStr = player.path_masteries?.[cleanP] || player.path_masteries?.[p] || "Ordinary";
          const levelIdx = ["Ordinary", "Master", "Grandmaster", "Great Grandmaster", "Supreme Grandmaster"].indexOf(levelStr);
          if (levelIdx > maxMastery) maxMastery = levelIdx;
        });
      }
      const score = spec.rank * 100 + (pathMatch ? 500 : 0) + (maxDaoMark > 0 ? spec.rank * 50 : 0) + maxMastery * 100;
      return { gu, score, pathMatch };
    });

    scored.sort((a, b) => b.score - a.score);

    // Pick best Core
    const bestCore = scored[0];
    if (!bestCore) return;
    setCoreGuId(bestCore.gu.instance_id);

    // Pick supports sharing the same path as core to minimize multi-path penalty
    const corePathList = guDictionary[bestCore.gu.gu_id]?.path || [];
    const supportCandidates = scored
      .filter(s => s.gu.instance_id !== bestCore.gu.instance_id)
      .sort((a, b) => {
        const aSpec = guDictionary[a.gu.gu_id];
        const bSpec = guDictionary[b.gu.gu_id];
        const aPathMatch = aSpec?.path && corePathList.some(p => aSpec.path.includes(p)) ? 1 : 0;
        const bPathMatch = bSpec?.path && corePathList.some(p => bSpec.path.includes(p)) ? 1 : 0;
        return (bPathMatch * 1000 + b.score) - (aPathMatch * 1000 + a.score);
      });

    setSupportGuIds(supportCandidates.slice(0, 6).map(s => s.gu.instance_id));

    // Pick killer moves matching the core path (only valid ones)
    const validKMs = (() => {
      const activeGuIds = new Set(
        (player.gu_inventory || [])
          .filter(gu => gu.status === "ACTIVE")
          .map(gu => gu.instance_id)
      );
      return killerMoves.filter(move => {
        if (!activeGuIds.has(move.core_player_gu_id)) return false;
        if (move.slots) {
          for (const slot of move.slots) {
            if (!activeGuIds.has(slot.player_gu_id)) return false;
          }
        }
        return true;
      });
    })();
    const matchingKMs = validKMs.filter(km => {
      if (corePathList.length === 0) return true;
      return corePathList.includes(km.path);
    });
    setSelectedKillerMoveIds(matchingKMs.slice(0, 3).map(km => km.id));
  }, [eligibleGus, guDictionary, bestPath, player.dao_marks, player.path_masteries, killerMoves]);

  // Sorting for Gu list
  // Only show killer moves whose component Gus are all ACTIVE
  const validKillerMoves = useMemo(() => {
    const activeGuIds = new Set(
      (player.gu_inventory || [])
        .filter(gu => gu.status === "ACTIVE")
        .map(gu => gu.instance_id)
    );
    return killerMoves.filter(move => {
      if (!activeGuIds.has(move.core_player_gu_id)) return false;
      if (move.slots) {
        for (const slot of move.slots) {
          if (!activeGuIds.has(slot.player_gu_id)) return false;
        }
      }
      return true;
    });
  }, [killerMoves, player.gu_inventory]);

  const sortedEligibleGus = useMemo(() => {
    const gus = [...eligibleGus];
    if (guSortMode === "rank") {
      gus.sort((a, b) => (guDictionary[b.gu_id]?.rank || 0) - (guDictionary[a.gu_id]?.rank || 0));
    } else {
      const coreSpec = coreGuId ? guDictionary[eligibleGus.find(g => g.instance_id === coreGuId)?.gu_id || ""] : null;
      const corePath = coreSpec?.path || bestPath;
      gus.sort((a, b) => {
        const aSpec = guDictionary[a.gu_id];
        const bSpec = guDictionary[b.gu_id];
        const aMatch = aSpec?.path === corePath ? 1 : 0;
        const bMatch = bSpec?.path === corePath ? 1 : 0;
        return (bMatch - aMatch) || (bSpec?.rank || 0) - (aSpec?.rank || 0);
      });
    }
    return gus;
  }, [eligibleGus, guSortMode, guDictionary, coreGuId, bestPath]);

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
              <span className="text-zinc-200 font-medium truncate flex-1">{c.name}</span>
              <span className="text-zinc-400 shrink-0">{rankLabel(c.rank)}</span>
              <span className="text-zinc-500 shrink-0">{subRankLabel(c.sub_rank)}</span>
              <span className={`shrink-0 px-1.5 py-0.5 rounded text-[10px] font-bold ${c.grid_x === 1 ? 'text-rose-400 bg-rose-950/40' : c.grid_x === 2 ? 'text-amber-400 bg-amber-950/40' : 'text-cyan-400 bg-cyan-950/40'}`}>
                {gridXLabel(c.grid_x)}
              </span>
              {enemyInfoRevealed.showHP && (
                <span className="text-zinc-500 shrink-0">
                  {c.max_hp?.toLocaleString() || "?"} HP
                </span>
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
    const allEnemies = enemy.enemies && enemy.enemies.length > 0 ? enemy.enemies : [];
    return renderCombatantList(
      "Thông Tin Địch",
      allEnemies,
      <Eye className={`w-4 h-4 ${investigationLevel >= 5 ? 'text-emerald-400' : 'text-rose-400'}`} />,
      true
    );
  };

  const renderAllyIntel = () => {
    const allAllies = enemy.allies && enemy.allies.length > 0 ? enemy.allies : [];
    if (allAllies.length === 0) return null;
    return renderCombatantList(
      "Đồng Đội",
      allAllies,
      <Shield className="w-4 h-4 text-emerald-400" />,
      false
    );
  };

  const renderLoadoutPhase = () => (
    <div className="flex flex-col gap-6">
      {/* Enemy Intel Panel */}
      {renderEnemyIntel()}

      {/* Ally Intel Panel */}
      {renderAllyIntel()}

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
                  <span key={p.path} className={`px-2 py-0.5 rounded-full border text-[10px] font-bold ${
                    p.path === bestPath ? 'bg-emerald-900/40 text-emerald-300 border-emerald-700' : 'bg-zinc-800 text-zinc-400 border-zinc-700'
                  }`}>
                    {pathLabel(p.path)} · {p.daoMarks.toLocaleString()} Đạo
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
                <ChevronDown className="w-3 h-3 inline" /> Rank
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
          {!hasTankOrShield && (coreGuId || supportGuIds.length > 0) && (
            <div className="flex items-center gap-2 px-3 py-2 bg-rose-950/30 border border-rose-800/30 rounded-lg text-[11px] text-rose-300">
              <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
              Không có Cổ chống chịu / khiên ảo! Nên chọn ít nhất 1 Cổ có kỹ năng tạo Khiên Ảo hoặc phòng thủ.
            </div>
          )}
        </div>

        {/* Core Gu */}
        <div className="mb-6">
          <div className="flex items-center justify-between border-b border-zinc-800 pb-2 mb-3">
            <h3 className="text-sm font-bold text-zinc-300">Cổ Hạch Tâm <span className="text-zinc-600 font-normal">(bắt buộc 1)</span></h3>
            <span className="text-[10px] text-zinc-600">{eligibleGus.length} Cổ khả dụng</span>
          </div>
          {eligibleGus.length === 0 ? (
            <p className="text-sm text-zinc-600 italic py-4 text-center">Không có Cổ nào đủ điều kiện (cần ACTIVE + equipped)</p>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              {sortedEligibleGus.map(gu => {
                const spec = guDictionary[gu.gu_id];
                const isSelected = coreGuId === gu.instance_id;
                const pathMatch = spec?.path && spec.path.some(p => p.toLowerCase().includes(bestPath.toLowerCase()));
                return (
                  <div
                    key={gu.instance_id}
                    onClick={() => setCoreGuId(gu.instance_id)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all relative ${
                      isSelected ? 'border-emerald-500 bg-emerald-950/30 ring-1 ring-emerald-500/30' :
                      pathMatch && !coreGuId ? 'border-emerald-800/40 bg-emerald-950/10' :
                      'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'
                    }`}
                  >
                    <div className="font-bold text-sm text-zinc-200 truncate">{spec?.name || gu.gu_id}</div>
                    <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                      <span className="text-[10px] text-zinc-500">R{spec?.rank}</span>
                      {spec?.path && spec.path.map(p => (
                        <span key={p} className={`text-[9px] px-1.5 py-0.5 rounded-full ${
                          p.toLowerCase().includes(bestPath.toLowerCase()) ? 'bg-emerald-900/40 text-emerald-300' : 'bg-zinc-800 text-zinc-500'
                        }`}>
                          {pathLabel(p)}
                        </span>
                      ))}
                      {spec?.primary_role && (
                        <span className="text-[9px] text-zinc-600">{spec.primary_role}</span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Support Gu */}
        <div className="mb-6">
          <div className="flex items-center justify-between border-b border-zinc-800 pb-2 mb-3">
            <h3 className="text-sm font-bold text-zinc-300">Cổ Phụ Trợ <span className="text-zinc-600 font-normal">(tối đa 6)</span></h3>
            <span className="text-[10px] text-zinc-600">{supportGuIds.length}/6 đã chọn</span>
          </div>
          {eligibleGus.filter(gu => gu.instance_id !== coreGuId).length === 0 ? (
            <p className="text-sm text-zinc-600 italic py-4 text-center">Không còn Cổ nào để chọn</p>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              {sortedEligibleGus.filter(gu => gu.instance_id !== coreGuId).map(gu => {
                const spec = guDictionary[gu.gu_id];
                const isSelected = supportGuIds.includes(gu.instance_id);
                const coreSpec = coreGuId ? guDictionary[eligibleGus.find(g => g.instance_id === coreGuId)?.gu_id || ""] : null;
                const pathMatch = coreSpec?.path && spec?.path && spec.path.some(p => coreSpec.path.includes(p));
                return (
                  <div
                    key={gu.instance_id}
                    onClick={() => toggleSupportGu(gu.instance_id)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all ${
                      isSelected ? 'border-cyan-500 bg-cyan-950/30 ring-1 ring-cyan-500/30' :
                      pathMatch ? 'border-zinc-700 bg-zinc-900/60' :
                      'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'
                    }`}
                  >
                    <div className="font-bold text-sm text-zinc-200 truncate">{spec?.name || gu.gu_id}</div>
                    <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                      <span className="text-[10px] text-zinc-500">R{spec?.rank}</span>
                      {spec?.path && spec.path.map(p => (
                        <span key={p} className={`text-[9px] px-1.5 py-0.5 rounded-full ${
                          coreSpec?.path?.includes(p) ? 'bg-emerald-900/40 text-emerald-300' : 'bg-zinc-800 text-zinc-500'
                        }`}>
                          {pathLabel(p)}
                        </span>
                      ))}
                    </div>
                    {isSelected && (
                      <div className="absolute top-1 right-1 w-4 h-4 bg-cyan-500 rounded-full flex items-center justify-center">
                        <span className="text-[8px] text-white font-bold">✓</span>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Killer Moves */}
        <div className="mb-6">
          <div className="flex items-center justify-between border-b border-zinc-800 pb-2 mb-3">
            <h3 className="text-sm font-bold text-zinc-300 flex items-center gap-2">
              <Swords className="w-4 h-4 text-purple-400" />
              Sát Chiêu
            </h3>
            <span className="text-[10px] text-zinc-600">{selectedKillerMoveIds.length} đã chọn</span>
          </div>
          {validKillerMoves.length === 0 ? (
            <p className="text-sm text-zinc-500 italic">Chưa có sát chiêu hợp lệ (cần đầy đủ Cổ trùng thành phần).</p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {validKillerMoves.map(move => {
                const isSelected = selectedKillerMoveIds.includes(move.id);
                const coreSpec = coreGuId ? guDictionary[eligibleGus.find(g => g.instance_id === coreGuId)?.gu_id || ""] : null;
                const pathMatch = coreSpec?.path ? coreSpec.path.includes(move.path) : move.path === bestPath;
                return (
                  <div
                    key={move.id}
                    onClick={() => toggleKillerMove(move.id)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all ${
                      isSelected ? 'border-purple-500 bg-purple-950/30 ring-1 ring-purple-500/30' :
                      pathMatch ? 'border-zinc-700 bg-zinc-900/60' :
                      'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="font-bold text-sm text-zinc-200">{move.name}</div>
                      <span className={`text-[10px] px-2 py-0.5 rounded-full ${(move.stability ?? 0) >= 80 ? 'bg-emerald-900/50 text-emerald-300' : (move.stability ?? 0) >= 50 ? 'bg-yellow-900/50 text-yellow-300' : 'bg-red-900/50 text-red-300'}`}>
                        {(move.stability ?? 0).toFixed(0)}%
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span className={`text-[9px] px-1.5 py-0.5 rounded-full ${
                        pathMatch ? 'bg-purple-900/40 text-purple-300' : 'bg-zinc-800 text-zinc-500'
                      }`}>
                        {move.path}
                      </span>
                      <span className="text-[10px] text-zinc-500">DMG {move.damage_value.toFixed(0)}</span>
                      <span className="text-[10px] text-zinc-500">Mana {move.total_mana_cost.toFixed(0)}</span>
                    </div>
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
            · Nguyên Thạch: {player.primeval_stones?.toLocaleString() || 0}
            · Tiên Nguyên Thạch: {player.immortal_stones?.toLocaleString() || 0}
          </div>
        </div>

        <div className="flex gap-4">
          <button
            onClick={handleStartSimulation}
            disabled={actionLoading || !coreGuId}
            className="flex-1 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold py-4 rounded-xl transition-all shadow-[0_0_15px_rgba(16,185,129,0.3)] disabled:opacity-50 flex items-center justify-center gap-2"
          >
            <Play className="w-5 h-5" />
            BẮT ĐẦU
          </button>
          {onRetreat && (
            <button
              onClick={onRetreat}
              disabled={actionLoading}
              className="px-6 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 font-bold py-4 rounded-xl border border-zinc-800 transition-all"
            >
              Rút Lui
            </button>
          )}
        </div>
      </div>
    </div>
  );

  const renderReplayPhase = () => (
    <CombatReplayView
      logs={simulationLogs as any}
      combatants={simulationCombatants as any}
      winner={simulationWinner || "Phe Địch"}
      onBack={handleExitSimulation}
      playerName={player.username}
    />
  );

  return (
    <div className="w-full max-w-4xl mx-auto">
      {simulationLogs.length === 0 ? renderLoadoutPhase() : renderReplayPhase()}
    </div>
  );
};
