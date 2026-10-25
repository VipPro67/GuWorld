# Reconnect, checkpoint & recovery (PR-26)

Realtime positions/HP are durable but **never written per tick**. The legacy
`players.x/y/hp` rows remain the fallback for players outside the MMO cohort.

## Durable state

`player_world_state_v2` (`backend/internal/db/schema_realtime.go`):

```text
player_id PK -> players(id)
zone_id, instance_id, x, y, hp, shield
safe_x, safe_y
map_version, content_version
updated_at
```

- `zone_id` is the world map ID, `instance_id` is the zone instance.
- `safe_x/y` is the entry safe spawn used as the fallback.
- Map/content versions prevent resuming a stale artifact.

## Join / leave

- **Join**: the gateway loads the checkpoint best-effort. It is used only when
  `zone_id`, `map_version` and `content_version` match the join request. HP 0
  checkpoints fall back to the legacy HP; blocked/out-of-map coordinates go
  through `Manager.SafeSpawn`. A configured store read failure denies join with
  `realtime_checkpoint_unavailable` rather than guessing.
- **Leave/disconnect**: the gateway snapshots the owned entity, persists
  x/y/hp/shield + safe entry before releasing the lease. Reconnect creates a
  fresh entity and receives a full `world.welcome` baseline; input ack resets.
- **Periodic checkpoint**: the server runs a coarse 10s worker that batches all
  live player entities per zone through one transaction. It never runs inside
  the zone tick and does not save dead players.
- A crash can lose at most the interval since the last checkpoint/leave; it does
  not create a duplicate entity or lease in the process-local preview.

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
