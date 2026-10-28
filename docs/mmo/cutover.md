# Cutover & rollback (PR-31, staged)

The legacy Pixi world and turn-based handlers are still shipped. Do **not** delete
them until the canary/rollback rehearsal in `docs/mmo/vertical-slice.md` passes.

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
5. **Cleanup**: a separate PR after canary stability removes Pixi/legacy
   movement/combat adapters. This PR does **not** remove them.

## Rollback

1. Set `MMO_ADMISSION_PAUSED=true` (or disable `PHASER_WORLD_CLIENT_ENABLED` /
   `REALTIME_WORLD_ENABLED` on a restart) to block new MMO joins.
2. Call the admission drain path (`Admission.Drain`, also invoked during graceful
   server shutdown):
   - pauses admission;
   - checkpoints location to `players` and vitals/map versions to `player_world_state_v2`;
   - releases every process-local lease;
   - clients receive no more snapshots and reconnect to the legacy world.
3. Keep pending loot/ledger rows; the loot grant transaction is idempotent. Do
   not delete `realtime_loot_grants` or run destructive down-migrations.
4. Restore the legacy frontend route/flag and verify the player's legacy
   last-safe spawn.
5. Investigate root cause, re-run the slice/benchmark, then resume the cohort
   gradually.

## Current guard coverage

`RequireLegacyWorldMode` blocks explore/move/turn-combat/capture/
arena/PvP/ground-loot routes plus party invitations and acceptance, merchant,
scouting, spatial realm mutations and sect defense/attack while a realtime
lease owns the character. Teleport arrays are handled by a lease handoff in
`/api/teleport/travel` (flush live position, persist destination, release
lease, client rejoins the destination zone) instead of being rejected. The old
WebSocket `world.move_path` and `match.input` are also rejected. Domain
inventory/quest/tu luyện mutations remain available because they are needed by
the React business panels; they do not drive world position/HP.
Invitation rejection remains available. Scouting omits MMO-owned targets from
legacy coordinate results.

Still pending before a full cutover: complete mutation audit for every
world-affecting endpoint, durable/cross-process lease fencing, a controlled
admin drain endpoint, and a rehearsal on staging with real accounts.
The location table is currently preparation schema only; see
`docs/mmo/location-audit.md` for remaining readers and writers.

## Pixi/legacy removal

Deferred intentionally. Removing `WorldCanvas`, `PvPArenaCanvas`, the old store
paths or simulator adapters is a cleanup PR after:
- flag-OFF legacy E2E passes;
- no MMO cohort can mutate legacy world state;
- rollback has been rehearsed and observed for at least one canary cycle.
