# Kế hoạch chuyển vị trí người chơi sang một bảng MMO

Trạng thái: **đề xuất triển khai**, chưa sửa code/schema. Mục tiêu sản phẩm: bỏ bản đồ cũ, mọi người chơi dùng MMO. Phạm vi của kế hoạch là **vị trí và quyền sở hữu nhân vật**, không gộp toàn bộ dữ liệu nhân vật vào bảng vị trí.

## Quyết định đích

Tạo `player_world_location` làm **bảng vị trí bền vững duy nhất**. Mỗi người chơi có đúng một hàng. Khi người chơi online, zone simulation giữ vị trí hiện tại trong RAM; bảng này là checkpoint và điểm khôi phục sau mất kết nối/restart. Không ghi PostgreSQL mỗi tick. Mọi thao tác cần xét vị trí tức thời của người chơi online phải hỏi zone owner; checkpoint chỉ dùng để khôi phục và cho tác vụ không đòi hỏi vị trí tức thời sau khi nhân vật offline.

| Dữ liệu | Nguồn sau cutover | Ghi chú |
| --- | --- | --- |
| Vùng logic `region_id` | `player_world_location` | FK `regions(id)`; dùng cho luật vùng, chat, nhiệm vụ. Không suy đoán vùng chỉ từ `zone_id` vì một map có thể chứa nhiều vùng. |
| MMO `zone_id`, `instance_id`, `x`, `y` | `player_world_location` | `x/y` là `DOUBLE PRECISION` theo đơn vị map MMO; `instance_id` là chuỗi zone instance. |
| `safe_x/y`, `map_version`, `content_version`, `updated_at` | `player_world_location` | Dùng để xác thực checkpoint và chọn spawn an toàn. |
| HP/shield checkpoint | `player_world_location` trong giai đoạn này | Giữ hành vi MMO hiện có; thiết kế HP chung với `players.hp` là việc riêng, phải tránh hai nguồn ghi cạnh tranh. |
| `players.current_region`, `players.x/y`, `players.map_instance_id` | Ngừng đọc/ghi | Chỉ dùng làm nguồn nhập dữ liệu một lần và có thể còn là cột vật lý tạm thời trên DB cũ. |
| `player_world_state_v2` | Ngừng đọc/ghi sau chuyển dữ liệu | Bảng nguồn của checkpoint MMO cũ; giữ bản sao lưu cho đến khi cổng xóa dữ liệu đạt. |

Không chọn `players` làm bảng đích: `x/y` hiện là số nguyên tọa độ cũ, `map_instance_id` là UUID bí cảnh cũ, còn MMO dùng tọa độ số thực và instance chuỗi. Không sao chép trực tiếp giữa hai hệ tọa độ. Chọn bảng mới thay vì thêm cột vào bảng đã chạy, phù hợp quy ước schema của repo: định nghĩa `CREATE TABLE IF NOT EXISTS` trong `backend/internal/db/schema_realtime.go`, đăng ký theo thứ tự FK trong `InitSchema`; không thêm `ALTER` lúc startup.

### Hợp đồng của bảng đích

`player_world_location`: `player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE`, `region_id VARCHAR(50) NOT NULL REFERENCES regions(id)`, `zone_id VARCHAR(80) NOT NULL`, `instance_id VARCHAR(80) NOT NULL`, `x/y DOUBLE PRECISION NOT NULL`, `safe_x/y DOUBLE PRECISION NOT NULL`, `hp/shield DOUBLE PRECISION NOT NULL`, `map_version/content_version VARCHAR(80) NOT NULL`, `revision BIGINT NOT NULL`, `updated_at TIMESTAMPTZ NOT NULL`. Thêm index theo `(zone_id, instance_id)` và `(region_id)` dựa trên các truy vấn đã chuyển; không nhân đôi các index của `players` một cách máy móc. Ràng buộc `CHECK` cho HP/shield không âm và tọa độ hữu hạn cần được xác nhận trên PostgreSQL; collision và map bounds vẫn do server kiểm tra theo map artifact.

`revision` tăng khi vị trí được commit. Lệnh ghi phải kèm ownership/fencing token hoặc so sánh revision để checkpoint cũ, worker muộn và kết quả zone transfer muộn không ghi đè trạng thái mới. Thiết kế token phải đi cùng cơ chế lease bền vững khi chạy nhiều process; process-local `OwnsCharacter` hiện chưa đủ.

## Các lỗ hổng hiện tại cần đóng trước cutover

- Join lấy `current_region`, `map_instance_id`, `x/y` từ `players`, sau đó ưu tiên `player_world_state_v2` nếu zone/version hợp lệ (`realtime_join_state.go`, `gateway/admission.go`). Checkpoint không đối chiếu `instance_id` với instance được chọn. Sửa join để chỉ đọc `player_world_location` và khôi phục đúng instance hoặc thực hiện handoff rõ ràng.
- Checkpoint MMO chỉ ghi `player_world_state_v2`; `players` tiếp tục chứa vị trí cũ (`realtime/persistence/state.go`). Sau leave, các handler cũ có thể hoạt động ở vị trí cũ. Xóa đường quay lại legacy trước khi bật MMO cho mọi người.
- `RequireLegacyWorldMode` chỉ chặn một số route và dựa vào lease trong process. Tổ đội còn có đường ghi `players.current_region/x/y`; tông môn, thương nhân, scouting, chat, profile và world index còn có đường đọc vị trí cũ.
- `world_index_sync.go` tái tạo toàn bộ chỉ mục người chơi từ `players` mỗi chu kỳ. MMO phải cung cấp nearby/online từ zone owner; bỏ luồng đồng bộ legacy này khi các consumer đã chuyển.
- `Manager.TransferCharacter` mới là primitive nội bộ; portal trigger, cập nhật vùng/DB và client handoff chưa hoàn chỉnh (`docs/mmo/transfer.md`). Không thể xóa `current_region` khỏi đường join trước khi có luồng chuyển zone thật.

### Ma trận chuyển các đường đang biết

| Đường hiện tại | Thay bằng | Điều kiện nghiệm thu |
| --- | --- | --- |
| `handlers/realtime_join_state.go`, `realtime/gateway/admission.go` | Load vị trí từ bảng đích, bind đúng instance; tài nguyên/stats vẫn từ `players` | Join không đọc cột vị trí legacy; checkpoint sai version/instance được xử lý tường minh |
| `realtime/persistence/state.go`, checkpoint worker trong `cmd/server/main.go` | Upsert bảng đích có revision/fencing | Checkpoint muộn không ghi đè transfer; leave thất bại không mất vị trí |
| `handlers/party.go` | Party join/range/follow dùng vị trí MMO; yêu cầu chuyển thành lệnh zone owner | Không có `UPDATE players SET ... x/y/current_region` từ party |
| `handlers/sect_warfare.go`, `merchant.go`, `scouting.go`, `ground_drops.go`, `spatial_realms.go` | Đọc vị trí hiệu lực; kiểm tra cùng zone **và instance** trước khoảng cách/tương tác | Không thể nhận quyền lợi ở tọa độ cũ hoặc từ instance khác |
| `handlers/world_index_sync.go`, `world_index.go`, `exploration_views.go`, `chat_events.go` | Nearby/online và kênh vùng theo zone owner hoặc projection MMO cùng instance | Không còn bóng ma từ `players`; chat đúng vùng và instance |
| `handlers/profile.go`, `player_events.go`, `models/player.go`, `services/player_service.go` | DTO vị trí đọc qua module vị trí; dữ liệu nhân vật vẫn qua `PlayerSvc` | Snapshot/response không tái xuất tọa độ cũ |
| `frontend/src/components/game/MMOBusinessPanels.tsx`, `PhaserWorldHost.tsx`, `app/page.tsx`, Zustand stores | Map panel/HUD lấy zone/region/vị trí MMO; join request dựa vào vị trí đã xác thực của server | Atlas và HUD khớp cùng một vị trí; client không tự chọn zone trái hàng đích |
| Đăng ký/onboarding, admin update, teleport, workers có SQL vị trí | Tạo hoặc chuyển một hàng bảng đích qua đường ghi thống nhất | Không còn writer vị trí ngoài đường này |

## Thứ tự triển khai

### P0 — Chặn ghi đồng thời và lập ma trận phụ thuộc

1. Lập danh sách **mọi** SQL đọc/ghi `players.current_region`, `map_instance_id`, `x`, `y` và mọi chỗ đọc `models.Player.CurrentRegion/X/Y`, frontend `player.current_region/x/y`. Phân loại: vị trí tức thời, vùng logic, hiển thị, spawn, chuyển vùng, quản trị, worker. Ghi chủ sở hữu dữ liệu và cách thay thế cho từng call site; dùng `rg` và review thủ công, không chỉ đếm chuỗi SQL.
2. Trong thời gian chuyển đổi, chặn route/worker cũ có thể đổi vị trí hoặc quyết định khoảng cách khi nhân vật do MMO sở hữu. Ưu tiên party join/follow teleport, tông môn, thương nhân, scouting, bí cảnh, admin teleport; xử lý giao dịch đang chạy trước khi cấp lease. Bất kỳ hoạt động chưa có ngữ nghĩa MMO phải trả lỗi rõ, không dùng `players` như fallback ngầm.
3. Gắn phép đo: số lần đọc vị trí legacy, số lần blocked action, checkpoint/lease lỗi, chênh lệch vị trí nguồn. Mọi lần đọc legacy sau khi cohort MMO bật phải truy ra được call site.

**Cổng P0:** không còn đường ghi vị trí legacy có thể chạy đồng thời với MMO; test hai tab, request đang chạy lúc join và worker chạy song song.

### P1 — Tạo bảng đích và chuyển dữ liệu một lần

1. Thêm schema `player_world_location` vào `schema_realtime.go`; cập nhật thứ tự `InitSchema` để `regions` và `players` tồn tại trước. Chỉ tạo bảng ở bản triển khai chuẩn bị; đăng ký/onboarding tiếp tục dùng đường cũ cho đến lúc đổi writer tại P2.
2. Viết công cụ chuyển dữ liệu **idempotent, chạy có chủ đích**, không auto-seed/auto-migrate khi khởi động. Chạy thử trên bản sao DB và đối soát khi traffic thật còn dùng đường cũ. Bản ghi thử không được trở thành nguồn đọc chính thức. Ở lần chuyển thật, mở maintenance window ngắn: ngừng admission MMO mới, drain checkpoint và ngừng mọi writer vị trí cũ; chụp/backup nguồn, chạy backfill cuối cùng, đổi toàn bộ reader/writer sang bảng đích, kiểm tra cổng P2 rồi mới mở traffic. Không để dữ liệu nguồn thay đổi giữa backfill cuối cùng và cutover.
3. Với người chơi có checkpoint v2: ưu tiên hàng v2 nếu `zone_id`, map/content version, tọa độ, vùng logic và instance đều hợp lệ. Xác định `region_id` từ dữ liệu vùng của nhân vật kết hợp mapping map; không gán mặc định `region_id = zone_id` cho mọi map. Với người chưa có checkpoint: đổi `players.x/y` sang MMO bằng `(legacy - origin) / scale` của map tương ứng, rồi kiểm tra bounds/collision; nếu không hợp lệ, dùng safe spawn do server chọn và ghi lý do. Tọa độ cũ không được copy nguyên số.
4. Backfill theo batch với `INSERT ... ON CONFLICT` có điều kiện: không ghi đè hàng đích có `revision` mới hơn. Đếm đủ một hàng cho mỗi người chơi đủ điều kiện; báo cáo checkpoint lỗi, mapping thiếu, instance không hợp lệ và spawn thay thế. Không xóa nguồn trước khi đối soát xong.

**Cổng P1:** trên DB sao chép, không có người chơi thiếu vị trí, zone/region hợp lệ, vị trí nằm trên ô đi được, bản sao lưu và báo cáo đối soát có thể chạy lại; bài test import chạy hai lần cho cùng kết quả. Traffic thật tiếp tục dùng đường cũ cho đến maintenance window P2.

### P2 — Chuyển quyền sở hữu vị trí trong backend

1. Tạo một module vị trí với interface nhỏ cho các handler: đọc `region/zone/instance/position` theo mục đích, kiểm tra cùng vị trí/khoảng cách, commit spawn/transfer/checkpoint. Adapter online lấy snapshot từ zone owner; adapter offline đọc `player_world_location`. Không thêm service/repository layer rộng cho gameplay; logic hành động vẫn ở handlers như quy ước repo.
2. Join chỉ đọc hàng đích. Server chọn hoặc khôi phục **đúng** instance; nếu instance đã mất, thực hiện chính sách chuyển sang instance còn sống qua safe spawn và commit revision mới trước khi chấp nhận input. Map/content version mới có đường rebasing được kiểm thử, không âm thầm dùng tọa độ cũ.
3. Checkpoint định kỳ/leave/drain ghi bảng đích với revision và fencing. Zone tick vẫn không ghi DB. Khi checkpoint thất bại, thử lại có giới hạn và báo lỗi; không xác nhận leave thành công nếu chưa commit. Nếu process chết trước commit, lease bền vững hết hạn/được fencing và lần join sau khôi phục **checkpoint cuối đã commit** (có thể mất tối đa khoảng thời gian checkpoint). DB không đọc được thì từ chối join thay vì đoán vị trí. Không giữ lease process-local vô thời hạn.
4. Hoàn thiện portal/zone transfer: validate điểm đến; reserve target ownership; commit `region_id`, `zone_id`, `instance_id`, `x/y`, safe spawn và revision như một chuyển trạng thái; đổi subscription/client map; rollback khi một bước thất bại. Xử lý duplicate transfer và reconnect giữa transfer.
5. Chuyển các tác vụ theo thứ tự rủi ro: (a) kiểm tra vị trí để cấp quyền/trao thưởng: tông môn, merchant, scouting, ground drops, bí cảnh; (b) thao tác di chuyển người khác: party và admin; (c) chat vùng, nearby/online/world index và worker; (d) profile, bản đồ, HUD/frontend. Hoạt động cần vị trí tức thời chỉ dùng snapshot online có lease hợp lệ, không dùng checkpoint 10 giây để kiểm tra khoảng cách.

**Cổng P2:** kiểm thử tích hợp join → move → tương tác gần/xa → transfer → disconnect → reconnect → restart; không có SQL vị trí sản phẩm nào đọc/ghi cột legacy, kể cả worker/admin/front-end projection. Race test cho lease, checkpoint, transfer; DB transaction test cho revision/fencing và rollback.

### P3 — Tắt bản đồ cũ và xóa đường phụ thuộc

1. Bật MMO cho cohort nhỏ, theo dõi chênh lệch rồi mở rộng cohort. Rollback bình thường là quay về bản MMO trước đó **đã đọc bảng đích**, giữ nguyên revision/lease. Nếu cần bật lại UI bản đồ cũ, phải có công cụ reverse projection `region/zone/instance/x/y` → legacy region/tọa độ và cập nhật `players` một lần trong maintenance window sau khi drain MMO; kiểm tra vị trí/spawn hợp lệ và diễn tập trước. Không bật legacy UI trực tiếp trên `players` đã cũ. Không cho một nhân vật đồng thời có legacy movement session và MMO lease.
2. Đổi trang mặc định sang MMO; bỏ `WorldCanvas`, đường `/explore/move*`, teleport/encounter/turn-combat của bản đồ cũ và worker chỉ phục vụ map cũ sau khi đã chứng minh không còn consumer. Phân biệt hành động gameplay vẫn cần giữ với renderer/transport cũ; không xóa tính năng chỉ vì nó từng ở màn hình cũ.
3. Ngừng đọc/ghi `player_world_state_v2` và vị trí trong `players`. Với DB mới, bỏ DDL tạo bảng v2 và các cột vị trí legacy khỏi `CREATE TABLE` cùng index tương ứng khi code không còn tham chiếu. DB đang tồn tại có thể còn bảng/cột vật lý không được đọc/ghi; muốn xóa vật lý cần đợt bảo trì schema riêng với backup và phê duyệt vận hành, vì repo hiện không dùng migration/ALTER lúc startup.
4. Cập nhật `docs/GAME_DATA.md` (schema), `docs/API.md` (route), `docs/DEVELOPMENT.md`/`README.md` (startup/flags), `WIKI.md` (di chuyển người chơi), `PROJECT.md` (scope) và tài liệu MMO recovery/cutover/transfer theo hành vi đã triển khai.

**Cổng hoàn tất:** một bảng `player_world_location` là nguồn vị trí bền vững duy nhất; không có đường chạy còn phụ thuộc dữ liệu vị trí ở `players` hoặc `player_world_state_v2`; rollback về MMO release trước đã diễn tập, và nếu còn hỗ trợ rollback sang legacy thì reverse projection cũng đã diễn tập; dữ liệu cũ có thể phục hồi từ backup; kiểm thử backend, race test, frontend lint/build và luồng browser với nhiều người chơi đều đạt.

## Kiểm thử bắt buộc theo rủi ro

- Backfill: checkpoint hợp lệ, checkpoint sai version/map/instance, vị trí ngoài map, không có checkpoint, hai lần chạy, hàng đích mới hơn dữ liệu nguồn.
- Đồng thời: checkpoint cũ đến sau transfer, disconnect trùng, hai tab cùng nhân vật, worker và lệnh quản trị cùng ghi, crash ngay trước/sau commit; revision/fencing phải ngăn rollback vị trí.
- Gameplay: merchant/tông môn/party/scouting chỉ nhìn thấy vị trí MMO thật; người khác không thấy bóng ma ở `players.x/y`; chat và nearby theo đúng instance.
- Recovery: DB mất kết nối, restart zone, version map đổi, instance biến mất, portal thất bại, replay checkpoint; nhân vật luôn xuất hiện ở safe spawn hợp lệ và không nhân đôi ownership.

## Phạm vi chưa được giải quyết bởi kế hoạch vị trí

`players.hp` và checkpoint MMO `hp/shield` hiện cũng có nguy cơ lệch. Kế hoạch này giữ cơ chế checkpoint HP đang có để không mở rộng thay đổi cùng lúc, nhưng P2 phải kiểm tra tất cả hoạt động đọc/ghi HP và chốt một owner trước khi mở MMO đại trà. Cơ chế lease bền vững giữa nhiều process và durable death/loot cũng là điều kiện vận hành riêng; `docs/mmo/cutover.md` và `docs/mmo/roadmap.md` hiện ghi chúng chưa hoàn tất.
