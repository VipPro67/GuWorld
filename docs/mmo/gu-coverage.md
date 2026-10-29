# Realtime Gu coverage (PR-17)

Generated from `backend/data/Gu.csv` by:

```bash
cd backend
go run ./cmd/gudata audit-realtime \
  -data data \
  -csv ../docs/mmo/gu-coverage.csv \
  -strict
```

The CSV has one row per effect block and is the authoritative inventory. The
JSON report can be generated on demand with `-report <path>`; it is not
committed to keep the repository small. See `docs/mmo/gu-realtime-schema.md` for
the authored profile schema and compiled MVP artifact.

## Effect audit snapshot (2026-10-25)

| Status | Entries |
| --- | ---: |
| `SUPPORTED` | 3374 |
| `NEEDS_REAUTHOR` | 871 |
| `UNSUPPORTED` | 0 |
| `NOT_ACTIVE` | 182 |
| **Total** | **4427** |

- 5,021 blocks / 6,380 operations audited.
- `unclassified = 0`: every trigger/operation is explicitly handled.
- `UNSUPPORTED = 0` at block level because every current block passes the strict
  Block v2 validator. Unknown/removed operations would fail this check instead
  of being silently skipped.

## Realtime cast-profile snapshot (2026-10-26)

`audit-realtime` classifies effect vocabulary; it does **not** mean an entry is
already usable on the MMO hotbar. The transpiler supports flat damage, flat
self shield/heal, self cleanse, and explicitly authored profile overrides.
When a safe action also contains a legacy `STATUS`, it preserves the safe
portion and omits the status; it never infers realtime durations from turns.
Conditional and other unsupported operations still keep the action out of the
hotbar. Domain/world hooks can remain beside a supported combat action.

Current catalog: 4,427 profile records, 2,794 active profiles, 1,261
passive-only profiles, and 372 entries without a castable hotbar skill. The
nine authored overrides preserve the five preview hotbar profiles and the
additional test/form profiles.
The active-profile coverage test checks every `ON_ACTION` entry in `Gu.csv`
against the transpiler rules.

Regenerate and compile the catalog with:

```bash
cd backend
go run ./cmd/gudata transpile-realtime-gu \
  -data data \
  -overrides data/mmo/gu/authored_overrides.json \
  -output data/mmo/gu/transpiled_profiles.json \
  -compile-report /tmp/gu-compiled-profiles.json
go run ./cmd/gudata compile-realtime-gu \
  -data data \
  -profiles data/mmo/gu/transpiled_profiles.json
```

Actions that combine safe damage/heal/shield effects with a legacy `STATUS`
remain castable with the safe effects only. The status is omitted until its
duration and stacking behavior are authored for realtime. Status-only actions,
conditional actions, and unsupported operations remain visible in inventory
but cannot be cast until their realtime behavior is explicitly implemented.

## Six MVP Gu

| Slot role | Gu | Rank | Status | Required reauthor |
| --- | --- | ---: | --- | --- |
| Damage / projectile | `GU_WATER_ARROW` | 1 | `NEEDS_REAUTHOR` | `COL` formation → `DIRECTION` line/cone or bound `TARGET` |
| Area damage | `GU_SPIRAL_HAMMER_2` | 2 | `SUPPORTED` (maps to `GROUND` AoE) | radius/target cap |
| Defense | `GU_WATER_SHIELD` | 2 | `NEEDS_REAUTHOR` | remove formation target; keep `SELF` operation target |
| Control | `GU_WHITE_FROST` | 3 | `NEEDS_REAUTHOR` | `STATUS FREEZE`: durationMs/periodMs/stack/ICD/PvP DR |
| Heal | `GU_WATER_CLEANSING` | 4 | `SUPPORTED` | instant self heal is realtime-safe |
| Passive | `GU_STRENGTH` | 1 | `SUPPORTED` | apply passive stats on loadout/equip |

Hidden passive: **no Gu block in the current CSV uses conditions or stack
counters**, so a hidden passive cannot be selected honestly from existing data.
PR-19 must author one (for example HP threshold + 3 consecutive hits + ICD) and
add it to the MVP list in `DefaultMVP()`.

## Classification rules

- `SUPPORTED`: `ON_ACTION`/`ON_CAST`/`ON_HIT`/`ALWAYS_ON`/`PASSIVE`/
  `COMBAT_PASSIVE`, target `SELF`/`TARGET`/`ALL*`, operations
  `DAMAGE`/`SHIELD`/`HEAL`/`DIRECT_HP`/`CLEANSE` and simple `STAT_MOD`.
- `NEEDS_REAUTHOR`:
  - formation targets `ROW`/`COL`;
  - turn-based triggers (`ON_TURN_START`, `ON_BATTLE_START`,
    `BEFORE_ENEMY_ATTACK`, `ON_DEATH`, `BEFORE_HIT`, `ON_TICK`);
  - any `STATUS`, `UNIQUE`, growth-cap `STAT_MOD`, timed/stacked `BARRIER`;
  - blocks with conditions.
- `NOT_ACTIVE`: world/domain triggers (`CAPABILITY`, `ON_CONSUME`, `ON_USE`);
  they are not part of zone combat.
- `UNSUPPORTED`: invalid Block v2, unknown trigger/operation, or missing native
  hook. The report fails the audit instead of pretending the effect works.

Proposals never convert turns to milliseconds by default. Timed statuses must be
authored explicitly (`durationMs`, `periodMs`, stack/refresh, PvP DR), and
`UNIQUE` codes require versioned Go handlers with deterministic tests.

## Alpha catalog (PR-35)

`backend/data/mmo/gu/transpiled_profiles.json` includes 40 compiled, catalog-only
(`slot: 0`) profiles selected from `SUPPORTED`-compatible low-rank blocks:

| Role | Count | Mode |
| --- | ---: | --- |
| Single-target damage | 10 | `TARGET` |
| Area damage | 10 | `GROUND` |
| Self heal | 10 | `SELF` |
| Self shield | 10 | `SELF` |

Selection rules: exactly one `ON_ACTION` block, only `DAMAGE`/`SHIELD`/`HEAL`,
flat values only, rank 1–4, value ≤ 300, and no MVP overlap. This intentionally
avoids pretending that `STATUS`, formation, turn-based or `UNIQUE` blocks are
playable. Those stay in `NEEDS_REAUTHOR` until an explicit schema/native handler
exists.

Combined compiled hash (MVP + Alpha):
`fedc583f0672f48bb08396435c1ad01d42ca4c72d1774f2551c656c6489125d9`.
Startup and CI must use the comma-separated profile list; loading only the MVP
file is still valid for the six-slot preview bar.

The combined catalog also includes authored MMO-scale profiles for
`GU_HEALING_BELL_2`, `GU_IMMORTAL_DRAGON_SLAYING_SWORD`, and
`GU_STRENGTH_INFUSION`. Their source effects include turn-based FEAR or
outsize damage, so the realtime profiles deliberately omit FEAR and re-author
the dragon Gu's damage/status numbers for the current runtime budget.

## Known gaps

- Hidden passive not authored.
- `STATUS` semantics/status registry mapping still needs the PR-19 effect engine.
- `GU_WATER_SHIELD` is conservatively flagged because its block target is `COL`
  even though the shield operation overrides target to `SELF`.
- The audit does not yet validate VFX/icon assets; that is PR-18/PR-21.
