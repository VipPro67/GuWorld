import assert from "node:assert/strict";
import { describe, it, mock } from "node:test";
import {
  installPlayerLifecycleReconciliation,
  shouldApplyReconciliation,
  shouldRefreshCollections,
  LatestRequestGate,
  type VisibilityTarget,
} from "./player-reconciliation.ts";

class FakeVisibilityTarget extends EventTarget implements VisibilityTarget {
  visibilityState: DocumentVisibilityState = "hidden";
}

describe("player lifecycle reconciliation", () => {
  it("refreshes collections after an offline mutation and rejects the older response", () => {
    const onlineTarget = new EventTarget();
    const visibilityTarget = new FakeVisibilityTarget();
    const reconcile = mock.fn();
    const cleanup = installPlayerLifecycleReconciliation(onlineTarget, visibilityTarget, reconcile);

    onlineTarget.dispatchEvent(new Event("online"));

    assert.equal(reconcile.mock.callCount(), 1);
    assert.equal(shouldRefreshCollections(false, 7, 8), true);
    assert.equal(shouldApplyReconciliation(1, 2, 7, 8), false);
    cleanup();
  });

  it("force-refreshes when a long-hidden tab returns to the foreground", () => {
    const onlineTarget = new EventTarget();
    const visibilityTarget = new FakeVisibilityTarget();
    const reconcile = mock.fn();
    const cleanup = installPlayerLifecycleReconciliation(onlineTarget, visibilityTarget, reconcile);

    visibilityTarget.dispatchEvent(new Event("visibilitychange"));
    assert.equal(reconcile.mock.callCount(), 0);

    visibilityTarget.visibilityState = "visible";
    visibilityTarget.dispatchEvent(new Event("visibilitychange"));

    assert.equal(reconcile.mock.callCount(), 1);
    assert.equal(shouldRefreshCollections(true, 12, 12), true);
    cleanup();
  });

  it("does not let a slow pre-offline response overwrite the foreground refresh", async () => {
    const gate = new LatestRequestGate();
    const applied: string[] = [];
    let releaseOld!: () => void;
    const oldBlocked = new Promise<void>((resolve) => { releaseOld = resolve; });

    const oldRevision = gate.next();
    const oldRequest = oldBlocked.then(() => {
      if (gate.isCurrent(oldRevision)) applied.push("old");
    });

    const foregroundRevision = gate.next();
    if (gate.isCurrent(foregroundRevision)) applied.push("foreground");
    releaseOld();
    await oldRequest;

    assert.deepEqual(applied, ["foreground"]);
  });
});
