# Zone scaling & instance capacity (PR-32, in-process stage)

## What is implemented

- Multiple zone instances per map in one process:
  - `MMO_INSTANCES_PER_ZONE` (1–8, default `1`);
  - instance IDs: `realtime-preview`, `realtime-preview-2`, …;
  - every instance gets its own fixed-step goroutine, world, AOI subscriber map.
- Admission allocation:
  - `Manager.ZoneKeyForMapWithCapacity(mapID)` picks the least-loaded instance
    with `players < MMO_MAX_PLAYERS_PER_ZONE`;
  - no map instances → `realtime_map_unavailable`;
  - all instances full → `realtime_zone_full` (429);
  - already-owned character still returns `realtime_character_already_owned`.
- Capacity is per instance, not per map; legacy AOI/collision/map artifacts are
  shared by loading the same map file per instance.
- Telemetry: `Manager.Stats()` exposes per-instance tick/entity/player/subscriber
  count and last step duration for a future routed endpoint.
- Single-owner transfer remains process-local (`Manager.TransferCharacter`).

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
- Config bounds: `MMO_INSTANCES_PER_ZONE` accepts 1–8, invalid values disable
  admission fail-closed.
