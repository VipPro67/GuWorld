package sim

import (
	"math"
	"time"
)

// Player shields are temporary. The zone bleeds a fraction of the current
// shield every decay interval, and a remainder below ShieldDecayMin is dropped
// to zero so fractional dust cannot linger forever. The same curve ages the
// durable checkpoint on reconnect, so a barrier cannot survive a logout.
const (
	ShieldDecayIntervalTicks = TickRate // one decay step per second at 20 Hz
	ShieldDecayPerSecond     = 0.10     // 10% of the current shield per second
	ShieldDecayMin           = 1.0      // below this, the shield is spent
)

// DecayedShield returns the shield left after elapsed of the decay curve. The
// zone tick and the reconnect path share it so online and offline shields lose
// value at the same rate.
func DecayedShield(shield float64, elapsed time.Duration) float64 {
	if !finite(shield) || shield <= 0 {
		return 0
	}
	if elapsed <= 0 {
		return shield
	}
	value := shield * math.Pow(1-ShieldDecayPerSecond, elapsed.Seconds())
	if value < ShieldDecayMin {
		return 0
	}
	return value
}

// decayShieldsLocked ages every living player's shield on the zone tick.
// Monsters keep their authored spawn shield; only player barriers decay.
func (w *World) decayShieldsLocked() {
	if w.tick%ShieldDecayIntervalTicks != 0 {
		return
	}
	elapsed := time.Duration(ShieldDecayIntervalTicks) * TickDuration
	for _, entity := range w.entities {
		if entity.Kind != EntityPlayer || entity.Vitals.HP <= 0 || entity.Vitals.Shield <= 0 {
			continue
		}
		next := DecayedShield(entity.Vitals.Shield, elapsed)
		if next == entity.Vitals.Shield {
			continue
		}
		entity.Vitals.Shield = next
		entity.Revision++
	}
}
