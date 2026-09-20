"use client";

import { useState } from "react";
import { formatNumber } from "@/lib/utils";

interface NumProps {
  value: number;
  className?: string;
}

export function Num({ value, className }: NumProps) {
  const [showFull, setShowFull] = useState(false);
  const display = showFull ? value.toLocaleString("en-US") : formatNumber(value);

  return (
    <span
      className={`cursor-pointer ${className ?? ""}`}
      title={value.toLocaleString("en-US")}
      onClick={(e) => {
        e.stopPropagation();
        setShowFull(!showFull);
      }}
    >
      {display}
    </span>
  );
}
