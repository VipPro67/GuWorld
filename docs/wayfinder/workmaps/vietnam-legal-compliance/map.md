# Wayfinder Map: Tuân Thủ Pháp Luật Việt Nam & Hệ Thống Điều Khoản Người Dùng (Vietnam Legal & Regulatory Compliance)

## Destination

Đặc tả toàn diện bộ văn bản pháp lý (Điều khoản dịch vụ & Quy chế game theo Nghị định 147/2024/NĐ-CP, Chính sách bảo vệ dữ liệu cá nhân theo Nghị định 13/2023/NĐ-CP, Quy chế nạp tiền PayOS & Tuyên bố phi thương mại/bản quyền tác phẩm) và lộ trình kiến trúc kỹ thuật triển khai lên GuWorld (Checkbox Consent khi đăng ký, Trang/Modal điều khoản, Banner cảnh báo sức khỏe 180 phút, Popup nhắc nhở phiên chơi, cùng giải pháp lưu vết chấp thuận và API hỗ trợ quyền yêu cầu xóa tài khoản/dữ liệu).

## Notes

- Domain: Pháp luật Việt Nam liên quan đến Internet & Game Online (Nghị định 147/2024/NĐ-CP hiệu lực từ 25/12/2024, Nghị định 13/2023/NĐ-CP về bảo vệ dữ liệu cá nhân, Luật An ninh mạng 2018), Hệ thống Auth & Đăng ký (`backend/internal/handlers/player_auth_register.go`, `schema_players.go`), Giao diện người dùng (`frontend/src/app/`, `frontend/src/components/auth/`).
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`.
- Standing preferences:
  - **Bảo vệ tối đa cho dự án**: Định vị rõ ràng GuWorld là dự án nghiên cứu công nghệ / thử nghiệm Alpha phục vụ cộng đồng, không phải dịch vụ phát hành thương mại có cam kết vĩnh viễn; giảm thiểu tối đa rủi ro pháp lý về giấy phép game G1.
  - **Tách biệt ToS và Chính sách Dữ liệu**: Tuân thủ Điều 9 & Điều 11 NĐ 13/2023/NĐ-CP, tách riêng Điều khoản Dịch vụ (ToS) và Chính sách Bảo vệ Dữ liệu Cá nhân (Privacy Policy). Không gộp chung một điều khoản duy nhất.
  - **Nguyên tắc Chấp thuận Rõ ràng (Opt-in Consent)**: Checkbox đồng ý trên form Đăng ký không được tích chọn sẵn; người chơi phải chủ động bấm chọn mới được kích hoạt nút "Đăng Ký".
  - **Khóa chặt tính chất Phi Tiền Tệ của Tài Sản Ảo**: Khẳng định tuyệt đối Tiên Thạch, Ngọc, Cổ trùng, Nguyên Thạch không có giá trị quy đổi ra tiền thật; nghiêm cấm giao dịch tiền thật ngoài đời (RMT) dưới mọi hình thức để tránh vi phạm quy định cấm cờ bạc/kinh doanh trái phép của NĐ 147/2024/NĐ-CP.
  - **Bảo toàn Kiến trúc Codebase**: Triển khai giải pháp nhẹ gọn nhất (nhỏ nhất mà giải quyết trọn vẹn yêu cầu), không can thiệp sâu vào các luồng gameplay độc lập.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-terms-of-service-and-game-rules](issues/01-terms-of-service-and-game-rules.md)]: Chuẩn hóa văn bản Điều khoản Dịch vụ & Quy chế Hoạt động Web Game gồm 9 điều khoản theo Nghị định 147/2024/NĐ-CP và Luật An ninh mạng 2018; khóa chặt tính chất phi tiền tệ của vật phẩm ảo, nghiêm cấm tuyệt đối RMT, cơ chế chế tài 4 cấp độ và chính sách bảo lưu Ngọc khi Wipe; kiểm chứng qua văn bản đặc tả và prototype preview HTML.
- [[02-personal-data-protection-policy](issues/02-personal-data-protection-policy.md)]: Chuẩn hóa văn bản Chính sách Bảo vệ Dữ liệu Cá nhân & Thông báo Xử lý Dữ liệu gồm 11 điều khoản theo Nghị định 13/2023/NĐ-CP; xác lập ma trận dữ liệu tối thiểu, cam kết không thu thập dữ liệu nhạy cảm, nguyên tắc chia sẻ giới hạn với PayOS và quy trình 72h thực thi quyền xóa dữ liệu (Right to be forgotten); kiểm chứng qua văn bản đặc tả và prototype preview HTML.
- [[03-payment-disclaimer-and-ip-notice](issues/03-payment-disclaimer-and-ip-notice.md)]: Chuẩn hóa văn bản Quy chế Nạp tiền PayOS, Tuyên bố Thử nghiệm Alpha và Bản quyền Tác phẩm Cổ Chân Nhân gồm 7 điều khoản; xác định bản chất đóng góp tự nguyện (Donation), chính sách không hoàn tiền (Non-refundable), cam kết bảo lưu 100% Ngọc khi Wipe Server, và tuyên bố Fan-game tôn trọng quyền tác giả tiểu thuyết gốc; kiểm chứng qua văn bản đặc tả và prototype preview HTML.
- [[04-frontend-consent-and-health-warnings](issues/04-frontend-consent-and-health-warnings.md)]: Đặc tả kiến trúc UI/UX tích hợp Điều khoản & Cảnh báo sức khỏe: Checkbox Consent Opt-in trên form Register, LegalModal đọc 3 văn bản, LegalFooter gắn thường trực cảnh báo 180 phút NĐ 147 và hook usePlaytimeWarning nhắc nhở tại các mốc 60m, 120m, 180m; kiểm chứng qua đặc tả component và prototype preview HTML.
- [[05-backend-consent-audit-and-data-erasure](issues/05-backend-consent-audit-and-data-erasure.md)]: Đặc tả mở rộng DB schema bảng players và API Backend theo NĐ 13/2023: Lưu vết 4 trường consent audit khi đăng ký, endpoint POST /api/player/request-deletion vô danh hóa email/IP bảo toàn toàn vẹn FK và PayOS audit, cùng cơ chế versioning re-consent khi nâng cấp ToS; kiểm chứng qua đặc tả schema và prototype demo HTML.

## Open Tickets (Frontier & Blocked)

*(Không còn ticket mở — Toàn bộ bản đồ kiến trúc pháp lý & kỹ thuật đã được đặc tả và kiểm chứng hoàn tất)*

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Bộ lọc kiểm duyệt ngôn từ xúc phạm & an ninh mạng (Profanity & Cyber-security Filter)**: Giải pháp tự động kiểm tra và ngăn chặn từ khóa độc hại, xúc phạm danh dự nhân phẩm hoặc vi phạm Điều 8 Luật An ninh mạng 2018 trên kênh Chat thế giới và tên Tông Môn.
- **Xác thực danh tính bằng Số điện thoại / OTP**: Nghiên cứu phương án tích hợp SMS OTP hoặc Zalo ZNS nếu dự án mở rộng quy mô cần đáp ứng Điều 82 Nghị định 147/2024/NĐ-CP về xác thực tài khoản mạng xã hội / game có tương tác.
- **Cơ chế giới hạn cứng thời gian chơi cho người dưới 18 tuổi (Hard Playtime Lockout)**: Tự động ngắt kết nối sau 60 phút/ngày nếu áp dụng chặt chẽ quy định hạn chế người chơi chưa đủ 18 tuổi theo NĐ 147 Điều 40.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Thủ tục hành chính xin cấp phép thực tế tại Cục PTTH&TTĐT**: Bản đồ này chỉ xây dựng văn bản pháp lý, quy chế công bố và tích hợp kỹ thuật trên nền tảng web game; không phụ trách thủ tục thành lập pháp nhân và nộp hồ sơ xin giấy phép G1/G2 ngoài đời thực.
- **Tích hợp Cổng thanh toán quốc tế phức tạp (Stripe/PayPal)**: Giữ nguyên cổng nạp PayOS nội địa hiện tại, không mở rộng sang các nghĩa vụ thuế và pháp lý xuyên biên giới khác.
