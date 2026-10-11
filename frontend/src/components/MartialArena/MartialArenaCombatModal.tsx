"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameModal } from "@/components/ui";
import { Swords } from "lucide-react";
import { CombatReportView } from "../CombatReportView";
import type { CombatLogEntry } from "../CombatReplayView";

interface MartialArenaCombatModalProps {
  matchId: string | null;
  onClose: () => void;
}

interface MartialArenaMatchData {
  match?: {
    round_number: number;
    combatant_a_id: string;
    combatant_b_id?: string;
    combatant_a_name: string;
    combatant_b_name?: string;
    winner_combatant_id?: string;
    bracket_position: number;
  };
  combat_log?: CombatLogEntry[];
}

export default function MartialArenaCombatModal({ matchId, onClose }: MartialArenaCombatModalProps) {
  const { fetchMartialMatchLog } = useGameStore();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<MartialArenaMatchData | null>(null);
  const [prevMatchId, setPrevMatchId] = useState<string | null>(null);

  if (matchId !== prevMatchId) {
    setPrevMatchId(matchId);
    if (matchId) {
      setLoading(true);
    } else {
      setData(null);
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!matchId) return;
    let isMounted = true;
    fetchMartialMatchLog(matchId)
      .then((res) => {
        if (isMounted) setData(res as MartialArenaMatchData);
      })
      .catch((err) => {
        console.error("Error fetching martial match log:", err);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });
    return () => {
      isMounted = false;
    };
  }, [matchId, fetchMartialMatchLog]);

  if (!matchId) return null;

  const match = data?.match;
  const logs: CombatLogEntry[] = data?.combat_log || [];

  const isAWinner = match?.winner_combatant_id && match?.winner_combatant_id === match?.combatant_a_id;
  const winnerName = isAWinner
    ? match?.combatant_a_name
    : match?.winner_combatant_id
    ? match?.combatant_b_name || "BYE"
    : "draw";

  return (
    <GameModal
      open={matchId !== null}
      onClose={onClose}
      size="5xl"
      title={match ? `Chiến Báo Trận Đấu - Vòng ${match.round_number}` : "Chiến Báo Trận Đấu"}
    >
      {loading ? (
        <div className="py-12 text-center text-gray-400 animate-pulse">
          <Swords className="w-8 h-8 mx-auto mb-2 animate-spin text-amber-500" />
          Đang tải dữ liệu chiến báo...
        </div>
      ) : !match ? (
        <div className="py-8 text-center text-red-400">Không tìm thấy thông tin trận đấu này.</div>
      ) : (
        <CombatReportView
          logs={logs}
          combatants={[]}
          winner={winnerName}
          onBack={onClose}
          teamAName={match.combatant_a_name}
          teamBName={match.combatant_b_name || "BYE"}
          title={`Chiến Báo Trận Đấu - Vòng ${match.round_number}`}
        />
      )}
    </GameModal>
  );
}
