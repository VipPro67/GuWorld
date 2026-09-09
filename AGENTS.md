# AGENTS.md – GuWorld

## Startup

```bash
cd backend && go run ./cmd/server        # backend on :8080
docker-compose up                        # PostgreSQL on :5432
cd frontend && npm install && npm run dev  # frontend on :3000
```

**.env search order:** `.env` → `backend/.env` → `../.env` → `../../.env`
**DB defaults:** localhost:5432, user `gu_admin`, pass `gu_pass_123`, db `gu_db`
**CSV files live in `backend/data/`.** Import/export now done via Admin panel (tab "Dữ liệu") — no auto-seed on startup. All csvPath functions search `data/` first, then legacy paths.

## Tests & Checks

```bash
cd backend && go test ./...       # all backend tests
cd frontend && npm run lint       # ESLint 9
cd frontend && npm run build      # Next.js build
```

No CI/CD. No typecheck script. Test files are `*_test.go` in `backend/internal/handlers/`.

## Backend Conventions (agent would likely miss)

- **No service layer.** Handlers are fat (combat.go: 2400+ lines, player.go: 95KB). All gameplay logic is in `backend/internal/handlers/`.
- **No migrations.** Schema is split per domain in `backend/internal/db/schema_*.go` (each table a `CREATE TABLE IF NOT EXISTS` with all columns inline, plus indexes and FK). `InitSchema()` in `backend/internal/db/db.go` calls them in FK order and seeds static runtime data (system players, hub region `Southern_Border`, default race/constitution rows + `''` sentinel keys, 31 mine slots, 5 sect mines, wallet/leaderboard backfills). When adding **new columns** to existing tables, you MUST append an `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...` statement immediately after the `CREATE TABLE` to automatically patch existing databases without requiring the user to wipe their data. Modify SQL in `schema_*.go` accordingly. Per-region starting hubs (`hub_*`) come from `Region.csv` (imported via Admin); `CompleteOnboarding` maps each registration region to its hub (`startingRegionHubs` in `player.go`) and falls back to `Southern_Border`.
- **DB reset note:** `players.race`, `players.constitution`, `players.current_region` have FKs. `InitSchema` pre-seeds the `''` race/constitution keys and the `Southern_Border` region so registration works before CSV import; race/constitution registry loaders filter out `''` keys.
- CSV data is managed through the Admin panel.
- CSV files are not automatically seeded on startup.
- Changing CSV files requires importing the relevant dataset through the Admin panel.
- Registries are cached in memory on startup; restart the backend after data changes when required by the registry implementation.
- **Registries cached in memory** on startup (`models.GuRegistry`, `models.ItemRegistry`, `gamefx.ConstitutionRegistry`, `gamefx.RaceRegistry`). Changing CSV requires app restart.
- **Thể chất (constitution) là data-driven.** Danh sách seed từ `data/Constitution.csv` → bảng `constitution_dictionary`, effect resolve từ `gamefx.Effects` (JSON trong cột `effects`). Thêm thể chất mới: sửa CSV + Import bảng "Thể chất" trong Admin. Format effects xem `docs/GAME_DATA.md`.
- **Chủng tộc (race) là data-driven.** Danh sách seed từ `data/Race.csv` → bảng `race_dictionary`, effect resolve từ `gamefx.Effects` (JSON trong cột `effects`). Thêm chủng tộc mới: sửa CSV + Import bảng "Chủng tộc" trong Admin. `players.race` luôn là 1 key duy nhất (default `human`).
- **Workers are goroutine polling loops**, not job queues. Hunger, tribulation, mine monsters. All workers accept `context.Context` for graceful shutdown.
- **Redis is required.** Server fails to start without Redis.
- **JSONB fields** use `interface{}` with custom `Scan`/`Value`. Watch for type assertion panics. `CombatState.Scan` now returns empty struct (not nil) on null.
- **Handlers return errors** to Gin; no panic recovery middleware.

## Frontend Conventions

- **Zustand store is both state + API client.** `frontend/src/store/useGameStore.ts` has no separate service/API layer. Methods like `store.attackEnemy()` call fetch + update state.
- **Optimistic updates** in combat and some actions; handle rollback on failure.
- **SSE subscriptions** in Zustand for player events, trade, chat. Player events now have auto-reconnect with exponential backoff.
- **Next.js 16 has breaking changes.** Check `node_modules/next/dist/docs/` before writing framework-specific code.

## Documentation (update when changing code)

| Change | Update |
|--------|--------|
| Routes | `docs/API.md` |
| Schema/CSV | `docs/GAME_DATA.md` |
| Startup/commands/env | `docs/DEVELOPMENT.md` + `README.md` |
| Gameplay (player-visible) | `WIKI.md` |
| Project scope | `PROJECT.md` |

## Key File Pointers

- `backend/cmd/server/main.go` — entrypoint, startup orchestration
- `backend/internal/server/server.go` — all `/api/*` routes registered here
- `backend/internal/db/db.go` — `Connect` + `InitSchema` orchestrator + CSV seed/export logic
- `backend/internal/db/schema_*.go` — per-domain table DDL (columns, indexes, FK)
- `backend/internal/handlers/*.go` — all feature logic (no service layer)
- `backend/internal/simulator/engine.go` — deterministic auto-battle loop, combat log format
- `frontend/src/store/useGameStore.ts` — Zustand + API client + SSE
- `frontend/src/app/` — Next.js App Router pages
## AI Agent Workflow

### General Rules

* Read the relevant existing code before modifying it.
* Prefer extending existing patterns over introducing new abstractions.
* Do not introduce a service/repository layer unless explicitly requested.
* Keep gameplay logic in `backend/internal/handlers/`.
* Reuse existing helpers, registries, models, and database patterns.
* Do not modify unrelated files.
* Do not rewrite large files unnecessarily.
* Before changing database schema, inspect the existing schema in `backend/internal/db/schema_*.go`.
* Before changing gameplay values, inspect the corresponding CSV/data source and `docs/GAME_DATA.md`.

### Delegation

Use subagents for independent investigation before implementing complex features.

* `@explore`:

  * Search and understand existing backend/frontend implementations.
  * Locate related handlers, models, database tables, CSV data, and routes.
  * Do not modify code.
  * Prefer multiple independent `@explore` tasks when investigating unrelated areas.

* `@scout`:

  * Research external libraries, framework behavior, APIs, or documentation.
  * Do not modify project code.

* `@general`:

  * Perform independent technical analysis.
  * Compare implementation approaches.
  * Identify edge cases and potential regressions.
  * Do not modify code unless explicitly instructed.

* `@reviewer`:

  * Review completed changes.
  * Focus on correctness, race conditions, SQL issues, transaction safety, performance, and regressions.
  * Do not modify files.
  * Report concrete issues with file paths and suggested fixes.

### Parallel Investigation

For complex tasks, investigate independent areas in parallel before implementation.

Example:

```text
Feature: Boss combat system

@explore → inspect existing combat flow
@explore → inspect monster/database structures
@scout → research relevant external requirements
```

Do not delegate dependent tasks in parallel.

### Implementation Workflow

For non-trivial changes:

1. Inspect the existing implementation.
2. Delegate independent investigation to subagents.
3. Consolidate findings.
4. Plan the smallest compatible change.
5. Implement the change.
6. Run relevant tests/checks.
7. Ask `@reviewer` to review the changes.
8. Fix important issues found by the review.
9. Run tests/checks again.
10. Update documentation when required.

### Database Safety

* Never assume a table or column exists.
* Inspect `backend/internal/db/schema_*.go` before writing SQL against project tables.
* Preserve existing constraints and indexes.
* Avoid destructive schema changes unless explicitly requested. When adding new columns, use `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` instead of dropping tables.
* For data migrations, consider existing production rows and duplicate-key constraints.
* Be careful with concurrent updates to `player`, `player_gu`, combat state, and other frequently updated tables.
* Prefer transactions for operations that modify multiple related records.

### Go Concurrency

* Inspect existing goroutines and workers before adding new ones.
* Respect `context.Context` cancellation.
* Avoid unbounded goroutine creation.
* Check for race conditions when modifying shared registries or cached state.
* Run:

```bash
cd backend && go test -race ./...
```

when changing concurrent gameplay logic.

### Combat Changes

Before modifying combat:

* Inspect `backend/internal/handlers/combat.go`.
* Inspect `backend/internal/simulator/engine.go`.
* Inspect relevant player/Gu/effect models.
* Check mana/energy handling.
* Check optimistic frontend updates if the API response affects combat state.
* Preserve deterministic simulation behavior where applicable.

### Data-Driven Gameplay

When adding or modifying Gu, constitution, items, effects, recipes, or other game data:

1. Check the relevant CSV file in `backend/data/`.
2. Check the corresponding database table.
3. Check `docs/GAME_DATA.md`.
4. Check the effect resolution code in `gamefx`.
5. Update all required data/documentation together.

Do not hard-code data that is already represented by CSV/JSONB registries.

### Validation

After backend changes:

```bash
cd backend && go test ./...
```

After frontend changes:

```bash
cd frontend && npm run lint
cd frontend && npm run build
```

For concurrency-sensitive backend changes:

```bash
cd backend && go test -race ./...
```

Before finishing, verify that:

* No unrelated files were modified.
* Existing behavior was preserved unless intentionally changed.
* Relevant tests pass.
* Required documentation was updated.
