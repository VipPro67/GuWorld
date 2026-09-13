# Backend API Map

Base path: `/api`.

Các route protected yêu cầu cookie/JWT hợp lệ qua `middleware.AuthRequired()` trừ khi ghi public.

## Debug endpoints (ngoài `/api`)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/debug/pprof/` | Pprof index / dashboard trình duyệt hoặc `go tool pprof`. |
| GET | `/debug/pprof/profile` | CPU profiling (`?seconds=30`), phân tích CPU tiêu thụ bằng `go tool pprof`. |
| GET | `/debug/pprof/trace` | Execution tracer profile (`?seconds=5`), xem timeline goroutines bằng `go tool trace`. |
| GET | `/debug/pprof/cmdline` | Command line arguments của tiến trình Go. |
| GET/POST | `/debug/pprof/symbol` | Symbol resolution cho trình pprof. |
| GET | `/debug/pprof/goroutine` | Goroutine dump (chẩn đoán request treo hoặc goroutine leak). |
| GET | `/debug/pprof/goroutine/stack` | Toàn bộ goroutine stack dạng text thô. |
| GET | `/debug/pprof/heap` | Heap memory profile (phát hiện memory leak). |
| GET | `/debug/pprof/allocs` | Past memory allocation profile. |
| GET | `/debug/pprof/block` | Block profile (bật `runtime.SetBlockProfileRate` để có data). |
| GET | `/debug/pprof/mutex` | Mutex contention profile. |
| GET | `/debug/pprof/threadcreate` | OS thread creation profile. |
| GET | `/debug/db-pool` | DB connection pool stats (open/inUse/idle/wait). |
| GET | `/debug/system` | Public diagnostic: tổng quan CPU %, Goroutines, RAM, DB Pool, Redis, WebSockets, Uptime. |
| GET | `/debug/requests/summary` | Public diagnostic: tổng quan RPS, in-flight, status codes, latency percentiles. |

> **Log request:** mọi request (trừ OPTIONS) được log `REQ START` khi bắt đầu và `REQ END` khi kết thúc kèm duration. Request > 5s được đánh dấu `REQ SLOW` — dùng để phát hiện handler đang treo mà log cũ không thấy (Gin chỉ log khi request hoàn thành).

> **DB hardening:** connection string mặc định kèm `statement_timeout=15000ms`, `lock_timeout=10000ms` (override qua `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS`). Query bị lock/treo sẽ tự abort thay vì treo vô hạn.

> **SEO routes:** `robots.txt`, `sitemap.xml`, `/wiki`, `/wiki/[guId]` do Next.js frontend xử lý (SSG/ISR). Xem `frontend/src/app/`.

## Public / auth / payment entrypoints

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/dictionary` | Public Gu + item + monster + constitution + race + loot + region + spawn dictionary. Response: `{gu, items, monsters, constitutions, races, loot_tables, regions, monster_spawns}` — `constitutions` và `races` là mảng `{key, name, effects}` (effects là JSON theo format trong `docs/GAME_DATA.md`). `loot_tables` là mảng `{id, name, entries}` (entries chứa item_id/gu_id + chance + min/max qty). `regions` là mảng `WikiRegion {id, name, description, min_rank, max_rank, parent_id, zone_type, pool_weights}` (pool_weights chứa resources[] và wild_gus[]). `monster_spawns` là mảng `{region_id, monster_id, chance, min_rank, max_rank, is_boss_spawn}`. |
| GET | `/dictionary/gus` | Public Gu dictionary only. Returns a flat array containing every Gu rank. |
| GET | `/dictionary/items` | Public item dictionary only. Returns the item map keyed by item ID. |
| GET | `/dictionary/monsters` | Public monster dictionary only. Returns the monster map keyed by monster ID. |
| GET | `/dictionary/constitutions` | Public constitution dictionary only. Returns an array of `{key, name, effects}`. |
| GET | `/dictionary/races` | Public race dictionary only. Returns an array of `{key, name, effects}`. |
| GET | `/dictionary/loot-tables` | Public loot table dictionary only. |
| GET | `/dictionary/regions` | Public wiki region dictionary only. |
| GET | `/dictionary/monster-spawns` | Public wiki monster spawn dictionary only. |
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
| POST | `/player/complete-onboarding` | Hoàn tất thiết lập ban đầu. Body: `{"starting_region","attack_gu_id","defense_gu_id","vital_gu_choice"}`. `starting_region` gồm `hub_nam_cuong`/`hub_trung_chau`/`hub_bac_nguyen`/`hub_tay_mac`/`hub_dong_hai`; người chơi được đặt vào HUB tương ứng của Đại Vực đã chọn. |
| POST | `/player/use-item` |
| POST | `/player/use-items` |
| POST | `/player/recharge` |
| POST | `/player/clear-stress` |
| GET | `/player/profile/:username` |
| PUT | `/player/profile/settings` |
| GET | `/player/combat-settings` | Đọc `combat_settings` (auto-heal/mana + **thứ tự ưu tiên loại** `category_priorities` + **bộ cổ sử dụng** `gu_priorities` + `km_priorities`). |
| PUT | `/player/combat-settings` | Lưu `combat_settings`. `category_priorities` = thứ tự ưu tiên loại kỹ năng (`shield`, `heal`, `damage`, `other`); `gu_priorities` = **bộ cổ mang theo** (danh sách `gu_id` theo thứ tự `priority`): dùng khi tấn công (ưu tiên hơn gợi ý UI) và khi bị tấn công; rỗng = dùng tất cả Cổ đang trang bị ACTIVE. |
| GET | `/player/referrals` |
| POST | `/player/simulate-deposit` | Admin only. |
| POST | `/player/avatar` |
| POST | `/player/sacrifice` |
| GET | `/player/beast-betting/current` | Lấy phiên cược đấu thú hiện tại của user. |
| GET | `/player/beast-betting/history` | Lấy lịch sử 10 trận đấu thú gần nhất và log chiến đấu. |
| POST | `/player/beast-betting/bet` | Đặt cược Nguyên Thạch vào phe Đỏ hoặc Xanh. Body: `{"team": "team_a"|"team_b", "amount": float64}`. |

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
| POST | `/inventory/status` | Cập nhật trạng thái Cổ trùng (`ACTIVE`, `SLEEPING`, `SEALED`). Khi phong ấn (`SEALED`), tiêu hao `sealing_stone` theo scale Rank Cổ (`max(1, rank)` đá). |
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
| POST | `/combat/capture` | Bắt quái vật làm sủng vật. |
| POST | `/explore` | Du hành tới zone: body `{"player_id","node_id"}` (node_id = id Đại Vực / địa danh / `hub_nam_cuong`). |
| POST | `/explore/retreat/:id` | |
| POST | `/explore/flee/:id` | Bỏ chạy khỏi trận đang chiến. **Mỏ & PvP: luôn thành công, không phạt.** Còn lại: 70% thành công (BOSS/Tribulation 40%). Thành công → trừ 20% Max HP nếu là PvE thường (có thể ngất + rơi đồ). Thất bại → địch phản công; trả `flee_success:false` kèm `defeat` / `victory` (địch tự chết → thắng tại chỗ + nhận loot + xoá địch khỏi map) và `logs`. |
| GET | `/explore/grid` | Viewport 5×5 quanh player; trả thêm `world_width`, `world_height`. Bỏ `explored_cells`. |
| GET | `/explore/regions` | Danh sách zone (id, name, geometry `min_x..max_y`, `entry_x/y`, `parent_id`, `zone_type`). `zone_type`: `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC` — khi chồng lấn, ưu tiên `HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC`. |
| GET | `/explore/world` | Cấu hình thế giới: `width`, `height`, `large_zone_size`, `sub_zone_size`, `hub_region_id`, `hub_x/y`. |
| GET | `/explore/objects` | Các ô persistent (map_cells): village/trạm nghỉ, boss, linh tài — cho bản đồ lớn. Trả `{region_id,x,y,cell_type,content_id,rank,respawn_seconds,cleared}`. |
| GET | `/explore/cells` | Các ô non-empty (Monster/Boss/Resource/WildGu) quanh player (`?x=&y=&r=`, mặc định quanh player, r cap 300) để bản đồ lớn vẽ marker quái/tài nguyên. Nội dung **ổn định theo ô** (không re-randomize theo thời gian), khớp đúng nội dung khi di chuyển. Trả `[{x,y,cell_type,content,rank,sub_rank}]`. |
| GET | `/explore/cells/rect` | Các ô non-empty trong hình chữ nhật `?min_x=&min_y=&max_x=&max_y=` (mỗi cạnh ≤ 900) — dùng để prefetch đúng vùng nhìn / dải phía trước hướng đi; client gộp vào cache ổn định thay vì thay thế. Trả cùng định dạng `[{x,y,cell_type,content,rank,sub_rank}]`. |
| POST | `/explore/move` | Di chuyển tới toạ độ trong thế giới (bounds = `world_width`/`world_height`). |
| POST | `/explore/move-path` | Di chuyển tự do theo đường thẳng từ ô hiện tại tới toạ độ đích, trigger cell action trên từng ô đi qua (gom loot; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss**, không có phục kích ngẫu nhiên trên ô trống). Mỗi request tiến **tối đa 150 ô** để commit vị trí mịn (người chơi khác thấy chuyển động gần real-time, đổi hướng/dừng sớm hơn); trả `{player, log, combat_triggered, reached, skip_result?}` (đã tối ưu: loại bỏ `logs` mảng và `stats` thừa, `player` là snapshot cập nhật vị trí/sinh lực/trạng thái tinh gọn, tránh query DB dư thừa); nếu `reached=false` và không combat, client chain tiếp request từ vị trí mới cho tới khi tới đích. |
| POST | `/explore/teleport` | Dịch chuyển tức thời đến vùng an toàn của một khu vực (KHU_VUC/NGU_VUC). Body: `{"region_id": "..."}`. Chi phí dựa trên khoảng cách Chebyshev, tu vi (rank/sub_rank), và độ đầy túi đồ. Trả Primeval Stones (cấp < 6) hoặc Immortal Stones (cấp ≥ 6). Response: `{player, x, y, region, cost, currency, message}`. |

## Sủng vật (Pets)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/pets` | Danh sách sủng vật của người chơi. |
| POST | `/pets/feed` | Cho sủng vật ăn. Body: `{"pet_id": "...", "item_id": "..."}`. |
| POST | `/pets/formation` | Đặt vị trí trong đội hình. Body: `{"pet_id": "...", "slot": 1}`. Slot 0 = storage. |
| POST | `/pets/sleep` | Chuyển sủng vật active ↔ sleeping. Body: `{"pet_id": "..."}`. |
| POST | `/pets/release` | Thả sủng vật. Body: `{"pet_id": "..."}`. |
| POST | `/pets/merge` | Gộp sủng vật cùng loại. Body: `{"source_ids": ["..."], "target_id": "..."}`. Giới hạn bầy = `max_soul/10` (chặn toàn bộ khi vượt, trả `merge_limit`). Chỉ số scale sub-linear `base×N^0.65`. |

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
| POST | `/immortal-aperture/nodes/collect-all` |
| POST | `/immortal-aperture/nodes/:id/uninstall` |
| DELETE | `/immortal-aperture/nodes/:id` |
| POST | `/immortal-aperture/time-rate` |
| POST | `/immortal-aperture/labor` |
| POST | `/immortal-aperture/nodes/:id/repair` |
| POST | `/immortal-aperture/nodes/repair-all` |
| GET | `/immortal-aperture/variant-humans` |
| GET | `/dao-marks` |
| GET | `/dao-marks/dao-ngan` | Đạo ngân (composite dao power) calculation. |
| GET | `/tribulations/active` |
| POST | `/tribulations/:eventId/engage` | Xuất trận chống quái tai kiếp — tạo `active_combat` (nhiều địch, `enemy_type` = `TRIBULATION_BOSS`/`TRIBULATION_MONSTER`) rồi đánh qua combat UI thông thường (`/combat/start`). Giáp + máu người chơi tiêu hao trước, sát thương tràn sẽ trừ `immortal_apertures.aperture_hp`. |
| POST | `/tribulations/:eventId/resolve` |

## Shop, marketplace and P2P trade

| Method | Path |
| --- | --- |
| GET | `/shop/packages` |
| GET | `/shop/list` |
| POST | `/shop/buy-package` |
| POST | `/shop/exchange` | Đổi đá giữa Nguyên Thạch và Tiên Nguyên Thạch. Body: `{"currency": "PRIMEVAL_STONE" | "IMMORTAL_ESSENCE_STONE", "amount": <int64>}`. Tỷ giá: 1 Tiên Nguyên Thạch = 100 Nguyên Thạch; 100.000 Nguyên Thạch = 1 Tiên Nguyên Thạch (amount bằng `IMMORTAL_ESSENCE_STONE` trả `amount*100` Nguyên Thạch; bằng `PRIMEVAL_STONE` phải là bội số của 100.000 và trả `amount/100000` Tiên Nguyên Thạch). Trả `{"message": "...", "wallet": {...}}`. |
| POST | `/shop/buy` |
| POST | `/shop/sell` |
| GET | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/buy` |
| POST | `/shop/bao-hoang/bid` |
| POST | `/shop/bao-hoang/cancel` | Hủy đơn đăng bán. Nếu là `AUCTION` và đã có người đặt giá (`highest_bidder_id`), yêu cầu bị từ chối. |
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

**Quy định phí & thời gian Bảo Hoàng Thiên:**
- **Cửa hàng (`MARKET`):** Phí Upfront thu ngay lập tức khi đăng lên kệ = **4% giá trị niêm yết** (`price * quantity`), khấu trừ theo loại tiền tệ của đơn hàng (Nguyên Thạch / Tiên Nguyên Thạch). Phí hoa hồng khi bán thành công = **6%** doanh thu.
- **Đấu giá (`AUCTION`):** Phí hoa hồng khi đấu giá thành công = **15%** giá thắng thầu. Thời gian đấu giá giới hạn từ **1 đến 24 giờ** (dưới 1 ngày). Không được hủy đơn đấu giá khi đã có người đặt giá (`highest_bidder_id`).

**Loại tiền theo rank (Thương Đội):** `/shop/list` trả thêm `rank` và `currency` cho mỗi món hàng; `/shop/buy` trừ và `/shop/sell` / `/shop/sell-bulk` trả theo loại tiền tương ứng rank của món (item/Gu/bí phương): **rank < 6 → Nguyên Thạch (`primeval_stones`), rank ≥ 6 → Tiên Nguyên Thạch (`immortal_stones`)**. Quy tắc tập trung ở `pkg/pricing.StoneTierByRank`.

**Rank cho Cổ đa rank:** `/shop/buy` nhận thêm `rank` (body, mặc định 0 = tự suy variant rank thấp nhất); `rank > 0` mua đúng variant đó. `/shop/sell` và `/shop/sell-bulk` cũng nhận thêm `rank` cho từng món. `caravan_shop_items` có khóa `(id, rank)`, `rank = 0` = auto.

**Xoay vòng hàng theo ngày:** `is_default = true` luôn hiển thị; món `is_default = false` được shuffle deterministic theo ngày (seed `YYYYMMDD` UTC) theo tỷ lệ **~40 Cổ / ~60 item**. Ngoài ra mọi **Cổ trùng rank < 5** và **thức ăn của chúng** được tự thêm vào pool ngẫu nhiên (không persist, không có trong danh sách admin/export). Giá món random = `base_price × 1.5`; mỗi món có **stock hằng ngày theo rank** (R1=100, R2=50, R3=20, R4=5, R5=2, R6+=1), lưu trong `caravan_shop_daily_stock`, `/shop/buy` trừ stock và từ chối khi hết.

**Admin shop** (thêm/sửa/xóa theo rank): `PUT /admin/shop/:id/:rank`, `DELETE /admin/shop/:id/:rank`; body POST/PUT chứa `rank` (0 = auto).

## Chat/mail, soul and instances

| Method | Path |
| --- | --- |
| POST | `/chat/mail/send` | Gửi thư (tiêu hao 1 Nguyên Thạch, +1 Nhân Khí). Trả về `{ message }`; thông tin player đồng bộ qua WebSocket `player.updated`. |
| GET | `/chat/mail/inbox` |
| POST | `/chat/bao-hoang/send` | Gửi tin nhắn chat. Trả về `{ message }`; thông tin player (Nguyên Thạch, Nhân Khí) đồng bộ qua WebSocket `player.updated`. |
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
| PUT | `/admin/players/:id` | Sửa thông tin người chơi. Body là object chứa 1+ field (chỉ gửi field thay đổi): `username`, `email`, `password`, `bio`, `avatar_url`, `rank`, `sub_rank`, `cultivation`, `realm`, `hp`, `max_hp`, `mana`, `max_mana`, `mortal_stones`, `immortal_stones`, `primeval_stones`, `soul`, `hp_regen_rate`, `mana_regen_rate`, `is_admin`, `is_banned`, `is_online`. Thêm: `race`, `gender`, `constitution` (string), `statuses` (array string hoặc array object `{id, expires_at?, source?, value?}`, ví dụ `["severely_wounded"]` — key phải tồn tại trong `status_dictionary`). Trả list field đã cập nhật. |
| GET | `/admin/gus/search` | Tìm cổ trùng trên toàn server. Params: `gu_id` và/hoặc `instance_id` (match registry ID + tên, ILIKE), `page`, `limit`. Quét 3 nguồn: `players.gu_inventory`, `auction` (`item_type='Gu'`), `caravan_shop_items` + `arena_shop_items`. `location`: `inventory`/`auction`/`shop`. |
| POST | `/admin/players/:id/items` | Thêm item hoặc cổ phương (Công thức) cho người chơi. Body: `{item_id, quantity}`. `item_id` là id item trong `Item.csv` hoặc id recipe trong `Recipe.csv`. Nếu là recipe → mở khóa (unlock) Cổ phương cho người chơi (`player_recipe_progress`, `completion_rate=100`). Trả `kind` = `item`/`recipe`. |
| DELETE | `/admin/players/:id/items` | Xóa item hoặc cổ phương khỏi người chơi. Body: `{item_id, quantity}`. Nếu là recipe → thu hồi (delete) khỏi `player_recipe_progress`. Trả `kind` = `item`/`recipe`. |
| POST | `/admin/players/:id/gus` | Thêm gu cho người chơi. Body: `{gu_id, quantity, rank?}`. `rank` tùy chọn — khi gửi sẽ cấp đúng bậc đó (validate theo `gu_dictionary`); khi bỏ trống dùng bậc cao nhất của `gu_id`. |
| DELETE | `/admin/players/:id/gus/:instance_id` | Xóa gu theo instance_id. |
| GET | `/admin/players/:id/pets` | Lazy-load sủng vật của người chơi. Params: `page`, `limit`, `q` (search monster name/ID). |
| DELETE | `/admin/players/:id/pets/:pet_id` | Xóa vĩnh viễn sủng vật. |
| PUT | `/admin/players/:id/stones` | Đổi đá (stones) của người chơi. |
| GET | `/admin/audit-logs` | Audit log (bao gồm `admin.update_player`). |

## Admin — Monitoring & Tracing (CPU, Memory, Request Traces)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/monitor/system` | Báo cáo chi tiết hệ thống: CPU % (process & system), NumCPU, Goroutines, CgoCalls, Go MemStats (Alloc, Heap, Stack, GC pause), DB Connection Pool, Redis health & latency, WebSocket clients/channels, Process Uptime. |
| GET | `/admin/monitor/requests/summary` | Thống kê request toàn diện: Tổng số request, in-flight requests, RPS (cửa sổ trượt 60s), phân bố mã trạng thái (2xx/3xx/4xx/5xx), độ trễ (min/max/avg/p50/p90/p99), top endpoint được gọi nhiều nhất và top endpoint chậm nhất. |
| GET | `/admin/monitor/requests/traces` | Tra cứu chi tiết danh sách các request gần nhất (Ring Buffer 500 requests). Hỗ trợ query params: `limit` (default 50, max 200), `offset`, `method` (GET/POST/...), `status` (`200`, `500`, hoặc `errors` để lọc lỗi ≥ 400), `path` (chuỗi con URL), `min_duration_ms` (lọc slow request), `search` (tìm theo player_id, username, client_ip). |
| GET | `/admin/monitor/requests/traces/:id` | Xem thông tin chi tiết một request cụ thể qua `trace_id` (Method, Path, Query, Status, Latency ms, Client IP, Player ID, Username, User Agent, Error message). |
| POST | `/admin/monitor/requests/reset` | Đặt lại (reset) toàn bộ chỉ số đếm và xóa danh sách request trace gần nhất. |

## Realtime Communication — WebSocket `GET /api/ws` (Auth)

WebSocket endpoint chính thức thay thế SSE, sử dụng Redis Pub/Sub đa phiên bản phục vụ mở rộng horizontal. Kết nối yêu cầu token qua header `Authorization: Bearer <token>` hoặc query parameter `?token=<token>`.

### Các kênh hỗ trợ (Channels):
| Channel | Direction | Mục đích |
| --- | --- | --- |
| `player` | S→C | Snapshot & cập nhật chỉ số người chơi thời gian thực |
| `chat:world` | Hai chiều | Chat thế giới |
| `chat:general` | Hai chiều | Chat Bảo Hoàng Thiên (yêu cầu Thông Thiên Cổ & Thần Niệm Cổ) |
| `chat:sect` | Hai chiều | Chat tông môn |
| `trade` | S→C | Trạng thái giao dịch realtime |
| `world:pos` | S→C | Vị trí người chơi di chuyển trên bản đồ lớn |
| `world:cell` | S→C | Sự kiện Boss/tài nguyên bị tiêu diệt hoặc hồi sinh |
| `combat` | S→C | Biến động chiến đấu realtime |
| `notification` | S→C | Thông báo mới (chuông thông báo) |

### Lệnh WorldMap qua WebSocket

WorldMap ưu tiên dùng cùng kết nối WebSocket để tránh tạo nhiều HTTP request khi
di chuyển và kéo camera. Client gửi envelope
`{type:"command", channel, request_id, payload}` và nhận
`{type:"command.result", channel, request_id, ok, status, data}`. Các lệnh hiện có:

| Channel | Payload | Tương đương HTTP |
| --- | --- | --- |
| `world.move_path` | `{x,y}` | `POST /explore/move-path` |
| `world.cells_rect` | `{min_x,min_y,max_x,max_y}` | `GET /explore/cells/rect` |
| `world.objects` | `{}` | `GET /explore/objects` |

Danh tính người chơi luôn lấy từ phiên WebSocket đã xác thực, không lấy từ
payload. Các HTTP endpoint vẫn được giữ làm fallback khi WebSocket chưa kết nối.

### SSE realtime — `GET /events` (Auth) [Deprecated Fallback]

Stream hợp nhất (`StreamUnifiedEvents`) giữ lại làm fallback cho authenticated player. Các event-type:

| Event | Mô tả |
| --- | --- |
| `mana.snapshot` | Snapshot player ban đầu khi kết nối. |
| `player.updated` | Cập nhật player realtime (mana, hp, vị trí, …). |
| `player.heartbeat` | Heartbeat ~60s. |
| `combat.started` / `combat.ended` | Trạng thái chiến đấu. |
| `trade.updated` | Trạng thái giao dịch. |
| `chat.messages.world` / `chat.messages.general` / `chat.messages.sect` | Chat channel. |
| `world.player.pos` | **Vị trí người chơi trên bản đồ lớn**: broadcast sau mỗi commit di chuyển (`/explore/move`, `/explore/move-path`). Payload `{id, username, avatar_url, x, y, ts}`. Client lọc theo viewport và nội suy marker để hiển thị chuyển động mượt (không teleport). Broadcast qua hub `world_events.go`; frontend phát `CustomEvent("gu_world_pos")` → `useWorldStore.upsertPlayerPosition`. Fan-out được interest-manage theo vị trí (chỉ gửi cho subscriber trong bán kính `WorldPosInterestRadius`). |
| `world.cell.cleared` | **Cell cố định (boss/resource cao cấp) bị clear**: payload `{x, y, cell_type, content_id}`. Frontend phát `CustomEvent("gu_world_cell")` → refresh `useWorldStore.fetchObjects()`. |
| `world.cell.respawn` | **Cell cố định respawn** (hết `respawn_seconds`, reset lười khi đọc): payload `{x, y, cell_type, content_id}`. Frontend xử lý tương tự `gu_world_cell`. |
| `notification.new` | **Thông báo mới** cho người chơi (bell icon). Payload là object `{id, player_id, type, title, content, is_read, created_at}`. |

Kèm theo: `GET /player/events` (stream player snapshot + heartbeat riêng), `GET /trade/events`, `GET /chat/bao-hoang/events`.

## Thông báo (Notification) — Auth

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| GET | `/player/notifications?page=&pageSize=` | Danh sách thông báo của người chơi (mới nhất trước). Trả `{notifications, page, pageSize}`. |
| GET | `/player/notifications/unread-count` | Số thông báo chưa đọc. Trả `{unread}`. |
| POST | `/player/notifications/read` | Đánh dấu đã đọc theo `{ids: [...]}`. |
| POST | `/player/notifications/read-all` | Đánh dấu tất cả đã đọc. |

Các loại `type` thông báo: `gu_death`, `vital_gu_death` (cổ chết đói), `death_drop` (tử đạo mất tài sản), `loot_gained` (PvP nhặt chiến lợi phẩm), `combat_loot` (nhặt đồ sau khi thắng quái), `market_sold`/`market_bought` (mua bán), `auction_bid` (ai đặt giá), `auction_won`/`auction_sold`/`auction_expired` (đấu giá).

## Đấu Trường Sinh Tử (Death Arena) — Auth

Diễn ra vào **17:00 (5h chiều) Thứ 7 hàng tuần** (Mở đăng ký: 16:00 - 17:00 Thứ 7).

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| GET | `/death-arena/current` | Lấy thông tin giải đấu hiện tại, trạng thái đăng ký của bản thân và thông tin đội. |
| POST | `/death-arena/register` | Đăng ký tham gia giải đấu (giới hạn tối đa 96 người chơi / 16 đội). |
| POST | `/death-arena/unregister` | Hủy đăng ký tham gia giải đấu (khi còn mở đăng ký). |
| GET | `/death-arena/bracket` | Lấy sơ đồ bảng đấu và các trận đấu (hỗ trợ Redis cache 3s). |
| GET | `/death-arena/my-team` | Lấy danh sách thành viên và trạng thái HP/Mana đội của bản thân. |
| GET | `/death-arena/matches/:id/log` | Lấy lịch sử chi tiết (Combat Log Replay) của trận đấu. |
| GET | `/death-arena/rewards` | Lấy danh sách phần thưởng giải đấu của bản thân. |
| POST | `/death-arena/rewards/:id/claim` | Nhận phần thưởng giải đấu. |
| GET | `/death-arena/history` | Lịch sử các mùa giải Đấu Trường Sinh Tử trước. |
