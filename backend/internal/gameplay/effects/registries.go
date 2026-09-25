package effects

import (
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// These registries are authoritative vocabularies used by compilation. Adding
// authored values requires adding them here rather than string dispatch in a loop.
var Triggers = set(TriggerAlwaysOn, TriggerOnAction, TriggerOnHit, TriggerBeforeHit, TriggerOnTurnStart, TriggerOnDeath, TriggerOnTick, TriggerCapability, TriggerCombatPassive, TriggerPassive, TriggerOnBattleStart, TriggerBeforeEnemyAttack, TriggerOnCast, TriggerOnConsume, TriggerOnUse)
var Selectors = set(TargetSelf, TargetSingle, TargetRow, TargetCol, TargetAllEnemies, TargetAllAllies, TargetAll)
var Operations = set(OpDamage, OpShield, OpHeal, OpDirectHP, OpStatMod, OpStatus, OpBarrier, OpCleanse, OpUnique)
var Conditions = map[string]bool{"ACTOR_HP_PERCENT": true, "TARGET_HP_PERCENT": true, "RANK": true, "MASTERY": true}
var ConditionOperators = map[string]bool{"LT": true, "LTE": true, "GT": true, "GTE": true, "EQ": true}
var Stats = map[string]bool{"ATTACK": true, "DEFENSE": true, "HP": true, "SPEED": true, "DAMAGE_DEALT": true, "ACCURACY": true, "EVASION": true, "TENACITY": true, "CC_RESIST": true, "LUCK": true, "CRIT_RATE": true, "ALL": true, "MANA": true}
var NativeMechanicDomains = map[string]Domain{
	"REBORN": DomainCombat, "FORGET_INTENT_INTERRUPT": DomainCombat, "FORGET_WILL_CONFUSE": DomainCombat, "POEM_INTENT": DomainCombat, "SEIZE_INITIATIVE": DomainCombat, "SUMMON_WOLF_FRONTLINE": DomainCombat, "FULL_SOUL_HEAL": DomainCombat, "SWORD_DRAGON_TRANSFORM": DomainCombat, "BORROW_PAST_POWER": DomainCombat, "UNRAVEL_RIDDLE": DomainCombat, "CHAOS_GAMBLE": DomainCombat, "BYPASS_SHIELD": DomainCombat, "FIRST_TURN_EVADE": DomainCombat, "DOUBLE_DAMAGE_DEALT": DomainCombat,
	"MARKETPLACE_ACCESS": DomainCapability, "REMOTE_TRADE": DomainCapability, "CHAT_CONNECT": DomainCapability, "CHAT_PERCEIVE": DomainCapability, "IGNORE_MOVEMENT_RANGE": DomainCapability, "LOOT_PREVIEW": DomainCapability, "DISGUISE": DomainCapability, "CHANGE_FORM_DISGUISE": DomainCapability, "PERFECT_DISGUISE": DomainCapability, "ARENA_IDENTITY_MASK": DomainCapability, "SACRIFICE_NPC": DomainCapability,
	"SACRIFICE_APTITUDE": DomainItemUse, "RANK_PROGRESS": DomainItemUse, "ADD_SOUL_MAX": DomainItemUse, "ADD_LIFESPAN_YEARS": DomainItemUse, "RESTORE_SOUL_HP": DomainItemUse, "RESTORE_MENTAL": DomainItemUse, "INCREASE_TIME_SPEED": DomainItemUse, "REDUCE_STRESS": DomainItemUse, "ADD_THOUGHT_MAX": DomainItemUse, "DECREASE_TIME_SPEED": DomainItemUse,
	"MANA_REGEN_TICK": DomainGrowth, "THOUGHT_REGEN_TICK": DomainGrowth, "SOUL_REGEN_TICK": DomainGrowth, "GENERATE_STONES_TICK": DomainGrowth, "GENERATE_IMMORTAL_STONES_TICK": DomainGrowth,
	"MAN_BEAST_BURIAL": DomainItemUse, "POISON_VOW": DomainItemUse, "SPRING_EGG_HATCH": DomainItemUse, "STONE_APERTURE": DomainItemUse, "THIRD_WATCH": DomainItemUse, "RECORD_MESSAGE": DomainWorld,
}
var NativeMechanics = func() map[string]bool {
	m := map[string]bool{}
	for k := range NativeMechanicDomains {
		m[k] = true
	}
	return m
}()

func set[T comparable](values ...T) map[T]bool {
	m := make(map[T]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

// Statuses exposes statusfx's catalog rather than maintaining a competing list.
func Statuses() []string {
	result := make([]string, 0, len(statusfx.Registry))
	for id := range statusfx.Registry {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
