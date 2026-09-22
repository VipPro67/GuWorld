export type PendingActions = Record<string, true>;

export function acquirePendingAction(pending: PendingActions, key: string): PendingActions | null {
  if (pending[key]) return null;
  return { ...pending, [key]: true };
}

export function releasePendingAction(pending: PendingActions, key: string): PendingActions {
  if (!pending[key]) return pending;
  const next = { ...pending };
  delete next[key];
  return next;
}
