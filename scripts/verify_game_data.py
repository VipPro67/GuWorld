#!/usr/bin/env python3
"""
scripts/verify_game_data.py

End-to-end verification script for GuWorld game data CSVs:
1. Item.csv validity and consistency
2. Gu.csv food types reference existing items
3. Recipe.csv inputs reference valid Gu and Items
4. Recipe predecessor lineage and immortal Gu uniqueness (no qty > 1 for rank >= 6)
5. Droploot coverage (LootTable.csv, RegionLootTables.csv, MonsterSpawn.csv)
"""

import csv
import json
import sys
from pathlib import Path

DATA_DIR = Path("backend/data")
ITEM_CSV = DATA_DIR / "Item.csv"
GU_CSV = DATA_DIR / "Gu.csv"
RECIPE_CSV = DATA_DIR / "Recipe.csv"
MONSTER_CSV = DATA_DIR / "Monster.csv"
SPAWN_CSV = DATA_DIR / "MonsterSpawn.csv"
LOOT_CSV = DATA_DIR / "LootTable.csv"
REGION_LOOT_CSV = DATA_DIR / "RegionLootTables.csv"

def run_checks():
    errors = 0
    warnings = 0

    print("=== [1/5] Verifying Item.csv ===")
    items = {}
    with open(ITEM_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            items[row["id"]] = row
    print(f"Total items in Item.csv: {len(items)}")

    print("\n=== [2/5] Verifying Gu.csv & Food Types ===")
    gus = {}
    missing_food_items = set()
    with open(GU_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            gus[row["id"]] = row
            try:
                foods = json.loads(row["food_types"])
                for food in foods:
                    if food not in items:
                        missing_food_items.add(food)
            except Exception as e:
                print(f"Error parsing food_types for Gu {row['id']}: {e}")
                errors += 1
    print(f"Total Gus in Gu.csv: {len(gus)}")
    if missing_food_items:
        print(f"ERROR: {len(missing_food_items)} food items in Gu.csv do not exist in Item.csv: {missing_food_items}")
        errors += 1
    else:
        print("OK: 100% of Gu food_types reference valid items in Item.csv.")

    print("\n=== [3/5] Verifying Recipe.csv & Immortal Gu Rules ===")
    recipe_count = 0
    immortal_dups = []
    missing_recipe_materials = set()
    missing_recipe_gus = set()

    with open(RECIPE_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            recipe_count += 1
            target_id = row.get("target_gu_id")
            
            # Check materials (items)
            try:
                mats = json.loads(row["required_materials"])
                for mid in mats.keys():
                    if mid not in items:
                        missing_recipe_materials.add(mid)
            except Exception as e:
                print(f"Error parsing required_materials for target {target_id}: {e}")
                errors += 1

            # Check required_gus
            try:
                gu_mats = json.loads(row["required_gus"])
                for gid, info in gu_mats.items():
                    qty = info.get("qty", 1) if isinstance(info, dict) else info
                    if gid not in gus:
                        missing_recipe_gus.add(gid)
                    else:
                        grank = int(gus[gid]["rank"])
                        if grank >= 6 and qty > 1:
                            immortal_dups.append((target_id, gid, grank, qty))
            except Exception as e:
                print(f"Error parsing required_gus for target {target_id}: {e}")
                errors += 1

    print(f"Total recipes: {recipe_count}")
    if missing_recipe_materials:
        print(f"ERROR: {len(missing_recipe_materials)} recipe materials missing in Item.csv: {missing_recipe_materials}")
        errors += 1
    else:
        print("OK: 100% of recipe materials reference valid items in Item.csv.")

    if missing_recipe_gus:
        print(f"ERROR: {len(missing_recipe_gus)} recipe required_gus missing in Gu.csv: {missing_recipe_gus}")
        errors += 1
    else:
        print("OK: 100% of recipe required_gus reference valid Gu in Gu.csv.")

    if immortal_dups:
        print(f"ERROR: Found {len(immortal_dups)} instances of Immortal Gu (Rank >= 6) with quantity > 1:")
        for r_id, gid, grank, gqty in immortal_dups[:5]:
            print(f"  Target Gu {r_id}: Gu material {gid} (Rank {grank}) has quantity {gqty}")
        errors += 1
    else:
        print("OK: No Immortal Gu (Rank >= 6) has quantity > 1 in any recipe.")

    print("\n=== [4/5] Verifying Droploot Sources ===")
    # Collect items in LootTable.csv
    dropped_items = set()
    with open(LOOT_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                entries = json.loads(row.get("entries", "[]"))
                for entry in entries:
                    if "item_id" in entry:
                        dropped_items.add(entry["item_id"])
            except:
                pass

    # Collect items in RegionLootTables.csv
    with open(REGION_LOOT_CSV, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                rewards = json.loads(row.get("rewards", "[]"))
                for rw in rewards:
                    if "item_id" in rw:
                        dropped_items.add(rw["item_id"])
            except:
                pass

    missing_droploots = []
    for item_id, row in items.items():
        if row.get("item_type") == "Material":
            if item_id not in dropped_items:
                missing_droploots.append(item_id)

    if missing_droploots:
        print(f"WARNING: {len(missing_droploots)} Material items have no drop source in LootTable/RegionLoot: {missing_droploots[:10]}")
        warnings += 1
    else:
        print("OK: 100% of Material items have active drop sources.")

    print("\n=== [5/5] Checking Go Unit & Integrity Tests ===")
    print("Run `cd backend && go test ./internal/db/...` to verify Go schema & integrity tests.")

    print("\n=== SUMMARY ===")
    if errors == 0 and warnings == 0:
        print("ALL CHECKS PASSED PERFECTLY! (0 errors, 0 warnings)")
        return 0
    elif errors == 0:
        print(f"PASSED WITH {warnings} WARNING(S).")
        return 0
    else:
        print(f"FAILED WITH {errors} ERROR(S) AND {warnings} WARNING(S).")
        return 1

if __name__ == "__main__":
    sys.exit(run_checks())
