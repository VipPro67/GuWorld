# Ticket 07: Chốt kiến trúc Block v2 Schema và tích hợp Simulator Turn-based

- GitHub Issue: [#454](https://github.com/VipPro67/GuWorld/issues/454)
- Type: `wayfinder:task`
- Status: Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448), [#449](https://github.com/VipPro67/GuWorld/issues/449), [#450](https://github.com/VipPro67/GuWorld/issues/450), [#451](https://github.com/VipPro67/GuWorld/issues/451)
- Part of: [#447](https://github.com/VipPro67/GuWorld/issues/447)

## Question

Quy chuẩn mặt kỹ thuật để tích hợp các cơ chế mới vào hạ tầng GuWorld:
1. **Mở rộng Block v2 JSON Schema**:
   - Cần thêm Triggers nào (ví dụ: `ON_COLLISION`, `ON_GAUGE_FULL`, `ON_TERRAIN_ENTER`)?
   - Cần thêm Operations nào (ví dụ: `GAUGE_ADD`, `GAUGE_CONSUME`, `TERRAIN_APPLY`, `RULE_ENFORCE`)?
2. **Engine Simulator Turn-based**:
   - Tích hợp xử lý va chạm Grid (Collision detection).
   - Lưu trữ state của Custom Gauges và Terrain Tiles trong `CombatState`.
   - Đảm bảo tính tất định (Deterministic) tuyệt đối với seed RNG.
3. **Data Mapping**:
   - Ánh xạ vào `Gu.csv`, `KillerMoves.csv` và `manifest.go` mà không làm vỡ các effect cũ.
