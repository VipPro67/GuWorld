Type: research
Status: resolved

## Question

Xác định quy chuẩn cho Vật Phẩm Dẫn Dắt / Tiêu Bản Nghiên Cứu (Catalyst):
1. Điều kiện của Cổ Phôi hiến tế: Cổ trùng cùng hệ có Rank <= Rank Cổ đích, kiểm tra tình trạng trong Không Khiếu (không đang trang bị trận hình, không phải Cổ bản mệnh).
2. Vật Liệu Thay Thế: Nếu không dùng Cổ phôi, cho phép dùng bao nhiêu nguyên liệu đặc trưng cùng hệ (ví dụ khoáng thạch/thảo dược cùng hệ thuộc Rank tương ứng) để làm vật dẫn.
3. Trọng số bonus chính xác: Mức tăng % tỉ lệ trúng đích khi có Cổ phôi (ví dụ +25%) so với nguyên liệu thường (+15%), đảm bảo không giảm chi phí Niệm Lực cố định theo quyết định thiết kế.

## Answer

### 1. Điều Kiện Cổ Phôi Hiến Tế (Specimen Gu Validation)
Khi người chơi chọn dùng Cổ trùng sống làm tiêu bản mổ xẻ cấu trúc đạo ngân:
- **Tồn tại trong Không Khiếu**: Kiểm tra qua `FindGuByInstanceID` trong `player.GuInventory`.
- **An toàn bảo vệ**:
  - `!gu.IsVital`: Nghiêm cấm dùng Cổ Bản Mệnh.
  - `!gu.Equipped`: Cổ không được ở trạng thái đang trang bị (phải nằm trong kho/túi).
  - `!gu.IsLocked`: Cổ không bị khóa chống thao tác nhầm.
  - `gu.Status != "DEAD"`: Cổ chưa bị tử vong (phải là Cổ sống để cảm thụ mạch đạo ngân tự nhiên).
- **Tương thích Cổ đích**:
  - Cùng Lưu Phái: `cleanPath(gu.Path) == cleanPath(targetGu.Path)`.
  - Phẩm giai: `gu.Rank <= targetRank`.
- **Hệ quả tiêu hao**: Khi bắt đầu phiên khai sáng, Cổ phôi này sẽ bị tiêu hao hoàn toàn (loại bỏ khỏi `player.GuInventory`).

### 2. Vật Liệu Dẫn Dắt Thay Thế (Material Catalyst Rules)
Nếu người chơi không muốn hiến tế Cổ sống:
- **Danh mục hợp lệ**:
  - Thuộc loại `Material` trong `item_dictionary`.
  - Có `path` trùng khớp với Lưu phái chính của Cổ đích.
  - Phẩm giai `rank <= targetRank`.
- **Định lượng tiêu hao theo Rank**:
  - Cùng Rank (`material.Rank == targetRank`): Tiêu hao **10** đơn vị (Phàm phẩm R1-5) hoặc **5** đơn vị (Tiên tài R6+).
  - Kém 1 Rank (`material.Rank == targetRank - 1`): Tiêu hao **25** đơn vị.
  - Kém từ 2 Rank trở lên: Tiêu hao **50** đơn vị.

### 3. Trọng Số Bonus Tỉ Lệ Trúng Đích (`catalystBonus`)
- **Không dùng vật dẫn**: `catalystBonus = 0.0%`.
- **Dùng Nguyên Liệu cùng hệ**:
  - Cùng Rank: `catalystBonus = +12.0%`.
  - Thấp hơn Rank: `catalystBonus = +8.0%`.
- **Dùng Cổ Phôi sống cùng hệ**:
  - Cùng Rank: `catalystBonus = +20.0%`.
  - Kém 1 Rank: `catalystBonus = +16.0%`.
  - Kém từ 2 Rank trở lên: `catalystBonus = +12.0%`.
- **Quy tắc chi phí**:
  - Vật dẫn **CHỈ** gia tăng tỉ lệ trúng đích Cổ mong muốn, **KHÔNG** làm giảm chi phí Niệm Lực và Nguyên Thạch / Tiên Thạch cố định theo Rank Cổ đích.

