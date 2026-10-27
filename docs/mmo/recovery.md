# Reconnect, checkpoint & recovery (PR-26)

Realtime position is checkpointed in `players` and is **never written per
tick**. `players.x/y` remain an integer compatibility projection for legacy
gameplay; `players.world_x/world_y` are the authoritative MMO-local coordinates.

## Durable state

`players` (`backend/internal/db/schema_players.go`):

```text
current_region, world_zone_id, world_instance_id
world_x, world_y, world_safe_x, world_safe_y, world_location_revision
```

`player_world_state_v2` now stores only HP/shield and map/content versions.
Location writes compare `world_location_revision`, so a late checkpoint cannot
overwrite a newer legacy move or zone transfer. The legacy-move trigger clears
the realtime zone marker and advances the revision when an old movement path
changes location.

## Join / leave

- **Join**: the gateway loads location from `players`. Map/content versions in
  the vitals checkpoint gate restoration of checkpoint HP/shield; legacy
  coordinates are transformed only when no realtime location has been committed.
  Blocked/out-of-map coordinates go through `Manager.SafeSpawn`. A configured
  store read failure denies join with
  `realtime_checkpoint_unavailable` rather than guessing.
- **Leave/disconnect**: the gateway snapshots the owned entity, persists
  MMO position to `players` and HP/shield + versions to the checkpoint before releasing the lease. Reconnect creates a
  fresh entity and receives a full `world.welcome` baseline; input ack resets.
- **Periodic checkpoint**: the server runs a coarse 10s worker that batches all
  live player entities per zone through one transaction. It never runs inside
  the zone tick and does not save dead players.
- A crash can lose at most the interval since the last checkpoint/leave; it does
  not create a duplicate entity or lease in the process-local preview.

## Legacy HP bridge

Inventory panels stay on the legacy HTTP paths while a character is in the
realtime zone. HP-changing items and support/consumable Gu therefore do not read
or write `players.hp` blindly:

- Before applying effects, the handler seeds `player.HP` from the live zone
  entity through `middleware.RealtimePlayerHP`.
- After the database transaction commits, the resulting HP delta is applied back
  to the zone entity (`Manager.ApplyHPDelta` → `sim.World.QueueHPDelta`) on the
  zone tick. The zone remains authoritative and its next checkpoint writes the
  healed value.

Active Gu heals are separate: every cast mode (TARGET, DIRECTION, GROUND and
SELF) now resolves `HEAL`/`SHIELD` on the caster, matching the turn simulator,
so a targeted recovery Gu heals while still damaging its target. Dead zone
entities are ignored by the bridge; item use cannot revive a character.

## Safe spawn

`Manager.SafeSpawn` validates the preferred position against the shared map
collision and falls back to authored spawns/free tile. This runs both for legacy
coordinates and stale checkpoints. Death/respawn policy is not implemented yet;
dead-player checkpoints are skipped.

## Loot / economy recovery

Loot grants are already exactly-once in PostgreSQL (`realtime_loot_grants`).
If a grant transaction commits, replaying the same death key cannot duplicate
items. If the process crashes before the death worker handles the event, that
unprocessed death event is lost (bounded in-memory queue); a durable death
nonce/outbox is still future work. Do not treat Redis as the source of truth.

## Known gaps

- Session resume token/zone transfer are not implemented; reconnect rebuilds
  from the checkpoint.
- `safe_x/y` in the periodic checkpoint currently tracks the latest live
  position after validation, not a distinct “last in-combat-safe” marker.
- Death/respawn/AFK/combat-logout policy is pending.
- Durable death nonce and outbox replay for unprocessed kills are pending.
- No PostgreSQL integration test for the checkpoint worker/idempotent upsert in
  the unit suite yet.
