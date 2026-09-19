# WF-06: Database Schema & API Routes specification for Deduction & Tracking

**Part of:** [Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận](../scouting-divination-concealment-map.md)  
**Type:** `wayfinder:task`  
**Status:** Closed  
**Blocked by:** WF-02, WF-03, WF-04, WF-05 (All Closed)  

---

## Question

Thiết kế cấu trúc DDL Database và danh sách API Routes chi tiết dựa trên toàn bộ các quyết định thiết kế đã chốt từ WF-01 $\rightarrow$ WF-05.

---

## Resolution

### 1. Database Schema DDL (`backend/internal/db/schema_scouting.go`)

Tuân thủ quy ước DB của GuWorld: bảng mới dùng `CREATE TABLE IF NOT EXISTS`, thêm cột mới vào bảng hiện hữu dùng `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.

```sql
-- 1. Bảng lưu trữ Ấn Ký Truy Tung (Tracking Marks)
CREATE TABLE IF NOT EXISTS player_tracking_marks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tracker_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    mark_type VARCHAR(30) NOT NULL, -- 'wisdom', 'blood', 'dao'
    power FLOAT8 NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tracking_marks_tracker ON player_tracking_marks(tracker_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_tracking_marks_target ON player_tracking_marks(target_id);

-- 2. Bảng lưu Tàn Dư Khí Tức tại ô bản đồ (Residual Scent / Cell Traces)
CREATE TABLE IF NOT EXISTS world_residual_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    x INT NOT NULL,
    y INT NOT NULL,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    action_type VARCHAR(30) NOT NULL, -- 'move', 'combat', 'mine', 'tribulation'
    dominant_path VARCHAR(30) NOT NULL DEFAULT '',
    dao_pulse_strength FLOAT8 NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_residual_traces_region_xy ON world_residual_traces(region_id, x, y, expires_at);

-- 3. Bổ sung các cột phục vụ Che Giấu & Phòng Ngự vào bảng `players`
ALTER TABLE players ADD COLUMN IF NOT EXISTS active_concealment JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE players ADD COLUMN IF NOT EXISTS last_sever_clues_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE players ADD COLUMN IF NOT EXISTS last_resonance_scan_at TIMESTAMPTZ DEFAULT NULL;
```

---

### 2. Chi Tiết Danh Sách API Routes (`backend/internal/server/server.go`)

Tất cả các route trinh sát và suy diễn được nhóm tại `/api/scouting/*`:

#### 2.1 `POST /api/scouting/inspect` (Trinh Sát Cục Bộ)
- **Auth**: Yêu cầu Token
- **Request Body**:
  ```json
  {
    "target_id": "uuid-cua-muc-tieu"
  }
  ```
- **Validation**:
  - Kiểm tra `viewer.current_region == target.current_region` và `hypot(viewer.x - target.x, viewer.y - target.y) <= investigationRange`.
  - Ngoại lệ: Cùng Party (`viewer.party_id == target.party_id`) hoặc cùng Sect.
- **Response**: Trả về `PlayerProfileResponse` theo công thức tính lọc $diff = \text{scouting} - \text{disguise}$.

#### 2.2 `POST /api/scouting/divine` (Suy Diễn Thiên Cơ Từ Xa)
- **Auth**: Yêu cầu Token
- **Request Body**:
  ```json
  {
    "target_username": "CoSuA",
    "suspected_region": "hub_nam_cuong",
    "suspected_sect": "CoNguyetSonTrai",
    "suspected_gu": "GU_MOONLIGHT",
    "item_clue_id": "",
    "resource_type": "mana", // "mana" | "immortal_stone"
    "resource_amount": 100
  }
  ```
- **Response**:
  ```json
  {
    "status_type": "full" | "partial" | "fogged" | "backfire",
    "delta_score": 35.5,
    "divination_report": {
      "target_id": "uuid",
      "username": "CoSuA",
      "region": "hub_nam_cuong",
      "coordinates": { "x": 120, "y": 45 }, // null nếu partial/fogged
      "approx_rank": 4,
      "sub_rank": 2,
      "dominant_path": "Moon",
      "detected_gus": ["GU_MOONLIGHT", "GU_SHADOW"],
      "status_summary": "Đang tu luyện tại mỏ linh thạch"
    },
    "backfire_damage": 0,
    "can_apply_tracking_mark": true
  }
  ```

#### 2.3 `POST /api/scouting/apply-mark` (Gieo Ấn Ký Truy Tung)
- **Auth**: Yêu cầu Token
- **Request Body**:
  ```json
  {
    "target_id": "uuid",
    "mark_type": "wisdom" // "wisdom" | "blood"
  }
  ```
- **Response**: `200 OK` kèm `expires_at` và số lượng ấn ký đang hoạt động `active_marks_count`.

#### 2.4 `GET /api/scouting/tracked-targets` (Lấy Danh Sách Mục Tiêu Đang Theo Dõi)
- **Auth**: Yêu cầu Token
- **Response**:
  ```json
  {
    "tracked_targets": [
      {
        "target_id": "uuid",
        "username": "CoSuA",
        "avatar_url": "",
        "mark_type": "wisdom",
        "current_region": "hub_nam_cuong",
        "x": 120,
        "y": 45,
        "expires_at": "2026-09-19T10:30:00Z"
      }
    ]
  }
  ```

#### 2.5 `POST /api/scouting/sever-clues` (Chém Đứt Manh Mối / Đoạn Tuyệt Thiên Cơ)
- **Auth**: Yêu cầu Token
- **Request Body**:
  ```json
  {
    "use_killer_move": false
  }
  ```
- **Response**:
  ```json
  {
    "success": true,
    "purged_marks_count": 2,
    "purged_cell_traces": 1,
    "message": "Đã chém đứt 2 ấn ký đang âm thầm theo dõi bạn và xóa sạch tàn dư khí tức tại vị trí này!"
  }
  ```

#### 2.6 `POST /api/scouting/resonance-scan` (Cảm Ứng Cổ Trùng & Đạo Ngân)
- **Auth**: Yêu cầu Token
- **Request Body**:
  ```json
  {
    "target_path": "wisdom",
    "min_rank": 3,
    "scan_region": "hub_nam_cuong"
  }
  ```
- **Response**:
  ```json
  {
    "pings": [
      { "x": 115, "y": 40, "intensity": "high", "gu_id": "GU_TOAN_DAU", "entity_type": "wild_gu" },
      { "x": 120, "y": 45, "intensity": "medium", "gu_id": "???", "entity_type": "player" }
    ],
    "dao_pulses": [
      { "region": "hub_nam_cuong", "x": 200, "y": 180, "path": "sword", "event": "tribulation", "expires_in_seconds": 120 }
    ]
  }
  ```
