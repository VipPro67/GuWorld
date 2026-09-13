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

Đạo Ngân chỉ lưu trong **1 nguồn duy nhất**: cột JSONB `players.dao_marks` (`backend/internal/db/db.go:156`).

```json
{
  "strength": 100,
  "fire": 2500
}
```

> Bảng quan hệ `dao_marks` (relational mirror) đã bị **xóa** để thống nhất nguồn dữ liệu. Không còn `dao_marks` table.

**Danh sách path hợp lệ** (49 path, lấy từ cột `path` trong `backend/cmd/server/data/Gu.csv` — key = lowercase bỏ hậu tố `" path"`):

Blade, Blood, Bone, Cloud, Concealment, Creation, Dark, Death, Destruction, Dream, Earth, Enchantment, Enslavement, Fire, Food, Formation, Heaven, Human, Ice, Illusion, Information, Investigation, Life, Light, Lightning, Luck, Metal, Moon, Movement, Poison, Qi, Refinement, Rule, Shadow, Smoke, Soul, Sound, Space, Strength, Sun, Sword, Theft, Theme, Time, Transformation, Water, Wind, Wisdom, Wood.

> `Sun`, `Creation`, `Destruction`, `Death` được bổ sung qua các Cổ `GU_SUN_EMPEROR`, `GU_CREATION_ORIGIN`, `GU_DESTRUCTION_NOVA`, `GU_DEATH_REAPER` để đủ 8 cặp đối nghịch trong `isOpposingPath`.

### 1.3 Công Thức Tính DaoBonus

Áp dụng trong sát chiêu (`killer_move.go:494-501`) và hiệu ứng Cổ (`combat_effects.go:109-118`):

```
daoMultiplier = 1.0 + amount / 100      // mỗi 1 điểm = +1%

finalDamage = baseDamage × daoMultiplier
finalDefense = baseDefense × daoMultiplier
```

**Auto-combat (chiến đấu tự động):** Đạo Ngân được engine `simulator/engine.go` áp dụng **cho cả 2 phe** (người chơi lẫn quái) theo `DaoMultiplier()` (`engine.go`), dựa trên `SimAction.Path` — áp dụng cho **sát thương, Khiên Ảo (`Defense`) và hồi phục (`Healing`)** nếu hành động thuộc cùng đường (path). Log chiến đấu hiển thị phần cộng thêm dạng `gây X + Y sát thương do Đạo Ngân`, `nhận X + Y Khiên Ảo do Đạo Ngân`, `hồi phục X + Y HP do Đạo Ngân`. Trong auto-combat, sát chiêu được bỏ phần dao đã snapshot lúc tạo rồi engine tính lại theo Đạo Ngân hiện tại (`combat_auto.go:buildSimActionFromKillerMove`), còn kỹ năng Cổ thì `CalculateGuEffectForCombatant` được gọi với `applyDao=false` để engine tự áp dụng (`combat_auto.go:buildSimActionFromGu`).

Frontend đồng bộ: `ImmortalApertureView.tsx:1850-1855`

### 1.4 Cách Nhận Đạo Ngân

| Nguồn | Handler | Công Thức |
|-------|---------|-----------|
| **Thiên Kiếp** (Tribulation) | `tribulation.go:874-883` | Earth: 250 + diff×50. Heavenly: 750 + diff×150. Grand: 7250 + diff×500. Myriad: 86750 + diff×2000 |
| **Item effect** `dao_mark_gain` | `item_effects.go:118-131` | `player.DaoMarks[pathKey] += int(val)` |
| **Item effect** `soul_boost` | `item_effects.go:109-115` | `player.DaoMarks["soul"] += int(val)` |
| **Phi Thăng** (Ascension) | `cultivate.go:385-386` | Ban đầu: heaven=100, earth=100 (key lowercase theo `normalizePath`) |
| **Sát Chiêu** (V2) | `SAT_CHIEU_MECHANIC.md:254-255` | `DaoMarks[CorePath] += BaseDaoMarkValue × dao_mark_multiplier` |

### 1.5 Cập Nhật Đạo Ngân

Mọi ghi thêm Đạo Ngân đi qua `addDaoMarksTx()` (`tribulation.go:586`) — thao tác trực tiếp read-modify-write trên `players.dao_marks` JSONB trong transaction:

1. `SELECT COALESCE(dao_marks, '{}') ... FOR UPDATE`
2. Cộng dồn `amount` vào `map[path]int` (normalize path qua `TrimSuffix(strings.ToLower(...), " path")`)
3. `UPDATE players SET dao_marks = $json`

Các nguồn ghi khác (item effect, phi thăng, sát chiêu, gift code, quest) đều cập nhật trực tiếp `player.DaoMarks` map trong code — tất cả cùng 1 nguồn JSONB, không còn đồng bộ 2 chiều.

### 1.6 API

- `GET /api/dao-marks` → `GetDaoMarks` (`aperture.go:345-379`) — đọc trực tiếp `players.dao_marks` JSONB, trả về danh sách `DaoMark[]` theo `amount DESC` (giữ shape `{id, player_id, path, amount, created_at, updated_at}` tương thích frontend)

### 1.7 Frontend

- `ImmortalApertureView.tsx:71` — state `daoMarks`
- `ImmortalApertureView.tsx:430` — bảng "Bảng Đạo Ngân"
- `ImmortalApertureView.tsx:1114` — "Thống Kê Đạo Ngân Tiên Khiếu"
- `Aperture.tsx:1935` — "Đạo Ngân Tích Lũy"
- `useGameStore.ts:792` — `fetchDaoMarks()`

---

## II. CẢNH GIỚI LƯU PHÁI (Path Masteries / Attainment / Chân Ý)

### 2.1 Định Nghĩa

Cảnh giới lưu phái là cấp độ lĩnh ngộ Chân Ý (True Meaning) của một đường. Mỗi đường có 5 cấp. **`players.path_masteries` lưu dạng JSONB `map[string]int` (path → điểm tích lũy), giống cột `dao_marks` của `immortal_apertures`.** Cảnh giới được suy ra từ **mốc điểm**:

| Level | Key | Hệ Số (Multiplier) | Mốc Điểm |
|-------|-----|-------------------|----------|
| 0 | `"Ordinary"` — Phàm Thai | ×1.0 | < 100 |
| 1 | `"Master"` / "Đại Sư" | ×1.1 | ≥ 100 |
| 2 | `"Grandmaster"` / "Tông Sư" | ×1.25 | ≥ 500 |
| 3 | `"Great Grandmaster"` / "Đại Tông Sư" | ×1.4 | ≥ 2.500 |
| 4 | `"Supreme Grandmaster"` / "Vô Thượng Đại Tông Sư" | ×1.5 | ≥ 10.000 |

- **Struct:** `PathMasteriesMap map[string]int` (`player.go:56`)
- **Player field:** `PathMasteries PathMasteriesMap` (`player.go`)
- **DB column:** `players.path_masteries JSONB DEFAULT '{}'` (`db.go`)
- **Hàm tính cảnh giới:** `gamefx.PathMasteryLevel(points)` (`gamefx/path_mastery.go`) — cảnh giới là hàm thuần theo mốc điểm, không bị chặn bởi rank.

**Cảnh giới là ĐIỀU KIỆN để thăng tiên rank** (ngược với ràng buộc cũ): để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, người chơi phải có **ít nhất 1 lưu phái** đạt cảnh giới tương ứng:

| Rank đích | Cảnh giới yêu cầu | Điểm tối thiểu |
|-----------|-------------------|----------------|
| Rank 7 | Tông Sư (Grandmaster, level 2) | 500 |
| Rank 8 | Đại Tông Sư (Great Grandmaster, level 3) | 2.500 |
| Rank 9 | Vô Thượng Đại Tông Sư (Supreme Grandmaster, level 4) | 10.000 |

Kiểm tra tại `tribulation.go` (`handleSuccess`) qua `gamefx.PathMasteryLevelRequiredForRank` + `gamefx.HasPathMasteryAtLeast`. Nếu chưa đủ, đột phá bị chặn (chỉ thông báo, không tăng rank, không trừ kiếp).

Ví dụ JSON:
```json
{ "fire": 1250, "water": 340 }
```

> Dữ liệu cũ lưu tên cấp (vd `{"fire": "Grandmaster"}`) được migration tự động đổi sang mốc điểm khi khởi động (`migrate_path_masteries_to_points`).

### 2.2 Cách Nhận Điểm

| Nguồn | Handler | Điểm |
|-------|---------|------|
| **Sát Chiêu thành công** | `killer_move.go` CastKillerMove | +2 điểm cho path hạch tâm |
| **Luyện Cổ thành công** | `refine.go` | +2 điểm cho path Cổ đích; **+10 điểm nếu Cổ Tiên (rank ≥ 6)** |
| **Chân Ý từ phó bản** | `instance.go` | cộng thẳng `amount` điểm vào path |

> **Giới hạn mỗi trận (battle cap):** các nguồn trong trận chiến thường — đánh tự động/Simulator (`simulator/engine.go`), sát chiêu (`killer_move.go`) — bị chặn ở **`PathMasteryBattleGainLimit(rank)` = Rank × 10 điểm/trận** (`gamefx/path_mastery.go`). Counter lưu trong `players.active_combat.path_mastery_gained` (JSONB, reset mỗi trận); với auto-combat, cap được áp dụng ngay trong vòng lặp simulator qua `CombatantAuto.PathMasteryGainLimit`. Khi chạm trần, điểm Chân Ý không cộng thêm và log có ghi `PATH_MASTERY_CAP`. Luyện cổ và thưởng phó bản không tính vào giới hạn này.

### 2.3 Công Thức Ảnh Hưởng

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
successRate = 95 + Attainment×2 + successRateBonus - supportCount×5 - pathConflictPenalty
```

Trong đó `Attainment` = level (0-4).

### 2.3 Cách Thăng Cấp

Cảnh giới không còn nhảy trực tiếp — điểm tích lũy từ nhiều nguồn (sát chiêu, luyện cổ, phó bản) và cảnh giới được suy ra theo mốc điểm + rank.

**Phó Bản (Instance) — `instance.go:540-586`:**
- Rewards chứa `attainment.path` và `attainment.amount`
- Cộng thẳng `amount` điểm vào `player.PathMasteries[path]` (bỏ cơ chế nhảy 1-2 cấp cũ)
- Log: "nhận được X điểm Chân Ý lưu phái ..."

**Dữ liệu seed — `db.go` `SeedInstanceTemplates`:**
- `dungeon_flower_wine`: `{"attainment": {"path": "Star Path", "amount": 10}}`
- Mộng cảnh được đưa vào **CSV-driven** từ `backend/cmd/server/data/DreamRealm.csv` (import/export qua Admin tab "Dữ liệu" → "Mộng cảnh"):
  - `dream_realm_star_constellation`: `{"attainment": {"path": "Dream Path", "amount": 50}}`
  - **49 mộng cảnh** `dream_realm_<path>` — 1 cho mỗi lưu phái (danh sách `dreamRealmPaths` trong `db.go`), thưởng `{"attainment": {"path": "<Path> Path", "amount": 50}}` + tài liệu path + 100 Nguyên Thạch, vào bằng `dream_path_material`, cooldown 24h. Cột `map_data` trống → sinh tự động theo `path`; điền `map_data` để ghi cốt truyện riêng.
- **Bí cảnh** `SECRET_REALM` (`secret_realm_sword_mountain`, `secret_realm_fire_cave`, `secret_realm_water_palace`) — thưởng **Đạo Ngân** qua `rewards.dao_marks` (xử lý tại `instance.go` completion), cooldown 48h.

---

## III. TÍCH HỢP VỚI SÁT CHIÊU V2

### 3.1 Đạo Ngân trong Sát Chiêu

Tại `killer_move.go:494-501`:

```go
daoMarks = player.DaoMarks[corePath]
daoBonus = 1.0 + daoMarks / 100.0

finalDamage *= daoBonus
finalDefense *= daoBonus
```

### 3.2 Cảnh Giới trong Sát Chiêu

Tại `killer_move.go:505-513`:

```go
attainment = attainmentLevel(player, corePath)  // 0-4
successRate = 95 + attainment×2 + successRateBonus - supportCount×5 - pathConflictPenalty
```

### 3.3 Path Conflict

Tại `killer_move.go:449-457`, `calculatePathConflict()`:

- Cùng đường: +0% penalty
- Khác đường: +10% penalty (`killerMovePathConflictPenaltyBase`)
- Đối nghịch: +20% penalty (`killerMovePathOppositionPenalty`)

**Cặp đối nghịch** (`killer_move.go:1000-1019`) — cả 8 cặp đều có path tồn tại trong Gu.csv (49 paths, gồm Sun/Creation/Destruction/Death đã bổ sung qua GU_SUN_EMPEROR, GU_CREATION_ORIGIN, GU_DESTRUCTION_NOVA, GU_DEATH_REAPER):

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
| `backend/internal/models/player.go` | 30-34, 141-204, 362-381 | DaoMarksMap, PathMasteriesMap (map[string]int), Player struct, scanners |
| `backend/internal/gamefx/path_mastery.go` | — | Mốc điểm, rank gating, PathMasteryLevel/Multiplier |
| `backend/internal/models/aperture.go` | 7-52, 88-99 | DaoMark, ImmortalAperture, ResourceNode |
| `backend/internal/db/db.go` | 156-157 | `players.dao_marks` JSONB (schema); bảng `dao_marks` đã xóa |
| `backend/internal/handlers/killer_move.go` | 20-33, 413-554, 936-953 | Constants, V2 formula, attainmentLevel(), calculatePathConflict() |
| `backend/internal/handlers/combat_effects.go` | 12-126 | Path mastery multiplier, DaoBonus, region resonance |
| `backend/internal/handlers/tribulation.go` | 586-630, 405-470, 860-889 | addDaoMarksTx (JSONB read-modify-write), reward calculation |
| `backend/internal/handlers/aperture.go` | 345-379 | GetDaoMarks API (đọc JSONB) |
| `backend/internal/handlers/item_effects.go` | 109-131 | dao_mark_gain, soul_boost effects |
| `backend/internal/handlers/gu_effects.go` | 80-97 | getPathMasteryMultiplier |
| `backend/internal/handlers/instance.go` | 540-586 | Attainment reward từ phó bản (cộng điểm) |
| `backend/internal/handlers/refine.go` | 353-378 | Luyện cổ: attainment bonus + cộng điểm path |
| `backend/internal/handlers/profile.go` | 25-54, 155-254 | getDominantPath, scouting |
| `backend/internal/handlers/energy.go` | 29-50 | Immortal aperture capacity |
| `backend/internal/server/server.go` | 180-200, 288-317 | Route registration |
| `frontend/src/store/useGameStore.ts` | 477-577, 785-860, 3158-3169, 3501-4171 | Types, state, API calls |
| `frontend/src/components/ImmortalApertureView.tsx` | 71, 430, 1114-1188, 1850-1855 | Dao marks UI + formula |
| `frontend/src/components/Aperture.tsx` | 216, 1800, 1826-1829, 1935 | Đạo Ngân display |
| `frontend/src/components/PlayerProfileModal.tsx` | 226, 337-357 | Profile dao marks |
