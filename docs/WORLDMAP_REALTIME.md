# WorldMap Realtime — Thiết kế & Lộ trình

Mục tiêu: WorldMap gần như realtime cho **200–2000 player online** trên **một server**
(Golang + Gin + PostgreSQL), đồng thời dựng nền tảng dễ mở rộng cho **combat thủ công**
thay thế auto combat sau này — mà **không phá vỡ auto combat hiện tại**.

## 1. Hiện trạng (trước khi tối ưu)

| Thành phần | Cách hoạt động hiện tại | Chi phí |
|---|---|---|
| Procedural cells | `GenerateCell` — SHA-256 seed per (zone,x,y), thuần, không DB | rẻ |
| Persistent cells | `map_cells` nạp 1 lần vào RAM (`zoneReg.persistent`) | rẻ |
| Vị trí player | DB `players.x/y`, ghi khi di chuyển | ổn |
| SSE vị trí | `broadcastWorldPos` gửi tới **mọi** client (O(N²)) | **nút thắt** |
| Player gần | `GET /api/explore/players` query SQL mỗi 2s/client | **nút thắt** |
| Presence | `EvaluateAndPersistMana` ghi `last_mana_update` mọi request | **nút thắt** |
| Cells | `cells/rect` tính lại mỗi request dù deterministic | CPU dư thừa |
| Respawn persistent | không reset live (chỉ reset khi restart) | bug tươi bản đồ |

## 2. Nút thắt & cơ chế giải quyết

1. **SSE fan-out toàn cục** → interest management theo vị trí (chỉ gửi cho sub gần mover).
2. **Poll SQL 2s/client** → `PositionIndex` in-memory (spatial grid), poll đọc từ RAM.
3. **Write amplification presence** → throttle `last_mana_update` 60s; index `ts` làm authority "active".
4. **Cells recompute** → LRU cache key `(x,y)`.
5. **Respawn không live** → reset lười khi đọc + push delta event.

## 3. Thiết kế mới

### 3.1 `PositionIndex` — spatial index vị trí player (in-memory)

File: `backend/internal/handlers/world_index.go` (package `handlers`, không tạo layer mới).

```go
type WorldPosEntry struct {
    ID, Username, AvatarURL string
    X, Y int
    Region string
    TS   int64 // activity timestamp (unix ms) — authority cho "active" 15 phút
}

type PositionIndex struct {
    mu      sync.RWMutex
    bucket  int                                // 300
    byID    map[string]*WorldPosEntry
    grid    map[bucketKey]map[string]*WorldPosEntry
}
```

- `Set(e)` — upsert + chuyển bucket khi vị trí đổi.
- `Get(id)`, `Nearby(x, y, r, activeWindow, excludeID)` — quét các bucket chồng lấn vùng, lọc khoảng cách + `TS` còn active.
- `Reconcile(entries)` — thay toàn bộ (dùng bởi sync worker).
- `Len()` — số entry (để endpoint biết khi nào nên fallback SQL).

**Authority / tự hồi phục:** DB vẫn là nguồn đúng cuối cùng. Sync worker
(`StartWorldIndexSyncWorker`) mỗi 30s chạy 1 query
`SELECT ... FROM players WHERE last_mana_update >= NOW() - INTERVAL '16 minutes'`
rồi `Reconcile` — bootstrap lúc start, tự sửa ghost nếu lỡ sót call-site hoặc sau crash.
Các đường đi realtime (move) gọi `Set` ngay (qua `broadcastWorldPos`) để không bị trễ 30s.

### 3.2 SSE interest management

- `subscribeWorldPos(playerID)` thay vì `subscribeWorldPos()`.
- `broadcastWorldPos(evt)`: `Set` mover vào index → lấy `Nearby(mover, WorldPosInterestRadius=300)`
  → chỉ gửi cho các player **có subscription** nằm trong danh sách đó.
- Client **không đổi**: vẫn tự lọc theo `nearbyWindow` (radius investigation của chính nó).
- Subscriber được nạp vị trí vào index lúc connect (1 query) nếu chưa có.

### 3.3 Throttle presence

Trong `EvaluateAndPersistMana`: map `map[uuid]time` + mutex; chỉ ghi `last_mana_update`
nếu cách lần ghi trước ≥ 60s. Cửa sổ "active" ở tầng đọc vẫn 15 phút (index `TS` đủ tươi nhờ
sync worker 30s + Set khi move).

### 3.4 LRU cache cells

`GetNearbyCells` / `GetCellsRect` dùng cache `map[cellKey]resolvedCell` (LRU ~100k, key `x,y`)
vì `ResolveCell` là hàm thuần theo (x,y). Hit cao ở vùng đông người.

### 3.5 Cell delta events + respawn lazy

- SSE events `world.cell.cleared` / `world.cell.respawn` (hub riêng `cellHub`, fan-out toàn cục
  vì ít & nhỏ) phát từ `MarkPersistentCellCleared` và lúc reset respawn.
- `PersistentCellAt`: khi thấy `cleared` mà `time.Since(LastCleared) >= respawn_seconds`
  thì reset `cleared=false` ngay tại chỗ (không cần goroutine) + broadcast respawn.

### 3.6 Nền tảng combat thủ công (Phase 2 — chỉ thiết kế, chưa code)

- `PositionIndex` trở thành index dùng chung cho entity (player, sau này monster/battle).
- Server-authoritative tick loop tái dùng determinism của `simulator/engine.go`; hành động POST
  → validate → queue theo tick; state delta phát qua SSE channel phạm vi đúng bên tham chiến.
- Transport (SSE+POST hiện tại vs WebSocket) để sau, nằm sau interface.

**Interface gợi ý (để tách transport khỏi logic, code sau khi có yêu cầu):**

```go
// PositionedEntity là thứ có vị trí trên bản đồ (player, monster, battle instance).
type PositionedEntity interface {
    ID() string
    Pos() (x, y int)
}

// BattleTransport trừu tượng hoá kênh đồng bộ state chiến đấu.
// phase hiện tại: SSE + POST; sau này có thể thay bằng WebSocket.
type BattleTransport interface {
    // SendState gửi delta/trạng thái cho danh sách người tham chiến.
    SendState(participantIDs []string, state interface{})
    // RecvAction nhận hành động từ client (được validate ở service).
    RecvAction() <-chan BattleAction
}

// BattleInstance chạy tick loop server-authoritative, tái dùng determinism
// của simulator/engine.go. Hành động được queue và resolve theo tick.
type BattleInstance interface {
    Tick(ctx context.Context)
    Enqueue(a BattleAction) error
    Subscribers() []string // dùng chung PositionIndex cho interest/range check
}
```

Range-check & targeting dùng `PositionIndex.Nearby(...)` — O(1) theo bucket, thay vì duyệt toàn bộ.

## 4. Call-site cập nhật index

Các nơi ghi `players.x/y` (đã rà soát, `tribulation_enemies` KHÔNG phải player):

1. `exploration.go` MovePlayer — qua `broadcastWorldPos`.
2. `exploration.go` MovePlayerPath — qua `broadcastWorldPos`.
3. `player.go` onboarding — `worldPosIndex.Set` sau commit.
4. `combat_explore.go` explore node — `Set` sau commit.
5. `combat_retreat.go` retreat — `Set` sau commit.
6. `combat_retreat.go` flee failed — `Set` sau commit.
7. `combat_retreat.go` flee success — `Set` sau commit.
8. `combat_auto.go` defeat → NearestHub — `Set` sau commit.
9. `combat_attack.go` defeat → NearestHub — `Set` sau commit.

(1)(2) là đường realtime chính; còn lại được sync worker 30s bảo hiểm nếu lỡ.

## 5. Lộ trình

- **Phase 1 (backend):** 3.1 → 3.2 → 3.3 → 3.4 → 3.5. Verify: `go test ./...`, `go test -race ./...`.
- **Phase 1.5 (frontend):** nghe `world.cell.*`, giảm poll cells 8s→30s.
- **Phase 2:** worldservice interface + tick loop combat thủ công (khi có yêu cầu).
