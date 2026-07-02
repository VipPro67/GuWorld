"use client";
import React, { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
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
  BookOpen
} from "lucide-react";

export const DungeonView: React.FC = () => {
  const {
    player,
    activeInstanceSession,
    activeInstanceTemplate,
    fetchActiveInstance,
    enterInstance,
    actionInstance,
    setView,
    actionLoading
  } = useGameStore();

  useEffect(() => {
    fetchActiveInstance();
  }, [fetchActiveInstance]);

  if (!player) return null;

  // Static list of templates matching db seeds
  const dungeons = [
    {
      id: "dungeon_flower_wine",
      name: "Truyền Thừa Hoa Tửu Hành Giả",
      type: "INHERITANCE_DEMONIC",
      required_rank: 1,
      entry_item: "NONE",
      description: "Di tích động đá ẩn mật của Hoa Tửu Hành Giả lưu lại từ trăm năm trước, ẩn chứa cơ duyên Moonlight Gu và nguy cơ tử vong.",
      rewards: {
        materials: { guts_gu: 2, green_bamboo_wine: 1 },
        primeval_stones: 100,
        attainment: { path: "Moon Path", amount: 10 }
      }
    },
    {
      id: "dream_realm_star_constellation",
      name: "Mộng Cảnh Tinh Túc Tiên Tôn",
      type: "DREAM_REALM",
      required_rank: 5,
      entry_item: "NONE",
      description: "Thám hiểm thời niên thiếu đầy thử thách của Tinh Túc Tiên Tôn khi nghiên cứu tinh đồ giữa bầu trời tinh thần đại đạo.",
      rewards: {
        attainment: { path: "Dream Path", amount: 50 },
        materials: { guts_gu: 5 }
      }
    }
  ];

  // Helper to parse requirements for UI rendering
  const renderReqs = (requirements: Record<string, any>) => {
    const items: string[] = [];
    if (requirements.min_soul_hp !== undefined) {
      items.push(`Yêu cầu Soul HP >= ${requirements.min_soul_hp}`);
    }
    if (requirements.material_id) {
      items.push(`Cần ${requirements.amount}x ${requirements.material_id === 'green_bamboo_wine' ? 'Rượu Xanh' : requirements.material_id}`);
    }
    if (requirements.gu_id) {
      items.push(`Cần Cổ trùng: ${requirements.gu_id === 'GU_DREAM' ? 'Cổ Giải Mộng' : requirements.gu_id}`);
    }
    return items.join(" • ");
  };

  const currentStageIndex = activeInstanceSession?.current_stage ?? 0;
  const currentStage = activeInstanceTemplate?.map_data
    ? (activeInstanceTemplate.map_data as any[]).find((s: any) => s.stage_index === currentStageIndex)
    : null;

  const isDreamRealm = activeInstanceTemplate?.type === "DREAM_REALM";

  return (
    <div className="w-full max-w-4xl mx-auto px-4 py-4 flex flex-col gap-6">
      {/* Header bar */}
      <div className="flex items-center justify-between bg-zinc-950/60 border border-zinc-900 rounded-2xl p-4 shadow-lg backdrop-blur-md">
        <div className="flex items-center gap-3">
          <BookOpen className="w-6 h-6 text-purple-400" />
          <div>
            <h1 className="text-lg font-serif font-bold text-zinc-100">Bí Cảnh & Mộng Cảnh</h1>
            <p className="text-xs text-zinc-500 font-mono">Thám hiểm nhận Chân Ý tăng cảnh giới</p>
          </div>
        </div>
        {!activeInstanceSession && (
          <button
            onClick={() => setView("dashboard")}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl border border-zinc-850 hover:bg-zinc-900/60 text-zinc-400 hover:text-zinc-200 text-xs font-mono font-medium transition-all cursor-pointer"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            Trở Về
          </button>
        )}
      </div>

      {/* ADVENTURE IN PROGRESS VIEW */}
      {activeInstanceSession && activeInstanceTemplate && currentStage ? (
        <div className={`border rounded-3xl p-6 shadow-2xl backdrop-blur-md transition-all duration-500 relative overflow-hidden flex flex-col gap-6 ${
          isDreamRealm
            ? "bg-purple-950/20 border-purple-900/40 shadow-[0_0_50px_rgba(168,85,247,0.1)]"
            : "bg-red-950/20 border-red-900/40 shadow-[0_0_50px_rgba(239,68,68,0.1)]"
        }`}>
          {/* Decorative background aura */}
          <div className={`absolute top-0 right-0 w-80 h-80 rounded-full blur-3xl opacity-10 pointer-events-none -mr-20 -mt-20 ${
            isDreamRealm ? "bg-purple-500" : "bg-red-500"
          }`} />

          {/* Instance title and warning */}
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 border-b border-zinc-900 pb-4">
            <div>
              <span className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold ${
                isDreamRealm ? "bg-purple-900/50 text-purple-300 border border-purple-800/40" : "bg-red-900/50 text-red-300 border border-red-800/40"
              }`}>
                {isDreamRealm ? "Mộng Cảnh Sinh Tồn" : "Truyền Thừa Ma Đạo"}
              </span>
              <h2 className="text-xl font-serif font-bold text-zinc-100 mt-1">{activeInstanceTemplate.name}</h2>
            </div>
            <div className="flex items-center gap-4 text-xs font-mono">
              <div className="flex items-center gap-1.5">
                <Heart className="w-4 h-4 text-rose-500" />
                <span className="text-zinc-300">HP: {Math.round(player.hp)}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Activity className="w-4 h-4 text-indigo-400" />
                <span className="text-zinc-300">Soul HP: {player.soul_hp} / {player.max_soul}</span>
              </div>
            </div>
          </div>

          {/* Lore Warning for Dream Realms */}
          {isDreamRealm && (
            <div className="bg-purple-950/40 border border-purple-800/20 rounded-2xl p-4 text-xs text-purple-200 leading-relaxed">
              💡 <strong className="font-semibold text-purple-300">Cơ chế Mộng Cảnh:</strong> Cổ trùng và tu vi của đạo hữu bị khóa. Mỗi bước thám hiểm hoặc thất bại sẽ trừ thẳng vào <strong className="text-indigo-300">Soul HP</strong>. Nếu Soul HP giảm về 0, đạo hữu sẽ rơi vào trạng thái hôn mê sâu thảm hại!
            </div>
          )}

          {/* Scenario text box */}
          <div className="bg-zinc-950/70 border border-zinc-900/80 rounded-2xl p-5 sm:p-6 shadow-inner relative">
            <Scroll className="absolute top-4 right-4 w-5 h-5 text-zinc-800 pointer-events-none" />
            <p className="text-zinc-200 text-sm leading-loose font-serif indent-4">
              {currentStage.description}
            </p>
          </div>

          {/* Options grid */}
          <div className="flex flex-col gap-3">
            <span className="text-xs font-mono text-zinc-500 uppercase tracking-wider">Chọn Quyết Định:</span>
            {currentStage.options.map((opt: any, idx: number) => {
              const reqText = renderReqs(opt.requirements || {});
              return (
                <button
                  key={idx}
                  onClick={() => actionInstance(idx)}
                  disabled={actionLoading}
                  className={`w-full text-left p-4 rounded-2xl border text-sm transition-all duration-300 cursor-pointer flex flex-col gap-1 ${
                    isDreamRealm
                      ? "bg-purple-950/10 hover:bg-purple-950/40 border-purple-900/30 hover:border-purple-600/50 text-purple-100"
                      : "bg-red-950/10 hover:bg-red-950/40 border-red-900/30 hover:border-red-600/50 text-red-100"
                  }`}
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
        /* INSTANCE TEMPLATES LIST VIEW */
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {dungeons.map((dg) => {
            const isQualified = player.rank >= dg.required_rank;
            const isDream = dg.type === "DREAM_REALM";

            return (
              <div
                key={dg.id}
                className={`bg-zinc-950/40 border border-zinc-900 rounded-3xl p-6 flex flex-col justify-between gap-5 transition-all duration-350 hover:-translate-y-1 hover:border-zinc-800 ${
                  isDream
                    ? "hover:shadow-[0_0_30px_rgba(168,85,247,0.08)]"
                    : "hover:shadow-[0_0_30px_rgba(239,68,68,0.08)]"
                }`}
              >
                <div className="space-y-3">
                  <div className="flex justify-between items-start gap-2">
                    <h3 className="text-base font-serif font-bold text-zinc-100">{dg.name}</h3>
                    <span className={`px-2 py-0.5 rounded text-[9px] font-mono font-semibold shrink-0 ${
                      isDream ? "bg-purple-950/50 text-purple-300 border border-purple-800/40" : "bg-red-950/50 text-red-300 border border-red-800/40"
                    }`}>
                      {isDream ? "Mộng Cảnh" : "Truyền Thừa Ma Đạo"}
                    </span>
                  </div>

                  <p className="text-xs text-zinc-400 leading-relaxed line-clamp-3">
                    {dg.description}
                  </p>

                  {/* Badges / Requirements */}
                  <div className="flex flex-wrap gap-2 text-[10px] font-mono pt-1">
                    <span className="bg-zinc-900 text-zinc-400 border border-zinc-800 px-2 py-0.5 rounded-lg">
                      Yêu cầu: Rank {dg.required_rank}+
                    </span>
                    <span className="bg-zinc-900 text-zinc-400 border border-zinc-800 px-2 py-0.5 rounded-lg">
                      Vật phẩm: {dg.entry_item === 'NONE' ? 'Tự do' : dg.entry_item}
                    </span>
                  </div>

                  {/* Rewards summary */}
                  <div className="bg-zinc-900/30 border border-zinc-900 rounded-xl p-3 space-y-1.5">
                    <span className="text-[10px] font-mono text-zinc-500 uppercase">Phần thưởng Chinh Phục:</span>
                    <div className="flex flex-wrap gap-1.5">
                      {dg.rewards.attainment && (
                        <span className="inline-flex items-center gap-1 bg-amber-950/30 text-amber-300 border border-amber-900/40 px-2 py-0.5 rounded text-[10px]">
                          <Award className="w-3 h-3" />
                          Chân Ý {dg.rewards.attainment.path} (+{dg.rewards.attainment.amount})
                        </span>
                      )}
                      {dg.rewards.materials && Object.entries(dg.rewards.materials).map(([k, v]) => (
                        <span key={k} className="inline-flex items-center gap-1 bg-indigo-950/30 text-indigo-300 border border-indigo-900/40 px-2 py-0.5 rounded text-[10px]">
                          <Sparkles className="w-3 h-3" />
                          {v}x {k === 'guts_gu' ? 'Can Đảm Cổ' : k}
                        </span>
                      ))}
                      {dg.rewards.primeval_stones && (
                        <span className="inline-flex items-center gap-1 bg-yellow-950/30 text-yellow-300 border border-yellow-900/40 px-2 py-0.5 rounded text-[10px]">
                          <Zap className="w-3 h-3" />
                          {dg.rewards.primeval_stones} Nguyên Thạch
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {isQualified ? (
                  <button
                    onClick={() => enterInstance(dg.id)}
                    disabled={actionLoading}
                    className={`w-full py-2.5 rounded-xl text-xs font-mono font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5 shadow-md ${
                      isDream
                        ? "bg-purple-600 hover:bg-purple-500 border border-purple-500/20 text-white"
                        : "bg-red-600 hover:bg-red-500 border border-red-500/20 text-white"
                    }`}
                  >
                    <Compass className="w-4 h-4 animate-spin-slow" />
                    Bắt Đầu Thám Hiểm
                  </button>
                ) : (
                  <div className="w-full text-center py-2.5 rounded-xl bg-zinc-900/40 border border-zinc-900 text-zinc-650 text-xs font-mono">
                    Khóa
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
