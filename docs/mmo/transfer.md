# Zone transfers & portals (PR-27, partial)

## Portal metadata

Tiled object layer `portals`:

```json
{
  "type": "objectgroup",
  "name": "portals",
  "objects": [{
    "name": "portal_hub",
    "type": "portal",
    "x": 13.5,
    "y": 8.5,
    "properties": [
      { "name": "target_zone_id", "value": "realtime_preview" },
      { "name": "target_x", "value": 8.5 },
      { "name": "target_y", "value": 8.5 }
    ]
  }]
}
```

`nav.Map` validates:
- target zone ID non-empty/bounded;
- target coordinates finite/non-negative;
- portal position inside the map and not inside a solid tile.

The client parses the same metadata (`MapLoader`) and renders portal markers.
On generated region maps the marker labels the side; the actual trigger is the
whole border band and is resolved server-side (see below).

## Atomic process-local handoff

`Manager.TransferCharacter(lease, targetKey, spawn)`:
1. validates the source controller/token/lease;
2. rejects unknown target zones and target capacity overflow;
3. snapshots the source entity;
4. queues remove on source and waits for the zone owner tick;
5. queues add on target with the supplied spawn and waits;
6. rolls back to the source entity if target add fails;
7. updates the single owner lease to the target zone (same token/controller).

The old zone lease immediately rejects input because its `ZoneKey` no longer
matches. Tests assert no duplicate entity across zones, old-lease rejection,
target-lease movement and unknown-target errors.

## Edge-triggered region handoff (KHU_VUC/SAFE_ZONE)

Every generated child map is 2000×2000 and carries at most one portal per side.
The whole border band (not just the portal marker) is a crossing:

- The zone snapshot loop checks the owned player's authoritative position.
  `EdgeTriggerMargin = 2.5` units from a side with an authored portal resolves
  the destination zone from that side's portal.
- The arrival point is inset `EdgeArrivalInset = 4` units from the opposite
  border; the cross axis is preserved, so crossing near a corner keeps the
  player near the same corner.
- `Admission.TransferAtEdge` validates region entry (rank/Gu), flushes the source
  checkpoint/mana, writes `players.current_region`, clears `map_instance_id`,
  and persists `players.x/y` as the **legacy projection** of the destination
  map (`legacy_origin + local × legacy_units_per_world_unit`).
- It then releases the lease, deletes the source-zone checkpoint (so a later
  re-admission cannot restore the exact edge cell), and sends `world.transfer`
  with both coordinate pairs. The client reconnects with the new `zone_id`;
  `world.welcome` loads the destination map.
- Sides without an authored portal (Đại Vực rim) remain a wall.
- `MMO_ZONE_IDS=all` (or `*`) expands to every map in `MMO_MAP_DIR` whose
  `map_version`/`content_version` match the rollout config, so all leg targets
  are admittable without listing 125 ids. Preview/Alpha/legacy artifacts in the
  same directory are skipped.

`MapPortal` metadata (`portals` object group, property `edge`) stays the
authoritative adjacency source; the server never accepts a client-supplied
destination region.

## Still missing before PR-27 can be accepted

- Multi-process ownership/lease fencing and transfer rollback on target crash.
- Party follow across a border (the handoff is leader-only).
- Client loading overlay (the existing reconnect path swaps the map from
  `world.welcome` without creating a second Phaser Game).
