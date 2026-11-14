# Backend API

Base path: `/api`. Route được đăng ký trong `backend/internal/server/server.go`; handler/domain validation có thể bổ sung điều kiện ngoài các ghi chú dưới đây. Đây là bản đồ route theo nhóm, không thay thế contract payload của handler.

## Xác thực chung

- Route public không yêu cầu session trừ khi ghi chú riêng.
- Các nhóm gameplay thường yêu cầu `AuthRequired` và `RequireOnboarded`; nhóm `/player` cũng áp rate limit theo player.
- Admin yêu cầu session và permission cụ thể cho từng route. Một số mutation yêu cầu lý do, xác nhận, password re-auth hoặc idempotency key.
- Session dùng cookie/JWT theo cấu hình backend. Frontend gọi qua `authFetch` để áp dụng chính sách hết hạn tập trung.

## Health, dictionary và auth

| Method | Path | Truy cập |
| --- | --- | --- |
| GET | `/health` | Public; phiên bản build/protocol và uptime |
| GET | `/dictionary`, `/dictionary/{gus,items,monsters,constitutions,races,statuses,loot-tables,regions,monster-spawns,region-loot}` | Public; rate limited |
| GET | `/recipes` | Public |
| GET | `/gus/:instance_id/history` | Public; rate limited |
| GET | `/auth/config` | Public |
| POST | `/auth/send-otp`, `/auth/register`, `/auth/login` | Public; rate limited |
| POST | `/auth/logout` | Public |
| GET | `/auth/google/login`, `/auth/google/callback` | Public; OAuth cấu hình tùy chọn |
| GET | `/auth/me` | Authenticated |
| POST | `/auth/claim-tab` | Authenticated; claim gameplay tab |

## Player, ví và thanh toán

`/player` yêu cầu đăng nhập và onboarding. Các route:

- GET `/player/respawn-choice`, `/player/status/:id`, `/player/inventory/gus`, `/player/stats`, `/player/passive-stats`, `/player/profile/:username`, `/player/referrals`, `/player/combat-settings`, `/player/audit-logs`, `/player/notifications`, `/player/notifications/unread-count`, `/player/beast-betting/current`, `/player/beast-betting/history`.
- POST `/player/respawn-choice`, `/player/use-item`, `/player/use-items`, `/player/recharge`, `/player/clear-stress`, `/player/referrals/claim`, `/player/referrals/bind`, `/player/avatar`, `/player/redeem-gift-code`, `/player/complete-tutorial`, `/player/complete-onboarding`, `/player/notifications/read`, `/player/notifications/read-all`, `/player/beast-betting/bet`.
- PUT `/player/profile/settings`, `/player/combat-settings`.
- GET `/wallet`; thanh toán dùng GET `/topup/packages`, POST `/topup/orders`, GET `/topup/orders/:orderCode`, POST `/webhooks/payos`.
- GET `/shop/packages` trả gói premium; POST mua gói nằm tại `/shop/buy-package`.

## Gameplay: cultivation, inventory và combat

| Prefix | Route chính |
| --- | --- |
| `/cultivate` | POST `/break-wall`, `/use-relic-gu`; GET `/ascension/session`; POST `/ascension/shatter`, `/ascension/temper`, `/ascension/calamity/engage`, `/ascension/release` |
| `/quest` | GET `/`; POST `/complete` |
| `/achievements` | GET `/`; POST `/claim`, `/select` |
| `/daily-missions` | GET `/`; POST `/claim` |
| `/inventory` | POST `/feed`, `/feed-group`, `/feed-all`, `/equip`, `/subjugate-gu`, `/nourish-gu`, `/repair-gu`, `/unequip-all`, `/status`, `/set-vital`, `/use-gu`, `/use-support`, `/drop`, `/drop-batch` |
| `/refine` | POST `/`, `/deduce`, `/conceive`; GET `/conceive/candidates`, `/preview`, `/sessions/active`, `/sessions/:id`; POST `/sessions`, `/sessions/:id/steps`, `/sessions/:id/abort`, `/sessions/:id/ack` |
| `/combat` | POST `/start`, `/session/start`, `/session/:id/command`, `/flee/:id`, `/capture`, `/practice`, `/practice/exit`; GET `/session`, `/session/:id`, `/history`, `/history/:id` |
| `/killer-moves` | GET `/options`, `/`; POST `/preview`, `/`, `/:id/equip`, `/:id/use`, `/:id/repair`; DELETE `/:id` |
| `/gu-houses` | GET `/`; POST `/preview`, `/`, `/:id/equip`, `/:id/use`, `/:id/repair` |
| `/pets` | GET `/`; POST `/feed`, `/formation`, `/sleep`, `/release`, `/merge` |
| `/actors` | GET `/`, `/:id`; chỉ đọc roster actor của player hiện tại |

Combat được xử lý theo lượt. `/combat/start` chạy tự động; `/combat/session/*` dành cho session command có revision. `/combat/retreat/:id` cho phép rút lui khỏi encounter trước khi session bắt đầu; PvP không hỗ trợ rút lui ở bước chuẩn bị. `/combat/flee/:id` giữ tương thích; khi session hoạt động, flee phải gửi như command trong lượt.

## Thế giới, aperture và instance

| Prefix | Route chính |
| --- | --- |
| `/explore` | GET `/grid`, `/regions`, `/cells/rect`; POST `/move` |
| `/world` | GET `/map-objects`, `/monsters`, `/drops`; POST `/portal/transfer`, `/drops/:id/loot` |
| `/teleport` | GET `/templates`, `/effect-options`, `/mine`, `/directory`; POST `/preview`, `/build`, `/travel`, `/manage`, `/repair`, `/withdraw`, `/dismantle` |
| `/immortal-aperture` | GET aperture, nodes, points, world-points, workers, portal, guests, swarms, refinement queue, settlements, NPCs; POST/DELETE quản lý nodes, resource points/workers, time rate/labor, portals/guest, Gu swarms, refinement, settlements, NPCs |
| `/dao-marks` | GET `/`, `/dao-ngan` |
| `/tribulations` | GET `/active`; POST `/:eventId/engage`, `/:eventId/resolve` |
| `/instance` | GET `/active`, `/list`, `/spatial`, `/spatial/active`; POST `/enter`, `/action`, `/spatial/enter`, `/spatial/exit` |
| `/spatial-realms` | GET `/`, `/active`, `/altar-info`; POST `/enter`, `/exit`, `/claim-altar` |
| `/dream-realms` | GET `/`, `/active`; POST `/enter`, `/action`, `/exit` |
| `/veins` | POST `/:id/harvest` |
| `/scouting` | POST `/inspect`, `/divine`, `/divine-gu`, `/apply-mark`, `/sever-clues`, `/resonance-scan`; GET `/tracked-targets` |

## Social, economy và thi đấu

- `/party`: POST `/create`, `/invite`, `/leave`, `/invitations/:id/respond`, `/formation`; GET `/status`, `/invitations`.
- `/chat`: mail private và chat Bảo Hoàng ở `/mail/*`, `/bao-hoang/*`.
- `/shop`: GET `/list`; POST `/buy`, `/sell`, `/sell-bulk`, `/exchange`, `/buy-package`; marketplace `/bao-hoang/listings`, `/buy`, `/bid`, `/cancel`, `/settle`, `/buyback/*`, `/trade-offers/*`.
- `/merchant`: GET `/`; POST `/quote`, `/sell`, `/buy`.
- `/trade`: POST `/initiate`, `/accept/:id`, `/decline/:id`, `/cancel/:id`, `/offer/:id`, `/lock/:id`, `/unlock/:id`, `/confirm/:id`; GET `/active`.
- `/sects`: đọc/join/leave/create, rank, donation, vault, missions, membership requests, territory, defense/attack, building và relocation. Danh sách đầy đủ ở router `server.go`.
- `/arena`: GET `/leaderboard`, `/shop`; POST `/challenge`, `/shop/buy`.
- `/infinite-tower`: GET `/state`, `/leaderboard`; POST `/challenge`, `/abandon`.
- `/death-arena`, `/martial-arena`: tournament, bracket, matches/log, history, rewards và claim reward.
- `/leaderboard/:category`, settings/visibility; `/gacha/pools`, `/gacha/open`, `/gacha/open10`; `/soul/refine`; feedback đọc public, submit/vote/comment cần auth.

## Admin

Admin UI dùng `/admin/*`. Mọi route yêu cầu `AuthRequired`, `admin` rate limit và permission tương ứng. Route inventory là `adminRouteSpecs()` trong `server.go`; các nhóm bao gồm access/resources, monitor/traces, billing, combat history/rankings, player/inventory, aperture, Gu, sect, data import/export, cache, audit, regions/map cells/drops, shop, arena shop, monsters, pets, moderation và role management. Các thao tác ghi được audit; mutation nguy hiểm có thể yêu cầu reason/confirmation/re-auth/idempotency.

Dataset API: GET `/admin/data/tables`; POST `/admin/data/import`, `/admin/data/import/commit`, `/admin/data/import/:table`, `/admin/data/import/:table/commit`, `/admin/data/export`, `/admin/data/export/:table`.

## Realtime

- WebTransport: `CONNECT /api/wt`, xử lý ngoài Gin khi bật.
- WebSocket fallback: `/api/ws`, tùy `WS_FALLBACK_ENABLED`.
- `POST /api/realtime/metrics` nhận telemetry client có xác thực.
- Transport phát world/player/chat/trade synchronization. Combat simulation vẫn dùng HTTP combat session API.

Rate limits, role requirements và payload validation có thể thay đổi theo middleware/handler; kiểm tra source trước khi xây client contract mới.
