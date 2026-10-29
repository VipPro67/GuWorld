# Development Guide

## Auditing legacy effect data

Run `go run ./cmd/migrate_gu_effects -format gu -source data/Gu.csv` from
`backend/` for a non-destructive JSON audit. Candidate files are only written
when both `-write` and a non-source `-output` path are supplied. See
`docs/GAME_DATA.md` for report fields and the migration gate. Non-empty
candidates remain closed until runtime shadow evidence is bound to the selected
snapshot and catalog revision in the machine-readable report.

## CSV authoring tools (`gudata`)

`go run ./cmd/gudata <subcommand>` operates directly on `backend/data/*.csv`.
Every subcommand is dry-run by default; mutating runs require `-write` and a
distinct `-output` path (never overwrites source). Reports are JSON on stdout.

| Subcommand | Purpose |
| --- | --- |
| `check-effects` | Validate `Gu.csv` (full catalog + Block-v2 + unclassified-key) and `Item.csv` (Block-v2). Exits non-zero on errors. |
| `audit-realtime` | Classify every Gu block for the realtime zone (`SUPPORTED`/`NEEDS_REAUTHOR`/`UNSUPPORTED`/`NOT_ACTIVE`), emit coverage CSV/JSON and MVP check. `-strict` fails on unclassified operations or missing MVP. |
| `transpile-realtime-gu` | Generate the catalog-only MMO hotbar profiles from `Gu.csv`, preserving explicitly authored profiles from the selected override catalog; actions are emitted only when every cast effect is represented by the current realtime contract. |
| `compile-realtime-gu` | Compile authored realtime Gu profiles against `Gu.csv` + combat contract, fail closed on invalid timing/mode/effect, optionally write the immutable compiled artifact. |
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
go run ./cmd/gudata audit-realtime -data data -csv ../docs/mmo/gu-coverage.csv -strict
go run ./cmd/gudata transpile-realtime-gu -data data -overrides data/mmo/gu/authored_overrides.json -output data/mmo/gu/transpiled_profiles.json -compile-report /tmp/gu-compiled-profiles.json
go run ./cmd/gudata compile-realtime-gu -data data -profiles data/mmo/gu/transpiled_profiles.json -report ../docs/mmo/gu-compiled.json
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
| Realtime MMO | Bật mặc định cho mọi tài khoản đã đăng nhập. Backend tự nạp toàn bộ map công khai khớp release; map Tiên Khiếu/độ kiếp riêng vẫn được admission theo instance và trạng thái nhân vật. Không cần env bật, chọn zone hay allowlist. |
| `MMO_MAX_PLAYERS_PER_ZONE` | Giới hạn admission mỗi zone; mặc định `5`, cho phép `1`–`100`. |
| `MMO_INSTANCES_PER_ZONE` | Số instance của mỗi map trong process; mặc định `1`, cho phép `1`–`8`. Join được route tới instance ít người nhất còn capacity. Mỗi map được đọc/kiểm tra một lần lúc startup và dùng chung artifact collision/spawn giữa các instance; world, quái và runner 20 Hz chỉ được tạo khi instance có lease đầu tiên và bị giải phóng sau 5 phút không còn lease/subscriber/projectile/event chờ. |
| `MMO_MAP_DIR` | Thư mục map JSON server đọc lúc startup; mặc định `data/mmo/maps`. Mỗi map đọc `<map_id>.json`; thiếu/sai map hoặc version không khớp release đi kèm code thì manager không khởi chạy và admission fail closed. Việc kiểm tra vẫn chạy đủ cho mọi map cấu hình kể cả khi chưa có instance nào hoạt động. |
| `MMO_GU_PROFILES` | Danh sách catalog profile Gu realtime, phân tách bằng dấu phẩy; mặc định `data/mmo/gu/transpiled_profiles.json`. Compile/validate lúc startup khi realtime bật; đường dẫn tương đối được dò từ thư mục đang chạy lên các thư mục cha (hỗ trợ `backend/cmd/server`). Lỗi sẽ log và hotbar realtime fail-closed, không có chiêu mẫu thay thế. |

DB đã khởi tạo trước khi thêm các cột `world_*` cần backup, drain realtime/movement, rồi chạy thủ công `psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f backend/scripts/migrate_player_world_location.sql` trước khi nâng backend. Không chạy script khi server cũ còn nhận movement hoặc checkpoint.

Authenticated WS protocol v1 nhận `world.input` sau khi client chờ `world.welcome`: server xác thực sequence/vector, giới hạn 30 input/giây mỗi kết nối, xếp qua queue riêng tối đa 64 lệnh rồi mới gửi vào queue mô phỏng có giới hạn. Input không kèm player ID; identity lấy từ phiên đăng nhập và controller lease. Snapshot preview 20 Hz chỉ chứa entity trong AOI tròn bán kính 32 world units, có delta từ baseline định kỳ, `removed` và `ack_seq`; queue per-session coalesce trạng thái mới nhất. Client interpolation/resync và budget byte chưa làm xong.

Frontend:

| Variable | Mục đích |
| --- | --- |
| `NEXT_PUBLIC_BACKEND_URL` | Backend base URL; mặc định `http://localhost:8080`. |
| `NEXT_PUBLIC_MMO_ZONE_ID` | Zone dự phòng cho preview/test. Khi đăng nhập, client ưu tiên `current_region` của nhân vật để chọn map KHU_VUC/SAFE_ZONE tương ứng. |
| `NEXT_PUBLIC_MMO_MAP_URL` | URL map JSON client tải; mặc định `/game/maps/<zone>.json`. File này phải giống bản trong `backend/data/mmo/maps/`. |

Map/content versions follow the shipped map release and are not environment
variables; the backend and frontend use the same built-in values.

Sau đăng nhập và onboarding, Phaser 4.2.1 là gameplay world duy nhất; `dashboard`/`world-map` cũ chỉ
còn là view ID tương thích, các component map turn-based đã bị gỡ khỏi frontend và không fallback
client kết nối Phaser MMO cho mọi tài khoản đã đăng nhập; map/content versions được lấy từ release đã đóng gói. Client gửi
`world.join` bằng zone ID của `current_region`, tải map tương ứng từ `/game/maps/<zone>.json`, render snapshot AOI 20 Hz, WASD + click-to-move và để server xác nhận vị trí/HP. Không có
Phaser physics và không có sát thương do client tính. Thiếu cấu hình zone thì Phaser hiện preview/error,
không quay lại world cũ. Vùng khởi đầu lưu trong `current_region` vẫn được dùng làm metadata để xác
nhận map mà nhân vật được phép vào.
Map collision dùng chung một JSON ở `backend/data/mmo/maps/` và `frontend/public/game/maps/`;
format và quy trình copy nằm trong `docs/mmo/map-format.md`. Monster spawn khai báo trong map
được tạo lại mỗi khi instance được kích hoạt từ `monster_templates` và chạy AI server-owned. Combat preview dùng phím
1–6 (`TARGET`/`DIRECTION`/`GROUND`/`SELF`), damage/projectile/AoE do server tính; PR-19 thêm
status tick/DoT/HoT/stack/cleanse, freeze chặn di chuyển/cast và hidden passive preview
(HP thấp + 3 hit → shield, ICD). HUD React lấy hotbar/mana/cooldown/cast từ snapshot; aim DIRECTION/GROUND
là hai bước (chọn skill → click xác nhận), projectile do server mô phỏng. Art combat hiện là placeholder
procedural (`placeholder-1`) với animation idle/walk/cast/hit/death và VFX pool; xem
`docs/mmo/assets.md`. Click-to-move dùng A* preview phía client rồi gửi từng waypoint
`world.move_target`; server vẫn xác nhận từng đoạn và phát `MOVE_TARGET_REJECTED` khi chặn.
Camera có deadzone/zoom/shake + chỉ báo mục tiêu ngoài màn hình. Loadout hotbar lúc join lấy từ
`players.equipped_gus` + `player_gu` đang equipped/ACTIVE, match profile realtime đã compile và
re-slot 1–6; Gu chưa compile bị bỏ qua, fallback preview nếu không có Gu nào. View MMO có drawer
Túi/Tu luyện/Tổ đội/Cài đặt + chat overlay; mở panel đặt UI-modal chặn input Phaser nhưng server
không dừng zone. Loot auto-grant idempotent qua `realtime_loot_grants`; vị trí MMO được checkpoint
trong `players` (join resume, leave/disconnect flush, worker 10s), còn
`player_world_state_v2` chỉ giữ HP/shield và phiên bản map, xem `docs/mmo/recovery.md`.
Death Arena và Martial Arena tạo `realtime_combat_sessions` mode `tournament` trên map ảo cô lập;
participants được route vào cùng instance, worker chỉ chuyển vòng sau khi session kết thúc. Disconnect
được xử thua sau 30 giây; Death Arena lưu HP/Mana còn lại cho vòng sau, Martial Arena hồi đầy chỉ số
mỗi trận. Kết quả và phần thưởng được ghi trong transaction settlement của realtime death handler.
Snapshots trận riêng lưu combat events cùng checkpoint HP/Mana trong
`realtime_combat_session_events`; khi trận kết thúc, các event được dùng làm
combat log cho lịch sử đấu trường, võ đài và chiến tranh tông môn.
Chiến tranh tông môn dùng session `sect_war`, có Hộ Sơn Thần Thú trên map ảo,
settlement giữ logic cướp kho/khiên phòng thủ/đóng góp và Qi. Chỉ thành viên
đang có lease realtime được đưa vào trận; sủng vật chưa được mô hình hóa ở map này.
Portal metadata được parse/render và `Manager.TransferCharacter` handoff atomic trong process, nhưng
chưa tự trigger portal/update DB/scene swap; xem `docs/mmo/transfer.md`. Ruleset map `pve/pvp/safe`, party-aware
harm rules, boss threat + splash và nhãn/tint party trong snapshot; xem `docs/mmo/pvp-party.md`.
Chưa có corpse/F loot, death/respawn UI, party loot/duel consent hoặc live equip refresh.

Mô phỏng MMO headless (không cần DB/Redis): `cd backend && go run ./cmd/mmo_headless -ticks 10000`.
Nó chạy 2 player và 10 monster qua tick cố định 20 Hz rồi in checksum snapshot.
Smoke slice đầy đủ hơn: `go run ./cmd/mmo_slice -players 5 -mobs 10 -ticks 400` (movement/cast/death/metrics).
Synthetic QoS: `go run ./cmd/mmo_bench -players 100 -mobs 50 -ticks 1000`; report ở
`docs/mmo/benchmark.md`. Checklist Gate B: `docs/mmo/vertical-slice.md`.
Trong MMO scene, **F3** bật/tắt debug HUD (tick/FPS/network/prediction).
CI/observability/content versioning: `docs/mmo/operations.md`; admin stats endpoint
`GET /api/admin/monitor/realtime` (permission `monitoring.read`).
Soak/reconnect: `go run ./cmd/mmo_soak -duration 60s -players 10 -mobs 20 -reconnect-every 5s`;
fuzz/security plan: `docs/mmo/security.md`.
Release gate tự động (format/vet/test/race/Gu/map/slice/soak/frontend):
`scripts/mmo-release-gate.sh`. Gate D checklist + sign-off: `docs/mmo/release-checklist.md`;
SLO/capacity candidate: `docs/mmo/slo.md`; roadmap sau beta: `docs/mmo/roadmap.md`.
Tổng hợp giới hạn còn lại của PR-01…PR-36: `docs/mmo/remaining-limitations.md`.

## Run with Docker Compose

Bạn có thể khởi chạy nhanh toàn bộ môi trường backend (bao gồm Go app, PostgreSQL, Redis và Caddy làm reverse proxy) bằng Docker Compose:

```bash
docker-compose up --build
```

Production Compose lấy DB/JWT/OAuth từ `backend/.env` và thông tin Grafana từ
root `.env`; không commit hai file này. Redis production chỉ ở trong Compose
network. `docker-compose.local.yml` bind Redis vào loopback để tiện phát triển.

Lệnh này sẽ tự động:
1. Tải và cấu hình PostgreSQL (`gu_db`) và Redis.
2. Build Go backend dựa trên `backend/Dockerfile`; CSV nguồn nằm trong `backend/data/` để quản trị viên import khi cần.
3. Khởi chạy Caddy Server để làm reverse proxy và tự động cấp chứng chỉ SSL (HTTPS) cho tên miền `api.cogioi.io.vn` (cấu hình trong `Caddyfile`).
4. Khởi chạy Grafana Alloy container (`guworld-alloy`) kết nối tới Docker Socket để tự động thu thập stdout/stderr logs của toàn bộ container và đẩy về Grafana Cloud Loki & Prometheus.
5. Sử dụng cơ chế healthcheck để trì hoãn khởi động Go app cho đến khi PostgreSQL đã hoàn toàn sẵn sàng nhận kết nối.

API backend sau đó sẽ lắng nghe tại `http://localhost:8080` (truy cập cục bộ từ host) và được expose ra ngoài qua Caddy tại `https://api.cogioi.io.vn`.

## Run backend (Manual)

```bash
cd backend
go run ./cmd/server
```

Server sẽ:

1. Kết nối DB.
2. Tạo/migrate schema.
3. Seed các bản ghi hệ thống tối thiểu; riêng quest hiện được đồng bộ từ `Quest.csv`. Các dictionary gameplay khác **không tự import**.
4. Load registry từ database. Nếu chưa có dữ liệu, vào Admin → **Dữ liệu** để import CSV rồi khởi động lại backend.
5. Start workers.
6. Listen HTTP.

`InitSchema` dùng PostgreSQL advisory lock để chỉ một instance thay đổi schema tại
một thời điểm, đồng thời thử lại có giới hạn khi gặp SQLSTATE `55P03`, `40P01`
hoặc `40001`. Nếu vẫn hết thời gian chờ, hãy kiểm tra transaction dài hoặc
`idle in transaction` trên PostgreSQL và tạm dừng import/traffic ghi trong cửa
sổ migrate; không nên chỉ tăng timeout vô hạn. Với rolling deployment, nên chạy
migration như một bước riêng trước khi đưa replica mới nhận traffic.

## Run frontend

```bash
cd frontend
npm install
npm run dev
```

Mở `http://localhost:3000`.

## Checks used in this repo

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
npm run lint
npm run build
```

## Background workers

Every recurring background task goes through `backend/internal/worker/runner.go`:

```go
// Tick loop + panic recovery, spawned as its own goroutine.
worker.RunEvery(ctx, 3*time.Hour, "Hunger Worker", func(ctx context.Context) {
	processScheduledTicks(ctx, db)
})

// Same recovery, for work that runs outside RunEvery (startup pre-runs, or
// loops that must flush on shutdown).
worker.SafeRun("Batch Persister", ctx, flushAll)
```

Rules:

- A panic inside a tick is logged with a stack trace and the loop keeps running. Never
  hand-roll a `for { select { case <-ctx.Done(): ... case <-ticker.C: ... } }` loop:
  it has no recovery, and a panic in a worker goroutine takes the whole process down.
- Always pass `ctx`; `RunEvery` stops on cancellation.
- Workers that must flush on shutdown (batch persister, last-activity batcher) or that
  need a `done` channel keep their own loop, but wrap the tick body in `SafeRun`.
- Work that a tick hands off (because it opens a transaction or runs heavy CPU) goes to a
  bounded pool rather than running inline, so one slow item cannot blow the tick budget.
  Example: `dispatchWorldMonsterAttack` (`handlers/world_monsters.go`) reserves a slot on
  a buffered channel and applies a realtime monster attack under `SafeRun` in a goroutine;
  when the pool is full the attack is deferred to its next cooldown instead of queueing.

## Quản lý dữ liệu gameplay

Quy trình bootstrap/khôi phục admin, điều tra audit, rollback import và xử lý
outage nằm tại [`ADMIN_OPERATIONS_RUNBOOK.md`](ADMIN_OPERATIONS_RUNBOOK.md).

- Nguồn CSV chuẩn nằm tại `backend/data/`.
- Admin → **Dữ liệu** hỗ trợ xem danh sách dataset, import/export từng bảng hoặc toàn bộ.
- Import ghi dữ liệu vào PostgreSQL; việc chỉ sửa file CSV không làm thay đổi game đang chạy.
- Các registry Cổ, vật phẩm, chủng tộc, thể chất, trạng thái, quái và vùng được cache trong tiến trình. Khởi động lại backend sau khi import nếu registry tương ứng không tự refresh.

## Documentation maintenance checklist

- Route thay đổi: cập nhật `docs/API.md`.
- Component/view gameplay thay đổi: cập nhật `PROJECT.md` và `WIKI.md` nếu ảnh hưởng người chơi.
- CSV/schema thay đổi: cập nhật `docs/GAME_DATA.md`.
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
