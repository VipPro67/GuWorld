// Package handlers implements HTTP handlers and gameplay logic for the GuWorld game.
// Combat logic is split across files with the combat_ prefix. See:
//
//	combat_path_mastery.go - applyPathMasteryGain (shared path mastery helper)
//	combat_sim_build.go  - shared auto-combat core: buildSimCombatState, applySimConsumption, combatOutcome, applyLoveGuPenalties, combatTypeLabel
//	combat_result_loot.go - shared victory loot application (PVP drop + loot table + distribution)
//	combat_result_mine.go - shared mine PvE victory occupation
//	combat_result_defeat.go - shared defeat resolution (death drop, hub teleport, virtual arena)
//	combat_finalize.go   - shared post-combat tail (ON_COMBAT_END, tribulation sync, persist, ally/pet updates)
//	combat_effects.go    - CalculateGuEffect*, getActorPathMasteryMultiplier
//	combat_state.go      - InitializeCombatState, StartCombat
//	combat_targeting.go  - ResolveSingleTarget, ResolveTargets, findCombatant
//	combat_damage.go     - applyDamageWithLogs
//	combat_ai.go         - BuildEnemyActions, BuildBasicEnemyAction, SelectActionsByAI
//	combat_loot.go       - RollLoot, ApplyLoot
//	combat_integration.go- IntegratePassiveGus
//	combat_retreat.go    - Retreat handler
//	combat_auto.go       - AutoCombatStart handler (manual/replay)
//	combat_auto_resolve.go - ResolveCombat (skip battle + manual replay, applies results)
package handlers
