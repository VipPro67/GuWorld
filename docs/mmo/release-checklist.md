# Release checklist & Gate D evidence (PR-36)

This is the beta/release checklist for the opt-in realtime MMO. It is a
**template plus current evidence**, not a signed release. Gate D is **not
passed** in this repository: the automated subset is green, but browser/network
load, staging canary, DB crash/dupe replay and on-call drills have not been
measured.

Related documents:

- rollout flags and rollback: `docs/mmo/cutover.md`, `docs/mmo/rollout.md`
- SLO candidates and capacity decision: `docs/mmo/slo.md`
- soak/fuzz/dupe plan: `docs/mmo/security.md`
- reconnect/checkpoint bounds: `docs/mmo/recovery.md`
- staging operations: `docs/mmo/operations.md`
- known content gaps: `docs/mmo/vertical-slice.md`, `docs/mmo/gu-coverage.md`
- all remaining PR-01…PR-36 gaps: `docs/mmo/remaining-limitations.md`

## 1. Release identity

Fill before every candidate; attach to the QA ticket.

| Field | Value |
| --- | --- |
| Candidate commit | _fill_ |
| Backend image/tag | _fill_ |
| Frontend build/tag | _fill_ |
| Map artifacts/hash | _fill_ |
| Gu compiled hash | _fill_ |
| Protocol version | 1 |
| Config flags | _fill allowed zones/versions/cohort_ |
| QA owner / date | _fill_ |
| Operations owner / date | _fill_ |

## 2. Automated subset (required)

```bash
scripts/mmo-release-gate.sh
```

(or run the manual **MMO release gate** GitHub workflow and attach its log).

The script runs formatting, vet, backend tests, realtime/ws race tests, Gu
coverage/compile parity, map server/client parity, the 5-player slice, a short
reconnect/ownership soak, and frontend lint/typecheck/test/build. It prints an
explicit `NOT MEASURED` list and must not be used as the only Gate D evidence.

Current status on this branch:

| Check | Status | Evidence |
| --- | --- | --- |
| Go format/vet/tests | PASS | `go test ./...`, `go vet ./...` |
| Realtime/ws race | PASS | `go test -race ./internal/realtime/... ./internal/ws/...` |
| Gu coverage CSV parity | PASS | strict audit; CSV unchanged |
| Compiled Gu catalog parity | PASS | 46 profiles, combined hash |
| Map server/client parity | PASS | preview + 4 Alpha maps |
| 5-player headless slice | PASS | `cmd/mmo_slice` |
| Reconnect/ownership smoke | PASS (short) | `cmd/mmo_soak`; long soak NOT MEASURED |
| Frontend lint/typecheck/test/build | PASS | 127 tests + production build |
| Browser E2E | **NOT MEASURED** | no Playwright/Puppeteer in this repo |
| Real-network load | **NOT MEASURED** | synthetic harness only |
| DB/Redis crash + dupe replay | **NOT MEASURED** | no DB fixture in this environment |
| Staging canary/rollback | **NOT MEASURED** | requires staging accounts |
| On-call drill | **NOT MEASURED** | no paging/dashboard drill |

## 3. Feature-freeze regression matrix

Freeze gameplay/content merges before the candidate build. Run the following
and attach pass/fail with links to logs.

| Area | Scenarios | Automated evidence | Manual/browser evidence |
| --- | --- | --- | --- |
| Flag OFF legacy | login, map, explore/turn combat, arena, inventory | existing backend/frontend suites | **required** |
| Movement | WASD, click path, replan, blocked tile, resize/focus loss | sim/input/path tests | **required** |
| Aim/cast | TARGET/DIRECTION/GROUND/SELF, projectile, AoE, interrupt | combat/sim tests | **required** |
| AI | idle/patrol/aggro/chase/attack/leash/respawn, boss splash/threat | world/AI tests | **required** |
| Gu loadout | MVP + Alpha catalog, missing/unsupported Gu, revision | effects/gateway tests | **required** with owned Gu |
| Economy/loot | kill -> grant once, retry/reconnect, inventory/bag refresh | transaction logic tests | DB replay **required** |
| Party/PvP | friendly fire, support, safe/PvE/PvP ruleset, DR | party/world tests | **required** |
| Cross-zone | portal metadata, transfer, duplicate ownership | transfer tests | staging **required** |
| Disconnect/reconnect | leave, WS drop, checkpoint resume, combat drop | gateway/fake-store tests | browser+DB **required** |
| Social/business panels | chat/party/inventory/cultivation while combat runs | HUD/store tests | **required** |
| Content | map link/spawns/monster IDs, Gu compile, coverage drift | nav/effects/gudata tests | spot-check **required** |
| Rollback | pause admission, drain, legacy route, rerun | design/unit coverage | rehearsal **required** |

Minimum exit: no new P0/P1 defect open in movement/combat/economy/social, and
every manual row has an owner/date/log.

## 4. SLO / capacity gate

Use `docs/mmo/slo.md` thresholds. Gate D requires:

- 5-client and 30-client real-network runs with p50/p95/p99 tick, RTT/jitter/
  loss, FPS/VRAM and AOI bytes recorded;
- queue depth/drop/resync and reconnect-rate evidence;
- PostgreSQL/Redis latency and failure injection results;
- explicit decision for 100/zone: pass, defer, or require instance/worker split.

Do not raise `MMO_MAX_PLAYERS_PER_ZONE` or advertise 100/zone from the synthetic
benchmark alone. The current process-local instance split does not fence
cross-process ownership.

## 5. Capacity rollout

Realtime MMO is open to all signed-in accounts. Increase capacity only after
measuring stable load; there is no player allowlist or legacy-world cohort.

| Stage | Cohort | Duration/exit | Rollback trigger |
| --- | --- | --- | --- |
| Internal | QA + ops accounts, 1 zone | 1 session + full regression | any duplicate loot/owner, crash loop |
| 5% | ~5% of target cohort (min 1 account) | 24 h stable or 2 peak cycles | tick p99 > 25 ms sustained, reconnect < 95%, unexplained 5xx |
| 25% | ~25% of target cohort | 48 h stable or 2 peak cycles | queue drops without resync, FPS regression, economy mismatch |
| Public | all signed-in accounts | 7 days of measured stable operation | any release blocker or unowned state corruption |

Per stage:

1. Freeze content; record hashes and config.
2. Apply the allowlist/cohort change; do not change gameplay values mid-stage.
3. Watch admin `/api/admin/monitor/realtime`, tick/queue metrics, error logs,
   loot-grant counts, reconnect rate and player reports.
4. Run a smoke flow: join, move/aim/cast, kill+ loot, party/PvP, leave/rejoin,
   portal if enabled.
5. Write the stage result and decision (advance / hold / rollback) in the QA
   ticket. No verbal-only advancement.

## 6. Rollback rehearsal

Rehearse before the first external cohort:

1. Trigger graceful drain/shutdown; confirm checkpoint upserts, leases released
   and no duplicate owner after restart.
2. Redeploy the previous known-good application version if rollback is needed;
   MMO admission is not controlled by environment feature flags.
3. Keep `realtime_loot_grants` and checkpoints; do not delete rows.
4. Restart, re-run the release gate and a signed-in smoke session.
5. Record elapsed time, data loss window and any manual steps.

Any rehearsal failure is a Gate D blocker.

## 7. Anti-dupe / consistency checks

Before release and during every canary stage:

- replay the same death key 100× against PostgreSQL and verify one grant row and
  one inventory/Gu delta;
- kill the process between claim and commit, restart, replay, verify no
  duplicate and no orphan;
- disconnect mid-combat, reconnect, verify one entity/lease and checkpoint
  version gates;
- inspect `realtime_loot_grants`, `player_world_state_v2` and inventory counts
  for the test accounts;
- verify unsupported/missing Gu are skipped, never cast as empty skills;
- verify dead players do not write checkpoints and stale map/content versions
  are rejected.

## 8. Database / backup gate

The repository has **no migration runner**; schema lives in
`backend/internal/db/schema_*.go` and `InitSchema` creates missing tables/indexes
on startup. Treat any release that changes schema as a coordinated deploy, not a
casual restart.

Before releasing:

- [ ] Take a PostgreSQL backup/snapshot; record restore point and elapsed time.
- [ ] Restore that backup into staging and boot the candidate once.
- [ ] Confirm `realtime_loot_grants` and `player_world_state_v2` exist with the
      expected unique keys/foreign keys, and that the realtime checkpoint worker
      can upsert.
- [ ] Never run destructive down-migrations or delete loot/checkpoint rows to
      "fix" a canary issue; roll back flags/routes instead.
- [ ] For new columns, declare them inline in the matching `schema_*.go` and
      verify a **fresh** database init plus an existing staging database boot.
- [ ] Deploy backend before/with the frontend build that expects the new
      protocol/content version; reject stale clients via `world.join` gates.
- [ ] Record DB/Redis versions and any extension assumptions in the release
      ticket.

## 9. Operations / on-call readiness

| Item | Required evidence |
| --- | --- |
| Dashboards | per-zone tick/entity/player/subscriber, WS errors, queue depth, loot/checkpoint |
| Alerts | page on sustained tick p99, crash-loop, duplicate-grant log, DB/Redis outage |
| Runbooks | cutover/rollback, incident, DB restore, projection/cache rebuild, on-call contacts |
| Access | admin monitor permission, DB read-only access, deploy/rollback credentials |
| Comms | player-facing note for world/Gu changes, known issues, support macro |
| Drill | one simulated tick spike + one simulated duplicate grant, timed |

The realtime admin endpoint exists; Prometheus/OpenTelemetry dashboards and the
redaction audit are still pending (`docs/mmo/operations.md`).

## 10. Player communication

For any beta cohort stage, publish:

- what is opt-in and how to leave/rollback;
- known limitations (preview skills, no corpse/F loot interaction, NPC/quest
  gaps, Alpha balance provisional);
- that legacy remains the fallback during canary;
- how to report a bug (account/zone/time, screenshot/F3 counters, no secrets);
- expected maintenance window and rollback behavior.

## 11. Known release blockers

These must be resolved or explicitly accepted by the release owner:

- browser E2E and measured FPS/network/AOI evidence missing;
- no PostgreSQL/Redis crash/dupe replay in the automated suite;
- process-local leases only; cross-process split-brain unfenced;
- durable death nonce/outbox for kills that crash before the worker handles them;
- NPC/quest/social/drop authoring and unique Gu handlers incomplete;
- Prometheus/OTel dashboards and log-redaction audit incomplete;
- no canary/rollback rehearsal on staging yet.

## 12. Sign-off

Gate D requires all three signatures plus attached evidence. A checkbox without
a log link is not evidence.

| Role | Name | Decision (pass/hold/reject) | Date | Evidence link |
| --- | --- | --- | --- | --- |
| QA lead | | | | |
| Backend/realtime owner | | | | |
| Operations/on-call owner | | | | |
| Product/content owner | | | | |

**Current decision: HOLD.** Gate D is not passed; this branch only provides the
automated subset and the checklist needed to run the remaining gates.
