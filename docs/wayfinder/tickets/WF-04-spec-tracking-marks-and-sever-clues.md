# WF-04: Spec Tracking Marks lifecycle, location tracking, and Sever Clues mechanisms

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:grilling`  
**Status:** Closed  
**Blocked by:** None  

---

## Question

Đặc tả cơ chế hoạt động của Ấn Ký Truy Tung và Chém Đứt Manh Mối.

---

## Resolution

### 1. Các Nguồn Gieo Ấn Ký (Mark Triggering)
- **Trong Giao Tranh (Combat)**: Đánh trúng hoặc hạ gục mục tiêu khi trang bị Cổ có effect `TRACKING_MARK` $\rightarrow$ gieo *Huyết Mạch Ấn (Blood Mark)* hoặc *Trí Niệm Ấn (Wisdom Mark)*.
- **Suy Diễn Từ Xa (Remote Divination)**: Đạt kết quả Đại Thành Công $\rightarrow$ có thể kích hoạt gieo *Trí Niệm Ấn (Wisdom Mark)* từ xa (tiêu hao Chân/Tiên nguyên).
- **Gặp Mặt Trực Tiếp (Local Proximity)**: Lựa chọn "Âm thầm gieo ấn ký" khi cùng ô FOV (so khớp Trí đạo người gieo vs Cảnh giác mục tiêu).

### 2. Phân Loại, Thời Gian & Giới Hạn Ấn Ký
| Loại Ấn Ký | Nguồn Gốc | Thời Gian Tồn Tại | Đặc Điểm |
|---|---|---|---|
| **Khí Tức Tàn Dư (Residual Scent)** | Ô map khi di chuyển/PK | Cố định **15 phút** | Gắn với ô tọa độ `(region, x, y)`, dùng cho trinh sát lần theo vết |
| **Trí Niệm Ấn (Wisdom Mark)** | Suy diễn / Cổ Trí Đạo | **$30 \text{ phút} \times \text{Rank}$** | Theo dõi tọa độ thời gian thực |
| **Huyết Mạch Ấn (Blood Mark)** | Đổ máu trong giao tranh | **$2 \text{ giờ} \times \text{Rank}$** | Theo dõi thời gian thực, độ bền cao khó bị thanh tẩy |

- **Giới hạn số lượng mục tiêu theo dõi đồng thời**:
  - Phàm nhân (Rank 1-5): Tối đa **3 mục tiêu**.
  - Tiên nhân (Rank 6+): Tối đa **10 mục tiêu**.

### 3. Cơ Chế Chém Đứt Manh Mối (Sever Clues / Purge Marks)
- **Hành động**: Nút *"Chém Đứt Manh Mối / Đoạn Tuyệt Thiên Cơ"* trong giao diện Nhân vật / Tiên Khiếu.
- **Tiêu hao**: $20\%$ Chân Nguyên hiện tại (hoặc 5 Tiên Nguyên Thạch).
- **Quy tắc**: So khớp chỉ số effect `SEVER_CLUES` từ Cổ Kiếm Đạo / Trí Đạo / Ám Đạo + Cảnh giới Chân ý với sức mạnh của người gieo ấn ký.
- **Kết quả**: Xóa toàn bộ ấn ký bám trên người, xóa tàn dư khí tức tại ô hiện tại, hiển thị thông báo: *"Đã chém đứt X ấn ký đang âm thầm theo dõi bạn!"*.

### 4. Giao Diện Radar Theo Dõi (Tracking Radar UI)
- Drawer / Tab *"Mục Tiêu Theo Dõi"* trên Worldmap hiển thị:
  - Avatar, Username mục tiêu.
  - Tọa độ tức thời `(Region, X, Y)`.
  - Loại ấn ký & Thời gian đếm ngược.
  - Nút **"Định vị"**: Tự động lia camera và đánh dấu icon radar phát sáng tại vị trí con mồi.
