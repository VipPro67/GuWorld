"use client";
import React, { useEffect, useState } from "react";
import {
  Award,
  Crown,
  Check,
  Sparkles,
  Image as ImageIcon,
  Flame,
  Zap,
  Atom,
  ShieldAlert,
  LayoutGrid,
  List,
  X,
  Gift,
} from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import { checkQuest, formatRewards, isClaimableAchievement } from "@/data/quests";

interface AchievementItem {
  id: string;
  name: string;
  description: string;
  icon: string;
  has_frame?: boolean;
  reward_type?: string;
  reward_val?: number;
  reward_label?: string;
  title_effect?: { pve_monster_type: string; pve_damage_pct: number; utility_type?: string; utility_pct?: number };
}

interface QuestItem {
  id: string;
  title: string;
  desc: string;
  req_desc: string;
  rewards: Array<{ type: string; id?: string; amount: number; label: string }>;
}

interface DailyMissionItem {
  id: string;
  title: string;
  description: string;
  target: number;
  progress: number;
  reward_type: string;
  reward: number;
  completed: boolean;
  claimable: boolean;
  claimed: boolean;
}

export default function AchievementsPanel() {
  const store = useGameStore();
  const { achievements, earnedAchievements, selectedTitle, selectedFrame } = store;
  const [subTab, setSubTab] = useState<"quests" | "achievements">("quests");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [inspectedAch, setInspectedAch] = useState<AchievementItem | null>(null);

  useEffect(() => {
    store.fetchAchievements();
    store.fetchQuests();
    store.fetchDailyMissions();
    store.fetchAperture();
    // eslint-disable-next-deps
  }, []);

  const isEarned = (id: string) => earnedAchievements.some((e: { achievement_id: string }) => e.achievement_id === id);

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
      {/* Sub-tab selection */}
      <div className="flex gap-2 w-full bg-zinc-950/60 p-1.5 rounded-2xl border border-zinc-900 mb-2">
        <button
          onClick={() => setSubTab("quests")}
          className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${
            subTab === "quests"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Nhiệm Vụ
        </button>
        <button
          onClick={() => setSubTab("achievements")}
          className={`flex-1 text-xs py-1.5 rounded-xl font-mono font-bold transition-all cursor-pointer ${
            subTab === "achievements"
              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Thành Tựu
        </button>
      </div>

      {subTab === "quests" && (
        <>
          <DailyMissionsPanel
            missions={store.dailyMissions as DailyMissionItem[]}
            resetAt={store.dailyMissionsResetAt}
            onClaim={async (id) => {
              const result = await store.claimDailyMission(id);
              if (result?.success) store.addToast(result.message, "success");
            }}
          />
          <MainQuestsPanel />
        </>
      )}

      {subTab === "achievements" && (
        <>
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
                      {achievements.find((a: AchievementItem) => a.id === selectedTitle)?.icon || "🏆"}
                    </span>
                    <span className="text-xs font-mono text-zinc-300">
                      {achievements.find((a: AchievementItem) => a.id === selectedTitle)?.name || selectedTitle}
                    </span>
                  </div>
                )}
                {selectedFrame && (
                  <div className="flex items-center gap-2 bg-zinc-950/80 px-3 py-1.5 rounded-xl border border-zinc-800">
                    <ImageIcon className="w-4 h-4 text-purple-400" />
                    <span className="text-xs font-mono text-zinc-300">
                      Frame: {achievements.find((a: AchievementItem) => a.id === selectedFrame)?.name || selectedFrame}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Achievement List */}
          <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4 text-left">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2.5">
              <div className="flex items-center gap-2">
                <Award className="w-5 h-5 text-amber-400" />
                <span className="font-serif font-bold text-zinc-100 text-sm">Danh Sách Thành Tựu</span>
                <span className="text-[10px] font-mono text-zinc-500 bg-zinc-950 px-2 py-0.5 rounded-full border border-zinc-850">
                  {earnedAchievements.length} / {achievements.length} Đã Đạt
                </span>
              </div>

              {/* View Toggle */}
              <div className="flex items-center gap-1 bg-zinc-950/80 p-0.5 rounded-xl border border-zinc-800">
                <button
                  type="button"
                  onClick={() => setViewMode("grid")}
                  className={`p-1.5 rounded-lg text-xs font-mono transition-all flex items-center gap-1 cursor-pointer ${
                    viewMode === "grid"
                      ? "bg-amber-500/20 text-amber-300 border border-amber-500/40 shadow-sm"
                      : "text-zinc-500 hover:text-zinc-300"
                  }`}
                  title="Dạng Huy Chương RPG"
                >
                  <LayoutGrid className="w-3.5 h-3.5" />
                  <span className="text-[10px] hidden sm:inline">Huy Chương</span>
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("list")}
                  className={`p-1.5 rounded-lg text-xs font-mono transition-all flex items-center gap-1 cursor-pointer ${
                    viewMode === "list"
                      ? "bg-amber-500/20 text-amber-300 border border-amber-500/40 shadow-sm"
                      : "text-zinc-500 hover:text-zinc-300"
                  }`}
                  title="Dạng Danh Sách"
                >
                  <List className="w-3.5 h-3.5" />
                  <span className="text-[10px] hidden sm:inline">Danh Sách</span>
                </button>
              </div>
            </div>

            {achievements.length === 0 ? (
              <p className="text-xs text-zinc-500 italic">Đang tải...</p>
            ) : viewMode === "grid" ? (
              /* RPG Trophy Grid */
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
                {achievements.map((ach: AchievementItem) => {
                  const earned = isEarned(ach.id);
                  const canClaim = !earned && isClaimableAchievement(ach.id, store.player, store.aperture);
                  const isSelectedTitle = selectedTitle === ach.id;
                  const isSelectedFrame = selectedFrame === ach.id;

                  return (
                    <div
                      key={ach.id}
                      onClick={() => setInspectedAch(ach)}
                      className={`relative rounded-2xl border p-3 flex flex-col justify-between items-center text-center transition-all duration-200 cursor-pointer select-none group hover:-translate-y-0.5 hover:shadow-xl ${
                        canClaim
                          ? "border-amber-400 bg-amber-950/30 shadow-[0_0_15px_rgba(245,158,11,0.25)] ring-1 ring-amber-400/50 animate-pulse"
                          : earned
                          ? "border-amber-500/40 bg-zinc-900/80 shadow-[0_0_10px_rgba(245,158,11,0.1)] hover:border-amber-400"
                          : "border-zinc-800/60 bg-zinc-950/60 opacity-60 hover:opacity-90"
                      }`}
                    >
                      {/* Top status indicator */}
                      <div className="w-full flex items-center justify-between mb-1">
                        {isSelectedTitle || isSelectedFrame ? (
                          <span className="text-[9px] font-mono font-bold px-1.5 py-0.2 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">
                            Đang TB
                          </span>
                        ) : earned ? (
                          <span className="text-[9px] font-mono text-emerald-400 flex items-center gap-0.5">
                            <Check className="w-3 h-3 text-emerald-400" /> Đạt
                          </span>
                        ) : canClaim ? (
                          <span className="text-[9px] font-mono text-amber-300 font-bold flex items-center gap-0.5">
                            <Sparkles className="w-3 h-3 text-amber-400 animate-spin" /> Nhận
                          </span>
                        ) : (
                          <span className="text-[9px] font-mono text-zinc-600">Khóa</span>
                        )}

                        {ach.has_frame && (
                          <span className="text-[9px] font-mono text-purple-400" title="Có khung ảnh">
                            🖼️
                          </span>
                        )}
                      </div>

                      {/* Big Trophy Icon */}
                      <div className="py-2 text-3xl sm:text-4xl group-hover:scale-110 transition-transform flex items-center justify-center">
                        {ach.icon}
                      </div>

                      {/* Name */}
                      <div className="w-full mt-1">
                        <div className="text-xs font-serif font-bold text-zinc-100 truncate group-hover:text-amber-300 transition-colors">
                          {ach.name}
                        </div>
                        <div className="text-[10px] text-amber-300/80 font-mono truncate mt-0.5">
                          {ach.reward_label}
                        </div>
                      </div>

                      {/* Bottom action/status tag */}
                      <div className="w-full mt-2 pt-1.5 border-t border-zinc-800/60">
                        {canClaim ? (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleClaim(ach.id);
                            }}
                            disabled={store.actionLoading}
                            className="w-full py-1 rounded-lg text-[10px] font-mono font-bold bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 shadow-sm active:scale-95 cursor-pointer"
                          >
                            Nhận Quà
                          </button>
                        ) : (
                          <span className="text-[9px] font-mono text-zinc-500 block truncate">
                            {earned ? "Xem chi tiết ›" : "Chưa hoàn thành"}
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              /* Classic List Mode */
              <div className="grid grid-cols-1 gap-3 lg:max-h-[600px] overflow-y-visible lg:overflow-y-auto pr-1">
                {achievements.map((ach: AchievementItem) => {
                  const earned = isEarned(ach.id);
                  const canClaim = !earned && isClaimableAchievement(ach.id, store.player, store.aperture);
                  const isSelectedTitle = selectedTitle === ach.id;
                  const isSelectedFrame = selectedFrame === ach.id;
                  return (
                    <div
                      key={ach.id}
                      onClick={() => setInspectedAch(ach)}
                      className={`flex items-center justify-between p-3 rounded-xl border cursor-pointer hover:border-amber-500/40 transition-colors ${
                        earned
                          ? "bg-zinc-800/40 border-zinc-700"
                          : canClaim
                          ? "bg-amber-950/15 border-amber-500/40 shadow-[0_0_10px_rgba(245,158,11,0.08)]"
                          : "bg-zinc-950/40 border-zinc-900 opacity-70"
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
                          {ach.title_effect && (
                            <div className="text-[10px] text-cyan-300 mt-0.5">
                              +{ach.title_effect.pve_damage_pct}% sát thương PvE vs {ach.title_effect.pve_monster_type}
                              {ach.title_effect.utility_pct ? ` · +${ach.title_effect.utility_pct}% sản lượng Tiên Khiếu` : ''}
                            </div>
                          )}
                          <div className="text-[10px] text-amber-300 font-mono mt-0.5">
                            {ach.reward_label}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0" onClick={(e) => e.stopPropagation()}>
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
                                : "bg-zinc-900/60 text-zinc-650 border border-zinc-900 cursor-not-allowed"
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
                              : "bg-zinc-900/60 text-zinc-650 border border-zinc-900 cursor-not-allowed"
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
                            disabled={!canClaim || store.actionLoading}
                            onClick={() => handleClaim(ach.id)}
                            className={`text-[10px] px-3 py-1.5 rounded-lg font-mono font-bold transition-all shrink-0 ${
                              canClaim
                                ? "bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 shadow-[0_0_10px_rgba(245,158,11,0.3)] active:scale-95 cursor-pointer"
                                : "bg-zinc-900 text-zinc-650 border border-zinc-850 cursor-not-allowed"
                            }`}
                          >
                            {canClaim ? "Nhận" : "Chưa đạt"}
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Achievement Detail Modal */}
          {inspectedAch && (() => {
            const earned = isEarned(inspectedAch.id);
            const canClaim = !earned && isClaimableAchievement(inspectedAch.id, store.player, store.aperture);
            const isSelectedTitle = selectedTitle === inspectedAch.id;
            const isSelectedFrame = selectedFrame === inspectedAch.id;

            return (
              <div
                className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/80 backdrop-blur-sm animate-fade-in"
                onClick={() => setInspectedAch(null)}
              >
                <div
                  className="relative w-full max-w-md bg-zinc-950 border border-amber-500/40 rounded-3xl p-5 sm:p-6 shadow-[0_0_40px_rgba(245,158,11,0.25)] text-left space-y-4"
                  onClick={(e) => e.stopPropagation()}
                >
                  {/* Close button */}
                  <button
                    type="button"
                    onClick={() => setInspectedAch(null)}
                    className="absolute top-4 right-4 text-zinc-500 hover:text-zinc-200 p-1.5 rounded-xl hover:bg-zinc-900 cursor-pointer"
                    title="Đóng"
                  >
                    <X className="w-5 h-5" />
                  </button>

                  {/* Header visual */}
                  <div className="flex flex-col items-center text-center pt-2">
                    <div className="w-24 h-24 rounded-3xl bg-gradient-to-br from-amber-500/20 via-zinc-900 to-zinc-950 border-2 border-amber-500/40 flex items-center justify-center text-5xl shadow-inner relative">
                      {inspectedAch.icon}
                      {earned && (
                        <div className="absolute -top-1 -right-1 w-6 h-6 rounded-full bg-emerald-500 text-zinc-950 flex items-center justify-center text-xs font-bold shadow-md">
                          ✓
                        </div>
                      )}
                    </div>

                    <h3 className="text-lg font-serif font-bold text-zinc-100 mt-3">
                      {inspectedAch.name}
                    </h3>

                    {/* Status Badge */}
                    <div className="mt-1">
                      {canClaim ? (
                        <span className="text-[10px] font-mono font-bold px-2.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/40 animate-pulse">
                          ✨ Đủ Điều Kiện Nhận Thưởng
                        </span>
                      ) : earned ? (
                        <span className="text-[10px] font-mono font-bold px-2.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
                          ✓ Đã Mở Khóa Thành Tựu
                        </span>
                      ) : (
                        <span className="text-[10px] font-mono px-2.5 py-0.5 rounded-full bg-zinc-900 text-zinc-500 border border-zinc-800">
                          🔒 Chưa Hoàn Thành
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Description Box */}
                  <div className="bg-zinc-900/60 border border-zinc-850 rounded-2xl p-3.5 space-y-2">
                    <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">
                      Điển Tích / Mô Tả
                    </div>
                    <p className="text-xs text-zinc-300 font-serif leading-relaxed">
                      {inspectedAch.description}
                    </p>
                  </div>

                  {/* Title Effect */}
                  {inspectedAch.title_effect && (
                    <div className="bg-cyan-950/20 border border-cyan-500/30 rounded-2xl p-3.5 space-y-1">
                      <div className="text-[10px] text-cyan-400 font-mono uppercase tracking-wider font-bold">
                        Hiệu Ứng Danh Hiệu Kích Hoạt
                      </div>
                      <div className="text-xs text-cyan-200 font-mono">
                        +{inspectedAch.title_effect.pve_damage_pct}% sát thương PvE vs {inspectedAch.title_effect.pve_monster_type}
                      </div>
                      {inspectedAch.title_effect.utility_pct ? (
                        <div className="text-xs text-cyan-200 font-mono">
                          +{inspectedAch.title_effect.utility_pct}% sản lượng Tiên Khiếu
                        </div>
                      ) : null}
                    </div>
                  )}

                  {/* Reward Box */}
                  <div className="bg-amber-950/20 border border-amber-500/30 rounded-2xl p-3 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Award className="w-4 h-4 text-amber-400" />
                      <span className="text-xs font-mono font-bold text-amber-300">
                        {inspectedAch.reward_label}
                      </span>
                    </div>
                    {inspectedAch.has_frame && (
                      <span className="text-[10px] font-mono text-purple-400 bg-purple-950/40 px-2 py-0.5 rounded-lg border border-purple-800/40">
                        Tặng Kèm Khung
                      </span>
                    )}
                  </div>

                  {/* Modal Action Buttons */}
                  <div className="space-y-2 pt-1">
                    {canClaim && (
                      <button
                        type="button"
                        onClick={async () => {
                          await handleClaim(inspectedAch.id);
                          setInspectedAch(null);
                        }}
                        disabled={store.actionLoading}
                        className="w-full py-2.5 rounded-xl bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 font-mono font-bold text-xs shadow-lg active:scale-98 transition-all cursor-pointer"
                      >
                        Nhận Phần Thưởng Ngay
                      </button>
                    )}

                    {earned && (
                      <div className="grid grid-cols-2 gap-2">
                        <button
                          type="button"
                          onClick={async () => {
                            const newTitle = isSelectedTitle ? "" : inspectedAch.id;
                            const r = await store.selectTitleFrame(newTitle, selectedFrame);
                            if (r?.success) store.addToast(r.message, "success");
                          }}
                          className={`py-2 px-3 rounded-xl text-xs font-mono font-bold transition-all cursor-pointer border ${
                            isSelectedTitle
                              ? "bg-amber-500/20 text-amber-300 border-amber-500/40"
                              : "bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border-zinc-800"
                          }`}
                        >
                          {isSelectedTitle ? "Gỡ Danh Hiệu" : "Trang Bị Danh Hiệu"}
                        </button>

                        {inspectedAch.has_frame ? (
                          <button
                            type="button"
                            onClick={async () => {
                              const newFrame = isSelectedFrame ? "" : inspectedAch.id;
                              const r = await store.selectTitleFrame(selectedTitle, newFrame);
                              if (r?.success) store.addToast(r.message, "success");
                            }}
                            className={`py-2 px-3 rounded-xl text-xs font-mono font-bold transition-all cursor-pointer border ${
                              isSelectedFrame
                                ? "bg-purple-500/20 text-purple-300 border-purple-500/40"
                                : "bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border-zinc-800"
                            }`}
                          >
                            {isSelectedFrame ? "Gỡ Khung" : "Trang Bị Khung"}
                          </button>
                        ) : (
                          <div className="text-[10px] text-zinc-600 font-mono flex items-center justify-center">
                            Không có khung
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })()}
        </>
      )}
    </div>
  );
}

function DailyMissionsPanel({ missions, resetAt, onClaim }: {
  missions: DailyMissionItem[];
  resetAt: string;
  onClaim: (id: string) => Promise<void>;
}) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const secondsLeft = Math.max(0, Math.floor((Date.parse(resetAt) - now) / 1000));
  const countdown = `${Math.floor(secondsLeft / 3600).toString().padStart(2, "0")}:${Math.floor((secondsLeft % 3600) / 60).toString().padStart(2, "0")}:${(secondsLeft % 60).toString().padStart(2, "0")}`;

  return (
    <section className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4 sm:p-5 shadow-inner space-y-3">
      <div className="flex items-center justify-between gap-3 border-b border-zinc-900 pb-2.5">
        <div className="flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-amber-400" />
          <h3 className="font-serif font-bold text-zinc-100 text-sm">Nhiệm Vụ Hằng Ngày</h3>
        </div>
        <span className="text-[10px] font-mono text-zinc-400">Làm mới sau {countdown}</span>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
        {missions.map((mission) => {
          const pct = Math.min(100, Math.floor((mission.progress / Math.max(1, mission.target)) * 100));
          const rewardLabel = mission.reward_type === "immortal_stones" ? "Tiên Nguyên Thạch" : "Nguyên Thạch";
          return (
            <div key={mission.id} className="rounded-xl border border-zinc-800 bg-zinc-950/60 p-3 space-y-2">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="text-xs font-bold text-zinc-100">{mission.title}</div>
                  <div className="text-[10px] text-zinc-400 mt-0.5">{mission.description}</div>
                </div>
                <span className="text-[10px] text-amber-300 whitespace-nowrap">+{mission.reward} {rewardLabel}</span>
              </div>
              <div className="h-1.5 rounded-full bg-zinc-800 overflow-hidden">
                <div className="h-full bg-amber-400 transition-all" style={{ width: `${pct}%` }} />
              </div>
              <div className="flex items-center justify-between text-[10px]">
                <span className="text-zinc-500">{Math.min(mission.progress, mission.target)} / {mission.target}</span>
                {mission.claimed ? (
                  <span className="text-emerald-400">Đã nhận</span>
                ) : mission.claimable ? (
                  <button onClick={() => void onClaim(mission.id)} className="px-2.5 py-1 rounded-lg bg-amber-500/15 text-amber-300 border border-amber-500/30 hover:bg-amber-500/25">Nhận</button>
                ) : mission.completed ? (
                  <span className="text-emerald-400">Hoàn thành</span>
                ) : (
                  <span className="text-zinc-500">Đang thực hiện</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
      {missions.length === 0 && <p className="text-xs text-zinc-500">Đang tải nhiệm vụ…</p>}
    </section>
  );
}

function MainQuestsPanel() {
  const store = useGameStore();
  const player = store.player;
  if (!player) return null;

  const completed = player.completed_quests || [];
  const quests = store.quests || [];

  const getQuestIcon = (id: string) => {
    switch (id) {
      case "quest_1_awaken":
        return <Sparkles className="w-3.5 h-3.5 text-emerald-400" />;
      case "quest_2_vital_gu":
        return <Flame className="w-3.5 h-3.5 text-amber-500 animate-pulse" />;
      case "quest_3_rank_2":
        return <Zap className="w-3.5 h-3.5 text-cyan-400" />;
      case "quest_4_refine_gu":
        return <Atom className="w-3.5 h-3.5 text-purple-400" />;
      case "quest_5_fame":
        return <Crown className="w-3.5 h-3.5 text-yellow-500 animate-bounce" />;
      case "quest_6_rank_5":
        return <ShieldAlert className="w-3.5 h-3.5 text-rose-500" />;
      default:
        return <Award className="w-3.5 h-3.5 text-zinc-400" />;
    }
  };

  const buttonLabel = (q: QuestItem, isDone: boolean, canClaim: boolean) => {
    if (isDone) return "ĐÃ NHẬN";
    if (!canClaim) return "CHƯA ĐẠT";
    const rewardStr = formatRewards(q.rewards);
    return `NHẬN${rewardStr ? " " + rewardStr : ""}`;
  };

  return (
    <div className="w-full text-center min-h-full bg-zinc-900/40 border border-zinc-900 rounded-2xl p-5 shadow-inner flex flex-col gap-4 text-left">
      <div className="text-[10px] text-zinc-500 uppercase tracking-widest text-center mb-3 font-mono font-bold flex items-center justify-center gap-1.5 border-b border-zinc-900 pb-2.5">
        <Award className="w-4 h-4 text-amber-500 animate-pulse" />
        Nhiệm Vụ Chính Tuyến
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 lg:max-h-[600px] overflow-y-visible lg:overflow-y-auto pr-1">
        {quests.length === 0 ? (
          <div className="col-span-full text-zinc-500 text-xs italic py-6 bg-zinc-950/20 rounded-2xl border border-zinc-900/30 text-center">
            Chưa có nhiệm vụ nào.
          </div>
        ) : (
          quests
            .map((q: QuestItem) => {
              const isDone = completed.includes(q.id);
              const canClaim = !isDone && checkQuest(q.id, player, store.guDictionary || {});

              return (
                <div
                  key={q.id}
                  className={`p-3.5 rounded-2xl border text-xs flex flex-col justify-between gap-3 transition-all duration-300 hover:scale-[1.01] ${
                    isDone
                      ? "bg-emerald-950/10 border-emerald-950/20 opacity-70"
                      : canClaim
                      ? "bg-amber-950/20 border-amber-500/40 shadow-[0_0_12px_rgba(245,158,11,0.1)] ring-1 ring-amber-400/30"
                      : "bg-zinc-950/40 border-zinc-900"
                  }`}
                >
                  <div className="text-left flex-1 min-w-0">
                    <div
                      className={`font-semibold flex items-center gap-2 ${
                        isDone ? "text-emerald-600 dark:text-emerald-400 line-through" : "text-zinc-100"
                      }`}
                    >
                      <div className="p-1.5 rounded-lg bg-zinc-900/80 border border-zinc-800">
                        {getQuestIcon(q.id)}
                      </div>
                      <span className="truncate text-sm font-serif">{q.title}</span>
                    </div>
                    <div className="text-[11px] text-zinc-400 mt-1.5 leading-snug">{q.desc}</div>
                    <div className="text-[10px] text-amber-400/80 mt-1 font-mono flex items-center gap-1">
                      <span>Mục tiêu:</span>
                      <span className="text-zinc-300">{q.req_desc}</span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between border-t border-zinc-900 pt-2.5">
                    <div className="text-[10px] font-mono text-zinc-500 truncate mr-2">
                      {q.rewards && q.rewards.length > 0 && `Thưởng: ${formatRewards(q.rewards)}`}
                    </div>
                    <button
                      disabled={isDone || !canClaim || store.actionLoading}
                      onClick={async () => {
                        await store.completeQuest(q.id);
                      }}
                      className={`px-4 py-1.5 rounded-xl text-xs font-bold font-mono transition-all shrink-0 cursor-pointer whitespace-nowrap ${
                        isDone
                          ? "bg-zinc-900/60 text-zinc-650 border border-zinc-950 cursor-not-allowed"
                          : canClaim
                          ? "bg-gradient-to-r from-amber-500 to-yellow-500 hover:from-amber-400 hover:to-yellow-400 text-zinc-950 shadow-[0_0_10px_rgba(245,158,11,0.3)] active:scale-95"
                          : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                      }`}
                    >
                      {isDone ? "ĐÃ XONG" : canClaim ? "NHẬN THƯỞNG" : "CHƯA ĐẠT"}
                    </button>
                  </div>
                </div>
              );
            })
        )}
      </div>
    </div>
  );
}
