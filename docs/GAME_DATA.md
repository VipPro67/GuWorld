# Game Data

Nguồn dữ liệu chính nằm trong các file CSV ở `backend/`. Khi backend khởi động, `db.InitSchema()` tạo bảng cần thiết và seed/upsert dữ liệu vào database.

## CSV seed files

| File | Số dòng dữ liệu | Nội dung |
| --- | ---: | --- |
| `backend/Gu.csv` | 227 | Dictionary Cổ: rank, path, type, mana cost, effects, hunger, food, targeting, rarity, role/tags. |
| `backend/Item.csv` | 136 | Vật phẩm, nguyên liệu, tiền tệ, consumable, effects/tags/description. |
| `backend/Recipe.csv` | 31 | Công thức luyện Cổ: target, Cổ/nguyên liệu yêu cầu, success rate, cost. |
| `backend/Monster.csv` | 13 | Monster templates: chỉ số, Gu inventory, loot table, AI profile. |
| `backend/LootTable.csv` | 8 | Bảng loot cho combat/monster. |
| `backend/MonsterSpawn.csv` | 1 | Rule spawn monster theo region/toạ độ/rank. |
| `backend/Region.csv` | 3 | Vùng map: kích thước, mô tả, rank range. |
| `backend/ShopItem.csv` | 15 | Shop NPC/caravan items. |
| `backend/PremiumShopPackage.csv` | 4 | Gói premium shop mua bằng ví. |
| `backend/TopupPackage.csv` | 5 | Gói nạp tiền. |

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
- `aperture_hp`, `max_aperture_hp` — HP của Tiên Khiếu
- `area_size` — diện tích grid, quyết định số node tối đa
- `immortal_essence`, `essence_production_per_year` — Tiên Nguyên
- `rank_start_year` — năm bắt đầu rank hiện tại (dùng cho auto-breakthrough)
- `earth_calamity_count`, `heavenly_tribulation_count`, `grand_tribulation_count`, `myriad_tribulation_count`, `chaos_calamity_count` — bộ đếm kiếp nạn
- `state` — STABLE / DAMAGED

### Bảng `dao_marks`

Lưu Đạo Ấn theo dạng quan hệ (song song với JSONB `players.dao_marks`):
- `player_id` (UUID, FK → players)
- `path` (VARCHAR) — đường tu
- `amount` (INTEGER)
- `UNIQUE(player_id, path)`

### Bảng `aperture_resource_nodes`

Node tài nguyên đặt trong Tiên Khiếu:
- `aperture_id` (UUID, FK → immortal_apertures, CASCADE DELETE)
- `node_type`, `name`, `path` — loại node
- `level`, `hp`, `max_hp` — cấp và HP
- `x`, `y` — toạ độ grid (0-19)
- `production_type`, `production_amount`, `production_interval_hours` — sản lượng
- `is_destroyed` — trạng thái hỏng

## Nhóm dữ liệu gameplay

### Cổ

Các trường quan trọng trong dictionary:

- `id`, `name`, `rank`, `path` (mảng JSON các Đạo học), `gu_type`.
- `base_mana_cost`.
- `effects` JSON: active/passive/combat/support tuỳ Cổ.
- `hunger_rate`, `food_types`, `max_hunger`.
- `target_type`: `SINGLE`, `ROW`, `COLUMN`, `ALL`, v.v.
- `rarity`, `is_immortal`, `primary_role`, `combat_roles`, `combat_tags`, `support_tags`.

### Item

Item dùng cho consumable, material, currency và shop. Effects hỗ trợ hồi mana/HP, soul, đạo ngân, xóa trạng thái xấu, điều kiện mortal/immortal hoặc các tác động đặc thù khác.

### Recipe

Recipe yêu cầu Cổ instance và/hoặc material. Một số recipe cần suy diễn bằng `/api/refine/deduce` trước khi luyện ổn định.

### Monster/combat

Monster template chứa chỉ số, Gu inventory, item inventory, dao marks, path masteries, AI profile và loot table. Combat handler dùng dữ liệu này để khởi tạo trận, chọn hành động và roll loot.

### Region/map

Region định nghĩa map lớn. `map_cells` có thể được generate theo vùng/toạ độ và lưu lại khi người chơi khám phá.

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

## Lưu ý khi thay đổi schema

- `CREATE TABLE IF NOT EXISTS` toàn bộ ở `backend/internal/db/db.go`. Không có migration riêng.
- `ALTER TABLE ADD COLUMN IF NOT EXISTS` dùng cho các cột mới thêm vào sau.
- Thêm cột mới vào schema + thêm ALTER TABLE, restart server để áp dụng.
