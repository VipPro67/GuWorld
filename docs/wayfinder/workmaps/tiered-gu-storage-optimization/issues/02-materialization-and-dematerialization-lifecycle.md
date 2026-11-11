# Ticket 02: Vòng đời chuyển đổi hai chiều (Materialization & Dematerialization) giữa Đàn Cổ và Cá Thể

Type: prototype
Status: resolved
Blocked by: 01

## Question

Làm thế nào để thiết lập quy trình chuyển đổi hai chiều an toàn, nhất quán trong Transaction (`sql.Tx`) giữa Đàn Cổ xếp chồng (Stack) và Cổ cá thể (`player_gu`)?

Cụ thể cần giải quyết:
1. **Quy trình Cụ Thể Hóa (Materialization - Rút ra trang bị)**:
   - Khi Cổ Tiên bấm "Trang bị" một Phàm Cổ từ Đàn Cổ:
     - Kiểm tra `quantity >= 1` trong `aperture_gu_swarms`.
     - Trừ số lượng: `UPDATE aperture_gu_swarms SET quantity = quantity - 1 WHERE aperture_id = $1 AND gu_id = $2 AND rank = $3`.
     - Sinh bản ghi cá thể mới trong `player_gu`: UUID mới, `player_id`, `gu_id`, `rank`, `hunger = 100`, `durability = 100`, `refinement_progress = 100`, `state = 'ACTIVE'`, `equipped = true`, `location = 'APERTURE'`.
     - Kiểm tra giới hạn slot trang bị Không Khiếu (`MaxEquippedGus = 60`).
2. **Quy trình Hòa Nhập Đàn (Dematerialization - Tháo trang bị nhập đàn)**:
   - Khi Cổ Tiên tháo một Phàm Cổ và chọn "Thả vào Tiên Khiếu":
     - Điều kiện nhập đàn: `durability >= 100.0` (nguyên vẹn), `is_vital == false` (không phải Bản Mệnh), `is_locked == false` (không khóa).
     - Xóa bản ghi cá thể: `DELETE FROM player_gu WHERE id = $1 AND player_id = $2`.
     - Cộng dồn số lượng vào Đàn Cổ: `INSERT INTO aperture_gu_swarms (aperture_id, gu_id, rank, quantity) VALUES ($1, $2, $3, 1) ON CONFLICT (aperture_id, gu_id, rank) DO UPDATE SET quantity = aperture_gu_swarms.quantity + 1`.
3. **Xử lý Cổ bị thương / đặc thù**:
   - Nếu Phàm Cổ có `durability < 100` (bị rạn nứt trong giao tranh) hoặc `is_vital == true`: Hệ thống từ chối tự động nhập đàn, giữ nguyên dạng cá thể trong `player_gu` để người chơi tu bổ xong mới được phép nhập đàn.
   - Thử nghiệm prototype logic này để xác nhận không xảy ra race condition hay mất mát Cổ khi người chơi thao tác nhanh/đồng thời.

## Answer

Đã chuẩn hóa toàn bộ vòng đời chuyển đổi hai chiều giữa Đàn Cổ và Cá Thể, kiểm chứng hoàn tất qua prototype tại [`prototypes/02-materialization-and-dematerialization-lifecycle.html`](../prototypes/02-materialization-and-dematerialization-lifecycle.html):

### 1. Quy Trình Cụ Thể Hóa (Materialization - Rút ra Trang Bị)
Được bọc trong `sql.Tx` nguyên tử:
1. **Kiểm tra slot trang bị**: Đếm số Cổ đang trang bị của người chơi (`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND equipped`). Nếu `>= MaxEquippedGus (60)` ➔ Trả lỗi `MAX_EQUIPPED_REACHED`.
2. **Khóa và khấu trừ Đàn Cổ**:
   ```sql
   UPDATE aperture_gu_swarms
   SET quantity = quantity - 1, updated_at = NOW()
   WHERE aperture_id = $1 AND gu_id = $2 AND gu_rank = $3 AND quantity >= 1
   RETURNING quantity;
   ```
   Nếu `rows_affected == 0` ➔ Trả lỗi `SWARM_INSUFFICIENT` ("Số lượng trong đàn không đủ").
3. **Sinh cá thể mới vào `player_gu`**:
   - `id`: Sinh UUID mới (vd `gen_random_uuid()`).
   - `hunger = 100`, `durability = 100.0`, `refinement_progress = 100.0` (Cổ sinh trưởng trong Tiên Khiếu mặc định toàn vẹn sinh mệnh và thuần phục).
   - `state = 'ACTIVE'`, `equipped = true`, `location = 'APERTURE'`.
4. Cập nhật mảng JSONB `players.equipped_gus` bổ sung UUID mới.

### 2. Quy Trình Hòa Nhập Đàn (Dematerialization - Tháo & Thả Vào Tiên Khiếu)
Khi Cổ Tiên chọn "Thả vào Tiên Khiếu":
1. **Khóa và thẩm định cá thể (`FOR UPDATE`)**:
   ```sql
   SELECT id, gu_id, gu_rank, durability, is_vital, is_locked, equipped
   FROM player_gu WHERE id = $1 AND player_id = $2 FOR UPDATE;
   ```
2. **Chốt chặn an toàn (Pre-flight Validation)**:
   - **Chặn Tiên Cổ (`gu_rank >= 6`)**: Tiên Cổ là "Thiên Địa duy nhất", tuyệt đối không được phép nhập đàn (báo lỗi `IMMORTAL_GU_CANNOT_SWARM`).
   - **Chặn Bản Mệnh Cổ (`is_vital == true`)**: Gắn liền sinh mệnh Cổ Tiên, không thể giải phóng (báo lỗi `VITAL_GU_CANNOT_SWARM`).
   - **Chặn Cổ Khóa (`is_locked == true`)**: Báo lỗi `GU_LOCKED`.
   - **Chặn Cổ Bị Thương (`durability < 100.0`)**: Bảo đảm tính đồng nhất của Đàn Cổ (mọi con trong đàn đều nguyên vẹn 100% độ bền). Nếu Cổ bị thương do giao tranh, hệ thống từ chối nhập đàn và yêu cầu tu bổ lành lặn trước (báo lỗi `GU_DAMAGED`).
3. **Thực thi xóa & tăng số lượng nguyên tử**:
   - Nếu đang trang bị ➔ gỡ khỏi `players.equipped_gus`.
   - `DELETE FROM player_gu WHERE id = $1 AND player_id = $2;`
   - Cập nhật số lượng đàn:
     ```sql
     INSERT INTO aperture_gu_swarms (id, aperture_id, gu_id, gu_rank, quantity, created_at, updated_at)
     VALUES (gen_random_uuid(), $1, $2, $3, 1, NOW(), NOW())
     ON CONFLICT (aperture_id, gu_id, gu_rank)
     DO UPDATE SET quantity = aperture_gu_swarms.quantity + 1, updated_at = NOW();
     ```

### 3. Cơ Chế Thu Hồi Nhanh Một Chạm (Deposit All Healthy Mortal Gus)
* Cung cấp endpoint `POST /api/immortal-aperture/swarms/deposit-all`.
* Quét và gom toàn bộ Phàm Cổ trong `player_gu` thỏa mãn:
  `WHERE player_id = $1 AND gu_rank <= 5 AND durability >= 100.0 AND is_vital = false AND is_locked = false AND equipped = false`.
* Nhóm theo `(gu_id, gu_rank)`, thực hiện batch upsert vào `aperture_gu_swarms` và 1 lệnh `DELETE FROM player_gu WHERE id = ANY(eligible_ids)`.
* Cổ Tiên sau khi farm dã ngoại về chỉ cần 1 click là dọn sạch túi, toàn bộ Phàm Cổ lành lặn được đưa vào đàn.

