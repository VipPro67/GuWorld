#!/usr/bin/env python3
"""
scripts/adjust_game_prices.py

Scale and balance item/Gu base prices and recipe fees in backend/data CSVs.
Adheres strictly to the canonical pricing tables defined in:
  - docs/GuRankScaling.md
  - backend/pkg/pricing/pricing.go
  - frontend/src/components/CaravanShop.tsx & SectView.tsx

Currency tiers:
  - Rank 1–5 (Phàm Cổ / Phàm Vật): Nguyên Thạch (Primeval Stones)
  - Rank 6–9 (Tiên Cổ / Tiên Tài): Tiên Nguyên Thạch (Immortal Essence Stones)
"""

import argparse
import csv
import json
import os
import sys
import tempfile
from decimal import Decimal, InvalidOperation, ROUND_HALF_UP
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend" / "data"

# Canonical base price tables from docs/specs/system_pricing_economy_spec.md and backend/pkg/pricing/pricing.go
# Index 0 unused; index i = rank i.
# R1-5 in Nguyên Thạch; R6-9 in Tiên Nguyên Thạch.
CANONICAL_GU_PRICES = (0, 250, 1000, 4000, 22000, 250000, 3500, 35000, 550000, 14000000)

# Canonical item prices (Master Spec balanced curve)
CANONICAL_ITEM_PRICES = (0, 2, 8, 35, 200, 2000, 5, 50, 750, 15000)

# Legacy Material item prices
MATERIAL_ITEM_PRICES = (0, 2, 8, 35, 200, 2000, 5, 50, 750, 15000)

PROTECTED_ITEMS = {
    "arena_coin",
    "mine_chest",
    "qi_gathering_pill_rank1",
    "qi_gathering_pill_rank2",
    "qi_gathering_pill_rank3",
    "qi_gathering_pill_rank4",
    "qi_gathering_pill_rank5",
    "healing_pill_rank1",
    "healing_pill_rank2",
    "healing_pill_rank3",
    "healing_pill_rank4",
    "healing_pill_rank5",
    "blood_clot_pill_rank1",
    "spirit_cleansing_pill_rank2",
}


def parse_curve(curve_str):
    """Parse a comma-separated list of 9 prices (rank 1 to 9)."""
    parts = [p.strip() for p in curve_str.split(",") if p.strip()]
    if len(parts) != 9:
        raise ValueError(f"Bảng giá phải có đúng 9 phần tử cho Rank 1-9 (nhận được {len(parts)})")
    vals = [0] + [float(p) for p in parts]
    return tuple(vals)


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
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        "--target",
        choices=["all", "gu", "item", "recipe"],
        default="all",
        help="chỉ định đối tượng cần cân bằng: all, gu, item, hoặc recipe (mặc định all)",
    )
    parser.add_argument(
        "--item-mode",
        choices=["canonical", "material"],
        default="material",
        help="chế độ giá item: 'material' (R1-5: 100-5000 NT, R6-9: 10-150 TNT) hoặc 'canonical' (R1-5: 1-25 NT, R6-9: 10-150 TNT)",
    )
    parser.add_argument(
        "--gu-curve",
        help="tuỳ chỉnh bảng giá Cổ 9 rank (vd: '200,500,1200,2500,5000,200,500,1200,2500')",
    )
    parser.add_argument(
        "--item-curve",
        help="tuỳ chỉnh bảng giá Item 9 rank (vd: '100,300,800,2000,5000,10,25,60,150')",
    )
    parser.add_argument(
        "--rarity-factor",
        type=float,
        default=0.0,
        help="hệ số tăng giá theo rarity cùng rank (vd: 0.1 nghĩa là +10%% mỗi điểm rarity, mặc định 0.0)",
    )
    parser.add_argument(
        "--recipe-primeval",
        help="hệ số nhân phí Nguyên Thạch cho Recipe",
    )
    parser.add_argument(
        "--recipe-immortal",
        help="hệ số nhân phí Tiên Nguyên Thạch cho Recipe",
    )
    parser.add_argument(
        "--include-protected-items",
        action="store_true",
        help="cho phép ghi đè cả đan dược và vật phẩm tiền tệ đặc biệt",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="ghi thay đổi vào CSV (mặc định chỉ xem trước / dry-run)",
    )
    args = parser.parse_args()

    # Determine curves
    gu_curve = parse_curve(args.gu_curve) if args.gu_curve else CANONICAL_GU_PRICES
    if args.item_curve:
        item_curve = parse_curve(args.item_curve)
    elif args.item_mode == "canonical":
        item_curve = CANONICAL_ITEM_PRICES
    else:
        item_curve = MATERIAL_ITEM_PRICES

    factors = {
        "recipe-primeval": multiplier(args.recipe_primeval, "recipe-primeval"),
        "recipe-immortal": multiplier(args.recipe_immortal, "recipe-immortal"),
    }

    fields_to_process = {}
    if args.target in ("all", "item"):
        fields_to_process["Item.csv"] = {"base_price": "item_prices"}
    if args.target in ("all", "gu"):
        fields_to_process["Gu.csv"] = {"base_price": "gu_prices"}
    if args.target in ("all", "recipe"):
        fields_to_process["Recipe.csv"] = {
            "primeval_stones_cost": "recipe-primeval",
            "immortal_essence_stones_cost": "recipe-immortal",
        }

    edits = {}
    summaries = []

    for filename, columns in fields_to_process.items():
        path = DATA / filename
        if not path.exists():
            continue
        raw = path.read_bytes()
        bom = b"\xef\xbb\xbf" if raw.startswith(b"\xef\xbb\xbf") else b""
        text = raw[len(bom):].decode("utf-8")
        rows = csv_field_spans(text)
        if not rows:
            continue
        headers = [field[0] for field in rows[0]]
        id_index = headers.index("id") if "id" in headers else (headers.index("target_gu_id") if "target_gu_id" in headers else 0)
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
            rarity_index = headers.index("rarity") if factor_name in ("item_prices", "gu_prices") and "rarity" in headers else None

            for line, row in enumerate(rows[1:], start=2):
                if index >= len(row) or rank_index >= len(row):
                    continue
                old, start, end = row[index]
                item_id = row[id_index][0] if id_index < len(row) else ""

                if filename == "Item.csv":
                    type_idx = headers.index("item_type") if "item_type" in headers else -1
                    item_type = row[type_idx][0] if 0 <= type_idx < len(row) else "Material"
                    if not args.include_protected_items and (item_id in PROTECTED_ITEMS or item_type != "Material"):
                        continue

                try:
                    rank = int(row[rank_index][0] or "1")
                except ValueError as exc:
                    raise ValueError(f"{filename}:{line} rank không hợp lệ") from exc
                rank = rank if 1 <= rank <= 9 else 1

                rarity = 0
                if rarity_index is not None and rarity_index < len(row):
                    try:
                        rarity = int(row[rarity_index][0] or "0")
                    except ValueError:
                        rarity = 0
                rarity = max(0, min(9, rarity))

                if factor_name == "item_prices":
                    base_val = item_curve[rank]
                    if args.rarity_factor > 0 and rarity > 0:
                        base_val = base_val * (1.0 + args.rarity_factor * rarity)
                    new = Decimal(str(base_val)).quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)
                    rank_counts.setdefault(rank, [0, 0])
                    rank_counts[rank][0] += 1
                elif factor_name == "gu_prices":
                    base_val = gu_curve[rank]
                    if args.rarity_factor > 0 and rarity > 0:
                        base_val = base_val * (1.0 + args.rarity_factor * rarity)
                    new = Decimal(str(base_val)).quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)
                    rank_counts.setdefault(rank, [0, 0])
                    rank_counts[rank][0] += 1
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

                if new == int(new):
                    new_str = str(int(new))
                else:
                    new_str = f"{new:.3f}".rstrip("0").rstrip(".")
                if old.strip() != new_str:
                    changes.append((start, end, new_str))
                    column_counts[column] = column_counts.get(column, 0) + 1
                    if factor_name in ("item_prices", "gu_prices"):
                        rank_counts[rank][1] += 1

        if changes:
            edits[path] = (bom, text, changes)
            if rank_counts:
                unit = "TNT" if False else ""
                detail_parts = []
                for rk, (total, changed) in sorted(rank_counts.items()):
                    curr = "TNT" if rk >= 6 else "NT"
                    target_price = item_curve[rk] if filename == "Item.csv" else gu_curve[rk]
                    detail_parts.append(f"R{rk} ({curr}): {changed}/{total} đổi -> {target_price}")
                detail = "; ".join(detail_parts)
                summaries.append(f"{filename}: {len(changes)} giá được cân bằng theo Rank ({detail})")
            else:
                detail = ", ".join(f"{col}: {cnt}" for col, cnt in column_counts.items())
                summaries.append(f"{filename}: {len(changes)} chi phí sẽ đổi ({detail})")

    if not summaries:
        print("Tất cả giá trị trong CSV đã chuẩn xác theo bảng giá cân bằng. Không cần thay đổi.")
        return

    print("=== BÁO CÁO CÂN BẰNG GIÁ ===")
    print("\n".join(summaries))

    if not args.apply:
        print("\n[CHẾ ĐỘ XEM TRƯỚC (DRY-RUN)] Thêm cờ --apply để ghi trực tiếp vào CSV.")
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

    print("\n[HOÀN TẤT] Đã ghi các thay đổi vào CSV.")
    print("Vui lòng vào Admin -> tab 'Dữ liệu' -> Import dataset tương ứng để cập nhật vào Database.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"Lỗi: {exc}", file=sys.stderr)
        sys.exit(1)
