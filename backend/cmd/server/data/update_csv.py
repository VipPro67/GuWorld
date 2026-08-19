import json

updates = {
    "qing_mao_mountain": ["sealing_stone", "stone_monkey_fur", "black_lacquer_mud", "iron", "nine_beast_kings_eyes", "blood_coagulation_grass", "blood_dew", "blood_silkworm_cocoon", "grass_puppet", "star_shoot_grass", "arrow_bamboo_water", "spirit_deer_meat"],
    "bach_cot_son": ["sealing_stone", "stone_man_rice", "spirit_deer_meat"],
    "three_kings_mountain": ["summit_stone", "nine_palace_flower", "golden_crow_meteor", "copper_spike", "sealing_stone"],
    "blood_lake_purple_sand": ["blood_bat_egg", "blood_qi_pearl", "blood_seal_clay", "blood_coagulation_grass", "blood_dew", "sealing_stone"],
    "van_duc_dong": ["blood_bat_egg", "blood_silkworm_cocoon", "sealing_stone"],
    "wan_xa_goc": ["four_flavors_wine", "seven_spices", "six_headed_snake_blood", "sealing_stone"],
    "hu_tien_phuc_dia": ["mountain_water_qi_laughing_stone", "book_worm", "dan_qing_incense", "sealing_stone"],
    "tuyet_nguyen_nhan_tuyet": ["spirit_deer_meat", "white_lotus_giant_silkworm", "sealing_stone"],
    "huyet_sa_khau": ["blood_qi_pearl", "blood_seal_clay", "blood_marrow_king", "blood_coagulation_grass", "blood_dew", "sealing_stone"],
    "bat_bai_coc": ["golden_crow_meteor", "nine_palace_flower", "blade_food", "sealing_stone"],
    "lac_thien_coc": ["glory_of_the_immemorial_light", "guts_gu", "sealing_stone"],
    "long_ngu_dao": ["thread_water", "water_defense_shell", "six_headed_snake_blood"],
    "khi_hai_vuc": ["regret_water", "sand_gull_soil", "heavenly_crystal"],
    "quang_am_chi_ha": ["historical_dust", "regret_pool_fog_flower", "water_of_mask"],
    "dang_hon_son": ["guts_gu", "hairy_man_slaves", "blood_pool_immortal_essence"],
    "nghich_luu_ha": ["abundant_spring_light", "mystic_dark_wind", "rank8_blood_sea_marrow"],
    "sinh_tu_mon": ["black_phoenix_heart", "gale_of_the_nine_heavens", "rank7_blood_marrow", "heart_blood"],
    "dai_vuc_hac_thien": ["blade_path_material", "cloud_path_material", "enslavement_path_material", "food_path_material", "formation_path_material", "human_path_material", "information_path_material", "light_path_material", "moon_path_material", "pill_path_material", "rule_path_material", "sound_path_material", "space_path_material", "transformation_path_material", "water_path_material", "wind_path_material", "wisdom_path_material", "wood_path_material", "ice_path_material"]
}

file_path = '/home/vippro67/Documents/GitHub/GuWorld/backend/cmd/server/data/Region.csv'

with open(file_path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

new_lines = []
for line in lines:
    line_strip = line.strip()
    if not line_strip:
        new_lines.append(line)
        continue
    
    parts = line_strip.split(',')
    region_id = parts[0]
    
    if region_id in updates:
        idx = line_strip.find('"{""cell_type_weights""')
        if idx != -1:
            prefix = line_strip[:idx]
            json_str = line_strip[idx+1:-1].replace('""', '"')
            data = json.loads(json_str)
            
            if 'resources' in data:
                current_resources = set(data['resources'])
                for item in updates[region_id]:
                    current_resources.add(item)
                data['resources'] = sorted(list(current_resources))
            
            new_json_str = json.dumps(data, separators=(', ', ': '))
            new_json_str_escaped = new_json_str.replace('"', '""')
            
            new_line = f'{prefix}"{new_json_str_escaped}"\n'
            new_lines.append(new_line)
        else:
            new_lines.append(line)
    else:
        new_lines.append(line)

with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(new_lines)
print("Updated successfully!")
