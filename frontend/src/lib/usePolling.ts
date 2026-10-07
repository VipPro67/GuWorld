import { useEffect, useRef } from "react";

// Polls fn every intervalMs while `active`, but only fires when the tab is
// visible — hidden tabs produce no network traffic.
export function usePolling(fn: () => void, intervalMs: number, active = true) {
  const saved = useRef(fn);
  useEffect(() => {
    saved.current = fn;
  }, [fn]);
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => {
      if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
      saved.current();
    }, intervalMs);
    return () => clearInterval(id);
  }, [intervalMs, active]);
}
