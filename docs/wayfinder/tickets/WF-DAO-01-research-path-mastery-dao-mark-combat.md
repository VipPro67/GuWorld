# [WF-DAO-01] Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:research`  
**Status:** Open  
**Assignee:** Unassigned  

---

## Question

Cần rà soát toàn bộ codebase hiện tại về:
1. Nơi lưu trữ và cấu trúc dữ liệu của `dao_marks` và `path_masteries` (`immortal_apertures`, `player_dao_path`, `monster_templates`, `players.path_masteries`).
2. Cách tính toán chỉ số đạo ngân hiện tại trong `player_daongan.go` (`CalculateDaoNgan`) và `gamefx.DaoPathMultiplier`.
3. Cách `simulator/engine.go` và `handlers/combat.go` áp dụng `PathMasteryMultiplier`, Gu damage, và các hiệu ứng tộc/thể chất (`gamefx.Effects`).
4. Các điểm nghẽn hiệu năng (performance bottlenecks) nếu bổ sung tính toán xung đột giữa 50 lưu phái trong mỗi lượt combat.
