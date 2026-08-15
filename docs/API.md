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
| GET | `/dictionary` | Public Gu + item + monster + constitution + race dictionary. Response: `{gu, items, monsters, constitutions, races}` — `constitutions` và `races` là mảng `{key, name, effects}` (effects là JSON theo format trong `docs/GAME_DATA.md`). |
| GET | `/recipes` | Public refining recipes. Thêm query param ?all=true để lấy toàn bộ danh sách thay vì chỉ những công thức đã mở khóa của người chơi. |
| POST | `/auth/send-otp` | Send OTP via email to register. |
| POST | `/auth/register` | Rate limited. Requires valid OTP. Body thêm `race` (`human`/`fur_people`) và `gender` (`male`/`female`), mặc định `human`/`male`. |
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
| POST | `/refine` | Luyện Cổ. Recipe `required_gus` là map `gu_id → {rank, qty}`; Cổ nguyên liệu phải khớp `gu_id` và đúng `rank` (khi `rank != 0`). |
| POST | `/refine/deduce` | Suy diễn recipe. |
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
| POST | `/explore` | Du hành tới zone: body `{"player_id","node_id"}` (node_id = id Đại Vực / địa danh / `Southern_Border`). |
| POST | `/explore/retreat/:id` | |
| GET | `/explore/grid` | Viewport 5×5 quanh player; trả thêm `world_width`, `world_height`. Bỏ `explored_cells`. |
| GET | `/explore/regions` | Danh sách zone (id, name, geometry `min_x..max_y`, `entry_x/y`, `parent_id`, `zone_type`). `zone_type`: `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC` — khi chồng lấn, ưu tiên `HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC`. |
| GET | `/explore/world` | Cấu hình thế giới: `width`, `height`, `large_zone_size`, `sub_zone_size`, `hub_region_id`, `hub_x/y`. |
| GET | `/explore/objects` | Các ô persistent (map_cells): village/trạm nghỉ, boss, linh tài — cho bản đồ lớn. Trả `{region_id,x,y,cell_type,content_id,rank,respawn_seconds,cleared}`. |
| GET | `/explore/cells` | Các ô non-empty (Monster/Boss/Resource/WildGu) quanh player (`?x=&y=&r=`, mặc định quanh player, r cap 300) để bản đồ lớn vẽ marker quái/tài nguyên. Nội dung **ổn định theo ô** (không re-randomize theo thời gian), khớp đúng nội dung khi di chuyển. Trả `[{x,y,cell_type,content,rank,sub_rank}]`. |
| POST | `/explore/move` | Di chuyển tới toạ độ trong thế giới (bounds = `world_width`/`world_height`). |
| POST | `/explore/move-path` | Di chuyển tự do theo đường thẳng từ ô hiện tại tới toạ độ đích, trigger cell action trên từng ô đi qua (gom loot; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss**, không có phục kích ngẫu nhiên trên ô trống). Trả `{player, logs, log, combat_triggered, reached, stats}`; nếu `reached=false` và không combat, client chain tiếp request từ vị trí mới cho tới khi tới đích. |

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
| POST | `/trade/offer/:id` | Body có thể chứa `recipes: [{"recipe_id": "..."}]` cùng với stones/gus/materials. |
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
| GET | `/instance/list` | Danh sách phó bản + `cooldown_remaining` (giây) của player |
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
| POST | `/sects/donate/recipe` | Quyên góp một Bí phương đã học. Body: `{"recipe_id": "..."}`. Nhận điểm cống hiến theo hạng Cổ trùng mục tiêu.
| POST | `/sects/donate/bulk` | Quyên góp hàng loạt linh tài, cổ trùng và bí phương. Body: `{"materials": [{"material_id": "...", "amount": 5}], "gus": [{"gu_instance_id": "..."}], "recipes": [{"recipe_id": "..."}]}`.
| POST | `/sects/vault/buy` | Đổi vật phẩm từ bảo khố. Body: `{"item_type": "gu|material|recipe", "item_id": "...", "quantity": 5}` (`quantity` mặc định 1; recipe chỉ nhận 1).
| GET | `/sects/missions` | Danh sách nhiệm vụ trên bảng (đã lọc nhiệm vụ hết hạn). Trả về `expires_at` mỗi nhiệm vụ. |
| POST | `/sects/missions/:missionId/accept` | Nhận nhiệm vụ. Copy `expires_at` sang nhiệm vụ cá nhân. |
| POST | `/sects/missions/:missionId/complete` | Nộp nhiệm vụ. |
| DELETE | `/sects/missions/:missionId` | Trưởng Lão/Chưởng Môn xóa nhiệm vụ trên bảng (xóa theo cascade nhiệm vụ đệ tử đang làm). |
| DELETE | `/sects/missions/:missionId/abandon` | Đệ tử hủy nhiệm vụ đang thực hiện của chính mình. |
| POST | `/sects/create` |
| GET | `/sects/me/members` | Danh sách đệ tử trong tông môn (thêm `power` — sức mạnh phòng thủ/tấn công của mỗi đệ tử). |
| POST | `/sects/members/:memberId/rank` |
| POST | `/sects/members/:memberId/kick` |
| POST | `/sects/missions/create` | Trưởng Lão/Chưởng Môn phát hành nhiệm vụ. Body: `{"title","description","mission_type","target_id","target_amount","reward_contribution","required_member_rank","duration_hours"}`. `duration_hours`: 0 (hoặc bỏ trống) = mặc định 24h; `-1` = không hết hạn; `>0` = số giờ có hiệu lực. |
| POST | `/sects/vault/stock` | Nhập kho 1 loại vật phẩm. Body: `{"item_type": "gu|material|recipe", "item_id": "...", "quantity": 5}`. |
| POST | `/sects/vault/stock/bulk` | Nhập kho hàng loạt (1 giao dịch). Body: `{"items": [{"item_type": "gu|material|recipe", "item_id": "...", "quantity": 5}]}`. |
| POST | `/sects/upgrade` |
| POST | `/sects/toggle-approval` | Bật/tắt duyệt thành viên gia nhập. Body: `{"require_approval": true}`. |
| POST | `/sects/toggle-promotion-approval` | Bật/tắt duyệt lên cấp (mặc định `false`). Khi `true`, mọi thăng cấp đều tạo yêu cầu chờ Trưởng Lão/Chưởng Môn duyệt. Body: `{"require_promotion_approval": true}`. |
| GET | `/sects/join-requests` |
| POST | `/sects/join-requests/:requestId/action` |
| GET | `/sects/promotion-requests` |
| POST | `/sects/promotion-requests/:requestId/action` |
| GET | `/sects/mines` | Danh sách Lãnh địa + phòng thủ (`defenders`), `total_defense_power`, `output_per_hour`. |
| POST | `/sects/mines/defend` | Bố trí phòng thủ (Trưởng Lão/Hộ Pháp/Chưởng Môn). Body: `{"mine_id": 1, "defender_type": "member|beast", "player_id": "...", "monster_id": "...", "monster_rank": 3}`. `beast` tốn `hạng × 500` Quỹ. |
| POST | `/sects/mines/undefend` | Rút phòng thủ. Body: `{"defender_id": "uuid"}`. |
| POST | `/sects/mines/attack` | Công kích Lãnh địa — **chỉ Trưởng Lão/Hộ Pháp/Chưởng Môn**. Body: `{"mine_id": 1, "attacker_id": "uuid (rỗng = người phát động)"}`. So sánh sức mạnh, trả `won`, `attacker_power`, `defense_power`. Nhân Khí thuộc về đệ tử được chọn ra trận. |

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
| GET | `/admin/players/:id` | Chi tiết người chơi. Response bỏ `gu_inventory`/`materials_inventory`, trả `gu_count` + `item_count`. Kèm `race`, `gender`, `constitution`, `statuses` (array). |
| GET | `/admin/players/:id/items` | Lazy-load item của người chơi từ `player_inventory_items`. Trả thêm các cổ phương đã mở khóa (`category=recipe`) từ `player_recipe_progress`. Params: `page`, `limit`, `q` (match item_id/item_name theo registry). |
| GET | `/admin/players/:id/gus` | Lazy-load gu trong `players.gu_inventory` (JSONB). Params: `page`, `limit`, `q` (match gu_id/instance_id theo registry). |
| PUT | `/admin/players/:id` | Sửa thông tin người chơi. Body là object chứa 1+ field (chỉ gửi field thay đổi): `username`, `email`, `password`, `bio`, `avatar_url`, `rank`, `sub_rank`, `cultivation`, `realm`, `hp`, `max_hp`, `mana`, `max_mana`, `mortal_stones`, `immortal_stones`, `primeval_stones`, `soul`, `hp_regen_rate`, `mana_regen_rate`, `is_admin`, `is_banned`, `is_online`. Thêm: `race`, `gender`, `constitution` (string), `statuses` (array string, ví dụ `["severely_wounded"]`). Trả list field đã cập nhật. |
| GET | `/admin/gus/search` | Tìm cổ trùng trên toàn server. Params: `gu_id` và/hoặc `instance_id` (match registry ID + tên, ILIKE), `page`, `limit`. Quét 3 nguồn: `players.gu_inventory`, `auction` (`item_type='Gu'`), `caravan_shop_items` + `arena_shop_items`. `location`: `inventory`/`auction`/`shop`. |
| POST | `/admin/players/:id/items` | Thêm item hoặc cổ phương (Công thức) cho người chơi. Body: `{item_id, quantity}`. `item_id` là id item trong `Item.csv` hoặc id recipe trong `Recipe.csv`. Nếu là recipe → mở khóa (unlock) Cổ phương cho người chơi (`player_recipe_progress`, `completion_rate=100`). Trả `kind` = `item`/`recipe`. |
| DELETE | `/admin/players/:id/items` | Xóa item hoặc cổ phương khỏi người chơi. Body: `{item_id, quantity}`. Nếu là recipe → thu hồi (delete) khỏi `player_recipe_progress`. Trả `kind` = `item`/`recipe`. |
| POST | `/admin/players/:id/gus` | Thêm gu cho người chơi. Body: `{gu_id, quantity, rank?}`. `rank` tùy chọn — khi gửi sẽ cấp đúng bậc đó (validate theo `gu_dictionary`); khi bỏ trống dùng bậc cao nhất của `gu_id`. |
| DELETE | `/admin/players/:id/gus/:instance_id` | Xóa gu theo instance_id. |
| GET | `/admin/players/:id/pets` | Lazy-load sủng vật của người chơi. Params: `page`, `limit`, `q` (search monster name/ID). |
| DELETE | `/admin/players/:id/pets/:pet_id` | Xóa vĩnh viễn sủng vật. |
| PUT | `/admin/players/:id/stones` | Đổi đá (stones) của người chơi. |
| GET | `/admin/audit-logs` | Audit log (bao gồm `admin.update_player`). |
