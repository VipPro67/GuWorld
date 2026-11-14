# Ticket 06: Cơ Chế Thu Hoạch Kiếp Tài, Dư Ba Môi Trường & Thanh Tẩy Ô Nhiễm Đạo Ngân

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để kết toán giai đoạn hậu độ kiếp (Phase 3: `AFTERMATH`) một cách trọn vẹn, bao gồm việc **rơi rớt Kiếp Tài quý hiếm trên lưới ô Tiên Khiếu**, mô phỏng **Dư Ba Tổn Thất Sinh Thái**, và cung cấp công cụ cho Cổ Tiên **Thanh Tẩy Ô Nhiễm Đạo Ngân**?

Cụ thể cần giải quyết:
1. **Kiếp Tài Rơi Vãi Trên Lưới Ô (Grid-based Spoils Drops)**:
   - Thay vì chỉ cộng điểm số vào thuộc tính, các ô diễn ra kiếp nạn kết tinh ra các vật phẩm Kiếp Tài quý hiếm (Lôi tinh, Băng phách, Viêm hạch, Tàn hồn thần ma...).
   - Người chơi (hoặc Dị nhân thu thập) tương tác trên bản đồ để thu hoạch Kiếp Tài.
   - Bảng ánh xạ Kiếp Tài với công thức Luyện Cổ Tiên cấp và Nâng cấp Tiên Cổ Phòng.
2. **Mô Phỏng Tổn Thất & Ô Nhiễm Đạo Ngân (Fallout & Contamination)**:
   - Các ô tài nguyên bị thiên tai/thú triều quét qua bị biến tính (ví dụ: mỏ linh mộc bị cháy thành than, hồ nước bị nhiễm độc).
   - Đạo Ngân dị chủng của kiếp nạn cắm rễ vào vách khiếu: làm biến đổi tỷ lệ sản sinh tài nguyên tự nhiên của Tiên Khiếu nếu không dọn dẹp.
3. **Thao Tác Tái Thiết & Thanh Tẩy (Cleanse & Ecological Healing)**:
   - Giao diện thao tác cho Cổ Tiên: tiêu hao Tiên Nguyên Thạch / Sát Chiêu Thanh Tẩy để gột rửa Đạo Ngân tạp chất, hồi phục sinh lực đất đai.
   - Cơ chế "Biến Nguy Thành Cơ": Cổ Tiên có thể giữ lại một phần Đạo Ngân dị chủng nếu muốn chuyển đổi hệ sinh thái Tiên Khiếu sang hướng đa dạng hoặc khai thác loại tài nguyên mới.

## Answer

Đã hoàn thành thiết kế và kiểm chứng trọn vẹn cơ chế giai đoạn hậu độ kiếp (Phase 3: `AFTERMATH`):

### 1. Cơ Chế Rơi Rớt Kiếp Tài Trên Lưới Ô (Grid-based Spoils Drops)
- **Cơ chế Spawn**:
  Khi kiếp nạn chuyển trạng thái từ `ORDEAL_ACTIVE` sang `AFTERMATH`:
  - Dữ liệu `tribulation_events.hazard_metadata` hoặc bảng tạm thời lưu danh sách các tọa độ `(x, y)` nơi thiên tai bị dập tắt hoặc quái vật bị hạ sát.
  - Mỗi tọa độ kết tinh thành một **Điểm Kiếp Tài (Spoils Drop)** với vật phẩm tương ứng theo Rank và Lưu Phái (tham chiếu trực tiếp `Item.csv` đã khảo cứu ở Ticket 07):
    * Băng Tuyết: `ice_crystal_rank6` (Vạn Niên Hàn Băng Tinh).
    * Liệt Hỏa: `fire_crystal_rank6` (Hỏa Diễm Tinh Hạch).
    * Thiên Tiêu Lôi Hải: `lightning_crystal_rank7` (Thần Tiêu Lôi Tinh), `heaven_tribulation_shard_rank7` (Mảnh Vỡ Thiên Kiếp).
    * Thú Vương / Cốt Lang: `wolf_king_fang_rank6`, `metal_ore_rank6` (Huyền Thiết Bì Cương).
- **Thao tác Thu Hoạch**:
  - **Manual Claim**: Cổ Tiên nhấp chuột vào từng ô kiếp tài trên giao diện Tiên Khiếu (`ImmortalApertureView.tsx`) để thu thập trực tiếp vào kho đồ cá nhân.
  - **Dị Nhân Thu Gom (Auto-gather)**: Cổ Tiên sở hữu quần thể Dị Nhân (Mao Dân, Tuyết Dân, Đản Dân...) có thể kích hoạt nút "Phái Dị Nhân Càn Quét" để gom toàn bộ kiếp tài trên lưới ô chỉ trong 1 thao tác.
- **Giá Trị Sử Dụng**:
  - Kiếp tài là nguyên liệu Tiên phẩm bắt buộc trong các công thức Luyện Tiên Cổ (`Recipe.csv`) cùng Rank.
  - Dùng để đột phá và nâng cấp kết cấu Tiên Cổ Phòng (`gu_houses`).

### 2. Mô Phỏng Dư Ba Tổn Thất & Đạo Ngân Ô Nhiễm (Fallout & Contamination)
- **Tổn Thất Mỏ Tài Nguyên (`aperture_resource_points`)**:
  - Các mỏ tài nguyên nằm trong phạm vi quét qua của kiếp nạn bị tụt `vitality` từ 30% đến 80%.
  - Mỏ có vitality $< 20\%$ chuyển sang trạng thái hư hại `DAMAGED`, tạm dừng chu kỳ sản xuất tự động cho đến khi được sửa chữa.
- **Ô Thổ Nhưỡng Bị Nhiễm Đạo Ngân (Contaminated Soil Cells)**:
  - Các ô đất trống nơi kiếp nạn quét qua bị Đạo Ngân dị chủng cắm rễ sâu vào vách khiếu, biến dị thành:
    * `LAVA_CRUST` (Vỏ dung nham nóng chảy) do Hỏa Kiếp.
    * `PERMAFROST_MIRE` (Băng đầm lầy cực hàn) do Băng/Thủy Kiếp.
    * `TOXIC_BOG` (Vũng bùn độc địa) do Độc Kiếp.
  - **Hệ quả sinh thái**: Mỗi ô nhiễm làm giảm $4\%$ đến $5\%$ chỉ số `Ecological Health` của Tiên Khiếu. Đạo Ngân dị chủng xung đột gây áp chế vi khí hậu, làm chậm tốc độ hồi phục tự nhiên của các mỏ lân cận trong bán kính 2 ô.

### 3. Cơ Chế Tái Thiết: Thanh Tẩy vs Biến Nguy Thành Cơ (Cleanse vs Ecological Pivot)
Cổ Tiên có 2 hướng chiến lược xử lý từng ô đất ô nhiễm:
- **Hướng 1 - Thanh Tẩy Gột Rửa (`CLEANSE`)**:
  - Tiêu hao: $50$ Tiên Nguyên Thạch (hoặc kích hoạt Sát Chiêu Tịnh Hóa như *Tẩy Trần Cam Lộ*).
  - Tác dụng: Gột rửa sạch sẽ toàn bộ Đạo Ngân dị chủng, đưa ô đất trở về trạng thái màu mỡ nguyên bản (`NORMAL_SOIL`), hồi phục chỉ số sinh thái và bảo vệ tính thuần nhất của lưu phái chủ tu.
- **Hướng 2 - Biến Nguy Thành Cơ (`ECOLOGICAL_PIVOT`)**:
  - Triết lý chuẩn nguyên tác Cổ Chân Nhân: Cổ Tiên kiệt xuất không xem tai kiếp là gánh nặng mà xem là cơ hội trời ban để mở rộng tài nguyên sinh thái!
  - Tiêu hao: $120$ Tiên Nguyên Thạch cùng tài nguyên xây dựng.
  - Tác dụng: Dung hợp Đạo Ngân kiếp nạn, chuyển hóa vĩnh viễn ô đất ô nhiễm thành một **Mỏ Tài Nguyên Đặc Thù Mới** (`aperture_resource_points`) thuộc hệ tương ứng (ví dụ: dung nham biến thành *Mỏ Viêm Tinh Khoáng Mạch*, đầm độc biến thành *U Độc Dược Điền*).

### 4. Kiểm Chứng Logic Hoàn Tất Qua Prototype
- Toàn bộ chu trình từ rơi rớt Kiếp Tài, Dị Nhân thu gom một nút, thanh lọc ô nhiễm và quy hoạch mở mỏ tài nguyên mới đã được hiện thực hóa và kiểm chứng hoạt động trực quan tại prototype:
  [06_tribulation_spoils_and_fallout_demo.html](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/tribulation-multiform-system/prototypes/06_tribulation_spoils_and_fallout_demo.html).
