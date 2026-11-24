# Effect primitive inventory

This inventory guides the migration from native `UNIQUE` codes to typed effect operations. Counts come from the current Item, Gu, Race, Constitution and Status CSV snapshots; runtime classifications come from the current consumers, not merely catalog registration. Re-run the count script below after editing these CSVs.

## Current scope

| Dataset | Rows | Native occurrences | Distinct codes | Status |
| --- | ---: | ---: | ---: | --- |
| Item.csv | 897 | 0 | 0 | All 110 item operations use typed operations; no `UNIQUE` remains in the dataset. |
| Gu.csv | 4,713 | 0 | 0 | Typed migration complete across all data-driven effects; zero UNIQUE operations remain in the dataset. Decomposed active skills and passive rules into primitives (DAMAGE MULTIPLIER, phase-aware DAMAGE_MODIFIER and HP_COST, TARGET_DAMAGE_MODIFIER, MITIGATE_DAMAGE, DETONATE_STATUS, DIRECT STATUS, SET_RULE, target-aware HEAL RESOURCE, combat NOTIFY including PRE_CHAIN, BEFORE_HIT and FINAL_RIDER, TRANSFER_RESOURCE, RESOURCE_CONVERSION, TARGET_REDIRECTION, OUTCOME_ROULETTE, SHIELD, HEAL, CLEANSE, REVOKE_STATUS, DISPEL, HIT_RULE including counter-threshold rules, TURN_ORDER_RULE, DISPLACE including battle-start SET_ROW, SUMMON, DIRECT_HP, GRANT_CAPABILITY, MODIFY_APTITUDE, GRANT_BODY_MOD, DEDUCTION_FINISH_CHANCE, BONUS_STRIKE, CHAIN_DAMAGE, COUNTER_ADD, STAT_MOD counter-delta growth, COUNTER_COUNT damage scaling, COUNTER_PERCENT healing, THRESHOLD_RECOVERY, TENACITY_MODIFIER, conditional STATUS_IMMUNITY, SLOT_INTERACTION, ON_CAST STATUS, END_BATTLE, GU_SEAL, REVIVE NIRVANA_FIRE); removed unsupported no-op hooks. |
| Race.csv | 18 | 0 | 0 | Supported passive stats use `STAT_MOD`; boolean lifespan/regen rules use `SET_RULE`. |
| Constitution.csv | 23 | 0 | 0 | Passive scalars/path modifiers, including `SOUL_DEFENSE`, use typed `STAT_MOD`. |
| Status.csv | 22 | 0 | 0 | Supported scalar/stat-map hooks use typed `STAT_MOD`; the regen restriction uses `SET_RULE`. Combat `STATUS` is not automatically a compatible replacement. |

Across these five datasets, 0 `UNIQUE` operations remain. Recent migration batches decomposed combat moves, player-persistent status actions, consumable effects, and the final 4 specialized hooks into typed operations with parity coverage (`TestCombatEffectMigrationParity`, `TestCSVNativeUniqueAllowlist`):
- `LUCK_ROULETTE`: `OUTCOME_ROULETTE` (`sub_type: "LUCK"`, `ON_ACTION`, `TARGET`) with weighted threshold parameters, damage multipliers, mana refund, self-damage and curse duration authored in `params`.
- `CHAOS_GAMBLE`: `OUTCOME_ROULETTE` (`sub_type: "CHAOS"`, `ON_CAST`, `SELF`) with `effect_choices` and `cost_choices` in `params`; retains in-combat buffs/debuffs and the persistent post-combat player penalties (`applyLoveGuPenalties`).
- `DREAM_UNRAVEL`: `RESOURCE_CONVERSION` (`sub_type: "DISPEL_TO_RESOURCES"`, `ON_ACTION`, `SELF`) with `base_conversion_units`, `mana_percent_per_unit`, `shield_percent_per_unit`, and `clear_control_states` in `params`.
- `FORGET_WILL_CONFUSE`: `TARGET_REDIRECTION` (`sub_type: "CONFUSION"`, `BEFORE_ENEMY_ATTACK`, `SELF`) with `miss_chance`, `backfire_chance`, `hit_self_chance`, `hit_user_chance`, `activation_chance`, and `backfire_disable_turns` in `params`.
- `SWORD_MARK_DEATH`: `DAMAGE MULTIPLIER 1.5` + 2 stacks `STATUS SWORD_MARK` (duration 4, `DIRECT`, `STACK`)
- `THREE_MEN_MAKE_TIGER`: `DAMAGE MULTIPLIER 1.5` + `STATUS FEAR` (turns 3, `DIRECT`, `STACK`) + `STATUS WEAK 25` (turns 3, `DIRECT`, `STACK`)
- `IMMORTAL_BINDING_VINE`: `STATUS BIND` (turns 2, `DIRECT`, `STACK`) + `TRANSFER_RESOURCE` (`resource: "MANA"`, `sub_type: "PERCENT_OF_TARGET"`, `value: 0.20`)
- `EXTREME_ICE_CRYSTAL_COFFIN`: `DAMAGE MULTIPLIER 1.4` + `STATUS FREEZE` (turns 2, `DIRECT`, `STACK`) + `STATUS SILENCE` (turns 2, `DIRECT`, `STACK`)
- `INFINITE_WIND`: `DAMAGE MULTIPLIER 1.75` + `STATUS HASTE 60` (turns 3, `DIRECT`, `STACK`, target `SELF`) + `STATUS EVASION 35` (turns 3, `DIRECT`, `STACK`, target `SELF`)
- `MYRIAD_STAR_FIREFLY`: `DAMAGE MULTIPLIER 1.6` + `TRANSFER_RESOURCE` (`resource: "MANA"`, `sub_type: "PERCENT_OF_TARGET"`, `value: 0.40`)
- `YOU_DU_STRENGTH_TIGER`: `DAMAGE MULTIPLIER 1.85` + `STATUS BLEED 20` (turns 3, `DIRECT`, `STACK`) + `STATUS STUN` (turns 1, `DIRECT`, `STACK`)
- `FUTURE_SELF`: `DAMAGE MULTIPLIER 1.80` + `STATUS FUTURE_SELF 80` (turns 3, `DIRECT`, `STACK`, target `SELF`) + `STATUS HASTE 50` (turns 3, `DIRECT`, `STACK`, target `SELF`)
- `FAREWELL_FRIEND_WIND`: `DAMAGE MULTIPLIER 1.50` + `HIT_RULE BYPASS_SHIELD` + `STATUS FAREWELL_CURSE 10` (turns 4, `DIRECT`, `STACK`)
- `SPATIAL_SHIFT` & `DIMENSION_SLASH`: `DAMAGE MULTIPLIER 1.25` + `STATUS EVASION 75` (turns 1, `DIRECT`, `STACK`, target `SELF`)
- `JUSTICE_VERDICT`: `DAMAGE MULTIPLIER 1.35`
- `NO_EXTREME_SPACE`: `STATUS EVASION 50` (turns 3, `DIRECT`, `STACK`, target `SELF`)
- `HEAVEN_NET_RESTRICTION`: `DAMAGE MULTIPLIER 1.50` + `HIT_RULE GUARANTEED` + `STATUS SLOW1 40` (turns 3, `DIRECT`, `STACK`)
- `TOTEM_GOLDEN_BELL`: `STATUS TOTEM_GOLDEN_BELL` (turns 4, `DIRECT`, `STACK`, target `SELF`)
- `POISON_VOW` (5 rows): converted to atomic `NOTIFY` with `TriggerOnConsume` support and message `"{gu} kết khế ước — hai bên không thể phản bội."`
- `VAN_DAN_DONG_TAM`: `SHIELD PERCENT 35` (target `SELF`)
- `EXECUTE_STRIKE`: `DAMAGE MULTIPLIER 2.0` (conditional on `TARGET_HP_PERCENT LTE 0.35`)
- `BONE_THORNS`: `STATUS BONE_ARMOR` (turns 3, `DIRECT`, `STACK`, target `SELF`)
- `PAINTING_IMPRISONMENT`: `STATUS BIND` (turns 1, `DIRECT`, `STACK`)
- `TIEP_THIEN_DIA`: `HEAL RESOURCE MANA PERCENT 25` (target `SELF`) + `CLEANSE ALL` (target `SELF`)
- `TIME_REWIND`: `HEAL PERCENT 35` (target `SELF`) + `HEAL RESOURCE MANA PERCENT 40` (target `SELF`)
- `REFINE_SELF` (2 rows): `STATUS REFINE_SELF 20` (turns 99, `DIRECT`, `STACK`, target `SELF`)
- `CALCULATED_STRIKE`: `DAMAGE MULTIPLIER 1.35`
- `YAMA_EMPEROR` (2 rows): `STATUS YAMA_EMPEROR` (turns 4, `DIRECT`, `STACK`, target `SELF`)
- `ALL_PEOPLE_UNITED` (2 rows): `SHIELD PERCENT 30` (target `SELF`) + `STATUS ALL_PEOPLE_UNITED 35` (turns 4, `DIRECT`, `STACK`, target `ALL_ALLIES`)
- `DRY_WOOD_SPRING` (3 rows): `STATUS DRY_WOOD_SPRING` (turns 3, `DIRECT`, `STACK`, target `ALL_ALLIES`)
- `SWORD_ARMOR_PIERCE`: `DAMAGE MULTIPLIER 1.25`
- `SOUL_BEAST_SUMMON` (3 rows): `SHIELD PERCENT 35` (target `SELF`) + `DAMAGE MULTIPLIER 1.30`
- `LIGHT_PURIFY`: `HEAL PERCENT 25` (target `SELF`) + `CLEANSE DEBUFF` (target `SELF`)
- `DARK_STEALTH` & `DARK_CURSE_WEAK` (2 occurrences): `STATUS STEALTH` (turns 2, target `SELF`) + `STATUS WEAK 30` (turns 3) + `TRANSFER_RESOURCE` (`resource: "MANA"`, `sub_type: "PERCENT_OF_TARGET"`, `value: 0.15`)
- `LIGHT_TIME_FLYING_BLADE` (2 rows): `DAMAGE MULTIPLIER 1.65` + `TRANSFER_RESOURCE` (`resource: "MANA"`, `sub_type: "PERCENT_OF_TARGET"`, `value: 0.30`) + `STATUS SLOW 50` (turns 3, `DIRECT`, `STACK`)
- `CHAOS_CAVE_DRILL` (2 rows): `DAMAGE MULTIPLIER 1.80` + `DIRECT_HP SHIELD_STRIP 100` + `HIT_RULE BYPASS_SHIELD`
- `LANDSCAPE_AS_BEFORE`: `SHIELD PERCENT 40` (target `SELF`) + `CLEANSE DEBUFF` (target `SELF`)
- `SOUL_DEFENSE` (Gu, Race, Constitution): migrated to passive `STAT_MOD SOUL_DEFENSE`; projection preserves the existing aggregate, though combat has no soul-damage consumer yet.
- `BLINDING_MIRAGE`: `STATUS CONFUSE` (turns 2, `DIRECT`, `STACK`, chance 0) + `STATUS BLIND 40` (turns 2, `DIRECT`, `STACK`, chance 0); direct chance 0 avoids consuming RNG for legacy-guaranteed applications.
- `FORM_BEAR`: `STATUS BEAR_FORM 30` (turns 3, `DIRECT`, `STACK`, target `SELF`) + `SHIELD PERCENT 25` (target `SELF`); explicit status value preserves the native 30% defense behavior.
- `SOUL_SHOCK`: `HIT_RULE BYPASS_SHIELD`; removed the legacy SOUL_WOUND marker because no runtime consumer reads it (typed `STATUS SOUL_WOUND` is direct damage and has different semantics)
- `HUMAN_WILL`: `DAMAGE_MODIFIER MISSING_HP` (`value: 1.50`, `per_missing_hp: 0`) with `HP_PERCENT LTE 0.40` at `EARLY_RIDER`, preserving its native hook position.
- `PASSION_OVERLOAD`: `DAMAGE_MODIFIER MISSING_HP` (`value: 1.50`, `per_missing_hp: 0`) with `HP_PERCENT LTE 0.35` at `LATE_RIDER`, plus selective `CLEANSE STATUS STUN` and `FREEZE`; unrelated debuffs remain.
- `GU_DETONATION`: `HP_COST CURRENT_HP_PERCENT` (`value: 0.05`, `min_cost: 1`, `min_remaining: 1`) at `EARLY_RIDER`, preserving the cost point in `GU_ASCEND_WELL_1`.
- `MANA_EAT`: `HEAL RESOURCE MANA PERCENT 25` (target `SELF`); fixed `GU_ANCIENT_FISH_1`'s stale `BEFORE_HIT` trigger, which had no native consumer.
- `SONG_OF_SEPARATION` (2 rows): `DAMAGE MULTIPLIER 1.65` + `DIRECT_HP SHIELD_STRIP 30` + `DISPEL BUFFS 2` + `STATUS WEAK 25` (turns 3, `DIRECT`, `STACK`); shield strip runs before hit damage.
- `SWORD_WAVE_TRIPLE` (3 rows): `DAMAGE MULTIPLIER 1.80` + `HIT_RULE MULTI_HIT count 3` + direct `STATUS BLEED 25` on the primary target.
- `HERO_AMONG_MEN` (2 rows): `DAMAGE_MODIFIER MISSING_HP` (`value: 1.30`, `per_missing_hp: 1.50`); final multiplier is `1.30 + missingHP/maxHP × 1.50`, applied to action and child operation damage.
- `SWORD_WAVE_TRIPLE` (3 rows): `DAMAGE MULTIPLIER 1.80` + `STATUS BLEED 25` (turns 3, `DIRECT`, `STACK`)
- `THIRTY_THREE_HEAVEN_LIGHT` (2 rows): `DAMAGE MULTIPLIER 1.45` + `CLEANSE DEBUFF` (target `ALL_ALLIES`) + direct `STATUS BLIND 35` (turns 3, target `ALL_ENEMIES`). The legacy native hook wrote blind value 0 (no accuracy penalty); both paths now use the intended 35% penalty.
- `WORDS_ARE_FRIGHTENING` (2 rows): `DAMAGE_MODIFIER TARGET_DEBUFF_COUNT` (`value: 1.40`, `per_target_debuff: 0.25`) + direct `STATUS SILENCE` (2 turns); modifier/log counts the target's debuffs before applying this block's silence.
- `IGNITE_BURST`: `DAMAGE_MODIFIER TARGET_HAS_STATUS` (`status_id: "BURN"`, multiplier 1.60); checks pre-existing BURN before this action applies its own status.
- `BLEED_BURST`: `DETONATE_STATUS BLEED` (`per_stack: 100`, `max_hp_percent: 0.08`, `phase: EARLY_RIDER`); consumes all active BLEED stacks and applies direct HP damage, preserving its old phase and log.
- `IMMEASURABLE_QI_SEA` (2 rows): `DAMAGE MULTIPLIER 1.7` + `TARGET_DAMAGE_MODIFIER REDUCE 0.3` (minimum multiplier 0.2) + direct `STATUS SLOW1 30` for 3 turns; modifier and status execute before damage, and the CSV-authored aggregate log is emitted once.
- `THREE_QI_RETURN` (2 rows): `DAMAGE MULTIPLIER 1.85` + `DEFENSE_PENETRATION 0.5` on the enemy, then `HEAL HP PERCENT 30` and `HEAL MANA PERCENT 100` on SELF. The native hook remains for old stored definitions.
- `MYRIAD_DISMANTLE` (2 rows): `DAMAGE MULTIPLIER 1.6` + rank-independent `DIRECT_HP SHIELD_STRIP 100` + `DISPEL ALL_BUFFS` + `DEFENSE_PENETRATION 1`; the authored aggregate log reports the full dismantling. Removed the rank-7 extra 25% shield strip because the same action already strips the entire shield.
- `BEAST_PHANTOM_STRIKE`: `BONUS_STRIKE` (`chance: 0.45`, `value: 0.75`, `min_damage: 50`) with CSV-authored log. Its dispatcher hook runs at the former inline rider location, preserving the RNG draw position and the legacy direct-HP semantics.
- `CHAIN_LIGHTNING`: `CHAIN_DAMAGE` (`value: 0.5`, `min_damage: 30`, `target_count: 1`) with CSV-authored log. The phase dispatcher runs at the former inline rider location and preserves secondary-target selection and direct-HP semantics; the native hook remains for stored definitions.
- `UNRAVEL_RIDDLE`: `DISPEL BUFFS 2` on `ON_CAST`/`TARGET` with CSV-authored aggregate log. The dispatcher removes buffs at the former hook position; its legacy “interrupt” only logged and had no channeling consumer, so current data no longer claims interruption.
- `POISON_DETONATE`: `DETONATE_STATUS POISON` (`per_strength: 2.5`, `min_damage: 60`, `phase: LATE_RIDER`); consumes poison at the former native-hook phase and applies direct HP damage with the old minimum-damage rule.
- `STEAL_LIFE`: `TRANSFER_RESOURCE HP/PERCENT_OF_MAX_HP` (`value: 0.15`, `min_amount: 30`, `phase: LATE_RIDER`); drains HP directly and restores the same configured amount, preserving the old hook's floor and timing.
- `FULL_SOUL_HEAL` on `GU_GUTS` was removed: its `ON_CAST` combat hook had no executor in the Gu-consumption flow, which only evaluates `ON_CONSUME`; the soul-capacity grant remains.
- `DAO_ADAPTIVE_RESIST`: `STATUS ADAPTIVE_RESIST 25` (turns 3, `DIRECT`, `STACK`); corrected the stale `BEFORE_HIT` Gu trigger to `ON_ACTION`, which is the native consumer timing.
- `BLOOD_SACRIFICE`: `HP_COST CURRENT_HP_PERCENT 15%` (`min_cost: 1`, `min_remaining: 1`) at `PRE_RIDER`; the old x2 multiplier operated on a zero-damage action and is removed.
- `DEVOUR_HEAVEN_EARTH`: `DAMAGE MULTIPLIER 1.50` + `TRANSFER_RESOURCE HP/PERCENT_OF_TARGET_SHIELD 0.50` at `EARLY_RIDER`; the transfer consumes half the current shield and restores that amount as HP before damage.
- `BLOOD_DEMON_DISINTEGRATION`: `HP_COST CURRENT_HP_PERCENT 0.20` (`min_cost: 10`, `min_remaining: 1`) + `DAMAGE MULTIPLIER 2.20` + `HEAL LIFESTEAL 50`; the HP cost log preserves the native hook message and timing.
- `ARENA_IDENTITY_MASK`: removed from `GU_ATTITUDE_GU` and the authoring allowlist; no gameplay consumer exists.
- `FORMATION_DOMAIN`: migrated to `SHIELD PERCENT 20` + `STATUS DEF_UP 25` (3 turns), both targeting `SELF`; simulator parity covers shield and status state.
- `WIND_HASTE_BURST`: `STATUS HASTE 40` + `STATUS EVASION 25` (mỗi status 3 turns, `DIRECT`, `STACK`, target `SELF`); first status owns the combined legacy message and second uses validated `suppress_log`.
- `IMMUNE_DREAM_STATUS`: `SET_RULE IMMUNE_DREAM_STATUS enabled=true`, preserving dream-status immunity through the shared rule projection.
- `FIRST_TURN_EVADE`: `HIT_RULE FIRST_TURN_EVADE` at `BEFORE_HIT`/`SELF`; the compiled combat adapter routes it through the defensive hit phase with the first-hit behavior intact.
- `LETHAL_LUCK_WARD`: `HIT_RULE LETHAL_LUCK_WARD` at `BEFORE_HIT`/`SELF`; CSV configures `chance`, `shield_percent` and `shield_min`. Compiled typed data reaches the same lethal-damage consumer and retains its one-use state and RNG draw position.
- `SEIZE_INITIATIVE` and `SUPPRESS_PRIORITY`: `TURN_ORDER_RULE` at `ON_BATTLE_START`/`SELF`; initiative keeps its authored lifespan threshold and suppression routes into the existing opposing-priority logic.
- `RULE_IMPOSITION`: `DISPEL ALL_BUFFS` on `TARGET`; the `DISPEL` primitive supports unlimited buff removal without a numeric cap.
- `SACRIFICE_APTITUDE`: `MODIFY_APTITUDE ADD_FLAT 5`; retains aptitude-tier recalculation, MaxMana delta/cap and the separate 20-hunger use cost.
- `SOVEREIGN_BODY_TRANSFORM`: `GRANT_BODY_MOD SOVEREIGN_IMMORTAL`; remains one-use and keeps duplicate-grant rejection.
- `PACK_FRENZY`: `DAMAGE_MODIFIER MINION_COUNT` (`COMBAT_PASSIVE`/`SELF`, base 1 + 0.15 per living allied minion); evaluates at its historical pre-rider point, scales both the action and child damage, and retains its authored log template.
- `HEAVEN_SUPPRESSION`: `DAMAGE MULTIPLIER 1.35` + `DISPEL BUFFS 2` on `TARGET`; separated this hostile rider from the Gu row’s self-target defensive operations.
- `SONG_OF_DESTINY` (2 rows): `DAMAGE MULTIPLIER 1.50` + `STATUS SONG_OF_DESTINY_WEAK` (30, 1 turn, `DIRECT`, `REPLACE`, `ALL_ENEMIES`) + `STATUS SONG_OF_DESTINY` (40, 1 turn, `DIRECT`, `REPLACE`, `ALL_ALLIES`) + `HEAL RESOURCE MANA PERCENT 20` (`ALL_ALLIES`) + authored combat `NOTIFY`. REPLACE preserves the native hook's refresh-by-status-ID behavior across Gu sources. Status and mana target selectors apply to each living recipient; mana percentage uses each recipient MaxMana. Parity covers outcome and authored log.
- `BERSERK_STATE`: `DAMAGE_MODIFIER MISSING_HP` (`value: 1`, `per_missing_hp: 0.8`), `TENACITY_MODIFIER` (`per_missing_hp: 0.5`), and conditional `STATUS_IMMUNITY STUN/FREEZE` (`HP_PERCENT LTE 0.3`). Native hook remains for older definitions.
- `EATING_LOSS_BLESSING`: `COUNTER_ADD` + two `STAT_MOD ADD_COUNTER_DELTA` operations grant +5 ATTACK and +5 DEFENSE only for each effective `eating_loss_stacks` increment at `BEFORE_HIT`, respecting the counter cap. `NOTIFY BEFORE_HIT` authors the per-stack log. Removed `UNIQUE` from all three current ranks; native compatibility hook remains for stored definitions.
- `QI_JUE_RETURN_LIFE` (2 rows): `THRESHOLD_RECOVERY` configures the 30% HP trigger, one-use limit, 40% MaxHP heal, debuff cleanse and authored log. It runs at the former BEFORE_HIT hook point; the ineffective `SHIELD` operations on those rows were removed.
- `BLOOD_STAINED_ROBE` (2 rows): `MITIGATE_DAMAGE CONVERT_TO_SHIELD` (`value: 0.4`) reduces each incoming strike by 40% and converts the removed amount into shield at the former native hook phase. Parity compares combat state and exact log output; the ineffective `SHIELD` operation formerly authored at `BEFORE_HIT` was removed.

The final 4 `UNIQUE` operations have been migrated into typed primitives, achieving zero `UNIQUE` operations across all datasets: `CHAOS_GAMBLE` (1), `DREAM_UNRAVEL` (1), `FORGET_WILL_CONFUSE` (1), `LUCK_ROULETTE` (1). `CHAOS_GAMBLE` uses typed `OUTCOME_ROULETTE` (`sub_type: "CHAOS"`, `ON_CAST`, `SELF`) with `effect_choices` and `cost_choices` in `params`, preserving in-combat mechanics and the post-combat persistent penalties in `applyLoveGuPenalties`. `LUCK_ROULETTE` uses typed `OUTCOME_ROULETTE` (`sub_type: "LUCK"`, `ON_ACTION`, `TARGET`) with all threshold, damage multiplier, mana refund, self-damage, and curse parameters in `params`. `DREAM_UNRAVEL` uses typed `RESOURCE_CONVERSION` (`sub_type: "DISPEL_TO_RESOURCES"`, `ON_ACTION`, `SELF`) with `base_conversion_units`, `mana_percent_per_unit`, `shield_percent_per_unit`, and `clear_control_states` in `params`. `FORGET_WILL_CONFUSE` uses typed `TARGET_REDIRECTION` (`sub_type: "CONFUSION"`, `BEFORE_ENEMY_ATTACK`, `SELF`) with `miss_chance`, `backfire_chance`, `hit_self_chance`, `hit_user_chance`, `activation_chance`, and `backfire_disable_turns` in `params`. `TestCSVNativeUniqueAllowlist` now enforces an empty allowlist (0 `UNIQUE` codes across Item, Gu, Race, Constitution, and Status CSVs). `NIRVANA_FIRE` now uses typed `REVIVE NIRVANA_FIRE` with the same CSV-configured parameters and lethal-damage executor. `FORGET_INTENT_INTERRUPT` now uses typed `GU_SEAL REACTIVE`, with success chance, seal duration, activation cap and backfire duration configured in Gu.csv; its native code remains only as a compatibility reader. The five former `REBORN` rows now use typed `REVIVE` and share the existing death executor, including rewind/in-place behavior and authored parameters. `FAKE_REGIONAL_WALL` now uses `RANK_SCALED_DEBUFF REGIONAL_WALL` with CSV-configured strength, status, source and rank-based duration across combat-stat and persistent-PvP consumers; its native hook remains for old definitions. `REVERSE_CALAMITY` now uses `EFFECT_IMMUNITY HOSTILE_EFFECTS`, which rejects hostile status/Gu-seal effects at their existing gates while leaving raw damage untouched; its native hook is retained for persisted definitions. `NIRVANA_FIRE` is authored as `REVIVE NIRVANA_FIRE`; the subtype projects to its specialized lethal-damage executor so retaliation remains at the same point. `DREAM_UNRAVEL` remains a composite hook for removing dispellable effects and cached control flags, but base conversion units, mana/shield percentage per unit and flag clearing are now CSV-configured and validated. `SAT_KHI_GUARANTEE` now uses `HIT_RULE COUNTER_THRESHOLD` with CSV-authored key, strict threshold, guaranteed hit, shield bypass and immunity bypass; the native hook remains only for older persisted definitions. `FATE_VANQUISHED` now uses `DAMAGE MULTIPLIER 2.2` and `DISPEL PROTECTION`, which clears shield and invincibility/last-stand flags and markers before hit damage; the native hook remains for older persisted definitions. `POEM_INTENT` now uses `COUNTER_ADD` for its start-of-turn stack; the native rider remains to apply damage reduction, threshold reset, and shield destruction. `SWORD_DRAGON_TRANSFORM` now uses `RESOURCE_CONVERSION CURRENT_HP_TO_SHIELD` and direct `STATUS ATK_UP` in the same ON_CAST block; the conversion holds 1 HP, grants the remainder as shield, and preserves the native transform log. `BORROW_PAST_POWER` now uses `STATUS ATK_UP` on `ON_CAST`/`SELF`; its native hook remains only as a compatibility reader for older persisted definitions. `GU_XUAN` now uses `END_BATTLE FORCE_VICTORY`; the native `INSTANT_WIN` hook remains only to read older definitions. `CHAIN_LIGHTNING` now uses `CHAIN_DAMAGE`; its native hook remains only for compatibility with old persisted definitions. `UNRAVEL_RIDDLE` now uses `DISPEL BUFFS` on ON_CAST; its native hook remains for old definitions. `CLEANSE_REGIONAL_WALL` is now represented by `REVOKE_STATUS` in current Gu.csv; its native hook remains only for compatibility with older stored definitions. `GU_WOLF_ENSLAVEMENT_CORE` uses `SUMMON` with actor-stat scaling and `DISPLACE SET_ROW 3` at battle start; the wolf-specific hook remains only for older stored definitions. `PERM_STEAL_GU` and `TEMP_STEAL_GU` use `SLOT_INTERACTION` to configure mode, rank-scaled chance, vital-Gu penalty, random target selection and full-combat duration; `combat_steal.go` retains the PvP transaction and inventory rules.

`UNIQUE` may remain temporarily while its code has a verified consumer. An operation is migrated only when data validation and the runtime executor agree on trigger, context, target and parameters.

## Item.csv migration completed

All 12 former item `UNIQUE` occurrences now use typed operations: `GRANT_RESOURCE`, `RESTORE_RESOURCE`, `MODIFY_APERTURE`, `RANDOMIZE_TRAIT`, `REPAIR_APERTURE`, `OPEN_REWARD_CONTAINER`, and `PLACE_WORLD_OBJECT`. The legacy flat-item migration converter also emits typed operations for known aliases and reports unsupported/no-op aliases instead of recreating `UNIQUE`. Domain-specific executors retain resource caps, transaction requirements, RNG behavior and placement/reward rules. Legacy item handlers remain for older persisted definitions; new CSV data contains no native item hooks. Generic item conditions remain unsupported.

Three Gu `SPRING_EGG_HATCH` occurrences now share the item's `PLACE_WORLD_OBJECT PRIMEVAL_SPRING` operation and the same transaction-backed placement rule. The legacy native hook remains registered for old definitions.

117 Gu capability rows use `GRANT_CAPABILITY`, including disguise, capture bonus, movement-range, portal, loot-preview and access capabilities. The typed projection remains compatible with legacy definitions. `ARENA_IDENTITY_MASK` was removed from Gu data and capability validation because no gameplay consumer exists.

59 `GROWTH_DURATION_HOURS` hooks now use `GROWTH_DURATION` (`ALWAYS_ON`, `SELF`, value in hours). Growth projection reads the typed scalar and falls back to the native hook for persisted legacy definitions.

63 resource tick rows now use `PERIODIC_RESOURCE`: `REGEN_PER_HOUR` carries MANA/THOUGHT/SOUL and keeps rank scaling; `GENERATION_WORKER_TICK` carries PRIMEVAL_STONE/IMMORTAL_STONE and keeps the worker's 3-hour cadence and no-rank-scaling rule. Both old projected Gu definitions and workers still read native hooks for compatibility.

17 consume-time resource effects now use shared resource operations: grant LIFESPAN, modify max SOUL/THOUGHT, or restore SOUL/THOUGHT. The Gu event executor retains the historical default max initialization and caps; legacy native hooks remain readable.

15 `RANK_PROGRESS` hooks now use `ADVANCE_PROGRESSION` (`ON_CONSUME`, `SELF`). Its Gu executor preserves the matching-rank check, peak-subrank restriction and aperture-petrification breakthrough guard; it changes only the subrank and does not cross a major-rank boundary.

Four `MAN_BEAST_BURIAL` hooks now use `ADVANCE_MAJOR_REALM` (`ON_CONSUME`, `SELF`). The typed executor preserves Rank 1–4 and matching dictionary/instance rank checks, peak-subrank and aperture-petrification guards, then advances one major rank, resets wall progress and reduces max mana by 10 with a floor of 20.

Five Gu `PATH_DAMAGE_MODIFIER` hooks now use `STAT_MOD` (`ALWAYS_ON`, `SELF`) with `stat=PATH_DAMAGE`, a lowercase path key and the authored additive ratio. `gamefx` projects this typed stat into the same path-damage aggregate used by combat; legacy map-valued aliases remain for race/constitution data.

Four `STAR_THOUGHT_RESTORE` combat hooks now use `HEAL` with `sub_type=RESOURCE`, `resource=MANA`, `unit=PERCENT`, and `value=30`. The combat adapter resolves the percentage against the actor MaxMana, restores only to the cap, and honors the operation-level `SELF` target even though the parent attack targets an enemy.

Eight `RECORD_MESSAGE` Gu hooks now use `NOTIFY` (`ON_USE`, `SELF`). The message is authored in Gu.csv, `{gu}` expands to the source Gu name, and the use handler returns the same player-facing sentence as the native hook without a native-code switch.

Ten status-only combat hooks now use `STATUS`: BIND (2 turns), FREEZE (1), BLIND (value 60, 2), SILENCE (2), SOUL_WOUND (value 25, 3), SLOW_DAMAGE (value 35, 3), CHARM (2), and GU_SEAL (3). They set `application_mode=DIRECT`, which keeps the legacy guaranteed application semantics and avoids rank suppression, resistance, reversal, tenacity and RNG gates. Native handlers remain available for persisted legacy Gu definitions. `BONE_SPIKE` remains native because its BLEED effect uses legacy `ActiveEffects`, which is consumed differently from typed `ActiveDoT` statuses.

`EARTH_GRAVITY` now composes `STATUS SLOW` (value 40, 3 turns, `DIRECT`, `STACK`) with `phase=PRE_DAMAGE`, matching the native hook's application point and preserving its authored combat log through the shared status adapter. Its Go hook remains for old persisted definitions. Three `RAGING_ANGRY_BIRD` rows now combine `DAMAGE MULTIPLIER 1.5` and `STATUS BURN` (value 15, 3 turns, `DIRECT`, `STACK`, `PRE_DAMAGE`); their former Gu-specific mutations and log now use shared damage and status adapters.

Race, Constitution, and Status passive hooks for stat maps, resource/healing modifiers, path modifiers, refinement, cultivation, luck, resistance, and aperture stress use `STAT_MOD`; the typed projection preserves their existing `gamefx` aggregates. Six lifespan/regen flags use `SET_RULE` with a closed rule subtype and required boolean `enabled`, projecting into the existing `Effects` flags. `SOUL_DEFENSE` now projects through typed `STAT_MOD` to the same player/combatant aggregate as before. The simulator still has no soul-damage resolution consumer, so the value is visible in stats but does not currently reduce damage.

Three `GUARANTEED_HIT` action hooks now use `HIT_RULE GUARANTEED`. Combat action adaptation sets the shared hit-rule flag, and the simulator applies it at the existing evasion/miss checks.

`HEAVY_LIFESTEAL` now uses `HEAL LIFESTEAL` in the same action block as its `DAMAGE`, so both resolve in one combat operation and lifesteal uses the damage actually dealt. The native hook remains available for older definitions.

11 consume-time stress/time-rate effects now use `MODIFY_APERTURE`: `STRESS_REDUCE` floors stress at zero, `TIME_RATE_ADD` adds the declared amount, and `TIME_RATE_SUBTRACT` floors time rate at 0.1, and `TIME_RATE_MULTIPLY` multiplies it by the authored factor. The two legacy negative decrease values are normalized to 0.1 because the old executor treated every nonpositive amount as its default.

13 summons now use typed `SUMMON` with closed monster IDs, scale, position and stat-source schema. Combat projection resolves monster templates at build time; actor-based summons can configure rank/speed, opt out of rank scaling and author stable IDs/log templates. Dao scaling, clamps and summon timing remain in the simulator. `GU_WOLF_ENSLAVEMENT_CORE` additionally uses `DISPLACE SET_ROW 3` at battle start to place its owner on the middle line.

## Gu codes grouped by candidate primitive

Groups below identify semantic candidates, not an automatic mapping. Native handlers may combine multiple effects or rely on simulator timing. Keep a native typed executor for a behavior when flattening it would change timing, RNG, provenance, persistence, or battle control.

| Candidate family | Current examples and occurrences | Suggested typed surface | Notes |
| --- | --- | --- | --- |
| Resource and permanent progression | `TRANSFER_RESOURCE` 11; `ADVANCE_PROGRESSION` 15; `ADVANCE_MAJOR_REALM` 4; lifespan/max/restore resource effects are migrated below | `TRANSFER_RESOURCE`, `GRANT_RESOURCE`, `MODIFY_MAX_RESOURCE`, `RESTORE_RESOURCE`, `ADVANCE_PROGRESSION`, `ADVANCE_MAJOR_REALM` | Resource transfer supports MANA from current target mana; HP from target MaxHP with a configured minimum at `LATE_RIDER`; and HP from a percentage of current target shield at `EARLY_RIDER`. Transfers preserve receiver caps and authored combat logs. Both progression operations preserve their rank and aperture-petrification guards; major breakthrough also preserves the max-mana cost. |
| Refinement/deduction chance | Two `DEDUCTION_INSTANT_FINISH` hooks | `DEDUCTION_FINISH_CHANCE` | The deduction handler still chooses the best chance among equipped, fed Gu and makes one random roll after deducting costs; chance parameters are validated and authored in CSV. |
| Tick/resource production | `PERIODIC_RESOURCE` 63: MANA 28, THOUGHT 8, SOUL 4, PRIMEVAL_STONE 16, IMMORTAL_STONE 7 | `PERIODIC_RESOURCE` with subtype, resource and amount | `REGEN_PER_HOUR` preserves rank scaling; `GENERATION_WORKER_TICK` preserves worker cadence and rank behavior. Legacy readers remain. |
| Consume-time resource changes | LIFESPAN +7, modify max SOUL +2 / THOUGHT +2, restore SOUL +3 / THOUGHT +3 | `GRANT_RESOURCE`, `MODIFY_MAX_RESOURCE`, `RESTORE_RESOURCE` | Gu `ON_CONSUME` executor preserves max initialization and restore caps. |
| Growth and passive stats | `GROWTH_DURATION` 59; `PATH_DAMAGE` 5; `CULTIVATION_EFFICIENCY`, `TENACITY`, `LUCK`, `CC_RESIST`, `DAO_MARK_GAIN` (14 rows); `HP_REGEN`, `MENTAL_RESISTANCE`, `DAO_MARK` (3 rows migrated) | `GROWTH_DURATION`, `STAT_MOD`, `PATH_MODIFIER` | Growth projection reads typed values and retains legacy fallback. Supported passive stats use typed, context-aware `gamefx` projections; SOUL_DEFENSE uses typed STAT_MOD; its damage consumer remains unimplemented. |
| Status, stacks and counters | Combat candidates include `SOUL_DEVOUR` and `SWORD_MARK_DEATH` | `COUNTER_ADD` for ON_ACTION target counters, ON_TURN_START owner counters and BEFORE_HIT self counters; `STAT_MOD ADD_COUNTER_DELTA` for stats per effective stack; `APPLY_STACKING_STATUS` for the existing post-combat PvP hit consumer; `THRESHOLD_CHECK` only after a trigger executor exists | `COUNTER_ADD` executes from action targets and at turn start. The persistent-status consumer reads the full authored stacking policy (`max_stacks`, duration, dispel rule and tiers), persists it with the status, and applies its cleanse rule; validation requires real PvP (`pvp_only=true`) and a complete tier table. `THRESHOLD_CHECK` remains rejected by validator/compiler because it has no runtime executor. `POEM_INTENT` authors its increment as `COUNTER_ADD`; its reset and later-hit consequences remain in its native compatibility rider. `EATING_LOSS_BLESSING` now authors its counter, per-stack stat growth, per-stack log and payout through typed operations; the native hook remains only for stored definitions. |
| Persistent player status | `CLEANSE_REGIONAL_WALL` (1 Gu) | `REVOKE_STATUS` on ON_BATTLE_START/SELF | Runs before player combat stats are projected, validates the player status ID against the runtime registry, and removes the configured persistent status. The native hook remains as a compatibility reader for existing definitions. |
| Damage and mitigation riders | Armor break, execute, lifesteal, shield bypass/strip, damage reflection and path damage hooks | `DAMAGE`, `DAMAGE_MODIFIER`, `TARGET_DAMAGE_MODIFIER`, `MITIGATE_DAMAGE`, `REDIRECT_DAMAGE`, `LIFESTEAL` | Bind each command to a combat phase; preserve per-hit semantics and RNG draws. `TARGET_DAMAGE_MODIFIER REDUCE` now preserves battle-persistent target-side output reduction with an authored floor and log. |
| Movement and formation | `KNOCKBACK`, `PULL_TARGET`, `SWAP_TARGET`, `RETREAT_SELF` (4 rows migrated) | `DISPLACE` with direction/steps/selector | Runs through the `combatPhaseOnActionRider` hook registry; preserves current grid bounds, logs, and legacy hook compatibility. |
| Summon and sacrifice | `SUMMON` 12 plus other battle/action summons; blood/aptitude/entity sacrifice | `SUMMON`, `SACRIFICE_RESOURCE`, `SACRIFICE_ENTITY` | Typed summon preserves template resolution, rank/dao scaling, clamps and trigger timing; other sacrifice mechanics remain domain-specific. |
| Capability and disguise | `DISGUISE` 65, `CAPTURE_BONUS` 17, `IGNORE_MOVEMENT_RANGE` 14, portal abilities 12 combined (all 117 migrated) | `GRANT_CAPABILITY` with catalogued subtype and validated parameters | `Compiled.Capability` accepts typed grants and legacy hooks. Consumers must query the grant; catalog presence alone does not create behavior. Unimplemented capabilities are excluded from the authoring allowlist. |
| Turn/death/battle control | `REVIVE` 5, `TIME_REWIND`, steal Gu, regional/domain rules | Specialized typed operations (`REVIVE`, `REWIND_BATTLE`, `STEAL_GU`, `END_BATTLE`, `SET_BATTLE_RULE`); `TURN_ORDER_RULE` handles priority and suppression | `END_BATTLE FORCE_VICTORY` now executes from typed ON_BATTLE_START data; `INSTANT_WIN` is a compatibility reader. Keep Go executors for control flow; CSV owns parameters and activation. `LIGHTNING_STUN_BURST` now composes `DAMAGE MULTIPLIER`, direct `STATUS STUN` with a CSV-authored 40% proc at `PRE_CHAIN`, and phase-bound `NOTIFY`; this status phase runs immediately before chain riders and preserves the former RNG draw position. |
| World, record and oath | `RECORD_MESSAGE` 8 migrated to `NOTIFY`; portals, `POISON_VOW` 5, terrain and object effects remain | `CREATE_PORTAL`, `CREATE_OATH`, `MODIFY_WORLD` | `NOTIFY` preserves the narrative response only. `POISON_VOW` currently logs without enforcing an oath; verify persistence/consumer before designing a world-state primitive. |

## Special handling and non-goals for conversion

- Catalogued hooks are not proof of gameplay behavior. Mark each code `active`, `partial`, `narrative_only`, `no_consumer` or `legacy` after tracing a real consumer.
- Remove no-op effect data when it has no intended gameplay meaning; do not migrate it into a primitive just to preserve a CSV row.
- Keep `REVIVE`, legacy `REBORN`, `TIME_REWIND`, turn priority, theft/ownership changes, world placement and reward settlement as typed domain commands with explicit Go executors.
- Preserve operation order, event phase, source provenance, snapshots, transaction boundaries and deterministic RNG consumption.

## Combat execution architecture and migration order

The combat phase registry is an incremental seam, not a complete event pipeline. `combatActionRunner.runOne` owns the mutable state captured by the former closure. `BEFORE_ENEMY_ATTACK` now dispatches through the phase registry. `FORGET_INTENT_INTERRUPT` uses the typed `GU_SEAL REACTIVE` operation at its original location and RNG position; its native hook remains as a compatibility reader. Typed combat passives also support explicit early/late rider positions and condition evaluation; the method remains large and many native checks are still interleaved with action resolution. Each migration must connect validated operations to a real phase consumer. A catalog entry or schema allowance alone does not mean the simulator executes it.

Next, wire the existing stack/counter helpers through a typed simulator execution contract that defines trigger, owner, lifecycle, threshold/reset policy, and phase; then migrate only the Gu behaviors those fields express exactly. Keep control-flow effects (`REVIVE`, legacy `REBORN`, rewind, theft, threshold resets with later-hit consequences) in specialized Go executors with CSV-authored parameters. Treat damage-routing fixes (shield, immunity, thorns, spillover) as a separate behavior change with dedicated cases, and preserve RNG draw order for behavior-preserving migrations.

`PACK_FRENZY` now uses `DAMAGE_MODIFIER MINION_COUNT`. The simulator counts living minions on the actor's side at its historical pre-rider position, applies the multiplier to action and child-operation damage, and renders the configured log. `POEM_INTENT` uses a typed start-of-turn counter; its threshold reset and shield/hit consequences remain a specialized native rider.

## Data count command

Run from the repository root:

```bash
python3 - <<'PY'
import csv, json, collections
for dataset in ("Item", "Gu", "Race", "Constitution", "Status"):
    counts = collections.Counter()
    for row in csv.DictReader(open(f"backend/data/{dataset}.csv", encoding="utf-8-sig")):
        raw = json.loads(row["effects"])
        blocks = raw if isinstance(raw, list) else raw.get("blocks", [])
        for block in blocks:
            for operation in block.get("operations", []):
                if operation.get("op") == "UNIQUE":
                    counts[operation.get("code", "")] += 1
    print(dataset, "occurrences", sum(counts.values()), "codes", len(counts))
    for code, count in counts.most_common():
        print(f"  {count:3} {code}")
PY
```
