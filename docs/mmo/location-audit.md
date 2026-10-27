# Location source cutover audit

The durable realtime location now lives on `players`. `world_*` fields hold the MMO zone, instance, local coordinates, safe spawn and CAS revision. `current_region` remains the logical gameplay region. Legacy `players.x/y` are maintained as an integer projection when the map transform is available.

`player_world_state_v2` now stores only HP/shield and map/content versions. Existing databases must be backed up and drained, then explicitly migrated with `backend/scripts/migrate_player_world_location.sql`; `InitSchema` does not alter existing tables.

| Path | Current behavior | Remaining work |
| --- | --- | --- |
| Join/checkpoint | Admission restores realtime location from `players`; location writes CAS the revision. | Integration test the DB migration and stale version/safe spawn path. |
| Realtime transfer | Destination region/zone/instance/local position and revision commit to `players` in one transaction. | Exercise transfer/reconnect under PostgreSQL and multi-session races. |
| Legacy movement | Existing handlers still consume `current_region/x/y`; a trigger invalidates `world_zone_id` and advances revision when those legacy fields move. | Convert proximity, instance scoping, admin and worker operations that need live MMO position to the zone owner or canonical fields before broad MMO rollout. |
| Existing schema | Explicit SQL script backfills committed v2 positions and drops old v2 location columns and dormant `player_world_location`. | Run only in a maintenance window after backup and drain; verify row counts and rollback procedure. |

The `players.x/y` compatibility projection means legacy readers continue to work without becoming a second durable table. It is not suitable for exact realtime proximity checks; those must use canonical world coordinates and instance ownership.
