"use client";

import type { FormEvent, ReactNode } from "react";
import { GameButton, GameCard } from "@/components/ui";

export type AdminValidationResult = { valid: boolean; message?: ReactNode; errors?: Record<string, ReactNode> };

export interface AdminFormProps {
  children: ReactNode;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  validation?: AdminValidationResult;
  submitting?: boolean;
  disabled?: boolean;
  submitLabel?: string;
  cancelLabel?: string;
  onCancel?: () => void;
  actions?: ReactNode;
  className?: string;
}

export function AdminForm({ children, onSubmit, validation, submitting = false, disabled = false, submitLabel = "Lưu thay đổi", cancelLabel = "Hủy", onCancel, actions, className = "" }: AdminFormProps) {
  return (
    <GameCard padding="md" className={className}>
      <form onSubmit={onSubmit} noValidate className="space-y-5">
        <div className="grid gap-4">{children}</div>
        {validation?.message && <div role={validation.valid ? "status" : "alert"} className={`rounded-lg border px-3 py-2 text-xs ${validation.valid ? "border-emerald-500/20 bg-emerald-500/5 text-emerald-300" : "border-red-500/20 bg-red-500/5 text-red-300"}`}>{validation.message}</div>}
        <div className="flex flex-wrap justify-end gap-2 border-t border-zinc-800 pt-4">
          {onCancel && <GameButton type="button" variant="ghost" onClick={onCancel} disabled={submitting}>{cancelLabel}</GameButton>}
          {actions}
          <GameButton type="submit" variant="warning" loading={submitting} disabled={disabled || validation?.valid === false}>{submitLabel}</GameButton>
        </div>
      </form>
    </GameCard>
  );
}
