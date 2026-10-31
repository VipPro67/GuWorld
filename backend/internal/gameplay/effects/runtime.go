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
	HP, MaxHP, Mana, MaxMana float64
	Stats                    map[string]float64
}
type Context struct {
	RNG                             *rand.Rand
	Actor, Target                   Snapshot
	Rank, Mastery, Stacks           int
	DaoMarks                        map[string]int64
	Path                            []string
	CatalogRevision                 uint64
	Depth, Events, Repeats, Summons int
}
type Event struct{ Trigger Trigger }
type Domain string

const (
	DomainCombat     Domain = "combat"
	DomainDerived    Domain = "derived_player"
	DomainCapability Domain = "capability"
	DomainWorld      Domain = "world"
	DomainItemUse    Domain = "item_use"
	DomainGrowth     Domain = "growth"
	DomainRefinement Domain = "refinement"
)

type Command struct {
	Domain                                                 Domain
	Operation                                              OperationType
	Code, SubType, Stat, Path, Unit, Resource, StackPolicy string
	Target                                                 Selector
	Value                                                  float64
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
	Type                        OperationType
	Trigger                     Trigger
	SubType                     string
	Target                      *Target
	Value                       float64
	Path, Stat, Code            string
	Unit, Resource, StackPolicy string
	Params                      []byte
	Values                      map[string]any
	Status                      *statusfx.StatusOp
}

func (Compiler) Compile(input []Block) (*Compiled, error) {
	blocks := make([]compiledBlock, 0, len(input))
	for _, b := range input {
		if !effectcatalog.TriggerCatalog[string(b.Trigger)] {
			return nil, fmt.Errorf("effect %q: unknown trigger %q", b.ID, b.Trigger)
		}
		if !effectcatalog.TargetCatalog[string(b.Target.Selector)] {
			return nil, fmt.Errorf("effect %q: unknown selector %q", b.ID, b.Target.Selector)
		}
		for _, condition := range b.Conditions {
			if !effectcatalog.ConditionFields[strings.ToUpper(condition.Field)] || !effectcatalog.ConditionOperators[strings.ToUpper(condition.Operator)] {
				return nil, fmt.Errorf("effect %q: unknown condition %q/%q", b.ID, condition.Field, condition.Operator)
			}
		}
		conditions := append([]Condition(nil), b.Conditions...)
		for i := range conditions {
			conditions[i].Field = strings.ToUpper(conditions[i].Field)
			conditions[i].Operator = strings.ToUpper(conditions[i].Operator)
		}
		cb := compiledBlock{id: b.ID, trigger: b.Trigger, target: b.Target, conditions: conditions}
		for _, op := range b.Operations {
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
			if subType != "" && len(definition.SubTypes) > 0 && !definition.SubTypes[subType] {
				return nil, fmt.Errorf("effect %q: unknown subtype %q for %q", b.ID, op.SubType, op.Op)
			}
			co := CompiledOperation{Type: op.Op, SubType: subType, Value: op.Value, Path: op.Path, Stat: strings.ToUpper(op.Stat), Unit: strings.ToUpper(op.Unit), Resource: strings.ToUpper(op.Resource), StackPolicy: strings.ToUpper(op.StackPolicy), Code: strings.ToUpper(op.Code), Params: append([]byte(nil), op.Params...)}
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
				resolved, err := statusfx.ResolveStatus(op.StatusID, op.Value, op.Turns)
				if err != nil {
					return nil, fmt.Errorf("effect %q: %w", b.ID, err)
				}
				resolved.Chance = op.Chance
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
			if op.Type != OpUnique {
				continue
			}
			out = append(out, Capability{op.Code, append([]byte(nil), op.Params...)})
		}
	}
	return out
}

// Capability returns a defensive copy of the first compiled capability with
// the requested code. Capability consumers must query this projection instead
// of interpreting the persisted authoring JSON.
func (c *Compiled) Capability(code string) (Capability, bool) {
	if c == nil {
		return Capability{}, false
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, capability := range c.Capabilities() {
		if strings.ToUpper(capability.Code) == code {
			return capability, true
		}
	}
	return Capability{}, false
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

// StatValue sums a typed stat projection for one trigger and subtype.
func (c *Compiled) StatValue(trigger Trigger, stat, subType string) float64 {
	stat, subType = strings.ToUpper(stat), strings.ToUpper(subType)
	var total float64
	for _, op := range c.Operations(trigger) {
		if op.Type == OpStatMod && op.Stat == stat && (subType == "" || op.SubType == subType) {
			total += op.Value
		}
	}
	return total
}
func cloneOps(in []CompiledOperation) []CompiledOperation {
	out := append([]CompiledOperation(nil), in...)
	for i := range out {
		out[i].Params = append([]byte(nil), out[i].Params...)
		out[i].Values = cloneValues(out[i].Values)
		if out[i].Target != nil {
			t := *out[i].Target
			out[i].Target = &t
		}
		if out[i].Status != nil {
			s := *out[i].Status
			out[i].Status = &s
		}
	}
	return out
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
				cloned := cloneOps([]CompiledOperation{op})[0]
				cloned.Trigger = b.trigger
				out = append(out, cloned)
			}
		}
	}
	return out
}

func (c *Compiled) Evaluate(ctx Context, event Event) (Result, error) {
	limits := DefaultLimits
	if ctx.Depth >= limits.MaxTriggerDepth || ctx.Events >= limits.MaxEventsPerAction || ctx.Repeats > limits.MaxRepeats || ctx.Summons > limits.MaxSummons {
		return Result{Truncated: true}, nil
	}
	var r Result
	repeats, summons := ctx.Repeats, ctx.Summons
	for _, b := range c.blocks {
		if b.trigger != event.Trigger || !conditionsMatch(b.conditions, ctx) {
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
			cmd := Command{Domain: domain, Operation: op.Type, Code: op.Code, SubType: op.SubType, Stat: op.Stat, Path: op.Path, Unit: op.Unit, Resource: op.Resource, StackPolicy: op.StackPolicy, Target: target, Value: op.Value, Params: append([]byte(nil), op.Params...), Values: cloneValues(op.Values)}
			if op.Status != nil {
				s := *op.Status
				cmd.Status = &s
			}
			r.Commands = append(r.Commands, cmd)
		}
	}
	return r, nil
}
func conditionsMatch(cs []Condition, ctx Context) bool {
	for _, c := range cs {
		var got float64
		switch strings.ToUpper(c.Field) {
		case "ACTOR_HP_PERCENT", "HP_PERCENT":
			if ctx.Actor.MaxHP > 0 {
				got = ctx.Actor.HP / ctx.Actor.MaxHP
			}
		case "MANA_PERCENT":
			if ctx.Actor.MaxMana > 0 {
				got = ctx.Actor.Mana / ctx.Actor.MaxMana
			}
		case "TARGET_HP_PERCENT":
			if ctx.Target.MaxHP > 0 {
				got = ctx.Target.HP / ctx.Target.MaxHP
			}
		case "RANK":
			got = float64(ctx.Rank)
		case "MASTERY":
			got = float64(ctx.Mastery)
		case "STACKS":
			got = float64(ctx.Stacks)
		default:
			return false
		}
		switch c.Operator {
		case "LT":
			if !(got < c.Value) {
				return false
			}
		case "LTE":
			if !(got <= c.Value) {
				return false
			}
		case "GT":
			if !(got > c.Value) {
				return false
			}
		case "GTE":
			if !(got >= c.Value) {
				return false
			}
		case "EQ":
			if got != c.Value {
				return false
			}
		case "NE":
			if got == c.Value {
				return false
			}
		default:
			return false
		}
	}
	return true
}
func domainFor(op CompiledOperation) Domain {
	if op.Type == OpStatMod {
		return DomainDerived
	}
	if op.Type == OpUnique {
		if hook, ok := effectcatalog.NativeHooks[op.Code]; ok && hook.Domain != "" {
			return Domain(hook.Domain)
		}
	}
	return DomainCombat
}
