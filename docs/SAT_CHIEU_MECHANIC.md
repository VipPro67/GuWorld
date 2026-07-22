# Cơ Chế Sát Chiêu (Killer Move System) — V2

> Tài liệu này mô tả thiết kế **mục tiêu** dựa trên Gu.csv Structure V2.
> Mọi thay đổi code sau này phải align với spec này.

---

## 1. Tổng Quan

Sát Chiêu là kỹ năng do người chơi lắp ráp từ Cổ Trùng (Gu). Mỗi sát chiêu gồm:

- **1 Core Gu (Hạch Tâm)** — quyết định Path, Damage/Defense Base, Mana Cost, Dao Mark Type.
- **0–8 Support Gu (Phụ Trợ)** — bổ sung hệ số, hiệu ứng, CC, AOE, success rate.

Tất cả dữ liệu Cổ Trùng được định nghĩa duy nhất trong **Gu.csv**.

---

## 2. Vai Trò Cổ Trùng

### Core Gu
- Là Cổ đầu tiên được chọn khi tạo sát chiêu.
- Quyết định:
  - **Path** của sát chiêu.
  - **DamageBase** — gốc sát thương.
  - **DefenseBase** — gốc phòng thủ.
  - **DaoMarkType** — loại Đạo Ấn được tích luỹ.
  - **ManaCostBase** — mana gốc.
  - **ChannelTimeBase** — thời gian tụ lực gốc.

### Support Gu
- **Không tạo** DamageBase hay DefenseBase.
- Chỉ đóng góp:
  - `base_damage_bonus` — cộng thẳng vào damage.
  - `base_defense_bonus` — cộng thẳng vào defense.
  - `damage_multiplier` / `defense_multiplier` — nhân tỷ lệ.
  - `mana_cost_reduction` / `mana_cost_multiplier` — giảm mana.
  - `success_rate_bonus` — tăng tỷ lệ thành công.
  - `aoe_*` — mở rộng phạm vi.
  - `stun` / `freeze` / `bind` / `silence` / ... — crowd control.
  - `range` / `pierce` / `curve_attack` — đặc tính đòn đánh.

---

## 3. Công Thức Tính Sát Chiêu

### 3.1 Damage / Defense

```
DamageBase     = CoreGu.BaseDamage
DefenseBase    = CoreGu.BaseDefense

DamageMultiplier  = 1 + Σ(damage_multiplier từ Support Gu được áp dụng)
DefenseMultiplier = 1 + Σ(defense_multiplier từ Support Gu được áp dụng)

DaoBonus = 1 + Player.DaoMarks[CorePath] / 100

FinalDamage  = (DamageBase  + BaseDamageBonus)  × DamageMultiplier  × DaoBonus
FinalDefense = (DefenseBase + BaseDefenseBonus) × DefenseMultiplier × DaoBonus
```

**BaseDamageBonus / BaseDefenseBonus** là tổng `base_damage_bonus` / `base_defense_bonus` từ Support Gu.

### 3.2 Mana Cost

```
ManaCost = CoreGu.ManaCostBase × (1 + Σ(mana_cost_multiplier)) − Σ(mana_cost_reduction)
```

### 3.3 Success Rate

```
SuccessRate = 95
              + Attainment × 2
              + Σ(success_rate_bonus)
              − SupportCount × 5
              − PathConflictPenalty

Clamp(SuccessRate, 5%, 100%)
```

**Attainment** = cấp độ cảnh giới lưu phái (`player.PathMasteries[path]`):
- 0: chưa đạt cảnh giới
- 1: Đại Sư (Master)
- 2: Tông Sư (Grandmaster)
- 3: Đại Tông Sư (Great Grandmaster)
- 4: Vô Thượng Đại Tông Sư (Supreme Grandmaster)

### 3.4 Channel Time

```
ChannelTime = CoreGu.ChannelTimeBase × (1 − Σ(channeling_reduction))
```

Nếu `ChannelTime > 0`, người dùng phải chờ N lượt trước khi sát chiêu phát động.

### 3.5 Instant Cast

```
IsInstantCast = (ChannelTime <= 0)
```

(Không còn điều kiện support ≤ 2 như V1)

---

## 4. Xung Đột Lưu Phái (Path Conflict)

Mỗi Support Gu có Path khác với Core Gu:

| Điều kiện | Penalty |
|-----------|---------|
| Support Path ≠ Core Path | +10% |
| Support Path đối nghịch với Core Path | +20% (cộng dồn) |

**Path đối nghịch** (ví dụ):
- Fire ↔ Water
- Strength ↔ Wisdom
- Metal ↔ Wood
- (có thể mở rộng)

```
PathConflictPenalty = Σ penalty từ mỗi support khác path
```

`path_conflict_reduction` làm giảm penalty này (trừ trực tiếp vào tổng).

---

## 5. Phản Phệ (Backlash)

Khi `SuccessRate` thất bại (random roll > SuccessRate):

```
BacklashDamage = ManaCost × 0.5 × SupportCount
```

Hậu quả khi phản phệ:
- Mất HP = `BacklashDamage`
- Tổn thương **Không Khiếu** (Aperture) — giảm dung lượng / hiệu quả.
- Hủy **Channeling** nếu đang tụ lực.

`backlash_reduction` giảm sát thương phản phệ (trừ thẳng vào `BacklashDamage`).

---

## 6. Channeling (Tụ Lực)

Một số sát chiêu yêu cầu `ChannelTime > 0`.

Trong lúc Channeling, nếu người dùng bị:

| Hiệu ứng | Hậu quả |
|----------|---------|
| Stun | Hủy sát chiêu |
| Freeze | Hủy sát chiêu |
| Silence | Hủy sát chiêu |
| Counter (phá chiêu) | Hủy sát chiêu |

`counter_resistance` giúp chống lại bị phá chiêu (tỉ lệ kháng).

Khi Channeling bị hủy:
- Mana **không** được hoàn lại.
- Sát chiêu cooldown **vẫn** được kích hoạt (hoặc chỉ một phần — cần balance).

---

## 7. Effect Categories

### 7.1 Combat Effects

Chỉ dùng trong Combat Engine.

| Key | Kiểu | Mô tả |
|-----|------|-------|
| `base_damage_bonus` | additive | Cộng vào DamageBase |
| `damage_multiplier` | multiplicative | Nhân vào FinalDamage |
| `base_defense_bonus` | additive | Cộng vào DefenseBase |
| `defense_multiplier` | multiplicative | Nhân vào FinalDefense |
| `armor_penetration` | % | Xuyên giáp % |
| `ignore_same_rank_defense_chance` | % | Cơ hội bỏ qua phòng thủ |
| `lifesteal` | % | Hút máu theo % sát thương |
| `heal` | flat | Hồi máu trực tiếp |
| `shield` | flat | Tạo khiên |
| `stun` | bool/turns | Choáng |
| `freeze` | bool/turns | Đóng băng |
| `bind` | bool/turns | Trói |
| `paralyze` | bool/turns | Tê liệt |
| `slow` | % | Làm chậm |
| `silence` | bool/turns | Câm lặng |
| `bleed` | DoT | Chảy máu theo lượt |
| `burn` | DoT | Thiêu đốt theo lượt |
| `poison` | DoT | Trúng độc theo lượt |
| `evasion` | % | Né tránh |
| `damage_reflect` | % | Phản sát thương |
| `range` | number | Tầm đánh |
| `pierce` | bool | Xuyên hàng |
| `curve_attack` | bool | Đòn cong (né đỡ đầu) |
| `aoe_row` | bool | Phạm vi theo hàng |
| `aoe_col` | bool | Phạm vi theo cột |
| `aoe_radius` | number | Phạm vi hình tròn |
| `aoe_all` | bool | Toàn bộ địch |
| `teleportation` | bool | Dịch chuyển |
| `escape` | bool | Thoát chiến |
| `summon_phantom` | bool | Triệu hồi hư ảnh |
| `summon_clone` | bool | Triệu hồi phân thân |
| `time_acceleration` | % | Tăng tốc bản thân |
| `speed_multiplier` | % | Nhân tốc độ |

### 7.2 Killer Move Effects

Chỉ dùng khi assemble sát chiêu (tính SuccessRate, Channel, Backlash).

| Key | Kiểu | Mô tả |
|-----|------|-------|
| `success_rate_bonus` | additive | +% thành công |
| `channeling_reduction` | % giảm | Rút ngắn thời gian tụ lực |
| `backlash_reduction` | % giảm | Giảm sát thương phản phệ |
| `path_conflict_reduction` | additive | Giảm penalty xung đột path |
| `counter_resistance` | % | Chống bị phá chiêu |
| `mana_cost_reduction` | flat | Giảm mana |
| `mana_cost_multiplier` | % | Nhân mana (có thể >1) |
| `dao_mark_multiplier` | % | Nhân Đạo Ấn nhận được |

### 7.3 Non-Combat Effects

Bị bỏ qua trong Combat Engine. Dùng bởi Exploration Engine, Refinement Engine.

| Key | Mô tả |
|-----|-------|
| `storage` | Tăng dung lượng kho |
| `investigation` | Khả năng điều tra / phát hiện |
| `tracking` | Truy tìm |
| `portal_creation` | Tạo cổng dịch chuyển |
| `generate_primeval_stones` | Sinh nguyên thạch |
| `generate_materials` | Sinh vật liệu |
| `generate_gu` | Sinh Cổ mới |
| `refinement_bonus` | Tỷ lệ tinh luyện thành công |
| `refinement_power` | Hiệu quả tinh luyện |
| `memory_storage` | Lưu trữ ký ức (Mộng Cảnh) |
| `contract_oath` | Hiệu ứng khế ước / minh thệ |
| `increase_aptitude` | Tăng tư chất |
| `force_rank_up` | Ép thăng cấp (có giới hạn) |

---

## 8. Dao Mark (Đạo Ấn)

Khi thi triển sát chiêu **thành công**, người chơi tích luỹ Đạo Ấn cho Path tương ứng.

```
DaoMarks[CorePath] += BaseDaoMarkValue × dao_mark_multiplier
```

DaoBonus trong công thức sát thương:

```
DaoBonus = 1 + Player.DaoMarks[CorePath] / 100
```

Không giới hạn trên của DaoMarks (nhưng diminishing returns có thể được áp dụng sau này).


## 9. Hướng Dẫn Implement

### 9.1 Thứ tự ưu tiên

1. Cập nhật Gu.csv thêm các cột: `base_damage`, `base_defense`, `mana_cost_base`, `channel_time_base`, `dao_mark_type`, `path_opposition`.
2. Sửa `calculateKillerMoveStatsForPlayer` — áp dụng công thức V2.
3. Đổi `Stability` → `SuccessRate` trong DB, model, response.
4. Thêm `DaoMarks` vào Player model + DB migration.
5. Implement Channeling cơ chế trong combat loop.
6. (Đã xoá) Wisdom stat — dùng cổ trùng để tăng tỉ lệ thành công.
7. Cập nhật `applyKillerMoveToCombat` — đọc effect theo category.

### 9.2 Những thứ giữ lại từ V1

- Cấu trúc DB `killer_moves` / `killer_move_slots` (thêm/xoá column).
- Luồng Calculate → Preview → Create → Cast.
- Cooldown giảm mỗi turn.
- Mastery (usage_count ≥ 50 → +15 SuccessRate thay vì Stability).

### 9.3 Những thứ cần xoá

- `killerMoveMultiTaskPenalty` (hằng số + logic).
- `ApplyMultiTaskPenalty` field trong request.
- `CastTime` (thay bằng `ChannelTime`).
- `compatibilityPenalty` accumulation cũ (thay bằng PathConflictPenalty mới).
