"use client";

import type { ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { GameButton, GameModal } from "@/components/ui";

export type AdminRiskLevel = "low" | "medium" | "high" | "critical";

export interface AdminConfirmDialogProps {
  open: boolean;
  title: string;
  description?: string;
  children?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  loading?: boolean;
  confirmDisabled?: boolean;
  riskLevel?: AdminRiskLevel;
  onConfirm: () => void;
  onCancel: () => void;
}

const riskStyle: Record<AdminRiskLevel, { modal: "default" | "warning" | "danger"; button: "primary" | "warning" | "danger"; icon: string }> = {
  low: { modal: "default", button: "primary", icon: "text-sky-400" },
  medium: { modal: "warning", button: "warning", icon: "text-amber-400" },
  high: { modal: "danger", button: "danger", icon: "text-red-400" },
  critical: { modal: "danger", button: "danger", icon: "text-red-400" },
};

export function AdminConfirmDialog({ open, title, description, children, confirmLabel = "Xác nhận", cancelLabel = "Hủy", loading = false, confirmDisabled = false, riskLevel = "medium", onConfirm, onCancel }: AdminConfirmDialogProps) {
  const style = riskStyle[riskLevel];
  return (
    <GameModal open={open} onClose={onCancel} title={title} description={description} variant={style.modal} size="sm" closeOnOverlayClick={!loading} closeOnEscape={!loading} showCloseButton={!loading} footer={<><GameButton type="button" variant="ghost" onClick={onCancel} disabled={loading}>{cancelLabel}</GameButton><GameButton type="button" variant={style.button} onClick={onConfirm} loading={loading} disabled={confirmDisabled}>{confirmLabel}</GameButton></>}>
      <div className="flex items-start gap-3 text-sm leading-6 text-zinc-300"><AlertTriangle className={`mt-0.5 h-5 w-5 shrink-0 ${style.icon}`} /><div className="min-w-0 flex-1">{children ?? "Hãy kiểm tra thông tin trước khi tiếp tục."}</div></div>
    </GameModal>
  );
}
