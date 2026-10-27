# [SPECIFICATION] Công Thức Điểm Chính Xác → Tỉ Lệ Thành Công

**Ticket:** [WF-REF-03: Công thức Điểm Chính Xác → Tỉ Lệ Thành Công, hình phạt chết & thưởng Chân Ý](../tickets/WF-REF-03-cong-thuc-diem-chinh-xac-va-hinh-phat.md)
**Map:** [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Ngày hoàn thành:** 2026-10-27

---

## 1. Nguyên tắc thiết kế

Điểm chính xác (accuracy) là **thành phần phụ**, không phải trục chính của tỉ lệ luyện cổ.

Thứ tự ưu tiên của tỉ lệ thành công:

1. **Công thức gốc + Cảnh giới lưu phái Luyện Đạo** — trục chính.
2. **Cảnh giới lưu phái của Cổ đích** — phụ.
3. **Đạo ngân / may mắn / môi trường / nguyên liệu / pity** — phụ.
4. **Điểm chính xác từ minigame** — phụ, và **giảm dần theo rank Cổ**.

Vì vậy accuracy chỉ **cộng dương**, có trần nhỏ, và trần nhỏ dần khi rank tăng. Chơi dở không bị trừ tỉ lệ; chơi tốt chỉ được thêm một phần khiêm tốn.

---

## 2. Điểm chính xác

### 2.1 Điểm từng bước
- Mỗi bước trả về `0..100` do server chấm (xem WF-REF-04 cho luật từng minigame).
- Bước bỏ lỡ / quá hạn = **0 điểm**, không phạt thêm các bước khác.
- Không có điểm âm.

### 2.2 Tổng hợp
Trọng số **cộng đều** cho v1:

$$\text{Accuracy} = \frac{1}{N}\sum_{i=1}^{N} \text{StepScore}_i$$

Trong đó $N$ là tổng số bước của phiên (đã bao gồm cả bước bị bỏ lỡ). Không nhân hệ số cho bước cuối/khai lò.

Làm tròn: giữ 2 chữ số thập phân khi lưu và hiển thị, dùng giá trị chưa làm tròn khi tính bonus.

---

## 3. Bonus tỉ lệ từ accuracy

### 3.1 Công thức

$$\text{AccuracyBonus} = \text{MaxBonus}(\text{rank}) \times \mathrm{clamp}\left(\frac{\text{Accuracy} - 50}{50},\ 0,\ 1\right)$$

- Accuracy $\le 50$ → bonus $= 0$.
- Accuracy $= 100$ → bonus $= \text{MaxBonus}(\text{rank})$.
- Tuyến tính ở giữa.

### 3.2 Trần bonus theo rank

| Rank Cổ | MaxBonus |
|---|---|
| 1–2 | **+8%** |
| 3–4 | **+7%** |
| 5 | **+6%** |
| 6–7 | **+5%** |
| 8–9 | **+4%** |

Rank càng cao → trần càng thấp, đúng định hướng "accuracy là phần phụ, về late-game càng nhỏ".

### 3.3 Vị trí trong pipeline tỉ lệ hiện có

Thứ tự áp dụng (thay đổi tối thiểu so với `refine_gu.go` hiện tại):

```
baseRate = recipe.SuccessRate
positive = refinementDaoBonus
         + refinementMasteryBonus
         + targetPathMastery
         + luckBonus
         + supportBonus
         + envBonus
         + materialQualityBonus
         + insightBonus
         + accuracyBonus          // MỚI

// ImmortalPenalty áp dụng cho toàn bộ bonus dương, gồm accuracyBonus
if immortalPenalty > 1.0:
    mọi bonus dương /= immortalPenalty

capped = min(1.0, baseRate + positive)
final  = max(0.0, capped - statusPenalty - deathPenalty)
```

Quy tắc:
- `accuracyBonus` cộng **trước** bước chặn trần 100%.
- Chịu `ImmortalPenalty` giống các bonus dương khác.
- `refine_fatigue` / `refine_backfire` vẫn trừ **sau** cap như hiện tại.
- `deathPenalty` trừ **sau** cap và **sau** status penalty, trước sàn 0.

### 3.4 Trường hợp vô hiệu
`accuracyBonus = 0` khi:
- Nguyên liệu không khớp công thức (`guRequirementsNotMet`) — tỉ lệ đã bị ép 0.
- Tiên Cổ mục tiêu đã tồn tại (`isUniqueImmortalFail`) — lần luyện chắc chắn thất bại, accuracy không cứu được.
- Phiên bị hủy (`ABORTED`).

---

## 4. Hình phạt chết

Người chơi chết trong lúc phiên đang chạy:

- **Không** kết thúc phiên, **không** mất thêm tài nguyên ngoài hình phạt chết hiện có.
- Tỉ lệ thành công cuối bị trừ **25 điểm phần trăm** (`deathPenalty = 0.25`).
- **Không stack**: chết nhiều lần trong cùng phiên vẫn chỉ trừ 0.25.
- Áp dụng một lần, từ thời điểm chết đến hết phiên.
- Có thể đẩy tỉ lệ về 0 nhưng không âm.
- Hiển thị banner/trạng thái trong UI: "Tâm thần chấn động — tỉ lệ thành công −25%".

Ghi chú triển khai: cần một cờ `death_penalty_applied` (hoặc giá trị `death_penalty` trong bảng phiên) để worker/lazy-resolve không trừ lặp.

---

## 5. Thưởng Chân Ý Luyện Đạo

Chỉ khi **thành công**:

$$\text{MasteryGain} = \text{calculateRefinementPathGain}(\text{rank}, \text{realm}) \times \left(1 + \frac{\text{Accuracy}}{100}\right)$$

- Accuracy $= 100$ → $\times 2$.
- Accuracy $= 0$ (bỏ lỡ hết) → $\times 1$ (bằng hiện tại).
- Làm tròn xuống số nguyên.
- **Không** cộng Chân Ý khi thất bại (giữ luật hiện tại).

Pity **không** bị ảnh hưởng:
- Fail hợp lệ → `insight_<rank>` +1 stack.
- Success → reset stack của rank đó.
- Accuracy không thay đổi điều kiện tăng/giảm pity.

---

## 6. Preview tỉ lệ

Thêm endpoint (chi tiết route thuộc WF-REF-05):

```
GET /api/refine/preview?recipe_id=...
```

Trả về:
- `base_rate` và toàn bộ breakdown hiện có (như `RefineRateBreakdown`).
- `accuracy_max_bonus` theo rank.
- `rate_min` = tỉ lệ nếu accuracy 0 (sau cap, trừ debuff hiện có, chưa trừ death).
- `rate_max` = tỉ lệ nếu accuracy 100 (sau cap, trừ debuff hiện có, chưa trừ death).
- `death_penalty` (hằng số 0.25) để UI hiển thị cảnh báo.
- Cảnh báo thiếu tài nguyên/nguyên liệu/Cổ.

Không trả seed, kế hoạch bước hay đáp án minigame.

**Kết quả final vẫn hiển thị đầy đủ breakdown** như hiện tại, bổ sung các trường:
- `accuracy` (0–100)
- `accuracy_bonus`
- `accuracy_max_bonus`
- `death_penalty`
- `step_scores` (mảng điểm từng bước)

Các trường mới thêm vào JSON, không đổi tên/ý nghĩa trường cũ → tương thích ngược với `RefiningFurnace.tsx`.

---

## 7. Ví dụ

### Ví dụ 1 — Rank 3, accuracy 100, không chết
- Base + bonus hiện có: 62%
- MaxBonus rank 3: +7%
- AccuracyBonus: +7%
- Tỉ lệ final: **69%**

### Ví dụ 2 — Rank 3, accuracy 75
- `(75−50)/50 = 0.5` → +3.5%
- Tỉ lệ final: **65.5%**

### Ví dụ 3 — Rank 7, accuracy 90, chết một lần
- Base + bonus hiện có: 41%
- MaxBonus rank 7: +5%
- AccuracyBonus: `5% × 0.8 = +4%` → 45%
- Death penalty: −25% → **20%**

### Ví dụ 4 — Rank 9, bỏ lỡ 2/6 bước, không chết
- Step scores: `[88, 0, 92, 0, 85, 90]` → accuracy = 59.17
- MaxBonus rank 9: +4%
- `(59.17−50)/50 = 0.1833` → +0.73%
- Bonus gần như không đáng kể — đúng thiết kế "phần phụ".

### Ví dụ 5 — Rank 5, accuracy 100, chết hai lần
- AccuracyBonus: +6%
- Death penalty: −25% (không stack)
- Nếu tỉ lệ sau cap là 55% → **30%**

---

## 8. Hằng số & nơi triển khai

Đề xuất đặt trong `backend/internal/gamefx/refine_accuracy.go` (hoặc file tương đương trong `gamefx`) để handler và test dùng chung:

```go
const RefineDeathPenalty = 0.25
const RefineAccuracyFloor = 50.0
const RefineAccuracySpan  = 50.0

func RefineAccuracyMaxBonus(rank int) float64
func RefineAccuracyBonus(accuracy float64, rank int) float64
func RefineMasteryMultiplier(accuracy float64) float64
```

Quy tắc:
- Không hardcode bảng trong handler.
- **Đã chuyển sang data-driven bởi WF-REF-06:** `MaxBonus` lấy từ cột `accuracy_max_bonus` của `refine_stage_templates`, snapshot vào phiên lúc tạo. Hàm `gamefx.RefineAccuracyBonus` chỉ còn là fallback/test helper.
- `death_penalty` và `mastery_multiplier_max` cũng lấy từ cùng bảng cấu hình (baseline 0.25 và 2.0), không hardcode trong handler.
- `RefineRateBreakdown` mở rộng ở `handlers/refine_core.go`.

---

## 9. Kiểm thử bắt buộc

- Accuracy dưới 50 → bonus 0.
- Accuracy 100 → đúng MaxBonus từng rank (bảng 5 mốc).
- Bonus cộng trước cap: base 98% + accuracy 8% → final 100%, không vượt 1.
- ImmortalPenalty chia đều accuracyBonus.
- Death penalty trừ sau cap, không stack, không âm.
- Unique immortal fail / sai nguyên liệu → accuracyBonus 0.
- Mastery gain ×2 ở accuracy 100, ×1 ở accuracy 0, làm tròn xuống.
- Pity không đổi khi accuracy cao/thấp.
- JSON breakdown có đủ trường mới, trường cũ giữ nguyên.

---

## 10. Việc chưa chốt (chuyển ticket khác)

- Luật chấm điểm từng minigame → WF-REF-04.
- Schema/API/worker → WF-REF-05.
- Bảng nhịp độ & cấu hình data-driven → WF-REF-06.
- UI hiển thị, preview, banner chết → WF-REF-07.
