# Ticket 01: Chuẩn Hóa Schema Database World Maps Bổ Sung Tầng 0 (Địa Uyên) & Mở Khóa Nhị Thiên

Type: task
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để mở rộng bảng `world_maps` và `region_world_maps` trong `backend/internal/db/schema_world.go` nhằm hỗ trợ đầy đủ cấu trúc 4 tầng thế giới (Tầng 0: Địa Uyên, Tầng 1: Ngũ Vực, Tầng 2: Bạch Thiên, Tầng 3: Hắc Thiên) mà không phá vỡ tính tương thích ngược với dữ liệu bản đồ hiện tại?

Cụ thể cần giải quyết:
1. **Bổ sung Tầng 0 (`dia_uyen`) vào bảng `world_maps`**:
   - `id`: `'dia_uyen'`
   - `name`: `'Địa Uyên · Tầng 0'`
   - `description`: `'Thế giới vực sâu lòng đất dưới chân Ngũ Vực, nơi đan xen địa câu, địa mạch, khoáng thạch và ma quái u tối.'`
   - `width`: 30000, `height`: 20000
   - `min_rank`: 5, `max_rank`: 9
   - `is_active`: `TRUE`
2. **Kích hoạt Tầng 2 (`bai_thien`) và Tầng 3 (`hac_thien`)**:
   - Chuyển `is_active` từ `FALSE` thành `TRUE`.
   - Cập nhật mô tả chuẩn nguyên tác:
     - `bai_thien`: `'Thái Cổ Bạch Thiên · Tầng 2 - Cõi trời ánh sáng vĩnh hằng, biển mây bát ngát, chốn ngự trị của chim hạc và Động Thiên chính đạo.'` (Rank 6–9).
     - `hac_thien`: `'Thái Cổ Hắc Thiên · Tầng 3 - Cõi trời bóng đêm vô tận, thế giới của Hồn thú Thái cổ, mộng cảnh trôi dạt và cứ địa Dị Nhân ẩn náu.'` (Rank 8–9).
3. **Mở rộng Admin Catalog & Memory Cache**:
   - Đảm bảo `InitWorldCache` và các endpoint lấy danh sách bản đồ (`/api/world/maps`, `/api/admin/world/regions`) nạp đầy đủ cả 4 tầng thế giới.
   - Bỏ thuộc tính `disabled` trên dropdown `AdminView.tsx` để Admin có thể gán `world_map_id` cho các region thuộc `dia_uyen`, `bai_thien`, `hac_thien`.

## Answer

Đã hoàn thành chuẩn hóa schema database và nạp 4 tầng thế giới:

1. **Schema DDL & Compatibility Seed (`backend/internal/db/schema_world.go`)**:
   - Bổ sung `dia_uyen` (Địa Uyên · Tầng 0) với kích thước `30000x20000`, cảnh giới Rank 5–9, `is_active = TRUE`.
   - Kích hoạt `bai_thien` (Thái Cổ Bạch Thiên · Tầng 2, Rank 6–9) và `hac_thien` (Thái Cổ Hắc Thiên · Tầng 3, Rank 8–9) với mô tả chi tiết chuẩn nguyên tác.
   - Cập nhật DDL dạng idempotent (`ON CONFLICT (id) DO UPDATE SET ...` và các câu lệnh `UPDATE` tương thích ngược) để các cơ sở dữ liệu đã khởi tạo sẵn lập tức nhận cập nhật trạng thái `is_active = TRUE` và mô tả mới.

2. **Sắp xếp thứ bậc tầng thế giới (`backend/internal/handlers/exploration_world.go`)**:
   - Cập nhật câu truy vấn trong `GetWorldMaps` theo thứ tự từ thấp lên cao:
     `CASE id WHEN 'dia_uyen' THEN 0 WHEN 'world_map' THEN 1 WHEN 'bai_thien' THEN 2 WHEN 'hac_thien' THEN 3 ELSE 4 END`.

3. **Mở route API cho World Layers (`backend/internal/server/server.go`)**:
   - Đăng ký `GET /api/world/maps` cho người chơi.
   - Đăng ký `GET /api/admin/world-maps` trong danh mục routes quản trị.

4. **Kích hoạt gán tầng trong Admin UI (`frontend/src/components/AdminView.tsx`)**:
   - Thêm option `dia_uyen` (Địa Uyên · Tầng 0) vào dropdown chọn tầng bản đồ của khu vực.
   - Bỏ thuộc tính `disabled` cho `bai_thien` và `hac_thien`, cho phép quản trị viên phân bổ khu vực vào cả 4 tầng.

5. **Đồng bộ tài liệu**:
   - Cập nhật [API.md](../../guides/API.md) và [world-and-exploration.md](../../features/world-and-exploration.md).
   - Kiểm tra toàn diện test suite (`go test ./...` và `npm run typecheck`, `npm run lint`) đều đạt 100%.
