# [WF-REF-08] Prototype một minigame + vòng bước rank 1

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:prototype`
**Status:** Open
**Blocked by:** WF-REF-04

---

## Question

Dựng prototype rẻ, chạy được, để trả lời: **nhịp 60 giây / 3 bước cho rank 1 có đủ thú vị và không gây sốt ruột không?**

Phạm vi đề xuất:
- Một trang HTML/React độc lập (hoặc component tạm trong `frontend/src/components/prototype/`), **không nối backend**.
- Dữ liệu giả: recipe rank 1, 3 bước, 1 minigame.
- Minigame chọn **Canh Nhịp** (dễ dựng nhất): kim chạy, tap để dừng, hiện điểm 0–100.
- Mô phỏng timeline: bước 1 mở ở 0s, cửa sổ 20s; bước 2 mở sau 20s; bước 3 mở sau 40s; kết thúc 60s.
- Hiển thị điểm từng bước, điểm chính xác, bonus tỉ lệ giả định, kết quả success/fail giả lập.
- Có nút "bỏ lỡ bước" để thử cảm giác mất cửa sổ.
- Đo được: thời gian phản hồi, số lần tap, cảm nhận chủ quan của người chơi.

Không cần:
- Auth, DB, API, WS, inventory thật.
- 4 minigame, animation đẹp, mobile hoàn chỉnh.

Câu hỏi cần prototype trả lời:
1. Cửa sổ 20s có quá dài/ngắn không?
2. 3 bước trong 60s có tạo nhịp "đang luyện" hay chỉ là 3 lần tap rời rạc?
3. Có nên giảm thời gian chờ giữa các bước (ví dụ bước mở liên tiếp thay vì chờ hết cửa sổ) không?
4. Người chơi có muốn quay lại sau khi đóng UI không, hay thấy phiền?

Đầu ra: một file prototype + ghi chú cảm nhận, link vào ticket này. Kết quả sẽ điều chỉnh bảng nhịp độ WF-REF-06 và luật Canh Nhịp WF-REF-04.

---

## Resolution

_Chưa giải quyết._
