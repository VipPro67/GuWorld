import { useCallback, useRef } from "react";

export function useTribulationCombatResolution(
  eventId: string | undefined,
  isPracticeMode: boolean | undefined,
  resolveTribulation: (eventId: string) => Promise<boolean>,
) {
  const resolvedEventRef = useRef<string | null>(null);

  return useCallback(async (winner: string | null | undefined, sessionEventId?: string) => {
    const resolvedEventId = sessionEventId || eventId;
    if (winner !== "Phe Ta" || !resolvedEventId || isPracticeMode || resolvedEventRef.current === resolvedEventId) return;
    resolvedEventRef.current = resolvedEventId;
    await resolveTribulation(resolvedEventId);
  }, [eventId, isPracticeMode, resolveTribulation]);
}
