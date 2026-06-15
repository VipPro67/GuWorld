"use client";

import React, { useState, useEffect, useRef } from "react";
import { Player, CombatState, GuSpec, GuInstance, useGameStore } from "@/store/useGameStore";
import { Flame, Sparkles, Play, FastForward, ShieldAlert } from "lucide-react";

interface CombatScreenProps {
  player: Player;
  enemy: CombatState;
  guDictionary: Record<string, GuSpec>;
  actionLoading: boolean;
  onAttack?: () => Promise<unknown>; // Optional now
  onRetreat?: () => Promise<void>;
}

export const CombatScreen: React.FC<CombatScreenProps> = ({
  player,
  enemy,
  guDictionary,
  actionLoading,
  onRetreat,
}) => {
  const [coreGuId, setCoreGuId] = useState<string>("");
  const [supportGuIds, setSupportGuIds] = useState<string[]>([]);
  
  const [simulationLogs, setSimulationLogs] = useState<any[]>([]);
  const [currentLogIndex, setCurrentLogIndex] = useState(-1);
  const [isReplaying, setIsReplaying] = useState(false);
  const [simulationWinner, setSimulationWinner] = useState<string | null>(null);

  const executeAutoCombat = useGameStore((state) => state.executeAutoCombat);
  const fetchPlayerStatus = useGameStore((state) => state.fetchPlayerStatus);
  const logEndRef = useRef<HTMLDivElement | null>(null);

  // Auto-scroll combat logs
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [currentLogIndex]);

  // Replay interval
  useEffect(() => {
    if (isReplaying && currentLogIndex < simulationLogs.length - 1) {
      const timer = setTimeout(() => {
        setCurrentLogIndex((prev) => prev + 1);
      }, 500); // 500ms per log line
      return () => clearTimeout(timer);
    } else if (isReplaying && currentLogIndex >= simulationLogs.length - 1) {
      setIsReplaying(false);
    }
  }, [isReplaying, currentLogIndex, simulationLogs]);

  const eligibleGus = (player.gu_inventory || []).filter((gu) => gu.status === "ACTIVE" && gu.equipped);

  const handleStartSimulation = async () => {
    if (!coreGuId) {
      alert("Vui lòng chọn ít nhất 1 Cổ Hạch Tâm!");
      return;
    }

    const loadout = {
      CoreGuId: eligibleGus.find(g => g.instance_id === coreGuId)?.gu_id || coreGuId,
      SupportGuIds: supportGuIds.map(id => eligibleGus.find(g => g.instance_id === id)?.gu_id || id),
      KillerMoveIds: [],
      ImmortalGuId: null
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

  const handleExitSimulation = async () => {
    await fetchPlayerStatus(player.id);
  };

  const renderLoadoutPhase = () => (
    <div className="flex flex-col gap-6">
      <div className="bg-black/40 border border-zinc-800 p-6 rounded-2xl">
        <h2 className="text-xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-cyan-400 mb-4 flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-emerald-400" />
          Chuẩn bị Đội Hình (Loadout)
        </h2>
        <p className="text-sm text-zinc-400 mb-6">
          Hệ thống chiến đấu đơn định yêu cầu Đạo Hữu sắp xếp cấu hình trước khi giao tranh.
        </p>

        {/* Core Gu Selection */}
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
                  <div className="text-[10px] text-zinc-500 mt-1">Rank {spec?.rank}</div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Support Gu Selection */}
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
                  <div className="text-[10px] text-zinc-500 mt-1">Rank {spec?.rank}</div>
                </div>
              );
            })}
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
              <span className={`px-4 py-2 rounded-lg font-bold text-sm ${simulationWinner === 'Phe Ta' ? 'bg-emerald-950/50 text-emerald-400 border border-emerald-900' : 'bg-rose-950/50 text-rose-400 border border-rose-900'}`}>
                Kết Quả: {simulationWinner}
              </span>
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

      {/* Logs Window */}
      <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded-xl p-4 overflow-y-auto font-mono text-sm space-y-3 min-h-[400px] max-h-[600px] shadow-inner">
        {simulationLogs.slice(0, currentLogIndex + 1).map((log, idx) => (
          <div key={idx} className={`p-3 rounded-lg border ${log.action === 'BATTLE_END' ? 'border-amber-500/50 bg-amber-950/20' : log.trigger ? 'border-cyan-500/30 bg-cyan-950/10' : log.action === 'DIED' ? 'border-rose-900 bg-rose-950/30 text-rose-400' : 'border-zinc-800/50 bg-zinc-900/30'}`}>
            <div className="flex items-center gap-2 mb-1">
               {log.turn > 0 && <span className="text-[10px] px-1.5 py-0.5 bg-zinc-800 text-zinc-400 rounded">Turn {log.turn}</span>}
               {log.trigger && <span className="text-[10px] px-1.5 py-0.5 bg-cyan-900 text-cyan-200 rounded flex items-center gap-1"><ShieldAlert className="w-3 h-3"/> Trigger</span>}
            </div>
            <div className={`${log.action === 'BATTLE_END' ? 'text-amber-400 font-bold' : 'text-zinc-300'}`}>
              {log.detail}
            </div>
            {log.damage > 0 && (
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
