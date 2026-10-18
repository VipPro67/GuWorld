#!/usr/bin/env python3
"""
sync_gu_food.py
Đồng bộ trường food_types cho toàn bộ Cổ trùng trong backend/data/Gu.csv.
Mỗi Cổ trùng (Path P, Rank R) sẽ ăn:
1. Món chính 1: Vật phẩm Dải A của Path P ở đúng Rank R (base nutrition = 100).
2. Món chính 2: Vật phẩm Dải B của Path P ở đúng Rank R (base nutrition = 100).
3. Món phụ cứu đói: Vật phẩm Dải B của Path P ở Rank R - 1 (nếu R > 1).
Nếu Cổ có nhiều Path, tự động bổ sung trọn bộ của các Path đó.
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
    gu_file = DATA_DIR / "Gu.csv"
    dai_a, dai_b = build_path_item_maps()
    
    updated_rows = []
    fieldnames = []
    
    with open(gu_file, "r", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        fieldnames = reader.fieldnames
        for row in reader:
            grank = int(row["rank"])
            try:
                gpaths = json.loads(row["path"])
            except:
                gpaths = ["Strength Path"]
            if not gpaths:
                gpaths = ["Strength Path"]
                
            foods = []
            for p in gpaths:
                # Dải A cùng rank
                if p in dai_a and grank in dai_a[p]:
                    foods.append(dai_a[p][grank])
                # Dải B cùng rank
                if p in dai_b and grank in dai_b[p]:
                    foods.append(dai_b[p][grank])
                # Dải B rank thấp hơn (nếu có) làm thức ăn phụ
                if grank > 1 and p in dai_b and (grank - 1) in dai_b[p]:
                    foods.append(dai_b[p][grank - 1])
                    
            unique_foods = []
            for fd in foods:
                if fd not in unique_foods:
                    unique_foods.append(fd)
                    
            row["food_types"] = json.dumps(unique_foods, ensure_ascii=False)
            updated_rows.append(row)
            
    with open(gu_file, "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(updated_rows)
        
    print(f"sync_gu_food: Đã đồng bộ thức ăn cho toàn bộ {len(updated_rows)} Cổ trùng trong {gu_file}!")

if __name__ == "__main__":
    main()
