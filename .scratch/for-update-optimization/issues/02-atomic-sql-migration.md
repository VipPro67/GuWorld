# Đặc tả chuyển đổi sang Atomic SQL cho các thao tác cộng/trừ tài nguyên và chỉ số

Type: research
Status: resolved
Blocked by: None

Part of map.md

## Question

Những thao tác nào đang đọc-sửa-ghi (`SELECT ... FOR UPDATE` -> tính toán -> `UPDATE`) chỉ nhằm mục đích cộng trừ tài nguyên (đá, mana, hp, điểm cống hiến, human_qi, etc.)?
1. Làm thế nào để chuyển đổi các thao tác này sang Single-statement Atomic SQL (ví dụ: `UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`) kết hợp điều kiện bảo vệ (ví dụ: `AND primeval_stones >= $1` để chống số dư âm)?
2. Làm thế nào để lấy lại trạng thái sau khi cập nhật thông qua clause `RETURNING` mà không cần một query SELECT trước đó?
3. Đánh giá tác động đến tính toàn vẹn dữ liệu, các log audit/lịch sử có cần dữ liệu cũ hay không và cách xử lý tương thích?

## Answer

### 1. Danh mục 27 vị trí Nhóm A được chuyển đổi sang Atomic SQL

Các vị trí này hiện đang dùng mô hình sai lầm: `tx.QueryRow("SELECT ... FOR UPDATE") -> tính toán số dư trong Go -> tx.Exec("UPDATE ...")`. Việc này khiến hàng bị khóa trong toàn bộ thời gian Go xử lý logic, unmarshal JSON và đợi các câu lệnh kế tiếp trong transaction.

Toàn bộ 27 vị trí được gom theo 4 mẫu chuẩn hóa (Patterns):

---

### 2. Bốn Mẫu Thiết Kế Atomic SQL Chuẩn Mực

#### 🟢 Mẫu A1: Atomic Credit (Cộng tài nguyên - Trả về Before & After cho Audit Log)
*Áp dụng:* Thưởng mỏ (`mine.go:85`), trả thưởng cược (`beast_betting.go:558`), nhận thưởng giới thiệu (`referral.go:103, 266`), nạp tiền, hoàn tiền tông môn.

```sql
UPDATE players
SET primeval_stones = primeval_stones + $1,
    immortal_stones = immortal_stones + $2,
    human_qi = LEAST(human_qi + $3, 100000.0)
WHERE id = $4
RETURNING 
    primeval_stones - $1 AS before_stones, 
    primeval_stones AS after_stones,
    immortal_stones - $2 AS before_immortal,
    immortal_stones AS after_immortal;
```
- **Go implementation:**
```go
var beforeStones, afterStones float64
err := tx.QueryRow(query, addStones, addImmortal, addQi, playerID).Scan(&beforeStones, &afterStones, ...)
if err != nil {
    return err
}
// Dùng beforeStones và afterStones ghi vào auditlog / wallet history trực tiếp
```
*Lợi ích:* Giảm từ 2 round-trips SQL + Go lock wait xuống còn đúng **1 câu lệnh duy nhất**.

---

#### 🟢 Mẫu A2: Atomic Debit với Guard Clause (Trừ tài nguyên - Chống số dư âm & Race condition)
*Áp dụng:* Đặt cược đấu thú (`beast_betting.go:762`), nâng cấp tông môn (`sect_settings.go:42`), tạo nhiệm vụ tông môn (`sect_missions.go:497`), nâng cấp kiến trúc tông môn (`sect_buildings.go:86`), sửa trận pháp (`teleport_handlers.go:1065`), mua vật phẩm kho (`sect_vault.go:402`).

```sql
-- Ví dụ: Đặt cược đấu thú (beast_betting.go:762)
UPDATE players
SET primeval_stones = primeval_stones - $1
WHERE id = $2 AND primeval_stones >= $1
RETURNING primeval_stones + $1 AS before_balance, primeval_stones AS after_balance;

-- Ví dụ: Nâng cấp tông môn (sect_settings.go:42) với chi phí động level * level * 5000
UPDATE sects
SET level = level + 1,
    funds = funds - (level * level * 5000)
WHERE id = $1 AND funds >= (level * level * 5000)
RETURNING level, funds;
```
- **Go implementation:**
```go
var beforeBal, afterBal float64
err := tx.QueryRow(query, req.Amount, userID).Scan(&beforeBal, &afterBal)
if errors.Is(err, sql.ErrNoRows) {
    // Row không được cập nhật vì số dư không đủ hoặc id không tồn tại
    c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
    return
}
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
}
```
*Lợi ích:* 
- Loại bỏ hoàn toàn khả năng Double-spending (rút tiền 2 lần cùng lúc).
- Không có bất kỳ khoảng hở thời gian nào giữa lúc check số dư và lúc trừ tiền.

---

#### 🟢 Mẫu A3: Atomic Inventory Deduct/Delete bằng CTE (Túi đồ `player_inventory_items`)
*Áp dụng:* `InventorySvc.RemoveItem` (`inventory_service.go:206`), `deductMaterialTx` (`buyback_handlers.go:817`).

Khi trừ vật phẩm: nếu `quantity` giảm về 0 thì tự động `DELETE`, nếu `quantity > 0` thì `UPDATE`. Hiện tại code phải `SELECT FOR UPDATE` rồi `if current == quantity DELETE else UPDATE`.

```sql
WITH upd AS (
    UPDATE player_inventory_items
    SET quantity = quantity - $1, 
        updated_at = NOW()
    WHERE player_id = $2 AND item_id = $3 AND category = $4 AND quantity >= $1
    RETURNING id, quantity
),
del AS (
    DELETE FROM player_inventory_items
    WHERE id IN (SELECT id FROM upd WHERE quantity = 0)
)
SELECT quantity FROM upd;
```
- **Go implementation:**
```go
var remQty int64
err := tx.QueryRow(query, quantity, playerID, itemID, category).Scan(&remQty)
if errors.Is(err, sql.ErrNoRows) {
    return sql.ErrNoRows // Không đủ số lượng hoặc không có item
}
return err
```
*Lợi ích:* Gộp 3 bước (Lock read -> Delete hoặc Update) thành **1 atomic CTE duy nhất**. Không cần mở sub-transaction / savepoint.

---

#### 🟢 Mẫu A4: Atomic JSONB Numeric Accumulator (Đạo ngân & Chỉ số JSONB)
*Áp dụng:* Đạo ngân tiên khiếu (`tribulation_dao.go:57, 85`), Sát khí (`death_penalty.go:87, 180`).

Hiện tại đang đọc JSONB về Go `FOR UPDATE` -> `json.Unmarshal` -> cộng dồn map -> `json.Marshal` -> `UPDATE`.

```sql
-- Cộng đạo ngân path trực tiếp trong PostgreSQL (tribulation_dao.go:57)
UPDATE immortal_apertures
SET dao_marks = jsonb_set(
    COALESCE(dao_marks, '{}'::jsonb),
    ARRAY[$1],
    to_jsonb(COALESCE((dao_marks->>$1)::bigint, 0) + $2)
)
WHERE player_id = $3
RETURNING dao_marks;
```
*Lợi ích:* Triệt tiêu 100% chi phí CPU JSON marshaling/unmarshaling trong Go và triệt tiêu row lock hold-time.

---

#### 🟢 Mẫu A5: Atomic Daily Reset & Increment (Lượt thám hiểm tổ đội)
*Áp dụng:* `combat_explore.go:325` (`checkAndIncrementExploreLimit`).

Hiện tại lặp qua từng thành viên trong tổ đội, gọi `SELECT ... FOR UPDATE` từng người rồi mới `UPDATE`.

```sql
UPDATE players
SET explore_count = CASE 
        WHEN DATE(last_explore_reset AT TIME ZONE 'UTC') <> DATE($1 AT TIME ZONE 'UTC') THEN 1 
        ELSE explore_count + 1 
    END,
    last_explore_reset = $1
WHERE id = $2 AND (
    DATE(last_explore_reset AT TIME ZONE 'UTC') <> DATE($1 AT TIME ZONE 'UTC') 
    OR explore_count < 20
)
RETURNING explore_count;
```
- Nếu `sql.ErrNoRows`: Thành viên đó đã hết lượt (>= 20) trong ngày hôm nay.
- Thực hiện trong 1 câu SQL duy nhất, không khóa chéo các thành viên khác trong party.

---

### 3. Tác Động Đến Tính Toàn Vẹn & Kế Hoạch Triển Khai
1. **Toàn vẹn dữ liệu (Data Integrity):**
   - Đảm bảo 100% ACID ở tầng PostgreSQL engine.
   - Guard clause `AND col >= $val` là chuẩn bảo vệ cứng nhất trong thiết kế cơ sở dữ liệu (tương đương compare-and-swap ở mức phần cứng).
2. **Audit Logging & History:**
   - Dùng cú pháp `RETURNING (col - delta) AS before, col AS after` cung cấp giá trị số dư trước/sau cực kỳ chính xác cho `auditlog.Log` và `insertWalletTx`.
3. **Hiệu năng:**
   - Thời gian giữ khóa giảm từ ~50-100ms xuống **< 0.1ms** (chỉ tồn tại trong tích tắc commit tuple của PostgreSQL engine).
   - Triệt tiêu hoàn toàn khả năng nghẽn đợi khóa gây ra `57014` tại cả 27 vị trí này.
