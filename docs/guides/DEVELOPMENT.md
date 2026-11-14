# Hướng dẫn phát triển

## Yêu cầu

- Go version theo `backend/go.mod`.
- Node.js theo `.nvmrc` và npm.
- PostgreSQL 16 và Redis 7 khi chạy bằng Compose local.

## Chạy local bằng Docker Compose

```bash
docker compose -f docker-compose.local.yml up --build
```

Profile local chạy PostgreSQL, Redis, backend HTTP ở `:8080` và frontend ở `:3000`. Database volume được giữ giữa các lần chạy. Tạo/cập nhật `backend/.env` từ `backend/.env.example` trước khi chạy; không commit secrets.

## Chạy từng phần

```bash
docker compose up -d db redis
cd backend && go run ./cmd/server
```

Ở terminal khác:

```bash
cd frontend && npm install && npm run dev
```

Frontend mặc định dùng backend `http://localhost:8080` nếu không cấu hình URL. Với server chạy HTTP local, để `TLS_ENABLED=false`, `WT_ENABLED=false`; WebSocket fallback dùng `/api/ws`. WebTransport yêu cầu HTTPS và certificate tin cậy.

Sau lần cài DB mới, đăng nhập Admin và import các bộ dữ liệu cần thiết ở tab **Dữ liệu**. Không phải mọi CSV đều được tự nạp khi khởi động.

## Kiểm tra

```bash
(cd backend && go test ./...)
(cd frontend && npm run lint)
(cd frontend && npm run typecheck)
(cd frontend && npm test)
(cd frontend && npm run build)
```

Khi thay đổi logic concurrency backend, chạy thêm `go test -race ./...`.

## Startup backend

`cmd/server/main.go` thực hiện các bước chính:

1. Tìm `.env` theo thứ tự `.env`, `backend/.env`, `../.env`, `../../.env`.
2. Kết nối PostgreSQL và chạy `db.InitSchema()`.
3. Kết nối Redis với retry giới hạn. Khi kết nối thất bại, server ghi lỗi và dùng fallback trong bộ nhớ cho một số chức năng.
4. Nạp registry Gu, item, race, constitution, status, monster, zone, quest, loot và recipe từ DB/dataset.
5. Khởi tạo service được dùng chung, các worker, router và realtime hub.
6. Dừng worker/server theo context khi nhận SIGINT/SIGTERM.

Schema khởi tạo idempotent; đây không phải versioned migration framework. DDL mới phải có cả fresh-install definition và compatibility DDL theo schema domain. Kiểm tra riêng các script data migration có bước chạy thủ công.

## Cấu hình thường dùng

Nguồn đầy đủ là `backend/.env.example`; các biến dưới đây là nhóm quan trọng:

| Nhóm | Biến | Ghi chú |
| --- | --- | --- |
| HTTP | `PORT`, `GIN_MODE`, `PROD_ENV`, `LOG_LEVEL` | HTTP port mặc định 8080; logger mặc định INFO |
| PostgreSQL | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | DB local mặc định `localhost:5432`, `gu_admin`, `gu_db` |
| DB timeouts/pool | `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS`, `DB_IDLE_IN_TX_TIMEOUT_MS`, `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS` | Timeout query/lock và kích thước pool |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_SSL`, `REDIS_CONNECT_ATTEMPTS`, `REDIS_CONNECT_DELAY_MS` | TLS Redis có `REDIS_TLS_SERVER_NAME`, `REDIS_TLS_CA_FILE` |
| Auth | `JWT_SECRET`, `AUTH_COOKIE_DOMAIN`, `AUTH_COOKIE_SECURE`, `AUTH_COOKIE_SAMESITE`, `CORS_ALLOWED_ORIGINS` | Đặt secret an toàn ở môi trường deploy |
| Email/OAuth | `REQUIRE_EMAIL_VERIFICATION`, `SMTP_*`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` | OTP/Google login chỉ hoạt động khi cấu hình đủ |
| Realtime | `TLS_ENABLED`, `WT_ENABLED`, `WS_FALLBACK_ENABLED`, `TLS_ADDR`, `WT_ADDR`, `TLS_CERT_FILE`, `TLS_KEY_FILE` | WebTransport cần HTTPS; đọc runbook production trước khi public |
| Thanh toán | `PAYOS_*` | PayOS webhook/order yêu cầu cấu hình provider |
| Gameplay | `COMBAT_MAX_GU_ACTIONS_PER_TURN`, `MONSTER_RANK_SCALING_ENABLED`, `MONSTER_STAT_MULTIPLIER` | Thay đổi rule áp dụng theo code runtime |

Các biến client `NEXT_PUBLIC_*` được nhúng vào frontend lúc build. Cần build lại frontend sau khi đổi URL public. `INTERNAL_BACKEND_URL` dành cho server-side frontend.

## Worker nền

Worker hiện là goroutine polling loop, không phải queue. Mọi ticker lặp phải dùng `internal/worker/runner.go` (`RunEvery`/`RunEveryE`); công việc một lần hoặc cần flush khi dừng dùng `SafeRun`/`SafeRunE`. Nhận `context.Context`, giới hạn thời gian/khối lượng mỗi tick và ghi log lỗi. Worker gồm xử lý đói/tăng trưởng Gu và pet, độ kiếp, phiên luyện Cổ/combat, mỏ, auction, tournament, daily mission, sect maintenance/mission/territory, trạng thái hết hạn và mô phỏng xã hội tiên khiếu.

## Quan sát lỗi

Backend gắn trace ID cho request và cung cấp thống kê/traces trong Admin → **Giám sát**. Không thêm endpoint debug public để lộ dữ liệu runtime. `DEBUG_ERRORS` chỉ dùng ngoài production khi cần response lỗi chi tiết.

## Khi cập nhật docs

- Route: [API.md](API.md).
- Schema hoặc dataset: [GAME_DATA.md](GAME_DATA.md).
- Hành vi gameplay: [WIKI.md](../../WIKI.md).
- Kiến trúc player: [PLAYER_DOMAIN.md](../architecture/PLAYER_DOMAIN.md).
