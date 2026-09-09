package worker

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/lib/pq"
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

	var succeededIDs []string

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

		_, err = tx.Exec(`
			UPDATE players SET
				x = $1, y = $2, current_region = $3,
				cleared_cells = $4, visited_subzones = $5
			WHERE id = $6
		`,
			p.X, p.Y, p.CurrentRegion,
			clearedCellsVal, visitedSubzonesVal,
			p.ID,
		)
		if err != nil {
			tx.Rollback()
			logger.S.Warnf("Batch update movement postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}

		if err := tx.Commit(); err != nil {
			logger.S.Warnf("Batch movement: failed to commit for player %s: %v", p.ID, err)
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

	// Parse all payloads
	parsed := make([]PlayerStatsPayload, 0, len(payloads))
	for _, pStr := range payloads {
		var p PlayerStatsPayload
		if err := json.Unmarshal([]byte(pStr), &p); err != nil {
			logger.S.Errorf("Failed to unmarshal PlayerStatsPayload: %v", err)
			continue
		}
		parsed = append(parsed, p)
	}

	if len(parsed) == 0 {
		return cache.RemovePendingIDs("player_stats")
	}

	// Process in chunks to reduce lock contention
	const chunkSize = 25
	var allIDs []string

	for i := 0; i < len(parsed); i += chunkSize {
		end := i + chunkSize
		if end > len(parsed) {
			end = len(parsed)
		}
		chunk := parsed[i:end]

		ids := make([]string, len(chunk))
		hps := make([]float64, len(chunk))
		soulHps := make([]int, len(chunk))
		currentManas := make([]float64, len(chunk))
		maxManas := make([]float64, len(chunk))
		currentThoughts := make([]float64, len(chunk))
		lifespans := make([]float64, len(chunk))
		timeSpeedMultipliers := make([]float64, len(chunk))
		apertureStresses := make([]float64, len(chunk))
		lastCultivateTimes := make([]time.Time, len(chunk))
		lastManaUpdates := make([]time.Time, len(chunk))

		for j, p := range chunk {
			ids[j] = p.ID
			hps[j] = p.HP
			soulHps[j] = p.SoulHP
			currentManas[j] = p.CurrentMana
			maxManas[j] = p.MaxMana
			currentThoughts[j] = p.CurrentThought
			lifespans[j] = p.Lifespan
			timeSpeedMultipliers[j] = p.TimeSpeedMultiplier
			apertureStresses[j] = p.ApertureStress
			lastCultivateTimes[j] = p.LastCultivateTime
			lastManaUpdates[j] = p.LastManaUpdate
		}

		// Each chunk runs in its own short transaction with lock/statement timeout
		tx, err := db.DB.BeginTx(ctx, nil)
		if err != nil {
			logger.S.Warnf("Batch stats: failed to begin tx: %v", err)
			return err
		}

		if _, err := tx.Exec(`SET LOCAL lock_timeout = '5000'`); err != nil {
			logger.S.Warnf("Batch stats: failed to set lock_timeout: %v", err)
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`SET LOCAL statement_timeout = '8000'`); err != nil {
			logger.S.Warnf("Batch stats: failed to set statement_timeout: %v", err)
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`
			WITH updates(id, hp, soul_hp, current_mana, max_mana, current_thought, lifespan,
			            time_speed_multiplier, aperture_stress, last_cultivate_time, last_mana_update) AS (
				SELECT unnest($1::text[]), unnest($2::float8[]), unnest($3::int[]),
				       unnest($4::float8[]), unnest($5::float8[]), unnest($6::float8[]),
				       unnest($7::float8[]), unnest($8::float8[]), unnest($9::float8[]),
				       unnest($10::timestamptz[]), unnest($11::timestamptz[])
			)
			UPDATE players p SET
				hp = u.hp, soul_hp = u.soul_hp, current_mana = u.current_mana, max_mana = u.max_mana,
				current_thought = u.current_thought, lifespan = u.lifespan, time_speed_multiplier = u.time_speed_multiplier,
				aperture_stress = u.aperture_stress, last_cultivate_time = u.last_cultivate_time,
				last_mana_update = u.last_mana_update,
				version = version + 1
			FROM updates u WHERE p.id = u.id::uuid AND (p.last_mana_update IS NULL OR p.last_mana_update <= u.last_mana_update)
		`, pq.Array(ids), pq.Array(hps), pq.Array(soulHps), pq.Array(currentManas), pq.Array(maxManas),
			pq.Array(currentThoughts), pq.Array(lifespans), pq.Array(timeSpeedMultipliers),
			pq.Array(apertureStresses), pq.Array(lastCultivateTimes), pq.Array(lastManaUpdates))
		if err != nil {
			tx.Rollback()
			logger.S.Warnf("Batch update stats chunk failed: %v", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			logger.S.Warnf("Batch stats: failed to commit tx: %v", err)
			return err
		}

		allIDs = append(allIDs, ids...)
	}

	_ = cache.RemovePendingIDs("player_stats", allIDs...)
	return nil
}

