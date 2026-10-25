# Architecture

## Runtime overview

```text
Browser / Next.js UI
  -> Zustand domain slices (state + API client)
  -> HTTP `/api/*` or authenticated `/api/ws`
  -> Gin handlers -> shared service (where present) -> PostgreSQL
  -> Redis cache/pub-sub, WebSocket hub and polling/batch workers
```

Gameplay rules remain primarily in handlers, but reusable player, inventory, map, combat-session, recipe, quest, achievement and dao operations now live in `internal/services`. This is an incremental split, not a universal service/repository layer.

The durable shape of a player read lives in `internal/playerstate`: one column projection, one collection-revision expression, one Gu/materials attachment policy and one not-found rule. `internal/services.PlayerService` adds the time projection on top of that snapshot. Player-initiated cell changes go through the movement chain in `internal/handlers/movement.go`, so ownership, alive/combat checks, region policy, the destination write, the regional-wall debuff and post-commit publication do not drift between walking, portals, teleport arrays, Gu teleports, spatial realms and party joins.

## Backend startup

`backend/cmd/server/main.go` performs these steps:

1. Loads the first available env file in `.env` → `backend/.env` → `../.env` → `../../.env`.
2. Connects to PostgreSQL and runs the idempotent schema functions in `internal/db/schema_*.go`.
3. Seeds only runtime prerequisites (sentinel dictionaries, starting hub, system players and backfills). Gameplay dictionaries are imported from Admin; quest is currently the one CSV exception refreshed during schema initialization.
4. Connects to required Redis, configures the rate limiter and clears dictionary cache keys.
5. Loads Gu, item, constitution, race, status, monster, zone, quest and instance registries from PostgreSQL.
6. Wires shared services, starts background workers, the sharded WebSocket hub and Redis subscribers.
7. Starts Gin on `PORT` (default `8080`) and shuts down through `context.Context` on SIGINT/SIGTERM.

## Backend folders

| Path | Responsibility |
| --- | --- |
| `backend/cmd/server` | Process entrypoint and lifecycle. |
| `backend/internal/server` | Middleware setup and canonical route table. |
| `backend/internal/db` | Connection, domain schema, static seeds and Admin CSV import/export helpers. |
| `backend/internal/models` | Domain structs, JSON scanners and runtime registries. |
| `backend/internal/handlers` | HTTP orchestration and domain gameplay rules. |
| `backend/internal/services` | Shared player/inventory/map/combat/recipe/quest/achievement/dao operations and time projection. |
| `backend/internal/playerstate` | Canonical player row projection, snapshot views, collection revision and lazy repairs. |
| `backend/internal/middleware` | Authentication, rate limits and request context. |
| `backend/internal/cache`, `backend/internal/ws` | Redis and realtime WebSocket infrastructure. |
| `backend/internal/worker` | Polling, simulation and batch persistence loops. |
| `backend/data/*.csv` | Canonical import/export files for gameplay data. |

## Frontend folders

| Path | Responsibility |
| --- | --- |
| `frontend/src/app` | App Router pages for game, top-up and dynamic wiki pages. |
| `frontend/src/components` | Gameplay views and dialogs. |
| `frontend/src/store/useGameStore.ts` | Root Zustand composition and shared realtime wiring. |
| `frontend/src/store/slices` | Domain state/actions; slices call APIs and coordinate optimistic updates. |
| `frontend/src/lib` | Shared auth, config and fetch helpers. |

## State and networking

- Protected HTTP calls use cookie credentials. `NEXT_PUBLIC_BACKEND_URL` defaults to `http://localhost:8080`.
- `/api/ws` is the primary realtime transport. The hub supports player events, world position/cell deltas, trade and chat channels via Redis pub/sub when available.
- World-map queries remain HTTP for snapshots, rectangular cell fetches and actions; WebSocket events keep nearby state fresh between snapshots.
- The Zustand root composes domain slices for inventory, combat, exploration, aperture, marketplace, sect, party, notifications, achievements, pets, gacha, betting and other systems.

## Player state and movement

- `playerstate.Load` is the only reader of the full `players` row. Callers choose what to attach (`Gu`, `Materials`, `Repair`) and receive `sql.ErrNoRows` for a missing player; `playerstate.RevisionExpr` is the only collection-revision definition.
- `handlers/movement.go` is the one movement chain. Callers pass a `movementSpec` (kind + destination); the per-kind `movementPolicies` table decides party-leader, region-entry and region-action checks plus their status codes, so a new entrance cannot quietly skip a guard. Every entrance maps a guard failure through the same JSON shape, writes its own extra columns inside its transaction, then publishes through the shared helpers.
- Server-driven placements (respawn, realm returns) use `placePlayerAt` and deliberately skip the guard chain.

## Background workers

Workers are cancellable polling/simulation loops, not an external job queue:

- Gu/pet hunger, growth and status expiry/lazy updates.
- Tribulation, world-index synchronization and Primeval Vein spawn/cleanup.
- Arena, auction, beast-betting and Death Arena settlement/round processing.
- Sect mission, territory and maintenance processing.
- Batch player persistence, pool watchdog and optional orphaned Redis-key cleanup.
- Variant-human production (10 minutes), society simulation (5 minutes) and Gu Master spawning (30 minutes).

## Persistence model

Schema is split by domain under `backend/internal/db/schema_*.go`. Important groups include:

- identity/economy: players, referrals, wallets, top-up orders, transactions, achievements, quest progress, notifications and audit logs;
- dictionaries/world: Gu, item, recipe, race, constitution, status, regions, cells, monsters, spawns, loot and map ground drops;
- combat/social: combat sessions/history, parties/invitations, pets, killer moves, arena/shop, Death Arena, beast betting, trade, marketplace, chat and feedback/comments;
- aperture: apertures, resource nodes, tribulations, variant-human populations, node workers, refinement queue, settlements, events and NPCs;
- sect/activities: sect membership, requests, missions, vault/mines and instances.

Dao marks and several extensible effect/state structures use JSONB. Player row reads go through `internal/playerstate`; full-row scans elsewhere are considered debt. See `docs/GAME_DATA.md` for field and CSV details; the router in `backend/internal/server/server.go` remains the source of truth for endpoints.
