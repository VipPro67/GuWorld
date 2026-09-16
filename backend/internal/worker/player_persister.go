package worker

import (
	"context"
	"encoding/json"
	"sort"
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
	PrimevalStones  float64         `json:"primeval_stones"`
	HumanQi         float64         `json:"human_qi"`
	VisitedSubzones json.RawMessage `json:"visited_subzones,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Version         int             `json:"version"`
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

	// Parse all payloads first
	type parsedPayload struct {
		PlayerMovementPayload
		raw string
	}
	parsed := make([]parsedPayload, 0, len(payloads))
	for _, pStr := range payloads {
		var p PlayerMovementPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerMovementPayload: %v", err)
			continue
		}
		parsed = append(parsed, parsedPayload{PlayerMovementPayload: p, raw: pStr})
	}

	// Sort by UpdatedAt so we process older updates first; newer ones will
	// win the version check.
	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].UpdatedAt.Before(parsed[j].UpdatedAt)
	})

	for _, p := range parsed {
		var clearedCellsVal any = "{}"
		if len(p.ClearedCells) > 0 {
			clearedCellsVal = string(p.ClearedCells)
		}

		var visitedSubzonesVal any = "[]"
		if len(p.VisitedSubzones) > 0 {
			visitedSubzonesVal = string(p.VisitedSubzones)
		}

		// Each update in its own short transaction with lock/statement timeout
		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			logger.S.Warnf("Batch movement: failed to begin tx for player %s: %v", p.ID, err)
			continue
		}

		if _, err := tx.Exec(`SET LOCAL lock_timeout = '3000'`); err != nil {
			logger.S.Warnf("Batch movement: failed to set lock_timeout for player %s: %v", p.ID, err)
			tx.Rollback()
			continue
		}
		if _, err := tx.Exec(`SET LOCAL statement_timeout = '5000'`); err != nil {
			logger.S.Warnf("Batch movement: failed to set statement_timeout for player %s: %v", p.ID, err)
			tx.Rollback()
			continue
		}

		result, err := tx.Exec(`
			UPDATE players SET
				x = $1, y = $2, current_region = $3,
				cleared_cells = $4, visited_subzones = $5,
				human_qi = $6, version = GREATEST(version, $8)
			WHERE id = $7
			  AND version IN ($8 - 1, $8)
			  AND (last_mana_update IS NULL OR last_mana_update <= $9)
		`,
			p.X, p.Y, p.CurrentRegion,
			clearedCellsVal, visitedSubzonesVal,
			p.HumanQi,
			p.ID, p.Version, p.UpdatedAt,
		)
		if err != nil {
			tx.Rollback()
			logger.S.Warnf("Batch update movement postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}

		rows, err := result.RowsAffected()
		if err != nil || rows > 1 {
			tx.Rollback()
			logger.S.Warnf("Batch movement: failed checking player %s result: rows=%d err=%v", p.ID, rows, err)
			continue
		}
		if rows == 0 {
			tx.Rollback()
			if _, ackErr := cache.AcknowledgePendingJSON("player_movement", p.ID, p.raw); ackErr != nil {
				logger.S.Warnf("Batch movement: failed to acknowledge stale player %s: %v", p.ID, ackErr)
			}
			continue
		}
		if err := tx.Commit(); err != nil {
			logger.S.Warnf("Batch movement: failed to commit for player %s: %v", p.ID, err)
			continue
		}
		if _, err := cache.AcknowledgePendingJSON("player_movement", p.ID, p.raw); err != nil {
			logger.S.Warnf("Batch movement: failed to acknowledge player %s: %v", p.ID, err)
		}
	}

	return nil
}

func handlePlayerStatsBatch(ctx context.Context, payloads []string) error {
	if len(payloads) == 0 {
		return nil
	}

	// Parse and deduplicate payloads by player ID, keeping the newest UpdatedAt
	type pendingStats struct {
		payload PlayerStatsPayload
		raw     string
	}
	latestByPlayer := make(map[string]pendingStats)
	for _, pStr := range payloads {
		var p PlayerStatsPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerStatsPayload: %v", err)
			continue
		}
		if existing, ok := latestByPlayer[p.ID]; !ok || p.UpdatedAt.After(existing.payload.UpdatedAt) {
			latestByPlayer[p.ID] = pendingStats{payload: p, raw: pStr}
		}
	}

	if len(latestByPlayer) == 0 {
		return nil
	}

	for _, pending := range latestByPlayer {
		p := pending.payload
		// Each player update runs in its own short transaction with lock/statement timeout
		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			logger.S.Warnf("Batch stats: failed to begin tx for player %s: %v", p.ID, err)
			continue
		}

		if _, err := tx.Exec(`SET LOCAL lock_timeout = '3000'`); err != nil {
			logger.S.Warnf("Batch stats: failed to set lock_timeout for player %s: %v", p.ID, err)
			tx.Rollback()
			continue
		}
		if _, err := tx.Exec(`SET LOCAL statement_timeout = '5000'`); err != nil {
			logger.S.Warnf("Batch stats: failed to set statement_timeout for player %s: %v", p.ID, err)
			tx.Rollback()
			continue
		}

		result, err := tx.Exec(`
			UPDATE players SET
				hp = $1, soul_hp = $2, current_mana = $3, max_mana = $4,
				current_thought = $5, lifespan = $6, time_speed_multiplier = $7,
				aperture_stress = $8, last_cultivate_time = $9,
				last_mana_update = $10,
				version = version + 1
			WHERE id = $11 AND (last_mana_update IS NULL OR last_mana_update <= $10)
		`,
			p.HP, p.SoulHP, p.CurrentMana, p.MaxMana,
			p.CurrentThought, p.Lifespan, p.TimeSpeedMultiplier,
			p.ApertureStress, p.LastCultivateTime,
			p.LastManaUpdate,
			p.ID,
		)
		if err != nil {
			tx.Rollback()
			logger.S.Warnf("Batch update stats postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}

		rows, err := result.RowsAffected()
		if err != nil || rows > 1 {
			tx.Rollback()
			logger.S.Warnf("Batch stats: failed checking player %s result: rows=%d err=%v", p.ID, rows, err)
			continue
		}
		if rows == 0 {
			tx.Rollback()
			if _, ackErr := cache.AcknowledgePendingJSON("player_stats", p.ID, pending.raw); ackErr != nil {
				logger.S.Warnf("Batch stats: failed to acknowledge stale player %s: %v", p.ID, ackErr)
			}
			continue
		}
		if err := tx.Commit(); err != nil {
			logger.S.Warnf("Batch stats: failed to commit for player %s: %v", p.ID, err)
			continue
		}
		if _, err := cache.AcknowledgePendingJSON("player_stats", p.ID, pending.raw); err != nil {
			logger.S.Warnf("Batch stats: failed to acknowledge player %s: %v", p.ID, err)
		}
	}

	return nil
}
