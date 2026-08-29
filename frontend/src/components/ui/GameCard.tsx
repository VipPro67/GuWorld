"use client";

import React, { forwardRef } from "react";

export type GameCardVariant =
  | "default"
  | "hover"
  | "selected"
  | "vital"
  | "danger"
  | "compact";

export interface GameCardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: GameCardVariant;
  padding?: "none" | "sm" | "md" | "lg";
  onClick?: () => void;
}

const variantClasses: Record<GameCardVariant, string> = {
  default: "bg-surface/60 border-border",
  hover: "bg-surface/60 border-border transition-all duration-300 hover:bg-surface hover:border-border-hover hover:-translate-y-1 hover:shadow-lg",
  selected: "bg-brand-dim border-brand/50 shadow-[0_0_12px_rgba(5,150,105,0.15)] ring-1 ring-brand/20",
  vital: "bg-amber-950/10 border-amber-500/30 shadow-[0_0_15px_rgba(245,158,11,0.1)] animate-gold-glow",
  danger: "bg-red-950/10 border-red-500/30",
  compact: "bg-surface/40 border-border",
};

const paddingClasses = {
  none: "",
  sm: "p-3",
  md: "p-4 sm:p-5",
  lg: "p-5 sm:p-6",
};

export const GameCard = forwardRef<HTMLDivElement, GameCardProps>(
  (
    {
      variant = "default",
      padding = "md",
      onClick,
      className = "",
      children,
      ...props
    },
    ref
  ) => {
    const clickable = onClick !== undefined;
    const clickHandler = clickable
      ? (e: React.MouseEvent) => {
          e.currentTarget.classList.add("active:scale-[0.99]");
          onClick();
        }
      : undefined;

    return (
      <div
        ref={ref}
        className={[
          variantClasses[variant],
          paddingClasses[padding],
          "rounded-2xl",
          clickable ? "cursor-pointer" : "",
          className,
        ].join(" ")}
        onClick={clickHandler}
        role={clickable ? "button" : undefined}
        tabIndex={clickable ? 0 : undefined}
        onKeyDown={clickable ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onClick(); }} : undefined}
        {...props}
      >
        {children}
      </div>
    );
  }
);

GameCard.displayName = "GameCard";

export const GameCardHeader = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={["flex items-center justify-between gap-3 mb-3", className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GameCardHeader.displayName = "GameCardHeader";

export const GameCardContent = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={["space-y-2", className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GameCardContent.displayName = "GameCardContent";

export const GameCardFooter = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={["flex items-center justify-end gap-2 mt-3 pt-3 border-t border-border/50", className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GameCardFooter.displayName = "GameCardFooter";

export const GameCardMedia = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={["relative aspect-square rounded-xl overflow-hidden bg-zinc-900/50", className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GameCardMedia.displayName = "GameCardMedia";