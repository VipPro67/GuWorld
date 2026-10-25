# MMO migration baseline audit

Snapshot: 2026-10-25 (Asia/Ho_Chi_Minh)  
Branch: `mmo/realtime-foundation`  
HEAD at capture: `d6444cb0`  
Worktree: dirty; this report describes the checked-out working tree and does not
attribute its changes to a commit.

## Scope and reproducible checks

This is a code and local-check baseline for the Phaser realtime migration. It
does not claim a production baseline. The source archive used for the original
plan was not present in the repository, so no ZIP checksum can be recorded.

| Artifact/check | Result |
| --- | --- |
| `frontend/package-lock.json` SHA-256 | `275dca4295dde1f643eca83126d17125da2a3eb3c36620ce1e017f641ba2df95` |
| `backend/go.sum` SHA-256 | `ba97ed34f5bf8e3bee58ba37ac670d9e6c83b6c84e9532f16ea5d553c47e4544` |
| Go toolchain in this environment | `go1.27.1`; repository `go.mod` requests Go 1.26.3, so this is not the pinned baseline toolchain |
| `go test ./...` | PASS (2026-10-25; see terminal output from baseline run) |
| `npm run typecheck` | PASS |
| `npm test` | PASS, 33 tests |
| `npm run lint` | PASS |
| `npm run build` using default Turbopack | FAIL in this sandbox: Turbopack could not spawn/bind its CSS worker (`Operation not permitted`, OS error 1) |
| `next build --webpack` retry | FAIL: Next could not parse TypeScript `--showConfig`; standalone `tsc --showConfig --project tsconfig.json --pretty false` exits 0 and emits JSON, so this remains an unresolved build-tool integration issue |
| Local app browser inspection | PASS (limited) | Authenticated page reached the legacy world at `Thanh Mao Sơn`; no MMO Beta entry was rendered. `GET http://localhost:8080/api/admin/monitor/realtime` returned `{"error":"realtime runtime is not configured"}` |
| Backend/frontend startup from shell | NOT MEASURED | The browser-hosted app is reachable, but the execution sandbox cannot inspect or start its host processes |
| Browser E2E / FPS / VRAM / latency | NOT MEASURED | Only the current legacy screen was inspected; no movement/combat/reconnect or multi-client scenario was exercised |
| PostgreSQL/media backup and restore | NOT MEASURED; no restore fixture or staging evidence was available |

The working tree includes broad pre-existing modifications and untracked MMO
work. Do not use this snapshot as a clean-commit diff or overwrite those files
while continuing the migration.

## Follow-up readiness check (2026-10-25)

- The live frontend is on the legacy map; the gated `MMO Beta` control is not
  present. `featureFlags.ts` requires both public flags to equal `true`, and
  the inspected local environment files define neither flag.
- The backend monitor explicitly reports that the realtime runtime is not
  configured. Even if the frontend flags were changed, live joining separately
  requires the backend runtime flags, a zone mapped from the character's
  current region, an allowlisted player UUID, and exact map/content versions.
- The visible character is in `Thương Gia Thành`; its `hub_thuong_gia_thanh`
  row in `backend/data/Region.csv` has no `world_map_id`, so the importer uses
  legacy `world_map`. The authored realtime maps use separate IDs. A targeted
  dev mapping is needed before this character can pass the server's
  region-to-zone equality check; mapping the shared hub globally would also
  affect other characters there. The schema now seeds the five MMO map IDs in
  `world_maps`, but the live database still needs a backend restart before
  Admin can assign a dedicated test region.
- `MMO_RELEASE_SKIP_FRONTEND=1 MMO_RELEASE_SOAK_SECONDS=15
  scripts/mmo-release-gate.sh`: backend formatting, vet, all Go tests, realtime/
  WS race tests, Gu catalog parity, map parity, five-player headless slice and
  15-second reconnect/ownership soak passed. Frontend was intentionally skipped
  by this invocation; frontend lint/typecheck/33 tests passed in the separate
  baseline run. Production build remains failed as detailed above.
- This is enough to confirm why the UI stays legacy. It is not enough to admit
  the current character safely: the region-to-zone mapping and account
  allowlist have not been verified, and production ownership/mutation gates are
  still open.

## Entrypoints and ownership boundaries

| Area | Current path | Migration role / current limitation |
| --- | --- | --- |
| Frontend entry | `frontend/src/app/page.tsx` | Existing Next.js home/view router. It exposes `mmo-world` only through `canShowPhaserWorld`; the legacy `world-map` still renders `WorldPage`. |
| Phaser host | `frontend/src/components/game/PhaserWorldHost.tsx` | Client-only dynamic host owns Phaser and `WorldRuntime`; joins only when client zone/map/content config is present. |
| Phaser runtime | `frontend/src/game/{bootstrap,scenes,network,input,world,combat}` | Realtime client implementation. Existing art and some combat profiles remain preview-grade. |
| Frontend world socket | `frontend/src/game/network/WorldSocket.ts` | Dedicated v1 realtime protocol transport; legacy store/socket remains for business/social features. |
| Backend process | `backend/cmd/server/main.go` | Starts legacy services plus opt-in realtime manager/admission. Realtime flags default off. |
| HTTP/WS router | `backend/internal/server/server.go`, `backend/internal/handlers/world_ws.go` | Registers API routes and authenticated WS world-message routing. |
| Realtime admission | `backend/internal/realtime/{config,gateway,world}` | Zone manager and gateway are process-local; no durable cross-process owner fencing. |
| Legacy world | `backend/internal/handlers/{exploration_*,world_*,combat.go}`, `frontend/src/components/WorldCanvas.tsx` | Remains the rollback and default path. Mutation audit is not complete; do not cut over production. |
| Canonical business state | `backend/internal/services/`, `frontend/src/store/` | Existing character, inventory and progression services/store remain authoritative for business UI; projection/refresh into a live zone is incomplete. |

See [legacy-flow.md](legacy-flow.md) for the request and rollback paths.

## Security and data notes

- `backend/.env` exists locally and is ignored by Git; its contents were not
  opened or copied into this report. Keep real credentials out of tracked
  files and logs.
- No database/media backup or restore was performed for this baseline.
- Local server startup and database state were not available to validate
  schema version, media storage or a restored deployment.

## Exit criteria still open

PR-01 remains partial until the pinned Go 1.26.3 run, frontend lint/build,
baseline archive identity (if the source archive can be recovered), DB/media
backup restore, and schema record are attached. Runtime/browser performance is
also still unmeasured.
