// Package combatmath contains small deterministic combat calculations shared
// by realtime gameplay systems.
package combatmath

import "math"

// LastStandMinHP is the HP floor held by the LAST_STAND effect.
const LastStandMinHP = 1.0

// ResolveBaseDamage applies the shared defense step.
func ResolveBaseDamage(baseDamage, defense float64) float64 {
	return math.Max(1, baseDamage-defense)
}

// BasicDamageResult is the side-effect-free result of one basic hit after
// defense, damage reduction and shield absorption.
type BasicDamageResult struct {
	Damage         float64
	HPDamage       float64
	ShieldAbsorbed float64
	HP             float64
	Shield         float64
}

// ResolveBasicDamage routes one ordinary attack through defense, damage
// reduction and a virtual shield. Pass minHP=0 when there is no HP floor.
func ResolveBasicDamage(baseDamage, defense, reduction, hp, shield, minHP float64) BasicDamageResult {
	damage := ResolveBaseDamage(baseDamage, defense)
	if reduction < 0 {
		reduction = 0
	}
	if reduction > 1 {
		reduction = 1
	}
	damage *= 1 - reduction

	result := BasicDamageResult{Damage: damage, HP: hp, Shield: math.Max(0, shield)}
	result.ShieldAbsorbed = math.Min(result.Shield, damage)
	result.Shield -= result.ShieldAbsorbed
	result.HPDamage = damage - result.ShieldAbsorbed
	result.HP = math.Max(minHP, hp-result.HPDamage)
	if result.HP < 0 {
		result.HP = 0
	}
	return result
}
