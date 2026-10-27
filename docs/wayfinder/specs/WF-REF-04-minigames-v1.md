# [SPECIFICATION] Bộ Minigame v1 cho Phiên Luyện Cổ

**Ticket:** [WF-REF-04: Đặc tả bộ minigame v1: luật chơi, chấm điểm, seed & chống replay](../tickets/WF-REF-04-spec-minigame-v1.md)
**Map:** [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Ngày hoàn thành:** 2026-10-27

---

## 1. Phạm vi

4 dạng minigame v1:

| Mã | Tên | Input | Vai trò |
|---|---|---|---|
| `TAP` | Canh Nhịp | 1 lần tap | Bước khai lò / minigame giữa |
| `SEQUENCE` | Dãy Ấn | Mảng ký tự theo thứ tự | Minigame giữa |
| `CHOICE` | Chọn Hành Động | Chỉ số lựa chọn | Bước dẫn hỏa / bước giữa |
| `ORDER` | Ghép Liệu | Hoán vị danh sách | Minigame giữa |

Không thuộc phạm vi: Hỏa Hậu (`HOLD`) và Phản Xạ (`REACTION`) — để v1.1.

---

## 2. Cấu trúc phiên & lịch chạy

### 2.1 Tham số từ bảng rank

Mỗi rank cung cấp:
- `N` — tổng số bước.
- `K` — số bước là minigame vật lý (`TAP`/`SEQUENCE`/`ORDER`), `1 ≤ K ≤ N`.
- `duration_seconds` — tổng thời lượng phiên.
- `window_seconds` — cửa sổ thao tác mỗi bước.

Bảng hiện hành (WF-REF-06 sẽ đưa vào CSV):

| Rank | N | K | Duration | Window |
|---|---|---|---|---|
| 1 | 3 | 1 | 60s | 20s |
| 2 | 3 | 1 | 120s | 15s |
| 3 | 4 | 2 | 300s | 12s |
| 4 | 4 | 2 | 600s | 10s |
| 5 | 5 | 3 | 1200s | 8s |
| 6 | 5 | 3 | 2700s | 8s |
| 7 | 6 | 4 | 5400s | 6s |
| 8 | 6 | 4 | 7200s | 6s |
| 9 | 7 | 4 | 14400s | 5s |

### 2.2 Thành phần bước

- Bước 1 luôn `CHOICE` ("dẫn hỏa").
- Bước N luôn `TAP` ("khai lò") và được tính vào `K`.
- Bước giữa (2..N−1) gồm:
  - `K − 1` minigame vật lý, chọn từ `TAP` / `SEQUENCE` / `ORDER`.
  - `N − K − 1` bước `CHOICE`.
  - Thứ tự xáo bằng seed server.
- Ràng buộc: hai minigame vật lý liền kề không cùng loại; bước N−1 không được là `TAP`.
- `CHOICE` được phép liền kề `CHOICE` (rank 1 có cấu trúc `[CHOICE, CHOICE, TAP]`).

Ví dụ:
- Rank 1 (N=3, K=1): `CHOICE, CHOICE, TAP`.
- Rank 3 (N=4, K=2): `CHOICE, SEQUENCE, CHOICE, TAP` hoặc `CHOICE, CHOICE, ORDER, TAP`.
- Rank 9 (N=7, K=4): `CHOICE, SEQUENCE, ORDER, CHOICE, TAP, CHOICE, TAP` — 3 minigame giữa + bước cuối `TAP`, bước 6 là `CHOICE` nên không lặp `TAP`.

### 2.3 Lịch cố định từ `started_at`

- `gap = duration_seconds / N`.
- Bước `i` (1-based) mở tại `step_open_at = started_at + (i−1) × gap`.
- Đóng tại `min(step_open_at + window_seconds, ends_at)`.
- `ends_at = started_at + duration_seconds`.
- Bước chưa tới giờ mở: client không được biết params.
- Bước đã đóng: không nhận submit; bước đó 0 điểm.
- Server không dịch chuyển lịch vì người chơi offline.

Hệ quả: rank 1 gần như liên tục; rank cao có khoảng chờ dài giữa các bước — đúng mô hình "phiên chạy nền".

### 2.4 Đồng hồ

- Mọi tính toán dùng `NOW()` phía server.
- Response step trả `server_now`, `step_open_at`, `window_end` để client đồng bộ animation.
- Client hiển thị theo `server_now` + offset ước lượng; không dùng đồng hồ máy làm nguồn chấm.

---

## 3. Hợp đồng input

### 3.1 Request

```json
{
  "step_index": 2,
  "nonce": "8f2c…",
  "action": "TAP | SEQUENCE | CHOICE | ORDER",
  "payload": {},
  "client_ts": 1730000000000
}
```

- `nonce` do server phát cùng params của bước hiện tại, dùng một lần.
- `client_ts` chỉ ghi log/audit; **không** dùng để chấm.
- `payload` tối đa 2KB; validate theo schema từng action.

### 3.2 Validate phía server (theo thứ tự)

1. Session tồn tại, thuộc player, `status = ACTIVE`.
2. `step_index` là bước hiện tại và chưa submit.
3. `nonce` khớp nonce đã phát cho bước, chưa dùng.
4. `NOW()` nằm trong `[step_open_at, window_end + 1s grace]`.
5. `action` khớp loại bước.
6. Payload đúng schema/giới hạn.
7. Chấm điểm từ seed + payload + `NOW()`.

Sai bất kỳ bước nào → `400/409/410` (chi tiết mã thuộc WF-REF-05), không ghi điểm.

### 3.3 Idempotency

- Cùng `nonce` gửi lại: trả **kết quả đã lưu** (200), không chấm lại, không tạo thêm gì.
- Khác nonce cho cùng bước: từ chối.
- Bước đã đóng: từ chối, kể cả khi nonce đúng.

### 3.4 Grace mạng

- Cho phép submit trong **1.000ms** sau `window_end`.
- Thời điểm chấm bị **clamp** về `window_end` nếu vượt.
- Grace không cộng dồn, không áp dụng cho bước đã bị khóa bởi bước sau.

---

## 4. Params công khai cho client

Server chỉ gửi params của **bước hiện tại**; không gửi seed, `correct_index`, hay params bước tương lai.

| Action | Params |
|---|---|
| `TAP` | `marker_period_ms`, `center` (0..1), `half_width`, `step_open_at`, `window_end` |
| `SEQUENCE` | `symbols` (mảng 0..3), `display_ms`, `target_input_ms`, `window_end` |
| `CHOICE` | `situation`, `options` (text đã xáo), `window_end` |
| `ORDER` | `items` (id, name, icon, rank), `window_end` |

---

## 5. Luật & công thức chấm

Mọi điểm làm tròn 2 chữ số, kẹp trong `[0, 100]`.

### 5.1 `TAP` — Canh Nhịp

- Kim chạy tuần hoàn chu kỳ `P = marker_period_ms`.
- Vị trí tại thời điểm `t`:

```
u = ((t − step_open_at) mod P) / P
pos = 1 − |2u − 1|          // 0 → 1 → 0
```

- Tâm `c ∈ [0.2, 0.8]`, nửa độ rộng `w` lấy theo bảng độ khó.
- Người chơi tap **một lần**; `t = NOW()` khi server nhận (đã clamp grace).
- Điểm:

```
d = |pos − c|
score = 100 × max(0, 1 − d / w)
```

- Không tap trong cửa sổ → 0.
- Tap lần hai → từ chối (bước đã khóa).

### 5.2 `SEQUENCE` — Dãy Ấn

- Bộ 4 ký tự `{0,1,2,3}`; chuỗi dài `L` sinh từ seed.
- Hiển thị lần lượt, mỗi ký tự `display_ms`; client được phép phát lại animation khi mở muộn, nhưng submit vẫn phải trước `window_end`.
- Payload: `{"sequence": [0,3,1,…]}`.
- `correct` = số vị trí khớp.
- `base = 100 × correct / L`.
- Nếu `correct == L`:
  - `target = display_total + L × input_per_symbol`.
  - `used = NOW() − step_open_at`.
  - Nếu `used ≤ target` → 100.
  - Ngược lại → `max(80, 100 − 20 × (used/target − 1))`.
- Nếu `correct < L` → `base` (không xét thời gian).

### 5.3 `CHOICE` — Chọn Hành Động

- Server chọn một dòng từ `RefineDecision.csv` theo seed, xáo option.
- Payload: `{"choice_index": 0..2}` (chỉ số hiển thị).
- Điểm: đúng = 100, trung tính = 50, sai = 0, hết giờ = 0.
- Rank 1–2 chỉ dùng 2 option; rank 3+ dùng 3 (nếu dòng CSV thiếu option thứ 3, dùng 2).

### 5.4 `ORDER` — Ghép Liệu

- Chọn `M` vật phẩm (`M` theo bảng độ khó), ưu tiên nguyên liệu trong công thức; thiếu thì bổ sung vật phẩm cùng đường từ registry.
- Thứ tự đúng: **rank vật phẩm tăng dần**, cùng rank thì theo `id` tăng dần.
- Client hiển thị tên + icon + số rank (pips) để người chơi suy luận.
- Payload: `{"order": ["item_id_1", …]}` — phải là hoán vị đúng tập đã phát.
- Điểm:

```
adjacent_correct = số cặp (order[i], order[i+1]) đúng thứ tự
score = 100 × adjacent_correct / (M − 1)
```

- `M < 2` → 100.

---

## 6. Bảng độ khó

| Rank | L chuỗi | display_ms | `w` (TAP) | `P` (TAP) | M (ORDER) | Option (CHOICE) | input_per_symbol |
|---|---|---|---|---|---|---|---|
| 1–2 | 3 | 600 | 0.12 | 2000 | 3 | 2 | 1500ms |
| 3–4 | 4 | 500 | 0.10 | 1700 | 4 | 3 | 1300ms |
| 5–6 | 5 | 450 | 0.08 | 1400 | 5 | 3 | 1200ms |
| 7–8 | 6 | 380 | 0.065 | 1100 | 6 | 3 | 1000ms |
| 9 | 6 | 320 | 0.05 | 900 | 6 | 3 | 900ms |

`w` là nửa độ rộng vùng ăn điểm (tỉ lệ thanh). Rank 1 vùng ăn điểm rộng 24% thanh; rank 9 rộng 10%.

---

## 7. Dữ liệu `RefineDecision.csv`

### 7.1 Schema

```csv
id,situation,option_1,option_2,option_3,correct_index,neutral_index,explanation
```

- `correct_index` / `neutral_index`: chỉ số 0-based trong 3 option gốc.
- `explanation`: hiển thị sau khi chấm (nếu UI bật).
- 10–12 dòng generic, không phụ thuộc 49 lưu phái ở v1.
- Import qua Admin; registry đọc khi tạo phiên (cache hoặc DB read).

### 7.2 Ví dụ nội dung

| situation | option_1 | option_2 | option_3 | correct | neutral |
|---|---|---|---|---|---|
| Hỏa hậu bỗng bùng lên, lò rung nhẹ | Tăng hỏa | Hạ hỏa | Giữ nguyên | 1 | 2 |
| Nguyên liệu tan chậm hơn dự kiến | Thêm dung môi | Chờ thêm | Tăng nhiệt | 0 | 1 |
| Khói trong lò chuyển màu xám | Mở lò ngay | Niêm phong lò | Hạ hỏa từ từ | 2 | 1 |

### 7.3 Fallback

Nếu registry rỗng:
- Ghi log lỗi cấu hình.
- Các bước `CHOICE` chuyển thành `TAP` cùng độ khó (không chặn người chơi).
- Admin vẫn phải import CSV để có trải nghiệm đầy đủ.

---

## 8. Chống replay & chống gian lận

### 8.1 Đã có
- Seed sinh bằng `crypto/rand` khi tạo phiên, chỉ lưu server.
- Nonce mỗi bước, một lần, gắn session + step.
- Chỉ gửi params bước hiện tại, không gửi đáp án/`correct_index`/seed.
- Một bước một submit, validate thời gian bằng `NOW()`.
- Rate limit submit (đề xuất 30/phút/player; chi tiết WF-REF-05).
- Audit metadata: `score`, `receipt_offset_ms`, `action`, `step_index`.

### 8.2 Rủi ro chấp nhận
- Client phải biết vùng ngọt / chuỗi ấn / option để hiển thị → client sửa có thể gửi input hoàn hảo.
- Vì accuracy chỉ là phần phụ (+4–8% tỉ lệ), động cơ bot thấp.
- Không xây hệ thống chống bot thời gian thực ở v1.

### 8.3 Có thể thêm sau (fog)
- Phát hiện entropy nhịp input bất thường.
- Cảnh báo khi cùng tài khoản đạt 100 điểm liên tục nhiều phiên.
- Server-side minigame qua WebSocket nếu kinh tế bị lạm dụng.

---

## 9. Kiểm thử bắt buộc

- Sinh bước đúng cho toàn bộ 9 dòng bảng rank: đủ N, đúng K, bước 1 `CHOICE`, bước N `TAP`, không lặp minigame liền kề, bước N−1 ≠ `TAP`.
- Cùng seed → cùng kế hoạch bước.
- `TAP`: tâm vùng → 100; mép vùng → 0; ngoài vùng → 0; clamp grace.
- `SEQUENCE`: đúng toàn bộ nhanh → 100; đúng toàn bộ chậm → ≥80; đúng một nửa → 50.
- `CHOICE`: đúng/trung tính/sai → 100/50/0.
- `ORDER`: hoán vị đúng → 100; đảo 1 cặp → `100×(M−2)/(M−1)`; sai hoàn toàn → 0.
- Nonce dùng lại trả kết quả cũ, không chấm lại.
- Submit sai bước/sai action/sai payload bị từ chối.
- Bước quá hạn bị từ chối.
- Không rò seed/`correct_index` trong response.
- CSV thiếu/sai → fallback `CHOICE` → `TAP` + log.
- Điểm luôn trong `[0,100]`, làm tròn 2 chữ số.

---

## 10. Việc chưa chốt (chuyển ticket khác)

- Chấm accuracy tổng → tỉ lệ: WF-REF-03 (đã xong).
- Schema, nonce storage, API, worker: WF-REF-05.
- Bảng rank/độ khó vào CSV + Admin: WF-REF-06.
- UI component, đồng bộ đồng hồ, realtime: WF-REF-07.
- Prototype Canh Nhịp: WF-REF-08.
