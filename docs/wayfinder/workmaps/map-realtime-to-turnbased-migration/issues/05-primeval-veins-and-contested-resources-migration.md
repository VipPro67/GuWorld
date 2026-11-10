# Chuyển Đổi Mỏ Nguyên Tuyền, Vận Tiêu & Tài Nguyên Cạnh Tranh Sang Mô Hình Node
Status: resolved
Type: grilling
Blocked by: 02
Part of #map

## Question

Các tài nguyên dã ngoại động trước đây phụ thuộc vào Realtime Zone Sim (như `primeval_veins`, Mỏ Môn Phái, Điểm tài nguyên thế giới) sẽ được chuyển đổi sang mô hình Node như thế nào?
- Thu hoạch Nguyên Tuyền: Cơ chế tích lũy sản lượng và thu hoạch khi đứng tại Node mỏ.
- Cơ chế Vận Tiêu / Cướp Tiêu (Caravan Escort & Intercept): Lộ trình vận chuyển hàng hóa giữa các thành thị, cách người chơi khác phát hiện và kích hoạt trận phục kích cướp tiêu.
- Tranh đoạt Lãnh địa & Mỏ Môn Phái: Cách thức tuyên chiến, trấn giữ mỏ và giải quyết tranh chấp tài nguyên theo thể thức Turn-based.

## Answer

1. **Cơ chế Khai Thác Mỏ Nguyên Tuyền ([`backend/internal/handlers/primeval_vein.go`](file:///home/vippro67/GitHub/GuWorld/backend/internal/handlers/primeval_vein.go)):**
   - Bảng `primeval_veins` tiếp tục quản lý các mỏ tự nhiên trên khắp các Node bản đồ.
   - Khi đứng tại Node mỏ, người chơi chọn hành động **"Đả Tọa Hấp Thu"**:
     - Nhân vật chuyển sang trạng thái khai thác, hệ thống worker nền tích lũy Nguyên Thạch vào túi đồ của người chơi theo chu kỳ (không phụ thuộc vào 20Hz zone sim).
     - Mỗi mỏ có số lượng vị trí đả tọa hữu hạn. Khi hết chỗ, Cổ Sư khác có thể phát động **"Tranh Chấp Chiếm Mỏ"** để thách đấu người đang ngồi; kẻ chiến thắng giành lấy vị trí đả tọa.

2. **Hệ thống Vận Tiêu & Phục Kích Cướp Tiêu (Core PvP Activity):**
   - **Lập Thương Đội:** Người chơi dùng vốn Nguyên Thạch mua các loại đặc sản (tơ tằm Nam Cương, quặng sắt Tây Mạc, dạ minh châu Đông Hải) và lập hành trình vận tiêu sang đại vực khác.
   - **Thuê Hộ Tiêu:** Người vận tiêu có thể mở phòng tuyển mộ người chơi khác làm "Hộ Tiêu" (chia % lợi nhuận khi cập bến an toàn).
   - **Cơ chế Phục Kích:** Khi đoàn tiêu di chuyển qua các tuyến đường nối giữa các Node, người chơi mang danh phận Ma Đạo có thể dùng Cổ Trinh Sát để dò tìm và chọn **"Phục Kích Cướp Tiêu"**.
   - Trận chiến Turn-based nổ ra giữa nhóm Cướp tiêu và nhóm Hộ tiêu (với cơ chế Mirror Clone hỗ trợ nếu hộ tiêu đang bận/offline). Nếu cướp thành công, kẻ cướp đoạt 40-60% giá trị hàng hóa; nếu hộ tiêu đẩy lùi được cướp tiêu, đoàn tiêu tiếp tục hành trình.

3. **Tranh Đoạt Mỏ Môn Phái & Lãnh Thổ:**
   - Bang hội chiếm giữ các mỏ khoáng sản lớn sẽ cắt cử thành viên vào danh sách "Trấn Thủ Mỏ".
   - Môn phái đối địch có thể phát động lệnh "Tấn Công Mỏ Môn Phái" trong khung giờ chiến sự; các trận đánh Turn-based phân hạng sẽ quyết định điểm chiếm đóng và quyền kiểm soát mỏ.

