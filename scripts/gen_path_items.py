#!/usr/bin/env python3
"""
gen_path_items.py
Tạo và chuẩn hóa hệ thống 792 Items thuộc 44 Đạo (396 Dải A + 396 Dải B).
Ghép vào backend/data/Item.csv đồng thời giữ lại các Đan Dược và Tiền tệ thiết yếu.
"""

import csv
import json
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "backend" / "data"

RANK_PREFIXES = {
    1: "Phàm",
    2: "Linh",
    3: "Huyền",
    4: "Địa",
    5: "Thiên",
    6: "Tiên",
    7: "Thần",
    8: "Cổ",
    9: "Thái Cổ"
}

PRICES_A = {1: 100, 2: 300, 3: 800, 4: 2000, 5: 5000, 6: 15000, 7: 40000, 8: 100000, 9: 300000}
PRICES_B = {1: 120, 2: 350, 3: 900, 4: 2400, 5: 6000, 6: 18000, 7: 50000, 8: 120000, 9: 350000}

DAI_A_DATA = [
    ("Blade Path", "blade_stone", "Đao Phách Thạch", "Khối quặng ngưng tụ phong mang đao ý"),
    ("Blood Path", "blood_crystal", "Huyết Tinh", "Tinh thạch ngưng tụ từ tinh huyết chi khí dồi dào"),
    ("Bone Path", "bone_marrow", "Cốt Tủy", "Linh tủy xương cốt ẩn chứa ngạo cốt chi lực"),
    ("Cloud Path", "cloud_crystal", "Vân Tinh", "Mây mù ngưng kết thành tinh thạch phiêu dật"),
    ("Creation Path", "creation_stone", "Tạo Hóa Thạch", "Thần thạch dựng dục sinh cơ tạo hóa thiên địa"),
    ("Dark Path", "dark_crystal", "Ám Tinh", "Tinh thể bóng tối thôn phệ mọi quang mang chiếu rọi"),
    ("Death Path", "death_sand", "Tử Tịch Sa", "Hạt cát ngưng tụ tử khí tịch diệt của vạn vật"),
    ("Destruction Path", "destruction_iron", "Hủy Diệt Thiết", "Hắc thiết chứa sức mạnh phá hủy vạn cổ chi vật"),
    ("Dream Path", "dream_pearl", "Mộng Cảnh Châu", "Bảo châu lưu chuyển quang mang huyền ảo như thực như mộng"),
    ("Earth Path", "earth_soil", "Địa Mạch Thổ", "Khối bùn đất ngưng tụ địa mạch linh khí thâm hậu"),
    ("Emotion Path", "emotion_silk", "Tình Tơ", "Sợi tơ đan dệt từ thất tình lục dục thế gian"),
    ("Enslavement Path", "enslavement_stone", "Nô Dịch Thạch", "Bảo thạch phát ra dao động thuần phục tâm trí vạn thú"),
    ("Fire Path", "fire_crystal", "Hỏa Tinh", "Tinh thạch hỏa diễm thiêu đốt linh khí xung quanh"),
    ("Food Path", "food_essence", "Linh Lương Tinh", "Tinh hoa thực đạo mang lại no đủ và dồi dào sinh mệnh"),
    ("Formation Path", "formation_stone", "Trận Cơ Thạch", "Đá trận cơ khắc họa trận văn thiên địa tự nhiên"),
    ("Heaven Path", "heaven_sand", "Thiên Đạo Sa", "Cát thiên đạo ẩn chứa quy tắc vận hành của thương khung"),
    ("Human Path", "human_essence", "Nhân Đạo Tinh", "Tinh hoa nhân đạo ngưng tụ ý chí bất khuất của nhân tộc"),
    ("Ice Path", "ice_crystal", "Hàn Băng Tinh", "Khối băng tinh vạn năm tản ra hàn khí thấu xương"),
    ("Information Path", "information_stone", "Tín Tức Thạch", "Linh thạch ghi khắc và truyền lưu tin tức huyền bí"),
    ("Investigation Path", "investigation_pearl", "Thám Tra Châu", "Minh châu soi thấu sương mù và ngụy trang vạn dặm"),
    ("Killing Path", "killing_iron", "Sát Lục Thiết", "Huyết thiết đẫm sát khí ngút trời khiến người run sợ"),
    ("Light Path", "light_crystal", "Quang Diệu Tinh", "Pha lê phát ra ánh sáng thuần khiết soi rọi tà ma"),
    ("Lightning Path", "lightning_crystal", "Lôi Đình Tinh", "Tinh thể chứa đựng lôi đình cuồng bạo thiên kiếp"),
    ("Luck Path", "luck_pearl", "Khí Vận Châu", "Bảo châu phát ra khí vận hào quang tăng cường phúc duyên"),
    ("Metal Path", "metal_ore", "Canh Kim Quặng", "Quặng kim loại cứng rắn sắc bén vô song"),
    ("Painting Path", "painting_ink", "Đan Thanh Mặc", "Mực thần kỳ vẽ nên cảnh vật hư ảo hóa thành chân thực"),
    ("Phantom Path", "phantom_crystal", "Huyễn Ảnh Tinh", "Tinh thạch biến ảo hư hư thực thực khó phân biệt"),
    ("Poison Path", "poison_sand", "Độc Sa", "Cát độc mang kịch tính ăn mòn huyết nhục và linh hồn"),
    ("Qi Path", "qi_stone", "Nguyên Khí Thạch", "Thạch bảo tụ tập thiên địa nguyên khí tinh thuần vô cùng"),
    ("Refinement Path", "refinement_iron", "Luyện Đạo Thiết", "Kim loại hỗ trợ cô đọng và hợp luyện vạn vật"),
    ("Rule Path", "rule_stone", "Quy Tắc Thạch", "Khối đá ẩn chứa trật tự và quy củ bất biến"),
    ("Soul Path", "soul_sand", "Hồn Phách Sa", "Cát linh hồn tẩm nhuần hồn lực tinh túy"),
    ("Sound Path", "sound_resonance", "Âm Luật Thạch", "Bảo thạch cộng hưởng vang lên âm luật thiên địa kỳ diệu"),
    ("Space Path", "space_crystal", "Không Gian Tinh", "Tinh thể vặn vẹo không gian chứa dung lượng vô tận"),
    ("Star Path", "star_sand", "Tinh Thần Sa", "Cát sao lấp lánh tinh quang rực rỡ trời đêm"),
    ("Strength Path", "strength_crystal", "Lực Đạo Tinh", "Tinh thể mang sức nặng ngàn cân bộc phát cự lực"),
    ("Sword Path", "sword_stone", "Kiếm Ý Thạch", "Thạch khối ngưng kết kiếm khí sắc bén xé toạc hư không"),
    ("Theft Path", "theft_crystal", "Thâu Thiên Tinh", "Tinh thạch huyền diệu có khả năng che giấu và dịch chuyển vật phẩm"),
    ("Time Path", "time_sand", "Tuế Nguyệt Sa", "Hạt cát lưu chuyển dòng thời gian vội vã"),
    ("Transformation Path", "transformation_essence", "Biến Hóa Tinh", "Tinh chất thiên biến vạn hóa thích ứng mọi hình hài"),
    ("Water Path", "water_pearl", "Huyền Thủy Châu", "Châu ngọc tích tụ linh thủy uốn lượn bất tận"),
    ("Wind Path", "wind_stone", "Cương Phong Thạch", "Phiến đá ngưng đọng ngọn gió bão xé gió gầm rú"),
    ("Wisdom Path", "wisdom_pearl", "Trí Tuệ Châu", "Minh châu khai mở thần trí, suy diễn thiên cơ"),
    ("Wood Path", "wood_crystal", "Ất Mộc Tinh", "Tinh chất cây ngàn năm đượm nhuần sinh mệnh chi lực")
]

DAI_B_DATA = [
    ("Blade Path", "asura_blade_bone", "Tu La Đao Cốt", "Mảnh xương lưỡi đao sắc bén rơi từ Tu La chiến trường"),
    ("Blood Path", "blood_bat_wing", "Huyết Bức Dực", "Đôi cánh dơi hút máu ngưng tụ huyết khí nồng đậm"),
    ("Bone Path", "bone_splinter", "Hài Cốt Toái Phiến", "Mảnh hài cốt vạn năm bất hoại của cốt thú"),
    ("Cloud Path", "flying_fox_pelt", "Phi Hồ Bì", "Bộ lông hồ ly lượn mây nhẹ như tơ tằm"),
    ("Creation Path", "spirit_guardian_clay", "Sơn Thần Linh Nê", "Mảnh đất sét sinh linh mang năng lượng tạo hóa"),
    ("Dark Path", "dark_python_hide", "Hắc Mãng Xà Bì", "Lớp da rắn đen ngâm trong vực sâu tăm tối"),
    ("Death Path", "death_remnant", "Tử Sa Thi Hài", "Di hài tàn lụi tịch diệt của dị thú sa mạc"),
    ("Destruction Path", "destruction_beast_horn", "Hủy Diệt Thú Giác", "Cặp sừng sắc nhọn mang sức mạnh đập tan vạn vật"),
    ("Dream Path", "dream_butterfly_dust", "Mộng Điệp Dực Phấn", "Bột phấn lấp lánh rơi từ cánh bướm mộng cảnh"),
    ("Earth Path", "centipede_carapace", "Ngô Công Giáp Xác", "Lớp giáp lưng cứng cáp của rết khổng lồ lòng đất"),
    ("Emotion Path", "emotion_beast_horn", "Tình Thú Giác", "Sừng dị thú ngưng đọng chấp niệm và tình cảm"),
    ("Enslavement Path", "wolf_king_fang", "Lang Vương Răng Nanh", "Chiếc nanh sắc nhọn biểu tượng thống lĩnh đàn sói"),
    ("Fire Path", "fire_rat_fur", "Ly Hỏa Viêm Mao", "Túm lông chuột lửa không bao giờ tàn lụi"),
    ("Food Path", "gluttonous_stomach_sac", "Thao Thiết Vị Nang", "Túi dạ dày dị thú có thể tiêu hóa vạn vật"),
    ("Formation Path", "formation_herb_root", "Trận Linh Thảo Căn", "Rễ cây linh thảo hấp thụ năng lượng trận pháp"),
    ("Heaven Path", "heaven_tribulation_shard", "Thiên Kiếp Toái Phiến", "Mảnh tinh thể kết tinh sau khi thiên kiếp lôi đình giáng xuống"),
    ("Human Path", "cultivator_blood_essence", "Tán Tu Tinh Huyết", "Giọt tinh huyết ngưng tụ công pháp tu luyện nhân đạo"),
    ("Ice Path", "ice_cicada_wing", "Hàn Băng Thiền Dực", "Cánh ve sầu trong suốt tỏa ra khí lạnh thấu xương"),
    ("Information Path", "rabbit_spirit_ear", "Dược Thố Linh Nhĩ", "Đôi tai thỏ linh mẫn nghe ngóng tin tức trăm dặm"),
    ("Investigation Path", "hawk_investigation_eye", "Thiên Lý Ưng Đồng", "Mắt chim ưng tinh tường nhìn thấu ngụy trang"),
    ("Killing Path", "killing_blood_blade", "Sát Lục Huyết Nhận", "Lưỡi đao sát khí nhuốm máu kẻ thù của hung thú"),
    ("Light Path", "luminescent_beast_horn", "Huỳnh Thú Quang Giác", "Chiếc sừng phát quang soi tỏ bóng đêm"),
    ("Lightning Path", "lightning_wolf_pelt", "Điện Lang Lôi Bì", "Bộ da sói sét tích tụ dòng điện cuồng bạo"),
    ("Luck Path", "luck_dog_golden_fur", "Vận Khuyển Kim Mao", "Sợi lông vàng óng của thần khuyển mang lại vận may"),
    ("Metal Path", "iron_bear_paw", "Thiết Hùng Hùng Chưởng", "Móng vuốt gấu sắt đập nát đá tảng"),
    ("Painting Path", "painting_butterfly_scale", "Mặc Điệp Thải Lân", "Vảy bướm đan thanh vẽ nên cảnh tượng thần kỳ"),
    ("Phantom Path", "phantom_beast_tail", "Hư Ảnh Thú Vĩ", "Cái đuôi dị thú thoắt ẩn thoắt hiện"),
    ("Poison Path", "poison_sac", "Độc Mãng Nọc Nang", "Túi nọc kịch độc của độc xà ngàn năm"),
    ("Qi Path", "qi_dragon_scale", "Khí Long Nghịch Lân", "Vảy rồng nguyên khí bay lượn trên chín tầng mây"),
    ("Refinement Path", "langya_refinement_shard", "Lang Gia Luyện Tinh", "Mảnh tàn phiến chứa đựng bí pháp luyện đạo"),
    ("Rule Path", "rule_beast_horn", "Luật Lệnh Thú Giác", "Chiếc sừng chấp pháp uy nghiêm bất khả xâm phạm"),
    ("Soul Path", "soul_devourer_stinger", "Phệ Hồn Trùng Châm", "Ngòi châm chuyên thôn phệ và đâm thủng hồn phách"),
    ("Sound Path", "roar_monkey_throat_bone", "Ma Hầu Hống Cốt", "Xương thanh quản vượn ma phát ra tiếng hú xé trời"),
    ("Space Path", "void_beast_hide", "Hư Không Thú Bì", "Tấm da dã thú có thể hòa tan vào vết nứt không gian"),
    ("Star Path", "star_beast_scale", "Tinh Thú Lân Phiến", "Lớp vảy ngưng đọng tinh tú trời đêm"),
    ("Strength Path", "wild_boar_tusk", "Dã Trư Liêu Nha", "Cặp răng nanh lợn rừng cắm sâu vào đất đá"),
    ("Sword Path", "sword_rock_scale", "Kiếm Ý Thú Lân", "Vảy đá sắc nhọn như kiếm của dị thú núi đá"),
    ("Theft Path", "theft_rat_claw", "Thâu Thử Miêu Trảo", "Móng vuốt chuột trộm đồ vật không phát ra tiếng động"),
    ("Time Path", "time_toad_sand", "Tuế Nguyệt Thiềm Sa", "Hạt cát đọng trên lưng cóc thời gian"),
    ("Transformation Path", "polymorph_beast_skin", "Biến Hóa Thú Bì", "Lớp da có thể tự do thay đổi hình dáng và hoa văn"),
    ("Water Path", "crocodile_dragon_scale", "Bích Ba Ngạc Lân", "Vảy cá sấu xanh bơi lặn ngàn trượng đáy nước"),
    ("Wind Path", "storm_eagle_feather", "Phong Bão Điêu Linh", "Lông vũ chim điêu vút bay qua cuồng phong bão táp"),
    ("Wisdom Path", "fox_wisdom_brain", "Linh Hồ Tuệ Tủy", "Tủy não hồ ly thông thái tích tụ linh trí"),
    ("Wood Path", "rooster_herb_crest", "Linh Kê Thảo Mào", "Chiếc mào đỏ tươi như hoa của linh kê rừng rậm")
]

KEPT_SPECIAL_IDS = {
    'arena_coin', 'mine_chest',
    'ITEM_HEALING_PELLET_SMALL', 'ITEM_MEDICINAL_PASTE', 'ITEM_HEALING_PELLET_MEDIUM',
    'ITEM_BLOOD_GINSENG', 'ITEM_HEALING_PELLET_LARGE', 'ITEM_LIFE_SAVING_PILL',
    'ITEM_IMMORTAL_RECOVERY_PILL', 'ITEM_IMMORTAL_SPRING_WATER', 'ITEM_RANDOM_CONSTITUTION',
    'HUMAN_QI_PILL', 'ITEM_RANDOM_RACE', 'ITEM_APERTURE_EXPANSION_PILL', 'ITEM_APERTURE_TIME_PILL'
}

def generate_all_items():
    new_items = []
    
    # 1. Dải A
    for path_name, base_id, base_vn, desc_base in DAI_A_DATA:
        for rk in range(1, 10):
            prefix = RANK_PREFIXES[rk]
            item_id = f"{base_id}_rank{rk}"
            name = f"{prefix} {base_vn}"
            desc = f"{desc_base}, phẩm cấp {prefix} cấp."
            new_items.append({
                "id": item_id,
                "name": name,
                "item_type": "Material",
                "rank": rk,
                "effects": "[]",
                "description": desc,
                "base_price": f"{PRICES_A[rk]}.000",
                "image_url": "",
                "rarity": 0,
                "path": json.dumps([path_name])
            })
            
    # 2. Dải B
    for path_name, base_id, base_vn, desc_base in DAI_B_DATA:
        for rk in range(1, 10):
            prefix = RANK_PREFIXES[rk]
            item_id = f"{base_id}_rank{rk}"
            name = f"{prefix} {base_vn}"
            desc = f"{desc_base}, phẩm cấp {prefix} cấp."
            new_items.append({
                "id": item_id,
                "name": name,
                "item_type": "Material",
                "rank": rk,
                "effects": "[]",
                "description": desc,
                "base_price": f"{PRICES_B[rk]}.000",
                "image_url": "",
                "rarity": 0,
                "path": json.dumps([path_name])
            })
            
    return new_items

def main():
    item_file = DATA_DIR / "Item.csv"
    existing_rows = []
    fieldnames = []
    
    if item_file.exists():
        with open(item_file, "r", encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            fieldnames = reader.fieldnames
            existing_rows = list(reader)
            
    # Preserve kept special items (pills / currency)
    special_rows = [r for r in existing_rows if r["id"] in KEPT_SPECIAL_IDS]
    
    new_path_items = generate_all_items()
    all_combined = special_rows + new_path_items
    
    # Deduplicate while preserving order
    seen = set()
    final_rows = []
    for r in all_combined:
        if r["id"] not in seen:
            seen.add(r["id"])
            final_rows.append(r)
            
    with open(item_file, "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(final_rows)
        
    print(f"gen_path_items: Đã tạo và ghi {len(final_rows)} items vào {item_file}!")

if __name__ == "__main__":
    main()
