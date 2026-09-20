# [SPECIFICATION] Phòng Thủ Tự Nhiên (Natural Dao Resistance) & Đặc Tính Bất Tương Xung

**Ticket:** [WF-DAO-04: Đặc tả Phòng Thủ Tự Nhiên (Natural Resistance) & Trade-off Đạo Ngân Bất Tương Xung](../tickets/WF-DAO-04-spec-natural-dao-resistance-and-traits.md)  
**Tác giả:** Antigravity  
**Ngày hoàn thành:** 2026-09-20  

---

## 1. Bản Chất Phòng Thủ Tự Nhiên Của Cổ Tiên

Trong nguyên tác Cổ Chân Nhân:
- Cơ thể Cổ Tiên được khắc sâu vô số Đạo ngân. Khi một sát chiêu ngoại lai xâm nhập cơ thể, các Đạo ngân bản thân sẽ tự động bài xích và triệt tiêu quy tắc của đòn đánh dị chủng.
- Lớp phòng thủ này là **bảo hộ bị động tự nhiên** (Passive Natural Defense), tồn tại liên tục mà không tiêu tốn Tiên Nguyên hay kích hoạt kỹ năng.
- Tuy nhiên, người sở hữu đặc tính **Đạo Ngân Bất Tương Xung (`NON_CONFLICTING_DAO_MARKS`)** thì Đạo ngân trong cơ thể hoàn toàn không bài xích bất kỳ quy tắc nào, dẫn đến việc mất đi lớp phòng thủ tự nhiên này.

---

## 2. Công Thức Tính Kháng Tính Tự Nhiên (Natural Dao Resistance)

Khi nhân vật (Defender) nhận một đòn đánh thuộc lưu phái $\text{attackPath}$:

### 2.1. Xung Đột Phòng Ngự ($\text{DefenseConflict}$):
$$\text{DefenseConflict}(\text{attackPath}) = \sum_{P_{\text{def}}} M(P_{\text{def}}) \times \left(1.0 - \text{Compatibility}(\text{attackPath}, P_{\text{def}})\right)$$
*(Áp dụng toàn bộ Đạo ngân của người thủ, tính cả tương thích nhóm và ghi đè cặp)*.

### 2.2. Kháng Tính Tự Nhiên ($\text{NaturalResistance}$):
$$\text{NaturalResistance}(\text{attackPath}) = \text{MaxResistance} \times \frac{\text{DefenseConflict}}{\text{DefenseConflict} + \text{ResistanceScale}}$$
- **Trần kháng tối đa:** $\text{MaxResistance} = 0.30$ ($30\%$).
- **Hằng số co giãn:** $\text{ResistanceScale} = 50.000$ (Tương đương mốc chuẩn Tiên nhân Rank 7).

---

## 3. Đặc Tính "Đạo Ngân Bất Tương Xung" (`NON_CONFLICTING_DAO_MARKS`)

### 3.1. Quy Tắc Hoạt Động:
- **Khi Công Kích (Offense):**
  $$\text{InternalConflictMultiplier} = 1.00$$
  - Nhân vật có thể tự do thi triển bất kỳ sát chiêu thuộc 50 lưu phái nào với $100\%$ hiệu suất khuếch đại Đạo ngân mà không bị trừ sát thương hay tăng tiêu hao mana.
- **Khi Phòng Thủ (Defense):**
  $$\text{NaturalResistance} = 0.00$$
  - Không có lớp giảm sát thương tự nhiên của Đạo ngân.

### 3.2. Nguyên Tắc Cài Đặt (Implementation Guideline):
- Tuyệt đối **không** cài đặt thành debuff "nhận thêm $25\%$ sát thương":
  ```go
  // SAI:
  if NonConflictingDaoMarks {
      incomingDamage *= 1.25
  }
  
  // ĐÚNG:
  if NonConflictingDaoMarks {
      naturalResistance = 0.0
  }
  ```
- Điều này phản ánh chính xác cơ chế: Người có trait không bị dính bùa hại; họ chỉ đơn giản không sở hữu lớp giảm trừ tự nhiên mà người khác có.

---

## 4. Thứ Tự Phân Lớp Sát Thương Khi Phòng Ngự (Defense Pipeline)

Một đòn đánh khi giáng vào mục tiêu sẽ đi qua các lớp độc lập:

```
[Raw Incoming Damage]
        │
        ▼
1. Kháng Tự Nhiên Đạo Ngân: × (1.0 - NaturalResistance)
        │
        ▼
2. Kháng Lưu Phái (Tộc/Thể chất): × (1.0 - PathResistance)
        │
        ▼
3. Giáp Cơ Bản (Defense Stat): × (100 / (100 + BaseDefense))
        │
        ▼
4. Khiên Năng Lượng (Shield): EffectiveDamage - Shield
        │
        ▼
5. Trừ Máu Thực Tế (HP Loss)
```

$$\text{DamageAfterReductions} = \text{RawDamage} \times (1.0 - \text{NaturalResistance}) \times (1.0 - \text{PathResistance}) \times \frac{100.0}{100.0 + \text{BaseDefense}}$$
$$\text{FinalHPLoss} = \max\left(0, \text{DamageAfterReductions} - \text{ActiveShield}\right)$$

---

## 5. Ví Dụ So Sánh Thực Tế (Case Studies)

Giả sử đòn đánh Thủy Đạo gây **$10.000$ sát thương thô**:

| Đối tượng phòng thủ | Đạo ngân | Kháng tự nhiên | Giáp ($50\%$) | Sát thương nhận |
|---|---|---|---|---|
| **Cổ Tiên Hỏa Đạo (Rank 7)** | $50.000$ Hỏa | $\mathbf{9.31\%}$ | $50\%$ | **$4.534$ HP** |
| **Cổ Tiên Hỏa Đạo (Rank 8)** | $200.000$ Hỏa | $\mathbf{19.28\%}$ | $50\%$ | **$4.036$ HP** |
| **Bất Tương Xung (Rank 7)** | $50.000$ Đa phái | $\mathbf{0.00\%}$ | $50\%$ | **$5.000$ HP** |
| **Bất Tương Xung (Rank 8)** | $200.000$ Đa phái | $\mathbf{0.00\%}$ | $50\%$ | **$5.000$ HP** |

$\implies$ Người có Bất Tương Xung nhận sát thương cao hơn người thường khoảng $10\% - 20\%$ nếu không chủ động dựng khiên chắn hoặc sát chiêu phòng hộ.
