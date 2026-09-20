# [NOTE] Định Hướng Sát Chiêu Phức Hợp (Compound Killer Moves) & Song Đạo Điều Hòa

**Trạng thái:** Tạm hoãn (Deferred) — Chờ đợt refactor toàn diện hệ thống Sát Chiêu (Killer Moves).  
**Liên kết ticket:** [WF-DAO-05](../tickets/WF-DAO-05-spec-compound-killer-moves-and-dual-path.md)  
**Ngày ghi nhận:** 2026-09-20  

---

## 1. Lý Do Tạm Hoãn
Hệ thống Sát Chiêu (Killer Moves) hiện tại trong codebase còn mang tính sơ khởi và sẽ được thiết kế lại toàn diện trong một roadmap riêng. Vì vậy, việc tích hợp sâu các cơ chế Sát chiêu Phức hợp đa phái vào đợt cân bằng Đạo ngân này được tách riêng ra để tránh phụ thuộc vào cấu trúc cũ.

---

## 2. Các Ý Niệm Thiết Kế Cốt Lõi Đã Ghi Nhận (Để Triển Khai Sau)

1. **Khái niệm Sát Chiêu Phức Hợp (Compound Killer Moves):**
   - Sát chiêu kết hợp từ $2$ lưu phái trở lên (ví dụ *Thủy Hỏa Tương Dung*, *Thời Không Tỏa Khóa*, *Huyết Ảnh Kiếm Quang*).
   - Mỗi chiêu thức khai báo danh sách `paths: ["fire", "water"]` và chỉ số `compound_bonus: 0.25` (đại diện cho cấu trúc sát chiêu đã được dung hợp quy tắc từ trước).

2. **Cảnh Giới Điều Hòa Của Sát Chiêu Phức Hợp:**
   $$\text{CompoundRealmControl} = \min_{P \in \text{paths}} \left(\text{RealmControl}(P)\right)$$
   - *Nguyên lý mắt xích yếu nhất:* Người chơi muốn thi triển hoàn hảo sát chiêu Hỏa + Thủy thì bắt buộc phải có Cảnh giới Chân Ý cao ở **cả 2 phái**.

3. **Cơ Chế Song Đạo Điều Hòa (Dual-Path Harmony):**
   - Thừa kế từ các truyền thừa song tu cổ đại hoặc Cổ trùng dung hợp đặc thù, giúp giảm trừ trực tiếp xung đột giữa 2 phái chỉ định:
     $$\text{PairConflict} = \text{BasePairConflict} \times (1.0 - \text{HarmonyReduction})$$

4. **Hệ Số Độ Phức Tạp Khi Kết Hợp Nhiều Phái:**
   - 2 Lưu phái: $\times 1.25$ độ phức tạp.
   - 3 Lưu phái: $\times 1.60$ độ phức tạp.
   - 4 Lưu phái: $\times 2.10$ độ phức tạp.
   - 5 Lưu phái trở lên: $\times 3.00$ độ phức tạp (Yêu cầu cảnh giới Vô Thượng hoặc Tiên Thai).
