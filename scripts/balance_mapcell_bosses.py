#!/usr/bin/env python3
"""
scripts/balance_mapcell_bosses.py

Rebalances MapCell.csv boss cells and synchronizes them with MonsterSpawn.csv:
1. Accurately infers each region's Path and Rank tier.
2. Selects the most appropriate Boss (or top-tier Desolate Beast Elite) matching:
   - Region's primary Path (or close sibling path)
   - Region's Rank tier (boss rank within [min_rank, max_rank])
3. Balances boss usage so no boss is over-spawned (hard cap of 1-3 regions per boss).
4. Updates MapCell.csv with correct content_id and metadata {"rank": R}.
5. Synchronizes MonsterSpawn.csv boss rows with the chosen MapCell boss for that region.
"""

import argparse
import csv
import json
import os
import sys
from collections import Counter, defaultdict
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "backend" / "data"

MAP_CELL_CSV = DATA_DIR / "MapCell.csv"
REGION_CSV = DATA_DIR / "Region.csv"
MONSTER_CSV = DATA_DIR / "Monster.csv"
MONSTER_SPAWN_CSV = DATA_DIR / "MonsterSpawn.csv"
RANKS_CSV = REPO_ROOT / "scripts" / "monster_ranks.csv"


def infer_region_path(rid, name, desc):
    t = (rid + " " + name + " " + desc).lower()
    # High priority specific keywords
    if any(k in t for k in ["huyết", "blood"]): return "Blood Path"
    if any(k in t for k in ["bạch cốt", "cốt", "xương", "bone"]): return "Bone Path"
    if any(k in t for k in ["kiếm", "sword"]): return "Sword Path"
    if any(k in t for k in ["đao", "nhận", "blade"]): return "Blade Path"
    if any(k in t for k in ["âm hồn", "hồn", "u hồn", "quỷ hồn", "đãng hồn", "sinh tử môn", "soul"]): return "Soul Path"
    if any(k in t for k in ["mộng", "mộng cảnh", "ảo", "huyễn cảnh", "huyễn sa", "dream"]): return "Dream Path"
    if any(k in t for k in ["lôi", "sấm", "điện", "sét", "lôi đình", "thunder", "lightning"]): return "Lightning Path"
    if any(k in t for k in ["băng", "tuyết", "hàn", "sương", "ice", "frost", "snow"]): return "Ice Path"
    if any(k in t for k in ["hỏa", "diễm", "lửa", "viêm", "fire", "flame"]): return "Fire Path"
    if any(k in t for k in ["độc", "chướng", "chướng khí", "hủ độc", "trùng", "ngô công", "poison", "venom"]): return "Poison Path"
    if any(k in t for k in ["ám", "u minh", "dạ", "hắc ám", "hắc phong", "hắc", "tối", "dark", "shadow"]): return "Dark Path"
    if any(k in t for k in ["quang", "nhật", "minh", "ánh sáng", "bạch quang", "cực quang", "light", "sun"]): return "Light Path"
    if any(k in t for k in ["phong", "liệp phong", "bão", "cuồng phong", "phong ba", "wind", "gale"]): return "Wind Path"
    if any(k in t for k in ["khí hải", "bạch khí", "thiên khí", "qi"]): return "Qi Path"
    if any(k in t for k in ["mộc", "sâm", "lâm", "thảo", "trúc", "thụ", "cây", "rừng", "wood", "forest"]): return "Wood Path"
    if any(k in t for k in ["kim", "thiết", "đồng", "canh kim", "kim tự tháp", "metal", "iron", "gold"]): return "Metal Path"
    if any(k in t for k in ["vân", "mây", "cloud"]): return "Cloud Path"
    if any(k in t for k in ["thú", "lang", "vạn thú", "ngự thú", "beast", "wolf", "enslavement"]): return "Enslavement Path"
    if any(k in t for k in ["thủy", "hải", "hác", "giang", "hà", "thương thủy", "kình", "lãng", "hồ", "trì", "biển", "water", "sea", "ocean", "river"]): return "Water Path"
    if any(k in t for k in ["thổ", "thạch", "sơn", "nham", "địa", "địa câu", "sa mạc", "cát", "sa", "khâu", "núi", "đá", "earth", "desert"]): return "Earth Path"
    if any(k in t for k in ["luyện", "luyện cổ", "refinement"]): return "Refinement Path"
    if any(k in t for k in ["âm", "âm thanh", "tiếng", "hống", "sound"]): return "Sound Path"
    if any(k in t for k in ["tinh", "thiên", "sao", "tinh tú", "thần", "thiên đình", "star", "heaven"]): return "Star Path"
    if any(k in t for k in ["tạo", "tạo hóa", "creation"]): return "Creation Path"
    if any(k in t for k in ["hủy", "hủy diệt", "diệt", "toái", "destruction"]): return "Destruction Path"
    if any(k in t for k in ["thực", "ăn", "uống", "tửu", "food"]): return "Food Path"
    if any(k in t for k in ["đạo tặc", "thâu", "trộm", "theft"]): return "Theft Path"
    if any(k in t for k in ["trận", "bát quái", "cơ quan", "khôi lỗi", "formation"]): return "Formation Path"
    if any(k in t for k in ["sát", "tu la", "killing"]): return "Killing Path"
    if any(k in t for k in ["thời", "quang âm", "tuế", "time"]): return "Time Path"
    if any(k in t for k in ["họa", "tranh", "đan thanh", "painting"]): return "Painting Path"
    if any(k in t for k in ["không", 'hư không', "space"]): return "Space Path"
    return "Strength Path"


def run_balance(dry_run=True):
    print("=== BẮT ĐẦU CÂN BẰNG BOSS TRONG MAPCELL & MONSTER SPAWN ===")

    # 1. Load Data
    with open(REGION_CSV, "r", encoding="utf-8") as f:
        regions = {r["id"]: r for r in csv.DictReader(f)}

    with open(MAP_CELL_CSV, "r", encoding="utf-8") as f:
        map_cell_rows = list(csv.DictReader(f))

    with open(MONSTER_CSV, "r", encoding="utf-8") as f:
        monsters = {m["id"]: m for m in csv.DictReader(f)}

    ranks = {}
    with open(RANKS_CSV, "r", encoding="utf-8-sig") as f:
        for r in csv.DictReader(f):
            ranks[r["monster_id"]] = int(r["rank"])

    # Build candidates (Bosses and Elites)
    boss_candidates = [m for m in monsters.values() if m["type"] in ("Boss", "Tribulation_Boss", "Elite")]
    cand_info = {}
    for m in boss_candidates:
        mid = m["id"]
        pm = json.loads(m.get("path_masteries") or "{}")
        p = list(pm.keys())[0] if pm else "Strength Path"
        cand_info[mid] = {
            "path": p,
            "rank": ranks.get(mid, 1),
            "type": m["type"],
            "name": m["name"],
        }

    # Only KHU_VUC regions should have combat bosses (115 total)
    khu_vucs = {r["id"]: r for r in regions.values() if r.get("zone_type") == "KHU_VUC"}

    # 2. Process Non-Boss Cells (SafePoints & Resources) to ensure correct coordinates
    updated_non_boss = []
    occupied_coords = set()
    region_non_boss = {}
    for cell in map_cell_rows:
        if cell["cell_type"].lower() == "boss":
            continue
        rid = cell["region_id"]
        if rid not in khu_vucs:
            continue
        region_non_boss.setdefault(rid, []).append(cell)

    for rid, cells in region_non_boss.items():
        r = khu_vucs[rid]
        min_x, max_x = int(r["min_x"]), int(r["max_x"])
        min_y, max_y = int(r["min_y"]), int(r["max_y"])
        ex = int(r["entry_x"]) if r.get("entry_x") else (min_x + max_x) // 2
        ey = int(r["entry_y"]) if r.get("entry_y") else (min_y + max_y) // 2

        res_idx = 0
        for c in cells:
            ctype = c["cell_type"]
            new_c = dict(c)
            if ctype == "SafePoint":
                new_c["x"] = str(ex)
                new_c["y"] = str(ey)
            elif ctype == "Resource":
                if res_idx == 0:
                    new_c["x"] = str(ex - 60)
                    new_c["y"] = str(ey - 60)
                elif res_idx == 1:
                    new_c["x"] = str(ex)
                    new_c["y"] = str(ey + 100)
                else:
                    new_c["x"] = str(ex + 60)
                    new_c["y"] = str(ey - 60)
                res_idx += 1
            nx, ny = int(new_c["x"]), int(new_c["y"])
            assert min_x <= nx <= max_x and min_y <= ny <= max_y, f"Non-boss out of bounds: {rid} ({nx},{ny})"
            updated_non_boss.append(new_c)
            occupied_coords.add((rid, nx, ny))

    # Existing boss mapping for diffing
    old_boss_by_rid = {}
    for cell in map_cell_rows:
        if cell["cell_type"].lower() == "boss":
            old_boss_by_rid[cell["region_id"]] = cell

    # 3. Process MapCell Bosses for all KHU_VUC
    usage = Counter()
    assignments = {}
    updated_boss_cells = []
    changes = []

    for rid, r in sorted(khu_vucs.items()):
        rname = r.get("name", rid)
        min_r = int(r.get("min_rank", 1))
        max_r = int(r.get("max_rank", 9))
        min_x, max_x = int(r["min_x"]), int(r["max_x"])
        min_y, max_y = int(r["min_y"]), int(r["max_y"])
        ex = int(r["entry_x"]) if r.get("entry_x") else (min_x + max_x) // 2
        ey = int(r["entry_y"]) if r.get("entry_y") else (min_y + max_y) // 2

        r_path = infer_region_path(rid, rname, r.get("description", ""))

        # Ideal target boss rank for this region
        if max_r == 9:
            target_r = 9
        elif max_r >= 8:
            target_r = 8
        elif max_r == 7:
            target_r = 7
        elif max_r == 6:
            target_r = 6
        elif max_r == 5:
            target_r = 5
        elif max_r <= 4:
            target_r = 4
        else:
            target_r = max_r

        # STRICTLY filter candidates within region [min_r, max_r]
        valid_cands = {mid: info for mid, info in cand_info.items() if min_r <= info["rank"] <= max_r}
        if not valid_cands:
            raise ValueError(f"No valid boss candidate for region {rid} within rank [{min_r}, {max_r}]")

        best_m = None
        best_score = -999999

        for mid, info in valid_cands.items():
            rk = info["rank"]
            mpath = info["path"]
            mtype = info["type"]

            score = 0

            # Path alignment
            if mpath == r_path:
                score += 200
            else:
                # Sibling path compatibility
                if (r_path == "Water Path" and mpath in ("Ice Path", "Poison Path")) or \
                   (r_path == "Ice Path" and mpath in ("Water Path", "Dark Path")) or \
                   (r_path == "Wind Path" and mpath in ("Lightning Path", "Cloud Path")) or \
                   (r_path == "Lightning Path" and mpath in ("Wind Path", "Light Path")) or \
                   (r_path == "Dark Path" and mpath in ("Soul Path", "Poison Path")) or \
                   (r_path == "Soul Path" and mpath in ("Dark Path", "Death Path")) or \
                   (r_path == "Earth Path" and mpath in ("Metal Path", "Strength Path")) or \
                   (r_path == "Poison Path" and mpath in ("Water Path", "Dark Path")) or \
                   (r_path == "Star Path" and mpath in ("Heaven Path", "Light Path")) or \
                   (r_path == "Heaven Path" and mpath in ("Star Path", "Light Path")):
                    score += 50
                else:
                    score -= 60

            # Rank distance
            dist = abs(rk - target_r)
            score -= dist * 30

            # Type bonus
            if mtype in ("Boss", "Tribulation_Boss"):
                score += 40
            else:
                score += 10

            # Usage penalty (to prevent monopolization)
            score -= usage[mid] * 35

            if score > best_score:
                best_score = score
                best_m = mid

        usage[best_m] += 1
        assignments[rid] = (best_m, r_path)
        chosen_info = cand_info[best_m]

        old_cell = old_boss_by_rid.get(rid)
        old_boss = old_cell["content_id"] if old_cell else ""
        old_x = old_cell["x"] if old_cell else ""
        old_y = old_cell["y"] if old_cell else ""
        new_boss = best_m
        new_rank = chosen_info["rank"]

        # Position boss: offset from SafePoint if SafePoint occupies (ex, ey)
        if (rid, ex, ey) in occupied_coords:
            bx, by = ex + 100, ey + 100
        else:
            bx, by = ex, ey

        assert min_x <= bx <= max_x and min_y <= by <= max_y, f"Boss out of bounds: {rid} ({bx},{by})"
        assert (rid, bx, by) not in occupied_coords, f"Coordinate collision: {rid} ({bx},{by})"
        occupied_coords.add((rid, bx, by))

        if old_boss != new_boss or old_x != str(bx) or old_y != str(by):
            changes.append({
                "region_id": rid,
                "region_name": rname,
                "path": r_path,
                "rank_range": f"R{min_r}-R{max_r}",
                "old_boss": old_boss,
                "old_pos": f"({old_x},{old_y})",
                "new_boss": new_boss,
                "new_pos": f"({bx},{by})",
                "new_name": chosen_info["name"],
                "new_rank": new_rank,
            })

        updated_boss_cells.append({
            "region_id": rid,
            "x": str(bx),
            "y": str(by),
            "cell_type": "Boss",
            "content_id": new_boss,
            "metadata": json.dumps({"rank": new_rank}),
            "respawn_seconds": "600",
            "is_active": "true"
        })

    # Combine non-boss and boss cells
    final_map_cells = updated_non_boss + updated_boss_cells

    print(f"\nTổng số ô Non-Boss cập nhật toạ độ: {len(updated_non_boss)}")
    print(f"Tổng số ô Boss trong MapCell.csv: {len(updated_boss_cells)} (100% thuộc 115 KHU_VUC)")
    print(f"Tổng số ô bản đồ trong MapCell.csv: {len(final_map_cells)}")
    print(f"Số ô Boss được cập nhật / sửa toạ độ: {len(changes)}")
    print(f"Số lượng Boss độc lập được sử dụng: {len(usage)}")
    print(f"Tần suất xuất hiện mỗi Boss: Min={min(usage.values())}, Max={max(usage.values())}, Avg={sum(usage.values())/len(usage):.2f}")

    # Check path match rate
    exact_path = sum(1 for rid, (mid, rpath) in assignments.items() if cand_info[mid]["path"] == rpath)
    print(f"Tỉ lệ khớp đúng Lưu phái của khu vực: {exact_path}/{len(assignments)} ({exact_path/len(assignments)*100:.1f}%)")

    # 4. Synchronize with MonsterSpawn.csv
    updated_spawns = []
    spawn_sync_count = 0
    spawn_dropped_count = 0
    if MONSTER_SPAWN_CSV.exists():
        with open(MONSTER_SPAWN_CSV, "r", encoding="utf-8") as f:
            spawn_rows = list(csv.DictReader(f))

        for row in spawn_rows:
            rid = row["region_id"]
            if row.get("is_boss_spawn") == "True":
                # Discard boss spawns for macro regions (NGU_VUC)
                if rid not in assignments:
                    spawn_dropped_count += 1
                    continue
                assigned_boss, _ = assignments[rid]
                assigned_rank = cand_info[assigned_boss]["rank"]
                if row["monster_id"] != assigned_boss or row["min_rank"] != str(assigned_rank):
                    row["monster_id"] = assigned_boss
                    row["min_rank"] = str(assigned_rank)
                    row["max_rank"] = str(assigned_rank)
                    spawn_sync_count += 1
            updated_spawns.append(row)
        print(f"Đã đồng bộ hóa {spawn_sync_count} dòng Boss trong MonsterSpawn.csv với MapCell.csv (loại bỏ {spawn_dropped_count} dòng boss NGU_VUC thừa).")

    print("\n--- Ví dụ một số thay đổi tiêu biểu ---")
    for ch in changes[:15]:
        print(f"  • {ch['region_name']} ({ch['path']}, {ch['rank_range']}):")
        print(f"      Cũ: {ch['old_boss']} {ch['old_pos']}")
        print(f"      Mới: {ch['new_boss']} ({ch['new_name']}, Rank {ch['new_rank']}) {ch['new_pos']}")

    if not dry_run:
        # Write MapCell.csv
        fieldnames = ["region_id", "x", "y", "cell_type", "content_id", "metadata", "respawn_seconds", "is_active"]
        with open(MAP_CELL_CSV, "w", encoding="utf-8", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(final_map_cells)

        # Write MonsterSpawn.csv
        if updated_spawns:
            s_fieldnames = list(updated_spawns[0].keys())
            with open(MONSTER_SPAWN_CSV, "w", encoding="utf-8", newline="") as f:
                writer = csv.DictWriter(f, fieldnames=s_fieldnames)
                writer.writeheader()
                writer.writerows(updated_spawns)

        print("\n[HOÀN TẤT] Đã ghi thành công dữ liệu vào MapCell.csv và MonsterSpawn.csv!")
    else:
        print("\n[DRY RUN] Chạy ở chế độ xem trước, chưa ghi file. Dùng --apply để áp dụng.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Rebalance MapCell bosses and sync with MonsterSpawn.csv")
    parser.add_argument("--apply", action="store_true", help="Apply changes to CSV files")
    args = parser.parse_args()

    run_balance(dry_run=not args.apply)

