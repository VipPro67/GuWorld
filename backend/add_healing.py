import csv

gu_file = 'Gu.csv'
item_file = 'Item.csv'

new_gus = [
    ["GU_HEALING_GRASS", "Sinh Cơ Thảo Cổ", "1", "Wood Path", "Healing", "10.0", '{"active":{"healing":30.0}}', "2", '["water"]', "100", "Khôi phục sinh lực nhỏ lẻ", "Southern_Border", "COMMON", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]'],
    ["GU_WHITE_BOAR_RECOVERY", "Bạch Trư Hồi Phục Cổ", "2", "Strength Path", "Healing", "20.0", '{"active":{"healing":60.0,"add_base_attack":10.0}}', "3", '["meat"]', "100", "Hồi phục kết hợp cường hóa", "Northern_Plains", "COMMON", "false", "HEALER", '["HEALER","FIGHTER"]', '["healing","buff"]', '["healing"]'],
    ["GU_BONE_FLESH_UNITY", "Cốt Nhục Đoàn Viên Cổ", "3", "Human Path", "Healing", "50.0", '{"active":{"healing":150.0}}', "4", '["bone"]', "100", "Trị liệu cực mạnh cho bản thân", "Southern_Border", "UNCOMMON", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]'],
    ["GU_BLOOD_HEALING", "Huyết Liệu Cổ", "3", "Blood Path", "Healing", "30.0", '{"active":{"healing":200.0,"backlash":50.0}}', "5", '["blood"]', "100", "Chữa thương mạnh nhưng tà đạo", "Central_Continent", "RARE", "false", "HEALER", '["HEALER"]', '["healing","blood"]', '["healing"]'],
    ["GU_SELF_RECOVERY", "Tự Lực Canh Sinh Cổ", "4", "Strength Path", "Healing", "80.0", '{"active":{"healing":300.0}}', "5", '["meat"]', "100", "Tự thân khôi phục thương thế", "Northern_Plains", "RARE", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]'],
    ["GU_MAN_AS_BEFORE", "Nhân Như Cố Cổ", "6", "Time Path", "Healing", "200.0", '{"active":{"healing":1000.0,"restore_aperture":100.0}}', "10", '["time_water"]', "100", "Khôi phục trạng thái như cũ", "Southern_Border", "LEGENDARY", "true", "HEALER", '["HEALER"]', '["healing","time"]', '["healing"]'],
    ["GU_VITALITY_SPROUT", "Sinh Mệnh Nha Cổ", "2", "Wood Path", "Healing", "15.0", '{"active":{"regen_hp":10.0}}', "2", '["water"]', "100", "Hồi phục kéo dài", "Eastern_Sea", "COMMON", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]'],
    ["GU_WOOD_REGENERATION", "Mộc Sinh Cổ", "3", "Wood Path", "Healing", "40.0", '{"active":{"healing":100.0,"regen_hp":20.0}}', "3", '["wood"]', "100", "Khôi phục sinh lực Mộc đạo", "Eastern_Sea", "UNCOMMON", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]'],
    ["GU_BLOOD_REGENERATION", "Huyết Tái Sinh Cổ", "4", "Blood Path", "Healing", "60.0", '{"active":{"regen_hp":50.0,"backlash":10.0}}', "6", '["blood"]', "100", "Hồi phục liên tục nhưng tốn máu", "Central_Continent", "RARE", "false", "HEALER", '["HEALER"]', '["healing","blood"]', '["healing"]'],
    ["GU_LIFE_RETURNING", "Hồi Sinh Khí Cổ", "5", "Wood Path", "Healing", "100.0", '{"active":{"healing":500.0}}', "8", '["life_water"]', "100", "Trị liệu cường đại", "Eastern_Sea", "EPIC", "false", "HEALER", '["HEALER"]', '["healing"]', '["healing"]']
]

# Items: id,name,item_type,description,rarity,rank,path,effects,tags
new_items = [
    ["ITEM_HEALING_PELLET_SMALL", "Tiểu Hồi Huyết Đan", "Consumable", "Đan dược cấp thấp hồi phục sinh lực.", "COMMON", "1", "Human Path", '{"restore_hp": 30.0}', '["healing","pill"]'],
    ["ITEM_HEALING_PELLET_MEDIUM", "Trung Hồi Huyết Đan", "Consumable", "Đan dược cấp trung hồi phục sinh lực.", "UNCOMMON", "2", "Human Path", '{"restore_hp": 80.0}', '["healing","pill"]'],
    ["ITEM_HEALING_PELLET_LARGE", "Đại Hồi Huyết Đan", "Consumable", "Đan dược cấp cao hồi phục sinh lực.", "RARE", "3", "Human Path", '{"restore_hp": 200.0}', '["healing","pill"]'],
    ["ITEM_LIFE_SAVING_PILL", "Cứu Mệnh Đan", "Consumable", "Đan dược cứu nguy khi trọng thương.", "EPIC", "4", "Human Path", '{"restore_hp": 500.0}', '["healing","pill"]'],
    ["ITEM_IMMORTAL_RECOVERY_PILL", "Tiên Nguyên Hồi Sinh Đan", "Consumable", "Tiên đan chữa thương cấp bậc Cổ Tiên.", "LEGENDARY", "6", "Human Path", '{"restore_hp": 1000.0,"heal_aperture":100.0}', '["healing","pill","immortal"]'],
    ["ITEM_MEDICINAL_PASTE", "Kim Sang Dược", "Consumable", "Thuốc mỡ trị thương ngoài chiến đấu.", "COMMON", "1", "Human Path", '{"restore_hp": 50.0}', '["healing","paste"]'],
    ["ITEM_HERBAL_SOUP", "Linh Thảo Thang", "Consumable", "Súp thảo mộc thanh tâm, dưỡng thần.", "COMMON", "1", "Human Path", '{"restore_hp": 40.0,"restore_mana":20.0}', '["healing","soup"]'],
    ["ITEM_BLOOD_GINSENG", "Huyết Sâm", "Consumable", "Dược liệu quý hiếm hồi máu số lượng lớn.", "UNCOMMON", "2", "Wood Path", '{"restore_hp": 150.0}', '["healing","herb"]'],
    ["ITEM_VITALITY_DEW", "Sinh Cơ Linh Lộ", "Consumable", "Dược dịch hồi phục liên tục theo thời gian.", "RARE", "3", "Wood Path", '{"regen_hp_out_of_combat": 300.0}', '["healing","liquid"]'],
    ["ITEM_IMMORTAL_SPRING_WATER", "Tiên Tuyền Thủy", "Consumable", "Nước suối tiên khôi phục cả đạo ngân và tiên khiếu.", "LEGENDARY", "6", "Water Path", '{"restore_hp": 500.0,"heal_aperture":500.0}', '["healing","liquid","immortal"]']
]

with open(gu_file, 'a', newline='', encoding='utf-8') as f:
    writer = csv.writer(f)
    for row in new_gus:
        writer.writerow(row)

with open(item_file, 'a', newline='', encoding='utf-8') as f:
    writer = csv.writer(f)
    for row in new_items:
        writer.writerow(row)
print("Added healing items and Gus")
