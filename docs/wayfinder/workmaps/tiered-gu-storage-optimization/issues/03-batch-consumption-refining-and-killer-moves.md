# Ticket 03: Cơ chế Tiêu thụ Hàng loạt từ Đàn Cổ trong Hợp Luyện và Sát Chiêu

Type: research
Status: resolved
Blocked by: 01

## Question

Làm thế nào để hệ thống Hợp Luyện (`refine_session.go`, `refine_core.go`) và Sát Chiêu Tiêu Hao (`killer_moves.go`, `simulator/killer_move_effects.go`) có thể khấu trừ trực tiếp hàng trăm đến hàng vạn Phàm Cổ từ Đàn Cổ Tiên Khiếu mà không cần chọn hay xóa từng Instance ID?

Cụ thể cần nghiên cứu:
1. **Hợp Luyện (Refining Session)**:
   - Hiện tại `refine_session.go` lưu mảng `selected_gu_ids []string` và khóa Cổ `is_locked = true`.
   - Với các công thức Tiên Cổ hoặc Sát Chiêu Trận cần 500 con Phàm Cổ làm phôi:
     - Làm thế nào để `RequiredGusMap` trong công thức (`models/recipe.go`) phân biệt được Cổ cần chọn cá thể (Tiên Cổ) vs Phàm Cổ trừ từ Đàn Cổ?
     - Thiết kế luồng trừ trực tiếp: `UPDATE aperture_gu_swarms SET quantity = quantity - $1 WHERE ... AND quantity >= $1`.
2. **Sát Chiêu Tiêu Hao Phàm Cổ (Killer Moves Consuming Mortal Gus)**:
   - Một số sát chiêu Tiên giai tiêu hao hàng trăm Phàm Cổ như đạn dược (ví dụ: Vạn Phong Nhận tiêu hao 500 Phong Nhận Cổ).
   - Kiểm tra xem engine chiến đấu (`simulator/engine.go`) hoặc handler chuẩn bị sát chiêu cần interface nào để kiểm tra và trừ đạn Cổ từ Tiên Khiếu trước khi xuất chiêu.
3. **Rollback & Abort Session**:
   - Nếu phiên luyện Cổ bị hủy (`AbortRefineSession`), cơ chế hoàn lại 50% số lượng Phàm Cổ vào Đàn Cổ như thế nào để đảm bảo tính nguyên tử?

## Answer

Đã hoàn thành nghiên cứu kỹ thuật và chuẩn hóa giải pháp tiêu thụ hàng loạt Phàm Cổ trong Hợp Luyện và Sát Chiêu:

### 1. Phân Tách Cổ Cá Thể vs Phàm Cổ Đàn Trong Hợp Luyện (Refining Escrow)
* **Vấn đề**: Hiện tại `validateRefineSelection` bắt buộc người chơi gửi mảng `selected_gu_ids` bằng đúng tổng số lượng Cổ nguyên liệu (`totalReqGusCount`). Nếu công thức cần 500 con Phàm Cổ làm phôi, payload sẽ chứa 500 UUID và bắt buộc người chơi phải có 500 dòng trong `player_gu`.
* **Cơ chế Phân Tầng Chuẩn Hóa**:
  - **Cổ Hạch Tâm / Tiên Cổ (Rank 6+ hoặc Phàm Cổ cá thể đặc thù)**: Người chơi gửi `selected_gu_instance_ids` (chỉ 1–3 UUID của phôi chính).
  - **Phàm Cổ Phụ Trợ Hàng Loạt (Rank 1–5, Qty lớn)**: Nếu người chơi là Cổ Tiên có Tiên Khiếu, hệ thống tự động kiểm tra và khấu trừ trực tiếp từ `aperture_gu_swarms` trong phiên khởi tạo (`CreateRefineSession`).
* **DDL Mở Rộng Cho `refine_sessions` (100% Backward Compatible)**:
  ```sql
  ALTER TABLE refine_sessions ADD COLUMN IF NOT EXISTS swarms_deducted JSONB NOT NULL DEFAULT '{}';
  ```
  Lưu snapshot số lượng Phàm Cổ đã ký quỹ: `{"gu_moonlight_1": 500, "gu_wind_blade_2": 200}`.
* **Thao tác Khấu trừ Nguyên tử (Atomic Escrow)**:
  ```sql
  UPDATE aperture_gu_swarms
  SET quantity = quantity - $1, updated_at = NOW()
  WHERE aperture_id = $2 AND gu_id = $3 AND gu_rank = $4 AND quantity >= $1;
  ```
  Chỉ tốn đúng 1 phép trừ SQL, không sinh bất kỳ UUID rác nào.

### 2. Sát Chiêu Tiêu Hao Phàm Cổ Làm Đạn (Killer Move Gu Ammo)
* **Nguyên tác Cổ Chân Nhân**: Các đại sát chiêu Tiên giai (như Vạn Phong Nhận, Kiếm Khí Trường Hà) bắn ra hàng trăm đến hàng vạn Phàm Cổ làm đạn dược tiêu hao.
* **Cấu trúc dữ liệu trong `storedKillerMove`**:
  Bổ sung trường `SwarmAmmoCost map[string]int` (ví dụ: `{"gu_wind_blade_2": 500}`).
* **Luồng Kiểm Định & Khấu Trừ Khi Xuất Chiêu (`killer_moves.go` / `engine.go`)**:
  - Khi Cổ Tiên thôi động sát chiêu: Hệ thống kiểm tra số dư trong `aperture_gu_swarms`. Nếu không đủ đạn ➔ Báo lỗi `KILLER_MOVE_AMMO_DEPLETED` hoặc suy giảm 80% uy lực sát chiêu.
  - Khi xuất chiêu thành công: Khấu trừ đạn Cổ từ Tiên Khiếu qua 1 câu lệnh `UPDATE aperture_gu_swarms SET quantity = quantity - $1`.
  - Không sinh thêm bất kỳ bản ghi nào trong `player_gu`, giữ cho database của Cổ Tiên luôn siêu nhẹ.

### 3. Cơ Chế Hoàn Lại Khi Hủy Phiên Luyện (`AbortRefineSession`)
* Khi người chơi chủ động hủy phiên (`AbortRefineSession`), luật game hoàn lại 50% nguyên liệu:
  ```go
  for swarmKey, qty := range row.SwarmsDeducted {
      refund := qty / 2
      if refund > 0 {
          guID, rank := parseSwarmKey(swarmKey)
          _, _ = tx.Exec(`
              INSERT INTO aperture_gu_swarms (id, aperture_id, gu_id, gu_rank, quantity, created_at, updated_at)
              VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
              ON CONFLICT (aperture_id, gu_id, gu_rank)
              DO UPDATE SET quantity = aperture_gu_swarms.quantity + $4, updated_at = NOW()`,
              apertureID, guID, rank, refund)
      }
  }
  ```
* **Khi Thất Bại (`OutcomeFailNormal` / `OutcomeFailDeath`)**: Toàn bộ Phàm Cổ ký quỹ trong `swarms_deducted` đã bị trừ từ lúc tạo phiên, tự động xem như tiêu tán thành tro bụi trong lò luyện mà không cần thêm bất kỳ câu lệnh `DELETE` nào.


## Question

Làm thế nào để hệ thống Hợp Luyện (`refine_session.go`, `refine_core.go`) và Sát Chiêu Tiêu Hao (`killer_moves.go`, `simulator/killer_move_effects.go`) có thể khấu trừ trực tiếp hàng trăm đến hàng vạn Phàm Cổ từ Đàn Cổ Tiên Khiếu mà không cần chọn hay xóa từng Instance ID?

Cụ thể cần nghiên cứu:
1. **Hợp Luyện (Refining Session)**:
   - Hiện tại `refine_session.go` lưu mảng `selected_gu_ids []string` và khóa Cổ `is_locked = true`.
   - Với các công thức Tiên Cổ hoặc Sát Chiêu Trận cần 500 con Phàm Cổ làm phôi:
     - Làm thế nào để `RequiredGusMap` trong công thức (`models/recipe.go`) phân biệt được Cổ cần chọn cá thể (Tiên Cổ) vs Phàm Cổ trừ từ Đàn Cổ?
     - Thiết kế luồng trừ trực tiếp: `UPDATE aperture_gu_swarms SET quantity = quantity - $1 WHERE ... AND quantity >= $1`.
2. **Sát Chiêu Tiêu Hao Phàm Cổ (Killer Moves Consuming Mortal Gus)**:
   - Một số sát chiêu Tiên giai tiêu hao hàng trăm Phàm Cổ như đạn dược (ví dụ: Vạn Phong Nhận tiêu hao 500 Phong Nhận Cổ).
   - Kiểm tra xem engine chiến đấu (`simulator/engine.go`) hoặc handler chuẩn bị sát chiêu cần interface nào để kiểm tra và trừ đạn Cổ từ Tiên Khiếu trước khi xuất chiêu.
3. **Rollback & Abort Session**:
   - Nếu phiên luyện Cổ bị hủy (`AbortRefineSession`), cơ chế hoàn lại 50% số lượng Phàm Cổ vào Đàn Cổ như thế nào để đảm bảo tính nguyên tử?
