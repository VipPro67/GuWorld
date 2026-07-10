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
