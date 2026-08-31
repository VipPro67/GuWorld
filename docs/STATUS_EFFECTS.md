# GuWorld - Status Effects Reference

Comprehensive documentation of all effects, statuses, and their mechanics in GuWorld.

---

## 1. Core Effect Structures

### `gamefx.Effects` (Character Aggregated Effects)
Central struct aggregating modifiers from Race, Constitution, and Active Statuses.

```go
type Effects struct {
    StatMods              map[string]float64  // "attack", "defense", "speed" → % modifier
    RefineBonus           float64             // Additive success-rate bonus for refining (0..1)
    DaoNganBonus          map[string]float64  // Per-dao-path additive multiplier
    PathDamageBonus       map[string]float64  // Per-path % damage bonus
    PathResistanceBonus   map[string]float64  // Per-path % damage reduction taken
    LuckBonus             float64             // Flat luck increase
    MaxManaBonus          float64             // % max mana increase
    ManaRegenBonus        float64             // % mana regen increase
    CultivationEfficiencyBonus float64        // % cultivation gains increase
    DaoMarkGainBonus      float64             // % Đạo Ngân earned increase
    HealingReceivedBonus  float64             // % healing received increase
    HPRegenBonus          float64             // Flat HP/hour bonus (out-of-combat)
    MentalResistanceBonus float64             // % resistance to control/mental effects
    SoulDefenseBonus      float64             // % soul damage reduction
    SoulRegenBonus        float64             // % passive soul_hp regen increase
    ApertureStressBonus   float64             // Additive multiplier on aperture stress
    BlockedActions        map[string]bool     // "explore", "cultivate", "combat", "refine", "trade"
}
```

### `models.GuEffects` (Gu-Specific Effects)
```go
type GuEffects struct {
    Active  *ActiveEffect     // Combat-active effects
    Passive *PassiveEffect    // Permanent stat bonuses
    Growth  *GrowthEffect     // Growth over time
    Other   []SpecialMechanic // Event-driven mechanics (ON_TICK, ON_CONSUME, ON_DEATH, ON_USE)
}
```

### `simulator.Effect` (Combat Runtime)
```go
type Effect struct {
    ID       string // Effect constant
    Type     string // "buff" or "debuff"
    Tier     int    // 1-5 for scaling
    Value    int    // Computed percent/duration
    Duration int    // Remaining turns
}
```

---

## 2. Gu Active Effects (44 keys in data)

### Damage & Healing
| Key | Type | Description | Count |
|-----|------|-------------|-------|
| `damage_flat` | number | Fixed damage | 207 |
| `heal_flat` | number | Direct heal | 33 |
| `shield_flat` | number | Shield/barrier | 96 |
| `damage_mult` | number | Damage multiplier (×1.075–1.35) | 18 |

### Status Application
| Key | Type | Description |
|-----|------|-------------|
| `apply_status` | enum | Status: BLEED, POISON, STUN, SLOW, FREEZE, SILENCE, THORNS, REGEN, ATK_UP, DEF_UP, TAUNT, WEAK, SLOW1, SLOW_DAMAGE, BURN, FEAR, CONFUSE, STEALTH, SLEEP, BLIND, CHARM, BIND |
| `status_tick_damage` | number | DoT damage per tick / status intensity |
| `status_duration_ticks` | int | Duration in turns |

### Targeting & Special
| Key | Type | Description | Count |
|-----|------|-------------|-------|
| `target_selector` | enum | Targeting: COL, ALL, ROW, SINGLE | 55 |
| `target_count` | int | Number of targets | |
| `blood_drain` | bool | +50% lifesteal | |
| `summon_phantom` | bool | Summon phantom (dmg = BaseAttack) | |
| `range` | number | Attack range | |
| `curve_attack` | bool | Bypass frontline | |
| `mana_regen` | number | Instant mana restore | |
| `refine_mana_tier` | number | Refine mana tier | |
| `cleansing_power` | number | Cleansing level | |
| `capture_bonus` | number | Monster catch rate bonus | |
| `replicate` | bool | Replicate effect | |
| `wood_transformation` | bool | Wood transformation | |
| `portal_create` | bool | Create portal | |
| `random_portal` | bool | Random portal | |

---

## 3. Gu Passive Effects (32 keys in data)

### Base Stats
| Key | Type | Description | Count |
|-----|------|-------------|-------|
| `stat_base_attack` | number | Permanent attack | 151 |
| `stat_base_defense` | number | Permanent defense | 151 |
| `stat_base_hp` | number | Permanent HP (via EffectiveMaxHP) | 15 |
| `stat_base_speed` | number | Permanent speed | 7 |

### Utility
| Key | Type | Description | Count |
|-----|------|-------------|-------|
| `mana_regen_mult` | number | Mana regen multiplier | |
| `path_conflict_reduction` | number | Dao path conflict reduction | 40 |
| `damage_multiplier_on_hit` | number | Damage mult on hit | |
| `soul_refinement` | number | Soul refinement | |
| `refine_mana_tier` | number | Refine mana tier | 9 |
| `luck` | number | Luck | |
| `time_speed_multiplier` | number | Time speed bonus | 1 |
| `evasion` | number | Evasion | |
| `control_power` | number | Control power | |
| `storage_capacity` | number | Storage capacity | |
| `success_rate_bonus` | number | Success rate bonus | 13 |
| `movement` | number | Movement | 27 |

---

## 4. Special Mechanic Action Codes (20 types)

| Code | Trigger | Description |
|------|---------|-------------|
| `MANA_REGEN_TICK` | ON_TICK | Mana regen per tick |
| `THOUGHT_REGEN_TICK` | ON_TICK | Thought regen per tick |
| `SOUL_REGEN_TICK` | ON_TICK | Soul regen per tick |
| `GENERATE_STONES_TICK` | ON_TICK | Generate stones per tick |
| `GENERATE_IMMORTAL_STONES_TICK` | ON_TICK | Generate immortal stones per tick |
| `ADD_LIFESPAN_YEARS` | ON_TICK | Add lifespan years |
| `RANK_PROGRESS` | ON_TICK | Rank progression |
| `SACRIFICE_APTITUDE` | ON_TICK | Sacrifice aptitude |
| `ADD_SOUL_MAX` | ON_TICK | Add soul max HP |
| `SOUL_BOOST` | ON_TICK | Soul boost |
| `RESTORE_SOUL_HP` | ON_TICK | Restore soul HP |
| `INCREASE_TIME_SPEED` | ON_TICK | Increase time speed |
| `REDUCE_STRESS` | ON_TICK | Reduce stress |
| `RESTORE_MENTAL` | ON_TICK | Restore mental |
| `RECORD` | ON_TICK | Record |
| `CONTRACT_OATH` | ON_TICK | Contract oath |
| `PORTAL_CREATE` | ON_TICK | Portal create |
| `REGENERATE_TERRAIN` | ON_TICK | Regenerate terrain |
| `HOT_TICK` | ON_TICK | Heal over time tick |
| `ENSLAVE_BEAST_TIER` | ON_TICK | Enslave beast tier |

---

## 5. Combat Status Effects

### Legacy Tier-Based (14 constants)

| Constant | Type | Resistable | Description |
|----------|------|------------|-------------|
| `EffectSlow` | debuff | MentalResistance | Reduces speed |
| `EffectSlow1` | debuff | **No** | Reduces speed (guaranteed) |
| `EffectSlowDamage` | debuff | MentalResistance | Reduces speed + deals damage/turn |
| `EffectWeak` | debuff | MentalResistance | Reduces damage dealt |
| `EffectFreeze` | control | MentalResistance | Like stun, breakable by damage (50%) |
| `EffectSilence` | control | MentalResistance | Prevents skills, only basic attack |
| `EffectThorns` | buff | No | Reflects damage when hit |
| `EffectRegen` | buff | No | Heals HP each turn |
| `EffectAtkUp` | buff | No | Increases attack |
| `EffectDefUp` | buff | No | Increases defense |
| `EffectTaunt` | buff | No | Forces enemies to attack |
| `EffectStun` | control | MentalResistance | Skips turn entirely |
| `EffectBleed` | DoT | No | Damage over time |
| `EffectPoison` | DoT | No | Damage over time |

### Registry-Driven Statuses (18 catalog)

| Status ID | Behaviour | Resistable | Description |
|-----------|-----------|------------|-------------|
| `BLEED` | DOT | No | Damage over time |
| `POISON` | DOT | No | Damage over time |
| `BURN` | DOT | No | Fire DoT |
| `FEAR` | force_basic | MentalResistance | Only basic attack allowed |
| `CONFUSE` | confuse | MentalResistance | 40% chance to mis-target ally |
| `STEALTH` | stealth | No | Untargetable by single-target, breaks on attack |
| `SLEEP` | sleep | MentalResistance | Skip turn, breaks on any damage |
| `BLIND` | blind | No | 35% miss chance |
| `CHARM` | charm | MentalResistance | Attack allies |
| `BIND` | bind | MentalResistance | Skip turn |
| `HASTE` | haste | No | Speed up |
| `ARMOR_BREAK` | armor_break | No | Defense down |
| `DAMAGE_REDUCTION_UP` | damage_reduction | No | Reduce damage taken |
| `INVINCIBLE` | invincible | No | Immune to damage |
| `REVIVE_LOCK` | revive_lock | No | Prevent revival |
| `STUN` | (legacy) | MentalResistance | Skip turn |
| `SLOW` | (legacy) | MentalResistance | Speed debuff |
| `SILENCE` | (legacy) | MentalResistance | No skills |

---

## 6. Character Effects Sources

### Races (chủng tộc) - 10 total
Loaded from `race_dictionary` (seeded from `Race.csv`):
- `human` — no bonus (default)
- `fur_people`, `merman`, `feathered`, `inkman`, `rockman`, `snowman`, `variant_beastman`, `dragonman`, `giant_sun_descendant`

Effects granted: StatMods, RefineBonus, DaoNganBonus, PathDamageBonus, PathResistanceBonus, LuckBonus, CultivationEfficiencyBonus

### Constitutions (thể chất) - 11 total (10 Thập Tuyệt + 1 Mộng)
Loaded from `constitution_dictionary` (seeded from `Constitution.csv`):
- All Tier 3
- Each grants: DaoNganBonus (path affinity ×1.5), MaxManaBonus (+40-50%), ManaRegenBonus (+100%)
- Plus unique: StatMods, PathDamageBonus, PathResistanceBonus, MentalResistanceBonus, HealingReceivedBonus

### Statuses (trạng thái) - 5 total
Loaded from `status_dictionary`:
- `severely_wounded`, `poisoned`, `exhausted`, `blessed`, `death_penalty`
- Effects: StatMods, HealingReceivedBonus, BlockedActions, CultivationEfficiencyBonus

---

## 7. Effect Resolution Flow

### Character Effects Aggregation
```go
func Get(p *models.Player) Effects {
    fx := Effects{...}
    // 1. Race effects
    if def, ok := effectiveRaces()[p.Race]; ok {
        addEffects(&fx, def.Effects)
    }
    // 2. Active status effects
    for _, status := range p.Statuses.Active() {
        if def, ok := effectiveStatuses()[status.ID]; ok {
            addEffects(&fx, def.Effects)
        }
    }
    // 3. Constitution effects
    if def, ok := effectiveConstitutions()[p.Constitution]; ok {
        addEffects(&fx, def.Effects)
    }
    return fx
}
```

### Additive Merging
All numeric fields **add together** across Race + Constitution + Statuses.

### Combat Integration
- `DaoMultiplierForPaths()` - Computes dao mark damage multiplier per path
- `PathDamageMultiplierForPaths()` - Computes path damage bonus
- `EffectiveMaxHP()` - Includes passive Gu HP + buff_hp status
- `EffectiveMaxMana()` - Includes MaxManaBonus from constitution

---

## 8. Combat State Tracking

### CombatantAuto Fields
```go
ActiveEffects []Effect        // Buffs/debuffs
ActiveDoT     []DoTEffect     // Damage/heal over time

// Derived stats (computed each turn)
EffectiveSpeed        float64
DamageMultiplier      float64  // from weak/atk_up
DefenseMultiplier     float64  // from def_up
ThornsDamage          float64  // % reflect
IsSilenced, IsFrozen, IsSleeping, IsFeared, IsCharmed, IsStealthed,
IsConfused, IsBlinded, IsBound, IsReviveLocked bool
AccuracyPenalty       float64  // BLIND miss chance
DamageTakenReduction  float64  // DAMAGE_REDUCTION_UP / INVINCIBLE
```

### Turn Processing
1. **Tick DoTs** (bleed, poison, regen)
2. **Tick status durations** + compute multipliers
3. **Apply actions** with all multipliers

---

## 9. Damage Formula (Simulator)

```go
hitDmg = a.Damage * dmgMultiplier * daoMult * (1.0 + pathBonus(actor.PathDamageBonus, a.Path))
netDmg = hitDmg - targetDef
netDmg *= (1.0 - pathBonus(target.PathResistance, a.Path))
if netDmg < 1 { netDmg = 1 }
```

### Multipliers Applied

| Source | Multiplier |
|--------|------------|
| `sword_dragon_form` buff | ×1.50 |
| `borrow_past_power` buff | ×1.40 |
| `Weak` debuff | ×(1 - weak%) |
| `AtkUp` buff | ×(1 + atk_up%) |
| `GU_SOVEREIGN_FETUS` equipped | ×2.0 |
| Dao marks | `DaoMultiplierForPaths()` |
| Path damage bonus | `PathDamageMultiplierForPaths()` |
| Target path resistance | `1.0 - PathResistance` |

---

## 10. Status Resistance

| Status | Resistance Check |
|--------|------------------|
| STUN, FREEZE, SILENCE, SLOW, WEAK, FEAR, CONFUSE, SLEEP, CHARM, BIND | `rng.Float64() < MentalResistance` |
| FREEZE break | 50% chance on taking damage |
| SLEEP break | Breaks on any damage |
| STEALTH break | Breaks on attack |
| SLOW1 | **Cannot be resisted** |

---

## 11. Effect Storage & Serialization

| Location | Format | Notes |
|----------|--------|-------|
| `players.race` | VARCHAR FK → `race_dictionary.key` | Effects in JSONB |
| `players.constitution` | VARCHAR FK → `constitution_dictionary.key` | Effects in JSONB |
| `players.statuses` | JSONB array | `{"id", "expires_at", "source", "value"}` |
| `gu_dictionary.effects` | JSONB → `models.GuEffects` | Active/Passive/Growth/Other |
| `item_dictionary.effects` | JSONB → `models.ItemEffects` | Flexible map[string]float64 |
| `immortal_apertures.dao_marks` | JSONB map[string]int | Per-path Đạo Ngân |
| `players.path_masteries` | JSONB map[string]int | Per-path mastery |
| `players.dao_marks` | JSONB map[string]int | Player Đạo Ngân |

---

## 12. Adding New Effects

### Character Effect (Race/Constitution/Status)
1. Add field to `gamefx.Effects` struct (`fx.go`)
2. Update `addEffects()` merger (`fx.go:106-148`)
3. Add to CSV (`Constitution.csv` / `Race.csv` / `Status.csv`)
4. Admin Panel → Data tab → Import table
5. Restart backend (registries cached in memory)

### Gu Active Effect
1. Add field to `models.ActiveEffect` (`gu.go`)
2. Update `normalize()` for shorthand status mapping
3. Update `BuildCombatAction()` in `handlers/gu_effects.go`
4. Update simulator if new mechanic

### Combat Status
1. Add constant to `statusfx/status.go`
2. Add entry to `statusfx.Registry` in `statusfx/spec.go` with `StatusSpec`
3. Add behaviour hook in `simulator/status_apply.go`
4. Add to `handlers/combat_attack.go` for manual combat

### Special Mechanic Action Code
1. Add case to `handleActionCode()` in `handlers/gu_effects.go`
2. Define params structure
3. Add to Gu CSV with `other` array entry

---

## 13. Key Files Reference

| File | Purpose |
|------|---------|
| `backend/internal/gamefx/fx.go` | Core Effects, Get(), Dao/Path multipliers |
| `backend/internal/gamefx/statuses.go` | StatusDef, Registry, LoadStatusRegistry |
| `backend/internal/gamefx/constitutions.go` | ConstitutionDef, Registry, defaults |
| `backend/internal/gamefx/races.go` | RaceDef, Registry, defaults |
| `backend/internal/gamefx/energy.go` | EnergyCost, mana calculation |
| `backend/internal/gamefx/path_mastery.go` | Path mastery levels/multipliers |
| `backend/internal/statusfx/spec.go` | StatusSpec, Registry, ResolveStatus |
| `backend/internal/statusfx/status.go` | Status IDs, Behaviour constants |
| `backend/internal/models/gu.go` | GuEffects, ActiveEffect, PassiveEffect, GuRegistry |
| `backend/internal/models/item.go` | ItemEffects (flexible map) |
| `backend/internal/simulator/models.go` | Effect, DoTEffect, CombatantAuto, CombatState |
| `backend/internal/simulator/engine.go` | Simulate(), tickDoTs, tickStatusEffects |
| `backend/internal/simulator/status_apply.go` | applyStatusOps, registry-driven status |
| `backend/internal/handlers/combat_effects.go` | CalculateGuEffectForCombatant |
| `backend/internal/handlers/gu_effects.go` | BuildCombatAction, status mapping |
| `docs/GAME_DATA.md` | Complete data documentation |

---

## 14. Data Files (CSV → DB Import via Admin)

| CSV | Table | Effect Column |
|-----|-------|---------------|
| `Gu.csv` | `gu_dictionary` | `effects` JSONB |
| `Constitution.csv` | `constitution_dictionary` | `effects` JSONB |
| `Race.csv` | `race_dictionary` | `effects` JSONB |
| `Status.csv` | `status_dictionary` | `effects` JSONB, `blocked_actions` JSONB |
| `Item.csv` | `item_dictionary` | `effects` JSONB |

---

## 15. JSONB Format Examples

### Gu Effects
```json
{
  "active": {
    "damage_flat": 100,
    "apply_status": "BLEED",
    "status_tick_damage": 25,
    "status_duration_ticks": 3,
    "target_selector": "COL"
  },
  "passive": {
    "stat_base_attack": 50,
    "stat_base_hp": 200
  },
  "other": [
    {"event_trigger": "ON_TICK", "action_code": "MANA_REGEN_TICK", "params": {"value": 0.5}}
  ]
}
```

### Character Effect (Race/Constitution/Status)
```json
{
  "stat_mods": {"attack": 0.10, "defense": 0.10},
  "dao_mark": {"fire": 0.20},
  "path_damage": {"fire": 0.15},
  "path_resistance": {"water": 0.10},
  "max_mana": 0.50,
  "mana_regen": 1.0,
  "luck": 5,
  "healing_received": 0.50
}
```

---

*Last updated: Auto-generated from codebase analysis*
*Run `cd backend && go test ./...` to verify changes*