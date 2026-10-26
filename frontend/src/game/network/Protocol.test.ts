import assert from "node:assert/strict";
import test from "node:test";
import { parseWorldSnapshotPayload, parseWorldTransferPayload, parseWorldWelcomePayload } from "./Protocol.ts";

test("parseWorldSnapshotPayload accepts floating-point mana and max_mana", () => {
  const payload = {
    tick: 100,
    ack_seq: 5,
    baseline_id: 1,
    base_tick: 100,
    is_baseline: true,
    entities: [
      {
        entity_id: "player-admin",
        entity_revision: 1,
        kind: "player",
        x: 10.5,
        y: 20.25,
        hp: 100,
        max_hp: 100,
        shield: 50.5,
        mana: 562.1754027627987,
        max_mana: 99999999999.0,
        cooldowns: [0, 0, 0, 0, 0, 0, 0],
      },
    ],
  };

  const parsed = parseWorldSnapshotPayload(payload);
  assert.equal(parsed.entities[0].mana, 562.1754027627987);
  assert.equal(parsed.entities[0].max_mana, 99999999999.0);
  assert.equal(parsed.entities[0].shield, 50.5);
});

test("parseWorldTransferPayload keeps map-local and legacy coordinates separate", () => {
  const parsed = parseWorldTransferPayload({
    from_zone_id: "thanh_mao_son",
    to_zone_id: "thiet_gia_thanh",
    target_x: 4,
    target_y: 1234,
    legacy_x: 2004,
    legacy_y: 5234,
    hp: 87.5,
  });
  assert.equal(parsed.to_zone_id, "thiet_gia_thanh");
  assert.equal(parsed.target_x, 4);
  assert.equal(parsed.legacy_x, 2004);
  assert.equal(parsed.hp, 87.5);
  assert.throws(() => parseWorldTransferPayload({ to_zone_id: "x", target_x: 1, target_y: 1, legacy_x: -1, legacy_y: 1, hp: 1 }));
});

test("parseWorldWelcomePayload validates matching owner baseline and float mana", () => {
  const welcome = {
    session_id: "sess-1",
    zone_id: "world_map#0",
    map_version: "v1",
    content_version: "v1",
    tick_rate: 20,
    server_tick: 50,
    entity_id: "player-admin",
    entity_revision: 2,
    baseline_id: 1,
    snapshot: {
      tick: 50,
      ack_seq: 1,
      baseline_id: 1,
      base_tick: 50,
      is_baseline: true,
      entities: [
        {
          entity_id: "player-admin",
          entity_revision: 2,
          kind: "player",
          x: 15,
          y: 25,
          hp: 85.5,
          mana: 42.123,
          max_mana: 100,
        },
      ],
    },
  };

  const parsed = parseWorldWelcomePayload(welcome);
  assert.equal(parsed.entity_id, "player-admin");
  assert.equal(parsed.snapshot.entities[0].mana, 42.123);
});
