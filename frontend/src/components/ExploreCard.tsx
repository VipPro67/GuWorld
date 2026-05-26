"use client";

import React from "react";
import { Compass, Sparkles, AlertCircle } from "lucide-react";

interface ExploreCardProps {
  currentMana: number;
  maxMana: number;
  onExplore: () => void;
  isLoading: boolean;
  error: string | null;
}

export const ExploreCard: React.FC<ExploreCardProps> = ({
  currentMana,
  maxMana,
  onExplore,
  isLoading,
  error,
}) => {
  const canExplore = currentMana >= 10.0;

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-lg flex flex-col justify-between relative overflow-hidden group">
      {/* Decorative background aura */}
      <div className="absolute -right-16 -top-16 w-32 h-32 rounded-full bg-emerald-500/10 blur-2xl group-hover:bg-emerald-500/15 transition-all duration-700 pointer-events-none" />

      <div>
        <div className="flex items-center gap-3 mb-4">
          <div className="p-2.5 rounded-2xl bg-emerald-950/50 border border-emerald-500/20 text-emerald-400">
            <Compass className="w-5 h-5 animate-pulse" />
          </div>
          <div>
            <span className="text-[10px] text-amber-500 uppercase tracking-widest font-mono">
              Địa Danh (Map Node)
            </span>
            <h3 className="text-xl font-bold text-zinc-100 font-serif">
              Thanh Mao Sơn (Qing Mao Mountain)
            </h3>
          </div>
        </div>

        <p className="text-sm text-zinc-400 leading-relaxed mb-6 font-serif">
          Thanh Mao Sơn trải dài hàng ngàn dặm, rừng trúc xanh biếc bạt ngàn, linh khí dư dật. Cổ sư có thể tiêu hao Chân nguyên của bản thân để thâm nhập sâu vào các khe đá, tìm kiếm **Nguyên Thạch (Primeval Stones)** và nguyên liệu tu luyện quý giá.
        </p>

        <div className="bg-zinc-950/60 rounded-2xl p-4 border border-zinc-900 mb-6 space-y-2">
          <div className="flex justify-between text-xs">
            <span className="text-zinc-500">Chi phí tiêu hao:</span>
            <span className="text-emerald-400 font-semibold font-mono">
              10.0 Chân Nguyên (Mana)
            </span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="text-zinc-500">Trạng thái hiện tại:</span>
            <span
              className={`font-semibold font-mono ${
                canExplore ? "text-emerald-400" : "text-rose-500"
              }`}
            >
              {currentMana.toFixed(1)} / {maxMana.toFixed(0)} Chân Nguyên
            </span>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {error && (
          <div className="flex items-start gap-2 text-rose-400 bg-rose-950/20 border border-rose-900/40 p-3.5 rounded-xl text-xs">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <button
          onClick={onExplore}
          disabled={isLoading || !canExplore}
          className={`w-full py-4 px-6 rounded-2xl font-serif text-sm font-semibold tracking-wider transition-all duration-300 flex items-center justify-center gap-2 cursor-pointer shadow-lg ${
            isLoading
              ? "bg-zinc-800 text-zinc-500 cursor-not-allowed"
              : !canExplore
              ? "bg-zinc-900/50 text-zinc-600 border border-zinc-800/80 cursor-not-allowed hover:bg-zinc-900/50"
              : "bg-gradient-to-r from-emerald-600 to-teal-600 text-white hover:from-emerald-500 hover:to-teal-500 hover:shadow-[0_0_20px_rgba(16,185,129,0.3)] active:scale-[0.98]"
          }`}
        >
          {isLoading ? (
            <>
              <div className="w-4 h-4 border-2 border-zinc-500 border-t-zinc-200 rounded-full animate-spin" />
              Đang thâm nhập khám phá...
            </>
          ) : (
            <>
              <Sparkles className="w-4 h-4 text-yellow-300" />
              Khám Phá Sơn Mạch
            </>
          )}
        </button>

        {!canExplore && !isLoading && (
          <p className="text-[10px] text-rose-500/80 text-center italic">
            Chân nguyên cạn kiệt! Hãy đợi Chân nguyên trong Không Khiếu tự động hồi phục.
          </p>
        )}
      </div>
    </div>
  );
};
