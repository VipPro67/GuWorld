package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRandomConstitutionEffect(t *testing.T) {
	item := models.ItemSpec{ID: "ITEM_RANDOM_CONSTITUTION", Name: "Hoán Thể Đan"}
	for i := 0; i < 50; i++ {
		p := &models.Player{CharacterState: models.CharacterState{Constitution: "solar_sun_moon_body"}}
		msg, err := randomConstitutionEffect(nil, p, item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg == "" {
			t.Fatalf("expected a message")
		}
		if p.Constitution == "solar_sun_moon_body" {
			t.Fatalf("constitution should change from previous value")
		}
		if !gamefx.ValidConstitutions()[p.Constitution] {
			t.Fatalf("invalid constitution key %q", p.Constitution)
		}
	}
}

func TestRandomRaceEffect(t *testing.T) {
	item := models.ItemSpec{ID: "ITEM_RANDOM_RACE", Name: "Hoán Huyết Đan"}
	for i := 0; i < 50; i++ {
		p := &models.Player{CharacterState: models.CharacterState{Race: "human"}}
		msg, err := randomRaceEffect(nil, p, item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg == "" {
			t.Fatalf("expected a message")
		}
		if p.Race == "human" {
			t.Fatalf("race should change from previous value")
		}
		if !gamefx.ValidRaces()[p.Race] {
			t.Fatalf("invalid race key %q", p.Race)
		}
	}
}
