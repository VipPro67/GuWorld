# Ticket 01: Quy chuẩn giao thức kích hoạt và giải trừ Dung Nhập Tiên Cổ Phòng

Type: prototype
Status: resolved
Blocked by:

## Question

Làm thế nào để Zone Sim 20Hz và Client thiết lập luồng kích hoạt và giải trừ Dung Nhập Tiên Cổ Phòng (`TransformationGuHouse = "GU_HOUSE"`) an toàn, ổn định?

Cụ thể cần giải quyết các vấn đề:
1. Giao thức mạng: Client gửi opcode/lệnh nào qua WebSocket/Channel để yêu cầu vào hoặc rời Cổ Phòng?
2. Điều kiện xác thực trên Server: Kiểm tra điều kiện Cổ Tiên đã trang bị Tiên Cổ Phòng (`is_equipped = true`), độ bền Cổ Phòng > 0 (`current_durability > 0`), kiểm tra lượng Tiên Nguyên tối thiểu khởi động, và đảm bảo không bị dính khống chế cứng (Hard CC).
3. Giai đoạn Niệm chú (Windup): Quá trình tụ lực 2.0s (40 ticks ở 20Hz). Nếu bị Hard CC ngắt thì xử lý thế nào?
4. Hoàn tất Dung Nhập: Sinh sự kiện `TRANSFORMATION_STARTED` với `form_kind: "GU_HOUSE"`, `form_id: house.id`, cấp Form HP, mở rộng bán kính va chạm, và chuẩn bị hoán đổi thanh kỹ năng.
5. Lệnh rời phòng chủ động: Slot 6 hoặc lệnh ngắt khẩn cấp xử lý giải trừ như thế nào để hoàn nguyên Cổ Tiên mà không bị lỗi trạng thái?

## Answer

Đã chuẩn hóa toàn bộ giao thức và luồng xử lý dung nhập / giải trừ Tiên Cổ Phòng trong Zone Sim 20Hz:

1. **Giao Thức Mạng Dây (Wire Protocol)**:
   - Bổ sung opcode trong `protocol.go`:
     ```go
     const TypeGuHouseIntegrate = "world.gu_house_integrate"
     ```
   - Định dạng gói tin Client gửi (`GuHouseIntegrateIntent`):
     ```json
     {
       "v": 1,
       "type": "world.gu_house_integrate",
       "request_id": "req-101",
       "payload": {
         "action": "ENTER" // hoặc "EXIT"
       }
     }
     ```
   - Server phản hồi:
     - Thất bại: Gửi `world.reject` kèm mã lỗi (`gu_house_not_equipped`, `gu_house_broken`, `hard_cc_active`, `silenced`, `insufficient_mana`).
     - Thành công: Broadcast qua `world.events` sự kiện `TRANSFORMATION_STARTED` (`form_kind: "GU_HOUSE"`). Khi rời phòng broadcast `TRANSFORMATION_ENDED`.

2. **Tiêu Chuẩn Kiểm Định Điều Kiện Server (Server Validation Matrix)**:
   - **Trang bị**: Kiểm tra thực thể người chơi có bản ghi Tiên Cổ Phòng đang trang bị (`player_gu_houses.is_equipped = true`). Nếu không có: Reject `gu_house_not_equipped`.
   - **Độ bền**: `current_durability > 0`. Nếu độ bền = 0 (đã hỏng nặng do vỡ trận trước đó): Reject `gu_house_broken`.
   - **Khống chế**: Không dính các trạng thái vô hiệu hóa: `STUN`, `FREEZE`, `SILENCE`, `GU_SEAL`. Nếu vi phạm: Reject `hard_cc_active` hoặc `silenced`.
   - **Trạng thái hành động**: `caster.Combat.Pending == nil` (không đang dở niệm chú chiêu khác).
   - **Chi phí khởi động**: Trừ 50 Tiên Nguyên (`InitialManaCost = 50`) kích hoạt đại trận.

3. **Giai Đoạn Niệm Chú (Windup 2.0s = 40 ticks @ 20Hz)**:
   - Server chuyển `TransformationState`:
     ```go
     entity.Transformation = TransformationState{
         Active: false,
         FormID: house.ID,
         FormName: house.Name,
         FormKind: TransformationGuHouse,
         IsWindup: true,
         WindupEndTick: w.tick + 40,
     }
     ```
   - Phát event `EventCastAccepted` (`SkillID: "gu_house_integrate"`).
   - Trong Snapshot 20Hz, trường `transformation` gửi `is_windup: true`, `windup_progress: elapsed / 40.0`.
   - Client hiển thị thanh tụ lực màu vàng kim trên đầu nhân vật.
   - **Ngắt chiêu khi dính Hard CC**: Nếu trong 40 ticks này Cổ Tiên bị trúng Stun hoặc Freeze, server kích hoạt `InterruptTransformation`, phát event `EventCastInterrupted` (`Reason: "cc_interrupted_STUN"`), hủy bỏ windup, Cổ Tiên không vào được Cổ Phòng và mất 50 Tiên Nguyên khởi động.

4. **Kích Hoạt Hoàn Tất Dung Nhập**:
   - Khi `w.tick >= WindupEndTick`:
     - Chuyển `Transformation.Active = true`, `IsWindup = false`.
     - Cấp `MaxFormHP` và `CurrentFormHP` từ Cổ Phòng (ví dụ 4,500 HP).
     - Áp dụng `FormDefenseBonus` (+80 giáp) và `DamageReduction` (25% giảm sát thương).
     - Phóng to Hitbox bán kính va chạm: `entity.Radius = originalRadius * 1.8`.
     - Sao lưu thanh kỹ năng người (`SavedSkills`) và hoán đổi sang 6 Trận Kích của Cổ Phòng (`SwapSkillPalette`).
     - Khóa các slot Alt 9..12 (`form_skill_locked`).
     - Phát event `EventTransformationStarted` (`form_kind: "GU_HOUSE"`).

5. **Giải Trừ Dung Nhập (EXIT)**:
   - Khi người chơi bấm Slot 6 (Lệnh rời Cổ Phòng) hoặc nhấn phím tắt `Alt + G` khi đang ở trong Cổ Phòng:
     - Sim thực thi giải trừ an toàn không cần windup:
     - Phục hồi Hitbox gốc `OriginalRadius`.
     - Phục hồi thanh kỹ năng 1..6 của Cổ Tiên (`RestoreSkillPalette`).
     - Reset `TransformationState.Active = false`.
     - Phát event `EventTransformationEnded` (`Reason: "voluntary_exit"`).

6. **Tài Sản Kiểm Chứng Logic**:
   - Prototype tương tác đầy đủ kịch bản đã được xây dựng tại:
     [`backend/internal/realtime/sim/prototype_gu_house_integration_demo.html`](file:///home/vippro67/GitHub/GuWorld/backend/internal/realtime/sim/prototype_gu_house_integration_demo.html)
