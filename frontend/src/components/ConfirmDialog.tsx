import React, { useEffect, useRef } from "react";
import { useUIStore } from "@/store/useUIStore";
import { AlertTriangle, X } from "lucide-react";

export const ConfirmDialog: React.FC = () => {
  const confirmDialog = useUIStore((s) => s.confirmDialog);
  const closeConfirm = useUIStore((s) => s.closeConfirm);
  const dialogRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!confirmDialog.open) return;
    previouslyFocusedRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const frame = requestAnimationFrame(() => {
      dialogRef.current?.querySelector<HTMLElement>("[data-confirm-cancel]")?.focus();
    });
    return () => {
      cancelAnimationFrame(frame);
      document.body.style.overflow = previousOverflow;
      if (previouslyFocusedRef.current?.isConnected) previouslyFocusedRef.current.focus();
    };
  }, [confirmDialog.open]);

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      closeConfirm(false);
      return;
    }
    if (event.key !== "Tab") return;
    const controls = dialogRef.current?.querySelectorAll<HTMLElement>(
      'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled])'
    );
    if (!controls?.length) {
      event.preventDefault();
      dialogRef.current?.focus();
    } else if (event.shiftKey && document.activeElement === controls[0]) {
      event.preventDefault();
      controls[controls.length - 1].focus();
    } else if (!event.shiftKey && document.activeElement === controls[controls.length - 1]) {
      event.preventDefault();
      controls[0].focus();
    }
  };

  if (!confirmDialog.open) return null;

  return (
    <div className="fixed inset-0 z-[10000] h-[100dvh] flex items-center justify-center overscroll-contain p-4 pt-[calc(env(safe-area-inset-top)+1rem)] pb-[calc(env(safe-area-inset-bottom)+1rem)]" onKeyDown={handleKeyDown}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => closeConfirm(false)} />
      <div
        ref={dialogRef}
        role="alertdialog"
        aria-modal="true"
        aria-label="Xác nhận hành động"
        aria-describedby="confirm-message"
        tabIndex={-1}
        className="relative bg-zinc-950 border border-zinc-800 rounded-2xl shadow-2xl w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-200"
        style={{ boxShadow: "0 0 40px rgba(0,0,0,0.6)" }}
      >
        <div className="flex items-start gap-3 p-5">
          <div className="shrink-0 mt-0.5 text-amber-400">
            <AlertTriangle className="w-6 h-6" />
          </div>
          <div className="flex-1 min-w-0">
            <p id="confirm-message" className="text-sm font-mono text-zinc-200 leading-relaxed whitespace-pre-wrap">
              {confirmDialog.message}
            </p>
          </div>
          <button
            type="button"
            aria-label="Hủy xác nhận"
            onClick={() => closeConfirm(false)}
            className="inline-flex min-h-11 min-w-11 shrink-0 items-center justify-center rounded-lg text-zinc-600 hover:text-zinc-300 hover:bg-zinc-900/60 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex gap-2 px-5 pb-5">
          <button
            type="button"
            data-confirm-cancel
            onClick={() => closeConfirm(false)}
            className="min-h-11 flex-1 py-2.5 rounded-xl border border-zinc-700 bg-zinc-900/50 text-zinc-300 hover:bg-zinc-800 text-sm font-bold transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] cursor-pointer"
          >
            {confirmDialog.cancelText}
          </button>
          <button
            type="button"
            onClick={() => closeConfirm(true)}
            className="min-h-11 flex-1 py-2.5 rounded-xl bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 text-white text-sm font-bold shadow-lg shadow-rose-950/30 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] cursor-pointer"
          >
            {confirmDialog.confirmText}
          </button>
        </div>
      </div>
    </div>
  );
};
