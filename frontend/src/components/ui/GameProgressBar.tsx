"use client";

import React, { forwardRef } from "react";

export type GameProgressVariant =
  | "default"
  | "hunger"
  | "mana"
  | "exp"
  | "cultivation"
  | "refinement"
  | "success"
  | "warning"
  | "danger"
  | "brand";

export type GameProgressSize = "xs" | "sm" | "md" | "lg";

export interface GameProgressBarProps extends React.HTMLAttributes<HTMLDivElement> {
  value: number;
  max?: number;
  variant?: GameProgressVariant;
  size?: GameProgressSize;
  showLabel?: boolean;
  label?: string;
  labelPosition?: "inside" | "outside" | "top";
  striped?: boolean;
  animated?: boolean;
  rounded?: boolean;
}

const sizeClasses: Record<GameProgressSize, string> = {
  xs: "h-1",
  sm: "h-1.5",
  md: "h-2",
  lg: "h-3",
};

const variantColors: Record<GameProgressVariant, { bg: string; track: string }> = {
  default: { bg: "bg-brand", track: "bg-zinc-800" },
  hunger: { bg: "bg-gradient-to-r from-amber-500 to-orange-500", track: "bg-zinc-800" },
  mana: { bg: "bg-gradient-to-r from-blue-500 to-indigo-500", track: "bg-zinc-800" },
  exp: { bg: "bg-gradient-to-r from-violet-500 to-purple-500", track: "bg-zinc-800" },
  cultivation: { bg: "bg-gradient-to-r from-emerald-500 to-teal-500", track: "bg-zinc-800" },
  refinement: { bg: "bg-gradient-to-r from-rose-500 to-pink-500", track: "bg-zinc-800" },
  success: { bg: "bg-success", track: "bg-emerald-900/50" },
  warning: { bg: "bg-warning", track: "bg-amber-900/50" },
  danger: { bg: "bg-error", track: "bg-red-900/50" },
  brand: { bg: "bg-brand", track: "bg-brand-dim" },
};

const hungerColor = (percentage: number): string => {
  if (percentage >= 80) return "bg-emerald-500";
  if (percentage >= 50) return "bg-amber-500";
  if (percentage >= 25) return "bg-orange-500";
  return "bg-red-500";
};

export const GameProgressBar = forwardRef<HTMLDivElement, GameProgressBarProps>(
  (
    {
      value,
      max = 100,
      variant = "default",
      size = "md",
      showLabel = false,
      label,
      labelPosition = "inside",
      striped = false,
      animated = false,
      rounded = true,
      className = "",
      children,
      ...props
    },
    ref
  ) => {
    const percentage = Math.max(0, Math.min(100, (value / max) * 100));
    const { bg, track } = variantColors[variant];
    const sizeClass = sizeClasses[size];
    const trackRounded = rounded ? "rounded-full" : "rounded-md";
    const barRounded = rounded ? "rounded-full" : "rounded-md";

    let barColor = bg;
    if (variant === "hunger") {
      barColor = hungerColor(percentage);
    }

    const labelContent = label ?? (showLabel ? `${Math.round(percentage)}%` : null);

    return (
      <div
        ref={ref}
        className={["w-full", className].join(" ")}
        {...props}
      >
        {(labelPosition === "top" || labelPosition === "outside") && labelContent && (
          <div className="flex items-center justify-between text-[10px] text-text-muted mb-1 font-mono">
            <span>{labelContent.split("/")[0]?.trim() || ""}</span>
            <span>{labelContent.includes("/") ? labelContent.split("/")[1]?.trim() : `${Math.round(percentage)}%`}</span>
          </div>
        )}
        <div
          className={[
            "relative overflow-hidden",
            track,
            sizeClass,
            trackRounded,
            striped && "bg-[linear-gradient(45deg,rgba(255,255,255,.15)_25%,transparent_25%,transparent_50%,rgba(255,255,255,.15)_50%,rgba(255,255,255,.15)_75%,transparent_75%,transparent)] bg-[size:1rem_1rem]",
            animated && "animate-[progress-stripes_1s_linear_infinite]",
          ].join(" ")}
          role="progressbar"
          aria-valuenow={Math.round(percentage)}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={label || "Progress"}
        >
          <div
            className={[
              "h-full transition-all duration-500 ease-out",
              barColor,
              barRounded,
              striped && "bg-[linear-gradient(45deg,rgba(255,255,255,.15)_25%,transparent_25%,transparent_50%,rgba(255,255,255,.15)_50%,rgba(255,255,255,.15)_75%,transparent_75%,transparent)] bg-[size:1rem_1rem]",
              animated && "animate-[progress-stripes_1s_linear_infinite]",
            ].join(" ")}
            style={{ width: `${percentage}%` }}
          >
            {(labelPosition === "inside" && labelContent && percentage > 15) && (
              <span className="absolute inset-0 flex items-center justify-center text-[9px] font-mono font-bold text-white drop-shadow-sm">
                {labelContent}
              </span>
            )}
          </div>
        </div>
        {(labelPosition === "outside" || labelPosition === "top") && labelContent && percentage < 100 && (
          <div className="absolute right-0 top-1/2 -translate-y-1/2 text-[9px] font-mono text-text-muted">
            {labelContent}
          </div>
        )}
        {children}
      </div>
    );
  }
);

GameProgressBar.displayName = "GameProgressBar";

export interface GameCircularProgressProps {
  value: number;
  max?: number;
  size?: number;
  strokeWidth?: number;
  variant?: GameProgressVariant;
  showLabel?: boolean;
  label?: string;
  className?: string;
}

export const GameCircularProgress = ({
  value,
  max = 100,
  size = 48,
  strokeWidth = 4,
  variant = "default",
  showLabel = true,
  label,
  className = "",
}: GameCircularProgressProps) => {
  const percentage = Math.max(0, Math.min(100, (value / max) * 100));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (percentage / 100) * circumference;

  const { bg } = variantColors[variant];
  const strokeColor = bg.replace("bg-", "stroke-").replace("gradient-to-r from-", "").replace(" to-", " ").replace("500", "500");

  return (
    <div className={["relative inline-flex items-center justify-center", className].join(" ")} style={{ width: size, height: size }}>
      <svg className="transform -rotate-90" width={size} height={size}>
        <circle
          className="text-zinc-800"
          strokeWidth={strokeWidth}
          stroke="currentColor"
          fill="transparent"
          r={radius}
          cx={size / 2}
          cy={size / 2}
        />
        <circle
          className={`transition-all duration-500 ease-out ${strokeColor}`}
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          strokeLinecap="round"
          fill="transparent"
          r={radius}
          cx={size / 2}
          cy={size / 2}
          style={{ filter: "drop-shadow(0 0 4px currentColor)" }}
        />
      </svg>
      {(showLabel || label) && (
        <div className="absolute inset-0 flex items-center justify-center text-center">
          <span className="font-mono font-bold text-text-primary" style={{ fontSize: size * 0.22 }}>
            {label ?? `${Math.round(percentage)}%`}
          </span>
        </div>
      )}
    </div>
  );
};

GameCircularProgress.displayName = "GameCircularProgress";

export const GameStepProgress = ({
  steps,
  currentStep,
  variant = "default",
  className = "",
}: {
  steps: { label: string; icon?: React.ReactNode; completed?: boolean }[];
  currentStep: number;
  variant?: GameProgressVariant;
  className?: string;
}) => {
  const { bg } = variantColors[variant];
  const lineColor = bg.replace("bg-", "bg-").replace("gradient-to-r from-", "").replace(" to-", " ").replace("500", "500/30");

  return (
    <div className={["relative flex items-start gap-4", className].join(" ")}>
      <div className="absolute left-[10px] top-0 bottom-0 w-0.5" style={{ background: lineColor }} />
      <div className="flex flex-col gap-4 relative z-10">
        {steps.map((step, index) => {
          const isCompleted = index < currentStep;
          const isCurrent = index === currentStep;
          const isFuture = index > currentStep;

          return (
            <div key={step.label} className="flex items-start gap-3">
              <div
                className={[
                  "flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center border-2 transition-all duration-300",
                  isCompleted
                    ? "bg-brand border-brand text-white"
                    : isCurrent
                    ? `border-2 ${bg.replace("bg-", "border-")} bg-transparent`
                    : "border-zinc-800 bg-transparent",
                ].join(" ")}
              >
                {isCompleted && (
                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                  </svg>
                )}
                {isCurrent && !isCompleted && (
                  <div className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
                )}
                {isFuture && <span className="text-[9px] font-mono text-zinc-600">{index + 1}</span>}
              </div>
              <div className="flex-1 min-w-0 pt-1">
                <p className={[
                  "text-xs font-mono truncate transition-colors",
                  isCompleted ? "text-text-primary" : isCurrent ? "text-text-primary font-bold" : "text-text-muted",
                ].join(" ")}>
                  {step.label}
                </p>
                {step.icon && <div className="text-[9px] text-text-muted mt-0.5">{step.icon}</div>}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

GameStepProgress.displayName = "GameStepProgress";