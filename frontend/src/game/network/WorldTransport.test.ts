import assert from "node:assert/strict";
import test from "node:test";
import { WorldTransport } from "./WorldTransport.ts";
import {
  DatagramOp,
  Mask,
  Vital,
  decodeDatagram,
  decodeInput,
  dequantize,
  encodeSnapshotDelta,
  encodeSnapshotFull,
  encodeTransfer,
  encodeWelcome,
  encodeFrame,
  Op,
  quantize,
  type DeltaEntity,
  Writer,
  type SnapshotFull,
} from "./v2/Codec.ts";

function emptyState(netId: number): DeltaEntity {
  return {
    netId, mask: 0, hasExt: false, ext: 0, x: 0, y: 0, vitalsSub: 0, hp: 0, shield: 0,
    mana: 0, thought: 0, revision: 0, statuses: [], skillSlot: 0, castEndTick: 0,
    castWindup: 0, flags: 0, formKind: 0, formNetId: 0, modelScaleX100: 0,
    collisionRadiusX10: 0, currentFormHp: 0, formFlags: 0, aiState: 0, aggroNetId: 0,
    ownerNetId: 0, companionKind: "", stance: 0, radiusX10: 0, maxHp: 0, maxMana: 0,
  };
}

function baseline(): SnapshotFull {
  const state = emptyState(7);
  state.mask = Mask.Position | Mask.Vitals | Mask.Revision;
  state.x = quantize(12);
  state.y = quantize(34);
  state.vitalsSub = Vital.HP | Vital.Mana;
  state.hp = 90;
  state.mana = 40;
  state.revision = 3;
  return {
    tick: 100,
    baselineId: 1,
    baseTick: 100,
    ackSeq: 0,
    isBaseline: true,
    spawns: [{
      netId: 7, entityId: "player-1", name: "Tester", templateId: "gu_master", kind: 0,
      partyId: "", flags: 0, radiusX10: 4, maxHp: 100, maxMana: 80, maxThought: 10,
      ownerNetId: 0, companionKind: "", stance: 0,
    }],
    states: [state],
    removed: [],
    projectiles: [],
  };
}

function fakeTransport() {
  let streamController!: ReadableStreamDefaultController<Uint8Array>;
  let datagramController!: ReadableStreamDefaultController<Uint8Array>;
  const sentFrames: Uint8Array[] = [];
  const sentDatagrams: Uint8Array[] = [];
  const streamReadable = new ReadableStream<Uint8Array>({ start(c) { streamController = c; } });
  const datagramReadable = new ReadableStream<Uint8Array>({ start(c) { datagramController = c; } });
  const transport = {
    ready: Promise.resolve(),
    closed: new Promise<{ closeCode?: number; reason?: string }>(() => {}),
    datagrams: {
      readable: datagramReadable,
      writable: new WritableStream<Uint8Array>({ write(chunk) { sentDatagrams.push(chunk); } }),
      maxDatagramSize: 1200,
    },
    createBidirectionalStream: async () => ({
      readable: streamReadable,
      writable: new WritableStream<Uint8Array>({ write(chunk) { sentFrames.push(chunk); } }),
    }),
    close() {},
  };
  return { transport, pushFrame: (frame: Uint8Array) => streamController.enqueue(frame), pushDatagram: (d: Uint8Array) => datagramController.enqueue(d), sentFrames, sentDatagrams };
}

test("joins over WebTransport and exposes the baseline snapshot", async () => {
  const fake = fakeTransport();
  const client = new WorldTransport({
    url: "https://localhost:8443/api/wt",
    getToken: () => "token",
    getTabId: () => "tab-1",
    createTransport: () => fake.transport,
  });
  const pending = client.connect({ zone_id: "forest", map_version: "map-1", content_version: "content-1" });
  await new Promise((resolve) => setTimeout(resolve, 0));

  const full = baseline();
  const welcome = encodeWelcome({
    sessionId: "s1", zoneId: "forest#realtime-preview", mapVersion: "map-1",
    contentVersion: "content-1", tickRate: 20, movementSpeedX100: 500, serverTick: 100,
    entityId: "player-1", entityRevision: 3, baselineId: 1, mapHash: "hash",
    skillsJson: new TextEncoder().encode("[]"), loadoutRevision: "rev",
    snapshot: encodeSnapshotFull(full),
  });
  fake.pushFrame(encodeFrame(Op.Welcome, welcome));
  const joined = await pending;

  assert.equal(joined.zone_id, "forest#realtime-preview");
  assert.equal(joined.movement_speed, 5);
  assert.equal(client.isJoined(), true);
  const snapshot = client.getSnapshot();
  assert.ok(snapshot);
  assert.equal(snapshot.entities.length, 1);
  assert.equal(snapshot.entities[0].entity_id, "player-1");
  assert.equal(dequantize(quantize(snapshot.entities[0].x)), 12);
  assert.equal(snapshot.entities[0].hp, 90);

  const seq = client.sendInput({ x: 1, y: 0 });
  assert.equal(seq, 1);
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(fake.sentDatagrams.length, 1);
  const datagram = decodeDatagram(fake.sentDatagrams[0]);
  assert.equal(datagram.op, DatagramOp.Input);
  assert.deepEqual(decodeInput(datagram.payload), { seq: 1, moveX: 127, moveY: 0 });
});

test("delivers combat events on the reliable stream", async () => {
  const fake = fakeTransport();
  const client = new WorldTransport({
    url: "https://localhost:8443/api/wt",
    getToken: () => "token",
    createTransport: () => fake.transport,
  });
  const pending = client.connect({ zone_id: "forest", map_version: "map-1", content_version: "content-1" });
  await new Promise((resolve) => setTimeout(resolve, 0));
  fake.pushFrame(encodeFrame(Op.Welcome, encodeWelcome({
    sessionId: "s1", zoneId: "forest", mapVersion: "map-1", contentVersion: "content-1",
    tickRate: 20, movementSpeedX100: 500, serverTick: 100, entityId: "player-1",
    entityRevision: 3, baselineId: 1, mapHash: "", skillsJson: new Uint8Array(0),
    loadoutRevision: "", snapshot: encodeSnapshotFull(baseline()),
  })));
  await pending;

  const received: Array<{ type: string; source?: string; amount?: number }> = [];
  client.onEvents((event) => received.push({
    type: event.type,
    source: event.source_id,
    amount: typeof event.payload?.amount === "number" ? event.payload.amount : undefined,
  }));

  const payload = new Writer();
  payload.varint(0).u8(1).varint(101).varint(100).u8(8).u16(7).u16(0);
  const body = new TextEncoder().encode('{"amount":12}');
  payload.u16(body.length).raw(body);
  fake.pushFrame(encodeFrame(Op.Events, payload.bytes()));
  await new Promise((resolve) => setTimeout(resolve, 10));

  assert.equal(received.length, 1);
  assert.equal(received[0].type, "DAMAGE_APPLIED");
  assert.equal(received[0].source, "player-1");
  assert.equal(received[0].amount, 12);
});

test("applies snapshot deltas from datagrams", async () => {
  const fake = fakeTransport();
  const client = new WorldTransport({
    url: "https://localhost:8443/api/wt",
    getToken: () => "token",
    createTransport: () => fake.transport,
  });
  const pending = client.connect({ zone_id: "forest", map_version: "map-1", content_version: "content-1" });
  await new Promise((resolve) => setTimeout(resolve, 0));
  fake.pushFrame(encodeFrame(Op.Welcome, encodeWelcome({
    sessionId: "s1", zoneId: "forest", mapVersion: "map-1", contentVersion: "content-1",
    tickRate: 20, movementSpeedX100: 500, serverTick: 100, entityId: "player-1",
    entityRevision: 3, baselineId: 1, mapHash: "", skillsJson: new Uint8Array(0),
    loadoutRevision: "", snapshot: encodeSnapshotFull(baseline()),
  })));
  await pending;

  const moved = emptyState(7);
  moved.mask = Mask.Position | Mask.Vitals | Mask.Revision;
  moved.x = quantize(20);
  moved.y = quantize(34);
  moved.vitalsSub = Vital.HP;
  moved.hp = 70;
  moved.revision = 4;
  const delta = encodeSnapshotDelta({
    tick: 101, baselineId: 1, ackSeq: 5, hasProjectiles: true,
    entities: [moved], removed: [],
    projectiles: [{ id: 3, sourceNetId: 7, skillId: "moon_blade", vfx: "slash", x: quantize(14), y: quantize(15), radiusX10: 5 }],
  });
  fake.pushDatagram(delta);
  await new Promise((resolve) => setTimeout(resolve, 10));

  const snapshot = client.getSnapshot();
  assert.equal(snapshot?.tick, 101);
  assert.equal(snapshot?.entities[0].entity_revision, 4);
  assert.equal(snapshot?.entities[0].hp, 70);
  assert.equal(dequantize(quantize(snapshot!.entities[0].x)), 20);
  assert.equal(snapshot?.projectiles?.length, 1);
  assert.equal(snapshot?.projectiles?.[0].source_id, "player-1");
  assert.equal(snapshot?.projectiles?.[0].skill_id, "moon_blade");
  assert.equal(snapshot?.projectiles?.[0].radius, 0.5);

  const transfers: Array<{ to: string; legacyX: number; legacyY: number; hp: number }> = [];
  client.onTransfer((t) => transfers.push({ to: t.to_zone_id, legacyX: t.legacy_x, legacyY: t.legacy_y, hp: t.hp }));
  fake.pushFrame(encodeFrame(Op.Transfer, encodeTransfer({
    fromZoneId: "forest", toZoneId: "valley", transferId: "tr-1",
    targetX: quantize(4), targetY: quantize(8), legacyX: -12, legacyY: 45, hp: 70,
  })));
  await new Promise((resolve) => setTimeout(resolve, 10));
  assert.deepEqual(transfers, [{ to: "valley", legacyX: -12, legacyY: 45, hp: 70 }]);
});

test("preserves revision on partial deltas and requests Op.Resync on baseline mismatch", async () => {
  const fake = fakeTransport();
  const client = new WorldTransport({
    url: "https://localhost:8443/api/wt",
    getToken: () => "token",
    createTransport: () => fake.transport,
  });
  const pending = client.connect({ zone_id: "forest", map_version: "map-1", content_version: "content-1" });
  await new Promise((resolve) => setTimeout(resolve, 0));
  fake.pushFrame(encodeFrame(Op.Welcome, encodeWelcome({
    sessionId: "s1", zoneId: "forest", mapVersion: "map-1", contentVersion: "content-1",
    tickRate: 20, movementSpeedX100: 500, serverTick: 100, entityId: "player-1",
    entityRevision: 3, baselineId: 1, mapHash: "", skillsJson: new Uint8Array(0),
    loadoutRevision: "", snapshot: encodeSnapshotFull(baseline()),
  })));
  await pending;

  const resyncErrors: string[] = [];
  client.onResyncRequired((reason) => resyncErrors.push(reason));

  // Partial delta without Mask.Revision keeps previous entity_revision=3.
  const posOnly = emptyState(7);
  posOnly.mask = Mask.Position;
  posOnly.x = quantize(15);
  posOnly.y = quantize(34);
  fake.pushDatagram(encodeSnapshotDelta({
    tick: 101, baselineId: 1, ackSeq: 1, hasProjectiles: false,
    entities: [posOnly], removed: [], projectiles: [],
  }));
  await new Promise((resolve) => setTimeout(resolve, 10));
  assert.equal(client.getSnapshot()?.tick, 101);
  assert.equal(client.getSnapshot()?.entities[0].entity_revision, 3);
  assert.equal(resyncErrors.length, 0);

  // Delta referencing unknown baselineId=2 triggers Op.Resync on stream instead of fatal error.
  const framesBefore = fake.sentFrames.length;
  fake.pushDatagram(encodeSnapshotDelta({
    tick: 102, baselineId: 2, ackSeq: 1, hasProjectiles: false,
    entities: [], removed: [], projectiles: [],
  }));
  await new Promise((resolve) => setTimeout(resolve, 10));
  assert.equal(resyncErrors.length, 0, "should send Op.Resync instead of failing immediately");
  assert.equal(fake.sentFrames.length, framesBefore + 1);
  assert.equal(fake.sentFrames.at(-1)?.[4], Op.Resync);
});
