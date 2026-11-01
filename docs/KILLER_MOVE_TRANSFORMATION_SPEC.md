# Đặc Tả Sát Chiêu Biến Thân & Cấu Hình Mẫu (Killer Move Transformation Spec)

## 1. Bản Chất Kiến Trúc
Trong GuWorld, **Sát Chiêu Biến Thân (Metamorphosis Killer Moves)** là các sát chiêu thuộc **Biến Hóa Đạo (Transformation Path)** hoặc tổ hợp lai (Lai Lực Đạo, Băng Tuyết Đạo, Biến Hóa Đạo). Khi kích hoạt thành công:
1. Nhân vật bước vào giai đoạn niệm chú (Windup). Nếu dính Hard CC (Choáng/Đóng Băng) thì sát chiêu bị ngắt.
2. Khi hoàn tất niệm chú, Server Zone Sim 20Hz thực hiện:
   - Kích hoạt `TransformationState` tương ứng với `FormID` được định nghĩa trong `TransformationForm.csv`.
   - Scale Hitbox bán kính va chạm $R_{\text{eff}} = R_{\text{base}} \times \text{Mult}$.
   - Cấp một lớp máu riêng `FormHP` và chỉ số `DamageReduction`.
   - Hoán đổi toàn bộ thanh kỹ năng 1..6 sang bộ kỹ năng thú (`SwapSkillPalette`).
   - Tiêu hao duy trì (`UpkeepCost`) mỗi tick 50ms cho đến khi chủ động giải trừ hoặc cạn Tiên Nguyên.

---

## 2. Danh Mục Sát Chiêu Biến Thân Mẫu

### 2.1. Lục Tí Tuyết Băng Ma Vương (Six-Arm Ice-Snow Heavenly Demon)
* **Lưu phái**: Lực Đạo (Chủ đạo) + Biến Hóa Đạo (Phụ trợ) + Băng Tuyết Đạo (Môi giới).
* **Phẩm cấp**: Tứ Chuyển Đỉnh Phong / Ngũ Chuyển Ma Đạo Sát Chiêu.
* **Hình thái kích hoạt**: `FORM_SIX_ARM_ICE_DEMON`
  - **Kích thước & Hitbox**: Scale 1.8x, Bán kính va chạm 1.6x.
  - **Lớp máu Form HP**: 400 Form HP (Hấp thụ sát thương trước máu thực).
  - **Phòng ngự**: +40 Giáp, -30% Sát thương nhận vào.
  - **Tiêu hao duy trì**: 18 Mana / giây.
  - **Thời gian niệm**: 2.0 giây (40 ticks).
* **Tổ hợp Cổ trùng cấu thành**:
  - *Hạch tâm*: Ngũ Chuyển Bách Thú Biến Cổ (`GU_TRANSFORM_CHIMERA_5`) hoặc Tứ Chuyển Cuồng Lang Bì Cổ (`GU_WOLF_SKIN_4`).
  - *Phụ Cổ Lực Đạo*: Tứ Giai Hùng Lực Cổ, Tứ Giai Tượng Lực Cổ, Tứ Giai Ngưu Lực Cổ.
  - *Phụ Cổ Băng Tuyết*: Băng Cơ Cổ, Tuyết Băng Cổ, Hàn Băng Cổ.
* **Bộ 6 Kỹ Năng Hình Thái (Form Skills Palette)**:
  1. **Lục Tí Bạo Liệp (Slot 1)**: 6 cánh tay ma thần tung quyền liên hoàn cận chiến dồn dập (Sát thương: 90, CD: 1.25s).
  2. **Hàn Băng Thần Chưởng (Slot 2)**: Chưởng khí hàn băng tầm xa làm chậm 40% kẻ địch (Sát thương: 120, CD: 4s).
  3. **Ma Vương Chấn Địa (Slot 3)**: Nhảy cao đập 6 tay xuống đất, gây chấn động nứt vỡ địa hình, Choáng 2s (Sát thương: 180, CD: 7s).
  4. **Tuyết Băng Cuồng Nộ (Slot 4)**: Gầm thét tạo cơn lốc tuyết bao bọc ma thân, phản xạ 20% sát thương và đóng băng kẻ đánh cận chiến (CD: 8s).
  5. **Lục Tí Toái Không Quyền (Slot 5 - Đại Chiêu)**: Dồn toàn bộ băng kình và cự lực vào 6 tay đấm xuyên thấu mục tiêu (Sát thương: 350, CD: 18s).
  6. **Giải Trừ Ma Thân (Slot 6)**: Hoàn nguyên trở lại dạng người phàm (0 mana, CD: 0.5s).

---

### 2.2. Cuồng Man Thú Thần Biến (Reckless Savage Beast Divinity)
* **Lưu phái**: Biến Hóa Đạo Tiên Tôn Truyền Thừa (Thuần Biến Hóa Đạo + Lực Đạo Thần Thông).
* **Phẩm cấp**: Bát Chuyển / Cửu Chuyển Tiên Cổ Sát Chiêu.
* **Hình thái kích hoạt**: `FORM_RECKLESS_SAVAGE`
  - **Kích thước & Hitbox**: Scale 2.2x, Bán kính va chạm 2.0x (Dị thú cự thể).
  - **Lớp máu Form HP**: 1,200 Form HP.
  - **Phòng ngự**: +80 Giáp, -40% Sát thương nhận vào, Miễn nhiễm khống chế cứng thường.
  - **Tiêu hao duy trì**: 30 Tiên Nguyên / giây.
  - **Thời gian niệm**: 3.0 giây (60 ticks).
* **Tổ hợp Tiên Cổ cấu thành**:
  - *Hạch tâm*: Cửu Chuyển Biến Hóa Khởi Nguyên Tiên Cổ (`GU_TRANSFORM_ANCESTOR_I`).
  - *Phụ Tiên Cổ*: Thôn Thiên Tiên Cổ, Cương Lực Tiên Cổ, Vạn Tượng Tiên Cổ.
* **Bộ 6 Kỹ Năng Hình Thái (Form Skills Palette)**:
  1. **Thôn Thiên Thú Phệ (Slot 1)**: Há miệng cắn nuốt hút sinh mệnh và chân nguyên mục tiêu (Sát thương: 180, Hồi 50 HP, CD: 1.5s).
  2. **Hồng Hoang Bạo Dược (Slot 2)**: Nhảy bổ vượt địa hình 12m, đè bẹp mọi kẻ địch trên đường rơi (Sát thương: 220, CD: 5s).
  3. **Vạn Thú Hóa Cực (Slot 3)**: Kích phát thiên phú biến hóa của vạn thú, tự thanh tẩy mọi debuff và tăng 50% tốc chạy (CD: 10s).
  4. **Chấn Nhiếp Bát Hoang (Slot 4)**: Tiếng gầm thái cổ áp chế linh hồn toàn bộ thực thể trong 15m, gây Câm Lặng 3s (CD: 9s).
  5. **Thái Cổ Tuyệt Diệt Kích (Slot 5 - Thần Thông)**: Tụ lực đòn hủy diệt quét sạch bình nguyên (Sát thương: 600 AoE, CD: 25s).
  6. **Giải Trừ Thú Thần (Slot 6)**: Thu liễm thú thân về tiên thể (0 mana, CD: 0.5s).

---

## 3. Cấu Trúc Bảng Từ Điển `transformation_form_dictionary`
Được ánh xạ tự động từ file CSV `backend/data/TransformationForm.csv`:

| Tên Cột | Kiểu Dữ Liệu | Ý Nghĩa |
| :--- | :--- | :--- |
| `key` | `VARCHAR(80) PK` | Mã định danh hình thái (`FORM_FLAME_BEAR`, `FORM_WOLF_PARTIAL`...) |
| `name` | `VARCHAR(120)` | Tên hiển thị tiếng Việt |
| `form_kind` | `VARCHAR(30)` | `FULL` (Toàn phần), `PARTIAL` (Bộ phận), `GU_HOUSE` (Cổ phòng) |
| `model_scale` | `DOUBLE PRECISION` | Tỉ lệ phóng to hình ảnh / avatar trên client canvas |
| `collision_radius_mult` | `DOUBLE PRECISION` | Tỉ lệ phóng to hitbox bán kính va chạm trong zone sim 20Hz |
| `max_form_hp` | `DOUBLE PRECISION` | Lớp máu form riêng biệt |
| `form_defense` | `DOUBLE PRECISION` | Giáp cộng thêm khi ở hình thái |
| `damage_reduction` | `DOUBLE PRECISION` | Tỉ lệ giảm sát thương trực tiếp (0.0 đến 1.0) |
| `mana_upkeep_sec` | `DOUBLE PRECISION` | Lượng Mana/Tiên nguyên tiêu hao mỗi giây |
| `windup_sec` | `DOUBLE PRECISION` | Thời gian niệm chú kích hoạt |
| `skills_json` | `JSONB` | Danh sách 6 skills dạng thú hoặc overlay config |
| `description` | `TEXT` | Mô tả truyền thừa và điển tích |
