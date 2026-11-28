"use client";

import type { ReactNode } from "react";
import { GameCard } from "@/components/ui";
import { CheckCircle2, Coins, Crown, ScrollText, Shield, Sparkles, Swords, Trophy } from "lucide-react";

export interface BracketMatchItem {
  id: string;
  round_number: number;
  bracket_position: number;
  status: string;
  sideAId: string;
  sideAName: string;
  sideBId?: string;
  sideBName?: string;
  winnerId?: string;
  rewardStones?: number;
  rewardLabel?: string;
}

export function TournamentEmptyState({
  icon = "trophy",
  title,
  description,
}: {
  icon?: "trophy" | "swords" | "shield";
  title: string;
  description: string;
}) {
  const Icon = icon === "swords" ? Swords : icon === "shield" ? Shield : Trophy;
  return (
    <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-gradient-to-b from-zinc-900/70 to-black/40 px-6 py-12 text-center">
      <div className="mx-auto mb-3.5 grid h-12 w-12 place-items-center rounded-2xl border border-amber-400/25 bg-amber-400/10 text-amber-300 shadow-[0_0_24px_rgba(251,191,36,0.12)]">
        <Icon className="h-6 w-6" />
      </div>
      <p className="text-base font-semibold text-zinc-100">{title}</p>
      <p className="mx-auto mt-1.5 max-w-md text-sm text-zinc-400">{description}</p>
    </div>
  );
}

export function TournamentStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "REGISTRATION":
      return (
        <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-400/30 bg-emerald-400/10 px-2.5 py-1 text-xs font-semibold text-emerald-300">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
          Đang mở đăng ký
        </span>
      );
    case "LOCKED":
      return (
        <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-400/30 bg-amber-400/10 px-2.5 py-1 text-xs font-semibold text-amber-300">
          <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
          Đã khóa · Tạo bảng đấu
        </span>
      );
    case "IN_PROGRESS":
      return (
        <span className="inline-flex items-center gap-1.5 rounded-full border border-rose-400/35 bg-rose-500/15 px-2.5 py-1 text-xs font-semibold text-rose-300">
          <span className="h-1.5 w-1.5 rounded-full bg-rose-400 animate-ping" />
          Đang giao chiến
        </span>
      );
    case "FINISHED":
      return (
        <span className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-xs font-medium text-zinc-300">
          Đã kết thúc
        </span>
      );
    default:
      return (
        <span className="inline-flex items-center rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-xs text-zinc-300">
          {status}
        </span>
      );
  }
}

export function TournamentBracketGrid({
  matches,
  totalRounds,
  onSelectMatch,
}: {
  matches: BracketMatchItem[];
  totalRounds: number;
  onSelectMatch: (matchId: string) => void;
}) {
  if (!matches || matches.length === 0) {
    return (
      <TournamentEmptyState
        icon="swords"
        title="Chưa có dữ liệu bảng đấu"
        description="Các cặp đấu sẽ hiển thị tại đây ngay khi vòng đấu được khởi tạo."
      />
    );
  }

  const matchesByRound: Record<number, BracketMatchItem[]> = {};
  for (const match of matches) {
    if (!matchesByRound[match.round_number]) {
      matchesByRound[match.round_number] = [];
    }
    matchesByRound[match.round_number].push(match);
  }
  const rounds = Object.keys(matchesByRound)
    .map(Number)
    .sort((a, b) => a - b);

  const getRoundLabel = (round: number) => {
    if (totalRounds > 0) {
      if (round === totalRounds) return "Chung Kết";
      if (round === totalRounds - 1 && totalRounds >= 2) return "Bán Kết";
      if (round === totalRounds - 2 && totalRounds >= 3) return "Tứ Kết";
    }
    return `Vòng ${round}`;
  };

  const getMatchBadge = (status: string) => {
    switch (status) {
      case "PENDING":
        return (
          <span className="rounded-md border border-white/10 bg-white/5 px-2 py-0.5 text-[11px] font-medium text-zinc-300">
            Chờ đấu
          </span>
        );
      case "BYE":
        return (
          <span className="rounded-md border border-sky-400/30 bg-sky-400/10 px-2 py-0.5 text-[11px] font-semibold text-sky-300">
            Miễn đấu
          </span>
        );
      case "IN_PROGRESS":
        return (
          <span className="inline-flex items-center gap-1 rounded-md border border-amber-400/35 bg-amber-400/15 px-2 py-0.5 text-[11px] font-semibold text-amber-200 animate-pulse">
            <Swords className="h-3 w-3" /> Đang đấu
          </span>
        );
      case "FINISHED":
        return (
          <span className="rounded-md border border-emerald-400/30 bg-emerald-400/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-300">
            Hoàn tất
          </span>
        );
      default:
        return null;
    }
  };

  return (
    <div className="overflow-x-auto pb-2">
      <div className="flex min-w-max gap-5 pb-2">
        {rounds.map((round) => {
          const isFinal = totalRounds > 0 && round === totalRounds;
          return (
            <div key={round} className="flex w-72 flex-col gap-3">
              <div
                className={`flex items-center justify-between rounded-xl border px-3.5 py-2 text-xs font-semibold uppercase tracking-wider ${
                  isFinal
                    ? "border-amber-400/35 bg-gradient-to-r from-amber-400/15 to-amber-500/5 text-amber-200"
                    : "border-white/10 bg-white/[0.04] text-zinc-300"
                }`}
              >
                <span className="flex items-center gap-1.5">
                  {isFinal && <Crown className="h-3.5 w-3.5 text-amber-300" />}
                  {getRoundLabel(round)}
                </span>
                <span className="text-[11px] font-normal text-zinc-400">
                  {matchesByRound[round].length} trận
                </span>
              </div>

              <div className="flex flex-col gap-3">
                {matchesByRound[round].map((match) => {
                  const aWon = match.winnerId === match.sideAId;
                  const bWon = Boolean(match.sideBId && match.winnerId === match.sideBId);
                  const finished = match.status === "FINISHED";

                  return (
                    <GameCard
                      key={match.id}
                      className={`group border p-3 transition-all ${
                        isFinal
                          ? "border-amber-400/25 bg-gradient-to-b from-amber-400/[0.06] to-zinc-950/70"
                          : "border-white/10 bg-zinc-900/65 hover:border-white/20"
                      }`}
                    >
                      <div className="mb-2.5 flex items-center justify-between">
                        <span className="font-mono text-[11px] text-zinc-500">
                          Trận #{match.bracket_position}
                        </span>
                        {getMatchBadge(match.status)}
                      </div>

                      <div className="space-y-1.5">
                        <div
                          className={`flex items-center justify-between gap-2 rounded-lg border px-2.5 py-2 transition ${
                            aWon
                              ? "border-emerald-400/40 bg-emerald-500/12 text-emerald-100"
                              : finished && bWon
                              ? "border-white/5 bg-black/25 text-zinc-500"
                              : "border-white/10 bg-black/30 text-zinc-200"
                          }`}
                        >
                          <span className="truncate text-sm font-medium">{match.sideAName}</span>
                          {aWon && <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-400" />}
                        </div>

                        {match.sideBName ? (
                          <div
                            className={`flex items-center justify-between gap-2 rounded-lg border px-2.5 py-2 transition ${
                              bWon
                                ? "border-emerald-400/40 bg-emerald-500/12 text-emerald-100"
                                : finished && aWon
                                ? "border-white/5 bg-black/25 text-zinc-500"
                                : "border-white/10 bg-black/30 text-zinc-200"
                            }`}
                          >
                            <span className="truncate text-sm font-medium">{match.sideBName}</span>
                            {bWon && <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-400" />}
                          </div>
                        ) : (
                          <div className="rounded-lg border border-dashed border-white/10 bg-black/20 px-2.5 py-2 text-center text-xs text-zinc-500">
                            Miễn đấu (BYE)
                          </div>
                        )}
                      </div>

                      {(Boolean(match.rewardStones && match.rewardStones > 0) || finished) && (
                        <div className="mt-2.5 flex items-center justify-between gap-2 border-t border-white/10 pt-2.5">
                          {match.rewardStones && match.rewardStones > 0 ? (
                            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-amber-300/90">
                              <Coins className="h-3 w-3" />+{match.rewardStones} {match.rewardLabel || "Nguyên Thạch"}
                            </span>
                          ) : (
                            <span />
                          )}
                          {finished && (
                            <button
                              onClick={() => onSelectMatch(match.id)}
                              className="inline-flex cursor-pointer items-center gap-1 rounded-lg border border-amber-400/30 bg-amber-400/10 px-2.5 py-1 text-[11px] font-semibold text-amber-200 transition hover:bg-amber-400/20"
                            >
                              <ScrollText className="h-3 w-3" /> Chiến báo
                            </button>
                          )}
                        </div>
                      )}
                    </GameCard>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function TournamentHistoryItem({
  title,
  subtitle,
  championText,
  extra,
}: {
  title: string;
  subtitle: string;
  championText?: string | null;
  extra?: ReactNode;
}) {
  return (
    <GameCard className="border border-white/10 bg-gradient-to-r from-zinc-900/80 to-black/40 p-4 transition hover:border-white/20">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-amber-400/20 bg-amber-400/10 text-amber-300">
            <Sparkles className="h-4 w-4" />
          </div>
          <div>
            <div className="font-semibold text-zinc-100">{title}</div>
            <div className="mt-0.5 text-xs text-zinc-400">{subtitle}</div>
          </div>
        </div>
        <span className="rounded-full border border-white/10 bg-white/5 px-2.5 py-1 text-xs font-medium text-zinc-300">
          Đã kết thúc
        </span>
      </div>
      {championText && (
        <div className="mt-3 flex items-center gap-2 rounded-lg border border-amber-400/20 bg-amber-400/[0.07] px-3 py-2 text-xs font-medium text-amber-200">
          <Trophy className="h-3.5 w-3.5 shrink-0 text-amber-300" />
          <span>{championText}</span>
        </div>
      )}
      {extra}
    </GameCard>
  );
}
