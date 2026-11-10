# Chuyển combat realtime sang turn-based

## Trạng thái đã xác nhận

- Combat PvE chuẩn hiện đi qua `POST /api/combat/start` và simulator `backend/internal/simulator/`; `CombatScreen` phát lại log theo lượt.
- Zone simulator combat 20 Hz đã không còn trong `backend/internal`; `schemaRealtime()` xóa các bảng session realtime cũ. Gu skill profile với cast geometry/cooldown mili giây đã bị gỡ khỏi CSV, registry, database và Wiki.
- Các handler Môn Khiếu và worker giải đấu vẫn còn truy vấn `realtime_combat_session_participants` / `realtime_combat_sessions` dù schema đã xóa chúng. Những truy vấn này đã được gỡ; Môn Khiếu hiện chặn thao tác dựa trên `players.active_combat`, còn trận giải đấu `IN_PROGRESS` được đưa lại `PENDING` khi reconcile để engine giải đấu xử lý tiếp.

## Còn thiếu hoặc chưa được nối

- UI thế giới Phaser vẫn giữ di chuyển và tương tác phi chiến đấu. Combat realtime, hotbar thi triển, auto-farm và AI quái đuổi đánh đã bị gỡ. Di chuyển tới ô có quái chưa tự mở encounter; actor quái trong Redis chưa được nối với simulator.
- Các cột realtime skill loadout/hotbar, model/snapshot/mutation/API, combat hotbar keybinds và Gu `skill_profile` đã bị gỡ. Schema idempotent xóa các cột cũ trên DB hiện hữu; Gu CSV/import/export không còn profile cast realtime.
- Profile Sát Chiêu/Trận Kích không còn trường windup/cooldown mili giây hoặc payload `realtime_effects`; schema dọn các khóa cũ khỏi JSONB đã lưu. API và state client cho hotbar kỹ năng realtime 9–12 cũng đã bị gỡ.
- `/api/combat/practice` hiện tạo encounter giả lập xác định theo lượt và được giữ như tiện ích tập luyện.
- Cooldown/warmup lượt cho Sát Chiêu/Trận Kích chưa được định nghĩa đồng đều; dữ liệu profile cũ có thể không có các trường `cooldown_turns`/`warmup_turns`. Các effect ngoài tập damage/heal/shield cơ bản, target selection theo hàng/cột và PvP cần được rà soát trước khi khẳng định đã chuyển đầy đủ.
- Tai kiếp đã có flow encounter/kết toán turn-based qua `/engage`, simulator và `/resolve`; chuyển động quái theo tick, damage node và worker auto-close cũ đã bị gỡ. Mô hình chiến trường/node-defense chưa có luật turn-based.
- Các mode có PvP/đấu trường cần kiểm tra riêng về settlement, semantics và UI; chưa được audit end-to-end. Những gap này được ghi nhận, không được xem là combat realtime đang chạy.

## Ranh giới

Realtime chat, notifications, presence và đồng bộ sự kiện không phải combat và không thuộc phạm vi gỡ bỏ này. Dữ liệu quái, loot, bản đồ và combat simulator theo lượt cần được giữ lại. Thay đổi schema phải có DDL tương thích cho DB hiện hữu trong `backend/internal/db/schema_*.go`.
