# WF-01 Research Report: Worldmap Presence, FOV, Profile Access, and Dao/Scouting Mechanics

> Khảo sát lịch sử của luồng legacy. `GET /api/explore/players` và consumer `WorldCanvas.tsx` được mô tả bên dưới đã bị gỡ khỏi code hiện hành; Phaser MMO dùng realtime WebSocket/AOI.

## 1. Worldmap Presence & FOV Logic

### 1.1 Coordinates & Persistence
- **Storage**: Player location is stored in PostgreSQL table `players` via columns `current_region` (VARCHAR), `x` (INT), `y` (INT), along with `last_mana_update` / `last_activity` (TIMESTAMPTZ).
- **In-Memory Spatial Index (`PositionIndex`)**:
  - Located in `backend/internal/handlers/world_index.go`.
  - In-memory grid with bucket size `WorldPosIndexBucket = 300` cells.
  - Active window: `WorldPosActiveWindow = 15 * time.Minute` based on timestamp `TS` (unix ms).
  - Methods: `Set(WorldPosEntry)`, `Get(id)`, `Nearby(region, x, y, r, excludeID)`, `Reconcile(entries)`.
- **Call-Sites Updating Coordinates & Position Index**:
  1. `exploration_move.go`: `MovePlayer` & `MovePlayerPath` invoke `broadcastWorldPos(WorldPosEvent{...})`.
  2. `player_auth_register.go`: `CompleteOnboarding` invokes `setWorldPosFromPlayer(&player)`.
  3. `combat_explore.go`: Exploration node movement invokes `setWorldPosFromPlayer(&player)`.
  4. `combat_retreat.go`: Retreat, flee success/failure, and defeat escape invoke `setWorldPosFromPlayer(&player)`.
  5. `combat_auto.go`: Defeat teleport to NearestHub invokes `setWorldPosFromPlayer(&player)`.
  6. Periodic Reconcile Worker: `StartWorldIndexSyncWorker` (`world_index_sync.go`) runs every 30s querying active players (`last_mana_update >= NOW() - INTERVAL '16 minutes'`) to bootstrap and self-heal the index.

### 1.2 Nearby Players & Real-time Fan-out
- **API Endpoint**: `GET /api/explore/players?x=&y=&r=` (`handlers.GetNearbyPlayers` in `exploration_views.go`).
  - Maximum query radius `r` capped at `300`.
  - **Server-Authoritative AOI**: The server forces coordinates from `worldPosIndex.Get(authID)` (or DB fallback), ignoring client-supplied query coordinates to prevent arbitrary inspection across the region.
  - Returns: `NearbyPlayer { id, username, avatar_url, x, y }`.
  - Fast-path queries `worldPosIndex.Nearby(...)`; falls back to `queryNearbyPlayersSQL` if index is cold (`Len() == 0`).
- **WebSocket / SSE Broadcast**:
  - `broadcastWorldPos` (`world_events.go`) queries `worldPosIndex.Nearby(region, x, y, WorldPosInterestRadius=300, moverID)` and sends targeted WebSocket/SSE events `world:pos` only to subscribers within radius.
- **Frontend FOV & Rendering**:
  - Store `useWorldStore.ts`: tracks `nearbyPlayers`, `playerPositions` (with local timestamp pruning after 60s), and `investigationRange` (default min 10).
  - Canvas `WorldCanvas.tsx`: uses `investigationRange` to request `fetchNearbyCells` and `fetchNearbyPlayers`, rendering player markers, boss labels, and cells within `Math.hypot(dx, dy) <= investigationRange`.

---

## 2. Existing Profile Endpoints and Consumers

### 2.1 Backend Endpoints
1. **`GET /api/player/profile/:username`** (`handlers.GetPlayerProfile` in `profile.go`):
   - Group rate limit: 60/min.
   - Redis caching: `profile_<viewerID>_viewing_<targetUsername>` with TTL 60s.
   - Compares Viewer `scoutingLevel` vs Target `disguiseLevel`:
     - Viewer Scouting Level = `viewerRank` + $\sum$(`Investigation` on equipped active Gus) + `PathMasteryBonus(Wisdom, Information)`.
     - Target Disguise Level = `targetRank` + $\sum$(`DisguiseLevel` on equipped active Gus) + `PathMasteryBonus(Wisdom, Dark)`.
     - $diff = \text{scoutingLevel} - \text{disguiseLevel}$:
       - **Full Success ($diff \ge 2$)**: Returns true username, rank, subrank, aptitude, exact mana/HP, stones, sect name, all equipped Gus, `DaoMarks`, `PathMasteries`, `Statuses`.
       - **Partial Success ($0 \le diff < 2$)**: Returns true username & rank, `???` for aptitude/mana/stones/HP, 2 random equipped Gus + `???`, empty `DaoMarks` and `PathMasteries`.
       - **Failure ($diff < 0$)**:
         - If `target.IsDisguised`: Returns fake settings configured by target (`FakeRank`, `FakeSubRank`, `FakeVitalGu`, canned bio).
         - Else if target has `PERFECT_DISGUISE` capability (Attitude Gu / Thái Độ Cổ): Returns spoofed profile ("Ẩn Danh Cổ Sư", Rank 1, "Đinh" 25%, `GU_SHADOW`, Wisdom Dao Marks 50, Wisdom Mastery Master).
         - Else: Scrambles username (`ab****`), rank $\pm 1$, `???` stats.
2. **`PUT /api/player/profile/settings`** (`handlers.UpdateProfileSettings` in `player_profile.go`):
   - Allows updating `username`, `bio`, `new_password`, `fake_rank`, `fake_sub_rank`, `fake_vital_gu`, `is_disguised`.
   - Requires active Gu with `DISGUISE` capability (`InventoryHasGuMechanic(..., "DISGUISE", true)`).
   - If player lacks `PERFECT_DISGUISE`, `fake_rank` cannot exceed actual `rank` ("giả yếu"). With `PERFECT_DISGUISE`, can fake rank 1–9.
   - `fake_vital_gu` requires `CHANGE_FORM_DISGUISE` or `PERFECT_DISGUISE`.

### 2.2 Frontend Consumers & UI Trigger Points
- **Store Slice**: `frontend/src/store/slices/profileSlice.ts`:
  - Actions: `fetchPlayerProfile(username)`, `updateProfileSettings(settings)`, `uploadAvatar(file)`.
  - State: `inspectedProfile`, `inspectedUsernameQuery`, `profileLoading`.
- **Consumers calling `fetchPlayerProfile`**:
  1. `GlobalChatOverlay.tsx`: Clicking on sender username in chat messages.
  2. `LeaderboardView.tsx`: Clicking on rank rows in global leaderboards.
  3. `ArenaView.tsx`: Clicking on rival entries in arena leaderboard.
  4. `SectView.tsx`: Clicking on sect members or membership applicants.
  5. `PartyView.tsx`: Clicking on party members.
  6. `ExplorationMap.tsx` / `WorldPage.tsx`: Clicking on player markers on the world map.
- **UI Modal**: `frontend/src/components/PlayerProfileModal.tsx`:
  - Rendered at top level in `app/page.tsx` when `inspectedProfile !== null`.
  - Shows stats, aptitude, Gus, Dao Marks, Path Masteries, Statuses, and a button to initiate direct P2P Trade (`initiateTrade(inspectedUsernameQuery)`).

---

## 3. Dao Marks, Path Masteries, and Gu Scouting/Disguise Effects

### 3.1 Dao Marks (`dao_marks`)
- **Structure**: `models.DaoMarksMap` (`map[string]int64`), e.g. `{"strength": 50000, "sword": 12000, "wisdom": 8000}`.
- **Persistence**:
  - For Immortals (Rank $\ge 6$), stored in PostgreSQL table `immortal_apertures.dao_marks` (JSONB).
  - Read via `player_daongan.go` (`CalculateDaoNgan`: $power = marks \times (1.0 + \frac{marks}{100})$).

### 3.2 Path Masteries (`path_masteries`)
- **Structure**: `models.PathMasteriesMap` (`map[string]int`), stored on `players.path_masteries` (JSONB).
- **Attainment Levels (`gamefx/path_mastery.go`)**:
  - `Level 0: Ordinary (Phàm Thai)`: $< 100$ pts (Multiplier 1.0)
  - `Level 1: Master (Đại Sư)`: $\ge 100$ pts (Multiplier 1.1)
  - `Level 2: Grandmaster (Tông Sư)`: $\ge 1,000$ pts (Multiplier 1.25)
  - `Level 3: Great Grandmaster (Đại Tông Sư)`: $\ge 10,000$ pts (Multiplier 1.4)
  - `Level 4: Supreme Grandmaster (Vô Thượng Đại Tông Sư)`: $\ge 100,000$ pts (Multiplier 1.5)
- **Path Mastery Gain Calculation**: `CalculatePathMasteryGain` scales gain down by current level ($1.0 - 0.20 \times \text{level}$) and applies rank gating.

### 3.3 Gu Scouting, Disguise & Mechanics
- **`Investigation` Stat**:
  - Defined in `models/gu.go` (`PassiveEffects.Investigation`).
  - Found in ~100+ Gus in `Gu.csv` (e.g. Nhục Nhĩ Cổ +1, Địa Thính Nhục Nhĩ Thảo +2, Thính Phong Cổ +2, Hồn Thính Cổ +15, Thần Tích Cổ +500).
- **`DisguiseLevel` Stat**:
  - Defined in `models/gu.go` (`PassiveEffects.DisguiseLevel`). E.g., Thái Độ Cổ (Attitude Gu) rank 8 has `disguise_level: 8`.
- **Capability Action Codes** (`effectcatalog/manifest.go`):
  - `DISGUISE`: Standard disguise permission.
  - `CHANGE_FORM_DISGUISE`: Allows spoofing fake vital Gu.
  - `PERFECT_DISGUISE`: Thái Độ Cổ (Attitude Gu), allows faking tu vi 1–9, complete stealth, spoofing profile on scouting failure.
  - `ARENA_IDENTITY_MASK`: Hides name in PvP Arena.
