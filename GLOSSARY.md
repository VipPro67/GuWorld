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

**Môn Hộ**:
Cửa ra vào nối Tiên Khiếu với thế giới bên ngoài khi chủ nhân Hạ Khiếu.

**Hạ Khiếu**:
Đưa Tiên Khiếu ra thế giới bên ngoài và mở Môn Hộ tại một vị trí trong thế giới.

**Xâm nhập Tiên Khiếu**:
Vào Tiên Khiếu với tư cách người ngoài; hành động này không chuyển quyền làm chủ Tiên Khiếu.

**Trục xuất khách Tiên Khiếu**:
Chủ Tiên Khiếu buộc một khách rời khỏi Tiên Khiếu và trở lại thế giới bên ngoài.
