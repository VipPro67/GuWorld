# GuWorld: trạng thái codebase

Tài liệu này là inventory ngắn về ứng dụng hiện có, căn cứ vào route backend và view frontend. Dữ liệu/flag runtime có thể làm thay đổi nội dung hoặc quyền truy cập trong từng môi trường.

## Sản phẩm

GuWorld là game RPG web về tu luyện và Cổ trùng. Backend Go/Gin xử lý luật game, persistence và combat theo lượt. Frontend Next.js cung cấp lobby, bản đồ và các màn hình domain. PostgreSQL lưu trạng thái; Redis hỗ trợ cache/realtime và có fallback một phần.

## Thành phần

| Thành phần | Vị trí | Vai trò |
| --- | --- | --- |
| Backend entrypoint | `backend/cmd/server/main.go` | Env, DB/schema, registry, workers, router và realtime server |
| Router | `backend/internal/server/server.go` | Route `/api`, auth/rate limit, permission admin |
| Gameplay handlers | `backend/internal/handlers/` | Điều phối HTTP và đa số luật gameplay |
| Shared services | `backend/internal/services/` | Player, inventory, quest, recipe, achievement |
| Player projection | `backend/internal/playerstate/` | Snapshot/projection player và collection revision |
| Turn based simulator | `backend/internal/simulator/` | Combat deterministic theo lượt |
| Schema | `backend/internal/db/schema_*.go` | DDL idempotent, compatibility changes và backfill |
| Frontend | `frontend/src/app`, `components`, `store` | App Router, UI gameplay và Zustand state/API client |
| Game data | `backend/data/` | CSV dataset quản lý qua Admin |

## Feature inventory

- Tài khoản, onboarding/tutorial, hồ sơ, avatar, referral, gift code, notification, audit log, wallet/top-up.
- Tu luyện, phá vách, Thăng Tiên, độ kiếp, nhiệm vụ/ngày, thành tựu và leaderboard.
- Inventory Gu/item, trạng thái/độ bền Gu, luyện hóa, Cổ phương, phiên luyện Cổ, Sát Chiêu và Tiên Cổ Ốc.
- Bản đồ theo vùng, movement/portal/teleport, quái, loot trên map, scouting/divination.
- Combat turn based, session PvP, capture, replay/lịch sử, party và practice.
- Sủng vật và canonical actor roster.
- Tiên khiếu: node/điểm tài nguyên, dị nhân, refinement queue, settlements, NPC, Gu swarms, portal/guest.
- Instance, Mộng Cảnh, Bí Cảnh Không Gian và truyền thừa.
- Môn phái, kho/nhiệm vụ/lãnh địa; shop/merchant, marketplace/auction/buyback, P2P trade; chat/mail.
- Gacha, cược linh thú, Võ Đài, Tháp Chiến Thuật, Đấu Trường Sinh Tử, Đại Hội Võ Thuật và feedback.
- Admin: RBAC, player/domain operations, data import/export, moderation, audit và monitoring.

Danh sách này không khẳng định mọi dataset đều đã được import hoặc mọi flow luôn mở cho mọi nhân vật. Điều kiện theo rank, vị trí, quyền, trạng thái và cấu hình nằm ở server.

## Runtime notes

- PostgreSQL DDL được khởi tạo qua `InitSchema()`; không có migration framework versioned.
- CSV được import qua Admin. Registry được load trong startup và một số luồng import có reload cụ thể.
- Redis được thử kết nối có giới hạn; khi lỗi, backend có fallback bộ nhớ cho một số chức năng nhưng không đảm bảo đủ realtime/cache.
- Worker là polling goroutine có context cancellation; recurring loop dùng `internal/worker/runner.go`.
- WebSocket/WebTransport phục vụ đồng bộ world/player/chat/trade. Combat simulation dùng API/session theo lượt.

## Tài liệu chi tiết

- [README và chạy nhanh](README.md)
- [Wiki gameplay](WIKI.md)
- [Chi tiết từng tính năng và source map](docs/guides/FEATURES.md)
- [API](docs/guides/API.md)
- [Phát triển và cấu hình](docs/guides/DEVELOPMENT.md)
- [Dữ liệu game](docs/guides/GAME_DATA.md)
- [Kiến trúc](docs/architecture/ARCHITECTURE.md)
- [Miền player](docs/architecture/PLAYER_DOMAIN.md)
