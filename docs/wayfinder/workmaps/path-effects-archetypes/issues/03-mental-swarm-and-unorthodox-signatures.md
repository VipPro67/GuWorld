# Ticket 03: Chốt cơ chế riêng cho Nhóm Tinh Thần, Bầy Đàn và Tà Đạo

- GitHub Issue: [#450](https://github.com/VipPro67/GuWorld/issues/450)
- Type: `wayfinder:grilling`
- Status: Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448)
- Part of: [#447](https://github.com/VipPro67/GuWorld/issues/447)

## Question

Thiết kế cơ chế riêng đặc thù (Signatures) cho 3 nhóm Archetype:
1. **Tinh Thần & Quy Tắc (Trí Đạo, Hồn Đạo, Luật Đạo)**:
   - Trí Đạo: Cơ chế Niệm Đầu / Thôi Diễn (đọc trước ý định skill của địch / Telegraphed Intent, phản xạ giảm sát thương).
   - Hồn Đạo: Đòn đánh trực tiếp trừ Hồn Phách (bỏ qua giáp vật lý), gây hoảng loạn mất quyền kiểm soát.
   - Luật Đạo: Áp đặt quy tắc cưỡng chế (Cấm đổi vị trí, Cấm dùng Cổ phòng ngự, Phạt phản phệ khi vi phạm luật).
2. **Bầy Đàn & Quân Số (Nô Đạo, Trận Đạo)**:
   - Nô Đạo: Triệu hồi linh thú chiếm các ô trống trên Grid 5x5, chia sẻ sát thương bầy đàn, buff theo số lượng đàn.
   - Trận Đạo: Dựng cọc trận / mắt trận trên Grid, tạo vùng hào quang (Auras) kiểm soát hàng/cột.
3. **Tà Đạo & Độc Lập (Huyết Đạo, Độc Đạo, Ám Đạo, Vận Đạo)**:
   - Huyết Đạo: Hút máu, tăng sức mạnh khi máu thấp (Berserk).
   - Độc Đạo: Tích tầng độc rút máu theo lượt (DoT stacks), lây lan khi mục tiêu bị dịch chuyển/va chạm.
   - Ám Đạo: Ẩn nặc (Untargetable), đòn đánh hiểm hóc từ sau lưng.
   - Vận Đạo: Can thiệp tỷ lệ may rủi (ép trúng đòn, trừ vận khí né tránh của địch).
