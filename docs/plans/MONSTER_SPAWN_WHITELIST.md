# Plan: Whitelist quái theo `MonsterSpawn`

> **Trạng thái:** PLAN — chưa implement.
> **Phạm vi:** backend (Go) + `backend/data/MonsterSpawn.csv` + docs.
> **Mục tiêu:** quái sinh trên bản đồ / combat du ngoạn phải lấy từ `monster_spawns`, không bao giờ chọn ngẫu nhiên từ registry toàn cục.

---

## 1. Nguyên tắc sau thay đổi

> **Pool quái của một vùng = dòng `monster_spawns` active của chính nó;
> nếu vùng không có dòng nào → lấy của vùng cha (`ParentID`);
> không có ở bất kỳ cấp nào → không sinh quái thường.
> Không bao giờ chọn quái từ `models.MonsterRegistry` toàn cục.**

### Quy tắc walk-up (quan trọng)

Tìm **cấp đầu tiên** theo thứ tự `self → cha → ông → …` mà **có ≥ 1 dòng active bất kỳ**
(mặc cả dòng boss), rồi lấy **subset non-boss** của cấp đó.

Quy tắc "dừng ở cấp có dòng, kể cả dòng boss" là điểm mấu chốt — nó cân được cả ba quyết định ở §4:

| Tình huống | Cấp có dòng | Subset non-boss | Kết quả |
|---|---|---|---|
| Vùng có pool đầy đủ (41 vùng) | chính nó | có | spawn đúng pool của nó, không pha cha |
| Vùng con không có dòng (9 vùng active) | cha (Đại Vực) | có | kế thừa quái Tây Mạc / Đông Hải |
| Vùng chỉ có dòng boss (4 vùng) | **chính nó** | **rỗng** | **không sinh quái thường** |
| Vùng không có dòng + cha cũng không | không nơi nào | — | không sinh quái thường |

Đã xác minh trên data hiện tại:

- 9 vùng con không có dòng → kế thừa `dai_vuc_tay_mac` / `dai_vuc_dong_hai` → pool **không rỗng**.
- 4 vùng boss-only (`bach_quang_thien_cung`, `khi_hai_vuc`, `nghich_luu_ha`, `quang_am_chi_ha`)
  **đều có dòng**, nên dừng ở mình → subset rỗng → **đúng mong đợi**, dù cha có pool.
- Sau quy tắc: **30 vùng** pool ambient rỗng = 26 Elite-only (sửa data, §6) + 4 boss-only (cố ý).

---

## 2. Bối cảnh: cơ chế seed hiện tại (đã điều tra)

`GET /api/world/monsters` → `GetWorldMonsters`
(`backend/internal/handlers/world_monsters.go:650`) gộp **4 lớp**:

| # | Nguồn | Điều kiện | Đóng góp thực tế |
|---|---|---|---|
| 1 | `monster_spawns` có tọa độ | `x/y IS NOT NULL AND is_boss_spawn=FALSE` (`world_monsters.go:677-680`) | **0 con** — 73 dòng có tọa độ đều `Fixed` + `is_boss_spawn=True` |
| 2 | `map_cells` loại `Boss` | 23 dòng active trong `MapCell.csv` | boss cố định |
| 3 | Redis AOI bucket | actor đã materialize, quái đã đuổi ra xa | quái đang hoạt động |
| 4 | **Procedural** | `proceduralMonsterSpawns` (`world_monsters.go:560-581`) → `ResolveCell` → `generateCellForZone` | **hầu hết quái thường** |

`spawn_mode` **không** điều khiển vị trí: runtime không branch theo nó.
Chỉ `Fixed` có tọa độ (import enforce ở `import_workflow.go:185-200`); `Roaming`/`Patrol`/`Random`
chỉ nạp vào `zoneRegistry.spawnPools` (`zone_registry.go:190-219`) để procedural chọn **loài**.

Vị trí quái procedural là **deterministic có seed**, không phải random mỗi lần vào:

- `GetCellSeed(x, y, zone)` = sha256 (`exploration_cells.go:33-39`).
- Lưới bãi: `monsterCampSpacing=48`, tâm bãi `cx = gx*48 + 8 + rng.Intn(32)` (`:264-336`).
- Mật độ theo khoảng cách tâm: `0.8 / 0.55 / 0.3` (`monsterCampSpawnChance:338-347`).
- 10% Elite ở tâm bãi (`monsterCampEliteChance=0.10`).
- Ngoài bãi: quái đi lẻ theo `pool_weights.CellWeights`.
- Runtime actor sống trong Redis (`world:monster:*`), lazy tạo khi có player snapshot đầu tiên,
  TTL = `respawn_seconds + 5 phút` (`monsterTTL:366`).

---

## 3. Lỗ hổng cần sửa

| ID | Lỗ hổng | Vị trí | Vùng bị ảnh hưởng |
|---|---|---|---|
| **G1** | `pickMonsterFromPool` lọc template `Elite` khỏi pool ambient → candidates rỗng → rơi vào `randomNormalMonsterID()` chọn **Normal bất kỳ toàn server** | `exploration_cells.go:161`, `:295` | **26 vùng** chỉ có dòng non-boss là Elite |
| **G2** | `zoneReg.spawnPools[regionID]` tra **khóa chính xác**, không đi lên `ParentID` (khác `persistentCellForRegion`) | `zone_registry.go:760` | **9 vùng active** không có dòng riêng → random toàn cục dù cha có pool |
| **G3** | `PickRandomMonsterFromRegion` chỉ nhận `spawn_mode IN ('Roaming','Random')`, **bỏ Patrol** → fallback random Normal toàn cục | `db/monster_db.go:93` | **13 vùng** chỉ có Patrol non-boss, gồm cả `dai_vuc_bach_thien`, `dai_vuc_hac_thien`. Dùng ở combat du ngoạn `combat_explore.go:137` |
| **G4** | `pickGuardTemplate` query `ORDER BY RANDOM()` mỗi guard, không walk cha, pool rỗng → trả `""` → **bỏ luôn guard** | `world_monsters.go:426-437` | Elite/Boss mọi vùng (1 query DB mỗi guard) |

### 26 vùng Elite-only (G1)

Non-boss rows duy nhất là template `Elite` → pool ambient rỗng.

```
am_luu_cu_hac, azure_ghost_desert, bat_bai_coc, blood_lake_purple_sand,
dai_vuc_bach_thien, dai_vuc_hac_thien, dang_hon_son, dao_kiep_am_vuc,
hu_tien_phuc_dia, huyet_hai_co_vuc, lang_gia_phuc_dia, long_ngu_dao,
luu_sa_hai, nguyen_lien_hoa_gioi, shadow_glimpse_trench, sinh_tu_mon,
thai_co_van_cung, thao_thiet_hai_khau, thien_dinh_di_tich, three_kings_mountain,
tu_vuc_u_minh, van_sam_hai_vuc, van_thu_hoang_nguyen, vo_bien_thu_lam,
vuong_dinh_phuc_dia, xuong_hon_sa_mac
```

(18 `KHU_VUC`, 6 `SECRET_REALM`, 2 `NGU_VUC`.)

### 9 vùng con không có dòng (G2)

```
Tây Mạc:  lieu_hoa_sa_hai, bach_cot_sa_nguyen, thien_sa_co_dia,
          van_doc_sa_trieu, hoang_co_di_tich, kim_sa_linh_mach
Đông Hải: bach_sa_quan, long_mon_hai_vuc, van_luu_tinh_hai
```

Cả 9 đều `is_active=True` và khai `cell_type_weights = {Empty:60, Monster:30, Resource:10}`
→ vẫn sinh quái, nhưng loài random từ pool Normal toàn server.

### 13 vùng chỉ có Patrol (G3)

```
dai_vuc_bach_thien, dai_vuc_hac_thien, dao_kiep_am_vuc, long_ngu_dao,
luu_sa_hai, nguyen_lien_hoa_gioi, shadow_glimpse_trench, thai_co_van_cung,
thien_dinh_di_tich, van_sam_hai_vuc, van_thu_hoang_nguyen,
vuong_dinh_phuc_dia, xuong_hon_sa_mac
```

---

## 4. Quyết định đã chốt

| # | Câu hỏi | Quyết định |
|---|---|---|
| D1 | 26 vùng Elite-only xử lý sao? | **Sửa data, không sửa code** — thêm dòng Normal vào `MonsterSpawn.csv`. Code giữ nguyên filter Elite (Elite = leader bãi), pool Elite-only → trả fail → ô `Empty`. |
| D2 | 9 vùng con có kế thừa pool `ParentID`? | **Có.** Thêm walk-up, bắt chước `persistentCellForRegion`. |
| D3 | 4 vùng chỉ có dòng boss → quái thường thế nào? | **Không sinh quái thường.** Chỉ boss đã khai báo. |
| D4 | Bảng Normal cho 26 vùng? | **Dùng bảng đề xuất** ở §6. |
| D5 | Dead code `FindFixedSpawn` + `PickDeterministicMonsterFromRegion`? | **Xóa cả 2** (không có caller). |

---

## 5. Thay đổi code (5 file)

### 5.1 `backend/internal/handlers/zone_registry.go`

**Thêm helper walk-up** — pattern giống `persistentCellForRegion` (`zone_registry.go:681-697`),
có guard `ParentID == current` và depth limit:

```go
// spawnPoolForRegion returns the spawn rows that govern regionID: the rows of
// the first ancestor (self included) that configures any spawn at all. A region
// that only configures bosses therefore resolves to its own (empty) non-boss
// subset instead of silently inheriting its parent's ambient pool.
func spawnPoolForRegion(regionID string) []models.MonsterSpawn {
	const maxDepth = 8
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	current := regionID
	for depth := 0; depth < maxDepth && current != ""; depth++ {
		if pool, ok := zoneReg.spawnPools[current]; ok && len(pool) > 0 {
			return pool
		}
		zone, ok := zoneReg.byID[current]
		if !ok || zone.ParentID == "" || zone.ParentID == current {
			break
		}
		current = zone.ParentID
	}
	return nil
}
```

**Sửa 2 consumer:**

- `pickMonsterFromPool(regionID, rng, wantBoss)` (`:758`) — thay
  `pool := zoneReg.spawnPools[regionID]` bằng `pool := spawnPoolForRegion(regionID)`.
- `pickMonsterFromPoolOfType(regionID, rng, monsterType)` (`:797`) — tương tự
  (hiện đang copy slice từ map, thay bằng helper).

**Giữ nguyên:**
- Filter Elite khỏi pool ambient (`:769-774`) — "Elites are camp leaders".
- Filter `IsBossSpawn != wantBoss`.
- Trọng số `Chance`.

> ⚠️ `spawnPoolForRegion` tự lấy lock RLock → **không** được gọi trong khi đã giữ lock
> (hiện cả 2 caller đều tự lock, an toàn). Tránh `defer` Double-RLock nếu Go version
> không re-entrant — `sync.RWMutex.RLock` lồng nhau an toàn trừ khi có writer chờ
> (có thể deadlock). Để an toàn: tách thành `spawnPoolForRegionLocked()` không lock
> và để caller giữ lock, hoặc helper tự lock và caller không giữ lock nào.
> **Chọn cách: helper tự lock, 2 caller không giữ lock.**

### 5.2 `backend/internal/handlers/exploration_cells.go`

**(a) Ô wanderer — `:155-170`**

```go
case "Monster":
	spawn, ok := pickMonsterFromPool(zoneID, rng, false)
	if !ok {
		// Whitelist: this zone configures no ambient monster. Never fall back
		// to a random global template.
		return models.Cell{X: x, Y: y, Type: "Empty", Content: ""}
	}
	monsterID := spawn.MonsterID
	rank := zone.MinRank
	// ... giữ nguyên phần tính rank/SubRank
```

**(b) Bãi (camp) — `findProceduralMonsterCamp:288-336`**

```go
spawn, ok := pickMonsterFromPool(zone.ID, rng, false)
if !ok {
	// This grid cell configures no ambient monster: no camp here.
	continue
}
monsterID := spawn.MonsterID
minRank, maxRank := spawn.MinRank, spawn.MaxRank
// bỏ hẳn nhánh `if monsterID == "" { monsterID = randomNormalMonsterID(rng) }`
```

> ⚠️ **Bắt buộc `continue`, không được `break`**, và **không set `found = true`**.
> Nếu set `found` với `MonsterID == ""`, ô camp sẽ rơi lại nhánh weighted pick
> và cũng fail → vẫn `Empty`, nhưng vị trí bãi chiếm chỗ (đè lên grid cell khác
> vì `distance >= best.distance` short-circuit). `continue` cho grid cell khác
> tranh thắng hợp lệ.
>
> Phần Elite (`:311-331`) giữ nguyên — `pickMonsterFromPoolOfType` đã trả
> `(zero,false)` khi không có Elite → `hasElite=false`, không fallback.

**(c) Xóa `randomNormalMonsterID`** (`:349-363`) — 2 caller duy nhất đã xử lý ở trên.

**(d) `generateCellForZone:112`** — giữ nguyên `ZoneTypeHub → Empty`.

> **Đ determinism:** thay đổi chỉ ảnh hưởng 30 vùng pool rỗng.
> 41 vùng còn lại đi đúng nhánh `ok == true` như cũ → RNG call sequence không đổi
> → nội dung ô **không đổi**. Test `TestGenerateCellStableAcrossCalls` phải pass nguyên.

### 5.3 `backend/internal/handlers/world_monsters.go`

**Đổi `pickGuardTemplate` (`:426-437`)** — bỏ query DB per-guard, dùng pool in-memory:

```go
// pickGuardTemplate picks a guard from the region's configured spawn pool so
// guards obey the same whitelist as every other ambient monster. Returns ""
// when the zone configures no ambient monster (caller skips that guard).
func pickGuardTemplate(regionID string, spawnX, spawnY int) string {
	rng := rand.New(rand.NewSource(GetCellSeed(spawnX, spawnY, regionID+":guard")))
	spawn, ok := pickMonsterFromPool(regionID, rng, false)
	if !ok {
		return ""
	}
	return spawn.MonsterID
}
```

Gọi tại `spawnGuards:406`:
`MonsterID: pickGuardTemplate(leader.RegionID, int(leader.SpawnX), int(leader.SpawnY))`

Kết quả: guard lấy từ **cùng pool** với quái thường (bao gồm walk-up cha),
deterministic theo vị trí leader, hết 1 query DB/guard.
Pool rỗng → `""` → guard bị bỏ (behavior hiện có ở `:414-416`, giữ nguyên).

### 5.4 `backend/internal/db/monster_db.go`

**(a) `PickRandomMonsterFromRegion` (`:86-124`)** — sửa G3 + walk-up + bỏ fallback:

```go
// PickRandomMonsterFromRegion picks a monster configured for the zone, walking
// up the region ancestry the same way the in-memory spawn pool does. It returns
// sql.ErrNoRows when neither the zone nor any ancestor configures an ambient
// monster — callers must handle that instead of substituting a random template.
func PickRandomMonsterFromRegion(tx *sql.Tx, regionID string) (string, int, int, error) {
	const query = `
		WITH RECURSIVE chain(depth, region_id) AS (
			SELECT 0, $1::text
			UNION ALL
			SELECT c.depth + 1, r.parent_id
			FROM chain c
			JOIN regions r ON r.id = c.region_id
			WHERE r.parent_id IS NOT NULL AND r.parent_id <> '' AND c.depth < 8
		)
		SELECT s.monster_id, s.min_rank, s.max_rank
		FROM monster_spawns s
		JOIN chain c ON c.region_id = s.id
		WHERE s.is_active = TRUE AND s.is_boss_spawn = FALSE
		ORDER BY c.depth, RANDOM()
		LIMIT 1`

	var monsterID string
	var minRank, maxRank int
	err := tx.QueryRow(query, regionID).Scan(&monsterID, &minRank, &maxRank)
	if err == sql.ErrNoRows {
		return "", 0, 0, err
	}
	if err != nil {
		return "", 1, 1, err
	}
	return monsterID, minRank, maxRank, nil
}
```

- **Bỏ** filter `spawn_mode IN ('Roaming','Random')` → nhận mọi dòng non-boss active
  (sửa G3: `Patrol` không còn bị bỏ sót; `Fixed` non-boss cũng được nhận — hợp lý
  vì đây là "quái được khai báo cho vùng").
- **Bỏ** fallback `SELECT id FROM monster_templates WHERE type='Normal' ORDER BY RANDOM()`
  (`:100-104`) → trả `sql.ErrNoRows`.
- `ORDER BY c.depth, RANDOM()` trong Postgres: sort tuple `(depth, rand)` →
  lấy row depth nhỏ nhất, random tie-break. Pool ≤ ~10 row → chi phí không đáng kể.
- CTE bọc guard `c.depth < 8` tránh vòng lặp `ParentID` tự trỏ.

**(b) Xóa 2 hàm dead code (D5):**

- `FindFixedSpawn` — `monster_db.go:60-82`, 0 caller.
- `PickDeterministicMonsterFromRegion` — `monster_db.go:177-248`, 0 caller
  (cùng có bug `spawn_mode IN ('Roaming','Random')` y hệt G3, và import
  `"math/rand"` có thể trở nên unused → kiểm tra lại import sau khi xóa).

### 5.5 `backend/internal/handlers/combat_explore.go`

Hiện `:137-141`:

```go
monsterID, minRank, maxRank, err := db.PickRandomMonsterFromRegion(tx, req.NodeID)
if err != nil || monsterID == "" {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chọn quái vật từ cơ sở dữ liệu"})
    return
}
```

→ HTTP **500** cho 4 vùng boss-only nếu bỏ fallback.

**Sửa:**

1. Extract nhánh tìm Nguyên Thạch (`:130-133`) thành helper:

```go
// rollExploreStones rewards the non-combat explore branch. Shared with the
// "zone configures no monster" case so a whitelist-clean zone still yields a
// useful outcome instead of an error.
func rollExploreStones(player *models.Player) string {
	stonesFound := 1 + rand.Intn(3)
	player.PrimevalStones += float64(stonesFound)
	return fmt.Sprintf("Khám phá hoàn tất tại %s. Đạo hữu may mắn phát hiện thấy %d viên Nguyên Thạch!",
		nodeName, stonesFound)   // ← cần truyền nodeName vào thay vì closure
}
```

> `nodeName` là biến local (`:117-118`) → helper phải nhận `nodeName string` làm tham số.

2. Gọi lại khi zone không khai báo quái:

```go
monsterID, minRank, maxRank, err := db.PickRandomMonsterFromRegion(tx, req.NodeID)
if errors.Is(err, sql.ErrNoRows) {
    // Zone configures no ambient monster: fall through to the resource branch.
    logMsg = rollExploreStones(&player, nodeName)
    combatTriggered = false
} else if err != nil || monsterID == "" {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chọn quái vật từ cơ sở dữ liệu"})
    return
} else {
    // ... existing combat setup giữ nguyên
}
```

Cần thêm import `"errors"` và `"database/sql"` nếu chưa có
(`combat_explore.go` đã import `"database/sql"` ở `:78` — dùng `sql.NullString`).

> Kiểm tra cấu trúc `if/else if` hiện có (`:130` `roll < 40`, `:134`
> `roll < monsterThreshold`) — việc thêm nhánh `ErrNoRows` **không** được đẩy
> `combatTriggered` thành `true`.

---

## 6. Data: `backend/data/MonsterSpawn.csv`

Thêm **52 dòng** (26 vùng × 2 con Normal).

**Quy ước dòng mới:**

| Cột | Giá trị |
|---|---|
| `region_id` | theo bảng |
| `x`, `y` | **để trống** (mode theo zone) |
| `spawn_mode` | `Roaming` cho con thứ nhất, `Patrol` cho con thứ hai |
| `chance` | `0.55` (Roaming) / `0.35` (Patrol) — khớp CSV hiện có |
| `respawn_seconds` | `60` (Roaming) / `120` (Patrol) |
| `min_rank`, `max_rank` | theo rank vùng (bảng dưới) |
| `is_boss_spawn` | `False` |
| `is_active` | `True` |

**Bảng region → Normal monster** (D4):

| # | Vùng | zone_type | Rank | Normal 1 (Roaming) | Normal 2 (Patrol) |
|---|---|---|---|---|---|
| 1 | `am_luu_cu_hac` | KHU_VUC | 4-6 | `monster_ghost_bat` | `monster_withered_bone_wraith` |
| 2 | `azure_ghost_desert` | KHU_VUC | 4-6 | `monster_desert_ghost_wraith` | `monster_wan_xa_poison_snake` |
| 3 | `bat_bai_coc` | KHU_VUC | 4-7 | `monster_fire_refine_crab` | `monster_creation_sprout_spirit` |
| 4 | `blood_lake_purple_sand` | KHU_VUC | 2-5 | `monster_baby_blood_spider` | `monster_fire_refine_crab` |
| 5 | `dai_vuc_bach_thien` \* | NGU_VUC | 6-9 | `monster_heaven_sparrow` | `monster_meteor_firefly` |
| 6 | `dai_vuc_hac_thien` \* | NGU_VUC | 8-9 | `monster_ghost_bat` | `monster_withered_bone_wraith` |
| 7 | `dang_hon_son` | SECRET_REALM | 5-9 | `monster_withered_bone_wraith` | `monster_decay_bone_beetle` |
| 8 | `dao_kiep_am_vuc` | KHU_VUC | 4-7 | `monster_thief_monkey` | `monster_flying_fox` |
| 9 | `hu_tien_phuc_dia` | KHU_VUC | 3-6 | `monster_herb_rabbit` | `monster_flying_fox` |
| 10 | `huyet_hai_co_vuc` | KHU_VUC | 6-8 | `monster_ghost_bat` | `monster_slaughter_pup` |
| 11 | `lang_gia_phuc_dia` | KHU_VUC | 7-9 | `monster_stone_ape` | `monster_thief_monkey` |
| 12 | `long_ngu_dao` | KHU_VUC | 3-7 | `monster_koi_fish` | `monster_sea_beast` |
| 13 | `luu_sa_hai` | KHU_VUC | 3-6 | `monster_desert_ghost_wraith` | `monster_thief_monkey` |
| 14 | `nguyen_lien_hoa_gioi` | SECRET_REALM | 6-9 | `monster_dan_qing_spirit_butterfly` | `monster_flower_butterfly` |
| 15 | `shadow_glimpse_trench` | KHU_VUC | 2-4 | `monster_shadow_spider_cub` | `monster_wind_bee` |
| 16 | `sinh_tu_mon` | SECRET_REALM | 7-9 | `monster_desert_ghost_wraith` | `monster_wan_xa_poison_snake` |
| 17 | `thai_co_van_cung` | SECRET_REALM | 7-9 | `monster_heaven_sparrow` | `monster_sound_spirit_beast` |
| 18 | `thao_thiet_hai_khau` | KHU_VUC | 4-8 | `monster_linh_thao_ke` | `monster_herb_rabbit` |
| 19 | `thien_dinh_di_tich` | SECRET_REALM | 7-9 | `monster_stone_man` | `monster_stone_ape` |
| 20 | `three_kings_mountain` | KHU_VUC | 3-5 | `monster_stone_ape` | `monster_stone_man` |
| 21 | `tu_vuc_u_minh` | SECRET_REALM | 6-9 | `monster_desert_ghost_wraith` | `monster_ghost_bat` |
| 22 | `van_sam_hai_vuc` | KHU_VUC | 7-9 | `monster_koi_fish` | `monster_sea_beast` |
| 23 | `van_thu_hoang_nguyen` | KHU_VUC | 6-8 | `monster_slaughter_pup` | `monster_destruction_ant` |
| 24 | `vo_bien_thu_lam` | KHU_VUC | 3-5 | `monster_clay_boar` | `monster_stone_ape` |
| 25 | `vuong_dinh_phuc_dia` | KHU_VUC | 3-6 | `monster_flying_fox` | `monster_herb_rabbit` |
| 26 | `xuong_hon_sa_mac` | KHU_VUC | 7-9 | `monster_withered_bone_wraith` | `monster_fire_refine_crab` |

\* 2 tầng trên (`dai_vuc_bach_thien`, `dai_vuc_hac_thien`) hiện `is_active=False` ở `Region.csv`
→ sửa cho nhất quán, không ảnh hưởng gameplay tức thì.

**Ràng buộc data đã kiểm:**
- Tất cả 52 `monster_id` đều tồn tại trong `Monster.csv` và `type = 'Normal'`, `is_active = True`.
  (Danh sách 46 con Normal toàn game đã đối chiếu — mọi id trong bảng đều nằm trong đó.)
- Không có cặp `(region_id, monster_id)` trùng lặp (274 dòng hiện tại đều unique).
- **Sau khi sửa CSV:** Admin → tab "Dữ liệu" → Import bảng **"Spawn quái"** → **restart backend**.

---

## 7. Test

File: `backend/internal/handlers/exploration_test.go`
(pattern stub `zoneReg` + `models.MonsterRegistry` đã có sẵn ở `:36-66`, `:163-190`).

| # | Test | Assert |
|---|---|---|
| T1 | Pool chỉ có Elite | Quét ô trong 1 vùng → **không có** ô `Monster` nào; tuyệt đối không xuất hiện id Normal ngoài pool |
| T2 | Pool chỉ có dòng boss | Tương tự → mọi ô `Empty` |
| T3 | Vùng con không có dòng, cha có pool | Ô `Monster` chứa **đúng** loài của cha |
| T4 | Vùng con **có** dòng riêng | Ô chỉ chứa loài của chính nó, **không** pha loài của cha |
| T5 | Camp determinism | Giữ nguyên `TestProceduralMonsterCampsAreStableAndDenseNearCenter`; thêm case pool Elite-only → `findProceduralMonsterCamp` trả `ok=false` |
| T6 | `pickGuardTemplate` | Pool có dòng → trả id trong pool; pool rỗng → trả `""`; cùng seed → cùng kết quả |
| T7 | `spawnPoolForRegion` | Trả pool self khi self có dòng (kể cả boss-only); trả pool cha khi self rỗng; trả `nil` khi cả chuỗi rỗng; chống vòng lặp `ParentID` tự trỏ |
| T8 | `TestGenerateCellStableAcrossCalls` | Giữ nguyên, phải pass (không đổi nội dung 41 vùng có pool) |

Test `db` (T9, nếu có hạ tầng test DB): `PickRandomMonsterFromRegion` trả `ErrNoRows`
cho vùng boss-only, trả đúng loài `Patrol` cho vùng chỉ có Patrol, walk-up đúng depth.

---

## 8. Documentation

| File | Vị trí | Nội dung |
|---|---|---|
| `docs/GAME_DATA.md` | ~`:1028` (mục `monster_spawns`) | Whitelist rule, kế thừa `ParentID`, bỏ fallback registry toàn cục, danh sách 4 vùng không sinh quái thường |
| `WIKI.md` | mục bản đồ / khám phá | 1 câu player-visible: quái mỗi vùng lấy từ data Admin khai báo |
| `docs/ADMIN_RESOURCE_MATRIX.md` | `:117` | Không đổi (đã ghi "Reload Zone/Monster registry") |

---

## 9. Kiểm chứng

```bash
# backend
cd backend && go test ./...
cd backend && go test -race ./...      # có thay đổi đọc pool dưới lock

# frontend (không đổi code frontend, chủ yếu xác nhận không vỡ)
cd frontend && npm run lint
cd frontend && npm run build
```

Checklist trước khi kết thúc:

- [ ] Không file nào ngoài phạm vi bị sửa (đặc biệt các file sect đang WIP trong working tree).
- [ ] `randomNormalMonsterID` không còn reference nào.
- [ ] `FindFixedSpawn`, `PickDeterministicMonsterFromRegion` không còn reference nào.
- [ ] `spawn_mode` vẫn là data-only (runtime không branch theo nó, trừ SQL ở §5.4a).
- [ ] Test T1–T8 pass.
- [ ] Docs đã cập nhật.

---

## 10. Rollout

1. Implement code (§5) → `go test ./...` xanh.
2. Sửa `MonsterSpawn.csv` (§6) → verify bằng script đối chiếu (id tồn tại, không trùng key).
3. Commit tách 2 commit: `fix(spawn): whitelist ...` + `data(monster): add ambient normals ...`.
4. **Deploy:**
   - Admin → tab "Dữ liệu" → Import bảng **"Spawn quái"** (`monster_spawn`).
     `reloadImportedData()` (`admin_data.go:114-135`) đã reload **Zone registry** +
     **Monster registry** và clear `cellGenCache` (`zone_registry.go:233`).
   - **Restart backend** để nạp ZoneRegistry (cache registry in-memory).
5. Kiểm tra: `GET /api/dictionary/monster-spawns` có52 dòng mới;
   vào 1 trong 26 vùng → quái procedural là Normal đã khai báo.

---

## 11. Rủi ro & lưu ý

| Rủi ro | Mức | Xử lý |
|---|---|---|
| Đổi nội dung ô procedural → vị trí quái trên map dịch chuyển | Cao (player-visible) | Chỉ 30 vùng pool rỗng đổi. 41 vùng còn lại **không đổi** vì nhánh `ok==true` giữ nguyên RNG sequence. Vùng đổi thì hành vi cố ý (rời đi Normal ngoài whitelist) |
| Actor Redis cũ vẫn mang loài cũ sau import | Trung bình | TTL = `respawn + 5 phút` → tự hết, không cần flush. Nếu muốn tức thì: xóa keys `world:monsters:aoi:*`, `world:monster:spawn:*` khi import |
| `sync.RWMutex` RLock lồng nhau → deadlock nếu writer chờ | Trung bình | Như §5.1: helper tự lock, caller không giữ lock |
| 6 khu Tây Mạc sau kế thừa chỉ có `monster_fire_mouse` | Thấp (gameplay nghèo) | Ngoài scope — bổ sung Normal riêng cho từng khu Tây Mạc ở lần sau |
| `ExploreNode` ở zone không có quái → trước đây 500 | Đã sửa | §5.5 → rơi về nhánh Nguyên Thạch |
| Quái mỏ / quái kiếp / boss `MapCell.csv` vẫn ngoài whitelist | Cố ý | Xem §12 |

---

## 12. Ngoài phạm vi (không làm lần này)

- **Boss `MapCell.csv` lệch region với `MonsterSpawn.csv`** — vd `monster_boar_king` có ô ở
  `wan_xa_goc` / `gu_luan_slope` nhưng pool `dai_vuc_nam_cuong` không chứa nó;
  5 con `*_calamity_boss` / `*_tribulation_boss` chỉ có trong `MapCell.csv`.
  → tách issue riêng.
- **4 dòng Fixed boss của 4 Đại Vực nằm trong bounding box hub** (vd `dai_vuc_nam_cuong`
  5080,5082 với hub 4500–5500) → có thể bị `isWorldMonsterSafeZone` reject lúc runtime.
- **Làm giàu pool Tây Mạc** (6 khu con chỉ thừa hưởng `monster_fire_mouse`).
- **Quái mỏ** (`mine.go:628-654`, random từ `monster_templates`) — hệ thống riêng.
- **Quái sinh bởi hệ kiếp** (`monster_tribulation_beast` + 5 boss kiếp) — worker riêng,
  chủ đích không có dòng `MonsterSpawn` (xem `GAME_DATA.md:962`).
- **Sủng thú / NPC dị nhân** (`non_player_actors`) — không phải quái map.
- **Flush Redis khi import** để đổi loài tức thì thay vì chờ TTL.
