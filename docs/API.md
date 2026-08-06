# Backend API Map

Base path: `/api`.

Các route protected yêu cầu cookie/JWT hợp lệ qua `middleware.AuthRequired()` trừ khi ghi public.

## Debug endpoints (ngoài `/api`)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/debug/pprof/goroutine` | Goroutine dump (chẩn đoán request treo). |
| GET | `/debug/pprof/goroutine/stack` | Toàn bộ goroutine stack dạng text. |
| GET | `/debug/pprof/heap` | Heap profile. |
| GET | `/debug/pprof/allocs` | Allocation profile. |
| GET | `/debug/pprof/block` | Block profile (bật `runtime.SetBlockProfileRate` để có data). |
| GET | `/debug/pprof/mutex` | Mutex profile. |
| GET | `/debug/db-pool` | DB connection pool stats (open/inUse/idle/wait). |

> **Log request:** mọi request (trừ OPTIONS) được log `REQ START` khi bắt đầu và `REQ END` khi kết thúc kèm duration. Request > 5s được đánh dấu `REQ SLOW` — dùng để phát hiện handler đang treo mà log cũ không thấy (Gin chỉ log khi request hoàn thành).

> **DB hardening:** connection string mặc định kèm `statement_timeout=15000ms`, `lock_timeout=10000ms` (override qua `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS`). Query bị lock/treo sẽ tự abort thay vì treo vô hạn.

> **SEO routes:** `robots.txt`, `sitemap.xml`, `/wiki`, `/wiki/[guId]` do Next.js frontend xử lý (SSG/ISR). Xem `frontend/src/app/`.

## Public / auth / payment entrypoints

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/dictionary` | Public Gu + item + monster dictionary. |
| GET | `/recipes` | Public refining recipes. |
| POST | `/auth/send-otp` | Send OTP via email to register. |
| POST | `/auth/register` | Rate limited. Requires valid OTP. |
| POST | `/auth/login` | Rate limited. |
| POST | `/auth/logout` | Public logout endpoint. |
| GET | `/auth/me` | Current player, protected. |
| GET | `/auth/google/login` | Google OAuth start, rate limited. |
| GET | `/auth/google/callback` | Google OAuth callback, rate limited. |
| GET | `/topup/packages` | Public topup packages. |
| POST | `/webhooks/payos` | PayOS webhook. |
| GET | `/admin/topup-orders` | Protected admin listing. |
| GET | `/admin/wallet-transactions` | Protected admin listing. |
| GET | `/wallet` | Protected wallet. |
| POST | `/topup/orders` | Protected create order, rate limited. |
| GET | `/topup/orders/:orderCode` | Protected order detail. |

## Player

| Method | Path |
| --- | --- |
| GET | `/player/events` |
| GET | `/player/status/:id` | Projects only equipped/vital Gus. |
| GET | `/player/inventory/gus` | Full Gu inventory (lazy-loaded). |
| GET | `/player/stats` |
| GET | `/player/passive-stats` | Detailed passive stats breakdown with per-source attribution. |
| POST | `/player/explore/:id` |
| POST | `/player/use-item` |
| POST | `/player/use-items` |
| POST | `/player/recharge` |
| POST | `/player/clear-stress` |
| GET | `/player/profile/:username` |
| PUT | `/player/profile/settings` |
| GET | `/player/referrals` |
| POST | `/player/simulate-deposit` | Admin only. |
| POST | `/player/avatar` |
| POST | `/player/tick-regen` |
| POST | `/player/sacrifice` |

## Cultivation and quests

| Method | Path |
| --- | --- |
| POST | `/cultivate/break-wall` |
| POST | `/cultivate/use-relic-gu` |
| POST | `/cultivate/ascension` |
| POST | `/cultivate/tribulation` |
| POST | `/quest/complete` |

## Inventory, refining and killer moves

| Method | Path |
| --- | --- |
| POST | `/inventory/feed` |
| POST | `/inventory/feed-group` |
| POST | `/inventory/equip` |
| POST | `/inventory/status` |
| POST | `/inventory/set-vital` |
| POST | `/inventory/use-gu` |
| POST | `/inventory/use-support` |
| POST | `/inventory/drop` |
| POST | `/refine` |
| POST | `/refine/deduce` |
| POST | `/killer-moves/calculate` |
| POST | `/killer-moves` |
| GET | `/killer-moves` |
| GET | `/killer-moves/:id` |
| DELETE | `/killer-moves/:id` |
| POST | `/combat/cast-killer-move` |

## Exploration and combat

| Method | Path | Ghi chú |
| --- | --- | --- |
| POST | `/combat/attack` | |
| POST | `/combat/capture` | Bắt quái vật làm sủng vật. |
| POST | `/explore` | |
| POST | `/explore/retreat/:id` | |
| GET | `/explore/grid` | |
| POST | `/explore/move` | |

## Sủng vật (Pets)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/pets` | Danh sách sủng vật của người chơi. |
| POST | `/pets/feed` | Cho sủng vật ăn. Body: `{"pet_id": "...", "item_id": "..."}`. |
| POST | `/pets/formation` | Đặt vị trí trong đội hình. Body: `{"pet_id": "...", "slot": 1}`. Slot 0 = storage. |
| POST | `/pets/sleep` | Chuyển sủng vật active ↔ sleeping. Body: `{"pet_id": "..."}`. |
| POST | `/pets/release` | Thả sủng vật. Body: `{"pet_id": "..."}`. |
| POST | `/pets/merge` | Gộp sủng vật cùng loại. Body: `{"source_ids": ["..."], "target_id": "..."}`. |

## Party, arena, mine

| Method | Path |
| --- | --- |
| POST | `/party/create` |
| GET | `/party/status` |
| POST | `/party/invite` |
| POST | `/party/leave` |
| GET | `/arena/leaderboard` |
| POST | `/arena/challenge` |
| GET | `/leaderboard/:category` | BXH tổng hợp. category: `rank`, `primeval_stones`, `gu_strongest`, `immortal_stones`. Trả về top 50, kèm rank người gọi |
| POST | `/leaderboard/toggle` | Bật/tắt hiển thị trên BXH |
| GET | `/leaderboard/settings` | Trạng thái `show_on_leaderboard` hiện tại |
| GET | `/mine/slots` |
| POST | `/mine/occupy` |
| POST | `/mine/claim` |
| POST | `/mine/abandon` |
| POST | `/mine/attack` |

## Immortal aperture, dao marks and tribulations

| Method | Path |
| --- | --- |
| GET | `/immortal-aperture` |
| GET | `/immortal-aperture/nodes` |
| POST | `/immortal-aperture/nodes` |
| POST | `/immortal-aperture/nodes/:id/collect` |
| POST | `/immortal-aperture/nodes/:id/uninstall` |
| DELETE | `/immortal-aperture/nodes/:id` |
| POST | `/immortal-aperture/time-rate` |
| POST | `/immortal-aperture/labor` |
| POST | `/immortal-aperture/nodes/:id/repair` |
| GET | `/immortal-aperture/variant-humans` |
| GET | `/dao-marks` |
| GET | `/dao-marks/dao-ngan` | Đạo ngân (composite dao power) calculation. |
| GET | `/tribulations/active` |
| POST | `/tribulations/:eventId/attack` |
| POST | `/tribulations/:eventId/resolve` |

## Shop, marketplace and P2P trade

| Method | Path |
| --- | --- |
| GET | `/shop/packages` |
| GET | `/shop/list` |
| POST | `/shop/buy-package` |
| POST | `/shop/buy` |
| POST | `/shop/sell` |
| GET | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/buy` |
| POST | `/shop/bao-hoang/bid` |
| POST | `/shop/bao-hoang/cancel` |
| POST | `/shop/bao-hoang/settle` |
| POST | `/trade/initiate` |
| GET | `/trade/active` |
| GET | `/trade/events` |
| POST | `/trade/accept/:id` |
| POST | `/trade/decline/:id` |
| POST | `/trade/cancel/:id` |
| POST | `/trade/offer/:id` |
| POST | `/trade/lock/:id` |
| POST | `/trade/confirm/:id` |

## Chat/mail, soul and instances

| Method | Path |
| --- | --- |
| POST | `/chat/mail/send` |
| GET | `/chat/mail/inbox` |
| POST | `/chat/bao-hoang/send` |
| GET | `/chat/bao-hoang/messages` |
| GET | `/chat/bao-hoang/events` |
| POST | `/soul/use-guts` |
| POST | `/soul/refine` |
| POST | `/soul/calm` |
| GET | `/instance/active` |
| POST | `/instance/enter` |
| POST | `/instance/action` |

## Gacha (Cổ Từ / Mở Đá)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/gacha/pools` | Danh sách gói mở. |
| POST | `/gacha/open` | Mở N lần. Body: `{"pool_id": "...", "quantity": 10}` (`quantity` mặc định 1, tối đa 100; ≥ 10 được giảm 10% chi phí). |
| POST | `/gacha/open10` | Tương đương `/gacha/open` với `quantity: 10` (giữ để tương thích). |

## Sects

| Method | Path |
| --- | --- |
| GET | `/sects` |
| POST | `/sects/:id/join` |
| POST | `/sects/join-request/cancel` |
| POST | `/sects/leave` |
| GET | `/sects/me` |
| POST | `/sects/promote` |
| POST | `/sects/donate/stones` |
| POST | `/sects/donate/materials` |
| POST | `/sects/donate/gu` |
| POST | `/sects/donate/bulk` | Quyên góp hàng loạt linh tài và cổ trùng. Body: `{"materials": [{"material_id": "...", "amount": 5}], "gus": [{"gu_instance_id": "..."}]}`. |
| POST | `/sects/vault/buy` | Đổi vật phẩm từ bảo khố. Body: `{"item_type": "gu|material|recipe", "item_id": "...", "quantity": 5}` (`quantity` mặc định 1; recipe chỉ nhận 1). |
| GET | `/sects/missions` |
| POST | `/sects/missions/:missionId/accept` |
| POST | `/sects/missions/:missionId/complete` |
| POST | `/sects/create` |
| GET | `/sects/me/members` |
| POST | `/sects/members/:memberId/rank` |
| POST | `/sects/members/:memberId/kick` |
| POST | `/sects/missions/create` |
| POST | `/sects/vault/stock` |
| POST | `/sects/upgrade` |
| POST | `/sects/toggle-approval` |
| GET | `/sects/join-requests` |
| POST | `/sects/join-requests/:requestId/action` |
| GET | `/sects/promotion-requests` |
| POST | `/sects/promotion-requests/:requestId/action` |
| GET | `/sects/mines` |
| POST | `/sects/mines/claim` |
| POST | `/sects/mines/attack` |

## Feedback

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/feedback` | Public list/filter. |
| POST | `/feedback` | Protected submit. |
| POST | `/feedback/:id/vote` | Protected vote (chỉ khi status = `active`, rank ≥ 2). |
| PUT | `/feedback/:id/status` | Protected status update (admin). |
| GET | `/feedback/:id/comments` | Public list góp ý của topic. |
| POST | `/feedback/:id/comments` | Protected góp ý (rank ≥ 2; status phải `active`/`in_progress`/`completed`). |
| DELETE | `/feedback/:id/comments/:commentId` | Protected xóa góp ý (tác giả hoặc admin). |

## Admin — players & gus

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/players` | List người chơi, phân trang (`page`, `limit`). Search: `search_by=all|id|username|email|item|gu`. Khi `search_by=all` auto-detect UUID. `item`/`gu` match theo registry ID + tên (trả `gu_ids`/`item_ids` mới hơn bản ghi cũ). |
| GET | `/admin/players/online` | List người chơi online. |
| GET | `/admin/players/:id` | Chi tiết người chơi. Response bỏ `gu_inventory`/`materials_inventory`, trả `gu_count` + `item_count`. |
| GET | `/admin/players/:id/items` | Lazy-load item của người chơi từ `player_inventory_items`. Trả thêm các cổ phương đã mở khóa (`category=recipe`) từ `player_recipe_progress`. Params: `page`, `limit`, `q` (match item_id/item_name theo registry). |
| GET | `/admin/players/:id/gus` | Lazy-load gu trong `players.gu_inventory` (JSONB). Params: `page`, `limit`, `q` (match gu_id/instance_id theo registry). |
| PUT | `/admin/players/:id` | Sửa thông tin người chơi. Body là object chứa 1+ field (chỉ gửi field thay đổi): `username`, `email`, `password`, `bio`, `avatar_url`, `rank`, `sub_rank`, `cultivation`, `realm`, `hp`, `max_hp`, `mana`, `max_mana`, `mortal_stones`, `immortal_stones`, `primeval_stones`, `soul`, `hp_regen_rate`, `mana_regen_rate`, `is_admin`, `is_banned`, `is_online`. Trả list field đã cập nhật. |
| GET | `/admin/gus/search` | Tìm cổ trùng trên toàn server. Params: `gu_id` và/hoặc `instance_id` (match registry ID + tên, ILIKE), `page`, `limit`. Quét 3 nguồn: `players.gu_inventory`, `auction` (`item_type='Gu'`), `caravan_shop_items` + `arena_shop_items`. `location`: `inventory`/`auction`/`shop`. |
| POST | `/admin/players/:id/items` | Thêm item hoặc cổ phương (Công thức) cho người chơi. Body: `{item_id, quantity}`. `item_id` là id item trong `Item.csv` hoặc id recipe trong `Recipe.csv`. Nếu là recipe → mở khóa (unlock) Cổ phương cho người chơi (`player_recipe_progress`, `completion_rate=100`). Trả `kind` = `item`/`recipe`. |
| DELETE | `/admin/players/:id/items` | Xóa item hoặc cổ phương khỏi người chơi. Body: `{item_id, quantity}`. Nếu là recipe → thu hồi (delete) khỏi `player_recipe_progress`. Trả `kind` = `item`/`recipe`. |
| POST | `/admin/players/:id/gus` | Thêm gu cho người chơi. |
| DELETE | `/admin/players/:id/gus/:instance_id` | Xóa gu theo instance_id. |
| GET | `/admin/players/:id/pets` | Lazy-load sủng vật của người chơi. Params: `page`, `limit`, `q` (search monster name/ID). |
| DELETE | `/admin/players/:id/pets/:pet_id` | Xóa vĩnh viễn sủng vật. |
| PUT | `/admin/players/:id/stones` | Đổi đá (stones) của người chơi. |
| GET | `/admin/audit-logs` | Audit log (bao gồm `admin.update_player`). |
