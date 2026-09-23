"use client";

import React, { useState } from "react";
import { Trash2, AlertTriangle, RefreshCw, X, CheckCircle2 } from "lucide-react";
import { clearLocalGameCache } from "@/lib/cache-utils";

interface ClearCacheModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ClearCacheModal: React.FC<ClearCacheModalProps> = ({ isOpen, onClose }) => {
  const [preserveAuth, setPreserveAuth] = useState(true);
  const [clearing, setClearing] = useState(false);

  if (!isOpen) return null;

  const handleClear = async () => {
    try {
      setClearing(true);
      await clearLocalGameCache({
        preserveAuth,
        reloadAfterClear: true,
      });
    } catch (err) {
      console.error("Lỗi khi xóa cache:", err);
      setClearing(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div
        className="relative w-full max-w-md bg-zinc-950 border border-zinc-800 rounded-2xl p-6 shadow-2xl text-zinc-100 flex flex-col gap-4 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Ambient Top Glow */}
        <div className="absolute -top-12 left-1/2 -translate-x-1/2 w-48 h-24 bg-amber-500/10 rounded-full blur-2xl pointer-events-none" />

        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400">
              <Trash2 className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-base font-bold font-serif text-zinc-100">
                Dọn Dẹp Bộ Nhớ Cục Bộ
              </h2>
              <p className="text-[11px] text-zinc-400 font-mono">
                Xóa cache & khắc phục lỗi dữ liệu
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            disabled={clearing}
            className="p-1 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60 transition-colors disabled:opacity-50 cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Warning info box */}
        <div className="bg-amber-950/20 border border-amber-500/20 rounded-xl p-3 text-xs text-amber-200/90 flex gap-2.5 items-start">
          <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
          <p className="leading-relaxed">
            Sử dụng khi gặp sự cố: <b>kẹt màn hình tải</b>, <b>không thấy vật phẩm/cổ trùng mới cập nhật</b>, hoặc giao diện hiển thị bất thường do bộ nhớ đệm trình duyệt bị lỗi.
          </p>
        </div>

        {/* Options */}
        <div className="flex flex-col gap-2.5 py-1">
          {/* Option 1: Preserve auth */}
          <div
            onClick={() => !clearing && setPreserveAuth(true)}
            className={`p-3 rounded-xl border transition-all cursor-pointer flex items-start gap-3 ${
              preserveAuth
                ? "bg-emerald-950/20 border-emerald-500/40 text-zinc-200"
                : "bg-zinc-900/40 border-zinc-800 text-zinc-400 hover:bg-zinc-900/70"
            }`}
          >
            <div className={`mt-0.5 w-4 h-4 rounded-full border flex items-center justify-center shrink-0 ${
              preserveAuth ? "border-emerald-400 bg-emerald-500/20 text-emerald-400" : "border-zinc-600"
            }`}>
              {preserveAuth && <div className="w-2 h-2 rounded-full bg-emerald-400" />}
            </div>
            <div className="flex-1">
              <div className="text-xs font-semibold flex items-center gap-1.5 text-zinc-200">
                <span>Xóa Cache & Giữ Đăng Nhập</span>
                <span className="text-[10px] px-1.5 py-0.2 bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 rounded">Khuyên Dùng</span>
              </div>
              <p className="text-[11px] text-zinc-400 mt-0.5 leading-relaxed">
                Xóa sạch IndexedDB, từ điển game, Cache Storage. Giữ lại thông tin tài khoản để không cần đăng nhập lại.
              </p>
            </div>
          </div>

          {/* Option 2: Full wipe */}
          <div
            onClick={() => !clearing && setPreserveAuth(false)}
            className={`p-3 rounded-xl border transition-all cursor-pointer flex items-start gap-3 ${
              !preserveAuth
                ? "bg-red-950/20 border-red-500/40 text-zinc-200"
                : "bg-zinc-900/40 border-zinc-800 text-zinc-400 hover:bg-zinc-900/70"
            }`}
          >
            <div className={`mt-0.5 w-4 h-4 rounded-full border flex items-center justify-center shrink-0 ${
              !preserveAuth ? "border-red-400 bg-red-500/20 text-red-400" : "border-zinc-600"
            }`}>
              {!preserveAuth && <div className="w-2 h-2 rounded-full bg-red-400" />}
            </div>
            <div className="flex-1">
              <div className="text-xs font-semibold text-zinc-200">
                Xóa Toàn Bộ Dữ Liệu (Bao gồm Đăng Xuất)
              </div>
              <p className="text-[11px] text-zinc-400 mt-0.5 leading-relaxed">
                Xóa toàn bộ bộ nhớ cục bộ, đăng xuất tài khoản và tải lại game về trạng thái nguyên bản.
              </p>
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center justify-end gap-2.5 pt-2 border-t border-zinc-800/80">
          <button
            onClick={onClose}
            disabled={clearing}
            className="px-4 py-2 rounded-xl text-xs font-serif text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 border border-zinc-800 transition-colors disabled:opacity-50 cursor-pointer"
          >
            Hủy Bỏ
          </button>
          <button
            onClick={handleClear}
            disabled={clearing}
            className="px-4 py-2 rounded-xl text-xs font-serif font-bold text-white bg-gradient-to-r from-amber-600 to-orange-600 hover:from-amber-500 hover:to-orange-500 transition-all flex items-center gap-2 shadow-lg shadow-amber-950/30 active:scale-95 disabled:opacity-50 cursor-pointer"
          >
            {clearing ? (
              <>
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                <span>Đang dọn dẹp...</span>
              </>
            ) : (
              <>
                <Trash2 className="w-3.5 h-3.5" />
                <span>Xác Nhận & Tải Lại Trang</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export const ClearCacheButton: React.FC<{ className?: string; showLabel?: boolean }> = ({
  className = "",
  showLabel = false,
}) => {
  const [modalOpen, setModalOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setModalOpen(true)}
        title="Xóa bộ nhớ đệm cục bộ (Clear Cache)"
        className={`px-2.5 py-1.5 rounded-xl bg-[var(--bg-surface)] hover:bg-[var(--bg-elevated)] text-xs text-amber-500/90 dark:text-amber-400 hover:text-amber-600 dark:hover:text-amber-300 font-serif transition-colors border border-[var(--border-default)] cursor-pointer inline-flex items-center gap-1.5 ${className}`}
      >
        <Trash2 className="w-3.5 h-3.5 shrink-0" />
        {showLabel && <span>Xóa Cache</span>}
      </button>

      <ClearCacheModal isOpen={modalOpen} onClose={() => setModalOpen(false)} />
    </>
  );
};
