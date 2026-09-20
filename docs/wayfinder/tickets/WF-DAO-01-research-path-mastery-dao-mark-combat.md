# [WF-DAO-01] Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:research`  
**Status:** Closed  
**Assignee:** Antigravity  

---

## Question

Cần rà soát toàn bộ codebase hiện tại về:
1. Nơi lưu trữ và cấu trúc dữ liệu của `dao_marks` và `path_masteries` (`immortal_apertures`, `player_dao_path`, `monster_templates`, `players.path_masteries`).
2. Cách tính toán chỉ số đạo ngân hiện tại trong `player_daongan.go` (`CalculateDaoNgan`) và `gamefx.DaoPathMultiplier`.
3. Cách `simulator/engine.go` và `handlers/combat.go` áp dụng `PathMasteryMultiplier`, Gu damage, và các hiệu ứng tộc/thể chất (`gamefx.Effects`).
4. Các điểm nghẽn hiệu năng (performance bottlenecks) nếu bổ sung tính toán xung đột giữa 50 lưu phái trong mỗi lượt combat.

---

## Resolution

Đã hoàn thành khảo sát chi tiết tại [WF-DAO-01 Findings](../research/WF-DAO-01-findings.md):
- **Storage:** `immortal_apertures.dao_marks` (`map[string]int64`), `players.path_masteries` (`map[string]int`), `player_dao_path` (bảng phụ), `monster_templates`.
- **Logic hiện tại:** `DaoPathMultiplier` tính đơn hệ tuyến tính ($1 + \text{marks}/100$), chưa có diminishing return và chưa có xung đột ngoại hệ; `PathMasteryMultiplier` cố định theo bậc ($1.0 \to 1.5$), chưa có vai trò điều hòa.
- **Combat Loop:** `handlers/combat_effects.go` và `simulator/engine.go` thiếu `InternalConflictMultiplier` và `NaturalDaoResistance`.
- **Hiệu năng:** Giải quyết bài toán 50 lưu phái bằng cách tiền tính toán **`DaoCombatProfile`** khi khởi tạo combatant, giúp tra cứu trong vòng lặp đạt $O(1)$.

