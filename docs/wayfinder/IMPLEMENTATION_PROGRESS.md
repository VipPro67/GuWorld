# Tiến độ thi công WebTransport — 2026-11-02

> Mục tiêu: thay WebSocket bằng WebTransport cho realtime, bỏ Caddy, TLS/QUIC trong Go.
> Map quyết định: [#303](https://github.com/VipPro67/GuWorld/issues/303) (đã đóng). Spec: `docs/wayfinder/specs/WT-*.md`.
> File này là điểm tiếp tục cho phiên sau. Cập nhật lần cuối: 2026-11-02.

## TL;DR

- **World socket đã chạy được trên WebTransport** (join, welcome, input, snapshot, cast/attack/capture/stance, spawn, resume, combat events), bật bằng `WT_ENABLED=true` ở backend + `NEXT_PUBLIC_REALTIME_TRANSPORT=wt` ở frontend.
- **WS vẫn còn** cho UI channels (chat/trade/notify/refine/arena/catalog) và vẫn là transport mặc định. Chưa xoá WS.
- Còn **4 bước** để xoá hẳn WS: hub sinks + UIStream → shared RealtimeClient → RPC UI → cutover/xoá WS.
- Toàn bộ test xanh: backend `go test ./...`; frontend `npm test` **203 pass**, `tsc`, lint, build.

## Commit đã hoàn thành

| Commit | Nội dung |
|---|---|
| `a105f38e` | Baseline metrics stall-aware (`stalls`, p99/max/rate), rate limit theo player |
| `48d59d7f` | Codec v2 Go + TS + golden vector chung |
| `1a162140` | Resume session (grace 15/30 s, takeover, entity suspended) |
| `9fb637b3` | `cmd/wtspike` (prototype đo WT/WS) + pin quic-go v0.62/webtransport v0.13/certmagic |
| `d513da6e` | Client seed `seq` từ `welcome.ack_seq` sau resume |
| `0c3c5328` | Codec `WELCOME` + `SnapshotFull` (Go+TS) |
| `11a84164` | **WT server** `internal/realtime/wttransport` + `/api/wt` + `WT_ENABLED` |
| `10f0fe08` | Baseline v2 mang entity state (vị trí/HP/mana/revision) |
| `8b59ccba` | **WT client** `WorldTransport` + interface `WorldTransportLike`, chọn qua env |
| `3acc5ccf` | Forward combat events qua WT (enum 25 loại + payload JSON) |
| `31240f9c` | Docs event stream + phạm vi còn lại |

Ngoài ra có các commit của người dùng xen kẽ (assets/caching, WT-316…) — không đụng.

## Kiến trúc hiện tại

```
Backend (Go)
├── :8080 plain HTTP (mặc định, WS + REST)
└── WT_ENABLED=true
    └── WT_ADDR=:8443 : http.Server TLS (REST + /api/ws wss)
                        webtransport.Server UDP (CONNECT /api/wt)
                        └── internal/realtime/wttransport → gateway.Admission
                            ├── world stream (binary v2)
                            ├── datagram input/delta
                            └── events (Op.Events)

Frontend
├── NEXT_PUBLIC_REALTIME_TRANSPORT=ws (default) → WorldSocket (v1 JSON/WS)
└── =wt → WorldTransport (v2 binary/WebTransport), cùng interface WorldTransportLike
    └── WorldRuntime không đổi logic gameplay
```

- Env mới: `WT_ENABLED`, `WT_ADDR`, `TLS_CERT_FILE`, `TLS_KEY_FILE`; frontend `NEXT_PUBLIC_REALTIME_TRANSPORT`, `NEXT_PUBLIC_REALTIME_URL`.
- Route WT: `CONNECT /api/wt` (Auth, `?token=`, `?tab_id=`), `CheckOrigin` dùng chung `CORS_ALLOWED_ORIGINS`.
- Codec: `backend/internal/realtime/protocol/v2` ↔ `frontend/src/game/network/v2/Codec.ts` (golden vector `82d204...`).

## Đã kiểm chứng

- Go: `go test ./...` pass; `go test -race ./internal/realtime/wttransport ./internal/realtime/gateway ./internal/realtime/sim ./internal/ws` pass.
- Integration WT thật (client webtransport-go ↔ server): handshake → join → welcome baseline → input datagram → snapshot delta → **events frame**.
- Frontend: `npm test` 203 pass (gồm join/baseline/delta/events WT), `npx tsc --noEmit`, `npm run lint`, `npm run build`.
- Resume: test cùng tab giữ entity, hết grace nhả lease, tab khác takeover, suspended không ăn damage/không bị AI nhắm.

## Việc cần làm tiếp (theo thứ tự)

### 1. Hub sinks + UI stream (server) — bắt buộc để bỏ WS
Mục tiêu: message channel hiện broadcast tới `*ws.Client` cũng tới sink WT.

- `backend/internal/ws/hub.go`
  - Thêm `type MessageSink interface{ Deliver([]byte) }` (hoặc `func([]byte)`).
  - `Hub.sinks map[ChannelName]map[uuid.UUID]MessageSink`.
  - Trong `run()` case broadcast: sau vòng `subs`, gửi payload cho sink (cùng Exclude/Targets).
  - `SubscribeSink(playerID, channel, sink)` / `UnsubscribeSink(playerID, channel)`.
  - Xoá sink khi client disconnect? UI sink tự unsubscribe; thêm `UnsubscribeAllSinks(playerID)`.
- `backend/internal/ws/sharded_hub.go`: route `SubscribeSink`/`UnsubscribeSink` theo channel shard (dùng hàm hash hiện có).
- `backend/internal/realtime/wttransport/ui.go` (mới):
  - Nhận bidi stream có byte đầu `0x02` (`StreamUI`).
  - Đọc NDJSON từng dòng: `{type:"subscribe"|"unsubscribe"|"ping"|"command", channel, data, request_id}`.
  - Subscribe → `shub.SubscribeSink(playerID, channel, sink)`; sink ghi `{"type":"event","channel":"<ch>","data":<payload-json>}\n`.
  - Auto-subscribe giống WS UI: `player:<id>`, `refine:<id>`, `trade:<id>`, `notif:<id>`, `combat:<id>`, `chat:world`, `chat:general`, sect channel.
  - `command`: v1 trả `{"type":"error","request_id":...,"message":"use HTTP"}`; client fallback HTTP.
  - `ping` → `{"type":"pong"}`.
- `wttransport.Server.Handle`: nếu stream đầu tiên là `0x02` thay vì `0x01`, chạy UI loop (hiện chỉ chấp nhận world stream). Có thể tách `HandleUI`.
- Test: fake hub/sink + NDJSON, đảm bảo subscribe nhận event và unsubscribe dừng.

**Xong khi:** chat/trade/notify nhận qua WT UI stream trong integration test; không cần `/api/ws` cho UI.

### 2. Shared `RealtimeClient` (frontend) — đúng #311 (1 session/tab)
Vấn đề hiện tại: WT world mở session riêng; UI vẫn WS → 2 kết nối/tab, trái #311.

- Tạo `frontend/src/lib/realtime/RealtimeClient.ts`:
  - singleton, ref-count `acquireWorld()` / `acquireUI()`, grace 30 s.
  - mở 1 `WebTransport` session, tạo world stream (`0x01`) + ui stream (`0x02`).
  - `WorldTransport` nhận session/stream từ client thay vì tự `new WebTransport`.
  - `wsManager` khi `NEXT_PUBLIC_REALTIME_TRANSPORT=wt` dùng UI stream; giữ API `subscribe/request/isConnected/acquireConnection/onOpen`.
  - `request`: gửi `command` + `request_id`; nếu server trả `error use HTTP` thì throw để caller fallback HTTP hiện có.
- Duplicate tab (BroadcastChannel rotate `tab_id`) theo #318 §5.
- Test: fake transport, đảm bảo 1 session, 2 stream, ref-count/grace.

**Xong khi:** mở game 1 tab chỉ có 1 session WT; UI + world dùng chung; `wsManager` không còn tạo WebSocket khi mode wt.

### 3. RPC UI qua WT hoặc HTTP
- `chat.send`, `world.cells_rect`, `world.drops`, `world.objects`, `match.input`: chọn HTTP fallback (đã có sẵn ở store) hoặc implement command trên UI stream.
- Đảm bảo `chatSlice.sendMessage`, `useWorldStore.fetchNearbyCells`, `inventorySlice` hoạt động khi mode wt (test store).
- Notification/refine đã auto-subscribe qua sink.

### 4. Cutover & xoá WS
- Deploy cert thật (`rt.cogioi.io.vn` hoặc `spike`) + `WT_ENABLED=true` + env frontend.
- Smoke: join, chạy, cast, damage, chat, trade, notify, resume 3 s/10 s, mở tab thứ 2.
- Xoá `/api/ws` route, `internal/ws` code (hoặc giữ hub cho sink rồi bỏ dần), `frontend/src/lib/websocket.ts` WebSocket internals.
- Cập nhật `Caddyfile` (xoá), compose, docs (`README`, `DEVELOPMENT`, `API`), `.env.example`.
- Theo kế hoạch cutover `docs/wayfinder/specs/WT-309-rollout-rollback.md`.

## Cách chạy thử WT hôm nay

Backend:
```bash
cd backend/cmd/wtspike && ./gen-cert.sh /tmp/wtcert && cd ../..
WT_ENABLED=true WT_ADDR=:8443 \
TLS_CERT_FILE=/tmp/wtcert/srv.crt TLS_KEY_FILE=/tmp/wtcert/srv.key \
go run ./cmd/server
```
Frontend `.env.local`:
```bash
NEXT_PUBLIC_REALTIME_TRANSPORT=wt
NEXT_PUBLIC_REALTIME_URL=https://localhost:8443/api/wt
```
Lưu ý: Chrome yêu cầu cert tin cậy **cho QUIC**; cert tự ký bị `ERR_QUIC_CERT_ROOT_NOT_KNOWN`. Dùng cert Let's Encrypt thật (xem `backend/cmd/wtspike/README.md` mục Option B) hoặc policy `CACertificates` (Chrome 132+).

## Giới hạn/known issues

- WT client chưa map projectiles/transfer (server chưa gửi transfer qua WT; projectile bị bỏ ở client).
- `WorldTransport` chưa gửi PING/PONG (RTT đang lấy từ ack input; server đã hỗ trợ PONG).
- Server WT chưa gửi `TRANSFER`; edge transfer hiện dựa vào reconnect/resume.
- UI channels vẫn WS; `Hub` chỉ giao cho `*Client`.
- Chưa có graceful drain `4010`; shutdown hiện đóng TCP rồi `wt.Close()`.
- Chưa bật `WT_ENABLED` production; chưa có cert `rt.cogioi.io.vn`.
- Không claim scale/50 người; guard vẫn `MMO_MAX_PLAYERS_PER_ZONE` (default 5).

## Checklist trước khi tuyên bố “đã thay WS”

- [ ] UI stream WT nhận chat/trade/notify/refine/arena/catalog.
- [ ] 1 session/tab (world + ui), tab thứ 2 takeover.
- [ ] RPC UI hoạt động (WT hoặc HTTP fallback).
- [ ] Projects/transfer đầy đủ (hoặc cắt khỏi gameplay).
- [ ] Resume 3 s/10 s p95 < 2 s.
- [ ] `/api/ws` trả 404/410; không còn `new WebSocket(` trong `frontend/src`.
- [ ] Cert `rt` DNS-only + `WT_ENABLED=true` production; smoke WiFi/4G.
- [ ] Docs + compose + `.env.example` cập nhật; Caddyfile xoá.
