#!/usr/bin/env python3
"""
Migrate Region.csv and MapCell.csv to local coordinates (width=1000, height=1000, start at 0,0).
Also records atlas_x, atlas_y for macro world atlas rendering and adjacency.
"""
import csv
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend/data"

def main():
    region_csv = DATA / "Region.csv"
    mapcell_csv = DATA / "MapCell.csv"

    with region_csv.open(encoding="utf-8-sig", newline="") as f:
        reader = csv.DictReader(f)
        old_region_fields = reader.fieldnames or []
        old_regions = list(reader)

    # 1. Prepare new Region.csv rows
    new_regions = []
    region_map = {} # id -> old row
    for r in old_regions:
        region_map[r["id"]] = r

    # Determine atlas position:
    # Nam Cuong: min_x=0..10000, min_y=0..10000 -> atlas_x = min_x / 2, atlas_y = min_y / 2
    # Trung Chau: min_x=10000..20000, min_y=0..10000 -> atlas_x = min_x / 2, atlas_y = min_y / 2
    # Bac Nguyen: min_x=20000..30000, min_y=0..10000 -> atlas_x = min_x / 2, atlas_y = min_y / 2
    # Tay Mac: min_x=0..10000, min_y=10000..20000 -> atlas_x = min_x / 2, atlas_y = min_y / 2
    # Dong Hai: min_x=10000..20000, min_y=10000..20000 -> atlas_x = min_x / 2, atlas_y = min_y / 2

    for r in old_regions:
        row = dict(r)
        zone_type = row.get("zone_type", "KHU_VUC")
        old_min_x = int(float(row.get("min_x") or 0))
        old_min_y = int(float(row.get("min_y") or 0))
        old_w = int(float(row.get("width") or 2000))
        old_h = int(float(row.get("height") or 2000))

        if zone_type in ("KHU_VUC", "SAFE_ZONE"):
            new_w = 1000
            new_h = 1000
            row["width"] = str(new_w)
            row["height"] = str(new_h)
            row["min_x"] = "0"
            row["min_y"] = "0"
            row["max_x"] = str(new_w)
            row["max_y"] = str(new_h)
            row["entry_x"] = str(new_w // 2)
            row["entry_y"] = str(new_h // 2)
            row["safe_zone_x"] = str(new_w // 2)
            row["safe_zone_y"] = str(new_h // 2)
            row["atlas_x"] = str(old_min_x // 2)
            row["atlas_y"] = str(old_min_y // 2)
        elif zone_type == "SECRET_REALM":
            new_w = old_w
            new_h = old_h
            row["width"] = str(new_w)
            row["height"] = str(new_h)
            row["min_x"] = "0"
            row["min_y"] = "0"
            row["max_x"] = str(new_w)
            row["max_y"] = str(new_h)
            row["entry_x"] = str(new_w // 2)
            row["entry_y"] = str(new_h // 2)
            row["safe_zone_x"] = str(new_w // 2)
            row["safe_zone_y"] = str(new_h // 2)
            row["atlas_x"] = str(old_min_x // 2)
            row["atlas_y"] = str(old_min_y // 2)
        elif zone_type == "NGU_VUC":
            new_w = 5000
            new_h = 5000
            row["width"] = str(new_w)
            row["height"] = str(new_h)
            row["min_x"] = str(old_min_x // 2)
            row["min_y"] = str(old_min_y // 2)
            row["max_x"] = str((old_min_x // 2) + new_w)
            row["max_y"] = str((old_min_y // 2) + new_h)
            row["entry_x"] = str((old_min_x // 2) + (new_w // 2))
            row["entry_y"] = str((old_min_y // 2) + (new_h // 2))
            row["safe_zone_x"] = row["entry_x"]
            row["safe_zone_y"] = row["entry_y"]
            row["atlas_x"] = str(old_min_x // 2)
            row["atlas_y"] = str(old_min_y // 2)

        new_regions.append(row)

    # 2. Update MapCell.csv
    with mapcell_csv.open(encoding="utf-8-sig", newline="") as f:
        reader = csv.DictReader(f)
        cell_fields = reader.fieldnames or []
        old_cells = list(reader)

    new_cells = []
    for c in old_cells:
        cell = dict(c)
        reg_id = cell["region_id"]
        old_reg = region_map.get(reg_id)
        if not old_reg:
            new_cells.append(cell)
            continue

        old_min_x = int(float(old_reg.get("min_x") or 0))
        old_min_y = int(float(old_reg.get("min_y") or 0))
        old_w = int(float(old_reg.get("width") or 2000))
        old_h = int(float(old_reg.get("height") or 2000))

        new_reg = next(r for r in new_regions if r["id"] == reg_id)
        new_w = int(new_reg["width"])
        new_h = int(new_reg["height"])

        old_x = int(float(cell["x"]))
        old_y = int(float(cell["y"]))
        rel_x = old_x - old_min_x
        rel_y = old_y - old_min_y

        new_x = int(round(rel_x * new_w / old_w))
        new_y = int(round(rel_y * new_h / old_h))
        new_x = max(1, min(new_w - 1, new_x))
        new_y = max(1, min(new_h - 1, new_y))

        cell["x"] = str(new_x)
        cell["y"] = str(new_y)

        # Update guardian_coord in metadata if present
        meta_str = cell.get("metadata")
        if meta_str and "guardian_coord" in meta_str:
            try:
                meta = json.loads(meta_str)
                if "guardian_coord" in meta and isinstance(meta["guardian_coord"], dict):
                    g = meta["guardian_coord"]
                    gx_rel = int(g.get("x", 0)) - old_min_x
                    gy_rel = int(g.get("y", 0)) - old_min_y
                    meta["guardian_coord"]["x"] = max(1, min(new_w - 1, int(round(gx_rel * new_w / old_w))))
                    meta["guardian_coord"]["y"] = max(1, min(new_h - 1, int(round(gy_rel * new_h / old_h))))
                    cell["metadata"] = json.dumps(meta, ensure_ascii=False)
            except Exception as e:
                print(f"Error parsing metadata for cell {cell['region_id']}: {e}")

        new_cells.append(cell)

    # 3. Write back new Region.csv
    new_region_fields = list(old_region_fields)
    if "atlas_x" not in new_region_fields:
        new_region_fields.append("atlas_x")
    if "atlas_y" not in new_region_fields:
        new_region_fields.append("atlas_y")

    with region_csv.open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=new_region_fields, lineterminator="\r\n")
        writer.writeheader()
        writer.writerows(new_regions)
    print(f"Wrote {len(new_regions)} rows to {region_csv}")

    # 4. Write back new MapCell.csv
    with mapcell_csv.open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=cell_fields, lineterminator="\r\n")
        writer.writeheader()
        writer.writerows(new_cells)
    print(f"Wrote {len(new_cells)} rows to {mapcell_csv}")

if __name__ == "__main__":
    main()
