// Package ai defines placeholder monster archetype parameters until the
// realtime combat/content pipeline authors per-monster tuning in data. These
// values are server-side defaults, not client gameplay state.
package ai

import "strings"

const (
	DefaultThinkTicks          = 5        // 4 Hz at a 20 Hz zone.
	DefaultRespawnTicks        = 20 * 30  // 30 seconds for normal pack monsters
	EliteRespawnTicks          = 20 * 60  // 60 seconds for elite monsters
	BossRespawnTicks           = 20 * 120 // 2 minutes for boss monsters
	DefaultAttackCooldownTicks = 20
)

type Stats struct {
	Speed               float64
	Radius              float64
	AggroRadius         float64
	LeashRadius         float64
	AttackRange         float64
	PatrolRadius        float64
	AttackCooldownTicks uint32
	RespawnTicks        uint32
	SplashRadius        float64
	SplashDamage        float64
}

// DefaultStats derives movement/AI budgets from the Monster.csv type column.
func DefaultStats(templateType string) Stats {
	switch strings.ToLower(strings.TrimSpace(templateType)) {
	case "boss", "tribulation_boss":
		return Stats{Speed: 2.6, Radius: 0.8, AggroRadius: 14, LeashRadius: 24, AttackRange: 1.8, PatrolRadius: 3, AttackCooldownTicks: 30, RespawnTicks: BossRespawnTicks, SplashRadius: 4, SplashDamage: 10}
	case "elite", "tribulation_monster":
		return Stats{Speed: 2.9, Radius: 0.6, AggroRadius: 12, LeashRadius: 20, AttackRange: 1.5, PatrolRadius: 4, AttackCooldownTicks: 25, RespawnTicks: EliteRespawnTicks}
	default:
		return Stats{Speed: 3.2, Radius: 0.5, AggroRadius: 10, LeashRadius: 16, AttackRange: 1.25, PatrolRadius: 5, AttackCooldownTicks: DefaultAttackCooldownTicks, RespawnTicks: DefaultRespawnTicks}
	}
}
