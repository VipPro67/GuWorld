import React from "react";
import { useGameStore } from "@/store/useGameStore";
import { AlertTriangle, X } from "lucide-react";

export const ConfirmDialog: React.FC = () => {
  const { confirmDialog, closeConfirm } = useGameStore();

  if (!confirmDialog.open) return null;

  return (
    <div className="fixed inset-0 z-[10000] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => closeConfirm(false)} />
      <div
        className="relative bg-zinc-950 border border-zinc-800 rounded-2xl shadow-2xl w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-200"
        style={{ boxShadow: "0 0 40px rgba(0,0,0,0.6)" }}
      >
        <div className="flex items-start gap-3 p-5">
          <div className="shrink-0 mt-0.5 text-amber-400">
            <AlertTriangle className="w-6 h-6" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-mono text-zinc-200 leading-relaxed whitespace-pre-wrap">
              {confirmDialog.message}
            </p>
          </div>
          <button
            onClick={() => closeConfirm(false)}
            className="shrink-0 p-1 rounded-lg text-zinc-600 hover:text-zinc-300 hover:bg-zinc-900/60 transition-all cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex gap-2 px-5 pb-5">
          <button
            onClick={() => closeConfirm(false)}
            className="flex-1 py-2.5 rounded-xl border border-zinc-700 bg-zinc-900/50 text-zinc-300 hover:bg-zinc-800 text-sm font-bold transition-all cursor-pointer"
          >
            {confirmDialog.cancelText}
          </button>
          <button
            onClick={() => closeConfirm(true)}
            className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-rose-700 to-red-600 hover:from-rose-600 hover:to-red-500 text-white text-sm font-bold shadow-lg shadow-rose-950/30 transition-all cursor-pointer"
          >
            {confirmDialog.confirmText}
          </button>
        </div>
      </div>
    </div>
  );
};
