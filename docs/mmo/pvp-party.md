# Party, threat and PvP rules (PR-28, partial)

## Zone ruleset

The shared map JSON can set a top-level property:

```json
{ "name": "ruleset", "value": "pve" }
```

Accepted values: `pve` (default), `pvp`, `safe`. `nav.Map` validates the value;
the manager enables player-vs-player damage only for `pvp` maps via
`sim.World.SetPvPEnabled`.

## Player harm rules

`sim.canHarm`:
- player ↔ monster: allowed;
- player → player:
  - denied outside a `pvp` zone;
  - denied when both players have the same non-empty `PartyID`;
  - allowed for different parties (or both party-less) inside `pvp`.

Targeted support skills use `canApplySkill`: heals/shields/cleanse may target
self or a same-party ally, not enemies. Offensive damage/status uses `canHarm`.
CC duration DR still applies for player→player hard/soft CC.

`players.party_id` is loaded at join into `Entity.PartyID`; snapshots include
`party_id` so the client can label/tint party members. Live party changes while
inside the zone are not refreshed yet (rejoin needed).

## Boss threat

- Damage applied to an active monster records `ThreatTable[source] += actual damage`.
- AI selects the highest living threat within leash, with deterministic ID
  tie-break; if no threat remains it falls back to nearest player in aggro range.
- Leash/wipe/respawn clears the threat table.
- Boss archetype (`ai.DefaultStats("Boss")`) carries `SplashRadius=4` and
  `SplashDamage=10`; an attack also damages other players near the primary
  target, emitting `HIT_CONFIRMED` with `reason=splash`.

## Tests

- PvP allowed only in pvp zones with different parties; same-party/PDVE cast is
  rejected.
- Boss selects the top threat and splash hits a nearby party member.
- Leash clears threat.

## Not implemented

- Explicit duel consent/arena queue; PvP rules are zone-wide.
- Party creation/accept UI inside MMO (legacy Party panel is mounted in PR-24,
  but membership changes do not hot-apply).
- Party-shared loot, loot master, party markers beyond the entity label/tint.
- Boss AoE telegraph/VFX, enrage, wipe rewards, arena world instance.
- Safe-zone combat flag is ruleset-only; no per-region sub-zones yet.
- PvP damage scaling/rating and death/respawn consequences.
