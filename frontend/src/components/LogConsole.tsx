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
    <div className="bg-[var(--bg-surface)] border border-[var(--border-default)] rounded-2xl sm:rounded-3xl p-3 sm:p-6 w-full flex flex-col h-[220px] sm:h-[280px] shadow-xl">
      <div className="flex items-center gap-2 border-b border-[var(--border-default)] pb-2 mb-3 sm:pb-3 sm:mb-4">
        <Terminal className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
        <span className="text-xs font-mono uppercase tracking-widest text-[var(--text-secondary)] font-semibold">
          Ký Sự Tu Hành
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
              style={{ whiteSpace: "pre-line" }}
              className={`p-2 rounded-lg transition-colors border leading-relaxed ${
                isNewest
                  ? isError
                    ? "bg-rose-100 dark:bg-rose-950 border-rose-300 dark:border-rose-800 text-rose-800 dark:text-rose-300 font-semibold"
                    : "bg-emerald-100 dark:bg-emerald-950 border-emerald-300 dark:border-emerald-800 text-emerald-800 dark:text-emerald-300 font-semibold"
                  : isError
                  ? "border-transparent text-rose-700 dark:text-rose-400"
                  : "border-transparent text-[var(--text-secondary)]"
              }`}
            >
              <span className="text-[var(--text-muted)] mr-2 select-none">&gt;&gt;</span>
              {log}
            </div>
          );
        })}
      </div>
    </div>
  );
};
