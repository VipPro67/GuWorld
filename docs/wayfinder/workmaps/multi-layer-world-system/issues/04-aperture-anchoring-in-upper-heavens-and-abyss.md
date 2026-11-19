# Ticket 04: Cơ Chế Cổ Tiên Hạ Khiếu Cắm Rễ Tiên Khiếu Trên Tầng Trời & Vực Sâu

Type: prototype
Status: resolved
Blocked by: 01-vertical-world-schema-and-dia-uyen-layer, 03-canon-regions-and-biomes-specification
Assignee: Antigravity

## Question

Làm thế nào để tích hợp cơ chế Hạ Khiếu (Aperture Anchoring / Placement) của Cổ Tiên khi đứng trên Tầng 0 (Địa Uyên) hoặc Tầng 2/3 (Bạch Thiên / Hắc Thiên), tạo ra sự phân hóa rõ rệt về sản lượng Thiên Địa Nhị Khí và chu kỳ Tai Kiếp đúng theo nguyên tác?

Cụ thể cần giải quyết:
1. **Phân hóa Thiên Địa Nhị Khí theo Tầng đặt Tiên Khiếu**:
   - Trong `aperture_core.go` và worker sinh khí:
     - Hạ khiếu tại **Tầng 1 (Ngũ Vực)**: Cân bằng tiêu chuẩn (100% Thiên Khí, 100% Địa Khí).
     - Hạ khiếu tại **Tầng 2 (Bạch Thiên)**: Dồi dào Thiên Khí (300% Thiên Khí), nhưng thiếu hụt Địa Khí (30% Địa Khí) ➔ Buộc Cổ Tiên phải mang theo Địa Khí Đan hoặc định kỳ hạ phàm nạp Địa Khí.
     - Hạ khiếu tại **Tầng 3 (Hắc Thiên)**: Thiên Khí dị biến (Thiên Khí pha trộn Ám Đạo / Tinh Đạo), chu kỳ Tai Kiếp bị kéo dài thêm 50% thời gian (Thiên Ý khó cảm ứng).
     - Hạ khiếu tại **Tầng 0 (Địa Uyên)**: Dồi dào Địa Khí (350% Địa Khí), nhưng cực kỳ hiếm Thiên Khí (20% Thiên Khí) ➔ Rất thích hợp cho các Cổ Tiên tu luyện Thổ Đạo, Luyện Đạo, Kim Đạo, Cốt Đạo.
2. **Hiển thị Môn Hộ theo `world_map_id`**:
   - Bảng `aperture_portals` đã có `region_id`, `x`, `y`.
   - Cần đảm bảo khi Môn Hộ sinh ra trên Bạch Thiên hoặc Địa Uyên, chỉ người chơi đang ở cùng `world_map_id` đó mới nhìn thấy và do thám được Môn Hộ.

## Answer

Đã hoàn thành toàn diện cơ chế Hạ Khiếu (Aperture Anchoring) cắm rễ Tiên Khiếu trên các tầng không gian thẳng đứng:

### 1. Phân hóa Thiên Địa Nhị Khí theo Tầng Đặt Tiên Khiếu
- Tạo module `aperture_layer_qi.go` định nghĩa chính xác hệ số nạp khí cho từng tầng thế giới (`LayerQiMultipliers(layer)`):
  - **Tầng 0 (`dia_uyen`)**: Thiên Khí 20%, Địa Khí 350% (cực kỳ thích hợp cho Cổ Tiên Thổ Đạo, Kim Đạo, Luyện Đạo, Cốt Đạo).
  - **Tầng 1 (`world_map`)**: Thiên Khí 100%, Địa Khí 100% (cân bằng tự nhiên chuẩn mực của Ngũ Vực).
  - **Tầng 2 (`bai_thien`)**: Thiên Khí 300%, Địa Khí 30% (dồi dào thanh khí nhưng thiếu hụt địa khí nghiêm trọng).
  - **Tầng 3 (`hac_thien`)**: Thiên Khí 100% (dị biến), Địa Khí 50%. Khi hạ khiếu tại Hắc Thiên từ 5 phút trở lên, thiên khí dị biến sẽ tẩm bổ thêm Đạo Ngân Ám Đạo và Tinh Đạo (`Dark Path` & `Star Path`) trực tiếp vào Tiên Khiếu khi Thu Khiếu.
- Cập nhật cả lazy projection (`GetImmortalAperture`) và nghiệm thu hạ khiếu (`UnanchorAperture`) trong `aperture_core.go` để tính toán chính xác lượng khí hấp thụ thực tế theo thời gian thực.
- Thêm hai vật phẩm tiên đan mới trong `Item.csv`:
  - `EARTH_QI_PILL` (Địa Khí Đan): Nạp trực tiếp +30.000 Địa Khí cho Tiên Khiếu khi bế quan ở Bạch Thiên.
  - `HEAVEN_QI_PILL` (Thiên Khí Đan): Nạp trực tiếp +30.000 Thiên Khí cho Tiên Khiếu khi hạ khiếu dưới Địa Uyên.
  - Đăng ký handlers `restore_earth_qi` và `restore_heaven_qi` trong `item_effects.go` và `block_manifest.go`.

### 2. Kéo Dài Chu Kỳ Tai Kiếp Tại Hắc Thiên (Thiên Ý Khó Cảm Ứng)
- Theo canon Cổ Chân Nhân, bóng đêm và tàn tích Thái Cổ ở Hắc Thiên làm suy giảm khả năng cảm ứng của Thiên Ý, giúp Cổ Tiên trì hoãn tai kiếp.
- Trong `aperture_core.go` và `tribulation_trigger.go`, khi Tiên Khiếu hạ khiếu (`is_placed = true`) tại Hắc Thiên, hệ số tiến trình dòng sông thời gian `effectiveRate` được chia cho 1.5 (`1.0 / 1.5 ~ 0.6667`). Nhờ đó tốc độ tích lũy năm chủ quan chậm hơn 1.5 lần, tương đương chu kỳ tai kiếp được kéo dài thêm đúng 50% thời gian thực tế.

### 3. Cách Ly Tầm Nhìn Môn Hộ (Aperture Portal) Theo Tầng Không Gian
- Trong `map_objects.go` (`aperturePortalMarkersInRegion`), đã bổ sung kiểm tra tầng không gian (`portalLayer := worldMapIDForRegion(regionID)` và `viewerLayer := worldMapIDForRegion(viewerRegion)`):
  - Người chơi đang ở Ngũ Vực (`world_map`) sẽ hoàn toàn không thể nhìn thấy hoặc do thám được Môn Hộ đặt tại Bạch Thiên (`bai_thien`) hay Địa Uyên (`dia_uyen`).
  - Chỉ khi người chơi đích thân vượt Khí Cương Phong Bích lên Bạch Thiên hoặc lặn xuống Địa Uyên (cùng `world_map_id`) thì mới có thể nhìn thấy và kích hoạt cơ chế dò thám Môn Hộ.
  - Chủ Khiếu (`detail.IsMine = true`) luôn có linh giác kết nối với Tiên Khiếu của chính mình nên luôn nhìn thấy Môn Hộ của bản thân trên bản đồ bất kể đang đứng ở tầng nào.


## Question

Làm thế nào để tích hợp cơ chế Hạ Khiếu (Aperture Anchoring / Placement) của Cổ Tiên khi đứng trên Tầng 0 (Địa Uyên) hoặc Tầng 2/3 (Bạch Thiên / Hắc Thiên), tạo ra sự phân hóa rõ rệt về sản lượng Thiên Địa Nhị Khí và chu kỳ Tai Kiếp đúng theo nguyên tác?

Cụ thể cần giải quyết:
1. **Phân hóa Thiên Địa Nhị Khí theo Tầng đặt Tiên Khiếu**:
   - Trong `aperture_core.go` và worker sinh khí:
     - Hạ khiếu tại **Tầng 1 (Ngũ Vực)**: Cân bằng tiêu chuẩn (100% Thiên Khí, 100% Địa Khí).
     - Hạ khiếu tại **Tầng 2 (Bạch Thiên)**: Dồi dào Thiên Khí (300% Thiên Khí), nhưng thiếu hụt Địa Khí (30% Địa Khí) ➔ Buộc Cổ Tiên phải mang theo Địa Khí Đan hoặc định kỳ hạ phàm nạp Địa Khí.
     - Hạ khiếu tại **Tầng 3 (Hắc Thiên)**: Thiên Khí dị biến (Thiên Khí pha trộn Ám Đạo / Tinh Đạo), chu kỳ Tai Kiếp bị kéo dài thêm 50% thời gian (Thiên Ý khó cảm ứng).
     - Hạ khiếu tại **Tầng 0 (Địa Uyên)**: Dồi dào Địa Khí (350% Địa Khí), nhưng cực kỳ hiếm Thiên Khí (20% Thiên Khí) ➔ Rất thích hợp cho các Cổ Tiên tu luyện Thổ Đạo, Luyện Đạo, Kim Đạo, Cốt Đạo.
2. **Hiển thị Môn Hộ theo `world_map_id`**:
   - Bảng `aperture_portals` đã có `region_id`, `x`, `y`.
   - Cần đảm bảo khi Môn Hộ sinh ra trên Bạch Thiên hoặc Địa Uyên, chỉ người chơi đang ở cùng `world_map_id` đó mới nhìn thấy và do thám được Môn Hộ.
