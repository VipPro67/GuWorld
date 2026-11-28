"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameButton, GameCard, GameTabs, GameProgressBar } from "@/components/ui";
import { CalendarClock, Crown, Shield, Swords, Trophy, Users } from "lucide-react";
import DeathArenaBracket from "./DeathArenaBracket";
import DeathArenaTeamView from "./DeathArenaTeamView";
import DeathArenaRewards from "./DeathArenaRewards";
import DeathArenaHistory from "./DeathArenaHistory";
import { TournamentEmptyState, TournamentStatusBadge } from "../Arena/TournamentShared";

type DeathArenaTab = "current" | "bracket" | "team" | "rewards" | "history";

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

  const [activeTab, setActiveTab] = useState<DeathArenaTab>("current");

  useEffect(() => {
    fetchDeathArenaCurrent();
    fetchDeathArenaRewards();
  }, [fetchDeathArenaCurrent, fetchDeathArenaRewards]);

  const isRegistered = deathArenaMyTeam !== null;
  const pendingRewardCount = deathArenaRewards.filter((reward) => !reward.claimed).length;

  return (
    <GamePanel className="p-4 sm:p-6 space-y-5">
      {/* Hero Header */}
      <header className="relative overflow-hidden rounded-2xl border border-rose-500/20 bg-gradient-to-br from-rose-950/50 via-zinc-950/90 to-amber-950/30 p-5">
        <div className="pointer-events-none absolute -right-10 -top-10 h-40 w-40 rounded-full bg-rose-500/15 blur-3xl" />
        <div className="relative flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-start gap-3.5">
            <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl border border-rose-400/30 bg-rose-500/15 text-rose-200 shadow-[0_0_24px_rgba(244,63,94,0.2)]">
              <Swords className="h-6 w-6" />
            </div>
            <div>
              <div className="flex flex-wrap items-center gap-2.5">
                <h2 className="text-xl font-bold tracking-tight text-zinc-100">Đấu Trường Sinh Tử</h2>
                {deathArenaTournament && <TournamentStatusBadge status={deathArenaTournament.status} />}
              </div>
              <p className="mt-1 text-sm text-zinc-400">
                Sinh tử tổ đội 6v6 · Lập đội, giữ máu qua từng vòng và tranh ngôi quán quân.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2 text-xs text-zinc-300">
            <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
              <CalendarClock className="h-3.5 w-3.5 text-amber-300" />
              16:00 – 17:00 Thứ 7
            </span>
            {deathArenaTournament && (
              <>
                <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
                  <Users className="h-3.5 w-3.5 text-sky-300" />
                  {deathArenaTournament.total_players} người
                </span>
                <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
                  <Shield className="h-3.5 w-3.5 text-emerald-300" />
                  {deathArenaTournament.total_teams} đội
                </span>
              </>
            )}
          </div>
        </div>

        {deathArenaTournament?.status === "IN_PROGRESS" && deathArenaTournament.total_rounds > 0 && (
          <div className="relative mt-4 space-y-1.5 border-t border-white/10 pt-3.5">
            <div className="flex items-center justify-between text-xs">
              <span className="font-medium text-zinc-300">
                Tiến độ vòng đấu: Vòng {deathArenaTournament.current_round} / {deathArenaTournament.total_rounds}
              </span>
              <span className="text-rose-300 font-semibold">
                {Math.round((deathArenaTournament.current_round / deathArenaTournament.total_rounds) * 100)}%
              </span>
            </div>
            <GameProgressBar
              value={deathArenaTournament.current_round}
              max={deathArenaTournament.total_rounds}
              variant="danger"
            />
          </div>
        )}

        {deathArenaTournament?.status === "REGISTRATION" && (
          <div className="relative mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-white/10 pt-3.5">
            <p className="text-xs text-zinc-300">
              Cổng đăng ký đang mở tới <span className="font-semibold text-amber-300">17:00</span>. Hệ thống tự ghép đội tối đa 6 người.
            </p>
            {!isRegistered ? (
              <GameButton onClick={registerDeathArena} disabled={deathArenaLoading} variant="success" size="sm">
                {deathArenaLoading ? "Đang đăng ký..." : "Đăng Ký Tham Gia"}
              </GameButton>
            ) : (
              <GameButton onClick={unregisterDeathArena} disabled={deathArenaLoading} variant="warning" size="sm">
                {deathArenaLoading ? "Đang xử lý..." : "Hủy Đăng Ký"}
              </GameButton>
            )}
          </div>
        )}
      </header>

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
        onChange={(tab) => setActiveTab(tab as DeathArenaTab)}
      />

      {/* Tab Content */}
      <div>
        {activeTab === "current" && (
          <div className="grid gap-4 md:grid-cols-2">
            <GameCard className="border border-white/10 bg-zinc-900/60 p-4 sm:p-5 space-y-3">
              <div className="flex items-center gap-2 text-sm font-semibold text-zinc-100">
                <Trophy className="h-4 w-4 text-amber-300" />
                Trạng thái hiện tại
              </div>
              {deathArenaTournament ? (
                <div className="space-y-2.5 text-sm text-zinc-300">
                  <p>
                    {deathArenaTournament.status === "REGISTRATION"
                      ? "Giải đấu đang mở đăng ký. Hãy ghi danh để được xếp vào tổ đội 6 người."
                      : deathArenaTournament.status === "IN_PROGRESS"
                      ? `Các tổ đội đang giao chiến tại Vòng ${deathArenaTournament.current_round}/${deathArenaTournament.total_rounds}.`
                      : deathArenaTournament.status === "LOCKED"
                      ? "Đã khóa danh sách, đang khởi tạo bảng đấu loại trực tiếp."
                      : "Giải đấu tuần này đã khép lại. Kiểm tra mục Phần thưởng hoặc Lịch Sử."}
                  </p>
                  {isRegistered && deathArenaMyTeam ? (
                    <div className="rounded-xl border border-emerald-400/25 bg-emerald-400/10 p-3">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-emerald-200">{deathArenaMyTeam.team_name}</span>
                        <span className="text-xs font-medium text-emerald-300">
                          {deathArenaMyTeam.member_count}/6 thành viên
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-emerald-200/80">
                        Hạt giống #{deathArenaMyTeam.seed_number} · HP &amp; Chân Nguyên được bảo lưu giữa các vòng thắng.
                      </p>
                    </div>
                  ) : (
                    <div className="rounded-xl border border-white/10 bg-black/25 p-3 text-xs text-zinc-400">
                      Bạn chưa thuộc tổ đội nào trong giải đấu hiện tại.
                    </div>
                  )}
                </div>
              ) : (
                <TournamentEmptyState
                  icon="trophy"
                  title="Chưa có giải đang mở"
                  description="Cổng đăng ký tự động mở từ 16:00 – 17:00 Thứ 7 hàng tuần và khai chiến lúc 17:00."
                />
              )}
            </GameCard>

            <GameCard className="border border-white/10 bg-zinc-900/60 p-4 sm:p-5 space-y-3">
              <div className="flex items-center gap-2 text-sm font-semibold text-zinc-100">
                <Crown className="h-4 w-4 text-amber-300" />
                Cơ cấu phần thưởng Top 3
              </div>
              <div className="space-y-2 text-xs">
                <div className="flex items-center justify-between rounded-xl border border-amber-400/25 bg-amber-400/[0.07] px-3.5 py-2.5">
                  <span className="font-semibold text-amber-200">🥇 Quán Quân</span>
                  <span className="text-zinc-200">5.000 NT · 500 TNT · Thọ Cổ 500 năm</span>
                </div>
                <div className="flex items-center justify-between rounded-xl border border-slate-300/20 bg-slate-300/[0.05] px-3.5 py-2.5">
                  <span className="font-semibold text-slate-200">🥈 Á Quân</span>
                  <span className="text-zinc-200">3.000 NT · 300 TNT · Thọ Cổ 100 năm</span>
                </div>
                <div className="flex items-center justify-between rounded-xl border border-orange-400/20 bg-orange-400/[0.05] px-3.5 py-2.5">
                  <span className="font-semibold text-orange-200">🥉 Quý Quân</span>
                  <span className="text-zinc-200">1.500 NT · 150 TNT · Thọ Cổ 50 năm</span>
                </div>
              </div>
              <p className="text-[11px] text-zinc-400">
                Ngoài ra, thành viên sống sót sau mỗi trận thắng được thưởng thêm Nguyên Thạch trực tiếp theo vòng.
              </p>
            </GameCard>
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
