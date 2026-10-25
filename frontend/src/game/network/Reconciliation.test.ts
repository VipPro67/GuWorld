import assert from "node:assert/strict";
import test from "node:test";
import { resolveCorrection } from "./Reconciliation.ts";

test("resolveCorrection smooths a small error", () => {
  const correction = resolveCorrection({ x: 0, y: 0 }, { x: 1, y: 0 }, 0.25, 2);
  assert.equal(correction.snapped, false);
  assert.equal(correction.position.x, 0.25);
  assert.equal(correction.distance, 1);
});

test("resolveCorrection hard snaps above the threshold", () => {
  const correction = resolveCorrection({ x: 0, y: 0 }, { x: 5, y: 0 }, 0.25, 2);
  assert.equal(correction.snapped, true);
  assert.deepEqual(correction.position, { x: 5, y: 0 });
});
