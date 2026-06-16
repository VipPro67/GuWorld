// Package handlers implements HTTP handlers and gameplay logic for the GuWorld game.
// Combat logic is split across files with the combat_ prefix. See:
//
//	combat_attack.go     - CombatAttack handler (main PVE/PVP turn executor)
//	combat_effects.go    - CalculateGuEffect*, getActorPathMasteryMultiplier
//	combat_state.go      - InitializeCombatState, StartCombat
//	combat_targeting.go  - ResolveSingleTarget, ResolveTargets, findCombatant
//	combat_damage.go     - applyDamageWithLogs, calculateLooseGuFailureChance
//	combat_ai.go         - BuildEnemyActions, BuildBasicEnemyAction, SelectActionsByAI
//	combat_loot.go       - RollLoot, ApplyLoot, TransferTempLootToInventory
//	combat_integration.go- IntegratePassiveGus
//	combat_explore.go    - ExploreNode, checkAndIncrementExploreLimit
//	combat_retreat.go    - Retreat handler
//	combat_auto.go       - AutoCombatStart handler (deterministic simulation)
package handlers
