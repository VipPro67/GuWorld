#!/usr/bin/env python3
"""Build runtime loot CSVs from backend/data/Loot_Monster_NguVuc.csv.

Region loot pack: each (KHU_VUC region, tier) gets one loot table, applied as
Layer 3 (region_loot_tables) on top of the per-monster table and the shared
common pool.

  - LootTable.csv       : keep existing tables, drop the superseded
                          loot_region_dong_hai_* and any previous run of the
                          spec ids, append the 345 per-zone tables.
  - RegionLootTables.csv: regenerate 345 rows (region x tier -> table).

Idempotent: re-running replaces the spec tables instead of duplicating them.
Run from anywhere:  python3 scripts/gen_region_loot.py [data_dir]

Spec rows are KHU_VUC only, one per (region, tier). tier NORMAL/ELITE/BOSS
maps to monster_type Normal/Elite/Boss because RollRegionLoot matches
monster.Type case-sensitively. Ranks come from the region band; the entry.rank
gates inside each table prevent over-tier drops.
"""
import csv
import io
import os
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA = sys.argv[1] if len(sys.argv) > 1 else os.path.join(REPO, "backend", "data")

SPEC = os.path.join(DATA, "Loot_Monster_NguVuc.csv")
LOOT = os.path.join(DATA, "LootTable.csv")
REGION = os.path.join(DATA, "Region.csv")
REGION_LOOT = os.path.join(DATA, "RegionLootTables.csv")

OLD_REGION_TABLES = {
    "loot_region_dong_hai_normal",
    "loot_region_dong_hai_elite",
    "loot_region_dong_hai_boss",
}
TIER_ORDER = {"NORMAL": 0, "ELITE": 1, "BOSS": 2}
TIER_TYPE = {"NORMAL": "Normal", "ELITE": "Elite", "BOSS": "Boss"}


def read_spec(path):
    lines = open(path, encoding="utf-8-sig").read().splitlines()
    hi = next(i for i, l in enumerate(lines) if l.startswith("id,name,monster_id"))
    return [r for r in csv.DictReader(io.StringIO("\n".join(lines[hi:]))) if r.get("id")]


def render_row(fields):
    buf = io.StringIO()
    csv.writer(buf, lineterminator="\n").writerow(fields)
    return buf.getvalue().rstrip("\n")


def main():
    regions = {r["id"]: r for r in csv.DictReader(open(REGION, encoding="utf-8-sig"))}
    spec = read_spec(SPEC)

    assert len(spec) == 345, f"expected 345 spec rows, got {len(spec)}"
    assert len({r["id"] for r in spec}) == 345, "duplicate loot ids in spec"
    assert len({(r["region_id"], r["tier"]) for r in spec}) == 345, "duplicate (region,tier)"
    missing = {r["region_id"] for r in spec} - set(regions)
    assert not missing, f"spec regions missing from Region.csv: {missing}"

    # --- LootTable.csv: replace spec ids + drop superseded region tables ---
    spec_ids = {r["id"] for r in spec}
    drop = OLD_REGION_TABLES | spec_ids
    raw = open(LOOT, encoding="utf-8").read().splitlines()
    header, rows = raw[0], raw[1:]
    kept = [r for r in rows if r and r.split(",", 1)[0] not in drop]

    new_lines = [header] + kept
    for r in spec:
        ts = r.get("updated_at", "")
        new_lines.append(render_row([r["id"], r["name"], r["entries"], ts, ts]))
    open(LOOT, "w", encoding="utf-8").write("\n".join(new_lines) + "\n")

    # --- RegionLootTables.csv: one row per (region, tier) ---
    out = [["region_id", "loot_table_id", "monster_type", "min_rank", "max_rank", "chance", "is_active"]]
    for r in sorted(spec, key=lambda r: (r["region_id"], TIER_ORDER[r["tier"]])):
        reg = regions[r["region_id"]]
        lo = int(reg["min_rank"]) if reg["min_rank"].strip() else int(r["rank"])
        hi = int(reg["max_rank"]) if reg["max_rank"].strip() else int(r["rank"])
        out.append([r["region_id"], r["id"], TIER_TYPE[r["tier"]], str(lo), str(hi), "1.0", "true"])
    buf = io.StringIO()
    csv.writer(buf, lineterminator="\n").writerows(out)
    open(REGION_LOOT, "w", encoding="utf-8").write(buf.getvalue())

    print(f"LootTable.csv: kept {len(kept)} existing, added {len(spec)} -> {len(new_lines)-1} tables")
    print(f"RegionLootTables.csv: {len(out)-1} rows")


if __name__ == "__main__":
    main()
