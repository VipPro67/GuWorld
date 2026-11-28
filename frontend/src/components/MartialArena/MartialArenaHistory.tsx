"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { rankLabel } from "@/lib/combat";
import { TournamentEmptyState, TournamentHistoryItem } from "../Arena/TournamentShared";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

export default function MartialArenaHistory() {
  const { martialHistory, fetchMartialHistory } = useGameStore();

  useEffect(() => {
    fetchMartialHistory();
  }, [fetchMartialHistory]);

  if (!martialHistory || martialHistory.length === 0) {
    return (
      <TournamentEmptyState
        icon="trophy"
        title="Chưa có lịch sử giải đấu"
        description="Lịch sử các kỳ Đại Hội Võ Thuật theo từng cảnh giới sẽ hiển thị tại đây."
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
      {martialHistory.map((tournament) => (
        <TournamentHistoryItem
          key={tournament.id}
          title={`${rankName(tournament.rank)} · ${formatDate(tournament.scheduled_at)}`}
          subtitle={`${tournament.total_players} võ giả tham gia`}
          championText={
            tournament.champion_player_id ? "Đã vinh danh Quán Quân bảng đấu" : null
          }
        />
      ))}
    </div>
  );
}
