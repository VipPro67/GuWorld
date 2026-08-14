import { useState, useEffect, useRef } from "react";
import { useWorldStore } from "@/store/useWorldStore";

export function WorldLogViewer() {
  const logs = useWorldStore((s) => s.worldLogs);
  const [isOpen, setIsOpen] = useState(true);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen && endRef.current) {
      endRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs, isOpen]);

  if (!isOpen) {
    return (
      <div className="fixed bottom-4 left-4 z-50">
        <button
          onClick={() => setIsOpen(true)}
          className="bg-zinc-900/80 border border-zinc-700/50 text-zinc-300 px-3 py-1.5 rounded-lg text-sm shadow-lg backdrop-blur-sm hover:bg-zinc-800 transition-colors flex items-center gap-2"
        >
          <span>📋</span> Nhật ký hành trình
        </button>
      </div>
    );
  }

  return (
    <div className="fixed bottom-4 left-4 z-50 w-80 sm:w-96 bg-zinc-950/80 border border-zinc-800/80 rounded-xl shadow-2xl backdrop-blur-md overflow-hidden flex flex-col pointer-events-auto">
      <div className="flex items-center justify-between px-3 py-2 bg-zinc-900/50 border-b border-zinc-800/80">
        <span className="text-sm font-medium text-zinc-300 flex items-center gap-2">
          <span>📋</span> Nhật ký hành trình
        </span>
        <button
          onClick={() => setIsOpen(false)}
          className="text-zinc-500 hover:text-zinc-300 transition-colors p-1"
        >
          ✕
        </button>
      </div>
      <div className="p-3 max-h-60 overflow-y-auto text-xs sm:text-sm text-zinc-400 space-y-2 font-mono">
        {logs.length === 0 ? (
          <div className="text-center italic text-zinc-600 py-2">Chưa có bản ghi nào...</div>
        ) : (
          logs.map((log) => (
            <div key={log.id} className="border-l-2 border-emerald-500/30 pl-2 py-0.5 whitespace-pre-wrap">
              {log.text}
            </div>
          ))
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}
