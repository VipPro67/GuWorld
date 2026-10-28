# Cutover & rollback

Phaser is the only player world client. The turn-based combat start, capture,
retreat and flee entrypoints have been removed; combat resolves through the
authoritative realtime runtime. The player model and fresh schema no longer
contain the turn-based `active_combat` snapshot. Existing databases may retain
the unused legacy column because this project does not apply schema migrations.

## Flags

Backend:

| Flag | Purpose |
| --- | --- |
| `REALTIME_WORLD_ENABLED` | starts the realtime runtime/zones |
| `PHASER_WORLD_CLIENT_ENABLED` | admits the Phaser client cohort |
| `MMO_ALLOWLIST` | player cohort (empty = deny everyone) |
| `MMO_ADMISSION_PAUSED` | block **new** joins while existing sessions continue |

Frontend:

| Flag | Purpose |
| --- | --- |
| `NEXT_PUBLIC_REALTIME_WORLD_ENABLED` | show the MMO UI entry |
| `NEXT_PUBLIC_PHASER_WORLD_CLIENT_ENABLED` | allow the Phaser route |

All defaults are OFF. Admission is fail-closed; invalid config disables the
runtime.

## Rollout stages

Gate D evidence/sign-off is in `docs/mmo/release-checklist.md`; candidate SLOs
and the capacity decision matrix are in `docs/mmo/slo.md`. The stages below are
the operational view of the same cohort plan.

1. **Local/headless**: `mmo_slice`, unit/race tests, flags OFF.
2. **Staging allowlist**: one map, one account, `MMO_ALLOWLIST` non-empty.
3. **Opt-in cohort**: add player IDs; keep legacy world available for everyone
   else. Do not add a player to both an MMO lease and a legacy movement session.
4. **Canary default**: only after Gate B evidence and a rollback rehearsal; flip
   the frontend entry and keep backend allowlists explicit until the next gate.
5. **Cleanup**: remaining legacy movement and map adapters can be removed after
   the mutation audit and canary rollback rehearsal are complete.

## Rollback

1. Set `MMO_ADMISSION_PAUSED=true` (or disable `PHASER_WORLD_CLIENT_ENABLED` /
   `REALTIME_WORLD_ENABLED` on a restart) to block new MMO joins.
2. Call the admission drain path (`Admission.Drain`, also invoked during graceful
   server shutdown):
   - pauses admission;
   - checkpoints location to `players` and vitals/map versions to `player_world_state_v2`;
   - releases every process-local lease;
   - clients receive no more snapshots; admission stays paused until realtime
     service is restored.
3. Keep pending loot/ledger rows; the loot grant transaction is idempotent. Do
   not delete `realtime_loot_grants` or run destructive down-migrations.
4. Restore realtime service and verify the player's last-safe world spawn.
5. Investigate root cause, re-run the slice/benchmark, then resume the cohort
   gradually.

## Current guard coverage

`RequireLegacyWorldMode` blocks remaining legacy explore/move/arena/PvP/ground-loot
routes plus party invitations and acceptance, merchant,
scouting, spatial realm mutations and sect defense/attack while a realtime
lease owns the character. Teleport arrays and realtime tribulation
entry/victory/escape are handled by a lease handoff (flush live position,
persist destination, release lease, client rejoins the destination zone)
instead of being rejected. The old Legacy WebSocket `world.move_path` is no longer accepted; `match.input` is
also rejected. Domain
inventory/quest/tu luyện mutations remain available because they are needed by
the React business panels; they do not drive world position/HP.
Invitation rejection remains available. Scouting omits MMO-owned targets from
legacy coordinate results.

Still pending before a full cutover: complete mutation audit for every
world-affecting endpoint, durable/cross-process lease fencing, a controlled
admin drain endpoint, and a rehearsal on staging with real accounts.
The location table is currently preparation schema only; see
`docs/mmo/location-audit.md` for remaining readers and writers.

## Remaining legacy removal

Combat entrypoints and the turn-based outcome pipeline are retired. Remaining
movement and map adapters stay until the world mutation audit and rollback
rehearsal are complete.
