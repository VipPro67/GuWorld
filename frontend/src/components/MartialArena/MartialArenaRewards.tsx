"use client";

import { useGameStore } from "@/store/useGameStore";
import { GameCard, GameButton } from "@/components/ui";
import { Check, Crown, Gem, Trophy } from "lucide-react";
import { formatNumber } from "@/lib/utils";

export default function MartialArenaRewards() {
  const { martialRewards, claimMartialReward, martialLoading } = useGameStore();

  if (!martialRewards || martialRewards.length === 0) {
    return (
      <div className="rounded-xl border border-white/10 bg-black/20 px-6 py-10 text-center">
        <Trophy className="mx-auto mb-3 h-8 w-8 text-amber-300/70" />
        <p className="font-semibold text-zinc-200">Chưa có thưởng xếp hạng</p>
        <p className="mt-1 text-sm text-zinc-400">Phần thưởng được ghi sau khi giải đấu kết thúc. Ba vị trí đầu có thể nhận tại đây.</p>
      </div>
    );
  }

  const placementLabel = (placement: number) => {
    switch (placement) {
      case 1: return "Quán Quân";
      case 2: return "Á Quân";
      case 3: return "Quý Quân";
      default: return `Hạng ${placement}`;
    }
  };

  const placementColor = (placement: number) => {
    switch (placement) {
      case 1: return "border-amber-300/35 bg-gradient-to-br from-amber-400/10 to-zinc-950/50";
      case 2: return "border-slate-300/25 bg-gradient-to-br from-slate-300/10 to-zinc-950/50";
      case 3: return "border-orange-400/25 bg-gradient-to-br from-orange-400/10 to-zinc-950/50";
      default: return "border-white/10 bg-black/20";
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3 px-1">
        <div>
          <h3 className="font-semibold text-zinc-100">Thưởng xếp hạng</h3>
          <p className="text-xs text-zinc-400">Nhận thưởng để cộng tài nguyên và danh hiệu vào nhân vật.</p>
        </div>
        <span className="shrink-0 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-zinc-300">{martialRewards.filter((reward) => !reward.claimed).length} đang chờ</span>
      </div>
      {martialRewards.map((reward) => (
        <GameCard key={reward.id} className={`overflow-hidden border p-4 sm:p-5 ${placementColor(reward.placement)}`}>
          <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
            <div>
              <div className="flex items-center gap-2 text-lg font-bold text-zinc-100"><Trophy className="h-5 w-5 text-amber-300" />{placementLabel(reward.placement)}</div>
              <div className="mt-1 text-sm text-zinc-400">{reward.title}</div>
            </div>
            {reward.claimed ? (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-400/25 bg-emerald-400/10 px-3 py-1.5 text-xs font-semibold text-emerald-300"><Check className="h-3.5 w-3.5" />Đã nhận</span>
            ) : (
              <GameButton onClick={() => claimMartialReward(reward.id)} disabled={martialLoading} variant="success" size="sm">
                {martialLoading ? "Đang nhận..." : "Nhận thưởng"}
              </GameButton>
            )}
          </div>
          <div className="grid gap-2 border-t border-white/10 pt-4 sm:grid-cols-2">
            <div className="flex items-center gap-2.5 rounded-lg bg-black/20 px-3 py-2.5"><Gem className="h-4 w-4 text-amber-300" /><span className="text-sm text-zinc-200">{formatNumber(reward.reward_stones, 0)} <span className="text-zinc-400">Nguyên Thạch</span></span></div>
            <div className="flex items-center gap-2.5 rounded-lg bg-black/20 px-3 py-2.5"><Gem className="h-4 w-4 text-sky-300" /><span className="text-sm text-zinc-200">{formatNumber(reward.reward_immortal_stones, 0)} <span className="text-zinc-400">Tiên Nguyên Thạch</span></span></div>
            {reward.reward_gu_id && <div className="flex items-center gap-2.5 rounded-lg bg-black/20 px-3 py-2.5"><span className="text-purple-300">✦</span><span className="text-sm text-purple-200">Cổ Trùng thưởng: {reward.reward_gu_id.replace("GU_", "").replace(/_/g, " ")}{reward.reward_gu_rank && <span className="ml-1 text-xs text-zinc-400">(Rank {reward.reward_gu_rank})</span>}</span></div>}
            {reward.title && <div className="flex items-center gap-2.5 rounded-lg bg-black/20 px-3 py-2.5"><Crown className="h-4 w-4 text-amber-300" /><span className="text-sm text-zinc-200">{reward.title}</span></div>}
          </div>
        </GameCard>
      ))}
    </div>
  );
}
