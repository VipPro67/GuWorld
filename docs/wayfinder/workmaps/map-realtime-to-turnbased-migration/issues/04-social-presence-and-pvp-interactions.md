# Cơ Chế Hiện Diện Cùng Điểm (Presence) & Tương Tác Xã Hội / PvP
Status: resolved
Type: grilling
Blocked by: 02
Part of #map

## Question

Làm thế nào để người chơi cảm nhận rõ ràng sự kết nối, đông đúc và cạnh tranh với nhau tại các địa danh mà không cần thấy avatar chạy trên canvas?
- Cơ chế Hiện diện (Presence): Broadcast danh sách người chơi cùng Node/Ô qua kênh nào (WebSocket SSE hiện có, hay polling nhẹ)?
- Chợ Vỉa Hè (Local Player Stalls): Cơ chế mở sạp buôn bán P2P trực tiếp tại các thành trấn / sơn trại mà không phải trả phí cắt cổ của Bảo Hoàng Thiên được vận hành ra sao?
- Thách đấu & Đồ sát (Wild PK): Trong môi trường Turn-based, nếu người chơi A muốn tấn công người chơi B ở dã ngoại, trận chiến sẽ diễn ra theo dạng Đồng bộ trực tiếp (cả 2 cùng online điều khiển theo lượt) hay Bán đồng bộ (người tấn công đánh với AI/Bóng phản chiếu - Mirror Clone của đối phương)?

## Answer

1. **Cơ chế Hiện Diện Sống (Node Presence):**
   - Sử dụng Redis Set theo key `presence:node:<region_id>:<node_id>` kết hợp WebSocket / SSE stream hiện có.
   - Khi bước vào Node, client nhận danh sách Cổ Sư đang có mặt tại đây, gồm: Avatar, Đạo hiệu, Cảnh giới, Môn phái, và **Trạng thái thực tế** (Đang bày sạp, Đang đả tọa hấp thu mỏ, Đang bế quan, Đang dưỡng thương, Đang tìm đội).
   - Tương tác 1-click: Xem thông tin công khai, Mời vào tổ đội, Đề nghị giao dịch P2P, hoặc Khiêu chiến.

2. **Chợ Vỉa Hè (Local Stalls):**
   - Tại các Node thành trấn/sơn trại an toàn, người chơi có thể chuyển sang trạng thái "Bày Sạp" (Stall Mode).
   - Hàng hóa trên sạp được niêm yết bằng Nguyên Thạch hoặc trao đổi Cổ trùng trực tiếp mà không phải chịu thuế giao dịch 5-10% của Thiên Linh Bảo Hoàng Thiên. Người khác đứng cùng Node có thể ghé thăm và mua hàng.

3. **Phân Tầng Thể Thức PvP (Lựa chọn C):**
   - **Võ Đài & Đấu Trường Sinh Tử ([`backend/internal/handlers/arena.go`](file:///home/vippro67/GitHub/GuWorld/backend/internal/handlers/arena.go)):** Áp dụng thi đấu Turn-based **Đồng bộ thời gian thực** (hai bên cùng online, đếm ngược 15-20s mỗi lượt ra chiêu).
   - **Dã Ngoại Đồ Sát & Cướp Tiêu (Wild PK):** Áp dụng cơ chế **Bán đồng bộ (Asynchronous Mirror Clone)**:
     - Kẻ tấn công bước vào trận Turn-based đối đầu với "Thần Niệm Phân Thân" của nạn nhân do AI điều khiển theo đúng bộ Cổ trùng, sát chiêu phòng thủ và đạo ngân đã cài đặt sẵn.
     - Sau trận đánh, server gửi Chiến Báo chi tiết qua Hòm thư cho nạn nhân. Nếu nạn nhân thua, tài nguyên/hàng hóa mang theo người bị cướp đoạt và nhân vật rơi vào trạng thái trọng thương.
     - Phương thức này triệt tiêu hoàn toàn vấn nạn "combat logging" (cố tình ngắt kết nối mạng để trốn chết) và phù hợp hoàn hảo với lối chơi web RPG.

