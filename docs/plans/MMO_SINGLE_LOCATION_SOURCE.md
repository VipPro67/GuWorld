# Player row as the durable location source

## Decision

`players` is the sole durable location row. `current_region` carries the logical gameplay region. The `world_*` columns carry realtime map/instance, local floating-point coordinates, safe spawn, and a location revision. The legacy integer `x/y` fields remain a compatibility projection while old gameplay handlers still consume them; they are not the realtime source of truth. Zone simulation remains authoritative in memory while a player is online.

`player_world_state_v2` contains only HP/shield and map/content versions. The previously prepared `player_world_location` table is removed from fresh schema and dropped by the explicit cutover script after backup.

## Runtime behavior

- Realtime admission reads the player's current region and canonical world location. It restores checkpoint vitals only when map/content versions match; invalid or stale coordinates pass through safe-spawn validation.
- Checkpoint writes compare `world_location_revision` and advance it atomically with the location. A legacy movement write invalidates the realtime zone marker and increments the revision, preventing a stale realtime checkpoint from restoring the old location.
- Successful realtime region transfers write the destination region, zone, instance, local position, legacy projection, and revision in one player-row transaction.
- Position is checkpointed at join/leave/periodic boundaries, not on every simulation tick.

## Existing database cutover

After backing up the database and draining movement/realtime sessions, run `backend/scripts/migrate_player_world_location.sql` explicitly. It adds the player location columns, copies committed v2 realtime positions when available, installs legacy-write invalidation, then removes location columns from the old checkpoint table and drops the unused preparation table. Players without a v2 checkpoint initialize their world location from legacy coordinates when they first join realtime; admission converts through the configured map transform and validates safe spawn.

This script is not run by `InitSchema`. Fresh databases receive the columns from `schema_players.go` and the vitals-only checkpoint table from `schema_realtime.go`.

## Compatibility and remaining migration work

Existing gameplay handlers continue reading integer `players.x/y` and `current_region`. Realtime checkpoints keep that projection synchronized whenever a map transform is available, and legacy location writes invalidate the realtime location. Handlers that need exact realtime-local coordinates or realtime instance scoping still need to read `world_*` fields or a live zone-owner snapshot before broad MMO rollout. The process-local realtime lease also does not fence separate server processes; multi-process ownership remains a separate deployment requirement.
