# Game Data

Nguồn dữ liệu chính nằm trong các file CSV ở `backend/`. Khi backend khởi động, `db.InitSchema()` tạo bảng cần thiết và seed/upsert dữ liệu vào database.

## CSV seed files

| File | Số dòng dữ liệu | Nội dung |
| --- | ---: | --- |
| `backend/Gu.csv` | 594 | Dictionary Cổ: rank, path, type, mana cost, effects, hunger, food, targeting, rarity, role/tags. Chuẩn scale sức mạnh theo rank xem `docs/GuRankScaling.md`. |
| `backend/Item.csv` | 229 | Vật phẩm, nguyên liệu, tiền tệ, consumable, effects/tags/description. |
| `backend/Recipe.csv` | 31 | Công thức luyện Cổ: target, Cổ/nguyên liệu yêu cầu, success rate, cost. |
| `backend/Monster.csv` | 30 | Monster templates: chỉ số, Gu inventory, loot table, AI profile, bắt sủng vật. |
| `backend/LootTable.csv` | 25 | Bảng loot cho combat/monster (thức ăn cổ rank<6 được thêm vào loot quái Normal/Elite theo dải rank). |
| `backend/MonsterSpawn.csv` | 48 | Pool spawn monster theo zone/toạ độ/rank (Roaming/Random). |
| `backend/Region.csv` | 31 | Zone lồng nhau (1 HUB + 6 Đại Vực 3×2 + 7 địa danh): geometry, rank range, pool_weights. `resources` của mỗi vùng bao gồm thức ăn cổ rank<6 khớp dải rank vùng. |
| `backend/MapCell.csv` | 29 | Ô persistent dùng chung: boss, tài nguyên cao, điểm an toàn. |
| `backend/ShopItem.csv` | 61 | Shop NPC/caravan items. |
| `backend/ArenaShopItem.csv` | 26 | Shop đấu trường (tiền ARENA_COIN). |
| `backend/PremiumShopPackage.csv` | 4 | Gói premium shop mua bằng ví. |
| `backend/TopupPackage.csv` | 5 | Gói nạp tiền. |

`PremiumShopPackage.csv` và `TopupPackage.csv` nằm ở `backend/cmd/server/data/` và được **đọc trực tiếp từ CSV mỗi lần request** (không seed qua Admin; sửa CSV là đổi gói). Loader `handlerCSVPath` tìm theo thứ tự `data/` → `cmd/server/data/` → legacy paths.

## Registry runtime

- `models.LoadGuRegistry(db.DB)` load `gu_dictionary` vào `models.GuRegistry` khi server start.
- `models.LoadItemRegistry(db.DB)` load `item_dictionary` vào `models.ItemRegistry` khi server start.
- API `/api/dictionary` trả dữ liệu dictionary cho frontend.
- API `/api/recipes` trả recipe refining cho frontend.

## Cấu trúc cơ sở dữ liệu

### Bảng `immortal_apertures` (Tiên Khiếu)

Tạo ngay tại thời điểm người chơi thăng tiên lên Rank 6. `grade` được roll theo Nhân Khí lúc thăng tiên (LOW / MID / HIGH / EXTREME). Các cột chính:
- `time_rate` (DOUBLE PRECISION, DEFAULT 10) — gia tốc thời gian nội bộ
- `grade` (VARCHAR) — LOW / MID / HIGH / EXTREME
- `aperture_hp`, `max_aperture_hp` (BIGINT) — HP của Tiên Khiếu (BIGINT để tránh tràn số ở cấp độ cao)
- `area_size` — diện tích grid, quyết định số node tối đa
- `immortal_essence`, `essence_production_per_year` — Tiên Nguyên
- `rank_start_year` — năm bắt đầu rank hiện tại (dùng cho auto-breakthrough)
- `earth_calamity_count`, `heavenly_tribulation_count`, `grand_tribulation_count`, `myriad_tribulation_count`, `chaos_calamity_count` — bộ đếm kiếp nạn
- `state` — STABLE / DAMAGED

### Đạo Ngân (Đạo Ấn)

Đạo Ngân chỉ lưu trong cột JSONB `players.dao_marks` (ví dụ `{"strength": 100, "fire": 2500}`). Bảng quan hệ `dao_marks` đã bị xóa — không còn mirror. Path được normalize dạng lowercase (không có hậu tố `" path"`) khi lưu.

### Cảnh Giới Lưu Phái — cột `players.path_masteries`

Cột `path_masteries` (JSONB) lưu **điểm tích lũy** dạng `map[string]int`, giống cột `dao_marks` của `immortal_apertures` — ví dụ `{"fire": 1250, "water": 340}`. Cảnh giới (cấp độ 0-4) được suy ra từ mốc điểm qua `gamefx.PathMasteryLevel(points)`:

| Level | Tên | Hệ số | Mốc điểm |
|-------|-----|-------|----------|
| 0 | Ordinary (Phàm Thai) | ×1.0 | < 100 |
| 1 | Master (Đại Sư) | ×1.1 | ≥ 100 |
| 2 | Grandmaster (Tông Sư) | ×1.25 | ≥ 500 |
| 3 | Great Grandmaster (Đại Tông Sư) | ×1.4 | ≥ 2.500 |
| 4 | Supreme Grandmaster (Vô Thượng Đại Tông Sư) | ×1.5 | ≥ 10.000 |

Điểm nhận từ: sát chiêu thành công (+2/path hạch tâm), luyện cổ thành công (+2, +10 nếu Cổ Tiên rank ≥ 6), thưởng Chân Ý phó bản (cộng thẳng `amount`). Dữ liệu cũ lưu tên cấp được migration tự động đổi sang mốc điểm khi khởi động.

**Giới hạn điểm Chân Ý mỗi trận:** các nguồn trong trận chiến thường (đánh tay, đánh tự động/Simulator, sát chiêu) bị chặn ở `gamefx.PathMasteryBattleGainLimit(rank)` = **Rank × 10 điểm/trận**. Counter lưu trong `players.active_combat.path_mastery_gained`, tự reset khi trận mới bắt đầu. Thưởng từ phó bản (attainment) và luyện cổ không tính vào giới hạn này.

### Phó Bản — bảng `instance_templates`

- `type` — `INHERITANCE_RIGHTEOUS`, `INHERITANCE_DEMONIC`, `DREAM_REALM` (Mộng Cảnh), `SECRET_REALM` (Bí Cảnh).
- `cooldown_hours` — số giờ chờ trước khi vào lại phó bản (mặc định 24; `0` = không giới hạn). Cooldown tính từ session gần nhất của player cho instance đó.
- `rewards` JSONB gồm `attainment` (Chân Ý → `players.path_masteries`), `dao_marks` (Đạo Ngân → `immortal_apertures.dao_marks`), `materials`, `primeval_stones`.
- Seed (`SeedInstanceTemplates`): truyền thừa `dungeon_flower_wine`, **mộng cảnh data-driven từ `data/DreamRealm.csv`** (gồm `dream_realm_star_constellation` + **49 mộng cảnh** `dream_realm_<path>`, 1 cho mỗi lưu phái, thưởng +50 Chân Ý cho đúng path đó, vào bằng `dream_material`, cooldown 24h) và **3 bí cảnh** (`SECRET_REALM`, thưởng Đạo Ngân, cooldown 48h).
- `DreamRealm.csv` — cột `id, name, path, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours`. Cột `path` là key lưu phái (vd `fire`); nếu `map_data` để trống, stage tree sẽ được sinh tự động theo path (`buildDreamRealmMapData`). Muốn viết cốt truyện riêng cho một mộng cảnh, điền `map_data` (JSON array stage, xem format mộng cảnh thủ công `dream_realm_star_constellation` trong file). Import/export qua Admin tab "Dữ liệu" → bảng "Mộng cảnh".
- `player_instance_sessions` — session đang chạy; index bán phần `player_active_session_idx` đảm bảo 1 phó bản ACTIVE/player.

**Cảnh giới là điều kiện thăng tiên rank:** để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, người chơi cần ≥ 1 lưu phái đạt Tông Sư (500)/Đại Tông Sư (2.500)/Vô Thượng (10.000). Kiểm tra trong `tribulation.go` `handleSuccess`.

### Tiêu hao Chân Nguyên / Tiên Nguyên khi dùng Cổ

Công thức tính tiêu hao nằm ở `gamefx/energy.go` (`EnergyCost` / `EnergyCostForCombatant`):

- **Phàm nhân (Rank 1–5):** chi phí = `(BaseManaCost/100) × MaxMana` (mặc định 10% MaxMana). Cổ rank thấp hơn giảm chi phí ÷10/rank; **Cổ rank cao hơn bị nhân hệ số lũy tiến**.
- **Cổ Tiên (Rank ≥ 6):** chi phí = `BaseManaCost` (mặc định 1). Cổ dưới Rank 6 miễn phí; Cổ rank cao hơn nhân hệ số lũy tiến; Cổ rank thấp hơn chia 100^chênh lệch.

**Hệ số rank cao hơn** (`HigherRankManaMultiplier(guRank, userRank)`, theo chênh lệch rank Cổ − rank người dùng):

- **Cổ phàm (rank 1–5):** theo bảng `{1, 2, 5, 25, 125}` cho gap `{0,1,2,3,4}` — ví dụ rank 1 dùng Cổ rank 5 (gap 4) tốn **125×** Chân Nguyên.
- **Cổ Tiên (rank ≥ 6):** mỗi bậc chênh lệch nhân ×100 (`100^gap`) — rank 6 dùng Cổ rank 7 tốn ×100, rank 6 dùng rank 8 tốn ×10.000.

- `refine_mana_tier` (passive Cổ cùng rank đang trang bị) giảm 50% chi phí.
- **Trang bị:** không thể **trang bị** Cổ có rank cao hơn rank người chơi (`EquipGu` trong `player.go`) — nhưng vẫn có thể **sử dụng** Cổ rank cao hơn qua các đường dùng không yêu cầu trang bị (kèm hệ số mana lũy tiến + backlash HP trong chiến đấu). Mỗi lần equip/unequip, server **tự động tháo** các Cổ không hợp lệ trong list đang trang bị (rank > rank người chơi, loại Consumable, hoặc không có spec) và ghi audit `gu.unequip_invalid`; Cổ Bản Mệnh không bao giờ bị tự tháo.

### Nhân vật — bảng `players` (race / gender / constitution / statuses)

Các cột mô tả nhân dạng và trạng thái của người chơi, giá trị lưu bằng **key tiếng Anh**, effect được resolve tập trung trong `backend/internal/gamefx/fx.go`:

| Cột | Kiểu | Giá trị | Effect |
| --- | --- | --- | --- |
| `race` | VARCHAR(50) | Xem bảng chủng tộc bên dưới. Mặc định `human` lúc đăng ký. **Key phải tồn tại trong bảng `race_dictionary`** | Hiệu ứng đọc từ `race_dictionary.effects` (per-row) |
| `gender` | VARCHAR(20) | `male`, `female` | Chưa có effect (chỉ hiển thị) |
| `constitution` | VARCHAR(100) | Xem bảng thể chất bên dưới. **Key phải tồn tại trong bảng `constitution_dictionary`** | Hiệu ứng đọc từ `constitution_dictionary.effects` (per-row) |
| `statuses` | JSONB (`[]string`) | `severely_wounded` (Trọng Thương) | -20% attack/defense/speed |

Chọn lúc đăng ký (`race`, `gender`); `constitution`/`statuses` đổi qua Admin panel (`PUT /api/admin/players/:id`) hoặc event trong game.

**Bảng chủng tộc (race) — data-driven, seed từ `data/Race.csv` vào bảng `race_dictionary`, nạp registry trong memory (`gamefx.RaceRegistry`) khi khởi động:**

Bảng `race_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.race` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `effects` | JSONB | Các effect của chủng tộc (cùng format `effects` như thể chất bên dưới) |

Danh sách hiện tại (10 chủng tộc):

| Key | Tên Việt | Hiệu ứng |
| --- | --- | --- |
| `human` | Nhân Tộc | Không bonus đặc biệt (mặc định) |
| `fur_people` | Người Lông | +10% tỉ lệ Luyện Cổ thành công |
| `merman` | Nhân Ngư | Thủy Đạo +10% affinity, +5% sát thương Thủy Đạo |
| `feathered` | Vũ Dân | +10% tốc độ, Không Gian Đạo +5% affinity |
| `inkman` | Mặc Nhân | Trí Tuệ Đạo +8%, Hồn Đạo +5% affinity |
| `rockman` | Thạch Nhân | +15% phòng thủ, -10% tốc độ |
| `snowman` | Tuyết Dân | Hàn Băng Đạo +15% affinity, -10% sát thương Hỏa nhận |
| `variant_beastman` | Thú Nhân | +8% tấn công, +5% phòng thủ, -5% tỉ lệ Luyện Cổ |
| `dragonman` | Long Nhân | +10% tấn công, +10% phòng thủ, -5% tỉ lệ Luyện Cổ |
| `giant_sun_descendant` | Cự Dương Huyết Mạch | +5 Vận, +5% hiệu quả tu luyện |

**Thêm chủng tộc mới:** sửa `backend/cmd/server/data/Race.csv` (thêm dòng `key,name,effects`) → Admin panel tab **Dữ liệu** → Import bảng `Chủng tộc`. Nếu database rỗng, Go có `defaultRaces()` trong `backend/internal/gamefx/races.go` làm fallback để không vỡ hành vi cũ. `players.race` luôn là **1 key duy nhất** (default `human`).

**Bảng thể chất (constitution) — data-driven, seed từ `data/Constitution.csv` vào bảng `constitution_dictionary`, nạp registry trong memory (`gamefx.ConstitutionRegistry`) khi khởi động:**

Bảng `constitution_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.constitution` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `effects` | JSONB | Các effect của thể chất (format bên dưới) |

Format `effects` (JSON object, toàn bộ field đều optional):

| Key | Ý nghĩa |
| --- | --- |
| `stat_mods` | `{"attack": 0.10, "defense": 0.10, "speed": 0.10}` — % chỉ số |
| `dao_mark` | `{"fire": 0.20}` — cộng affinity Đạo Ngân theo hệ |
| `path_damage` | `{"dream": 0.15}` — +% sát thương hệ đó |
| `path_resistance` | `{"fire": 0.10}` — giảm % sát thương nhận từ hệ đó |
| `mana_regen` | % hồi Chân Nguyên (mỗi thể chất tự quyết, VD `1` = +100%) |
| `max_mana` | % Chân Nguyên tối đa |
| `cultivation_efficiency` | % hiệu quả tu luyện |
| `dao_mark_gain` | % Đạo Ngân nhận được |
| `luck` | Vận (số phẳng) |
| `refine` | % tỉ lệ Luyện Cổ thành công |
| `healing_received` | % healing nhận (và % hồi máu thụ động) |
| `hp_regen` | HP/giờ cộng thẳng vào hồi máu thụ động ngoài combat (số phẳng, VD `50` = +50 HP/giờ) |
| `mental_resistance` | % kháng hiệu ứng tâm lý |
| `soul_defense` | % phòng thủ hồn |
| `aperture_stress_bonus` | hệ số cộng thêm áp lực Không Khiếu mỗi lần trùng kích (VD `1` = tăng ×2) |

**Thêm thể chất mới:** sửa `backend/cmd/server/data/Constitution.csv` (thêm dòng `key,name,effects`) → Admin panel tab **Dữ liệu** → Import bảng `Thể chất`. Nếu database rỗng, Go có `defaultConstitutions()` trong `backend/internal/gamefx/constitutions.go` làm fallback để không vỡ hành vi cũ.

Danh sách hiện tại (11 thể chất — 10 Thập Tuyệt + Thuần Mộng Cầu Chân Thể):

| Key | Tên Việt | Hiệu ứng |
| --- | --- | --- |
| `solar_sun_moon_body` | Thái Nhật Dương Mãng Thể | Quang Đạo đạo ngân ×150%, +20% sát thương Quang Đạo, +50% max mana, +100% hồi mana |
| `ancient_moon_yin_desolate_body` | Cổ Nguyệt Âm Hoang Thể | Hắc Ám Đạo đạo ngân ×150%, +50% max mana, +100% hồi mana, +20% kháng hiệu ứng tâm lý |
| `northern_dark_ice_soul_body` | Bắc Minh Băng Phách Thể | Hàn Băng Đạo đạo ngân ×150%, giảm 10% sát thương Hàn Băng nhận, +50% max mana, +100% hồi mana |
| `forest_sea_reincarnation_body` | Sâm Hải Luân Hồi Thể | Thủy Đạo đạo ngân ×150%, +50% hồi phục nhận, +50% max mana, +100% hồi mana |
| `myriad_gold_brilliant_flower_body` | Vạn Kim Diệu Hoa Thể | Kim Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana |
| `blazing_glory_lightning_brilliance_body` | Viêm Hoàng Lôi Trạch Thể | Lôi Đạo đạo ngân ×150%, +20% speed, +50% max mana, +100% hồi mana |
| `great_strength_true_martial_body` | Đại Lực Chân Võ Thể | Lực Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana |
| `carefree_wisdom_heart_body` | Tiêu Dao Trí Tâm Thể | Trí Đạo đạo ngân ×150%, +30% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana |
| `great_earth_origin_body` | Hậu Thổ Nguyên Ương Thể | Thổ Đạo đạo ngân ×150%, +45% defense, +45% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana |
| `universe_great_evolution_body` | Vũ Trụ Đại Diễn Thể | Không Gian Đạo đạo ngân ×150%, +15% attack/defense/speed, +50% max mana, +100% hồi mana |
| `pure_dream_reality_seeker_body` | Thuần Mộng Cầu Chân Thể | Mộng Đạo đạo ngân ×140%, +10% sát thương Mộng Đạo, +40% max mana, +100% hồi mana |

### Cổ của người chơi — bảng `player_gu`

Cột `players.gu_inventory` (JSONB) đã được **thay thế hoàn toàn** bằng bảng quan hệ `player_gu`. Mỗi row = một Cổ instance của một người chơi:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | VARCHAR(80) PK | = `instance_id` của Cổ. Được tham chiếu bởi `killer_moves.core_player_gu_id` / `killer_move_slots.player_gu_id`. |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `gu_id` | VARCHAR(50) | FK (cùng `gu_rank`) → `gu_dictionary(id, rank)` |
| `gu_rank` | INTEGER | Cấp Cổ |
| `rank` | INTEGER | Cột legacy, giữ bằng `gu_rank` |
| `level` | INTEGER | Mặc định 1 |
| `hunger` | DOUBLE PRECISION | Độ no |
| `state` | VARCHAR(30) | ACTIVE / SLEEPING / DEAD / ... |
| `location` | VARCHAR(30) | Mặc định `INVENTORY` |
| `is_locked` | BOOLEAN | |
| `equipped` | BOOLEAN | Đang trang bị |
| `is_vital` | BOOLEAN | Bản Mệnh Cổ |
| `integration_time` | DOUBLE PRECISION | |
| `equipped_at` | TIMESTAMPTZ | NULL nếu chưa trang bị |
| `growth_hours` | DOUBLE PRECISION | |
| `created_at` / `updated_at` | TIMESTAMPTZ | |

- Migration `migrate_player_gu_from_inventory` (chạy một lần khi còn cột `gu_inventory`) copy toàn bộ dữ liệu cũ sang `player_gu`, sau đó `DROP COLUMN IF EXISTS gu_inventory` loại bỏ cột. An toàn chạy lặp lại.
- Cổ đang trang bị/Bản Mệnh vẫn được mirror trong cột JSONB `players.equipped_gus` (danh sách `instance_id`) để các truy vấn nhẹ (status) không cần join.
- API `GET /api/player/status/:id` chỉ trả Cổ đang trang bị/vital; toàn bộ túi Cổ lấy qua `GET /api/player/inventory/gus` (helper `db.GuInventoryAggregate` / `db.LoadPlayerGus`).
- `temp_loot.gu_inventory` và `monster_templates.gu_inventory` (JSON trong `temp_loot` / cột monster) **giữ nguyên**, không đổi.

### Bảng `aperture_resource_nodes`

Node tài nguyên đặt trong Tiên Khiếu:
- `aperture_id` (UUID, FK → immortal_apertures, CASCADE DELETE)
- `node_type`, `name`, `path` — loại node
- `level`, `hp`, `max_hp` — cấp và HP
- `x`, `y` — toạ độ grid (0-19)
- `production_type`, `production_amount`, `production_interval_hours` — sản lượng
- `is_destroyed` — trạng thái hỏng

### Bảng `tribulation_events`

Lưu các sự kiện Kiếp Nạn của Tiên Khiếu:
- `id` (UUID PRIMARY KEY)
- `aperture_id` (UUID FK → `immortal_apertures(id)`)
- `player_id` (UUID FK → `players(id)`)
- `tribulation_type` (VARCHAR) — EARTH_CALAMITY / HEAVENLY_TRIBULATION / GRAND_TRIBULATION / MYRIAD_TRIBULATION / CHAOS_CALAMITY
- `path` (VARCHAR) — Đường hướng kiếp nạn (ví dụ: fire, water, ice)
- `difficulty` (INTEGER)
- `status` (VARCHAR) — ACTIVE / SUCCESS / FAILED
- `aperture_hp_before` (BIGINT) — HP Tiên Khiếu trước kiếp nạn
- `aperture_hp_after` (BIGINT) — HP Tiên Khiếu sau kiếp nạn (NULL nếu chưa kết thúc)
- `reward_dao_marks` (BIGINT) — Số Đạo Ngân thưởng
- `started_at` / `ended_at` (TIMESTAMPTZ)

### Bảng `tribulation_enemies`

Lưu các quái vật sinh ra trong Kiếp Nạn Tiên Khiếu:
- `id` (UUID PRIMARY KEY)
- `event_id` (UUID FK → `tribulation_events(id)`)
- `name` (VARCHAR)
- `enemy_type` (VARCHAR) — BOSS / MINION
- `path` (VARCHAR)
- `hp`, `max_hp` (BIGINT) — HP hiện tại và tối đa của quái vật (dùng BIGINT chống tràn số ở rank cao)
- `attack`, `defense` (BIGINT) — Tấn công và phòng thủ của quái vật (dùng BIGINT chống tràn số)
- `target_node_id` (UUID NULL FK → `aperture_resource_nodes(id)`)
- `x`, `y` (INTEGER) — Toạ độ grid (0-19)
- `is_boss` (BOOLEAN)
- `is_dead` (BOOLEAN)

## Nhóm dữ liệu gameplay

### Cổ

Các trường quan trọng trong dictionary:

- `id`, `name`, `rank`, `path` (mảng JSON các Đạo học — 49 path có thể có, xem danh sách ở `docs/CANH_GIOI_DAO_NGAN.md`), `gu_type`.
- `base_mana_cost`.
- `effects` JSON: active/passive/combat/support tuỳ Cổ.
- `hunger_rate`, `food_types`, `max_hunger`. **Quy ước `hunger_rate`:** cổ có `food_types` không rỗng → `6` nếu `effects.passive` không rỗng, ngược lại `3`; cổ không có thức ăn → `0` (không đói). `max_hunger` mặc định `rank × 100`.
- `target_type`: `SINGLE`, `ROW`, `COLUMN`, `ALL`, v.v.
- `rarity`, `is_immortal`, `primary_role`, `combat_roles`, `combat_tags`, `support_tags`.
- `base_price`: **đồng đều theo rank** — 200, 500, 1.200, 2.500, 5.000, 100.000, 250.000, 600.000, 1.500.000 cho rank 1–9 (nhảy mạnh ở Tiên Cổ rank ≥ 6). Fallback trong `pkg/pricing/pricing.go`.
- **Phân phối thức ăn:** mọi item trong `food_types` của cổ rank <6 (72 item) được thêm vào (1) loot của quái **Normal/Elite** theo dải rank spawn, (2) `pool_weights.resources` của vùng non-HUB theo dải rank vùng, (3) `vault_materials` khởi đầu của 4 tông (qty 100, khớp dải rank vùng tông). Boss/Tribulation không rơi thêm thức ăn.

### Item

Item dùng cho consumable, material, currency và shop. Effects hỗ trợ hồi mana/HP, soul, đạo ngân, xóa trạng thái xấu, điều kiện mortal/immortal hoặc các tác động đặc thù khác. **Quy ước `base_price`:** đồng đều theo rank — 1, 3, 6, 13, 25, 500, 1.250, 3.000, 7.500 cho rank 1–9 (rank 1 = 1, scale tỉ lệ như Cổ); item rank 0 quy về rank 1.

### Recipe

Recipe yêu cầu Cổ instance và/hoặc material. Một số recipe cần suy diễn bằng `/api/refine/deduce` trước khi luyện ổn định.

- Cột `required_gus` là JSON map `gu_id → {rank, qty}`. `rank` là bậc Cổ nguyên liệu bắt buộc; `rank = 0` nghĩa là chấp nhận mọi bậc (chỉ dùng cho dữ liệu legacy — mọi recipe import từ CSV đều được chuẩn hóa sang rank cụ thể).
- Khi import từ CSV (`seedRecipesFromCSV`), `required_gus` được tự chuẩn hóa: value dạng số `N` cũ được chuyển thành `{rank: <base_rank>, qty: N}`, trong đó `base_rank` là bậc nhỏ nhất của Cổ đó trong `gu_dictionary` (fallback `1` nếu Cổ chưa có trong dictionary).
- Backend chỉ chấp nhận Cổ nguyên liệu khớp cả `gu_id` lẫn `rank` (khi `rank != 0`); frontend (Lò Luyện, Wiki, Admin) hiển thị và chọn Cổ theo đúng bậc yêu cầu.
- Ví dụ: `{"GU_WHITE_BOAR": {"qty": 1, "rank": 1}}`.

### Monster/combat

Monster template chứa chỉ số, Gu inventory, item inventory, dao marks, path masteries, AI profile và loot table. Combat handler dùng dữ liệu này để khởi tạo trận, chọn hành động và roll loot.

### Region / Bản đồ thế giới (World Map)

Từ phiên bản map 3×2, thế giới là **một bản đồ liên tục duy nhất** 30.000 × 20.000 ô, chia làm **6 Đại Vực (NGU_VUC)** mỗi cái 10.000 × 10.000 (lưới 3 cột × 2 hàng). Mỗi Đại Vực được chia số học thành **tiểu vực** 1.000 × 1.000 (600 tiểu vực toàn thế giới). Các địa danh có tên (KHU_VUC — giữ nguyên 7 Lưu Vực cũ) nằm lồng/chồng bên trong Đại Vực. Ngoài `Southern_Border` (HUB mặc định), mỗi Đại Vực còn có **HUB khởi đầu riêng** (`hub_thuong_gia_thanh`, `hub_trung_chau`, `hub_bac_nguyen`, `hub_tay_mac`, `hub_dong_hai`). Khi hoàn tất onboarding (`complete-onboarding`), người chơi được đặt vào HUB tương ứng với vị trí đăng ký (xem `startingRegionHubs` trong `player.go`), không sinh quái trong HUB.

**Bảng `regions`** — các cột mới:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `parent_id` | VARCHAR | zone cha (KHU_VUC trỏ tới NGU_VUC của nó). |
| `zone_type` | VARCHAR | `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC`. |
| `min_x, min_y, max_x, max_y` | INT | hình chữ nhật toàn cầu (inclusive) của zone. |
| `entry_x, entry_y` | INT | tọa độ cổng truyền tống khi du hành tới zone. |
| `pool_weights` | JSONB | pool sinh nội dung procedural của zone (xem dưới). |

**`pool_weights`** JSONB — ví dụ:
```json
{
  "cell_type_weights": { "Monster": 22, "Resource": 14, "WildGu": 5, "Empty": 59 },
  "resources": ["moon_orchid_petal", "jade_stone"],
  "wild_gus": ["GU_LITTLE_LIGHT", "GU_JADE_SKIN"]
}
```
`cell_type_weights` quyết định tỉ lệ các loại ô; `resources` / `wild_gus` là pool tài nguyên và Cổ cấp thấp. Không khai báo → dùng pool mặc định.

**Thứ tự ưu tiên zone_type** — khi một điểm nằm trong nhiều zone chồng lấn, `FindZoneAt` chọn zone có priority cao nhất (bằng → diện tích nhỏ nhất = innermost):

```
HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC
```

Zone type không nhận diện fallback về priority của `KHU_VUC`. `FindLargeZoneAt` luôn trả `NGU_VUC` chứa điểm.

**Bảng `map_cells`** — các ô **dùng chung toàn thế giới** (chỉ lưu boss + tài nguyên cấp cao + điểm an toàn). `UNIQUE(x, y)` toàn cầu. Cột mới: `last_cleared_at TIMESTAMPTZ`, `cleared_by UUID`. Khi giết boss / hái tài nguyên cao, `last_cleared_at` được set; ô sẽ trống cho tới khi hết `respawn_seconds`.

**Bảng `monster_spawns`** — giờ là **pool theo zone** (không còn ý nghĩa toạ độ cho loại Roaming/Random). `region_id` trỏ tới Đại Vực hoặc địa danh. Pool được cache trong bộ nhớ lúc start.

**Phân lớp sinh nội dung**:
- Quái nhỏ / tài nguyên thấp / Cổ cấp thấp → sinh **procedural** bằng hash seed theo `(zone, x, y, window 180s)`, không đụng DB.
- Boss / tài nguyên cấp cao / làng → ô **persistent** trong `map_cells`, nạp vào index bộ nhớ lúc start, có respawn dùng chung.

Import/Export qua Admin tab "Dữ liệu": bảng `region` (Region.csv), `map_cell` (MapCell.csv), `monster_spawn` (MonsterSpawn.csv). Sau khi import **restart backend** để nạp ZoneRegistry.

## Quy ước cập nhật dữ liệu

1. Sửa CSV khi thay đổi dictionary/seed gốc.
2. Kiểm tra schema seed tương ứng trong `backend/internal/db/db.go` nếu thêm cột mới.
3. Nếu thêm field Cổ/item mới, cập nhật struct model và frontend type trước khi dùng trong UI.
4. Cập nhật số dòng trong bảng trên nếu thêm/xóa seed data.

### Bảng `leaderboard_settings` (Bảng Xếp Hạng)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `player_id` | UUID PK | FK → `players(id)` ON DELETE CASCADE |
| `show_on_leaderboard` | BOOLEAN | DEFAULT TRUE. Người chơi có thể tắt để rút khỏi BXH |

Không có seed CSV cho bảng này. Row được tạo tự động qua `INSERT ON CONFLICT DO NOTHING` khi người chơi truy cập BXH lần đầu.

### Bảng `sects` (Tông môn)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `funds` | BIGINT | Quỹ tông môn (đổi từ INTEGER → BIGINT để tránh tràn số khi tích lũy lãnh địa). |
| `require_approval` | BOOLEAN | DEFAULT FALSE. Bật = phải duyệt đơn gia nhập. |
| `require_promotion_approval` | BOOLEAN | DEFAULT FALSE. Bật = **mọi** thăng cấp đều tạo yêu cầu chờ duyệt. |
| `vault_recipes` | JSONB | Danh sách Bí phương trong bảo khố. Định dạng `[{"recipe_id": "recipe_...", "completion_rate": 100}]`. Seed từ `Sect.csv`. |

Các cột được thêm sau khi tạo bảng bằng `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` trong `db.go`.

### Bảng `sect_missions` (Nhiệm vụ tông môn — Bảng nhiệm vụ)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `sect_id` | VARCHAR(50) FK | → `sects(id)` ON DELETE CASCADE |
| `title` | VARCHAR(100) | |
| `description` | TEXT | |
| `mission_type` | VARCHAR(30) | `GATHER`, `DONATE`, `COMBAT`, `REFINE` |
| `target_id` | VARCHAR(50) | ID mục tiêu (linh tài / Cổ) |
| `target_amount` | INTEGER | Số lượng yêu cầu |
| `reward_contribution` | INTEGER | Điểm cống hiến thưởng |
| `required_member_rank` | VARCHAR(30) | Cấp bậc tối thiểu để nhận |
| `expires_at` | TIMESTAMP NULL | Thời điểm tự hết hạn. `NULL` = không hết hạn (mặc định cho nhiệm vụ seed từ CSV). Khi tạo qua UI: mặc định `now()+24h`, nhập `-1` = vĩnh viễn, nhập `0` = dùng mặc định 24h. Worker (`sect_mission.go`) tự xóa nhiệm vụ hết hạn mỗi phút; hàng `player_sect_missions` liên quan bị xóa theo CASCADE. |

Seed từ `SectMission.csv` (append-only, `ON CONFLICT DO NOTHING`) luôn để `expires_at = NULL`.

### Bảng `player_sect_missions` (Nhiệm vụ cá nhân đang thực hiện)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `mission_id` | UUID FK | → `sect_missions(id)` ON DELETE CASCADE |
| `current_progress` | INTEGER | |
| `status` | VARCHAR(30) | `IN_PROGRESS`, `COMPLETED`, `EXPIRED` |
| `assigned_at` | TIMESTAMP | |
| `expires_at` | TIMESTAMP NULL | Sao chép từ bảng nhiệm vụ khi nhận. Worker đổi `IN_PROGRESS` → `EXPIRED` khi quá hạn. |
| UNIQUE | (player_id, mission_id) | Mỗi người chơi chỉ nhận 1 lần mỗi nhiệm vụ |

### Bảng `sect_mine_defenders` (Phòng thủ Lãnh địa)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `mine_id` | INTEGER FK | → `sect_mines(id)` ON DELETE CASCADE |
| `sect_id` | VARCHAR(50) FK | → `sects(id)` ON DELETE CASCADE |
| `defender_type` | VARCHAR(20) | `member` (đệ tử) hoặc `beast` (thần thú thuê) |
| `player_id` | UUID NULL | Đệ tử phòng thủ. UNIQUE (partial) — mỗi đệ tử chỉ giữ 1 mỏ |
| `name` | VARCHAR(100) | Tên hiển thị (snapshot) |
| `monster_id` / `monster_rank` / `monster_sub_rank` | VARCHAR(80)/INT | Dữ liệu thần thú |
| `power` | DOUBLE PRECISION | Điểm phòng thủ (tính lúc gán) |
| `assigned_at` | TIMESTAMP | |

`power` đệ tử = `(atk + def + hp/20) × (1 + 0.15×(rank−1))`; thần thú = `atk + def + hp/20` (qua `ScaleMonsterStats`).

### Bảng `sect_mines` — cột tích lũy

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `pending_funds` | DOUBLE PRECISION | Bộ tích lũy Quỹ (worker cộng theo phút, đổi sang số nguyên khi ≥ 1) |
| `pending_materials` | DOUBLE PRECISION | Bộ tích lũy Linh tài |

### Bảng `sect_members` — cột cống hiến lãnh địa

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `territory_pending` | DOUBLE PRECISION | Bộ tích lũy điểm cống hiến theo giờ |
| `territory_credited_at` | DATE | Ngày đã cộng (để reset cap) |
| `territory_credited_today` | INTEGER | Số điểm đã cộng hôm nay (cap 200/ngày) |

## Lưu ý khi thay đổi schema

- Schema được tách theo domain ở `backend/internal/db/schema_*.go`, mỗi bảng là `CREATE TABLE IF NOT EXISTS` với toàn bộ cột inline. `InitSchema()` trong `db.go` gọi các schema function theo thứ tự FK. Không có migration riêng, không dùng `ALTER TABLE ADD COLUMN`.
- Thêm/sửa bảng: sửa file `schema_*.go` tương ứng → xóa DB (hoặc `DROP TABLE` thủ công) → restart server.
- FK cho các cột `players.race` / `players.constitution` / `players.current_region`: các dòng sentinel (`''`) và vùng `Southern_Border` được pre-seed trong `InitSchema` nên đăng ký vẫn hoạt động trước khi import CSV. Các HUB khởi đầu theo Đại Vực (`hub_*`) nằm trong `Region.csv` và được import qua Admin tab "Dữ liệu"; nếu chưa import, `complete-onboarding` fallback về `Southern_Border`. Các bảng dictionary (`race_dictionary`, `constitution_dictionary`) tự seed default dataset khi DB rỗng; CSV import (Admin tab "Dữ liệu") vẫn là nguồn chính thức.

### Bảng `player_pets` (Sủng vật)

Bảng lưu sủng vật bắt được từ quái vật trong combat.

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `monster_template_id` | VARCHAR(80) FK | → `monster_templates(id)` |
| `monster_name` | VARCHAR(100) | Tên quái tại thời điểm bắt |
| `rank` | INTEGER | Rank của quái vật |
| `current_hp` | DOUBLE PRECISION | HP hiện tại |
| `max_hp` | DOUBLE PRECISION | HP tối đa |
| `attack` | DOUBLE PRECISION | Tấn công |
| `defense` | DOUBLE PRECISION | Phòng thủ |
| `hunger` | DOUBLE PRECISION | Độ no (0-100) |
| `status` | VARCHAR(20) | `active` hoặc `sleeping` |
| `formation_slot` | INTEGER | 0 = storage, 1..N = formation slot |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | Dùng cho hunger decay theo elapsed time |

Indexes: `idx_player_pets_player_id`, `idx_player_pets_player_id_template`.

### Cột bắt sủng vật trong `monster_templates`

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `is_capturable` | BOOLEAN | DEFAULT FALSE. Có thể bắt làm sủng vật không |
| `capture_chance` | DOUBLE PRECISION | Tỷ lệ bắt cơ bản (0.0-1.0). rank 1 = 0.25, rank 5 = 0.10 |
| `pet_hunger_per_hour` | DOUBLE PRECISION | Tốc độ đói mỗi giờ. Mặc định 2 |

### Gộp sủng vật (bầy) và phí Hồn Phách khi ra trận

- **Gộp bầy** qua `/api/pets/merge` (`MergePets`): chỉ gộp pet cùng `monster_template_id`, nguồn phải ở kho (`formation_slot = 0`).
- **Giới hạn bầy** = `max(1, max_soul / 10)` (`mergeSoulDivisor = 10` trong `pets.go`). Nếu `count` sau khi gộp vượt giới hạn → **chặn toàn bộ gộp** (không xóa nguồn) và trả `merge_limit`.
- **Công thức chỉ số** (sub-linear, không cộng dồn compound): lấy base `stat / count^0.65` rồi scale `base × newCount^0.65` cho HP/Atk/Def; `speed` giữ nguyên; gộp hồi đầy HP. Hệ số `packScaleExponent = 0.65`.
- **Phí Hồn Phách khi ra trận:** mỗi pet (bầy) tiêu tốn `soul_hp` = `count` (số cá thể) khi load vào trận thật (`combat_state.go`). Nếu `soul_hp < count` → **bỏ qua pet đó**. Trừ dần theo `formation_slot ASC`.
  - Áp dụng cho cả pet của **đối phương phòng thủ** khi bị tấn công PvP/mine/map (trừ `soul_hp` của họ, thiếu thì bỏ pet của họ).
  - **Võ Đài (arena, virtual) không tốn soul.**
  - Cờ `active_combat.pet_soul_charged` đảm bảo mỗi trận chỉ tính phí một lần (tránh re-init giữa trận tính lại).

### Cho sủng vật ăn (thức ăn loại Food)
- Sủng vật được cho ăn qua `/api/pets/feed` (handler `FeedPet`).
- Độ no cộng thêm lấy từ `item_dictionary.effects.hunger` (data-driven).
- **Quy ước dữ liệu:** mọi item loại `Food` có `hunger = rank × 10` (rank 1 = +10 ... rank 9 = +90), cập nhật qua `backend/cmd/server/data/Item.csv` → Import bảng "Vật phẩm". Backend fallback cùng công thức `rank × 10` nếu effects chưa có `hunger`.
- Item được tiêu hao từ bucket `consumables` hoặc `materials` trong `player_inventory_items`.

### Giá tiền theo rank (Thương Đội)

Bảng `caravan_shop_items` chỉ lưu `price` (số lượng, không lưu loại tiền). Loại tiền được suy ra từ **rank** của món hàng qua `pkg/pricing.StoneTierByRank`:

- **rank < 6** → Nguyên Thạch (`primeval_stones`)
- **rank ≥ 6** → Tiên Nguyên Thạch (`immortal_stones`)

Áp dụng cho cả **item** (`item_dictionary.rank`), **cổ trùng** (`gu_dictionary.rank`, lấy variant rank thấp nhất như `models.GetGuAny`) và **bí phương / công thức** (`refining_recipes.target_rank`):

- `/api/shop/list` trả thêm `rank` + `currency` mỗi món.
- `/api/shop/buy` trừ đúng balance theo `currency`.
- `/api/shop/sell` và `/api/shop/sell-bulk` trả về đúng loại tiền theo rank món bán (bí phương dùng `target_rank`).

**`base_price` rank ≥ 6 là Tiên Nguyên Thạch** — bảng giá mặc định (fallback trong `pkg/pricing/pricing.go`, dùng khi CSV không ghi `base_price`):

| Rank | 6 | 7 | 8 | 9 |
| --- | ---: | ---: | ---: | ---: |
| Cổ trùng | 200 | 500 | 1.200 | 2.500 |
| Vật phẩm | 10 | 25 | 60 | 150 |

Ví dụ bán-lại cho Thương Đội (10% `base_price`): Cổ rank 9 = 250 Tiên Nguyên Thạch; item rank 6 = 1 Tiên Nguyên Thạch.

### Đổi đá (exchange)

Đổi giữa hai loại đá qua `POST /api/shop/exchange` (xem `ExchangeStones` trong `handlers/premium.go`, tỷ giá hằng số trong file):

- **1 Tiên Nguyên Thạch = 100 Nguyên Thạch** — đổi chiều `IMMORTAL_ESSENCE_STONE` → `PRIMEVAL_STONE` (`amount * 100`).
- **100.000 Nguyên Thạch = 1 Tiên Nguyên Thạch** — đổi chiều `PRIMEVAL_STONE` → `IMMORTAL_ESSENCE_STONE` (`amount` phải là bội số của 100.000, nhận `amount / 100000`).

Giao dịch đồng bộ cả `players.primeval_stones/immortal_stones` lẫn `wallets.primeval_stone/immortal_essence_stone`, ghi 2 dòng sổ cái `wallet_transactions` với `type = 'EXCHANGE'` (thêm vào CHECK constraint của bảng `wallet_transactions`; `InitSchema` tự `ALTER TABLE ... DROP/ADD CONSTRAINT` để áp dụng lên DB cũ).
