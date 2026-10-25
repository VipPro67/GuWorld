# GuWorld Phaser 4 Realtime — Branch Implementation Plan

**Nguồn kiến trúc:** `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` v2.0  
**Branch đề xuất:** `feature/phaser4-realtime-mmo`  
**Ngày lập:** 2026-10-24  
**Mục đích:** Chuyển master plan thành chuỗi PR nhỏ, có thể review và triển khai an toàn trên codebase hiện tại.

> Đây là kế hoạch triển khai, chưa phải xác nhận rằng Phaser hay MMO runtime đã được cài đặt. Phạm vi trước mắt là playable vertical slice: top-down 2.5D, WASD + click-to-move, combat target-lock + skillshot + AoE, Go authoritative, React cho UI nghiệp vụ và Phaser cho mô phỏng/render.

## 1. Kết quả cần đạt

### Vertical slice đầu tiên

- Người chơi đăng nhập bằng tài khoản hiện có và vào một zone MMO riêng qua feature flag.
- Một bản đồ Tiled có collision do server xác nhận; client có thể dùng WASD hoặc click-to-move.
- Nhiều client thấy vị trí, quái, cast và combat event của nhau qua WebSocket.
- Một bộ kỹ năng đại diện cho target, projectile/skillshot, ground AoE, shield, slow và dash chạy trong Go simulation.
- HP, vị trí hợp lệ, cooldown, hit, death và loot do server xác nhận; Phaser chỉ dự đoán phản hồi hiển thị.
- Inventory/Gu và các cửa sổ tu luyện tiếp tục dùng domain/UI React hiện có.
- Legacy world và arena vẫn chạy ngoài cohort/zone được bật MMO; có quy trình rollback.

### Không nằm trong slice đầu

100 người/zone, mobile controls, seamless world vô hạn, đầy đủ Gu parity, microservice tách zone, và xóa Pixi/legacy combat. Chỉ xem xét sau benchmark và nghiệm thu các gate trước.

## 2. Đối chiếu plan với repository hiện tại

Các điểm đã xác minh trong repo:

- `frontend/package.json` dùng Next 16 / React 19, Zustand và `pixi.js`; chưa có dependency Phaser. Đang có script `lint`, `typecheck`, `test`, `build`.
- Backend đã có WebSocket hiện hữu ở `backend/internal/ws/`, router `backend/internal/handlers/world_ws.go`, world movement/monster handlers và PvP realtime trong `backend/internal/match/`.
- `backend/cmd/server/main.go` khởi tạo registry, worker legacy và `match.StartTickLoop(ctx)`. Cần thêm runtime mới như một thành phần độc lập, có lifecycle rõ ràng; không thay tick loop PvP hiện tại.
- Go effects đã có contract/runtime/executor ở `backend/internal/gameplay/effects/`; không nên tạo một catalog song song trước khi audit semantics và ranh giới tái sử dụng.
- Backend đang chạy worker quái legacy và chúng có thể kích hoạt encounter theo lượt. MMO zone cần quyền sở hữu/quy tắc loại trừ rõ ràng trước khi đưa player hoặc monster vào đó.
- `frontend/src` cần được rà soát lại ngay đầu PR-01 để chốt các đường dẫn UI thực tế; các tên trong master plan dựa trên Archive.zip và có thể khác repo hiện tại.

**Điểm cần sửa trong master plan trước khi dùng như checklist file-by-file:** bảng nguồn có một số đường dẫn frontend/backend lấy từ Archive, còn code hiện tại cần inventory mới. PR-01 tạo inventory có đường dẫn thật và đánh dấu `present / change / new / verify`; mọi PR sau đó dùng inventory này thay cho giả định từ ZIP.

## 3. Quy tắc branch và phạm vi Git

Tên branch: `feature/phaser4-realtime-mmo`, tạo từ branch nền đã được team chọn (mặc định `deploy` nếu đó vẫn là branch tích hợp).

```bash
git status --short --branch
git switch -c feature/phaser4-realtime-mmo
```

Giữ file `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` nguyên trạng nếu nó chỉ là tài liệu tham chiếu chưa được track; không đưa thay đổi cá nhân vào PR nếu chưa xác nhận chủ sở hữu. Tài liệu này được thêm riêng ở `docs/plans/GUWORLD_PHASER4_REALTIME_BRANCH_PLAN.md`.

Quy ước PR:

- Mỗi PR có một thay đổi có thể chạy/test độc lập; giữ feature flag OFF mặc định.
- Không gộp legacy world/arena vào MMO runtime bằng các sửa đổi lớn tại chỗ.
- Không thêm cột DB nếu snapshot/checkpoint có thể thực hiện ngoài tick; nếu phải thêm thì cập nhật `schema_*.go` theo quy ước repo, không thêm ALTER patch.
- Mọi thao tác kinh tế phải dùng service/transaction hiện hữu hoặc adapter có idempotency; không ghi PostgreSQL theo từng tick.
- Mỗi PR cập nhật tài liệu phù hợp (`docs/API.md`, `docs/DEVELOPMENT.md`, `docs/GAME_DATA.md`, `WIKI.md`) khi thay đổi route, startup, schema/data hoặc gameplay.

## 4. Backlog triển khai theo PR

| PR | Mục tiêu / phạm vi | File/module trọng tâm | Điều kiện hoàn tất |
|---|---|---|---|
| 01 — Baseline & inventory | Ghi trạng thái Git, toolchain, route, websocket, frontend view/store, world workers, combat/effects và test hiện có. Tạo sơ đồ đường đi legacy và danh sách file thật. | `frontend/package.json`, `frontend/src/app/`, `frontend/src/components/`, `frontend/src/store/`, `backend/cmd/server/main.go`, `backend/internal/server/server.go`, `backend/internal/handlers/world_ws.go`, `backend/internal/ws/`, `backend/internal/match/`, `backend/internal/gameplay/effects/`; tài liệu này | Baseline lint/build/test được ghi nhận; file inventory có status `present/change/new/verify`; không sửa runtime behavior. |
| 02 — Runtime seam & safety flags | Thêm config/env/feature flag MMO, ownership guard và no-op startup lifecycle. Runtime mới không start khi flag OFF. Xác định cohort/zone ID và vùng không thể cùng lúc thuộc legacy worker. | `backend/cmd/server/main.go`, `backend/internal/server/server.go`, package mới `backend/internal/realtime/config`, `backend/internal/realtime/world`; `docs/DEVELOPMENT.md` | Flag OFF giữ nguyên startup; flag ON tạo runtime nhưng chưa nhận player; shutdown tôn trọng context; không đổi `match.StartTickLoop`. |
| 03 — Phaser host song song | Pin `phaser@4.2.1`; dynamic import phù hợp Next.js client; tạo host mount/unmount và scene rỗng. Đưa vào route/view riêng sau flag, giữ Pixi làm legacy. | `frontend/package.json` + lockfile, `frontend/src/app/page.tsx`, component host mới, `frontend/src/game/bootstrap/`, `frontend/src/game/scenes/` | Phaser canvas render được, Strict Mode/route đổi không nhân đôi canvas/game instance; flag OFF vẫn vào legacy. |
| 04 — Protocol v1 & fixtures | Định nghĩa versioned envelope, message size/validation, sequence, timestamp, ack, join/leave, input, cast, snapshot và rejection codes. Chia fixtures dùng chung Go/TS. | `backend/internal/realtime/protocol/`, `backend/internal/handlers/world_ws.go`, `backend/internal/ws/message.go`, `frontend/src/game/network/Protocol.ts`, `WorldSocket.ts`, `tests/fixtures/realtime/` | JSON fixtures decode/encode ổn định ở Go và TypeScript; malformed input/rate limit bị từ chối; không thay wire contract của PvP/legacy. |
| 05 — World session & gateway | Authenticated join zone, session state machine, bounded queues, reconnect policy sơ bộ, backpressure và giới hạn input; route message realtime tới owner. | `backend/internal/realtime/gateway/`, `backend/internal/realtime/world/`, `backend/internal/handlers/world_ws.go`, `backend/internal/ws/`; client `GameSession.ts`, `WorldSocket.ts` | Join/leave hai client hoạt động; queue đầy không làm block broadcast toàn server; client slow được resync/disconnect chính sách rõ ràng. |
| 06 — Headless simulation kernel | Tạo fixed-step 20Hz, zone owner đơn luồng, command queue có giới hạn, entity ID/revision, deterministic clock/RNG và tick metrics. Tạo bot harness tối thiểu. | `backend/internal/realtime/sim/`, `entity/`, `world/`, `telemetry/` | Headless test chạy N tick lặp lại cho cùng kết quả; panic/tick quá hạn có log/metric; không cần DB/Redis ở từng tick. |
| 07 — Bản đồ, units & collision | Chọn một map nguồn; sinh/kiểm tra collision/nav artifact cho server từ Tiled; định nghĩa unit/world-pixel transform và map revision hash. | `backend/data/` hoặc thư mục map được thống nhất, `scripts/` validator/generator, `backend/internal/realtime/movement/`, `nav/`, `frontend/public/game/maps/`, `frontend/src/game/world/` | Client/server cùng đọc revision; server collision tests cho tường/góc/biên; sai map revision bị từ chối khi join. |
| 08 — Movement authoritative | Một `MoveIntent` cho WASD và click-to-move; normalize diagonal, stop, sequence/ack, speed modifier, collision/slide, path target validation; client prediction và reconciliation. | `backend/internal/realtime/movement/`, `nav/`, protocol fixtures; frontend `input/{InputRouter,KeyboardMovement,ClickToMove}.ts`, `network/{Prediction,Reconciliation}.ts` | Test WASD/click/blur/key reset/wall/repath; không gọi `world.move_path` cho mỗi bước MMO; legacy handler không bị đổi cho user ngoài zone MMO. |
| 09 — AOI & snapshot interpolation | Spatial index, visibility policy, join/leave AOI, baseline/delta sequencing, server clock, client interpolation cho entities khác. | `backend/internal/realtime/spatial/`, `world/`, `gateway/`; frontend `SnapshotBuffer.ts`, `RemoteInterpolator.ts`, `WorldClock.ts`, `world/EntityRegistry.ts` | Hai client thấy đúng entity trong AOI; không ghost/duplicate khi vào-ra vùng; baseline mất gói dẫn đến full resync. |
| 10 — Map visual & input UX | Tiled render, camera follow, depth theo feet anchor, click arbitration, UI focus, chọn mục tiêu và reticle ground aim. | `frontend/src/game/scenes/WorldScene.ts`, `world/{MapLoader,CameraController,DepthSort}.ts`, `input/`, React game overlay | Top-down 2.5D slice dùng placeholder hợp pháp; click UI không di chuyển nhân vật; WASD hủy click path; resize/zoom không làm lệch aim. |
| 11 — Monster entities & AI | Spawn một nhóm quái test trong zone runtime; FSM aggro/leash/attack/cooldown; budget AI; loại monster MMO khỏi worker/encounter legacy trong zone đang được owner mới quản lý. | `backend/internal/realtime/ai/`, `entity/`, `world/`; adapter đọc registry/spawn hiện hữu; guards quanh legacy handlers/workers nếu cần | Monster state là server-owned; không tồn tại hai owner; không gọi turn-based encounter khi player ở MMO zone; target hợp lệ và leash test pass. |
| 12 — Combat pipeline v1 | Cast validate -> windup/instant -> target/ground/direction/self -> projectile/area hit -> server events. Tách policy/rules khỏi Phaser animation. | `backend/internal/realtime/combat/`, protocol; frontend `combat/{CastTelegraph,CombatEventPresenter,ProjectileView}.ts` | Test range/LOS/faction/safe zone/cooldown/resource/interruption; client `hit` message không thể gây damage; duplicate event không hit hai lần. |
| 13 — Effects audit & adapter | Đối chiếu `gameplay/effects` contract/runtime/executor và `Gu.csv`, `GuMoba.csv`, `Status.csv`; lập bảng supported/needs-native-handler/deferred. Viết compiler/translator realtime có lỗi rõ ràng. | `backend/internal/gameplay/effects/`, `backend/internal/realtime/effects/`, `backend/data/Gu.csv`, `GuMoba.csv`, `Status.csv`, `docs/GAME_DATA.md` | Không map turn thành milliseconds tùy tiện; unsupported op được report và từ chối kích hoạt; fixture cho damage/heal/shield/status/cleanse/hidden passive. |
| 14 — Gu loadout & hotbar | Load loadout qua domain service/adapter, validate ownership/revision, slot 1–6; bind cooldown/resource snapshot vào React HUD; target-lock, skillshot và ground AoE UX. | `backend/internal/realtime/persistence/`, `combat/`, existing `backend/internal/services/`; `frontend/src/components/game/`, `frontend/src/game/input/` | 6 kỹ năng đại diện hoạt động theo bốn targeting mode; account không thể cast Gu không sở hữu; React không nhận position mỗi frame. |
| 15 — Persist, loot & reconnect | Checkpoint thưa, safe spawn, session resume, idempotent reward/grant, inventory service transaction, crash recovery. | `backend/internal/realtime/{persistence,loot,gateway,world}/`, `backend/internal/services/inventory_service.go`, `backend/internal/db/schema_*.go` nếu thực sự cần; docs | Retry grant cùng id chỉ phát sinh một phần thưởng; reconnect không tạo duplicate entity; checkpoint không ghi DB theo tick. |
| 16 — UI integration & vertical slice | Tích hợp React inventory/tu luyện/party/chat theo bridge throttled; end-to-end login → zone → move → cast → kill → loot → bag → disconnect/rejoin. | `frontend/src/components/WorldPage.tsx` (sau khi xác minh đường dẫn), `frontend/src/store/`, `frontend/src/components/game/`, `backend/internal/server/server.go`, `tests/e2e/mmo/` | Gate playable slice đạt 2–5 người, map đơn, bộ skill representative, legacy unaffected, feature flag OFF mặc định ngoài staging. |
| 17 — Perf/security/soak | Đo tick p50/p95/p99, AOI, outbound bytes, RTT, GC/heap, FPS/frame time; fuzz protocol và test packet loss/reconnect/dupe. Chỉ tối ưu điểm nóng đo được. | `backend/internal/realtime/telemetry/`, `frontend/src/game/debug/`, `tests/load/`, CI workflows (nếu đã chọn CI) | Có report tái lập, cấu hình máy/test rõ; không ghi nhận mục tiêu 30/100 player là đạt nếu chưa đo. |
| 18 — Canary/cutover & mở rộng | Cohort rollout, rollback rehearsal; sau khi ổn định mới thêm portal/zone transfer, party/boss/PvP, process splitting, gỡ Pixi. | feature flags, `backend/internal/realtime/{transfer,world}/`, ops docs, frontend dependencies/canvas legacy | Không có hai owner cho player/monster/reward; rollback bảo toàn vị trí/phần thưởng; Pixi chỉ được gỡ trong PR riêng sau canary hoàn tất. |

## 5. Gate nghiệm thu

### G0 — Safe setup

- Phaser host có thể render song song với legacy; feature flag OFF mặc định.
- Protocol fixture chạy được ở Go và TypeScript; backend/frontend baseline checks được ghi trong PR.

### G1 — Multiplayer movement

- 2–5 client cùng zone; WASD và click-to-move cùng tuân thủ collision server.
- AOI/snapshot resync hoạt động; không có DB write per movement node.

### G2 — Combat slice

- Target-lock, skillshot, ground AoE; client không thể tuyên bố hit/damage.
- 3 loại quái và các hiệu ứng đại diện; quái không kích hoạt combat turn-based.

### G3 — RPG persistence

- Loadout tài khoản cũ; loot idempotent; inventory/tu luyện UI hiện có hoạt động; reconnect không mất/nhân đôi reward.

### G4 — Release readiness

- Benchmark tải, crash/dupe/security, staging canary và rollback đã ghi lại.
- Chỉ mở rộng vùng/người chơi theo số đo; 100 người/zone là mục tiêu đo sau, không phải tiêu chí mặc định của vertical slice.

## 6. Lệnh kiểm tra theo loại thay đổi

Không chạy tất cả checks ở giai đoạn chỉ viết tài liệu. Khi PR có code, dùng checks tương ứng:

```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm run lint && npm run typecheck && npm run build

# Concurrency-sensitive backend simulation
cd backend && go test -race ./...
```

PR protocol/map/effects cần thêm fixture/compiler checks; PR persistence cần retry/idempotency tests; PR movement/combat cần deterministic sim tests. Thêm script mới vào package/Makefile chỉ khi PR giới thiệu script đó.

## 7. Các rủi ro phải giải quyết trước khi code tương ứng

1. **Repo drift so với Archive.zip:** PR-01 phải chốt inventory thực. Không dựa mù vào tên component/store trong ZIP.
2. **Hai chủ sở hữu world:** phải định nghĩa zone ownership và exclusion với `StartWorldMonsterWorker`, `world_monster_encounter` và cache state trước khi mở zone.
3. **Semantics Cổ Trùng:** phải audit operation/trigger/status trước khi quy đổi thời gian theo tick; unknown operation không được âm thầm bỏ qua.
4. **WS shared capacity:** world snapshot không được làm nghẽn chat/meta; dùng queue/budget và chính sách slow-client.
5. **Economy:** reward phải idempotent/transactional; Redis không là nguồn duy nhất bảo đảm item grant.
6. **Phaser/Next lifecycle:** dynamic import client-only, Strict Mode cleanup, route transition, HMR và một WS gameplay session duy nhất.
7. **Kỳ vọng tải:** các con số ở master plan là target; chỉ claim đạt sau benchmark có cấu hình tái lập.

## 8. Việc bắt đầu ở PR-01

1. Tạo branch `feature/phaser4-realtime-mmo` từ branch tích hợp đã chọn.
2. Ghi `git status`/commit base và không đưa file untracked cá nhân vào PR ngoài chủ đích.
3. Chụp baseline: `go test ./...`, `npm run lint`, `npm run typecheck`, `npm run build`; ghi môi trường/toolchain và kết quả, không sửa lỗi ngoài scope nếu chưa tách issue.
4. Lập bảng file inventory thực tế cho các đường đi React world, Pixi, Zustand, WS, route, monster worker, encounter, match simulation và effect runtime.
5. Cập nhật mục “repository paths verified” trong master plan hoặc link tới inventory mới; phân loại từng đường dẫn Archive là `same`, `renamed`, `missing`, `new`.
6. Mở PR-02 chỉ sau khi ownership conflict legacy/MMO và flag boundary đã được review.

## 9. Giả định và còn cần xác nhận trong PR-01

- Branch nền tích hợp: `deploy` (xác nhận lại bằng repo policy trước khi tạo branch thật).
- Phaser pin: `4.2.1` như ADR-01 trong master plan.
- MVP tick: 20 Hz như master plan; thay đổi chỉ dựa trên benchmark.
- Feature flag MMO OFF mặc định; staging là môi trường đầu tiên được bật.
- Map đầu tiên và 6 kỹ năng cụ thể được chọn sau kiểm kê asset/data và kiểm thử effects, không suy ra từ tên Gu trong CSV.
