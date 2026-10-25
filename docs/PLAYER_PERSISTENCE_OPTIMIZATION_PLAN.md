# Kế hoạch tối ưu lưu trữ player và xử lý lock timeout

**Ngày lập:** 2026-09-11  
**Phạm vi:** PostgreSQL, Redis write-behind, transaction gameplay, worker nền và các luồng đồng bộ trạng thái player.  
**Mục tiêu:** lưu đúng và đủ dữ liệu, không để snapshot cũ ghi đè trạng thái mới, giảm thời gian giữ row lock và loại bỏ phần lớn `55P03`/`40P01` mà không tăng `lock_timeout` để che vấn đề.

**Tiến độ triển khai:** Batch đầu tiên đã chuyển movement/runtime stats sang commit đồng bộ trong transaction nghiệp vụ; legacy Redis payload vẫn được xử lý trong cửa sổ tương thích bằng version/timestamp guard và compare-and-ACK. Growth đã thống nhất lock `players -> player_gu`, passive-stats dùng projection read-only và full player loader không còn lấy connection DB thứ hai trong transaction. Các hạng mục domain table/CAS đầy đủ, idempotency/outbox và refactor combat vẫn theo backlog bên dưới.
**Cập nhật 2026-10-26:** `internal/playerstate` đã gom shape đọc của `players` (column list, revision, Gu/materials, repair, not-found) và `services.PlayerService` chỉ còn phần time projection. Phần ghi/unit-of-work và phân quyền field vẫn nằm trong backlog của kế hoạch này.

## 1. Kết luận điều hành

`players` hiện là một **hot aggregate**: thông tin tài khoản, hồ sơ, tài nguyên, HP/mana, vị trí, combat và nhiều JSONB cùng nằm trên một tuple. PostgreSQL khóa theo row chứ không theo column, vì vậy heartbeat, di chuyển, combat, hunger, growth và giao dịch vẫn tranh cùng một row dù cập nhật các trường khác nhau.

Ba nguyên nhân phải xử lý trước:

1. **Thứ tự lock không thống nhất:** hunger khóa `players` rồi `player_gu`, trong khi growth cập nhật `player_gu` rồi `players`; các luồng nhiều người chơi cũng khóa theo thứ tự nghiệp vụ thay vì UUID.
2. **Transaction giữ lock quá lâu:** combat/hunger khóa player trước khi mô phỏng, dựng JSON, xử lý loot, notification và nhiều truy vấn phụ. Một số query còn lấy connection thứ hai trong khi transaction đang giữ connection/lock đầu tiên.
3. **Write-behind có thể mất hoặc ghi đè dữ liệu:** movement và stats cùng ghi row `players`; payload có `version` nhưng SQL chưa dùng CAS. Chu trình `đọc pending -> ghi DB -> SREM` không atomic nên có thể ACK nhầm một update mới vừa tới.

Không chọn giải pháp “tăng `DB_LOCK_TIMEOUT_MS`”. Timeout dài hơn chỉ làm request và connection chờ lâu hơn. Đích kiến trúc là **transaction ngắn, ownership từng field rõ ràng, CAS theo từng domain, lock theo thứ tự cố định và queue có claim/ack an toàn**.

## 2. Mục tiêu định lượng và bất biến dữ liệu

### 2.1 SLO sau tối ưu

| Chỉ số | Mục tiêu production | Cách đo |
|---|---:|---|
| Tỷ lệ transaction lỗi `55P03` | `< 0,1%` mutation trong 15 phút | Counter theo route/operation |
| Deadlock `40P01` | `0` trong vận hành bình thường | PostgreSQL log + app metric |
| Thời gian giữ transaction p95/p99 | `< 100 ms` / `< 500 ms` với OLTP thông thường | Histogram từ begin đến commit/rollback |
| Pool acquire wait p95 | `< 20 ms` | App histogram/tracing quanh `Conn`/`BeginTx`; DBStats delta chỉ cho mean/saturation |
| Tuổi pending write p99 | `< 15 s` | Redis enqueue timestamp |
| Lost update trong stress test | `0` | Đối soát event/result với DB |
| Stale response áp vào frontend | `0` | `domain_version` monotonic |

Ngưỡng trên là điểm khởi đầu; chốt baseline thực tế trong 48 giờ đầu rồi điều chỉnh nếu hạ tầng yêu cầu.

### 2.2 Bất biến bắt buộc

- Số dư đá, HP, mana, Nhân Khí và vật phẩm không được âm ngoài luật game.
- Một mutation thành công phải **atomic** trên tất cả bảng liên quan; thất bại không được để trạng thái nửa chừng.
- Retry cùng `idempotency_key` không được trừ/cộng tài nguyên lần hai.
- Snapshot async cũ không được ghi đè mutation sync mới.
- Mọi flow khóa nhiều player phải khóa theo UUID tăng dần; sau player mới đến child rows theo khóa chính tăng dần.
- Không gọi Redis, publish event, HTTP hoặc công việc CPU nặng trong lúc giữ transaction DB.
- Chỉ trả thành công sau khi durable state cần thiết đã commit. Event realtime được phát sau commit hoặc qua outbox.

## 3. Hiện trạng và rủi ro cụ thể

### 3.1 Một row chứa quá nhiều miền ghi

Bảng `players` đang trộn:

- **Identity/account:** `username`, `password_hash`, email/Google, referral, session, admin.
- **Profile/cold state:** race, gender, constitution, avatar, bio, disguise/title/frame.
- **Runtime resources:** rank, mana, stones, HP, lifespan, thought, stress, Human Qi.
- **World:** region, x/y, visited subzones, cleared cells, activity.
- **Combat/config:** `active_combat`, equipped Gus, settings, statuses, path masteries và growth data.

Các GIN index trên JSONB làm mỗi lần sửa JSON tương ứng tốn thêm WAL/index maintenance. Quan trọng hơn, mọi `UPDATE players` đều cần khóa cùng tuple.

### 3.2 Deadlock và thời gian giữ lock

- Hunger lấy `players ... FOR UPDATE`, sau đó thao tác `player_gu`; growth làm chiều ngược lại. Đây là vòng lock trực tiếp.
- Combat mở transaction và khóa actor sớm, sau đó mới mô phỏng, xử lý target/allies/mine/loot và commit ở cuối.
- Trade đã có pattern khóa hai player theo UUID; pattern này chưa được áp dụng nhất quán cho combat, marketplace, party và worker batch.
- Nhiều nơi dùng `DB.Begin()` nên request bị hủy không tự hủy transaction. Timeout toàn cục hiện chỉ là hàng rào cuối, không phải quản lý transaction theo use case.
- `PlayerService.loadRow` có thể đọc inventory bằng `s.db` thay vì query handle/transaction đang truyền vào. Flow mutation vì vậy vừa có snapshot không nhất quán, vừa giữ một connection trong lúc chờ connection thứ hai.
- `GetPassiveStatsHandler` dùng transaction và rollback ở cuối cho một luồng đọc/projection. Nếu evaluator có lazy write/lock, endpoint polling này tạo contention nhưng không lưu kết quả.

### 3.3 Write-behind Redis chưa có ranh giới ownership an toàn

- `player_movement` chứa cả HP/mana/stones/version nhưng SQL thực tế chỉ lưu vị trí/progress/Human Qi. Contract payload và contract lưu DB không khớp.
- Movement ghi `human_qi` theo snapshot nên có thể ghi đè Nhân Khí mới từ mutation khác.
- Stats dùng `last_mana_update` làm điều kiện nhưng không kiểm tra `RowsAffected`; một update không ghi row vẫn bị xem là thành công và pending ID bị xóa.
- Worker stats tự tăng `players.version`, trong khi service tăng version trong object ngay cả lúc mới enqueue; version API và version durable có thể lệch.
- Queue dùng một Redis key mới nhất cộng một set pending. Nếu producer ghi bản mới giữa lúc consumer đọc payload và `SREM`, consumer xóa luôn marker của bản mới.
- `flushAll` duyệt map handler không có thứ tự ổn định; movement và stats có thể tranh cùng player row mỗi 5 giây.

### 3.4 Batch làm khuếch đại blocking

`last_activity` và một số worker cập nhật nhiều player trong một statement. Chỉ cần một row bị khóa, statement có thể giữ các lock đã lấy trên những row khác trong khi chờ, làm vùng ảnh hưởng lớn hơn. Input từ map/array cũng không đảm bảo cùng thứ tự với flow nhiều player khác.

### 3.5 Frontend tạo request chồng lấn

- Status có polling toàn cục, trong khi tab growth tiếp tục gọi status và passive stats theo interval riêng.
- Grid reload theo interval và theo thay đổi tọa độ có thể overlap.
- Helper “dedupe” chưa được dùng cho các GET nóng; cơ chế timestamp/abort hiện tại không bảo đảm response cũ không apply.
- Một boolean `actionLoading` toàn cục không đủ bảo vệ multi-tab, retry sau timeout hoặc hai loại mutation cùng chạy.
- Snapshot/WS chưa có kiểm tra version theo domain, vì vậy response cũ có thể merge lên state mới.

## 4. Mô hình ownership dữ liệu đích

Mỗi field chỉ có **một writer authority**. “Async” chỉ dùng cho state có thể coalesce theo last-write-wins và có CAS riêng; tài sản hoặc hành động có side effect phải commit đồng bộ.

| Nhóm | Trường tiêu biểu | Source of truth | Cách ghi | Quy tắc merge/concurrency |
|---|---|---|---|---|
| Account/auth | username, password hash, email, Google ID, session, admin | PostgreSQL | Sync | Unique constraint; không đi write-behind |
| Profile | race, gender, constitution, avatar, bio, disguise/title/frame | PostgreSQL | Sync; debounce settings không quan trọng | CAS `profile_version` hoặc atomic column update |
| Currency | primeval/immortal stones, Human Qi, wallet | PostgreSQL | Sync trong transaction | Atomic delta/check (`SET balance=balance+$n`, guard đủ tiền); ledger/idempotency |
| Runtime stats | HP, mana, thought, lifespan, stress, timestamp regen | PostgreSQL | Sync với mutation; projection đọc không ghi | CAS `runtime_version`; không nhận full stale snapshot |
| Location | region, x/y | PostgreSQL durable + Redis latest cache | Async coalesced nếu move thuần; sync khi teleport/combat | CAS `location_version`; payload chỉ chứa location |
| Map progress | cleared cells, visited subzones | PostgreSQL | Sync cho unlock quan trọng; có thể async theo operation | Set-union/upsert, không replace toàn JSON bằng snapshot |
| Combat | active combat, combat result | PostgreSQL | Sync | `combat_version`; state machine transition có expected state/version |
| Inventory Gu/items | `player_gu`, inventory item tables | PostgreSQL normalized | Sync | Row delta/upsert; lock child theo ID; không rebuild full snapshot nếu không cần |
| Status/achievement | trạng thái có expiry, unlock | Child table | Sync | Unique key/upsert; grant/revoke không overwrite cả array |
| Presence/activity | last activity | Redis TTL; PostgreSQL chỉ snapshot phục vụ báo cáo | Async/best effort | Không được tranh hot runtime row |
| Realtime event | snapshot/delta WS | Redis Pub/Sub/cache | Sau commit | Mang domain version; frontend bỏ event cũ |

### 4.1 Vertical partition đề xuất

Không tách big-bang. Tạo dần các bảng 1:1 theo write affinity:

1. `player_runtime_stats(player_id PK/FK, hp, mana, thought, lifespan, stress, regen timestamps, version, updated_at)`.
2. `player_resources(player_id PK/FK, primeval_stones, immortal_stones, human_qi, version, updated_at)`; cân nhắc ledger riêng cho giao dịch giá trị cao.
3. `player_location(player_id PK/FK, region, x, y, version, updated_at)`.
4. `player_combat_state(player_id PK/FK, active_combat, version, updated_at)`.
5. `player_profile`/`player_account` chỉ tách sau khi hot domains ổn định.
6. `player_statuses(player_id, status_id, source, value, expires_at, updated_at, PK/unique phù hợp)` thay JSONB read-modify-write.

`players` vẫn là identity root trong giai đoạn chuyển đổi. Bảng mới khai báo đầy đủ cột trong `CREATE TABLE IF NOT EXISTS`; khi bổ sung cột vào bảng đã tồn tại phải đặt `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` ngay sau `CREATE` theo convention dự án. Không xóa cột cũ cho đến khi đối soát hoàn tất.

## 5. Thiết kế transaction chuẩn

### 5.1 Chu trình mutation

1. Validate auth và request ngoài transaction.
2. Đọc reference/static data và dựng kế hoạch mutation ngoài transaction.
3. Với combat, thực hiện mô phỏng deterministic/RNG và dựng delta ngoài transaction; ghi seed/input cần kiểm chứng.
4. Mở `BeginTx(requestContextWithDeadline, ...)`.
5. `SET LOCAL lock_timeout` ngắn (khởi điểm 1–2 giây), `statement_timeout` 3–5 giây cho OLTP và `idle_in_transaction_session_timeout` phù hợp.
6. Lock toàn bộ player liên quan trong **một query**, UUID tăng dần. Sau đó lock bảng theo thứ tự toàn cục bên dưới.
7. Revalidate expected version/state/balance.
8. Ghi delta nhỏ nhất, kiểm tra `RowsAffected` và invariant.
9. Ghi idempotency result/outbox trong cùng transaction.
10. Commit; sau commit mới publish/cache invalidation. Nếu publish lỗi, outbox retry.

### 5.2 Thứ tự lock toàn cục

Trước khi refactor, dựng lock graph từ toàn bộ `FOR UPDATE`, `UPDATE`/`DELETE`, FK cascade/trigger và advisory lock. Sau đó ban hành **total order tới cấp table và key**, không chỉ gom nhiều bảng vào cùng một bậc. Thứ tự khởi điểm dưới đây phải được xác nhận bằng graph:

1. `players` hoặc các bảng 1:1 mới, player UUID tăng dần.
2. Shared aggregate theo thứ tự cố định, ví dụ `parties -> sects -> trades -> auctions -> mine_slots`, mỗi bảng theo ID tăng dần.
3. Child player tables: `player_gu`, inventory item, status/achievement theo primary key tăng dần.
4. Ledger/idempotency/outbox append-only.

Không được khóa player A, child A rồi mới player B. Các flow hiện khóa trade/auction trước player phải được refactor trọn vẹn sang order mới, không đổi một nửa. Nếu cần đọc shared row để biết player ID, đọc optimistic trước, khóa theo total order rồi revalidate shared-row version. Dual-write old/new table cũng phải có order cố định. Với batch maintenance, dùng CTE `SELECT ... ORDER BY ... FOR UPDATE SKIP LOCKED LIMIT N` rồi update tập ID đó trong cùng transaction; có checkpoint/fairness để row nóng không bị bỏ đói.

### 5.3 CAS theo domain thay cho một version toàn cục

Một `players.version` duy nhất khiến movement xung đột không cần thiết với profile. Mỗi hot domain dùng `BIGINT` version riêng, backfill giá trị đầu và trả authoritative version/snapshot sau mutation:

```sql
UPDATE player_location
SET region = $1, x = $2, y = $3,
    version = version + 1, updated_at = NOW()
WHERE player_id = $4 AND version = $5
RETURNING version;
```

Nếu không có row trả về:

- Sync mutation: reload/revalidate và retry có giới hạn nếu operation idempotent; nếu không, trả `409 Conflict` cùng version mới.
- Async snapshot: coi là stale, không ghi, ACK với metric `stale_dropped_total`.
- Lock timeout/deadlock: rollback toàn transaction rồi retry với exponential backoff + jitter; không retry riêng statement trong transaction đã abort.
- Guard balance/invariant có `RowsAffected=0`: trả domain conflict/insufficient-resource, tuyệt đối không coi là success hoặc retry vô hạn.

Trước khi bật CAS, audit mọi writer của domain: tất cả đều phải increment/check version thông qua helper chung và contract test, hoặc DB trigger trong giai đoạn tương thích. Chỉ một legacy writer không tăng version cũng đủ làm stale CAS thành công.

## 6. Queue/write-behind an toàn

### 6.1 Thu hẹp payload

- `player_movement`: chỉ `player_id`, region, x/y, `location_version`, `occurred_at` và progress operation nếu thật sự cùng ownership. Loại HP, mana, stones, combat và Human Qi.
- Không dùng full JSON snapshot cho map progress; lưu operation idempotent như `cell_cleared(cell_id)`/`subzone_visited(id)` hoặc normalized upsert.
- `player_stats`: bỏ khỏi write-behind đối với HP/mana chịu tác động combat. Nếu giữ cho lazy regen, chỉ queue một phép materialize có `base_version`, `evaluated_at` và CAS.

### 6.2 Claim/ack atomic

Thay key + pending set hiện tại bằng một trong hai phương án:

- **Khuyến nghị:** Redis Stream consumer group, event có `event_id`, domain version; `XACK` chỉ sau commit/stale determination, pending entries được `XAUTOCLAIM` khi consumer chết. DB lưu unique `(domain, event_id)` trong cùng transaction để redelivery sau trường hợp “commit xong nhưng chưa ACK” không áp dụng hai lần. Quy định max delivery, poison-message DLQ, trim policy và ordering theo player.
- **Thay đổi nhỏ hơn:** Lua script claim `{payload,generation}` sang processing lease; ACK chỉ xóa khi generation hiện tại đúng generation đã persist. Lease dùng fencing token để consumer cũ không thể ghi sau khi lease hết và consumer mới đã claim.

Mọi failure phải giữ/requeue item với backoff+jitter. Sau N lần đưa vào DLQ, cảnh báo và cho phép replay. Metrics phải phân biệt `persisted`, `stale_dropped`, `retry`, `dead_letter`; không ghi nhận cả batch thành công khi một số player timeout.

### 6.3 Batch policy

- Coalesce bản mới nhất theo `(player_id, domain)` trước khi xuống DB.
- Sort UUID, chunk 25–100 và giới hạn tổng thời gian mỗi flush.
- Không để movement/stats cùng update một row/bảng sau khi vertical partition.
- Presence để Redis TTL làm nguồn realtime; snapshot PostgreSQL theo chunk và `SKIP LOCKED`, hoặc bảng riêng `player_presence`.

## 7. Kế hoạch triển khai theo pha

### Pha 0 — Baseline và chặn mất dữ liệu (1–2 ngày)

**Việc làm**

- Bật `pg_stat_statements`; staging bật `log_lock_waits=on`, giảm `deadlock_timeout` để thấy blocker sớm.
- Mở rộng pool watchdog: log `xact_start`, `query_start`, `pg_blocking_pids(pid)`, application/route/operation correlation; không log dữ liệu nhạy cảm.
- Metric theo SQLSTATE `55P03`, `40P01`, `57014`, route, operation; histogram transaction duration/rows touched; queue age/retry/stale/DLQ.
- Sửa worker kiểm tra `RowsAffected`. Không ACK nếu write lỗi; stale CAS được ACK với metric riêng.
- Xóa các field không thuộc location khỏi movement payload/SQL; trước mắt tuyệt đối không để movement ghi `human_qi`.
- Sửa race ACK bằng generation/Lua hoặc Stream trước khi tiếp tục mở rộng write-behind.

**Exit criteria**

- Có dashboard blocker/blocked transaction và top SQL; xác định được ít nhất 95% lock timeout theo route.
- Stress test movement song song cultivate/combat không mất HP, mana, đá hoặc Human Qi.
- Producer ghi update N+1 đúng lúc consumer ACK N vẫn còn N+1 để xử lý.

### Pha 1 — Sửa deadlock và transaction dài (3–5 ngày)

**Việc làm**

- Chuẩn hóa order `players -> player_gu`; growth phải khóa player trước child hoặc chuyển sang optimistic update không cần khóa chéo.
- Tạo transaction helper thống nhất có request context, per-use-case timeout, SQLSTATE classification và full-transaction retry chỉ cho operation idempotent.
- Sửa `loadRow` để mọi query mutation dùng cùng `tx`; không lấy `s.db` connection thứ hai. Tạo projection loader nhỏ theo endpoint.
- Tách passive status thành read-only projection; không mở transaction có write rồi rollback trên polling GET.
- Refactor combat thành compute ngoài transaction + commit delta ngắn. Lock actor/target/allies bằng một sorted query; notification/event ra sau commit/outbox.
- Áp dụng sorted multi-player lock cho marketplace, party, mine, sect và trade.

**Exit criteria**

- Test hunger và growth chạy đồng thời nhiều vòng không có deadlock.
- p99 transaction gameplay dưới 500 ms trên staging; không query Redis/network khi transaction mở.
- Hủy HTTP request làm transaction cancel/rollback ngay; không còn idle transaction giữ lock.

### Pha 2 — Domain version, idempotency và frontend ordering (2–3 sprint, rollout từng domain)

**Việc làm**

- Thêm `location_version`, `runtime_version`, `resource_version`, `combat_version`; response REST và WS delta trả version tương ứng.
- Thêm bảng idempotency với unique `(player_id, operation, key)`, request hash, status/result. Reserve key bằng `INSERT ... ON CONFLICT` trong cùng transaction **trước side effect**; duplicate `IN_PROGRESS` trả trạng thái retry phù hợp, duplicate đã commit replay kết quả đã lưu. Retention phải dài hơn retry window tối đa, không cleanup key đang chạy. Outbox có unique event key, lease `FOR UPDATE SKIP LOCKED`, retry và DLQ.
- Bắt buộc `X-Idempotency-Key` cho combat, teleport/move có chi phí hoặc side effect, claim, buy/sell, refine và mutation tiêu tài nguyên. Movement snapshot thuần dùng `(session_epoch, monotonic_client_sequence/location_version)`, không tạo idempotency row cho từng bước chân.
- Currency dùng guarded atomic delta/ledger; không lưu lại số dư từ object snapshot.
- Frontend chỉ apply response/event có version không thấp hơn state hiện tại.
- GET status/grid/aperture dùng single-flight/latest-wins; polling bằng recursive timeout sau khi request hoàn tất, có jitter, pause khi tab ẩn/offline. Khi WebSocket khỏe, status polling chỉ là fallback.
- Thay `actionLoading` toàn cục bằng in-flight key/counter theo operation/resource.

**Exit criteria**

- Gửi cùng mutation 10 lần với một key chỉ có một side effect.
- Test đảo thứ tự REST/WS không làm state frontend lùi version.
- Không có hai status/grid request cùng key đồng thời trong browser test.

### Pha 3 — Tách hot row theo write affinity (1–2 tuần)

**Thứ tự migration an toàn**

1. Expand schema: tạo bảng mới và index/FK idempotent; xác định source of truth cho từng bước.
2. Deploy compatibility trigger/layer để cả binary cũ chỉ ghi `players` cũng cập nhật bảng mới trong rolling deployment. Trigger chỉ bắt legacy-source update (hoặc dùng session flag/version guard) để không dual-write lặp/đệ quy. Quy định lock order old/new row và rollback switch.
3. Backfill theo chunk nhỏ, checkpoint được, dùng watermark/version guard (`INSERT ... ON CONFLICT ... WHERE incoming_version > current_version`) để không ghi đè write đồng thời; nếu version bằng nhau nhưng hash/value khác thì dừng ghi và phát metric mismatch.
4. Shadow write có metric sai lệch; writer mới ghi cả nguồn cũ/mới trong cùng transaction trong cửa sổ chuyển tiếp.
5. Shadow read/compare sampling; reconcile mọi mismatch.
6. Canary read từ bảng mới theo tỷ lệ 1% -> 10% -> 50% -> 100%.
7. Chuyển writer từng domain, ngừng binary legacy rồi giữ rollback switch.
8. Sau ít nhất một chu kỳ vận hành ổn định mới bỏ trigger/dual-write/index/cột legacy ở thay đổi riêng.

Ưu tiên tách `player_location`, `player_runtime_stats`, `player_resources`, rồi `player_combat_state` và presence. Tách profile/account sau cùng vì ít ghi và rủi ro thấp hơn.

**Exit criteria**

- Movement, activity, stats và combat không còn tranh cùng tuple.
- Shadow comparison không mismatch trong cửa sổ canary đã thống nhất.
- WAL bytes/update và table/index bloat giảm so với baseline.

### Pha 4 — Normalize collection JSONB và vận hành dài hạn (1 tuần+)

- Chuyển statuses sang child table để grant/revoke/expiry là upsert/delete riêng, worker expiry dùng index `(expires_at)` và `SKIP LOCKED`.
- Đánh giá `cleared_cells`, `visited_subzones`, path mastery, achievements và settings theo kích thước/tần suất ghi; chỉ normalize phần thực sự nóng/lớn.
- Gỡ GIN index không có query sử dụng sau khi xác nhận bằng `pg_stat_user_indexes`; không xóa chỉ dựa trên suy đoán.
- Autovacuum/analyze cho hot tables dựa trên update rate và bloat đo được; không hard-code tuning trước baseline.

## 8. Kiểm thử bắt buộc

### 8.1 Correctness và concurrency

- Unit test ownership: mỗi writer chỉ thay đổi domain được phép.
- Schema/invariant test: audit dữ liệu, thêm `CHECK` phù hợp bằng `NOT VALID`, sửa row lỗi rồi `VALIDATE CONSTRAINT`; HP/mana không âm, stones dùng `BIGINT`, float có tolerance/cap rõ ràng.
- CAS tests: stale version không overwrite; `RowsAffected=0` được phân loại đúng.
- Idempotency tests: retry trước/sau commit và concurrent duplicate.
- Integration test 2 transaction khóa actor/target ngược input nhưng DB luôn lock UUID tăng dần.
- Race test hunger/growth/status expiry với combat/refine/trade trên cùng player.
- Combat CAS test: snapshot version tất cả actor/target/allies/mine/inventory, dùng deterministic seed + event ID; khi revalidate fail phải đọc lại và recompute theo cùng policy, không commit delta trên state cũ và không lộ RNG result trước commit.
- Queue test crash ở mọi điểm: sau claim, sau DB commit trước ACK, sau producer generation mới.
- Reconciliation test tổng currency/ledger, item quantities và active combat.

### 8.2 Load test

Kịch bản staging tối thiểu:

1. 500–1.000 player ảo; movement liên tục, status fallback, combat và marketplace hỗn hợp.
2. Cùng một “hot player” nhận movement + combat + hunger/growth + settings để ép contention.
3. Kill/restart worker Redis giữa flush; ngắt DB ngắn hạn rồi phục hồi.
4. Đo `55P03`, `40P01`, p95/p99 latency/transaction duration, pool wait, pending age, WAL và bloat.

Không đạt nếu chỉ “không lỗi HTTP”; phải đối soát state cuối với event ledger/model kỳ vọng.

### 8.3 Lệnh kiểm chứng khi triển khai

```bash
cd backend && go test ./...
cd backend && go test -race ./...
cd frontend && npm run lint
cd frontend && npm run build
```

Bổ sung test tích hợp PostgreSQL thật trong Docker cho lock/CAS; mock DB không phát hiện row-lock deadlock.

## 9. Runbook lock timeout

Khi cảnh báo tăng:

1. Phân loại SQLSTATE và route/operation; không retry mù ở frontend.
2. Tìm blocker bằng `pg_blocking_pids`, xem `xact_start`, `state`, `wait_event`, query đã rút gọn.
3. Nếu blocker là `idle in transaction`, hủy session và sửa code path/context; nếu query đang chạy hợp lệ, ưu tiên giảm scope/chunk.
4. Kiểm tra pool `InUse`, `WaitCount`, `WaitDuration`; pool đầy có thể là hậu quả transaction dài, không mặc định tăng pool.
5. Kiểm tra pending age/DLQ và dừng worker không thiết yếu nếu nó khuếch đại contention.
6. Chỉ terminate backend PID theo playbook production đã phê duyệt; ghi incident và correlation ID.

Query chẩn đoán đề xuất:

```sql
SELECT a.pid,
       a.xact_start,
       a.query_start,
       a.state,
       a.wait_event_type,
       a.wait_event,
       pg_blocking_pids(a.pid) AS blockers,
       left(a.query, 300) AS query
FROM pg_stat_activity a
WHERE a.datname = current_database()
  AND (cardinality(pg_blocking_pids(a.pid)) > 0 OR a.state = 'idle in transaction')
ORDER BY a.xact_start NULLS LAST;
```

## 10. Danh sách công việc ưu tiên

| ID | Ưu tiên | Công việc | Phụ thuộc | Rủi ro nếu chưa làm |
|---|---|---|---|---|
| PP-01 | P0 | Metrics blocker/SQLSTATE/tx duration/queue age | Không | Không biết timeout đến từ đâu |
| PP-02 | P0 | Queue generation claim/ack atomic + `RowsAffected` | PP-01 | Mất update im lặng |
| PP-03 | P0 | Hotfix thu hẹp movement ownership, bỏ Human Qi/stats | Không; phối hợp producer contract | Snapshot cũ ghi đè tài sản |
| PP-04 | P0 | Chuẩn lock order hunger/growth | Không | Deadlock trực tiếp |
| PP-05 | P0 | Loại nested `s.db` query trong transaction | Không | Pool starvation/snapshot lệch |
| PP-06 | P0 | Passive stats read-only | Không | Polling giữ lock vô ích |
| PP-07 | P1 | Transaction helper/context/timeouts/retry policy | PP-01 | Transaction mồ côi/chờ dài |
| PP-08 | P1 | Rút ngắn combat, sorted multi-player locks | PP-04, PP-07 | Hot player tiếp tục timeout |
| PP-09 | P2 | Domain CAS/version trong REST + WS, canary từng domain | PP-03 | Response/snapshot stale |
| PP-10 | P2 | Idempotency + currency atomic delta/ledger | PP-07 | Retry tạo side effect kép |
| PP-11 | P2 | Frontend single-flight/version-aware polling | PP-09 | Request overlap/state lùi |
| PP-12 | P3 | Tách location/runtime/resources/combat/presence | PP-09 | Mọi writer vẫn tranh một row |
| PP-13 | P4 | Normalize statuses và collection nóng | PP-12 | JSONB RMW/lost update/bloat |

## 11. Quyết định không nên làm

- Không tăng lock timeout/pool size trước khi đo và rút ngắn transaction.
- Không dùng một mutex Go theo player làm cơ chế correctness chính: không hoạt động khi chạy nhiều instance và dễ giữ lock qua network.
- Không chuyển currency/inventory sang Redis-only hoặc eventual consistency.
- Không dual-write DB và Redis theo kiểu best effort mà không version/reconciliation.
- Không retry mutation có side effect nếu chưa có idempotency key hoặc chưa chứng minh operation tự nhiên idempotent/CAS-safe.
- Không xóa cột/index legacy trong cùng release tạo bảng mới.

Khi hoàn thành PP-01 đến PP-08, hệ thống phải hết các nguyên nhân lock timeout cấp bách. PP-09 đến PP-13 bảo đảm khả năng mở rộng và ngăn lỗi stale/lost update quay lại khi số người chơi và số worker tăng.
