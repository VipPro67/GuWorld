# GuWorld

Effect migration is non-destructive by default. From `backend/`, run
`go run ./cmd/migrate_gu_effects -format gu -source data/Gu.csv` to produce a
machine-readable audit without changing CSV data. See `docs/GAME_DATA.md` for
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
| [`docs/API.md`](./docs/API.md) | Nhóm API backend theo router hiện tại. |
| [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) | Kiến trúc thư mục, luồng khởi động và module runtime. |
| [`docs/GAME_DATA.md`](./docs/GAME_DATA.md) | Nguồn dữ liệu CSV/seed và registry runtime. |
| [`docs/DEVELOPMENT.md`](./docs/DEVELOPMENT.md) | Cách chạy, biến môi trường, kiểm tra nhanh. |
| [`frontend/README.md`](./frontend/README.md) | Ghi chú riêng cho frontend. |

Các tài liệu thiết kế/roadmap chỉ mô tả hướng phát triển. Các tài liệu trong bảng trên là điểm bắt đầu để đối chiếu hành vi đang chạy.

## Chạy nhanh local

### Docker Compose (Khuyên dùng cho Backend)

```bash
docker-compose up --build
```

Lệnh này khởi chạy Go backend, PostgreSQL, Redis và reverse proxy. Backend nghe tại `:8080`, tự khởi tạo schema cùng dữ liệu hệ thống tối thiểu. Sau lần cài mới, quản trị viên phải import dữ liệu gameplay từ tab **Dữ liệu**; `Quest.csv` hiện là ngoại lệ được đồng bộ khi khởi tạo schema.

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

Backend mặc định nghe `:8080`, tự khởi tạo schema và các bản ghi hệ thống tối thiểu khi kết nối được PostgreSQL. Quá trình này được serialize giữa các instance bằng PostgreSQL advisory lock và tự thử lại các lỗi lock tạm thời; cấu hình timeout/retry được mô tả trong `docs/DEVELOPMENT.md`. Hãy import CSV trong Admin trước khi chơi và khởi động lại backend để nạp lại các registry được cache.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend mặc định gọi backend qua `NEXT_PUBLIC_BACKEND_URL` hoặc `http://localhost:8080` nếu biến môi trường chưa được đặt.

Kiểm tra frontend bằng `npm run lint` và `npm run build`.

### MMO realtime headless (đang phát triển)

```bash
cd backend
go run ./cmd/mmo_headless -ticks 10000
```

Lệnh chạy mô phỏng zone 20 Hz với 2 người chơi và 10 quái, in checksum của
snapshot cuối; không cần PostgreSQL, Redis hoặc frontend.

Smoke vertical slice (5 player + 10 quái, di chuyển/cast/death/metrics):
`cd backend && go run ./cmd/mmo_slice -players 5 -mobs 10 -ticks 400`.
Synthetic load: `go run ./cmd/mmo_bench -players 100 -mobs 50 -ticks 1000`.
Checklist Gate B: `docs/mmo/vertical-slice.md`; benchmark: `docs/mmo/benchmark.md`.

Phía client, bật đồng thời `NEXT_PUBLIC_REALTIME_WORLD_ENABLED` và
`NEXT_PUBLIC_PHASER_WORLD_CLIENT_ENABLED`, cấu hình
`MAP_VERSION`/`CONTENT_VERSION` khớp backend; sau đăng nhập client tự chọn map
theo `current_region` của nhân vật (hoặc dùng `NEXT_PUBLIC_MMO_ZONE_ID` làm zone
dự phòng khi preview). Client gửi `world.join`, render snapshot AOI và
dự phòng WASD/click-to-move; server vẫn xác nhận vị trí, HP, collision tĩnh, monster AI và
combat preview (phím 1–6, projectile/AoE/shield) với animation placeholder và VFX pool. Map preview
dùng chung JSON cho server/client; ngoài `realtime_preview` có pack Alpha 4 map liên kết
(`misty_forest`/`bone_crypt`/`crimson_ravine`/`emerald_grotto`, gồm boss/PvP/safe hub; xem
`docs/mmo/map-format.md`). Catalog Gu realtime gồm 6 MVP + 40 Alpha profile (xem
`docs/mmo/gu-realtime-schema.md`); art manifest và quy trình thay asset ở `docs/mmo/assets.md`.
Loot/checkpoint/reconnect đã có một phần; xem `docs/mmo/protocol-v1.md` và `docs/mmo/rollout.md`.
Trước khi beta, chạy `scripts/mmo-release-gate.sh` và điền evidence/sign-off trong
`docs/mmo/release-checklist.md`; SLO/capacity candidate ở `docs/mmo/slo.md`, roadmap ở
`docs/mmo/roadmap.md`. Automated subset pass không đồng nghĩa Gate D pass.
Tổng hợp giới hạn còn lại của PR-01…PR-36: `docs/mmo/remaining-limitations.md`.

## Module gameplay hiện có

- Đăng ký/đăng nhập, Google OAuth, hồ sơ, avatar, referral.
- Tu luyện phàm nhân, phá vách, dùng Xá Lợi Cổ, thăng tiên, độ kiếp.
- Không khiếu/tiên khiếu, đạo ngân, tài nguyên node, dị nhân và tai kiếp.
- Khám phá bản đồ realtime, PvE/PvP combat, đội hình 3x3, ground drop và retreat/flee.
- Inventory Cổ/vật phẩm/nguyên liệu, cho ăn, trang bị, bản mệnh, tiêu hao/hỗ trợ.
- Luyện Cổ, suy diễn công thức, sát chiêu.
- Shop NPC, Bảo Hoàng Thiên marketplace/auction/trade listing, P2P trade.
- Môn phái, nhiệm vụ, kho, đơn xin gia nhập/thăng chức, mỏ môn phái.
- Party, võ đài và shop võ đài, Đấu Trường Sinh Tử, Nguyên Tuyền, chat/mail, phó bản, hồn phách.
- Gacha, thành tựu/danh hiệu/khung, bảng xếp hạng, cược linh thú, thông báo, gift code, lịch sử chiến đấu và ký sự hoạt động.
- Ví/nạp tiền, premium shop, feedback và trang quản trị dữ liệu/giám sát.

## Nguyên tắc cập nhật tài liệu

1. Nếu thay đổi route backend, cập nhật `docs/API.md`.
2. Nếu thay đổi schema/seed CSV, cập nhật `docs/GAME_DATA.md`.
3. Nếu thay đổi flow chạy app, biến môi trường hoặc lệnh kiểm tra, cập nhật `docs/DEVELOPMENT.md` và README liên quan.
4. Nếu thay đổi gameplay nhìn thấy được, cập nhật `WIKI.md`.
