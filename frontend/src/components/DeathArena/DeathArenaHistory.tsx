"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard } from "@/components/ui";

export default function DeathArenaHistory() {
  const { deathArenaHistory, fetchDeathArenaHistory } = useGameStore();

  useEffect(() => {
    fetchDeathArenaHistory();
  }, [fetchDeathArenaHistory]);

  if (!deathArenaHistory || deathArenaHistory.length === 0) {
    return (
      <div className="text-center text-gray-400">
        Chưa có lịch sử giải đấu
      </div>
    );
  }

  const formatDate = (dateStr: string) => {
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString("vi-VN", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className="space-y-3">
      {deathArenaHistory.map((tournament) => (
        <GameCard key={tournament.id} className="p-4">
          <div className="flex justify-between items-start">
            <div>
              <div className="font-bold">Giải Đấu {formatDate(tournament.scheduled_at)}</div>
              <div className="text-sm text-gray-400">
                {tournament.total_players} người | {tournament.total_teams} đội
              </div>
            </div>
            <span className="px-3 py-1 text-sm bg-gray-600 rounded">Đã kết thúc</span>
          </div>
          {tournament.champion_team_id && (
            <div className="mt-2 text-sm text-yellow-400">
              🏆 Vô địch: Đội #{tournament.champion_team_id.substring(0, 8)}
            </div>
          )}
        </GameCard>
      ))}
    </div>
  );
}