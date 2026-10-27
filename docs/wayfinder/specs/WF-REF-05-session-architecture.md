# [SPECIFICATION] Kiến trúc Phiên Luyện Cổ: Schema, State Machine, API & Worker

**Ticket:** [WF-REF-05: Schema `refine_sessions`, state machine, API contract & worker resolve](../tickets/WF-REF-05-schema-state-machine-api.md)
**Map:** [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Ngày hoàn thành:** 2026-10-27

---

## 1. Mục tiêu & ràng buộc

- Một phiên luyện sống trên server, chạy theo `started_at`, không phụ thuộc client online.
- Không giữ lock row `players` qua thời gian chờ. Chỉ lock player trong transaction tạo phiên và transaction resolve/abort.
- Một người chơi chỉ có **một phiên `ACTIVE`**.
- Server-authoritative: seed, kế hoạch bước, cờ chết, điểm số đều do server giữ.
- Tương thích ngược: `POST /api/refine` vẫn là đường nhanh cho cổ rank thấp.
- Tái dùng pattern `player_instance_sessions` (state machine) + `tribulation` (deadline worker) + `variant_refinement` (lock ngắn).

---

## 2. State machine

```
                 ┌──────────────────────────────┐
                 │            ACTIVE            │
                 └───────┬───────────┬──────────┘
        submit bước cuối │           │ ends_at <= NOW()
                         │           │ (worker hoặc lazy)
                         ▼           ▼
                 ┌──────────────────────────────┐
                 │           RESOLVED           │
                 │  result.reason = completed   │
                 │                 | expired    │
                 └──────────────┬───────────────┘
                                │ POST /ack
                                ▼
                     acknowledged_at != NULL
                     (status giữ nguyên)

     ACTIVE ──(POST /abort)──> ABORTED
```

- **ACTIVE**: nhận submit bước trong cửa sổ; chưa có kết quả.
- **RESOLVED**: đã roll + áp dụng toàn bộ hiệu ứng (trừ/tạo Cổ, mastery, status, notification). Không cần "claim" để nhận thưởng.
- **ABORTED**: người chơi hủy; đã hoàn 50% nguyên liệu, trả Cổ về Hấp Hối.
- `acknowledged_at` chỉ là cờ UI đã xem kết quả, không ảnh hưởng phần thưởng.
- Không có trạng thái `EXPIRED` riêng: hết hạn resolve thành `RESOLVED` với `result.reason = "expired"`.

**Resolve idempotent**: mọi entry point (submit bước cuối, worker, lazy GET) đều mở transaction, `SELECT … FOR UPDATE` row session, kiểm tra `status = 'ACTIVE'`; ai thấy status khác thì trả kết quả hiện có, không chạy lại.

---

## 3. Schema

### 3.1 File & đăng ký

- Tạo mới `backend/internal/db/schema_refine.go`, hàm `schemaRefine() error`.
- Đăng ký trong `backend/internal/db/db.go` → `InitSchema()` `steps` **sau** `player_items`:

```go
{"refine", schemaRefine},
```

- Không dùng `ALTER TABLE` patch. Mọi cột khai báo inline trong `CREATE TABLE`.

### 3.2 DDL

```sql
CREATE TABLE IF NOT EXISTS refine_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id           UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,

    recipe_id           VARCHAR(100) NOT NULL,
    target_gu_id        VARCHAR(50)  NOT NULL,
    target_rank         INTEGER      NOT NULL CHECK (target_rank BETWEEN 1 AND 9),
    variant             INTEGER      NOT NULL DEFAULT 0,

    status              VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                        CHECK (status IN ('ACTIVE', 'RESOLVED', 'ABORTED')),

    selected_gu_ids     TEXT[]       NOT NULL,
    gu_snapshot         JSONB        NOT NULL DEFAULT '[]',
    materials_deducted  JSONB        NOT NULL DEFAULT '{}',
    rate_snapshot       JSONB        NOT NULL,

    plan                JSONB        NOT NULL,
    step_results        JSONB        NOT NULL DEFAULT '{}',

    accuracy            DOUBLE PRECISION NOT NULL DEFAULT 0,
    death_penalty       DOUBLE PRECISION NOT NULL DEFAULT 0,
    death_marked_at     TIMESTAMPTZ,

    result              JSONB,

    started_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ends_at             TIMESTAMPTZ  NOT NULL,
    resolved_at         TIMESTAMPTZ,
    acknowledged_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS refine_sessions_one_active
    ON refine_sessions (player_id)
    WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_refine_sessions_due
    ON refine_sessions (ends_at)
    WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_refine_sessions_player_created
    ON refine_sessions (player_id, created_at DESC);
```

### 3.3 Ghi chú thiết kế

- **Không FK tới `refining_recipes` / `gu_dictionary`**: phiên là snapshot lịch sử; Admin re-import CSV không được làm hỏng phiên đang chạy.
- **`selected_gu_ids` kiểu `TEXT[]`** để dùng `ANY($1::text[])` khi lock/unlock/delete.
- **`gu_snapshot`** giữ `models.GuInstance` tại thời điểm tạo (gồm `IsVital`) để xử lý failure/abort đúng trạng thái ban đầu.
- **`materials_deducted`** là `map[string]int64` để hoàn 50% khi abort.
- **`rate_snapshot`** là snapshot tỉ lệ tại thời điểm tạo; xem §7.
- **`plan`** là server-only, tuyệt đối không trả nguyên object cho client.
- **`step_results`** là map key string `"1".."N"` → kết quả từng bước.
- `gen_random_uuid()` đã có sẵn (`uuid-ossp`/PG 13+). Repo dùng `gen_random_uuid()` ở nhiều bảng.

---

## 4. JSON shapes

### 4.1 `rate_snapshot`

```json
{
  "breakdown": { "...RefineRateBreakdown cũ..." : 0 },
  "raw_positive_rate": 0.72,
  "status_penalty": 0.0,
  "refinement_realm": 2,
  "insight_stacks": 3,
  "immortal_penalty": 1.0,
  "accuracy_max_bonus": 0.07,
  "death_penalty": 0.25,
  "mastery_multiplier_max": 2.0
}
```

- `raw_positive_rate = base_rate + toàn bộ bonus dương` **trước** cap 100%, đã chia `ImmortalPenalty`.
- `status_penalty` = tổng debuff `refine_fatigue`/`refine_backfire` tại thời điểm tạo.
- `breakdown.final_rate` lưu tỉ lệ tại thời điểm tạo (chưa có accuracy/death) để hiển thị lịch sử.
- `accuracy_max_bonus`, `death_penalty`, `mastery_multiplier_max` là snapshot từ `refine_stage_templates` (WF-REF-06) để import giữa phiên không đổi kết quả.

### 4.2 `plan`

```json
{
  "seed": "hex-32-bytes",
  "n": 5,
  "k": 3,
  "gap_ms": 240000,
  "window_ms": 8000,
  "steps": [
    {
      "i": 1,
      "type": "CHOICE",
      "seed": "hex",
      "nonce": "uuid",
      "window_start": "2026-10-27T10:00:00Z",
      "window_end": "2026-10-27T10:00:08Z",
      "config": {
        "decision_id": 4,
        "option_order": [2, 0, 1],
        "correct_display_index": 1,
        "neutral_display_index": 0
      }
    },
    {
      "i": 2,
      "type": "SEQUENCE",
      "seed": "hex",
      "nonce": "uuid",
      "config": {
        "symbols": [0, 3, 1, 2],
        "display_ms": 450,
        "target_input_ms": 4800
      }
    },
    {
      "i": 5,
      "type": "TAP",
      "seed": "hex",
      "nonce": "uuid",
      "config": {
        "center": 0.43,
        "half_width": 0.08,
        "marker_period_ms": 1400
      }
    }
  ]
}
```

- `config` chứa đáp án/symbol/center — **chỉ server đọc**.
- Client chỉ nhận `publicStep(step)` đã lọc (xem §9.1).

### 4.3 `step_results`

```json
{
  "1": {
    "score": 100,
    "action": "CHOICE",
    "payload": {"choice_index": 1},
    "submitted_at": "2026-10-27T10:00:04Z",
    "receipt_offset_ms": 3412
  },
  "2": {
    "score": 75,
    "action": "SEQUENCE",
    "payload": {"sequence": [0,3,1,0]},
    "submitted_at": "2026-10-27T10:04:10Z",
    "receipt_offset_ms": 2110
  }
}
```

- `receipt_offset_ms = submitted_at − window_start` (dùng audit/chống gian lận).
- Không lưu `client_ts` trong cột riêng; ghi vào audit metadata.

### 4.4 `result`

```json
{
  "reason": "completed",
  "outcome": "SUCCESS",
  "message": "Hợp luyện thành công! ... (Tỉ lệ: 68.0%)",
  "new_gu": { "...models.GuInstance..." : null },
  "rate_breakdown": {
    "...các trường cũ...": 0,
    "accuracy": 82.5,
    "accuracy_bonus": 0.052,
    "accuracy_max_bonus": 0.07,
    "death_penalty": 0.0,
    "step_scores": [100, 75, 80, 0, 90]
  }
}
```

- `reason ∈ {completed, expired, aborted}`.
- Với abort: `outcome` vắng mặt, `message` mô tả phần mất/hoàn.

---

## 5. Luồng tạo phiên

Endpoint: `POST /api/refine/sessions`

### 5.1 Validate
Giống `RefineGu` hiện tại, bổ sung:
1. Player không chết (`gamefx.IsDeadState`).
2. Recipe đã mở 100% (`RecipeSvc.Get`).
3. Không có phiên `ACTIVE` (unique index bắt, trả `409 REFINE_SESSION_ACTIVE`).
4. Đủ mana, Nguyên Thạch, Tiên Nguyên Thạch, nguyên liệu.
5. Đúng số lượng/loại Cổ, không `DEAD`, **không `is_locked`**.
6. `target_rank >= player.rank || target_rank >= 6` (nếu không, trả `REFINE_QUICK_PATH_AVAILABLE` hoặc cho phép tạo — chốt: cho phép tạo; UI tự chọn đường nhanh).

### 5.2 Escrow trong một transaction
1. `evaluateManaTx(tx, playerID)` — lock player, project mana.
2. Trừ `CurrentMana`, `PrimevalStones`, `ImmortalStones`.
3. Trừ nguyên liệu bằng `InventorySvc.DeductItem`; lưu `materials_deducted`.
4. Lock Cổ:

```sql
UPDATE player_gu
SET is_locked = true, updated_at = NOW()
WHERE player_id = $1 AND id = ANY($2::text[]) AND NOT is_locked;
```

Kiểm tra `RowsAffected == len(selected_gu_ids)`; lệch → rollback + `409 REFINE_GU_LOCKED`.

5. Tính `rate_snapshot` bằng helper dùng chung (xem §7).
6. Sinh seed (`crypto/rand` 32 bytes), plan bước (WF-REF-04), `started_at = NOW()`, `ends_at = started_at + duration`, insert session.
7. Update player row (mana/stones) với `version = version + 1`.
8. Commit.

### 5.3 Sau commit
- `ClearPendingPlayerQueues`, `PublishPlayerSnapshot(player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)`.
- Publish event `refine.session` (create).
- Audit `refine.session.create`.
- Trả `respondActionWithInventory` + `session`.

---

## 6. Luồng submit bước

Endpoint: `POST /api/refine/sessions/:id/steps`

### 6.1 Transaction ngắn (không lock player)
1. `SELECT … FROM refine_sessions WHERE id=$1 AND player_id=$2 FOR UPDATE`.
2. `status = 'ACTIVE'`.
3. Xác định bước `i = step_index`; lấy `plan.steps[i-1]`.
4. Validate:
   - `i` chưa có trong `step_results`.
   - `NOW() >= window_start`.
   - `NOW() <= window_end + 1s`.
   - `nonce` khớp.
   - `action` khớp `type`.
   - Payload đúng schema.
5. Chấm điểm server-side (WF-REF-04).
6. Ghi `step_results[i]`; tính `accuracy` tạm (các bước chưa có = 0).
7. `UPDATE refine_sessions SET step_results=$1, accuracy=$2`.
8. Commit.

Nếu là bước cuối (`i == N`) hoặc tất cả bước đã có kết quả → gọi `resolveRefineSession` ở transaction thứ hai (cùng request).

### 6.2 Idempotency
- Nếu `nonce` đã dùng cho đúng bước → trả kết quả đã lưu (200), không chấm lại.
- Nếu bước đã có result nhưng nonce khác → `409 REFINE_STEP_ALREADY_DONE`.
- Nếu bước đã đóng → `410 REFINE_STEP_CLOSED`.

### 6.3 Response
- `{session, last_step}`; nếu resolve vừa xảy ra, thêm `result` + `player` (inventory/stones).
- Publish `refine.session` sau commit.
- Không publish player snapshot cho mỗi bước (tránh spam); chỉ publish khi resolve/abort.

---

## 7. Snapshot tỉ lệ & tích hợp accuracy

### 7.1 Refactor helper
Trích logic tính tỉ lệ từ `RefineGu` (`refine_gu.go:215-347`) thành:

```go
type RefineRateSnapshot struct {
    Breakdown       RefineRateBreakdown `json:"breakdown"`
    RawPositiveRate float64             `json:"raw_positive_rate"`
    StatusPenalty   float64             `json:"status_penalty"`
    RefinementRealm int                 `json:"refinement_realm"`
    InsightStacks   int                 `json:"insight_stacks"`
    ImmortalPenalty float64             `json:"immortal_penalty"`
}

func computeRefineRate(tx *sql.Tx, player *models.Player, recipe models.RefiningRecipe) (RefineRateSnapshot, error)
```

- Dùng chung cho đường nhanh, tạo phiên, preview.
- Không đọc/ghi gì ngoài dữ liệu đã có trong `player` + `tx` (path, region, statuses, dao marks).
- `breakdown.final_rate` tại thời điểm tạo = `min(1, raw_positive) - status_penalty` (chưa accuracy/death).

### 7.2 Resolve
```go
accuracy := averageStepScores(stepResults, plan.N)              // 0..100
bonus := snap.AccuracyMaxBonus * clamp((accuracy-50)/50, 0, 1)  // WF-REF-06 snapshot
capped := math.Min(1.0, snap.RawPositiveRate + bonus)
final := math.Max(0.0, capped - snap.StatusPenalty - deathPenalty)
roll := rand.Float64()
outcome := determineRefineOutcome(roll, final, snap.RefinementRealm, rank)
```

- `deathPenalty` đọc từ cột `death_penalty` (0 hoặc giá trị snapshot `rate_snapshot.death_penalty`); hook chết set từ snapshot thay vì hardcode.
- `accuracyBonus = 0` nếu `isUniqueImmortalFail` tại resolve.
- Mastery khi success: `gain × (1 + (snap.MasteryMultiplierMax−1) × accuracy/100)`; baseline 2.0 cho kết quả trùng WF-REF-03.
- `breakdown` trả về gồm toàn bộ trường cũ + `accuracy`, `accuracy_bonus`, `accuracy_max_bonus`, `death_penalty`, `step_scores`.

### 7.3 Khi nào snapshot
- Tạo phiên: snapshot cố định. Người chơi tu luyện/đổi build trong lúc chờ **không** làm thay đổi tỉ lệ phiên.
- Pity/status sau resolve dùng trạng thái **hiện tại** (xem §8.3), không dùng snapshot.

---

## 8. Resolve

Hàm chung: `resolveRefineSession(ctx, sessionID, reason)` — gọi từ submit bước cuối, worker, lazy GET.

### 8.1 Transaction
1. `SET LOCAL statement_timeout='15000'; SET LOCAL lock_timeout='5000'`.
2. `SELECT … FROM refine_sessions WHERE id=$1 FOR UPDATE`.
3. Nếu `status != 'ACTIVE'` → trả kết quả hiện có.
4. `evaluateManaTx(tx, playerID)` — lock player; load inventory.
5. Nếu `target_rank >= 6`: `db.CheckImmortalGuExists(tx, targetGuID, targetRank, selectedIDs)`.
   - Nếu đã tồn tại → `isUniqueImmortalFail = true`, `accuracyBonus = 0`, ép outcome `FAIL_NORMAL`.
6. Tính `accuracy`, `bonus`, `final`, roll, outcome.
7. Áp dụng outcome (xem §8.2).
8. Pity + fatigue/backfire (xem §8.3).
9. Recompute `equipped_gus` từ inventory.
10. Persist Gu (`RemovePlayerGus` / `UpdatePlayerGu`), player row, session row (`status='RESOLVED'`, `result`, `resolved_at`, `accuracy`, `result`).
11. `TrackDailyMissionProgress` nếu success.
12. Commit.

### 8.2 Outcome & Cổ
- **SUCCESS**:
  - Xóa toàn bộ `selected_gu_ids` (`db.RemovePlayerGus`).
  - Tạo `models.GuInstance` mới: `uuid.New()`, `Hunger=100`, `Status="ACTIVE"`, `Equipped=false`, `IsVital = any(gu_snapshot.IsVital)`.
  - **Savepoint** trước khi insert; nếu `db.IsImmortalConflict(err)` → `ROLLBACK TO SAVEPOINT`, chuyển outcome `FAIL_NORMAL` với message "Tiên cổ đã xuất thế", không phản phệ.
  - Ghi `gu_history` `CREATED_REFINE` + thông báo thế giới nếu rank ≥ 6.
- **FAIL_SAFE**:
  - Xóa toàn bộ Cổ thường; Cổ Bản Mệnh (theo snapshot) → `Hunger=1`, `Status="DYING"`, `Equipped=false`, `is_locked=false`.
  - Nguyên liệu đã trừ từ đầu, không trừ thêm.
- **FAIL_NORMAL**:
  - Xóa Cổ thường; Cổ Bản Mệnh → `DYING`.
  - Không phản phệ.
- **FAIL_BACKFIRE**:
  - Như FAIL_NORMAL + HP −50% max (sàn 10), Soul −30% max (sàn 1).
  - Thêm status `refine_backfire`.

### 8.3 Status sau resolve
Giữ nguyên hành vi hiện tại (`refine_gu.go:553-593`):
- `isTiringRefine = targetRank >= 6 || targetRank >= player.Rank`.
- Nếu đang `refine_fatigue` → chuyển/thêm `refine_backfire` 20 phút.
- Nếu đang `refine_backfire` → gia hạn 20 phút.
- Nếu backfire outcome → thêm `refine_backfire` 20 phút.
- Ngược lại → thêm `refine_fatigue` 10 phút.
- Pity: success + hợp lệ → reset `insight_<rank>`; fail + hợp lệ (không unique fail) → +1 stack.
- Mastery: success → `calculateRefinementPathGain(rank, realm) × (1 + accuracy/100)`.

### 8.4 Message
- Thêm `(Tỉ lệ: X%)` như hiện tại.
- Nếu `reason = expired`: thêm "Phiên luyện kết thúc khi đạo hữu vắng mặt (điểm chính xác: Y%)."
- Nếu `death_penalty > 0`: thêm "Tâm thần chấn động: tỉ lệ giảm 25%."

### 8.5 Sau commit
- `ClearPendingPlayerQueues` + `PublishPlayerSnapshot(...)`.
- Publish `refine.session` (resolved).
- `notify.Send` "Luyện Cổ Hoàn Tất".
- Audit `refine.session.resolve`.

---

## 9. API contract

### 9.1 Public session view

```json
{
  "id": "uuid",
  "status": "ACTIVE",
  "recipe_id": "recipe_...",
  "target_gu_id": "GU_...",
  "target_rank": 3,
  "variant": 1,
  "started_at": "...",
  "ends_at": "...",
  "accuracy": 62.5,
  "death_penalty": 0.25,
  "steps": [
    {"index": 1, "type": "CHOICE", "state": "DONE", "score": 100},
    {"index": 2, "type": "SEQUENCE", "state": "OPEN", "score": null},
    {"index": 3, "type": "TAP", "state": "LOCKED", "score": null}
  ],
  "step": {
    "index": 2,
    "type": "SEQUENCE",
    "nonce": "uuid",
    "window_start": "...",
    "window_end": "...",
    "params": {"symbols": [0,3,1,2], "display_ms": 450, "target_input_ms": 4800}
  },
  "next_step_at": null,
  "result": null
}
```

- `state ∈ {DONE, OPEN, LOCKED, MISSED}`.
- `step.params` chỉ có params công khai (WF-REF-04 §4).
- Không trả `plan`, `seed`, `gu_snapshot`, `rate_snapshot`.

### 9.2 Endpoints

| Method | Route | Body | Response |
|---|---|---|---|
| POST | `/api/refine/sessions` | `{recipe_id, selected_gu_instance_ids}` | `{...player action fields, session}` |
| GET | `/api/refine/sessions/active` | — | `{session}` hoặc `{session:null}` |
| GET | `/api/refine/sessions/:id` | — | `{session}` |
| POST | `/api/refine/sessions/:id/steps` | `{step_index, nonce, action, payload, client_ts}` | `{session, last_step, result?, player?}` |
| POST | `/api/refine/sessions/:id/abort` | — | `{session, ...player action fields}` |
| POST | `/api/refine/sessions/:id/ack` | — | `{ok:true}` |
| GET | `/api/refine/preview` | `?recipe_id=` | `{rate_breakdown, rate_min, rate_max, accuracy_max_bonus, death_penalty, warnings}` |

**`GET /active`**:
- Nếu có ACTIVE và `ends_at <= NOW()` → resolve trước, trả RESOLVED.
- Nếu không có ACTIVE, trả RESOLVED gần nhất chưa `acknowledged_at` trong 7 ngày (để UI hiện kết quả), ngược lại `{session:null}`.

**`GET /preview`**:
- Load player không lock (`playerstate.Load` hoặc `PlayerSvc.Load`).
- Dùng `computeRefineRate`.
- `rate_min` = accuracy 0; `rate_max` = accuracy 100; cả hai sau cap + status, trước death.
- `warnings`: thiếu mana/đá/nguyên liệu/Cổ, recipe chưa 100%, đang có phiên active.

**Route group**: thêm vào nhóm `refining` hiện có (`server.go:690-698`) để dùng chung auth/onboard/rate limit 20/phút.

### 9.3 Error codes

Trả `{error: "...", code: "..."}`:

| Code | HTTP | Khi nào |
|---|---|---|
| `REFINE_SESSION_ACTIVE` | 409 | đã có phiên ACTIVE |
| `REFINE_SESSION_NOT_FOUND` | 404 | không tồn tại / không thuộc player |
| `REFINE_SESSION_REQUIRED` | 400 | đường nhanh nhận cổ ngang cấp/Tiên Cổ |
| `REFINE_STEP_NOT_OPEN` | 409 | gửi trước `window_start` |
| `REFINE_STEP_CLOSED` | 410 | sau `window_end + 1s` |
| `REFINE_STEP_ALREADY_DONE` | 409 | bước đã có kết quả, nonce khác |
| `REFINE_NONCE_INVALID` | 409 | nonce sai |
| `REFINE_PAYLOAD_INVALID` | 400 | action/payload sai |
| `REFINE_GU_LOCKED` | 409 | Cổ đã bị khóa bởi phiên khác |
| `REFINE_NOT_ENOUGH_MANA` / `..._STONES` / `..._MATERIALS` | 400 | thiếu tài nguyên |
| `REFINE_IMMORTAL_CONFLICT` | 409 | Tiên Cổ vừa được người khác luyện xong |

---

## 10. Realtime event

### 10.1 Kênh mới
- Redis: `refine:<playerID>`.
- Client channel: `"refine"`.
- Payload:

```json
{
  "type": "refine.session",
  "revision": 12,
  "session": { "...public session view..." : null }
}
```

### 10.2 Plumbing
- `backend/internal/ws/redis_sub.go`:
  - `go subscribePattern(shub, "refine:*")`.
  - `formatClientEvent`: prefix `refine:` → `"refine"`.
  - `isPlayerScopedChannel`: thêm `"refine:"`.
- `backend/internal/ws/client.go`: auto-subscribe `refine:<playerID>`.
- `backend/internal/handlers/refine_session.go`: `publishRefineSession(playerID, session)` dùng `cache.PublishEventQuick("refine:"+id, payload)`.
- Frontend: `wsManager.subscribe("refine", handler)` trong `useGameStore.subscribeEvents` hoặc slice `refineSlice` mới.

### 10.3 Khi nào publish
- Tạo phiên.
- Mỗi bước submit (session view nhẹ, không kèm inventory).
- Resolve.
- Abort.

Ngoài ra, create/resolve/abort gọi `PublishPlayerSnapshot` để cập nhật inventory/mana/stones.

---

## 11. Locking & transaction rules

| Thao tác | Lock session | Lock player | Timeout |
|---|---|---|---|
| Create | — (insert) | `FOR UPDATE` | mặc định (2s lock / 8s stmt) |
| Submit step | `FOR UPDATE` | **Không** | mặc định |
| Resolve / abort | `FOR UPDATE` | `FOR UPDATE` | 5s lock / 15s stmt |
| Worker sweep | đọc ID ngoài tx | chỉ trong resolve | như trên |

- Không sleep/giữ transaction qua thời gian chờ.
- Không dùng goroutine giữ `*sql.Tx` qua `time.After`.
- Worker xử lý từng phiên một transaction ngắn; lỗi một phiên không chặn các phiên khác.
- `resolveRefineSession` phải chịu được gọi đồng thời từ worker + submit + lazy GET → guard `status='ACTIVE'` dưới row lock.

---

## 12. Worker & lazy resolve

### 12.1 Worker
File mới `backend/internal/worker/refine_session.go`:

```go
func StartRefineSessionWorker(ctx context.Context) {
    worker.RunEvery(ctx, 30*time.Second, "refine-session-worker", func(ctx context.Context) {
        resolveDueRefineSessions(ctx, 50)
    })
}
```

- `resolveDueRefineSessions` query:

```sql
SELECT id FROM refine_sessions
WHERE status = 'ACTIVE' AND ends_at <= NOW()
ORDER BY ends_at ASC
LIMIT $1;
```

- Với mỗi ID gọi `handlers.ResolveRefineSessionByID(ctx, id)` (exported wrapper).
- Đăng ký trong `cmd/server/main.go` cạnh các worker hiện có, truyền `ctx` shutdown.

### 12.2 Lazy resolve
- `GET /sessions/active` và `GET /sessions/:id` kiểm tra `ends_at <= NOW()`; nếu đúng gọi resolve trước khi trả.
- Lazy resolve là đường chính khi worker chết hoặc Redis/DB chập chờn.

---

## 13. Hook chết trong phiên

Trong `backend/internal/handlers/death_penalty.go` → `persistDeathDrop` (choke point của mọi death drop), thêm:

```go
if _, err := tx.Exec(`
    UPDATE refine_sessions
    SET death_penalty = COALESCE((rate_snapshot->>'death_penalty')::float8, 0.25),
        death_marked_at = NOW()
    WHERE player_id = $1 AND status = 'ACTIVE' AND death_penalty = 0
`, loserID); err != nil {
    logger.S.Warnf("failed to mark refine session death for %s: %v", loserID, err)
}
```

- Chỉ đánh dấu một lần, không stack.
- Giá trị lấy từ snapshot cấu hình rank (WF-REF-06), không hardcode, để Admin chỉnh được.
- Vì nằm trong savepoint của death drop, nếu death drop rollback thì cờ cũng rollback — đúng.
- Party members chết cùng được xử lý qua `applyDeathDropRecord` → `persistDeathDrop`.
- Ghi chú: nếu sau này có đường chết khác không đi qua `persistDeathDrop`, phải gọi cùng helper.

---

## 14. Đường nhanh & tương thích

- Giữ `POST /api/refine` (`RefineGu`).
- Thêm guard đầu handler:

```go
if recipe.TargetRank >= 6 || recipe.TargetRank >= player.Rank {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "Cổ này cần luyện qua phiên có thời gian.",
        "code":  "REFINE_SESSION_REQUIRED",
    })
    return
}
```

- `RefineGu` từ chối Cổ có `is_locked` (query `player_gu`).
- `RefineGu` dùng `computeRefineRate` để tránh trùng lặp logic; accuracy = 0, death = 0.
- Hành vi roll/outcome/status/pity giữ nguyên.

---

## 15. Danh sách file thay đổi

**Mới**
- `backend/internal/db/schema_refine.go`
- `backend/internal/handlers/refine_session.go` — create/get/submit/abort/ack
- `backend/internal/handlers/refine_session_plan.go` — seed, plan bước, public view
- `backend/internal/handlers/refine_session_score.go` — chấm 4 minigame
- `backend/internal/handlers/refine_session_resolve.go` — resolve/abort dùng chung
- `backend/internal/worker/refine_session.go`
- `backend/internal/handlers/refine_session_test.go`
- `backend/internal/handlers/refine_session_plan_test.go`
- `backend/internal/handlers/refine_minigame_test.go`

**Sửa**
- `backend/internal/db/db.go` — đăng ký `schemaRefine`
- `backend/internal/server/server.go` — route mới trong nhóm `/refine`
- `backend/internal/handlers/refine_core.go` — `RefineRateSnapshot`, `computeRefineRate`, mở rộng `RefineRateBreakdown`
- `backend/internal/handlers/refine_gu.go` — dùng helper, guard quick path, check locked
- `backend/internal/handlers/death_penalty.go` — hook death
- `backend/internal/ws/redis_sub.go`, `backend/internal/ws/client.go` — channel `refine`
- `backend/cmd/server/main.go` — start worker
- `frontend/src/lib/websocket.ts` (nếu cần type) + `frontend/src/store/useGameStore.ts` — subscribe `"refine"`
- `docs/API.md`, `WIKI.md`, `docs/GAME_DATA.md`, `docs/DEVELOPMENT.md`

---

## 16. Audit, notification, nhiệm vụ

- Audit:
  - `refine.session.create` — recipe, target, Gu ids.
  - `refine.session.step` — step, score, offset.
  - `refine.session.resolve` — outcome, rate, accuracy, death.
  - `refine.session.abort` — lý do, hoàn nguyên liệu.
- `notify.Send` khi resolve (thành công/thất bại), kể cả expired.
- `TrackDailyMissionProgress` chỉ khi success, dùng `"refine:"+newGuInstance.InstanceID`.
- `gu_history` chỉ ghi khi tạo Cổ thành công; không ghi từng bước.

---

## 17. Kiểm thử bắt buộc

### Tạo phiên
- Trừ đúng mana/đá/nguyên liệu; lock đúng Cổ; không lock Cổ khác.
- Unique index chặn phiên thứ hai → 409.
- Cổ đã lock/DEAD/sai số lượng → 400/409.
- Recipe chưa 100% → 400.
- Rollback khi insert session lỗi → tài nguyên hoàn nguyên.

### Bước
- Gửi trước giờ mở → 409; sau giờ đóng + 1s → 410.
- Sai nonce → 409; nonce lặp → trả kết quả cũ.
- Sai action/payload → 400.
- Chấm đúng cả 4 dạng (threshold 0/50/80/100).
- Không lock player: một player khác login/combat trong lúc submit không bị chặn (kiểm tra không có `FOR UPDATE players` trong tx submit).

### Resolve
- Success: Cổ cũ bị xóa, Cổ mới tạo, `IsVital` kế thừa, mastery ×(1+acc/100).
- Fail/FailSafe/Backfire: Cổ thường xóa, Cổ Bản Mệnh DYING, HP/Soul đúng.
- Unique immortal: phát hiện tại resolve → FAIL_NORMAL, không phản phệ, accuracyBonus 0.
- Race insert immortal: savepoint rollback → FAIL_NORMAL, không 500.
- Pity: fail +1, success reset; không đổi khi unique fail.
- Fatigue/backfire: đúng chuỗi escalate hiện tại.
- Death penalty: 0.25 áp dụng một lần, final rate giảm 25 điểm phần trăm.
- Expired: accuracy 0, vẫn roll, message đúng, notification gửi.
- Resolve gọi 2 lần đồng thời → chỉ một lần áp dụng hiệu ứng.

### Abort
- Hoàn `floor(qty/2)` nguyên liệu.
- Không hoàn mana/đá.
- Unlock Cổ + `Hunger=1`, `Status='DYING'`, `Equipped=false`.
- Recompute `equipped_gus`.
- Session `ABORTED`; gọi abort lần hai trả trạng thái cũ.

### Worker & lazy
- Worker resolve phiên quá hạn, không đụng phiên chưa hạn.
- Lazy GET resolve ngay khi hết hạn.
- Redis down không làm worker panic (dùng `RunEvery` + log).

### Quick path
- Cổ rank thấp vẫn refine tức thời, kết quả không đổi.
- Cổ ngang cấp/Tiên Cổ → `REFINE_SESSION_REQUIRED`.
- Cổ bị lock → 409.
- So sánh kết quả quick path trước/sau refactor helper (cùng seed roll) — không lệch công thức.

### Realtime
- Create/step/resolve/abort publish đúng channel `refine:<id>`, payload không lộ seed/đáp án.
- Client reconnect → `GET /active` đồng bộ.

---

## 18. Việc chuyển ticket khác

- Bảng nhịp độ/độ khó + CSV/Admin → WF-REF-06.
- UI components, HUD, modal, đồng bộ đồng hồ → WF-REF-07.
- Prototype Canh Nhịp → WF-REF-08.
- Kinh tế đường nhanh, deprecate API cũ → WF-REF-09.
