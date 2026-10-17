package gamefx

// Immortal Aperture harvest yield (WIKI §21).
//
//	count/lần = 10 × min(TOTAL_CAP, levelM × daoM × areaM × xung đột × sinh thái × lao động × dân số)
//
// The node's stored production_amount, its cell count and the produced item's
// rank take no part: every rank starts from the same 10-per-harvest floor and
// pays the same 100-seed planting cost, so a node always repays its seed in at
// most 10 harvests.
const (
	// ApertureYieldBase is both the per-harvest base and the hard floor, so a
	// debuffed (starved / overgrazed / impure) node still yields this much.
	ApertureYieldBase = 10.0
	// ApertureYieldTotalCap bounds the combined multiplier (level included).
	ApertureYieldTotalCap = 50.0
	// ApertureYieldDaoCap bounds the Dao Mark multiplier (caps at 490,000 marks).
	ApertureYieldDaoCap = 50.0
	// ApertureYieldPopCap bounds the variant-human population multiplier.
	ApertureYieldPopCap = 5.0
	// ApertureYieldLevelStep is the production gain per node level.
	ApertureYieldLevelStep = 0.5
	// ApertureSeedRepayHarvests is the planting cost in harvests a plain node
	// must cover: 100 seeds ÷ 10 per harvest.
	ApertureSeedRepayHarvests = 10
)

// ApertureYieldInputs carries the pre-computed multipliers for one harvest.
// Callers stay responsible for reading Dao marks, ecology and labor from the
// database; this function only applies the shared combination rules.
type ApertureYieldInputs struct {
	Level        int
	AreaSize     int64
	DaoMult      float64
	ConflictMult float64
	EcologyMult  float64
	EcoMult      float64
	LaborMult    float64
	PopBonus     float64
}

// ApertureYield is the combination result. Parts holds every multiplier keyed
// by name (level, dao, area, conflict, ecology, eco, labor, pop) so callers can
// render the breakdown without re-deriving it.
type ApertureYield struct {
	Count    float64
	Bonus    float64
	RawBonus float64
	Capped   bool
	Parts    map[string]float64
}

// ComputeApertureYield combines the multipliers and applies both caps. It is
// pure so the whole harvesting economy can be unit-tested without a database.
func ComputeApertureYield(in ApertureYieldInputs) ApertureYield {
	level := in.Level
	if level < 1 {
		level = 1
	}
	levelM := 1 + ApertureYieldLevelStep*float64(level-1)

	daoM := clampMin(in.DaoMult, 1)
	if daoM > ApertureYieldDaoCap {
		daoM = ApertureYieldDaoCap
	}
	conflictM := clampMin(in.ConflictMult, 1)
	ecologyM := clampMin(in.EcologyMult, 1)
	ecoM := clampMin(in.EcoMult, 1)
	laborM := clampMin(in.LaborMult, 1)
	popM := clampMin(in.PopBonus, 1)
	if popM > ApertureYieldPopCap {
		popM = ApertureYieldPopCap
	}
	areaM := 1 + float64(in.AreaSize)/10000000.0

	raw := levelM * daoM * areaM * conflictM * ecologyM * ecoM * laborM * popM
	used := raw
	capped := false
	if used > ApertureYieldTotalCap {
		used = ApertureYieldTotalCap
		capped = true
	}

	count := ApertureYieldBase * used
	if count < ApertureYieldBase {
		count = ApertureYieldBase
	}

	return ApertureYield{
		Count:    count,
		Bonus:    used,
		RawBonus: raw,
		Capped:   capped,
		Parts: map[string]float64{
			"level":    levelM,
			"dao":      daoM,
			"area":     areaM,
			"conflict": conflictM,
			"ecology":  ecologyM,
			"eco":      ecoM,
			"labor":    laborM,
			"pop":      popM,
		},
	}
}

func clampMin(v, min float64) float64 {
	if v < min {
		return min
	}
	return v
}
