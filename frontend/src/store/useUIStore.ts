"use client";

import { create } from "zustand";

export interface ToastMessage {
  id: string;
  message: string;
  type: "success" | "error" | "info";
  duration?: number;
}

export interface ConfirmDialogState {
  open: boolean;
  message: string;
  confirmText: string;
  cancelText: string;
}

interface UIStoreState {
  toasts: ToastMessage[];
  addToast: (message: string, type?: "success" | "error" | "info", duration?: number) => void;
  removeToast: (id: string) => void;
  confirmDialog: ConfirmDialogState;
  showConfirm: (message: string, options?: { confirmText?: string; cancelText?: string }) => Promise<boolean>;
  closeConfirm: (result: boolean) => void;
  currentView: string;
  setView: (view: string) => void;
  showChangelog: boolean;
  setShowChangelog: (v: boolean) => void;
}

let confirmResolve: ((value: boolean) => void) | null = null;

export const useUIStore = create<UIStoreState>((set, get) => ({
  toasts: [],
  currentView: "mmo-world",
  showChangelog: false,
  confirmDialog: { open: false, message: "", confirmText: "Xác nhận", cancelText: "Hủy" },

  addToast: (message, type = "info", duration = 4000) => {
    const id = Math.random().toString(36).substring(2, 9);
    set((state) => ({
      toasts: [...state.toasts, { id, message, type, duration }],
    }));
    setTimeout(() => {
      get().removeToast(id);
    }, duration);
  },
  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  showConfirm: async (message, options) => {
    return new Promise((resolve) => {
      confirmResolve = resolve;
      set({
        confirmDialog: {
          open: true,
          message,
          confirmText: options?.confirmText || "Xác nhận",
          cancelText: options?.cancelText || "Hủy",
        },
      });
    });
  },
  closeConfirm: (result) => {
    if (confirmResolve) {
      confirmResolve(result);
      confirmResolve = null;
    }
    set({ confirmDialog: { open: false, message: "", confirmText: "Xác nhận", cancelText: "Hủy" } });
  },

  setView: (view) => set({ currentView: view }),
  setShowChangelog: (v) => set({ showChangelog: v }),
}));
