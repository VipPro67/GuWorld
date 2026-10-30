#!/usr/bin/env python3
"""Compile the detailed legacy region grid into one realtime map per child region.

KHU_VUC and SAFE_ZONE become local maps. NGU_VUC stays on the shared outer
world_map. Existing Region.csv coordinates remain the legacy coordinate system;
each map stores its origin so the authoritative server can rebase player rows.
"""
import csv
import hashlib
import json
import math
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend/data"
SCALE = 1
MAP_VERSION = "regions-v1"
CONTENT_VERSION = "region-content-v1"
EDGE = 1
SPAWN_MARGIN = 2.0
# Realtime spawn budget per 1,000,000 units^2 at the reference Monster weight
# (KHU_VUC default 30/100). Region.csv pool_weights scale this up or down.
BASE_PACKS = 38
REFERENCE_MONSTER_WEIGHT = 30
# MaxMonstersPerZone in backend/internal/realtime/world/manager.go is 1200.
TOTAL_SPAWNS_CAP = 1100


def read_csv(name):
    with (DATA / name).open(encoding="utf-8-sig", newline="") as handle:
        return list(csv.DictReader(handle))


def truth(value):
    return str(value or "").strip().lower() in {"true", "1", "yes"}


def number(row, key, default=0):
    try:
        return float(row.get(key) or default)
    except (TypeError, ValueError):
        return float(default)


def prop(name, value):
    return {"name": name, "value": value}


def cell_weights(row):
    """Region.csv pool_weights.cell_type_weights, with legacy-safe defaults."""
    raw = row.get("pool_weights") or ""
    if not raw:
        return {}
    try:
        parsed = json.loads(raw)
    except (TypeError, ValueError):
        return {}
    weights = parsed.get("cell_type_weights") or {}
    return {str(key): int(value) for key, value in weights.items() if isinstance(value, (int, float))}


def monster_density(row):
    """Relative monster density vs the KHU_VUC reference weight of 30/100."""
    weights = cell_weights(row)
    monster = weights.get("Monster", 0)
    total = sum(max(0, value) for value in weights.values())
    if monster <= 0 or total <= 0:
        return 0.0
    share = monster / total
    reference = REFERENCE_MONSTER_WEIGHT / 100.0
    return share / reference


def weighted_archetype(archetypes, seed):
    """Deterministic pick weighted by MonsterSpawn.chance (default 0.5)."""
    if not archetypes:
        return None
    total = 0.0
    weights = []
    for spawn in archetypes:
        try:
            chance = float(spawn.get("chance") or 0.5)
        except (TypeError, ValueError):
            chance = 0.5
        chance = max(0.01, chance)
        weights.append(chance)
        total += chance
    roll = (int.from_bytes(hashlib.sha256(seed.encode()).digest()[:4], "big") / 0xFFFFFFFF) * total
    cursor = 0.0
    for spawn, weight in zip(archetypes, weights):
        cursor += weight
        if roll <= cursor:
            return spawn
    return archetypes[-1]


def stable_spawn(region_id, row, index, width, height):
    digest = hashlib.sha256(f"{region_id}:{row.get('monster_id','')}:{index}".encode()).digest()
    x_span = max(1, int(width - SPAWN_MARGIN * 2))
    y_span = max(1, int(height - SPAWN_MARGIN * 2))
    return (
        SPAWN_MARGIN + int.from_bytes(digest[:4], "big") % x_span,
        SPAWN_MARGIN + int.from_bytes(digest[4:8], "big") % y_span,
    )


def atlas_bounds(row):
    ax = number(row, "atlas_x", number(row, "min_x"))
    ay = number(row, "atlas_y", number(row, "min_y"))
    w = number(row, "width", 1000)
    h = number(row, "height", 1000)
    return ax, ay, ax + w, ay + h


def overlaps_edge(a, b):
    ax0, ay0, ax1, ay1 = a["atlas_bounds"]
    bx0, by0, bx1, by1 = b["atlas_bounds"]
    epsilon = 1e-6
    if abs(ax1 - bx0) < epsilon or abs(bx1 - ax0) < epsilon:
        start, end = max(ay0, by0), min(ay1, by1)
        if end > start:
            return ("east", "west", start, end) if abs(ax1 - bx0) < epsilon else ("west", "east", start, end)
    if abs(ay1 - by0) < epsilon or abs(by1 - ay0) < epsilon:
        start, end = max(ax0, bx0), min(ax1, bx1)
        if end > start:
            return ("south", "north", start, end) if abs(ay1 - by0) < epsilon else ("north", "south", start, end)
    return None


def portal_pair(source, target, source_side, target_side, start, end):
    along_atlas = (start + end) / 2
    along_source = (along_atlas - (source["atlas_bounds"][1] if source_side in ("east", "west") else source["atlas_bounds"][0])) / SCALE
    along_target = (along_atlas - (target["atlas_bounds"][1] if target_side in ("east", "west") else target["atlas_bounds"][0])) / SCALE
    sw, sh = source["width"], source["height"]
    tw, th = target["width"], target["height"]
    if source_side == "east":
        source_xy = (sw - 2, min(sh - 2, max(2, along_source)))
        target_xy = (2, min(th - 2, max(2, along_target)))
    elif source_side == "west":
        source_xy = (2, min(sh - 2, max(2, along_source)))
        target_xy = (tw - 2, min(th - 2, max(2, along_target)))
    elif source_side == "south":
        source_xy = (min(sw - 2, max(2, along_source)), sh - 2)
        target_xy = (min(tw - 2, max(2, along_target)), 2)
    else:
        source_xy = (min(sw - 2, max(2, along_source)), 2)
        target_xy = (min(tw - 2, max(2, along_target)), th - 2)
    return source_xy, target_xy


def main():
    all_regions = [row for row in read_csv("Region.csv") if truth(row.get("is_active", "true"))]
    active = [row for row in all_regions if row.get("zone_type") in {"KHU_VUC", "SAFE_ZONE"}]
    regions = {}
    for row in active:
        width = int(number(row, "width", 1000))
        height = int(number(row, "height", 1000))
        if width < 8 or height < 8:
            raise ValueError(f"region {row['id']} is too small for collision-safe map: {width}x{height}")
        regions[row["id"]] = {
            "row": row,
            "bounds": (0, 0, width, height),
            "atlas_bounds": atlas_bounds(row),
            "width": width,
            "height": height,
            "portals": [],
        }

    # Every neighboring child region receives a reciprocal walk-through portal.
    for index, source in enumerate(regions.values()):
        for target in list(regions.values())[index + 1:]:
            if source["row"].get("parent_id") != target["row"].get("parent_id"):
                continue
            edge = overlaps_edge(source, target)
            if not edge:
                continue
            source_side, target_side, start, end = edge
            source_xy, target_xy = portal_pair(source, target, source_side, target_side, start, end)
            source["portals"].append((target, source_xy, target_xy))
            target["portals"].append((source, target_xy, source_xy))

    cells = {}
    for index, cell in enumerate(read_csv("MapCell.csv")):
        if truth(cell.get("is_active", "true")):
            cells.setdefault(cell.get("region_id", ""), []).append((index, cell))
    monster_spawns = {}
    for index, spawn in enumerate(read_csv("MonsterSpawn.csv")):
        if truth(spawn.get("is_active", "true")):
            monster_spawns.setdefault(spawn.get("region_id", ""), []).append((index, spawn))

    out_dirs = [ROOT / "backend/data/mmo/maps", ROOT / "frontend/public/game/maps"]
    for directory in out_dirs:
        directory.mkdir(parents=True, exist_ok=True)

    artifact_report = []
    for region_id, region in regions.items():
        row = region["row"]
        width, height = region["width"], region["height"]
        origin_x, origin_y = 0.0, 0.0

        entry_x = number(row, "entry_x", width / 2)
        entry_y = number(row, "entry_y", height / 2)
        player_spawn = {
            "name": "entry_" + region_id, "type": "player",
            "x": round(min(width - 2, max(2, entry_x)), 2),
            "y": round(min(height - 2, max(2, entry_y)), 2),
        }
        spawns = [player_spawn]

        # 1. Spawn Boss / Monster cells from MapCell.csv
        for index, cell in cells.get(region_id, []):
            if row.get("zone_type") == "SAFE_ZONE":
                continue
            ctype = cell.get("cell_type", "")
            cid = cell.get("content_id", "")
            if ctype not in ("Boss", "Monster") or not cid:
                continue
            cx = number(cell, "x")
            cy = number(cell, "y")
            # If cell is too close to player entry, offset slightly so player isn't spawn-camped
            if math.hypot(cx - entry_x, cy - entry_y) < 25:
                cx += 35
                cy += 35
            cx = min(width - SPAWN_MARGIN, max(SPAWN_MARGIN, cx))
            cy = min(height - SPAWN_MARGIN, max(SPAWN_MARGIN, cy))
            spawns.append({
                "name": f"{region_id}_{cid}_cell_{index}",
                "type": "monster",
                "x": round(cx, 2),
                "y": round(cy, 2),
                "properties": [
                    prop("monster_id", cid),
                    prop("region_id", region_id),
                    prop("source_spawn_mode", "Stationary" if ctype == "Boss" else "Patrol"),
                    prop("legacy_approximate", False),
                    prop("is_boss_spawn", ctype == "Boss"),
                ],
            })

        # 2. Spawn archetype packs from MonsterSpawn.csv, weighted by Region.csv
        # cell_type_weights. Every child map (including SAFE_ZONE) honours its
        # authored Monster ratio; SAFE_ZONE monsters are made passive at runtime.
        region_archetypes = [s for _, s in monster_spawns.get(region_id, []) if s.get("monster_id")]
        region_min_rank = number(row, "min_rank", 1)
        region_max_rank = number(row, "max_rank", 9)
        in_band = [
            s for s in region_archetypes
            if number(s, "min_rank", region_min_rank) <= region_max_rank
            and number(s, "max_rank", region_max_rank) >= region_min_rank
        ]
        if in_band:
            region_archetypes = in_band

        density = monster_density(row)
        area_scale = (width * height) / 1_000_000
        pack_count = 0
        if region_archetypes and density > 0:
            pack_count = max(1, round(BASE_PACKS * density * area_scale))

        if pack_count > 0:
            grid_n = max(1, math.ceil(math.sqrt(pack_count)))
            step_w = width / grid_n
            step_h = height / grid_n
            reserved = len(spawns)
            for c_idx in range(pack_count):
                if len(spawns) >= TOTAL_SPAWNS_CAP - reserved:
                    break
                gx = c_idx % grid_n
                gy = c_idx // grid_n
                jitter = hashlib.sha256(f"{region_id}:camp:{c_idx}".encode()).digest()
                jx = int.from_bytes(jitter[:4], "big") / 0xFFFFFFFF
                jy = int.from_bytes(jitter[4:8], "big") / 0xFFFFFFFF
                cx = (gx + 0.15 + 0.7 * jx) * step_w
                cy = (gy + 0.15 + 0.7 * jy) * step_h

                # Keep one pack visible from the authored entry point so a fresh
                # arrival immediately sees the region's Monster ratio.
                if c_idx == 0:
                    angle = (int.from_bytes(jitter[8:12], "big") % 360) * (math.pi / 180)
                    cx = entry_x + math.cos(angle) * 14.0
                    cy = entry_y + math.sin(angle) * 14.0
                cx = min(width - SPAWN_MARGIN, max(SPAWN_MARGIN, cx))
                cy = min(height - SPAWN_MARGIN, max(SPAWN_MARGIN, cy))

                # 1 đàn gồm 4-6 con cùng chủng loại, chọn theo MonsterSpawn.chance
                pack_size = 4 + (int.from_bytes(jitter[12:16], "big") % 3)
                pack_tmpl = weighted_archetype(region_archetypes, f"{region_id}:pack:{c_idx}")
                if pack_tmpl is None:
                    continue
                mid = pack_tmpl.get("monster_id", "")
                mode = pack_tmpl.get("spawn_mode", "Roaming")
                is_boss = truth(pack_tmpl.get("is_boss_spawn", "false"))

                for m_offset in range(pack_size):
                    if len(spawns) >= TOTAL_SPAWNS_CAP:
                        break

                    seed_str = f"{region_id}:{c_idx}:{m_offset}:{mid}"
                    digest = hashlib.sha256(seed_str.encode()).digest()
                    val_a = int.from_bytes(digest[:4], "big")
                    val_b = int.from_bytes(digest[4:8], "big")

                    if m_offset == 0:
                        x, y = cx, cy
                    else:
                        angle = (val_a % 360) * (math.pi / 180)
                        dist = 1.5 + (val_b % 40) / 10.0
                        x = min(width - SPAWN_MARGIN, max(SPAWN_MARGIN, cx + dist * math.cos(angle)))
                        y = min(height - SPAWN_MARGIN, max(SPAWN_MARGIN, cy + dist * math.sin(angle)))

                    spawns.append({
                        "name": f"{region_id}_{mid}_c{c_idx}_{m_offset}",
                        "type": "monster",
                        "x": round(x, 2),
                        "y": round(y, 2),
                        "properties": [
                            prop("monster_id", mid),
                            prop("region_id", region_id),
                            prop("source_spawn_mode", mode),
                            prop("legacy_approximate", True),
                            prop("is_boss_spawn", is_boss and m_offset == 0),
                            prop("pack_id", f"{region_id}_pack_{c_idx}"),
                            prop("is_pack_leader", m_offset == 0),
                            prop("passive", row.get("zone_type") == "SAFE_ZONE"),
                        ],
                    })

        pois = []
        for index, cell in cells.get(region_id, []):
            x = number(cell, "x")
            y = number(cell, "y")
            if not (0 <= x < width and 0 <= y < height):
                continue
            pois.append({
                "name": cell.get("content_id") or cell.get("cell_type", "poi"),
                "type": cell.get("cell_type", "poi"), "x": round(x, 2), "y": round(y, 2),
                "properties": [prop("region_id", region_id), prop("content_id", cell.get("content_id", ""))],
            })

        def portal_side(x, y):
            if x <= 2:
                return "west"
            if x >= width - 2:
                return "east"
            if y <= 2:
                return "north"
            return "south"

        portals = []
        for target, source_xy, target_xy in region["portals"]:
            side = portal_side(source_xy[0], source_xy[1])
            portals.append({
                "name": f"to_{target['row']['id']}", "type": "portal", "x": source_xy[0], "y": source_xy[1],
                "properties": [
                    prop("target_zone_id", target["row"]["id"]),
                    prop("target_x", target_xy[0]), prop("target_y", target_xy[1]),
                    prop("edge", side),
                ],
            })

        is_safe = row.get("zone_type") == "SAFE_ZONE"
        data = {
            "type": "map", "orientation": "orthogonal", "renderorder": "right-down",
            "width": width, "height": height, "tilewidth": 24, "tileheight": 24,
            "map_id": region_id, "map_version": MAP_VERSION, "content_version": CONTENT_VERSION,
            "properties": [
                prop("ruleset", "safe" if is_safe else "pve"),
                prop("legacy_origin_x", origin_x), prop("legacy_origin_y", origin_y),
                prop("legacy_units_per_world_unit", SCALE),
                prop("legacy_region_id", region_id), prop("legacy_region_kind", row["zone_type"]),
                prop("legacy_parent_id", row.get("parent_id", "")),
                prop("region_name", row.get("name", region_id)),
            ],
            "layers": [
                {"type": "tilelayer", "name": "ground", "width": width, "height": height, "data": []},
                {"type": "tilelayer", "name": "collision", "width": width, "height": height, "data": []},
                {"type": "objectgroup", "name": "spawns", "objects": spawns},
                {"type": "objectgroup", "name": "portals", "objects": portals},
                {"type": "objectgroup", "name": "legacy_pois", "objects": pois},
            ],
        }
        encoded = json.dumps(data, ensure_ascii=False, separators=(",", ":")) + "\n"
        for directory in out_dirs:
            (directory / f"{region_id}.json").write_text(encoded, encoding="utf-8")
        monster_count = sum(1 for spawn in spawns if spawn["type"] == "monster")
        artifact_report.append({
            "id": region_id, "type": row["zone_type"], "parent_id": row.get("parent_id", ""),
            "width": width, "height": height, "monster_spawns": monster_count,
            "monster_density": round(density, 4), "monster_packs": pack_count,
            "pois": len(pois), "portals": len(portals),
        })

    # Set the Region CSV's map binding: parent NGU_VUC rows remain the outer map;
    # detailed KHU_VUC and SAFE_ZONE cells each use their own generated artifact.
    csv_path = DATA / "Region.csv"
    with csv_path.open(encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        fields = reader.fieldnames or []
        rows = list(reader)
    if "world_map_id" not in fields:
        fields.append("world_map_id")
    region_ids = set(regions)
    for row in rows:
        if row.get("id") in region_ids:
            row["world_map_id"] = row["id"]
        elif row.get("zone_type") == "NGU_VUC":
            row["world_map_id"] = "world_map"
        elif not row.get("world_map_id"):
            row["world_map_id"] = "world_map"
    with csv_path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields, lineterminator="\r\n")
        writer.writeheader()
        writer.writerows(rows)

    report = {
        "ngu_vuc_outer_maps": sum(1 for row in all_regions if row.get("zone_type") == "NGU_VUC"),
        "khu_vuc_maps": sum(1 for row in active if row.get("zone_type") == "KHU_VUC"),
        "safe_zone_maps": sum(1 for row in active if row.get("zone_type") == "SAFE_ZONE"),
        "secret_realms_left_on_world_map": sum(1 for row in all_regions if row.get("zone_type") == "SECRET_REALM"),
        "total_region_maps": len(artifact_report),
        "monster_spawns_total": sum(item["monster_spawns"] for item in artifact_report),
        "map_version": MAP_VERSION, "content_version": CONTENT_VERSION,
        "collision_source": "Region.csv has no terrain; generated maps have a safe outer wall only",
        "maps": artifact_report,
    }
    (ROOT / "docs/mmo/region-map-conversion-report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({key: value for key, value in report.items() if key != "maps"}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
