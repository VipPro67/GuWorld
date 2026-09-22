import assert from "node:assert/strict";
import test from "node:test";
import { applyRevisionedState } from "./playerRevision.ts";

test("a websocket event arriving before its HTTP response wins at the same revision", () => {
  const websocket = applyRevisionedState({ revision: 10, value: "before" }, 11, "websocket");
  const http = applyRevisionedState(websocket, 11, "http-stale-copy");
  assert.deepEqual(http, { revision: 11, value: "websocket" });
});

test("out-of-order deltas cannot roll state back", () => {
  const newest = applyRevisionedState({ revision: 20, value: "old" }, 22, "newest");
  const late = applyRevisionedState(newest, 21, "late");
  assert.deepEqual(late, { revision: 22, value: "newest" });
});

test("a reconnect snapshot with a newer revision replaces local state", () => {
  const reconnected = applyRevisionedState({ revision: 30, value: "disconnected" }, 37, "snapshot");
  assert.deepEqual(reconnected, { revision: 37, value: "snapshot" });
});
