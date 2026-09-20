# [WF-DAO-02] Thiết kế 10 Nhóm Lưu Phái (Dao Groups), Ma Trận Tương Thích & Vùng Đệm An Toàn

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:grilling`  
**Status:** Closed  
**Assignee:** Antigravity  

---

## Question

1. Phân loại ~50 lưu phái hiện có trong GuWorld vào **10 Dao Groups** như thế nào để đảm bảo tính logic và cân bằng?
2. Bảng ma trận tương thích giữa 10 nhóm ($10 \times 10$) với giá trị mặc định $\in [0.55, 1.00]$.
3. Danh sách các cặp ngoại lệ ghi đè (Pair Overrides) phản ánh tương sinh / tương khắc trong nguyên tác Cổ Chân Nhân.
4. Cơ chế Vùng Đệm An Toàn (**Grace Threshold**) chống hỏng build khi nhặt đạo ngân rác.

---

## Resolution

Đã hoàn thành đặc tả chi tiết tại [WF-DAO-02 Spec](../specs/WF-DAO-02-dao-groups-compatibility.md):
- **Phân nhóm 49 lưu phái** (theo `dreamRealmPaths` & `pathToVietnamese`) vào 10 Dao Groups: `ELEMENTAL`, `SPACE_TIME`, `RULE_ORDER`, `KINETIC_WEAPON`, `BODY_LIFE`, `SOUL_MIND`, `INFORMATION_STEALTH`, `ORGANIC_CORRUPTION`, `CRAFT_ALCHEMY`, `HEAVEN_HUMAN`.
- **Ma trận 10x10 Group Compatibility** đối xứng với khoảng giá trị từ $0.80 \to 0.96$.
- **14 Cặp Pair Overrides** đặc thù (Khắc sâu: Hỏa-Thủy 0.55, Hỏa-Băng 0.60, Quang-Ám 0.55; Sinh/Hòa hợp: Ngũ hành tương sinh 0.95, Vũ-Trụ 0.97, Trí-Tín 0.98, Lực-Cốt 0.96, v.v.).
- **Grace Threshold:** Bỏ qua đạo ngân ngoại hệ $\le \max(1000, \text{TotalDaoMarks} \times 0.02)$.

