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
