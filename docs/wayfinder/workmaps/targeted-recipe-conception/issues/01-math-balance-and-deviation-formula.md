Type: research
Status: resolved

## Question

Xác định mô hình toán học chi tiết cho hệ thống Khai Sáng Tàn Phương:
1. Công thức tính Tỉ lệ Trúng Đích (Accuracy Rate) dựa trên Cảnh giới Lưu Phái Cổ Đích (`targetRealm`), Trí Đạo (`wisdomRealm`), và Đạo Ngân (`wisdomDaoMarks`).
2. Thuật toán chọn Cổ thay thế khi bị Lệch Hướng (Deviation): tiêu chí lọc ứng viên cùng Lưu Phái (`path`), cùng Rank hoặc Rank - 1 từ `gu_dictionary` mà người chơi chưa mở khóa bí phương.
3. Ngưỡng kích hoạt và độ nặng của Trí Đạo Phản Phệ: điều kiện chênh lệch Rank giữa Cổ đích và tu vi/cảnh giới, tỉ lệ mất máu, chân nguyên và cơ chế khóa hồi chiêu chấn thương hồn phách đối với Tiên Cổ.
4. Công thức tính Độ Hoàn Thiện Ban Đầu của Tàn Phương (15% - 35%) và xác suất nổ Linh Quang Chớp Lóe (hoàn thiện 100%).

## Answer

### 1. Công thức Tỉ lệ Trúng Đích (Accuracy Rate)
- **Chuẩn Cảnh Giới Yêu Cầu theo Rank**:
  - Rank 1-2: Cảnh giới 0 (Ordinary)
  - Rank 3-4: Cảnh giới 1 (Master / Sư Phụ)
  - Rank 5-6: Cảnh giới 2 (Grandmaster / Tông Sư)
  - Rank 7-8: Cảnh giới 3 (Great Grandmaster / Đại Tông Sư)
  - Rank 9: Cảnh giới 4 (Supreme Grandmaster / Vô Thượng Đại Tông Sư)
- **Công thức**:
  $$\text{diff} = \text{targetRealm} - \text{requiredRealm}$$
  $$\text{realmMod} = \begin{cases} \text{diff} \times 0.08 & \text{khi } \text{diff} \ge 0 \\ \text{diff} \times 0.20 & \text{khi } \text{diff} < 0 \end{cases}$$
  $$\text{wisdomMod} = \text{wisdomRealm} \times 0.04$$
  $$\text{daoMod} = \min(0.08, \sqrt{\text{wisdomDaoMarks}} \times 0.004)$$
  $$\text{Accuracy} = \text{clamp}(0.15, 0.88, 0.60 + \text{realmMod} + \text{wisdomMod} + \text{daoMod} + \text{catalystBonus})$$
- **Kết quả điều chỉnh thực tế**:
  - Đủ chuẩn cảnh giới: 65% - 72% (không có vật dẫn), lên 80% - 88% (có vật dẫn).
  - Cảnh giới thấp (chưa đạt chuẩn): 25% - 35% (không có vật dẫn), lên 40% - 50% (có vật dẫn).
  - Trần tỉ lệ tối đa: 88% (thay vì 95%, đảm bảo luôn có 12% yếu tố bất định thiên cơ).

### 2. Thuật toán Lệch Hướng (Deviation Candidate Selection)
Khi $\text{rand}() > \text{Accuracy}$:
- Truy vấn từ `gu_dictionary` kết hợp `refining_recipes`:
  - Lọc theo Lưu Phái: trùng với Lưu phái chính của Cổ đích (`cleanPath`).
  - Lọc theo Rank: `rank = targetRank` (trọng số 2) hoặc `rank = targetRank - 1` (trọng số 1).
  - Loại trừ chính Cổ đích (`id != targetGuID`).
  - Loại trừ các Cổ người chơi đã sở hữu bí phương hoàn thiện 100% trong `player_recipe_progress`.
- Bốc ngẫu nhiên có trọng số 1 Cổ trong danh sách ứng viên.
- Nếu người chơi đã mở khóa hết các Cổ cùng hệ trong tầm Rank đó: Ý niệm quy nguyên, hệ thống trao lại chính Cổ đích kèm thông điệp thông suốt thiên cơ.

### 3. Trí Đạo Phản Phệ (Wisdom Backlash Rules)
- **Điều kiện kích hoạt**:
  - Phàm nhân (Rank <= 5) cố suy tính Tiên Cổ (Rank >= 6): Chặn ngay từ đầu (Out of scope).
  - Vượt cấp tu vi: $\text{diff} = \text{targetRank} - \text{playerRank} > 0 \implies P(\text{Backlash}) = 0.35 + \text{diff} \times 0.25$ (Ví dụ lệch 1 Rank = 60%, lệch 2 Rank = 85%).
  - Bằng cấp nhưng thiếu cảnh giới: $\text{targetRank} \le \text{playerRank}$ nhưng $\text{wisdomRealm} == 0 \land \text{refineRealm} == 0 \land \text{targetRank} \ge 4 \implies P(\text{Backlash}) = 0.15$.
  - Trường hợp còn lại: $P(\text{Backlash}) = 0.0$ (An toàn).
- **Hậu quả phản phệ**:
  - `CurrentThought = 0` (hút cạn niệm lực).
  - `CurrentMana = CurrentMana * 0.5` (mất 50% chân nguyên).
  - `CurrentHP = max(1.0, CurrentHP * 0.5)` (thổ huyết, mất 50% máu).
  - Với Tiên Cổ: gắn trạng thái Chấn Thương Hồn Phách (`wisdom_backlash_soul_injury`) và đặt hồi chiêu không thể khai sáng trong 20 phút.
  - Lần suy tính này thất bại hoàn toàn, không tạo ra tàn phương.

### 4. Độ Hoàn Thiện Khởi Điểm & Linh Quang Chớp Lóe
- **Độ hoàn thiện khởi điểm của Tàn Phương**:
  $$\text{Base} = 15.0 + \text{rand}(0.0, 15.0)$$
  $$\text{Bonus} = \text{wisdomRealm} \times 2.0 + \text{refineRealm} \times 1.5 + \text{targetRealm} \times 1.0$$
  $$\text{InitialRate} = \text{round}(\text{Base} + \text{Bonus}, 1) \in [15.0\%, 35.0\%]$$
- **Linh Quang Chớp Lóe (Flash of Inspiration)**:
  $$P(\text{Inspiration}) = \frac{0.02 \times \text{DeductionRealmMultiplier}}{\sqrt{\text{targetRank}}}$$
  Khi kích hoạt, thiết lập `InitialRate = 100.0%` ngay lập tức!
- **Hỗ trợ Cổ chuyên dụng**: Kế thừa kiểm tra cơ chế `DEDUCTION_INSTANT_FINISH` từ các Cổ trang bị như Định Trí Cổ.

### 5. Chi Phí Tài Nguyên Cố Định (Fixed Costs)
- **Niệm Lực (Current Thought)**:
  - Bảng tra theo Rank: R1: 30, R2: 50, R3: 80, R4: 130, R5: 200, R6: 400, R7: 600, R8: 900, R9: 1500.
- **Nguyên Thạch (Primeval Stones)**:
  - R1: 50, R2: 150, R3: 500, R4: 2,000, R5: 10,000.
- **Tiên Nguyên Thạch (Immortal Stones)** (Rank >= 6):
  - R6: 100, R7: 500, R8: 2,500, R9: 10,000.

