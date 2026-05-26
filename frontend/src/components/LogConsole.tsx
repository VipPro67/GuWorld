"use client";

import React, { useEffect, useRef } from "react";
import { Terminal } from "lucide-react";

interface LogConsoleProps {
  logs: string[];
}

export const LogConsole: React.FC<LogConsoleProps> = ({ logs }) => {
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to top when a new log arrives (since logs are pre-pended, wait, useGameStore prepends them: state.logs = [message, ...state.logs]. So index 0 is the newest!
  // Since index 0 is the newest, we don't need to auto-scroll to bottom, the newest is always at the top! This is a great design.
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = 0;
    }
  }, [logs]);

  return (
    <div className="glass-panel rounded-3xl p-6 w-full max-w-lg flex flex-col h-[280px]">
      <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-3 mb-4">
        <Terminal className="w-4 h-4 text-emerald-400" />
        <span className="text-xs font-mono uppercase tracking-widest text-zinc-400 font-semibold">
          Ký Sự Tu Hành (Cultivation Log)
        </span>
      </div>

      <div
        ref={containerRef}
        className="flex-1 overflow-y-auto custom-scrollbar space-y-2.5 font-mono text-xs pr-1"
      >
        {logs.map((log, index) => {
          // If it is the first/newest item, highlight it
          const isNewest = index === 0;
          const isError = log.includes("failed") || log.includes("cạn kiệt") || log.includes("Thất bại");

          return (
            <div
              key={index}
              className={`p-2 rounded-lg transition-colors border leading-relaxed ${
                isNewest
                  ? isError
                    ? "bg-rose-950/20 border-rose-900/30 text-rose-300 font-semibold"
                    : "bg-emerald-950/20 border-emerald-900/30 text-emerald-300 font-semibold shadow-[0_0_10px_rgba(16,185,129,0.05)]"
                  : isError
                  ? "border-transparent text-rose-400/80"
                  : "border-transparent text-zinc-400"
              }`}
            >
              <span className="text-zinc-600 mr-2 select-none">&gt;&gt;</span>
              {log}
            </div>
          );
        })}
      </div>
    </div>
  );
};
