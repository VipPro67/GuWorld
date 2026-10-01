"use client";

import { Braces, WandSparkles } from "lucide-react";
import { GameButton, GameTextarea } from "@/components/ui";
import type { AdminValidationResult } from "./AdminForm";

export interface AdminJsonEditorProps {
  value: string;
  onChange: (value: string) => void;
  validation?: AdminValidationResult;
  label?: string;
  description?: string;
  disabled?: boolean;
  rows?: number;
  onFormat?: () => void;
}

export function AdminJsonEditor({ value, onChange, validation, label = "JSON", description, disabled = false, rows = 12, onFormat }: AdminJsonEditorProps) {
  return (
    <section className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-950/40">
      <div className="flex items-start justify-between gap-3 border-b border-zinc-800 px-4 py-3">
        <div><h3 className="flex items-center gap-2 text-xs font-bold text-zinc-200"><Braces className="h-4 w-4 text-amber-400" />{label}</h3>{description && <p className="mt-1 text-[11px] text-zinc-500">{description}</p>}</div>
        {onFormat && <GameButton type="button" variant="ghost" size="xs" onClick={onFormat} disabled={disabled} leftIcon={<WandSparkles className="h-3.5 w-3.5" />}>Định dạng</GameButton>}
      </div>
      <div className="p-3">
        <GameTextarea aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} disabled={disabled} minRows={rows} spellCheck={false} className="font-mono" error={validation?.valid === false && typeof validation.message === "string" ? validation.message : undefined} hint={validation?.valid && typeof validation.message === "string" ? validation.message : undefined} />
        {validation?.message && typeof validation.message !== "string" && <div role={validation.valid ? "status" : "alert"} className={`mt-2 text-xs ${validation.valid ? "text-emerald-400" : "text-red-400"}`}>{validation.message}</div>}
      </div>
    </section>
  );
}
