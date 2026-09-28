package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var (
	growthWorkerStarted bool
	growthWorkerMutex   sync.Mutex
)

type guUpdate struct {
	InstanceID  string
	GrowthHours float64
	EquippedAt  time.Time
}

type playerUpdate struct {
	GrowthData string
	BaseHP     float64
}

type playerGrowthBatch struct {
	guUpdates    []guUpdate
	playerUpdate *playerUpdate
}

func StartGrowthWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	growthWorkerMutex.Lock()
	if growthWorkerStarted {
		growthWorkerMutex.Unlock()
		return
	}
	growthWorkerStarted = true
	growthWorkerMutex.Unlock()

	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Growth Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Growth Worker stopped")
				return
			case <-ticker.C:
				processGrowthTick(ctx, db)
			}
		}
	}()
}

// processGrowthTick advances Gu growth for every equipped ACTIVE Gu that has a
// growth profile. Instead of one massive transaction for all players, it batch-reads and
// processes players in chunks of 50 to prevent statement timeouts and reduce lock contention.
func processGrowthTick(ctx context.Context, db *sql.DB) {
	// ── Step 1: find affected player IDs ──────────────────────────────────
	playerIDs, err := findGrowthAffectedPlayers(ctx, db)
	if err != nil {
		logger.S.Errorf("Growth Worker Error finding affected players: %v", err)
		return
	}
	if len(playerIDs) == 0 {
		return
	}

	const batchSize = 50
	for i := 0; i < len(playerIDs); i += batchSize {
		end := i + batchSize
		if end > len(playerIDs) {
			end = len(playerIDs)
		}
		processGrowthBatch(ctx, db, playerIDs[i:end])
	}
}

func processGrowthBatch(ctx context.Context, db *sql.DB, playerIDs []string) {
	pqIDs := pq.Array(playerIDs)

	fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fetchCancel()

	// ── Step 2: batch-fetch player metadata ───────────────────────────────
	type playerRow struct {
		GrowthData []byte
		BaseHP     float64
		TimeSpeed  float64
	}
	playerMap := make(map[string]*playerRow, len(playerIDs))

	pRows, err := db.QueryContext(fetchCtx, `
		SELECT id, growth_data, base_hp, time_speed_multiplier
		FROM players WHERE id = ANY($1::uuid[])`, pqIDs)
	if err != nil {
		logger.S.Errorf("Growth Worker Error fetching players: %v", err)
		return
	}
	defer pRows.Close()

	for pRows.Next() {
		var (
			id string
			pr playerRow
			gd sql.NullString
		)
		if err := pRows.Scan(&id, &gd, &pr.BaseHP, &pr.TimeSpeed); err != nil {
			logger.S.Errorf("Growth Worker Error scanning player row: %v", err)
			continue
		}
		if gd.Valid {
			pr.GrowthData = []byte(gd.String)
		}
		playerMap[id] = &pr
	}
	if err := pRows.Err(); err != nil {
		logger.S.Errorf("Growth Worker Error iterating players: %v", err)
		return
	}

	// ── Step 3: batch-fetch equipped active player_gu rows ───────────────────
	type guRow struct {
		PlayerID    string
		InstanceID  string
		GuID        string
		Rank        int
		State       string
		Equipped    bool
		IsVital     bool
		IntegrTime  float64
		EquippedAt  *time.Time
		GrowthHours float64
	}

	guByPlayer := make(map[string][]guRow)

	gRows, err := db.QueryContext(fetchCtx, `
		SELECT player_id::text, id, gu_id, gu_rank, state, equipped,
		       is_vital, integration_time, equipped_at, growth_hours
		FROM player_gu
		WHERE player_id = ANY($1::uuid[]) AND equipped = true AND state = 'ACTIVE'`, pqIDs)
	if err != nil {
		logger.S.Errorf("Growth Worker Error fetching player_gu: %v", err)
		return
	}
	defer gRows.Close()

	for gRows.Next() {
		var g guRow
		if err := gRows.Scan(
			&g.PlayerID, &g.InstanceID, &g.GuID, &g.Rank, &g.State,
			&g.Equipped, &g.IsVital, &g.IntegrTime, &g.EquippedAt, &g.GrowthHours,
		); err != nil {
			continue
		}
		guByPlayer[g.PlayerID] = append(guByPlayer[g.PlayerID], g)
	}
	if err := gRows.Err(); err != nil {
		logger.S.Errorf("Growth Worker Error iterating player_gu: %v", err)
		return
	}

	// ── Step 4: process growth in Go (same logic, no DB access) ──────────
	updatesByPlayer := make(map[string]*playerGrowthBatch)
	now := time.Now()

	type typeKey struct {
		guID string
		rank int
	}
	type growthGroup struct {
		spec     models.GuDictionaryItem
		active   []int
		maxHours float64
	}

	for pID, pRow := range playerMap {
		guList := guByPlayer[pID]
		if len(guList) == 0 {
			continue
		}

		// Build GuInventory from rows.
		guInv := make(models.GuInventory, len(guList))
		for i, g := range guList {
			guInv[i] = models.GuInstance{
				InstanceID:      g.InstanceID,
				GuID:            g.GuID,
				Rank:            g.Rank,
				Status:          g.State,
				Equipped:        g.Equipped,
				IsVital:         g.IsVital,
				IntegrationTime: g.IntegrTime,
				EquippedAt:      g.EquippedAt,
				GrowthHours:     g.GrowthHours,
			}
		}

		timeSpeed := pRow.TimeSpeed
		if timeSpeed <= 0 {
			timeSpeed = 1.0
		}
		timeSpeed += models.EquippedTimeSpeedBonus(guInv)

		var growthData models.GrowthData
		if err := json.Unmarshal(pRow.GrowthData, &growthData); err != nil {
			growthData = models.GrowthData{
				Boosts:     models.GrowthBoosts{},
				FullyGrown: []models.FullyGrownInfo{},
			}
		}

		groups := make(map[typeKey]*growthGroup)

		for i := range guInv {
			gu := &guInv[i]
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if _, hasGrowth := models.GrowthProfile(spec); !ok || !hasGrowth {
				continue
			}
			key := typeKey{guID: gu.GuID, rank: gu.Rank}
			g, exists := groups[key]
			if !exists {
				g = &growthGroup{spec: spec}
				groups[key] = g
			}
			if gu.GrowthHours > g.maxHours {
				g.maxHours = gu.GrowthHours
			}
			if gu.Equipped && gu.Status == "ACTIVE" {
				g.active = append(g.active, i)
			}
		}

		hasGrowthGu := false
		baseHP := pRow.BaseHP
		var pGuUpdates []guUpdate

		for _, g := range groups {
			spec := g.spec
			hasGrowthGu = true

			typeDone := false
			for _, fg := range growthData.FullyGrown {
				if fg.GuID == spec.ID && fg.Rank == spec.Rank {
					typeDone = true
					break
				}
			}
			if typeDone || len(g.active) == 0 {
				continue
			}

			// Initialize EquippedAt for legacy Gus.
			for _, idx := range g.active {
				gu := &guInv[idx]
				if gu.EquippedAt == nil || gu.EquippedAt.IsZero() {
					gu.EquippedAt = &now
					pGuUpdates = append(pGuUpdates, guUpdate{
						InstanceID:  gu.InstanceID,
						GrowthHours: gu.GrowthHours,
						EquippedAt:  now,
					})
				}
			}

			// Carrier = the active instance holding the type's progress.
			carrier := g.active[0]
			for _, idx := range g.active[1:] {
				if guInv[idx].GrowthHours > guInv[carrier].GrowthHours {
					carrier = idx
				}
			}
			gu := &guInv[carrier]

			// Bump non-carrier active anchors.
			for _, idx := range g.active {
				if idx == carrier {
					continue
				}
				guInv[idx].EquippedAt = &now
				pGuUpdates = append(pGuUpdates, guUpdate{
					InstanceID:  guInv[idx].InstanceID,
					GrowthHours: guInv[idx].GrowthHours,
					EquippedAt:  now,
				})
			}

			elapsed := now.Sub(*gu.EquippedAt).Hours() * timeSpeed
			if elapsed <= 0 {
				continue
			}

			if gu.GrowthHours < g.maxHours {
				gu.GrowthHours = g.maxHours
			}

			atk, def, hpBoost, spd := ApplyGrowthProgressWorker(gu, spec, elapsed)
			if atk != 0 || def != 0 || hpBoost != 0 || spd != 0 {
				growthData.Boosts.Attack += atk
				growthData.Boosts.Defense += def
				growthData.Boosts.HP += hpBoost
				growthData.Boosts.Speed += spd
				baseHP += hpBoost
			}

			growth, _ := models.GrowthProfile(spec)
			if gu.GrowthHours >= growth.HoursToMax {
				growthData.FullyGrown = append(growthData.FullyGrown, models.FullyGrownInfo{
					GuID:   gu.GuID,
					Rank:   gu.Rank,
					Attack: growth.MaxAttack, Defense: growth.MaxDefense,
					HP: growth.MaxHP, Speed: growth.MaxSpeed,
				})
			}

			// Carrier update.
			pGuUpdates = append(pGuUpdates, guUpdate{
				InstanceID:  gu.InstanceID,
				GrowthHours: gu.GrowthHours,
				EquippedAt:  now,
			})
		}

		if !hasGrowthGu {
			continue
		}

		growthRaw, _ := json.Marshal(growthData)
		updatesByPlayer[pID] = &playerGrowthBatch{
			guUpdates: pGuUpdates,
			playerUpdate: &playerUpdate{
				GrowthData: string(growthRaw),
				BaseHP:     baseHP,
			},
		}
	}

	if len(updatesByPlayer) == 0 {
		return
	}

	// ── Step 5: per-player isolated transactions ────────────────────────────
	for pID, pUpdate := range updatesByPlayer {
		if err := updatePlayerGrowthWithRetry(ctx, db, pID, pUpdate.guUpdates, pUpdate.playerUpdate); err != nil {
			logger.S.Errorf("Growth Worker Error updating player %s: %v", pID, err)
		}
	}
}

func updatePlayerGrowthWithRetry(ctx context.Context, db *sql.DB, playerID string, guUpdates []guUpdate, pUpdate *playerUpdate) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = updatePlayerGrowthOnce(ctx, db, playerID, guUpdates, pUpdate)
		if lastErr == nil {
			return nil
		}
		if !isDeadlockOrLockTimeout(lastErr) {
			return lastErr
		}
		logger.S.Warnf("Growth Worker: lock timeout/deadlock updating player %s (attempt %d/3): %v", playerID, attempt+1, lastErr)
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return fmt.Errorf("growth update failed for player %s after 3 attempts: %w", playerID, lastErr)
}

func updatePlayerGrowthOnce(ctx context.Context, db *sql.DB, playerID string, guUpdates []guUpdate, pUpdate *playerUpdate) error {
	txCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '5000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '3000'`); err != nil {
		return err
	}

	// All player/Gu mutation flows lock the parent player before player_gu.
	// Keeping this order aligned with hunger evaluation prevents a direct
	// players <-> player_gu deadlock cycle.
	var lockedPlayerID string
	if err := tx.QueryRow(`SELECT id::text FROM players WHERE id = $1::uuid FOR UPDATE`, playerID).Scan(&lockedPlayerID); err != nil {
		return err
	}

	if len(guUpdates) > 0 {
		ids := make([]string, len(guUpdates))
		ghs := make([]float64, len(guUpdates))
		eas := make([]time.Time, len(guUpdates))
		for i, u := range guUpdates {
			ids[i] = u.InstanceID
			ghs[i] = u.GrowthHours
			eas[i] = u.EquippedAt
		}
		_, err = tx.Exec(`
			UPDATE player_gu AS pg
			SET growth_hours = v.gh, equipped_at = v.ea, updated_at = NOW()
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS gh, unnest($3::timestamptz[]) AS ea) AS v
			WHERE pg.id = v.id AND pg.player_id = $4::uuid
		`, pq.Array(ids), pq.Array(ghs), pq.Array(eas), playerID)
		if err != nil {
			return err
		}
	}

	if pUpdate != nil {
		_, err = tx.Exec(`
			UPDATE players
			SET growth_data = $1::jsonb, base_hp = $2
			WHERE id = $3::uuid
		`, pUpdate.GrowthData, pUpdate.BaseHP, playerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// findGrowthAffectedPlayers returns players who own at least one equipped ACTIVE
// Gu with a growth profile, using the in-memory Gu registry to avoid heavy JSONB scans.
func findGrowthAffectedPlayers(ctx context.Context, db *sql.DB) ([]string, error) {
	type guKey struct {
		id   string
		rank int
	}
	var growthKeys []guKey

	for id, ranks := range models.GuRegistry {
		for rank, spec := range ranks {
			if _, ok := models.GrowthProfile(spec); ok {
				growthKeys = append(growthKeys, guKey{id: id, rank: rank})
			}
		}
	}

	if len(growthKeys) == 0 {
		return nil, nil
	}

	ids := make([]string, len(growthKeys))
	ranks := make([]int, len(growthKeys))
	for i, k := range growthKeys {
		ids[i] = k.id
		ranks[i] = k.rank
	}

	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(queryCtx, `
		SELECT DISTINCT pg.player_id::text
		FROM player_gu pg
		JOIN unnest($1::text[], $2::int[]) AS g(gu_id, gu_rank)
		  ON pg.gu_id = g.gu_id AND pg.gu_rank = g.gu_rank
		WHERE pg.equipped AND pg.state = 'ACTIVE'`,
		pq.Array(ids), pq.Array(ranks))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	return playerIDs, rows.Err()
}

func ApplyGrowthProgressWorker(gu *models.GuInstance, spec models.GuDictionaryItem, elapsedHours float64) (attack, defense, hp, speed float64) {
	g, ok := models.GrowthProfile(spec)
	if !ok || elapsedHours <= 0 {
		return 0, 0, 0, 0
	}
	maxHours := g.HoursToMax
	if maxHours <= 0 {
		maxHours = 168
	}
	oldHours := gu.GrowthHours
	newHours := oldHours + elapsedHours
	if newHours > maxHours {
		newHours = maxHours
	}
	progress := newHours / maxHours
	if progress > 1.0 {
		progress = 1.0
	}
	oldProgress := oldHours / maxHours

	attack = g.MaxAttack * (progress - oldProgress)
	defense = g.MaxDefense * (progress - oldProgress)
	hp = g.MaxHP * (progress - oldProgress)
	speed = g.MaxSpeed * (progress - oldProgress)

	gu.GrowthHours = newHours
	now := time.Now()
	gu.EquippedAt = &now
	return
}
