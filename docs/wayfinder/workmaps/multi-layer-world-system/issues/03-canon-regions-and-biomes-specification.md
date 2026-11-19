# Ticket 03: Đặc Tả Toàn Diện Hệ Sinh Thái Region, MapCell, Quái Vật & Mỏ Quặng Cho 3 Tầng Mới

Type: prototype
Status: resolved
Blocked by: 01-vertical-world-schema-and-dia-uyen-layer, 02-heavenly-wind-qi-wall-and-layer-transit-mechanics
Assignee: Antigravity

## Question

Làm thế nào để xây dựng bộ dữ liệu hoàn chỉnh (`Region.csv`, `MapCell.csv`, `Monster.csv`, `MonsterSpawn.csv`, `LootTable.csv`) cho Tầng 0 (Địa Uyên), Tầng 2 (Bạch Thiên) và Tầng 3 (Hắc Thiên) phản ánh trung thực bản sắc sinh thái của Cổ Giới?

Cụ thể cần giải quyết:
1. **Thiết kế Hệ sinh thái Tầng 0: Địa Uyên (`dia_uyen`)**:
   - Các Region:
     - `du_am_luu_dia_mach`: Dòng sông ngầm cuồn cuộn bùn đen và thủy tức quái dị.
     - `du_huyet_tinh_tham_hac`: Hầm quặng thạch anh đỏ ngập tràn huyết khí rỉ ra từ cội nguồn Huyết Đạo.
     - `du_thach_nhan_co_do`: Di tích thành quách của Thạch Nhân bộ tộc dưới đáy sâu địa mạch.
   - Quái vật: Địa Giáp Ma Trùng, Cự Thạch Cốt Ma, Âm Thi Khôi Lỗi (Rank 5–7).
   - Tài nguyên: `earth_soil_rank6`, `blood_crystal_rank6`, `dark_crystal_rank6`.
2. **Thiết kế Hệ sinh thái Tầng 2: Bạch Thiên (`bai_thien`)**:
   - Các Region:
     - `bt_thai_co_van_hai`: Biển mây trập trùng, nơi trú ngụ của bầy Thiết Trảo Phi Hạc và Lôi Bằng.
     - `bt_quang_minh_tinh_doi`: Vành đai thiên thạch lơ lửng chứa Quang Tinh Thạch và Thái Dương Tinh Hoa.
     - `bt_thien_dinh_ngoai_vuc`: Vùng trời giáp ranh cấm địa Thiên Đình, ánh kim sắc lộng lẫy.
   - Quái vật: Thiết Trảo Phi Hạc Vương, Thái Cổ Bạch Điểu, Quang Mang Huyễn Điệp (Rank 6–8).
   - Tài nguyên: `cloud_crystal_rank6`, `light_crystal_rank7`, `flying_crane_feather`.
3. **Thiết kế Hệ sinh thái Tầng 3: Hắc Thiên (`hac_thien`)**:
   - Các Region:
     - `ht_da_mac_tinh_hai`: Biển sao đen kịt, nơi săn bắn Tinh thú và Hồn thú Thái Cổ vạn trượng.
     - `ht_u_minh_tham_uyen`: Vực sâu bóng tối cấm kỵ, sương mù đen nuốt chửng ánh sáng.
     - `ht_di_nhan_vuong_quoc`: Căn cứ của liên minh Người Lông, Tuyết Nhân, Thạch Nhân ẩn thế.
   - Quái vật: Thái Cổ Hồn Thú, Tinh Thú Đầu Đàn, Hắc Ám Cự Mãng (Rank 8–9).
   - Tài nguyên: `dark_crystal_rank8`, `soul_sand_rank8`, `star_stone_rank8`.

## Answer

Đã hoàn thành toàn diện việc xây dựng hệ sinh thái sinh vật, vùng đất, quái vật, mỏ khoáng sản và bàn rơi phần thưởng (loot) cho cả 3 tầng không gian thẳng đứng cùng với việc tái cân bằng triệt để quái vật trên Ngũ Vực (`world_map`):

### 1. Tái cân bằng Quái vật trên Bề Mặt Ngũ Vực (`world_map`)
- Đã điều chỉnh dải cấp bậc của 5 đại vực (`dai_vuc_nam_cuong`, `dai_vuc_trung_chau`, `dai_vuc_bac_nguyen`, `dai_vuc_tay_mac`, `dai_vuc_dong_hai`) trong `Region.csv` từ `1-9` thành `1-5`. Nhờ đó thuật toán `clampRankToZoneBand` sẽ giới hạn quái vật xuất hiện ngẫu nhiên trên mặt đất ở mức Phàm cấp tối đa (Rank 5).
- Trong `MonsterSpawn.csv`, đã loại bỏ toàn bộ các quái vật cấp Tiên/Thái Cổ (Rank 6–8) trước đây bị gán nhầm vào bản đồ Ngũ Vực (`monster_strength_thai_co_r8r9`, `monster_strength_tier4_normal`, `monster_immemorial_creation_ruin_golem`, `monster_immemorial_lightning_eel`, `monster_water_tier4_normal`, v.v.), thay thế bằng các quái thú phàm cấp Rank 1–4 chuẩn nguyên tác như `monster_boar_king`, `monster_flying_fox`, `monster_night_wolf`, `monster_sand_wraith`, `monster_water_serpent`, `monster_crescent_crocodile`.

### 2. Hệ sinh thái Tầng 0: Vạn Trượng Địa Uyên (`dia_uyen`)
- **3 Vùng đất mới (`Region.csv`)**:
  - `du_am_luu_dia_mach` (Âm Lưu Địa Mạch, Rank 5–7): Dòng sông bùn đen cuồn cuộn dưới lòng đất, chứa tài nguyên `earth_soil_rank6` và `dark_crystal_rank6`.
  - `du_huyet_tinh_tham_hac` (Huyết Tinh Thâm Hác, Rank 6–8): Hầm quặng thạch anh đỏ ngập tràn huyết khí rỉ ra từ cội nguồn Huyết Đạo, chứa `blood_crystal_rank6` và `dark_crystal_rank6`.
  - `du_thach_nhan_co_do` (Thạch Nhân Cổ Đô, Rank 5–7): Tàn tích thành trì của tộc Thạch Nhân, chứa `earth_soil_rank6`.
- **Quái vật & Boss (`Monster.csv`)**:
  - `monster_dia_giap_ma_trung` (Địa Giáp Ma Trùng, Normal, Rank 5)
  - `monster_cu_thach_cot_ma` (Cự Thạch Cốt Ma, Elite, Rank 6)
  - `monster_am_thi_khoi_loi` (Âm Thi Khôi Lỗi, Normal, Rank 5)
  - `boss_dia_uyen_hoang_thu` (Địa Uyên Thú Hoàng, Boss, Rank 7–8)
- **Điểm spawn & Boss Cell**: Phân bổ đầy đủ qua `MonsterSpawn.csv`, `MapCell.csv` và `RegionLootTables.csv`.

### 3. Hệ sinh thái Tầng 2: Thái Cổ Bạch Thiên (`bai_thien`)
- **3 Vùng đất mới (`Region.csv`)**:
  - `bt_thai_co_van_hai` (Thái Cổ Vân Hải, Rank 6–8): Biển mây vạn dặm, chứa `cloud_crystal_rank6` và `flying_crane_feather`.
  - `bt_quang_minh_tinh_doi` (Quang Minh Tinh Đới, Rank 6–8): Vành đai thiên thạch phát sáng, chứa `light_crystal_rank7`.
  - `bt_thien_dinh_ngoai_vuc` (Thiên Đình Ngoại Vực, Rank 7–9): Vùng trời mây trắng áp sát Thiên Đình, chứa `cloud_crystal_rank6` và `light_crystal_rank7`.
- **Quái vật & Boss (`Monster.csv`)**:
  - `monster_thiet_trao_phi_hac` (Thiết Trảo Phi Hạc, Normal, Rank 6)
  - `monster_thai_co_bach_dieu` (Thái Cổ Bạch Điểu, Elite, Rank 7)
  - `monster_quang_mang_huyen_diep` (Quang Mang Huyễn Điệp, Normal, Rank 6)
  - `boss_bach_thien_hac_vuong` (Thiết Trảo Hạc Hoàng, Boss, Rank 8)
- **Vật phẩm mới (`Item.csv`)**: Thêm nguyên liệu Lục Chuyển Tiên Tài `flying_crane_feather` (Thiết Hạc Linh Vũ).

### 4. Hệ sinh thái Tầng 3: Thái Cổ Hắc Thiên (`hac_thien`)
- **3 Vùng đất mới (`Region.csv`)**:
  - `ht_da_mac_tinh_hai` (Dạ Mạc Tinh Hải, Rank 8–9): Biển sao đen kịt, chứa `dark_crystal_rank8` và `star_stone_rank8`.
  - `ht_u_minh_tham_uyen` (U Minh Thâm Uyên, Rank 8–9): Vực sâu bóng tối nuốt chửng linh giác, chứa `dark_crystal_rank8` và `soul_sand_rank8`.
  - `ht_di_nhan_vuong_quoc` (Dị Nhân Vương Quốc, Rank 7–9): Căn cứ địa liên minh Dị Nhân cổ, chứa `dark_crystal_rank8` và `star_stone_rank8`.
- **Quái vật & Boss (`Monster.csv`)**:
  - `monster_tinh_thu_dau_dan` (Tinh Thú Đầu Đàn, Normal, Rank 8)
  - `monster_thai_co_hon_thu` (Thái Cổ Hồn Thú, Elite, Rank 8)
  - `monster_hac_am_cu_mang` (Hắc Ám Cự Mãng, Normal, Rank 8)
  - `boss_hac_thien_hon_de` (Hồn Đế Dạ Thần, Boss, Rank 9)

### 5. Dữ liệu Loot Table (`LootTable.csv` & `RegionLootTables.csv`)
- Khởi tạo 12 bàn phần thưởng mới chuẩn hóa xác suất rơi các loại Tiên Tài, Cổ Trùng, Nguyên Thạch tương ứng với đạo ngân đặc trưng (Thổ/Cốt/Huyết, Vân/Quang/Phong, Hồn/Ám/Tinh).
- Bổ sung cấu hình loot theo vùng trong `RegionLootTables.csv` và các tọa độ Boss thủ lĩnh trong `MapCell.csv`.

## Question

Làm thế nào để xây dựng bộ dữ liệu hoàn chỉnh (`Region.csv`, `MapCell.csv`, `Monster.csv`, `MonsterSpawn.csv`, `LootTable.csv`) cho Tầng 0 (Địa Uyên), Tầng 2 (Bạch Thiên) và Tầng 3 (Hắc Thiên) phản ánh trung thực bản sắc sinh thái của Cổ Giới?

Cụ thể cần giải quyết:
1. **Thiết kế Hệ sinh thái Tầng 0: Địa Uyên (`dia_uyen`)**:
   - Các Region:
     - `du_am_luu_dia_mach`: Dòng sông ngầm cuồn cuộn bùn đen và thủy tức quái dị.
     - `du_huyet_tinh_tham_hac`: Hầm quặng thạch anh đỏ ngập tràn huyết khí rỉ ra từ cội nguồn Huyết Đạo.
     - `du_thach_nhan_co_do`: Di tích thành quách của Thạch Nhân bộ tộc dưới đáy sâu địa mạch.
   - Quái vật: Địa Giáp Ma Trùng, Cự Thạch Cốt Ma, Âm Thi Khôi Lỗi (Rank 5–7).
   - Tài nguyên: `earth_soil_rank6`, `blood_crystal_rank6`, `dark_crystal_rank6`.
2. **Thiết kế Hệ sinh thái Tầng 2: Bạch Thiên (`bai_thien`)**:
   - Các Region:
     - `bt_thai_co_van_hai`: Biển mây trập trùng, nơi trú ngụ của bầy Thiết Trảo Phi Hạc và Lôi Bằng.
     - `bt_quang_minh_tinh_doi`: Vành đai thiên thạch lơ lửng chứa Quang Tinh Thạch và Thái Dương Tinh Hoa.
     - `bt_thien_dinh_ngoai_vuc`: Vùng trời giáp ranh cấm địa Thiên Đình, ánh kim sắc lộng lẫy.
   - Quái vật: Thiết Trảo Phi Hạc Vương, Thái Cổ Bạch Điểu, Quang Mang Huyễn Điệp (Rank 6–8).
   - Tài nguyên: `cloud_crystal_rank6`, `light_crystal_rank7`, `flying_crane_feather`.
3. **Thiết kế Hệ sinh thái Tầng 3: Hắc Thiên (`hac_thien`)**:
   - Các Region:
     - `ht_da_mac_tinh_hai`: Biển sao đen kịt, nơi săn bắn Tinh thú và Hồn thú Thái Cổ vạn trượng.
     - `ht_u_minh_tham_uyen`: Vực sâu bóng tối cấm kỵ, sương mù đen nuốt chửng ánh sáng.
     - `ht_di_nhan_vuong_quoc`: Căn cứ của liên minh Người Lông, Tuyết Nhân, Thạch Nhân ẩn thế.
   - Quái vật: Thái Cổ Hồn Thú, Tinh Thú Đầu Đàn, Hắc Ám Cự Mãng (Rank 8–9).
   - Tài nguyên: `dark_crystal_rank8`, `soul_sand_rank8`, `star_stone_rank8`.
