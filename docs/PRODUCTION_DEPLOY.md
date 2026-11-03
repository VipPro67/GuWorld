# Production deployment — HTTPS + WebTransport (no Caddy, no WebSocket)

Tài liệu này là checklist triển khai production cho GuWorld sau khi WebTransport
là transport realtime duy nhất. Caddy, WebSocket và `/api/ws` đã bị xoá; Go phục
vụ HTTPS (TCP/443) và QUIC/WebTransport (UDP/443) trong cùng process.

## 1. Kiến trúc đích

```
Internet
├── TCP 443 ──► Cloudflare (api.cogioi.io.vn, proxied)
│                └──► Lightsail :443 ──► container :8443 (http.Server, h1/h2)
│                       REST /api/*, /api/health, CONNECT /api/wt
└── UDP 443 ──► rt.cogioi.io.vn (DNS-only, không qua Cloudflare)
                 └──► Lightsail :443/udp ──► container :8443/udp (webtransport-go)
```

- `TLS_ENABLED=true`: bật HTTPS trên `TLS_ADDR` (mặc định `:8443`) và WT trên
  `WT_ADDR` (mặc định `:8443`, UDP). Thiếu cert/key hoặc realtime runtime thì
  process **dừng**, không fallback HTTP.
- Dev không đặt `TLS_ENABLED`/`WT_ENABLED`: backend chạy plain HTTP trên `PORT`
  (`:8080`), phù hợp `go run ./cmd/server` + frontend `localhost:3000`.
- Frontend mặc định suy ra WT URL từ `NEXT_PUBLIC_BACKEND_URL` (đổi `https`) +
  `/api/wt`; override bằng `NEXT_PUBLIC_REALTIME_URL`.

## 2. Chuẩn bị một lần

### 2.1 DNS Cloudflare

| Bản ghi | Loại | Proxy | Ghi chú |
| --- | --- | --- | --- |
| `api.cogioi.io.vn` | `A` → IP Lightsail | **Proxied (cam)** | SSL mode **Full (strict)** |
| `rt.cogioi.io.vn` | `A` → IP Lightsail | **DNS only (xám)** | Bắt buộc: Cloudflare không proxy QUIC/WebTransport tới origin |

TTL `rt` để `60 s` trong ngày cắt, sau ổn định 24 h tăng lên `300 s`.

> Nếu `rt` để Proxied, browser sẽ bắt tay QUIC với Cloudflare và không tới được
> server; WebTransport sẽ fail dù firewall đúng.

### 2.2 Firewall Lightsail

Mở từ `0.0.0.0/0` (và `::/0` nếu có IPv6):

- `Custom TCP 443`
- `Custom UDP 443`
- Giữ `SSH 22`

Đóng public `5432`, `6379` sau khi compose chỉ bind loopback.

CLI:

```bash
aws lightsail open-instance-public-ports --instance-name <name> \
  --port-info fromPort=443,toPort=443,protocol=tcp,cidrs=0.0.0.0/0
aws lightsail open-instance-public-ports --instance-name <name> \
  --port-info fromPort=443,toPort=443,protocol=udp,cidrs=0.0.0.0/0
```

### 2.3 Sysctl UDP buffer

quic-go yêu cầu buffer lớn hơn mặc định:

```bash
printf 'net.core.rmem_max=7340032\nnet.core.wmem_max=7340032\n' \
  | sudo tee /etc/sysctl.d/99-gu-quic.conf
sudo sysctl --system
sysctl net.core.rmem_max net.core.wmem_max
```

### 2.4 Gỡ Caddy khỏi host

Caddy từng chạy ngoài Docker và chiếm TCP/443. Phải dừng trước khi publish
`443:8443`:

```bash
sudo systemctl stop caddy 2>/dev/null || true
sudo systemctl disable caddy 2>/dev/null || true
docker rm -f guworld-caddy 2>/dev/null || true
sudo ss -ltnp '( sport = :443 )'   # phải trống
sudo ss -lunp '( sport = :443 )'   # phải trống
```

### 2.5 Chứng chỉ TLS

Browser bắt buộc cert **publicly trusted** cho cả HTTPS và QUIC. Cert tự ký
(self-signed) làm Chrome từ chối QUIC với `ERR_QUIC_CERT_ROOT_NOT_KNOWN`.

Khuyến nghị: Let's Encrypt bằng DNS-01 (Cloudflare), cấp **một** cert cho cả
`api` và `rt`.

```bash
sudo apt-get update && sudo apt-get install -y certbot python3-certbot-dns-cloudflare

sudo install -m 600 /dev/null /etc/letsencrypt/cloudflare.ini
sudo tee /etc/letsencrypt/cloudflare.ini >/dev/null <<'EOF'
dns_cloudflare_api_token = <TOKEN_ZONE_DNS_EDIT>
EOF

sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d api.cogioi.io.vn -d rt.cogioi.io.vn \
  --agree-tos -m ops@cogioi.io.vn --non-interactive
```

Token Cloudflare chỉ cần scope `Zone:DNS:Edit` + `Zone:Read` cho zone
`cogioi.io.vn`.

Copy/symlink vào thư mục compose mount:

```bash
sudo mkdir -p /home/<user>/GuWorld/certs
sudo cp /etc/letsencrypt/live/api.cogioi.io.vn/fullchain.pem \
        /home/<user>/GuWorld/certs/fullchain.pem
sudo cp /etc/letsencrypt/live/api.cogioi.io.vn/privkey.pem \
        /home/<user>/GuWorld/certs/privkey.pem
sudo chown root:10001 /home/<user>/GuWorld/certs
sudo chmod 755 /home/<user>/GuWorld/certs
sudo chown root:10001 /home/<user>/GuWorld/certs/fullchain.pem /home/<user>/GuWorld/certs/privkey.pem
sudo chmod 644 /home/<user>/GuWorld/certs/fullchain.pem
sudo chmod 640 /home/<user>/GuWorld/certs/privkey.pem
```

Container chạy bằng user `app` với UID/GID cố định `10001` (xem
`backend/Dockerfile`). Cert phải đọc được bởi GID `10001`; `640` cho `privkey.pem`
là đủ và không mở quyền cho user khác trên host.

Renewal hook (certbot tự renew, nhưng phải copy sang `certs/`):

```bash
sudo tee /etc/letsencrypt/renewal-hooks/deploy/guworld.sh >/dev/null <<'EOF'
#!/bin/sh
set -e
DOMAIN=api.cogioi.io.vn
DEST=/home/<user>/GuWorld/certs
cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "$DEST/fullchain.pem"
cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" "$DEST/privkey.pem"
chown root:10001 "$DEST/fullchain.pem" "$DEST/privkey.pem"
chmod 644 "$DEST/fullchain.pem"
chmod 640 "$DEST/privkey.pem"
docker restart guworld-backend
EOF
sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/guworld.sh
sudo certbot renew --dry-run
```

> `docker restart` khi renew sẽ gửi close code `4010` cho client đang chơi; client
> tự reconnect sau khi backend lên lại. Có thể dời restart vào cửa sổ bảo trì.

### 2.6 `backend/.env` trên host

Các biến tối thiểu (giá trị thật không commit):

```dotenv
PROD_ENV=production
JWT_SECRET=<random-64-bytes>
DB_HOST=db
DB_PORT=5432
DB_USER=gu_admin
DB_PASSWORD=<strong-password>
DB_NAME=gu_db
DB_SSLMODE=disable
REDIS_HOST=redis
REDIS_PORT=6379
CORS_ALLOWED_ORIGINS=https://www.cogioi.io.vn,https://cogioi.io.vn
```

`TLS_*`/`WT_*` đã được đặt trong `docker-compose.yml`, không cần lặp lại. Nếu
muốn đổi đường dẫn cert trong container, sửa compose tương ứng.

### 2.7 Env build của frontend

`NEXT_PUBLIC_*` được nhúng lúc **build frontend**, không phải biến runtime của
backend. Đặt trong môi trường build/CI của frontend (hoặc `frontend/.env.local`):

```dotenv
NEXT_PUBLIC_BACKEND_URL=https://api.cogioi.io.vn
NEXT_PUBLIC_REALTIME_URL=https://rt.cogioi.io.vn/api/wt
```

`NEXT_PUBLIC_REALTIME_URL` là bắt buộc trên production: mặc định client suy ra
`https://<api host>/api/wt`, nhưng `api` đi qua Cloudflare proxy (không chuyển
QUIC tới origin). `rt` phải là DNS-only và trỏ thẳng Lightsail. Nếu build bằng
`frontend/Dockerfile`, truyền cả hai làm build arg:

```bash
docker build -f frontend/Dockerfile \
  --build-arg NEXT_PUBLIC_BACKEND_URL=https://api.cogioi.io.vn \
  --build-arg NEXT_PUBLIC_REALTIME_URL=https://rt.cogioi.io.vn/api/wt \
  -t guworld-frontend .
```

## 3. Deploy

### 3.1 Lần đầu

```bash
cd /home/<user>/GuWorld
cp .env.example .env          # POSTGRES_PASSWORD cho compose
# điền backend/.env như mục 2.6
mkdir -p certs                # đã copy cert ở 2.5

docker compose pull
docker compose up -d
docker compose ps
docker compose logs -f backend
```

Log thành công phải có dòng tương tự:

```
Cultivation Backend listening on :8443 (HTTPS + WebTransport on :8443/udp)
```

Kiểm tra nhanh:

```bash
# HTTPS qua origin (bỏ qua DNS/Cloudflare, vẫn xác minh cert)
curl -fsS --resolve api.cogioi.io.vn:443:127.0.0.1 \
  https://api.cogioi.io.vn/api/health

# HTTP/3 (nếu curl có hỗ trợ --http3-only)
curl -fsS --http3-only --resolve api.cogioi.io.vn:443:127.0.0.1 \
  https://api.cogioi.io.vn/api/health

# UDP đang mở
sudo ss -lunp '( sport = :443 )'
```

### 3.2 Từ GitHub Actions

Workflow `.github/workflows/deploy-backend.yml` build image, SSH vào host, `git
reset --hard origin/deploy`, `docker compose pull backend`, `up -d --no-deps
backend`, rồi health check HTTPS và cảnh báo nếu HTTP/3 probe fail.

Secrets cần có: `LIGHTSAIL_HOST`, `LIGHTSAIL_USERNAME`, `LIGHTSAIL_SSH_KEY`.
`GITHUB_TOKEN` dùng để pull GHCR image.

## 4. Smoke test sau deploy

1. Mở `https://www.cogioi.io.vn`, đăng nhập, vào thế giới.
2. DevTools → Network → filter `wt`: phải thấy **một** request `CONNECT
   /api/wt?token=…&tab_id=…` status `200`, protocol `h3`.
3. Di chuyển, cast chiêu, nhận snapshot; chat thế giới hoạt động.
4. Mở tab thứ hai cùng tài khoản: tab mới vào world, tab cũ nhận
   `4001 session_replaced` và dừng reconnect.
5. Deploy một image mới: client đang chơi nhận `4010 server_draining`, sau đó tự
   reconnect khi backend lên.
6. Tắt Wi-Fi/4G và bật lại: client resume cùng `tab_id`, giữ entity/vị trí
   trong cửa sổ 15 s/30 s.
7. Kiểm tra chặn UDP (VPN/browser cũ): UI hiện trạng thái không realtime, HTTP
   (chợ, tu luyện, inventory, chat HTTP) vẫn dùng được; world bị khóa.

## 5. Rollback

Không còn Caddy để bật lại. Rollback là deploy image backend trước đó:

```bash
cd /home/<user>/GuWorld
export BACKEND_IMAGE=ghcr.io/vippro67/guworld-backend:<previous-sha>
docker compose pull backend
docker compose up -d --no-deps backend
```

Nếu sự cố nằm ở DNS/firewall, khôi phục bản ghi/proxy hoặc mở lại port tương
ứng; cert và compose không đổi.

## 6. Troubleshooting

| Triệu chứng | Nguyên nhân thường gặp | Xử lý |
| --- | --- | --- |
| Backend thoát ngay khi start | Thiếu `TLS_CERT_FILE`/`TLS_KEY_FILE`, cert không đọc được, hoặc realtime runtime nil | `docker compose logs backend`; kiểm tra file trong `certs/` và quyền đọc |
| `permission denied` khi đọc `privkey.pem` | Cert không thuộc GID `10001` (user `app` trong container) | `sudo chown root:10001 certs/*; sudo chmod 640 certs/privkey.pem` |
| `bind: address already in use` | Caddy/host service còn giữ 443 | Dừng Caddy, kiểm tra `ss -ltnp`/`ss -lunp` |
| Browser báo `ERR_QUIC_CERT_ROOT_NOT_KNOWN` | Dùng cert self-signed | Dùng Let's Encrypt (mục 2.5) |
| WT không kết nối, HTTPS vẫn OK | Cloudflare proxy `rt`, UDP/443 bị chặn, hoặc firewall thiếu UDP | Đặt `rt` DNS-only; mở UDP 443; kiểm tra `ss -lunp` |
| `net::ERR_HTTP3_FAILURE` | UDP bị chặn giữa client và origin | Thử mạng khác/tắt VPN; kiểm tra firewall nhà mạng |
| Chat/notification không realtime nhưng world chạy | UI stream `0x02` lỗi hoặc subscribe sai scope | Xem console `[RealtimeUI]`; kiểm tra close code |
| Tab cũ liên tục reconnect | Không nhận được `4001` (mạng đứt trước khi đóng) | Hành vi an toàn: client coi như mất mạng và retry; đóng tab cũ |
| Cert hết hạn sau 90 ngày | Renewal hook không chạy | `sudo certbot renew --dry-run`; kiểm tra hook copy + restart |
| `401 Unauthorized` trên `/api/wt` | Token query hết hạn hoặc thiếu | Đăng nhập lại; kiểm tra `NEXT_PUBLIC_REALTIME_URL` cùng host |

## 7. Ghi chú vận hành

- Log access phải redact `token` trong query `/api/wt`; không đưa token vào
  dashboard/log bên thứ ba.
- `docker-compose.yml` bind DB/Redis vào loopback; không mở public.
- Healthcheck dùng `curl -fsSk https://127.0.0.1:8443/api/health` vì cert cấp cho
  hostname, không cho loopback (`-k` chỉ dùng nội bộ container).
- `stop_grace_period: 30s` để backend kịp gửi `4010` và đóng session trước
  SIGKILL.
- Chỉ chạy **một** backend dùng chung PostgreSQL/Redis trong mô hình hiện tại;
  nhiều instance vẫn fan-out UI qua Redis nhưng character lease/sim là
  per-process.
