# Nghiên Cứu & Phân Loại Danh Mục Hiệu Ứng Trùng Lặp & Bản Đồ Nguyên Tử Hóa
*(Effect Deduplication, Fragmentation Audit & Atomic Operations Mapping Matrix)*

- **Ticket**: [#68](https://github.com/VipPro67/GuWorld/issues/68) - Rà soát và phân loại danh mục hiệu ứng trùng lặp & nguyên tử hóa
- **Mục tiêu**: Chuẩn hóa toàn bộ hệ thống hiệu ứng phân mảnh, loại bỏ trùng lặp 3 tầng (CSV shorthand, struct fields, simulator tiers) và xây dựng bản đồ ánh xạ sang mô hình nguyên tử hóa **Composable Effect Block** (ECA - Event Condition Action).
- **Trạng thái**: Hoàn tất nghiên cứu & thiết kế ma trận ánh xạ.

---

## 1. Khảo Sát & Phân Tích Các Nguồn Dữ Liệu Gốc (Primary Sources Analysis)

### 1.1 `backend/data/Gu.csv`
- Tập tin chứa hơn 4,200 bản ghi Cổ trùng với cột JSON `effects`.
- Thực trạng: Dữ liệu hiện đại trong CSV đã sử dụng cấu trúc `apply_status` (ví dụ `[{"id": "FREEZE", "turns": 1}]`), `damage_flat`, `shield_flat`, `stat_base_attack`, v.v.
- Tuy nhiên, hệ thống vẫn duy trì các key rác, key metadata (`description`), và các trường legacy chưa từng được runtime thực thi (ví dụ `replicate`, `generate_gu`, `wood_transformation`, `convert_damage_to_heal_percent`, `absorb_damage_flat`).

### 1.2 `backend/internal/models/gu.go`
- **`ActiveEffect`** (L185–247): Định nghĩa 50+ trường gồm chỉ số phẳng, tỷ lệ phần trăm, và danh sách 13 shorthand status (`freeze`, `slow`, `slow1`, `weak`, `stun`, `silence`, `bleed`, `poison`, `thorns`, `regen`, `atk_up`, `def_up`, `taunt`).
- **`normalize()`** (L428–490): Tự động chuyển đổi các shorthand status sang `StatusApplication` nếu `apply_status` rỗng.
- **`PassiveEffect`** (L249–281): Lưu trữ các chỉ số tăng cường nhân vật vĩnh viễn (thuộc tính cơ bản, đạo ngân, kháng tính, giảm sát thương theo phái).
- **`SpecialMechanic`** (L283–287): Lưu danh sách cơ chế phi tuyến tính dạng `{event_trigger, action_code, params}`.

### 1.3 `backend/internal/statusfx/` (`status.go`, `spec.go`)
- Là **Single Source of Truth** cho trạng thái chiến đấu:
  - 31 trạng thái định danh chuẩn (`StatusBleed`, `StatusPoison`, `StatusStun`, `StatusGuSeal`, v.v.).
  - Phân loại `StatusCategory`: `CategoryPhysicalDoT`, `CategoryMentalCC`, `CategoryAperture`, `CategoryDaoConflict`, `CategoryBuff`, `CategoryImmortal`.
  - Quy tắc thực thi dựa trên `Behaviour` hook: `dot`, `regen`, `stun`, `freeze`, `silence`, `force_basic`, `confuse`, `stealth`, `sleep`, `blind`, `charm`, `bind`, `taunt`, `thorns`, `atk_up`, `def_up`, `weak`, `slow`, `haste`, `armor_break`, `damage_reduction`, `invincible`, `revive_lock`, `essence_leak`, `gu_seal`, `soul_wound`, `thought_chaos`, `dao_conflict`, `backlash`.
  - Cơ chế tính toán xác suất áp chế: `CanApplySelfStatus` (L337) và `CanApplyStatusWithGuRank` (L367) tính theo chênh lệch Chuyển (Rank), Đạo Ngân, và Kháng tính.

### 1.4 `backend/internal/effectcatalog/` (`manifest.go`, `catalog.go`, `validate.go`, `timing.go`)
- **Manifest**: Phân định `Disposition` (`supported`, `partial`, `metadata`, `migrate`, `known_bug`) và `Owner` (`combat`, `player-stats`, `world`, v.v.).
- **Timing** (`timing.go` L23–30): Phân loại thời điểm kích hoạt: `ALWAYS_ON`, `ON_ACTION`, `ON_EVENT`, `SCHEDULED`, `CAPABILITY`, `META`.

### 1.5 `backend/internal/handlers/gu_effects.go` & `backend/internal/simulator/engine.go`
- **`gu_effects.go`** (L161–340): Trình biên dịch action cho turn-based legacy combat, ánh xạ qua `legacyCombatStatusType` và sinh log dạng text.
- **`combat_auto_resolve.go`** (L417–518): Biên dịch `GuDictionaryItem` thành `simulator.SimAction`.
- **`simulator/models.go`** & **`engine.go`**: Thực thi tự động. Tồn tại song song cả `SimAction.StatusOps`, các trường Tier (`StunTier`, `SlowTier`...), và các trường độc lập (`DisableGuTurns`, `DamageHPPercent`, `ShieldStripPercent`, `SelfEvasionPct`, `TargetWeakPct`).

---

## 2. Danh Mục Chi Tiết Các Trường Hiệu Ứng Trùng Lặp & Phân Mảnh

### 2.1 Trùng Lặp 3 Tầng: Shorthand Status vs `apply_status` vs `SimAction.*Tier`
- **Hiện tượng**: Một hiệu ứng như Choáng (`STUN`) hay Đóng Băng (`FREEZE`) có thể được biểu diễn ở:
  1. `ActiveEffect.Freeze: 1` (Shorthand JSON key).
  2. `ActiveEffect.ApplyStatus: [{"id": "FREEZE", "turns": 1}]` (Registry format).
  3. `SimAction.FreezeTier: 1` (Simulator legacy field).
  4. `SimAction.StatusOps: [{ID: "FREEZE", Turns: 1}]` (Registry op).
- **Hệ quả**: Trong `simulator/status_apply.go` (`collectAllStatusOps`, L19–91), engine phải gom cả 2 nguồn `StatusOps` và các trường `*Tier`. Điều này gây lãng phí bộ nhớ, khó bảo trì và dễ sinh lỗi lệch duration/intensity giữa 2 đường nạp.

### 2.2 Phân Mảnh Cơ Chế DoT / HoT (Bleed, Poison, Burn, Slow_Damage vs Regen)
- **DoT**:
  - `SimAction.BleedPerTurn` và `SimAction.PoisonPerTurn` được xử lý tách biệt với các DoT mới như `BURN` và `SLOW_DAMAGE`.
  - Trong `simulator/status_apply.go` (L24–29), engine phải tự tạo thêm `StatusOp` thủ công cho `BleedPerTurn` và `PoisonPerTurn`.
  - `status_tick_damage` và `status_duration_ticks` trong `ActiveEffect` đóng vai trò giá trị mặc định cho toàn bộ mảng `apply_status`, xung đột với cấu trúc tự thân `{id, value, turns, chance}` của từng phần tử.
- **HoT**:
  - `RegenTier` / `ActiveEffect.Regen` đối đầu với `apply_status: "REGEN"`. Cả hai đều map vào `DoTEffect{Type: "regen"}` nhưng cách tính hồi máu bị phân nhánh (giữa `TierToPercent` và % MaxHP mặc định 5%).

### 2.3 Trùng Lặp & Xung Đột Cơ Chế Phong Ấn Cổ (Gu Sealing)
Hệ thống có tới **6 cách thức khác nhau** để làm cùng một việc: phong ấn Cổ của đối phương trong 1 lượt:
1. `ActiveEffect.DisableGuNextTurn` (cờ boolean trong active).
2. `ActiveEffect.resolveStatusApplications()` tự động tiêm `statusfx.StatusGuSeal` nếu cờ trên bật (models/gu.go L173).
3. `combat_auto_resolve.go` (L464) gán `act.DisableGuTurns = 1`.
4. `engine.go` (L1786) kiểm tra `a.DisableGuTurns > 0` và gọi hàm riêng `sealGuTurns(target, a.DisableGuTurns)`.
5. `applyStatusOps` lại tiếp tục áp dụng `statusfx.StatusGuSeal` từ `StatusOps`.
6. Trạng thái `statusfx.StatusForgetEgo` ("Lãng Quên Bản Ngã") cũng dùng chung `BehaviourGuSeal`.

### 2.4 Trùng Lặp Cơ Chế Suy Yếu / Giảm Sát Thương (Weak vs TargetDamageReducePct)
- `ActiveEffect.Weak` (tier 1-5).
- `ActiveEffect.TargetDamageReducePct` (tỷ lệ 0..1).
- `SimAction.TargetWeakPct` (tỷ lệ 0..1).
- `statusfx.StatusWeak` (debuff `BehaviourWeak`).
- `statusfx.StatusForgetReality` (debuff `BehaviourWeak`, lấy giá trị từ `TargetDamageReducePct * 100`).
- Trong `engine.go` (L1778), engine lại tự tay tạo một `StatusOp{ID: StatusWeak, Value: a.TargetWeakPct * 100}` để nạp vào danh sách áp dụng.

### 2.5 Phân Mảnh Cơ Chế Hút Máu (Life Steal)
- `ActiveEffect.BloodDrain` (cờ boolean): Khi bật, code gán cứng `LifeStealRate += 0.5` (`gu_effects.go` L204, `combat_auto_resolve.go` L491).
- `ActiveEffect.LifeStealRate` (float64): Giá trị thực (0..1).
- Cả hai đều gộp vào `SimAction.LifeStealPct`. Việc tồn tại song song cả cờ boolean lẫn float rate là thừa thãi.

### 2.6 Phân Mảnh Xuyên Giáp, Phá Khiên & Sát Thương Trực Tiếp (Shield Piercing & Stripping)
- **Bỏ qua khiên (Shield Bypass)**:
  - `other: [{"event_trigger": "ON_HIT", "action_code": "BYPASS_SHIELD"}]`.
  - `ActiveEffect.DamageHPPercent` (gây sát thương % MaxHP thẳng vào máu, bỏ qua khiên).
  - `statusfx.StatusSoulWound` (`DirectHP: true` - sát thương DoT ăn thẳng vào HP bỏ qua khiên).
- **Hủy khiên (Shield Strip / Break)**:
  - `ActiveEffect.ShieldFlatReducePercent`: trừ % khiên ảo hiện tại của mục tiêu.
  - `statusfx.StatusArmorBreak`: giảm phòng thủ gián tiếp.
- **Bỏ qua phòng ngự (Defense Penetration)**:
  - `ActiveEffect.IgnoreDefense` (trừ flat thủ).
  - `ActiveEffect.IgnoreDefensePct` (trừ % thủ).

### 2.7 Phân Mảnh Tăng / Giảm Chỉ Số (StatMod)
- **Active Flat Bonus**: `ActiveEffect.StatBaseAttackFlat` (cộng thẳng công kích trước khi tính scale).
- **Active Multipliers**: `damage_mult`, `shield_mult`, `refine_mana_tier` (+20% dmg/tier), `wood_transformation` (+30 def), `all_stats_buff_percent`, `speed_boost_flat`. ~~`self_evasion_boost_percent`~~ → **đã gỡ**, thay bằng status `EVASION`.
- **Status Buffs**: `ATK_UP`, `DEF_UP`, `HASTE`, `DAMAGE_REDUCTION_UP`.
- **Passive Stats**: `stat_base_attack`, `stat_base_defense`, `stat_base_hp`, `stat_base_speed`, `tenacity`, `cc_resist`, `damage_multiplier_on_hit`. ~~`accuracy`~~ và ~~`evasion`~~ **đã gỡ** khỏi hệ thống.

### 2.8 Bất Cập Giữa SpecialMechanic (`other`) Và Built-in Active/Passive
Nhiều hiệu ứng bản chất giống nhau nhưng một bên được viết thành code riêng trong `ActiveEffect`, một bên lại đẩy vào mảng `other`:
- `DOUBLE_DAMAGE_DEALT` nằm trong `other` (`PASSIVE` / `COMBAT_PASSIVE`), trong khi `damage_mult` nằm trong `active`, và `damage_multiplier_on_hit` nằm trong `passive`.
- ~~`FIRST_TURN_EVADE` nằm trong `other`, trong khi `self_evasion_boost_percent` nằm trong `active`, và `evasion` nằm trong `passive`~~ → **đã giải quyết**: né tránh gộp về một nguồn duy nhất là status `EVASION` trong `statusfx` (`FIRST_TURN_EVADE` vẫn là mechanic turn-1 độc lập, bị Cấm Cổ chặn).
- `BYPASS_SHIELD` nằm trong `other`, trong khi `damage_hp_percent` nằm trong `active`.
- `SUMMON_WOLF_FRONTLINE` nằm trong `other`, trong khi `summon_phantom` và `summon_growth_phantoms` nằm trong `active`.

### 2.9 Các Trường Rác / Mồ Côi (Orphan & Unused Fields)
- `replicate`: Chỉ in log `"sinh sản nhân bản ma cổ"`, không có logic chiến đấu thực tế.
- `generate_gu`: Chỉ in log `"ngưng tụ Sinh Cơ Diệp"`.
- `wood_transformation`: Gán cứng `+30.0` phòng thủ, chỉ mang tính cục bộ của 1-2 Cổ trùng.
- `range`: Chỉ in log kéo dài khoảng cách tấn công, không tác động đến vị trí trên Grid chiến đấu.
- `cleansing_power` trong `active`: Chỉ in log `"Thanh tẩy dị lực"`, không hề xóa debuff trong `engine.go`.
- Các trường `DispositionMigrate` trong manifest: `reveal_all_stealth`, `reveal_weakness`, `invulnerable_ticks`, `summon_monster_id`, `decoy_count`, `convert_damage_to_heal_percent`, `absorb_damage_flat`.

---

## 3. Phân Loại Toàn Bộ Hiệu Ứng Thành 9 Nhóm Nguyên Tử (Atomic Operations)

Mô hình **Composable Effect Block** tái cấu trúc mọi hiệu ứng thành các hành động nguyên tử (Atomic Operations) gắn liền với mô hình **ECA (Event - Condition - Action)**:

```
┌─────────────────────────────────────────────────────────┐
│                 Composable Effect Block                 │
├─────────────────────────────────────────────────────────┤
│ Trigger   : ON_ACTION | ON_HIT | ON_TURN_START | ...     │
│ Conditions: TargetHpBelow | CasterRankGte | ...          │
│ Target    : TARGET | CASTER | ALL_ENEMIES | ALL_ALLIES  │
│ Scaling   : DaoMarkScale | RankAttenuation | ...       │
│ Operations: [ AtomicOperation_1, AtomicOperation_2 ]    │
└─────────────────────────────────────────────────────────┘
```

9 nhóm nguyên tử bao gồm:

### 1. `Damage` (Sát Thương Cơ Bản)
- **Ý nghĩa**: Tính toán và gây sát thương chịu ảnh hưởng bởi Phòng ngự, Giáp Chân Nguyên (Virtual Shield), và Đạo Ngân.
- **Thuộc tính**:
  - `type`: `"FLAT"` | `"PERCENT"` | `"MULTIPLIER"` | `"MULTI_HIT"`
  - `value`: Giá trị cơ bản
  - `path`: Lưu phái (e.g. `"Blade Path"`, `"Fire Path"`)
  - `penetration`: Flat / Percent phòng ngự bỏ qua (`ignore_defense`)

### 2. `Shield` (Hộ Thể / Khiên Ảo)
- **Ý nghĩa**: Cung cấp hoặc biến đổi lớp giáp Chân Nguyên Giáp / Khiên bảo vệ.
- **Thuộc tính**:
  - `type`: `"FLAT"` | `"PERCENT_MAX_HP"` | `"MULTIPLIER"`
  - `value`: Giá trị khiên cấp cho bản thân hoặc đồng đội
  - `path`: Lưu phái hộ thể

### 3. `Heal` (Hồi Phục Sinh Lực & Chân Nguyên)
- **Ý nghĩa**: Tái tạo lượng HP hoặc Chân Nguyên / Tinh Thần Lực tiêu hao.
- **Thuộc tính**:
  - `resource`: `"HP"` | `"MANA"` | `"SOUL"` | `"THOUGHT"`
  - `type`: `"FLAT"` | `"PERCENT_MAX"` | `"LIFESTEAL"` (hút máu theo % damage)
  - `value`: Giá trị hồi phục

### 4. `DirectHP` (Sát Thương Chuẩn & Trừ Máu / Hủy Khiên Trực Tiếp)
- **Ý nghĩa**: Tác động trực tiếp vào thanh máu sinh mệnh hoặc thanh khiên ảo, bỏ qua hoàn toàn cơ chế phòng ngự vật lý.
- **Thuộc tính**:
  - `sub_type`:
    - `"DIRECT_HP_DAMAGE"`: Trừ HP thẳng, xuyên thủng hoàn toàn virtual shield (thay thế `BYPASS_SHIELD`).
    - `"MAX_HP_PERCENT"`: Sát thương theo % Max HP (thay thế `damage_hp_percent`).
    - `"SHIELD_STRIP"`: Hủy trực tiếp % khiên của địch (thay thế `shield_flat_reduce_percent`).
    - `"SOUL_PIERCE"`: Sát thương hồn phách trực tiếp (thay thế `SOUL_WOUND` DoT DirectHP).

### 5. `StatMod` (Biến Đổi Chỉ Số Động & Tĩnh)
- **Ý nghĩa**: Điều chỉnh các thuộc tính chiến đấu hoặc thuộc tính nhân vật (Công, Thủ, Tốc, Bạo kích, Né tránh, Chính xác, Kháng hiệu ứng).
- **Thuộc tính**:
  - `stat`: `"ATTACK"` | `"DEFENSE"` | `"SPEED"` | `"CRIT_RATE"` | `"TENACITY"` | `"CC_RESIST"` | `"DAMAGE_REDUCTION"` (~~`EVASION`~~ / ~~`ACCURACY`~~ đã gỡ khỏi `StatCatalog`)
  - `mode`: `"ADD_FLAT"` | `"MULTIPLY_PERCENT"`
  - `value`: Giá trị tăng/giảm
  - `duration`: `0` (vĩnh viễn / ALWAYS_ON) hoặc `n` lượt (combat stance)

### 6. `StatusEffect` (Trạng Thái Khống Chế & Hiệu Ứng Theo Thời Gian)
- **Ý nghĩa**: Gây các trạng thái đặc biệt được quản lý bởi `statusfx.Registry`.
- **Thuộc tính**:
  - `status_id`: ID trạng thái chuẩn (`STUN`, `FREEZE`, `BLEED`, `POISON`, `BURN`, `GU_SEAL`, `TAUNT`, `THORNS`, v.v.)
  - `intensity`: Cường độ (DoT damage, % phản đòn, % làm chậm)
  - `turns`: Số lượt duy trì
  - `chance`: Tỉ lệ kích hoạt cơ bản (áp dụng công thức Rank / Dao Mark Scaling)

### 7. `Barrier` (Bảo Hộ Tuyệt Đối / Miễn Nhiễm)
- **Ý nghĩa**: Cơ chế vô hiệu hóa sát thương hoặc chặn trạng thái bất lợi hoàn toàn.
- **Thuộc tính**:
  - `barrier_type`:
    - `"INVINCIBLE"`: Bất tử / Miễn dịch mọi sát thương trong n lượt.
    - `"STATUS_IMMUNITY"`: Miễn nhiễm trạng thái (theo phái hoặc toàn bộ).
    - `"REVIVE_LOCK"`: Khóa khả năng hồi sinh của mục tiêu.

### 8. `Cleanse` (Thanh Tẩy & Giải Trừ)
- **Ý nghĩa**: Xóa bỏ các hiệu ứng bất lợi (debuff) hoặc tước đoạt buff có lợi của đối phương.
- **Thuộc tính**:
  - `target_type`: `"DISPEL_DEBUFF"` (giải debuff bản thân/đồng đội) | `"DISPEL_BUFF"` (hóa giải buff của địch)
  - `category`: `"ALL"` | `"MENTAL_CC"` | `"PHYSICAL_DOT"` | `"APERTURE"` | `"DAO_CONFLICT"`
  - `count`: Số lượng hiệu ứng tối đa được giải trừ (ưu tiên theo độ nguy hiểm)

### 9. `UniqueMechanic` (Cơ Chế Sự Kiện Đặc Thù & Năng Lực Ngoại Vi)
- **Ý nghĩa**: Các quy tắc chiến đấu đặc biệt theo kịch bản hoặc các tính năng tương tác thế giới ngoài chiến đấu.
- **Thuộc tính**:
  - `mechanic_code`:
    - Combat: `"REBORN"` (Hồi sinh khi chết), `"SUMMON_PHANTOM"` (Triệu hồi thú lực/hư ảnh), `"SEIZE_INITIATIVE"` (Tranh tiên cơ/đoạt quyền đánh trước), `"FIRST_TURN_EVADE"` (Né đòn lượt đầu), `"CHAOS_GAMBLE"` (Xúc xắc ngẫu nhiên).
    - World/Explore: `"DISGUISE"` (Ngụy trang/che giấu thiên cơ), `"MARKETPLACE_ACCESS"`, `"REMOTE_TRADE"`, `"CHAT_CONNECT"`, `"CAPTURE_BONUS"`, `"PORTAL_TELEPORT"`.
  - `params`: JSON tham số mở rộng.

---

## 4. Bảng Ma Trận Ánh Xạ (Mapping Matrix)

| Trường / Cơ Chế Cũ | Vị Trí Cũ | Atomic Operation Nhóm | Thao Tác Chuẩn Hóa Đề Xuất | Ghi Chú & Giải Pháp Chuyển Đổi |
| :--- | :--- | :--- | :--- | :--- |
| `damage_flat` | `active` | **Damage** | `type: "FLAT", value: X` | Giữ nguyên giá trị cơ bản |
| `damage_mult` | `active` | **Damage** | `type: "MULTIPLIER", value: X` | Thống nhất hệ số nhân |
| `stat_base_attack` (trong active) | `active` | **Damage** | `type: "FLAT", value: X` | Gộp vào flat damage của đòn đánh |
| `damage_path` | `active` | **Damage** | `path: "X Path"` | Override lưu phái đòn đánh |
| `ignore_defense` | `active` | **Damage** | `penetration: {mode: "FLAT", value: X}` | Bỏ qua phòng thủ đối phương |
| `ignore_defense_pct` | `active` | **Damage** | `penetration: {mode: "PERCENT", value: X}` | Bỏ qua % phòng thủ đối phương |
| `shield_flat` | `active` | **Shield** | `type: "FLAT", value: X` | Tạo khiên ảo cho bản thân/đồng đội |
| `shield_mult` | `active` | **Shield** | `type: "MULTIPLIER", value: X` | Hệ số khuếch đại khiên |
| `heal_flat` | `active` | **Heal** | `resource: "HP", type: "FLAT", value: X` | Hồi máu trực tiếp |
| `mana_regen` | `active` | **Heal** | `resource: "MANA", type: "FLAT", value: X` | Hồi phục chân nguyên |
| `lifesteal_rate` | `active` | **Heal** | `resource: "HP", type: "LIFESTEAL", value: X` | Hút máu theo % damage gây ra |
| `blood_drain: true` | `active` | **Heal** | `resource: "HP", type: "LIFESTEAL", value: 0.5` | Loại bỏ cờ bool, quy về giá trị 0.5 |
| `damage_hp_percent` | `active` | **DirectHP** | `sub_type: "MAX_HP_PERCENT", value: X` | Sát thương ăn thẳng máu bỏ qua khiên |
| `shield_flat_reduce_percent` | `active` | **DirectHP** | `sub_type: "SHIELD_STRIP", value: X` | Hủy % khiên ảo của mục tiêu |
| `BYPASS_SHIELD` | `other` (`ON_HIT`) | **DirectHP** | `sub_type: "DIRECT_HP_DAMAGE"` | Đòn đánh xuyên qua virtual shield |
| `SOUL_WOUND` | `statusfx` | **DirectHP** + **Status** | `status_id: "SOUL_WOUND", DirectHP: true` | DoT xuyên giáp gây tổn thương hồn |
| `StatBaseAttackFlat` | `ActiveEffect` | **StatMod** / **Damage** | Gộp vào `Damage` flat | Không duy trì trường riêng |
| `speed_boost_flat` | `active` | **StatMod** | `stat: "SPEED", mode: "ADD_FLAT", value: X` | Tăng tốc độ flat |
| `all_stats_buff_percent` | `active` | **StatMod** | `stat: "ALL", mode: "MULTIPLY_PERCENT", value: X` | Tăng toàn diện chỉ số |
| `crit_rate_boost` | `active` | **StatMod** | `stat: "CRIT_RATE", mode: "ADD_FLAT", value: X` | Tăng tỉ lệ bạo kích |
| ~~`self_evasion_boost_percent`~~ | ~~`active`~~ | — | `apply_status: [{id:"EVASION",value:X,...}]` | **Đã thực thi**: gỡ field, migrate 64 dòng `Gu.csv` sang status `EVASION` |
| `wood_transformation: true` | `active` | **StatMod** | `stat: "DEFENSE", mode: "ADD_FLAT", value: 30` | Xóa cờ dị biệt, đưa về StatMod +30 |
| `refine_mana_tier` (trong active) | `active` | **StatMod** | `stat: "ATTACK", mode: "MULTIPLY_PERCENT", value: X*0.2` | Đưa về hệ số tăng công chuẩn |
| `stat_base_*` (passive) | `passive` | **StatMod** | `stat: "*", mode: "ADD_FLAT", duration: 0` | Chỉ số tĩnh ALWAYS_ON |
| `luck`, `movement`, `hearing`... | `passive` | **StatMod** | `stat: "*", mode: "ADD_FLAT", duration: 0` | Thuộc tính tĩnh ALWAYS_ON |
| ~~`accuracy`, `evasion`~~ | ~~`passive`~~ | — | — | **Đã gỡ khỏi hệ thống**: `accuracy` trừ thẳng (tới 1000) vào xác suất né 0..1 nên vô hiệu hoá né tránh; `evasion` → status `EVASION` |
| `damage_multiplier_on_hit` | `passive` | **StatMod** | `stat: "DAMAGE_DEALT", mode: "MULTIPLY_PERCENT"` | Thay thế `DOUBLE_DAMAGE_DEALT` |
| `DOUBLE_DAMAGE_DEALT` | `other` (`PASSIVE`) | **StatMod** | `stat: "DAMAGE_DEALT", mode: "MULTIPLY_PERCENT", value: 2.0` | Thống nhất với `damage_multiplier_on_hit` |
| Shorthand: `freeze`, `stun`, `silence`, `taunt` | `active` | **StatusEffect** | `status_id: "ID", turns: X` | **Loại bỏ shorthand**, chỉ dùng StatusOp |
| Shorthand: `slow`, `slow1`, `weak`, `thorns`, `regen`, `atk_up`, `def_up` | `active` | **StatusEffect** | `status_id: "ID", intensity: Y, turns: X` | **Loại bỏ shorthand**, dùng StatusOp |
| `SimAction.*Tier` (12 trường) | `SimAction` | **StatusEffect** | `StatusOps: []statusfx.StatusOp` | **Xóa bỏ toàn bộ các trường *Tier** |
| `disable_gu_next_turn` | `active` | **StatusEffect** | `status_id: "GU_SEAL", turns: 1` | **Xóa cờ bool**, dùng chung `GU_SEAL` |
| `DisableGuTurns` | `SimAction` | **StatusEffect** | Nằm trong `StatusOps` | **Xóa trường riêng** trong SimAction |
| `FORGET_EGO` | `statusfx` | **StatusEffect** | `status_id: "GU_SEAL"` (alias hoặc gộp) | Cùng chung cơ chế khóa Cổ |
| `target_damage_reduce_percent` | `active` | **StatusEffect** | `status_id: "WEAK", intensity: X*100` | Thay thế bằng status `WEAK` chuẩn |
| `FORGET_REALITY` | `statusfx` | **StatusEffect** | `status_id: "WEAK"` (lưu phái mộng/trí) | Dùng chung behavior `weak` |
| `SimAction.BleedPerTurn` | `SimAction` | **StatusEffect** | `status_id: "BLEED", intensity: X` | Bỏ trường độc lập trong SimAction |
| `SimAction.PoisonPerTurn` | `SimAction` | **StatusEffect** | `status_id: "POISON", intensity: X` | Bỏ trường độc lập trong SimAction |
| `invulnerable_ticks` | `active` | **Barrier** | `barrier_type: "INVINCIBLE", turns: X` | Hợp nhất với status `INVINCIBLE` |
| `immune_dream_status` | `passive` | **Barrier** | `barrier_type: "STATUS_IMMUNITY", path: "dream"` | Miễn nhiễm debuff theo phái |
| `cleansing_power` | `active` / `passive` | **Cleanse** | `target_type: "DISPEL_DEBUFF", count: X` | Biến cờ giả thành hành động giải trừ thực |
| `REBORN` | `other` (`ON_DEATH`) | **UniqueMechanic** | `mechanic_code: "REBORN"` | Cơ chế hồi sinh có tham số |
| `SUMMON_PHANTOM` | `active` / `other` | **UniqueMechanic** | `mechanic_code: "SUMMON_PHANTOM"` | Hợp nhất summon đơn & summon bầy |
| `SEIZE_INITIATIVE` | `other` (`ON_BATTLE_START`) | **UniqueMechanic** | `mechanic_code: "SEIZE_INITIATIVE"` | Cướp quyền đánh trước dựa trên Thọ nguyên |
| `DISGUISE`, `REMOTE_TRADE`... | `other` (`CAPABILITY`) | **UniqueMechanic** | `mechanic_code: "CAPABILITY_NAME"` | Giữ nguyên vai trò ngoài combat |
| `curve_attack` | `active` | **Targeting Modifier** | `target: { bypass_frontline: true }` | Thuộc tính của Target trong Block |
| `target_selector` | `active` | **Targeting Modifier** | `target: { selector: "ROW"|"COL"|"ALL" }` | Thuộc tính của Target trong Block |
| `replicate`, `generate_gu`, `range` | `active` | **LOẠI BỎ** | Xóa khỏi active runtime | Key rác không có logic gameplay |
