# Ticket 02: Kiến trúc Không Gian Tiên Khiếu Giới Vực và Ma Trận Phân Quyền Khách

Type: prototype
Status: resolved
Assignee: @antigravity
Blocked by: [01-aperture-anchoring-world-gateway](01-aperture-anchoring-world-gateway.md)

## Question

Làm thế nào để Zone Sim 20Hz khởi tạo và quản lý không gian tiểu giới Tiên Khiếu (`Aperture Domain Instance`) cho nhiều người chơi cùng hiện diện, đồng thời áp đặt ma trận phân quyền và giới hạn Áp Lực Đạo Ngân?

Cụ thể cần giải quyết các vấn đề:
1. **Quản Lý Vòng Đời Vùng Nhớ (Dynamic Instance Management)**:
   - Cơ chế tạo Zone động trong Zone Sim: Zone `aperture:<aperture_id>` được kích hoạt khi có ít nhất 1 người chơi (Chủ hoặc Khách) tiến vào, và hibernate (đóng băng / giải phóng bộ nhớ) an toàn khi toàn bộ người chơi rời khỏi.
   - Cơ chế định tuyến di chuyển chuyển vùng qua `movement.go`: Giao thức an toàn đưa người chơi từ map thế giới xuyên qua Môn Hộ vào tọa độ spawn mặc định trong Tiên Khiếu.
2. **Ma Trận Phân Quyền Khách (Guest Permission Matrix)**:
   - Các vai trò khách mời (`GUEST_ROLE`):
     - `VISITOR` (Khách Thăm Quan): Được tự do đi lại, ngắm cảnh, chat nội bộ, giao dịch P2P với chủ khiếu. Cấm hoàn toàn hành vi thu hoạch điểm tài nguyên (`aperture_resource_points`), cấm sửa đổi bố trí lưới 20x20.
     - `ASSISTANT` (Trợ Thủ / Khách Khanh): Được phép tiếp tế, khai thác điểm tài nguyên theo ủy quyền, vận hành công trình phòng thủ.
     - `PRISONER` (Tù Nhân): Bị phong ấn pháp lực, khóa dùng Cổ trùng và kỹ năng, bị giam giữ tại tọa độ chỉ định.
3. **Áp Lực Đạo Ngân & Trần Chịu Tải Cảnh Giới (Aperture Dao Stress & Rank Ceiling)**:
   - Quy tắc nguyên tác: Cổ Tiên cấp cao chứa đựng lượng Đạo Ngân khổng lồ, khi bước vào Tiên Khiếu cấp thấp sẽ khiến không gian bị quá tải.
   - Công thức tính Stress: Nếu $\text{Rank}_{\text{guest}} > \text{Rank}_{\text{aperture}}$, mỗi giây phát sinh lượng sát thương cấu trúc lên `aperture_hp` trừ khi Khách thi triển sát chiêu Thu Liễm Khí Tức / Đạo Ngân (tiêu hao Tiên Nguyên liên tục).
4. **Quy Chế Trục Xuất & Rời Khỏi Chủ Động (Eviction & Exit Protocol)**:
   - Khách có thể chủ động bước đến vị trí Môn Hộ nội giới để trở lại thế giới thực.
   - Chủ Khiếu có quyền thi triển lệnh Trục Xuất khẩn cấp (*Evict Guest*) tống khứ khách ra ngoài thế giới ngay lập tức (miễn là khách không đang trong trạng thái giao tranh trực tiếp).

## Answer

Đã chuẩn hóa toàn diện kiến trúc quản lý không gian tiểu giới Tiên Khiếu, ma trận phân quyền khách mời và cơ chế tính toán áp lực Đạo Ngân. Kiểm chứng hoạt động hoàn tất qua prototype tương tác tại [02-aperture-instance-and-guest-matrix.html](../prototypes/02-aperture-instance-and-guest-matrix.html).

### 1. Vòng Đời Vùng Nhớ Zone Sim (Lazy Dynamic Instance Lifecycle)
- **Cơ chế tái sử dụng hạ tầng sẵn có**: Tận dụng cơ chế `NewLazyManagerWithMaps` và `RegisterZoneMap` trong `backend/internal/realtime/world/manager.go`.
- **Khóa định danh Zone**:
  ```go
  sim.ZoneKey{
      MapID: "immortal_aperture",
      InstanceID: ap.ID.String(),
      RulesetVersion: "v1",
  }
  ```
- **Kích hoạt & Ngủ đông (Spin-up & Eviction)**:
  - Khi người chơi đầu tiên bước qua Môn Hộ, server gọi `RegisterZoneMap` nạp bản đồ Tiên Khiếu (lưới 20x20, chướng ngại vật, công trình và mỏ tài nguyên).
  - Vòng lặp tick 20Hz khởi chạy tự động khi có Lease.
  - Khi toàn bộ người chơi (Chủ và Khách) rời khỏi Tiên Khiếu, cơ chế eviction của `Manager` tự động đưa zone vào trạng thái ngủ đông (hibernate), giải phóng bộ nhớ đệm, bảo đảm không vượt quá giới hạn `MaxRuntimeZonesPerProcess = 2048`.
- **Định Tuyến Di Chuyển**: Sử dụng chuỗi di chuyển nhất quán trong `handlers/movement.go`:
  - Cập nhật `players.current_region = "aperture:" + ap.ID.String()`.
  - Tọa độ xuất hiện mặc định: (X: 10, Y: 10) ngay vị trí Môn Hộ nội giới.

### 2. Ma Trận Phân Quyền Khách (Guest Permission Matrix)
- **Cơ sở dữ liệu**: Bổ sung bảng `aperture_guest_roster` trong `backend/internal/db/schema_aperture.go`:
  ```sql
  CREATE TABLE IF NOT EXISTS aperture_guest_roster (
      aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
      player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
      role VARCHAR(20) NOT NULL DEFAULT 'VISITOR', -- VISITOR, ASSISTANT, PRISONER
      granted_by UUID NOT NULL REFERENCES players(id),
      entered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      PRIMARY KEY (aperture_id, player_id)
  );
  ```
- **Ràng buộc nghiệp vụ theo vai trò (`GUEST_ROLE`)**:
  - `VISITOR` (Khách Thăm Quan - Mặc định):
    - Được phép: Di chuyển tự do trên lưới, trò chuyện chat nội bộ, giao dịch P2P với Chủ Khiếu.
    - Bị cấm: Thao tác thu hoạch tại `aperture_points.go` bị chặn với mã lỗi `403: err_guest_visitor_forbidden`; cấm can thiệp kiến trúc hoặc kích hoạt cơ quan trận pháp.
  - `ASSISTANT` (Trợ Thủ / Khách Khanh):
    - Được Chủ Khiếu ủy quyền thu hoạch mỏ tài nguyên (sản lượng tự động chuyển vào kho tích lũy Tiên Khiếu hoặc kho cá nhân theo thiết lập).
    - Được phép sửa chữa, nạp Tiên Nguyên vận hành các Tháp Canh/Tường phòng thủ.
  - `PRISONER` (Tù Nhân):
    - Bị áp đặt bùa hại cưỡng chế `SUPPRESSION_SEAL`: Niêm phong hoàn toàn Cổ trùng và Sát chiêu (Silenced), giảm 50% tốc độ di chuyển, không thể tự ý bước qua Môn Hộ rời đi.

### 3. Áp Lực Đạo Ngân & Trần Chịu Tải Cảnh Giới (Aperture Dao Stress & Rank Ceiling)
- **Quy luật nguyên tác**: Cổ Tiên cấp cao sở hữu lượng Đạo Ngân khổng lồ; khi bước vào Phúc Địa cấp thấp, sự xung đột quy tắc sẽ gây chấn động không gian làm nứt vỡ Tiên Khiếu.
- **Công thức tính sát thương cấu trúc**:
  $$\Delta R = \text{Rank}_{\text{guest}} - \text{Grade}_{\text{aperture}}$$
  - Nếu $\Delta R > 0$ và Khách **không bật** sát chiêu thu liễm:
    $$\text{StressDamagePerTick} = \Delta R \times 250 \text{ HP/giây}$$
    Khấu trừ trực tiếp vào `immortal_apertures.aperture_hp`. Client phát cảnh báo chấn động đỏ toàn màn hình.
  - Nếu Khách **kích hoạt** Sát Chiêu "Thu Liễm Đạo Ngân":
    - Áp lực Đạo Ngân triệt tiêu về 0 HP/giây.
    - Khách chịu hao tổn duy trì: $5 \times \Delta R$ Tiên Nguyên/phút. Cạn Tiên Nguyên sẽ tự động hủy thu liễm và áp lực bùng phát trở lại.

### 4. Giao Thức Rời Khỏi & Trục Xuất Khẩn Cấp (Exit & Eviction Protocol)
- **Chủ Động Thoát**: Khách bước vào ô Môn Hộ nội giới (X: 10, Y: 10) chọn "Rời Khỏi" -> Xóa khỏi `aperture_guest_roster`, luân chuyển trở lại vị trí thế giới bên ngoài cổng.
- **Lệnh Trục Xuất Khẩn Cấp (`POST /api/immortal-aperture/guests/:guestId/evict`)**:
  - Chủ Khiếu có thể thực thi bất kỳ lúc nào nếu khách không dính combat.
  - Server xóa record trong `aperture_guest_roster`, dịch chuyển tức thời khách ra ngoài Môn Hộ thế giới và gửi thông báo `GUEST_EVICTED` qua SSE.


## Question

Làm thế nào để Zone Sim 20Hz khởi tạo và quản lý không gian tiểu giới Tiên Khiếu (`Aperture Domain Instance`) cho nhiều người chơi cùng hiện diện, đồng thời áp đặt ma trận phân quyền và giới hạn Áp Lực Đạo Ngân?

Cụ thể cần giải quyết các vấn đề:
1. **Quản Lý Vòng Đời Vùng Nhớ (Dynamic Instance Management)**:
   - Cơ chế tạo Zone động trong Zone Sim: Zone `aperture:<aperture_id>` được kích hoạt khi có ít nhất 1 người chơi (Chủ hoặc Khách) tiến vào, và hibernate (đóng băng / giải phóng bộ nhớ) an toàn khi toàn bộ người chơi rời khỏi.
   - Cơ chế định tuyến di chuyển chuyển vùng qua `movement.go`: Giao thức an toàn đưa người chơi từ map thế giới xuyên qua Môn Hộ vào tọa độ spawn mặc định trong Tiên Khiếu.
2. **Ma Trận Phân Quyền Khách (Guest Permission Matrix)**:
   - Các vai trò khách mời (`GUEST_ROLE`):
     - `VISITOR` (Khách Thăm Quan): Được tự do đi lại, ngắm cảnh, chat nội bộ, giao dịch P2P với chủ khiếu. Cấm hoàn toàn hành vi thu hoạch điểm tài nguyên (`aperture_resource_points`), cấm sửa đổi bố trí lưới 20x20.
     - `ASSISTANT` (Trợ Thủ / Khách Khanh): Được phép tiếp tế, khai thác điểm tài nguyên theo ủy quyền, vận hành công trình phòng thủ.
     - `PRISONER` (Tù Nhân): Bị phong ấn pháp lực, khóa dùng Cổ trùng và kỹ năng, bị giam giữ tại tọa độ chỉ định.
3. **Áp Lực Đạo Ngân & Trần Chịu Tải Cảnh Giới (Aperture Dao Stress & Rank Ceiling)**:
   - Quy tắc nguyên tác: Cổ Tiên cấp cao chứa đựng lượng Đạo Ngân khổng lồ, khi bước vào Tiên Khiếu cấp thấp sẽ khiến không gian bị quá tải.
   - Công thức tính Stress: Nếu $\text{Rank}_{\text{guest}} > \text{Rank}_{\text{aperture}}$, mỗi giây phát sinh lượng sát thương cấu trúc lên `aperture_hp` trừ khi Khách thi triển sát chiêu Thu Liễm Khí Tức / Đạo Ngân (tiêu hao Tiên Nguyên liên tục).
4. **Quy Chế Trục Xuất & Rời Khỏi Chủ Động (Eviction & Exit Protocol)**:
   - Khách có thể chủ động bước đến vị trí Môn Hộ nội giới để trở lại thế giới thực.
   - Chủ Khiếu có quyền thi triển lệnh Trục Xuất khẩn cấp (*Evict Guest*) tống khứ khách ra ngoài thế giới ngay lập tức (miễn là khách không đang trong trạng thái giao tranh trực tiếp).
