import assert from "node:assert/strict";
import test from "node:test";
import { WorldSocket, type WorldSocketOptions } from "./WorldSocket.ts";

class FakeSocket {
  readyState = 0;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent<unknown>) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  sent: string[] = [];
  closeCode: number | undefined;

  open(): void {
    this.readyState = 1;
    this.onopen?.(new Event("open"));
  }

  receive(message: unknown): void {
    this.onmessage?.({ data: JSON.stringify(message) } as MessageEvent<unknown>);
  }

  send(data: string): void {
    this.sent.push(data);
  }

  close(code?: number): void {
    this.closeCode = code;
    this.readyState = 3;
    this.onclose?.(new CloseEvent("close"));
  }
}

const join = { zone_id: "starter", map_version: "map-1", content_version: "content-1" };

test("WorldSocket joins, buffers snapshots, and sequences normalized input", async () => {
  const fake = new FakeSocket();
  const options: WorldSocketOptions = {
    url: "wss://api.example/api/ws",
    getToken: () => "token-value",
    getTabId: () => "tab-value",
    createSocket: () => fake as unknown as ReturnType<NonNullable<WorldSocketOptions["createSocket"]>>,
  };
  const client = new WorldSocket(options);
  const received: number[] = [];
  client.onSnapshot((state) => received.push(state.tick));

  const pending = client.connect(join);
  fake.open();
  const request = JSON.parse(fake.sent[0]) as { request_id: string; type: string };
  assert.equal(request.type, "world.join");
  fake.receive({
    v: 1,
    type: "world.welcome",
    request_id: request.request_id,
    payload: {
      session_id: "session-1",
      zone_id: "starter",
      map_version: "map-1",
      content_version: "content-1",
      tick_rate: 20,
      server_tick: 10,
      entity_id: "player-1",
      entity_revision: 0,
      baseline_id: 1,
      skills: [{ slot: 1, id: "PREVIEW_BOLT", name: "Bolt", mode: "TARGET", mana_cost: 4, range: 9, radius_units: 0, cooldown_ticks: 30, windup_ticks: 3 }],
      loadout_revision: "loadout-1",
      snapshot: {
        tick: 10,
        ack_seq: 0,
        baseline_id: 1,
        base_tick: 10,
        is_baseline: true,
        entities: [{
          entity_id: "player-1", entity_revision: 0, kind: "player", party_id: "party-1", x: 0, y: 0, hp: 100, max_hp: 120,
          mana: 90, max_mana: 100, cooldowns: [0, 3, 0, 0, 0, 0, 0], cast_skill_id: "PREVIEW_BOLT", cast_end_tick: 13, cast_windup_ticks: 3,
        }],
      },
    },
  });
  await pending;
  assert.equal(client.isJoined(), true);
  assert.deepEqual(received, [10]);
  assert.equal(client.getSnapshot()?.entities[0].kind, "player");
  assert.equal(client.getSnapshot()?.entities[0].max_hp, 120);
  assert.equal(client.getSnapshot()?.entities[0].party_id, "party-1");
  assert.equal(client.getSnapshot()?.entities[0].mana, 90);
  assert.equal(client.getSnapshot()?.entities[0].cooldowns?.[1], 3);
  assert.equal(client.getSnapshot()?.entities[0].cast_skill_id, "PREVIEW_BOLT");
  assert.equal(client.getWelcome()?.skills?.[0].mode, "TARGET");
  assert.equal(client.getWelcome()?.loadout_revision, "loadout-1");
  assert.equal(client.sendInput({ x: 1, y: 1 }), 1);
  const input = JSON.parse(fake.sent[1]) as { payload: { seq: number; move: { x: number; y: number } } };
  assert.equal(input.payload.seq, 1);
  assert.ok(Math.abs(input.payload.move.x - Math.SQRT1_2) < 1e-12);
  assert.equal(client.sendMoveTarget({ x: 12.5, y: 8 }), 2);
  const moveTarget = JSON.parse(fake.sent[2]) as {
    type: string;
    payload: { seq: number; x: number; y: number };
  };
  assert.equal(moveTarget.type, "world.move_target");
  assert.deepEqual(moveTarget.payload, { seq: 2, x: 12.5, y: 8 });
  assert.equal(client.sendMoveTarget({ x: Number.NaN, y: 0 }), null);

  const events: string[] = [];
  client.onEvents((event) => events.push(event.type));
  assert.equal(client.sendCast({ slot: 3, mode: "GROUND", x: 12.5, y: 8 }), 3);
  const cast = JSON.parse(fake.sent[3]) as { type: string; payload: { seq: number; slot: number; mode: string; x: number } };
  assert.equal(cast.type, "world.cast");
  assert.deepEqual(cast.payload, { seq: 3, slot: 3, mode: "GROUND", x: 12.5, y: 8 });
  assert.equal(client.sendBasicAttack("m:1"), 4);
  const attack = JSON.parse(fake.sent[4]) as { type: string; payload: { seq: number; target_id: string } };
  assert.equal(attack.type, "world.attack");
  assert.deepEqual(attack.payload, { seq: 4, target_id: "m:1" });
  fake.receive({
    v: 1,
    type: "world.events",
    payload: {
      from_event_id: 9,
      events: [{ event_id: 9, tick: 11, type: "DAMAGE_APPLIED", target_id: "m:0", payload: { amount: 12 } }],
    },
  });
  assert.deepEqual(events, ["DAMAGE_APPLIED"]);

  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false, entities: [], removed: [],
      projectiles: [{ id: 1, source_id: "player-1", skill_id: "PREVIEW_WATER_ARROW", x: 2, y: 3, radius: 0.35 }],
    },
  });
  assert.deepEqual(received, [10, 11]);
  assert.equal(client.getSnapshot()?.projectiles?.[0].skill_id, "PREVIEW_WATER_ARROW");
  client.disconnect();
  assert.equal(client.isJoined(), false);
});

test("WorldSocket rejects a join cancelled before the server welcomes it", async () => {
  const fake = new FakeSocket();
  const client = new WorldSocket({
    url: "wss://api.example/api/ws",
    getToken: () => "token-value",
    createSocket: () => fake as unknown as ReturnType<NonNullable<WorldSocketOptions["createSocket"]>>,
  });
  const pending = client.connect(join);
  client.disconnect();
  await assert.rejects(pending, /cancelled/);
});
