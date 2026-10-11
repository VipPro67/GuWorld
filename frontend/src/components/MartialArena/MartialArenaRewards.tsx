"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard, GameButton } from "@/components/ui";

export default function MartialArenaRewards() {
  const { martialRewards, fetchMartialRewards, claimMartialReward, martialLoading } = useGameStore();

  useEffect(() => {
    fetchMartialRewards();
  }, [fetchMartialRewards]);

  if (!martialRewards || martialRewards.length === 0) {
    return <div className="text-center text-gray-400">Chưa có phần thưởng nào</div>;
  }

  const placementLabel = (placement: number) => {
    switch (placement) {
      case 1: return "🥇 Quán Quân";
      case 2: return "🥈 Á Quân";
      case 3: return "🥉 Quý Quân";
      default: return `#${placement}`;
    }
  };

  const placementColor = (placement: number) => {
    switch (placement) {
      case 1: return "border-yellow-500 bg-yellow-900/30";
      case 2: return "border-gray-400 bg-gray-700/30";
      case 3: return "border-amber-600 bg-amber-900/30";
      default: return "border-gray-600 bg-black/30";
    }
  };

  return (
    <div className="space-y-3">
      {martialRewards.map((reward) => (
        <GameCard key={reward.id} className={`p-4 ${placementColor(reward.placement)}`}>
          <div className="flex justify-between items-start mb-2">
            <div>
              <div className="font-bold text-lg">{placementLabel(reward.placement)}</div>
              <div className="text-sm text-gray-400">{reward.title}</div>
            </div>
            {reward.claimed ? (
              <span className="px-3 py-1 text-sm bg-green-600 rounded">Đã nhận</span>
            ) : (
              <GameButton
                onClick={() => claimMartialReward(reward.id)}
                disabled={martialLoading}
                variant="success"
                size="sm"
              >
                {martialLoading ? "..." : "Nhận"}
              </GameButton>
            )}
          </div>

          <div className="mt-3 space-y-1">
            <div className="flex items-center gap-2">
              <span className="text-yellow-400">💎</span>
              <span className="text-sm">{reward.reward_stones.toLocaleString()} Nguyên Thạch</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-blue-400">💎</span>
              <span className="text-sm">{reward.reward_immortal_stones.toLocaleString()} Tiên Nguyên Thạch</span>
            </div>
            {reward.reward_gu_id && (
              <div className="flex items-center gap-2">
                <span className="text-purple-400">🐛</span>
                <span className="text-sm text-purple-300">
                  {reward.reward_gu_id.replace("GU_", "").replace(/_/g, " ")}
                  {reward.reward_gu_rank && <span className="text-xs text-gray-400 ml-1">(Rank {reward.reward_gu_rank})</span>}
                </span>
              </div>
            )}
            {reward.title && (
              <div className="flex items-center gap-2">
                <span className="text-yellow-400">👑</span>
                <span className="text-sm">{reward.title}</span>
              </div>
            )}
          </div>
        </GameCard>
      ))}
    </div>
  );
}
