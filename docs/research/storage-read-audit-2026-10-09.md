# Kiểm tra lưu trữ và đọc dữ liệu game — 2026-10-09

## Phạm vi

Rà soát tĩnh các luồng đọc bản đồ, trạng thái người chơi, kho Gu/vật phẩm, polling
frontend và chỉ mục vị trí trong backend. Đây là danh sách ứng viên tối ưu từ mã
hiện tại, không phải kết luận hiệu năng production. Chưa đo request rate, p95/p99,
query plan trên dữ liệu thật hoặc kích thước bảng; không thêm index chỉ dựa vào
việc đọc thấy một query. Hiện chưa có baseline mới cho ngày rà soát này.

Workspace đã có sẵn các thay đổi ở `WIKI.md` và frontend trước lần rà soát này;
chúng không thuộc phạm vi và không được chỉnh sửa.

## Baseline request đã lưu

`docs/OPTIMIZATION_ROADMAP_2026-09.md` lưu snapshot production lịch sử với 91.831
request: p50 `14,48 ms`, p90 `69,48 ms`, p99 `329,55 ms`. Ở snapshot đó,
`/api/auth/me`, inventory, sect và notification nằm trong khoảng 5,6k–7,2k lượt;
max của `/api/auth/me` khoảng `9,4 s`. Snapshot được ghi trước các tối ưu sau đó,
không có timestamp window chi tiết ở phần báo cáo này và không thể dùng thay baseline
hiện tại. Nó làm `/api/auth/me` đáng ưu tiên profile trước, nhưng không chứng minh
fan-out hiện tại còn giữ cùng tần suất.

Trong workspace hiện tại chỉ có PostgreSQL và Redis container đang chạy, không có
backend app để lấy live request summary/traces; vì vậy chưa thể lấy baseline mới từ
monitor runtime.

## Query plan cục bộ

PostgreSQL cục bộ đang chạy nên đã lấy `EXPLAIN (ANALYZE, BUFFERS)` cho các query
đọc inventory Gu, vật phẩm, và snapshot player của world index. Dữ liệu hiện quá nhỏ
để đại diện tải thật: 3 player, 2 Gu, 99 dòng vật phẩm, 36 ground drop; chỉ một
player có inventory (tối đa 2 Gu và 99 dòng vật phẩm).

- Query Gu dùng sequential scan + sort, đọc 2 dòng, hoàn tất khoảng `0.13 ms`.
- Query vật phẩm dùng sequential scan + sort, đọc 99 dòng, hoàn tất khoảng
  `0.16 ms` (sort dùng 35 kB).
- Query world index dùng sequential scan trên 3 player, đọc 1 dòng hoạt động,
  hoàn tất khoảng `0.08 ms`.

Các plan này không chứng minh các query sẽ nhanh ở production; chúng chỉ cho thấy
local fixture chưa đủ lớn để đánh giá index. Không thêm index từ kết quả này. Muốn
quyết định cần staging/production-like dataset, `EXPLAIN (ANALYZE, BUFFERS)` với
player đại diện và theo dõi số dòng/latency theo percentile.

`pg_stat_statements` chưa được cài trong DB cục bộ; `pg_stat_database.stats_reset`
cũng không có mốc reset. Vì vậy hiện không có lịch sử query có thể quy về API hoặc
so sánh trước/sau. Các bộ đếm theo bảng/index cục bộ không đủ để ưu tiên tối ưu
production.

## Phát hiện và hướng xử lý

### Mã tiềm năng không được mount: `ExplorationMap`

`ExplorationMap.tsx` chứa polling grid và tải ground drops toàn region, nhưng `rg`
không tìm thấy import/render nào của component; trang chính dùng `WorldPage` và
`WorldCanvas`. Vì vậy request trong component này hiện không phải tải runtime, không
xếp nó vào hotspot đang hoạt động. Dù vậy, request ground drops của component đã
được đổi sang dùng tâm và `vision_radius` mà grid trả về, để không tái tạo truy vấn
toàn region nếu component được mount lại.

**Bằng chứng mã:** tìm kiếm toàn `frontend/src` chỉ thấy định nghĩa component và
không có callsite; `frontend/src/app/page.tsx` render `WorldPage`.

### P1 — Polling không giới hạn request đang chạy

Trước khi sửa, `usePolling` gọi callback bằng `setInterval` mà không đợi Promise
trước hoàn tất. Hook được dùng bởi `MineView` và `TrackingRadarDrawer` đang mount;
`ExplorationMap` là callsite không hoạt động.

**Đã xử lý một phần:** `frontend/src/lib/usePolling.ts` hiện đợi Promise của lượt
poll trước kết thúc rồi mới cho callback timer chạy tiếp. Điều này chặn chồng request
từ chính interval của mỗi hook. Nó chưa chặn các request khởi tạo thủ công hoặc
request phụ được callback gọi mà không `await`; stale response/cancellation vẫn cần
được xử lý theo từng luồng. Theo dõi riêng số request đồng thời và stale response.
Đây cũng là mục liên quan đến phần “request lifecycle” trong
`docs/REQUEST_PERFORMANCE_OPTIMIZATION_GUIDE.md`.

### P2 — Đồng bộ world position dựng lại toàn bộ snapshot mỗi 30 giây

`StartWorldIndexSyncWorker` tải mọi player hoạt động, tạo lại hai map `byID` và
`grid`, rồi thay toàn bộ position index. Vị trí đã được cập nhật eager trong luồng
realtime nên đây là đường bootstrap/self-heal; chi phí DB, allocation và thời gian
giữ lock tăng theo số người chơi đang hoạt động.

**Hướng tối ưu:** ghi nhận thời gian query/reconcile, số dòng đọc, allocation và
thời gian chặn reader; chỉ khi các chỉ số này đáng kể mới cân nhắc tăng chu kỳ hoặc
đối soát incremental. Query hiện có index riêng cho `last_activity` và
`last_mana_update`; predicate `OR` cần xác nhận bằng `EXPLAIN (ANALYZE, BUFFERS)`
trên volume đại diện trước khi thay index/query.

**Bằng chứng mã:** `backend/internal/handlers/world_index_sync.go`;
`backend/internal/handlers/world_index.go` (`Reconcile`);
`backend/internal/db/schema_players.go` (các index activity).

### P2 — Cell rectangle endpoint lặp công việc theo từng ô khi di chuyển

Luồng đang mount trong `WorldCanvas` gọi `fetchNearbyCells` sau khi vị trí player
đổi, ngoài reconciliation định kỳ. Mỗi request `/api/explore/cells/rect` duyệt mọi
tọa độ trong hình chữ nhật. Ở luồng UI hiện tại, `GetGrid` cap investigation radius
ở 25 và `WorldPage` đặt radius đó vào store, nên rectangle tối đa thường là
`51 × 51` (2.601) ô. Store có hard cap 127 cho caller khác, nhưng hiện không tìm
thấy UI nào đặt radius cao hơn 25. Mỗi ô lại xác định region trên map, tra persistent
cell và sinh nội dung procedural nếu cần; sau đó cleared-state được lấy theo một
Redis `MGET` batch. Đây chủ yếu là CPU/Redis work, không phải query DB.

DB cục bộ có 68 dòng `regions`; nếu registry runtime có cùng số zone, một rectangle
đầy đủ cỡ UI có thể đi qua tối đa khoảng `2.601 × 68 = 176.868` lần kiểm tra zone
đơn giản (ước lượng upper bound, chưa đo CPU và không tính các vòng lặp dừng sớm).
Đây chỉ là dấu hiệu để benchmark: registry runtime có thể khác DB cục bộ và phần lớn
zone thường bị loại nhanh theo map/bounds.

**Hướng tối ưu:** đo thời gian handler theo diện tích rectangle và số zones; nếu có
đuôi chậm, ưu tiên giảm vùng giao nhau giữa các lần tải (coverage/tile cache có TTL
và refresh khi WS invalidation hoặc reconciliation) hoặc tính region/persistent-cell
snapshot theo cả rectangle. Giữ batch `MGET`; không thay bằng lookup Redis cho từng ô.

Ở chế độ `full`, lúc effect khởi động gọi `pollCells()` và `prefetch()` liền nhau.
Polling tải viewport theo radius, còn prefetch tải tile hiện tại (và tile phía trước
khi đang đi), nên hai rectangle có vùng giao nhau và handler có thể resolve lại các
ô đó. Effect còn phụ thuộc `investigationRange`; nếu `fetchGrid` đổi range sau mount,
effect khởi động lại và phát thêm một lượt polling/prefetch. Đây là overlap chắc chắn
theo luồng điều khiển nhưng chưa đo được phần trăm ô trùng hoặc chi phí thực tế.
Trước khi bỏ polling hay thay TTL, cần đo số ô giao nhau/request và bảo đảm vùng rỗng
cũng được xem là đã tải, đồng thời xử lý invalidation/respawn realtime.

**Bằng chứng mã:** `frontend/src/components/WorldCanvas.tsx` (request khi tọa độ
player đổi, `pollCells`/`prefetch` lúc mở map và lịch reconciliation);
`frontend/src/store/useWorldStore.ts`
(`fetchNearbyCells`, giới hạn radius 127); `backend/internal/handlers/exploration_grid.go`
(`MaxMapVisionRadius = 25`); `backend/internal/handlers/exploration_views.go`
(`GetCellsRect`, giới hạn 256 mỗi cạnh và batch `filterClearedCells`);
`backend/internal/handlers/zone_registry.go` (`FindZoneAtOnMap`, `PersistentCellAt`).

### Ground drops tải toàn vùng dù camera chỉ hiển thị viewport — đã xử lý

Trước đó `WorldPage` gọi `GET /api/world/drops` không có tọa độ. Handler khi đó
đọc mọi drop còn hạn trong region và lấy cả JSONB `loot_data`, còn `WorldCanvas`
chỉ vẽ những marker nằm trong camera. Endpoint đã hỗ trợ `x/y/radius`, nhưng UI
không truyền viewport; `fetchGrid` đồng thời ghi cùng state bằng một snapshot nhỏ
hơn, tạo race ghi đè giữa hai request.

`WorldCanvas` hiện tính tâm và bán kính bao viewport từ kích thước canvas, scale và
stage translation. Các đổi camera được debounce 300 ms; request mới nhất mới được
phép cập nhật state. Backend giới hạn radius tối đa 127 ô, trả 400 cho query sai;
caller cũ không truyền tham số giờ nhận viewport radius 25 quanh player thay vì
toàn region. `ExplorationMap` gửi
tọa độ cùng `vision_radius` của grid. Tâm viewport ngoài bản đồ chỉ trả empty khi
khung nhìn không giao với world bounds, để camera sát mép vẫn thấy các ô giao nhau.
`fetchGrid` không còn
ghi đè `groundDrops`; sau khi loot thành công marker bị xóa ngay từ response xác
nhận thành công, nên không cần tải lại cả region.

**Bằng chứng mã:** `frontend/src/components/WorldCanvas.tsx` (viewport camera);
`frontend/src/store/slices/inventorySlice.ts` (query và latest-request gate);
`frontend/src/components/WorldPage.tsx` (bỏ full-region mount fetch);
`backend/internal/handlers/ground_drops.go` (radius cap và query bounds);
`backend/internal/db/schema_world.go` (index theo region/tọa độ và region/expiry).

### GET bản đồ môn phái trùng khi mở world map — đã xử lý

`WorldPage` gọi `fetchSectsMap()` trong effect khởi tạo và `WorldCanvas` cũng gọi
cùng endpoint khi mount. Ngoài ra, effect khởi tạo trước đó phụ thuộc
`regionsLoaded`, nên khi `fetchRegions()` hoàn tất cả nhóm request của world page có
thể chạy lần nữa. `WorldCanvas` cũng từng gọi riêng `fetchRegions()` và
`fetchMyTeleportArrays()` dù parent `WorldPage` đã gọi cả hai; thêm vào đó, effect
cũ còn tải lại teleport arrays khi `regionsLoaded` đổi.

**Đã xử lý:** `WorldPage` giờ tải regions bằng effect riêng; effect tải dữ liệu bản
đồ không còn phụ thuộc `regionsLoaded`. `WorldCanvas` không tự tải sect map nữa;
`WorldPage` là owner duy nhất cho lần tải này. Các mutation môn phái vẫn có thể
invalidate/refetch dữ liệu theo luồng hiện có. `WorldPage` cũng là owner duy nhất
cho regions và teleport arrays; `WorldCanvas` chỉ tải persistent objects. Như vậy
mount canvas không tạo request regions/teleport dư, và cập nhật `regionsLoaded` không
kích hoạt thêm request không liên quan.

**Bằng chứng mã:** `frontend/src/components/WorldPage.tsx`;
`frontend/src/components/WorldCanvas.tsx`; `WorldCanvas` chỉ được render từ
`WorldPage`.

### P2 — GET kho Gu có bước kiểm tra hunger thêm một lần đọc DB

`GetPlayerInventoryGus` gọi `NeedsHungerEvaluation` để đọc mốc tick trước khi tải
toàn bộ Gu. Đây là kiểm tra rẻ và chỉ mở transaction khi tick đến hạn, nhưng mỗi
lần GET vẫn có thêm một round-trip DB. Chưa đủ bằng chứng để bỏ vì hunger cần giữ
đúng khi người chơi mở kho sau thời gian dài.

**Hướng tối ưu:** đo tần suất gọi endpoint và phần trăm người chơi đến hạn tick.
Nếu request bị gọi lặp từ frontend, ưu tiên giảm/dedupe GET trước; chỉ gộp kiểm tra
hunger vào lần đọc Gu nếu vẫn cần tối ưu và có thể bảo toàn cập nhật decay atomic.

**Bằng chứng mã:** `backend/internal/handlers/player_feed.go`;
`backend/internal/worker/hunger.go` (`NeedsHungerEvaluation`);
`backend/internal/db/player_gu.go` (`LoadPlayerGus`).

### P2/P3 — Kiểm tra query plan khi tải inventory Gu

`LoadPlayerGus` lọc `player_id`, loại `location = 'ACTOR'`, có thể lọc thêm
`equipped OR is_vital`, và sắp `created_at, id`. Schema có các index bắt đầu bằng
`player_id`, nhưng chưa thấy index khớp toàn bộ thứ tự sắp xếp cho query tải đầy
đủ. Tác động phụ thuộc số Gu mỗi người chơi và planner; index bổ sung cũng làm
tăng chi phí ghi.

**Hướng tối ưu:** lấy `EXPLAIN (ANALYZE, BUFFERS)`, số Gu/player và sort spill trên
dữ liệu đại diện cho cả query đầy đủ và equipped-only. Chỉ cân nhắc index mới nếu
sort/scan đo được đáng kể và lợi ích vượt chi phí duy trì index.

**Bằng chứng mã:** `backend/internal/db/player_gu.go` (`LoadPlayerGus`);
`backend/internal/db/schema_player_items.go` (index `player_gu`).

### GET trạng thái vật hóa model rộng rồi trim JSON — đã xử lý

`GetPlayerStatus` gọi projection dùng `loadRow`, tải nhiều cột trạng thái cùng
materials/equipped Gus. Trước đây handler marshal toàn bộ `models.Player`, unmarshal
thành map JSON, xóa field rồi `c.JSON` marshal map lần nữa.

**Đã xử lý:** response view nhúng `models.Player` và shadow 10 field server-only bằng
field `omitempty`. Handler marshal view trực tiếp một lần, không còn round-trip qua
`map[string]json.RawMessage`; field còn lại giữ nguyên. Unit test so JSON output với
cách trim cũ trên dữ liệu mẫu có giá trị ở cả field bị loại và field được giữ.
Phần query vẫn load full projection; chỉ serialization CPU/allocation được giảm.

**Bằng chứng mã:** `backend/internal/handlers/player_status.go`
(`GetPlayerStatus`, `writeTrimmedPlayerStatus`);
`backend/internal/services/player_service.go` (`loadRow`, `ProjectStatus`).

### P2 — `GET /sect/members` có N+1 khi tính power thành viên

Handler đọc sect ID và danh sách thành viên bằng hai query, sau đó với mỗi thành
viên gọi `PlayerSvc.LoadEquipped`. Mỗi lần gọi đọc player row, inventory materials,
và equipped Gu; tiếp theo `computeMemberPower` → `CalculatePlayerStats` gọi
`CheckPlayerSectTerritory` thêm query theo player. Như vậy chi phí đọc tối thiểu là
`4 × số_thành_viên + 2` round-trip, chưa tính query dao marks cho rank 6+, validate
leader khi có active combat, hoặc cleanup status hết hạn.

**Đã xử lý:** endpoint hiện đọc sect config và danh sách thành viên một lần, rồi
batch-load các trường player cần cho stats/territory và equipped ACTIVE Gu. Territory
bonus được tính trong memory với cùng `region/x/y`, membership và main hall level.
Số query đọc chính giảm còn 4 dù số thành viên tăng; không còn tải materials,
combat state hoặc chạy cleanup status hết hạn trong đường hiển thị power. `Active()`
vẫn bỏ status hết hạn khi tính stats; worker status expiry tiếp tục quản lý dữ liệu
lưu. Nếu batch stats lỗi, power mặc định 0 như trường hợp không load được player
trước đây; lỗi đọc Gu vẫn bỏ phần Gu và tính từ stat còn lại.

Test mới xác nhận công thức power có territory override dùng đúng buff main hall;
`cd backend && go test ./...` đã qua sau thay đổi này.

**Bằng chứng mã:** `backend/internal/handlers/sect_members.go` (`GetSectMembers`);
`backend/internal/services/player_service.go` (`LoadEquipped`, `loadRow`);
`backend/internal/db/player_gu.go` (`LoadPlayerGus`);
`backend/internal/handlers/sect_warfare.go` (`computeMemberPower`);
`backend/internal/handlers/player_stats.go` (`GetPassiveStats`);
`backend/internal/handlers/sect_territory_check.go` (`CheckPlayerSectTerritory`).

### P2 — `/api/auth/me` dựng session qua nhiều lần đọc DB

`GetCurrentPlayer` gọi `ProjectAndPersistBestEffort(..., true)`. Phần projection
đọc row `players`, rồi `LoadEquipped` đọc tiếp `player_inventory_items` và
`player_gu`; player rank từ 6 trở lên còn đọc dao marks và aperture riêng. Trước
đây handler đọc completed quests và recipe progress bằng hai query riêng. Đường
session thường có ít nhất 5 query cho player thường và 7 cho immortal, chưa tính
truy vấn phụ chỉ xảy ra khi dọn status hết hạn, active combat cần validate party,
hoặc admin cache miss.

**Đã xử lý một phần:** quest hoàn thành và recipe progress giờ được đọc bằng một
`UNION ALL` trong `loadSessionProgress`, giảm một round-trip mỗi lần `/auth/me` chạy
(thường 5→4; immortal thường 7→6). Hai danh sách giữ điều kiện lọc, thứ tự và các
trường response cũ. Nếu query gộp lỗi, handler fallback sang hai reader hiện hữu.
Query đã chạy thành công trên PostgreSQL local để kiểm tra kiểu và cú pháp; DB test
RDS vẫn chưa truy cập được. Các query player/materials/Gu/dao/aperture vẫn riêng.

`refreshPlayerSummary()` cũng dùng endpoint này nên chi phí lặp lại sau một số action,
dù frontend đã có single-flight cho các lần gọi đồng thời. Các callsite hiện thấy gồm
party, scouting, instance, NPC cultivate và khi trade đóng; chúng không nhất thiết
chạy cùng lúc nên single-flight không loại bỏ được các lần đọc nối tiếp.

**Tiếp theo:** trace query count/latency và bytes cho `/auth/me` tách bootstrap với
refresh. Nếu DB round-trip là phần đáng kể, cân nhắc batch vật phẩm/Gu và dao/aperture;
giữ nguyên shape và độ mới của player, collections và tiến độ. Không gọi `/auth/me`
để cập nhật domain đã có mutation response authoritative.

**Bằng chứng mã:** `backend/internal/handlers/player_auth_login.go`
(`GetCurrentPlayer`); `backend/internal/services/player_service.go`
(`LoadEquipped`, `loadRow`, `ProjectStatusEquipped`);
`backend/internal/handlers/respond.go` (`loadCompletedQuests`, `loadUnlockedRecipes`);
`frontend/src/store/slices/profileSlice.ts` (`initializeSession`,
`refreshPlayerSummary`); `frontend/src/store/slices/{party,scouting,instance,aperture}Slice.ts`
và trade completion trong `frontend/src/store/useGameStore.ts`.

### P1 — Kiểm tra sức chứa kho đọc lại inventory và territory trong từng ô di chuyển

`MovePlayerPath` đi tối đa 5 ô và gọi `resolveCellArrival` cho từng ô. Mỗi lần vào
ô trước đây còn đọc lại `map_instance_id` và chạy territory query; nếu ô cho tài
nguyên hoặc Cổ trùng, `HasExplorationInventorySpace` lại gọi `CalculatePlayerStats`
(một territory query khác) và
`InventorySvc.ListByPlayer` (đọc toàn bộ inventory qua `*sql.DB`, ngoài transaction
đang xử lý đường đi). Ngoài round-trip thừa, query kho ngoài transaction không thấy
được vật phẩm mới chưa commit từ các ô trước; tính capacity giữa các bước có thể dùng
snapshot cũ.

**Đã xử lý:** capacity dùng `len(player.GuInventory)` cộng số item ID duy nhất trong
`player.MaterialsInventory`, map đã được `loadRow` tổng hợp từ inventory trong cùng
transaction. Sau khi thêm tài nguyên, handler cập nhật map ngay trong model để ô tiếp
theo thấy slot mới. Tính stats nhận `playerTerritory` đã truy vấn cho ô hiện tại, bỏ
territory query lặp khi tính capacity. `map_instance_id` được nạp một lần trước vòng
path; player row đang bị khóa trong transaction nên giá trị này không thể đổi bởi
transaction cạnh tranh. Territory config vẫn được đọc theo từng ô để giữ behavior
READ COMMITTED nếu membership/cấu hình lãnh địa đổi giữa lúc đi. Chỉ giữ fallback
đọc inventory nếu snapshot map nil vì không tải được ở `loadRow`.

Test mới xác nhận ngưỡng capacity dựa trên inventory snapshot; focused test và
`cd backend && go test ./...` đều đã qua sau thay đổi.

**Bằng chứng mã:** `backend/internal/handlers/exploration_move.go`
(`resolveCellArrivalWithInstance`, `MovePlayerPath`); `backend/internal/handlers/exploration_grid.go`
(`HasExplorationInventorySpace`); `backend/internal/services/player_service.go`
(`loadRow`).

### P2 — World map tải toàn bộ ground drops của region

`WorldPage` gọi `fetchGroundDrops()` không có tọa độ nên `/api/world/drops` trả mọi
drop chưa hết hạn trong region; `WorldCanvas.drawGroundDrops` tạo graphic và label
cho từng kết quả. Cùng lúc đó `fetchGrid()` trả drop trong viewport 5×5 và cũng ghi
đè `groundDrops`; hai request mount độc lập nên kết quả cuối phụ thuộc thứ tự response.
Camera giữ vùng quan sát trong AOI khoảng 254 ô và được căn lại theo người chơi khi
di chuyển, nên toàn bộ region có thể vượt xa dữ liệu đang hiển thị. Endpoint hỗ trợ
tọa độ/radius, nhưng hiện radius không bị chặn và frontend không truyền phạm vi.

DB local chỉ có 36 drop tổng cộng, tất cả đã hết hạn, loot JSON trung bình 373 bytes
(tối đa 1.506 bytes); dữ liệu này không đủ để benchmark lượng row đang hoạt động hay
response production. Chưa thay đổi behavior ở đây vì cần xác nhận tương tác pan/zoom
và cách cập nhật marker khi người chơi di chuyển hoặc người khác nhặt drop.

**Hướng tối ưu:** tải theo vùng nhìn thực tế (có cap radius khớp camera AOI), debounce
khi camera/player đổi vùng, và merge/cache theo vùng để không làm rơi marker ở request
viewport khác. Sau khi loot thành công, xóa đúng drop khỏi store; không tải lại toàn
region nếu chỉ cần đồng bộ bản ghi vừa bị xóa. Cần bảo đảm drop mới/đã bị người khác
nhặt vẫn được cập nhật qua refresh có giới hạn hoặc event.

**Bằng chứng mã:** `frontend/src/components/WorldPage.tsx` (mount loader);
`frontend/src/store/slices/inventorySlice.ts` (`fetchGroundDrops`, `lootGroundDrop`);
`frontend/src/components/WorldCanvas.tsx` (`drawGroundDrops`, AOI/camera scale);
`backend/internal/handlers/ground_drops.go` (`GetTileGroundDrops`);
`backend/internal/handlers/exploration_grid.go` (viewport drops).

### P2 — Middleware trace giữ response body không giới hạn

Trước sửa, `monitor.Middleware` bọc mọi HTTP response bằng `bytes.Buffer`, giữ nguyên
body không giới hạn trên heap và lưu response của tối đa 500 trace; mỗi lần gọi
`Body()` còn tạo string để gắn vào trace. Đây là overhead chung cho GET lớn của game,
không phải thời gian truy vấn PostgreSQL. Response body còn nằm trong trace detail
admin dù logging thông thường chỉ ghi response body cho HTTP lỗi.

**Đã xử lý:** buffer response trace bị giới hạn ở 8 KiB/request. Trace giữ prefix và
thêm `response_body_truncated`; khi body JSON lỗi bị cắt và không parse được,
`error_message` trả marker thay vì coi prefix là thông báo lỗi hoàn chỉnh. Phần body
được giữ bởi ring trace do đó bị chặn ở tối đa khoảng 4 MiB mỗi process, chưa tính
metadata/string overhead. Trace còn ghi `response_bytes` bằng kích thước thực gửi từ
Gin writer; tab Admin hiển thị kích thước này cạnh latency. Nhờ vậy có thể phát hiện
GET có payload lớn mà không cần giữ response đầy đủ trong bộ nhớ. Contract được ghi
trong `docs/API.md`.

**Bằng chứng mã:** `backend/internal/monitor/monitor.go`
(`loggingResponseWriter`, `Middleware`, `RequestTrace`);
`backend/internal/server/server.go` (HTTP error/completed logs).

### P3 — World progress JSONB đi theo mỗi lần đọc và ghi movement

`loadRow` lấy `cleared_cells` và `visited_subzones` cho cả status/session projection.
`savePlayerMovement` cũng đưa cả hai JSONB vào câu `UPDATE` cho mỗi lần di chuyển,
dù `visited_subzones` chỉ đổi khi vào sub-zone mới và `cleared_cells` có cooldown.
Local DB hiện chỉ có 3 player; giá trị lớn nhất đo được là 48 bytes cho
`cleared_cells` và 221 bytes cho `visited_subzones`, nên hiện không có cơ sở normalize
chúng hay thay schema. `docs/PLAYER_PERSISTENCE_OPTIMIZATION_PLAN.md` đã yêu cầu đo
kích thước và tần suất ghi trước khi tách hot player row.

**Tiếp theo:** trên production-like data đo p95/max kích thước hai cột, tần suất
movement không làm thay đổi từng trường, WAL/TOAST bytes và thời gian query. Nếu
chúng tăng đáng kể, chỉ cập nhật JSONB khi giá trị đổi; chỉ cân nhắc bảng riêng khi
đo được row/TOAST write amplification lớn và có thể giữ nguyên đọc quest/achievement.

**Bằng chứng mã:** `backend/internal/services/player_service.go` (`loadRow`);
`backend/internal/handlers/exploration_world.go` (`savePlayerMovement`);
`backend/internal/handlers/zone_registry.go` (`recordVisitedSubZone`).

### P2 — Một số GIN index JSONB trên `players` có vẻ không phục vụ query hiện tại

`schema_players.go` tạo GIN cho `active_combat`, `statuses`, `equipped_gus`,
`path_masteries` và `body_mods`. Rà SQL trong backend: `statuses` có predicate
JSONPath (`@?`) ở `worker/status_expiry.go`; các cột `active_combat`, `equipped_gus`,
`path_masteries`, `body_mods` hiện không có predicate GIN-compatible. Các phép đọc
`active_combat` còn lại đều theo `players.id` hoặc kiểm tra `IS NOT NULL`, không dùng
GIN này.

DB local cho các index ứng viên `active_combat`, `equipped_gus`, `path_masteries` và
`body_mods` báo `idx_scan = 0`, chiếm lần lượt 672, 40, 40 và 24 KiB (tổng 776 KiB).
Tuy vậy DB chỉ có 3 player và `pg_stat_database.stats_reset` là NULL; con số scan 0
không đại diện production. GIN index duy trì thêm dữ liệu/index write khi JSONB đổi,
nên các index này đáng kiểm tra trên production-like workload trước khi quyết định.

**Hướng tối ưu:** xem `pg_stat_user_indexes` sau một cửa sổ traffic đại diện, đối chiếu
mọi query ngoài backend nếu có; kiểm tra index size, WAL/write rate và query plan.
Chỉ gỡ index khi chứng minh không có query/consumer cần nó. Giữ `statuses` riêng vì
có JSONPath predicate trong mã hiện tại.

**Bằng chứng mã:** `backend/internal/db/schema_players.go` (GIN index definitions);
`backend/internal/worker/status_expiry.go` (predicate dùng GIN trên `statuses`);
`backend/internal/handlers/tribulation_suppression.go` (JSON field access trên row
đã chọn bằng primary key).

## Thứ tự làm tiếp

1. Thu baseline cho các GET nóng: request/user/phút, p50/p95/p99, response bytes,
   DB pool wait, query count/latency và số request đồng thời.
2. Thiết kế fetch ground drops theo AOI/camera và xử lý freshness khi di chuyển, loot,
   hoặc nhận cập nhật từ người chơi khác.
3. Profile phần query còn lại của `/api/auth/me` và `MovePlayerPath`; kiểm tra tần suất
   gọi `refreshPlayerSummary()` theo action và reuse movement context an toàn trong 5 ô.
4. Với `LoadPlayerGus`, world index và player status, lấy query plan/profiling trước
   khi thay schema hoặc projection.
5. Đối chiếu usage/size của GIN JSONB index trên cửa sổ traffic production-like;
   không dùng `idx_scan` local hiện tại làm căn cứ gỡ index.

## Thay đổi áp dụng sau audit

Trong `frontend/src/store/slices/apertureSlice.ts`, các thao tác NPC đã đổi sang
refresh đúng domain:

- `cultivateNPC` tải lại NPC và player summary; bỏ hai GET không liên quan là kho Gu
  và unread notifications.
- `equipNPCGu` và `unequipNPCGu` tải lại NPC cùng kho Gu; không gọi lại toàn bộ
  session/player summary và unread notifications.

Trong `frontend/src/store/slices/petsSlice.ts`, `captureMonster` áp dụng inventory
và tài nguyên từ mutation response; danh sách pet chỉ tải lại nếu capture thành
công. Handler capture dùng `respondActionGuOnly`, trả inventory/tài nguyên và phát
player snapshot qua WebSocket, nên không cần gọi `initializeSession()` sau mỗi lần
thử bắt (kể cả thất bại).

Đây là giảm fan-out dựa trên response hiện tại của các handler: cultivate cần đồng
bộ tài nguyên player, còn equip/unequip cần đồng bộ membership Gu và trạng thái NPC.
Capture failure không tạo pet; capture success chỉ cần thêm một GET danh sách pet.
Trong `frontend/src/lib/usePolling.ts`, callback async của cùng một hook được chạy
single-flight; timer bỏ qua tick khi lượt trước còn chạy.
Trong `WorldPage`/`WorldCanvas`, sect map chỉ có một owner khởi tạo và việc regions
tải xong không còn kích hoạt lại các GET khác của world page.
Trong `WorldPage`/`WorldCanvas`, `WorldPage` là owner tải regions và teleport arrays;
`WorldCanvas` chỉ tải persistent objects. Điều này loại request regions/teleport dư
khi canvas mount và tránh request khác bị phát lại theo transition `regionsLoaded`.
Trong `backend/internal/monitor/monitor.go`, response body giữ cho mỗi trace bị chặn
ở 8 KiB; trace có cờ báo truncation và số byte response thực gửi. Tab monitoring Admin
hiển thị số byte cạnh latency để có thể ưu tiên endpoint payload lớn.
`cd backend && go test ./...` đã qua sau thay đổi giới hạn response trace.
`cd frontend && npm run build` đã qua; prebuild có ghi lại `src/data/wiki-cache.json`,
đã hoàn nguyên artifact đó sau khi xác nhận nó phát sinh từ lệnh build. ESLint toàn
frontend còn thất bại ở 3 lỗi ref trong `CombatScreen.tsx` và `MovementJoystick.tsx`,
cùng một cảnh báo directive thừa trong `TrackingRadarDrawer.tsx`; các file này không
thuộc thay đổi audit hiện tại. `git diff --check` đã qua. Sau lần chỉnh tiếp
`WorldCanvas.tsx`, ESLint chạy riêng file và frontend production build đều qua.
Sau tối ưu serializer `/api/player/status`, `cd backend && go test ./...` cũng qua,
bao gồm test so payload mới với hợp đồng JSON cũ.

## Nguyên tắc dữ liệu

- PostgreSQL tiếp tục là nguồn sự thật cho tiền, vật phẩm và tiến độ quan trọng.
- Không chuyển inventory/currency sang Redis-only để giảm độ trễ.
- Không thêm index hoặc cache dữ liệu người chơi trước khi có query plan, tần suất,
  freshness và invalidation rõ ràng.
- Các thay đổi schema tuân theo quy ước hiện tại của dự án và phải bảo toàn transaction,
  quyền sở hữu dữ liệu, cùng invariant kinh tế game.

## Kiểm tra DB test ngày 2026-10-09

Đã thử mở kết nối read-only tới RDS được cấu hình trong `backend/.env`, dùng SSL,
giới hạn connect timeout 6 giây, statement timeout 8 giây và chỉ chạy các truy vấn
đọc metadata/thống kê. DNS phân giải được nhưng TCP tới PostgreSQL port 5432 hết
thời gian chờ trước khi vào được DB; không có truy vấn hay thay đổi nào chạy trên
database này. Vì vậy chưa lấy được table/index size, `pg_stat_user_indexes` hoặc
`pg_stat_statements` production-like. Cần cho phép kết nối từ môi trường audit trong
security group/VPN rồi chạy lại; không nên mở DB công khai chỉ để audit.
