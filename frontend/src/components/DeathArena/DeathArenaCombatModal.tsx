"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameModal } from "@/components/ui";
import { Swords } from "lucide-react";
import { CombatReportView } from "../CombatReportView";
import type { CombatLogEntry } from "../CombatReplayView";

interface DeathArenaCombatModalProps {
  matchId: string | null;
  onClose: () => void;
}

interface DeathArenaMember {
  player_id?: string;
  id?: string;
  username: string;
  max_hp?: number;
  current_hp?: number;
  max_mana?: number;
  current_mana?: number;
  rank?: number;
  eliminated?: boolean;
  avatar_url?: string;
}

interface DeathArenaMatchData {
  match?: {
    round_number: number;
    team_a_id: string;
    team_b_id: string;
    team_a_name: string;
    team_b_name?: string;
    winner_team_id?: string;
    bracket_position: number;
  };
  team_a_members?: DeathArenaMember[];
  team_b_members?: DeathArenaMember[];
  combat_log?: CombatLogEntry[];
}

export default function DeathArenaCombatModal({ matchId, onClose }: DeathArenaCombatModalProps) {
  const { fetchDeathArenaMatchLog } = useGameStore();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<DeathArenaMatchData | null>(null);
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
    if (!matchId) {
      return;
    }

    let isMounted = true;
    fetchDeathArenaMatchLog(matchId)
      .then((res) => {
        if (isMounted) {
          setData(res as DeathArenaMatchData);
        }
      })
      .catch((err) => {
        console.error("Error fetching match log:", err);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [matchId, fetchDeathArenaMatchLog]);

  if (!matchId) return null;

  const match = data?.match;
  const teamAMembers = data?.team_a_members || [];
  const teamBMembers = data?.team_b_members || [];
  const logs: CombatLogEntry[] = data?.combat_log || [];

  const isTeamAWinner = match?.winner_team_id && match?.winner_team_id === match?.team_a_id;
  const isTeamBWinner = match?.winner_team_id && match?.winner_team_id === match?.team_b_id;

  // Build combatants list for CombatReportView
  const replayCombatants = [
    ...teamAMembers.map((m: DeathArenaMember, idx: number) => ({
      id: m.player_id || m.id || `teamA-${idx}`,
      name: m.username,
      kind: "ally" as const,
      max_hp: m.max_hp || 1000,
      hp: m.current_hp ?? m.max_hp ?? 1000,
      max_mana: m.max_mana || 100,
      mana: m.current_mana ?? m.max_mana ?? 100,
      grid_x: 0,
      grid_y: idx,
      rank: m.rank,
      eliminated: m.eliminated,
      avatar_url: m.avatar_url,
    })),
    ...teamBMembers.map((m: DeathArenaMember, idx: number) => ({
      id: m.player_id || m.id || `teamB-${idx}`,
      name: m.username,
      kind: "enemy" as const,
      max_hp: m.max_hp || 1000,
      hp: m.current_hp ?? m.max_hp ?? 1000,
      max_mana: m.max_mana || 100,
      mana: m.current_mana ?? m.max_mana ?? 100,
      grid_x: 4,
      grid_y: idx,
      rank: m.rank,
      eliminated: m.eliminated,
      avatar_url: m.avatar_url,
    })),
  ];

  const winnerKind = isTeamAWinner
    ? match?.team_a_name
    : isTeamBWinner
    ? match?.team_b_name || "BYE"
    : "draw";

  const getRoundTitle = (round: number) => {
    return `Chiến Báo Trận Đấu - Vòng ${round}`;
  };

  return (
    <GameModal
      open={matchId !== null}
      onClose={onClose}
      size="5xl"
      title={match ? getRoundTitle(match.round_number) : "Chiến Báo Trận Đấu"}
    >
      {loading ? (
        <div className="py-12 text-center text-gray-400 animate-pulse">
          <Swords className="w-8 h-8 mx-auto mb-2 animate-spin text-amber-500" />
          Đang tải dữ liệu chiến báo...
        </div>
      ) : !match ? (
        <div className="py-8 text-center text-red-400">
          Không tìm thấy thông tin trận đấu này.
        </div>
      ) : (
        <CombatReportView
          logs={logs}
          combatants={replayCombatants}
          winner={winnerKind}
          onBack={onClose}
          teamAName={match.team_a_name}
          teamBName={match.team_b_name || "BYE"}
          title={getRoundTitle(match.round_number)}
          extraHeaderInfo={
            <div className="text-[10px] text-zinc-500">
              Vị trí #{match.bracket_position}
            </div>
          }
        />
      )}
    </GameModal>
  );
}
