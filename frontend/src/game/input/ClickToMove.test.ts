import assert from "node:assert/strict";
import test from "node:test";
import { ClickToMove } from "./ClickToMove.ts";
import { clampMoveTarget, normalizeWorldVector } from "../world/WorldUnits.ts";

test("ClickToMove tracks a request and completes near the target", () => {
  const click = new ClickToMove();
  const request = click.begin({ x: 10, y: 4 });
  assert.equal(request?.requestId, 1);
  assert.deepEqual(click.getTarget(), { x: 10, y: 4 });
  assert.equal(click.completeIfNear({ x: 9, y: 4 }), false);
  assert.equal(click.completeIfNear({ x: 10, y: 4.1 }), true);
  assert.equal(click.hasTarget(), false);
});

test("ClickToMove rejects non-finite destinations and supports cancel", () => {
  const click = new ClickToMove();
  assert.equal(click.begin({ x: Number.NaN, y: 0 }), null);
  click.begin({ x: 1, y: 1 });
  click.cancel();
  assert.equal(click.hasTarget(), false);
});

test("World units clamp destinations and normalize vectors", () => {
  const target = clampMoveTarget({ x: 0, y: 0 }, { x: 1000, y: 0 });
  assert.ok(Math.abs(target.x - 64) < 1e-9);
  const normalized = normalizeWorldVector({ x: 3, y: 4 });
  assert.ok(Math.abs(normalized.x - 0.6) < 1e-12);
  assert.ok(Math.abs(normalized.y - 0.8) < 1e-12);
});
