"use client";

import { useState } from "react";
import { formatNumber } from "@/lib/utils";

interface NumProps {
  value: number;
  decimals?: number;
  className?: string;
}

export function Num({ value, decimals = 2, className }: NumProps) {
  const [showFull, setShowFull] = useState(false);
  const safeValue = Number.isFinite(value) ? value : 0;
  const fullText = safeValue.toLocaleString("vi-VN", { maximumFractionDigits: Math.max(0, Math.min(4, decimals)) });
  const display = showFull ? fullText : formatNumber(safeValue, decimals);

  return (
    <span
      className={`cursor-pointer ${className ?? ""}`}
      title={fullText}
      onClick={(e) => {
        e.stopPropagation();
        setShowFull(!showFull);
      }}
    >
      {display}
    </span>
  );
}
