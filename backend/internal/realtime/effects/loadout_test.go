package effects

import (
	"strconv"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func testCatalog(t *testing.T) *CompiledCatalog {
	t.Helper()
	profiles, err := LoadProfiles("../../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := loadGuCSV("../../../data/Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := CompileCatalog(profiles, entries, DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func TestBuildLoadoutUsesEquippedOrderAndReslots(t *testing.T) {
	compiled := testCatalog(t)
	equipped := []EquippedGu{
		{InstanceID: "inst-frost", GU: "GU_WHITE_FROST", Rank: 3},
		{InstanceID: "inst-arrow", GU: "GU_WATER_ARROW", Rank: 1},
		{InstanceID: "inst-missing", GU: "GU_NOT_COMPILED", Rank: 1},
		{InstanceID: "inst-shield", GU: "GU_WATER_SHIELD", Rank: 2},
		{InstanceID: "inst-heal", GU: "GU_WATER_CLEANSING", Rank: 4},
	}
	result := BuildLoadout(compiled, equipped)
	if len(result.Skills) != 4 || len(result.Skipped) != 1 {
		t.Fatalf("skills=%d skipped=%d", len(result.Skills), len(result.Skipped))
	}
	if result.Skills[0].ID != "GU_WHITE_FROST" || result.Skills[0].Slot != 1 ||
		result.Skills[1].ID != "GU_WATER_ARROW" || result.Skills[1].Slot != 2 ||
		result.Skills[3].ID != "GU_WATER_CLEANSING" || result.Skills[3].Slot != 5 {
		t.Fatalf("unexpected re-slotted loadout: %+v", result.Skills)
	}
	if result.Skipped[0].InstanceID != "inst-missing" {
		t.Fatalf("wrong skipped Gu: %+v", result.Skipped)
	}
	if result.Revision == "" {
		t.Fatal("loadout revision is empty")
	}
	again := BuildLoadout(compiled, equipped)
	if again.Revision != result.Revision {
		t.Fatal("loadout revision is not deterministic")
	}
	different := BuildLoadout(compiled, append([]EquippedGu{{InstanceID: "inst-extra", GU: "GU_STRENGTH", Rank: 1}}, equipped...))
	if different.Revision == result.Revision {
		t.Fatal("loadout revision ignored equipment changes")
	}
}

func TestBuildLoadoutCapsSixSlotsAndFallsBackWithoutCatalog(t *testing.T) {
	compiled := testCatalog(t)
	equipped := make([]EquippedGu, 0, 8)
	for i := 0; i < 8; i++ {
		equipped = append(equipped, EquippedGu{InstanceID: "inst-" + strconv.Itoa(i), GU: "GU_WATER_ARROW", Rank: 1})
	}
	result := BuildLoadout(compiled, equipped)
	if len(result.Skills) != 6 || len(result.Skipped) != 2 {
		t.Fatalf("cap failed: skills=%d skipped=%d", len(result.Skills), len(result.Skipped))
	}
	for index, skill := range result.Skills {
		if skill.Slot != index+1 {
			t.Fatalf("slot order broken: %+v", result.Skills)
		}
	}
	if result.Skills[0].Mode != combat.ModeDirection {
		t.Fatalf("compiled arrow mode=%s", result.Skills[0].Mode)
	}
	none := BuildLoadout(nil, equipped)
	if len(none.Skills) != 0 || len(none.Skipped) != len(equipped) {
		t.Fatalf("nil catalog should skip everything: %+v", none)
	}
}
