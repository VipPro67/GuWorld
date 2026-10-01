"use client";

import { useEffect, useState } from "react";
import { GameInput } from "@/components/ui";
import { AdminConfirmDialog, type AdminConfirmDialogProps, type AdminRiskLevel } from "./AdminConfirmDialog";

export interface AdminDangerDialogProps extends Omit<AdminConfirmDialogProps, "riskLevel" | "onConfirm"> {
  riskLevel: AdminRiskLevel;
  onConfirm: () => void;
  confirmationText?: string;
}

export function AdminDangerDialog({ riskLevel, onConfirm, confirmationText, open, onCancel, children, ...props }: AdminDangerDialogProps) {
  const [confirmation, setConfirmation] = useState("");
  const needsPhrase = Boolean(confirmationText);
  useEffect(() => setConfirmation(""), [open, confirmationText]);
  const close = () => { setConfirmation(""); onCancel(); };
  const confirm = () => { if (needsPhrase && confirmation !== confirmationText) return; setConfirmation(""); onConfirm(); };
  const confirmDisabled = props.confirmDisabled || (needsPhrase && confirmation !== confirmationText);
  return (
    <AdminConfirmDialog {...props} open={open} onCancel={close} onConfirm={confirm} confirmDisabled={confirmDisabled} riskLevel={riskLevel} confirmLabel={props.confirmLabel ?? "Thực hiện thao tác"}>
      <div className="space-y-3">
        {children}
        {needsPhrase && <div><p className="mb-2 text-xs text-zinc-400">Nhập <strong className="font-mono text-red-300">{confirmationText}</strong> để xác nhận:</p><GameInput aria-label="Chuỗi xác nhận" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} autoComplete="off" /></div>}
      </div>
    </AdminConfirmDialog>
  );
}
