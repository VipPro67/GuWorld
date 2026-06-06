Chào bạn, tôi là AI Gemini. Nhìn vào hai bức ảnh thiết kế UI của bạn, tôi phải dành lời khen vì giao diện trông rất gọn gàng, hiện đại và trực quan (rất đúng chuẩn một web game Next.js/React). Chức năng lọc theo Lưu phái (Trí Đạo, Huyết Đạo, Thổ Đạo...) và việc hiển thị nguyên liệu (`wild_boar_tusk: 5`) cùng tỷ lệ thành công (20%) đã chạy rất ổn.

**Tuy nhiên**, với tư duy của thế giới *Cổ Chân Nhân*, việc "phơi bày" toàn bộ Cổ phương (tất cả đều public) và hiển thị sẵn công thức sẽ biến game thành một dạng "click-to-craft" (nhấn để chế tạo) quá dễ dàng. Nó làm mất đi giá trị của Trí Đạo và sự độc quyền của Luyện Đạo.

Để chuyển hóa từ UI hiện tại thành hệ thống **Tàn Phương & Thôi Diễn** mà chúng ta đã bàn, dưới đây là bản triển khai chi tiết (Action Plan) từ Front-end (UI) đến Back-end (Logic):

---

### Bước 1: Thay đổi triết lý hiển thị Danh sách Cổ (Cải tạo Ảnh 1)

Hiện tại, trang "Công Thức Cổ Trùng" đang fetch toàn bộ dữ liệu từ Database và show lên cho mọi player. Chúng ta cần cá nhân hóa nó.

* **Tên trang:** Đổi từ *Công Thức Cổ Trùng* (Global Recipe) thành **Đông Song Cổ / Tàng Kinh Các Cá Nhân** (Personal Recipe Library).
* **Phân loại Tabs (Mới):** Thay vì chỉ lọc theo Lưu phái, hãy thêm một bộ lọc trạng thái:
* **Hoàn chỉnh (100%):** Những công thức player đã sở hữu hoặc tự thôi diễn xong. Hiển thị rõ ràng như UI hiện tại.
* **Tàn phương (Incomplete):** Những công thức player nhặt được lúc đánh quái hoặc mua từ chợ đen. Tiêu đề Cổ trùng có thể bị nhiễu (glitch effect) hoặc làm mờ đi. Hiển thị rõ % hoàn thiện (Ví dụ: *Tàn phương Tự Lực Cổ - 60%*).
* **Manh mối (Clues):** Thậm chí không biết tên Cổ, chỉ là một tờ giấy rách ghi "Một loại Cổ Thổ Đạo cần Hắc Thỉ Cổ làm mồi...".


* **Cơ chế Ẩn (Hide Mismatch):** Nếu hệ thống có 1000 Cổ trùng, nhưng player chỉ mới khám phá được 15 công thức, UI chỉ hiển thị đúng 15 thẻ Card đó. Các ô còn lại hiện bóng đen (Silhouette) có dấu chấm hỏi `?`.

### Bước 2: Nâng cấp Modal Chi tiết Công thức (Cải tạo Ảnh 2)

Trong hình thứ 2, Modal của "Tự Lực Cổ" đang hiển thị thẳng nguyên liệu và nút **[ Bắt đầu luyện ]**. Bạn cần chia Modal này thành hai trạng thái (States) khác nhau:

**Trạng thái 1: Khi là Tàn Phương (Ví dụ: 80% Hoàn thiện)**

* **Hiển thị nguyên liệu:** Giấu đi một số nguyên liệu. Ví dụ:
* `wild_boar_tusk`: 5
* `???`: 2 (Bị ẩn do công thức chưa hoàn chỉnh).


* **Nút Hành động:** Không phải là [ Bắt đầu luyện ], mà đổi thành nút **[ Đưa vào Thôi Diễn (Trí Đạo) ]**.
* **Cảnh báo rủi ro:** Có một nút phụ nhỏ **[ Ép Luyện (Forced Refine) ]** màu đỏ. Nếu player cố tình bấm, tỷ lệ thành công giảm mạnh, tỷ lệ nổ lò (trừ HP, hỏng Không khiếu) cực cao.

**Trạng thái 2: Khi đã Hoàn chỉnh (100%)**

* **Tỉ lệ thành công cơ bản:** (Giống UI hiện tại của bạn đang để 20%).
* **Khu vực "Tài nguyên phụ trợ":** Thay vì chỉ nhập một con số chung chung, hãy đổi thành **Slot đặt vật phẩm**. Người chơi có thể kéo thả Cổ Vận Đạo (tăng luck), hoặc vật liệu điều hòa nhiệt độ (Nước băng, Lửa mồi) vào đây để tăng cái mức *20%* kia lên.
* **Hiển thị Cảnh giới:** Thêm một dòng text nhỏ báo hiệu: *"Cảnh giới Luyện Đạo của bạn: Tân thủ (-5% Tỷ lệ thành công)"* hoặc *"Tông sư (+30% Tỷ lệ thành công)"*. Cảnh giới quyết định sống còn.

### Bước 3: Cấu trúc lại Database (Backend Logic)

Để UI hiển thị đúng, cấu trúc lưu trữ phải thay đổi. Developer của bạn không thể chỉ dùng một bảng `Recipes` chung nữa.

* **Bảng `Global_Recipes` (Master Data):** Chứa công thức chuẩn 100%, ẩn hoàn toàn khỏi API fetch của client.
* **Bảng `Player_Unlocked_Recipes`:** Một JSONB hoặc bảng Mapping lưu trong profile của mỗi player.
```json
{
  "player_id": "P001",
  "recipes": [
    {
      "gu_id": "GU_SELF_STRENGTH",
      "completion_rate": 60, // Hiển thị là Tàn phương trên UI
      "discovered_mats": ["wild_boar_tusk"] // Các nguyên liệu hiển thị, phần còn lại là ???
    },
    {
      "gu_id": "GU_WHITE_BOAR",
      "completion_rate": 100 // Hiển thị đẩy đủ, cho phép nút [Bắt đầu luyện]
    }
  ]
}

```



---

Sự thay đổi này biến cái giao diện "bấm là chế đồ" vô hồn thành một hành trình theo đuổi tri thức thực sự. Người chơi sẽ thèm khát đi tìm những mảnh tàn phương, và sẽ vỡ òa khi tự tay thôi diễn ra công thức 100%.

Theo bạn, với màn hình Modal chi tiết (Ảnh 2), chúng ta nên thiết kế giao diện (UI) của **"Mini-game Thôi Diễn"** đè lên trên Modal này luôn, hay nên chuyển hướng (Redirect) người chơi sang hẳn một Khu vực/Trang riêng biệt dành cho việc bế quan thôi diễn?