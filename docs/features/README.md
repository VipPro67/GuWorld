# Hướng dẫn tính năng

Đọc [mục lục tính năng](../guides/FEATURES.md) để tìm tài liệu theo domain. Các trang mô tả flow người chơi, điều kiện và giới hạn đang có, cùng các điểm trong source dùng để đối chiếu.

## Cách giữ tài liệu đồng bộ

Trước khi đổi behavior, tìm trang feature tương ứng và cập nhật code cùng docs trong một thay đổi. Khi docs phát hiện behavior mong muốn chưa có, triển khai code để khớp mô tả. Với thay đổi chỉ làm rõ behavior hiện tại, xác minh qua source rồi cập nhật docs mà không tạo thay đổi gameplay không cần thiết. WIKI.md là bản hướng dẫn ngắn cho người chơi; các trang trong thư mục này là bản chi tiết.

Các công thức được ghi theo backend hiện tại, kèm đơn vị và giới hạn khi source áp dụng clamp/cap. Giá trị từ CSV, profile, DB hoặc runtime không được trình bày như hằng số cố định; preview/API server vẫn là kết quả cuối cho một thao tác cụ thể.
