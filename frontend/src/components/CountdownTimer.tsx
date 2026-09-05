"use client";

import React, { useState, useEffect } from "react";
import { ApertureResourceNode } from "@/store/useGameStore";

interface CountdownTimerProps {
  node: ApertureResourceNode;
  timeRate: number;
  onComplete?: () => void;
}

export const CountdownTimer: React.FC<CountdownTimerProps> = ({ node, timeRate, onComplete }) => {
  const [timeLeft, setTimeLeft] = useState<string>("");

  useEffect(() => {
    const calculateTime = () => {
      if (!node.last_collected_at) {
        setTimeLeft("Sẵn sàng");
        return;
      }
      const lastTime = new Date(node.last_collected_at).getTime();
      const rate = timeRate <= 0 ? 1.0 : timeRate;
      const nextCollectTime = lastTime + (node.production_interval_hours * 60 * 60 * 1000) / rate;
      const diff = nextCollectTime - Date.now();

      if (diff <= 0) {
        setTimeLeft("Sẵn sàng");
        if (onComplete) onComplete();
        return;
      }

      const hours = Math.floor(diff / (60 * 60 * 1000));
      const minutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
      const seconds = Math.floor((diff % (60 * 1000)) / 1000);

      const parts = [];
      if (hours > 0) parts.push(`${hours}h`);
      if (minutes > 0 || hours > 0) parts.push(`${minutes}m`);
      parts.push(`${seconds}s`);

      setTimeLeft(parts.join(" "));
    };

    calculateTime();
    const interval = setInterval(calculateTime, 1000);
    return () => clearInterval(interval);
  }, [node, timeRate]);

  return <span>{timeLeft}</span>;
};