# Wayfinder Research #461: Khảo sát kiến trúc DaoCombatProfile, Simulator, Region và Hạ Khiếu hiện tại

- **Map**: #460 (`venerable-environmental-dao-marks`)
- **Ticket**: #461
- **Branch**: `research/461-codebase-anchors`
- **Mục tiêu**: Thu thập các điểm neo kỹ thuật trong codebase hiện tại làm cơ sở thiết kế hệ thống Đạo Ngân Môi Trường (Environmental Dao Marks) không xung đột.

---

## 1. Kiến trúc `DaoCombatProfile`, `Simulator` và truyền Context trận đấu

### 1.1. `gamefx.DaoCombatProfile` (`backend/internal/gamefx/dao_combat_profile.go`)

- **`DaoCombatProfile`** (`L444-452`) & **`PathCombatModifiers`** (`L427-441`): Lưu sẵn toàn bộ hệ số công/thủ theo từng lưu phái để tra cứu $O(1)$ trong mô phỏng.
- **`BuildDaoCombatProfile`** (`L456-463`) $\rightarrow$ **`BuildDaoCombatProfileWithReduction`** (`L468-649`):
  - Chuẩn hóa tên lưu phái qua `CleanPath` (`L194-202`), tính `totalMarks`, `dominantPath`, và `graceThreshold = max(1000, 2% * totalMarks)` (`L504-507`).
  - `BelowConflictFloor`: `totalMarks <= DaoConflictTotalFloor` (`10000`, `L245`, `L520`). Dưới ngưỡng này, `InternalConflictMult = 1.0` và `NaturalResistance = 0.0` (`L543-559`).
  - Với mỗi lưu phái (`CanonicalPaths` + các path phụ có mặt trong `cleanDaoMarks`):
    - `DaoAmplification = DaoAmplification(amt) + bonus` (`1.0 + marks/100.0`, `L234-239`, `L532`).
    - `TargetDaoResonance = DaoAmplification(amt)` (`L533`, không cộng `bonus` ngoài).
    - `RealmControl = RealmControlForLevel(lvl)` (`0 / 0.10 / 0.25 / 0.45 / 1.00`, `L254-266`) & `RealmEfficiency = RealmEfficiencyForLevel(lvl)` (`1.00 / 1.05 / 1.12 / 1.25 / 1.45`, `L275-287`).
    - `InternalConflictMult` (`L561-581`): tính từ `rawConflict * MultiPathFactor * (1 - realmCtrl) * (1 - conflictReductionPct/100)`, quy đổi qua `1 / (1 + conflictRatio * 0.8 * 2.0)`, sàn `0.08`. Nếu `hasNonConflictingDao == true`, `conflictReductionPct = 100` (`L475-477`).
    - `NaturalResistance` (`L584-594`): `0.30 * (defConflict / (defConflict + 50000.0))` (tắt nếu `hasNonConflictingDao == true`).
- Các hàm tra cứu & phạt xung đột:
  - `GetOffensiveFactors(path, sensitivity...)` (`L653-680`): trả về `(daoAmp, realmEff, conflictMult)`.
  - `GetDefensiveResistance(attackPath)` (`L684-693`): trả về `NaturalResistance` (`0.0` nếu `HasNonConflictingDao`).
  - `GetTargetDaoResonance(attackPath)` (`L698-707`): trả về `TargetDaoResonance`.
  - `ManaTaxMultiplier(conflictMult)` (`L318-323`): `1.0 + (1.0 - conflictMult) * 1.5` khi `conflictMult < 1.0`.
  - `CastFailureChance(conflictMult)` (`L327-336`): `0.0` khi `conflictMult >= 0.70`, tối đa `0.25`.
  - `BackfireChance(conflictMult)` (`L340-349`): `0.0` khi `conflictMult >= 0.60`, tối đa `0.30`.

### 1.2. Khởi tạo Context trận đấu (`CombatState` & `CombatantAuto`)

- **`simulator.CombatState`** (`backend/internal/simulator/models.go:732-761`):
  - Hiện tại chỉ chứa `Turn`, `Seed`, `MaxGuActionsPerTurn`, `Allies`, `Enemies`, `CaptureTurnSnapshots`, `FastResolve`, `BossPhase`, `VirtualMode`, `ApertureHP`, `ApertureMaxHP`, `InitialLogs`, `Interactive`, `FledBy`.
  - **Chưa có** `RegionID`, `WorldMapID`, hay trường Đạo Ngân môi trường nào trên `CombatState`.
- **`buildSimCombatState`** (`backend/internal/handlers/combat_sim_build.go:26-150`):
  - Là điểm dựng `CombatState` duy nhất cho cả **Auto-Combat** (`settleCombat` trong `backend/internal/handlers/combat_auto_resolve.go:117`) và **Interactive Turn-Based** (`backend/internal/handlers/combat_session.go:841`).
  - Nhận `player *models.Player` (có `player.CurrentRegion`), dựng `simPlayer` qua `buildSimCombatant` (`backend/internal/handlers/combat_auto.go:1309-1405`) và địch qua `buildSimEnemyCombatant` (`backend/internal/handlers/combat_auto.go:197-340`).
  - `buildSimCombatant` (`combat_auto.go:1345-1351`) gọi `gamefx.BuildDaoCombatProfileWithReduction(...)`.
  - `buildSimEnemyCombatant` (`combat_auto.go:334-340`) đặt `simEnemy.HasNonConflictingDao = true` và gọi `gamefx.BuildDaoCombatProfile(...)`.
- **`CalculateGuEffectForCombatant`** (`backend/internal/handlers/combat_effects.go:91-134`):
  - Khi `applyDao == false` (luồng simulator), giữ nguyên chỉ số gốc của Cổ trùng để `simulator` tự nhân Đạo Ngân lúc giao tranh.
  - Khi `applyDao == true` (hành động ngoài simulator), nhân sẵn `daoMultiplier = (1.0 + (daoAmp-1.0)*realmEff) * conflictMult` (`L117-120`).
- **`SuccessChance` của Sát Chiêu (`KILLER_MOVE`)**:
  - Tính lúc build action tại `backend/internal/handlers/combat_auto.go:1122` qua `composition.AdjustedSuccessChance(...)` và lưu vào `SimAction.SuccessChance` (`simulator/models.go:138`).

### 1.3. Tính toán trong `simulator` (`engine.go`, `engine_action_runner.go`, `dao_damage.go`)

- **`Simulate(state *CombatState)`** (`backend/internal/simulator/engine.go:1263`): chạy vòng lặp chiến đấu tất định bằng `rng := rand.New(rand.NewSource(state.Seed))` (`L1273`).
- **`actorDaoMultiplier(c, paths)`** (`engine.go:946-961`): gọi `ensureDaoProfile(c)` (`L877-893`) rồi `c.DaoProfile.GetOffensiveFactors(pathKey)` để trả về `(1.0 + (daoAmp-1.0)*realmEff) * conflictMult`.
- **`actionConflictMult(c, paths)`** (`engine.go:897-911`): trả về `conflictMult` (`1.0` nếu `HasNonConflictingDao`).
- Trong **`combatActionRunner.runOne`** (`backend/internal/simulator/engine_action_runner.go:27`):
  1. **Mana Tax (`ManaTaxMultiplier`)**: `L175-179` tính `conflictMult := actionConflictMult(actor, a.Path)` và nhân `effectiveManaCost *= gamefx.ManaTaxMultiplier(conflictMult)`. (Lưu ý: `backend/internal/simulator/interactive.go:21, 172` cũng dùng cùng công thức để kiểm tra/trừ mana cho phiên turn-based).
  2. **Cast Failure & Backfire do xung đột Đạo Ngân**: `L250-261` kiểm tra `gamefx.CastFailureChance(conflictMult)` (`DAO_CAST_FAILED`); `L266-300` kiểm tra `gamefx.BackfireChance(conflictMult)` (`DAO_BACKFIRE`, gán `conflictPenaltyMult = 0.5`).
  3. **`SuccessChance` (Killer Move)**: `L302-356` tung `roll := r.rng.Float64()`; nếu `roll >= chance` (`a.SuccessChance`), gây `BackfireDamage` (mặc định `12% MaxHP`) và log `KILLER_MOVE_FAILED`.
  4. **Hệ số Đạo Ngân chung của hành động**: `L616`: `daoMult, hasDao := actorDaoMultiplier(actor, a.Path)`.
  5. **Shield**: `L622-676`: `baseShield = a.Defense * combinedMult`, `daoShield = baseShield * (daoMult - 1.0)` (nếu `hasDao && daoMult > 1.0`), `totalShield = baseShield + daoShield` cộng vào `actor.VirtualShield`.
  6. **Healing**: `L679-708`: `daoHeal = baseHeal * (daoMult - 1.0)` (nếu `hasDao && daoMult > 1.0`), `heal := (baseHeal + daoHeal) * effectiveHealingReceived(actor)` (`dao_damage.go:50-59`: `1 + c.HealingReceived - c.HealingReceivedPenalty`). DoT regen cũng dùng `effectiveHealingReceived(actor)` tại `engine.go:2166`.
  7. **Damage**: `L3247-3286`:
     - `hitPath := daoPathForAction(a.Path)` (`dao_damage.go:10-19`, fallback `"strength"`).
     - `natResVal, targetResonance := targetDaoFactors(target, hitPath)` (`dao_damage.go:23-33`).
     - `hitDmg := a.Damage * skillMult * dmgMultiplier * daoMult * conflictPenaltyMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path)) * (1.0 + raceDamageBonus) * targetResonance`.
     - `netDmg := hitDmg - targetDef`, sau đó nhân `(1.0 - natResVal) * (1.0 - pathBonus(target.PathResistance, a.Path)) * (1.0 - raceResistance)`, qua `ResolveBaseDamage(netDmg, 0)` và `applyDamageToTarget(...)`. Nếu `hpLost > 0 && natResVal > 0`, gọi `applyForeignDaoWound(target)` (`dao_damage.go:35-48`).

### 1.4. Điểm neo gắn Đạo Ngân Môi Trường vào Combat

1. **Truyền ngữ cảnh**: Bổ sung thông tin vùng/tầng hoặc snapshot Đạo Ngân môi trường vào `simulator.CombatState` (`simulator/models.go:732`) tại `buildSimCombatState` (`combat_sim_build.go:26`, nơi có sẵn `player.CurrentRegion` và `tx`).
2. **Hook tính toán trong `simulator`**:
   - **Tỉ lệ thành công & tiêu hao Chân Nguyên**: `engine_action_runner.go:175-179` (`ManaTaxMultiplier`), `L250-261` (`CastFailureChance`), `L307-314` (`SuccessChance`), và đồng bộ với `interactive.go:21, 172`.
   - **Sát thương / Khiên / Hồi phục**: `actorDaoMultiplier` (`engine.go:946`) hoặc biến `daoMult` tại `engine_action_runner.go:616` (tác động đồng nhất lên Damage, Shield, Heal), hoặc tách hệ số cộng hưởng/áp chế môi trường riêng tại `Damage` (`L3263`), `Shield` (`L638`), `Healing` (`L681`).

---

## 2. Kiến trúc `regions`, `map_cells`, `world_maps` và World Change Feed

### 2.1. Lưu trữ Schema (`backend/internal/db/schema_world.go`)

- **`world_maps`** (`L18-66`): quản lý các tầng thế giới (`dia_uyen`, `world_map`, `bai_thien`, `hac_thien`), gồm `map_type`, `width`, `height`, `min_rank`, `max_rank`, `rules JSONB`, `is_active`.
- **`regions`** (`L72-98`): phân cấp theo `parent_id` và `zone_type` (`HUB`, `SAFE_ZONE`, `BOSS_ZONE`, `SECRET_REALM`, `DUNGEON`, `PVP_ZONE`, `NGU_VUC`, `KHU_VUC`), tọa độ `min_x..max_y`, `entry_x/y`, `safe_zone_x/y`, `atlas_x/y`, `pool_weights JSONB`, `paths JSONB`, `rules JSONB`.
- **`region_world_maps`** (`L102-110`): bảng nối `region_id` $\rightarrow$ `world_map_id` (mặc định `'world_map'`).
- **`map_cells`** (`L179-203`): ô bản đồ bền vững với unique index `(region_id, x, y)`, lưu `cell_type`, `content_id`, `metadata JSONB`, `respawn_seconds`, `last_cleared_at`, `cleared_by`, `is_active`.

### 2.2. Nạp vào bộ nhớ, Import/Export & Chỉnh sửa

- **Model**: `models.Region` (`backend/internal/models/region.go:21-52`) ánh xạ `WorldMapID`, `PoolWeights` (`ZonePools`), `Paths []string`, `Rules json.RawMessage`, `EffectiveRules`.
- **In-memory Registry**: `zoneReg` (`backend/internal/handlers/zone_registry.go:42-50`) được nạp qua `LoadZoneRegistry(db.DB)` (`L83-160+`) khi khởi động: query `regions LEFT JOIN region_world_maps` (`WHERE is_active = TRUE`), `map_cells`, và `monster_spawns`.
- **Admin CRUD** (`backend/internal/handlers/admin_world.go`):
  - `AdminListRegions` (`L58`), `AdminCreateRegion` (`L165`), `AdminUpdateRegion` (`L253`), `AdminDeleteRegion` (`L366`): validate `RegionRules` qua `validateRegionRules`, cập nhật `regions` + `region_world_maps` trong transaction, sau đó gọi `LoadZoneRegistry(db.DB)` và `invalidateCachedDictionaryPart("explore_regions_v2")`.
  - `map_cells` CRUD nằm tại `admin_world.go:433-672`.
- **CSV Import/Export**:
  - `backend/internal/db/import_workflow.go:79, 226`, `backend/internal/db/datasets.go:115`, `backend/internal/db/db.go:1985-2210, 3692-3694`: dataset `"region"` đọc/ghi `Region.csv` vào `regions` và `region_world_maps`.

### 2.3. Kiểm tra luật vùng (`region_policy.go`) & Di chuyển (`movement.go`)

- **`region_policy.go`**:
  - `RegionRules` (`L15-34`): cấu trúc JSON cộng dồn gồm `rank` (`min`/`max`), `paths` (`allow`/`deny`), `gu` (`max_rank`, `deny_ids`), `items` (`deny_ids`), `actions` (`deny`).
  - `regionRulesFor(regionID)` (`L165-212`): duyệt `ParentID` từ gốc xuống con để gộp từng nhóm rule, dùng `regionRulesDecodeCache` (`L130-163`, tự invalidate khi raw bytes `region.Rules` thay đổi).
  - Các hàm kiểm tra: `checkRegionAction` (`L214`), `checkRegionEntry` (`L228`), `checkRegionGu` (`L242`), `checkRegionEquippedGus` (`L279`), `checkRegionItem` (`L295`).
- **`movement.go`**:
  - Chuỗi dùng chung (`L31-42`): `checkMovement` (`L138-180`) $\rightarrow$ `applyDestination` (`L216-234`) $\rightarrow$ `ExecuteSpatialTransition` (`L265-303`).
  - Khi đổi `region`, `applyDestination` kiểm tra `worldMapIDForRegion` để áp dụng `applyLayerTransitionEffects` (nếu khác tầng `world_maps`) hoặc `applyRegionalWallDebuff` (nếu khác `NGU_VUC`). Post-commit gọi `broadcastPlayerWorldPos`, `PublishPlayerPositionDelta`, `PublishPlayerRealtimeEvent`.

### 2.4. World Change Feed (SSE / WebSocket)

- **`backend/internal/worldfeed/event.go`**:
  - Kênh Redis: `RedisChannel = "world:change"` (`L20`).
  - `Scope{RegionID, WorldMapID, InstanceID}` (`L24-28`) và `Event{EventID, EntityType, Action, Scope, X, Y, Data, Epoch, Revision}` (`L40-50`).
  - `PublishBatch` (`L74-127`): xác định danh sách người chơi nhận qua `resolveTargets(scope)` (`L186-227`, truy vấn đệ quy `region_scope` theo `parent_id` + `world_map_id` + `map_instance_id` cho player hoạt động trong 5 phút gần nhất), cấp số thứ tự Redis theo `RevisionKey(scope)` (`L131-138`), và publish `Envelope{Deliveries}` lên `world:change`.
- **`backend/internal/handlers/world_change_events.go`**:
  - Các helper `publishWorldChange` (`L85`), `publishWorldChanges` (`L92`), `publishWorldCellChange` (`L119`), `publishAperturePortalMarkerChange` (`L128`), và gắn header `X-World-Revision` / `X-World-Epoch` qua `setWorldSnapshotRevision` (`L35-74`).
- **`backend/internal/uihub/redis_sub.go`**:
  - `StartRedisSubscribers` (`L16-27`) subscribe `world:*`. Với `world:change` (`L80-99`), decode `Envelope` và gọi `shub.BroadcastToTargets(ChannelName(clientChannel), clientPayload, targets)` để đẩy xuống client qua WebSocket/SSE. (Còn dịch chuyển vị trí nhân vật đi qua kênh `world:pos` tại `world_events.go:67-117` và `redis_sub.go:62-79`).

---

## 3. Cơ chế Hạ Khiếu (`is_placed`, `placed_region`) và Tiên Khiếu Công Cộng

### 3.1. Schema (`backend/internal/db/schema_aperture.go`)

- **`immortal_apertures`** (`L9-48`):
  - `player_id` (`UUID UNIQUE REFERENCES players(id) ON DELETE SET NULL`, cho phép `NULL` với Tiên Khiếu vô chủ/công cộng).
  - `public_state` (`PRIVATE` / `PUBLIC` / `RUIN` / `CLOSING`, `L13`, `L100`).
  - `dao_marks JSONB`, `heaven_qi`, `earth_qi`, `max_heaven_qi`, `max_earth_qi`.
  - Trường Hạ Khiếu: `is_placed`, `placed_region`, `placed_x`, `placed_y`, `placed_at`, `return_region`, `return_x`, `return_y` (`L37-44`).
- **`public_aperture_site_configs`** (`L110-130`): cấu hình điểm Tiên Khiếu công cộng tĩnh trên bản đồ (`site_key`, `region_id`, `x`, `y`, `aperture_rank`, `grade`, `area_size`, `dao_marks`, `lifecycle_state` `UNSPAWNED`/`ACTIVE`).

### 3.2. Cơ chế Hạ Khiếu & Thu Khiếu (`aperture_core.go`, `aperture_layer_qi.go`)

- **Hạ Khiếu** (`backend/internal/handlers/aperture_core.go:460-558`):
  - Chặn nếu đang ở trong Tiên Khiếu khác (`L462`), đã `is_placed` (`L477`), hoặc `DESTROYED` (`L482`).
  - Ghi `is_placed = true`, `placed_region = player.CurrentRegion`, `placed_x/y`, `placed_at = NOW()`, `return_region/x/y` (`L488-493`).
  - Upsert `aperture_portals` (Môn Hộ) tại vị trí hạ khiếu (`L500-506`), chuyển thân thể Cổ Tiên vào instance `aperture_interior` (`map_instance_id = ap.ID`, `L513-522`), và phát `publishAperturePortalMarkerChange("portal_spawn", player.CurrentRegion)` (`L544`).
- **Thu Khiếu (`UnanchorAperture`)** (`aperture_core.go:562-754`):
  - Chặn nếu đang combat (`L570-579`), còn ngoại nhân bên trong (`aperture_guest_roster`, `L623-628`), hoặc đang độ kiếp (`L631-635`).
  - Tính `elapsedMinutes` từ `placed_at`, lấy `layer := worldMapIDForRegion(destRegion)` (`L658`) và `LayerQiMultipliers(layer)` (`backend/internal/handlers/aperture_layer_qi.go:14-25`):
    - `dia_uyen`: Thiên Khí `0.20`, Địa Khí `3.50`
    - `bai_thien`: Thiên Khí `3.00`, Địa Khí `0.30`
    - `hac_thien`: Thiên Khí `1.00`, Địa Khí `0.50`
    - `world_map` (mặc định): `1.00` / `1.00`
  - Nạp `+1% max/phút * multiplier` cho Thiên/Địa Khí (`L662-666`). Riêng `hac_thien` với `elapsedMinutes >= 5.0` tẩm bổ thêm `Dark Path` và `Star Path` Dao Marks (`elapsedMinutes / 10.0`, `L674-685`).
  - Đóng `aperture_portals`, reset `is_placed = false`, `placed_* = NULL`, đưa player về `destRegion (destX, destY)` và phát `publishAperturePortalMarkerChange("portal_despawn", destRegion)` (`L735`).
- **`aperture_layer_qi.go`**: ngoài `LayerQiMultipliers`, còn có `LayerTribulationTimeRateMultiplier(layer)` (`L33-38`, `hac_thien` = `1.0/1.5`, các tầng khác `1.0`) và `LayerPlacementDescription` (`L42-56`).

### 3.3. Tiên Khiếu Công Cộng

- **Khởi tạo từ site cấu hình** (`backend/internal/handlers/aperture_public_sites.go:18-128`): `EnsureConfiguredPublicApertures` $\rightarrow$ `materializePublicApertureSite` tạo `immortal_apertures` (`player_id = NULL`, `public_state = 'PUBLIC'`, `is_placed = TRUE`, `placed_region = regionID`), sinh điểm tài nguyên (`GenerateApertureResourcePoints`), mở `aperture_portals` (`access_policy = 'OPEN'`), chuyển site sang `ACTIVE` và phát `portal_spawn`.
- **Độ kiếp tự nhiên** (`backend/internal/handlers/aperture_public_tribulation.go:26-167`): `ProcessPublicApertureTribulations` đẩy `internal_years_passed` cho các Tiên Khiếu `player_id IS NULL AND public_state = 'PUBLIC'` và tự động tất toán tai kiếp môi trường (`FAILED`/`RESOLVED`, `archetype = 'ENVIRONMENTAL'`), trừ `aperture_hp`.
- **Sụp đổ thành Tàn Tích (`RUIN`)** (`backend/internal/handlers/aperture_lifecycle.go:40-118`): `TransitionCollapsedPublicApertures` chuyển Tiên Khiếu `PUBLIC` có `aperture_hp <= 0` sang `public_state = 'RUIN'` với hạn `public_ruin_expires_at`, hủy tai kiếp đang chạy và phát `worldfeed` `aperture_ruin`.
- **Truy cập Môn Hộ** (`backend/internal/handlers/aperture_portal.go:184-220`): cho phép vào khi `target_aperture.player_id IS NULL AND target_aperture.public_state IN ('PUBLIC', 'RUIN')` (có kiểm tra hết hạn tàn tích).
