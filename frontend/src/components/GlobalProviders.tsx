"use client";

import { ToastContainer } from "@/components/ToastContainer";
import { ConfirmDialog } from "@/components/ConfirmDialog";

export function GlobalProviders({ children }: { children: React.ReactNode }) {
  return (
    <>
      {children}
      <ToastContainer />
      <ConfirmDialog />
    </>
  );
}
