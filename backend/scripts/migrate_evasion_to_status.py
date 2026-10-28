#!/usr/bin/env python3
"""Historical one-time migration: EVASION passive stat -> legacy status data.

This script predates realtime combat. Its emitted turn-based `turns` effect is
not a realtime profile and must not be treated as a currently castable skill.

Background
----------
EVASION used to be an ALWAYS_ON passive stat (STAT_MOD / stat=EVASION) that the
stat pipeline summed into stats["evasion"] -- a value no combatant builder ever
read, so it had no gameplay effect. It is now a self-targeted status in the
statusfx registry, applied by a Gu action exactly like INVINCIBLE:

    {"op": "STATUS", "status_id": "EVASION", "chance": 0.15, "turns": 1, "value": X}

The old simulator applied rank scaling to `chance` and `turns`; `value` was
interpreted as dodge strength in percent. Those turn-based semantics are retired.

ACCURACY is removed outright: it was subtracted as a flat stat (values up to
1000) from a 0..1 dodge chance, so any holder of one Accuracy Gu could never be
evaded. Its roll site and stat key are gone from the simulator.

Usage
-----
    python3 migrate_evasion_to_status.py            # dry run, prints a report
    python3 migrate_evasion_to_status.py --write    # rewrite data/Gu.csv

Run --write only after reviewing the dry-run report. Re-import the "Cổ trùng"
table through the Admin panel afterwards: gu_dictionary rows keep the old JSONB
payload until then, and the registries are cached in memory at startup.
"""

import argparse
import csv
import io
import json
import os
import sys

CSV_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "data", "Gu.csv")

# Base proc chance / legacy turn duration authored on converted rows.
BASE_CHANCE = 0.15
BASE_TURNS = 1

EVASION_BLOCK_ID = "EVASION_SKILL"

# Fields worth rewriting when the description mentions the old passive stat.
DESC_REPLACEMENTS = (
    ("tăng 50% né tránh", "tự buff [Né Tránh] 50%"),
    ("+50% né tránh", "tự buff [Né Tránh] 50%"),
    ("tăng khả năng né tránh và", "tăng"),
)


def dodge_percent(old_value: float) -> float:
    """Convert a legacy ALWAYS_ON EVASION ADD_FLAT value into a dodge percent.

    The authored data mixed two units: 10 rows wrote a 0..1 ratio (0.1, 0.15)
    while 54 rows wrote flat percent points (2..40). Values <= 1 are treated as
    ratios; anything larger is already a percent.
    """
    return round(old_value * 100, 4) if old_value <= 1 else round(old_value, 4)


def evasion_block(value_pct: float) -> dict:
    return {
        "id": EVASION_BLOCK_ID,
        "target": {"selector": "SELF"},
        "trigger": "ON_ACTION",
        "operations": [
            {
                "op": "STATUS",
                "status_id": "EVASION",
                "chance": BASE_CHANCE,
                "turns": BASE_TURNS,
                "value": value_pct,
            }
        ],
    }


def is_evasion_stat(op: dict) -> bool:
    return op.get("op") == "STAT_MOD" and str(op.get("stat", "")).upper() == "EVASION"


def is_accuracy_stat(op: dict) -> bool:
    return op.get("op") == "STAT_MOD" and str(op.get("stat", "")).upper() == "ACCURACY"


def is_active_block(block: dict) -> bool:
    return block.get("trigger") == "ON_ACTION" or block.get("id") == "ACTIVE_SKILL"


def fix_description(desc: str) -> str:
    for old, new in DESC_REPLACEMENTS:
        desc = desc.replace(old, new)
    return desc


def migrate_effects(effects: list) -> tuple[list, dict]:
    """Strip EVASION/ACCURACY stats and append the EVASION status block."""
    report = {
        "evasion_value": None,
        "accuracy_removed": 0,
        "evasion_blocks_dropped": 0,
        "accuracy_blocks_dropped": 0,
        "appended_block": False,
    }

    kept: list = []
    for block in effects:
        ops = block.get("operations")
        if not isinstance(ops, list):
            kept.append(block)
            continue

        remaining = []
        for op in ops:
            if is_accuracy_stat(op):
                report["accuracy_removed"] += 1
                continue
            if is_evasion_stat(op) and block.get("trigger") == "ALWAYS_ON":
                report["evasion_value"] = op.get("value")
                continue
            remaining.append(op)

        if len(remaining) == len(ops):
            kept.append(block)
            continue

        if not remaining:
            # Whole block carried only removed stats -- drop it.
            if any(is_evasion_stat(o) for o in ops):
                report["evasion_blocks_dropped"] += 1
            else:
                report["accuracy_blocks_dropped"] += 1
            continue

        new_block = dict(block)
        new_block["operations"] = remaining
        kept.append(new_block)

    if report["evasion_value"] is not None:
        pct = dodge_percent(float(report["evasion_value"]))
        already = any(
            b.get("id") == EVASION_BLOCK_ID
            and any(o.get("status_id") == "EVASION" for o in b.get("operations", []))
            for b in kept
        )
        if not already:
            kept.append(evasion_block(pct))
            report["appended_block"] = True
        report["evasion_value"] = pct

    return kept, report


def split_raw_fields(line: str) -> list[str]:
    """Split one CSV line into raw field texts, keeping each field's quoting.

    Gu.csv quotes some numeric fields and not others, so re-serializing a whole
    row would rewrite untouched columns. Working on raw field text instead lets
    the script substitute only the columns it actually changes.
    """
    fields: list[str] = []
    buf: list[str] = []
    in_quotes = False
    i = 0
    while i < len(line):
        ch = line[i]
        if in_quotes:
            if ch == '"':
                if i + 1 < len(line) and line[i + 1] == '"':
                    buf.append('""')  # escaped quote, stays inside the field
                    i += 2
                    continue
                buf.append('"')  # closing quote, kept for round-tripping
                in_quotes = False
                i += 1
                continue
            buf.append(ch)
            i += 1
            continue
        if ch == '"':
            in_quotes = True
            buf.append(ch)  # opening quote, kept so the field round-trips
            i += 1
            continue
        if ch == ",":
            fields.append("".join(buf))
            buf = []
            i += 1
            continue
        buf.append(ch)
        i += 1
    fields.append("".join(buf))
    return fields


def quote_field(field: str) -> str:
    """Quote a field the way csv does when it must be quoted."""
    if field == "":
        return '""'
    if any(c in field for c in ',"\n\r'):
        return '"' + field.replace('"', '""') + '"'
    return field


def unquote_field(raw: str) -> str:
    if len(raw) >= 2 and raw.startswith('"') and raw.endswith('"'):
        return raw[1:-1].replace('""', '"')
    return raw


def format_row(row: list[str]) -> str:
    """Serialize a data row from raw (still-quoted) field texts."""
    return ",".join(row)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="write changes back to data/Gu.csv")
    parser.add_argument("--csv", default=CSV_PATH, help="path to Gu.csv")
    args = parser.parse_args()

    # Gu.csv is not consistently quoted (some numeric columns are quoted, some
    # are not), so lines are handled as raw text: each row keeps its original
    # field text and only the columns this script changes get re-serialized.
    with open(args.csv, encoding="utf-8", newline="") as fh:
        lines = fh.read().split("\n")
    if lines and lines[-1] == "":
        lines.pop()

    header_line = lines[0]
    header = [unquote_field(f) for f in split_raw_fields(header_line)]
    idx = {name: i for i, name in enumerate(header)}
    for required in ("id", "rank", "effects", "description"):
        if required not in idx:
            print(f"error: column {required!r} missing from {args.csv}", file=sys.stderr)
            return 1

    out_lines: list[str] = [header_line]
    evas_rows = 0
    accuracy_rows = 0
    gained_block = 0
    desc_changed = 0
    total_accuracy_ops = 0
    total_evasion_ops = 0
    emptied = []

    for line in lines[1:]:
        fields = split_raw_fields(line)
        raw_effects = unquote_field(fields[idx["effects"]])
        if "EVASION" not in raw_effects and "ACCURACY" not in raw_effects:
            out_lines.append(line)
            continue

        try:
            effects = json.loads(raw_effects)
        except json.JSONDecodeError as exc:
            print(f"error: {unquote_field(fields[idx['id']])} has invalid effects JSON: {exc}", file=sys.stderr)
            return 1
        if not isinstance(effects, list):
            out_lines.append(line)
            continue

        gu_id = unquote_field(fields[idx["id"]])
        new_effects, report = migrate_effects(effects)
        total_accuracy_ops += report["accuracy_removed"]
        if report["evasion_value"] is not None:
            evas_rows += 1
            total_evasion_ops += 1
        if report["accuracy_removed"]:
            accuracy_rows += 1
        if report["appended_block"]:
            gained_block += 1
        if not new_effects:
            emptied.append(gu_id)

        fields[idx["effects"]] = quote_field(
            json.dumps(new_effects, ensure_ascii=False, separators=(",", ":"))
        )
        raw_desc = unquote_field(fields[idx["description"]])
        new_desc = fix_description(raw_desc)
        if new_desc != raw_desc:
            fields[idx["description"]] = quote_field(new_desc)
            desc_changed += 1
        out_lines.append(format_row(fields))

    print("=== migration report (EVASION passive -> EVASION status, ACCURACY removed) ===")
    print(f"  rows total                 : {len(lines) - 1}")
    print(f"  rows with EVASION passive  : {evas_rows} (ops rewritten: {total_evasion_ops})")
    print(f"  rows with ACCURACY passive : {accuracy_rows} (ops removed: {total_accuracy_ops})")
    print(f"  EVASION_SKILL blocks added : {gained_block}")
    print(f"  descriptions rewritten     : {desc_changed}")
    if emptied:
        print(f"  rows left with empty effects: {len(set(emptied))}")
        for gid in sorted(set(emptied)):
            print(f"    - {gid}  <-- REVIEW: Gu has no effects left")

    if not args.write:
        print("\n(dry run -- pass --write to apply)")
        return 0

    with open(args.csv, "w", encoding="utf-8", newline="") as fh:
        fh.write("\n".join(out_lines) + "\n")
    print(f"\nwritten: {args.csv}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
