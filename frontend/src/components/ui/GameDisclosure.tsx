"use client";

import React from "react";

export interface GameDisclosureProps extends React.DetailsHTMLAttributes<HTMLDetailsElement> {
  summary?: string;
}

export const GameDisclosure = ({
  summary = "Xem chi tiết",
  className = "",
  children,
  ...props
}: GameDisclosureProps) => (
  <details className={["game-disclosure", className].join(" ")} {...props}>
    <summary className="game-disclosure-summary">
      <span>{summary}</span>
      <svg
        className="game-disclosure-chevron"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        aria-hidden="true"
      >
        <path strokeLinecap="round" strokeLinejoin="round" d="m6 9 6 6 6-6" />
      </svg>
    </summary>
    <div className="game-disclosure-content">{children}</div>
  </details>
);

GameDisclosure.displayName = "GameDisclosure";
