import assert from "node:assert/strict";
import test from "node:test";
import { SnapshotBuffer } from "./SnapshotBuffer.ts";
import type { WorldSnapshotPayload } from "./Protocol.ts";

const entity = (entity_id: string, x: number, entity_revision = 0) => ({
  entity_id,
  entity_revision,
  x,
  y: 0,
  hp: 100,
});

const baseline = (tick: number, entities = [entity("player-1", 0)]): WorldSnapshotPayload => ({
  tick,
  ack_seq: 0,
  baseline_id: 1,
  base_tick: tick,
  is_baseline: true,
  entities,
});

test("SnapshotBuffer applies a baseline then a coalesced baseline-relative delta", () => {
  const buffer = new SnapshotBuffer();
  const first = buffer.apply(baseline(10, [entity("player-1", 0), entity("mob-1", 5)]));
  assert.equal(first.status, "applied");

  const delta = buffer.apply({
    tick: 18,
    ack_seq: 7,
    baseline_id: 1,
    base_tick: 10,
    is_baseline: false,
    entities: [entity("player-1", 3, 1)],
    removed: ["mob-1"],
  });
  assert.equal(delta.status, "applied");
  if (delta.status !== "applied") return;
  assert.equal(delta.state.tick, 18);
  assert.equal(delta.state.ackSeq, 7);
  assert.deepEqual(delta.state.entities, [entity("player-1", 3, 1)]);
});

test("SnapshotBuffer ignores old ticks and requests recovery for an unknown baseline", () => {
  const buffer = new SnapshotBuffer();
  buffer.apply(baseline(10));
  assert.deepEqual(buffer.apply(baseline(9)), { status: "ignored", reason: "stale_tick" });
  const result = buffer.apply({
    tick: 11,
    ack_seq: 0,
    baseline_id: 2,
    base_tick: 11,
    is_baseline: false,
    entities: [],
  });
  assert.equal(result.status, "resync_required");
});

test("SnapshotBuffer rejects malformed state and duplicate entity IDs", () => {
  const buffer = new SnapshotBuffer();
  const invalid = buffer.apply({ ...baseline(1), entities: [entity("same", 0), entity("same", 1)] });
  assert.equal(invalid.status, "resync_required");
  assert.equal(buffer.getState(), null);
});
