package effects

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

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
	Rank, Mastery                   int
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
	Domain                    Domain
	Operation                 OperationType
	Code, SubType, Stat, Path string
	Target                    Selector
	Value                     float64
	Params                    []byte
	Values                    map[string]any
	Status                    *statusfx.StatusOp
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
	Type             OperationType
	Trigger          Trigger
	SubType          string
	Target           *Target
	Value            float64
	Path, Stat, Code string
	Params           []byte
	Values           map[string]any
	Status           *statusfx.StatusOp
}

func (Compiler) Compile(input []Block) (*Compiled, error) {
	blocks := make([]compiledBlock, 0, len(input))
	for _, b := range input {
		if !Triggers[b.Trigger] {
			return nil, fmt.Errorf("effect %q: unknown trigger %q", b.ID, b.Trigger)
		}
		if !Selectors[b.Target.Selector] {
			return nil, fmt.Errorf("effect %q: unknown selector %q", b.ID, b.Target.Selector)
		}
		for _, condition := range b.Conditions {
			if !Conditions[strings.ToUpper(condition.Field)] || !ConditionOperators[strings.ToUpper(condition.Operator)] {
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
			if !Operations[op.Op] {
				return nil, fmt.Errorf("effect %q: unknown operation %q", b.ID, op.Op)
			}
			co := CompiledOperation{Type: op.Op, SubType: op.SubType, Value: op.Value, Path: op.Path, Stat: strings.ToUpper(op.Stat), Code: op.Code, Params: append([]byte(nil), op.Params...)}
			if len(op.Params) > 0 {
				if err := json.Unmarshal(op.Params, &co.Values); err != nil {
					return nil, fmt.Errorf("effect %q: invalid native params: %w", b.ID, err)
				}
			}
			if op.Target != nil {
				t := *op.Target
				if !Selectors[t.Selector] {
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
			if op.Stat != "" && !Stats[co.Stat] {
				return nil, fmt.Errorf("effect %q: unknown stat/resource %q", b.ID, op.Stat)
			}
			if op.Op == OpUnique && !NativeMechanics[op.Code] {
				return nil, fmt.Errorf("effect %q: unknown native mechanic %q", b.ID, op.Code)
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
			cmd := Command{Domain: domain, Operation: op.Type, Code: op.Code, SubType: op.SubType, Stat: op.Stat, Path: op.Path, Target: target, Value: op.Value, Params: append([]byte(nil), op.Params...), Values: cloneValues(op.Values)}
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
		case "ACTOR_HP_PERCENT":
			if ctx.Actor.MaxHP > 0 {
				got = ctx.Actor.HP / ctx.Actor.MaxHP
			}
		case "TARGET_HP_PERCENT":
			if ctx.Target.MaxHP > 0 {
				got = ctx.Target.HP / ctx.Target.MaxHP
			}
		case "RANK":
			got = float64(ctx.Rank)
		case "MASTERY":
			got = float64(ctx.Mastery)
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
		if NativeMechanicDomains[op.Code] != "" {
			return NativeMechanicDomains[op.Code]
		}
	}
	return DomainCombat
}
