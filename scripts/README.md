# Scripts Quản Lý Game Data (GuWorld)

Thư mục `scripts/` chứa các bộ công cụ Python tự động hóa để tái tạo, đồng bộ và kiểm tra tính toàn vẹn của dữ liệu trong `backend/data/`.

---

## 1. Danh sách các Script

| Script | Chức năng chính | File tác động |
| :--- | :--- | :--- |
| `gen_path_items.py` | Tạo 792 item mới (396 Dải A + 396 Dải B) cho 44 lưu phái từ Rank 1–9. Giữ lại các đan dược Pill và vật phẩm tiền tệ cần thiết. | `backend/data/Item.csv` |
| `sync_gu_food.py` | Thiết kế thức ăn (food) cho toàn bộ Cổ theo 792 item mới (Rank $R$ ăn Dải A + Dải B Rank $R$, và Dải B Rank $R-1$). | `backend/data/Gu.csv` |
| `rebuild_recipes.py` | Xây dựng lại toàn bộ công thức luyện Cổ (`Recipe.csv`). Ưu tiên Cổ cùng ID/stem ở Rank $R-1$ $\rightarrow$ cùng Lưu phái $\rightarrow$ Luyện đạo/Khí đạo. Tiên Cổ (Rank $\ge 6$) không trùng lặp (`qty == 1`). | `backend/data/Recipe.csv` |
| `sync_droploot_coverage.py` | Bổ sung các item Dải A & B vào loot table quái vật và khu vực để đảm bảo 100% item đều có nguồn rơi rớt trong game. | `backend/data/LootTable.csv` |
| `adjust_game_prices.py` | Cân bằng giá Cổ, Item và phí Recipe theo Rank chuẩn (`GuRankScaling.md`, `pkg/pricing`). Hỗ trợ chọn lọc (`--target`), bảo vệ tiền tệ và phân cấp Nguyên Thạch / Tiên Nguyên Thạch. | `Item.csv`, `Gu.csv`, `Recipe.csv` |
| `balance_gu_effects.py` | Cân bằng sức mạnh hiệu ứng Cổ theo chuẩn quy mô Rank (`P(rank)`). Giới hạn ngân sách sát thương DoT, phủ 100% các key thưa bằng `HARD_REFS` và sửa lỗi chỉ số bất thường. | `backend/data/Gu.csv` |
| `overhaul_spawns_and_loot.py` | Tối ưu toàn diện Monster Spawn & Loot Tables: phân loại lại Boss/Elite/Normal, sinh quái bổ sung cho các phái/rank thiếu, giới hạn spawn (1–4 khu vực), thu gọn loot table (2–3 entries quái thường, 4–5 entries boss), đảm bảo 0 item chỉ rớt từ Boss. | `Monster.csv`, `MonsterSpawn.csv`, `LootTable.csv`, `RegionLootTables.csv` |
| `balance_mapcell_bosses.py` | Cân bằng lại Boss thế giới trong `MapCell.csv`: gán Boss phù hợp 100% với Lưu phái và Rank của từng khu vực, cập nhật metadata rank và đồng bộ hóa với `MonsterSpawn.csv`. | `MapCell.csv`, `MonsterSpawn.csv` |
| `normalize_monster_stats.py` | Chuẩn hóa chỉ số Base của toàn bộ Quái vật và Boss trong `Monster.csv` về dải chuẩn Rank-Independent. Giảm Base DEF (0.06 – 0.60) để triệt tiêu tình trạng DEF bị nhân phóng đại tại Rank 6 khiến Cổ Rank 5 không thể xuyên thủng. | `Monster.csv` |
| `verify_game_data.py` | Kiểm tra toàn diện toàn bộ data: khớp ID, thức ăn hợp lệ, công thức hợp lệ, Tiên Cổ duy nhất, 100% droploot coverage. | Toàn bộ CSV |

---

## 2. Thứ Tự Chạy Khi Cần Tái Tạo / Cân Bằng Lại

Khi bạn muốn làm mới hoặc sinh lại toàn bộ hệ thống item, thức ăn, công thức và droploot, hãy chạy theo thứ tự sau:

```bash
# Bước 1: Sinh 792 item dải A & B và làm sạch Item.csv
python3 scripts/gen_path_items.py

# Bước 2: Đồng bộ thức ăn của Cổ theo item mới
python3 scripts/sync_gu_food.py

# Bước 3: Tạo lại công thức luyện Cổ với logic thừa kế & Tiên Cổ duy nhất
python3 scripts/rebuild_recipes.py

# Bước 4: Tối ưu quái vật, phân bố spawn cân bằng và thu gọn loot table
PYTHONPATH=scripts python3 scripts/overhaul_spawns_and_loot.py --apply

# Bước 5: Kiểm tra toàn vẹn dữ liệu
python3 scripts/verify_game_data.py

# Bước 6: Chạy kiểm thử Go Backend
cd backend && go test ./internal/db/...
```

---

## 3. Lưu ý khi nạp dữ liệu vào Game
Theo kiến trúc của GuWorld:
- CSV không tự động nạp khi khởi động server.
- Để cập nhật dữ liệu vào database, vào giao diện **Admin Panel** $\rightarrow$ Tab **Dữ liệu** $\rightarrow$ bấm **Import** tương ứng cho từng bảng (Item, Gu, Recipe, LootTable, v.v.).
- Khởi động lại backend server để cập nhật các Registry in-memory.
