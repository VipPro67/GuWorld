# Realtime Gu authoring schema v1 (PR-18)

At world admission, the server keeps the profile's MMO-scale base damage but
projects the character's path attainment, Dao amplification/conflict, Gu
damage multiplier and path bonus onto it. Cast cost comes from the legacy
`gamefx.EnergyCost` rule plus Dao conflict tax, rather than the profile's
authored preview cost. Basic attacks use passive player attack. Targets apply
passive defense, natural Dao resistance and path resistance before the
minimum-hit floor. Rank-higher Gu inflict the legacy HP backlash. The zone
does not grant passive mana regeneration during combat; `PlayerService`
projects elapsed world-time recovery on admission. Checkpoints settle spent
mana to the player row every 10 seconds and when leaving. Checkpoint and mana
settlement share one database transaction; a failed disconnect remains owned
for a later retry. A database outage lasting through process shutdown is
logged as an unsettled session.

Live hotbar slots also carry the owned `player_gu.id` and its admission-time
hunger. Each valid cast attempt spends one hunger before backlash and mana
validation, matching the realtime combat runtime; an exhausted instance cannot cast.
The zone counts uses per slot, and the same checkpoint transaction subtracts
the per-instance usage delta from `player_gu.hunger`. A failed checkpoint keeps
the session and its unsettled counters for retry. The preview hidden passive is
restricted to preview combat states and is not granted to live characters.

Cooldowns, timed statuses and hidden-passive counters are checkpointed in
`player_world_state_combat` by the same transaction that writes position, HP,
mana and hunger. Remaining durations are stored in zone ticks, so a rejoin or
border transfer continues the exact timer instead of resetting it. Cooldowns are
only restored when the same authored skill still occupies the hotbar slot; a
dead character clears its restorable combat state. After a checkpoint that
spent hunger, the server publishes a `gu_inventory` player event so the client
refetches the committed inventory.

`movement_policy` runs during windup and recovery: `SLOW` halves ground speed,
`ROOT` stops it, and `FREE` (or the empty legacy default) leaves it unchanged.
`recovery_ms` blocks the next cast and basic attack after the cast resolves,
even if the global cooldown has already expired. Interrupted casts do not enter
recovery.

Source profile catalog: `backend/data/mmo/gu/transpiled_profiles.json` (unified catalog covering all Gu, with hand-authored MVP and Alpha profiles integrated). Compiled artifact:
`docs/mmo/gu-compiled.json` (regenerate, do not hand-edit).

```bash
cd backend
go run ./cmd/gudata compile-realtime-gu \
  -data data \
  -profiles data/mmo/gu/transpiled_profiles.json \
  -report ../docs/mmo/gu-compiled.json
```

The compiler validates the authored profile against `Gu.csv`, the combat
contract and the Block v2 source. Invalid/ambiguous profiles fail with a path
and reason; they are never loaded as a “skill without effects”.

## Profile shape

`visuals` is optional presentation metadata; it never changes simulation rules.
The generated catalog infers reusable cast and impact families from the Gu path,
name and active effects. `projectile_vfx` is emitted only when the active profile
has projectile geometry. The frontend maps these symbolic recipe keys onto the
existing VFX atlas catalog, and non-Gu skills use the same name-based fallback.
An authored non-placeholder value overrides the inferred family; legacy generic
placeholder values are ignored.

```json
{
  "schema_version": 1,
  "profiles": [
    {
      "gu_id": "GU_WATER_ARROW",
      "rank": 1,
      "slot": 2,
      "active": {
        "mode": "DIRECTION",
        "windup_ms": 200,
        "recovery_ms": 150,
        "cooldown_ms": 6000,
        "range_units": 16,
        "mana_cost": 5,
        "movement_policy": "SLOW",
        "projectile": {"speed_units_per_sec": 18, "radius_units": 0.35, "lifetime_ticks": 40},
        "effects": [{"type": "DAMAGE", "value": 24}]
      },
      "passives": [],
      "hidden_passives": [],
      "visuals": {
        "icon": "gu/water_arrow",
        "cast_vfx": "cast/water",
        "projectile_vfx": "projectiles/water_arrow",
        "impact_vfx": "impact/water"
      }
    }
  ]
}
```

- `mode`: `TARGET`, `DIRECTION`, `GROUND`, `SELF`.
- `windup_ms`/`recovery_ms` ≤ 5,000 ms; `cooldown_ms` ≤ 120,000 ms. Durations
  are converted to ticks with ceiling at 20 Hz (`ms*20/1000`, minimum 1 tick).
- `movement_policy`: `SLOW` (half ground speed), `ROOT` (no ground movement) or
  `FREE`; an omitted value keeps legacy unrestricted movement. The zone applies
  it during windup and recovery, never as a turn conversion.
- `range_units` uses world units (1 cell = 1 unit). `DIRECTION`/`GROUND`/
  `TARGET` need range > 0; `SELF` uses 0.
- `projectile` is required for `DIRECTION`: speed > 0, radius > 0,
  lifetime 1–120 ticks.
- `GROUND` needs `radius_units` and `max_targets` 1–100.
- Active effects: `DAMAGE`, `SHIELD`, `HEAL`, `STATUS`, `CLEANSE`.
  - `STAT_MOD` is accepted on passives only; an active `STAT_MOD` is rejected
    until a timed-buff runtime exists, rather than advertising a no-op skill.
  - `STATUS` needs `status_id`, `duration_ms` 100–30,000 and
    `pvp_dr_category`; turns are **not** accepted or auto-converted.
  - Optional STATUS fields: `period_ms` (DoT/HoT interval, 100–10,000),
    `tick_value` (damage when `debuff=true`, heal when false),
    `move_multiplier` (0–1; 0 blocks movement/casting), `max_stacks` (1–20),
    `stack_policy` (`REPLACE`/`REFRESH`/`STACK`/`MAX`/`IGNORE`), `debuff`.
  - `CLEANSE` removes debuff statuses.
  - `UNIQUE` is rejected by the compiler (native handler required, PR-19/35).
- Passives: trigger `ALWAYS_ON`/`PASSIVE`/`COMBAT_PASSIVE`, effects `STAT_MOD`.
- `slot` is required for an active profile (1–6, unique) and omitted/0 for
  passive-only or catalog-only Gu.
- A defensive transformation is authored separately under `form`. Its activation
  and in-form ability occupy reserved realtime slots 7 and 8; they are generated
  from the equipped Gu profile and are not part of the player's six Gu hotbar
  assignments. `form` contains `windup_ms`, `mana_cost`, `mana_drain_per_sec`,
  `defense_bonus`, and an `ability` object using the normal active-profile
  shape.
- `slot: 0` active profiles are compiled but not placed on the default bar; the
  join-time owned-Gu adapter re-slots them from the player's equipment (PR-23).
  This is how the 40 Alpha catalog profiles ship without overriding the MVP
  preview bar.

## Committed MVP catalog

| Slot | Gu | Mode | Windup/cooldown (ticks) | Payload |
| ---: | --- | --- | --- | --- |
| 2 | `GU_WATER_ARROW` | `DIRECTION` | 4 / 120 | 24 damage, projectile 18 u/s |
| 3 | `GU_SPIRAL_HAMMER_2` | `GROUND` | 7 / 160 | 36 damage, r=2.5, max 5 |
| 4 | `GU_WATER_SHIELD` | `SELF` | 2 / 180 | 60 shield |
| 5 | `GU_WATER_CLEANSING` | `SELF` | 4 / 240 | 400 heal |
| 6 | `GU_WHITE_FROST` | `TARGET` | 5 / 200 | `FREEZE` 1.5s, HARD_CC |
| — | `GU_STRENGTH` | passive | — | 5× `STAT_MOD` |
| — | `GU_HEALING_BELL_2` | `TARGET` | 4 / 120 | 60 damage (FEAR chưa có hành vi realtime) |
| — | `GU_IMMORTAL_DRAGON_SLAYING_SWORD` | `TARGET` | 7 / 280 | 120 damage + BLEED 3s, 12 damage/s |
| — | `GU_STRENGTH_INFUSION` | `TARGET` | 5 / 180 | 45 damage + 45 shield |

Compiled MVP-only catalog hash:
`8552ec10bb9b50b6267257248e496bbfc6ab5fc60a140ca55bac0b683711586a`.

Compiled combined catalog hash (MVP + 40 Alpha, the default startup set):
`fedc583f0672f48bb08396435c1ad01d42ca4c72d1774f2551c656c6489125d9`.

The generated Alpha catalog contains 40 low-rank profiles. Authored profiles
cover supported equipped Gu using MMO-scale values, so high-rank Gu do not
inherit their turn-based damage literally. `GU_BEAST_BODY_I` also authors a
defensive form with a separate ability. FEAR remains unsupported. Other
higher-rank and `NEEDS_REAUTHOR` Gu remain covered by
`docs/mmo/gu-coverage.csv`.

`GU_WATER_CLEANSING`'s 400 heal is the authored Gu value; the zone still caps
at max HP. The zone executes tick-scheduled statuses (DoT/HoT, stacks,
refresh/ICD, movement blocking) and supported hidden passives. Compiled Gu
profiles are attached from each player's equipped loadout at join.

## Per-player loadout (join adapter)

At join the gateway loads `players.equipped_gus` and `player_gu` rows where
`equipped=true AND state='ACTIVE'`, orders them by `players.equipped_gus`, and
matches each instance to a compiled profile by `(gu_id, rank)`. Matched actives
are re-slotted 1..6 in equipped order; unsupported Gu are skipped and reported
in the build result. If no compiled Gu remains, the server preview hotbar is
used rather than sending a partial/empty skill set.

The immutable loadout and SHA-256 `loadout_revision` are cached in the session;
the zone never queries the DB per cast. Form activation and ability profiles
are appended to reserved slots 7 and 8, with the ability only castable while
the server reports the transformed state active. Transformation windup can be
interrupted by hard crowd control; ordinary damage does not interrupt it.
Defense bonus and mana drain begin after completion, and the form ends when
manually toggled off or when mana can no longer pay its per-tick drain. Form
state is runtime-only and resets on rejoin. Live equipment changes while inside
a zone require a rejoin (refresh command is future work).

## Startup validation

`MMO_GU_PROFILES` (default
`data/mmo/gu/transpiled_profiles.json`) is loaded and
compiled at startup when the realtime world is enabled. The value is a
comma-separated catalog list; catalogs are merged in order. Success logs
`profiles`, `actives` and `hash`; failure logs the path/reason and keeps the
preview loadout rather than running a partially compiled skill set.

## Still not covered

- Applying compiled passives to player stats.
- Native handlers for `UNIQUE` effects / `NEEDS_REAUTHOR` Gu.
- Per-Gu hidden passives beyond the preview one.
- VFX asset existence validation for Alpha icons/cast/impact placeholders.
