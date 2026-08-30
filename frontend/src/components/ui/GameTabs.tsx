"use client";

import React, { useRef, useEffect, useCallback } from "react";

export interface GameTabItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  count?: number;
  disabled?: boolean;
  badge?: React.ReactNode;
}

export type GameTabsVariant = "pills" | "underline" | "segmented" | "minimal";

export interface GameTabsProps {
  items: GameTabItem[];
  activeId: string;
  onChange: (id: string) => void;
  variant?: GameTabsVariant;
  size?: "sm" | "md" | "lg";
  fullWidth?: boolean;
  className?: string;
  scrollable?: boolean;
}

const variantTabClasses: Record<GameTabsVariant, { base: string; active: string; inactive: string }> = {
  pills: {
    base: "rounded-xl transition-all duration-200",
    active: "bg-surface text-text-primary shadow-md shadow-brand/10 border border-brand/20",
    inactive: "text-text-muted hover:text-text-secondary hover:bg-surface/30",
  },
  underline: {
    base: "relative pb-2 transition-colors duration-200",
    active: "text-text-primary font-bold",
    inactive: "text-text-muted hover:text-text-secondary",
  },
  segmented: {
    base: "rounded-xl transition-all duration-200",
    active: "bg-surface text-text-primary shadow-sm",
    inactive: "text-text-muted hover:text-text-secondary hover:bg-surface/30",
  },
  minimal: {
    base: "transition-colors duration-200",
    active: "text-text-primary font-bold",
    inactive: "text-text-muted hover:text-text-secondary",
  },
};

const sizeClasses = {
  sm: "px-3 py-1.5 text-[10px] gap-1",
  md: "px-4 py-2 text-xs gap-1.5",
  lg: "px-5 py-2.5 text-sm gap-2",
};

export const GameTabs = React.forwardRef<HTMLDivElement, GameTabsProps>(
  (
    {
      items,
      activeId,
      onChange,
      variant = "pills",
      size = "md",
      fullWidth = false,
      className = "",
      scrollable = true,
      ...props
    },
    ref
  ) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const scrollContainerRef = useRef<HTMLDivElement>(null);

    const scrollToActive = useCallback(() => {
      if (!scrollContainerRef.current || !containerRef.current) return;
      const activeTab = containerRef.current.querySelector(`[data-tab-id="${activeId}"]`) as HTMLElement;
      if (activeTab) {
        const container = scrollContainerRef.current;
        const offsetLeft = activeTab.offsetLeft;
        const offsetWidth = activeTab.offsetWidth;
        const containerWidth = container.clientWidth;
        const scrollLeft = container.scrollLeft;
        if (offsetLeft < scrollLeft || offsetLeft + offsetWidth > scrollLeft + containerWidth) {
          container.scrollTo({
            left: offsetLeft - (containerWidth - offsetWidth) / 2,
            behavior: "smooth",
          });
        }
      }
    }, [activeId]);

    useEffect(() => {
      scrollToActive();
    }, [scrollToActive]);

    const { base, active, inactive } = variantTabClasses[variant];
    const sizeClass = sizeClasses[size];

    return (
      <div
        ref={ref}
        className={[
          "w-full",
          variant === "underline" ? "" : (fullWidth ? "w-full" : "inline-flex"),
          className,
        ].join(" ")}
        {...props}
      >
        <div
          ref={(el) => {
            if (el) {
              scrollContainerRef.current = el;
              containerRef.current = el;
            }
          }}
          className={[
            "flex items-center",
            variant === "pills" || variant === "segmented" ? "gap-1.5" : "gap-1",
            fullWidth && (variant === "pills" || variant === "segmented") ? "w-full" : "",
            scrollable ? "overflow-x-auto scrollbar-hide pb-1 -mb-1" : "",
          ].join(" ")}
          role="tablist"
          aria-label="Tabs"
        >
          {items.map((item) => {
            const isActive = item.id === activeId;
            const isDisabled = item.disabled || false;

            return (
              <button
                key={item.id}
                data-tab-id={item.id}
                type="button"
                role="tab"
                aria-selected={isActive}
                aria-disabled={isDisabled}
                disabled={isDisabled}
                onClick={() => !isDisabled && onChange(item.id)}
                className={[
                  "flex items-center justify-center gap-1.5",
                  "font-mono font-semibold",
                  "whitespace-nowrap shrink-0",
                  "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:ring-offset-2 focus:ring-offset-bg-primary",
                  base,
                  sizeClass,
                  isActive ? active : inactive,
                  isDisabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
                  variant === "underline" && isActive ? "after:absolute after:bottom-0 after:left-0 after:right-0 after:h-1 after:bg-brand after:rounded-full" : "",
                  variant === "underline" ? "border-b-2 border-transparent" : "",
                ].join(" ")}
              >
                {item.icon && <span className="flex-shrink-0" aria-hidden="true">{item.icon}</span>}
                <span>{item.label}</span>
                {item.count !== undefined && item.count > 0 && (
                  <span className="text-[9px] font-mono bg-brand/20 text-brand px-1.5 rounded-full min-w-[1.25rem] text-center">
                    {item.count > 99 ? "99+" : item.count}
                  </span>
                )}
                {item.badge && <span className="flex-shrink-0">{item.badge}</span>}
              </button>
            );
          })}
        </div>
      </div>
    );
  }
);

GameTabs.displayName = "GameTabs";

export interface GameTabPanelProps {
  id: string;
  activeId: string;
  children: React.ReactNode;
  className?: string;
  animate?: boolean;
}

export const GameTabPanel = ({
  id,
  activeId,
  children,
  className = "",
  animate = true,
}: GameTabPanelProps) => {
  const isActive = id === activeId;

  if (!isActive) return null;

  return (
    <div
      role="tabpanel"
      aria-labelledby={`tab-${id}`}
      className={[
        animate ? "animate-in fade-in slide-in-from-bottom-2 duration-300" : "",
        className,
      ].join(" ")}
    >
      {children}
    </div>
  );
};

GameTabPanel.displayName = "GameTabPanel";