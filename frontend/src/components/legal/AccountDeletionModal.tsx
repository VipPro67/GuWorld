"use client";

import React, { useState } from "react";
import { AlertTriangle, Trash2, X, Lock } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";

interface AccountDeletionModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AccountDeletionModal: React.FC<AccountDeletionModalProps> = ({
  isOpen,
  onClose,
}) => {
  const [password, setPassword] = useState("");
  const [reason, setReason] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  const deleteAccount = useGameStore((state) => state.deleteAccount);
  const player = useGameStore((state) => state.player);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) {
      setErrorMsg("Vui lòng nhập mật khẩu xác nhận.");
      return;
    }

    setIsSubmitting(true);
    setErrorMsg("");

    try {
      const success = await deleteAccount(password, reason);
      if (success) {
        onClose();
      } else {
        setErrorMsg("Xác nhận mật khẩu không chính xác hoặc có lỗi xảy ra.");
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Đã có lỗi xảy ra.";
      setErrorMsg(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center overflow-y-auto overscroll-contain p-3 pt-[calc(env(safe-area-inset-top)+0.75rem)] pb-[calc(env(safe-area-inset-bottom)+0.75rem)] sm:p-4 bg-black/85 backdrop-blur-sm animate-in fade-in duration-200"
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-md max-h-[90dvh] overflow-y-auto overscroll-contain bg-zinc-950 border border-red-900/60 rounded-2xl p-4 sm:p-6 shadow-2xl shadow-red-950/40 text-zinc-100 flex flex-col gap-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-red-950/60 border border-red-800/50 flex items-center justify-center text-red-400">
              <Trash2 className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-base font-bold font-serif text-red-400">
                Xóa Tài Khoản & Dữ Liệu Cá Nhân
              </h2>
              <p className="text-[11px] text-zinc-400 font-mono">
                Quyền được xóa (Nghị định 13/2023/NĐ-CP)
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isSubmitting}
            aria-label="Đóng"
            className="p-1.5 text-zinc-400 hover:text-white rounded-lg hover:bg-zinc-800/80 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Warning Banner */}
        <div className="p-3 bg-red-950/30 border border-red-900/40 rounded-xl text-xs text-red-300 flex items-start gap-2.5">
          <AlertTriangle className="w-4 h-4 text-red-400 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <p className="font-semibold text-red-200">
              Cảnh báo: Hành động này KHÔNG THỂ HOÀN TÁC!
            </p>
            <p className="text-[11px] leading-relaxed text-zinc-300">
              Tài khoản <strong>{player?.username}</strong> sẽ bị vô hiệu hóa vĩnh viễn, phiên đăng nhập bị hủy bỏ, email và IP đăng ký được vô danh hóa theo đúng quy định bảo vệ dữ liệu cá nhân.
            </p>
          </div>
        </div>

        {errorMsg && (
          <div className="p-2.5 bg-red-900/40 border border-red-500/50 rounded-lg text-xs text-red-200">
            {errorMsg}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-3.5">
          <div>
            <label className="block text-xs font-semibold text-zinc-300 mb-1">
              Nhập Mật Khẩu Để Xác Nhận <span className="text-red-400">*</span>
            </label>
            <div className="relative">
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Mật khẩu tài khoản của đạo hữu"
                disabled={isSubmitting}
                className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-red-500 pl-9"
              />
              <Lock className="w-4 h-4 text-zinc-500 absolute left-3 top-3" />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-300 mb-1">
              Lý Do Xóa (Không bắt buộc)
            </label>
            <input
              type="text"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="VD: Không còn nhu cầu trải nghiệm, v.v."
              disabled={isSubmitting}
              className="w-full bg-zinc-900 border border-zinc-700 rounded-xl px-3.5 py-2.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-red-500"
            />
          </div>

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-zinc-800">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 rounded-xl text-xs font-serif text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 border border-zinc-800 transition-colors cursor-pointer"
            >
              Hủy Bỏ
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !password}
              className="px-4 py-2 rounded-xl text-xs font-serif font-bold text-white bg-red-700 hover:bg-red-600 border border-red-500 transition-all flex items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-lg shadow-red-950/50"
            >
              <Trash2 className="w-3.5 h-3.5" />
              {isSubmitting ? "Đang xử lý..." : "Xác Nhận Xóa Vĩnh Viễn"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
