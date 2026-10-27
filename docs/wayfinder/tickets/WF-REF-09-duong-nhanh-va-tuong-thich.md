# [WF-REF-09] Kinh tế đường nhanh (quick path) & tương thích API cũ

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:grilling`
**Status:** Open
**Blocked by:** WF-REF-03

---

## Question

Cổ rank thấp hơn rank người chơi vẫn đi đường nhanh `POST /api/refine` tức thời. Cần chốt để đường nhanh không lấn át phiên.

### 1. Điều kiện đường nhanh
- Chính xác: `target_rank < player_rank` **và** `target_rank < 6`? (Tiên Cổ luôn qua phiên.)
- Người chơi rank 0/1 có được dùng đường nhanh không? (Rank 1 luyện Cổ rank 1 là ngang cấp → phiên; rank 1 luyện rank 0 không tồn tại.)
- Có giới hạn số lần đường nhanh mỗi ngày/phút không? (Rate limit hiện 20/phút.)

### 2. Kinh tế đường nhanh
Đã chốt: giữ nguyên công thức tỉ lệ, không accuracy, không tăng Chân Ý Luyện Đạo. Cần chốt thêm:
- Có **giảm** tỉ lệ đường nhanh (ví dụ -5% toàn dải) để khuyến khích phiên không?
- Có tăng chi phí (mana/đá) cho đường nhanh không?
- Có bỏ hoàn toàn phần thưởng Chân Ý, hay vẫn cho một phần nhỏ (ví dụ 25%)?
- Pity `insight_<rank>` vẫn áp dụng cho đường nhanh chứ?

### 3. Ranh giới hiển thị
- UI phân biệt thế nào để người chơi hiểu "Cổ này bắt buộc luyện có thời gian"?
- Nếu người chơi gọi `POST /api/refine` với Cổ ngang cấp: trả lỗi 400 kèm hướng dẫn mở phiên, hay tự động chuyển thành tạo phiên?
- Có endpoint trả metadata "đường nào áp dụng cho recipe này" không?

### 4. Tương thích ngược
- Giữ `POST /api/refine` vô thời hạn hay deprecate dần?
- Client cũ (nếu có) gọi route này với Cổ cao sẽ nhận lỗi gì?
- `refine_deduce` không đổi.
- Tài liệu cần cập nhật: `docs/API.md`, `WIKI.md` mục Luyện Cổ, `docs/GAME_DATA.md` nếu thêm bảng.
- Frontend: `cultivationSlice.refineGu` có giữ nguyên hay tách `refineQuick` / `startRefineSession`?

### 5. Chống lạm dụng
- Một người chơi có thể spam đường nhanh để farm Chân Ý không? (Đã chốt không tăng Chân Ý, nhưng cần xác nhận không có nguồn gián tiếp.)
- Có thể dùng đường nhanh để né hình phạt chết/backfire của phiên không? (Đường nhanh vẫn roll backfire như hiện tại.)
- Có cần log/audit riêng để phát hiện bất thường?

### 6. Mốc chuyển đổi
- Người chơi mới (rank 0–1) trải nghiệm phiên ngay từ đầu hay chỉ khi lên rank 2?
- Có tutorial/onboarding cho phiên luyện đầu tiên không? (Onboarding hiện đã nhắc luyện Cổ.)
- Có chế độ "luyện thử" không tốn tài nguyên cho lần đầu không?

---

## Resolution

_Chưa giải quyết._
