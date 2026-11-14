# Ticket 02: Cơ Chế Thiên Tai Môi Trường & Hóa Giải Bằng Tiên Đạo Sát Chiêu

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để thiết kế và vận hành dạng tai kiếp **Thiên Tai Sinh Thái (`ENVIRONMENTAL`)** trên lưới ô Tiên Khiếu (`aperture_nodes`, `aperture_resource_points`), nơi người chơi hóa giải hiểm họa bằng Tiên Đạo Sát Chiêu và Trận Pháp thay vì đánh quái, đồng thời **bao quát toàn bộ 40+ Lưu Phái Đạo Ngân trong thế giới Cổ Chân Nhân** thay vì chỉ giới hạn ở một vài hệ cơ bản?

Cụ thể cần giải quyết:
1. **Mô hình Vùng Ảnh Hưởng (Hazard Zones on Grid)**:
   - Cách tính toán tọa độ lan tỏa của thiên tai (ví dụ: bão tuyết lan từ rìa vào trung tâm, mưa lửa rải rác theo cụm, lũ lụt nhấn chìm các vùng trũng, vết nứt không gian, mưa kiếm khí, mộng cảnh bành trướng).
   - Tác động theo chu kỳ (tick/turn): trừ sinh lực vách khiếu (`aperture_hp`), giảm độ phì nhiêu/vitality của các Mỏ Điểm Tài Nguyên (`ApertureResourcePoint`), đóng băng hoặc thiêu rụi công trình phòng thủ.
2. **Ma Trận Khắc Chế Toàn Diện 10 Nhóm Đại Lưu Phái (40+ Dao Paths)**:
   - Xây dựng ma trận tương sinh tương khắc bao quát: Binh Khí (Kiếm/Đao/Sát), Không-Thời (Vũ/Trụ), Quy Luật & Trận, Thể Phách (Lực/Cốt/Thực), Hồn Trí (Hồn/Trí/Mộng/Tình), Huyết Độc Tử, Tín Vận, Luyện Đan, Khí Đạo & Thiên Nhân.
   - Chi phí thi triển: tiêu hao Tiên Nguyên / Chân Nguyên và độ no của Cổ Trùng.
   - Hiệu quả hóa giải: dập tắt ô hazard, bảo toàn tài nguyên, chuyển hóa năng lượng thiên tai thành Kiếp Tài rơi rớt.
3. **Hiệp đồng Trận Pháp Phòng Thủ (Formation Shielding)**:
   - Công trình phòng thủ trên lưới ô tiêu hao Tiên Nguyên Thạch để mở lá chắn bao bọc các ô lân cận, chặn đứng sự ăn mòn của thiên tai trong một số lượt.
4. **Giao diện & Phản hồi trực quan**:
   - Hiệu ứng overlay thời tiết trên `ImmortalApertureView.tsx` (bão tuyết phủ trắng, ngọn lửa bập bùng, mây độc tím thẫm, vết nứt không gian xanh thẳm, mưa kiếm khí kim sắc...) kèm thanh tiến độ dập tắt thiên tai ("Mức Độ Thảm Họa: 75% ➔ 0%").

## Answer

Đã chuẩn hóa toàn diện cơ chế vận hành của dạng tai kiếp **Thiên Tai Sinh Thái (`ENVIRONMENTAL`)** trên lưới ô Tiên Khiếu $20 \times 20$, mở rộng quy chuẩn bao quát toàn bộ **10 Nhóm Đại Lưu Phái (40+ Lưu Phái Đạo Ngân)** chuẩn nguyên tác Cổ Chân Nhân:

### 1. Thuật Toán Lan Tỏa Vùng Hiểm Họa (Grid Hazard Spread Algorithm)
- **4 Kiểu Lan Tỏa Cốt Lõi (Cellular Automata Patterns)**:
  1. *Edge Inward (Rìa vào trong)*: Băng Tuyết Kiếp, Cửu Cung Phong, Sát Khí Đãng Nhiễm lan từ 4 đường biên giới bích co cụm dần vào trung tâm bản đồ.
  2. *Cluster Burst (Cụm bộc phát & Vết nứt)*: Liệt Hỏa, Địa Chấn, Bạch Cốt Rừng Chông sinh ra 2–3 ổ nứt vỡ ngẫu nhiên, mỗi tick lan sang các ô kế cận (4 hướng von Neumann).
  3. *Acid Rainfall & Sky Barrage (Mưa dội từ trời)*: Độc Vũ, Thần Tiêu Lôi Hải, Kiếm Khí Phong Bạo, Tinh Sa Sa Trụy dội ngẫu nhiên 8–15 ô khắp bản đồ.
  4. *Rift Expansion (Vết nứt dị thường bành trướng)*: Không Gian Phá Toái, Tuế Nguyệt Cuồng Lưu, Mộng Cảnh Bành Trướng xuất hiện từ 1 tâm điểm và nuốt chửng dần các ô xung quanh theo hình xoắn ốc hoặc ô cờ.
- **Tác Động Theo Chu Kỳ Tick (Tick Damage Settlement)**:
  - Ăn mòn vách ngăn Tiên Khiếu: $\Delta HP = \text{HazardCellsCount} \times 80$ mỗi tick. Nếu `aperture_hp <= 5% MaxHP`, độ kiếp thất bại, Tiên Khiếu chuyển sang trạng thái `DAMAGED`.
  - Giảm sinh lực Mỏ Điểm Tài Nguyên (`ApertureResourcePoint`): Trừ $-25$ điểm `vitality` mỗi tick. Nếu `vitality <= 0`, mỏ kiệt quệ (`EXHAUSTED`) và tạm ngừng sản xuất nguyên liệu.
  - Tổn thất bộ tộc Dị Nhân: Giảm $-5\%$ dân số mỗi tick, giảm `happiness` và tăng `oppression`.

---

### 2. Ma Trận Thiên Tai Môi Trường & Sát Chiêu Khắc Chế Toàn Diện 10 Nhóm Lưu Phái

Hệ thống phân hóa thành **10 Dạng Thiên Tai Môi Trường** tương ứng với 10 Nhóm Lưu Phái của codebase:

| Nhóm Lưu Phái (Dao Group) | Tên Thiên Tai Điển Tịch | Hiện Tượng Hiểm Họa Trên Lưới Ô | Sát Chiêu Khắc Chế Chuẩn | Kiếp Tài Rơi Rớt Tương Ứng | Phản Phệ Khi Dùng Sai Hệ |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1. ELEMENTAL**<br>*(Kim, Mộc, Thủy, Hỏa, Thổ, Băng, Lôi, Phong, Tinh, Quang, Ám)* | **Băng Tuyết Kiếp / Liệt Hỏa / Thần Tiêu Lôi** | Đóng băng ô đất, dung nham thiêu đốt, sét đánh phá hủy linh thảo mộc | **Viêm Đạo** dập Băng; **Thủy/Thổ** dập Hỏa; **Thổ/Vũ** phân tán Lôi; **Quang** xua tan Ám | `ice_crystal`, `fire_crystal`, `lightning_crystal`, `earth_soil` | Dùng Thủy vào Băng làm băng dày thêm; dùng Mộc vào Hỏa làm bùng cháy thêm 3 ô |
| **2. SPACE_TIME**<br>*(Vũ Đạo, Trụ Đạo)* | **Không Gian Phá Toái / Tuế Nguyệt Cuồng Lưu** | Vết nứt hư không xé toạc tọa độ ô đất; dòng thời gian bẻ cong làm già hóa mỏ tài nguyên | **Vũ Đạo** (*Không Gian Tọa Cố* vá rách); **Trụ Đạo** (*Định Thời Tiên Phù* ổn định dòng quang âm) | `space_crystal`, `time_sand` | Dùng Lực Đạo đánh vào vết nứt làm không gian vỡ vụn gấp đôi (+4 ô) |
| **3. KINETIC_WEAPON**<br>*(Kiếm, Đao, Binh, Sát)* | **Kiếm Khí Vũ / Đao Cương Trảm Liệt / Sát Khí** | Mưa kiếm khí kim sắc cắm thẳng xuống đất chém đứt địa mạo; sát khí làm dị nhân phát điên | **Kim Đạo / Cốt Đạo** (*Kim Cương Bất Hoại* thu gom kiếm khí); **Trí/Tình Đạo** (hóa giải sát khí) | `sword_crystal`, `metal_ore`, `cultivator_blood_essence` | Dùng Mộc Đạo bị kiếm khí chém nát vụn, gây sát thương x2 cho mỏ thảo mộc |
| **4. BODY_LIFE**<br>*(Lực, Cốt, Biến Hóa, Thực)* | **Cự Lực Trọng Áp / Bạch Cốt Rừng Chông / Cơ Ngạ** | Trọng lực tăng 50 lần đè bẹp tháp trận; rừng chông xương đâm từ lòng đất; nạn đói hút cạn lương thực | **Lực Đạo** (nâng đỡ càn khôn); **Biến Hóa Đạo** (đồng hóa xương); **Thực Đạo** (tiết chế cơ ngạ) | `strength_crystal`, `bone_marrow`, `food_essence` | Dùng Thổ Đạo gia cố đất làm rừng chông xương mọc càng dày đặc |
| **5. SOUL_MIND**<br>*(Hồn, Trí, Tình, Mộng, Nô)* | **Oan Hồn Dạ Vụ / Mộng Cảnh Bành Trướng** | Sương mù oan hồn nuốt chửng linh tính đất đai; mộng cảnh bảy màu lan tỏa bẫy kẹt các ô đất | **Quang Đạo / Hồn Đạo** (tịnh hóa siêu độ); **Mộng Đạo** (*Giải Mộng Sát Chiêu* thu nạp mộng cảnh) | `soul_sand`, `dream_pearl`, `wisdom_pearl` | Dùng Biến Hóa Đạo đi vào mộng cảnh sẽ bị đồng hóa và kẹt cứng vĩnh viễn |
| **6. ORGANIC_CORRUPTION**<br>*(Huyết, Độc, Tử)* | **U Minh Huyết Vũ / Độc Vụ Trướng Khí / Tử Vực** | Mưa máu làm ô uế sông ngòi; khói độc tím ăn mòn vách khiếu; tử khí làm cỏ cây khô héo | **Mộc Đạo / Dược Đạo** (sinh cơ lọc độc); **Quang Đạo** (thanh tẩy uế huyết); **Luyện Đạo** (tinh luyện độc tố) | `blood_crystal`, `poison_sand`, `death_sand` | Dùng Viêm Đạo làm bốc hơi khói độc bay tán loạn khắp toàn bộ bản đồ |
| **7. RULE_ORDER**<br>*(Luật, Cấm, Trận, Hủy Diệt)* | **Nghịch Đảo Càn Khôn / Tuyệt Pháp Cấm Cương** | Trọng lực đảo ngược làm bay nóc công trình; cấm vận hành sát chiêu thông thường | **Luật Đạo** (*Định Luật Chân Ngôn* lập lại trật tự); **Quy Tắc Đạo / Cấm Đạo** (phá giải cấm chế) | `rule_stone`, `destruction_iron` | Dùng sát chiêu phạm vào điều cấm sẽ bị nổ tung phản phệ ngược 50% HP Cổ Tiên |
| **8. INFORMATION_STEALTH**<br>*(Tín, Vận, Thâu, Huyễn, Họa)* | **Ách Vận Hắc Vân / Huyễn Thần Mê Vụ** | Vận rủi đen kịt làm nổ các mỏ khai thác; ảo ảnh che giấu vị trí thật của các ô tài nguyên | **Vận Đạo** (*Chuyển Vận Tiên Sát* xua tan xui xẻo); **Trí/Tín Đạo** (nhìn thấu chân tướng ảo ảnh) | `luck_sand`, `information_crystal` | Càng di chuyển dị nhân trong vùng ảo ảnh càng bị phân tán mất tích |
| **9. CRAFT_ALCHEMY**<br>*(Luyện, Đan, Tạo Hóa)* | **Hóa Luyện Chi Hỏa / Đan Khí Ô Nhiễm** | Lửa luyện tự nhiên làm tan chảy mỏ quặng; đan khí độc tràn ra từ khe nứt địa mạch | **Luyện Đạo / Đan Đạo** (*Nghịch Luyện Hóa Thuần* thu hồi tiên tài); **Thủy/Băng Đạo** (làm nguội dung dịch) | `refinement_stone`, `pill_essence` | Dùng Kim Đạo vào vùng hóa luyện khiến toàn bộ khoáng thạch tan chảy bốc hơi |
| **10. HEAVEN_HUMAN**<br>*(Thiên, Nhân, Khí)* | **Thiên Địa Khí Triều / Thiên Ý Đồng Hóa** | Khí triều bùng nổ làm vỡ vách giới bích; Thiên Ý trực tiếp xóa bỏ Đạo Ngân cá nhân | **Khí Đạo** (*Thu Khí Hóa Bình* điều hòa thiên địa khí); **Nhân Đạo** (*Nhân Định Thắng Thiên* bảo vệ bản ngã) | `heaven_sand`, `heaven_tribulation_shard`, `qi_essence` | Dùng sát chiêu dị hệ cưỡng ép chống trả sẽ bị Thiên Ý áp chế giảm 80% công lực |

---

### 3. Quy Tắc Khắc Chế Phổ Quát & Sát Chiêu Vạn Năng (Universal Containment)
- **Cơ chế Tương Sinh / Tương Trợ (Backfire Trigger)**:
  - Nếu Cổ Tiên dùng sát chiêu đồng hệ hoặc hệ tiếp tay cho thiên tai (ví dụ: dùng Thủy vào Tuyết Kiếp, dùng Mộc vào Hỏa Kiếp, dùng Lực vào Vết Nứt Không Gian), thiên tai sẽ bị **kích phát mạnh hơn (+2 đến +4 ô hazard mới)**!
- **Sát Chiêu Vạn Năng (Universal Containment Sát Chiêu)**:
  - Các lưu phái: **Khí Đạo (Qi)**, **Trận Đạo (Formation)**, **Vũ Đạo (Space)**, **Quy Luật Đạo (Rule)** sở hữu các sát chiêu "Phong Tỏa / Cách Ly Vùng" (*Isolation Barrier*): Không dập tắt hoàn toàn bản chất năng lượng nhưng đóng băng sự lan tỏa của mọi loại thiên tai trong 3 lượt.

### 4. Hiệp Đồng Trận Pháp Phòng Thủ (Formation Barrier Shielding)
- Tháp Trận Pháp phòng thủ (`DEFENSE` nodes trên grid) có thể tiêu hao 50 Tiên Nguyên Thạch kích hoạt **Lồng Chắn Cấm Chế**:
  - Bảo vệ vùng bán kính $R=1$ (cụm 9 ô xung quanh tháp) trong 3 lượt/tick.
  - Toàn bộ Mỏ Tài Nguyên và Dị Nhân trong lồng chắn miễn nhiễm 100% trước sự lan tỏa và ăn mòn của bất kỳ loại thiên tai nào.

### 5. Giao Diện & API Endpoint Chuẩn Hóa
- **Endpoint**: `POST /api/apertures/tribulations/:id/mitigate`
  - Payload: `{ "target_x": 4, "target_y": 7, "sat_chieu_id": "...", "path": "fire", "aoe": true }`
  - Xử lý: Kiểm tra lưu phái khắc chế trong Ma Trận 10 Nhóm, khấu trừ mana/tiên nguyên, dập tắt ô hazard, ngưng tụ Kiếp Tài rơi tại tọa độ ô, và đồng bộ SSE về client.
- **Client Overlay (`ImmortalApertureView.tsx`)**:
  - Hỗ trợ đầy đủ bộ lọc thời tiết: Bão tuyết (`❄️`), Liệt hỏa (`🔥`), Độc vũ (`☣️`), Vết nứt không gian (`🌀`), Mưa kiếm khí (`⚔️`), Sương mù oan hồn (`👻`), Rừng chông xương (`🦴`), Mộng cảnh lan tỏa (`🌈`), Khí triều bùng nổ (`💨`).
- **Đã kiểm chứng thành công qua Prototype HTML**:
  [`prototypes/02_environmental_mitigation_demo.html`](../prototypes/02_environmental_mitigation_demo.html).
