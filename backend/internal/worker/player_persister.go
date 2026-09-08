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

	now := time.Now()
	var succeededIDs []string

	for _, p := range parsed {
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

		result, err := db.DB.ExecContext(ctx, `
			UPDATE players SET
				current_mana = $1, x = $2, y = $3, current_region = $4,
				cleared_cells = $5, active_combat = $6, last_mana_update = $7,
				primeval_stones = $8, hp = $9, max_mana = $10, human_qi = $11,
				visited_subzones = $12,
				version = version + 1
			WHERE id = $13 AND version = $14
		`,
			p.CurrentMana, p.X, p.Y, p.CurrentRegion,
			clearedCellsVal, activeCombatVal, now,
			p.PrimevalStones, p.HP, p.MaxMana, p.HumanQi,
			visitedSubzonesVal, p.ID, p.Version,
		)
		if err != nil {
			logger.S.Warnf("Batch update movement postponed for player %s (locked/retry): %v", p.ID, err)
			continue
		}

		affected, _ := result.RowsAffected()
		if affected == 0 {
			// Version mismatch: a newer update already ran. Drop this stale
			// payload instead of retrying forever with the same stale version.
			logger.S.Infof("Batch movement dropped for player %s: version %d stale", p.ID, p.Version)
			Metrics.RecordDrop(1)
			succeededIDs = append(succeededIDs, p.ID)
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

	// Batch UPDATE using unnest for all affected players in one query
	ids := make([]string, len(parsed))
	hps := make([]float64, len(parsed))
	soulHps := make([]int, len(parsed))
	currentManas := make([]float64, len(parsed))
	maxManas := make([]float64, len(parsed))
	currentThoughts := make([]float64, len(parsed))
	lifespans := make([]float64, len(parsed))
	timeSpeedMultipliers := make([]float64, len(parsed))
	apertureStresses := make([]float64, len(parsed))
	lastCultivateTimes := make([]time.Time, len(parsed))
	lastManaUpdates := make([]time.Time, len(parsed))
	humanQis := make([]float64, len(parsed))
	primevalStones := make([]int64, len(parsed))
	immortalStones := make([]int64, len(parsed))

	for i, p := range parsed {
		ids[i] = p.ID
		hps[i] = p.HP
		soulHps[i] = p.SoulHP
		currentManas[i] = p.CurrentMana
		maxManas[i] = p.MaxMana
		currentThoughts[i] = p.CurrentThought
		lifespans[i] = p.Lifespan
		timeSpeedMultipliers[i] = p.TimeSpeedMultiplier
		apertureStresses[i] = p.ApertureStress
		lastCultivateTimes[i] = p.LastCultivateTime
		lastManaUpdates[i] = p.LastManaUpdate
		humanQis[i] = p.HumanQi
		primevalStones[i] = p.PrimevalStones
		immortalStones[i] = p.ImmortalStones
	}

	_, err := db.DB.ExecContext(ctx, `
		WITH updates(id, hp, soul_hp, current_mana, max_mana, current_thought, lifespan,
		            time_speed_multiplier, aperture_stress, last_cultivate_time, last_mana_update,
		            human_qi, primeval_stones, immortal_stones) AS (
			SELECT unnest($1::text[]), unnest($2::float8[]), unnest($3::int[]),
			       unnest($4::float8[]), unnest($5::float8[]), unnest($6::float8[]),
			       unnest($7::float8[]), unnest($8::float8[]), unnest($9::float8[]),
			       unnest($10::timestamptz[]), unnest($11::timestamptz[]),
			       unnest($12::float8[]), unnest($13::int8[]), unnest($14::int8[])
		)
		UPDATE players p SET
			hp = u.hp, soul_hp = u.soul_hp, current_mana = u.current_mana, max_mana = u.max_mana,
			current_thought = u.current_thought, lifespan = u.lifespan, time_speed_multiplier = u.time_speed_multiplier,
			aperture_stress = u.aperture_stress, last_cultivate_time = u.last_cultivate_time,
			last_mana_update = u.last_mana_update, human_qi = u.human_qi,
			primeval_stones = u.primeval_stones, immortal_stones = u.immortal_stones,
			version = version + 1
		FROM updates u WHERE p.id = u.id::uuid
	`, pq.Array(ids), pq.Array(hps), pq.Array(soulHps), pq.Array(currentManas), pq.Array(maxManas),
		pq.Array(currentThoughts), pq.Array(lifespans), pq.Array(timeSpeedMultipliers),
		pq.Array(apertureStresses), pq.Array(lastCultivateTimes), pq.Array(lastManaUpdates),
		pq.Array(humanQis), pq.Array(primevalStones), pq.Array(immortalStones))
	if err != nil {
		logger.S.Warnf("Batch update stats failed: %v", err)
		return err
	}

	_ = cache.RemovePendingIDs("player_stats", ids...)
	return nil
}

