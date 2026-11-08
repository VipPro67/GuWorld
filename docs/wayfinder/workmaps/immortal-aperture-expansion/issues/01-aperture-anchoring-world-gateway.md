# Ticket 01: Quy chuẩn Môn Hộ Hạ Khiếu và Vòng Đời Hiện Diện Thế Giới

Type: prototype
Status: resolved
Assignee: @antigravity
Blocked by:

## Question

Làm thế nào để Zone Sim 20Hz và Backend thiết lập luồng Hạ Khiếu (`AnchorAperture`) sinh thực thể Môn Hộ (`World Portal Entity`) trên bản đồ thế giới Ngũ Vực an toàn, quản lý chính sách cổng (Open/Party/Sect/Key/Hidden) và xử lý thu khiếu an toàn?

Cụ thể cần giải quyết các vấn đề:
1. **Khởi tạo Môn Hộ Thực Thể (World Portal Entity Lifecycle)**:
   - Khi Cổ Tiên thực hiện `AnchorAperture`, làm sao để Zone Sim tại vùng đó (`placed_region`) sinh ra một thực thể cổng có tọa độ `(placed_x, placed_y)`?
   - Cấu trúc bản ghi lưu trữ cổng (`aperture_portals` hoặc mở rộng từ `immortal_apertures`) và cách đồng bộ qua Redis/Zone Manager.
2. **Chính Sách Kiểm Soát Môn Hộ (Gateway Access Policy)**:
   - Hỗ trợ các chế độ cổng: `OPEN` (mở rộng đón khách), `PARTY` (chỉ thành viên tổ đội), `SECT` (chỉ đồng môn trong Tông Môn), `KEY` (yêu cầu mật mã/đạo phù), và `HIDDEN` (ẩn thân).
   - Quy trình xác thực khi một người chơi bên ngoài tương tác với cổng: kiểm tra điều kiện, gửi yêu cầu xin vào tới chủ khiếu (Request Prompt) nếu ở chế độ xét duyệt.
3. **Ẩn Thân Thiên Cơ (Concealment Integration)**:
   - Tương tác với module Trinh Sát & Che Giấu Thiên Cơ: Nếu Cổ Tiên sử dụng sát chiêu Ám Đạo/Huyễn Đạo/Trí Đạo để che giấu môn hộ, người ngoài phải có điểm Trinh Sát cao hơn điểm Che Giấu mới thấy được cổng trên bản đồ.
4. **Trạng Thái Thân Thể Chủ Khiếu (Host Presence Options)**:
   - Lựa chọn khi Hạ Khiếu: Chủ khiếu có thể lựa chọn bước vào trong Tiên Khiếu của mình (nhập vào instance), hoặc đứng ngoài thế giới hộ vệ Môn Hộ.
   - Xử lý tọa độ xuất hiện khi bước ra khỏi Tiên Khiếu.
5. **Quy Chế Thu Khiếu An Toàn (Unanchor Aperture Validation)**:
   - Các điều kiện ràng buộc ngặt nghèo trước khi cho phép `UnanchorAperture`: Không thể thu khiếu nếu còn người lạ bên trong, không thể thu khi đang trong trạng thái giao tranh (In Combat) hoặc đang độ kiếp (Tribulation Active).

## Answer

Đã chuẩn hóa toàn diện mô hình máy trạng thái (State Machine) và kiến trúc backend cho Môn Hộ Hạ Khiếu & Hiện Diện Thế Giới. Kiểm chứng hoạt động hoàn tất qua prototype tương tác tại [01-aperture-gateway-lifecycle.html](../prototypes/01-aperture-gateway-lifecycle.html).

### 1. Cơ Sở Dữ Liệu & Vòng Đời Môn Hộ (Data DDL & World Presence)
- Thêm bảng chuyên trách `aperture_portals` trong `backend/internal/db/schema_aperture.go`:
  ```sql
  CREATE TABLE IF NOT EXISTS aperture_portals (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
      owner_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
      region_id VARCHAR(50) NOT NULL REFERENCES regions(id),
      x DOUBLE PRECISION NOT NULL,
      y DOUBLE PRECISION NOT NULL,
      name VARCHAR(100) NOT NULL,
      access_policy VARCHAR(20) NOT NULL DEFAULT 'OPEN', -- OPEN, PARTY, SECT, KEY, APPROVAL, HIDDEN
      access_key_hash VARCHAR(255),
      concealment_power INT NOT NULL DEFAULT 0,
      is_active BOOLEAN NOT NULL DEFAULT TRUE,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      CONSTRAINT uq_aperture_active_portal UNIQUE (aperture_id)
  );
  CREATE INDEX IF NOT EXISTS idx_aperture_portals_region ON aperture_portals (region_id, is_active);
  ```
- Tích hợp với `map_objects.go`: Bổ sung `aperturePortalMarkersInRegion(...)` vào feed marker thế giới (`/explore/map-objects`), định danh marker `Kind: "aperture_portal"`.

### 2. Ma Trận Chính Sách Kiểm Soát Môn Hộ (Access Policy Matrix)
- `OPEN`: Mọi Cổ Tiên/Cổ Sư đứng cách cổng $\le 1.5$ ô đều có thể tương tác chọn "Bước Vào Tiên Khiếu".
- `PARTY`: Chỉ người chơi có chung `party_id` với Chủ Khiếu mới được phép qua cổng.
- `SECT`: Kiểm tra `players.sect_id == portal.owner_sect_id`.
- `KEY`: Yêu cầu client cung cấp mã khoá hoặc tiêu hao 1 vật phẩm "Thông Hành Đạo Phù" trong túi đồ.
- `APPROVAL`: Khi bấm vào cổng, client phát sự kiện SSE gửi yêu cầu đến Chủ Khiếu: Toast xác nhận với 2 nút [Đồng Ý] / [Từ Chối].
- `HIDDEN`: Đi kèm với điểm che giấu thiên cơ.

### 3. Tích Hợp Trinh Sát & Che Giấu Thiên Cơ (Concealment Engine)
- Giá trị `concealment_power` của Môn Hộ được tự động cập nhật khi Hạ Khiếu dựa trên:
  $$\text{ConcealmentScore} = \text{BaseKillerMoveScore} + (\text{DaoMarks}_{\text{Dark/Illusion/Wisdom}} \times 0.05)$$
- Khi trả về danh sách marker trên bản đồ thế giới, server thực thi bộ lọc bảo mật:
  - Nếu `portal.concealment_power > player.ScoutingScore`, Môn Hộ bị giấu hoàn toàn khỏi response JSON (chống gian lận memory injection phía client).

### 4. Nhất Thể Đồng Nhất Cổ Tiên & Tiên Khiếu (Inseparable Host-Aperture Unity)
- **Quy tắc cốt truyện Cổ Chân Nhân**: Tiên Khiếu và Cổ Tiên là một thể thống nhất, không bao giờ tách rời:
  - **Trạng thái bình thường**: Tiên Khiếu nằm bên trong cơ thể Cổ Tiên (vị trí không khiếu cũ dưới rốn ba thốn). Trên bản đồ thế giới **hoàn toàn KHÔNG có** Tiên Khiếu hay Môn Hộ của người chơi. Cổ Tiên tự do mang theo tiểu thế giới của mình chu du khắp Ngũ Vực.
  - **Khi Hạ Khiếu (`AnchorAperture`)**:
    - Tiên Khiếu cắm rễ vào địa mạch thế giới tại tọa độ hiện tại `(placed_region, placed_x, placed_y)`.
    - Tại tọa độ đó trên bản đồ thế giới xuất hiện **Môn Hộ (Aperture World Gateway)**.
    - Nhục thân Cổ Tiên **lập tức chuyển dịch vào BÊN TRONG Tiên Khiếu** để tọa trấn.
    - Cổ Tiên **TUYỆT ĐỐI KHÔNG THỂ** đi lại ngoài thế giới khi đang Hạ Khiếu (triệt tiêu hoàn toàn nghịch lý phân thân).
  - **Khi muốn trở lại thế giới thực**:
    - Bắt buộc phải thực hiện **Thu Khiếu (`UnanchorAperture`)**.
    - Khi thu khiếu thành công: Môn Hộ thế giới khép lại và biến mất, Tiên Khiếu thu gọn trở lại vào nhục thân, và Cổ Tiên xuất hiện trở lại bản đồ thế giới tại đúng tọa độ `(placed_x, placed_y)`.

### 5. Khóa An Toàn Tuyệt Đối Khi Thu Khiếu (Unanchor Safety Locks)
Hàm `UnanchorAperture` trong `aperture_core.go` áp đặt 3 chốt kiểm duyệt nghiêm ngặt:
1. **Khóa Ngoại Nhân**: Kiểm tra số lượng người chơi đang hiện diện trong Tiên Khiếu `guest_count == 0`. Nếu còn khách, reject mã lỗi `400: aperture_has_active_guests` (nguyên tác: không thể thu khiếu khi có người lạ vì Đạo Ngân ngoại lai sẽ xé toạc vách ngăn tiểu thế giới).
2. **Khóa Giao Tranh / Tai Kiếp**: Reject nếu `in_combat == true` hoặc `tribulation_active == true`.
3. **Hoàn Nguyên Trạng Thái**:
   - Vô hiệu hoá bản ghi `aperture_portals` (`is_active = false`).
   - Khôi phục tốc độ sông Quang Âm từ 1.0x trở lại hệ số gia tốc tự thân (`time_rate`).
   - Tự động di dời Chủ Khiếu xuất hiện an toàn tại tọa độ `(placed_x, placed_y)` trên bản đồ thế giới nếu đang ở trong khiếu.


## Question

Làm thế nào để Zone Sim 20Hz và Backend thiết lập luồng Hạ Khiếu (`AnchorAperture`) sinh thực thể Môn Hộ (`World Portal Entity`) trên bản đồ thế giới Ngũ Vực an toàn, quản lý chính sách cổng (Open/Party/Sect/Key/Hidden) và xử lý thu khiếu an toàn?

Cụ thể cần giải quyết các vấn đề:
1. **Khởi tạo Môn Hộ Thực Thể (World Portal Entity Lifecycle)**:
   - Khi Cổ Tiên thực hiện `AnchorAperture`, làm sao để Zone Sim tại vùng đó (`placed_region`) sinh ra một thực thể cổng có tọa độ `(placed_x, placed_y)`?
   - Cấu trúc bản ghi lưu trữ cổng (`aperture_portals` hoặc mở rộng từ `immortal_apertures`) và cách đồng bộ qua Redis/Zone Manager.
2. **Chính Sách Kiểm Soát Môn Hộ (Gateway Access Policy)**:
   - Hỗ trợ các chế độ cổng: `OPEN` (mở rộng đón khách), `PARTY` (chỉ thành viên tổ đội), `SECT` (chỉ đồng môn trong Tông Môn), `KEY` (yêu cầu mật mã/đạo phù), và `HIDDEN` (ẩn thân).
   - Quy trình xác thực khi một người chơi bên ngoài tương tác với cổng: kiểm tra điều kiện, gửi yêu cầu xin vào tới chủ khiếu (Request Prompt) nếu ở chế độ xét duyệt.
3. **Ẩn Thân Thiên Cơ (Concealment Integration)**:
   - Tương tác với module Trinh Sát & Che Giấu Thiên Cơ: Nếu Cổ Tiên sử dụng sát chiêu Ám Đạo/Huyễn Đạo/Trí Đạo để che giấu môn hộ, người ngoài phải có điểm Trinh Sát cao hơn điểm Che Giấu mới thấy được cổng trên bản đồ.
4. **Trạng Thái Thân Thể Chủ Khiếu (Host Presence Options)**:
   - Lựa chọn khi Hạ Khiếu: Chủ khiếu có thể lựa chọn bước vào trong Tiên Khiếu của mình (nhập vào instance), hoặc đứng ngoài thế giới hộ vệ Môn Hộ.
   - Xử lý tọa độ xuất hiện khi bước ra khỏi Tiên Khiếu.
5. **Quy Chế Thu Khiếu An Toàn (Unanchor Aperture Validation)**:
   - Các điều kiện ràng buộc ngặt nghèo trước khi cho phép `UnanchorAperture`: Không thể thu khiếu nếu còn người lạ bên trong, không thể thu khi đang trong trạng thái giao tranh (In Combat) hoặc đang độ kiếp (Tribulation Active).
