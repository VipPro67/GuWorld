# [WF-REF-04] Đặc tả bộ minigame v1: luật chơi, chấm điểm, seed & chống replay

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:grilling`
**Status:** Closed
**Assignee:** Agent
**Blocked by:** None

---

## Question

Chốt 4 minigame v1 và hợp đồng input để server chấm lại được. Ràng buộc đã chốt: chỉ input rời rạc, không kéo-thả tự do/vật lý.

### 1. Bộ 4 dạng
Đặc tả chi tiết cho từng dạng:
- **Canh Nhịp**: kim chạy qua vùng ngọt; người chơi tap để dừng. Cần: tốc độ, độ rộng vùng ngọt, số lần tap được phép, công thức điểm theo khoảng cách tâm.
- **Dãy Ấn**: hiện chuỗi 3–6 ấn trong N giây, người chơi lặp lại. Cần: độ dài theo rank, thời gian hiện, thời gian nhập, điểm theo số ấn đúng và thời gian.
- **Chọn Hành Động**: 2–3 lựa chọn ứng biến (ví dụ "tăng hỏa", "hạ hỏa", "ổn định"). Cần: nguồn dữ liệu tình huống, cách xác định đáp án đúng (theo path/region/recipe/rank), điểm cho lựa chọn đúng/sai/gần đúng.
- **Ghép Liệu**: sắp xếp thứ tự nguyên liệu. Cần: thứ tự đúng lấy từ đâu (`required_materials`? path? random seed?), số mảnh theo rank, điểm theo số vị trí đúng (Kendall tau hay đếm đơn giản).

### 2. Chọn dạng cho từng bước
- Ngẫu nhiên đều theo seed, hay có trọng số theo rank/path/recipe?
- Có cấm lặp cùng dạng ở 2 bước liền (đã gợi ý) không?
- Bước cuối (khai lò) có luôn là một dạng cố định không?

### 3. Hợp đồng input
Đề xuất payload mỗi bước:

```json
{
  "step_index": 2,
  "nonce": "uuid-server-issued",
  "action": "TAP | SEQUENCE | CHOICE | ORDER",
  "payload": { "...": "..." },
  "client_ts": 1730000000000
}
```

Cần chốt:
- `client_ts` có dùng để chấm không (server vẫn dùng `NOW()`), hay chỉ để log?
- Một bước chỉ được submit **một lần**; sai/trễ thì khóa bước (chống sửa điểm).
- Có cho submit lại khi mạng lỗi không? Đề xuất: idempotency key, server trả kết quả cũ nếu cùng nonce.
- Giới hạn kích thước payload và validate kiểu dữ liệu.

### 4. Seed & chống replay
- Seed sinh ở server khi tạo phiên, lưu cùng kế hoạch bước; sequence/decoy/vị trí ngọt sinh từ seed + step index.
- Client không được biết trước seed/kế hoạch.
- Chống replay: nonce mỗi bước, một lần dùng, gắn với session + step; từ chối timestamp ngoài cửa sổ.
- Có cần ký/checksum kế hoạch bước không?

### 5. Chấm điểm & hiển thị
- Điểm mỗi bước 0–100; công thức cho partial credit từng dạng.
- Trả về client ngay sau mỗi bước: điểm bước, tổng tạm, thời gian còn lại, bước kế tiếp (không lộ seed).
- Có hiện đáp án đúng sau khi chấm không (để học), hay chỉ hiện điểm?

### 6. Khả năng tiếp cận & thiết bị
- Bàn phím (1/2/3, Space) và touch.
- Không phụ thuộc màu sắc; có text/icon thay thế.
- Tỉ lệ khung hình và kích thước tối thiểu trên mobile.

### 7. Cấu hình độ khó
- Độ rộng cửa sổ, số decoy, độ dài chuỗi, tốc độ kim thay đổi theo rank thế nào? (Bảng nhịp độ ở WF-REF-06.)
- Có random trong biên độ nhỏ để mỗi lần luyện khác nhau không?

---

## Resolution

Đã chốt đặc tả tại [WF-REF-04 Spec](../specs/WF-REF-04-minigames-v1.md). Tóm tắt:

- **4 dạng v1:** `TAP` (Canh Nhịp), `SEQUENCE` (Dãy Ấn), `CHOICE` (Chọn Hành Động), `ORDER` (Ghép Liệu). Hỏa Hậu & Phản Xạ để v1.1.
- **Cấu trúc bước:** bước 1 luôn `CHOICE`, bước N luôn `TAP`; giữa gồm `K−1` minigame vật lý + `N−K−1` `CHOICE`; không lặp minigame cùng loại liền kề; bước N−1 không phải `TAP`.
- **Lịch cố định từ `started_at`:** `gap = duration/N`, bước i mở tại `started_at + (i−1)×gap`, đóng tại `min(open + window, ends_at)`. Offline vẫn trôi; bỏ lỡ = 0 điểm.
- **Chấm điểm:** TAP tuyến tính theo khoảng cách tâm; SEQUENCE `100×đúng/L` (đúng hết nhanh = 100, chậm sàn 80); CHOICE 100/50/0; ORDER `100×cặp_liền_kề_đúng/(M−1)`.
- **Độ khó** theo bảng rank (L 3→6, display 600→320ms, w 0.12→0.05, P 2000→900ms, M 3→6, option 2→3).
- **`RefineDecision.csv`** generic 10–12 tình huống, server xáo option, không gửi `correct_index`; registry rỗng → fallback `CHOICE` thành `TAP` + log.
- **Hợp đồng input** `{step_index, nonce, action, payload, client_ts}`; chấm bằng `NOW()`, `client_ts` chỉ log; 1 nonce/bước, idempotent; grace mạng 1s và clamp về `window_end`.
- **Chống gian lận:** seed `crypto/rand`, chỉ gửi params bước hiện tại, rate limit + audit. Chấp nhận rủi ro client tự động hoá vì accuracy chỉ +4–8%; chống bot sâu để fog.

