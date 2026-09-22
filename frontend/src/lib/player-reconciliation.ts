export interface VisibilityTarget extends EventTarget {
  visibilityState: DocumentVisibilityState;
}

export function installPlayerLifecycleReconciliation(
  onlineTarget: EventTarget,
  visibilityTarget: VisibilityTarget,
  reconcile: () => void,
): () => void {
  const onVisible = () => {
    if (visibilityTarget.visibilityState === "visible") reconcile();
  };
  onlineTarget.addEventListener("online", reconcile);
  visibilityTarget.addEventListener("visibilitychange", onVisible);
  return () => {
    onlineTarget.removeEventListener("online", reconcile);
    visibilityTarget.removeEventListener("visibilitychange", onVisible);
  };
}

export function shouldApplyReconciliation(
  requestRevision: number,
  latestRequestRevision: number,
  serverVersion: number,
  currentVersion: number,
): boolean {
  return requestRevision === latestRequestRevision && serverVersion >= currentVersion;
}

export function shouldRefreshCollections(
  forceCollections: boolean,
  clientVersion: number,
  serverVersion: number,
): boolean {
  return forceCollections || serverVersion > clientVersion;
}

export class LatestRequestGate {
  private revision = 0;

  next(): number {
    return ++this.revision;
  }

  isCurrent(revision: number): boolean {
    return revision === this.revision;
  }

  invalidate(): void {
    this.revision++;
  }
}
