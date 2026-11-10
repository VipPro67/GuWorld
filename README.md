# GuWorld

Effect migration is non-destructive by default. From `backend/`, run
`go run ./cmd/migrate_gu_effects -format gu -source data/Gu.csv` to produce a
machine-readable audit without changing CSV data. See `docs/guides/GAME_DATA.md` for
format-specific conversion and migration-gate rules.

GuWorld là web game RPG text-based lấy cảm hứng từ hệ thống tu luyện/Cổ trùng. Codebase hiện tại gồm:

- **Backend:** Go + Gin, PostgreSQL, Redis (bắt buộc), JWT/cookie auth.
- **Frontend:** Next.js App Router + React + Zustand.
- **Dữ liệu gameplay:** CSV trong `backend/data/`, được quản trị viên import/export ở tab **Dữ liệu**; backend không tự import các dictionary khi khởi động.

## Tài liệu đang dùng

| Tài liệu | Mục đích |
| --- | --- |
| [`PROJECT.md`](./PROJECT.md) | Trạng thái dự án, module chính và phạm vi hiện tại. |
| [`WIKI.md`](./WIKI.md) | Wiki gameplay cho người chơi/tester. |
| [`docs/guides/API.md`](./docs/guides/API.md) | Nhóm API backend theo router hiện tại. |
| [`docs/architecture/ARCHITECTURE.md`](./docs/architecture/ARCHITECTURE.md) | Kiến trúc thư mục, luồng khởi động và module runtime. |
| [`docs/guides/GAME_DATA.md`](./docs/guides/GAME_DATA.md) | Nguồn dữ liệu CSV/seed và registry runtime. |
| [`docs/guides/DEVELOPMENT.md`](./docs/guides/DEVELOPMENT.md) | Cách chạy, biến môi trường, kiểm tra nhanh. |
| [`docs/guides/PRODUCTION_DEPLOY.md`](./docs/guides/PRODUCTION_DEPLOY.md) | Checklist production: DNS, TLS, firewall, cert, WebTransport, rollback và smoke test. |
| [`frontend/README.md`](./frontend/README.md) | Ghi chú riêng cho frontend. |

Xem [mục lục tài liệu](./docs/README.md) để tìm tài liệu theo nhóm. Các tài liệu thiết kế/roadmap chỉ mô tả hướng phát triển. Các tài liệu trong bảng trên là điểm bắt đầu để đối chiếu hành vi đang chạy.

## Chạy nhanh local

### Docker Compose (Khuyên dùng cho Backend)

```bash
docker compose -f docker-compose.local.yml up --build
```

Lệnh này khởi chạy Go backend, PostgreSQL và Redis ở chế độ local (plain HTTP
`:8080`, không TLS, không Caddy). Production dùng `docker-compose.yml`: Go phục
vụ HTTPS/WebSocket (TCP/443) và WebTransport (UDP/443) trong cùng process. Client ưu tiên WebTransport và dùng WebSocket fallback khi WT không thiết lập được. Compose production cần `certs/fullchain.pem` +
`certs/privkey.pem` (đọc bởi GID `10001`) và host phải mở TCP/UDP 443 — xem
[`docs/guides/PRODUCTION_DEPLOY.md`](./docs/guides/PRODUCTION_DEPLOY.md). Muốn chạy local
không TLS bằng binary, đặt `TLS_ENABLED=false`, `WT_ENABLED=false` và dùng
`go run ./cmd/server` (plain HTTP `:8080`); WebSocket fallback có thể dùng `ws://localhost:8080/api/ws` khi dev không bật WT. WebTransport cần HTTPS/cert tin cậy. Backend tự khởi tạo schema cùng
dữ liệu hệ thống tối thiểu. Sau lần cài mới, quản trị viên phải import dữ liệu
gameplay từ tab **Dữ liệu**; `Quest.csv` hiện là ngoại lệ được đồng bộ khi khởi
tạo schema.

Frontend Compose chạy ở `:3000`. Cấu hình `NEXT_PUBLIC_BACKEND_URL`,
`NEXT_PUBLIC_REALTIME_WS_URL`, `NEXT_PUBLIC_SITE_URL` và
`INTERNAL_BACKEND_URL` trong root `.env`; các biến `NEXT_PUBLIC_*` được nhúng
vào client bundle lúc build, nên cần build lại frontend sau khi đổi:
`docker compose -f docker-compose.local.yml up --build frontend`.

Thông tin DB/JWT/OAuth phải đặt trong `backend/.env`; thông tin Grafana Alloy đặt
trong root `.env`. Không lưu credential trong Compose hoặc Git. Khi dùng Redis
TLS, backend xác minh certificate và hỗ trợ `REDIS_TLS_SERVER_NAME` cùng
`REDIS_TLS_CA_FILE` cho CA riêng.

### Thủ công (Manual)

#### Backend

```bash
cd backend
go run ./cmd/server
```

Backend mặc định nghe `:8080`, tự khởi tạo schema và các bản ghi hệ thống tối thiểu khi kết nối được PostgreSQL. Quá trình này được serialize giữa các instance bằng PostgreSQL advisory lock và tự thử lại các lỗi lock tạm thời; cấu hình timeout/retry được mô tả trong `docs/guides/DEVELOPMENT.md`. Hãy import CSV trong Admin trước khi chơi và khởi động lại backend để nạp lại các registry được cache.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend mặc định gọi backend qua `NEXT_PUBLIC_BACKEND_URL` hoặc `http://localhost:8080` nếu biến môi trường chưa được đặt.

Kiểm tra frontend bằng `npm run lint` và `npm run build`.

### Realtime combat (đã nghỉ hưu)

Realtime combat simulation binaries and world combat were retired. Turn-based encounters use the backend simulator; remaining world actor, PvP, and arena conversion gaps are tracked in `docs/architecture/REALTIME_COMBAT_TO_TURN_BASED.md`.

## Module gameplay hiện có

- Đăng ký/đăng nhập, Google OAuth, hồ sơ, avatar, referral.
- Tu luyện phàm nhân, phá vách, dùng Xá Lợi Cổ, thăng tiên, độ kiếp.
- Không khiếu/tiên khiếu, đạo ngân, tài nguyên node, dị nhân và tai kiếp.
- Khám phá bản đồ, combat theo lượt trong các flow được hỗ trợ, đội hình 3x3, ground drop và retreat/flee.
- Inventory Cổ/vật phẩm/nguyên liệu, cho ăn, trang bị, bản mệnh, tiêu hao/hỗ trợ.
- Luyện Cổ, suy diễn công thức, sát chiêu.
- Shop NPC, Bảo Hoàng Thiên marketplace/auction/trade listing, P2P trade.
- Môn phái, nhiệm vụ, kho, đơn xin gia nhập/thăng chức, mỏ môn phái.
- Party, võ đài và shop võ đài, Đấu Trường Sinh Tử, Nguyên Tuyền, chat/mail, phó bản, hồn phách.
- Gacha, thành tựu/danh hiệu/khung, bảng xếp hạng, cược linh thú, thông báo, gift code, lịch sử chiến đấu và ký sự hoạt động.
- Ví/nạp tiền, premium shop, feedback và trang quản trị dữ liệu/giám sát.

## Nguyên tắc cập nhật tài liệu

1. Nếu thay đổi route backend, cập nhật `docs/guides/API.md`.
2. Nếu thay đổi schema/seed CSV, cập nhật `docs/guides/GAME_DATA.md`.
3. Nếu thay đổi flow chạy app, biến môi trường hoặc lệnh kiểm tra, cập nhật `docs/guides/DEVELOPMENT.md` và README liên quan.
4. Nếu thay đổi gameplay nhìn thấy được, cập nhật `WIKI.md`.
