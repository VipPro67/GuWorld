#!/usr/bin/env python3
"""
rebuild_recipes.py
Tái tạo toàn bộ 4.479 Bí phương hợp luyện trong backend/data/Recipe.csv:
1. Áp dụng 792 nguyên liệu MỚI (Dải A + Dải B của 44 Path).
2. Ưu tiên dòng tiến hóa:
   - Cùng chính ID ở Rank thấp hơn (Rank R - 1)
   - Cùng Stem ID (_1 -> _2 -> _3 -> _4 -> _5 -> _I)
   - Cùng tên gốc Tiếng Việt
   - Cùng Đạo (Path) ở Rank thấp hơn
3. Ràng buộc Tiên Cổ (Rank 6, 7, 8, 9):
   - Mọi Tiên Cổ làm nguyên liệu ĐỘC BẢN LUÔN CÓ qty = 1.
   - Các Cổ phụ tá phải là các Cổ HOÀN TOÀN KHÁC NHAU.
   - Đòi hỏi hàng trăm Tiên tài và hàng trăm đến hàng vạn Tiên Nguyên Thạch.
"""

import csv
import json
import re
import zlib
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

def get_stem(gid):
    m = re.match(r'^(.*?)_([1-9]|I|II|III|IV|V|VI|VII|VIII|IX)$', gid)
    return m.group(1) if m else gid

def clean_vn_name(name):
    prefixes = ['Sơ Giai ', 'Nhị Giai ', 'Tam Giai ', 'Tứ Giai ', 'Ngũ Giai ', 'Cửu Chuyển ', 'Bát Chuyển ', 'Thất Chuyển ', 'Lục Chuyển ', 'Tiên ']
    for p in prefixes:
        if name.startswith(p):
            name = name[len(p):]
    name = name.replace(' Tiên Cổ', ' Cổ')
    return name.strip()

def main():
    recipe_file = DATA_DIR / "Recipe.csv"
    gu_file = DATA_DIR / "Gu.csv"
    
    dai_a, dai_b = build_path_item_maps()
    
    gus_by_id_rank = {}
    gus_by_rank = {}
    gus_by_path_rank = {}

    with open(gu_file, 'r', encoding='utf-8-sig') as f:
        for r in csv.DictReader(f):
            gid = r['id']
            grk = int(r['rank'])
            name = r['name']
            try:
                paths = json.loads(r['path'])
            except:
                paths = ['Strength Path']
            if not paths:
                paths = ['Strength Path']
            gus_by_id_rank[(gid, grk)] = (name, paths)
            gus_by_rank.setdefault(grk, []).append(gid)
            for p in paths:
                gus_by_path_rank.setdefault((p, grk), []).append(gid)

    names_map = {}
    for (gid, grk), (name, paths) in gus_by_id_rank.items():
        cname = clean_vn_name(name)
        names_map.setdefault(cname, {})[grk] = gid

    def find_primary_predecessor(target_id, target_rank, primary_path):
        if target_rank <= 1:
            return None
        prev_rank = target_rank - 1
        
        # 1. Exact same ID at target_rank - 1
        if (target_id, prev_rank) in gus_by_id_rank:
            return target_id
            
        # 2. Same stem at target_rank - 1
        t_stem = get_stem(target_id)
        stem_candidates = [gid for gid in gus_by_rank.get(prev_rank, []) if get_stem(gid) == t_stem]
        if stem_candidates:
            return stem_candidates[0]
            
        # 3. Same clean Vietnamese base name
        t_name = gus_by_id_rank.get((target_id, target_rank), ('', []))[0]
        cname = clean_vn_name(t_name)
        if cname in names_map and prev_rank in names_map[cname]:
            return names_map[cname][prev_rank]
            
        # 4. Same Path
        path_candidates = [gid for gid in gus_by_path_rank.get((primary_path, prev_rank), []) if gid != target_id]
        if path_candidates:
            return path_candidates[0]
            
        # 5. Any Gu at prev_rank
        all_prev = [gid for gid in gus_by_rank.get(prev_rank, []) if gid != target_id]
        return all_prev[0] if all_prev else None

    def pick_distinct_secondary_gu(primary_path, rank, exclude_set, target_id):
        # Xoay vòng ứng viên một cách deterministic theo target_id để các
        # công thức khác nhau cùng bucket dung nguyên liệu khác nhau, thay vì
        # luôn chọn cổ đầu tiên (Dưỡng Đan Tiên Cổ từng bị dính vào > 450
        # công thức rank 8 vì luôn là phần tử đầu của Refinement Path).
        seed = zlib.crc32(target_id.encode('utf-8'))

        def take(bucket):
            cands = [gid for gid in bucket if gid not in exclude_set]
            if not cands:
                return None
            chosen = cands[seed % len(cands)]
            exclude_set.add(chosen)
            return chosen

        chosen = take(gus_by_path_rank.get((primary_path, rank), []))
        if chosen:
            return chosen
        for fallback_p in ['Refinement Path', 'Qi Path', 'Strength Path']:
            chosen = take(gus_by_path_rank.get((fallback_p, rank), []))
            if chosen:
                return chosen
        return take(gus_by_rank.get(rank, []))

    old_recipes = list(csv.DictReader(open(recipe_file, 'r', encoding='utf-8-sig')))
    new_recipes = []

    # Danh sách công thức gốc từ file cũ, cộng thêm mọi Gu rank 6-9 chưa có
    # công thức (Gu mới thêm vào Gu.csv sau lần build trước sẽ được phủ kín).
    covered_targets = {(row['target_gu_id'], int(row['target_rank'])) for row in old_recipes}
    pending_targets = [
        (row['target_gu_id'], int(row['target_rank']), int(row['variant']))
        for row in old_recipes
    ]
    for (gid, grk) in sorted(gus_by_id_rank.keys(), key=lambda t: (t[1], t[0])):
        if 6 <= grk <= 9 and (gid, grk) not in covered_targets:
            pending_targets.append((gid, grk, 1))
            covered_targets.add((gid, grk))

    for target_id, target_rank, variant in pending_targets:
        gmeta = gus_by_id_rank.get((target_id, target_rank))
        primary_path = gmeta[1][0] if gmeta else 'Strength Path'
        
        req_gus = {}
        req_mats = {}
        used_gus = set()
        
        mat_a = dai_a.get(primary_path, {}).get(target_rank)
        mat_b = dai_b.get(primary_path, {}).get(target_rank)
        
        if target_rank == 1:
            req_gus = {}
            if mat_a:
                req_mats[mat_a] = 8
            if mat_b:
                req_mats[mat_b] = 5
            success_rate = 0.90
            primeval_cost = 100.0
            immortal_cost = 0.0

        elif target_rank == 2:
            main_gu = find_primary_predecessor(target_id, 2, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 1}
                used_gus.add(main_gu)
            if mat_a:
                req_mats[mat_a] = 16
            if mat_b:
                req_mats[mat_b] = 10
            success_rate = 0.75
            primeval_cost = 300.0
            immortal_cost = 0.0

        elif target_rank == 3:
            main_gu = find_primary_predecessor(target_id, 3, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 2}
                used_gus.add(main_gu)
            if mat_a:
                req_mats[mat_a] = 28
            if mat_b:
                req_mats[mat_b] = 18
            req_mats["qi_stone_rank3"] = 8
            success_rate = 0.60
            primeval_cost = 800.0
            immortal_cost = 0.0

        elif target_rank == 4:
            main_gu = find_primary_predecessor(target_id, 4, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 3}
                used_gus.add(main_gu)
            sub_gu = pick_distinct_secondary_gu(primary_path, 2, used_gus, target_id)
            if sub_gu:
                req_gus[sub_gu] = {"qty": 1, "rank": 2}
            if mat_a:
                req_mats[mat_a] = 45
            if mat_b:
                req_mats[mat_b] = 32
            req_mats["refinement_iron_rank4"] = 15
            success_rate = 0.45
            primeval_cost = 2500.0
            immortal_cost = 0.0

        elif target_rank == 5:
            main_gu = find_primary_predecessor(target_id, 5, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 4}
                used_gus.add(main_gu)
            sub1 = pick_distinct_secondary_gu(primary_path, 4, used_gus, target_id)
            if sub1:
                req_gus[sub1] = {"qty": 1, "rank": 4}
            sub2 = pick_distinct_secondary_gu(primary_path, 3, used_gus, target_id)
            if sub2:
                req_gus[sub2] = {"qty": 1, "rank": 3}
            if mat_a:
                req_mats[mat_a] = 75
            if mat_b:
                req_mats[mat_b] = 55
            req_mats["qi_stone_rank5"] = 25
            req_mats["langya_refinement_shard_rank5"] = 20
            success_rate = 0.30
            primeval_cost = 10000.0
            immortal_cost = 0.0

        elif target_rank == 6:
            main_gu = find_primary_predecessor(target_id, 6, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 5}
                used_gus.add(main_gu)
            g_r5_sub = pick_distinct_secondary_gu(primary_path, 5, used_gus, target_id)
            if g_r5_sub:
                req_gus[g_r5_sub] = {"qty": 1, "rank": 5}
            g_r4_sub1 = pick_distinct_secondary_gu(primary_path, 4, used_gus, target_id)
            if g_r4_sub1:
                req_gus[g_r4_sub1] = {"qty": 1, "rank": 4}
            g_r4_sub2 = pick_distinct_secondary_gu('Refinement Path', 4, used_gus, target_id)
            if g_r4_sub2:
                req_gus[g_r4_sub2] = {"qty": 1, "rank": 4}
            if mat_a:
                req_mats[mat_a] = 120
            if mat_b:
                req_mats[mat_b] = 90
            req_mats["refinement_iron_rank6"] = 50
            req_mats["qi_dragon_scale_rank6"] = 35
            success_rate = 0.08
            primeval_cost = 50000.0
            immortal_cost = 200.0

        elif target_rank == 7:
            main_gu = find_primary_predecessor(target_id, 7, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 6}
                used_gus.add(main_gu)
            g_immortal_r6_sub = pick_distinct_secondary_gu(primary_path, 6, used_gus, target_id)
            if g_immortal_r6_sub:
                req_gus[g_immortal_r6_sub] = {"qty": 1, "rank": 6}
            g_r5_sub1 = pick_distinct_secondary_gu(primary_path, 5, used_gus, target_id)
            if g_r5_sub1:
                req_gus[g_r5_sub1] = {"qty": 1, "rank": 5}
            g_r5_sub2 = pick_distinct_secondary_gu('Refinement Path', 5, used_gus, target_id)
            if g_r5_sub2:
                req_gus[g_r5_sub2] = {"qty": 1, "rank": 5}
            g_r4_sub = pick_distinct_secondary_gu('Qi Path', 4, used_gus, target_id)
            if g_r4_sub:
                req_gus[g_r4_sub] = {"qty": 1, "rank": 4}
            if mat_a:
                req_mats[mat_a] = 200
            if mat_b:
                req_mats[mat_b] = 160
            mat_a_lower = dai_a.get(primary_path, {}).get(6)
            if mat_a_lower:
                req_mats[mat_a_lower] = 80
            req_mats["langya_refinement_shard_rank7"] = 70
            req_mats["qi_stone_rank7"] = 50
            success_rate = 0.03
            primeval_cost = 200000.0
            immortal_cost = 800.0

        elif target_rank == 8:
            main_gu = find_primary_predecessor(target_id, 8, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 7}
                used_gus.add(main_gu)
            g_immortal_r7_sub = pick_distinct_secondary_gu(primary_path, 7, used_gus, target_id)
            if g_immortal_r7_sub:
                req_gus[g_immortal_r7_sub] = {"qty": 1, "rank": 7}
            g_immortal_r6_sub1 = pick_distinct_secondary_gu(primary_path, 6, used_gus, target_id)
            if g_immortal_r6_sub1:
                req_gus[g_immortal_r6_sub1] = {"qty": 1, "rank": 6}
            g_immortal_r6_sub2 = pick_distinct_secondary_gu('Refinement Path', 6, used_gus, target_id)
            if g_immortal_r6_sub2:
                req_gus[g_immortal_r6_sub2] = {"qty": 1, "rank": 6}
            g_r5_sub = pick_distinct_secondary_gu(primary_path, 5, used_gus, target_id)
            if g_r5_sub:
                req_gus[g_r5_sub] = {"qty": 1, "rank": 5}
            if mat_a:
                req_mats[mat_a] = 350
            if mat_b:
                req_mats[mat_b] = 280
            mat_b_lower = dai_b.get(primary_path, {}).get(7)
            if mat_b_lower:
                req_mats[mat_b_lower] = 150
            req_mats["refinement_iron_rank8"] = 120
            req_mats["qi_dragon_scale_rank8"] = 90
            success_rate = 0.01
            primeval_cost = 1000000.0
            immortal_cost = 4000.0

        elif target_rank == 9:
            main_gu = find_primary_predecessor(target_id, 9, primary_path)
            if main_gu:
                req_gus[main_gu] = {"qty": 1, "rank": 8}
                used_gus.add(main_gu)
            g_immortal_r8_sub = pick_distinct_secondary_gu(primary_path, 8, used_gus, target_id)
            if g_immortal_r8_sub:
                req_gus[g_immortal_r8_sub] = {"qty": 1, "rank": 8}
            g_immortal_r7_sub1 = pick_distinct_secondary_gu(primary_path, 7, used_gus, target_id)
            if g_immortal_r7_sub1:
                req_gus[g_immortal_r7_sub1] = {"qty": 1, "rank": 7}
            g_immortal_r7_sub2 = pick_distinct_secondary_gu('Refinement Path', 7, used_gus, target_id)
            if g_immortal_r7_sub2:
                req_gus[g_immortal_r7_sub2] = {"qty": 1, "rank": 7}
            g_immortal_r6_sub = pick_distinct_secondary_gu('Qi Path', 6, used_gus, target_id) or pick_distinct_secondary_gu(primary_path, 6, used_gus, target_id)
            if g_immortal_r6_sub:
                req_gus[g_immortal_r6_sub] = {"qty": 1, "rank": 6}
            g_r5_sub = pick_distinct_secondary_gu(primary_path, 5, used_gus, target_id)
            if g_r5_sub:
                req_gus[g_r5_sub] = {"qty": 1, "rank": 5}
            if mat_a:
                req_mats[mat_a] = 600
            if mat_b:
                req_mats[mat_b] = 500
            mat_a_lower = dai_a.get(primary_path, {}).get(8)
            if mat_a_lower:
                req_mats[mat_a_lower] = 250
            mat_b_lower = dai_b.get(primary_path, {}).get(8)
            if mat_b_lower:
                req_mats[mat_b_lower] = 200
            req_mats["langya_refinement_shard_rank9"] = 150
            success_rate = 0.002
            primeval_cost = 5000000.0
            immortal_cost = 20000.0

        new_recipes.append({
            'target_gu_id': target_id,
            'target_rank': target_rank,
            'variant': variant,
            'required_gus': json.dumps(req_gus, ensure_ascii=False),
            'required_materials': json.dumps(req_mats, ensure_ascii=False),
            'success_rate': f"{success_rate:.6f}".rstrip('0').rstrip('.'),
            'primeval_stones_cost': f"{primeval_cost:.3f}",
            'immortal_essence_stones_cost': f"{immortal_cost:.3f}"
        })

    fieldnames = ["target_gu_id", "target_rank", "variant", "required_gus", "required_materials", "success_rate", "primeval_stones_cost", "immortal_essence_stones_cost"]

    with open(recipe_file, 'w', encoding='utf-8', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(new_recipes)

    print(f"rebuild_recipes: Đã tái tạo {len(new_recipes)} công thức vào {recipe_file}!")

if __name__ == "__main__":
    main()
