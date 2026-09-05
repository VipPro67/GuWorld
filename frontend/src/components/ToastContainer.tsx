import React from "react";
import { useUIStore, ToastMessage } from "@/store/useUIStore";
import { CheckCircle2, AlertTriangle, Info, X } from "lucide-react";

export const ToastContainer: React.FC = () => {
  const toasts = useUIStore((s) => s.toasts);
  const removeToast = useUIStore((s) => s.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 left-4 sm:bottom-6 sm:left-6 z-[9999] flex flex-col gap-2 sm:gap-3 max-w-[280px] sm:max-w-sm w-full pointer-events-none">
      {toasts.map((toast) => {
        return (
          <ToastItem
            key={toast.id}
            toast={toast}
            onClose={() => removeToast(toast.id)}
          />
        );
      })}
    </div>
  );
};

interface ToastItemProps {
  toast: ToastMessage;
  onClose: () => void;
}

const ToastItem: React.FC<ToastItemProps> = ({ toast, onClose }) => {
  const { message, type } = toast;

  // Configuration based on toast type
  const configs = {
    success: {
      borderColor: "border-emerald-500/30",
      shadowColor: "shadow-emerald-950/20",
      glowColor: "rgba(16, 185, 129, 0.15)",
      bgColor: "bg-emerald-950/30",
      iconColor: "text-emerald-400",
      icon: CheckCircle2,
    },
    error: {
      borderColor: "border-rose-500/30",
      shadowColor: "shadow-rose-950/20",
      glowColor: "rgba(244, 63, 94, 0.15)",
      bgColor: "bg-rose-950/30",
      iconColor: "text-rose-400",
      icon: AlertTriangle,
    },
    info: {
      borderColor: "border-violet-500/30",
      shadowColor: "shadow-violet-950/20",
      glowColor: "rgba(139, 92, 246, 0.15)",
      bgColor: "bg-violet-950/30",
      iconColor: "text-violet-400",
      icon: Info,
    },
  };
  const config = configs[type] || configs.info;

  const Icon = config.icon;

  return (
    <div
      className={`pointer-events-auto flex items-start gap-2 sm:gap-3 w-full p-2.5 sm:p-4 rounded-xl sm:rounded-2xl border ${config.borderColor} bg-zinc-950/90 backdrop-blur-md shadow-xl ${config.shadowColor} transition-all duration-300 transform translate-x-0 animate-toast-in relative overflow-hidden`}
      style={{
        boxShadow: `0 8px 32px 0 rgba(0, 0, 0, 0.6), 0 0 15px ${config.glowColor}`,
      }}
    >
      {/* Decorative accent background */}
      <div className={`absolute top-0 left-0 w-1 h-full ${config.bgColor}`} />

      {/* Icon */}
      <div className={`shrink-0 mt-0.5 ${config.iconColor}`}>
        <Icon className="w-4 h-4 sm:w-5 sm:h-5" />
      </div>

      {/* Message content */}
      <div className="flex-grow pr-3 sm:pr-4">
        <p className="text-[11px] sm:text-sm font-mono font-medium text-zinc-200 leading-snug sm:leading-relaxed break-words">
          {message}
        </p>
      </div>

      {/* Close button */}
      <button
        onClick={onClose}
        className="shrink-0 p-0.5 rounded-lg text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900/60 transition-all cursor-pointer"
      >
        <X className="w-3.5 h-3.5 sm:w-4 h-4" />
      </button>
    </div>
  );
};
