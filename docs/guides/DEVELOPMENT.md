# Development Guide

## Auditing legacy effect data

Run `go run ./cmd/migrate_gu_effects -format gu -source data/Gu.csv` from
`backend/` for a non-destructive JSON audit. Candidate files are only written
when both `-write` and a non-source `-output` path are supplied. See
`docs/guides/GAME_DATA.md` for report fields and the migration gate. Non-empty
candidates remain closed until runtime shadow evidence is bound to the selected
snapshot and catalog revision in the machine-readable report.

## CSV authoring tools (`gudata`)

`go run ./cmd/gudata <subcommand>` operates directly on `backend/data/*.csv`.
Every subcommand is dry-run by default; mutating runs require `-write` and a
distinct `-output` path (never overwrites source). Reports are JSON on stdout.

| Subcommand | Purpose |
| --- | --- |
| `check-effects` | Validate `Gu.csv` (full catalog + Block-v2 + unclassified-key) and `Item.csv` (Block-v2). Exits non-zero on errors. |
| `balance` | Score each Gu's flat combat power vs its rank peers (median band ± `tol`) and flag outliers. `-write` scales the flagged Gu's `DAMAGE`/`SHIELD`/`HEAL`/`DIRECT_HP` FLAT values toward the band edge. Tune via `-config` (`balance.tol`, `min_score_frac`, `weights`). |
| `rebalance` | Clamp supported combat, DoT, and base-stat values to documented rank anchors (±35% by default). Other numeric effects use their same-rank median when at least 5 distinct Gu definitions share the effect; otherwise they are listed in `unanchored_effects` and left unchanged. Dry-run reports JSON; `-write -output` writes a separate candidate CSV; explicit `-in-place` replaces `Gu.csv` after creating `Gu.csv.bak` (refuses to overwrite an existing backup). |
| `match-food` | Flag `food_types` entries that reference missing items/Gu, and count low-nutrition food. `-write` remaps dangling references to a path/rank-matched Food. |
| `gen-recipe` | Generate missing refinement recipe rows (unit = `(id, rank)`). Success rate, costs, material quantity (`material_qty`, `material_qty_scale`) and required-Gu quantity (`required_gu_qty`, `required_gu_rank_delta`, `required_gu` map) are config-driven via `-config`. |

Example config (optional, `{}` falls back to defaults):

```json
{
  "balance": { "tol": 0.25, "min_score_frac": 0.01 },
  "recipe":  { "material_qty": 10, "material_qty_scale": 1.0, "required_gu_qty": 1 }
}
```

```bash
go run ./cmd/gudata check-effects -data data
go run ./cmd/gudata balance -data data
go run ./cmd/gudata balance -data data -config data/gudata.json -write -output /tmp/Gu.bal.csv
go run ./cmd/gudata rebalance -data data
go run ./cmd/gudata rebalance -data data -tolerance 0.35 -write -output /tmp/Gu.rebalanced.csv
go run ./cmd/gudata rebalance -data data -in-place
go run ./cmd/gudata match-food -data data
go run ./cmd/gudata gen-recipe -data data -write -output /tmp/Recipe.gen.csv
```

After writing a candidate CSV, import it through the Admin panel ("Dữ liệu") —
the tool never touches the database.

## Prerequisites

- Go theo `backend/go.mod`.
- Node.js/npm tương thích Next.js 16.
- PostgreSQL cho backend.
- Redis (bắt buộc).

## Environment variables

Backend đọc `.env` từ các vị trí: `.env`, `backend/.env`, `../.env`, `../../.env`.

Các biến thường dùng:

| Variable | Mục đích |
| --- | --- |
| `PORT` | Port backend, mặc định `8080`. |
| `TLS_ENABLED`, `TLS_ADDR` | Bật HTTPS trong Go. `TLS_ENABLED=true` phục vụ HTTPS trên `TLS_ADDR` (mặc định `:8443`) và bắt buộc `TLS_CERT_FILE` + `TLS_KEY_FILE`; thiếu cert thì process dừng, không fallback HTTP. Dev để trống và dùng `PORT`. |
| `WT_ENABLED`, `WT_ADDR` | Bật QUIC/WebTransport. `WT_ENABLED=true` cần cert/key như trên; `WT_ADDR` mặc định `:8443` (UDP). `TLS_ENABLED=true` cũng bật WT nếu chưa đặt `WT_ENABLED`. |
| `WS_FALLBACK_ENABLED` | Bật route WebSocket `/api/ws` cho world và UI realtime, mặc định bật; đặt `false` để tắt khẩn cấp. |
| `TLS_CERT_FILE`, `TLS_KEY_FILE` | Cert/key PEM (fullchain + privkey) cho HTTPS + WebTransport. |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Kết nối PostgreSQL; có default local trong `db.Connect()`. |
| `DB_STATEMENT_TIMEOUT_MS`, `DB_LOCK_TIMEOUT_MS` | Timeout câu lệnh/chờ lock PostgreSQL cho request path; mặc định lần lượt 15000/10000 ms. |
| `DB_MIGRATION_STATEMENT_TIMEOUT_MS`, `DB_MIGRATION_LOCK_TIMEOUT_MS` | Timeout riêng cho từng câu lệnh/lock của `InitSchema` lúc boot (backfill/consolidate trên bảng lớn); mặc định 300000/10000 ms. Request path vẫn giữ timeout chặt. |
| `DB_MIGRATION_RETRY_ATTEMPTS`, `DB_MIGRATION_RETRY_DELAY_MS` | Số lần thử lại câu lệnh khởi tạo khi PostgreSQL trả lỗi lock/deadlock/serialization tạm thời và độ trễ ban đầu; mặc định 6 lần/1000 ms, giới hạn 20 lần/60000 ms và backoff tối đa 10 giây. |
| `DB_MIGRATION_ADVISORY_LOCK_TIMEOUT_MS` | Thời gian tối đa chờ một instance GuWorld khác hoàn tất `InitSchema`; mặc định 120000 ms, tối đa 3600000 ms. |
| `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS` | Giới hạn pool kết nối PostgreSQL. Mặc định 30/10 (RDS hiện rất nhỏ, `max_connections` 81); phải để dưới trần `max_connections` để không tranh slot với client khác. Nâng lên sau khi upsizing instance. |
| `DB_CONNECT_TIMEOUT_SECONDS` | Timeout thiết lập kết nối lib/pq; mặc định 5 giây. Chặn việc RDS chập chờn làm treo dial ~2 phút và cạn pool. |
| `CORS_ALLOWED_ORIGINS` | Danh sách origin frontend, mặc định `http://localhost:3000`. |
| `JWT_SECRET` | Ký token/cookie auth. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `MAIL_FROM` | Cấu hình SMTP (ví dụ: Brevo) để gửi mã OTP đăng ký. |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_USERNAME`, `REDIS_PASSWORD`, `REDIS_SSL` | Redis connection config. Redis TLS luôn xác minh certificate. |
| `REDIS_CONNECT_ATTEMPTS`, `REDIS_CONNECT_DELAY_MS` | Số lần thử kết nối Redis lúc boot và độ trễ backoff cơ bản; mặc định 8 lần/1000 ms. Hết lượt thử mà vẫn lỗi thì server vẫn chạy với fallback in-memory (cache client = nil) thay vì crash-loop. |
| `REDIS_TLS_SERVER_NAME`, `REDIS_TLS_CA_FILE` | Tùy chọn hostname xác minh và CA PEM riêng khi `REDIS_SSL=true`. |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` | Dùng cho `/auth/google/*`; chỉ cấu hình bằng environment/secret store. |
| PayOS vars | Dùng cho topup/webhook nếu bật payment thật. |
| R2 vars (`CF_ENDPOINT`, `CF_BUCKET`, `CF_KEY`, `CF_SEC`, `CF_PUBLIC_BASE_URL`) | Dùng cho upload avatar lên Cloudflare R2. `CF_KEY`/`CF_SEC` là S3 Access Key ID/Secret, `CF_PUBLIC_BASE_URL` là custom domain public (vd `https://cdn.cogioi.io.vn`). |
| `GRAFANA_URL`, `GRAFANA_API_KEY`, `GRAFANA_USER`, `GRAFANA_SERVICE_NAME` | Cấu hình log shipping tới Grafana Cloud qua OpenTelemetry OTLP HTTP (`/otlp/v1/logs`). |
| `AUTO_BAN_FINGERPRINT_LIMIT`, `AUTO_BAN_REFERRAL_LIMIT`, `AUTO_BAN_DURATION_HOURS`, `AUTO_BAN_BLOCK_DEVICE` | Auto-ban chống lạm dụng: ngưỡng số tài khoản/thiết bị (mặc định 5), ngưỡng referral đáng ngờ (mặc định 5), thời hạn ban giờ (mặc định 72, ≤0 = vĩnh viễn), và có chặn device kèm hay không (mặc định true). Đặt 0 để tắt rule tương ứng. |
| `REGIONAL_WALL_SUPPRESSION_ENABLED` | Bật/tắt phạt [Áp Chế Giới Vực] (`regional_wall_suppression`) khi người chơi (hoặc thành viên tổ đội theo đội trưởng) vượt Giới Vực Bích giữa các Đại Vực. Mặc định bật; đặt `false`/`0`/`off`/`no` để tắt hoàn toàn việc gắn status. Cần restart backend sau khi đổi. |
| `MONSTER_RANK_SCALING_ENABLED` | Bật/tắt nhân `RankScale(rank)` vào chỉ số quái/boss trong `models.ScaleMonsterStats`. Mặc định bật; đặt `false`/`0`/`off`/`no` để chỉ số = `base × typeMultiplier × subRankMultiplier` (bỏ cả bump rank 10). Áp cho mọi đường spawn: world monster, preview vùng, linh thú tông môn, boss phi thăng/độ kiếp. Cần restart backend sau khi đổi. |
| `MONSTER_STAT_MULTIPLIER` | Hệ số nhân toàn cục cho HP/ATK/DEF của quái/boss, áp **sau** rank/type/sub-rank (mặc định `1.0`). Dùng để nerf/buff tạm thời khi cơ chế chiêu realtime chưa hoàn tất, ví dụ `0.5` = còn 50%. Thiếu/không hợp lệ/≤ 0 → `1.0`. Cần restart backend sau khi đổi. |
| `COMBAT_MAX_GU_ACTIONS_PER_TURN` | Số chiêu Cổ/Sát Chiêu tối đa trong một kế hoạch combat turn-based (mặc định `3`, giới hạn hợp lệ `1`–`10`). Đánh thường tùy chọn ở cuối không tính vào giới hạn; giá trị sai dùng mặc định. Cần restart backend sau khi đổi. |
| Realtime MMO | Bật mặc định cho mọi tài khoản đã đăng nhập. Backend tự nạp toàn bộ map công khai khớp release; map Tiên Khiếu/độ kiếp riêng vẫn được admission theo instance và trạng thái nhân vật. Không cần env bật, chọn zone hay allowlist. |
| `MMO_MAX_PLAYERS_PER_ZONE` | Giới hạn admission mỗi zone; mặc định `5`, cho phép `1`–`100`. |
| `MMO_INSTANCES_PER_ZONE` | Số instance của mỗi map trong process; mặc định `1`, cho phép `1`–`8`. Join được route tới instance ít người nhất còn capacity. Mỗi map được đọc/kiểm tra một lần lúc startup và dùng chung artifact collision/spawn giữa các instance; world, quái và runner 20 Hz chỉ được tạo khi instance có lease đầu tiên và bị giải phóng sau 5 phút không còn lease/subscriber/projectile/event chờ. |
| `MMO_MAP_DIR` | Thư mục map JSON server đọc lúc startup; mặc định `data/mmo/maps`. Mỗi map đọc `<map_id>.json`; thiếu/sai map hoặc version không khớp release đi kèm code thì manager không khởi chạy và admission fail closed. Việc kiểm tra vẫn chạy đủ cho mọi map cấu hình kể cả khi chưa có instance nào hoạt động. |
| `MMO_GU_PROFILES` | Legacy setting for the retired realtime combat runtime; it is no longer read by the server. |

DB đã khởi tạo trước khi thêm các cột `world_*` cần backup, drain realtime/movement, rồi chạy thủ công `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f backend/scripts/migrate_player_world_location.sql` trước khi nâng backend. Không chạy script khi server cũ còn nhận movement hoặc checkpoint.

Authenticated realtime world stream trên WebTransport nhận `world.input` sau khi client chờ `world.welcome`: server xác thực sequence/vector, giới hạn 30 input/giây mỗi kết nối, xếp qua queue riêng tối đa 64 lệnh rồi mới gửi vào queue mô phỏng có giới hạn. Input không kèm player ID; identity lấy từ phiên đăng nhập và controller lease. Snapshot preview 20 Hz chỉ chứa entity trong AOI tròn bán kính 32 world units, có delta từ baseline định kỳ, `removed` và `ack_seq`; queue per-session coalesce trạng thái mới nhất.

Frontend:

| Variable | Mục đích |
| --- | --- |
| `NEXT_PUBLIC_BACKEND_URL` | Backend base URL; mặc định `http://localhost:8080`. Production dùng `https://api.cogioi.io.vn`. |
| `INTERNAL_BACKEND_URL` | Backend base URL cho SSR/server-side fetch; mặc định dùng `NEXT_PUBLIC_BACKEND_URL`. |
| `NEXT_PUBLIC_SITE_URL` | Origin của frontend cho metadata, robots và sitemap; mặc định `https://www.cogioi.io.vn`. |
| `NEXT_PUBLIC_REALTIME_URL` | URL WT đầy đủ, ví dụ `https://rt.cogioi.io.vn/api/wt`; mặc định suy ra từ backend (HTTPS + `/api/wt`). Client thử WT trước. |
| `NEXT_PUBLIC_REALTIME_WS_URL` | URL WebSocket fallback; nếu có URL WT cấu hình thì dùng cùng host và đổi sang `/api/ws`, nếu không thì suy ra từ `NEXT_PUBLIC_BACKEND_URL`. Production host phải phục vụ TCP/TLS. |
| `NEXT_PUBLIC_MMO_ZONE_ID` | Zone dự phòng cho preview/test. Khi đăng nhập, client ưu tiên `current_region` của nhân vật để chọn map KHU_VUC/SAFE_ZONE tương ứng. |
| `NEXT_PUBLIC_MMO_MAP_URL` | URL map JSON client tải; mặc định `/game/maps/<zone>.json`. File này phải giống bản trong `backend/data/mmo/maps/`. |

Map/content versions follow the shipped map release and are not environment
variables; the backend and frontend use the same built-in values.

Frontend business screens do not require a world connection. While the world or
UI realtime stream is unavailable, the basic feature navigation remains accessible. When
the UI stream is down, HTTP reads refresh trade and the open chat every 5 seconds
after the previous round completes; player status, Gu inventory, recipes and unread
notification count refresh at most once per 15 seconds. Polling pauses in hidden
or offline tabs, stops after tab replacement/unmount, and stops when UI realtime recovers.
Bao Hoang chat sends use HTTP once, with no replay through another transport after
an uncertain mutation result. World movement and non-combat synchronization still use realtime transport.
Client chuyển sang WebSocket khi WebTransport không được hỗ trợ hoặc không thiết lập được kết nối. World và UI dùng chung một kết nối đã chọn trong mỗi tab. Khi mất kết nối, các request đang chờ bị từ chối; nút kết nối lại hoặc sự kiện online có thể khởi động lại ngân sách retry. Trình duyệt không dùng được cả hai transport vẫn có các chức năng HTTP, còn world map bị khóa.

The Phaser world movement runtime and its realtime transport remain for movement, presence, chat, and synchronization. Realtime combat, skill hotbars, combat simulation, combat session settlement, and their event logs have been retired. Death Arena and Martial Arena use turn-based encounter workers. PvP and other arena conversion gaps are documented in `docs/architecture/REALTIME_COMBAT_TO_TURN_BASED.md`.

Frontend:

```bash
cd frontend
npm run lint
npm run build
npm run validate:landscape  # Tiled scenery IDs, maps, and spawn entry buffer
```

## Background workers

Active workers (`cmd/server/main.go`): Hunger, Tribulation, **Refine Session** (30s sweep — resolves overdue
`refine_sessions`, idempotent với lazy GET `/refine/sessions*`), **World Resource Points** (boot sweep + 10 phút/lần,
top-up mỏ công khai `scope='WORLD'` cho mọi vùng active — idempotent nhờ partial unique index `(region_id,x,y)`),
party/pets/society/arena, batch persister…

Every recurring background task goes through `backend/internal/worker/runner.go`:

```go
// Tick loop + panic recovery, spawned as its own goroutine.
worker.RunEvery(ctx, 3*time.Hour, "Hunger Worker", func(ctx context.Context) {
	processScheduledTicks(ctx, db)
})

// Same recovery, for work that runs outside RunEvery (startup pre-runs, or
// loops that must flush on shutdown).
worker.SafeRun("Batch Persister", ctx, flushAll)

// Error-returning variant: non-nil errors and panics are recorded into
// the admin diagnostic trace ring buffer (monitor.RecordWorkerEvent).
worker.RunEveryE(ctx, time.Minute, "Custom Worker", func(ctx context.Context) error {
	return doWork(ctx)
})
```

Rules:

- A panic inside a tick is logged with a stack trace and the loop keeps running. Panics
  and `RunEveryE` error returns are recorded in the admin monitor trace ring buffer. Never
  hand-roll a `for { select { case <-ctx.Done(): ... case <-ticker.C: ... } }` loop:
  it has no recovery, and a panic in a worker goroutine takes the whole process down.
- Always pass `ctx`; `RunEvery` stops on cancellation.
- Workers that must flush on shutdown (batch persister, last-activity batcher) or that
  need a `done` channel keep their own loop, but wrap the tick body in `SafeRun` (or `SafeRunE`).
- Work that a tick hands off (because it opens a transaction or runs heavy CPU) goes to a
  bounded pool rather than running inline, so one slow item cannot blow the tick budget.
  Example: bounded dispatch is appropriate for expensive tick work that remains active; the retired world monster attack dispatcher is no longer present.

### Observability & Debugging

- **Trace ID**: Mỗi request HTTP và tick worker có một Trace ID độc bản (`X-Trace-ID`), tự động liên kết từ Client Toast -> HTTP Response -> Middleware -> Context Logger -> Admin Monitor.
- `LOG_LEVEL`: Mức thấp nhất được ghi ra log. Hỗ trợ `DEBUG`, `INFO` (mặc định), `WARN`, `ERROR`, `DPANIC`, `PANIC`, `FATAL`; không phân biệt hoa thường và bỏ khoảng trắng ở hai đầu. Giá trị không hợp lệ phát cảnh báo rồi dùng `INFO`. Cần khởi động lại backend sau khi đổi `.env`.
- `DEBUG_ERRORS=true`: Chỉ khi bật tường minh ngoài production, response lỗi từ `httperr` mới đính kèm trường `"debug"` chứa lỗi gốc. Mặc định JSON không lộ lỗi gốc; production luôn ẩn trường này. Lỗi vẫn được ghi trong structured Zap log kèm trace ID. Thông điệp JSON giữ phần mô tả thân thiện, không kèm `err.Error()`.
- Lỗi nội bộ trong handlers cần đi qua `httperr.Respond` (hoặc `httperr.AbortInternal`) với nguyên nhân gốc. Middleware ghi riêng lỗi handler và bản tổng kết request; dùng cùng trace ID để nối hai bản ghi.
- Tra cứu Trace: Trên giao diện Admin (tab "Giám sát"), nhập mã Trace ID vào ô tìm kiếm để xem toàn bộ thông tin chi tiết của request/worker gặp lỗi.

## Quản lý dữ liệu gameplay

Quy trình bootstrap/khôi phục admin, điều tra audit, rollback import và xử lý
outage nằm tại [`ADMIN_OPERATIONS_RUNBOOK.md`](ADMIN_OPERATIONS_RUNBOOK.md).

- Nguồn CSV chuẩn nằm tại `backend/data/`.
- Admin → **Dữ liệu** hỗ trợ xem danh sách dataset, import/export từng bảng hoặc toàn bộ.
- Import ghi dữ liệu vào PostgreSQL; việc chỉ sửa file CSV không làm thay đổi game đang chạy.
- Các registry Cổ, vật phẩm, chủng tộc, thể chất, trạng thái, quái và vùng được cache trong tiến trình. Khởi động lại backend sau khi import nếu registry tương ứng không tự refresh.

## Documentation maintenance checklist

- Route thay đổi: cập nhật `docs/guides/API.md`.
- Component/view gameplay thay đổi: cập nhật `PROJECT.md` và `WIKI.md` nếu ảnh hưởng người chơi.
- CSV/schema thay đổi: cập nhật `docs/guides/GAME_DATA.md`.
- Lệnh chạy/env thay đổi: cập nhật tài liệu này và README tương ứng.

## Effect catalog Block v2 migration

Stop the backend, then run `cd backend && go run ./cmd/migrate_effect_catalog`.
The command snapshots all seven effect owners, rejects hash conflicts and unsafe
legacy conversions, and commits the backfill in one serializable transaction.
Restart the backend after success; the migration deliberately does not hot-swap
registries. Restore the exact stored payloads with
`go run ./cmd/migrate_effect_catalog -rollback <revision>`.
In a later rollout, after every process has restarted on the validated catalog,
install the block-only checks with
`go run ./cmd/migrate_effect_catalog -install-constraints`.
