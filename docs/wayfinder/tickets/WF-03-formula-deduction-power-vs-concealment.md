# WF-03: Formula design for Deduction Power vs Concealment Defense & Backfire Thresholds

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:grilling`  
**Status:** Closed  
**Blocked by:** None  

---

## Question

Thiết kế mô hình toán học và công thức tính toán so khớp giữa **Sức Mạnh Suy Diễn ($P_{div}$)** và **Phòng Ngự Ẩn Thân ($D_{con}$)**.

---

## Resolution

### 1. Nguyên Tắc Cốt Lõi (Data-Driven Effects)
Tất cả các chỉ số cộng/trừ từ Cổ trùng và Sát Chiêu **bắt buộc đọc từ Cấu Trúc Hiệu Ứng (Effects / Capabilities / Passive Stats)** trong Registry (`models.GetGu`, `Effects.Passive.Investigation`, `Effects.Passive.DisguiseLevel`, capability codes `PERFECT_DISGUISE`, `DISGUISE`, v.v.), tuyệt đối **không hardcode theo `gu_id`**.

### 2. Công Thức Lực Suy Diễn ($P_{div}$)
$$P_{div} = \Big(\text{BaseRank}_{viewer} + \sum \text{Investigation}_{effects}\Big) \times M_{attain} \times \Big(1 + \frac{\text{Dao}_{wisdom} + \text{Dao}_{info}}{1000}\Big) + Bonus_{clue} + Bonus_{resource}$$

- **BaseRank**: $\text{Rank} \times 10$ (Rank 1: 10, Rank 5: 50, Rank 6: 100, Rank 7: 180, Rank 8: 300, Rank 9: 500).
- **Investigation Effects**: Tổng chỉ số `Investigation` lấy từ passive effects của toàn bộ Cổ trùng đang kích hoạt.
- **Attainment Multiplier ($M_{attain}$)**: Lấy theo cảnh giới cao nhất của `Wisdom`, `Information`, hoặc `Investigation`:
  - Phàm Thai: $\times 1.0$
  - Đại Sư (Master): $\times 1.2$
  - Tông Sư (Grandmaster): $\times 1.5$
  - Đại Tông Sư (Great Grandmaster): $\times 2.0$
  - Vô Thượng (Supreme Grandmaster): $\times 3.0$
- **Đạo Ngân Multiplier**: Áp dụng cho Rank 6+ với Đạo ngân Trí Đạo và Tín Đạo: $1 + \frac{\text{Dao}_{wisdom} + \text{Dao}_{info}}{1000}$.
- **Điểm Manh Mối ($Bonus_{clue}$)**:
  - Đúng Username: $+20$
  - Đúng Tông Môn: $+10$
  - Đúng Vùng (Region): $+15$
  - Có Dấu Vết Khí Tức (Residual Scent): $+30$
  - Có Ấn Ký Truy Tung (Tracking Mark): $+50$
  - Dò tìm mù (không có tên cụ thể): $-40$
- **Tài Nguyên Đầu Tư ($Bonus_{resource}$)**: Tối đa $+25\%$ tổng điểm dựa trên tỷ lệ Chân Nguyên (100%) hoặc Tiên Nguyên Thạch (tối đa 50 viên).

### 3. Công Thức Phòng Ngự Ẩn Thân ($D_{con}$)
$$D_{con} = \Big(\text{BaseRank}_{target} + \sum \text{DisguiseLevel}_{effects} \times 5 + Bonus_{capability}\Big) \times M_{attain} \times \Big(1 + \frac{\text{Dao}_{conceal} + \text{Dao}_{dark} + \text{Dao}_{trans}}{1000}\Big) + Pen_{dist}$$

- **BaseRank**: $\text{Rank}_{target} \times 10$.
- **Disguise Effects**: Tổng chỉ số `DisguiseLevel` từ passive effects của Cổ trùng đang trang bị.
- **Bonus Capability**: Nếu có capability `PERFECT_DISGUISE` (hoặc sát chiêu che giấu đặc biệt từ effect spec) $\rightarrow$ cộng thêm $+80 \dots +150$ điểm phòng ngự.
- **Attainment Multiplier ($M_{attain}$)**: Theo cảnh giới của `Dark`, `Transformation`, `Wisdom`, hoặc `Concealment`.
- **Đạo Ngân Multiplier**: Áp dụng cho Rank 6+ với Đạo ngân Ám Đạo, Biến Hóa Đạo, Ẩn Đạo.
- **Rào Cản Không Gian ($Pen_{dist}$)**:
  - Cùng ô tọa độ: $+0$
  - Cùng Region: $+10$
  - Khác Region trong Ngũ Vực (Vượt Ngũ Vực Giới Bích): $+35$
  - Hai Thiên (Bạch/Hắc Thiên) hoặc Đóng cửa Tiên Khiếu: $+60$

### 4. Bảng Kết Quả Theo $\Delta = P_{div} - D_{con}$
| Hiệu số $\Delta$ | Kết quả | Chi tiết hiển thị & Hậu quả |
|---|---|---|
| $\Delta \ge +30$ | **Đại Thành Công (Full Insight)** | Thấu thị $100\%$ tọa độ `(region, x, y)`, Rank/SubRank thật, HP/Mana, toàn bộ Cổ trùng trang bị. |
| $0 \le \Delta < +30$ | **Tiểu Thành Công (Partial Insight)** | Biết khu vực `region`, tu vi ước lượng $(\pm 1)$, 2 Cổ trùng ngẫu nhiên. |
| $-30 \le \Delta < 0$ | **Thất Bại (Fogged)** | Bị màn sương che mắt, tiêu hao tài nguyên, không thu được tin tức. |
| $\Delta < -30$ | **Thiên Cơ Phản Phệ (Backfire)** | Trọng thương HP & Soul HP ($Dam = \|\Delta\| \times \text{Rank}_{target} \times 5$), mất $100\%$ Chân Nguyên / 10 Tiên Nguyên Thạch. **Mục tiêu nhận cảnh báo tức thì kèm tọa độ $(region, x, y)$ của kẻ suy diễn.** |
