// Package effects owns the typed gameplay-effect seam. Definitions are mutable
// authoring DTOs; Compiled is the immutable representation used at runtime.
package effects

import "encoding/json"

type Trigger string

const (
	TriggerAlwaysOn          Trigger = "ALWAYS_ON"
	TriggerOnAction          Trigger = "ON_ACTION"
	TriggerOnHit             Trigger = "ON_HIT"
	TriggerBeforeHit         Trigger = "BEFORE_HIT"
	TriggerOnTurnStart       Trigger = "ON_TURN_START"
	TriggerOnDeath           Trigger = "ON_DEATH"
	TriggerOnTick            Trigger = "ON_TICK"
	TriggerCapability        Trigger = "CAPABILITY"
	TriggerCombatPassive     Trigger = "COMBAT_PASSIVE"
	TriggerPassive           Trigger = "PASSIVE"
	TriggerOnBattleStart     Trigger = "ON_BATTLE_START"
	TriggerBeforeEnemyAttack Trigger = "BEFORE_ENEMY_ATTACK"
	TriggerOnCast            Trigger = "ON_CAST"
	TriggerOnConsume         Trigger = "ON_CONSUME"
	TriggerOnUse             Trigger = "ON_USE"
)

type Selector string

const (
	TargetSelf       Selector = "SELF"
	TargetSingle     Selector = "TARGET"
	TargetRow        Selector = "ROW"
	TargetCol        Selector = "COL"
	TargetAllEnemies Selector = "ALL_ENEMIES"
	TargetAllAllies  Selector = "ALL_ALLIES"
	TargetAll        Selector = "ALL"
)

type Target struct {
	Selector        Selector `json:"selector"`
	BypassFrontline bool     `json:"bypass_frontline,omitempty"`
}
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}
type OperationType string

const (
	OpDamage                OperationType = "DAMAGE"
	OpBonusStrike           OperationType = "BONUS_STRIKE"
	OpChainDamage           OperationType = "CHAIN_DAMAGE"
	OpDamageModifier        OperationType = "DAMAGE_MODIFIER"
	OpDefensePenetration    OperationType = "DEFENSE_PENETRATION"
	OpTargetDamageModifier  OperationType = "TARGET_DAMAGE_MODIFIER"
	OpMitigateDamage        OperationType = "MITIGATE_DAMAGE"
	OpDetonateStatus        OperationType = "DETONATE_STATUS"
	OpHPCost                OperationType = "HP_COST"
	OpShield                OperationType = "SHIELD"
	OpHeal                  OperationType = "HEAL"
	OpDirectHP              OperationType = "DIRECT_HP"
	OpStatMod               OperationType = "STAT_MOD"
	OpSetRule               OperationType = "SET_RULE"
	OpTransferResource      OperationType = "TRANSFER_RESOURCE"
	OpResourceConversion    OperationType = "RESOURCE_CONVERSION"
	OpStatus                OperationType = "STATUS"
	OpBarrier               OperationType = "BARRIER"
	OpCleanse               OperationType = "CLEANSE"
	OpDispel                OperationType = "DISPEL"
	OpRequire               OperationType = "REQUIRE"
	OpGrantResource         OperationType = "GRANT_RESOURCE"
	OpRestoreResource       OperationType = "RESTORE_RESOURCE"
	OpModifyMaxResource     OperationType = "MODIFY_MAX_RESOURCE"
	OpModifyAptitude        OperationType = "MODIFY_APTITUDE"
	OpGrantBodyMod          OperationType = "GRANT_BODY_MOD"
	OpAdvanceProgression    OperationType = "ADVANCE_PROGRESSION"
	OpSummon                OperationType = "SUMMON"
	OpAdvanceMajorRealm     OperationType = "ADVANCE_MAJOR_REALM"
	OpPetrifyAperture       OperationType = "PETRIFY_APERTURE"
	OpModifyAperture        OperationType = "MODIFY_APERTURE"
	OpRandomizeTrait        OperationType = "RANDOMIZE_TRAIT"
	OpRepairAperture        OperationType = "REPAIR_APERTURE"
	OpOpenRewardContainer   OperationType = "OPEN_REWARD_CONTAINER"
	OpPlaceWorldObject      OperationType = "PLACE_WORLD_OBJECT"
	OpDisplace              OperationType = "DISPLACE"
	OpHitRule               OperationType = "HIT_RULE"
	OpTurnOrderRule         OperationType = "TURN_ORDER_RULE"
	OpGrantCapability       OperationType = "GRANT_CAPABILITY"
	OpGrowthDuration        OperationType = "GROWTH_DURATION"
	OpPeriodicResource      OperationType = "PERIODIC_RESOURCE"
	OpDeductionFinishChance OperationType = "DEDUCTION_FINISH_CHANCE"
	OpNotify                OperationType = "NOTIFY"
	OpApplyStackingStatus   OperationType = "APPLY_STACKING_STATUS"
	OpCounterAdd            OperationType = "COUNTER_ADD"
	OpRevokeStatus          OperationType = "REVOKE_STATUS"
	OpThresholdCheck        OperationType = "THRESHOLD_CHECK"
	OpThresholdRecovery     OperationType = "THRESHOLD_RECOVERY"
	OpTenacityModifier      OperationType = "TENACITY_MODIFIER"
	OpStatusImmunity        OperationType = "STATUS_IMMUNITY"
	OpSlotInteraction       OperationType = "SLOT_INTERACTION"
	OpRevive                OperationType = "REVIVE"
	OpEffectImmunity        OperationType = "EFFECT_IMMUNITY"
	OpRankScaledDebuff      OperationType = "RANK_SCALED_DEBUFF"
	OpGuSeal                OperationType = "GU_SEAL"
	OpEndBattle             OperationType = "END_BATTLE"
	OpTargetRedirection     OperationType = "TARGET_REDIRECTION"
	OpOutcomeRoulette       OperationType = "OUTCOME_ROULETTE"
	OpUnique                OperationType = "UNIQUE"
)

type StackingStatusTier struct {
	Tier        int     `json:"tier"`
	StatPercent float64 `json:"stat_percent"`
}

type Operation struct {
	Op              OperationType        `json:"op"`
	SubType         string               `json:"sub_type,omitempty"`
	Target          *Target              `json:"target,omitempty"`
	Value           float64              `json:"value,omitempty"`
	PerMissingHP    float64              `json:"per_missing_hp,omitempty"`
	PerTargetDebuff float64              `json:"per_target_debuff,omitempty"`
	PerMinion       float64              `json:"per_minion,omitempty"`
	MinMultiplier   float64              `json:"min_multiplier,omitempty"`
	PerStack        float64              `json:"per_stack,omitempty"`
	PerStrength     float64              `json:"per_strength,omitempty"`
	MaxHPPercent    float64              `json:"max_hp_percent,omitempty"`
	MinDamage       float64              `json:"min_damage,omitempty"`
	TargetCount     int                  `json:"target_count,omitempty"`
	MinCost         float64              `json:"min_cost,omitempty"`
	MinRemaining    float64              `json:"min_remaining,omitempty"`
	MinAmount       float64              `json:"min_amount,omitempty"`
	Enabled         *bool                `json:"enabled,omitempty"`
	Turns           int                  `json:"turns,omitempty"`
	Chance          float64              `json:"chance,omitempty"`
	Path            string               `json:"path,omitempty"`
	StatusID        string               `json:"status_id,omitempty"`
	Stat            string               `json:"stat,omitempty"`
	Unit            string               `json:"unit,omitempty"`
	Resource        string               `json:"resource,omitempty"`
	StackPolicy     string               `json:"stack_policy,omitempty"`
	ApplicationMode string               `json:"application_mode,omitempty"`
	Phase           string               `json:"phase,omitempty"`
	Message         string               `json:"message,omitempty"`
	LogAction       string               `json:"log_action,omitempty"`
	SuppressLog     bool                 `json:"suppress_log,omitempty"`
	RankSuppression *bool                `json:"rank_suppression,omitempty"`
	Code            string               `json:"code,omitempty"`
	Params          json.RawMessage      `json:"params,omitempty"`
	StatusKey       string               `json:"status_key,omitempty"`
	MaxStacks       int                  `json:"max_stacks,omitempty"`
	DurationHours   float64              `json:"duration_hours,omitempty"`
	DispelRule      string               `json:"dispel_rule,omitempty"`
	PvPOnly         bool                 `json:"pvp_only,omitempty"`
	TierScaling     []StackingStatusTier `json:"tier_scaling,omitempty"`
	Key             string               `json:"key,omitempty"`
	Delta           int                  `json:"delta,omitempty"`
	Min             int                  `json:"min,omitempty"`
	Max             int                  `json:"max,omitempty"`
	Threshold       int                  `json:"threshold,omitempty"`
	HPThreshold     float64              `json:"hp_threshold,omitempty"`
	HealPercent     float64              `json:"heal_percent,omitempty"`
	MaxUses         int                  `json:"max_uses,omitempty"`
	CleanseDebuffs  bool                 `json:"cleanse_debuffs,omitempty"`
	AutoReset       bool                 `json:"auto_reset,omitempty"`
}
type Block struct {
	ID         string      `json:"id,omitempty"`
	Trigger    Trigger     `json:"trigger"`
	Target     Target      `json:"target"`
	Conditions []Condition `json:"conditions,omitempty"`
	Operations []Operation `json:"operations"`
}
