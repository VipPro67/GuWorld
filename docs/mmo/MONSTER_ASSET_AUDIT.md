# Monster asset audit

Repository snapshot: `mmo/realtime-foundation`, 2026-10-25. Source of truth for this audit is `backend/data/Monster.csv`; live database contents can differ until Admin imports are applied. All 507 CSV templates are active.

## Summary

| Measure | Count |
|---|---:|
| Backend monster templates | 507 |
| Templates mapped to an existing atlas frame | 507 |
| Templates without unique atlas frame | 0 |
| Templates referenced by active `MonsterSpawn.csv` rows | 471 |

The manifest has no duplicate or unknown monster IDs, and every mapped atlas file exists. Backend templates have no image field. Phaser receives `template_id` in realtime entity snapshots, uses `frontend/src/game/assets/monsterAtlasManifest.ts` to select an atlas frame, and loads that sheet when the monster appears. The generated `token-monster` remains a fallback while art loads or if loading fails.

The atlas path in the table is relative to `frontend/public`. `R` and `C` are one-based frame row and column. `Spawns` counts active CSV rows, not live instances.

## Complete catalog

| ID | Name | Type | Spawns | Asset | Frame |
|---|---|---|---:|---|---|
| `boss_wolf_king` | Vạn Thú Cuồng Lang Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R1 C1 |
| `monster_ancient_blade_spirit_asura` | Thượng Cổ Đao Phách Tu La | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R1 C2 |
| `monster_boar_king` | Dã Trư Vương | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R1 C3 |
| `monster_bone_general` | Bạch Cốt Tướng Quân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R1 C4 |
| `monster_cave_centipede` | Địa Huyệt Ngô Công | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R2 C1 |
| `monster_crescent_crocodile` | Nguyệt Nha Bích Ba Ngạc | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R2 C2 |
| `monster_dark_swamp_python` | Hắc Thủy U Mãng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R2 C3 |
| `monster_desert_ghost_boss` | Sa Mạc Quỷ Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R2 C4 |
| `monster_stone_ape` | Thạch Hầu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R3 C1 |
| `monster_earth_trench_centipede` | Địa Câu Thâm Ngô | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R3 C2 |
| `monster_flying_fox` | Thiên Đề Phi Hồ | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R3 C3 |
| `monster_lava_python` | Mãng Xà Dung Nham | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R3 C4 |
| `monster_sand_wraith` | Thanh Quỷ Sa | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R4 C1 |
| `monster_three_kings_boss` | Tam Vương Khôi Lỗi | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R4 C2 |
| `monster_shadow_beast` | Ám Ảnh Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R4 C3 |
| `monster_wolf_leader` | Bách Thú Độc Lang Vương | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-01.webp` | R4 C4 |
| `monster_ghost_scorpion` | Quỷ Hạt Tử | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R1 C1 |
| `monster_lightning_wolf` | Điện Lang | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R1 C2 |
| `monster_fire_refine_beast` | Luyện Diễm Linh Điểu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R1 C3 |
| `monster_jade_golem` | Ngọc Thạch Cự Nhân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R1 C4 |
| `monster_night_wolf` | Dạ Lang Thảo Nguyên | Normal | 1 | `/assets/world/monster-atlas-common.png` | R1 C3 |
| `monster_soul_devourer` | Phệ Hồn Trùng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R2 C1 |
| `monster_linh_thao_ke` | Linh Thảo Kê | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R2 C2 |
| `monster_doc_ngoc_thu` | Độc Ngọc Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R2 C3 |
| `monster_blood_bat` | Huyết Dực Ma Bức | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R2 C4 |
| `monster_blood_lake_python` | Tử Sa Huyết Mãng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R3 C1 |
| `monster_ghost_bat` | Quỷ Dơi | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R3 C2 |
| `monster_thai_co_tinh_linh` | Thái Cổ Tinh Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R3 C3 |
| `monster_baby_blood_spider` | Huyết Nhện Con | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R3 C4 |
| `monster_blood_hound` | Huyết Liệp Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R4 C1 |
| `monster_clay_boar` | Bản Trư | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R4 C2 |
| `monster_dao_spirit` | Đạo Linh Hư Ảnh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R4 C3 |
| `monster_evil_cultivator` | Ma Đạo Cổ Sư Tán Tu | HumanCultivator | 2 | `/assets/world/monsters/monster-atlas-batch-02.webp` | R4 C4 |
| `monster_featherman_scout` | Vũ Nhân Trinh Sát | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R1 C1 |
| `monster_hu_tien_fox` | Hồ Tiên Biến Hóa Phi Hồ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R1 C2 |
| `monster_snow_yeti` | Tuyết Nguyên Nhân Tuyết Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R1 C3 |
| `monster_sound_golem` | Âm Ba Khôi Lỗi | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R1 C4 |
| `monster_stone_man` | Thạch Nhân Thổ Cư | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R2 C1 |
| `monster_venom_centipede` | Bách Độc Ngô Công | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R2 C2 |
| `monster_wan_xa_poison_snake` | Vạn Xà Cốc Độc Mãng | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R2 C3 |
| `monster_wind_bee` | Phong Bạo Độc Phong | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R2 C4 |
| `monster_sword_intent_golem` | Kiếm Ý Tàn Linh Boss | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R3 C1 |
| `monster_sword_sakura` | Kiếm Ý Anh Đào | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R3 C2 |
| `monster_vampire_bat_king` | Vạn Bức Ma Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R3 C3 |
| `monster_dark_squid` | Ám Ảnh Hải Khôi Ô Tặc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R3 C4 |
| `monster_east_sea_dragonfish` | Đông Hải Thượng Cổ Long Ngư | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R4 C1 |
| `monster_quicksand_worm` | Hoàng Sa Khí Giáp Hạt | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R4 C2 |
| `monster_iron_bear` | Thiết Giáp Hùng | Elite | 1 | `/assets/world/monster-atlas-common.png` | R4 C4 |
| `monster_herb_rabbit` | Dược Dã Thố | Normal | 3 | `/assets/world/monster-atlas-common.png` | R1 C4 |
| `monster_sword_bird` | Trảm Kiếm Điểu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R4 C3 |
| `monster_grand_calamity_boss` | Hạo Kiếp Thần Ma | Tribulation_Boss | 0 | `/assets/world/monsters/monster-atlas-batch-03.webp` | R4 C4 |
| `monster_luck_hound` | Chân Dương Vận Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R1 C1 |
| `monster_starlight_beast` | Tinh Quang Huỳnh Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R1 C2 |
| `monster_wild_boar` | Dã Trư | Normal | 3 | `/assets/world/monster-atlas-common.png` | R2 C2 |
| `monster_heaven_tribulation_boss` | Thiên Kiếp Thần Ma | Tribulation_Boss | 0 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R1 C3 |
| `monster_shadow_glimpse_boss` | Lược Ảnh Ma Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R1 C4 |
| `monster_desert_ghost_wraith` | Âm Hồn Nhai Quỷ Linh | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R2 C1 |
| `monster_blood_spider` | Xích Ti Huyết Chu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R2 C2 |
| `monster_bone_dog` | Cốt Thương Dã Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R2 C3 |
| `monster_night_mist` | U Dạ Sương | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R2 C4 |
| `monster_spirit_deer` | Linh Lộc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R3 C1 |
| `monster_spirit_guardian` | Thủ Vệ Sơn Thần | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R3 C2 |
| `monster_fire_mouse` | Ly Hỏa Thử | Normal | 3 | `/assets/world/monster-atlas-common.png` | R3 C3 |
| `monster_baby_bear` | Ấu Hùng | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R3 C3 |
| `monster_abyssal_ghost_scorpion` | U Minh Thần Hạt | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R3 C4 |
| `monster_ancient_blood_golem` | Cổ Huyết Nham Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R4 C1 |
| `monster_blood_sand_scorpion` | Huyết Sa Hoàng Kim Hạt | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R4 C2 |
| `monster_sound_spirit_beast` | Âm Ba Linh Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R4 C3 |
| `monster_year_beast_tiger` | Thái Cổ Dần Hổ Niên Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-04.webp` | R4 C4 |
| `monster_reverse_flow_spirit` | Nghịch Lưu Thủy Linh Boss | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R1 C1 |
| `monster_earth_calamity_boss` | Địa Tai Thần Ma | Tribulation_Boss | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R1 C2 |
| `monster_tribulation_beast` | Kiếp Lực Ma Vật | Tribulation_Monster | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R1 C3 |
| `monster_chaos_calamity_boss` | Hỗn Độn Thần Ma | Tribulation_Boss | 0 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R1 C4 |
| `monster_lightning_eel` | Lôi Điện Mạn Ngư | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R2 C1 |
| `monster_soul_gate_godfiend` | Sinh Tử Môn U Hồn Thần Ma | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R2 C2 |
| `monster_tide_dragon` | Thái Cổ Triều Long | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R2 C3 |
| `monster_white_hair_wolf` | Bạch Phát Lang | Normal | 1 | `/assets/world/monster-atlas-common.png` | R3 C1 |
| `monster_water_serpent` | Hải Xà | Normal | 2 | `/assets/world/monster-atlas-common.png` | R3 C4 |
| `monster_flower_butterfly` | Linh Hoa Điệp | Normal | 3 | `/assets/world/monster-atlas-common.png` | R2 C1 |
| `monster_sea_beast` | Loạn Lưu Hải Thú | Normal | 2 | `/assets/world/monster-atlas-common.png` | R4 C1 |
| `monster_desolate_cave_centipede` | Hoang Thú Địa Huyệt Ngô Công | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R2 C4 |
| `monster_deep_sea_horror` | Thâm Hải Cự Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R3 C1 |
| `monster_earth_elemental` | Thổ Nguyên Tinh | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R3 C2 |
| `monster_treasure_rat` | Kim Ngân Thử | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R3 C3 |
| `monster_koi_fish` | Cẩm Lý Ngư | Normal | 3 | `/assets/world/monster-atlas-common.png` | R1 C2 |
| `monster_immemorial_soul_beast` | Đãng Hồn Thái Cổ Hồn Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R3 C4 |
| `monster_wandering` | Cổ Sư Lang Thang | HumanCultivator | 1 | `/assets/world/monster-atlas-common.png` | R1 C1 |
| `monster_myriad_tribulation_boss` | Vạn Kiếp Thần Ma | Tribulation_Boss | 0 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R4 C1 |
| `monster_ancient_desert_ghost_king` | Hoang Thú Sa Mạc Quỷ Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R4 C2 |
| `monster_blade_beetle` | Thiết Nhận Giáp Trùng | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R4 C3 |
| `monster_bone_marrow_golem` | Thiên Cốt Nham Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-05.webp` | R4 C4 |
| `monster_meditation_stone_golem` | Ngộ Đạo Thạch Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R1 C1 |
| `monster_scholar_puppet` | Thư Đồng Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R1 C2 |
| `monster_chimera_beast` | Biến Huyễn Dị Thú Khôi | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R1 C3 |
| `monster_fire_refine_crab` | Luyện Hỏa Bàng Cua | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R1 C4 |
| `monster_frost_cicada` | Hàn Băng Thiền | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R2 C1 |
| `monster_cloud_dragon_beast` | Vân Long Hộ Vệ Thú | Elite | 1 | `/assets/world/monster-atlas-common.png` | R2 C3 |
| `monster_iron_bull` | Man Ngưu Thú Vương | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R2 C2 |
| `monster_thunder_cloud_eagle` | Lôi Vân Thần Ưng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R2 C3 |
| `monster_formation_spirit_grass` | Trận Linh Thảo | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R2 C4 |
| `monster_wine_monkey_king` | Hầu Nhi Tửu Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R3 C1 |
| `monster_desolate_luck_hound` | Hoang Thú Chân Dương Vận Khuyển | Normal | 2 | `/assets/world/monster-atlas-common.png` | R3 C2 |
| `monster_slaughter_pup` | Sát Lục Ấu Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R3 C2 |
| `monster_fortune_frost_beast` | Hàn Nguyệt Cát Tường Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R3 C3 |
| `monster_mystic_formation_general` | Huyền Trận Vạn Binh Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R3 C4 |
| `monster_nether_sand_wraith` | Sinh Tử Sa Ma | Elite | 1 | `/assets/world/monster-atlas-common.png` | R4 C3 |
| `monster_night_mist_lord` | U Dạ Sương Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R4 C1 |
| `monster_creation_earth_beetle` | Tạo Hóa Giáp Trùng | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R4 C2 |
| `monster_six_headed_serpent` | Lục Đầu Xà Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R4 C3 |
| `monster_immemorial_lightning_eel` | Thái Cổ Lôi Man Ngư | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-06.webp` | R4 C4 |
| `monster_desolate_lightning_tree` | Lôi Đình Hoang Thụ | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R1 C1 |
| `monster_thief_monkey` | Đạo Chỉ Linh Hầu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R1 C2 |
| `monster_dream_butterfly` | Mộng Huyễn Hoa Điệp | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R1 C3 |
| `monster_ancient_divine_tree_spirit` | Thượng Cổ Thần Mộc Tinh | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R1 C4 |
| `monster_dream_soul_phantom` | Mộng Cảnh Huyễn Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R2 C1 |
| `monster_desolate_hairy_man_chieftain` | Mao Dân Hoang Thú Tù Trưởng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R2 C2 |
| `monster_dream_dao_supreme_avatar` | Mộng Cảnh Vô Thượng Pháp Thân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R2 C3 |
| `monster_desolate_iron_blade_bear` | Trảm Thiết Hoang Hùng | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R2 C4 |
| `monster_desolate_formation_golem` | Hộ Thể Hoang Thạch Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R3 C1 |
| `monster_blood_sword_asura_lord` | Huyết Kiếm Tu La Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R3 C2 |
| `monster_ancient_lava_python` | Thượng Cổ Dung Nham Mãng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R3 C3 |
| `monster_dan_qing_spirit_butterfly` | Đan Thanh Mặc Điệp | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R3 C4 |
| `monster_immemorial_sword_emperor` | Thái Cổ Kiếm Ma Chi Linh | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R4 C1 |
| `monster_nine_heavens_primordial_emperor` | Cửu Thiên Nguyên Thủy Thần Hoàng | Boss | 0 | `/assets/world/monster-atlas-common.png` | R2 C4 |
| `monster_mirror_turtle` | Bình Kính Minh Tưởng Quy | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R4 C2 |
| `monster_immemorial_dream_phoenix` | Thái Cổ Mộng Huyễn Thần Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R4 C3 |
| `monster_immemorial_dragon_turtle` | Thái Cổ Long Quy Thần Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-07.webp` | R4 C4 |
| `monster_primordial_qi_dragon` | Thái Cổ Khí Long Boss | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R1 C1 |
| `monster_river_of_time_lord` | Quang Âm Chi Chủ | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R1 C2 |
| `monster_ancient_flying_sword_dragon` | Thượng Cổ Phi Kiếm Long | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R1 C3 |
| `monster_immemorial_celestial_star_beast` | Thái Cổ Cửu Thiên Tinh Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R1 C4 |
| `monster_desolate_blood_fiend` | Sát Lục Hoang Thi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R2 C1 |
| `monster_desolate_fire_phoenix` | Liệt Diễm Hoang Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R2 C2 |
| `monster_immemorial_creation_ruin_golem` | Thái Cổ Tạo Hóa Diệt Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R2 C3 |
| `monster_origin_sea_immortal_sovereign` | Nguyên Hải Thủy Ma Tôn | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R2 C4 |
| `monster_world_tree_primordial_spirit` | Thế Giới Thụ Cổ Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R3 C1 |
| `monster_collapse_iron_beast` | Băng Hoại Hắc Giáp Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R3 C2 |
| `monster_colored_pigment_spirit` | Thạch Thanh Họa Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R3 C3 |
| `monster_creation_clay_puppet` | Tạo Hóa Thổ Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R3 C4 |
| `monster_creation_life_deer` | Tạo Hóa Tức Lộc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R4 C1 |
| `monster_creation_sprout_spirit` | Tạo Hóa Nha Linh | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R4 C2 |
| `monster_dan_qing_crane` | Đan Thanh Linh Hạc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R4 C3 |
| `monster_decay_bone_beetle` | Hủ Cốt Giáp Trùng | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-08.webp` | R4 C4 |
| `monster_destruction_ant` | Toái Diệt Hắc Nghĩ | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R1 C1 |
| `monster_dream_mist_fox` | Mộng Vụ Linh Hồ | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R1 C2 |
| `monster_fortune_flow_badger` | Lưu Vận Điềm Điêu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R1 C3 |
| `monster_glacial_bone_wolf` | Băng Cốt Tuyết Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R1 C4 |
| `monster_gourmet_toad` | Bách Vị Linh Cáp | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R2 C1 |
| `monster_heaven_law_cloud_beast` | Thiên Luật Vân Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R2 C2 |
| `monster_heaven_order_falcon` | Thiên Luật Thần Ưng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R2 C3 |
| `monster_heaven_sparrow` | Thiên Uy Linh Tước | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R2 C4 |
| `monster_meteor_firefly` | Lưu Tinh Huỳnh Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R3 C1 |
| `monster_mirage_clam` | Thận Lâu Trai | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R3 C2 |
| `monster_phantom_smoke_moth` | Huyễn Yên Nga | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R3 C3 |
| `monster_rule_marking_beetle` | Quy Củ Giáp Trùng | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R3 C4 |
| `monster_shadow_spider_cub` | Diệu Thủ Ấu Chu | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R4 C1 |
| `monster_shattering_shock_ore` | Chấn Toái Nham Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R4 C2 |
| `monster_tomb_death_specter` | U Mộ Tử Quỷ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R4 C3 |
| `monster_unseen_theft_rat` | Vô Hình Thâu Khí Thử | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-09.webp` | R4 C4 |
| `monster_withered_bone_wraith` | Khô Tịch Cốt Linh | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R1 C1 |
| `monster_wolf_brush_golem` | Lang Hào Bút Khôi Lỗi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R1 C2 |
| `monster_ancient_calamity_destroyer` | Thượng Cổ Diệt Thế Viêm Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R1 C3 |
| `monster_ancient_creation_womb_titan` | Thượng Cổ Tạo Hóa Thai Thạch Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R1 C4 |
| `monster_ancient_dream_lotus_spirit` | Thượng Cổ Mộng Liên Tinh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R2 C1 |
| `monster_ancient_formation_pillar_titan` | Thượng Cổ Trận Cột Cự Nhân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R2 C2 |
| `monster_ancient_luck_cloud_dragon` | Thượng Cổ Kim Vận Long | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R2 C3 |
| `monster_ancient_nether_reaper` | Thượng Cổ U Minh Cốt Ma | Elite | 1 | `/assets/world/monster-atlas-common.png` | R4 C2 |
| `monster_ancient_nine_nether_ice_dragon` | Thượng Cổ Cửu U Băng Long | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R2 C4 |
| `monster_ancient_taotie_devourer` | Thượng Cổ Thao Thiết Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R3 C1 |
| `monster_ancient_tribulation_emperor` | Thượng Cổ Thiên Kiếp Ma Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R3 C2 |
| `monster_ancient_white_gold_golem` | Thượng Cổ Bạch Kim Cự Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R3 C3 |
| `monster_desolate_creation_spring_turtle` | Tạo Hóa Tuyền Hoang Quy | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R3 C4 |
| `monster_desolate_death_hydra` | Tử Vực Hoang Mãng | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R4 C1 |
| `monster_desolate_destruction_rhino` | Toái Liệt Hoang Tê | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R4 C2 |
| `monster_desolate_dream_tapir` | Mộng Yểm Hoang Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R4 C3 |
| `monster_desolate_formless_phantom` | Vô Tướng Hoang Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-10.webp` | R4 C4 |
| `monster_desolate_frost_mammoth` | Cực Băng Hoang Tượng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R1 C1 |
| `monster_desolate_ink_carp` | Tiên Mặc Hoang Cẩm Lý | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R1 C2 |
| `monster_desolate_thief_ape` | Đạo Kiếp Hoang Hầu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R1 C3 |
| `monster_gluttonous_pig_emperor` | Bách Vị Trư Hoàng | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R1 C4 |
| `monster_phantom_chameleon` | Huyễn Sắc Thuẫn Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R2 C1 |
| `monster_star_river_serpent` | Tinh Hà Linh Xà | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R2 C2 |
| `monster_void_theft_badger` | Không Không Đạo Điêu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R2 C3 |
| `monster_ancient_mirage_dragon_clam` | Thượng Cổ Thận Long Trai | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R2 C4 |
| `monster_ancient_mountain_lifting_ape` | Thượng Cổ Bạt Sơn Cự Hầu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R3 C1 |
| `monster_ancient_star_sea_whale` | Thượng Cổ Tinh Hải Cự Kình | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R3 C2 |
| `monster_blade_beetle_commander` | Trảm Thiết Đao Tướng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R3 C3 |
| `monster_genesis_lotus_will_avatar` | Nguyên Liên Tiên Tôn Họa Ý Thần Thân | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R3 C4 |
| `monster_immemorial_heaven_will_avatar` | Thái Cổ Thiên Ý Hóa Thân | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R4 C1 |
| `monster_immemorial_human_ancestor_will` | Thái Cổ Nhân Tổ Tàn Niệm | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R4 C2 |
| `monster_immemorial_landscape_phoenix` | Thái Cổ Giang Sơn Họa Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R4 C3 |
| `monster_immemorial_law_ruler_spirit` | Thái Cổ Luật Lệnh Thần Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-11.webp` | R4 C4 |
| `monster_immemorial_myriad_jun_titan` | Thái Cổ Vạn Quân Thần Ma | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R1 C1 |
| `monster_kill_intent_puppet` | Sát Ý Thiết Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R1 C2 |
| `boss_dan_qing_scholar` | Đan Thanh Đại Họa Sư | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R1 C3 |
| `monster_ancient_cloud_canvas_dragon` | Thượng Cổ Vân Họa Long | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R1 C4 |
| `monster_ancient_heaven_stealing_dragon` | Thượng Cổ Thâu Thiên Bích Long | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R2 C1 |
| `monster_ancient_myriad_beast_king` | Thượng Cổ Vạn Thú Tôn Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R2 C2 |
| `monster_dream_bubble_spirit` | Mộng Cảnh Bọt Khí Linh | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R2 C3 |
| `monster_immemorial_chaos_ice_beast` | Thái Cổ Hỗn Độn Băng Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R2 C4 |
| `monster_immemorial_divine_feaster` | Thái Cổ Bách Vị Thần Ma | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R3 C1 |
| `monster_wine_monkey` | Tửu Cổ Hầu | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R3 C2 |
| `monster_refinement_ancestor_will` | Lang Gia Luyện Đạo Tàn Niệm | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R3 C3 |
| `monster_immemorial_refinement_golem` | Thái Cổ Luyện Kim Cự Thạch Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R3 C4 |
| `monster_nether_bone_dragon_lord` | U Minh Thái Cốt Long Vương | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R4 C1 |
| `monster_tide_dragon_emperor` | Vạn Sấm Cửu Cung Long Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R4 C2 |
| `monster_immemorial_cloud_phoenix` | Thái Cổ Bạch Vân Thần Phượng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R4 C3 |
| `monster_sky_palace_lord` | Cửu Thiên Vân Cung Chủ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-12.webp` | R4 C4 |
| `monster_white_light_divine_sovereign` | Bạch Quang Cửu Chuyển Tiên Tôn Hóa Thân | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-13.webp` | R1 C1 |
| `monster_silvermane_wolf` | Bạc Liêm Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R1 C1 |
| `monster_thunder_ape` | Lôi Đình Cự Viên | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R1 C2 |
| `monster_venom_scale_serpent` | Độc Lân Xà | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R1 C3 |
| `monster_ancient_mountain_golem` | Thái Cổ Sơn Khôi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R1 C4 |
| `monster_blood_moon_tiger` | Huyết Nguyệt Hổ | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R2 C1 |
| `monster_abyss_maw_beast` | Vực Uyên Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R2 C2 |
| `monster_primordial_dire_wolf` | Nguyên Sơ Cự Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R2 C3 |
| `monster_nine_heavens_qilin` | Cửu Tiêu Kỳ Lân | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R2 C4 |
| `monster_chaos_devouring_beast` | Thôn Thiên Yêu Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R3 C1 |
| `monster_elite_iron_tyrant` | Thiết Giáp Bá Vương | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R3 C2 |
| `monster_elite_storm_phoenix` | Bạo Phong Phượng Hoàng | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R3 C3 |
| `monster_elite_void_stalker` | Hư Không Tập Kích Giả | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R3 C4 |
| `monster_elite_heaven_sunderer` | Phá Thiên Tôn Giả | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R4 C1 |
| `monster_elite_abyssal_overlord` | Vực Sâu Bá Chủ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R4 C2 |
| `monster_elite_celestial_guardian` | Thiên Giới Thủ Hộ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R4 C3 |
| `monster_thousand_li_hawk` | Thiên Lý Ưng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-14.webp` | R4 C4 |
| `monster_white_fox_scholar` | Bạch Diện Linh Hồ | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R1 C1 |
| `monster_blade_thai_co_r8r9` | Thái Cổ Đao Phách | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R1 C2 |
| `monster_blood_di_bien_r4r5` | Dị Biến Huyết Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R1 C3 |
| `monster_blood_hoang_thu_r6r7` | Hoang Thú Huyết Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R1 C4 |
| `monster_blood_thai_co_r8r9` | Thái Cổ Huyết Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R2 C1 |
| `monster_bone_thai_co_r8r9` | Thái Cổ Cốt Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R2 C2 |
| `monster_cloud_hoang_da_r1r3` | Hoang Dã Vân Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R2 C3 |
| `monster_cloud_di_bien_r4r5` | Dị Biến Vân Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R2 C4 |
| `monster_cloud_hoang_thu_r6r7` | Hoang Thú Vân Thú | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R3 C1 |
| `monster_cloud_thai_co_r8r9` | Thái Cổ Vân Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R3 C2 |
| `monster_creation_hoang_da_r1r3` | Hoang Dã Tạo Hóa Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R3 C3 |
| `monster_creation_di_bien_r4r5` | Dị Biến Tạo Hóa Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R3 C4 |
| `monster_creation_hoang_thu_r6r7` | Hoang Thú Tạo Hóa Khôi Lỗi | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R4 C1 |
| `monster_creation_thai_co_r8r9` | Thái Cổ Tạo Hóa Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R4 C2 |
| `monster_dark_thai_co_r8r9` | Thái Cổ Ám Ảnh Dạ Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R4 C3 |
| `monster_dream_thai_co_r8r9` | Thái Cổ Mộng Cảnh Ảo Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-15.webp` | R4 C4 |
| `monster_emotion_thai_co_r8r9` | Thái Cổ Tình Ma Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R1 C1 |
| `monster_enslavement_thai_co_r8r9` | Thái Cổ Ngự Thú Lang Vương | Boss | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R1 C2 |
| `monster_fire_di_bien_r4r5` | Dị Biến Hỏa Diễm Viêm Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R1 C3 |
| `monster_fire_thai_co_r8r9` | Thái Cổ Hỏa Diễm Viêm Linh | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R1 C4 |
| `monster_food_thai_co_r8r9` | Thái Cổ Thao Thiết Phệ Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R2 C1 |
| `monster_formation_thai_co_r8r9` | Thái Cổ Trận Linh Thạch Nhân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R2 C2 |
| `monster_human_thai_co_r8r9` | Thái Cổ Nhân Đạo Khôi Lỗi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R2 C3 |
| `monster_ice_thai_co_r8r9` | Thái Cổ Hàn Băng Thần Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R2 C4 |
| `monster_information_thai_co_r8r9` | Thái Cổ Linh Thính Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R3 C1 |
| `monster_investigation_di_bien_r4r5` | Dị Biến Đồng Triệt Ưng Điểu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R3 C2 |
| `monster_investigation_hoang_thu_r6r7` | Hoang Thú Đồng Triệt Ưng Điểu | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R3 C3 |
| `monster_investigation_thai_co_r8r9` | Thái Cổ Đồng Triệt Ưng Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R3 C4 |
| `monster_killing_hoang_da_r1r3` | Hoang Dã Sát Lục Tu La | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R4 C1 |
| `monster_killing_hoang_thu_r6r7` | Hoang Thú Sát Lục Tu La | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R4 C2 |
| `monster_killing_thai_co_r8r9` | Thái Cổ Sát Lục Tu La | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R4 C3 |
| `monster_light_thai_co_r8r9` | Thái Cổ Quang Diệu Thần Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-16.webp` | R4 C4 |
| `monster_lightning_thai_co_r8r9` | Thái Cổ Lôi Đình Cuồng Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R1 C1 |
| `monster_luck_thai_co_r8r9` | Thái Cổ Lưu Vận Kim Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R1 C2 |
| `monster_metal_di_bien_r4r5` | Dị Biến Canh Kim Giáp Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R1 C3 |
| `monster_metal_hoang_thu_r6r7` | Hoang Thú Canh Kim Giáp Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R1 C4 |
| `monster_metal_thai_co_r8r9` | Thái Cổ Canh Kim Giáp Thú | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R2 C1 |
| `monster_painting_thai_co_r8r9` | Thái Cổ Đan Thanh Họa Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R2 C2 |
| `monster_phantom_hoang_da_r1r3` | Hoang Dã Huyễn Ảo Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R2 C3 |
| `monster_phantom_di_bien_r4r5` | Dị Biến Huyễn Ảo Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R2 C4 |
| `monster_phantom_hoang_thu_r6r7` | Hoang Thú Huyễn Ảo Dị Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R3 C1 |
| `monster_phantom_thai_co_r8r9` | Thái Cổ Huyễn Ảo Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R3 C2 |
| `monster_poison_thai_co_r8r9` | Thái Cổ Kịch Độc Ma Xà | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R3 C3 |
| `monster_qi_thai_co_r8r9` | Thái Cổ Nguyên Khí Cự Long | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R3 C4 |
| `monster_refinement_thai_co_r8r9` | Thái Cổ Luyện Đạo Hỏa Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R4 C1 |
| `monster_rule_thai_co_r8r9` | Thái Cổ Quy Tắc Chấp Pháp Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R4 C2 |
| `monster_soul_thai_co_r8r9` | Thái Cổ U Hồn Oán Linh | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R4 C3 |
| `monster_space_hoang_da_r1r3` | Hoang Dã Hư Không Di Dời Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-17.webp` | R4 C4 |
| `monster_space_di_bien_r4r5` | Dị Biến Hư Không Di Dời Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R1 C1 |
| `monster_space_hoang_thu_r6r7` | Hoang Thú Hư Không Di Dời Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R1 C2 |
| `monster_star_thai_co_r8r9` | Thái Cổ Tinh Thần Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R1 C3 |
| `monster_strength_thai_co_r8r9` | Thái Cổ Cự Lực Thần Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R1 C4 |
| `monster_sword_thai_co_r8r9` | Thái Cổ Kiếm Ý Linh Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R2 C1 |
| `monster_theft_di_bien_r4r5` | Dị Biến Thâu Thiên Thử Ma | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R2 C2 |
| `monster_theft_hoang_thu_r6r7` | Hoang Thú Thâu Thiên Thử Ma | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R2 C3 |
| `monster_theft_thai_co_r8r9` | Thái Cổ Thâu Thiên Thử Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R2 C4 |
| `monster_time_thai_co_r8r9` | Thái Cổ Tuế Nguyệt Thiềm Thừ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R3 C1 |
| `monster_transformation_hoang_da_r1r3` | Hoang Dã Thiên Biến Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R3 C2 |
| `monster_transformation_di_bien_r4r5` | Dị Biến Thiên Biến Dị Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R3 C3 |
| `monster_transformation_hoang_thu_r6r7` | Hoang Thú Thiên Biến Dị Thú | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R3 C4 |
| `monster_transformation_thai_co_r8r9` | Thái Cổ Thiên Biến Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R4 C1 |
| `monster_water_thai_co_r8r9` | Thái Cổ Bích Ba Long Ngạc | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R4 C2 |
| `monster_wind_thai_co_r8r9` | Thái Cổ Cương Phong Thần Ưng | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R4 C3 |
| `monster_wisdom_thai_co_r8r9` | Thái Cổ Thông Thiên Linh Hồ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-18.webp` | R4 C4 |
| `monster_wood_hoang_da_r1r3` | Hoang Dã Ất Mộc Thần Cây | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R1 C1 |
| `monster_wood_hoang_thu_r6r7` | Hoang Thú Ất Mộc Thần Cây | Boss | 0 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R1 C2 |
| `monster_wood_thai_co_r8r9` | Thái Cổ Ất Mộc Thần Cây | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R1 C3 |
| `monster_blade_tier1_normal` | Đao Tí Đường Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R1 C4 |
| `monster_blade_tier1_elite` | Tinh Anh Đao Tí Đường Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R2 C1 |
| `monster_blade_tier2_normal` | Đao Giác Tê Ngưu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R2 C2 |
| `monster_blade_tier2_elite` | Tinh Anh Đao Giác Tê Ngưu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R2 C3 |
| `monster_blade_tier3_normal` | Hoang Thú Đao Dực Ưng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R2 C4 |
| `monster_blade_tier3_elite` | Tinh Anh Hoang Thú Đao Dực Ưng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R3 C1 |
| `monster_blade_tier4_normal` | Thái Cổ Đao Ma Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R3 C2 |
| `monster_blade_tier4_elite` | Tinh Anh Thái Cổ Đao Ma Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R3 C3 |
| `monster_blood_tier2_normal` | Huyết Lang | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R3 C4 |
| `monster_blood_tier3_normal` | Hoang Thú Huyết Ngạc | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R4 C1 |
| `monster_blood_tier3_elite` | Tinh Anh Hoang Thú Huyết Ngạc | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R4 C2 |
| `monster_blood_tier4_normal` | Thái Cổ Huyết Ma | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R4 C3 |
| `monster_bone_tier2_normal` | Cốt Thương Hầu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-19.webp` | R4 C4 |
| `monster_bone_tier3_normal` | Hoang Thú Bạch Cốt Tượng | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R1 C1 |
| `monster_bone_tier3_elite` | Tinh Anh Hoang Thú Bạch Cốt Tượng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R1 C2 |
| `monster_bone_tier4_normal` | Thái Cổ Cốt Ma | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R1 C3 |
| `monster_cloud_tier2_normal` | Phi Vân Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R1 C4 |
| `monster_cloud_tier3_normal` | Hoang Thú Vân Vụ Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R2 C1 |
| `monster_cloud_tier3_elite` | Tinh Anh Hoang Thú Vân Vụ Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R2 C2 |
| `monster_cloud_tier4_normal` | Thái Cổ Vân Long | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R2 C3 |
| `monster_creation_tier1_normal` | Tạo Hóa Linh Lộc | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R2 C4 |
| `monster_creation_tier2_normal` | Sinh Cơ Khôi Lỗi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R3 C1 |
| `monster_creation_tier3_normal` | Hoang Thú Tạo Hóa Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R3 C2 |
| `monster_creation_tier3_elite` | Tinh Anh Hoang Thú Tạo Hóa Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R3 C3 |
| `monster_creation_tier4_normal` | Thái Cổ Tạo Hóa Cự Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R3 C4 |
| `monster_dark_tier1_elite` | Tinh Anh Ám Dạ Bức | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R4 C1 |
| `monster_dark_tier2_normal` | U Dạ Lang | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R4 C2 |
| `monster_dark_tier3_normal` | Hoang Thú Hắc Ám Cự Mãng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R4 C3 |
| `monster_dark_tier3_elite` | Tinh Anh Hoang Thú Hắc Ám Cự Mãng | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-20.webp` | R4 C4 |
| `monster_dark_tier4_normal` | Thái Cổ Ám Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R1 C1 |
| `monster_death_tier1_elite` | Tinh Anh Tử Khí Quạ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R1 C2 |
| `monster_death_tier2_normal` | Tử Tịch Hạt Tử | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R1 C3 |
| `monster_death_tier3_normal` | Hoang Thú Tử Sa Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R1 C4 |
| `monster_death_tier4_normal` | Thái Cổ Tịch Diệt Thần | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R2 C1 |
| `monster_death_tier4_elite` | Tinh Anh Thái Cổ Tịch Diệt Thần | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R2 C2 |
| `monster_destruction_tier1_normal` | Toái Diệt Hắc Nghĩ | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R2 C3 |
| `monster_destruction_tier1_elite` | Tinh Anh Toái Diệt Hắc Nghĩ | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R2 C4 |
| `monster_destruction_tier2_normal` | Phá Diệt Ma Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R3 C1 |
| `monster_destruction_tier2_elite` | Tinh Anh Phá Diệt Ma Khuyển | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R3 C2 |
| `monster_destruction_tier3_normal` | Hoang Thú Hủy Diệt Cự Hầu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R3 C3 |
| `monster_destruction_tier3_elite` | Tinh Anh Hoang Thú Hủy Diệt Cự Hầu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R3 C4 |
| `monster_destruction_tier4_normal` | Thái Cổ Hủy Diệt Cự Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R4 C1 |
| `monster_destruction_tier4_elite` | Tinh Anh Thái Cổ Hủy Diệt Cự Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R4 C2 |
| `monster_dream_tier1_normal` | Mộng Điệp | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R4 C3 |
| `monster_dream_tier1_elite` | Tinh Anh Mộng Điệp | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-21.webp` | R4 C4 |
| `monster_dream_tier2_elite` | Tinh Anh Mộng Huyễn Hoa Hồ | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R1 C1 |
| `monster_dream_tier4_normal` | Thái Cổ Mộng Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R1 C2 |
| `monster_earth_tier3_normal` | Hoang Thú Địa Mạch Khôi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R1 C3 |
| `monster_earth_tier4_normal` | Thái Cổ Sơn Khôi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R1 C4 |
| `monster_emotion_tier1_normal` | Thất Tình Tước | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R2 C1 |
| `monster_emotion_tier2_normal` | Tình Ti Yêu Tri Chu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R2 C2 |
| `monster_emotion_tier2_elite` | Tinh Anh Tình Ti Yêu Tri Chu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R2 C3 |
| `monster_emotion_tier3_normal` | Hoang Thú Tình Ý Linh Viên | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R2 C4 |
| `monster_enslavement_tier1_normal` | Ngự Thú Dã Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R3 C1 |
| `monster_enslavement_tier2_normal` | Thuần Phục Ngưu Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R3 C2 |
| `monster_enslavement_tier2_elite` | Tinh Anh Thuần Phục Ngưu Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R3 C3 |
| `monster_enslavement_tier3_normal` | Hoang Thú Vạn Thú Tướng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R3 C4 |
| `monster_enslavement_tier3_elite` | Tinh Anh Hoang Thú Vạn Thú Tướng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R4 C1 |
| `monster_enslavement_tier4_normal` | Thái Cổ Ngự Thú Lang | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R4 C2 |
| `monster_enslavement_tier4_elite` | Tinh Anh Thái Cổ Ngự Thú Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R4 C3 |
| `monster_fire_tier1_elite` | Tinh Anh Ly Hỏa Thử | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-22.webp` | R4 C4 |
| `monster_fire_tier2_normal` | Hỏa Diễm Tích Dịch | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R1 C1 |
| `monster_fire_tier4_normal` | Thái Cổ Viêm Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R1 C2 |
| `monster_food_tier1_normal` | Linh Lương Tham Trư | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R1 C3 |
| `monster_food_tier1_elite` | Tinh Anh Linh Lương Tham Trư | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R1 C4 |
| `monster_food_tier2_normal` | Thực Thôn Yến | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R2 C1 |
| `monster_food_tier2_elite` | Tinh Anh Thực Thôn Yến | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R2 C2 |
| `monster_food_tier3_normal` | Hoang Thú Phệ Lương Thiềm | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R2 C3 |
| `monster_food_tier3_elite` | Tinh Anh Hoang Thú Phệ Lương Thiềm | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R2 C4 |
| `monster_food_tier4_normal` | Thái Cổ Thao Thiết Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R3 C1 |
| `monster_formation_tier1_elite` | Tinh Anh Trận Cơ Thạch Nhân | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R3 C2 |
| `monster_formation_tier2_normal` | Huyền Trận Linh Quy | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R3 C3 |
| `monster_formation_tier3_normal` | Hoang Thú Bát Quái Cự Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R3 C4 |
| `monster_formation_tier4_normal` | Thái Cổ Trận Linh | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R4 C1 |
| `monster_heaven_tier2_normal` | Thiên Cương Bạch Viên | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R4 C2 |
| `monster_heaven_tier3_normal` | Hoang Thú Thiên Khí Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R4 C3 |
| `monster_heaven_tier4_normal` | Thái Cổ Thiên Đạo Khôi | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-23.webp` | R4 C4 |
| `monster_human_tier2_normal` | Tán Tu Tàn Hồn | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R1 C1 |
| `monster_human_tier2_elite` | Tinh Anh Tán Tu Tàn Hồn | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R1 C2 |
| `monster_human_tier3_normal` | Hoang Thú Nhân Đạo Dị Chủng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R1 C3 |
| `monster_human_tier4_normal` | Thái Cổ Nhân Hoàng Khôi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R1 C4 |
| `monster_ice_tier2_normal` | Băng Sương Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R2 C1 |
| `monster_ice_tier3_normal` | Hoang Thú Băng Phách Cự Hùng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R2 C2 |
| `monster_ice_tier4_normal` | Thái Cổ Hàn Long | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R2 C3 |
| `monster_information_tier2_normal` | Lưu Âm Thước Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R2 C4 |
| `monster_information_tier2_elite` | Tinh Anh Lưu Âm Thước Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R3 C1 |
| `monster_information_tier3_normal` | Hoang Thú Thiên Lý Độc Giác | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R3 C2 |
| `monster_information_tier4_normal` | Thái Cổ Tri Thức Dị Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R3 C3 |
| `monster_investigation_tier1_elite` | Tinh Anh Thiên Lý Ưng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R3 C4 |
| `monster_investigation_tier2_elite` | Tinh Anh Đồng Triệt Thần Báo | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R4 C1 |
| `monster_investigation_tier3_normal` | Hoang Thú Minh Mục Cự Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R4 C2 |
| `monster_investigation_tier3_elite` | Tinh Anh Hoang Thú Minh Mục Cự Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R4 C3 |
| `monster_investigation_tier4_normal` | Thái Cổ Thần Đồng Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-24.webp` | R4 C4 |
| `monster_killing_tier2_normal` | Tu La Sát Thứ | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R1 C1 |
| `monster_killing_tier2_elite` | Tinh Anh Tu La Sát Thứ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R1 C2 |
| `monster_killing_tier3_normal` | Hoang Thú Sát Lục Huyết Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R1 C3 |
| `monster_killing_tier3_elite` | Tinh Anh Hoang Thú Sát Lục Huyết Ma | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R1 C4 |
| `monster_killing_tier4_normal` | Thái Cổ Sát Lục Tu La | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R2 C1 |
| `monster_light_tier1_normal` | Quang Diệu Huỳnh Trùng | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R2 C2 |
| `monster_light_tier2_normal` | Huyễn Quang Bạch Lộc | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R2 C3 |
| `monster_light_tier2_elite` | Tinh Anh Huyễn Quang Bạch Lộc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R2 C4 |
| `monster_light_tier3_normal` | Hoang Thú Cực Quang Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R3 C1 |
| `monster_light_tier3_elite` | Tinh Anh Hoang Thú Cực Quang Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R3 C2 |
| `monster_light_tier4_normal` | Thái Cổ Quang Diệu Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R3 C3 |
| `monster_lightning_tier1_elite` | Tinh Anh Điện Lang | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R3 C4 |
| `monster_lightning_tier2_normal` | Lôi Đình Báo | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R4 C1 |
| `monster_lightning_tier3_normal` | Hoang Thú Lôi Điện Cự Mãng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R4 C2 |
| `monster_lightning_tier4_normal` | Thái Cổ Lôi Man Ngư | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R4 C3 |
| `monster_luck_tier1_elite` | Tinh Anh Cát Tường Linh Ngư | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-25.webp` | R4 C4 |
| `monster_luck_tier2_normal` | Chân Dương Vận Khuyển | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R1 C1 |
| `monster_luck_tier4_normal` | Thái Cổ Tụ Vận Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R1 C2 |
| `monster_metal_tier2_normal` | Canh Kim Tê Ngưu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R1 C3 |
| `monster_metal_tier3_normal` | Hoang Thú Bạch Kim Cự Hầu | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R1 C4 |
| `monster_metal_tier3_elite` | Tinh Anh Hoang Thú Bạch Kim Cự Hầu | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R2 C1 |
| `monster_metal_tier4_normal` | Thái Cổ Thiết Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R2 C2 |
| `monster_painting_tier2_normal` | Mặc Họa Tiên Hạc | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R2 C3 |
| `monster_painting_tier2_elite` | Tinh Anh Mặc Họa Tiên Hạc | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R2 C4 |
| `monster_painting_tier3_normal` | Hoang Thú Thạch Thanh Họa Linh | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R3 C1 |
| `monster_painting_tier4_normal` | Thái Cổ Đan Thanh Cự Điểu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R3 C2 |
| `monster_phantom_tier2_normal` | Hư Linh Yêu Hồ | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R3 C3 |
| `monster_phantom_tier3_normal` | Hoang Thú Vô Tướng Ảo Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R3 C4 |
| `monster_phantom_tier4_normal` | Thái Cổ Huyễn Ảo Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R4 C1 |
| `monster_poison_tier2_normal` | U Độc Chu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R4 C2 |
| `monster_poison_tier2_elite` | Tinh Anh U Độc Chu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R4 C3 |
| `monster_poison_tier3_normal` | Hoang Thú Kịch Độc Ma Xà | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-26.webp` | R4 C4 |
| `monster_poison_tier4_normal` | Thái Cổ Vạn Độc Cự Mãng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R1 C1 |
| `monster_qi_tier1_normal` | Khí Tức Linh Tước | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R1 C2 |
| `monster_qi_tier1_elite` | Tinh Anh Khí Tức Linh Tước | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R1 C3 |
| `monster_qi_tier2_normal` | Vân Khí Cự Mãng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R1 C4 |
| `monster_qi_tier2_elite` | Tinh Anh Vân Khí Cự Mãng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R2 C1 |
| `monster_qi_tier3_normal` | Hoang Thú Nguyên Khí Ma Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R2 C2 |
| `monster_qi_tier3_elite` | Tinh Anh Hoang Thú Nguyên Khí Ma Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R2 C3 |
| `monster_qi_tier4_normal` | Thái Cổ Khí Long | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R2 C4 |
| `monster_refinement_tier1_normal` | Luyện Hỏa Kỳ Nhông | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R3 C1 |
| `monster_refinement_tier1_elite` | Tinh Anh Luyện Hỏa Kỳ Nhông | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R3 C2 |
| `monster_refinement_tier2_normal` | Đỉnh Lô Giáp Trùng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R3 C3 |
| `monster_refinement_tier2_elite` | Tinh Anh Đỉnh Lô Giáp Trùng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R3 C4 |
| `monster_refinement_tier3_normal` | Hoang Thú Luyện Đạo Hỏa Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R4 C1 |
| `monster_refinement_tier3_elite` | Tinh Anh Hoang Thú Luyện Đạo Hỏa Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R4 C2 |
| `monster_refinement_tier4_normal` | Thái Cổ Luyện Lô Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R4 C3 |
| `monster_rule_tier1_normal` | Cân Xứng Linh Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-27.webp` | R4 C4 |
| `monster_rule_tier1_elite` | Tinh Anh Cân Xứng Linh Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R1 C1 |
| `monster_rule_tier2_normal` | Giới Luật Thạch Khôi | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R1 C2 |
| `monster_rule_tier2_elite` | Tinh Anh Giới Luật Thạch Khôi | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R1 C3 |
| `monster_rule_tier3_normal` | Hoang Thú Chấp Pháp Dị Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R1 C4 |
| `monster_rule_tier3_elite` | Tinh Anh Hoang Thú Chấp Pháp Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R2 C1 |
| `monster_rule_tier4_normal` | Thái Cổ Quy Tắc Thần Thú | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R2 C2 |
| `monster_soul_tier3_normal` | Hoang Thú Đãng Hồn Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R2 C3 |
| `monster_soul_tier3_elite` | Tinh Anh Hoang Thú Đãng Hồn Thú | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R2 C4 |
| `monster_soul_tier4_normal` | Thái Cổ U Hồn Cự Ma | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R3 C1 |
| `monster_sound_tier3_normal` | Hoang Thú Cuồng Âm Thần Viên | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R3 C2 |
| `monster_sound_tier4_normal` | Thái Cổ Long Ngâm Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R3 C3 |
| `monster_sound_tier4_elite` | Tinh Anh Thái Cổ Long Ngâm Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R3 C4 |
| `monster_space_tier1_normal` | Hư Không Thử | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R4 C1 |
| `monster_space_tier2_normal` | Thiên Di Hư Điêu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R4 C2 |
| `monster_space_tier3_normal` | Hoang Thú Không Gian Di Dời Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R4 C3 |
| `monster_space_tier3_elite` | Tinh Anh Hoang Thú Không Gian Di Dời Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-28.webp` | R4 C4 |
| `monster_space_tier4_normal` | Thái Cổ Hư Không Kình | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R1 C1 |
| `monster_space_tier4_elite` | Tinh Anh Thái Cổ Hư Không Kình | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R1 C2 |
| `monster_star_tier1_elite` | Tinh Anh Lưu Tinh Huỳnh Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R1 C3 |
| `monster_star_tier2_elite` | Tinh Anh Tinh Hà Linh Xà | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R1 C4 |
| `monster_star_tier3_normal` | Hoang Thú Tinh Quang Cự Hùng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R2 C1 |
| `monster_star_tier3_elite` | Tinh Anh Hoang Thú Tinh Quang Cự Hùng | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R2 C2 |
| `monster_star_tier4_normal` | Thái Cổ Tinh Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R2 C3 |
| `monster_strength_tier1_elite` | Tinh Anh Cự Lực Dã Trư | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R2 C4 |
| `monster_strength_tier3_normal` | Hoang Thú Cự Lực Viên | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R3 C1 |
| `monster_strength_tier3_elite` | Tinh Anh Hoang Thú Cự Lực Viên | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R3 C2 |
| `monster_strength_tier4_normal` | Thái Cổ Cự Lực Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R3 C3 |
| `monster_sword_tier2_normal` | Phi Kiếm Lang | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R3 C4 |
| `monster_sword_tier2_elite` | Tinh Anh Phi Kiếm Lang | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R4 C1 |
| `monster_sword_tier3_normal` | Hoang Thú Kiếm Ý Cự Long | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R4 C2 |
| `monster_sword_tier4_normal` | Thái Cổ Kiếm Ma Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R4 C3 |
| `monster_theft_tier1_elite` | Tinh Anh Đạo Chỉ Linh Hầu | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-29.webp` | R4 C4 |
| `monster_theft_tier2_elite` | Tinh Anh Thâu Quang Điêu | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R1 C1 |
| `monster_theft_tier3_normal` | Hoang Thú Diệu Thủ Yêu Miêu | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R1 C2 |
| `monster_theft_tier3_elite` | Tinh Anh Hoang Thú Diệu Thủ Yêu Miêu | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R1 C3 |
| `monster_theft_tier4_normal` | Thái Cổ Thâu Thiên Thử | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R1 C4 |
| `monster_time_tier1_normal` | Tuế Nguyệt Bàng Cua | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R2 C1 |
| `monster_time_tier1_elite` | Tinh Anh Tuế Nguyệt Bàng Cua | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R2 C2 |
| `monster_time_tier2_normal` | Thời Quang Linh Thiền | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R2 C3 |
| `monster_time_tier2_elite` | Tinh Anh Thời Quang Linh Thiền | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R2 C4 |
| `monster_time_tier3_normal` | Hoang Thú Tuế Nguyệt Thiềm | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R3 C1 |
| `monster_time_tier3_elite` | Tinh Anh Hoang Thú Tuế Nguyệt Thiềm | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R3 C2 |
| `monster_time_tier4_normal` | Thái Cổ Quang Âm Cự Quy | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R3 C3 |
| `monster_transformation_tier1_normal` | Bách Biến Tắc Kè | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R3 C4 |
| `monster_transformation_tier2_normal` | Thiên Huyễn Yêu Hồ | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R4 C1 |
| `monster_transformation_tier3_normal` | Hoang Thú Vạn Tượng Dị Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R4 C2 |
| `monster_transformation_tier3_elite` | Tinh Anh Hoang Thú Vạn Tượng Dị Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R4 C3 |
| `monster_transformation_tier4_normal` | Thái Cổ Thiên Biến Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-30.webp` | R4 C4 |
| `monster_water_tier2_normal` | Thủy Tinh Thao | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R1 C1 |
| `monster_water_tier3_normal` | Hoang Thú Lam Thủy Long | Normal | 3 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R1 C2 |
| `monster_water_tier3_elite` | Tinh Anh Hoang Thú Lam Thủy Long | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R1 C3 |
| `monster_water_tier4_normal` | Thái Cổ Thủy Ma | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R1 C4 |
| `monster_wind_tier2_normal` | Phong Bạo Độc Phong | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R2 C1 |
| `monster_wind_tier2_elite` | Tinh Anh Phong Bạo Độc Phong | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R2 C2 |
| `monster_wind_tier3_normal` | Hoang Thú Cương Phong Thần Ưng | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R2 C3 |
| `monster_wind_tier4_normal` | Thái Cổ Phong Ma | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R2 C4 |
| `monster_wisdom_tier1_elite` | Tinh Anh Bạch Diện Linh Hồ | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R3 C1 |
| `monster_wisdom_tier2_normal` | Thông Tuệ Viên | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R3 C2 |
| `monster_wisdom_tier2_elite` | Tinh Anh Thông Tuệ Viên | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R3 C3 |
| `monster_wisdom_tier3_normal` | Hoang Thú Thông Thiên Linh Hồ | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R3 C4 |
| `monster_wisdom_tier3_elite` | Tinh Anh Hoang Thú Thông Thiên Linh Hồ | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R4 C1 |
| `monster_wisdom_tier4_normal` | Thái Cổ Trí Tuệ Thần Thú | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R4 C2 |
| `monster_wood_tier3_normal` | Hoang Thú Bách Niên Thần Cây | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R4 C3 |
| `monster_wood_tier3_elite` | Tinh Anh Hoang Thú Bách Niên Thần Cây | Elite | 2 | `/assets/world/monsters/monster-atlas-batch-31.webp` | R4 C4 |
| `monster_wood_tier4_normal` | Thái Cổ Mộc Linh | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R1 C1 |
| `monster_investigation_tier4_elite` | Tinh Anh Thái Cổ Thần Đồng Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R1 C2 |
| `monster_luck_tier4_elite` | Tinh Anh Thái Cổ Tụ Vận Thần Thú | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R1 C3 |
| `monster_poison_tier4_elite` | Tinh Anh Thái Cổ Vạn Độc Cự Mãng | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R1 C4 |
| `monster_theft_tier4_elite` | Tinh Anh Thái Cổ Thâu Thiên Thử | Normal | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R2 C1 |
| `monster_water_tier4_elite` | Tinh Anh Thái Cổ Thủy Ma | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R2 C2 |
| `monster_wind_tier3_elite` | Tinh Anh Hoang Thú Cương Phong Thần Ưng | Elite | 3 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R2 C3 |
| `monster_enslavement_r9_normal` | Thái Cổ Ngự Thú Lang Chuyển 9 | Elite | 1 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R2 C4 |
| `monster_soul_r3_normal` | U Hồn Điệp Chuyển 3 | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R3 C1 |
| `monster_sound_r3_normal` | Âm Ba Linh Thú Chuyển 3 | Normal | 2 | `/assets/world/monsters/monster-atlas-batch-32.webp` | R3 C2 |

## Render and preload paths

- Phaser: backend `MonsterSpawn.TemplateID` is sent as snapshot `template_id`, stored in `WorldRuntime` as `templateId`, and passed from `WorldScene.ts` to `MonsterArt.ts`. The Phaser manifest maps IDs to atlas frames. `MonsterArt.ts` loads each atlas on demand, deduplicates requests by path, crops the selected frame, and caches a Phaser texture per template. `PreloadScene.ts` creates `token-monster` as the loading and failure fallback. Failed art requests may retry after ten seconds.
- Legacy Pixi: `frontend/src/components/WorldCanvas.tsx` still imports a compatibility re-export of the Phaser manifest. Its atlas loop currently starts loading all sheets when the map initializes. Phaser does not import Pixi code or the legacy component manifest, so those components can be removed separately.

## Asset generation notes

- New art is in `monster-atlas-batch-14.webp` through `monster-atlas-batch-32.webp`, generated as transparent creature sheets. Batches 14–31 use aligned 4×4 frames; batch 32 uses 4×3 with its final two cells empty. The manifest keeps each frame in CSV order.
- These are AI-generated illustrations. Inspect individual frames in game at target size before final art approval; some related tier or elite variants deliberately share anatomy and differ through palette, armor, or effects.

The built-in image generation prompt set used the existing monster atlas as a style reference. Each sheet requested painterly, semi-realistic high-fantasy bestiary cutouts on a transparent background, exactly one full-body creature per numbered grid cell, with no text, scenery, frame, watermark, or overlap. The numbered subjects were the corresponding manifest IDs with their `name`, `type`, and `description` from `Monster.csv`. The new sheets were recomposed onto uniform grids for Phaser frame cropping.
