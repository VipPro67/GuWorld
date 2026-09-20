# [WF-DAO-04] Đặc tả Phòng Thủ Tự Nhiên (Natural Resistance) & Trade-off Đạo Ngân Bất Tương Xung

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:grilling`  
**Status:** Closed  
**Assignee:** Antigravity  

---

## Question

1. Cơ chế **Phòng Thủ Tự Nhiên (Natural Dao Resistance)**:
   - Công thức suy giảm tiệm cận $\text{MaxResistance} \times \frac{\text{DefenseConflict}}{\text{DefenseConflict} + \text{ResistanceScale}}$ với trần $30\%$.
2. Cơ chế & Trade-off của đặc tính **Đạo Ngân Bất Tương Xung (`NON_CONFLICTING_DAO_MARKS`)**:
   - Khi tấn công: $\text{ConflictMultiplier} = 1.00$.
   - Khi phòng thủ: $\text{NaturalResistance} = 0.00$.
   - Tuyệt đối không cài đặt thành "nhận thêm $25\%$ sát thương".
3. Đan xen thứ tự tính toán giữa Natural Resistance, Path Resistance, Defense Stat và Shield.

---

## Resolution

Đã hoàn thành đặc tả chi tiết tại [WF-DAO-04 Spec](../specs/WF-DAO-04-natural-dao-resistance.md):
- **Defense Conflict:** $\sum M(P_{\text{def}}) \times (1 - \text{Comp}(\text{attackPath}, P_{\text{def}}))$.
- **Natural Resistance:** Tiệm cận với trần $30\%$ ($\text{Scale} = 50.000$), giúp Cổ Tiên có lượng đạo ngân dị chủng lớn giảm tới $20\% - 30\%$ sát thương nhận vào.
- **Non-Conflicting Trade-off:** $\text{ConflictMultiplier} = 1.0$ khi công, nhưng $\text{NaturalResistance} = 0.0$ khi thủ.
- **Defense Pipeline:** Phân lớp thứ tự tính sát thương: Kháng tự nhiên $\to$ Kháng tộc/thể chất $\to$ Giáp $\to$ Khiên $\to$ Trừ máu.

