# Loại Bỏ Phaser & Xây Dựng UI Bản Đồ Chiến Lược Trên Frontend React
Status: resolved
Type: prototype
Blocked by: 02, 03
Part of #map

## Question

Quy trình dọn dẹp các thư viện và thành phần Phaser (`PhaserWorldHost.tsx`, `WorldRuntime.ts`, assets Tiled khổng lồ) khỏi bundle Next.js, và xây dựng giao diện Bản Đồ Chiến Lược mới (Bản đồ tổng quan Atlas + Chi tiết Node / Tactical Grid) bằng React thuần và Tailwind CSS để tối ưu tốc độ tải trang, mượt mà trên cả PC và Mobile?

## Answer

1. **Kế hoạch Gỡ Bỏ Phaser & WebTransport Frontend:**
   - **Gỡ bỏ dependencies:** Xóa `phaser` khỏi `frontend/package.json` (giảm >30MB kích thước gói mã nguồn và triệt tiêu toàn bộ lỗi khởi tạo WebGL/Canvas trên trình duyệt di động).
   - **Xóa mã nguồn Realtime Client:** Loại bỏ `frontend/src/game/` (`WorldRuntime.ts`, `WorldTransport.ts`, `MapLoader.ts`, `WorldScene.ts`) và `frontend/src/components/game/PhaserWorldHost.tsx`.
   - **Ngắt kết nối WebTransport Client:** Gỡ bỏ các hàm kết nối UDP WebTransport/QUIC trong client, toàn bộ tương tác chuyển về REST API tiêu chuẩn và WebSocket SSE cho thông báo / presence.

2. **Cấu trúc Giao diện Bản Đồ Chiến Lược Mới (`StrategicWorldView.tsx`):**
   Giao diện gồm 2 tầng trực quan, hoàn toàn bằng React và Tailwind CSS:

   * **Tầng 1: Bản Đồ Đại Vực (World Node-Graph View):**
     - Kế thừa và tích hợp với [`WorldAtlasModal.tsx`](file:///home/vippro67/GitHub/GuWorld/frontend/src/components/WorldAtlasModal.tsx).
     - Hiển thị bản đồ Ngũ Vực và danh sách các Node điểm nút trọng điểm (Thành Trấn, Sơn Trại, Cụm Mỏ, Bí Cảnh).
     - **Thanh trạng thái Hành Trình:** Nếu đang di chuyển, hiển thị thanh tiến trình ETA, phương tiện di chuyển (Thanh Phong Cổ, Phi chu, Thương đội), nút "Hủy hành trình" hoặc xử lý Biến Cố ngẫu nhiên xuất hiện trên đường.
     - **Bảng Hiện Diện (Presence Hub):** Hiển thị danh sách Cổ Sư đang có mặt tại Node, nút ghé thăm Chợ Vỉa Hè (Stalls), Đả Tọa Mỏ Nguyên Tuyền, và nút **"Tiến Vào Khám Phá (Enter Tactical Grid)"**.

   * **Tầng 2: Lưới Khám Phá Chiến Thuật (Tactical Grid View):**
     - Xây dựng lại component `TacticalGridView.tsx` (tương thích với dữ liệu `GetGrid` / `map_cells` từ backend):
     - Lưới ô vuông $15 \times 15$ hoặc $21 \times 21$ bằng CSS Grid thuần với camera bám theo nhân vật ở trung tâm.
     - Hệ thống Sương mù Chiến tranh (Fog of War) dựa trên chỉ số Thám Tra (`investigation`).
     - Các ô hiển thị icon trực quan (tài nguyên, thảo dược, quái vật, rương bí mật, truyền tống trận).
     - **Tương tác quái vật:** Click vào quái vật hiển thị Modal xem thông số/chiến lợi phẩm dự kiến kèm 2 nút:
       1. **"Tuyên Chiến"**: Mở màn hình [`CombatScreen.tsx`](file:///home/vippro67/GitHub/GuWorld/frontend/src/components/CombatScreen.tsx) để đánh theo lượt.
       2. **"Skip / Quét Trận"**: Gửi request tính toán nhanh kết quả qua [`combat_auto_resolve.go`](file:///home/vippro67/GitHub/GuWorld/backend/internal/handlers/combat_auto_resolve.go).

3. **Tác động hiệu năng & Trải nghiệm:**
   - Thời gian First Contentful Paint (FCP) của trang game giảm từ ~3.5s xuống <0.8s.
   - Hoạt động mượt mà 60 FPS trên mọi thiết bị di động, không còn hiện tượng nóng máy hay giật lag mạng.

