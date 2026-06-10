import csv
import json

with open('backend/Gu.csv', 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    for i, row in enumerate(reader):
        if i >= 5:
            break
        print(f"Name: {row['name']}, Rank: {row['rank']}, Type: {row['gu_type']}")
        print(f"Effects: {json.dumps(json.loads(row['effects']), indent=2, ensure_ascii=False)}")
        print("-" * 40)
