package effects

import "testing"

var benchmarkStatValue float64
var benchmarkCapability Capability
var benchmarkNativeOperation CompiledOperation

func BenchmarkStatValue(b *testing.B) {
	blocks := make([]Block, 8)
	for i := range blocks {
		blocks[i] = Block{
			ID:      "passive",
			Trigger: TriggerAlwaysOn,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpStatMod, Stat: "ATTACK", SubType: "ADD_FLAT", Value: 2},
				{Op: OpStatMod, Stat: "DEFENSE", SubType: "ADD_FLAT", Value: 3},
				{Op: OpStatMod, Stat: "HP", SubType: "ADD_FLAT", Value: 4},
				{Op: OpStatMod, Stat: "SPEED", SubType: "ADD_FLAT", Value: 5},
			},
		}
	}
	compiled, err := (Compiler{}).Compile(blocks)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkStatValue = compiled.StatValue(TriggerAlwaysOn, "ATTACK", "ADD_FLAT")
	}
}

func BenchmarkCapability(b *testing.B) {
	blocks := make([]Block, 8)
	for i := range blocks {
		blocks[i] = Block{
			ID:      "capability",
			Trigger: TriggerCapability,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{{
				Op: OpUnique, Code: "MARKETPLACE_ACCESS", Params: []byte(`{}`),
			}},
		}
	}
	compiled, err := (Compiler{}).Compile(blocks)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkCapability, _ = compiled.Capability("marketplace_access")
	}
}

func BenchmarkNativeMechanic(b *testing.B) {
	blocks := make([]Block, 8)
	for i := range blocks {
		blocks[i] = Block{
			ID:      "mechanic",
			Trigger: TriggerCapability,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{{
				Op: OpUnique, Code: "MARKETPLACE_ACCESS", Params: []byte(`{}`),
			}},
		}
	}
	compiled, err := (Compiler{}).Compile(blocks)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkNativeOperation, _ = compiled.NativeMechanic("marketplace_access")
	}
}
