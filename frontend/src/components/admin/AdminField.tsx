"use client";

import { useId, type ReactNode } from "react";

export interface AdminFieldProps {
  label: ReactNode;
  children: (props: { id: string; "aria-describedby"?: string; "aria-invalid": boolean }) => ReactNode;
  name?: string;
  hint?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  className?: string;
}

export function AdminField({ label, children, name, hint, error, required, className = "" }: AdminFieldProps) {
  const generatedId = useId();
  const id = name ?? generatedId;
  const descriptionId = error ? `${id}-error` : hint ? `${id}-hint` : undefined;
  const control = children({ id, "aria-describedby": descriptionId, "aria-invalid": Boolean(error) });
  return (
    <div className={className}>
      <label htmlFor={id} className="mb-1.5 block text-[10px] font-mono font-medium text-zinc-400">{label}{required && <span className="ml-1 text-red-400">*</span>}</label>
      {control}
      {error ? <div id={descriptionId} role="alert" className="mt-1.5 text-[10px] text-red-400">{error}</div> : hint ? <div id={descriptionId} className="mt-1.5 text-[10px] text-zinc-500">{hint}</div> : null}
    </div>
  );
}
