# Chuẩn hóa Thứ tự Khóa (Deterministic Lock Ordering) và Phạm vi Giao dịch (Tx Boundary)

Type: research
Status: resolved
Blocked by: None

Part of map.md

## Question

Trong các giao dịch chạm đến nhiều bản ghi (ví dụ: Trade giữa 2 player, Sect warfare đánh phá kho tông môn, Party mời/rời nhóm, Admin bulk operation):
1. Có những vị trí nào đang khóa nhiều dòng mà không sắp xếp thứ tự nhất quán (`ORDER BY id FOR UPDATE`), dẫn đến nguy cơ Deadlock (`40P01`) hoặc Lock contention kéo dài dẫn đến Statement Timeout (`57014`)?
2. Có những transaction nào đang giữ row lock trong khi vẫn gọi các hàm tốn kém (như JSON marshaling lớn, gửi WebSocket notification, audit log nặng, hoặc truy vấn bảng khác không liên quan)?
3. Nguyên tắc thu hẹp phạm vi transaction (Transaction Scope Minimization): đưa cái gì ra ngoài transaction và chỉ giữ lại những gì tối thiểu trong transaction?

## Answer

### 1. Nhận Diện Các Vị Trí Khóa Đa Dòng Thiếu Thứ Tự (Nguy Cơ Deadlock `40P01` & Timeout `57014`)

Khi câu lệnh SQL thực hiện `SELECT ... WHERE ... FOR UPDATE` trên nhiều dòng mà không có mệnh đề `ORDER BY`, PostgreSQL sẽ khóa các dòng theo thứ tự quét vật lý (heap pages hoặc index scan bất định). Nếu 2 transaction cùng tác động lên các bản ghi đó theo thứ tự ngược nhau:
`T1 khóa A -> chờ B; T2 khóa B -> chờ A` => **Deadlock `40P01`** hoặc xếp hàng chờ đến khi hết hạn `statement_timeout 15s`.

#### Các vị trí nguy hiểm đã phát hiện trong codebase:
1. **Kho nguyên liệu tông môn chiến (`sect_warfare.go:793`)**:
   ```sql
   -- ❌ Cũ (bất định):
   SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0 FOR UPDATE
   --  Mới:
   SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0 ORDER BY item_id ASC FOR UPDATE
   ```
2. **Kho Cổ trùng tông môn chiến (`sect_warfare.go:884`)**:
   ```sql
   -- ❌ Cũ:
   SELECT id, gu_id ... FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0 FOR UPDATE
   --  Mới:
   SELECT id, gu_id ... FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0 ORDER BY id ASC FOR UPDATE
   ```
3. **Kho công thức tông môn chiến (`sect_warfare.go:973`)**:
   ```sql
   -- ❌ Cũ:
   SELECT recipe_id, completion_rate FROM sect_vault_recipes WHERE sect_id = $1 FOR UPDATE
   --  Mới:
   SELECT recipe_id, completion_rate FROM sect_vault_recipes WHERE sect_id = $1 ORDER BY recipe_id ASC FOR UPDATE
   ```
4. **Trừ vật phẩm túi đồ hàng loạt (`inventory_service.go:409 - BulkRemoveItems`)**:
   ```sql
   -- ❌ Cũ:
   SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1 AND category = $2 AND item_id = ANY($3) FOR UPDATE
   --  Mới:
   SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1 AND category = $2 AND item_id = ANY($3) ORDER BY item_id ASC FOR UPDATE
   ```
5. **Giao dịch P2P giữa 2 người chơi (`trade_confirm.go:373-395`)**:
   - Code hiện tại đã có bước so sánh `if senderID.String() < receiverID.String()`.
   - **Quy chuẩn chuẩn mực:** Luôn sắp xếp UUID theo thứ tự từ điển trước khi bắt đầu transaction khóa 2 nhân vật:
     ```go
     firstID, secondID := senderID, receiverID
     if firstID.String() > secondID.String() {
         firstID, secondID = receiverID, senderID
     }
     p1, err := evaluateManaTx(tx, firstID)
     p2, err := evaluateManaTx(tx, secondID)
     ```

---

### 2. Nhận Diện Các Anti-Pattern Giữ Lock Khi Xử Lý Tốn Kém

1. **Khóa dòng không cần thiết (Unnecessary Locking):**
   - Trong `party.go:160` (`InvitePlayer`): Transaction khóa `targetPlayer` bằng `FOR UPDATE` chỉ để kiểm tra `targetPartyID == ""`! Việc này khóa cứng người được mời, khiến họ không thể di chuyển hay đánh quái trong khi trưởng nhóm đang mở transaction.
   - Trong `marketplace.go:504`: `JOIN players p ON a.seller_id = p.id FOR UPDATE` vô tình khóa luôn người bán dù người mua chỉ đang đấu giá trên bảng `auction`.
2. **Giữ lock trong khi chạy logic nặng trong RAM:**
   - Trong `party.go:171`: Vừa giữ khóa `parties` vừa chạy hàm tìm kiếm không gian `worldPosIndex.Nearby(...)` và duyệt mảng vòng lặp.
   - Trong `trade_confirm.go:412-422`: Giữ khóa cả 2 người chơi trong khi duyệt mảng kiểm tra năng lực Thông Thiên Cổ (`InventoryHasGuCapability`).

---

### 3. Ba Nguyên Tắc Thu Hẹp Phạm Vi Transaction (Transaction Scope Minimization)

#### Nguyên Tắc 1: Deterministic Multi-Row Lock Ordering (Thứ tự khóa tất định)
- Mọi câu lệnh khóa nhiều hàng trong cùng 1 bảng **BẮT BUỘC** phải có `ORDER BY <primary_key> ASC FOR UPDATE`.
- Mọi logic khóa 2 thực thể cùng loại (Trade A & B, PvP A & B) phải so sánh và khóa theo thứ tự `MIN(idA, idB)` trước, `MAX(idA, idB)` sau.
- *Hệ quả:* Loại trừ 100% khả năng xảy ra chu trình chờ đợi (circular wait) -> Deadlock `40P01` không bao giờ xuất hiện.

#### Nguyên Tắc 2: Read-Validate-Lock-Write (Thu hẹp Critical Section)
- **Bước 1 (Validate & Calc):** Đọc dữ liệu không khóa, kiểm tra quyền hạn, tính toán tọa độ bản đồ, unmarshal dữ liệu, roll xúc xắc tỉ lệ.
- **Bước 2 (Lock & Mutate):** Mở transaction, thực hiện khóa / cập nhật đúng các hàng cần ghi bằng Atomic SQL hoặc short lock, commit ngay lập tức.
- Tuyệt đối không gọi `FOR UPDATE` trên các bảng chỉ dùng để đọc thông tin tham chiếu (như người được mời party, thông tin người bán trong sàn đấu giá).

#### Nguyên Tắc 3: Đưa Tác Vụ Phụ Ra Ngoài Transaction (Post-Commit Side Effects)
- **WebSocket Broadcast & Event Dispatcher:** Các lời gọi như `PlayerEventPublisher(...)`, `notifyTradeParticipants(...)`, SSE push chỉ được gọi **sau khi `tx.Commit()` thành công**.
- **Audit Logging nặng & format chuỗi:** Chuẩn bị sẵn struct log từ trước; nếu log vào bảng DB thì ghi nhanh, các tác vụ xuất log ra bên ngoài (file, remote) phải đẩy vào channel bất đồng bộ (goroutine worker).
