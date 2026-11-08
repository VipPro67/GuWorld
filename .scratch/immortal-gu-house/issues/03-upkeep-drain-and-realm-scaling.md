# Ticket 03: Công thức tiêu hao Tiên Nguyên duy trì theo chênh lệch cảnh giới Cổ Tiên và Rank Cổ Phòng

Type: research
Status: resolved
Blocked by: 01

## Question

Công thức chuẩn xác để tính toán tốc độ tiêu hao Tiên Nguyên duy trì (`UpkeepCost`) mỗi tick 50ms trong Zone Sim theo tỷ lệ chênh lệch cảnh giới giữa Cổ Tiên và Rank của Tiên Cổ Phòng là gì?

Cụ thể cần giải quyết các vấn đề:
1. Rà soát hệ thống cảnh giới trong codebase (`Rank 1..5` Phàm nhân, `Rank 6..9` Cổ Tiên / Tiên Tôn).
2. Xây dựng công thức tiêu hao duy trì (`mana_per_second` hoặc `immortal_essence_per_second`):
   - Khi Cổ Tiên đồng cấp với Cổ Phòng (ví dụ Cổ Tiên Thất Chuyển lái Cổ Phòng Thất Chuyển): Mức tiêu hao tiêu chuẩn $U_{\text{base}}$.
   - Khi Cổ Tiên vượt cấp lái Cổ Phòng cao hơn (ví dụ Cổ Tiên Lục Chuyển lái Tiên Cổ Phòng Bát Chuyển): Hệ số phạt tiêu hao lũy thừa do gánh nặng Tiên Nguyên.
   - Khi Cổ Tiên cấp cao lái Cổ Phòng cấp thấp: Giảm tiêu hao tương ứng.
3. Cơ chế cạn kiệt Tiên Nguyên: Khi Mana/Tiên Nguyên về 0, thực thi giải trừ cưỡng chế (`revertTransformationLocked(entity, "mana_depleted", true)`) và áp dụng trạng thái kiệt sức / phản phệ như thế nào?

## Answer

Dựa trên nghiên cứu chuyên sâu về hệ thống bản vị Tiên Nguyên (`HigherRankManaMultiplier` trong `gamefx/energy.go:79-92` và `rank_scale.go:13-24`), cấu trúc lưu trữ `CurrentMana` (`player_service.go:124-129`), và vòng lặp `TickTransformation` (`transformation.go:284-304`), công thức chuẩn xác được khóa như sau:

1. **Khắc phục Nghịch Lý Lệch Bản Vị (Currency Unit Paradox)**:
   - Trong GuWorld, `Combat.Mana` của Cổ Tiên lưu số lượng hạt Tiên Nguyên theo bản vị của chính cảnh giới họ (Lục chuyển: Thanh Đề; Thất chuyển: Hồng Táo; Bát chuyển: Bạch Ly; Cửu chuyển: Hoàng Hạnh). Tỷ lệ quy đổi chuẩn giữa 2 Rank liền kề là $\mathbf{100\times}$.
   - Công thức duy trì phải nhân với tỷ lệ chuyển đổi bản vị $100^{\Delta R}$ trước khi áp dụng hệ số phạt, triệt tiêu hoàn toàn nghịch lý Cổ Tiên Rank thấp lại tiêu tốn ít năng lượng hơn Rank cao khi lái cùng một Cổ Phòng.

2. **Công Thức Hợp Nhất (Unified Formula)**:
   $$\text{ManaUpkeepPerSecond} = U_{\text{base}} \times \mathcal{M}(R_H, R_P) \times F_{\text{sub}}$$
   
   Trong đó:
   - **$U_{\text{base}}$ (Tiêu hao cơ sở theo bản vị Rank Cổ Phòng $R_H$)**:
     $$U_{\text{base}} = \left( N_{\text{core}} \times 0.04 + N_{\text{aux}} \times 0.0005 \right) \times S_{\text{aux}}$$
     - $0.04$: Tiêu hao duy trì của mỗi Tiên Cổ hạch tâm (hạt Tiên Nguyên cấp $R_H$ / giây).
     - $0.0005$: Tiêu hao duy trì kết giới của mỗi Cổ phụ (200 Cổ phụ tăng thêm $+0.10$ hạt/s).
     - $S_{\text{aux}}$ (Hệ số ổn định ma trận trận pháp):
       $$S_{\text{aux}} = 1.0 + \max\left(0, \frac{30 \times N_{\text{core}} - N_{\text{aux}}}{100 \times N_{\text{core}}}\right)$$
       *(Nếu thiếu Cổ phụ làm đệm điều hòa Đạo Ngân giữa các Tiên Cổ, Đạo Ngân rò rỉ làm tăng tiêu hao lên đến $+30\%$; khi có $\ge 30$ Cổ phụ/Tiên Cổ thì $S_{\text{aux}} = 1.0$).*
   - **$F_{\text{sub}}$ (Chiết giảm Tiểu cảnh giới)**:
     $$F_{\text{sub}} = 1.0 - (\text{SubRank} - 1) \times 0.05$$
     *(Sơ kỳ: $1.0$; Trung kỳ: $0.95$; Hậu kỳ: $0.90$; Đỉnh phong: $0.85$ nhờ khả năng vi thao Tiên Nguyên thuần thục).*
   - **$\mathcal{M}(R_H, R_P)$ (Hệ số quy đổi bản vị & Chênh lệch Cảnh giới)**:
     $$\mathcal{M}(R_H, R_P) = \begin{cases}
     1.0 & \text{khi } R_P = R_H \text{ (Đồng Rank)} \\
     100^{(R_H - R_P)} \times 1.5^{(R_H - R_P)} & \text{khi } R_P < R_H \text{ (Vượt Rank - Phạt lũy tiến)} \\
     100^{(R_H - R_P)} \times 0.8^{(R_P - R_H)} & \text{khi } R_P > R_H \text{ (Rank cao lái Rank thấp)}
     \end{cases}$$

3. **Bảng Giá Trị Tham Chiếu Thực Tế (Cấu hình chuẩn 3 Core Gu, 100 Aux Gu $\Rightarrow U_{\text{base}} = 0.17$)**:
   - **Đồng Rank (Rank 7 lái Cổ Phòng Rank 7)**: Tiêu hao $0.17$ hạt Hồng Táo/s. Kho 200 hạt duy trì được ~20 phút chiến đấu liên tục.
   - **Vượt 1 Rank (Rank 6 Đỉnh phong lái Cổ Phòng Rank 7)**: Tiêu hao $0.17 \times 150 \times 0.85 = \mathbf{21.675\text{ hạt Thanh Đề / giây}}$. Với kho dự trữ 300 hạt, Cổ Tiên Lục chuyển chỉ gánh gượng được tối đa **~13.8 giây** trước khi kiệt quệ Tiên Khiếu.
   - **Vượt 2 Rank (Rank 6 lái Cổ Phòng Rank 8)**: Tiêu hao $> 3.000$ hạt Thanh Đề / giây $\rightarrow$ Tiên Khiếu cạn kiệt trong 1-2 tick, vỡ phòng tức thì.
   - **Rank Cao Lái Rank Thấp (Rank 8 lái Cổ Phòng Rank 7)**: Tiêu hao chỉ $0.0013$ hạt Bạch Ly / giây $\rightarrow$ Duy trì hơn 40 giờ, vận hành dễ dàng như đồ chơi.

4. **Cơ Chế Cạn Kiệt Tiên Nguyên & Phản Phệ**:
   - Trong vòng lặp 20Hz: $\text{ManaPerTick} = \text{ManaUpkeepPerSecond} / 20.0$.
   - Khi `entity.Combat.Mana < upkeep.ManaPerTick`:
     - Lập tức kích hoạt `revertTransformationLocked(entity, "mana_depleted", true)`.
     - Cổ Tiên bị trục xuất cưỡng chế ra ngoài nhục thân.
     - Dính debuff Kiệt Sức `EXHAUSTION` (100 ticks = 5.0 giây, giảm 30% tốc độ chạy).
     - Phát event `combat.EventTransformationBroken` (`Reason: "mana_depleted"`).
