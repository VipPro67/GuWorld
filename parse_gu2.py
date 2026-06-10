import csv
import json

keys = set()
with open('backend/Gu.csv', 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for row in reader:
        effects = json.loads(row['effects'])
        for cat in ['active', 'passive', 'support', 'house']:
            for k in effects.get(cat, {}):
                if any(x in k for x in ['damage', 'defense', 'attack', 'shield', 'heal', 'burn', 'bleed', 'poison']):
                    keys.add(f"{cat}.{k}")
print(keys)
