# Ticket 03: Hệ Thống Trợ Trận Độ Kiếp và Thiên Đạo Thích Ứng Biến Dị

Type: prototype
Status: resolved
Assignee: @antigravity
Blocked by: [02-aperture-instance-and-guest-matrix](02-aperture-instance-and-guest-matrix.md)

## Question

Làm thế nào để nâng cấp hệ thống tai kiếp thời gian thực (`tribulation_realtime.go`) hỗ trợ tổ đội trợ trận, áp dụng quy luật Thiên Đạo trừng phạt lũy tiến và phân phối lợi ích sau kiếp nạn?

Cụ thể cần giải quyết các vấn đề:
1. **Quy Trình Khởi Động Trợ Trận Độ Kiếp (Co-op Tribulation Launch Flow)**:
   - Khi Chủ Khiếu bấm bắt đầu Độ Kiếp tại NPC/Menu ([`RealtimeEngageTribulation`](backend/internal/handlers/tribulation_realtime.go)), server kiểm tra toàn bộ người chơi đang hiện diện trong Tiên Khiếu.
   - Các trợ thủ nhận thông báo đếm ngược và tự động gia nhập đội hình phòng ngự bảo vệ Tiên Khiếu.
2. **Thuật Toán Thiên Đạo Thích Ứng & Biến Dị (Reactive Mutation Engine)**:
   - Cơ chế tăng chỉ số tai kiếp: Quái kiếp nạn (Quái thường và Boss kiếp nạn) được nhân hệ số HP, Công, Thủ theo công thức:
     $$\text{DifficultyMultiplier} = 1.0 + \sum_{h \in \text{Helpers}} 0.5 \times (\text{Rank}_h - \text{Rank}_{\text{aperture}} + 1)$$
   - Đợt Biến Dị Hỗn Loạn (*Chaos Calamity Waves*): Thiên Đạo phát hiện lưu phái chính của người trợ thủ (ví dụ: Viêm Đạo, Băng Tuyết Đạo, Lôi Đạo) và sinh ra các quái kiếp mang Đạo Ngân tương khắc trực tiếp với lưu phái đó để trừng phạt việc can thiệp kiếp nạn.
3. **Phân Bổ Chiến Lợi Phẩm & Đạo Ngân (Reward Distribution Matrix)**:
   - Đạo Ngân thăng hoa: 100% Đạo Ngân vượt kiếp và sự mở rộng diện tích Tiên Khiếu thuộc về Chủ Khiếu (người trợ thủ không thể lấy trộm Đạo Ngân Thiên Địa).
   - Phần thưởng trợ thủ: Quái và Boss kiếp nạn rơi rớt Tiên Tài quý hiếm, phôi Cổ hoang dã; hệ thống cho phép thiết lập Khế Ước Thù Lao tự động (trích Tiên Thạch từ kho Chủ Khiếu chuyển cho Trợ Thủ khi vượt kiếp thành công).
4. **Cơ Chế Suy Thoái & Phòng Hộ Tài Nguyên (Resource Protection & Destruction)**:
   - Quái tai kiếp ưu tiên tấn công các điểm tài nguyên (`aperture_resource_points`) và công trình phòng thủ.
   - Cơ chế chia sẻ trách nhiệm: Trợ thủ có thể hy sinh thanh máu/tiên nguyên để che chắn cho điểm tài nguyên hoặc bản thể Chủ Khiếu.

## Answer

Đã chuẩn hóa toàn diện quy trình Trợ Trận Độ Kiếp, thuật toán Thiên Đạo thích ứng biến dị và ma trận phân bổ chiến lợi phẩm. Kiểm chứng hoạt động hoàn tất qua prototype tương tác tại [03-coop-tribulation-mutation.html](../prototypes/03-coop-tribulation-mutation.html).

### 1. Quy Trình Khởi Động Trợ Trận & Bảng Danh Sách (Launch Handshake)
- **Cơ sở dữ liệu**: Bổ sung bảng `tribulation_coop_participants` trong `backend/internal/db/schema_aperture.go`:
  ```sql
  CREATE TABLE IF NOT EXISTS tribulation_coop_participants (
      event_id UUID NOT NULL REFERENCES tribulation_events(id) ON DELETE CASCADE,
      player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
      rank INT NOT NULL,
      primary_path VARCHAR(50) NOT NULL,
      agreed_bounty_immortal_stones INT NOT NULL DEFAULT 0,
      is_alive BOOLEAN NOT NULL DEFAULT TRUE,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
      PRIMARY KEY (event_id, player_id)
  );
  ```
- **Luồng Handshake**:
  - Khi Chủ Khiếu bấm `RealtimeEngageTribulation` ([`tribulation_realtime.go`](backend/internal/handlers/tribulation_realtime.go)), server quét `aperture_guest_roster` lấy danh sách khách đang ở trong khiếu.
  - Khách nhận thông báo xác nhận tham chiến qua SSE:
    * Chấp thuận: Đăng ký vào `tribulation_coop_participants` với mức thù lao cam kết.
    * Từ chối: Lập tức được luân chuyển ra ngoài Môn Hộ thế giới để tránh bị sấm sét kiếp nạn đánh trúng.

### 2. Thuật Toán Thiên Đạo Thích Ứng & Biến Dị Tương Khắc (Reactive Mutation Engine)
- **Hệ số tăng tiến theo tổng thực lực tổ đội**:
  Trong `tribulation_trigger.go`, các chỉ số HP và Công của quái kiếp nạn được nhân theo công thức:
  $$\text{DifficultyMultiplier} = 1.0 + \sum_{h \in \text{Helpers}} \left( 0.4 + 0.3 \times \max(0, \text{Rank}_h - \text{Rank}_{\text{aperture}} + 1) \right)$$
  - Ví dụ:
    - Solo: $1.0\times$.
    - 1 Trợ Thủ Rank 6: $1.7\times$ HP và $+40\%$ Công.
    - 1 Trợ Thủ Rank 7 giúp Phúc Địa Rank 6: $2.0\times$ HP và $+60\%$ Công.
- **Đợt Kiếp Nạn Biến Dị Hỗn Loạn (*Chaos Nemesis Wave*)**:
  - Thiên Đạo phân tích lưu phái chủ đạo của trợ thủ có thực lực cao nhất (dựa trên `path_masteries` và Cổ trang bị).
  - Tự động triệu hồi thêm 1 Boss Phụ Biến Dị (`is_boss = true, is_mutation = true`) mang Đạo Ngân tương khắc:
    * Trợ thủ Viêm Đạo $\rightarrow$ Sinh ra Cực Hàn Băng Long (Băng Tuyết Đạo, kháng $80\%$ Hỏa sát thương).
    * Trợ thủ Lực Đạo $\rightarrow$ Sinh ra Huyễn Ảnh Ma Thú (Huyễn Đạo, miễn nhiễm sát thương vật lý).
    * Trợ thủ Lôi Đạo $\rightarrow$ Sinh ra Tịch Lôi Nham Thú (Thổ Đạo, hấp thu lôi điện).
  - Cơ chế này buộc Chủ Khiếu và Trợ Thủ phải phối hợp chiêu thức chéo cánh để triệt hạ mục tiêu, xóa bỏ hoàn toàn tình trạng kéo rank đơn giản.

### 3. Phân Bổ Chiến Lợi Phẩm & Khế Ước Thù Lao (Reward Distribution Matrix)
- Tại `RealtimeResolveTribulationVictory` ([`tribulation_realtime.go`](backend/internal/handlers/tribulation_realtime.go)):
  - **Chủ Khiếu**:
    * Tiếp nhận trọn vẹn $100\%$ Đạo Ngân thưởng (`reward_dao_marks`) giáng hạ vào Tiên Khiếu.
    * Mở rộng diện tích đất (`area_size`) và cộng dồn số lần vượt kiếp để chuẩn bị đột phá cảnh giới mới.
  - **Trợ Thủ**:
    * **Chiến lợi phẩm rớt tại chỗ**: Boss biến dị rớt Tiên Tài hiếm bậc cao và phôi Cổ hoang dã.
    * **Khế Ước Thù Lao Tự Động**: Hệ thống trích trực tiếp số Tiên Nguyên Thạch đã cam kết (`agreed_bounty_immortal_stones`) từ kho Chủ Khiếu chuyển vào túi đồ của Trợ Thủ trong cùng transaction DB nguyên tử `sql.Tx`.

### 4. Cơ Chế Bảo Vệ Điểm Tài Nguyên & AI Chia Tải
- AI quái kiếp nạn phân chia mục tiêu: $60\%$ nhắm vào Cổ Tiên, $40\%$ tấn công các Điểm Tài Nguyên (`aperture_resource_points`) và Tháp Canh.
- Khi trận chiến kết thúc thắng lợi, Đạo Ngân thăng hoa tự động chữa lành và hồi phục $100\%$ sinh khí (`vitality = 100`) cho toàn bộ các điểm tài nguyên chưa bị phá hủy hoàn toàn.


## Question

Làm thế nào để nâng cấp hệ thống tai kiếp thời gian thực (`tribulation_realtime.go`) hỗ trợ tổ đội trợ trận, áp dụng quy luật Thiên Đạo trừng phạt lũy tiến và phân phối lợi ích sau kiếp nạn?

Cụ thể cần giải quyết các vấn đề:
1. **Quy Trình Khởi Động Trợ Trận Độ Kiếp (Co-op Tribulation Launch Flow)**:
   - Khi Chủ Khiếu bấm bắt đầu Độ Kiếp tại NPC/Menu ([`RealtimeEngageTribulation`](backend/internal/handlers/tribulation_realtime.go)), server kiểm tra toàn bộ người chơi đang hiện diện trong Tiên Khiếu.
   - Các trợ thủ nhận thông báo đếm ngược và tự động gia nhập đội hình phòng ngự bảo vệ Tiên Khiếu.
2. **Thuật Toán Thiên Đạo Thích Ứng & Biến Dị (Reactive Mutation Engine)**:
   - Cơ chế tăng chỉ số tai kiếp: Quái kiếp nạn (Quái thường và Boss kiếp nạn) được nhân hệ số HP, Công, Thủ theo công thức:
     $$\text{DifficultyMultiplier} = 1.0 + \sum_{h \in \text{Helpers}} 0.5 \times (\text{Rank}_h - \text{Rank}_{\text{aperture}} + 1)$$
   - Đợt Biến Dị Hỗn Loạn (*Chaos Calamity Waves*): Thiên Đạo phát hiện lưu phái chính của người trợ thủ (ví dụ: Viêm Đạo, Băng Tuyết Đạo, Lôi Đạo) và sinh ra các quái kiếp mang Đạo Ngân tương khắc trực tiếp với lưu phái đó để trừng phạt việc can thiệp kiếp nạn.
3. **Phân Bổ Chiến Lợi Phẩm & Đạo Ngân (Reward Distribution Matrix)**:
   - Đạo Ngân thăng hoa: 100% Đạo Ngân vượt kiếp và sự mở rộng diện tích Tiên Khiếu thuộc về Chủ Khiếu (người trợ thủ không thể lấy trộm Đạo Ngân Thiên Địa).
   - Phần thưởng trợ thủ: Quái và Boss kiếp nạn rơi rớt Tiên Tài quý hiếm, phôi Cổ hoang dã; hệ thống cho phép thiết lập Khế Ước Thù Lao tự động (trích Tiên Thạch từ kho Chủ Khiếu chuyển cho Trợ Thủ khi vượt kiếp thành công).
4. **Cơ Chế Suy Thoái & Phòng Hộ Tài Nguyên (Resource Protection & Destruction)**:
   - Quái tai kiếp ưu tiên tấn công các điểm tài nguyên (`aperture_resource_points`) và công trình phòng thủ.
   - Cơ chế chia sẻ trách nhiệm: Trợ thủ có thể hy sinh thanh máu/tiên nguyên để che chắn cho điểm tài nguyên hoặc bản thể Chủ Khiếu.
