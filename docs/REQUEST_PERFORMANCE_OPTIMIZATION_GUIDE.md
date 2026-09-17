# Hướng dẫn tối ưu request chậm và request bị gọi lặp

## 1. Mục tiêu và phạm vi

Tài liệu này là hướng dẫn triển khai, **không phải thay đổi code**. Mục tiêu là:

1. giảm số request không cần thiết từ frontend;
2. giảm p95/p99 và giá trị `Max` của endpoint chậm;
3. giảm lỗi do request chồng lấn, timeout, rate limit và tranh chấp database;
4. giữ nguyên tính đúng đắn của tài sản, combat và trạng thái người chơi.

Không tối ưu theo cột `TB` đơn lẻ. Một endpoint có trung bình thấp nhưng `Max` cao
có thể đang chờ row lock, Redis hoặc connection pool. Cột `Lỗi` hiện cũng phải
được tách theo HTTP status/reason trước khi kết luận là lỗi hệ thống: `400/409/429`
do luật game khác hoàn toàn `500/502/503/504`.

## 2. Baseline và tiêu chí hoàn tất

Trước mỗi thay đổi, lấy baseline ít nhất 24 giờ và lưu theo **route template**
(không dùng URL chứa player/drop ID):

- request/phút, số user và số request/user/phút;
- p50, p95, p99, max và kích thước response;
- status code, error reason và latency tách riêng success/4xx/5xx;
- thời gian DB pool wait, query, transaction, Redis và publish realtime;
- số query/endpoint, rows scanned/returned, lock wait và timeout;
- số request bị dedupe, abort, cache hit và request còn in-flight ở frontend.

Đặt `request_id`/trace ID xuyên frontend → handler → PostgreSQL/Redis. Với endpoint
mutation, thêm `operation` và `player_id` đã hash; không log token, password hoặc
payload chứa dữ liệu nhạy cảm.

### SLO khởi điểm

| Nhóm | Mục tiêu |
|---|---:|
| GET nóng | p95 `< 150 ms`, p99 `< 500 ms` |
| Mutation thông thường | p95 `< 300 ms`, p99 `< 1 s` |
| Request vượt 2 giây | `< 0,1%` |
| 5xx | `< 0,1%` |
| GET đồng thời cùng cache key từ một client | tối đa 1 network request |
| Poll khi tab ẩn | 0, trừ heartbeat bắt buộc |
| Stale response ghi đè state mới | 0 |

Chỉ đánh dấu hoàn tất khi so sánh cùng traffic/kịch bản và không làm tăng lỗi,
DB pool wait hoặc sai lệch dữ liệu.

## 3. Nguyên tắc chung

### 3.1 Giảm request ở nguồn trước khi làm backend nhanh hơn

Thứ tự ưu tiên cho GET là:

1. không gọi nếu màn hình không dùng dữ liệu;
2. mutation trả về delta/snapshot mới để không cần GET lại;
3. dedupe request đang chạy theo `method + URL + player/session`;
4. cache có TTL và invalidate theo domain;
5. dùng SSE hiện có cho sự kiện thay đổi, polling chỉ làm fallback/reconciliation;
6. cuối cùng mới tối ưu SQL/cache backend.

Không gộp tất cả dữ liệu vào một endpoint bootstrap khổng lồ. Điều đó giảm số
request nhưng tăng payload, coupling và làm một phần chậm kéo chậm toàn phiên.
Nên tải projection tối thiểu trước, phần theo tab tải khi cần.

### 3.2 Quy tắc vòng đời GET ở frontend

- **Single-flight:** các caller đồng thời dùng chung một Promise; xóa Promise trong
  `finally`. Key phải chứa danh tính/session để không dùng nhầm dữ liệu sau logout.
- **TTL/stale-while-revalidate:** trả cache còn mới ngay; cache cũ có thể hiển thị
  tạm rồi refresh nền. TTL gợi ý: dictionary 30–60 phút; sect/aperture 15–30 giây;
  unread count 30–60 giây nếu SSE không phát delta; status 30–60 giây.
- **Latest-request-wins:** dùng `AbortController` hoặc request generation. Response
  cũ không được apply nếu request mới hơn đã bắt đầu hoặc player/session đã đổi.
- **Visibility:** dừng interval khi `document.hidden`; refresh một lần khi tab trở
  lại. Không tạo interval mới nếu request cũ còn chạy.
- **Backoff:** lỗi mạng/5xx dùng exponential backoff có jitter và giới hạn; `4xx`
  nghiệp vụ không retry tự động. Tôn trọng `Retry-After` cho `429`.
- **Mutation:** khóa nút khi đang gửi, thêm idempotency key cho thao tác có thể bị
  retry, và chỉ retry tự động khi server bảo đảm idempotent.

### 3.3 Quy tắc backend

- Đo từng phase trước khi sửa. Không tăng timeout hoặc connection pool để che
  query/transaction chậm.
- Validate và đọc registry tĩnh trước transaction; transaction chỉ lock, revalidate,
  ghi delta nhỏ nhất và commit. Không publish Redis/SSE hay chạy mô phỏng nặng khi
  đang giữ row lock.
- Mọi query nóng phải kiểm tra bằng `EXPLAIN (ANALYZE, BUFFERS)` trên dữ liệu gần
  production. Chỉ thêm index khi chứng minh giảm scan; kiểm tra chi phí ghi và
  index trùng trước khi thêm.
- Projection GET chỉ chọn cột cần trả, tránh load full player/JSONB/inventory rồi
  bỏ phần lớn dữ liệu.
- Cache chỉ dành cho dữ liệu đọc nhiều và có chiến lược invalidate rõ. Không cache
  mù số dư/tài sản hoặc kết quả phụ thuộc quyền người dùng.
- Mutation nhiều bảng phải atomic. Lock player theo UUID tăng dần và tuân thủ thứ
  tự lock toàn cục; timeout phải rollback toàn bộ.

## 4. Kế hoạch giảm các request bị gọi nhiều

### P0 — Chặn fan-out và polling chồng nhau

| Endpoint | Hiện trạng cần kiểm tra | Hướng tối ưu |
|---|---|---|
| `GET /api/sects/me` | Được gọi sau nhiều sect mutation và khi vào `SectView`; đã có single-flight nhưng chưa có freshness policy | Mutation sect trả projection/delta cần thiết; chỉ invalidate domain sect; thêm TTL 15–30 giây; không đồng thời gọi lại cả sect và session nếu response đã đủ |
| `GET /api/auth/me` | `initializeSession()` được gọi sau nhiều mutation ở các slice; GET này có thể materialize hunger/state và đụng write/lock | Chỉ gọi một lần khi bootstrap/token đổi; mutation trả authoritative player delta; tách projection session nhẹ khỏi materialization nếu characterization test chứng minh an toàn |
| `GET /api/player/inventory/gus` | Có thể bị gọi sau mutation đã trả inventory | Dùng `applyInventoryFromResponse`; chỉ refetch khi response không có delta hoặc có sự kiện invalidation; single-flight + TTL ngắn |
| `GET /api/player/notifications/unread-count` | Poll/count có thể trùng với notification fetch hoặc mở bell | SSE tăng/giảm count; response danh sách chứa unread count; fallback polling 30–60 giây, pause khi hidden |
| `GET /api/immortal-aperture`, `/nodes`, `/dao-marks`, `/tribulations/active` | Component aperture gọi lúc mount, refresh và interval; một số GET đã single-flight nhưng Dao Marks/tribulation chưa | Một orchestrator theo tab, không để nhiều component tự poll; dedupe cả bốn; polling chỉ domain thật sự thay đổi; mutation trả delta và invalidate đúng key |
| `GET /api/player/status/:id` | Poll toàn trang cộng refresh sau combat/action | SSE player event là nguồn cập nhật nhanh; một reconciliation timer duy nhất 30–60 giây; không poll khi combat active/hidden; coalesce refresh sau mutation |

Nguồn fan-out chính cần xử lý trước là `initializeSession()`: hiện một lần gọi có
thể kéo theo `/auth/me`, `/sects/me`, `/player/inventory/gus` và
`/player/notifications/unread-count`, trong khi nhiều mutation lại dùng hàm này
như một thao tác “refresh tất cả”. Hãy tách rõ `restoreSession()` (chỉ dùng khi
mount/login/token đổi) khỏi các hàm refresh theo domain. Đây là thay đổi có khả
năng giảm đồng thời bốn endpoint nóng mà không cần cache backend.

Thực hiện audit call graph cho `initializeSession`, `fetchPlayerStatus`,
`fetchMySectStatus`, `fetchAperture*`, `fetchDaoMarks` và
`fetchActiveTribulation`. Với mỗi call site, ghi rõ một trong bốn lý do hợp lệ:
bootstrap, tab-visible, mutation-invalidation hoặc recovery. Xóa/ghép mọi call
không có lý do riêng.

### P1 — Contract response và invalidation

Chuẩn hóa mutation response theo domain:

```json
{
  "data": {},
  "versions": { "player": 42, "inventory": 18, "sect": 7 },
  "invalidates": ["aperture:nodes"]
}
```

Đây là contract định hướng, không bắt buộc mọi endpoint dùng ngay. Ưu tiên cho
sect donation, gacha, refine, set vital Gu, collect-all, combat và loot. Frontend apply
delta có version; chỉ refetch key trong `invalidates`. Event SSE và GET response
cũ hơn version đang có phải bị bỏ qua.

## 5. Kế hoạch cho các endpoint chậm

### `POST /api/explore/move` — TB ~3,9 giây, Max ~13,5 giây

Độ trễ đáng lo ngại nhưng mới có 22 mẫu. Trước khi xếp ưu tiên cuối cùng, cần tái
hiện hoặc thu thêm mẫu và tách status/phase; không kết luận chỉ từ một giá trị Max.

1. Trace riêng validate map, DB lock/update, cleared-cell load, Redis cache/publish
   và dựng response.
   Đặc biệt đo nhánh auto-resolve combat, đồng bộ Gu và party vì các công việc này
   hiện có thể nằm trong cùng transaction di chuyển.
2. Kiểm tra thời gian có gần Redis timeout hay lock timeout; không tăng timeout.
3. Nếu trace chứng minh có nhiều Redis round-trip cho cleared cells thì batch hoặc
   pipeline chúng; đồng thời kiểm tra và bỏ snapshot publish trùng.
4. Không giữ transaction khi gọi Redis/publish. Chỉ update location/progress delta.
5. Client không gửi move mới khi move cũ chưa settle, hoặc dùng sequence để server
   loại move cũ; không retry POST nếu chưa có idempotency/move sequence.

### `POST /api/gacha/open` — nhiều request và lỗi nhất

1. Tách `4xx` thiếu tài nguyên/rate limit khỏi `5xx`; đo success latency riêng.
2. Thêm idempotency key để timeout/retry không trừ tiền hoặc cấp thưởng hai lần.
3. Load pool/registry ngoài transaction; trong transaction chỉ lock balance,
   revalidate, ghi reward/ledger rồi commit.
   Đo và loại vòng lặp upsert toàn bộ recipe đã mở: chỉ ghi recipe thực sự thay đổi
   hoặc dùng bulk upsert.
4. Response trả balance và inventory delta để bỏ `initializeSession()` ngay sau đó.
5. Nếu random/result cần audit, lưu seed/result trong cùng transaction, không làm
   giảm tính atomic.

### `POST /api/refine` — request nhiều, tail gần 10 giây

1. Tách lỗi luật game (công thức/vật liệu/mana không hợp lệ) khỏi 5xx, lock timeout
   và DB timeout; đo success latency riêng.
2. Frontend đã chặn action đồng thời và apply inventory từ response; giữ contract
   này, không thay bằng `initializeSession()` hoặc GET inventory sau mỗi lần refine.
3. Thêm operation/idempotency key ở server trước khi cho phép retry sau timeout để
   không tiêu hao vật liệu hoặc tạo Gu hai lần.
4. Trace thời gian load player/inventory, recipe, từng material lookup, RNG/effect,
   ghi inventory và commit. Batch lookup/consume material thay vì query từng item.
5. Đọc registry/reference data không cần lock ở ngoài transaction; trong transaction
   chỉ lock, revalidate tài nguyên, ghi kết quả/refinement audit và commit.

### `POST /api/sects/donate/bulk`

- Gom donation thành set-based SQL/upsert thay vì vòng lặp query; đo query count.
- Batch lookup inventory và item dictionary; không query price/name và remove item
  riêng cho từng material trong request.
- Lock sect/player theo thứ tự cố định, giữ transaction ngắn.
- Response trả contribution, balance và sect projection; không refetch cả
  `/sects/me` lẫn `/auth/me` nếu không cần.
- Idempotency key bắt buộc nếu client có retry.

### `GET /api/sects`

- Phân trang/cursor, giới hạn cột và kích thước danh sách; không trả member/detail
  không dùng ở màn hình list.
- Kiểm tra index đúng `WHERE + ORDER BY`; cache trang công khai ngắn hạn với ETag.
- Tránh N+1 count/member query; pre-aggregate hoặc join/group sau khi đo.

### `POST /api/inventory/set-vital`

- Chuẩn bị/evaluate effect ngoài transaction nếu không phụ thuộc state khóa.
- Chỉ clear Gu đang là vital (`is_vital = true`), không update mọi Gu của player;
  kiểm chứng partial unique index cho invariant một vital Gu/player.
- Cập nhật player/player_gu theo lock order thống nhất, tránh full inventory reload.
- Trả equipped Gu, derived stats và inventory delta; frontend không gọi session/status
  lại ngay nếu version mới đã được áp dụng.

### `POST /api/immortal-aperture/nodes/collect-all`

- Chụp thời gian theo số node/settlement; loại N+1 bằng query/batch upsert.
- Lock/claim các node đủ điều kiện khi đọc để hai collect-all đồng thời không cùng
  nhận thưởng; batch cập nhật timer thay vì một `UPDATE` cho mỗi node.
- Tính reward plan ngoài transaction, revalidate version trong transaction rồi ghi
  aggregate/set-based; bảo đảm atomic và idempotent.
- Response trả node/resource delta, không gọi lại aperture + nodes + status toàn bộ.

### `GET /api/dictionary/gus`

- Đây là dữ liệu registry tĩnh: ưu tiên cache public/private phù hợp, `ETag` hoặc
  version hash và `Cache-Control` dài; invalidation khi Admin import/restart registry.
- Sitemap và browser không được tạo các lần tải trùng không cần thiết.
- Pre-serialize/cache JSON nếu profiling cho thấy encode là chi phí chính.

### `POST /api/auth/login`

- Tách thời gian password hash (CPU), DB lookup, session/Redis và dựng player.
- Query xác thực trước chỉ lấy ID/password/security fields; sau khi mật khẩu đúng
  mới tải session projection, thay vì aggregate Gu/player payload rộng cho cả lần
  đăng nhập thất bại.
- Giữ password hashing đủ an toàn; không giảm cost chỉ để đẹp latency. Giới hạn
  concurrency/rate để chống CPU exhaustion.
- Login trả bootstrap tối thiểu; tải domain theo tab sau đó. Không gọi `/auth/me`
  ngay nếu login đã trả cùng session/player projection.

### `POST /api/combat/start`

- Profile riêng thời gian mô phỏng, lock wait, persistence và publish.
- Mô phỏng deterministic và dựng delta ngoài transaction; transaction ngắn để
  lock, revalidate input/version, ghi combat state/cost và commit.
- Dùng operation/combat ID chống double-start; trả authoritative player/combat delta
  để tránh status refetch tức thì.

### `POST /api/world/drops/:id/loot`

- Atomic claim bằng conditional update/delete hoặc row lock ngắn; `RowsAffected=0`
  trả conflict đã được loot, không phải 500.
- Index lookup theo drop ID/tile/expiry sau khi kiểm tra execution plan.
- Ghi inventory/reward cùng transaction; response trả inventory/balance delta và
  ground-drop removal event.

## 6. Thứ tự triển khai đề xuất

1. **Instrumentation (1–2 ngày):** status/reason, phase timing, DB/Redis/pool/lock,
   frontend call counters; lấy baseline.
2. **Frontend P0 (2–4 ngày):** một polling owner, visibility pause, single-flight,
   latest-request-wins và bỏ `initializeSession()` thừa.
3. **Response delta (3–5 ngày):** ưu tiên gacha, refine, sect donation, collect-all,
   set-vital, combat và loot.
4. **Backend tail latency (3–7 ngày):** điều tra move trước và chỉ chốt thứ tự sửa
   sau khi có đủ mẫu; tiếp theo chọn gacha/refine/collect-all/sects theo p95, volume
   và error breakdown thực tế.
5. **Cache/index (sau khi đo):** dictionary và sect list; không cache mutation.
6. **Load/regression test:** traffic burst, hai tab, tab background/foreground,
   response đảo thứ tự, retry sau timeout và hai request mutation trùng ID; bắt
   buộc có kịch bản double-submit cho gacha/refine/collect-all/loot.

Mỗi bước triển khai độc lập, có feature flag/rollback và dashboard so sánh trước
sau. Không gộp refactor frontend, schema/index và transaction lớn vào cùng một
release vì sẽ khó xác định nguyên nhân regression.

## 7. Checklist review trước khi merge thay đổi tối ưu

### Frontend

- [ ] Caller có thực sự cần dữ liệu ở thời điểm gọi không?
- [ ] GET cùng key đã single-flight và có freshness policy chưa?
- [ ] Interval có cleanup, pause khi hidden và in-flight guard chưa?
- [ ] Abort signal/timer được cleanup trong `finally` chưa?
- [ ] Response cũ/session cũ có thể ghi đè state mới không?
- [ ] Mutation response đã đủ để tránh refetch toàn domain chưa?
- [ ] `429`, `4xx`, `5xx` có retry policy khác nhau chưa?

### Backend/database

- [ ] Trace đã chỉ ra phase chậm, hay thay đổi chỉ dựa trên phỏng đoán?
- [ ] Query plan được đo trên dữ liệu đại diện chưa?
- [ ] Có N+1, full JSONB/full player load hoặc serialize payload thừa không?
- [ ] Transaction có network call, mô phỏng CPU hoặc publish event bên trong không?
- [ ] Lock order, atomicity, idempotency và `RowsAffected` đã được kiểm tra chưa?
- [ ] Index mới có đúng predicate/order và không trùng index hiện hữu không?
- [ ] Cache có key theo user/quyền, TTL, version và invalidation rõ ràng chưa?

### Xác nhận kết quả

- [ ] Request/user/phút và request đồng thời giảm theo mục tiêu.
- [ ] p95/p99/Max giảm; không chỉ giảm trung bình.
- [ ] 5xx, conflict, rate limit và timeout không tăng.
- [ ] Không double reward/double charge/lost update trong test retry và concurrency.
- [ ] SSE mất kết nối vẫn hội tụ đúng nhờ reconciliation có backoff.

## 8. Những cách “tối ưu” không nên làm

- Tăng timeout, pool size hoặc rate limit trước khi biết bottleneck.
- Cache toàn bộ `/auth/me` mà không tách dữ liệu nhạy cảm, version và invalidation.
- Debounce mutation tài sản theo kiểu last-write-wins.
- Retry mù mọi POST sau timeout.
- Thêm index cho mọi cột được thấy trong query.
- Thay nhiều GET bằng một payload bootstrap rất lớn và luôn tải ở mọi màn hình.
- Chuyển polling nhanh sang WebSocket/SSE nhưng vẫn giữ polling cũ chạy song song.
- Đánh giá hiệu quả chỉ bằng `Max` của vài mẫu hoặc `TB` toàn bộ status code.
