# [WF-DAO-06] Kiến trúc Cache Dao Combat Profile O(1), Data Models & API Contract

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:task`  
**Status:** Closed  
**Assignee:** Antigravity  

---

## Question

1. Cấu trúc dữ liệu `DaoCombatProfile` trong Go (`gamefx` / `simulator` / `models`).
2. Hàm dựng Profile (`gamefx.BuildDaoCombatProfile`) với chi phí tính toán $1$ lần cho $49$ lưu phái.
3. Tích hợp truy xuất $O(1)$ vào `simulator.CombatantAuto` và `handlers/combat_effects.go`.
4. API Endpoint & DTO response mở rộng cho UI (`GET /api/player/daongan`).

---

## Resolution

Đã hoàn thành thiết kế kiến trúc chi tiết tại [WF-DAO-06 Spec](../specs/WF-DAO-06-combat-profile-architecture.md):
- **Data Model:** Định nghĩa `DaoCombatProfile` và `PathCombatModifiers` lưu sẵn `DaoAmp`, `RealmControl`, `RealmEfficiency`, `InternalConflictMult`, `NaturalResistance`, `Purity`.
- **Precomputation Algorithm:** Hàm `BuildDaoCombatProfile` tính toán $1$ lần cho $49$ phái $(< 5\mu s)$, đảm bảo hàng triệu đòn đánh trong vòng lặp combat chỉ tra cứu map $O(1)$.
- **Combat Pipeline:** Phương thức `GetOffensiveFactors` và `GetDefensiveResistance` tích hợp trực tiếp vào `simulator/engine.go` và `handlers/combat_effects.go`.
- **API Response:** Mở rộng DTO endpoint `GET /api/player/daongan` cung cấp breakdown trực quan cho giao diện người dùng.

