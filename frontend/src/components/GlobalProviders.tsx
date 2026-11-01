"use client";

import { useEffect } from "react";
import { ToastContainer } from "@/components/ToastContainer";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { applyMenuThemeToCss } from "@/lib/ui-preferences";

export function GlobalProviders({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    applyMenuThemeToCss();
  }, []);

  return (
    <>
      {children}
      <ToastContainer />
      <ConfirmDialog />
    </>
  );
}
