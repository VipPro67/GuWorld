# Đặc Tả Kỹ Thuật: Hệ Thống Định Hướng Thôi Diễn Bí Phương (Targeted Recipe Conception)

## 1. Tổng Quan & Bối Cảnh
Trong thế giới *Cổ Chân Nhân*, việc thu thập bí phương luyện Cổ không chỉ phụ thuộc vào cơ duyên nhặt được trong di tích hay rương quặng (Mine Chest / Drop) mà các Cổ sư đại năng Trí Đạo và Luyện Đạo có thể vận dụng Chân Ý Cảnh Giới, Niệm Lực và đạo ngân để chủ động suy diễn thiên cơ, khai sáng ra tàn phương hoặc hoàn thiện bí phương cho Cổ trùng mong muốn.

Tính năng **"Khai Sáng Tàn Phương" (Targeted Recipe Conception)** cho phép người chơi chủ động chọn một Cổ trùng mục tiêu trong Cổ Phổ thế giới (`gu_dictionary`), đầu tư Niệm Lực, Nguyên Thạch và Cổ phôi/Vật liệu dẫn dắt để khai sáng tàn phương, giải quyết triệt để sự bất cập của cơ chế phụ thuộc 100% vào rơi rớt ngẫu nhiên.

---

## 2. Thuật Ngữ Miền Nghiệp Vụ (Domain Model)
- **Khai Sáng Tàn Phương (Targeted Recipe Conception)**: Hành vi vận dụng Niệm Lực, Nguyên Thạch kết hợp Chân Ý Cảnh Giới Tam Trụ (Trí Đạo, Luyện Đạo, Bản Lưu Phái Cổ Đích) và Cổ phôi/vật liệu dẫn dắt để khai mở tàn phương cho Cổ trùng mong muốn.
- **Trí Đạo Phản Phệ (Wisdom Backlash)**: Phản chấn nguy hiểm khi cưỡng cầu suy diễn Cổ trùng vượt quá Rank tu vi bản thân hoặc phàm nhân cưỡng suy Tiên Cổ. Khiến cạn kiệt Niệm Lực (`current_thought = 0`), mất 50% HP và Chân Nguyên, gây Chấn Thương Hồn Phách đối với Tiên Cổ.
- **Vật Dẫn / Tiêu Bản Nghiên Cứu (Catalyst)**: Cổ phôi sống cùng hệ trong Không Khiếu hoặc nguyên liệu đặc trưng cùng hệ dùng để mổ xẻ cấu trúc đạo ngân, gia tăng đáng kể tỉ lệ trúng đích Cổ mong muốn.

---

## 3. Mô Hình Toán Học & Công Thức Cân Bằng

### 3.1. Chuẩn Cảnh Giới Yêu Cầu theo Rank
- Rank 1-2: Cảnh giới 0 (`Ordinary`)
- Rank 3-4: Cảnh giới 1 (`Master` / Sư Phụ)
- Rank 5-6: Cảnh giới 2 (`Grandmaster` / Tông Sư)
- Rank 7-8: Cảnh giới 3 (`Great Grandmaster` / Đại Tông Sư)
- Rank 9: Cảnh giới 4 (`Supreme Grandmaster` / Vô Thượng Đại Tông Sư)

### 3.2. Công Thức Tỉ Lệ Trúng Đích (Accuracy Rate)
$$\text{diff} = \text{targetRealm} - \text{requiredRealm}$$
$$\text{realmMod} = \begin{cases} \text{diff} \times 0.08 & \text{khi } \text{diff} \ge 0 \\ \text{diff} \times 0.20 & \text{khi } \text{diff} < 0 \end{cases}$$
$$\text{wisdomMod} = \text{wisdomRealm} \times 0.04$$
$$\text{daoMod} = \min(0.08, \sqrt{\text{wisdomDaoMarks}} \times 0.004)$$
$$\text{Accuracy} = \text{clamp}(0.15, 0.88, 0.60 + \text{realmMod} + \text{wisdomMod} + \text{daoMod} + \text{catalystBonus})$$

*Ghi chú*: Áp trần 88% thiên cơ, luôn có 12% bất định dù đạt cảnh giới tối cao.

### 3.3. Thuật Toán Lệch Hướng (Deviation Candidate Selection)
Khi $\text{rand}() > \text{Accuracy}$:
- Lọc từ `gu_dictionary`: Trùng Lưu Phái chính (`cleanPath`), có `rank == targetRank` (trọng số 2) hoặc `rank == targetRank - 1` (trọng số 1), khác `targetGuID`.
- Loại bỏ các Cổ người chơi đã sở hữu bí phương hoàn chỉnh 100% trong `player_recipe_progress`.
- Bốc ngẫu nhiên có trọng số 1 Cổ ứng viên. Nếu người chơi đã mở khóa hết: Ý niệm quy nguyên, trao lại chính Cổ đích.

### 3.4. Cơ Chế Trí Đạo Phản Phệ (Backlash)
- Phàm nhân (Rank <= 5) cố suy tính Tiên Cổ (Rank >= 6): Chặn không cho thực hiện.
- Vượt cấp tu vi: $\text{diff} = \text{targetRank} - \text{playerRank} > 0 \implies P(\text{Backlash}) = 0.35 + \text{diff} \times 0.25$.
- Bằng cấp nhưng thiếu cảnh giới: $\text{targetRank} \le \text{playerRank} \land \text{wisdomRealm} == 0 \land \text{refineRealm} == 0 \land \text{targetRank} \ge 4 \implies P(\text{Backlash}) = 0.15$.
- Hậu quả:
  - `CurrentThought = 0`
  - `CurrentMana *= 0.5`
  - `CurrentHP = max(1.0, CurrentHP * 0.5)`
  - Tiên Cổ: Status `wisdom_backlash_soul_injury` + Cooldown 20 phút.

### 3.5. Độ Hoàn Thiện Khởi Điểm & Linh Quang Chớp Lóe
$$\text{Base} = 15.0 + \text{rand}(0.0, 15.0)$$
$$\text{Bonus} = \text{wisdomRealm} \times 2.0 + \text{refineRealm} \times 1.5 + \text{targetRealm} \times 1.0$$
$$\text{InitialRate} = \text{round}(\text{Base} + \text{Bonus}, 1) \in [15.0\%, 35.0\%]$$
- Xác suất nổ Linh Quang Chớp Lóe (100% Bí Phương):
  $$P(\text{Inspiration}) = \frac{0.02 \times \text{DeductionRealmMultiplier}}{\sqrt{\text{targetRank}}}$$

### 3.6. Bảng Chi Phí Cố Định
- **Niệm Lực (Current Thought)**: R1: 30, R2: 50, R3: 80, R4: 130, R5: 200, R6: 400, R7: 600, R8: 900, R9: 1500.
- **Nguyên Thạch (Primeval Stones)**: R1: 50, R2: 150, R3: 500, R4: 2,000, R5: 10,000.
- **Tiên Nguyên Thạch (Immortal Stones)**: R6: 100, R7: 500, R8: 2,500, R9: 10,000.

---

## 4. Quy Chuẩn Vật Dẫn / Tiêu Bản (Catalyst Rules)
- **Cổ Phôi sống**:
  - Không phải Cổ bản mệnh (`!is_vital`), không trang bị (`!equipped`), không khóa (`!is_locked`), chưa chết (`status != "DEAD"`).
  - Cùng Lưu phái, `gu.Rank <= targetRank`.
  - Mức bonus: Cùng Rank `+20%`, Kém 1 Rank `+16%`, Kém >= 2 Rank `+12%`.
  - Tiêu hao: Xóa khỏi Không Khiếu và xóa row trong `player_gus`.
- **Nguyên Liệu cùng hệ**:
  - Thuộc loại `Material`, cùng Lưu phái, `rank <= targetRank`.
  - Tiêu hao: Cùng Rank tốn 10 (R1-5) hoặc 5 (R6+); Kém 1 Rank tốn 25; Kém >= 2 Rank tốn 50.
  - Mức bonus: Cùng Rank `+12%`, Kém Rank `+8%`.
- **Nguyên tắc chi phí**: Vật dẫn chỉ tăng tỉ lệ chính xác, không giảm Niệm Lực hay Nguyên Thạch yêu cầu.

---

## 5. Thiết Kế API Backend & Database Contract

### 5.1. `GET /api/refine/conceive/candidates`
- Trả về danh sách Cổ trong `gu_dictionary` lọc theo `path`, `rank`, `search`, tự động loại trừ các Cổ người chơi đã sở hữu bí phương hoàn thiện 100%.
- Kèm tính toán trước: `required_thought`, `required_stones`, `estimated_accuracy`, `has_backlash_risk`.

### 5.2. `POST /api/refine/conceive`
- **Request Body**:
  ```json
  {
    "target_gu_id": "GU_SWORD_SHADOW_3",
    "target_rank": 3,
    "catalyst_gu_instance_id": "optional-gu-id",
    "catalyst_item_id": "optional-item-id"
  }
  ```
- **Response Body**:
  ```json
  {
    "status": "SUCCESS", // "SUCCESS" | "DEVIATED" | "BACKLASH"
    "is_deviated": false,
    "recipe_id": "recipe_GU_SWORD_SHADOW_3_3_1",
    "gu_id": "GU_SWORD_SHADOW_3",
    "gu_name": "Kiếm Ảnh Cổ",
    "rank": 3,
    "completion_rate": 28.5,
    "is_instant_100": false,
    "message": "Khai sáng thành công! Lĩnh ngộ Tàn Phương [Kiếm Ảnh Cổ] (Độ hoàn thiện: 28.5%).",
    "catalyst_consumed": "GU_SWORD_LIGHT_2",
    "thought_cost": 80,
    "stones_cost": 500,
    "backlash": null
  }
  ```

---

## 6. Giao Diện Người Dùng & Trải Nghiệm (Frontend UX)
1. **Tích hợp Tab trong `RefiningFurnace.tsx`**:
   - Tab "Khai Sáng Bí Phương" bên cạnh tab "Lò Luyện & Thôi Diễn".
   - Thanh chọn Lưu Phái và Rank, danh sách thẻ Cổ mục tiêu.
2. **Khu vực Đàn Suy Diễn**:
   - Thẻ hình ảnh Cổ đích, ô chọn Vật Dẫn từ Không Khiếu/Túi đồ.
   - Bảng Dự Toán Thiên Cơ: Progress bar tỉ lệ trúng đích, khả năng lệch nhánh, chi phí, cảnh báo phản phệ viền đỏ rực.
   - Nút hành động: `Khởi Niệm Thôi Diễn`.
3. **Modal Kết Quả Sống Động (`ConceiveResultModal`)**:
   - Thành công đúng Cổ đích (Hào quang vàng kim).
   - Lệch nhánh sang Cổ cùng hệ (Hào quang tím huyền bí).
   - Linh quang chớp lóe 100% (Pháo hoa rực rỡ).
   - Trí Đạo Phản Phệ (Rung lắc, hiệu ứng vỡ máu).
4. **Liên kết nhanh từ Cổ Phổ (`GuCodexModal`)**:
   - Nút `✨ Suy Tính Bí Phương Này` mở thẳng tab Khai Sáng và chọn sẵn Cổ đó.
