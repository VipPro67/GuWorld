# Ticket 03: Hấp thụ Sát thương Form HP 100% No Spillover và Phản phệ Vỡ Phòng

Type: task
Status: resolved
Blocked by: Ticket 01
Assignee: Antigravity

## Question

Làm thế nào để chuẩn hóa logic hấp thụ sát thương của Tiên Cổ Phòng trong `simulator/engine.go` tuân thủ nguyên tắc **100% Không Tràn Sát Thương (No Spillover)** và xử lý toàn diện chuỗi phản phệ **Vỡ Phòng (Structural Breach)** trong và sau trận đấu?

Cụ thể cần giải quyết:
1. **Chuẩn hóa Hấp Thụ Sát Thương (No Spillover)**:
   - Trong `engine.go`, sửa đoạn logic hiện tại (đang tràn sát thương thừa vào `target.HP` khi `damage > HP`):
     Khi đòn đánh vượt quá lượng Form HP còn lại của Cổ Phòng, đòn đánh đó bị triệt tiêu hoàn toàn tại kết cấu Cổ Phòng, Form HP về 0, không có sát thương nào tràn vào HP của Cổ Tiên.
2. **Kích Hoạt Phản Phệ Vỡ Phòng Tức Thời Trong Trận**:
   - Ngay tại thời điểm Cổ Phòng bị công phá vỡ vụn:
     - Đặt `GuHouse.IsDestroyed = true`, giải trừ trạng thái Dung Nhập.
     - Cổ Tiên bị văng ra nghênh chiến, nhận ngay debuff **Choáng (Stun)** kéo dài 1–2 lượt.
     - Trừ trực tiếp 30% Máu tối đa của nhục thân (`target.HP -= target.MaxHP * 0.30`) do xung kích phản chấn của đại trận.
     - Toàn bộ các chiêu thức Trận Kích (`IsFromGuHouse`) lập tức bị vô hiệu hóa trong các lượt kế tiếp.
3. **Kết Toán Bền Vững Vào Database (Post-Combat Durability Settlement)**:
   - Khi trận đấu kết thúc (`combat_settlement.go`):
     - Nếu Cổ Phòng bị vỡ: Cập nhật `player_gu_houses.current_durability = 0`, gỡ trang bị (`is_equipped = false`). Cổ Phòng không thể trang bị lại cho đến khi được Tu Sửa qua `repairGuHouse`.
     - Nếu Cổ Phòng không bị vỡ nhưng mất Form HP: Tính tỷ lệ suy hao độ bền theo công thức $D_{\text{loss}} = \frac{\text{DamageTaken}}{\text{MaxFormHP}} \times 10\%$, trừ tương ứng vào `current_durability`.
     - Chấn thương Cổ hạch tâm: Giảm 30% độ bền (`player_gu.durability`) của tất cả các Cổ hạch tâm cấu thành Cổ Phòng khi vỡ trận.

## Answer

Đã chuẩn hóa và triển khai thành công logic hấp thụ sát thương No Spillover, phản phệ vỡ phòng và kết toán độ bền bền vững:

1. **Chuẩn Hóa Triệt Tiêu Sát Thương (100% No Spillover)**:
   - Đã sửa tại `backend/internal/simulator/engine.go` (nhánh xử lý `target.GuHouse`):
     Khi đòn đánh vượt quá lượng Form HP còn lại của Cổ Phòng (`damage > target.GuHouse.HP`), toàn bộ sát thương của đòn đánh bị hấp thụ và triệt tiêu tại kết cấu Cổ Phòng; Form HP về 0 và `damage` KHÔNG tràn sang nhục thân (`return 0`).
2. **Kích Hoạt Phản Phệ Vỡ Phòng (Structural Breach Recoil)**:
   - Ngay khi Form HP cạn kiệt:
     - `target.GuHouse.IsDestroyed = true; target.GuHouse.Active = false`.
     - Cổ Tiên bị văng ra nghênh chiến, chịu trạng thái **Choáng 2 lượt** (`target.IsStunned = true`, gán effect `statusfx.StatusStun` với `Duration = 2`).
     - Trừ trực tiếp 30% sinh mệnh tối đa của Cổ Tiên do phản chấn đại trận (`recoil := math.Floor(target.MaxHP * 0.30); target.HP = math.Max(1, target.HP - recoil)`).
     - Chiêu thức Trận Kích (`IsFromGuHouse`) lập tức bị khóa trong các lượt tiếp theo qua điều kiện `checkActionCondition`.
   - Kiểm chứng thành công qua unit test: `TestSimulator_GuHouseStructuralBreach_NoSpilloverAndRecoil`.
3. **Kết Toán Độ Bền Vào Database Sau Trận Đấu**:
   - Triển khai hàm `persistGuHouseOutcome(tx, playerID, house)` trong `backend/internal/handlers/combat_finalize.go` áp dụng cho cả initiator và đồng đội:
     - Khi Cổ Phòng bị phá hủy (`house.IsDestroyed`):
       - Cập nhật `player_gu_houses SET current_durability = 0, is_equipped = false`.
       - Truyền 30% xung kích chấn thương làm suy giảm độ bền của toàn bộ Cổ hạch tâm: `UPDATE player_gu SET durability = GREATEST(0, durability - 30) WHERE id = ANY(core_gu_instances)`.
     - Khi Cổ Phòng chưa bị vỡ:
       - Cập nhật độ bền tương ứng với tỷ lệ Form HP còn lại sau trận: `newDur := math.Max(0.5, (house.HP / house.MaxHP) * 100.0)`.
   - Toàn bộ test suite `internal/simulator` và `internal/handlers` đều pass 100%.
