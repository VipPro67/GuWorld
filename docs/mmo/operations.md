# Operations, CI and content versioning (PR-33)

## CI gates (`.github/workflows/ci.yml`)

Backend job:
- `gofmt` check on the realtime surface;
- `go vet ./...`;
- `go test ./...`;
- `go test -race ./internal/realtime/... ./internal/ws/...`;
- `gudata audit-realtime -strict` (unclassified/MVP failures block CI);
- `gudata compile-realtime-gu` over the combined MVP+Alpha catalogs (invalid
  profile/payload blocks CI).

Frontend job (Node 22, npm lockfile):
- `npm ci`;
- `npm run lint`;
- `npm run typecheck`;
- `npm test`;
- `npm run build` (Phaser pinned `4.2.1` in lockfile).

The map parity test (`backend/internal/realtime/nav`) is part of `go test ./...`
and fails when the server/client map copies drift.

## Release gate (PR-36)

```bash
scripts/mmo-release-gate.sh
```

Runs the automated Gate D subset (format/vet/tests/race, Gu coverage and compile
parity, map parity, 5-player slice, reconnect soak smoke, frontend
lint/typecheck/test/build). Skip frontend or race with
`MMO_RELEASE_SKIP_FRONTEND=1` / `MMO_RELEASE_SKIP_RACE=1`; set
`MMO_RELEASE_SOAK_SECONDS` for a longer smoke. It prints an explicit
`NOT MEASURED` list: browser E2E, real-network load, DB crash/dupe replay,
staging canary and on-call drill still require separate evidence.

Gate D evidence/sign-off lives in `docs/mmo/release-checklist.md`; candidate SLO
thresholds and capacity decisions are in `docs/mmo/slo.md`. Do not mark Gate D
passed from the automated script alone.

The same gate can be run manually from GitHub Actions via the
**MMO release gate** workflow (`.github/workflows/release-gate.yml`,
`workflow_dispatch`, optional soak seconds / frontend skip). Attach its log to
the release ticket; it still excludes browser, network-load, DB crash/dupe,
staging-canary and on-call evidence.

## Content versioning

- Client sends `map_version` + `content_version` in `world.join`; backend
  compares both against `MMO_MAP_VERSION`/`MMO_CONTENT_VERSION` and rejects
  stale versions.
- `world.welcome.map_hash` is the SHA-256 of the server collision artifact; the
  client recomputes the same hash from its downloaded JSON and shows `LỆCH HASH`.
- The compiled Gu catalog hash is logged at startup and pinned by the compiler
  test. Changing a profile requires regenerating `docs/mmo/gu-compiled.json` and
  updating the baseline intentionally.
- Gu coverage CSV has a golden SHA in the audit test; drift forces a conscious
  regenerate.

## Observability

- `GET /api/admin/monitor/realtime` (permission `monitoring.read`) returns
  per-zone `{tick, entities, players, subscribers, last_step_ms}` from
  `Manager.Stats`.
- Existing admin monitor endpoints cover system/request traces and WebSocket
  stats.
- Client **F3** debug HUD shows tick/FPS/network counters/prediction correction.
- Startup logs record map hash, monster count and compiled Gu profile hash.

Still missing:
- Prometheus/OpenTelemetry metrics and dashboard provisioning by zone/player/
  network/economy.
- Per-client AOI bytes/sec, slow-client drops/resync, RTT/jitter histograms.
- Loot grant dedupe counters and transfer-failure counters.
- Structured redaction audit for logs (no tokens/passwords).

## Staging smoke

1. `docker-compose` staging with PostgreSQL/Redis; run backups before imports.
2. Import map/Gu data through Admin; restart if a registry is cached.
3. Enable flags with an explicit allowlist (see `docs/mmo/cutover.md`).
4. `go run ./cmd/mmo_slice -players 5 -mobs 10 -ticks 400`.
5. Browser checklist in `docs/mmo/vertical-slice.md` on 2–5 accounts.
6. Check `/api/admin/monitor/realtime`, client F3, loot grants and reconnect.
7. Record results in `docs/mmo/benchmark.md`; never claim load targets without
   measurement.

## Incident/rollback playbook

- Duplicate grant / dual ownership: rollback immediately
  (`MMO_ADMISSION_PAUSED=true`, drain, keep ledger rows).
- Tick p99 > 50ms sustained, severe backpressure or FPS collapse: pause new
  admits, drain, restore legacy route, investigate hotspots.
- Map/collider error: disable new MMO admits; do not delete durable state.
- DB/Redis outage: realtime runtime keeps running with bounded retries/fallback,
  but stop new admits if checkpoint/loot cannot be trusted.
- See `docs/mmo/cutover.md` for the full staged rollback.
