# Cutover & rollback

Phaser is the only player world client. The turn-based combat start, capture,
retreat and flee entrypoints have been removed; combat resolves through the
authoritative realtime runtime. The player model and fresh schema no longer
contain the turn-based `active_combat` snapshot. Existing databases may retain
the unused legacy column because this project does not apply schema migrations.

## Runtime configuration

Realtime MMO is enabled for every signed-in player. The server automatically
loads all public maps matching the shipped release; feature flags, player
allowlists, and zone allowlists are not used. `MMO_MAX_PLAYERS_PER_ZONE`,
`MMO_INSTANCES_PER_ZONE`, and asset directory variables remain optional
capacity/deployment tuning.

## Rollout stages

Gate D evidence/sign-off is in `docs/mmo/release-checklist.md`; candidate SLOs
and the capacity decision matrix are in `docs/mmo/slo.md`. The stages below are
the operational view of the same cohort plan.

1. Deploy backend and matching map artifacts together.
2. Verify database world-location migrations and map catalogue load in startup logs.
3. Monitor capacity and zone health; tune per-zone capacity if needed.

## Rollback

1. Call the admission drain path (`Admission.Drain`, also invoked during graceful
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
