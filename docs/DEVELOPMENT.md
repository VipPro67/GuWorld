# Development Guide

## Prerequisites

- Go theo `backend/go.mod`.
- Node.js/npm tương thích Next.js 16.
- PostgreSQL cho backend.
- Redis optional.

## Environment variables

Backend đọc `.env` từ các vị trí: `.env`, `backend/.env`, `../.env`, `../../.env`.

Các biến thường dùng:

| Variable | Mục đích |
| --- | --- |
| `PORT` | Port backend, mặc định `8080`. |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Kết nối PostgreSQL; có default local trong `db.Connect()`. |
| `CORS_ALLOWED_ORIGINS` | Danh sách origin frontend, mặc định `http://localhost:3000`. |
| `JWT_SECRET` | Ký token/cookie auth. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `MAIL_FROM` | Cấu hình SMTP (ví dụ: Brevo) để gửi mã OTP đăng ký. |
| `REDIS_ADDR`/Redis config | Optional cache/worker support nếu được cấu hình. |
| Google OAuth vars | Dùng cho `/auth/google/*` nếu bật Google login. |
| PayOS vars | Dùng cho topup/webhook nếu bật payment thật. |
| Cloudinary vars | Dùng cho upload avatar nếu bật upload cloud. |

Frontend:

| Variable | Mục đích |
| --- | --- |
| `NEXT_PUBLIC_BACKEND_URL` | Backend base URL; mặc định `http://localhost:8080`. |

## Run with Docker Compose

Bạn có thể khởi chạy nhanh toàn bộ môi trường backend (bao gồm Go app, PostgreSQL, Redis và Caddy làm reverse proxy) bằng Docker Compose:

```bash
docker-compose up --build
```

Lệnh này sẽ tự động:
1. Tải và cấu hình PostgreSQL (`gu_db`) và Redis.
2. Build Go backend dựa trên `backend/Dockerfile` và sao chép các file CSV dữ liệu đi kèm.
3. Khởi chạy Caddy Server để làm reverse proxy và tự động cấp chứng chỉ SSL (HTTPS) cho tên miền `api.cogioi.io.vn` (cấu hình trong `Caddyfile`).
4. Sử dụng cơ chế healthcheck để trì hoãn khởi động Go app cho đến khi PostgreSQL đã hoàn toàn sẵn sàng nhận kết nối.

API backend sau đó sẽ lắng nghe tại `http://localhost:8080` (truy cập cục bộ từ host) và được expose ra ngoài qua Caddy tại `https://api.cogioi.io.vn`.

## Run backend (Manual)

```bash
cd backend
go run ./cmd/server
```

Server sẽ:

1. Kết nối DB.
2. Tạo/migrate schema.
3. Seed CSV.
4. Load registry.
5. Start workers.
6. Listen HTTP.

## Run frontend

```bash
cd frontend
npm install
npm run dev
```

Mở `http://localhost:3000`.

## Checks used in this repo

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
npm run lint
npm run build
```

## Documentation maintenance checklist

- Route thay đổi: cập nhật `docs/API.md`.
- Component/view gameplay thay đổi: cập nhật `PROJECT.md` và `WIKI.md` nếu ảnh hưởng người chơi.
- CSV/schema thay đổi: cập nhật `docs/GAME_DATA.md`.
- Lệnh chạy/env thay đổi: cập nhật tài liệu này và README tương ứng.
