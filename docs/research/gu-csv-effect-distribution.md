# Thống kê phân bố giá trị và thông số thực tế của Cổ Trùng trong `Gu.csv` theo Rank 1–9

- **Ticket**: [#482](https://github.com/VipPro67/GuWorld/issues/482) (Map: [#481](https://github.com/VipPro67/GuWorld/issues/481))
- **Nguồn chính (Primary Sources)**:
  - `backend/data/Gu.csv` (4.713 dòng dữ liệu, 100% `schema_version: 2`, 0 lỗi parse JSON)
  - `backend/internal/effectcatalog/block_manifest.go` (`OperationCatalog` gồm 58 `OperationType`, `TriggerCatalog` gồm 15 `Trigger`, `TargetCatalog` gồm 7 `Target`, `CapabilityCatalog` gồm 15 capability)
  - `backend/internal/models/rank_scale.go` (`RankScaleTable` cho Rank 1–9)

---

## 1. Tổng quan & Số lượng Cổ Trùng theo Rank

Tổng cộng `backend/data/Gu.csv` có **4.713 dòng** (gồm **3.472 `id` duy nhất**, trong đó **466 `id`** xuất hiện ở nhiều Rank). Toàn bộ 4.713 dòng đều dùng `effects` JSON `schema_version: 2` (không có dòng rỗng hay lỗi parse).

| Rank | Nhóm | Số lượng | `RankScale(rank)` | TB Blocks/Gu | TB Ops/Gu | `base_mana_cost` (min / p50 / max) | `base_price` (min / p50 / max) | Có `scale_ratio` |
|---:|---|---:|---:|---:|---:|---|---|---:|
| **1** | Phàm Cổ | 505 | `1` | 1.30 | 1.74 | 10 / 10 / 10 | 250 / 250 / 250 | 22 |
| **2** | Phàm Cổ | 488 | `2` | 1.30 | 1.68 | 10 / 10 / 10 | 1,000 / 1,000 / 1,000 | 20 |
| **3** | Phàm Cổ | 518 | `5` | 1.28 | 1.69 | 6 / 10 / 10 | 4,000 / 4,000 / 4,000 | 20 |
| **4** | Phàm Cổ | 508 | `20` | 1.29 | 1.73 | 6 / 10 / 10 | 22,000 / 22,000 / 22,000 | 21 |
| **5** | Phàm Cổ | 516 | `100` | 1.28 | 1.67 | 10 / 10 / 10 | 250,000 / 250,000 / 250,000 | 24 |
| **6** | Tiên Cổ | 594 | `10^4` | 1.44 | 1.87 | 5 / 5 / 10 | 3,500 / 3,500 / 3,500 | 18 |
| **7** | Tiên Cổ | 585 | `10^6` | 1.37 | 1.83 | 0 / 5 / 10 | 35,000 / 35,000 / 35,000 | 19 |
| **8** | Tiên Cổ | 557 | `10^8` | 1.38 | 1.86 | 0 / 5 / 5 | 550,000 / 550,000 / 550,000 | 21 |
| **9** | Tiên Cổ | 441 | `10^10` | 1.32 | 1.79 | 0 / 5 / 10 | 250,000 / 14,000,000 / 14,000,000 | 20 |
| **10** | Ngoại lệ | 1 | *N/A (clamp R9)* | 1.00 | 1.00 | 10 / 10 / 10 | 14,000,000 / 14,000,000 / 14,000,000 | 0 |

> **Lưu ý dữ liệu Rank 10**: Có 1 dòng ngoài dải 1–9 là `GU_XUAN` (*Huyền Tiên Cổ*, Rank 10, `END_BATTLE:FORCE_VICTORY`), trùng `id` với `GU_XUAN` (*Huyền Cổ*, Rank 9). `RankScale(rank)` trong `backend/internal/models/rank_scale.go` clamp `rank >= 9` về `10^10`.

---

## 2. Phân bố `OperationType`, `Trigger`, `Target`, `Condition`, `Duration`, `Chance`, `Params`

### 2.1 `OperationType` (`52/58` op trong `OperationCatalog` được sử dụng)

Tổng cộng có **8.326 operation** trên toàn bộ 4.713 Cổ. **4 op cốt lõi** chiếm **89.2%** tổng số operation:
1. `STAT_MOD`: **3.018** (36.2%)
2. `DAMAGE`: **2.385** (28.6%)
3. `STATUS`: **1.095** (13.2%)
4. `SHIELD`: **927** (11.1%)
5. Nhóm phổ biến tiếp theo: `HEAL` (**388**), `GRANT_CAPABILITY` (**118**), `GROWTH_DURATION` (**59**), `NOTIFY` (**32**), `CLEANSE` (**26**), `TRANSFER_RESOURCE` (**26**), `DISPEL` (**25**), `HIT_RULE` (**24**), `MODIFY_APERTURE` (**23**), `DISPLACE` (**21**), `PERIODIC_RESOURCE` (**21**), `DAMAGE_MODIFIER` (**15**), `ADVANCE_PROGRESSION` (**14**), `DIRECT_HP` (**12**), `HP_COST` (**11**).

**6 op có trong `OperationCatalog` (`block_manifest.go`) nhưng chưa xuất hiện trong `Gu.csv`**:
`BARRIER`, `OPEN_REWARD_CONTAINER`, `RANDOMIZE_TRAIT`, `REPAIR_APERTURE`, `REQUIRE`, `UNIQUE`.

Phân bố nhóm op đặc biệt theo Rank:
- **Chỉ Phàm Cổ (R1–5)**: `ADVANCE_MAJOR_REALM` (R1–4: 4), `GRANT_RESOURCE` (R1–5: 7), `MODIFY_APTITUDE` (R4: 1), `MODIFY_MAX_RESOURCE` (R1, R3: 3), `PETRIFY_APERTURE` (R3: 1), `PLACE_WORLD_OBJECT` (R5: 1), `RESTORE_RESOURCE` (R3: 2), `STATUS_IMMUNITY` (R4: 2), `TENACITY_MODIFIER` (R4: 1), `BONUS_STRIKE` (R1: 1).
- **Chỉ Tiên Cổ (R6–9)**: `REVIVE` (R6:1, R7:3, R8:2, R9:1), `DEFENSE_PENETRATION` (R7:2, R8:2), `COUNTER_ADD` (R6–9: 4), `MITIGATE_DAMAGE` (R7:1, R8:1), `TARGET_DAMAGE_MODIFIER` (R8:1, R9:1), `THRESHOLD_RECOVERY` (R8:1, R9:1), `SLOT_INTERACTION` (R8:1, R9:1), `GU_SEAL` (R7:1), `TARGET_REDIRECTION` (R7:1), `SET_RULE` (R7:1), `DEDUCTION_FINISH_CHANCE` (R9:2), `EFFECT_IMMUNITY` (R9:1), `END_BATTLE` (R9:1, R10:1), `GRANT_BODY_MOD` (R9:1), `RANK_SCALED_DEBUFF` (R9:1), `REVOKE_STATUS` (R9:1), `TURN_ORDER_RULE` (R9:2), `APPLY_STACKING_STATUS` (R9:1).

### 2.2 `Trigger` (14/15 trigger trong `TriggerCatalog` được sử dụng; tổng 6.278 block)

| Trigger | Tổng | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 | R10 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `ON_ACTION` | 3.870 | 406 | 396 | 422 | 410 | 413 | 519 | 474 | 458 | 372 | 0 |
| `ALWAYS_ON` | 2.153 | 227 | 215 | 211 | 213 | 212 | 314 | 299 | 287 | 175 | 0 |
| `CAPABILITY` | 107 | 10 | 12 | 9 | 10 | 18 | 14 | 11 | 12 | 11 | 0 |
| `ON_CONSUME` | 67 | 9 | 7 | 13 | 11 | 10 | 4 | 4 | 4 | 5 | 0 |
| `ON_TICK` | 21 | 0 | 1 | 3 | 3 | 3 | 2 | 2 | 2 | 5 | 0 |
| `COMBAT_PASSIVE` | 15 | 1 | 1 | 3 | 5 | 0 | 1 | 1 | 1 | 2 | 0 |
| `ON_BATTLE_START` | 15 | 0 | 0 | 0 | 0 | 5 | 1 | 0 | 1 | 7 | 1 |
| `BEFORE_HIT` | 11 | 1 | 1 | 1 | 1 | 0 | 1 | 2 | 3 | 1 | 0 |
| `ON_DEATH` | 8 | 0 | 0 | 0 | 0 | 0 | 1 | 3 | 2 | 2 | 0 |
| `ON_USE` | 5 | 2 | 1 | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `ON_CAST` | 4 | 0 | 0 | 0 | 0 | 0 | 1 | 2 | 0 | 1 | 0 |
| `BEFORE_ENEMY_ATTACK` | 3 | 0 | 0 | 0 | 0 | 0 | 0 | 2 | 0 | 1 | 0 |
| `ON_HIT` / `ON_TURN_START` | 1 / 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 / 1 | 0 |

(`PASSIVE` không xuất hiện trong `Gu.csv`.)

### 2.3 `Target` (7.534 kế thừa từ block, 792 ghi đè ở cấp operation)

| Target | Tổng | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 | R10 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `SELF` | 4.861 | 529 | 484 | 511 | 523 | 518 | 665 | 612 | 588 | 431 | 0 |
| `TARGET` | 3.059 | 325 | 310 | 339 | 321 | 316 | 398 | 397 | 364 | 289 | 0 |
| `ALL_ENEMIES` | 318 | 20 | 16 | 15 | 26 | 21 | 35 | 51 | 71 | 62 | 1 |
| `COL` | 67 | 6 | 6 | 11 | 8 | 8 | 7 | 8 | 8 | 5 | 0 |
| `ALL_ALLIES` | 21 | 1 | 2 | 1 | 2 | 0 | 3 | 3 | 5 | 4 | 0 |

(`ROW` và `ALL` không xuất hiện trong `Gu.csv`.) Tỉ lệ `ALL_ENEMIES` tăng mạnh từ **~2.2%** ở Phàm Cổ (98/4.365 op) lên **~5.5%** ở Tiên Cổ (219/3.960 op).

### 2.4 `Condition` & `ContextRules`
- **Cực kỳ hiếm**: chỉ **13/6.278 block** (0.21%) dùng `conditions` (R1:1, R2:1, R3:2, R4:3, R5:1, R6:1, R7:1, R8:1, R9:2).
- **0** operation dùng `condition` cấp op và **0** block dùng `context_rules`.

### 2.5 `Duration` (`turns`) & `Chance`
- **`turns`** (chủ yếu trên `STATUS`): trung vị **p50 = 2 lượt** đồng nhất ở cả 9 Rank; `min = 1`, `p90 = 3–5`. Ngoại lệ `max = 99` ở R8–9 (hiệu ứng kéo dài cả trận).
- **`chance`**: `p50 = 1.0` và `p90 = 1.0` ở tất cả Rank 1–9. Tuy nhiên có **52 operation** đặt `chance: 0` (ví dụ `GU_ALERT_BELL` với `SILENCE`/`SOUL_WOUND`, `GU_BEAM_SWORD_*` với `BLIND`, `GU_ALL_SPIRIT_IMMORTAL` với `BIND`) — cần lưu ý đây là các nhánh status bị tắt sẵn trong data.

### 2.6 `Params`
Chỉ xuất hiện trên **12 loại op**, tập trung nhiều nhất ở `GRANT_CAPABILITY` (115 lần, keys `value`, `range`) và các op cơ chế đặc biệt ở Tiên Cổ:
- `SUMMON` (5 lần: R5×4, R6×1)
- `REVIVE` (7 lần: R6×1, R7×3, R8×2, R9×1)
- `HIT_RULE` (9 lần có `params`: `LETHAL_LUCK_WARD` R1–4, `MULTI_HIT` R6–8, `COUNTER_THRESHOLD` R9)
- `OUTCOME_ROULETTE` (5 lần: `LUCK` R1–4, `CHAOS` R9)
- `SLOT_INTERACTION` (2 lần: R8, R9), `DEDUCTION_FINISH_CHANCE` (2 lần: R9), `GU_SEAL` (R7), `TARGET_REDIRECTION` (R7), `RANK_SCALED_DEBUFF` (R9), `TURN_ORDER_RULE` (R9), `RESOURCE_CONVERSION` (R5).

---

## 3. Phân bố `Value` (Min / Median / P90 / Max) theo Rank 1–9

### 3.1 Nhóm Flat theo công thức `Ratio × RankScale(rank)`

Phần lớn dữ liệu flat trong `Gu.csv` bám sát `RankScaleTable = [1, 2, 5, 20, 100, 10^4, 10^6, 10^8, 10^{10}]`:

| Rank | `DAMAGE [FLAT]` (min / p50 / p90 / max) | `SHIELD [FLAT]` (min / p50 / p90 / max) | `HEAL [FLAT]` (min / p50 / p90 / max) | `STAT_MOD [ADD_FLAT]` `ATTACK` (p50 / max) | `STAT_MOD [ADD_FLAT]` `DEFENSE` (p50 / max) | `STAT_MOD [ADD_FLAT]` `HP` (p50 / max) | `STAT_MOD [GROWTH_CAP]` (p50 / max) |
|---:|---|---|---|---|---|---|---|
| **1** | 18 / **30** / 30 / 30 | 30 / **30** / 30 / 30 | 20 / **30** / 30 / 30 | **1** / 10 | **5** / 10 | **10** / 10 | **3** / 10 |
| **2** | 36 / **60** / 60 / 60 | 60 / **60** / 60 / 60 | 35 / **40** / 60 / 60 | **2** / 20 | **10** / 20 | **20** / 20 | **6** / 20 |
| **3** | 40 / **150** / 150 / 150 | 120 / **150** / 150 / 3,000 | 100 / **100** / 150 / 150 | **5** / 5 | **25** / 25 | **50** / 50 | **15** / 50 |
| **4** | 360 / **600** / 600 / 600 | 600 / **600** / 600 / 800 | 400 / **400** / 600 / 1,500 | **20** / 20 | **100** / 100 | **200** / 200 | **60** / 200 |
| **5** | 1,800 / **3,000** / 3,000 / 3,000 | 3,000 / **3,000** / 3,000 / 60,000 | 2,000 / **2,000** / 3,000 / 7,500 | **100** / 10,000 | **500** / 50,000 | **1,000** / 100,000 | **300** / 1,000 |
| **6** | 3e4 / **3e5** / 3e5 / 3e5 | 3e4 / **3e5** / 3e5 / 4.5e5 | 3e4 / **2e5** / 3e5 / 7.5e5 | **1e4** / 1.5e4 | **5e4** / 7.5e4 | **1e5** / 1e5 | **3e4** / 1e5 |
| **7** | 3e6 / **3e7** / 3e7 / 3e7 | 2.1e7 / **3e7** / 3e7 / 4.5e7 | 2e7 / **2.35e7** / 3e7 / 7.5e7 | **1e6** / 1e8 | **5e6** / 5e8 | **1e7** / 1e9 | **3e6** / 1e7 |
| **8** | 2.4e8 / **3e9** / 3e9 / 3e9 | 3e8 / **3e9** / 3e9 / 4.5e9 | 2e9 / **2e9** / 3e9 / 7.5e9 | **1e8** / 1e10 | **5e8** / 5e10 | **1e9** / 1e11 | **3e8** / 1e9 |
| **9** | 3e7 / **3e11** / 3e11 / 3e11 | 3e11 / **3e11** / 3e11 / 3e11 | 2e11 / **3e11** / 3e11 / 7.5e11 | **1e10** / 1e12 | **5e10** / 5e12 | **1e11** / 1e13 | **1e10** / 1e11 |

- Chuẩn nền thực tế của Cổ Trùng trong `Gu.csv` là:
  - `DAMAGE [FLAT]` & `SHIELD [FLAT]`: **`30 × RankScale(rank)`**
  - `HEAL [FLAT]`: **`20–30 × RankScale(rank)`** (max lên tới `75 × RankScale(rank)`)
  - `STAT_MOD [ADD_FLAT]`: `ATTACK = 1 × RankScale`, `DEFENSE = 5 × RankScale`, `HP = 10 × RankScale`, `STORAGE = 20 × RankScale`.
- **Nhóm `STAT_MOD [ADD_FLAT]` KHÔNG scale theo `RankScale`** (giữ giá trị phẳng nhỏ xuyên suốt R1–9):
  - `SPEED` (p50 = `2`, max tăng tuyến tính `2 → 18`), `MOVEMENT` (p50 = `1`, max `1 → 8`), `INVESTIGATION` (p50 = `1–6`), `DISGUISE` (p50 = `1–3`), `REFINE_MANA_TIER` (p50 = `1–2`, max `4–5`), `PATH_CONFLICT_REDUCTION` (p50 = `12`, max `14`), `CLEANSING_POWER` (`1`).

### 3.2 Nhóm Percentage / Multiplier / Ratio (ổn định theo Rank)

Khác với nhóm Flat, các op phần trăm / hệ số gần như **không nhảy bậc theo Rank**:
- `DAMAGE [MULTIPLIER]`: `min = 1.0`, `p50 = 1.09–1.50`, `p90 = 1.38–2.0`, `max = 2.0–2.90` (R5).
- `SHIELD [PERCENT]`: `min = 20`, `p50 = 27.5–30`, `p90 = 30–38`, `max = 30–40`.
- `HEAL [PERCENT]`: `p50 = 17.5–30`, `max = 25` (R1–6) → `30` (R7–8) → `35` (R9).
- `HEAL [LIFESTEAL]`: R6 dùng `40–50` (thang %) trong khi R9 dùng `0.50` (thang thập phân) — **bất nhất đơn vị**.
- `STATUS [DEFAULT]`:
  - Nhóm % buff/debuff giữ ổn định `p50 = 10–20`, `max = 75–100` across R1–9.
  - Nhóm DoT/HoT (`BLEED`, `BURN`, `POISON`, `REGEN`) ở R6–9 nhảy sang giá trị flat theo `RankScale` (`5e4` ở R6, `5e6` ở R7, `5e8` ở R8, `1e8` ở R9).
- `TRANSFER_RESOURCE [PERCENT_OF_TARGET]`: `0.15–0.25` (R1–6, R9), tăng nhẹ lên `0.40` ở R7–8.
- `HP_COST [CURRENT_HP_PERCENT]`: `0.05` (R1–5, R8–9), lên tới `0.15–0.20` ở R6–7.

---

## 4. Nhảy bậc Phàm Cổ (R1–5) → Tiên Cổ (R6–9) & Ngoại lệ đáng chú ý

### 4.1 Nhảy bậc định lượng (Quantitative Step-Jump)
1. **Nhảy bậc chỉ số Flat (`×100` tại R5 → R6)**:
   - Nội bộ Phàm Cổ tăng khoảng `×2 → ×2.5 → ×4 → ×5` mỗi bậc (`1 → 2 → 5 → 20 → 100`).
   - Tại ranh giới **R5 → R6**, giá trị nhảy **`×100`** (`100 → 10^4`) và tiếp tục nhân **`×100`** mỗi bậc Tiên Cổ (`10^4 → 10^6 → 10^8 → 10^{10}`).
   - Vì vậy, `DAMAGE [FLAT]` nhảy từ `3,000` (R5) lên `300,000` (R6) và đạt `3e11` (R9).
2. **Mở rộng quy mô mục tiêu & độ phức tạp**:
   - `ALL_ENEMIES` tăng từ ~2.2% (R1–5) lên ~5.5% (R6–9).
   - Số block/op trung bình mỗi Cổ tăng nhẹ ở Tiên Cổ (`1.29 / 1.70` ở R1–5 lên `1.38 / 1.84` ở R6–9).
3. **Đổi hệ tiền tệ ở `base_price`**:
   - R1–5 tăng từ `250 → 250,000` (Nguyên Thạch), nhưng R6 tụt về `3,500` rồi tăng lên `35,000` (R7), `550,000` (R8), `14,000,000` (R9) (Tiên Nguyên Thạch).

### 4.2 Nhảy bậc cơ chế (Qualitative Step-Jump) & Ngoại lệ (Outliers)
1. **Cơ chế độc quyền Tiên Cổ (Min Rank Gate tự nhiên trong dữ liệu)**:
   - **R6+**: `REVIVE` (`GU_SPRING_AUTUMN_CICADA` R6–9, `GU_FORGET_DEFEAT` R7, `GU_NIRVANA_FIRE_I` R7–8), `COUNTER_ADD`.
   - **R7+**: `DEFENSE_PENETRATION` (`0.5–1.0`), `MITIGATE_DAMAGE` (`0.4`), `GU_SEAL` (`GU_FORGET_INTENT`), `TARGET_REDIRECTION` (`GU_FORGET_WILL`), `SET_RULE` (`GU_FORGET_DREAM`).
   - **R8+**: `SLOT_INTERACTION` (`GU_SHADOW_STEAL_8` `TEMPORARY_STEAL`, `GU_FORMLESS_THIEF_9` `PERMANENT_STEAL`), `TARGET_DAMAGE_MODIFIER`, `THRESHOLD_RECOVERY`.
   - **R9+**: `END_BATTLE:FORCE_VICTORY` (`GU_XUAN`), `EFFECT_IMMUNITY:HOSTILE_EFFECTS` (`GU_REVERSE`), `GRANT_BODY_MOD:SOVEREIGN_IMMORTAL` (`GU_SOVEREIGN_FETUS`), `TURN_ORDER_RULE` (`GU_BAN`, `GU_LET_YOU_GO_FIRST`), `RANK_SCALED_DEBUFF` (`FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY`), `DEDUCTION_FINISH_CHANCE`.
2. **Các Cổ vượt chuẩn / lệch chuẩn số liệu trong `Gu.csv`**:
   - **Vượt chuẩn Flat (`×20`–`×100` so với p50 cùng Rank)**:
     - `GU_HEART_ARMOR` (R3): `SHIELD [FLAT] = 3,000` (gấp **20×** p50=150).
     - `GU_BLOOD_STONE` (R5): `SHIELD [FLAT] = 60,000` (gấp **20×** p50=3,000).
     - `GU_STRENGTH_5` (R5): `DEFENSE = 50,000`, `HP = 100,000` (gấp **100×** p50 cùng Rank).
     - `GU_STRENGTH_I` (R9): `ATTACK = 1e12`, `DEFENSE = 5e12`, `HP = 1e13` (gấp **100×** p50 cùng Rank).
     - `GU_NO_COLOR_THUNDER` (R1): `STORAGE = 100` (gấp **5×** p50=20).
   - **Thấp bất thường ở R9**: `GU_SWORD_DRAGON_I` (R9, `DAMAGE [FLAT] = 3e7`, thấp hơn p50 `3e11` tới 10.000×), `GU_THEFT_LIFE_9` (R9, `DAMAGE [FLAT] = 3e9`, thấp hơn p50 100×).
   - **Ngoại lệ Rank 10**: `GU_XUAN` xuất hiện ở cả Rank 9 và Rank 10.
   - **Bất nhất đơn vị**: `HEAL [LIFESTEAL]` dùng `40–50` ở R6 nhưng `0.50` ở R9; `DIRECT_HP [SHIELD_STRIP]` dùng `0.20–0.40` xen kẽ `20–100`.
