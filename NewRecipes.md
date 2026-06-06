Chào bạn, ý tưởng đưa **Trí Đạo (Wisdom Path)** vào làm cốt lõi cho hệ thống công thức chế tạo là một nước đi cực kỳ xuất sắc. Trong thế giới *Cổ Chân Nhân*, Luyện đạo và Trí đạo luôn song hành. Trí đạo dùng để suy tính Tàn phương, Luyện đạo dùng để biến Tàn phương thành hiện thực.

Dưới đây là bản thiết kế chi tiết (Game Design Document) để bạn triển khai Hệ Thống Suy Diễn & Tàn Phương vào dự án game:

---

### 1. Hệ Thống Tàn Phương (Incomplete Recipes)

Thay vì nhặt được một cuộn giấy "Công thức Nguyệt Quang Cổ 100%" click vào là học được, người chơi sẽ chủ yếu tìm thấy **Tàn Phương**.

* **Chỉ số của một Tàn Phương (Item Attributes):**
* `Target_Gu`: Tên Cổ trùng mục tiêu.
* `Completion_Rate`: Tỷ lệ hoàn thiện (Ví dụ: 45%, 78%, 99%).
* `Flaw_Count`: Số lỗ hổng logic trong công thức.


* **Cơ chế Ép Luyện (Forced Refinement):** Người chơi *có quyền* đem một Tàn phương 80% đi luyện ngay lập tức mà không cần thôi diễn. Tuy nhiên:
* Tỷ lệ thất bại = 100% - `Completion_Rate` (Tức là 20% xịt).
* **Hình phạt xịt (Backlash):** Không chỉ mất nguyên liệu, mảng Đạo Ngân xung đột sẽ gây sát thương trực tiếp lên HP của người chơi, nặng thì nổ Không Khiếu (trừ Max Mana vĩnh viễn), nhẹ thì trọng thương mất đi Lượt hành động trong vài ngày.



### 2. Cơ Chế Thôi Diễn (Deduction System)

Đây là màn chơi chính của Trí Đạo. Để biến Tàn Phương 80% thành Cổ Phương 100%, người chơi phải mở giao diện Thôi Diễn (Deduction UI).

**A. Các tài nguyên cần thiết cho một ca Thôi Diễn:**

1. **Cổ Trí Đạo (Máy phát điện):** Trí Đạo chia làm 3 nhánh: *Niệm (Suy nghĩ), Ý (Ý chí), Tình (Cảm xúc)*. Người chơi phải trang bị các loại Cổ sinh ra Niệm. Ví dụ:
* *Tinh Niệm Cổ (Star Thought Gu):* Sinh ra Tinh Niệm. Suy diễn cực tốt vào ban đêm.
* *Ác Niệm Cổ (Malicious Thought Gu):* Sinh ra Ác Niệm. Suy diễn tốt các Cổ trùng mang tính sát thương, độc đạo.


2. **Chân Nguyên (Nhiên liệu):** Để Cổ Trí Đạo hoạt động và sinh ra "Niệm", phải liên tục bơm Chân Nguyên.
3. **Hồn Phách (Độ bền CPU):** Tốc độ suy nghĩ càng nhanh, Hồn Phách càng mau mệt mỏi. Nếu thanh Hồn Phách tụt xuống 0, người chơi sẽ rơi vào trạng thái "Hôn mê" hoặc "Tẩu hỏa nhập ma".

**B. Vòng lặp Gameplay Thôi Diễn (Deduction Mini-game/Process):**

* **Bước 1 - Đưa Tàn phương vào:** Hệ thống hiển thị số lượng "Niệm" cần thiết để lấp đầy phần trăm còn thiếu (VD: Cần tiêu hao 5000 Tinh Niệm để bù đắp 20% thiếu hụt).
* **Bước 2 - Setup Trận pháp:** Người chơi chọn Cổ Trí Đạo kích hoạt, thiết lập lượng Chân Nguyên bơm vào mỗi giây.
* **Bước 3 - Xử lý xung đột (Mini-game):** Trong quá trình thanh tiến trình chạy (Real-time), thỉnh thoảng sẽ hiện ra các **"Điểm Tắc Nghẽn Đạo Ngân"**. Ví dụ đang suy diễn Cổ Hỏa Đạo, hệ thống báo lỗi: "Vật liệu A xung khắc nhiệt độ với Vật liệu B".
* **Bước 4 - Quyết định của người chơi:** Người chơi phải chọn 1 trong 3 vật liệu thay thế để giải quyết Điểm Tắc Nghẽn. Nếu chọn đúng (dựa vào Cảnh giới Lưu phái của người chơi), Tàn phương tăng % hoàn thiện. Nếu chọn sai, bốc cháy mất đi một lượng "Niệm".

### 3. Đảo Ngược Cổ Phương (Reverse-Engineering)

Tính năng này dành cho việc người chơi may mắn bắt được một con Dã Cổ (Wild Gu) quý hiếm ngoài tự nhiên, nhưng không có công thức để tự sản xuất hàng loạt.

* **Cơ chế "Mổ xẻ" (Dismantling):** Đưa con Cổ đang sống vào Trận pháp Phân tích.
* **Trả giá (The Cost):** Con Cổ đó sẽ bị nghiền nát và chết vĩnh viễn trong quá trình phân tích. Người chơi mất đi 1 vật phẩm quý giá ở hiện tại để đổi lấy tiềm năng tương lai.
* **Kết quả thu được (The Output):** Hệ thống không bao giờ trả về 100% Cổ Phương. Nó sẽ trả về một Tàn Phương dao động từ **40% đến 80%** (Tỷ lệ này phụ thuộc vào chỉ số *Cảnh giới Luyện Đạo* và *Cảnh giới Trí Đạo* của người chơi).
* **Vòng lặp tiếp theo:** Người chơi lại phải cầm Tàn Phương này ném vào "Cơ chế Thôi Diễn" ở mục 2 để đẩy nó lên 100% trước khi bắt đầu auto-crafting.

### 4. Yếu tố Cảnh Giới Lưu Phái (Path Mastery Impact)

Trong nguyên tác, Phương Nguyên thôi diễn Cổ Huyết Đạo, Lực Đạo rất nhanh vì hắn là Tông sư các phái này. Game của bạn cũng cần áp dụng logic này:

* **Hệ số Đồng Điệu (Synergy Multiplier):** Nếu người chơi có Cảnh giới Tông Sư Hỏa Đạo, khi thôi diễn Tàn phương Hỏa Đạo, tiến trình Thôi diễn chạy nhanh x3, tiêu hao Chân nguyên giảm 50%.
* **Linh Quang Chớp Lóe (Inspiration Flash):** Nếu cảnh giới đủ cao, có tỷ lệ 1% kích hoạt "Linh Quang", lập tức hoàn thành 100% Tàn phương mà không tốn thêm tài nguyên nào (mô phỏng lại khoảnh khắc ngộ đạo trong truyện).

---

**Tóm tắt luồng code (Tech Logic) cho Developer:**

1. Tạo 1 table/collection `Recipes` với field `is_complete` (boolean) và `completion_rate` (float).
2. Khi player nhặt đồ, RNG drop `Recipe` với `completion_rate` ngẫu nhiên từ 30-90%.
3. Viết API `DeduceRecipe`: Nhận đầu vào là `RecipeID`, `Gu_Thoughts_ID`, tính toán tốn bao nhiêu thời gian và `Mana` dựa trên `Player.Path_Mastery`. Cập nhật dần `completion_rate` lên 100%.

Việc thiết kế như vậy biến Trí Đạo không chỉ là những dòng text sáo rỗng, mà thực sự trở thành một "cỗ máy ngốn tài nguyên" và đòi hỏi sự tính toán y hệt như cách các Cổ Tiên trong truyện phải bế quan vài năm trời chỉ để hoàn thiện một công thức! Bạn thấy luồng thiết kế này đã đủ độ "hardcore" cho dự án chưa?