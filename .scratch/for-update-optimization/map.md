# Wayfinder Map: Tối ưu hóa FOR UPDATE và triệt tiêu Statement Timeout (57014)

## Destination

Rà soát toàn diện 117 vị trí `FOR UPDATE` trong codebase, phân loại theo mức độ tranh chấp, loại bỏ row lock không cần thiết bằng Atomic SQL và Optimistic Concurrency Control (OCC), chuẩn hóa Fail-Fast (`NOWAIT` / `lock_timeout`) cho tài nguyên cạnh tranh, và ban hành tài liệu tiêu chuẩn khóa giao dịch để loại bỏ triệt để lỗi timeout 57014.

## Notes

- Domain: PostgreSQL Concurrency, Row-level Locking, Statement & Lock Timeouts, Transaction Scopes.
- Skills to consult: `research`, `codebase-design`, `domain-modeling`.
- Standing preferences:
  - Tuân thủ quy ước repo trong `AGENTS.md`: không tạo thêm tầng service/repository layer trừ khi bắt buộc; gameplay logic nằm tại `backend/internal/handlers/` và `backend/internal/services/player_service.go`.
  - Không lạm dụng Redis lock thay thế toàn bộ DB transactions: giữ tính toàn vẹn dữ liệu ACID của PostgreSQL.
  - Tối đa hóa việc dùng Single-statement Atomic SQL (`UPDATE ... SET val = val + delta WHERE ...`) để triệt tiêu thời gian giữ lock.
  - Sử dụng Fail-fast `NOWAIT` hoặc short `lock_timeout` (<= 2-3s) thay vì để query treo đến ngưỡng 15s của `statement_timeout`.

## Decisions so far

- [[01-audit-and-classification](issues/01-audit-and-classification.md)]: Rà soát toàn bộ 117 vị trí `FOR UPDATE`, phân loại 6 nhóm giải pháp (Atomic SQL: 27, OCC: 15, NOWAIT: 24, SKIP LOCKED: 6, Multi-row ORDER BY: 18, Admin: 17) và nhận diện Top 5 Hot-spots gây 57014 (`PlayerSvc.LoadForUpdate`, `mine_slots`, `sect_warfare`, `auction`, `trade_confirm`).
- [[02-atomic-sql-migration](issues/02-atomic-sql-migration.md)]: Đặc tả 5 mẫu chuẩn Atomic SQL (Credit, Debit với Guard clause, CTE Inventory Delete/Update, JSONB Numeric Accumulator, Daily Reset Increment) cho 27 vị trí Nhóm A; tận dụng `RETURNING` để cấp before/after chính xác cho Audit Log mà không cần `SELECT FOR UPDATE`.
- [[03-optimistic-locking-occ](issues/03-optimistic-locking-occ.md)]: Đặc tả chuyển đổi `PlayerService.LoadForUpdate` và các single-player mutations sang OCC dựa trên `players.version`; thiết lập helper `WithOCCRetry` (3 lần thử, exponential backoff 15ms-60ms kèm jitter); bảo lưu khóa bi quan cho P2P Trade và Combat.
- [[04-fail-fast-nowait-skip-locked](issues/04-fail-fast-nowait-skip-locked.md)]: Đặc tả áp dụng `FOR UPDATE NOWAIT` cho 24 vị trí cạnh tranh (`mine_slots`, `auction`, `map_ground_drops`), bắt lỗi `55P03` trả về HTTP 409 Conflict (< 5ms); áp dụng `FOR UPDATE SKIP LOCKED` cho 6 worker quét nền để loại bỏ hoàn toàn việc worker làm nghẽn hoặc bị nghẽn bởi người chơi online.
- [[05-lock-ordering-and-tx-boundaries](issues/05-lock-ordering-and-tx-boundaries.md)]: Chuẩn hóa thứ tự khóa tất định (`ORDER BY id ASC FOR UPDATE`, so sánh MIN/MAX ID cho Trade/PvP) để triệt tiêu vĩnh viễn Deadlock `40P01`; thiết lập 3 nguyên tắc thu hẹp phạm vi transaction (Read-Validate-Lock-Write, loại bỏ khóa tham chiếu chỉ đọc, đẩy WebSocket/SSE ra sau `tx.Commit()`).
- [[06-postgres-timeout-policy](issues/06-postgres-timeout-policy.md)]: Phân tích nguyên nhân lỗi 57014 (tổng thời gian tích lũy chờ lock + scan vượt 15s hoặc client cancel); chuẩn hóa helper `BeginTxWithTimeout` (lock_timeout 2s, statement_timeout 5s); xây dựng helper `ClassifyPostgresError` chuẩn hóa mã HTTP 409/504 và thông báo thân thiện cho frontend.

## Open Frontier Tickets

*Không còn ticket mở nào trên Frontier. Toàn bộ các quyết định kiến trúc đã được làm rõ và hoàn tất (Way is clear).*

## Phased Implementation Status (Trạng thái thực thi mã nguồn)

- ✅ **Giai đoạn 1 (Core Utilities & Helpers) — ĐÃ HOÀN THÀNH**:
  - `backend/internal/db/tx_helpers.go`: `BeginTxWithTimeout`, `ClassifyPostgresError`, `IsLockNotAvailable`, `IsStatementTimeout`, `IsDeadlock`.
  - `backend/internal/services/occ_retry.go`: `WithOCCRetry[T]`, `ErrOCCVersionConflict`.
- ✅ **Giai đoạn 2 (Dập tắt Timeout Hotspots #2, #4 & Mỏ Khoáng) — ĐÃ HOÀN THÀNH**:
  - `mine.go:236, 309, 391, 495` & `combat_result_mine.go:29`: Chuyển sang `FOR UPDATE NOWAIT`, bắt lỗi `db.IsLockNotAvailable` trả về HTTP 409 Conflict tức thì (< 5ms).
  - `marketplace.go:504, 666, 868`: Chuyển `PlaceBid` sang `FOR UPDATE OF a NOWAIT` (không khóa lây sang seller), `BuyListing` và `CancelListing` sang `NOWAIT`.
  - `ground_drops.go:186`: `FOR UPDATE NOWAIT` cho bọc đồ rơi trên bản đồ.
- ✅ **Giai đoạn 3 (Atomic SQL & Multi-row Deadlock Fix) — ĐÃ HOÀN THÀNH**:
  - `mine.go:78` (`AddMineRewardsToPlayerTx`): Chuyển sang Atomic UPDATE duy nhất, xóa bỏ hoàn toàn `SELECT ... FOR UPDATE` và 2 round-trips thừa.
  - `sect_settings.go:42` (`UpgradeSect`) & `sect_missions.go:497` (`CreateCustomMission`): Atomic UPDATE với guard clause và `RETURNING`.
  - `beast_betting.go:558, 762` (Payout & PlaceBet): Atomic UPDATE với `RETURNING` before/after cho audit wallet history.
  - `sect_warfare.go:793, 884, 973`: Thêm `ORDER BY item_id ASC`, `ORDER BY id ASC`, `ORDER BY recipe_id ASC FOR UPDATE` triệt tiêu Deadlock 40P01 khi cướp kho tông môn.
  - `inventory_service.go:409`: Thêm `ORDER BY item_id ASC FOR UPDATE`.
- ✅ **Giai đoạn 4 (Worker Non-blocking & Nghiệm thu) — ĐÃ HOÀN THÀNH**:
  - `growth.go:397`: `FOR UPDATE SKIP LOCKED` (bỏ qua không làm nghẽn người chơi online).
  - `variant_refinement.go:127, 264`: `FOR UPDATE OF q SKIP LOCKED` và `ORDER BY quantity DESC FOR UPDATE SKIP LOCKED`.
  - `marketplace_helpers.go:207, 299`: `FOR UPDATE SKIP LOCKED` cho worker quét hoàn tiền lệnh hết hạn.
  - Kiểm thử: `go test ./...` và `go test -race ./...` toàn bộ backend **Passed 100%**.

## Out of scope

- Chuyển đổi toàn bộ sang kiến trúc Event Sourcing hay thay thế PostgreSQL bằng NoSQL / NewSQL.
- Thay thế hoàn toàn DB transaction bằng Redis Redlock trên toàn hệ thống.
- Viết lại toàn bộ cấu trúc kiến trúc server sang Clean Architecture / Repository pattern trái với `AGENTS.md`.
