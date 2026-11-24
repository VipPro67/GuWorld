# Ticket 02: Soạn thảo Chính sách Bảo vệ Dữ liệu Cá nhân (Personal Data Protection Policy)

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Cần xây dựng một văn bản **Chính sách Bảo vệ Dữ liệu Cá nhân (Privacy Policy / Thông báo Xử lý Dữ liệu)** hoàn chỉnh theo đúng các tiêu chuẩn bắt buộc của **Nghị định 13/2023/NĐ-CP** (PDPD) của Chính phủ Việt Nam?

Cụ thể cần giải quyết và chuẩn hóa:
1. **Danh mục Dữ liệu Cá nhân Thu thập**:
   - Xác định danh mục dữ liệu tối thiểu GuWorld đang thực tế thu thập: Email, Tên tài khoản / Đạo hiệu, Mật khẩu (lưu dưới dạng mã băm một chiều an toàn bcrypt), Địa chỉ IP, Lịch sử đăng nhập và nhật ký giao dịch nạp tiền qua mã đơn hàng PayOS.
   - Khẳng định GuWorld **không thu thập** dữ liệu cá nhân nhạy cảm (số CCCD, tài khoản ngân hàng, vị trí GPS chính xác, sinh trắc học).
2. **Mục đích Xử lý Dữ liệu**:
   - Nêu rõ dữ liệu chỉ dùng để: Định danh và duy trì phiên đăng nhập của người chơi, bảo vệ an ninh tài khoản, hỗ trợ khôi phục mật khẩu, đối soát kỹ thuật với cổng thanh toán PayOS và giải quyết khiếu nại kỹ thuật.
3. **Chia sẻ Dữ liệu với Bên Thứ Ba**:
   - Nêu rõ chỉ truyền thông tin giao dịch cần thiết (Mã giao dịch, Số tiền VND) sang cổng thanh toán đối tác (PayOS) để tạo mã QR nạp tiền; không chia sẻ email hoặc thông tin cá nhân khác cho mục đích thương mại/quảng cáo.
4. **Thời hạn Lưu trữ & Biện pháp Bảo vệ Kỹ thuật**:
   - Lưu trữ dữ liệu trong suốt thời gian tài khoản hoạt động.
   - Các biện pháp kỹ thuật: Mã hóa đường truyền HTTPS/TLS, băm mật khẩu bằng Bcrypt có Salt, phân quyền truy cập quản trị chặt chẽ (RBAC).
5. **Quyền của Chủ thể Dữ liệu (Người chơi) theo NĐ 13/2023**:
   - Nêu rõ quyền được biết, quyền đồng ý, quyền truy cập/chỉnh sửa, quyền rút lại sự đồng ý, và đặc biệt là **quyền yêu cầu xóa dữ liệu cá nhân (Right to be forgotten)**.
   - Quy định kênh tiếp nhận yêu cầu (Email quản trị viên hoặc nút chức năng trong giao diện người dùng).

## Answer

Đã hoàn thiện toàn diện văn bản **Chính sách Bảo vệ Dữ liệu Cá nhân & Thông Báo Xử Lý Dữ Liệu (Phiên bản 2024.1)** với 11 điều khoản chi tiết, tuân thủ nghiêm ngặt **Nghị định 13/2023/NĐ-CP (PDPD)**:

1. **Phân Loại Chuẩn Hóa Dữ Liệu Thu Thập (Điều 2)**:
   - **Dữ liệu cá nhân cơ bản**: Email, Tên tài khoản (Đạo hiệu), Mật khẩu (mã hóa băm một chiều Bcrypt có Salt), Địa chỉ IP, User-Agent, Mã giao dịch nạp PayOS.
   - **Tuyên bố tuyệt đối không thu thập Dữ liệu nhạy cảm**: Không thu thập số CCCD, tài khoản ngân hàng, thẻ tín dụng, vị trí định vị GPS vệ tinh hay dữ liệu sinh trắc học.

2. **Mục Đích & Nguyên Tắc Chấp Thuận Rõ Ràng (Điều 3 & 4)**:
   - Xử lý dữ liệu thuần túy phục vụ: Xác thực đăng nhập, duy trì tiến trình game, bảo vệ an ninh mạng, hỗ trợ khôi phục tài khoản và đối soát nạp Ngọc.
   - Áp dụng nguyên tắc **Opt-in Consent (Điều 9 & 11 NĐ 13/2023)**: Checkbox trên form Đăng ký là độc lập, không tích chọn sẵn; người chơi chủ động lựa chọn đồng ý và có quyền rút lại sự đồng ý bất kỳ lúc nào.

3. **Nguyên Tắc Chia Sẻ Hạn Chế với Đối Tác PayOS (Điều 5)**:
   - Tuyệt đối không bán hay cung cấp dữ liệu người chơi cho bên thứ ba vì mục đích quảng cáo/thương mại.
   - Với cổng thanh toán PayOS: Chỉ chuyển thông tin kỹ thuật tối thiểu gồm Mã đơn hàng và Số tiền VND để khởi tạo mã VietQR; không chia sẻ Email hay mật khẩu.

4. **Biện Pháp An Toàn Kỹ Thuật & Thời Hạn Lưu Trữ (Điều 6 & 7)**:
   - Mã hóa toàn trình qua giao thức HTTPS / TLS 1.3; băm mật khẩu Bcrypt đạt chuẩn an toàn cao.
   - Phân quyền nội bộ chặt chẽ (RBAC) và lưu vết thao tác kỹ thuật qua Admin Audit Logs.
   - Lưu trữ nhật ký hệ thống tối thiểu 12 tháng theo Luật An ninh mạng 2018.

5. **Đảm Bảo Trọn Vẹn 11 Quyền của Chủ Thể Dữ Liệu (Điều 8 & 9)**:
   - Nêu rõ 11 quyền theo Điều 9 NĐ 13/2023 (Quyền được biết, đồng ý, truy cập, rút lại sự đồng ý, xóa dữ liệu, hạn chế xử lý, cung cấp dữ liệu, phản đối, khiếu nại, bồi thường, tự bảo vệ).
   - Thiết lập quy trình chuẩn hóa **"Right to be Forgotten"**: Người chơi có thể tự yêu cầu xóa tài khoản/dữ liệu trên giao diện hoặc qua email `privacy@guworld.local`, BQT xử lý trong tối đa 72 giờ làm việc qua quy trình vô danh hóa (Anonymization).

6. **Tài Sản Đặc Tả & Giao Diện Kiểm Chứng Trực Quan**:
   - Văn bản pháp lý hoàn chỉnh (Markdown): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/02_privacy_policy_draft.md`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/02_privacy_policy_draft.md)
   - Giao diện HTML xem thử tương tác (với Ma trận dữ liệu & Trình mô phỏng trích xuất JSON / Vô danh hóa tài khoản): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/02_privacy_policy_preview.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/02_privacy_policy_preview.html)
