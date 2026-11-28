"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import DeathArenaCombatModal from "./DeathArenaCombatModal";
import { TournamentBracketGrid, TournamentEmptyState } from "../Arena/TournamentShared";

export default function DeathArenaBracket() {
  const { deathArenaTournament, deathArenaBracket, fetchDeathArenaBracket } = useGameStore();
  const [selectedMatchId, setSelectedMatchId] = useState<string | null>(null);

  useEffect(() => {
    fetchDeathArenaBracket();
  }, [fetchDeathArenaBracket]);

  if (!deathArenaTournament) {
    return (
      <TournamentEmptyState
        icon="swords"
        title="Không có dữ liệu giải đấu"
        description="Bảng đấu sẽ xuất hiện khi giải đấu Đấu Trường Sinh Tử được mở."
      />
    );
  }

  if (deathArenaTournament.status === "REGISTRATION") {
    return (
      <TournamentEmptyState
        icon="swords"
        title="Đang trong thời gian đăng ký"
        description="Bảng đấu loại trực tiếp sẽ được khởi tạo tự động ngay khi cổng đăng ký đóng lúc 17:00."
      />
    );
  }

  const mappedMatches = deathArenaBracket.map((match) => ({
    id: match.id,
    round_number: match.round_number,
    bracket_position: match.bracket_position,
    status: match.status,
    sideAId: match.team_a_id,
    sideAName: match.team_a_name,
    sideBId: match.team_b_id,
    sideBName: match.team_b_name,
    winnerId: match.winner_team_id,
    rewardStones: match.reward_stones_per_member,
    rewardLabel: "NT/người",
  }));

  return (
    <div>
      <TournamentBracketGrid
        matches={mappedMatches}
        totalRounds={deathArenaTournament.total_rounds}
        onSelectMatch={setSelectedMatchId}
      />

      <DeathArenaCombatModal
        matchId={selectedMatchId}
        onClose={() => setSelectedMatchId(null)}
      />
    </div>
  );
}
