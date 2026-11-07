interface Connection {
  isConnected(): boolean;
  onConnectionChange(listener: () => void): () => void;
}

interface HttpFallbackOptions {
  connection: Connection;
  activityTarget: EventTarget;
  visibilityTarget: EventTarget & { visibilityState: DocumentVisibilityState };
  isOnline: () => boolean;
  refresh: () => Promise<unknown>;
  intervalMs?: number;
}

/** Poll only while push updates are unavailable; never overlap slow requests. */
export function installHttpFallbackPolling({
  connection, activityTarget, visibilityTarget, isOnline, refresh, intervalMs = 5000,
}: HttpFallbackOptions): () => void {
  let stopped = false;
  let running = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const available = () => !stopped && !connection.isConnected()
    && visibilityTarget.visibilityState === "visible" && isOnline();
  const clearTimer = () => {
    clearTimeout(timer);
    timer = undefined;
  };
  const tick = async () => {
    clearTimer();
    if (running || !available()) return;
    running = true;
    try {
      await refresh();
    } catch {
      // A failed read is retried next round; it must not stop recovery.
    } finally {
      running = false;
      if (available()) timer = setTimeout(() => void tick(), intervalMs);
    }
  };
  const sync = () => {
    clearTimer();
    if (available()) void tick();
  };
  const unsubscribe = connection.onConnectionChange(sync);
  const stop = () => {
    stopped = true;
    clearTimer();
    unsubscribe();
    activityTarget.removeEventListener("online", sync);
    activityTarget.removeEventListener("offline", sync);
    activityTarget.removeEventListener("game:tab-replaced", stop);
    visibilityTarget.removeEventListener("visibilitychange", sync);
  };
  activityTarget.addEventListener("online", sync);
  activityTarget.addEventListener("offline", sync);
  activityTarget.addEventListener("game:tab-replaced", stop);
  visibilityTarget.addEventListener("visibilitychange", sync);
  sync();
  return stop;
}
