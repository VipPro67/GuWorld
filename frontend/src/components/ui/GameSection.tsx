"use client";

import React, { forwardRef } from "react";

export interface GameSectionProps extends React.HTMLAttributes<HTMLDivElement> {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  badge?: React.ReactNode;
  variant?: "default" | "bordered" | "elevated";
  className?: string;
}

const variantClasses = {
  default: "",
  bordered: "border-l-4 border-brand pl-4",
  elevated: "bg-surface/50 p-4 rounded-2xl border border-border/50",
};

export const GameSection = forwardRef<HTMLDivElement, GameSectionProps>(
  (
    {
      title,
      description,
      icon,
      action,
      badge,
      variant = "default",
      className = "",
      children,
      ...props
    },
    ref
  ) => (
    <div
      ref={ref}
      className={[
        "w-full",
        variantClasses[variant],
        className,
      ].join(" ")}
      {...props}
    >
      <div className="flex items-start justify-between gap-4 mb-4">
        <div className="flex items-start gap-3 min-w-0">
          {icon && (
            <div className="flex-shrink-0 w-8 h-8 rounded-xl bg-brand/10 flex items-center justify-center text-brand">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h2 className="text-lg font-bold font-serif text-text-primary flex items-center gap-2">
              {title}
              {badge && <span>{badge}</span>}
            </h2>
            {description && (
              <p className="mt-1 text-sm text-text-muted">{description}</p>
            )}
          </div>
        </div>
        {action && <div className="flex-shrink-0">{action}</div>}
      </div>
      <div className="space-y-3">{children}</div>
    </div>
  )
);

GameSection.displayName = "GameSection";

export interface GameSectionHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  badge?: React.ReactNode;
}

export const GameSectionHeader = ({
  title,
  description,
  icon,
  action,
  badge,
  className = "",
  ...props
}: GameSectionHeaderProps) => (
  <div className={["flex items-start justify-between gap-4 mb-4", className].join(" ")} {...props}>
    <div className="flex items-start gap-3 min-w-0">
      {icon && (
        <div className="flex-shrink-0 w-8 h-8 rounded-xl bg-brand/10 flex items-center justify-center text-brand">
          {icon}
        </div>
      )}
      <div className="min-w-0">
        <h2 className="text-lg font-bold font-serif text-text-primary flex items-center gap-2">
          {title}
          {badge && <span>{badge}</span>}
        </h2>
        {description && <p className="mt-1 text-sm text-text-muted">{description}</p>}
      </div>
    </div>
    {action && <div className="flex-shrink-0">{action}</div>}
  </div>
);

GameSectionHeader.displayName = "GameSectionHeader";

export const GameSectionContent = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <div ref={ref} className={["space-y-3", className].join(" ")} {...props}>
      {children}
    </div>
  )
);

GameSectionContent.displayName = "GameSectionContent";

export interface GameSectionRowProps extends React.HTMLAttributes<HTMLDivElement> {
  columns?: 1 | 2 | 3 | 4;
  gap?: "sm" | "md" | "lg";
}

export const GameSectionRow = ({
  columns = 2,
  gap = "md",
  className = "",
  children,
  ...props
}: GameSectionRowProps) => {
  const gridCols = {
    1: "grid-cols-1",
    2: "grid-cols-1 sm:grid-cols-2",
    3: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3",
    4: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-4",
  }[columns];

  const gapClass = {
    sm: "gap-3",
    md: "gap-4",
    lg: "gap-6",
  }[gap];

  return (
    <div
      className={["grid", gridCols, gapClass, className].join(" ")}
      {...props}
    >
      {children}
    </div>
  )
};

GameSectionRow.displayName = "GameSectionRow";

export interface GameStatItemProps {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  trend?: "up" | "down" | "neutral";
  trendValue?: string;
  variant?: "default" | "highlight";
}

export const GameStatItem = ({
  label,
  value,
  icon,
  trend,
  trendValue,
  variant = "default",
}: GameStatItemProps) => (
  <div className={[
    "flex items-center gap-3 p-3 rounded-xl transition-colors",
    variant === "highlight" ? "bg-brand-dim/50 border border-brand/20" : "bg-zinc-900/30 border border-zinc-800/50",
  ].join(" ")}>
    {icon && <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-zinc-900/50 flex items-center justify-center text-brand">{icon}</div>}
    <div className="flex-1 min-w-0">
      <p className="text-[10px] text-text-muted font-mono">{label}</p>
      <p className="text-sm font-bold font-mono text-text-primary">{value}</p>
    </div>
    {trend && trendValue && (
      <div className={[
        "flex items-center gap-1 text-[10px] font-mono font-bold",
        trend === "up" ? "text-success" : trend === "down" ? "text-error" : "text-text-muted",
      ].join(" ")}>
        {trend === "up" && <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" /></svg>}
        {trend === "down" && <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V4" /></svg>}
        {trendValue}
      </div>
    )}
  </div>
);

GameStatItem.displayName = "GameStatItem";