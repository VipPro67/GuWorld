# Ticket 07: Khảo Cứu Ma Trận Tai Kiếp Nguyên Tác & Ánh Xạ Dữ Liệu Codebase

Type: research
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Khảo cứu toàn diện các điển tích tai kiếp trong tiểu thuyết Cổ Chân Nhân (Reverend Insanity) và đối chiếu với dữ liệu hiện có trong codebase (`backend/data/`, `backend/internal/models/`, `backend/internal/gamefx/`) để xây dựng Ma Trận Tai Kiếp Tiêu Chuẩn:

Cụ thể cần làm rõ:
1. **Phân Tầng Cấp Độ Kiếp Nạn Theo Rank**:
   - Rank 6 (Lục Chuyển): Địa Tai (Earth Calamity) - 10 năm một lần, tích lũy 30 lần.
   - Rank 7 (Thất Chuyển): Thiên Kiếp (Heavenly Tribulation) - 10 Địa Tai gặp 1 Thiên Kiếp.
   - Rank 8 (Bát Chuyển): Hạo Kiếp (Grand Tribulation) - 3 Thiên Kiếp gặp 1 Hạo Kiếp.
   - Rank 9 (Cửu Chuyển): Vạn Kiếp & Hỗn Độn Kiếp (Myriad Tribulation & Chaos Calamity).
2. **Danh Mục Điển Tịch Các Dạng Kiếp Nạn Theo Nguyên Tác**:
   - Tuyết Kiếp vạn dặm (Phương Nguyên độ kiếp ở Bắc Nguyên).
   - Liệt Hỏa Diệt Thế & Bão Lửa Viêm Đạo.
   - Cửu Cung Phong & Cuồng Lôi Điện Kiếp.
   - Ngũ Uẩn Tâm Kiếp & Huyễn Cảnh Mộng Điệp.
   - Huyết Thú Triều & Thiết Bì Tê Giác Bầy.
   - Sương Mù Hỗn Độn (Chaos Fog) cấp Tôn Giả.
3. **Ánh Xạ Với Hệ Thống Codebase Hiện Có**:
   - Hệ thống 40+ Lưu Phái Đạo Ngân trong `models.AllGuPaths()`.
   - Danh sách vật phẩm có sẵn trong `data/Item.csv` có thể làm Kiếp Tài hoặc cần bổ sung mới.
   - Monster templates trong `data/Monster.csv`.

## Answer

Đã hoàn tất khảo cứu toàn diện kiến trúc codebase và kho tàng nguyên tác Cổ Chân Nhân (Reverend Insanity). Kết quả cung cấp ma trận ánh xạ kỹ thuật chuẩn xác cho toàn bộ hệ thống Tai Kiếp:

### 1. Phân Tầng Cấp Độ Kiếp Nạn Theo Rank & Chu Kỳ Codebase
- **Rank 6 (Lục Chuyển)**: Địa Tai (`EARTH_CALAMITY`, chu kỳ 10 năm), Thiên Kiếp (`HEAVENLY_TRIBULATION`, 100 năm). Vượt qua 30 Địa Tai + 3 Thiên Kiếp để đột phá Rank 7. Phần thưởng: 250 Đạo Ngân (Địa Tai), 750 Đạo Ngân (Thiên Kiếp).
- **Rank 7 (Thất Chuyển)**: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (`GRAND_TRIBULATION`, 100 năm). Vượt qua 3 Hạo Kiếp để đột phá Rank 8. Phần thưởng: 1,500 Đạo Ngân (Địa Tai), 5,000 Đạo Ngân (Thiên Kiếp), 20,000 Đạo Ngân (Hạo Kiếp).
- **Rank 8 (Bát Chuyển)**: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (`MYRIAD_TRIBULATION`, 100 năm). Vượt qua 3 Vạn Kiếp để thành Á Tiên Tôn. Phần thưởng: 10,000 Đạo Ngân (Thiên Kiếp), 40,000 Đạo Ngân (Hạo Kiếp), 150,000 Đạo Ngân (Vạn Kiếp).
- **Rank 9 (Cửu Chuyển Tôn Giả)**: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Kiếp (`CHAOS_CALAMITY`, 100 năm). Phần thưởng: 50,000 Đạo Ngân (Hạo Kiếp), 200,000 Đạo Ngân (Vạn Kiếp), 1,500,000 Đạo Ngân (Hỗn Độn Kiếp).
- **Rank 10 (Vĩnh Sinh)**: Bất tử, bất diệt, miễn nhiễm toàn bộ tai kiếp (`IsVinhSinhRank(rank)`).

### 2. Ma Trận 4 Archetype Tai Kiếp Chuẩn Nguyên Tác & Ánh Xạ Kiếp Tài (`Item.csv`)

| Rank & Loại Kiếp | Tên Kiếp Nạn Tiêu Biểu | Archetype | Lưu Phái | Mô Tả & Cơ Chế Đối Kháng | Kiếp Tài Rơi Rớt Tương Ứng (`Item.csv`) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **RANK 6**<br>*Địa Tai* | **Băng Tuyết Kiếp** | `ENVIRONMENTAL` | Ice / Water | Hàn phong vạn dặm đóng băng lưới ô, giảm 50% vitality mỏ. Cần Hỏa Đạo sát chiêu dập băng. | `ice_crystal_rank6`, `ice_cicada_wing_rank6` |
| | **Liệt Hỏa Bộc Phát** | `ENVIRONMENTAL` | Fire / Earth | Dung nham phun trào lan 3 ô/lượt thiêu rụi thảo mộc. Dùng Thủy Đạo / Thổ Đạo trấn áp. | `fire_crystal_rank6`, `fire_rat_fur_rank6` |
| | **Độc Vũ Tử Vực** | `ENVIRONMENTAL` | Poison / Water | Mưa tím ăn mòn vách khiếu (`aperture_hp -5%/turn`). Dùng Mộc Đạo hấp thụ / Trận Pháp lọc độc. | `poison_sand_rank6`, `poison_sac_rank6` |
| | **Thiết Bì Tê Đàn** | `BEAST_SWARM` | Strength / Metal | 3 đợt thú hoang da sắt húc phá tường thành và mỏ quặng. Cần Tháp Trận Pháp + Hộ vệ cản đường. | `metal_ore_rank6`, `wild_boar_tusk_rank6` |
| | **Cốt Lang Triều** | `BEAST_SWARM` | Bone / Enslavement | Bầy sói xương tấn công khu dân cư Mao Dân. Cổ Tiên xuất trận hoặc dùng Nô Đạo thu phục. | `bone_marrow_rank6`, `wolf_king_fang_rank6` |
| | **Sơ Khảo Ngũ Uẩn** | `MIND_SOUL` | Wisdom / Emotion | Ảo cảnh cám dỗ về thọ nguyên và danh lợi phàm trần. Cần Định Lực / Ý Chí hóa giải. | `wisdom_pearl_rank6`, `emotion_silk_rank6` |
| | **Mộng Yểm Mê Điệp** | `MIND_SOUL` | Dream | Phấn bướm gây mê man, xuất hiện ác mộng. Cần Mộng Đạo hoặc Hồn Đạo thanh tỉnh. | `dream_pearl_rank6`, `dream_butterfly_dust_rank6` |
| | **Địa Tai Thần Ma** | `ANOMALOUS_COMBAT` | Earth / Rule | Boss `monster_earth_calamity_boss` giáng lâm, sàn đấu chấn động gây choáng ngẫu nhiên. | `heaven_tribulation_shard_rank6`, `earth_soil_rank6` |
| **RANK 7**<br>*Thiên Kiếp* | **Cửu Cung Phong Kiếp** | `ENVIRONMENTAL` | Wind / Space | Gió xoáy 9 cung càn quét phá hủy tháp trận. Cần Sát Chiêu Không Gian / Phong Đạo định phong. | `wind_stone_rank7`, `storm_eagle_feather_rank7` |
| | **U Minh Huyết Vũ** | `ENVIRONMENTAL` | Blood / Dark | Mưa máu ô nhiễm thổ nhưỡng thành Huyết Ma Vực. Dùng Quang Đạo / Mộc Đạo tịnh hóa. | `blood_crystal_rank7`, `dark_crystal_rank7` |
| | **Thần Tiêu Lôi Hải** | `ENVIRONMENTAL` | Lightning / Heaven | Biển sét dội xuống liên tục, khóa Chân Nguyên Tiên Cổ. Cần Tiên Cổ Phòng chắn sét. | `lightning_crystal_rank7`, `heaven_tribulation_shard_rank7` |
| | **Thượng Cổ Huyết Bức** | `BEAST_SWARM` | Blood / Flight | Bầy dơi quỷ thượng cổ bay trên cao phớt lờ chướng ngại vật mặt đất, cắn nuốt sinh linh dị nhân. | `blood_crystal_rank7`, `blood_bat_wing_rank7` |
| | **Địa Câu Thâm Ngô** | `BEAST_SWARM` | Poison / Earth | Rết khổng lồ đục thủng địa mạch, làm sụp đổ các node tài nguyên ngầm. | `centipede_carapace_rank7`, `earth_soil_rank7` |
| | **Vấn Tâm Luân Hồi** | `MIND_SOUL` | Wisdom / Soul | Thiên Ý tái hiện 3 kiếp luân hồi thử thách Đạo Tâm. Lựa chọn sai dẫn đến nứt vỡ Thần Hồn. | `wisdom_pearl_rank7`, `soul_sand_rank7` |
| | **Tâm Ma Bản Thể** | `MIND_SOUL` | Soul / Transformation | Quyết đấu trong Thức Hải với bản sao bóng tối sở hữu toàn bộ sát chiêu Rank 7 của bản thân. | `cultivator_blood_essence_rank7`, `dream_pearl_rank7` |
| | **Thiên Kiếp Thần Ma** | `ANOMALOUS_COMBAT` | Heaven / Rule | Boss `monster_heaven_tribulation_boss` áp dụng "Cấm Pháp Tuyệt Vực" (khóa 1 nhánh cổ trùng). | `heaven_sand_rank7`, `heaven_tribulation_shard_rank7` |
| **RANK 8**<br>*Hạo Kiếp* | **Tinh Hà Trụy Lạc** | `ENVIRONMENTAL` | Star / Space | Mưa thiên thạch Thái Cổ từ bầu trời rơi xuống hủy diệt diện rộng. Cần Sát Chiêu Vũ Đạo dịch chuyển. | `star_sand_rank8`, `space_crystal_rank8` |
| | **Cực Dạ Hắc Ám** | `ENVIRONMENTAL` | Dark / Death | Màn đêm vĩnh cửu nuốt trọn ánh sáng, suy thoái toàn bộ thảm thực vật. Cần Quang Đạo đại trận. | `dark_crystal_rank8`, `death_sand_rank8` |
| | **Thái Cổ Niên Thú** | `BEAST_SWARM` | Time / Sound | Đàn thú Niên Thú vượt sông Thời Gian tràn vào cắn nuốt thọ nguyên và nguyên khí. | `sound_resonance_rank8`, `time_sand_rank8` |
| | **Thao Thiết Cự Thú** | `BEAST_SWARM` | Food / Strength | Đàn thú háu ăn ngốn ngấu vách khiếu và nuốt chửng các mỏ tiên tài. | `food_essence_rank8`, `strength_crystal_rank8` |
| | **Chấp Niệm Vạn Niên** | `MIND_SOUL` | Emotion / Wisdom | Đào sâu nỗi đau day dứt nhất trong đời tu tiên, nếu không phá vỡ chấp niệm sẽ tự hóa đạo tiêu vong. | `emotion_silk_rank8`, `wisdom_pearl_rank8` |
| | **Vạn Kiếp Thần Ma** | `ANOMALOUS_COMBAT` | Sound / Destruction | Boss `monster_myriad_tribulation_boss` áp dụng "Nghịch Chuyển Càn Khôn" (phản hồi sát thương, cấm hồi phục). | `heaven_sand_rank8`, `heaven_tribulation_shard_rank8`, `destruction_iron_rank8` |
| **RANK 9**<br>*Hỗn Độn Kiếp* | **Hỗn Độn Vụ Lực** | `ENVIRONMENTAL` | Chaos / Void | Sương mù xám từ hư không vũ trụ rò rỉ vào xé rách màng giới bích, phân rã vạn vật thành hư vô. | `destruction_iron_rank9`, `ITEM_APERTURE_EXPANSION_PILL` |
| | **Hỗn Độn Thần Ma Triều** | `BEAST_SWARM` | Chaos / Destruction | Thực thể hỗn mang vô hình vô tướng tràn vào, miễn nhiễm toàn bộ sát chiêu thông thường. | `destruction_iron_rank9`, `heaven_tribulation_shard_rank9` |
| | **Hỗn Độn Đồng Hóa** | `MIND_SOUL` | Void / Wisdom | Khảo nghiệm cấp Tôn Giả: Hỗn độn xóa nhòa bản ngã và Chân Ý. Cần ý chí trường sinh tối thượng. | `dream_pearl_rank9`, `ITEM_RANDOM_CONSTITUTION` |
| | **Hỗn Độn Thần Ma** | `ANOMALOUS_COMBAT` | Chaos / Destruction | Boss `monster_chaos_calamity_boss` áp dụng "Vô Pháp Vô Thiên" (xóa bỏ mọi Buff/Khiên mỗi 3 turn, sát thương xuyên thấu). | `heaven_sand_rank9`, `heaven_tribulation_shard_rank9`, `ITEM_RANDOM_RACE` |

### 3. Vòng Đời 3 Giai Đoạn & Cơ Chế Kiếp Tài Rơi Rớt Lưới Ô
- **Giai Đoạn 1: ĐIỀM BÁO (`HARBINGER`)**: Dự báo trước loại Kiếp và Lưu phái để Cổ Tiên gia cố trận pháp, sơ tán dị nhân hoặc điều chỉnh sát chiêu.
- **Giai Đoạn 2: NGHÊNH KIẾP (`ORDEAL_ACTIVE`)**: Thực thi 1 trong 4 Archetype (Dập tắt thiên tai môi trường / Thủ hộ thú triều / Khảo nghiệm đạo tâm / Quyết đấu dị tượng sàn đấu).
- **Giai Đoạn 3: DƯ BA & THU HOẠCH (`AFTERMATH`)**:
  - Các ô lưới ô tham chiến ngưng tụ thành các **Điểm Thu Hoạch Kiếp Tài (Spoils Nodes)** chứa các nguyên liệu tương ứng trong bảng trên, dùng trực tiếp cho Luyện Tiên Cổ và Nâng cấp Tiên Cổ Phòng.
  - Tổn thất sinh thái và ô nhiễm Đạo Ngân: để lại các ô đất biến dị (dung nham, đầm lầy độc, dải băng) cần Cổ Tiên dùng Sát Chiêu / Tiên Nguyên Thạch để **Thanh Tẩy (Cleanse)** hoặc **Biến Nguy Thành Cơ (Ecological Pivot)**.
