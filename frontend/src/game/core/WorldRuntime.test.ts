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
    movement_speed: 5.6,
    server_tick: tick,
    entity_id: "player-1",
    entity_revision: 0,
    baseline_id: 1,
    skills: [
      { slot: 1, id: "PREVIEW_BOLT", name: "Bolt", mode: "TARGET", mana_cost: 4, range: 9, cooldown_ticks: 30, windup_ticks: 3 },
      { slot: 2, id: "DASH", name: "Dash", mode: "DIRECTION", mana_cost: 4, range: 9, cooldown_ticks: 30, windup_ticks: 0, move_distance: 4, move_duration_ticks: 6, move_behavior: "MOVE_ONLY" },
    ],
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
  runtime.update(0.1);
  assert.ok(Math.abs(runtime.getSelfPosition().x - 10.56) < 1e-9);
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

test("dash snapshots control self position until DASH_ENDED", async () => {
  const fake = new FakeSocket();
  const runtime = new WorldRuntime({ socketOptions: {
    url: "wss://api.example/api/ws", getToken: () => "token", getTabId: () => "tab", createSocket: () => fake as never,
  } });
  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  fake.open();
  const joinRequest = JSON.parse(fake.sent[0]) as { request_id: string };
  fake.receive(welcome(joinRequest.request_id));
  await pending;
  runtime.setMoveVector({ x: 0, y: 1 });
  fake.receive({ v: 1, type: "world.events", payload: { from_event_id: 1,
    events: [{ event_id: 1, tick: 10, type: "CAST_ACCEPTED", source_id: "player-1", payload: { skill_id: "DASH", duration_ticks: 0 } }],
  } });
  runtime.update(0.1);
  assert.deepEqual(runtime.getSelfPosition(), { x: 10, y: 10 }, "ordinary movement predicted during dash");
  fake.receive({ v: 1, type: "world.snapshot", payload: { tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.7, y: 10, hp: 80 }], removed: [],
  } });
  assert.deepEqual(runtime.getSelfPosition(), { x: 10.7, y: 10 }, "dash did not follow authoritative snapshot");
  fake.receive({ v: 1, type: "world.events", payload: { from_event_id: 2,
    events: [{ event_id: 2, tick: 11, type: "DASH_ENDED", source_id: "player-1", payload: { skill_id: "DASH" } }],
  } });
  runtime.update(0.1);
  assert.ok(runtime.getSelfPosition().y > 10, "ordinary prediction did not resume after dash");
  runtime.disconnect();
});

test("held movement does not step backward when a delayed snapshot arrives", async () => {
  const fake = new FakeSocket();
  let now = 0;
  const runtime = new WorldRuntime({
    nowMs: () => now,
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
  await pending;

  runtime.setMoveVector({ x: 1, y: 0 });
  runtime.update(0.1);
  now = 100;
  const before = runtime.getSelfPosition().x;
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.28, y: 10, hp: 80 }],
      removed: [],
    },
  });
  assert.ok(runtime.getSelfPosition().x >= before, `snapshot pulled player backward from ${before} to ${runtime.getSelfPosition().x}`);
  runtime.setMoveVector({ x: 0, y: 0 });
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 12, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 2, x: before, y: 10, hp: 80 }],
      removed: [],
    },
  });
  assert.ok(runtime.getSelfPosition().x >= before, "snapshot pulled player back before the stop input was acknowledged");
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 13, ack_seq: 2, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 3, x: before, y: 10, hp: 80 }],
      removed: [],
    },
  });
  assert.ok(Math.abs(runtime.getSelfPosition().x - before) < 0.1);
  runtime.disconnect();
});

test("server blocked and slowed movement still corrects local prediction", async () => {
  for (const serverStep of [0, 0.14]) {
    const fake = new FakeSocket();
    let now = 0;
    const runtime = new WorldRuntime({
      nowMs: () => now,
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
    await pending;
    runtime.setMoveVector({ x: 1, y: 0 });
    for (let step = 1; step <= 30; step++) {
      runtime.update(0.05);
      now += 50;
      fake.receive({
        v: 1,
        type: "world.snapshot",
        payload: {
          tick: 10 + step, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
          entities: [{ entity_id: "player-1", entity_revision: step, x: 10 + serverStep * step, y: 10, hp: 80 }],
          removed: [],
        },
      });
      assert.ok(runtime.getSelfPosition().x <= 10 + serverStep * step + 0.6,
        `server step ${serverStep} accumulated prediction error at snapshot ${step}`);
      assert.ok(runtime.getDebugStats().lastCorrection < 2,
        `server step ${serverStep} caused a hard snap at snapshot ${step}`);
    }
    runtime.disconnect();
  }
});

test("click-to-move stays smooth after prediction reaches the target before the server", async () => {
  const fake = new FakeSocket();
  let now = 0;
  const runtime = new WorldRuntime({
    nowMs: () => now,
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
  await pending;
  assert.equal(runtime.sendMoveTarget({ x: 10.56, y: 10 }), true);
  runtime.update(0.1);
  const before = runtime.getSelfPosition().x;
  now = 100;
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.28, y: 10, hp: 80 }],
      removed: [],
    },
  });
  assert.ok(runtime.getSelfPosition().x >= before - 0.001);
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 12, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 2, x: 10.56, y: 10, hp: 80 }],
      removed: [],
    },
  });
  assert.ok(Math.abs(runtime.getSelfPosition().x - 10.56) < 0.001);
  runtime.disconnect();
});

test("turning away from a wall resumes immediate local movement", async () => {
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
  await pending;
  runtime.setMoveVector({ x: 1, y: 0 });
  runtime.update(0.05);
  fake.receive({
    v: 1,
    type: "world.snapshot",
    payload: {
      tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
      entities: [{ entity_id: "player-1", entity_revision: 1, x: 10, y: 10, hp: 80 }],
      removed: [],
    },
  });
  runtime.setMoveVector({ x: 0, y: 1 });
  const before = runtime.getSelfPosition().y;
  runtime.update(0.05);
  assert.ok(runtime.getSelfPosition().y > before + 0.2);
  runtime.disconnect();
});

const waitUntil = async (predicate: () => boolean, timeoutMs = 3_000): Promise<void> => {
  const startedAt = Date.now();
  while (!predicate()) {
    if (Date.now() - startedAt > timeoutMs) throw new Error("timed out waiting for condition");
    await new Promise((resolve) => setTimeout(resolve, 10));
  }
};

test("WorldRuntime retries a join while the previous controller still owns the lease", async () => {
  const sockets: FakeSocket[] = [];
  const runtime = new WorldRuntime({
    socketOptions: {
      url: "wss://api.example/api/ws",
      getToken: () => "token",
      getTabId: () => "tab",
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket as never;
      },
    },
  });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  const first = sockets[0];
  first.open();
  const firstRequest = JSON.parse(first.sent[0]) as { request_id: string };
  first.receive({
    v: 1,
    type: "world.reject",
    request_id: firstRequest.request_id,
    payload: { code: "realtime_character_already_owned", message: "Character already has an active realtime controller" },
  });

  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "connecting", "a stale lease must retry instead of parking in error");
  assert.equal(runtime.getErrorCode(), "realtime_character_already_owned");

  await waitUntil(() => sockets.length === 2);
  const second = sockets[1];
  second.open();
  const secondRequest = JSON.parse(second.sent[0]) as { request_id: string };
  second.receive(welcome(secondRequest.request_id));

  await waitUntil(() => runtime.getStatus() === "joined");
  assert.equal(runtime.getErrorCode(), "");
  runtime.disconnect();
});

test("WorldRuntime stops retrying on a non-retryable join rejection", async () => {
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
  fake.receive({
    v: 1,
    type: "world.reject",
    request_id: joinRequest.request_id,
    payload: { code: "realtime_map_mismatch", message: "Requested map does not match the character's current region" },
  });

  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "error");
  assert.equal(runtime.getErrorCode(), "realtime_map_mismatch");
  runtime.disconnect();
});

test("disconnect cancels a scheduled join retry", async () => {
  const sockets: FakeSocket[] = [];
  const runtime = new WorldRuntime({
    socketOptions: {
      url: "wss://api.example/api/ws",
      getToken: () => "token",
      getTabId: () => "tab",
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket as never;
      },
    },
  });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  const first = sockets[0];
  first.open();
  const joinRequest = JSON.parse(first.sent[0]) as { request_id: string };
  first.receive({
    v: 1,
    type: "world.reject",
    request_id: joinRequest.request_id,
    payload: { code: "realtime_character_already_owned", message: "busy" },
  });
  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "connecting");

  runtime.disconnect();
  assert.equal(runtime.getStatus(), "closed");

  await new Promise((resolve) => setTimeout(resolve, 600));
  assert.equal(sockets.length, 1, "a disconnected runtime must not open another socket");
  assert.equal(runtime.getStatus(), "closed");
});
