"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard, GameProgressBar } from "@/components/ui";
import { Shield, Skull, Users } from "lucide-react";
import { TournamentEmptyState } from "../Arena/TournamentShared";

export default function DeathArenaTeamView() {
  const { deathArenaMyTeam, deathArenaMyTeamMembers, deathArenaTournament, fetchDeathArenaMyTeam } = useGameStore();

  useEffect(() => {
    fetchDeathArenaMyTeam();
  }, [fetchDeathArenaMyTeam]);

  if (!deathArenaMyTeam) {
    return (
      <TournamentEmptyState
        icon="shield"
        title="Bạn chưa tham gia tổ đội nào"
        description="Hãy đăng ký tham gia Đấu Trường Sinh Tử trong khung giờ mở cổng để được ghép đội."
      />
    );
  }

  const formatHP = (current: number, max: number) => {
    const percent = max > 0 ? (current / max) * 100 : 0;
    return { percent, display: `${Math.round(current)}/${Math.round(max)}` };
  };

  const placementLabel = (place: number) => {
    if (place === 1) return "🥇 Quán Quân";
    if (place === 2) return "🥈 Á Quân";
    if (place === 3) return "🥉 Quý Quân";
    return `Hạng #${place}`;
  };

  return (
    <div className="space-y-4">
      <GameCard className="border border-white/10 bg-gradient-to-r from-zinc-900/80 to-black/40 p-4 sm:p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-xl border border-amber-400/25 bg-amber-400/10 text-amber-300">
              <Shield className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-zinc-100">{deathArenaMyTeam.team_name}</h3>
              <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-zinc-400">
                <span>Hạt giống #{deathArenaMyTeam.seed_number}</span>
                <span>·</span>
                <span className="inline-flex items-center gap-1">
                  <Users className="h-3.5 w-3.5" /> {deathArenaMyTeamMembers.length}/6 thành viên
                </span>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {deathArenaMyTeam.final_placement && (
              <span className="rounded-full border border-amber-400/35 bg-amber-400/15 px-3 py-1 text-xs font-semibold text-amber-200">
                {placementLabel(deathArenaMyTeam.final_placement)}
              </span>
            )}
            {deathArenaMyTeam.eliminated ? (
              <span className="rounded-full border border-rose-400/30 bg-rose-500/15 px-3 py-1 text-xs font-semibold text-rose-300">
                Bị loại ở vòng {deathArenaMyTeam.eliminated_at_round}
              </span>
            ) : (
              <span className="rounded-full border border-emerald-400/30 bg-emerald-400/10 px-3 py-1 text-xs font-semibold text-emerald-300">
                Đang thi đấu
              </span>
            )}
          </div>
        </div>
      </GameCard>

      <div className="grid gap-3 md:grid-cols-2">
        {deathArenaMyTeamMembers.map((member) => {
          const hp = formatHP(member.current_hp, member.max_hp);
          const mpPercent = member.max_mana > 0 ? (member.current_mana / member.max_mana) * 100 : 0;
          return (
            <GameCard
              key={member.id}
              className={`border p-3.5 transition ${
                member.eliminated
                  ? "border-rose-500/20 bg-zinc-950/70 opacity-75"
                  : "border-white/10 bg-zinc-900/65"
              }`}
            >
              <div className="mb-3 flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="truncate font-semibold text-zinc-100">{member.username}</span>
                  <span className="shrink-0 rounded-md border border-white/10 bg-white/5 px-2 py-0.5 text-[11px] font-medium text-zinc-300">
                    Chuyển {member.rank}
                  </span>
                </div>
                {member.eliminated ? (
                  <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-rose-400/30 bg-rose-500/15 px-2 py-0.5 text-[11px] font-semibold text-rose-300">
                    <Skull className="h-3 w-3" /> Hy sinh
                  </span>
                ) : (
                  <span className="shrink-0 rounded-full border border-emerald-400/25 bg-emerald-400/10 px-2 py-0.5 text-[11px] font-medium text-emerald-300">
                    Sẵn sàng
                  </span>
                )}
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <span className="w-7 text-[11px] font-semibold text-zinc-400">HP</span>
                  <div className="flex-1">
                    <GameProgressBar
                      value={hp.percent}
                      max={100}
                      variant={hp.percent > 50 ? "success" : hp.percent > 25 ? "warning" : "danger"}
                    />
                  </div>
                  <span className="w-20 text-right font-mono text-xs text-zinc-300">{hp.display}</span>
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-7 text-[11px] font-semibold text-zinc-400">MP</span>
                  <div className="flex-1">
                    <GameProgressBar value={mpPercent} max={100} variant="mana" />
                  </div>
                  <span className="w-20 text-right font-mono text-xs text-zinc-300">
                    {Math.round(member.current_mana)}/{Math.round(member.max_mana)}
                  </span>
                </div>
              </div>
            </GameCard>
          );
        })}
      </div>

      {deathArenaTournament?.status === "IN_PROGRESS" && (
        <p className="text-center text-xs text-zinc-400">
          Sinh lực và Chân Nguyên còn lại sẽ được giữ nguyên sang vòng tiếp theo nếu đội thắng trận.
        </p>
      )}
    </div>
  );
}
