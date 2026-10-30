package persistence

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestStateValidation(t *testing.T) {
	state := State{PlayerID: uuid.New(), ZoneID: "forest", InstanceID: "i", X: 1, Y: 2, HP: 10, SafeX: 1, SafeY: 2}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	state.HP = -1
	if err := state.Validate(); err == nil {
		t.Fatal("negative HP accepted")
	}
	state.HP = 10
	state.X = math.NaN()
	if err := state.Validate(); err == nil {
		t.Fatal("NaN coordinate accepted")
	}
	state.X = 1
	state.PlayerID = uuid.Nil
	if err := state.Validate(); err == nil {
		t.Fatal("nil player accepted")
	}
	state.PlayerID = uuid.New()
	state.HasLocationRevision = true
	state.LocationRevision = -1
	if err := state.Validate(); err == nil {
		t.Fatal("negative location revision accepted")
	}
}

func TestApplyPathMasteryUsesUsesRankTable(t *testing.T) {
	masteries := models.PathMasteriesMap{}
	if !applyPathMasteryUses(masteries, "Fire Path", 1, 3) {
		t.Fatal("rank 1 cast did not grant Chân Ý")
	}
	if got := masteries["fire"]; got != 3 {
		t.Fatalf("rank 1 x3 = %d, want 3", got)
	}
	if !applyPathMasteryUses(masteries, "fire", 9, 1) {
		t.Fatal("rank 9 cast did not grant Chân Ý")
	}
	if got, want := masteries["fire"], 3+gamefx.PathMasteryGuUseBaseGain(9); got != want {
		t.Fatalf("rank 9 gain = %d, want %d", got, want)
	}
}

func TestApplyPathMasteryUsesStopsAtRankCap(t *testing.T) {
	masteries := models.PathMasteriesMap{"fire": gamefx.PathMasteryGrandmasterPoints}
	if applyPathMasteryUses(masteries, "fire", 1, 1) {
		t.Fatal("rank 1 Cast should not grant points past its cap")
	}
	if got := masteries["fire"]; got != gamefx.PathMasteryGrandmasterPoints {
		t.Fatalf("capped path changed to %d", got)
	}
}

func TestApplyPathMasteryUsesDefaultsPathAndRank(t *testing.T) {
	masteries := models.PathMasteriesMap{}
	if !applyPathMasteryUses(masteries, "", 0, 1) {
		t.Fatal("default path/rank did not grant Chân Ý")
	}
	if got := masteries["strength"]; got != 1 {
		t.Fatalf("default strength gain = %d, want 1", got)
	}
}

func TestNormalizeSkillUseKeyDefaultsPathAndRank(t *testing.T) {
	key := normalizeSkillUseKey("Fire Path", 0)
	if key.Path != "fire" || key.Rank != 1 {
		t.Fatalf("normalize = %+v, want fire/1", key)
	}
	empty := normalizeSkillUseKey("", 5)
	if empty.Path != "strength" || empty.Rank != 5 {
		t.Fatalf("empty path normalize = %+v, want strength/5", empty)
	}
}

func TestApplyPathMasteryUsesCrossesMilestoneWithNewEfficiency(t *testing.T) {
	masteries := models.PathMasteriesMap{"fire": gamefx.PathMasteryMasterPoints - 1}
	if !applyPathMasteryUses(masteries, "fire", 9, 2) {
		t.Fatal("milestone crossing did not grant Chân Ý")
	}
	// First use at Ordinary: +300. Second use at Master: round(300 * 0.8) = 240.
	want := gamefx.PathMasteryMasterPoints - 1 + 300 + 240
	if got := masteries["fire"]; got != want {
		t.Fatalf("milestone crossing points=%d, want %d", got, want)
	}
}
