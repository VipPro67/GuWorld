# Ticket 05: Thiết kế Giao diện Quản lý Đàn Cổ Tiên Khiếu trên Client (Aperture Gu Swarm UI/UX)

Type: prototype
Status: resolved
Blocked by: 02

## Question

Làm thế nào để giao diện Client (React + Next.js trong `frontend/src/components/Aperture.tsx` hoặc tab chuyên biệt) hiển thị hàng vạn con Phàm Cổ một cách mượt mà, trực quan, không lag giật và mang đậm phong thái Cổ Tiên?

Cụ thể cần giải quyết:
1. **Thiết kế Thẻ Đàn Cổ (Swarm Card)**:
   - Hiển thị Cổ theo từng loại `(gu_id, rank)` kèm huy hiệu số lượng lớn (ví dụ: `x12.5k`, `x50.0k`).
   - Có bộ lọc theo Lưu Phái (Kiếm Đạo, Lôi Đạo, Ám Đạo, Biến Hóa Đạo...).
   - Hiển thị mô tả ngắn, thức ăn của loài Cổ này, và trạng thái nuôi dưỡng của Tiên Khiếu.
2. **Thao Tác Tương Tác Nhanh (Action Controls)**:
   - Nút "Trang bị (Rút 1 con)": Bấm vào để rút 1 con cá thể lắp ngay vào slot trang bị Không Khiếu.
   - Nút "Thả vào Tiên Khiếu (Nhập đàn toàn bộ)": Một chạm thu hồi tất cả các Phàm Cổ lành lặn đang nằm rải rác trong túi đồ trở về Tiên Khiếu.
3. **Hiệu Suất Hiển Thị (Rendering Performance)**:
   - Vì số loại Phàm Cổ khác nhau trong game chỉ tầm 100-200 loại, danh sách Đàn Cổ chỉ có tối đa 100-200 card, Client render cực kỳ nhẹ và nhanh.
   - Thử nghiệm prototype component để bảo đảm độ phản hồi và trải nghiệm người dùng liền mạch.

## Answer

Đã chuẩn hóa toàn bộ kiến trúc giao diện người dùng (UI/UX) và kiểm chứng trải nghiệm trực quan tại [`prototypes/05-frontend-aperture-gu-swarm-ui.html`](../prototypes/05-frontend-aperture-gu-swarm-ui.html):

### 1. Cấu Trúc Giao Diện Chuẩn Hóa (UI Components)
* **Thanh Thống Kê Tổng Quan (Top Stats Bar)**:
  - Tổng số lượng Phàm Cổ đang nuôi dưỡng (ví dụ: `68,450 con`).
  - Số lượng loài Cổ độc lập (`24 loài`).
  - Số slot Cổ cá thể đang trang bị trên người (`14 / 60 slot`).
  - Trạng thái sinh thái Tiên Khiếu: `● Tự Cung Dưỡng (0 Tiêu Hao)`.
* **Thẻ Đàn Cổ (Swarm Card)**:
  - Tên Cổ và Icon chuẩn theo Rank (`Rank 1 - 5`).
  - Huy hiệu Lưu Phái có màu sắc nhận diện (`Nguyệt Đạo`, `Phong Đạo`, `Lôi Đạo`...).
  - Huy hiệu số lượng lớn nổi bật định dạng gọn (`x25.0k`, `x18.4k`, `x6.5k`).
  - Thức ăn đặc thù và mô tả ngắn.
  - Phím tắt hành động nhanh: Nút *"⚡ Trang Bị (1 con)"* bốc 1 con cá thể lắp thẳng vào Không Khiếu mà không cần mở nhiều popup.
* **Bộ Lọc Đa Tầng (Search & Filter Bar)**:
  - Tìm kiếm tức thì theo Tên hoặc ID Cổ.
  - Bộ lọc dropdown theo Lưu Phái (`Path`).
  - Bộ lọc dropdown theo Phẩm giai (`Rank 1 - 5`).

### 2. Thao Tác Thu Gom Một Chạm (Batch Deposit Control)
* Đặt nút nổi bật *"📥 Thả Tất Cả Phàm Cổ Lành Lặn Vào Tiên Khiếu"* ngay trên Header.
* Khi Cổ Tiên bấm nút, toàn bộ Phàm Cổ nhặt được trong túi thân sẽ được hòa nhập vào các đàn tương ứng; hiển thị Toast thông báo số lượng Cổ đã thu hồi.

### 3. Hiệu Năng Render Tuyệt Đối (Zero Lag @ 60 FPS)
* **Triệt tiêu bùng nổ DOM**: Thay vì phải render 50.000 phần tử DOM của từng con Cổ cá thể (vốn gây đơ trình duyệt và crash mobile), Client chỉ render từ 20 đến tối đa 150 card của các bầy đàn.
* **Tích hợp vào Codebase**: Đã hoàn tất tích hợp trong Tab `"swarms"` của `frontend/src/components/Aperture.tsx` khi người chơi đạt cảnh giới Cổ Tiên (`isImmortal = true`), bổ sung các phím tắt "Thả Vào Đàn" tiện ích trong `frontend/src/components/GuInventory.tsx`, và hỗ trợ đầy đủ các actions `fetchApertureSwarms`, `materializeGuFromSwarm`, `dematerializeGuToSwarm`, `depositAllHealthyGus` trong `frontend/src/store/slices/apertureSlice.ts`.

