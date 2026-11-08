# Ticket 04: Cơ chế hấp thu 100% sát thương, suy hao độ bền DB và xử lý Vỡ Phòng

Type: grilling
Status: resolved
Blocked by: 01, 02

## Question

Trong Zone Sim 20Hz, cơ chế hấp thu 100% sát thương bảo vệ nhục thân Cổ Tiên, đồng bộ suy hao độ bền về cơ sở dữ liệu (`player_gu_houses`), và quy trình xử lý thảm họa khi Cổ Phòng bị vỡ (`Structural Breach`) được hiện thực hóa như thế nào?

Cụ thể cần giải quyết các vấn đề:
1. Cơ chế hấp thu sát thương qua `ApplyFormDamage`:
   - Mọi sát thương đánh trúng Cổ Tiên khi đang ở hình thái `GU_HOUSE` đều bị chặn 100% bởi `CurrentFormHP` của Cổ Phòng (không có sát thương xuyên thấu đến máu gốc nhân vật, trừ các sát thương đặc thù vô thị phòng hộ nếu có).
   - Chỉ số Giáp Cổ Phòng (`HouseDefense`) và Giảm sát thương phần trăm (`DamageReduction`) can thiệp ra sao vào công thức tính sát thương thực tế trước khi trừ vào Form HP?
2. Đồng bộ độ bền DB (`player_gu_houses.current_durability`):
   - Thời điểm đồng bộ độ bền: Cập nhật định kỳ (periodic flush) hay chốt khi giải trừ dung nhập / rời map?
   - Tỷ lệ quy đổi: Từ Form HP bị mất sang điểm độ bền (thang 0-100).
3. Xử lý Vỡ Phòng (Form HP = 0):
   - Trục xuất Cổ Tiên ra ngoài ngay lập tức.
   - Hiệu ứng phản chấn thảm khốc: Cổ Tiên dính Choáng (Stun 3.0s), mất 30% HP tối đa do phản phệ trận pháp nổ tung.
   - Gọi `ApplyGuHouseDamage` trong DB: Độ bền Cổ Phòng về 0 (hỏng nặng, khóa tính năng trang bị cho đến khi tu sửa) và truyền 30% xung kích chấn thương làm suy giảm độ bền của toàn bộ Cổ hạch tâm (`player_gu`).

## Answer

Đã thống nhất và chuẩn hóa toàn diện cơ chế hấp thu sát thương, suy hao độ bền DB và quy trình xử lý thảm họa Vỡ Phòng (`Gu House Structural Breach`):

1. **Cơ Chế Hấp Thu 100% Sát Thương (No Spillover)**:
   - Trong Zone Sim 20Hz (`applyFormDamageLocked` khi `TransformationKind == TransformationGuHouse`):
     - Sát thương đầu vào được tính toán giảm trừ trước bởi Giáp Cổ Phòng (`HouseDefense`) và Tỷ lệ giảm thương (`DamageReduction` = 10% đến 35% tùy số lượng Tiên Cổ hạch tâm).
     - Sát thương thực tế được khấu trừ 100% vào `CurrentFormHP` của Cổ Phòng. Máu gốc nhân vật Cổ Tiên được che chắn tuyệt đối (không có sát thương xuyên thấu thông thường).
     - **Triệt tiêu sát thương tràn (No Spillover)**: Khi đòn đánh vượt quá Form HP còn lại, Cổ Phòng nhận toàn bộ đòn đánh và vỡ nát (`CurrentFormHP = 0`). Phần sát thương vượt ngưỡng KHÔNG tràn vào máu nhân vật. Thay vào đó, Cổ Tiên phải nhận **Sát Thương Phản Phệ Vỡ Trận** cố định.

2. **Cơ Chế Suy Hao Độ Bền DB (`player_gu_houses.current_durability`)**:
   - Để tránh việc người chơi chỉ bật/tắt Cổ Phòng thử nghiệm hoặc giao tranh nhẹ mà bị tụt quá nhiều độ bền, độ hao mòn được thiết kế phân tầng mượt mà:
     - **Khi bật/tắt hoặc giao tranh nhẹ (mất ít HP)**: Độ bền chỉ hao mòn vận hành tối thiểu $0.5\% - 1.0\%$ mỗi phiên dung nhập.
     - **Khi giao tranh có tổn thất Form HP**: Độ bền giảm theo công thức:
       $$\Delta \text{Durability} = 0.5\% + 10\% \times \left(1.0 - \frac{\text{CurrentFormHP}}{\text{MaxFormHP}}\right)$$
       *(Ví dụ: Mất 50% Form HP trong trận thì độ bền DB giảm $0.5\% + 5\% = 5.5\%$).*
     - **Khi Vỡ Phòng (`CurrentFormHP == 0`)**: Độ bền lập tức giảm thẳng về $0\%$.
   - **Thời điểm đồng bộ DB**:
     - Tuân thủ nguyên tắc `persistence/state.go`: Zone ticks 20Hz không gọi DB trực tiếp.
     - Mức độ bền mới được ghi nhận vào `persistence.Checkpoint` khi Cổ Tiên chủ động rời Cổ Phòng, khi chuyển map, hoặc lưu định kỳ qua background checkpoint worker.
     - Khi xảy ra sự kiện Vỡ Phòng, Zone Sim phát sinh async event để cập nhật DB tức thời.

3. **Quy Trình Xử Lý Thảm Họa Vỡ Phòng (Gu House Structural Breach)**:
   - Khi `CurrentFormHP` chạm mốc 0:
     - Kích hoạt `revertTransformationLocked(entity, "structural_breach", true)`.
     - **Phản phệ nhục thân**: Cổ Tiên bị hất văng ra ngoài, lập tức mất cố định 30% Max HP nhục thân (không thể né tránh, nếu máu hiện tại $\le 30\%$ thì giữ lại tối thiểu 1 HP để không chết oan do lỗi sim), và dính trạng thái Choáng cứng `STUN` trong 60 ticks (3.0 giây).
     - **Vô hiệu hóa Cổ Phòng**: Độ bền Cổ Phòng trong `player_gu_houses` gán bằng 0 (`current_durability = 0`), tự động gỡ trang bị (`is_equipped = false`). Khóa toàn bộ tính năng Dung Nhập và Trận Kích cho đến khi người chơi dùng tính năng `RepairGuHouse` tu sửa về 100%.
     - **Phản chấn Cổ hạch tâm**: Gọi `ApplyGuHouseDamage(tx, houseID, 100)`: Truyền 30% xung kích chấn thương làm suy giảm độ bền của toàn bộ các Tiên Cổ hạch tâm (`player_gu`). Nếu bất kỳ Cổ hạch tâm nào tụt độ bền về $\le 0$, Cổ đó chuyển sang trạng thái Trọng Thương (`State: INJURED`) và bị cấm xuất chiến cho đến khi được nuôi dưỡng/tu bổ.

4. **Cập Nhật Domain Model**:
   - Đã chuẩn hóa và bổ sung định nghĩa thuật ngữ `Vỡ Phòng Tiên Cổ Phòng (Gu House Structural Breach)` vào từ điển nghiệp vụ [`CONTEXT.md`](file:///home/vippro67/GitHub/GuWorld/CONTEXT.md).
