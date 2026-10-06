# Audit và Phân loại toàn bộ 117 vị trí FOR UPDATE trong Codebase

Type: research
Status: resolved
Blocked by: None

Part of map.md

## Question

Trong codebase hiện hữu có 117 vị trí xuất hiện `FOR UPDATE`. Cụ thể:
1. Những vị trí nào thuộc về "Hot-spot" có nguy cơ cao nhất gây ra lock contention và statement timeout `57014`?
2. Có thể phân loại 117 vị trí này thành các nhóm giải pháp nào:
   - Nhóm A: Thay thế hoàn toàn bằng Atomic SQL (không cần SELECT FOR UPDATE).
   - Nhóm B: Chuyển sang Optimistic Concurrency Control (OCC dựa trên version).
   - Nhóm C: Bắt buộc dùng Pesimistic Lock nhưng cần thêm `NOWAIT` / Fail-fast.
   - Nhóm D: Worker/Batching cần dùng `SKIP LOCKED`.
   - Nhóm E: Khóa đa dòng (Multi-row lock) cần chuẩn hóa `ORDER BY id` để chống deadlock `40P01`.
   - Nhóm F: Vị trí an toàn / tần suất cực thấp (Admin, setup).
3. Đâu là danh sách chi tiết các file và dòng code tương ứng với từng nhóm?

## Answer

### 1. Thống kê tổng quan
Lệnh rà soát trả về chính xác 117 kết quả:
- **107 câu lệnh SQL trực tiếp** thực hiện khóa dòng (`FOR UPDATE`, `FOR UPDATE OF`, `ORDER BY ... FOR UPDATE`).
- **10 dòng comment** giải thích kiến trúc lock-free hoặc chú thích gọi `EvaluateAndPersistMana`.

Phân bổ theo bảng: `players` (24), `mine_slots` (5), `sects` & kho tông môn (16), `auction` (3), `trade_*` (6), `buyback_orders` (3), `player_inventory_items` (5), `player_gu` (6), `non_player_actors` (8), `parties` (6), `teleport_arrays` (5), `immortal_apertures` (9), khác (11).

### 2. Top 5 Hot-spots nguy hiểm nhất gây lỗi 57014 (Statement Timeout)
1. **`PlayerSvc.LoadForUpdate` trong `EvaluateAndPersistMana` (`player_service.go:176`)**:
   Nhúng vào hơn 70 API endpoints. Khi người chơi thao tác đồng thời (auto-combat + dùng item + SSE refresh), các transaction của cùng player xếp hàng trên `players`. Transaction giữ lock tính toán regen mana và hunger tick (50-200ms) khiến các query sau vượt `statement_timeout` 15s.
2. **Tranh chấp Mỏ Khoáng `mine_slots` (`mine.go:236, 309, 391, 495`, `combat_result_mine.go:29`)**:
   Chỉ có 31 mỏ toàn server. Khi nhiều người/bot đồng thời chiếm mỏ hoặc claim thưởng, không có `NOWAIT` khiến các transaction sau bị treo đợi đến 15s.
3. **Tông Môn Chiến & Cướp Kho `sects` + `sect_vault_*` (`sect_warfare.go:528, 761, 793, 884, 973`)**:
   Transaction nặng vừa khóa tông môn vừa duyệt hàng chục item trong 3 bảng kho mà không có thứ tự khóa chuẩn, làm nghẽn mọi thành viên thao tác kho.
4. **Đấu Giá & Mua Bán Sàn `auction` (`marketplace.go:504, 666, 868`)**:
   Spam bid snipe giờ chót gây nghẽn transaction khóa listing và gọi chéo `evaluateManaTx` cho cả buyer lẫn seller.
5. **Giao Dịch Trực Tiếp P2P `trade_confirm.go:52, 210, 268, 330`**:
   Khóa chéo 2 player và session trade mà không chuẩn hóa thứ tự `MIN(sender, receiver)` -> nguy cơ Deadlock 40P01 và kéo dài thời gian giữ lock.

### 3. Phân loại 6 nhóm giải pháp:
- **Nhóm A - Atomic SQL (27 vị trí)**: Thay thế `SELECT FOR UPDATE` -> tính toán -> `UPDATE` bằng 1 câu `UPDATE ... WHERE ... AND col >= $val RETURNING ...`. Triệt tiêu hoàn toàn thời gian giữ lock trong Go (ví dụ: `mine.go:85`, `beast_betting.go`, `inventory_service.go`).
- **Nhóm B - OCC Optimistic Concurrency Control (15 vị trí)**: Sử dụng `players.version` với retry 3 lần, backoff 20ms cho `player_service.go:176`, `pets.go`, `variant_npc_gameplay.go`.
- **Nhóm C - NOWAIT / Fail-Fast (24 vị trí)**: Đổi sang `FOR UPDATE NOWAIT`, bắt lỗi `55P03` và trả về ngay HTTP 409 Conflict cho mỏ (`mine_slots`), đấu giá (`auction`), nhặt đồ rơi (`map_ground_drops`).
- **Nhóm D - Worker SKIP LOCKED (6 vị trí)**: Thêm `SKIP LOCKED` cho `growth.go`, `hunger.go`, `variant_refinement.go`, `marketplace_helpers.go` để worker bỏ qua hàng đang bị người chơi online thao tác.
- **Nhóm E - Multi-row ORDER BY id (18 vị trí)**: Chuẩn hóa khóa theo thứ tự ID tăng dần (`ORDER BY id ASC FOR UPDATE`) cho P2P Trade, Sect Warfare, Inventory Bulk Remove để chống triệt để deadlock `40P01`.
- **Nhóm F - An toàn / Admin (17 vị trí)**: Các thao tác Admin, webhook SePay, bảo trì tiên khiếu tần suất thấp; chỉ cần đặt timeout bảo vệ 3s.
