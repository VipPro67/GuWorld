"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard } from "@/components/ui";
import { ScrollText } from "lucide-react";
import { rankLabel } from "@/lib/combat";
import type { MartialBracketEntry, MartialMatch } from "@/store/slices/martialArenaSlice";
import MartialArenaCombatModal from "./MartialArenaCombatModal";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

function BracketColumn({ entry }: { entry: MartialBracketEntry }) {
  const [selectedMatchId, setSelectedMatchId] = useState<string | null>(null);
  const totalRounds = entry.tournament.total_rounds;

  const matchesByRound: Record<number, MartialMatch[]> = {};
  entry.matches.forEach((match) => {
    if (!matchesByRound[match.round_number]) matchesByRound[match.round_number] = [];
    matchesByRound[match.round_number].push(match);
  });
  const rounds = Object.keys(matchesByRound).map(Number).sort((a, b) => a - b);

  const roundLabel = (round: number) => {
    if (round === totalRounds) return "Chung Kết";
    if (round === totalRounds - 1) return "Bán Kết";
    if (round === totalRounds - 2) return "Tứ Kết";
    return `Vòng ${round}`;
  };

  const statusBadge = (status: string) => {
    switch (status) {
      case "PENDING": return <span className="px-2 py-1 text-xs bg-gray-600 rounded">Chờ</span>;
      case "BYE": return <span className="px-2 py-1 text-xs bg-blue-600 rounded">BYE</span>;
      case "IN_PROGRESS": return <span className="px-2 py-1 text-xs bg-yellow-600 rounded animate-pulse">Đang đấu</span>;
      case "FINISHED": return <span className="px-2 py-1 text-xs bg-green-600 rounded">Xong</span>;
      default: return null;
    }
  };

  return (
    <div className="mb-6">
      <div className="font-bold text-lg mb-2">{rankName(entry.rank)}</div>
      <div className="overflow-x-auto">
        <div className="flex gap-4 min-w-max pb-4">
          {rounds.map((round) => (
            <div key={round} className="flex flex-col gap-2">
              <div className="text-center font-bold text-sm mb-2 text-gray-300">
                {roundLabel(round)}
              </div>
              {matchesByRound[round].map((match) => (
                <GameCard key={match.id} className="p-3 w-64">
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-xs text-gray-400">#{match.bracket_position}</span>
                    {statusBadge(match.status)}
                  </div>
                  <div className="space-y-1">
                    <div
                      className={`p-2 rounded ${
                        match.winner_combatant_id === match.combatant_a_id
                          ? "bg-green-900/50 border border-green-500"
                          : "bg-black/30"
                      }`}
                    >
                      <div className="flex justify-between items-center">
                        <span className="font-medium text-sm truncate">{match.combatant_a_name}</span>
                        {match.winner_combatant_id === match.combatant_a_id && (
                          <span className="text-green-400 text-xs">✓</span>
                        )}
                      </div>
                    </div>
                    {match.combatant_b_name ? (
                      <div
                        className={`p-2 rounded ${
                          match.winner_combatant_id === match.combatant_b_id
                            ? "bg-green-900/50 border border-green-500"
                            : "bg-black/30"
                        }`}
                      >
                        <div className="flex justify-between items-center">
                          <span className="font-medium text-sm truncate">{match.combatant_b_name}</span>
                          {match.winner_combatant_id === match.combatant_b_id && (
                            <span className="text-green-400 text-xs">✓</span>
                          )}
                        </div>
                      </div>
                    ) : (
                      <div className="p-2 rounded bg-black/30 text-gray-500 text-sm text-center">BYE</div>
                    )}
                  </div>
                  {match.status === "FINISHED" && (
                    <div className="mt-2.5 pt-2 border-t border-zinc-800/80 flex justify-end">
                      <button
                        onClick={() => setSelectedMatchId(match.id)}
                        className="flex items-center gap-1 text-[11px] px-2 py-1 rounded bg-amber-500/15 hover:bg-amber-500/25 text-amber-300 border border-amber-500/30 transition font-medium cursor-pointer"
                      >
                        <ScrollText className="w-3 h-3" /> Xem chiến báo
                      </button>
                    </div>
                  )}
                </GameCard>
              ))}
            </div>
          ))}
        </div>
      </div>

      <MartialArenaCombatModal matchId={selectedMatchId} onClose={() => setSelectedMatchId(null)} />
    </div>
  );
}

export default function MartialArenaBracket() {
  const { martialBracket, fetchMartialBracket } = useGameStore();

  useEffect(() => {
    fetchMartialBracket();
  }, [fetchMartialBracket]);

  if (!martialBracket || martialBracket.length === 0) {
    return <div className="text-center text-gray-400 py-8">Không có bảng đấu nào</div>;
  }

  return (
    <div className="space-y-4">
      {martialBracket.map((entry) => (
        <BracketColumn key={entry.tournament.id} entry={entry} />
      ))}
    </div>
  );
}
