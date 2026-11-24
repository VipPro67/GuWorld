# Ticket 05: Đặc tả Kiến trúc Backend & Database cho Lưu Vết Chấp Thuận và Quyền Xóa Dữ Liệu

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Cần thiết kế cấu trúc cơ sở dữ liệu và API Backend như thế nào để ghi nhận chứng cứ chấp thuận (Consent Audit Log) theo Nghị định 13/2023/NĐ-CP và hỗ trợ quyền yêu cầu xóa/khóa tài khoản (Right to be forgotten) mà vẫn đảm bảo an toàn toàn vẹn dữ liệu (Data Integrity) trong GuWorld?

Cụ thể cần đặc tả:
1. **Lưu vết Chấp thuận khi Đăng ký (Register Consent Audit)**:
   - Mở rộng bảng `players` hoặc bảng liên kết (`user_consents`) với các trường:
     - `accepted_terms_version`: Phiên bản điều khoản người dùng đã đồng ý (vd: `"2024.1"`).
     - `accepted_privacy_version`: Phiên bản chính sách bảo mật đã đồng ý.
     - `terms_accepted_at`: Dấu thời gian (Timestamp) đồng ý.
     - `terms_accepted_ip`: Địa chỉ IP tại thời điểm gửi form đăng ký.
   - Cập nhật handler `RegisterPlayer` trong `backend/internal/handlers/player_auth_register.go` để xác thực `agree_terms: true` từ request body; nếu false hoặc thiếu thì từ chối HTTP 400.
2. **Quyền Rút lại Sự đồng ý & Yêu cầu Xóa Tài khoản (Right to be Forgotten - NĐ 13/2023)**:
   - Thiết kế luồng "Yêu cầu Xóa Tài khoản / Hủy Bỏ Thông Tin":
     - Người chơi có thể kích hoạt trong mục Hồ sơ (Profile / Settings) bằng cách nhập lại mật khẩu xác nhận.
     - Xử lý kỹ thuật (Soft-delete vs Hard-delete & Anonymization):
       - Chuyển trạng thái tài khoản thành `is_deleted = true`, ẩn khỏi bảng xếp hạng, ngắt kết nối phiên đăng nhập ngay lập tức.
       - Làm mờ/vô danh hóa dữ liệu cá nhân: Thay thế email thành `deleted_<uuid>@guworld.local`, xóa IP lịch sử, giữ lại các bản ghi giao dịch thanh toán PayOS ở dạng vô danh (anonymized) trong 1 năm để phục vụ đối soát kế toán và pháp lý theo luật lưu trữ.
3. **Quản lý Phiên bản Điều khoản (Terms Versioning)**:
   - Cơ chế phát hiện khi ban quản trị cập nhật Điều khoản lên phiên bản mới (Major version change): Nếu `accepted_terms_version` của người chơi cũ hơn phiên bản hiện tại, client sẽ hiển thị popup thông báo cập nhật điều khoản và yêu cầu xác nhận trước khi tiếp tục vào game.

## Answer

Đã hoàn thiện toàn diện đặc tả kiến trúc Backend & Cơ sở dữ liệu PostgreSQL cho cơ chế Lưu vết Chấp thuận (Consent Audit) và Thực thi Quyền Xóa Dữ Liệu Cá Nhân (Right to be Forgotten) theo chuẩn **Nghị định 13/2023/NĐ-CP**:

1. **Mở Rộng Bảng `players` & Compatibility DDL (`schema_players.go`)**:
   - Thêm các cột lưu vết pháp lý:
     - `accepted_terms_version VARCHAR(20) NOT NULL DEFAULT ''`
     - `accepted_privacy_version VARCHAR(20) NOT NULL DEFAULT ''`
     - `terms_accepted_at TIMESTAMPTZ DEFAULT NULL`
     - `terms_accepted_ip VARCHAR(50) NOT NULL DEFAULT ''`
     - `is_deleted BOOLEAN NOT NULL DEFAULT false`
     - `deleted_at TIMESTAMPTZ DEFAULT NULL`
     - `deletion_reason TEXT NOT NULL DEFAULT ''`
   - Khối compatibility DDL sử dụng `ALTER TABLE players ADD COLUMN IF NOT EXISTS ...` đảm bảo tính lũy nghiệm (idempotent), an toàn cho cả database mới khởi tạo lẫn dữ liệu đang chạy.

2. **Cập Nhật Luồng Đăng Ký (`RegisterPlayer` trong `player_auth_register.go`)**:
   - Thêm trường `AgreeTerms bool` vào struct `CreatePlayerRequest`.
   - Nếu `!req.AgreeTerms`: Ngay lập tức từ chối với HTTP 400 và thông báo yêu cầu chấp thuận điều khoản (tuân thủ nguyên tắc không chấp thuận ngầm định).
   - Khi tạo tài khoản: Tự động ghi `accepted_terms_version = "2024.1"`, `accepted_privacy_version = "2024.1"`, `terms_accepted_at = NOW()`, `terms_accepted_ip = c.ClientIP()`.

3. **Thiết Kế API Vô Danh Hóa Tài Khoản (`POST /api/player/request-deletion`)**:
   - Xác thực danh tính chính chủ bằng mật khẩu hiện tại (Bcrypt verification).
   - Áp dụng phương thức **Vô danh hóa dữ liệu (Data Anonymization)**:
     - Gán `email = 'deleted_' || id || '@guworld.local'`.
     - Xóa rỗng `password_hash`, đưa `register_ip` và `last_login_ip` về `"0.0.0.0"`.
     - Đặt cờ `is_deleted = true`, cập nhật `deleted_at = NOW()`.
     - Thu hồi JWT Token và ngắt kết nối phiên realtime / SSE.
     - Bảo toàn các bản ghi thanh toán PayOS (`topup_orders`) dưới dạng liên kết ẩn danh phục vụ nghĩa vụ lưu trữ kế toán tài chính theo luật định mà không để lộ email người dùng.

4. **Quản Lý Phiên Bản Điều Khoản (Major Versioning & Re-Consent)**:
   - Khi BQT phát hành bản điều khoản mới (vd: `"2025.1"`), hệ thống kiểm tra lúc đăng nhập; nếu phiên bản cũ hơn sẽ trả cờ `requires_terms_reconsent = true`.
   - Cung cấp endpoint `POST /api/player/reconsent-terms` để người dùng xác nhận điều khoản mới mà không gián đoạn tiến trình tu luyện.

5. **Tài Sản Đặc Tả & Giao Diện Kiểm Chứng Trực Quan**:
   - Văn bản đặc tả schema & API chi tiết (Markdown): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/05_backend_schema_and_api_spec.md`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/05_backend_schema_and_api_spec.md)
   - Giao diện HTML xem thử tương tác (Mô phỏng Request Đăng Ký từ chối khi thiếu consent, Endpoint xóa tài khoản và Bảng theo dõi biến đổi DB theo thời gian thực): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/05_backend_consent_and_erasure_demo.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/05_backend_consent_and_erasure_demo.html)
