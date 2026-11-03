import assert from "node:assert/strict";
import test from "node:test";
import { installHttpFallbackPolling } from "./http-fallback.ts";

class Connection {
  connected = false;
  listeners = new Set<() => void>();
  isConnected() { return this.connected; }
  onConnectionChange(listener: () => void) {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  }
  setConnected(connected: boolean) {
    this.connected = connected;
    this.listeners.forEach((listener) => listener());
  }
}
class Visibility extends EventTarget {
  visibilityState: DocumentVisibilityState = "visible";
}
const flush = async () => { await Promise.resolve(); await Promise.resolve(); };

test("HTTP polling starts on loss and stops on recovery, hidden/offline tabs and cleanup", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const connection = new Connection();
  connection.connected = true;
  const activityTarget = new EventTarget();
  const visibilityTarget = new Visibility();
  let online = true;
  let reads = 0;
  const stop = installHttpFallbackPolling({
    connection, activityTarget, visibilityTarget, isOnline: () => online,
    refresh: async () => { reads++; },
  });
  assert.equal(reads, 0);
  connection.setConnected(false);
  await flush();
  assert.equal(reads, 1);
  t.mock.timers.tick(5000);
  await flush();
  assert.equal(reads, 2);
  visibilityTarget.visibilityState = "hidden";
  visibilityTarget.dispatchEvent(new Event("visibilitychange"));
  t.mock.timers.tick(20000);
  assert.equal(reads, 2);
  visibilityTarget.visibilityState = "visible";
  visibilityTarget.dispatchEvent(new Event("visibilitychange"));
  await flush();
  assert.equal(reads, 3);
  online = false;
  activityTarget.dispatchEvent(new Event("offline"));
  t.mock.timers.tick(20000);
  assert.equal(reads, 3);
  online = true;
  activityTarget.dispatchEvent(new Event("online"));
  await flush();
  assert.equal(reads, 4);
  connection.setConnected(true);
  t.mock.timers.tick(20000);
  assert.equal(reads, 4);
  stop();
  connection.setConnected(false);
  activityTarget.dispatchEvent(new Event("online"));
  assert.equal(reads, 4);
  assert.equal(connection.listeners.size, 0);
});

test("slow HTTP reads do not overlap and tab replacement prevents rescheduling", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const connection = new Connection();
  const activityTarget = new EventTarget();
  const visibilityTarget = new Visibility();
  let release!: () => void;
  let reads = 0;
  installHttpFallbackPolling({
    connection, activityTarget, visibilityTarget, isOnline: () => true,
    refresh: () => { reads++; return new Promise<void>((resolve) => { release = resolve; }); },
  });
  activityTarget.dispatchEvent(new Event("online"));
  visibilityTarget.dispatchEvent(new Event("visibilitychange"));
  t.mock.timers.tick(30000);
  assert.equal(reads, 1);
  activityTarget.dispatchEvent(new Event("game:tab-replaced"));
  release();
  await flush();
  t.mock.timers.tick(30000);
  assert.equal(reads, 1);
  assert.equal(connection.listeners.size, 0);
});

test("a failed HTTP refresh retries rather than abandoning basic features", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let reads = 0;
  const stop = installHttpFallbackPolling({
    connection: new Connection(), activityTarget: new EventTarget(), visibilityTarget: new Visibility(), isOnline: () => true,
    refresh: async () => { if (++reads === 1) throw new Error("temporary outage"); },
  });
  await flush();
  t.mock.timers.tick(5000);
  await flush();
  assert.equal(reads, 2);
  stop();
});
