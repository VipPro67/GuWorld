"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameCard, GameTabs } from "@/components/ui";
import { Swords, Trophy } from "lucide-react";
import { rankLabel } from "@/lib/combat";
import MartialArenaBracket from "./MartialArenaBracket";
import MartialArenaRewards from "./MartialArenaRewards";
import MartialArenaHistory from "./MartialArenaHistory";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

const statusLabel = (status: string) => {
  switch (status) {
    case "LOCKED": return "Đang tạo bảng đấu";
    case "IN_PROGRESS": return "Đang diễn ra";
    case "FINISHED": return "Đã kết thúc";
    default: return status;
  }
};

const statusColor = (status: string) => {
  switch (status) {
    case "LOCKED": return "text-yellow-400";
    case "IN_PROGRESS": return "text-red-400";
    case "FINISHED": return "text-gray-400";
    default: return "text-white";
  }
};

export default function MartialArenaView() {
  const { martialCurrent, martialRewards, fetchMartialCurrent, fetchMartialRewards } = useGameStore();
  const [activeTab, setActiveTab] = useState<"current" | "bracket" | "rewards" | "history">("current");
  const pendingRewardCount = martialRewards.filter((reward) => !reward.claimed).length;

  useEffect(() => {
    fetchMartialCurrent();
    fetchMartialRewards();
  }, [fetchMartialCurrent, fetchMartialRewards]);

  return (
    <GamePanel className="p-4 sm:p-5">
      <header className="mb-5 flex items-center gap-3 border-b border-white/10 pb-4">
        <div className="grid h-11 w-11 shrink-0 place-items-center rounded-xl border border-amber-300/20 bg-amber-300/10 text-amber-200">
          <Swords className="h-5 w-5" />
        </div>
        <div>
          <h2 className="text-xl font-bold text-zinc-100">Đại Hội Võ Thuật</h2>
          <p className="mt-0.5 text-sm text-zinc-400">Đấu loại trực tiếp 1v1 giữa các đạo hữu cùng cảnh giới.</p>
        </div>
      </header>

      <GameTabs
        items={[
          { id: "current", label: "Đang Diễn Ra" },
          { id: "bracket", label: "Bảng Đấu" },
          { id: "rewards", label: pendingRewardCount ? `Phần thưởng · ${pendingRewardCount}` : "Phần thưởng" },
          { id: "history", label: "Lịch Sử" },
        ]}
        activeId={activeTab}
        onChange={(tab) => setActiveTab(tab as ApiResponseData)}
      />

      <div className="mt-4">
        {activeTab === "current" && (
          <div className="space-y-3">
            {martialCurrent.length === 0 ? (
              <div className="rounded-xl border border-white/10 bg-black/20 px-6 py-10 text-center">
                <Trophy className="mx-auto mb-3 h-8 w-8 text-amber-300/70" />
                <p className="font-semibold text-zinc-200">Chưa có bảng đấu đang mở</p>
                <p className="mt-1 text-sm text-zinc-400">Người chơi đủ điều kiện và đang online sẽ được đưa vào giải tự động.</p>
              </div>
            ) : (
              martialCurrent.map((entry) => (
                <GameCard key={entry.tournament.id} className="border border-white/10 bg-gradient-to-r from-amber-300/[0.06] to-black/20 p-4">
                  <div className="flex justify-between items-center">
                    <div className="font-bold">
                      {rankName(entry.rank)} — {statusLabel(entry.tournament.status)}
                    </div>
                    <span className={`font-semibold ${statusColor(entry.tournament.status)}`}>
                      {entry.tournament.status === "IN_PROGRESS"
                        ? `Vòng ${entry.tournament.current_round}/${entry.tournament.total_rounds}`
                        : entry.tournament.status === "FINISHED"
                        ? "Đã kết thúc"
                        : ""}
                    </span>
                  </div>
                  <div className="text-sm text-gray-400 mt-1">
                    {entry.tournament.total_players} người tham gia
                  </div>
                  {entry.my_combatant ? (
                    <div className="mt-2 text-sm">
                      <span className="text-green-400">Bạn tham gia</span>
                      <span className="text-gray-400 ml-2">
                        Seed #{entry.my_combatant.seed_number}
                        {entry.my_combatant.eliminated
                          ? ` — bị loại${entry.my_combatant.eliminated_at_round ? ` ở vòng ${entry.my_combatant.eliminated_at_round}` : ""}`
                          : entry.my_combatant.final_placement
                          ? ` — Hạng ${entry.my_combatant.final_placement}`
                          : ""}
                      </span>
                    </div>
                  ) : (
                    <div className="mt-2 text-sm text-gray-500">Bạn không thuộc bảng đấu này</div>
                  )}
                </GameCard>
              ))
            )}
          </div>
        )}
        {activeTab === "bracket" && <MartialArenaBracket />}
        {activeTab === "rewards" && <MartialArenaRewards />}
        {activeTab === "history" && <MartialArenaHistory />}
      </div>
    </GamePanel>
  );
}
