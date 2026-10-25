import assert from "node:assert/strict";
import test from "node:test";
import { SelfPredictor } from "./Prediction.ts";

test("SelfPredictor moves locally and records pending input", () => {
  const predictor = new SelfPredictor();
  predictor.setSpeed(4);
  predictor.recordInput(1, { x: 1, y: 0 }, 0.5);
  predictor.advance({ x: 1, y: 0 }, 0.5);

  const position = predictor.getPosition();
  assert.ok(Math.abs(position.x - 2) < 1e-9);
  assert.equal(predictor.pendingCount(), 1);
});

test("SelfPredictor reconciles with an acknowledged authoritative position", () => {
  const predictor = new SelfPredictor();
  predictor.setSpeed(4);
  predictor.setAuthoritative({ x: 10, y: 5 }, 7);
  predictor.advance({ x: 0, y: 1 }, 0.25);

  const result = predictor.reconcile({ x: 10, y: 6 }, 8);
  assert.equal(result.correction, 0);
  assert.equal(result.replayed, 0);
  assert.deepEqual(predictor.getPosition(), { x: 10, y: 6 });
  assert.equal(predictor.getAckSeq(), 8);
});

test("SelfPredictor soft reconciliation smooths small errors and snaps large ones", () => {
  const predictor = new SelfPredictor();
  predictor.setSpeed(4);
  predictor.setAuthoritative({ x: 0, y: 0 }, 1);

  const small = predictor.softReconcile({ x: 1, y: 0 }, 2, 2, 0.5);
  assert.equal(small.snapped, false);
  assert.equal(small.position.x, 0.5);

  const large = predictor.softReconcile({ x: 100, y: 0 }, 3, 2, 0.5);
  assert.equal(large.snapped, true);
  assert.deepEqual(large.position, { x: 100, y: 0 });
});
