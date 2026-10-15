#!/usr/bin/env python3
"""Rank-tagged region spawns.

Every monster carries a fixed rank (scripts/monster_ranks.csv, derived from the
original authored spawn data). A region only spawns monsters whose tag rank is
inside the region's [min_rank, max_rank] band, and each spawn row pins
min_rank == max_rank == the tag rank, so the SAME monster always fights at the
SAME difficulty and rolls the SAME monster loot.

Outputs (overwritten in place):

  - Monster.csv    : appends missing high-rank Normal/Elite templates (idempotent)
  - LootTable.csv  : appends a per-monster loot table for each new template
  - MonsterSpawn.csv : Normal + Elite only, rank-tagged, no coordinates
  - MapCell.csv    : keeps valid Resource/SafePoint cells, one rank-tagged Boss
                     per combat region

Run from anywhere: python3 scripts/gen_region_spawns.py [data_dir]
"""
import csv
import io
import json
import os
import random
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA = sys.argv[1] if len(sys.argv) > 1 else os.path.join(REPO, "backend", "data")

REGION = os.path.join(DATA, "Region.csv")
MONSTER = os.path.join(DATA, "Monster.csv")
LOOT = os.path.join(DATA, "LootTable.csv")
SPAWN = os.path.join(DATA, "MonsterSpawn.csv")
MAPCELL = os.path.join(DATA, "MapCell.csv")
RANKS = os.path.join(REPO, "scripts", "monster_ranks.csv")

SEED = 20261015
NORMAL_PER_REGION = 3
ELITE_PER_REGION = 2
NORMAL_MODE, NORMAL_CHANCE, NORMAL_RESPAWN = "Roaming", "0.55", "60"
ELITE_MODE, ELITE_CHANCE, ELITE_RESPAWN = "Patrol", "0.35", "120"
BOSS_RESPAWN = "600"
TS = "2026-10-15 12:00:00"

# Template fields for the high-rank filler monsters the roster was missing:
# (id, name, type, rank, max_hp, base_attack, base_defense, description)
NEW_MONSTERS = [
    ("monster_silvermane_wolf", "Bạc Liêm Lang", "Normal", 7, 700, 70, 35, "Sói bạc lông như sương, săn mồi theo bầy."),
    ("monster_thunder_ape", "Lôi Đình Cự Viên", "Normal", 7, 820, 78, 40, "Viên vượn gọi sấm sét trong cơn thịnh nộ."),
    ("monster_venom_scale_serpent", "Độc Lân Xà", "Normal", 7, 680, 66, 30, "Mãng xà vảy độc, hơi thở ăn mòn xương cốt."),
    ("monster_ancient_mountain_golem", "Thái Cổ Sơn Khôi", "Normal", 8, 1000, 105, 60, "Khôi lỗi đá cổ, thân hình cao như quả núi."),
    ("monster_blood_moon_tiger", "Huyết Nguyệt Hổ", "Normal", 8, 950, 115, 50, "Mãnh hổ săn mồi dưới ánh trăng máu."),
    ("monster_abyss_maw_beast", "Vực Uyên Thú", "Normal", 8, 1080, 100, 62, "Dã thú háu đói trú ngụ nơi vực sâu không đáy."),
    ("monster_primordial_dire_wolf", "Nguyên Sơ Cự Lang", "Normal", 9, 1400, 150, 80, "Lang thú nguyên sơ, tiếng gầm vang khắp hoang nguyên."),
    ("monster_nine_heavens_qilin", "Cửu Tiêu Kỳ Lân", "Normal", 9, 1500, 160, 90, "Kỳ lân linh thú, vảy ánh chư thiên chiếu rọi."),
    ("monster_chaos_devouring_beast", "Thôn Thiên Yêu Thú", "Normal", 9, 1600, 170, 85, "Quái thú tham lam nuốt chửng mọi sinh linh."),
    ("monster_elite_iron_tyrant", "Thiết Giáp Bá Vương", "Elite", 8, 3600, 250, 175, "Bá chủ thiết giáp, da dày không gì xuyên thủng."),
    ("monster_elite_storm_phoenix", "Bạo Phong Phượng Hoàng", "Elite", 8, 3400, 270, 160, "Phượng hoàng bão tố, đôi cánh xé toạc trời cao."),
    ("monster_elite_void_stalker", "Hư Không Tập Kích Giả", "Elite", 8, 3800, 240, 185, "Kẻ ẩn mình trong hư không, chuyên đánh lén."),
    ("monster_elite_heaven_sunderer", "Phá Thiên Tôn Giả", "Elite", 9, 5200, 360, 250, "Tôn giả một đao phá tan tầng mây trời."),
    ("monster_elite_abyssal_overlord", "Vực Sâu Bá Chủ", "Elite", 9, 5500, 340, 270, "Bá chủ vực sâu, nơi ánh sáng không chạm tới."),
    ("monster_elite_celestial_guardian", "Thiên Giới Thủ Hộ", "Elite", 9, 5000, 380, 240, "Hộ vệ thiên giới, canh giữ cõi niết bàn."),
]

SPAWN_HEADER = ["region_id", "monster_id", "x", "y", "spawn_mode", "chance",
                "respawn_seconds", "min_rank", "max_rank", "is_boss_spawn", "is_active"]
LOOT_HEADER = ["id", "name", "entries", "created_at", "updated_at"]


def render(fields):
    buf = io.StringIO()
    csv.writer(buf, lineterminator="\n").writerow(fields)
    return buf.getvalue().rstrip("\n")


def load_ranks():
    ranks = {}
    with open(RANKS, encoding="utf-8-sig") as f:
        for r in csv.DictReader(f):
            ranks[r["monster_id"]] = int(r["rank"])
    for mid, _name, _type, rank, *_ in NEW_MONSTERS:
        ranks[mid] = rank
    return ranks


def ensure_new_monsters(monsters):
    """Append any missing NEW_MONSTERS rows to Monster.csv + LootTable.csv."""
    monster_raw = open(MONSTER, encoding="utf-8").read().splitlines()
    m_header, m_rows = monster_raw[0], [r for r in monster_raw[1:] if r]
    have = {r.split(",", 1)[0] for r in m_rows}

    loot_raw = open(LOOT, encoding="utf-8").read().splitlines()
    l_header, l_rows = loot_raw[0], [r for r in loot_raw[1:] if r]
    loot_have = {r.split(",", 1)[0] for r in l_rows}

    added_m = added_l = 0
    for mid, name, mtype, rank, hp, atk, df, desc in NEW_MONSTERS:
        if mid not in have:
            m_rows.append(render([
                mid, name, mtype, f"{hp}.0", f"{atk}.0", f"{df}.0",
                "[]", "{}", "{}", "{}", "beast_basic", f"loot_{mid}",
                desc, "True", "False", "0.2", "2.0", TS, TS, "[]"]))
            have.add(mid)
            added_m += 1
        lt_id = f"loot_{mid}"
        if lt_id not in loot_have:
            entries = json.dumps([
                {"item_id": f"beast_meat_rank{rank}", "min_qty": 1, "max_qty": 2,
                 "chance": 0.8 if mtype == "Elite" else 0.7, "rank": rank},
                {"item_id": f"heaven_qi_rank{rank}", "min_qty": 1, "max_qty": 2,
                 "chance": 0.6 if mtype == "Elite" else 0.5, "rank": rank},
            ], ensure_ascii=False)
            l_rows.append(render([lt_id, f"Loot {name}", entries, TS, TS]))
            loot_have.add(lt_id)
            added_l += 1

    open(MONSTER, "w", encoding="utf-8").write("\n".join([m_header] + m_rows) + "\n")
    open(LOOT, "w", encoding="utf-8").write("\n".join([l_header] + l_rows) + "\n")
    monsters.update({n[0]: {"id": n[0], "type": n[2]} for n in NEW_MONSTERS})
    return added_m, added_l


def main():
    regions = {r["id"]: r for r in csv.DictReader(open(REGION, encoding="utf-8-sig"))}
    combat = sorted((r for r in regions.values() if r["zone_type"] in ("KHU_VUC", "NGU_VUC")),
                    key=lambda r: r["id"])
    monsters = {r["id"]: r for r in csv.DictReader(open(MONSTER, encoding="utf-8-sig"))}
    added_m, added_l = ensure_new_monsters(monsters)
    ranks = load_ranks()

    pools = {"Normal": {}, "Elite": {}, "Boss": {}}
    for mid, rk in ranks.items():
        m = monsters.get(mid)
        if not m or m["type"] not in pools:
            continue
        pools[m["type"]].setdefault(rk, []).append(mid)
    for t in pools:
        for rk in pools[t]:
            pools[t][rk].sort()

    def eligible(t, lo, hi):
        out = []
        for rk, ids in pools[t].items():
            if lo <= rk <= hi:
                out.extend(ids)
        return sorted(out)

    rng = random.Random(SEED)

    def pick(pool, n):
        if not pool:
            return []
        if len(pool) >= n:
            return rng.sample(pool, n)
        return [rng.choice(pool) for _ in range(n)]

    # --- MonsterSpawn.csv ---
    lines = [",".join(SPAWN_HEADER)]
    per_region = {}
    for reg in combat:
        lo, hi = int(reg["min_rank"]), int(reg["max_rank"])
        npool, epool = eligible("Normal", lo, hi), eligible("Elite", lo, hi)
        per_region[reg["id"]] = (npool, epool)
        for mon in pick(npool, NORMAL_PER_REGION):
            rk = ranks[mon]
            lines.append(render([reg["id"], mon, "", "", NORMAL_MODE, NORMAL_CHANCE,
                                 NORMAL_RESPAWN, str(rk), str(rk), "False", "True"]))
        for mon in pick(epool, ELITE_PER_REGION):
            rk = ranks[mon]
            lines.append(render([reg["id"], mon, "", "", ELITE_MODE, ELITE_CHANCE,
                                 ELITE_RESPAWN, str(rk), str(rk), "False", "True"]))
    open(SPAWN, "w", encoding="utf-8").write("\n".join(lines) + "\n")

    # --- MapCell.csv ---
    raw = open(MAPCELL, encoding="utf-8").read().splitlines()
    header, rows = raw[0], [r for r in raw[1:] if r]
    combat_ids = {r["id"] for r in combat}
    type_of = {m["id"]: m["type"] for m in monsters.values()}

    kept, coord_hint, dropped = [], {}, 0
    for r in rows:
        parts = next(csv.reader([r]))
        if parts[0] not in combat_ids:
            dropped += 1
            continue
        if parts[3].strip().lower() == "boss":
            coord_hint.setdefault(parts[0], (parts[1], parts[2]))
            continue  # bosses are regenerated below
        kept.append(r)

    occupied = {(next(csv.reader([r]))[0], next(csv.reader([r]))[1], next(csv.reader([r]))[2]) for r in kept}
    new_boss = []
    for reg in combat:
        rid = reg["id"]
        pool = eligible("Boss", int(reg["min_rank"]), int(reg["max_rank"])) or eligible("Boss", 3, 9)
        mon = rng.choice(pool)
        rk = ranks[mon]
        x, y = coord_hint.get(rid, (reg["entry_x"], reg["entry_y"]))
        while (rid, x, y) in occupied:
            x = str(int(x) + 1)
        occupied.add((rid, x, y))
        new_boss.append(render([rid, x, y, "Boss", mon, json.dumps({"rank": rk}), BOSS_RESPAWN, "true"]))

    open(MAPCELL, "w", encoding="utf-8").write("\n".join([header] + kept + new_boss) + "\n")

    print(f"new monster templates: +{added_m}  new loot tables: +{added_l}")
    print(f"MonsterSpawn.csv: {len(combat)} regions -> {len(lines)-1} rows "
          f"({NORMAL_PER_REGION} Normal + {ELITE_PER_REGION} Elite each)")
    print(f"MapCell.csv: kept {len(kept)} cells (dropped {dropped} stale), {len(new_boss)} rank-tagged bosses")


if __name__ == "__main__":
    main()
