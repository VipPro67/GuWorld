"use client";

import { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { GameCard, GameProgressBar } from "@/components/ui";

export default function DeathArenaTeamView() {
  const { deathArenaMyTeam, deathArenaMyTeamMembers, deathArenaTournament, fetchDeathArenaMyTeam } = useGameStore();

  useEffect(() => {
    fetchDeathArenaMyTeam();
  }, [fetchDeathArenaMyTeam]);

  if (!deathArenaMyTeam) {
    return (
      <div className="text-center text-gray-400">
        Bạn chưa đăng ký giải đấu nào
      </div>
    );
  }

  const formatHP = (current: number, max: number) => {
    const percent = max > 0 ? (current / max) * 100 : 0;
    return { percent, display: `${Math.round(current)}/${Math.round(max)}` };
  };

  return (
    <div>
      <div className="mb-4">
        <h3 className="text-lg font-bold">{deathArenaMyTeam.team_name}</h3>
        <div className="text-sm text-gray-400">
          Hạt giống: #{deathArenaMyTeam.seed_number} | 
          {deathArenaMyTeam.eliminated ? (
            <span className="text-red-400"> Đã bị loại ở vòng {deathArenaMyTeam.eliminated_at_round}</span>
          ) : (
            <span className="text-green-400"> Đang thi đấu</span>
          )}
        </div>
        {deathArenaMyTeam.final_placement && (
          <div className="text-sm text-yellow-400">
            Thứ hạng: {deathArenaMyTeam.final_placement === 1 ? "🥇 Quán Quân" : 
                       deathArenaMyTeam.final_placement === 2 ? "🥈 Á Quân" : 
                       deathArenaMyTeam.final_placement === 3 ? "🥉 Qúa Quân" : 
                       `#${deathArenaMyTeam.final_placement}`}
          </div>
        )}
      </div>

      <div className="space-y-2">
        {deathArenaMyTeamMembers.map((member) => {
          const hp = formatHP(member.current_hp, member.max_hp);
          return (
            <GameCard key={member.id} className="p-3">
              <div className="flex justify-between items-start mb-2">
                <div>
                  <span className="font-medium">{member.username}</span>
                  <span className="text-xs text-gray-400 ml-2">Rank {member.rank}</span>
                </div>
                {member.eliminated && (
                  <span className="px-2 py-1 text-xs bg-red-600 rounded">Đã hy sinh</span>
                )}
              </div>
              
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400 w-8">HP:</span>
                  <div className="flex-1">
                    <GameProgressBar
                      value={hp.percent}
                      max={100}
                      variant={hp.percent > 50 ? "success" : hp.percent > 25 ? "warning" : "danger"}
                    />
                  </div>
                  <span className="text-xs text-gray-300 w-20 text-right">{hp.display}</span>
                </div>
                
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400 w-8">MP:</span>
                  <div className="flex-1">
                    <GameProgressBar
                      value={member.max_mana > 0 ? (member.current_mana / member.max_mana) * 100 : 0}
                      max={100}
                      variant="mana"
                    />
                  </div>
                  <span className="text-xs text-gray-300 w-20 text-right">
                    {Math.round(member.current_mana)}/{Math.round(member.max_mana)}
                  </span>
                </div>
              </div>
            </GameCard>
          );
        })}
      </div>

      {deathArenaTournament?.status === "IN_PROGRESS" && (
        <div className="mt-4 text-center">
          <p className="text-sm text-gray-400">
            HP sẽ được giữ nguyên sau mỗi vòng nếu thắng
          </p>
        </div>
      )}
    </div>
  );
}