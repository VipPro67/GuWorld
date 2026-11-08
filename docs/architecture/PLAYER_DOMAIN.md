# Player Domain Guide

Use this guide when changing a player route, durable player state, player-visible gameplay, or the frontend player snapshot. It is an index and workflow, not a second API or schema specification. The route table in `backend/internal/server/server.go`, the owning code, and `docs/guides/API.md` remain authoritative.

## Ownership Map

| Concern | Start here | Boundary / notes |
| --- | --- | --- |
| Route and HTTP contract | `backend/internal/server/server.go`, `docs/guides/API.md` | The router is the source of truth for registration; follow the registered handler before editing. |
| Registration and onboarding | `backend/internal/handlers/player_auth_register.go` | Also inspect schema foreign keys and region/hub data when changing creation defaults. |
| Login, logout, current player, cookies, Google auth | `backend/internal/handlers/player_auth_login.go`, `player_auth_cookies.go`, `player_auth_google.go` | Session and current-player projection behavior is auth-sensitive. |
| Durable player snapshot | `backend/internal/playerstate/snapshot.go` | Owns full-row columns and scan order, collection revision, Dao-mark hydration, materials, expired-status repair, and not-found behavior. |
| Time/status projection | `backend/internal/services/player_service.go` | `PlayerService` builds gameplay projections over `playerstate`; it is not the owner of the durable row shape. |
| Profile and disguise | `backend/internal/handlers/player_profile.go` | Player-facing profile mutations and associated validation. |
| Stats, status, recharge | `backend/internal/handlers/player_stats.go`, `player_status.go`, `player_recharge.go` | Check projection and resource update behavior together when changing returned player state. |
| Player mutation pipeline | `backend/internal/handlers/player_mutation.go` | Canonical transaction & persistence gateway (`MutatePlayer`) for all player-state mutations. Owns locking, passive evaluation, canonical row update with monotonic `version++` & `state_revision++`, and post-commit snapshot publication. |
| Gu and item actions | `backend/internal/handlers/player_equip.go`, `player_use.go`, `player_feed.go`, `player_gu_use.go` | Use `InventoryService` for shared material inventory operations where applicable; Gu persistence also has its own tables/helpers. |
| Dao marks | `backend/internal/handlers/player_daongan.go` | Verify related aperture persistence and revision behavior when changing durable marks. |
| Player movement | `backend/internal/handlers/movement.go` | Player-initiated cell changes use the shared `ExecuteSpatialTransition` flow (covering guard validation, atomic coordinates/revision/v2-state persistence, party sync, and post-commit publication). Server-driven placement is intentionally separate. |
| Player realtime events | `backend/internal/handlers/player_events.go` | Preserve post-commit publication and event/revision contracts. |
| Frontend player state | `frontend/src/store/useGameStore.ts`, then the owning file under `frontend/src/store/slices/` | Trace the API action and response application; preserve optimistic rollback and central auth handling. |

Shared handler services are initialized in `backend/internal/handlers/services.go`: `PlayerSvc`, `InventorySvc`, `RecipeSvc`, `QuestSvc`, and `AchievementSvc`. The service package is selective; keep domain rules in their existing owner and do not assume there is a service for every gameplay area.

## Change Workflow

1. Find the route in `server.go`, read its contract in `docs/guides/API.md`, and follow the registered handler. For non-route gameplay, start from the owning handler or caller.
2. Classify the data being changed: canonical `players` row, a player collection/table, a derived response projection, or transient/realtime state. Inspect the matching schema and existing writer before changing persistence.
3. For a full player snapshot, use `playerstate.Load`/`Scan` or the appropriate `PlayerService` loader. `Options.Gu` selects Gu scope only; materials, Dao-mark hydration, and expired-status repair are included by `playerstate.Load`. A narrow projection is appropriate when the caller needs only specific fields; do not duplicate the full canonical projection or collection revision.
4. For player state mutations, use `MutatePlayer` in `backend/internal/handlers/player_mutation.go`. It executes the caller's mutation closure inside an atomic database transaction with `FOR UPDATE` locking, applies passive evaluations, executes a canonical full-row update that monotonically increments `version` and `state_revision`, commits atomically, and publishes post-commit player snapshot events (`PublishPlayerSnapshot`) to Redis and connected clients. Do not write hand-rolled `UPDATE players` statements with partial column subsets that omit `version` or `state_revision`.
5. For a player-initiated move, route through `ExecuteSpatialTransition` in `movement.go`; do not create a parallel endpoint-specific guard sequence or ad-hoc coordinate update. Keep server-directed placement on its separate path when appropriate.
6. If the response changes frontend player state, use `executeGameAction` from `frontend/src/store/actionGateway.ts`. It provides a unified gateway with concurrency locking, token validation, automatic response reconciliation via `applyInventoryFromResponse`/`setPlayerFromServer`, activity logging, and user toast notifications.
7. Add or update a test beside the owning package. Run `cd backend && go test ./...` for backend behavior; for frontend changes run `cd frontend && npm run lint` and `npm run build`. For concurrency-sensitive backend changes, also run `cd backend && go test -race ./...`.

## Persistence Guardrails

- The canonical full-row projection lives in `playerstate`; use its scan order and revision definition rather than maintaining a second full-row read.
- `playerstate.Load` may repair expired statuses and therefore is not a pure, side-effect-free read. Choose it with that behavior in mind; use an existing narrow projection when only response fields are needed.
- Materials are loaded from `player_inventory_items`; Gu inventory is loaded only for the selected Gu scope. A missing player is reported as `sql.ErrNoRows` and can be checked with `playerstate.IsNotFound`.
- Schema definitions and compatibility DDL live in `backend/internal/db/schema_*.go`. The project has no versioned migration framework; inspect whether a change also needs an explicit operational data migration.
- For player-visible gameplay values backed by CSV/JSON registries, inspect the source data and `docs/guides/GAME_DATA.md`; update `WIKI.md` when player-facing rules change.