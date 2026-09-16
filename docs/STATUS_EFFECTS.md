# GuWorld - Hệ Thống Effect & Status Effects (Reference & Developer Guide)

Tài liệu chi tiết về kiến trúc, luồng hoạt động, danh mục status effect, và hướng dẫn lập trình viên **thêm Cổ trùng / Effect mới** vào hệ thống GuWorld.

---

## 1. Tổng Quan Kiến Trúc (Architecture Overview)

Hệ thống Effect trong GuWorld được thiết kế theo mô hình **Data-Driven**, phân tầng từ khai báo dữ liệu JSON đến tính toán chỉ số và mô phỏng chiến đấu:

```
[CSV Data (backend/data/*.csv)]
       │
       ▼ (Admin Import / DB Seed)
[Database (gu_dictionary, constitution_dictionary, race_dictionary)]
       │
       ▼ (Server Startup / Memory Registry)
[In-Memory Registries (models.GuRegistry, gamefx.*)]
       │
       ├───────────────────────────────┼───────────────────────────────┐
       ▼                               ▼                               ▼
[Stat Aggregation]            [Combat Engine]               [World Capabilities]
 (gamefx.Get(player))      (simulator/engine.go)      (models.InventoryHasGuMechanic)
```

### Các Struct Cốt Lõi

1. **`models.GuEffects`** ([backend/internal/models/gu.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/models/gu.go#L106)):
   ```go
   type GuEffects struct {
       Active  *ActiveEffect     `json:"active"`            // Effect kích hoạt trong combat
       Passive *PassiveEffect    `json:"passive"`           // Chỉ số bị động cộng thẳng
       Growth  *GrowthEffect     `json:"growth,omitempty"`  // Tốc độ nuôi dưỡng/tăng trưởng
       Other   []SpecialMechanic `json:"other"`             // Cơ chế đặc thù/capability ngoài combat
   }
   ```

2. **`gamefx.Effects`** ([backend/internal/gamefx/fx.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/gamefx/fx.go#L14)):
   Cấu trúc tổng hợp tất cả modifier của nhân vật từ Chủng tộc (Race), Thể chất (Constitution), Cổ bị động (Passive Gu), và Trạng thái (Statuses).

3. **`effectcatalog`** ([backend/internal/effectcatalog/manifest.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/effectcatalog/manifest.go)):
   Bộ kiểm soát và xác thực (Validator & Manifest) đảm bảo mọi key JSON trong `Gu.csv` đều hợp lệ, có owner và disposition rõ ràng.

4. **`simulator.Effect`** ([backend/internal/simulator/engine.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/simulator/engine.go)):
   Cấu trúc đại diện cho trạng thái/buff/debuff đang duy trì theo lượt trong combat deterministic engine.

---

## 2. Luồng Hoạt Động Liên Quan (Runtime & Processing Flow)

### A. Đọc Dữ Liệu & Registry Cache
- Khi khởi động server, backend nạp dữ liệu từ DB vào bộ nhớ (`models.GuRegistry`, `gamefx.ConstitutionRegistry`, `gamefx.RaceRegistry`).
- Khi sửa `backend/data/Gu.csv`, admin thực hiện **Import Cổ trùng** qua Admin Panel và **restart server** để registry cập nhật.

### B. Cộng Gộp Chỉ Số Bị Động (Stat Aggregation)
Hàm `gamefx.Get(p *models.Player)` thực hiện tính toán cộng gộp:
1. Cộng chỉ số từ Chủng tộc (`p.Race`).
2. Cộng chỉ số từ các Trạng thái active (`p.Statuses`).
3. Cộng chỉ số từ Thể chất (`p.Constitution`).
4. Các thuộc tính Cổ bị động (`stat_base_attack`, `stat_base_hp`, `stat_base_speed`...) được tính cộng gộp trong `EffectiveMaxHP()`, `EffectiveBaseAttack()`, v.v.

### C. Xử Lý Chiêu Thức & Combat Engine
1. Khi dùng Cổ active trong combat, [handlers/combat_effects.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/handlers/combat_effects.go) tính toán sát thương/khiên dựa trên Cảnh giới, Điểm Đạo Ngân (`DaoMultiplier`), và Cảnh giới áp chế (`RankFactor`).
2. Trong [simulator/engine.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/simulator/engine.go), trạng thái bất lợi (BLEED, POISON, STUN, FREEZE, SILENCE...) được đưa vào danh sách `ActiveEffects` của combatant và được tick/trừ thời gian mỗi lượt.

### D. Cơ Chế Ngoài Combat (World Capabilities)
Các tính năng ngoài combat (như ngụy trang `DISGUISE`, chợ `MARKETPLACE_ACCESS`, giao dịch từ xa `REMOTE_TRADE`...) sử dụng mảng `other` trong JSON.
Hệ thống sử dụng helper `models.InventoryHasGuMechanic(playerGuInv, "ACTION_CODE", requireEquipped)` để kiểm tra quyền hạn của người chơi.

---

## 3. Hướng Dẫn Chi Tiết: Thêm Effect & Cổ Mới (Developer Guide)

### Trường Hợp 1: Thêm Cổ Mới Sử Dụng Effect ĐÃ CÓ SẴN (Không cần sửa Go Code)

Nếu hiệu ứng bạn muốn dùng đã có trong hệ thống (như `damage_flat`, `shield_flat`, `stat_base_attack`, `apply_status: "STEALTH"`...):

1. Mở file [backend/data/Gu.csv](file:///home/vippro67/Documents/GitHub/GuWorld/backend/data/Gu.csv).
2. Thêm dòng Cổ mới với cấu trúc JSON trong cột `effects`:
   ```csv
   GU_LIGHTNING_BOLT,Lôi Đình Cổ,3,Active,6,"[""lightning_stone""]",300,"Bắn ra tia sét gây sát thương và có cơ hội làm choáng đối thủ.","[""Lightning Path""]",10.0,"{""active"": {""damage_flat"": 450, ""apply_status"": ""STUN"", ""status_duration_ticks"": 1}, ""passive"": {""stat_base_attack"": 15}, ""growth"": {}, ""other"": []}",,False,ATTACK_CORE,"[""ATTACK_CORE""]","[""lightning"",""damage""]","[""damage_boost""]",1200.0,
   ```
3. Đăng nhập Admin Panel -> Tab **Dữ liệu** -> Import bảng **Cổ Trùng**.
4. Restart server backend.

---

### Trường Hợp 2: Thêm Chỉ Số Passive Mới (Ví dụ: `stat_crit_damage_pct` - Tăng % sát thương bạo kích)

1. **Khai báo field trong struct `PassiveEffect`** tại [backend/internal/models/gu.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/models/gu.go#L69):
   ```go
   type PassiveEffect struct {
       // ... các field khác
       StatCritDamagePct float64 `json:"stat_crit_damage_pct,omitempty"`
   }
   ```

2. **Đăng ký key vào Catalog Manifest** tại [backend/internal/effectcatalog/manifest.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/effectcatalog/manifest.go#L58):
   ```go
   add("passive", DispositionSupported, "player-stats", 
       "stat_base_attack", "stat_base_defense", "stat_base_hp", "stat_base_speed", 
       "movement", "investigation", "hearing", "evasion", "luck", "accuracy", 
       "damage_multiplier_on_hit", "cleansing_power", "disguise_level", 
       "stat_crit_damage_pct") // <--- Thêm key mới vào đây
   ```

3. **Cập nhật logic tính toán chỉ số**:
   Trong hàm tính toán combat/stats (như `gamefx/fx.go` hoặc simulator), đọc `spec.Effects.Passive.StatCritDamagePct` và cộng dồn vào thuộc tính nhân vật.

4. **Thêm dữ liệu vào `Gu.csv`**, Import qua Admin Panel và restart backend.

---

### Trường Hợp 3: Thêm Hiệu Ứng Active Combat Mới (Ví dụ: `burn_mana_flat` - Đốt Chân nguyên đối thủ)

1. **Khai báo field trong struct `ActiveEffect`** tại [backend/internal/models/gu.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/models/gu.go#L16):
   ```go
   type ActiveEffect struct {
       // ... các field khác
       BurnManaFlat float64 `json:"burn_mana_flat,omitempty"`
   }
   ```

2. **Đăng ký key vào Catalog Manifest** tại [backend/internal/effectcatalog/manifest.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/effectcatalog/manifest.go#L47):
   ```go
   add("active", DispositionSupported, "combat",
       "damage_flat", "heal_flat", "shield_flat", ..., "burn_mana_flat")
   ```

3. **Xử lý logic chiêu thức trong Combat Engine**:
   Trong [simulator/engine.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/simulator/engine.go) hoặc [handlers/combat_effects.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/handlers/combat_effects.go):
   ```go
   if guSpec.Effects.Active != nil && guSpec.Effects.Active.BurnManaFlat > 0 {
       target.Mana = math.Max(0, target.Mana - guSpec.Effects.Active.BurnManaFlat)
   }
   ```

4. **Thêm dữ liệu vào `Gu.csv`**, Import qua Admin Panel và restart backend.

---

### Trường Hợp 4: Thêm Special Mechanic / Capability Mới (Ví dụ: `TELEPORT_HUB` - Dịch chuyển về Hub)

1. **Đăng ký Action Code & Trigger** tại [backend/internal/effectcatalog/manifest.go](file:///home/vippro67/Documents/GitHub/GuWorld/backend/internal/effectcatalog/manifest.go#L124):
   ```go
   add("CAPABILITY", DispositionSupported, "world", 
       "MARKETPLACE_ACCESS", "REMOTE_TRADE", "CHAT_CONNECT", ..., "TELEPORT_HUB")
   ```

2. **Xử lý logic kiểm tra trong Handler**:
   Trong handler di chuyển/hành động (ví dụ `player_explore.go`):
   ```go
   if models.InventoryHasGuMechanic(playerGuInv, "TELEPORT_HUB", true) {
       // Cho phép dịch chuyển tức thời về thành chính
   }
   ```

3. **Định nghĩa Cổ trùng trong `Gu.csv`**:
   ```csv
   GU_SPACE_TELEPORT,Định Hướng Cổ,4,Support,6,"[""space_crystal""]",400,"Cho phép dịch chuyển tức thời về Hub.","[""Space Path""]",20.0,"{""active"": {}, ""passive"": {}, ""growth"": {}, ""other"": [{""event_trigger"": ""CAPABILITY"", ""action_code"": ""TELEPORT_HUB"", ""params"": {}}]}",,False,SUPPORT_UTILITY,"[""SUPPORT_UTILITY""]","[""space""]","[""utility_boost""]",2500.0,
   ```

4. **Import qua Admin Panel & Restart backend**.

---

## 4. Danh Mục Hiệu Ứng Chiến Đấu & Status Effects

### A. Debuff Cơ Bản & Khống Chế

| Hiệu ứng | T1 | T2 | T3 | T4 | T5 | Cơ chế vận hành | Lưu Phái |
|---|---|---|---|---|---|---|---|
| **SLOW** (Chậm) | 15% / 3t | 25% / 3t | 35% / 4t | 45% / 4t | 55% / 5t | Giảm `EffectiveSpeed`, tụt thứ tự hành động trong lượt. | Thổ / Trụ Đạo |
| **WEAK** (Hư Nhược) | 10% / 3t | 20% / 3t | 30% / 4t | 40% / 4t | 50% / 5t | Giảm % sát thương gây ra của mục tiêu. | Âm / Khí Đạo |
| **FREEZE** (Đóng Băng) | 1 lượt | 1 lượt | 2 lượt | 2 lượt | 3 lượt | Mất lượt; 50% vỡ băng khi nhận sát thương; chịu Melt Damage nếu trúng Hỏa. | Băng Tuyết Đạo |
| **SILENCE** (Câm Lặng) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Khóa Cổ kỹ năng và Sát Chiêu, chỉ được dùng Đánh Thường. | Âm Thanh / Khí Đạo |
| **STUN** (Choáng Váng) | 1 lượt | 1 lượt | 2 lượt | 2 lượt | 3 lượt | Mất lượt hoàn toàn, không bị phá vỡ bởi sát thương. | Lực / Lôi Đạo |
| **BLEED** (Xuất Huyết) | DoT | DoT | DoT | DoT | DoT | Trừ HP đầu lượt; kích hoạt bẫy nổ máu khi bị giải trừ cưỡng chế. | Huyết Đạo |
| **POISON** (Trúng Độc) | DoT | DoT | DoT | DoT | DoT | Trừ HP đầu lượt; giảm 20% lượng hồi máu nhận vào. | Độc Đạo |
| **BURN** (Thiêu Đốt) | DoT | DoT | DoT | DoT | DoT | Trừ HP mỗi lượt; bị dập tắt ngay bởi kỹ năng thuộc tính Thủy/Băng. | Viêm Đạo |
| **BLIND** (Mù Mắt) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | 35% xác suất đòn đánh bị trượt (Miss hoàn toàn). | Quang / Ám Đạo |
| **SLEEP** (Ngủ Say) | 1 lượt | 2 lượt | 2 lượt | 3 lượt | 3 lượt | Mất lượt; lập tức tỉnh dậy khi nhận bất kỳ sát thương nào. | Mộng Đạo |
| **CONFUSE** (Mê Loạn) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | 40% xác suất xuất chiêu nhầm mục tiêu vào đồng minh. | Trí / Biến Hóa Đạo |
| **CHARM** (Mê Hoặc) | 2 lượt | 3 lượt | 3 lượt | 4 lượt | 5 lượt | Bị thao túng hoàn toàn, quay sang tấn công phe mình. | Mị / Tình Đạo |
| **BIND** (Trói Buộc) | 2 lượt | 3 lượt | 3 lượt | 4 lượt | 5 lượt | Khóa cử động vật lý, mất lượt hành động. | Mộc / Thổ Đạo |
| **FEAR** (Khiếp Sợ) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Chỉ có thể đánh thường, mất khả năng kích hoạt phản đòn. | Hồn / Ám Đạo |

---

### B. Hiệu Ứng Cổ Giới Đặc Thù (Cổ Chân Nhân Mechanics)

| Hiệu ứng | Tier / Thời hạn | Bản chất & Cơ chế chi tiết | Lưu Phái |
|---|---|---|---|
| **ESSENCE_LEAK** (Ô Khiếu / Rò Rỉ Chân Nguyên) | 5%–25% / 3 lượt | Mỗi đầu lượt mất $X\%$ Chân Nguyên tối đa trong không khiếu. Khi Chân Nguyên = 0, Cổ Sư rơi vào trạng thái kiệt quệ, chỉ có thể đánh thường. | Thổ / Khí Đạo |
| **GU_SEAL** (Phong Cổ / Trầm Miên) | 1 Cổ / 2–3 lượt | Phong ấn ngẫu nhiên 1 Cổ trùng đang trang bị. Cổ bị phong ấn không thể kích hoạt chủ động và vô hiệu hóa toàn bộ thuộc tính cộng thêm (Passive). | Luyện / Trụ Đạo |
| **SOUL_WOUND** (Hồn Thương) | 2–4 lượt | Đòn đánh xâm thực hồn phách: Gây sát thương chuẩn (True Damage) bỏ qua $100\%$ Hộ giáp/Khiên; tăng 20% Chân Nguyên tiêu hao mỗi kỹ năng. | Hồn Đạo |
| **THOUGHT_CHAOS** (Tạp Niệm / Loạn Ý) | 2–3 lượt | Chấn động niệm đầu: Mỗi khi kích hoạt Sát Chiêu (Killer Move), có $35\%$ xác suất bị đứt đoạn liên kết niệm đầu, dẫn tới thất bại, mất lượt và tiêu hao Chân Nguyên vô ích. | Trí Đạo |
| **DAO_CONFLICT** (Xung Đột Đạo Ngân) | 3–5 lượt | Đạo ngân dị chủng bài xích: Giảm $50\%$ hiệu lực hồi phục/khiên từ đồng đội; tăng $25\%$ sát thương nhận vào từ các đòn đánh khắc hệ. | Biến Hóa Đạo |
| **BACKLASH** (Phản Phệ) | Tức thì / 1 lượt | Bản mệnh chấn thương: Xảy ra khi Sát Chiêu bị cắt đứt hoặc Khiên bản mệnh vỡ nát, lập tức nhận sát thương phản chấn bằng $150\%$ uy lực chiêu thức. | Quy Tắc Đạo |
| **REVIVE_LOCK** (Khóa Hồi Sinh) | 3 lượt | Phong tỏa dòng chảy sinh mệnh: Ngăn chặn hoàn toàn việc phục sinh, vô hiệu hóa Xuân Thu Thiền hoặc các Cổ mệnh bài tái sinh. | Trụ / Vũ Đạo |

---

### C. Buff Hộ Thể & Trợ Lực Bản Thân

| Hiệu ứng | T1 | T2 | T3 | T4 | T5 | Cơ chế vận hành |
|---|---|---|---|---|---|---|
| **THORNS** (Khiên Gai) | 10% / 3t | 15% / 3t | 20% / 4t | 25% / 4t | 30% / 5t | Phản % sát thương trực tiếp lại cho kẻ tấn công (bị giảm trừ bởi Def kẻ địch). |
| **REGEN** (Tự Dưỡng) | 3% / 3t | 5% / 4t | 8% / 4t | 12% / 5t | 16% / 5t | Hồi phục HP theo % Max HP vào đầu mỗi lượt hành động. |
| **ATK_UP** (Cường Công) | 10% / 3t | 15% / 3t | 20% / 4t | 25% / 4t | 30% / 5t | Tăng % hệ số sát thương gây ra (`DamageMultiplier`). |
| **DEF_UP** (Thiết Vách) | 10% / 3t | 15% / 3t | 20% / 4t | 25% / 4t | 30% / 5t | Tăng % chỉ số phòng ngự thực tế (`DefenseMultiplier`). |
| **TAUNT** (Khiêu Khích) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Cưỡng chế mục tiêu đơn của đối phương phải tấn công người thi triển. |
| **STEALTH** (Ẩn Thân) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Không thể bị chọn làm mục tiêu của kỹ năng đơn mục tiêu (vẫn chịu sát thương AoE). |
| **HASTE** (Gia Tốc) | 15% / 2t | 25% / 3t | 35% / 3t | 45% / 4t | 55% / 4t | Tăng % tốc độ hành động (`EffectiveSpeed`), đẩy sớm lượt đánh. |
| **INVINCIBLE** (Vô Địch) | 1 lượt | 1 lượt | 1 lượt | 1 lượt | 2 lượt | Miễn nhiễm $100\%$ mọi nguồn sát thương và kháng toàn bộ hiệu ứng mới. |

---

### D. Tỉ Lệ Áp Dụng (Proc Rate & Cảnh Giới Áp Chế)

Xác suất dính hiệu ứng bất lợi phụ thuộc vào Tỉ lệ gốc của Cổ, Chênh lệch Cảnh giới (Chuyển), và Điểm Đạo Ngân:

$$P_{\text{apply}} = P_{\text{base}} \times \left(1 + \frac{\text{DaoAttacker} - \text{DaoDefender}}{100}\right) \times \text{RankFactor} \times (1 - \text{Resist})$$

**Bảng Hệ Số Cảnh Giới (`RankFactor`):**

| Tương Quan Cảnh Giới | `RankFactor` | Ý Nghĩa Thực Chiến |
|---|---|---|
| Caster cao hơn $\ge 2$ Chuyển | **1.5** | Áp chế cảnh giới tuyệt đối; tỉ lệ dính gần như chắc chắn. |
| Caster cao hơn $1$ Chuyển | **1.2** | Ưu thế lớn về áp lực Chân Nguyên. |
| Cùng Cảnh Giới (Đồng cấp) | **1.0** | Tỉ lệ tiêu chuẩn theo phẩm chất Cổ. |
| Caster thấp hơn $1$ Chuyển | **0.6** | Bị áp chế ngược, khó xuyên thấu phòng ngự. |
| Caster thấp hơn $\ge 2$ Chuyển | **0.1** | Miễn nhiễm gần như hoàn toàn (Phàm Cổ không trói được Tiên Nhân). |

*Quy tắc Tiên Cổ (Rank 6+):* Khi sử dụng Tiên Cổ hoặc Tiên Đạo Sát Chiêu, `RankFactor` tối thiểu luôn là **1.0** bất kể mục tiêu có cảnh giới cao hơn, và tự động nâng lên **1.5** khi đánh vào Phàm Cổ Sư.