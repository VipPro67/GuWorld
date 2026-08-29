"use client";

import React from "react";

export type GamePanelVariant =
  | "default"
  | "elevated"
  | "gold"
  | "danger"
  | "success"
  | "brand"
  | "transparent";

export type GamePanelPadding = "none" | "sm" | "md" | "lg" | "xl";

export interface GamePanelProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: GamePanelVariant;
  padding?: GamePanelPadding;
  rounded?: "none" | "sm" | "md" | "lg" | "xl" | "full";
  bordered?: boolean;
  shadow?: "none" | "sm" | "md" | "lg" | "xl";
  hoverable?: boolean;
  clickable?: boolean;
}

const variantClasses: Record<GamePanelVariant, string> = {
  default: "bg-surface border-border",
  elevated: "bg-elevated border-border-hover",
  gold: "bg-surface border-warning shadow-[0_0_12px_rgba(217,119,6,0.15)]",
  danger: "bg-red-950/10 border-red-500/30",
  success: "bg-emerald-950/10 border-emerald-500/30",
  brand: "bg-brand-dim border-brand/30",
  transparent: "bg-transparent border-transparent",
};

const paddingClasses: Record<GamePanelPadding, string> = {
  none: "",
  sm: "p-3",
  md: "p-4 sm:p-5",
  lg: "p-5 sm:p-6",
  xl: "p-6 sm:p-8",
};

const roundedClasses = {
  none: "",
  sm: "rounded-lg",
  md: "rounded-xl",
  lg: "rounded-2xl",
  xl: "rounded-3xl",
  full: "rounded-full",
};

const shadowClasses = {
  none: "",
  sm: "shadow-sm",
  md: "shadow-md",
  lg: "shadow-lg",
  xl: "shadow-xl",
};

export const GamePanel = React.forwardRef<HTMLDivElement, GamePanelProps>(
  (
    {
      variant = "default",
      padding = "md",
      rounded = "lg",
      bordered = true,
      shadow = "md",
      hoverable = false,
      clickable = false,
      className = "",
      children,
      ...props
    },
    ref
  ) => {
    const borderClass = bordered ? "border border-border" : "border-0";
    const hoverClass = hoverable
      ? "transition-all duration-300 hover:-translate-y-1 hover:border-border-hover hover:shadow-lg"
      : "";
    const clickableClass = clickable
      ? "cursor-pointer active:scale-[0.99] active:shadow-md"
      : "";

    return (
      <div
        ref={ref}
        className={[
          variantClasses[variant],
          borderClass,
          paddingClasses[padding],
          roundedClasses[rounded],
          shadowClasses[shadow],
          hoverClass,
          clickableClass,
          className,
        ].join(" ")}
        {...props}
      >
        {children}
      </div>
    );
  }
);

GamePanel.displayName = "GamePanel";

export const GamePanelHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={[
        "flex items-center justify-between gap-4 mb-4 pb-3 border-b border-border",
        className,
      ].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GamePanelHeader.displayName = "GamePanelHeader";

export const GamePanelFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={[
        "flex items-center justify-end gap-3 mt-4 pt-3 border-t border-border",
        className,
      ].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GamePanelFooter.displayName = "GamePanelFooter";

export const GamePanelSection = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div
      ref={ref}
      className={["space-y-3", className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
);

GamePanelSection.displayName = "GamePanelSection";