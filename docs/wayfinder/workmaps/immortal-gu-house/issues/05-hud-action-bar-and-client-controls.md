# Ticket 05: Thiết kế giao diện HUD Action Bar, Hotkeys và Hiển thị Client khi Dung Nhập

Type: prototype
Status: resolved
Blocked by: 01, 02

## Question

Client frontend (React UI + Phaser canvas `WorldScene.ts`) cần hiển thị nút điều khiển, hoán đổi giao diện và phản hồi hiệu ứng như thế nào khi Cổ Tiên Dung Nhập vào Tiên Cổ Phòng?

Cụ thể cần giải quyết các vấn đề:
1. Nút kích hoạt Dung Nhập trên HUD:
   - Khi Cổ Tiên đã trang bị Tiên Cổ Phòng (`is_equipped = true`), hiển thị nút icon Tiên Cổ Phòng (Castle icon) cạnh Action Bar hoặc phím tắt `Alt + G`.
   - Hiển thị thanh tiến trình niệm chú Windup 2.0s trên đầu nhân vật.
2. Hoán đổi giao diện Action Bar khi ở trong Cổ Phòng:
   - Khung viền Action Bar đổi sang chủ đề Hoàng Kim Kim Cang (Gold/Amber border rực rỡ).
   - 6 ô phím tắt số [1]..[6] hiển thị 6 chiêu thức Trận Kích (Pháo kích, Xung kích, Phòng hộ, Sửa chữa, Thần thông hủy diệt, Rời phòng).
   - Slot [6] có icon Giải Trừ Dung Nhập (màu đỏ cam hoặc nút cửa mở).
3. Thanh Form HP kép và Hiệu ứng Canvas:
   - Thanh máu Form HP Cổ Phòng (màu vàng kim `#eab308` hoặc tím) hiển thị ngay phía trên thanh HP đỏ của nhân vật, kèm số máu cụ thể.
   - Hiệu ứng hào quang kết giới hoàng kim lấp lánh (Gold aura ring) dưới chân thực thể theo tỉ lệ scale model (1.8x - 2.2x).
   - Hiệu ứng vỡ tan (Shatter particles) và rung màn hình (Camera shake) khi Cổ Phòng bị vỡ.

## Answer

Đã chuẩn hóa toàn diện thiết kế giao diện HUD, hệ thống phím tắt và hiệu ứng hiển thị client khi Cổ Tiên Dung Nhập Tiên Cổ Phòng:

1. **Nút Điều Khiển Dung Nhập Trên HUD (`MMOHud.tsx`) & Phím Tắt `Alt + G`**:
   - Khi Cổ Tiên trang bị Tiên Cổ Phòng (`guHouses.some(h => h.is_equipped)`), HUD hiển thị một nút thao tác nổi (*Floating Action Button*) đặt phía trên Desktop Hotbar với icon Lâu Đài 🏰 cùng nhãn `Dung Nhập Tiên Cổ Phòng [Alt+G]`.
   - Phím tắt `Alt + G` được đăng ký trong `matchHotbarSlot` / `WorldScene.ts` cho phép kích hoạt nhanh bằng bàn phím. Khi đang ở trong Cổ Phòng, nút tự động đổi nhãn và màu sắc sang `Rời Tiên Cổ Phòng [Alt+G]` (icon 🚪).
   - **Thanh tiến trình Windup 2.0s**: Khi nhận trạng thái `transformation.is_windup = true`, thanh tụ lực màu xanh lam / vàng kim hiển thị phía trên đầu nhân vật với tỉ lệ hoàn tất từ 0.0 đến 1.0 (40 ticks @ 20Hz).

2. **Hoán Đổi Giao Diện Action Bar Hoàng Kim Kim Cang**:
   - Khi thực thể kích hoạt `transformation.active` với `form_kind == "GU_HOUSE"`:
     - Khung Action Bar hoán đổi sang theme Hoàng Kim Kim Cang rực rỡ: viền kép màu vàng kim `#d29922`, nền nâu sẫm hổ phách gradient, hào quang phát sáng `box-shadow: 0 0 30px rgba(210, 153, 34, 0.35)`.
     - 6 slot số [1]..[6] biểu diễn đầy đủ 6 chiêu thức Trận Kích:
       - **[1]**: Pháo Kích (Artillery Bombardment)
       - **[2]**: Xung Kích (Siege Trample Dash)
       - **[3]**: Đại Trận (Fortress Barrier Shield)
       - **[4]**: Tu Bổ (Emergency Mending)
       - **[5]**: Tuyệt Diệt (Ultimate Destruction Strike)
       - **[6]**: Rời Phòng (Disembark Revert) với viền tím nhạt / đỏ cam nổi bật.
     - Các phím Alt 9..12 bị ẩn hoặc khóa hiển thị, tập trung toàn bộ giao diện vào 6 phím chiến đấu của Cổ Phòng.

3. **Hiển Thị Thanh Máu Kép & Hiệu Ứng Canvas (`WorldScene.ts`)**:
   - **Thanh máu Form HP**: Thanh máu màu vàng kim lấp lánh `#f1c40f` (hoặc `#eab308`) xuất hiện song song ngay phía trên thanh máu đỏ truyền thống của Cổ Tiên, hiển thị chính xác tỉ lệ `CurrentFormHP / MaxFormHP`. Khi nhận sát thương, thanh Form HP tụt trước.
   - **Hào quang Kim Cang Trận (Aura Ring)**: Dưới chân nhân vật vẽ vòng elip kết giới hoàng kim chu kỳ xoay và nhấp nháy 1.8s (`0xeab308`), bán kính mở rộng theo `model_scale: 1.8x`. Avatar nhân vật phóng to thể hiện vóc dáng cự đại của Cổ Phòng.
   - **Hiệu ứng Vỡ Phòng (Structural Breach VFX)**: Khi nhận sự kiện `TRANSFORMATION_BROKEN`, client kích hoạt hệ thống hạt vỡ tan `vfx.spawnShatterParticles(pixels.x, pixels.y, 0xeab308)` kèm hiệu ứng rung màn hình chấn động `cameras.main.shake(300, 0.015)`.

4. **Tài Sản Kiểm Chứng Giao Diện Prototype**:
   - Prototype trực quan đầy đủ HUD, avatar, aura ring, phím `Alt+G`, 6 phím [1]..[6], thanh máu kép và rung màn hình đã được xây dựng tại:
     [`backend/internal/realtime/sim/prototype_gu_house_hud_demo.html`](file:///home/vippro67/GitHub/GuWorld/backend/internal/realtime/sim/prototype_gu_house_hud_demo.html)
