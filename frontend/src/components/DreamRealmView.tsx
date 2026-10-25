"use client";
import React, { useEffect } from "react";
import { GamePanel, GameCard } from "@/components/ui";
import { useGameStore, InstanceListItem } from "@/store/useGameStore";
import {
  Compass,
  ArrowLeft,
  Flame,
  Award,
  Sparkles,
  Zap,
  Activity,
  Heart,
  Scroll,
  BookOpen,
  Hourglass,
  LogOut
} from "lucide-react";

const formatCooldown = (seconds: number): string => {
  if (seconds <= 0) return "";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return `Còn ${h}h ${m}m`;
  return `Còn ${m}m`;
};

export const DreamRealmView: React.FC = () => {
  const {
    player,
    activeDreamSession,
    activeDreamTemplate,
    availableDreamRealms,
    dreamDailyEntriesUsed,
    dreamDailyEntriesMax,
    fetchDreamRealms,
    fetchActiveDreamSession,
    enterDreamRealm,
    actionDreamRealm,
    exitDreamRealm,
    setView,
    dreamActionLoading
  } = useGameStore();

  useEffect(() => {
    fetchActiveDreamSession();
    fetchDreamRealms();
  }, [fetchActiveDreamSession, fetchDreamRealms]);

  if (!player) return null;

  const realms: InstanceListItem[] = availableDreamRealms;
  const isDailyLimitReached = (dreamDailyEntriesUsed ?? 0) >= (dreamDailyEntriesMax ?? 5);

  const renderReqs = (requirements: Record<string, ApiResponseData>) => {
    const items: string[] = [];
    if (requirements.min_soul_hp !== undefined) {
      items.push(`Yêu cầu Hồn Lực >= ${requirements.min_soul_hp}`);
    }
    if (requirements.material_id) {
      items.push(`Cần ${requirements.amount}x ${requirements.material_id === 'green_bamboo_wine' ? 'Rượu Xanh' : requirements.material_id}`);
    }
    if (requirements.gu_id) {
      items.push(`Cần Cổ trùng: ${requirements.gu_id === 'GU_DREAM' ? 'Cổ Giải Mộng' : requirements.gu_id}`);
    }
    return items.join(" • ");
  };

  const currentStageIndex = activeDreamSession?.current_stage ?? 0;
  const currentStage = activeDreamTemplate?.map_data
    ? (activeDreamTemplate.map_data as ApiResponseData[]).find((s: ApiResponseData) => s.stage_index === currentStageIndex)
    : null;

  return (
    <GamePanel variant="default" className="w-full mx-auto px-4 py-4 flex flex-col gap-6">
      {/* Header bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between bg-zinc-950/60 border border-zinc-900 rounded-2xl p-4 shadow-lg backdrop-blur-md gap-3">
        <div className="flex items-center gap-3">
          <BookOpen className="w-6 h-6 text-purple-400" />
          <div>
            <h1 className="text-lg font-serif font-bold text-zinc-100">Mộng Cảnh Tiềm Thức</h1>
            <p className="text-xs text-zinc-400 font-mono">Thám hiểm tiềm thức tiền nhân · Lĩnh ngộ Chân Ý lưu phái · Tiêu hao Hồn Lực</p>
          </div>
        </div>
        <div className="flex items-center gap-3 self-end sm:self-auto">
          <span className="px-3 py-1.5 rounded-xl text-xs font-mono font-semibold bg-purple-950/60 border border-purple-800/40 text-purple-200">
            Lượt hôm nay: {dreamDailyEntriesUsed ?? 0}/{dreamDailyEntriesMax ?? 5}
          </span>
          <button
            onClick={() => setView("mmo-world")}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl border border-zinc-800 bg-zinc-900/60 hover:bg-zinc-800 text-xs text-zinc-300 font-mono transition-all cursor-pointer"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            Trở Về
          </button>
        </div>
      </div>

      {/* ACTIVE DREAM REALM SESSION VIEW */}
      {activeDreamSession && activeDreamTemplate && currentStage ? (
        <div className="space-y-6">
          <div className="bg-purple-950/20 border border-purple-800/40 rounded-2xl p-5 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-purple-950/80 text-purple-300 border border-purple-800/60">
                  Mộng Cảnh Đang Diễn Ra
                </span>
                <h2 className="text-base font-bold font-serif text-purple-200">
                  {activeDreamTemplate.name}
                </h2>
              </div>
              <p className="text-xs text-zinc-400 font-mono">
                {activeDreamTemplate.description}
              </p>
            </div>

            <div className="flex items-center gap-4 shrink-0">
              <div className="flex items-center gap-2 bg-zinc-900/80 border border-purple-900/30 px-3 py-1.5 rounded-xl">
                <Activity className="w-4 h-4 text-purple-400" />
                <div className="flex flex-col">
                  <span className="text-[10px] font-mono text-zinc-400">Giai Đoạn</span>
                  <span className="text-xs font-mono font-bold text-purple-300">
                    Ải #{currentStageIndex + 1}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2 bg-zinc-900/80 border border-purple-900/30 px-3 py-1.5 rounded-xl">
                <Heart className="w-4 h-4 text-indigo-400" />
                <div className="flex flex-col">
                  <span className="text-[10px] font-mono text-zinc-400">Hồn Lực</span>
                  <span className="text-xs font-mono font-bold text-indigo-300">
                    {player.soul_hp} / {player.max_soul}
                  </span>
                </div>
              </div>

              <button
                onClick={() => exitDreamRealm()}
                disabled={dreamActionLoading}
                className="flex items-center gap-1.5 px-3 py-2 rounded-xl border border-red-900/40 bg-red-950/30 hover:bg-red-950/60 text-xs text-red-300 font-mono transition-all cursor-pointer"
                title="Rút lui khỏi mộng cảnh (Thất bại)"
              >
                <LogOut className="w-3.5 h-3.5" />
                Rút Lui
              </button>
            </div>
          </div>

          {/* DREAM REALM WARNING BANNER */}
          <div className="p-3.5 rounded-xl border border-purple-900/40 bg-purple-950/20 text-xs font-mono text-purple-200/90 leading-relaxed">
            💡 <strong className="font-semibold text-purple-300">Quy tắc Mộng Cảnh:</strong> Cổ trùng và tu vi của đạo hữu bị khóa lại trong tiềm thức. Mỗi bước thám hiểm hoặc thất bại sẽ trừ thẳng vào <strong className="text-indigo-300">Hồn Lực</strong>. Nếu Hồn Lực giảm về 0, đạo hữu sẽ rơi vào trạng thái ngủ say / hôn mê!
          </div>

          {/* STAGE STORY CONTENT */}
          <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-2xl p-6 relative overflow-hidden">
            <div className="flex items-center gap-2 mb-3 text-purple-400">
              <BookOpen className="w-4 h-4" />
              <span className="text-xs font-mono font-semibold uppercase tracking-wider">Diễn Biến Mộng Cảnh</span>
            </div>
            <p className="text-sm text-zinc-200 leading-relaxed whitespace-pre-line font-serif">
              {currentStage.description}
            </p>
          </div>

          {/* DECISION OPTIONS */}
          <div className="space-y-3">
            <span className="text-xs font-mono text-zinc-500 uppercase tracking-wider">Lựa Chọn Quyết Định:</span>
            {currentStage.options.map((opt: ApiResponseData, idx: number) => {
              const reqText = renderReqs(opt.requirements || {});
              return (
                <button
                  key={idx}
                  onClick={() => actionDreamRealm(idx)}
                  disabled={dreamActionLoading}
                  className="w-full text-left p-4 rounded-2xl border text-sm transition-all duration-300 cursor-pointer flex flex-col gap-1 bg-purple-950/10 hover:bg-purple-950/40 border-purple-900/30 hover:border-purple-600/50 text-purple-100 disabled:opacity-50"
                >
                  <span className="font-medium">{opt.text}</span>
                  {reqText && (
                    <span className="text-[11px] font-mono text-zinc-500 font-medium">{reqText}</span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      ) : (
        /* DREAM REALM TEMPLATES DIRECTORY */
        realms.length === 0 ? (
          <div className="text-center text-sm text-zinc-500 font-mono py-16">
            Đang tải danh sách mộng cảnh...
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {isDailyLimitReached && (
              <div className="bg-amber-950/40 border border-amber-800/40 rounded-2xl p-4 text-xs font-mono text-amber-200 flex items-center gap-2">
                ⚠️ Đạo hữu đã sử dụng hết lượt thám hiểm mộng cảnh hôm nay ({dreamDailyEntriesUsed}/{dreamDailyEntriesMax} lượt). Hãy quay lại vào ngày mai!
              </div>
            )}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {realms.map((realm) => {
                const isQualified = player.rank >= realm.required_rank;
                const inCooldown = (realm.cooldown_remaining ?? 0) > 0;
                const cdText = formatCooldown(realm.cooldown_remaining ?? 0);
                const hasSoulHP = (player.soul_hp ?? 0) > 0;

                return (
                  <GameCard
                    key={realm.id}
                    variant="danger"
                    className="border rounded-3xl p-6 flex flex-col justify-between gap-5 transition-all duration-350 hover:-translate-y-1 hover:border-zinc-800 hover:shadow-[0_0_30px_rgba(168,85,247,0.12)] border-purple-900/20 bg-zinc-950/70"
                  >
                    <div className="space-y-3">
                      <div className="flex justify-between items-start gap-2">
                        <h3 className="text-base font-serif font-bold text-zinc-100">{realm.name}</h3>
                        <span className="px-2 py-0.5 rounded text-[9px] font-mono font-semibold shrink-0 bg-purple-950/50 text-purple-300 border border-purple-800/40">
                          Mộng Cảnh
                        </span>
                      </div>

                      <p className="text-xs text-zinc-400 leading-relaxed line-clamp-3">
                        {realm.description}
                      </p>

                      {/* Badges / Requirements */}
                      <div className="flex flex-wrap gap-2 text-[10px] font-mono pt-1">
                        <span className="bg-zinc-900 text-zinc-400 border border-zinc-800 px-2 py-0.5 rounded-lg">
                          Yêu cầu: Bậc {realm.required_rank}+
                        </span>
                        <span className="bg-zinc-900 text-zinc-400 border border-zinc-800 px-2 py-0.5 rounded-lg">
                          Vật phẩm: {realm.entry_item === 'NONE' ? 'Tự do' : realm.entry_item}
                        </span>
                        {inCooldown && (
                          <span className="inline-flex items-center gap-1 bg-zinc-900 text-amber-300 border border-amber-800/40 px-2 py-0.5 rounded-lg">
                            <Hourglass className="w-3 h-3" />
                            {cdText}
                          </span>
                        )}
                      </div>

                      {/* Attainment and Rewards */}
                      <div className="bg-zinc-900/30 border border-zinc-900 rounded-xl p-3 space-y-1.5">
                        <span className="text-[10px] font-mono text-zinc-500 uppercase">Cơ Duyên Lĩnh Ngộ:</span>
                        <div className="flex flex-wrap gap-1.5">
                          {realm.rewards.attainment && (
                            <span className="inline-flex items-center gap-1 bg-purple-950/40 text-purple-300 border border-purple-900/50 px-2 py-0.5 rounded text-[10px]">
                              <Award className="w-3 h-3" />
                              Chân Ý {realm.rewards.attainment.path} (+{realm.rewards.attainment.amount})
                            </span>
                          )}
                          {realm.rewards.dao_marks && Object.entries(realm.rewards.dao_marks as Record<string, unknown>).map(([k, v]) => (
                            <span key={k} className="inline-flex items-center gap-1 bg-amber-950/30 text-amber-300 border border-amber-900/40 px-2 py-0.5 rounded text-[10px]">
                              <Flame className="w-3 h-3" />
                              {String(v)} Đạo Ngân {k}
                            </span>
                          ))}
                          {realm.rewards.materials && Object.entries(realm.rewards.materials as Record<string, unknown>).map(([k, v]) => (
                            <span key={k} className="inline-flex items-center gap-1 bg-indigo-950/30 text-indigo-300 border border-indigo-900/40 px-2 py-0.5 rounded text-[10px]">
                              <Sparkles className="w-3 h-3" />
                              {String(v)}x {k === 'guts_gu' ? 'Can Đảm Cổ' : k}
                            </span>
                          ))}
                          {realm.rewards.primeval_stones && (
                            <span className="inline-flex items-center gap-1 bg-yellow-950/30 text-yellow-300 border border-yellow-900/40 px-2 py-0.5 rounded text-[10px]">
                              <Zap className="w-3 h-3" />
                              {realm.rewards.primeval_stones} Nguyên Thạch
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {isQualified && !inCooldown && !isDailyLimitReached && hasSoulHP ? (
                      <button
                        onClick={() => enterDreamRealm(realm.id)}
                        disabled={dreamActionLoading}
                        className="w-full py-2.5 rounded-xl text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 shadow-md bg-purple-600 hover:bg-purple-500 border border-purple-500/20 text-white disabled:opacity-50"
                      >
                        <Compass className="w-4 h-4" />
                        Bắt Đầu Thám Hiểm Mộng Cảnh
                      </button>
                    ) : (
                      <div className="w-full text-center py-2.5 rounded-xl bg-zinc-900/40 border border-zinc-900 text-zinc-500 text-xs font-mono">
                        {inCooldown
                          ? `Hồi chiêu: ${cdText}`
                          : !hasSoulHP
                          ? "Hồn Lực cạn kiệt (Hôn mê)"
                          : isDailyLimitReached
                          ? "Hết lượt thám hiểm hôm nay"
                          : "Chưa đủ điều kiện"}
                      </div>
                    )}
                  </GameCard>
                );
              })}
            </div>
          </div>
        )
      )}
    </GamePanel>
  );
};
