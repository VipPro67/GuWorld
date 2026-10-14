package services

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func regenTestPlayer() models.Player {
	p := models.Player{}
	p.Race = "human"
	p.Rank = 1
	p.AptitudeType = "Giáp"
	p.Lifespan = 100
	p.BaseHP = 100
	p.HP = 10
	p.MaxMana = 100
	p.CurrentMana = 10
	p.MaxThought = 100
	p.CurrentThought = 10
	p.RegenRate = 4
	p.MaxSoul = 100
	p.SoulHP = 10
	p.TimeSpeedMultiplier = 1
	return p
}

func TestCalculateAutomaticRegenRatesUsesBlockV2DisablePolicy(t *testing.T) {
	oldRaces := gamefx.RaceRegistry
	t.Cleanup(func() { gamefx.RaceRegistry = oldRaces })
	gamefx.RaceRegistry = nil // use the built-in Block-v2-compatible fallback

	human := regenTestPlayer()
	humanRates := calculateAutomaticRegenRates(&human, 0)
	if humanRates.mana <= 0 || humanRates.hp <= 0 || humanRates.thought <= 0 || humanRates.soul <= 0 {
		t.Fatalf("human automatic regen should be positive: %+v", humanRates)
	}

	zombie := regenTestPlayer()
	zombie.Race = "cuong_thi"
	zombieRates := calculateAutomaticRegenRates(&zombie, 0)
	if zombieRates != (automaticRegenRates{}) {
		t.Fatalf("cuong_thi automatic regen = %+v, want all zero", zombieRates)
	}
}

func TestApplyProjectedStatusDoesNotRegenerateCuongThi(t *testing.T) {
	oldRaces := gamefx.RaceRegistry
	t.Cleanup(func() { gamefx.RaceRegistry = oldRaces })
	gamefx.RaceRegistry = nil

	now := time.Now()
	p := regenTestPlayer()
	p.Race = "cuong_thi"
	p.LastManaUpdate = now.Add(-time.Hour)
	wantHP, wantMana := p.HP, p.CurrentMana
	wantThought, wantSoul := p.CurrentThought, p.SoulHP

	changed, _ := applyProjectedStatus(&p, 0, 1, now)

	if p.HP != wantHP || p.CurrentMana != wantMana || p.CurrentThought != wantThought || p.SoulHP != wantSoul {
		t.Fatalf("cuong_thi pools changed: hp=%v mana=%v thought=%v soul=%v", p.HP, p.CurrentMana, p.CurrentThought, p.SoulHP)
	}
	if p.HPRegenRate != 0 || p.ManaRegenRate != 0 || p.SoulRegenRate != 0 {
		t.Fatalf("cuong_thi exposed non-zero regen rates: hp=%v mana=%v soul=%v", p.HPRegenRate, p.ManaRegenRate, p.SoulRegenRate)
	}
	if p.ThoughtRegenRate != 0 {
		t.Fatalf("cuong_thi exposed thought regen rate %v, want zero", p.ThoughtRegenRate)
	}
	if !changed {
		t.Fatal("elapsed disabled interval must be consumed for last_mana_update persistence")
	}
}

func TestCalculateAutomaticRegenRatesDoesNotMutateThoughtRate(t *testing.T) {
	p := regenTestPlayer()
	want := p.RegenRate

	calculateAutomaticRegenRates(&p, 0)
	calculateAutomaticRegenRates(&p, 0)

	if p.RegenRate != want {
		t.Fatalf("stored thought regen rate = %v, want unchanged %v", p.RegenRate, want)
	}
}

func TestManaRegenBonusAppliedOnce(t *testing.T) {
	oldConstitutions := gamefx.ConstitutionRegistry
	t.Cleanup(func() { gamefx.ConstitutionRegistry = oldConstitutions })
	gamefx.ConstitutionRegistry = map[string]gamefx.ConstitutionDef{
		"test": {Key: "test", Effects: gamefx.Effects{ManaRegenBonus: 1}},
	}

	p := regenTestPlayer()
	p.Constitution = "test"
	rates := calculateAutomaticRegenRates(&p, 0)
	if rates.mana != 20 { // Giáp base 10/hour, +100% exactly once.
		t.Fatalf("mana regen = %v, want 20", rates.mana)
	}
}

func TestImmortalManaRegenProducesAnnualApertureEssence(t *testing.T) {
	p := regenTestPlayer()
	p.Rank = 6
	p.SubRank = 4

	rates := calculateAutomaticRegenRates(&p, 20)
	if rates.mana != 2 {
		t.Fatalf("immortal mana regen = %v/hour before time rate, want 2", rates.mana)
	}

	now := time.Now()
	p.LastManaUpdate = now.Add(-24 * time.Minute) // 10 hours / time_rate 25
	p.CurrentMana = 0
	applyProjectedStatus(&p, 20, 25, now)
	if p.CurrentMana != 20 {
		t.Fatalf("mana after one internal year = %v, want annual production 20", p.CurrentMana)
	}
}

func TestApplyProjectedStatusLifespanImmunityForRank10(t *testing.T) {
	now := time.Now()

	// Rank 10 Vĩnh Sinh: lifespan must NOT decrease over time
	p10 := regenTestPlayer()
	p10.Rank = 10
	p10.Lifespan = 100
	p10.LastManaUpdate = now.Add(-36000 * time.Second) // 1 year passed
	applyProjectedStatus(&p10, 0, 1, now)
	if p10.Lifespan != 100 {
		t.Fatalf("rank 10 lifespan changed: got %v, want 100", p10.Lifespan)
	}

	// Rank 1 mortal: lifespan MUST decrease over time
	p1 := regenTestPlayer()
	p1.Rank = 1
	p1.Lifespan = 100
	p1.LastManaUpdate = now.Add(-36000 * time.Second) // 1 year passed
	applyProjectedStatus(&p1, 0, 1, now)
	if p1.Lifespan >= 100 {
		t.Fatalf("rank 1 lifespan did not decrease: got %v, want < 100", p1.Lifespan)
	}
}

