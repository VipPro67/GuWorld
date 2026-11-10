# Audit & Ranh Giới Deprecation Của Realtime Stack
Status: resolved
Type: research
Part of #map

## Question

Những thành phần nào trong `backend/internal/realtime/` (Zone Sim 20Hz, WTTransport / WebTransport QUIC, Spatial Index, Gateway Admission) cần xóa bỏ hoặc vô hiệu hóa hoàn toàn, và những thành phần nào (như Loot Idempotent `realtime_loot_grants`, Gu Catalog Compiler, Monster Template Data, Boss Threat logic) có thể được giữ lại hoặc tái cấu trúc cho hệ thống World Turn-based mới? Ranh giới chuyển đổi mã nguồn giữa hai hệ thống sẽ được phân định như thế nào để không làm gãy các tính năng phụ thuộc (như Môn Phái, Độ Kiếp, Arena)?

## Answer

1. **Các thành phần tháo dỡ / loại bỏ hoàn toàn (Deprecate & Remove):**
   - `backend/internal/realtime/sim/`: Toàn bộ vòng lặp tick 20Hz (`runner.go`, `world.go`, `combat.go`, `ai.go`, `vitals.go`, `transformation.go`).
   - `backend/internal/realtime/wttransport/`: Server WebTransport (QUIC UDP) và multiplexing binary stream `0x01` / `0x02`.
   - `backend/internal/realtime/spatial/`: Cấu trúc chỉ mục không gian 2D liên tục (`spatial.Index`).
   - `backend/internal/realtime/gateway/`: Hệ thống Lease ownership (`admission.go`), checkpoint định kỳ 10s và sweeper.
   - Frontend: Xóa bỏ `frontend/src/game/` (Phaser scenes, `WorldRuntime.ts`, `WorldTransport.ts`, `MapLoader.ts`) và `PhaserWorldHost.tsx`.

2. **Các thành phần giữ lại và tái cấu trúc cho Turn-based:**
   - **Database & Dữ liệu:** Giữ nguyên các bảng `regions`, `map_cells`, `monster_spawns`, `monster_templates`, `primeval_veins`, `world_maps`.
   - **Logic Loot Idempotent:** Bảng `realtime_loot_grants` được tái sử dụng thành bảng ghi nhận chiến lợi phẩm idempotent cho trận chiến Turn-based (`combat_loot_grants`).
   - **Hệ thống Combat Turn-based:** Khôi phục `backend/internal/simulator/` (engine tính toán lượt, hiệu ứng Cổ trùng, sát chiêu, buff/debuff) và kết nối với `backend/internal/handlers/combat_*.go`.
   - **Mạng giao tiếp:** Giữ lại WebSocket / SSE tiêu chuẩn cho Chat, Thông báo, Biến cố và Đồng bộ Presence.

3. **Chiến lược phân ranh giới chuyển đổi:**
   - Tắt flag `WT_ENABLED` và `REALTIME_WORLD_ENABLED`.
   - Chuyển `backend/cmd/server/main.go` về chạy HTTP Gin Server tiêu chuẩn, loại bỏ việc khởi tạo `WorldManager` và `WebTransportServer`.
   - Các tính năng phụ thuộc (Độ Kiếp `tribulations`, Môn Phái, Thí Luyện) chuyển sang gọi trực tiếp Turn-based Combat Session thay vì mount vào Zone Sim.

