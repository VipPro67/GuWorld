# WT-312 — UX cho client không hỗ trợ WebTransport

> Ticket: [Chốt UX cho client không hỗ trợ WebTransport](https://github.com/VipPro67/GuWorld/issues/312) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Liên quan: [Spec: client rewrite sang WebTransport](https://github.com/VipPro67/GuWorld/issues/316), [Spec: ingress & deploy mới (bỏ Caddy)](https://github.com/VipPro67/GuWorld/issues/315)

## 0. Bối cảnh

Sau khi bỏ WebSocket, client không có WebTransport sẽ **không vào được thế giới realtime**. Phạm vi ảnh hưởng: iOS Safari < 26.4, Chrome/Edge < 97, Firefox < 114, WebView trong app (Zalo/Facebook/Instagram), mạng chặn UDP/443 (Wi-Fi công cộng, VPN, nhà mạng hạn chế). Browser support ~91.6 % global; phần lớn thiệt hại dự kiến nằm ở iOS cũ và WebView.

## 1. Quyết định

### 1.1 Ba trạng thái, xử lý khác nhau

| Trạng thái | Cách phát hiện | Hành vi |
|---|---|---|
| `unsupported` | `typeof WebTransport === "undefined"` hoặc constructor throw | Chặn realtime **cả phiên**, không retry; hiện hướng dẫn đổi/cập nhật trình duyệt |
| `blocked` | API có, nhưng `transport.ready` không resolve trong **8 s**, hoặc `transport.closed` trước khi ready | Retry **3 lần trong 30 s** (0/5/15 s); vẫn fail → chặn realtime cả phiên + hướng dẫn đổi mạng/tắt VPN; nút "Thử lại" reset |
| `supported` | `ready` resolve và nhận frame đầu tiên | Chạy bình thường |

- Kiểm tra **trước khi mount Phaser** để không tải map/asset vô ích.
- Tự kiểm tra lại khi `navigator.connection` đổi hoặc tab visible lại (throttle 1 lần/phút) — chỉ khi đang `blocked`, không khi `unsupported`.
- Không dùng UA để cấp quyền; UA/platform chỉ dùng cho telemetry.

### 1.2 Phạm vi: HTTP-only mode, không chặn cả game

| Vẫn dùng được (HTTP) | Bị khóa (cần WT) |
|---|---|
| Đăng nhập, hồ sơ, inventory, tu luyện, chợ, bang phái, wiki, nhiệm vụ | Bản đồ world/Phaser, combat realtime, arena/match |
| Chat gửi/đọc qua HTTP (`/chat/bao-hoang/send`, `/messages` polling 5 s) | Chat realtime/private, notification, trade live, refine live |

- `canJoinMMOZone` đổi thành phụ thuộc `capability === "supported"`.
- Nút vào thế giới bị disable + tooltip lý do; route world hiện overlay tương ứng thay vì màn hình trắng.
- Banner cố định trong game shell khi HTTP-only: **"Chế độ không realtime — bản đồ thế giới tạm khóa."**
- Ẩn/disable chuông notification realtime; chat overlay chuyển sang polling HTTP.

### 1.3 Copy tiếng Việt

| Tình huống | Tiêu đề | Nội dung | Nút |
|---|---|---|---|
| `unsupported` | **Trình duyệt này chưa hỗ trợ WebTransport** | "Thế giới realtime cần Chrome/Edge 97+, Firefox 114+ hoặc Safari 26.4+. Bạn vẫn dùng được chợ, tu luyện và các tính năng khác." | **Cập nhật trình duyệt** (link theo browser) + **Tiếp tục không realtime** |
| WebView (`Zalo`, `FBAN`, `FBAV`, `Instagram`, `Line`) | **Hãy mở bằng Chrome hoặc Safari** | "Trình duyệt trong ứng dụng không hỗ trợ chế độ chơi realtime. Sao chép liên kết và mở bằng Chrome/Safari để vào thế giới." | **Sao chép liên kết** + **Tiếp tục không realtime** |
| `blocked` | **Không kết nối được thế giới realtime** | "Mạng của bạn có thể đang chặn UDP/443 — thường gặp ở Wi-Fi công cộng, VPN hoặc mạng hạn chế. Hãy thử Wi-Fi/4G khác hoặc tắt VPN." | **Thử lại** + **Tiếp tục không realtime** |
| Banner HTTP-only | — | "Chế độ không realtime — bản đồ thế giới tạm khóa." | (không nút) |
| Sau grace resume thất bại (#318) | **Mất kết nối** | "Không thể vào lại thế giới. Kiểm tra mạng rồi thử lại." | **Thử lại** + **Bảng nhân vật** |

Nút "Tiếp tục không realtime" đóng overlay, đặt cờ phiên `gu_realtime_opt_out=1` (sessionStorage) để không hỏi lại mỗi lần mở panel.

### 1.4 Không có fallback WS runtime

- Không thêm `NEXT_PUBLIC_REALTIME_MODE=ws|wt`, không giữ code WebSocket.
- Rollback là việc của quy trình deploy (#309): deploy lại image cũ. UI chỉ có "Thử lại" và "Tiếp tục không realtime".

## 2. Telemetry đo thiệt hại (qua HTTP, không cần WT)

### 2.1 Client

- Sau khi phân loại, gửi **1 lần / player / ngày** (localStorage `gu_capability_report_day`):
  `POST /api/realtime/capability` (Auth):
  ```json
  { "status": "supported|unsupported|blocked", "reason": "missing_api|webview|ready_timeout|closed_before_ready|quic_blocked", "platform": "ios|android|webview|desktop|other", "effective_type": "wifi|4g|3g|slow-2g|unknown", "ready_ms": 350 }
  ```
- `supported` cũng gửi (sau lần kết nối WT đầu tiên thành công) để có **mẫu số** tính % bị chặn.
- Không gửi UA trong body; server lấy `User-Agent` từ header (giới hạn 256 ký tự).

### 2.2 Server

- Bảng mới trong `backend/internal/db/schema_realtime.go`:
  ```sql
  CREATE TABLE IF NOT EXISTS realtime_client_capability (
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    day DATE NOT NULL DEFAULT CURRENT_DATE,
    status VARCHAR(16) NOT NULL,
    reason VARCHAR(64) NOT NULL DEFAULT '',
    platform VARCHAR(24) NOT NULL DEFAULT 'other',
    user_agent VARCHAR(256) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (player_id, day, status, reason)
  );
  CREATE INDEX IF NOT EXISTS idx_realtime_capability_day ON realtime_client_capability(day);
  ```
  Upsert `ON CONFLICT ... DO UPDATE`, validate `status`/`platform`/`reason` theo allowlist, giới hạn body 8 KiB.
- Endpoint player: `POST /api/realtime/capability` (Auth, rate limit 6/phút/IP).
- Endpoint admin: `GET /api/admin/monitor/realtime/capability?days=7` (`monitoring.read`) trả:
  ```json
  {
    "days": 7,
    "totals": { "supported": 812, "unsupported": 23, "blocked": 11 },
    "blocked_pct": 4.0,
    "by_platform": [{ "platform": "ios", "supported": 120, "unsupported": 20, "blocked": 3 }],
    "by_reason": [{ "reason": "missing_api", "count": 20 }],
    "series": [{ "day": "2026-11-02", "supported": 200, "unsupported": 6, "blocked": 2 }]
  }
  ```
- Retention: worker xoá row cũ hơn **30 ngày** (`worker.RunEvery` hằng ngày).
- Mẫu số là **distinct player có report `supported` trong ngày**; `blocked_pct = (unsupported + blocked) / tổng report`.
- Ngưỡng cảnh báo: **> 8 % tổng** hoặc **> 20 % trên iOS** trong 3 ngày liên tiếp ⇒ xem lại quyết định bỏ WS (ghi log warn + hiển thị badge đỏ ở admin).

## 3. Điểm sửa client

| File | Thay đổi |
|---|---|
| `frontend/src/lib/realtime/capability.ts` (mới) | `detectCapability()`, `classifyConnectFailure()`, webview UA detect, trạng thái module + subscribe |
| `frontend/src/lib/mmoConfig.ts` | `canJoinMMOZone` = config && capability === "supported" |
| `frontend/src/components/game/PhaserWorldHost.tsx` | Overlay 3 trạng thái + copy §1.3; không mount Phaser khi unsupported; nút thử lại reset bộ đếm |
| `frontend/src/components/game/MMOHud.tsx` / game shell | Banner HTTP-only, disable nút vào thế giới, chat polling HTTP, ẩn notification |
| `frontend/src/store/useGameStore.ts` | Cờ `realtimeCapability`, gửi report 1 lần/ngày, fallback chat HTTP |
| `frontend/src/lib/realtime/RealtimeClient.ts` (từ #316) | Map lỗi handshake → `blocked` reason; gọi reporter |

## 4. Test

- Unit `capability.ts`: giả lập `navigator` không có WebTransport / constructor throw / ready timeout → đúng trạng thái + reason; UA WebView (Zalo/FBAN/Instagram/Line) → `webview`.
- Unit: retry 3 lần đúng mốc 0/5/15 s; nút "Thử lại" reset; `connection.change` re-probe khi `blocked`, không khi `unsupported`.
- Go: handler validate allowlist, upsert idempotent trong ngày, body quá lớn → 400, admin aggregation đúng `%`, retention xoá > 30 ngày.
- UI: snapshot/overlay cho 3 trạng thái; banner HTTP-only; nút vào thế giới disabled.
- Thủ công: Chrome 96 (unsupported), Safari iOS 26.3 (unsupported), Wi-Fi chặn UDP/VPN (blocked), Chrome mới (supported).

## 5. Điều kiện hoàn thành

- Không tải Phaser/map asset khi `unsupported`.
- Có đủ 3 trạng thái + copy §1.3; "Tiếp tục không realtime" hoạt động.
- Report capability 1 lần/ngày; admin endpoint trả `%` và series; retention chạy.
- HTTP-only mode: chợ/tu luyện/inventory/chat HTTP dùng được, world bị khóa rõ ràng.
- `npm test`, lint, build pass; Go test pass.

## 6. Docs phải sửa

| File | Nội dung |
|---|---|
| `docs/API.md` | 2 route mới (`/api/realtime/capability`, `/api/admin/monitor/realtime/capability`) |
| `docs/GAME_DATA.md` | Bảng `realtime_client_capability` |
| `WIKI.md` | Người chơi: yêu cầu trình duyệt/mạng, chế độ không realtime |
| `README.md` / `docs/DEVELOPMENT.md` | Ghi chú browser support nếu cần |

## 7. Out of scope

- Fallback WebSocket runtime hoặc chế độ "legacy transport" theo query flag.
- Block toàn bộ game / trang maintenance cho trình duyệt cũ.
- Native wrapper để né giới hạn browser.
