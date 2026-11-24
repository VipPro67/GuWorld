"use client";

import React from "react";
import { LegalDocType } from "./LegalModal";

interface LegalFooterProps {
  onOpenDoc: (doc: LegalDocType) => void;
  className?: string;
}

export const LegalFooter: React.FC<LegalFooterProps> = ({
  onOpenDoc,
  className = "",
}) => {
  return (
    <footer
      className={`w-full py-3.5 px-4 bg-zinc-950/80 border-t border-zinc-900/80 text-zinc-400 text-xs backdrop-blur-md transition-all ${className}`}
    >
      <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-center justify-between gap-3 text-center md:text-left">
        {/* Statutory 180-Minute Warning Banner (Decree 147/2024/NĐ-CP) */}
        <div className="flex items-center flex-wrap justify-center md:justify-start gap-2">
          <span className="px-1.5 py-0.5 rounded bg-red-600 text-white font-extrabold text-[10px] tracking-wider shrink-0">
            18+
          </span>
          <span className="text-[11px] text-zinc-400 font-medium">
            ⚠️ <strong>Cảnh Báo Pháp Định:</strong> Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe.
          </span>
        </div>

        {/* Legal Document Navigation Links */}
        <div className="flex items-center flex-wrap justify-center gap-4 text-[11px]">
          <button
            type="button"
            onClick={() => onOpenDoc("TOS")}
            className="hover:text-amber-400 underline decoration-zinc-700 underline-offset-4 transition-colors cursor-pointer"
          >
            Điều Khoản Dịch Vụ
          </button>
          <span className="text-zinc-700">•</span>
          <button
            type="button"
            onClick={() => onOpenDoc("PRIVACY")}
            className="hover:text-amber-400 underline decoration-zinc-700 underline-offset-4 transition-colors cursor-pointer"
          >
            Chính Sách Bảo Mật
          </button>
          <span className="text-zinc-700">•</span>
          <button
            type="button"
            onClick={() => onOpenDoc("PAYMENT")}
            className="hover:text-amber-400 underline decoration-zinc-700 underline-offset-4 transition-colors cursor-pointer"
          >
            Quy Chế Nạp & Bản Quyền
          </button>
        </div>

        {/* Disclaimer / Non-commercial note */}
        <div className="text-[10px] text-zinc-400">
          GuWorld © 2024. Bản thử nghiệm Alpha phi thương mại.
        </div>
      </div>
    </footer>
  );
};
