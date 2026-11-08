import assert from "node:assert/strict";
import test from "node:test";
import { RemoteInterpolator } from "./RemoteInterpolator.ts";

const entity = (id: string, x: number, y = 0) => ({
  entity_id: id,
  entity_revision: 0,
  x,
  y,
  hp: 100,
});

test("RemoteInterpolator interpolates between ticks and clamps at the ends", () => {
  const interpolator = new RemoteInterpolator();
  interpolator.push(10, [entity("mob", 0)]);
  interpolator.push(20, [entity("mob", 10)]);

  assert.deepEqual(interpolator.positionAt("mob", 15), { x: 5, y: 0 });
  assert.deepEqual(interpolator.positionAt("mob", 5), { x: 0, y: 0 });
  assert.deepEqual(interpolator.positionAt("mob", 25), { x: 10, y: 0 });
});

test("RemoteInterpolator drops duplicate ticks, stale entities and reset clears state", () => {
  const interpolator = new RemoteInterpolator();
  interpolator.push(10, [entity("a", 0), entity("b", 2)]);
  interpolator.push(10, [entity("a", 99)]);
  assert.deepEqual(interpolator.positionAt("a", 10), { x: 0, y: 0 });

  interpolator.push(11, [entity("a", 1)]);
  assert.equal(interpolator.positionAt("b", 11), null);
  assert.equal(interpolator.size(), 1);

  interpolator.reset();
  assert.equal(interpolator.positionAt("a", 11), null);
});

test("RemoteInterpolator adapts its render buffer to snapshot gaps and recovers", () => {
  const interpolator = new RemoteInterpolator();
  interpolator.push(10, [entity("mob", 0)]);
  assert.equal(interpolator.getInterpolationDelayTicks(), 2);

  interpolator.push(18, [entity("mob", 1)]);
  assert.equal(interpolator.getInterpolationDelayTicks(), 6);

  for (let tick = 19; tick <= 50; tick += 1) {
    interpolator.push(tick, [entity("mob", 1)]);
  }
  assert.equal(interpolator.getInterpolationDelayTicks(), 2);
});

test("RemoteInterpolator extrapolates briefly but never across a long snapshot gap", () => {
  const interpolator = new RemoteInterpolator();
  interpolator.push(10, [entity("mob", 1)]);
  interpolator.push(11, [entity("mob", 1.2)]);

  assert.deepEqual(interpolator.positionAt("mob", 12), { x: 1.4, y: 0 });
  assert.ok(Math.abs(interpolator.positionAt("mob", 100)!.x - 1.6) < 1e-9);

  interpolator.push(20, [entity("mob", 3)]);
  assert.deepEqual(interpolator.positionAt("mob", 22), { x: 3, y: 0 });
});
