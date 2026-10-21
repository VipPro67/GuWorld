# Backend API Map

Base path: `/api`.

Các route protected yêu cầu cookie/JWT hợp lệ qua `middleware.AuthRequired()` trừ khi ghi public.

## Diagnostics

Các route public `/debug/*` và `/api/debug/monsters` đã bị loại bỏ vì làm lộ
thông tin runtime. Thống kê hệ thống/request chỉ còn trong nhóm protected
`/api/admin/monitor/*` dành cho admin.

> **Log request:** mọi request (trừ OPTIONS) phát event structured `http_request_started` và `http_request_completed`, liên kết bằng `trace_id`. Event hoàn tất có method, route/path, status, duration, IP, player ID/username, user-agent và JSON body đã lọc. Request body chỉ được ghi với `Content-Type: application/json`, tối đa 8 KiB; password/token/OTP/secret và các key nhạy cảm được thay bằng `[REDACTED]`, body lỗi/quá dài chỉ lưu marker, còn multipart/binary và các nhóm auth/webhook/top-up/chat không được ghi. Admin request trace có `response_bytes` (số byte body gửi đi); response body giữ trong trace bị giới hạn 8 KiB và có cờ `response_body_truncated`. Query string được redact. Request ≥ 5 giây được log mức warning.

> **DB hardening:** connection string mặc định kèm `statement_timeout=15000ms`, `lock_timeout=10000ms` (override qua `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS`). Query bị lock/treo sẽ tự abort thay vì treo vô hạn. Riêng `InitSchema` lúc boot chạy trên connection riêng với timeout nới lỏng (`DB_MIGRATION_STATEMENT_TIMEOUT_MS`/`DB_MIGRATION_LOCK_TIMEOUT_MS`, mặc định 300000/60000 ms) để backfill/consolidate trên bảng lớn không FATAL.

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
| GET | `/dictionary/region-loot` | Public wiki region loot rules dictionary only (`[{region_id, loot_table_id, monster_type, min_rank, max_rank, chance}]`). |
| GET | `/recipes` | Public refining recipes. Thêm query param `?all=true` để lấy toàn bộ danh sách thay vì chỉ những công thức đã mở khóa của người chơi. Hỗ trợ `If-None-Match`; response trả `ETag` và `X-Catalog-Revision`, còn `304` giữ nguyên cache hiện tại của client. |
| GET | `/gus/:instance_id/history` | Lịch sử vòng đời một cá thể Cổ (chỉ Tiên Cổ rank ≥ 6 có log). `instance_id` bất biến qua trade/bán/kho; destroy (thả/đói/chết) đóng timeline. |
| POST | `/auth/send-otp` | Send OTP via email to register. |
| POST | `/auth/register` | Rate limited. Requires valid OTP. Body thêm `race` (`human`/`fur_people`) và `gender` (`male`/`female`), mặc định `human`/`male`. |
| POST | `/auth/login` | Rate limited. |
| POST | `/auth/logout` | Public logout endpoint. |
| POST | `/auth/claim-tab` | Authenticated gameplay tab claim; a newer gameplay tab replaces the previous one. Admin endpoints are exempt. |
| GET | `/auth/me` | Current player, protected. |
| GET | `/auth/google/login` | Google OAuth start, rate limited. |
| GET | `/auth/google/callback` | Google OAuth callback, rate limited. |
| GET | `/topup/packages` | Public topup packages. |
| POST | `/webhooks/payos` | PayOS webhook. |
| GET | `/admin/topup-orders` | Protected server-side listing. Query: `page`, `page_size`, `search`, allowlisted `sort`, `direction`, `status`, `provider`, `user_id`. Response: `{data,total,page,page_size}`. |
| GET | `/admin/wallet-transactions` | Protected server-side listing. Query: `page`, `page_size`, `search`, allowlisted `sort`, `direction`, `user_id`, `type`, `currency`. Response: `{data,total,page,page_size}`. |
| GET | `/wallet` | Protected wallet. |
| POST | `/topup/orders` | Protected create order, rate limited. |
| GET | `/topup/orders/:orderCode` | Protected order detail. |

## Player

| Method | Path |
| --- | --- |
| GET | `/player/status/:id` | Projects only equipped/vital Gus. |
| GET | `/player/inventory/gus` | Full Gu inventory (lazy-loaded). |
| GET | `/player/stats` |
| GET | `/player/passive-stats` | Detailed passive stats breakdown with per-source attribution. Không còn trả `evasion`/`accuracy` (né tránh giờ là status `EVASION` trong combat). |
| POST | `/player/explore/:id` |
| POST | `/player/complete-tutorial` | Hoàn tất giai đoạn Tân Thủ (lv0): chuyển rank 0 → Nhất Chuyển (rank 1). Idempotent với player đã rank ≥ 1 sẽ trả 400. |
| POST | `/player/complete-onboarding` | Hoàn tất thiết lập ban đầu. Body: `{"starting_region","attack_gu_id","defense_gu_id","vital_gu_choice"}`. `starting_region` gồm `hub_nam_cuong`/`hub_trung_chau`/`hub_bac_nguyen`/`hub_tay_mac`/`hub_dong_hai`; người chơi được đặt vào HUB tương ứng của Đại Vực đã chọn. |
| POST | `/player/use-item` |
| POST | `/player/use-items` |
| POST | `/player/recharge` |
| POST | `/player/clear-stress` |
| GET | `/player/profile/:username` |
| PUT | `/player/profile/settings` | Cập nhật hồ sơ (bio, avatar, mật khẩu, ngụy trang). Username không trùng nhau dù khác hoa/thường; nhập tên người chơi tại các tính năng được hỗ trợ cũng không phân biệt hoa/thường. Đổi tên khác biệt thực sự: Rank 1-2 miễn phí; Rank 3-5 tốn Nguyên Thạch (5k/20k/100k); Rank 6-9 tốn Tiên Nguyên Thạch (100/500/2.5k/10k), tụt Đạo Ngân & Chân Ý Lưu Phái (-30%/-40%/-50%/-70%), tổn hại Tiên Khiếu và phát thông báo Thế Giới (không tụt tiểu cảnh giới). |
| GET | `/player/combat-settings` | Đọc `combat_settings` (auto-heal/mana + **thứ tự ưu tiên loại** `category_priorities` + **bộ cổ sử dụng** `gu_priorities` + `km_priorities`). |
| PUT | `/player/combat-settings` | Lưu `combat_settings`. `category_priorities` = thứ tự ưu tiên loại kỹ năng (`shield`, `heal`, `damage`, `other`); `gu_priorities` = **bộ cổ mang theo** (danh sách `gu_id` theo thứ tự `priority`); `killer_move_priorities` (hoặc `km_priorities`) = **sát chiêu trang bị** trong chiến đấu (danh sách ID sát chiêu theo thứ tự ưu tiên và điều kiện kích hoạt): sát chiêu không được trang bị ở đây hoặc trong loadout trận đấu sẽ không tự động kích hoạt. |
| GET | `/player/referrals` |
| POST | `/player/referrals/claim` |
| POST | `/player/referrals/bind` |
| POST | `/player/simulate-deposit` | Admin only. |
| POST | `/player/avatar` |
| POST | `/player/redeem-gift-code` |
| GET | `/player/audit-logs` |
| GET | `/player/notifications` |
| GET | `/player/notifications/unread-count` |
| POST | `/player/notifications/read` |
| POST | `/player/notifications/read-all` |
| GET | `/player/beast-betting/current` | Lấy phiên cược đấu thú hiện tại của user. |
| GET | `/player/beast-betting/history` | Lấy lịch sử 10 trận đấu thú gần nhất và log chiến đấu. |
| POST | `/player/beast-betting/bet` | Đặt cược Nguyên Thạch vào phe Đỏ hoặc Xanh. Body: `{"team": "team_a"|"team_b", "amount": float64}`. |

## Actors

Các route protected dưới đây trả canonical Actor Instance thuộc người chơi. Sủng vật và NPC dị nhân đã dùng cùng nguồn dữ liệu này; contract của `/pets/*` và `/immortal-aperture/npcs` vẫn được giữ tương thích.

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/actors` | Trả `{ "actors": [...] }`; danh sách rỗng luôn là `[]`. |
| GET | `/actors/:id` | Trả `{ "actor": {...} }` cho actor thuộc người gọi. UUID sai trả `400`; actor không tồn tại hoặc không thuộc người gọi trả `404`. |

### Named variant NPC gameplay

Các command dưới `/immortal-aperture/npcs/:npc_id` chỉ chấp nhận NPC thuộc người gọi và còn sống. Công việc, tu luyện và chiến đấu loại trừ lẫn nhau qua `execution_state`.

| Method | Path | Ghi chú |
| --- | --- | --- |
| POST | `/formation` | Body `{ "slot": 0..8 }`; slot `0` rút khỏi đội hình, chỉ NPC `NONE/STORED` được vào đội. |
| POST | `/cultivate` | Tiêu Nguyên Thạch, tăng tiến độ deterministic theo thiên phú; tối đa Ngũ Chuyển. |
| POST | `/gu/equip` | Body `{ "gu_instance_id", "slot" }`; chuyển Cổ khả dụng từ túi chủ sang NPC. |
| DELETE | `/gu/:gu_id` | Thu hồi Cổ từ NPC về túi chủ. |

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
| GET | `/daily-missions` | Lấy 5 nhiệm vụ cá nhân theo Rank đã khóa trong ngày Việt Nam; lần đầu truy cập cũng khởi tạo bộ nhiệm vụ nếu cần. |
| POST | `/daily-missions/claim` | Nhận thưởng nhiệm vụ đã hoàn thành; body `{ "mission_id": "..." }`; mỗi nhiệm vụ nhận một lần. |

## Inventory and refining

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
| POST | `/inventory/use-support` | Dùng Cổ hồi máu ngoài combat (bất kỳ Cổ không phải Tiêu Hao có block `ON_ACTION` `HEAL`/`FLAT`); xử lý `ON_USE` và hồi HP từ `active.heal_flat`. Body: `{ "gu_instance_id": "..." }`. |
| POST | `/inventory/drop` |
| POST | `/inventory/drop-batch` |
| GET | `/killer-moves/options` | Cổ người chơi trong `player_gu` và effect combat/ON_USE được phép kéo vào công thức. |
| GET | `/killer-moves` | Danh sách sát chiêu đã lưu, độ thành thạo, trạng thái thiếu Cổ và số ô trang bị. |
| POST | `/killer-moves/preview` | Tính thử công thức. Body: `{ "main_gu_instance_id", "component_instance_ids": [], "selected_effect_keys": [] }`. Trả ngân sách, hệ số effect, độ hòa hợp, chi phí và tỉ lệ thành công. |
| POST | `/killer-moves` | Lưu sát chiêu. Body preview cộng `name`; giới hạn 12 sát chiêu lưu và 4 sát chiêu trang bị. |
| POST | `/killer-moves/:id/equip` | Bật/tắt trang bị. Mỗi Cổ thành phần phải còn đúng instance/rank trong `player_gu`. |
| POST | `/killer-moves/:id/use` | Dùng ngoài combat nếu công thức có effect hồi phục `ON_USE`; hao Chân Nguyên và mỗi Cổ thành phần giảm 1 độ no. |
| DELETE | `/killer-moves/:id` | Xóa sát chiêu của người chơi hiện tại. |
| POST | `/refine` | Luyện Cổ. Recipe `required_gus` là map `gu_id → {rank, qty}`; Cổ nguyên liệu phải khớp `gu_id` và đúng `rank` (khi `rank != 0`). Target rank ≥ 6 trùng toàn server → **409** (kể cả race thua khi concurrent). |
| POST | `/refine/deduce` | Suy diễn recipe. |
| POST | `/combat/start` | Bắt đầu mô phỏng tự động chiến đấu. Body nhận `loadout` (`GuIds[]`, `KillerMoveIds[]`, `AutoHealHP`, `HealThreshold`, `AutoRestoreMana`, `ManaThreshold`). Trả `logs[]` trong đó mỗi entry sát thương có `breakdown` (công thức: `base_damage`, `skill_mult`, `dao_mult`, `buff_mult`, `crit_mult`, `path_bonus`, `target_def`, `natural_resist`, `path_resist`, `damage_reduction`, `shield_absorbed`, `hp_damage`, `final_damage`, `raw_damage`, `hit_count`, `reason`). Kể từ nay kèm provenance trạng thái: `attacker_attack` / `defender_defense` / `healer_healing` (`{stat, base_value, final_value, mods:[{key,name,pct}]}`) và các trường hồi máu `healing_base`, `healing_dao_bonus`, `healing_final`. |
| POST | `/combat/practice` | Như `/combat/start`. |
| GET | `/combat/history` | 20 trận gần nhất của người chơi, bản tinh gọn cho list: `{id, player_id, winner, summary, enemy_name, enemy_type, region, logs_count, created_at}` (không kèm JSONB `logs`/`combatants` — lấy chi tiết qua endpoint dưới). Gồm PVP (mỗi người chơi tham chiến có bản ghi riêng), **mọi trận boss** (`BOSS`/`World_Boss`/`TRIBULATION_BOSS`, cả thắng lẫn thua), **mọi trận Tông Môn Chiến** (`SECT_WAR`, cả thắng lẫn thua, ghi cho cả người tấn công và từng đệ tử trấn thủ tham chiến với góc nhìn đảo phe) và các trận PVE **thua**; PVE thường thắng không lưu. Bản ghi PVE/boss cũ hơn 30 ngày bị worker xóa theo lô (PVP giữ vô hạn). |
| GET | `/combat/history/:id` | Full một entry: kèm `logs` và `combatants` JSONB để replay. Log `TURN_END` có snapshot cuối lượt của hai phe (HP, Chân Nguyên, khiên, công/thủ/tốc độ và `statuses[]` gồm loại, giá trị, thời hạn còn lại); các trận lịch sử cũ không có snapshot giữa lượt vẫn đọc được. |

## Exploration and combat

| Method | Path | Ghi chú |
| --- | --- | --- |
| POST | `/combat/capture` | Bắt quái vật làm sủng vật. |
| POST | `/explore` | Du hành tới zone: body `{"player_id","node_id"}` (node_id = id Đại Vực / địa danh / `hub_nam_cuong`). |
| POST | `/explore/retreat/:id` | |
| POST | `/explore/flee/:id` | Bỏ chạy khỏi trận đang chiến. **Mỏ & PvP: luôn thành công, không phạt.** Còn lại: 70% thành công (BOSS/Tribulation 40%). Thành công → trừ 20% Max HP nếu là PvE thường (có thể ngất + rơi đồ). Thất bại → địch phản công; trả `flee_success:false` kèm `defeat` / `victory` (địch tự chết → thắng tại chỗ + nhận loot + xoá địch khỏi map) và `logs`. |
| GET | `/explore/grid` | Viewport 5×5 quanh player; trả `{x, y, vision_radius, cells, stats (chỉ movement + investigation), players_on_map, ground_drops, world_width, world_height}`. `?stats_only=true` chỉ trả tọa độ, `vision_radius`, movement/investigation và world dimensions; bỏ dựng cell/Redis, players, drops cho màn hình chỉ cần range. |
| GET | `/explore/regions` | Danh sách zone (id, `world_map_id`, name, geometry `min_x..max_y`, `entry_x/y`, `parent_id`, `zone_type`, `rules`, `effective_rules`). `rules` là luật khai báo tại khu đó; `effective_rules` là luật sau khi kế thừa theo nhóm từ khu cha. Client dùng luật hiệu lực để hiển thị cấm chế; backend vẫn tự kiểm tra. Xem `docs/GAME_DATA.md` để biết schema. `zone_type`: `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC` — khi chồng lấn, ưu tiên `HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC`. |
| GET | `/explore/maps` | Danh mục tầng bản đồ (`maps[]`: id, tên, kích thước, rank, rules, trạng thái). `bai_thien` và `hac_thien` hiện được cấu hình nhưng chưa mở cho người chơi cho tới khi luồng tọa độ/hiện diện được scope theo tầng. |
| GET | `/explore/world` | Cấu hình thế giới: `width`, `height`, `large_zone_size`, `sub_zone_size`, `hub_region_id`, `hub_x/y`. |
| GET | `/explore/objects` | Các ô persistent (map_cells): village/trạm nghỉ, boss, linh tài — cho bản đồ lớn. Trả `{region_id,x,y,cell_type,content_id,rank,respawn_seconds,cleared}`. |
| GET | `/explore/cells/rect` | Các ô non-empty trong hình chữ nhật `?min_x=&min_y=&max_x=&max_y=` (mỗi cạnh được cap 256) — dùng để prefetch vùng nhìn / dải phía trước hướng đi. Client cache theo chunk 64×64 và namespace khu vực; trạng thái cleared được đọc theo batch. Trả cùng định dạng `[{x,y,cell_type,content,rank,sub_rank,region_id}]`. |
| POST | `/explore/move` | Di chuyển tới toạ độ trong thế giới (bounds = `world_width`/`world_height`). |
| POST | `/explore/move-path` | Di chuyển tự do theo đường thẳng từ ô hiện tại tới toạ độ đích, trigger cell action trên từng ô đi qua (gom loot; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss**, không có phục kích ngẫu nhiên trên ô trống). Mỗi request tiến **tối đa 150 ô** để commit vị trí mịn (người chơi khác thấy chuyển động gần real-time, đổi hướng/dừng sớm hơn); trả `{player, log, combat_triggered, reached, skip_result?}` (đã tối ưu: loại bỏ `logs` mảng và `stats` thừa, `player` là snapshot cập nhật vị trí/sinh lực/trạng thái tinh gọn, tránh query DB dư thừa); nếu `reached=false` và không combat, client chain tiếp request từ vị trí mới cho tới khi tới đích. |
| POST | `/explore/teleport` | *(Đã vô hiệu hóa)* Dịch chuyển bằng tiền đã bị hủy bỏ. Trả về `400 Bad Request` yêu cầu sử dụng Cổ trùng dịch chuyển (`POST /explore/use_gu`). |
| GET | `/explore/players` | Người chơi ở gần để hiển thị/tương tác trên bản đồ. |
| POST | `/explore/use_gu` | Dùng Cổ bản đồ (ví dụ mở cổng). Cổ dịch chuyển (`PORTAL_CREATE`/`RANDOM_PORTAL`) bị giới hạn tầm theo rank Cổ hoặc `params.range`; trả `synced_party_members` khi đội trưởng kéo tổ đội theo. |
| GET | `/world/drops` | Chiến lợi phẩm/hài cốt còn hạn trong viewport; mặc định quanh người chơi (radius 25), hỗ trợ `x`, `y`, `radius` tùy chỉnh (radius tối đa 127). Query theo bounding box tọa độ, không giới hạn theo region của người chơi. |
| POST | `/world/drops/:id/loot` | Nhặt một ground drop khi đủ điều kiện khóa sở hữu. |

## Truyền Tống Trận (Teleportation Arrays)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/teleport/templates` | Lấy danh sách các phẩm cấp Trận Pháp (Tier 1: Vi Trận, Tier 2: Đại Vực Trận, Tier 3: Tiên Trận) kèm yêu cầu chi phí và cự ly. |
| GET | `/teleport/mine` | Lấy danh sách các Trận Pháp do bản thân hoặc Tông Môn sở hữu. |
| POST | `/teleport/build` | Bố trí Trận Pháp mới tại toạ độ hiện tại. Body: `{"name","tier","direction_type","access_policy","fee_enabled","fee_currency","fee_amount"}`. |
| GET | `/teleport/directory` | Lấy danh bạ các điểm truyền tống khả dụng từ Trận Pháp mà người chơi đang đứng (lọc cự ly, hướng kết nối, quyền riêng tư, phí). |
| POST | `/teleport/travel` | Kích hoạt truyền tống đến trận pháp đích. Body: `{"target_array_id": "uuid"}`. Kiểm tra cấm chế teleport và điều kiện rank/Cổ tại khu đích; không dùng được trong bí cảnh. Trừ phí (95% vào két sắt trận, 5% hao tán), giảm 1 độ bền, áp dụng debuff Giới Vực Bích nếu nhảy xuyên vực. |
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
| POST | `/party/formation` | Trưởng nhóm xếp đội hình xuất trận. Body: `{"order": [{"kind": "player"\|"pet"\|"npc", "id": "...", "pos": 1-8}]}` (tối đa 8 sau trưởng nhóm; `pos` trùng/0 = tự xếp ô trống; rỗng = về tự động). |
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

## Đấu trường thời gian thực (Realtime PvP) — Auth

Chế độ PvP mô phỏng theo thời gian thực (kiểu MOBA): nhân vật di chuyển tự do
trong đấu trường, cổ trùng được dùng như chiêu thức có hướng/vùng, né đòn được.
Server là chủ sở hữu trạng thái (authoritative) và chạy vòng lặp tick 20Hz.

| Method | Path | Mô tả |
| --- | --- | --- |
| POST | `/pvp/match/practice` | Tạo trận luyện tập 1 người vs Hình Nhân Gỗ. Trả `Setup` (đấu trường, đội hình, kit chiêu, hằng số di chuyển). |

Trạng thái trực tiếp không trả qua HTTP: client đăng ký kênh `match:<match_id>`
và nhận `frame` mỗi tick, gửi ý định điều khiển bằng lệnh WebSocket `match.input`.

- `Setup.tick_ms` = 50; `Setup.speed_to_tiles_per_tick` và `Setup.actor_radius`
  là hằng số di chuyển server dùng, client dùng lại để dự đoán vị trí (không
  hard-code, tránh lệch).
- `frame.events` là danh sách one-shot (`cast_start`, `cast_end`, `hit`,
  `shield_absorb`, `shield`, `blink`, `death`, `match_end`) để client phát hiệu ứng.
- `frame.participants[].cast` là **vùng ngắm (telegraph)** của chiêu đang niệm, gửi
  cho mọi người trong trận: `start_tick`, `ends_tick`, `cast_mode`, `shape`,
  `radius`, `length`, `width`, `angle_deg`, `move_distance`, `aim_x`, `aim_y`,
  `target_id`. Nhờ vậy đối thủ vẽ được đúng vùng mình sắp ăn đòn. `target_id` có
  nghĩa là đòn chọn mục tiêu (không né được) — client đánh dấu nạn nhân thay vì vẽ
  vùng. Vùng ngắm vẽ bằng cùng hình học mà server dùng để resolve.
- Server kiểm tra lại mọi ý định: mode khớp `cast_mode`, hồi chiêu, mana,
  thời gian niệm, tầm chiêu và tốc độ di chuyển. Client chỉ gửi ý định.

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
| POST | `/immortal-aperture/cleanse-dao-marks` | Thanh Đạo Tẩy Thể — bóc tách Đạo Ngân ngoại hệ (body `{path, marks}`), tiêu hao Tiên Nguyên Thạch, hoàn trả 30% dưới dạng Đạo Tài (`refund_item`, `refund_qty`, `refund_marks`), cooldown 12 giờ. |
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

`POST /immortal-aperture/nodes` accepts `{node_type, x, y, path, prod_key, cells}`. `cells` is a connected array of `{x,y}` cells on the 20×20 grid; omitted `cells` means the single anchor cell. The seed cost is `ApertureNodeTemplate.seed_amount` (100) for every rank. Resource zones are capped at `min(area_size / 100000, 200)` per aperture.

`GET /immortal-aperture/nodes` returns each node's `cells`, its combat-derived `dao_conflict_multiplier` when below 1, and a server-computed `yield_preview`:

```json
"yield_preview": {
  "base": 10, "expected": 10.7, "whole": 10, "remainder": 0.7,
  "bonus": 1.07, "raw_bonus": 1.07, "capped": false,
  "per_hour": 32.1,
  "factors": [{"key": "level", "value": 1}, {"key": "dao", "value": 1}, ...]
}
```

`factors` keys are `level, dao, area, conflict, ecology, eco, labor, pop`. Yield is `max(10, 10 × bonus)`, so every rank — and every debuffed node — yields at least 10 per harvest, and the 100-seed planting cost always repays within 10 harvests. `production_amount` is legacy: the harvest formula no longer reads it.

`POST /immortal-aperture/nodes/batch` accepts `{action, node_ids}` for `repair`, `upgrade`, or permanent `delete` of up to 400 selected nodes. Repair and upgrade validate and charge the combined cost in one transaction.

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
| POST | `/trade/unlock/:id` | Mở khóa ưu đãi để chỉnh sửa lại vật phẩm/đá, tự động hủy xác nhận của cả hai bên để đảm bảo an toàn. |
| POST | `/trade/confirm/:id` |
| GET | `/merchant` | Thương nhân đứng cạnh (roaming, ≤ 2 ô) + cờ `at_village` (region HUB/SAFE_ZONE) để UI tự bật/tắt. Trả `{region, zone_type, at_village, merchants}`. |
| GET | `/merchant/map` | Toàn bộ roaming trong region (không lọc khoảng cách) để vẽ marker trên bản đồ. |
| POST | `/merchant/quote` | Body `{merchant_id, items:[{item_id,item_type,rank,quantity}]}` → `{items:[{item_id,rank,accepted,unit_price,quota_left}]}`. |
| POST | `/merchant/sell` | Bán cho thương nhân: giá động theo nhu cầu + quota toàn server. Body như quote. Hỗ trợ Cổ/Item/cổ phương. |
| POST | `/merchant/buy` | Mua từ `merchant_stock`. Body `{merchant_id, items:[{item_id,rank,quantity}]}`. Hỗ trợ Cổ/Item/cổ phương (mua cổ phương = mở khóa). |

**Thương nhân (2 loại + gating):** (1) **Cố định** tại HUB/SAFE_ZONE = Caravan/Thương Hội: bán random theo giờ, mua tất cả 1/10 (`village_general`, quota vô hạn) — **UI chỉ mở khi `at_village`**. (2) **Lang thang**: mỗi con có bể mua/bể bán riêng (allow-list + giá + quota riêng), mua/bán cả Cổ/Item/cổ phương, **UI chỉ mở khi đứng cạnh (≤ 2 ô)**, marker 🪙 hiện trên bản đồ qua `/merchant/map`. `/shop/sell` + `/sell-bulk` delegate sang `village_general`. Chi tiết: `docs/GAME_DATA.md` §"Thương nhân".

**Quy định phí & thời gian Bảo Hoàng Thiên:**
- **Cửa hàng (`MARKET`):** Phí Upfront thu ngay lập tức khi đăng lên kệ = **4% giá trị niêm yết** (`price * quantity`), khấu trừ theo loại tiền tệ của đơn hàng (Nguyên Thạch / Tiên Nguyên Thạch). Phí hoa hồng khi bán thành công = **6%** doanh thu.
- **Đấu giá (`AUCTION`):** Phí hoa hồng khi đấu giá thành công = **15%** giá thắng thầu. Thời gian đấu giá giới hạn từ **1 đến 24 giờ** (dưới 1 ngày). Không được hủy đơn đấu giá khi đã có người đặt giá (`highest_bidder_id`).

**Loại tiền theo rank (Thương Đội):** `/shop/list` trả thêm `rank` và `currency` cho mỗi món hàng; `/shop/buy` trừ và `/shop/sell` / `/shop/sell-bulk` trả theo loại tiền tương ứng rank của món (item/Gu/bí phương): **rank < 6 → Nguyên Thạch (`primeval_stones`), rank ≥ 6 → Tiên Nguyên Thạch (`immortal_stones`)**. Quy tắc tập trung ở `pkg/pricing.StoneTierByRank`.

**Rank cho Cổ đa rank:** `/shop/buy` nhận thêm `rank` (body, mặc định 0 = tự suy variant rank thấp nhất); `rank > 0` mua đúng variant đó. `/shop/sell` và `/shop/sell-bulk` cũng nhận thêm `rank` cho từng món. `caravan_shop_items` có khóa `(id, rank)`, `rank = 0` = auto.

**Xoay vòng hàng theo giờ:** `is_default = true` luôn hiển thị; món `is_default = false` được shuffle deterministic theo khung giờ 1h (seed = Unix giây bucket UTC) theo tỷ lệ **~40 Cổ / ~60 item**. Ngoài ra mọi **Cổ trùng rank < 6 và rarity < 6** cùng **thức ăn của chúng** được tự thêm vào pool ngẫu nhiên (không persist, không có trong danh sách admin/export). Giá món random = `base_price × 1.5`; mỗi món có **stock hằng ngày theo rank** (R1=100, R2=50, R3=20, R4=5, R5=2, R6+=0), lưu trong `caravan_shop_daily_stock`, `/shop/buy` trừ stock và từ chối khi hết. Frontend hiển thị đếm ngược tới khung kế tiếp.

**Admin shop** (thêm/sửa/xóa theo rank): `PUT /admin/shop/:id/:rank`, `DELETE /admin/shop/:id/:rank`; body POST/PUT chứa `rank` (0 = auto).

## Chat/mail, soul and instances

| Method | Path |
| --- | --- |
| POST | `/chat/mail/send` | Gửi tin riêng 1:1, tối đa 500 ký tự. Sender chịu 1 Nguyên Thạch/tin; tin đầu tiên giữa một cặp người chơi thu thêm 1 Nguyên Thạch mở kết nối. Kết nối dùng chung hai chiều, lưu lâu dài. |
| GET | `/chat/mail/inbox` |
| GET | `/chat/mail/conversations` | Danh sách 100 hội thoại gần nhất, tin cuối và số tin chưa đọc. |
| GET | `/chat/mail/thread?username=...` | 50 tin gần nhất của cuộc trò chuyện, đánh dấu tin nhận được là đã đọc; mỗi tin có `sender_rank` để tô màu tên người gửi. |
| POST | `/chat/bao-hoang/send` | Gửi tin nhắn chat. Trả về `{ message }`; thông tin player (Nguyên Thạch, Nhân Khí) đồng bộ qua WebSocket `player.updated`. |
| GET | `/chat/bao-hoang/messages?channel=world\|general\|sect\|region` | Mỗi tin có `sender_rank` để giao diện tô màu tên theo rank. `region` tự lấy khu vực và instance hiện tại từ player; `scope_only=1` chỉ trả khóa WebSocket hiện tại để đồng bộ đăng ký. Chat khu vực miễn phí và không nhận region do client chỉ định. |

| POST | `/soul/refine` |
| GET | `/instance/active` |
| GET | `/instance/list` | Danh sách phụ bản cũ + `cooldown_remaining` (giây) của player |
| POST | `/instance/enter` |
| POST | `/instance/action` |
| GET | `/instance/spatial` | Danh sách template `SPATIAL_REALM`, số người đang ở phiên chung và cổng cố định (`entry_region_id`, `entry_x`, `entry_y`) |
| GET | `/instance/spatial/active` | Bí cảnh bản đồ mà người chơi đang tham gia, nếu có |
| POST | `/instance/spatial/enter` | Body `{ "template_id": "..." }`; chỉ tạo hoặc tham gia phiên chung khi player đứng tại đúng `ENTRY_PORTAL` của template, lưu điểm hồi quy riêng cho người vào |
| POST | `/instance/spatial/exit` | Thoát tại ô cổng có `cell_type=EXIT_PORTAL`/`PORTAL_EXIT`, `content_id=EXIT_SPATIAL_REALM`, hoặc `metadata.action=EXIT_SPATIAL_REALM` |
| GET | `/dream-realms` | Danh sách Mộng Cảnh (`DREAM_REALM`), số lượt thám hiểm trong ngày và thời gian hồi chiêu |
| GET | `/dream-realms/active` | Phiên Mộng Cảnh đang diễn ra của người chơi (nếu có) |
| POST | `/dream-realms/enter` | Tiến vào Mộng Cảnh. Body: `{ "instance_id": "..." }` |
| POST | `/dream-realms/action` | Chọn quyết định trong ải Mộng Cảnh. Body: `{ "option_index": 0 }` |
| POST | `/dream-realms/exit` | Thoát / rút lui sớm khỏi Mộng Cảnh đang diễn ra |
| GET | `/spatial-realms` | Danh sách Bí Cảnh Không Gian & Truyền Thừa (`SPATIAL_REALM`, `SPATIAL_INHERITANCE`), hỗ trợ phân loại `scope` (`SHARED`, `PARTY`, `SOLO`) và `rules` JSONB |
| GET | `/spatial-realms/active` | Bí cảnh bản đồ hiện tại của người chơi (kèm thông tin phiên, thời hạn `expires_at`, quy tắc `rules`) |
| GET | `/spatial-realms/altar-info` | Tra cứu chi tiết Đài Truyền Thừa (`altar_id`), điều kiện khảo nghiệm (cảnh giới, Đạo Ngân, tín vật, trạng thái thủ hộ thú, đã nhận hay chưa) |
| POST | `/spatial-realms/enter` | Tiến vào Bí Cảnh / Truyền Thừa. Body: `{ "template_id": "..." }`. Kiểm tra Rule Engine (cảnh giới, áp chế, tín vật tiêu hao `rules.entry_item`, tổ đội theo `scope=PARTY`) |
| POST | `/spatial-realms/exit` | Thoát tại ô cổng thoát của bí cảnh |
| POST | `/spatial-realms/claim-altar` | Tiếp nhận di sản Đài Truyền Thừa (`INHERITANCE_ALTAR`). Body: `{ "altar_id": "..." }`. Kiểm tra khoảng cách, diệt Thủ hộ giả, điều kiện Đạo Ngân, cảnh giới, tín vật, chống nhận lặp (độc bản Rank 6+ qua `player_inheritance_claims`). Cấp phát Cổ trùng, phương thuốc, nguyên thạch, đạo tài. |

`POST /chat/bao-hoang/send` nhận `channel=region` để gửi miễn phí vào khu vực hiện tại. Lịch sử khu vực được cô lập theo `current_region` và `map_instance_id`; kênh cập nhật theo vị trí mới khi frontend đồng bộ khóa hiện tại. Mỗi lần tải chỉ trả tối đa 50 tin gần nhất.

Bí cảnh bản đồ và Vùng truyền thừa (`SPATIAL_REALM`, `SPATIAL_INHERITANCE`) vận hành theo mô hình Hybrid Scope:
- **`SHARED`**: 1 phiên chung toàn server per template. Mọi người chơi cùng vào 1 bản đồ.
- **`PARTY`**: 1 phiên riêng cho từng tổ đội (`party_id`), kiểm tra số lượng thành viên tối thiểu/tối đa.
- **`SOLO`**: 1 phiên riêng cho từng cá nhân (`owner_id`).
- **Rule Engine (`rules` JSONB)**: `duration_minutes` (hết hạn worker dọn), `pvp_mode` (`SAFE` cấm PK / `FREE_PK`), `max_allowed_rank` & `apply_rank_suppression`, `allow_teleport` (cấm truyền tống khi đang ở bí cảnh), `death_penalty` (`EVICT` / `SAFE_RESPAWN`), `entry_item` & `entry_item_count`. Ground drops được cô lập per `map_instance_id`.

## Gacha (Cổ Từ / Mở Đá)

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/gacha/pools` | Danh sách gói mở: `{id, name, description, cost_type, cost_amount}` (bảng reward + trọng số chỉ nằm server-side). |
| POST | `/gacha/open` | Mở N lần. Body: `{"pool_id": "...", "quantity": 10}` (`quantity` mặc định 1, tối đa 100; ≥ 10 được giảm 10% chi phí). Trúng Tiên Cổ đã tồn tại → **409**. |
| POST | `/gacha/open10` | Tương đương `/gacha/open` với `quantity: 10` (giữ để tương thích). |

Pool hiện có: `da_thuong` (2.000 Nguyên Thạch), `da_vip` (1.000 Tiên Nguyên Thạch), `da_thuong_pham` (20.000 Nguyên Thạch) và `da_cuc_pham` (100.000 Nguyên Thạch). Gacha **không còn thưởng bí phương**; phần thưởng gồm Nguyên Thạch/Tiên Nguyên Thạch, nguyên liệu (Material/Food) và Cổ trùng. Rate limit: 5 lần/phút/người chơi; frontend tự khoá nút ~12s sau mỗi lần mở và bỏ qua 429 im lặng.

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
| POST | `/sects/donate/recipe` | Quyên góp một Bí phương đã học. Body: `{"recipe_id": "..."}`. Nhận điểm cống hiến theo hạng Cổ trùng mục tiêu (tra `refining_recipes`, không parse id). Response kèm `unlocked_recipes` để client làm mới danh sách.
| POST | `/sects/donate/bulk` | Quyên góp hàng loạt linh tài, cổ trùng, bí phương và Nguyên Thạch. Body: `{"materials": [{"material_id": "...", "amount": 5}], "gus": [{"gu_instance_id": "..."}], "recipes": [{"recipe_id": "..."}], "stones": 100}`. Chat Tông Môn gộp 1 tin `[Quyên Góp]`. `recipe_id` trùng hoặc không còn trong túi đồ bị bỏ qua (không rollback cả đơn) và trả về `skipped_recipes`; response kèm `unlocked_recipes`.
| POST | `/sects/vault/buy` | Đổi vật phẩm từ bảo khố. Body: `{"item_type": "gu|material|recipe", "item_id": "...", "quantity": 5}` (`quantity` mặc định 1; recipe chỉ nhận 1). Học bí phương trả kèm `unlocked_recipes` (đọc sau commit) và phát realtime domain `recipes`.
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
| POST | `/sects/vault/stock` | Nhập kho 1 loại vật phẩm (không cho phép item_type Consumable, Currency). Body: `{"item_type": "material|recipe", "item_id": "...", "quantity": 5}`. |
| POST | `/sects/vault/stock/bulk` | Nhập kho hàng loạt 1 giao dịch (không cho phép item_type Consumable, Currency). Body: `{"items": [{"item_type": "material|recipe", "item_id": "...", "quantity": 5}]}`. |
| POST | `/sects/vault/destroy` | **Chưởng Môn / Hộ Pháp** tiêu huỷ vĩnh viễn **cả stack** một vật phẩm trong bảo khố (không hoàn quỹ). Body: `{"item_type": "gu|material|recipe", "item_id": "...", "rank": 1}` (`rank` bắt buộc khi `item_type=gu`). |
| POST | `/sects/upgrade` |
| POST | `/sects/toggle-approval` | Bật/tắt duyệt thành viên gia nhập. Body: `{"require_approval": true}`. |
| POST | `/sects/toggle-promotion-approval` | Bật/tắt duyệt lên cấp (mặc định `false`). Khi `true`, mọi thăng cấp đều tạo yêu cầu chờ Trưởng Lão/Chưởng Môn duyệt. Body: `{"require_promotion_approval": true}`. |
| GET | `/sects/join-requests` |
| POST | `/sects/join-requests/:requestId/action` |
| GET | `/sects/promotion-requests` |
| POST | `/sects/promotion-requests/:requestId/action` |
| GET | `/sects/mines` | Danh sách Lãnh địa + phòng thủ (`defenders`), `total_defense_power`, `output_per_hour`. |
| POST | `/sects/attack` | Khiêu chiến Tông Môn (SECT_WAR) tại tọa độ tông môn. Body: `{"target_sect_id": "..."}`. Hộ Sơn Đại Trận sẵn sàng sẽ hủy lần công kích đầu, báo động toàn tông và bảo hộ tập hợp 10 phút; kỹ năng hồi 12 giờ. Trường hợp này trả `409` với `formation_cancelled`, `shield_until`, `cooldown_until`. Thắng: cướp Quỹ/linh tài/Cổ trùng/Bí phương kho tông chia cho đội (một phần bị Thiên Đạo tiêu hủy); sơn môn thất thủ bị cưỡng chế lưu vong đến tọa độ ngẫu nhiên không đè lấn trong Đại Vực và nhận 1 giờ chấn chỉnh sơn môn. Thua: đệ tử trấn thủ nhận cống hiến, bảo hộ 4 giờ. |
| GET | `/sects/relocate/estimate` | Chưởng Môn xem dự toán chi phí di dời sơn môn, kiểm tra địa thế phong thủy (không HUB/SAFE_ZONE, không đè lấn lãnh địa tông môn khác) và thời gian hồi (cooldown 72h). |
| POST | `/sects/relocate` | Chưởng Môn kích hoạt đại điển di dời sơn môn đến tọa độ dã ngoại hiện tại của bản thân. Khấu trừ Quỹ Tông Môn theo số thành viên & bảo khố, cập nhật tọa độ căn cứ và phát thông báo kênh Tông Môn. |
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

## Admin — identity & resource catalog

Hai endpoint sau thuộc nhóm protected `/api/admin`; chúng yêu cầu JWT/cookie hợp lệ và quyền admin:

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/me` | Trả snapshot quyền thực tế của actor từ RBAC: `{actor_id, username, roles, effective_permissions, legacy_fallback, permission_version}`. `permission_version` là marker opaque để client invalidation cache quyền. Tài khoản `is_admin` cũ chưa được gán role nhận `super_admin`, wildcard `admin.*` và `legacy_fallback=true`. Frontend dùng permission key (không dùng tên role) để bảo vệ module. |
| GET | `/admin/resources` | Trả `{resources, permission_version}` từ allowlist compile-time và chỉ gồm resource mà actor có ít nhất một quyền. Mỗi phần tử có metadata `{key, domain, label, capabilities, read_permissions, write_permissions, risk_level, cache_invalidation, registry_reload, restart_required, audit_action_prefix, frontend_route}`. Catalog không dùng `information_schema`, không trả database schema/tên bảng/cột/SQL, và không dùng resource key từ client để sinh SQL. |

Danh sách `roles`, `effective_permissions`, catalog và thứ tự resource là deterministic. `read_permissions` và `write_permissions` là các permission thực tế được route backend dùng; metadata cache/registry mô tả tác động vận hành của thao tác ghi. Hiện mọi thao tác hỗ trợ đều reload trong process nên `restart_required=false`. Client nên coi `permission_version` là giá trị opaque và tải lại quyền/catalog khi marker thay đổi.

## Admin — combat history & map drops

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/combat-history` | Phân trang và tìm kiếm lịch sử chiến đấu (PVP, mọi trận boss thắng/thua, mọi trận Tông Môn Chiến và PVE thua); `search` khớp cả `enemy_type` nên gõ `boss` để lọc trận boss. Danh sách gọn, không tải logs lớn. |
| GET | `/admin/combat-history/:id` | Chi tiết một bản ghi gồm summary, combatants và logs để replay, cùng snapshot cuối mỗi lượt cho trạng thái và chỉ số hai phe. Chỉ đọc. |
| GET | `/admin/map-drops` | Phân trang map drops, tìm kiếm và lọc còn hạn/đã hết hạn. |
| PUT | `/admin/map-drops/:id` | Cập nhật vị trí, người chơi, loot JSON và thời hạn; được audit. |
| DELETE | `/admin/map-drops/:id` | Xóa drop với lý do và xác nhận target. |
| POST | `/admin/map-drops/:id/return` | Trả toàn bộ loot về người rơi đồ (loser) rồi xóa drop; cần lý do, xác nhận target và `Idempotency-Key`. |

## Admin — players & gus

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/admin/players` | List người chơi server-side với `page`, `page_size` (`limit` vẫn được chấp nhận), `search`, `search_by=all|id|username|email|item|gu`, `sort`, `direction`; sort chỉ nhận allowlist. Khi `search_by=all` auto-detect UUID. Response gồm `data`, `total`, `page`, `page_size`. **Không trả về `email`** (dữ liệu nhạy cảm); tìm kiếm/filter theo email vẫn hoạt động. |
| GET | `/admin/players/online` | List người chơi online. |
| GET | `/admin/players/:id` | Chi tiết người chơi. Response bỏ `gu_inventory`/`materials_inventory`, trả `gu_count` + `item_count`. Kèm `race`, `gender`, `constitution`, `statuses` (array). **Không trả về `email`** (dữ liệu nhạy cảm); PUT vẫn nhận `email` để sửa. |
| GET | `/admin/players/:id/items` | Lazy-load item của người chơi từ `player_inventory_items`. Trả thêm các cổ phương đã mở khóa (`category=recipe`) từ `player_recipe_progress`. Params: `page`, `limit`, `q` (match item_id/item_name theo registry). |
| GET | `/admin/players/:id/gus` | Lazy-load từ bảng `player_gu`. Params: `page`, `limit`, `q` (match gu_id/instance_id theo registry). |
| PUT | `/admin/players/:id` | Sửa thông tin gameplay của người chơi. Body chỉ gửi field thay đổi. Username được kiểm tra trùng không phân biệt hoa/thường. Trường tương thích `is_admin` là **read-only** và không còn được endpoint này xử lý; dùng endpoint roles riêng để thay đổi quyền. Hỗ trợ `race`, `gender`, `constitution` (string), `statuses` (array string hoặc array object `{id, expires_at?, source?, value?}`, ví dụ `["severely_wounded"]` — key phải tồn tại trong `status_dictionary`). |
| GET | `/admin/roles` | Danh mục role quản trị, sắp theo `level`: `{roles: [{key,name,description,level}]}`. |
| GET | `/admin/players/:id/roles` | Role hiện tại của người chơi: `{player_id, roles: [...]}`. `is_admin` ở các API player cũ chỉ là projection read-only để tương thích. |
| PUT | `/admin/players/:id/roles` | Thay toàn bộ role bằng body `{roles: ["admin", "super_admin"]}`. Thao tác và audit `{before,after}` nằm trong cùng transaction. Không được tự nâng quyền, cấp role cao hơn role cao nhất của actor (trừ super admin), sửa role cao hơn thẩm quyền, hoặc xóa super admin cuối cùng (`409`). |
| GET | `/admin/gus/search` | Tìm cổ trùng trên toàn server. Params: `gu_id` và/hoặc `instance_id` (match registry ID + tên, ILIKE), `page`, `limit`. Quét 3 nguồn: `players.gu_inventory`, `auction` (`item_type='Gu'`), `caravan_shop_items` + `arena_shop_items`. `location`: `inventory`/`auction`/`shop`. |
| POST | `/admin/players/:id/items` | Cần `players.inventory.grant`. Thêm item hoặc cổ phương (Công thức) cho người chơi. Body: `{item_id, quantity}`. `item_id` là id item trong `Item.csv` hoặc id recipe trong `Recipe.csv`. Nếu là recipe → mở khóa (unlock) Cổ phương cho người chơi (`player_recipe_progress`, `completion_rate=100`). Trả `kind` = `item`/`recipe`. |
| DELETE | `/admin/players/:id/items` | Xóa item hoặc cổ phương khỏi người chơi. Body: `{item_id, quantity}`. Nếu là recipe → thu hồi (delete) khỏi `player_recipe_progress`. Trả `kind` = `item`/`recipe`. |
| POST | `/admin/players/:id/gus` | Cần `players.inventory.grant`. Thêm gu cho người chơi. Body: `{gu_id, quantity, rank?}`. `rank` tùy chọn — khi gửi sẽ cấp đúng bậc đó (validate theo `gu_dictionary`); khi bỏ trống dùng bậc cao nhất của `gu_id`. |
| DELETE | `/admin/players/:id/gus/:instance_id` | Xóa gu theo instance_id. |
| GET | `/admin/players/:id/pets` | Lazy-load sủng vật của người chơi. Params: `page`, `limit`, `q` (search monster name/ID). |
| DELETE | `/admin/players/:id/pets/:pet_id` | Xóa vĩnh viễn sủng vật. |
| PUT | `/admin/players/:id/stones` | Đổi đá (stones) của người chơi. |
| PUT | `/admin/players/:id/ban` | Cần `players.ban`. Khóa tài khoản: body `{reason, duration_hours?, permanent?, chat_mute?, chat_mute_hours?, block_device?}`. `permanent=true` → khóa vĩnh viễn; ngược lại phải có `duration_hours > 0`. Đặt `is_banned`/`ban_reason`/`banned_until`, xoay `active_session_id` (kick phiên đang chạy), gửi notification `session.kicked`. `block_device=true` (mặc định frontend) chặn IP/fingerprint đăng nhập gần nhất. `chat_mute=true` hạn chế chat kèm theo. |
| PUT | `/admin/players/:id/unban` | Cần `players.ban`. Mở khóa tài khoản (`is_banned=false`), đồng thời xóa device block khớp IP/fingerprint đăng nhập gần nhất. |
| PUT | `/admin/players/:id/chat-mute` | Cần `players.ban`. Hạn chế gửi chat/mail: body `{reason, duration_hours?, permanent?}` → `is_chat_muted`/`chat_muted_until`. |
| PUT | `/admin/players/:id/chat-unmute` | Cần `players.ban`. Bỏ hạn chế chat. |
| GET | `/admin/audit-logs` | Audit log server-side (bao gồm `admin.update_player`), hỗ trợ `page`, `page_size`, `search`, allowlisted `sort`, `direction`, `action`, `entity_type`, `player_id`, `success`; trả total count. |

### Admin — nhóm quản trị còn lại

Mọi route dưới đây đều thuộc `/api/admin`, yêu cầu auth, rate limit nhóm và permission RBAC theo domain. `GET /api/admin/me` yêu cầu `admin.dashboard.access`; wildcard `admin.*` cấp toàn bộ quyền cho `super_admin`. Handler vẫn có thể kiểm tra principal quản trị như lớp phòng vệ bổ sung.

Permission nhạy cảm gồm: `players.read`/`players.update`, `game_data.read`/`game_data.import`/`game_data.import_all`/`game_data.export`, `monitoring.read`/`monitoring.reset`, `cache.reload`/`cache.flush`, và `audit.read`. `game_data.import` chỉ cho phép một dataset; import toàn bộ có mức rủi ro cao hơn và bắt buộc `game_data.import_all`. Các domain quản trị còn lại dùng permission `<domain>.read` cho GET và `<domain>.update` cho thao tác ghi; inventory chuẩn được khai báo cùng route trong `backend/internal/server/server.go`.

Các endpoint commit import và `POST /admin/clear-redis-cache` bắt buộc body JSON có `reason` khác rỗng. Frontend khai báo permission, risk, confirmation, reason và re-authentication trong action policy; khi backend trả 403 kèm `permission`, client rollback optimistic state (nếu có), tải lại `/admin/me` rồi thông báo quyền đã thay đổi.

#### Workflow import CSV

| Method | Path | Permission | Mô tả |
| --- | --- | --- | --- |
| POST | `/admin/data/import` | `game_data.import_all` | Preflight read-only toàn bộ CSV trên server. |
| POST | `/admin/data/import/:table` | `game_data.import` | Preflight một dataset; nhận multipart field `file` tùy chọn, nếu thiếu dùng CSV trên server. Upload tối đa 10 MiB. |
| POST | `/admin/data/import/commit` | `game_data.import_all` | Commit preflight toàn bộ với JSON `{preflight_id, reason}`. |
| POST | `/admin/data/import/:table/commit` | `game_data.import` | Commit preflight một dataset với JSON `{preflight_id, reason}`. |

Preflight parse/validate CSV, chạy UPSERT trong transaction rồi rollback và trả `plan` gồm `totals`/từng dataset với `rows`, `inserts`, `updates`, `unchanged`, diff mẫu, `validation_errors`, cờ reload/restart và `can_commit`. Lỗi nghiêm trọng đặt `can_commit=false` và không ghi database. `preflight_id` hết hạn sau 15 phút, chỉ dùng một lần, gắn với admin/phạm vi/hash nguồn. Commit chạy lại cùng nguồn trong một transaction, ghi audit cùng transaction, sau đó reload registry và invalidate cache; response nêu rõ `reload_ok`, `reload_warnings` và `restart_required`.

| Nhóm | Routes hiện có |
| --- | --- |
| Tiên khiếu | `GET /apertures`, `GET/PUT /apertures/:id`; CRUD node và xem/tạo/xóa quần thể dị nhân theo aperture. |
| Cổ/pet/history | cập nhật cá thể Cổ; tạo/cập nhật pet; `GET /gu-history`; `GET /batch-metrics`. |
| Môn phái | list/detail/update/delete sect; thêm/xóa/đặt chưởng môn; CRUD mission; nhập/xóa hàng kho; quản lý sect mine. |
| Dữ liệu & cache | `GET /data/tables`, workflow preflight/commit import toàn bộ hoặc `/:table`, export và clear application/Redis cache. |
| World/activity | CRUD region, map cell, monster spawn, dungeon; list/update/reset mine slot. |
| Dictionary/shop | CRUD shop, arena shop, Gu, item, recipe, monster, loot table, constitution, race, status và quest. |
| Hệ thống | gift code list/create, upload image, top-up order, wallet transaction và monitoring/traces. |

`GET /api/admin/data/tables` trả cả `tables` (danh sách key tương thích ngược) và
`datasets`. Mỗi phần tử `datasets` là nguồn sự thật cho Admin UI, gồm `key`,
`label`, `csv_file`, capability `import`/`export`/`preflight`, permission tương
ứng, `ui_supported`, cùng khai báo `cache_invalidation` và `registry_reload`.
UI chỉ hiển thị dataset/các nút thao tác khi capability, UI support và permission
được công bố; dataset chưa có implementation không được trả về danh sách này.

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
| GET | `/admin/monitor/admin-operations` | Counter trong memory cho vận hành admin: 403 theo permission, mutation lỗi theo action, destructive action, import failure, audit write failure, legacy `is_admin` fallback và permission resolution error. Yêu cầu `monitoring.read`; counter reset khi process restart hoặc gọi endpoint reset. |
| GET | `/admin/monitor/requests/traces` | Tra cứu server-side các request gần nhất (Ring Buffer 500 requests). Hỗ trợ `page`, `page_size`, allowlisted `sort`, `direction`, `method`, `status` (`200`, `500`, hoặc `errors`), `path`, `min_duration_ms`, `search`; trả `{data,traces,total,page,page_size}` (`traces` là alias tương thích). |
| GET | `/admin/gu-history` | Lịch sử Cổ trùng server-side: `page`, `page_size`, `search`, `event_type`, allowlisted `sort`, `direction`; trả `logs`, `total`, `page`, `page_size`. |
| GET | `/admin/monitor/requests/traces/:id` | Xem thông tin chi tiết một request cụ thể qua `trace_id` (method, path/query đã lọc, status, latency, `response_bytes`, client IP, player ID/username, user-agent, error, `content_length`, JSON `request_body` đã redact cùng `body_truncated`, và `response_body` tối đa 8 KiB cùng `response_body_truncated`). Các trace chỉ lưu trong memory và hiển thị tại tab **Giám sát** của Admin. |
| POST | `/admin/monitor/requests/reset` | Đặt lại toàn bộ request/admin-operation metrics và xóa danh sách request trace gần nhất. |

## Realtime Communication — WebSocket `GET /api/ws` (Auth)

WebSocket endpoint chính thức thay thế SSE, sử dụng Redis Pub/Sub đa phiên bản phục vụ mở rộng horizontal. Kết nối yêu cầu token qua header `Authorization: Bearer <token>` hoặc query parameter `?token=<token>`.

### Các kênh hỗ trợ (Channels):
| Channel | Direction | Mục đích |
| --- | --- | --- |
| `player` | S→C | Snapshot & cập nhật chỉ số người chơi thời gian thực; payload snapshot có `version` để client loại bỏ event/response cũ |
| `chat:world` | Hai chiều | Chat thế giới |
| `chat:general` | Hai chiều | Chat Bảo Hoàng Thiên (yêu cầu Thông Thiên Cổ & Thần Niệm Cổ) |
| `chat:sect` | Hai chiều | Chat tông môn |
| `chat:region:<region_id>[:<map_instance_id>]` | Hai chiều | Chat khu vực miễn phí; server chỉ chấp nhận đăng ký đúng scope hiện tại của player. |
| `chat:private:<player_id>` | Hai chiều | Thông báo tin nhắn 1:1 chỉ cho chủ player ID đó; server tự đăng ký và chặn subscribe ID người khác. |
| `trade` | S→C | Trạng thái giao dịch realtime |
| `world:pos` | S→C | Vị trí người chơi di chuyển trên bản đồ lớn; payload mang `region_id` và fan-out theo AOI cùng khu vực |
| `world:cell` | S→C | Sự kiện Boss/tài nguyên bị tiêu diệt hoặc hồi sinh; payload mang `region_id`, client apply theo tọa độ global kể cả khi viewport cắt qua ranh giới khu vực |
| `combat` | S→C | Biến động chiến đấu realtime |
| `match:<match_id>` | S→C | Khung hình (frame) mô phỏng đấu trường thời gian thực, 20Hz. Server chỉ gửi cho người chơi có mặt trong trận đó. |
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
| `match.input` | `{seq, move:{dx,dy}, casts:[{gu_id,rank,mode,ax,ay,target_id}]}` | Không có (chỉ WebSocket) |

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

## Đại Hội Võ Thuật (Martial Arena) — Auth

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| GET | `/martial-arena/current` | Danh sách bảng đấu theo cảnh giới của kỳ đang diễn ra + tham gia của bản thân. |
| GET | `/martial-arena/bracket` | Bảng đấu (nhiều bracket, mỗi rank 1 cái) kèm các trận 1v1. |
| GET | `/martial-arena/matches` | Danh sách trận của người chơi. |
| GET | `/martial-arena/matches/:id/log` | Chiến báo (Combat Log Replay) của một trận 1v1. |
| GET | `/martial-arena/history` | Lịch sử các kỳ đại hội đã kết thúc. |
| GET | `/martial-arena/rewards` | Danh sách phần thưởng của bản thân. |
| POST | `/martial-arena/claim-reward` | Nhận phần thưởng (nguyên thạch, tiên nguyên thạch, item, danh hiệu). |

> Đại Hội Võ Thuật tự gom người online theo cảnh giới (`players.rank`) mỗi **Thứ 3 12:00 VN**: mỗi rank có ≥ 8 người online thì mở 1 bảng đấu 1v1 loại trực tiếp (tối đa 128 người), seed theo `sub_rank`. Không có giai đoạn đăng ký.

## Trinh Sát & Suy Diễn Thiên Cơ (Scouting & Divination) — Auth

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/scouting/inspect` | Trinh sát cục bộ mục tiêu. Bắt buộc cùng vị trí/FOV (hoặc cùng Party/Sect). Body: `{"target_id", "target_username"}`. Response: `PlayerProfileResponse` + `delta_score` + `my_score` (full ≥30, partial ≥−10). |
| POST | `/scouting/divine` | Suy diễn thiên cơ từ xa. Ưu tiên điểm: **Cảnh giới lưu phái > Cổ trùng > Đạo ngân**. Body: `{"target_username", "suspected_region", "suspected_sect", "suspected_gu", "item_clue_id", "resource_type", "resource_amount"}`. Mọi clue đều validate đúng-thật mới tính điểm (sai = 0). Response thêm `my_score` (điểm của người dò; điểm thủ không lộ). |
| POST | `/scouting/divine-gu` | Suy tính cổ trùng (ai nắm giữ, đã luyện ra chưa, số lượng thiên hạ, bí phương & thức ăn). Việc đếm bao gồm túi/tiên khiếu người chơi, kho tông môn, niêm yết chợ/đấu giá **còn hạn**, và chiến lợi phẩm trên bản đồ **chưa tan biến**; response tách `player_count`, `sect_vault_count`, `market_count`, `ground_drop_count`. Body: `{"gu_query", "target_rank", "resource_type", "resource_amount"}`. Điểm thô chia cho divisor theo bậc Cổ (R1-5: 1/2/3/4/5, R6-9: 10/18/30/50); khó còn lại = che giấu chủ mạnh nhất ×0.5 (Cổ kho/ngủ ×0.5 nữa; tự sở hữu cộng hưởng +250, bỏ qua def chủ) + màn sương 15. Response thêm `my_score` (kèm `attain_base`, `rank_divisor`). |

#### Công thức điểm trinh sát

Hằng số ở `backend/internal/gamefx/balance.go` (mục `7. TRINH SÁT / SUY DIỄN THIÊN CƠ`), engine ở `internal/gamefx/scouting_engine.go`.

```
stat     = BaseRankPower(rank) + 5.0 × Σ INVESTIGATION        // Cổ trùng (Cổ rank n → +5n)
realmDao = stat × (AttainMult[L] × DaoMult(m) − 1) + AttainBase[L]
clue     = clamp(±40, clue thô × tín) + min(resBonus, 25% core)
Δ_n      = (P_raw − D_raw) × (10 / BaseRankPower(rank_mục_tiêu))^0.6
```

| Bậc cảnh giới | 0 | 1 | 2 | 3 | 4 |
| --- | --- | --- | --- | --- | --- |
| `AttainBase` (phẳng) | 0 | 50 | 140 | 280 | 470 |
| `AttainMult` | ×1.00 | ×1.05 | ×1.12 | ×1.20 | ×1.30 |

`DaoMult(m) = 1 + 0.20·m/(m+3000)` — bão hòa, trần **+20%** stat (chỉ rank ≥ 6).

**Thứ tự ưu tiên** (mức tăng tối đa mỗi trụ, R7 + 4 Cổ R9): cảnh giới **+578** · cổ trùng **+360** · đạo ngân **+68**.

**Ngưỡng kết quả** (`/divine`, `/inspect`): full Δ ≥ +30 · partial Δ ≥ 0 · fogged Δ ≥ −30 · backfire Δ < −30.
**Ngưỡng Cổ** (`/divine-gu`): full ≥ +20 · partial ≥ −10 · fogged ≥ −40.

**Phản phệ** chỉ trừ **15% max HP và 15% max Soul** (`ScoutingBackfireRatio`) cộng 20 Chân Nguyên cố định — không còn công thức tuyệt đối `|Δ|×rank×5` nên không thể bị giết chết tức thời.

`ScoreDetail` trả về: `stat`, `realm_dao`, `clue`, `luck_roll`, `total`, `attain_base` (base cảnh giới phẳng, đã nằm trong `realm_dao`), `rank_divisor` (chỉ `/divine-gu`).
| POST | `/scouting/apply-mark` | Gieo ấn ký truy tung lên mục tiêu (Trí Niệm / Huyết Mạch). Giới hạn 3 (phàm nhân) / 10 (tiên nhân). Body: `{"target_id", "mark_type"}`. |
| GET | `/scouting/tracked-targets` | Lấy danh sách con mồi đang mang ấn ký kèm tọa độ thời gian thực `(region, x, y)`. |
| POST | `/scouting/sever-clues` | Đoạn tuyệt thiên cơ, chém đứt toàn bộ ấn ký đang theo dõi và tàn dư ô bản đồ. Tốn 20 Chân Nguyên (cố định). |
| POST | `/scouting/resonance-scan` | Cảm ứng Cổ trùng và xung kích Đạo ngân (Dao Pulses) trong khu vực. Tốn 50 Chân Nguyên (cố định) + 200 Nguyên Thạch, hoặc TNT với Tiên nhân. |
### Effect catalog Block v2 (Admin)

- `POST /api/admin/effect-catalog/validate` validates `{ "source": "gu|item|race|constitution|status", "key": "...", "effects": { ... } }` without writing. Definitions are block-only and require `schema_version: 2` plus `blocks`; legacy namespaces and unknown fields are rejected.
- `GET /api/admin/effect-catalog/metadata` returns the canonical trigger, target, operation, stat, status, resource, unit, stack-policy, condition, and native-code catalogs used by the validator. Admin editors must use this metadata instead of maintaining a separate hard-coded vocabulary.
- `GET /api/admin/effect-catalog/audit` audits Gu, Item, Race, Constitution, and Status definitions in the database.
- Validation responses contain `source`, `row` (for CSV), `key`, `block_id`, `operation_index`, `path`, `error_code`, and `message`. CSV import returns HTTP 400 with the same `issues` payload and completes validation for the whole effect-bearing batch before the first database write.
- The older `/api/admin/gu-effects/validate` and `/api/admin/gu-effects/audit` route names remain as aliases during the endpoint migration; validation payloads follow the strict Block v2 contract above.

### Admin mutation audit contract

Every `POST`, `PUT`, `PATCH`, and `DELETE` route below `/api/admin` declares a canonical
`admin.*` audit action. The server writes an audit event for successful and failed
handler/authorization outcomes. Events contain the actor ID and username snapshot,
effective roles and required permission, target type and ID, recursively redacted
request/response before-and-after data, reason, trace/request ID, client IP, result, sanitized HTTP error, and UTC
timestamp. If callers omit `reason`, the server records the declared action as the
operational reason. `X-Admin-Reason` may be used for requests without a JSON body.

Audit payloads never persist credential fields whose names identify passwords, tokens,
cookies, OTPs, authorization/CSRF values, card verification values, or payment/client/API
secrets. Multipart bodies are not captured.

### Hợp đồng an toàn cho mutation quản trị nhạy cảm

Các thao tác flush Redis, commit bulk import, xóa dữ liệu, điều chỉnh stones/tài nguyên và gán role dùng permission riêng. Mutation nhạy cảm phải gửi `reason` (hoặc `X-Admin-Reason`) và `confirmation_target` (hoặc `X-Admin-Confirmation-Target`) khớp chính xác target của route. Những thao tác có thể retry (flush Redis, commit import, đặt số dư tuyệt đối và thay role) phải gửi `Idempotency-Key` không rỗng, tối đa 128 ký tự; key được ghi vào audit; cùng actor/route/key và payload sẽ nhận lại response đã lưu trong vòng đời process, còn payload khác nhận `409`. Redis flush và gán role còn yêu cầu `reauth_password` của actor; giá trị bí mật này luôn bị redact khỏi audit.

| Thao tác | Permission | Confirmation target | Re-auth | Idempotency-Key |
|---|---|---|---|---|
| Flush Redis | `cache.flush` | `Redis` | Có | Bắt buộc |
| Commit toàn bộ / một dataset | `game_data.import_all` / `game_data.import` | `all` / key dataset | Không | Bắt buộc |
| Xóa dữ liệu | `<domain>.delete` | đường dẫn API đầy đủ của target | Không | Không (client phải đọc kết quả trước khi retry) |
| Đặt stones/tài nguyên player | `players.resources.adjust` | player ID | Không | Bắt buộc |
| Khóa / mở khóa / hạn chế chat player | `players.ban` | player ID | Không | Không |
| Thay role player | `admin_roles.assign` | player ID | Có | Bắt buộc |
| Payment mutation tương lai | phải dùng `billing.mutate`, không dùng `billing.read` | order/transaction ID | Có | Bắt buộc |

Mọi route mutation vẫn đi qua audit middleware trước permission middleware, vì vậy cả kết quả thành công lẫn thất bại/permission denied đều tạo audit record. Các thay đổi nhiều bảng (bulk import và role assignment) ghi success audit trong cùng transaction; nếu audit hoặc cập nhật thất bại, transaction rollback.

## Admin — Boss thế giới (runtime)

Boss là actor runtime trong Redis, không phải bảng dữ liệu: chúng chỉ tồn tại khi spawn đang active (`world:monsters:bosses:{region}` + `world:monster:{region}:{instance}`), được materialize từ `map_cells` (`cell_type='boss'`) và `monster_spawns.is_boss_spawn=true`. Các route dưới đây cần permission `world_bosses.*`. Mỗi spawn chỉ có tối đa một actor sống: actor trùng `spawn_id` (do spawn pointer hết hạn trước actor) được dọn tự động khi đọc set boss.

| Method | Path | Permission | Ghi chú |
| --- | --- | --- | --- |
| GET | `/admin/world-bosses` | `world_bosses.read` | Liệt kê boss đang sống toàn server; query `region_id` để lọc. Trả `{data, total, boss_regions}`, mỗi phần tử có `region_id`, `instance_id`, `spawn_id`, `monster_template_id`, `name`, `rank`/`sub_rank`, `x`/`y`, `spawn_x`/`spawn_y`, `hp`/`max_hp`, `state`, `target_id`, `respawn_seconds`, `respawn_at`, `revision`, `last_update_at`; `boss_regions` là các khu vực có ô boss tĩnh (kể cả chưa spawn). Redis không khả dụng → `503`. |
| POST | `/admin/world-bosses/spawn` | `world_bosses.update` | Body `{region_id?, x?, y?}`. Tạo actor từ ô boss (`cell_type='boss'`) của khu vực (hoặc đúng ô `x,y`); bỏ trống `region_id` để chạy toàn bộ khu vực trong một lần; hồi sinh boss đang `DEFEATED` tại chỗ, xoá cửa sổ hồi sinh (`map_cells.last_cleared_at`) và broadcast `monster_spawned`/`monster_respawned`. Trả `{created, revived, alive, errors}`. |
| PUT | `/admin/world-bosses/:region_id/:instance_id` | `world_bosses.update` | Body `{x?, y?, hp?, state?}`. Toạ độ phải nằm trong khu vực; `hp` được clamp `[0, max_hp]`. `hp=0` → `DEFEATED` + `respawn_at`; `state="IDLE"` → hồi sinh tại điểm spawn với đầy HP và xoá `respawn_at`. Ghi bằng CAS theo `revision` như worker (xung đột trả `409`), broadcast `monster_moved`/`monster_updated`/`monster_defeated`/`monster_respawned`. Boss không còn trong Redis → `404`. |
| DELETE | `/admin/world-bosses?region_id=` | `world_bosses.delete` | Xoá mọi boss đang sống của khu vực khỏi Redis (snapshot kế tiếp materialize lại từ dữ liệu tĩnh). Bắt buộc `reason` + `confirmation_target`; được audit là destructive. |

## World monster runtime

`GET /api/world/monsters?r=25` returns the authoritative ordinary-monster snapshot for the authenticated player's current region and AOI. `r` defaults to and is capped at 25, while clients normally send their actual 10–25-cell investigation radius. A snapshot materializes intersecting non-boss `monster_spawns` on demand; each result has a fresh `instance_id`, stable `spawn_id`, position, HP, state, revision, and server timestamps.

The client should send its actual investigation radius (currently 10–25 cells).
The server groups runtime monsters into 32×32-cell buckets and activates only
the bucket neighborhood intersecting active players; the separate 300-cell
world-position broadcast radius is not used as the monster activation radius.

`POST /api/explore/use_gu` accepts stable monster targets:

```json
{"command_id":"uuid","gu_instance_id":"uuid","target":{"kind":"monster","id":"monster-instance-uuid"}}
```

The server validates region, authoritative distance, life state, mana, hunger, cooldown and command idempotency. Runtime revision conflicts return HTTP 409. The targeted WebSocket channel `world:combat` carries `event_id`, `server_time`, `region_id`, `entity_id`, `entity_revision`, `event_type`, `position`, and `payload`. Event types include `monster_spawned`, `monster_moved`, `monster_targeted`, `action_started`, `action_resolved`, `entity_damaged`, `status_changed`, `monster_returned`, `monster_defeated`, `monster_despawned`, `monster_respawned`, and `monster_updated` (admin runtime edits).
