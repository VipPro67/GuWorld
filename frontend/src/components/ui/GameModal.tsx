"use client";

import React, { useEffect, useCallback, forwardRef } from "react";
import { createPortal } from "react-dom";
import { GameButton } from "./GameButton";

export type GameModalSize = "sm" | "md" | "lg" | "xl" | "full";

export interface GameModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  size?: GameModalSize;
  variant?: "default" | "danger" | "success" | "warning";
  children: React.ReactNode;
  footer?: React.ReactNode;
  closeOnOverlayClick?: boolean;
  closeOnEscape?: boolean;
  showCloseButton?: boolean;
  className?: string;
  portalContainer?: HTMLElement | null;
}

const sizeClasses: Record<GameModalSize, string> = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-lg",
  xl: "max-w-xl",
  full: "max-w-4xl",
};

const variantClasses = {
  default: "border-border",
  danger: "border-red-500/30 shadow-[0_0_30px_rgba(239,68,68,0.15)]",
  success: "border-emerald-500/30 shadow-[0_0_30px_rgba(16,185,129,0.15)]",
  warning: "border-amber-500/30 shadow-[0_0_30px_rgba(245,158,11,0.15)]",
};

export const GameModal = forwardRef<HTMLDivElement, GameModalProps>(
  (
    {
      open,
      onClose,
      title,
      description,
      size = "md",
      variant = "default",
      children,
      footer,
      closeOnOverlayClick = true,
      closeOnEscape = true,
      showCloseButton = true,
      className = "",
      portalContainer,
      ...props
    },
    ref
  ) => {
    const handleKeyDown = useCallback(
      (e: KeyboardEvent) => {
        if (e.key === "Escape" && closeOnEscape) {
          onClose();
        }
      },
      [closeOnEscape, onClose]
    );

    useEffect(() => {
      if (open) {
        document.addEventListener("keydown", handleKeyDown);
        document.body.style.overflow = "hidden";
      }
      return () => {
        document.removeEventListener("keydown", handleKeyDown);
        document.body.style.overflow = "";
      };
    }, [open, handleKeyDown]);

    if (!open) return null;

    const modalContent = (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? "modal-title" : undefined}
        aria-describedby={description ? "modal-description" : undefined}
      >
        <div
          className="absolute inset-0 bg-zinc-950/80 backdrop-blur-sm transition-opacity"
          onClick={closeOnOverlayClick ? onClose : undefined}
          aria-hidden="true"
        />
        <div
          ref={ref}
          className={[
            "relative w-full animate-in fade-in zoom-in-95 duration-200",
            "bg-surface",
            "rounded-3xl",
            "shadow-2xl",
            "max-h-[90vh] flex flex-col overflow-hidden",
            sizeClasses[size],
            variantClasses[variant],
            className,
          ].join(" ")}
          {...props}
        >
          {(title || showCloseButton) && (
            <div className="flex items-start justify-between gap-4 p-5 border-b border-border/50">
              <div className="flex-1 min-w-0">
                {title && (
                  <h2
                    id="modal-title"
                    className="text-lg font-bold font-serif text-text-primary"
                  >
                    {title}
                  </h2>
                )}
                {description && (
                  <p
                    id="modal-description"
                    className="mt-1 text-sm text-text-muted"
                  >
                    {description}
                  </p>
                )}
              </div>
              {showCloseButton && (
                <button
                  type="button"
                  onClick={onClose}
                  className="flex-shrink-0 p-1.5 rounded-xl text-text-muted hover:text-text-primary hover:bg-zinc-900/50 transition-colors"
                  aria-label="Đóng"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>
          )}
          <div className="flex-1 overflow-y-auto p-5">
            {children}
          </div>
          {footer && (
            <div className="flex items-center justify-end gap-3 p-5 border-t border-border/50 bg-zinc-900/20">
              {footer}
            </div>
          )}
        </div>
      </div>
    );

    if (portalContainer) {
      return createPortal(modalContent, portalContainer);
    }

    if (typeof document !== "undefined") {
      return createPortal(modalContent, document.body);
    }

    return null;
  }
);

GameModal.displayName = "GameModal";

export interface GameConfirmModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "danger" | "warning" | "success";
  loading?: boolean;
}

export const GameConfirmModal = ({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = "Xác nhận",
  cancelText = "Hủy",
  variant = "danger",
  loading = false,
}: GameConfirmModalProps) => (
  <GameModal
    open={open}
    onClose={onClose}
    title={title}
    variant={variant}
    size="sm"
    footer={
      <>
        <GameButton variant="ghost" onClick={onClose} disabled={loading}>
          {cancelText}
        </GameButton>
        <GameButton variant={variant} onClick={onConfirm} loading={loading}>
          {confirmText}
        </GameButton>
      </>
    }
  >
    <p className="text-text-secondary">{message}</p>
  </GameModal>
);

GameConfirmModal.displayName = "GameConfirmModal";

export interface GameAlertModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  message: string;
  variant?: "success" | "error" | "warning" | "info";
  actionText?: string;
  onAction?: () => void;
}

export const GameAlertModal = ({
  open,
  onClose,
  title,
  message,
  variant = "info",
  actionText = "Đóng",
  onAction,
}: GameAlertModalProps) => {
  const variantIcon = {
    success: (
      <svg className="w-12 h-12 text-success mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    error: (
      <svg className="w-12 h-12 text-error mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    warning: (
      <svg className="w-12 h-12 text-warning mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
    info: (
      <svg className="w-12 h-12 text-info mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  };

  return (
    <GameModal
      open={open}
      onClose={onClose}
      title={title}
      variant={variant === "error" ? "danger" : (variant === "info" ? "default" : variant)}
      size="sm"
      showCloseButton={false}
      footer={
        <div className="w-full">
          <GameButton variant="primary" className="w-full" onClick={() => { onAction?.(); onClose(); }}>
            {actionText}
          </GameButton>
        </div>
      }
    >
      <div className="text-center">
        {variantIcon[variant]}
        <p className="text-text-secondary">{message}</p>
      </div>
    </GameModal>
  );
};

GameAlertModal.displayName = "GameAlertModal";