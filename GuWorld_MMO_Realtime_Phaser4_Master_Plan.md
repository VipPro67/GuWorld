# GUWORLD — MASTER IMPLEMENTATION PLAN v2.0
## MMORPG Realtime 2D — Phaser 4.2.1 + Go Authoritative World Server

**Cập nhật:** 24/10/2026  
**Nguồn đối chiếu:** `Archive.zip` (frontend, backend) và master plan v1 đã tạo trước đó.  
**Trạng thái:** kiến trúc + backlog đang được triển khai trên branch `mmo/realtime-foundation`; PR-02/PR-04–PR-36 đã có một phần code và test. Phaser 4.2.1 đã được pin; entry MMO join zone, render snapshot/map, di chuyển, collision, monster AI, combat + status, hybrid HUD, animation/VFX, loadout Gu sở hữu, business panels, loot idempotent, checkpoint/reconnect, handoff, party/PvP/boss threat, slice/benchmark, multi-instance, rollback drain, CI/observability, fuzz/soak, Alpha content pack và release-gate artifacts đã có; Gate B/C/D chưa PASS vì thiếu browser/network load, Prometheus/OTel, cross-process scaling, DB dupe replay, soak dài, NPC/quest, unique-handler coverage, staging canary và on-call drill.
**Giới hạn còn lại của cả 36 PR:** xem `docs/mmo/remaining-limitations.md` (index duy nhất, cập nhật theo evidence thực tế).
**Chốt từ chủ dự án:** top-down 2.5D; hybrid target-lock + skillshot + AoE; WASD **và** click-to-move; ưu tiên MMO realtime cuối cùng; được phép thay/bỏ code cũ; giữ Go backend.

> **North star:** Toàn bộ PvE/PvP diễn ra **trên bản đồ sống**, nhiều người cùng đi/đánh/cast, không tạo trận theo lượt; input phản hồi tức thì ở client nhưng kết quả vị trí/hit/sát thương/vật phẩm do server Go xác nhận. Phaser chỉ là frontend gameplay; React xử lý UI nghiệp vụ.

---

# 1. Những quyết định không thay đổi (Architecture Decision Records)

| ID | Quyết định | Lý do | Điều không được làm |
|---|---|---|---|
| ADR-01 | **Phaser 4.2.1** trong frontend Next.js; dùng TypeScript, pin version/lockfile | Có scene, camera, input, tilemap, animation/VFX; thay prototype Pixi hiện tại | Không chuyển nguyên xi `WorldCanvas.tsx` hoặc `PvPArenaCanvas.tsx` sang cú pháp Phaser |
| ADR-02 | **Go authoritative zone simulation** (MVP monolith, khi đủ tải mới split) | Một chủ sở hữu duy nhất cho vị trí, HP, AI, cooldown, hit | Không để Phaser Arcade/Matter Physics quyết định hit PvP/PvE cuối cùng |
| ADR-03 | **Top-down 2.5D** kiểu 3/4 nhìn từ trên, 2D collision mặt phẳng XY | Dễ làm MMO hành động, asset, AI, LOS; depth bằng sort theo chân nhân vật | Không dùng hình phối cảnh giả làm nguồn collision; không dựng bản đồ isometric thực sự trong MVP |
| ADR-04 | **WASD + click-to-move**, cùng một đường `MoveIntent` ở server | Cho cả người thích action và người thích chuột | Không dùng DB `world.move_path` cho từng đoạn trong gameplay mới |
| ADR-05 | **Hybrid combat**: TARGET / DIRECTION / GROUND / SELF và nâng lên CONE, CHANNEL, DASH, SUMMON | Hỗ trợ cả Cổ Trùng chỉ mục tiêu, skillshot, AoE | Không tính damage bằng sự kiện client `hit` hoặc animation hoàn thành |
| ADR-06 | **Combat + effects runtime realtime mới**, tái dùng compiler/registries có chọn lọc | Turn-based semantics không quy đổi được cơ học thành ms | Không tự biến mọi `1 turn` thành `2 seconds`; không âm thầm bỏ operation lạ |
| ADR-07 | **UI React / simulation-render Phaser độc lập** | Tránh React rerender trên từng tick/frame | Không đưa entity position mỗi frame vào Zustand/React store |
| ADR-08 | **Chạy đồng thời legacy và MMO với feature flag** | An toàn dữ liệu người chơi, có đường rollback | Không vừa dùng world worker cũ vừa để zone mới sở hữu cùng quái/vị trí |
| ADR-09 | **Postgres giữ data bền vững; Redis không chạy per-tick** | Bảo vệ kinh tế, scale được, chống dupe | Không dùng Redis pub/sub làm cơ chế duy nhất đảm bảo phần thưởng |
| ADR-10 | **MVP desktop-first**; mobile là phase riêng sau core movement/combat | Tránh đội thời gian dev vào joystick/touch | Không coi demo desktop là mobile-ready |

**Phaser 4.2.1:** phát hành chính thức 09/07/2026, `phaser` npm version `4.2.1`. Tham khảo: https://phaser.io/download/release/v4.2.1 và https://www.phaser.io/phaser4. Không mặc định các ví dụ Phaser 3 hoặc plugin ngoài tương thích Phaser 4; thử nghiệm trước mỗi plugin và ghi license/compatibility.

## 1.1 Định nghĩa thế giới 2.5D

- **Gameplay coordinates:** `(x, y)` world units liên tục (quy ước 1 tile logic = 1 unit trong giai đoạn đầu, dùng float; pixel = unit × scale); server và client dùng cùng phép biến đổi, collider dựa trên mặt đất.
- **Visual:** camera top-down 3/4; sprite 4 hoặc 8 hướng; điểm neo (feet anchor) dùng depth sorting `depth = footY`, không dùng tâm sprite để va chạm.
- **Tilesets:** nền, vật thể trang trí, mái che, cầu thang, foreground; collision ở layer riêng; line of sight dựa trên collision/occluder logic.
- **2.5D không có nghĩa 3D physics:** elevation có thể bổ sung sau bằng metadata vùng, không làm platform jumping ở MVP.
- **Bản đồ:** Tiled `.tmj`/JSON xuất vào `public/assets/world/` hoặc `public/game/maps/`; cần build pipeline sinh **server collision/nav artifacts** từ cùng source; hash phiên bản bản đồ trong handshake.

## 1.2 Điều khiển và quyền ưu tiên input

| Input | Ý nghĩa | Chi tiết |
|---|---|---|
| WASD / arrows | Chạy 8 hướng, normalize vector | Ưu tiên tạm thời so với đường click; nhả phím có thể resume tùy setting |
| Click đất | Đặt đích click-to-move | Pathfinding client để preview; server xác nhận path hợp lệ, actual movement cùng MoveIntent |
| Click target | Chọn/khóa mục tiêu | Ưu tiên entity hit-test; không đồng thời đặt điểm đi |
| Shift+click hoặc configurable | Ground-target skill | Khi đang aiming, click xác nhận cast; Esc/chuột phải hủy; không kích hoạt click move |
| Chuột hướng | Skillshot DIRECTION | Aim world position được lượng tử hóa hợp lý trước khi gửi |
| 1–6 | Sáu ô Cổ Trùng | Không chồng WASD như `QWERDF` trong arena cũ |
| Space | Dodge (nếu đã mở kỹ năng né) | Server quyết định i-frame/stamina/charge, client chỉ dự đoán motion/VFX |
| Tab | Next viable target | Ưu tiên target theo khoảng cách/LOS/faction |
| F | Tương tác / loot | Kiểm tra tầm tương tác trên server |
| Enter, / | Chat | Khi focus input UI thì tắt hotkey gameplay |

**Input arbitration:** `UIFocused > Modal > AimingSkill > DirectWASD > ClickPath > Idle`. Chuyển focus/blur/visibility phải gửi `STOP` và xóa pressed keys để tránh nhân vật chạy vô hạn. Click-to-move không được gửi request DB theo từng node; hệ thống path chỉ tạo **ý định dịch chuyển ngắn hạn**, server vẫn kiểm tra mọi tick. Với bản đồ lớn, server có thể thực hiện nav/path validation/waypoint khi cần, không tin path client.

---

# 2. Phân tích source gốc: đường dẫn thật và quyết định migration

> Quan sát từ ZIP; những thay đổi bên dưới **chưa được áp dụng**. Nơi ghi “giữ” không đảm bảo code hoàn toàn phù hợp trước khi có integration test.

| Đường dẫn thật trong ZIP | Hiện trạng/ý nghĩa | Quyết định | PR bắt đầu |
|---|---|---|---|
| `frontend/package.json` | Next 16 / React 19, `pixi.js` 8.19, Zustand; chưa có Phaser | Thêm `phaser@4.2.1`, giữ Pixi tạm; chỉ gỡ Pixi sau cutover | PR-03, PR-31 |
| `frontend/src/app/page.tsx` | Dynamic import `WorldPage` và `PvPArenaView`, view routing | Giữ legacy path; thêm entry `mmo-world` theo flag, không đổi mặc định | PR-04, PR-29 |
| `frontend/src/components/WorldPage.tsx` | UI thế giới, atlas, inventory, modal, hotbar; lazy `WorldCanvas` | Giữ React business UI; gắn `PhaserWorldHost` qua adapter | PR-04, PR-24 |
| `frontend/src/components/WorldCanvas.tsx` | Pixi + world map, lớn/phức tạp; path movement | Legacy-only đến cutover; không refactor lớn nữa | PR-04, PR-31 |
| `frontend/src/components/PvPArena/PvPArenaCanvas.tsx` | Prototype Pixi arena, local correction `SNAP_TILES` / `RECONCILE_RATE` | Dùng làm tham khảo visual; **không** bê correction logic sang MMO | PR-07, PR-31 |
| `frontend/src/components/PvPArena/PvPArenaView.tsx` | Match input + hotkeys `QWERDF`/`1–6` xung đột WASD | Giữ tạm arena cũ; input MMO viết mới & test focus | PR-06, PR-31 |
| `frontend/src/components/PvPArena/MobaControls.tsx`, `MobaSkillSlot.tsx`, `SkillIcon.tsx` | UI prototype skills | Tận dụng style/data hữu ích; tách thành UI hotbar mới | PR-20 |
| `frontend/src/store/useWorldStore.ts` | World cache + async move `world.move_path` | Giữ store legacy; MMO dùng `WorldSession` riêng, chỉ mirror meta UI | PR-04, PR-07 |
| `frontend/src/store/slices/pvpSlice.ts` | `sendPvPInput` gọi `match.input` | Arena legacy, tạo `world.input` protocol riêng | PR-05, PR-08 |
| `frontend/src/lib/websocket.ts` | Reconnect, request/response, subscribe | Giữ cho social/meta; đánh giá WS transport chung vs dedicated MMO gateway | PR-05, PR-08 |
| `backend/cmd/server/main.go` | Bootstrap DB, Redis, registries, workers, ws | Khởi tạo `realtime` sau load registry; feature flag; chỉ start runtime ở zone được bật | PR-02, PR-10 |
| `backend/internal/ws/{message.go,client.go,hub.go}` | WS envelope, hub, subscriptions, worker | Tái dùng auth/session; world messages qua bounded queues + permission check, không broadcast đại trà | PR-05, PR-08 |
| `backend/internal/handlers/world_ws.go` | Chỉ cho `world.move_path`, `world.cells_rect`, `world.objects`, `match.input`, `chat.send` | Thêm router world realtime sau auth; bỏ `world.move_path` từ MMO client | PR-08, PR-29 |
| `backend/internal/handlers/exploration_move.go` | `ExecuteMovePlayerPath` DB transaction, 5 cell/request, encounter | Legacy only; chuyển logic tương tác cell có giá trị thành zone triggers | PR-13, PR-23, PR-31 |
| `backend/internal/handlers/world_monster_encounter.go` | Chạm quái -> turn-based encounter | Không gọi trong zone MMO; quái là entity sống | PR-16, PR-29 |
| `backend/internal/handlers/world_monsters.go` | World worker interval 250 ms | Không chạy trên zone do runtime mới quản lý; thay AI có budget | PR-16, PR-29 |
| `backend/internal/match/{match.go,balance.go,input.go,geometry.go,net.go}` | Arena 20 Hz; input, cast, projectile, `frameLocked()` toàn trận | Tái dùng ý tưởng/toán học và test cases có chọn lọc; tạo zone simulation/AOI riêng | PR-09, PR-14 |
| `backend/internal/match/effects.go` | `extractPayload` chỉ xử lý subset; `StatusTicksPerTurn=40` | Không dùng làm runtime toàn bộ Cổ Trùng; chuyển sang semantic compiler mới | PR-18, PR-19 |
| `backend/internal/gameplay/effects/{contract.go,runtime.go,executor.go,registries.go}` | Definitions typed: `DAMAGE`, `SHIELD`, `HEAL`, `DIRECT_HP`, `STAT_MOD`, `STATUS`, `BARRIER`, `CLEANSE`, `UNIQUE`... | Giữ registry/data DTO; viết realtime translator, semantic audit và applicator | PR-17–19 |
| `backend/internal/simulator/*` | Turn-based combat | Chỉ chạy legacy và làm oracle cho công thức thuần; không đưa vào MMO tick loop | PR-19, PR-31 |
| `backend/data/{Gu.csv,GuMoba.csv,Status.csv}` | Gu quy mô lớn, profile realtime mẫu, durations cũ | Giữ raw data; thêm catalog `gu_realtime` có version và status migration | PR-17, PR-18 |
| `backend/data/{Region.csv,MapCell.csv,Monster.csv,MonsterSpawn.csv}` | Metadata vùng, cell, monster và spawn cũ | Viết script kiểm định/chuyển đổi sang Tiled/map metadata; chưa coi old cells là collision đúng | PR-12, PR-15 |
| `backend/internal/models/*`, `backend/internal/services/*` | Player, inventory, registry, services | Giữ API nghiệp vụ; thêm snapshot/adapter/service commands; không gọi trực tiếp DB từ tick | PR-23, PR-26 |
| `frontend/public/assets/world/*` | Một phần ảnh world/monster/item | Audit rights/size, tạo manifest và atlas; asset thiếu dùng placeholder hợp pháp | PR-11, PR-21 |

**Source caveats bắt buộc:** `backend/go.mod` yêu cầu Go `1.26.3`; pin toolchain/container phù hợp trước CI. Trong ZIP không có bằng chứng đủ để khẳng định latency, FPS, tải game; mọi ngưỡng bên dưới là *acceptance targets*, không phải số đo hiện có. Không suy diễn tất cả dòng `Gu.csv` là kỹ năng đã chuyển được.

---

# 3. Hình thái sản phẩm theo từng cổng phát hành

## Gate A — Headless authoritative demo
- 1 zone/test map, 2 player-bot, 10 monster-bot; fixed step; collision; input seq; snapshot; quan sát được tick metric; **không cần art**.
- Không có turn-based combat trigger từ zone mới.

## Gate B — Playable Phaser slice (đích sớm nhất để cảm nhận)
- 1 map Tiled, 2–5 player online đồng thời; WASD + click-to-move; camera theo player; 3 loại quái; 6 Cổ Trùng: target, projectile, ground AoE, shield, slow, dash.
- Ít nhất 1 nội tại ẩn dạng stack/condition; hit từ server, phản hồi VFX dự đoán ở client.
- Đăng nhập tài khoản cũ; rời vào lại zone không dupe/teleport sai.

## Gate C — Alpha MMO
- 3–5 vùng kết nối bằng portal, 30 player/zone (đích benchmark), 50 monster, 30–50 Gu được xác thực bằng test; Boss party, NPC tương tác, loot/inventory, chat, đột phá/tu luyện React.
- Party, death/respawn, reconnect, zone handoff, PvP một vùng chuyên dụng.

## Gate D — Beta MMO
- Nhiều zone/instance trên các worker; AOI/delta; metrics; kiểm thử 100 người/zone; economy ledger/outbox, recovery sau crash; tối ưu skill/VFX/asset.
- Chỉ mở rộng số người khi p95 tick, network, DB và memory trong ngân sách; nếu không đạt thì scale bằng shard/instance/culling trước khi tăng capacity.

## Ngoài MVP
- Không cố hỗ trợ tất cả Cổ Trùng ngay; không đạt 100 người chỉ bằng cách tăng server tick; không seamless map vô hạn; chưa GvG hàng trăm người cùng một màn; chưa mobile touch chuẩn.

---

# 4. Kiến trúc đích end-to-end

```text
[Next.js 16 + React 19]  — account / tu luyện / bag / chat / quest / admin
      |   ^
      |   +-- UI Bridge (domain state, selected target, cooldown/HP snapshots)
      |
[Phaser 4.2.1 Game Client]
      |- Boot/Preload/World Scene; Map/Camera/Depth/Entities/VFX/UI overlay
      |- Input arbitration -> WASD, click path, targeting, skills
      |- Local prediction -> self; remote interpolation -> others
      |- GameSocket/Protocol -> ack/baseline/events
      |
      +========== WebSocket world protocol v1 ==========+
                                                        |
[Go Gin API + WS Auth] -> [Realtime Gateway] -> [Zone Manager]
                                                  |        |
                                             [Zone 1] [Zone 2] ...
                                             20Hz fixed tick, one owner
                                             Movement / Path / Collision
                                             AI / Cast / Hits / Effects
                                             Loot intent / AOI snapshots
                                                  |
                                    Durable command bus / outbox
                                                  |
                              [Existing Go Domain Services + PostgreSQL]
                                       [Redis session/coordination]
```

**Deployment stage 1:** all Go modules in one process, separate packages. **Stage 2:** zone runtime workers in dedicated processes when profiler + single-owner transfer tests pass. **Không microservice hóa sớm**. WS hub meta/chat có thể giữ song song, nhưng world runtime cần bounded outbound queue / priority separation để chat, UI action hay slow client không nghẽn snapshot.

## 4.1 Runtime Go đề xuất

```text
backend/internal/realtime/                    # NEW namespace, đừng sửa match làm world trực tiếp
  config/              flags, tickHz, limits, config validation
  protocol/            wire DTO v1, JSON fixtures, version negotiation
  gateway/             authenticated join, routing, seq/rate limits, queues
  world/               zone manager, lifecycle, map handoff, ownership
  sim/                 fixed-step, commands, event scheduler, RNG, tracing
  entity/              data-oriented structs, transform, vitals, component refs
  movement/            MOVE intents, speed/stop, collision, nav requests
  nav/                 A*, static map graph/grid, async path planner (bounded)
  spatial/             spatial hash, query, AOI, visibility policy
  ai/                  monster FSM, threat, leash, cooldown, AI budget
  combat/              cast pipeline, target checks, hit solver, projectiles
  effects/             compiled Gu runtime, trigger dispatch, status scheduling
  loot/                reward intents + grants, idempotency keys
  persistence/         session recovery, checkpoint, outbox/ledger adapters
  transfer/            zone-to-zone transfer protocol
  telemetry/           metrics, tracing, structured game events
```

## 4.2 Client Phaser đề xuất

```text
frontend/src/game/
  bootstrap/
    createPhaserGame.ts       # dynamic import, create/destroy đúng vòng đời
    config.ts                 # phaser scale/renderer/scene configuration
  scenes/
    BootScene.ts
    PreloadScene.ts
    WorldScene.ts
  core/
    GameSession.ts            # state machine CONNECTING, JOINED, TRANSFERRING...
    GameEvents.ts             # typed bus to React HUD, throttled
    WorldClock.ts             # tick/time offset and render timeline
  network/
    WorldSocket.ts            # app protocol v1
    Protocol.ts               # decode/validation + version checks
    SnapshotBuffer.ts
    Prediction.ts
    Reconciliation.ts
    RemoteInterpolator.ts
  input/
    InputRouter.ts            # UI focus / skill aiming / movement precedence
    KeyboardMovement.ts
    ClickToMove.ts
    TargetSelector.ts
    SkillAimController.ts
  world/
    MapLoader.ts              # Tiled JSON + collision manifest
    WorldUnits.ts             # unit <-> px, axes
    CameraController.ts
    DepthSort.ts
    EntityRegistry.ts
    VisibilityCulling.ts
  entities/
    PlayerView.ts
    MonsterView.ts
    NPCView.ts
    ProjectileView.ts
    EntityAnimator.ts
  combat/
    CastTelegraph.ts
    VFXPool.ts
    FloatingCombatText.ts
    CombatEventPresenter.ts
  assets/
    AssetManifest.ts
    AnimationLibrary.ts
  debug/
    RealtimeDebugHUD.ts
  bridge/
    ReactGameBridge.ts
frontend/src/components/game/
  PhaserWorldHost.tsx
  MMOHud.tsx
  GuHotbar.tsx
  TargetPanel.tsx
  CastBar.tsx
  MMOErrorBoundary.tsx
frontend/public/game/maps/
frontend/public/game/atlases/
frontend/public/game/fx/
```

**Không import Phaser tại top-level ở SSR component.** Dùng `dynamic(..., { ssr:false })` + `useEffect` để mount client; `Game.destroy(true)` trong cleanup; event listeners/subscriptions/raf timers đã đăng ký phải cleanup. Bảo đảm Next Fast Refresh không sinh hai canvas/socket/zone session; không tạo mới Phaser khi React HUD đổi state.

## 4.3 Trách nhiệm rõ ràng

| Thành phần | Được làm | Không được làm |
|---|---|---|
| Phaser | Nhận input, dự đoán local visual, camera, animation/VFX, hiển thị UI world | Quyết định sát thương, phần thưởng, auth, máy trạng thái AI authoritative |
| React | Modal, bag, market, tu luyện, hotbar setup, UI social, log | Rerender component lớn theo mỗi tick; tự tick HP/projectile |
| Go Zone | Vị trí, speed/collision, NPC/AI, cast, LoS/hit, DoT, cooldown, death, combat flags | Gọi transaction DB cho mỗi movement tick; tin `damage`/`x,y` do client gửi |
| Go Domain Services | Owned Gu, equipment, inventory, economy, tu luyện, audit log | Trực tiếp đổi HP/tọa độ entity đang thuộc zone ngoài cơ chế command |
| Postgres | Truth durable, reward ledger, session/transfer history | Chứa realtime frame state cần update 20 lần/giây |
| Redis | Session lookup/presence, cache, worker coordination | Sole-source of truth cho loot/reward, broadcast liên tục mọi actor |

---

# 5. Server realtime specification

## 5.1 Fixed-tick Zone Loop (MVP 20 Hz)

- 50 ms/step; internal time = `tickIndex`, không phụ thuộc frame clock máy khách. Input cuối/đầu tick được ghi trace có thứ tự. Một goroutine là sole owner của zone mutable state, gateway chỉ enqueue; dùng bounded queue.
- Trình tự xác định: `accept inputs -> validate seq/session -> advance movement + collision -> monster AI -> progress windup/channel/cooldowns -> spawn/update projectiles -> hit resolution -> effect event queue -> death/loot intents -> AOI -> snapshot -> persistence events`.
- **Event ordering:** priority + source + monotonic event sequence, có chain depth/number budget, nguồn random server có seed. Snapshot là output, không phải input của sim.
- **Cadence độc lập:** tick 20Hz, snapshot 10–20Hz, AI think 2–5Hz theo độ gần và ưu tiên; visual 60FPS. Không bắt AI pathfind toàn bản đồ mỗi tick.
- **Overload:** log tick p50/p95/p99, slow-zone warnings; giới hạn catch-up; không tự động tăng skip logic mà không có policy; ưu tiên instance/giảm AI/PvE density.
- Go race test, deterministic sim tests không cần sleep; injection `FakeClock`/step counter.

### State entity

```text
EntityID (runtime unique), CharacterID (durable UUID), ZoneInstanceID, revision
Transform{x,y,facing,velocity,radius,feetY}
Vitals{hp,maxHp,mana,maxMana,shield}
Stats{movementSpeed,attack,defense,crit,resist,...}
Combatant{faction,target,castState,gcdUntil,cooldownByGu,controlFlags}
Effects{statusStacks,expiryTick,nextPeriodicTick,triggerCounters,hiddenFlags}
GuLoadout{guInstanceId -> skillDefinitionVersion}
AIState{home,threat,aggroRange,leash,thinkTick}
Appearance{atlas,animationDirection,skin}
```

- Mỗi entity mang `revision` hoặc spawn nonce để tránh event/damage áp vào entity ID vừa respawn.
- Tất cả timer gameplay lưu dạng `tick` hoặc duration có nguồn giờ thống nhất; content có thể author theo milliseconds rồi compile tick với quy tắc làm tròn đã kiểm thử.

## 5.2 Movement authoritative: hai kiểu cùng chung đường

**WASD**: gửi normalized `direction` + seq + pressed flags cần thiết. Server kiểm tra speed/stun/root, collision, acceleration (nếu dùng) và cập nhật vị trí.

**Click-to-move**:
1. Client chuyển pixel clicked thành world coordinate theo camera/zoom; hit-test trước để không vô tình di chuyển khi bấm target/UI.
2. Client dùng nav grid để vẽ preview path; gửi `world.move_target` (`targetX,Y`, `requestId`, mapVersion), không gửi hàng trăm waypoint tùy ý.
3. Server kiểm tra map version/destination/traversability (hoặc cấp route), tạo `NavigationIntent`/waypoints có giới hạn; có thể pathfind async trong budget.
4. Tick loop biến waypoint tiếp theo thành **cùng MoveIntent** như WASD; không “teleport đến click”; vật cản động gây repath hoặc stop; server có quyền reject `PATH_BLOCKED`, `OUT_OF_BOUNDS`.
5. Người chơi bấm WASD thì **hủy click path ngay**; nếu bật option “resume” phải rõ ràng. Skill có root/channel/dash điều chỉnh nav intent theo movement policy.

**Movement anti-cheat:** clamp speed/acceleration, finite coordinates, teleport/zone boundary, replay input seq, max commands/second, arena collision and world collision parity tests. Server không dùng Phaser Physics để quyết định chuyển động thật.

**Prediction**: local client lưu input history `{seq,dt,axes}`; khi snapshot nhận `ackSeq`, restore authoritative state rồi replay phần chưa ack; correction hiển thị mượt nếu sai số nhỏ, hard snap khi quá ngưỡng hoặc teleport. Remote actors lấy timestamp/tick-based interpolation; extrapolate ngắn và freeze an toàn. Pathfinding preview cho click-to-move phải giống logic Go nhưng **không cấp đặc quyền vị trí**.

## 5.3 Combat Hybrid server-first

| Target mode | Kiểm tra authoritative | UX Phaser |
|---|---|---|
| `SELF` | Có Gu, đủ mana, status cho phép, cooldown | Ấn 1 lần cast; VFX quanh nhân vật |
| `TARGET` | Target ID đúng revision, phe/LOS/range còn hợp lệ tại resolve | Highlight target, range ring, castbar |
| `DIRECTION` | Aim normalized, clamp ray, vật cản/projectile collision | Aim cone/line preview; projectile đi qua world |
| `GROUND` | Điểm ground trong range, valid terrain/LOS; hit theo shape và thời điểm | Reticle AoE, telegraph cho cả đồng minh/địch |
| `CONE`, `CHANNEL`, `DASH`, `SUMMON` | Rules riêng; move policy, interrupt, collision, cap summons | Cone preview, channel VFX, dash trail, summon sprite |

- `cast_intent` chỉ chứa slot/gu instance ID, target/aim/location/requestId; tuyệt đối không truyền `damage`/`cooldown` do client tính.
- Cast state machine: `REQUESTED -> WINDUP -> EXECUTE -> RECOVERY -> READY`; có interrupt/cancel/refund policy cụ thể; một slot hay skill không thể cast liên tục bằng spam req.
- Phân tách event `CAST_ACCEPTED`, `CAST_STARTED`, `PROJECTILE_SPAWNED`, `HIT_CONFIRMED`, `DAMAGE_APPLIED`, `STATUS_APPLIED`, `CAST_REJECTED`, `ENTITY_DIED`; UI/VFX có thể chơi dự đoán hiệu ứng vô hại, rollback/cancel khi reject.
- Trúng hit: spatial index + target policy + hitbox/ray/shape + LoS, có limit maxTargets & duplicate-hit per projectile. Có chế độ PvP-specific CC diminishing returns / damage scaling khi cân bằng.
- Projectile server sim 20Hz có thể substep/swept collision cho đạn nhanh, tránh xuyên mục tiêu (tunneling). Không nhất thiết broadcast tọa độ projectile mọi tick nếu client có spawn/trajectory/lifetime đã xác nhận và correction.
- Enemy AI: `IDLE -> PATROL -> AGGRO -> CHASE -> TELEGRAPH/ATTACK -> RECOVER -> LEASH -> RESPAWN`, có threat và cooldown; shared AI budget.

## 5.4 Event/Effect Cổ Trùng realtime

**Không** dùng `match.extractPayload()` cho tất cả Gu: phiên bản này chỉ trích `DAMAGE`, `SHIELD`, `STATUS`, bỏ qua nhiều operation. Block v2 có `HEAL`, `DIRECT_HP`, `STAT_MOD`, `BARRIER`, `CLEANSE`, `UNIQUE` và trigger như `ON_TURN_START`, `ON_BATTLE_START`.

**Pipeline mới:** `Raw Gu Definition -> Validate -> Semantic Classification -> Realtime Compile -> Immutable Executable Skill -> Event Runtime -> Visual Event Mapping`. Mỗi Gu có `ACTIVE` tùy quy tắc trang bị + `PASSIVES[]` + `HIDDEN_PASSIVES[]` + optional `RESONANCE[]` + special native handlers có version.

**Bắt buộc có report:** `GU_ID/RANK`, effect block, trigger, target selector, opcode, trạng thái `{SUPPORTED, NEEDS_REAUTHOR, UNSUPPORTED}`, severity, đề xuất semantic mapping, unit test link. Skill `UNSUPPORTED` **không** được âm thầm cast như skill mất hiệu ứng; chọn ẩn/khóa trong mode MMO và giải thích cho người chơi/admin.

Ví dụ đề xuất thời gian **cần cân bằng**, không đổi 1 turn mặc định:

| Turn-based logic | Realtime semantic mới | Notes |
|---|---|---|
| `ON_TURN_START` | `ON_INTERVAL`/`ON_ENTER_COMBAT` tùy content | Cần author rõ mỗi skill; không mapping hàng loạt vô điều kiện |
| Đốt 3 lượt | `DOT` `durationMs`, `periodMs`, stacks policy | Có giới hạn refresh/max stack |
| Stun 1 lượt | `CC_STUN` `durationMs`, DR category | PvP CC cap cần tính |
| HP hồi mỗi lượt | `HOT` theo time events | Không phụ thuộc FPS/network |
| `ON_BATTLE_START` | `ON_ENTER_COMBAT` hoặc passive on equip | Không kích liên tục khi nhảy ranh giới combat |
| `ON_DEATH` | `ON_PRE_DEATH`/`ON_DEATH` có priority | Bất tử/life-save xử lý trước tử vong |
| `UNIQUE` | Native Go handlers / audited WASM? **Go handler trước** | Deterministic, versioned, tests; không chạy script người dùng trực tiếp |
| “3 crit liên tiếp khi HP thấp” | Event counter / condition + hidden passive, ICD | Server lưu counter & discovery state |

**Event safety:** counter origin/chain depth/event budget, anti-self-proc, per-source internalCooldown, stack caps, cooldown groups, resource floors; quyết định proc trước/sau shield/crit phải ổn định. Nội tại ẩn do server chạy kể cả chưa khám phá nếu luật yêu cầu; trạng thái khám phá trong DB không làm lộ secret trong snapshot người khác.

### Ví dụ contract khai báo Gu realtime

```yaml
id: GU_WATER_ARROW
rank: 1
schema_version: 1
active:
  mode: DIRECTION
  windup_ms: 250
  recovery_ms: 250
  cooldown_ms: 6000
  range_units: 16
  movement_policy: SLOW
  projectile: {speed_units_per_sec: 14, radius_units: 0.25, pierce: 3}
  effects:
    - {type: DAMAGE, formula_id: WATER_ARROW_DAMAGE}
passives: []
hidden_passives: []
visuals:
  icon: gu/water_arrow
  cast_vfx: cast/water
  projectile_vfx: projectiles/water_arrow
  impact_vfx: impact/water
```

**Note:** YAML chỉ là dữ liệu mục tiêu để thống nhất authoring; định dạng runtime có thể là JSON/DB. Không thay `Gu.csv` hàng loạt khi chưa có migration report và golden tests.

## 5.5 Zone / MMO / persistence

- **ZoneKey:** `{mapId, instanceId, rulesetVersion}`. `ZoneManager` quản lý create/start/stop/join/leave, `SessionLease` đảm bảo **một nhân vật chỉ một zone owner**. Multi-connection chỉ một active gameplay controller; tab khác có thể xem UI (chính sách do product quyết định).
- **AOI:** spatial grid, subscribe radius + hysteresis, visibility masks. Chỉ gửi nearby actors, danger telegraphs, party critical info và relevant events; có `spawn/despawn` trong snapshot. Delta state + baseline validation; resync khi gap.
- **Zone transfer atomic**: `freeze input -> serialize transfer snapshot -> reserve target owner -> commit handoff -> release source -> resume`. Timeout/fail rollback vào source hoặc safe spawn; **không có hai owner**.
- **Reward path**: Zone tạo `LootGrantIntent` với `idempotencyKey=zoneId:entityDeathId:recipientId:rewardVersion`; domain service transaction DB ghi ledger + item/grant + outbox. Retry an toàn và đảm bảo không nhân đôi. Không grant vật phẩm từ client event, không async bỏ quên khi crash.
- **Checkpoint**: player presence/last safe pos/HP snapshot cadence thích hợp, combat lock/transfer transitions, log command/intent quan trọng; tách operational checkpoint và durable financial transaction.
- **Recovery**: crash server không nhân đôi corpse/loot; session có revision/lease expiry; instance recovery hoặc safe fallback spawn; retry idempotent; disconnect gây đứng im hoặc safe auto-logout theo chính sách vùng (PvP/combat log cần công bằng).

---

# 6. Network protocol v1 & QoS

**Tận dụng** WS auth/existing hub nếu đạt bandwidth/queue budget, nhưng bắt buộc world route riêng; nếu nghẽn bởi generic command requests, mở endpoint WS gameplay riêng dùng cùng auth. Không gửi `world.move_path` từ Phaser game.

### Handshake

```json
{"type":"world.join","v":1,"requestId":"join-123","mapId":"forest_01","clientBuild":"mmo-alpha.1"}
```

Server `world.welcome`: protocol/content version, `zoneId`, `playerEntityId`, `tickHz`, `snapshotHz`, `mapHash`, spawn state, clock sync, limits; deny with structured reason/version mismatch if needed.

### Movement input / cast / click target

```json
{"type":"world.input","v":1,"seq":812,"clientTick":450,"move":{"dx":1,"dy":0},"casts":[{"requestId":"c812","slot":2,"mode":"DIRECTION","aim":{"x":12.3,"y":31.2}}]}
```

```json
{"type":"world.move_target","v":1,"requestId":"path-44","mapHash":"forest-2026-01","target":{"x":33.0,"y":18.5}}
```

### Snapshot and events

```json
{"type":"world.snapshot","v":1,"zoneId":"forest_01#1","tick":915,"ackSeq":812,"baselineId":11,"entities":[{"id":"p:7","revision":3,"x":12.2,"y":31.1,"hp":870}],"removed":[]}
```

```json
{"type":"world.events","v":1,"zoneId":"forest_01#1","fromEventId":3001,"events":[{"id":3001,"tick":915,"type":"DAMAGE_APPLIED","source":"p:7","target":"m:4","guId":"GU_WATER_ARROW","value":170}]}
```

**Protocol rules:**
- `v`, `zoneId`, `session`, `seq`, `requestId`, `tick` semantics rõ ràng; reject invalid floats/size/rate/target range; no client damage.
- `seq` per movement/control stream; `requestId` cho action idempotency. Ack self movement; `baselineId` kiểm soát delta; snapshot thiếu baseline phải full-resync. Thứ tự input/cast xử lý xác định.
- Snapshot có thể **coalesce/drop** dưới áp lực; critical gameplay results/reward events phải có replay/reconcile. WS/TCP đảm bảo truyền trên kết nối sống nhưng reconnect vẫn cần application-level sync.
- `heartbeat + ping estimate + clock offset` phục vụ interpolation buffer theo server tick, không rely solely arrival times.
- Khởi đầu JSON schema, đo bytes/sec/client và CPU; chỉ chuyển binary khi profiler cho thấy cần thiết. Thiết kế DTO stable để hỗ trợ cả hai.
- Không push toàn thế giới; không publish broadcast `world:pos` cho MMO scene mới. AOI + bounded write queues, slow-client disconnect/resync, rate limits.

### Ngân sách hiệu năng định hướng (cần benchmark)

| Metric | Gate B (demo) | Gate C (Alpha) | Gate D (Beta) |
|---|---:|---:|---:|
| Render desktop reference (medium laptop, 1080p) | ≥ 55 FPS p95 frame-time tương đương (đo FPS/frametime) | mục tiêu 60FPS, p95 frame ≤ 25ms | profiling theo mật độ/VFX |
| Zone tick (20Hz, budget 50ms) | p95 ≤ 15ms | p95 ≤ 25ms | p95 ≤ 35ms, p99 < 50ms |
| Concurrent players/zone (stress) | 5–10 | 30 | 100 **nếu** tick/net/DB đạt |
| Reconnect to resumed play | < 5s thông thường trên LAN | < 10s có phục hồi | đo p95, crash recovery có policy |
| Snapshot outbound per player | Đo baseline, cảnh báo > 30 KB/s | mục tiêu kiểm soát < 25 KB/s trung vị | tùy AOI; công bố số thực đo |
| Client RTT testing | 50/100/150/200ms, jitter/loss | regression stable | load + bad network |
| Economy duplicate grants | 0 trong test retry/crash | 0 | 0 |

*Số liệu trên là target đề xuất để phát hiện lỗi sớm, không là SLA/giới hạn cứng cho mọi thiết bị. Đo p50/p95/p99 frame time chứ không đánh giá bằng FPS trung bình đơn lẻ.*

---

# 7. Phaser 4 game client: implementation specification

## 7.1 Lifecycle và Next.js

- Host một DOM element duy nhất; `Game` khởi tạo khi mount sau login, giải phóng khi unmount/logout/hot reload; không giữ singleton bị leak. Khi `WorldPage` chuyển tab, chọn rõ pause/hide hay unmount/leave zone; tránh logout giả.
- `BootScene`: load manifest tối thiểu; `PreloadScene`: progress/cancel; `WorldScene`: map/entities/network bind. Một `GameSession` tồn tại xuyên scene reload do zone transfer, nhưng `Scene` luôn cleanup listeners.
- React bridge: `GameEvents` typed, throttled (HP/cooldown khoảng 5–10Hz hay event-driven), UI actions phát command qua session, không expose mutable `Phaser.GameObjects` cho React.
- Nếu WebGL không hỗ trợ, show fallback có thông báo; decide Canvas fallback sau test Phaser 4 renderer + effects, không hứa support mặc định.

## 7.2 Camera / world / tilemap

- Camera follow local feet point, lerp có giới hạn, zoom/scale cho top-down 2.5D; viewport clamp với vùng map, resize và DPR.
- Static map layers/chunks; dynamic props/NPC/entity separate. Atlas packed, texture filtering đã kiểm tra, animation hướng tương ứng vector; z-order theo `feetY`, mái che/foreground xử lý explicit.
- Tiled object layers: `collision`, `occlusion`, `monster_spawn`, `npc_spawn`, `portal`, `safe_zone`, `pvp_zone`, `resource`, `boss_arena`; schema lint và xuất collision/nav hash cho backend.
- Click pixel-to-world: `camera.getWorldPoint(pointer.x,pointer.y)` hoặc API đã xác minh 4.2.1, quy về đơn vị logic; mọi skill telegraph từ authoritative params.

## 7.3 Combat visual + asset format

- `VisualSkillManifest` map `gu_id/rank` -> cast / projectile / impact / status VFX. Missing VFX = graceful generic effect, **không** biến thành missing damage.
- Atlas/spritesheet nhất quán: nhân vật (idle/walk/run/cast/hit/death 4 hoặc 8 hướng), quái (tương tự), Cổ Trùng icon 128×128 source (scale tùy UI), projectile/VFX frames và shader/particles dùng pooling.
- Combat hit feedback: âm thanh, hit flash, floating damage, HP bar, telegraph, camera shake có giới hạn; accessibility option giảm rung/chớp.
- Asset provenance: giấy phép, độ phân giải, tên sprite, animation fps, atlas hash, content version. Không copy trực tiếp art, logo, map hoặc UI của game khác.

## 7.4 Test frontend cần viết

- Pure tests: vector normalize, key priority, click-to-move behavior, target selection, seq buffer ack/replay, interpolation, projection, aim targeting & map hash handling.
- Browser E2E: login -> join zone -> move WASD/click -> cast -> target/loot -> reload -> rejoin. E2E test canvas qua instrumentation/debug overlay và structured logs, không screenshot-only.
- Hot reload/React StrictMode: mount/unmount lặp 20 lần, active Phaser instance/canvas/socket = 1/0 tương ứng; không double listener.
- Performance: 100 static sprites + 50 monsters + 20 VFX mẫu trên máy tham chiếu; profile draw calls, object churn, texture VRAM.

---

# 8. Migration data & legacy cutover (bắt buộc)

1. **Backup** Postgres và media trước mọi thay đổi. Test restore vào môi trường staging; migrations add-only/expand-contract, reversible khi có thể.
2. **Account, CharacterID, InventoryID, GuInstanceID giữ nguyên**. Tạo `player_world_state_v2` hoặc bảng tương đương cho `{zoneInstanceId, xFloat, yFloat, mapVersion, lastCheckpoint}`; không ép float trở về old grid X/Y rồi mất tọa độ.
3. **Old map conversion:** `Region/MapCell/MonsterSpawn` chuyển thành nội dung Tiled metadata qua công cụ; dữ liệu cell event cũ được phân loại: NPC/resource/trap/portal/quest. Không chạy `resolveCellArrivalWithInstance` trong movement tick; tương tác event-driven khi overlap/press F.
4. **Service boundaries:** progress/tu luyện/inventory/shop/sect xử lý bằng Go domain services; zone đọc snapshot cache/subscribe change revision; mọi thay đổi loadout đang trong combat tuân thủ state policy.
5. **Effect migration:** tạo report coverage + reauthoring catalog; giữ `Gu.csv` làm nguồn định danh/content ban đầu, thêm `RealtimeGuProfiles` theo version. Build artifacts kiểm tra duplicate IDs/rank, missing VFX, unsupported effects và timing constraints.
6. **Dual mode với exclusivity:** user trong `legacy_world` thì legacy world worker có thể quản lý; user vào `mmo_zone` thì legacy worker **không** mutate world state/HP cùng entity. Cần migration flag per user/zone; không chỉ `if` ở frontend.
7. **Release gates:** dev sandbox -> QA internal -> opt-in alpha -> migration pilot -> default MMO -> disable legacy write paths -> remove Pixi dependency/simulator adapters -> cleanup schema sau nhiều kỳ deploy.
8. **Rollback:** feature flag đổi UI entry và admission (chặn join MMO mới), drain zone về safe positions, reconcile pending rewards/ledger, restore legacy last-safe spawn; không rollback schema xóa data trong tình huống nóng.

---

# 9. Điều kiện nghiệm thu (Definition of Done) dùng cho mọi PR

- Có unit/integration/regression tests và không làm đỏ test legacy ngoài những breaking changes được phê duyệt; tsc/Go compile với toolchain pin; formatting/lint sạch.
- Server authoritative: không tin state/damage client; tất cả API/WS đều qua authz và rate limit; lỗi mạng và rollback được kiểm tra.
- Không làm mất/dupe inventory, currency, Gu hoặc chuyển vùng; mọi reward có idempotency key.
- Có logs/metrics tối thiểu ở luồng critical; các benchmark đạt gate tương ứng; nếu chưa đo ghi `NOT MEASURED`, không báo đạt.
- Code mới có tests ở pure logic; config flags documented; feature default OFF trên production đến khi QA pass.
- PR tối đa tập trung một vertical concern; không xóa `WorldCanvas`/arena cũ trước khi có parity đủ dùng và kế hoạch cutover.

---

# 10. Backlog theo PR — triển khai theo thứ tự phụ thuộc

Các đường dẫn `backend/internal/realtime/*` và `frontend/src/game/*` ban đầu là dự kiến; một phần đã được tạo trên branch, phần còn lại vẫn TODO. PR được đánh số theo thứ tự phụ thuộc; cập nhật trạng thái bên dưới dựa trên code hiện có, không đồng nghĩa với việc gate phát hành đã đạt.


## PR-01 — Inventory, baseline & backup  
**Milestone:** M0 · **Phụ thuộc:** none · **Trạng thái:** IN PROGRESS (code-flow inventory and partial local baseline recorded; pinned-toolchain, full build, backup/restore and browser evidence remain open)

**Đường dẫn:** `backend/go.mod` (existing), `frontend/package.json` (existing), `backend/cmd/server/main.go` (existing), `docs/mmo/audit.md` (new), `docs/mmo/legacy-flow.md` (new)

**Các task:**
- [ ] Ghi checksum ZIP, dependency lock, branch `mmo/phaser4`; tạo danh mục entrypoints/routing/player/inventory/combat/WS.
- [ ] Pin toolchain Go 1.26.3 từ `go.mod`; chạy `go test ./...` với môi trường phù hợp, `npm run typecheck`, `npm test`, `npm run build`; ghi lỗi baseline, không sửa đại trà.
- [ ] Backup DB media + thực hiện restore test ở staging; ghi version schema, kiểm tra file env không commit secrets.
- [ ] Đo baseline latency/FPS nếu khởi động được; ghi NOT MEASURED nếu không đủ DB/CI.

**Tiêu chí merge/nghiệm thu:** Report file-by-file + test outputs có timestamp; backup restore xác nhận; không thay đổi gameplay.

## PR-02 — Flags, ownership & safety boundaries  
**Milestone:** M0 · **Phụ thuộc:** 01 · **Trạng thái:** IN PROGRESS

**Đường dẫn:** `backend/cmd/server/main.go` (existing), `backend/internal/realtime/config/config.go` (new), `frontend/src/lib/featureFlags.ts` (new), `docs/mmo/rollout.md` (new)

**Các task:**
- [x] Định nghĩa flag `REALTIME_WORLD_ENABLED`, `MMO_ZONE_IDS`, `MMO_ALLOWLIST`, `PHASER_WORLD_CLIENT_ENABLED`, `MMO_MAP_VERSION`, `MMO_CONTENT_VERSION`, giới hạn 100 người/zone và tối đa 64 zone/process.
- [x] Legacy monster worker chỉ xử lý `world_map` với `InstanceID` rỗng; thêm regression test để bỏ qua instance realtime.
- [x] Thêm primitive lease đơn chủ ở process-level (claim/replace/release, token chống stale disconnect) và guard legacy movement, combat, loot, teleport cùng lệnh `world.move_path`/`match.input` khi nhân vật đang được realtime sở hữu.
- [x] Nối lease process-local vào authenticated join/reconnect/disconnect và instance zone riêng; guard worker quái legacy cùng các world movement/combat/loot/teleport paths đã được audit.
- [ ] Tạo worldMode/lease bền vững hoặc shared; fence transaction/encounter legacy đã chạy, quarantine legacy position index và guard các mutation path còn lại. Admission vẫn chỉ dùng staging allowlist.
- [ ] Chốt logout/disconnect/cross-tab semantics và bootstrap fail-safe. Config parser có test; middleware chỉ chặn khi runtime ownership checker có lease.
- [x] Viết runbook bật/tắt flag, đổi mode không xóa old player rows.

**Tiêu chí merge/nghiệm thu:** Production default OFF; test chuyển flags không cho double-owner hay spawn vào 2 modes.

## PR-03 — Pin Phaser 4.2.1 + smoke render  
**Milestone:** M0 · **Phụ thuộc:** 01 · **Trạng thái:** DONE (Phaser 4.2.1 pinned in `package.json`/lockfile; boot/preload/world preview scenes mount in the gated MMO entry, no Phaser physics or authoritative state)

**Đường dẫn:** `frontend/package.json` (existing), `frontend/package-lock.json`/lockfile thực tế (existing or new), `frontend/src/game/bootstrap/{createPhaserGame.ts,config.ts}` (new), `frontend/src/game/scenes/{BootScene,PreloadScene,WorldScene}.ts` (new)

**Các task:**
- [x] Cài `phaser@4.2.1` phiên bản cố định; giữ `pixi.js` cho legacy đến PR-31; kiểm tra build/SSR.
- [x] Tạo scene boot/preload/world, nền placeholder và một sprite; scale/responsive/camera test.
- [x] Chọn WebGL renderer theo capability; confirm Canvas fallback behavior, không dùng plugin Phaser 3 chưa thử. (`Phaser.AUTO` = WebGL → Canvas.)
- [x] Tạo TS test hoặc smoke browser đầu tiên để detect duplicate canvas/graphics init. (`gameRegistry` tests cover attach/detach idempotency; browser StrictMode pass chưa chạy.)

**Tiêu chí merge/nghiệm thu:** Next build/typecheck pass; một canvas Phaser mount/unmount ổn, legacy Pixi vẫn chạy.

## PR-04 — React Phaser host + parallel route  
**Milestone:** M0 · **Phụ thuộc:** 02,03 · **Trạng thái:** IN PROGRESS (Phaser host, error boundary, gated `mmo-world` entry, live `WorldSocket` join and WASD/click render landed; browser StrictMode/E2E lifecycle proof pending)

**Đường dẫn:** `frontend/src/components/game/{PhaserWorldHost,MMOErrorBoundary}.tsx` (new), `frontend/src/components/WorldPage.tsx` (existing), `frontend/src/app/page.tsx` (existing)

**Các task:**
- [x] Tạo host `use client`, lazy import Phaser, init/destroy canvas và resize; không tạo game trên SSR.
- [x] Thêm entry route/view `mmo-world` gated; giữ `world-map` và `pvp-arena` cũ nguyên mặc định.
- [x] Tách UI bridge typed và lifecycle `CONNECTING -> JOINED -> LEAVING` mock.
- [ ] Test React StrictMode, đổi tab 20 lần, route back/forward không leak listener/canvas.

**Tiêu chí merge/nghiệm thu:** E2E legacy và Phaser side-by-side; khi flag OFF không load bundle Phaser game scene vào page cũ.

## PR-05 — Protocol v1 shared fixtures & schema  
**Milestone:** M0 · **Phụ thuộc:** 01 · **Trạng thái:** IN PROGRESS (v1 client decoder, snapshot buffer and authenticated socket consumer added; fixtures and remaining message families pending)

**Đường dẫn:** `backend/internal/realtime/protocol/*` (new), `frontend/src/game/network/{Protocol,WorldSocket}.ts` (new), `docs/mmo/protocol-v1.md` (new)

**Các task:**
- [x] Định nghĩa server fields cho `world.join/welcome/input/reject/snapshot`; `world.input` validates positive seq and normalized finite vector. [ ] Chốt các message families còn lại và shared client/server fixtures.
- [ ] Server + client shared JSON fixtures; client tests now cover welcome/join, delta application, input sequencing and cancellation. Golden coverage for units, ticks, `ackSeq`, eventId, baselineId, `entityRevision`, content/map version remains.
- [ ] Reject malformed/NaN/Inf/huge payload, invalid slot, world version mismatch; ghi max sizes/rates.
- [ ] Phiên bản `v=1` phải hoạt động với feature flag, không phá `match.input` hay `world.move_path`.

**Tiêu chí merge/nghiệm thu:** Protocol fixtures validate cả hai phía; test negative cho malformed/duplicate/replay requests.

## PR-06 — Unified input arbitration & aim UI  
**Milestone:** M1 · **Phụ thuộc:** 03,04,05 · **Trạng thái:** IN PROGRESS (WASD/click arbitration, blur/visibility reset and UI-focus blocking landed; aim/target/skill hotkeys and browser input E2E pending)

**Đường dẫn:** `frontend/src/game/input/{InputRouter,KeyboardMovement,ClickToMove,TargetSelector,SkillAimController}.ts` (new), `frontend/src/game/scenes/WorldScene.ts` (new)

**Các task:**
- [x] Xây precedence `UIFocused > Aiming > WASD > ClickPath`; riêng chat/modal phải chặn movement và skills. (Blockers + `isTextField`; chưa có chat/modal runtime.)
- [x] WASD chuẩn hóa vector, on blur/visibility gửi STOP; click đất/target/aim là ba hit-test khác nhau. (Click đất có; target/aim chưa.)
- [ ] Test keys 1–6, Space, F, Tab, Esc, right-click cancel; loại mọi xung đột `QWERDF`/WASD.
- [x] Input click path có cancel/replan, event chuyên biệt `MoveTarget` không gọi `useWorldStore.movePath`. (Escape/WASD cancel; chưa có replan UI.)

**Tiêu chí merge/nghiệm thu:** Unit test 30+ tình huống input; demo chơi offline không phải lệnh nào cũng fire 2 lần.

## PR-07 — Client timing, snapshot & prediction core  
**Milestone:** M1 · **Phụ thuộc:** 05,06 · **Trạng thái:** IN PROGRESS (baseline/delta storage, soft/hard reconciliation, tick-based remote interpolation and a server-tick clock are wired; synthetic latency/jitter suite pending)

**Đường dẫn:** `frontend/src/game/network/{SnapshotBuffer,Prediction,Reconciliation,RemoteInterpolator}.ts` (new), `frontend/src/game/core/WorldClock.ts` (new)

**Các task:**
- [x] Tạo state buffer keyed by entity ID/revision and apply monotonic ticks/ackSeq with baseline-relative deltas; prediction replay and interpolation by clock offset remain.
- [x] Tách authoritative transform và render transform, không mutate server state trong Phaser. (SelfPredictor/interpolator; scene chỉ đọc.)
- [x] Áp dụng correction có threshold theo **world units** và teleport handling; không copy `SNAP_TILES` / `RECONCILE_RATE` cũ. (2-unit snap threshold, 25% smoothing.)
- [ ] Test deterministic jitter, out-of-order, duplicate snapshot, packet burst, event gap và resync.

**Tiêu chí merge/nghiệm thu:** Synthetic test 100ms RTT+jitter: self input phản hồi local ngay, remote mượt, không ping-pong/replay vô hạn.

## PR-08 — Gateway world routing / session / backpressure  
**Milestone:** M1 · **Phụ thuộc:** 02,05 · **Trạng thái:** IN PROGRESS (authenticated join/leave/input and AOI baseline/delta stream wired; reliable events/QoS pending)

**Đường dẫn:** `backend/internal/ws/{client.go,message.go}` (existing), `backend/internal/handlers/world_ws.go` (existing), `backend/internal/realtime/gateway/*` (new), `frontend/src/game/network/{WorldSocket,SnapshotBuffer}.ts` (new)

**Các task:**
- [x] Auth bind character->session/zone; protocol-v1 `world.join` binds player ID from AuthRequired and rejects malformed/spoofed payload fields; `world.leave` and disconnect release membership.
- [x] Join checks rollout player/map allowlists, current map/instance/combat/onboarding state, and exact map/content versions before adding a server-built entity.
- [x] Giới hạn 6 lần thử join/phút theo nhân vật; dọn ID hết hạn khi limiter chạm ngưỡng 10.000 entry; failed admission giải phóng lease. Test vượt ngưỡng và hết cửa sổ 1 phút.
- [x] Tách movement input vào queue riêng có giới hạn (64/client), giới hạn 30 input/giây/kết nối, kiểm tra controller lease rồi mới nộp vào simulation queue.
- [x] Fanout AOI `world.snapshot` 20 Hz qua feed mỗi controller có buffer 1; WS outbound snapshot slot coalesce delta tương đối baseline mới nhất, tách khỏi reliable command queue.
- [x] Slow consumer không block zone tick; snapshot cũ bị thay, write timeout đóng kết nối chậm. [ ] Reliable event replay và metrics QoS còn pending; mở rộng rate limit cho cast/chat sau khi các route đó bật.
- [x] Join/leave/reconnect and second-tab single-controller lease within one process; [ ] logging queue len, drops, malformed, RTT and multi-process lease coordination.
- [x] Phaser-side protocol transport authenticates with existing token/tab identity, sends `world.join`, awaits/validates `world.welcome`, buffers AOI snapshots, and sequences normalized `world.input`.

**Tiêu chí merge/nghiệm thu:** WS tests unauthorized, spoofed ID, flood, disconnect, backpressure; `match.input`/chat legacy không lỗi.

## PR-09 — Headless 20Hz simulation & entities  
**Milestone:** M1 · **Phụ thuộc:** 05 · **Trạng thái:** IN PROGRESS

**Đường dẫn:** `backend/internal/realtime/{sim,entity,telemetry}/*` (new), `backend/internal/match/{match.go,geometry.go}` (read/reference existing)

**Các task:**
- [x] Build fixed-step `Step()` deterministic, actor-owned state, bounded command inbox; float XY, collision radius, static tile collision and wall sliding.
- [x] Define `EntityID`, `CharacterID`, structured `ZoneKey {MapID, InstanceID, RulesetVersion}`, transform/vitals/movement stats/faction/revision and 2D movement constants.
- [x] Keep DB/network calls outside `Step()`; [ ] trace event ordering until gameplay events are introduced; [x] initialize each runtime zone with a fresh server-generated RNG seed and allow explicit seeds for repeatable replays/tests.
- [x] Bound per-tick command intake and movement substeps; [ ] add a separate budget if a deferred gameplay-event queue is introduced.
- [x] Add automated sim coverage for two players, ten mobs and 10k steps, plus concurrent input/step coverage; [x] run Go race test at queue boundaries. The headless command also exercises a 12-entity 10k-step run.

**Tiêu chí merge/nghiệm thu:** 20Hz fixed tick independent from wall clock; no data race; client never modifies authoritative position/HP.

## PR-10 — Zone Manager lifecycle & ownership lease  
**Milestone:** M1 · **Phụ thuộc:** 02,08,09 · **Trạng thái:** IN PROGRESS (authenticated join, WASD/click-to-move, loaded static map collision and safe spawns wired; persistent ownership and multi-process lease pending)

**Đường dẫn:** `backend/internal/realtime/world/*` (new), `backend/cmd/server/main.go` (existing)

**Các task:**
- [x] Optional startup creates/stops configured zones; max 64 zones/process. Manager supports process-local join/leave with single-owner leases, stale-token protection, capacity enforcement, and bounded lifecycle commands applied on zone ticks.
- [x] Legacy monster tick filters to the default world with empty `InstanceID`; selected legacy movement/combat/loot/teleport handlers reject characters with an active realtime lease.
- [x] Wire manager lifecycle to authenticated join/leave/reconnect, trusted DB region/instance/position/vitals snapshot, map validation, and rollout allowlist/version gates.
- [ ] Derive full gameplay stats and spawn/collision-safe coordinates from the canonical character/map services; audit and guard every remaining legacy state mutation path. Lease persistence/expiry remain TODO.
- [x] Join refuses ineligible, stale-version, unallowlisted, already-owned, and full-zone requests with structured errors.
- [x] Shutdown cancels and waits for configured zone loops; process stop clears process-local leases. [ ] Drain player input, flush checkpoints and notify clients once gateway admission/persistence exist.

**Tiêu chí merge/nghiệm thu:** Concurrent joins/leave/reconnect do not create dual ownership; process stop cleans zone safely.

## PR-11 — Asset manifest & world visual foundations  
**Milestone:** M1 · **Phụ thuộc:** 03 · **Trạng thái:** IN PROGRESS (world-unit scale, feet-anchor depth sort, map rendering, camera follow/resize and placeholder textures landed; atlas manifest/animation library and debug grid overlay pending)

**Đường dẫn:** `frontend/src/game/assets/{AssetManifest,AnimationLibrary}.ts` (new), `frontend/public/game/{atlases,fx,maps}/*` (new), `frontend/src/game/world/{CameraController,DepthSort,WorldUnits}.ts` (new)

**Các task:**
- [x] Chốt world-units <-> pixel ratio, character feet anchor, depth by ground Y, layer order. (`PIXELS_PER_UNIT=24`, chân sprite ở `y`, depth = pixel feetY.)
- [ ] Định nghĩa image atlas manifest (icon, frames, animation states, sound, fallback, content hash/license).
- [x] Load placeholder player/NPC/monster, idle/walk/cast; object pools for repetitive fx. (Placeholder token textures; chưa có animation/pool.)
- [x] Scene resize/zoom 2.5D, camera follow, debug coordinate/grid overlay. (Resize/camera/HUD tọa độ; grid overlay mới ở mức tile nền.)

**Tiêu chí merge/nghiệm thu:** Một scene Phaser hiển thị map + 2 player sprites + 3 monsters đúng tọa độ/độ sâu ở nhiều screen sizes.

## PR-12 — Tiled map pipeline & server collision artifacts  
**Milestone:** M1 · **Phụ thuộc:** 09,11 · **Trạng thái:** IN PROGRESS (legacy Region/MapCell/MonsterSpawn conversion now generates shared `world_map`; collision terrain, region-aware rules/transfers and A* nav remain pending)

**Đường dẫn:** `frontend/src/game/world/MapLoader.ts` (new), `frontend/public/game/maps/*` (new), `backend/internal/realtime/nav/*` (new), `backend/scripts/convert_world_map.*` (new)

**Các task:**
- [x] Chọn Tiled JSON `.tmj`/export JSON format thống nhất; collision, nav, spawn, portal, AOI chunk, region layer. (Subset orthogonal: ground/collision/spawns; portal/AOI/region chưa.)
- [x] Build script chuyển/gắn metadata từ `Region.csv`, `MapCell.csv`, `MonsterSpawn.csv` có report không chuyển được. (`scripts/convert_legacy_regions_to_mmo.py`; missing collision is not fabricated, missing spawn positions are deterministic estimates, runtime cap reported.)
- [x] Xuất server-readable collision/grid/nav hashes và client asset map manifest **từ cùng source**. (Cùng JSON; hash collision ở server; test Go bắt buộc hai bản copy giống nhau.)
- [x] Validate map bounds, blockers, no spawn in solid, portal partner exists, tile scale and mapVersion. (Bounds/blockers/spawn/scale/version có; portal chưa có.)
- [x] Require every enabled realtime zone to load its configured map/content version before manager/admission starts; report loaded map dimensions/hash/legacy transform/monster spawn count in runtime monitor. Missing or mismatched artifacts now fail closed instead of admitting into an empty zone.

**Tiêu chí merge/nghiệm thu:** Client và Go shared golden collision data, đường đi qua tường bị chặn; file map và hash nhất quán.

## PR-13 — Authoritative movement / WASD + click target  
**Milestone:** M1 · **Phụ thuộc:** 06,08,09,10,12 · **Trạng thái:** IN PROGRESS (WASD and bounded click-to-move share the sim movement path and ack; map collision/nav and dynamic broadphase pending)

**Đường dẫn:** `backend/internal/realtime/{movement,nav,spatial}/*` (new), `frontend/src/game/input/ClickToMove.ts` (new)

**Các task:**
- [x] Tick computes move velocities based on stats/status; collision sweep/sliding test, stop on stun/root. (Speed/status hook chưa có; collision sweep/sliding đã có trong sim test.)
- [x] Implement `move_target` authoritative bounded path/waypoint route; WASD cancels target; user click cannot teleport. (Đường thẳng bounded 64 units, chưa có A*.)
- [x] Static collision + dynamic broadphase, movement clamp and anti-speed hacks; server sends ackSeq, explicit errors. (Clientside/server static collision loaded; dynamic broadphase chưa.)
- [x] No SQL or `ExecuteMovePlayerPath` calls from MMO movement; pathfinding budget and timeouts. (Không SQL; chưa có nav budget vì chưa có pathfinding.)

**Tiêu chí merge/nghiệm thu:** 5–10 browsers move in same zone smoothly; WASD and mouse work interchangeably, no pass-through walls, no DB-per-step.

## PR-14 — AOI, snapshots, delta + clock sync  
**Milestone:** M2 · **Phụ thuộc:** 07,08,09,10,13 · **Trạng thái:** IN PROGRESS (spatial AOI, baseline-relative deltas, server-tick clock and client interpolation wired; AOI hysteresis and automatic resync pending)

**Đường dẫn:** `backend/internal/realtime/spatial/*` (new), `backend/internal/realtime/gateway/*` (new), `frontend/src/game/network/SnapshotBuffer.ts` (new)

**Các task:**
- [x] Add a 16-unit spatial hash and per-controller circular AOI radius 32; always include the owner and sort visible entities by ID. AOI hysteresis/visibility masks remain pending.
- [x] Stream AOI snapshots at 20 Hz; send a full baseline at join and every 40 ticks, with baseline-relative deltas carrying `entityRevision`, `removed`, and `ackSeq`. Repeated deltas allow latest-state coalescing without losing earlier changes/removals.
- [ ] Clock-sync snapshots, client time interpolation/resync; budget bytes/client and slow-client telemetry.
- [x] Test AOI filtering, stable ordering, delta removals and baseline refresh. [ ] Test delayed/lost baseline recovery and AOI enter/leave hysteresis; never stream entire world to each browser.

**Tiêu chí merge/nghiệm thu:** With 2 distant clusters, clients only see nearby entities; resync correct; snapshot bytes measured.

## PR-15 — World NPC/monster entity spawns & AI  
**Milestone:** M2 · **Phụ thuộc:** 09,10,12,13 · **Trạng thái:** IN PROGRESS (server-owned monster spawns from map+monster_templates, 4 Hz FSM, attacks, leash, death/respawn and client monster rendering landed; threat/party/LOS/loot and archetype tests beyond the sim layer pending)

**Đường dẫn:** `backend/internal/realtime/ai/*` (new), `backend/internal/realtime/world/spawns.go` (new), `backend/data/{Monster.csv,MonsterSpawn.csv}` (existing input)

**Các task:**
- [x] Adapt monster templates and spawn groups; idle/patrol/aggro/chase/attack/leash/death/respawn FSM. (Templates load from `monster_templates`; spawns authored per map.)
- [x] Spatial query and AI think 2–5Hz; cap repath and search; deterministic aggro/threat tie-break. (Linear nearest-player query at 4 Hz, no repath/nav yet.)
- [x] Prevent old `world_monsters.go` from updating those zone monsters; no `startWorldMonsterEncounter`. (Zone monsters live only in the new sim; legacy worker skips realtime-owned players.)
- [ ] Integration tests 3 monster archetypes including blocked path and leash distance; AI metrics. (Sim tests cover aggro/attack/leash/patrol/damage/respawn; 3 archetypes + blocked path pending.)

**Tiêu chí merge/nghiệm thu:** 10 monsters chase player around obstacles, stop/reset and respawn; no turn-based transition.

## PR-16 — Realtime combat v1: cast, projectile, hit  
**Milestone:** M2 · **Phụ thuộc:** 05,09,13,14,15 · **Trạng thái:** IN PROGRESS (server cast pipeline, four target modes, projectiles, ground AoE, shield/heal, interruption and world.events landed; Gu loadout/effect parity, LOS and death UI pending)

**Đường dẫn:** `backend/internal/realtime/combat/*` (new), `backend/internal/match/{input.go,geometry.go,match.go}` (read/reference)

**Các task:**
- [x] Cast state machine with windup/execute/recovery/cooldown/GCD; reject unknown Gu, range/LOS, status and mana. (Windup/GCD/cooldown/mana/range có; LOS/status chưa.)
- [x] TARGET, DIRECTION, GROUND, SELF; projectile collision, AoE and target cap; server damage, shield, first debuff. (Damage/shield; debuff chưa.)
- [x] Server emits authoritative events for animation, hit, damage, death; no client damage field. (world.events; chưa có replay tin cậy.)
- [x] Tests simultaneous hits/interrupt, projectile crossing, LOS, movement-cast policy and long RTT duplicate cast. (Interrupt/projectile/duplicate/movement-lock sim tests; LOS/long-RTT chưa.)

**Tiêu chí merge/nghiệm thu:** 3 players can simultaneously attack same boss in zone without separate combat screen; HP agrees for all.

## PR-17 — Complete Gu effect data inventory/audit  
**Milestone:** M2 · **Phụ thuộc:** 01,16 · **Trạng thái:** IN PROGRESS (deterministic realtime audit CLI + generated coverage CSV + six MVP selection landed; hidden passive authoring and VFX/asset validation pending)

**Đường dẫn:** `backend/cmd/gudata/*` (existing), `backend/internal/gameplay/effects/*` (existing), `backend/data/{Gu.csv,GuMoba.csv,Status.csv}` (existing), `docs/mmo/gu-coverage.csv` (generated)

**Các task:**
- [x] Enumerate `(gu_id,rank,block_id,trigger,operation,selector,condition)` and rank coverage; detect unknown/unhandled ops. (`audit-realtime`, unclassified=0.)
- [x] Categorize `SUPPORTED`, `NEEDS_REAUTHOR`, `UNSUPPORTED`, `NOT_ACTIVE`; produce machine-readable report + summary counts. (`gu-coverage.csv` + on-demand JSON.)
- [ ] Pick six MVP Gu and a balanced 30–50 Alpha Gu across damage/defense/control/heal/passive/hidden. (Six MVP chọn xong; hidden passive chưa có trong data; Alpha pool chưa.)
- [ ] Golden unit tests and baseline hashes for Gu catalog; admin warning for unsupported skills. (Real-catalog test + count checks có; baseline hash/admin warning chưa.)

**Tiêu chí merge/nghiệm thu:** Every authored Gu block in source has an explicit migration status; no silent ignore path.

## PR-18 — Gu authoring schema & compiler adapter  
**Milestone:** M3 · **Phụ thuộc:** 16,17 · **Trạng thái:** IN PROGRESS (schema v1, six authored profiles, compiler with fail-closed validation, CLI and startup compile landed; per-player loadout adapter, passive application and status execution pending)

**Đường dẫn:** `backend/internal/realtime/effects/{schema.go,compiler.go,validation.go}` (new), `backend/internal/gameplay/effects/{contract.go,registries.go}` (existing)

**Các task:**
- [x] Author new realtime fields: active mode/timing/movement policy/range/status durations/stack/visual IDs; versioned schema.
- [x] Compiler maps safe turn-based primitives to realtime formula; marks semantic turn-specific triggers for reauthoring. (STATUS bắt buộc durationMs; UNIQUE bị từ chối.)
- [x] Implement validation for missing assets, invalid durations, target policy, rank mismatch, unsupported operation. (Asset chỉ kiểm tra presence tên; existence check ở PR-21.)
- [x] Build immutable compiled skills loaded at zone startup/content deploy; tests don't depend on live DB. (Startup compile/log; runtime loadout adapter để PR-23.)

**Tiêu chí merge/nghiệm thu:** Six MVP Gu compile with correct mode and effect payload; unsupported blocks fail visibly with actionable errors.

## PR-19 — Realtime effect engine + hidden passive  
**Milestone:** M3 · **Phụ thuộc:** 16,18 · **Trạng thái:** IN PROGRESS (tick statuses/DoT/HoT/stacks/cleanse/PvP duration DR, movement/cast blocking and a preview hidden passive landed; generic hidden-passive authoring/loadout and native UNIQUE handlers pending)

**Đường dẫn:** `backend/internal/realtime/effects/{runtime,event_bus,scheduler,status,passive,handlers}*.go` (new), `backend/internal/gameplay/effects/*` (existing/reference)

**Các task:**
- [x] Implement damage/heal/shield/direct_hp/stat_mod/status/barrier/cleanse; scope special UNIQUE via native handler registry. (Damage/heal/shield/status/cleanse có; barrier có qua shield hiện tại, native UNIQUE vẫn bị compiler từ chối.)
- [x] Timed tick-based DoT/HoT, stacks, dispel, immunity, proc chance, internal cooldown, chain depth/anti-infinite events. (DoT/HoT/stacks/dispel/ICD có; immunity/proc chance/chain budget chưa.)
- [x] Hidden passive triggers (e.g. HP low + 3 hits) server side; discovery state separate from proc state. (Preview `PREVIEW_LAST_STAND`; chưa author theo Gu/loadout.)
- [x] Golden tests for before-hit/damage/after-hit/death ordering, shield priority, cooldown and cross-gu combos. (Shield priority/status/interrupt/hidden tests; cross-gu combo và event ordering đầy đủ chưa.)

**Tiêu chí merge/nghiệm thu:** MVP 6 Gu incl hidden passive; deterministic 1k combat replay tests; unsupported UNIQUE cannot silently pass.

## PR-20 — Hybrid skill targeting + cast UI in Phaser  
**Milestone:** M3 · **Phụ thuộc:** 06,11,16,18 · **Trạng thái:** IN PROGRESS (server hotbar projection, React HP/mana/cooldown/cast/target HUD, two-step aim telegraphs, target lock and snapshot projectile rendering landed; VFX/aim cancel polish and automated UI tests pending)

**Đường dẫn:** `frontend/src/game/combat/{CastTelegraph,ProjectileView}.ts` (new), `frontend/src/components/game/MMOHud.tsx` (new), `frontend/src/game/core/WorldRuntime.ts` (snapshot HUD projection)

**Các task:**
- [x] Hotbar 1–6 maps equipped Gu instances to server slots; active range/cooldown/mana with controlled UI throttling. (Hotbar lấy `world.welcome.skills`; loadout theo Gu sở hữu để PR-23.)
- [x] TARGET lock, DIRECTION aim, GROUND reticle, SELF immediate; ESC cancel, target death invalidation. (Right-click cancel có; aim UI 2 bước.)
- [x] Client anticipatory cast VFX (cosmetic) reconciles to server accepted/rejected events; no duplicate explosion. (Telegraph/projectile + events; VFX pool chưa đầy đủ.)
- [ ] Test input UI focus conflicts, aim outside map, target revision, rejection reason and effect cancel.

**Tiêu chí merge/nghiệm thu:** Player casts 4 target modes via keyboard/mouse, VFX matches server hit; no W/D hotkey collision.

## PR-21 — Sprites/animation/assets integration  
**Milestone:** M3 · **Phụ thuộc:** 11,12,15,20 · **Trạng thái:** IN PROGRESS (procedural placeholder actor frames, animation state machine, asset manifest/license, VFX pool and combat event → animation mapping landed; packed atlas, 4/8-direction art, asset existence validation and profiling pending)

**Đường dẫn:** `frontend/src/game/{entities,assets,combat}/*` (new), `frontend/public/game/{atlases,fx,maps}/*` (new)

**Các task:**
- [ ] Create top-down 2.5D sprites/atlases or licensed placeholders: player idle/run/cast/hit/death, monsters, NPC, projectiles. (Placeholder generated frames có; atlas thật chưa.)
- [x] Depth/camera/foreground layering, multi-direction animation, server event -> animation graph. (Depth/camera + animator; multi-direction mới flipX.)
- [x] Pool floating numbers, projectiles and VFX; cap particles per frame and texture memory. (Text/ring pool; projectile view; particle cap chưa.)
- [x] Add explicit asset manifest/version/licenses and fallback sprites for missing Gu icons/skills. (Manifest/license/fallback tokens; Gu icon thật chưa.)

**Tiêu chí merge/nghiệm thu:** First map plays 30 minutes with 3 monsters, stable depth and VFX; profiling reports frame-time & VRAM.

## PR-22 — Click-to-move polish, target/camera/UX  
**Milestone:** M3 · **Phụ thuộc:** 06,07,12,13,20,21 · **Trạng thái:** IN PROGRESS (client A* path preview/waypoint execution, blocked-path feedback, camera deadzone/zoom/shake, offscreen target marker landed; NPC interaction markers and full E2E scenarios pending)

**Đường dẫn:** `frontend/src/game/{input,world,network}/*` (new), `frontend/src/components/game/MMOHud.tsx` (new)

**Các task:**
- [x] Path preview markers + blocked/repath feedback; keyboard cancels path, manual skill aim disables movement click. (A* client preview; WASD/Esc cancel; aim mode chặn click move.)
- [x] Camera follow deadzone, UI scale, interaction markers, zoom restrictions and offscreen indicators. (Deadzone/zoom/shake/offscreen; interaction markers NPC chưa.)
- [x] Path target NPC/monster behavior explicitly (move into interact/cast range then stop) and not an implicit autoattack. (Click quái chỉ target-lock; chưa auto-approach NPC.)
- [ ] Test interrupted path, click target behind wall, player death, teleports and loss/rejoin.

**Tiêu chí merge/nghiệm thu:** Both modes equally smooth and no character position snap larger than configured correction under normal RTT.

## PR-23 — Old domain adapters: character/Gu/quest/tu luyện  
**Milestone:** M4 · **Phụ thuộc:** 10,16,18 · **Trạng thái:** IN PROGRESS (join-time equipped-Gu loadout adapter, per-session immutable skills/loadout revision and real skill casting landed; live equip refresh, stat/equipment projection and quest/NPC triggers pending)

**Đường dẫn:** `backend/internal/realtime/persistence/{character_adapter,gu_loadout_adapter}.go` (new), `backend/internal/services/*` (existing), `backend/internal/models/*` (existing)

**Các task:**
- [x] Character enter zone: load stats/equipment/Gu instances/progression from domain services, cache immutable loadout revision. (HP/position + equipped Gu; stats/progression chưa project.)
- [ ] Equipment/Gu changes command to zone at safe point; balance version and authorize ownership server-side.
- [ ] Quest/NPC world triggers as explicit domain commands; block old turn-based combat path for MMO zones. (Legacy combat path đã bị guard bởi lease; trigger quest/NPC chưa.)
- [ ] Test changes while casting, expired item, insufficient resources, invalid Gu ownership, reconnect refresh.

**Tiêu chí merge/nghiệm thu:** Existing accounts, inventory and Gu loadouts work in new zone without duplicating domain logic.

## PR-24 — React gameplay HUD & business UI integration  
**Milestone:** M4 · **Phụ thuộc:** 04,20,23 · **Trạng thái:** IN PROGRESS (inventory/Gu, cultivation, party, settings drawers and global chat mounted in MMO view; modal focus blocks Phaser input; deeper panel state/typed bridge tests and NPC dialog pending)

**Đường dẫn:** `frontend/src/components/WorldPage.tsx` (existing), `frontend/src/components/{GuQuickBar,DirectActionHotbar,GuInventory,Aperture}.tsx` (existing), `frontend/src/components/game/*` (new)

**Các task:**
- [x] Reuse existing React overlays for cultivation, inventory, gu equipment, chats, settings, party with typed event bridge. (Drawer + chat; bridge dùng Zustand + runtime coarse state.)
- [x] Keep game state snapshots within `GameSession`; React only subscribes coarse HP/cooldown/target and domain state. (HUD throttle 10 Hz; panel state từ store.)
- [x] Introduce soft pause policy: opening bag doesn't freeze MMO world; modal focus blocks hotkeys. (Client modal chặn input; server tiếp tục chạy.)
- [ ] Correct responsive overlay, tooltips, NPC dialog, interaction prompts and error messages.

**Tiêu chí merge/nghiệm thu:** Open/close legacy business panels during live combat without duplicated movement or frame-time spikes.

## PR-25 — Loot, inventory transactional grants & death  
**Milestone:** M4 · **Phụ thuộc:** 15,16,23 · **Trạng thái:** IN PROGRESS (idempotent death→loot worker, DB transaction grant for items/Gu and replay-safe keys landed; corpse/ground drop, F interaction, party loot, client toast and DB integration tests pending)

**Đường dẫn:** `backend/internal/realtime/loot/*` (new), `backend/internal/services/InventoryService...` (existing as applicable), `backend/internal/realtime/persistence/*` (new)

**Các task:**
- [x] Monster death chooses reward entitlement server-side; generate unique death/loot grant id per recipient and source. (Auto-grant killer; party entitlement chưa.)
- [x] Domain service transactional ledger + unique constraint/outbox; retry safely on reconnect/process crash. (Grant + inventory/Gu trong một transaction, unique key; crash nonce còn thiếu.)
- [ ] Corpse/ground drop TTL, F interaction, distance/ownership checks and bag capacity policy.
- [ ] Simulate duplicate message, replay kill event, concurrent party loot, DB failure recovery; enforce no dupe. (Unit roll/key tests + manager death test; DB integration chưa.)

**Tiêu chí merge/nghiệm thu:** 100× retry same grant yields exactly one economic outcome; drop cannot be collected across forbidden zone/LOS.

## PR-26 — Reconnect, checkpoint, safe spawn & recovery  
**Milestone:** M4 · **Phụ thuộc:** 08,10,23,25 · **Trạng thái:** IN PROGRESS (durable world checkpoint table, join resume/version gates, leave+10s flush and safe spawn resume landed; session resume token, combat logout policy, durable death nonce/outbox and DB integration tests pending)

**Đường dẫn:** `backend/internal/realtime/{persistence,world,gateway}/*` (new), `backend/internal/db/*` (existing migration adapters)

**Các task:**
- [x] Durable world state v2 stores zoneId/x/y/HP/mapRevision/lastSafeSpawn; checkpoint not per tick.
- [x] Reconnect actor/session resume token, input ack reset and full snapshot; separate action event replay. (Checkpoint resume/ack reset/full welcome có; resume token chưa.)
- [ ] Simulate WS disconnect, refresh during PvP, zone crash and slow DB; handle combat logout & AFK rules.
- [ ] Crash-safe inventory grants/ledger + transfer, no lost Gu or duplicated player entity. (Grant transaction có; unprocessed death outbox/transfer chưa.)

**Tiêu chí merge/nghiệm thu:** Player can reload and return near last validated position without duplication or turn-based fallback trigger.

## PR-27 — Portals, multiple zones, transfers  
**Milestone:** M4 · **Phụ thuộc:** 10,12,14,26 · **Trạng thái:** IN PROGRESS (portal map metadata + client markers, process-local atomic Manager.TransferCharacter, and KHU_VUC/SAFE_ZONE region-map cutover landed; automatic portal trigger, DB map update, world.transfer/client scene swap and multi-process fencing pending)

**Đường dẫn:** `backend/internal/realtime/{transfer,world}/*` (new), `frontend/src/game/core/GameSession.ts` (new)

**Các task:**
- [x] Portal definitions in map Tiled metadata and zone routing, safe spawn constraints. (Parse/validate/render portal; trigger/routing chưa.)
- [x] Atomic ownership handoff; freeze/transfer/revision/commit; on failure resume source or safe spawn. (Process-local atomic remove/add/owner update + rollback; freeze/revision/DB chưa.)
- [ ] Client loading overlay and scene/map/content swap without constructing duplicate Game/WS.
- [ ] Cross-zone entity and skill cleanup + party follow/late join handling and retries.

**Tiêu chí merge/nghiệm thu:** Transition 3 maps under disconnect/restart/retry tests: exactly one zone owns character and rewards intact.

## PR-28 — Party, Boss, PvP ruleset/arena world  
**Milestone:** M5 · **Phụ thuộc:** 16,19,23,25,27 · **Trạng thái:** IN PROGRESS (map ruleset pve/pvp/safe, party-aware harm rules, party snapshot projection, boss threat table + leash reset + splash landed; duel consent, party loot, boss telegraph/arena world, live party changes pending)

**Đường dẫn:** `backend/internal/realtime/{world,combat,ai}/*` (new), `frontend/src/components/{PartyView,ArenaView}.tsx` (existing), `frontend/src/components/game/*` (new)

**Các task:**
- [ ] Party/group ID, share loot rules, party target marker, Boss threat and AoE telegraph. (Party ID/threat/splash có; party loot/marker/telegraph chưa.)
- [ ] Region rules: safe/PvE/PvP, friendly-fire, duel consent, damage/CC diminishing returns. (Ruleset+same-party deny+CC DR có; duel consent/safe sub-zone chưa.)
- [x] PvP action allowed/blocked by server and same combat engine; old `match` can remain separate legacy until cutover.
- [ ] Multi-player Boss regression: threat switching, wipe/leash, simultaneous final hit, no duplicate drop. (Threat/splash/leash tests có; wipe/final-hit/loot chưa.)

**Tiêu chí merge/nghiệm thu:** One group clears dungeon boss; 2 player duel uses same zone combat, party loot and PvP rules consistent.

## PR-29 — Vertical-slice end-to-end gate  
**Milestone:** M5 · **Phụ thuộc:** 13,14,15,16,19,20,21,23,24,25,26 · **Trạng thái:** IN PROGRESS (headless 5-player/10-mob slice harness and Gate B checklist landed; browser E2E, per-owned-Gu runtime status, corpse/F loot, DB reconnect-under-combat and load measurement pending)

**Đường dẫn:** `frontend/src/app/page.tsx` (existing), `frontend/src/components/WorldPage.tsx` (existing), `backend/cmd/server/main.go` (existing), `tests/e2e/mmo/*` (new)

**Các task:**
- [ ] Create E2E flow login -> mmo gate -> move WASD -> mouse travel -> target -> cast -> kill -> loot -> bag -> exit -> reconnect. (Headless flow có; browser E2E chưa.)
- [ ] Verify legacy player outside sandbox unaffected, no legacy `world.move_path` or `startWorldMonsterEncounter` for zone members. (Guards/default OFF có; E2E assertion chưa.)
- [ ] Formal checklist for all 6 MVP Gu incl hidden passive; QA logs/screenshot/video manually if helpful. (`docs/mmo/vertical-slice.md` checklist; per-Gu hidden passive chưa.)
- [ ] Gate OFF by default production; bundle/version/flags documented; known issues triaged. (Flags/version docs có; browser/bundle verification chưa.)

**Tiêu chí merge/nghiệm thu:** Gate B PASS with 5 players, 10 mobs, 6 Gu; no blocker bugs; reproducible one-command staging setup.

## PR-30 — QoS profiling, client/server optimization  
**Milestone:** M5 · **Phụ thuộc:** 14,21,26,29 · **Trạng thái:** IN PROGRESS (synthetic tick benchmark, Manager.Stats telemetry, client network/prediction F3 HUD and measured 5/30/100-player CPU report landed; browser FPS/VRAM, RTT/jitter/loss, AOI bytes and DB/Redis load not measured)

**Đường dẫn:** `backend/internal/realtime/telemetry/*` (new), `frontend/src/game/debug/RealtimeDebugHUD.ts` (new), `tests/load/*` (new)

**Các task:**
- [ ] Instrumentation tick p50/p95/p99, GC/heap, AOI entities/client, outbound bandwidth, drop/resync, RTT, frame-time FPS. (Tick/entity/client HUD + Manager stats có; AOI bytes/drop/resync/RTT chưa.)
- [ ] Network emulation 50/100/150/200ms RTT + jitter/packet loss, local prediction/reconcile; projectile smoothing.
- [ ] Bench 5, 30, 100 players/zone with realistic skill bursts/mobs; capture hardware/setup & plots. (Synthetic in-process 5/30/100 có; real network/browser chưa.)
- [ ] Optimize atlas/batching/pooling, query spatial, payload sizes, AI budget based on actual hotspots. (Alloc hotspots identified, optimization chưa.)

**Tiêu chí merge/nghiệm thu:** Published benchmark report with pass/fail against Gate B/C/D; no performance claim without recorded results.

## PR-31 — Safe cutover, deprecate Pixi/turn combat hot path  
**Milestone:** M6 · **Phụ thuộc:** 26,27,28,29,30 · **Trạng thái:** IN PROGRESS (admission pause/drain rollback, graceful-shutdown drain, legacy mutation guard coverage and cutover runbook landed; cohort canary, full mutation audit, admin drain endpoint and Pixi removal deferred)

**Đường dẫn:** `frontend/{package.json,src/app/page.tsx}` (existing), `frontend/src/components/{WorldPage,WorldCanvas}.tsx` (existing), `frontend/src/components/PvPArena/*` (existing), `backend/internal/handlers/{exploration_move,world_monster_encounter,world_monsters}.go` (existing)

**Các task:**
- [ ] Enable opt-in by cohort -> canary -> default MMO via runtime flag; migrate old player coordinates using safe spawn mapping. (Flags/allowlist/safe spawn có; canary thật chưa.)
- [x] Disable old movement/combat handlers **for MMO mode** using server-side guards; keep legacy endpoints only while any cohort needs them. (Guard explore/move/teleport/combat/capture/arena/PvP/ground loot + WS; domain mutation audit chưa.)
- [ ] After successful canary, remove Pixi dependency and old canvas/arena visuals plus stale store code; stage removal separate from critical migration.
- [x] Safe rollback: block new MMO admits, drain & ledger reconcile, send players to legacy last-safe spawn without item-loss. (Pause/Drain checkpoint/release + runbook; admin endpoint/real rehearsal chưa.)

**Tiêu chí merge/nghiệm thu:** Old and MMO cohorts cannot mutate same player/monster; cutover & rollback rehearsed on staging; UI contains no Pixi for MMO.

## PR-32 — Multi-zone worker scaling + instance capacity  
**Milestone:** M6 · **Phụ thuộc:** 10,14,26,27,30 · **Trạng thái:** IN PROGRESS (in-process multi-instance allocation/capacity, per-zone goroutines and Stats landed; dedicated worker processes, cross-process leases/routing, zone drain and transfer storms pending)

**Đường dẫn:** `backend/internal/realtime/{world,gateway,transfer,telemetry}/*` (new), deployment configs (new)

**Các task:**
- [ ] Split zone worker process only after single-process bench proven; registry/lease location and graceful zone drain. (Instances/goroutines/capacity có; worker process/drain chưa.)
- [ ] Allocate instances by load/capacity; route authenticated WS/gateway to owner, no broadcast all-zone. (Allocation trong process có; routing/cross-process chưa.)
- [ ] Cross-process portal/party transfer, lease fencing, recovery on worker crash; worker capacity autoscaling policy.
- [ ] Load tests multiple zones with transfer storm & reconnection, observe memory/latency.

**Tiêu chí merge/nghiệm thu:** 10 zones/worker test with accurate ownership; no split-brain, no double rewards under worker crash simulations.

## PR-33 — CI/CD staging + observability + content versioning  
**Milestone:** M6 · **Phụ thuộc:** 01,02,05,12,17,30 · **Trạng thái:** IN PROGRESS (CI Go/frontend/Gu checks, admin realtime stats endpoint, map/Gu hash gates and operations runbook landed; Prometheus/OTel dashboards, redaction audit and staging dry-run pending)

**Đường dẫn:** `backend/Dockerfile` (existing), `frontend/package.json` (existing), `.github/workflows/*` (existing), `docs/mmo/operations.md` (new)

**Các task:**
- [x] Pin Go 1.26.3/Node and Phaser lockfile in CI; pipeline Go tests+race, TypeScript typecheck/test/build, map lint and Gu compiler report. (Map parity test + Gu audit/compile trong CI; Prometheus chưa.)
- [ ] Build frontend/backend content hash compatibility check on join; roll back artifacts/flags without corrupting DB. (Welcome map hash + version gates có; Gu content hash handshake chưa.)
- [ ] Prometheus/OpenTelemetry or equivalent logs/metrics by zone, player/session, network, economy; redact user secrets.
- [ ] Staging smoke and migration dry run; backups/restores & incident playbook.

**Tiêu chí merge/nghiệm thu:** CI blocks incompatible protocol/map/effects; deploy canary+rollback repeatable; dashboards for tick/WS/errors/economy.

## PR-34 — Security, fuzz, soak, crash & dupe testing  
**Milestone:** M7 · **Phụ thuộc:** 25,26,30,32,33 · **Trạng thái:** IN PROGRESS (protocol/sim/map fuzz seeds, controller-reuse guard and reconnect/ownership soak harness landed; DB dupe replay, 6–24h soak, crash/jitter injection and cross-account audit pending)

**Đường dẫn:** `backend/internal/realtime/{protocol,sim,nav}/*_test.go` (new), `backend/cmd/mmo_soak` (new), `docs/mmo/security.md` (new)

**Các task:**
- [x] Fuzz protocol invalid floats, oversized payload, aim/range, invalid Gu/equipment, replay inputs; cross-zone identity spoof. (Protocol/sim/map fuzz + spoof/oversize/controller tests; Gu/equipment fuzz chưa.)
- [ ] Soak test 6–24h load with realistic AI/Gu frequency, reconnect storms, snapshot packet queues and memory. (Harness `mmo_soak` có; chạy dài/thật chưa.)
- [ ] Failure injection DB/Redis disconnect, zone crash, gateway crash, network jitter; verify ledger/checkpoint/transfer integrity.
- [ ] Audit cross-account data/quest/party/PvP, unauthorized hit/loot and dangerous debug endpoints.

**Tiêu chí merge/nghiệm thu:** No critical exploit or inventory dupe in test plan; documented soak report, known residual risks and rollback path.

## PR-35 — Alpha content expansion + Cổ Trùng parity  
**Milestone:** M7 · **Phụ thuộc:** 17,18,19,21,28,34 · **Trạng thái:** IN PROGRESS (46 compiled Gu profiles, 4 linked maps with boss/PvP/safe zones, catalog/parity tests landed; NPC/quest/social/drop authoring, unique handlers, balance pass and GM replay pending)

**Đường dẫn:** `backend/data/Gu.csv` (source), `backend/data/mmo/gu/alpha_profiles.json` (new), `backend/data/mmo/maps/*` + `frontend/public/game/maps/*` (new), `docs/mmo/gu-coverage.csv` (generated)

**Các task:**
- [x] Expand vetted Gu pool 30–50 then prioritize by rarity/usage, add unique handlers and multiple synergy combos. (40 Alpha catalog profiles compiled; unique handlers/synergies chưa.)
- [x] Add 3–5 Tiled maps, NPC, Boss, quests, social and drops; content QA through compiler/assets validation. (4 map + boss/PvP/safe; NPC/quest/social/drop chưa.)
- [ ] Damage/status/hidden passive balance for PvE and PvP separate modifiers; GM/debug replay tools.
- [x] Track coverage % and issue count; defer unsupported high-concept mechanics explicitly rather than force fake behavior. (Coverage CSV + compiler strict; số % cập nhật trong `docs/mmo/gu-coverage.md`.)

**Tiêu chí merge/nghiệm thu:** Gate C: curated content works reliably, regression test for every playable Gu; documented migration gaps.

## PR-36 — Beta release, operational criteria & future roadmap  
**Milestone:** M7 · **Phụ thuộc:** 30,31,32,33,34,35 · **Trạng thái:** IN PROGRESS (release checklist, candidate SLO/capacity gates, canary/rollback plan, post-beta roadmap, script release-gate tự động và workflow manual đã có; browser E2E, load thật, DB crash/dupe replay, staging canary và on-call drill chưa đo nên Gate D vẫn HOLD)

**Đường dẫn:** `docs/mmo/release-checklist.md` (new), `docs/mmo/slo.md` (new), `docs/mmo/roadmap.md` (new), `scripts/mmo-release-gate.sh` (new), `.github/workflows/release-gate.yml` (new)

**Các task:**
- [ ] Feature freeze + regression across movement/aim/AI/economy/social/cross-zone and disconnect scenarios. (Ma trận regression + script có; chạy thật/browser NOT MEASURED.)
- [ ] SLO thresholds from measured load, server budget and visibility radius; scaling or instance split if 100/zone fails. (Candidate thresholds + decision matrix có trong `docs/mmo/slo.md`; chưa có số đo thật.)
- [ ] Canary 5% -> 25% -> 100% with metrics and rollback; communicate world/Gu mechanic changes to players. (Kế hoạch cohort/rollback/comms có; chưa diễn tập staging.)
- [x] Plan later mobile touch UI, GvG/large-world optimization, expanded Gu catalog based on telemetry. (`docs/mmo/roadmap.md`.)

**Tiêu chí merge/nghiệm thu:** Gate D release only with signed QA/report: crash recovery, load, anti-dupe, consistency and operational on-call.

---

# 11. Milestone schedule, dependencies, checkpoints

| Milestone | PR chính | Phụ thuộc cứng | Exit gate | Thời gian tham khảo (1 dev có kinh nghiệm, full-time) |
|---|---|---|---|---|
| M0 — Setup/migration seam | 01–05 | toolchain, DB staging | Next+Phaser smoke và protocol fixtures; production legacy không đổi | 1–2 tuần |
| M1 — World foundation | 06–13 | M0, map colliders + zone owner | headless sim, Phaser điều khiển 5 người, WASD+click path | 2–4 tuần |
| M2 — First MMO combat | 14–17 | zone movement, AOI | 3 quái + PvE realtime + Gu coverage report | 2–3 tuần |
| M3 — Gu + visual gameplay | 18–22 | combat core, effect audit, asset pipeline | 6 Gu + hidden passive, VFX, hybrid controls hoàn chỉnh | 3–5 tuần |
| M4 — RPG integration + zones | 23–27 | gameplay core và domain services | login, bag, loot, persistence, 3 zones, no dupe | 3–5 tuần |
| M5 — Playable slice, party/PvP/load | 28–30 | end-to-end integration | playable Gate B/C + latency/perf report | 2–4 tuần |
| M6 — Migration, scaling, ops | 31–33 | reliable persistence & metrics | canary cutover + recovery/observability | 2–4 tuần |
| M7 — Safety, content & beta | 34–36 | all prior | measured Gate D + curated content + test report | 3–6 tuần + content |

**Tổng ước tính:** khoảng **18–33 tuần** nếu tuần tự với một kỹ sư dày dạn và đủ asset/hạ tầng; thực tế có thể lâu hơn khi cân bằng Cổ Trùng và làm nội dung. Các giai đoạn có thể chồng lấp giữa 2+ người, song nên bảo vệ đường găng **Protocol → Zone → Movement → Combat → Effect → Persistence → AOI/Scaling → Cutover**. Không hứa 100 player/zone trước benchmark.

### Gate rõ ràng để không lệch mục tiêu

1. **G0:** `npm run build` (Phaser 4), backend compile, E2E flag OFF legacy OK.
2. **G1:** 5–10 client trong cùng zone, WASD và click-to-move, camera 2.5D, collision, no DB path movement.
3. **G2:** 3 player/3 quái/6 Gu đánh realtime, target-lock + skillshot + AoE, no encounter/turn switch.
4. **G3:** hidden passive + loot idempotent + inventory/tu luyện UI; reconnect & zone transfer pass.
5. **G4:** AOI, 30 then 100 player load with measured p95/p99 and zero major dupe/security findings.
6. **G5:** old cohort cutover gradually, debug/rollback/release runbooks verified.

---

# 12. Checklist test matrix (bắt buộc, có thể đưa vào CI/QA)

| Suite | Trường hợp cần test | Pass condition |
|---|---|---|
| Movement-WASD | diagonal normalize, key-repeat, blur, stun/root, movement speed buff | không vượt tốc độ, dừng đúng |
| Movement-click | click on wall, behind blocker, target, UI click, WASD cancel, repath | không teleport/xuyên tường/auto skill |
| Projection | world<->pixel across zoom, DPR, camera edge, resizes | aim/click sai số trong tolerance đã định |
| Collision | static wall, corner slide, high velocity dash, dynamic entity | server/client parity, no tunneling |
| Network | 0/100/200ms RTT, ±50ms jitter, simulated loss/disconnect | local responsive; ack/replay stable, resync OK |
| Snapshot AOI | enter/leave AOI, entity reuse, baseline gap | no ghost/duplicate, no unauthorized view |
| Combat | target out of range, interrupted windup, cooldown, mana, LOS, PvP safe zone | no damage when invalid, deterministic results |
| Projectile/AoE | simultaneous hits, pierce, hit once, wall, moving target | no extra damage or dropped valid hit |
| Gu effects | stack/refresh, DOT/HOT, shield/cleanse, hidden proc, recursion, UNIQUE handler | event order correct and bounded |
| AI | multi-target aggro, leash, blocked path, boss wipe | no stuck infinite path/CPU spike |
| Loot | repeated kill/reward request, concurrent party, crash during commit | exactly-once durable effect |
| Persistence | reload during cast, disconnect in combat, zone crash, portal retry | one owner, no item dupe/loss |
| Phaser lifecycle | strict mode mount/unmount, route change, refresh/hot reload | no leak, one canvas and one WS gameplay session |
| Economy/regression | equip/unequip Gu, tu luyện, crafting, market, inventory | legacy semantics preserved unless explicitly reauthored |
| Load | 5, 30, 100 players in one zone, 50 monsters and skill bursts | captured tick p95/p99, CPU, bytes/client, frame times |

### Gợi ý câu lệnh CI (chạy ở thư mục phù hợp)

```bash
# frontend — có Node/npm và dependencies đã cài
cd frontend
npm ci
npm run typecheck
npm test
npm run build

# backend — dùng phiên bản Go phù hợp go.mod (1.26.3)
cd ../backend
go test ./...
go test -race ./internal/realtime/...  # sau khi tạo runtime

# load tests (NEW commands phải tự triển khai, ví dụ)
# go run ./cmd/mmo-loadtest -clients 30 -zone forest_01#1 -duration 5m
# node ./scripts/check-map-and-gu-assets.mjs
```

Các câu lệnh bắt đầu bằng comment `NEW` là ý tưởng tool tương lai, **không phải lệnh đã có trong ZIP**. Những test Go hiện tại có thể cần DB/services thật, phải thiết kế CI fixture có cấu hình riêng.

---

# 13. Critical path và phân chia người/agent

| Nhóm | Phụ trách | PR ưu tiên | Không đụng việc của nhóm khác |
|---|---|---|---|
| Backend Core | Zone loop, entity, movement, AI, combat | 05,08–10,12–16 | Không nhúng SQL vào tick; không sửa React UI |
| Effects & Gu | audit/translator/trigger/hidden passives | 17–19,35 | Không tự biến turn -> ms; không đổi Gu.csv không migration |
| Frontend Phaser | scenes, input, network prediction, camera/VFX | 03,04,06,07,11,20–22 | Không tự tính HP/loot; không trực tiếp sửa domain services |
| Domain & Data | account/inventory/quest/ledger/recovery/portal | 23–27 | Không gọi legacy encounter trong MMO zone |
| Ops/QA | E2E, load, canary, deployment | 01,02,29–34,36 | Không tuyên bố target đạt khi chưa đo |

**Nếu chỉ có 1 developer + coding agent**, mỗi PR tạo một nhánh nhỏ; merge ngay khi đạt test; không để 4 agent đồng thời sửa `frontend/src/app/page.tsx`, `backend/cmd/server/main.go` hoặc WS gateway. Công việc có thể chạy song song khi các DTO/protocol đã khóa bằng fixtures.

---

# 14. Những câu hỏi kỹ thuật cần trả lời bằng spike/test trước khi viết nhiều content

1. **Phaser 4.2.1 + Next 16 / Cloudflare deployment** có vấn đề WebGL/fetch asset/CDN/CSP nào? PR-03 phải đo bundle, build, SSR và browser refresh trên deployment target.
2. **Use existing shared WS or dedicated MMO WS?** PR-08 đo queue saturation khi chat/notifications + snapshot cạnh tranh; tách ngay nếu ảnh hưởng p95.
3. **Server authority for click path:** tính A* tại Go trong budget, hay allow bounded waypoint nav validated? Chốt ở PR-12/13 với replay test; không để cả hai đồng thời là truth.
4. **Collision format parity**: Tiled object polygons/slopes có cần? MVP chỉ grid/convex static/round actors; những shapes khác phải được lint/deny hoặc implement rõ.
5. **Gu runtime coverage**: số lượng op/trigger per Gu trong `Gu.csv` và `Status.csv` quyết định khối lượng cần viết handler; xem report PR-17 trước khi ước lượng hàng nghìn Gu.
6. **Persistence last-safe spawn**: người đang legacy có X/Y rất lớn; chuyển sang map MMO bằng mapping vùng/portal, không dùng trực tiếp số tọa độ làm pixel.
7. **Party loot in crash**: test exactly-once ledger bắt buộc trước Alpha có kinh tế thật.
8. **Player authority cross-tab**: xác định session lease và tab spectator, tránh bot nhân đôi control packet.
9. **Content tooling**: một bản đồ Tiled duy nhất xuất client JSON và server nav/collision có hash thống nhất, không cập nhật hai nơi bằng tay.

---

# 15. Rollout plan, observability and rollback

### Canary 0 — local
- Fake auth/DB fixtures; unit Go/TS; Phaser smoke + seed map; 2 bots.

### Canary 1 — staging isolated
- Dedicated sandbox zone; 5–10 accounts nội bộ; fake currency/items hoặc snapshot dataset; feature flag on only for allowlist; old world unaffected.

### Canary 2 — opt-in limited
- 30 user concurrent target, inventory/live progression có ledger; heatmap AI/VFX; daily backup, 24h soak; explicit feedback tool and rollback.

### Canary 3 — MMO default
- Chỉ thực hiện khi cutover/rollback rehearsal pass; stop legacy ownership, migrate safe spawns; kill old WS `world.move_path` usage from MMO clients; observe 48h.

### Hard rollback triggers (định lượng sẽ chuẩn hóa sau benchmark)
- Duplicated grant hoặc xung đột authoritative owner: **rollback ngay**.
- p99 zone tick >50ms kéo dài theo chính sách cảnh báo; severe RTT/packet backpressure; game FPS tụt dưới ngưỡng trên máy tham chiếu; lỗi map collider/death state ảnh hưởng nhiều người.
- Cách rollback: close new admits -> drain zone -> flush ledger/outbox -> safe-spawn / mode flag switch -> restore frontend route -> fix root cause; không chạy `down migration` mất dữ liệu.

**Dashboards tối thiểu:** `zone_tick_duration_ms`, `zone_entity_count`, `zone_players`, `zone_ai_cost_ms`, `zone_effect_events_processed`, `ws_outbound_bytes`, `ws_queue_depth`, `ws_snapshot_drops`, `ws_reconnects`, `client_frame_ms`, `client_prediction_error`, `loot_grant_dedupe`, `zone_transfer_failed`.

---

# 16. Anti-patterns tuyệt đối tránh

- Không viết lại toàn bộ game thành Phaser một lần, vì Go/domain/DB đang có giá trị.
- Không dùng `match` PvP đấu trường như `World Runtime` duy nhất; đây là bounded prototype với full roster snapshot và thiếu nhiều effect semantics.
- Không “dịch” toàn bộ turn-based effect bằng hằng số `StatusTicksPerTurn`.
- Không để React Zustand làm cache mọi entity world snapshot 20Hz.
- Không gọi `ExecuteMovePlayerPath` hoặc Postgres transaction cho mỗi bước/chuyển động.
- Không xử lý click-to-move bằng cách client tự cập nhật server `x,y`.
- Không chuyển sang binary protocol, ECS framework hoặc nhiều microservice trước khi baseline JSON/single-process chứng minh bottleneck.
- Không hiện secret của hidden passives qua payload cho client khác; chỉ gửi VFX/telegraph hợp lý.
- Không xóa Pixi/old handlers ngay; tắt ownership từng mode, thực hiện migration an toàn rồi mới cleanup.
- Không tuyên bố “MMO 100 người mượt” khi chưa có tải giả lập tương đương hành vi thật.

---

# 17. Đầu việc ngay ngày mai: sprint Zero -> first playable

**Thứ tự commit bắt buộc cho một dev/agent:**

1. PR-01: chạy baseline tests, lập backup + flow audit.
2. PR-02 + PR-03: feature flags và Phaser 4.2.1 smoke.
3. PR-04 + PR-05: mount Phaser cạnh legacy, chốt protocol fixtures.
4. PR-09 + PR-10: dựng Go world loop/zone manager và bots headless.
5. PR-12 + PR-13: map collision + movement hai chế độ server-authoritative.
6. PR-06 + PR-07 + PR-08: input/game WS/prediction đúng chuẩn và nối vòng end-to-end.
7. PR-14 + PR-15 + PR-16: AOI, quái, cast/projectile/damage realtime trên map.
8. PR-17–20: sáu Cổ Trùng, phân loại toàn catalog, hybrid aim & hidden passive.
9. PR-21–22: thêm asset, animation, camera, feedback; không được đánh đổi correctness.
10. PR-23–29: login, loadout, domain UI, loot, reconnect, portal/party và slice E2E.

**Release tối thiểu đáng chơi** không phải bản demo Phaser chỉ di chuyển sprite: phải có 2 người cùng thấy nhau, 3 quái AI, 6 Cổ Trùng 3 mode targeting, 1 nội tại ẩn, đánh và nhặt loot trong cùng bản đồ, save/reconnect.

---

# 18. Handoff prompt mẫu cho coding agent (dùng từng PR, không đưa toàn bộ 36 PR vào 1 lần)

> Bạn đang làm GuWorld MMO Realtime. Đọc `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` mục ADR, đường dẫn source và mô tả PR-XX. Chỉ triển khai phạm vi PR-XX; không sửa/loại legacy gameplay ngoài phạm vi. Phaser cố định 4.2.1, frontend Next.js/React, backend Go authoritative; top-down 2.5D; WASD + click-to-move; targeting TARGET/DIRECTION/GROUND/SELF. Thêm tests theo acceptance criteria. Trước khi code: liệt kê file đang tồn tại và file sẽ tạo, chốt hợp đồng giữa server/client. Sau khi code: chạy tests/typecheck/build phù hợp, ghi logs, các trường hợp chưa kiểm thử, migration/rollback impact và checklist nghiệm thu. Không tự nhận test pass nếu không chạy được.

---

## Verification / limitations của bản plan

- Đã đối chiếu các đường dẫn trọng yếu và đọc các luồng WS, movement, arena, effect compiler trong **ZIP**. **Không** thay đổi source trong ZIP, không chạy game end-to-end và chưa có kết quả benchmark thực tế.
- Source có dependencies, DB schema, middleware, game loop hiện hành; việc tái sử dụng từng hàm phải kiểm tra signature/trạng thái transaction cụ thể trong PR. Các file `backend/internal/realtime/*` và `frontend/src/game/*` là **thiết kế mới**, không được hiểu nhầm đã tồn tại.
- Các mục tiêu số người chơi, throughput và thời gian phát triển là **định hướng cần xác nhận bằng phép đo**.

**Kết luận:** Giữ GuWorld như một RPG product đã có dữ liệu và kinh tế; thay **gameplay runtime** thành một Go authoritative MMO world và thay **world visual/input** bằng Phaser 4.2.1. Tập trung hiện thực hóa movement → combat → Cổ Trùng → loot/persistence → AOI/zone/cutover; UI/asset làm song song nhưng không thay thế các gate correctness.
