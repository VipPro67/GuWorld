Dưới đây là cấu trúc file Markdown (`.md`) mô tả chi tiết dự án, được chia thành từng giai đoạn (Phase) với các yêu cầu (Prompt/Task) cụ thể để bạn có thể copy/paste và giao việc cho AI Agent (như Cursor, GitHub Copilot, hoặc ChatGPT) code từng bước một.

```markdown
# THIẾT KẾ DỰ ÁN WEB GAME: CỔ CHÂN NHÂN (REVEREND INSANITY)

## 1. Tổng quan dự án (Project Overview)
- **Thể loại:** Text-based RPG, Quản lý tài nguyên vĩ mô, Sinh tồn, Thẻ bài (Deck-building).
- **Core Loop:** Khám phá (PVE) -> Kiếm Nguyên thạch/Nguyên liệu -> Nuôi Cổ (sinh tồn) -> Luyện Cổ (Nâng cấp) -> Ghép Sát Chiêu -> Chiến đấu (Turn-based Text Log).
- **Nguyên tắc cốt lõi:** Nhân vật chỉ là bình chứa năng lượng (Không khiếu), mọi sức mạnh, kỹ năng, phòng ngự đều đến từ trang bị (Cổ trùng).

## 2. Công nghệ sử dụng (Tech Stack)
- **Backend:** Golang (Fiber/Gin) xử lý logic đồng thời, tính toán combat và cron jobs.
- **Frontend:** Next.js (React), TailwindCSS, Zustand/Redux quản lý state.
- **Database:** PostgreSQL (Lưu dữ liệu người chơi, túi đồ), Redis (Xử lý cache, countdown thời gian hồi mana/độ đói).

---

## 3. KẾ HOẠCH GIAO VIỆC CHO AI AGENT (PHASE-BY-PHASE DELEGATION)

> **Hướng dẫn cho người quản lý dự án:** Hãy giao từng Task một cho AI. Đợi AI code xong, test chạy thành công mới giao Task tiếp theo. Không nhồi nhét quá nhiều logic vào một prompt.

### GIAI ĐOẠN 1: Xây dựng Core Models & Hệ thống Không Khiếu (Aperture)
**Task 1.1: Thiết kế Database & Models (Golang)**
- **Yêu cầu AI:** Tạo các struct Golang và schema PostgreSQL cho `Player`. Player cần có các trường: ID, Tên, Tư chất (Aptitude: Giáp, Ất, Bính, Đinh), Cấp độ (Rank 1-9), Tiểu cảnh giới (Sơ, Trung, Cao, Đỉnh phong), Chân nguyên hiện tại (Current Mana), Chân nguyên tối đa (Max Mana).
- **Lưu ý:** Max Mana phụ thuộc vào Tư chất (Ví dụ: Đinh 20-39%, Bính 40-59%, Ất 60-79%, Giáp 80-99%).

**Task 1.2: Thuật toán hồi phục Chân nguyên (Golang + Redis)**
- **Yêu cầu AI:** Viết một Cron Job hoặc Background Worker trong Golang. Cứ mỗi X giây, tự động cộng thêm Mana cho tất cả Player. Tốc độ hồi phục phụ thuộc vào Tư chất (Giáp hồi nhanh nhất, Đinh hồi chậm nhất).
- **API:** Viết API `GET /api/player/status` để trả về lượng Mana hiện tại.

**Task 1.3: Giao diện Không Khiếu (Next.js)**
- **Yêu cầu AI:** Tạo màn hình `Dashboard` hiển thị thông tin Player. Vẽ một thanh Progress Bar thẳng đứng (hoặc hình giọt nước) đại diện cho Không Khiếu. Dùng React Query polling API `status` mỗi 2 giây để thanh Mana tự động dâng lên giống thời gian thực. Cập nhật màu sắc chân nguyên theo cấp độ (Thanh đồng, Xích thiết, Bạch ngân...).

### GIAI ĐOẠN 2: Hệ thống Cổ Trùng & Quản lý Tài nguyên (Inventory & Feeding)
**Task 2.1: Từ điển Cổ Trùng & Túi đồ (Backend)**
- **Yêu cầu AI:** Tạo struct `GuDictionary` (ID, Name, Rank, Type, BaseDamage/Defense, HungerRate, FoodType) và `PlayerInventory`. Dùng JSONB trong PostgreSQL để lưu danh sách Cổ trùng mà Player đang sở hữu. Thêm dữ liệu mẫu: Nguyệt Quang Cổ, Ngọc Bì Cổ, Tửu Trùng.

**Task 2.2: Cơ chế Nuôi Cổ (Sinh tồn cơ bản)**
- **Yêu cầu AI:** Thêm chỉ số `Hunger` (Độ đói) vào từng con Cổ của Player. Viết Worker trong Golang định kỳ trừ độ đói của Cổ trùng. Viết API `/api/inventory/feed` để cho Cổ ăn, tiêu hao vật phẩm trong kho (Ví dụ: Nguyệt Quang cổ ăn hoa Nguyệt Lan). Nếu `Hunger == 0`, đánh dấu trạng thái Cổ là "Ngủ đông" hoặc "Chết".

**Task 2.3: Giao diện Quản lý Cổ (Next.js)**
- **Yêu cầu AI:** Tạo Component `GuInventory`. Hiển thị danh sách Cổ trùng dưới dạng Grid. Mỗi item hiển thị icon, Tên, Cấp độ và một thanh Độ đói (Hunger bar). Thêm nút "Cho Ăn" (Feed) gọi API feed.

### GIAI ĐOẠN 3: Luyện Cổ (Crafting / Gacha Mechanic)
**Task 3.1: Thuật toán Hợp Luyện Cổ (Golang)**
- **Yêu cầu AI:** Tạo bảng `RefiningRecipes` (Bí phương). Viết API `/api/refine`. Nhận đầu vào là 2 (hoặc nhiều) ID Cổ trùng (VD: Ngọc Bì Cổ + Nguyệt Quang Cổ = Nguyệt Nghê Thường / Bảo Nguyệt Quang Vương cổ). 
- **Logic cốt lõi:** Luyện cổ có xác suất thành công (RNG). Sử dụng `math/rand`. Nếu thành công, xóa nguyên liệu cũ, thêm Cổ mới. Nếu thất bại, xóa nguyên liệu cũ (Cổ chết). Trả về kết quả JSON.

**Task 3.2: Giao diện Lò Luyện Cổ (Next.js)**
- **Yêu cầu AI:** Tạo Component `RefiningFurnace`. Cho phép User kéo thả Cổ trùng từ Inventory vào các ô nguyên liệu. Bấm nút "Hợp Luyện". Thêm hiệu ứng loading hồi hộp (Gacha style) trước khi hiển thị Toast message "Thành công" hoặc "Thất bại, Cổ trùng đã chết".

### GIAI ĐOẠN 4: Hệ thống Kỹ năng & Cơ chế Chiến đấu (Sát Chiêu)
**Task 4.1: Xây dựng Design Pattern cho Cổ Kỹ Năng (Golang)**
- **Yêu cầu AI:** Sử dụng **Strategy Pattern**. Tạo interface `GuAbility` có hàm `Execute(CombatContext)`. Triển khai logic tiêu hao Mana cho từng loại Cổ.
- **Rule quan trọng (Cân bằng game):** Tính lượng tiêu hao mana dựa trên chênh lệch cấp độ giữa Player và Cổ trùng. Nếu Player Rank < Gu Rank -> Trả về lỗi "Vượt cấp, Không khiếu bị thương" và trừ HP. Nếu Player Rank >= Gu Rank, chia nhỏ lượng tiêu hao dựa trên Tiểu cảnh giới.

**Task 4.2: Cơ chế Tổ hợp Sát Chiêu (Killer Moves)**
- **Yêu cầu AI:** Cho phép User trang bị nhiều Cổ trùng (Công, Thủ, Trị liệu). Khi combat, tính tổng Damage = `Base_Damage * Hệ số Đạo ngân - Target_Defense`. Trả về một mảng String chứa Log chiến đấu (Ví dụ: "Bạn dùng Cổ Toàn Lực Ứng Phó + Khổ Lực, đánh ra Hư ảnh Tê giác!").

**Task 4.3: Áp chế Dị vực (Regional Suppression)**
- **Yêu cầu AI:** Thêm thuộc tính `Region` cho Map và Cổ trùng. Trong hàm tính Damage, nếu `Gu.OriginRegion != Player.CurrentRegion`, giảm Rank (Cấp độ) thực tế của Cổ trùng xuống 1 bậc, nhưng giữ nguyên lượng Mana tiêu hao.

**Task 4.4: Giao diện Chiến đấu Text-based (Next.js)**
- **Yêu cầu AI:** Tạo màn hình `Combat`. Chia làm 2 nửa (Player vs Enemy). Bên dưới là Action Bar hiển thị các Cổ trùng đang trang bị. Khi bấm dùng Cổ, thanh Mana phải giật xuống ngay lập tức. Hiển thị một khung `<ScrollableLog />` để in ra các dòng text chiến đấu trả về từ Backend.

### GIAI ĐOẠN 5: Khám phá Thế giới (Map Nodes)
**Task 5.1: Xây dựng Map Node cơ bản (Golang + Next.js)**
- **Yêu cầu AI:** Tạo dữ liệu tĩnh cho các khu vực (Thanh Mao Sơn, Truyền thừa Tam Vương, v.v.). Tạo nút "Khám phá" (Explore). Khi bấm, tiêu hao X Chân nguyên, gọi API trả về sự kiện ngẫu nhiên: Nhặt được nguyên thạch, gặp Thú hoang, hoặc nhặt được nguyên liệu cho Cổ ăn.

---
## 4. Ghi chú thiết kế dành cho AI
- Tuyệt đối không thiết kế nhân vật có cấp độ level up truyền thống tăng máu/sát thương. Sát thương cơ bản luôn cố định, máu cố định. Mọi chỉ số phải được cộng từ việc dùng **Cổ Thể Chất** (Passive Gu) cải tạo vĩnh viễn (lưu vào dạng JSONB trong DB).
- Chú trọng xử lý Race Condition trong Golang khi thao tác trừ tiền (Nguyên thạch) và trừ Mana để chống hack/cheat.
```