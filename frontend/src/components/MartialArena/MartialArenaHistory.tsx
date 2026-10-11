"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard } from "@/components/ui";
import { rankLabel } from "@/lib/combat";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

export default function MartialArenaHistory() {
  const { martialHistory, fetchMartialHistory } = useGameStore();

  useEffect(() => {
    fetchMartialHistory();
  }, [fetchMartialHistory]);

  if (!martialHistory || martialHistory.length === 0) {
    return <div className="text-center text-gray-400">Chưa có lịch sử giải đấu</div>;
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
        <GameCard key={tournament.id} className="p-4">
          <div className="flex justify-between items-start">
            <div>
              <div className="font-bold">{rankName(tournament.rank)} — {formatDate(tournament.scheduled_at)}</div>
              <div className="text-sm text-gray-400">{tournament.total_players} người tham gia</div>
            </div>
            <span className="px-3 py-1 text-sm bg-gray-600 rounded">Đã kết thúc</span>
          </div>
          {tournament.champion_player_id && (
            <div className="mt-2 text-sm text-yellow-400">🏆 Đã có nhà vô địch</div>
          )}
        </GameCard>
      ))}
    </div>
  );
}
