# GuWorld

Thuật ngữ gameplay và thế giới dùng chung trong GuWorld.

## Thế giới realtime

**World Change Feed**:
Luồng thông báo những thay đổi trong thế giới theo khu vực, tọa độ và instance để các màn hình đang xem có thể cập nhật trạng thái.
_Avoid_: World event bus, map polling

## Effect gameplay

**Nội tại có điều kiện**:
Nội tại chỉ đóng góp hiệu lực khi điều kiện riêng của nó đang thỏa mãn.

**Hiệu ứng có thời hạn**:
Hiệu ứng đã kích hoạt và duy trì hiệu lực trong một khoảng thời gian được xác định.

**Nguồn effect**:
Cổ, chủng tộc, thể chất, trạng thái hoặc thực thể khác khởi phát hay cung cấp một effect.

**Bộ chọn mục tiêu effect**:
Quy tắc xác định đối tượng nhận effect, như nguồn, mục tiêu hành động, nhóm theo quan hệ hoặc vùng/mốc thế giới.

**Chủ sở hữu trạng thái effect**:
Đối tượng mà trạng thái effect đã áp dụng gắn vào trong thời hạn của nó; khi đối tượng này bị xóa, trạng thái effect đó cũng kết thúc.

**Path chính của hành động**:
Lưu phái đại diện cho hành động, dùng khi effect cần kiểm tra affinity theo path.

**Điều kiện race mục tiêu**:
Điều kiện effect so khớp race key cụ thể của tác nhân hoặc mục tiêu theo vai trò được khai báo; race không suy ra từ path và thiếu race trong context thì không khớp.

## Tiên Khiếu

**Tiên Khiếu vô chủ**:
Tiên Khiếu không có Cổ Tiên làm chủ, có thể tồn tại như địa điểm công cộng hoặc di sản còn lại sau khi chủ nhân thọ tận.

**Tiên Khiếu công cộng**:
Tiên Khiếu vô chủ vận hành như một map riêng mở cho mọi người chơi; người chơi có thể khám phá, thu hoạch mỏ và giao chiến PvP, nhưng không thể sửa đổi địa hình. Khi bị hấp thụ, nó chuyển thành Tiên Khiếu riêng của người hấp thụ.

**Hấp thụ Tiên Khiếu**:
Hợp nhất một Tiên Khiếu vô chủ vào Tiên Khiếu của Cổ Tiên đủ rank và cảnh giới lưu phái. Khi thành công, diện tích, Đạo Ngân, bộ đếm tai kiếp và mỏ tài nguyên được cộng vào khiếu nhận; khiếu bị hấp thụ bị xóa.

**Lưu phái chủ đạo của Tiên Khiếu**:
Lưu phái có Đạo Ngân cao nhất trong Tiên Khiếu; nếu nhiều lưu phái đồng hạng cao nhất, điều kiện hấp thụ phải đạt ở tất cả các phái đó.

**Môn Hộ**:
Cửa ra vào nối Tiên Khiếu với thế giới bên ngoài khi chủ nhân Hạ Khiếu.

**Hạ Khiếu**:
Đưa Tiên Khiếu ra thế giới bên ngoài và mở Môn Hộ tại một vị trí trong thế giới.

**Xâm nhập Tiên Khiếu**:
Vào Tiên Khiếu với tư cách người ngoài; hành động này không chuyển quyền làm chủ Tiên Khiếu.

**Trục xuất khách Tiên Khiếu**:
Chủ Tiên Khiếu buộc một khách rời khỏi Tiên Khiếu và trở lại thế giới bên ngoài.

## Đạo Ngân Môi Trường & Đạo Vực

**Đạo Ngân Môi Trường**:
Tổng thể đạo ngân đang hiện diện tại nơi diễn ra hành động hoặc giao chiến, ở thế giới bên ngoài hoặc bên trong Tiên Khiếu.

**Đạo Ngân Tự Nhiên**:
Đạo ngân vốn có của trời đất tại một khu vực hoặc ô bản đồ, chưa bị Tôn Giả nào luyện hóa.

**Đạo Vực**:
Khu vực môi trường có Đạo Ngân Tự Nhiên đã bị một Cửu Chuyển Cổ Tôn luyện hóa và kiểm soát.

## Lộ Trình Tu Hành

**Lộ Trình Tu Hành**:
Chuỗi nhiệm vụ chính tuyến tuần tự dẫn dắt người chơi từ Nhất Chuyển (Tân thủ) tới Cửu Chuyển Cổ Tôn, tách bạch với Nhiệm Vụ Hàng Ngày, Thành Tựu và Nhiệm Vụ Tông Môn.

**Chặng Tu Hành**:
Một giai đoạn lớn trong Lộ Trình Tu Hành gắn với một mốc cảnh giới và nhóm cơ chế trọng tâm.

**Bước Tu Hành**:
Một nhiệm vụ đơn lẻ có thứ tự tiên quyết trong Chặng Tu Hành, gồm bối cảnh giải thích cơ chế, mục tiêu cụ thể và phần thưởng.
