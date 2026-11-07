import assert from "node:assert/strict";
import test from "node:test";
import {
  DatagramOp,
  Mask,
  Vital,
  dequantize,
  decodeDatagram,
  decodeInput,
  decodeSnapshotDelta,
  decodeSnapshotFull,
  decodeWelcome,
  encodeInput,
  encodeSnapshotDelta,
  encodeSnapshotFull,
  encodeWelcome,
  emptyDeltaEntity,
  quantize,
  type DeltaEntity,
  type SnapshotDelta,
} from "./Codec.ts";

// Must match backend/internal/realtime/protocol/v2/codec_test.go.
const GOLDEN_SNAPSHOT_HEX = "82d204000007006300022c0101400620030c000720004000058407782c010500";

function hexToBytes(hex: string): Uint8Array {
  const out = new Uint8Array(hex.length / 2);
  for (let i = 0; i < out.length; i += 1) {
    out[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

function bytesToHex(bytes: Uint8Array): string {
  return [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

test("input roundtrip matches Go size", () => {
  const encoded = encodeInput({ seq: 42, moveX: 127, moveY: -64 });
  assert.equal(encoded.length, 4);
  const { op, payload } = decodeDatagram(encoded);
  assert.equal(op, DatagramOp.Input);
  assert.deepEqual(decodeInput(payload), { seq: 42, moveX: 127, moveY: -64 });
});

test("moving entity delta is seven payload bytes", () => {
  const entity = emptyDeltaEntity(300);
  entity.mask = Mask.Position;
  entity.x = quantize(100);
  entity.y = quantize(50);
  const snapshot: SnapshotDelta = {
    tick: 1234,
    baselineId: 7,
    ackSeq: 99,
    hasProjectiles: false,
    entities: [entity],
    removed: [],
    projectiles: [],
  };
  const encoded = encodeSnapshotDelta(snapshot);
  assert.equal(encoded.length, 18);
  const { payload } = decodeDatagram(encoded);
  const decoded = decodeSnapshotDelta(payload);
  assert.equal(decoded.entities.length, 1);
  assert.equal(decoded.entities[0].netId, 300);
  assert.equal(dequantize(decoded.entities[0].x), 100);
  assert.equal(dequantize(decoded.entities[0].y), 50);
});

test("full entity roundtrip keeps every masked field", () => {
  const entity: DeltaEntity = {
    ...emptyDeltaEntity(12),
    mask: Mask.Position | Mask.Vitals | Mask.Revision | Mask.Statuses | Mask.Cast | Mask.Flags,
    x: 32,
    y: 64,
    vitalsSub: Vital.HP | Vital.Shield | Vital.Mana | Vital.Thought,
    hp: 900,
    shield: 50,
    mana: 300,
    thought: 12,
    revision: 44,
    statuses: [1, 7, 9],
    skillSlot: 3,
    castEndTick: 9999,
    castWindup: 4,
    flags: 0b1010,
  };
  const snapshot: SnapshotDelta = {
    tick: 77,
    baselineId: 2,
    ackSeq: 1500,
    hasProjectiles: true,
    entities: [entity],
    removed: [9, 10],
    projectiles: [
      { id: 5, sourceNetId: 1, skillId: "blood_moon_gu", vfx: "slash", x: 320, y: 640, radiusX10: 3 },
    ],
  };
  const encoded = encodeSnapshotDelta(snapshot);
  const { payload } = decodeDatagram(encoded);
  const decoded = decodeSnapshotDelta(payload);
  const got = decoded.entities[0];
  assert.equal(got.hp, 900);
  assert.equal(got.shield, 50);
  assert.equal(got.mana, 300);
  assert.equal(got.thought, 12);
  assert.equal(got.revision, 44);
  assert.deepEqual(got.statuses, [1, 7, 9]);
  assert.equal(got.castEndTick, 9999);
  assert.deepEqual(decoded.removed, [9, 10]);
  assert.equal(decoded.projectiles[0].skillId, "blood_moon_gu");
  assert.equal(decoded.projectiles[0].x, 320);
  assert.equal(decoded.hasProjectiles, true);
});

test("welcome and full snapshot roundtrip", () => {
  const full = {
    tick: 1234,
    baselineId: 7,
    baseTick: 1234,
    ackSeq: 99,
    isBaseline: true,
    spawns: [
      {
        netId: 12,
        entityId: "uuid-1",
        name: "Người chơi",
        templateId: "gu_master",
        kind: 0,
        partyId: "party-1",
        flags: 1,
        radiusX10: 4,
        maxHp: 1000,
        maxMana: 500,
        maxThought: 120,
        ownerNetId: 0,
        companionKind: "",
        stance: 0,
      },
    ],
    states: [],
    removed: [5],
    projectiles: [
      { id: 3, sourceNetId: 12, skillId: "bolt", vfx: "fire", x: 100, y: 200, radiusX10: 2 },
    ],
  };
  const encodedFull = encodeSnapshotFull(full);
  const decodedFull = decodeSnapshotFull(encodedFull);
  assert.equal(decodedFull.spawns[0].name, "Người chơi");
  assert.equal(decodedFull.spawns[0].maxMana, 500);
  assert.equal(decodedFull.projectiles[0].skillId, "bolt");

  const welcome = {
    sessionId: "session-1",
    zoneId: "forest#realtime-preview",
    mapVersion: "regions-v1",
    contentVersion: "region-content-v1",
    tickRate: 20,
    movementSpeedX100: 560,
    serverTick: 1234,
    entityId: "uuid-1",
    entityRevision: 44,
    baselineId: 7,
    mapHash: "hash",
    skillsJson: new TextEncoder().encode('[{"slot":1}]'),
    loadoutRevision: "rev-1",
    snapshot: encodedFull,
  };
  const decodedWelcome = decodeWelcome(encodeWelcome(welcome));
  assert.equal(decodedWelcome.zoneId, welcome.zoneId);
  assert.equal(decodedWelcome.movementSpeedX100, 560);
  assert.equal(new TextDecoder().decode(decodedWelcome.skillsJson), '[{"slot":1}]');
  assert.deepEqual(decodedWelcome.snapshot, encodedFull);
});

test("golden vector matches the Go codec byte-for-byte", () => {
  const bytes = hexToBytes(GOLDEN_SNAPSHOT_HEX);
  const { op, payload } = decodeDatagram(bytes);
  assert.equal(op, DatagramOp.SnapshotDelta);
  const decoded = decodeSnapshotDelta(payload);
  assert.equal(decoded.tick, 1234);
  assert.equal(decoded.baselineId, 7);
  assert.equal(decoded.ackSeq, 99);
  assert.equal(decoded.entities.length, 2);
  assert.equal(decoded.entities[0].netId, 300);
  assert.equal(decoded.entities[1].netId, 12);
  assert.equal(decoded.entities[1].hp, 900);
  assert.equal(decoded.entities[1].mana, 120);
  assert.deepEqual(decoded.removed, [5]);

  const reencoded = encodeSnapshotDelta(decoded);
  assert.equal(bytesToHex(reencoded), GOLDEN_SNAPSHOT_HEX);
});

test("delta above the datagram budget is rejected", () => {
  const entities: DeltaEntity[] = [];
  for (let i = 0; i < 255; i += 1) {
    const entity = emptyDeltaEntity(i);
    entity.mask = Mask.Position;
    entity.x = i;
    entity.y = i;
    entities.push(entity);
  }
  assert.throws(() =>
    encodeSnapshotDelta({
      tick: 1,
      baselineId: 1,
      ackSeq: 0,
      hasProjectiles: false,
      entities,
      removed: [],
      projectiles: [],
    }),
  );
});
