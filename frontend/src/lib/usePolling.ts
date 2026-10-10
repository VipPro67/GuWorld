import { useEffect, useRef } from "react";

// Polls fn every intervalMs while `active`, but only fires when the tab is
// visible — hidden tabs produce no network traffic. Async polls are single-flight
// so a slow request cannot accumulate overlapping copies of itself.
export function usePolling(fn: () => void | Promise<void>, intervalMs: number, active = true) {
  const saved = useRef(fn);
  const running = useRef(false);
  useEffect(() => {
    saved.current = fn;
  }, [fn]);
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => {
      if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
      if (running.current) return;
      running.current = true;
      void Promise.resolve()
        .then(() => saved.current())
        .catch((err: unknown) => console.error("Polling callback failed", err))
        .finally(() => {
          running.current = false;
        });
    }, intervalMs);
    return () => clearInterval(id);
  }, [intervalMs, active]);
}
