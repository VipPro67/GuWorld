import csv
import json
import os

base_stats = {
    1: {"dmg": 25, "def": 20, "shield": 100, "mana": 10.0},
    2: {"dmg": 50, "def": 40, "shield": 200, "mana": 20.0},
    3: {"dmg": 125, "def": 100, "shield": 500, "mana": 30.0},
    4: {"dmg": 500, "def": 400, "shield": 2000, "mana": 40.0},
    5: {"dmg": 2500, "def": 2000, "shield": 10000, "mana": 50.0},
    6: {"dmg": 25000, "def": 20000, "shield": 100000, "mana": 1.0},
    7: {"dmg": 250000, "def": 200000, "shield": 1000000, "mana": 2.0},
    8: {"dmg": 2500000, "def": 2000000, "shield": 10000000, "mana": 3.0},
    9: {"dmg": 250000000, "def": 200000000, "shield": 1000000000, "mana": 4.0},
}

dmg_keys = ["damage", "trap_damage", "water_damage", "soul_damage", "burn", "add_base_attack", "heal"]
def_keys = ["add_base_defense"]
shield_keys = ["virtual_shield", "team_shield", "orbit_shield"]

csv_path = 'backend/Gu.csv'
temp_path = 'backend/Gu_temp.csv'

with open(csv_path, 'r', encoding='utf-8-sig') as f_in, open(temp_path, 'w', encoding='utf-8-sig', newline='') as f_out:
    reader = csv.DictReader(f_in)
    fieldnames = reader.fieldnames
    writer = csv.DictWriter(f_out, fieldnames=fieldnames)
    writer.writeheader()
    
    for row in reader:
        rank = int(row['rank'])
        b = base_stats.get(rank, base_stats[1])
        
        dmg_val = b["dmg"]
        def_val = b["def"]
        shield_val = b["shield"]
        
        if row['id'] == 'GU_FLYING_SWORD':
            dmg_val = 350000.0
        elif row['id'] == 'GU_BLOOD_CURTAIN_SKY_FLOWER':
            shield_val = 120000.0
            
        row['base_mana_cost'] = str(b["mana"])
        
        effects = json.loads(row['effects'])
        
        for cat in ['active', 'passive']:
            cat_dict = effects.get(cat, {})
            for k, v in list(cat_dict.items()):
                if k in dmg_keys:
                    cat_dict[k] = float(dmg_val)
                elif k in def_keys:
                    cat_dict[k] = float(def_val)
                elif k in shield_keys:
                    cat_dict[k] = float(shield_val)
        
        row['effects'] = json.dumps(effects, separators=(',', ':'))
        
        writer.writerow(row)

os.replace(temp_path, csv_path)
print("Updated Gu.csv successfully.")
