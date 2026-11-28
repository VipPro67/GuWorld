package effectcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// PowerRankScaleTable mirrors models.RankScaleTable so effectcatalog can
// compute Rank 1-9 power budgets without introducing an import cycle.
var PowerRankScaleTable = [10]float64{
	0: 1.0,
	1: 1.0,
	2: 2.0,
	3: 5.0,
	4: 20.0,
	5: 100.0,
	6: 10000.0,
	7: 1000000.0,
	8: 100000000.0,
	9: 10000000000.0,
}

// RankScale returns the shared rank scale factor for ranks 1-9.
func RankScale(rank int) float64 {
	if rank <= 1 {
		return PowerRankScaleTable[1]
	}
	if rank >= 9 {
		return PowerRankScaleTable[9]
	}
	return PowerRankScaleTable[rank]
}

// PowerBudget returns the maximum effect power budget for a single Gu of rank 1-9:
// PowerBudget(rank) = 100 * RankScale(rank).
func PowerBudget(rank int) float64 {
	return 100.0 * RankScale(rank)
}

// MaxOpsForRank returns the maximum number of operations allowed on a single Gu of rank 1-9.
func MaxOpsForRank(rank int) int {
	switch {
	case rank <= 2:
		return 2
	case rank <= 4:
		return 3
	case rank == 5:
		return 4
	case rank <= 7:
		return 5
	default:
		return 8
	}
}

type CostGroup string

const (
	CostGroupFlatCombat   CostGroup = "FLAT_COMBAT"
	CostGroupPercentScale CostGroup = "PERCENT_SCALE"
	CostGroupStatusCC     CostGroup = "STATUS_CC"
	CostGroupRuleBreaking CostGroup = "RULE_BREAKING"
	CostGroupNonCombat    CostGroup = "NON_COMBAT"
	CostGroupMeta         CostGroup = "META"
)

// PowerCostSpec defines the base cost, scaling behavior, and minimum rank gate
// for an OperationType in OperationCatalog.
type PowerCostSpec struct {
	Group          CostGroup `json:"cost_group"`
	BaseCost       float64   `json:"base_cost"`
	ValueWeight    float64   `json:"value_weight"`
	FlatNoScale    bool      `json:"flat_no_scale"`
	MinRankGate    int       `json:"min_rank_gate"`
	AoEMinRankGate int       `json:"aoe_min_rank_gate,omitempty"`
}

// PowerCostCatalog maps every OperationType to its PowerCostSpec.
var PowerCostCatalog = map[string]PowerCostSpec{
	// Group 1: Flat combat (value already scales by RankScale in Gu data)
	"DAMAGE":    {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 1.0, FlatNoScale: true, MinRankGate: 1},
	"SHIELD":    {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 0.9, FlatNoScale: true, MinRankGate: 1},
	"BARRIER":   {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 0.9, FlatNoScale: true, MinRankGate: 1},
	"HEAL":      {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 1.2, FlatNoScale: true, MinRankGate: 1},
	"DIRECT_HP": {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 1.2, FlatNoScale: true, MinRankGate: 1},
	"STAT_MOD":  {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: 1.5, FlatNoScale: true, MinRankGate: 1},
	"HP_COST":   {Group: CostGroupFlatCombat, BaseCost: 0, ValueWeight: -0.4, FlatNoScale: true, MinRankGate: 1},

	// Group 2: Percentage / Multiplier combat (scaled by RankScale)
	"BONUS_STRIKE":           {Group: CostGroupPercentScale, BaseCost: 45, ValueWeight: 0.4, MinRankGate: 1},
	"CHAIN_DAMAGE":           {Group: CostGroupPercentScale, BaseCost: 35, ValueWeight: 0.35, MinRankGate: 1},
	"DAMAGE_MODIFIER":        {Group: CostGroupPercentScale, BaseCost: 10, ValueWeight: 1.5, MinRankGate: 1},
	"DEFENSE_PENETRATION":    {Group: CostGroupPercentScale, BaseCost: 10, ValueWeight: 1.5, MinRankGate: 6},
	"TARGET_DAMAGE_MODIFIER": {Group: CostGroupPercentScale, BaseCost: 10, ValueWeight: 1.2, MinRankGate: 1},
	"MITIGATE_DAMAGE":        {Group: CostGroupPercentScale, BaseCost: 15, ValueWeight: 1.4, MinRankGate: 6},
	"DETONATE_STATUS":        {Group: CostGroupPercentScale, BaseCost: 25, ValueWeight: 1.0, MinRankGate: 3},
	"TRANSFER_RESOURCE":      {Group: CostGroupPercentScale, BaseCost: 20, ValueWeight: 1.0, MinRankGate: 5},
	"RESOURCE_CONVERSION":    {Group: CostGroupPercentScale, BaseCost: 20, ValueWeight: 1.0, MinRankGate: 5},
	"TENACITY_MODIFIER":      {Group: CostGroupPercentScale, BaseCost: 15, ValueWeight: 1.0, MinRankGate: 5},
	"RANK_SCALED_DEBUFF":     {Group: CostGroupPercentScale, BaseCost: 45, ValueWeight: 0.5, MinRankGate: 8},

	// Group 3: Status, CC & Stacking (scaled by RankScale)
	"STATUS":                {Group: CostGroupStatusCC, BaseCost: 20, ValueWeight: 0.3, MinRankGate: 1},
	"APPLY_STACKING_STATUS": {Group: CostGroupStatusCC, BaseCost: 25, ValueWeight: 0.3, MinRankGate: 1},
	"COUNTER_ADD":           {Group: CostGroupStatusCC, BaseCost: 15, ValueWeight: 0.2, MinRankGate: 1},
	"CLEANSE":               {Group: CostGroupStatusCC, BaseCost: 15, ValueWeight: 0, MinRankGate: 1, AoEMinRankGate: 3},
	"DISPEL":                {Group: CostGroupStatusCC, BaseCost: 18, ValueWeight: 0, MinRankGate: 1, AoEMinRankGate: 3},
	"REVOKE_STATUS":         {Group: CostGroupStatusCC, BaseCost: 20, ValueWeight: 0, MinRankGate: 1},
	"THRESHOLD_CHECK":       {Group: CostGroupStatusCC, BaseCost: 20, ValueWeight: 0.2, MinRankGate: 1},
	"THRESHOLD_RECOVERY":    {Group: CostGroupStatusCC, BaseCost: 35, ValueWeight: 0.5, MinRankGate: 1},

	// Group 4: Rule-Breaking / Special (scaled by RankScale + MinRankGate)
	"REVIVE":             {Group: CostGroupRuleBreaking, BaseCost: 80, ValueWeight: 0, MinRankGate: 6},
	"END_BATTLE":         {Group: CostGroupRuleBreaking, BaseCost: 90, ValueWeight: 0, MinRankGate: 9},
	"EFFECT_IMMUNITY":    {Group: CostGroupRuleBreaking, BaseCost: 70, ValueWeight: 0, MinRankGate: 9},
	"STATUS_IMMUNITY":    {Group: CostGroupRuleBreaking, BaseCost: 50, ValueWeight: 0, MinRankGate: 6},
	"GU_SEAL":            {Group: CostGroupRuleBreaking, BaseCost: 45, ValueWeight: 0, MinRankGate: 6},
	"TARGET_REDIRECTION": {Group: CostGroupRuleBreaking, BaseCost: 45, ValueWeight: 0, MinRankGate: 6},
	"SUMMON":             {Group: CostGroupRuleBreaking, BaseCost: 40, ValueWeight: 0, MinRankGate: 5},
	"SLOT_INTERACTION":   {Group: CostGroupRuleBreaking, BaseCost: 45, ValueWeight: 0, MinRankGate: 8},
	"HIT_RULE":           {Group: CostGroupRuleBreaking, BaseCost: 30, ValueWeight: 0, MinRankGate: 1},
	"TURN_ORDER_RULE":    {Group: CostGroupRuleBreaking, BaseCost: 35, ValueWeight: 0, MinRankGate: 8},
	"DISPLACE":           {Group: CostGroupRuleBreaking, BaseCost: 25, ValueWeight: 0.5, MinRankGate: 3},
	"SET_RULE":           {Group: CostGroupRuleBreaking, BaseCost: 35, ValueWeight: 0, MinRankGate: 1},
	"OUTCOME_ROULETTE":   {Group: CostGroupRuleBreaking, BaseCost: 35, ValueWeight: 0, MinRankGate: 1},

	// Group 5: Non-combat / Economy / World / Growth (scaled by RankScale)
	"GRANT_RESOURCE":          {Group: CostGroupNonCombat, BaseCost: 20, ValueWeight: 0.05, MinRankGate: 1},
	"RESTORE_RESOURCE":        {Group: CostGroupNonCombat, BaseCost: 15, ValueWeight: 0.05, MinRankGate: 1},
	"MODIFY_MAX_RESOURCE":     {Group: CostGroupNonCombat, BaseCost: 25, ValueWeight: 0.1, MinRankGate: 1},
	"MODIFY_APTITUDE":         {Group: CostGroupNonCombat, BaseCost: 40, ValueWeight: 0.2, MinRankGate: 1},
	"GRANT_BODY_MOD":          {Group: CostGroupNonCombat, BaseCost: 85, ValueWeight: 0, MinRankGate: 6},
	"ADVANCE_PROGRESSION":     {Group: CostGroupNonCombat, BaseCost: 40, ValueWeight: 0, MinRankGate: 1},
	"ADVANCE_MAJOR_REALM":     {Group: CostGroupNonCombat, BaseCost: 85, ValueWeight: 0, MinRankGate: 6},
	"PETRIFY_APERTURE":        {Group: CostGroupNonCombat, BaseCost: 25, ValueWeight: 0, MinRankGate: 1},
	"MODIFY_APERTURE":         {Group: CostGroupNonCombat, BaseCost: 45, ValueWeight: 0.1, MinRankGate: 6},
	"RANDOMIZE_TRAIT":         {Group: CostGroupNonCombat, BaseCost: 35, ValueWeight: 0, MinRankGate: 1},
	"REPAIR_APERTURE":         {Group: CostGroupNonCombat, BaseCost: 25, ValueWeight: 0.1, MinRankGate: 1},
	"OPEN_REWARD_CONTAINER":   {Group: CostGroupNonCombat, BaseCost: 20, ValueWeight: 0, MinRankGate: 1},
	"PLACE_WORLD_OBJECT":      {Group: CostGroupNonCombat, BaseCost: 35, ValueWeight: 0, MinRankGate: 1},
	"GRANT_CAPABILITY":        {Group: CostGroupNonCombat, BaseCost: 20, ValueWeight: 0, MinRankGate: 1},
	"GROWTH_DURATION":         {Group: CostGroupNonCombat, BaseCost: 15, ValueWeight: 0.1, MinRankGate: 1},
	"PERIODIC_RESOURCE":       {Group: CostGroupNonCombat, BaseCost: 25, ValueWeight: 0.05, MinRankGate: 1},
	"DEDUCTION_FINISH_CHANCE": {Group: CostGroupNonCombat, BaseCost: 25, ValueWeight: 0, MinRankGate: 1},

	// Group 6: Meta
	"REQUIRE": {Group: CostGroupMeta, BaseCost: 0, ValueWeight: 0, MinRankGate: 1},
	"NOTIFY":  {Group: CostGroupMeta, BaseCost: 0, ValueWeight: 0, MinRankGate: 1},
	"UNIQUE":  {Group: CostGroupMeta, BaseCost: 30, ValueWeight: 0, MinRankGate: 1},
}

// PowerViolation describes a power budget, op cap, or min rank gate violation.
type PowerViolation struct {
	Code    string `json:"code"`
	BlockID string `json:"block_id,omitempty"`
	Op      string `json:"op,omitempty"`
	Message string `json:"message"`
}

// PowerBudgetReport is the machine-readable output of EvaluatePowerBudget.
type PowerBudgetReport struct {
	Rank            int              `json:"rank"`
	PowerBudget     float64          `json:"power_budget"`
	PowerCost       float64          `json:"power_cost"`
	UsagePct        float64          `json:"usage_pct"`
	OpCount         int              `json:"op_count"`
	OpCap           int              `json:"op_cap"`
	MinRankRequired int              `json:"min_rank_required"`
	Violations      []PowerViolation `json:"violations"`
}

// PowerTargetInput is the target selector & bypass flag used for power evaluation.
type PowerTargetInput struct {
	Selector        string `json:"selector"`
	BypassFrontline bool   `json:"bypass_frontline,omitempty"`
}

// PowerOpInput is the operation fields used for power evaluation.
type PowerOpInput struct {
	Op      string            `json:"op"`
	SubType string            `json:"sub_type,omitempty"`
	Unit    string            `json:"unit,omitempty"`
	Status  string            `json:"status_id,omitempty"`
	Target  *PowerTargetInput `json:"target,omitempty"`
	Value   float64           `json:"value,omitempty"`
	Turns   int               `json:"turns,omitempty"`
	Chance  *float64          `json:"chance,omitempty"`
}

// PowerBlockInput is a Block v2 block projected for power evaluation.
type PowerBlockInput struct {
	ID         string           `json:"id"`
	Trigger    string           `json:"trigger"`
	Target     PowerTargetInput `json:"target"`
	Conditions []any            `json:"conditions,omitempty"`
	Operations []PowerOpInput   `json:"operations"`
}

// EvaluatePowerBudgetJSON parses a raw Block v2 JSON array/envelope and evaluates its power budget.
func EvaluatePowerBudgetJSON(rank int, raw []byte) (PowerBudgetReport, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return EvaluatePowerBudget(rank, nil), nil
	}
	var blocks []PowerBlockInput
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &blocks); err != nil {
			return PowerBudgetReport{}, err
		}
	} else if trimmed[0] == '{' {
		var env struct {
			Blocks []PowerBlockInput `json:"blocks"`
		}
		if err := json.Unmarshal(trimmed, &env); err != nil {
			return PowerBudgetReport{}, err
		}
		blocks = env.Blocks
	}
	return EvaluatePowerBudget(rank, blocks), nil
}

// EvaluatePowerBudget evaluates blocks against the default PowerBudget(rank) and MaxOpsForRank(rank).
func EvaluatePowerBudget(rank int, blocks []PowerBlockInput) PowerBudgetReport {
	return EvaluatePowerBudgetWithLimits(rank, PowerBudget(rank), MaxOpsForRank(rank), blocks)
}

// EvaluatePowerBudgetWithLimits evaluates blocks against custom budget/opCap limits (e.g. Killer Move / Gu House).
func EvaluatePowerBudgetWithLimits(rank int, budget float64, opCap int, blocks []PowerBlockInput) PowerBudgetReport {
	scale := RankScale(rank)
	report := PowerBudgetReport{
		Rank:            rank,
		PowerBudget:     budget,
		OpCap:           opCap,
		MinRankRequired: 1,
		Violations:      []PowerViolation{},
	}

	totalCost := 0.0
	for _, block := range blocks {
		blockPositiveCost := 0.0
		blockDiscount := 0.0
		for _, op := range block.Operations {
			report.OpCount++
			target := block.Target
			if op.Target != nil {
				target = *op.Target
			}
			opName := strings.ToUpper(strings.TrimSpace(op.Op))
			spec, ok := PowerCostCatalog[opName]
			if !ok {
				spec = PowerCostSpec{Group: CostGroupMeta, BaseCost: 30, MinRankGate: 1}
			}

			minRank := effectiveMinRankGate(opName, op, target, spec)
			if minRank > report.MinRankRequired {
				report.MinRankRequired = minRank
			}
			if rank > 0 && minRank > rank {
				report.Violations = append(report.Violations, PowerViolation{
					Code:    "min_rank_gate_violated",
					BlockID: block.ID,
					Op:      opName,
					Message: fmt.Sprintf("%s yêu cầu tối thiểu Rank %d (hiện tại Rank %d)", opName, minRank, rank),
				})
			}

			opCost := computeOpCost(opName, op, block.Trigger, target, len(block.Conditions), scale, spec)
			if opCost < 0 {
				blockDiscount += -opCost
			} else {
				blockPositiveCost += opCost
			}
		}
		if blockDiscount > 0 && blockPositiveCost > 0 {
			maxDiscount := blockPositiveCost * 0.30
			if blockDiscount > maxDiscount {
				blockDiscount = maxDiscount
			}
			totalCost += blockPositiveCost - blockDiscount
		} else {
			totalCost += blockPositiveCost
		}
	}

	report.PowerCost = math.Round(totalCost*100) / 100
	if budget > 0 {
		report.UsagePct = math.Round((report.PowerCost/budget)*10000) / 100
		if report.PowerCost > budget {
			report.Violations = append(report.Violations, PowerViolation{
				Code:    "power_budget_exceeded",
				Message: fmt.Sprintf("Chi phí Effect (%.0f) vượt Ngân sách Sức mạnh (%.0f)", report.PowerCost, budget),
			})
		}
	}
	if opCap > 0 && report.OpCount > opCap {
		report.Violations = append(report.Violations, PowerViolation{
			Code:    "op_cap_exceeded",
			Message: fmt.Sprintf("Số lượng Operation (%d) vượt giới hạn cho phép (%d)", report.OpCount, opCap),
		})
	}
	return report
}

func effectiveMinRankGate(opName string, op PowerOpInput, target PowerTargetInput, spec PowerCostSpec) int {
	minRank := spec.MinRankGate
	if minRank <= 0 {
		minRank = 1
	}
	sel := strings.ToUpper(strings.TrimSpace(target.Selector))
	if spec.AoEMinRankGate > minRank && (sel == "ALL" || sel == "ALL_ENEMIES" || sel == "ALL_ALLIES") {
		minRank = spec.AoEMinRankGate
	}
	if opName == "SET_RULE" && strings.EqualFold(op.SubType, "IMMORTAL") && minRank < 9 {
		minRank = 9
	}
	return minRank
}

func computeOpCost(opName string, op PowerOpInput, trigger string, target PowerTargetInput, conditionCount int, rankScale float64, spec PowerCostSpec) float64 {
	if opName == "STATUS" && op.Chance != nil && *op.Chance <= 0 {
		return 0
	}

	baseCost := spec.BaseCost
	val := math.Abs(op.Value)
	isFlat := spec.FlatNoScale && !isPercentOrMultiplierOp(opName, op)

	if opName == "HP_COST" {
		// HP_COST is a self-sacrifice discount scaled to RankScale.
		return -math.Min(30.0, val*0.4) * rankScale * 0.5
	}

	if isFlat {
		baseCost += val * spec.ValueWeight
	} else {
		if opName == "STATUS" {
			baseCost = statusBaseCost(op.Status)
		}
		raw := baseCost + val*spec.ValueWeight
		baseCost = raw * rankScale
	}

	chanceMult := 1.0
	if op.Chance != nil && *op.Chance > 0 {
		chanceMult = math.Max(0.25, math.Min(1.0, *op.Chance))
	}

	return baseCost *
		targetMultiplier(target) *
		triggerMultiplier(trigger) *
		durationMultiplier(trigger, op.Turns) *
		chanceMult *
		conditionDiscount(conditionCount)
}

func isPercentOrMultiplierOp(opName string, op PowerOpInput) bool {
	sub := strings.ToUpper(strings.TrimSpace(op.SubType))
	unit := strings.ToUpper(strings.TrimSpace(op.Unit))
	if unit == "PERCENT" || strings.Contains(sub, "PERCENT") || sub == "MULTIPLIER" || sub == "MULTIPLY_PERCENT" || sub == "LIFESTEAL" || sub == "SHIELD_STRIP" || sub == "RESOURCE" {
		return true
	}
	if opName == "STAT_MOD" && sub != "ADD_FLAT" {
		return true
	}
	return false
}

func statusBaseCost(statusID string) float64 {
	switch strings.ToUpper(strings.TrimSpace(statusID)) {
	case "STUN", "FREEZE", "PETRIFY", "SLEEP":
		return 35.0
	case "SILENCE", "SEAL", "CONFUSE", "BLIND", "DISARM":
		return 25.0
	default:
		return 15.0
	}
}

func targetMultiplier(target PowerTargetInput) float64 {
	mult := 1.0
	switch strings.ToUpper(strings.TrimSpace(target.Selector)) {
	case "ROW", "COL":
		mult = 1.5
	case "ALL_ENEMIES", "ALL_ALLIES":
		mult = 2.2
	case "ALL":
		mult = 2.6
	}
	if target.BypassFrontline {
		mult += 0.25
	}
	return mult
}

func triggerMultiplier(trigger string) float64 {
	switch strings.ToUpper(strings.TrimSpace(trigger)) {
	case "ALWAYS_ON", "COMBAT_PASSIVE", "PASSIVE", "CAPABILITY":
		return 1.5
	case "ON_HIT", "BEFORE_HIT", "BEFORE_ENEMY_ATTACK", "ON_TURN_START", "ON_TICK":
		return 1.15
	case "ON_BATTLE_START":
		return 1.1
	case "ON_DEATH":
		return 0.7
	default:
		return 1.0
	}
}

func durationMultiplier(trigger string, turns int) float64 {
	trig := strings.ToUpper(strings.TrimSpace(trigger))
	if trig == "ALWAYS_ON" || trig == "COMBAT_PASSIVE" || trig == "PASSIVE" || trig == "CAPABILITY" {
		return 1.0
	}
	if turns <= 1 {
		return 1.0
	}
	return 1.0 + 0.35*float64(turns-1)
}

func conditionDiscount(conditionCount int) float64 {
	if conditionCount <= 0 {
		return 1.0
	}
	if conditionCount == 1 {
		return 0.85
	}
	if conditionCount == 2 {
		return 0.75
	}
	return 0.65
}
