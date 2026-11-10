# Kích Hoạt Chạm Trán & Tích Hợp Turn-based Combat Trên Bản Đồ
Status: resolved
Type: grilling
Blocked by: 02
Part of #map

## Question

Khi người chơi di chuyển hoặc thám hiểm thế giới, cơ chế kích hoạt Turn-based Combat (`CombatScreen.tsx` và `backend/internal/simulator`) sẽ diễn ra như thế nào?
- Khi chạm vào ô quái vật (Cell step encounter) hay nhấn nút "Tấn công / Khiêu chiến" tại Node?
- Quái vật và Boss trên bản đồ có khả năng "Chủ động phục kích / Aggro" theo phạm vi hay chỉ là encounter bị động?
- Xử lý trạng thái người chơi khi đang trong trận chiến: Khóa di chuyển ở bản đồ chính, tạm lưu session chiến đấu vào DB/Redis, và khi kết thúc (Thắng/Thua/Bỏ chạy) thì hồi vị trí hoặc văng về điểm hồi sinh an toàn như thế nào?

## Answer

1. **Phương thức kích hoạt trận chiến:**
   - **Tại Node cấp Vực:** Người chơi xem danh sách quái vật/thực thể tại địa danh và nhấn nút **"Khiêu chiến / Tấn công"**.
   - **Trong Tactical Grid:** Ô quái vật hiển thị rõ trên bản đồ sau khi sương mù được giải tỏa. Người chơi bước vào ô kề cận có thể chọn **"Tấn công"**. Riêng các loài Hung Thú hung hãn hoặc Boss thủ lĩnh có tỷ lệ **Chủ động phục kích (Aggro Ambush)** nếu người chơi bước vào bán kính lãnh địa của chúng.

2. **Bảo tồn cơ chế Skip / Quét Trận (Fast Auto-resolve):**
   - Giữ cơ chế **Skip Combat**: Khi người chơi có chênh lệch tu vi/chiến lực vượt trội so với quái vật hoặc muốn tiết kiệm thời gian, có thể bấm nút **"Bỏ qua / Kết thúc nhanh (Skip)"**.
   - Backend sử dụng [`backend/internal/handlers/combat_auto_resolve.go`](file:///home/vippro67/GitHub/GuWorld/backend/internal/handlers/combat_auto_resolve.go) để mô phỏng toàn bộ lượt đấu tức thì, trừ HP/chân nguyên tương ứng và trả ngay kết quả chiến thắng/chiến lợi phẩm mà không cần tải diễn hoạt từng hiệp.

3. **Quản lý Trạng thái Trận đấu & Hậu quả:**
   - Khi vào trận, khởi tạo bản ghi session chiến đấu (`combat_sessions`), khóa quyền di chuyển của nhân vật trên bản đồ.
   - **Thắng trận:** Nhận phần thưởng vào túi đồ qua cơ chế Idempotent Loot Grant (`combat_loot_grants`), ô quái trong `map_cells` cập nhật `last_cleared_at = NOW()` và đếm ngược thời gian hồi sinh (`respawn_seconds`).
   - **Bỏ chạy (Flee):** Khả năng đào tẩu phụ thuộc vào Cổ di chuyển và chênh lệch thân pháp; nếu thành công nhân vật lùi về ô an toàn trước đó.
   - **Thất bại / Tử vong:** Nhân vật bị hất văng về Điểm An Toàn gần nhất (Hub thành trấn hoặc Sơn trại môn phái), nhận trạng thái trọng thương cần đả tọa hồi phục.

