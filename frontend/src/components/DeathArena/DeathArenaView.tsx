"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameButton, GameCard, GameTabs } from "@/components/ui";
import DeathArenaBracket from "./DeathArenaBracket";
import DeathArenaTeamView from "./DeathArenaTeamView";
import DeathArenaRewards from "./DeathArenaRewards";
import DeathArenaHistory from "./DeathArenaHistory";

export default function DeathArenaView() {
  const {
    deathArenaTournament,
    deathArenaMyTeam,
    deathArenaLoading,
    fetchDeathArenaCurrent,
    registerDeathArena,
    unregisterDeathArena,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"current" | "bracket" | "team" | "rewards" | "history">("current");

  useEffect(() => {
    fetchDeathArenaCurrent();
  }, [fetchDeathArenaCurrent]);

  const getStatusLabel = (status: string) => {
    switch (status) {
      case "REGISTRATION": return "Đang mở đăng ký";
      case "LOCKED": return "Đã khóa - Đang tạo bảng đấu";
      case "IN_PROGRESS": return "Đang diễn ra";
      case "FINISHED": return "Đã kết thúc";
      default: return status;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "REGISTRATION": return "text-green-400";
      case "LOCKED": return "text-yellow-400";
      case "IN_PROGRESS": return "text-red-400";
      case "FINISHED": return "text-gray-400";
      default: return "text-white";
    }
  };

  const isRegistered = deathArenaMyTeam !== null;

  return (
    <GamePanel className="p-4">
      <h2 className="text-xl font-bold mb-4 text-center">Đấu Trường Sinh Tử</h2>
      
      {/* Tournament Status Banner */}
      {deathArenaTournament ? (
        <div className="mb-4 p-3 bg-black/30 rounded-lg">
          <div className="flex justify-between items-center">
            <span className={`font-semibold ${getStatusColor(deathArenaTournament.status)}`}>
              {getStatusLabel(deathArenaTournament.status)}
            </span>
            <span className="text-sm text-gray-400">
              {deathArenaTournament.total_players} người tham gia | {deathArenaTournament.total_teams} đội
            </span>
          </div>
          {deathArenaTournament.status === "REGISTRATION" && (
            <div className="mt-2 text-sm text-gray-300">
              Thời gian đăng ký: 16:00 - 17:00 Thứ 7 (Khai chiến: 17:00 chiều Thứ 7)
            </div>
          )}
          {deathArenaTournament.status === "IN_PROGRESS" && (
            <div className="mt-2 text-sm text-gray-300">
              Vòng {deathArenaTournament.current_round} / {deathArenaTournament.total_rounds}
            </div>
          )}
        </div>
      ) : (
        <div className="mb-4 p-3 bg-black/30 rounded-lg text-center text-gray-400">
          Chưa có giải đấu nào. Giải đấu tiếp theo: 17:00 Thứ 7 hàng tuần (Mở đăng ký từ 16:00)
        </div>
      )}

      {/* Registration Buttons */}
      {deathArenaTournament?.status === "REGISTRATION" && (
        <div className="mb-4 flex justify-center gap-4">
          {!isRegistered ? (
            <GameButton
              onClick={registerDeathArena}
              disabled={deathArenaLoading}
              variant="success"
            >
              {deathArenaLoading ? "Đang đăng ký..." : "Đăng Ký Tham Gia"}
            </GameButton>
          ) : (
            <GameButton
              onClick={unregisterDeathArena}
              disabled={deathArenaLoading}
              variant="warning"
            >
              {deathArenaLoading ? "Đang xử lý..." : "Hủy Đăng Ký"}
            </GameButton>
          )}
        </div>
      )}

      {/* Tabs */}
      <GameTabs
        items={[
          { id: "current", label: "Giải Đấu" },
          { id: "bracket", label: "Bảng Đấu" },
          { id: "team", label: "Đội Của Tôi" },
          { id: "rewards", label: "Phần Thưởng" },
          { id: "history", label: "Lịch Sử" },
        ]}
        activeId={activeTab}
        onChange={(tab) => setActiveTab(tab as any)}
      />

      {/* Tab Content */}
      <div className="mt-4">
        {activeTab === "current" && (
          <div className="text-center text-gray-300">
            {deathArenaTournament ? (
              <div>
                <p>Giải đấu {deathArenaTournament.status === "REGISTRATION" ? "đang mở đăng ký" : "đang diễn ra"}</p>
                {isRegistered && deathArenaMyTeam && (
                  <div className="mt-2">
                    <p className="text-green-400">Bạn đã đăng ký vào đội: {deathArenaMyTeam.team_name}</p>
                    <p className="text-sm text-gray-400">{deathArenaMyTeam.member_count}/6 thành viên</p>
                  </div>
                )}
              </div>
            ) : (
              <p>Đang tải thông tin...</p>
            )}
          </div>
        )}
        {activeTab === "bracket" && <DeathArenaBracket />}
        {activeTab === "team" && <DeathArenaTeamView />}
        {activeTab === "rewards" && <DeathArenaRewards />}
        {activeTab === "history" && <DeathArenaHistory />}
      </div>
    </GamePanel>
  );
}