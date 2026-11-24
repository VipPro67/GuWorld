# Ticket 01: Chốt quy chuẩn Khung Cơ Chế Chung (Universal Combat Baseline) trong combat 5x5

- GitHub Issue: [#448](https://github.com/VipPro67/GuWorld/issues/448)
- Type: `wayfinder:grilling`
- Status: Frontier (Unblocked)
- Part of: [#447](https://github.com/VipPro67/GuWorld/issues/447)

## Question

Làm thế nào để chuẩn hóa Khung Cơ Chế Chung (Universal Combat Baseline) áp dụng đồng nhất cho mọi trận đánh 5x5:
1. **Khống chế (CC)**: Định nghĩa chính xác 4 trạng thái cốt lõi (Stun, Freeze, Silence, Root/Immobilize), quy tắc thời hạn, kháng hiệu ứng (Tenacity) và thanh tẩy (Cleanse).
2. **Cơ học vị trí (Displacement & Collision)**: Mở rộng các cơ chế Đẩy lùi (Knockback), Kéo lại (Pull), Tráo đổi (Swap), Rút lui (Retreat) vừa thêm với luật Va Chạm (Collision): Nếu ô đích bị chặn bởi mục tiêu khác hoặc chạm rìa bàn cờ (Grid Boundary), tính toán sát thương va đập (Crush Damage) và gây Stun như thế nào?
3. **Phòng hộ (Shield & Barrier)**: Phân định cơ chế Lá chắn (Shield) trừ dần theo sát thương vs Rào chắn (Barrier) chặn hoàn toàn số lần đòn đánh (Hit counts), cùng cách tương tác với Đạo Ngân.
4. **Tiêu hao tài nguyên**: Cơ chế tiêu hao Chân Nguyên (Mana), Tinh Lực (Stamina), và Hồn Phách (Soul HP) trong vòng lặp chiến đấu theo lượt.
