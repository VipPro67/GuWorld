import csv
import json
from collections import defaultdict

rank_stats = defaultdict(list)
with open('backend/Gu.csv', 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for row in reader:
        r = int(row['rank'])
        effects = json.loads(row['effects'])
        for cat in ['active', 'passive']:
            for k, v in effects.get(cat, {}).items():
                if type(v) in (int, float) and v > 0:
                    if 'damage' in k or 'attack' in k:
                        rank_stats[f"R{r}_DMG"].append(v)
                    elif 'defense' in k:
                        rank_stats[f"R{r}_DEF"].append(v)
                    elif 'shield' in k:
                        rank_stats[f"R{r}_SHIELD"].append(v)

for k in sorted(rank_stats.keys()):
    vals = sorted(rank_stats[k])
    if vals:
        print(f"{k}: min={min(vals)}, max={max(vals)}, median={vals[len(vals)//2]}")
