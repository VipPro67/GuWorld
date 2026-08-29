"use client";

import React, { forwardRef } from "react";

export interface GamePageProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  fullWidth?: boolean;
}

export const GamePage = forwardRef<HTMLDivElement, GamePageProps>(
  (
    {
      title,
      description,
      icon,
      action,
      className = "",
      fullWidth = false,
      children,
      ...props
    },
    ref
  ) => (
    <div
      ref={ref}
      className={[
        "min-h-screen bg-bg-primary",
        fullWidth ? "w-full px-4 sm:px-6 lg:px-8 py-6" : "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6",
        className,
      ].join(" ")}
      {...props}
    >
      {(title || action) && (
        <header className="mb-6 sm:mb-8">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-3 min-w-0">
              {icon && (
                <div className="flex-shrink-0 w-12 h-12 rounded-2xl bg-brand/10 flex items-center justify-center text-brand text-2xl">
                  {icon}
                </div>
              )}
              <div className="min-w-0">
                {title && (
                  <h1 className="text-2xl sm:text-3xl font-bold font-serif text-text-primary">
                    {title}
                  </h1>
                )}
                {description && (
                  <p className="mt-2 text-sm sm:text-base text-text-muted max-w-2xl">
                    {description}
                  </p>
                )}
              </div>
            </div>
            {action && <div className="flex-shrink-0 mt-1">{action}</div>}
          </div>
        </header>
      )}
      <main className="space-y-6">{children}</main>
    </div>
  )
);

GamePage.displayName = "GamePage";

export const GamePageHeader = ({
  title,
  description,
  icon,
  action,
  className = "",
}: Omit<GamePageProps, "children" | "fullWidth">) => (
  <header className={["mb-6 sm:mb-8", className].join(" ")}>
    <div className="flex items-start justify-between gap-4">
      <div className="flex items-start gap-3 min-w-0">
        {icon && (
          <div className="flex-shrink-0 w-12 h-12 rounded-2xl bg-brand/10 flex items-center justify-center text-brand text-2xl">
            {icon}
          </div>
        )}
        <div className="min-w-0">
          {title && (
            <h1 className="text-2xl sm:text-3xl font-bold font-serif text-text-primary">
              {title}
            </h1>
          )}
          {description && (
            <p className="mt-2 text-sm sm:text-base text-text-muted max-w-2xl">
              {description}
            </p>
          )}
        </div>
      </div>
      {action && <div className="flex-shrink-0 mt-1">{action}</div>}
    </div>
  </header>
);

GamePageHeader.displayName = "GamePageHeader";

export const GamePageContent = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className = "", children, ...props }, ref) => (
    <main ref={ref} className={["space-y-6", className].join(" ")} {...props}>
      {children}
    </main>
  )
);

GamePageContent.displayName = "GamePageContent";

export const GamePageGrid = ({
  children,
  columns = { base: 1, sm: 2, lg: 3, xl: 4 },
  gap = "md",
  className = "",
}: {
  children: React.ReactNode;
  columns?: { base?: number; sm?: number; md?: number; lg?: number; xl?: number };
  gap?: "sm" | "md" | "lg" | "xl";
  className?: string;
}) => {
  const colClasses = [];
  if (columns.base) colClasses.push(`grid-cols-${columns.base}`);
  if (columns.sm) colClasses.push(`sm:grid-cols-${columns.sm}`);
  if (columns.md) colClasses.push(`md:grid-cols-${columns.md}`);
  if (columns.lg) colClasses.push(`lg:grid-cols-${columns.lg}`);
  if (columns.xl) colClasses.push(`xl:grid-cols-${columns.xl}`);

  const gapClass = {
    sm: "gap-3",
    md: "gap-4",
    lg: "gap-6",
    xl: "gap-8",
  }[gap];

  return (
    <div className={["grid", ...colClasses, gapClass, className].join(" ")}>
      {children}
    </div>
  );
};

GamePageGrid.displayName = "GamePageGrid";

export const GamePageSection = ({
  title,
  description,
  icon,
  action,
  children,
  className = "",
}: Omit<GamePageProps, "fullWidth"> & { children: React.ReactNode }) => (
  <section className={["space-y-4", className].join(" ")}>
    {(title || action) && (
      <div className="flex items-start justify-between gap-4 mb-4">
        <div className="flex items-start gap-3 min-w-0">
          {icon && (
            <div className="flex-shrink-0 w-10 h-10 rounded-xl bg-brand/10 flex items-center justify-center text-brand">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            {title && <h2 className="text-xl font-bold font-serif text-text-primary">{title}</h2>}
            {description && <p className="mt-1 text-sm text-text-muted">{description}</p>}
          </div>
        </div>
        {action && <div className="flex-shrink-0">{action}</div>}
      </div>
    )}
    <div className="space-y-4">{children}</div>
  </section>
);

GamePageSection.displayName = "GamePageSection";