"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameButton, GameCard, GamePanel } from "@/components/ui";
import { Swords, Trophy } from "lucide-react";

export function InfiniteTowerView() {
  const { towerProgress, towerLeaderboard, fetchTowerState, fetchTowerLeaderboard, challengeTower, actionLoading } = useGameStore();
  useEffect(() => { void fetchTowerState(); void fetchTowerLeaderboard(); }, [fetchTowerState, fetchTowerLeaderboard]);
  useEffect(() => {
    const timer = setInterval(() => { void fetchTowerState(); void fetchTowerLeaderboard(); }, 15_000);
    return () => clearInterval(timer);
  }, [fetchTowerState, fetchTowerLeaderboard]);
  const progress = (towerProgress?.progress || {}) as ApiResponseData;
  return (
    <GamePanel variant="gold" className="mx-auto w-full space-y-5 rounded-3xl p-4 sm:p-6">
      <header className="flex items-center gap-3 border-b border-zinc-800 pb-4">
        <Swords className="h-7 w-7 text-amber-400" />
        <div><h2 className="text-xl font-bold text-zinc-100">Tháp Chiến Thuật Vô Tận</h2>
          <p className="text-sm text-zinc-400">Đấu theo lượt, lưu tầng cao nhất. Thắng không tốn lượt; thua hoặc bỏ trận tính 1 lượt tuần.</p></div>
      </header>
      <GameCard className="flex flex-wrap items-center justify-between gap-4 p-4">
        <div className="space-y-1 text-sm text-zinc-300">
          <div>Tầng cao nhất: <strong className="text-amber-300">{progress.best_floor ?? 0}</strong></div>
          <div>Lượt thất bại tuần này: <strong>{progress.weekly_losses ?? 0}/{progress.loss_limit ?? 5}</strong></div>
          <div>Vào trận với HP và chân nguyên đầy; không tiêu hao vật phẩm.</div>
        </div>
        <GameButton onClick={() => void challengeTower()} disabled={actionLoading} variant="warning">
          <Swords className="mr-2 h-4 w-4" /> Khiêu chiến tầng {Number(towerProgress?.next_floor || 1)}
        </GameButton>
      </GameCard>
      <section className="space-y-2">
        <h3 className="flex items-center gap-2 font-semibold text-zinc-200"><Trophy className="h-4 w-4 text-amber-400" /> Bảng xếp hạng</h3>
        <div className="max-h-96 space-y-1 overflow-y-auto">
          {towerLeaderboard.map((row, index) => <div key={String(row.player_id)} className="flex justify-between rounded-lg bg-zinc-900/70 px-3 py-2 text-sm">
            <span className="text-zinc-300">#{index + 1} {String(row.username)}</span><strong className="text-amber-300">Tầng {String(row.best_floor)}</strong>
          </div>)}
          {towerLeaderboard.length === 0 && <p className="py-6 text-center text-sm text-zinc-500">Chưa có người ghi danh trên tháp.</p>}
        </div>
      </section>
    </GamePanel>
  );
}
