#!/usr/bin/env python3
"""
scripts/normalize_monster_stats.py

Normalizes monster and boss stats in Monster.csv:
1. Engine ScaleMonsterStats multiplies base stats by RankScale(rank) and typeMultiplier.
2. In Monster.csv, base stats must be rank-independent:
   - Normal: HP ~70-105, ATK ~9-14, DEF ~0.06-0.12
   - Elite:  HP ~120-175, ATK ~14-21, DEF ~0.15-0.30
   - Boss:   HP ~220-325, ATK ~22-32, DEF ~0.30-0.60
3. Resolves the issue where monsters had inflated DEF (e.g. 50-2000 DEF), which when multiplied
   by 10,000 at Rank 6 resulted in 500,000+ DEF, making them completely immune to lower/equal rank Gu!
   With normalized base DEF (0.06 - 0.40):
   - Rank 6 Normal monster has ~800 DEF (Rank 5 Gu dealing 2,000 flat damage can punch through for 1,200 net damage!).
   - Rank 6 Boss has ~12,000 DEF (Rank 6 Gu dealing 200,000 flat damage penetrates cleanly!).
"""

import argparse
import csv
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "backend" / "data"
MONSTER_CSV = DATA_DIR / "Monster.csv"


def get_archetype_multipliers(name, desc):
    t = (name + " " + desc).lower()
    # Tank / Tough / Armor / Shield
    if any(k in t for k in ["quy", "rùa", "golem", "khôi lỗi", "thạch", "nham", "giáp", "tê ngưu", "tượng", "trư", "heo", "kình", "ngạc", "sơn khôi"]):
        return 1.25, 0.85, 1.50
    # Agility / Assassin / Flying / Predator
    elif any(k in t for k in ["bức", "dơi", "xà", "rắn", "điểu", "chim", "ưng", "lang", "sói", "thử", "chuột", "báo", "quạ", "thiền", "ve", "thứ", "chuồn chuồn"]):
        return 0.85, 1.25, 0.70
    # Mystic / Caster / Spirit / Will
    elif any(k in t for k in ["tinh linh", "mộng", "hồ", "cáo", "hoa hồ", "tàn hồn", "tiên phong", "đồng", "thần", "long", "rồng", "phượng"]):
        return 0.95, 1.15, 0.85
    return 1.00, 1.00, 1.00


def run_normalization(dry_run=True):
    print("=== BẮT ĐẦU CHUẨN HÓA & GIẢM CHỈ SỐ / DEFENSE TRONG MONSTER.CSV ===")

    with open(MONSTER_CSV, "r", encoding="utf-8") as f:
        monsters = list(csv.DictReader(f))

    updated = []
    def_reductions = 0
    hp_reductions = 0

    for m in monsters:
        mtype = m.get("type", "Normal")
        hp_mult, atk_mult, def_mult = get_archetype_multipliers(m.get("name", ""), m.get("description", ""))

        if mtype == "Normal":
            base_hp, base_atk, base_def = 85.0, 11.0, 0.08
        elif mtype == "Elite":
            base_hp, base_atk, base_def = 140.0, 17.0, 0.20
        elif mtype in ("Boss", "Tribulation_Boss"):
            base_hp, base_atk, base_def = 260.0, 26.0, 0.40
        elif mtype == "Tribulation_Monster":
            base_hp, base_atk, base_def = 160.0, 16.0, 0.25
        elif mtype == "HumanCultivator":
            base_hp, base_atk, base_def = 100.0, 15.0, 0.10
        else:
            base_hp, base_atk, base_def = 85.0, 11.0, 0.08

        new_hp = round(base_hp * hp_mult, 1)
        new_atk = round(base_atk * atk_mult, 1)
        new_def = round(base_def * def_mult, 2)

        old_hp = float(m["max_hp"])
        old_def = float(m["base_defense"])

        if old_def > new_def:
            def_reductions += 1
        if old_hp > new_hp:
            hp_reductions += 1

        new_m = dict(m)
        new_m["max_hp"] = f"{new_hp:.1f}"
        new_m["base_attack"] = f"{new_atk:.1f}"
        new_m["base_defense"] = f"{new_def:.2f}"
        updated.append(new_m)

    print(f"Tổng số quái vật: {len(monsters)}")
    print(f"Số quái vật được giảm DEF: {def_reductions}/{len(monsters)} ({def_reductions/len(monsters)*100:.1f}%)")
    print(f"Số quái vật được giảm HP lạm phát: {hp_reductions}/{len(monsters)} ({hp_reductions/len(monsters)*100:.1f}%)")

    # Stats distribution
    hp_vals = [float(x["max_hp"]) for x in updated]
    atk_vals = [float(x["base_attack"]) for x in updated]
    def_vals = [float(x["base_defense"]) for x in updated]

    print(f"\nPhân bố Base HP sau chuẩn hóa: Min={min(hp_vals):.1f}, Max={max(hp_vals):.1f}, Median={sorted(hp_vals)[len(hp_vals)//2]:.1f}")
    print(f"Phân bố Base ATK sau chuẩn hóa: Min={min(atk_vals):.1f}, Max={max(atk_vals):.1f}, Median={sorted(atk_vals)[len(atk_vals)//2]:.1f}")
    print(f"Phân bố Base DEF sau chuẩn hóa: Min={min(def_vals):.2f}, Max={max(def_vals):.2f}, Median={sorted(def_vals)[len(def_vals)//2]:.2f}")

    print("\n--- Ví dụ mẫu sau khi chuẩn hóa ---")
    for m in updated[:10]:
        print(f"  • {m['id']:35} ({m['type']:10}): HP={m['max_hp']:>6}, ATK={m['base_attack']:>5}, DEF={m['base_defense']:>5}")

    if not dry_run:
        fieldnames = list(updated[0].keys())
        with open(MONSTER_CSV, "w", encoding="utf-8", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(updated)
        print("\n[HOÀN TẤT] Đã ghi thành công dữ liệu vào Monster.csv!")
    else:
        print("\n[DRY RUN] Đang chạy ở chế độ xem trước. Dùng --apply để áp dụng.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Normalize monster and boss stats in Monster.csv")
    parser.add_argument("--apply", action="store_true", help="Apply changes to Monster.csv")
    args = parser.parse_args()

    run_normalization(dry_run=not args.apply)
