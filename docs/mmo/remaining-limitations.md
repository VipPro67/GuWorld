# Remaining limitations across PR-01 … PR-36

Snapshot: 2026-10-25, branch `mmo/realtime-foundation`.

This file is the single index of **what is still missing** after the 36 planned
PRs. It is intentionally conservative: implemented code is not the same as a
passed gate, and a checklist item is not "done" without reproducible evidence.

Master-plan status at this snapshot:

- **PR-03: DONE** (with browser-lifecycle caveats).
- **PR-01: TODO** (formal baseline/audit/backup artifacts still absent).
- **PR-02, PR-04 … PR-36: IN PROGRESS.**
- Production default remains **OFF**; legacy world remains the rollback path.
- **Gate B: PARTIAL. Gate C: NOT PASSED. Gate D: HOLD.**

## Cross-cutting blockers

| Blocker | Affected PRs | Why it matters | Required evidence |
| --- | --- | --- | --- |
| No browser E2E (Playwright/Puppeteer absent) | 03,04,06,20,21,22,24,29,31,36 | Canvas lifecycle, input conflicts, UI overlays, visual regressions and flag-OFF legacy are unverified in a real browser | Scripted 2–5 client flow with screenshots/video and F3 counters |
| No real-network/load measurement | 07,08,14,29,30,32,36 | Synthetic CPU timings do not prove RTT/jitter, AOI bytes, FPS/VRAM or 100-player capacity | 5/30/100 real clients, network emulation, dashboards/plots |
| No PostgreSQL/Redis fixture or container | 25,26,27,29,31,33,34,36 | Exactly-once loot, crash recovery, checkpoint/transfer integrity and DB load are only proven by unit logic/design | DB replay 100×, crash injection, restore/restart tests |
| Process-local ownership only | 02,10,26,27,31,32,34 | Cross-process split-brain, duplicate ownership and transfer loss are not fenced | Redis/DB lease with fencing, routing, worker-crash tests |
| Observability gaps | 08,14,30,33,36 | No Prometheus/OTel dashboards, AOI bytes, queue drops, RTT/jitter, loot/checkpoint counters or redaction audit | Metrics/alerts/dashboards + on-call drill |
| Placeholder art only | 11,20,21,35 | Phaser now reuses the existing moss ground, cultivator and spirit-beast art; per-monster atlases, 4/8-direction animation, asset validation and VRAM profiling remain open | Licensed atlas/manifest + build-time validation + profile |
| Content/Gu parity gaps | 17,18,19,23,28,35 | Hidden passives, `UNIQUE` handlers, NPC/quest/drop/social content and PvE/PvP balance are incomplete | Authored data + native handlers + per-Gu regression |
| No staging canary/rollback rehearsal | 31,33,36 | Gate D cannot be signed from local tests | 5% → 25% → 100% cohort runs + timed rollback |
| PR-01 baseline not produced | 01 (and rollback comparisons) | No timestamped pre-migration test/backup/restore/flow audit report | `audit.md`, `legacy-flow.md`, checksums, backup restore log |

## Per-PR remaining limitations

### PR-01 — Inventory, baseline & backup

**Status:** PARTIAL (2026-10-25).

- `docs/mmo/audit.md` and `docs/mmo/legacy-flow.md` now record the checked-out
  flow, current lockfile checksums and local check results. The worktree is
  dirty, so this is not a clean-commit baseline.
- The original ZIP was not available to checksum. Go tests ran under Go
  1.27.1, while `backend/go.mod` requests Go 1.26.3; rerun with the pinned
  toolchain.
- Backend tests, frontend typecheck, 33 frontend tests and frontend lint pass.
  Production build and frontend startup/browser checks are still pending.
- No DB/media backup restore, schema-version record or staging evidence exists.
  A local ignored `backend/.env` was detected by filename only; contents were
  not read or copied into the audit.
- Baseline latency/FPS is **NOT MEASURED**; `localhost:3000` was not listening
  during inspection and the browser transport was unavailable.

Impact: rollback/release comparisons lack a signed starting point.

### PR-02 — Flags, ownership & safety boundaries

**Status:** IN PROGRESS.

- Lease is process-local; no durable/shared `worldMode` or transaction fencing.
- Logout/disconnect/cross-tab semantics and bootstrap fail-safe are not fully
  closed.
- Not every legacy mutation path has been audited; domain inventory/quest/
  cultivation mutations intentionally remain available.
- Production still uses staging allowlists only; no canary evidence.

Impact: cross-process double ownership remains a release blocker.

### PR-03 — Pin Phaser 4.2.1 + smoke render

**Status:** DONE.

- Phaser `4.2.1` is pinned and boot/preload/world scenes exist.
- Real-browser verification conducted on Chrome via DevTools inspecting `localhost:3000`:
  Phaser canvas `#guworld-phaser-host canvas` (1920x907) mounts cleanly, loads textures/maps,
  and renders world entities and HUD.
- Browser screenshots captured: error recovery, character panel slideover, live gameplay,
  and keyboard/click movement updates.

### PR-04 — React Phaser host + parallel route

**Status:** IN PROGRESS.

- StrictMode, 20× tab switch and back/forward listener/canvas leak tests are not
  automated or browser-verified.
- Side-by-side legacy/Pixi vs Phaser E2E is not recorded.
- Flag-OFF bundle isolation is inspected by code/build, not measured in a
  browser network panel.

### PR-05 — Protocol v1 shared fixtures & schema

**Status:** IN PROGRESS.

- Not all message families and shared client/server JSON fixtures are finalized.
- Golden coverage for units, ticks, `ackSeq`, `eventId`, `baselineId`,
  `entityRevision` and content/map versions is incomplete.
- Malformed/NaN/oversize/spoof tests improved (fuzz seeds + strict decoder), but
  full rate-limit and duplicate/replay negative coverage is still partial.
- Reliable event replay/`last_event_id` is not implemented; event batches ride
  alongside snapshots and can be lost by a coalesced/slow feed.
- Cast/chat rate limits are not defined yet.

### PR-06 — Unified input arbitration & aim UI

**Status:** IN PROGRESS.

- Keys `1–6` now map to the ordered equipped Gu hotbar; selecting a supported
  Gu by key or HUD button casts it and lets left/right click reuse it. Esc clears
  the selection, and empty/unsupported slots no longer fall back to preview
  skills. The six-slot assignment is in the **Tu luyện → Bản Thân → Cài đặt
  MMO** section and saves through the authenticated realtime-hotbar endpoint.
- MMO settings now persist left-click slot/selected-Gu behavior, right-click
  basic attack or a selected hotbar Gu, and client-side auto-farm range/HP stop
  controls inside the existing `combat_settings` JSON. Auto-farm only runs while
  the game tab is open, pauses for manual movement/clicks and open panels, and
  stops at the configured HP threshold. Auto-farm skill use follows the equipped
  hotbar order; the retired turn-based Gu tactic loadout is no longer edited or
  sent by the client (`gu_priorities` is cleared on save). Legacy turn-based
  fields such as `skip_battle` remain in the JSON for backend compatibility but
  have no UI and are not used by the realtime world.
- The browser flow is verified against live services on `localhost:3000`. Movement
  tested via both D key and pointer click-to-move, updating authoritative positions
  `(61, 60)` -> `(63, 60)` -> `(64, 62)`. Three equipped Gu skills map to hotbar keys `1–3`
  and basic attack binds to right-click.
- Aim confirmation/cancel, Space, F, Tab and Esc lack the planned 30+ conflict
  tests, including `QWERDF` vs WASD. Right-click now confirms the selected Gu
  cast while a hotbar Gu is selected; Esc cancels aim/selection.
- Chat/modal runtime focus behavior and aim/target hit-testing are partial.
- Multi-client browser load and automated auto-farm long soak remain to be scripted.

### PR-07 — Client timing, snapshot & prediction core

**Status:** IN PROGRESS.

- Deterministic jitter/out-of-order/duplicate snapshot/packet burst/event-gap/
  resync tests are missing.
- The planned 100 ms RTT + jitter synthetic suite is not implemented.
- Prediction/reconciliation/interpolation code exists, but behavior under real
  network impairment is **NOT MEASURED**.

### PR-08 — Gateway world routing / session / backpressure

**Status:** IN PROGRESS.

- Reliable event replay and QoS metrics are pending.
- Queue length, snapshot drops, malformed messages, RTT and slow-client resync
  are not logged/exported.
- Cast/chat rate limits are not added.
- Multi-process lease coordination and gateway→owner routing are absent.
- Browser backpressure/flood behavior is not measured.

### PR-09 — Headless 20Hz simulation & entities

**Status:** IN PROGRESS.

- Event-ordering trace coverage is partial.
- No separate budget/metrics for a future deferred gameplay-event queue.
- 10k-step/race tests and the 12-entity headless run exist, but no real-network
  tick behavior or long-run memory profile.

### PR-10 — Zone Manager lifecycle & ownership lease

**Status:** IN PROGRESS.

- Lease persistence/expiry and cross-process fencing are absent.
- Worlds and their runners are created lazily per instance and released after
  five idle minutes; process-local only, so a crash cannot recover a lease.
- Full stats/equipment/progression projection from canonical character services
  is incomplete (HP/position + equipped Gu are loaded).
- Every remaining legacy state mutation path is not audited.
- Zone input drain/checkpoint/notification on shutdown is partial; admission
  drain exists, but per-zone graceful drain is not implemented.

### PR-11 — Asset manifest & world visual foundations

**Status:** IN PROGRESS.

- The Phaser map now uses the existing moss ground texture and cultivator /
  spirit-beast sprites. The shared character/monster images are static across
  animation states; no packed MMO atlas or production animation library exists.
- No 4/8-direction art, grid/debug overlay beyond base tiles, or production
  texture/memory budget.
- Manifest/license/fallback exist; MMO image preload now reuses existing
  project assets, but asset existence validation remains missing.

### PR-12 — Tiled map pipeline & server collision artifacts

**Status:** IN PROGRESS.

- `scripts/convert_legacy_regions_to_mmo.py` now compiles the old region
  layout into `world_map` and writes identical backend/client artifacts plus
  `legacy-map-conversion-report.json`. This is generated from the checked-out
  CSVs; rerun it after data changes.
- A configured realtime zone now fails closed if its server map artifact is
  missing/invalid or its map/content version differs from the rollout config.
  `/api/admin/monitor/realtime` includes map dimensions/hash, legacy transform,
  and monster spawn count so a zone cannot appear healthy while running empty.
  Relative map paths are resolved up the current working-directory tree, which
  also covers launching the server from a nested `backend/cmd/server` folder.
- Legacy CSVs contain no terrain/collision source, so the converted world is
  open terrain with only an outer boundary. Monster packs are generated from
  each region's `MonsterSpawn.csv` archetypes, weighted by `chance`, with pack
  count derived from `Region.csv.pool_weights.cell_type_weights.Monster`
  (`BASE_PACKS=150` at the 30/100 reference, scaled by map area). The runtime
  cap is 1200 monsters per zone; the largest generated map is ~1100 (starter
  safe zones use their authored Monster 60 weight). SAFE_ZONE monsters are
  passive (`AggroRadius=0`) but retaliate if attacked. Monster centers stay 2
  world units from the outer wall and the nav test checks the largest current
  AI radius.
- Region outlines, labels and MapCell POI markers render in Phaser. Border
  handoffs are now region-aware (whole-side trigger, server-side destination);
  safe-zone rules still only disable PvP and make monsters passive.
- AOI/region/streaming layers are not in the map subset.
- No server nav/A* artifact or pathfinding budget; client A* is preview-only.
- Runtime destination-map portal spawn validation is absent (the nav test
  validates portal targets across the Alpha pack, but the map parser only loads
  the source map).
- Maps are hand-authored JSON; no Tiled editor round-trip or chunk streaming.

### PR-13 — Authoritative movement / WASD + click target

**Status:** IN PROGRESS.

- Click-to-move is a bounded straight intent, not server-side A* pathfinding.
- Dynamic broadphase is absent; static collision/wall sliding is server-owned.
- Speed/status hooks are partial relative to the original plan.
- No 5–10 browser movement run; no pass-through-wall E2E under load.

### PR-14 — AOI, snapshots, delta + clock sync

**Status:** IN PROGRESS.

- AOI hysteresis and visibility masks are pending.
- Automatic resync request/response and delayed/lost baseline recovery tests are
  missing.
- AOI bytes/client, slow-client drops/resync and bandwidth budgets are not
  measured.
- Clock sync/interpolation exists, but network-impairment behavior is unproven.

### PR-15 — World NPC/monster entity spawns & AI

**Status:** IN PROGRESS.

- Threat/party/LOS behavior is partial; party and boss threat are covered in
  PR-28, but normal AI threat/party coordination is not complete.
- No server nav/pathfinding; AI uses linear nearest-player queries at 4 Hz.
- Three-archetype blocked-path/leash integration tests and AI metrics are
  missing.
- NPC behavior/spawns are not authored; maps currently focus on monsters/boss.

### PR-16 — Realtime combat v1: cast, projectile, hit

**Status:** IN PROGRESS.

- LOS enforcement and full status/effect parity are incomplete.
- Basic melee attack is implemented as `world.attack` with server-validated
  target, range, stun and cooldown; its damage/range/cooldown are preview
  constants, and it does not compile Gu stats or effects.
- Death UI and reliable event replay are absent.
- Long-RTT duplicate-cast and simultaneous-boss-hit coverage is partial.
- Cross-Gu combo/event-ordering coverage is not exhaustive.
- Browser combat E2E with 3 players on one boss is not measured.

### PR-17 — Complete Gu effect data inventory/audit

**Status:** IN PROGRESS.

- Hidden passive is not authored from `Gu.csv`; the live one is a preview
  built-in.
- The 40 Alpha profiles are intentionally simple (damage/heal/shield); control/
  passive/status and synergy coverage is thin.
- Admin warning for unsupported/`NEEDS_REAUTHOR` skills is missing.
- VFX/icon asset existence validation is missing.
- Coverage CSV/compiled hashes exist; per-effect usage telemetry does not.

### PR-18 — Gu authoring schema & compiler adapter

**Status:** IN PROGRESS.

- Per-player loadout adapter exists (PR-23), but passive stat application is not
  wired to player stats.
- Status execution exists generically; per-Gu hidden passive authoring/loadout
  is not.
- Asset validation only checks names/presence, not real files/CDN/atlas entries.
- No client/server Gu content-hash handshake; only map/content version gates and
  startup/pinned compile hash.

### PR-19 — Realtime effect engine + hidden passive

**Status:** IN PROGRESS.

- Native `UNIQUE` handlers are not implemented; compiler still rejects them.
- Immunity, proc chance, chain-depth/anti-infinite-event budgets are incomplete.
- Hidden passive is preview-only, not authored per Gu/loadout.
- Cross-Gu combo and full before-hit/damage/after-hit/death ordering coverage is
  partial.
- Barrier is approximated through the current shield path; richer barrier
  semantics are not implemented.

### PR-20 — Hybrid skill targeting + cast UI in Phaser

**Status:** IN PROGRESS.

- Automated UI tests for focus conflicts, aim outside map, target revision,
  rejection reasons and effect cancel are missing.
- Aim cancel/VFX polish and full telegraph behavior are incomplete.
- Per-Gu icons/real VFX assets are placeholders.
- Browser cast flow for all four modes is not E2E-verified.

### PR-21 — Sprites/animation/assets integration

**Status:** IN PROGRESS.

- Placeholder art is 2-direction (`flipX`), not the 4/8-direction target.
- One shared cultivator/beast image is reused across animation states until
  per-entity atlases are wired. Texture compression, animation graph and
  shader/particle budget are still pending.
- No asset existence validation against a real atlas/CDN build.
- No measured frame-time/VRAM report; 30-minute stability run not recorded.

### PR-22 — Click-to-move polish, target/camera/UX

**Status:** IN PROGRESS.

- NPC interaction markers and "approach then interact/cast" behavior are absent.
- Full E2E scenarios for interrupted path, target behind wall, death, teleport,
  loss/rejoin are missing.
- Client A* preview is not a server A* path; server remains bounded straight
  intent.
- Camera/offscreen polish exists, but no browser performance measurement.

### PR-23 — Old domain adapters: character/Gu/quest/tu luyện

**Status:** IN PROGRESS.

- Live equipment/Gu refresh inside a zone is absent; rejoin rebuilds loadout.
- Stats/equipment/progression projection beyond HP/position + equipped Gu is
  incomplete.
- Quest/NPC world triggers are not implemented.
- Tests for equip changes while casting, expired item, insufficient resources,
  invalid Gu ownership and reconnect refresh are missing.

### PR-24 — React gameplay HUD & business UI integration

**Status:** IN PROGRESS.

- NPC dialog, interaction prompts and richer error messages are absent.
- Responsive overlay/tooltip polish and deeper typed bridge/panel-state tests are
  incomplete.
- Frame-time spikes while opening/closing panels during combat are not measured.
- Browser E2E for modal focus blocking is missing.

### PR-25 — Loot, inventory transactional grants & death

**Status:** IN PROGRESS.

- No corpse/ground drop entity, TTL, F interaction, distance/LOS/ownership
  pickup checks or bag-capacity policy.
- Party loot rules are not implemented.
- No client loot toast; rewards are auto-granted to the killer.
- DB replay 100×, crash-between-claim-and-commit and concurrent party loot tests
  require PostgreSQL and are missing.
- Death ID is process-local; durable death nonce/outbox and queue-full telemetry
  are absent.

### PR-26 — Reconnect, checkpoint, safe spawn & recovery

**Status:** IN PROGRESS.

- No session resume token; reconnect rebuilds from checkpoint.
- Combat logout/AFK/death-respawn policy is not defined.
- WS disconnect, refresh during PvP, zone crash and slow DB simulations are
  missing.
- Durable death nonce/outbox and crash-safe transfer/ledger replay are
  incomplete.
- No PostgreSQL integration test for checkpoint upsert/worker.
- Cross-process lease recovery is absent.

### PR-27 — Portals, multiple zones, transfers

**Status:** IN PROGRESS.

- Region.csv now maps all active KHU_VUC and SAFE_ZONE rows to generated
  region-local Phaser maps; NGU_VUC remains an outer legacy/background layer.
  Backend import registers the generated child maps, and the Phaser host joins
  using the character's current region and loads that map from the welcome.
- Local dev env targets the 125 child maps with matching `regions-v1` /
  `region-content-v1` versions. This covers the detailed child grid, including
  the SAFE_ZONE cells needed to avoid gaps.
- Region.csv remains an Admin-imported dataset; after restarting on the updated
  backend, import Region.csv once to register the generated map catalogue rows.
  Admission now resolves KHU_VUC/SAFE_ZONE directly to their child map IDs even
  if an older `region_world_maps` row still points at `world_map`.
- Border-band contact (anywhere on a side with an authored portal) now triggers
  a server-owned handoff; the portal marker is visual only. The client removes
  its old 1.4-cell proximity trigger.
- `Admission.TransferAtEdge` persists `current_region`, `x/y` (legacy
  projection), HP and clears `map_instance_id`; it then sends `world.transfer`
  and the client reconnects with the new zone.
- Still missing: party follow across a border, a dedicated loading overlay
  (the reconnect path is reused), destination-map spawn validation absent from
  the parser, multi-process lease fencing, and crash-safe transfer rollback.

### PR-28 — Party, Boss, PvP ruleset/arena world

**Status:** IN PROGRESS.

- Party loot rules and party target markers are missing.
- Duel consent, safe sub-zone behavior and live party changes are incomplete.
- Boss AoE telegraphs and multi-player wipe/final-hit/no-duplicate-drop
  regression are missing.
- Arena world integration is partial; old arena remains separate.
- Boss threat/splash/leash exist, but full group dungeon E2E is not measured.

### PR-29 — Vertical-slice end-to-end gate

**Status:** IN PROGRESS.

- No browser E2E for login → MMO → move → aim/cast → kill → loot → bag → exit →
  reconnect.
- Per-owned-Gu runtime verification and full 6-Gu QA matrix are incomplete.
- Corpse/F loot and DB reconnect-under-combat are missing.
- Gate B is **PARTIAL**; only headless 5-player/10-mob evidence exists.
- No reproducible one-command staging setup or load measurement.

### PR-30 — QoS profiling, client/server optimization

**Status:** IN PROGRESS.

- Tick/entity/player/subscriber stats and F3 exist, but AOI bytes, queue depth,
  drop/resync, RTT/jitter and frame-time metrics are missing.
- No 50/100/150/200 ms network emulation.
- 5/30/100 real-network/browser benchmarks are **NOT MEASURED**; only synthetic
  in-process CPU timings exist.
- Allocator churn (~1.1 MB/tick at 100 synthetic players) is identified but not
  optimized.
- Atlas/batching/pooling/spatial/AI budget work remains driven by unmeasured
  hotspots.

### PR-31 — Safe cutover, deprecate Pixi/turn combat hot path

**Status:** IN PROGRESS.

- No real cohort canary; only allowlist/runbook design.
- Full mutation audit and admin-controlled drain endpoint are missing.
- The old turn-based map UI (`WorldPage`, `WorldCanvas`, `ExplorationMap`) has
  been removed from the active frontend. Legacy world store/API paths remain
  for ancillary features; Pixi/arena/store cleanup is still deferred.
- Rollback rehearsal on staging with real accounts is not performed.
- Cross-process lease fencing and flag-OFF browser E2E are still prerequisites.

### PR-32 — Multi-zone worker scaling + instance capacity

**Status:** IN PROGRESS.

- All zone instances are goroutines in one process; no dedicated workers.
- Idle instances release their world/runner after five minutes without a lease,
  subscriber, projectile or pending event, and share one immutable map artifact
  per map. Load tests for many simultaneously active instances are still absent.
- No cross-process lease registry/fencing, gateway routing or session directory.
- Per-zone graceful drain, transfer storms and worker-crash recovery are missing.
- No autoscaling policy or multi-zone load test.
- 100 players/zone and multi-worker capacity are **NOT MEASURED/UNCLAIMED**.

### PR-33 — CI/CD staging + observability + content versioning

**Status:** IN PROGRESS.

- CI covers Go/race/frontend/Gu/map/release-gate, but no browser E2E or load
  benchmark.
- Gu content-hash handshake on join is absent (map hash/version gates exist).
- No Prometheus/OpenTelemetry metrics/dashboards by zone/player/network/economy.
- No log-redaction audit.
- Staging smoke, migration dry run, backup/restore and canary/rollback
  repeatability are not measured.
- Admin realtime stats endpoint exists; alerting is not wired.

### PR-34 — Security, fuzz, soak, crash & dupe testing

**Status:** IN PROGRESS.

- Gu/equipment/loadout fuzzing is incomplete.
- No 6–24 h soak; only short reconnect/ownership smoke runs.
- No DB/Redis disconnect, zone/gateway crash, network jitter/loss or
  split-brain injection.
- No cross-account quest/party/PvP/loot/debug-endpoint audit.
- DB loot replay 100×, crash between claim/commit and checkpoint-loss bounds are
  not measured.
- Bounded death/input queues can drop without counters.
- No production exploit/dupe-free claim is justified.

### PR-35 — Alpha content expansion + Cổ Trùng parity

**Status:** IN PROGRESS.

- No NPC/quest/social/drop authoring; 4 maps contain monsters/boss/portals only.
- No native `UNIQUE` handlers or multi-Gu synergy combos.
- No PvE/PvP separate balance pass, hidden-passive authoring or GM/debug replay
  tools.
- No per-playable-Gu combat regression; only compile/load tests.
- Real icons/cast/impact assets are placeholders.
- Gate C is **NOT PASSED**; Alpha profiles are intentionally simple.

### PR-36 — Beta release, operational criteria & future roadmap

**Status:** IN PROGRESS; Gate D **HOLD**.

- Feature-freeze regression matrix and release script exist, but the real
  regression run is **NOT MEASURED**.
- SLO thresholds are candidates, not measured from load; no 5/30/100
  real-network data.
- No staging canary 5% → 25% → 100% and no rollback rehearsal.
- No browser/network load, DB crash/dupe replay, on-call drill or signed QA/
  operations approval.
- Prometheus/OTel dashboards, alerting and redaction audit remain incomplete.
- Gate D cannot be marked passed from the automated subset alone.

## What is safe to claim today

- Flags default OFF; legacy world remains the rollback path.
- Headless fixed-step slice: 5 players / 10 mobs, movement/cast/death metrics.
- Protocol v1 join/input/cast/move-target and server-authoritative combat core.
- 46 compiled Gu profiles (6 MVP + 40 catalog-only Alpha).
- 5 shared maps (preview + 4 Alpha) with server/client byte parity and linked
  portals.
- Idempotent loot transaction logic, checkpoint/reconnect logic, process-local
  transfer, party/PvP ruleset, in-process instance allocation.
- Race tests, fuzz seed corpus, short reconnect soak, CI checks and a repeatable
  automated release gate.

## What must not be claimed yet

- Gate B, Gate C or Gate D passed.
- 100 players/zone or multi-worker capacity.
- Browser FPS/VRAM, RTT/jitter or AOI bandwidth quality.
- Production-grade dupe-free economy under crash/concurrency.
- Complete Gu parity, NPC/quest/social content or final art.
- Mobile touch, GvG or large-world readiness.

## Evidence needed to close the remaining gaps

1. Produce PR-01 baseline audit/flow docs, backup restore and timestamped test
   report.
2. Run browser E2E with 2–5 clients and flag-OFF legacy; capture FPS/VRAM,
   input/UI and reconnect/loot flows.
3. Run 5/30/100 real-network load with RTT/jitter/loss, AOI bytes, queue/drop/
   resync and server tick p99.
4. Add PostgreSQL/Redis crash, replay-100×, checkpoint/transfer and restore
   tests; fence ownership cross-process.
5. Add Prometheus/OTel dashboards, alerts and redaction audit; rehearse on-call.
6. Complete NPC/quest/drop/social content, native `UNIQUE` handlers, hidden
   passives and a PvE/PvP balance/regression pass.
7. Rehearse canary 5% → 25% → 100% and rollback, then fill and sign
   `docs/mmo/release-checklist.md`.
