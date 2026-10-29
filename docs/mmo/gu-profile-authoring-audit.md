# Realtime Gu profile authoring audit

## Current coverage

`transpiled_profiles.json` contains one catalog row for each of the 4,427 Gu
ranks. The current compile snapshot has 2,794 active skills, 1,261 passive-only
profiles, and 372 rows with neither an active nor a supported passive. Of the
rows with `ON_ACTION` source blocks, 300 currently have no active skill; 233 of
those contain only legacy `STATUS` operations. The source has 671 `STATUS`
operations overall, but only explicitly realtime-authored statuses can be
executed safely.

The catalog is therefore complete as an inventory, but active skill coverage
is not complete. A Gu with no `ON_ACTION` combat block (for example, a
capability or consume hook) must not be given an invented attack. `UNIQUE`
operations need a native realtime handler before they can become skills.

## General authoring template

Use one `(gu_id, rank)` row per rank. `mode` describes how the player aims the
cast; the effect describes what the server does. Keep timing in milliseconds,
range and geometry in world units, and effects in a list so a skill can combine
damage with a status or a defensive payload.

```json
{
  "gu_id": "GU_EXAMPLE",
  "rank": 1,
  "active": {
    "mode": "TARGET",
    "windup_ms": 250,
    "recovery_ms": 150,
    "cooldown_ms": 6000,
    "range_units": 9,
    "mana_cost": 5,
    "movement_policy": "SLOW",
    "max_targets": 1,
    "effects": [
      { "type": "DAMAGE", "value": 24 }
    ]
  },
  "visuals": {
    "icon": "gu/example",
    "cast_vfx": "cast/example",
    "impact_vfx": "impact/example"
  }
}
```

## Offensive Gu example

`GU_WATER_ARROW` is a direction/projectile skill. Its authored profile includes
aim mode, projectile speed/radius/lifetime, cost and timing, plus server-owned
damage. The client may preview the aim and VFX, but it does not decide hits,
damage, cooldown or resource cost.

```json
{
  "gu_id": "GU_WATER_ARROW",
  "rank": 1,
  "active": {
    "mode": "DIRECTION",
    "windup_ms": 200,
    "recovery_ms": 150,
    "cooldown_ms": 6000,
    "range_units": 16,
    "mana_cost": 5,
    "movement_policy": "SLOW",
    "projectile": {
      "speed_units_per_sec": 18,
      "radius_units": 0.35,
      "lifetime_ticks": 40
    },
    "effects": [{ "type": "DAMAGE", "value": 24 }]
  }
}
```

## Defensive Gu example

Use `SELF` for a self-shield/heal/cleanse. A defensive Gu that grants a
persistent transformation uses `form`; its reserved ability is authored as a
normal `active` profile under `form.ability`. Do not encode a self shield as a
hostile target skill.

```json
{
  "gu_id": "GU_WATER_SHIELD",
  "rank": 2,
  "active": {
    "mode": "SELF",
    "windup_ms": 100,
    "recovery_ms": 150,
    "cooldown_ms": 9000,
    "range_units": 0,
    "mana_cost": 6,
    "movement_policy": "SLOW",
    "effects": [{ "type": "SHIELD", "value": 60 }]
  }
}
```

## Sát Chiêu profile boundary

Sát Chiêu are composite skills assembled from their equipped Gu components;
they use slots 9–12 and the separate `KillerMove` contract. Keep their
component IDs, hunger costs, success/backfire values, and combination rules in
that contract. Do not duplicate a composite Sát Chiêu as a normal profile for
each component Gu. A component's own Gu skill and a Sát Chiêu that consumes it
are separate authored actions.

## Fields to add before claiming full per-Gu active coverage

The current profile contract has enough fields for the shipped single-target,
area, projectile, self-defense, and one-status examples. It still compresses
effects into scalar `Damage`, `Shield`, `Heal`, and one `Status` on
`combat.Skill`. Before broadening source coverage, extend the contract with:

- an explicit effect target (`SELF`, `ALLY`, `ENEMY`, `ALL_ALLIES`,
  `ALL_ENEMIES`) separate from cast aim mode;
- per-effect chance and ordering, so mixed blocks do not lose target or chance
  semantics;
- multiple timed statuses per cast, each with authored duration, tick period,
  stack policy, PvP diminishing-return category, and supported runtime
  behavior;
- typed resource effects (HP, mana, thought) so a legacy `HEAL` of `MANA` is
  not mistaken for HP healing or silently discarded;
- explicit support markers for native hooks, summons, forms, movement,
  cleanse targets, and conditional effects.

These fields require matching server simulation and validation. Adding fields
to JSON alone would create effects that appear castable but do nothing.

## Authoring rules

- Never convert legacy turns into milliseconds automatically. Author a
  realtime duration for each supported status profile.
- Never turn `UNIQUE`, world hooks, passive-only Gu, or unknown operations into
  generic damage as a fallback.
- Treat `primary_role` as classification metadata, not as a target or effect.
- A status should only be offered as a skill after its runtime behavior is
  implemented. Some existing status IDs are tags only; control, damage,
  movement, and resource behavior must be explicitly supported by the server.
- Preserve safe supported effects in a mixed action only when their targets and
  ordering remain unambiguous. Otherwise keep the action uncastable until it is
  reauthored.
