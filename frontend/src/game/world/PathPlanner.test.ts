import assert from "node:assert/strict";
import test from "node:test";
import { findPath, lineClear } from "./PathPlanner.ts";
import { parseWorldMap } from "./MapLoader.ts";

function mapWith(solid: boolean[]): ReturnType<typeof parseWorldMap> {
  return parseWorldMap({
    orientation: "orthogonal",
    width: 10,
    height: 10,
    tilewidth: 32,
    tileheight: 32,
    map_id: "path-test",
    map_version: "v1",
    layers: [
      { type: "tilelayer", name: "collision", data: solid.map((value) => (value ? 1 : 0)) },
    ],
  });
}

function border(): boolean[] {
  const solid = new Array(100).fill(false);
  for (let i = 0; i < 10; i += 1) {
    solid[i] = true;
    solid[90 + i] = true;
    solid[i * 10] = true;
    solid[i * 10 + 9] = true;
  }
  return solid;
}

test("PathPlanner returns a direct path across open ground", () => {
  const map = mapWith(border());
  const path = findPath(map, { x: 2.5, y: 5.5 }, { x: 7.5, y: 5.5 });
  assert.ok(path);
  assert.deepEqual(path, [{ x: 7.5, y: 5.5 }]);
  assert.equal(lineClear(map, { x: 2.5, y: 5.5 }, { x: 7.5, y: 5.5 }), true);
});

test("PathPlanner routes around an obstacle and avoids solid cells", () => {
  const solid = border();
  for (let y = 1; y <= 8; y += 1) {
    if (y === 3) continue;
    solid[y * 10 + 5] = true;
  }
  const map = mapWith(solid);
  assert.equal(lineClear(map, { x: 2.5, y: 5.5 }, { x: 8.5, y: 5.5 }), false);
  const path = findPath(map, { x: 2.5, y: 5.5 }, { x: 8.5, y: 5.5 });
  assert.ok(path);
  assert.ok(path.length > 1);
  for (const point of path) {
    assert.equal(map.isSolid(point.x, point.y), false);
  }
  assert.deepEqual(path[0], { x: 2.5, y: 5.5 });
  assert.deepEqual(path[path.length - 1], { x: 8.5, y: 5.5 });
});

test("PathPlanner returns null for blocked or out-of-bounds goals", () => {
  const solid = border();
  for (let y = 1; y <= 8; y += 1) solid[y * 10 + 5] = true;
  const map = mapWith(solid);
  assert.equal(findPath(map, { x: 2.5, y: 5.5 }, { x: 8.5, y: 5.5 }), null);
  assert.equal(findPath(map, { x: 2.5, y: 5.5 }, { x: 99, y: 99 }), null);
  assert.equal(findPath(map, { x: 2.5, y: 5.5 }, { x: 5.5, y: 5.5 }), null);
});
