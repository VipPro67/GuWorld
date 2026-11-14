# Triển khai production

Tài liệu này ghi các điều kiện runtime hiện tại; điều chỉnh địa chỉ/domain theo môi trường thực tế.

## Thành phần và network

Production Compose (`docker-compose.yml`) chạy backend, PostgreSQL và Redis. Frontend được build/deploy theo cấu hình riêng của môi trường. Backend cần kết nối PostgreSQL và Redis. Giữ secrets trong secret manager hoặc env file ngoài Git.

Khi bật `TLS_ENABLED=true`, backend cần `TLS_CERT_FILE`, `TLS_KEY_FILE`; TLS listener mặc định `:8443` hoặc `TLS_ADDR`. WebTransport cần TLS/certificate hợp lệ và UDP port được mở theo listener/config. WebSocket dùng TCP và có thể fallback qua `/api/ws` nếu bật `WS_FALLBACK_ENABLED`.

Frontend cần `NEXT_PUBLIC_BACKEND_URL`, `NEXT_PUBLIC_SITE_URL` và realtime URL phù hợp với host public. Các `NEXT_PUBLIC_*` được nhúng lúc build. Server-side frontend có thể dùng `INTERNAL_BACKEND_URL` để kết nối nội bộ.

## Trước khi khởi động

- Đặt `JWT_SECRET` riêng, credential PostgreSQL/Redis và PayOS/OAuth/SMTP cần thiết.
- Cấu hình `CORS_ALLOWED_ORIGINS`, cookie domain/secure/samesite theo HTTPS domain.
- Đảm bảo persistent volume PostgreSQL và backup/restore đã được vận hành.
- Cấp certificate/key với quyền đọc cho container backend.
- Mở TCP cho HTTPS/API/WebSocket và UDP cho WebTransport khi bật.
- Đặt `TLS_ENABLED`/`WT_ENABLED` theo topology thật; không bật WT trên plain HTTP.

## Data rollout

`InitSchema()` tự chạy DDL và compatibility backfill idempotent, nhưng không thay thế backup hoặc script data migration cần thao tác. Sau khi deploy lần đầu, import CSV cần thiết qua Admin → **Dữ liệu**. Theo metadata import để biết registry/cache nào reload hoặc cần restart.

## Kiểm tra vận hành

- Gọi `GET /api/health` và kiểm tra build/protocol.
- Kiểm tra kết nối DB/Redis qua log và màn hình Admin → Giám sát.
- Đăng nhập frontend, kiểm tra dictionary/game data đã import, xác thực cookie, thao tác movement và session combat.
- Kiểm tra realtime transport từ client với domain/certificate public.
- Trước khi nâng cấp schema lớn, backup DB và đọc compatibility DDL/script migration liên quan.

Chi tiết option Compose/env được duy trì tại `docker-compose.yml`, `docker-compose.local.yml` và `backend/.env.example`.
