# Ticket 05: Giao diện Form HP kép, Aura Cổ Phòng và Bộ Điều Khiển Interactive

Type: prototype
Status: resolved
Blocked by: Ticket 01, Ticket 02
Assignee: Antigravity

## Question

Làm thế nào để nâng cấp giao diện chiến đấu `CombatScreen.tsx` và `InteractiveCombatControls.tsx` để thể hiện sinh động trạng thái Dung Nhập Tiên Cổ Phòng, thanh máu kép Form HP màu hoàng kim, hiệu ứng aura pháo đài và bảng nút chọn chiêu thức Trận Kích chuyên biệt?

Cụ thể cần giải quyết:
1. **Thanh Máu Form HP Kép (Dual HP Bar)**:
   - Trên thanh trạng thái của combatant (cả người chơi và đối thủ nếu có Cổ Phòng):
     - Hiển thị thanh máu Form HP màu vàng kim hổ phách (`amber-500` / `yellow-400`) nằm xếp chồng hoặc song song phía trên thanh HP nhục thân (`emerald-500` / `red-500`).
     - Hiển thị rõ số lượng Form HP hiện tại / tối đa (ví dụ: `[Cổ Phòng] 4,200 / 5,000`).
2. **Hiệu Ứng Aura & Huy Hiệu Biến Thân (Visual Aura & Stance Badge)**:
   - Khi `GuHouseActive == true`:
     - Avatar hoặc khung combatant có viền hào quang kim quang rực rỡ (`ring-2 ring-amber-400 shadow-amber-500/50`).
     - Huy hiệu icon Cổ Phòng hiển thị cạnh tên nhân vật kèm Rank Cổ Phòng (ví dụ: `[Tiên Cổ Ốc - Rank 6]`).
3. **Bộ Nút Điều Khiển Interactive (`InteractiveCombatControls.tsx`)**:
   - Khi đang Dung Nhập Cổ Phòng:
     - Bảng chiêu thức nhóm lại hoặc làm nổi bật 6 nút Trận Kích chuẩn hóa với theme phong cách pháo đài/trận pháp.
     - Hiển thị nút "Thoát Cổ Phòng" (Slot 6) riêng biệt để người chơi có thể ra quyết định giải trừ khẩn cấp trước khi Cổ Phòng bị công phá.
     - Nếu chưa vào Cổ Phòng nhưng có Cổ Phòng khả dụng: Hiển thị nút kích hoạt "Dung Nhập Cổ Phòng".
4. **Phản Hồi Trực Quan Khi Vỡ Phòng**:
   - Khi sự kiện `GU_HOUSE_DESTROYED` xảy ra trong chiến báo (hoặc trong interactive turn): Hiển thị hiệu ứng vỡ nứt màn hình (screen shake nhẹ), thông báo đỏ cảnh báo phản phệ và thanh máu Cổ Phòng vỡ vụn.

## Answer

Đã chuẩn hóa toàn bộ thiết kế giao diện UX/UI và hoàn thiện prototype tương tác trực quan cho Tiên Cổ Phòng trên màn hình chiến đấu theo lượt:

1. **Thanh Máu Form HP Kép Hoàng Kim (Dual HP Bar)**:
   - Trên card chiến binh (`TeamCombatant`), thanh Form HP được đặt nằm ngay trên thanh HP nhục thân với dải màu gradient hoàng kim đặc trưng (`from-amber-600 via-amber-400 to-yellow-300`).
   - Nhãn hiển thị số lượng rõ ràng: `🛡️ Form HP Cổ Phòng: {formHp} / {maxFormHp}`.
   - Sát thương nhận vào sẽ trừ trực quan vào thanh Form HP trước, bảo vệ trọn vẹn thanh HP xanh lục của nhục thân.
2. **Hiệu Ứng Hào Quang Aura & Huy Hiệu Biến Thân (Visual Aura & Badge)**:
   - Khi `GuHouse.Active == true`, card chiến binh mang class `has-gu-house` với animation nhịp thở kim quang `pulseAura` (`box-shadow: 0 0 20px rgba(245, 158, 11, 0.4)`).
   - Avatar có viền vàng kim rực rỡ; cạnh tên nhân vật xuất hiện huy hiệu phong thái pháo đài: `[🏛️ Tiên Cổ Ốc · Rank X]`.
3. **Bảng Nút Điều Khiển Interactive Chuyên Biệt**:
   - 6 nút Trận Kích chuẩn hóa được tạo khung theme trận pháp hoàng kim (`border-amber-500/50`, nền gradient hổ phách mờ).
   - Nút Slot 6 hoán đổi thông minh:
     - Khi đang ở trong Cổ Phòng: Hiển thị nút `🚪 [6] Lệnh Rời Phòng` (màu đỏ cảnh báo nhẹ) để người chơi có thể rút về nhục thân khi Form HP sắp cạn.
     - Khi ở dạng nhục thân nhưng Cổ Phòng còn nguyên: Hiển thị nút `🏛️ [6] Dung Nhập Cổ Phòng` để tái kích hoạt pháo đài.
   - Khi Slot 5 Thần Thông Tối Hậu đang tụ lực: Nút chuyển sang trạng thái rực lửa `⚡ [5] BẮN ĐẠI CHIÊU DIỆT THẾ!` sẵn sàng phóng thích.
4. **Phản Hồi Đồ Họa Khi Vỡ Phòng (Breach Screen Shake)**:
   - Khi xảy ra sự kiện Vỡ Phòng, hiệu ứng rung màn hình nhẹ (`shake` animation) được kích hoạt trên đấu trường.
   - Thanh Form HP tan biến, card chuyển sang trạng thái cảnh báo đỏ `card breached` với huy hiệu `💥 Vỡ Phòng (Choáng 2 lượt)`.
5. **Tài Sản Kiểm Chứng Trực Quan**:
   - Prototype độc lập hoàn chỉnh với đầy đủ tương tác, hiệu ứng CSS animation và thanh công cụ giả lập tình huống chiến đấu đã được xây dựng tại:
     [`docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/05_combat_screen_hud_prototype.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/05_combat_screen_hud_prototype.html)
