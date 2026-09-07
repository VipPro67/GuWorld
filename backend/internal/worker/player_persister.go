package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	ID             string    `json:"id"`
	HP             float64   `json:"hp"`
	CurrentMana    float64   `json:"current_mana"`
	MaxMana        float64   `json:"max_mana"`
	CurrentThought float64   `json:"current_thought"`
	ApertureStress float64   `json:"aperture_stress"`
	HumanQi        float64   `json:"human_qi"`
	PrimevalStones int64     `json:"primeval_stones"`
	ImmortalStones int64     `json:"immortal_stones"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func init() {
	RegisterBatchHandler("player_movement", handlePlayerMovementBatch)
	RegisterBatchHandler("player_stats", handlePlayerStatsBatch)
}

func handlePlayerMovementBatch(ctx context.Context, payloads []string) error {
	if len(payloads) == 0 {
		return nil
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx for player_movement batch: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE players SET 
			current_mana = $1, x = $2, y = $3, current_region = $4, 
			cleared_cells = $5, active_combat = $6, last_mana_update = $7, 
			primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11, 
			visited_subzones = $12
		WHERE id = $13
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare stmt for player_movement batch: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
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

		_, err := stmt.ExecContext(ctx,
			p.CurrentMana, p.X, p.Y, p.CurrentRegion,
			clearedCellsVal, activeCombatVal, now,
			p.PrimevalStones, p.HP, p.MaxMana, p.HumanQi,
			visitedSubzonesVal, p.ID,
		)
		if err != nil {
			logger.S.Errorf("Failed to batch update player movement for %s: %v", p.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit player_movement batch: %w", err)
	}

	logger.S.Debugf("Successfully batch persisted %d player movement updates to PostgreSQL", len(payloads))
	return nil
}

func handlePlayerStatsBatch(ctx context.Context, payloads []string) error {
	if len(payloads) == 0 {
		return nil
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx for player_stats batch: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE players SET 
			hp = $1, current_mana = $2, max_mana = $3, 
			current_thought = $4, aperture_stress = $5, human_qi = $6, 
			primeval_stones = $7, immortal_stones = $8
		WHERE id = $9
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare stmt for player_stats batch: %w", err)
	}
	defer stmt.Close()

	for _, pStr := range payloads {
		var p PlayerStatsPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerStatsPayload: %v", err)
			continue
		}

		_, err := stmt.ExecContext(ctx,
			p.HP, p.CurrentMana, p.MaxMana,
			p.CurrentThought, p.ApertureStress, p.HumanQi,
			p.PrimevalStones, p.ImmortalStones, p.ID,
		)
		if err != nil {
			logger.S.Errorf("Failed to batch update player stats for %s: %v", p.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit player_stats batch: %w", err)
	}

	logger.S.Debugf("Successfully batch persisted %d player stats updates to PostgreSQL", len(payloads))
	return nil
}

