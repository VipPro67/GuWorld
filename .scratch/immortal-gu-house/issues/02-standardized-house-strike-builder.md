# Ticket 02: Cơ chế tự động sinh và biên dịch Bộ 6 Chiêu Thức Trận Kích

Type: prototype
Status: resolved
Blocked by: 01

## Question

Khi Cổ Tiên hoàn tất Dung Nhập vào Tiên Cổ Phòng, hệ thống Zone Sim làm thế nào để tự động cấu trúc, biên dịch và nạp bộ 6 chiêu thức Trận Kích (`Full House Strike Set`) chuẩn hóa vào Action Bar (Slots 1..6) của thực thể?

Cụ thể cần giải quyết các vấn đề:
1. Ánh xạ vai trò 6 chiêu chuẩn hóa theo `CONTEXT.md`:
   - Slot 1 (Pháo kích công kích): Tự động lấy Trận Kích công kích tầm xa từ Cổ hạch tâm số 1 (hoặc Cổ có damage cao nhất).
   - Slot 2 (Xung kích di chuyển): Kỹ năng cơ động / lướt ủi càn quét của Cổ Phòng (cự ly, tốc độ, đẩy lùi vật thể).
   - Slot 3 (Đại trận phòng hộ): Kỹ năng kích hoạt khiên chắn quang thuẫn / đại trận hộ thể.
   - Slot 4 (Hồi phục sửa chữa): Chiêu thức tự tiêu hao Tiên Nguyên để hồi phục Form HP tức thời cho Cổ Phòng.
   - Slot 5 (Thần thông hủy diệt tối hậu): Đại chiêu AoE quy mô lớn, tổng hợp uy lực từ toàn bộ Cổ hạch tâm và số lượng Cổ phụ.
   - Slot 6 (Lệnh rời khỏi Cổ Phòng): Chiêu giải trừ dung nhập (cooldown 1s, 0 mana).
2. Quy tắc biên dịch realtime: Kiểm tra và đảm bảo các effect của 6 chiêu thức vượt qua `realtimeeffects.BuildKillerMoveSkills`.
3. Cơ chế tiêu hao độ no: Trận Kích thi triển sẽ tiêu hao độ no của Cổ hạch tâm tương ứng như thế nào trong trận?

## Answer

Đã chuẩn hóa toàn diện cơ chế kiến tạo, biên dịch và nạp bộ 6 chiêu thức Trận Kích (`Full House Strike Set`) của Tiên Cổ Phòng khi người chơi Dung Nhập:

1. **Ma Trận 6 Vai Trò Chuẩn Hóa Theo CONTEXT.md**:
   Hàm kiến tạo `BuildGuHouseSkillProfile(house models.GuHouseCombatProfile)` sẽ tự động xuất ra `FormSkillProfile` gồm đúng 6 slot:
   - **Slot 1 (Pháo kích công kích - Artillery Strike)**:
     - *Nguồn*: Trích xuất hành động tấn công có `BaseDamage` cao nhất từ `house.Actions` (hoặc Cổ hạch tâm 1).
     - *Thông số*: `ModeDirection`, Range 16.0m, ProjectileSpeed 24.0m/s, ProjectileRadius 0.6m.
     - *Sát thương*: $D = D_{\text{base}} \times (1.0 + \text{AuxGuCount} \times 0.005)$ (Cổ phụ buff uy lực pháo).
     - *Tiêu hao độ no*: Khấu trừ `DefaultGuHungerCostPerCast` (0.1) từ Cổ hạch tâm 1 (`GuInstanceID: core_1`).
   - **Slot 2 (Xung kích di chuyển - Siege Trample)**:
     - *Nguồn*: Đặc tính cơ động của pháo đài Cổ Phòng.
     - *Thông số*: `ModeDirection`, Range 10.0m, `MoveDistance: 10.0m`, `MoveDurationTicks: 8` (0.4s lướt nhanh), `KnockbackDistance: 3.5m` (hất tung quái vật cản đường).
     - *Sát thương*: $D = \text{Rank} \times 80 + \text{AuxGuCount} \times 2$.
     - *Thời gian hồi*: Cooldown 80 ticks (4.0s). Mana: $15 + \text{Rank} \times 3$.
   - **Slot 3 (Đại trận phòng hộ - Fortress Barrier)**:
     - *Nguồn*: Kết cấu phòng hộ trận pháp của Cổ Phòng (hoặc Cổ hạch tâm phòng thủ có `BaseShield`).
     - *Thông số*: `ModeSelf`, cấp một lớp quang thuẫn khiên `Shield = house.MaxHouseHP * 0.20` (hấp thu 20% Form HP tối đa, tồn tại 200 ticks = 10s).
     - *Tiêu hao độ no*: Khấu trừ 0.1 độ no của Cổ hạch tâm phòng thủ 2 (nếu có).
     - *Thời gian hồi*: Cooldown 200 ticks (10.0s). Mana: $25 + \text{Rank} \times 4$.
   - **Slot 4 (Hồi phục sửa chữa - Structural Mending)**:
     - *Nguồn*: Khả năng điều phối Tiên Nguyên tu bổ khẩn cấp kết cấu Cổ Phòng.
     - *Thông số*: `ModeSelf`, hồi phục trực tiếp Form HP: `Heal = house.MaxHouseHP * 0.15` (không vượt trần `MaxHouseHP`, chỉ hồi Form HP, không hồi HP Cổ Tiên).
     - *Thời gian hồi*: Cooldown 300 ticks (15.0s). Mana: $35 + \text{Rank} \times 5$. Windup: 20 ticks (1.0s).
   - **Slot 5 (Thần thông hủy diệt tối hậu - Ultimate Destruction)**:
     - *Nguồn*: Tuyệt chiêu tối hậu kết hợp toàn bộ 2-5 Tiên Cổ hạch tâm và đại lượng Cổ phụ.
     - *Thông số*: `ModeGround`, Range 12.0m, bán kính AoE `Radius: 6.5m`. Tụ lực Windup: 30 ticks (1.5s).
     - *Sát thương*: $D = \text{Rank} \times 250 + \text{CoreGuCount} \times 150 + \text{AuxGuCount} \times 10$.
     - *Tiêu hao độ no*: Khấu trừ đồng loạt 0.2 độ no của TOÀN BỘ các Cổ hạch tâm trong `GuComponentIDs`.
     - *Thời gian hồi*: Cooldown 600 ticks (30.0s). Mana: $60 + \text{Rank} \times 10$.
   - **Slot 6 (Lệnh rời Cổ Phòng - Disembark)**:
     - *Thông số*: `ID: "REVERT_GU_HOUSE"`, `ModeSelf`, Mana: 0, Cooldown: 20 ticks (1.0s), Windup: 0. Gọi `revertTransformationLocked(caster, "voluntary_exit", false)`.

2. **Quy Tắc Biên Dịch & Nạp Vào Realtime Zone Sim**:
   - Khi kích hoạt Dung Nhập thành công tại `completeTransformationLocked`, sim gọi `BuildGuHouseSkillProfile(house)` để lấy `FormSkillProfile`.
   - Các hiệu ứng của Trận Kích tuân thủ schema `gameplayeffects.Block` tương thích với `realtimeeffects.BuildKillerMoveSkills`.
   - `SwapSkillPalette(entity, w.tick, profile)` nạp 6 chiêu này vào `entity.Combat.Skills[1..6]`, đồng thời đặt Cooldown ban đầu = 0 để sẵn sàng thi triển ngay lập tức.
   - Các slot Alt 9..12 tự động bị khóa trong thời gian Dung Nhập.

3. **Cơ Chế Tiêu Hao Độ No Cổ Trùng (Gu Hunger Deduction)**:
   - Áp dụng qua trường `GuInstanceID` (đối với chiêu hạch tâm đơn như Slot 1, Slot 3) hoặc `GuComponentIDs` (đối với đại chiêu tổng lực Slot 5).
   - Mỗi lần cast chiêu thành công, sim Zone gọi trừ độ no tương ứng qua `combat.DefaultGuHungerCostPerCast`.
   - Nếu bất kỳ Cổ hạch tâm nào bị đói kiệt sức (`hunger <= 0`), Cổ đó rơi vào trạng thái ngừng hoạt động, khiến chiêu thức tương ứng bị khóa cho đến khi được cho ăn.

4. **Tài Sản Kiểm Chứng Logic Prototype**:
   - Prototype độc lập mô phỏng đầy đủ 6 vai trò, thi triển qua phím [1]..[6], trừ độ no và cooldown đã được xây dựng tại:
     [`backend/internal/realtime/sim/prototype_gu_house_strike_palette_demo.html`](file:///home/vippro67/GitHub/GuWorld/backend/internal/realtime/sim/prototype_gu_house_strike_palette_demo.html)
