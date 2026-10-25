import assert from "node:assert/strict";
import test from "node:test";
import { GameEventBus } from "./GameEvents.ts";

test("GameEventBus delivers typed payloads and unsubscribes", () => {
  const bus = new GameEventBus();
  const statuses: string[] = [];
  const off = bus.on("session:status", (payload) => statuses.push(payload.status));

  bus.emit("session:status", { status: "CONNECTING" });
  off();
  bus.emit("session:status", { status: "JOINED" });

  assert.deepEqual(statuses, ["CONNECTING"]);
});

test("GameEventBus clear removes every listener", () => {
  const bus = new GameEventBus();
  let errors = 0;
  bus.on("world:error", () => {
    errors += 1;
  });

  bus.clear();
  bus.emit("world:error", { code: "x", message: "boom" });

  assert.equal(errors, 0);
});

test("off only removes the matching handler", () => {
  const bus = new GameEventBus();
  const seen: string[] = [];
  const first = () => seen.push("first");
  const second = () => seen.push("second");
  bus.on("session:status", first);
  bus.on("session:status", second);

  bus.off("session:status", first);
  bus.emit("session:status", { status: "JOINED" });

  assert.deepEqual(seen, ["second"]);
});
