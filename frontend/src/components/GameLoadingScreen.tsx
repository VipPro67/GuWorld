"use client";

import React from "react";
import { Sparkles, RefreshCw, AlertTriangle } from "@/lib/icons";

interface GameLoadingScreenProps {
  error?: string | null;
  onRetry?: () => void;
  loadingMessage?: string;
}

export const GameLoadingScreen: React.FC<GameLoadingScreenProps> = ({
  error,
  onRetry,
  loadingMessage = "Đang đồng bộ linh thức và dữ liệu Cổ Giới...",
}) => {
  return (
    <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-zinc-950 px-4 select-none">
      {/* Background ambient glow */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-emerald-950/30 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-64 h-64 bg-amber-950/20 rounded-full blur-2xl pointer-events-none" />

      {/* Main card */}
      <div className="relative z-10 w-full max-w-md bg-zinc-900/90 border border-zinc-800 rounded-2xl p-6 sm:p-8 shadow-2xl shadow-emerald-950/50 backdrop-blur text-center flex flex-col items-center">
        {/* Emblem / Icon */}
        <div className="relative mb-5 flex items-center justify-center">
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-emerald-500/20 to-teal-500/10 border border-emerald-500/40 flex items-center justify-center shadow-lg shadow-emerald-500/10">
            {error ? (
              <AlertTriangle className="w-8 h-8 text-amber-400 animate-bounce" />
            ) : (
              <Sparkles className="w-8 h-8 text-emerald-400 animate-pulse" />
            )}
          </div>
          {!error && (
            <div className="absolute -inset-1 rounded-2xl border border-emerald-500/30 animate-ping pointer-events-none" />
          )}
        </div>

        {/* Title */}
        <h1 className="text-2xl font-bold font-serif tracking-wider text-zinc-100 mb-1">
          Cổ Giới
        </h1>
        <p className="text-xs text-zinc-400 font-serif mb-6">
          Thế Giới Cổ Chân Nhân
        </p>

        {error ? (
          <div className="w-full flex flex-col items-center space-y-4">
            <div className="w-full bg-amber-950/40 border border-amber-500/30 rounded-lg p-3 text-left text-xs text-amber-200">
              <p className="font-semibold mb-1 flex items-center gap-1.5 text-amber-400">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                Không thể tải dữ liệu:
              </p>
              <p className="text-zinc-300 leading-relaxed">{error}</p>
            </div>

            <p className="text-xs text-zinc-400">
              Vui lòng kiểm tra đường truyền mạng hoặc bấm nút bên dưới để thử lại.
            </p>

            <button
              onClick={onRetry}
              className="w-full py-2.5 px-4 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-medium text-sm flex items-center justify-center gap-2 shadow-lg shadow-emerald-900/40 active:scale-[0.98] transition-all cursor-pointer"
            >
              <RefreshCw className="w-4 h-4" />
              Thử tải lại dữ liệu
            </button>
          </div>
        ) : (
          <div className="w-full flex flex-col items-center space-y-4">
            {/* Animated progress bar */}
            <div className="w-full bg-zinc-800/80 rounded-full h-2 overflow-hidden border border-zinc-700/50">
              <div className="bg-gradient-to-r from-emerald-500 via-teal-400 to-emerald-500 h-full w-full rounded-full animate-[shimmer_2s_infinite] -translate-x-full" style={{
                animation: "pulse 1.5s ease-in-out infinite",
              }} />
            </div>

            {/* Status message */}
            <p className="text-xs text-zinc-400 animate-pulse flex items-center justify-center gap-2 font-mono">
              <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-400" />
              {loadingMessage}
            </p>
          </div>
        )}

        {/* Footer info */}
        <div className="mt-8 pt-4 border-t border-zinc-800/60 w-full flex justify-between items-center text-[11px] text-zinc-500 font-mono">
          <span>Khởi tạo linh trận</span>
          <span>v2.0 • WebGL/IDB Ready</span>
        </div>
      </div>
    </div>
  );
};
