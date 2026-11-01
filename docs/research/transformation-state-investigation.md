# Báo Cáo Nghiên Cứu Kỹ Thuật: Rà Soát Cấu Trúc Entity, Buff System và Loadout Cho Transformation State

- **Mã Issue**: [#283](https://github.com/VipPro67/GuWorld/issues/283)
- **Map Cha**: [#282](https://github.com/VipPro67/GuWorld/issues/282) (Triển khai Biến Thân & Hình Thái Chiến Đấu)
- **Nhánh**: `research/transformation-state-283`
- **Tài liệu Domain liên quan**: `CONTEXT.md` (Domain Biến Hóa Đạo & Hình Thái Chiến Đấu), `docs/PLAYER_DOMAIN.md`, `docs/STATUS_EFFECTS.md`

---

## Tóm Tắt Điều Hành (Executive Summary)

Nghiên cứu này rà soát hiện trạng mã nguồn thực thi chiến đấu thời gian thực (Realtime Zone Sim 20Hz) tại `backend/internal/realtime/sim/`, hệ thống biên dịch hiệu ứng `backend/internal/realtime/effects/`, bộ nạp kỹ năng `backend/internal/handlers/skill_loadout.go` và kho dữ liệu `backend/data/Gu.csv`.

**Kết luận cốt lõi:**
1. **Entity & CombatState**: Đã có sẵn một prototype sơ khai về Form trong `CombatState` (`FormActive`, `FormManaDrainPerTick`, `FormDefenseBonus`), nhưng hoàn toàn là hardcode đơn biến, chưa hỗ trợ định danh hình thái (`FormID`), phân loại hình thái (`FormKind`), biến đổi kích thước/hitbox (`Scale`), lượng máu riêng biệt của hình thái (`FormHP`), tiêu hao đa tài nguyên (`UpkeepPerTick`), và hoán đổi thanh kỹ năng (`OriginalSkills` vs `TransformedSkills`).
2. **Buff System (`StatusEffect`) vs Form Stance Subsystem**: `StatusEffect` hiện tại đã có cơ chế trừ Chân Nguyên (`ManaDrainPerTick`) và Niệm Đầu (`ThoughtDrainPerTick`), nhưng **không thể** dùng làm nền tảng cho Form Stance vì các nhược điểm chí mạng: dễ bị thanh tẩy (cleanse/dispel), chu kỳ kiểm tra dạng periodic thay vì vòng lặp duy trì 20Hz liên tục, không gắn kết được với hitbox/scale, máu hình thái (FormHP) và hoán đổi thanh chiêu thức. Do đó, **bắt buộc cần một subsystem Stance/Transformation chuyên trách trong Zone Sim**.
3. **Danh mục Cổ Biến Hóa Đạo (`Gu.csv`)**: Hiện có 105 mục thuộc `Transformation Path` và 37 mục có `gu_type == "Transformation"` hoặc role `TRANSFORMATION`. Đã có đầy đủ các phàm cổ và tiên cổ biến thú tiêu biểu như `GU_TRANSFORM_WOLF_1` (Lang Biến), `GU_TRANSFORM_FISH_2` (Ngư Biến), `GU_TRANSFORM_BEAR_3` (Hùng Biến), `GU_TRANSFORM_EAGLE_4` (Ưng Biến), `GU_TRANSFORM_DRAGON_I` (Giao Long Biến), `GU_TRANSFORM_MYRIAD_SHAPES_I` (Vạn Tượng Thiên Biến). Tuy nhiên, các Cổ biến đổi bộ phận kinh điển như `Lang Bì Cổ` (phàm phẩm 1-4) và `Quy Giáp Cổ` (phàm phẩm 1-5) chưa có mã riêng biệt trong CSV (mới chỉ có `GU_BEAST_SKIN` cấp 1 và `GU_TURTLE_JADE_WOLF_SKIN` cấp 5), cần bổ sung trong Task #286.

---

## 1. Khảo Sát Chi Tiết `Entity` & `CombatState`

### 1.1 Cấu Trúc Hiện Tại Của `Entity`
*Nguồn tham chiếu: [`backend/internal/realtime/sim/world.go:138-164`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/sim/world.go#L138-L164)*

```go
type Entity struct {
	ID             EntityID                 `json:"id"`
	CharacterID    CharacterID              `json:"character_id,omitempty"`
	Name           string                   `json:"name,omitempty"`
	Kind           EntityKind               `json:"kind"`
	Transform      Transform                `json:"transform"`
	Vitals         Vitals                   `json:"vitals"`
	Defense        float64                  `json:"-"`
	DaoProfile     *gamefx.DaoCombatProfile `json:"-"`
	PathResistance map[string]float64       `json:"-"`
	Speed          float64                  `json:"speed"`
	Radius         float64                  `json:"radius"`
	Faction        string                   `json:"faction"`
	PartyID        string                   `json:"party_id,omitempty"`
	PvPEnabled     bool                     `json:"pvp_enabled,omitempty"`
	Revision       uint64                   `json:"revision"`
	AckSeq         uint64                   `json:"ack_seq,omitempty"`
	LeaseToken     string                   `json:"-"`
	AI             AIState                  `json:"-"`
	Combat         CombatState              `json:"-"`
	Companion      CompanionState           `json:"-"`

	moveIntent    Vec2
	moveTarget    *Vec2
	targetTick    uint64
	lastInputTick uint64
}
```

**Nhận xét:**
- `Radius`: Đại diện cho bán kính va chạm/hitbox của entity trong thế giới (dùng bởi `collision.go` và thuật toán quét mục tiêu `firstCollisionAlongAim`). Hiện tại chỉ là một biến float64 tĩnh.
- `Vitals`: Lưu trữ `HP`, `MaxHP`, `Shield`. Hiện không có trường lưu độ bền riêng của hình thái biến thân hoặc Tiên Cổ Phòng (`FormHP`).
- `Companion`: Tiền lệ kiến trúc tốt nhất gần đây (PR #277) là thêm một struct trạng thái chuyên trách `CompanionState` trực tiếp trong `Entity`.

### 1.2 Cấu Trúc Hiện Tại Của `CombatState`
*Nguồn tham chiếu: [`backend/internal/realtime/sim/combat.go:66-91`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/sim/combat.go#L66-L91)*

```go
type CombatState struct {
	Mana                     float64
	MaxMana                  float64
	ManaRegenPerSecond       float64
	Thought                  float64
	MaxThought               float64
	ThoughtRegenPerSecond    float64
	BasicAttackDamage        float64
	GCDUntil                 uint64
	RecoveryUntil            uint64
	RecoveryPolicy           string
	BasicAttackCooldownUntil uint64
	Cooldowns                [13]uint64
	Skills                   [13]combat.Skill
	GuHunger                 [13]float64
	GuUses                   [13]uint64
	GuMasteryUses            [13]uint64
	KillerMoveUses           [13]uint64
	BasicAttackUses          uint64
	FormActive               bool
	FormManaDrainPerTick     float64
	FormDefenseBonus         float64
	Pending                  *PendingCast
	Statuses                 []StatusEffect
	Hidden                   combat.HiddenPassive
}
```

**Các cơ chế Form đã có sẵn (Prototype):**
- Dòng 85-87: Đã có 3 trường:
  - `FormActive bool`: Cờ báo đang trong dạng biến thân.
  - `FormManaDrainPerTick float64`: Lượng Chân Nguyên rút mỗi tick (20Hz).
  - `FormDefenseBonus float64`: Điểm thủ cộng thêm khi biến thân.
- Dòng 297-302: Khi thi triển lại chiêu thức kích hoạt Form (`skill.FormActivation && caster.Combat.FormActive`), hệ thống toggle tắt biến thân.
- Dòng 304-307: Chiêu thức dạng biến thân (`skill.FormAbility`) bị từ chối với lý do `"form_inactive"` nếu `FormActive == false`.
- Dòng 412-433: Vòng lặp `tickCombat()` duyệt qua toàn bộ entities mỗi tick 50ms, kiểm tra nếu `FormActive` thì trừ Chân Nguyên `FormManaDrainPerTick`. Nếu không đủ Chân Nguyên thì hủy `FormActive = false`.
- [`backend/internal/realtime/sim/combat_damage.go:60-62`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/sim/combat_damage.go#L60-L62): Khi nhận sát thương, nếu `FormActive == true` thì cộng thêm `FormDefenseBonus` vào chỉ số thủ.

### 1.3 Thiếu Sót & Các Field Cần Bổ Sung Cho `TransformationState`

Prototype hiện tại không đáp ứng được yêu cầu của Map #282 vì thiếu các trường sau:

| Trường Cần Thêm | Kiểu Dữ Liệu | Mục Đích |
| :--- | :--- | :--- |
| `FormID` | `string` | Định danh hình thái biến thân (vd: `FORM_WOLF`, `FORM_ANCIENT_BEAR`, `FORM_GU_HOUSE_BAMBOO_BUILDING`). Client dựa vào ID này để render model/sprite/VFX. |
| `FormKind` | `TransformationKind` | Phân loại: `PARTIAL` (Biến thân bộ phận), `FULL` (Biến thân toàn phần), `GU_HOUSE` (Dung nhập Tiên Cổ Phòng). |
| `Scale` | `float64` | Tỉ lệ phóng to hitbox và ngoại hình (vd: Hóa gấu hoang thú khổng lồ scale 1.8x, Tiên Cổ Phòng scale 2.5x). Bán kính va chạm hiệu dụng: $R_{\text{eff}} = R_{\text{base}} \times \text{Scale}$. |
| `OriginalRadius` | `float64` | Lưu lại bán kính gốc của nhân vật để hoàn nguyên khi giải trừ biến thân. |
| `FormHP` / `MaxFormHP` | `float64` | Lớp máu riêng biệt của hình thái (hoặc độ bền Tiên Cổ Phòng). Khi nhận đòn, trừ vào FormHP trước; nếu FormHP về 0, hình thái vỡ vụn (`Form Broken`), gây choáng/phản phệ. |
| `UpkeepPerTick` | `UpkeepCost` | Thay vì chỉ trừ Mana, cấu trúc hỗ trợ: Mana (Chân Nguyên), Thought (Niệm Đầu - duy trì biến thân phức tạp), ImmortalEssence (Tiên Nguyên cho Tiên cấp). |
| `OriginalSkills` | `*[13]combat.Skill` | Mảng sao lưu bộ chiêu thức ban đầu của nhân vật (Slots 1-6 và Slots 9-12). |
| `TransformedSkills` | `[13]combat.Skill` | Bộ chiêu thức chuyên biệt của hình thái (ví dụ khi hóa Lang: Cắn Xé, Sói Vồ, Gầm Thét). |
| `SourceGuID` | `string` | Cổ trùng kích hoạt (để tính toán Chân Ý và tiêu hao độ đói). |
| `SourceKillerMoveID` | `string` | Sát chiêu biến thân kích hoạt (nếu là Sát Chiêu Biến Hóa Đạo). |

### 1.4 Thiết Kế Cấu Trúc Khuyến Nghị

Đề xuất tạo struct riêng `TransformationState` được nhúng vào `Entity` (cho đồng bộ world & wire snapshot) và liên kết với `CombatState`:

```go
type TransformationKind string

const (
	TransformationNone    TransformationKind = ""
	TransformationPartial TransformationKind = "PARTIAL"
	TransformationFull    TransformationKind = "FULL"
	TransformationGuHouse TransformationKind = "GU_HOUSE"
)

type UpkeepCost struct {
	ManaPerTick    float64 `json:"mana_per_tick,omitempty"`
	ThoughtPerTick float64 `json:"thought_per_tick,omitempty"`
}

type TransformationState struct {
	Active           bool               `json:"active"`
	FormID           string             `json:"form_id,omitempty"`
	FormKind         TransformationKind `json:"form_kind,omitempty"`
	Scale            float64            `json:"scale,omitempty"`
	FormHP           float64            `json:"form_hp,omitempty"`
	MaxFormHP        float64            `json:"max_form_hp,omitempty"`
	Upkeep           UpkeepCost         `json:"upkeep,omitempty"`
	DefenseBonus     float64            `json:"defense_bonus,omitempty"`
	SpeedMultiplier  float64            `json:"speed_multiplier,omitempty"`
	OriginalRadius   float64            `json:"-"`
	OriginalSkills   *[13]combat.Skill  `json:"-"`
	TransformedSkills [13]combat.Skill  `json:"-"`
	SourceGuID       string             `json:"source_gu_id,omitempty"`
	SourceKillerMove string             `json:"source_killer_move,omitempty"`
}
```

---

## 2. Rà Soát Hệ Thống Buff / Status (`StatusEffect`) & Phân Tích Upkeep Drain

### 2.1 Cơ Chế Hiện Tại Trong `status.go`
*Nguồn tham chiếu: [`backend/internal/realtime/sim/status.go:11-28`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/sim/status.go#L11-L28) và [`status.go:179-206`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/sim/status.go#L179-L206)*

```go
type StatusEffect struct {
	ID                  string
	SourceID            EntityID
	ExpireTick          uint64
	DurationTicks       uint64
	PeriodTicks         uint64
	NextTick            uint64
	DamagePerTick       float64
	HealPerTick         float64
	MoveMultiplier      float64
	ManaDrainPerTick    float64
	ThoughtDrainPerTick float64
	Stacks              int
	MaxStacks           int
	StackPolicy         string
	Debuff              bool
	PvPDrCategory       string
}
```

Trong hàm `tickStatuses()` (dòng 179-206):
- Khi `w.tick >= status.NextTick`:
  - `status.ManaDrainPerTick`: Rút Chân Nguyên. Nếu `entity.Combat.Mana < drain`, phát sinh `EventStatusExpired` và gỡ bỏ status.
  - `status.ThoughtDrainPerTick`: Rút Niệm Đầu. Nếu `entity.Combat.Thought < drain`, phát sinh `EventStatusExpired` và gỡ bỏ status.

### 2.2 Đánh Giá: `StatusEffect` Có Đáp Ứng Được Upkeep Drain Không?

| Tiêu Chí | `StatusEffect` Hiện Tại | Yêu Cầu Của Form Stance | Đánh Giá Khả Thi |
| :--- | :--- | :--- | :--- |
| **Tiêu hao Chân Nguyên / Niệm Đầu** | Có sẵn (`ManaDrainPerTick`, `ThoughtDrainPerTick`) | Duy trì Chân Nguyên / Tiên Nguyên / Niệm Đầu theo tick | **Đạt** |
| **Tần suất kiểm tra (Tick Rate)** | Periodic: nhảy theo `PeriodTicks` (vd: mỗi 1s) | Realtime 20Hz: tiêu hao mượt mà từng nhịp 50ms | **Bất cập**: Phải gán `PeriodTicks = 1`, làm phình to overhead lặp qua mảng `Statuses` |
| **Bảo vệ trước Dispel / Cleanse** | Dễ bị xóa bởi `cleanseDebuffsLocked` hoặc kỹ năng giải trừ buff | Biến thân là biến đổi nhục thân thực thể, không được phép bị giải trừ bởi các chiêu hóa giải thông thường | **Chí mạng (FAIL)**: Xung đột nghiêm trọng logic chiến đấu |
| **Thay đổi Kích Thước & Hitbox** | Không hỗ trợ (chỉ có `MoveMultiplier`) | Cần can thiệp `Entity.Radius` và scale của Transform | **Không đáp ứng (FAIL)** |
| **Lớp Máu Hình Thái (FormHP)** | Không hỗ trợ (chỉ có Shield chung trong Vitals) | Cần máu riêng biệt, vỡ vụn khi FormHP <= 0 | **Không đáp ứng (FAIL)** |
| **Hoán Đổi Kỹ Năng (Palette Swap)** | Không hỗ trợ | Cần tráo đổi `Combat.Skills` và khôi phục khi hết hạn | **Không đáp ứng (FAIL)** |
| **Ngắt Chiêu & Phản Phệ** | Chỉ xóa khỏi danh sách | Cần kích hoạt Backlash, choáng, fatigue khi cạn kiệt | **Không đáp ứng (FAIL)** |

### 2.3 Kết Luận Subsystem
> [!IMPORTANT]
> **Không thể dùng `StatusEffect` để quản lý Form Stance.**
> Cần xây dựng một **Form Stance Subsystem riêng biệt** nằm trong Zone Sim (tương tự như cách `tickCombat()` hiện tại đang xử lý `FormActive`, hoặc tách thành file chuyên trách `backend/internal/realtime/sim/transformation.go`).
> `StatusEffect` chỉ nên giữ vai trò bổ trợ các hiệu ứng buff/debuff đi kèm hình thái (ví dụ: bốc cháy `BURN`, xuất huyết `BLEED`, tăng tốc `HASTE`).

---

## 3. Rà Soát Danh Mục Cổ Biến Hóa Đạo Trong `backend/data/Gu.csv`

Khảo sát toàn bộ 4.636 dòng Cổ trùng trong `backend/data/Gu.csv`:
- Tổng số Cổ thuộc đạo `Transformation Path`: **105 Cổ**.
- Tổng số Cổ có `gu_type == "Transformation"` hoặc role `TRANSFORMATION`: **37 Cổ**.

### 3.1 Nhóm Biến Thân Toàn Phần (Full Metamorphosis)

Đây là các Cổ cho phép biến đổi toàn bộ cơ thể thành Thú hoặc Ma Thần:

| Mã Cổ (ID) | Rank | Tên Cổ Trùng | Loại | Vai Trò | Mô Tả & Hiệu Lực Thực Tế |
| :--- | :---: | :--- | :--- | :--- | :--- |
| `GU_TRANSFORM_WOLF_1` | 1 | Nhất Chuyển Lang Biến Cổ | Transformation | TRANSFORMATION | Hóa thân sói hoang: mọc nanh vuốt sói, tăng mạnh tốc độ di chuyển và đòn cắn. |
| `GU_TRANSFORM_FISH_2` | 2 | Nhị Chuyển Ngư Biến Cổ | Transformation | TRANSFORMATION | Mọc vảy và mang cá, bơi lội dưới nước, tăng né tránh và tốc độ bơi. |
| `GU_TRANSFORM_BEAR_3` | 3 | Tam Chuyển Hùng Biến Cổ | Transformation | TRANSFORMATION | Hóa thân cự hùng: thể hình tăng vọt, tăng mạnh phòng ngự và lực đập. |
| `GU_TRANSFORM_EAGLE_4` | 4 | Tứ Chuyển Ưng Biến Cổ | Transformation | TRANSFORMATION | Hóa thân hùng ưng: mọc đôi cánh đại bàng, cho phép lướt nhanh và tấn công từ không trung. |
| `GU_TRANSFORM_CHIMERA_5` | 5 | Ngũ Chuyển Bách Thú Biến Cổ | Transformation | TRANSFORMATION | Kết hợp đặc tính nhiều loài thú, đa dạng hóa thuộc tính công thủ. |
| `GU_TRANSFORM_DRAGON_I` | 6 | Giao Long Biến Tiên Cổ | Transformation | TRANSFORMATION | Tiên Cổ Lục Chuyển, biến thân Giao Long cuộn trào sóng nước, mở rộng hitbox diện rộng. |
| `GU_TRANSFORM_MYRIAD_SHAPES_I` | 7 | Vạn Tượng Thiên Biến Tiên Cổ | Transformation | TRANSFORMATION | Thất Chuyển Tiên Cổ: biến hóa tùy ý thành bất kỳ hình thái nào đã thấu hiểu đạo ngân. |
| `GU_TRANSFORM_ANCESTOR_I` | 9 | Biến Hóa Khởi Nguyên Tiên Tôn Tiên Cổ | Transformation | TRANSFORMATION | Cửu Chuyển Tiên Cổ của Cuồng Man Ma Tôn, hình thái chí tôn vô địch biến hóa. |
| `GU_BEAST_BODY_3..5, I` | 3..9 | Thú Thể Cổ / Thú Thể Tiên Cổ | Transformation | TRANSFORMATION | Biến đổi hoàn toàn cơ thể thành dã thú hoang dã. |

### 3.2 Nhóm Biến Thân Bộ Phận (Partial Transformation)

Các Cổ chỉ biến đổi một phần chi thể, tăng cường chỉ số và mở khóa kỹ năng đặc thù:

1. **Lớp Da & Giáp Ngoài (Skin / Hide / Armor)**:
   - `GU_BEAST_SKIN` (Rank 1): Thú Bì Cổ - Tăng độ dày lớp da phòng ngự.
   - `GU_BLACK_BRISTLE` (Rank 2): Hắc Tông Cổ - Biến lông tơ thành lớp giáp đen thô ráp.
   - `GU_TURTLE_JADE_WOLF_SKIN` (Rank 5): Quy Ngọc Lang Bì Cổ - Cổ trùng cải tạo màng da tối cao, kết hợp màng da rùa ngọc và da sói.
   - `GU_BRISTLE_BRISTLE_1..5, I`: Tông Tông Cổ / Tiên Cổ - Cường hóa lông bờm dã thú.
2. **Vuốt & Chi Thể Tấn Công (Claws / Limbs)**:
   - `GU_BRISTLE_CLAW_1` (Rank 1): Nhất Chuyển Tông Trảo Cổ - Mọc móng vuốt nhọn hoắt.
   - `GU_TRANSFORM_CLAW_1..5, I` (Rank 1..9): Biến Hóa Trảo Cổ - Series móng vuốt biến hóa từ phàm đến tiên.
3. **Giác Quan & Mắt (Eyes / Senses)**:
   - `GU_WOLF_EYE_1..5, I` (Rank 1..9): Lang Nhãn Cổ - Mắt sói nhìn trong đêm, tăng tầm nhìn và bạo kích.
   - `GU_TRANSFORM_EYE_1..5, I` (Rank 1..9): Biến Hóa Nhãn Cổ.
4. **Cánh & Dị Thể (Wings / Morphology)**:
   - `GU_BLOOD_BLADE_BAT` (Rank 3): Đao Sí Huyết Bức Cổ - Biến cánh tay thành đôi cánh dơi sắc bén như đao.

### 3.3 Khoảng Trống Dữ Liệu (Data Gaps) Cần Bổ Sung Cho Task #286

Khảo sát cho thấy trong `Gu.csv` hiện tại:
1. **Thiếu Lang Bì Cổ phàm phẩm chuỗi 1-4 (`GU_WOLF_SKIN_1..4`)**: Hiện tại chỉ có `GU_BEAST_SKIN` (Thú Bì) và nhảy cóc lên `GU_TURTLE_JADE_WOLF_SKIN` (Quy Ngọc Lang Bì) Rank 5. Cần bổ sung chuỗi `GU_WOLF_SKIN_1` đến `GU_WOLF_SKIN_4` cho Cổ sư phàm cấp tại Nam Cương.
2. **Thiếu Quy Giáp Cổ phàm phẩm chuỗi 1-5 (`GU_TURTLE_ARMOR_1..5`)**: Hiện tại chỉ có Cốt Giáp (`GU_BONE_ARMOR`) của Cốt đạo hoặc Thủy Giáp (`GU_WATER_ARMOR`) của Thủy đạo. Cần bổ sung Quy Giáp Cổ đặc trưng Biến Hóa Đạo.
3. **Dữ liệu `skill_profile` chưa có cấu trúc Form chuyên biệt**: Hầu hết các Cổ Biến Hóa trong CSV đang gán `skill_profile` dạng projectile tấn công đơn mục tiêu hoặc để trống (`null`), chưa có cấu hình `FormProfile` JSON với `windup_ms`, `mana_drain_per_sec`, `defense_bonus`, `scale`, `hitbox_radius`.

---

## 4. Khảo Sát Tương Tác Giữa Hệ Thống Kỹ Năng & Loadout

### 4.1 Cơ Chế Loadout Hiện Tại
*Nguồn tham chiếu: [`backend/internal/realtime/effects/loadout.go:59-65`](file:///home/vippro67/Documents/GuWorld/backend/internal/realtime/effects/loadout.go#L59-L65) và [`handlers/skill_loadout.go:23-35`](file:///home/vippro67/Documents/GuWorld/backend/internal/handlers/skill_loadout.go#L23-L35)*

- **Slots 1–6**: Gán cho 6 Cổ trùng chủ động trang bị (`EquippedGu`).
- **Slot 7**: Hardcode cho `FormActivation` (kỹ năng niệm chú kích hoạt biến thân).
- **Slot 8**: Hardcode cho `FormAbility` (chiêu thức đặc thù duy nhất khi đang biến thân).
- **Slots 9–12**: Dành riêng cho Sát Chiêu (`KillerMove`) hoặc Trận Kích của Tiên Cổ Phòng (`gu_house`).

### 4.2 Nhược Điểm Của Kiến Trúc Slot 7 & 8 Hiện Tại
1. **Không hỗ trợ Skill Palette Swap**: Khi hóa gấu hay hóa rồng, người chơi chỉ có thêm đúng 1 chiêu ở Slot 8, các chiêu Slots 1-6 vẫn giữ nguyên bộ chiêu nhân loại phàm nhân. Điều này không đúng với tinh thần tiểu thuyết Cổ Chân Nhân (khi biến thành thái cổ hoang thú, toàn bộ thân xác vận hành bằng bản năng và chiêu thức của loài thú đó).
2. **Giới hạn 1 Form duy nhất**: Dòng 59 `effects/loadout.go` dùng cờ `!formAdded`, chặn không cho phép người chơi trang bị hoặc chuẩn bị nhiều dạng biến thân khác nhau.

---

## 5. Kiến Trúc Đề Xuất & Khuyến Nghị Cho Các Task Tiếp Theo

### 5.1 Cho Task #284 (Prototype: TransformationState & Vòng Lặp Duy Trì 20Hz)
- Tạo file mới `backend/internal/realtime/sim/transformation.go`.
- Nhúng `Transformation TransformationState` vào `Entity` struct.
- Thay thế vòng lặp đơn sơ trong `combat.go:tickCombat()` bằng `w.tickTransformations()`:
  - Giảm Mana/Thought/ImmortalEssence mỗi tick.
  - Kiểm tra điều kiện ngắt: Hết tài nguyên $\rightarrow$ Revert; FormHP <= 0 $\rightarrow$ Form Broken (gây Stun 1s + ngắt biến thân).
  - Cập nhật hitbox: `entity.Radius = entity.Transformation.OriginalRadius * entity.Transformation.Scale`.

### 5.2 Cho Task #285 (Prototype: Cơ Chế Hoán Đổi Kỹ Năng Skill Palette Swap)
- Khi `EnterTransformation(formID)`:
  - Sao chép `entity.Combat.Skills` sang `Transformation.OriginalSkills`.
  - Nạp các kỹ năng thú vào `entity.Combat.Skills[1..6]`.
  - Slot 7 trở thành chiêu "Hủy Biến Thân" (`CancelForm`).
- Khi `ExitTransformation()`:
  - Hoàn nguyên toàn bộ `entity.Combat.Skills` từ `Transformation.OriginalSkills`.
  - Đặt thời gian hồi phục (`RecoveryUntil`) để tránh spam biến thân liên tục.

### 5.3 Cho Task #286 (Task: Schema & CSV Seed Cho Cổ Trùng & Sát Chiêu Biến Hóa Đạo Mẫu)
- Thêm seed dữ liệu cho:
  - `GU_WOLF_SKIN_1` đến `GU_WOLF_SKIN_4` (Lang Bì Cổ cấp 1-4).
  - `GU_TURTLE_ARMOR_1` đến `GU_TURTLE_ARMOR_5` (Quy Giáp Cổ cấp 1-5).
  - Chuẩn hóa `skill_profile` cho `GU_TRANSFORM_WOLF_1` và `GU_TRANSFORM_BEAR_3` với đầy đủ thông số `form` JSON.
- Thiết kế Sát Chiêu Biến Hóa Đạo mẫu (ví dụ: *Lang Biến Sát Chiêu*, *Cự Hùng Bạo Chấn*).

### 5.4 Cho Task #287 & #288 (Tiên Cổ Phòng & Wire Protocol Sync)
- Mở rộng wire protocol trong `backend/internal/realtime/protocol/protocol.go`:
  - Thêm `FormID string`, `FormKind string`, `Scale float64`, `FormHP float64`, `MaxFormHP float64` vào Entity snapshot delta.
  - Client Next.js/Canvas render aura/hitbox tương ứng theo `Scale`.

---

## 6. Danh Mục File và Dòng Code Trích Dẫn (Primary Citations)

1. `backend/internal/realtime/sim/world.go:138-164` – Khai báo `Entity` struct.
2. `backend/internal/realtime/sim/combat.go:66-91` – Khai báo `CombatState` struct với các trường form nguyên thủy.
3. `backend/internal/realtime/sim/combat.go:297-307` – Logic toggle form và chặn `FormAbility` khi form inactive.
4. `backend/internal/realtime/sim/combat.go:412-433` – Vòng lặp `tickCombat()` rút mana theo tick.
5. `backend/internal/realtime/sim/combat_damage.go:60-62` – Tính toán cộng dồn `FormDefenseBonus`.
6. `backend/internal/realtime/sim/status.go:11-28, 179-206` – Định nghĩa `StatusEffect` và vòng lặp `tickStatuses()`.
7. `backend/internal/realtime/sim/status.go:131-136` – Xử lý Hard CC ngắt chiêu niệm `FormActivation`.
8. `backend/internal/realtime/effects/profile.go:48-56, 306-328` – Định nghĩa `FormProfile` và hàm biên dịch Slot 7, Slot 8.
9. `backend/internal/realtime/effects/loadout.go:59-65` – Cơ chế nhồi kỹ năng Form vào loadout.
10. `backend/internal/handlers/skill_loadout.go:23-35` – Định nghĩa slots cho Sát Chiêu và Cổ Phòng (Slots 9-12).
11. `backend/data/Gu.csv` – 105 mục dữ liệu `Transformation Path` và các Cổ trùng biến thân.
