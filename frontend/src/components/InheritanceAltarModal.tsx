"use client";

import React from "react";
import { useGameStore } from "@/store/useGameStore";
import {
  Landmark,
  X,
  CheckCircle2,
  XCircle,
  Gem,
  Coins,
  BookOpen,
  Sparkles,
  PackageCheck,
  ShieldAlert,
} from "lucide-react";

export const InheritanceAltarModal: React.FC = () => {
  const selectedAltarInfo = useGameStore((state) => state.selectedAltarInfo);
  const altarModalOpen = useGameStore((state) => state.altarModalOpen);
  const setAltarModalOpen = useGameStore((state) => state.setAltarModalOpen);
  const claimAltar = useGameStore((state) => state.claimAltar);
  const altarLoading = useGameStore((state) => state.altarLoading);
  const player = useGameStore((state) => state.player);

  if (!altarModalOpen || !selectedAltarInfo) {
    return null;
  }

  const {
    altar_id,
    altar_name,
    path,
    tier,
    in_range,
    req_rank,
    req_dao_marks,
    required_item,
    required_item_count,
    guardian_defeated,
    already_claimed,
    legacy_text,
    rewards,
  } = selectedAltarInfo;

  const isImmortalTier = tier.toUpperCase() === "IMMORTAL";
  const playerRank = player?.rank || 1;
  const isRankMet = req_rank <= 0 || playerRank >= req_rank;

  // Check dao marks
  const daoMarksChecks: Array<{ path: string; req: number; current: number; ok: boolean }> = [];
  let isDaoMarksMet = true;
  if (req_dao_marks) {
    for (const [pName, minMarks] of Object.entries(req_dao_marks)) {
      let currentVal = 0;
      if (player?.dao_marks) {
        for (const [k, v] of Object.entries(player.dao_marks)) {
          if (k.toLowerCase() === pName.toLowerCase()) {
            currentVal = Number(v) || 0;
            break;
          }
        }
      }
      const ok = currentVal >= minMarks;
      if (!ok) isDaoMarksMet = false;
      daoMarksChecks.push({ path: pName, req: minMarks, current: currentVal, ok });
    }
  }

  const canClaim = in_range && guardian_defeated && isRankMet && isDaoMarksMet && !already_claimed;

  const handleClaim = async () => {
    if (!canClaim || altarLoading) return;
    await claimAltar(altar_id);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-fade-in">
      <div className="relative w-full max-w-xl max-h-[90vh] overflow-y-auto bg-gradient-to-b from-zinc-950 via-zinc-900 to-zinc-950 border border-amber-500/50 rounded-3xl p-6 shadow-[0_0_50px_rgba(245,158,11,0.25)] text-left">
        {/* Close Button */}
        <button
          onClick={() => setAltarModalOpen(false)}
          className="absolute top-4 right-4 p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800/60 rounded-full transition-all"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3 border-b border-amber-500/20 pb-4 mb-5">
          <div className="w-12 h-12 rounded-2xl bg-amber-500/10 border border-amber-500/40 flex items-center justify-center text-amber-400 shadow-[0_0_15px_rgba(245,158,11,0.3)]">
            <Landmark className="w-6 h-6 animate-pulse" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold font-serif text-amber-100 tracking-wide">
                {altar_name}
              </h2>
              <span
                className={`px-2.5 py-0.5 text-[10px] font-bold rounded-full border ${
                  isImmortalTier
                    ? "bg-amber-500/20 text-amber-300 border-amber-500/50 shadow-[0_0_10px_rgba(245,158,11,0.4)]"
                    : "bg-zinc-800 text-zinc-300 border-zinc-700"
                }`}
              >
                {isImmortalTier ? "TIÊN CẤP CHÂN TRUYỀN" : "PHÀM CẤP TRUYỀN THỪA"}
              </span>
            </div>
            <p className="text-xs text-amber-400/80 mt-0.5">Lưu phái: {path || "Toàn Lưu Phái"}</p>
          </div>
        </div>

        {/* Narrative Legacy Text */}
        {legacy_text && (
          <div className="relative mb-5 p-4 rounded-2xl bg-amber-950/20 border border-amber-500/20 italic text-sm text-amber-200/90 leading-relaxed font-serif">
            <span className="text-2xl text-amber-500/40 font-serif leading-none mr-1">“</span>
            {legacy_text}
            <span className="text-2xl text-amber-500/40 font-serif leading-none ml-1">”</span>
          </div>
        )}

        {/* Condition Checklist */}
        <div className="space-y-3 mb-6">
          <h3 className="text-xs font-bold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
            <ShieldAlert className="w-4 h-4 text-amber-400" />
            Điều Kiện Khảo Nghiệm
          </h3>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
            {/* 1. Distance */}
            <div className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800">
              <span className="text-zinc-300">Vị trí tương tác</span>
              <span className="flex items-center gap-1 font-semibold">
                {in_range ? (
                  <span className="text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-4 h-4" /> Đạt phạm vi
                  </span>
                ) : (
                  <span className="text-rose-400 flex items-center gap-1">
                    <XCircle className="w-4 h-4" /> Quá xa (&gt;2 ô)
                  </span>
                )}
              </span>
            </div>

            {/* 2. Guardian */}
            <div className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800">
              <span className="text-zinc-300">Thủ hộ giả</span>
              <span className="flex items-center gap-1 font-semibold">
                {guardian_defeated ? (
                  <span className="text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-4 h-4" /> Đã tiêu diệt
                  </span>
                ) : (
                  <span className="text-rose-400 flex items-center gap-1">
                    <XCircle className="w-4 h-4" /> Chưa tiêu diệt
                  </span>
                )}
              </span>
            </div>

            {/* 3. Rank */}
            {req_rank > 0 && (
              <div className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800">
                <span className="text-zinc-300">Cảnh giới yêu cầu</span>
                <span className="flex items-center gap-1 font-semibold">
                  {isRankMet ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <CheckCircle2 className="w-4 h-4" /> Chuyển {playerRank}/{req_rank}
                    </span>
                  ) : (
                    <span className="text-rose-400 flex items-center gap-1">
                      <XCircle className="w-4 h-4" /> Chuyển {playerRank}/{req_rank}
                    </span>
                  )}
                </span>
              </div>
            )}

            {/* 4. Token */}
            {required_item && (
              <div className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800">
                <span className="text-zinc-300">Tín vật</span>
                <span className="text-amber-300 font-semibold">
                  {required_item} (SL: {required_item_count || 1})
                </span>
              </div>
            )}

            {/* 5. Dao Marks */}
            {daoMarksChecks.map((chk) => (
              <div
                key={chk.path}
                className="flex items-center justify-between p-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800"
              >
                <span className="text-zinc-300">Đạo Ngân: {chk.path}</span>
                <span className="flex items-center gap-1 font-semibold">
                  {chk.ok ? (
                    <span className="text-emerald-400 flex items-center gap-1">
                      <CheckCircle2 className="w-4 h-4" /> {chk.current}/{chk.req}
                    </span>
                  ) : (
                    <span className="text-rose-400 flex items-center gap-1">
                      <XCircle className="w-4 h-4" /> {chk.current}/{chk.req}
                    </span>
                  )}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Rewards Preview */}
        {rewards && (
          <div className="space-y-3 mb-6">
            <h3 className="text-xs font-bold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
              <Sparkles className="w-4 h-4 text-amber-400" />
              Di Sản Truyền Thừa
            </h3>

            <div className="p-3.5 rounded-2xl bg-zinc-900/40 border border-zinc-800 space-y-2 text-xs">
              {/* Gus */}
              {rewards.gus && rewards.gus.length > 0 && (
                <div className="flex items-center gap-2">
                  <Gem className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                  <span className="text-zinc-400">Cổ trùng:</span>
                  <span className="font-semibold text-emerald-300">
                    {rewards.gus.join(", ")}
                  </span>
                </div>
              )}

              {/* Recipes */}
              {rewards.recipes && rewards.recipes.length > 0 && (
                <div className="flex items-center gap-2">
                  <BookOpen className="w-4 h-4 text-purple-400 flex-shrink-0" />
                  <span className="text-zinc-400">Phương thuốc:</span>
                  <span className="font-semibold text-purple-300">
                    {rewards.recipes.join(", ")}
                  </span>
                </div>
              )}

              {/* Primeval Stones */}
              {rewards.primeval_stones > 0 && (
                <div className="flex items-center gap-2">
                  <Coins className="w-4 h-4 text-amber-400 flex-shrink-0" />
                  <span className="text-zinc-400">Nguyên thạch:</span>
                  <span className="font-semibold text-amber-300">
                    +{rewards.primeval_stones.toLocaleString()}
                  </span>
                </div>
              )}

              {/* Materials */}
              {rewards.materials && Object.keys(rewards.materials).length > 0 && (
                <div className="flex items-center gap-2">
                  <PackageCheck className="w-4 h-4 text-cyan-400 flex-shrink-0" />
                  <span className="text-zinc-400">Linh tài / Đạo tài:</span>
                  <span className="font-semibold text-cyan-300">
                    {Object.entries(rewards.materials)
                      .map(([item, qty]) => `${item} x${qty}`)
                      .join(", ")}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Claim Action */}
        <div className="flex items-center justify-end gap-3 pt-3 border-t border-zinc-800">
          <button
            onClick={() => setAltarModalOpen(false)}
            className="px-4 py-2 text-xs font-semibold text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            Đóng
          </button>

          <button
            onClick={handleClaim}
            disabled={!canClaim || altarLoading}
            className={`px-6 py-2.5 text-xs font-bold rounded-xl transition-all shadow-lg flex items-center gap-2 ${
              already_claimed
                ? "bg-zinc-800 text-zinc-500 cursor-not-allowed border border-zinc-700"
                : canClaim
                ? "bg-gradient-to-r from-amber-500 to-yellow-600 hover:from-amber-400 hover:to-yellow-500 text-zinc-950 shadow-[0_0_20px_rgba(245,158,11,0.4)] active:scale-95"
                : "bg-zinc-800/80 text-zinc-500 cursor-not-allowed border border-zinc-700/50"
            }`}
          >
            <Landmark className="w-4 h-4" />
            <span>
              {altarLoading
                ? "Đang Lĩnh Nhận..."
                : already_claimed
                ? "Đã Hoàn Thành Khảo Nghiệm"
                : canClaim
                ? "Tiếp Nhận Khảo Nghiệm"
                : "Chưa Đạt Điều Kiện Khảo Nghiệm"}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
};
