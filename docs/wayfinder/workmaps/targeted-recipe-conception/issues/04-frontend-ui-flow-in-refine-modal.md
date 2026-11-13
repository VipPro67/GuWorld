Type: prototype
Status: resolved
Blocked by: 03

## Question

Thiết kế giao diện người dùng và luồng tương tác trên Frontend:
1. Bố cục Tab "Khai Sáng Bí Phương" trong `RefineModal.tsx`: Thanh chọn Lưu Phái và Rank, danh sách thẻ Cổ mục tiêu (kèm hình ảnh, tên, rank, mô tả ngắn).
2. Khu vực chuẩn bị suy tính: Ô đặt Vật Dẫn (Cổ phôi hoặc Nguyên liệu), thanh hiển thị Tỉ lệ trúng đích dự kiến, Tiêu hao Niệm Lực / Nguyên Thạch, và Cảnh báo Phản Phệ màu đỏ nếu cố suy tính Cổ vượt cấp.
3. Hiển thị kết quả: Modal/Animation thông báo kết quả sinh động (Thành công khai sáng đúng Cổ đích / Khai sáng lệch hướng sang Cổ liên quan / Linh quang chớp lóe 100% / Bị Trí Đạo Phản Phệ chấn thương).
4. Tích hợp nút chuyển nhanh từ Cổ Phổ (Gu Codex).

## Answer

### 1. Bố Cục Tab "Khai Sáng Bí Phương" trong Lò Luyện Cổ (`RefiningFurnace.tsx`)
Tại thanh điều hướng của `RefiningFurnace`, bổ sung Tab chuyển đổi chế độ:
- Tab 1: **"Lò Luyện & Thôi Diễn"** (Chức năng hợp luyện cổ và bồi dưỡng tàn phương hiện tại).
- Tab 2: **"Khai Sáng Bí Phương"** (Tính năng định hướng suy tính mới).

```
+-----------------------------------------------------------------------------------+
|  [Lò Luyện & Thôi Diễn]   |  ✨ [Khai Sáng Bí Phương]                             |
|  Niệm Lực: 160 / 200 🧠   |  Nguyên Thạch: 15,200 🪙   |  Tiên Thạch: 0 💎         |
+-----------------------------------------------------------------------------------+
|  BỘ LỌC CỔ ĐÍCH:                                                                  |
|  [ Lưu Phái: Kiếm Đạo v ]  [ Phẩm Giai: Chuyển 3 v ]  [ Tìm kiếm: Kiếm...       ] |
+-----------------------------------------+-----------------------------------------+
|  DANH SÁCH CỔ CÓ THỂ SUY TÍNH           |  ĐÀN SUY DIỄN THIÊN CƠ (SETUP PANEL)   |
|                                         |                                         |
|  +-----------------------------------+  |  [ HÌNH ẢNH LỚN: KIẾM ẢNH CỔ (R3) ]     |
|  | [Icon] Kiếm Ảnh Cổ (R3)           |  |  Hệ: Kiếm Đạo | Phẩm: Tam Chuyển        |
|  | Tỉ lệ: 72% | Cần: 80 🧠 - 500 🪙  |  |  Mô tả: Phân tách kiếm khí thành ảo ảnh |
|  | Trạng thái: Chưa có tàn phương    |  +-----------------------------------------+
|  +-----------------------------------+  |  VẬT DẪN NGHIÊN CỨU (CATALYST SLOT)     |
|  +-----------------------------------+  |  [ + Chọn Cổ Phôi / Nguyên Liệu Cùng Hệ]|
|  | [Icon] Kiếm Quang Cổ (R2)         |  |  --> Đã chọn: Kiếm Tinh Thạch (+12% TL) |
|  | Tỉ lệ: 84% | Cần: 50 🧠 - 150 🪙  |  +-----------------------------------------+
|  +-----------------------------------+  |  DỰ TOÁN THIÊN CƠ (FORECAST)            |
|  +-----------------------------------+  |  * Tỉ lệ trúng đích: [===== 84% =====]  |
|  | [Icon] Tru Tiên Kiếm (R6) [ĐỎ]    |  |  * Khả năng lệch nhánh: 16%             |
|  | ⚠️ VƯỢT CẤP - NGUY CƠ PHẢN PHỆ 85% |  |  * Tiêu hao: 80 Niệm Lực + 500 Đá       |
|  +-----------------------------------+  |  * Nguy cơ phản phệ: 0% (An toàn)       |
|                                         |  +-------------------------------------+|
|                                         |  |  🔮 KHỞI NIỆM THÔI DIỄN (BUTTON)   ||
|                                         |  +-------------------------------------+|
+-----------------------------------------+-----------------------------------------+
```

### 2. Khu Vực Dự Toán Thiên Cơ & Cảnh Báo Phản Phệ
- **Thanh Tỉ Lệ Trúng Đích**: Progress bar gradient xanh lam $\implies$ vàng kim, hiển thị chính xác % dựa trên tính toán thời gian thực từ vé 1 và vé 2.
- **Hộp Vật Dẫn**: Bấm vào mở Drawer/Modal mini hiển thị các Cổ trong Không Khiếu và Nguyên Liệu trong Túi đáp ứng tiêu chí cùng hệ.
- **Cảnh Báo Vượt Cấp (Backlash Warning)**:
  - Nếu `targetRank > player.Rank`: Hiện khung cảnh báo viền đỏ rực nhấp nháy:
    `⚠️ CẢNH BÁO: Cố suy tính Cổ vượt quá tu vi hiện tại! Có 60% nguy cơ bị Trí Đạo Phản Phệ (mất 50% HP/Chân Nguyên, hút cạn Niệm Lực)!`

### 3. Modal Thông Báo Kết Quả Sống Động (`ConceiveResultModal`)
Dựa trên `status` trả về từ Backend:
1. **Thành công đúng Cổ đích (`SUCCESS`)**:
   - Viền vàng kim hào quang, âm thanh đốn ngộ.
   - Text: *"Khai sáng thành công! Đạo hữu đã lĩnh ngộ được Tàn Phương [Kiếm Ảnh Cổ] (Độ hoàn thiện: 28.5%)."*
   - Nút hành động: *"Đi đến Thôi Diễn Hoàn Thiện"* (chuyển sang tab thôi diễn thường) hoặc *"Tiếp tục suy tính"*.
2. **Bị lệch hướng sang Cổ liên quan (`DEVIATED`)**:
   - Viền tím huyền bí, tia sáng rẽ nhánh.
   - Text: *"Thiên cơ rẽ nhánh! Cảnh giới kiếm đạo chưa đủ sâu khiến ý niệm chuyển hóa, đạo hữu ngộ ra Tàn Phương Cổ trùng liên quan [Kiếm Quang Cổ] (Độ hoàn thiện: 22.0%)."*
3. **Linh Quang Chớp Lóe (`IS_INSTANT_100`)**:
   - Hiệu ứng pháo hoa linh quang rực rỡ toàn màn hình.
   - Text: *"✨ LINH QUANG CHỚP LÓE! Đạo hữu đốn ngộ huyền cơ vạn cổ, lập tức hoàn thiện 100% Bí Phương [Kiếm Ảnh Cổ]!"*
4. **Trí Đạo Phản Phệ (`BACKLASH`)**:
   - Màn hình rung lắc, vỡ vụn máu đỏ.
   - Text: *"💥 TRÍ ĐẠO PHẢN PHỆ! Cưỡng cầu thiên cơ thất bại khiến đạo hữu thổ huyết chấn thương! Mất 140 HP, Niệm Lực cạn kiệt!"*

### 4. Tích Hợp Nút Chuyển Nhanh Từ Cổ Phổ (Gu Codex)
- Trong danh mục tra cứu Cổ Phổ (`GuCodexModal`), đối với mọi Cổ người chơi chưa sở hữu bí phương 100%:
  - Hiển thị nút bấm: `✨ Suy Tính Bí Phương Này`.
  - Khi bấm: Tự động mở `RefineModal`, chuyển sang Tab "Khai Sáng Bí Phương" và kích hoạt chọn sẵn con Cổ đó trong bảng Đàn Suy Diễn.

### 5. Kiến Trúc State Management (Zustand Slice)
Bổ sung vào `frontend/src/store/slices/cultivationSlice.ts`:
- `fetchConceiveCandidates(params)`: Gọi `GET /api/refine/conceive/candidates`.
- `conceiveRecipe(req)`: Gọi `POST /api/refine/conceive`, tự động cập nhật lại snapshot người chơi và danh sách bí phương trong store khi thành công.

