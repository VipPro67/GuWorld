# Zone scaling & instance capacity (PR-32, in-process stage)

## What is implemented

- Multiple zone instances per map in one process:
  - `MMO_INSTANCES_PER_ZONE` (1–8, default `1`);
  - instance IDs: `realtime-preview`, `realtime-preview-2`, …;
  - a world, monster set, 20 Hz runner and broadcaster exist only while the
    instance is active; they are created on the first claim/join/transfer.
- Idle lifecycle:
  - an instance with no lease, subscriber, projectile or pending snapshot event
    for 5 minutes (checked once a minute) is stopped and its world is dropped;
  - a claim that is still loading character state counts as a lease and keeps
    the instance alive;
  - joins racing a teardown wait for the same transition, so there are never
    two worlds for one instance; allocation prefers a live instance and only
    waits when no other instance of the map is available;
  - reopening recreates monsters from the map spawns; monster state is not
    persisted across release.
- Admission allocation:
  - `Manager.ZoneKeyForMapWithCapacity(mapID)` picks the least-loaded instance
    with `players < MMO_MAX_PLAYERS_PER_ZONE`;
  - no map instances → `realtime_map_unavailable`;
  - all instances full → `realtime_zone_full` (429);
  - already-owned character still returns `realtime_character_already_owned`.
- Capacity is per instance, not per map. Startup reads and validates every
  configured map once; the immutable collision/spawn artifact is shared by all
  instances of that map instead of being loaded and retained per instance.
- Telemetry: `Manager.Stats()` returns one row per configured instance with
  `map_loaded` (artifact checked at startup), `running` (world currently
  exists), `evicting` (teardown in progress), tick/entity/player/subscriber
  counts and last step duration.
- Single-owner transfer remains process-local (`Manager.TransferCharacter`) and
  activates an idle target instance on demand.

## Not implemented (required for real worker scaling)

- Dedicated worker processes per zone; all instances are goroutines in one Go
  process.
- Cross-process lease registry / fencing (Redis/DB) and stale-owner recovery.
- Gateway routing a WebSocket to the zone-owner process; no session directory.
- Per-zone graceful drain without stopping the manager (`Stop` is global;
  admission `Drain` releases every session).
- Cross-process portal/party transfer and transfer storms.
- Instance autoscaling by load/capacity; allocation currently only on join.
- Load test across multiple zones/workers, split-brain and worker-crash
  simulation.

Do not claim 100 players/zone or multi-worker capacity until those are measured.

## Configuration

```env
MMO_INSTANCES_PER_ZONE=3
MMO_MAX_PLAYERS_PER_ZONE=5
MMO_ZONE_IDS=realtime_preview
```

This creates three instances of `realtime_preview`, each capped at five players
(15 total in this process) and routes joins to the least-loaded instance.

## Tests

- `TestManagerAllocatesMapInstancesByCapacity`: one-player capacity across two
  instances; next join gets the free instance, then `ErrZoneFull`; unknown map
  returns `ErrUnknownZone`.
- Lazy lifecycle: maps are checked without starting worlds; a pending claim
  keeps the instance alive; idle instances are not evicted before five minutes
  and are recreated from spawn on the next claim; in-flight projectiles defer
  eviction; concurrent claims wait for an in-progress teardown and shutdown
  wakes waiters.
- Config bounds: `MMO_INSTANCES_PER_ZONE` accepts 1–8, invalid values disable
  admission fail-closed.
