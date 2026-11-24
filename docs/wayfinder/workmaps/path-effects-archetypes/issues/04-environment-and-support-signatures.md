# Ticket 04: Chốt cơ chế riêng cho Nhóm Môi Trường & Bổ Trợ

- GitHub Issue: [#451](https://github.com/VipPro67/GuWorld/issues/451)
- Type: `wayfinder:grilling`
- Status: Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448)
- Part of: [#447](https://github.com/VipPro67/GuWorld/issues/447)

## Question

Thiết kế cơ chế riêng đặc thù (Signatures) cho 2 nhóm Archetype:
1. **Môi Trường & Nguyên Tố (Hỏa, Băng, Lôi, Phong, Thổ, Kim, Mộc, Thủy)**:
   - Biến đổi ô môi trường trên Grid 5x5 (Terrain Tiles):
     - Ô Lửa: Thiêu đốt liên tục theo lượt.
     - Ô Băng: Trơn trượt (mất thăng bằng, tăng cự ly đẩy lùi, giảm tốc).
     - Ô Sét: Dẫn điện lan truyền sát thương cho các mục tiêu đứng cùng hàng/cột.
     - Ô Bùn / Vách Đất: Chặn đường di chuyển hoặc cản đường đạn.
2. **Bổ Trợ & Khởi Sinh (Luyện Đạo, Quang Đạo, Âm Dương Đạo, Thực Đạo)**:
   - Luyện Đạo: Cường hóa Cổ trùng ngay trong giao tranh (tăng tạm thời phẩm cấp/chỉ số Cổ).
   - Quang / Âm Dương / Thực Đạo: Tẩy rửa debuff cấp tốc, hồi sinh lực/chân nguyên, đảo ngược thuộc tính công thủ.
