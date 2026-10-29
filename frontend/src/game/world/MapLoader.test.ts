import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import test from "node:test";
import { LoadedWorldMap, computeWorldMapHash, parseWorldMap } from "./MapLoader.ts";

const fixtureUrl = new URL("../../../public/game/maps/realtime_preview.json", import.meta.url);
const mapsDir = new URL("../../../public/game/maps/", import.meta.url);

test("all alpha maps parse and link to known maps", () => {
  const maps = new Map<string, LoadedWorldMap>();
  for (const file of readdirSync(mapsDir).filter((name) => name.endsWith(".json"))) {
    const map = parseWorldMap(JSON.parse(readFileSync(new URL(file, mapsDir), "utf8")));
    assert.equal(`${map.id}.json`, file);
    assert.equal(maps.has(map.id), false, `duplicate map ${map.id}`);
    maps.set(map.id, map);
  }
  assert.ok(maps.size >= 4);
  for (const map of maps.values()) {
    assert.ok(map.spawns.length > 0, `${map.id} has no spawns`);
    assert.ok(map.spawns.every((spawn) => map.isSpawnFree(spawn)), `${map.id} has a solid spawn`);
    for (const portal of map.portals) {
      const target = maps.get(portal.targetZoneId);
      assert.ok(target, `${map.id} portal ${portal.name} targets ${portal.targetZoneId}`);
      assert.ok(portal.targetX < target.width && portal.targetY < target.height);
      assert.equal(target.isSolid(portal.targetX, portal.targetY), false, `${map.id} portal target is solid`);
    }
  }
});

test("computeWorldMapHash matches the Go nav loader", async () => {
  const map = parseWorldMap(JSON.parse(readFileSync(fixtureUrl, "utf8")));
  const hash = await computeWorldMapHash(map);
  assert.equal(hash, "7fa20da7a1961930c87bf793b9514bb125c5a045afadb58629f60d4ffbf76e7a");
});

test("parseWorldMap loads the shared preview map and identifies blockers", () => {
  const map = parseWorldMap(JSON.parse(readFileSync(fixtureUrl, "utf8")));
  assert.equal(map.id, "realtime_preview");
  assert.equal(map.version, "regions-v1");
  assert.equal(map.width, 64);
  assert.equal(map.height, 48);
  assert.equal(map.pixelWidth, 64 * 24);
  assert.equal(map.isSolid(0.5, 0.5), true);
  assert.equal(map.isSolid(-1, 5), true);
  assert.equal(map.isSolid(500, 5), true);
  assert.equal(map.isSolid(8.5, 8.5), false);
  assert.ok(map.spawns.length >= 3);
  assert.ok(map.spawns.every((spawn) => map.isSpawnFree(spawn)));
  assert.equal(map.portals.length, 1);
  assert.equal(map.portals[0].targetZoneId, "realtime_preview");
  assert.equal(map.portals[0].targetX, 8.5);
});

test("parseWorldMap rejects missing collision, bad cells and unsafe spawns", () => {
  const base = {
    orientation: "orthogonal",
    width: 4,
    height: 4,
    tilewidth: 32,
    tileheight: 32,
    map_id: "test",
    map_version: "v1",
    layers: [
      { type: "tilelayer", name: "collision", data: new Array(16).fill(0) },
    ],
  };
  assert.ok(parseWorldMap(base) instanceof LoadedWorldMap);

  assert.throws(
    () => parseWorldMap({ ...base, layers: [] }),
    /collision layer/,
  );
  assert.throws(
    () => parseWorldMap({ ...base, layers: [{ type: "tilelayer", name: "collision", data: [0, 1] }] }),
    /wrong size/,
  );
  assert.throws(
    () => parseWorldMap({ ...base, orientation: "isometric" }),
    /orthogonal/,
  );
  assert.throws(
    () =>
      parseWorldMap({
        ...base,
        layers: [
          ...base.layers,
          { type: "objectgroup", name: "spawns", objects: [{ type: "player", x: 99, y: 1 }] },
        ],
      }),
    /outside the map/,
  );
});

test("parseWorldMap supports procedural sparse border maps", () => {
  const map = parseWorldMap({
    orientation: "orthogonal",
    width: 2000,
    height: 2000,
    tilewidth: 24,
    tileheight: 24,
    map_id: "test_sparse",
    map_version: "v1",
    layers: [
      { type: "tilelayer", name: "ground", data: [] },
      { type: "tilelayer", name: "collision", data: [] },
      { type: "objectgroup", name: "spawns", objects: [{ type: "player", name: "spawn", x: 1000, y: 1000 }] },
    ],
  });
  assert.equal(map.width, 2000);
  assert.equal(map.height, 2000);
  assert.equal(map.pixelWidth, 2000 * 24);
  assert.equal(map.isSolid(0.5, 0.5), true);
  assert.equal(map.isSolid(1999.5, 1000), true);
  assert.equal(map.isSolid(500, 0.5), true);
  assert.equal(map.isSolid(500, 1999.5), true);
  assert.equal(map.isSolid(1000, 1000), false);
  assert.equal(map.isSolid(2, 2), false);
  assert.equal(map.isSolid(1998, 1998), false);
});

