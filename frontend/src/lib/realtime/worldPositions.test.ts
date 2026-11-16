import assert from "node:assert/strict";
import test from "node:test";
import { applyWorldPosition, type WorldMapPlayerSnapshot, type WorldPositionEvent } from "./worldPositions.ts";

const snapshot: WorldMapPlayerSnapshot = {
  x: 10,
  y: 10,
  vision_radius: 2,
  world_map_id: "world_map",
  map_instance_id: "",
  players_on_map: [{ id: "other", username: "Other", avatar_url: "", x: 9, y: 10 }],
};

const event: WorldPositionEvent = {
  id: "other",
  username: "Other",
  avatar_url: "avatar.png",
  x: 12,
  y: 11,
  region_id: "north",
  world_map_id: "world_map",
  ts: 100,
};

test("world position delta replaces an existing map marker immediately", () => {
  const updated = applyWorldPosition(snapshot, event, "local");
  assert.deepEqual(updated.players_on_map, [{ id: "other", username: "Other", avatar_url: "avatar.png", x: 12, y: 11 }]);
});

test("world position delta removes a marker that moves outside the viewport", () => {
  const updated = applyWorldPosition(snapshot, { ...event, x: 30 }, "local");
  assert.deepEqual(updated.players_on_map, []);
});

test("world position delta ignores another map or instance", () => {
  assert.equal(applyWorldPosition(snapshot, { ...event, world_map_id: "other_map" }, "local"), snapshot);
  assert.equal(applyWorldPosition(snapshot, { ...event, map_instance_id: "private" }, "local"), snapshot);
});

test("world position leave removes the marker from its old scope", () => {
  const updated = applyWorldPosition(snapshot, { ...event, action: "left" }, "local");
  assert.deepEqual(updated.players_on_map, []);
});
