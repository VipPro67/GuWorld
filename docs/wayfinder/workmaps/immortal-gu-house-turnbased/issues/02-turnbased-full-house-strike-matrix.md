# Ticket 02: Ma trận 6 Chiêu thức Trận Kích chuẩn hóa và Công thức Scaling

Type: prototype
Status: resolved
Blocked by: Ticket 01
Assignee: Antigravity

## Question

Quy tắc kiến tạo, cấu trúc dữ liệu và công thức scaling cho bộ 6 chiêu thức Trận Kích (`Full House Strike Set`) trong Combat Theo Lượt là gì để vừa tuân thủ chuẩn 6 vai trò của `CONTEXT.md`, vừa tận dụng sức mạnh tổng của Cổ Phòng và các Cổ hạch tâm đã ghép qua `gu_house_action.v1`?

Cụ thể cần giải quyết:
1. **Ánh xạ 6 Slot Vai Trò Chuẩn Hóa Theo Lượt**:
   - **Slot 1 (Pháo kích công kích - Artillery Strike)**: Sát thương đơn mục tiêu tầm xa, thừa hưởng Base Damage từ Cổ hạch tâm số 1 (hoặc Cổ công kích mạnh nhất).
   - **Slot 2 (Xung kích càn quét - Siege Trample)**: Sát thương diện rộng lên hàng trước hoặc toàn đội địch, kèm hiệu ứng **Turn / Initiative Pushback** (đẩy lùi thanh lượt hoặc giảm Speed của đối thủ).
   - **Slot 3 (Đại trận phòng hộ - Fortress Barrier)**: Tạo khiên quang thuẫn hấp thụ sát thương dựa trên % Form HP tối đa hoặc Base Shield của Cổ hạch tâm phòng hộ.
   - **Slot 4 (Hồi phục sửa chữa - Structural Mending)**: Hồi phục trực tiếp Form HP của Cổ Phòng (không hồi máu nhục thân).
   - **Slot 5 (Thần thông hủy diệt tối hậu - Ultimate Destruction)**: Kỹ năng tụ lực `ChargingTurnsLeft` (1 lượt tụ khí đại trận), sang lượt sau phóng thích đại sát thương AoE cực lớn.
   - **Slot 6 (Lệnh rời Cổ Phòng - Disembark)**: Lệnh giải trừ dung nhập, trả Cổ Tiên về dạng nhục thân.
2. **Công thức Scaling Theo Sức Mạnh Tổng Thể Cổ Phòng**:
   - Mỗi chiêu thức được khuếch đại theo công thức tổng hòa:
     $$\text{EffectivePower} = \text{BaseVal} \times \left(1.0 + \frac{\text{AuxGuCount}}{10,000} \times \alpha + \text{CoreGuBonus} \times \beta\right) \times \text{RankMultiplier}$$
   - Làm thế nào để Rank Cổ Phòng (Rank 6–9) và số lượng Phàm Cổ phụ trợ (500–100,000 con) tạo nên sự khác biệt rõ rệt về uy lực và độ dày giáp?
3. **Tích hợp với Effect Composition Graph (`gu_house_action.v1`)**:
   - Các Trận Kích do người chơi tự ghép trong `GuHousePanel.tsx` sẽ map vào Slot 1, 3, 4 như thế nào? Nếu người chơi không ghép đủ 3 loại thì Cổ Phòng có chiêu thức mặc định trám vào hay không?

## Answer

Đã chuẩn hóa toàn bộ ma trận 6 vai trò Trận Kích, công thức scaling sức mạnh tổng thể và quy tắc tích hợp với hệ thống Effect Composition:

1. **Ma Trận 6 Vai Trò Chuẩn Hóa Theo Lượt**:
   - **Slot 1 (Pháo Kích Công Kích - Artillery Strike)**:
     - *Nguồn*: Cổ hạch tâm có `OpDamage` cao nhất (hoặc Cổ 1).
     - *Mục tiêu*: `SINGLE`.
     - *Sát thương*: $D = (\text{BaseCoreDmg} + \text{Rank} \times 100) \times \text{HousePowerScale}$.
     - *Chi phí*: $30 + \text{Rank} \times 5$ Mana. Cooldown: 1 lượt. Trừ 1 độ no Cổ hạch tâm 1.
   - **Slot 2 (Xung Kích Càn Quét - Siege Trample)**:
     - *Nguồn*: Thuộc tính cơ động/pháo đài nội tại của Cổ Phòng.
     - *Mục tiêu*: `ALL_ENEMIES` (AoE toàn bộ đối thủ).
     - *Sát thương*: $D = (\text{Rank} \times 120 + \frac{\text{AuxGuCount}}{200}) \times \text{RankMult} \times \text{CoreMult}$.
     - *Hiệu ứng Turn-based*: **Turn / Initiative Pushback** — Đẩy lùi 30% điểm thanh lượt (`Initiative`) của toàn bộ đối thủ bị trúng đòn. Cooldown: 3 lượt.
   - **Slot 3 (Đại Trận Phòng Hộ - Fortress Barrier)**:
     - *Nguồn*: Cổ hạch tâm phòng ngự (hoặc kết cấu đại trận).
     - *Mục tiêu*: `SELF`.
     - *Tác dụng*: Tạo lớp Quang Thuẫn hấp thu sát thương bảo vệ Form HP:
       $$\text{Shield} = \text{MaxFormHP} \times 0.20 + (\text{Rank} \times 150) \times \text{HousePowerScale}$$
     - Cooldown: 3 lượt. Trừ 1 độ no Cổ phòng thủ.
   - **Slot 4 (Hồi Phục Sửa Chữa - Structural Mending)**:
     - *Nguồn*: Cổ hạch tâm trị liệu (hoặc cơ chế tự tu bổ).
     - *Mục tiêu*: `SELF`.
     - *Tác dụng*: Hồi phục trực tiếp Form HP cho Cổ Phòng:
       $$\text{HealFormHP} = \text{MaxFormHP} \times \left(0.15 + \frac{\text{AuxGuCount}}{100,000} \times 0.10\right)$$
       (Tương đương 15% - 25% Form HP tối đa; không hồi HP nhục thân). Cooldown: 4 lượt.
   - **Slot 5 (Thần Thông Hủy Diệt Tối Hậu - Ultimate Destruction)**:
     - *Nguồn*: Tuyệt chiêu liên hoàn từ TOÀN BỘ 2–5 Cổ hạch tâm + đại lượng Phàm Cổ.
     - *Mục tiêu*: `ALL_ENEMIES`.
     - *Cơ chế 2 Lượt (Charging Turns)*:
       - Lượt 1: Bắt đầu tụ khí đại trận (`ChargingTurnsLeft = 1`), nhận trạng thái tăng thêm 30% Damage Reduction trong lượt tụ lực.
       - Lượt 2: Phóng thích đại sát thương bùng nổ:
         $$\text{UltDmg} = \left(\text{Rank} \times 500 + \text{CoreCount} \times 300 + \text{AuxGuCount} \times 0.08\right) \times \text{HousePowerScale}$$
         Kèm thuộc tính **Bỏ qua 30% Giáp** (`IgnoreDefensePct = 0.30`).
       - Chi phí: $120 + \text{Rank} \times 20$ Mana. Trừ đồng loạt 2 độ no của TẤT CẢ các Cổ hạch tâm. Cooldown: 5 lượt.
   - **Slot 6 (Lệnh Rời Cổ Phòng - Disembark)**:
     - *Tác dụng*: Thu hồi tâm niệm, giải trừ dung nhập về nhục thân (`GuHouse.Active = false`), 0 Mana, 0 Cooldown.

2. **Công Thức Scaling Sức Mạnh Toàn Thể (House Power Scale)**:
   $$\text{HousePowerScale} = \text{RankMult} \times \text{AuxMult} \times \text{CoreMult}$$
   Trong đó:
   - $\text{RankMult}$: Rank 6 (1.0x), Rank 7 (1.5x), Rank 8 (2.2x), Rank 9 (3.5x).
   - $\text{AuxMult} = 1.0 + \frac{\text{AuxGuCount}}{10,000} \times 0.15$ (500 con: ~1.01x; 10,000 con: 1.15x; 50,000 con: 1.75x; 100,000 con: 2.50x).
   - $\text{CoreMult} = 1.0 + (\text{CoreCount} - 2) \times 0.10$ (2 Cổ: 1.0x; 3 Cổ: 1.1x; 5 Cổ: 1.3x).
   => Một Cổ Phòng Rank 8 với 50,000 Phàm Cổ và 5 Cổ hạch tâm đạt hệ số khuếch đại: $2.2 \times 1.75 \times 1.3 = 5.0x$ so với Cổ Phòng Rank 6 khởi điểm!

3. **Quy Tắc Tích Hợp Effect Composition (`gu_house_action.v1`) & Fallback**:
   - Khi người chơi ghép Trận Kích tùy biến từ Cổ hạch tâm trong `GuHousePanel.tsx`:
     - Trận Kích có `OpDamage` tự động ánh xạ vào Slot 1.
     - Trận Kích có `OpShield / OpBarrier` tự động ánh xạ vào Slot 3.
     - Trận Kích có `OpHeal` tự động ánh xạ vào Slot 4.
   - **Fallback Tự Động**: Nếu người chơi để trống hoặc Cổ hạch tâm không có effect tương ứng, hệ thống tự động sinh chiêu thức nội tại mặc định của Cổ Phòng theo công thức chuẩn ở trên, đảm bảo Cổ Phòng luôn luôn có đủ bộ 6 vai trò toàn diện khi xuất chiến.

4. **Tài Sản Kiểm Chứng Logic Prototype**:
   - Toàn bộ công thức scaling trực quan, thanh điều chỉnh Rank/Phàm Cổ, cơ chế Pushback và 4 kịch bản walkthrough đã được kiểm chứng tại:
     [`docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/02_turnbased_strike_matrix_demo.html`](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/immortal-gu-house-turnbased/prototypes/02_turnbased_strike_matrix_demo.html)
