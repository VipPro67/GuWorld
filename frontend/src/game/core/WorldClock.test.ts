import assert from "node:assert/strict";
import test from "node:test";
import { WorldClock } from "./WorldClock.ts";

test("WorldClock estimates ticks from the last sync", () => {
  const clock = new WorldClock();
  clock.sync(100, 20, 1_000);
  assert.equal(clock.isSynced(), true);
  assert.equal(clock.estimatedTick(1_500), 110);
  assert.equal(clock.estimatedTick(1_000), 100);
  assert.equal(clock.getTickRate(), 20);
});

test("WorldClock ignores invalid sync values and resets", () => {
  const clock = new WorldClock();
  clock.sync(Number.NaN, 20);
  clock.sync(10, 0);
  assert.equal(clock.isSynced(), false);
  clock.sync(10, 20, 0);
  clock.reset();
  assert.equal(clock.isSynced(), false);
  assert.equal(clock.estimatedTick(5_000), 0);
});
