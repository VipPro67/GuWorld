"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameCard, GameTabs } from "@/components/ui";
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
  const { martialCurrent, fetchMartialCurrent } = useGameStore();
  const [activeTab, setActiveTab] = useState<"current" | "bracket" | "rewards" | "history">("current");

  useEffect(() => {
    fetchMartialCurrent();
  }, [fetchMartialCurrent]);

  return (
    <GamePanel className="p-4">
      <h2 className="text-xl font-bold mb-1 text-center">Đại Hội Võ Thuật</h2>
      <p className="text-sm text-gray-400 text-center mb-4">
        Giải đấu 1v1 theo cảnh giới — mở định kỳ 12:00 Thứ 3 hàng tuần. Các đạo hữu online cùng cảnh giới sẽ tự động được gom vào bảng đấu.
      </p>

      <GameTabs
        items={[
          { id: "current", label: "Đang Diễn Ra" },
          { id: "bracket", label: "Bảng Đấu" },
          { id: "rewards", label: "Phần Thưởng" },
          { id: "history", label: "Lịch Sử" },
        ]}
        activeId={activeTab}
        onChange={(tab) => setActiveTab(tab as ApiResponseData)}
      />

      <div className="mt-4">
        {activeTab === "current" && (
          <div className="space-y-3">
            {martialCurrent.length === 0 ? (
              <div className="text-center text-gray-400 py-8">
                Hiện không có giải đấu nào đang diễn ra.
              </div>
            ) : (
              martialCurrent.map((entry) => (
                <GameCard key={entry.tournament.id} className="p-4">
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
