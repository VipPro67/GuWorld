# Backend API Map

Base path: `/api`.

Các route protected yêu cầu cookie/JWT hợp lệ qua `middleware.AuthRequired()` trừ khi ghi public.

## Diagnostics

Các route public `/debug/*` và `/api/debug/monsters` đã bị loại bỏ vì làm lộ
thông tin runtime. Thống kê hệ thống/request chỉ còn trong nhóm protected
`/api/admin/monitor/*` dành cho admin.

> **Log request:** mọi request (trừ OPTIONS) phát event structured `http_request_started` và `http_request_completed`, liên kết bằng `trace_id`. Event hoàn tất có method, route/path, status, duration, IP, player ID/username, user-agent và JSON body đã lọc. Body chỉ được ghi với `Content-Type: application/json`, tối đa 8 KiB; password/token/OTP/secret và các key nhạy cảm được thay bằng `[REDACTED]`, body lỗi/quá dài chỉ lưu marker, còn multipart/binary và các nhóm auth/webhook/top-up/chat không được ghi. Query string cũng được redact. Request ≥ 5 giây được log mức warning.

> **DB hardening:** connection string mặc định kèm `statement_timeout=15000ms`, `lock_timeout=10000ms` (override qua `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS`). Query bị lock/treo sẽ tự abort thay vì treo vô hạn.

> **SEO routes:** `robots.txt`, `sitemap.xml`, `/wiki`, `/wiki/[guId]` do Next.js frontend xử lý (SSG/ISR). Xem `frontend/src/app/`.

## Public / auth / payment entrypoints

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/dictionary` | Unified catalog cho game client: `{gu, items, monsters, constitutions, races, statuses}` — `constitutions`, `races`, `statuses` là mảng `{key, name, effects}` (effects là JSON theo format trong `docs/GAME_DATA.md`). Payload đã được tinh gọn: các phần wiki-only (`loot_tables`, `regions`, `monster_spawns`) và field nội bộ của monster (gu_inventory, gu_pool, item_inventory, dao_marks, path_masteries, ai_profile) không còn được trả — dùng các endpoint `/dictionary/*` riêng cho wiki. |
| GET | `/dictionary/gus` | Public Gu dictionary only. Returns a flat array containing every Gu rank. |
| GET | `/dictionary/items` | Public item dictionary only. Returns the item map keyed by item ID. |
| GET | `/dictionary/monsters` | Public monster dictionary only. Returns the monster map keyed by monster ID, projected to client-facing fields `{id, name, type, max_hp, base_attack, base_defense, loot_table_id, description, is_capturable, capture_chance, pet_hunger_per_hour}` (internal combat/loot loadouts omitted). |
| GET | `/dictionary/constitutions` | Public constitution dictionary only. Returns an array of `{key, name, effects}`. |
| GET | `/dictionary/races` | Public race dictionary only. Returns an array of `{key, name, effects}`. |
| GET | `/dictionary/statuses` | Public status dictionary only. Returns an array of `{key, name, effects, default_duration_minutes, blocked_actions, description}`. |
| GET | `/dictionary/loot-tables` | Public loot table dictionary only. |
| GET | `/dictionary/regions` | Public wiki region dictionary only. |
| GET | `/dictionary/monster-spawns` | Public wiki monster spawn dictionary only. |
| GET | `/recipes` | Public refining recipes. Thêm query param `?all=true` để lấy toàn bộ danh sách thay vì chỉ những công thức đã mở khóa của người chơi. Hỗ trợ `If-None-Match`; response trả `ETag` và `X-Catalog-Revision`, còn `304` giữ nguyên cache hiện tại của client. |
| GET | `/gus/:instance_id/history` | Lịch sử vòng đời một cá thể Cổ (chỉ Tiên Cổ rank ≥ 6 có log). `instance_id` bất biến qua trade/bán/kho; destroy (thả/đói/chết) đóng timeline. |
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
| PUT | `/player/profile/settings` | Cập nhật hồ sơ (bio, avatar, mật khẩu, ngụy trang). Khi đổi Đạo Hiệu (`username`): Rank 1-2 miễn phí; Rank 3-5 tốn Nguyên Thạch (5k/20k/100k); Rank 6-9 tốn Tiên Nguyên Thạch (100/500/2.5k/10k), tụt Đạo Ngân & Chân Ý Lưu Phái (-30%/-40%/-50%/-70%), tổn hại Tiên Khiếu và phát thông báo Thế Giới (không tụt tiểu cảnh giới). |
| GET | `/player/combat-settings` | Đọc `combat_settings` (auto-heal/mana + **thứ tự ưu tiên loại** `category_priorities` + **bộ cổ sử dụng** `gu_priorities` + `km_priorities`). |
| PUT | `/player/combat-settings` | Lưu `combat_settings`. `category_priorities` = thứ tự ưu tiên loại kỹ năng (`shield`, `heal`, `damage`, `other`); `gu_priorities` = **bộ cổ mang theo** (danh sách `gu_id` theo thứ tự `priority`): dùng khi tấn công (ưu tiên hơn gợi ý UI) và khi bị tấn công; rỗng = dùng tất cả Cổ đang trang bị ACTIVE. |
| GET | `/player/referrals` |
| POST | `/player/referrals/claim` |
| POST | `/player/referrals/bind` |
| POST | `/player/simulate-deposit` | Admin only. |
| POST | `/player/avatar` |
| POST | `/player/sacrifice` |
| POST | `/player/redeem-gift-code` |
| GET | `/player/audit-logs` |
| GET | `/player/notifications` |
| GET | `/player/notifications/unread-count` |
| POST | `/player/notifications/read` |
| POST | `/player/notifications/read-all` |
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
| GET | `/quest` |
| POST | `/quest/complete` |
| GET | `/achievements` |
| POST | `/achievements/claim` |
| POST | `/achievements/select` |

## Inventory, refining and killer moves

| Method | Path |
| --- | --- |
| POST | `/inventory/feed` |
| POST | `/inventory/feed-group` |
| POST | `/inventory/feed-all` | Cho ăn toàn bộ Cổ trùng. Nhận body JSON tùy chọn `{ "scope": "equipped" \| "rest" \| "unequipped" \| "all" }` hoặc `{ "equipped": true \| false }` để lọc theo Cổ trùng đang xuất trận hoặc trong túi. |
| POST | `/inventory/equip` |
| POST | `/inventory/unequip-all` |
| POST | `/inventory/status` | Cập nhật trạng thái Cổ trùng (`ACTIVE`, `SLEEPING`, `SEALED`). Khi phong ấn (`SEALED`), tiêu hao `sealing_stone` theo scale Rank Cổ (`max(1, rank)` đá). |
| POST | `/inventory/set-vital` |
| POST | `/inventory/use-gu` |
| POST | `/inventory/use-support` |
| POST | `/inventory/drop` |
| POST | `/inventory/drop-batch` |
| POST | `/refine` | Luyện Cổ. Recipe `required_gus` là map `gu_id → {rank, qty}`; Cổ nguyên liệu phải khớp `gu_id` và đúng `rank` (khi `rank != 0`). Target rank ≥ 6 trùng toàn server → **409** (kể cả race thua khi concurrent). |
| POST | `/refine/deduce` | Suy diễn recipe. |
| POST | `/killer-moves/calculate` |
| POST | `/killer-moves` |
| GET | `/killer-moves` |
| GET | `/killer-moves/:id` |
| DELETE | `/killer-moves/:id` |
| POST | `/combat/cast-killer-move` |
| POST | `/combat/start` | Trả `logs[]` trong đó mỗi entry sát thương có `breakdown` (công thức: `base_damage`, `skill_mult`, `dao_mult`, `buff_mult`, `crit_mult`, `path_bonus`, `target_def`, `natural_resist`, `path_resist`, `damage_reduction`, `shield_absorbed`, `hp_damage`, `final_damage`, `raw_damage`, `hit_count`, `reason`). Kể từ nay kèm provenance trạng thái: `attacker_attack` / `defender_defense` / `healer_healing` (`{stat, base_value, final_value, mods:[{key,name,pct}]}`) và các trường hồi máu `healing_base`, `healing_dao_bonus`, `healing_final`. |
| POST | `/combat/practice` | Như `/combat/start`. |
| GET | `/combat/history` | 20 entry gần nhất, bản tinh gọn cho list: `{id, player_id, winner, summary, enemy_name, enemy_type, region, logs_count, created_at}` (không kèm JSONB `logs`/`combatants` — lấy chi tiết qua endpoint dưới). |
| GET | `/combat/history/:id` | Full một entry: kèm `logs` và `combatants` JSONB để replay. |

## Exploration and combat

| Method | Path | Ghi chú |
| --- | --- | --- |
| POST | `/combat/capture` | Bắt quái vật làm sủng vật. |
| POST | `/explore` | Du hành tới zone: body `{"player_id","node_id"}` (node_id = id Đại Vực / địa danh / `hub_nam_cuong`). |
| POST | `/explore/retreat/:id` | |
| POST | `/explore/flee/:id` | Bỏ chạy khỏi trận đang chiến. **Mỏ & PvP: luôn thành công, không phạt.** Còn lại: 70% thành công (BOSS/Tribulation 40%). Thành công → trừ 20% Max HP nếu là PvE thường (có thể ngất + rơi đồ). Thất bại → địch phản công; trả `flee_success:false` kèm `defeat` / `victory` (địch tự chết → thắng tại chỗ + nhận loot + xoá địch khỏi map) và `logs`. |
| GET | `/explore/grid` | Viewport 5×5 quanh player; trả `{x, y, vision_radius, cells, stats (chỉ movement + investigation), players_on_map, ground_drops, world_width, world_height}` — đã bỏ `teleport_arrays` (frontend không dùng) và phần stat dư thừa. |
| GET | `/explore/regions` | Danh sách zone (id, name, geometry `min_x..max_y`, `entry_x/y`, `parent_id`, `zone_type`). `zone_type`: `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC` — khi chồng lấn, ưu tiên `HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC`. |
| GET | `/explore/world` | Cấu hình thế giới: `width`, `height`, `large_zone_size`, `sub_zone_size`, `hub_region_id`, `hub_x/y`. |
| GET | `/explore/objects` | Các ô persistent (map_cells): village/trạm nghỉ, boss, linh tài — cho bản đồ lớn. Trả `{region_id,x,y,cell_type,content_id,rank,respawn_seconds,cleared}`. |
| GET | `/explore/cells/rect` | Các ô non-empty trong hình chữ nhật `?min_x=&min_y=&max_x=&max_y=` (mỗi cạnh được cap 256) — dùng để prefetch vùng nhìn / dải phía trước hướng đi. Client cache theo chunk 64×64 và namespace khu vực; trạng thái cleared được đọc theo batch. Trả cùng định dạng `[{x,y,cell_type,content,rank,sub_rank,region_id}]`. |
| POST | `/explore/move` | Di chuyển tới toạ độ trong thế giới (bounds = `world_width`/`world_height`). |
| POST | `/explore/move-path` | Di chuyển tự do theo đường thẳng từ ô hiện tại tới toạ độ đích, trigger cell action trên từng ô đi qua (gom loot; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss**, không có phục kích ngẫu nhiên trên ô trống). Mỗi request tiến **tối đa 150 ô** để commit vị trí mịn (người chơi khác thấy chuyển động gần real-time, đổi hướng/dừng sớm hơn); trả `{player, log, combat_triggered, reached, skip_result?}` (đã tối ưu: loại bỏ `logs` mảng và `stats` thừa, `player` là snapshot cập nhật vị trí/sinh lực/trạng thái tinh gọn, tránh query DB dư thừa); nếu `reached=false` và không combat, client chain tiếp request từ vị trí mới cho tới khi tới đích. |
| POST | `/explore/teleport` | *(Đã vô hiệu hóa)* Dịch chuyển bằng tiền đã bị hủy bỏ. Trả về `400 Bad Request` yêu cầu sử dụng Cổ trùng dịch chuyển (`POST /explore/use_gu`). |
| GET | `/explore/players` | Người chơi ở gần để hiển thị/tương tác trên bản đồ. |
| POST | `/explore/use_gu` | Dùng Cổ bản đồ (ví dụ mở cổng). |
| GET | `/world/drops` | Danh sách chiến lợi phẩm/hài cốt tại ô hiện tại. |
| POST | `/world/drops/:id/loot` | Nhặt một ground drop khi đủ điều kiện khóa sở hữu. |

## Truyền Tống Trận (Teleportation Arrays)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/teleport/templates` | Lấy danh sách các phẩm cấp Trận Pháp (Tier 1: Vi Trận, Tier 2: Đại Vực Trận, Tier 3: Tiên Trận) kèm yêu cầu chi phí và cự ly. |
| GET | `/teleport/mine` | Lấy danh sách các Trận Pháp do bản thân hoặc Tông Môn sở hữu. |
| POST | `/teleport/build` | Bố trí Trận Pháp mới tại toạ độ hiện tại. Body: `{"name","tier","direction_type","access_policy","fee_enabled","fee_currency","fee_amount"}`. |
| GET | `/teleport/directory` | Lấy danh bạ các điểm truyền tống khả dụng từ Trận Pháp mà người chơi đang đứng (lọc cự ly, hướng kết nối, quyền riêng tư, phí). |
| POST | `/teleport/travel` | Kích hoạt truyền tống đến trận pháp đích. Body: `{"target_array_id": "uuid"}`. Trừ phí (95% vào két sắt trận, 5% hao tán), giảm 1 độ bền, áp dụng debuff Giới Vực Bích nếu nhảy xuyên vực. |
| POST | `/teleport/manage` | Thay đổi tên, hướng (hai chiều/1 chiều), quyền (`PUBLIC`/`SECT`/`PERSONAL`), bật/tắt phí và mức giá vé. |
| POST | `/teleport/repair` | Tiêu hao Nguyên Thạch phục hồi độ bền trận pháp. Body: `{"array_id": "uuid", "amount": 10}`. |
| POST | `/teleport/withdraw` | Rút số dư doanh thu từ Két Sắt (`vault_balance`) về túi đồ của chủ sở hữu. Body: `{"array_id": "uuid", "amount": 0}` (0 = rút tất cả). |
| POST | `/teleport/dismantle` | Thu hồi và phá dỡ trận pháp, hoàn trả 50% nguyên liệu chế tạo (tỉ lệ theo % độ bền) và tự động rút két sắt. Body: `{"array_id": "uuid"}`. |

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
| GET | `/party/invitations` |
| POST | `/party/invitations/:id/respond` |
| POST | `/party/leave` |
| GET | `/arena/leaderboard` |
| POST | `/arena/challenge` |
| GET | `/arena/shop` |
| POST | `/arena/shop/buy` |
| GET | `/leaderboard/:category` | BXH tổng hợp. category: `rank`, `primeval_stones`, `gu_strongest`, `immortal_stones`, `sat_khi`. Trả về top 50, kèm rank người gọi |
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
| POST | `/immortal-aperture/nodes/:id/upgrade` |
| GET | `/immortal-aperture/variant-humans` |
| GET/POST | `/immortal-aperture/nodes/:id/workers` |
| DELETE | `/immortal-aperture/nodes/:id/workers/:worker_id` |
| GET/POST | `/immortal-aperture/refinement-queue` |
| DELETE | `/immortal-aperture/refinement-queue/:job_id` |
| GET | `/immortal-aperture/settlements` |
| POST | `/immortal-aperture/settlements/:settlement_id/upgrade` |
| GET | `/immortal-aperture/npcs` |
| POST | `/immortal-aperture/npcs/:npc_id/role` |
| DELETE | `/immortal-aperture/npcs/:npc_id` |
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
| POST | `/shop/sell-bulk` |
| GET | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/buy` |
| POST | `/shop/bao-hoang/bid` |
| POST | `/shop/bao-hoang/cancel` | Hủy đơn đăng bán. Nếu là `AUCTION` và đã có người đặt giá (`highest_bidder_id`), yêu cầu bị từ chối. |
| POST | `/shop/bao-hoang/settle` |
| POST | `/trade/initiate` |
| GET | `/trade/active` |
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
| POST | `/soul/refine` |
| GET | `/instance/active` |
| GET | `/instance/list` | Danh sách phó bản + `cooldown_remaining` (giây) của player |
| POST | `/instance/enter` |
| POST | `/instance/action` |

## Gacha (Cổ Từ / Mở Đá)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/gacha/pools` | Danh sách gói mở: `{id, name, description, cost_type, cost_amount}` (bảng reward + trọng số chỉ nằm server-side). |
| POST | `/gacha/open` | Mở N lần. Body: `{"pool_id": "...", "quantity": 10}` (`quantity` mặc định 1, tối đa 100; ≥ 10 được giảm 10% chi phí). Trúng Tiên Cổ đã tồn tại → **409**. |
| POST | `/gacha/open10` | Tương đương `/gacha/open` với `quantity: 10` (giữ để tương thích). |

## Sects

| Method | Path |
| --- | --- |
| GET | `/sects` | Danh sách các tông môn (ẩn tọa độ bản đồ cụ thể x, y để bảo mật phòng thủ tông môn). Redis cache 15s (khớp Cache-Control 15s của response). |
| POST | `/sects/:id/join` |
| POST | `/sects/join-request/cancel` |
| POST | `/sects/leave` |
| GET | `/sects/me` |
| POST | `/sects/promote` |
| POST | `/sects/donate/stones` |
| POST | `/sects/donate/materials` |
| POST | `/sects/donate/gu` | Quyên góp Cổ (giữ nguyên `instance_id`). Tiên Cổ đã tồn tại ở nơi khác → **409**; rút Tiên Cổ mỗi lần 1 instance. |
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
| GET | `/admin/players/:id/gus` | Lazy-load từ bảng `player_gu`. Params: `page`, `limit`, `q` (match gu_id/instance_id theo registry). |
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

### Admin — nhóm quản trị còn lại

Mọi route dưới đây đều thuộc `/api/admin`, yêu cầu auth/admin và rate limit nhóm:

| Nhóm | Routes hiện có |
| --- | --- |
| Tiên khiếu | `GET /apertures`, `GET/PUT /apertures/:id`; CRUD node và xem/tạo/xóa quần thể dị nhân theo aperture. |
| Cổ/pet/history | cập nhật cá thể Cổ; tạo/cập nhật pet; `GET /gu-history`; `GET /batch-metrics`. |
| Môn phái | list/detail/update/delete sect; thêm/xóa/đặt chưởng môn; CRUD mission; nhập/xóa hàng kho; quản lý sect mine. |
| Dữ liệu & cache | `GET /data/tables`, import/export toàn bộ hoặc `/:table`, clear application cache và Redis cache. |
| World/activity | CRUD region, map cell, monster spawn, dungeon; list/update/reset mine slot. |
| Dictionary/shop | CRUD shop, arena shop, Gu, item, recipe, monster, loot table, constitution, race, status và quest. |
| Hệ thống | gift code list/create, upload image, top-up order, wallet transaction và monitoring/traces. |

Tên method/path chi tiết phải đối chiếu trực tiếp nhóm `admin` trong `backend/internal/server/server.go`; bảng này gom theo domain để tránh hiểu nhầm rằng Admin chỉ quản lý player/Cổ.

### Admin — validation effect Cổ

`POST /api/admin/gu-dictionary` và `PUT /api/admin/gu-dictionary/:id` chuẩn hóa `effects` về envelope tám namespace trước khi ghi. Dữ liệu sai trả `400` với `{ "error": string, "issues": [{ "severity", "code", "path", "message" }] }`. Thành công vẫn trả `issues`; mảng này chứa warning cho field legacy/chưa được runtime xử lý và có thể rỗng. Warning không chặn ghi và không bảo đảm effect đã có gameplay implementation.

| Method | Path | Ghi chú |
| --- | --- | --- |
| POST | `/admin/gu-effects/validate` | Preview read-only một payload `{effects: object}`. Trả `{valid, report, normalized_effects?}`; validation errors nằm trong `report.issues`, còn `normalized_effects` chỉ có khi hợp lệ. Payload tối đa 256 KiB. |
| GET | `/admin/gu-effects/audit` | Audit read-only toàn bộ `gu_dictionary` bằng một query. Trả coverage theo namespace/key, `owner`, `disposition`, số lần xuất hiện và issues gắn với `{id,rank}`. Giới hạn an toàn 100.000 definition/32 MiB effect JSON, 256 KiB/definition và tối đa 10.000 issue chi tiết; xem `issues_truncated` nếu summary còn issue chưa trả chi tiết. |

## Admin — Monitoring & Tracing (CPU, Memory, Request Traces)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/monitor/system` | Báo cáo chi tiết hệ thống: CPU % (process & system), NumCPU, Goroutines, CgoCalls, Go MemStats (Alloc, Heap, Stack, GC pause), DB Connection Pool, Redis health & latency, WebSocket clients/channels, Process Uptime. |
| GET | `/admin/monitor/requests/summary` | Thống kê request toàn diện: Tổng số request, in-flight requests, RPS (cửa sổ trượt 60s), phân bố mã trạng thái (2xx/3xx/4xx/5xx), độ trễ (min/max/avg/p50/p90/p99), top endpoint được gọi nhiều nhất và top endpoint chậm nhất. |
| GET | `/admin/monitor/requests/traces` | Tra cứu chi tiết danh sách các request gần nhất (Ring Buffer 500 requests). Hỗ trợ query params: `limit` (default 50, max 200), `offset`, `method` (GET/POST/...), `status` (`200`, `500`, hoặc `errors` để lọc lỗi ≥ 400), `path` (chuỗi con URL), `min_duration_ms` (lọc slow request), `search` (tìm theo player_id, username, client_ip). |
| GET | `/admin/monitor/requests/traces/:id` | Xem thông tin chi tiết một request cụ thể qua `trace_id` (method, path/query đã lọc, status, latency, client IP, player ID/username, user-agent, error, `content_length`, JSON `request_body` đã redact và `body_truncated`). Các trace chỉ lưu trong memory và hiển thị tại tab **Giám sát** của Admin. |
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
| `world:pos` | S→C | Vị trí người chơi di chuyển trên bản đồ lớn; payload mang `region_id` và fan-out theo AOI cùng khu vực |
| `world:cell` | S→C | Sự kiện Boss/tài nguyên bị tiêu diệt hoặc hồi sinh; payload mang `region_id`, client apply theo tọa độ global kể cả khi viewport cắt qua ranh giới khu vực |
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

> SSE endpoint cũ không còn được đăng ký trong router hiện tại. Client phải dùng `/api/ws`; các HTTP endpoint vẫn là fallback cho thao tác/snapshot.

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
| DELETE | `/death-arena/register` | Hủy đăng ký tham gia giải đấu (khi còn mở đăng ký). |
| GET | `/death-arena/bracket` | Lấy sơ đồ bảng đấu và các trận đấu (hỗ trợ Redis cache 3s). |
| GET | `/death-arena/my-team` | Lấy danh sách thành viên và trạng thái HP/Mana đội của bản thân. |
| GET | `/death-arena/matches` | Danh sách trận của người chơi. |
| GET | `/death-arena/matches/:id/log` | Lấy lịch sử chi tiết (Combat Log Replay) của trận đấu. |
| GET | `/death-arena/rewards` | Lấy danh sách phần thưởng giải đấu của bản thân. |
| POST | `/death-arena/claim-reward` | Nhận phần thưởng giải đấu. |
| GET | `/death-arena/history` | Lịch sử các mùa giải Đấu Trường Sinh Tử trước. |

## Trinh Sát & Suy Diễn Thiên Cơ (Scouting & Divination) — Auth

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/scouting/inspect` | Trinh sát cục bộ mục tiêu. Bắt buộc cùng vị trí/FOV (hoặc cùng Party/Sect). Body: `{"target_id", "target_username"}`. Response: `PlayerProfileResponse` + `delta_score` + `my_score{stat, realm_dao, clue, luck_roll, total}` (full ≥30, partial ≥−10). |
| POST | `/scouting/divine` | Suy diễn thiên cơ từ xa bằng Đạo ngân & Chân ý Trí/Tín Đạo. Body: `{"target_username", "suspected_region", "suspected_sect", "suspected_gu", "item_clue_id", "resource_type", "resource_amount"}`. Mọi clue đều validate đúng-thật mới tính điểm (sai = 0). Response thêm `my_score` (điểm của người dò; điểm thủ không lộ). |
| POST | `/scouting/divine-gu` | Suy tính cổ trùng (ai nắm giữ, đã luyện ra chưa, số lượng thiên hạ, bí phương & thức ăn). Body: `{"gu_query", "target_rank", "resource_type", "resource_amount"}`. Điểm thô chia cho divisor theo bậc Cổ (R1-5: 1/2/3/4/5, R6-9: 10/18/30/50); khó còn lại = che giấu chủ mạnh nhất ×0.5 (Cổ kho/ngủ ×0.5 nữa; tự sở hữu cộng hưởng +250, bỏ qua def chủ) + màn sương 15. Response thêm `my_score` (kèm `rank_divisor`). |
| POST | `/scouting/apply-mark` | Gieo ấn ký truy tung lên mục tiêu (Trí Niệm / Huyết Mạch). Giới hạn 3 (phàm nhân) / 10 (tiên nhân). Body: `{"target_id", "mark_type"}`. |
| GET | `/scouting/tracked-targets` | Lấy danh sách con mồi đang mang ấn ký kèm tọa độ thời gian thực `(region, x, y)`. |
| POST | `/scouting/sever-clues` | Đoạn tuyệt thiên cơ, chém đứt toàn bộ ấn ký đang theo dõi và tàn dư ô bản đồ. Tốn 20 Chân Nguyên (cố định). |
| POST | `/scouting/resonance-scan` | Cảm ứng Cổ trùng và xung kích Đạo ngân (Dao Pulses) trong khu vực. Tốn 50 Chân Nguyên (cố định) + 200 Nguyên Thạch, hoặc TNT với Tiên nhân. |

