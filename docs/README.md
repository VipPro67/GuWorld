# Tài liệu GuWorld

Các tài liệu bên dưới mô tả hành vi hiện có trong mã nguồn. Các bản thiết kế hoặc ghi chú triển khai cũ không được xem là cam kết tính năng.

| Tài liệu | Đối tượng đọc | Nội dung |
| --- | --- | --- |
| [Hướng dẫn phát triển](guides/DEVELOPMENT.md) | Developer, operator | Chạy ứng dụng, cấu hình, vòng đời backend và kiểm tra |
| [API](guides/API.md) | Frontend, tích hợp | Route backend hiện đăng ký và yêu cầu xác thực tổng quát |
| [Tính năng gameplay](guides/FEATURES.md) | Developer, tester | Từng tính năng, điều kiện/flow và nơi sở hữu trong source |
| [Chi tiết theo domain](features/README.md) | Developer, tester | Hướng dẫn chi tiết và quy ước giữ docs đồng bộ với code |
| [Dữ liệu game](guides/GAME_DATA.md) | Designer, admin, developer | CSV, import/export, registry và effects |
| [Triển khai production](guides/PRODUCTION_DEPLOY.md) | Operator | Cấu hình triển khai, TLS và realtime transport |
| [Kiến trúc](architecture/ARCHITECTURE.md) | Developer | Thành phần runtime và ranh giới module |
| [Miền Player](architecture/PLAYER_DOMAIN.md) | Developer | Quyền sở hữu dữ liệu player, snapshot và mutation |
| [Wiki gameplay](../WIKI.md) | Người chơi, tester | Mô tả các hệ thống gameplay đang có |
| [Trạng thái dự án](../PROJECT.md) | Cả nhóm | Tóm tắt codebase và feature inventory |

## Nguyên tắc nguồn chuẩn

- Route được đăng ký trong `backend/internal/server/server.go`.
- Schema được khởi tạo từ `backend/internal/db/schema_*.go` và `InitSchema()`.
- Gameplay nằm chủ yếu trong `backend/internal/handlers/`, với simulator combat trong `backend/internal/simulator/`.
- Màn hình và client state nằm trong `frontend/src/app/`, `frontend/src/components/` và `frontend/src/store/`.
- Giá trị gameplay dạng bảng lấy từ `backend/data/` và dữ liệu đã import vào PostgreSQL.

Khi tài liệu và code mâu thuẫn, code hiện tại là hành vi thực tế; hãy cập nhật tài liệu liên quan cùng thay đổi.
