# Design Cơ Chế Xung Đột Đạo Ngân – GuWorld

> Tài liệu thiết kế cơ chế gameplay lấy cảm hứng từ **Cổ Chân Nhân**, nhưng được điều chỉnh để phù hợp với game có khoảng **50 lưu phái** và hệ thống combat / sát chiêu / đạo ngân / cảnh giới lưu phái của GuWorld.
>
> Mục tiêu chính: biến Đạo Ngân thành một hệ thống vừa tăng sức mạnh, vừa tạo ra **trade-off** giữa chuyên tu – kiêm tu – đa tu, đồng thời ảnh hưởng đến:
>
> - Sát thương đầu ra
> - Sát thương nhận vào
> - Hiệu quả sát chiêu
> - Tỷ lệ thi triển thành công
> - Độ ổn định sát chiêu
> - Buff / debuff / hồi phục / khống chế
> - Phòng thủ tự nhiên của Cổ Tiên
> - Song tu / phức hợp sát chiêu
> - Cảnh giới lưu phái
> - Đặc tính “Đạo Ngân Bất Tương Xung”

---

# 1. Mục tiêu thiết kế

Hệ thống Đạo Ngân không nên chỉ là:

```text
100 Đạo Ngân = +10% damage
```

Nếu chỉ làm như vậy thì người chơi đơn giản farm càng nhiều Đạo Ngân càng tốt, không có lý do để quan tâm:

- Đạo Ngân thuộc lưu phái nào
- Cảnh giới lưu phái
- Kiêm tu hay chuyên tu
- Sát chiêu thuộc hệ nào
- Môi trường chiến đấu
- Đặc tính thể chất
- Khả năng điều hòa nhiều lưu phái

Cơ chế mới nên khiến:

```text
Đạo Ngân = lượng sức mạnh / nền tảng quy tắc
Cảnh giới = mức độ hiểu và kiểm soát lưu phái
Mastery = độ thành thạo kỹ thuật / sử dụng
Compatibility = mức độ tương thích giữa các lưu phái
```

Bốn yếu tố này có vai trò riêng.

---

# 2. Triết lý cốt lõi

## 2.1. Đạo Ngân cùng lưu phái khuếch đại sát chiêu

Ví dụ:

```text
Hỏa đạo sát chiêu
+
Hỏa đạo đạo ngân
=
Uy lực tăng
```

Nhưng:

```text
Hỏa đạo sát chiêu
+
Thủy đạo đạo ngân
=
Có thể gây xung đột
```

Xung đột không nhất thiết khiến chiêu thức thất bại hoàn toàn.

Nó nên tác động theo nhiều mức:

```text
nhẹ  → giảm hiệu quả
vừa  → giảm damage / duration / healing
cao  → tăng mana cost / tăng instability
rất cao → có nguy cơ sát chiêu thất bại / phản phệ
```

---

# 3. Các lớp ảnh hưởng của xung đột Đạo Ngân

Một sát chiêu nên đi qua các tầng sau:

```text
Base Skill
    ↓
Cảnh giới lưu phái
    ↓
Đạo Ngân cùng hệ
    ↓
Xung đột Đạo Ngân nội tại
    ↓
Độ ổn định sát chiêu
    ↓
Môi trường
    ↓
Đạo Ngân phòng thủ của mục tiêu
    ↓
Defense / Shield / Resistance
    ↓
Final Result
```

Không nên gom tất cả vào một multiplier duy nhất.

---

# 4. Cấu trúc dữ liệu cho 50 lưu phái

Không nên cấu hình thủ công 50 × 50 = 2.500 cặp.

Nên chia thành:

```text
Path
↓
Dao Group
↓
Group compatibility
↓
Pair override nếu có
```

Ví dụ:

```text
ELEMENTAL
MENTAL
SOUL
BODY
SPACE_TIME
RULE
INFORMATION
TRANSFORMATION
BLOOD
HEAVEN_EARTH
```

Ví dụ mapping:

```text
Fire Path        → ELEMENTAL
Water Path       → ELEMENTAL
Metal Path       → ELEMENTAL

Wisdom Path      → MENTAL
Emotion Path     → MENTAL

Soul Path        → SOUL

Strength Path    → BODY
Blood Path       → BLOOD

Space Path       → SPACE_TIME
Time Path        → SPACE_TIME

Information Path → INFORMATION
Rule Path        → RULE
```

---

# 5. Compatibility

Mỗi cặp lưu phái có:

```text
compatibility ∈ [0, 1]
```

Ý nghĩa:

```text
1.00 = gần như không xung đột
0.90 = xung đột rất nhẹ
0.75 = xung đột vừa
0.50 = xung đột mạnh
0.25 = cực kỳ xung đột
```

Không nên cho phần lớn cặp quá thấp.

Khuyến nghị:

```text
đa số cặp:       0.85 – 1.00
cặp đối lập:     0.55 – 0.80
cặp cực khắc:    0.30 – 0.55
```

---

# 6. Ma trận Group Compatibility

Ví dụ khởi đầu:

| Group A | Group B | Compatibility |
|---|---:|---:|
| ELEMENTAL | ELEMENTAL | 0.85 |
| ELEMENTAL | MENTAL | 0.92 |
| ELEMENTAL | SOUL | 0.88 |
| ELEMENTAL | SPACE_TIME | 0.90 |
| MENTAL | INFORMATION | 0.97 |
| MENTAL | SOUL | 0.95 |
| BODY | BLOOD | 0.96 |
| BODY | SOUL | 0.87 |
| SPACE_TIME | RULE | 0.95 |
| INFORMATION | RULE | 0.98 |

Sau đó override các cặp đặc biệt:

```text
Fire ↔ Water       = 0.55
Fire ↔ Ice         = 0.60
Wisdom ↔ Info      = 0.98
Soul ↔ Wisdom      = 0.95
Strength ↔ Blood   = 0.94
Space ↔ Time       = 0.97
```

Các con số trên là **balance game**, không phải con số canon.

---

# 7. Raw Conflict Score

Giả sử người chơi đang dùng sát chiêu thuộc `mainPath`.

Xung đột đến từ tất cả Đạo Ngân ngoại hệ:

```text
RawConflict =
Σ OtherDaoMarks × (1 - Compatibility(mainPath, otherPath))
```

Ví dụ:

```text
Player:
Fire     = 20.000
Water    = 10.000
Wisdom   = 5.000
Soul     = 2.000
```

Sử dụng Fire skill:

```text
Fire ↔ Water  = 0.55
Fire ↔ Wisdom = 0.90
Fire ↔ Soul   = 0.85
```

Ta có:

```text
Water conflict
= 10.000 × 0.45
= 4.500

Wisdom conflict
= 5.000 × 0.10
= 500

Soul conflict
= 2.000 × 0.15
= 300
```

Tổng:

```text
RawConflict = 5.300
```

---

# 8. Ảnh hưởng của Cảnh Giới Lưu Phái

Cảnh giới không nên trực tiếp cộng Đạo Ngân.

Nó thể hiện:

```text
hiểu quy luật
khả năng điều khiển
khả năng điều hòa
khả năng tạo sát chiêu
khả năng xử lý xung đột
```

Ví dụ hệ cảnh giới:

```text
COMMON
MASTER
GRANDMASTER
GREAT_GRANDMASTER
QUASI_SUPREME
SUPREME_GRANDMASTER
```

Khuyến nghị:

| Realm | Conflict Control |
|---|---:|
| Common | 0% |
| Master | 10% |
| Grandmaster | 25% |
| Great Grandmaster | 45% |
| Quasi Supreme | 65% |
| Supreme Grandmaster | 80% |

Công thức:

```text
EffectiveConflict
=
RawConflict × (1 - RealmControl)
```

Ví dụ:

```text
RawConflict = 5.300
Great Grandmaster = 45%
```

Kết quả:

```text
EffectiveConflict
=
5300 × 0.55
=
2915
```

---

# 9. Conflict Ratio

Không nên dùng raw conflict trực tiếp.

Nên chuẩn hóa theo quy mô Đạo Ngân.

Ví dụ:

```text
ConflictRatio
=
EffectiveConflict
/
(MainDaoMarks + ConflictScale)
```

Trong đó:

```text
ConflictScale = 50.000
```

hoặc scale theo rank.

Ví dụ:

```text
Main Fire = 20.000
EffectiveConflict = 2.915
Scale = 50.000
```

```text
ConflictRatio
=
2915 / 70000
≈ 0.0416
```

---

# 10. Conflict Penalty – Diminishing Return

Không dùng:

```text
penalty = 1 - conflict
```

vì rất dễ âm hoặc phá balance.

Nên dùng:

```text
ConflictMultiplier
=
1 / (1 + ConflictRatio × K)
```

Ví dụ:

```text
K = 2
ConflictRatio = 0.0416
```

```text
Multiplier
=
1 / (1 + 0.0832)
≈ 0.923
```

Tức giảm khoảng:

```text
7.7%
```

---

# 11. Ảnh hưởng đến sát thương đầu ra

Damage cuối:

```text
FinalDamage
=
BaseDamage
× DaoAmplification
× RealmEfficiency
× SkillMastery
× InternalConflictMultiplier
× EnvironmentMultiplier
× TargetDaoResistance
× DefenseMultiplier
```

---

# 12. Dao Amplification

Không nên tăng vô hạn tuyến tính.

Một lựa chọn:

```text
DaoAmplification
=
1 + DaoMarks / DaoScale
```

Ví dụ:

```text
DaoScale = 10.000
20.000 Fire marks
```

```text
Amplification = 3.0
```

Nhưng về late game nên thêm diminishing return:

```text
DaoAmplification
=
1 + log2(1 + DaoMarks / 1000)
```

hoặc theo Rank.

---

# 13. Realm Efficiency

Cảnh giới cũng tăng khả năng sử dụng Đạo Ngân:

| Realm | Efficiency |
|---|---:|
| Common | 1.00 |
| Master | 1.05 |
| Grandmaster | 1.12 |
| Great Grandmaster | 1.20 |
| Quasi Supreme | 1.30 |
| Supreme Grandmaster | 1.45 |

Điều này khiến:

```text
nhiều Đạo Ngân + cảnh giới thấp
```

khác với:

```text
ít Đạo Ngân hơn + cảnh giới cao
```

---

# 14. Ví dụ damage đầu ra

Player:

```text
Fire marks      = 20.000
Water marks     = 10.000
Wisdom marks    = 5.000

Fire Realm      = Great Grandmaster
Base Damage     = 1.000
```

Giả sử:

```text
Dao Amplification   = ×3.0
Realm Efficiency    = ×1.20
Conflict Multiplier = ×0.92
```

Final:

```text
1000 × 3 × 1.2 × 0.92
≈ 3312
```

---

# 15. Xung đột ảnh hưởng sát thương nhận

Đây là phần rất quan trọng.

Cổ Tiên bình thường có thể nhận một lớp phòng thủ tự nhiên do:

```text
Đạo Ngân bản thân
↕
Đạo Ngân / sát chiêu ngoại lai
=
xung đột
```

Tức sát chiêu của địch bị suy yếu khi xâm nhập cơ thể.

---

# 16. Natural Dao Resistance

Khi bị một đòn thuộc `attackPath`:

```text
DefenseConflict =
Σ DefenderDaoMarks × (1 - Compatibility(attackPath, defenderPath))
```

Nhưng cần giới hạn để tránh tank vô hạn.

Ví dụ:

```text
NaturalResistance
=
MaxResistance
×
DefenseConflict
/
(DefenseConflict + ResistanceScale)
```

Ví dụ:

```text
MaxResistance = 0.35
ResistanceScale = 50.000
DefenseConflict = 30.000
```

```text
Resistance
=
0.35 × 30000 / 80000
≈ 13.1%
```

Damage nhận:

```text
100.000
→ 86.900
```

---

# 17. Vì sao không cho Natural Resistance quá cao

Nếu cho tối đa:

```text
80% / 90%
```

thì late game sẽ trở thành bất tử.

Khuyến nghị:

```text
Natural Dao Resistance cap:
20% – 35%
```

Phần còn lại nên đến từ:

```text
Defense
Shield
Killer Move
Gu effect
Formation
Immortal Gu House
```

---

# 18. Đạo Ngân Bất Tương Xung

Trait:

```text
NON_CONFLICTING_DAO_MARKS
```

Tác dụng:

### Ưu điểm

```text
InternalConflict = 0
```

Khi sử dụng sát chiêu:

```text
không bị Đạo Ngân ngoại hệ của bản thân làm giảm hiệu quả
```

Rất mạnh cho:

```text
đa tu
song tu
compound killer move
đa lưu phái
```

### Nhược điểm

Khi bị tấn công:

```text
NaturalDaoResistance = 0
```

vì Đạo Ngân bản thân không xung đột với đạo của địch.

Điều này tạo trade-off:

```text
Công mạnh
đa dụng
dễ đa tu

nhưng

thủ yếu hơn tự nhiên
```

---

# 19. Không nên implement thành “nhận thêm damage”

Không nên:

```go
if NonConflictingDaoMarks {
    incomingDamage *= 1.25
}
```

Nên:

```go
if NonConflictingDaoMarks {
    naturalDaoResistance = 0
}
```

Khác biệt rất quan trọng.

Người có trait không bị debuff.

Họ chỉ mất đi lớp giảm damage tự nhiên mà người khác có.

---

# 20. Tùy Ý Tương Xung – cấp cao hơn

Có thể tạo trait endgame:

```text
CONTROLLABLE_DAO_CONFLICT
```

Khi tấn công:

```text
InternalConflict = 0
```

Khi phòng thủ:

```text
NaturalDaoResistance = enabled
```

Tức:

```text
Offense → bất tương xung
Defense → chủ động tương xung
```

Đây nên là trait cực hiếm / late-game / rank rất cao.

---

# 21. Ảnh hưởng đến hiệu quả sát chiêu

Không phải skill nào cũng gây damage.

ConflictMultiplier có thể áp lên:

```text
damage
healing
shield
buff
debuff
duration
movement
speed
storage
soul effect
deduction
refinement
tracking
sealing
```

Nhưng mỗi loại skill nên có trọng số khác nhau.

---

# 22. Skill Conflict Sensitivity

Mỗi skill có:

```text
conflict_sensitivity
```

Ví dụ:

| Skill type | Sensitivity |
|---|---:|
| Basic Attack | 0.3 |
| Normal Killer Move | 0.6 |
| Complex Killer Move | 0.8 |
| Compound Killer Move | 1.0 |
| Healing | 0.5 |
| Movement | 0.4 |
| Sealing | 0.8 |
| Formation | 0.7 |

Công thức:

```text
SkillConflictMultiplier
=
1 - (1 - BaseConflictMultiplier)
× ConflictSensitivity
```

Ví dụ:

```text
BaseConflictMultiplier = 0.80
Sensitivity = 0.50
```

```text
SkillMultiplier
=
1 - 0.20 × 0.50
=
0.90
```

Skill ít nhạy với xung đột chỉ mất 10%.

---

# 23. Tỷ lệ thi triển thành công

Xung đột Đạo Ngân cũng có thể ảnh hưởng:

```text
success rate
```

Đặc biệt cho:

```text
complex killer move
compound killer move
formation
sealing
deduction
refinement-related killer move
```

Không nên ảnh hưởng mạnh tới skill cơ bản.

---

# 24. Base Success Rate

Ví dụ:

```text
Basic skill           = 100%
Normal killer move    = 99%
Complex killer move   = 95%
Compound killer move  = 90%
Experimental move     = 75%
```

Sau đó:

```text
SuccessRate
=
BaseSuccess
+ RealmBonus
+ MasteryBonus
- ConflictPenalty
- ComplexityPenalty
```

---

# 25. Conflict Failure Penalty

Ví dụ:

```text
ConflictFailurePenalty
=
min(
    MaxFailurePenalty,
    ConflictRatio × FailureScale
)
```

Khuyến nghị:

```text
MaxFailurePenalty = 25%
```

Không nên để một player có 50 path tự động xuống 0%.

---

# 26. Ví dụ success rate

Sát chiêu phức hợp:

```text
Base Success = 90%

Realm Bonus   = +8%
Mastery Bonus = +5%
Conflict      = -12%
Complexity    = -4%
```

Kết quả:

```text
90 + 8 + 5 - 12 - 4
= 87%
```

---

# 27. Không nên random thất bại cho mọi combat skill

Nếu mọi skill đều:

```text
87% thành công
```

thì combat dễ gây khó chịu.

Khuyến nghị:

### Skill thường

```text
Không fail
Chỉ giảm hiệu quả
```

### Sát chiêu phức tạp / experimental

```text
Có success chance
```

### Ultimate / Compound

```text
Có instability / backlash
```

Như vậy RNG có ý nghĩa mà không phá trải nghiệm.

---

# 28. Instability

Thay vì chỉ thành công / thất bại:

```text
Instability
```

có thể thú vị hơn.

Ví dụ:

```text
0 – 20   ổn định
20 – 40  tiêu hao thêm
40 – 60  giảm hiệu quả
60 – 80  nguy cơ fail
80 – 100 nguy cơ phản phệ
```

Công thức:

```text
Instability
=
BaseComplexity
+ ConflictScore
- RealmControl
- MasteryControl
```

---

# 29. Backlash

Khi sát chiêu thất bại có thể:

```text
mất mana
cooldown tăng
self damage
temporary debuff
Dao instability
Gu injury
soul damage
```

Không nhất thiết mọi failure đều gây damage.

Ví dụ:

```text
minor failure:
mana mất nhưng skill không ra

major failure:
mana + cooldown

backlash:
self damage / Gu damage / debuff
```

---

# 30. Mastery và Realm phải khác nhau

Đề xuất:

```text
Realm
=
hiểu bản chất lưu phái

Mastery
=
thành thạo kỹ thuật
```

Ví dụ:

Một người có:

```text
Fire Realm: Grandmaster
Fire Mastery: 30
```

nghĩa là:

```text
hiểu Hỏa đạo sâu
nhưng ít thực chiến / ít luyện skill cụ thể
```

Ngược lại:

```text
Fire Realm: Master
Fire Mastery: 95
```

là:

```text
rất thành thạo kỹ thuật đang dùng
nhưng trần hiểu biết thấp hơn
```

---

# 31. Mastery ảnh hưởng gì

Mastery nên ảnh hưởng:

```text
mana efficiency
cooldown
success rate
cast speed
refinement
deduction
skill stability
```

Không nên giảm toàn bộ Đạo Ngân conflict quá mạnh.

Realm mới là thứ chính xử lý xung đột.

---

# 32. Sát chiêu song đạo

Ví dụ:

```text
Fire + Water
```

Không nên dùng chỉ Realm Fire.

Cần xét Realm cả hai.

Đề xuất:

```text
CompoundRealmControl
=
min(
    RealmControl(Fire),
    RealmControl(Water)
)
```

Lý do:

```text
mắt xích yếu nhất giới hạn toàn bộ compound move
```

---

# 33. Alternative: geometric mean

Nếu muốn mềm hơn:

```text
CompoundControl
=
sqrt(FireControl × WaterControl)
```

Nhưng `min()` tạo cảm giác progression rõ hơn.

Khuyến nghị GuWorld:

```text
min()
```

---

# 34. Compound Compatibility

Một compound skill có thể khai báo:

```json
{
  "paths": ["FIRE", "WATER"],
  "compound_compatibility_bonus": 0.25
}
```

Nghĩa là bản thân sát chiêu đã được thiết kế để giải quyết xung đột.

Ví dụ:

```text
Fire ↔ Water base = 0.55
Compound bonus = +0.25
```

Effective:

```text
0.80
```

---

# 35. Truyền thừa song tu

Player có thể sở hữu:

```text
dual_path_harmony
```

Ví dụ:

```text
Fire + Water conflict reduction = 70%
```

Công thức:

```text
PairConflict
=
BasePairConflict
× (1 - HarmonyReduction)
```

---

# 36. Sát chiêu tam đạo trở lên

Mỗi thêm một path nên tăng complexity.

Ví dụ:

```text
1 path = ×1.00
2 paths = ×1.25
3 paths = ×1.60
4 paths = ×2.10
5 paths = ×3.00
```

Không nên cấm.

Nhưng cần:

```text
Realm cao
Mastery cao
Compatibility tốt
Inheritance / special trait
```

---

# 37. Healing

Healing trên Cổ Tiên nên bị ảnh hưởng bởi Đạo Ngân.

Nếu healer dùng Wood Path:

```text
Wood healing
```

nhưng target có大量 Fire / Metal marks xung đột:

```text
Healing effectiveness giảm
```

Công thức:

```text
FinalHealing
=
BaseHealing
× HealerAmplification
× TargetHealingCompatibility
```

---

# 38. Dị chủng Đạo Ngân trong vết thương

Có thể thêm mechanic nâng cao:

Khi trúng Immortal Killer Move:

```text
một phần attacking dao marks
→ residual dao marks
→ bám vào wound
```

Ví dụ:

```text
Sword wound:
ResidualSwordMarks = 300
```

Khi heal:

```text
healing first removes residual dao marks
then restores HP
```

Điều này giúp combat cấp cao có chiều sâu hơn.

---

# 39. Debuff / Control

Các hiệu ứng:

```text
stun
freeze
seal
soul suppression
thought disruption
tracking
curse
```

có thể tính:

```text
EffectStrength
=
BaseEffect
× DaoAmplification
× RealmEfficiency
× ConflictMultiplier
```

Mục tiêu chống lại:

```text
TargetResistance
=
TargetDaoConflictResistance
+ SpecificResistance
```

---

# 40. Duration

Không nhất thiết chỉ giảm strength.

Có thể:

```text
duration
=
baseDuration
× effectMultiplier
```

Ví dụ:

```text
10s stun
× 0.75
=
7.5s
```

---

# 41. Môi trường

Mỗi region có thể có:

```text
environment_dao_marks
```

Ví dụ:

```text
Fire region:
Fire 50.000
Earth 20.000
```

Sử dụng Fire skill:

```text
được environmental amplification
```

Water skill:

```text
có environmental conflict
```

---

# 42. Không nên tính toàn bộ 50 path mỗi hit

Nếu mỗi player có 50 path và mỗi hit loop 50 path:

```text
combat load tăng mạnh
```

Đặc biệt khi:

```text
nhiều player
nhiều skill
PvP
AoE
NPC
```

Nên cache.

---

# 43. Dao Combat Profile Cache

Khi Dao Marks / Realm thay đổi:

```text
recalculate profile
```

Ví dụ:

```go
type DaoCombatProfile struct {
    OffensiveConflict map[string]float64
    DefensiveResistance map[string]float64
    RealmEfficiency map[string]float64
}
```

Có thể cache 50 path.

Mỗi combat chỉ lookup:

```text
O(1)
```

---

# 44. Khi nào rebuild cache

Rebuild khi:

```text
dao marks thay đổi
realm tăng
trait thay đổi
equipped Gu thay đổi nếu Gu ảnh hưởng harmony
inheritance thay đổi
```

Không rebuild mỗi hit.

---

# 45. Redis hay memory

Nếu profile được dùng liên tục:

```text
in-memory cache backend
```

sẽ nhanh hơn Redis.

Redis có thể dùng:

```text
persistence / multi-instance sync
```

Nếu backend hiện chỉ một instance thì:

```text
memory + DB source of truth
```

là đủ.

---

# 46. Data model đề xuất

```go
type PlayerPathProgress struct {
    PathID   string
    DaoMarks int64
    Realm    int
    Mastery  float64
}
```

---

# 47. Dao Path Config

```go
type DaoPathConfig struct {
    ID       string
    Group    string

    BaseConflictSensitivity float64
}
```

---

# 48. Compatibility override

```go
type DaoPairOverride struct {
    PathA string
    PathB string

    Compatibility float64
}
```

---

# 49. Trait

```go
type DaoTrait struct {
    NonConflicting bool

    ControllableConflict bool

    GlobalConflictReduction float64
}
```

---

# 50. Skill config

```go
type KillerMoveConfig struct {
    ID string

    Paths []string

    BasePower float64

    BaseSuccessRate float64

    ConflictSensitivity float64

    Complexity float64

    CanFail bool

    CanBacklash bool
}
```

---

# 51. Full Offensive Formula

Một phiên bản đủ mạnh nhưng vẫn quản lý được:

```text
FinalEffect
=
BaseEffect
× DaoAmplification
× RealmEfficiency
× MasteryEfficiency
× ConflictMultiplier
× EnvironmentMultiplier
× TargetResistanceMultiplier
```

---

# 52. ConflictMultiplier

```text
ConflictMultiplier
=
1
/
(
    1
    +
    ConflictRatio
    × ConflictSensitivity
    × ConflictConstant
)
```

---

# 53. Defensive Formula

```text
IncomingEffect
=
RawIncomingEffect
× NaturalDaoMultiplier
× DefenseMultiplier
× ShieldMultiplier
× SpecificResistanceMultiplier
```

Trong đó:

```text
NaturalDaoMultiplier
=
1 - NaturalDaoResistance
```

---

# 54. Với Non-Conflicting Dao Marks

```text
Offense:
ConflictMultiplier = 1.0

Defense:
NaturalDaoResistance = 0
```

---

# 55. Success Formula

```text
SuccessRate
=
BaseSuccess
+ RealmSuccessBonus
+ MasterySuccessBonus
+ InheritanceBonus
- ConflictFailurePenalty
- ComplexityPenalty
```

Clamp:

```text
MinSuccess ≤ SuccessRate ≤ MaxSuccess
```

Ví dụ:

```text
MinSuccess = 5%
MaxSuccess = 100%
```

---

# 56. Combat skill thường

Khuyến nghị:

```text
CanFail = false
```

Xung đột chỉ ảnh hưởng:

```text
damage
effect
duration
cost
cooldown
```

---

# 57. Killer Move cao cấp

Có thể:

```text
CanFail = true
CanBacklash = true
```

---

# 58. Mana Cost

Xung đột cũng có thể khiến:

```text
mana cost tăng
```

Ví dụ:

```text
ManaCostMultiplier
=
1 + ConflictRatio × ManaConflictScale
```

Cap khoảng:

```text
+50%
```

Không nên tăng vô hạn.

---

# 59. Cooldown

Sát chiêu bất ổn có thể:

```text
CooldownMultiplier
=
1 + Instability × 0.005
```

Ví dụ:

```text
Instability = 40
→ +20% cooldown
```

---

# 60. Phản phệ khi xung đột quá cao

Ví dụ threshold:

```text
Instability < 40:
safe

40–60:
minor failure chance

60–80:
failure + mana loss

80+:
backlash chance
```

---

# 61. Build chuyên tu

Ví dụ:

```text
Fire 100.000
Water 2.000
Other 3.000
```

Ưu điểm:

```text
damage Fire cao
conflict thấp
ổn định
natural defense tốt
```

Nhược:

```text
ít lựa chọn chiến thuật
dễ bị khắc chế
```

---

# 62. Build song tu

Ví dụ:

```text
Fire 60.000
Water 50.000
```

Nếu không có truyền thừa phù hợp:

```text
conflict cao
```

Nếu có:

```text
Fire-Water Harmony
Great Grandmaster cả hai
```

thì:

```text
rất mạnh
đa dụng
compound killer move tốt
```

---

# 63. Build đa tu

Ví dụ:

```text
10 path × 20.000 marks
```

Người thường:

```text
conflict cao
skill đơn hệ bị giảm
compound khó ổn định
```

Người có:

```text
NON_CONFLICTING_DAO_MARKS
```

thì:

```text
gần như không có offensive conflict
```

nhưng mất natural defense.

---

# 64. Vai trò của Rank

Có thể scale conflict theo Rank.

Ví dụ:

```text
Rank 1–5:
xung đột gần như không đáng kể

Rank 6:
bắt đầu rõ

Rank 7:
quan trọng

Rank 8:
rất quan trọng

Rank 9:
cốt lõi combat
```

Điều này phù hợp progression.

---

# 65. Rank Conflict Weight

Ví dụ:

| Rank | Conflict Weight |
|---|---:|
| 1 | 0.00 |
| 2 | 0.00 |
| 3 | 0.02 |
| 4 | 0.05 |
| 5 | 0.10 |
| 6 | 0.50 |
| 7 | 0.75 |
| 8 | 0.90 |
| 9 | 1.00 |

Nếu muốn sát lore hơn:

```text
chỉ Rank 6+ áp full mechanic
```

---

# 66. Cách tránh người chơi bị “kẹt build”

Không nên khiến player vô tình lấy vài Đạo Ngân khác hệ rồi build hỏng luôn.

Nên có:

```text
conflict threshold
```

Ví dụ:

```text
< 5% tổng Dao Marks
→ bỏ qua conflict
```

Hoặc:

```text
minor dao marks below 1000
→ không tính
```

---

# 67. Dominant Path

Có thể xác định:

```text
DominantPath
=
path có nhiều Dao Marks nhất
```

Nếu:

```text
dominant ≥ 70% tổng marks
```

cho bonus chuyên tu.

Ví dụ:

```text
Conflict Reduction +10%
Realm Efficiency +5%
```

---

# 68. Path Purity

```text
PathPurity
=
MainPathMarks
/
TotalDaoMarks
```

Ví dụ:

```text
Fire = 80k
Total = 100k
Purity = 80%
```

Có thể dùng cho:

```text
stability
natural defense
special achievements
inheritance
```

---

# 69. Không nên dùng Path Purity làm penalty trực tiếp

Không nên:

```text
20% foreign marks
→ -20% damage
```

Vì không xét compatibility.

Nên dùng:

```text
foreign marks
× compatibility
```

---

# 70. UI nên hiển thị gì

Không nên show toàn bộ công thức.

Player nên thấy:

```text
Độ Tinh Thuần Hỏa Đạo: 72%
Xung Đột Đạo Ngân: Trung Bình
Ổn Định Sát Chiêu Hỏa Đạo: 91%
Kháng Tự Nhiên Kiếm Đạo: 12%
```

---

# 71. Tooltip chi tiết

Ví dụ:

```text
Xung Đột Đạo Ngân: 8.4%

Nguồn chính:
- Thủy Đạo: 5.2%
- Kim Đạo: 1.9%
- Hồn Đạo: 1.3%

Cảnh giới Hỏa Đạo đã giảm:
- 43% xung đột
```

Rất dễ hiểu hơn việc show 50 dòng.

---

# 72. API response

Ví dụ:

```json
{
  "path": "FIRE",
  "dao_marks": 20000,
  "realm": "GREAT_GRANDMASTER",
  "purity": 0.63,
  "conflict_ratio": 0.084,
  "effectiveness": 0.923,
  "natural_resistance": {
    "WATER": 0.14,
    "SWORD": 0.08
  }
}
```

---

# 73. Logging / Debug

Backend nên có debug breakdown:

```json
{
  "base_damage": 1000,
  "dao_amp": 3.0,
  "realm_eff": 1.2,
  "mastery_eff": 1.05,
  "conflict_mult": 0.92,
  "environment_mult": 1.1,
  "target_dao_resist": 0.88,
  "final_damage": 3365
}
```

Cực kỳ hữu ích khi balance.

---

# 74. PvP

Trong PvP cần cap các multiplier.

Ví dụ:

```text
Max Dao Amplification: tùy rank
Max Natural Resistance: 30%
Max Conflict Damage Penalty: 40%
Max Mana Penalty: 50%
```

Không nên để build sai trở thành không chơi được.

---

# 75. PvE

PvE có thể cho boss:

```text
natural dao resistance cao
```

để buộc người chơi đổi path / kỹ năng.

Ví dụ:

```text
Fire Beast
→ resistance cao với Fire
→ Water / Ice có lợi hơn
```

Nhưng vẫn phải tách:

```text
element resistance
```

khỏi:

```text
dao conflict resistance
```

nếu muốn hệ thống rõ ràng.

---

# 76. Không nên double-dip quá mạnh

Cẩn thận trường hợp:

```text
attacker bị conflict -30%
target natural resist -30%
defense -50%
```

Damage sẽ tụt quá mạnh.

Nên đặt global floor:

```text
FinalEffectMultiplier >= 0.20
```

trừ các mechanic đặc biệt.

---

# 77. Suggested v1

Để implement nhanh, bản đầu chỉ cần 4 ảnh hưởng:

```text
1. Damage / Effect đầu ra
2. Natural Resistance
3. Mana Cost
4. Stability
```

Chưa cần success/failure ngay.

---

# 78. Suggested v2

Sau khi balance ổn:

```text
5. Success rate
6. Backlash
7. Healing conflict
8. Residual Dao Marks
9. Environment Dao
```

---

# 79. Suggested v3

Endgame:

```text
10. Compound killer move
11. Dao harmony inheritance
12. Controllable Dao Conflict
13. Dao territory / Dao field
14. Dao vein
```

---

# 80. Recommended Core Formula Set

## Offensive

```text
RawConflict
=
Σ ForeignMarks × (1 - Compatibility)
```

```text
EffectiveConflict
=
RawConflict
× (1 - RealmControl)
× (1 - HarmonyReduction)
```

```text
ConflictRatio
=
EffectiveConflict
/
(MainMarks + ConflictScale)
```

```text
ConflictMultiplier
=
1
/
(
1 + ConflictRatio × SkillSensitivity × K
)
```

```text
FinalEffect
=
BaseEffect
× DaoAmplification
× RealmEfficiency
× MasteryEfficiency
× ConflictMultiplier
```

---

## Defensive

```text
DefenseConflict
=
Σ DefenderMarks × (1 - Compatibility(AttackPath, DefenderPath))
```

```text
NaturalResistance
=
MaxResistance
× DefenseConflict
/
(DefenseConflict + ResistanceScale)
```

Nếu:

```text
NON_CONFLICTING_DAO_MARKS
```

thì:

```text
NaturalResistance = 0
```

---

## Success

```text
SuccessRate
=
BaseSuccess
+ RealmBonus
+ MasteryBonus
- ConflictPenalty
- ComplexityPenalty
```

---

# 81. Giá trị khởi đầu đề xuất

```text
ConflictScale = 50.000
Conflict K = 2.0

Natural Resistance Cap = 30%
ResistanceScale = 50.000

Max Conflict Output Penalty = 40%
Max Mana Penalty = 50%

Minor foreign marks threshold:
max(1000, TotalDaoMarks × 0.02)
```

Đây là giá trị để test, không phải final balance.

---

# 82. Ví dụ hoàn chỉnh

Player A:

```text
Rank 7

Fire       = 50.000
Water      = 20.000
Wisdom     = 10.000

Fire Realm = Great Grandmaster
Fire Mastery = 80
```

Fire ↔ Water:

```text
0.55
```

Fire ↔ Wisdom:

```text
0.90
```

Raw conflict:

```text
20.000 × 0.45
+
10.000 × 0.10

= 10.000
```

Realm control:

```text
45%
```

Effective:

```text
10.000 × 0.55
=
5.500
```

Conflict ratio:

```text
5500 / (50000 + 50000)
=
0.055
```

Skill sensitivity:

```text
0.8
```

K:

```text
2
```

Conflict multiplier:

```text
1 / (1 + 0.055 × 0.8 × 2)
≈ 0.919
```

Skill mất khoảng:

```text
8.1% hiệu quả
```

Không quá nặng nhưng đủ để cảm nhận.

---

# 83. Ví dụ Non-Conflicting Dao Marks

Cùng Player A:

```text
NON_CONFLICTING_DAO_MARKS = true
```

Khi đánh:

```text
ConflictMultiplier = 1
```

Skill mạnh hơn khoảng:

```text
8.8%
```

so với phiên bản thường trong ví dụ trên.

Nhưng khi bị đánh:

```text
NaturalDaoResistance = 0
```

trong khi Cổ Tiên thường có thể đang có:

```text
10–25%
```

natural resistance.

Đây là trade-off rõ ràng.

---

# 84. Mục tiêu balance cuối

Hệ thống tốt khi:

### Chuyên tu

```text
ổn định nhất
damage cùng hệ cao
phòng thủ tự nhiên tốt
```

### Song tu

```text
đa dụng
có thể rất mạnh
cần realm + truyền thừa
```

### Đa tu

```text
rất linh hoạt
nhưng khó điều hòa
```

### Non-Conflicting build

```text
cực mạnh về flexibility/offense
yếu hơn về natural defense
```

### Cảnh giới cao

```text
không tạo sức mạnh từ hư không
nhưng tận dụng Đạo Ngân hiệu quả hơn
```

---

# 85. Nguyên tắc quan trọng nhất

Không nên biến xung đột Đạo Ngân thành:

```text
“có nhiều path = bị phạt”
```

Mà nên biến nó thành:

```text
“có nhiều path = cần đủ hiểu biết để điều hòa”
```

Đó mới là điểm thú vị.

Một người đa tu không nên yếu vì game cấm đa tu.

Họ yếu nếu:

```text
Dao Marks hỗn loạn
Realm thấp
không có truyền thừa
không có killer move phù hợp
```

Nhưng nếu có:

```text
Realm cao
Mastery cao
Compound killer move
Harmony inheritance
Special physique
```

thì đa tu có thể trở thành một con đường build cực mạnh.

---

# 86. Kết luận

Cơ chế Xung Đột Đạo Ngân nên là một **hệ thống nền**, tác động xuyên suốt combat chứ không phải một modifier nhỏ.

Nó nên quyết định:

```text
Ta có bao nhiêu Đạo Ngân?
Ta hiểu lưu phái này đến đâu?
Ta có kiểm soát được các Đạo khác không?
Sát chiêu của ta có ổn định không?
Đối phương có Đạo Ngân nào?
Môi trường có thuận lợi không?
Ta đang chuyên tu, song tu hay đa tu?
```

Nếu triển khai đúng, hệ thống này sẽ tự tạo ra rất nhiều archetype build khác nhau mà không cần hard-code từng class.

---

# 87. Hướng triển khai khuyến nghị cho GuWorld

Thứ tự nên làm:

```text
Phase 1
├─ Dao Path Group
├─ Compatibility
├─ Pair Override
├─ Realm Control
├─ Offensive Conflict
└─ Natural Dao Resistance

Phase 2
├─ Skill Conflict Sensitivity
├─ Mana Penalty
├─ Stability
└─ UI breakdown

Phase 3
├─ Success rate
├─ Backlash
├─ Compound killer move
├─ Healing conflict
└─ Residual Dao Marks

Phase 4
├─ Environment Dao
├─ Dao field
├─ Dao vein
├─ Non-Conflicting Dao Marks
└─ Controllable Dao Conflict
```

Đây là hướng đủ linh hoạt để scale lên 50+ lưu phái mà vẫn giữ được khả năng balance và tối ưu hiệu năng.
