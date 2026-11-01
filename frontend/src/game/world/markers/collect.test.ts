import assert from "node:assert/strict";
import test from "node:test";
import { collectMarkers, findNearestMarker } from "./collect.ts";

test("sects and teleports come from the zone feed with normalised tone", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    zoneMarkers: [
      {
        id: "sect-a",
        kind: "sect",
        region_id: "bach_cot_son",
        x: 120,
        y: 240,
        name: "Thanh Vân Tông",
        detail: { level: 3, buildings: { main_hall: 4 }, shield_until: null, is_mine: true },
      },
      {
        id: "tp-a",
        kind: "teleport",
        region_id: "bach_cot_son",
        x: 300,
        y: 400,
        name: "Truyền Tống Trận",
        detail: { tier: "TIER_2", access_policy: "PUBLIC" },
      },
    ],
  });

  const sect = markers.find((marker) => marker.kind === "sect");
  assert.ok(sect);
  assert.equal(sect.tone, "self");
  assert.equal(sect.radiusCells, 9);
  assert.equal(sect.interaction?.label, "Mở tông môn");
  assert.equal(sect.sect?.level, 3);

  const teleport = markers.find((marker) => marker.kind === "teleport");
  assert.ok(teleport);
  assert.equal(teleport.tone, "neutral");
  assert.equal(teleport.interaction?.action.type, "open-teleport");
});

test("enemy sects are hostile and shielded sects keep their shield timestamp", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    zoneMarkers: [
      {
        id: "sect-b",
        kind: "sect",
        region_id: "bach_cot_son",
        x: 10,
        y: 10,
        name: "Huyết Ma Giáo",
        detail: { level: 2, buildings: {}, shield_until: "2026-01-01T00:00:00Z" },
      },
    ],
  });

  const sect = markers[0];
  assert.equal(sect.tone, "enemy");
  assert.equal(sect.interaction?.action.type, "attack-sect");
  assert.equal(sect.sect?.shieldUntil, Date.parse("2026-01-01T00:00:00Z"));
});

test("veins, drops and portals are collected from their own transports", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    cells: [
      {
        x: 40,
        y: 50,
        cell_type: "PrimevalVein",
        content: "vein-1",
        vein: { id: "vein-1", vein_type: "immortal", rarity: "Hiếm", accumulated_stones: 12 },
      },
    ],
    groundDrops: [{ id: "drop-1", x: 80, y: 90, killer_name: "Sát Thủ" }],
    mapPortals: [{ name: "", targetZoneId: "tan_thu_coc_nam_cuong", x: 5, y: 6 }],
  });

  const vein = markers.find((marker) => marker.kind === "vein");
  assert.ok(vein);
  assert.equal(vein.name, "Tiên Tuyền");
  assert.equal(vein.tone, "rare");
  assert.equal(vein.interaction?.action.targetId, "vein-1");

  const drop = markers.find((marker) => marker.kind === "drop");
  assert.ok(drop);
  assert.equal(drop.name, "Sát Thủ");
  assert.equal(drop.interaction?.radiusCells, 3.2);

  const portal = markers.find((marker) => marker.kind === "portal");
  assert.ok(portal);
  assert.equal(portal.name, "tan thu coc nam cuong");
  assert.equal(portal.interaction, undefined);
});

test("a teleport array present in both feeds is drawn once", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    zoneMarkers: [
      { id: "tp-a", kind: "teleport", region_id: "bach_cot_son", x: 30, y: 30, name: "Trận A", detail: {} },
    ],
    cells: [{ x: 30, y: 30, cell_type: "TeleportArray", content: "tp-a" }],
  });

  assert.equal(markers.filter((marker) => marker.kind === "teleport").length, 1);
  assert.equal(markers[0].interaction?.action.targetId, "tp-a");
});

test("markers from the previous region do not appear in a private map", () => {
  const markers = collectMarkers({
    regionId: "aperture_tribulation",
    cells: [{ x: 50, y: 50, region_id: "tan_thu_coc_nam_cuong", cell_type: "TeleportArray", content: "old-teleport" }],
    zoneMarkers: [{ id: "old-structure", kind: "teleport", region_id: "tan_thu_coc_nam_cuong", x: 51, y: 51, name: "Old structure", detail: {} }],
    groundDrops: [{ id: "old-drop", x: 52, y: 52, region_id: "tan_thu_coc_nam_cuong", killer_name: "" }],
  });

  assert.deepEqual(markers, []);
});

test("findNearestMarker respects per-kind interaction ranges", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    groundDrops: [{ id: "far", x: 4, y: 0, killer_name: "" }],
    cells: [
      {
        x: 3,
        y: 0,
        cell_type: "PrimevalVein",
        content: "vein-1",
        vein: { id: "vein-1", vein_type: "mortal", rarity: "Thường", accumulated_stones: 1 },
      },
    ],
  });

  assert.equal(findNearestMarker(markers, { x: 0, y: 0 }, "drop"), null);
  assert.equal(findNearestMarker(markers, { x: 0, y: 0 }, "vein")?.id, "vein-1");
  assert.equal(findNearestMarker(markers, { x: 0, y: 0 }, "drop"), null);
});

test("world resource points become mine markers within the 6-cell range", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    resourcePoints: [
      {
        id: "point-1",
        region_id: "bach_cot_son",
        x: 10,
        y: 10,
        name: "Thung Lũng Linh Thảo",
        grade: "LOW",
        point_class: "NORMAL",
        state: "ACTIVE",
        vitality: 80,
        stock_total: 12,
        stock_cap: 40,
      },
      {
        id: "point-2",
        region_id: "bach_cot_son",
        x: 20,
        y: 20,
        name: "Núi Đãng Hồn",
        grade: "TOP",
        point_class: "FAMOUS",
        state: "EXHAUSTED",
        vitality: 0,
        stock_total: 0,
        stock_cap: 60,
      },
    ],
  });

  const normal = markers.find((marker) => marker.id === "point-1");
  assert.ok(normal);
  assert.equal(normal.kind, "resource");
  assert.equal(normal.tone, "neutral");
  assert.equal(normal.interaction?.action.type, "harvest-resource");
  assert.equal(normal.interaction?.radiusCells, 6);
  assert.equal(normal.interaction?.label, "Khai thác");

  const famous = markers.find((marker) => marker.id === "point-2");
  assert.ok(famous);
  assert.equal(famous.tone, "rare");
  assert.equal(famous.interaction?.label, "Chưa có tài nguyên");
});

test("resource markers from another region are filtered out", () => {
  const markers = collectMarkers({
    regionId: "bach_cot_son",
    resourcePoints: [
      {
        id: "point-3",
        region_id: "am_hon_sa_vuc",
        x: 1,
        y: 1,
        name: "Mạch Quặng",
        grade: "MID",
        point_class: "NORMAL",
        state: "ACTIVE",
        vitality: 100,
        stock_total: 5,
        stock_cap: 30,
      },
    ],
  });
  assert.deepEqual(markers, []);
});
