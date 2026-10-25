import assert from "node:assert/strict";
import test from "node:test";
import { WorldRuntime } from "./WorldRuntime.ts";

class FakeSocket {
  readyState = 0;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent<unknown>) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  sent: string[] = [];

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

  close(): void {
    this.readyState = 3;
    this.onclose?.(new CloseEvent("close"));
  }
}

const welcome = (requestId: string, tick = 10, playerX = 10) => ({
  v: 1,
  type: "world.welcome",
  request_id: requestId,
  payload: {
    session_id: "session-1",
    zone_id: "starter",
    map_version: "map-1",
    content_version: "content-1",
    tick_rate: 20,
    server_tick: tick,
    entity_id: "player-1",
    entity_revision: 0,
    baseline_id: 1,
    skills: [{ slot: 1, id: "PREVIEW_BOLT", name: "Bolt", mode: "TARGET", mana_cost: 4, range: 9, cooldown_ticks: 30, windup_ticks: 3 }],
    snapshot: {
      tick,
      ack_seq: 0,
      baseline_id: 1,
      base_tick: tick,
      is_baseline: true,
      entities: [
        { entity_id: "player-1", entity_revision: 0, x: playerX, y: 10, hp: 80, max_hp: 100, mana: 70, max_mana: 100, cooldowns: [0, 0, 5, 0, 0, 0, 0] },
        { entity_id: "remote-1", entity_revision: 0, x: 20, y: 20, hp: 80 },
        { entity_id: "monster-1", entity_revision: 1, kind: "monster", x: 12, y: 10, hp: 70, max_hp: 100, template_id: "monster_wolf", name: "Lang Yêu", ai_state: "chase", aggro_target_id: "player-1" },
      ],
    },
  },
});

test("WorldRuntime joins, predicts movement and handles click targets", async () => {
  const fake = new FakeSocket();
  const runtime = new WorldRuntime({
    socketOptions: {
      url: "wss://api.example/api/ws",
      getToken: () => "token",
      getTabId: () => "tab",
      createSocket: () => fake as never,
    },
  });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  fake.open();
  const joinRequest = JSON.parse(fake.sent[0]) as { request_id: string };
  fake.receive(welcome(joinRequest.request_id));
  const result = await pending;

  assert.equal(result?.zone_id, "starter");
  assert.equal(runtime.getStatus(), "joined");
  assert.deepEqual(runtime.getSelfPosition(), { x: 10, y: 10 });
  assert.equal(runtime.getSelfHp(), 80);
  assert.equal(runtime.getEntities().length, 3);
  const monster = runtime.getEntity("monster-1");
  assert.equal(monster?.name, "Lang Yêu");
  assert.equal(monster?.templateId, "monster_wolf");
  assert.equal(monster?.aiState, "chase");
  assert.equal(monster?.aggroTargetId, "player-1");
  assert.equal(runtime.getSkills()[0].mode, "TARGET");
  assert.equal(runtime.getSelfCombat().mana, 70);
  assert.equal(runtime.getSelfCombat().cooldowns[2], 5);

  runtime.setMoveVector({ x: 1, y: 0 });
  runtime.update(0.5);
  assert.ok(runtime.getSelfPosition().x > 10);
  const sentInput = fake.sent
    .map((line) => JSON.parse(line) as { type: string })
    .filter((message) => message.type === "world.input");
  assert.ok(sentInput.length >= 1);

  assert.equal(runtime.sendMoveTarget({ x: 12, y: 10 }), true);
  const moveTarget = fake.sent
    .map((line) => JSON.parse(line) as { type: string; payload: { x: number; y: number } })
    .find((message) => message.type === "world.move_target");
  assert.equal(moveTarget?.payload.x, 12);
  assert.equal(moveTarget?.payload.y, 10);
  assert.deepEqual(runtime.getMoveTarget(), { x: 12, y: 10 });

  runtime.disconnect();
  assert.equal(runtime.getStatus(), "closed");
});
