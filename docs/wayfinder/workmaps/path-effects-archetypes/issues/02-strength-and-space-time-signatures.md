# Ticket 02: Chốt cơ chế riêng và Custom Gauges cho Nhóm Thể Thuật & Không - Thời Gian

- GitHub Issue: [#449](https://github.com/VipPro67/GuWorld/issues/449)
- Type: `wayfinder:grilling`
- Status: Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448)
- Part of: [#447](https://github.com/VipPro67/GuWorld/issues/447)

## Question

Thiết kế cơ chế riêng đặc thù (Signatures) và thanh tài nguyên phụ (Custom Gauges) cho 2 nhóm Archetype:
1. **Thể Thuật & Xung Kích (Lực Đạo, Biến Hóa Đạo, Cốt Đạo)**:
   - Lực Đạo: Thú Hư Ảnh & Hiệu ứng Va Chạm Vách (Wall Crush) - dồn ép đối thủ vào chân tường hoặc ném đối thủ vào nhau.
   - Biến Hóa Đạo: Kích thước ô chiếm dụng (1x1 thành 2x2 trên Grid), thừa hưởng đặc tính loài thú hóa thân.
   - Cốt Đạo: Giáp gai phản đòn, thương tật thể xác tích tụ.
2. **Không - Thời Gian (Vũ Đạo, Trụ Đạo)**:
   - Vũ Đạo: Thao túng vị trí ô (Dịch chuyển tức thời, bẻ cong khoảng cách, hoán đổi vị trí bất kể hàng ngũ).
   - Trụ Đạo: Thao túng dòng thời gian lượt đi (Tua lượt hành động Turn Meter, hoãn Cooldown, Trì hoãn sát thương tích tụ / Damage Delay).
