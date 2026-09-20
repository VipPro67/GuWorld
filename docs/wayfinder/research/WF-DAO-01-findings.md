# [RESEARCH FINDINGS] Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Codebase

**Ticket:** [WF-DAO-01: Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers](../tickets/WF-DAO-01-research-path-mastery-dao-mark-combat.md)  
**Tác giả:** Antigravity  
**Ngày hoàn thành:** 2026-09-20  

---

## 1. Hiện trạng Lưu Trữ Dữ Liệu (Storage & Models)

| Thực thể | Vị trí lưu trữ | Kiểu dữ liệu | Ý nghĩa |
|---|---|---|---|
| **Đạo Ngân Tiên Khiếu** | `immortal_apertures.dao_marks` | `JSONB` (`models.DaoMarksMap` $\to$ `map[string]int64`) | Lưu trữ toàn bộ Đạo ngân của người chơi trong Tiên Khiếu. |
| **Đạo Ngân Lưu Phái Bảng Phụ** | `player_dao_path` | Row `(player_id, path, dao_marks BIGINT)` | Bảng phụ tra cứu / index theo từng phái. |
| **Cảnh Giới Lưu Phái (Chân Ý)** | `players.path_masteries` | `JSONB` (`models.PathMasteriesMap` $\to$ `map[string]int`) | Điểm tích lũy Chân Ý theo từng lưu phái. |
| **Đạo Ngân & Cảnh Giới Quái** | `monster_templates.dao_marks`, `path_masteries` | `JSONB` | Template đạo ngân và cảnh giới của quái/boss. |
| **Combatant State** | `models.Combatant`, `simulator.CombatantAuto` | `DaoMarks`, `PathMasteries`, `DaoBonus`, `PathDamageBonus` | Được snapshot từ player/monster khi khởi tạo combat. |

---

## 2. Công Thức & Logic Tính Toán Hiện Hành

### 2.1. Đạo Ngân & Khuếch Đại Đơn Hệ (`gamefx/fx.go` & `player_daongan.go`)
- **Công thức hiện tại:**
  $$\text{DaoPathMultiplier} = 1.0 + \frac{\text{marks}}{100.0} + \text{bonus}$$
- **Nhược điểm:**
  1. Tuyến tính vô hạn: $100.000$ đạo ngân $\to$ multiplier $= \times 1001.0$ (chưa có diminishing return).
  2. Độc lập hoàn toàn giữa các phái: Dùng Hỏa Cổ nhưng có $100.000$ Thủy đạo ngân thì Thủy đạo ngân hoàn toàn bị bỏ qua, không gây bất kỳ ảnh hưởng hay xung đột nào.

### 2.2. Cảnh Giới Lưu Phái (`gamefx/path_mastery.go`)
- **Mốc điểm Chân Ý:**
  - `Ordinary` (Phàm Thai): $0$ điểm ($\times 1.0$)
  - `Master` (Đại Sư): $\ge 100$ điểm ($\times 1.1$)
  - `Grandmaster` (Tông Sư): $\ge 1.000$ điểm ($\times 1.25$)
  - `Great Grandmaster` (Đại Tông Sư): $\ge 10.000$ điểm ($\times 1.4$)
  - `Supreme Grandmaster` (Vô Thượng): $\ge 100.000$ điểm ($\times 1.5$)
- **Nhược điểm:** Cảnh giới hiện chỉ là một multiplier khuếch đại sát thương cố định, chưa có cơ chế điều hòa xung đột ($\text{RealmControl}$) hay tăng hiệu quả khai thác đạo ngân ($\text{RealmEfficiency}$).

### 2.3. Vòng Lặp Combat (`simulator/engine.go` & `handlers/combat_effects.go`)
- **Sát thương đầu ra:**
  $$\text{Damage} = (\text{BaseDmg} + \text{BaseAttack}) \times \text{PowerScale} \times \text{MasteryMult} \times \text{DaoMult} \times \text{PathDamageMult}$$
- **Thiếu sót cốt lõi:**
  1. Thiếu hệ số xung đột nội tại ($\text{InternalConflictMultiplier}$).
  2. Thiếu lớp giảm trừ sát thương tự nhiên của người phòng thủ ($\text{NaturalDaoResistance}$).
  3. Chưa hỗ trợ sát chiêu phức hợp kết hợp đa hệ.

---

## 3. Phân Tích Điểm Nghẽn Hiệu Năng (Performance Bottlenecks)

1. **Chi phí tính toán trên mỗi đòn đánh:**
   - Trong `simulator/engine.go`, một trận đấu có thể kéo dài $50 - 100$ turns giữa nhiều combatants.
   - Nếu mỗi đòn đánh phải duyệt qua toàn bộ $50$ lưu phái để tính tổng $\sum \text{ForeignMarks} \times (1 - \text{Compatibility})$, độ phức tạp sẽ là $O(N_{\text{paths}})$, làm chậm simulator vòng lặp.
2. **Giải pháp tối ưu:**
   - Xây dựng **`DaoCombatProfile`** được tính sẵn $1$ lần khi khởi tạo `CombatState` / `CombatantAuto`.
   - Lưu trữ dạng `map[string]float64` (hoặc fixed array) cho các hệ số:
     - `OffensiveConflict[path]` $\to$ Hệ số xung đột khi dùng chiêu phái đó.
     - `NaturalResistance[path]` $\to$ Kháng tính tự nhiên khi bị đánh bởi chiêu phái đó.
     - `RealmEfficiency[path]` $\to$ Hiệu suất cảnh giới.
   - Khi ra chiêu, việc áp dụng công thức chỉ tốn $O(1)$.

---

## 4. Đề Xuất Kế Hoạch Cho Các Ticket Tiếp Theo

1. **WF-DAO-02:** Định nghĩa enum/struct 10 Dao Groups (`ELEMENTAL`, `MENTAL`, `SOUL`, `BODY`, `SPACE_TIME`, `RULE`, `INFORMATION`, `TRANSFORMATION`, `BLOOD`, `HEAVEN_EARTH`), ma trận tương thích nhóm $10 \times 10$, các cặp override đặc thù, và vùng đệm an toàn `GraceThreshold` ($\le 1.000$ marks hoặc $\le 2\%$).
2. **WF-DAO-03:** Hoàn thiện công thức toán học cho `RealmControl`, `RealmEfficiency`, `DaoAmplification` (với diminishing returns) và `InternalConflictMultiplier`.
3. **WF-DAO-04:** Thiết kế chi tiết cơ chế `NaturalDaoResistance` (cap $20\% - 35\%$) và logic cho trait `NON_CONFLICTING_DAO_MARKS`.
4. **WF-DAO-05:** Đặc tả Sát chiêu Phức Hợp (`Compound Killer Moves`) và cơ chế Song Đạo Điều Hòa (`Dual-Path Harmony`).
5. **WF-DAO-06:** Thiết kế struct `DaoCombatProfile`, thuật toán cache $O(1)$ và API endpoint trả về thông số xung đột cho UI.
