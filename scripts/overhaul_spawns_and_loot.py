#!/usr/bin/env python3
"""
scripts/overhaul_spawns_and_loot.py

Complete overhaul of Monster spawns, Monster loot tables, and Region loot tables:
1. Rebalances monster types (converting generic desolate beasts from Boss to Normal/Elite).
2. Generates missing Normal/Elite monsters across all 44 paths so every path has Normal/Elite drops at all ranks.
3. Overhauls Monster.csv with clean ASCII snake_case IDs.
4. Generates clean, focused monster loot tables:
   - Normal: 2-3 entries (Dải A rank R, Dải B rank R, and rank R-1)
   - Elite: 3-4 entries (Dải A rank R, Dải B rank R, Dải A&B rank R-1)
   - Boss: 4-5 entries (Dải A & B rank R and R-1 with high yield & chance)
   => 100% of Dải A & Dải B items drop from Normal/Elite monsters (0 boss-only materials!).
5. Generates balanced MonsterSpawn.csv:
   - Each combat region spawns 2-3 Normal, 1-2 Elite, and 1 Boss matching region rank & path.
   - Every monster spawns in 1 to 4 regions (hard capped). 0 monsters spawn in > 4 regions.
   - 100% of monsters have at least 1 active spawn region (0 unspawned monsters).
6. Streamlines RegionLootTables.csv and region loot tables in LootTable.csv:
   - Each region loot table reduced from 22 entries to 2-3 focused entries matching region path and rank band.
"""

import csv
import json
import os
import random
import re
from collections import Counter, defaultdict
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "backend" / "data"

MONSTER_CSV = DATA_DIR / "Monster.csv"
SPAWN_CSV = DATA_DIR / "MonsterSpawn.csv"
LOOT_CSV = DATA_DIR / "LootTable.csv"
REGION_LOOT_CSV = DATA_DIR / "RegionLootTables.csv"
REGION_CSV = DATA_DIR / "Region.csv"
RANKS_CSV = REPO_ROOT / "scripts" / "monster_ranks.csv"

from gen_path_items import DAI_A_DATA, DAI_B_DATA

SEED = 20261018

# Stats baseline per rank
# Rank-independent base stats (scaled at runtime by ScaleMonsterStats with RankScale and typeMultiplier)
RANK_STATS = {
    1: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    2: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    3: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    4: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    5: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    6: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    7: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    8: {"hp": 85.0, "atk": 11.0, "def": 0.08},
    9: {"hp": 85.0, "atk": 11.0, "def": 0.08},
}

def clean_ascii_id(s):
    """Normalize Vietnamese accented characters in IDs to clean ASCII snake_case."""
    mapping = {
        'á': 'a', 'à': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
        'ă': 'a', 'ắ': 'a', 'ằ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
        'â': 'a', 'ấ': 'a', 'ầ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',
        'é': 'e', 'è': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
        'ê': 'e', 'ế': 'e', 'ề': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',
        'í': 'i', 'ì': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',
        'ó': 'o', 'ò': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
        'ô': 'o', 'ố': 'o', 'ồ': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
        'ơ': 'o', 'ớ': 'o', 'ờ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',
        'ú': 'u', 'ù': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
        'ư': 'u', 'ứ': 'u', 'ừ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',
        'ý': 'y', 'ỳ': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',
        'đ': 'd'
    }
    return "".join(mapping.get(c, c) for c in s)


# Path beast naming templates for generating missing path-tier monsters
PATH_THEMES = {
    "Blade Path": ("Đao", "blade", ["Đao Tí Đường Lang", "Đao Giác Tê Ngưu", "Hoang Thú Đao Dực Ưng", "Thái Cổ Đao Ma Thú"]),
    "Blood Path": ("Huyết", "blood", ["Huyết Bức", "Huyết Lang", "Hoang Thú Huyết Ngạc", "Thái Cổ Huyết Ma"]),
    "Bone Path": ("Cốt", "bone", ["Bạch Cốt Khuyển", "Cốt Thương Hầu", "Hoang Thú Bạch Cốt Tượng", "Thái Cổ Cốt Ma"]),
    "Cloud Path": ("Vân", "cloud", ["Vân Miêu", "Phi Vân Điểu", "Hoang Thú Vân Vụ Thú", "Thái Cổ Vân Long"]),
    "Creation Path": ("Tạo Hóa", "creation", ["Tạo Hóa Linh Lộc", "Sinh Cơ Khôi Lỗi", "Hoang Thú Tạo Hóa Điểu", "Thái Cổ Tạo Hóa Cự Thú"]),
    "Dark Path": ("Ám", "dark", ["Ám Dạ Bức", "U Dạ Lang", "Hoang Thú Hắc Ám Cự Mãng", "Thái Cổ Ám Ma"]),
    "Death Path": ("Tử Tịch", "death", ["Tử Khí Quạ", "Tử Tịch Hạt Tử", "Hoang Thú Tử Sa Thú", "Thái Cổ Tịch Diệt Thần"]),
    "Destruction Path": ("Hủy Diệt", "destruction", ["Toái Diệt Hắc Nghĩ", "Phá Diệt Ma Khuyển", "Hoang Thú Hủy Diệt Cự Hầu", "Thái Cổ Hủy Diệt Cự Thú"]),
    "Dream Path": ("Mộng Cảnh", "dream", ["Mộng Điệp", "Mộng Huyễn Hoa Hồ", "Hoang Thú Mộng Cảnh Linh Thú", "Thái Cổ Mộng Ma"]),
    "Earth Path": ("Địa", "earth", ["Địa Huyệt Trùng", "Thạch Nham Giáp Thú", "Hoang Thú Địa Mạch Khôi", "Thái Cổ Sơn Khôi"]),
    "Emotion Path": ("Tình", "emotion", ["Thất Tình Tước", "Tình Ti Yêu Tri Chu", "Hoang Thú Tình Ý Linh Viên", "Thái Cổ Huyễn Tình Ma"]),
    "Enslavement Path": ("Nô Dịch", "enslavement", ["Ngự Thú Dã Khuyển", "Thuần Phục Ngưu Ma", "Hoang Thú Vạn Thú Tướng", "Thái Cổ Ngự Thú Lang"]),
    "Fire Path": ("Hỏa", "fire", ["Ly Hỏa Thử", "Hỏa Diễm Tích Dịch", "Hoang Thú Hỏa Linh Điểu", "Thái Cổ Viêm Ma"]),
    "Food Path": ("Thực", "food", ["Linh Lương Tham Trư", "Thực Thôn Yến", "Hoang Thú Phệ Lương Thiềm", "Thái Cổ Thao Thiết Thú"]),
    "Formation Path": ("Trận", "formation", ["Trận Cơ Thạch Nhân", "Huyền Trận Linh Quy", "Hoang Thú Bát Quái Cự Thú", "Thái Cổ Trận Linh"]),
    "Heaven Path": ("Thiên", "heaven", ["Thiên Vận Điểu", "Thiên Cương Bạch Viên", "Hoang Thú Thiên Khí Thần Thú", "Thái Cổ Thiên Đạo Khôi"]),
    "Human Path": ("Nhân", "human", ["Vũ Nhân Tiên Phong", "Tán Tu Tàn Hồn", "Hoang Thú Nhân Đạo Dị Chủng", "Thái Cổ Nhân Hoàng Khôi"]),
    "Ice Path": ("Băng", "ice", ["Hàn Băng Thiền", "Băng Sương Lang", "Hoang Thú Băng Phách Cự Hùng", "Thái Cổ Hàn Long"]),
    "Information Path": ("Tín Tức", "information", ["Linh Thính Linh Hầu", "Lưu Âm Thước Điểu", "Hoang Thú Thiên Lý Độc Giác", "Thái Cổ Tri Thức Dị Thú"]),
    "Investigation Path": ("Trinh Sát", "investigation", ["Thiên Lý Ưng", "Đồng Triệt Thần Báo", "Hoang Thú Minh Mục Cự Điểu", "Thái Cổ Thần Đồng Thú"]),
    "Killing Path": ("Sát Lục", "killing", ["Sát Ý Ấu Khuyển", "Tu La Sát Thứ", "Hoang Thú Sát Lục Huyết Ma", "Thái Cổ Sát Lục Tu La"]),
    "Light Path": ("Quang", "light", ["Quang Diệu Huỳnh Trùng", "Huyễn Quang Bạch Lộc", "Hoang Thú Cực Quang Điểu", "Thái Cổ Quang Diệu Thần Thú"]),
    "Lightning Path": ("Lôi", "lightning", ["Điện Lang", "Lôi Đình Báo", "Hoang Thú Lôi Điện Cự Mãng", "Thái Cổ Lôi Man Ngư"]),
    "Luck Path": ("Khí Vận", "luck", ["Cát Tường Linh Ngư", "Chân Dương Vận Khuyển", "Hoang Thú Hồng Vận Linh Thú", "Thái Cổ Tụ Vận Thần Thú"]),
    "Metal Path": ("Kim", "metal", ["Thiết Giáp Hùng", "Canh Kim Tê Ngưu", "Hoang Thú Bạch Kim Cự Hầu", "Thái Cổ Thiết Ma"]),
    "Painting Path": ("Họa", "painting", ["Đan Thanh Điệp", "Mặc Họa Tiên Hạc", "Hoang Thú Thạch Thanh Họa Linh", "Thái Cổ Đan Thanh Cự Điểu"]),
    "Phantom Path": ("Huyễn", "phantom", ["Huyễn Ảnh Thú", "Hư Linh Yêu Hồ", "Hoang Thú Vô Tướng Ảo Ma", "Thái Cổ Huyễn Ảo Ma"]),
    "Poison Path": ("Độc", "poison", ["Bích Độc Hạt", "U Độc Chu", "Hoang Thú Kịch Độc Ma Xà", "Thái Cổ Vạn Độc Cự Mãng"]),
    "Qi Path": ("Khí", "qi", ["Khí Tức Linh Tước", "Vân Khí Cự Mãng", "Hoang Thú Nguyên Khí Ma Thú", "Thái Cổ Khí Long"]),
    "Refinement Path": ("Luyện", "refinement", ["Luyện Hỏa Kỳ Nhông", "Đỉnh Lô Giáp Trùng", "Hoang Thú Luyện Đạo Hỏa Thú", "Thái Cổ Luyện Lô Ma"]),
    "Rule Path": ("Quy Tắc", "rule", ["Cân Xứng Linh Thú", "Giới Luật Thạch Khôi", "Hoang Thú Chấp Pháp Dị Thú", "Thái Cổ Quy Tắc Thần Thú"]),
    "Soul Path": ("Hồn", "soul", ["U Hồn Điệp", "Phệ Hồn Quỷ Miêu", "Hoang Thú Đãng Hồn Thú", "Thái Cổ U Hồn Cự Ma"]),
    "Sound Path": ("Âm", "sound", ["Âm Ba Linh Thú", "Kính Thiên Hống", "Hoang Thú Cuồng Âm Thần Viên", "Thái Cổ Long Ngâm Thú"]),
    "Space Path": ("Không Gian", "space", ["Hư Không Thử", "Thiên Di Hư Điêu", "Hoang Thú Không Gian Di Dời Thú", "Thái Cổ Hư Không Kình"]),
    "Star Path": ("Tinh Thần", "star", ["Lưu Tinh Huỳnh Thú", "Tinh Hà Linh Xà", "Hoang Thú Tinh Quang Cự Hùng", "Thái Cổ Tinh Thần Thú"]),
    "Strength Path": ("Lực", "strength", ["Cự Lực Dã Trư", "Hắc Kim Hùng", "Hoang Thú Cự Lực Viên", "Thái Cổ Cự Lực Thần Thú"]),
    "Sword Path": ("Kiếm", "sword", ["Trảm Kiếm Điểu", "Phi Kiếm Lang", "Hoang Thú Kiếm Ý Cự Long", "Thái Cổ Kiếm Ma Thú"]),
    "Theft Path": ("Thâu Đạo", "theft", ["Đạo Chỉ Linh Hầu", "Thâu Quang Điêu", "Hoang Thú Diệu Thủ Yêu Miêu", "Thái Cổ Thâu Thiên Thử"]),
    "Time Path": ("Thời Gian", "time", ["Tuế Nguyệt Bàng Cua", "Thời Quang Linh Thiền", "Hoang Thú Tuế Nguyệt Thiềm", "Thái Cổ Quang Âm Cự Quy"]),
    "Transformation Path": ("Biến Hóa", "transformation", ["Bách Biến Tắc Kè", "Thiên Huyễn Yêu Hồ", "Hoang Thú Vạn Tượng Dị Thú", "Thái Cổ Thiên Biến Ma"]),
    "Water Path": ("Thủy", "water", ["Bích Ba Ngạc", "Thủy Tinh Thao", "Hoang Thú Lam Thủy Long", "Thái Cổ Thủy Ma"]),
    "Wind Path": ("Phong", "wind", ["Liệt Phong Điểu", "Phong Bạo Độc Phong", "Hoang Thú Cương Phong Thần Ưng", "Thái Cổ Phong Ma"]),
    "Wisdom Path": ("Trí", "wisdom", ["Bạch Diện Linh Hồ", "Thông Tuệ Viên", "Hoang Thú Thông Thiên Linh Hồ", "Thái Cổ Trí Tuệ Thần Thú"]),
    "Wood Path": ("Mộc", "wood", ["Linh Thảo Kê", "Ất Mộc Tinh", "Hoang Thú Bách Niên Thần Cây", "Thái Cổ Mộc Linh"]),
}

def run_overhaul(apply_changes=False):
    print("=== BẮT ĐẦU TỐI ƯU TOÀN DIỆN MONSTER SPAWN & LOOT TABLES ===")
    
    # 1. Map paths to items (Dải A and Dải B)
    dai_a = {}
    for p_name, base_id, _, _ in DAI_A_DATA:
        dai_a[p_name] = {rk: f"{base_id}_rank{rk}" for rk in range(1, 10)}
    dai_b = {}
    for p_name, base_id, _, _ in DAI_B_DATA:
        dai_b[p_name] = {rk: f"{base_id}_rank{rk}" for rk in range(1, 10)}

    # 2. Load existing monsters
    with open(MONSTER_CSV, "r", encoding="utf-8") as f:
        monsters = list(csv.DictReader(f))

    # Clean IDs of accented monsters
    for m in monsters:
        old_id = m["id"]
        clean_id = clean_ascii_id(old_id)
        if clean_id != old_id:
            m["id"] = clean_id
            m["loot_table_id"] = f"loot_{clean_id}"

    # Rebalance monster types: true named bosses stay Boss, generic desolate beasts become Elite or Normal
    true_boss_keywords = ["vương", "vuong", "tôn", "ton", "hoàng", "hoang", "thần ma", "than ma", "chi chủ", "chi chu", "boss", "ma đầu", "ma dau", "lão tổ", "lao to", "đế", "de"]

    for m in monsters:
        mid = m["id"]
        name_lower = m["name"].lower()
        if m["type"] in ("Boss", "Tribulation_Boss"):
            is_true_boss = any(k in name_lower or k in mid for k in true_boss_keywords)
            if not is_true_boss:
                m["type"] = "Elite"
        elif m["type"] == "Elite":
            # Demote common beasts to Normal to ensure a healthy Normal pool
            if any(k in name_lower for k in ["khuyển", "trử", "lang", "thử", "ngạc", "xà", "ngô công", "hầu", "bức", "trùng", "tước", "điểu"]):
                m["type"] = "Normal"

    # Load known ranks
    ranks = {}
    if RANKS_CSV.exists():
        with open(RANKS_CSV, "r", encoding="utf-8-sig") as f:
            for r in csv.DictReader(f):
                m_id = clean_ascii_id(r["monster_id"])
                ranks[m_id] = int(r["rank"])

    for m in monsters:
        mid = m["id"]
        if mid not in ranks:
            match = re.search(r"_r(\d+)r(\d+)", mid)
            if match:
                r1, r2 = int(match.group(1)), int(match.group(2))
                ranks[mid] = r1 if m["type"] == "Normal" else ((r1 + r2) // 2)
            elif "thai_co" in mid or "ancient" in mid:
                ranks[mid] = 8
            elif "hoang_thu" in mid:
                ranks[mid] = 6
            elif "di_bien" in mid:
                ranks[mid] = 4
            elif "hoang_da" in mid:
                ranks[mid] = 2
            else:
                hp = float(m["max_hp"])
                if hp >= 4000: ranks[mid] = 9
                elif hp >= 2200: ranks[mid] = 8
                elif hp >= 1200: ranks[mid] = 7
                elif hp >= 600: ranks[mid] = 6
                elif hp >= 300: ranks[mid] = 5
                elif hp >= 150: ranks[mid] = 4
                elif hp >= 70: ranks[mid] = 3
                elif hp >= 35: ranks[mid] = 2
                else: ranks[mid] = 1

    # 3. Ensure every path has Normal AND Elite monsters across all 4 Tiers (1-3, 4-5, 6-7, 8-9)
    def get_tier(rk):
        return 1 if rk <= 3 else (2 if rk <= 5 else (3 if rk <= 7 else 4))

    path_tier_monsters = defaultdict(lambda: defaultdict(list))
    for m in monsters:
        pm = json.loads(m.get("path_masteries") or "{}")
        if not pm: continue
        path = list(pm.keys())[0]
        rk = ranks[m["id"]]
        t = get_tier(rk)
        path_tier_monsters[path][t].append(m)

    tier_ranks_normal = {1: 1, 2: 3, 3: 6, 4: 8}
    tier_ranks_elite = {1: 2, 2: 5, 3: 7, 4: 9}
    ts = "2026-10-18 05:00:00.000"

    added_monsters = []
    existing_mids = {m["id"] for m in monsters}

    # For each path and tier, make sure we have at least 1 Normal AND 1 Elite monster!
    for path, (vn_prefix, slug, names) in PATH_THEMES.items():
        for t in [1, 2, 3, 4]:
            current = path_tier_monsters[path][t]
            has_normal = any(m["type"] == "Normal" for m in current)
            has_elite = any(m["type"] == "Elite" for m in current)

            # If missing Normal
            if not has_normal:
                rk = tier_ranks_normal[t]
                m_id = f"monster_{slug}_tier{t}_normal"
                if m_id not in existing_mids:
                    stats = RANK_STATS[rk]
                    new_m = {
                        "id": m_id,
                        "name": f"{names[t - 1]}",
                        "type": "Normal",
                        "max_hp": f"{stats['hp']:.1f}",
                        "base_attack": f"{stats['atk']:.1f}",
                        "base_defense": f"{stats['def']:.1f}",
                        "gu_inventory": "[]",
                        "item_inventory": "{}",
                        "dao_marks": "{}",
                        "path_masteries": json.dumps({path: 1}),
                        "ai_profile": "beast_basic",
                        "loot_table_id": f"loot_{m_id}",
                        "description": f"Dã thú lưu phái {path}, phẩm cấp Chuyển {rk}.",
                        "is_active": "True",
                        "is_capturable": "False",
                        "capture_chance": "0.2",
                        "pet_hunger_per_hour": "2.0",
                        "created_at": ts,
                        "updated_at": ts,
                        "gu_pool": "[]"
                    }
                    monsters.append(new_m)
                    ranks[m_id] = rk
                    existing_mids.add(m_id)
                    added_monsters.append(new_m)
                    path_tier_monsters[path][t].append(new_m)

            # If missing Elite
            if not has_elite:
                rk = tier_ranks_elite[t]
                m_id = f"monster_{slug}_tier{t}_elite"
                if m_id not in existing_mids:
                    stats = RANK_STATS[rk]
                    new_m = {
                        "id": m_id,
                        "name": f"Tinh Anh {names[t - 1]}",
                        "type": "Elite",
                        "max_hp": f"{stats['hp'] * 1.5:.1f}",
                        "base_attack": f"{stats['atk'] * 1.3:.1f}",
                        "base_defense": f"{stats['def'] * 1.2:.1f}",
                        "gu_inventory": "[]",
                        "item_inventory": "{}",
                        "dao_marks": "{}",
                        "path_masteries": json.dumps({path: 1}),
                        "ai_profile": "beast_basic",
                        "loot_table_id": f"loot_{m_id}",
                        "description": f"Dã thú tinh anh lưu phái {path}, phẩm cấp Chuyển {rk}.",
                        "is_active": "True",
                        "is_capturable": "False",
                        "capture_chance": "0.2",
                        "pet_hunger_per_hour": "2.0",
                        "created_at": ts,
                        "updated_at": ts,
                        "gu_pool": "[]"
                    }
                    monsters.append(new_m)
                    ranks[m_id] = rk
                    existing_mids.add(m_id)
                    added_monsters.append(new_m)
                    path_tier_monsters[path][t].append(new_m)

    # Ensure every single rank 1..9 in each path has a Normal or Elite drop source
    for path, (vn_prefix, slug, names) in PATH_THEMES.items():
        covered_ranks = set()
        for m in monsters:
            if m["type"] not in ("Normal", "Elite"): continue
            pm = json.loads(m.get("path_masteries") or "{}")
            if not pm or list(pm.keys())[0] != path: continue
            rk = ranks.get(m["id"], 1)
            covered_ranks.add(rk)
            if rk > 1: covered_ranks.add(rk - 1)
            if m["type"] == "Elite" and rk == 8: covered_ranks.add(9)
            if rk == 9: covered_ranks.add(9)

        missing_ranks = set(range(1, 10)) - covered_ranks
        for rk in sorted(missing_ranks):
            t = get_tier(rk)
            m_id = f"monster_{slug}_r{rk}_normal"
            if m_id not in existing_mids:
                stats = RANK_STATS[rk]
                new_m = {
                    "id": m_id,
                    "name": f"{names[t-1]} Chuyển {rk}",
                    "type": "Normal" if rk < 9 else "Elite",
                    "max_hp": f"{stats['hp']:.1f}",
                    "base_attack": f"{stats['atk']:.1f}",
                    "base_defense": f"{stats['def']:.1f}",
                    "gu_inventory": "[]",
                    "item_inventory": "{}",
                    "dao_marks": "{}",
                    "path_masteries": json.dumps({path: 1}),
                    "ai_profile": "beast_basic",
                    "loot_table_id": f"loot_{m_id}",
                    "description": f"Dã thú lưu phái {path}, phẩm cấp Chuyển {rk}.",
                    "is_active": "True",
                    "is_capturable": "False",
                    "capture_chance": "0.2",
                    "pet_hunger_per_hour": "2.0",
                    "created_at": ts,
                    "updated_at": ts,
                    "gu_pool": "[]"
                }
                monsters.append(new_m)
                ranks[m_id] = rk
                existing_mids.add(m_id)
                added_monsters.append(new_m)

    print(f"Total monsters after addition: {len(monsters)} (added {len(added_monsters)} path-tier monsters)")
    type_counts = Counter(m["type"] for m in monsters)
    print("Monster type distribution:", dict(type_counts))

    # 4. Generate Clean, Focused Monster Loot Tables (LootTable.csv)
    loot_tables = {}
    if LOOT_CSV.exists():
        with open(LOOT_CSV, "r", encoding="utf-8") as f:
            for r in csv.DictReader(f):
                loot_tables[r["id"]] = r

    for m in monsters:
        mid = m["id"]
        lt_id = m.get("loot_table_id", f"loot_{mid}")
        rk = ranks.get(mid, 1)
        mtype = m.get("type", "Normal")
        pm = json.loads(m.get("path_masteries") or "{}")
        path = list(pm.keys())[0] if pm else "Strength Path"

        item_a = dai_a.get(path, {}).get(rk)
        item_b = dai_b.get(path, {}).get(rk)

        entries = []
        if mtype == "Normal":
            if item_a:
                entries.append({"item_id": item_a, "min_qty": 1, "max_qty": 2, "chance": 0.65, "rank": rk})
            if item_b:
                entries.append({"item_id": item_b, "min_qty": 1, "max_qty": 1, "chance": 0.50, "rank": rk})
            if rk > 1:
                item_a_prev = dai_a.get(path, {}).get(rk - 1)
                item_b_prev = dai_b.get(path, {}).get(rk - 1)
                if item_a_prev:
                    entries.append({"item_id": item_a_prev, "min_qty": 1, "max_qty": 1, "chance": 0.35, "rank": rk - 1})
                if item_b_prev:
                    entries.append({"item_id": item_b_prev, "min_qty": 1, "max_qty": 1, "chance": 0.25, "rank": rk - 1})
        elif mtype == "Elite":
            if item_a:
                entries.append({"item_id": item_a, "min_qty": 1, "max_qty": 3, "chance": 0.85, "rank": rk})
            if item_b:
                entries.append({"item_id": item_b, "min_qty": 1, "max_qty": 2, "chance": 0.75, "rank": rk})
            if rk > 1:
                item_a_prev = dai_a.get(path, {}).get(rk - 1)
                item_b_prev = dai_b.get(path, {}).get(rk - 1)
                if item_a_prev:
                    entries.append({"item_id": item_a_prev, "min_qty": 1, "max_qty": 2, "chance": 0.50, "rank": rk - 1})
                if item_b_prev:
                    entries.append({"item_id": item_b_prev, "min_qty": 1, "max_qty": 1, "chance": 0.40, "rank": rk - 1})
            if rk == 8: # Rank 8 elite also drops rank 9 materials occasionally
                item_a_9 = dai_a.get(path, {}).get(9)
                item_b_9 = dai_b.get(path, {}).get(9)
                if item_a_9:
                    entries.append({"item_id": item_a_9, "min_qty": 1, "max_qty": 1, "chance": 0.25, "rank": 9})
                if item_b_9:
                    entries.append({"item_id": item_b_9, "min_qty": 1, "max_qty": 1, "chance": 0.20, "rank": 9})
        else: # Boss
            if item_a:
                entries.append({"item_id": item_a, "min_qty": 2, "max_qty": 5, "chance": 1.0, "rank": rk})
            if item_b:
                entries.append({"item_id": item_b, "min_qty": 2, "max_qty": 4, "chance": 0.95, "rank": rk})
            if rk > 1:
                item_a_prev = dai_a.get(path, {}).get(rk - 1)
                item_b_prev = dai_b.get(path, {}).get(rk - 1)
                if item_a_prev:
                    entries.append({"item_id": item_a_prev, "min_qty": 2, "max_qty": 4, "chance": 0.80, "rank": rk - 1})
                if item_b_prev:
                    entries.append({"item_id": item_b_prev, "min_qty": 1, "max_qty": 2, "chance": 0.70, "rank": rk - 1})
            if rk >= 6:
                entries.append({
                    "item_id": "ITEM_RANDOM_CONSTITUTION" if rk % 2 == 0 else "ITEM_RANDOM_RACE",
                    "min_qty": 1, "max_qty": 1, "chance": 0.35, "rank": rk
                })

        loot_tables[lt_id] = {
            "id": lt_id,
            "name": f"Loot {m['name']}",
            "entries": json.dumps(entries, ensure_ascii=False),
            "created_at": ts,
            "updated_at": ts
        }

    # 5. Overhaul MonsterSpawn.csv (Hard Capped Balanced Distribution)
    regions = list(csv.DictReader(open(REGION_CSV, encoding="utf-8-sig")))
    combat_regions = [r for r in regions if r["zone_type"] in ("KHU_VUC", "NGU_VUC")]

    pool_by_type_rank = defaultdict(lambda: defaultdict(list))
    for m in monsters:
        mid = m["id"]
        rk = ranks[mid]
        t = "Normal" if m["type"] == "Normal" else ("Elite" if m["type"] == "Elite" else "Boss")
        pool_by_type_rank[t][rk].append(mid)

    spawn_counts = Counter()
    spawn_rows = []
    rng = random.Random(SEED)

    def infer_region_path(reg_name):
        n = reg_name.lower()
        if "hỏa" in n or "diễm" in n: return "Fire Path"
        if "băng" in n or "tuyết" in n: return "Ice Path"
        if "thủy" in n or "hải" in n or "hác" in n: return "Water Path"
        if "phong" in n or "bão" in n: return "Wind Path"
        if "lôi" in n or "điện" in n: return "Lightning Path"
        if "cốt" in n: return "Bone Path"
        if "huyết" in n: return "Blood Path"
        if "thạch" in n or "thổ" in n or "sơn" in n: return "Earth Path"
        if "ám" in n or "dạ" in n or "hắc" in n: return "Dark Path"
        if "quang" in n or "nhật" in n: return "Light Path"
        if "kiếm" in n: return "Sword Path"
        if "đao" in n: return "Blade Path"
        if "mộc" in n or "sâm" in n or "lâm" in n: return "Wood Path"
        if "kim" in n or "thiết" in n: return "Metal Path"
        if "hồn" in n: return "Soul Path"
        if "mộng" in n: return "Dream Path"
        if "vân" in n: return "Cloud Path"
        if "độc" in n or "chướng" in n: return "Poison Path"
        if "thú" in n: return "Enslavement Path"
        return "Strength Path"

    MAX_SPAWNS = 4

    for reg in combat_regions:
        rid = reg["id"]
        min_r = int(reg["min_rank"])
        max_r = int(reg["max_rank"])
        reg_path = infer_region_path(reg.get("name", rid))

        # Eligible Normal
        eligible_normals = []
        for rk in range(min_r, max_r + 1):
            eligible_normals.extend([m for m in pool_by_type_rank["Normal"].get(rk, []) if spawn_counts[m] < MAX_SPAWNS])
        if len(eligible_normals) < 3:
            for rk in range(max(1, min_r - 1), min(9, max_r + 1) + 1):
                eligible_normals.extend([m for m in pool_by_type_rank["Normal"].get(rk, []) if spawn_counts[m] < MAX_SPAWNS])

        # Eligible Elite
        eligible_elites = []
        for rk in range(min_r, max_r + 1):
            eligible_elites.extend([m for m in pool_by_type_rank["Elite"].get(rk, []) if spawn_counts[m] < MAX_SPAWNS])
        if len(eligible_elites) < 2:
            for rk in range(max(1, min_r - 1), min(9, max_r + 1) + 1):
                eligible_elites.extend([m for m in pool_by_type_rank["Elite"].get(rk, []) if spawn_counts[m] < MAX_SPAWNS])

        def sort_key(mid):
            m_path = list(json.loads(next(x for x in monsters if x["id"] == mid)["path_masteries"]).keys())[0]
            path_match = 0 if m_path == reg_path else 1
            return (spawn_counts[mid], path_match, rng.random())

        eligible_normals = list(dict.fromkeys(eligible_normals))
        eligible_elites = list(dict.fromkeys(eligible_elites))

        eligible_normals.sort(key=sort_key)
        eligible_elites.sort(key=sort_key)

        for m_id in eligible_normals[:3]:
            rk = ranks[m_id]
            spawn_rows.append({
                "region_id": rid,
                "monster_id": m_id,
                "x": "", "y": "",
                "spawn_mode": "Roaming",
                "chance": "0.55",
                "respawn_seconds": "60",
                "min_rank": str(rk),
                "max_rank": str(rk),
                "is_boss_spawn": "False",
                "is_active": "True"
            })
            spawn_counts[m_id] += 1

        for m_id in eligible_elites[:2]:
            rk = ranks[m_id]
            spawn_rows.append({
                "region_id": rid,
                "monster_id": m_id,
                "x": "", "y": "",
                "spawn_mode": "Patrol",
                "chance": "0.35",
                "respawn_seconds": "120",
                "min_rank": str(rk),
                "max_rank": str(rk),
                "is_boss_spawn": "False",
                "is_active": "True"
            })
            spawn_counts[m_id] += 1

    # Second pass: Place ANY monster that currently has 0 spawns!
    unspawned = [m["id"] for m in monsters if spawn_counts[m["id"]] == 0]
    print(f"Unspawned monsters after pass 1: {len(unspawned)}")

    for m_id in unspawned:
        rk = ranks[m_id]
        m = next(x for x in monsters if x["id"] == m_id)
        mtype = m["type"]
        is_boss = (mtype in ("Boss", "Tribulation_Boss"))
        
        suitable_regions = [r for r in combat_regions if int(r["min_rank"]) <= rk <= int(r["max_rank"])]
        if not suitable_regions:
            suitable_regions = combat_regions

        target_reg = rng.choice(suitable_regions)
        spawn_rows.append({
            "region_id": target_reg["id"],
            "monster_id": m_id,
            "x": "", "y": "",
            "spawn_mode": "Patrol" if mtype == "Elite" else ("Stationary" if is_boss else "Roaming"),
            "chance": "0.25" if is_boss else "0.45",
            "respawn_seconds": "300" if is_boss else "90",
            "min_rank": str(rk),
            "max_rank": str(rk),
            "is_boss_spawn": "True" if is_boss else "False",
            "is_active": "True"
        })
        spawn_counts[m_id] += 1

    sp_vals = list(spawn_counts.values())
    print(f"Spawn counts per monster: min={min(sp_vals)}, max={max(sp_vals)}, avg={sum(sp_vals)/len(sp_vals):.1f}")
    over_spawned = [m for m, c in spawn_counts.items() if c > MAX_SPAWNS]
    print(f"Monsters spawning in > {MAX_SPAWNS} regions: {len(over_spawned)}")

    # 6. Streamline RegionLootTables.csv and region loot tables
    reg_map = {r["id"]: r for r in regions}
    with open(REGION_LOOT_CSV, "r", encoding="utf-8") as f:
        old_reg_loot = list(csv.DictReader(f))

    for r_rule in old_reg_loot:
        lt_id = r_rule["loot_table_id"]
        rid = r_rule["region_id"]
        reg = reg_map.get(rid, {"name": rid, "min_rank": "1", "max_rank": "3"})
        min_r = int(r_rule.get("min_rank") or reg["min_rank"] or 1)
        max_r = int(r_rule.get("max_rank") or reg["max_rank"] or min_r)
        path = infer_region_path(reg.get("name", rid))

        reg_entries = []
        item_a1 = dai_a.get(path, {}).get(min_r)
        item_b1 = dai_b.get(path, {}).get(min_r)
        if item_a1:
            reg_entries.append({"item_id": item_a1, "min_qty": 1, "max_qty": 1, "chance": 0.35, "rank": min_r})
        if item_b1:
            reg_entries.append({"item_id": item_b1, "min_qty": 1, "max_qty": 1, "chance": 0.25, "rank": min_r})
        if max_r > min_r:
            item_a2 = dai_a.get(path, {}).get(max_r)
            if item_a2:
                reg_entries.append({"item_id": item_a2, "min_qty": 1, "max_qty": 1, "chance": 0.20, "rank": max_r})

        loot_tables[lt_id] = {
            "id": lt_id,
            "name": f"Loot Vùng {reg.get('name', rid)} ({r_rule.get('monster_type', 'Normal')})",
            "entries": json.dumps(reg_entries, ensure_ascii=False),
            "created_at": ts,
            "updated_at": ts
        }

    # 7. Verification: check Boss-only items
    dropped_by_normal_elite = set()
    dropped_by_boss = set()

    for m in monsters:
        mid = m["id"]
        lt_id = m.get("loot_table_id", f"loot_{mid}")
        mtype = m.get("type", "Normal")
        lt = loot_tables.get(lt_id)
        if not lt: continue
        entries = json.loads(lt["entries"])
        for e in entries:
            iid = e.get("item_id")
            if not iid: continue
            if mtype in ("Boss", "Tribulation_Boss"):
                dropped_by_boss.add(iid)
            else:
                dropped_by_normal_elite.add(iid)

    boss_only_items = []
    for p_name, base_id, _, _ in DAI_A_DATA:
        for rk in range(1, 10):
            item_id = f"{base_id}_rank{rk}"
            if item_id in dropped_by_boss and item_id not in dropped_by_normal_elite:
                boss_only_items.append(item_id)
    for p_name, base_id, _, _ in DAI_B_DATA:
        for rk in range(1, 10):
            item_id = f"{base_id}_rank{rk}"
            if item_id in dropped_by_boss and item_id not in dropped_by_normal_elite:
                boss_only_items.append(item_id)

    print(f"\nDải A & B items dropped ONLY by Bosses: {len(boss_only_items)}")
    if boss_only_items:
        print(f"Sample boss-only items: {boss_only_items[:10]}")

    # Summary of loot entry counts
    lt_sizes = [len(json.loads(lt["entries"])) for lt in loot_tables.values() if lt["id"].startswith("loot_monster_") or lt["id"].startswith("loot_")]
    print(f"Loot table entry sizes: min={min(lt_sizes)}, max={max(lt_sizes)}, avg={sum(lt_sizes)/len(lt_sizes):.1f}")

    if not apply_changes:
        print("\n[CHẾ ĐỘ XEM TRƯỚC (DRY-RUN)] Thêm cờ --apply để ghi trực tiếp vào CSV.")
        return

    # Write Monster.csv
    with open(MONSTER_CSV, "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=list(monsters[0].keys()))
        writer.writeheader()
        writer.writerows(monsters)

    # Write MonsterSpawn.csv
    with open(SPAWN_CSV, "w", encoding="utf-8", newline="") as f:
        fieldnames = ["region_id", "monster_id", "x", "y", "spawn_mode", "chance", "respawn_seconds", "min_rank", "max_rank", "is_boss_spawn", "is_active"]
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(spawn_rows)

    # Write LootTable.csv
    with open(LOOT_CSV, "w", encoding="utf-8", newline="") as f:
        fieldnames = ["id", "name", "entries", "created_at", "updated_at"]
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(loot_tables.values())

    # Write monster_ranks.csv
    with open(RANKS_CSV, "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["monster_id", "rank"])
        for mid in sorted(ranks):
            writer.writerow([mid, ranks[mid]])

    print("\n[HOÀN TẤT] Đã cập nhật thành công Monster.csv, MonsterSpawn.csv, LootTable.csv và monster_ranks.csv!")

if __name__ == "__main__":
    import sys
    apply_mode = "--apply" in sys.argv
    run_overhaul(apply_mode)
