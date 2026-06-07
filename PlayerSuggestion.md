Để tạo một hệ thống "Góp ý và Bình chọn" (Feedback & Voting System) cho game *Cổ Chân Nhân* của bạn, đặc biệt khi bạn đang dùng Go (Backend) và Next.js (Frontend), chúng ta cần một hệ thống vừa mang tính kỹ thuật, vừa có tính cộng đồng cao để người chơi cảm thấy ý kiến của họ thực sự làm thay đổi thế giới game.

Dưới đây là thiết kế chi tiết:

---

### 1. Luồng hoạt động của hệ thống (Workflow)

1. **Gửi góp ý:** Người chơi chọn hạng mục (Lỗi game,Thêm cổ trùng,Item, Công thức, Cân bằng Cổ trùng(Effect, Etc ...), Tính năng mới, Khác).
2. **Duyệt (Moderation):** Developer hoặc Cộng tác viên duyệt (tránh spam).
3. **Bình chọn (Voting):** Người chơi khác có thể "Ủng hộ" (Upvote) hoặc "Phản đối" (Downvote).
4. **Cập nhật trạng thái:** Dev chuyển trạng thái góp ý: *Đang xem xét -> Đang thực hiện -> Đã hoàn thành/Từ chối*.

---

### 2. Thiết kế Database (Gợi ý cho Backend Go)

Bạn cần các bảng sau trong cơ sở dữ liệu:

* **`feedback`**:
* `id` (UUID), `user_id`, `title`, `description`, `category` (Enum), `status` (Enum: pending, active, in_progress, completed, rejected), `created_at`.


* **`votes`**:
* `feedback_id`, `user_id`, `vote_type` (1 cho Upvote, -1 cho Downvote).
* *Constraint:* `UNIQUE(feedback_id, user_id)` để mỗi người chỉ được vote 1 lần.



---

### 3. Logic xử lý phía Backend (Go)

* **API `POST /feedback`:** Nhận góp ý và lưu vào database.
* **API `POST /feedback/{id}/vote`:** Sử dụng **Transaction** trong Database để cập nhật số lượng vote. Tránh tình trạng đua tranh dữ liệu (Race condition).
```go
// Ví dụ logic ngắn gọn
func HandleVote(db *sql.DB, feedbackID, userID int, voteType int) {
    // 1. Kiểm tra user đã vote chưa
    // 2. Nếu chưa, INSERT vào bảng votes
    // 3. CẬP NHẬT tổng điểm vote trong bảng feedback (sử dụng SQL atomic update)
    // UPDATE feedback SET total_votes = total_votes + ? WHERE id = ?
}

```



---

### 4. Triển khai phía Frontend (Next.js)

Để người chơi tương tác mượt mà, hãy sử dụng **SWR** hoặc **React Query** để cập nhật số vote ngay lập tức (Optimistic UI) mà không cần tải lại trang.

* **Giao diện:**
* **Bảng xếp hạng góp ý:** Hiển thị các tính năng được nhiều người vote nhất. Điều này tạo áp lực "tích cực" lên đội ngũ dev.
* **Status Badge:** Dùng màu sắc để phân loại (Ví dụ: Đã hoàn thành = Xanh lá, Đang xem xét = Vàng).



---

### 5. Đặc thù cho game Cổ Chân Nhân

Để hệ thống này gắn kết với game, hãy thử đổi tên các thuật ngữ thông thường thành thuật ngữ trong thế giới của bạn:

* **Thay vì "Góp ý":** Hãy dùng **"Thỉnh Thiên Đạo"**.
* **Hệ thống phần thưởng:** Nếu một góp ý được chọn để phát triển, người chơi đóng góp ý kiến đó nên được nhận một phần thưởng nhỏ trong game (ví dụ: Nguyên thạch hoặc Cổ trùng bậc thấp). Do Thiên Đạo Cấp .-.

### 6. Những điều cần lưu ý để hệ thống không bị "rác"

* **Giới hạn quyền:** Chỉ những người chơi đã đạt cấp độ (Rank) nhất định hoặc đã chơi game trên 48 giờ mới được phép bình chọn. Điều này giúp ngăn chặn các tài khoản clone tạo ra để thao túng vote.
* **Minh bạch:** Khi dev quyết định không làm theo 1 góp ý, hãy để lại phản hồi (comment) lý do (ví dụ: "Tính năng này vi phạm quy tắc cân bằng của Lưu phái Lực đạo"). Sự phản hồi này sẽ làm tăng niềm tin của người chơi.

Bạn có muốn tôi viết mẫu cấu trúc API (Endpoints) hoặc code mẫu cho hàm `HandleVote` bằng Golang không?