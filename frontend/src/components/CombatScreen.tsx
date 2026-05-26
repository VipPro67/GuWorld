"use client";

import React, { useState, useEffect, useRef } from "react";
import { Player, CombatState, GuSpec, GuInstance, useGameStore } from "@/store/useGameStore";
import { Flame, Shield, Heart, Zap, Sparkles, Award, Play } from "lucide-react";

interface CombatScreenProps {
  player: Player;
  enemy: CombatState;
  guDictionary: Record<string, GuSpec>;
  actionLoading: boolean;
  onAttack: (equippedGuIds: string[]) => Promise<any>;
  onRetreat?: () => Promise<void>;
}

export const CombatScreen: React.FC<CombatScreenProps> = ({
  player,
  enemy,
  guDictionary,
  actionLoading,
  onAttack,
  onRetreat,
}) => {
  const [selectedGuInstanceIds, setSelectedGuInstanceIds] = useState<string[]>([]);
  const [shouldFlashRed, setShouldFlashRed] = useState(false);
  const [shouldShake, setShouldShake] = useState(false);
  const logEndRef = useRef<HTMLDivElement | null>(null);

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

  // Filter active Gus in player inventory
  const eligibleGus = (player.gu_inventory || []).filter(
    (gu) => gu.status !== "DEAD"
  );

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

      // Base mana cost logic: 10% player max mana
      let cost = 0.1 * player.max_mana;
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

      expectedDamage += (spec.base_damage || 0) * powerScale;
      expectedDefense += (spec.base_defense || 0) * powerScale;
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
    
    // Clear selection
    setSelectedGuInstanceIds([]);
    await onAttack(selectedGuInstanceIds);
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
      {/* 1. TOP VIEW: Character vs Enemy stats splits */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 items-center bg-zinc-950/30 border border-zinc-900/60 p-5 rounded-2xl">
        {/* Left: Player status */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <h4 className="text-sm font-bold text-emerald-400 font-serif flex items-center gap-1.5">
              <Sparkles className="w-4 h-4 text-emerald-500" />
              Đạo hữu: {player.username}
            </h4>
            <span className="text-[9px] font-mono text-zinc-500 uppercase tracking-widest">
              Khung Không Khiếu
            </span>
          </div>

          {/* HP Bar */}
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] font-mono">
              <span className="text-rose-400 font-semibold flex items-center gap-1">
                <Heart className="w-3.5 h-3.5" /> Thể Trạng (HP):
              </span>
              <span className="text-rose-400 font-bold">{Math.round(player.hp)}/100</span>
            </div>
            <div className="w-full bg-zinc-900 h-2.5 rounded-full overflow-hidden border border-zinc-800">
              <div
                className="bg-rose-500 h-full rounded-full transition-all duration-300 shadow-[0_0_8px_rgba(239,68,68,0.4)]"
                style={{ width: `${player.hp}%` }}
              />
            </div>
          </div>

          {/* Mana Bar */}
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] font-mono">
              <span className="text-emerald-400 font-semibold flex items-center gap-1">
                <Zap className="w-3.5 h-3.5" /> Chân Nguyên (Essence):
              </span>
              <span className="text-emerald-400 font-bold">
                {player.current_mana.toFixed(1)}/{player.max_mana}
              </span>
            </div>
            <div className="w-full bg-zinc-900 h-2.5 rounded-full overflow-hidden border border-zinc-800">
              <div
                className="bg-emerald-500 h-full rounded-full transition-all duration-300 shadow-[0_0_8px_rgba(16,185,129,0.4)]"
                style={{ width: `${(player.current_mana / player.max_mana) * 100}%` }}
              />
            </div>
          </div>

          {/* Player region & Dao Marks */}
          <div className="flex items-center justify-between text-[10px] text-zinc-500 pt-1 font-mono">
            <span>Môi trường: {player.current_region}</span>
            <span className="flex items-center gap-1 text-amber-500">
              <Award className="w-3 h-3" /> Dao Marks Lực Đạo: {player.dao_marks?.["strength"] || 0}
            </span>
          </div>
        </div>

        {/* Right: Enemy status */}
        <div className="space-y-4 md:border-l md:border-zinc-900 md:pl-8 h-full flex flex-col justify-center">
          <div className="flex justify-between items-center">
            <h4 className="text-sm font-bold text-rose-400 font-serif flex items-center gap-1.5">
              <Flame className="w-4.5 h-4.5 text-rose-500 animate-pulse" />
              Kẻ Địch: {enemy.enemy_name}
            </h4>
            <span className="text-[9px] font-mono text-zinc-500 uppercase tracking-widest">
              Dã Thú PVE
            </span>
          </div>

          {/* Enemy HP Bar */}
          <div className="space-y-1.5">
            <div className="flex justify-between text-[10px] font-mono">
              <span className="text-zinc-400">Sinh lực địch:</span>
              <span className="text-rose-400 font-extrabold">
                {Math.round(enemy.enemy_hp)}/{enemy.enemy_max_hp}
              </span>
            </div>
            <div className="w-full bg-zinc-900 h-3 rounded-full overflow-hidden border border-zinc-800">
              <div
                className="bg-gradient-to-r from-red-600 to-orange-500 h-full rounded-full transition-all duration-300 shadow-[0_0_10px_rgba(239,68,68,0.3)]"
                style={{ width: `${(enemy.enemy_hp / enemy.enemy_max_hp) * 100}%` }}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 text-[10px] text-zinc-500 font-mono">
            <div>Công Kích: <span className="text-zinc-300 font-bold">{enemy.enemy_damage}</span></div>
            <div>Phòng Ngự: <span className="text-zinc-300 font-bold">{enemy.enemy_defense}</span></div>
          </div>
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
            Không có Cổ trùng có thể sử dụng (Toàn bộ đã chết hoặc chưa sở hữu).
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {eligibleGus.map((gu) => {
              const spec = guDictionary[gu.gu_id] || {
                name: gu.gu_id,
                rank: 1,
                origin_region: "",
                base_damage: 0,
                base_defense: 0,
              };

              const isSelected = selectedGuInstanceIds.includes(gu.instance_id);
              const isNorthernSuppressed = spec.origin_region !== player.current_region;

              return (
                <div
                  key={gu.instance_id}
                  onClick={() => toggleGuSelection(gu.instance_id)}
                  className={`p-3 rounded-xl border transition-all cursor-pointer text-left relative flex flex-col justify-between h-28 ${
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
                      {spec.base_damage > 0 && (
                        <span className="text-[8px] bg-rose-950/20 border border-rose-500/10 px-1 rounded text-rose-400 font-mono">
                          Dmg: {spec.base_damage}
                        </span>
                      )}
                      {spec.base_defense > 0 && (
                        <span className="text-[8px] bg-blue-950/20 border border-blue-500/10 px-1 rounded text-blue-400 font-mono">
                          Def: {spec.base_defense}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="flex justify-between items-center mt-2 pt-2 border-t border-zinc-900">
                    <span className="text-[8px] text-zinc-500 font-mono leading-none">
                      No: {gu.hunger}/100
                    </span>
                    {isNorthernSuppressed && (
                      <span
                        className="text-[6px] text-rose-400 bg-rose-950/20 px-1 py-0.2 rounded font-bold uppercase font-mono tracking-wide"
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
              onClick={handleExecuteKillerMove}
              disabled={!canAttack}
              className={`py-3 px-6 rounded-xl font-bold font-serif text-xs tracking-wider flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                canAttack
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
