# Ticket 04: Tiêu hao Tiên Nguyên và Khấu trừ Độ No Cổ Trùng theo Chiêu Thức

Type: task
Status: resolved
Blocked by: Ticket 02
Assignee: Antigravity

## Question

Cơ chế kiểm soát tài nguyên (Chân Nguyên / Tiên Nguyên) và khấu trừ Độ No (Hunger) của Cổ hạch tâm khi thi triển các Trận Kích trong Simulator (`engine.go` và `interactive.go`) cần được cài đặt như thế nào để đảm bảo tính cân bằng và đúng nguyên lý lore?

Cụ thể cần giải quyết:
1. **Tiêu Hao Tiên Nguyên Theo Từng Chiêu Thức (Action Cost)**:
   - Thống nhất loại bỏ cơ chế trừ mana thụ động mỗi đầu lượt (per-turn upkeep).
   - Từng chiêu thức Trận Kích (Slot 1–5) sẽ mang chỉ số `ManaCost` cụ thể, được điều chỉnh theo Rank Cổ Phòng và chênh lệch cảnh giới tu vi của Cổ Tiên.
   - Thần thông hủy diệt tối hậu (Slot 5) tiêu hao lượng Tiên Nguyên cực lớn; nếu không đủ Tiên Nguyên, hành động không thể thi triển.
2. **Khấu Trừ Độ No Cổ Trùng (Gu Hunger Deduction)**:
   - Khi Cổ Tiên kích hoạt Trận Kích từ Cổ hạch tâm đơn lẻ (như Slot 1 Pháo kích hoặc Slot 3 Phòng hộ), hệ thống khấu trừ độ no của chính Cổ hạch tâm đó trong `CombatantAuto.GuHunger`.
   - Khi thi triển Thần thông tối hậu (Slot 5), toàn bộ các Cổ hạch tâm trong `ComponentGuIDs` đều bị trừ độ no đồng loạt.
3. **Xử Lý Trạng Thái Kiệt Sức / Đói Khát Trong Trận**:
   - Trong `interactive.go` (`combatActionAvailable`) và `engine.go`, nếu một Cổ hạch tâm có `GuHunger <= 0`, chiêu thức Trận Kích tương ứng sẽ bị khóa (mờ đi trong bảng Interactive hoặc bị bỏ qua trong Auto mode).
   - Đồng bộ trạng thái suy giảm độ no của các Cổ hạch tâm vào bảng `player_gu` sau khi kết thúc trận đánh.

## Answer

Đã chuẩn hóa và triển khai toàn diện cơ chế tiêu hao Tiên Nguyên và khấu trừ độ no của Cổ hạch tâm trong combat theo lượt:

1. **Tiêu Hao Tiên Nguyên Theo Chiêu Thức (Action-level Mana Cost)**:
   - Đã loại bỏ hoàn toàn cơ chế trừ mana thụ động mỗi đầu lượt.
   - Mỗi Trận Kích mang `ManaCost` xác định, tính toán qua `actionConflictMult` và `gamefx.ManaTaxMultiplier`.
   - Khi Cổ Tiên không đủ Tiên Nguyên, `combatActionAvailable` trả về `false` (khóa chiêu trong Interactive Combat) và engine simulator phát sinh sự kiện log `INSUFFICIENT_MANA` rồi bỏ qua lượt.
2. **Khấu Trừ Độ No & Trường `HungerCost` Mở Rộng**:
   - Mở rộng struct `SimAction` với trường `HungerCost int` (mặc định = 1 nếu `<= 0`; có thể cấu hình = 2 cho Slot 5 Thần thông tối hậu).
   - Khi thi triển Trận Kích (`IsFromGuHouse == true`), engine khấu trừ đúng `HungerCost` vào `actor.GuHunger[guID]` và ghi nhận `actor.GuUsedCounts[guID] += HungerCost` cho từng Cổ thành phần trong `ComponentGuIDs`.
3. **Kiểm Soát Trạng Thái Kiệt Sức (Hunger Depletion)**:
   - Bổ sung kiểm tra độ no cho `KILLER_MOVE` và `IsFromGuHouse` trong cả hai chốt chặn:
     - `interactive.go` (`combatActionAvailable`): Nếu bất kỳ Cổ hạch tâm nào có `GuHunger < HungerCost`, chiêu thức bị đánh dấu không khả dụng, không hiển thị/không cho phép chọn trong bảng Interactive.
     - `engine.go` (`checkActionCondition`): Đảm bảo engine simulator trong auto-mode tự động bỏ qua Trận Kích của Cổ đã kiệt sức để chuyển sang đánh thường.
   - Khi trận đấu kết thúc, số lượng độ no hao hụt được đồng bộ tự động vào database `player_gu` thông qua `db.SyncPlayerGus` (cho người chơi chính) và `UPDATE player_gu SET hunger` (cho đồng đội).
   - Kiểm chứng thành công qua unit test: `TestGuHouseAction_HungerAndManaDeduction`.
