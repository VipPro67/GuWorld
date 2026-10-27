# [SPECIFICATION] Cấu Hình Nhịp Độ Luyện Cổ Theo Rank (Data-driven)

**Ticket:** [WF-REF-06: Cấu hình nhịp độ theo rank, dữ liệu CSV & Admin import](../tickets/WF-REF-06-cau-hinh-nhip-do-data-driven.md)
**Map:** [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Ngày hoàn thành:** 2026-10-27

---

## 1. Quyết định

- Dùng **bảng riêng** `refine_stage_templates` (khóa chính `rank` 1–9), không nhồi JSON vào `refining_recipes`.
- Nguồn chỉnh sửa: **CSV `RefineStage.csv`** + Admin panel "Dữ liệu".
- `InitSchema` seed 9 dòng baseline bằng Go (`ON CONFLICT DO NOTHING`) để game chạy được ngay; Admin import ghi đè.
- Handler đọc bảng **mỗi lần tạo phiên**, không cache in-memory → import có hiệu lực ngay, không cần restart.
- Phiên đang chạy giữ snapshot cấu hình tại lúc tạo; import giữa chừng chỉ ảnh hưởng phiên mới.
- Không làm per-recipe override ở v1 (để fog).

---

## 2. Schema

Thêm vào `backend/internal/db/schema_refine.go` (cùng file với `refine_sessions`):

```sql
CREATE TABLE IF NOT EXISTS refine_stage_templates (
    rank                   INTEGER PRIMARY KEY CHECK (rank BETWEEN 1 AND 9),
    duration_seconds       INTEGER NOT NULL CHECK (duration_seconds > 0),
    step_count             INTEGER NOT NULL CHECK (step_count BETWEEN 3 AND 12),
    minigame_count         INTEGER NOT NULL CHECK (minigame_count >= 1),
    window_seconds         INTEGER NOT NULL CHECK (window_seconds > 0),

    accuracy_max_bonus     DOUBLE PRECISION NOT NULL DEFAULT 0
                           CHECK (accuracy_max_bonus >= 0 AND accuracy_max_bonus <= 1),
    death_penalty          DOUBLE PRECISION NOT NULL DEFAULT 0.25
                           CHECK (death_penalty >= 0 AND death_penalty <= 1),
    mastery_multiplier_max DOUBLE PRECISION NOT NULL DEFAULT 2.0
                           CHECK (mastery_multiplier_max >= 1),

    tap_half_width         DOUBLE PRECISION NOT NULL CHECK (tap_half_width > 0 AND tap_half_width <= 0.5),
    tap_period_ms          INTEGER NOT NULL CHECK (tap_period_ms >= 300),
    sequence_length        INTEGER NOT NULL CHECK (sequence_length BETWEEN 2 AND 10),
    sequence_display_ms    INTEGER NOT NULL CHECK (sequence_display_ms BETWEEN 100 AND 2000),
    sequence_input_ms      INTEGER NOT NULL CHECK (sequence_input_ms BETWEEN 200 AND 5000),
    order_item_count       INTEGER NOT NULL CHECK (order_item_count BETWEEN 2 AND 10),
    choice_option_count    INTEGER NOT NULL CHECK (choice_option_count BETWEEN 2 AND 3),

    minigame_weights       JSONB NOT NULL DEFAULT '{"TAP":1,"SEQUENCE":1,"ORDER":1}',

    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT refine_stage_minigame_le_steps CHECK (minigame_count <= step_count - 1)
);
```

**Ràng buộc nghiệp vụ** (preflight, không chỉ CHECK):
- `minigame_count <= step_count - 1` vì bước 1 luôn là `CHOICE`.
- `window_seconds <= duration_seconds / step_count` để cửa sổ không chồng lên bước sau.
- `minigame_weights` phải là object JSON với các khóa `TAP`, `SEQUENCE`, `ORDER`, giá trị số ≥ 0, tổng > 0.

---

## 3. Giá trị baseline

| Rank | Duration | N | K | Window | AccMax | Death | MasteryMax | TapW | TapP | SeqL | SeqDisp | SeqInput | OrderN | ChoiceN |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 60 | 3 | 1 | 20 | 0.08 | 0.25 | 2.0 | 0.12 | 2000 | 3 | 600 | 1500 | 3 | 2 |
| 2 | 120 | 3 | 1 | 15 | 0.08 | 0.25 | 2.0 | 0.12 | 2000 | 3 | 600 | 1500 | 3 | 2 |
| 3 | 300 | 4 | 2 | 12 | 0.07 | 0.25 | 2.0 | 0.10 | 1700 | 4 | 500 | 1300 | 4 | 3 |
| 4 | 600 | 4 | 2 | 10 | 0.07 | 0.25 | 2.0 | 0.10 | 1700 | 4 | 500 | 1300 | 4 | 3 |
| 5 | 1200 | 5 | 3 | 8 | 0.06 | 0.25 | 2.0 | 0.08 | 1400 | 5 | 450 | 1200 | 5 | 3 |
| 6 | 2700 | 5 | 3 | 8 | 0.05 | 0.25 | 2.0 | 0.08 | 1400 | 5 | 450 | 1200 | 5 | 3 |
| 7 | 5400 | 6 | 4 | 6 | 0.05 | 0.25 | 2.0 | 0.065 | 1100 | 6 | 380 | 1000 | 6 | 3 |
| 8 | 7200 | 6 | 4 | 6 | 0.04 | 0.25 | 2.0 | 0.065 | 1100 | 6 | 380 | 1000 | 6 | 3 |
| 9 | 14400 | 7 | 4 | 5 | 0.04 | 0.25 | 2.0 | 0.05 | 900 | 6 | 320 | 900 | 6 | 3 |

Ghi chú:
- `N` = `step_count`, `K` = `minigame_count`.
- `TapW` = `tap_half_width`; `TapP` = `tap_period_ms`; `SeqL` = `sequence_length`; `SeqDisp` = `sequence_display_ms`; `SeqInput` = `sequence_input_ms`; `OrderN` = `order_item_count`; `ChoiceN` = `choice_option_count`.
- `minigame_weights` baseline: `{"TAP":1,"SEQUENCE":1,"ORDER":1}` cho mọi rank.

---

## 4. CSV

### 4.1 File & header

`backend/data/RefineStage.csv`:

```csv
rank,duration_seconds,step_count,minigame_count,window_seconds,accuracy_max_bonus,death_penalty,mastery_multiplier_max,tap_half_width,tap_period_ms,sequence_length,sequence_display_ms,sequence_input_ms,order_item_count,choice_option_count,minigame_weights
```

9 dòng dữ liệu, đúng rank 1–9. Trường `minigame_weights` là JSON nên phải quote và escape dấu `"`:

```csv
1,60,3,1,20,0.08,0.25,2.0,0.12,2000,3,600,1500,3,2,"{""TAP"":1,""SEQUENCE"":1,""ORDER"":1}"
```

### 4.2 Quy tắc import
- Header phải có đủ 16 cột; thiếu cột khóa/không parse được → lỗi.
- Đúng 9 dòng, không trùng rank, đủ 1–9.
- Mọi giá trị số phải parse được; JSON `minigame_weights` hợp lệ.
- Import là **upsert theo rank**; không xóa rank ngoài CSV (preflight đã bắt buộc đủ 9).
- Dry-run hiển thị diff insert/update/unchanged như các dataset khác.
- Import không cần restart; phiên mới đọc cấu hình mới.

---

## 5. Đăng ký dataset

### 5.1 `import_workflow.go`

Thêm vào `importDatasets` (sau `recipe`):

```go
{"refine_stage", "RefineStage.csv", "refine_stage_templates", false, false, []string{"rank"}},
```

- `reload = false`, `restart = false` vì handler đọc DB trực tiếp.
- Thêm `"minigame_weights": true` vào map `jsonFields` trong `validateImportCSV`.

### 5.2 `datasets.go`

Thêm vào `Datasets()` (sau `recipe`):

```go
dataset(
    "refine_stage",
    "Nhịp độ luyện cổ",
    "RefineStage.csv",
    func() error { return seedRefineStagesFromCSV("RefineStage.csv") },
    individualExportDefs["refine_stage"],
    preflightRefineStages,
    []string{"none"},
    []string{"none"},
),
```

- `knownDatasetCacheBehaviors` và `knownDatasetRegistryReloads` đã có `"none"`, không cần sửa.
- Admin UI tự hiện dataset mới từ `DatasetInfos()`; không cần sửa frontend.

### 5.3 Export

Thêm vào `individualExportDefs`:

```go
"refine_stage": {
    filename: "RefineStage.csv",
    columns: []string{
        "rank", "duration_seconds", "step_count", "minigame_count", "window_seconds",
        "accuracy_max_bonus", "death_penalty", "mastery_multiplier_max",
        "tap_half_width", "tap_period_ms", "sequence_length", "sequence_display_ms",
        "sequence_input_ms", "order_item_count", "choice_option_count", "minigame_weights",
    },
    query: `SELECT rank, duration_seconds, step_count, minigame_count, window_seconds,
                   accuracy_max_bonus, death_penalty, mastery_multiplier_max,
                   tap_half_width, tap_period_ms, sequence_length, sequence_display_ms,
                   sequence_input_ms, order_item_count, choice_option_count,
                   COALESCE(minigame_weights::text,'{}')
            FROM refine_stage_templates ORDER BY rank`,
},
```

---

## 6. Parse / seed / preflight

Tạo `backend/internal/db/refine_stage.go`:

```go
type refineStageRow struct {
    Rank                int
    DurationSeconds     int
    StepCount           int
    MinigameCount       int
    WindowSeconds       int
    AccuracyMaxBonus    float64
    DeathPenalty        float64
    MasteryMultiplierMax float64
    TapHalfWidth        float64
    TapPeriodMs         int
    SequenceLength      int
    SequenceDisplayMs   int
    SequenceInputMs     int
    OrderItemCount      int
    ChoiceOptionCount   int
    MinigameWeights     map[string]float64
}

func parseRefineStageCSV(filename string) ([]refineStageRow, error)
func validateRefineStageRows(rows []refineStageRow) error
func preflightRefineStages() error
func seedRefineStagesFromCSV(filename string) error
```

### 6.1 Parse
- `openCSV(filename)`; `colIndex`; hỗ trợ cả trường hợp cột `minigame_weights` rỗng → mặc định `{"TAP":1,"SEQUENCE":1,"ORDER":1}`.
- Parse số bằng `strconv` (không dùng `fmt.Sscanf` nuốt lỗi).
- Trả lỗi kèm số dòng khi parse thất bại.

### 6.2 Validate
- Đúng 9 dòng, rank 1..9 đủ, không trùng.
- `duration > 0`, `step_count 3..12`, `1 <= minigame_count <= step_count-1`, `window > 0`.
- `window <= duration / step_count` (so sánh float).
- `accuracy_max_bonus 0..1`, `death_penalty 0..1`, `mastery_multiplier_max >= 1`.
- `tap_half_width 0..0.5`, `tap_period_ms >= 300`.
- `sequence_length 2..10`, `sequence_display_ms 100..2000`, `sequence_input_ms 200..5000`.
- `order_item_count 2..10`, `choice_option_count 2..3`.
- `minigame_weights` chỉ chứa `TAP`/`SEQUENCE`/`ORDER`, giá trị ≥ 0, tổng > 0.

### 6.3 Seed
- Upsert:

```sql
INSERT INTO refine_stage_templates (
    rank, duration_seconds, step_count, minigame_count, window_seconds,
    accuracy_max_bonus, death_penalty, mastery_multiplier_max,
    tap_half_width, tap_period_ms, sequence_length, sequence_display_ms,
    sequence_input_ms, order_item_count, choice_option_count, minigame_weights,
    updated_at
) VALUES (...)
ON CONFLICT (rank) DO UPDATE SET
    duration_seconds = EXCLUDED.duration_seconds,
    ...
    minigame_weights = EXCLUDED.minigame_weights,
    updated_at = NOW();
```

- Dùng `activeCSVImportDB()` để chạy trong transaction import.
- Trả lỗi nếu preflight fail; không ghi một phần.

### 6.4 Baseline seed trong `schemaRefine()`
- Sau DDL, chèn 9 dòng baseline bằng Go với `ON CONFLICT (rank) DO NOTHING`.
- Không đọc CSV lúc startup (đúng quy ước repo).
- Admin import ghi đè; restart không reset vì `DO NOTHING`.

---

## 7. Đọc cấu hình lúc runtime

### 7.1 Model
Thêm `backend/internal/models/refine_stage.go`:

```go
type RefineStageTemplate struct {
    Rank                 int                `json:"rank"`
    DurationSeconds      int                `json:"duration_seconds"`
    StepCount            int                `json:"step_count"`
    MinigameCount        int                `json:"minigame_count"`
    WindowSeconds        int                `json:"window_seconds"`
    AccuracyMaxBonus     float64            `json:"accuracy_max_bonus"`
    DeathPenalty         float64            `json:"death_penalty"`
    MasteryMultiplierMax float64            `json:"mastery_multiplier_max"`
    TapHalfWidth         float64            `json:"tap_half_width"`
    TapPeriodMs          int                `json:"tap_period_ms"`
    SequenceLength       int                `json:"sequence_length"`
    SequenceDisplayMs    int                `json:"sequence_display_ms"`
    SequenceInputMs      int                `json:"sequence_input_ms"`
    OrderItemCount       int                `json:"order_item_count"`
    ChoiceOptionCount    int                `json:"choice_option_count"`
    MinigameWeights      map[string]float64 `json:"minigame_weights"`
}

func LoadRefineStageTemplate(q db.Queryer, rank int) (*RefineStageTemplate, error)
```

- `LoadRefineStageTemplate` query 1 dòng theo rank; `sql.ErrNoRows` → `ErrRefineStageMissing`.
- Không cache; mỗi lần tạo phiên 1 query nhỏ.

### 7.2 Snapshot vào phiên
Khi tạo phiên (`refine_session.go`):

- `stage := models.LoadRefineStageTemplate(tx, recipe.TargetRank)`.
- Dùng `stage` để sinh `plan` (số bước, loại bước, trọng số, độ khó, `gap_ms`, `window_ms`).
- Ghi vào `rate_snapshot`:

```json
{
  "breakdown": { ... },
  "raw_positive_rate": 0.0,
  "status_penalty": 0.0,
  "refinement_realm": 0,
  "insight_stacks": 0,
  "immortal_penalty": 1.0,
  "accuracy_max_bonus": 0.07,
  "death_penalty": 0.25,
  "mastery_multiplier_max": 2.0
}
```

- Resolve dùng các giá trị snapshot này, **không** đọc lại bảng.
- Hook chết dùng `rate_snapshot->>'death_penalty'` thay vì hardcode 0.25.

### 7.3 Công thức resolve cập nhật
```go
bonus := snap.AccuracyMaxBonus * clamp((accuracy-50)/50, 0, 1)
masteryMult := 1 + (snap.MasteryMultiplierMax-1)*accuracy/100
deathPenalty := session.DeathPenalty // 0 hoặc snap.DeathPenalty
```

- Với baseline, kết quả trùng WF-REF-03/04.
- `gamefx.RefineAccuracyBonus` giữ làm fallback/test helper; handler dùng snapshot.

---

## 8. Trường hợp lỗi

| Tình huống | Hành vi |
|---|---|
| Bảng rỗng / thiếu rank | `POST /sessions` trả `503 REFINE_CONFIG_MISSING` kèm hướng dẫn import |
| CSV thiếu rank / trùng rank | Dry-run fail, không commit |
| `window > gap` | Dry-run fail |
| `minigame_count >= step_count` | Dry-run fail |
| JSON weights sai | Dry-run fail |
| Admin import giữa phiên | Phiên cũ không đổi; phiên mới dùng cấu hình mới |
| Admin xóa hết bảng | Phiên mới bị chặn; phiên cũ vẫn resolve bằng snapshot |

---

## 9. File thay đổi

**Mới**
- `backend/data/RefineStage.csv`
- `backend/internal/db/refine_stage.go`
- `backend/internal/models/refine_stage.go`

**Sửa**
- `backend/internal/db/schema_refine.go` — bảng `refine_stage_templates` + baseline seed
- `backend/internal/db/db.go` — `jsonFields["minigame_weights"]`, export def
- `backend/internal/db/import_workflow.go` — dataset entry
- `backend/internal/db/datasets.go` — dataset + preflight
- `backend/internal/handlers/refine_session.go` — load stage, snapshot fields
- `backend/internal/handlers/refine_session_plan.go` — dùng stage thay hardcode
- `backend/internal/handlers/refine_session_resolve.go` — dùng snapshot bonus/mastery/death
- `backend/internal/handlers/death_penalty.go` — đọc `rate_snapshot->>'death_penalty'`
- `docs/GAME_DATA.md` — bảng + CSV + quy trình import
- `docs/API.md` — lỗi `REFINE_CONFIG_MISSING` (nếu có trong bảng lỗi)

---

## 10. Kiểm thử

- Parse CSV baseline 9 dòng thành công; seed idempotent; restart không reset Admin edit.
- Thiếu rank / trùng rank / rank ngoài 1–9 → lỗi.
- `window > duration/N` → lỗi.
- `minigame_count >= step_count` → lỗi.
- `minigame_weights` sai JSON/khóa/tổng 0 → lỗi.
- `choice_option_count` 2 và 3 đều dùng được.
- Export ra CSV rồi import lại → không diff.
- Tạo phiên đọc đúng `gap_ms`, `window_ms`, số bước, độ khó.
- Snapshot `accuracy_max_bonus`, `death_penalty`, `mastery_multiplier_max` đúng.
- Import sửa rank 3 giữa lúc phiên rank 3 đang chạy → phiên cũ giữ snapshot, phiên mới dùng số mới.
- Bảng rỗng → `503 REFINE_CONFIG_MISSING`.
- Resolve dùng `death_penalty` từ snapshot, không hardcode.

---

## 11. Tài liệu

Thêm vào `docs/GAME_DATA.md`:

- Bảng `refine_stage_templates` và ý nghĩa từng cột.
- File `RefineStage.csv`, header, ví dụ 1 dòng.
- Quy trình: Admin → "Dữ liệu" → "Nhịp độ luyện cổ" → Dry-run → Import.
- Lưu ý: import không cần restart; phiên đang chạy giữ snapshot.
- Baseline mặc định và cách khôi phục (import lại CSV baseline).

---

## 12. Ngoài phạm vi

- Override theo từng recipe hoặc từng lưu phái.
- Admin UI chỉnh trực tiếp không qua CSV.
- Trọng số cho `HOLD`/`REACTION` (v1.1) — JSON đã sẵn chỗ.
- Versioning cấu hình / rollback nhiều phiên bản.
