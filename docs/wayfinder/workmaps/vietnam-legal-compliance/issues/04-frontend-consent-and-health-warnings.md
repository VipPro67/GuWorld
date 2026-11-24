# Ticket 04: Đặc tả Kiến trúc UI/UX Tích Hợp Điều Khoản, Checkbox Consent & Cảnh Báo Sức Khỏe

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để tích hợp trải nghiệm người dùng (UI/UX) cho toàn bộ hệ thống Điều khoản, Cơ chế Chấp thuận (Consent) và Cảnh báo Pháp định theo Nghị định 147/2024/NĐ-CP trên giao diện Web của GuWorld mà vẫn giữ được sự tinh gọn, thẩm mỹ và không gây cản trở trải nghiệm chơi game?

Cụ thể cần thiết kế và định hình:
1. **Form Đăng ký (Register Modal/Page)**:
   - Thêm checkbox bắt buộc: "Tôi đã đủ 18 tuổi và đồng ý với [Điều khoản Dịch vụ] và [Chính sách Bảo vệ Dữ liệu Cá nhân]".
   - Checkbox mặc định để trống (Unchecked - Opt-in theo đúng chuẩn NĐ 13/2023). Nút "Đăng Ký" chỉ kích hoạt (enabled) khi người chơi đã tích chọn.
   - Cho phép bấm vào chữ [Điều khoản Dịch vụ] và [Chính sách Bảo vệ Dữ liệu Cá nhân] để mở Modal đọc nhanh văn bản đầy đủ mà không bị mất dữ liệu form đang nhập.
2. **Thành phần Điều hướng & Chân trang (Footer / Settings Links)**:
   - Bổ sung thanh chân trang tối giản (Minimal Legal Footer) ở màn hình Đăng nhập / Chọn nhân vật và mục Cài đặt trong game, gồm các link: "Điều khoản dịch vụ", "Chính sách bảo mật", "Quy chế nạp tiền & Miễn trừ".
   - Tạo trang/modal hiển thị nội dung Markdown văn bản pháp lý chuyên dụng có định dạng đẹp mắt, dễ đọc trên cả PC lẫn Mobile.
3. **Dòng Cảnh báo Pháp định 180 phút (Nghị định 147/2024/NĐ-CP)**:
   - Hiển thị dòng chữ cảnh báo tiêu chuẩn: *"Trò chơi điện tử dành cho người từ 18 tuổi trở lên (18+). Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe."* ở vị trí trang trọng tại chân trang màn hình chính / đăng nhập.
4. **Hệ thống Nhắc nhở Thời gian Chơi (Playtime Alert Popup)**:
   - Bổ sung logic tính thời gian phiên chơi (Session Timer) trên client (hoặc đồng bộ nhẹ qua state).
   - Khi người chơi liên tục trong 60 phút, 120 phút và 180 phút, hiển thị toast hoặc modal nhỏ nhắc nhở nhẹ nhàng: *"Đạo hữu đã tu luyện liên tục [X] phút. Hãy nghỉ ngơi, điều hòa khí tức để bảo vệ sức khỏe nhục thân!"*.

## Answer

Đã hoàn thiện toàn diện đặc tả kiến trúc UI/UX và tài sản nguyên mẫu (Prototype) tích hợp hệ thống Điều khoản, Cơ chế Chấp thuận (Opt-in Consent) và Cảnh báo Sức khỏe định kỳ theo chuẩn pháp luật Việt Nam:

1. **Chuẩn Hóa Form Đăng Ký (`Registration.tsx`)**:
   - Bổ sung `agreeTerms: boolean` (mặc định `false`). Nút "Thức Tỉnh Không Khiếu" chỉ kích hoạt khi người chơi đã tự tay tích chọn ô chấp thuận.
   - Nhãn Checkbox tích hợp nút mở trực tiếp Modal đọc nhanh văn bản mà không làm mất dữ liệu người dùng đang nhập dở dang trong các ô input.
   - Truyền trường `agree_terms: true` vào payload gọi sang backend `onRegister(...)`.

2. **Kiến Trúc Modal Đọc Văn Bản Pháp Lý (`LegalModal.tsx`)**:
   - Modal chuyên dụng chuẩn hóa theme Tiên hiệp tối (`dark slate`), hỗ trợ thanh chuyển Tab mượt mà giữa 3 văn bản:
     1. Điều Khoản Dịch Vụ (ToS);
     2. Chính Sách Bảo Mật Dữ Liệu (Privacy Policy NĐ 13/2023);
     3. Quy Chế Nạp Tiền PayOS & Bản Quyền Cổ Chân Nhân.
   - Nút "Ta Đã Hiểu & Đóng" ở chân modal.

3. **Chân Trang Pháp Lý Cố Định (`LegalFooter.tsx`)**:
   - Đặt cố định ở chân trang màn hình Đăng nhập / Chọn nhân vật / Lobby chính.
   - Tích hợp cố định dòng cảnh báo pháp định Điều 39 & 40 Nghị định 147/2024/NĐ-CP:
     *Badge `18+` kèm dòng chữ: "Cảnh Báo Pháp Định: Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe."*
   - Cung cấp 3 đường dẫn nhanh mở `LegalModal`.

4. **Hệ Thống Theo Dõi Phiên Chơi & Cảnh Báo Sức Khỏe (`usePlaytimeWarning.ts`)**:
   - Hook client tự động tính thời lượng phiên chơi liên tục (`sessionSeconds`), lưu mốc vào `sessionStorage` để không bị mất khi reload tab.
   - **Các mốc thông báo**:
     - *Mốc 60 phút*: Bắn Toast thông tin nhẹ nhàng nhắc nhở uống nước, chớp mắt nghỉ ngơi.
     - *Mốc 120 phút*: Bắn Toast cảnh báo nhắc nhở vận chuyển chân nguyên quá lâu tổn hao tinh thần.
     - *Mốc 180 phút & mỗi 60 phút sau đó*: Bật Modal Pop-up Cảnh báo Sức khỏe Pháp định (`PlaytimeWarningModal`) yêu cầu người chơi xác nhận đã hiểu khuyến cáo y tế.

5. **Tài Sản Đặc Tả & Giao Diện Kiểm Chứng Trực Quan**:
   - Văn bản đặc tả kiến trúc chi tiết (Markdown): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/04_frontend_legal_components_spec.md`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/04_frontend_legal_components_spec.md)
   - Giao diện HTML xem thử tương tác (với Form Đăng Ký, Checkbox Consent, Modal đọc ToS và Trình giả lập tua nhanh thời gian chơi +60m / +120m / +180m): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/04_frontend_ui_flow_preview.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/04_frontend_ui_flow_preview.html)
