#!/usr/bin/env python3
"""
scripts/balance_gu_effects.py

Cân bằng value effect của Gu.csv theo (key, rank) và thang chuẩn GuRankScaling.

"Key" của một effect = (op, stat/status_id/code, sub_type), ví dụ:
  - ("DAMAGE", "", "FLAT")
  - ("STAT_MOD", "ATTACK", "ADD_FLAT")
  - ("STATUS", "BLEED", "")
  - ("UNIQUE", "ADD_LIFESPAN_YEARS", "")

Chế độ:
  - Key "dày" (>= --min-samples mẫu ở mọi rank có mặt): ref = median theo rank.
  - Key "thưa": cân bằng dựa trên bảng chuẩn HARD_REFS (bật qua cờ --hard hoặc tự động).
  - DOT (BLEED/POISON/BURN/SLOW_DAMAGE): tổng damage một lần áp dụng tối đa
    60% damage FLAT chuẩn cùng rank, chia theo số lượt hiệu lực (turns).
  - Ngưỡng lệch: --factor (mặc định 10×) hoặc --band (dung sai ±PCT, vd: 0.35).
  - Scale tuyệt đối: --scale kiểm tra tỉ lệ chuẩn với RANK_SCALE[rank] (docs/GuRankScaling.md).
  - Mặc định chỉ xem trước (dry-run / audit); thêm --apply để ghi trực tiếp vào CSV.

Cách dùng:
  python3 scripts/balance_gu_effects.py                     # audit phát hiện lệch chuẩn & DoT vượt ngân sách
  python3 scripts/balance_gu_effects.py --hard              # audit toàn diện cả các key thưa (sparse)
  python3 scripts/balance_gu_effects.py --scale             # audit tuyệt đối theo tỷ lệ P(rank)
  python3 scripts/balance_gu_effects.py --hard --apply      # áp dụng cân bằng vào Gu.csv
  python3 scripts/balance_gu_effects.py --scale --apply     # snap các chỉ số core về tỷ lệ chuẩn

Sau --apply: Admin -> tab "Dữ liệu" -> Import bảng "Cổ trùng".
"""

import argparse
import csv
import json
import os
import statistics
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "backend" / "data"
GU = DATA / "Gu.csv"

# P(rank): bậc quy mô chuẩn dùng bởi docs/GuRankScaling.md và gamefx.RankScale.
RANK_SCALE = {1: 1, 2: 2, 3: 5, 4: 20, 5: 100, 6: 10**4, 7: 10**6, 8: 10**8, 9: 10**10}
DOT_STATUS_IDS = {"BLEED", "POISON", "BURN", "SLOW_DAMAGE"}
DOT_TOTAL_DAMAGE_RATIO = 0.6
DOT_DEFAULT_TURNS = 3
FLAT_DAMAGE_PER_RANK_SCALE = 30

# Canon tuyệt đối của dữ liệu: mỗi value phải là <ratio> × RANK_SCALE[rank].
# Ratio lấy từ phân bố thực tế của Gu.csv (median theo rank giống hệt nhau):
# SHIELD 30, HEAL 20 (bậc 15 cũ remap sang 20), HP 10, ATTACK 1, DEFENSE 5.
# DAMAGE FLAT tách riêng theo phạm vi mục tiêu (xem DAMAGE_SCOPE_RATIO).
SCALE_ARCHETYPES = {
    "SHIELD||FLAT": [30],
    "HEAL||FLAT": [15, 20],
    "STAT_MOD|HP|ADD_FLAT": [10],
    "STAT_MOD|ATTACK|ADD_FLAT": [1],
    "STAT_MOD|DEFENSE|ADD_FLAT": [5],
}

# DAMAGE FLAT theo phạm vi mục tiêu (× RANK_SCALE[rank]): đơn mục tiêu 30 (nền),
# hàng/cột 24 (80% nền), toàn bộ địch 18 (60% nền). Block multi-target bị bỏ
# DAMAGE/MULTIPLIER để ưu đãi diện rộng không nhân chồng lên sát thương.
DAMAGE_SCOPE_RATIO = {
    "SELF": 30,
    "TARGET": 30,
    "ROW": 24,
    "COL": 24,
    "ALL_ENEMIES": 18,
}
MULTI_TARGET_SELECTORS = {"ROW", "COL", "ALL_ENEMIES"}

# Bậc nền cũ -> bậc nền mới: ratio khớp CHÍNH XÁC được đổi sang ratio mới (bỏ qua
# dung sai), dùng khi nâng hệ số nền mà vẫn giữ các bậc tiện ích/đặc biệt.
SCALE_RATIO_REMAP = {
    "HEAL||FLAT": {15: 20},
}

# Bảng quy chuẩn cho các key thưa (sparse keys, n < min-samples).
# Định dạng:
#   {"flat": x}                -> giá trị hằng số (cho mọi rank)
#   {"per_rank": {r: x, ...}}  -> giá trị riêng theo từng rank
#   {"curve": "P", "k": x}     -> x * RANK_SCALE[rank]
HARD_REFS = {
    # Stat Multipliers & Flat Bonuses
    "STAT_MOD|DAMAGE_DEALT|MULTIPLY_PERCENT": {"flat": 0.12},
    "STAT_MOD|DAO_MARK|MULTIPLY_PERCENT": {"flat": 10},
    "STAT_MOD|REFINE_SUCCESS|ADD_FLAT": {"flat": 4},
    "STAT_MOD|CC_RESIST|ADD_FLAT": {"flat": 1},
    "STAT_MOD|CRIT_RATE|ADD_FLAT": {"flat": 1},
    "STAT_MOD|CLEANSING_POWER|ADD_FLAT": {"flat": 1},
    "STAT_MOD|HEARING|ADD_FLAT": {"flat": 5},
    "STAT_MOD|MANA|ADD_FLAT": {"flat": 2},
    "STAT_MOD|TIME_SPEED|ADD_FLAT": {"flat": 0.5},
    "STAT_MOD|DISGUISE|ADD_FLAT": {"flat": 1},
    "STAT_MOD|REFINE_MANA_TIER|ADD_FLAT": {"flat": 1},
    "STAT_MOD|SOUL_REFINEMENT|ADD_FLAT": {"flat": 2},
    "STAT_MOD|TENACITY|ADD_FLAT": {"flat": 5},
    "STAT_MOD|PATH_RESISTANCE|ADD_FLAT": {"flat": 0.5},
    "STAT_MOD|PATH_RESISTANCE|MULTIPLY_PERCENT": {"flat": 0.1},
    "STAT_MOD|LUCK|ADD_FLAT": {
        "per_rank": {"1": 5, "2": 10, "3": 15, "4": 20, "5": 30, "6": 50, "7": 100, "8": 200, "9": 500}
    },
    "STAT_MOD|STORAGE|ADD_FLAT": {
        "per_rank": {"1": 20, "2": 30, "3": 50, "4": 80, "5": 100, "6": 200, "7": 350, "8": 500, "9": 1000}
    },

    # Growth Caps
    "STAT_MOD|ATTACK|GROWTH_CAP": {"flat": 2},
    "STAT_MOD|DEFENSE|GROWTH_CAP": {"flat": 2},
    "STAT_MOD|HP|GROWTH_CAP": {"flat": 2},
    "STAT_MOD|SPEED|GROWTH_CAP": {
        "per_rank": {"1": 1, "2": 1, "3": 2, "4": 2, "5": 4, "6": 6, "7": 8, "8": 10, "9": 12}
    },

    # Direct HP & Shield & Heals
    "HEAL||LIFESTEAL": {"flat": 0.5},
    "HEAL||RESOURCE": {"flat": 1},
    "DIRECT_HP||SHIELD_STRIP": {"flat": 0.5},
    "DIRECT_HP||MAX_HP_PERCENT": {"flat": 0.4},
    "SHIELD||MULTIPLIER": {"flat": 2},

    # Status Effects
    "STATUS|CHARM|": {"flat": 1},
    "STATUS|SLOW|": {"flat": 1},
    "STATUS|SLOW1|": {"flat": 1},
    "STATUS|SILENCE|": {"flat": 1},
    "STATUS|SLEEP|": {"flat": 1},
    "STATUS|GU_SEAL|": {"flat": 1},
    "STATUS|REVIVE_LOCK|": {"flat": 1},
    "STATUS|LAST_STAND|": {"flat": 1},
    "STATUS|ATK_UP|": {"flat": 1},
    "STATUS|EVASION|": {"flat": 1},
    "STATUS|WEAK|": {"flat": 1},

    # Unique Effects
    "UNIQUE|INCREASE_TIME_SPEED|": {"flat": 0.5},
    "UNIQUE|DECREASE_TIME_SPEED|": {"flat": -0.5},
    "UNIQUE|REDUCE_STRESS|": {"flat": 25},
    "UNIQUE|RESTORE_MENTAL|": {"flat": 25},
    "UNIQUE|RESTORE_SOUL_HP|": {"flat": 20},
    "UNIQUE|SOUL_REGEN_TICK|": {"flat": 2},
    "UNIQUE|MANA_REGEN_TICK|": {"flat": 10},
    "UNIQUE|ADD_SOUL_MAX|": {"flat": 10},
    "UNIQUE|ADD_THOUGHT_MAX|": {"flat": 10},
    "UNIQUE|CAPTURE_BONUS|": {"flat": 0.25},
    "UNIQUE|CC_RESIST|": {"flat": 0.5},
    "UNIQUE|CULTIVATION_EFFICIENCY|": {"flat": 0.5},
    "UNIQUE|DAO_MARK_GAIN|": {"flat": 1.0},
    "UNIQUE|LUCK|": {"flat": 10},
    "UNIQUE|MENTAL_RESISTANCE|": {"flat": 0.8},
    "UNIQUE|PATH_RESISTANCE_MODIFIER|": {"flat": 0.5},
    "UNIQUE|SOUL_DEFENSE|": {"flat": 0.6},
    "UNIQUE|STAT_MODS|": {"flat": 10},
    "UNIQUE|TENACITY|": {"flat": 0.3},
    "UNIQUE|HP_REGEN|": {"curve": "P", "k": 5},
    "UNIQUE|ADD_LIFESPAN_YEARS|": {
        "per_rank": {"1": 1, "2": 5, "3": 10, "4": 50, "5": 100, "6": 500, "7": 1000, "8": 2000, "9": 5000}
    },
    "UNIQUE|GROWTH_DURATION_HOURS|": {
        "per_rank": {"1": 24, "2": 24, "3": 30, "4": 50, "5": 120, "6": 240, "7": 240, "8": 240, "9": 240}
    },
    "UNIQUE|GENERATE_STONES_TICK|": {
        "per_rank": {"1": 10, "2": 20, "3": 50, "4": 100, "5": 200, "6": 500, "7": 1000, "8": 2000, "9": 5000}
    },
    "UNIQUE|GENERATE_IMMORTAL_STONES_TICK|": {
        "per_rank": {"1": 1, "2": 1, "3": 1, "4": 5, "5": 10, "6": 100, "7": 1000, "8": 1000, "9": 1000}
    },
    "UNIQUE|THOUGHT_REGEN_TICK|": {
        "per_rank": {"1": 1, "2": 2, "3": 5, "4": 10, "5": 20, "6": 50, "7": 100, "8": 200, "9": 500}
    },
}


def key_str(op, stat, sub):
    return f"{op or ''}|{stat or ''}|{sub or ''}"


def csv_field_spans(text):
    """Trả span từng field CSV, giữ nguyên format gốc của file."""
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


def is_number(v):
    return isinstance(v, (int, float)) and not isinstance(v, bool)


def dump_style(orig, obj):
    """Chọn separator khiến json.dumps tái tạo đúng chuỗi gốc (giảm diff)."""
    for sep in ((", ", ": "), (",", ":")):
        out = json.dumps(obj, ensure_ascii=False, separators=sep)
        if out == orig:
            return out, sep
    out = json.dumps(obj, ensure_ascii=False, separators=(", ", ": "))
    return out, (", ", ": ")


def num_out(x):
    if isinstance(x, float) and x.is_integer() and abs(x) < 1e15:
        return int(x)
    if isinstance(x, float):
        return round(x, 6)
    return x


def compute_ref(values):
    return statistics.median(values)


def clamp_value(v, ref, factor):
    """Kẹp v vào khoảng [ref/factor, ref*factor]."""
    lo, hi = ref / factor, ref * factor
    if v < lo:
        return lo
    if v > hi:
        return hi
    return v


def in_band(v, ref, factor, band):
    if band is not None:
        return abs(v - ref) / ref <= band
    return max(v / ref, ref / v) <= factor


def dot_damage_cap(rank, turns, ratio=DOT_TOTAL_DAMAGE_RATIO):
    """Giới hạn tổng sát thương DoT ở ratio (mặc định 60%) damage FLAT chuẩn cùng rank."""
    rank_scale = RANK_SCALE[min(max(rank, 1), 9)]
    duration = turns if turns > 0 else DOT_DEFAULT_TURNS
    return FLAT_DAMAGE_PER_RANK_SCALE * rank_scale * ratio / duration


def build_parsed(rows, id_idx, rank_idx, eff_idx):
    parsed = {}
    for ri, row in enumerate(rows):
        if eff_idx >= len(row):
            continue
        eff = row[eff_idx][0].strip()
        if not eff:
            continue
        try:
            obj = json.loads(eff)
        except json.JSONDecodeError:
            continue
        try:
            rank = int(row[rank_idx][0] or "1")
        except (ValueError, IndexError):
            continue
        if rank < 1 or rank > 9:
            rank = 1
        parsed[ri] = (rank, obj, {"id": row[id_idx][0] if id_idx < len(row) else ""})
    return parsed


def locate(rec):
    obj = rec["obj"]
    bi, oi, _ = rec["src"]
    blocks = obj if isinstance(obj, list) else obj["blocks"]
    return blocks[bi]["operations"][oi]


def _extract(parsed):
    """Pass 1: gom record value số từ parsed states."""
    records = []
    for ri, (rank, obj, meta) in parsed.items():
        blocks = obj if isinstance(obj, list) else obj.get("blocks", [])
        for bi, blk in enumerate(blocks):
            if not isinstance(blk, dict):
                continue
            for oi, op in enumerate(blk.get("operations", [])):
                if not isinstance(op, dict):
                    continue
                o = op.get("op")
                tgt = blk.get("target")
                selector = tgt.get("selector") if isinstance(tgt, dict) else None
                if o == "STATUS":
                    key, val, src = key_str("STATUS", op.get("status_id"), ""), op.get("value"), (bi, oi, "value")
                elif o == "UNIQUE":
                    params = op.get("params")
                    if not isinstance(params, dict) or not is_number(params.get("value")):
                        continue
                    key, val, src = key_str("UNIQUE", op.get("code"), ""), params.get("value"), (bi, oi, "params.value")
                else:
                    key, val, src = key_str(o, op.get("stat"), op.get("sub_type")), op.get("value"), (bi, oi, "value")
                if not is_number(val):
                    continue
                records.append({
                    "row": ri, "rank": rank, "key": key, "value": val,
                    "src": src, "obj": obj, "meta": meta, "op": op,
                    "selector": selector,
                })
    return records


def apply_changes(path, rows, parsed, state_changes, text, bom, eff_idx):
    """Ghi CSV, chỉ thay đúng cell effects, giữ nguyên format gốc."""
    replacements = []
    for ri in sorted(state_changes):
        _, obj, _ = parsed[ri]
        token = rows[ri][eff_idx]
        text_out, _ = dump_style(token[0], obj)
        replacements.append((token[1], token[2], '"' + text_out.replace('"', '""') + '"'))
    for start, end, value in sorted(replacements, reverse=True):
        text = text[:start] + value + text[end:]
    fd, tmp = tempfile.mkstemp(dir=path.parent, prefix=f".{path.name}.")
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(bom + text.encode("utf-8"))
        os.chmod(tmp, path.stat().st_mode)
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def nearest_arch(ratio, archetypes):
    return min(archetypes, key=lambda a: abs(ratio - a))


def x_dev(value, ref):
    return (value / ref - 1) * 100 if ref else 0


def strip_multitarget_multipliers(parsed, allow):
    """Bỏ DAMAGE/MULTIPLIER trong block multi-target (ưu đãi diện rộng đã nằm ở ratio FLAT)."""
    changes = {}
    for ri, (_, obj, meta) in parsed.items():
        if meta["id"] in allow:
            continue
        blocks = obj if isinstance(obj, list) else obj.get("blocks", [])
        for blk in blocks:
            if not isinstance(blk, dict):
                continue
            tgt = blk.get("target")
            selector = tgt.get("selector") if isinstance(tgt, dict) else None
            if selector not in MULTI_TARGET_SELECTORS:
                continue
            ops = blk.get("operations")
            if not isinstance(ops, list):
                continue
            kept = [op for op in ops
                    if not (isinstance(op, dict) and op.get("op") == "DAMAGE"
                            and op.get("sub_type") == "MULTIPLIER")]
            if len(kept) != len(ops):
                blk["operations"] = kept
                changes[ri] = meta["id"]
    return changes


def check_scale(parsed, records, args, rows, text, bom, eff_idx, path, allow):
    """Audit tuyệt đối: mỗi value core phải là <ratio> × RANK_SCALE[rank]."""
    tol = args.scale_tol
    issues, skipped = [], 0
    state_changes = {}
    for ri, gu_id in strip_multitarget_multipliers(parsed, allow).items():
        state_changes.setdefault(ri, []).append(gu_id)

    for rec in records:
        key = rec["key"]
        # DAMAGE FLAT theo phạm vi mục tiêu; DAMAGE MULTIPLIER đã bị bỏ ở block multi-target.
        if key == "DAMAGE||MULTIPLIER":
            continue
        if key == "DAMAGE||FLAT":
            target = DAMAGE_SCOPE_RATIO.get(rec.get("selector") or "")
            if target is None:
                continue
        else:
            archetypes = SCALE_ARCHETYPES.get(key)
            if not archetypes:
                continue
        gu_id = parsed[rec["row"]][2]["id"]
        if gu_id in allow:
            skipped += 1
            continue
        rank, val = rec["rank"], rec["value"]
        if not isinstance(val, (int, float)) or isinstance(val, bool) or val == 0:
            continue
        ratio = val / RANK_SCALE[rank]
        if key == "DAMAGE||FLAT":
            if target * (1 - tol) <= ratio <= target * (1 + tol):
                continue
            near = target
        else:
            near = SCALE_RATIO_REMAP.get(key, {}).get(ratio)
            if near is None:
                near = nearest_arch(ratio, archetypes)
                if near * (1 - tol) <= ratio <= near * (1 + tol):
                    continue
        new = num_out(near * RANK_SCALE[rank])
        if new == val:
            continue
        op = locate(rec)
        if rec["src"][2] == "params.value":
            op["params"]["value"] = new
        else:
            op["value"] = new
        issues.append((gu_id, rank, key, val, new, near, ratio))
        state_changes.setdefault(rec["row"], []).append(gu_id)

    covered = sum(1 for r in records if r["key"] in SCALE_ARCHETYPES or r["key"] == "DAMAGE||FLAT")
    print(f"[--scale] {covered} value core, ratio chuẩn: "
          + ", ".join(f"{k}= {v}" for k, v in SCALE_ARCHETYPES.items())
          + ", DAMAGE||FLAT= " + str(DAMAGE_SCOPE_RATIO)
          + f". Dung sai ±{tol:g}.")
    if skipped:
        print(f"Bỏ qua {skipped} value thuộc --allow.")
    if not issues:
        print("Tất cả value core đúng scale.")
        return 0
    issues.sort(key=lambda x: -abs(x[6] / x[5] - 1) if x[5] else 0)
    for gu_id, rank, key, val, new, near, ratio in issues:
        print(f"  {gu_id:<40} R{rank} {key:<28} {val:g} -> {new:g}  "
              f"(ratio {ratio:g} -> {near:g}, {x_dev(ratio, near):+.0f}%)")
    print(f"\n{len(issues)} value lệch scale trên {len(state_changes)} dòng.")
    if not args.apply:
        print("Chỉ xem trước. Thêm --apply để ghi CSV.")
        return 0
    apply_changes(path, rows, parsed, state_changes, text, bom, eff_idx)
    print(f"Đã ghi {path.name} ({len(state_changes)} dòng). Admin -> Dữ liệu -> Import 'Cổ trùng'.")
    return 0


def self_test():
    assert compute_ref([12, 12, 12, 12000000]) == 12
    assert clamp_value(12000000, 12, 10) == 120
    assert clamp_value(1, 12, 10) == 1.2
    assert in_band(20, 12, 10, None) is True
    assert in_band(121, 12, 10, None) is False
    assert in_band(16, 12, None, 0.35) is True
    assert in_band(20, 12, None, 0.35) is False
    assert num_out(16.2) == 16.2
    assert num_out(4.0) == 4
    assert num_out(2000.0) == 2000
    assert key_str("STAT_MOD", "ATTACK", "ADD_FLAT") == "STAT_MOD|ATTACK|ADD_FLAT"
    assert key_str("STATUS", "BLEED", None) == "STATUS|BLEED|"
    assert dot_damage_cap(4, 3) == 120
    assert dot_damage_cap(4, 6) == 60
    assert dot_damage_cap(99, 3) == 60000000000
    assert SCALE_RATIO_REMAP["HEAL||FLAT"][15] == 20
    assert DAMAGE_SCOPE_RATIO["ALL_ENEMIES"] == 18 and DAMAGE_SCOPE_RATIO["COL"] == 24
    _p = {0: (1, [{"target": {"selector": "ALL_ENEMIES"}, "operations": [
        {"op": "DAMAGE", "sub_type": "FLAT", "value": 8},
        {"op": "DAMAGE", "sub_type": "MULTIPLIER", "value": 2.5}]}], {"id": "GU_X"})}
    assert strip_multitarget_multipliers(_p, set()) == {0: "GU_X"}
    assert _p[0][1][0]["operations"] == [{"op": "DAMAGE", "sub_type": "FLAT", "value": 8}]
    obj = [{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]
    out, _ = dump_style('[{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]', obj)
    assert out == '[{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]'
    print("self-test OK")
    return 0


def main():
    ap = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    ap.add_argument("--factor", type=float, default=10.0, help="lệch > factor lần ref thì coi là bất thường (mặc định 10)")
    ap.add_argument("--band", type=float, default=None, help="thay --factor bằng dung sai ±PCT (vd 0.35 theo docs)")
    ap.add_argument("--scale", action="store_true", help="audit tuyệt đối theo scale: value = ratio_chuẩn × RANK_SCALE[rank]")
    ap.add_argument("--scale-tol", type=float, default=0.4, help="dung sai ± cho --scale (mặc định 0.4)")
    ap.add_argument("--min-samples", type=int, default=5, help="số mẫu tối thiểu mỗi rank để coi là key dày")
    ap.add_argument("--hard", action="store_true", help="dùng HARD_REFS cho key thưa")
    ap.add_argument("--hard-file", help="JSON bổ sung/ghi đè HARD_REFS")
    ap.add_argument("--snap", action="store_true", help="gán bằng ref thay vì kẹp về biên")
    ap.add_argument("--allow", default="", help="danh sách gu_id bỏ qua, phân tách bằng dấu phẩy")
    ap.add_argument("--dot-ratio", type=float, default=DOT_TOTAL_DAMAGE_RATIO, help="tỷ lệ ngân sách damage của DoT so với Flat chuẩn (mặc định 0.6 = 60%%)")
    ap.add_argument("--apply", action="store_true", help="ghi CSV (mặc định chỉ xem trước)")
    ap.add_argument("--list-keys", action="store_true", help="chỉ in phân loại key rồi thoát")
    ap.add_argument("--dump-hard", action="store_true", help="in HARD_REFS skeleton từ median của key thưa")
    ap.add_argument("--self-test", action="store_true", help="chạy kiểm tra logic nội bộ")
    ap.add_argument("--csv", default=str(GU), help="đường dẫn Gu.csv")
    args = ap.parse_args()

    if args.self_test:
        return self_test()

    path = Path(args.csv)
    raw = path.read_bytes()
    bom = b"\xef\xbb\xbf" if raw.startswith(b"\xef\xbb\xbf") else b""
    text = raw[len(bom):].decode("utf-8")
    rows = csv_field_spans(text)
    headers = [f[0] for f in rows[0]]
    id_idx = headers.index("id")
    rank_idx = headers.index("rank")
    eff_idx = headers.index("effects")

    hard = dict(HARD_REFS)
    if args.hard_file:
        hard.update(json.loads(Path(args.hard_file).read_text(encoding="utf-8")))
    hard_enabled = args.hard or bool(args.hard_file)
    allow = {x.strip() for x in args.allow.split(",") if x.strip()}

    parsed = build_parsed(rows, id_idx, rank_idx, eff_idx)
    records = _extract(parsed)

    if args.scale:
        return check_scale(parsed, records, args, rows, text, bom, eff_idx, path, allow)

    # --- gom mẫu theo (key, rank) ---
    samples = {}
    for rec in records:
        samples.setdefault(rec["key"], {}).setdefault(rec["rank"], []).append(rec["value"])

    # --- phân loại key + reference ---
    dense, sparse = set(), set()
    for key, per_rank in samples.items():
        if min(len(v) for v in per_rank.values()) >= args.min_samples:
            dense.add(key)
        else:
            sparse.add(key)

    def ref_for(key, rank):
        if key in dense:
            return compute_ref(samples[key][rank]), "auto"
        if hard_enabled and key in hard:
            spec = hard[key]
            if "flat" in spec:
                return float(spec["flat"]), "hard"
            if "curve" in spec:
                return float(spec["k"]) * RANK_SCALE[rank], "hard"
            if "per_rank" in spec and str(rank) in spec["per_rank"]:
                return float(spec["per_rank"][str(rank)]), "hard"
        return None, None

    if args.list_keys or args.dump_hard:
        if args.dump_hard:
            skeleton = {}
            for key in sorted(sparse):
                per = {str(r): compute_ref(samples[key][r]) for r in sorted(samples[key])}
                skeleton[key] = {"per_rank": per}
            print(json.dumps(skeleton, ensure_ascii=False, indent=2))
            return
        print(f"dense keys ({len(dense)}):")
        for k in sorted(dense):
            ranks = sorted(samples[k])
            print(f"  {k:<52} ranks={min(ranks)}-{max(ranks)}")
        print(f"\nsparse keys ({len(sparse)}):")
        for k in sorted(sparse):
            if hard_enabled and k in hard:
                print(f"  {k:<52} HARD")
            else:
                print(f"  {k:<52} uncovered (audit only)")
        return

    # --- đánh giá & sửa ---
    state_changes = {}
    issues, uncovered = [], set()

    for rec in records:
        key, rank, val = rec["key"], rec["rank"], rec["value"]
        status_id = rec["op"].get("status_id", "")

        # 1. Kiểm tra ngân sách DoT (Bleed, Poison, Burn, Slow Damage)
        if rec["op"].get("op") == "STATUS" and isinstance(status_id, str) and status_id in DOT_STATUS_IDS:
            gu_id = parsed[rec["row"]][2]["id"]
            if gu_id in allow:
                continue
            turns = rec["op"].get("turns", DOT_DEFAULT_TURNS)
            turns = turns if isinstance(turns, int) and not isinstance(turns, bool) else DOT_DEFAULT_TURNS
            ref = dot_damage_cap(rank, turns, args.dot_ratio)
            if val <= ref:
                continue
            new = num_out(ref)
            op = locate(rec)
            op["value"] = new
            issues.append((gu_id, rank, key, val, new, ref, val / ref - 1, "dot-budget"))
            state_changes.setdefault(rec["row"], []).append(gu_id)
            continue

        # 2. Kiểm tra key dense hoặc hard ref
        ref, source = ref_for(key, rank)
        if ref is None:
            if key in sparse:
                uncovered.add(key)
            continue
        if ref == 0:
            continue
        gu_id = parsed[rec["row"]][2]["id"]
        if gu_id in allow:
            continue
        if in_band(val, ref, args.factor, args.band):
            continue

        if args.band is not None:
            lo, hi = ref * (1 - args.band), ref * (1 + args.band)
            new = ref if args.snap else (lo if val < lo else hi)
        else:
            new = ref if args.snap else clamp_value(val, ref, args.factor)
        new = num_out(new)
        if new == val:
            continue

        dev = val / ref - 1
        op = locate(rec)
        if rec["src"][2] == "params.value":
            op["params"]["value"] = new
        else:
            op["value"] = new
        issues.append((gu_id, rank, key, val, new, ref, dev, source))
        state_changes.setdefault(rec["row"], []).append(gu_id)

    # --- Báo cáo ---
    mode = f"band ±{args.band:g}" if args.band is not None else f"factor {args.factor:g}×"
    print(f"Đọc {len(parsed)} Cổ có effect, {len(records)} value số, "
          f"{len(dense)} key dày / {len(sparse)} key thưa. Ngưỡng: {mode}.")
    if not issues:
        print("Tất cả value effect đã chuẩn xác và nằm trong ngưỡng cân bằng.")
    else:
        by_key = {}
        issues.sort(key=lambda x: -abs(x[6]))
        print(f"\nPhát hiện {len(issues)} effect cần cân bằng:")
        for gu_id, rank, key, val, new, ref, dev, source in issues:
            by_key.setdefault(key, 0)
            by_key[key] += 1
            print(f"  {gu_id:<40} R{rank} {key:<44} {val:g} -> {new:g}  (ref={ref:g}, {dev*100:+.0f}%, {source})")
        print("\nTổng hợp theo loại effect:")
        for k, n in sorted(by_key.items(), key=lambda x: -x[1]):
            print(f"  {k:<52} {n}")

    if uncovered:
        print(f"\nKey thưa chưa phủ ({len(uncovered)}) — cần --hard-file hoặc bổ sung HARD_REFS:")
        for k in sorted(uncovered):
            print(f"  {k}")
    else:
        if hard_enabled:
            print("\n[ĐỘ PHỦ 100%] Toàn bộ các key thưa (sparse keys) đã được bao phủ bởi HARD_REFS.")

    if not args.apply or not issues:
        if issues and not args.apply:
            print("\n[CHẾ ĐỘ XEM TRƯỚC (DRY-RUN)] Thêm cờ --apply để ghi trực tiếp vào Gu.csv.")
        return

    # --- Ghi CSV giữ nguyên format ---
    apply_changes(path, rows, parsed, state_changes, text, bom, eff_idx)
    print(f"\n[HOÀN TẤT] Đã ghi các thay đổi vào {path.name} ({len(state_changes)} dòng Cổ).")
    print("Vui lòng vào Admin -> Dữ liệu -> Import 'Cổ trùng' để cập nhật vào Database.")


if __name__ == "__main__":
    try:
        sys.exit(main() or 0)
    except (OSError, UnicodeError, ValueError, json.JSONDecodeError) as exc:
        print(f"Lỗi: {exc}", file=sys.stderr)
        sys.exit(1)
