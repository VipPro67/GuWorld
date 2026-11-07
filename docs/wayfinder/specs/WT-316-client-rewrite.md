# WT-316 — Client Rewrite sang WebTransport

> Ticket: [Spec: client rewrite sang WebTransport](https://github.com/VipPro67/GuWorld/issues/316) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Phụ thuộc: [Chốt mô hình session & kênh realtime](https://github.com/VipPro67/GuWorld/issues/311), [Spec: wire protocol v2 (binary, datagram, versioning)](https://github.com/VipPro67/GuWorld/issues/313)
> Phạm vi: toàn bộ phần client (Phaser 4 / Next 16). Backend WT do các spec khác chốt.

## 0. Hiện trạng phải thay

| Vấn đề | Bằng chứng |
|---|---|
| 2 socket/tab (`wsManager` UI + `WorldSocket` world) | `frontend/src/lib/websocket.ts`, `frontend/src/game/network/WorldSocket.ts` |
| JSON text, mỗi frame 1 dòng | `WorldSocket.handleRawMessage` split `\n` |
| Replay chết | `SelfPredictor.recordInput()` có nhưng `WorldRuntime` không gọi; `pending` luôn rỗng |
| Reconcile cố định | `softReconcile(..., 2, 0.08)` — 8%/snapshot, không phụ thuộc tần số |
| Nội suy cố định | `INTERPOLATION_DELAY_TICKS = 2`, không extrapolate |
| Retry rời rạc | `JOIN_RETRYABLE_CODES` / `STALE_WORLD_CONNECTION_MS` trong `WorldRuntime` |
| `wsManager` là WebSocket impl | `WebSocketManager` + ping/pong + reconnect riêng |

## 1. Sở hữu session: một `RealtimeClient` ref-count / tab

```
frontend/src/lib/realtime/RealtimeClient.ts   (singleton module)
├── acquireUI(): () => void        // store gọi 1 lần khi initializeSession
├── acquireWorld(): () => void     // WorldRuntime gọi khi connect
├── world: WorldStream | null      // mở khi acquireWorld
├── ui: UIStream | null            // mở khi acquireUI
└── session lifecycle + backoff + close-code state machine
```

- Một `WebTransport` session/tab, `?token=<jwt>&tab_id=<uuid>` (cookie không được gửi — #306).
- Session mở khi refcount đầu tiên > 0; đóng sau khi **cả hai** release và hết grace **30 s** (giữ hành vi `releaseConnection` hiện tại).
- `WorldRuntime.disconnect()` chỉ đóng **world stream** (gửi `LEAVE`), **không** đóng session UI. Unmount Phaser (đổi panel/resize) không làm rớt chat/notify.
- Logout / `4001 session_replaced` → đóng session ngay, bỏ grace.

## 2. Chiến lược adapter: giữ public API, thay ruột

| File hiện tại | Sau rewrite |
|---|---|
| `game/network/WorldSocket.ts` | Adapter mở **world stream**; giữ nguyên `connect/sendInput/sendMoveTarget/sendCast/sendBasicAttack/sendCapture/sendCompanionStance/onSnapshot/onEvents/onReject/onTransfer/onResyncRequired/onError/disconnect/getMetrics/isJoined` |
| `lib/websocket.ts` (`wsManager`) | Adapter mở **ui stream**; giữ `subscribe/request/command/send/isConnected/acquireConnection/onOpen/setOnTokenRefresh/disconnect`; export `wsManager` giữ nguyên |
| Call-site store/components | **Không đổi** trong pha 1 |

Nội bộ xoá hẳn: `WebSocket` instance, `onmessage` JSON split, `pingTimer`/`pongTimeout`, reconnect timer của `WebSocketManager`, 4 subscription chết `world:pos` / `world:cell` / `world:cells` / `world:combat` (theo #311).

**Pha 2 (sau khi WT chạy ≥ 1 tuần, ngoài ticket này):** đổi tên `wsManager` → `realtimeManager`, xoá lớp adapter, sửa ~15 call-site, xoá `WSMessage`/`WebSocketRequestError` cũ nếu đã thay bằng type mới. Ghi ở fog của map.

### 2.1 Ánh xạ lời gọi

| Call-site hiện tại | Cơ chế mới |
|---|---|
| `wsManager.subscribe(ch, cb)` | `UIStream` gửi `subscribe` frame NDJSON; map kênh giữ nguyên |
| `wsManager.request(ch, payload)` | `request_id` trong NDJSON; resolve từ `response` |
| `wsManager.acquireConnection(token)` | `RealtimeClient.acquireUI()` (token lấy từ `localStorage`, không truyền) |
| `wsManager.onOpen({reconnected})` | `RealtimeClient` phát `open` sau khi session `ready` + UI stream mở; `reconnected` = lần mở thứ ≥ 2 |
| `WorldSocket.connect(join)` | `RealtimeClient.acquireWorld()` → mở world stream → gửi `JOIN` (binary) → chờ `WELCOME` |
| `WorldSocket.isJoined()` | World stream `ready` + đã nhận `WELCOME` |
| `WorldSocket.disconnect()` | Gửi `LEAVE`, đóng world stream, release world ref |

## 3. Decode binary

- Codec TS nằm ở `frontend/src/game/network/v2/` (codec + golden vector chung với Go — #313 §8).
- Decoder trả về **đúng các type hiện có** (`WorldSnapshotPayload`, `WorldWelcomePayload`, `WorldEvent`…) để `SnapshotBuffer`, `WorldRuntime`, HUD và UI không phải đổi kiểu.
- Datagram `SNAPSHOT_DELTA` → dựng `WorldSnapshotPayload` với `entities` chỉ chứa delta, giữ `baseline_id/base_tick`. Baseline/tràn khung đến từ stream.
- `EVENTS`: binary header + `payload_json` → `JSON.parse` payload thành `WorldEvent.payload`; `type_id` tra bảng enum, `0` = unknown (giữ `type` string từ payload nếu có).
- Không còn `JSON.parse` cho snapshot/input/join.
- **Thiếu datagram support** (Safari `reliability` chỉ stream): server đã có đường tràn stream cho snapshot; client chỉ cần **không** mở datagram writer và báo `datagramsAvailable=false` cho `RealtimeMetrics` (đo tỉ lệ người chơi không có datagram).

## 4. Prediction + Input replay (thay `SelfPredictor`/`softReconcile`)

### 4.1 Mô hình segment

```
pending: { seq, move, dt }[]      // tối đa 64, tổng dt tối đa 250 ms
open: { move, dt }                 // segment đang tích luỹ
```

- `advance(move, dt)` mỗi frame: cộng dồn `open.dt += dt` khi vector khác 0, đồng thời áp vào vị trí logic.
- Khi gửi input thành công (`seq`): chốt `open` thành `{seq, move: lastSentVector, dt: min(open.dt, 0.1)}`, đẩy vào `pending`, mở segment mới. (Input gửi 10 Hz → mỗi segment ~100 ms.)
- Click-to-move đi qua `sendMoveTarget` (reliable, #313) — vẫn ghi segment theo vector per-frame đang dùng; `MOVE_TARGET_REJECTED` → xoá `pending` + snap.

### 4.2 Reconcile mỗi snapshot

1. `ack = snapshot.ack_seq`; bỏ mọi segment `seq <= ack`.
2. `target = serverPos`; replay `pending` bằng `predictionSpeed` (tốc độ server quan sát, đã có trong runtime).
3. `renderOffset += (oldLogicPos − target)` (offset vector).
4. `logicPos = target`.
5. Nếu `|offset| > 3 unit`, hoặc vừa `world.transfer`/respawn: `renderOffset = 0` (hard snap).
6. `getEntities()` trả `logicPos + renderOffset`.

Offset decay trong `update(dt)`:

```
renderOffset *= exp(-dt / 0.1)        // τ = 100 ms, độc lập tần số snapshot/frame
if (|renderOffset| < 0.01) renderOffset = 0
```

- Không còn `softReconcile` blend 8%; `hardSnaps` trong telemetry #308 đếm theo ngưỡng 3 unit + decay.
- Tổng `pending.dt > 250 ms` (mạng mất > 250 ms): bỏ segment cũ nhất, hạ ngưỡng snap còn 1.5 unit.

## 5. Nội suy entity khác (thay `RemoteInterpolator`)

- `RemoteInterpolator` giữ API `push(tick, entities)` / `positionAt(id, renderTick)`, thêm extrapolation có giới hạn.
- Buffer động: `renderDelayTicks = clamp(1 + ceil(jitterP95 / tickIntervalMs), 1, 3)`, lấy `jitterP95` từ `RealtimeMetrics` (#308). 20 Hz → 50–150 ms; 60 Hz → 17–50 ms.
- Khi `renderTick > last.tick`: extrapolate theo vector 2 sample cuối, tối đa **1.5 tick**, sau đó giữ nguyên.
- Khi sample mới về sau extrapolation: blend 80 ms về vị trí nội suy thật (tránh giật ngược).
- **Self không extrapolate** (đã có replay); projectiles giữ nội suy tuyến tính như hiện tại.

## 6. Reconnect state machine

| Sự kiện / mã | Hành vi |
|---|---|
| `4001 session_replaced` | Dừng hẳn; phát `game:tab-replaced`; hiện copy "Phiên chơi này đã được thay thế bởi tab mới. Hãy đóng tab này." |
| `4002 protocol_version_mismatch` | Dừng; hiện "Bản game đã cũ. Hãy tải lại trang." |
| `4000 superseded_same_tab` | Im lặng, không retry (instance cũ tự chết) |
| `4003 join_timeout` | Retry trong cùng state machine |
| `4008 handshake_rate_limited` | Retry nhánh chậm (base 5 s) |
| Mạng đứt / `transport.closed` không mã / watchdog 3 s không snapshot | Backoff: `300ms × 1.6^n`, cap **5 s**, jitter ±20 %, tối đa **10 lần**, sau đó overlay "Mất kết nối" + nút thử lại |
| `typeof WebTransport === "undefined"` | Không retry; thuộc #312 |

- Vào lại: mở world stream mới → `JOIN` → nhận `WELCOME` baseline → reset `seq = 0`. Hành vi **resume** (giữ entity/lease, token) do [Spec: fast reconnect & session resume khi mạng đổi](https://github.com/VipPro67/GuWorld/issues/318) quyết; spec này chỉ định nghĩa điểm cắm `resume(token)` trong state machine.
- Watchdog stale 3 s giữ lại (datagram/stream đều có thể im lặng), nhưng reset khi có **bất kỳ** frame world nào.

## 7. Kế hoạch file

### Tạo mới
- `frontend/src/lib/realtime/RealtimeClient.ts` — session, refcount, backoff, close-code.
- `frontend/src/lib/realtime/WorldStream.ts` — frame reader/writer binary + datagram.
- `frontend/src/lib/realtime/UIStream.ts` — NDJSON subscribe/request/command.
- `frontend/src/game/network/v2/` — codec TS + golden vector (dùng chung với Go).
- Test: `RealtimeClient.test.ts`, `WorldStream.test.ts`, `UIStream.test.ts`, `v2/Codec.test.ts`.

### Viết lại
- `game/network/WorldSocket.ts` → adapter (public API giữ nguyên).
- `lib/websocket.ts` → `wsManager` chạy trên `UIStream`; xoá `WebSocketManager` internals.
- `game/network/Prediction.ts` → segment model + replay + offset (thay `softReconcile`).
- `game/core/WorldRuntime.ts` → dùng replay/offset, buffer động, state machine mới.
- `game/network/RemoteInterpolator.ts` → dynamic buffer + extrapolation.
- `game/network/RealtimeMetrics.ts` → nhận byte từ stream, thêm `datagramsAvailable`.

### Xoá
- Toàn bộ code `new WebSocket(...)`, JSON envelope, ping/pong thủ công.
- 4 subscription chết `world:pos/cell/cells/combat` trong `useGameStore.ts`.
- `frontend/src/data/wiki-cache.json` không liên quan (không đụng).

### Không đổi (pha 1)
- Store slices, components, `PhaserWorldHost` API; `useWorldStore`/`inventorySlice` RPC vẫn gọi `wsManager.request`.

## 8. Chiến lược test

| Test | Nội dung |
|---|---|
| Unit — replay | Chuỗi input/ack cố định → vị trí sau reconcile khớp tính tay; mất ack 5 frame vẫn hội tụ; vượt 250 ms thì snap |
| Unit — offset | Sau reconcile, offset decay ~63 % sau 100 ms; offset > 3 unit → 0 |
| Unit — interpolation | 60 Hz samples, mất 2 gói → extrapolate ≤ 1.5 tick rồi dừng; sample mới → blend |
| Unit — state machine | Bảng mã đóng §6 → đúng hành vi retry/dừng, jitter trong biên |
| Unit — codec | Golden vector Go ↔ TS giống 100 % (#313 §8) |
| Integration | Fake `WebTransport` (interface) chạy join → snapshot delta → mất gói → reconnect |
| Thủ công | Chrome/Firefox/Safari 26.4+: join, chạy, chat, đổi WiFi→4G, mở tab thứ 2 |

## 9. Điều kiện hoàn thành khi thi công

- Không còn `new WebSocket(` trong `frontend/src`.
- `WorldSocket` public API và 15 call-site `wsManager` không đổi hành vi.
- Replay/offset/interpolation/state machine có unit test; browser matrix chạy tay đạt.
- Telemetry #308 tiếp tục upload qua session WT, có thêm `datagrams_available`.
- `npm test`, `npm run lint`, `npm run build` pass.

## 10. Rủi ro

- **Token trong query** lộ qua log/proxy — chấp nhận (cookie không dùng được); access log backend phải redact `token` (kiểm tra ở spec ingress #315).
- **Tab background** bị throttle → watchdog 3 s có thể bắn khi tab ẩn; chỉ áp watchdog khi `document.visibilityState === "visible"`, và reconnect ngay khi tab hiện lại nếu quá hạn.
- **Draft drift** — pin version, test browser matrix.
- **Đổi transport làm hỏng UX cũ** — pha 1 giữ nguyên API/call-site để rollback dễ; pha 2 tách riêng.
- **`4000` vs mạng đứt** — không thể phân biệt tuyệt đối nếu frame đóng mất; nếu không có mã thì coi là mạng đứt và retry (an toàn hơn dừng nhầm).
