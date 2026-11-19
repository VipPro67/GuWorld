# Ticket 02: Cơ Chế Vượt Khí Cương Phong Bích Thăng Thiên & Thám Hiểm Địa Câu Xuống Địa Uyên

Type: research
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để thiết kế và hiện thực hóa cơ chế di chuyển xuyên tầng (Layer Transition) trong `backend/internal/handlers/movement.go` và `map_cells` tuân thủ nghiêm ngặt quy luật tu vi và sinh tử của nguyên tác Cổ Chân Nhân?

Cụ thể cần giải quyết:
1. **Khí Cương Phong Bích (Mặt đất ➔ Bạch Thiên)**:
   - Các điểm thông đạo trên đỉnh núi cao Ngũ Vực (Thiên Đề Sơn, Phong Bạo Đỉnh, Băng Phong Hắc Uyên, Phi Sa Cự Hác...).
   - Luật thẩm tra tu vi:
     - `Rank < 6`: Không có Tiên Thể, bị bão gió Cương Phong xé rách (nhận sát thương 90% Max HP, nếu HP thấp sẽ tử vong, văng ngược trở lại chân núi).
     - `Rank >= 6` (Cổ Tiên): Vượt qua tầng gió an toàn, tiêu hao một lượng Chân Nguyên/Tiên Nguyên hoặc HP khiên, dịch chuyển thành công lên tọa độ tương ứng trên `bai_thien`.
     - Hỗ trợ Tiên Cổ Ốc: Người ngồi trong Tiên Cổ Ốc (kể cả phàm nhân) được bảo hộ 100% khi bay qua Cương Phong Bích.
2. **Cực Điểm Không Gian (Bạch Thiên ➔ Hắc Thiên)**:
   - Cổng lốc xoáy sao chổi (Astral Vortex) tại ranh giới cao nhất của Bạch Thiên.
   - Yêu cầu tu vi: `Rank >= 8` (Bát chuyển Động Thiên Cổ Tiên hoặc Thất chuyển mang Tiên Cổ Phòng cấp 8). Thất chuyển bình thường bước vào chịu áp lực không gian khổng lồ.
3. **Địa Câu Nứt Nẻ & Âm Lưu Cự Hác (Mặt đất ➔ Địa Uyên)**:
   - Lối xuống đặt tại các rãnh vực sâu nứt toác của Ngũ Vực (Trung Châu Địa Câu, Âm Lưu Cự Hác Nam Cương, Hủ Độc Chiểu Trạch...).
   - Yêu cầu tu vi: `Rank >= 5` (Ngũ chuyển đỉnh phong hoặc Cổ Tiên). Phàm nhân cấp thấp chịu chướng khí ngạt thở và áp lực lòng đất.
4. **Cơ chế Hồi Quy / Rơi Tự Do (Descent Mechanics)**:
   - Các điểm "Rơi Khỏi Tầng Trời" (Skyfall / Void Drop): Cổ Tiên có thể lao mình xuống dưới, xuyên qua Cương Phong Bích để đáp xuống vị trí địa lý tương ứng ở tầng dưới (dựa trên tỷ lệ chiếu tọa độ 1:1 giữa các tầng).

## Answer

Đã hoàn thành thiết kế kiến trúc và hiện thực hóa toàn diện cơ chế di chuyển xuyên tầng trong backend:

### 1. Kiến trúc Cốt Lõi: Tách biệt Giới Bích Ngang và Tầng Thế Giới Dọc
- **Kênh điều phối tập trung (`backend/internal/handlers/movement.go`)**:
  - `checkMovement` bổ sung `checkLayerTransitionGuard`: Chạy trước khi cho phép nhân vật bước qua ranh giới giữa 2 vùng khác `world_map_id`.
  - `applyDestination`: Khi chuyển vùng có `fromMap != toMap`, gọi `applyLayerTransitionEffects` để thanh toán độ bền Tiên Cổ Ốc, trừ Chân Nguyên/HP và sinh văn bản chuyển tầng thay vì áp dụng Giới Vực Bích thông thường (`applyRegionalWallDebuff`).

### 2. Chi tiết Quy tắc Chuyển Tầng (`backend/internal/handlers/movement_layers.go`)
1. **Khí Cương Phong Bích (Ngũ Vực ➔ Thái Cổ Bạch Thiên)**:
   - Kiểm tra Tiên Cổ Ốc đang trang bị (`player_gu_houses` với `is_equipped = true` và `current_durability > 0`).
   - Nếu `Rank < 6` và không có Tiên Cổ Ốc:
     - Gây sát thương cuồng bạo bằng 90% Max HP (`playerEffectiveMaxHP`).
     - Gán trạng thái `severely_wounded` trong 30 phút.
     - Persist sát thương ra DB (`UPDATE players SET hp=...`) ngay cả khi transaction chuyển vùng bị rollback, từ chối di chuyển với `http.StatusForbidden` và cảnh báo văng trở lại.
   - Nếu có Tiên Cổ Ốc: Tiêu hao 20 điểm độ bền Cổ Ốc và 10% phàm cổ, bảo hộ phàm nhân phi thăng an toàn.
   - Nếu `Rank >= 6` (Cổ Tiên): Tiên Thể lướt gió an toàn; trừ 50 Chân Nguyên (nếu thiếu Chân Nguyên sẽ chịu 5% Max HP mài mòn nhẹ).
2. **Cực Điểm Không Gian (Bạch Thiên ➔ Hắc Thiên)**:
   - `Rank >= 8`: Vượt qua tự do với uy áp Bát Chuyển.
   - `Rank == 7`: Có Tiên Cổ Ốc thì tiêu hao 15 độ bền; nếu không có Cổ Ốc, chịu 40% Max HP sát thương và chịu debuff `spatial_suppression` (giảm 30% tốc độ trong 10 phút).
   - `Rank < 7`: Cấm tiệt, chịu 70% Max HP sát thương chấn động khi cố tình xâm nhập.
3. **Địa Câu Chướng Khí (Ngũ Vực ➔ Địa Uyên)**:
   - `Rank >= 5` hoặc có Tiên Cổ Ốc: Bước vào an toàn.
   - `Rank < 5`: Bị chướng khí ngạt thở chặn lại, nhận 50% Max HP sát thương và nhiễm độc `dia_uyen_miasma`.
4. **Rơi Tự Do / Hồi Quy (Skyfall - `POST /api/world/skyfall`)**:
   - Hiện thực hóa handler `SkyfallDescent` tại `movement_layers.go` và đăng ký route tại `server.go`.
   - Chiếu tọa độ 1:1: Từ tọa độ `(X, Y)` ở Bạch Thiên, nhân vật lao mình rơi tự do xuống đúng tọa độ `(X, Y)` của đại vực bên dưới tại Ngũ Vực (`DeriveRegionAtOnMap(x, y, "world_map")`), hoặc từ Hắc Thiên rơi xuống Bạch Thiên.
   - Cổ Tiên tiếp đất an toàn; phàm nhân rơi không có Cổ Ốc chịu 40% Max HP sát thương chấn thương va chạm.

### 3. Kiểm định & Xác thực (Verification)
- Viết test suite chuyên biệt trong `backend/internal/handlers/movement_layers_test.go` (8 test cases kiểm tra đầy đủ các kịch bản phàm nhân, Cổ Tiên, Tiên Cổ Ốc, Địa Uyên, Hắc Thiên, Skyfall).
- Chạy `go test -race ./internal/handlers`: **PASS** (100% data race free).
- Chạy toàn bộ backend test `go test ./...`: **PASS**.
- Chạy `npm run typecheck`: **PASS**.
