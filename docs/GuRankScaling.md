# GuRankScaling – Bảng chuẩn sức mạnh Cổ trùng theo Rank

Tài liệu này định nghĩa chuẩn scale sức mạnh (công / thủ / khiên / hồi máu / chỉ số passive) cho **effects** của từng Cổ trùng theo rank, dùng làm tham chiếu khi soạn sửa `effects` JSON trong `backend/cmd/server/data/Gu.csv`.

> Quy tắc vàng: **sức mạnh Cổ trùng nằm 100% trong `effects` JSON**, không có hệ số nhân theo rank ở tầng combat (`combat_effects.go`, `player.go`). Chỉ nên thêm helper `GuPowerScale(rank)` làm công cụ **cảnh báo khi seed/import** (lệch quá ±35% so với chuẩn), **không** nhân vào công thức chiến đấu (tránh double-scale).

## 1. Reference curves

Bảng gốc do người chơi/thiết kế cung cấp (tương ứng 9 rank):

| Đại lượng | Công thức | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Sức mạnh quái `P` | `P(rank)` | 1 | 2 | 5 | 20 | 100 | 1.000 | 10.000 | 100.000 | 10.000.000 |
| Chi phí năng lượng `E` (Phàm) | `E = P` | 1 | 2 | 5 | 20 | 100 | — | — | — | — |
| Chi phí năng lượng `E` (Tiên) | `100^(rank-5)` | — | — | — | — | — | 10 | 10 | 10 | 10|
| Giá base | theo bảng | 200 | 500 | 1.200 | 2.500 | 5.000 | 100000 | 250000 | 600000 | 1500000 |

Ngưỡng Tiên cổ: `immortalRankThreshold = 6` (rank ≥ 6 phải là Tiên Cổ, `is_immortal = true`).

**Giá Item theo rank** — bắt đầu rank 1 = 1, scale tỉ lệ như Cổ (≈ Gu/200): `1, 3, 6, 13, 25, 500, 1.250, 3.000, 7.500` cho rank 1–9 (item rank 0 quy về rank 1 = 1). Fallback trong `pkg/pricing/pricing.go` và mảng cứng frontend (`CaravanShop.tsx`, `SectView.tsx`) phải khớp 2 bảng trên.

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
| `passive.stat_base_speed` | `1 × P` | cộng tốc độ nền |
| `active.status_tick_damage` | `10 × P` | sát thương tick theo lượt (BURN/BLEED/PETRIFY...) |

## 3. Bảng chuẩn rank 1–9

| Rank | `P` | dmg | shield | heal | sATK | sDEF | sHP | sSPD | `E` | Giá |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 1 | 25 | 20 | 15 | 1 | 5 | 10 | 1 | 1 | 200 |
| 2 | 2 | 50 | 40 | 30 | 2 | 10 | 20 | 2 | 2 | 500 |
| 3 | 5 | 125 | 100 | 75 | 5 | 25 | 50 | 5 | 5 | 1.200 |
| 4 | 20 | 500 | 400 | 300 | 20 | 100 | 200 | 20 | 20 | 2.500 |
| 5 | 100 | 2.500 | 2.000 | 1.500 | 100 | 500 | 1.000 | 100 | 100 | 5.000 |
| 6 | 1.000 | 25.000 | 20.000 | 15.000 | 1.000 | 5.000 | 10.000 | 1.000 | 100 | 100.000 |
| 7 | 10.000 | 250.000 | 200.000 | 150.000 | 10.000 | 50.000 | 100.000 | 10.000 | 10.000 | 250.000 |
| 8 | 100.000 | 2.500.000 | 2.000.000 | 1.500.000 | 100.000 | 500.000 | 1.000.000 | 100.000 | 1.000.000 | 600.000 |
| 9 | 10.000.000 | 250.000.000 | 200.000.000 | 150.000.000 | 10.000.000 | 50.000.000 | 100.000.000 | 10.000.000 | 100.000.000 | 1.500.000 |

> Ví dụ đối chiếu thực tế: `GU_DIAMOND_THOUGHT` (R6) `damage_flat=25.000`, `GU_FLYING_SWORD` (R7) `damage_flat=250.000`, `GU_BLOOD_DEITY` (R9) `damage_flat=250.000.000` — khớp chuẩn `25×P`. `GU_MYRIAD_SELF` (R7) `damage_flat` dùng `passive.stat_base_attack=10.000` vì phân thân nhân theo `BaseAttack` (`summon_phantom` → `PhantomDamageScale`).

## 4. Cách áp dụng khi sửa Gu.csv

1. Xác định rank, role chính (`primary_role`) và mô tả của cổ trùng.
2. Nếu mô tả **rõ tấn công** → điền `active.damage_flat` theo bảng (+ tùy chọn `damage_path`, `apply_status`, `target_selector`).
3. Nếu mô tả **rõ phòng thủ / thủ hộ / đỡ đòn** → điền `active.shield_flat` (hoặc `passive.stat_base_defense`).
4. Nếu mô tả **hồi máu / chữa trị** → điền `active.heal_flat`.
5. Passive chỉ số (`stat_base_*`) dùng khi cổ trùng "ban sức mạnh nền" / nuôi cổ / hóa thân (vd Biến Hóa Cổ, phân thân).
6. Giữ nguyên `role`, `combat_roles`, `combat_tags`, `support_tags`, `description`, `base_mana_cost` trừ khi có yêu cầu riêng.
7. Rank ≥ 6 phải gắn `is_immortal = true`.

## 5. Kiểm chứng

- Sau khi sửa CSV: chạy audit script so mỗi rank (min/max dmg, shield, heal, passive) với bảng chuẩn; sai lệch > ±35% cần soi lại.
- Áp dụng không cần restart server: Admin → tab "Dữ liệu" → Import (`admin.go: AdminImportData` → `SeedFromCSV` + `LoadGuRegistry`).
- Với seed mới từ đầu: `db.go: seedGusFromCSV` upsert `INSERT ... ON CONFLICT (id, rank) DO UPDATE`.
