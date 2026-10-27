# [WF-REF-05] Schema `refine_sessions`, state machine, API contract & worker resolve

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:task`
**Status:** Closed
**Assignee:** Agent
**Blocked by:** WF-REF-03, WF-REF-04

---

## Question

Thiết kế kỹ thuật chi tiết để implement phiên luyện cổ. Ticket này là `task` vì phần lớn là chốt cấu trúc dựa trên quyết định ở 03/04, không cần tranh luận sản phẩm.

### 1. State machine
Đề xuất các trạng thái và chuyển tiếp:

```
ACTIVE ──(đủ bước, submit bước cuối)──> READY_TO_CLAIM ──(claim)──> CLAIMED
   │                                          │
   ├──(hết hạn toàn phiên, worker/lazy)──────> EXPIRED ──(claim)──> CLAIMED
   ├──(người chơi abort)──────────────────────> ABORTED
   └──(chết? chỉ giảm tỉ lệ, không đổi trạng thái)
```

Cần chốt:
- Có trạng thái `READY_TO_CLAIM` riêng hay resolve ngay khi submit bước cuối rồi chờ claim?
- `EXPIRED` resolve tự động hay vẫn phải claim? Đề xuất: resolve tự động, claim chỉ để xem kết quả + nhận Cổ.
- Claim idempotent: gọi lại trả cùng kết quả, không tạo Cổ lần hai.
- Ai được claim: chỉ chủ phiên.
- Phiên bị bỏ quên (không claim) có bị dọn sau N ngày không?

### 2. Bảng dữ liệu
Đề xuất `refine_sessions` (khai báo inline trong `backend/internal/db/schema_dictionaries.go` hoặc file `schema_refine.go` mới — chốt theo quy ước repo):

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | UUID PK | |
| `player_id` | UUID FK players | |
| `recipe_id` | TEXT | khớp `target_gu_id/rank/variant` |
| `target_gu_id` / `target_rank` / `variant` | TEXT/INT/INT | để resolve không cần join |
| `gu_instance_ids` | JSONB | Cổ nguyên liệu bị khóa |
| `plan` | JSONB | danh sách bước: type, seed, window_start/end, config |
| `current_step` | INT | |
| `step_results` | JSONB | điểm từng bước, input, thời điểm |
| `accuracy` | NUMERIC | tổng hợp khi resolve |
| `death_penalty` | NUMERIC | mặc định 0 |
| `status` | TEXT | ACTIVE / READY_TO_CLAIM / CLAIMED / ABORTED / EXPIRED |
| `started_at` | TIMESTAMPTZ | mốc tính thời gian |
| `ends_at` | TIMESTAMPTZ | hết hạn toàn phiên |
| `resolved_at` | TIMESTAMPTZ | |
| `resolution` | JSONB | outcome, rate breakdown, Gu mới, message |
| `created_at` | TIMESTAMPTZ | |

Bắt buộc:
- Partial unique index **một phiên active / người chơi**: `WHERE status IN ('ACTIVE','READY_TO_CLAIM')` (hoặc chỉ `ACTIVE`, chốt rõ).
- Index theo `status, ends_at` cho worker sweep.
- FK và cascade phù hợp; không phá constraint hiện có.
- Không dùng `ALTER TABLE` patch; schema mới khai báo inline.

### 3. API contract
Đề xuất (điều chỉnh nếu cần):

| Method | Route | Mục đích |
|---|---|---|
| POST | `/api/refine/sessions` | tạo phiên: recipe + Gu ids; validate, trừ tài nguyên, sinh plan/seed |
| GET | `/api/refine/sessions/active` | lấy phiên đang chạy (nếu có), lazy-resolve nếu quá hạn |
| POST | `/api/refine/sessions/:id/steps` | submit một bước (nonce + action + payload) |
| POST | `/api/refine/sessions/:id/claim` | nhận kết quả, idempotent |
| POST | `/api/refine/sessions/:id/abort` | hủy, hoàn 50% nguyên liệu, Cổ về DYING |
| GET | `/api/refine/preview` | xem dải tỉ lệ trước khi start (nếu WF-REF-03 chốt có) |

Cần chốt:
- Response chung: dùng `respondActionWithInventory` khi claim tạo Cổ; các bước chỉ trả state phiên, không trả toàn inventory.
- HTTP status cho lỗi nghiệp vụ (bước sai thứ tự, quá hạn, nonce sai, phiên không tồn tại).
- Rate limit riêng cho submit bước (chống spam) — ví dụ 60/phút/player.
- Route giữ `AuthRequired + RequireOnboarded`.
- Tương thích: `POST /api/refine` vẫn tồn tại cho đường nhanh (WF-REF-09); nếu request rơi vào diện bắt buộc phiên thì trả lỗi hướng dẫn rõ ràng.

### 4. Transaction & locking
- Tạo phiên: một tx ngắn; lock `players` để trừ tài nguyên/khóa Cổ, rồi nhả.
- Submit bước: một tx ngắn; `SELECT … FOR UPDATE` **row session**, validate step/nonce/window bằng `NOW()`, ghi `step_results`, tăng `current_step`; **không lock `players`**.
- Finalize/claim: tx ngắn; lock session rồi lock player qua `PlayerSvc`; áp dụng công thức, trừ/tạo Cổ, cập nhật status/statuses, publish.
- Tuyệt đối không sleep/giữ tx qua thời gian chờ.
- Chống double-claim: `UPDATE … WHERE status=… RETURNING` hoặc lock row.

### 5. Worker & lazy resolve
- Worker mới `worker/refine_session.go` chạy qua `worker.RunEvery` (ví dụ 15–30 giây), quét `status='ACTIVE' AND ends_at < NOW()`, resolve theo lô nhỏ; mỗi phiên một tx ngắn có `statement_timeout`/`lock_timeout`.
- Lazy-resolve ở `GET /sessions/active` để người chơi thấy kết quả ngay cả khi worker chưa chạy.
- Resolve phải **idempotent** (worker và lazy có thể đụng nhau) — dùng `UPDATE … WHERE status='ACTIVE'` làm guard.
- Ghi notification + publish realtime khi resolve.
- Đăng ký worker trong `cmd/server/main.go`; graceful shutdown theo `context`.

### 6. Realtime event
- Chốt dùng kênh `player:<id>` với event type mới (ví dụ `refine.session`) hay kênh riêng.
- Nếu dùng kênh `"player"`, phải thêm nhánh xử lý tường minh trong `useGameStore.subscribeEvents` trước fallback snapshot (xem WF-REF-07).
- Payload: session id, status, current_step, step result, ends_at, accuracy tạm; **không** lộ seed/đáp án.

### 7. Audit & lịch sử
- Có ghi `gu_history`/audit log cho phiên không? Đề xuất: audit `refine.session.start/submit/claim/abort`, không ghi từng bước vào `gu_history`.
- Có bảng lịch sử riêng hay giữ `resolution` JSONB là đủ?

---

## Resolution

Đã chốt kiến trúc tại [WF-REF-05 Spec](../specs/WF-REF-05-session-architecture.md). Tóm tắt:

- **State machine:** `ACTIVE → RESOLVED` (completed/expired) hoặc `ACTIVE → ABORTED`; `acknowledged_at` chỉ là cờ UI, không phải điều kiện nhận thưởng. Resolve idempotent bằng row lock + guard `status='ACTIVE'`.
- **Schema:** bảng `refine_sessions` trong `schema_refine.go` mới, đăng ký sau `player_items`; `selected_gu_ids TEXT[]`, `gu_snapshot JSONB`, `materials_deducted JSONB`, `rate_snapshot JSONB`, `plan JSONB`, `step_results JSONB`, `result JSONB`; partial unique index một phiên `ACTIVE`/player; index `ends_at` cho worker.
- **Tạo phiên:** một tx ngắn lock player, trừ mana/đá/nguyên liệu, lock Cổ bằng `player_gu.is_locked`, snapshot tỉ lệ, sinh seed/plan, insert; rollback sạch nếu lỗi.
- **Submit bước:** tx ngắn chỉ lock row session (không lock `players`), validate lịch/nonce/payload, chấm server-side, ghi `step_results`. Bước cuối → resolve ở tx thứ hai.
- **Resolve:** tx 5s lock / 15s statement; lock session rồi player; re-check Tiên Cổ bằng `CheckImmortalGuExists`; roll với `raw_positive + accuracyBonus` cap 100% rồi trừ status + death; savepoint quanh insert Tiên Cổ để bắt `IsImmortalConflict` → FAIL_NORMAL.
- **Abort:** hoàn `floor(qty/2)` nguyên liệu, không hoàn mana/đá, unlock Cổ + `hunger=1`, `state='DYING'`, `equipped=false`.
- **Chết:** hook vào `persistDeathDrop`, set `death_penalty=0.25` một lần cho phiên `ACTIVE`.
- **API:** `POST /api/refine/sessions`, `GET /active`, `GET /:id`, `POST /:id/steps`, `POST /:id/abort`, `POST /:id/ack`, `GET /api/refine/preview`; nhóm route `/refine` hiện có.
- **Realtime:** kênh Redis `refine:<playerID>` → client `"refine"`, payload `refine.session`; publish khi create/step/resolve/abort; kèm `PublishPlayerSnapshot` khi inventory/mana đổi.
- **Worker:** `worker.StartRefineSessionWorker` chạy 30s/lần qua `worker.RunEvery`, resolve phiên `ends_at <= NOW()`; lazy-resolve ở GET.
- **Tương thích:** `POST /api/refine` giữ cho cổ rank thấp; guard `REFINE_SESSION_REQUIRED` cho cổ ngang cấp/Tiên Cổ; refactor `computeRefineRate` dùng chung để không lệch công thức.

