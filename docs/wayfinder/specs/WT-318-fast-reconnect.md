# WT-318 — Fast Reconnect & Session Resume khi mạng đổi

> Ticket: [Spec: fast reconnect & session resume khi mạng đổi](https://github.com/VipPro67/GuWorld/issues/318) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Phụ thuộc: [Chốt mô hình session & kênh realtime](https://github.com/VipPro67/GuWorld/issues/311), [Spec: client rewrite sang WebTransport](https://github.com/VipPro67/GuWorld/issues/316)

## 0. Vấn đề hiện tại

- `controllerID = uuid.New()` mỗi lần mở WS (`backend/internal/ws/client.go:159`) ⇒ reconnect luôn là session mới.
- `Admission.Disconnect` nhả lease ngay (`admission.go:1230`) ⇒ join lại trong lúc socket cũ chưa dọn xong nhận `409 realtime_character_already_owned`; client retry 300 ms→2 s (tối đa 7 lần ≈ 7 s).
- `world.Manager.ReplaceCharacterLease()` (`manager.go:1875`) đã là **đường reconnect nguyên tử** (giữ entity, đổi token) nhưng **chưa được gọi ngoài test**.
- `sim` chỉ nhận input khi `seq > entity.AckSeq` (`sim/world.go:655`) ⇒ resume giữ entity mà client reset `seq = 0` sẽ mất input.

## 1. Quyết định

### 1.1 Resume = grace server-side theo `(playerID, tab_id)`, không thêm credential

- `Admission` thêm `suspended map[suspendKey]session`, `suspendKey{playerID, tabID}`.
- Khi socket đóng (`Disconnect`): session chuyển sang **suspended**, **giữ lease + entity**, không checkpoint/nhả ngay.
- Khi join lại cùng JWT + cùng `tab_id` trong grace: gọi `manager.ReplaceCharacterLease(oldLease, newControllerID.String())`, cập nhật session sang controller mới, trả `WELCOME` với entity cũ (vị trí/HP/status/cooldown còn nguyên).
- `tab_id` lấy từ query `?tab_id=` (đã có ở #311), lưu vào session.
- **Không có resume token ký riêng.** JWT xác thực player; `tab_id` chỉ chọn session zombie của chính player đó; lease token không bao giờ ra client. Replay `tab_id` bởi cùng tài khoản không cấp thêm quyền.
- Multi-instance/cross-process **không hỗ trợ** (trạng thái nằm process-local) — nằm trong fog multi-instance của map.

### 1.2 Grace

| Loại zone | Grace |
|---|---|
| Zone mở (thành phố, bãi quái, bí cảnh shared) | **15 s** |
| Instance riêng (`combat:`, `aperture_tribulation`, practice/arena) | **30 s** |

- Hết grace: `flushState` → `LeaveCharacter` → xoá session (đúng luồng `Disconnect` hiện tại). Checkpoint lỗi ⇒ `markClosing`, worker định kỳ retry (đã có).
- Hết grace khi đang `transferring`: không suspend; transfer owner xử lý như hiện tại.
- Server restart/crash: mất suspended state; join lại = session mới tại checkpoint gần nhất (chấp nhận; ghi rõ trong UX).

### 1.3 Entity trong lúc suspended

- Giữ nguyên entity trong sim, **không despawn**.
- **Không nhận damage, không bị AI nhắm** (cờ `sim.Entity.Suspended`, damage/AI bỏ qua), tránh chết oan khi mạng 4G chập chờn.
- Status/cooldown/hunger **vẫn tick** theo sim ⇒ không tạo lợi thế treo buff.
- Không nhận input (client biến mất). Vị trí đứng yên.
- Resume/hết grace đều phải clear cờ.

### 1.4 Takeover vs resume

| Join mới | Hành vi |
|---|---|
| Cùng `(player, tab_id)`, session suspended còn hạn | **Resume**: `ReplaceCharacterLease`, WELCOME entity cũ |
| Khác `tab_id`, session cũ còn sống | **Takeover**: `ReplaceCharacterLease` từ session cũ, đóng controller cũ bằng `4001 session_replaced`; không còn `409` race |
| Khác `tab_id`, session cũ suspended | Takeover: replace lease từ suspended, xoá suspended, tab cũ nhận `4001` ở lần kết nối sau (claim-tab HTTP đã có) |
| Không có session, nhưng `manager.OwnsCharacter` (trạng thái lạ) | Giữ `409 realtime_character_already_owned` làm fallback an toàn |

Mã đóng giữ theo #311: `4000 superseded_same_tab` (cùng tab mở kết nối mới, F5 khi socket cũ chưa chết), `4001 session_replaced` (tab khác).

### 1.5 `seq` / ack sau resume

- Thêm `ack_seq` vào `WELCOME` (wire v2 — chưa thi công nên mở rộng ngay trong spec #313).
- Client seed `nextInputSeq = max(0, welcome.ack_seq)`; **không reset về 0**.
- Snapshot `ack_seq` tiếp tục là cumulative ack duy nhất; **không cần ack/nak từng input** sau resume.
- `SnapshotBuffer` reset baseline theo `WELCOME`; predictor bỏ toàn bộ pending cũ, replay bắt đầu từ seq mới.

## 2. Luồng server

```
WS close ──► Admission.Disconnect
              ├─ transferring?            → xử lý như cũ (transfer owner)
              ├─ flushState lỗi           → markClosing (worker retry), không suspend
              └─ OK                       → suspend: giữ lease, set Entity.Suspended
                                            expiresAt = now + grace(zone)

WS join ──► Admission.Join(player, newController, tabID)
              ├─ suspended match + còn hạn → ReplaceCharacterLease
              │                               clear Suspended, xoá suspended,
              │                               sessions[newController] = session
              │                               trả WELCOME (ack_seq hiện tại)
              ├─ session khác tab           → ReplaceCharacterLease + close 4001
              ├─ suspended hết hạn          → finalize (flush+leave+delete) rồi join mới
              └─ owns character (fallback)  → 409 realtime_character_already_owned
```

- Sweeper: `worker.RunEvery(1s)` quét `suspended` hết hạn → finalize. Không hand-roll ticker (theo AGENTS).
- `joinLimitPerMinute` hiện có vẫn áp dụng; resume không được tính là "join attempt mới" nếu cùng `(player, tab_id)` (tránh tự bóp mình khi mạng nhấp nháy) — hoặc nâng limit riêng cho resume.
- `Admission.ActiveSessionCount`/metrics đếm cả suspended, tách nhãn.

## 3. Luồng client

| Thời điểm | Hành vi |
|---|---|
| `transport.closed` / mất frame quá watchdog | Thử lại **ngay** (attempt 0, không chờ) |
| Attempt ≥ 1 | Backoff `300ms × 1.6^n`, cap **5 s**, jitter ±20 % |
| Trong grace | Không giới hạn cứng số lần retry; vẫn giữ `tab_id`, join cùng zone/version |
| Sau 3 s kể từ lúc mất | Hiện banner "Đang kết nối lại…" (không chặn UI khác) |
| Hết grace mà vẫn fail | Overlay "Mất kết nối" + nút **Thử lại** (join mới, entity mới) |
| `4001` | Dừng hẳn, `game:tab-replaced`, không retry |
| `4002` | Dừng, yêu cầu tải lại trang |
| `4000` | Im lặng; instance cũ tự chết |
| Tab ẩn | Không chạy watchdog; khi visible lại nếu quá hạn thì reconnect ngay |

- Client phải giữ `lastJoin` (zone/version) và `tab_id` xuyên reconnect.
- Resume thành công phát event `resume_ok`; thất bại do hết grace phát `resume_expired` (telemetry).

## 4. Đo lường

- **Server counters** (`internal/monitor`, admin monitor):
  - `resume_attempts`, `resume_success`, `resume_expired`, `takeover_replace`, `resume_errors_by_code`.
  - `resume_latency_ms` p50/p95: từ `suspendedAt` đến lúc join lại thành công.
- **Client** (#308 collector mở rộng cửa sổ):
  - `resumes_ok`, `resume_failures`, `resume_last_ms`.
- Kịch bản đo bắt buộc: mất mạng **3 s** và **10 s** (rút WiFi / bật airplane mode), lặp 20 lần mỗi loại, trên WiFi + 4G; ghi p95 thời gian vào lại và tỉ lệ thất bại. Mục tiêu: **p95 < 2 s** sau khi mạng trở lại, tỉ lệ resume thành công **> 95 %** trong grace.

## 5. Bảo mật & biên

- `tab_id` là UUID random trong `sessionStorage`; **không** log ra ngoài, không đưa vào query log production (spec ingress #315 phải redact `token`, cân nhắc redact `tab_id`).
- **Tab duplicate**: Chrome "Duplicate tab" copy `sessionStorage` ⇒ 2 tab cùng `tab_id`. Phòng ngừa: trước khi claim, dùng `BroadcastChannel("gu-tab-<player>")` ping; nếu có tab khác cùng `tab_id` đang sống, **rotate `tab_id`** rồi claim tab mới (tab cũ sẽ nhận `4001`). Ghi vào phần client của #316 khi thi công.
- Không resume khi `server restart`, khi zone bị idle-evict, hoặc khi character đã chết (`HP <= 0`) — join mới theo luật hiện tại.
- Chống lạm dụng "bất tử": grace tối đa 15/30 s, entity không nhận input, statuses vẫn tick; log khi resume lặp lại bất thường (`resume_attempts` quá cao/giờ).
- Nếu `ReplaceCharacterLease` trả `ErrInvalidLease` (lease đã bị transfer/checkpoint khác lấy), fallback: finalize suspended rồi join mới; không trả 500.

## 6. Checklist thi công

1. `Admission`: thêm `tabID` vào session, `suspended` map, suspend trong `Disconnect`, sweeper 1 s, nhánh resume/takeover trong `Join`.
2. `world.Manager`: gọi `ReplaceCharacterLease` (đã có); thêm `SetCharacterSuspended(lease, bool)` queue mutation set cờ.
3. `sim`: cờ `Suspended` — AI bỏ target, damage bỏ qua, input bỏ qua.
4. Protocol v2: `WELCOME.ack_seq`; client seed `nextInputSeq`.
5. Hub: takeover cùng player khác tab đóng controller cũ `4001`; cùng tab đóng `4000`.
6. Client: state machine #316 §6 + retry ngay/backoff/overlay + `resume_ok`/`resume_expired` + BroadcastChannel duplicate-tab.
7. Telemetry server + client như §4; test:
   - Unit: suspend→resume giữ entity/HP/vị trí; sweeper hết hạn finalize; takeover khác tab; resume với ack_seq cao không mất input.
   - Integration: fake transport đóng/mở lại trong grace; 3 s và 10 s.
   - Race test `-race` cho suspend/expire/resume đồng thời.

## 7. Việc chuyển giao

- Wire `WELCOME.ack_seq` → cập nhật [Spec: wire protocol v2 (binary, datagram, versioning)](https://github.com/VipPro67/GuWorld/issues/313) khi thi công (field mới, không đổi version).
- Client state machine/duplicate-tab → [Spec: client rewrite sang WebTransport](https://github.com/VipPro67/GuWorld/issues/316).
- Redact `tab_id`/`token` trong access log → [Spec: ingress & deploy mới (bỏ Caddy)](https://github.com/VipPro67/GuWorld/issues/315).
