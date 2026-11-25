# Chốt danh sách Nhóm Menu mặc định và phân bổ 25 mục tính năng

Type: grilling
Status: resolved
Blocked by: none

Part of [Wayfinder: Tái cấu trúc Menu Lobby & Tùy biến Lưới Lobby](https://github.com/VipPro67/GuWorld/issues/455); canonical ticket: [Chốt danh sách Nhóm Menu mặc định và phân bổ 25 mục tính năng](https://github.com/VipPro67/GuWorld/issues/456).

## Question

Chốt danh sách **Nhóm Menu** mặc định, tên gọi, icon/visual, và cách phân bổ toàn bộ 25 **Mục tính năng** hiện có trong `LobbyView.tsx`:
- Nhóm nào nên được tạo (ví dụ: Chiến Đấu / Thí Luyện, Thương Hội, Bí Cảnh & Giải Trí, Tiện Ích & Ký Sự)?
- Mỗi nhóm chứa chính xác những mục tính năng nào?
- Những mục cốt lõi nào mặc định hiển thị trực tiếp ở **Lưới Lobby**?
- Xử lý `UtilitiesView` (`utilities` - "Tiện Ích") hiện tại như thế nào: giữ làm 1 mục con trong nhóm, hay gộp/nâng cấp thành Nhóm Menu?
- Xử lý trạng thái khóa (`Tiên Khiếu` rank < 6) và nhãn động (`explore` ở `aperture_interior`) ra sao?
