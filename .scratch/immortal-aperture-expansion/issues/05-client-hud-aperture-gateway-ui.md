# Ticket 05: Giao Diện Người Dùng Điều Khiển Môn Hộ và HUD Tác Chiến Tiên Khiếu

Type: prototype
Status: resolved
Assignee: @antigravity
Blocked by: [01-aperture-anchoring-world-gateway](01-aperture-anchoring-world-gateway.md), [03-coop-tribulation-reactive-mutation](03-coop-tribulation-reactive-mutation.md), [04-aperture-lord-sovereignty-traitor-pk](04-aperture-lord-sovereignty-traitor-pk.md)

## Question

Làm thế nào để thiết kế giao diện Client React/Phaser hiển thị Môn Hộ trên bản đồ thế giới, bảng quản lý quyền hạn khách, HUD chỉ số Tiên Khiếu đa người chơi và cảnh báo Thiên Đạo Biến Dị?

Cụ thể cần giải quyết các vấn đề:
1. **Thực Thể Môn Hộ Trên Bản Đồ Thế Giới (World Map Gateway Sprite & Interactions)**:
   - Render cánh cổng Môn Hộ trên bản đồ Phaser 4 tại vị trí `(placed_x, placed_y)`.
   - Hiệu ứng phát quang huyền bí; click vào cổng mở Modal:
     - Với người qua đường: Xem tên chủ khiếu, phẩm cấp phúc địa, lời chào đón, ô nhập mật khẩu hoặc nút "Gửi Thỉnh Cầu Bái Phỏng".
     - Với chủ khiếu: Xem trạng thái nạp Tam Khí, danh sách khách bên trong, nút "Bước Vào Khiếu" hoặc "Thu Khiếu".
2. **Bảng Quản Lý Môn Hộ & Danh Sách Khách (Aperture Access Control Modal)**:
   - UI cho Chủ Khiếu thiết lập chế độ cổng (`OPEN`, `PARTY`, `SECT`, `KEY`, `HIDDEN`).
   - Danh sách theo dõi người chơi đang hiện diện trong Tiên Khiếu kèm theo vai trò (`VISITOR`, `ASSISTANT`, `PRISONER`) và nút thao tác nhanh: Đổi vai trò, Ký Minh Ước Tín Đạo, hoặc Trục Xuất lập tức.
3. **HUD Bên Trong Tiên Khiếu (In-Aperture Multi-player HUD)**:
   - Hiển thị thanh đo Tam Khí (Thiên Khí - Địa Khí - Nhân Khí).
   - Tốc độ sông Quang Âm (chỉ số 1.0x khi đang Hạ Khiếu).
   - Chỉ số Áp Lực Đạo Ngân (`Aperture Dao Stress`) kèm cảnh báo nguy cơ nứt vỡ Tiên Khiếu nếu có khách cấp cao bước vào.
   - Bùa lợi `Duy Ngã Độc Tôn` hiển thị nổi bật trên thanh trạng thái của Chủ Khiếu.
4. **Bảng Dự Báo Thiên Đạo Độ Kiếp (Tribulation Forecast & Danger Meter)**:
   - Giao diện chuẩn bị Độ Kiếp hiển thị danh sách tổ đội trợ trận.
   - Biểu đồ nhiệt nguy cơ: Cảnh báo mức độ cuồng bạo của Thiên Đạo (+X% HP/Công của quái kiếp nạn) và dự đoán loại Kiếp Nạn Biến Dị sắp giáng lâm theo lưu phái của trợ thủ.

## Answer

Đã chuẩn hóa toàn diện 4 thành phần giao diện người dùng Client React & Phaser 4. Kiểm chứng tương tác và trải nghiệm thị giác hoàn tất qua prototype tại [05-client-hud-aperture-ui.html](../prototypes/05-client-hud-aperture-ui.html).

### 1. Thực Thể Môn Hộ Trên Bản Đồ Thế Giới (Phaser 4 World Map Gateway)
- **Tích hợp Phaser Scene**:
  - Khi feed `/explore/map-objects` trả về marker có `Kind: "aperture_portal"`, hệ thống khởi tạo sprite cổng không gian xoay tròn phát quang (`portal-icon`) với hiệu ứng hào quang vàng kim hoặc luân chuyển theo lưu phái chủ đạo của chủ khiếu.
  - Gắn sự kiện `pointerdown` mở Modal tương tác:
    * **Đối với Chủ Khiếu**: Chủ Khiếu luôn ở bên trong Tiên Khiếu khi đang hạ khiếu; qua giao diện quản lý có thể xem thời gian cắm rễ, lượng Tam Khí đã nạp, danh sách khách bên trong, "Cài Đặt Quyền Hạn" và nút "Thu Khiếu" (khi bấm Thu Khiếu, Môn Hộ khép lại, Cổ Tiên xuất hiện trở lại tại tọa độ thế giới).
    * **Đối với Khách / Người Qua Đường**: Xem tên chủ khiếu, phẩm cấp phúc địa. Tùy theo `access_policy`:
      - `OPEN`: Nút "Bước Vào Bái Phỏng".
      - `KEY`: Ô nhập mật khẩu hoặc tiêu hao Thông Hành Đạo Phù.
      - `APPROVAL`: Nút "Gửi Thỉnh Cầu Bái Phỏng" (bắn Toast SSE đến chủ khiếu).

### 2. Modal Quản Lý Môn Hộ & Khách Mời (`ApertureAccessControlModal.tsx`)
- Tách thành component độc lập nhúng trong `ImmortalApertureView.tsx`:
  - **Khối Cài Đặt Môn Hộ**: Dropdown lựa chọn chính sách (`OPEN`, `PARTY`, `SECT`, `KEY`, `HIDDEN`) và input thiết lập mật mã.
  - **Bảng Theo Dõi Khách Mời Thời Gian Thực**:
    * Cột thông tin: Tên Cổ Tiên, Rank cảnh giới, Tọa độ hiện tại trong khiếu.
    * Thẻ vai trò (`GUEST_ROLE` Tag): `VISITOR` (xanh dương), `ASSISTANT` (xanh lá), `PRISONER` (đỏ).
    * Cột Minh Ước Tín Đạo: Thể hiện trạng thái ký kết hợp đồng hòa bình.
    * Nút thao tác: "Đổi Vai Trò", "Gửi Đề Nghị Ký Minh Ước", "Trục Xuất" (Evict) với xác nhận nhanh.

### 3. HUD Đa Người Chơi Bên Trong Tiên Khiếu (`In-Aperture Top Bar`)
- **Bộ Ba Thanh Đo Tam Khí**:
  - Thiên Khí (Thanh xanh thiên thanh), Địa Khí (Thanh hoàng kim), Nhân Khí / Dân Số Dị Nhân (Thanh hồng phấn) kèm chỉ số % độ no.
- **Chỉ Báo Tốc Độ Sông Quang Âm**:
  - Hiển thị nổi bật nhãn `1.0x (Đang Hạ Khiếu - Đồng bộ Ngũ Vực)` hoặc `35.0x (Gia tốc nội tại)`.
- **Huy Hiệu Vương Miện "Duy Ngã Độc Tôn"**:
  - Nổi bật trên thanh trạng thái của Chủ Khiếu với hiệu ứng đổ bóng hoàng kim: Tooltip hiển thị đầy đủ các buff (-50% mana tiêu hao, +25% sát thương, +40% kháng khống chế).
- **Thanh Cảnh Báo Áp Lực Đạo Ngân (Dao Stress Alert Bar)**:
  - Khi có khách Rank cao hơn phẩm cấp Phúc Địa bước vào mà chưa bật sát chiêu thu liễm, thanh cảnh báo nhấp nháy đỏ kèm âm báo cảnh giác, hiển thị lượng HP sụt giảm mỗi giây (-X HP/s) và nút gọi nhanh bảng Trục Xuất.

### 4. Sảnh Chuẩn Bị Độ Kiếp & Biểu Đồ Nhiệt Nguy Cơ (`TribulationCoopLobby.tsx`)
- Tích hợp trong `TribulationRealTimeInfo.tsx`:
  - **Đồng Hồ Đo Nguy Cơ (Danger Heat Meter)**: Thanh đo chuyển màu trực quan thể hiện hệ số phẫn nộ của Thiên Đạo (ví dụ `2.40x - Cực Nguy Hiểm`, hiển thị quái kiếp nạn tăng bao nhiêu % HP và Công).
  - **Hộp Dự Báo Kiếp Nạn Biến Dị (*Nemesis Mutation Forecast*)**:
    - Phân tích lưu phái của trợ thủ và đưa ra cảnh báo chính xác (ví dụ: "Trợ thủ [Hắc Lâu Lan] tu luyện Viêm Đạo -> Thiên Đạo dự kiến giáng hạ Cực Hàn Băng Long kháng 80% lửa!").
  - **Thanh Cam Kết Thù Lao**: Chủ khiếu thiết lập số Tiên Nguyên Thạch thù lao chi trả cho mỗi trợ thủ khi vượt kiếp thành công.
  - **Nút "Dẫn Phát Tai Kiếp"**: Bắt đầu trận chiến đồng bộ cho toàn bộ Cổ Tiên trong khiếu.


## Question

Làm thế nào để thiết kế giao diện Client React/Phaser hiển thị Môn Hộ trên bản đồ thế giới, bảng quản lý quyền hạn khách, HUD chỉ số Tiên Khiếu đa người chơi và cảnh báo Thiên Đạo Biến Dị?

Cụ thể cần giải quyết các vấn đề:
1. **Thực Thể Môn Hộ Trên Bản Đồ Thế Giới (World Map Gateway Sprite & Interactions)**:
   - Render cánh cổng Môn Hộ trên bản đồ Phaser 4 tại vị trí `(placed_x, placed_y)`.
   - Hiệu ứng phát quang huyền bí; click vào cổng mở Modal:
     - Với người qua đường: Xem tên chủ khiếu, phẩm cấp phúc địa, lời chào đón, ô nhập mật khẩu hoặc nút "Gửi Thỉnh Cầu Bái Phỏng".
     - Với chủ khiếu: Xem trạng thái nạp Tam Khí, danh sách khách bên trong, nút "Bước Vào Khiếu" hoặc "Thu Khiếu".
2. **Bảng Quản Lý Môn Hộ & Danh Sách Khách (Aperture Access Control Modal)**:
   - UI cho Chủ Khiếu thiết lập chế độ cổng (`OPEN`, `PARTY`, `SECT`, `KEY`, `HIDDEN`).
   - Danh sách theo dõi người chơi đang hiện diện trong Tiên Khiếu kèm theo vai trò (`VISITOR`, `ASSISTANT`, `PRISONER`) và nút thao tác nhanh: Đổi vai trò, Ký Minh Ước Tín Đạo, hoặc Trục Xuất lập tức.
3. **HUD Bên Trong Tiên Khiếu (In-Aperture Multi-player HUD)**:
   - Hiển thị thanh đo Tam Khí (Thiên Khí - Địa Khí - Nhân Khí).
   - Tốc độ sông Quang Âm (chỉ số 1.0x khi đang Hạ Khiếu).
   - Chỉ số Áp Lực Đạo Ngân (`Aperture Dao Stress`) kèm cảnh báo nguy cơ nứt vỡ Tiên Khiếu nếu có khách cấp cao bước vào.
   - Bùa lợi `Duy Ngã Độc Tôn` hiển thị nổi bật trên thanh trạng thái của Chủ Khiếu.
4. **Bảng Dự Báo Thiên Đạo Độ Kiếp (Tribulation Forecast & Danger Meter)**:
   - Giao diện chuẩn bị Độ Kiếp hiển thị danh sách tổ đội trợ trận.
   - Biểu đồ nhiệt nguy cơ: Cảnh báo mức độ cuồng bạo của Thiên Đạo (+X% HP/Công của quái kiếp nạn) và dự đoán loại Kiếp Nạn Biến Dị sắp giáng lâm theo lưu phái của trợ thủ.
