# MMO SLO candidates and capacity gates (PR-36)

These are **provisional Gate C/D targets**, not measured results. The current
repository has synthetic in-process timings (`docs/mmo/benchmark.md`) and admin
zone stats, but no browser, network, PostgreSQL or Redis load evidence. A target
is only "pass" when the evidence column is filled with a dated run and the run
is repeatable.

## Budget basis

- Simulation runs at a fixed 20 Hz tick → **50 ms tick budget**.
- One process owns all zone instances in the current build; per-zone instances
  are goroutines, not dedicated workers (`docs/mmo/scaling.md`).
- Snapshot AOI and combat are server-authoritative; client FPS never changes the
  server tick budget.
- The first supported cohort is small. Capacity beyond that is unproven until
  measured.

## Candidate thresholds

| Signal | Candidate target | Why | Evidence |
| --- | --- | --- | --- |
| Zone tick p99 | ≤ 25 ms at target population with 50 mobs | Keep 50% headroom for GC/network bursts | NOT MEASURED (synthetic p99 1.46 ms at 100 players only) |
| Zone tick p99.9 | ≤ 40 ms; no 5-minute window with sustained > 45 ms | Avoid visible rubber-banding | NOT MEASURED |
| Snapshot/outbound queue | p95 depth < 32 messages; no slow-client drop without resync | Bounded memory/latency | NOT MEASURED |
| Input→ack latency | p95 ≤ 150 ms excluding client RTT; no unbounded ack growth | Movement feels responsive | NOT MEASURED |
| AOI bytes/client | p95 ≤ 24 KB/s at 5 players, ≤ 40 KB/s at 30, at 1280 visibility | Bandwidth budget | NOT MEASURED |
| Client FPS | p95 frame time ≤ 22 ms at 1080p on the reference laptop | 45+ FPS floor | NOT MEASURED |
| Reconnect | ≥ 95% of attempts rejoin within 5 s; 0 duplicate owners | No ghost/duplicate entity | PARTIAL: headless ownership soak only |
| Loot exactly-once | 0 duplicate grants across 100 replays of the same death key | Economy integrity | PARTIAL: transaction/unique-key logic; DB replay NOT MEASURED |
| Checkpoint loss on crash | ≤ 10 s of movement/HP plus in-flight state | Bounded rollback | NOT MEASURED (no crash injection) |
| Server error rate | < 0.1% unexpected 5xx/WS closes per 1k requests | Stability | NOT MEASURED |
| Restart time | < 60 s to accepting joins with warm caches | On-call recovery | NOT MEASURED |
| Flag-OFF legacy | 0 legacy regressions; MMO bundle not loaded | Rollback safety | PARTIAL: unit/build evidence; browser E2E NOT MEASURED |

## Measurement protocol

1. Record commit, build hash, map/profile hashes, machine/container limits, DB
   and Redis versions, client browser/GPU, and network profile (RTT/jitter/loss).
2. Run the automated subset:
   ```bash
   scripts/mmo-release-gate.sh
   ```
3. Run a synthetic baseline for regression only:
   ```bash
   cd backend
   go run ./cmd/mmo_bench -players 5  -mobs 50 -ticks 1000
   go run ./cmd/mmo_bench -players 30 -mobs 50 -ticks 1000
   go run ./cmd/mmo_bench -players 100 -mobs 50 -ticks 1000
   ```
4. Run real clients (5 → 30 → 100) against staging with F3 enabled and admin
   `/api/admin/monitor/realtime` scraped by the metrics system. Capture p50/p95/
   p99, queue depth, drop/resync counts and AOI bytes per client.
5. Run the DB/Redis failure matrix: disconnect, restart, slow query, lock
   timeout, crash between loot claim and commit, reconnect under combat.
6. Attach raw output and a short interpretation to
   `docs/mmo/release-checklist.md`. Do not replace missing evidence with
   "looks fine".

## Capacity decision

| Result | Action |
| --- | --- |
| Tick p99 ≤ 25 ms and no queue/drop breach at 30 players | Keep `MMO_INSTANCES_PER_ZONE=1–2`, raise allowlist gradually |
| Tick p99 25–40 ms or queue depth breaches | Reduce per-zone players, lower visibility radius, fix allocation hot spots before raising |
| Tick p99 > 40 ms or slow-client drops without resync | Stop raising capacity; optimize or split instances/workers |
| 100 players/zone fails | Do not advertise 100/zone. Use more instances/processes only after cross-process leases/routing land |
| Any duplicate loot/ownership | Roll back admission immediately; treat as a release blocker |

**Hard rule:** the current process-local instance split is not a substitute for
cross-process zone ownership. Do not claim multi-worker or 100-player support
until `docs/mmo/scaling.md` prerequisites and the load matrix are measured.
