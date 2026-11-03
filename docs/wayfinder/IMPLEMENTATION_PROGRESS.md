# Tiến độ thi công WebTransport — 2026-11-03

> Mục tiêu: WebTransport là transport realtime **duy nhất**, bỏ Caddy và WebSocket.
> Map quyết định: [#303](https://github.com/VipPro67/GuWorld/issues/303) (đã đóng). Spec: `docs/wayfinder/specs/WT-*.md`.
> File này là điểm tiếp tục cho phiên sau. Cập nhật lần cuối: 2026-11-03.

## TL;DR

- **WebSocket và Caddy đã bị xoá khỏi code/infra.** Realtime chỉ còn WebTransport:
  - Stream `0x01` (`StreamWorld`, binary v2): join, welcome, input datagram, snapshot delta/full,
    projectiles, transfer (`Op.Transfer`), cast/attack/capture/stance, spawn, resume, combat events,
    PING/PONG datagram, graceful drain `4010 server_draining`.
  - Stream `0x02` (`StreamUI`, NDJSON): auto-subscribe + subscribe/unsubscribe/ping/command cho
    chat/trade/notify/refine/arena/catalog qua `uihub.Hub` / `uihub.ShardedHub` (sink-only).
  - Shared `RealtimeClient` (singleton ref-count `acquireWorld()` / `acquireUI()`, grace 30 s,
    duplicate-tab `BroadcastChannel` rotate `tab_id`, close code `4000`/`4001`/`4002`/`4010`).
  - Browser không có WebTransport → terminal `unsupported` cho cả phiên, không retry; giữ HTTP-only.
- **Không còn `NEXT_PUBLIC_REALTIME_TRANSPORT`, không còn `/api/ws`, không còn `internal/ws`.**
  Redis Pub/Sub vẫn fan-out sự kiện UI vào `internal/uihub`.
- **TLS/QUIC trong Go:** `TLS_ENABLED=true` phục vụ HTTPS (`TLS_ADDR`) + WT (`WT_ADDR`);
  `TLS_ENABLED`/`WT_ENABLED` thiếu cert hoặc realtime runtime ⇒ process dừng (fail fast).
- Test xanh: backend `go test ./...`; frontend `npm test` **215 pass**, `tsc --noEmit`, `eslint`.
- **Còn lại:** deploy cert thật + smoke production theo `docs/PRODUCTION_DEPLOY.md`.

## Commit & hạng mục đã hoàn thành

| Commit / Hạng mục | Nội dung |
|---|---|
| `a105f38e` | Baseline metrics stall-aware (`stalls`, p99/max/rate), rate limit theo player |
| `48d59d7f` | Codec v2 Go + TS + golden vector chung |
| `1a162140` | Resume session (grace 15/30 s, takeover, entity suspended) |
| `9fb637b3` | `cmd/wtspike` (prototype đo WT/WS) + pin quic-go v0.62/webtransport v0.13 |
| `d513da6e` | Client seed `seq` từ `welcome.ack_seq` sau resume |
| `0c3c5328` | Codec `WELCOME` + `SnapshotFull` (Go+TS) |
| `11a84164` | **WT server** `internal/realtime/wttransport` + `/api/wt` + `WT_ENABLED` |
| `10f0fe08` | Baseline v2 mang entity state (vị trí/HP/mana/revision) |
| `8b59ccba` | **WT client** `WorldTransport` + interface `WorldTransportLike` |
| `3acc5ccf` | Forward combat events qua WT (enum 25 loại + payload JSON) |
| `31240f9c` | Docs event stream + phạm vi còn lại |
| `44385ac8` | Giữ tính năng HTTP cơ bản hoạt động khi mất kết nối realtime |
| **Bước 1 (Xong)** | Hub sinks + `wttransport/ui.go` (`0x02` NDJSON) + multiplex nhiều BiDi stream trên 1 WT session |
| **Bước 2 (Xong)** | Shared `RealtimeClient` + `UIStream`, `WorldTransport` & UI manager dùng chung 1 WT session, `TabIdCoordinator`, xoá 4 kênh chết `world:pos/cell/cells/combat` |
| **Bước 3 (Xong)** | RPC UI (`world.cells_rect`, `world.drops`, `chat.send`) qua HTTP fallback + map projectiles trên `WorldTransport` |
| **Bước 4 — code (Xong)** | Xoá `internal/ws` client/limiter, thay bằng `internal/uihub` sink-only; xoá `/api/ws`; xoá `WorldSocket`, `websocket.ts`; rename sang `UIRealtimeManager`/`uiRealtimeManager`; `WorldRuntime` WT-only; TLS + WT trong `main.go`; drain 4010 + chờ session đóng |
| **Bước 4 — infra (Xong)** | Xoá `Caddyfile`; compose publish `443:8443/tcp+udp`, DB/Redis loopback, mount `./certs`, healthcheck HTTPS, `stop_grace_period`; Dockerfile `EXPOSE 8443 8443/udp`; deploy workflow health/HTTP3 probe; `.env.example` TLS/WT |
| **Bước 4 — docs (Xong)** | `README`, `docs/DEVELOPMENT.md`, `docs/API.md`, `docs/PRODUCTION_DEPLOY.md`, file tiến độ này |

## Kiến trúc hiện tại

```
Backend (Go)
├── TLS_ENABLED=true
│   ├── TLS_ADDR=:8443 : http.Server TLS (REST + CONNECT /api/wt)
│   └── WT_ADDR=:8443  : webtransport.Server UDP
│       └── internal/realtime/wttransport
│           ├── Stream 0x01 (world binary v2) → gateway.Admission
│           ├── Stream 0x02 (ui NDJSON)       → internal/uihub (sink-only)
│           ├── Datagram input/ping           → gateway.Admission
│           └── Datagram snapshot delta       ← sim.Snapshot
└── Dev (không TLS): plain HTTP :8080, REST/UI only; WT cần TLS

Frontend
└── RealtimeClient (1 WebTransport session / tab, grace 30 s)
    ├── WorldTransport       (Stream 0x01 + Datagrams)
    └── UIRealtimeManager    (Stream 0x02 NDJSON)
```

- Env backend: `TLS_ENABLED`, `TLS_ADDR`, `WT_ENABLED`, `WT_ADDR`, `TLS_CERT_FILE`, `TLS_KEY_FILE`.
- Env frontend: `NEXT_PUBLIC_REALTIME_URL` (tuỳ chọn); mặc định suy ra từ `NEXT_PUBLIC_BACKEND_URL` + `/api/wt`.
- Route WT: `CONNECT /api/wt` (Auth, `?token=`, `?tab_id=`), `CheckOrigin` dùng chung `CORS_ALLOWED_ORIGINS`.
- Codec: `backend/internal/realtime/protocol/v2` ↔ `frontend/src/game/network/v2/Codec.ts` (golden vector `82d204...`).
- Redis: `uihub.StartRedisSubscribers` vẫn subscribe `player:*`, `refine:*`, `chat:*`, `trade:*`, `combat:*`, `notif:*`, arena, `catalog:*`.

## Đã kiểm chứng

- Go: `go test ./...` pass; targeted `-race` trên `wttransport`, `gateway`, `sim`, `uihub` pass.
- Integration WT thật (client `webtransport-go` ↔ server): handshake → join → welcome baseline →
  input datagram → snapshot delta → events frame; 1 session mở đồng thời UI stream + world stream,
  nhận broadcast `chat:world` và `trade:<id>` từ `uihub`.
- Frontend: `npm test` 215 pass (gồm `UIStream.test.ts`, `RealtimeClient.test.ts`,
  `WorldRuntime.test.ts` với `transportFactory`, store fallback HTTP), `npm run typecheck`, `npm run lint`.

## Việc cần làm tiếp (production rollout)

1. Trỏ DNS: `api.cogioi.io.vn` proxied, `rt.cogioi.io.vn` **DNS-only**.
2. Mở TCP 443 + UDP 443 trên Lightsail; sysctl UDP buffer; gỡ Caddy host.
3. Cấp cert Let's Encrypt cho cả `api` + `rt`, copy vào `./certs/fullchain.pem` / `privkey.pem`.
4. `docker compose up -d`, kiểm tra log HTTPS + WT, chạy smoke test mục 4 của
   `docs/PRODUCTION_DEPLOY.md`.
5. Đo resume 3 s/10 s p95 < 2 s trên WiFi/4G thật.
6. Commit các thay đổi chưa commit trên branch `deploy`.

## Cách chạy thử WT local

```bash
cd backend/cmd/wtspike && ./gen-cert.sh /tmp/wtcert && cd ../..
WT_ENABLED=true WT_ADDR=:8443 \
TLS_CERT_FILE=/tmp/wtcert/srv.crt TLS_KEY_FILE=/tmp/wtcert/srv.key \
go run ./cmd/server
```

Frontend `.env.local`:

```bash
NEXT_PUBLIC_REALTIME_URL=https://localhost:8443/api/wt
```

Chrome yêu cầu cert tin cậy **cho QUIC**; cert tự ký bị
`ERR_QUIC_CERT_ROOT_NOT_KNOWN`. Dùng cert Let's Encrypt thật hoặc policy
`CACertificates` (Chrome 132+). Hướng dẫn production đầy đủ:
[`docs/PRODUCTION_DEPLOY.md`](../PRODUCTION_DEPLOY.md).

## Giới hạn/known issues còn lại

- Chưa chạy smoke production với cert `rt.cogioi.io.vn` (thuộc rollout infra).
- Chưa đo p95 resume trên mạng thật.
- Không claim scale/50 người; guard vẫn `MMO_MAX_PLAYERS_PER_ZONE` (default 5).

## Checklist trước khi tuyên bố “đã thay WS”

- [x] UI stream WT nhận chat/trade/notify/refine/arena/catalog.
- [x] 1 session/tab (world + ui), tab thứ 2 takeover + rotate `tab_id` khi duplicate tab.
- [x] RPC UI hoạt động (WT hoặc HTTP fallback).
- [x] Projectiles & `TRANSFER` đầy đủ trên WT + graceful drain `4010` + `/api/health`.
- [x] `/api/ws` bị xoá; không còn `new WebSocket(` trong `frontend/src`.
- [x] Docs + compose + `.env.example` cập nhật; `Caddyfile` đã xoá.
- [ ] Cert `rt` DNS-only + `TLS_ENABLED=true` production; smoke WiFi/4G.
- [ ] Resume 3 s/10 s p95 < 2 s đo thực tế trên mạng.
