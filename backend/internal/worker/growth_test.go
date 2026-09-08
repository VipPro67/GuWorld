package worker

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestApplyGrowthProgressWorker(t *testing.T) {
	now := time.Now().Add(-1 * time.Hour)
	gu := &models.GuInstance{
		InstanceID:  "inst-1",
		GuID:        "spring_autumn_cicada",
		Rank:        6,
		Status:      "ACTIVE",
		Equipped:    true,
		EquippedAt:  &now,
		GrowthHours: 0,
	}

	spec := models.GuDictionaryItem{
		ID:   "spring_autumn_cicada",
		Rank: 6,
		Effects: models.GuEffects{
			Growth: &models.GrowthEffect{
				HoursToMax: 100,
				MaxAttack:  100,
				MaxDefense: 50,
				MaxHP:      500,
				MaxSpeed:   20,
			},
		},
	}

	atk, def, hp, spd := ApplyGrowthProgressWorker(gu, spec, 10.0)

	if gu.GrowthHours != 10.0 {
		t.Errorf("expected GrowthHours = 10.0, got %f", gu.GrowthHours)
	}
	if atk != 10.0 {
		t.Errorf("expected atk = 10.0, got %f", atk)
	}
	if def != 5.0 {
		t.Errorf("expected def = 5.0, got %f", def)
	}
	if hp != 50.0 {
		t.Errorf("expected hp = 50.0, got %f", hp)
	}
	if spd != 2.0 {
		t.Errorf("expected spd = 2.0, got %f", spd)
	}
}
