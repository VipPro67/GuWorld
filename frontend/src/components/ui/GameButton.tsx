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
  xs: "text-[9px] px-2 py-1",
  sm: "text-[10px] px-3 py-1.5",
  md: "text-xs px-4 py-2",
  lg: "text-sm px-5 py-2.5",
};

const variantClasses: Record<GameButtonVariant, string> = {
  primary: [
    "bg-gradient-to-r from-violet-600 to-indigo-600",
    "hover:from-violet-500 hover:to-indigo-500",
    "text-white",
    "border border-transparent",
    "shadow-md shadow-violet-500/20 hover:shadow-violet-500/35",
    "disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50",
  ].join(" "),
  secondary: [
    "bg-zinc-900 hover:bg-zinc-800",
    "border border-zinc-800",
    "text-zinc-400 hover:text-zinc-200",
    "disabled:opacity-30 disabled:hover:bg-zinc-900",
  ].join(" "),
  ghost: [
    "bg-transparent hover:bg-zinc-900/60",
    "border border-zinc-900/60 hover:border-zinc-800",
    "text-zinc-400 hover:text-zinc-100",
    "disabled:opacity-30",
  ].join(" "),
  danger: [
    "bg-gradient-to-r from-red-600 to-rose-600",
    "hover:from-red-500 hover:to-rose-500",
    "text-white",
    "border border-transparent",
    "shadow-md shadow-red-500/20 hover:shadow-red-500/35",
    "disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50",
  ].join(" "),
  success: [
    "bg-gradient-to-r from-emerald-600 to-green-600",
    "hover:from-emerald-500 hover:to-green-500",
    "text-white",
    "border border-transparent",
    "shadow-md shadow-emerald-500/20 hover:shadow-emerald-500/35",
    "disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50",
  ].join(" "),
  warning: [
    "bg-gradient-to-r from-amber-500 to-orange-500",
    "hover:from-amber-400 hover:to-orange-400",
    "text-zinc-950",
    "border border-transparent",
    "shadow-lg shadow-amber-500/20 hover:shadow-amber-500/35",
    "disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50",
  ].join(" "),
  brand: [
    "bg-gradient-to-r from-brand to-brand-dim/50",
    "hover:from-brand-dim hover:to-brand",
    "text-white",
    "border border-brand/20",
    "shadow-md shadow-brand/20 hover:shadow-brand/35",
    "disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50",
  ].join(" "),
};

const baseClasses = [
  "inline-flex items-center justify-center gap-1.5",
  "font-semibold font-serif",
  "rounded-xl",
  "transition-all duration-200",
  "cursor-pointer",
  "active:scale-[0.98]",
  "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:ring-offset-2 focus:ring-offset-bg-primary",
  "disabled:cursor-not-allowed",
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