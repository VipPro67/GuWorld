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

The client parses the same metadata (`MapLoader`) and renders portal markers. The
sample map includes one marker targeting the same preview map; it is visual only
until the trigger/handoff wiring below is complete.

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

## Still missing before PR-27 can be accepted

- Automatic portal contact trigger and route selection.
- DB region/map-instance update and cross-zone admission (current admission
  requires the character's DB map to equal the requested zone).
- Gateway `world.transfer` request/response, client GameSession loading overlay
  and scene/map swap without creating a second Game/WS.
- Restarting snapshot subscriptions on the target zone (current subscription is
  bound to the source manager zone feed).
- Multi-process ownership/lease fencing and transfer rollback on target crash.
- Portal target spawn validation against the destination map (only source map is
  loaded at parse time).

This PR is intentionally **partial**: the authoritative single-process handoff
primitive exists, but portal-triggered end-to-end transfers are not enabled.
