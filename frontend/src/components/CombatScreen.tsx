"use client";

import React, { useState, useEffect, useRef, useMemo } from "react";
import { Player, CombatState, Combatant, GuSpec, GuInstance, KillerMove, useGameStore } from "@/store/useGameStore";
import { Flame, Sparkles, Play, FastForward, ShieldAlert, Swords, Eye, Skull, Shield, Heart } from "lucide-react";

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

  const [simulationLogs, setSimulationLogs] = useState<any[]>([]);
  const [currentLogIndex, setCurrentLogIndex] = useState(-1);
  const [isReplaying, setIsReplaying] = useState(false);
  const [simulationWinner, setSimulationWinner] = useState<string | null>(null);

  const [autoHealHP, setAutoHealHP] = useState(false);
  const [healThreshold, setHealThreshold] = useState(30);
  const [autoRestoreMana, setAutoRestoreMana] = useState(false);
  const [manaThreshold, setManaThreshold] = useState(30);

  const executeAutoCombat = useGameStore((state) => state.executeAutoCombat);
  const fetchPlayerStatus = useGameStore((state) => state.fetchPlayerStatus);
  const killerMoves = useGameStore((state) => state.killerMoves);
  const fetchKillerMoves = useGameStore((state) => state.fetchKillerMoves);
  const combatSettings = useGameStore((state) => state.combatSettings);
  const fetchCombatSettings = useGameStore((state) => state.fetchCombatSettings);
  const logEndRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    void fetchKillerMoves();
  }, [fetchKillerMoves]);

  // Pre-populate auto settings from persistent combat config
  useEffect(() => {
    if (!combatSettings) {
      fetchCombatSettings();
    } else {
      setAutoHealHP(combatSettings.auto_heal_hp);
      setHealThreshold(combatSettings.heal_threshold);
      setAutoRestoreMana(combatSettings.auto_restore_mana);
      setManaThreshold(combatSettings.mana_threshold);
    }
  }, [combatSettings]);

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [currentLogIndex]);

  useEffect(() => {
    if (isReplaying && currentLogIndex < simulationLogs.length - 1) {
      const timer = setTimeout(() => {
        setCurrentLogIndex((prev) => prev + 1);
      }, 500);
      return () => clearTimeout(timer);
    } else if (isReplaying && currentLogIndex >= simulationLogs.length - 1) {
      setIsReplaying(false);
    }
  }, [isReplaying, currentLogIndex, simulationLogs]);

  const eligibleGus = (player.gu_inventory || []).filter((gu) => gu.status === "ACTIVE" && gu.equipped);

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
      alert("Vui lòng chọn ít nhất 1 Cổ Hạch Tâm!");
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
      setCurrentLogIndex(result.logs.length - 1);
      setIsReplaying(false);
    }
  };

  const toggleSupportGu = (id: string) => {
    if (supportGuIds.includes(id)) {
      setSupportGuIds(supportGuIds.filter(i => i !== id));
    } else {
      if (supportGuIds.length >= 6) {
        alert("Chỉ được mang tối đa 6 Cổ Phụ Trợ!");
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

  const handleExitSimulation = async () => {
    await fetchPlayerStatus(player.id);
  };

  const parseResultSummary = () => {
    let totalDmgDealt = 0;
    let totalDmgTaken = 0;
    let shieldsUsed = 0;
    let finalPlayerHP = player.hp;
    let lootItems: string[] = [];
    let hpChanged = false;

    for (const log of simulationLogs) {
      if (log.damage > 0) {
        if (log.target_id) {
          const isDmgToPlayer = log.target_id === player.id || log.actor_name === player.username;
          if (isDmgToPlayer) totalDmgTaken += log.damage;
          else totalDmgDealt += log.damage;
        } else {
          totalDmgTaken += log.damage;
        }
      }
      if (log.action === "SHIELD_GAIN") {
        const m = log.detail?.match(/([\d.]+) Khiên/);
        if (m) shieldsUsed += parseInt(m[1]);
      }
      if (log.action === "SHIELD_ABSORB") shieldsUsed += log.damage;
      if (log.action === "LOOT") {
        const m = log.detail?.match(/(\d+) Nguyên Thạch/);
        if (m) lootItems.push(`${m[1]} Nguyên Thạch`);
        const mg = log.detail?.match(/(\d+) nguyên liệu/);
        if (mg) lootItems.push(`${mg[1]} nguyên liệu`);
        const gg = log.detail?.match(/(\d+) Cổ trùng/);
        if (gg) lootItems.push(`${gg[1]} Cổ trùng`);
      }
      if (log.action === "DIED" && (log.actor_name === player.username || log.actor_id === player.id)) {
        finalPlayerHP = 0;
        hpChanged = true;
      }
      if (log.action === "DAMAGE" && log.target_id) {
        const isDmgToPlayer = log.target_id === player.id;
        if (isDmgToPlayer) {
          const m = log.detail?.match(/còn (\d+) HP/);
          if (m) {
            finalPlayerHP = parseInt(m[1]);
            hpChanged = true;
          }
        }
      }
    }

    return { totalDmgDealt, totalDmgTaken, shieldsUsed, finalPlayerHP, lootItems, hpChanged };
  };

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

  const renderResultSummary = () => {
    const summary = parseResultSummary();

    return (
      <div className="bg-black/60 border border-zinc-800 rounded-xl p-4 mb-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
            <Swords className="w-4 h-4 text-amber-400" />
            Kết quả chiến đấu
          </h3>
          <span className={`px-3 py-1 rounded-lg text-xs font-bold ${simulationWinner === 'Phe Ta' ? 'bg-emerald-900/50 text-emerald-400 border border-emerald-800' : 'bg-rose-900/50 text-rose-400 border border-rose-800'}`}>
            {simulationWinner === 'Phe Ta' ? 'THẮNG LỢI' : 'THẤT BẠI'}
          </span>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
          <div className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-3 text-center">
            <div className="text-zinc-500 mb-1">HP còn lại</div>
            <div className={`font-bold text-lg ${summary.finalPlayerHP > 50 ? 'text-emerald-400' : summary.finalPlayerHP > 0 ? 'text-yellow-400' : 'text-rose-400'}`}>
              {summary.finalPlayerHP}
              <span className="text-xs text-zinc-600"> / {player.hp}</span>
            </div>
          </div>

          <div className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-3 text-center">
            <div className="text-zinc-500 mb-1">Sát thương gây</div>
            <div className="font-bold text-lg text-amber-400">{summary.totalDmgDealt.toLocaleString()}</div>
          </div>

          <div className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-3 text-center">
            <div className="text-zinc-500 mb-1">Sát thương nhận</div>
            <div className="font-bold text-lg text-rose-400">{summary.totalDmgTaken.toLocaleString()}</div>
          </div>

          <div className="bg-zinc-900/50 border border-zinc-800 rounded-lg p-3 text-center">
            <div className="text-zinc-500 mb-1">Khiên Ảo dùng</div>
            <div className="font-bold text-lg text-blue-400">{summary.shieldsUsed > 0 ? summary.shieldsUsed.toLocaleString() : '0'}</div>
          </div>
        </div>

        {summary.lootItems.length > 0 && (
          <div className="mt-3 bg-amber-950/20 border border-amber-800/30 rounded-lg p-3">
            <div className="text-xs text-amber-400 font-bold mb-1">Chiến lợi phẩm</div>
            <div className="text-xs text-zinc-300">{summary.lootItems.join(" · ") || "Không có"}</div>
          </div>
        )}
      </div>
    );
  };

  const renderLoadoutPhase = () => (
    <div className="flex flex-col gap-6">
      {/* Enemy Intel Panel */}
      {renderEnemyIntel()}

      {/* Ally Intel Panel */}
      {renderAllyIntel()}

      <div className="bg-black/40 border border-zinc-800 p-6 rounded-2xl">
        <h2 className="text-xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-cyan-400 mb-4 flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-emerald-400" />
          Chuẩn bị Đội Hình (Loadout)
        </h2>
        <p className="text-sm text-zinc-400 mb-6">
          Hệ thống chiến đấu đơn định yêu cầu Đạo Hữu sắp xếp cấu hình trước khi giao tranh.
        </p>

        {/* Core Gu */}
        <div className="mb-6">
          <h3 className="text-sm font-bold text-zinc-300 mb-3 border-b border-zinc-800 pb-2">Cổ Hạch Tâm (Bắt buộc 1)</h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {eligibleGus.map(gu => {
              const spec = guDictionary[gu.gu_id];
              const isSelected = coreGuId === gu.instance_id;
              return (
                <div
                  key={gu.instance_id}
                  onClick={() => setCoreGuId(gu.instance_id)}
                  className={`p-3 rounded-xl border cursor-pointer transition-all ${isSelected ? 'border-emerald-500 bg-emerald-950/30' : 'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'}`}
                >
                  <div className="font-bold text-sm text-zinc-200">{spec?.name || gu.gu_id}</div>
                  <div className="text-[10px] text-zinc-500 mt-1">Rank {spec?.rank} · {spec?.path || ""}</div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Support Gu */}
        <div className="mb-6">
          <h3 className="text-sm font-bold text-zinc-300 mb-3 border-b border-zinc-800 pb-2">Cổ Phụ Trợ (Tối đa 6)</h3>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {eligibleGus.filter(gu => gu.instance_id !== coreGuId).map(gu => {
              const spec = guDictionary[gu.gu_id];
              const isSelected = supportGuIds.includes(gu.instance_id);
              return (
                <div
                  key={gu.instance_id}
                  onClick={() => toggleSupportGu(gu.instance_id)}
                  className={`p-3 rounded-xl border cursor-pointer transition-all ${isSelected ? 'border-cyan-500 bg-cyan-950/30' : 'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'}`}
                >
                  <div className="font-bold text-sm text-zinc-200">{spec?.name || gu.gu_id}</div>
                  <div className="text-[10px] text-zinc-500 mt-1">Rank {spec?.rank} · {spec?.path || ""}</div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Killer Moves */}
        <div className="mb-6">
          <h3 className="text-sm font-bold text-zinc-300 mb-3 border-b border-zinc-800 pb-2 flex items-center gap-2">
            <Swords className="w-4 h-4 text-purple-400" />
            Sát Chiêu (Killer Moves)
          </h3>
          {killerMoves.length === 0 ? (
            <p className="text-sm text-zinc-500 italic">Chưa có sát chiêu nào. Vào Sát Chiêu Đường để tạo.</p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {killerMoves.map(move => {
                const isSelected = selectedKillerMoveIds.includes(move.id);
                return (
                  <div
                    key={move.id}
                    onClick={() => toggleKillerMove(move.id)}
                    className={`p-3 rounded-xl border cursor-pointer transition-all ${isSelected ? 'border-purple-500 bg-purple-950/30' : 'border-zinc-800 bg-zinc-900/50 hover:border-zinc-600'}`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="font-bold text-sm text-zinc-200">{move.name}</div>
                      <span className={`text-[10px] px-2 py-0.5 rounded-full ${move.stability >= 80 ? 'bg-emerald-900/50 text-emerald-300' : move.stability >= 50 ? 'bg-yellow-900/50 text-yellow-300' : 'bg-red-900/50 text-red-300'}`}>
                        {move.stability.toFixed(0)}%
                      </span>
                    </div>
                    <div className="text-[10px] text-zinc-500 mt-1">
                      {move.path} · DMG {move.damage_value.toFixed(0)} · Mana {move.total_mana_cost.toFixed(0)}
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
            BẮT ĐẦU MÔ PHỎNG
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
    <div className="flex flex-col gap-4 h-full">
      <div className="bg-black/60 border border-zinc-800 p-4 rounded-2xl flex justify-between items-center">
        <div>
          <h2 className="text-xl font-bold text-zinc-100 flex items-center gap-2">
            <Flame className="w-5 h-5 text-rose-500" />
            Chiến Đấu Mô Phỏng
          </h2>
          <p className="text-xs text-zinc-500 mt-1">Hệ thống đang phát lại các diễn biến trận đánh.</p>
        </div>
        <div className="flex gap-2">
          {isReplaying ? (
            <button onClick={() => setCurrentLogIndex(simulationLogs.length - 1)} className="p-2 bg-zinc-800 text-zinc-300 rounded hover:bg-zinc-700" title="Tua Nhanh">
              <FastForward className="w-4 h-4" />
            </button>
          ) : (
            <div className="flex items-center gap-2">
              <button
                onClick={() => { setCurrentLogIndex(0); setIsReplaying(true); }}
                className="px-3 py-2 bg-purple-900/50 text-purple-300 rounded hover:bg-purple-800/50 text-xs font-bold border border-purple-800 transition-all"
              >
                Xem Quá Trình
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Result Summary */}
      {!isReplaying && renderResultSummary()}

      {/* Logs Window */}
      <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded-xl p-4 overflow-y-auto font-mono text-sm space-y-3 min-h-[400px] max-h-[600px] shadow-inner">
        {simulationLogs.slice(0, currentLogIndex + 1).map((log, idx) => (
          <div key={idx} className={`p-3 rounded-lg border ${log.action === 'BATTLE_END' ? 'border-amber-500/50 bg-amber-950/20' : log.trigger ? 'border-cyan-500/30 bg-cyan-950/10' : log.action === 'DIED' ? 'border-rose-900 bg-rose-950/30 text-rose-400' : 'border-zinc-800/50 bg-zinc-900/30'}`}>
            <div className="flex items-center gap-2 mb-1">
              {log.turn > 0 && <span className="text-[10px] px-1.5 py-0.5 bg-zinc-800 text-zinc-400 rounded">Turn {log.turn}</span>}
              {log.trigger && <span className="text-[10px] px-1.5 py-0.5 bg-cyan-900 text-cyan-200 rounded flex items-center gap-1"><ShieldAlert className="w-3 h-3"/> Trigger</span>}
              {log.action === 'DIED' && <span className="text-[10px] px-1.5 py-0.5 bg-rose-900 text-rose-200 rounded flex items-center gap-1"><Skull className="w-3 h-3"/> Tử</span>}
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
        <div ref={logEndRef} />
      </div>

      {!isReplaying && (
        <button
          onClick={handleExitSimulation}
          className="w-full bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-bold py-4 rounded-xl transition-all shadow-lg"
        >
          Trở Về
        </button>
      )}
    </div>
  );

  return (
    <div className="w-full max-w-4xl mx-auto">
      {simulationLogs.length === 0 ? renderLoadoutPhase() : renderReplayPhase()}
    </div>
  );
};
