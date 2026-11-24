package effects

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// Limits are enforced for every evaluation, independently of authored data.
type Limits struct{ MaxEventsPerAction, MaxTriggerDepth, MaxRepeats, MaxSummons int }

var DefaultLimits = Limits{MaxEventsPerAction: 128, MaxTriggerDepth: 8, MaxRepeats: 16, MaxSummons: 16}

type Snapshot struct {
	ID                       string
	Race                     string
	HP, MaxHP, Mana, MaxMana float64
	Stats                    map[string]float64
}
type Context struct {
	RNG                             *rand.Rand
	Actor, Target                   Snapshot
	Rank, Mastery, Stacks           int
	HasRank, HasMastery, HasStacks  bool
	DaoMarks                        map[string]int64
	Path                            []string
	CatalogRevision                 uint64
	Depth, Events, Repeats, Summons int
}
type Event struct {
	Trigger Trigger
	Turn    int
	HasTurn bool
}
type Domain string

const (
	DomainCombat     Domain = "combat"
	DomainDerived    Domain = "derived_player"
	DomainCapability Domain = "capability"
	DomainWorld      Domain = "world"
	DomainGuUse      Domain = "gu_use"
	DomainItemUse    Domain = "item_use"
	DomainGrowth     Domain = "growth"
	DomainRefinement Domain = "refinement"
)

type Command struct {
	Domain                                                 Domain
	Operation                                              OperationType
	Code, SubType, Stat, Path, Unit, Resource, StackPolicy string
	Message                                                string
	LogAction                                              string
	Phase                                                  string
	SuppressLog                                            bool
	Target                                                 Selector
	Value                                                  float64
	MinCost                                                float64
	MinRemaining                                           float64
	MinAmount                                              float64
	Enabled                                                bool
	Params                                                 []byte
	Values                                                 map[string]any
	Status                                                 *statusfx.StatusOp
}
type Result struct {
	Commands  []Command
	Truncated bool
}
type Passive struct {
	Stat, Mode string
	Value      float64
}
type Capability struct {
	Code   string
	Params []byte
}
type Growth struct {
	Code   string
	Value  float64
	Params []byte
}
type CombatAction struct {
	Target     Target
	Operations []CompiledOperation
}

// DefinitionCompiler is the small compilation seam used by registry loaders.
type DefinitionCompiler interface {
	Compile([]Block) (*Compiled, error)
}

// Query exposes typed projections without leaking authored JSON.
type Query interface {
	Passives() []Passive
	Capabilities() []Capability
	Growth() []Growth
	Operations(Trigger) []CompiledOperation
}

// Evaluator is pure: it returns commands and never owns a database handle.
type Evaluator interface {
	CompileCombatAction() CombatAction
	Evaluate(Context, Event) (Result, error)
}

type Compiler struct{}
type Compiled struct{ blocks []compiledBlock }
type compiledBlock struct {
	id         string
	trigger    Trigger
	target     Target
	conditions []Condition
	operations []CompiledOperation
}
type CompiledOperation struct {
	BlockID                     string
	BlockIndex                  int
	OperationIndex              int
	Type                        OperationType
	Trigger                     Trigger
	SubType                     string
	Conditions                  []Condition
	Target                      *Target
	Value                       float64
	Chance                      float64
	PerMissingHP                float64
	PerTargetDebuff             float64
	PerMinion                   float64
	MinMultiplier               float64
	PerStack                    float64
	PerStrength                 float64
	MaxHPPercent                float64
	MinDamage                   float64
	TargetCount                 int
	MinCost                     float64
	MinRemaining                float64
	MinAmount                   float64
	Enabled                     bool
	Path, Stat, Code, StatusID  string
	Unit, Resource, StackPolicy string
	ApplicationMode             string
	Phase                       string
	Message                     string
	LogAction                   string
	SuppressLog                 bool
	RankSuppression             *bool
	Params                      []byte
	Values                      map[string]any
	Status                      *statusfx.StatusOp
	StatusKey                   string
	MaxStacks                   int
	DurationHours               float64
	DispelRule                  string
	PvPOnly                     bool
	TierScaling                 []StackingStatusTier
	Key                         string
	Delta                       int
	Min, Max                    int
	Threshold                   int
	HPThreshold                 float64
	HealPercent                 float64
	MaxUses                     int
	CleanseDebuffs              bool
	AutoReset                   bool
	GuSeal                      *effectcatalog.GuSealParams
	TargetRedirection           *effectcatalog.TargetRedirectionParams
	DispelToResources           *effectcatalog.DispelToResourcesParams
}

func (Compiler) Compile(input []Block) (*Compiled, error) {
	blocks := make([]compiledBlock, 0, len(input))
	for blockIndex, b := range input {
		if !effectcatalog.TriggerCatalog[string(b.Trigger)] {
			return nil, fmt.Errorf("effect %q: unknown trigger %q", b.ID, b.Trigger)
		}
		if !effectcatalog.TargetCatalog[string(b.Target.Selector)] {
			return nil, fmt.Errorf("effect %q: unknown selector %q", b.ID, b.Target.Selector)
		}
		conditions := append([]Condition(nil), b.Conditions...)
		for i := range conditions {
			conditions[i].Field = strings.ToUpper(strings.TrimSpace(conditions[i].Field))
			conditions[i].Operator = strings.ToUpper(strings.TrimSpace(conditions[i].Operator))
			value, err := effectcatalog.NormalizeConditionValue(conditions[i].Field, conditions[i].Operator, conditions[i].Value)
			if err != nil {
				return nil, fmt.Errorf("effect %q: %w", b.ID, err)
			}
			conditions[i].Value = value
		}
		cb := compiledBlock{id: b.ID, trigger: b.Trigger, target: b.Target, conditions: conditions}
		for operationIndex, op := range b.Operations {
			if len(conditions) > 0 && (op.Op == OpSummon || (op.Op == OpDisplace && b.Trigger == TriggerOnBattleStart)) {
				return nil, fmt.Errorf("effect %q: %s does not support block conditions in this combat adapter", b.ID, op.Op)
			}
			if op.Op == OpRevokeStatus && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: REVOKE_STATUS does not support block conditions", b.ID)
			}
			if op.SuppressLog && op.Op != OpStatus && op.Op != OpHeal {
				return nil, fmt.Errorf("effect %q: suppress_log is only supported by STATUS or HEAL", b.ID)
			}
			definition, knownOperation := effectcatalog.OperationCatalog[string(op.Op)]
			if !knownOperation {
				return nil, fmt.Errorf("effect %q: unknown operation %q", b.ID, op.Op)
			}
			if !definition.Triggers[string(b.Trigger)] {
				return nil, fmt.Errorf("effect %q: operation %q does not support trigger %q", b.ID, op.Op, b.Trigger)
			}
			effectiveTarget := string(b.Target.Selector)
			if op.Target != nil {
				effectiveTarget = string(op.Target.Selector)
			}
			if !definition.Targets[effectiveTarget] {
				return nil, fmt.Errorf("effect %q: operation %q does not support target %q", b.ID, op.Op, effectiveTarget)
			}
			subType := strings.ToUpper(op.SubType)
			if op.SuppressLog && op.Op == OpHeal && (subType != "RESOURCE" || !strings.EqualFold(op.Resource, "MANA") || effectiveTarget != string(TargetAllAllies) || b.Trigger != TriggerOnAction) {
				return nil, fmt.Errorf("effect %q: HEAL suppress_log requires ON_ACTION MANA RESOURCE targeting ALL_ALLIES", b.ID)
			}
			if subType != "" && len(definition.SubTypes) > 0 && !definition.SubTypes[subType] {
				return nil, fmt.Errorf("effect %q: unknown subtype %q for %q", b.ID, op.SubType, op.Op)
			}
			if err := effectcatalog.ValidateOperationCombination(string(op.Op), subType, string(b.Trigger), effectiveTarget); err != nil {
				return nil, fmt.Errorf("effect %q: %w", b.ID, err)
			}
			if op.Op == OpRevokeStatus {
				if err := effectcatalog.ValidatePersistentStatusID(op.StatusID); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpCounterAdd {
				if err := effectcatalog.ValidateCounterAdd(op.Key, op.Delta, op.Min, op.Max); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpStatMod && subType == "ADD_COUNTER_DELTA" {
				if err := effectcatalog.ValidateCounterStatModifier(op.Stat, op.Key, op.PerStack); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpThresholdRecovery {
				if err := effectcatalog.ValidateThresholdRecovery(op.HPThreshold, op.HealPercent, op.MaxUses, op.CleanseDebuffs, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: THRESHOLD_RECOVERY does not support block conditions", b.ID)
				}
			}
			if op.Op == OpTenacityModifier {
				if len(conditions) != 0 || op.Value < 0 || op.PerMissingHP <= 0 {
					return nil, fmt.Errorf("effect %q: TENACITY_MODIFIER requires non-negative value, positive per_missing_hp and no conditions", b.ID)
				}
			}
			if op.Op == OpStatusImmunity {
				if len(conditions) == 0 {
					return nil, fmt.Errorf("effect %q: STATUS_IMMUNITY requires conditions", b.ID)
				}
				for _, condition := range conditions {
					if condition.Field != "HP_PERCENT" {
						return nil, fmt.Errorf("effect %q: STATUS_IMMUNITY conditions must use HP_PERCENT", b.ID)
					}
				}
			}
			if op.Op == OpApplyStackingStatus {
				tiers := make([]effectcatalog.StackingStatusTierValue, len(op.TierScaling))
				for i, tier := range op.TierScaling {
					tiers[i] = effectcatalog.StackingStatusTierValue{Tier: tier.Tier, StatPercent: tier.StatPercent}
				}
				if err := effectcatalog.ValidateStackingStatus(op.StatusKey, op.MaxStacks, op.DurationHours, op.DispelRule, op.PvPOnly, tiers); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpDamageModifier && subType == "MINION_COUNT" && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: DAMAGE_MODIFIER MINION_COUNT does not support block conditions", b.ID)
			}
			if op.Op == OpDetonateStatus && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: DETONATE_STATUS does not support block conditions", b.ID)
			}
			if op.Op == OpHPCost && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: HP_COST does not support block conditions", b.ID)
			}
			if op.Op == OpDeductionFinishChance && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: DEDUCTION_FINISH_CHANCE does not support block conditions", b.ID)
			}
			if op.Op == OpCounterAdd && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: COUNTER_ADD does not support block conditions", b.ID)
			}
			if op.Op == OpApplyStackingStatus && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: APPLY_STACKING_STATUS does not support block conditions", b.ID)
			}
			if op.Op == OpApplyStackingStatus && !op.PvPOnly {
				return nil, fmt.Errorf("effect %q: APPLY_STACKING_STATUS currently supports persistent PvP status only", b.ID)
			}
			if op.Op == OpTransferResource && strings.EqualFold(op.Resource, "HP") && len(conditions) > 0 {
				return nil, fmt.Errorf("effect %q: HP TRANSFER_RESOURCE does not support block conditions", b.ID)
			}
			if op.Op == OpModifyAptitude {
				if err := effectcatalog.ValidateModifyAptitude(subType, op.Value); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpHitRule {
				if subType == "COUNTER_THRESHOLD" && len(conditions) > 0 {
					return nil, fmt.Errorf("effect %q: HIT_RULE COUNTER_THRESHOLD does not support conditions", b.ID)
				}
				if err := effectcatalog.ValidateHitRuleParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpSlotInteraction {
				if err := effectcatalog.ValidateSlotInteractionParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: SLOT_INTERACTION does not support conditions", b.ID)
				}
			}
			if op.Op == OpRevive {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: REVIVE does not support conditions", b.ID)
				}
				if err := effectcatalog.ValidateReviveParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: invalid REVIVE params: %w", b.ID, err)
				}
			}
			if op.Op == OpEffectImmunity && len(conditions) != 0 {
				return nil, fmt.Errorf("effect %q: EFFECT_IMMUNITY does not support conditions", b.ID)
			}
			if op.Op == OpRankScaledDebuff {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: RANK_SCALED_DEBUFF does not support conditions", b.ID)
				}
				if err := effectcatalog.ValidateRankScaledDebuffParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			var guSealParams *effectcatalog.GuSealParams
			if op.Op == OpGuSeal {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: GU_SEAL does not support block conditions", b.ID)
				}
				params, err := effectcatalog.ParseGuSealParams(op.Params)
				if err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				guSealParams = &params
			}
			var targetRedirectionParams *effectcatalog.TargetRedirectionParams
			if op.Op == OpTargetRedirection {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: TARGET_REDIRECTION does not support block conditions", b.ID)
				}
				params, err := effectcatalog.ParseTargetRedirectionParams(op.Params)
				if err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				targetRedirectionParams = &params
			}
			var dispelToResourcesParams *effectcatalog.DispelToResourcesParams
			if op.Op == OpResourceConversion && strings.EqualFold(subType, "DISPEL_TO_RESOURCES") {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: RESOURCE_CONVERSION DISPEL_TO_RESOURCES does not support block conditions", b.ID)
				}
				params, err := effectcatalog.ParseDispelToResourcesParams(op.Params)
				if err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				dispelToResourcesParams = &params
			}
			if op.Op == OpOutcomeRoulette {
				if len(conditions) != 0 {
					return nil, fmt.Errorf("effect %q: OUTCOME_ROULETTE does not support block conditions", b.ID)
				}
				if err := effectcatalog.ValidateOutcomeRouletteParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpTurnOrderRule {
				if err := effectcatalog.ValidateTurnOrderRuleParams(subType, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpEndBattle {
				if op.LogAction != "" {
					if err := effectcatalog.ValidateCombatLogAction(op.LogAction); err != nil {
						return nil, fmt.Errorf("effect %q: %w", b.ID, err)
					}
				}
				if op.Message != "" {
					if err := effectcatalog.ValidateNotifyMessage(op.Message); err != nil {
						return nil, fmt.Errorf("effect %q: %w", b.ID, err)
					}
				}
			}
			if op.Op == OpDamageModifier {
				if err := effectcatalog.ValidateDamageModifier(subType, op.Value, op.PerMissingHP, op.PerTargetDebuff, op.PerMinion, op.PerStack, op.Key, op.StatusID, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				phase := strings.ToUpper(op.Phase)
				validPhase := phase == "" || phase == "EARLY_RIDER" || phase == "LATE_RIDER"
				if subType == "COUNTER_COUNT" {
					validPhase = phase == "FINAL_RIDER" && b.Trigger == TriggerCombatPassive
				}
				if !validPhase || (phase != "" && b.Trigger != TriggerCombatPassive) {
					return nil, fmt.Errorf("effect %q: DAMAGE_MODIFIER COUNTER_COUNT requires FINAL_RIDER on COMBAT_PASSIVE; other phases are EARLY_RIDER or LATE_RIDER", b.ID)
				}
				if b.Trigger == TriggerCombatPassive && len(conditions) > 0 && op.Phase == "" {
					return nil, fmt.Errorf("effect %q: conditional COMBAT_PASSIVE DAMAGE_MODIFIER requires phase LATE_RIDER", b.ID)
				}
			}
			if op.Op == OpHeal && subType == "COUNTER_PERCENT" {
				if err := effectcatalog.ValidateCounterHeal(op.Key, op.Value); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if b.Trigger != TriggerCombatPassive || strings.ToUpper(op.Phase) != "FINAL_RIDER" {
					return nil, fmt.Errorf("effect %q: HEAL COUNTER_PERCENT requires FINAL_RIDER on COMBAT_PASSIVE", b.ID)
				}
			}
			if op.Op == OpTargetDamageModifier {
				if err := effectcatalog.ValidateTargetDamageModifier(subType, op.Value, op.MinMultiplier, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpDefensePenetration {
				if err := effectcatalog.ValidateDefensePenetration(op.Value); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpBonusStrike {
				if err := effectcatalog.ValidateBonusStrike(op.Chance, op.Value, op.MinDamage, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpChainDamage {
				if err := effectcatalog.ValidateChainDamage(op.Value, op.MinDamage, op.TargetCount, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpMitigateDamage {
				if len(conditions) > 0 {
					return nil, fmt.Errorf("effect %q: MITIGATE_DAMAGE does not support block conditions", b.ID)
				}
				if err := effectcatalog.ValidateMitigateDamage(subType, op.Value, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpCleanse && subType == "STATUS" {
				if err := effectcatalog.ValidateCombatStatusID(op.StatusID); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if op.Phase != "" && (strings.ToUpper(op.Phase) != "LATE_RIDER" || b.Trigger != TriggerCombatPassive) {
					return nil, fmt.Errorf("effect %q: CLEANSE phase LATE_RIDER requires COMBAT_PASSIVE STATUS", b.ID)
				}
				if b.Trigger == TriggerCombatPassive && op.Phase == "" {
					return nil, fmt.Errorf("effect %q: COMBAT_PASSIVE CLEANSE STATUS requires phase LATE_RIDER", b.ID)
				}
			}
			if op.Op == OpDetonateStatus {
				if err := effectcatalog.ValidateStatusDetonation(op.StatusID, op.PerStack, op.PerStrength, op.MaxHPPercent, op.MinDamage, strings.ToUpper(op.Phase)); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if err := effectcatalog.ValidateCombatLogAction(op.LogAction); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if err := effectcatalog.ValidateNotifyMessage(op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpHPCost {
				if err := effectcatalog.ValidateHPCost(subType, op.Value, op.MinCost, op.MinRemaining, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if op.Phase != "" && ((strings.ToUpper(op.Phase) != "EARLY_RIDER" && strings.ToUpper(op.Phase) != "PRE_RIDER") || b.Trigger != TriggerOnAction) {
					return nil, fmt.Errorf("effect %q: HP_COST phase PRE_RIDER or EARLY_RIDER requires ON_ACTION", b.ID)
				}
			}
			if op.Op == OpDispel {
				if err := effectcatalog.ValidateDispel(subType, op.Value, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpDeductionFinishChance {
				if err := effectcatalog.ValidateDeductionFinishChanceParams(op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			effectiveOpTarget := b.Target
			if op.Target != nil {
				effectiveOpTarget = *op.Target
			}
			var rankSuppression *bool
			if op.RankSuppression != nil {
				value := *op.RankSuppression
				rankSuppression = &value
			}
			statusID := strings.ToUpper(strings.TrimSpace(op.StatusID))
			if op.Op == OpRevokeStatus {
				statusID = strings.TrimSpace(op.StatusID)
			}
			co := CompiledOperation{
				BlockID: b.ID, BlockIndex: blockIndex, OperationIndex: operationIndex,
				Type: op.Op, Trigger: b.Trigger, SubType: subType, Conditions: append([]Condition(nil), conditions...),
				Target: &effectiveOpTarget, Value: op.Value, Chance: op.Chance, PerMissingHP: op.PerMissingHP, PerTargetDebuff: op.PerTargetDebuff, PerMinion: op.PerMinion, MinMultiplier: op.MinMultiplier, PerStack: op.PerStack, PerStrength: op.PerStrength, MaxHPPercent: op.MaxHPPercent, MinDamage: op.MinDamage, TargetCount: op.TargetCount, MinCost: op.MinCost, MinRemaining: op.MinRemaining, MinAmount: op.MinAmount, Enabled: op.Enabled != nil && *op.Enabled,
				Path: op.Path, Stat: strings.ToUpper(op.Stat), StatusID: statusID, Unit: strings.ToUpper(op.Unit),
				Resource: strings.ToUpper(op.Resource), StackPolicy: strings.ToUpper(op.StackPolicy),
				ApplicationMode: strings.ToUpper(strings.TrimSpace(op.ApplicationMode)),
				Phase:           strings.ToUpper(strings.TrimSpace(op.Phase)),
				Message:         strings.TrimSpace(op.Message), LogAction: strings.ToUpper(strings.TrimSpace(op.LogAction)),
				SuppressLog: op.SuppressLog, RankSuppression: rankSuppression, Code: strings.ToUpper(op.Code), Params: append([]byte(nil), op.Params...),
				StatusKey: strings.ToLower(strings.TrimSpace(op.StatusKey)), MaxStacks: op.MaxStacks,
				DurationHours: op.DurationHours, DispelRule: strings.ToUpper(strings.TrimSpace(op.DispelRule)),
				PvPOnly: op.PvPOnly, TierScaling: append([]StackingStatusTier(nil), op.TierScaling...),
				Key: strings.TrimSpace(op.Key), Delta: op.Delta, Min: op.Min, Max: op.Max,
				Threshold: op.Threshold, HPThreshold: op.HPThreshold, HealPercent: op.HealPercent,
				MaxUses: op.MaxUses, CleanseDebuffs: op.CleanseDebuffs, AutoReset: op.AutoReset,
				GuSeal:            guSealParams,
				TargetRedirection: targetRedirectionParams,
				DispelToResources: dispelToResourcesParams,
			}
			if op.Op == OpSetRule && op.Enabled == nil {
				return nil, fmt.Errorf("effect %q: SET_RULE enabled is required", b.ID)
			}
			if op.Op == OpGrantCapability {
				co.Code = subType
			}
			if len(op.Params) > 0 {
				if err := json.Unmarshal(op.Params, &co.Values); err != nil {
					return nil, fmt.Errorf("effect %q: invalid native params: %w", b.ID, err)
				}
			}
			if op.Target != nil {
				t := *op.Target
				if !effectcatalog.TargetCatalog[string(t.Selector)] {
					return nil, fmt.Errorf("effect %q: unknown selector %q", b.ID, t.Selector)
				}
				co.Target = &t
			}
			if op.Op == OpStatus {
				if co.ApplicationMode != "" && co.ApplicationMode != "DIRECT" {
					return nil, fmt.Errorf("effect %q: unknown STATUS application_mode %q", b.ID, op.ApplicationMode)
				}
				if co.Phase != "" && co.Phase != "PRE_DAMAGE" && co.Phase != "PRE_CHAIN" {
					return nil, fmt.Errorf("effect %q: unknown STATUS phase %q", b.ID, op.Phase)
				}
				resolved, err := statusfx.ResolveStatus(op.StatusID, op.Value, op.Turns)
				if err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				resolved.Chance = op.Chance
				resolved.ApplicationMode = co.ApplicationMode
				resolved.Phase = co.Phase
				resolved.Message = co.Message
				resolved.LogAction = co.LogAction
				resolved.SuppressLog = co.SuppressLog
				resolved.ApplyToTargets = effectiveTarget != string(TargetSelf)
				resolved.EffectID = fmt.Sprintf("%s:%d:%d", b.ID, blockIndex, operationIndex)
				resolved.StackPolicy = co.StackPolicy
				if co.Message != "" {
					if err := effectcatalog.ValidateNotifyMessage(co.Message); err != nil {
						return nil, fmt.Errorf("effect %q: invalid STATUS message: %w", b.ID, err)
					}
					if co.ApplicationMode != "DIRECT" {
						return nil, fmt.Errorf("effect %q: STATUS message requires application_mode DIRECT", b.ID)
					}
				}
				if co.LogAction != "" {
					if err := effectcatalog.ValidateCombatLogAction(co.LogAction); err != nil {
						return nil, fmt.Errorf("effect %q: invalid STATUS log_action: %w", b.ID, err)
					}
					if co.ApplicationMode != "DIRECT" {
						return nil, fmt.Errorf("effect %q: STATUS log_action requires application_mode DIRECT", b.ID)
					}
				}
				if co.SuppressLog && co.ApplicationMode != "DIRECT" {
					return nil, fmt.Errorf("effect %q: STATUS suppress_log requires application_mode DIRECT", b.ID)
				}
				co.Status = &resolved
			}
			if co.Stat != "" && !effectcatalog.StatCatalog[co.Stat] {
				return nil, fmt.Errorf("effect %q: unknown stat %q", b.ID, op.Stat)
			}
			if co.Unit != "" && !effectcatalog.UnitCatalog[co.Unit] {
				return nil, fmt.Errorf("effect %q: unknown unit %q", b.ID, op.Unit)
			}
			if co.Resource != "" && !effectcatalog.ResourceCatalog[co.Resource] {
				return nil, fmt.Errorf("effect %q: unknown resource %q", b.ID, op.Resource)
			}
			if op.Op == OpPeriodicResource {
				if err := effectcatalog.ValidatePeriodicResource(co.SubType, co.Resource); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpTransferResource {
				if err := effectcatalog.ValidateResourceTransfer(co.SubType, co.Resource, co.Value, co.MinAmount, co.Phase, co.LogAction, co.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if co.Message != "" {
					if err := effectcatalog.ValidateNotifyMessage(co.Message); err != nil {
						return nil, fmt.Errorf("effect %q: %w", b.ID, err)
					}
				}
			}
			if op.Op == OpResourceConversion {
				if err := effectcatalog.ValidateResourceConversion(co.SubType, co.Value, co.MinRemaining, co.LogAction, co.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpSummon {
				if err := effectcatalog.ValidateSummonParams(op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpDisplace {
				if err := effectcatalog.ValidateDisplacement(co.SubType, co.Value); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpNotify {
				if err := effectcatalog.ValidateNotifyOperation(string(b.Trigger), effectiveTarget, co.Phase, co.Key, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				if op.LogAction != "" {
					if err := effectcatalog.ValidateCombatLogAction(op.LogAction); err != nil {
						return nil, fmt.Errorf("effect %q: invalid NOTIFY log_action: %w", b.ID, err)
					}
				}
			}
			if co.StackPolicy != "" && !effectcatalog.StackPolicyCatalog[co.StackPolicy] {
				return nil, fmt.Errorf("effect %q: unknown stack policy %q", b.ID, op.StackPolicy)
			}
			if op.Op == OpUnique && effectcatalog.NativeHooks[co.Code].Triggers == nil {
				return nil, fmt.Errorf("effect %q: unknown native mechanic %q", b.ID, op.Code)
			}
			if op.Op == OpUnique {
				hook := effectcatalog.NativeHooks[co.Code]
				if !hook.Triggers[string(b.Trigger)] {
					return nil, fmt.Errorf("effect %q: native mechanic %q does not support trigger %q", b.ID, op.Code, b.Trigger)
				}
				if err := effectcatalog.ValidateNativeParams(co.Code, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			if op.Op == OpGrantCapability {
				hook, ok := effectcatalog.NativeHooks[co.Code]
				if !ok || hook.Domain != string(DomainCapability) {
					return nil, fmt.Errorf("effect %q: unknown capability %q", b.ID, co.Code)
				}
				if !hook.Triggers[string(b.Trigger)] {
					return nil, fmt.Errorf("effect %q: capability %q does not support trigger %q", b.ID, co.Code, b.Trigger)
				}
				if err := effectcatalog.ValidateNativeParams(co.Code, op.Params); err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
			}
			cb.operations = append(cb.operations, co)
		}
		blocks = append(blocks, cb)
	}
	return &Compiled{blocks: blocks}, nil
}

func (c *Compiled) Passives() []Passive {
	var out []Passive
	for _, b := range c.blocks {
		if b.trigger != TriggerAlwaysOn {
			continue
		}
		if len(b.conditions) > 0 {
			continue
		}
		for _, op := range b.operations {
			if op.Type == OpStatMod {
				out = append(out, Passive{op.Stat, op.SubType, op.Value})
			}
		}
	}
	return out
}
func (c *Compiled) Capabilities() []Capability {
	var out []Capability
	for _, b := range c.blocks {
		if b.trigger != TriggerCapability {
			continue
		}
		for _, op := range b.operations {
			if op.Type != OpUnique && op.Type != OpGrantCapability {
				continue
			}
			out = append(out, Capability{op.Code, append([]byte(nil), op.Params...)})
		}
	}
	return out
}

// Capability scans compiled operations and returns a defensive copy of the
// first capability with the requested code. Consumers should query this
// projection instead of interpreting persisted authoring JSON.
func (c *Compiled) Capability(code string) (Capability, bool) {
	if c == nil {
		return Capability{}, false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, block := range c.blocks {
		if block.trigger != TriggerCapability {
			continue
		}
		for _, op := range block.operations {
			if (op.Type == OpUnique || op.Type == OpGrantCapability) && op.Code == code {
				return Capability{Code: op.Code, Params: append([]byte(nil), op.Params...)}, true
			}
		}
	}
	return Capability{}, false
}

// LastNativeParamFloat returns the last numeric native-operation parameter
// matching trigger, code, and key without copying unrelated operations.
func (c *Compiled) LastNativeParamFloat(trigger Trigger, code, key string) (float64, bool) {
	if c == nil {
		return 0, false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	var value float64
	found := false
	for _, block := range c.blocks {
		if block.trigger != trigger {
			continue
		}
		for _, op := range block.operations {
			if op.Type != OpUnique || op.Code != code {
				continue
			}
			if len(op.Conditions) > 0 {
				continue
			}
			if number, ok := op.Values[key].(float64); ok {
				value, found = number, true
			}
		}
	}
	return value, found
}

func (c *Compiled) Growth() []Growth {
	var out []Growth
	for _, b := range c.blocks {
		if b.trigger != TriggerOnTick {
			continue
		}
		for _, op := range b.operations {
			out = append(out, Growth{op.Code, op.Value, append([]byte(nil), op.Params...)})
		}
	}
	return out
}

// GrowthDurationHours returns the last typed growth-duration value, falling
// back to the legacy native hook for definitions that have not been migrated.
func (c *Compiled) GrowthDurationHours() (float64, bool) {
	if c == nil {
		return 0, false
	}
	var value float64
	found := false
	for _, block := range c.blocks {
		if block.trigger != TriggerAlwaysOn {
			continue
		}
		for _, op := range block.operations {
			if op.Type == OpGrowthDuration {
				value, found = op.Value, true
			}
		}
	}
	if found {
		return value, true
	}
	return c.LastNativeParamFloat(TriggerAlwaysOn, "GROWTH_DURATION_HOURS", "value")
}
func (c *Compiled) CompileCombatAction() CombatAction {
	var a CombatAction
	for _, b := range c.blocks {
		if b.trigger != TriggerOnAction {
			continue
		}
		a.Target = b.target
		a.Operations = append(a.Operations, cloneOps(b.operations)...)
	}
	return a
}

// Operations returns a defensive copy of all operations authored for trigger.
// Domain code uses this projection instead of reaching back into the JSON DTO.
func (c *Compiled) Operations(trigger Trigger) []CompiledOperation {
	if c == nil {
		return nil
	}
	var out []CompiledOperation
	for _, block := range c.blocks {
		if block.trigger != trigger {
			continue
		}
		out = append(out, cloneOps(block.operations)...)
	}
	return out
}

// AllOperations returns a defensive copy in authored block/operation order.
func (c *Compiled) AllOperations() []CompiledOperation {
	if c == nil {
		return nil
	}
	var out []CompiledOperation
	for _, block := range c.blocks {
		out = append(out, cloneOps(block.operations)...)
	}
	return out
}

// StatValue sums a typed stat projection for one trigger and subtype. It scans
// immutable compiled operations directly so scalar reads do not clone them.
func (c *Compiled) StatValue(trigger Trigger, stat, subType string) float64 {
	stat, subType = strings.ToUpper(stat), strings.ToUpper(subType)
	var total float64
	if c == nil {
		return total
	}
	for _, block := range c.blocks {
		if block.trigger != trigger {
			continue
		}
		for _, op := range block.operations {
			if len(op.Conditions) == 0 && op.Type == OpStatMod && op.Stat == stat && (subType == "" || op.SubType == subType) {
				total += op.Value
			}
		}
	}
	return total
}
func cloneOps(in []CompiledOperation) []CompiledOperation {
	out := append([]CompiledOperation(nil), in...)
	for i := range out {
		out[i] = cloneOperation(out[i])
	}
	return out
}

func cloneOperation(op CompiledOperation) CompiledOperation {
	op.Params = append([]byte(nil), op.Params...)
	op.Conditions = append([]Condition(nil), op.Conditions...)
	op.TierScaling = append([]StackingStatusTier(nil), op.TierScaling...)
	op.Values = cloneValues(op.Values)
	if op.Target != nil {
		target := *op.Target
		op.Target = &target
	}
	if op.Status != nil {
		status := *op.Status
		op.Status = &status
	}
	return op
}

func cloneValues(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func (c *Compiled) NativeMechanics() []CompiledOperation {
	var out []CompiledOperation
	for _, b := range c.blocks {
		for _, op := range b.operations {
			if op.Type == OpUnique {
				cloned := cloneOperation(op)
				cloned.Trigger = b.trigger
				out = append(out, cloned)
			}
		}
	}
	return out
}

// NativeMechanic returns a defensive copy of the first native operation with
// the requested code without cloning unrelated operations.
func (c *Compiled) NativeMechanic(code string) (CompiledOperation, bool) {
	if c == nil {
		return CompiledOperation{}, false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, block := range c.blocks {
		for _, op := range block.operations {
			if op.Type != OpUnique || op.Code != code {
				continue
			}
			if len(op.Conditions) > 0 {
				continue
			}
			cloned := cloneOperation(op)
			cloned.Trigger = block.trigger
			return cloned, true
		}
	}
	return CompiledOperation{}, false
}

func (c *Compiled) Evaluate(ctx Context, event Event) (Result, error) {
	limits := DefaultLimits
	if ctx.Depth >= limits.MaxTriggerDepth || ctx.Events >= limits.MaxEventsPerAction || ctx.Repeats > limits.MaxRepeats || ctx.Summons > limits.MaxSummons {
		return Result{Truncated: true}, nil
	}
	var r Result
	repeats, summons := ctx.Repeats, ctx.Summons
	values := conditionValues(ctx, event)
	for _, b := range c.blocks {
		if b.trigger != event.Trigger {
			continue
		}
		matches, err := ConditionsMatch(b.conditions, values)
		if err != nil {
			return Result{}, fmt.Errorf("effect %q: %w", b.id, err)
		}
		if !matches {
			continue
		}
		for _, op := range b.operations {
			if ctx.Events+len(r.Commands) >= limits.MaxEventsPerAction {
				r.Truncated = true
				return r, nil
			}
			if value, ok := op.Values["repeats"].(float64); ok {
				repeats += int(value)
			}
			if strings.HasPrefix(op.Code, "SUMMON_") {
				summons++
			}
			if repeats > limits.MaxRepeats || summons > limits.MaxSummons {
				r.Truncated = true
				return r, nil
			}
			domain := domainFor(op)
			target := b.target.Selector
			if op.Target != nil {
				target = op.Target.Selector
			}
			cmd := Command{Domain: domain, Operation: op.Type, Code: op.Code, SubType: op.SubType, Stat: op.Stat, Path: op.Path, Unit: op.Unit, Resource: op.Resource, StackPolicy: op.StackPolicy, Message: op.Message, LogAction: op.LogAction, Phase: op.Phase, SuppressLog: op.SuppressLog, Target: target, Value: op.Value, MinCost: op.MinCost, MinRemaining: op.MinRemaining, MinAmount: op.MinAmount, Enabled: op.Enabled, Params: append([]byte(nil), op.Params...), Values: cloneValues(op.Values)}
			if op.Status != nil {
				s := *op.Status
				cmd.Status = &s
			}
			r.Commands = append(r.Commands, cmd)
		}
	}
	return r, nil
}
func conditionValues(ctx Context, event Event) ConditionValues {
	values := ConditionValues{}
	if race := strings.TrimSpace(ctx.Actor.Race); race != "" {
		values["ACTOR_RACE"] = race
	}
	if race := strings.TrimSpace(ctx.Target.Race); race != "" {
		values["TARGET_RACE"] = race
	}
	if ctx.HasRank || ctx.Rank > 0 {
		values["RANK"] = float64(ctx.Rank)
	}
	if ctx.HasMastery {
		values["MASTERY"] = float64(ctx.Mastery)
	}
	if ctx.HasStacks {
		values["STACKS"] = float64(ctx.Stacks)
	}
	if ctx.Actor.MaxHP > 0 {
		values["HP_PERCENT"] = ctx.Actor.HP / ctx.Actor.MaxHP
		values["ACTOR_HP_PERCENT"] = ctx.Actor.HP / ctx.Actor.MaxHP
	}
	if ctx.Actor.MaxMana > 0 {
		values["MANA_PERCENT"] = ctx.Actor.Mana / ctx.Actor.MaxMana
	}
	if ctx.Target.MaxHP > 0 {
		values["TARGET_HP_PERCENT"] = ctx.Target.HP / ctx.Target.MaxHP
	}
	if event.HasTurn {
		values["TURN"] = float64(event.Turn)
	}
	return values
}
func domainFor(op CompiledOperation) Domain {
	if op.Type == OpDeductionFinishChance {
		return DomainRefinement
	}
	if op.Type == OpNotify || op.Type == OpModifyAptitude || op.Type == OpGrantBodyMod {
		return DomainGuUse
	}
	if op.Type == OpDisplace {
		return DomainCombat
	}
	if op.Type == OpRequire || op.Type == OpGrantResource || op.Type == OpRestoreResource || op.Type == OpModifyMaxResource || op.Type == OpAdvanceProgression || op.Type == OpAdvanceMajorRealm || op.Type == OpPetrifyAperture || op.Type == OpModifyAperture || op.Type == OpRandomizeTrait || op.Type == OpRepairAperture || op.Type == OpOpenRewardContainer {
		return DomainItemUse
	}
	if op.Type == OpPlaceWorldObject {
		return DomainWorld
	}
	if op.Type == OpGrantCapability {
		return DomainCapability
	}
	if op.Type == OpGrowthDuration || op.Type == OpPeriodicResource {
		return DomainGrowth
	}
	if op.Type == OpStatMod || op.Type == OpSetRule {
		return DomainDerived
	}
	if op.Type == OpUnique {
		if hook, ok := effectcatalog.NativeHooks[op.Code]; ok && hook.Domain != "" {
			return Domain(hook.Domain)
		}
	}
	return DomainCombat
}
