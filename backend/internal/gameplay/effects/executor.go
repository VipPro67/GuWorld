package effects

import "context"

// Executor is the common domain seam. Implementations do not own transactions:
// handlers pass a transaction-backed Applicator for commands with DB effects.
type Executor interface {
	Domain() Domain
	Execute(context.Context, Command, Applicator) error
}
type Applicator interface {
	ApplyEffectCommand(context.Context, Command) error
}
type DomainExecutor struct{ Kind Domain }

func (e DomainExecutor) Domain() Domain { return e.Kind }
func (e DomainExecutor) Execute(ctx context.Context, command Command, tx Applicator) error {
	return tx.ApplyEffectCommand(ctx, command)
}

func Executors() []Executor {
	return []Executor{
		DomainExecutor{DomainCombat}, DomainExecutor{DomainDerived}, DomainExecutor{DomainCapability}, DomainExecutor{DomainWorld}, DomainExecutor{DomainItemUse}, DomainExecutor{DomainGrowth}, DomainExecutor{DomainRefinement},
	}
}
