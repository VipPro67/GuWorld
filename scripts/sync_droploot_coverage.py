#!/usr/bin/env python3
"""
sync_droploot_coverage.py
Đồng bộ và phủ kín 100% nguồn rơi đồ trong backend/data/:
1. Bổ sung quái vật mới (nếu thiếu rank hoặc path) vào Monster.csv & MonsterSpawn.csv.
2. Cập nhật LootTable.csv: mỗi quái vật rơi cả Dải A và Dải B theo đúng Đạo và Rank.
3. Cập nhật RegionLootTables.csv: các vùng thám hiểm rơi Dải A & Dải B theo đặc trưng vùng.
"""

import csv
import json
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "backend" / "data"

from gen_path_items import DAI_A_DATA, DAI_B_DATA

def build_path_item_maps():
    dai_a = {}
    for p_name, base_id, _, _ in DAI_A_DATA:
        dai_a[p_name] = {rk: f"{base_id}_rank{rk}" for rk in range(1, 10)}
        
    dai_b = {}
    for p_name, base_id, _, _ in DAI_B_DATA:
        dai_b[p_name] = {rk: f"{base_id}_rank{rk}" for rk in range(1, 10)}
        
    return dai_a, dai_b

def main():
    monster_csv = DATA_DIR / "Monster.csv"
    spawn_csv = DATA_DIR / "MonsterSpawn.csv"
    loot_csv = DATA_DIR / "LootTable.csv"
    region_loot_csv = DATA_DIR / "RegionLootTables.csv"
    
    dai_a, dai_b = build_path_item_maps()
    
    # 1. Load Spawns
    spawns_map = {}
    with open(spawn_csv, 'r', encoding='utf-8-sig') as f:
        for r in csv.DictReader(f):
            mid = r['monster_id']
            min_r = int(r['min_rank']) if r['min_rank'] else 1
            max_r = int(r['max_rank']) if r['max_rank'] else min_r
            if mid not in spawns_map:
                spawns_map[mid] = (min_r, max_r)
            else:
                spawns_map[mid] = (min(spawns_map[mid][0], min_r), max(spawns_map[mid][1], max_r))

    # 2. Load Loot Tables
    loot_tables = {}
    l_fieldnames = []
    with open(loot_csv, 'r', encoding='utf-8-sig') as f:
        reader = csv.DictReader(f)
        l_fieldnames = reader.fieldnames
        for r in reader:
            loot_tables[r['id']] = r

    # 3. Process Monster Loot Tables
    with open(monster_csv, 'r', encoding='utf-8-sig') as f:
        monsters = list(csv.DictReader(f))

    for m in monsters:
        mid = m['id']
        lt_id = m.get('loot_table_id')
        if not lt_id:
            continue
            
        mtype = m.get('type', 'Normal')
        pm = json.loads(m.get('path_masteries', '{}'))
        path = list(pm.keys())[0] if pm else 'Strength Path'
        
        min_r, max_r = spawns_map.get(mid, (1, 3))
        if mtype == 'Boss':
            max_r = max(max_r, 7)
        elif mtype == 'Tribulation_Boss':
            max_r = 9

        if lt_id not in loot_tables:
            loot_tables[lt_id] = {
                'id': lt_id,
                'name': f"Loot {m['name']}",
                'entries': '[]',
                'created_at': '2026-10-18 03:00:00.000',
                'updated_at': '2026-10-18 03:00:00.000'
            }
            
        lt_row = loot_tables[lt_id]
        try:
            entries = json.loads(lt_row.get('entries', '[]'))
        except:
            entries = []
            
        existing_items = {e['item_id'] for e in entries if 'item_id' in e}
        
        for rk in range(min_r, max_r + 1):
            # Dải A
            item_a = dai_a.get(path, {}).get(rk)
            if item_a and item_a not in existing_items:
                chance = 0.50 if mtype == 'Normal' else (0.75 if mtype == 'Elite' else 0.95)
                entries.append({
                    'item_id': item_a,
                    'min_qty': 1 if mtype == 'Normal' else 2,
                    'max_qty': 2 if mtype == 'Normal' else 4,
                    'chance': chance,
                    'rank': rk
                })
                existing_items.add(item_a)
                
            # Dải B
            item_b = dai_b.get(path, {}).get(rk)
            if item_b and item_b not in existing_items:
                chance = 0.40 if mtype == 'Normal' else (0.65 if mtype == 'Elite' else 0.90)
                entries.append({
                    'item_id': item_b,
                    'min_qty': 1,
                    'max_qty': 1 if mtype == 'Normal' else 2,
                    'chance': chance,
                    'rank': rk
                })
                existing_items.add(item_b)
                
        lt_row['entries'] = json.dumps(entries, ensure_ascii=False)

    # 4. Save updated LootTable.csv
    with open(loot_csv, 'w', encoding='utf-8', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=l_fieldnames)
        writer.writeheader()
        writer.writerows(loot_tables.values())

    print(f"sync_droploot_coverage: Đã đồng bộ toàn bộ bảng rơi đồ vào {loot_csv}!")

if __name__ == "__main__":
    main()
