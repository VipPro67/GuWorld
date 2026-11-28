"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GamePanel, GameCard, GameTabs, GameProgressBar } from "@/components/ui";
import { CalendarClock, CheckCircle2, Crown, Swords, Users } from "lucide-react";
import { rankLabel } from "@/lib/combat";
import MartialArenaBracket from "./MartialArenaBracket";
import MartialArenaRewards from "./MartialArenaRewards";
import MartialArenaHistory from "./MartialArenaHistory";
import { TournamentEmptyState, TournamentStatusBadge } from "../Arena/TournamentShared";

type MartialArenaTab = "current" | "bracket" | "rewards" | "history";

const rankName = (r: number) => (r >= 10 ? "Vĩnh Sinh" : rankLabel(r));

export default function MartialArenaView() {
  const { martialCurrent, martialRewards, fetchMartialCurrent, fetchMartialRewards } = useGameStore();
  const [activeTab, setActiveTab] = useState<MartialArenaTab>("current");
  const pendingRewardCount = martialRewards.filter((reward) => !reward.claimed).length;

  useEffect(() => {
    fetchMartialCurrent();
    fetchMartialRewards();
  }, [fetchMartialCurrent, fetchMartialRewards]);

  const totalPlayers = martialCurrent.reduce((sum, entry) => sum + (entry.tournament?.total_players || 0), 0);

  return (
    <GamePanel className="p-4 sm:p-6 space-y-5">
      {/* Hero Header */}
      <header className="relative overflow-hidden rounded-2xl border border-amber-400/20 bg-gradient-to-br from-amber-950/45 via-zinc-950/90 to-indigo-950/35 p-5">
        <div className="pointer-events-none absolute -right-10 -top-10 h-40 w-40 rounded-full bg-amber-400/15 blur-3xl" />
        <div className="relative flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-start gap-3.5">
            <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl border border-amber-400/30 bg-amber-400/15 text-amber-200 shadow-[0_0_24px_rgba(251,191,36,0.18)]">
              <Swords className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-xl font-bold tracking-tight text-zinc-100">Đại Hội Võ Thuật</h2>
              <p className="mt-1 text-sm text-zinc-400">
                Đấu loại trực tiếp 1v1 theo từng Cảnh Giới · Tự động ghi danh đạo hữu đang online.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2 text-xs text-zinc-300">
            <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
              <CalendarClock className="h-3.5 w-3.5 text-amber-300" />
              12:00 Thứ 3 hàng tuần
            </span>
            {martialCurrent.length > 0 && (
              <>
                <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
                  <Crown className="h-3.5 w-3.5 text-amber-300" />
                  {martialCurrent.length} bảng đấu
                </span>
                <span className="inline-flex items-center gap-1.5 rounded-xl border border-white/10 bg-black/35 px-3 py-1.5">
                  <Users className="h-3.5 w-3.5 text-sky-300" />
                  {totalPlayers} võ giả
                </span>
              </>
            )}
          </div>
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
        onChange={(tab) => setActiveTab(tab as MartialArenaTab)}
      />

      <div>
        {activeTab === "current" && (
          <div className="space-y-4">
            {martialCurrent.length === 0 ? (
              <TournamentEmptyState
                icon="trophy"
                title="Chưa có bảng đấu đang mở"
                description="Mỗi 12:00 Thứ 3 hàng tuần, hệ thống tự động đưa các đạo hữu đang online (tối thiểu 8 người cùng cảnh giới) vào bảng đấu."
              />
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {martialCurrent.map((entry) => (
                  <GameCard
                    key={entry.tournament.id}
                    className="border border-white/10 bg-gradient-to-br from-amber-400/[0.06] to-zinc-950/70 p-4 space-y-3"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span className="rounded-lg border border-amber-400/25 bg-amber-400/10 px-2.5 py-1 text-xs font-bold text-amber-200">
                          {rankName(entry.rank)}
                        </span>
                        <span className="text-xs text-zinc-400">{entry.tournament.total_players} võ giả</span>
                      </div>
                      <TournamentStatusBadge status={entry.tournament.status} />
                    </div>

                    {entry.tournament.status === "IN_PROGRESS" && entry.tournament.total_rounds > 0 && (
                      <div className="space-y-1">
                        <div className="flex justify-between text-[11px] text-zinc-400">
                          <span>Vòng {entry.tournament.current_round}/{entry.tournament.total_rounds}</span>
                          <span>{Math.round((entry.tournament.current_round / entry.tournament.total_rounds) * 100)}%</span>
                        </div>
                        <GameProgressBar
                          value={entry.tournament.current_round}
                          max={entry.tournament.total_rounds}
                          variant="warning"
                        />
                      </div>
                    )}

                    {entry.my_combatant ? (
                      <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-emerald-400/25 bg-emerald-400/10 px-3 py-2 text-xs">
                        <span className="inline-flex items-center gap-1.5 font-semibold text-emerald-200">
                          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
                          Bạn tham gia · Seed #{entry.my_combatant.seed_number}
                        </span>
                        <span className="text-zinc-300">
                          {entry.my_combatant.eliminated
                            ? `Bị loại${entry.my_combatant.eliminated_at_round ? ` ở vòng ${entry.my_combatant.eliminated_at_round}` : ""}`
                            : entry.my_combatant.final_placement
                            ? `Hạng ${entry.my_combatant.final_placement}`
                            : "Đang thi đấu"}
                        </span>
                      </div>
                    ) : (
                      <div className="rounded-xl border border-white/10 bg-black/25 px-3 py-2 text-xs text-zinc-500">
                        Bạn không thuộc bảng đấu cảnh giới này
                      </div>
                    )}
                  </GameCard>
                ))}
              </div>
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
