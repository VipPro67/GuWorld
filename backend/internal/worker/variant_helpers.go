package worker

import (
	"math"
	"strings"
)

// Shared helpers for the variant-human economy workers. The per-tick
// auto-production loop that used to live here was removed: aperture node yield
// is now granted only by the harvest endpoints (handlers/aperture_collect.go),
// so there is a single source of truth for production.

// namedNPCWorker is a variant NPC that works inside an aperture.
type namedNPCWorker struct {
	role            string
	rank            int
	talent, loyalty float64
}

func clampNPCPercent(v float64) float64 { return math.Max(0, math.Min(100, v)) / 100 }

func namedNPCBonus(npcs []namedNPCWorker, role string, cap float64) float64 {
	bonus := 0.0
	for _, n := range npcs {
		if n.role == role {
			bonus += .005 * float64(n.rank) * clampNPCPercent(n.talent) * clampNPCPercent(n.loyalty)
		}
	}
	return math.Min(cap, bonus)
}

func namedNPCNodeRole(nodeType, productionKey string) string {
	v := nodeType + " " + productionKey
	if strings.Contains(strings.ToUpper(v), "FARM") || strings.Contains(strings.ToUpper(v), "BREED") || strings.Contains(strings.ToUpper(v), "FOOD") {
		return "FARMER"
	}
	return "MINER"
}

func laborIntensityModifier(intensity string) float64 {
	switch intensity {
	case "LOW":
		return 0.7
	case "HIGH":
		return 1.5
	case "EXPLOIT":
		return 2.0
	default:
		return 1.0
	}
}
