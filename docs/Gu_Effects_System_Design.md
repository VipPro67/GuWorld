# Tài Liệu Đặc Tả Hệ Thống: Cơ Chế Vận Hành Hiệu Ứng Cổ Trùng Thượng Tầng
*Dự án Phát triển Web Game Chiến Thuật Bất Đồng Bộ Cổ Chân Nhân (Go Backend + Next.js Frontend)*

Tài liệu này thuyết minh chi tiết cách thức vận hành, trigger logic và cấu trúc tính toán số học của các Cổ trùng đặc biệt đã được cập nhật vào cơ sở dữ liệu `Gu_Effects_Update.xlsx`. Hệ thống bám sát các thiết lập cốt lõi của đạo hữu: xử lý trạng thái trận đấu bất đồng bộ qua **Redis Cache**, kết nối thời gian thực qua **WebSockets**, phân tách bố cục chiến đấu **2 hàng (Frontline/Backline)** và lớp **Chân Nguyên Giáp (Shield)**.

---

## 1. Kiến Trúc Vận Hành Engine (Event-Driven Architecture)

Để đảm bảo hiệu năng tối cao và chống gian lận tốc độ (hack speed), game áp dụng tư duy **Kiến trúc hướng sự kiện (Event-Driven)**:
- **Server-Side Calculation:** Toàn bộ hiệu ứng được tính toán hoàn toàn dưới Backend Go. Mỗi lượt đấu (Tick) được nạp vào bộ nhớ đệm Redis để thực hiện các phép so sánh logic If-Then cứng.
- **Client Render (Next.js):** Frontend không tự chạy logic game. Next.js nhận payload trạng thái qua kênh WebSocket (Event push) và chỉ thực thi render animation hoặc thanh máu/giáp dựa trên State đồng bộ.
- **JSONB Parsing:** Cột `effects` trong database được map trực tiếp vào các Structs cụ thể của Go giúp tối ưu hóa luồng đọc dữ liệu, tránh panic loại dữ liệu (type assertion panic).

---

## 2. Cơ Chế Vận Hành Trận Đấu (Combat Loop Specs)

### Hàng Tiền Tuyến (Frontline) & Hậu Phương (Backline)
Trận đấu chia làm hai dải vị trí đối xứng. Sát thương mặc định sẽ ưu tiên quét từ hàng Tiền Tuyến vào sâu bên trong. 
- Các lưu phái cận chiến như **Lực đạo**, **Biến hóa đạo** nhận thêm thuộc tính phòng ngự thụ động khi đứng Tiền tuyến.
- Các lưu phái điều khiển, bói toán như **Trí tuệ đạo**, **Hồn đạo** hoặc triệu hồi quân đoàn như **Nô dịch đạo** sẽ đứng ở Hậu phương để tối ưu thời gian tụ lực sát chiêu.

### Chân Nguyên Giáp (Shield Barrier)
- Mọi đòn đánh sát thương (Damage) thông thường từ Sát chiêu đối phương đều phải trừ vào lớp Giáp hộ thể này trước. Khi Giáp hộ thể giảm về 0, sát thương mới trừ trực tiếp vào HP cốt lõi của Cổ sư để tránh hiện tượng "Cổ sư giấy" bay màu lập tức.

---

## 3. Phân Tích Logic Chi Tiết Từng Loại Cổ Trùng Đặc Biệt

### 1. Xuân Thu Thiền (`GU_SPRING_AUTUMN_CICADA`)
* **Combat Logic (Trigger khi chết):** Server cài đặt một bộ hook lắng nghe sự kiện khi lượng máu của Cổ sư bằng 0 (`Player.HP <= 0`). Hệ thống sẽ thực hiện một lệnh kiểm tra ngẫu nhiên (Random Gamble) với xác suất đúng 15%.
* **Kết quả thành công:** State trận đấu hiện tại trong Redis bị xóa bỏ. Server khởi tạo lại trận đấu về `Lượt 1` với đầy đủ chỉ số gốc của cả 2 bên. Gửi một tín hiệu WebSocket đặc biệt `EVENT_REWIND_MATCH` về Next.js để chạy hiệu ứng lật ngược dòng sông thời gian.
* **Kết quả thất bại:** Hook bỏ qua, nhân vật tử vong chính thức, nhận phạt nặng chỉ số hoặc kích hoạt trạng thái Tiên Cương (Zombie).

### 2. Định Tiên Du (`GU_FIXED_IMMORTAL_TRAVEL`)
* **Cơ chế Bản đồ (Exploration Map Grid):** Lưới thế giới năm đại vực được cấu trúc theo hệ tọa độ Oxy (100x100). Bình thường, việc di chuyển qua các ô cần tiêu hao Thể lực và không khiếu bị chặn bởi Giới bích. Khi kích hoạt Định Tiên Du ngoài trận, API kiểm tra điều kiện di chuyển sẽ bỏ qua hoàn toàn hàm tính khoảng cách (`CalculateDistance`) và hàm kiểm tra vách ngăn (`CheckBoundary`), cho phép ghi đè trực tiếp tọa độ mới của người chơi vào DB vĩnh viễn với Stamina cost = 0.
* **Cơ chế Trong trận:** Tự động gán buff ẩn `FIRST_TURN_EVADE = true`. Khi nhận đòn tấn công đầu tiên ở Tick 1 từ đối phương, sát thương trả về bằng 0 và hủy trạng thái buff.

### 3. Trí Tuệ Cổ (`GU_WISDOM_GU`)
* **Cơ chế Tu Luyện & Hợp Luyện (Refinement):** Khi người chơi đưa Sát chiêu hoặc Cổ trùng vào lò Thôi Diễn (Deduction Engine), hệ thống Backend Go sẽ đọc thuộc tính `refinement_speed_boost: 0.50` để cắt giảm thời gian hồi job thụ động đi một nửa trong DB, đồng thời tăng biến số tỷ lệ thành công thêm 30%.
* **Cơ chế Trong trận:** Mỗi khi kết thúc một lượt đấu (Tick), lượng Chân Nguyên/Tiên Nguyên hồi phục của người chơi được nhân trực tiếp với hệ số `3.0`.

### 4. Thái Độ Cổ (`GU_ATTITUDE_GU`)
* **Tâm lý chiến PvP (Masking Payload):** Khi endpoint đối kháng PvP được khởi tạo, thay vì gửi toàn bộ payload cấu hình thực tế của người chơi cho đối thủ, Server sẽ chạy hàm ngụy tạo dữ liệu. Thái Độ Cổ sẽ hoán đổi mảng JSON `GuInventory` thực tế thành một cấu trúc rỗng hoặc giả lập các Cổ trùng khắc hệ cấp thấp để đánh lừa hệ thống Trigger auto-combat của đối phương, khiến đối thủ sắp xếp sai vị trí Tiền tuyến/Hậu phương.

### 5. Huyết Sọ Cổ (`GU_BLOOD_SKULL`)
* **Cải Tạo Chỉ Số Gốc (Out-of-combat Upgrade):** Thuộc tính Nhân đạo/Huyết đạo cực đoan. Khi người chơi kích hoạt tính năng hiến tế các NPC đệ tử hoặc đồng tộc trong bang phái, Server tiến hành xóa vĩnh viễn ID của các NPC đó khỏi bảng `characters/npcs` và cộng dồn lũy kế điểm phần trăm vào trường `Aptitude` của nhân vật chính (Max là 99% - Tư chất Giáp tối đỉnh). Đi kèm tăng chỉ số ẩn `Stress_Level` hoặc `Sat_Khi`.

### 6. Tửu Trùng (`GU_LIQUOR_WORM`)
* **Tinh Luyện Chân Nguyên:** Khi bước vào trận đấu, Server Go không thay đổi giới hạn dung lượng không khiếu của người chơi mà áp dụng một hệ số nhân sát thương ẩn `Effective_SubRank = Player.SubRank + 1`. Tất cả các Sát Chiêu sử dụng phàm cổ từ Rank 1 đến Rank 5 đi kèm sẽ tự động được cộng thêm 20% uy lực sát thương đầu ra nhờ chất lượng chân nguyên đậm đặc.

### 7. Cổ Gan Dạ (`GU_GUTS_GU`)
* **Trị Liệu Linh Hồn Trả Giá Bằng Lượt:** Khi Cổ sư kích hoạt Cổ Gan Dạ trong trận đấu để hồi phục chỉ số Linh Hồn (`Soul Foundation`), Server Go lập tức khóa quyền tấn công của người chơi ở vòng đó (`Turn_Cost = 1`). Bù lại, lượng máu tâm linh được hồi phục hoàn toàn 100%, bảo vệ người chơi khỏi các sát chiêu hủy diệt linh hồn của Hồn đạo.

### 8. Tiên Thai Chí Tôn (`GU_SOVEREIGN_FETUS`)
* **Xóa Bỏ Cơ Chế Bài Xích Đạo Ngân (Endgame Passives):** Đây là thuộc tính tối cao. Công thức tính toán sát thương chuẩn của game luôn có hàm kiểm tra xung đột đạo ngân:
    $$\text{Sát Thương Nhận Vào} = \text{Sát Thương Gốc} \times (1 + \text{Đạo Ngân Khắc Hệ}) \times (1 - \text{Xung Đột Đạo Ngân Phòng Thủ})$$
    Đối với cơ thể Tiên Thai Chí Tôn, thuộc tính `ignore_dao_mark_conflict` được set thành `True`. Hệ số xung đột luôn bằng `0`, cho phép người chơi vừa được tăng 100% sát thương tất cả các hệ (Hỏa, Thủy, Kiếm, Lực...) mà không bị giảm trừ bất kỳ chỉ số phòng thủ thụ động nào trên cơ thể.

### 9. Cổ Tình Yêu (`GU_LOVE_GU`)
* **Hệ Thống Trạng Thái Hỗn Loạn (Chaos Gamble Logic):** Khi kích hoạt Sát chiêu chứa Cổ Tình Yêu, Server chạy một mảng hàm ngẫu nhiên đôi (Double-Random Arrays).
    - Mảng 1 (Hiệu ứng phát ra): Ngẫu nhiên chọn một trong ba trạng thái cực hạn: Tấn công chí mạng tăng 500% / Miễn nhiễm mọi sát thương trong 1 lượt (Bất tử) / Giảm trực tiếp 90% lượng HP hiện tại của kẻ địch.
    - Mảng 2 (Cái giá phải trả): Ngẫu nhiên trừ trực tiếp vào cơ sở dữ liệu nhân vật: Trừ 10% Tư chất gốc / Trừ 5 năm tuổi thọ vĩnh viễn / Trừ 50% điểm Vận khí may mắn hiện tại.

### 10. Cổ Thiên Cơ (`GU_HEAVEN_SECRET`)
* **Hack Sương Mù Bản Đồ (Map Utility Tool):** Gửi payload tọa độ dạng bán kính xung quanh ô người chơi đang đứng. Phá bỏ trạng thái sương mù (`Fog of War = false`) trong phạm vi 10 ô trên Next.js. Đồng thời tiết lộ chính xác dải tỷ lệ rơi vật phẩm (Loot-rate JSON) của quái vật hoang thú tại tọa độ đó, giúp Cổ Sư toan tính lộ trình thám hiểm tối ưu nhất.