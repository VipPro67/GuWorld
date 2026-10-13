# GuRankScaling – Bảng chuẩn sức mạnh Cổ trùng theo Rank

Tài liệu này định nghĩa chuẩn scale sức mạnh (công / thủ / khiên / hồi máu / chỉ số passive) cho **effects** của từng Cổ trùng theo rank, dùng làm tham chiếu khi soạn sửa `effects` JSON trong `backend/cmd/server/data/Gu.csv`.

> Quy tắc vàng: **sức mạnh Cổ trùng nằm 100% trong `effects` JSON**, không có hệ số nhân theo rank ở tầng combat (`combat_effects.go`, `player.go`). Chỉ nên thêm helper `GuPowerScale(rank)` làm công cụ **cảnh báo khi seed/import** (lệch quá ±35% so với chuẩn), **không** nhân vào công thức chiến đấu (tránh double-scale).

## 1. Reference curves

Bảng gốc do người chơi/thiết kế cung cấp (tương ứng 9 rank):

| Đại lượng | Công thức | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Sức mạnh quái `P` | `P(rank)` | 1 | 2 | 5 | 20 | 100 | 10.000 | 1.000.000 | 100.000.000 | 10.000.000.000 |
| Chi phí năng lượng `E` (Phàm) | `E = P` | 1 | 2 | 5 | 20 | 100 | — | — | — | — |
| Chi phí năng lượng `E` (Tiên) | `100^(rank-5)` | — | — | — | — | — | 10 | 10 | 10 | 10|
| Giá base | theo bảng | 200 | 500 | 1.200 | 2.500 | 5.000 | 200\* | 500\* | 1.200\* | 2.500\* |

Ngưỡng Tiên cổ: `immortalRankThreshold = 6` (rank ≥ 6 là Tiên Cổ; trạng thái được suy ra từ rank).

\* **Rank 6–9 tính bằng Tiên Nguyên Thạch (immortal stones)** — xem `pkg/pricing.StoneTierByRank`; rank 1–5 tính bằng Nguyên Thạch (primeval stones). Đồng bộ ở `pkg/pricing/pricing.go`, `Gu.csv`/`Item.csv` (`base_price`) và mảng cứng frontend (`CaravanShop.tsx`, `SectView.tsx`).

**Giá Item theo rank** — bắt đầu rank 1 = 1, scale tỉ lệ như Cổ (≈ Gu/200): `1, 3, 6, 13, 25, 10, 25, 60, 150` cho rank 1–9 (item rank 0 quy về rank 1 = 1; **rank 6–9 tính bằng Tiên Nguyên Thạch**). Fallback trong `pkg/pricing/pricing.go` và mảng cứng frontend (`CaravanShop.tsx`, `SectView.tsx`) phải khớp 2 bảng trên.

**hunger_rate theo passive** — cổ **có thức ăn** (`food_types` không rỗng): `hunger_rate = 6` nếu `effects.passive` không rỗng, ngược lại `3`. Cổ **không thức ăn**: `hunger_rate = 0` (không đói).

## 2. Công thức anchor (per rank)

Mỗi thuộc tính hiệu ứng có một anchor, các cổ trùng theo vai trò được phép **±30%** quanh anchor (riêng Gu đặc biệt / huyền thoại có thể vượt trội hơn nhưng vẫn bám theo bậc rank):

| Trường effects | Công thức | Ghi chú |
| --- | --- | --- |
| `active.damage_flat` | `25 × P` | sát thương 1 đòn tiêu chuẩn |
| `active.shield_flat` | `20 × P` | lá chắn phòng thủ tiêu chuẩn |
| `active.heal_flat` | `15 × P` | hồi máu tiêu chuẩn |
| `passive.stat_base_attack` | `1 × P` | cộng tấn công nền |
| `passive.stat_base_defense` | `5 × P` | cộng phòng ngự nền |
| `passive.stat_base_hp` | `10 × P` | cộng máu nền |
| `passive.stat_base_speed` | `1` | cộng tốc độ nền | ko scale mạnh như các chỉ số khác
| DOT (`BLEED`/`POISON`/`BURN`/`SLOW_DAMAGE`) | Tổng damage mỗi lần áp dụng ≤ `60% × damage_flat chuẩn` cùng rank; `value/tick ≤ 0.6 × 25 × P / turns` | Với 3 lượt, mỗi tick bằng tối đa 20% damage FLAT chuẩn. DOT lâu hơn giảm damage/tick tương ứng. |

## 3. Bảng chuẩn rank 1–9

| Rank | `P` | dmg | shield | heal | sATK | sDEF | sHP | sSPD | `E` | Giá |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 1 | 25 | 20 | 15 | 1 | 5 | 10 | 1 | 1 | 200 |
| 2 | 2 | 50 | 40 | 30 | 2 | 10 | 20 | 2 | 2 | 500 |
| 3 | 5 | 125 | 100 | 75 | 5 | 25 | 50 | 5 | 5 | 1.200 |
| 4 | 20 | 500 | 400 | 300 | 20 | 100 | 200 | 20 | 20 | 2.500 |
| 5 | 100 | 2.500 | 2.000 | 1.500 | 100 | 500 | 1.000 | 100 | 100 | 5.000 |
| 6 | 10.000 | 250.000 | 200.000 | 150.000 | 10.000 | 50.000 | 100.000 | 1.000 | 100 | 200\* |
| 7 | 1.000.000 | 25.000.000 | 20.000.000 | 15.000.000 | 1.000.000 | 5.000.000 | 10.000.000 | 10.000 | 10.000 | 500\* |
| 8 | 100.000.000 | 2.500.000.000 | 2.000.000.000 | 1.500.000.000 | 100.000.000 | 500.000.000 | 1.000.000.000 | 100.000 | 1.000.000 | 1.200\* |
| 9 | 10.000.000.000 | 250.000.000.000 | 200.000.000.000 | 150.000.000.000 | 10.000.000.000 | 50.000.000.000 | 100.000.000.000 | 10.000.000 | 100.000.000 | 2.500\* |

> Ví dụ đối chiếu thực tế: `GU_DIAMOND_THOUGHT` (R6) `damage_flat=25.000`, `GU_FLYING_SWORD` (R7) `damage_flat=250.000`, `GU_BLOOD_DEITY` (R9) `damage_flat=250.000.000` — khớp chuẩn `25×P`. `GU_MYRIAD_SELF` (R7) `damage_flat` dùng `passive.stat_base_attack=10.000` vì phân thân nhân theo `BaseAttack` (`summon_phantom` → `PhantomDamageScale`).

## 4. Cách áp dụng khi sửa Gu.csv

1. Xác định rank, role chính (`primary_role`) và mô tả của cổ trùng.
2. Nếu mô tả **rõ tấn công** → điền `active.damage_flat` theo bảng (+ tùy chọn `damage_path`, `apply_status`, `target_selector`).
3. Nếu mô tả **rõ phòng thủ / thủ hộ / đỡ đòn** → điền `active.shield_flat` (hoặc `passive.stat_base_defense`).
4. Nếu mô tả **hồi máu / chữa trị** → điền `active.heal_flat`.
5. Passive chỉ số (`stat_base_*`) dùng khi cổ trùng "ban sức mạnh nền" / nuôi cổ / hóa thân (vd Biến Hóa Cổ, phân thân).
6. Giữ nguyên `role`, `combat_roles`, `combat_tags`, `support_tags`, `description`, `base_mana_cost` trừ khi có yêu cầu riêng.
7. Rank ≥ 6 tự động được tính là Tiên Cổ.

## 5. Kiểm chứng

- Sau khi sửa CSV: chạy `python3 scripts/balance_gu_effects.py` để so mỗi `(key, rank)` với chuẩn. Với DOT, script tự audit giới hạn tổng damage một lần áp dụng ở 60% damage FLAT chuẩn cùng rank, chia theo `turns` (mặc định 3 nếu thiếu); rule này không bị nới bởi `--factor`/`--band`. Mặc định chỉ audit, `--apply` mới ghi CSV giữ nguyên format. Với các key khác, ngưỡng lệch mặc định là `--factor 10`; `--band 0.35` dùng dung sai ±35%, còn `--hard` bật bảng ref cứng cho key thưa (sparse).
- Áp dụng không cần restart server: Admin → tab "Dữ liệu" → Import (`admin_data.go: AdminImportData` → `SeedFromCSV` + `LoadGuRegistry`).
- Với seed mới từ đầu: `db.go: seedGusFromCSV` upsert `INSERT ... ON CONFLICT (id, rank) DO UPDATE`.
