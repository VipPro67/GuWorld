package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type PlayerMovementPayload struct {
	ID              string          `json:"id"`
	CurrentMana     float64         `json:"current_mana"`
	MaxMana         float64         `json:"max_mana"`
	HP              float64         `json:"hp"`
	X               int             `json:"x"`
	Y               int             `json:"y"`
	CurrentRegion   string          `json:"current_region"`
	ClearedCells    json.RawMessage `json:"cleared_cells,omitempty"`
	ActiveCombat    json.RawMessage `json:"active_combat,omitempty"`
	PrimevalStones  int64           `json:"primeval_stones"`
	HumanQi         float64         `json:"human_qi"`
	VisitedSubzones json.RawMessage `json:"visited_subzones,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PlayerStatsPayload struct {
	ID                  string    `json:"id"`
	HP                  float64   `json:"hp"`
	SoulHP              int       `json:"soul_hp"`
	CurrentMana         float64   `json:"current_mana"`
	MaxMana             float64   `json:"max_mana"`
	CurrentThought      float64   `json:"current_thought"`
	Lifespan            float64   `json:"lifespan"`
	TimeSpeedMultiplier float64   `json:"time_speed_multiplier"`
	ApertureStress      float64   `json:"aperture_stress"`
	HumanQi             float64   `json:"human_qi"`
	PrimevalStones      int64     `json:"primeval_stones"`
	ImmortalStones      int64     `json:"immortal_stones"`
	LastManaUpdate      time.Time `json:"last_mana_update"`
	LastCultivateTime   time.Time `json:"last_cultivate_time"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func init() {
	RegisterBatchHandler("player_movement", handlePlayerMovementBatch)
	RegisterBatchHandler("player_stats", handlePlayerStatsBatch)
}

func handlePlayerMovementBatch(ctx context.Context, payloads []string) error {
	if len(payloads) == 0 {
		return nil
	}

	now := time.Now()
	var succeededIDs []string

	for _, pStr := range payloads {
		var p PlayerMovementPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerMovementPayload: %v", err)
			continue
		}

		var activeCombatVal any
		if len(p.ActiveCombat) > 0 && string(p.ActiveCombat) != "null" {
			activeCombatVal = string(p.ActiveCombat)
		}

		var clearedCellsVal any = "{}"
		if len(p.ClearedCells) > 0 {
			clearedCellsVal = string(p.ClearedCells)
		}

		var visitedSubzonesVal any = "[]"
		if len(p.VisitedSubzones) > 0 {
			visitedSubzonesVal = string(p.VisitedSubzones)
		}

		_, err := db.DB.ExecContext(ctx, `
			UPDATE players SET 
				current_mana = $1, x = $2, y = $3, current_region = $4, 
				cleared_cells = $5, active_combat = $6, last_mana_update = $7, 
				primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11, 
				visited_subzones = $12
			WHERE id = $13
		`,
			p.CurrentMana, p.X, p.Y, p.CurrentRegion,
			clearedCellsVal, activeCombatVal, now,
			p.PrimevalStones, p.HP, p.MaxMana, p.HumanQi,
			visitedSubzonesVal, p.ID,
		)
		if err != nil {
			logger.S.Warnf("Batch update movement postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}
		succeededIDs = append(succeededIDs, p.ID)
	}

	if len(succeededIDs) > 0 {
		_ = cache.RemovePendingIDs("player_movement", succeededIDs...)
	}

	return nil
}

func handlePlayerStatsBatch(ctx context.Context, payloads []string) error {
	if len(payloads) == 0 {
		return nil
	}

	var succeededIDs []string

	for _, pStr := range payloads {
		var p PlayerStatsPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerStatsPayload: %v", err)
			continue
		}

		_, err := db.DB.ExecContext(ctx, `
			UPDATE players SET 
				hp = $1, soul_hp = $2, current_mana = $3, max_mana = $4, 
				current_thought = $5, lifespan = $6, time_speed_multiplier = $7, 
				aperture_stress = $8, last_cultivate_time = $9, last_mana_update = $10,
				human_qi = $11, primeval_stones = $12, immortal_stones = $13,
				version = version + 1
			WHERE id = $14
		`,
			p.HP, p.SoulHP, p.CurrentMana, p.MaxMana,
			p.CurrentThought, p.Lifespan, p.TimeSpeedMultiplier,
			p.ApertureStress, p.LastCultivateTime, p.LastManaUpdate,
			p.HumanQi, p.PrimevalStones, p.ImmortalStones, p.ID,
		)
		if err != nil {
			logger.S.Warnf("Batch update stats postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}
		succeededIDs = append(succeededIDs, p.ID)
	}

	if len(succeededIDs) > 0 {
		_ = cache.RemovePendingIDs("player_stats", succeededIDs...)
	}

	return nil
}

