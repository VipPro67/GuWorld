# Ticket 05: Thiết Kế Bộ Chuyển Tầng Bản Đồ (Layer Switcher) & Hiệu Ứng Khí Quyển Frontend

Type: prototype
Status: resolved
Blocked by: 01-vertical-world-schema-and-dia-uyen-layer, 03-canon-regions-and-biomes-specification
Assignee: Antigravity

## Question

Làm thế nào để nâng cấp giao diện bản đồ (`WorldAtlasModal.tsx`, `ExplorationMap.tsx`) để người chơi có thể dễ dàng chuyển đổi góc nhìn giữa 4 tầng thế giới (Tầng 0: Địa Uyên, Tầng 1: Ngũ Vực, Tầng 2: Bạch Thiên, Tầng 3: Hắc Thiên) và cảm nhận rõ rệt sự biến đổi của môi trường khí quyển?

Cụ thể cần giải quyết:
1. **Bộ Chuyển Tầng Không Gian (Layer Switcher HUD)**:
   - Thêm widget chuyển tầng dạng cột đứng bên cạnh World Atlas hoặc thanh điều hướng trên cùng:
     - `Tầng 3: Thái Cổ Hắc Thiên` (Icon trăng khuyết / tinh vân, màu tím đen / indigo, yêu cầu Rank 8+).
     - `Tầng 2: Thái Cổ Bạch Thiên` (Icon mặt trời / mây trắng, màu vàng kim / thiên thanh, chỉ Cổ Tiên Rank 6+).
     - `Tầng 1: Ngũ Vực Đại Lục` (Icon núi sông / lục địa, màu xanh ngọc / đất vàng).
     - `Tầng 0: Vạn Trượng Địa Uyên` (Icon vực đá nứt / dung nham, màu đỏ thẫm / nâu đá, yêu cầu Rank 5+ hoặc Cổ Ốc).
   - Tự động highlight tầng mà nhân vật đang thực tế đứng.
2. **Hiệu Ứng Môi Trường & Khí Quyển (Atmospheric Overlays)**:
   - **Tầng 0 (Địa Uyên)**: Viền màn hình rung chuyển nhẹ, ánh sáng tù mù u tối, hiệu ứng bụi đá và dung nham ngầm bập bùng.
   - **Tầng 2 (Bạch Thiên)**: Ánh sáng thiên quang chói lọi, các dải mây lướt ngang màn hình (parallax cloud layers), tầm nhìn mở rộng (+1 bonus vision).
   - **Tầng 3 (Hắc Thiên)**: Bầu trời sao vô tận, sương mù đen bao phủ dày đặc, giảm bán kính nhìn xung quanh nhân vật nếu không đủ điểm trinh sát.
3. **Đồng Bộ Dữ Liệu Tầng Qua Header & Store & Nút Skyfall**:
   - `useWorldStore.ts` đã hỗ trợ header `X-World-Revision-Map`; tích hợp nút hành động **Rơi Tự Do (Skyfall)** ngay trên `ExplorationMap.tsx` khi người chơi đang đứng trên Bạch Thiên hoặc Hắc Thiên để hạ cánh xuống tầng bên dưới.
   - Siết chặt quy tắc tu vi theo yêu cầu nguyên tác: **Chỉ Cổ Tiên (Rank 6+) mới có thể lên Bạch Thiên; Chỉ Bát Chuyển Cổ Tiên (Rank 8+) mới có thể tiến vào Hắc Thiên**.

## Answer

Đã hoàn thành toàn diện giao diện bộ chuyển tầng bản đồ và hiệu ứng thị giác khí quyển:

### 1. Bộ Chuyển Tầng Không Gian (Layer Switcher HUD) trên `WorldAtlasModal.tsx`
- Tích hợp thanh điều hướng 4 tầng đứng đầu modal với icon biểu tượng canon:
  - **Tầng 3: Thái Cổ Hắc Thiên** (`hac_thien`): Biểu tượng Mặt Trăng (`Moon`), theme tinh vân tím đen/indigo. Hiển thị cảnh báo: *Chỉ Bát Chuyển Cổ Tiên (Rank 8+) mới có đủ uy áp và thể phách để bước vào!*
  - **Tầng 2: Thái Cổ Bạch Thiên** (`bai_thien`): Biểu tượng Mặt Trời (`Sun`), theme hoàng kim mây trắng/thiên thanh. Hiển thị cảnh báo: *Khí Cương Phong Bích cản đường — Chỉ Cổ Tiên (Rank 6+) đúc thành Tiên Thể mới có thể phi thăng!*
  - **Tầng 1: Ngũ Vực Đại Lục** (`world_map`): Biểu tượng Đại Sơn (`Mountain`), theme ngọc bích lục địa 5 đại vực.
  - **Tầng 0: Vạn Trượng Địa Uyên** (`dia_uyen`): Biểu tượng Liệt Hỏa (`Flame`), theme đỏ thẫm đá ngầm dung nham. Hiển thị cảnh báo: *Chướng Khí Địa Câu kịch độc — Cần tu vi Ngũ Chuyển (Rank 5+) hoặc Tiên Cổ Ốc hộ thân!*
- Tự động phát hiện tầng nhân vật đang đứng (`playerCurrentLayer`) và gắn badge nhấp nháy `VỊ TRÍ` / `ĐANG Ở ĐÂY`.
- Khi chọn Tầng 0, 2 hoặc 3: SVG map render 3 đại phân khu tương ứng kết nối bằng dòng năng lượng đặc trưng (magma ngầm ở Địa Uyên, cầu mây ở Bạch Thiên, vết nứt sao ở Hắc Thiên). Nhấp vào bất kỳ khu vực nào sẽ mở bảng chi tiết Yêu Thú, Boss, Rơi Đồ, Cổ Hoang Dã và Khoáng Sản ở cột phải.

### 2. Hiệu Ứng Môi Trường & Khí Quyển (Atmospheric Overlays) trên `ExplorationMap.tsx`
- Bổ sung các lớp phủ CSS atmosphere động và phong cảnh nền theo tầng thế giới:
  - **Tầng 0 (Địa Uyên)**: Nền hang động nham thạch tối tăm (`#160b0b`), viền đỏ thẫm u tối, luồng nhiệt dung nham bập bùng (`animate-pulse`).
  - **Tầng 2 (Bạch Thiên)**: Nền thiên thanh sáng lạn (`#0d2035`), thiên quang vàng kim rực rỡ, dải mây trắng lướt ngang vô tận. Bonus +1 tầm nhìn lưới khám phá trên backend `exploration_grid.go`.
  - **Tầng 3 (Hắc Thiên)**: Nền không gian tối tăm vạn trượng (`#05050f`), bụi sao tinh tú nhấp nháy trên nền đêm sâu thẳm. Hạn chế tầm nhìn nếu điểm trinh sát chưa đủ cao.
  - **Tầng 1 (Ngũ Vực)**: Bản đồ lục địa truyền thống.
- Hiển thị badge tầng không gian trực tiếp cạnh tọa độ nhân vật trên thanh tiêu đề `ExplorationMap.tsx`.

### 3. Nút Hành Động Rơi Tự Do (Skyfall)
- Khi nhân vật đang ở Bạch Thiên hoặc Hắc Thiên, trên thanh công cụ `ExplorationMap.tsx` xuất hiện nút **"Rơi Tự Do (Skyfall)"** nổi bật.
- Bấm nút sẽ hiển thị hộp thoại xác nhận hạ tầng và gọi `POST /api/world/skyfall` qua `worldSlice.skyfall()`, tự động xuyên qua tầng giới hạ cánh xuống tầng bên dưới và reload toàn bộ grid.

### 4. Siết Chặt Giới Hạn Tu Vi Theo Canon Truyện
- Trong `backend/internal/handlers/movement_layers.go` (`checkLayerTransitionGuard`):
  - **Bạch Thiên (`bai_thien`)**: Yêu cầu nghiêm ngặt `player.Rank >= 6` (Cổ Tiên). Phàm nhân bị Khí Cương Phong Bích xé rách nhận 90% sát thương và bị hất văng trở lại.
  - **Hắc Thiên (`hac_thien`)**: Yêu cầu nghiêm ngặt `player.Rank >= 8` (Bát Chuyển Cổ Tiên). Tu vi dưới Bát Chuyển bị Cực Điểm Không Gian từ chối và chịu sát thương hư không.
