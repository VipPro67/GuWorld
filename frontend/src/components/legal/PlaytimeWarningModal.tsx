"use client";

import React from "react";
import { AlertTriangle, Clock, Heart } from "lucide-react";

interface PlaytimeWarningModalProps {
  isOpen: boolean;
  playtimeMinutes: number;
  onDismiss: () => void;
}

export const PlaytimeWarningModal: React.FC<PlaytimeWarningModalProps> = ({
  isOpen,
  playtimeMinutes,
  onDismiss,
}) => {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200"
      onClick={onDismiss}
    >
      <div
        className="relative w-full max-w-md bg-[#111827] border-2 border-red-600/70 rounded-2xl shadow-2xl shadow-red-950/50 text-zinc-200 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center gap-2.5 px-6 py-4 bg-red-950/40 border-b border-red-900/40">
          <AlertTriangle className="w-6 h-6 text-red-400 shrink-0 animate-pulse" />
          <div>
            <h3 className="text-sm font-bold text-red-300 uppercase tracking-wider font-serif">
              Cảnh Báo Sức Khỏe Pháp Định
            </h3>
            <p className="text-[10px] text-red-400/80">
              Điều 39 & 40 Nghị định 147/2024/NĐ-CP
            </p>
          </div>
        </div>

        {/* Body */}
        <div className="p-6 space-y-4 text-xs sm:text-sm text-zinc-300 leading-relaxed">
          <div className="flex items-center gap-3 p-3 bg-zinc-900/80 border border-zinc-800 rounded-xl">
            <Clock className="w-5 h-5 text-amber-400 shrink-0" />
            <div>
              <span className="text-zinc-400 text-xs">Thời gian tu luyện liên tục:</span>
              <p className="font-bold text-amber-400 text-base">
                {playtimeMinutes} phút ({Math.floor(playtimeMinutes / 60)} giờ {playtimeMinutes % 60} phút)
              </p>
            </div>
          </div>

          <div className="p-3 bg-red-950/20 border-l-4 border-red-500 rounded-r-lg text-xs text-red-200">
            ⚠️ <strong>"Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe."</strong>
          </div>

          <p className="text-zinc-400 text-xs">
            Kính gửi Đạo hữu! Việc nhập định vận chuyển chân nguyên liên tục quá 180 phút có thể gây hao tổn thần trí và suy giảm thị lực. Xin hãy tạm ngưng phiên chơi, vận động nhẹ nhàng để bảo dưỡng nhục thân.
          </p>
        </div>

        {/* Footer Action */}
        <div className="px-6 py-3.5 bg-zinc-950/80 border-t border-zinc-800/80 flex items-center justify-between gap-3">
          <div className="flex items-center gap-1.5 text-[11px] text-zinc-500">
            <Heart className="w-3.5 h-3.5 text-rose-500" />
            <span>Bảo vệ sức khỏe đạo hữu</span>
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="px-5 py-2 rounded-xl bg-gradient-to-r from-red-600 to-rose-700 hover:from-red-500 hover:to-rose-600 text-white font-serif font-bold text-xs transition-all cursor-pointer shadow-lg shadow-red-950/50 active:scale-95"
          >
            Ta Sẽ Tạm Nghỉ Sớm
          </button>
        </div>
      </div>
    </div>
  );
};
