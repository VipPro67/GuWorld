import assert from "node:assert/strict";
import test from "node:test";
import { WorldRuntime } from "./WorldRuntime.ts";
import { WorldRealtimeError } from "../network/WorldRealtimeError.ts";
import type { WorldTransportLike } from "../network/Transport.ts";
import type {
  BasicAttackPayload,
  CapturePayload,
  CastPayload,
  CompanionStancePayload,
  WorldEvent,
  WorldJoinPayload,
  WorldRejectPayload,
  WorldSnapshotPayload,
  WorldTransferPayload,
  WorldWelcomePayload,
} from "../network/Protocol.ts";
import { SnapshotBuffer, type BufferedWorldState } from "../network/SnapshotBuffer.ts";

const welcome = (tick = 10, playerX = 10): WorldWelcomePayload => ({
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
});

class FakeTransport implements WorldTransportLike {
  public joins: WorldJoinPayload[] = [];
  public inputs: Array<{ x: number; y: number }> = [];
  public moveTargets: Array<{ x: number; y: number }> = [];
  public casts: Array<Omit<CastPayload, "seq">> = [];
  public attacks: Array<Omit<BasicAttackPayload, "seq">> = [];
  public captures: Array<{ targetId: string; guId?: string }> = [];
  public stances: Array<{ stance: "DEFEND" | "FOCUS" | "FOLLOW"; focusTargetId?: string }> = [];
  public disconnectCount = 0;
  public failConnect: Error | null = null;

  private readonly welcomePayload: WorldWelcomePayload;
  private readonly buffer = new SnapshotBuffer();
  private joined = false;
  private seq = 0;
  private snapshotCount = 0;
  private messages = 0;
  private messagesSent = 0;

  private snapshotListeners = new Set<(state: BufferedWorldState) => void>();
  private eventListeners = new Set<(event: WorldEvent) => void>();
  private rejectListeners = new Set<(reject: WorldRejectPayload) => void>();
  private transferListeners = new Set<(transfer: WorldTransferPayload) => void>();
  private resyncListeners = new Set<(reason: string) => void>();
  private errorListeners = new Set<(error: Error) => void>();

  constructor(welcomePayload: WorldWelcomePayload = welcome()) {
    this.welcomePayload = welcomePayload;
  }

  public connect(join: WorldJoinPayload): Promise<WorldWelcomePayload> {
    this.joins.push(join);
    if (this.failConnect) return Promise.reject(this.failConnect);
    this.buffer.apply(this.welcomePayload.snapshot);
    this.joined = true;
    return Promise.resolve(this.welcomePayload);
  }

  public isJoined(): boolean { return this.joined; }
  public getWelcome(): WorldWelcomePayload | null { return this.joined ? this.welcomePayload : null; }
  public getSnapshot(): BufferedWorldState | null { return this.buffer.getState(); }

  public sendInput(move: { x: number; y: number } = { x: 0, y: 0 }): number | null {
    this.inputs.push(move);
    this.messagesSent += 1;
    return this.seq++;
  }

  public sendMoveTarget(target: { x: number; y: number }): number | null {
    this.moveTargets.push(target);
    this.messagesSent += 1;
    return this.seq++;
  }

  public sendCast(cast: Omit<CastPayload, "seq">): number | null {
    this.casts.push(cast);
    this.messagesSent += 1;
    return this.seq++;
  }

  public sendBasicAttack(attack: Omit<BasicAttackPayload, "seq">): number | null {
    this.attacks.push(attack);
    this.messagesSent += 1;
    return this.seq++;
  }

  public sendCapture(targetId: string, guId?: string): number | null {
    this.captures.push({ targetId, guId });
    this.messagesSent += 1;
    return this.seq++;
  }

  public sendCompanionStance(stance: "DEFEND" | "FOCUS" | "FOLLOW", focusTargetId?: string): number | null {
    this.stances.push({ stance, focusTargetId });
    this.messagesSent += 1;
    return this.seq++;
  }

  public emitSnapshot(payload: WorldSnapshotPayload): void {
    this.messages += 1;
    const result = this.buffer.apply(payload);
    if (result.status === "applied") {
      this.snapshotCount += 1;
      for (const listener of this.snapshotListeners) listener(result.state);
      return;
    }
    if (result.status === "resync_required") {
      for (const listener of this.resyncListeners) listener(result.reason);
    }
  }

  public emitEvent(event: WorldEvent): void {
    this.messages += 1;
    for (const listener of this.eventListeners) listener(event);
  }

  public emitError(error: Error): void {
    for (const listener of this.errorListeners) listener(error);
  }

  public emitTransfer(transfer: WorldTransferPayload): void {
    for (const listener of this.transferListeners) listener(transfer);
  }

  public onEvents(callback: (event: WorldEvent) => void): () => void {
    this.eventListeners.add(callback);
    return () => this.eventListeners.delete(callback);
  }

  public onSnapshot(callback: (state: BufferedWorldState) => void): () => void {
    this.snapshotListeners.add(callback);
    return () => this.snapshotListeners.delete(callback);
  }

  public onReject(callback: (reject: WorldRejectPayload) => void): () => void {
    this.rejectListeners.add(callback);
    return () => this.rejectListeners.delete(callback);
  }

  public onTransfer(callback: (transfer: WorldTransferPayload) => void): () => void {
    this.transferListeners.add(callback);
    return () => this.transferListeners.delete(callback);
  }

  public onResyncRequired(callback: (reason: string) => void): () => void {
    this.resyncListeners.add(callback);
    return () => this.resyncListeners.delete(callback);
  }

  public onError(callback: (error: Error) => void): () => void {
    this.errorListeners.add(callback);
    return () => this.errorListeners.delete(callback);
  }

  public disconnect(): void {
    this.disconnectCount += 1;
    this.joined = false;
  }

  public getMetrics() {
    return {
      messages: this.messages,
      bytes: 0,
      messagesSent: this.messagesSent,
      bytesSent: 0,
      snapshots: this.snapshotCount,
      lastMessageAt: 0,
      lastSnapshotAt: 0,
      lastCloseCode: 0,
    };
  }
}

test("WorldRuntime joins, predicts movement and handles click targets", async () => {
  const fake = new FakeTransport();
  const runtime = new WorldRuntime({ transportFactory: () => fake });

  const result = await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });

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
  assert.ok(fake.inputs.length >= 1);

  assert.equal(runtime.sendMoveTarget({ x: 12, y: 10 }), true);
  assert.deepEqual(fake.moveTargets.at(-1), { x: 12, y: 10 });
  assert.deepEqual(runtime.getMoveTarget(), { x: 12, y: 10 });

  runtime.disconnect();
  assert.equal(runtime.getStatus(), "closed");
});

test("dash snapshots control self position until DASH_ENDED", async () => {
  const fake = new FakeTransport();
  const runtime = new WorldRuntime({ transportFactory: () => fake });
  await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });

  runtime.setMoveVector({ x: 0, y: 1 });
  fake.emitEvent({ event_id: 1, tick: 10, type: "CAST_ACCEPTED", source_id: "player-1", payload: { skill_id: "DASH", duration_ticks: 0 } });
  runtime.update(0.1);
  assert.deepEqual(runtime.getSelfPosition(), { x: 10, y: 10 }, "ordinary movement predicted during dash");
  fake.emitSnapshot({
    tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.7, y: 10, hp: 80 }],
    removed: [],
  });
  assert.deepEqual(runtime.getSelfPosition(), { x: 10.7, y: 10 }, "dash did not follow authoritative snapshot");
  fake.emitEvent({ event_id: 2, tick: 11, type: "DASH_ENDED", source_id: "player-1", payload: { skill_id: "DASH" } });
  runtime.update(0.1);
  assert.ok(runtime.getSelfPosition().y > 10, "ordinary prediction did not resume after dash");
  runtime.disconnect();
});

test("held movement does not step backward when a delayed snapshot arrives", async () => {
  const fake = new FakeTransport();
  let now = 0;
  const runtime = new WorldRuntime({ nowMs: () => now, transportFactory: () => fake });
  await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });

  runtime.setMoveVector({ x: 1, y: 0 });
  runtime.update(0.1);
  now = 100;
  const before = runtime.getSelfPosition().x;
  fake.emitSnapshot({
    tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.28, y: 10, hp: 80 }],
    removed: [],
  });
  assert.ok(runtime.getSelfPosition().x >= before, `snapshot pulled player backward from ${before} to ${runtime.getSelfPosition().x}`);
  runtime.setMoveVector({ x: 0, y: 0 });
  fake.emitSnapshot({
    tick: 12, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 2, x: before, y: 10, hp: 80 }],
    removed: [],
  });
  assert.ok(runtime.getSelfPosition().x >= before, "snapshot pulled player back before the stop input was acknowledged");
  fake.emitSnapshot({
    tick: 13, ack_seq: 2, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 3, x: before, y: 10, hp: 80 }],
    removed: [],
  });
  assert.ok(Math.abs(runtime.getSelfPosition().x - before) < 0.1);
  runtime.disconnect();
});

test("server blocked and slowed movement still corrects local prediction", async () => {
  for (const serverStep of [0, 0.14]) {
    const fake = new FakeTransport();
    let now = 0;
    const runtime = new WorldRuntime({ nowMs: () => now, transportFactory: () => fake });
    await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
    runtime.setMoveVector({ x: 1, y: 0 });
    for (let step = 1; step <= 30; step++) {
      runtime.update(0.05);
      now += 50;
      fake.emitSnapshot({
        tick: 10 + step, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
        entities: [{ entity_id: "player-1", entity_revision: step, x: 10 + serverStep * step, y: 10, hp: 80 }],
        removed: [],
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
  const fake = new FakeTransport();
  let now = 0;
  const runtime = new WorldRuntime({ nowMs: () => now, transportFactory: () => fake });
  await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  assert.equal(runtime.sendMoveTarget({ x: 10.56, y: 10 }), true);
  runtime.update(0.1);
  const before = runtime.getSelfPosition().x;
  now = 100;
  fake.emitSnapshot({
    tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 1, x: 10.28, y: 10, hp: 80 }],
    removed: [],
  });
  assert.ok(runtime.getSelfPosition().x >= before - 0.001);
  fake.emitSnapshot({
    tick: 12, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 2, x: 10.56, y: 10, hp: 80 }],
    removed: [],
  });
  assert.ok(Math.abs(runtime.getSelfPosition().x - 10.56) < 0.001);
  runtime.disconnect();
});

test("turning away from a wall resumes immediate local movement", async () => {
  const fake = new FakeTransport();
  const runtime = new WorldRuntime({ transportFactory: () => fake });
  await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  runtime.setMoveVector({ x: 1, y: 0 });
  runtime.update(0.05);
  fake.emitSnapshot({
    tick: 11, ack_seq: 1, baseline_id: 1, base_tick: 10, is_baseline: false,
    entities: [{ entity_id: "player-1", entity_revision: 1, x: 10, y: 10, hp: 80 }],
    removed: [],
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
  const transports: FakeTransport[] = [];
  const runtime = new WorldRuntime({
    transportFactory: () => {
      const transport = new FakeTransport();
      if (transports.length === 0) {
        transport.failConnect = new WorldRealtimeError("Character already has an active realtime controller", "realtime_character_already_owned");
      }
      transports.push(transport);
      return transport;
    },
  });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "connecting", "a stale lease must retry instead of parking in error");
  assert.equal(runtime.getErrorCode(), "realtime_character_already_owned");

  await waitUntil(() => transports.length === 2);
  await waitUntil(() => runtime.getStatus() === "joined");
  assert.equal(runtime.getErrorCode(), "");
  runtime.disconnect();
});

test("WorldRuntime stops retrying on a non-retryable join rejection", async () => {
  const fake = new FakeTransport();
  fake.failConnect = new WorldRealtimeError("Requested map does not match the character's current region", "realtime_map_mismatch");
  const runtime = new WorldRuntime({ transportFactory: () => fake });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "error");
  assert.equal(runtime.getErrorCode(), "realtime_map_mismatch");
  runtime.disconnect();
});

test("disconnect cancels a scheduled join retry", async () => {
  const transports: FakeTransport[] = [];
  const runtime = new WorldRuntime({
    transportFactory: () => {
      const transport = new FakeTransport();
      transport.failConnect = new WorldRealtimeError("busy", "realtime_character_already_owned");
      transports.push(transport);
      return transport;
    },
  });

  const pending = runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" });
  assert.equal(await pending, null);
  assert.equal(runtime.getStatus(), "connecting");

  runtime.disconnect();
  assert.equal(runtime.getStatus(), "closed");

  await new Promise((resolve) => setTimeout(resolve, 600));
  assert.equal(transports.length, 1, "a disconnected runtime must not open another transport");
  assert.equal(runtime.getStatus(), "closed");
});

test("unsupported WebTransport surfaces a terminal error without a WS fallback", async () => {
  const fake = new FakeTransport();
  fake.failConnect = new WorldRealtimeError("Trình duyệt không hỗ trợ WebTransport", "webtransport_unsupported");
  const runtime = new WorldRuntime({ transportFactory: () => fake });

  assert.equal(await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" }), null);
  assert.equal(runtime.getStatus(), "error");
  assert.equal(runtime.getErrorCode(), "webtransport_unsupported");
  runtime.disconnect();
});

test("application rejections surface the admission code without another transport", async () => {
  for (const code of ["unauthorized", "session_replaced", "realtime_map_mismatch"]) {
    const fake = new FakeTransport();
    fake.failConnect = new WorldRealtimeError("Rejected", code);
    const runtime = new WorldRuntime({ transportFactory: () => fake });
    assert.equal(await runtime.connect({ zone_id: "starter", map_version: "map-1", content_version: "content-1" }), null);
    assert.equal(runtime.getStatus(), "error");
    assert.equal(runtime.getErrorCode(), code);
    runtime.disconnect();
  }
});
