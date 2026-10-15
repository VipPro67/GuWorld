#!/usr/bin/env python3
"""Generate the SQL refund for the 2026-10-15 PVE death-penalty losses.

Scope = the 13 combat_history rows exported to Fix.txt (7 players). For every
row the BATTLE_END log carries the full death-drop summary produced by
applyDeathDrop (death_penalty.go), so the loss list is recoverable from the
export itself. Stones/items/recipes are exact; the only lossy part is Cổ trùng
because gu names are not unique across ranks (Gu.csv), so:

  1. unique name           -> use that (gu_id, rank)
  2. ambiguous name that IS in a ground drop (map_ground_drops.loot_data.Gus,
     exported to ground_drops_2026-10-15.tsv) -> use that rank
  3. otherwise             -> lowest rank (review list printed)

Item names that collide in Item.csv are resolved against DB evidence and must
be listed in ITEM_NAME_OVERRIDES, else generation aborts.

Usage:
  python3 refund_death_drop_2026-10-15.py --selftest
  python3 refund_death_drop_2026-10-15.py --out refund_death_drop_2026-10-15.sql
"""

import argparse
import csv
import json
import re
import sys
import uuid
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[1]
FIX_TXT = REPO_ROOT / "Fix.txt"
ITEM_CSV = REPO_ROOT / "backend" / "data" / "Item.csv"
GU_CSV = REPO_ROOT / "backend" / "data" / "Gu.csv"
GROUND_TSV = SCRIPT_DIR / "ground_drops_2026-10-15.tsv"

# Item.csv has duplicate display names. Only the ones that actually appear in
# this batch need an explicit id; evidence: both affected players still hold
# HEART_ESSENCE_RANK3 in player_inventory_items and never HEART_STAR_ESSENCE.
ITEM_NAME_OVERRIDES = {
    "Tâm Tinh": "HEART_ESSENCE_RANK3",
}

ITEM_TYPE_CATEGORY = {
    "food": "consumables",
    "consumable": "consumables",
    "material": "materials",
    "currency": "materials",
    "equipment": "equipment",
    "quest": "quest",
}

LOSS_RE = re.compile(
    r"rơi rớt mất: (.*?)(?: \[Oán Khí[^\]]*\]|\. Hồi sinh tại.*)$", re.S
)


def load_csv(path):
    with open(path, encoding="utf-8-sig", newline="") as fh:
        return list(csv.DictReader(fh))


def split_entries(seg):
    """Split the loss summary on top-level ', ' (names never contain a comma)."""
    parts, depth, cur = [], 0, ""
    for ch in seg:
        if ch == "(":
            depth += 1
        if ch == ")":
            depth -= 1
        if ch == "," and depth == 0:
            parts.append(cur.strip())
            cur = ""
        else:
            cur += ch
    parts.append(cur.strip())
    return [p for p in parts if p]


def parse_loss(detail):
    """Return (primeval, immortal, {item_name: qty}, [recipe_id], [gu_name])."""
    match = LOSS_RE.search(detail)
    if not match:
        raise SystemExit("cannot parse loss summary: " + detail[:120])
    primeval = immortal = 0.0
    items = Counter()
    recipes = []
    gus = []
    for entry in split_entries(match.group(1)):
        head = entry[: entry.rindex(" (")] if entry.endswith(")") else entry
        if head.endswith("Tiên Nguyên Thạch"):
            immortal += float(head.split(" ")[0])
        elif head.endswith("Nguyên Thạch"):
            primeval += float(head.split(" ")[0])
        elif head.startswith("Cổ trùng "):
            gus.append(head[len("Cổ trùng "):].strip())
        elif head.startswith("Bí phương "):
            recipes.append(head[len("Bí phương "):].strip())
        else:
            m = re.match(r"^(\d+)x (.*)$", head)
            if not m:
                raise SystemExit("cannot parse loss entry: " + repr(entry))
            items[m.group(2)] += int(m.group(1))
    return primeval, immortal, items, recipes, gus


def parse_fix():
    """Yield dicts with the parsed loss of every exported combat row."""
    rows = []
    for line in FIX_TXT.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        cols = line.split("\t")
        if len(cols) != 10:
            raise SystemExit("unexpected Fix.txt column count: %d" % len(cols))
        combat_id, player_id, winner, logs = cols[0], cols[1], cols[2], cols[3]
        details = [
            log["detail"]
            for log in json.loads(logs)
            if log.get("action") == "BATTLE_END"
        ]
        if not details:
            raise SystemExit("row %s has no BATTLE_END log" % combat_id)
        primeval, immortal, items, recipes, gus = parse_loss(details[-1])
        rows.append(
            {
                "combat_id": combat_id,
                "player_id": player_id,
                "winner": winner,
                "primeval": primeval,
                "immortal": immortal,
                "items": items,
                "recipes": recipes,
                "gus": gus,
            }
        )
    return rows


def ground_evidence():
    """Return (item_id -> category, {(player_id, gu_id, rank)} from ground drops)."""
    cat = {}
    gu = set()
    if not GROUND_TSV.exists():
        return cat, gu
    for line in GROUND_TSV.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        pid, _ts, loot = line.rstrip("\n").split("\t")
        data = json.loads(loot)
        for it in data.get("items", []):
            cat[it["item_id"]] = it["category"]
        for g in data.get("gus", []):
            gu.add((pid, g["gu_id"], int(g["rank"])))
    return cat, gu


def resolve(items_csv, gu_csv, gu_rows, cat_evidence):
    by_item_name = defaultdict(list)
    for it in items_csv:
        by_item_name[it["name"].strip()].append(it)
    item_by_id = {it["id"]: it for it in items_csv}

    gu_by_name = defaultdict(list)
    gu_name_by_idrank = {}
    for g in gu_csv:
        key = (g["id"], int(g["rank"]))
        gu_name_by_idrank[key] = g["name"].strip()
        gu_by_name[g["name"].strip()].append(key)

    # ground evidence keyed by (player, gu_name) -> set of (gu_id, rank)
    gu_ground = defaultdict(set)
    for (pid, gid, rank) in gu_rows:
        gu_ground[(pid, gu_name_by_idrank.get((gid, rank), ""))].add((gid, rank))

    def item_id(name, player_id):
        cands = by_item_name.get(name, [])
        if len(cands) == 1:
            return cands[0]["id"]
        if len(cands) > 1:
            if name in ITEM_NAME_OVERRIDES:
                return ITEM_NAME_OVERRIDES[name]
            raise SystemExit(
                "ambiguous item name %r (%s) for %s; add an override"
                % (name, [c["id"] for c in cands], player_id)
            )
        raise SystemExit("unknown item name %r" % name)

    def item_category(item_id_, name):
        if item_id_ in cat_evidence:
            return cat_evidence[item_id_]
        spec = item_by_id[item_id_]
        return ITEM_TYPE_CATEGORY.get(spec["item_type"].strip().lower(), "misc")

    def gu_idrank(name, player_id):
        cands = gu_by_name.get(name, [])
        if not cands:
            raise SystemExit("unknown gu name %r" % name)
        if len(cands) == 1:
            return cands[0], False
        grounded = gu_ground.get((player_id, name), set())
        if len(grounded) == 1:
            return next(iter(grounded)), False
        low = min(r for _, r in cands)
        same = [c for c in cands if c[1] == low]
        if len(same) != 1:
            raise SystemExit("ambiguous gu name %r at min rank" % name)
        return same[0], True

    return item_id, item_category, gu_idrank


def build():
    items_csv = load_csv(ITEM_CSV)
    gu_csv = load_csv(GU_CSV)
    cat_evidence, gu_evidence = ground_evidence()
    item_id, item_category, gu_idrank = resolve(items_csv, gu_csv, gu_evidence, cat_evidence)

    per_player = {}
    ambiguous_gus = []
    for row in parse_fix():
        pid = row["player_id"]
        acc = per_player.setdefault(
            pid,
            {
                "primeval": 0.0,
                "immortal": 0.0,
                "items": Counter(),
                "categories": {},
                "recipes": set(),
                "gus": [],
                "combats": [],
            },
        )
        acc["primeval"] += row["primeval"]
        acc["immortal"] += row["immortal"]
        acc["combats"].append(row["combat_id"])
        for name, qty in row["items"].items():
            iid = item_id(name, pid)
            acc["items"][iid] += qty
            acc["categories"][iid] = item_category(iid, name)
        acc["recipes"].update(row["recipes"])
        for name in row["gus"]:
            gid_rank, fallback = gu_idrank(name, pid)
            acc["gus"].append(gid_rank)
            if fallback:
                ambiguous_gus.append((pid, name, gid_rank))
    return per_player, ambiguous_gus


def sql_str(value):
    return "'" + str(value).replace("'", "''") + "'"


def generate(per_player, ambiguous_gus):
    out = []
    out.append("-- Auto-generated by scripts/refund_death_drop_2026-10-15.py -- do not edit.")
    out.append("-- Refund full death-penalty losses (rơi tại chỗ + chấn nát vĩnh viễn)")
    out.append("-- for the 13 PVE defeats exported to Fix.txt on 2026-10-15.")
    out.append("-- Generated: " + datetime.now(timezone.utc).isoformat())
    out.append("")
    out.append("-- ===== SECTION 1: PREVIEW (read-only) =====")
    out.append("-- Expected per-player refund:")
    for pid, acc in sorted(per_player.items(), key=lambda kv: -kv[1]["primeval"]):
        out.append(
            "--   %s: +%.3f Nguyên Thạch, +%.3f Tiên Nguyên Thạch, "
            "%d item rows, %d recipes, %d Cổ (%d combats)"
            % (
                pid,
                acc["primeval"],
                acc["immortal"],
                len(acc["items"]),
                len(acc["recipes"]),
                len(acc["gus"]),
                len(acc["combats"]),
            )
        )
    out.append("")
    out.append("SELECT id, username, primeval_stones, immortal_stones FROM players WHERE id IN (")
    out.append("  " + ",\n  ".join(sql_str(p) for p in per_player))
    out.append(");")
    if ambiguous_gus:
        out.append("")
        out.append("-- Cổ trùng rank không xác định được từ ground drop, đã dùng rank thấp nhất:")
        for pid, name, idrank in ambiguous_gus:
            out.append("--   %s %s -> %s" % (pid, name, idrank))
    out.append("")
    out.append("-- ===== SECTION 2: APPLY (transactional, idempotent) =====")
    out.append("BEGIN;")
    out.append("")
    out.append("DO $$")
    out.append("BEGIN")
    out.append("  IF EXISTS (SELECT 1 FROM audit_logs WHERE action = 'combat.refund_death_drop'")
    out.append("               AND entity_type = 'bulk_refund' AND entity_id = '2026-10-15') THEN")
    out.append("    RAISE EXCEPTION 'refund_death_drop 2026-10-15 already applied';")
    out.append("  END IF;")
    out.append("END $$;")
    out.append("")
    for pid, acc in per_player.items():
        out.append("-- %s (%d combats)" % (pid, len(acc["combats"])))
        out.append(
            "UPDATE players SET primeval_stones = primeval_stones + %.3f,"
            " immortal_stones = immortal_stones + %.3f WHERE id = %s;"
            % (acc["primeval"], acc["immortal"], sql_str(pid))
        )
        if acc["items"]:
            values = ",\n    ".join(
                "(%s, %s, %s)"
                % (sql_str(iid), str(qty), sql_str(acc["categories"][iid]))
                for iid, qty in sorted(acc["items"].items())
            )
            out.append(
                "INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)\n"
                "SELECT %s, v.item_id,\n"
                "       COALESCE((SELECT category FROM player_inventory_items"
                " WHERE player_id = %s AND item_id = v.item_id LIMIT 1), v.category),\n"
                "       v.quantity, NOW()\n"
                "FROM (VALUES\n    %s\n) AS v(item_id, quantity, category)\n"
                "ON CONFLICT (player_id, item_id, category) DO UPDATE"
                " SET quantity = player_inventory_items.quantity + EXCLUDED.quantity,"
                " updated_at = NOW();"
                % (sql_str(pid), sql_str(pid), values)
            )
        if acc["recipes"]:
            values = ",\n    ".join(
                "(%s, %s, 100, true)" % (sql_str(pid), sql_str(rid))
                for rid in sorted(acc["recipes"])
            )
            out.append(
                "INSERT INTO player_recipe_progress (player_id, recipe_id, completion_rate, is_unlocked)\n"
                "VALUES\n    %s\n"
                "ON CONFLICT (player_id, recipe_id) DO UPDATE"
                " SET is_unlocked = true, completion_rate = 100, updated_at = NOW();" % values
            )
        if acc["gus"]:
            values = ",\n    ".join(
                "(%s, %s, %s, %d, %d, 1, 100, 'ACTIVE', 'INVENTORY', false, false, NOW(), NOW(), NOW())"
                % ("'gu-" + str(uuid.uuid4()) + "'", sql_str(pid), sql_str(gid), rank, rank)
                for gid, rank in acc["gus"]
            )
            out.append(
                "INSERT INTO player_gu (id, player_id, gu_id, gu_rank, rank, level, hunger, state,"
                " location, equipped, is_vital, created_at, updated_at, last_fed_at)\n"
                "VALUES\n    %s;" % values
            )
        out.append("")
    out.append(
        "INSERT INTO audit_logs (action, entity_type, entity_id, request_summary, metadata, success)\n"
        "VALUES ('combat.refund_death_drop', 'bulk_refund', '2026-10-15',"
        " 'Death-penalty refund from Fix.txt export',"
        " %s, true);"
        % sql_str(json.dumps({
            "players": {p: {
                "primeval": round(a["primeval"], 3),
                "immortal": round(a["immortal"], 3),
                "items": len(a["items"]),
                "recipes": len(a["recipes"]),
                "gus": len(a["gus"]),
            } for p, a in per_player.items()},
            "source": "Fix.txt",
        }))
    )
    out.append("COMMIT;")
    out.append("")
    out.append("-- ===== SECTION 3: VERIFY =====")
    out.append("-- Audit marker (must be exactly 1 row):")
    out.append("-- SELECT * FROM audit_logs WHERE action='combat.refund_death_drop';")
    return "\n".join(out) + "\n"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out")
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()

    per_player, ambiguous_gus = build()

    if args.selftest:
        primeval = sum(a["primeval"] for a in per_player.values())
        immortal = sum(a["immortal"] for a in per_player.values())
        item_rows = sum(len(a["items"]) for a in per_player.values())
        recipes = sum(len(a["recipes"]) for a in per_player.values())
        gus = sum(len(a["gus"]) for a in per_player.values())
        assert len(per_player) == 7, len(per_player)
        assert abs(primeval - 147200214.897) < 0.001, primeval
        assert abs(immortal - 1590686.525) < 0.001, immortal
        assert item_rows == 715, item_rows
        assert recipes == 1646, recipes
        assert gus == 30, gus
        assert len(ambiguous_gus) == 5, ambiguous_gus
        print(
            "selftest ok: 7 players, %.3f primeval, %.3f immortal, %d item rows,"
            " %d recipes, %d Cổ, %d ambiguous"
            % (primeval, immortal, item_rows, recipes, gus, len(ambiguous_gus))
        )
        return

    sql = generate(per_player, ambiguous_gus)
    if args.out:
        Path(args.out).write_text(sql, encoding="utf-8")
        print("wrote " + args.out)
    else:
        sys.stdout.write(sql)


if __name__ == "__main__":
    main()
