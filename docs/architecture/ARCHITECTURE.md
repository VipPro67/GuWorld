# Kiến trúc GuWorld

## Tổng quan

GuWorld gồm backend Go/Gin, PostgreSQL, Redis và frontend Next.js. Backend phục vụ HTTP API; khi bật TLS, cùng process còn cung cấp WebSocket và WebTransport cho đồng bộ thế giới. Gameplay combat được mô phỏng theo lượt ở backend.

```text
Next.js UI + Zustand store
        │ HTTP API / WebSocket / WebTransport
        ▼
Gin router ── handlers ── services (một số domain)
                    │
          PostgreSQL + registries/cache
                    │
             worker polling loops
```

## Backend

- `cmd/server/main.go` nạp `.env`, kết nối DB/Redis, gọi `InitSchema`, nạp registry, khởi tạo service, worker, router và realtime server.
- `internal/server/server.go` đăng ký route `/api`, middleware xác thực/rate limit và quyền admin.
- `internal/handlers/` điều phối HTTP và chứa phần lớn luật gameplay.
- `internal/services/` hiện có `PlayerService`, `InventoryService`, `QuestService`, `RecipeService` và `AchievementService`. Chỉ dùng service khi domain đó thực sự được sở hữu ở đây.
- `internal/playerstate/` cung cấp projection chuẩn của hàng `players`, revision của bộ sưu tập và sửa chữa lười một số trạng thái.
- `internal/simulator/` xử lý combat theo lượt; `handlers/combat_*.go` điều phối session, lưu kết quả, loot và lịch sử.
- `internal/db/schema_*.go` chứa DDL theo domain. `InitSchema()` khởi tạo theo thứ tự phụ thuộc, tương thích DB đã tồn tại và seed dữ liệu runtime tối thiểu; dự án không dùng migration framework có version.
- `internal/worker/runner.go` chuẩn hóa ticker loop qua `RunEvery`/`SafeRun` để hỗ trợ cancel và recover panic ở từng tick.
- `internal/cache/` kết nối Redis. Khi Redis không sẵn sàng sau số lần retry giới hạn, server có thể chạy với fallback trong bộ nhớ; một số chức năng realtime/cache sẽ bị giới hạn.

## Frontend

- `frontend/src/app/` dùng Next.js App Router.
- `frontend/src/store/useGameStore.ts` hợp nhất các Zustand slice. Store cũng gọi API và cập nhật state; không có service/API layer riêng.
- `frontend/src/store/slices/` chia state theo domain như auth, world, inventory, combat, aperture, chat, trade, sect và arena.
- `frontend/src/lib/auth.ts`, `config.ts`, `fetch.ts` cung cấp helper dùng chung. `authFetch` xử lý session hết hạn tập trung.
- `frontend/src/components/` chứa các view gameplay, lobby, admin, wiki và overlay realtime.

## Persistence và cache

PostgreSQL là nguồn dữ liệu bền vững. CSV trong `backend/data/` được import qua Admin → **Dữ liệu**; không tự động import toàn bộ khi khởi động. Nhiều dictionary được nạp thành registry trong bộ nhớ lúc startup. Endpoint hoặc luồng import có thể reload một số registry; nếu thay file CSV trực tiếp thì cần import và khởi động lại khi registry tương ứng không được reload.

Redis dùng cho cache, pub/sub và một số phối hợp realtime. Rate limiter có fallback trong bộ nhớ. Không coi Redis là nguồn dữ liệu gameplay chính.

## Realtime và combat

- Combat thường, PvP và các flow có session được giải quyết bằng simulator theo lượt.
- `/api/wt` là WebTransport endpoint khi được bật; `/api/ws` là WebSocket fallback theo cấu hình.
- Realtime transport phục vụ di chuyển/vị trí thế giới, chat, notification và player events. Nó không chạy combat simulation.
- Client player events có cơ chế reconnect; WebTransport cần HTTPS và certificate hợp lệ.

## Ranh giới khi thay đổi

- Route: cập nhật `docs/guides/API.md`.
- Player persistence hoặc state nhìn thấy bởi player: đọc [PLAYER_DOMAIN.md](PLAYER_DOMAIN.md) trước.
- Schema/CSV: đọc [GAME_DATA.md](../guides/GAME_DATA.md) và sửa schema domain tương ứng.
- Gameplay người chơi nhìn thấy: cập nhật [WIKI.md](../../WIKI.md).
- Startup, env hoặc lệnh chạy: cập nhật [DEVELOPMENT.md](../guides/DEVELOPMENT.md) và README.
