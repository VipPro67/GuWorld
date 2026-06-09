"use client";

import React, { useState, useEffect, useRef } from "react";
import { Player, CombatState, GuSpec, GuInstance, useGameStore } from "@/store/useGameStore";
import { Flame, Shield, Heart, Zap, Sparkles, Award, Play } from "lucide-react";

interface CombatScreenProps {
  player: Player;
  enemy: CombatState;
  guDictionary: Record<string, GuSpec>;
  actionLoading: boolean;
  onAttack: (equippedGuIds: string[], isCharge?: boolean, targetId?: string, targetX?: number, targetY?: number) => Promise<any>;
  onRetreat?: () => Promise<void>;
}

const getPathColor = (path: string): string => {
  switch (path) {
    case "Moon Path":
      return "text-indigo-400 bg-indigo-950/30 border-indigo-900/40";
    case "Strength Path":
      return "text-orange-400 bg-orange-950/30 border-orange-900/40";
    case "Earth Path":
      return "text-amber-400 bg-amber-950/30 border-amber-900/40";
    case "Refinement Path":
      return "text-emerald-400 bg-emerald-950/30 border-emerald-900/40";
    case "Rule Path":
      return "text-sky-400 bg-sky-950/30 border-sky-900/40";
    default:
      return "text-zinc-400 bg-zinc-950/30 border-zinc-900/40";
  }
};

const FormatKey = (key: string): string => {
  switch (key) {
    case "damage":
    case "add_base_attack":
      return "Dmg";
    case "defense":
    case "add_base_defense":
      return "Def";
    case "refine_mana_tier":
      return "Refine";
    case "cleansing_power":
      return "Clean";
    default:
      return key.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase());
  }
};

const Grid3x3 = ({
  team,
  isAllyTeam,
  selectedId,
  onSelect,
  guDictionary
}: {
  team: any[];
  isAllyTeam: boolean;
  selectedId: string;
  onSelect?: (combatant: any) => void;
  guDictionary: Record<string, GuSpec>;
}) => {
  const cols = isAllyTeam ? [3, 2, 1] : [1, 2, 3];
  const rows = [1, 2, 3];

  return (
    <div className="grid grid-rows-3 gap-2 p-2 bg-black/45 border border-zinc-900 rounded-2xl w-full">
      {rows.map((y) => (
        <div key={y} className="grid grid-cols-3 gap-2">
          {cols.map((x) => {
            const c = team.find((member) => member.grid_x === x && member.grid_y === y);
            const isDead = c && c.hp <= 0;
            const isSelected = c && selectedId === c.id;

            return (
              <div
                key={x}
                onClick={() => c && !isDead && onSelect && onSelect(c)}
                className={`relative flex flex-col justify-between p-2 rounded-xl border aspect-square text-left transition-all ${
                  !c
                    ? "border-zinc-950/40 bg-zinc-950/10 opacity-20"
                    : isDead
                    ? "border-zinc-950 bg-black/60 opacity-40 cursor-not-allowed"
                    : isSelected
                    ? isAllyTeam
                      ? "border-emerald-500 bg-emerald-950/20 shadow-[0_0_8px_rgba(16,185,129,0.35)] cursor-pointer scale-[1.02]"
                      : "border-rose-500 bg-rose-950/20 shadow-[0_0_8px_rgba(239,68,68,0.35)] cursor-pointer scale-[1.02]"
                    : "border-zinc-900 hover:border-zinc-800 bg-zinc-900/40 cursor-pointer"
                }`}
              >
                <span className="absolute top-1 right-1 text-[7px] text-zinc-650 font-mono">
                  {x},{y}
                </span>

                {c ? (
                  <>
                    <div className="truncate">
                      <div className={`text-[10px] font-bold truncate leading-none ${isDead ? "line-through text-zinc-600" : "text-zinc-200"}`}>
                        {c.name}
                      </div>
                      <div className="text-[7.5px] text-zinc-500 leading-none mt-1">
                        SPD: {c.speed?.toFixed(1) || "10.0"}
                      </div>
                    </div>

                    {!isDead && (
                      <div className="space-y-1 mt-1">
                        <div className="w-full bg-zinc-950 h-1.5 rounded-full overflow-hidden border border-zinc-900 relative">
                          <div
                            className="bg-rose-500 h-full rounded-full transition-all duration-300"
                            style={{ width: `${(c.hp / c.max_hp) * 100}%` }}
                          />
                        </div>
                        {c.virtual_shield > 0 && (
                          <div className="w-full bg-zinc-955 h-1 rounded-full overflow-hidden border border-zinc-900 relative">
                            <div
                              className="bg-cyan-400 h-full rounded-full transition-all duration-300 shadow-[0_0_4px_rgba(34,211,238,0.6)]"
                              style={{ width: `${Math.min(100, (c.virtual_shield / 1000) * 100)}%` }}
                            />
                          </div>
                        )}
                        <div className="flex justify-between text-[7px] font-mono text-zinc-400 leading-none">
                          <span>HP: {Math.round(c.hp)}</span>
                          {c.virtual_shield > 0 && <span className="text-cyan-400">S: {Math.round(c.virtual_shield)}</span>}
                        </div>
                      </div>
                    )}

                    {isDead && (
                      <span className="text-[8px] bg-zinc-900 text-zinc-600 px-1 rounded uppercase tracking-wider font-bold text-center self-center w-full mt-2">
                        TỬ TRẬN
                      </span>
                    )}
                  </>
                ) : (
                  <div className="flex items-center justify-center h-full text-[9px] text-zinc-800 italic">
                    -
                  </div>
                )}
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
};

export const CombatScreen: React.FC<CombatScreenProps> = ({
  player,
  enemy,
  guDictionary,
  actionLoading,
  onAttack,
  onRetreat,
}) => {
  const [selectedGuInstanceIds, setSelectedGuInstanceIds] = useState<string[]>([]);
  const [selectedTargetID, setSelectedTargetID] = useState<string>("");
  const [selectedTargetX, setSelectedTargetX] = useState<number>(1);
  const [selectedTargetY, setSelectedTargetY] = useState<number>(2);
  const [shouldFlashRed, setShouldFlashRed] = useState(false);
  const [shouldShake, setShouldShake] = useState(false);
  const logEndRef = useRef<HTMLDivElement | null>(null);

  const isSelf = (ally: any) => {
    if (!ally) return false;
    return (
      ally.id?.toLowerCase() === player.id?.toLowerCase() ||
      ally.name === player.username
    );
  };

  const selfCombatant = enemy.allies?.find(isSelf);
  const currentHP = selfCombatant !== undefined ? selfCombatant.hp : player.hp;
  const maxHP = selfCombatant !== undefined ? selfCombatant.max_hp : (player.base_hp || 100);
  const currentMana = selfCombatant !== undefined ? selfCombatant.current_mana : player.current_mana;
  const maxMana = selfCombatant !== undefined ? selfCombatant.max_mana : player.max_mana;
  const alliesFiltered = enemy.allies ? enemy.allies.filter((ally) => !isSelf(ally)) : [];

  const alliesUnified = enemy.allies && enemy.allies.length > 0 ? enemy.allies : [
    {
      id: player.id,
      name: player.username,
      hp: player.hp,
      max_hp: player.base_hp || 100,
      current_mana: player.current_mana,
      max_mana: player.max_mana,
      speed: 10.0 + player.rank * 2.0 + player.sub_rank * 0.5,
      grid_x: 2,
      grid_y: 2,
      virtual_shield: 0
    }
  ];

  const enemiesList = enemy.enemies && enemy.enemies.length > 0 ? enemy.enemies : [
    {
      id: enemy.enemy_template_id || "enemy",
      name: enemy.enemy_name,
      hp: enemy.enemy_hp,
      max_hp: enemy.enemy_max_hp,
      speed: 8.0 + (enemy.enemy_rank || 1) * 2.0 + (enemy.enemy_sub_rank || 1) * 0.5,
      grid_x: 1,
      grid_y: 2,
      virtual_shield: 0
    }
  ];

  const combatants = [
    ...alliesUnified,
    ...enemiesList
  ].filter(c => c.hp > 0)
  .sort((a, b) => (b.speed || 0) - (a.speed || 0));

  const handleSelectEnemy = (c: any) => {
    setSelectedTargetID(c.id);
    setSelectedTargetX(c.grid_x || 1);
    setSelectedTargetY(c.grid_y || 2);
  };

  // Initialize target
  useEffect(() => {
    if (enemy.enemies && enemy.enemies.length > 0) {
      const activeEnemy = enemy.enemies.find((e) => e.hp > 0);
      if (activeEnemy) {
        setSelectedTargetID(activeEnemy.id);
        setSelectedTargetX(activeEnemy.grid_x || 1);
        setSelectedTargetY(activeEnemy.grid_y || 2);
      } else {
        setSelectedTargetID(enemy.enemies[0].id);
        setSelectedTargetX(enemy.enemies[0].grid_x || 1);
        setSelectedTargetY(enemy.enemies[0].grid_y || 2);
      }
    } else {
      setSelectedTargetID(enemy.enemy_template_id || "enemy");
      setSelectedTargetX(1);
      setSelectedTargetY(2);
    }
  }, [enemy.enemy_name, enemy.enemies, enemy.enemy_template_id]);

  const logs = useGameStore((state) => state.logs);

  // Auto-scroll combat logs to bottom on new log additions
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  // Flash & shake screen when player HP drops (e.g. backlash or counter-attack)
  const prevHpRef = useRef(player.hp);
  useEffect(() => {
    if (player.hp < prevHpRef.current) {
      setShouldFlashRed(true);
      setShouldShake(true);

      const timer = setTimeout(() => {
        setShouldFlashRed(false);
        setShouldShake(false);
      }, 400);

      return () => clearTimeout(timer);
    }
    prevHpRef.current = player.hp;
  }, [player.hp]);

  const isPassiveGu = (spec?: GuSpec): boolean => {
    if (!spec) return false;
    const t = (spec.gu_type || "").toLowerCase();
    return t.includes("passive") || t.includes("nội tại") || t.includes("the chat") || t.includes("thể chất");
  };

  // Only allow ACTIVE + EQUIPPED + non-passive Gu in map combat
  const eligibleGus = (player.gu_inventory || []).filter((gu) => {
    if (gu.status !== "ACTIVE" || !gu.equipped) return false;
    const spec = guDictionary[gu.gu_id];
    return !isPassiveGu(spec);
  });

  const toggleGuSelection = (instanceId: string) => {
    if (actionLoading) return;
    setSelectedGuInstanceIds((prev) =>
      prev.includes(instanceId)
        ? prev.filter((id) => id !== instanceId)
        : [...prev, instanceId]
    );
  };

  // Compute estimated stats & mana costs
  const calculateEstimatedCosts = () => {
    let totalCost = 0;
    let expectedDamage = 0;
    let expectedDefense = 0;
    let backlashRisk = false;

    selectedGuInstanceIds.forEach((instId) => {
      const guInst = player.gu_inventory.find((g) => g.instance_id === instId);
      if (!guInst) return;

      const spec = guDictionary[guInst.gu_id];
      if (!spec) return;

      // Check level penalty risk
      if (player.rank < spec.rank) {
        backlashRisk = true;
      }

      // Base mana cost logic: BaseManaCost percentage or fallback to 10% player max mana
      let baseManaPct = spec.base_mana_cost > 0.0 ? spec.base_mana_cost / 100.0 : 0.10;
      let cost = baseManaPct * player.max_mana;
      if (player.rank > spec.rank) {
        cost /= Math.pow(10, player.rank - spec.rank);
      }
      let divisor = Math.pow(2, player.sub_rank - 1);
      cost /= divisor;

      totalCost += cost;

      // Power calculation with regional suppression
      let powerScale = 1.0;
      if (spec.origin_region !== player.current_region) {
        const effectiveRank = Math.max(1, spec.rank - 1);
        powerScale = effectiveRank / spec.rank;
      }

      const activeEffects = spec.effects?.active || {};
      const baseDmg = activeEffects["damage"] || activeEffects["add_base_attack"] || 0;
      const baseDef = activeEffects["defense"] || activeEffects["add_base_defense"] || 0;

      // Extract player's mastery for spec.path
      let masteryMultiplier = 1.0;
      if (player.path_masteries && player.path_masteries[spec.path]) {
        const mLevel = player.path_masteries[spec.path];
        switch (mLevel) {
          case "Ordinary":
            masteryMultiplier = 1.0;
            break;
          case "Master":
            masteryMultiplier = 1.1;
            break;
          case "Grandmaster":
            masteryMultiplier = 1.25;
            break;
          case "Supreme Grandmaster":
            masteryMultiplier = 1.5;
            break;
        }
      }

      expectedDamage += baseDmg * powerScale * masteryMultiplier;
      expectedDefense += baseDef * powerScale * masteryMultiplier;
    });

    // Apply strength dao marks to damage estimate
    const strengthMarks = player.dao_marks?.["strength"] || 0;
    expectedDamage = expectedDamage * (1 + strengthMarks / 100);

    return { totalCost, expectedDamage, expectedDefense, backlashRisk };
  };

  const { totalCost, expectedDamage, expectedDefense, backlashRisk } = calculateEstimatedCosts();

  const hasEnoughMana = player.current_mana >= totalCost;
  const canAttack = !actionLoading && hasEnoughMana;

  const handleExecuteKillerMove = async () => {
    if (!canAttack) return;
    await onAttack(selectedGuInstanceIds, false, selectedTargetID, selectedTargetX, selectedTargetY);
    setSelectedGuInstanceIds([]);
  };

  const handleExecuteCharge = async () => {
    if (actionLoading) return;
    await onAttack([], true, selectedTargetID, selectedTargetX, selectedTargetY);
    setSelectedGuInstanceIds([]);
  };

  return (
    <div
      className={`glass-panel rounded-3xl p-5 w-full max-w-4xl mx-auto space-y-6 relative overflow-hidden transition-all duration-300 border ${
        shouldShake ? "animate-shake" : ""
      } ${
        shouldFlashRed
          ? "border-rose-500/80 shadow-[0_0_20px_rgba(239,68,68,0.3)] bg-rose-950/5"
          : "border-zinc-800"
      }`}
    >
      {/* Turn Queue Order Header */}
      <div className="flex items-center gap-2 overflow-x-auto pb-2 border-b border-zinc-900 scrollbar-none">
        <span className="text-[10px] font-bold uppercase tracking-wider text-zinc-500 shrink-0 font-mono">
          Thứ Tự Tốc Độ (Turn Order):
        </span>
        {combatants.map((c, idx) => {
          const isAlly = enemy.allies?.some(a => a.id === c.id) || c.id === player.id;
          return (
            <div
              key={c.id}
              className={`px-2.5 py-1 rounded-xl text-[10px] font-bold border flex items-center gap-1.5 shrink-0 transition-all ${
                isAlly
                  ? "bg-emerald-950/20 border-emerald-500/20 text-emerald-450"
                  : "bg-rose-950/20 border-rose-500/20 text-rose-450"
              }`}
            >
              <span className="font-mono bg-black/40 px-1 py-0.1 rounded text-[8px] text-zinc-500">
                {idx + 1}
              </span>
              <span className="truncate max-w-[80px]">{c.name}</span>
              <span className="text-[8px] text-zinc-500 font-mono">({c.speed?.toFixed(1) || "10.0"})</span>
            </div>
          );
        })}
      </div>

      {/* 3x3 Arena Battlefield Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch bg-zinc-950/40 border border-zinc-900/60 p-4 sm:p-5 rounded-2xl">
        {/* Allies Grid (Left) */}
        <div className="flex flex-col space-y-3">
          <div className="flex justify-between items-center px-1">
            <h4 className="text-xs sm:text-sm font-bold text-emerald-400 font-serif flex items-center gap-1.5">
              <Sparkles className="w-3.5 h-3.5 text-emerald-500" />
              Chiến Trận Đồng Minh (Allies Grid)
            </h4>
            <span className="text-[9px] text-zinc-500 font-mono uppercase tracking-wider">
              Hậu (X=3) ⟷ Trung (X=2) ⟷ Tiền (X=1)
            </span>
          </div>

          <Grid3x3
            team={alliesUnified}
            isAllyTeam={true}
            selectedId={selectedTargetID}
            guDictionary={guDictionary}
          />
        </div>

        {/* Enemies Grid (Right) */}
        <div className="flex flex-col space-y-3 lg:border-l lg:border-zinc-900 lg:pl-6">
          <div className="flex justify-between items-center px-1">
            <h4 className="text-xs sm:text-sm font-bold text-rose-450 font-serif flex items-center gap-1.5">
              <Flame className="w-3.5 h-3.5 text-rose-500 animate-pulse" />
              Chiến Trận Đối Thủ (Enemies Grid)
            </h4>
            <span className="text-[9px] text-zinc-500 font-mono uppercase tracking-wider">
              Tiền (X=1) ⟷ Trung (X=2) ⟷ Hậu (X=3)
            </span>
          </div>

          <Grid3x3
            team={enemiesList}
            isAllyTeam={false}
            selectedId={selectedTargetID}
            onSelect={handleSelectEnemy}
            guDictionary={guDictionary}
          />
        </div>
      </div>

      {/* 2. MIDDLE VIEW: Retro Console Log monitor */}
      <div className="bg-black/85 border border-zinc-900 rounded-2xl p-4 h-48 overflow-y-auto custom-scrollbar font-mono text-xs flex flex-col gap-1.5 shadow-[inset_0_0_20px_rgba(0,0,0,0.9)]">
        {logs.slice(0, 30).reverse().map((logLine, idx) => {
          let colorClass = "text-zinc-400";
          if (logLine.includes("sát thương lên")) colorClass = "text-amber-300 font-semibold";
          else if (logLine.includes("phản công")) colorClass = "text-rose-400";
          else if (logLine.includes("Chiến thắng")) colorClass = "text-emerald-400 font-bold text-sm tracking-wide";
          else if (logLine.includes("bại trận")) colorClass = "text-rose-500 font-bold text-sm tracking-wide";
          else if (logLine.includes("Backlash")) colorClass = "text-rose-500 font-bold animate-pulse";
          else if (logLine.includes("SÁT CHIÊU từ")) colorClass = "text-purple-300 font-bold border-b border-zinc-950 pb-1 block";

          return (
            <div key={idx} className={`${colorClass} leading-relaxed text-left`}>
              &gt; {logLine}
            </div>
          );
        })}
        <div ref={logEndRef} />
      </div>

      {/* 3. BOTTOM VIEW: Selector action bar */}
      <div className="space-y-4">
        <div className="flex justify-between items-center border-b border-zinc-800 pb-2">
          <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono">
            Kết Hợp Sát Chiêu (Equipped Gu Bugs & Killer Moves)
          </span>
          {backlashRisk && (
            <span className="text-[9px] font-bold text-rose-500 border border-rose-500/20 bg-rose-950/20 px-2 py-0.5 rounded animate-pulse uppercase">
              Cảnh báo: Phản phệ chấn thương!
            </span>
          )}
        </div>

        {eligibleGus.length === 0 ? (
          <div className="text-center py-6 border border-dashed border-zinc-800 rounded-2xl text-xs text-zinc-600 italic">
            Không có Cổ trùng hợp lệ để dùng trong chiến đấu bản đồ (chỉ nhận Cổ đang Xuất Trận và không phải Cổ nội tại).
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {eligibleGus.map((gu) => {
              const spec = guDictionary[gu.gu_id] || {
                id: gu.gu_id,
                name: gu.gu_id,
                rank: 1,
                path: "Unknown Path",
                gu_type: "Phàm Cổ",
                origin_region: "",
                base_mana_cost: 0,
                effects: {},
                hunger_rate: 0,
                food_type: "unknown",
                max_hunger: 100,
                description: "Cổ trùng chưa được giải nghĩa.",
              };

              const isSelected = selectedGuInstanceIds.includes(gu.instance_id);
              const isNorthernSuppressed = spec.origin_region !== player.current_region;

              return (
                <div
                  key={gu.instance_id}
                  onClick={() => toggleGuSelection(gu.instance_id)}
                  className={`p-3 rounded-xl border transition-all cursor-pointer text-left relative flex flex-col justify-between min-h-[7.5rem] h-auto ${
                    isSelected
                      ? "bg-amber-500/10 border-amber-500 shadow-md shadow-amber-950/15"
                      : "bg-zinc-950/40 border-zinc-900 hover:border-zinc-800 hover:bg-zinc-950/60"
                  }`}
                >
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold text-zinc-200 font-serif leading-none truncate max-w-[70%]">
                        {spec.name}
                      </span>
                      <span className="text-[7px] font-mono border border-amber-500/20 px-1 rounded text-amber-500 font-bold uppercase shrink-0">
                        R{spec.rank}
                      </span>
                    </div>

                    <div className="flex flex-wrap gap-1 items-center pt-1">
                      <span className={`text-[7px] px-1 rounded font-mono border leading-normal ${getPathColor(spec.path)}`}>
                        {spec.path}
                      </span>
                      {spec.effects?.active && Object.entries(spec.effects.active).map(([key, val]) => {
                        const isDmg = key === "damage" || key === "add_base_attack";
                        const isDef = key === "defense" || key === "add_base_defense";
                        const label = isDmg ? "Dmg" : isDef ? "Def" : FormatKey(key);
                        const color = isDmg 
                          ? "bg-rose-950/20 border-rose-500/20 text-rose-400" 
                          : isDef 
                          ? "bg-blue-950/20 border-blue-500/20 text-blue-400" 
                          : "bg-zinc-900 border-zinc-850 text-zinc-400";
                        return (
                          <span key={key} className={`text-[7px] px-1 rounded font-mono border leading-normal ${color}`}>
                            {label}: {val}
                          </span>
                        );
                      })}
                    </div>
                  </div>

                  <div className="flex justify-between items-center mt-2 pt-2 border-t border-zinc-900">
                    <span className="text-[8px] text-zinc-500 font-mono leading-none">
                      No: {gu.hunger}/100
                    </span>
                    {isNorthernSuppressed && (
                      <span
                        className="text-[6px] text-rose-450 bg-rose-950/20 border border-rose-800/40 px-1 py-0.2 rounded font-bold uppercase font-mono tracking-wide"
                        title="Regional Suppression: Effective Rank reduced by 1!"
                      >
                        Áp Chế
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Action Button & Live estimate bar */}
        <div className="flex flex-col sm:flex-row gap-4 items-stretch sm:items-center justify-between pt-4 border-t border-zinc-900/60">
          <div className="flex gap-4 text-[10px] text-zinc-500 font-mono bg-zinc-950/40 border border-zinc-900 px-4 py-2.5 rounded-xl flex-1">
            <div>Chân nguyên tiêu hao: <span className={`font-bold ${hasEnoughMana ? "text-emerald-400" : "text-rose-400 animate-pulse font-extrabold"}`}>{totalCost.toFixed(1)} / {player.current_mana.toFixed(1)}</span></div>
            <div>Sát thương ước tính: <span className="text-amber-400 font-bold">{expectedDamage.toFixed(1)}</span></div>
            {expectedDefense > 0 && <div>Lá chắn ước tính: <span className="text-blue-400 font-bold">{expectedDefense.toFixed(1)}</span></div>}
          </div>

          <div className="flex gap-2 shrink-0">
            {onRetreat && (
              <button
                type="button"
                onClick={onRetreat}
                disabled={actionLoading}
                className="py-3 px-5 rounded-xl font-bold font-serif text-xs tracking-wider border border-rose-900/50 hover:bg-rose-950/30 text-rose-450 active:scale-[0.98] transition-all cursor-pointer"
              >
                BỎ CHẠY (FLEE)
              </button>
            )}
            <button
              onClick={handleExecuteCharge}
              disabled={actionLoading}
              className={`py-3 px-5 rounded-xl font-bold font-serif text-xs tracking-wider border transition-all cursor-pointer active:scale-[0.98] ${
                enemy.charge_points > 0 
                  ? "bg-amber-600/20 border-amber-500 text-amber-400 hover:bg-amber-600/30" 
                  : "border-amber-900/50 hover:bg-amber-950/30 text-amber-500"
              }`}
            >
              {enemy.charge_points > 0 ? "BỘC PHÁ SÁT CHIÊU (RELEASE)" : "TÍCH TỤ CÔNG LỰC (CHARGE)"}
            </button>
            <button
              onClick={handleExecuteKillerMove}
              disabled={!canAttack || enemy.charge_points > 0}
              className={`py-3 px-6 rounded-xl font-bold font-serif text-xs tracking-wider flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                canAttack && enemy.charge_points === 0
                  ? "bg-gradient-to-r from-rose-600 to-amber-600 hover:from-rose-500 hover:to-amber-500 text-white shadow-md shadow-rose-950/30 active:scale-[0.98]"
                  : "bg-zinc-900 border border-zinc-850 text-zinc-500 cursor-not-allowed"
              }`}
            >
              <Play className="w-3.5 h-3.5" />
              <span>KÍCH HOẠT SÁT CHIÊU</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
