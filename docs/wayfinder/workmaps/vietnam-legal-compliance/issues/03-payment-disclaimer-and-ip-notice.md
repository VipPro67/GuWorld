# Ticket 03: Soạn thảo Quy chế Nạp tiền PayOS, Tuyên bố Thử nghiệm & Bản quyền Tác phẩm

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Cần soạn thảo văn bản **Quy chế Nạp tiền (Payment Terms), Tuyên bố Miễn trừ Trách nhiệm Máy chủ Thử nghiệm (Alpha Test Disclaimer) và Tuyên bố Bản quyền Tác phẩm (Intellectual Property Notice)** nhằm bảo vệ tối đa dự án trước các rủi ro pháp lý về giấy phép phát hành game và quyền sở hữu trí tuệ?

Cụ thể cần giải quyết:
1. **Quy chế Nạp tiền & Ví Ngọc (PayOS Payment Policy)**:
   - Nêu rõ các gói nạp qua PayOS (10.000 VND đến 200.000 VND) là hình thức ủng hộ duy trì máy chủ (Donation / Khảo nghiệm tính năng nạp tự động).
   - Chính sách không hoàn tiền (Non-refundable): Do số Ngọc/Tiên Thạch được cộng ngay lập tức vào tài khoản game sau khi webhook PayOS xác nhận thành công, các khoản nạp không được hoàn lại tiền mặt dưới bất kỳ hình thức nào.
   - Khẳng định Ngọc chỉ dùng để trải nghiệm các tính năng trong game (Gacha, Tiên Khiếu, Mua vật phẩm từ NPC), không thể bán ngược lại cho ban quản trị để lấy VND.
2. **Tuyên bố Miễn trừ Trách nhiệm Máy chủ Thử nghiệm (Alpha/Beta Test Disclaimer)**:
   - Tuyên bố rõ GuWorld đang trong giai đoạn thử nghiệm công nghệ giới hạn (Alpha/Beta Test).
   - Quy định về tính ổn định dữ liệu: Ban quản trị nỗ lực tối đa để bảo lưu tiến trình chơi, tuy nhiên người chơi chấp nhận rủi ro có thể có rollback hoặc wipe dữ liệu khi cập nhật cấu trúc hệ thống lớn (nếu có sẽ thông báo trước và có chính sách bù đắp Ngọc tương ứng).
   - Miễn trừ trách nhiệm về các gián đoạn mạng, bảo trì đột xuất, lỗi kết nối hoặc thiệt hại gián tiếp phát sinh từ việc sử dụng game.
3. **Tuyên bố Sở hữu Trí tuệ & Tác phẩm Phái sinh (IP Notice)**:
   - Ghi nhận bối cảnh, thuật ngữ và ý tưởng thế giới Cổ Sư / Cổ Tiên lấy cảm hứng từ tiểu thuyết "Cổ Chân Nhân" (Reverend Insanity) của tác giả Cổ Chân Nhân.
   - Tuyên bố đây là sản phẩm sáng tạo độc lập của người hâm mộ (Fan-game / Phi thương mại), không mạo danh tác giả hay đơn vị phát hành gốc.
   - Bản quyền mã nguồn backend/frontend thuộc về đội ngũ phát triển GuWorld.

## Answer

Đã hoàn thiện toàn diện văn bản **Quy Chế Đóng Góp Duy Trì Máy Chủ, Tuyên Bố Thử Nghiệm & Bản Quyền Tác Phẩm (Phiên bản 2024.1)** với 4 phần và 7 điều khoản chi tiết:

1. **Quy Chế Đóng Góp PayOS & Đồng Ngọc (Phần I - Điều 1, 2, 3)**:
   - Xác định bản chất các gói nạp (10.000 VND đến 200.000 VND) là đóng góp tài chính tự nguyện (Donation) phục vụ duy trì hạ tầng máy chủ và băng thông; hệ thống tri ân bằng việc cộng Ngọc (Jade) in-game.
   - **Khóa chặt tính chất phi tiền tệ theo Điều 53-57 NĐ 147/2024/NĐ-CP**: Ngọc là điểm thưởng giải trí nội bộ, tuyệt đối không có giá trị quy đổi thành tiền mặt, không được rút tiền và nghiêm cấm mua bán bằng tiền thật ngoài game.
   - **Chính sách không hoàn tiền (Strict Non-refundable Policy)**: Ngọc được cấp ngay tức thì sau khi webhook xác thực; mọi khoản thanh toán không được hoàn lại tiền mặt dưới bất kỳ lý do nào.

2. **Tuyên Bố Miễn Trừ Trách Nhiệm Máy Chủ Thử Nghiệm Alpha (Phần II - Điều 4 & 5)**:
   - Định vị dự án ở trạng thái Thử nghiệm Alpha/Beta Giới hạn nghiên cứu công nghệ; không cấu thành dịch vụ thương mại có cam kết vĩnh viễn (giảm thiểu tối đa rủi ro về Giấy phép G1).
   - Miễn trừ trách nhiệm trước các sự cố gián đoạn kết nối mạng, lỗi kỹ thuật đang tinh chỉnh.
   - **Chính sách Bảo Lưu Ngọc khi Reset Máy chủ (Wipe Policy)**: Cam kết thông báo trước 07 ngày và **bảo lưu hoàng kim 100% số lượng Ngọc** đã nạp qua PayOS để hoàn trả đầy đủ vào tài khoản của người chơi khi mở máy chủ mới.

3. **Tuyên Bố Bản Quyền Tác Phẩm Phái Sinh & Tác Quyền Phần Mềm (Phần III - Điều 6 & 7)**:
   - Tôn trọng quyền tác giả: Ghi nhận nguồn cảm hứng thế giới quan Cổ Sư / Cổ Tiên / Đạo Ngân từ tiểu thuyết *"Cổ Chân Nhân" (Reverend Insanity)* của tác giả Cổ Chân Nhân (Gu Zhen Ren / 蛊真人).
   - Tuyên bố Fan-game phi thương mại / Sử dụng hợp lý (Fair Use), không mạo danh tác giả gốc, không cạnh tranh thương mại.
   - Khẳng định bản quyền tác giả đối với toàn bộ mã nguồn backend (Go), frontend (Next.js/TypeScript), turn-based simulator và hệ thống gamefx thuộc về Đội ngũ Phát triển GuWorld; nghiêm cấm sao chép, trích xuất dữ liệu trái phép.

4. **Tài Sản Đặc Tả & Giao Diện Kiểm Chứng Trực Quan**:
   - Văn bản pháp lý hoàn chỉnh (Markdown): [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/03_payment_and_ip_disclaimer_draft.md`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/03_payment_and_ip_disclaimer_draft.md)
   - Giao diện HTML xem thử tương tác: [`docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/03_payment_and_ip_preview.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/vietnam-legal-compliance/prototypes/03_payment_and_ip_preview.html)  
     *(Tích hợp bảng 5 gói nạp PayOS, cảnh báo không hoàn tiền và Trình giả lập tính toán bảo lưu Ngọc khi Wipe Server).*
