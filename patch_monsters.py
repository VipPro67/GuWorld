import csv
import json
import uuid

gu_map = {
    'monster_wild_boar': ['GU_WHITE_BOAR'],
    'monster_stone_ape': ['GU_STONE_SKIN'],
    'monster_ghost_bat': ['GU_WOLF_SOUL'],
    'monster_boar_king': ['GU_BLACK_BRISTLE', 'GU_WHITE_BOAR'],
    'monster_blood_spider': ['GU_BLOOD_MOON'],
    'monster_lava_python': ['GU_SUN_FIRE', 'GU_FIRE_OIL'],
    'monster_iron_bear': ['GU_IRON_BONE', 'GU_BEAR_STRENGTH'],
    'boss_wolf_king': ['GU_MOONLIGHT', 'GU_WOLF_SOUL', 'GU_WOLF_ENSLAVEMENT_2'],
    'monster_earth_calamity_boss': ['GU_PULLING_MOUNTAIN'],
    'monster_heaven_tribulation_boss': ['GU_HEAVENLY_THUNDER'],
    'monster_grand_calamity_boss': ['GU_FATE'],
    'monster_myriad_tribulation_boss': ['GU_MYRIAD_SELF'],
    'monster_wandering': ['GU_MOONLIGHT'],
    'monster_wolf_leader': ['GU_BLACK_BOAR'],
    'monster_sand_wraith': ['GU_GHOST_CRY']
}

rows = []
with open('backend/Monster.csv', 'r') as f:
    reader = csv.DictReader(f)
    fieldnames = reader.fieldnames
    for row in reader:
        row['max_mana'] = '0'
        row['current_mana'] = '0'
        
        m_id = row['id']
        if m_id in gu_map:
            gu_list = []
            for g in gu_map[m_id]:
                gu_list.append({
                    "instance_id": f"mon-{uuid.uuid4().hex[:8]}",
                    "gu_id": g,
                    "hunger": 100,
                    "status": "ACTIVE",
                    "equipped": True
                })
            row['gu_inventory'] = json.dumps(gu_list)
        rows.append(row)

with open('backend/Monster.csv', 'w', newline='') as f:
    writer = csv.DictWriter(f, fieldnames=fieldnames)
    writer.writeheader()
    writer.writerows(rows)
