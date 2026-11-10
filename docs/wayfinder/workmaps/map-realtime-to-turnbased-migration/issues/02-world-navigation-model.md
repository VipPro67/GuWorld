# Mô Hình Định Tuyến & Di Chuyển Thế Giới (Navigation Model)
Status: resolved
Type: grilling
Blocked by: 01
Part of #map

## Question

Mô hình di chuyển và bản đồ thế giới mới sẽ được thiết kế theo cấu trúc nào để thay thế cho cơ chế WASD / Click-to-move liên tục của Phaser 2D?
- Lựa chọn A: Khôi phục Grid Từng Ô (Tile Step - toạ độ x, y) dựa trên bảng `map_cells` và tầm nhìn sương mù (Vision Radius), mỗi bước đi tốn Chân Nguyên / Thể Lực.
- Lựa chọn B: Mô hình Đồ thị Điểm (Node-Graph POI) kết nối các địa danh (Sơn Trại, Mỏ Khoáng, Bí Cảnh) với Thời gian di chuyển thực tế (Travel Time/ETA) và cơ chế Kỳ Ngộ dọc đường.
- Lựa chọn C: Mô hình Lai (Hybrid) — Cấp Vực / Thế giới dùng Node-Graph POI; khi bước vào một Khu vực / Bí Cảnh cụ thể thì mở ra Tactical Grid để thám hiểm từng ô.
Mô hình nào tối ưu nhất cho trải nghiệm tu tiên, kinh tế, và tính năng di chuyển bằng Cổ trùng / Thương đội?

## Answer

Thống nhất lựa chọn **Mô hình Lai (Hybrid Model)** theo Lựa chọn C:

1. **Cấp Đại Vực & Thế Giới (Macro World: Node-Graph POI):**
   - Bản đồ Ngũ Vực (Nam Cương, Tây Mạc, Bắc Nguyên, Đông Hải, Trung Châu) và Cửu Thiên hiển thị dưới dạng mạng lưới các Điểm Nút chiến lược (Thành Trấn, Sơn Trại Môn Phái, Cụm Mỏ Lớn, Cổng Bí Cảnh, Trạm Giới Bích).
   - Di chuyển giữa các Node áp dụng **Thời gian hành trình (Travel Time / ETA)**:
     - Tốc độ di chuyển phụ thuộc vào: Cảnh giới tu vi, Cổ di chuyển mang theo (Thanh Phong Cổ, Bạch Hạc Cổ...), hoặc hình thức di chuyển (Đi bộ, Gia nhập Thương Đội, Truyền Tống Trận, Vượt Giới Bích).
     - Trong lúc di chuyển, người chơi ở trạng thái "Đang trên đường" (In-transit), có thanh tiến trình và tỷ lệ kích hoạt Biến Cố / Kỳ Ngộ dọc đường (thú triều, ma đạo cướp bóc, thảo dược hiếm, gặp người chơi khác).

2. **Cấp Tiểu Vực & Địa Danh / Bí Cảnh (Micro Exploration: Tactical Grid):**
   - Khi người chơi dừng chân tại một Node khám phá (ví dụ: *Thanh Mao Sơn*, *Hang Động Huyền Bí*, *Bạch Cốt Bí Cảnh*, hoặc *Khu Mỏ*), giao diện mở ra bản đồ lưới **Tactical Grid ($x, y$)** kế thừa từ bảng `map_cells`.
   - Mỗi bước đi tiêu hao Chân Nguyên / Thể Lực, mở sương mù chiến tranh dựa trên chỉ số Thám Tra (`investigation`), phát hiện tài nguyên ẩn, bẫy rập, rương báu hoặc quái vật canh giữ.
   - Cơ chế này hoàn toàn không cần vòng lặp 20Hz hay engine vật lý: việc di chuyển chỉ là các request API nhẹ nhàng kiểm tra toạ độ hợp lệ qua [`backend/internal/handlers/movement.go`](file:///home/vippro67/GitHub/GuWorld/backend/internal/handlers/movement.go).

