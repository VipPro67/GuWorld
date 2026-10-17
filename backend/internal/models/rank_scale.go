package models

// RankScaleTable is the single source of truth for the 9-rank power/cost scale
// shared by Gu values, monster stats and player base stats.
//
// Mortals: Rank 1=1, Rank 2=2, Rank 3=5, Rank 4=20, Rank 5=100.
// Immortals: Rank 6=10^4, Rank 7=10^6, Rank 8=10^8, Rank 9=10^10 (x100 per rank).
//
// Gu values, monster base stats and player base stats are all meant to be
// expressed as <ratio> * RankScale(rank); see docs/GAME_DATA.md. Keep the table
// here (a leaf package) so both models and gamefx can share it without an
// import cycle.
var RankScaleTable = [10]float64{
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

// RankScale returns the shared rank scale factor for ranks 1-9. Ranks outside
// the supported range are clamped to rank 1 / rank 9.
func RankScale(rank int) float64 {
	if rank <= 1 {
		return RankScaleTable[1]
	}
	if rank >= 9 {
		return RankScaleTable[9]
	}
	return RankScaleTable[rank]
}
