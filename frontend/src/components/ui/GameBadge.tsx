"use client";

import React, { forwardRef } from "react";

export type GameBadgeVariant =
  | "default"
  | "rank"
  | "path"
  | "rarity"
  | "status"
  | "vital"
  | "equipped"
  | "sleeping"
  | "cooldown"
  | "approval"
  | "region"
  | "realm";

export type GameBadgeSize = "xs" | "sm" | "md";

export interface GameBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: GameBadgeVariant;
  size?: GameBadgeSize;
  rank?: number;
  path?: "kim" | "moc" | "thuy" | "hoa" | "tho";
  rarity?: "common" | "uncommon" | "rare" | "epic" | "legendary" | "mythic";
  status?: "active" | "inactive" | "pending" | "completed" | "failed";
  dot?: boolean;
  removable?: boolean;
  onRemove?: () => void;
}

const sizeClasses: Record<GameBadgeSize, string> = {
  xs: "text-[9px] px-1.5 py-0.5 gap-0.5",
  sm: "text-[10px] px-2 py-0.5 gap-1",
  md: "text-xs px-2.5 py-1 gap-1.5",
};

const baseClasses = "inline-flex items-center font-mono font-semibold rounded-full";

const variantClasses: Record<GameBadgeVariant, string> = {
  default: "bg-zinc-900/50 text-zinc-400 border border-zinc-800",
  rank: "border-0",
  path: "border-0 font-bold uppercase tracking-wider",
  rarity: "border-0 font-bold",
  status: "border-0",
  vital: "bg-amber-950/30 text-amber-400 border border-amber-500/30 animate-pulse",
  equipped: "bg-emerald-950/30 text-emerald-400 border border-emerald-500/30",
  sleeping: "bg-amber-950/30 text-amber-400 border border-amber-500/30",
  cooldown: "bg-blue-950/30 text-blue-400 border border-blue-500/30",
  approval: "bg-amber-950/30 text-amber-400 border border-amber-500/30",
  region: "bg-zinc-900 text-zinc-400 border border-zinc-800",
  realm: "bg-brand-dim text-brand border border-brand/30",
};

const rankColors: Record<number, string> = {
  1: "bg-emerald-950/30 text-emerald-400 border-emerald-500/30",
  2: "bg-red-950/30 text-red-400 border-red-500/30",
  3: "bg-blue-950/30 text-blue-400 border-blue-500/30",
  4: "bg-amber-950/30 text-amber-400 border-amber-500/30",
  5: "bg-violet-950/30 text-violet-400 border-violet-500/30",
  6: "bg-cyan-950/30 text-cyan-400 border-cyan-500/30",
  7: "bg-pink-950/30 text-pink-400 border-pink-500/30",
  8: "bg-stone-950/30 text-stone-400 border-stone-500/30",
  9: "bg-yellow-950/30 text-yellow-400 border-yellow-500/30",
};

const pathColors: Record<string, string> = {
  kim: "bg-amber-950/30 text-amber-400 border-amber-500/30",
  moc: "bg-emerald-950/30 text-emerald-400 border-emerald-500/30",
  thuy: "bg-blue-950/30 text-blue-400 border-blue-500/30",
  hoa: "bg-red-950/30 text-red-400 border-red-500/30",
  tho: "bg-yellow-950/30 text-yellow-400 border-yellow-500/30",
};

const rarityColors: Record<string, string> = {
  common: "bg-zinc-900/50 text-zinc-400 border-zinc-800",
  uncommon: "bg-emerald-950/30 text-emerald-400 border-emerald-500/30",
  rare: "bg-blue-950/30 text-blue-400 border-blue-500/30",
  epic: "bg-violet-950/30 text-violet-400 border-violet-500/30",
  legendary: "bg-amber-950/30 text-amber-400 border-amber-500/30",
  mythic: "bg-red-950/30 text-red-400 border-red-500/30",
};

const statusColors: Record<string, string> = {
  active: "bg-emerald-950/30 text-emerald-400 border-emerald-500/30",
  inactive: "bg-zinc-900/50 text-zinc-400 border-zinc-800",
  pending: "bg-amber-950/30 text-amber-400 border-amber-500/30",
  completed: "bg-blue-950/30 text-blue-400 border-blue-500/30",
  failed: "bg-red-950/30 text-red-400 border-red-500/30",
};

export const GameBadge = forwardRef<HTMLSpanElement, GameBadgeProps>(
  (
    {
      variant = "default",
      size = "sm",
      rank,
      path,
      rarity,
      status,
      dot = false,
      removable = false,
      onRemove,
      className = "",
      children,
      ...props
    },
    ref
  ) => {
    let variantClass = variantClasses[variant];

    if (variant === "rank" && rank && rankColors[rank]) {
      variantClass = rankColors[rank];
    } else if (variant === "path" && path && pathColors[path]) {
      variantClass = pathColors[path];
    } else if (variant === "rarity" && rarity && rarityColors[rarity]) {
      variantClass = rarityColors[rarity];
    } else if (variant === "status" && status && statusColors[status]) {
      variantClass = statusColors[status];
    }

    const dotElement = dot && (
      <span
        className="w-1.5 h-1.5 rounded-full bg-current animate-pulse"
        aria-hidden="true"
      />
    );

    const removeButton = removable && onRemove && (
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onRemove();
        }}
        className="ml-1 p-0.5 rounded hover:bg-black/20 transition-colors"
        aria-label="Xóa"
      >
        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    );

    return (
      <span
        ref={ref}
        className={[
          baseClasses,
          variantClass,
          sizeClasses[size],
          className,
        ].join(" ")}
        {...props}
      >
        {dotElement}
        {children}
        {removeButton}
      </span>
    );
  }
);

GameBadge.displayName = "GameBadge";

interface GameBadgeGroupProps extends React.HTMLAttributes<HTMLDivElement> {
  gap?: string;
}

export const GameBadgeGroup = ({ children, className = "", gap = "gap-1.5", ...props }: GameBadgeGroupProps) => (
  <div className={["flex flex-wrap items-center", gap, className].join(" ")} {...props}>{children}</div>
);

GameBadgeGroup.displayName = "GameBadgeGroup";