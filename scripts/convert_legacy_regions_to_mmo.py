#!/usr/bin/env python3
"""Compile legacy Region/MapCell/MonsterSpawn CSVs into the shared MMO world map.

MonsterSpawn rows without coordinates are placed deterministically inside their
legacy region and counted in the generated report. No terrain is invented;
only the world edge is blocked because the source CSV contains no collision.
"""
import csv
import hashlib
import json
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend/data"
WIDTH, HEIGHT, SCALE = 1200, 800, 25
# The largest server monster archetype has radius 0.8. Keep map-edge spawns at
# least one full tile from the border wall so the collision circle stays clear.
MONSTER_EDGE_MARGIN = 2.0


def rows(name):
    with (DATA / name).open(encoding="utf-8-sig", newline="") as f:
        return list(csv.DictReader(f))


def truth(value):
    return str(value).strip().lower() in {"true", "1", "yes"}


def num(row, key, default=0):
    try:
        return float(row.get(key) or default)
    except (TypeError, ValueError):
        return float(default)


def stable_point(region, monster, index, bounds):
    min_x, min_y, max_x, max_y = bounds
    digest = hashlib.sha256(f"{region}:{monster}:{index}".encode()).digest()
    # Keep a margin from the region boundary so approximate spawns remain in it.
    x_range = max(1, int(max_x - min_x - 2))
    y_range = max(1, int(max_y - min_y - 2))
    x = min_x + 1 + int.from_bytes(digest[:4], "big") % x_range
    y = min_y + 1 + int.from_bytes(digest[4:8], "big") % y_range
    return min(WIDTH - 2, max(1, round(x / SCALE, 2))), min(HEIGHT - 2, max(1, round(y / SCALE, 2)))


def prop(name, value):
    return {"name": name, "value": value}


def main():
    regions = [r for r in rows("Region.csv") if truth(r.get("is_active", "true"))]
    region_by_id = {r["id"]: r for r in regions}
    region_objects = []
    player_spawns = []
    for region in regions:
        x, y = num(region, "min_x"), num(region, "min_y")
        right, bottom = num(region, "max_x", x + num(region, "width")), num(region, "max_y", y + num(region, "height"))
        # Some historical rows only have width/height.
        if right <= x: right = x + num(region, "width")
        if bottom <= y: bottom = y + num(region, "height")
        sx = min(WIDTH - 1, max(0, round((right - x) / SCALE)))
        sy = min(HEIGHT - 1, max(0, round((bottom - y) / SCALE)))
        rx, ry = max(0, round(x / SCALE)), max(0, round(y / SCALE))
        sx, sy = min(sx, WIDTH - rx), min(sy, HEIGHT - ry)
        if sx <= 0 or sy <= 0: continue
        region_objects.append({"name": region.get("name", region["id"]), "type": "region", "x": rx, "y": ry,
            "width": sx, "height": sy, "properties": [prop("zone_type", region.get("zone_type", "KHU_VUC")), prop("region_id", region["id"])]})
        ex, ey = num(region, "entry_x", x + (right - x) / 2), num(region, "entry_y", y + (bottom - y) / 2)
        player_spawns.append({"name": "entry_" + region["id"], "type": "player", "x": min(WIDTH - 2, max(1, round(ex / SCALE, 2))), "y": min(HEIGHT - 2, max(1, round(ey / SCALE, 2)))})

    pois = []
    cell_rows = [r for r in rows("MapCell.csv") if truth(r.get("is_active", "true"))]
    for index, cell in enumerate(cell_rows):
        kind = (cell.get("cell_type") or "").strip()
        if kind.lower() not in {"boss", "monster", "entry_portal", "exit_portal", "inheritance_altar"}: continue
        x, y = num(cell, "x") / SCALE, num(cell, "y") / SCALE
        pois.append({"name": cell.get("content_id") or kind, "type": kind, "x": round(x, 2), "y": round(y, 2),
                     "properties": [prop("region_id", cell.get("region_id", "")), prop("content_id", cell.get("content_id", ""))]})

    spawn_rows = [r for r in rows("MonsterSpawn.csv") if truth(r.get("is_active", "true"))]
    buckets = {}
    approximated = 0
    for index, spawn in enumerate(spawn_rows):
        region_id = spawn.get("region_id", "")
        region = region_by_id.get(region_id)
        if region:
            bounds = (num(region, "min_x"), num(region, "min_y"), num(region, "max_x", num(region, "min_x") + num(region, "width")), num(region, "max_y", num(region, "min_y") + num(region, "height")))
        else:
            bounds = (0, 0, WIDTH * SCALE, HEIGHT * SCALE)
        if spawn.get("x") and spawn.get("y"):
            x, y = min(WIDTH - 2, max(1, num(spawn, "x") / SCALE)), min(HEIGHT - 2, max(1, num(spawn, "y") / SCALE))
        else:
            x, y = stable_point(region_id, spawn.get("monster_id", ""), index, bounds)
            approximated += 1
        # A point can be in a walkable tile while a large monster's collision
        # radius still overlaps the solid outer border. Keep a safe margin for
        # every archetype, including Boss (radius 0.8).
        x = min(WIDTH - MONSTER_EDGE_MARGIN, max(MONSTER_EDGE_MARGIN, x))
        y = min(HEIGHT - MONSTER_EDGE_MARGIN, max(MONSTER_EDGE_MARGIN, y))
        obj = {"name": f"{region_id}_{spawn.get('monster_id','monster')}_{index}", "type": "monster", "x": round(x, 2), "y": round(y, 2),
               "properties": [prop("monster_id", spawn.get("monster_id", "")), prop("region_id", region_id),
                             prop("source_spawn_mode", spawn.get("spawn_mode", "")), prop("legacy_approximate", not bool(spawn.get("x") and spawn.get("y"))),
                             prop("is_boss_spawn", truth(spawn.get("is_boss_spawn", "false")))]}
        buckets.setdefault(region_id, []).append(obj)
    # Round-robin across legacy regions so the 500-monster runtime cap doesn't
    # leave entire regions empty.
    selected = []
    while len(selected) < 500:
        progressed = False
        for region_id in sorted(buckets):
            bucket = buckets[region_id]
            if bucket:
                selected.append(bucket.pop(0)); progressed = True
                if len(selected) == 500: break
        if not progressed: break

    edge = [0] * (WIDTH * HEIGHT)
    for x in range(WIDTH):
        edge[x] = 1
        edge[(HEIGHT - 1) * WIDTH + x] = 1
    for y in range(HEIGHT):
        edge[y * WIDTH] = 1
        edge[y * WIDTH + WIDTH - 1] = 1
    data = {
        "type": "map", "orientation": "orthogonal", "renderorder": "right-down", "width": WIDTH, "height": HEIGHT,
        "tilewidth": 24, "tileheight": 24, "map_id": "world_map", "map_version": "legacy-converted-v2", "content_version": "legacy-regions-v2",
        "properties": [prop("ruleset", "pve"), prop("legacy_origin_x", 0), prop("legacy_origin_y", 0), prop("legacy_units_per_world_unit", SCALE), prop("legacy_source", "Region.csv + MapCell.csv + MonsterSpawn.csv")],
        "layers": [
            {"type": "tilelayer", "name": "ground", "width": WIDTH, "height": HEIGHT, "data": [0] * (WIDTH * HEIGHT)},
            {"type": "tilelayer", "name": "collision", "width": WIDTH, "height": HEIGHT, "data": edge},
            {"type": "objectgroup", "name": "spawns", "objects": player_spawns + selected},
            {"type": "objectgroup", "name": "legacy_regions", "objects": region_objects},
            {"type": "objectgroup", "name": "legacy_pois", "objects": pois},
        ],
    }
    out_dirs = [ROOT / "backend/data/mmo/maps", ROOT / "frontend/public/game/maps"]
    encoded = json.dumps(data, ensure_ascii=False, separators=(",", ":")) + "\n"
    for directory in out_dirs:
        directory.mkdir(parents=True, exist_ok=True)
        (directory / "world_map.json").write_text(encoded, encoding="utf-8")
    report = {"source_regions": len(regions), "region_boundaries": len(region_objects), "map_cells": len(cell_rows),
              "monster_spawn_rows": len(spawn_rows), "monster_spawns_exported": len(selected), "monster_spawns_omitted_runtime_cap": len(spawn_rows) - len(selected),
              "spawn_positions_approximated": approximated, "collision_source": "none; only outer world boundary is blocked",
              "monster_spawn_edge_margin_world_units": MONSTER_EDGE_MARGIN,
              "world_map": f"{WIDTH}x{HEIGHT}", "legacy_units_per_world_unit": SCALE, "artifact_bytes": len(encoded.encode())}
    (ROOT / "docs/mmo/legacy-map-conversion-report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
