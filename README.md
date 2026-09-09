# GuWorld

GuWorld là web game RPG text-based lấy cảm hứng từ hệ thống tu luyện/Cổ trùng. Codebase hiện tại gồm:

- **Backend:** Go + Gin, PostgreSQL, Redis (bắt buộc), JWT/cookie auth.
- **Frontend:** Next.js App Router + React + Zustand.
- **Dữ liệu seed:** CSV trong `backend/` cho Cổ, vật phẩm, công thức, quái, loot, shop, vùng và gói nạp.

## Tài liệu đang dùng

| Tài liệu | Mục đích |
| --- | --- |
| [`PROJECT.md`](./PROJECT.md) | Trạng thái dự án, module chính và phạm vi hiện tại. |
| [`WIKI.md`](./WIKI.md) | Wiki gameplay cho người chơi/tester. |
| [`docs/API.md`](./docs/API.md) | Nhóm API backend theo router hiện tại. |
| [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) | Kiến trúc thư mục, luồng khởi động và module runtime. |
| [`docs/GAME_DATA.md`](./docs/GAME_DATA.md) | Nguồn dữ liệu CSV/seed và registry runtime. |
| [`docs/DEVELOPMENT.md`](./docs/DEVELOPMENT.md) | Cách chạy, biến môi trường, kiểm tra nhanh. |
| [`frontend/README.md`](./frontend/README.md) | Ghi chú riêng cho frontend. |

Các file phase/fix/design rời rạc cũ đã được gom lại hoặc loại bỏ để tránh lệch với code hiện tại.

## Chạy nhanh local

### Docker Compose (Khuyên dùng cho Backend)

```bash
docker-compose up --build
```

Lệnh này sẽ tự động khởi chạy Go backend, PostgreSQL và Redis. Backend sẽ nghe tại `:8080`, tự khởi tạo schema và seed dữ liệu từ CSV.

### Thủ công (Manual)

#### Backend

```bash
cd backend
go run ./cmd/server
```

Backend mặc định nghe `:8080`, tự khởi tạo schema và seed dữ liệu từ CSV khi kết nối được PostgreSQL.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend mặc định gọi backend qua `NEXT_PUBLIC_BACKEND_URL` hoặc `http://localhost:8080` nếu biến môi trường chưa được đặt.

## Module gameplay hiện có

- Đăng ký/đăng nhập, Google OAuth, hồ sơ, avatar, referral.
- Tu luyện phàm nhân, phá vách, dùng Xá Lợi Cổ, thăng tiên, độ kiếp.
- Không khiếu/tiên khiếu, đạo ngân, tài nguyên node, dị nhân và tai kiếp.
- Khám phá bản đồ, PvE combat, đội hình 3x3, loot tạm, retreat.
- Inventory Cổ/vật phẩm/nguyên liệu, cho ăn, trang bị, bản mệnh, tiêu hao/hỗ trợ.
- Luyện Cổ, suy diễn công thức, sát chiêu.
- Shop NPC, Bảo Hoàng Thiên marketplace/auction/trade listing, P2P trade.
- Môn phái, nhiệm vụ, kho, đơn xin gia nhập/thăng chức, mỏ môn phái.
- Party, arena, mỏ cá nhân, chat/mail, phó bản, hồn phách, ví/nạp tiền, premium shop, feedback.

## Nguyên tắc cập nhật tài liệu

1. Nếu thay đổi route backend, cập nhật `docs/API.md`.
2. Nếu thay đổi schema/seed CSV, cập nhật `docs/GAME_DATA.md`.
3. Nếu thay đổi flow chạy app, biến môi trường hoặc lệnh kiểm tra, cập nhật `docs/DEVELOPMENT.md` và README liên quan.
4. Nếu thay đổi gameplay nhìn thấy được, cập nhật `WIKI.md`.
