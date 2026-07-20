"use client";
import React, { useEffect, useState } from "react";
import { Award, Crown, Check, Sparkles, Image as ImageIcon } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";

export default function AchievementsPanel() {
  const store = useGameStore();
  const { achievements, earnedAchievements, selectedTitle, selectedFrame } = store;

  useEffect(() => {
    store.fetchAchievements();
  }, []);

  const isEarned = (id: string) => earnedAchievements.some((e: any) => e.achievement_id === id);

  const handleClaim = async (id: string) => {
    const result = await store.claimAchievement(id);
    if (result?.success) {
      store.addToast(result.message, "success");
    } else {
      store.addToast(result?.message || "Lỗi", "error");
    }
  };

  return (
    <div className="w-full z-10 space-y-5">
      {/* Selected Title & Frame */}
      {(selectedTitle || selectedFrame) && (
        <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4 text-left">
          <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
            <Award className="w-5 h-5 text-amber-400" />
            <span className="font-serif font-bold text-zinc-100 text-sm">Trang Bị</span>
          </div>
          <div className="flex items-center gap-4">
            {selectedTitle && (
              <div className="flex items-center gap-2 bg-zinc-950/80 px-3 py-1.5 rounded-xl border border-zinc-800">
                <span className="text-lg">
                  {achievements.find((a: any) => a.id === selectedTitle)?.icon || "🏆"}
                </span>
                <span className="text-xs font-mono text-zinc-300">
                  {achievements.find((a: any) => a.id === selectedTitle)?.name || selectedTitle}
                </span>
              </div>
            )}
            {selectedFrame && (
              <div className="flex items-center gap-2 bg-zinc-950/80 px-3 py-1.5 rounded-xl border border-zinc-800">
                <ImageIcon className="w-4 h-4 text-purple-400" />
                <span className="text-xs font-mono text-zinc-300">
                  Frame: {achievements.find((a: any) => a.id === selectedFrame)?.name || selectedFrame}
                </span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Achievement List */}
      <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4 text-left">
        <div className="flex items-center gap-2 border-b border-zinc-900 pb-2.5">
          <Award className="w-5 h-5 text-amber-400" />
          <span className="font-serif font-bold text-zinc-100 text-sm">Danh Sách Thành Tựu</span>
        </div>
        {achievements.length === 0 ? (
          <p className="text-xs text-zinc-500 italic">Đang tải...</p>
        ) : (
          <div className="grid grid-cols-1 gap-3 max-h-[600px] overflow-y-auto pr-1">
            {achievements.map((ach: any) => {
              const earned = isEarned(ach.id);
              const isSelectedTitle = selectedTitle === ach.id;
              const isSelectedFrame = selectedFrame === ach.id;
              return (
                <div
                  key={ach.id}
                  className={`flex items-center justify-between p-3 rounded-xl border ${
                    earned
                      ? "bg-zinc-800/40 border-zinc-700"
                      : "bg-zinc-950/40 border-zinc-900"
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="text-2xl shrink-0">{ach.icon}</span>
                    <div className="min-w-0">
                      <div className="text-sm font-mono font-bold text-zinc-100 truncate">
                        {ach.name}
                      </div>
                      <div className="text-[10px] text-zinc-400 font-serif leading-tight">
                        {ach.description}
                      </div>
                      <div className="text-[10px] text-amber-300 font-mono mt-0.5">
                        {ach.reward_label}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {ach.has_frame && (
                      <button
                        onClick={async () => {
                          const r = await store.selectTitleFrame(selectedTitle, earned ? ach.id : "");
                          if (r?.success) store.addToast(r.message, "success");
                        }}
                        disabled={!earned}
                        className={`text-[10px] px-2 py-1 rounded-lg font-mono font-bold transition-all cursor-pointer ${
                          isSelectedFrame
                            ? "bg-purple-500/20 text-purple-400 border border-purple-500/30"
                            : earned
                            ? "bg-zinc-800 text-zinc-300 hover:bg-zinc-700 border border-zinc-700"
                            : "bg-zinc-900/60 text-zinc-600 border border-zinc-900 cursor-not-allowed"
                        }`}
                        title={earned ? "Đổi frame" : "Chưa mở khóa"}
                      >
                        <ImageIcon className="w-3 h-3 inline mr-0.5" />
                        Frame
                      </button>
                    )}
                    <button
                      onClick={async () => {
                        const r = await store.selectTitleFrame(earned ? ach.id : "", selectedFrame);
                        if (r?.success) store.addToast(r.message, "success");
                      }}
                      disabled={!earned}
                      className={`text-[10px] px-2 py-1 rounded-lg font-mono font-bold transition-all cursor-pointer ${
                        isSelectedTitle
                          ? "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                          : earned
                          ? "bg-zinc-800 text-zinc-300 hover:bg-zinc-700 border border-zinc-700"
                          : "bg-zinc-900/60 text-zinc-600 border border-zinc-900 cursor-not-allowed"
                      }`}
                      title={earned ? "Trang bị danh hiệu" : "Chưa mở khóa"}
                    >
                      <Award className="w-3 h-3 inline mr-0.5" />
                      {isSelectedTitle ? "Đã TB" : "TB"}
                    </button>
                    {earned ? (
                      <Check className="w-5 h-5 text-green-400 shrink-0" />
                    ) : (
                      <button
                        onClick={() => handleClaim(ach.id)}
                        className="text-[10px] px-3 py-1.5 rounded-lg font-mono font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20 hover:bg-amber-500/20 transition-all cursor-pointer shrink-0"
                      >
                        Nhận
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
