# [WF-REF-06] Cấu hình nhịp độ theo rank, dữ liệu CSV & Admin import

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:task`
**Status:** Closed
**Assignee:** Agent
**Blocked by:** WF-REF-03

---

## Question

Biến bảng nhịp độ đã chốt thành dữ liệu chỉnh được không cần deploy, theo đúng quy ước data-driven của repo.

### 1. Bảng nhịp độ đã chốt

| Rank | Thời lượng | Số bước | Minigame | Cửa sổ |
|---|---|---|---|---|
| 1 | 60s | 3 | 1 | 20s |
| 2 | 2m | 3 | 1 | 15s |
| 3 | 5m | 4 | 2 | 12s |
| 4 | 10m | 4 | 2 | 10s |
| 5 | 20m | 5 | 3 | 8s |
| 6 | 45m | 5 | 3 | 8s |
| 7 | 90m | 6 | 4 | 6s |
| 8 | 2h | 6 | 4 | 6s |
| 9 | 4h | 7 | 4 | 5s |

Cần bổ sung cột:
- `accuracy_max_bonus` theo rank (WF-REF-03).
- `death_penalty` theo rank hoặc dùng chung.
- `mastery_multiplier_max`.
- Trọng số chọn minigame theo rank/dạng.
- Độ khó chi tiết: tốc độ kim, số decoy, độ dài chuỗi, số mảnh ghép.

### 2. Nơi lưu
Chọn một:
- **Bảng riêng `refine_stage_templates`** (rank PK) + seed từ CSV `RefineStage.csv` qua Admin.
- **Cột JSONB `stage_config` trong `refining_recipes`** (per recipe/rank) — linh hoạt hơn nhưng phình dữ liệu 4.483 dòng.
- **Bảng rank-level + override JSONB per recipe** — lai.

Đề xuất: bảng `refine_stage_templates` theo rank, thêm cột `override JSONB` nullable cho recipe đặc biệt. Không hardcode trong handler.

### 3. CSV & Admin
- Tên file, header, khóa chính, kiểu dữ liệu.
- Đăng ký import trong `backend/internal/db/import_workflow.go` và tab Admin "Dữ liệu".
- Có export ngược ra CSV không? (Repo có cơ chế export.)
- Seed lần đầu khi `InitSchema` hay chờ Admin import? (Repo hiện không auto-seed CSV.)
- Validate khi import: rank 1–9 đủ, thời lượng > 0, số bước khớp số minigame, cửa sổ < thời lượng.

### 4. Registry / cache
- Có cần cache in-memory như `models.GuRegistry` không? Nếu có, nhớ restart backend sau khi import.
- Nếu đọc DB mỗi lần tạo phiên thì chi phí không đáng kể; cân nhắc đơn giản hóa.

### 5. Quy tắc chọn bước
- Bước nào là minigame, bước nào là "chọn hành động" cố định?
- Có bước mở lò/khai lò cố định ở đầu/cuối không?
- Cách phân bổ minigame theo rank lấy từ bảng hay random có trọng số?

### 6. Ảnh hưởng lên UI
- UI cần biết trước số bước, loại bước, thời lượng để dựng timeline; API tạo phiên trả `plan` đã lọc (không lộ seed).
- Có cần endpoint metadata riêng (`GET /api/refine/config`) để UI hiển thị trước khi start không?

---

## Resolution

Đã chốt cấu hình tại [WF-REF-06 Spec](../specs/WF-REF-06-stage-config.md). Tóm tắt:

- **Bảng riêng** `refine_stage_templates` khóa `rank` 1–9, chứa duration/step/minigame/window, `accuracy_max_bonus`, `death_penalty`, `mastery_multiplier_max`, độ khó 4 minigame và `minigame_weights` JSONB.
- **Baseline 9 dòng** seed bằng Go trong `schema_refine()` với `ON CONFLICT DO NOTHING`; Admin import ghi đè, restart không reset.
- **CSV `RefineStage.csv`** 16 cột; import qua Admin "Dữ liệu", có dry-run/diff/export; không cần restart vì handler đọc DB mỗi lần tạo phiên.
- **Preflight** bắt buộc đủ 9 rank, `window ≤ duration/N`, `minigame_count ≤ step_count−1`, giới hạn độ khó, JSON weights hợp lệ.
- **Snapshot** `accuracy_max_bonus`/`death_penalty`/`mastery_multiplier_max` vào `rate_snapshot` lúc tạo; phiên đang chạy không đổi khi import.
- **Hook chết** đọc `rate_snapshot->>'death_penalty'` thay vì hardcode 0.25.
- **Bảng rỗng/thiếu rank** → `503 REFINE_CONFIG_MISSING`.
- Không làm override per-recipe/per-path ở v1.

