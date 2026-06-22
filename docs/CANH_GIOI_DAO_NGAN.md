# Cơ Chế Cảnh Giới Lưu Phái & Đạo Ngân

> Dựa trên source code `backend/internal/handlers/` và `backend/internal/models/` — cập nhật cho V2 Killer Move.

---

## I. ĐẠO NGÂN (Dao Marks)

### 1.1 Định Nghĩa

Đạo Ngân là điểm tích lũy theo từng đường (path), đại diện cho sự lĩnh ngộ sâu sắc của tu sĩ đối với một loại pháp tắc. Mỗi điểm Đạo Ngân gia tăng tỉ lệ sát thương/phòng ngự khi sử dụng Cổ/Kỹ năng thuộc đường đó.

**Struct:**
- `backend/internal/models/player.go:168` — `DaoMarks DaoMarksMap` (JSONB `dao_marks`)
- `backend/internal/models/player.go:30-31` — `type DaoMarksMap map[string]int`
- `backend/internal/models/aperture.go:25-32` — `type DaoMark struct { ID, PlayerID, Path, Amount, CreatedAt, UpdatedAt }`

### 1.2 DB Schema

```
dao_marks (
    id UUID PRIMARY KEY,
    player_id UUID → players(id) ON DELETE CASCADE,
    path VARCHAR(50),
    amount INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(player_id, path)
);
```

- `backend/internal/db/db.go:223-236`

`players.dao_marks` JSONB mirror column: `backend/internal/db/db.go:156`

### 1.3 Công Thức Tính DaoBonus

Áp dụng trong sát chiêu (`killer_move.go:494-501`) và hiệu ứng Cổ (`combat_effects.go:109-118`):

```
Nếu amount <= 10000:
    daoMultiplier = 1.0 + amount / 100      // mỗi 1 điểm = +1%
Nếu amount > 10000:
    daoMultiplier = 1.0 + 100 + (amount - 10000) / 500  // mỗi 1 điểm = +0.2% (soft cap)

finalDamage = baseDamage × daoMultiplier
finalDefense = baseDefense × daoMultiplier
```

Frontend đồng bộ: `ImmortalApertureView.tsx:1850-1855`

### 1.4 Cách Nhận Đạo Ngân

| Nguồn | Handler | Công Thức |
|-------|---------|-----------|
| **Thiên Kiếp** (Tribulation) | `tribulation.go:874-883` | Earth: 250 + diff×50. Heavenly: 750 + diff×150. Grand: 7250 + diff×500. Myriad: 86750 + diff×2000 |
| **Item effect** `dao_mark_gain` | `item_effects.go:118-131` | `player.DaoMarks[pathKey] += int(val)` |
| **Item effect** `soul_boost` | `item_effects.go:109-115` | `player.DaoMarks["soul"] += int(val)` |
| **Phi Thăng** (Ascension) | `cultivate.go:302-306` | Ban đầu: Heaven=100, Earth=100 |
| **Sát Chiêu** (V2) | `SAT_CHIEU_MECHANIC.md:254-255` | `DaoMarks[CorePath] += BaseDaoMarkValue × dao_mark_multiplier` |

### 1.5 Đồng Bộ DB ↔ JSONB

Khi thêm Đạo Ngân qua `addDaoMarksTx()` (`tribulation.go:494-541`):
1. UPSERT vào bảng `dao_marks`
2. SELECT toàn bộ `path, amount` → ghi đè vào `players.dao_marks` JSONB

### 1.6 API

- `GET /api/dao-marks` → `GetDaoMarks` (`aperture.go:290-319`) — trả về danh sách `DaoMark[]` theo `amount DESC`

### 1.7 Frontend

- `ImmortalApertureView.tsx:71` — state `daoMarks`
- `ImmortalApertureView.tsx:430` — bảng "Bảng Đạo Ngân"
- `ImmortalApertureView.tsx:1114` — "Thống Kê Đạo Ngân Tiên Khiếu"
- `Aperture.tsx:1935` — "Đạo Ngân Tích Lũy"
- `useGameStore.ts:792` — `fetchDaoMarks()`

---

## II. CẢNH GIỚI LƯU PHÁI (Path Masteries / Attainment / Chân Ý)

### 2.1 Định Nghĩa

Cảnh giới lưu phái là cấp độ lĩnh ngộ Chân Ý (True Meaning) của một đường. Mỗi đường có 5 cấp:

| Level | Key | Hệ Số (Multiplier) |
|-------|-----|-------------------|
| 0 | `"Ordinary"` — Phàm Thai | ×1.0 |
| 1 | `"Master"` / "Đại Sư" | ×1.1 |
| 2 | `"Grandmaster"` / "Tông Sư" | ×1.25 |
| 3 | `"Great Grandmaster"` / "Đại Tông Sư" | ×1.4 |
| 4 | `"Supreme Grandmaster"` / "Vô Thượng Đại Tông Sư" | ×1.5 |

- **Struct:** `PathMasteriesMap map[string]string` (`player.go:33-34`)
- **Player field:** `PathMasteries PathMasteriesMap` (`player.go:169`)
- **DB column:** `players.path_masteries JSONB DEFAULT '{}'` (`db.go:157`)

### 2.2 Công Thức Ảnh Hưởng

#### a) Trong Combat (Gu Effect — `combat_effects.go:12-31`)

```go
masteryMultiplier = getActorPathMasteryMultiplier(actor, path)
// Master=1.1, Grandmaster=1.25, Great Grandmaster=1.4, Supreme Grandmaster=1.5
```

#### b) Power Scale Penalty (`combat_effects.go:79-81`)

Nếu `attainmentLevel < 2` (dưới Tông Sư):
```
powerScale *= 0.70  // giảm 30% uy lực
```

#### c) Region Resonance (`combat_effects.go:83-105`)

Nếu `attainmentLevel >= 3` (Đại Tông Sư+) và đang ở khu vực tương ứng:

| Khu Vực | Đường |
|----------|------|
| Eastern Sea | Water |
| Western Desert | Fire |
| Northern Plains | Wind |
| Southern Border | Earth |
| Central Continent | Qi |

```
powerScale *= 1.50  // tăng 50% uy lực
```

#### d) Trong Sát Chiêu V2 — Tỉ lệ thành công (`killer_move.go:505-513`)

```go
successRate = 95 + Wisdom×0.2 + Attainment×2 + successRateBonus - supportCount×5 - pathConflictPenalty
```

Trong đó `Attainment` = level (0-4).

### 2.3 Cách Thăng Cấp

**Phó Bản (Instance) — `instance.go:540-586`:**
- Rewards chứa `attainment.path` và `attainment.amount`
- Nếu `amount >= 50`: nhảy 2 cấp
- Nếu `amount < 50`: nhảy 1 cấp
- Tối đa: Supreme Grandmaster (level 4)
- Log: "Chúc mừng đạo hữu xuất sắc vượt ải thành công, nhận được Chân Ý thăng tiến..."

**Dữ liệu seed — `db.go:1840-1968`:**
- `dungeon_flower_wine`: `{"attainment": {"path": "Moon Path", "amount": 10}}`
- `dream_realm_star_constellation`: `{"attainment": {"path": "Dream Path", "amount": 50}}`

---

## III. TÍCH HỢP VỚI SÁT CHIÊU V2

### 3.1 Đạo Ngân trong Sát Chiêu

Tại `killer_move.go:494-501`:

```go
daoMarks = player.DaoMarks[corePath]
daoBonus = 1.0 + daoMarks / 100.0      // ≤ 10000
daoBonus = 101.0 + (daoMarks-10000)/500.0  // > 10000

finalDamage *= daoBonus
finalDefense *= daoBonus
```

### 3.2 Cảnh Giới trong Sát Chiêu

Tại `killer_move.go:505-513`:

```go
attainment = attainmentLevel(player, corePath)  // 0-4
successRate = 95 + Wisdom×0.2 + attainment×2 + successRateBonus - supportCount×5 - pathConflictPenalty
```

### 3.3 Path Conflict

Tại `killer_move.go:449-457`, `calculatePathConflict()`:

- Cùng đường: +0% penalty
- Khác đường: +10% penalty (`killerMovePathConflictPenaltyBase`)
- Đối nghịch: +20% penalty (`killerMovePathOppositionPenalty`)

**Cặp đối nghịch** (`killer_move.go`):

| Path A | Path B |
|--------|--------|
| Fire | Water |
| Strength | Wisdom |
| Metal | Wood |
| Light | Dark |
| Lightning | Earth |
| Moon | Sun |
| Creation | Destruction |
| Life | Death |

---

## IV. TÍCH HỢP VỚI THIÊN KIẾP (Tribulation)

### 4.1 Luồng Xử Lý

1. Aperture tích lũy `internal_years_passed`
2. Khi `internal_years_passed >= next_tribulation_year` → trigger kiếp
3. Player chiến đấu với boss + minion
4. Thành công → `addDaoMarksTx()` với path + reward amount
5. Thất bại → aperture nhận sát thương, không có Đạo Ngân

### 4.2 Reward Scale

| Loại Kiếp | Công Thức |
|-----------|-----------|
| Earth Calamity | 250 + difficulty×50 |
| Heavenly Tribulation | 750 + difficulty×150 |
| Grand Tribulation | 7250 + difficulty×500 |
| Myriad Tribulation | 86750 + difficulty×2000 |

- `tribulation.go:874-883`

---

## V. FRONTEND — TÓM TẮT

### Component & Data Flow

```
ImmortalApertureView.tsx
  ├── fetchDaoMarks() (useGameStore.ts:3158-3169)
  ├── calculateDaoMarkMultiplier() (line 1850-1855) — đồng bộ formula Go
  ├── Bảng "Đạo Ngân" (line 430)
  └── Thống Kê Đạo Ngân (line 1114)

Aperture.tsx
  ├── player.dao_marks["strength"] (line 216)
  ├── "Đạo Ngân Tích Lũy" (line 1935)
  └── fetchDaoMarks() (line 1800)

KillerMoveBuilder.tsx
  └── "Dung hợp Đạo Ngân" section (line 283)

```

---

## VI. FILE THAM CHIẾU

| File | Dòng | Nội Dung |
|------|------|----------|
| `backend/internal/models/player.go` | 30-34, 141-204, 362-381 | DaoMarksMap, PathMasteriesMap, Player struct, scanners |
| `backend/internal/models/aperture.go` | 7-52, 88-99 | DaoMark, ImmortalAperture, ResourceNode |
| `backend/internal/db/db.go` | 156-157, 223-261, 534-596, 767, 814-848, 932-947, 1840-2121 | Schema, seeds, migrations |
| `backend/internal/handlers/killer_move.go` | 20-33, 413-554, 936-953 | Constants, V2 formula, attainmentLevel(), calculatePathConflict() |
| `backend/internal/handlers/combat_effects.go` | 12-126 | Path mastery multiplier, DaoBonus, region resonance |
| `backend/internal/handlers/tribulation.go` | 494-541, 860-889 | addDaoMarksTx, reward calculation |
| `backend/internal/handlers/aperture.go` | 290-319 | GetDaoMarks API |
| `backend/internal/handlers/item_effects.go` | 109-131 | dao_mark_gain, soul_boost effects |
| `backend/internal/handlers/gu_effects.go` | 80-97 | getPathMasteryMultiplier |
| `backend/internal/handlers/instance.go` | 540-586 | Attainment reward từ phó bản |
| `backend/internal/handlers/profile.go` | 25-54, 155-254 | getDominantPath, scouting |
| `backend/internal/handlers/energy.go` | 29-50 | Immortal aperture capacity |
| `backend/internal/server/server.go` | 180-200, 288-317 | Route registration |
| `frontend/src/store/useGameStore.ts` | 477-577, 785-860, 3158-3169, 3501-4171 | Types, state, API calls |
| `frontend/src/components/ImmortalApertureView.tsx` | 71, 430, 1114-1188, 1850-1855 | Dao marks UI + formula |
| `frontend/src/components/Aperture.tsx` | 216, 1800, 1826-1829, 1935 | Đạo Ngân display |
| `frontend/src/components/PlayerProfileModal.tsx` | 226, 337-357 | Profile dao marks |
