#!/usr/bin/env python3
"""Scale item/Gu prices and recipe fees in backend/data CSVs."""

import argparse
import csv
import os
import sys
import tempfile
from decimal import Decimal, InvalidOperation, ROUND_HALF_UP
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend" / "data"
FIELDS = {
    "Item.csv": {"base_price": "item_prices"},
    "Gu.csv": {"base_price": "gu_prices"},
    "Recipe.csv": {
        "primeval_stones_cost": "recipe-primeval",
        "immortal_essence_stones_cost": "recipe-immortal",
    },
}
ITEM_PRICE_BY_RANK = (0, 1, 3, 6, 13, 25, 10, 25, 60, 150)
GU_PRICE_BY_RANK = (0, 200, 500, 1200, 2500, 5000, 200, 5000, 120000, 250000)


def csv_field_spans(text):
    """Return CSV field spans while preserving the source's original formatting."""
    rows, row, pos = [], [], 0
    while pos < len(text):
        start = pos
        if text[pos] == '"':
            pos += 1
            while pos < len(text):
                if text[pos] == '"':
                    if pos + 1 < len(text) and text[pos + 1] == '"':
                        pos += 2
                        continue
                    pos += 1
                    break
                pos += 1
            end = pos
            value = next(csv.reader([text[start:end]]))[0]
        else:
            while pos < len(text) and text[pos] not in ",\r\n":
                pos += 1
            end = pos
            value = text[start:end]
        row.append((value, start, end))

        if pos < len(text) and text[pos] == ",":
            pos += 1
            continue
        rows.append(row)
        row = []
        if pos < len(text) and text[pos] == "\r":
            pos += 1
        if pos < len(text) and text[pos] == "\n":
            pos += 1
    if row:
        rows.append(row)
    return rows


def multiplier(value, label):
    if value is None:
        return Decimal(1)
    try:
        result = Decimal(value)
    except InvalidOperation as exc:
        raise ValueError(f"Hệ số {label} không hợp lệ: {value}") from exc
    if not result.is_finite() or result <= 0:
        raise ValueError(f"Hệ số {label} phải lớn hơn 0")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--recipe-primeval", help="hệ số phí Nguyên Thạch recipe")
    parser.add_argument("--recipe-immortal", help="hệ số phí Tiên Nguyên Thạch recipe")
    parser.add_argument("--apply", action="store_true", help="ghi thay đổi vào CSV (mặc định chỉ xem trước)")
    args = parser.parse_args()

    factors = {
        "recipe-primeval": multiplier(args.recipe_primeval, "recipe-primeval"),
        "recipe-immortal": multiplier(args.recipe_immortal, "recipe-immortal"),
    }

    edits = {}
    summaries = []
    for filename, columns in FIELDS.items():
        path = DATA / filename
        raw = path.read_bytes()
        bom = b"\xef\xbb\xbf" if raw.startswith(b"\xef\xbb\xbf") else b""
        text = raw[len(bom):].decode("utf-8")
        rows = csv_field_spans(text)
        headers = [field[0] for field in rows[0]]
        changes = []
        rank_counts = {}
        column_counts = {}
        for column, factor_name in columns.items():
            if factor_name.startswith("recipe-") and getattr(args, factor_name.replace("-", "_")) is None:
                continue
            if column not in headers:
                raise ValueError(f"{filename} thiếu cột {column}")
            index = headers.index(column)
            rank_index = headers.index("rank") if filename != "Recipe.csv" else headers.index("target_rank")
            for line, row in enumerate(rows[1:], start=2):
                if index >= len(row) or rank_index >= len(row):
                    continue
                old, start, end = row[index]
                try:
                    rank = int(row[rank_index][0] or "1")
                except ValueError as exc:
                    raise ValueError(f"{filename}:{line} rank không hợp lệ") from exc
                rank = rank if 1 <= rank <= 9 else 1
                if factor_name in ("item_prices", "gu_prices"):
                    rank_counts.setdefault(rank, [0, 0])
                    rank_counts[rank][0] += 1
                if factor_name == "item_prices":
                    new = Decimal(ITEM_PRICE_BY_RANK[rank])
                elif factor_name == "gu_prices":
                    new = Decimal(GU_PRICE_BY_RANK[rank])
                else:
                    if not old.strip():
                        continue
                    try:
                        amount = Decimal(old.strip())
                    except InvalidOperation as exc:
                        raise ValueError(f"{filename}:{line} giá trị {column} không hợp lệ: {old}") from exc
                    if not amount.is_finite() or amount < 0:
                        raise ValueError(f"{filename}:{line} giá trị {column} phải là số không âm")
                    new = (amount * factors[factor_name]).quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)
                if old.strip() != f"{new:.3f}":
                    changes.append((start, end, f"{new:.3f}"))
                    column_counts[column] = column_counts.get(column, 0) + 1
                    if factor_name in ("item_prices", "gu_prices"):
                        rank_counts[rank][1] += 1
        if changes:
            edits[path] = (bom, text, changes)
            if rank_counts:
                detail = "; ".join(
                    f"R{rank}: {changed}/{total} → {ITEM_PRICE_BY_RANK[rank] if filename == 'Item.csv' else GU_PRICE_BY_RANK[rank]}"
                    for rank, (total, changed) in sorted(rank_counts.items())
                )
                summaries.append(f"{filename}: {len(changes)} giá sẽ được đồng bộ theo rank ({detail})")
            else:
                detail = ", ".join(f"{column}: {count}" for column, count in column_counts.items())
                summaries.append(f"{filename}: {len(changes)} phí sẽ đổi ({detail})")

    if not summaries:
        print("Không có giá trị nào cần thay đổi.")
        return
    print("\n".join(summaries))
    if not args.apply:
        print("Chỉ xem trước. Thêm --apply để ghi CSV.")
        return

    for path, (bom, text, changes) in edits.items():
        for start, end, value in sorted(changes, reverse=True):
            text = text[:start] + value + text[end:]
        fd, temp_path = tempfile.mkstemp(dir=path.parent, prefix=f".{path.name}.")
        try:
            with os.fdopen(fd, "wb") as out:
                out.write(bom + text.encode("utf-8"))
            os.chmod(temp_path, path.stat().st_mode)
            os.replace(temp_path, path)
        finally:
            if os.path.exists(temp_path):
                os.unlink(temp_path)
    print("Đã ghi CSV. Import từng dataset trong Admin để áp dụng vào DB.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"Lỗi: {exc}", file=sys.stderr)
        sys.exit(1)
