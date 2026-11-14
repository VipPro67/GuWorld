# Ticket 01: Vòng đời Dung Nhập và Thoát Cổ Phòng trong Combat Theo Lượt

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để Simulator (`backend/internal/simulator/`) và hệ thống Interactive Combat (`InteractiveCombatControls`) quản lý trạng thái Dung Nhập (Biến thân vào Tiên Cổ Phòng) và Thoát Cổ Phòng trong trận chiến theo lượt?

Cụ thể cần giải quyết:
1. **Phân hóa Auto-mode vs Interactive-mode**:
   - Ở chế độ Auto Combat (`POST /api/combat/start`): Nhân vật có Cổ Phòng đang trang bị (`is_equipped = true`, `current_durability > 0`) sẽ tự động xuất chiến trong trạng thái Dung Nhập ngay tại Turn 1.
   - Ở chế độ Interactive Combat (`interactive.go`): Cổ Tiên có thể chọn hành động đặc biệt "Dung Nhập Tiên Cổ Phòng" (tốn 1 lượt hoặc 0 lượt / instant action) khi đến lượt, hoặc chọn vào ngay từ đầu nếu cài đặt cấu hình xuất chiến.
2. **Hành động Thoát Cổ Phòng chủ động (Disembark / Voluntary Exit)**:
   - Trong Interactive Combat, Cổ Tiên có thể ra lệnh "Thoát Cổ Phòng" để chuyển quyền nhận sát thương và bộ chiêu thức về lại nhục thân (bảo toàn Cổ Phòng khi Form HP còn thấp trước đòn chí mạng).
3. **Quản lý Stance & Palette Swapping trong Simulator**:
   - Trường trạng thái nào trên `CombatantAuto` đại diện cho việc đang ở trong Cổ Phòng (`GuHouseActive: bool`)?
   - Khi `GuHouseActive == true`, simulator lọc và ưu tiên các chiêu thức Trận Kích như thế nào so với bộ chiêu thức Cổ trùng cá nhân?
4. **Kiểm tra Điều kiện Tiên quyết (Preconditions)**:
   - Không cho phép Dung Nhập nếu Cổ Phòng đã vỡ (`current_durability <= 0`), nhân vật đang bị Choáng (`IsStunned`), Đóng băng (`IsFrozen`), hoặc bị Phong ấn Cổ (`HasGuSeal`).

## Answer

Đã chuẩn hóa toàn diện mô hình trạng thái (State Machine) vòng đời Dung Nhập và Thoát Tiên Cổ Phòng trong hệ thống Turn-based Combat:

1. **Phân Hóa Chế Độ Chiến Đấu (Auto vs Interactive)**:
   - **Auto Combat (`POST /api/combat/start`)**:
     - Khi build `CombatantAuto` từ player có `is_equipped = true` và `current_durability > 0`, hệ thống tự động gán `GuHouse.Active = true`.
     - Ngay từ Turn 1, Cổ Tiên tự động xuất chiến bên trong Cổ Phòng, nhận Form HP đầy đủ, và hệ thống chỉ nạp bộ Trận Kích vào danh sách `Actions` hành động.
   - **Interactive Combat (`interactive.go` & `InteractiveCombatControls.tsx`)**:
     - Hỗ trợ cờ khởi tạo: mặc định tự động vào Cổ Phòng nếu đã trang bị; hoặc xuất chiến ở dạng nhục thân nếu người chơi muốn thăm dò.
     - Khi ở dạng nhục thân: xuất hiện hành động đặc biệt `ENTER_GU_HOUSE` ("Dung Nhập Tiên Cổ Phòng").
     - Khi thực thi lệnh `ENTER_GU_HOUSE`, simulator chuyển `actor.GuHouse.Active = true`, kích hoạt Form HP và hoán đổi thanh kỹ năng sang Trận Kích.

2. **Lệnh Rời Phòng Chủ Động (Slot 6: Disembark / Voluntary Exit)**:
   - Khi `GuHouse.Active == true`, danh sách chiêu thức luôn có lệnh Slot 6 `EXIT_GU_HOUSE` ("Lệnh Rời Cổ Phòng", `TargetType: "SELF"`, `ManaCost: 0`).
   - Lệnh này cho phép Cổ Tiên chủ động rút về nhục thân khi nhận thấy Cổ Phòng sắp cạn Form HP trước đòn đánh nguy hiểm của đối phương, qua đó bảo toàn độ bền Cổ Phòng khỏi thảm họa Vỡ Phòng.
   - Khi kích hoạt, `actor.GuHouse.Active = false`, phục hồi bảng chiêu thức Cổ trùng nhục thân.

3. **Cấu Trúc Trạng Thái & Palette Swapping trong Simulator**:
   - Mở rộng struct `GuHouseState` trong `models.go` với trường `Active bool`.
   - Trong `combatActionAvailable` và `selectCombatCommand`:
     - Khi `GuHouse != nil && GuHouse.Active`: Chỉ cho phép thi triển các chiêu thức Trận Kích (`IsFromGuHouse == true`) và lệnh `EXIT_GU_HOUSE`. Toàn bộ chiêu thức Cổ trùng cá nhân bị khóa ẩn.
     - Khi `GuHouse == nil || !GuHouse.Active`: Cho phép thi triển các chiêu thức cá nhân và lệnh `ENTER_GU_HOUSE` (nếu có Cổ Phòng chưa bị vỡ).
   - Trong tính toán sát thương: Đòn đánh chỉ chuyển hướng vào `GuHouse.HP` khi `GuHouse != nil && GuHouse.Active && !GuHouse.IsDestroyed`.

4. **Ma Trận Kiểm Định Điều Kiện Tiên Quyết (Preconditions Matrix)**:
   - Lệnh `ENTER_GU_HOUSE` bị từ chối nếu thỏa bất kỳ điều kiện nào sau:
     - Cổ Phòng bị vỡ: `GuHouse.IsDestroyed == true` hoặc `current_durability <= 0`.
     - Cổ Tiên dính khống chế cứng: `IsStunned == true`, `IsFrozen == true`, `IsSleeping == true`, `IsBound == true`.
     - Mạch trận Cổ bị khóa: `HasGuSeal == true`.
   - Lỗi được trả về rõ ràng qua message validation của Interactive Session.

5. **Tài Sản Kiểm Chứng Logic Độc Lập**:
   - Toàn bộ state machine, logic hoán đổi palette, các điều kiện biên và 4 kịch bản walkthrough đã được kiểm chứng qua file demo tương tác tại:
     [`docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/01_turnbased_gu_house_lifecycle.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/01_turnbased_gu_house_lifecycle.html)
