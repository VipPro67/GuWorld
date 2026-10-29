# Realtime MMO runtime

Realtime MMO is the primary world for every signed-in player. Public map
catalogue entries matching the shipped release are loaded automatically;
admission validates the authenticated character's actual location and map
release. Private aperture and combat instances remain isolated by server-owned
instance IDs.

## Current state

- All signed-in players can join the realtime world; no rollout flags, zone
  allowlist, or player allowlist are required.
- Map/content versions are fixed by the shipped map artifacts and checked by
  the backend; they are not deployment environment variables. The client sends
  the same versions from the shared release contract.
- `MMO_MAX_PLAYERS_PER_ZONE` defaults to `5` and accepts values from `1` to
  `100`.
- Startup loads every shipped public map and creates its zone loops. The manager
  supports process-local player entity join/leave, ownership leases, and a
  configured per-zone capacity.
- `backend/internal/realtime/config/config_test.go` covers public admission,
  fixed release versions, and capacity bounds.
- Phaser 4.2.1 is pinned in `frontend/package.json` and the lockfile. The
  client mounts a lazy, client-only Phaser host. `NEXT_PUBLIC_MMO_ZONE_ID` is
  only a fallback when the character has no current region. The host joins the
  v1 protocol, renders AOI snapshots, sends WASD/move_target
  intents and reconciles local prediction. The scene still uses placeholder art
  and no Phaser physics.
- `world.move_target` is now accepted after join. The zone clamps the requested
  destination to 64 world units, walks with the same collision/speed path as
  WASD, and stops on arrival or when it cannot advance. Blocked destinations and
  mid-path collisions emit `MOVE_TARGET_REJECTED`.
- PR-22 adds a client-only A* preview (`PathPlanner`) that decomposes a click
  into waypoints; each waypoint is still a normal server-validated
  `world.move_target`, never a client path authority. WASD/Esc cancels the
  queue; the camera has follow deadzone/zoom/shake and an offscreen target
  marker.
- PR-23 builds the hotbar at join from the character's equipped ACTIVE Gu
  (`player_gu` + `players.equipped_gus` order) matched to compiled realtime
  profiles. Unsupported Gu are skipped with an audit entry; characters with no
  compiled owned Gu fall back to the preview loadout. The loadout and a
  deterministic `loadout_revision` are cached in the session. Live loadout
  refresh without rejoin, equipment stat projection and quest/NPC world
  triggers are still pending.
- PR-24 integrates the legacy React business UI into the MMO view: inventory/Gu,
  cultivation, party and settings panels plus the existing global chat overlay.
  Opening a panel sets a client-side UI-modal flag that blocks movement/hotkeys
  in Phaser; the server does not pause the zone. Panel state uses the existing
  Zustand store and coarse `world.welcome`/snapshot projections (no per-frame
  entity state in React).
- PR-25 turns `ENTITY_DIED` into idempotent durable loot: a manager death worker
  resolves `monster_templates.loot_table_id`, claims a unique
  `realtime_loot_grants` key, and grants items/Gu in the same DB transaction.
  Replays return `AlreadyGranted`; crash/failure rolls back the claim and rewards.
  Rewards are currently auto-granted; corpse/ground drop, F interaction, party
  loot and client loot toast remain pending.
- PR-26 checkpoints realtime-local position in `players` with a location
  revision; `player_world_state_v2` keeps HP/shield and map/content versions.
  Join restores only matching-version vitals, leave/disconnect flushes the owned
  entity before releasing the lease, and a 10s worker checkpoints outside the
  zone tick. Safe spawn validation is reused; session
  resume token, death nonce, combat-logout policy and outbox replay remain
  pending.
- PR-27 lands portal metadata validation/rendering and a process-local atomic
  `Manager.TransferCharacter` handoff (remove source → add target → single owner
  update, rollback on failure). Automatic portal triggers, DB map/region update,
  `world.transfer`/client scene swap and multi-process fencing remain pending.
- PR-28 adds a zone ruleset (`pve`/`pvp`/`safe`), party-aware player harm rules,
  `players.party_id` projection, boss threat tables with leash reset and boss
  splash damage. Duel consent, party-shared loot, boss telegraph/arena world and
  live party changes remain pending.
- PR-29 adds a headless 5-player/10-mob vertical-slice harness and a Gate B
  checklist with PASS/PARTIAL/NOT MEASURED; browser E2E and real load remain
  unmeasured.
- PR-30 adds a synthetic 5/30/100-player tick benchmark, `Manager.Stats`
  telemetry and an F3 client debug HUD for tick/network/prediction/FPS. Browser
  frame-time/VRAM, RTT/jitter/loss, AOI bytes/client and DB/Redis load remain
  NOT MEASURED; see `docs/mmo/benchmark.md`.
- PR-31 adds the admission drain path, which checkpoints and releases every
  session on graceful shutdown. Old rollout environment gates have since been
  removed; see `docs/mmo/cutover.md` for current operations.
- PR-32 adds in-process instance scaling: `MMO_INSTANCES_PER_ZONE` (1–8),
  least-loaded instance allocation with per-instance capacity and
  `realtime_zone_full`; `Manager.Stats` exposes per-zone counts. Dedicated worker
  processes, cross-process leases/routing and transfer storms are pending; see
  `docs/mmo/scaling.md`.
- PR-33 expands CI to frontend lint/typecheck/test/build, backend race tests and
  strict Gu audit/compile, adds `GET /api/admin/monitor/realtime` per-zone stats
  and documents content hashes/staging/incident playbook. Prometheus/OTel
  dashboards, redaction audit and a staging dry run remain pending; see
  `docs/mmo/operations.md`.
- PR-34 adds protocol/sim/map fuzz seeds, spoof/oversized/controller-reuse tests
  and a reconnect ownership soak harness (`cmd/mmo_soak`). DB dupe replay, 6–24h
  soak, crash/jitter injection and full cross-account audit remain pending; see
  `docs/mmo/security.md`.
- PR-35 adds 40 catalog-only Alpha Gu profiles (46 compiled total) and four
  linked 64×48 maps (`misty_forest`, `bone_crypt`, `crimson_ravine` PvP,
  `emerald_grotto` safe) with a boss spawn and bidirectional portals. The nav
  test validates every portal target is walkable in the target map. NPC/quest/
  social/drop authoring, unique handlers and the Alpha balance pass remain
  pending; see `docs/mmo/map-format.md` and `docs/mmo/gu-realtime-schema.md`.
- PR-36 adds the Gate D release checklist, provisional SLO/capacity gates,
  canary/rollback/comms plan, post-beta roadmap and
  `scripts/mmo-release-gate.sh`. The automated subset can pass locally; Gate D
  remains HOLD until browser/network load, DB crash/dupe replay, staging canary
  and on-call evidence are attached; see `docs/mmo/release-checklist.md` and
  `docs/mmo/slo.md`.
- Configured zones load `<map_id>.json` from `MMO_MAP_DIR` (default
  `data/mmo/maps`) at startup. The sample `realtime_preview` map provides a
  64×48 collision grid, obstacle blocks and authored spawns. A missing or
  invalid file logs an error and the zone starts **without static collision**;
  that state is preview-only and must not be used for production admission.
  `nav/map_test.go` keeps the server and client map copies byte-identical, and
  `world.welcome.map_hash` lets the client detect a stale downloaded artifact.
- Admission validates the database position against the loaded map and replaces
  blocked/out-of-map legacy coordinates with an authored safe spawn. Client
  clicks on solid cells are rejected before a `world.move_target` is sent; the
  server still enforces collision independently.
- Monster spawns declared in the map (`type: "monster"` + `monster_id`) are
  created at zone startup from `monster_templates` with server-owned AI. They
  never call the legacy turn-based encounter worker. AI think runs at 4 Hz with
  idle/patrol/aggro/chase/attack/leash/death/respawn states; attacks reduce
  player HP on the zone tick.
- `world.cast` runs a preview combat pipeline: six placeholder skills across
  `TARGET`/`DIRECTION`/`GROUND`/`SELF`, windup/GCD/cooldown/mana checks,
  projectile swept collision, ground AoE with target cap, shield absorption,
  heal, cast interruption and server-authored `world.events`.
- PR-19 adds tick-scheduled statuses (DoT/HoT, stack policies
  `REPLACE`/`REFRESH`/`STACK`/`MAX`/`IGNORE`, refresh/expiry, PvP duration DR,
  zero move multiplier blocks movement/casting, cleanse) and a preview hidden
  passive (HP ≤ 40% + 3 hits → 25 shield, 10s ICD, server-only discovery). The
  preview slot 6 `Frost Lock` exercises freeze live; compiled
  `GU_WHITE_FROST` awaits the per-player loadout adapter. Loot, full Gu effect
  parity, reliable event replay and player death/respawn UI are still missing.
- PR-20 adds the Phaser/React hybrid skill UX: a server-projected hotbar in
  `world.welcome.skills`, React HP/mana/cooldown/cast/target HUD, TARGET lock,
  DIRECTION/GROUND two-step aim with range telegraphs, Esc/right-click cancel,
  and server snapshot projectiles rendered by Phaser. Cooldown/mana are display
  only; the zone still rejects invalid casts.
- The authenticated WebSocket accepts protocol-v1 `world.join`. It binds the
  character from auth middleware, loads region/instance/position/vitals directly
  from PostgreSQL, validates allowlist and version gates, and returns a
  `world.welcome` snapshot. It does not trust a player ID or spawn coordinates
  from the join payload.
- After join, `world.input` uses the authenticated controller lease, a per-
  connection queue capped at 64 messages, and a 30 messages/second ingress
  limit. Sequence and finite normalized movement vectors are validated before
  input enters the bounded simulation queue. Clients must wait for
  `world.welcome` before sending input. Snapshots are streamed at 20 Hz for a
  circular AOI of radius 32 world units, with periodic baselines, baseline
  deltas, removed entity IDs, and `ack_seq`. Client interpolation/resync and
  byte-budget metrics are still pending; keep admission allowlisted.
- The legacy monster worker now considers only the default `world_map` with an
  empty `InstanceID`; a non-empty isolated instance, including a future MMO
  session, does not activate that worker. It also skips characters with an
  active in-process realtime lease. Realtime-owned characters are also
  rejected by the legacy movement, combat, loot, and websocket move command
  paths. Teleport arrays and realtime tribulation entry/victory/escape are the
  exceptions: those handlers flush the live position, persist the destination,
  release the lease and ask the client to rejoin the destination zone. These
  guards activate only after an in-process lease exists.
- `Manager.JoinCharacter`/`LeaveCharacter` pair in-memory player membership
  with a controller lease and enforce the zone capacity. `ReplaceCharacterLease`
  protects reconnects from stale tokens; entity add/remove is delivered through
  a bounded lifecycle queue and applied on the zone tick. The authenticated
  gateway now calls these APIs; leases are memory-only and have no expiry,
  persistence, or cross-process coordination.
- Explicit `command/world.leave` and WebSocket disconnect release membership.
  Reconnect creates a fresh controller and lease; a concurrent second
  controller receives a conflict.
- Other gameplay mutations such as inventory, quests, trading, and progression
  have not been audited or guarded for realtime-owned characters. Keep
  production admission flags off until that audit, snapshot streaming, and the
  remaining gameplay command paths are complete.
- Claim-before-load prevents new legacy requests from starting after ownership
  is reserved, but already-running DB transactions/monster encounters can still
  finish. The legacy position index also still contains the pre-join DB position
  and may show a ghost to legacy clients. Both require a durable world-mode
  cutover and index quarantine before production admission.

## Safe activation order

1. Deploy the code and map artifacts together; no MMO feature flags are needed.
2. Confirm the server loaded all maps matching the shipped release and that
   database migrations for world location are applied.
3. Keep `MMO_MAX_PLAYERS_PER_ZONE` at its default or set an appropriate capacity.
4. For maintenance, use the existing graceful drain/shutdown path; do not use
   player or zone allowlists as feature switches.

## Ownership limits still to address

- One character may have one active controller/session at a time within one
  process; durable and cross-process lease coordination remain future work.
- Reconnect releases membership on disconnect before a new connection joins;
  cross-tab joins must not create two authoritative sessions within the process.
- Legacy operations already past their ownership check may still commit, and
  the old world-position index is not yet quarantined. Database-backed mode
  reservation/transaction fencing remains a release gate.
- A realtime zone must own its player and monster state exclusively. The legacy
  monster worker now skips owned players, while other legacy mutation paths
  still need a complete audit.
- The gateway now loads trusted state, checks map and rollout/version gates,
  and calls the manager lifecycle APIs. Production allowlisting must remain
  empty until the remaining legacy mutation audit and authoritative movement
  protocol are complete.
