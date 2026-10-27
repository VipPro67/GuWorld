package sim

import (
	"fmt"
	"math"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// TribulationHazard represents an environmental AoE hazard (such as a lightning strike)
// during a Heavenly Tribulation in the Immortal Aperture.
type TribulationHazard struct {
	ID         string
	X          float64
	Y          float64
	Radius     float64
	Damage     float64
	StrikeTick uint64
	HazardType string
}

// AddTribulationHazard schedules an environmental hazard (e.g. lightning strike) at (x, y).
// It emits a telegraph event to clients immediately, then strikes when tick reaches StrikeTick.
func (w *World) AddTribulationHazard(x, y, radius, damage float64, delayTicks uint64, hazardType string) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	if delayTicks == 0 {
		delayTicks = 20 // 1.0 second warning default at 20Hz
	}
	hazard := TribulationHazard{
		ID:         fmt.Sprintf("trib_haz_%d", w.tick),
		X:          x,
		Y:          y,
		Radius:     radius,
		Damage:     damage,
		StrikeTick: w.tick + delayTicks,
		HazardType: hazardType,
	}
	w.tribulationHazards = append(w.tribulationHazards, hazard)
	w.emitEvent(combat.Event{
		Type:          combat.EventHitConfirmed,
		Reason:        "tribulation_telegraph",
		X:             x,
		Y:             y,
		Amount:        radius,
		DurationTicks: delayTicks,
		SkillID:       hazardType,
	})
}

// tickTribulationHazardsLocked processes pending environmental hazards in the world.
func (w *World) tickTribulationHazardsLocked() {
	if len(w.tribulationHazards) == 0 {
		return
	}
	remaining := w.tribulationHazards[:0]
	for _, hazard := range w.tribulationHazards {
		if w.tick < hazard.StrikeTick {
			remaining = append(remaining, hazard)
			continue
		}
		// Strike!
		w.emitEvent(combat.Event{
			Type:    combat.EventHitConfirmed,
			Reason:  "tribulation_strike",
			X:       hazard.X,
			Y:       hazard.Y,
			Amount:  hazard.Damage,
			SkillID: hazard.HazardType,
		})
		for _, entity := range w.entities {
			if entity.Vitals.HP <= 0 {
				continue
			}
			dist := math.Hypot(entity.Transform.Position.X-hazard.X, entity.Transform.Position.Y-hazard.Y)
			if dist <= hazard.Radius {
				w.applyDamageLocked(nil, entity, hazard.Damage, hazard.HazardType)
			}
		}
	}
	w.tribulationHazards = remaining
}
