import assert from "node:assert/strict";
import test from "node:test";
import { legacyMonsterHasRuntimeReplacement, reduceWorldCombatEvent, type WorldCombatEvent, type WorldMonster, type WorldMonsterReducerState } from "./worldMonsterReducer.ts";

const monster = (overrides: Partial<WorldMonster> = {}): WorldMonster => ({
  instance_id: "instance-1", spawn_id: "spawn-1", monster_template_id: "wolf", region_id: "field",
  x: 1, y: 2, spawn_x: 1, spawn_y: 2, hp: 100, max_hp: 100, state: "IDLE", revision: 4,
  last_update_at: "2026-01-01T00:00:00Z", ...overrides,
});
const event = (overrides: Partial<WorldCombatEvent> = {}): WorldCombatEvent => ({
  event_id: "event-1", server_time: "2026-01-01T00:00:01Z", region_id: "field", entity_id: "instance-1",
  entity_revision: 5, event_type: "monster_moved", position: { x: 3, y: 4 },
  payload: { state: "CHASE", target_id: "player-1", hp: 100, max_hp: 100 }, ...overrides,
});
const initial = (): WorldMonsterReducerState => ({
  monsters: { "instance-1": monster() }, seenCombatEvents: [], selectedTarget: null, targetingGu: null,
});

test("ignores duplicate events and stale or equal revisions", () => {
  const once = reduceWorldCombatEvent(initial(), event());
  const duplicate = reduceWorldCombatEvent(once, event({ position: { x: 99, y: 99 } }));
  assert.equal(duplicate, once);
  const stale = reduceWorldCombatEvent(once, event({ event_id: "event-2", entity_revision: 5, position: { x: 88, y: 88 } }));
  assert.deepEqual(stale.monsters["instance-1"], once.monsters["instance-1"]);
});

test("orders movement by revision and applies authoritative state without predicting damage", () => {
  const moved = reduceWorldCombatEvent(initial(), event({ entity_revision: 6, payload: { state: "CHASE", target_id: "player-1" } }));
  assert.deepEqual({ x: moved.monsters["instance-1"].x, y: moved.monsters["instance-1"].y }, { x: 3, y: 4 });
  assert.equal(moved.monsters["instance-1"].state, "CHASE");
  assert.equal(moved.monsters["instance-1"].hp, 100);
  const oldMovement = reduceWorldCombatEvent(moved, event({ event_id: "event-old", entity_revision: 5, position: { x: 9, y: 9 } }));
  assert.equal(oldMovement.monsters["instance-1"].x, 3);
});

test("defeat removes the exact instance and clears selection and pending cast", () => {
  const state = initial();
  state.selectedTarget = { instanceId: "instance-1", x: 1, y: 2, hp: 100, maxHp: 100 };
  state.targetingGu = { guInstanceId: "gu-1" };
  const defeated = reduceWorldCombatEvent(state, event({ event_type: "monster_defeated" }));
  assert.deepEqual(defeated.monsters, {});
  assert.equal(defeated.selectedTarget, null);
  assert.equal(defeated.targetingGu, null);
});

test("respawn replaces the old spawn instance and clears its selection", () => {
  const state = initial();
  state.selectedTarget = { instanceId: "instance-1", x: 1, y: 2, hp: 100, maxHp: 100 };
  state.targetingGu = { guInstanceId: "gu-1" };
  const replacement = monster({ instance_id: "instance-2", revision: 1 });
  const respawned = reduceWorldCombatEvent(state, event({
    event_id: "respawn", entity_id: "instance-2", entity_revision: 1, event_type: "monster_respawned",
    payload: { entity: replacement, state: "IDLE", hp: 100, max_hp: 100, previous_instance_id: "instance-1" },
  }));
  assert.equal(respawned.monsters["instance-1"], undefined);
  assert.equal(respawned.monsters["instance-2"].state, "IDLE");
  assert.equal(respawned.selectedTarget, null);
  assert.equal(respawned.targetingGu, null);
});

test("an empty runtime snapshot does not hide procedural cell monsters", () => {
  assert.equal(legacyMonsterHasRuntimeReplacement({ x: 10, y: 12 }, {}), false);
  assert.equal(legacyMonsterHasRuntimeReplacement({ x: 10, y: 12 }, {
    "instance-1": monster({ x: 10, y: 12, spawn_x: 10, spawn_y: 12 }),
  }), true);
  assert.equal(legacyMonsterHasRuntimeReplacement({ x: 10, y: 12 }, {
    "instance-1": monster({ x: 11, y: 12, spawn_x: 11, spawn_y: 12 }),
  }), false);
  assert.equal(legacyMonsterHasRuntimeReplacement({ x: 10, y: 12 }, {
    "instance-1": monster({ x: 11, y: 12, spawn_x: 10, spawn_y: 12 }),
  }), true);
});

test("records distinct informational combat events at the same entity revision", () => {
  const started = reduceWorldCombatEvent(initial(), event({ event_id: "started", event_type: "action_started" }));
  const resolved = reduceWorldCombatEvent(started, event({ event_id: "resolved", event_type: "action_resolved" }));
  const damaged = reduceWorldCombatEvent(resolved, event({ event_id: "damaged", event_type: "entity_damaged" }));
  assert.deepEqual(damaged.seenCombatEvents, ["started", "resolved", "damaged"]);
});
