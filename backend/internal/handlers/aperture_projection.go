package handlers

import (
	"math"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type immortalApertureViewProjection struct {
	aperture                 models.ImmortalAperture
	effectiveTimeRate        float64
	shouldPersist            bool
	shouldTriggerTribulation bool
}

func projectImmortalApertureView(ap models.ImmortalAperture, player models.Player, updatedAt time.Time, timeSpeedBonus float64, hasActiveTribulation, canReconcileTribulation bool, now time.Time) immortalApertureViewProjection {
	originalState := ap.State
	originalEssence := ap.ImmortalEssence
	originalYears := ap.InternalYearsPassed
	immortal := gamefx.Get(&player).Immortal

	ap.ImmortalEssence = int(player.CurrentMana)
	if canReconcileTribulation {
		if hasActiveTribulation && ap.State == "STABLE" {
			ap.State = "TRIBULATION_ACTIVE"
		} else if !hasActiveTribulation && ap.State == "TRIBULATION_ACTIVE" && ap.ApertureHP >= ap.MaxApertureHP {
			ap.State = "STABLE"
		}
	}

	effectiveTimeRate := 0.0
	if ap.TimeRate > 0 {
		timeSpeedMultiplier := 1.0
		if !immortal {
			timeSpeedMultiplier = player.TimeSpeedMultiplier
			if timeSpeedMultiplier <= 0 {
				timeSpeedMultiplier = 1.0
			}
			timeSpeedMultiplier += timeSpeedBonus
		}
		effectiveTimeRate = ap.TimeRate * timeSpeedMultiplier
		if ap.IsPlaced {
			// Hạ Khiếu follows world time, adjusted by its vertical layer.
			effectiveTimeRate = LayerTribulationTimeRateMultiplier(worldMapIDForRegion(aperturePlacedRegion(ap)))
		}
	}

	if ap.State == "STABLE" || ap.State == "DAMAGED" {
		elapsedSeconds := now.Sub(updatedAt).Seconds()
		if elapsedSeconds > 0 && effectiveTimeRate > 0 {
			ap.InternalYearsPassed += elapsedSeconds * effectiveTimeRate / 86400.0
			ap.UpdatedAt = now
		}
	}

	if ap.IsPlaced && ap.PlacedAt != nil {
		elapsedMinutes := now.Sub(*ap.PlacedAt).Minutes()
		if elapsedMinutes > 0 {
			heavenMultiplier, earthMultiplier := LayerQiMultipliers(worldMapIDForRegion(aperturePlacedRegion(ap)))
			ap.HeavenQi = math.Min(ap.MaxHeavenQi, ap.HeavenQi+elapsedMinutes*(ap.MaxHeavenQi*0.01)*heavenMultiplier)
			ap.EarthQi = math.Min(ap.MaxEarthQi, ap.EarthQi+elapsedMinutes*(ap.MaxEarthQi*0.01)*earthMultiplier)
		}
	}

	shouldPersist := ap.State != originalState || ap.ImmortalEssence != originalEssence || (now.Sub(updatedAt) >= 60*time.Second && ap.InternalYearsPassed != originalYears)
	shouldTriggerTribulation := !immortal && (ap.State == "STABLE" || ap.State == "DAMAGED") && ap.NextTribulationYear > 0 && ap.InternalYearsPassed >= ap.NextTribulationYear
	return immortalApertureViewProjection{
		aperture:                 ap,
		effectiveTimeRate:        effectiveTimeRate,
		shouldPersist:            shouldPersist,
		shouldTriggerTribulation: shouldTriggerTribulation,
	}
}

func aperturePlacedRegion(ap models.ImmortalAperture) string {
	if ap.PlacedRegion == nil {
		return ""
	}
	return *ap.PlacedRegion
}

func apertureNextTribulationAt(ap models.ImmortalAperture, effectiveTimeRate float64, immortal bool) *time.Time {
	if immortal || (ap.State != "STABLE" && ap.State != "DAMAGED") || effectiveTimeRate <= 0 || ap.NextTribulationYear <= ap.InternalYearsPassed {
		return nil
	}
	remainingYears := ap.NextTribulationYear - ap.InternalYearsPassed
	arrival := ap.UpdatedAt.Add(time.Duration(remainingYears/effectiveTimeRate*86400.0) * time.Second)
	return &arrival
}
