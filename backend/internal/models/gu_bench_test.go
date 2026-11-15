package models

import "testing"

func BenchmarkGrowthProfile(b *testing.B) {
	var effects GuEffects
	if err := effects.Scan(`[{"id":"GROWTH","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":100},{"op":"STAT_MOD","stat":"DEFENSE","sub_type":"GROWTH_CAP","value":50},{"op":"STAT_MOD","stat":"HP","sub_type":"GROWTH_CAP","value":500},{"op":"STAT_MOD","stat":"SPEED","sub_type":"GROWTH_CAP","value":20},{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":100}}]}]`); err != nil {
		b.Fatal(err)
	}
	spec := GuDictionaryItem{ID: "growth_benchmark", Rank: 6, Effects: effects}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkGrowthProfile, benchmarkGrowthProfileOK = GrowthProfile(spec)
	}
}

var (
	benchmarkGrowthProfile   GuGrowthProfile
	benchmarkGrowthProfileOK bool
)
