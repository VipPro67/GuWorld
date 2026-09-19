# WF-01: Research existing Worldmap presence, FOV, and profile access endpoints

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:research`  
**Status:** Closed  
**Blocked by:** None  

---

## Question

Khảo sát toàn diện hiện trạng mã nguồn liên quan đến:
1. **Quản lý vị trí & Tầm nhìn người chơi**: Cơ chế tọa độ `current_region`, `x`, `y` trong `backend/internal/handlers/exploration_*.go`, `docs/WORLDMAP_REALTIME.md`, và cách frontend hiển thị người chơi khác trong tầm nhìn FOV.
2. **Endpoint Profile hiện tại**: Phân tích API `GET /api/player/profile/:username` (`handlers/profile.go`) và `PUT /api/player/profile/settings` (`handlers/player_profile.go`), cùng các nơi gọi endpoint này trong frontend (`frontend/src/`).
3. **Cơ chế Đạo Ngân & Cảnh Giới hiện có**: Cấu trúc lưu trữ `players.dao_marks` và `players.path_masteries`, hàm `gamefx.PathMasteryLevel()`, và các hiệu ứng Cổ liên quan đến `Investigation`, `DisguiseLevel`, `PERFECT_DISGUISE`.

---

## Resolution

- **Worldmap & FOV**: Vị trí lưu tại `players (current_region, x, y)` và cache trên in-memory spatial index `worldPosIndex` (`world_index.go`, bucket 300 cells). Endpoint `GET /api/explore/players` kiểm tra bán kính quanh tọa độ thực tế của người chơi.
- **Profile Access**: `GET /api/player/profile/:username` cho phép bất kỳ ai gửi username để xem profile (có filter theo `scoutingLevel - disguiseLevel`). Được gọi trực tiếp từ Chat, Leaderboard, Arena, Sect, Party, Worldmap.
- **Dao Marks & Masteries**: `players.dao_marks` (JSONB) và `players.path_masteries` (5 cấp độ 0-4 theo `PathMasteryLevel`). Đã có ~100 Cổ hỗ trợ `Investigation` và `DisguiseLevel` (kèm cờ `PERFECT_DISGUISE` của Thái Độ Cổ).
- Chi tiết báo cáo lưu tại: `docs/wayfinder/research/WF-01-findings.md`.
