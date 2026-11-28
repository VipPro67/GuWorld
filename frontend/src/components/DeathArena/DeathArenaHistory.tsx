"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { TournamentEmptyState, TournamentHistoryItem } from "../Arena/TournamentShared";

export default function DeathArenaHistory() {
  const { deathArenaHistory, fetchDeathArenaHistory } = useGameStore();

  useEffect(() => {
    fetchDeathArenaHistory();
  }, [fetchDeathArenaHistory]);

  if (!deathArenaHistory || deathArenaHistory.length === 0) {
    return (
      <TournamentEmptyState
        icon="trophy"
        title="Chưa có lịch sử giải đấu"
        description="Kết quả các mùa Đấu Trường Sinh Tử đã kết thúc sẽ được lưu lại tại đây."
      />
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
        <TournamentHistoryItem
          key={tournament.id}
          title={`Giải Đấu · ${formatDate(tournament.scheduled_at)}`}
          subtitle={`${tournament.total_players} chiến sĩ · ${tournament.total_teams} tổ đội`}
          championText={
            tournament.champion_team_id
              ? `Đội Vô Địch: #${tournament.champion_team_id.substring(0, 8)}`
              : null
          }
        />
      ))}
    </div>
  );
}
