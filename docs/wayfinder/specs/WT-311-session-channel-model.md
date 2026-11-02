# WT-311 — Mô hình Session & Kênh Realtime trên WebTransport

> Ticket: [Chốt mô hình session & kênh realtime](https://github.com/VipPro67/GuWorld/issues/311) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02

## 1. Quyết định cốt lõi

1. **Một Session WebTransport duy nhất mỗi người chơi (`1 Session / Player`)**:
   - Bỏ mô hình 2 kết nối song song (`purpose=world` + `purpose=ui`) của WebSocket cũ.
   - Bỏ tham số query `purpose`. Endpoint duy nhất:
     `https://rt.cogioi.io.vn/api/wt?token=<jwt>&tab_id=<uuid>`
   - Mọi luồng realtime (Phaser world 20 Hz + UI pubsub/RPC) đi chung trên 1 QUIC connection, tách bằng **Bidirectional Streams** và **QUIC Datagrams** bên trong session.

2. **Chính sách Single-Active-Tab (`1 Tab / Player`) & Takeover theo `tab_id`**:
   - Mỗi `playerID` chỉ có **1 active WebTransport session** tại một thời điểm trên `Hub`.
   - `Hub.clients` và `Hub.channels[ch]` đều trỏ về đúng `Session` đang active của `playerID` (xoá bỏ lỗi 2 socket cùng `playerID` đè kênh của nhau trong `backend/internal/ws/hub.go:54`).
   - Khi một session mới của cùng `playerID` hoàn tất handshake:
     - **Khác `tab_id` (Mở tab mới hoặc thiết bị mới — Active Takeover)**:
       - Server thu hồi character lease từ session cũ, chuyển quyền cho session mới, và gọi `oldSession.CloseWithError(4001, "session_replaced")`.
       - Client ở tab cũ nhận mã đóng `4001` phải **dừng hẳn auto-reconnect** và bật trạng thái `game:tab-replaced` (khóa tab cũ).
     - **Cùng `tab_id` (Cùng tab rớt mạng WiFi↔4G hoặc F5 nối lại — Same-Tab Supersede)**:
       - Server đóng ngầm session zombie cũ với `oldSession.CloseWithError(4000, "superseded_same_tab")` và chuyển giao nguyên tử (atomic) character lease sang session mới.
       - Tuyệt đối không trả `409 realtime_character_already_owned` khi cùng `tab_id` nối lại.

3. **Connection Limiter & Chống Flood trên UDP/443**:
   - Thay giới hạn cứng `3 connection/player` + `10/IP` (`backend/internal/ws/connection_limiter.go`) bằng:
     - **Theo Player**: Tối đa **2 session đồng thời** trong lúc chuyển giao; nếu vượt thì **evict oldest** (đóng session cũ nhất ngay lập tức thay vì từ chối session mới bằng `429`), kết hợp **rate-limit tốc độ tạo session mới: tối đa 6 handshake / 10 giây / player** (vượt ngưỡng mới trả `429`).
     - **Theo IP**: Nâng trần lên **30 active sessions / IP** (để không chặn nhầm người chơi chung IP CGNAT 4G Việt Nam hoặc quán net).

---

## 2. Sơ đồ Session & Phân luồng Stream / Datagram

Khi client mở `WebTransport` session, mỗi **Bidirectional Stream** do client khởi tạo sẽ gửi **1 byte đầu tiên (`StreamType`)** để định danh mục đích:

```text
WebTransport Session (UDP/443 — 1 QUIC Connection / Player)
│
├── BiDi Stream #1 [StreamType = 0x01: WORLD STREAM] (Binary length-prefixed frames)
│     ├── Client → Server: world.join, world.leave, world.cast, world.attack,
│     │                    world.capture, world.companion_stance
│     └── Server → Client: world.welcome, world.reject, world.transfer,
│                          world.events, và world.snapshot (khi > maxDatagramSize)
│
├── BiDi Stream #2 [StreamType = 0x02: UI / RPC STREAM] (NDJSON — 1 JSON object / line)
│     ├── Client → Server: subscribe, unsubscribe, ping,
│     │                    command (chat.send, world.cells_rect, world.drops, world.objects, match.input)
│     └── Server → Client: event (chat:*, trade, player, refine, notification,
│                          death_arena, martial_arena, catalog), response, error, pong
│
└── QUIC Datagrams [Unreliable, unordered, ≤ 1024 B payload]
      ├── Client → Server: world.input (vector di chuyển), world.move_target
      └── Server → Client: world.snapshot (delta/baseline vừa khung ≤ maxDatagramSize)
```

---

## 3. Bảng ánh xạ toàn bộ kênh cũ sang kiến trúc mới

| Kênh / Message hiện tại (WS) | Tầng vận chuyển mới (WT) | Framing (v1) | Ghi chú |
|---|---|---|---|
| `world.join`, `world.welcome`, `world.leave`, `world.transfer`, `world.reject` | BiDi Stream `0x01` (`world`) | Binary帧 | Cần đảm bảo thứ tự & độ tin cậy tuyệt đối |
| `world.cast`, `world.attack`, `world.capture`, `world.companion_stance` | BiDi Stream `0x01` (`world`) | Binary帧 | Tần suất thấp, không được phép rớt gói |
| `world.events` (batch theo tick) | BiDi Stream `0x01` (`world`) | Binary帧 | Giữ gộp batch mỗi tick 50 ms của zone sim |
| `world.input`, `world.move_target` | **QUIC Datagram** | Binary gọn | Kèm `seq` tăng dần; gói mới thay thế gói cũ |
| `world.snapshot` (≤ `maxDatagramSize` 1024 B) | **QUIC Datagram** | Binary gọn | Kèm `tick`, `ack_seq`, `baseline_tick` |
| `world.snapshot` (> `maxDatagramSize` 1024 B) | BiDi Stream `0x01` (`world`) | Binary帧 | Đường tràn tự động để không bị browser drop im lặng |
| `player`, `refine`, `trade`, `notification`, `catalog` | BiDi Stream `0x02` (`ui`) | NDJSON | Giữ nguyên cấu trúc payload cho Zustand slices |
| `chat:world`, `chat:general`, `chat:region:*`, `chat:sect:*`, `chat:private:*` | BiDi Stream `0x02` (`ui`) | NDJSON | Tần suất thấp; tách stream nên không gây HoL blocking lên `world` |
| `death_arena`, `martial_arena`, `match.input` | BiDi Stream `0x02` (`ui`) | NDJSON | Giữ nguyên luồng sự kiện đấu trường |
| RPC `chat.send`, `world.cells_rect`, `world.drops`, `world.objects` | BiDi Stream `0x02` (`ui`) | NDJSON | Request/response khớp qua `request_id` |
| `world:pos`, `world:cell`, `world:cells`, `world:combat` | **XOÁ BỎ** | — | Kênh thừa từ grid-world cũ (`useGameStore.ts:1944-1986`), hiện 0 listener |

---

## 4. Mã đóng Session (WebTransport Application Error Codes)

| Code | Tên | Ý nghĩa | Hành vi phía Client |
|---|---|---|---|
| `0` | `NO_ERROR` | Đóng chủ động bình thường (logout / rời game) | Không reconnect |
| `4000` | `SUPERSEDED_SAME_TAB` | Session cũ bị thay thế bởi kết nối mới của **cùng `tab_id`** | Bỏ qua sự kiện đóng của instance socket cũ |
| `4001` | `SESSION_REPLACED` | Bị tab khác hoặc thiết bị khác (`tab_id` khác) tiếp quản | **Dừng hẳn reconnect**, phát `game:tab-replaced`, hiện modal khoá tab |
| `4002` | `INVALID_WELCOME_OR_PROTOCOL` | Lỗi phiên bản giao thức / dữ liệu chào | Báo lỗi phiên bản, yêu cầu tải lại trang |
| `4003` | `JOIN_TIMEOUT` | Quá hạn `world.join` (10 s) | Cho phép retry với backoff |
| `4008` | `HANDSHAKE_RATE_LIMITED` | Vượt quá 6 lần tạo session / 10 s | Backoff chậm trước khi thử lại |
| `4010` | `SERVER_DRAINING` | Server đang shutdown/deploy, không nhận session mới | Retry với backoff (không phải tab-replaced) |
