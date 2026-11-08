# WT-309 — Rollout & Rollback khi xoá WebSocket

> Ticket: [Chốt kế hoạch rollout & rollback khi xoá WS](https://github.com/VipPro67/GuWorld/issues/309) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Liên quan: #305, #307, #308, #310, #312, #313, #315, #316, #318

## 0. Bối cảnh triển khai (GitOps)

- **Backend**: push lên branch `deploy` → GitHub Actions build image → SSH Lightsail → `docker compose up -d backend`. Workflow chỉ trigger khi đổi `backend/**`, `docker-compose.yml`, workflow.
- **Frontend**: Cloudflare **Workers Builds** (OpenNext) tự build & deploy khi có commit mới; không có workflow frontend trong repo.
- **Hệ quả**: không thể "deploy backend trước, client sau" bằng tay trong cùng một commit. Thứ tự phải điều khiển bằng **chuỗi commit tách biệt** + cửa sổ bảo trì ngắn.
- **Chưa xác minh từ repo** (bắt buộc kiểm tra trước D-1): CF Workers Builds có rebuild/deploy khi commit **không đổi frontend** hay không; branch trigger; preview deployment có bật không. Ghi kết quả vào ticket QA trước khi chốt lịch.

## 1. Quyết định

### 1.1 Ba pha, không chạy song song 2 transport

| Pha | Nội dung | Transport sống sau pha |
|---|---|---|
| **A — Pre-flight client (N-1)** | Client vẫn WS nhưng có: build-id, đọc `/api/health`, overlay "Bản game đã cũ → Tải lại", capability UX (#312), telemetry capability | WS (cả 2 phía) |
| **B — Backend WT** | Backend TLS/QUIC, `/api/wt`, **xoá `/api/ws`** | Chỉ WT; client N-1 (WS) đã biết báo "tải lại" |
| **C — Client WT (N)** | Client chuyển sang WT đầy đủ (#316) | WT |

- Cửa sổ "cả hai transport cùng sống" = **0**. Không giữ WS để fallback.
- Pha A deploy trước **≥ 48 h**, không downtime.
- Pha B + C trong **cùng cửa sổ bảo trì 30 phút** (04:00–04:30 ICT), cách nhau vài phút.

### 1.2 Version gate

- `GET /api/health` trả `{status, build_id, protocol_version, min_client_build, uptime_sec}`.
- `WELCOME` v2 mang `build_id` + `protocol_version`.
- Client nhúng `NEXT_PUBLIC_BUILD_ID` (CF env = commit SHA lúc build).
- Client N-1 kiểm tra **lúc boot** và **trước join**:
  - server `protocol_version` > client → overlay **"Bản game đã cũ"** + nút **Tải lại trang**;
  - `NEXT_PUBLIC_BUILD_ID` cũ hơn `min_client_build` → cùng overlay;
  - không auto-reload khi đang trong trận; chỉ reload theo nút.
- Server từ chối handshake v1 sau cutover bằng `4002 protocol_version_mismatch` (message yêu cầu cập nhật).

### 1.3 Rollback = revert commit, giữ image pin làm đường nhanh

- Mọi release gắn tag: backend `ghcr.io/vippro67/guworld-backend:<sha>`, frontend deployment id Cloudflare, commit SHA.
- **Rollback chuẩn**: `git revert` các commit B/C → push `deploy` → backend workflow + CF Workers tự deploy bản cũ. Thời gian mục tiêu **≤ 15 phút**.
- **Đường nhanh khi CF build chậm**: trên host đặt `BACKEND_IMAGE=<sha cũ>` rồi `docker compose up -d --no-deps backend`; frontend rollback qua Cloudflare dashboard (chọn deployment trước).
- Bật lại Caddy host **chỉ khi** rollback về backend WS (pha A/B cũ): theo #315 §3.3, dừng Caddy trước khi bật WT; làm ngược lại khi lùi.
- Schema chỉ additive ⇒ backend cũ chạy được trên DB mới; không xoá bảng/cột, không down-migration.
- Rehearsal bắt buộc trước go-live: revert B trên staging/box phụ hoặc trong maintenance window, đo thời gian, ghi lại.

### 1.4 Kill switch vận hành

- Thêm admin endpoint `POST /api/admin/realtime/drain` (permission mới `realtime.drain`; audit `admin.realtime_drain`; không cần reason nếu là khẩn cấp, nhưng nên có).
  - Gọi `Admission.Drain()`: pause admission, checkpoint, release lease; trả `{drained, pending}`.
  - Dùng khi sự cố transport giữa rollout; người chơi thấy "thế giới tạm đóng", dữ liệu an toàn; không phải fallback WS.
- Không dùng drain trong cutover bình thường (close `4010` + drain đã có ở shutdown #315).

### 1.5 Thông báo người chơi

| Thời điểm | Kênh | Copy |
|---|---|---|
| D-1 (24 h) | Banner trong game + post cộng đồng | "Bản cập nhật lớn: thế giới realtime chuyển sang giao thức mới. Bảo trì 04:00–04:30 ngày DD/MM. Vui lòng tải lại trang sau bảo trì." |
| D-0 trước 1 h | Banner đếm ngược | "Còn 60 phút nữa bảo trì thế giới realtime. Hãy hoàn tất trận đang chơi." |
| Trong bảo trì | Trang maintenance Cloudflare | "Đang nâng cấp thế giới realtime… Quay lại sau ít phút." |
| Sau deploy | Banner + post | "Cập nhật hoàn tất. Nếu game không vào được, hãy tải lại trang (Ctrl/Cmd+Shift+R)." |

- Pha A (pre-flight) không cần bảo trì; chỉ thông báo "có bản cập nhật".

## 2. Runbook cutover (D-0)

### 2.1 Trước cửa sổ (D-1 → D-0)

- [ ] Chạy `scripts/mmo-release-gate.sh` (Gate D automated subset) trên commit B; đính log.
- [ ] Xác nhận CF Workers Builds: branch trigger, có deploy khi commit backend-only không, preview deployment khả dụng.
- [ ] Chụp lại: backend image SHA hiện tại, frontend deployment id hiện tại, `min_client_build` hiện tại.
- [ ] Checklist #315: cert staging → production, `rt` DNS-only, UDP 443 mở, sysctl, Caddy host đã tắt, healthcheck xanh.
- [ ] Kiểm tra capability #312 từ pha A: `% unsupported+blocked` (ngưỡng < 8 %).
- [ ] Tập dượt rollback (revert commit trên nhánh thử + deploy vào box phụ) và ghi thời gian.

### 2.2 Pha A — Pre-flight client (N-1)

1. Commit chỉ đụng `frontend/**` (+ docs): build-id gate, health check, capability UX.
2. Push `deploy`; CF Workers build & deploy. Backend workflow không chạy (path filter).
3. Verify: bundle mới có `NEXT_PUBLIC_BUILD_ID` = commit SHA; `/api/health` đọc được; overlay "Bản game đã cũ" hiển thị khi giả lập `min_client_build` cao hơn.
4. Theo dõi 48 h: `POST /api/realtime/capability` có dữ liệu, lỗi client không tăng.

### 2.3 Pha B — Backend WT (trong maintenance)

1. Bật trang maintenance (Cloudflare rule/Worker) cho app + API.
2. Merge commit B (`backend/**` + `docker-compose.yml` + xoá `Caddyfile` + docs): backend TLS/QUIC, xoá `/api/ws`, `/api/health` có `protocol_version=2`, `min_client_build=<sha pha C>`, `APP_VERSION`.
3. Theo dõi workflow: image build → SSH deploy → Docker healthy (≤ 10 phút).
4. Verify §3 (mục 1–6) trước khi mở pha C.

### 2.4 Pha C — Client WT (N)

1. Merge commit C (`frontend/**`, protocol v2 + WT client #316).
2. CF Workers build & deploy (2–5 phút). Nếu CF build chậm, có thể tạm giữ maintenance.
3. Verify §3 (mục 7–12), mở maintenance, chạy smoke flow nội bộ 10 phút.
4. Công bố hoàn tất; theo dõi 24 h.

**Thứ tự bắt buộc**: A → (48 h) → B → C. Không đảo B/C.

## 3. Checklist verify sau cutover

1. `GET https://api.cogioi.io.vn/api/health` → `protocol_version=2`, `build_id` = commit backend.
2. `openssl s_client` 2 host hợp lệ; `ss -lunp :443` có process Go.
3. `POST /api/wt` handshake từ Chrome + Safari thành công (script smoke).
4. `/api/ws` trả 404/410 (không còn upgrade) — xác nhận WS đã xoá.
5. Drain endpoint trả `{drained:0}` khi không có session (không lỗi quyền).
6. Mở tab cũ (bundle N-1) → overlay "Bản game đã cũ" → **Tải lại** vào được bản N.
7. Join world, chạy, click-to-move, cast, nhận damage, loot.
8. Mất mạng 3 s / 10 s → resume p95 < 2 s, success > 95 % (telemetry #318).
9. Chat/trade/notification UI stream hoạt động trên WT.
10. `POST /api/realtime/capability` ghi nhận `supported`.
11. Không có lỗi 5xx mới trong `/api/admin/monitor/requests/summary` 30 phút.
12. Rollback rehearsal lần cuối: `docker compose` pin image A + revert client → đo thời gian.

## 4. Go / No-Go gate

| # | Điều kiện | Nguồn | Ngưỡng |
|---|---|---|---|
| 1 | Baseline WiFi + 4G có số | #308 | đủ để so sánh trước/sau |
| 2 | Spike WT không tệ hơn WS | #310 | jitter giảm ≥ 20 % **hoặc** p95 RTT ≤ baseline + 20 ms |
| 3 | Zone 50 người | #314 | tick p99 < 25 ms; fan-out không drop |
| 4 | Resume | #318 | p95 < 2 s; success > 95 % (3 s & 10 s) |
| 5 | Client bị chặn | #312 | < 8 % tổng (đo từ pha A) |
| 6 | Rollback | #309 | rehearsal ≤ 15 phút |
| 7 | Ingress | #315 | checklist cert/DNS/UDP/sysctl xanh |
| 8 | Chất lượng | Gate D | không P0/P1 mở ở movement/combat/economy/social |

Bất kỳ mục fail ⇒ **HOLD**, dời cutover; không có ngoại lệ "deploy rồi sửa".

## 5. Rollback runbook

### 5.1 Sự cố trong/ngay sau cutover

1. Kích hoạt maintenance.
2. `POST /api/admin/realtime/drain` (nếu backend còn sống) → checkpoint + release; ghi `drained`.
3. **Frontend**: Cloudflare dashboard → chọn deployment trước (hoặc revert commit C + push).
4. **Backend**:
   - Đường nhanh: `BACKEND_IMAGE=<sha pha A> docker compose up -d --no-deps backend` trên host.
   - Chuẩn: revert commit B + push `deploy`.
5. Nếu quay về bản WS: bật lại Caddy host (`systemctl start caddy`), đảm bảo Go bind `:8080` loopback; kiểm tra `https://api…` qua Caddy.
6. Verify: health, login, join world, chat; mở maintenance.
7. Ghi timeline, nguyên nhân, dữ liệu mất (nếu có) vào QA ticket.

### 5.2 Tiêu chí rollback

- WT handshake fail > 20 % trong 10 phút; hoặc
- Resume success < 90 % trong 15 phút; hoặc
- tick p99 > 25 ms kéo dài > 10 phút; hoặc
- lỗi 5xx mới > 1 % trên REST; hoặc
- phát hiện mất dữ liệu/dupe.

## 6. Docs phải cập nhật

| File | Nội dung |
|---|---|
| `docs/mmo/cutover.md` | Thay "rollout stages" bằng 3 pha A/B/C; rollback revert + image pin; bỏ mô tả flag/allowlist cũ nếu còn |
| `docs/mmo/rollout.md` | Trạng thái transport mới, build-id gate, GitOps order |
| `docs/mmo/release-checklist.md` | Gate D bổ sung mục WT (browser matrix, resume, capability, cert) |
| `docs/guides/DEVELOPMENT.md` | Quy trình deploy frontend CF Workers + backend workflow, rollback |
| `README.md` | Yêu cầu browser/mạng, bảo trì |
| `docs/guides/API.md` | `/api/health` fields, drain endpoint |

## 7. Rủi ro

- **CF build tự động trên mọi commit**: nếu commit B (backend-only) cũng rebuild frontend, client N-1 (WS) sẽ deploy lại đúng lúc WS bị xoá → chờ CF build C là cửa sổ lỗi. Giảm thiểu: giữ maintenance tới khi C lên; hoặc pause CF build bằng dashboard nếu cần.
- **Tab cũ trước pha A**: lỗi generic; ước tính nhỏ nhưng không đo được chính xác — chấp nhận, ghi nhận trong báo cáo.
- **DNS/cert**: nếu cert production fail lúc boot, backend fail-fast → rollback ngay theo §5.
- **Rollback frontend**: nếu Cloudflare không giữ deployment cũ đủ lâu, phải revert commit + build lại (thêm 2–5 phút).
- **Drain endpoint**: bảo vệ bằng permission + audit; không cho gọi ẩn danh.
