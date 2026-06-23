# AGENTS.md – GuWorld

## Startup

```bash
cd backend && go run ./cmd/server        # backend on :8080
docker-compose up                        # PostgreSQL on :5432
cd frontend && npm install && npm run dev  # frontend on :3000
```

**.env search order:** `.env` → `backend/.env` → `../.env` → `../../.env`
**DB defaults:** localhost:5432, user `gu_admin`, pass `gu_pass_123`, db `gu_db`
**CSV seed paths** are relative to `backend/internal/db/db.go` (reads `../../*.csv`). If seed fails silently, check `backend/*.csv` exists.

## Tests & Checks

```bash
cd backend && go test ./...       # all backend tests
cd frontend && npm run lint       # ESLint 9
cd frontend && npm run build      # Next.js build
```

No CI/CD. No typecheck script. Test files are `*_test.go` in `backend/internal/handlers/`.

## Backend Conventions (agent would likely miss)

- **No service layer.** Handlers are fat (combat.go: 2400+ lines, player.go: 95KB). All gameplay logic is in `backend/internal/handlers/`.
- **No migrations.** Schema is `CREATE TABLE IF NOT EXISTS` in `backend/internal/db/db.go`. Modify SQL there and re-run.
- **CSV seed is append-only.** New rows inserted once on first run. Updates require CSV edit + restart.
- **Registries cached in memory** on startup (`models.GuRegistry`, `models.ItemRegistry`). Changing CSV requires app restart.
- **Workers are goroutine polling loops**, not job queues. Hunger, tribulation, mine monsters. All workers accept `context.Context` for graceful shutdown.
- **Redis is optional.** Failure logs warning, server continues.
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
- `backend/internal/db/db.go` — schema SQL + CSV seed logic
- `backend/internal/handlers/*.go` — all feature logic (no service layer)
- `backend/internal/simulator/engine.go` — deterministic auto-battle loop, combat log format
- `frontend/src/store/useGameStore.ts` — Zustand + API client + SSE
- `frontend/src/app/` — Next.js App Router pages
