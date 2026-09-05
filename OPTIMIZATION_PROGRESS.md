# GuWorld Optimization Progress Tracking

## Phase 1: Quick Wins (High Impact, Low Risk) - 1-2 days

### Backend - Helper Functions Extraction
- [x] **Extract Gu inventory helper functions** in `models/utils.go`
  - `FindEquippedGuByID(inventory GuInventory, guID string) (*GuInstance, int, bool)`
  - `CountEquippedGuByID(inventory GuInventory, guID string) int`
  - `HasEquippedGu(inventory GuInventory, guID string) bool`
  - `FindGuByInstanceID(inventory GuInventory, instanceID string) (*GuInstance, int, bool)`
  - `FindGuByIDAndRank(inventory GuInventory, guID string, rank int) (*GuInstance, int, bool)`
  - `FilterEquippedActive(inventory GuInventory) GuInventory`
  - `RemoveGuByInstanceID(inventory GuInventory, instanceID string) GuInventory`

- [x] **Extract error handling helper** in `models/utils.go` or new `handlers/utils.go`
  - `HandleError(c *gin.Context, err error, msg string) bool` - returns true if handled

- [x] **Extract transaction helper** in `db/db.go` or new `handlers/utils.go`
  - `WithTx(ctx context.Context, fn func(*sql.Tx) error) error`

### Backend - Database Indexes
- [x] **Add GIN indexes on players JSONB columns** in `schema_players.go`
  - `idx_players_active_combat` on `active_combat` 
  - `idx_players_statuses` on `statuses`
  - `idx_players_equipped_gus` on `equipped_gus`
  - `idx_players_path_masteries` on `path_masteries`
  - `idx_players_body_mods` on `body_mods`

- [x] **Add composite/partial indexes** in `schema_players.go`
  - `idx_players_region_xy_active` on `(current_region, x, y)` WHERE `is_active = true`
  - `idx_players_party_id_active` on `(party_id, is_active)` WHERE `party_id IS NOT NULL`

- [x] **Add indexes on player_gu table** in `schema_player_items.go`
  - `idx_player_gu_equipped_active_full` on `(player_id, gu_id, gu_rank)` WHERE `equipped = true AND state = 'ACTIVE'`
  - `idx_player_gu_location_state` on `(state, location)` WHERE `state IN ('ACTIVE', 'SLEEPING', 'EQUIPPING')`

- [x] **Add indexes on combat tables** in `schema_combat.go`
  - `idx_combat_sessions_status` on `status` WHERE `status = 'ACTIVE'`
  - `idx_combat_history_player_created` on `(player_id, created_at DESC)`
  - `idx_combat_history_winner_created` on `(winner, created_at DESC)`
  - `idx_arena_rankings_points` on `points DESC`
  - `idx_arena_rankings_player` on `player_id`

- [x] **Add indexes on sect tables** in `schema_sects.go`
  - `idx_sect_members_sect_rank` on `(sect_id, member_rank)`
  - `idx_sect_promotion_requests` on `(sect_id, status, created_at)`

- [x] **Adjust connection pool** in `db.go`
  - Increase `MaxOpenConns` to 50-100
  - Increase `MaxIdleConns` to match

### Frontend - Bundle Size
- [x] **Create lucide icon barrel export** in `src/lib/icons.ts`
- [x] **Dynamic imports for all views** in `page.tsx`
- [x] **Optimize icon.png** — resized from 512x512 (476KB) to 256x256 (127KB), 73% reduction. Kept as PNG for favicon compatibility.

---

## Phase 2: Core Optimization (High Impact) - 3-5 days

### Frontend - Zustand Store Split
- [x] **Created useUIStore** (`src/store/useUIStore.ts`) for UI-only state:
  - `toasts`, `addToast`, `removeToast`
  - `confirmDialog`, `showConfirm`, `closeConfirm`
  - `currentView`, `setView`
  - `showChangelog`, `setShowChangelog`
- [x] **Updated useGameStore** to proxy UI actions to `useUIStore`
  - `addToast` → `useUIStore.getState().addToast(translateIDs(msg), ...)`
  - `setView` → `useUIStore.getState().setView(view)`
  - `showConfirm` → `useUIStore.getState().showConfirm(...)`
  - etc.
- [x] **Updated useWorldStore** to use `useUIStore` for toasts
- [x] **Updated ConfirmDialog.tsx** to use `useUIStore`
- [x] **Updated ToastContainer.tsx** to use `useUIStore`
- [ ] **Split remaining domains** (combat, trade, sect, etc.) — deferred to Phase 3

### Frontend - Selector Pattern & Memoization
- [x] **Selector pattern in page.tsx** — replaced 70-field destructure with `useShallow` array selector
- [x] **Memoize translateIDs** — added LRU cache (max 500 entries) with dictionary versioning
- [x] **UI state separated** — `currentView`, `showChangelog` now from `useUIStore`, preventing re-renders on toast/confirm changes

### Frontend - Component Splitting (Phase 3)
- [x] **Extracted CombatScreen pure helpers** to `src/lib/combat.ts`
  - `rankLabel`, `subRankLabel`, `typeColor`, `gridXLabel`
  - `EFFECT_STYLE`, `EFFECT_LABEL`, `getCombatEffects`, `EffectBadge`
  - Removed ~120 lines of duplication from CombatScreen.tsx

### Frontend - Component Optimization
- [ ] **WorldCanvas graphics reuse** - reuse Graphics objects instead of recreating
- [x] **CombatScreen memoization** - memoized `eligibleGus` with `useMemo` + `useCallback` for `isCombatGu`, inlined `computeInvestigation` to eliminate unnecessary closure. `recommendLoadout` already wrapped in `useCallback`.
- [x] **GuInventory consolidation** — analyzed 22 `useState` calls; individual `useState<string|null>` is already lightweight and avoids cascading re-renders. Consolidation would risk regressions without measurable gains. Marked as intentionally skipped.
- [x] **Request deduplication layer** - pendingFetches Map (completed in Phase 4)

---

## Phase 3: Deep Refactoring (Maintainability) - In Progress

### Backend - Use Extracted Helpers in Handlers
- [x] **Replaced `EquippedGus` rebuild loops** (8 instances) with `models.FilterEquippedActive()`
  - `player.go`: 3 instances
  - `shop.go`: 2 instances
  - `gu_effects.go`: 1 instance
  - `refine.go`: 1 instance
  - `marketplace.go`: 1 instance
- [x] **Replaced `equippedCount` loops** (2 instances) with `models.CountEquippedActive()`
  - `player.go`: 2 instances
- [x] **Replaced `hadVital` loop** (1 instance) with `models.FindFirstMatchingGu()`
  - `player.go`: 1 instance
- [x] **Replaced remaining patterns** across all handler files:
  - `marketplace.go`: 5 `GU_CONNECTING_HEAVEN` loops → `models.FindFirstMatchingGu()`
  - `profile.go`: 4 equipped-active filter loops → `models.FilterEquippedActive()`, 1 `GU_ATTITUDE` check → `models.HasEquippedGu()`
  - `trade.go`: 4 `GU_CONNECTING_HEAVEN` loops → `models.FindFirstMatchingGu()`, 2 InstanceID loops → `models.FindGuByInstanceID()`, 2 deduction loops → `models.FindGuByInstanceID()` + `models.RemoveGuByIndex()`
  - `exploration.go`: 2 equipped checks (`GU_HEAVEN_SECRET`, `GU_FIXED_IMMORTAL_TRAVEL`) → `models.HasEquippedGu()`, 1 filter → `models.FilterEquippedActive()`
  - `combat_auto.go`: 2 equipped-active filters → `models.FilterEquippedActive()`
  - `combat_auto_resolve.go`: 1 equipped-active filter → `models.FilterEquippedActive()`
  - `gu_effects.go`: `RemoveGuFromInventory` loop → `models.RemoveGuByInstanceID()`
  - `player_sacrifice.go`: 1 `GU_BLOOD_SKULL` check → `models.HasEquippedGu()`
  - `refine.go`: 1 filter → `models.FilterEquippedActive()`, 1 InstanceID → `models.FindGuByInstanceID()`
  - `killer_move.go`: 1 `GU_SOVEREIGN_FETUS` check → `models.HasEquippedGu()`, 1 InstanceID → `models.FindGuByInstanceID()`
  - `admin.go`: 1 InstanceID → `models.FindGuByInstanceID()`

### Backend - Component Splitting (Phase 3)
- [x] **Split combat_auto.go** (2374 lines)
  - `combat_auto.go` (1288 lines) - AutoCombatStart + types + buildSimCombatant
  - `combat_auto_resolve.go` (1108 lines) - AutoResolveCombat + helpers

### Frontend - Component Splitting (Phase 3)
- [x] **Extracted ImmortalApertureView sub-components**
  - `CountdownTimer.tsx` (45 lines) - extracted from ImmortalApertureView.tsx
  - `TribulationRealTimeInfo.tsx` (100 lines) - extracted from ImmortalApertureView.tsx
  - ImmortalApertureView.tsx reduced from 2759 to 2622 lines
- [x] **Extracted CombatScreen pure helpers** to `src/lib/combat.ts`
  - `rankLabel`, `subRankLabel`, `typeColor`, `gridXLabel`
  - `EFFECT_STYLE`, `EFFECT_LABEL`, `getCombatEffects`, `EffectBadge`
  - Removed ~120 lines of duplication from CombatScreen.tsx
- [ ] **Split player.go** (5296 lines) into:
  - `player_auth.go` - register, login, logout, OTP
  - `player_inventory.go` - equip, use items, drop
  - `player_exploration.go` - explore, feed
  - `player_progression.go` - onboarding, stats, leveling

- [ ] **Split admin.go** (4434 lines)
- [ ] **Split sect.go** (3839 lines)

- [ ] **Break down long functions** (>200 lines):
  - `RegisterPlayer` (697 lines)
  - `Explore` (756 lines)
  - `CompleteOnboarding` (611 lines)
  - `AutoCombatStart` (938 lines)
  - `AutoResolveCombat` (767 lines)

### Frontend - Component Splitting (Phase 5)
- [ ] **Split SectView.tsx** (3782 lines) — deferred (high risk, requires extensive testing)
- [ ] **Split AdminView.tsx** (5040 lines) — skipped per user request
- [ ] **Split CombatScreen.tsx** (1436 lines) — partial (helpers extracted to lib/combat.ts)
- [x] **Extract ImmortalApertureView sub-components** — CountdownTimer + TribulationRealTimeInfo extracted

---

## Phase 4: Architecture (Long-term) - Completed

- [x] **Request deduplication** in `useGameStore.ts`
  - Added `pendingFetches` Map + `fetchWithDedupe()` helper
  - Applied to `fetchDictionary` and `fetchNotifications`
  - Prevents concurrent duplicate API calls
- [x] **WorldCanvas interval consolidation**
  - Replaced 3 separate `setInterval` calls (30s, 2s, 3s) with 1 unified 1s ticker
  - Uses timestamp checks to schedule polls at correct intervals
  - Reduces timer overhead and simplifies cleanup
- [ ] Static content → SSR (Wiki, changelog) — deferred (requires larger refactor)
- [ ] Connection pool monitoring
- [ ] Metrics and tracing
- [ ] SSE per-feature EventSource management

---

## Validation Checklist (run after each phase)

```bash
# Backend
cd backend && go test ./...
cd backend && go test -race ./...
cd backend && go build ./...

# Frontend
cd frontend && npm run lint
cd frontend && npm run build
```

## ✅ Validation Results (All Phases)

| Check | Command | Result |
|-------|---------|--------|
| Backend build | `cd backend && go build ./...` | ✅ Pass |
| Backend tests | `cd backend && go test ./...` | ✅ Pass |
| Race detector | `cd backend && go test -race ./internal/models/ ./internal/db/ ./internal/handlers/` | ✅ Pass |
| Frontend build | `cd frontend && npm run build` | ✅ Pass |
| Frontend lint | `cd frontend && npm run lint` | ✅ Clean for new/modified files |