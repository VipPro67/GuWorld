# [WF-REF-01] Chốt mô hình phiên luyện cổ: tương tác, thời gian, phạm vi, cam kết tài nguyên

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:grilling`
**Status:** Closed
**Blocked by:** None

---

## Question

Chốt các quyết định cấp cao trước khi spec chi tiết:

1. Sản phẩm cuối của map: spec đầy đủ, prototype, hay chỉ quyết định?
2. Người chơi tương tác trong lúc luyện theo mô hình nào?
3. Thời gian phiên trôi khi nào, tính từ đâu?
4. Có thay thế hoàn toàn nút bấm nhanh không? Ngưỡng "cổ cao" là gì?
5. Server tin client đến đâu?
6. Trừ tài nguyên lúc nào, bỏ dở thì mất gì?
7. Bỏ lỡ bước / offline quá hạn thì sao?
8. Trong lúc phiên chạy, người chơi được làm gì? Chết thì phạt thế nào?

---

## Resolution

### 1. Sản phẩm cuối
**Spec đầy đủ** để agent implement sau, kèm **một prototype nhỏ** (WF-REF-08) để chốt cảm giác trước khi đóng băng số liệu.

### 2. Mô hình tương tác
**Hybrid, tăng theo cấp Cổ**: mỗi phiên gồm 3–7 bước; mỗi bước là một thao tác ngắn (canh nhịp / chuỗi ấn / lựa chọn / sắp thứ tự) trong cửa sổ thời gian. Điểm từng bước cộng dồn thành **điểm chính xác 0–100** và chỉ **cộng thêm** vào tỉ lệ thành công (không trừ). Không dùng mô hình chỉ-chờ-thời-gian, cũng không phải rhythm game liên tục.

### 3. Thời gian
Phiên **chạy nền trên server**, tính từ thời điểm bắt đầu (`started_at`), không phụ thuộc việc UI đang mở hay người chơi online. Mỗi bước có cửa sổ thao tác riêng; hết hạn cả phiên thì server resolve (worker + lazy-resolve khi mở UI).

### 4. Phạm vi & đường nhanh
- Bắt buộc qua phiên: **Cổ có rank ≥ rank người chơi**, cộng **toàn bộ Tiên Cổ (rank ≥ 6)**.
- Đường nhanh (`POST /api/refine` hiện tại) giữ cho cổ rank thấp hơn người chơi: **tức thời**, giữ nguyên công thức tỉ lệ, **không có điểm chính xác, không tăng Chân Ý Luyện Đạo**.

### 5. Quyền quyết định của server
**Server-authoritative hoàn toàn.** Server giữ seed, kế hoạch bước, deadline, sequence và tự tính điểm/tỉ lệ. Client chỉ gửi input rời rạc kèm timestamp; không bao giờ gửi `final_rate`, `success` hay điểm tổng.

### 6. Cam kết tài nguyên
**Trừ ngay khi bắt đầu phiên**: mana, Nguyên Thạch, Tiên Nguyên Thạch, nguyên liệu phụ; Cổ nguyên liệu bị **khóa** (không dùng/trade/cho ăn) cho tới khi resolve.
- Hủy giữa chừng: mất mana + đá; **hoàn 50% nguyên liệu phụ**; Cổ nguyên liệu trả về trạng thái **Hấp Hối** (đói 1).
- Chỉ **một phiên active / người chơi**.

### 7. Bỏ lỡ
- Bỏ lỡ một bước: bước đó **0 điểm**, các bước sau vẫn làm được; **không được cộng** phần tỉ lệ từ điểm chính xác.
- Hết hạn cả phiên: resolve với điểm chính xác 0, vẫn roll theo tỉ lệ gốc (không auto-fail), gửi notification.

### 8. Hành vi trong phiên & chết
Người chơi vẫn đi lại, combat, tu luyện, trade bình thường (chỉ Cổ nguyên liệu bị khóa).
**Chết khi phiên đang chạy → giảm tỉ lệ thành công của phiên** (không kết thúc phiên, không mất thêm tài nguyên ngoài hình phạt chết hiện có). Mức giảm cụ thể do WF-REF-03 chốt.

---

## Hệ quả cho các ticket sau

- WF-REF-03 phải định nghĩa: accuracy → bonus, mức giảm khi chết (có stack không), thưởng Chân Ý theo accuracy.
- WF-REF-04 phải định nghĩa 4 minigame v1 và cách chấm server.
- WF-REF-05 phải dùng mô hình lock row session, không lock `players` dài hạn.
- WF-REF-09 phải chốt kinh tế đường nhanh và tương thích `POST /api/refine`.
