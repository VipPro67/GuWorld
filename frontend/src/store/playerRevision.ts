export type RevisionedState<T> = { revision: number | null; value: T };

export function shouldApplyPlayerRevision(
  appliedRevision: number | null,
  incomingRevision: number | undefined,
  options?: { allowEqual?: boolean },
): boolean {
  if (incomingRevision === undefined) return true;
  if (appliedRevision === null) return true;
  return options?.allowEqual
    ? incomingRevision >= appliedRevision
    : incomingRevision > appliedRevision;
}

export function applyRevisionedState<T>(
  current: RevisionedState<T>,
  incomingRevision: number,
  value: T,
): RevisionedState<T> {
  return shouldApplyPlayerRevision(current.revision, incomingRevision)
    ? { revision: incomingRevision, value }
    : current;
}
