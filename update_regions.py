import csv
import json
import os

filepath = '/home/vippro67/Documents/GitHub/GuWorld/backend/cmd/server/data/Region.csv'

rows = []
with open(filepath, 'r', encoding='utf-8') as f:
    reader = csv.DictReader(f)
    fieldnames = reader.fieldnames
    for row in reader:
        rows.append(row)

for row in rows:
    if row['zone_type'] == 'HUB':
        continue
    
    try:
        max_rank = int(row['max_rank'])
    except ValueError:
        continue

    # Define high rank (e.g., max_rank >= 6)
    if max_rank >= 6:
        # pool_weights is like "{""cell_type_weights"": {""Monster"": 24, ""Resource"": 16, ""WildGu"": 4, ""Empty"": 56}, ...}"
        # we parse it normally:
        # replace any weird quoting if necessary? CSV reader handles double quotes.
        pool_weights = json.loads(row['pool_weights'])
        weights = pool_weights.get('cell_type_weights', {})
        
        if 'Monster' in weights:
            # Increase monster by 15
            weights['Monster'] = min(85, weights['Monster'] + 15)
            # Decrease resource
            if 'Resource' in weights:
                weights['Resource'] = max(1, weights['Resource'] // 2)
            # Decrease WildGu
            if 'WildGu' in weights:
                weights['WildGu'] = max(1, weights['WildGu'] // 2)
            
            # Recalculate Empty to make sum = 100
            total_non_empty = sum(v for k, v in weights.items() if k != 'Empty')
            if total_non_empty > 100:
                weights['Monster'] -= (total_non_empty - 100)
                total_non_empty = 100
            
            weights['Empty'] = 100 - total_non_empty
            
            pool_weights['cell_type_weights'] = weights
            row['pool_weights'] = json.dumps(pool_weights)

with open(filepath, 'w', encoding='utf-8', newline='') as f:
    # Need to ensure json fields are quoted like original
    writer = csv.DictWriter(f, fieldnames=fieldnames, quoting=csv.QUOTE_MINIMAL)
    writer.writeheader()
    writer.writerows(rows)

print("Updated Region.csv successfully.")
