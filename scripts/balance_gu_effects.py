#!/usr/bin/env python3
"""Cân bằng value effect của Gu.csv theo (key, rank).

"Key" của một effect = (op, stat/status_id/code, sub_type), ví dụ
("DAMAGE", "", "FLAT") hay ("STAT_MOD", "ATTACK", "ADD_FLAT").

Chế độ:
  - Key "dày" (>= --min-samples mẫu ở mọi rank có mặt): ref = median theo rank.
  - Key "thưa": chỉ cân khi có --hard và key nằm trong HARD_REFS (hoặc --hard-file).
  - Ngưỡng lệch: --factor (mặc định 10×, đo theo tỉ lệ) hoặc --band (dung sai ±PCT).
  - Mặc định chỉ audit (in báo cáo); --apply mới ghi CSV.

Ví dụ:
  python3 scripts/balance_gu_effects.py                     # audit, factor 10x
  python3 scripts/balance_gu_effects.py --band 0.35         # audit theo ±35% (docs)
  python3 scripts/balance_gu_effects.py --hard              # + bảng ref cứng cho key thưa
  python3 scripts/balance_gu_effects.py --hard --apply      # ghi CSV
  python3 scripts/balance_gu_effects.py --allow GU_A,GU_B   # bỏ qua Cổ đặc biệt
  python3 scripts/balance_gu_effects.py --list-keys         # phân loại key

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

# Ngưỡng lệch mạnh: coi là bất thường khi value lệch khỏi ref quá --factor lần,
# đo theo TỈ LỆ (max(v/ref, ref/v)) vì sức mạnh Cổ scale theo cấp số nhân.
# --band PCT (tuỳ chọn) đổi sang chế độ dung sai ±PCT như docs/GuRankScaling.md.

# Ref cứng cho key thưa (n < min-samples). Bật bằng --hard.
#   {"flat": x}                -> hằng số (mọi rank)
#   {"per_rank": {r: x, ...}}  -> theo từng rank
#   {"curve": "P", "k": x}     -> x * RANK_SCALE[rank]
# Chỉ seed các hằng số rõ ràng (mọi mẫu khớp nhau). Key thưa không có ở đây sẽ
# được liệt kê ở mục "sparse chưa phủ" để tự bổ sung, không bị âm thầm bỏ qua.
HARD_REFS = {
    "STAT_MOD|DAMAGE_DEALT|MULTIPLY_PERCENT": {"flat": 0.12},
    "STAT_MOD|DAO_MARK|MULTIPLY_PERCENT": {"flat": 10},
    "STAT_MOD|REFINE_SUCCESS|ADD_FLAT": {"flat": 4},
    "STAT_MOD|CC_RESIST|ADD_FLAT": {"flat": 1},
    "STAT_MOD|CRIT_RATE|ADD_FLAT": {"flat": 1},
    "STAT_MOD|CLEANSING_POWER|ADD_FLAT": {"flat": 1},
    "STAT_MOD|HEARING|ADD_FLAT": {"flat": 5},
    "STAT_MOD|MANA|ADD_FLAT": {"flat": 2},
    "STAT_MOD|TIME_SPEED|ADD_FLAT": {"flat": 0.5},
    "HEAL||LIFESTEAL": {"flat": 0.5},
    "DIRECT_HP||SHIELD_STRIP": {"flat": 0.5},
    "SHIELD||MULTIPLIER": {"flat": 2},
    "UNIQUE|INCREASE_TIME_SPEED|": {"flat": 0.5},
    "UNIQUE|REDUCE_STRESS|": {"flat": 25},
    "UNIQUE|RESTORE_MENTAL|": {"flat": 25},
    "UNIQUE|RESTORE_SOUL_HP|": {"flat": 20},
    "UNIQUE|SOUL_REGEN_TICK|": {"flat": 2},
    "UNIQUE|MANA_REGEN_TICK|": {"flat": 10},
    "STATUS|CHARM|": {"flat": 1},
    "STATUS|SLOW|": {"flat": 1},
    "STATUS|SLOW1|": {"flat": 1},
    "STATUS|SILENCE|": {"flat": 1},
    "STATUS|SLEEP|": {"flat": 1},
    "STATUS|GU_SEAL|": {"flat": 1},
    "STATUS|REVIVE_LOCK|": {"flat": 1},
    "STATUS|LAST_STAND|": {"flat": 1},
    "STATUS|ATK_UP|": {"flat": 1},
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


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--factor", type=float, default=10.0, help="lệch > factor lần ref thì coi là bất thường (mặc định 10)")
    ap.add_argument("--band", type=float, default=None, help="thay --factor bằng dung sai ±PCT (vd 0.35 theo docs)")
    ap.add_argument("--min-samples", type=int, default=5, help="số mẫu tối thiểu mỗi rank để coi là key dày")
    ap.add_argument("--hard", action="store_true", help="dùng HARD_REFS cho key thưa")
    ap.add_argument("--hard-file", help="JSON bổ sung/ghi đè HARD_REFS")
    ap.add_argument("--snap", action="store_true", help="gán bằng ref thay vì kẹp về biên")
    ap.add_argument("--allow", default="", help="danh sách gu_id bỏ qua, phân tách bằng dấu phẩy")
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
    state_changes = {}   # row -> list of applied info
    issues, uncovered = [], set()
    for rec in records:
        key, rank, val = rec["key"], rec["rank"], rec["value"]
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

    # --- báo cáo ---
    mode = f"band ±{args.band:g}" if args.band is not None else f"factor {args.factor:g}×"
    print(f"Đọc {len(parsed)} Cổ có effect, {len(records)} value số, "
          f"{len(dense)} key dày / {len(sparse)} key thưa. Ngưỡng: {mode}.")
    if not issues:
        print("Không có value nào lệch ngoài ngưỡng.")
    else:
        by_key = {}
        issues.sort(key=lambda x: -abs(x[6]))
        for gu_id, rank, key, val, new, ref, dev, source in issues:
            by_key.setdefault(key, 0)
            by_key[key] += 1
            print(f"  {gu_id:<40} R{rank} {key:<44} {val:g} -> {new:g}  (ref={ref:g}, {dev*100:+.0f}%, {source})")
        print("\nTheo key:")
        for k, n in sorted(by_key.items(), key=lambda x: -x[1]):
            print(f"  {k:<52} {n}")
    if uncovered:
        print(f"\nKey thưa chưa phủ ({len(uncovered)}) — cần --hard-file để cân:")
        for k in sorted(uncovered):
            print(f"  {k}")

    if not args.apply or not issues:
        if issues and not args.apply:
            print("\nChỉ xem trước. Thêm --apply để ghi CSV.")
        return

    # --- ghi CSV giữ nguyên format ---
    replacements = []
    for ri in sorted(state_changes):
        rank, obj, _ = parsed[ri]
        token = rows[ri][eff_idx]
        orig = token[0]
        text_out, _ = dump_style(orig, obj)
        encoded = '"' + text_out.replace('"', '""') + '"'
        replacements.append((token[1], token[2], encoded))
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
    print(f"\nĐã ghi {path.name} ({len(state_changes)} dòng). Admin -> Dữ liệu -> Import 'Cổ trùng'.")


def locate(rec):
    obj = rec["obj"]
    bi, oi, _ = rec["src"]
    blocks = obj if isinstance(obj, list) else obj["blocks"]
    return blocks[bi]["operations"][oi]


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
    obj = [{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]
    out, _ = dump_style('[{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]', obj)
    assert out == '[{"operations": [{"op": "DAMAGE", "value": 20, "sub_type": "FLAT"}]}]'
    print("self-test OK")
    return 0


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
                records.append({"row": ri, "rank": rank, "key": key, "value": val, "src": src, "obj": obj, "meta": meta})
    return records


if __name__ == "__main__":
    try:
        sys.exit(main() or 0)
    except (OSError, UnicodeError, ValueError, json.JSONDecodeError) as exc:
        print(f"Lỗi: {exc}", file=sys.stderr)
        sys.exit(1)
