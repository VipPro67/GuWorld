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

### 2. Công Thức Lực Suy Diễn ($P_{div}$ — 3 trụ, chỉ trả `my_score` cho client)
$$P_{div} = \underbrace{\text{Stat}}_{\text{Cổ}} + \underbrace{\text{Stat}\times(M_{trí}\times Dao-1)}_{\text{cảnh giới + đạo ngân}} + \underbrace{\min(40,\text{Clue}_{info}\times(1+0.15L_{tín})\times(1-\min(0.5,0.12L'_{tín}))) + Res_{25\%}}_{\text{manh mối}} + \underbrace{\text{LuckRoll}}_{\text{Vận}}$$

- **Stat**: $\text{BaseRank}_{viewer} + \sum \text{Investigation}_{effects}$ (BaseRank: R1-5 = 10-50, R6 = 100, R7 = 180, R8 = 300, R9 = 500).
- **Trí core**: $M_{attain}$ chỉ theo `wisdom` (1.0/1.2/1.5/2.0/3.0); Đạo ngân bão hòa $Dao(m) = 1 + 4m/(m+2000)$ theo $(\text{Dao}_{wisdom}+\text{Dao}_{info})$ (rank 6+, trần 5x: 1000 marks ≈ 2.3x). Tuyến tính $1+m/1000$ cũ nổ số với cá voi (100k marks = 100x nhân dồn mult).
- **Tín hỗ trợ**: khuếch đại clue $+15\%$/bậc cảnh giới `information` phe công; phe thủ bóp clue $-12\%$/bậc (trần $-50\%$).
- **Điểm Manh Mối** (đều validate server-side, sai = 0): đúng Username $+20$, đúng Region $+15$, đúng Sect $+10$, SuspectedGu đúng $+10$, ItemClueID đúng $+30$, dò mù $-40$. Trụ clue chặn trần $\pm 40$.
- **Tài nguyên (chỉ đá, không tốn mana)**: Nguyên Thạch $\times0.1$ (tối thiểu 100, cả divine người lẫn Cổ) / TNT $\times5$ (divine người), $\times6$ (divine Cổ, tối thiểu 1, chỉ Tiên nhân), trần $+25\%$ core (Cổ: $35\%$).
- **Vận**: roll ngẫu nhiên mỗi lần dò; công tu Vận cao thu hẹp biên, thủ Vận cao kéo mean âm. Seed qua `gamefx.ScoutingRand` để test.
- **Inspect cục bộ**: cùng engine, cộng sẵn $+20$ proximity clue; cùng Party/Sect auto-full; ngưỡng full $\ge 30$, partial $\ge -10$.

### 3. Công Thức Phòng Ngự Ẩn Thân ($D_{con}$ — server-side, không lộ)
$$D_{con} = \underbrace{\text{BaseRank}+ \sum \text{Disguise}\times5 + 80_{perfect} + 30_{conceal}}_{\text{stat}} + \underbrace{\text{Stat}\times(M_{attain}\times Dao(m)-1) + 8L_{biến hóa}}_{\text{cảnh giới}} + \underbrace{\text{Veil}_{thiên}}_{\text{Tiên}} + \underbrace{Pen_{dist} + 10_{\text{Ám}\ge1}}_{\text{anti-clue}}$$
($Dao(m)$ bão hòa như phe công, theo $\text{Dao}_{conceal}+\text{Dao}_{dark}+\text{Dao}_{trans}$.)

- **Vai 6 đạo**: Trí đối kháng trực tiếp (trong $M_{attain}$ cùng Ám/Biến hóa/concealment); Ám che thân/vị/khí + $+10$ rào khoảng cách; Tín bóp clue (mục trên); Vận roll; Thiên veil $= 20L_{thiên} + \text{Dao}_{heaven}/500$ (rank 6+, bị xuyên bởi $10L_{thiên} + \text{Dao}_{heaven}/500$ của công, trần = veil); Biến hóa $+8$/bậc + đổi output thành hàng giả khi partial/fogged.
- **Rào Không Gian**: cùng region $+10$, xuyên region $+35$.
- **Cổ có chủ (divine-gu)**: lấy $\max$ def chủ theo công thức trên (full $D_{con}$ như hiện tại); Cổ nằm kho/ngủ $\times 0.5$; Cổ trang bị $\times 1.0$. Viewer tự sở hữu → bỏ qua def chủ, cộng hưởng $+250$.
- **Sect vault**: $def = level\times20+10$, trọng số $0.5$ (thay $level\times10 / 0.4$); chợ công khai $def = 15$ cố định.
- **Suy tính Cổ**: $P_{raw}$ (stat + realm + resource + cộng hưởng sở hữu $+250$) **chia cho $Divisor = BaseRankPower/10$ theo bậc Cổ**: R1-5 = 1/2/3/4/5, R6 = 10, R7 = 18, R8 = 30, R9 = 50 (cùng họ với rankPenalty suy diễn bí phương). Không còn difficulty tuyến tính theo rank; phần khó còn lại chỉ là che giấu của chủ mạnh nhất $\times 0.5$ + màn sương tự nhiên $15$. Luck Vận cũng chia cho divisor. Ví dụ: R5 (P~85) full R1 ($85-15$), partial R5 ($17-15$); cá voi P~30k full R9 ($600-15$) nhưng phàm nhân dò Tiên Cổ phản phệ chắc chắn.

### 4. Bảng Kết Quả Theo $\Delta = P_{div} - D_{con}$ (so DUY NHẤT điểm tổng cuối)
| Hiệu số $\Delta$ | Kết quả | Chi tiết hiển thị & Hậu quả |
|---|---|---|
| $\Delta \ge +30$ (người/inspect) / $+20$ (Cổ) | **Đại Thành Công (Full Insight)** | Thấu thị $100\%$ tọa độ `(region, x, y)`, Rank/SubRank thật, HP/Mana, toàn bộ Cổ trùng trang bị. Kèm `my_score{stat, realm_dao, clue, luck_roll, total}`. |
| $0 \le \Delta < +30$ / $-10 \le \Delta < +20$ (Cổ) | **Tiểu Thành Công (Partial Insight)** | Biết khu vực `region`, tu vi ước lượng $(\pm 1)$, 2 Cổ trùng ngẫu nhiên. |
| $-30 \le \Delta < 0$ / $-40 \le \Delta < -10$ (Cổ) | **Thất Bại (Fogged)** | Bị màn sương che mắt, tiêu hao tài nguyên, không thu được tin tức. |
| $\Delta < -30$ / $< -40$ (Cổ) | **Thiên Cơ Phản Phệ (Backfire)** | Trọng thương HP & Soul HP ($Dam = \|\Delta\| \times \text{Rank}_{target} \times 5$; Cổ: $\times 3.5$ HP / $\times 1.5$ Soul), trừ 20 Chân Nguyên (cố định, không theo %) / 10 Tiên Nguyên Thạch. **Mục tiêu nhận cảnh báo tức thì kèm tọa độ $(region, x, y)$ của kẻ suy diễn.** |
