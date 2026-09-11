"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard } from "@/components/ui";

export default function DeathArenaBracket() {
  const { deathArenaTournament, deathArenaBracket, fetchDeathArenaBracket } = useGameStore();

  useEffect(() => {
    fetchDeathArenaBracket();
  }, [fetchDeathArenaBracket]);

  if (!deathArenaTournament) {
    return <div className="text-center text-gray-400">Không có dữ liệu giải đấu</div>;
  }

  if (deathArenaTournament.status === "REGISTRATION") {
    return (
      <div className="text-center text-gray-400">
        Bảng đấu sẽ được tạo sau khi kết thúc đăng ký
      </div>
    );
  }

  // Group matches by round
  const matchesByRound: Record<number, typeof deathArenaBracket> = {};
  deathArenaBracket.forEach((match) => {
    if (!matchesByRound[match.round_number]) {
      matchesByRound[match.round_number] = [];
    }
    matchesByRound[match.round_number].push(match);
  });

  const rounds = Object.keys(matchesByRound)
    .map(Number)
    .sort((a, b) => a - b);

  const getRoundLabel = (round: number, totalRounds: number) => {
    if (round === totalRounds) return "Chung Kết";
    if (round === totalRounds - 1) return "Bán Kết";
    if (round === totalRounds - 2) return "Tứ Kết";
    return `Vòng ${round}`;
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "PENDING":
        return <span className="px-2 py-1 text-xs bg-gray-600 rounded">Chờ</span>;
      case "BYE":
        return <span className="px-2 py-1 text-xs bg-blue-600 rounded">BYE</span>;
      case "IN_PROGRESS":
        return <span className="px-2 py-1 text-xs bg-yellow-600 rounded animate-pulse">Đang đấu</span>;
      case "FINISHED":
        return <span className="px-2 py-1 text-xs bg-green-600 rounded">Xong</span>;
      default:
        return null;
    }
  };

  return (
    <div className="overflow-x-auto">
      <div className="flex gap-4 min-w-max pb-4">
        {rounds.map((round) => (
          <div key={round} className="flex flex-col gap-2">
            <div className="text-center font-bold text-sm mb-2 text-gray-300">
              {getRoundLabel(round, deathArenaTournament.total_rounds)}
            </div>
            {matchesByRound[round].map((match) => (
              <GameCard key={match.id} className="p-3 w-64">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-xs text-gray-400">#{match.bracket_position}</span>
                  {getStatusBadge(match.status)}
                </div>
                <div className="space-y-1">
                  <div
                    className={`p-2 rounded ${
                      match.winner_team_id === match.team_a_id
                        ? "bg-green-900/50 border border-green-500"
                        : "bg-black/30"
                    }`}
                  >
                    <div className="flex justify-between items-center">
                      <span className="font-medium text-sm truncate">{match.team_a_name}</span>
                      {match.winner_team_id === match.team_a_id && (
                        <span className="text-green-400 text-xs">✓</span>
                      )}
                    </div>
                  </div>
                  {match.team_b_name ? (
                    <div
                      className={`p-2 rounded ${
                        match.winner_team_id === match.team_b_id
                          ? "bg-green-900/50 border border-green-500"
                          : "bg-black/30"
                      }`}
                    >
                      <div className="flex justify-between items-center">
                        <span className="font-medium text-sm truncate">{match.team_b_name}</span>
                        {match.winner_team_id === match.team_b_id && (
                          <span className="text-green-400 text-xs">✓</span>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="p-2 rounded bg-black/30 text-gray-500 text-sm text-center">
                      BYE
                    </div>
                  )}
                </div>
                {match.reward_stones_per_member > 0 && (
                  <div className="mt-2 text-xs text-gray-400 text-center">
                    +{match.reward_stones_per_member} Nguyên Thạch/người
                  </div>
                )}
              </GameCard>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}