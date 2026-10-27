package persistence

// CombatStateVersion identifies the durable combat-snapshot layout. Bump it
// whenever a field changes meaning so an old row is ignored instead of being
// restored with the wrong semantics.
const CombatStateVersion = 1

// CombatState is the durable part of a player's zone combat resources. All
// durations are stored as tick counts for the fixed 20 Hz zone clock; the
// gateway converts them to remaining ticks when a session rejoins.
//
// Mana and Gu hunger stay out of this snapshot on purpose: mana is settled as
// an additive delta against players.current_mana and hunger is settled against
// player_gu, so replaying a zone snapshot would double-count them.
type CombatState struct {
	Version                   int           `json:"version"`
	GCDRemainingTicks         uint64        `json:"gcd_remaining_ticks,omitempty"`
	BasicAttackRemainingTicks uint64        `json:"basic_attack_remaining_ticks,omitempty"`
	CooldownRemainingTicks    [7]uint64     `json:"cooldown_remaining_ticks"`
	CooldownSkillIDs          [7]string     `json:"cooldown_skill_ids"`
	Statuses                  []StatusState `json:"statuses,omitempty"`
	Hidden                    HiddenState   `json:"hidden,omitempty"`
	RecoveryRemainingTicks    uint64        `json:"recovery_remaining_ticks,omitempty"`
	RecoveryPolicy            string        `json:"recovery_policy,omitempty"`
}

// StatusState is one durable timed status. Durations are server ticks; the
// source may be gone when the player rejoins, so it is informational only.
type StatusState struct {
	ID                  string  `json:"id"`
	SourceID            string  `json:"source_id,omitempty"`
	RemainingTicks      uint64  `json:"remaining_ticks"`
	DurationTicks       uint64  `json:"duration_ticks"`
	PeriodTicks         uint64  `json:"period_ticks,omitempty"`
	NextTickInTicks     uint64  `json:"next_tick_in_ticks,omitempty"`
	DamagePerTick       float64 `json:"damage_per_tick,omitempty"`
	HealPerTick         float64 `json:"heal_per_tick,omitempty"`
	MoveMultiplier      float64 `json:"move_multiplier,omitempty"`
	ManaDrainPerTick    float64 `json:"mana_drain_per_tick,omitempty"`
	ThoughtDrainPerTick float64 `json:"thought_drain_per_tick,omitempty"`
	Stacks              int     `json:"stacks,omitempty"`
	MaxStacks           int     `json:"max_stacks,omitempty"`
	StackPolicy         string  `json:"stack_policy,omitempty"`
	Debuff              bool    `json:"debuff,omitempty"`
	PvPDrCategory       string  `json:"pvp_dr_category,omitempty"`
}

// HiddenState keeps the discovered hidden-passive counters so a rejoin cannot
// reset its internal cooldown or hit counter.
type HiddenState struct {
	ID                  string  `json:"id,omitempty"`
	HPPercentLTE        float64 `json:"hp_percent_lte,omitempty"`
	RequiredHits        int     `json:"required_hits,omitempty"`
	Shield              float64 `json:"shield,omitempty"`
	Heal                float64 `json:"heal,omitempty"`
	ICDTicks            uint64  `json:"icd_ticks,omitempty"`
	HitCount            int     `json:"hit_count,omitempty"`
	ReadyRemainingTicks uint64  `json:"ready_remaining_ticks,omitempty"`
	Discovered          bool    `json:"discovered,omitempty"`
	TriggeredOnce       bool    `json:"triggered_once,omitempty"`
}

// Empty reports whether the snapshot carries no restorable runtime state.
func (c CombatState) Empty() bool {
	if c.Version != CombatStateVersion {
		return true
	}
	if c.GCDRemainingTicks > 0 || c.BasicAttackRemainingTicks > 0 ||
		c.RecoveryRemainingTicks > 0 || len(c.Statuses) > 0 || c.Hidden.ID != "" {
		return false
	}
	for _, remaining := range c.CooldownRemainingTicks {
		if remaining > 0 {
			return false
		}
	}
	return true
}
