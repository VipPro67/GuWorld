# Status Effects — Hệ Thống Hiệu Ứng Chiến Đấu (Cổ Giới Webgame)

## 1. Tổng Quan & Triết Lý Thiết Kế

Hệ thống hiệu ứng chiến đấu được thiết kế chuẩn hóa theo thế giới quan nguyên tác **Cổ Chân Nhân (Reverend Insanity)**, kết hợp cùng cơ chế chiến đấu theo lượt (Turn-based Combat Engine) của Webgame backend Go.

Hệ thống xoay quanh 5 nguyên lý cốt lõi:
1. **Lưu Phái Độc Đạo:** Mỗi hiệu ứng gắn liền với bản chất của các lưu phái (Kiếm, Lực, Hồn, Trí, Huyết, Viêm, Băng Tuyết, Luyện, Trụ, Vũ, Biến Hóa, Nô, Độc...).
2. **Tiên Phàm Chi Cách & Cảnh Giới Áp Chế:** Chênh lệch Chuyển (Rank 1–5 Phàm Cảnh, Rank 6–8 Tiên Cảnh) tạo ra ưu thế áp chế tuyệt đối về xác suất áp dụng và thời lượng hiệu lực.
3. **Đạo Ngân Tương Khắc & Tương Trợ:** Điểm Đạo Ngân của Cổ Sư quyết định khả năng xuyên kháng hiệu ứng hoặc khuếch đại uy lực chiêu thức.
4. **Không Khiếu & Chân Nguyên / Niệm Đầu:** Các đòn hiểm tập trung đánh phá tài nguyên gốc rễ của Cổ Sư (làm cạn chân nguyên, chấn động niệm đầu, ô nhiễm không khiếu).
5. **Vạn Vật Tương Sinh Tương Khắc:** *"Không có Cổ trùng mạnh nhất, chỉ có Cổ Sư mạnh nhất"* — mọi trạng thái bất lợi đều có Cổ đối ứng để hóa giải, nhưng việc giải trừ vội vã có thể kích hoạt bẫy phản phệ tàn độc.

Mỗi hiệu ứng được chuẩn hóa theo **Tier 1–5**, quy định thời lượng duy trì và cường độ qua `TierToPercent()` và `TierToDuration()` trong `simulator/models.go`.

---

## 2. Danh Mục Hiệu Ứng Chiến Đấu

### A. Debuff Cơ Bản & Khống Chế (Phàm Cấp & Tâm Trí)

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

Các hiệu ứng này can thiệp trực tiếp vào cấu trúc Chân Nguyên, Không Khiếu, Niệm Đầu và Đạo Ngân:

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

## 3. Tỉ Lệ Áp Dụng (Proc Rate & Cảnh Giới Áp Chế)

Trong thế giới Cổ Chân Nhân, không có tỉ lệ dính hiệu ứng bất biến. Khả năng gieo rắc hiệu ứng phụ thuộc vào ba yếu tố: **Tỉ lệ gốc của Cổ**, **Chênh lệch Cảnh giới (Chuyển)**, và **Chênh lệch Đạo Ngân**.

### Công Thức Tính Xác Suất Thực Tế

$$P_{\text{apply}} = P_{\text{base}} \times \left(1 + \frac{\text{DaoAttacker} - \text{DaoDefender}}{100}\right) \times \text{RankFactor} \times (1 - \text{Resist})$$

Trong đó:
*   $P_{\text{base}}$: Tỉ lệ cơ sở quy định trong Cổ/Sát Chiêu (T1: 30%, T2: 45%, T3: 60%, T4: 75%, T5: 90%).
*   $\text{DaoAttacker}, \text{DaoDefender}$: Điểm số Đạo Ngân tương ứng với Lưu phái của chiêu thức.
*   $\text{RankFactor}$: Hệ số chênh lệch Chuyển giữa Cổ Sư thi triển và mục tiêu.
*   $\text{Resist}$: Kháng tính trạng thái của mục tiêu (từ nội tại hoặc Cổ phòng ngự).

### Bảng Hệ Số Cảnh Giới (`RankFactor`)

| Tương Quan Cảnh Giới | `RankFactor` | Ý Nghĩa Thực Chiến |
|---|---|---|
| Caster cao hơn $\ge 2$ Chuyển | **1.5** | Áp chế cảnh giới tuyệt đối; tỉ lệ dính gần như chắc chắn. |
| Caster cao hơn $1$ Chuyển | **1.2** | Ưu thế lớn về áp lực Chân Nguyên. |
| Cùng Cảnh Giới (Đồng cấp) | **1.0** | Tỉ lệ tiêu chuẩn theo phẩm chất Cổ. |
| Caster thấp hơn $1$ Chuyển | **0.6** | Bị áp chế ngược, khó xuyên thấu phòng ngự. |
| Caster thấp hơn $\ge 2$ Chuyển | **0.1** | Miễn nhiễm gần như hoàn toàn (Phàm Cổ không trói được Tiên Nhân). |

*Quy tắc Tiên Cổ (Rank 6+):* Khi sử dụng **Tiên Cổ** hoặc **Tiên Đạo Sát Chiêu**, `RankFactor` tối thiểu luôn là **1.0** bất kể mục tiêu có cảnh giới cao hơn, và tự động nâng lên **1.5** khi đánh vào Phàm Cổ Sư.

---

## 4. Cơ Chế Giải Trừ & Khắc Chế (Cleanse Mechanics)

### A. Phân Loại Trạng Thái (Status Categories)

```
CategoryPhysicalDoT : BLEED, POISON, BURN
CategoryMentalCC    : STUN, SILENCE, FEAR, CONFUSE, SLEEP, CHARM, BLIND, BIND
CategoryAperture    : ESSENCE_LEAK, GU_SEAL, THOUGHT_CHAOS
CategoryDaoConflict : DAO_CONFLICT, SOUL_WOUND
CategoryBuff        : THORNS, REGEN, ATK_UP, DEF_UP, TAUNT, STEALTH, HASTE, INVINCIBLE
CategoryImmortal    : REVIVE_LOCK, BACKLASH (Tuyệt đối không thể hóa giải bằng Phàm chiêu)
```

### B. Phương Thức Hóa Giải

1. **Cổ Trùng Hóa Giải Chuyên Dụng:**
   *   **Thanh Tâm Cổ / Ngã Ý Cổ (Trí Đạo):** Tẩy trừ các trạng thái `CategoryMentalCC`. Riêng *Ngã Ý Cổ* có $35\%$ xác suất tự giải phóng khống chế vào đầu mỗi lượt.
   *   **Cam Lâm Cổ / Tịnh Hóa Cổ (Thủy/Quang Đạo):** Hóa giải toàn bộ DoT thuộc `CategoryPhysicalDoT` và phục hồi tổn thương không khiếu `CategoryAperture`.
   *   **Phá Chướng Cổ / Đoạt Mệnh Cổ (Quy Tắc Đạo):** Trừ khử toàn bộ Buff có lợi trên thân đối phương (`CategoryBuff`).

2. **Lưu Phái Tương Khắc Tự Nhiên (Path Cancellation):**
   *   **Hỏa Phá Băng:** Mục tiêu đang bị `FREEZE` nếu trúng đòn tấn công hệ Viêm/Hỏa sẽ tan băng ngay lập tức, nhưng phải gánh thêm $30\%$ sát thương bạo kích tan băng (**Melt Damage**).
   *   **Thủy Dập Hỏa:** Các kỹ năng nước hoặc băng khi thi triển lên đồng minh sẽ dập tắt ngay trạng thái `BURN`.

3. **Cơ Chế Bẫy Phản Phệ Khi Giải Trừ (Cleanse Trap):**
   *   **Khí Huyết Bạo Phát (Huyết Đạo):** Nếu dùng Cổ tịnh hóa thông thường để xóa debuff `BLEED` do sát chiêu Huyết Đạo gieo rắc, huyết mạch lập tức nổ tung: Mục tiêu chịu ngay $50\%$ tổng sát thương DoT còn lại gom lại thành một đòn tức thời.

---

## 5. Luồng Xử Lý Combat Engine Mỗi Lượt (Turn Flow)

```
1. Tick DoTs & Resource Drain:
   - Trừ máu DoT: BLEED, POISON, BURN
   - Trừ máu chuẩn: SOUL_WOUND (bỏ qua Shield/Armor)
   - Trừ chân nguyên: ESSENCE_LEAK (hạ Chân Nguyên trong không khiếu)
   - Hồi máu: REGEN
2. Giảm thời hạn hiệu lực (Duration Tick):
   - Giảm Turns của tất cả buff/debuff; xóa hiệu ứng hết hạn
3. Tính toán Derived Stats & Multipliers:
   - EffectiveSpeed = BaseSpeed * (1 - Slow% + Haste%)
   - DamageMultiplier = (1 - Weak%) * (1 + AtkUp%)
   - DefenseMultiplier = 1 + DefUp% - ArmorBreak%
   - Flags: IsSilenced, IsFrozen, HasTaunt, IsStealthed, IsBound, IsSleeping
4. Sắp xếp thứ tự hành động toàn cục theo EffectiveSpeed
5. Với mỗi Actor đến lượt:
   a. Kiểm tra mất lượt: STUN, FREEZE, SLEEP, BIND -> Skip lượt hoàn toàn
   b. Kiểm tra ràng buộc hành động:
      - SILENCE / FEAR / Chân Nguyên = 0 -> Ép buộc dùng BASIC_ATTACK
      - THOUGHT_CHAOS -> 35% thất bại đứt chiêu nếu dùng KillerMove
   c. Kiểm tra chọn mục tiêu:
      - CHARM -> Quay đầu tấn công đồng minh
      - CONFUSE -> 40% nhắm sai mục tiêu sang phe mình
      - TAUNT -> Chuyển hướng mục tiêu đơn sang Taunter
      - STEALTH -> Bỏ qua mục tiêu ẩn thân nếu dùng chiêu đơn mục tiêu
   d. Thực thi Action:
      - BLIND check: 35% đánh trượt (Miss)
      - Áp dụng weak/atk_up/def_up multipliers
      - Sát thương trúng đích: SLEEP thức dậy (100%), FREEZE vỡ băng (50%)
      - Xử lý phản phệ THORNS
      - CanApplyStatus() kiểm tra Đạo Ngân & Cảnh giới -> Gán debuff/buff mới
```

---

## 6. Cấu Trúc Mã Nguồn Backend Go Đề Xuất

### `internal/statusfx/status.go`

```go
package statusfx

const (
	// Legacy & Elemental
	StatusSlow       = "SLOW"
	StatusWeak       = "WEAK"
	StatusFreeze     = "FREEZE"
	StatusSilence    = "SILENCE"
	StatusStun       = "STUN"
	StatusBleed      = "BLEED"
	StatusPoison     = "POISON"
	StatusBurn       = "BURN"
	StatusBlind      = "BLIND"
	StatusSleep      = "SLEEP"
	StatusConfuse    = "CONFUSE"
	StatusCharm      = "CHARM"
	StatusBind       = "BIND"
	StatusFear       = "FEAR"

	// Buffs
	StatusThorns     = "THORNS"
	StatusRegen      = "REGEN"
	StatusAtkUp      = "ATK_UP"
	StatusDefUp      = "DEF_UP"
	StatusTaunt      = "TAUNT"
	StatusStealth    = "STEALTH"
	StatusHaste      = "HASTE"
	StatusInvincible = "INVINCIBLE"

	// Gu World Dao Mechanics
	StatusEssenceLeak  = "ESSENCE_LEAK"
	StatusGuSeal       = "GU_SEAL"
	StatusSoulWound    = "SOUL_WOUND"
	StatusThoughtChaos = "THOUGHT_CHAOS"
	StatusDaoConflict  = "DAO_CONFLICT"
	StatusBacklash     = "BACKLASH"
	StatusReviveLock   = "REVIVE_LOCK"
)

type StatusCategory string

const (
	CategoryPhysicalDoT StatusCategory = "PHYSICAL_DOT"
	CategoryMentalCC    StatusCategory = "MENTAL_CC"
	CategoryAperture    StatusCategory = "APERTURE"
	CategoryDaoConflict StatusCategory = "DAO_CONFLICT"
	CategoryBuff        StatusCategory = "BUFF"
	CategoryImmortal    StatusCategory = "IMMORTAL"
)
```

### `simulator/status_apply.go`

```go
package simulator

import (
	"math/rand"
	"yourproject/internal/statusfx"
)

// CanApplyStatus tính toán xác suất thành công dựa trên Cảnh Giới và Đạo Ngân
func CanApplyStatus(caster, target *Actor, spec statusfx.StatusSpec) bool {
	if !spec.CanResist {
		return true
	}

	// 1. Hệ số chênh lệch Chuyển (Rank)
	rankDiff := caster.Rank - target.Rank
	rankFactor := 1.0
	switch {
	case rankDiff >= 2:
		rankFactor = 1.5
	case rankDiff == 1:
		rankFactor = 1.2
	case rankDiff == -1:
		rankFactor = 0.6
	case rankDiff <= -2:
		rankFactor = 0.1
	}

	// Tiên Cổ bảo lưu uy lực tối thiểu
	if caster.IsImmortalGuCaster && rankFactor < 1.0 {
		rankFactor = 1.0
	}

	// 2. Chênh lệch Đạo Ngân theo Lưu Phái
	daoDiff := caster.GetDaoMarks(spec.Path) - target.GetDaoMarks(spec.Path)
	daoMultiplier := 1.0 + (float64(daoDiff) / 100.0)
	if daoMultiplier < 0.2 {
		daoMultiplier = 0.2
	}

	// 3. Tính toán xác suất cuối cùng
	finalProcChance := spec.BaseChance * daoMultiplier * rankFactor * (1.0 - target.StatusResist)
	return rand.Float64() < finalProcChance
}
```

### `simulator/mechanics.go`

```go
package simulator

import "yourproject/internal/statusfx"

// DispelActorStatuses gỡ bỏ hiệu ứng theo phân loại và kích hoạt bẫy phản phệ
func DispelActorStatuses(actor *Actor, category statusfx.StatusCategory, maxCount int) []string {
	cleansed := make([]string, 0)
	remaining := make([]ActiveStatus, 0, len(actor.Statuses))

	for _, st := range actor.Statuses {
		spec := statusfx.Registry[st.ID]

		// Không thể giải trạng thái Tiên cấp hoặc không khớp danh mục
		if spec.Category == statusfx.CategoryImmortal || spec.Category != category || len(cleansed) >= maxCount {
			remaining = append(remaining, st)
			continue
		}

		// Kích hoạt bẫy nổ máu của Huyết Đạo khi bị cưỡng chế giải trừ
		if st.ID == statusfx.StatusBleed && st.SourcePath == "BLOOD" {
			burstDmg := st.Value * float64(st.RemainingTurns) * 0.5
			actor.HP -= burstDmg
		}

		cleansed = append(cleansed, st.ID)
	}

	actor.Statuses = remaining
	return cleansed
}
```

---

## 7. Kế Hoạch Ánh Xạ Dữ Liệu `Gu.csv`

1. **Bổ sung Tag Hiệu Ứng Mới vào CSV:** Bổ sung các cột/giá trị cho `apply_status`: `ESSENCE_LEAK`, `GU_SEAL`, `SOUL_WOUND`, `THOUGHT_CHAOS`, `DAO_CONFLICT`.
2. **Kích hoạt các Status đang chờ (Dormant Statuses):** Kết nối `HASTE`, `ARMOR_BREAK`, `DAMAGE_REDUCTION_UP`, `INVINCIBLE`, `REVIVE_LOCK` vào luồng tính toán chiến đấu thực tế.
3. **Đồng bộ KillerMove JSON:** Thêm các trường hiệu ứng đặc thù vào cấu trúc JSON của Sát Chiêu.