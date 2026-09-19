# WF-05: Spec Gu Resonance & World Dao Mark Detection across Mortal vs Immortal

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:grilling`  
**Status:** Closed  
**Blocked by:** None  

---

## Question

Quy định cơ chế Dò Tìm Cổ Trùng và Cảm Ứng Đạo Ngân trong thế giới.

---

## Resolution

### 1. Đối Tượng Quét & Cơ Chế Chống Dò Tìm
- **Đối tượng quét**:
  - *Cổ Hoang Dã & Tài Nguyên Bản Đồ*: Quái vật rơi Cổ, mỏ Cổ, vật phẩm rơi trên mặt đất.
  - *Cổ Trùng Của Người Chơi*: Cổ trùng đang mang trong người có dao động trùng khớp với tiêu chí tìm kiếm.
- **Chống dò tìm (Anti-Resonance)**: Người chơi trang bị Cổ có effect `CONCEAL_GU` (Phong Ấn Khí Tức / Ẩn Nặc) hoặc mang capability `PERFECT_DISGUISE` (Thái Độ Cổ) sẽ che chắn toàn bộ Cổ trùng khỏi radar quét của người khác.

### 2. Phân Cấp Tiên - Phàm (Phạm Vi & Chi Phí)
| Cấp bậc | Phạm Vi Quét | Chi Phí Tiêu Hao | Hồi Chiêu (Cooldown) |
|---|---|---|---|
| **Phàm nhân (Rank 1-5)** | Bán kính $R = \text{Rank} \times 20$ ô (nội vùng `current_region`) | $50\%$ Chân Nguyên + 200 Nguyên Thạch | 5 phút (giảm theo Cảnh giới Tín/Trí đạo) |
| **Tiên nhân (Rank 6+)** | Toàn Region hoặc quét liên Vực sang Region khác | $2$ Tiên Nguyên Thạch (nội vực) / $5$ TNT (liên vực) | 2 phút |

### 3. Cảm Ứng Đạo Ngân & Xung Kích Thiên Kiếp (Immortal Dao Pulse Events)
- Khi có Tiên nhân **Độ Kiếp (Tribulation)**, **Khai Mở Tiên Khiếu**, hoặc **Khai Thác Mỏ Tiên Cấp** $\rightarrow$ Hệ thống phát ra xung kích Đạo Ngân thời gian thực (*Dao Pulse*).
- Tiên nhân khác có Đạo ngân cùng lưu phái hoặc Trí/Tín Đạo Tông Sư sẽ nhận thông báo thiên cơ: *"Cảm ứng được dao động [Lưu Phái] cực mạnh tại [Region] (X, Y)!"* kèm điểm phát quang cảnh báo 3 phút trên Worldmap.

### 4. Giao Diện Radar Cảm Ứng (Radar Ping Overlay)
- Hiển thị các vòng sóng radar (Pings) trên Worldmap:
  - Điểm trinh sát cao: Hiện icon Cổ trùng và danh tính mục tiêu tại ô.
  - Điểm trinh sát trung bình: Hiện vùng sáng nghi vấn (bán kính 3 ô).
