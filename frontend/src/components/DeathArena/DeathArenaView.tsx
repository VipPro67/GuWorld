"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameButton, GameCard, GameTabs } from "@/components/ui";
import { Swords, Trophy } from "lucide-react";
import DeathArenaBracket from "./DeathArenaBracket";
import DeathArenaTeamView from "./DeathArenaTeamView";
import DeathArenaRewards from "./DeathArenaRewards";
import DeathArenaHistory from "./DeathArenaHistory";

export default function DeathArenaView() {
  const {
    deathArenaTournament,
    deathArenaMyTeam,
    deathArenaLoading,
    deathArenaRewards,
    fetchDeathArenaCurrent,
    fetchDeathArenaRewards,
    registerDeathArena,
    unregisterDeathArena,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"current" | "bracket" | "team" | "rewards" | "history">("current");

  useEffect(() => {
    fetchDeathArenaCurrent();
    fetchDeathArenaRewards();
  }, [fetchDeathArenaCurrent, fetchDeathArenaRewards]);

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
  const pendingRewardCount = deathArenaRewards.filter((reward) => !reward.claimed).length;

  return (
    <GamePanel className="p-4 sm:p-5">
      <header className="mb-5 flex items-center gap-3 border-b border-white/10 pb-4">
        <div className="grid h-11 w-11 shrink-0 place-items-center rounded-xl border border-amber-300/20 bg-amber-300/10 text-amber-200">
          <Swords className="h-5 w-5" />
        </div>
        <div>
          <h2 className="text-xl font-bold text-zinc-100">Đấu Trường Sinh Tử</h2>
          <p className="mt-0.5 text-sm text-zinc-400">Lập đội, vượt từng vòng và tranh ngôi quán quân.</p>
        </div>
      </header>

      {/* Tournament Status Banner */}
      {deathArenaTournament ? (
        <div className="mb-4 rounded-xl border border-white/10 bg-gradient-to-r from-amber-300/[0.08] to-black/25 p-4">
          <div className="flex justify-between items-center">
            <span className={`flex items-center gap-2 font-semibold ${getStatusColor(deathArenaTournament.status)}`}>
              <span className="h-2 w-2 rounded-full bg-current" />
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
        <div className="mb-4 flex items-center gap-3 rounded-xl border border-white/10 bg-black/20 p-4 text-sm text-zinc-300">
          <Trophy className="h-5 w-5 shrink-0 text-amber-300/80" />
          <span>Chưa có giải đang mở. Đăng ký từ 16:00, khai chiến lúc 17:00 Thứ 7 hàng tuần.</span>
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
          { id: "rewards", label: pendingRewardCount ? `Phần thưởng · ${pendingRewardCount}` : "Phần thưởng" },
          { id: "history", label: "Lịch Sử" },
        ]}
        activeId={activeTab}
        onChange={(tab) => setActiveTab(tab as ApiResponseData)}
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
