"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Crown } from "lucide-react";
import { rankLabel } from "@/lib/combat";
import type { MartialBracketEntry } from "@/store/slices/martialArenaSlice";
import MartialArenaCombatModal from "./MartialArenaCombatModal";
import { TournamentBracketGrid, TournamentEmptyState, TournamentStatusBadge } from "../Arena/TournamentShared";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

function BracketSection({ entry }: { entry: MartialBracketEntry }) {
  const [selectedMatchId, setSelectedMatchId] = useState<string | null>(null);

  const mappedMatches = entry.matches.map((match) => ({
    id: match.id,
    round_number: match.round_number,
    bracket_position: match.bracket_position,
    status: match.status,
    sideAId: match.combatant_a_id,
    sideAName: match.combatant_a_name,
    sideBId: match.combatant_b_id,
    sideBName: match.combatant_b_name,
    winnerId: match.winner_combatant_id,
    rewardStones: match.reward_stones,
    rewardLabel: "NT",
  }));

  return (
    <div className="space-y-3 rounded-2xl border border-white/10 bg-zinc-900/40 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 pb-3">
        <div className="flex items-center gap-2">
          <Crown className="h-4 w-4 text-amber-300" />
          <h3 className="text-base font-bold text-zinc-100">{rankName(entry.rank)}</h3>
          <span className="text-xs text-zinc-400">· {entry.tournament.total_players} võ giả</span>
        </div>
        <TournamentStatusBadge status={entry.tournament.status} />
      </div>

      <TournamentBracketGrid
        matches={mappedMatches}
        totalRounds={entry.tournament.total_rounds}
        onSelectMatch={setSelectedMatchId}
      />

      <MartialArenaCombatModal matchId={selectedMatchId} onClose={() => setSelectedMatchId(null)} />
    </div>
  );
}

export default function MartialArenaBracket() {
  const { martialBracket, fetchMartialBracket } = useGameStore();
  const [selectedRank, setSelectedRank] = useState<number | "all">("all");

  useEffect(() => {
    fetchMartialBracket();
  }, [fetchMartialBracket]);

  if (!martialBracket || martialBracket.length === 0) {
    return (
      <TournamentEmptyState
        icon="swords"
        title="Không có bảng đấu nào"
        description="Bảng đấu theo từng cảnh giới sẽ hiển thị tại đây khi Đại Hội Võ Thuật khai mở."
      />
    );
  }

  const visibleEntries =
    selectedRank === "all"
      ? martialBracket
      : martialBracket.filter((entry) => entry.rank === selectedRank);

  return (
    <div className="space-y-4">
      {martialBracket.length > 1 && (
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={() => setSelectedRank("all")}
            className={`cursor-pointer rounded-xl border px-3 py-1.5 text-xs font-semibold transition ${
              selectedRank === "all"
                ? "border-amber-400/40 bg-amber-400/15 text-amber-200"
                : "border-white/10 bg-white/5 text-zinc-400 hover:text-zinc-200"
            }`}
          >
            Tất cả ({martialBracket.length})
          </button>
          {martialBracket.map((entry) => (
            <button
              key={entry.tournament.id}
              onClick={() => setSelectedRank(entry.rank)}
              className={`cursor-pointer rounded-xl border px-3 py-1.5 text-xs font-semibold transition ${
                selectedRank === entry.rank
                  ? "border-amber-400/40 bg-amber-400/15 text-amber-200"
                  : "border-white/10 bg-white/5 text-zinc-400 hover:text-zinc-200"
              }`}
            >
              {rankName(entry.rank)}
            </button>
          ))}
        </div>
      )}

      <div className="space-y-5">
        {visibleEntries.map((entry) => (
          <BracketSection key={entry.tournament.id} entry={entry} />
        ))}
      </div>
    </div>
  );
}
