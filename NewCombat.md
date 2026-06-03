# THIẾT KẾ CẬP NHẬT HỆ THỐNG CHIẾN ĐẤU & CHỈ SỐ NHÂN VẬT

Dựa trên thế giới quan của *Cổ Chân Nhân*, chúng ta sẽ mở rộng chiến trường thành lưới 3x3, tối ưu hóa hệ thống Khiên (Shield), cải tạo thể chất vĩnh viễn (Đạo Ngân), và làm rõ nhịp độ tu luyện.

## 1. CƠ CHẾ TỔ ĐỘI LƯỚI 3x3 & TẦM ĐÁNH (AoE)

Thay vì chỉ chia như trước, chiến trường giờ đây sẽ là hai lưới 3x3 đối diện nhau. Việc định vị vị trí của Player và Monster vô cùng quan trọng để né tránh AoE.

### 1.1 Cấu trúc Lưới (Grid Structure)
Mỗi phe (Team A và Team B) sở hữu một lưới 3x3 với tọa độ `(X, Y)`:
- **X (Cột / Hàng ngang):** 1 (Tiền tuyến), 2 (Trung tuyến), 3 (Hậu phương).
- **Y (Hàng / Vị trí dọc):** 1 (Trên), 2 (Giữa), 3 (Dưới).

### 1.2 Phân loại Tầm Đánh của Cổ Trùng (AoE Types)
Bổ sung trường `target_type` vào `GuRegistry`:
- **Đơn (Single):** Chỉ định chính xác 1 tọa độ `(X, Y)`. Trừ khi Cổ có đặc tính "Ám Sát" hoặc "Phi Hành", sát thương đơn mục tiêu bắt buộc phải đánh vào Tiền Tuyến (X=1) trước, nếu Tiền Tuyến chết mới được đánh Trung Tuyến (X=2).
- **Ngang (Row AoE):** Quét sạch toàn bộ mục tiêu có cùng tọa độ X. Ví dụ: *Hỏa Hải Song Giao Sát* quét Tiền Tuyến sẽ trúng `(1,1), (1,2), (1,3)`.
- **Dọc / Xuyên Thấu (Column AoE):** Xuyên thấu từ trước ra sau trên cùng một trục Y. Ví dụ: *Huyết Nguyệt Cổ* đánh vào Y=2 sẽ trúng `(1,2), (2,2), (3,2)`.
- **Chéo / Chữ X (Cross AoE):** Đánh vào 1 mục tiêu trung tâm và tản ra 4 góc chéo.
- **Toàn bản đồ (All Enemies):** Đánh trúng cả 9 ô của Team địch. Ví dụ: *Vạn Tinh Phi Huỳnh*.

## 2. THỨ TỰ LƯỢT ĐÁNH (SPEED-BASED COMBAT) & HỆ THỐNG KHIÊN (SHIELDS)

### 2.1 Thứ tự tốc độ (Speed Turn Order)
Không xử lý đồng thời lộn xộn nữa. Ở mỗi turn, tất cả hành động được đưa vào hàng đợi và **sắp xếp theo Tốc độ (Speed) từ cao tới thấp**.
- Nhân vật có Speed cao nhất ra chiêu trước. Nếu mục tiêu của nhân vật tốc độ chậm bị giết trước khi kịp ra chiêu, hành động của kẻ chậm sẽ bị hủy (hoặc chuyển mục tiêu ngẫu nhiên).

### 2.2 Hệ thống Khiên Hộ Thể (Virtual Shields)
Con người vốn yếu ớt, sinh lực (Base HP) của Player luôn cố định ở mức rất thấp (VD: 100 HP) bất kể Rank nào. Sinh tồn hoàn toàn dựa vào Cổ phòng ngự.
- Khi sử dụng Cổ phòng ngự (VD: *Thiên Bồng Cổ*, *Bạch Ngọc Cổ*), Player nhận được một lớp `VirtualShield` (VD: 1500 Shield).
- **Logic Trừ Sát Thương:** Damage nhận vào -> Trừ vào `VirtualShield` -> Nếu Shield vỡ, phần sát thương dư thừa mới trừ thẳng vào HP. HP = 0 là tử vong.

## 3. TỐI ƯU CỘNG CHỈ SỐ PASSIVE CỔ (CẢI TẠO THỂ CHẤT VĨNH VIỄN)

Trong nguyên tác, các Cổ Thụ Động (Passive Gu) như *Đồng Bì cổ, Thiết Cốt cổ, Hắc Thỉ cổ* không chỉ buff tạm thời mà dùng để **khắc Đạo Ngân vĩnh viễn** lên thân thể.

### 3.1 Cơ chế Tích lũy (Equip to Permanent)
- **Giai đoạn Trang bị (Equipped):** Khi mang *Đồng Bì Cổ*, Player được +50 Base Defense. 
- **Độ thuần thục (Integration Progress):** Thêm trường `integration_time` vào vật phẩm. Nếu Player trang bị con Cổ này sống sót qua X trận chiến (hoặc đủ Y giờ đời thực), Cổ trùng sẽ hoàn toàn hòa làm một với cơ thể.
- **Giai đoạn Vĩnh viễn (Permanent):** Con Cổ bị biến mất khỏi Inventory, nhưng Player nhận được vĩnh viễn `+50 Base Defense` (Cộng vào Body Mods).

### 3.2 Tác động đến các tính năng khác (Interconnectivity)
Chỉ số vĩnh viễn (Khí lực, Cốt cách) sẽ tác động tới toàn bộ Lục Giác Chiến Đấu:
- **Tương tác Cổ Trị Liệu:** Cổ *Tự Lực Cánh Sinh* hồi HP dựa trên tổng Sức Mạnh (Base Attack / Khí lực) của cơ thể. Cơ thể mang sức mạnh "Song Trư" (2 lợn rừng) sẽ hồi máu nhiều gấp đôi cơ thể người thường.
- **Tương tác Sức Chứa (Túi đồ):** Base Attack cao giúp Player mang vác được nhiều nguyên thạch và vật liệu hơn trên bản đồ thám hiểm.

## 4. THỜI GIAN CHỜ GIỮA CÁC LẦN TU LUYỆN (CULTIVATION COOLDOWNS)

**CÓ NÊN THÊM KHÔNG? -> RẤT NÊN THÊM!** 
Việc tu luyện (Trùng kích vách khiếu) liên tục mà không có trừng phạt sẽ biến game thành dạng "Clicker" cày cuốc vô não.

### 4.1 Cơ chế "Áp lực Vách Khiếu" (Aperture Stress)
Trong *Cổ Chân Nhân*, việc dùng chân nguyên liên tục xông phá vách khiếu sẽ làm vách khiếu tổn thương. Phải có thời gian để không khiếu tự ôn dưỡng và phục hồi.
- Bổ sung thanh **Aperture Stress (0% - 100%)**.
- Mỗi lần bấm nút "Trùng Kích Vách Khiếu", Stress tăng 20%.
- **Cooldown:** Stress sẽ tự động giảm theo thời gian thực (VD: -10% mỗi giờ).
- **Hình phạt (Penalty):** Nếu Player cố tình tu luyện khi Stress >= 80%, tỷ lệ đột phá giảm mạnh. Nếu Stress đạt 100%, Không khiếu bị nứt: **Mất vĩnh viễn 5% Max Mana** và rớt cấp!
- **Đường tắt (P2W / Rich Player):** Cổ sư có thể mua Cổ trị liệu Không Khiếu hoặc dùng *Tịnh Thủy Cổ* để lập tức xóa thanh Stress, tạo ra nguồn thu tiêu hao Nguyên thạch khổng lồ.

---

## 💻 DATA MODEL UPDATES (Golang)

```go
// 1. Cập nhật Model Tổ đội và Vị trí
type CombatEntity struct {
    EntityID     string  `json:"entity_id"` // Player ID or Monster ID
    EntityType   string  `json:"entity_type"` // "PLAYER" or "MONSTER"
    GridX        int     `json:"grid_x"` // 1 (Tiền), 2 (Trung), 3 (Hậu)
    GridY        int     `json:"grid_y"` // 1 (Trên), 2 (Giữa), 3 (Dưới)
    Speed        float64 `json:"speed"`
    HP           float64 `json:"hp"`
    VirtualShield float64 `json:"virtual_shield"` // Khiên ảo từ Cổ trùng
}

// 2. Cập nhật Chỉ số Vĩnh viễn & Tu luyện cho Player
type Player struct {
    // ...
    BaseHP             float64 `json:"base_hp"` // Cố định 100
    BodyMods           JSONB   `json:"body_mods"` // {"iron_bone": true, "pig_strength": 2}
    ApertureStress     float64 `json:"aperture_stress"` // 0 to 100%
    LastCultivateTime  time.Time `json:"last_cultivate_time"`
}

// 3. Cơ chế Action Queue trong Combat (Sắp xếp theo Tốc độ)
type CombatAction struct {
    InitiatorID string `json:"initiator_id"`
    TargetX     int    `json:"target_x"`
    TargetY     int    `json:"target_y"`
    AoEType     string `json:"aoe_type"` // "SINGLE", "ROW", "COL", "ALL"
    Speed       float64 `json:"speed"` // Tốc độ của người ra chiêu để Sort
}
```
```