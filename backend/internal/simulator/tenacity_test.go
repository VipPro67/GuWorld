package simulator

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// Tenacity dispels one debuff on a winning roll, hard lock first.
func TestTenacityDispelsHardLockFirst(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	c := &CombatantAuto{
		ID: "u1", Name: "T", HP: 100, MaxHP: 100,
		Tenacity: 1.0, // capped 0.75 → roll < 0.75 wins unless unlucky; retry seeds
		ActiveEffects: []Effect{
			{ID: statusfx.StatusSlow, Type: "debuff", Value: 20, Duration: 3},
			{ID: statusfx.StatusStun, Type: "debuff", Value: 1, Duration: 2},
		},
	}
	var logs []CombatLog
	for i := int64(1); i < 20 && len(logs) == 0; i++ {
		c.ActiveEffects = []Effect{
			{ID: statusfx.StatusSlow, Type: "debuff", Value: 20, Duration: 3},
			{ID: statusfx.StatusStun, Type: "debuff", Value: 1, Duration: 2},
		}
		logs = TryTenacityDispel(c, 1, rand.New(rand.NewSource(i)))
		_ = rng
	}
	if len(logs) == 0 {
		t.Fatal("expected tenacity to win a roll within 20 seeds")
	}
	for _, eff := range c.ActiveEffects {
		if eff.ID == statusfx.StatusStun {
			t.Fatalf("expected STUN dispelled first, remaining: %+v", c.ActiveEffects)
		}
	}
}

// Combined resist is multiplicative with 0.75 cap.
func TestCombinedResistCap(t *testing.T) {
	if got := gamefx.CombinedResist(0.5, 0.5); got != 0.75 {
		t.Fatalf("cap = %v, want 0.75", got)
	}
	if got := gamefx.CombinedResist(0, 0); got != 0 {
		t.Fatalf("zero = %v, want 0", got)
	}
	if got := gamefx.CappedTenacity(2); got != 0.75 {
		t.Fatalf("tenacity cap = %v, want 0.75", got)
	}
}
