# WT-315 — Ingress & Deploy mới (bỏ Caddy, TLS/QUIC trong Go)

> Ticket: [Spec: ingress & deploy mới (bỏ Caddy)](https://github.com/VipPro67/GuWorld/issues/315) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Dựa trên: [Research: quic-go/webtransport-go + TLS/ACME trong Go trên :443](https://github.com/VipPro67/GuWorld/issues/305), [Research: ingress UDP/443 — Cloudflare DNS-only, ACME DNS-01, Lightsail](https://github.com/VipPro67/GuWorld/issues/307)
> Phạm vi: hạ tầng, TLS, container, DNS, deploy. Không gồm code protocol/client.

## 0. Hiện trạng

| Hạng mục | Hiện tại | Vấn đề |
|---|---|---|
| Backend | plain HTTP `:8080` (`main.go:935`) | không TLS, không WT |
| Caddy | service trong compose **đã comment**, Caddyfile còn; production trả `via: 1.1 Caddy` | Caddy có thể đang chạy host service, chiếm TCP/443 |
| Compose | publish `127.0.0.1:8080`; **5432 và 6379 mở ra ngoài** | lộ DB/Redis (finding #307) |
| Deploy | `git reset` + `compose up -d backend`, không volume cert | không có chỗ chứa cert, không smoke test |
| Origin | CORS đọc `CORS_ALLOWED_ORIGINS`; WS đọc riêng `WS_ALLOWED_ORIGINS` | hai nguồn lệch nhau |
| Shutdown | `srv.Shutdown` → admission drain → manager/hub stop, 10 s | không có drain cho WT, không đóng session chủ động |

## 1. Kiến trúc đích

```
Lightsail (1 box, 1 process Go)
├── TCP :8443  http.Server (HTTPS h1/h2)        ← publish 443:8443/tcp
│     └── cùng mux: REST /api/*, /api/health
├── UDP :8443  webtransport.Server              ← publish 443:8443/udp
│     ├── Extended CONNECT /api/wt → WT session
│     └── H3 request khác → http3.Server.Handler (cùng mux)
├── TCP 127.0.0.1:8081  health server (plain)   ← chỉ trong container
└── certmagic cache /data/certs (named volume)
      DNS-01 Cloudflare → api.cogioi.io.vn + rt.cogioi.io.vn
```

- `TLS_ENABLED=false` (dev): giữ nguyên `:8080` plain, không mở UDP.
- `TLS_ENABLED=true` (prod): bind `:8443` TCP+UDP; `PORT`/`:8080` không dùng.
- Một cert source, hai `tls.Config`: TCP `NextProtos ["h2","http/1.1"]`; QUIC qua `http3.ConfigureTLSConfig` (`["h3"]`).

## 2. Env mới / thay đổi

| Biến | Ví dụ | Ý nghĩa |
|---|---|---|
| `TLS_ENABLED` | `true` | Bật HTTPS + WT trong Go |
| `TLS_ADDR` | `:8443` | TCP HTTPS (tên cũ `PORT` giữ cho dev) |
| `WT_ADDR` | `:8443` | UDP QUIC/WebTransport |
| `TLS_HOSTS` | `api.cogioi.io.vn,rt.cogioi.io.vn` | Danh sách cert cần quản lý (SNI) |
| `TLS_CERT_DIR` | `/data/certs` | certmagic `FileStorage` — phải là volume |
| `ACME_EMAIL` | `ops@cogioi.io.vn` | Liên hệ LE |
| `ACME_CA` | `production` \| `staging` | Lần deploy đầu dùng staging |
| `CF_DNS_API_TOKEN` | (secret) | Token Cloudflare scope **Zone:DNS:Edit + Zone:Read**, 1 zone |
| `HEALTH_ADDR` | `127.0.0.1:8081` | Health server nội bộ khi TLS bật |
| `QUIC_REQUIRE_ADDRESS_VALIDATION` | `true` | Bật Retry/address validation |
| `APP_VERSION` | `${{ github.sha }}` | Trả trong `/api/health` |
| `CORS_ALLOWED_ORIGINS` | `https://cogioi.io.vn,https://www.cogioi.io.vn,http://localhost:3000` | **Dùng chung** cho CORS và WT `CheckOrigin` |
| `WS_ALLOWED_ORIGINS` | (deprecated) | Fallback 1 release, log warn nếu còn dùng |

Bổ sung `.env.example` + `docs/DEVELOPMENT.md`; không commit token thật.

## 3. Setup Lightsail / host (làm 1 lần, theo thứ tự)

1. **Mở firewall Lightsail** (instance → Networking → IPv4/IPv6 Firewall):
   - `Custom UDP 443` từ `0.0.0.0/0` (và `::/0` nếu có AAAA).
   - `Custom TCP 443` từ `0.0.0.0/0`.
   - Giữ `SSH 22`; **đóng** `5432`/`6379` public sau khi compose bind loopback.
   - CLI: `aws lightsail open-instance-public-ports --instance-name <name> --port-info fromPort=443,toPort=443,protocol=udp,cidrs=0.0.0.0/0`
2. **Sysctl UDP buffer** (quic-go xin 7 MiB):
   ```bash
   printf 'net.core.rmem_max=7340032\nnet.core.wmem_max=7340032\n' | sudo tee /etc/sysctl.d/99-gu-quic.conf
   sudo sysctl --system
   ```
3. **Gỡ Caddy khỏi host** (nếu đang chạy host service):
   ```bash
   sudo systemctl stop caddy && sudo systemctl disable caddy
   sudo ss -ltnp '( sport = :443 )'   # phải trống trước khi bật Go TLS
   ```
4. **Docker**: cho container bind `:8443` (không cần `CAP_NET_BIND_SERVICE`); publish `443:8443/tcp+udp` (do root docker làm).
5. **Rate limit UDP per-source** (khuyến nghị, iptables):
   ```bash
   sudo iptables -I INPUT -p udp --dport 443 -m hashlimit \
     --hashlimit-name gu_wt --hashlimit-upto 50/sec --hashlimit-burst 100 \
     --hashlimit-mode srcip --hashlimit-htable-expire 30000 -j ACCEPT
   sudo iptables -A INPUT -p udp --dport 443 -j DROP
   sudo netfilter-persistent save
   ```
   (hoặc nftables tương đương; **không** chặn TCP 443).
6. **DNS Cloudflare**:
   - `rt.cogioi.io.vn` → `A` → IP Lightsail, **Proxy = DNS only (xám)**, TTL **60 s** lúc cắt.
   - `api.cogioi.io.vn` giữ **Proxied (cam)**, SSL mode **Full (strict)**.
   - Sau ổn định 24 h: TTL `rt` → 300 s.
7. **Token Cloudflare**: tạo API token zone-scoped (`Zone:DNS:Edit`, `Zone:Read`, chỉ `cogioi.io.vn`), dán vào `backend/.env` trên host (`CF_DNS_API_TOKEN`), không đưa vào repo/GH secret nếu compose env_file đã có.

## 4. Dockerfile & docker-compose

**Dockerfile** thêm:
```dockerfile
RUN ... && mkdir -p /data/certs && chown -R app:app /data/certs
...
EXPOSE 8080 8443 8443/udp
```
(named volume sẽ kế thừa quyền `app:app` khi khởi tạo rỗng).

**docker-compose.yml** (diff chính):
```yaml
services:
  db:
    ports: ["127.0.0.1:5432:5432"]          # không mở public
  redis:
    ports: ["127.0.0.1:6379:6379"]          # không mở public
  backend:
    ports:
      - "443:8443/tcp"
      - "443:8443/udp"
      - "127.0.0.1:8080:8080"               # chỉ debug local
    environment:
      - TLS_ENABLED=true
      - TLS_ADDR=:8443
      - WT_ADDR=:8443
      - TLS_HOSTS=api.cogioi.io.vn,rt.cogioi.io.vn
      - TLS_CERT_DIR=/data/certs
      - HEALTH_ADDR=127.0.0.1:8081
      - ACME_EMAIL=ops@cogioi.io.vn
      - ACME_CA=production
      - QUIC_REQUIRE_ADDRESS_VALIDATION=true
      - APP_VERSION=${BACKEND_IMAGE:-dev}
      - CORS_ALLOWED_ORIGINS=https://www.cogioi.io.vn,https://cogioi.io.vn,http://localhost:3000
    volumes:
      - guworld_certs:/data/certs
    stop_grace_period: 30s
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8081/api/health >/dev/null || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 12
volumes:
  guworld_certs:
```
- Xoá service `caddy` (đã comment), volumes `caddy_data`/`caddy_config`, xoá file `Caddyfile`.
- `CF_DNS_API_TOKEN` để trong `backend/.env` (env_file) — không lặp trong compose.

## 5. Thay đổi code backend (bối cảnh thi công)

1. **`main.go`**: nhánh `TLS_ENABLED`:
   - `certmagic` cache + `ACMEIssuer` DNS-01 (`libdns/cloudflare`, provider nhận token env), `DisableHTTPChallenge/DisableTLSALPNChallenge`, `Storage: FileStorage{TLS_CERT_DIR}`, `ManageSync(ctx 2m, TLS_HOSTS)`; lỗi ⇒ fail fast (prod).
   - `tlsTCP := magic.TLSConfig(); tlsTCP.NextProtos = ["h2","http/1.1"]`; `tlsQUIC := http3.ConfigureTLSConfig(magic.TLSConfig())`.
   - `quic.Config`: `MaxIdleTimeout 90s`, `KeepAlivePeriod 20s`, `MaxIncomingStreams/UniStreams 16`, `MaxStreamReceiveWindow 256 KiB`, `MaxConnectionReceiveWindow 2 MiB`, `EnableDatagrams true`, `Allow0RTT false`, `RequireAddressValidation` = cờ env.
   - `http3.Server{TLSConfig: tlsQUIC, Handler: r, QUICConfig: ...}`; `webtransport.Server{H3: h3, CheckOrigin: shared helper, ApplicationProtocols: ["gu.realtime.v1"]}`.
   - `mux` thêm `POST/GET /api/wt` (nâng cấp) và `GET /api/health`; HTTP `:8443` dùng `r` hiện có.
   - `ListenAndServeTLS("", "")` (cert từ `GetCertificate`).
2. **Health**: `GET /api/health` không auth/DB, trả `{status:"ok", version, uptime_sec}`; health server plain tại `HEALTH_ADDR` khi TLS bật; miễn rate limit.
3. **Origin helper**: `middleware.OriginAllowlist()` parse `CORS_ALLOWED_ORIGINS` (fallback `WS_ALLOWED_ORIGINS` + warn); CORS middleware và `webtransport.CheckOrigin` cùng dùng; `CheckOrigin` so khớp cả origin có/không trailing slash; cho phép `Origin` rỗng = same-origin (giữ hành vi WS).
4. **Shutdown** (`main.go`): theo thứ tự §6, gửi close `4010 server_draining`, chờ tối đa 5 s, rồi `tcp.Shutdown(15s)` → `wt.Close()` → `udp.Close()` → admission/manager/hub.
5. **Close code `4010`**: thêm vào bảng mã của #311; client (#316) coi là retry backoff, không phải tab-replaced.
6. **Log redaction**: request logging chỉ log `URL.Path` (không query); nếu log full URL ở đâu, redact `token` và `tab_id`.

## 6. Thứ tự shutdown / deploy

```
SIGTERM
 1. draining = true            → join mới 503, WT upgrade 503
 2. gửi 4010 tới mọi session   → chờ ≤ 5 s client tự đóng
 3. tcpSrv.Shutdown(15s)       → h1/h2 GOAWAY, request đang chạy xong
 4. wtSrv.Close()              → đóng QUIC + http3.Server
 5. udp.Close()
 6. admission.Drain()/checkpoint → manager.Stop → hub.Stop → DB/Redis
```
- Compose `stop_grace_period: 30s` > tổng 5+15+~5.
- Rolling restart vẫn có thể làm rớt session nếu client không kịp; client resume theo #318 khi lên lại.

## 7. Deploy workflow (`.github/workflows/deploy-backend.yml`)

- `paths`: bỏ `Caddyfile`, giữ `backend/**`, `docker-compose.yml`, workflow.
- Build image như cũ; tag `latest` + `${{ github.sha }}`.
- Deploy script:
  1. `git fetch/reset deploy`.
  2. Ghi `BACKEND_IMAGE` = tag SHA; `docker compose pull backend`.
  3. Lưu image đang chạy để rollback: `docker inspect --format '{{.Config.Image}}' guworld-backend > /tmp/gu_prev_image`.
  4. `docker compose up -d --no-deps backend`.
  5. Chờ healthcheck (tối đa 90 s): `docker inspect -f '{{.State.Health.Status}}' guworld-backend` == `healthy`.
  6. Smoke: `curl -fsS https://api.cogioi.io.vn/api/health` (qua Cloudflare → origin TCP 443) và `openssl s_client -connect rt.cogioi.io.vn:443 -servername rt.cogioi.io.vn` kiểm tra cert.
  7. Nếu health fail: `BACKEND_IMAGE=$(cat /tmp/gu_prev_image) docker compose up -d --no-deps backend` + báo lỗi.

> **Frontend deploy tự động** qua Cloudflare Workers Builds khi push commit (OpenNext); backend workflow chỉ chạy khi `backend/**`/compose đổi. Thứ tự rollout/rollback giữa hai pipeline theo [Spec: rollout & rollback](https://github.com/VipPro67/GuWorld/issues/309) — frontend rollback chọn deployment trước trong Cloudflare hoặc revert commit.

- WT handshake smoke (Go client hoặc script trong repo) là **checklist tay** 1 tuần đầu; đưa vào CI sau khi ổn định.
- Rollback toàn bộ về WS: theo #309 (revert commit B/C + `BACKEND_IMAGE` pin + bật lại Caddy host) — spec này chỉ đảm bảo đường lùi hạ tầng.

## 8. Docs phải sửa

| File | Nội dung |
|---|---|
| `README.md` | bỏ "reverse proxy", mô tả `:8080` dev và `:8443` TLS + UDP WT prod |
| `docs/DEVELOPMENT.md` | env mới (§2), quy trình cert/DNS/firewall, lệnh kiểm tra, shutdown |
| `docs/API.md` | `/api/health`, `/api/wt` (handshake), ghi chú Cloudflare cho WT |
| `.env.example` | env §2, ghi rõ token Cloudflare |
| `AGENTS.md` | startup/commands nếu port mặc định đổi (giữ dev `:8080`) |
| `Caddyfile` | **xoá file** |
| `docker-compose.yml` | §4 |

## 9. Checklist xác minh sau deploy

- [ ] `docker compose ps` healthy; `curl -fsS https://api.cogioi.io.vn/api/health` → 200 `{status:"ok"}`.
- [ ] `openssl s_client -connect api.cogioi.io.vn:443 -servername api.cogioi.io.vn` → cert hợp lệ, chain ngắn (ECDSA).
- [ ] `openssl s_client -connect rt.cogioi.io.vn:443 -servername rt.cogioi.io.vn` → cùng cert source.
- [ ] `sudo ss -lunp '( sport = :443 )'` thấy process Go; `ss -ltnp` thấy `:8443` (hoặc docker-proxy `:443`).
- [ ] `dig +short rt.cogioi.io.vn` trả IP (không phải Cloudflare range); `dig` `api` trả Cloudflare.
- [ ] WT handshake từ browser (Chrome + Safari 26.4) vào `/api/wt`; `transport.ready` resolve; nhận 1 snapshot.
- [ ] Cloudflare WAF vẫn chặn REST (`/api/admin/*` không lộ qua `rt`).
- [ ] `sysctl net.core.rmem_max` = 7340032; log quic-go không cảnh báo buffer.
- [ ] Log không chứa `token=`/`tab_id=` ở query.
- [ ] Renewal: `certmagic` log có cert expiry; xoá `/data/certs` trên staging → tự cấp lại (test rate limit).
- [ ] Restart container: cert load từ volume, không gọi ACME lại.

## 10. Rủi ro & biên

- **Origin IP lộ** khi grey-cloud `rt` — không có WAF/DDoS L7; volumetric flood không mitigate được trên Lightsail (đã chấp nhận ở #307). Giảm thiểu: rate limit per-source, QUIC Retry, limiter app-level, alert UDP pps.
- **Cloudflare proxy cho `api`**: origin chỉ nói HTTP/2 với CF (không H3 tới origin) — chấp nhận; H3 chỉ dùng trực tiếp ở `rt`.
- **Cert thất bại lúc boot** (token hết hạn/mất mạng): prod fail fast; container restart loop sẽ hiện rõ trong health; staging dùng `ACME_CA=staging` để tránh đốt rate limit LE.
- **Caddy host còn chạy** chiếm TCP/443 → Go bind fail; checklist §3.3 bắt buộc.
- **Port 80 đóng hoàn toàn**: DNS-01 không cần; nếu còn dùng redirect HTTP→HTTPS thì thêm listener riêng sau, không thuộc spec này.
- **DB/Redis public** hiện tại: đổi bind loopback cùng lúc; nếu có client ngoài phụ thuộc 5432 public phải xử lý trước.
- **Tab ẩn/mobile sleep** khi deploy: session resume #318; close `4010` báo trước 5 s.
