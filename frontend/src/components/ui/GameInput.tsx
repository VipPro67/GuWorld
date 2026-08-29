"use client";

import React, { forwardRef, useId } from "react";

export interface GameInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> {
  label?: string;
  error?: string;
  hint?: string;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  leftAddon?: React.ReactNode;
  rightAddon?: React.ReactNode;
  variant?: "default" | "search" | "compact";
  size?: "sm" | "md" | "lg";
  fullWidth?: boolean;
}

const sizeClasses = {
  sm: "text-[10px] px-3 py-1.5",
  md: "text-xs px-3.5 py-2",
  lg: "text-sm px-4 py-2.5",
};

const variantClasses = {
  default: "",
  search: "pl-10",
  compact: "px-2 py-1",
};

const inputBaseClasses = [
  "w-full bg-zinc-900/50",
  "border border-zinc-800",
  "text-text-primary",
  "placeholder:text-zinc-600",
  "rounded-xl",
  "transition-all duration-200",
  "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:border-brand focus:ring-offset-0",
  "disabled:opacity-50 disabled:cursor-not-allowed",
  "hover:border-zinc-700",
].join(" ");

export const GameInput = forwardRef<HTMLInputElement, GameInputProps>(
  (
    {
      label,
      error,
      hint,
      leftIcon,
      rightIcon,
      leftAddon,
      rightAddon,
      variant = "default",
      size = "md",
      fullWidth = true,
      className = "",
      id: providedId,
      disabled,
      required,
      ...props
    },
    ref
  ) => {
    const generatedId = useId();
    const id = providedId || generatedId;
    const errorId = `${id}-error`;
    const hintId = `${id}-hint`;
    const describedBy = [error && errorId, hint && hintId].filter(Boolean).join(" ") || undefined;

    return (
      <div className={["w-full", fullWidth ? "" : "w-auto", className].join(" ")}>
        {label && (
          <label
            htmlFor={id}
            className="block text-[10px] font-mono font-medium text-text-secondary mb-1.5"
          >
            {label}
            {required && <span className="text-error ml-1" aria-hidden="true">*</span>}
          </label>
        )}
        <div className="relative">
          {(leftIcon || leftAddon) && (
            <div
              className="absolute inset-y-0 left-0 flex items-center pl-3 text-zinc-500 pointer-events-none"
              aria-hidden="true"
            >
              {leftAddon || leftIcon}
            </div>
          )}
          {(rightIcon || rightAddon) && (
            <div
              className="absolute inset-y-0 right-0 flex items-center pr-3 text-zinc-500 pointer-events-none"
              aria-hidden="true"
            >
              {rightAddon || rightIcon}
            </div>
          )}
          <input
            ref={ref}
            id={id}
            className={[
              inputBaseClasses,
              sizeClasses[size],
              variantClasses[variant],
              (leftIcon || leftAddon) && "pl-10",
              (rightIcon || rightAddon) && "pr-10",
              error && "border-error focus:ring-error/50 focus:border-error",
            ]
              .filter(Boolean)
              .join(" ")}
            aria-invalid={error ? "true" : "false"}
            aria-describedby={describedBy}
            aria-required={required}
            disabled={disabled}
            {...props}
          />
        </div>
        {error && (
          <p id={errorId} className="mt-1.5 text-[10px] text-error flex items-center gap-1" role="alert">
            <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            {error}
          </p>
        )}
        {hint && !error && (
          <p id={hintId} className="mt-1.5 text-[10px] text-zinc-500">
            {hint}
          </p>
        )}
      </div>
    );
  }
);

GameInput.displayName = "GameInput";

export interface GameTextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  hint?: string;
  size?: "sm" | "md" | "lg";
  fullWidth?: boolean;
  minRows?: number;
}

export const GameTextarea = forwardRef<HTMLTextAreaElement, GameTextareaProps>(
  (
    {
      label,
      error,
      hint,
      size = "md",
      fullWidth = true,
      minRows = 3,
      className = "",
      id: providedId,
      disabled,
      required,
      ...props
    },
    ref
  ) => {
    const generatedId = useId();
    const id = providedId || generatedId;
    const errorId = `${id}-error`;
    const hintId = `${id}-hint`;
    const describedBy = [error && errorId, hint && hintId].filter(Boolean).join(" ") || undefined;

    const sizeStyles = {
      sm: "text-[10px] px-3 py-2",
      md: "text-xs px-3.5 py-2.5",
      lg: "text-sm px-4 py-3",
    };

    return (
      <div className={["w-full", fullWidth ? "" : "w-auto", className].join(" ")}>
        {label && (
          <label
            htmlFor={id}
            className="block text-[10px] font-mono font-medium text-text-secondary mb-1.5"
          >
            {label}
            {required && <span className="text-error ml-1" aria-hidden="true">*</span>}
          </label>
        )}
        <textarea
          ref={ref}
          id={id}
          className={[
            "w-full bg-zinc-900/50",
            "border border-zinc-800",
            "text-text-primary",
            "placeholder:text-zinc-600",
            "rounded-xl",
            "transition-all duration-200",
            "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:border-brand focus:ring-offset-0",
            "disabled:opacity-50 disabled:cursor-not-allowed",
            "hover:border-zinc-700",
            "resize-y",
            sizeStyles[size],
            error && "border-error focus:ring-error/50 focus:border-error",
          ]
              .filter(Boolean)
              .join(" ")}
        aria-invalid={error ? "true" : "false"}
        aria-describedby={describedBy}
        aria-required={required}
        disabled={disabled}
        rows={minRows}
        {...props}
        />
        {error && (
          <p id={errorId} className="mt-1.5 text-[10px] text-error flex items-center gap-1" role="alert">
            <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            {error}
          </p>
        )}
        {hint && !error && (
          <p id={hintId} className="mt-1.5 text-[10px] text-zinc-500">
            {hint}
          </p>
        )}
      </div>
    );
  }
);

GameTextarea.displayName = "GameTextarea";

export interface GameSelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  hint?: string;
  options: { value: string; label: string; disabled?: boolean }[];
  placeholder?: string;
  sizeVariant?: "sm" | "md" | "lg";
  fullWidth?: boolean;
}

export const GameSelect = forwardRef<HTMLSelectElement, GameSelectProps>(
  (
    {
      label,
      error,
      hint,
      options,
      placeholder,
      sizeVariant = "md",
      fullWidth = true,
      className = "",
      id: providedId,
      disabled,
      required,
      ...props
    },
    ref
  ) => {
    const generatedId = useId();
    const id = providedId || generatedId;
    const errorId = `${id}-error`;
    const hintId = `${id}-hint`;
    const describedBy = [error && errorId, hint && hintId].filter(Boolean).join(" ") || undefined;

    const sizeStyles = {
      sm: "text-[10px] px-3 py-1.5",
      md: "text-xs px-3.5 py-2",
      lg: "text-sm px-4 py-2.5",
    };

    return (
      <div className={["w-full", fullWidth ? "" : "w-auto", className].join(" ")}>
        {label && (
          <label
            htmlFor={id}
            className="block text-[10px] font-mono font-medium text-text-secondary mb-1.5"
          >
            {label}
            {required && <span className="text-error ml-1" aria-hidden="true">*</span>}
          </label>
        )}
        <select
          ref={ref}
          id={id}
          className={[
            "w-full bg-zinc-900/50",
            "border border-zinc-800",
            "text-text-primary",
            "rounded-xl",
            "transition-all duration-200",
            "focus:outline-none focus:ring-2 focus:ring-brand/50 focus:border-brand focus:ring-offset-0",
            "disabled:opacity-50 disabled:cursor-not-allowed",
            "hover:border-zinc-700",
            "appearance-none",
            "bg-[url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke='%2371717a'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19 9l-7 7-7-7'%3E%3C/path%3E%3C/svg%3E\")] bg-[right_0.75rem_center] bg-no-repeat pr-10",
            sizeStyles[sizeVariant],
            error && "border-error focus:ring-error/50 focus:border-error",
          ]
              .filter(Boolean)
              .join(" ")}
          aria-invalid={error ? "true" : "false"}
          aria-describedby={describedBy}
          aria-required={required}
          disabled={disabled}
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((opt) => (
            <option key={opt.value} value={opt.value} disabled={opt.disabled}>
              {opt.label}
            </option>
          ))}
        </select>
        {error && (
          <p id={errorId} className="mt-1.5 text-[10px] text-error flex items-center gap-1" role="alert">
            <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            {error}
          </p>
        )}
        {hint && !error && (
          <p id={hintId} className="mt-1.5 text-[10px] text-zinc-500">
            {hint}
          </p>
        )}
      </div>
    );
  }
);

GameSelect.displayName = "GameSelect";

export interface GameCheckboxProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  description?: string;
}

export const GameCheckbox = forwardRef<HTMLInputElement, GameCheckboxProps>(
  ({ label, description, className = "", id: providedId, ...props }, ref) => {
    const generatedId = useId();
    const id = providedId || generatedId;

    return (
      <label className={["flex items-start gap-3 cursor-pointer", className].join(" ")} htmlFor={id}>
        <input
          ref={ref}
          type="checkbox"
          id={id}
          className={[
            "mt-0.5 w-4 h-4",
            "rounded border-zinc-700",
            "text-brand",
            "focus:ring-2 focus:ring-brand/50 focus:ring-offset-2 focus:ring-offset-bg-primary",
            "disabled:opacity-50 disabled:cursor-not-allowed",
            "transition-colors",
            "checked:bg-brand checked:border-brand",
          ].join(" ")}
          {...props}
        />
        <div className="min-w-0">
          <span className="block text-xs font-medium text-text-primary">{label}</span>
          {description && <span className="block text-[10px] text-text-muted mt-0.5">{description}</span>}
        </div>
      </label>
    );
  }
);

GameCheckbox.displayName = "GameCheckbox";

export interface GameRadioGroupProps {
  label?: string;
  options: { value: string; label: string; description?: string; disabled?: boolean }[];
  value: string;
  onChange: (value: string) => void;
  orientation?: "horizontal" | "vertical";
  className?: string;
}

export const GameRadioGroup = ({
  label,
  options,
  value,
  onChange,
  orientation = "vertical",
  className = "",
}: GameRadioGroupProps) => {
  const groupId = useId();

  return (
    <fieldset className={className}>
      {label && (
        <legend className="text-[10px] font-mono font-medium text-text-secondary mb-2">
          {label}
        </legend>
      )}
      <div
        className={[
          "flex gap-3",
          orientation === "horizontal" ? "flex-wrap" : "flex-col",
        ].join(" ")}
        role="radiogroup"
        aria-labelledby={label ? groupId : undefined}
      >
        {options.map((opt) => (
          <label
            key={opt.value}
            className={[
              "flex items-center gap-2 cursor-pointer",
              "p-3 rounded-xl border transition-all",
              "bg-zinc-900/50",
              value === opt.value
                ? "border-brand/50 bg-brand-dim/30"
                : "border-zinc-800/50 hover:border-zinc-700",
              opt.disabled && "opacity-50 cursor-not-allowed",
            ].join(" ")}
          >
            <input
              type="radio"
              name={groupId}
              value={opt.value}
              checked={value === opt.value}
              onChange={() => !opt.disabled && onChange(opt.value)}
              disabled={opt.disabled}
              className={[
                "w-4 h-4",
                "text-brand",
                "focus:ring-2 focus:ring-brand/50 focus:ring-offset-2 focus:ring-offset-bg-primary",
                "disabled:opacity-50",
              ].join(" ")}
            />
            <div className="min-w-0">
              <span className="block text-xs font-medium text-text-primary">{opt.label}</span>
              {opt.description && (
                <span className="block text-[10px] text-text-muted mt-0.5">{opt.description}</span>
              )}
            </div>
          </label>
        ))}
      </div>
    </fieldset>
  );
};

GameRadioGroup.displayName = "GameRadioGroup";