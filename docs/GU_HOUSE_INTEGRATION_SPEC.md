# Đặc Tả Kỹ Thuật: Cơ Chế Dung Nhập Tiên Cổ Phòng (Gu House Integration Spec)

## 1. Khái Niệm & Ranh Giới Kiến Trúc
Trong GuWorld, **Tiên Cổ Phòng (Immortal Gu House / Tiên Cổ Ốc)** là các cỗ máy chiến tranh, pháo đài lơ lửng, hoặc chiến xa di động được hợp thành từ nhiều Tiên Cổ hạch tâm (Rank 6+) và hàng trăm Phụ Cổ.
Khi Cổ Tiên kích hoạt lệnh **Dung Nhập (Boarding / Integration)**:
- Nhục thân Cổ Tiên ẩn vào bên trong không gian hạch tâm của Cổ Phòng.
- Quyền điều khiển di chuyển và thanh kỹ năng 1..6 được chuyển giao toàn bộ sang Tiên Cổ Phòng.
- Tiên Cổ Phòng trở thành thực thể che chắn chiến đấu chính thức trong Zone Sim 20Hz.

---

## 2. Các Quy Tắc Chiến Đấu Realtime (Settled Decisions)

### 2.1. Hitbox & Tốc Độ Di Chuyển
* **Bán kính va chạm (`Radius`)**: Phụ thuộc vào loại hình thái kiến trúc của Tiên Cổ Phòng (định nghĩa trong cấu hình Cổ Phòng):
  - *Chiến Xa Cơ Động* (Mobile Chariot - ví dụ: *Phi Hồ Chu*, *Chiến Xa Báo Bão*): Bán kính $R = 2.0m$.
  - *Lâu Các / Pháo Đài Kiên Cố* (Fortress / Pavilion - ví dụ: *Vạn Thọ Lâu*, *Hắc Lao*, *Tập Tiên Các*): Bán kính $R = 4.0m$.
* **Tốc độ di chuyển (`Speed`)**:
  - Tốc độ cơ sở bị giảm 40% so với tốc độ chạy bộ của Cổ Tiên do khối lượng kiến trúc đồ sộ.
  - Bổ sung tốc độ dựa trên số lượng Phụ Cổ thuộc **Phong Đạo / Vũ Đạo**:
    $$\text{BonusSpeed} = \text{WindAuxGuCount} \times 0.02$$

### 2.2. Phân Bổ Sát Thương & Cơ Chế Vỡ Phòng (Hull Breach)
* **Bảo hộ tuyệt đối 100%**:
  - Toàn bộ sát thương nhận vào từ mọi nguồn (trừ sát thương linh hồn chuyên biệt) đều được hấp thụ 100% bởi thanh máu của Tiên Cổ Phòng (`CurrentHouseHP`).
  - Cổ Tiên bên trong không bị mất máu và không bị dính các hiệu ứng debuff thông thường (`ImmuneDebuffs: true`).
* **Vỡ Phòng (Destruction / Breach)**:
  - Khi `CurrentHouseHP` tụt về 0:
    1. Tiên Cổ Phòng sụp đổ và tự động giải thể khỏi chiến trường.
    2. Cổ Tiên bị hất văng ra ngoài, nhận 10% sát thương nổ vỡ và dính debuff Choáng (`STUN` 1.5s) cùng Kiệt Sức (`EXHAUSTION` 5s).
    3. Tiên Cổ Phòng chuyển sang trạng thái `DAMAGED` trong CSDL, không thể tiếp tục dung nhập cho đến khi được bảo dưỡng hoặc sửa chữa.
* **Sửa chữa sau chiến đấu**:
  - `CurrentHouseHP` không tự hồi phục tức thì sau trận.
  - Người chơi tiêu hao Tiên Nguyên Thạch / Tiên Tài tại giao diện Tiên Cổ Ốc hoặc đưa vào Tiên Khiếu để hồi phục độ bền dần theo thời gian.

### 2.3. Bộ Trận Kích Toàn Diện (Full House Strike Set)
Khi dung nhập, thanh kỹ năng 1..6 của người chơi hoán đổi sang hệ thống Trận Kích đầy đủ 6 vai trò:
1. **Slot 1 - Pháo Kích Chủ Lực (Primary Cannon)**: Đòn đánh tầm xa cơ bản của Cổ Phòng (Direct Damage, CD 1.0s - 1.5s).
2. **Slot 2 - Xung Kích / Dược Thiên (Ramming / Warp Leap)**: Lao về phía trước hoặc dịch chuyển cự ly ngắn, gây sát thương va chạm và đẩy lùi kẻ cản đường.
3. **Slot 3 - Đại Trận Hộ Thể (Bastion Barrier)**: Tạo lớp khiên hộ thể ảo hấp thụ thêm sát thương trong 6s.
4. **Slot 4 - Tự Động Sửa Chữa (Emergency Field Repair)**: Tiêu hao lượng lớn Tiên Nguyên để hồi phục ngay một phần `CurrentHouseHP`.
5. **Slot 5 - Tuyệt Kỹ Hủy Diệt (Annihilation Strike)**: Sát chiêu tối hậu diện rộng của Tiên Cổ Phòng (Sát thương cực lớn, CD 20s - 30s).
6. **Slot 6 - Rời Khỏi Cổ Phòng (Disembark)**: Chủ động mở cửa bước ra ngoài, bảo toàn trạng thái Cổ Phòng và trở lại dạng Cổ Tiên bình thường (CD 1.0s, Mode Self).

### 2.4. Tiêu Hao Tiên Nguyên & Thang Đo Cảnh Giới (Realm Scaling Upkeep)
* Tiêu hao Tiên Nguyên liên tục theo nhịp 20Hz được tính theo công thức:
  $$\text{UpkeepManaSec} = \text{BaseUpkeep} \times (\text{CoreGuCount}) \times 2^{(\text{HouseRank} - \text{PlayerRank})}$$
* **Ý nghĩa cân bằng**:
  - *Cổ Tiên Rank 9 điều khiển Tiên Cổ Phòng Rank 6*:
    $$2^{(6 - 9)} = 2^{-3} = \frac{1}{8}$$
    Tiêu hao giảm sâu tới 87.5%, Cổ Tiên có thể duy trì Cổ Phòng cấp thấp gần như vô tận.
  - *Cổ Tiên Rank 6 gồng gánh Tiên Cổ Phòng Rank 7*:
    $$2^{(7 - 6)} = 2^1 = 2$$
    Tiêu hao tăng gấp đôi, chỉ có thể duy trì trong thời gian ngắn lúc nguy cấp.
* **Cạn Tiên Nguyên**:
  - Nếu Tiên Nguyên của người chơi cạn về 0, Tiên Cổ Phòng tự động cưỡng chế giải trừ, Cổ Tiên bước ra ngoài trong trạng thái cạn kiệt.
