# Audit endpoint bản đồ và chiến đấu — 2026-10-10

## Phạm vi và bằng chứng

Rà luồng từ caller frontend qua route/handler đến DB, Redis và payload response.
Đây là audit cấu trúc mã; chưa có p95/p99 hoặc query plan trên dữ liệu RDS đại diện.
Không đề xuất index mới chỉ từ việc đọc query. DB test trước đó không nhận kết nối TCP
đến PostgreSQL nên không có truy vấn nào chạy trên DB đó.

## Đã tối ưu

### `/explore/grid` cho WorldPage chỉ tải stat range

`WorldPage` chỉ dùng `stats.movement` và `stats.investigation`, nhưng trước đây vẫn
tạo 5×5 cells, kiểm tra cleared state, lấy nearby players/party và truy vấn/giải mã
ground drops. Màn canvas đã có request riêng cho các vùng dữ liệu đó.

`WorldPage` hiện gọi `GET /explore/grid?stats_only=true`. Nhánh này không dựng cells,
nearby players hoặc ground drops và không persist mana; cleanup `cleared_cells` vẫn giữ
nguyên để bảo toàn hành vi dọn dữ liệu. Nó vẫn dùng player/equipped projection hiện có,
nên còn dư địa tối ưu projection riêng nếu profiling xác nhận DB đọc quá rộng. Endpoint
mặc định giữ payload đầy đủ cho caller cũ.

### Cleared state trong grid dùng một Redis MGET

Grid đầy đủ có tối đa 25 ô. Trước đó mỗi ô gọi `IsCellCleared` riêng. Handler hiện
dùng `GetClearedCells` một lần; nếu MGET lỗi, dùng lại đường đọc từng ô để giữ hành vi
fallback cũ. Trên request thành công, Redis round-trip cho phần này giảm từ tối đa 25
xuống 1.

### GET cài đặt chiến đấu không còn dựng player projection

`GetCombatSettings` chỉ cần JSONB `combat_settings`, nhưng trước đó mở transaction,
đánh giá mana, đọc player projection/vật phẩm/Gu equipped rồi mới trả cấu hình. Handler
hiện đọc riêng cột `combat_settings` bằng một `QueryRowContext`; không đổi response JSON.

### PvP movement tái sử dụng territory đã đọc

Luồng tấn công đã có `SectTerritoryInfo` của attacker và target để kiểm tra bảo hộ.
Tính `attack_range` và target combat stats giờ dùng lại hai snapshot đó thay vì gọi
`CalculatePlayerStats` và truy vấn territory thêm lần nữa. Tối đa bỏ 2 query cho một
lần PvP trigger; các query kiểm tra territory ban đầu vẫn giữ nguyên.

## Ưu tiên tiếp theo

### P1 — Monster snapshot materialize và prune boss toàn region

`GET /world/monsters` giới hạn spawn thường theo AOI, nhưng đọc mọi boss `map_cells`
trong scope region, materialize từng boss vào Redis rồi quét boss-set để prune. Client
có thể yêu cầu snapshot sau mỗi 5 giây. Chỉ boss gần người chơi được trả về, nhưng phần
đọc/materialize/prune rộng vẫn chạy.

Đã gộp phần đọc actor từ boss-set thành một Redis MGET sau SMEMBERS, áp dụng cả cho
prune trong request và `WorldBossesInRegion` mà worker gọi mỗi 250 ms. Số actor và
logic prune vẫn giữ nguyên. Redis script materialize cũng trả luôn JSON actor hiện có,
bỏ GET thứ hai cho mỗi spawn và trả đúng `created=false` để không broadcast sự kiện
spawn lặp. Các EVAL theo từng boss cùng truy vấn DB toàn region trong request còn là
phần tốn kém cần tách ở bước sau. Các EVAL boss hiện được gửi theo Redis pipeline,
tối đa 64 boss mỗi batch: mỗi script vẫn nguyên tử, nhưng không còn một network
round-trip riêng cho từng boss trong request snapshot.

Không lọc query theo tọa độ đơn thuần: prune hiện cần danh sách đầy đủ spawn hợp lệ;
worker cũng dùng boss actors trong toàn region cho AI. Muốn tối ưu phải tách stale-spawn
reconciliation khỏi read request và vẫn bảo toàn boss actor đang di chuyển.

**Mã:** `backend/internal/handlers/world_monsters.go` (`GetWorldMonsters`);
`backend/internal/cache/world_monster.go` (`PruneWorldBosses`, `WorldBossesInRegion`);
`frontend/src/store/useWorldStore.ts` (`fetchMonsterSnapshot`).

### P1 — Combat giữ transaction/khóa player suốt lúc mô phỏng

`AutoCombatStart` tải player bằng đường `FOR UPDATE`, dựng combatants, chạy simulator,
áp dụng toàn bộ tiêu thụ/loot và commit. Với trận dài, thời gian CPU mô phỏng nằm trong
khoảng giữ khóa nên các thao tác khác trên player có thể phải đợi. Chưa tách mô phỏng
ra ngoài transaction vì cần thiết kế idempotency và optimistic guard theo version/active
combat để ngăn hai request cùng hoàn tất một trận.

**Mã:** `backend/internal/handlers/combat_auto.go`;
`backend/internal/handlers/combat_sim_build.go`;
`backend/internal/services/player_service.go` (`LoadForUpdate`).

Không thể chỉ commit trước `simulator.Simulate`: khi bỏ khóa, retreat/capture, request
resolve thứ hai hoặc thay đổi combat state có thể chen vào và khiến kết quả/thưởng/tiêu
hao bị áp trùng hoặc ghi lên trận đã đổi. Tách an toàn cần claim resolution bền vững,
token idempotent, bước finalize xác nhận claim và guard ở các đường có thể kết thúc/thay
đổi trận. `players.version` hay chỉ so `active_combat` không đủ để phát hiện mọi thay
đổi Gu/inventory/party.

`AutoCombatStart` hiện ghi cảnh báo khi transaction thành công kéo dài ít nhất 1 giây,
với thời gian nạp player, dựng simulator state, chạy simulator và tổng thời gian tx.
Dùng số đo này để xác nhận phần nào chiếm thời gian trước khi thay đổi ranh giới nguyên
tử. Request lỗi sớm không được tính bởi cảnh báo slow-combat này.

### P1/P2 — PvP party nạp lại cùng player/inventory nhiều lần

Khởi tạo combat và build simulator nạp party players tuần tự; player đã được đọc lúc
khởi tạo có thể được đọc lại, rồi mỗi combatant lại đọc killer moves và healing-item
inventory. Tối đa 8 ally làm số round-trip tăng theo party size. Hướng tối ưu là batch
snapshot theo player ID và reuse dữ liệu đã load, sau khi xác nhận các quy tắc HP lúc bắt
đầu trận, status, loadout và quyền sở hữu.

**Mã:** `backend/internal/handlers/combat_state.go`;
`backend/internal/handlers/combat_sim_build.go`;
`backend/internal/handlers/combat_auto.go` (`buildSimCombatant`);
`backend/internal/handlers/combat_finalize.go`.

Đã bỏ đọc toàn bộ `player_inventory_items` trong `buildSimCombatant` khi combatant
không bật `AutoHealHP`, vì simulator chỉ dùng heal charges/potency khi cờ này bật.
`StartCombat` cũng không còn `SELECT EXISTS` riêng cho từng ally trước thông báo:
ally đã được tải để tạo trận và `notify.Send` bảo vệ insert bằng savepoint nếu FK đổi
giữa chừng. Việc đọc lại ally khi mô phỏng trong request sau vẫn giữ vì Gu, status và
loadout có thể đã đổi; HP và vị trí tiếp tục lấy từ snapshot lúc bắt đầu trận.
`buildStatProvenance` nay chỉ chạy lần tính stat nền (kèm lookup lãnh địa) khi có
status điều chỉnh stat theo phần trăm cần giải thích trong replay; cooldown hoặc buff
phẳng không còn kích hoạt lần tính lại này.

### P2 — Cells rectangle có work bound lớn

`GET /explore/cells/rect` cho phép tới 256×256 tọa độ. Mỗi tọa độ dò region rồi resolve
cell. Đã chuyển zone lookup sang snapshot các zone giao với rectangle, giữ thứ tự và
quy tắc priority/area khi chồng nhau; instance dùng luôn region của instance. Như vậy
không còn quét toàn registry và lấy khóa zone cho từng tọa độ ở bước chọn region.
UI thường xin diện tích nhỏ hơn, nhưng public endpoint vẫn có giới hạn tối đa 65.536
tọa độ. Caller sau `useGuOnMap` trước đây gửi radius 300 rồi clamp còn 127, tương
đương tối đa 255×255. Hiện refresh này dùng `investigationRange` 10–25, tức 21×21
đến 51×51, và tải riêng một ô khi hạ mục tiêu ngoài vùng đó. Boss và địa điểm cố định
được tải qua `GetExploreObjects`; marker quái/tài nguyên thủ tục chỉ hiện trong tầm
điều tra. Đường teleport phụ cũng dùng tầm điều tra. `ResolveCellInRegion` vẫn xử lý
từng ô và Redis cleared-state vẫn dùng MGET. Cần đo latency/request thực tế nếu public
endpoint còn được gọi với rectangle lớn.

**Mã:** `backend/internal/handlers/exploration_views.go` (`GetCellsRect`);
`backend/internal/handlers/zone_registry.go` (`FindZoneAtOnMap`);
`frontend/src/store/useWorldStore.ts` (`fetchNearbyCells`);
`frontend/src/components/WorldCanvas.tsx` (poll/prefetch).

### P2 — Nearby players có caller radius 300

`GetNearbyPlayers` dùng index memory, nhưng radius 300 có thể thăm nhiều bucket hơn
viewport thường thấy. Một số movement/teleport path gọi radius 300; trước khi giảm cần
xác nhận chúng cần hiển thị party ở xa hay chỉ đồng bộ người trong viewport.

**Mã:** `backend/internal/handlers/exploration_views.go` (`GetNearbyPlayers`);
`frontend/src/store/useWorldStore.ts` (movement/teleport callers).

### P2 — Combat history đọc JSONB logs chỉ để tính số bước

Danh sách lịch sử chỉ trả summary nhưng query còn chạy `jsonb_array_length(logs)` cho
20 bản ghi. `logs` có thể là JSONB lớn/TOAST; index `(player_id, created_at DESC)` đã
hỗ trợ lọc và sắp xếp nên chưa cần thêm index. Cân nhắc ghi `logs_count` cùng lịch sử
hoặc metadata summary, có fallback cho dữ liệu cũ; cần theo dõi kích thước payload trước
khi đổi schema.

**Mã:** `backend/internal/handlers/combat_history.go` (`ListCombatHistory`);
`backend/internal/db/schema_combat.go`.

### P2 — PvP history lặp serialize và lưu trùng replay

Sau PvP, cùng logs/summary trước đây được marshal và insert lại theo từng người tham
gia. Đã chuyển sang marshal logs/summary một lần, combatants theo hai góc nhìn một lần
mỗi phe, rồi ghi tất cả người tham gia bằng một `INSERT ... SELECT` với hai mảng UUID.
Nếu batch lỗi FK vì người chơi bị xóa, code quay về insert từng người để giữ history cho
người còn tồn tại. Tối ưu lưu trữ dài hạn vẫn là một combat record dùng chung cùng
perspective theo player; cần đo dung lượng history trước khi đổi schema.
History vẫn ghi trong goroutine không được theo dõi khi shutdown; vì vậy server dừng
ngay sau combat có thể mất replay. Cần một bước thiết kế lưu bền vững nếu history là
dữ liệu bắt buộc, thay vì chỉ tối ưu thời gian INSERT.

**Mã:** `backend/internal/handlers/combat_history.go` (`SaveCombatHistory`,
`savePVPCombatHistory`); `backend/internal/handlers/combat_auto.go`;
`backend/internal/handlers/combat_auto_resolve.go`.

## Đã kiểm chứng từ schema

- `map_ground_drops` đã có index `(region_id, x, y)` và `(region_id, expires_at)`;
  query plan/active-drop cardinality thực tế vẫn cần đo trước khi thay.
- `map_cells` có unique index `(region_id, x, y)`.
- `monster_spawns` có spatial lookup theo `(region_id, x, y)`.
- Combat history có `(player_id, created_at DESC)`; lookup detail theo primary key.
- Arena leaderboard sắp theo `rank`, cột đã unique; không thêm index trùng.

## Kiểm tra sau thay đổi

`cd backend && go test ./...` thành công. ESLint chạy trên các file frontend đã đổi
(`WorldPage.tsx`, `ExplorationMap.tsx`, `inventorySlice.ts`) thành công; production build
`next build --webpack` hoàn tất cả TypeScript và static generation. Cần chạy
benchmark query và đo response bytes trên DB/request load đại diện trước khi chốt các
thay đổi cấu trúc kế tiếp.

Các thay đổi rectangle và PvP history tiếp theo đã qua `cd backend && go build ./...`
và `git diff --check`; chưa đo latency hoặc dung lượng replay trên môi trường test.
Batch boss và refresh ô sau dùng Gu đã qua `go build ./...`, ESLint cho
`useWorldStore.ts`, `next build --webpack` và `git diff --check`.
Tối ưu provenance combat tiếp theo cũng đã qua `go build ./...` và diff check.

## Bổ sung: thời điểm commit và số đo snapshot boss

`AutoCombatStart` và bốn caller của `AutoResolveCombat` nay đánh dấu ô đã dọn trong
Redis và phát sự kiện liên quan sau khi transaction commit thành công. Các caller
auto-resolve cũng chỉ bắt đầu ghi history sau commit, tránh replay cho trận bị rollback.
Trận PVE thắng không tạo goroutine history vì theo chính sách hiện tại không lưu replay
cho trường hợp này.
Thời gian giữ khóa hàng player giảm được phần thao tác Redis này; chưa có số đo runtime
để định lượng mức giảm. History vẫn là ghi bất đồng bộ nên chưa đảm bảo bền vững khi
server dừng ngay sau trận.

Log chậm của `AutoCombatStart` đã tách `begin`, `load`, `state_build`, `simulation`,
`outcome`, `commit` để phân biệt chờ kết nối/khóa với xử lý kết quả. `GetWorldMonsters`
ghi log khi snapshot thành công mất từ 500 ms, gồm thời gian tải spawn/boss,
materialize, prune, đọc bucket, số boss và số lỗi cache. Sau khi có log tải thực,
quyết định tiếp việc cache danh sách boss vùng hay tối ưu bucket theo phần tốn thời
gian nhất, không giới hạn boss theo AOI vì AI vẫn cần duy trì actor toàn vùng.
