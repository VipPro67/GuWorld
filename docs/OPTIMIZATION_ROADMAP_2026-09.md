# GuWorld Optimization Roadmap — 2026-09

## Mục tiêu và cách xếp hạng

Tài liệu này là kết quả audit tĩnh backend, frontend và vận hành tại thời điểm
2026-09-14. Roadmap WebSocket/Redis đã hoàn tất; các mục được đánh dấu hoàn thành
trong `OPTIMIZATION_PLAN.md` và `OPTIMIZATION_PROGRESS.md` không được lặp lại.

Thứ tự ưu tiên dựa trên bốn yếu tố:

1. **Ảnh hưởng** đến bảo mật, tính đúng đắn, độ ổn định và độ trễ.
2. **Khả năng xảy ra** trong vận hành thực tế.
3. **Phạm vi người dùng** bị ảnh hưởng.
4. **Chi phí/rủi ro triển khai**.

| Mức | Ý nghĩa | SLA đề xuất |
|---|---|---|
| **P0** | Rủi ro bảo mật hoặc sai dữ liệu; cần xử lý trước tối ưu thuần hiệu năng | Ngay lập tức / 1–3 ngày |
| **P1** | Nút thắt lớn về tải, độ ổn định hoặc trải nghiệm | Sprint kế tiếp / 1–2 tuần |
| **P2** | Tối ưu có ảnh hưởng vừa, cần đo đạc hoặc thay đổi kiến trúc nhỏ | 2–6 tuần |
| **P3** | Dài hạn, chủ yếu tăng khả năng bảo trì và quan sát | Sau khi có baseline |

## Kết luận nhanh

Ba việc có tỷ lệ ảnh hưởng/công sức tốt nhất là:

1. **Thu hồi bí mật đã commit và đóng các endpoint chẩn đoán công khai.**
2. **Sửa contract MGET của write-behind**, giữ đúng vị trí và dọn pending marker
   an toàn khi Redis thiếu key.
3. **Giảm re-render toàn bộ Zustand store và xử lý request cũ ghi đè request
   mới** ở các màn hình thường xuyên nhận cập nhật realtime.

Không nên tiếp tục tăng buffer, connection pool hay tách file lớn theo cảm tính.
Trước các thay đổi P1/P2 phải ghi lại baseline: p50/p95/p99 API, DB pool wait,
cardinality/oldest-age của pending write-behind, Redis pool, request/phút, React
commit time và kích thước chunk frontend.

### Tiến độ iteration 1

- Đã loại credential khỏi working tree/Compose, bỏ Google JSON fallback, thu hẹp
  Redis vào network nội bộ và bật xác minh Redis TLS. Việc rotate/revoke secret
  cũ và làm sạch Git history vẫn là thao tác vận hành bắt buộc.
- Đã gỡ toàn bộ public diagnostics, giữ admin monitoring, và thêm regression test.
- Đã sửa contract MGET/missing-marker của write-behind và thêm test alignment.
- Đã thêm backend test/vet gate trước build/deploy và deploy theo Git SHA. Frontend
  đã có lệnh typecheck; lint debt hiện hữu phải được xử lý trước khi bật frontend
  gate bắt buộc. Readiness/rollback và secret history scan vẫn chưa hoàn tất.

---

## P0 — Làm ngay

### 1. Thu hồi bí mật và thu hẹp bề mặt mạng

**Ảnh hưởng:** Rất cao (bảo mật) · **Công sức:** Thấp–trung bình

**Bằng chứng**

- `docker-compose.yml` chứa trực tiếp thông tin kết nối database, JWT và thông
  tin truy cập hệ thống quan sát thay vì chỉ tham chiếu biến môi trường.
- `backend/google.json` là credential OAuth được track trong Git.
- Redis được publish ra host (có thể ra ngoài tùy firewall) và cấu hình mẫu chưa
  bắt buộc xác thực.
- Khi bật TLS cho Redis, client đang dùng `InsecureSkipVerify: true`.

**Hành động**

1. Rotate/revoke toàn bộ credential đã xuất hiện trong Git; không chỉ xóa file ở
   commit mới. Đánh giá việc rewrite history theo phạm vi repository đã chia sẻ.
2. Chuyển secret sang secret store của môi trường triển khai hoặc file `.env`
   không được track; giữ repository chỉ có tên biến và placeholder.
3. Bỏ publish Redis ra Internet/host nếu không cần, hoặc bind nội bộ và bật
   ACL/password.
4. Cấu hình CA và `ServerName` cho Redis TLS; không bỏ kiểm tra chứng chỉ.
5. Thêm secret scan (ví dụ gitleaks) vào local check và CI.

**Tiêu chí hoàn tất:** scan cả working tree và history không phát hiện credential
hợp lệ; DB/JWT/Grafana/Google OAuth cũ bị revoke (có kế hoạch invalidation session
khi đổi JWT); kết nối Redis TLS thất bại với chứng chỉ sai; Redis không truy cập
được từ mạng ngoài dự kiến.

### 2. Đóng endpoint debug/pprof công khai

**Ảnh hưởng:** Rất cao (bảo mật và DoS) · **Công sức:** Thấp

**Bằng chứng**

- `registerDebugEndpoints` luôn đăng ký pprof, goroutine dump, DB pool và request
  summary trên router chính mà không có middleware xác thực.
- `/api/debug/monsters` cũng không xác thực và trả cả lỗi query/scan nội bộ.

**Hành động:** xóa `/api/debug/monsters`; trong production không đăng ký các route
pprof/profile/trace trên listener ứng dụng. Nếu cần chẩn đoán, dùng listener nội
bộ riêng; bảo vệ các summary còn lại bằng admin auth. Thêm route test cho cả
pprof, system/request summary và monster debug.

**Tiêu chí hoàn tất:** người dùng ẩn danh không lấy được profile, stack,
connection stats hay lỗi SQL; profile CPU dài không thể được kích hoạt từ public
network.

### 3. Sửa tính đúng đắn của write-behind và dựng CI tối thiểu

**Ảnh hưởng:** Rất cao (persistence backlog và ngăn regression) · **Công sức:** Trung bình

**Bằng chứng:** `GetManyJSON` bỏ các kết quả `nil` khỏi MGET, trong khi
`flushEntityType` giả định kết quả giữ vị trí so với danh sách ID ban đầu. Handler
hiện decode ID từ payload nên chưa ghi nhầm player, nhưng batch hỗn hợp không dọn
được marker của key thiếu và contract dễ bị consumer tương lai dùng sai. Đồng
thời workflow deploy chưa có quality gate đầy đủ và deploy bằng tag nổi.

**Hành động theo hai bước:**

1. Giữ kết quả MGET theo đúng vị trí, cleanup chính xác từng pending marker; thêm
   test thiếu key đầu/giữa/cuối và crash giữa persist/ack.
2. Bắt buộc secret scan, `go test ./...`, `go vet ./...`, frontend lint/build và
   typecheck trước deploy; dùng image theo Git SHA/digest.
3. Thêm health/readiness kiểm tra DB, Redis, registry và rollback rollout lỗi.

**Tiêu chí hoàn tất:** không payload nào ghép sai ID và marker thiếu được dọn đúng;
deploy không chạy nếu gate lỗi; production chỉ chạy artifact bất biến và rollout
không ready tự dừng/rollback.

---

## P1 — Sprint kế tiếp

### 4. Loại bỏ subscription toàn store ở frontend

**Ảnh hưởng:** Cao (render/UI) · **Công sức:** Trung bình

Nhiều component gọi `useGameStore()` không selector, gồm các view rất lớn như
Admin, Sect, Aperture, Lobby, Mine và Trade. Mỗi subscription kiểu này render lại
component khi store đổi; descendant còn phụ thuộc ranh giới memo của React.

**Hành động:** đo React Profiler trước; chuyển năm màn hình có commit time cao
nhất sang atomic selector hoặc `useShallow`; đưa subscription xuống component
con cần dữ liệu; memo hóa ranh giới đắt tiền. Không tách component chỉ để giảm số
dòng nếu profiler không chứng minh lợi ích.

**Mục tiêu:** với cùng browser/device và kịch bản 5 phút combat/world đã ghi lại,
giảm ít nhất 50% commit không cần thiết và p95 commit duration; không tăng số
subscription bị rò rỉ.

### 5. Chuẩn hóa vòng đời request và chống stale response

**Ảnh hưởng:** Cao (đúng dữ liệu + tải mạng) · **Công sức:** Trung bình

`authFetch` tạo timeout nhưng không clear khi request thành công; tracking
timestamp hiện chưa ngăn response cũ. Exploration map vừa tải khi tọa độ đổi vừa
poll định kỳ, không có in-flight guard/cancellation, nên response cũ có thể ghi
đè vị trí mới.

**Hành động:** clear timer/map trong `finally`; compose/forward `AbortSignal` của
caller; áp dụng
latest-request-wins cho dữ liệu thay thế; dùng `fetchWithDedupe` cho GET có cùng
key; pause polling khi tab ẩn và backoff khi lỗi. Thêm test fake-timer cho timeout,
unmount và response đảo thứ tự.

**Mục tiêu:** effect caller abort/guard khi unmount; request đồng thời cùng key chỉ
tạo một network call; response cũ không ghi đè state mới.

### 6. Chunking và backpressure cho write-behind persistence

**Ảnh hưởng:** Cao (DB/Redis khi spike) · **Công sức:** Trung bình–cao

Pending ID hiện được đọc toàn bộ bằng `SMEMBERS`, sau đó materialize toàn bộ key
và payload; từng player lại mở transaction tuần tự. Backlog lớn vì vậy làm tăng
memory, thời gian flush và tranh connection pool.

**Hành động:** xử lý theo chunk 100–500 bằng `SSCAN` hoặc lease an toàn, pipeline
read, giới hạn concurrency, giữ compare-and-ack nguyên tử; sau khi đo có thể dùng
bulk `UPDATE ... FROM VALUES`. Đặt hard cap, retry/dead-letter policy và metric
oldest-age/backlog/flush-duration.

**Mục tiêu:** memory theo flush bị chặn theo chunk; backlog hội tụ sau burst; p95
API và DB pool wait không tăng mạnh trong load test; không mất update khi worker
crash giữa flush và ack.

### 7. Bảo đảm worker an toàn khi rolling deploy/scale ngang

**Ảnh hưởng:** Cao (tính đúng đắn game economy) · **Công sức:** Cao

Mỗi process khởi động nhiều polling worker. Nếu hai replica chồng thời gian khi
rolling deploy hoặc scale ngang, các worker chưa có claim/leader thống nhất có
thể xử lý trùng settlement, reward hoặc tick.

**Hành động:** lập inventory từng worker; chọn leader election hoặc claim nguyên tử
bằng advisory lock/`FOR UPDATE SKIP LOCKED`; định nghĩa idempotency key, unique
constraint và retry cho từng loại side effect.

**Mục tiêu:** test hai replica chồng lấn và crash-after-commit chứng minh mỗi
tick/job/reward chỉ settle một lần.

### 8. Hợp nhất Redis subscriber và shutdown đúng context

**Ảnh hưởng:** Cao (ổn định/tài nguyên) · **Công sức:** Trung bình

Realtime subscriber tạo nhiều `PSubscribe`/goroutine riêng, dùng
`context.Background()` và `time.Sleep` khi reconnect. Server shutdown không hủy
được sleep/receive ngay và giữ nhiều dedicated connection.

**Hành động:** truyền server context, đóng PubSub khi cancel, thay sleep bằng timer
`select` theo context. Chỉ hợp nhất subscription nếu đo được áp lực connection và
chứng minh không tạo head-of-line blocking. Thêm test shutdown/reconnect.

**Mục tiêu:** shutdown hoàn tất trong thời hạn; goroutine/connection trở về
baseline sau reconnect; không mất channel routing.

### 9. Dọn artifact và làm deployment tái lập được

**Ảnh hưởng:** Cao (ngăn regression/deploy lỗi) · **Công sức:** Trung bình

Quality gate/readiness đã nằm ở P0. Repository vẫn track nhiều binary build lớn và
dump cục bộ, còn image hạ tầng dùng floating tag.

**Hành động:**

- Pin version/digest image hạ tầng.
- Xóa `backend/guworld_bin`, `backend/guworld_server`, `backend/tmp_server`,
  `backend/cmd/server/server`, `backend/cmd/server/main` và `goroutines.txt` khỏi
  Git; bổ sung `.gitignore`/`.dockerignore`.

**Mục tiêu:** checkout và Docker context không chứa executable/dump sinh ra từ
local build; deploy cùng SHA luôn resolve cùng image/config version.

---

## P2 — Sau khi P0/P1 ổn định

| Hạng mục | Ảnh hưởng | Hướng thực hiện | Chỉ số quyết định/hoàn tất |
|---|---|---|---|
| Rate limit nguyên tử | Trung bình–cao | Thay chuỗi `ZREM/ZCARD/ZADD` bằng Lua nguyên tử, timeout context và lifecycle cleanup | Load test song song không vượt quota ngoài sai số thiết kế |
| Query kiểm tra Gu bị khóa | Trung bình | Thay `JSONB::text LIKE` bằng JSONB path/containment đúng shape và index sau khi có test; audit riêng reservation semantics | Không match UUID ở field khác; query plan phù hợp trên dữ liệu đại diện |
| Auth DB round-trip | Trung bình, phụ thuộc benchmark | Trước tiên đo/index/timeout query; chỉ cache session version trong Redis nếu là bottleneck, với DB fallback fail-closed | Định nghĩa max revocation staleness và test logout/delete/rotation/Redis outage |
| Party loot N+1/O(quantity) | Trung bình–cao | Preload username; phân bổ count trong memory; một upsert/player/item hoặc bulk values | Số query theo số loại item/người chơi, không theo quantity |
| Hunger `ON_TICK` scan | Trung bình | Cache effect đã resolve, query một lượt các Gu liên quan, aggregate action | `EXPLAIN ANALYZE` và tick duration giảm có ý nghĩa |
| Worker query cancellation | Trung bình | Dùng `QueryContext`/`ExecContext`, per-tick timeout; final flush có deadline riêng | Shutdown bounded, query treo bị hủy |
| Giảm polling frontend | Trung bình–cao | Realtime invalidation + reconciliation thưa; coalesce status fetch; pause tab ẩn | Giảm request/phút nhưng vẫn tự hồi phục khi mất event |
| Translation hot path | Trung bình | Cache key bằng dictionary revision thay vì serialize toàn dictionary; precompile lookup | CPU/allocation của translate giảm qua benchmark |
| Frontend test harness | Trung bình | Vitest/RTL cho slice, rollback, request race, WS/SSE; Playwright smoke | Test script trong package và coverage các flow rủi ro |
| HTTP hardening | Trung bình | Header/body limit, `MaxHeaderBytes`, deadline phù hợp; ngoại lệ rõ cho SSE/WS | Slow-client test không giữ vô hạn tài nguyên |

---

## P3 — Dài hạn, dựa trên đo đạc

1. **Tách các view/module lớn có mục tiêu:** ưu tiên phần có profiler hot hoặc
   ownership khó kiểm thử (Admin, Sect, Aperture). Giữ API logic trong Zustand
   theo kiến trúc hiện tại.
2. **Quan sát connection pool:** đưa kích thước pool ra env, thu metric utilization,
   wait count/duration delta và alert. Không mặc định rằng tăng từ 50 connection
   sẽ tăng throughput.
3. **WorldCanvas/Pixi:** đo allocation, texture/Graphics churn và FPS trước khi
   triển khai object pooling/reuse; giữ bounded chunk cache hiện có.
4. **Bundle:** dùng analyzer để xếp hạng chunk; chỉ lazy-load thêm overlay/modal
   khi số liệu chứng minh. Các main view đã dynamic import nên không lặp lại việc
   này.

## Thứ tự triển khai đề xuất

### Tuần 0 — Containment

- P0.1 rotate secret/Redis TLS và P0.2 đóng diagnostics.
- P0.3 sửa ánh xạ write-behind và dựng quality gate/readiness tối thiểu.
- Chụp baseline vận hành và frontend profiler.

### Tuần 1–2 — Quick wins có đo lường

- P1.4 selector cho các view nóng.
- P1.5 request cancellation/deduplication.
- P1.7 bảo vệ worker khi rolling deploy; P1.9 dọn artifact/pin image.

### Tuần 3–4 — Scale path

- P1.6 write-behind chunking/backpressure.
- P1.8 subscriber lifecycle.

### Tuần 5–6 — Tối ưu theo số liệu

- Chọn các mục P2 dựa trên profiler, slow-query log và request metrics.
- Chỉ bắt đầu P3 khi P0/P1 đã có acceptance metric và dashboard theo dõi.

## Quy tắc thực thi cho từng hạng mục

1. Ghi baseline và test tái hiện trước khi sửa.
2. Chia PR nhỏ, có rollback path; không trộn refactor với thay đổi semantics.
3. Khi thêm cột vào bảng hiện có, đặt `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
   ngay sau `CREATE TABLE` theo convention dự án; index production cần kế hoạch
   lock/time.
4. Backend chạy `go test ./...`; logic concurrency chạy thêm `go test -race ./...`
   (ít nhất package liên quan trong CI, toàn bộ suite trước release lớn).
5. Frontend chạy lint, build, typecheck và test mới; thay đổi nhìn thấy được phải
   kiểm tra thủ công/screenshot.
6. So sánh số liệu sau deploy; rollback nếu metric mục tiêu xấu đi hoặc không có
   cải thiện đủ lớn để biện minh độ phức tạp.
