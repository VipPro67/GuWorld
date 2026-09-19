# WF-02: Spec Local Inspection vs Remote Divination access rules & UI flow

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:grilling`  
**Status:** Closed  
**Blocked by:** WF-01 (Closed)  

---

## Question

Đặc tả chi tiết quy tắc truy cập và giao diện người dùng cho 2 phương thức tiếp cận thông tin:
1. **Trinh Sát Cục Bộ (Local Inspection)**
2. **Suy Diễn Thiên Cơ Từ Xa (Remote Divination)**

---

## Resolution

1. **Trinh Sát Cục Bộ (Local Inspection)**:
   - **Xác thực Server**: Endpoint `POST /api/explore/inspect` kiểm tra người chơi và mục tiêu phải cùng `current_region` và khoảng cách $\text{hypot}(dx, dy) \le \text{investigationRange}$. Nếu không thỏa mãn $\rightarrow$ trả về lỗi `403 Forbidden`.
   - **Ngoại lệ Thân hữu**: Thành viên cùng Party và cùng Tông Môn (Sect) được miễn trừ kiểm tra FOV, có thể xem thông tin cơ bản của nhau thông qua liên lạc nội bộ.

2. **Xử lý UI Remote Context Menu**:
   - Khi click vào tên người chơi từ Global Chat, Leaderboard, Arena $\rightarrow$ Không mở Profile trực tiếp mà mở Context Menu:
     - *Sao chép danh tính / Lưu vào sổ ghi chép*.
     - *Nhắn tin riêng (Whisper)*.
     - *Thi triển Suy Diễn Thiên Cơ* (Tự động mở Modal Suy Diễn và điền sẵn thông tin mục tiêu).
     - *Mời vào Tổ đội / Mời vào Tông môn*.

3. **Cơ chế Suy Diễn Theo Lượng Manh Mối (Clue Multiplier & Fusion)**:
   - Modal "Suy Diễn Thiên Cơ" hỗ trợ nhập nhiều trường dữ liệu manh mối:
     - `Username` (Danh tính)
     - `SectName` (Tông môn)
     - `SuspectedRegion` (Khu vực nghi ngờ)
     - `SuspectedGu` (Cổ trùng nghi vấn)
     - `ItemClueID` (Vật phẩm chứa tàn dư khí tức / máu)
   - **Quy tắc**: Cung cấp càng nhiều thông tin ban đầu chính xác $\rightarrow$ cộng dồn Điểm Manh Mối ($Bonus_{clue}$), giúp tăng mạnh tỷ lệ đột phá màn sương và mở khóa càng nhiều thông tin chi tiết trong Báo Cáo Thiên Cơ (Divination Report).
