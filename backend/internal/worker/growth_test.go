package worker

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func mustScanGuEffects(t *testing.T, raw string) models.GuEffects {
	t.Helper()
	var eff models.GuEffects
	if err := eff.Scan(raw); err != nil {
		t.Fatalf("scan test effects: %v", err)
	}
	return eff
}

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
		ID:      "spring_autumn_cicada",
		Rank:    6,
		Effects: mustScanGuEffects(t, `[{"id":"GROWTH","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"GROWTH_CAP","value":100},{"op":"STAT_MOD","stat":"DEFENSE","sub_type":"GROWTH_CAP","value":50},{"op":"STAT_MOD","stat":"HP","sub_type":"GROWTH_CAP","value":500},{"op":"STAT_MOD","stat":"SPEED","sub_type":"GROWTH_CAP","value":20},{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":100}}]}]`),
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
