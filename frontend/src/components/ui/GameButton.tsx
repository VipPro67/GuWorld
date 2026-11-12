"use client";

import React, { forwardRef } from "react";

export type GameButtonVariant =
  | "primary"
  | "secondary"
  | "ghost"
  | "danger"
  | "success"
  | "warning"
  | "brand";

export type GameButtonSize = "xs" | "sm" | "md" | "lg";

export interface GameButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: GameButtonVariant;
  size?: GameButtonSize;
  loading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  fullWidth?: boolean;
}

const sizeClasses: Record<GameButtonSize, string> = {
  xs: "text-[10px] px-2 py-1",
  sm: "text-[11px] px-3 py-1.5",
  md: "text-xs px-4 py-2",
  lg: "text-sm px-5 py-2.5",
};

const variantClasses: Record<GameButtonVariant, string> = {
  primary: [
    "bg-brand hover:brightness-110",
    "text-zinc-950",
    "border border-brand/70",
    "shadow-md shadow-brand/15",
  ].join(" "),
  secondary: [
    "bg-elevated hover:brightness-110",
    "border border-border",
    "text-text-primary",
  ].join(" "),
  ghost: [
    "bg-transparent hover:bg-elevated/60",
    "border border-transparent hover:border-border",
    "text-text-secondary hover:text-text-primary",
  ].join(" "),
  danger: [
    "bg-gradient-to-r from-red-600 to-rose-600",
    "hover:from-red-500 hover:to-rose-500",
    "text-white",
    "border border-transparent",
    "shadow-md shadow-red-500/20 hover:shadow-red-500/35",
  ].join(" "),
  success: [
    "bg-gradient-to-r from-emerald-600 to-green-600",
    "hover:from-emerald-500 hover:to-green-500",
    "text-white",
    "border border-transparent",
    "shadow-md shadow-emerald-500/20 hover:shadow-emerald-500/35",
  ].join(" "),
  warning: [
    "bg-gradient-to-r from-amber-500 to-orange-500",
    "hover:from-amber-400 hover:to-orange-400",
    "text-zinc-950",
    "border border-transparent",
    "shadow-lg shadow-amber-500/20 hover:shadow-amber-500/35",
  ].join(" "),
  brand: [
    "bg-brand-dim hover:brightness-110",
    "text-brand",
    "border border-brand/40",
    "shadow-sm shadow-brand/10",
  ].join(" "),
};

const baseClasses = [
  "inline-flex items-center justify-center gap-1.5",
  "font-semibold font-serif",
  "rounded-lg",
  "transition-all duration-200",
  "cursor-pointer",
  "active:scale-[0.98]",
  "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:ring-offset-2 focus:ring-offset-bg-primary",
  "disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:brightness-100",
].join(" ");

export const GameButton = forwardRef<HTMLButtonElement, GameButtonProps>(
  (
    {
      variant = "primary",
      size = "md",
      loading = false,
      leftIcon,
      rightIcon,
      fullWidth = false,
      className = "",
      children,
      disabled,
      ...props
    },
    ref
  ) => {
    const isDisabled = disabled || loading;

    return (
      <button
        ref={ref}
        className={[
          baseClasses,
          variantClasses[variant],
          sizeClasses[size],
          fullWidth ? "w-full" : "",
          className,
        ].join(" ")}
        disabled={isDisabled}
        aria-busy={loading}
        aria-disabled={isDisabled}
        {...props}
      >
        {loading ? (
          <>
            <svg
              className="animate-spin h-4 w-4"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="3"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
            <span>Đang xử lý...</span>
          </>
        ) : (
          <>
            {leftIcon && <span className="flex-shrink-0" aria-hidden="true">{leftIcon}</span>}
            {children}
            {rightIcon && <span className="flex-shrink-0" aria-hidden="true">{rightIcon}</span>}
          </>
        )}
      </button>
    );
  }
);

GameButton.displayName = "GameButton";